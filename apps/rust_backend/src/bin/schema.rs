//! Schema CLI - Database schema management tool
//!
//! Provides commands for managing the SurrealDB schema:
//! - `migrate` - Apply schema migrations
//! - `seed` - Seed initial data
//! - `reset` - Reset database (drop and recreate)
//! - `status` - Show current schema status
//!
//! # Usage
//!
//! ```bash
//! # Apply migrations
//! cargo run --bin schema -- migrate
//!
//! # Seed data
//! cargo run --bin schema -- seed
//!
//! # Reset database (DESTRUCTIVE!)
//! cargo run --bin schema -- reset --force
//!
//! # Check status
//! cargo run --bin schema -- status
//! ```

use anyhow::Result;
use clap::{Parser, Subcommand};
use rust_backend::config::Settings;
use rust_backend::repositories::{init_schema, SchemaInitOptions};
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
    /// Apply schema migrations to the database
    Migrate,

    /// Seed initial data (admin user, sample data)
    Seed,

    /// Reset the database (DESTRUCTIVE - drops all data)
    Reset {
        /// Force reset without confirmation
        #[arg(long)]
        force: bool,
    },

    /// Show current schema/migration status
    Status,
}

#[tokio::main]
async fn main() -> Result<()> {
    let cli = Cli::parse();

    // Initialize logging
    let log_level = if cli.verbose { "debug" } else { "info" };
    tracing_subscriber::registry()
        .with(tracing_subscriber::EnvFilter::new(
            std::env::var("RUST_LOG").unwrap_or_else(|_| format!("schema={},rust_backend=info", log_level)),
        ))
        .with(tracing_subscriber::fmt::layer().with_target(false))
        .init();

    // Load configuration
    let _ = dotenvy::dotenv();
    let settings = Settings::new()?;

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
        Commands::Migrate => {
            tracing::info!("Applying schema migrations...");
            let opts = SchemaInitOptions::from(&settings);
            init_schema(&db, opts).await?;
            tracing::info!("Schema migrations applied successfully");
        }

        Commands::Seed => {
            tracing::info!("Seeding database...");
            let opts = SchemaInitOptions::from(&settings);
            init_schema(&db, opts).await?;
            tracing::info!("Database seeded successfully");
        }

        Commands::Reset { force } => {
            if !force {
                tracing::error!("Reset requires --force flag. This will DELETE ALL DATA!");
                std::process::exit(1);
            }

            tracing::warn!("Resetting database - this will DELETE ALL DATA!");

            // Remove all data from the database using raw query
            let remove_query = format!("REMOVE DATABASE IF EXISTS {}", settings.db.database);
            db.query(&remove_query).await?;

            tracing::info!("Database cleared");

            // Recreate namespace and database
            db.use_ns(&settings.db.namespace)
                .use_db(&settings.db.database)
                .await?;

            // Reapply schema
            let opts = SchemaInitOptions::from(&settings);
            init_schema(&db, opts).await?;

            tracing::info!("Database reset complete");
        }

        Commands::Status => {
            tracing::info!("Checking schema status...");

            // Query for tables
            let mut result = db.query("INFO FOR DB").await?;
            let info: Option<serde_json::Value> = result.take(0)?;

            if let Some(info) = info {
                println!("\n=== Database Info ===");
                println!("{}", serde_json::to_string_pretty(&info)?);
            }

            // Check user count
            let mut result = db.query("SELECT count() FROM user GROUP ALL").await?;
            let count: Option<serde_json::Value> = result.take(0)?;
            if let Some(count) = count {
                println!("\nUser count: {}", count);
            }

            // Check task count
            let mut result = db.query("SELECT count() FROM tasks GROUP ALL").await?;
            let count: Option<serde_json::Value> = result.take(0)?;
            if let Some(count) = count {
                println!("Task count: {}", count);
            }
        }
    }

    Ok(())
}

