#!/bin/sh
set -e

echo "Waiting for database to be ready..."
max_attempts=30
attempt=0

while [ $attempt -lt $max_attempts ]; do
    if PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c "SELECT 1" > /dev/null 2>&1; then
        echo "Database is ready!"
        break
    fi
    attempt=$((attempt + 1))
    echo "Waiting for database... (attempt $attempt/$max_attempts)"
    sleep 2
done

if [ $attempt -eq $max_attempts ]; then
    echo "Failed to connect to database after $max_attempts attempts"
    exit 1
fi

echo "Running database migrations..."

PGPASSWORD=$DB_PASSWORD psql -v ON_ERROR_STOP=1 -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c "
CREATE TABLE IF NOT EXISTS schema_migrations (
  version VARCHAR(255) PRIMARY KEY,
  applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);"

bootstrap_done=$(PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -tAc "SELECT 1 FROM schema_migrations WHERE version = '__baseline_complete__' LIMIT 1;")
if [ "$bootstrap_done" = "1" ]; then
    bootstrap_mode=0
    echo "Migration mode: strict"
else
    bootstrap_mode=1
    echo "Migration mode: bootstrap (one-time baseline catch-up)"
fi

for file in $(ls /app/migrations/*.up.sql 2>/dev/null | sort); do
    if [ ! -f "$file" ]; then
        continue
    fi

    version=$(basename "$file" .up.sql)
    already_applied=$(PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -tAc "SELECT 1 FROM schema_migrations WHERE version = '$version' LIMIT 1;")
    if [ "$already_applied" = "1" ]; then
        echo "Skipping migration (already applied): $version"
        continue
    fi

    echo "Applying migration: $version"
    apply_output=$(PGPASSWORD=$DB_PASSWORD psql -v ON_ERROR_STOP=1 -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -f "$file" 2>&1) || apply_rc=$?
    apply_rc=${apply_rc:-0}

    if [ "$apply_rc" -ne 0 ]; then
        if [ "$bootstrap_mode" -eq 1 ]; then
            echo "Migration $version failed during bootstrap; baseline-marking as applied."
            echo "$apply_output"
        else
            echo "Migration $version failed in strict mode:"
            echo "$apply_output"
            exit 1
        fi
    else
        echo "$apply_output"
    fi

    PGPASSWORD=$DB_PASSWORD psql -v ON_ERROR_STOP=1 -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c "INSERT INTO schema_migrations (version) VALUES ('$version') ON CONFLICT (version) DO NOTHING;" >/dev/null
done

if [ "$bootstrap_mode" -eq 1 ]; then
    PGPASSWORD=$DB_PASSWORD psql -v ON_ERROR_STOP=1 -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c "INSERT INTO schema_migrations (version) VALUES ('__baseline_complete__') ON CONFLICT (version) DO NOTHING;" >/dev/null
    echo "Bootstrap baseline completed; future startups will use strict migration mode."
fi

echo "Validating critical tables exist..."
missing_tables=$(PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -tAc "
SELECT string_agg(t, ',') FROM (
  SELECT t FROM (VALUES
    ('users'),
    ('mt_accounts'),
    ('orders'),
    ('positions'),
    ('api_keys'),
    ('backtest_runs'),
    ('strategy_execution_logs'),
    ('order_history'),
    ('system_operation_logs')
  ) AS v(t)
  WHERE to_regclass('public.' || t) IS NULL
) m;
")

if [ -n "$missing_tables" ] && [ "$missing_tables" != "" ]; then
    echo "Missing critical tables after migrations: $missing_tables"
    exit 1
fi

echo "Migrations completed, starting server..."
exec /app/antrader -config /app/configs/config.yaml
