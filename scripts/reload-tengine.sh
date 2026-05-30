#!/bin/bash
set -e

CONTAINER_NAME="${1:-tengineproxymanager-tengine-1}"

echo "[INFO] Testing Tengine config..."
docker exec "$CONTAINER_NAME" tengine -t

echo "[INFO] Reloading Tengine..."
docker exec "$CONTAINER_NAME" tengine -s reload

echo "[OK] Tengine reloaded successfully."
