package handler

import (
	"net/http"

	"github.com/daily-journal/backend/internal/auth"
	"github.com/daily-journal/backend/internal/model"
	"github.com/daily-journal/backend/internal/repository"
	"github.com/daily-journal/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

type AuthHandler struct {
	db        *repository.DB
	jwtSecret string
}

func NewAuthHandler(db *repository.DB, jwtSecret string) *AuthHandler {
	return &AuthHandler{db: db, jwtSecret: jwtSecret}
}

// Login godoc
// @Summary      Login to SurrealDB
// @Description  Login and receive a JWT
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        creds  body      model.AuthRequest  true  "Credentials"
// @Success      200    {object}  model.AuthResponse
// @Router       /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	logger := log.Ctx(c.Request.Context())

	var req model.AuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationFailed(c, err)
		return
	}

	logger.Debug().Str("username", req.Username).Msg("login attempt")

	// We sign in against the "account" access
	token, err := h.db.Client.SignIn(c.Request.Context(), &model.DBAuth{
		Namespace: "daily_journal",
		Database:  "core",
		Scope:     "account",
		Access:    "account",
		Username:  req.Username,
		Password:  req.Password,
	})

	if err != nil {
		logger.Warn().Err(err).Str("username", req.Username).Msg("login failed")
		response.Unauthorized(c, "Invalid credentials")
		return
	}

	// In a real app we'd extract the user ID from the token
	// For now, we return the token and the username as ID
	claims, err := auth.ParseToken(token, h.jwtSecret)
	if err != nil {
		logger.Error().Err(err).Msg("failed to verify auth token")
		response.InternalError(c, err)
		return
	}

	logger.Info().Str("user_id", claims.ID).Msg("login successful")

	response.Success(c, http.StatusOK, model.AuthResponse{
		Token: token,
		User:  claims.ID,
	})
}

// Register godoc
// @Summary      Register a new user
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        creds  body      model.AuthRequest  true  "Credentials"
// @Success      200    {object}  model.AuthResponse
// @Router       /auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	logger := log.Ctx(c.Request.Context())

	var req model.AuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationFailed(c, err)
		return
	}

	logger.Debug().Str("username", req.Username).Msg("registration attempt")

	// SignUp creates the record and returns a token
	token, err := h.db.Client.SignUp(c.Request.Context(), &model.DBAuth{
		Namespace: "daily_journal",
		Database:  "core",
		Scope:     "account",
		Access:    "account",
		Username:  req.Username,
		Password:  req.Password,
	})

	if err != nil {
		logger.Error().Err(err).Str("username", req.Username).Msg("registration failed")
		response.InternalError(c, err)
		return
	}

	claims, err := auth.ParseToken(token, h.jwtSecret)
	if err != nil {
		logger.Error().Err(err).Msg("failed to verify auth token")
		response.InternalError(c, err)
		return
	}

	logger.Info().Str("user_id", claims.ID).Msg("user registered successfully")

	response.Success(c, http.StatusOK, model.AuthResponse{
		Token: token,
		User:  claims.ID,
	})
}
