---
stepsCompleted: [1, 2, 3, 4, 5, 6, 7, 8]
inputDocuments:
  - FBTAX_CLOUD/_bmad-output/planning-artifacts/prd.md
  - FBTAX_CLOUD/_bmad-output/planning-artifacts/product-brief-FBTAX_CLOUD-2026-04-08.md
  - projetos/_bmad-output/planning-artifacts/project-context.md
workflowType: 'architecture'
project_name: 'FBTAX_CLOUD'
user_name: 'Claudio'
date: '2026-04-08'
status: 'complete'
completedAt: '2026-04-08'
---

# Architecture Decision Document

_This document builds collaboratively through step-by-step discovery. Sections are appended as we work through each architectural decision together._

---

## Project Context Analysis

### Requirements Overview

**Functional Requirements — 30 FRs em 7 áreas de capacidade:**

| Área | FRs | Implicação arquitetural |
|---|---|---|
| Vitrine de Produtos | FR1–FR5 | Frontend SPA estático; config de tenant no cliente |
| Notificações e Comunicados | FR6–FR9 | API de leitura pública + contador de visualizações |
| Contato Comercial | FR10–FR14 | Endpoint POST → SMTP + registro no banco |
| Administração de Produtos | FR15–FR19 | CRUD autenticado com JWT |
| Administração de Notificações | FR20–FR23 | CRUD autenticado com JWT |
| Métricas e Visibilidade | FR24–FR26 | Queries de agregação simples no banco |
| Infraestrutura e Operação | FR27–FR30 | Health check, rotas, SEO meta tags |

**Non-Functional Requirements — impacto arquitetural direto:**

| NFR | Alvo | Decisão arquitetural implicada |
|---|---|---|
| Page load < 2s | LCP Lighthouse | Bundle < 200kb; assets SVG/WebP |
| JWT admin | Expiração 8h | Padrão herdado FB_SMARTPICK |
| Prepared statements | 100% queries | Padrão herdado FB_SMARTPICK |
| Uptime ≥ 99% | Coolify health check | Docker Compose restart automático |
| WCAG 2.1 AA | aria-labels, contraste | Shadcn/UI atende por padrão |
| SMTP herdado | `services/email.go` | Zero modificação no serviço |

**Scale & Complexity:**

- Domínio primário: Full-stack web SPA — frontend estático + backend admin leve
- Complexidade: **Baixa** — menor que FB_SMARTPICK; sem processamento assíncrono, PDF ou CSV
- Componentes arquiteturais estimados: 3 páginas frontend + 7 handlers backend + 5 tabelas banco

### Technical Constraints & Dependencies

- Stack obrigatória: Go 1.22 + React 18 + PostgreSQL 15 + Coolify/Hostinger
- `services/email.go` herdado sem modificação — dependência do SMTP configurado no ambiente
- Porta dedicada a definir (não colidir com APU02:8081, SmartPick:8082)
- Subdomínio `www.fbtax.cloud` — configuração DNS + Traefik via Coolify

### Cross-Cutting Concerns Identified

- **JWT Auth** — apenas rotas `/api/admin/*` e painel `/admin` do frontend
- **CORS** — origem autorizada: `www.fbtax.cloud`
- **Audit log** — operações de escrita no painel admin (produtos e notificações)
- **SEO** — meta tags estáticas no `index.html` gerado no build
- **Tenant config** — mapeamento `tenant → produtos_ativos` em arquivo de configuração estático (sem banco)

---

## Starter Template Evaluation

### Primary Technology Domain

Full-stack web application — SPA React (frontend público + painel admin) com API Go (backend admin + formulário).

### Starter Options Considered

| Opção | Avaliação |
|---|---|
| Create React App / Vite fresh | ❌ Exigiria reconfigurar Tailwind, Shadcn, path aliases, Docker — trabalho desnecessário |
| T3 Stack / Next.js | ❌ Overhead desnecessário para portal simples; stack diferente do ecossistema fbtax.cloud |
| **Clone do FB_SMARTPICK** | ✅ Stack idêntica ao ecossistema, Shadcn configurado, auth herdado, Docker pronto |

