#!/bin/bash
set -e

echo "=== Tengine Proxy Manager Setup ==="

# Create .env if it doesn't exist
if [ ! -f .env ]; then
    cp .env.example .env
    echo "[INFO] .env file created from .env.example."
    echo "[WARN] Don't forget to update the passwords in .env!"
fi

# Start with Docker Compose
echo "[INFO] Starting services..."
docker compose up --build -d

echo "[INFO] Waiting for services to be ready..."
sleep 5

echo "=== Setup complete ==="
echo "  Backend:  http://localhost:4000"
echo "  Frontend: http://localhost:3000"
echo "  Tengine:  http://localhost:80"
echo "  Health:   http://localhost:8888/upstream_check"
