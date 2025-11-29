// Package server provides HTTP server setup and routing.
//
// This package handles:
//   - Router configuration with middleware
//   - Route registration for all features
//   - CORS configuration
//   - API documentation endpoints
package server

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/daily-journal/go-backend/internal/config"
	"github.com/daily-journal/go-backend/internal/features/auth"
	"github.com/daily-journal/go-backend/internal/features/categories"
	"github.com/daily-journal/go-backend/internal/features/health"
	"github.com/daily-journal/go-backend/internal/features/tasks"
	"github.com/daily-journal/go-backend/internal/shared/database"
	"github.com/daily-journal/go-backend/internal/shared/middleware"
	"github.com/daily-journal/go-backend/internal/shared/validator"
)

// =============================================================================
// SERVER CONFIGURATION
// =============================================================================

// Config holds server dependencies.
type Config struct {
	Cfg       *config.Config
	DB        *database.DB
	Validator *validator.Validator
}

// =============================================================================
// ROUTER SETUP
// =============================================================================

// NewRouter creates and configures the HTTP router.
//
// Architecture:
//
//	/health                     - Basic health check
//	/docs                       - Swagger UI
//	/api/v1/
//	    /health                 - Health with DB check
//	    /auth/login             - User login
//	    /auth/register          - User registration
//	    /tasks                  - Task CRUD (protected)
//	    /categories             - Category CRUD (protected)
func NewRouter(cfg Config) *chi.Mux {
	r := chi.NewRouter()

	// ==========================================================================
	// GLOBAL MIDDLEWARE
	// ==========================================================================

	// Request ID for tracing
	r.Use(chimiddleware.RequestID)

	// Real IP detection (behind proxy)
	r.Use(chimiddleware.RealIP)

	// Request/response logging
	r.Use(middleware.Logger)

	// Panic recovery
	r.Use(middleware.Recovery)

	// CORS configuration
	r.Use(corsHandler(cfg.Cfg))

	// ==========================================================================
	// PUBLIC ROUTES
	// ==========================================================================

	// Basic health check (no auth required)
	r.Mount("/health", health.Routes(cfg.DB))

	// API documentation
	r.Get("/docs", serveSwaggerUI)
	r.Get("/api-docs/openapi.json", serveOpenAPISpec)

	// ==========================================================================
	// API V1 ROUTES
	// ==========================================================================

	r.Route("/api/v1", func(r chi.Router) {
		// Health check with DB status
		h := health.NewHandler(cfg.DB)
		r.Get("/health", h.CheckWithDB)

		// Auth routes (public)
		authService := auth.NewService(cfg.DB, cfg.Cfg)
		r.Mount("/auth", auth.Routes(authService, cfg.Validator))

		// Protected routes (require authentication)
		r.Group(func(r chi.Router) {
			// Auth middleware
			r.Use(middleware.Auth(middleware.AuthConfig{
				JWTSecret: cfg.Cfg.JWT.Secret,
				Namespace: cfg.Cfg.Database.Namespace,
				Database:  cfg.Cfg.Database.Database,
			}))

			// Task routes
			taskRepo := tasks.NewRepository(cfg.DB)
			taskService := tasks.NewService(taskRepo)
			r.Mount("/tasks", tasks.Routes(taskService, cfg.Validator))

			// Category routes
			categoryRepo := categories.NewRepository(cfg.DB)
			categoryService := categories.NewService(categoryRepo)
			r.Mount("/categories", categories.Routes(categoryService, cfg.Validator))
		})
	})

	return r
}

// =============================================================================
// CORS CONFIGURATION
// =============================================================================

// corsHandler creates the CORS middleware based on configuration.
func corsHandler(cfg *config.Config) func(http.Handler) http.Handler {
	allowedOrigins := cfg.CORS.AllowedOrigins
	if cfg.IsDev() || len(allowedOrigins) == 0 {
		allowedOrigins = []string{"*"}
	}

	return cors.Handler(cors.Options{
		AllowedOrigins:   allowedOrigins,
		AllowedMethods:   cfg.CORS.AllowedMethods,
		AllowedHeaders:   cfg.CORS.AllowedHeaders,
		AllowCredentials: !cfg.IsDev(),
		MaxAge:           300,
	})
}