### Selected Starter: Clone do FB_SMARTPICK

**Rationale para seleção:**
O FBTAX_CLOUD segue a mesma estratégia do FB_SMARTPICK (que clonou o FB_APU02) — herdar toda a infraestrutura e remover o que não se aplica. Esta abordagem elimina ~60% do trabalho de setup e garante consistência visual e operacional com o ecossistema fbtax.cloud.

**Initialization Command:**

```bash
# Clonar FB_SMARTPICK como base
git clone <fb_smartpick_repo> FBTAX_CLOUD
cd FBTAX_CLOUD

# Reinicializar git para repositório limpo
rm -rf .git && git init

# Ajustar módulo Go
# go.mod: module fb_cloud (não fb_smartpick)

# Limpar código de domínio SmartPick (handlers específicos, migrations 100+, páginas de domínio)
# Manter: auth, middleware, hierarchy, environment, email, Docker, Nginx, Coolify config
```

### Architectural Decisions Provided by Starter

**Language & Runtime:**
- Go 1.22 — `net/http` standard library, sem framework externo
- TypeScript 5.2.2 — strict mode ativo

**Styling Solution:**
- Tailwind CSS 3.4.3 + Shadcn/UI (Radix UI) — componentes prontos
- Path alias `@/` para `src/` configurado no Vite

**Build Tooling:**
- Vite 5.2.0 — build otimizado, HMR em dev
- Docker multi-stage build (herdado) — imagem de produção enxuta

**Testing Framework:**
- Nenhum framework de teste no clone base — a ser adicionado se necessário

**Code Organization:**
- Backend: `handlers/`, `services/`, `migrations/` — padrão herdado
- Frontend: `pages/`, `components/ui/`, `contexts/` — padrão herdado

**Development Experience:**
- `docker-compose.yml` para desenvolvimento local (Go + React + PostgreSQL)
- `.env` com variáveis de ambiente — nunca commitado
- Scripts de conveniência herdados

**Nota:** A primeira história de implementação deve ser o setup do clone + limpeza do código SmartPick + renomeação do módulo Go para `fb_cloud`.

---

## Core Architectural Decisions

### Decision Priority Analysis

**Decisões Críticas (bloqueiam implementação):**
- Schema do banco: `portal` com prefixo `pt_`
- Tenant config: tabela `pt_tenant_products` (banco — Opção B)
- Porta da API: **8083** (sem conflito com APU02:8081, SmartPick:8082)
- Módulo Go: `fb_cloud` (não `fb_smartpick`)

**Decisões Importantes (moldam arquitetura):**
- Auth admin: JWT herdado do FB_SMARTPICK — sem SSO, sem login de usuário final no portal
- API pública (sem auth): products, notifications, contact, events
- API admin (auth JWT): `/api/admin/*` para CRUD completo e métricas

**Decisões Adiadas (pós-MVP):**
- Analytics avançado — Google Analytics externo suficiente no MVP
- Cache de resposta — sem necessidade dado o volume baixo

---

### Data Architecture

**Schema e nomenclatura:**
- Todas as tabelas do portal vivem no schema `portal` com prefixo `pt_`
- PKs: UUID (`gen_random_uuid()`) — padrão herdado do FB_SMARTPICK
- Colunas: snake_case
- FKs: `ON DELETE CASCADE` ou `ON DELETE SET NULL` explícito

**Tabelas do domínio portal:**

```
portal.pt_products          — portfólio de produtos (nome, descrição, ícone, url, ativo)
portal.pt_tenant_products   — produtos contratados por tenant (tenant_slug, product_id, ativo)
portal.pt_notifications     — comunicados (tipo, título, texto, publicado, created_at)
portal.pt_contact_leads     — formulários enviados (nome, email, mensagem, created_at)
portal.pt_events            — rastreamento (tipo: 'cta_click'|'notif_view', ref_id, created_at)
```

**Tabelas herdadas do clone (schema `public`):**
- `users` — apenas para o admin (Claudio) — sem modificação
- Migrations 001–09x herdadas sem alteração

