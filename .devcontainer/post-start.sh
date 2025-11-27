#!/bin/bash
# =============================================================================
# Post-Start Script
# =============================================================================
# Runs every time the dev container starts.
# Use this for things that need to happen on each start.
# =============================================================================

set -e

echo "🔄 Dev container starting..."

# Wait for SurrealDB to be ready
echo "⏳ Waiting for SurrealDB..."
max_attempts=30
attempt=1
while ! curl -s http://surrealdb:8000/health > /dev/null 2>&1; do
    if [ $attempt -ge $max_attempts ]; then
        echo "❌ SurrealDB failed to start after ${max_attempts} attempts"
        exit 1
    fi
    echo "  Attempt $attempt/$max_attempts..."
    sleep 1
    ((attempt++))
done
echo "✅ SurrealDB is ready!"

# Show helpful info
echo ""
echo "📋 Quick Reference:"
echo "  task rust:dev    - Start server with hot reload"
echo "  task rust:test   - Run tests"
echo "  task rust:lint   - Run clippy + fmt check"
echo "  task rust:help   - Show all commands"
echo ""
echo "📁 You're in: $(pwd)"
echo "🌐 API: http://localhost:8080"
echo "📚 Docs: http://localhost:8080/docs"

