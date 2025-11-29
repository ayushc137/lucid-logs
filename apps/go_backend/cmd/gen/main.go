// Command gen generates feature scaffolds for the Go backend.
//
// Usage:
//
//	go run cmd/gen/main.go <feature_name>
//	go run cmd/gen/main.go comments
//
// This creates a new feature with:
//   - models.go     (domain types)
//   - repository.go (data access using SurrealDB SDK)
//   - service.go    (business logic)
//   - handler.go    (HTTP handlers)
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"unicode"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run cmd/gen/main.go <feature_name>")
		fmt.Println("Example: go run cmd/gen/main.go comments")
		os.Exit(1)
	}

	name := strings.ToLower(os.Args[1])
	nameTitle := toTitle(name)
	namePlural := name + "s"

	data := TemplateData{
		Name:       name,
		NameTitle:  nameTitle,
		NamePlural: namePlural,
	}

	featureDir := filepath.Join("internal", "features", name)

	// Create directory
	if err := os.MkdirAll(featureDir, 0755); err != nil {
		fmt.Printf("Error creating directory: %v\n", err)
		os.Exit(1)
	}

	// Generate files
	files := map[string]string{
		"models.go":     modelsTemplate,
		"repository.go": repositoryTemplate,
		"service.go":    serviceTemplate,
		"handler.go":    handlerTemplate,
	}

	for filename, tmplContent := range files {
		filePath := filepath.Join(featureDir, filename)
		if err := generateFile(filePath, tmplContent, data); err != nil {
			fmt.Printf("Error generating %s: %v\n", filename, err)
			os.Exit(1)
		}
		fmt.Printf("  Created: %s\n", filePath)
	}

	fmt.Println()
	fmt.Printf("Feature '%s' created successfully!\n", name)
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  1. Review and customize the generated files")
	fmt.Printf("  2. Add migration: make migrate-create name=add_%s\n", namePlural)
	fmt.Println("  3. Register routes in internal/server/server.go:")
	fmt.Println()
	fmt.Printf("     %sRepo := %s.NewRepository(cfg.DB)\n", name, name)
	fmt.Printf("     %sService := %s.NewService(%sRepo)\n", name, name, name)
	fmt.Printf("     r.Mount(\"/%s\", %s.Routes(%sService, cfg.Validator))\n", namePlural, name, name)
}

// TemplateData holds the data for template rendering.
type TemplateData struct {
	Name       string // lowercase: comment
	NameTitle  string // TitleCase: Comment
	NamePlural string // plural: comments
}

