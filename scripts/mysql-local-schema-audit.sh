#!/usr/bin/env bash
set -euo pipefail

MYSQL_HOST="${MYSQL_HOST:-127.0.0.1}"
MYSQL_PORT="${MYSQL_PORT:-3306}"
MYSQL_USER="${MYSQL_USER:-root}"
MYSQL_PASSWORD="${MYSQL_PASSWORD:-${MYSQL_PWD:-}}"
MYSQL_CANONICAL_LOCAL_DATABASE="${MYSQL_CANONICAL_LOCAL_DATABASE:-ai_troubleshooter}"
DROP_LOCAL_SCHEMA_SPRAWL="${DROP_LOCAL_SCHEMA_SPRAWL:-false}"
CONFIRM_DROP_LOCAL_SCHEMA_SPRAWL="${CONFIRM_DROP_LOCAL_SCHEMA_SPRAWL:-}"

ARGS=(-h "$MYSQL_HOST" -P "$MYSQL_PORT" -u "$MYSQL_USER" --default-character-set=utf8mb4)

run_mysql() {
  if [[ -n "$MYSQL_PASSWORD" ]]; then
    MYSQL_PWD="$MYSQL_PASSWORD" mysql "${ARGS[@]}" "$@"
  else
    mysql "${ARGS[@]}" "$@"
  fi
}

truthy() {
  case "$(printf '%s' "$1" | tr '[:upper:]' '[:lower:]')" in
    1|true|yes|y|on) return 0 ;;
    *) return 1 ;;
  esac
}

if [[ ! "$MYSQL_CANONICAL_LOCAL_DATABASE" =~ ^[A-Za-z0-9_]+$ ]]; then
  echo "MYSQL_CANONICAL_LOCAL_DATABASE must contain only letters, digits, and underscores: $MYSQL_CANONICAL_LOCAL_DATABASE" >&2
  exit 2
fi

query="
SELECT SCHEMA_NAME
FROM information_schema.SCHEMATA
WHERE (
  SCHEMA_NAME LIKE 'ai\\_troubleshooter\\_%'
  OR SCHEMA_NAME LIKE 'hf\\_troubleshoot\\_%'
)
AND SCHEMA_NAME <> '${MYSQL_CANONICAL_LOCAL_DATABASE}'
ORDER BY SCHEMA_NAME;
"

schemas=()
while IFS= read -r schema; do
  if [[ -n "$schema" ]]; then
    schemas+=("$schema")
  fi
done < <(run_mysql --batch --skip-column-names -e "$query")

if [[ "${#schemas[@]}" -eq 0 ]]; then
  echo "No local troubleshooting schema sprawl detected."
  exit 0
fi

for schema in "${schemas[@]}"; do
  if [[ ! "$schema" =~ ^[A-Za-z0-9_]+$ ]]; then
    echo "Refusing suspicious schema name returned by MySQL: $schema" >&2
    exit 2
  fi
done

echo "Detected non-canonical local troubleshooting schemas:"
for schema in "${schemas[@]}"; do
  echo "  - $schema"
done

echo
echo "Suggested cleanup SQL:"
for schema in "${schemas[@]}"; do
  printf 'DROP DATABASE `%s`;\n' "$schema"
done

if ! truthy "$DROP_LOCAL_SCHEMA_SPRAWL"; then
  echo
  echo "No schema was dropped. To drop these local schemas, rerun with:"
  echo "  DROP_LOCAL_SCHEMA_SPRAWL=true CONFIRM_DROP_LOCAL_SCHEMA_SPRAWL=yes scripts/mysql-local-schema-audit.sh"
  exit 0
fi

if [[ "$CONFIRM_DROP_LOCAL_SCHEMA_SPRAWL" != "yes" ]]; then
  echo "Refusing to drop schemas without CONFIRM_DROP_LOCAL_SCHEMA_SPRAWL=yes" >&2
  exit 2
fi

for schema in "${schemas[@]}"; do
  run_mysql -e "DROP DATABASE \`$schema\`;"
  echo "dropped $schema"
done
