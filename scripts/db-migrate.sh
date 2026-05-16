#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
MIGRATIONS_DIR="${REPO_ROOT}/backend/migrations"
CONTAINER_NAME="aethernet-postgres"
DB_USER="aether"
DB_NAME="aethernet"

if [[ ! -d "${MIGRATIONS_DIR}" ]]; then
  echo "Migrations directory not found: ${MIGRATIONS_DIR}" >&2
  exit 1
fi

if ! docker ps --format '{{.Names}}' | grep -Fxq "${CONTAINER_NAME}"; then
  echo "Postgres container '${CONTAINER_NAME}' is not running. Start it first with: pnpm db:up" >&2
  exit 1
fi

invoke_psql() {
  local sql="$1"
  printf '%s' "${sql}" | docker exec -i "${CONTAINER_NAME}" psql -v ON_ERROR_STOP=1 -U "${DB_USER}" -d "${DB_NAME}"
}

invoke_psql_quiet() {
  local sql="$1"
  printf '%s' "${sql}" | docker exec -i "${CONTAINER_NAME}" psql -q -v ON_ERROR_STOP=1 -U "${DB_USER}" -d "${DB_NAME}"
}

invoke_psql "CREATE TABLE IF NOT EXISTS schema_migrations (
  version BIGINT PRIMARY KEY,
  dirty BOOLEAN NOT NULL DEFAULT FALSE,
  applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
);"

declare -A APPLIED_VERSIONS=()
while IFS= read -r version; do
  version="$(echo "${version}" | xargs)"
  if [[ -n "${version}" ]]; then
    APPLIED_VERSIONS["${version}"]=1
  fi
done < <(docker exec -i "${CONTAINER_NAME}" psql -t -A -U "${DB_USER}" -d "${DB_NAME}" -c "SELECT version FROM schema_migrations ORDER BY version;")

shopt -s nullglob
MIGRATION_FILES=("${MIGRATIONS_DIR}"/*.up.sql)
if [[ ${#MIGRATION_FILES[@]} -eq 0 ]]; then
  echo "No up migrations found in ${MIGRATIONS_DIR}"
  exit 0
fi

IFS=$'\n' MIGRATION_FILES=($(printf '%s\n' "${MIGRATION_FILES[@]}" | sort))
unset IFS

for file in "${MIGRATION_FILES[@]}"; do
  filename="$(basename "${file}")"
  version="${filename%%_*}"
  if [[ ! "${version}" =~ ^[0-9]+$ ]]; then
    echo "Migration file name must start with a numeric version: ${filename}" >&2
    exit 1
  fi

  if [[ -n "${APPLIED_VERSIONS[${version}]:-}" ]]; then
    echo "Skipping migration ${version} (${filename})"
    continue
  fi

  echo "Applying migration ${version} (${filename})"
  body="$(cat "${file}")"
  sql=$(cat <<SQL
BEGIN;
${body}
INSERT INTO schema_migrations(version, dirty) VALUES (${version}, FALSE);
COMMIT;
SQL
)
  invoke_psql_quiet "${sql}"
done

echo "Database migrations complete."
