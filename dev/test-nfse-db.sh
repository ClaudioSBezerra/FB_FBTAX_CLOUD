#!/usr/bin/env bash
# Aplica as migrations do financeiro (110-117 e 125) num banco DESCARTÁVEL e roda os testes de
# NFS-e que dependem de Postgres (contador de DPS concorrente). Nunca toca no banco de dev.
set -euo pipefail

cd "$(dirname "$0")/.."
BASE="${PG_BASE:-postgres://postgres:postgres@localhost:5432}"
DB="nfse_test_$$"

psql -q "${BASE}/postgres?sslmode=disable" -c "CREATE DATABASE ${DB}"
trap 'psql -q "${BASE}/postgres?sslmode=disable" -c "DROP DATABASE IF EXISTS ${DB}" >/dev/null' EXIT

for f in 110 111 112 113 114 115 116 117 125 126; do
  psql -q -v ON_ERROR_STOP=1 "${BASE}/${DB}?sslmode=disable" \
    -c 'CREATE SCHEMA IF NOT EXISTS financeiro; CREATE EXTENSION IF NOT EXISTS pg_trgm;' \
    -f backend/migrations/${f}_*.sql
done

cd backend
TEST_DATABASE_URL="${BASE}/${DB}?sslmode=disable" go test ./services/ ./handlers/ -run NFSe -count=1 -race -v
