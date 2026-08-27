#!/usr/bin/env bash
# Sobe backend (Go) e frontend (Vite) juntos. Ctrl+C derruba os dois.
set -euo pipefail

RAIZ="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
LOGS="${RAIZ}/dev/logs"
mkdir -p "${LOGS}"

azul() { printf '\033[1;36m%s\033[0m\n' "$*"; }
ok()   { printf '\033[1;32m  ✓ %s\033[0m\n' "$*"; }
erro() { printf '\033[1;31m  ✗ %s\033[0m\n' "$*" >&2; }

# ── PostgreSQL precisa estar no ar antes do backend ──────────────────────────
if ! pg_isready -q 2>/dev/null; then
  azul "▸ PostgreSQL parado — iniciando"
  if [ "$(ps -p 1 -o comm=)" = "systemd" ]; then
    sudo systemctl start postgresql
  else
    sudo service postgresql start
  fi
  for _ in $(seq 1 20); do pg_isready -q 2>/dev/null && break; sleep 0.5; done
fi
pg_isready -q || { erro "PostgreSQL fora do ar. Rode: make setup"; exit 1; }
ok "PostgreSQL no ar"

# Derruba os dois processos ao sair, seja por Ctrl+C ou por erro.
pids=()
encerrar() {
  echo
  azul "▸ Encerrando"
  for pid in "${pids[@]:-}"; do
    [ -n "${pid}" ] && kill "${pid}" 2>/dev/null || true
  done
  wait 2>/dev/null || true
  ok "processos encerrados"
}
trap encerrar EXIT INT TERM

azul "▸ Backend  :8086"
(cd "${RAIZ}/backend" && go run . 2>&1 | tee "${LOGS}/backend.log") &
pids+=($!)

# O backend aplica as migrations no boot; esperar deixa o log do frontend
# legível e evita a tela quebrar por API ainda fora do ar.
for _ in $(seq 1 60); do
  curl -sf http://localhost:8086/api/health >/dev/null 2>&1 && break
  sleep 0.5
done
if curl -sf http://localhost:8086/api/health >/dev/null 2>&1; then
  ok "backend respondendo"
else
  erro "backend não respondeu em 30s — veja dev/logs/backend.log"
fi

azul "▸ Frontend :3086"
(cd "${RAIZ}/frontend" && npm run dev 2>&1 | tee "${LOGS}/frontend.log") &
pids+=($!)

echo
azul "▸ No ar"
echo "    Frontend  http://localhost:3086"
echo "    Admin     http://localhost:3086/admin/financeiro/entregas"
echo "    Health    http://localhost:8086/api/health"
echo "    Login     claudio_bezerra@hotmail.com / 123456"
echo
echo "    Ctrl+C encerra os dois processos."
wait