**Migrations:**
- Numeradas a partir de `100`: `100_portal_schema.sql`, `101_pt_products.sql`, etc.
- Executadas automaticamente pelo `onDBConnected()` herdado

---

### Authentication & Security

**Admin auth:**
- JWT com expiração 8h — gerado no `POST /api/admin/login`
- Senha armazenada com bcrypt — padrão herdado
- Middleware chain: `SecurityMiddleware` → `AuthMiddleware` → handler
- Rota `/admin` no frontend protegida por `ProtectedRoute` (herdado)

**API pública (sem auth):**
- `GET /api/portal/products` — lista produtos (filtrando por `?tenant=`)
- `GET /api/portal/notifications` — lista notificações publicadas
- `POST /api/portal/contact` — recebe formulário → SMTP + registra no banco
- `POST /api/portal/events` — registra clique/visualização

**CORS:** apenas `www.fbtax.cloud` como origem autorizada
**Prepared statements:** 100% das queries — padrão herdado obrigatório

---

### API & Communication Patterns

**Endpoints REST completos:**

```
# Público (sem auth)
GET  /api/health
GET  /api/portal/products?tenant=
GET  /api/portal/notifications
POST /api/portal/contact
POST /api/portal/events

# Admin (JWT obrigatório)
POST   /api/admin/login
GET    /api/admin/products
POST   /api/admin/products
PUT    /api/admin/products/:id
DELETE /api/admin/products/:id
GET    /api/admin/notifications
POST   /api/admin/notifications
PUT    /api/admin/notifications/:id
DELETE /api/admin/notifications/:id
GET    /api/admin/tenant-products
POST   /api/admin/tenant-products
DELETE /api/admin/tenant-products/:id
GET    /api/admin/metrics
```

**Handlers Go — padrão herdado:**
```go
func GetProductsHandler(db *sql.DB) http.HandlerFunc { ... }
// Registrado em main.go: mux.HandleFunc("/api/portal/products", GetProductsHandler(db))
```

---

### Frontend Architecture

**Rotas React (`react-router-dom`):**
```
/                    → PortalPage (público)
/admin/login         → AdminLogin (público)
/admin               → AdminDashboard (ProtectedRoute)
/admin/products      → AdminProducts (ProtectedRoute)
/admin/notifications → AdminNotifications (ProtectedRoute)
/admin/metrics       → AdminMetrics (ProtectedRoute)
```

**Contexts:**
- `AuthContext` — herdado, não modificar
- `PortalContext` — novo: fornece `products`, `notifications`, `tenantSlug`

**Estado:** TanStack Query para dados do servidor; `useState` para UI local

**Componentes novos:**
```
pages/
  PortalPage.tsx          → vitrine pública
  AdminLogin.tsx          → login admin
  AdminDashboard.tsx      → painel principal
  AdminProducts.tsx       → CRUD produtos
  AdminNotifications.tsx  → CRUD notificações
  AdminMetrics.tsx        → visualização de métricas
components/
  ProductCard.tsx         → ícone + nome + descrição + estado ativo/inativo
  NotificationList.tsx    → lista de comunicados públicos
  ContactForm.tsx         → formulário de contato
```

---

### Infrastructure & Deployment

- Coolify/Hostinger — mesmo padrão dos outros produtos
- Porta API: **8083**
- Subdomínio: `www.fbtax.cloud`
- Docker Compose: serviços `backend`, `frontend`, `db`
- Health check: `GET /api/health` (herdado)

**Variáveis de ambiente:**
```
DB_URL, JWT_SECRET, SMTP_HOST, SMTP_PORT, SMTP_USER, SMTP_PASS
ADMIN_EMAIL, PORT=8083
```

### Decision Impact Analysis

**Sequência de implementação:**
1. Clone + limpeza + renomear módulo Go para `fb_cloud`
2. Migrations 100–104 (schema portal)
3. Handlers públicos (products, notifications, contact, events)
4. Frontend — PortalPage + componentes públicos
5. Handlers admin (CRUD + métricas)
6. Frontend — painel admin completo
7. Deploy Coolify + DNS `www.fbtax.cloud`

