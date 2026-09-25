#!/usr/bin/env bash

set -euo pipefail

export GOOSE_DRIVER="${GOOSE_DRIVER:-postgres}"
export GOOSE_DBSTRING="${GOOSE_DBSTRING:-postgres://kargo:kargo@127.0.0.1:15432/kargo?sslmode=disable}"
export GOOSE_MIGRATION_DIR="${GOOSE_MIGRATION_DIR:-db/migrations}"

if [[ ! -d "$GOOSE_MIGRATION_DIR" ]]; then
  echo "Migration directory does not exist: $GOOSE_MIGRATION_DIR" >&2
  exit 1
fi

# Pod readiness does not guarantee that Tilt's port forward is ready yet.
# Checking the version establishes a real database connection and initializes
# Goose's version table on a new database without applying any migrations.
connection_log=$(mktemp)
trap 'rm -f "$connection_log"' EXIT
for ((attempt = 1; attempt <= 30; attempt++)); do
  if go tool goose -timeout 2s version > "$connection_log" 2>&1; then
    break
  fi
  if ((attempt == 30)); then
    echo "Database did not become ready after 30 attempts:" >&2
    cat "$connection_log" >&2
    exit 1
  fi
  if ((attempt == 1)); then
    echo "Waiting for the development database..."
  fi
  sleep 1
done

# There is no application schema yet. Keep a fresh checkout usable until the
# first SQL migration is added; Goose reports an error for an empty directory.
shopt -s nullglob
migrations=("$GOOSE_MIGRATION_DIR"/*.sql)
if ((${#migrations[@]} == 0)); then
  echo "No SQL migrations in $GOOSE_MIGRATION_DIR"
  exit 0
fi

# Only retry connection readiness, never a failed migration.
go tool goose up
