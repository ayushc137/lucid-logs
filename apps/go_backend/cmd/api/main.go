// Package main is the entry point for the Daily Journal API server.
//
// This file handles:
//   - Configuration loading
//   - Logging setup
//   - Database connection
//   - Server startup with graceful shutdown
//
// Usage:
//
//	go run ./cmd/api
//	# or
//	make dev  (with hot reload)
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/daily-journal/go-backend/internal/config"
	"github.com/daily-journal/go-backend/internal/server"
	"github.com/daily-journal/go-backend/internal/shared/database"
	"github.com/daily-journal/go-backend/internal/shared/validator"
)

func main() {
	// =========================================================================
	// LOAD CONFIGURATION
	// =========================================================================

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// =========================================================================
	// SETUP LOGGING
	// =========================================================================

	setupLogging(cfg)

	log.Info().
		Str("env", cfg.App.Env).
		Int("port", cfg.Server.Port).
		Str("version", cfg.App.Version).
		Msg("Starting Daily Journal API")

	// =========================================================================
	// CONNECT TO DATABASE
	// =========================================================================

	ctx := context.Background()
	db, err := database.New(ctx, database.Config{
		URL:        cfg.Database.WebSocketURL(),
		Namespace:  cfg.Database.Namespace,
		Database:   cfg.Database.Database,
		Username:   cfg.Database.User,
		Password:   cfg.Database.Password,
		LogQueries: cfg.IsDev(),
	})
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to database")
	}
	defer db.Close(ctx)

	// =========================================================================
	// CREATE SERVICES
	// =========================================================================

	val := validator.New()

	// =========================================================================
	// BUILD ROUTER
	// =========================================================================

	router := server.NewRouter(server.Config{
		Cfg:       cfg,
		DB:        db,
		Validator: val,
	})

	// =========================================================================
	// CREATE HTTP SERVER
	// =========================================================================

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// =========================================================================
	// START SERVER
	// =========================================================================

	go func() {
		log.Info().
			Int("port", cfg.Server.Port).
			Msg("Server listening")
		log.Info().
			Str("url", fmt.Sprintf("http://localhost:%d/docs", cfg.Server.Port)).
			Msg("API documentation available")

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("Server failed")
		}
	}()

	// =========================================================================
	// GRACEFUL SHUTDOWN
	// =========================================================================

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("Shutting down server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("Server forced to shutdown")
	}

	log.Info().Msg("Server stopped")
}

// setupLogging configures zerolog based on environment.
func setupLogging(cfg *config.Config) {
	// Set log level
	if cfg.IsDev() {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
		// Pretty console output for development
		log.Logger = log.Output(zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
		})
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
		zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	}

	// Add default fields
	log.Logger = log.With().
		Str("service", "daily-journal-api").
		Str("version", cfg.App.Version).
		Logger()
}
