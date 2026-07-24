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
	"github.com/lucid-logs/go-backend/internal/features/activities"
	"github.com/lucid-logs/go-backend/internal/features/activitylogs"
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
	"github.com/lucid-logs/go-backend/internal/features/units"
	"github.com/lucid-logs/go-backend/internal/features/users"
	"github.com/lucid-logs/go-backend/internal/shared/database"
	"github.com/lucid-logs/go-backend/internal/shared/middleware"
	"github.com/lucid-logs/go-backend/internal/shared/validator"
	"github.com/lucid-logs/go-backend/internal/web"
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
//
// @title          Lucid Logs API
// @version        1.0
// @description    Time tracking and productivity API with rich analytics.
// @host           localhost:8080
// @BasePath       /api/v1
// @securityDefinitions.apikey BearerAuth
// @in             header
// @name           Authorization
func NewRouter(cfg Config) *gin.Engine {
	// Initialize Gin router
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
	corsConfig := cors.DefaultConfig()

	if cfg.Cfg.IsDev() {
		// Development - allow all origins
		corsConfig.AllowAllOrigins = true
		corsConfig.AllowHeaders = append(corsConfig.AllowHeaders,
			"Authorization",
			"Content-Type",
			"X-Request-ID",
			"Accept",
			"Origin",
		)
	} else {
		// Production - restrict origins
		corsConfig.AllowOrigins = []string{
			"https://lucidlogs.app",
			"https://www.lucidlogs.app",
		}
	}
	corsConfig.AllowCredentials = true
	r.Use(cors.New(corsConfig))

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
		}))
		{
			// Activity logs (unified activity logging)
			activityRepo := activitylogs.NewRepository(cfg.DB)
			activityLogger := activitylogs.NewActivityLogger(activityRepo)
			activitylogs.RegisterRoutes(protected.Group("/activity"), activityRepo)

			// Emotion routes
			emotions.RegisterRoutes(protected.Group("/emotions"), cfg.DB)

			// Task routes (with activity logging)
			taskRepo := tasks.NewRepository(cfg.DB)
			taskService := tasks.NewService(taskRepo, activityLogger)
			tasks.RegisterRoutes(protected.Group("/tasks"), taskService, cfg.Validator)

			// Category routes
			categoryRepo := categories.NewRepository(cfg.DB)
			categoryService := categories.NewService(categoryRepo)
			categories.RegisterRoutes(protected.Group("/categories"), categoryService, cfg.Validator)

			// User routes
			userRepo := users.NewRepository(cfg.DB)
			userService := users.NewService(userRepo)
			users.RegisterRoutes(protected.Group("/users"), userService, cfg.Validator, cfg.Cfg.LLM)

			// Activity routes
			activitiesRepo := activities.NewRepository(cfg.DB)
			activitiesService := activities.NewService(activitiesRepo, taskService, nil)
			activitiesHandler := activities.NewHandler(activitiesService)
			activitiesHandler.RegisterRoutes(protected)

			// Goal Logs routes (nested under goals)
			goalLogsRepo := goallogs.NewRepository(cfg.DB)
			goalLogger := goallogs.NewGoalLoggerAdapter(goalLogsRepo)

			// Goal routes (with activity auto-creation and auto-logging)
			goalRepo := goals.NewRepository(cfg.DB)
			activityCreator := activities.NewGoalActivityCreator(activitiesRepo)
			goalService := goals.NewService(goalRepo, activityCreator, goalLogger)
			goals.RegisterRoutes(protected.Group("/goals"), goalService, cfg.Validator)

			// Goal Logs API routes (for fetching history)
			goallogs.RegisterRoutes(protected.Group("/goals"), goalLogsRepo)

			// Units routes
			unitsRepo := units.NewRepository(cfg.DB)
			unitsService := units.NewService(unitsRepo)
			units.RegisterRoutes(protected.Group("/units"), unitsService, cfg.Validator)

			// Task-Goals linking routes (nested under tasks)
			taskGoalsRepo := taskgoals.NewRepository(cfg.DB)
			taskGoalsService := taskgoals.NewService(taskGoalsRepo, taskService, goalService, goalLogger)
			taskgoals.RegisterRoutes(protected.Group("/tasks/:id/goals"), taskGoalsService, cfg.Validator)

			// Analytics routes
			analyticsRepo := analytics.NewRepository(cfg.DB)
			analyticsService := analytics.NewService(analyticsRepo)
			analytics.RegisterRoutes(protected.Group("/analytics"), analyticsService, cfg.Validator)

			// Retrospectives routes
			retroRepo := retrospectives.NewRepository(cfg.DB)
			retroService := retrospectives.NewService(retroRepo, analyticsRepo, userRepo)
			retrospectives.RegisterRoutes(protected.Group("/retrospectives"), retroService, cfg.Validator)
		}
	}

	// ==========================================================================
	// STATIC SPA (all-in-one mode)
	// ==========================================================================
	// When the frontend is embedded (all-in-one binary/image) and STATIC_ENABLED
	// is not explicitly false, serve the UI from the same origin as the API.
	// Registered last so it only catches routes the API didn't handle.
	if cfg.Cfg.Static.Enabled {
		web.RegisterSPA(r)
	}

	return r
}

// =============================================================================
// CORS CONFIGURATION
// =============================================================================

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