// =============================================================================
// API DOCUMENTATION
// =============================================================================

// serveSwaggerUI serves the Swagger UI HTML page.
func serveSwaggerUI(w http.ResponseWriter, r *http.Request) {
	html := `<!DOCTYPE html>
<html>
<head>
    <title>Daily Journal API</title>
    <meta charset="utf-8"/>
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
    <style>
        body { margin: 0; }
        .swagger-ui .topbar { display: none; }
    </style>
</head>
<body>
    <div id="swagger-ui"></div>
    <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
    <script>
        SwaggerUIBundle({
            url: "/api-docs/openapi.json",
            dom_id: '#swagger-ui',
            presets: [
                SwaggerUIBundle.presets.apis,
                SwaggerUIBundle.SwaggerUIStandalonePreset
            ],
            layout: "BaseLayout"
        });
    </script>
</body>
</html>`
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

// serveOpenAPISpec serves the OpenAPI specification.
func serveOpenAPISpec(w http.ResponseWriter, r *http.Request) {
	spec := `{
  "openapi": "3.0.0",
  "info": {
    "title": "Daily Journal API",
    "version": "1.0.0",
    "description": "Backend API for the Daily Journal Application"
  },
  "servers": [{"url": "/"}],
  "components": {
    "securitySchemes": {
      "BearerAuth": {"type": "http", "scheme": "bearer", "bearerFormat": "JWT"}
    }
  },
  "paths": {
    "/health": {
      "get": {"summary": "Health check", "tags": ["health"], "responses": {"200": {"description": "OK"}}}
    },
    "/api/v1/auth/login": {
      "post": {
        "summary": "User login", "tags": ["auth"],
        "requestBody": {"content": {"application/json": {"schema": {"type": "object", "properties": {"username": {"type": "string"}, "password": {"type": "string"}}}}}},
        "responses": {"200": {"description": "Login successful"}}
      }
    },
    "/api/v1/auth/register": {
      "post": {
        "summary": "User registration", "tags": ["auth"],
        "requestBody": {"content": {"application/json": {"schema": {"type": "object", "properties": {"username": {"type": "string"}, "password": {"type": "string"}}}}}},
        "responses": {"200": {"description": "Registration successful"}}
      }
    },
    "/api/v1/tasks": {
      "get": {"summary": "List tasks", "tags": ["tasks"], "security": [{"BearerAuth": []}], "responses": {"200": {"description": "OK"}}},
      "post": {"summary": "Create task", "tags": ["tasks"], "security": [{"BearerAuth": []}], "responses": {"201": {"description": "Created"}}}
    },
    "/api/v1/tasks/{id}": {
      "get": {"summary": "Get task", "tags": ["tasks"], "security": [{"BearerAuth": []}], "responses": {"200": {"description": "OK"}}},
      "put": {"summary": "Update task", "tags": ["tasks"], "security": [{"BearerAuth": []}], "responses": {"200": {"description": "OK"}}},
      "delete": {"summary": "Delete task", "tags": ["tasks"], "security": [{"BearerAuth": []}], "responses": {"200": {"description": "OK"}}}
    },
    "/api/v1/categories": {
      "get": {"summary": "List categories", "tags": ["categories"], "security": [{"BearerAuth": []}], "responses": {"200": {"description": "OK"}}},
      "post": {"summary": "Create category", "tags": ["categories"], "security": [{"BearerAuth": []}], "responses": {"201": {"description": "Created"}}}
    },
    "/api/v1/categories/{id}": {
      "get": {"summary": "Get category", "tags": ["categories"], "security": [{"BearerAuth": []}], "responses": {"200": {"description": "OK"}}},
      "put": {"summary": "Update category", "tags": ["categories"], "security": [{"BearerAuth": []}], "responses": {"200": {"description": "OK"}}},
      "delete": {"summary": "Delete category", "tags": ["categories"], "security": [{"BearerAuth": []}], "responses": {"200": {"description": "OK"}}}
    }
  }
}`
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(spec))
}