**Dependências cross-component:**
- `pt_tenant_products` depende de `pt_products` estar populado
- Frontend admin depende de todos os endpoints admin disponíveis
- SMTP depende de variáveis de ambiente configuradas no Coolify

---

## Implementation Patterns & Consistency Rules

### Pontos de Conflito Identificados: 8 áreas críticas

---

### Naming Patterns

**Banco de Dados — Convenções de Nomenclatura:**

```
Schema:     portal  (apenas tabelas do domínio portal)
Prefixo:    pt_     (todas as tabelas novas)
Tabelas:    snake_case plural  → pt_products, pt_notifications
Colunas:    snake_case         → product_id, created_at, tenant_slug
PKs:        uuid DEFAULT gen_random_uuid()
FKs:        <tabela>_id        → product_id (FK para pt_products)
Índices:    idx_<tabela>_<col> → idx_pt_events_ref_id
```

Tabelas herdadas (schema `public`) — **sem prefixo, sem modificação**: `users`

**API — Convenções de Nomenclatura:**

```
Endpoints públicos:  /api/portal/<recurso>    → /api/portal/products
Endpoints admin:     /api/admin/<recurso>     → /api/admin/products
Operações CRUD:      plural no path           → /api/admin/products
Parâmetro de rota:   /:id                     → /api/admin/products/:id
Query strings:       snake_case               → ?tenant=fbtax_cloud
```

**Código Go — Convenções:**

```
Handlers:   PascalCase + sufixo "Handler"    → GetProductsHandler, CreateNotificationHandler
Funções DB: snake_case nas queries, PascalCase no Go
Arquivos:   snake_case.go                    → handlers/products.go, handlers/notifications.go
Serviços:   snake_case.go (herdado)          → services/email.go — NÃO RENOMEAR
```

**Código TypeScript — Convenções:**

```
Componentes:      PascalCase.tsx              → ProductCard.tsx, ContactForm.tsx
Pages:            PascalCase.tsx              → PortalPage.tsx, AdminProducts.tsx
Contexts:         PascalCase + sufixo Context → PortalContext, AuthContext (herdado)
Hooks:            camelCase + prefixo "use"   → usePortalData, useNotifications
Variáveis:        camelCase                   → tenantSlug, productList
Tipos/Interfaces: PascalCase                 → Product, Notification, TenantProduct
```

---

### Structure Patterns

**Organização Backend (Go) — herdada do FB_SMARTPICK:**

```
handlers/
  products.go          ← GetProductsHandler, CreateProductHandler, etc.
  notifications.go     ← GetNotificationsHandler, CreateNotificationHandler, etc.
  contact.go           ← PostContactHandler
  events.go            ← PostEventHandler
  tenant_products.go   ← tenant-products CRUD
  metrics.go           ← GetMetricsHandler
  auth.go              ← PostLoginHandler (herdado, revisar apenas)
services/
  email.go             ← HERDADO — NÃO MODIFICAR
migrations/
  100_portal_schema.sql
  101_pt_products.sql
  102_pt_tenant_products.sql
  103_pt_notifications.sql
  104_pt_contact_leads.sql
  105_pt_events.sql
```

**Organização Frontend (React) — herdada + novos:**

```
src/
  pages/
    PortalPage.tsx          ← página pública principal
    AdminLogin.tsx          ← login (pode herdar da base)
    AdminDashboard.tsx      ← painel admin
    AdminProducts.tsx       ← CRUD produtos
    AdminNotifications.tsx  ← CRUD notificações
    AdminMetrics.tsx        ← métricas
  components/
    ProductCard.tsx         ← novo
    NotificationList.tsx    ← novo
    ContactForm.tsx         ← novo
    ui/                     ← Shadcn/UI — NÃO MODIFICAR componentes base
  contexts/
    AuthContext.tsx          ← HERDADO — NÃO MODIFICAR
    PortalContext.tsx        ← novo
  lib/
    api.ts                  ← funções de fetch (adaptar do base)
    utils.ts                ← herdado (cn, formatação)
```

---

### Format Patterns

**Resposta da API — Formato padrão herdado:**

