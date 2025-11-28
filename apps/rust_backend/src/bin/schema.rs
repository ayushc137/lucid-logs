//! Schema CLI - Database schema management tool
//!
//! Provides commands for managing the SurrealDB schema:
//! - `migrate` - Apply pending migrations
//! - `status` - Show migration status
//! - `validate` - Validate applied migrations against files
//! - `seed` - Seed initial data (admin user)
//! - `reset` - Reset database (drop and recreate)
//!
//! # Usage
//!
//! ```bash
//! # Apply pending migrations
//! cargo run --bin schema -- migrate
//!
//! # Show migration status
//! cargo run --bin schema -- status
//!
//! # Validate migrations
//! cargo run --bin schema -- validate
//!
//! # Seed initial data
//! cargo run --bin schema -- seed
//!
//! # Reset database (DESTRUCTIVE!)
//! cargo run --bin schema -- reset --force
//! ```
//!
//! # Migration Strategy
//!
//! ## Development
//! - Edit `db/schema.surql` for quick iteration
//! - Schema is auto-applied on server startup with hot-reload
//!
//! ## Production
//! - Create versioned migrations in `db/migrations/`
//! - Run `schema migrate` before deploying
//! - Never modify already-applied migrations!

use anyhow::Result;
use clap::{Parser, Subcommand};
use rust_backend::core::{
    init_schema, resolve_migrations_dir, MigrationRunner, SchemaInitOptions, Settings,
};
use surrealdb::engine::remote::ws::{Client, Ws};
use surrealdb::Surreal;
use tracing_subscriber::{layer::SubscriberExt, util::SubscriberInitExt};

#[derive(Parser)]
#[command(name = "schema")]
#[command(author, version, about = "Database schema management CLI", long_about = None)]
struct Cli {
    #[command(subcommand)]
    command: Commands,

    /// Verbose output
    #[arg(short, long, global = true)]
    verbose: bool,
}

#[derive(Subcommand)]
enum Commands {
    /// Apply pending migrations from db/migrations/
    Migrate {
        /// Dry run - show what would be applied without applying
        #[arg(long)]
        dry_run: bool,
    },

    /// Show migration status (applied vs pending)
    Status,

    /// Validate applied migrations against their files (checksum verification)
    Validate,

    /// Apply the consolidated schema.surql (development mode)
    Apply,

    /// Seed initial data (admin user)
    Seed,

    /// Reset the database (DESTRUCTIVE - drops all data)
    Reset {
        /// Force reset without confirmation
        #[arg(long)]
        force: bool,
    },

    /// Create a new migration file
    New {
        /// Name of the migration (e.g., "add_field")
        name: String,
    },
}

#[tokio::main]
async fn main() -> Result<()> {
    let cli = Cli::parse();

    // Initialize logging
    let log_level = if cli.verbose { "debug" } else { "info" };
    tracing_subscriber::registry()
        .with(tracing_subscriber::EnvFilter::new(
            std::env::var("RUST_LOG")
                .unwrap_or_else(|_| format!("schema={},rust_backend=info", log_level)),
        ))
        .with(tracing_subscriber::fmt::layer().with_target(false))
        .init();

    // Load configuration
    let _ = dotenvy::dotenv();
    let settings = Settings::new()?;

    // Handle "new" command before connecting to DB
    if let Commands::New { name } = cli.command {
        return create_new_migration(&name, &settings).await;
    }

    tracing::info!(
        "Connecting to SurrealDB at {}:{}",
        settings.db.host,
        settings.db.port
    );

    // Connect to database
    let db: Surreal<Client> = Surreal::new::<Ws>(&settings.db.ws_url()).await?;

    // Sign in as root
    db.signin(surrealdb::opt::auth::Root {
        username: &settings.db.user,
        password: &settings.db.pass,
    })
    .await?;

    // Select namespace and database
    db.use_ns(&settings.db.namespace)
        .use_db(&settings.db.database)
        .await?;

    tracing::info!(
        "Connected to {}/{}",
        settings.db.namespace,
        settings.db.database
    );

    match cli.command {
        Commands::Migrate { dry_run } => {
            run_migrate(&db, &settings, dry_run).await?;
        },

        Commands::Status => {
            run_status(&db, &settings).await?;
        },

        Commands::Validate => {
            run_validate(&db, &settings).await?;
        },

        Commands::Apply => {
            tracing::info!("Applying consolidated schema.surql...");
            let opts = SchemaInitOptions::from(&settings);
            init_schema(&db, opts).await?;
            tracing::info!("Schema applied successfully");
        },

        Commands::Seed => {
            tracing::info!("Seeding database...");
            let opts = SchemaInitOptions::from(&settings);
            init_schema(&db, opts).await?;
            tracing::info!("Database seeded successfully");
        },

        Commands::Reset { force } => {
            run_reset(&db, &settings, force).await?;
        },

        Commands::New { .. } => unreachable!(), // Handled above
    }

    Ok(())
}

