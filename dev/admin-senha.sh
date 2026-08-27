#!/usr/bin/env bash
# Troca a senha de um usuário admin gerando o hash bcrypt localmente.
#
# A senha é lida do terminal e nunca aparece em argumento de linha de comando,
# em arquivo ou no histórico do shell. O hash é gerado por um programa Go
# temporário, usando a mesma biblioteca e o mesmo custo (14) que o backend.
#
#   ./dev/admin-senha.sh                          → banco local
#   DATABASE_URL='postgres://…' ./dev/admin-senha.sh  → outro ambiente
set -euo pipefail

RAIZ="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
URL="${DATABASE_URL:-postgres://postgres:postgres@localhost:5432/${DB_NOME:-fbtax_cloud}?sslmode=disable}"

command -v psql >/dev/null || { echo "psql não encontrado. Rode: make setup" >&2; exit 1; }

read -rp  "E-mail do usuário: " email
[ -n "${email}" ] || { echo "E-mail obrigatório." >&2; exit 1; }
read -rsp "Nova senha: "        senha; echo
read -rsp "Confirme a senha: "  senha2; echo
[ "${senha}" = "${senha2}" ] || { echo "As senhas não conferem." >&2; exit 1; }
[ ${#senha} -ge 12 ] || { echo "Use ao menos 12 caracteres." >&2; exit 1; }

tmp="$(mktemp -d)"
trap 'rm -rf "${tmp}"' EXIT

cat > "${tmp}/main.go" <<'GOEOF'
package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

func main() {
	linha, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	// Custo 14 — o mesmo de handlers.HashPassword.
	h, err := bcrypt.GenerateFromPassword([]byte(strings.TrimRight(linha, "\n")), 14)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Print(string(h))
}
GOEOF

# Reaproveita o módulo do backend, que já tem golang.org/x/crypto no go.sum.
hash="$(cd "${RAIZ}/backend" && printf '%s\n' "${senha}" | go run "${tmp}/main.go")"
[ -n "${hash}" ] || { echo "Falha ao gerar o hash." >&2; exit 1; }

atualizados="$(psql "${URL}" -tAc "UPDATE users SET password_hash = '${hash}' WHERE email = '${email}' RETURNING 1;" | wc -l)"
if [ "${atualizados}" -eq 0 ]; then
  echo "Nenhum usuário com o e-mail ${email}." >&2
  exit 1
fi
echo "Senha atualizada para ${email}."