```json
// Sucesso — lista
{ "data": [...], "total": 10 }

// Sucesso — item único
{ "data": { ... } }

// Erro
{ "error": "mensagem legível" }

// Health check
{ "status": "ok" }
```

**Datas:**
- Banco: `TIMESTAMPTZ` com timezone
- API JSON: ISO 8601 string → `"2026-04-08T14:30:00Z"`
- Frontend: formatar com `Intl.DateTimeFormat` ou `date-fns` (herdado)

**Campos JSON — mapeamento Go → TypeScript:**
- Structs Go: tags `json:"field_name"` sempre em snake_case
- Interfaces TypeScript: camelCase (mapeamento automático não existe — usar conversão explícita ou manter snake_case no frontend também)

```go
// Go struct — padrão obrigatório
type Product struct {
    ID          string `json:"id"`
    Name        string `json:"name"`
    Description string `json:"description"`
    IconURL     string `json:"icon_url"`
    ProductURL  string `json:"product_url"`
    IsActive    bool   `json:"is_active"`
    CreatedAt   string `json:"created_at"`
}
```

---

### Communication Patterns

**TanStack Query — query keys hierárquicos:**

```typescript
["products"]                    ← lista pública
["products", tenantSlug]        ← lista filtrada por tenant
["notifications"]               ← lista pública
["admin", "products"]           ← lista admin
["admin", "notifications"]      ← lista admin
["admin", "metrics"]            ← métricas

// Invalidação após mutação
queryClient.invalidateQueries({ queryKey: ["admin", "products"] })
```

**Estado local (useState):** apenas para estado de UI (modal aberto, form fields) — nunca para dados do servidor.

---

### Process Patterns

**Tratamento de Erros — Backend Go:**

```go
// Padrão obrigatório em todos os handlers
if err != nil {
    log.Printf("erro interno: %v", err)   // log interno apenas
    http.Error(w, "mensagem para o usuário", http.StatusBadRequest)
    return
}
```

**Tratamento de Erros — Frontend:**
- TanStack Query captura erros via `error` state da query/mutation
- Exibir feedback via Shadcn Toast component
- Nunca expor stack trace ou mensagem técnica ao usuário

**Loading States — padrão Shadcn/UI:**
- Usar `isLoading` / `isPending` do TanStack Query
- Skeleton components durante carregamento de listas
- Botões com `disabled` + spinner durante submissão de forms

**Queries SQL — obrigatório sem exceção:**

```go
// SEMPRE prepared statements
rows, err := db.Query(
    "SELECT id, name FROM portal.pt_products WHERE is_active = $1",
    true,
)
// NUNCA concatenação de strings em queries SQL
```

**Autenticação JWT — padrão herdado:**
- Header: `Authorization: Bearer <token>`
- Expiração: 8h (JWT_SECRET do ambiente)
- `AuthMiddleware` aplicado antes de qualquer handler admin — NÃO reimplementar
- `AuthContext` no frontend gerencia token — NÃO duplicar lógica

---

### Enforcement Guidelines

**Todos os agentes de IA DEVEM:**

1. Usar prepared statements em 100% das queries SQL
2. Usar prefixo `pt_` em todas as novas tabelas no schema `portal`
3. Registrar handlers em `main.go` no padrão `mux.HandleFunc("/api/...", Handler(db))`
4. Nunca modificar `services/email.go`, `AuthContext.tsx`, nem componentes `ui/`
5. Usar o módulo Go `fb_cloud` (não `fb_smartpick`)
6. Retornar erros no formato `{"error": "mensagem"}` em todos os handlers
7. Tags JSON em structs Go sempre em snake_case
8. Query keys do TanStack Query seguindo a hierarquia definida acima

**Anti-Patterns — PROIBIDO:**

```
❌ Criar tabelas sem prefixo pt_ no schema portal
❌ Usar fmt.Sprintf para montar queries SQL
❌ Duplicar lógica de auth fora do AuthMiddleware/AuthContext
❌ Usar fetch direto em vez de TanStack Query para dados do servidor
❌ Modificar components/ui/ (Shadcn base)
❌ Criar novo schema de banco fora de "portal" ou "public"
❌ Usar porta diferente de 8083
❌ Nomear migrations com número < 100 (conflito com herdadas)
```

