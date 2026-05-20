---
plan: 01-01
status: complete
completed_at: "2026-05-20"
commit: ff5e89c
---

# Summary: 01-01 — Schema financeiro + Handlers Go

## What was built

- `backend/migrations/108_financeiro_schema.sql` — Cria o schema `financeiro` com `CREATE SCHEMA IF NOT EXISTS`
- `backend/migrations/109_financeiro_empresas.sql` — Tabelas `financeiro.empresas` e `financeiro.dados_bancarios` com FK ON DELETE CASCADE e índice em `empresa_id`
- `backend/handlers/financeiro.go` — `EmpresaHandler` e `DadosBancariosHandler` com CRUD GET/POST/PUT; nullable columns (`nome_fantasia`, `complemento`, `titular`) tratadas com `sql.NullString`; queries parameterizadas
- `backend/main.go` — Rotas `/api/financeiro/empresa` e `/api/financeiro/dados-bancarios` registradas com `withAuth(..., "admin")`

## Verification

- `go build ./...` → exit 0 (sem erros)
- Todas as queries usam `financeiro.` fully-qualified e `$N` parameterized
- GET sem token → 401 (via AuthMiddleware)
- POST/PUT requerem role `admin`

## Decisions made

- `nome_fantasia`, `complemento` e `titular` são nullable no SQL e escaneados com `sql.NullString`
- Validação mínima no handler: campos obrigatórios verificados antes do INSERT
- Modelo singleton v1: GET retorna LIMIT 1 (empresa única por instância)
