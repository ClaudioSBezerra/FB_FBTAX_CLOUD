# Ambiente de desenvolvimento local — WSL Ubuntu.
# Detalhes em dev/README.md
.PHONY: help setup dev backend frontend build test psql migrations db-reset admin-senha clean

help:
	@echo "FBTax Cloud — desenvolvimento local"
	@echo
	@echo "  make setup       Prepara o ambiente (PostgreSQL, banco, .env, deps)"
	@echo "  make dev         Sobe backend + frontend juntos"
	@echo "  make backend     Sobe só o backend  (:8086)"
	@echo "  make frontend    Sobe só o frontend (:3086)"
	@echo
	@echo "  make test        Testes do backend + typecheck do frontend"
	@echo "  make build       Compila os dois como em produção"
	@echo
	@echo "  make psql        Console do banco"
	@echo "  make migrations  Últimas migrations aplicadas"
	@echo "  make db-reset    APAGA e recria o banco"
	@echo "  make admin-senha Troca a senha de um usuário admin"
	@echo "  make clean       Remove binários, dist e logs"

setup:
	@./dev/setup.sh

dev:
	@./dev/start.sh

backend:
	@cd backend && go run .

frontend:
	@cd frontend && npm run dev

test:
	@echo "▸ Backend"; cd backend && go vet ./... && go test ./...
	@echo "▸ Frontend"; cd frontend && ./node_modules/.bin/tsc --noEmit && echo "  typecheck OK"

build:
	@cd backend && go build -o fb_cloud . && echo "▸ backend/fb_cloud"
	@cd frontend && npm run build

psql:
	@./dev/db.sh psql

migrations:
	@./dev/db.sh migrations

db-reset:
	@./dev/db.sh reset

admin-senha:
	@./dev/admin-senha.sh

clean:
	@rm -rf backend/fb_cloud frontend/dist dev/logs
	@echo "limpo"
