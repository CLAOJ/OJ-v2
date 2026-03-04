#!/bin/bash

# CLAOJ v2 Quickstart Script
# This script helps you get started with CLAOJ v2

set -e

echo "========================================"
echo "  CLAOJ v2 - Quickstart"
echo "========================================"
echo ""

# Check for Docker
if ! command -v docker &> /dev/null; then
    echo "Error: Docker is not installed. Please install Docker first."
    exit 1
fi

if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null; then
    echo "Error: Docker Compose is not installed. Please install Docker Compose first."
    exit 1
fi

echo "✓ Docker is installed"
echo ""

# Create necessary directories
mkdir -p nginx/ssl
mkdir -p claoj-go/data

# Copy example config if not exists
if [ ! -f claoj-go/claoj.yaml ]; then
    echo "Creating claoj-go/claoj.yaml from example..."
    cp claoj-go/claoj.example.yaml claoj-go/claoj.yaml
    echo "✓ Created claoj-go/claoj.yaml"
    echo "  Please edit this file to configure your database and other settings."
    echo ""
fi

# Copy example env if not exists
if [ ! -f claoj-web/.env.local ]; then
    echo "Creating claoj-web/.env.local from example..."
    cp claoj-web/.env.example claoj-web/.env.local
    echo "✓ Created claoj-web/.env.local"
    echo ""
fi

# Ask user for deployment type
echo "Select deployment type:"
echo "1. Docker Compose (Recommended - Full stack with DB and Redis)"
echo "2. Frontend only (Connect to existing backend)"
echo "3. Backend only (Connect to existing frontend)"
echo ""
read -p "Enter choice (1-3): " deployment_type

case $deployment_type in
    1)
        echo ""
        echo "Starting full stack with Docker Compose..."
        echo "Note: If you get permission errors, run with: sudo docker compose up -d"
        echo "Or add your user to the docker group: sudo usermod -aG docker $USER"
        echo ""
        docker compose up -d
        echo ""
        echo "✓ Services started!"
        echo ""
        echo "Access points:"
        echo "  - Frontend: http://localhost:3000"
        echo "  - Backend API: http://localhost:8080"
        echo "  - MySQL: localhost:3306"
        echo "  - Redis: localhost:6379"
        echo ""
        echo "View logs: docker compose logs -f"
        echo "Stop services: docker compose down"
        ;;
    2)
        echo ""
        echo "Installing and starting frontend only..."
        cd claoj-web
        npm install
        npm run build
        npm start &
        echo ""
        echo "✓ Frontend started!"
        echo ""
        echo "Access points:"
        echo "  - Frontend: http://localhost:3000"
        echo ""
        echo "Stop frontend: pkill -f 'next-start'"
        ;;
    3)
        echo ""
        echo "Starting backend only..."
        cd claoj-go
        if ! command -v go &> /dev/null; then
            echo "Error: Go is not installed. Please install Go 1.24+ first."
            exit 1
        fi
        go mod download
        go run main.go &
        echo ""
        echo "✓ Backend started!"
        echo ""
        echo "Access points:"
        echo "  - Backend API: http://localhost:8080"
        echo ""
        echo "Stop backend: pkill -f 'claoj-go'"
        ;;
    *)
        echo "Invalid choice. Exiting."
        exit 1
        ;;
esac

echo ""
echo "========================================"
echo "  Setup Complete!"
echo "========================================"
echo ""
echo "Next steps:"
echo "1. Edit claoj-go/claoj.yaml to configure your settings"
echo "2. Set up your database (MySQL 8.0+)"
echo "3. Set up Redis for caching"
echo "4. Run migrations (if applicable)"
echo "5. Create an admin user"
echo ""
echo "For more information, see README.md"
echo ""
