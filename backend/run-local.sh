#!/bin/bash
# Helper script to run the backend locally with the correct data directory
# This ensures we use the same data location as docker-compose.override.yml

cd "$(dirname "$0")"

# Kill any existing process on port 8080 to avoid "address already in use" errors
echo "🔍 Checking for processes on port 8080..."
if lsof -ti:8080 > /dev/null 2>&1; then
    echo "⚠️  Killing existing process on port 8080..."
    lsof -ti:8080 | xargs kill -9 2>/dev/null || true
    sleep 1
    echo "✅ Port 8080 is now free"
else
    echo "✅ Port 8080 is already free"
fi

# Use Homebrew SQLite which has FTS5 support (macOS fix)
if [[ "$OSTYPE" == "darwin"* ]]; then
    # Check if Homebrew SQLite exists
    if [ -d "/opt/homebrew/opt/sqlite" ]; then
        export CGO_ENABLED=1
        export CGO_LDFLAGS="-L/opt/homebrew/opt/sqlite/lib"
        export CGO_CFLAGS="-I/opt/homebrew/opt/sqlite/include"
        echo "🔧 Using Homebrew SQLite (FTS5 enabled)"
    elif [ -d "/usr/local/opt/sqlite" ]; then
        export CGO_ENABLED=1
        export CGO_LDFLAGS="-L/usr/local/opt/sqlite/lib"
        export CGO_CFLAGS="-I/usr/local/opt/sqlite/include"
        echo "🔧 Using Homebrew SQLite (FTS5 enabled)"
    fi
fi

# Force rebuild with external SQLite library (not embedded)
echo "🔨 Building with external Homebrew SQLite (FTS5 enabled)..."
go build -a -tags="libsqlite3" -o /tmp/cratedrop-local-server . || exit 1

echo "🚀 Starting backend..."
# Set environment variables for local development
export DATA_DIR=../data/cratedrop
export JWT_SECRET="${JWT_SECRET:-dev-jwt-secret-local-only}"
export REFRESH_SECRET="${REFRESH_SECRET:-dev-refresh-secret-local-only}"
export APP_ENV="${APP_ENV:-development}"
export GIN_MODE="${GIN_MODE:-debug}"

/tmp/cratedrop-local-server

