#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

MYSQL_HOST="${MYSQL_HOST:-127.0.0.1}"
MYSQL_PORT="${MYSQL_PORT:-3306}"
MYSQL_USER="${MYSQL_USER:-root}"
MYSQL_DATABASE="${MYSQL_DATABASE:-ai_troubleshooter}"
MYSQL_PASSWORD="${MYSQL_PASSWORD:-${MYSQL_PWD:-}}"
MYSQL_CANONICAL_LOCAL_DATABASE="${MYSQL_CANONICAL_LOCAL_DATABASE:-ai_troubleshooter}"
ALLOW_NON_CANONICAL_LOCAL_DB="${ALLOW_NON_CANONICAL_LOCAL_DB:-false}"

is_truthy() {
  case "$(printf '%s' "$1" | tr '[:upper:]' '[:lower:]')" in
    1|true|yes|y|on) return 0 ;;
    *) return 1 ;;
  esac
}

is_local_mysql_host() {
  case "$MYSQL_HOST" in
    127.0.0.1|localhost|::1) return 0 ;;
    *) return 1 ;;
  esac
}

if ! command -v mysql >/dev/null 2>&1; then
  echo "mysql client is required" >&2
  exit 1
fi

if [[ ! "$MYSQL_DATABASE" =~ ^[A-Za-z0-9_]+$ ]]; then
  echo "MYSQL_DATABASE must contain only letters, digits, and underscores: $MYSQL_DATABASE" >&2
  exit 2
fi

if [[ ! "$MYSQL_CANONICAL_LOCAL_DATABASE" =~ ^[A-Za-z0-9_]+$ ]]; then
  echo "MYSQL_CANONICAL_LOCAL_DATABASE must contain only letters, digits, and underscores: $MYSQL_CANONICAL_LOCAL_DATABASE" >&2
  exit 2
fi

if is_local_mysql_host && [[ "$MYSQL_DATABASE" != "$MYSQL_CANONICAL_LOCAL_DATABASE" ]] && ! is_truthy "$ALLOW_NON_CANONICAL_LOCAL_DB"; then
  cat >&2 <<EOF
Refusing to migrate local MySQL database '$MYSQL_DATABASE'.
Use the canonical local platform schema '$MYSQL_CANONICAL_LOCAL_DATABASE'.
Only set ALLOW_NON_CANONICAL_LOCAL_DB=true for an intentional isolated experiment,
and record the cleanup plan in the current Program.
EOF
  exit 2
fi

ARGS=(-h "$MYSQL_HOST" -P "$MYSQL_PORT" -u "$MYSQL_USER" --default-character-set=utf8mb4)

run_mysql() {
  if [[ -n "$MYSQL_PASSWORD" ]]; then
    MYSQL_PWD="$MYSQL_PASSWORD" mysql "${ARGS[@]}" "$@"
  else
    mysql "${ARGS[@]}" "$@"
  fi
}

run_mysql -e "CREATE DATABASE IF NOT EXISTS \`$MYSQL_DATABASE\` DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;"
run_mysql "$MYSQL_DATABASE" -e "CREATE TABLE IF NOT EXISTS schema_migrations (version VARCHAR(128) PRIMARY KEY, applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP);"

for migration in "$ROOT_DIR"/migrations/*.sql; do
  version="$(basename "$migration")"
  if [[ ! "$version" =~ ^[0-9]{3}_[A-Za-z0-9_]+\.sql$ ]]; then
    echo "Refusing suspicious migration filename: $version" >&2
    exit 2
  fi
  applied="$(run_mysql "$MYSQL_DATABASE" --batch --skip-column-names -e "SELECT COUNT(1) FROM schema_migrations WHERE version = '$version';")"
  if [[ "$applied" == "1" ]]; then
    echo "skip $version"
    continue
  fi
  echo "apply $version"
  run_mysql "$MYSQL_DATABASE" < "$migration"
  run_mysql "$MYSQL_DATABASE" -e "INSERT INTO schema_migrations(version) VALUES ('$version');"
done

echo "migrations applied to $MYSQL_DATABASE"
