---
plan: 02-01
status: complete
completed_at: "2026-05-20"
commit: 827a6f4
---

# Summary: 02-01 — Migrations + Handlers + Rotas (Backend)

## What was built

- `backend/migrations/110_financeiro_clientes.sql` — Tabelas `financeiro.clientes` (CNPJ principal único) e `financeiro.cliente_cnpjs` (multi-CNPJ); GIN indexes com `gin_trgm_ops` em `razao_social` e `cnpj` para suporte a ILIKE
- `backend/migrations/111_financeiro_produtos.sql` — Tabelas `financeiro.produtos` e `financeiro.planos`; seed de 4 produtos (FB_APU02, FB_APU04, FB_SMARTPICK, FB_FAROL) e 5 planos cada (Lite, Standard, Premium, Enterprise, Sob Demanda) com `preco = NULL`; `ON CONFLICT DO NOTHING` idempotente
- `backend/migrations/112_financeiro_contratos.sql` — Tabelas `financeiro.contratos`, `financeiro.contrato_cnpjs` (FK → `cliente_cnpjs`) e `financeiro.contrato_itens` (FK → `planos`); UNIQUE constraints em ambas as junction tables
- `backend/handlers/clientes.go` — `ClientesHandler` com GET list (filtros `?q=` ILIKE + `?status=`), GET by id (inclui `[]ClienteCNPJResponse`), POST (transação: INSERT clientes + INSERT cliente_cnpjs is_principal=true), PUT, e action `add-cnpj`
- `backend/handlers/produtos.go` — `ProdutosHandler` (GET lista produtos com planos aninhados via `fetchPlanos`), `PlanosHandler` (PUT preço, aceita `null`)
- `backend/handlers/contratos.go` — `ContratosHandler` com GET list (`?cliente_id=`), GET by id, POST (transação: INSERT contratos + loop cnpj_ids + loop itens), PUT (status, periodicidade, valor_total, observacoes)
- `backend/main.go` — 4 novas rotas registradas dentro do bloco `// ── Financeiro ──` com `withAuth(..., "admin")`

## Verification

- `go build ./...` → exit 0 (sem erros de compilação)
- 4 rotas confirmadas com `grep -c`
- `sql.NullFloat64` usado para `preco` (planos) e `valor_item` (contrato_itens)
- Todas as transações usam `db.Begin()` + `defer tx.Rollback()` + `tx.Commit()`

## Decisions made

- `preco` dos planos é nullable (NULL = Sob Demanda, preço sob consulta)
- `valor_item` em contrato_itens é nullable (override por item; NULL = usa preço do plano)
- `handleListClientes` usa `DISTINCT` + `LEFT JOIN cliente_cnpjs` para que filtro `?q=` alcance CNPJs adicionais
- Ordem dos planos no `fetchPlanos` é hard-coded via CASE (Lite→Standard→Premium→Enterprise→Sob Demanda)