func toTitle(s string) string {
	if s == "" {
		return s
	}
	runes := []rune(s)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

func generateFile(path, tmplContent string, data TemplateData) error {
	tmpl, err := template.New("file").Parse(tmplContent)
	if err != nil {
		return err
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	return tmpl.Execute(f, data)
}

// =============================================================================
// TEMPLATES
// =============================================================================

const modelsTemplate = `// Package {{.Name}} provides {{.Name}} management functionality.
//
// This package implements:
//   - CRUD operations for {{.Name}}s
//   - Soft delete support
//   - Pagination
package {{.Name}}

import "time"

// =============================================================================
// DOMAIN MODEL
// =============================================================================

// {{.NameTitle}} represents a {{.Name}} entity in the system.
//
// Fields marked with json:"-" are hidden from API responses.
// System-managed fields (created_at, updated_at, deleted_at) are read-only.
type {{.NameTitle}} struct {
	ID        string     ` + "`" + `json:"id,omitempty"` + "`" + `
	Name      string     ` + "`" + `json:"name"` + "`" + `
	// TODO: Add more fields
	CreatedAt time.Time  ` + "`" + `json:"created_at"` + "`" + `
	UpdatedAt time.Time  ` + "`" + `json:"updated_at"` + "`" + `
	DeletedAt *time.Time ` + "`" + `json:"deleted_at,omitempty"` + "`" + `
	CreatedBy string     ` + "`" + `json:"-"` + "`" + ` // Hidden: ownership field
	UpdatedBy string     ` + "`" + `json:"-"` + "`" + ` // Hidden: audit field
}

// =============================================================================
// REQUEST TYPES
// =============================================================================

// CreateRequest is the request payload for creating a {{.Name}}.
//
// @Description Request payload for creating a {{.Name}}
type CreateRequest struct {
	Name string ` + "`" + `json:"name" validate:"required,min=1,max=100" example:"Example"` + "`" + `
	// TODO: Add more fields
}

// UpdateRequest is the request payload for updating a {{.Name}}.
//
// All fields are optional. Only provided fields will be updated.
//
// @Description Request payload for updating a {{.Name}}
type UpdateRequest struct {
	Name *string ` + "`" + `json:"name,omitempty" validate:"omitempty,min=1,max=100" example:"Updated"` + "`" + `
	// TODO: Add more fields
}

// =============================================================================
// CONSTANTS
// =============================================================================

const (
	// Table is the SurrealDB table name for {{.Name}}s.
	Table = "{{.NamePlural}}"
)
`

const repositoryTemplate = `package {{.Name}}

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"strings"
	"time"

	"github.com/daily-journal/go-backend/internal/shared/database"
	"github.com/daily-journal/go-backend/internal/shared/errors"
	"github.com/daily-journal/go-backend/internal/shared/pagination"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// =============================================================================
// REPOSITORY INTERFACE
// =============================================================================

// Repository defines the {{.Name}} data access interface.
//
// This interface enables:
//   - Dependency injection
//   - Easy mocking for tests
//   - Swapping implementations (e.g., caching layer)
type Repository interface {
	// FindByID retrieves a {{.Name}} by ID for a specific user.
	FindByID(ctx context.Context, id, userID string) (*{{.NameTitle}}, error)

	// FindPaginated retrieves {{.Name}}s for a user with pagination.
	FindPaginated(ctx context.Context, userID string, params pagination.Params) ([]*{{.NameTitle}}, int64, error)

	// Create creates a new {{.Name}}.
	Create(ctx context.Context, req *CreateRequest, userID string) (*{{.NameTitle}}, error)

	// Update updates an existing {{.Name}}.
	Update(ctx context.Context, id string, req *UpdateRequest, userID string) (*{{.NameTitle}}, error)

	// Delete soft-deletes a {{.Name}}.
	Delete(ctx context.Context, id, userID string) error
}

// =============================================================================
// REPOSITORY IMPLEMENTATION
// =============================================================================

type repository struct {
	db     *database.DB
	logger zerolog.Logger
}

// NewRepository creates a new {{.Name}} Repository.
func NewRepository(db *database.DB) Repository {
	return &repository{
		db:     db,
		logger: log.With().Str("repository", "{{.Name}}").Logger(),
	}
}

// =============================================================================
// DATABASE MODEL
// =============================================================================

// {{.Name}}DB is the internal database representation.
type {{.Name}}DB struct {
	ID        string     ` + "`" + `json:"id,omitempty"` + "`" + `
	Name      string     ` + "`" + `json:"name"` + "`" + `
	CreatedAt time.Time  ` + "`" + `json:"created_at"` + "`" + `
	UpdatedAt time.Time  ` + "`" + `json:"updated_at"` + "`" + `
	DeletedAt *time.Time ` + "`" + `json:"deleted_at,omitempty"` + "`" + `
	CreatedBy string     ` + "`" + `json:"created_by"` + "`" + `
	UpdatedBy string     ` + "`" + `json:"updated_by"` + "`" + `
}

func (e *{{.Name}}DB) toModel() *{{.NameTitle}} {
	return &{{.NameTitle}}{
		ID:        e.ID,
		Name:      e.Name,
		CreatedAt: e.CreatedAt,
		UpdatedAt: e.UpdatedAt,
		DeletedAt: e.DeletedAt,
		CreatedBy: e.CreatedBy,
		UpdatedBy: e.UpdatedBy,
	}
}

// =============================================================================
// FIND OPERATIONS
// =============================================================================

func (r *repository) FindByID(ctx context.Context, id, userID string) (*{{.NameTitle}}, error) {
	recordID := formatID(id)

	entity, err := database.QueryFirst[{{.Name}}DB](ctx, r.db, ` + "`" + `
		SELECT * FROM type::thing($id) WHERE deleted_at = NONE
	` + "`" + `, map[string]any{
		"id": recordID,
	})
	if err != nil {
		r.logger.Error().Err(err).Str("id", id).Msg("query failed")
		return nil, err
	}

	if entity == nil {
		return nil, errors.ErrNotFound
	}

	// Verify ownership
	if entity.CreatedBy != userID || entity.DeletedAt != nil {
		return nil, errors.ErrNotFound
	}

	return entity.toModel(), nil
}

func (r *repository) FindPaginated(ctx context.Context, userID string, params pagination.Params) ([]*{{.NameTitle}}, int64, error) {
	// Get count
	total, err := database.QueryScalar[float64](ctx, r.db, ` + "`" + `
		SELECT count() FROM {{.NamePlural}} WHERE created_by = $user AND deleted_at = NONE GROUP ALL
	` + "`" + `, map[string]any{
		"user": userID,
	})
	if err != nil {
		r.logger.Error().Err(err).Str("user_id", userID).Msg("count failed")
		return nil, 0, err
	}

	// Get records
	entities, err := database.QueryAll[{{.Name}}DB](ctx, r.db, ` + "`" + `
		SELECT * FROM {{.NamePlural}}
		WHERE created_by = $user AND deleted_at = NONE
		ORDER BY created_at DESC
		LIMIT $limit START $offset
	` + "`" + `, map[string]any{
		"user":   userID,
		"limit":  params.Limit,
		"offset": params.Offset,
	})
	if err != nil {
		r.logger.Error().Err(err).Str("user_id", userID).Msg("list failed")
		return nil, 0, err
	}

	result := make([]*{{.NameTitle}}, len(entities))
	for i := range entities {
		result[i] = entities[i].toModel()
	}

	return result, int64(total), nil
}

// =============================================================================
// CREATE OPERATION
// =============================================================================

type createData struct {
	Name      string ` + "`" + `json:"name"` + "`" + `
	CreatedBy string ` + "`" + `json:"created_by"` + "`" + `
	UpdatedBy string ` + "`" + `json:"updated_by"` + "`" + `
	CreatedAt string ` + "`" + `json:"created_at"` + "`" + `
	UpdatedAt string ` + "`" + `json:"updated_at"` + "`" + `
}

func (r *repository) Create(ctx context.Context, req *CreateRequest, userID string) (*{{.NameTitle}}, error) {
	recordID := generateID()
	now := time.Now().UTC().Format(time.RFC3339)

	data := createData{
		Name:      req.Name,
		CreatedBy: userID,
		UpdatedBy: userID,
		CreatedAt: now,
		UpdatedAt: now,
	}

	_, err := database.Create[{{.Name}}DB](ctx, r.db, recordID, data)
	if err != nil {
		r.logger.Error().Err(err).Str("id", recordID).Msg("create failed")
		return nil, err
	}

	r.logger.Info().Str("id", recordID).Msg("created")
	return r.FindByID(ctx, recordID, userID)
}

// =============================================================================
// UPDATE OPERATION
// =============================================================================

type mergeData struct {
	Name      *string ` + "`" + `json:"name,omitempty"` + "`" + `
	UpdatedBy string  ` + "`" + `json:"updated_by"` + "`" + `
	UpdatedAt string  ` + "`" + `json:"updated_at"` + "`" + `
}

func (r *repository) Update(ctx context.Context, id string, req *UpdateRequest, userID string) (*{{.NameTitle}}, error) {
	// Verify ownership
	if _, err := r.FindByID(ctx, id, userID); err != nil {
		return nil, err
	}

	recordID := formatID(id)
	data := mergeData{
		UpdatedBy: userID,
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
	}

	if req.Name != nil {
		data.Name = req.Name
	}

	_, err := database.Merge[{{.Name}}DB](ctx, r.db, recordID, data)
	if err != nil {
		r.logger.Error().Err(err).Str("id", id).Msg("update failed")
		return nil, err
	}

	r.logger.Info().Str("id", id).Msg("updated")
	return r.FindByID(ctx, id, userID)
}

// =============================================================================
// DELETE OPERATION
// =============================================================================

type softDeleteData struct {
	DeletedAt string ` + "`" + `json:"deleted_at"` + "`" + `
	UpdatedBy string ` + "`" + `json:"updated_by"` + "`" + `
	UpdatedAt string ` + "`" + `json:"updated_at"` + "`" + `
}

func (r *repository) Delete(ctx context.Context, id, userID string) error {
	// Verify ownership
	if _, err := r.FindByID(ctx, id, userID); err != nil {
		return err
	}

	recordID := formatID(id)
	now := time.Now().UTC().Format(time.RFC3339)

	_, err := database.Merge[{{.Name}}DB](ctx, r.db, recordID, softDeleteData{
		DeletedAt: now,
		UpdatedBy: userID,
		UpdatedAt: now,
	})
	if err != nil {
		r.logger.Error().Err(err).Str("id", id).Msg("delete failed")
		return err
	}

	r.logger.Info().Str("id", id).Msg("soft-deleted")
	return nil
}

// =============================================================================
// HELPERS
// =============================================================================

func formatID(id string) string {
	if strings.HasPrefix(id, Table+":") {
		return id
	}
	return Table + ":" + id
}

func generateID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return Table + ":" + hex.EncodeToString(bytes)
}
`

const serviceTemplate = `package {{.Name}}

import (
	"context"

	"github.com/daily-journal/go-backend/internal/shared/pagination"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// =============================================================================
// SERVICE INTERFACE
// =============================================================================

// Service defines the {{.Name}} business logic interface.
//
// This interface enables:
//   - Dependency injection in handlers
//   - Easy mocking for tests
//   - Decoupling from repository implementation
type Service interface {
	// List retrieves paginated {{.Name}}s for a user.
	List(ctx context.Context, userID string, params pagination.Params) (pagination.Response[*{{.NameTitle}}], error)

	// Get retrieves a single {{.Name}} by ID.
	Get(ctx context.Context, id, userID string) (*{{.NameTitle}}, error)

	// Create creates a new {{.Name}}.
	Create(ctx context.Context, req *CreateRequest, userID string) (*{{.NameTitle}}, error)

	// Update updates an existing {{.Name}}.
	Update(ctx context.Context, id string, req *UpdateRequest, userID string) (*{{.NameTitle}}, error)

	// Delete soft-deletes a {{.Name}}.
	Delete(ctx context.Context, id, userID string) error
}

// =============================================================================
// SERVICE IMPLEMENTATION
// =============================================================================

type service struct {
	repo   Repository
	logger zerolog.Logger
}

// NewService creates a new {{.Name}} Service.
func NewService(repo Repository) Service {
	return &service{
		repo:   repo,
		logger: log.With().Str("service", "{{.Name}}").Logger(),
	}
}

// =============================================================================
// LIST
// =============================================================================

func (s *service) List(ctx context.Context, userID string, params pagination.Params) (pagination.Response[*{{.NameTitle}}], error) {
	s.logger.Debug().
		Str("user_id", userID).
		Int("limit", params.Limit).
		Int("offset", params.Offset).
		Msg("listing {{.Name}}s")

	items, total, err := s.repo.FindPaginated(ctx, userID, params)
	if err != nil {
		return pagination.Response[*{{.NameTitle}}]{}, err
	}

	return pagination.NewResponse(items, total, params), nil
}

// =============================================================================
// GET
// =============================================================================

func (s *service) Get(ctx context.Context, id, userID string) (*{{.NameTitle}}, error) {
	s.logger.Debug().
		Str("id", id).
		Str("user_id", userID).
		Msg("getting {{.Name}}")

	return s.repo.FindByID(ctx, id, userID)
}

// =============================================================================
// CREATE
// =============================================================================

func (s *service) Create(ctx context.Context, req *CreateRequest, userID string) (*{{.NameTitle}}, error) {
	s.logger.Debug().
		Str("user_id", userID).
		Str("name", req.Name).
		Msg("creating {{.Name}}")

	entity, err := s.repo.Create(ctx, req, userID)
	if err != nil {
		s.logger.Error().Err(err).Msg("create failed")
		return nil, err
	}

	s.logger.Info().Str("id", entity.ID).Msg("{{.Name}} created")
	return entity, nil
}

// =============================================================================
// UPDATE
// =============================================================================

func (s *service) Update(ctx context.Context, id string, req *UpdateRequest, userID string) (*{{.NameTitle}}, error) {
	s.logger.Debug().
		Str("id", id).
		Str("user_id", userID).
		Msg("updating {{.Name}}")

	entity, err := s.repo.Update(ctx, id, req, userID)
	if err != nil {
		s.logger.Error().Err(err).Str("id", id).Msg("update failed")
		return nil, err
	}

	s.logger.Info().Str("id", id).Msg("{{.Name}} updated")
	return entity, nil
}

// =============================================================================
// DELETE
// =============================================================================

func (s *service) Delete(ctx context.Context, id, userID string) error {
	s.logger.Debug().
		Str("id", id).
		Str("user_id", userID).
		Msg("deleting {{.Name}}")

	if err := s.repo.Delete(ctx, id, userID); err != nil {
		s.logger.Error().Err(err).Str("id", id).Msg("delete failed")
		return err
	}

	s.logger.Info().Str("id", id).Msg("{{.Name}} deleted")
	return nil
}
`

const handlerTemplate = `package {{.Name}}

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/daily-journal/go-backend/internal/shared/errors"
	"github.com/daily-journal/go-backend/internal/shared/middleware"
	"github.com/daily-journal/go-backend/internal/shared/pagination"
	"github.com/daily-journal/go-backend/internal/shared/response"
	"github.com/daily-journal/go-backend/internal/shared/validator"
	"github.com/rs/zerolog/log"
)

// =============================================================================
// HANDLER
// =============================================================================

// Handler handles {{.Name}} HTTP endpoints.
type Handler struct {
	service   Service
	validator *validator.Validator
}

// NewHandler creates a new {{.Name}} Handler.
func NewHandler(service Service, validator *validator.Validator) *Handler {
	return &Handler{
		service:   service,
		validator: validator,
	}
}

// =============================================================================
// ROUTES
// =============================================================================

// Routes returns the {{.Name}} routes.
//
// Routes registered:
//   - GET    /        : List {{.Name}}s with pagination
//   - POST   /        : Create a new {{.Name}}
//   - GET    /{id}    : Get {{.Name}} by ID
//   - PUT    /{id}    : Update {{.Name}}
//   - DELETE /{id}    : Soft delete {{.Name}}
func Routes(service Service, validator *validator.Validator) chi.Router {
	r := chi.NewRouter()
	h := NewHandler(service, validator)

	r.Get("/", h.List)
	r.Post("/", h.Create)
	r.Get("/{id}", h.Get)
	r.Put("/{id}", h.Update)
	r.Delete("/{id}", h.Delete)

	return r
}

// =============================================================================
// LIST
// =============================================================================

// List handles GET /{{.NamePlural}} - list with pagination.
//
// @Summary      List {{.Name}}s
// @Description  Get paginated list of {{.Name}}s for the authenticated user
// @Tags         {{.NamePlural}}
// @Produce      json
// @Param        limit  query int false "Items per page (default 20, max 100)"
// @Param        offset query int false "Items to skip (default 0)"
// @Success      200 {object} pagination.Response[{{.NameTitle}}]
// @Failure      401 {object} response.APIResponse
// @Failure      500 {object} response.APIResponse
// @Security     BearerAuth
// @Router       /api/v1/{{.NamePlural}} [get]
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	user, appErr := middleware.MustGetAuthenticatedUser(r.Context())
	if appErr != nil {
		response.Error(w, appErr)
		return
	}

	params := pagination.FromRequest(r)

	log.Debug().
		Str("user_id", user.UserID).
		Int("limit", params.Limit).
		Int("offset", params.Offset).
		Msg("listing {{.Name}}s")

	resp, err := h.service.List(r.Context(), user.UserID, params)
	if err != nil {
		response.ErrorFromErr(w, err)
		return
	}

	response.OK(w, resp)
}

// =============================================================================
// GET
// =============================================================================

// Get handles GET /{{.NamePlural}}/{id} - get by ID.
//
// @Summary      Get {{.Name}} by ID
// @Description  Get a single {{.Name}} by its ID
// @Tags         {{.NamePlural}}
// @Produce      json
// @Param        id path string true "{{.NameTitle}} ID"
// @Success      200 {object} {{.NameTitle}}
// @Failure      401 {object} response.APIResponse
// @Failure      404 {object} response.APIResponse
// @Failure      500 {object} response.APIResponse
// @Security     BearerAuth
// @Router       /api/v1/{{.NamePlural}}/{id} [get]
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	user, appErr := middleware.MustGetAuthenticatedUser(r.Context())
	if appErr != nil {
		response.Error(w, appErr)
		return
	}

	id := chi.URLParam(r, "id")

	entity, err := h.service.Get(r.Context(), id, user.UserID)
	if err != nil {
		if errors.Is(err, errors.ErrNotFound) {
			response.NotFound(w)
			return
		}
		response.ErrorFromErr(w, err)
		return
	}

	response.OK(w, entity)
}

// =============================================================================
// CREATE
// =============================================================================

// Create handles POST /{{.NamePlural}} - create a new {{.Name}}.
//
// @Summary      Create {{.Name}}
// @Description  Create a new {{.Name}}
// @Tags         {{.NamePlural}}
// @Accept       json
// @Produce      json
// @Param        request body CreateRequest true "{{.NameTitle}} data"
// @Success      201 {object} {{.NameTitle}}
// @Failure      400 {object} response.APIResponse
// @Failure      401 {object} response.APIResponse
// @Failure      500 {object} response.APIResponse
// @Security     BearerAuth
// @Router       /api/v1/{{.NamePlural}} [post]
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	user, appErr := middleware.MustGetAuthenticatedUser(r.Context())
	if appErr != nil {
		response.Error(w, appErr)
		return
	}

	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid JSON body")
		return
	}

	if errs := h.validator.Validate(&req); errs != nil {
		response.ValidationFailed(w, errs)
		return
	}

	entity, err := h.service.Create(r.Context(), &req, user.UserID)
	if err != nil {
		response.ErrorFromErr(w, err)
		return
	}

	response.Created(w, entity)
}

// =============================================================================
// UPDATE
// =============================================================================

// Update handles PUT /{{.NamePlural}}/{id} - update a {{.Name}}.
//
// @Summary      Update {{.Name}}
// @Description  Update an existing {{.Name}}
// @Tags         {{.NamePlural}}
// @Accept       json
// @Produce      json
// @Param        id      path string        true "{{.NameTitle}} ID"
// @Param        request body UpdateRequest true "Update data"
// @Success      200 {object} {{.NameTitle}}
// @Failure      400 {object} response.APIResponse
// @Failure      401 {object} response.APIResponse
// @Failure      404 {object} response.APIResponse
// @Failure      500 {object} response.APIResponse
// @Security     BearerAuth
// @Router       /api/v1/{{.NamePlural}}/{id} [put]
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	user, appErr := middleware.MustGetAuthenticatedUser(r.Context())
	if appErr != nil {
		response.Error(w, appErr)
		return
	}

	id := chi.URLParam(r, "id")

	var req UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid JSON body")
		return
	}

	if errs := h.validator.Validate(&req); errs != nil {
		response.ValidationFailed(w, errs)
		return
	}

	entity, err := h.service.Update(r.Context(), id, &req, user.UserID)
	if err != nil {
		if errors.Is(err, errors.ErrNotFound) {
			response.NotFound(w)
			return
		}
		response.ErrorFromErr(w, err)
		return
	}

	response.OK(w, entity)
}

// =============================================================================
// DELETE
// =============================================================================

// Delete handles DELETE /{{.NamePlural}}/{id} - soft delete a {{.Name}}.
//
// @Summary      Delete {{.Name}}
// @Description  Soft delete a {{.Name}}
// @Tags         {{.NamePlural}}
// @Produce      json
// @Param        id path string true "{{.NameTitle}} ID"
// @Success      200 {object} response.OperationMessage
// @Failure      401 {object} response.APIResponse
// @Failure      404 {object} response.APIResponse
// @Failure      500 {object} response.APIResponse
// @Security     BearerAuth
// @Router       /api/v1/{{.NamePlural}}/{id} [delete]
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	user, appErr := middleware.MustGetAuthenticatedUser(r.Context())
	if appErr != nil {
		response.Error(w, appErr)
		return
	}

	id := chi.URLParam(r, "id")

	err := h.service.Delete(r.Context(), id, user.UserID)
	if err != nil {
		if errors.Is(err, errors.ErrNotFound) {
			response.NotFound(w)
			return
		}
		response.ErrorFromErr(w, err)
		return
	}

	response.Message(w, http.StatusOK, "{{.NameTitle}} deleted")
}
`
