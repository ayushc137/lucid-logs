#!/bin/bash
# =============================================================================
# Post-Create Script
# =============================================================================
# Runs once after the dev container is created.
# Use this for one-time setup that doesn't need to run on every start.
# =============================================================================

set -e

echo "🚀 Setting up development environment..."

# Create .env if it doesn't exist
if [ ! -f ".env" ]; then
    echo "📝 Creating .env from env.example..."
    cp env.example .env
    # Override DB_HOST for container networking
    sed -i 's/DB_HOST=localhost/DB_HOST=surrealdb/' .env
fi

# Navigate to Rust backend
cd apps/rust_backend

# Install lefthook git hooks
if command -v lefthook &> /dev/null; then
    echo "🪝 Installing git hooks..."
    lefthook install
fi

# Build the project (downloads dependencies, compiles)
echo "🔨 Building project (this may take a few minutes on first run)..."
cargo build

# Run clippy to download its data
echo "📎 Running initial clippy check..."
cargo clippy --all-targets || true

echo ""
echo "✅ Development environment ready!"
echo ""
echo "Quick start:"
echo "  task rust:dev          # Start with hot reload"
echo "  task rust:test         # Run tests"
echo "  task rust:lint         # Check code quality"
echo "  task rust:help         # Show all commands"
echo ""
echo "The database is available at: surrealdb:8000"
echo "API will be available at: http://localhost:8080"
echo "API docs at: http://localhost:8080/docs"

