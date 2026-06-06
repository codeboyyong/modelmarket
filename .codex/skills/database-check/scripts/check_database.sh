#!/usr/bin/env sh
set -eu

env_name="${1:-dev}"

load_env_file() {
  file_path="$1"
  if [ -f "$file_path" ]; then
    set -a
    # shellcheck disable=SC1090
    . "$file_path"
    set +a
  fi
}

load_env_file ".env"
load_env_file ".env.${env_name}"
load_env_file "deploy/env/${env_name}.env"
load_env_file "config/env/${env_name}.env"

MM_DB_DRIVER="${MM_DB_DRIVER:-postgres}"
MM_DB_HOST="${MM_DB_HOST:-localhost}"
MM_DB_PORT="${MM_DB_PORT:-5432}"
MM_DB_NAME="${MM_DB_NAME:-model_market_${env_name}}"
MM_DB_USER="${MM_DB_USER:-model_market}"
MM_DB_PASSWORD="${MM_DB_PASSWORD:-model_market}"

if [ -z "${MM_DB_SSL_MODE:-}" ]; then
  case "$env_name" in
    prod|production|qa|staging)
      MM_DB_SSL_MODE="require"
      ;;
    *)
      MM_DB_SSL_MODE="disable"
      ;;
  esac
fi

if [ "$MM_DB_DRIVER" != "postgres" ] && [ "$MM_DB_DRIVER" != "postgresql" ]; then
  echo "Unsupported MM_DB_DRIVER: ${MM_DB_DRIVER}. Phase 1 database check supports PostgreSQL via psql." >&2
  exit 1
fi

if ! command -v psql >/dev/null 2>&1; then
  echo "psql is required but was not found in PATH." >&2
  exit 1
fi

run_psql() {
  if [ -n "${MM_DATABASE_URL:-}" ]; then
    PGSSLMODE="$MM_DB_SSL_MODE" psql "$MM_DATABASE_URL" "$@"
  else
    PGPASSWORD="$MM_DB_PASSWORD" PGSSLMODE="$MM_DB_SSL_MODE" \
      psql \
        -h "$MM_DB_HOST" \
        -p "$MM_DB_PORT" \
        -U "$MM_DB_USER" \
        -d "$MM_DB_NAME" \
        "$@"
  fi
}

echo "Environment: ${env_name}"
if [ -n "${MM_DATABASE_URL:-}" ]; then
  echo "Database: MM_DATABASE_URL"
else
  echo "Database: ${MM_DB_USER}@${MM_DB_HOST}:${MM_DB_PORT}/${MM_DB_NAME} sslmode=${MM_DB_SSL_MODE}"
fi
echo

run_psql -v ON_ERROR_STOP=1 -X <<'SQL'
\pset pager off

\echo 'Counts'
select 'models' as metric, cast(count(*) as varchar) as value from models
union all
select 'tables' as metric, cast(count(*) as varchar) as value
from information_schema.tables
where table_schema = current_schema()
  and table_type = 'BASE TABLE'
union all
select 'users' as metric, cast(count(*) as varchar) as value from users
order by metric;

\echo ''
\echo 'Model pricing'
select
  p.slug as provider,
  m.slug as model,
  mp.slug as model_profile,
  pr.input_token_price,
  pr.input_token_price_unit,
  pr.output_token_price,
  pr.output_token_price_unit,
  pr.provider_input_token_cost,
  pr.provider_input_token_cost_unit,
  pr.provider_output_token_cost,
  pr.provider_output_token_cost_unit,
  pr.currency,
  pr.effective_at
from price_rules pr
left join models m on m.id = pr.model_id
left join providers p on p.id = m.provider_id
left join model_profiles mp on mp.id = pr.model_profile_id
order by p.slug, m.slug, mp.slug, pr.effective_at;
SQL
