use serde::Deserialize;
use std::env;

#[derive(Debug, Deserialize, Clone)]
pub struct Settings {
    pub app: AppSettings,
    pub server: ServerSettings,
    pub db: DatabaseSettings,
    pub jwt: JwtSettings,
    pub admin: AdminSettings,
    pub cors: CorsSettings,
}

#[derive(Debug, Deserialize, Clone)]
pub struct AppSettings {
    pub env: String,
}

#[derive(Debug, Deserialize, Clone)]
pub struct ServerSettings {
    pub port: u16,
}

#[derive(Debug, Deserialize, Clone)]
pub struct DatabaseSettings {
    pub host: String,
    pub port: u16,
    pub user: String,
    pub pass: String,
    pub namespace: String,
    pub database: String,
    pub schema_path: Option<String>,
}

impl DatabaseSettings {
    /// Returns the WebSocket URL for SurrealDB connection
    pub fn ws_url(&self) -> String {
        format!("{}:{}", self.host, self.port)
    }
}

#[derive(Debug, Deserialize, Clone)]
pub struct JwtSettings {
    pub secret: String,
}

#[derive(Debug, Deserialize, Clone)]
pub struct AdminSettings {
    pub username: String,
    pub password: String,
}

#[derive(Debug, Deserialize, Clone)]
pub struct CorsSettings {
    /// Allowed origins (comma-separated). Empty or "*" allows any origin (dev only).
    pub allowed_origins: Vec<String>,
}

impl Settings {
    pub fn new() -> Result<Self, ConfigError> {
        let app_env = env::var("APP_ENV").unwrap_or_else(|_| "development".into());
        let is_prod = matches!(app_env.as_str(), "production" | "prod");

        // Server settings - default to 8080 (different from DB port 8000)
        let http_port = env::var("HTTP_PORT")
            .or_else(|_| env::var("HOST_APP_PORT"))
            .unwrap_or_else(|_| "8080".into())
            .parse()
            .unwrap_or(8080);

        // Database settings (matching Go's env var names)
        let db_host = env::var("DB_HOST").unwrap_or_else(|_| "localhost".into());
        let db_port = env::var("DB_PORT")
            .unwrap_or_else(|_| "8000".into())
            .parse()
            .unwrap_or(8000);
        let db_user = env::var("DB_USER").unwrap_or_else(|_| "root".into());
        let db_pass = env::var("DB_PASS").unwrap_or_else(|_| "root".into());
        let db_namespace = env::var("DB_NAMESPACE").unwrap_or_else(|_| "daily_journal".into());
        let db_database = env::var("DB_DATABASE").unwrap_or_else(|_| "core".into());
        let db_schema_path = env::var("DB_SCHEMA_PATH").ok();

        // JWT settings - REQUIRE in production, generate with CSPRNG in dev
        let jwt_secret = match env::var("JWT_SECRET") {
            Ok(secret) if !secret.is_empty() => secret,
            _ => {
                if is_prod {
                    return Err(ConfigError::MissingRequired(
                        "JWT_SECRET is required in production. Generate with: openssl rand -base64 32".into(),
                    ));
                }
                let secret = generate_secret_csprng(32);
                tracing::warn!(
                    "[SECURITY] JWT_SECRET not set; generated ephemeral secret (tokens won't persist across restarts)"
                );
                secret
            }
        };

        // Admin settings - warn in production if using defaults
        let admin_username = env::var("ADMIN_USERNAME").ok();
        let admin_password = env::var("ADMIN_PASSWORD").ok();

        let (admin_username, admin_password) = match (admin_username, admin_password) {
            (Some(u), Some(p)) if !u.is_empty() && !p.is_empty() => (u, p),
            _ => {
                if is_prod {
                    tracing::warn!(
                        "[SECURITY] ADMIN_USERNAME/ADMIN_PASSWORD not set in production; admin seeding disabled"
                    );
                    (String::new(), String::new())
                } else {
                    tracing::info!(
                        "Using default admin credentials (admin@example.com). Set ADMIN_USERNAME/ADMIN_PASSWORD to override."
                    );
                    ("admin@example.com".into(), "adminadmin".into())
                }
            }
        };

        // CORS settings
        let cors_origins = env::var("CORS_ALLOWED_ORIGINS")
            .unwrap_or_else(|_| if is_prod { String::new() } else { "*".into() });
        let allowed_origins: Vec<String> = if cors_origins.is_empty() || cors_origins == "*" {
            vec![]
        } else {
            cors_origins
                .split(',')
                .map(|s| s.trim().to_string())
                .filter(|s| !s.is_empty())
                .collect()
        };

        if is_prod && allowed_origins.is_empty() {
            tracing::warn!(
                "[SECURITY] CORS_ALLOWED_ORIGINS not set in production; all origins blocked. Set to comma-separated list of allowed origins."
            );
        }

        Ok(Settings {
            app: AppSettings { env: app_env },
            server: ServerSettings { port: http_port },
            db: DatabaseSettings {
                host: db_host,
                port: db_port,
                user: db_user,
                pass: db_pass,
                namespace: db_namespace,
                database: db_database,
                schema_path: db_schema_path,
            },
            jwt: JwtSettings { secret: jwt_secret },
            admin: AdminSettings {
                username: admin_username,
                password: admin_password,
            },
            cors: CorsSettings { allowed_origins },
        })
    }

    pub fn is_development(&self) -> bool {
        matches!(self.app.env.as_str(), "development" | "dev")
    }

}

/// Generate a cryptographically secure secret using OS RNG
fn generate_secret_csprng(len: usize) -> String {
    use base64ct::{Base64, Encoding};
    use rand::RngCore;

    let mut bytes = vec![0u8; len];
    rand::rng().fill_bytes(&mut bytes);
    Base64::encode_string(&bytes)
}

#[derive(Debug, thiserror::Error)]
pub enum ConfigError {
    #[error("Environment variable error: {0}")]
    EnvVar(#[from] std::env::VarError),

    #[error("Missing required configuration: {0}")]
    MissingRequired(String),
}
