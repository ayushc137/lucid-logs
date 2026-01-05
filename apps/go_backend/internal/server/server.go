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

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	swaggerDocs "github.com/lucid-logs/go-backend/docs/swagger"
	"github.com/lucid-logs/go-backend/internal/config"
	"github.com/lucid-logs/go-backend/internal/features/analytics"
	"github.com/lucid-logs/go-backend/internal/features/auth"
	"github.com/lucid-logs/go-backend/internal/features/categories"
	"github.com/lucid-logs/go-backend/internal/features/emotions"
	"github.com/lucid-logs/go-backend/internal/features/goallogs"
	"github.com/lucid-logs/go-backend/internal/features/goals"
	"github.com/lucid-logs/go-backend/internal/features/health"
	"github.com/lucid-logs/go-backend/internal/features/retrospectives"
	"github.com/lucid-logs/go-backend/internal/features/taskgoals"
	"github.com/lucid-logs/go-backend/internal/features/tasks"
	"github.com/lucid-logs/go-backend/internal/features/templates"
	"github.com/lucid-logs/go-backend/internal/features/units"
	"github.com/lucid-logs/go-backend/internal/features/users"
	"github.com/lucid-logs/go-backend/internal/shared/database"
	"github.com/lucid-logs/go-backend/internal/shared/middleware"
	"github.com/lucid-logs/go-backend/internal/shared/validator"
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
func NewRouter(cfg Config) *gin.Engine {
	// Set gin mode
	if cfg.Cfg.IsDev() {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()

	// ==========================================================================
	// GLOBAL MIDDLEWARE
	// ==========================================================================

	// Trace ID injection (must run before logging)
	r.Use(middleware.Trace())

	// Request/response logging
	r.Use(middleware.Logger())

	// Panic recovery
	r.Use(middleware.Recovery())

	// CORS configuration
	r.Use(corsHandler(cfg.Cfg))

	// ==========================================================================
	// PUBLIC ROUTES
	// ==========================================================================

	// Basic health check (no auth required)
	health.RegisterRoutes(r.Group("/health"), cfg.DB)

	// API documentation
	r.GET("/docs", serveSwaggerUI)
	r.GET("/api-docs/openapi.json", serveOpenAPISpec)

	// ==========================================================================
	// API V1 ROUTES
	// ==========================================================================

	v1 := r.Group("/api/v1")
	{
		// Health check with DB status
		h := health.NewHandler(cfg.DB)
		v1.GET("/health", h.CheckWithDB)

		// Auth routes (public)
		authService := auth.NewService(cfg.DB, cfg.Cfg)
		auth.RegisterRoutes(v1.Group("/auth"), authService, cfg.Validator)

		// Protected routes (require authentication)
		protected := v1.Group("")
		protected.Use(middleware.Auth(middleware.AuthConfig{
			JWTSecret: cfg.Cfg.JWT.Secret,
			Namespace: cfg.Cfg.Database.Namespace,
			Database:  cfg.Cfg.Database.Database,
		}))
		{
			// Emotion routes
			emotions.RegisterRoutes(protected.Group("/emotions"), cfg.DB)

			// Task routes
			taskRepo := tasks.NewRepository(cfg.DB)
			taskService := tasks.NewService(taskRepo)
			tasks.RegisterRoutes(protected.Group("/tasks"), taskService, cfg.Validator)

			// Category routes
			categoryRepo := categories.NewRepository(cfg.DB)
			categoryService := categories.NewService(categoryRepo)
			categories.RegisterRoutes(protected.Group("/categories"), categoryService, cfg.Validator)

			// User routes
			userRepo := users.NewRepository(cfg.DB)
			userService := users.NewService(userRepo)
			users.RegisterRoutes(protected.Group("/users"), userService, cfg.Validator)

			// Template routes
			templateRepo := templates.NewRepository(cfg.DB)
			templateService := templates.NewService(templateRepo)
			templates.RegisterRoutes(protected.Group("/templates"), templateService, cfg.Validator)

			// Goal routes (with linked template auto-creation)
			goalRepo := goals.NewRepository(cfg.DB)
			templateCreator := templates.NewGoalTemplateCreator(templateRepo)
			goalService := goals.NewService(goalRepo, templateCreator)
			goals.RegisterRoutes(protected.Group("/goals"), goalService, cfg.Validator)

			// Goal Logs routes (nested under goals)
			goalLogsRepo := goallogs.NewRepository(cfg.DB)
			_ = goalLogsRepo // TODO: Add goal logs handler

			// Units routes
			unitsRepo := units.NewRepository(cfg.DB)
			unitsService := units.NewService(unitsRepo)
			units.RegisterRoutes(protected.Group("/units"), unitsService, cfg.Validator)

			// Task-Goals linking routes (nested under tasks)
			taskGoalsRepo := taskgoals.NewRepository(cfg.DB)
			taskGoalsService := taskgoals.NewService(taskGoalsRepo, taskService, goalService)
			taskgoals.RegisterRoutes(protected.Group("/tasks/:id/goals"), taskGoalsService, cfg.Validator)

			// Analytics routes
			analyticsRepo := analytics.NewRepository(cfg.DB)
			analyticsService := analytics.NewService(analyticsRepo)
			analytics.RegisterRoutes(protected.Group("/analytics"), analyticsService, cfg.Validator)

			// Retrospectives routes
			retroRepo := retrospectives.NewRepository(cfg.DB)
			retroService := retrospectives.NewService(retroRepo, analyticsRepo)
			retrospectives.RegisterRoutes(protected.Group("/retrospectives"), retroService, cfg.Validator)
		}
	}

	return r
}

// =============================================================================
// CORS CONFIGURATION
// =============================================================================

// corsHandler creates the CORS middleware based on configuration.
func corsHandler(cfg *config.Config) gin.HandlerFunc {
	// In development, be extremely permissive but respect AllowCredentials constraint
	// We cannot use AllowAllOrigins: true with AllowCredentials: true
	if cfg.IsDev() {
		return cors.New(cors.Config{
			AllowOriginFunc: func(origin string) bool {
				return true
			},
			AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
			AllowHeaders:     []string{"Origin", "Content-Length", "Content-Type", "Authorization", "Accept", "X-Requested-With"},
			ExposeHeaders:    []string{"Content-Length", "Content-Type"},
			AllowCredentials: true,
			MaxAge:           12 * 3600,
		})
	}

	return cors.New(cors.Config{
		AllowOrigins:     cfg.CORS.AllowedOrigins,
		AllowMethods:     cfg.CORS.AllowedMethods,
		AllowHeaders:     cfg.CORS.AllowedHeaders,
		AllowCredentials: true,
		MaxAge:           300,
	})
}

// =============================================================================
// API DOCUMENTATION
// =============================================================================

// serveSwaggerUI serves the Swagger UI HTML page.
func serveSwaggerUI(c *gin.Context) {
	html := `<!DOCTYPE html>
<html>
<head>
    <title>Lucid Logs</title>
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
        const AUTH_STORAGE_KEY = 'lucid-logs-swagger-token';

        function normalizeToken(token) {
            if (!token) return '';
            return token.replace(/^Bearer\s+/i, '').trim();
        }

        function attachPersistence(ui) {
            const saved = window.localStorage.getItem(AUTH_STORAGE_KEY);
            if (saved) {
                ui.preauthorizeApiKey('BearerAuth', saved);
            }

            const system = ui.getSystem();
            if (!system) {
                return;
            }

            const persistToken = () => {
                const authState = system.authSelectors?.authorized?.();
                const auth = authState && authState.toJS ? authState.toJS() : null;
                const token = auth?.BearerAuth?.value;
                if (token) {
                    window.localStorage.setItem(AUTH_STORAGE_KEY, normalizeToken(token));
                }
            };

            const clearToken = () => {
                window.localStorage.removeItem(AUTH_STORAGE_KEY);
            };

            system.events?.on?.('authorize', persistToken);
            system.events?.on?.('logout', clearToken);
        }

        const ui = SwaggerUIBundle({
            url: "/api-docs/openapi.json",
            dom_id: '#swagger-ui',
            presets: [
                SwaggerUIBundle.presets.apis,
                SwaggerUIBundle.SwaggerUIStandalonePreset
            ],
            layout: "BaseLayout",
            requestInterceptor: (req) => {
                const stored = window.localStorage.getItem(AUTH_STORAGE_KEY);
                if (stored && !req.headers.Authorization) {
                    req.headers.Authorization = 'Bearer ' + normalizeToken(stored);
                } else if (req.headers.Authorization) {
                    req.headers.Authorization = 'Bearer ' + normalizeToken(req.headers.Authorization);
                }
                return req;
            }
        });

        attachPersistence(ui);
    </script>
</body>
</html>`
	c.Header("Content-Type", "text/html")
	c.String(http.StatusOK, html)
}

// serveOpenAPISpec serves the OpenAPI specification.
func serveOpenAPISpec(c *gin.Context) {
	spec := swaggerDocs.SwaggerInfo.ReadDoc()
	c.Header("Content-Type", "application/json")
	c.String(http.StatusOK, spec)
}