---

## Project Structure & Boundaries

### Árvore Completa do Projeto

```
FBTAX_CLOUD/
├── .env                          ← variáveis locais (nunca commitado)
├── .env.example                  ← template de variáveis
├── .gitignore
├── docker-compose.yml            ← dev local: backend + frontend + db
├── docker-compose.prod.yml       ← produção (Coolify)
│
├── backend/                      ← Go 1.22
│   ├── Dockerfile
│   ├── go.mod                    ← module fb_cloud
│   ├── go.sum
│   ├── main.go                   ← entry point, registro de rotas
│   ├── handlers/
│   │   ├── auth.go               ← POST /api/admin/login (herdado)
│   │   ├── products.go           ← GET/POST/PUT/DELETE /api/portal+admin/products
│   │   ├── notifications.go      ← GET/POST/PUT/DELETE /api/portal+admin/notifications
│   │   ├── contact.go            ← POST /api/portal/contact
│   │   ├── events.go             ← POST /api/portal/events
│   │   ├── tenant_products.go    ← GET/POST/DELETE /api/admin/tenant-products
│   │   └── metrics.go            ← GET /api/admin/metrics
│   ├── middleware/
│   │   ├── auth.go               ← AuthMiddleware JWT (herdado)
│   │   ├── cors.go               ← CORS www.fbtax.cloud (herdado)
│   │   └── security.go           ← SecurityMiddleware (herdado)
│   ├── services/
│   │   └── email.go              ← HERDADO — NÃO MODIFICAR
│   ├── migrations/
│   │   ├── 001_...sql            ← herdadas do FB_SMARTPICK (sem modificação)
│   │   ├── ...
│   │   ├── 09x_...sql
│   │   ├── 100_portal_schema.sql ← CREATE SCHEMA portal
│   │   ├── 101_pt_products.sql
│   │   ├── 102_pt_tenant_products.sql
│   │   ├── 103_pt_notifications.sql
│   │   ├── 104_pt_contact_leads.sql
│   │   └── 105_pt_events.sql
│   └── db/
│       └── db.go                 ← conexão + onDBConnected (herdado)
│
├── frontend/                     ← React 18 + Vite 5 + TypeScript
│   ├── Dockerfile
│   ├── package.json
│   ├── tsconfig.json
│   ├── vite.config.ts
│   ├── tailwind.config.js
│   ├── index.html                ← meta tags SEO fbtax.cloud
│   ├── public/
│   │   ├── favicon.ico
│   │   ├── sitemap.xml
│   │   └── robots.txt
│   └── src/
│       ├── main.tsx              ← entry point React
│       ├── App.tsx               ← router + rotas protegidas
│       ├── pages/
│       │   ├── PortalPage.tsx          ← / (público)
│       │   ├── AdminLogin.tsx          ← /admin/login
│       │   ├── AdminDashboard.tsx      ← /admin
│       │   ├── AdminProducts.tsx       ← /admin/products
│       │   ├── AdminNotifications.tsx  ← /admin/notifications
│       │   └── AdminMetrics.tsx        ← /admin/metrics
│       ├── components/
│       │   ├── ProductCard.tsx         ← ícone + nome + status + CTA
│       │   ├── NotificationList.tsx    ← lista de comunicados
│       │   ├── ContactForm.tsx         ← formulário + envio
│       │   └── ui/                     ← Shadcn/UI — NÃO MODIFICAR
│       ├── contexts/
│       │   ├── AuthContext.tsx         ← HERDADO — NÃO MODIFICAR
│       │   └── PortalContext.tsx       ← produtos + notificações + tenantSlug
│       ├── lib/
│       │   ├── api.ts                  ← funções fetch (adaptar do base)
│       │   ├── queryClient.ts          ← TanStack Query config
│       │   └── utils.ts                ← cn(), formatDate() (herdado)
│       └── types/
│           └── index.ts               ← Product, Notification, TenantProduct, Event
│
└── nginx/
    └── nginx.conf                ← proxy reverso frontend → backend (herdado)
```

