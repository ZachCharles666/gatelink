#!/bin/bash
set -e

source .env 2>/dev/null || true

MIGRATIONS_PATH="internal/db/migrations"
DB_URL="${DATABASE_URL}"

case "$1" in
  up)
    echo "Running migrations UP..."
    go run -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate \
      -path "${MIGRATIONS_PATH}" \
      -database "${DB_URL}" up
    ;;
  down)
    echo "Running migrations DOWN (1 step)..."
    go run -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate \
      -path "${MIGRATIONS_PATH}" \
      -database "${DB_URL}" down 1
    ;;
  status)
    go run -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate \
      -path "${MIGRATIONS_PATH}" \
      -database "${DB_URL}" version
    ;;
  *)
    echo "Usage: $0 {up|down|status}"
    exit 1
    ;;
esac
