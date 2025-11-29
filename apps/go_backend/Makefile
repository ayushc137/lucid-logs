# =============================================================================
# Daily Journal Go Backend - Makefile
# =============================================================================
# Run 'make help' to see available commands

# Go parameters
GOCMD := go
GOBUILD := $(GOCMD) build
GORUN := $(GOCMD) run
GOTEST := $(GOCMD) test
GOMOD := $(GOCMD) mod
GOFMT := gofumpt

# Linting
GOLINT := golangci-lint

# Binary names
BINARY := go-backend
MIGRATE_BINARY := migrate

# Directories
BUILD_DIR := ./build
CMD_DIR := ./cmd

# Build flags
LDFLAGS := -ldflags="-w -s"

.PHONY: all build clean test lint fmt run dev deps docs migrate help tools setup

# =============================================================================
# MAIN TARGETS
# =============================================================================

## help: Show this help message
help:
	@echo ''
	@echo 'Usage:'
	@echo '  make <target>'
	@echo ''
	@echo 'Targets:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'
	@echo ''

## all: Build all binaries
all: build

## build: Build the API server binary
build:
	@echo "==> Building..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY) $(CMD_DIR)/api
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(MIGRATE_BINARY) $(CMD_DIR)/migrate
	@echo "==> Build complete: $(BUILD_DIR)/"

## clean: Remove build artifacts
clean:
	@echo "==> Cleaning..."
	@rm -rf $(BUILD_DIR) tmp/
	@echo "==> Clean complete"

# =============================================================================
# QUICK START
# =============================================================================

## setup: Complete project setup (deps + tools)
setup: deps tools
	@echo ""
	@echo "==> Setup complete! Next steps:"
	@echo "    1. Copy env.example to .env and configure"
	@echo "    2. Start SurrealDB: surreal start"
	@echo "    3. Run migrations: make migrate-up"
	@echo "    4. Start dev server: make dev"
	@echo ""

## quick: Quick start for development (just deps and run)
quick: deps dev

# =============================================================================
# DEVELOPMENT
# =============================================================================

## run: Build and run the server
run: build
	$(BUILD_DIR)/$(BINARY)

## dev: Run with hot reload (Air)
dev:
	@if command -v air > /dev/null; then \
		air; \
	else \
		echo "Air not installed. Installing..."; \
		go install github.com/air-verse/air@latest; \
		air; \
	fi

## deps: Download and tidy dependencies
deps:
	@echo "==> Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy
	@echo "==> Dependencies ready"

## watch: Alias for dev (hot reload)
watch: dev

# =============================================================================
# CODE QUALITY
# =============================================================================

## test: Run tests with coverage
test:
	@echo "==> Running tests..."
	$(GOTEST) -v -race -cover ./...

## test-short: Run tests without race detector (faster)
test-short:
	$(GOTEST) -v -cover ./...

## lint: Run golangci-lint
lint:
	@echo "==> Running linter..."
	@if command -v $(GOLINT) > /dev/null; then \
		$(GOLINT) run ./...; \
	else \
		echo "golangci-lint not installed. Run 'make tools' first."; \
		exit 1; \
	fi

## lint-fix: Run linter with auto-fix
lint-fix:
	$(GOLINT) run --fix ./...

## fmt: Format code with gofumpt
fmt:
	@echo "==> Formatting code..."
	@if command -v $(GOFMT) > /dev/null; then \
		$(GOFMT) -l -w .; \
	else \
		echo "gofumpt not installed. Run 'make tools' first."; \
		$(GOCMD) fmt ./...; \
	fi
	@echo "==> Format complete"

## vet: Run go vet
vet:
	@echo "==> Running go vet..."
	$(GOCMD) vet ./...
	@echo "==> Vet complete"

## check: Run all checks (fmt, vet, lint, test)
check: fmt vet lint test
	@echo "==> All checks passed!"

# =============================================================================
# DATABASE
# =============================================================================

## migrate-status: Check migration status
migrate-status: build
	$(BUILD_DIR)/$(MIGRATE_BINARY) -cmd status