async fn run_migrate(db: &Surreal<Client>, settings: &Settings, dry_run: bool) -> Result<()> {
    let migrations_dir = resolve_migrations_dir(settings.db.migrations_path.as_deref());
    let runner = MigrationRunner::new(db.clone(), &migrations_dir, settings.jwt.secret.clone());

    let pending = runner.get_pending().await?;

    if pending.is_empty() {
        println!("\n✅ No pending migrations\n");
        return Ok(());
    }

    println!("\n📦 Pending migrations:");
    for migration in &pending {
        println!("  {:03}_{}", migration.version, migration.name);
    }
    println!();

    if dry_run {
        println!("🔍 Dry run - no migrations applied\n");
        return Ok(());
    }

    tracing::info!("Applying {} pending migration(s)...", pending.len());
    let applied = runner.run_pending().await?;

    println!("\n✅ Applied {} migration(s):", applied.len());
    for migration in &applied {
        println!("  {:03}_{}", migration.version, migration.name);
    }
    println!();

    Ok(())
}

async fn run_status(db: &Surreal<Client>, settings: &Settings) -> Result<()> {
    let migrations_dir = resolve_migrations_dir(settings.db.migrations_path.as_deref());
    let runner = MigrationRunner::new(db.clone(), &migrations_dir, settings.jwt.secret.clone());

    let applied = runner.get_applied().await?;
    let pending = runner.get_pending().await?;

    println!("\n=== Migration Status ===\n");

    if applied.is_empty() {
        println!("📭 No migrations applied yet\n");
    } else {
        println!("✅ Applied migrations:");
        for m in &applied {
            println!(
                "  {:03}_{} (applied: {:?})",
                m.version,
                m.name,
                m.applied_at.as_deref().unwrap_or("unknown")
            );
        }
        println!();
    }

    if pending.is_empty() {
        println!("📭 No pending migrations\n");
    } else {
        println!("⏳ Pending migrations:");
        for m in &pending {
            println!("  {:03}_{}", m.version, m.name);
        }
        println!();
    }

    // Also show table counts
    println!("=== Data Summary ===\n");

    let mut result = db.query("SELECT count() FROM user GROUP ALL").await?;
    let count: Option<serde_json::Value> = result.take(0)?;
    if let Some(count) = count {
        println!(
            "  Users: {}",
            count.get("count").unwrap_or(&serde_json::json!(0))
        );
    } else {
        println!("  Users: 0");
    }

    let mut result = db
        .query("SELECT count() FROM categories WHERE deleted_at = NONE GROUP ALL")
        .await?;
    let count: Option<serde_json::Value> = result.take(0)?;
    if let Some(count) = count {
        println!(
            "  Categories: {}",
            count.get("count").unwrap_or(&serde_json::json!(0))
        );
    } else {
        println!("  Categories: 0");
    }

    let mut result = db
        .query("SELECT count() FROM tasks WHERE deleted_at = NONE GROUP ALL")
        .await?;
    let count: Option<serde_json::Value> = result.take(0)?;
    if let Some(count) = count {
        println!(
            "  Tasks: {}",
            count.get("count").unwrap_or(&serde_json::json!(0))
        );
    } else {
        println!("  Tasks: 0");
    }

    println!();

    Ok(())
}

