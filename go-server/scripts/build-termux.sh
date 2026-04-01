#!/usr/bin/env bash
# build-termux.sh — Cross-compile a static binary for Android ARM64 (Termux)
# Run this on any Linux/macOS machine with Go installed.
set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR/.."

echo "Building UI (React)..."
cd ..
pnpm --filter @nanoclip/ui build 2>/dev/null || echo "[warn] UI build skipped (no pnpm)"
cd go-server

echo "Cross-compiling Go server for Android ARM64..."
CGO_ENABLED=0 \
GOOS=linux \
GOARCH=arm64 \
go build \
  -ldflags="-s -w" \
  -trimpath \
  -o nanoclip-arm64 \
  .

echo ""
echo "Binary: go-server/nanoclip-arm64"
echo "Size:   $(du -sh nanoclip-arm64 | cut -f1)"
echo ""
echo "=== Deploy to Termux ==="
echo "1. Copy nanoclip-arm64 to your phone (adb push or scp)"
echo "2. On Termux:"
echo "   pkg install mariadb"
echo "   chmod +x nanoclip-arm64"
echo "   bash scripts/setup-mariadb.sh"
echo "   GO_PORT=8080 ./nanoclip-arm64"
echo ""
echo "=== (Optional) Copy UI dist ==="
echo "   cp -r ../ui/dist ui-dist"
echo "   # The server auto-detects and serves ui-dist/"
