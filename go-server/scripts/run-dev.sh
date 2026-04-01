#!/usr/bin/env bash
# run-dev.sh — Rebuild and run the Go server (used by the Replit workflow)
set -e
cd "$(dirname "${BASH_SOURCE[0]}")/.."

echo "=== Building NanoClip ==="
go build -o nanoclip . 2>&1
echo "=== Starting NanoClip (port ${GO_PORT:-8080}) ==="
exec ./nanoclip