---

### Fronteiras Arquiteturais

**API Boundaries:**

```
Público (sem auth)         → frontend direto, qualquer origem
  GET  /api/health
  GET  /api/portal/products?tenant=
  GET  /api/portal/notifications
  POST /api/portal/contact
  POST /api/portal/events

Admin (JWT obrigatório)    → AuthMiddleware valida token antes do handler
  POST /api/admin/login    ← única exceção: não requer JWT (gera o token)
  *    /api/admin/*        ← todos os demais exigem JWT válido
```

**Component Boundaries — Frontend:**

```
PortalContext  →  provê: products[], notifications[], tenantSlug
                  consome: GET /api/portal/products, GET /api/portal/notifications

AuthContext    →  provê: user, token, login(), logout()
                  consome: POST /api/admin/login
                  HERDADO — não alterar interface pública

ProtectedRoute →  wrapper herdado — lê AuthContext, redireciona para /admin/login
```

**Data Boundaries:**

```
schema public  →  tabelas herdadas (users) — acesso somente por handlers herdados
schema portal  →  tabelas pt_* — acesso exclusivo pelos novos handlers
```

---

### Mapeamento FRs → Estrutura

| Área de Requisitos | Handler Go | Page/Component React | Tabelas |
|---|---|---|---|
| Vitrine de Produtos (FR1–5) | `handlers/products.go` | `PortalPage.tsx` + `ProductCard.tsx` | `pt_products`, `pt_tenant_products` |
| Notificações Públicas (FR6–9) | `handlers/notifications.go` | `NotificationList.tsx` | `pt_notifications`, `pt_events` |
| Contato Comercial (FR10–14) | `handlers/contact.go` | `ContactForm.tsx` | `pt_contact_leads` |
| Admin Produtos (FR15–19) | `handlers/products.go` | `AdminProducts.tsx` | `pt_products`, `pt_tenant_products` |
| Admin Notificações (FR20–23) | `handlers/notifications.go` | `AdminNotifications.tsx` | `pt_notifications` |
| Métricas (FR24–26) | `handlers/metrics.go` | `AdminMetrics.tsx` | `pt_events`, `pt_contact_leads` |
| Infraestrutura (FR27–30) | `main.go` + `db/db.go` | `App.tsx` + `index.html` | — |

---

### Fluxo de Dados

```
Visitante público:
  Browser → nginx → Vite SPA → PortalPage
                             ↓ TanStack Query
                             → GET /api/portal/products?tenant=xxx
                             → GET /api/portal/notifications
                             → POST /api/portal/contact  (form)
                             → POST /api/portal/events   (clique/view)

Admin (Claudio):
  Browser → /admin/login → AuthContext → POST /api/admin/login → JWT
  Browser → /admin/* → ProtectedRoute → TanStack Query
                                       → /api/admin/* (com Bearer token)
```

---

## Architecture Validation Results

### Coherence Validation ✅

| Componente A | Componente B | Compatibilidade |
|---|---|---|
| Go 1.22 `net/http` | PostgreSQL 15 via `database/sql` | ✅ Padrão herdado, sem conflito |
| React 18.3 + Vite 5 | TanStack Query + Shadcn/UI | ✅ Versões compatíveis e testadas |
| JWT 8h + bcrypt | AuthMiddleware herdado | ✅ Sem modificação, sem risco |
| Porta 8083 | APU02:8081, SmartPick:8082 | ✅ Sem conflito |
| schema `portal` + prefixo `pt_` | Migrations herdadas (001–09x) | ✅ Isolamento total por namespace |
| `services/email.go` herdado | SMTP env vars | ✅ Dependência de env, sem código novo |

**Pattern Consistency:** Naming conventions snake_case (DB/Go tags) e PascalCase (Go handlers, React) são coerentes com o clone base. Query keys hierárquicos do TanStack Query não conflitam com nenhuma decisão.

**Structure Alignment:** Árvore de diretórios espelha exatamente o padrão FB_SMARTPICK com adições cirúrgicas — apenas handlers novos, pages novas e contexts novos.