## migrate-up: Run pending migrations
migrate-up: build
	$(BUILD_DIR)/$(MIGRATE_BINARY) -cmd up

## migrate-create: Create new migration (usage: make migrate-create name=add_tags)
migrate-create: build
	@if [ -z "$(name)" ]; then \
		echo "Error: name is required. Usage: make migrate-create name=my_migration"; \
		exit 1; \
	fi
	$(BUILD_DIR)/$(MIGRATE_BINARY) -cmd create $(name)

## migrate-reset: Reset database (DESTRUCTIVE!)
migrate-reset: build
	@echo "WARNING: This will delete all data!"
	@read -p "Are you sure? [y/N] " confirm; \
	if [ "$$confirm" = "y" ] || [ "$$confirm" = "Y" ]; then \
		$(BUILD_DIR)/$(MIGRATE_BINARY) -cmd reset; \
	else \
		echo "Cancelled"; \
	fi

## db-start: Start SurrealDB in memory (development)
db-start:
	@echo "==> Starting SurrealDB..."
	surreal start --log info --user root --pass root memory

## db-start-file: Start SurrealDB with file storage
db-start-file:
	@echo "==> Starting SurrealDB with file storage..."
	surreal start --log info --user root --pass root file:./data/surreal.db

# =============================================================================
# DOCUMENTATION
# =============================================================================

## docs: Generate Swagger documentation
docs:
	@echo "==> Generating API documentation..."
	@if command -v swag > /dev/null; then \
		swag init -g cmd/api/main.go -o docs; \
	else \
		echo "swag not installed. Run 'make tools' first."; \
		exit 1; \
	fi
	@echo "==> Documentation generated in docs/"

# =============================================================================
# TOOLS
# =============================================================================

## tools: Install development tools
tools:
	@echo "==> Installing development tools..."
	go install github.com/air-verse/air@latest
	go install github.com/swaggo/swag/cmd/swag@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install mvdan.cc/gofumpt@latest
	@echo "==> Tools installed:"
	@echo "    - air (hot reload)"
	@echo "    - swag (OpenAPI docs)"
	@echo "    - golangci-lint (linting)"
	@echo "    - gofumpt (formatting)"

## tools-check: Check if tools are installed
tools-check:
	@echo "Checking tools..."
	@command -v air > /dev/null && echo "✓ air" || echo "✗ air (run 'make tools')"
	@command -v swag > /dev/null && echo "✓ swag" || echo "✗ swag (run 'make tools')"
	@command -v golangci-lint > /dev/null && echo "✓ golangci-lint" || echo "✗ golangci-lint (run 'make tools')"
	@command -v gofumpt > /dev/null && echo "✓ gofumpt" || echo "✗ gofumpt (run 'make tools')"
	@command -v surreal > /dev/null && echo "✓ surreal" || echo "✗ surreal (install from surrealdb.com)"

# =============================================================================
# FEATURE SCAFFOLDING
# =============================================================================

## new: Create a new feature (usage: make new name=comments)
new:
	@if [ -z "$(name)" ]; then \
		echo "Error: name is required."; \
		echo "Usage: make new name=comments"; \
		echo ""; \
		echo "This will create:"; \
		echo "  internal/features/<name>/"; \
		echo "    ├── models.go     (domain types)"; \
		echo "    ├── repository.go (data access with SurrealDB SDK)"; \
		echo "    ├── service.go    (business logic)"; \
		echo "    └── handler.go    (HTTP handlers)"; \
		exit 1; \
	fi
	@echo "==> Creating feature: $(name)"
	@$(GOCMD) run cmd/gen/main.go $(name)

## feature: Alias for 'new' (usage: make feature name=comments)
feature: new

# =============================================================================
# DOCKER
# =============================================================================

## docker-build: Build Docker image
docker-build:
	docker build -t daily-journal-go-backend .

## docker-run: Run Docker container
docker-run:
	docker run -p 8080:8080 --env-file .env daily-journal-go-backend

## docker-dev: Run with docker-compose (includes SurrealDB)
docker-dev:
	docker-compose up -d

# Default target
.DEFAULT_GOAL := help
