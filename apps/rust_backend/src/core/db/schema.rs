//! Schema initialization and management
//!
//! Handles initial schema setup and file watching for development.

use crate::core::config::Settings;
use serde::Deserialize;
use std::path::{Path, PathBuf};
use std::sync::Arc;
use surrealdb::engine::remote::ws::Client;
use surrealdb::Surreal;
use tokio::sync::Mutex;

/// Options for schema initialization
#[derive(Clone)]
pub struct SchemaInitOptions {
    pub schema_path: Option<String>,
    pub app_env: String,
    pub admin_username: String,
    pub admin_password: String,
    pub jwt_secret: String,
}

impl From<&Settings> for SchemaInitOptions {
    fn from(settings: &Settings) -> Self {
        Self {
            schema_path: settings.db.schema_path.clone(),
            app_env: settings.app.env.clone(),
            admin_username: settings.admin.username.clone(),
            admin_password: settings.admin.password.clone(),
            jwt_secret: settings.jwt.secret.clone(),
        }
    }
}

/// Initialize the database schema from db/schema.surql
/// In development mode, also starts a watcher that reapplies schema on file changes
pub async fn init_schema(db: &Surreal<Client>, opts: SchemaInitOptions) -> anyhow::Result<()> {
    let schema_path = resolve_schema_path(opts.schema_path.as_deref())?;

    apply_schema_and_seed(db, &schema_path, &opts).await?;

    // Start file watcher in development mode
    if !opts.app_env.eq_ignore_ascii_case("production") {
        start_schema_watcher(db.clone(), schema_path, opts);
    }

    Ok(())
}

/// Resolve the schema file path, searching common locations
fn resolve_schema_path(preferred: Option<&str>) -> anyhow::Result<PathBuf> {
    let mut candidates = Vec::new();

    // Add preferred path if provided
    if let Some(p) = preferred {
        if !p.is_empty() {
            candidates.push(PathBuf::from(p));
        }
    }

    // Search upwards from current directory for db/schema.surql
    if let Ok(cwd) = std::env::current_dir() {
        let mut dir = cwd.clone();
        let mut visited = std::collections::HashSet::new();

        loop {
            if visited.contains(&dir) {
                break;
            }
            visited.insert(dir.clone());

            candidates.push(dir.join("db").join("schema.surql"));

            if let Some(parent) = dir.parent() {
                if parent == dir {
                    break;
                }
                dir = parent.to_path_buf();
            } else {
                break;
            }
        }
    }

    // Find first existing file
    let mut seen = std::collections::HashSet::new();
    for candidate in candidates {
        let candidate = candidate
            .canonicalize()
            .unwrap_or_else(|_| candidate.clone());

        if seen.contains(&candidate) {
            continue;
        }
        seen.insert(candidate.clone());

        if candidate.is_file() {
            return Ok(candidate);
        }
    }

    anyhow::bail!(
        "Schema file not found; set DB_SCHEMA_PATH or place db/schema.surql alongside the project"
    )
}

/// Apply schema and seed admin user
async fn apply_schema_and_seed(
    db: &Surreal<Client>,
    schema_path: &Path,
    opts: &SchemaInitOptions,
) -> anyhow::Result<()> {
    apply_schema(db, schema_path, opts).await?;
    seed_admin_user(db, &opts.admin_username, &opts.admin_password).await?;
    Ok(())
}

/// Apply the schema file to the database
async fn apply_schema(
    db: &Surreal<Client>,
    schema_path: &Path,
    opts: &SchemaInitOptions,
) -> anyhow::Result<()> {
    let content = tokio::fs::read_to_string(schema_path).await?;

    let mut schema = content.trim().to_string();

    // Replace JWT_SECRET placeholder
    if !opts.jwt_secret.is_empty() {
        schema = schema.replace("${JWT_SECRET}", &opts.jwt_secret);
    }

    if schema.is_empty() {
        anyhow::bail!("Schema file {} is empty", schema_path.display());
    }

    match db.query(&schema).await {
        Ok(_) => {
            tracing::info!(path = %schema_path.display(), "surreal schema applied");
        },
        Err(e) => {
            let err_msg = e.to_string().to_lowercase();
            if err_msg.contains("already exists") {
                tracing::info!(
                    path = %schema_path.display(),
                    "schema already applied (this is normal)"
                );
            } else {
                tracing::error!(path = %schema_path.display(), error = %e, "failed to apply schema");
                return Err(e.into());
            }
        },
    }

    Ok(())
}

/// Simple user struct for deserialization
#[derive(Debug, Deserialize)]
struct UserRecord {
    #[serde(rename = "id")]
    _id: Option<String>,
    #[serde(rename = "email")]
    _email: Option<String>,
}

