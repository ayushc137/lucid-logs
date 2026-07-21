// Package config handles application configuration.
//
// Configuration is loaded from:
//  1. Environment variables (highest priority)
//  2. config.yaml file (if present)
//  3. Default values
//
// Environment variables use the format: SECTION_KEY
// Example: DB_HOST, JWT_SECRET, HTTP_PORT
package config

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

// =============================================================================
// CONFIGURATION STRUCTURES
// =============================================================================

// Config holds all application configuration.
type Config struct {
	App      AppConfig
	Server   ServerConfig
	Database DatabaseConfig
	JWT      JWTConfig
	Admin    AdminConfig
	CORS     CORSConfig
}

// AppConfig contains application-level settings.
type AppConfig struct {
	Env     string // development, staging, production
	Name    string // Application name
	Version string // Application version
}

// ServerConfig contains HTTP server settings.
type ServerConfig struct {
	Port            int           // HTTP port
	ReadTimeout     time.Duration // Request read timeout
	WriteTimeout    time.Duration // Response write timeout
	ShutdownTimeout time.Duration // Graceful shutdown timeout
}

// DatabaseConfig contains local and optional Turso sync settings.
type DatabaseConfig struct {
	Path           string // Local database or replica path
	URL            string // Optional Turso remote URL
	AuthToken      string // Optional Turso authentication token
	MigrationsPath string // Path to SQL migrations directory
}

// IsSynced reports whether local-to-Turso synchronization is configured.
func (d DatabaseConfig) IsSynced() bool {
	return d.URL != "" && d.AuthToken != ""
}

// JWTConfig contains JWT authentication settings.
type JWTConfig struct {
	Secret          string // Signing secret
	ExpirationHours int    // Token expiration in hours
	Issuer          string // Token issuer
}

// AdminConfig contains admin user seeding settings.
type AdminConfig struct {
	Username string // Admin email
	Password string // Admin password
}

// CORSConfig contains CORS settings.
type CORSConfig struct {
	AllowedOrigins []string // Allowed origins (empty = all in dev)
	AllowedMethods []string // Allowed HTTP methods
	AllowedHeaders []string // Allowed headers
}

// =============================================================================
// LOADING
// =============================================================================

// Load reads configuration from environment variables and config files.
//
// Priority (highest to lowest):
//  1. Environment variables
//  2. config.yaml file
//  3. Default values
func Load() (*Config, error) {
	v := viper.New()

	// Set config file settings
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("./config")
	v.AddConfigPath("/etc/go-backend")

	// Read config file (optional)
	if err := v.ReadInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if !errors.As(err, &configFileNotFoundError) {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	// Enable environment variables
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Set defaults
	setDefaults(v)

	// Build config struct
	cfg := &Config{}

	// App settings
	cfg.App.Env = v.GetString("APP_ENV")
	cfg.App.Name = v.GetString("APP_NAME")
	cfg.App.Version = v.GetString("APP_VERSION")

	// Server settings
	cfg.Server.Port = v.GetInt("HTTP_PORT")
	cfg.Server.ReadTimeout = v.GetDuration("SERVER_READ_TIMEOUT")
	cfg.Server.WriteTimeout = v.GetDuration("SERVER_WRITE_TIMEOUT")
	cfg.Server.ShutdownTimeout = v.GetDuration("SERVER_SHUTDOWN_TIMEOUT")

	// Database settings
	cfg.Database.Path = v.GetString("DATABASE_PATH")
	cfg.Database.URL = v.GetString("TURSO_DATABASE_URL")
	cfg.Database.AuthToken = v.GetString("TURSO_AUTH_TOKEN")
	cfg.Database.MigrationsPath = v.GetString("DATABASE_MIGRATIONS_PATH")

	// JWT settings
	cfg.JWT.Secret = v.GetString("JWT_SECRET")
	cfg.JWT.ExpirationHours = v.GetInt("JWT_EXPIRATION_HOURS")
	cfg.JWT.Issuer = v.GetString("JWT_ISSUER")

	// Admin settings
	cfg.Admin.Username = v.GetString("ADMIN_USERNAME")
	cfg.Admin.Password = v.GetString("ADMIN_PASSWORD")

	// CORS settings
	corsOrigins := v.GetString("CORS_ALLOWED_ORIGINS")
	if corsOrigins != "" && corsOrigins != "*" {
		cfg.CORS.AllowedOrigins = strings.Split(corsOrigins, ",")
	}
	cfg.CORS.AllowedMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}
	cfg.CORS.AllowedHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization", "Accept", "X-Requested-With"}

	// Validate and apply security defaults
	if err := validateAndSecure(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// setDefaults sets default configuration values.
func setDefaults(v *viper.Viper) {
	// App defaults
	v.SetDefault("APP_ENV", "development")
	v.SetDefault("APP_NAME", "Lucid Logs")
	v.SetDefault("APP_VERSION", "1.0.0")

	// Server defaults
	v.SetDefault("HTTP_PORT", 8080)
	v.SetDefault("SERVER_READ_TIMEOUT", "15s")
	v.SetDefault("SERVER_WRITE_TIMEOUT", "15s")
	v.SetDefault("SERVER_SHUTDOWN_TIMEOUT", "10s")

	// Database defaults
	v.SetDefault("DATABASE_PATH", "./data/lucid-logs.db")
	v.SetDefault("DATABASE_MIGRATIONS_PATH", "../../db/migrations")

	// JWT defaults
	v.SetDefault("JWT_EXPIRATION_HOURS", 24)
	v.SetDefault("JWT_ISSUER", "lucid-logs")

	// Admin defaults (only for development)
	v.SetDefault("ADMIN_USERNAME", "admin@example.com")
	v.SetDefault("ADMIN_PASSWORD", "adminadmin")
}

// validateAndSecure validates security-critical settings.
func validateAndSecure(cfg *Config) error {
	isProd := cfg.IsProd()

	if cfg.Database.Path == "" {
		return fmt.Errorf("DATABASE_PATH is required")
	}
	if (cfg.Database.URL == "") != (cfg.Database.AuthToken == "") {
		return fmt.Errorf("TURSO_DATABASE_URL and TURSO_AUTH_TOKEN must be set together")
	}

	// Handle JWT secret
	if cfg.JWT.Secret == "" {
		if isProd {
			return fmt.Errorf("JWT_SECRET is required in production")
		}
		// Generate ephemeral secret for development
		secret, err := generateSecret(32)
		if err != nil {
			return fmt.Errorf("failed to generate JWT secret: %w", err)
		}
		cfg.JWT.Secret = secret
		log.Warn().Msg("[SECURITY] JWT_SECRET not set; using ephemeral secret")
	}

	// Handle admin credentials
	if cfg.Admin.Username == "" || cfg.Admin.Password == "" {
		if isProd {
			log.Warn().Msg("[SECURITY] Admin credentials not set in production")
			cfg.Admin.Username = ""
			cfg.Admin.Password = ""
		}
	}

	// Warn about permissive CORS in production
	if isProd && len(cfg.CORS.AllowedOrigins) == 0 {
		log.Warn().Msg("[SECURITY] CORS_ALLOWED_ORIGINS not set in production")
	}

	return nil
}

// =============================================================================
// HELPERS
// =============================================================================

// IsProd returns true if running in production environment.
func (c *Config) IsProd() bool {
	return c.App.Env == "production" || c.App.Env == "prod"
}

// IsDev returns true if running in development environment.
func (c *Config) IsDev() bool {
	return c.App.Env == "development" || c.App.Env == "dev"
}

// generateSecret generates a cryptographically secure random secret.
func generateSecret(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(bytes), nil
}
