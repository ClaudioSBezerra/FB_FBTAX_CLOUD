#!/usr/bin/env bash
# Atalhos de banco no ambiente local.
#   ./dev/db.sh psql      → abre o console
#   ./dev/db.sh migrations→ lista migrations aplicadas
#   ./dev/db.sh reset     → APAGA e recria o banco (pede confirmação)
set -euo pipefail

DB_NOME="${DB_NOME:-fbtax_cloud}"
URL="postgres://postgres:postgres@localhost:5432/${DB_NOME}?sslmode=disable"

case "${1:-psql}" in
  psql)
    exec psql "${URL}"
    ;;
  migrations)
    psql "${URL}" -c "SELECT filename, executed_at FROM schema_migrations ORDER BY filename DESC LIMIT 15;"
    ;;
  reset)
    printf '\033[1;31mIsso APAGA o banco %s por completo.\033[0m\n' "${DB_NOME}"
    read -rp "Digite o nome do banco para confirmar: " confirma
    [ "${confirma}" = "${DB_NOME}" ] || { echo "Cancelado."; exit 1; }
    sudo -u postgres dropdb --if-exists "${DB_NOME}"
    sudo -u postgres createdb -O postgres "${DB_NOME}"
    echo "Banco recriado. As migrations reaplicam no próximo 'make dev'."
    ;;
  *)
    echo "uso: $0 [psql|migrations|reset]" >&2
    exit 1
    ;;
esac