/// Seed an admin user if configured
async fn seed_admin_user(
    db: &Surreal<Client>,
    username: &str,
    password: &str,
) -> anyhow::Result<()> {
    let username_owned = username.trim().to_lowercase();
    let password_owned = password.trim().to_string();

    if username_owned.is_empty() || password_owned.is_empty() {
        tracing::debug!("Admin credentials not configured, skipping seed");
        return Ok(());
    }

    // Check if user already exists using typed query
    let check_sql = "SELECT id, email FROM user WHERE email = $email LIMIT 1";
    let mut result = db
        .query(check_sql)
        .bind(("email", username_owned.clone()))
        .await?;

    // Try to get results - if any exist, user is already created
    let existing: Vec<UserRecord> = result.take(0).unwrap_or_default();
    if !existing.is_empty() {
        tracing::debug!(username = %username_owned, "admin user already exists");
        return Ok(());
    }

    // Create admin user with argon2 hashed password
    let create_sql = r"
        CREATE user CONTENT {
            email: $email,
            pass: crypto::argon2::generate($password)
        }
    ";

    match db
        .query(create_sql)
        .bind(("email", username_owned.clone()))
        .bind(("password", password_owned))
        .await
    {
        Ok(_) => {
            tracing::info!(username = %username_owned, "admin user created for initial access");
        },
        Err(e) => {
            let err_msg = e.to_string().to_lowercase();
            // Ignore duplicate key errors (user already exists)
            if err_msg.contains("already exists") || err_msg.contains("unique") {
                tracing::debug!(username = %username_owned, "admin user already exists");
            } else {
                tracing::error!(error = %e, "failed to seed admin user");
                return Err(e.into());
            }
        },
    }

    Ok(())
}

/// Start a file watcher that reapplies schema on changes (development only)
fn start_schema_watcher(db: Surreal<Client>, schema_path: PathBuf, opts: SchemaInitOptions) {
    use notify::{Event, RecursiveMode, Watcher};
    use std::time::Duration;
    use tokio::sync::mpsc;

    let schema_path_arc = Arc::new(schema_path.clone());
    let db_arc = Arc::new(db);
    let opts_arc = Arc::new(opts);
    let debounce_timer: Arc<Mutex<Option<tokio::time::Instant>>> = Arc::new(Mutex::new(None));

    // Create async channel for file events
    let (tx, mut rx) = mpsc::unbounded_channel::<Event>();

    // Set up file watcher in blocking task (notify requires sync context)
    let watcher_schema_path = schema_path.clone();
    let watcher_schema_path_arc = schema_path_arc.clone();

    tokio::spawn(async move {
        // Initialize watcher in blocking context
        let watcher_result = tokio::task::spawn_blocking(move || {
            let tx_clone = tx;
            let mut watcher =
                notify::recommended_watcher(move |res: Result<Event, notify::Error>| {
                    if let Ok(event) = res {
                        let _ = tx_clone.send(event);
                    }
                })?;

            let dir = watcher_schema_path.parent().unwrap_or(&watcher_schema_path);
            watcher.watch(dir, RecursiveMode::NonRecursive)?;

            Ok::<_, notify::Error>(watcher)
        })
        .await;

        let _watcher = match watcher_result {
            Ok(Ok(w)) => {
                tracing::info!(path = %watcher_schema_path_arc.display(), "schema file watcher started");
                w
            },
            Ok(Err(e)) => {
                tracing::warn!(error = %e, "schema watcher disabled");
                return;
            },
            Err(e) => {
                tracing::warn!(error = %e, "schema watcher task failed");
                return;
            },
        };

        // Process events asynchronously
        while let Some(event) = rx.recv().await {
            // Only react to write/create events on our schema file
            let is_schema_change = event
                .paths
                .iter()
                .any(|p| p.canonicalize().unwrap_or_else(|_| p.clone()) == *schema_path_arc);

            if !is_schema_change {
                continue;
            }

            if !matches!(
                event.kind,
                notify::EventKind::Modify(_) | notify::EventKind::Create(_)
            ) {
                continue;
            }

            // Debounce: wait 250ms before applying
            let db_clone = Arc::clone(&db_arc);
            let path_clone = Arc::clone(&schema_path_arc);
            let opts_clone = Arc::clone(&opts_arc);
            let timer_clone = Arc::clone(&debounce_timer);

            tokio::spawn(async move {
                {
                    let mut timer = timer_clone.lock().await;
                    *timer = Some(tokio::time::Instant::now());
                }

                tokio::time::sleep(Duration::from_millis(250)).await;

                {
                    let timer = timer_clone.lock().await;
                    if let Some(last) = *timer {
                        if last.elapsed() < Duration::from_millis(250) {
                            return; // Another event came in, skip this one
                        }
                    }
                }

                match apply_schema_and_seed(&db_clone, &path_clone, &opts_clone).await {
                    Ok(_) => {
                        tracing::info!(path = %path_clone.display(), "schema reload applied");
                    },
                    Err(e) => {
                        tracing::error!(error = %e, "failed to reload schema");
                    },
                }
            });
        }
    });
}

