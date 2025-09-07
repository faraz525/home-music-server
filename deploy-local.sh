#!/bin/bash

# Local development deployment script
echo "🚀 Deploying CrateDrop locally..."

# Create data directory if it doesn't exist
mkdir -p ./data/cratedrop

# Build and start services
docker compose up -d --build

# Wait a moment for services to start
sleep 3

# Check status
echo "📊 Service Status:"
docker compose ps

# Show logs hint
echo ""
echo "📋 To view logs: docker compose logs -f"
echo "🌐 Frontend: http://localhost"
echo "🔧 Backend API: http://localhost:8080"

# Get admin credentials if available
echo ""
echo "🔑 Admin Credentials (check logs):"
docker compose logs backend | grep -i "admin\|created" | tail -5
