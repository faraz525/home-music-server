#!/bin/bash

# Production deployment script for Raspberry Pi
echo "🚀 Deploying CrateDrop to production (Raspberry Pi)..."

# Ensure production storage directory exists
sudo mkdir -p /mnt/music/cratedrop
sudo chown -R $USER:$USER /mnt/music/cratedrop

# Deploy without override file (uses production volumes)
docker compose up -d --build

# Wait for services to start
sleep 3

# Check status
echo "📊 Service Status:"
docker compose ps

# Show logs hint
echo ""
echo "📋 To view logs: docker compose logs -f"
echo "🌐 Frontend: http://your-pi-ip"
echo "🔧 Backend API: http://your-pi-ip:8080"

# Get admin credentials
echo ""
echo "🔑 Admin Credentials (check logs):"
docker compose logs backend | grep -i "admin\|created" | tail -5