async fn run_validate(db: &Surreal<Client>, settings: &Settings) -> Result<()> {
    let migrations_dir = resolve_migrations_dir(settings.db.migrations_path.as_deref());
    let runner = MigrationRunner::new(db.clone(), &migrations_dir, settings.jwt.secret.clone());

    let applied = runner.get_applied().await?;
    let files = runner.read_migration_files().await?;

    println!("\n=== Migration Validation ===\n");

    if applied.is_empty() {
        println!("📭 No migrations to validate\n");
        return Ok(());
    }

    let mut has_issues = false;

    for applied_migration in &applied {
        let maybe_file = files
            .iter()
            .find(|file| file.version == applied_migration.version);

        match maybe_file {
            None => {
                has_issues = true;
                println!(
                    "⚠️ {:03}_{}: Migration file not found on disk",
                    applied_migration.version, applied_migration.name
                );
            },
            Some(file) => {
                if applied_migration.checksum.as_deref() == Some(&file.checksum) {
                    println!(
                        "✅ {:03}_{}: Checksum matches",
                        applied_migration.version, applied_migration.name
                    );
                } else {
                    has_issues = true;
                    println!(
                        "❌ {:03}_{}: Checksum mismatch! Applied: {:?}, File: {}",
                        applied_migration.version,
                        applied_migration.name,
                        applied_migration.checksum,
                        file.checksum
                    );
                }
            },
        }
    }

    println!();

    if has_issues {
        println!("⚠️  Some migrations have validation issues!");
        println!("   - ChecksumMismatch: Migration file was modified after being applied");
        println!("   - MissingFile: Migration was applied but file no longer exists");
        println!();
        std::process::exit(1);
    } else {
        println!("✅ All migrations validated successfully\n");
    }

    Ok(())
}

async fn run_reset(db: &Surreal<Client>, settings: &Settings, force: bool) -> Result<()> {
    if !force {
        tracing::error!("Reset requires --force flag. This will DELETE ALL DATA!");
        println!("\n⚠️  This will DELETE ALL DATA in the database!");
        println!("   Run with --force to confirm.\n");
        std::process::exit(1);
    }

    tracing::warn!("Resetting database - this will DELETE ALL DATA!");

    // Remove all data from the database
    let remove_query = format!("REMOVE DATABASE IF EXISTS {}", settings.db.database);
    db.query(&remove_query).await?;

    tracing::info!("Database cleared");

    // Recreate namespace and database
    db.use_ns(&settings.db.namespace)
        .use_db(&settings.db.database)
        .await?;

    // Apply migrations
    let migrations_dir = resolve_migrations_dir(settings.db.migrations_path.as_deref());
    let runner = MigrationRunner::new(db.clone(), &migrations_dir, settings.jwt.secret.clone());
    runner.run_pending().await?;

    // Seed admin user
    let opts = SchemaInitOptions::from(settings);
    init_schema(db, opts).await?;

    println!("\n✅ Database reset complete\n");
    tracing::info!("Database reset complete");

    Ok(())
}

async fn create_new_migration(name: &str, settings: &Settings) -> Result<()> {
    let migrations_dir = resolve_migrations_dir(settings.db.migrations_path.as_deref());

    // Find the next version number
    let mut max_version: i64 = -1;

    if migrations_dir.exists() {
        let mut entries = tokio::fs::read_dir(&migrations_dir).await?;
        while let Some(entry) = entries.next_entry().await? {
            let path = entry.path();
            if path.extension().is_some_and(|ext| ext == "surql") {
                if let Some(filename) = path.file_stem().and_then(|s| s.to_str()) {
                    if let Some(version_str) = filename.split('_').next() {
                        if let Ok(version) = version_str.parse::<i64>() {
                            max_version = max_version.max(version);
                        }
                    }
                }
            }
        }
    } else {
        tokio::fs::create_dir_all(&migrations_dir).await?;
    }

    let next_version = max_version + 1;
    let filename = format!("{:03}_{}.surql", next_version, name);
    let filepath = migrations_dir.join(&filename);

    // Create migration file template
    let template = format!(
        r#"-- Migration: {:03}_{}
-- Description: TODO: Add description
-- Created: {}

-- TODO: Add your migration SQL here
-- 
-- Examples:
-- 
-- Add a new table:
-- DEFINE TABLE new_table SCHEMAFULL;
-- DEFINE FIELD name ON new_table TYPE string;
-- 
-- Add a new field to existing table:
-- DEFINE FIELD new_field ON existing_table TYPE option<string>;
-- 
-- Add an index:
-- DEFINE INDEX idx_name ON TABLE table_name COLUMNS field_name;
-- 
-- Note: SurrealDB schema changes are mostly idempotent with "IF NOT EXISTS"
"#,
        next_version,
        name,
        chrono::Utc::now().format("%Y-%m-%d")
    );

    tokio::fs::write(&filepath, template).await?;

    println!("\n✅ Created migration: {}\n", filepath.display());
    println!("   Edit the file and add your migration SQL.");
    println!("   Then run: task rust:migrate\n");

    Ok(())
}
