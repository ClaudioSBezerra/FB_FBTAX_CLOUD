---
phase: "01"
slug: fundacao-do-modulo
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-05-19
---

# Phase 01 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | `go test` (backend), nenhum detectado (frontend) |
| **Config file** | `tests/integration_test.go` (existente backend); frontend: Wave 0 instala |
| **Quick run command** | `go test ./... -timeout 30s` |
| **Full suite command** | `go test ./... -v -timeout 60s` |
| **Estimated runtime** | ~15 segundos (backend only) |

---

## Sampling Rate

- **After every task commit:** Run `go test ./... -timeout 30s`
- **After every plan wave:** Run `go test ./... -v -timeout 60s`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 30 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 01-migration-schema | 01 | 1 | EMP-01 | — | Migrations 108–109 criam schema `financeiro` e tabelas sem erros | integration | `go test ./... -run TestMigrations -v` | ❌ Wave 0 | ⬜ pending |
| 01-empresa-post | 01 | 1 | EMP-01 | T-01-sqli | POST `/api/financeiro/empresa` com dados válidos → 201 + registro em DB | integration | `go test ./handlers -run TestEmpresaPost -v` | ❌ Wave 0 | ⬜ pending |
| 01-dados-bancarios-post | 01 | 1 | EMP-02 | T-01-sqli | POST `/api/financeiro/dados-bancarios` com dados válidos → 201 + registro em DB | integration | `go test ./handlers -run TestDadosBancariosPost -v` | ❌ Wave 0 | ⬜ pending |
| 01-empresa-put | 01 | 1 | EMP-03 | T-01-authz | PUT `/api/financeiro/empresa` com ID existente → 200 + atualização em DB | integration | `go test ./handlers -run TestEmpresaPut -v` | ❌ Wave 0 | ⬜ pending |
| 01-role-check | 01 | 1 | EMP-01 | T-01-authz | Request sem token `admin` → 401/403 | integration | `go test ./handlers -run TestEmpresaAuthRequired -v` | ❌ Wave 0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `tests/handlers_financeiro_test.go` — testes de GET/POST/PUT para `EmpresaHandler` e `DadosBancariosHandler`, validação de role `admin`
- [ ] `tests/migrations_test.go` — verificação de que migrations 108–109 criam schema `financeiro` e tabelas corretamente
- [ ] Frontend: instalar `vitest` + criar teste básico para `EmpresaPage.tsx` (atualmente nenhum framework de testes detectado no projeto)

**Nota:** O projeto não possui testes frontend. Wave 0 deve instalar framework antes de implementar a página.

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Dois formulários independentes na UI | EMP-03 | Sem testes E2E | Abrir `/admin/financeiro/empresa`, editar "Dados da Empresa", salvar, verificar que "Dados Bancários" não foi alterado e vice-versa |
| Rota protegida redireciona sem auth | EMP-01 | Comportamento de redirect requer browser | Abrir `/admin/financeiro/empresa` sem estar logado → deve redirecionar para `/admin/login` |
| Persistência após reload | EMP-01, EMP-02 | Teste de regressão manual | Salvar dados, recarregar página, verificar que dados aparecem nos formulários |

---

## Threats Addressed

| Threat ID | Category | Mitigation in Phase |
|-----------|----------|---------------------|
| T-01-sqli | SQL Injection | Parameterized queries (`$1, $2`) em todos os handlers; sem string interpolation |
| T-01-authz | Unauthorized Access | `withAuth(handler, "admin")` obrigatório em todas as rotas `/api/financeiro/*` |
| T-01-cors | CORS bypass | `SecurityMiddleware` existente em `main.go:388` cobre novas rotas automaticamente |
