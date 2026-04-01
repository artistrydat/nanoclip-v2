#!/usr/bin/env bash
# setup-mariadb.sh — Initialize and start MariaDB for Paperclip Go
# Works on Replit (NixOS) and Termux (Android)
set -e

DATADIR="${PAPERCLIP_DB_DIR:-$HOME/.paperclip-go/mariadb}"
SOCKET="${PAPERCLIP_DB_SOCKET:-$HOME/.paperclip-go/mariadb.sock}"
PORT="${MARIADB_PORT:-3306}"
DB_USER="${MARIADB_USER:-paperclip}"
DB_PASS="${MARIADB_PASS:-paperclip}"
DB_NAME="${MARIADB_DB:-paperclip}"

MYSQL_BIN=$(which mariadbd 2>/dev/null || which mysqld 2>/dev/null || which mariadbd-safe 2>/dev/null || echo "")
MYSQL_INSTALL=$(which mariadb-install-db 2>/dev/null || which mysql_install_db 2>/dev/null || echo "")
MYSQL_CLI=$(which mariadb 2>/dev/null || which mysql 2>/dev/null || echo "")

if [ -z "$MYSQL_BIN" ]; then
  echo "[setup-mariadb] ERROR: MariaDB/MySQL server binary not found."
  echo "  On Termux: pkg install mariadb"
  echo "  On Nix:    nix-env -iA nixpkgs.mariadb"
  exit 1
fi

echo "[setup-mariadb] Using MariaDB binary: $MYSQL_BIN"
echo "[setup-mariadb] Data directory: $DATADIR"

# Initialize data directory if not already done
if [ ! -d "$DATADIR/mysql" ]; then
  echo "[setup-mariadb] Initializing data directory..."
  mkdir -p "$DATADIR"
  if [ -n "$MYSQL_INSTALL" ]; then
    "$MYSQL_INSTALL" --datadir="$DATADIR" --skip-test-db 2>&1 | tail -5
  else
    "$MYSQL_BIN" --initialize-insecure --datadir="$DATADIR" 2>&1 | tail -5
  fi
  echo "[setup-mariadb] Data directory initialized."
fi

# Check if already running
if [ -f "$SOCKET" ]; then
  echo "[setup-mariadb] MariaDB appears to already be running (socket found)."
else
  echo "[setup-mariadb] Starting MariaDB server..."
  "$MYSQL_BIN" \
    --datadir="$DATADIR" \
    --socket="$SOCKET" \
    --port="$PORT" \
    --pid-file="$DATADIR/mariadb.pid" \
    --log-error="$DATADIR/mariadb.err" \
    --bind-address=127.0.0.1 \
    --skip-networking=OFF \
    --user="$(whoami)" \
    --daemonize 2>/dev/null || \
  "$MYSQL_BIN" \
    --datadir="$DATADIR" \
    --socket="$SOCKET" \
    --port="$PORT" \
    --pid-file="$DATADIR/mariadb.pid" \
    --log-error="$DATADIR/mariadb.err" \
    --bind-address=127.0.0.1 \
    --user="$(whoami)" &

  echo "[setup-mariadb] Waiting for server to start..."
  for i in $(seq 1 20); do
    if "$MYSQL_CLI" --socket="$SOCKET" -u root -e "SELECT 1" >/dev/null 2>&1; then
      break
    fi
    sleep 1
  done
fi

# Create database and user
echo "[setup-mariadb] Creating database and user..."
"$MYSQL_CLI" --socket="$SOCKET" -u root 2>/dev/null <<EOF || \
"$MYSQL_CLI" -h 127.0.0.1 -P "$PORT" -u root 2>/dev/null <<EOF
CREATE DATABASE IF NOT EXISTS \`${DB_NAME}\` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE USER IF NOT EXISTS '${DB_USER}'@'localhost' IDENTIFIED BY '${DB_PASS}';
CREATE USER IF NOT EXISTS '${DB_USER}'@'127.0.0.1' IDENTIFIED BY '${DB_PASS}';
GRANT ALL PRIVILEGES ON \`${DB_NAME}\`.* TO '${DB_USER}'@'localhost';
GRANT ALL PRIVILEGES ON \`${DB_NAME}\`.* TO '${DB_USER}'@'127.0.0.1';
FLUSH PRIVILEGES;
EOF

echo "[setup-mariadb] Done! Database '${DB_NAME}' ready on port ${PORT}."
echo ""
echo "Add to your .env:"
echo "  MARIADB_DSN=${DB_USER}:${DB_PASS}@tcp(127.0.0.1:${PORT})/${DB_NAME}?charset=utf8mb4&parseTime=True&loc=UTC"
