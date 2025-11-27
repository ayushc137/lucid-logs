//! Database migration runner
//!
//! Handles versioned migrations for SurrealDB with tracking and rollback support.
//!
//! # Migration File Format
//!
//! Migration files should be named: `NNN_description.surql`
//! - NNN: Zero-padded version number (000, 001, 002, ...)
//! - description: Brief description with underscores
//!
//! Example: `003_add_categories.surql`
//!
//! # Usage
//!
//! ```ignore
//! use rust_backend::repositories::migrations::MigrationRunner;
//!
//! let runner = MigrationRunner::new(db.clone(), "db/migrations");
//! runner.run_pending().await?;
//! ```

use serde::{Deserialize, Serialize};
use sha2::{Digest, Sha256};
use std::path::{Path, PathBuf};
use surrealdb::engine::remote::ws::Client;
use surrealdb::Surreal;

use crate::error::AppError;

/// Migration record stored in the database
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MigrationRecord {
    pub version: i64,
    pub name: String,
    pub applied_at: Option<String>,
    pub checksum: Option<String>,
}

/// A pending migration to be applied
#[derive(Debug, Clone)]
pub struct Migration {
    pub version: i64,
    pub name: String,
    pub sql: String,
    pub checksum: String,
}

/// Migration runner for versioned database migrations
#[derive(Clone)]
pub struct MigrationRunner {
    db: Surreal<Client>,
    migrations_dir: PathBuf,
    jwt_secret: String,
}

impl MigrationRunner {
    /// Create a new migration runner
    pub fn new(db: Surreal<Client>, migrations_dir: impl AsRef<Path>, jwt_secret: String) -> Self {
        Self {
            db,
            migrations_dir: migrations_dir.as_ref().to_path_buf(),
            jwt_secret,
        }
    }

    /// Get list of applied migrations from the database
    pub async fn get_applied(&self) -> Result<Vec<MigrationRecord>, AppError> {
        // First ensure the _migrations table exists
        let init_sql = r#"
            DEFINE TABLE IF NOT EXISTS _migrations SCHEMAFULL;
            DEFINE FIELD IF NOT EXISTS version ON _migrations TYPE int;
            DEFINE FIELD IF NOT EXISTS name ON _migrations TYPE string;
            DEFINE FIELD IF NOT EXISTS applied_at ON _migrations TYPE datetime VALUE time::now();
            DEFINE FIELD IF NOT EXISTS checksum ON _migrations TYPE option<string>;
            DEFINE INDEX IF NOT EXISTS idx_migrations_version ON TABLE _migrations COLUMNS version UNIQUE;
        "#;

        self.db.query(init_sql).await?;

        let sql = "SELECT version, name, applied_at, checksum FROM _migrations ORDER BY version ASC";
        let mut result = self.db.query(sql).await?;
        let migrations: Vec<MigrationRecord> = result.take(0)?;
        Ok(migrations)
    }

    /// Get list of pending migrations from the filesystem
    pub async fn get_pending(&self) -> Result<Vec<Migration>, AppError> {
        let applied = self.get_applied().await?;
        let applied_versions: std::collections::HashSet<i64> =
            applied.iter().map(|m| m.version).collect();

        let mut pending = Vec::new();

        // Read migration files from directory
        let migrations = self.read_migration_files().await?;

        for migration in migrations {
            if !applied_versions.contains(&migration.version) {
                pending.push(migration);
            }
        }

        // Sort by version
        pending.sort_by_key(|m| m.version);
        Ok(pending)
    }

    /// Read all migration files from the migrations directory
    async fn read_migration_files(&self) -> Result<Vec<Migration>, AppError> {
        let mut migrations = Vec::new();

        // Check if directory exists
        if !self.migrations_dir.exists() {
            tracing::warn!(
                path = %self.migrations_dir.display(),
                "migrations directory does not exist"
            );
            return Ok(migrations);
        }

        let mut entries = tokio::fs::read_dir(&self.migrations_dir).await.map_err(|e| {
            tracing::error!(error = %e, path = %self.migrations_dir.display(), "failed to read migrations directory");
            AppError::Internal
        })?;

        while let Some(entry) = entries.next_entry().await.map_err(|e| {
            tracing::error!(error = %e, "failed to read directory entry");
            AppError::Internal
        })? {
            let path = entry.path();

            // Skip non-.surql files
            if path.extension().is_none_or(|ext| ext != "surql") {
                continue;
            }

            // Parse filename: NNN_description.surql
            let filename = path
                .file_stem()
                .and_then(|s| s.to_str())
                .ok_or(AppError::Internal)?;

            let parts: Vec<&str> = filename.splitn(2, '_').collect();
            if parts.len() != 2 {
                tracing::warn!(
                    filename = filename,
                    "skipping migration file with invalid name format"
                );
                continue;
            }

            let version: i64 = parts[0].parse().map_err(|_| {
                tracing::warn!(
                    filename = filename,
                    "skipping migration file with invalid version number"
                );
                AppError::Internal
            })?;

            let name = parts[1].to_string();

            // Read file content
            let sql = tokio::fs::read_to_string(&path).await.map_err(|e| {
                tracing::error!(error = %e, path = %path.display(), "failed to read migration file");
                AppError::Internal
            })?;

            // Calculate checksum
            let checksum = Self::calculate_checksum(&sql);

            migrations.push(Migration {
                version,
                name,
                sql,
                checksum,
            });
        }

        // Sort by version
        migrations.sort_by_key(|m| m.version);
        Ok(migrations)
    }

