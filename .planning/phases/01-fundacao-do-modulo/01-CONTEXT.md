# Phase 1: Fundação do Módulo - Context

**Gathered:** 2026-05-19
**Status:** Ready for planning

<domain>
## Phase Boundary

Estabelecer o schema PostgreSQL `financeiro` e entregar CRUD funcional dos dados da empresa Fortes Bezerra (dados cadastrais + dados bancários), acessível ao admin via rota simples no React. Esta fase NÃO constrói o painel admin completo (fase 5) nem cria tabelas para fases futuras — apenas o que a fase 1 usa.

**Requirements desta fase:** EMP-01, EMP-02, EMP-03

</domain>

<decisions>
## Implementation Decisions

### Schema PostgreSQL

- **D-01:** Usar schema separado `financeiro.*` — mesmo padrão do `portal.*` (migrations 100–107). Queries usarão nomes fully-qualified: `financeiro.empresas`, `financeiro.dados_bancarios`.
- **D-02:** Criar `CREATE SCHEMA IF NOT EXISTS financeiro;` na primeira migration desta fase (108).

### Migrations

- **D-03:** Migrations incrementais — cada fase cria apenas as tabelas que ela precisa. Phase 1 cria somente `financeiro.empresas` e `financeiro.dados_bancarios`.
- **D-04:** Numeração continua da sequência existente: migration 108 e 109 (última atual: 107).
- **D-05:** Duas tabelas separadas: `financeiro.empresas` (CNPJ, razão social, endereço) + `financeiro.dados_bancarios` (banco, agência, conta, tipo). Separação permite múltiplas contas bancárias no v2 sem refactor.

### UI / Frontend

- **D-06:** Rota simples `/admin/financeiro/empresa` em `frontend/src/App.tsx`, protegida pelo `ProtectedRoute` existente. Sem layout de painel — página standalone. O painel completo (sidebar + nav) vem na fase 5.
- **D-07:** Dois formulários separados na mesma página: seção "Dados da Empresa" com seu botão Salvar + seção "Dados Bancários" com seu botão Salvar independente. Admin pode editar cada seção sem afetar a outra.

### Auth / Role

- **D-08:** Rotas `/api/financeiro/*` protegidas com `withAuth(handler, "admin")` — reusar a role `admin` existente. O admin FBTax Cloud e o admin do módulo financeiro são a mesma pessoa no v1.
- **D-09:** Role `fb_admin` NÃO será criada nesta fase. Avaliar na fase 5 quando o painel separar os contextos de admin.

### Claude's Discretion

- Nomes exatos das colunas nas tabelas (ex: `logradouro`, `numero`, `bairro`, `cep`, `municipio`, `uf` para endereço) — seguir convenções do domínio fiscal brasileiro.
- Nome do arquivo handler no backend — `financeiro.go` ou `empresa.go` em `backend/handlers/`.
- Estrutura interna dos componentes React — usar shadcn/ui + React Hook Form + Zod conforme padrão do projeto.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Requirements e Roadmap
- `.planning/REQUIREMENTS.md` — EMP-01 (dados cadastrais), EMP-02 (dados bancários), EMP-03 (edição). Success criteria da fase.
- `.planning/ROADMAP.md` — Phase 1 goal e success criteria detalhados.

### Padrão de Schema Separado (precedente)
- `backend/migrations/100_pt_products.sql` — Como o schema `portal` foi criado. Mesmo padrão a seguir para `financeiro`.

### Padrão de Handlers e Auth
- `backend/main.go:322–342` — Closures `withDB` e `withAuth` — como registrar novas rotas.
- `backend/main.go:344–381` — Onde novas rotas são registradas (bloco `http.HandleFunc`).
- `backend/handlers/auth.go:207–259` — `AuthMiddleware` — lógica de verificação de role (`admin` tem bypass universal).
- `backend/handlers/portal_products.go` — Handler de referência para leitura (padrão de estrutura).

### Frontend / Routing
- `frontend/src/App.tsx` — Onde a nova rota `/admin/financeiro/empresa` será adicionada; padrão `ProtectedRoute`.
- `frontend/src/contexts/AuthContext.tsx` — Como `ProtectedRoute` verifica autenticação.

### Padrões de Migration
- `backend/migrations/015_create_auth_system.sql` — Exemplo de criação de tabela com UUID primary key e convenções de colunas.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `frontend/src/components/ui/` — Input, Button, Card, Label, Separator, Form — todos disponíveis via shadcn/ui. Usar para o formulário de empresa sem criar novos componentes.
- `ProtectedRoute` em `frontend/src/App.tsx:10` — Wrapper de autenticação já existe; apenas envolver a nova página.
- `backend/services/email.go` — Disponível para futuras notificações (não usado na fase 1).

### Established Patterns
- **Handler factory**: `func EmpresaHandler(db *sql.DB) http.HandlerFunc { return func(w, r) {...} }` — seguir exatamente.
- **Method dispatch manual**: O projeto não usa router com suporte a métodos; handlers verificam `r.Method` manualmente (`GET`, `POST`, `PUT`).
- **Sem URL path params**: Todos os IDs via query string ou body — não `/api/financeiro/empresa/:id`.
- **JSON tags snake_case**: Campos Go com json tag snake_case (`razao_social`, `dados_bancarios`).
- **SQL raw com `lib/pq`**: Sem ORM. `db.QueryRow(...)`, `db.Exec(...)`, `rows.Scan(...)`.
- **Migrations**: Arquivo `NNN_descricao.sql` em `backend/migrations/`, auto-aplicado no boot.

### Integration Points
- **Backend**: Novo arquivo `backend/handlers/financeiro.go` (ou `empresa.go`) + registrar rotas em `backend/main.go:344–381`.
- **Frontend**: Nova página `frontend/src/pages/EmpresaPage.tsx` + rota em `frontend/src/App.tsx`.
- **Migration**: Arquivos `backend/migrations/108_financeiro_schema.sql` + `109_financeiro_empresas.sql` (ou similar — Claude decide granularidade).

</code_context>

<specifics>
## Specific Ideas

- A tabela `financeiro.dados_bancarios` deve suportar `tipo_conta` (corrente/poupança) como VARCHAR para flexibilidade.
- Dois formulários independentes na UI refletem as duas tabelas separadas no banco — consistência UX/data model.
- A empresa Fortes Bezerra é um singleton (uma única empresa dona do módulo) — tabela `empresas` terá no máximo 1 registro no v1.

</specifics>

<deferred>
## Deferred Ideas

- **Role `fb_admin`**: Criar role separada para admin do módulo financeiro. Avaliar na fase 5 quando o painel precisar separar contextos (admin FBTax vs admin Fortes Bezerra).
- **Layout do painel financeiro** (sidebar + nav): Construir na fase 5 — Painel Admin. A rota `/admin/financeiro/empresa` da fase 1 é temporária e sem layout.
- **Múltiplas contas bancárias**: O modelo de tabelas separadas já abre espaço, mas a UI de múltiplas contas é v2+.

</deferred>

---

*Phase: 1-Fundação do Módulo*
*Context gathered: 2026-05-19*
