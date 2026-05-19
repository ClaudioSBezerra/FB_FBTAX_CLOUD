# Walking Skeleton — Phase 01: Fundação do Módulo

**Fase:** 01-fundacao-do-modulo
**Tipo:** Brownfield addition — novo módulo sobre FBTax Cloud existente
**Criado:** 2026-05-19

---

## O que este Skeleton Prova

Esta fase é brownfield: o FBTax Cloud já está rodando. O skeleton prova que o módulo financeiro
está conectado ponta-a-ponta:

```
Migration SQL roda no boot
  → schema financeiro existe no PostgreSQL
  → handler Go responde em /api/financeiro/empresa
  → React exibe a página /admin/financeiro/empresa com formulários funcionais
  → admin consegue salvar e recuperar dados da empresa Fortes Bezerra
```

Ao final da fase, essa cadeia completa funciona. Nenhuma parte fica "stub" ou sem conexão real.

---

## Decisões Arquiteturais (Fixadas para Fases Futuras)

### Schema PostgreSQL

| Decisão | Valor | Rationale |
|---------|-------|-----------|
| Nome do schema | `financeiro` | Mesmo padrão do `portal.*` (migrations 100–107) — isolamento de domínio |
| Primeira migration | 108 — cria schema | `CREATE SCHEMA IF NOT EXISTS financeiro;` |
| Tabelas fase 1 | `financeiro.empresas`, `financeiro.dados_bancarios` | Apenas o que a fase 1 usa (D-03) |
| Primary keys | UUID via `gen_random_uuid()` | Padrão do projeto inteiro |
| Timestamps | `created_at`, `updated_at` com timezone | Padrão do projeto inteiro |
| Nomes de coluna | Português (domínio fiscal BR): `razao_social`, `cnpj`, `logradouro`, etc. | Convenção estabelecida |

### Backend Go

| Decisão | Valor | Rationale |
|---------|-------|-----------|
| Novo arquivo handler | `backend/handlers/financeiro.go` | Convenção: um arquivo por domínio |
| Assinatura do handler | `func EmpresaHandler(db *sql.DB) http.HandlerFunc` | Padrão factory do projeto |
| Dispatch de método | `switch r.Method { case GET/POST/PUT }` | Sem router externo — padrão project |
| Autenticação | `withAuth(handlers.EmpresaHandler, "admin")` | Role `admin` existente (D-08, D-09) |
| Registro de rotas | Em `backend/main.go:344–381` após seção Portal | Bloco `// ── Financeiro ──` |
| Role `fb_admin` | NÃO criada nesta fase | Avaliada na fase 5 (D-09) |

### Frontend React

| Decisão | Valor | Rationale |
|---------|-------|-----------|
| Nova página | `frontend/src/pages/EmpresaPage.tsx` | Convenção: um arquivo por rota |
| Rota | `/admin/financeiro/empresa` | Namespace `/admin/financeiro/*` para o módulo (D-06) |
| Proteção | `ProtectedRoute` existente em `App.tsx` | Reutilizar — não criar novo wrapper |
| UI de formulário | Dois formulários independentes com botão Salvar separado | D-07 |
| Componentes | shadcn/ui: `Card`, `Input`, `Label`, `Button`, `Separator`, `Alert` | Padrão do projeto — sem novos componentes |
| Estado | `useState` + `useEffect` para fetch inicial | Sem TanStack Query nesta fase — padrão Login.tsx |
| Layout | Página standalone sem sidebar/nav | Painel completo vem na fase 5 (D-06) |

### Padrões que Fases Futuras DEVEM Seguir

1. **Todas as tabelas do módulo financeiro** ficam no schema `financeiro.*` — nunca em `public`
2. **Migrations são incrementais** — cada fase cria apenas o que ela precisa (D-03)
3. **Queries sempre fully-qualified**: `financeiro.empresas`, `financeiro.dados_bancarios`, etc.
4. **Handlers financeiro** seguem o padrão factory: `func XxxHandler(db *sql.DB) http.HandlerFunc`
5. **Rotas** registradas no bloco `// ── Financeiro ──` em `backend/main.go`
6. **Role `admin`** protege todas as rotas `/api/financeiro/*` até a fase 5 criar separação
7. **Namespace de URL**: `/api/financeiro/*` (backend) + `/admin/financeiro/*` (frontend)

---

## Estrutura de Diretórios Adicionada

```
backend/
├── migrations/
│   ├── 108_financeiro_schema.sql        ← CREATE SCHEMA IF NOT EXISTS financeiro
│   └── 109_financeiro_empresas.sql      ← CREATE TABLE empresas + dados_bancarios
└── handlers/
    └── financeiro.go                    ← EmpresaHandler + DadosBancariosHandler

frontend/src/
├── pages/
│   └── EmpresaPage.tsx                  ← Dois formulários independentes
└── App.tsx                              ← +rota /admin/financeiro/empresa
```

---

## Verificação do Skeleton (Definition of Done)

O skeleton está completo quando TODOS estes checks passam:

```bash
# 1. Backend compila sem erros
cd backend && go build ./...

# 2. Frontend compila sem erros
cd frontend && npm run build

# 3. GET /api/financeiro/empresa retorna 401 sem token (auth funciona)
curl -s -o /dev/null -w "%{http_code}" http://localhost:8086/api/financeiro/empresa
# Esperado: 401

# 4. GET /api/financeiro/empresa com token admin retorna 404 (schema existe, tabela vazia)
# Esperado: 404 com {"error":"empresa not found"}

# 5. POST /api/financeiro/empresa com token admin retorna 201
# Esperado: 201 com {"id":"<uuid>"}

# 6. GET /api/financeiro/empresa após POST retorna 200 com dados
# Esperado: 200 com objeto empresa

# 7. Frontend: /admin/financeiro/empresa carrega sem 404 ou crash
# Esperado: página com dois formulários
```

---

## Tecnologia Consolidada para o Módulo Financeiro

| Camada | Tecnologia | Versão | Status |
|--------|-----------|--------|--------|
| Banco de dados | PostgreSQL 15 + schema `financeiro` | 15.x | Brownfield existente |
| Backend API | Go 1.26 + `net/http` stdlib | 1.26 | Brownfield existente |
| Autenticação | `github.com/golang-jwt/jwt/v5` | v5.3.1 | Brownfield existente |
| DB Driver | `github.com/lib/pq` | v1.11.2 | Brownfield existente |
| Frontend UI | React 18 + TypeScript 5.2 | 18.3/5.2 | Brownfield existente |
| Componentes | shadcn/ui (Radix + Tailwind) | 3.4 | Brownfield existente |
| Deploy | Coolify + Docker (container único) | — | Brownfield existente |

Nenhuma nova dependência adicionada na fase 1.
