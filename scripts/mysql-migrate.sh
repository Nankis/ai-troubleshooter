#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

MYSQL_HOST="${MYSQL_HOST:-127.0.0.1}"
MYSQL_PORT="${MYSQL_PORT:-3306}"
MYSQL_USER="${MYSQL_USER:-root}"
MYSQL_DATABASE="${MYSQL_DATABASE:-ai_troubleshooter}"
MYSQL_PASSWORD="${MYSQL_PASSWORD:-${MYSQL_PWD:-}}"

if ! command -v mysql >/dev/null 2>&1; then
  echo "mysql client is required" >&2
  exit 1
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
