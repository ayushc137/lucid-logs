# Daily Journal App

## Overview

A self-hostable, cross-platform daily journaling application.
Stack: **Go (Gin)** + **Svelte** + **SurrealDB**.

## Developer Setup

We use **standard industry tooling** to ensure a reproducible environment. You can choose between the **Dev Container** (Recommended) or **Manual Setup**.

### Option A: Dev Container (Recommended)
*Requires: Docker Desktop or Docker Engine*

This project is configured with a `.devcontainer`.
1. Open this project in **Cursor** (or VS Code).
2. When prompted "Reopen in Container", click **Reopen**.
   (Or run command: `> Dev Containers: Reopen in Container`)
3. Wait for initialization. The environment will auto-install Go, Air, Swag, and Task.
4. Run:
   ```bash
   task dev
   ```

### Option B: Manual Setup
*Requires: [Docker](https://docs.docker.com/engine/install/), [Go 1.23+](https://go.dev/dl/), and [Task](https://taskfile.dev/installation/)*

1. **Install Dependencies**:
   Ensure `docker` and `go` are in your PATH. Install `task`:
   ```bash
   # macOS / Linux (Homebrew)
   brew install go-task/tap/go-task
   
   # Arch Linux
   sudo pacman -S go-task
   
   # NPM
   npm install -g @go-task/cli
   ```

2. **Fix Docker Permissions (Linux Only)**:
   If you get "permission denied" errors:
   ```bash
   sudo usermod -aG docker $USER
   newgrp docker
   ```

3. **Initialize & Run**:
   ```bash
   task dev
   ```

## Development Workflow

- **Run App**: `task dev` (Starts DB + Backend with Hot Reload)
- **Docs**: `http://localhost:8080/swagger/index.html`
- **DB Admin**: `http://localhost:8001` (Surrealist)
- **Update Deps**: `task backend:mod`
- **Regenerate Docs**: `task backend:docs`

## Configuration

Configuration is managed via `.env`. Copy `env.example` to `.env` to customize ports.

| Variable | Default | Description |
|----------|---------|-------------|
| `HOST_APP_PORT` | `8080` | API Port |
| `HOST_DB_PORT` | `8000` | DB Port |
