#!/usr/bin/env bash
# build-termux.sh — Build a self-contained production binary for Android ARM64 (Termux)
#
# The binary embeds the entire React UI so only ONE file is needed on the device.
#
# Usage:
#   bash go-server/scripts/build-termux.sh              # default: linux/arm64
#   GOARCH=amd64 bash go-server/scripts/build-termux.sh # linux/amd64 (for testing)
#
# Requirements: Go 1.21+, pnpm (for UI build)
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
GO_SERVER="$REPO_ROOT/go-server"

TARGET_OS="${GOOS:-linux}"
TARGET_ARCH="${GOARCH:-arm64}"
OUT_NAME="nanoclip-${TARGET_ARCH}"
[ "$TARGET_OS" != "linux" ] && OUT_NAME="nanoclip-${TARGET_OS}-${TARGET_ARCH}"

echo "╔══════════════════════════════════════════════════╗"
echo "║         NanoClip Production Build               ║"
echo "║   target: ${TARGET_OS}/${TARGET_ARCH}                        ║"
echo "╚══════════════════════════════════════════════════╝"
echo ""

# ── Step 1: Build React UI ──────────────────────────────────────────────────
echo "▶ [1/3] Building React frontend..."
cd "$REPO_ROOT"

if command -v pnpm &>/dev/null; then
    pnpm --filter @nanoclip/ui build
else
    echo "    [warn] pnpm not found — skipping UI build (using existing dist if any)"
fi

UI_DIST="$REPO_ROOT/ui/dist"
if [ ! -f "$UI_DIST/index.html" ]; then
    echo "    [error] UI dist not found at $UI_DIST/index.html"
    echo "    Run: pnpm --filter @nanoclip/ui build"
    exit 1
fi
echo "    UI built: $(du -sh "$UI_DIST" | cut -f1)"

# ── Step 2: Stage UI dist for embedding ────────────────────────────────────
echo "▶ [2/3] Staging UI for embedding..."
EMBED_DIR="$GO_SERVER/ui-dist"
rm -rf "$EMBED_DIR"
cp -r "$UI_DIST" "$EMBED_DIR"
# Strip source maps — saves ~30% binary size, not needed in production
find "$EMBED_DIR" -name "*.map" -delete
echo "    Staged: $(du -sh "$EMBED_DIR" | cut -f1) (source maps stripped)"

# ── Step 3: Cross-compile Go binary with embedded UI ───────────────────────
echo "▶ [3/3] Cross-compiling Go (${TARGET_OS}/${TARGET_ARCH}, CGO_ENABLED=0)..."
cd "$GO_SERVER"

BUILD_VERSION="$(git -C "$REPO_ROOT" describe --tags --always 2>/dev/null || echo dev)"

GOTOOLCHAIN=local \
CGO_ENABLED=0 \
GOOS="$TARGET_OS" \
GOARCH="$TARGET_ARCH" \
go build \
    -tags prod \
    -ldflags="-s -w -X main.buildVersion=${BUILD_VERSION}" \
    -trimpath \
    -o "$OUT_NAME" \
    .

echo ""
echo "╔══════════════════════════════════════════════════╗"
echo "║  ✓ Build complete                               ║"
echo "╚══════════════════════════════════════════════════╝"
echo ""
echo "  Binary : go-server/$OUT_NAME"
echo "  Size   : $(du -sh "$GO_SERVER/$OUT_NAME" | cut -f1)"
echo "  Target : ${TARGET_OS}/${TARGET_ARCH}"
echo "  UI     : embedded ($(du -sh "$EMBED_DIR" | cut -f1))"
echo "  Version: $BUILD_VERSION"
echo ""
echo "═══ Deploy to Termux (Android) ═══════════════════════"
echo ""
echo "  1. Copy binary to your phone:"
echo "     adb push go-server/$OUT_NAME /data/local/tmp/"
echo "     # or: scp go-server/$OUT_NAME user@phone:~/"
echo ""
echo "  2. On Termux:"
echo "     pkg install mariadb   # optional — skip for SQLite"
echo "     chmod +x $OUT_NAME"
echo "     ./$OUT_NAME            # SQLite mode (zero-config)"
echo ""
echo "     # MariaDB mode:"
echo "     DSN=\"user:pass@tcp(127.0.0.1:3306)/nanoclip?parseTime=true\" ./$OUT_NAME"
echo ""
echo "  3. Open browser: http://localhost:8080"
echo ""
echo "═══ Low-memory tips (Termux) ═════════════════════════"
echo ""
echo "  GOMEMLIMIT=256MiB ./$OUT_NAME     # cap Go heap"
echo "  GOGC=50 ./$OUT_NAME                # more frequent GC"
echo "  SQLite mode uses ~30 MB RAM at idle"
echo ""

# ── Restore placeholder so git stays clean ─────────────────────────────────
rm -rf "$EMBED_DIR"
mkdir -p "$EMBED_DIR"
echo "# ui-dist placeholder — filled by build-termux.sh before go build -tags prod" \
    > "$EMBED_DIR/.gitkeep"
echo "  Restored ui-dist/.gitkeep for git."
