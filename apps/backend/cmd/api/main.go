package main

//go:generate swag init -g main.go

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/daily-journal/backend/docs"
	"github.com/daily-journal/backend/internal/config"
	"github.com/daily-journal/backend/internal/handler"
	"github.com/daily-journal/backend/internal/middleware"
	"github.com/daily-journal/backend/internal/repository"
	"github.com/daily-journal/backend/pkg/logger"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title           Daily Journal API
// @version         1.0
// @description     Backend for the Daily Journal Application
// @host            localhost:8080
// @BasePath        /api/v1
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load config")
	}

	// Initialize structured logging
	logger.Init(cfg.App.Env)

	db, err := repository.NewDB(cfg.DB)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to database")
	}
	defer db.Close()

	// Init DB Schema (Scope)
	if err := repository.InitSchema(db, repository.SchemaInitOptions{
		SchemaPath:    cfg.DB.SchemaPath,
		AppEnv:        cfg.App.Env,
		AdminUsername: cfg.Admin.Username,
		AdminPassword: cfg.Admin.Password,
		JWTSecret:     cfg.JWT.Secret,
	}); err != nil {
		log.Error().Err(err).Msg("Failed to init schema (this is normal if scope already exists)")
	}

	if cfg.App.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()

	r.Use(gin.Recovery())
	r.Use(logger.Middleware())
	r.Use(corsMiddleware())

	// Repositories
	taskRepo := repository.NewTaskRepository(db)

	// Handlers
	taskHandler := handler.NewTaskHandler(taskRepo)
	authHandler := handler.NewAuthHandler(db, cfg.JWT.Secret)

	r.GET("/health", handler.HealthCheck)
	swaggerHandler := ginSwagger.WrapHandler(
		swaggerFiles.Handler,
		ginSwagger.PersistAuthorization(true),
		ginSwagger.DocExpansion("none"),
	)
	r.GET("/swagger/*any", swaggerHandler)

	v1 := r.Group("/api/v1")
	{
		v1.GET("/health", handler.HealthCheck)
		// Auth Routes
		auth := v1.Group("/auth")
		{
			auth.POST("/login", authHandler.Login)
			auth.POST("/register", authHandler.Register)
		}

		// Protected Routes
		protected := v1.Group("/")
		protected.Use(middleware.AuthMiddleware(cfg.JWT.Secret))
		{
			tasks := protected.Group("/tasks")
			{
				tasks.POST("", taskHandler.Create)
				tasks.GET("", taskHandler.List)
				tasks.GET("/:id", taskHandler.Get)
				tasks.PUT("/:id", taskHandler.Update)
				tasks.DELETE("/:id", taskHandler.Delete)
			}
		}
	}

	srv := &http.Server{
		Addr:    ":" + cfg.HTTP.Port,
		Handler: r,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("listen: failed")
		}
	}()
	log.Info().
		Str("port", cfg.HTTP.Port).
		Str("env", cfg.App.Env).
		Msg("Server started")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info().Msg("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal().Err(err).Msg("Server forced to shutdown")
	}

	log.Info().Msg("Server stopped gracefully")
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}
