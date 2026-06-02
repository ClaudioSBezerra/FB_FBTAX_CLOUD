---
phase: quick-260602-krc
plan: 01
subsystem: financeiro
tags: [ofx, ingest, upload, dedup, bank]
decisions:
  - Auto-detect de conta via BANKID+ACCTID com fallback manual (dialog 409)
  - Dedup por FITID direto ou SHA-256(contaID|data|valor|descNorm) quando FITID ausente
  - conciliado=false e origem='ofx_upload' em todas as inserções via OFX
  - Retry de encoding cp1252->utf-8 para bancos brasileiros (Inter, Itaú)
  - Índice único parcial em referencia_ext WHERE referencia_ext IS NOT NULL
  - Dependência github.com/aclindsa/ofxgo@v0.1.3 adicionada ao go.mod
metrics:
  duration: ~64 minutos (15:08 a 16:12 BRT)
  completed_date: "2026-06-02"
  tasks_completed: 3
  tasks_total: 3
key_files:
  created:
    - backend/migrations/123_ofx_origem.sql (20 linhas)
    - backend/services/ofx_ingest.go (213 linhas)
    - backend/handlers/ofx_upload.go (55 linhas)
  modified:
    - backend/go.mod (adicionado github.com/aclindsa/ofxgo v0.1.3 e github.com/aclindsa/xml v0.0.0-20201125035057-bbd5c9ec99ac)
    - backend/go.sum (checksum atualizado)
    - backend/main.go (1 linha adicionada — registro da rota)
    - frontend/src/pages/PainelFinanceiroPage.tsx (153 linhas adicionadas — botão + 2 dialogs + estados + handleOFXUpload)
---

# Quick Task 260602-krc: Ingestão OFX no Painel Financeiro — Summary

**One-liner:** Parser OFX multipart com dedup por FITID/SHA-256, auto-detect de conta BANKID+ACCTID, dialog 409 de fallback e botão "Importar OFX" no card Contas.

## Status: COMPLETE

Todas as 3 tasks executadas com sucesso. `go build ./...` passa sem erros.

## Tasks Concluídas

| Task | Nome | Commit | Arquivos |
|------|------|--------|----------|
| 1 | Migration coluna origem + índice único | `861a126` | `backend/migrations/123_ofx_origem.sql` |
| 2 | Serviço ofx_ingest.go — parse + dedup + INSERT | `fd30ec8` | `backend/services/ofx_ingest.go`, `backend/go.mod`, `backend/go.sum` |
| 3 | Handler OFXUpload + rota + UI dialog | `0ece098` | `backend/handlers/ofx_upload.go`, `backend/main.go`, `frontend/src/pages/PainelFinanceiroPage.tsx` |

## Arquivos Criados/Modificados

| Arquivo | Tipo | Linhas |
|---------|------|--------|
| `backend/migrations/123_ofx_origem.sql` | CRIADO | 20 |
| `backend/services/ofx_ingest.go` | CRIADO | 213 |
| `backend/handlers/ofx_upload.go` | CRIADO | 55 |
| `backend/go.mod` | MODIFICADO | +3 linhas |
| `backend/main.go` | MODIFICADO | +1 linha |
| `frontend/src/pages/PainelFinanceiroPage.tsx` | MODIFICADO | 995 → 1147 linhas (+152) |

## Detalhes por Task

### Task 1 — Migration

`123_ofx_origem.sql` adiciona:
- `ALTER TABLE financeiro.transacoes ADD COLUMN origem VARCHAR(20) DEFAULT 'api_sync'` via DO block idempotente
- `CREATE UNIQUE INDEX IF NOT EXISTS idx_transacoes_referencia_ext_uq ON financeiro.transacoes(referencia_ext) WHERE referencia_ext IS NOT NULL` (índice parcial — NULLs excluídos)

### Task 2 — Serviço ofx_ingest.go

Tipos exportados: `OFXDetectedAccount`, `OFXErroDetalhe`, `OFXIngestResult`, `ErrContaNaoDetectada`.

Função principal `IngestOFX(db *sql.DB, contaID string, r io.Reader) (*OFXIngestResult, error)`:
- Parse com `ofxgo.ParseResponse`; fallback de encoding cp1252→utf-8 via `golang.org/x/text/encoding/charmap`
- Auto-detect de conta por `(banco ILIKE $1 OR provedor_id = $1) AND conta = $2` quando `contaID == ""`
- Dedup via `SELECT EXISTS(...)` por `referencia_ext` antes de cada INSERT
- `referencia_ext = FITID` ou `SHA-256(contaID|data|valor|descNorm)` como fallback
- INSERT com `conciliado=false`, `origem='ofx_upload'`, `categoria=NULL`
- Best-effort por linha: erros incrementam `Erros` sem interromper o loop

### Task 3 — Handler + Rota + UI

**Backend** (`ofx_upload.go`):
- `OFXUploadHandler` aceita apenas POST; `ParseMultipartForm(5<<20)` para limite 5 MB
- Lê campo `file` e `conta_id` do multipart
- Retorna 409 + `{ detected: { bankid, acctid, branchid } }` para `*ErrContaNaoDetectada`
- Retorna 200 + JSON do `OFXIngestResult` em caso de sucesso
- Rota registrada em `main.go`: `http.HandleFunc("/api/financeiro/ofx/upload", withAuth(handlers.OFXUploadHandler, "admin"))`

**Frontend** (`PainelFinanceiroPage.tsx`):
- Botão "Importar OFX" (ícone `Upload`, variant=ghost, size=sm) no card Contas, abaixo de "Nova conta"
- Dialog principal: input `type="file" accept=".ofx,.ofc"` que dispara upload ao selecionar; spinner durante processamento
- Dialog 409 fallback: exibe BANKID/ACCTID detectados, select de conta existente, botão "Importar para esta conta"
- `handleOFXUpload(file, contaId?)`: fetch POST multipart, trata 409 abrindo dialog fallback, toast.success com resumo N importadas/M duplicadas, toast.warning se erros > 0, chama `loadAll()` ao final

## Deviations from PLAN.md

Nenhum desvio. Plano executado exatamente como especificado.

Nota técnica: A API do ofxgo v0.1.3 usa `TrnTypeCredit` como constante exportada (não string) para comparação de tipo de transação; o serviço usa `tx.TrnType == ofxgo.TrnTypeCredit` conforme os tipos reais da lib. `trnType` é un-exported — correto.

## Threat Flags

Nenhuma superfície nova além da coberta pelo threat model do PLAN.md (T-ofx-01 a T-ofx-SC). Todas as mitigações implementadas:
- `ParseMultipartForm(5<<20)` limita upload a 5 MB (T-ofx-01)
- Queries parametrizadas em todos os INSERTs/SELECTs (T-ofx-04)
- `withAuth(..., "admin")` na rota (T-ofx-05)
- Dependência verificada: `github.com/aclindsa/ofxgo` é lib madura com histórico em pkg.go.dev (T-ofx-SC)

## Known Stubs

Nenhum. Todos os campos inseridos são dados reais do arquivo OFX parseado.

## Self-Check

- [x] `backend/migrations/123_ofx_origem.sql` — existe
- [x] `backend/services/ofx_ingest.go` — existe
- [x] `backend/handlers/ofx_upload.go` — existe
- [x] Commit `861a126` — existe (Task 1)
- [x] Commit `fd30ec8` — existe (Task 2)
- [x] Commit `0ece098` — existe (Task 3)
- [x] `go build ./...` — passa sem erros
