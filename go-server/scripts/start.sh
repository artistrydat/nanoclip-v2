#!/usr/bin/env bash
# start.sh — Start MariaDB then the NanoClip server
set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "=== NanoClip ==="
echo "Starting MariaDB..."
bash "$SCRIPT_DIR/setup-mariadb.sh"

echo "Starting Go server..."
exec "$SCRIPT_DIR/../nanoclip" "$@"