    /// Calculate SHA256 checksum of migration content
    fn calculate_checksum(content: &str) -> String {
        let mut hasher = Sha256::new();
        hasher.update(content.as_bytes());
        format!("{:x}", hasher.finalize())
    }

    /// Run all pending migrations
    pub async fn run_pending(&self) -> Result<Vec<MigrationRecord>, AppError> {
        let pending = self.get_pending().await?;

        if pending.is_empty() {
            tracing::info!("no pending migrations");
            return Ok(Vec::new());
        }

        tracing::info!(count = pending.len(), "running pending migrations");

        let mut applied = Vec::new();

        for migration in pending {
            match self.apply_migration(&migration).await {
                Ok(record) => {
                    tracing::info!(
                        version = migration.version,
                        name = %migration.name,
                        "migration applied successfully"
                    );
                    applied.push(record);
                }
                Err(e) => {
                    tracing::error!(
                        version = migration.version,
                        name = %migration.name,
                        error = %e,
                        "migration failed"
                    );
                    return Err(e);
                }
            }
        }

        Ok(applied)
    }

    /// Apply a single migration
    async fn apply_migration(&self, migration: &Migration) -> Result<MigrationRecord, AppError> {
        tracing::debug!(
            version = migration.version,
            name = %migration.name,
            "applying migration"
        );

        // Replace JWT_SECRET placeholder
        let sql = if !self.jwt_secret.is_empty() {
            migration.sql.replace("${JWT_SECRET}", &self.jwt_secret)
        } else {
            migration.sql.clone()
        };

        // Execute the migration SQL
        match self.db.query(&sql).await {
            Ok(_) => {}
            Err(e) => {
                let err_msg = e.to_string().to_lowercase();
                // Ignore "already exists" errors for idempotent migrations
                if !err_msg.contains("already exists") {
                    return Err(AppError::Database(e));
                }
                tracing::debug!(
                    version = migration.version,
                    "migration contains already-existing definitions (idempotent)"
                );
            }
        }

        // Record the migration
        let record_sql = r#"
            CREATE _migrations CONTENT {
                version: $version,
                name: $name,
                checksum: $checksum
            }
        "#;

        self.db
            .query(record_sql)
            .bind(("version", migration.version))
            .bind(("name", migration.name.clone()))
            .bind(("checksum", migration.checksum.clone()))
            .await?;

        Ok(MigrationRecord {
            version: migration.version,
            name: migration.name.clone(),
            applied_at: None,
            checksum: Some(migration.checksum.clone()),
        })
    }

    /// Get migration status (applied vs pending)
    pub async fn status(&self) -> Result<MigrationStatus, AppError> {
        let applied = self.get_applied().await?;
        let pending = self.get_pending().await?;

        Ok(MigrationStatus {
            applied,
            pending: pending
                .into_iter()
                .map(|m| MigrationRecord {
                    version: m.version,
                    name: m.name,
                    applied_at: None,
                    checksum: Some(m.checksum),
                })
                .collect(),
        })
    }

    /// Validate that applied migrations match their checksums
    pub async fn validate(&self) -> Result<Vec<MigrationValidation>, AppError> {
        let applied = self.get_applied().await?;
        let files = self.read_migration_files().await?;

        let mut validations = Vec::new();

        for applied_migration in &applied {
            let file_migration = files.iter().find(|f| f.version == applied_migration.version);

            let validation = match file_migration {
                None => MigrationValidation {
                    version: applied_migration.version,
                    name: applied_migration.name.clone(),
                    status: ValidationStatus::MissingFile,
                    message: "Migration file not found on disk".to_string(),
                },
                Some(file) => {
                    if applied_migration.checksum.as_ref() == Some(&file.checksum) {
                        MigrationValidation {
                            version: applied_migration.version,
                            name: applied_migration.name.clone(),
                            status: ValidationStatus::Valid,
                            message: "Checksum matches".to_string(),
                        }
                    } else {
                        MigrationValidation {
                            version: applied_migration.version,
                            name: applied_migration.name.clone(),
                            status: ValidationStatus::ChecksumMismatch,
                            message: format!(
                                "Checksum mismatch! Applied: {:?}, File: {}",
                                applied_migration.checksum, file.checksum
                            ),
                        }
                    }
                }
            };

            validations.push(validation);
        }

        Ok(validations)
    }
}

/// Migration status report
#[derive(Debug)]
pub struct MigrationStatus {
    pub applied: Vec<MigrationRecord>,
    pub pending: Vec<MigrationRecord>,
}

/// Migration validation result
#[derive(Debug)]
pub struct MigrationValidation {
    pub version: i64,
    pub name: String,
    pub status: ValidationStatus,
    pub message: String,
}

/// Validation status for a migration
#[derive(Debug, PartialEq, Eq)]
pub enum ValidationStatus {
    Valid,
    ChecksumMismatch,
    MissingFile,
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_calculate_checksum() {
        let content = "CREATE TABLE test;";
        let checksum = MigrationRunner::calculate_checksum(content);
        assert!(!checksum.is_empty());
        assert_eq!(checksum.len(), 64); // SHA256 produces 64 hex chars
    }

    #[test]
    fn test_checksum_consistency() {
        let content = "SELECT * FROM users;";
        let checksum1 = MigrationRunner::calculate_checksum(content);
        let checksum2 = MigrationRunner::calculate_checksum(content);
        assert_eq!(checksum1, checksum2);
    }
}