---

### Requirements Coverage Validation ✅

| Área | FRs | Handler | Frontend | DB | Status |
|---|---|---|---|---|---|
| Vitrine de Produtos | FR1–5 | `products.go` | `PortalPage` + `ProductCard` | `pt_products`, `pt_tenant_products` | ✅ |
| Notificações | FR6–9 | `notifications.go` | `NotificationList` | `pt_notifications`, `pt_events` | ✅ |
| Contato Comercial | FR10–14 | `contact.go` | `ContactForm` | `pt_contact_leads` | ✅ |
| Admin Produtos | FR15–19 | `products.go` | `AdminProducts` | `pt_products`, `pt_tenant_products` | ✅ |
| Admin Notificações | FR20–23 | `notifications.go` | `AdminNotifications` | `pt_notifications` | ✅ |
| Métricas | FR24–26 | `metrics.go` | `AdminMetrics` | `pt_events`, `pt_contact_leads` | ✅ |
| Infraestrutura | FR27–30 | `main.go` health check | `index.html` SEO | — | ✅ |

**NFRs:** Performance (Vite bundle), Security (JWT/bcrypt/prepared statements), Accessibility (Shadcn WCAG 2.1 AA), Reliability (Docker restart + health check) — todos cobertos. ✅

---

### Gap Analysis Results

**Gaps Críticos:** Nenhum.

**Gaps Importantes:**
- Seed data inicial de produtos → incluir nas stories de setup
- Conteúdo do `.env.example` → documentar nas stories de deploy

---

### Architecture Completeness Checklist

- [x] 30 FRs e 20 NFRs analisados e cobertos
- [x] Stack completa especificada com versões (Go 1.22, React 18.3.1, PG 15, Vite 5.2.0)
- [x] Módulo Go `fb_cloud`, porta `8083`, schema `portal`, prefixo `pt_` definidos
- [x] 5 tabelas do domínio portal especificadas
- [x] 14 endpoints REST documentados (público + admin)
- [x] 6 páginas frontend + 3 componentes novos definidos
- [x] 8 áreas de conflito de padrões identificadas e endereçadas
- [x] Anti-patterns explicitamente documentados
- [x] Estrutura completa de diretórios definida
- [x] Fronteiras API, dados e componentes estabelecidas

**Overall Status: PRONTO PARA IMPLEMENTAÇÃO** — Confidence: Alta

---

## Architecture Completion Summary

**Architecture Decision Workflow:** COMPLETED ✅
**Total Steps Completed:** 8/8
**Date Completed:** 2026-04-08
**Document Location:** `_bmad-output/planning-artifacts/architecture.md`

### Final Deliverables

- **30 FRs** e **20 NFRs** cobertos arquiteturalmente
- **14 endpoints REST** documentados (5 públicos + 9 admin)
- **5 tabelas** do domínio portal especificadas (`pt_products`, `pt_tenant_products`, `pt_notifications`, `pt_contact_leads`, `pt_events`)
- **6 páginas** + **3 componentes** + **2 contexts** frontend definidos
- **8 áreas de conflito** de padrões endereçadas com anti-patterns explícitos
- Árvore completa de diretórios com todos os arquivos mapeados

### Para Agentes de IA

Este documento é o guia completo para implementação do FBTAX_CLOUD. Siga todas as decisões, padrões e estruturas exatamente como documentadas.

**Primeira ação:**
```bash
git clone <fb_smartpick_repo> FBTAX_CLOUD
cd FBTAX_CLOUD && rm -rf .git && git init
# go.mod: module fb_cloud
```

**Sequência de implementação:**
1. Clone + limpeza + renomear módulo Go para `fb_cloud`
2. Migrations 100–105 (schema portal)
3. Handlers públicos (products, notifications, contact, events)
4. Frontend — PortalPage + componentes públicos
5. Handlers admin (CRUD + métricas)
6. Frontend — painel admin completo
7. Deploy Coolify + DNS `www.fbtax.cloud`

---

**Architecture Status: READY FOR IMPLEMENTATION ✅**
