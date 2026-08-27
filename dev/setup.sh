#!/usr/bin/env bash
# Prepara o ambiente de desenvolvimento local no WSL Ubuntu.
# Idempotente: rodar de novo não quebra nada.
set -euo pipefail

RAIZ="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DB_NOME="${DB_NOME:-fbtax_cloud}"
DB_USER="${DB_USER:-postgres}"
DB_SENHA="${DB_SENHA:-postgres}"

azul()  { printf '\033[1;36m%s\033[0m\n' "$*"; }
ok()    { printf '\033[1;32m  ✓ %s\033[0m\n' "$*"; }
aviso() { printf '\033[1;33m  ! %s\033[0m\n' "$*"; }
erro()  { printf '\033[1;31m  ✗ %s\033[0m\n' "$*" >&2; }

azul "▸ Verificando ferramentas"
faltando=()
command -v go   >/dev/null || faltando+=("go")
command -v node >/dev/null || faltando+=("node")
command -v npm  >/dev/null || faltando+=("npm")
if [ ${#faltando[@]} -gt 0 ]; then
  erro "Faltam: ${faltando[*]}"
  echo "    Go:   https://go.dev/dl/  ·  Node: sudo apt install nodejs npm"
  exit 1
fi
ok "go $(go version | awk '{print $3}')  ·  node $(node --version)"

# ── PostgreSQL ───────────────────────────────────────────────────────────────
azul "▸ PostgreSQL"
if ! command -v psql >/dev/null && [ ! -d /usr/lib/postgresql ]; then
  aviso "Não instalado. Instalando (vai pedir sua senha do sudo)…"
  sudo apt-get update -qq
  sudo apt-get install -y -qq postgresql postgresql-contrib
fi
ok "instalado"

# No WSL com systemd, o serviço sobe por systemctl; sem systemd, cai no
# service(8), que é o que o WSL antigo usa.
if ! pg_isready -q 2>/dev/null; then
  aviso "Serviço parado. Iniciando (vai pedir sua senha do sudo)…"
  if [ "$(ps -p 1 -o comm=)" = "systemd" ]; then
    sudo systemctl enable --now postgresql
  else
    sudo service postgresql start
  fi
  for _ in $(seq 1 20); do pg_isready -q 2>/dev/null && break; sleep 0.5; done
fi
pg_isready -q || { erro "PostgreSQL não subiu."; exit 1; }
ok "no ar em $(pg_isready | head -1)"

azul "▸ Banco de dados"
# A senha do papel postgres precisa bater com a DATABASE_URL, que conecta por
# TCP em localhost e portanto exige autenticação por senha.
sudo -u postgres psql -qc "ALTER USER ${DB_USER} WITH PASSWORD '${DB_SENHA}';" >/dev/null
if sudo -u postgres psql -lqt | cut -d'|' -f1 | grep -qw "${DB_NOME}"; then
  ok "banco ${DB_NOME} já existe"
else
  sudo -u postgres createdb -O "${DB_USER}" "${DB_NOME}"
  ok "banco ${DB_NOME} criado"
fi

# ── backend/.env ─────────────────────────────────────────────────────────────
azul "▸ Configuração"
ENV_BACK="${RAIZ}/backend/.env"
if [ -f "${ENV_BACK}" ]; then
  ok "backend/.env preservado (não sobrescrevo o existente)"
else
  cat > "${ENV_BACK}" <<ENVEOF
PORT=8086
DATABASE_URL=postgres://${DB_USER}:${DB_SENHA}@localhost:5432/${DB_NOME}?sslmode=disable
JWT_SECRET=$(head -c 32 /dev/urandom | base64)
ADMIN_EMAIL=admin@fbtax.cloud
ALLOWED_ORIGINS=http://localhost:3086,http://localhost:3083
SMTP_HOST=localhost
SMTP_PORT=1025
SMTP_USER=
SMTP_PASS=
ENVEOF
  ok "backend/.env criado com JWT_SECRET aleatório"
fi

# ── Dependências ─────────────────────────────────────────────────────────────
azul "▸ Dependências"
(cd "${RAIZ}/backend" && go mod download) && ok "módulos Go"
if [ -d "${RAIZ}/frontend/node_modules" ]; then
  ok "node_modules já presente"
else
  (cd "${RAIZ}/frontend" && npm ci --silent) && ok "node_modules instalado"
fi

echo
azul "▸ Pronto. Suba tudo com:  make dev"
echo "    Frontend  http://localhost:3086"
echo "    Backend   http://localhost:8086/api/health"
echo "    Login     claudio_bezerra@hotmail.com / 123456"
