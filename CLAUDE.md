<!-- GSD:project-start source:PROJECT.md -->
## Project

**FBTax Cloud — Módulo Financeiro (Portal Fortes Bezerra)**

FBTax Cloud é uma plataforma SaaS de gestão fiscal para empresas brasileiras, hospedada em `www.fbtax.cloud`. O próximo milestone adiciona um **Módulo Financeiro** — o "Portal Fortes Bezerra" — que funciona simultaneamente como sistema de gestão de contratos/licenças dos produtos FB e como portal de acesso para clientes. O módulo vive dentro do mesmo repositório e deploy do FBTax Cloud (novas rotas Go + novas páginas React), acessível via botão "Acessar Fortes Bezerra" na interface existente.

**Core Value:** **O pagador recebe acesso, o inadimplente perde — automaticamente.** Um token de liberação válido vinculado a um contrato ativo é o que mantém os produtos FB_APU02, FB_APU04, FB_SMARTPICK e FB_FAROL funcionando. Esse fluxo (contrato → pagamento → token → produto) é o coração do módulo.

### Constraints

- **Tech stack**: Go (backend) + React/TypeScript (frontend) — sem adicionar novos runtimes no v1
- **Deploy**: mesma infraestrutura Coolify/Docker — módulo financeiro não cria novo container separado no v1
- **Banco de dados**: PostgreSQL existente — novas entidades via migrations SQL
- **Autenticação**: aproveitar o sistema JWT/roles existente — admin interno usará role específica
- **Sem gateway externo no v1**: cobranças manuais até o v2 resolver o spike de gateway
<!-- GSD:project-end -->

<!-- GSD:stack-start source:codebase/STACK.md -->
## Technology Stack

## Languages
- Go 1.26 — Backend API server (`backend/main.go`, `backend/handlers/`, `backend/services/`)
- TypeScript 5.2 — Frontend SPA (`frontend/src/`)
- Python 3.x — ERP Bridge daemon (`erp-bridge-aws/bridge.py`)
- SQL — PostgreSQL migrations (`backend/migrations/*.sql`) and Oracle ERP queries (`scripts_oracle/*.sql`)
## Runtime
- Go runtime 1.26 (backend, compiled CGO_ENABLED=0 for static binary)
- Node.js 18 (frontend build stage only, Alpine image)
- Python 3 with `venv` (ERP Bridge on AWS EC2, run via systemd)
- Go: Go modules — lockfile `backend/go.sum` present
- Frontend: npm — lockfile `frontend/package-lock.json` present
- Python: pip (dependencies: `oracledb`, `requests`, `pyyaml`) — no `requirements.txt`, install instructions inline in `erp-bridge-aws/bridge.py`
## Frameworks
- No web framework — uses Go stdlib `net/http` exclusively (`backend/main.go`)
- `github.com/golang-jwt/jwt/v5` v5.3.1 — JWT token auth
- `github.com/lib/pq` v1.11.2 — PostgreSQL driver
- `github.com/joho/godotenv` v1.5.1 — `.env` loading in development
- `golang.org/x/crypto` v0.49.0 — bcrypt password hashing
- `github.com/johnfercher/maroto/v2` v2.4.0 — PDF generation
- `github.com/google/uuid` v1.6.0 — UUID generation
- React 18.3 — UI framework (`frontend/src/`)
- Vite 5.2 — build tool and dev server (`frontend/vite.config.ts`)
- `@vitejs/plugin-react-swc` — SWC-based React transform (faster builds)
- React Router DOM 6.22 — SPA routing (`frontend/src/App.tsx`)
- TanStack Query 5.90 — server state management
- React Hook Form 7.71 + Zod 4.3 — form handling and validation
- Tailwind CSS 3.4 + Radix UI — component styling (full Radix UI primitive set)
- Recharts 3.7 — data visualization charts
- `xlsx` 0.18.5 — spreadsheet export
- shadcn/ui component system (Radix + Tailwind, `frontend/src/components/ui/`)
- `lucide-react` 0.363 — icon set
- `date-fns` 4.1 — date utilities
- `next-themes` 0.4.6 — dark/light theme support
- Go standard `testing` package — integration test in `tests/integration_test.go`
- No frontend testing framework detected
- Docker Buildx — multi-stage builds
- Vite dev server with `/api` proxy to backend on port 8086
- `start_dev.bat`, `start_docker.bat` — Windows dev helpers
## Key Dependencies
- `github.com/lib/pq` v1.11.2 — sole PostgreSQL driver; no ORM, raw SQL throughout
- `github.com/golang-jwt/jwt/v5` v5.3.1 — all auth flows depend on this
- `golang.org/x/crypto` — bcrypt for password storage
- `@tanstack/react-query` v5 — all API data fetching on frontend
- `python-oracledb` — Oracle thin client (no Oracle Instant Client needed), required for ERP Bridge
- `github.com/johnfercher/maroto/v2` v2.4.0 — PDF report generation (stored as `ai_reports`)
- `recharts` v3.7 — fiscal comparison charts on frontend
- `zod` v4 — runtime validation on frontend forms
## Configuration
- Backend reads env vars at startup via `godotenv.Load()` (dev) or Docker environment block (prod)
- Key vars: `DATABASE_URL`, `JWT_SECRET`, `SMTP_HOST`, `SMTP_PORT`, `SMTP_USER`, `SMTP_PASSWORD`, `SMTP_FROM`, `ALLOWED_ORIGINS`, `PORT`, `APP_URL`
- Template documented in `coolify-env-template.txt`
- ERP Bridge reads `erp-bridge-aws/config.yaml` (YAML config file, not env vars)
- Production: `Dockerfile.production` — 3-stage (Node build → Go build → Alpine runtime)
- Backend only: `backend/Dockerfile` — 2-stage (Go builder → Alpine runtime)
- Frontend only: `frontend/Dockerfile` — 2-stage (Node builder → nginx:stable-alpine)
- Frontend build env var: `VITE_APP_MODULE=fbtax_cloud` (passed as ARG)
- Frontend dev proxy: `VITE_API_TARGET` (defaults to `http://localhost:8086`)
## Database
- `pg_trgm` extension (trigram search on NF-e names, migration 067)
- `gen_random_uuid()` — primary keys throughout
- `portal` schema namespace for portal-related tables (migration 100+)
- Materialized views for fiscal summary data
- MaxOpenConns: 50
- MaxIdleConns: 15
- ConnMaxLifetime: 15 min
## Platform Requirements
- Docker + Docker Compose
- Node.js 18+ (frontend dev server)
- Go 1.26+ (backend dev)
- Python 3.11+ with `oracledb`, `requests`, `pyyaml` (ERP Bridge)
- Windows dev scripts: `start_dev.bat`, `start_docker.bat`
- Deployed on Coolify (self-hosted PaaS) via Docker Compose
- Traefik as reverse proxy — routes `/api/*` to Go backend (port 8086), all other paths to Nginx frontend (port 80)
- TLS via Let's Encrypt (`certresolver=letsencrypt`)
- Domain: `fbtax.cloud` and `www.fbtax.cloud`
- Container image pushed to GHCR (`ghcr.io/claudiosbezerra/...`) via GitHub Actions
- Staging environment: Azure (SSH deploy, no Coolify)
- ERP Bridge: runs on client AWS EC2 as a systemd service (`erp-bridge-aws/erp-bridge.service`)
<!-- GSD:stack-end -->

<!-- GSD:conventions-start source:CONVENTIONS.md -->
## Conventions

## Language Mix
| Language | Location | Notes |
|----------|----------|-------|
| Go 1.26 | `backend/` | Standard Go idioms; no linter config detected |
| TypeScript 5.2 + React 18 | `frontend/` | Strict TS; ESLint via Vite defaults |
| Python 3.12 | `erp-bridge-aws/bridge.py` | Single script; no formatter config detected |
- Comments in Go handler files: Portuguese for domain logic (`// Obtém o environment_id da empresa`), English for infrastructure patterns (`// Check blacklist before validating signature`)
- Log messages in Go: English prefixes with Portuguese content (`log.Printf("[Login] User not found: %s", req.Email)`)
- Python logging: mostly Portuguese (`log.info("Conectado ao Oracle (thin mode)")`)
- Frontend UI strings: Portuguese
## Naming Patterns
### Go (backend/)
### TypeScript/React (frontend/)
### Python (erp-bridge-aws/)
## Code Style
### Go
### TypeScript/React
### Python
## Import Organization
### Go
### TypeScript/React
## Error Handling
### Go
### TypeScript/React
### Python
## Logging
### Go
### Python
### Frontend
## Comments
## Function Design
### Go
### TypeScript/React
### Python
## Module Design
### Go
### TypeScript/React
<!-- GSD:conventions-end -->

<!-- GSD:architecture-start source:ARCHITECTURE.md -->
## Architecture

## System Overview
```text
```
## Component Responsibilities
| Component | Responsibility | Key File |
|-----------|----------------|----------|
| Go Backend | REST API, JWT auth, DB migrations, static serving | `backend/main.go` |
| Handlers | HTTP request logic, one package, no router lib | `backend/handlers/` |
| Services | Reusable business logic (email, PDF) | `backend/services/email.go` |
| Migrations | Sequential SQL, auto-applied on boot | `backend/migrations/*.sql` |
| React Frontend | SPA — portal page + admin login | `frontend/src/App.tsx` |
| AuthContext | Global auth state, JWT + company context via localStorage | `frontend/src/contexts/AuthContext.tsx` |
| UI Components | shadcn/ui components (Radix primitives + Tailwind) | `frontend/src/components/ui/` |
| Pages | Route-level views | `frontend/src/pages/` |
| ERP Bridge | Python daemon — reads Oracle ERP, pushes to backend via HTTP | `erp-bridge-aws/bridge.py` |
| Installer | Docker Compose + setup scripts for on-premise customer deploys | `installer/` |
| Traefik | TLS termination, routing by path prefix | `docker-compose.yml` labels |
## Pattern Overview
- Single Go binary handles API, migration runner, and static file serving (SPA fallback)
- No router framework — `net/http` standard library `http.HandleFunc` only, routes registered in `backend/main.go`
- DB injection pattern: every route is `withDB(handlers.SomeHandler)` or `withAuth(handlers.SomeHandler, role)` — closures inject `*sql.DB`
- Migration auto-apply on startup: `onDBConnected()` in `backend/main.go` scans `migrations/*.sql` and executes unapplied files tracked in `schema_migrations`
- Multi-tenant data model: `environments` → `enterprise_groups` → `companies` → users/data
- Portal schema (`portal.*`) is a separate PostgreSQL schema for the public product catalog and tenant subscriptions
- ERP Bridge is deployed independently on customer-side AWS, authenticates with `X-API-Key` header
- Frontend in dev proxies `/api/*` via Vite to Go backend; in production Traefik routes `/api/*` directly to the Go container and all other traffic to the Nginx container
## Layers
- Purpose: Route HTTP requests, inject DB, enforce auth
- Location: `backend/main.go` (lines 344–381)
- Contains: `http.HandleFunc` registrations, `withDB` and `withAuth` closures, static file server
- Depends on: `handlers` package
- Used by: Traefik (production), Vite dev proxy (development)
- Purpose: HTTP request parsing, validation, DB interaction, JSON responses
- Location: `backend/handlers/`
- Contains: `auth.go`, `admin.go`, `environment.go`, `hierarchy.go`, `middleware.go`, `portal_products.go`
- Depends on: `database/sql`, `services` package
- Used by: routing layer via `withDB` / `withAuth` closures
- Purpose: Business logic reused across handlers
- Location: `backend/services/`
- Contains: `email.go` — SMTP sending, HTML email templates for password reset, AI reports
- Depends on: OS environment variables (`SMTP_*`)
- Used by: `handlers/auth.go` (password reset emails), future report handlers
- Purpose: Database schema versioning, applied automatically at startup
- Location: `backend/migrations/*.sql`
- Contains: 107 numbered SQL files (000 through 107)
- Depends on: `schema_migrations` table in PostgreSQL
- Used by: `onDBConnected()` in `backend/main.go`
- Purpose: React SPA — public portal and admin interface
- Location: `frontend/src/`
- Contains: Pages, components, contexts, hooks
- Depends on: Go backend via `/api/*` fetch calls
- Used by: Nginx container (production), Vite dev server (development)
- Purpose: Extract fiscal data from Oracle ERP and push to FBTAX backend
- Location: `erp-bridge-aws/bridge.py`
- Contains: Oracle query logic, XML normalization, SAP S/4HANA batch mode, FBTaxClient HTTP wrapper, SQLite tracker
- Depends on: Go backend `/api/erp-bridge/*` endpoints, Oracle DB
- Used by: Customer AWS VMs running the bridge as a Linux daemon (`erp-bridge.service`)
## Data Flow
### Portal Page (Public)
### Authenticated Admin Login
### ERP Bridge Import (SAP Mode)
### ERP Bridge Import (Oracle XML Legacy Mode)
### DB Migration on Startup
- JWT token stored in `localStorage`; company/environment context also in `localStorage`
- `AuthContext` (`frontend/src/contexts/AuthContext.tsx`) is the single source of truth
- `window.fetch` is monkey-patched by `AuthContext` to inject `Authorization` and `X-Company-ID` headers globally
## Key Abstractions
- Purpose: Inject `*sql.DB` into handlers and optionally enforce JWT authentication
- Location: `backend/main.go:322–342`
- Pattern: Higher-order functions returning `http.HandlerFunc`, avoiding global state for DB
- Purpose: Parse and validate JWT from `Authorization: Bearer` header; store claims in `context.Value(ClaimsKey)`
- Location: `backend/handlers/auth.go` (exported as `handlers.AuthMiddleware`)
- Pattern: Standard Go middleware wrapping `http.HandlerFunc`
- Purpose: CORS enforcement, CSP headers, X-Frame-Options, HSTS, preflight handling
- Location: `backend/handlers/middleware.go:90`
- Pattern: Wraps the entire `http.DefaultServeMux`; uses `secureResponseWriter` to intercept and override headers
- Purpose: Multi-tenant data isolation
- Tables: `environments` → `enterprise_groups` → `companies` → `users` (via `user_environments`)
- Examples: `backend/migrations/013_create_environment_hierarchy.sql`, `backend/handlers/environment.go`
- Pattern: UUID primary keys, `ON DELETE CASCADE` throughout
- Purpose: Public product catalog and tenant subscriptions for `fbtax.cloud` portal page
- Tables: `portal.pt_products`, `portal.pt_tenants`, `portal.pt_tenant_products`, `portal.pt_notifications`, `portal.pt_events`, `portal.pt_product_urls`
- Location: `backend/migrations/100_pt_products.sql` through `107_pt_farol_description.sql`
- Pattern: Separate PostgreSQL schema; queries use `portal.pt_*` fully qualified names
- Purpose: HTTP client wrapper that handles authentication, token refresh, and all FBTax API calls
- Location: `erp-bridge-aws/bridge.py:291`
- Pattern: Session-based with auto-retry on 401; supports both user credentials (login flow) and `X-API-Key` (batch endpoints)
## Entry Points
- Location: `backend/main.go:259` — `func main()`
- Triggers: Docker CMD (`./fb_fbtax_cloud`), or `go run main.go` locally
- Responsibilities: Load `.env`, validate JWT secret, start async DB init, register all HTTP routes, start HTTP server
- Location: `frontend/src/main.tsx`
- Triggers: Loaded by Nginx (production) or Vite dev server
- Responsibilities: Mount React root, wrap in `QueryClientProvider`, delegate to `App.tsx`
- Location: `frontend/src/App.tsx`
- Responsibilities: Define routes — `/` → `PortalPage`, `/admin/login` → `Login`, `/admin` → protected dashboard stub, `*` → redirect to `/`
- Location: `erp-bridge-aws/bridge.py` — `if __name__ == "__main__":` block (end of file)
- Triggers: CLI invocation or `erp-bridge.service` systemd unit (`erp-bridge-aws/erp-bridge.service`)
- Responsibilities: Parse args (`--data`, `--mes`, `--daemon`, `--servidor`), load `config.yaml`, run import cycle
- Location: `docker-compose.yml`
- Services: `api` (Go), `web` (React/Nginx), `db` (PostgreSQL 15), `db-init` (one-shot init container)
- Location: `docker-compose.prod.yml`
- Services: same minus `db-init`; uses port 8083 instead of 8086
- Location: `Dockerfile.production`
- Pattern: 3-stage build — Node 18 builds frontend → Go 1.26 builds backend → Alpine final image bundles both; Go backend serves `./static/` as SPA fallback
## Architectural Constraints
- **Threading:** Go standard library event loop; `initDBAsync()` uses a single goroutine for DB connection retries; auth token cleanup uses a background goroutine with hourly ticker
- **Global state:** `db *sql.DB` and `dbMutex sync.RWMutex` in `backend/main.go` package scope; `refreshTokenStore sync.Map` and `tokenBlacklist sync.Map` in `backend/handlers/auth.go` package scope — both are in-memory and reset on restart
- **No external router:** Routes are registered directly on `http.DefaultServeMux`; all route matching is exact prefix, not parameterized — no URL path parameters (IDs must be query parameters or request body)
- **No Redis in main stack:** `config/app.yaml` references Redis but `docker-compose.yml` does not include it; Redis appears only in `installer/cliente-aws/docker-compose.yml` (a different product image)
- **Migrations are irreversible:** No down-migration mechanism; `.sql.disabled` extension used to skip files without deletion
## Anti-Patterns
### Handlers Declared But Not Registered
### In-Memory Token State
### Monkey-Patched `window.fetch`
## Error Handling
- DB errors return `http.StatusInternalServerError` with a plain text message
- Auth failures return `http.StatusUnauthorized` or `http.StatusForbidden`
- Validation failures return `http.StatusBadRequest`
- ERP Bridge logs errors to file (`erp-bridge-aws/logs/`) and continues processing remaining items; run status is set to `partial` or `error` in `erp_bridge_runs`
## Cross-Cutting Concerns
<!-- GSD:architecture-end -->

<!-- GSD:skills-start source:skills/ -->
## Project Skills

| Skill | Description | Path |
|-------|-------------|------|
| bmad-advanced-elicitation | 'Push the LLM to reconsider, refine, and improve its recent output. Use when user asks for deeper critique or mentions a known deeper critique method, e.g. socratic, first principles, pre-mortem, red team.' | `.claude/skills/bmad-advanced-elicitation/SKILL.md` |
| bmad-brainstorming | 'Facilitate interactive brainstorming sessions using diverse creative techniques and ideation methods. Use when the user says help me brainstorm or help me ideate.' | `.claude/skills/bmad-brainstorming/SKILL.md` |
| bmad-distillator | Lossless LLM-optimized compression of source documents. Use when the user requests to 'distill documents' or 'create a distillate'. | `.claude/skills/bmad-distillator/SKILL.md` |
| bmad-editorial-review-prose | 'Clinical copy-editor that reviews text for communication issues. Use when user says review for prose or improve the prose' | `.claude/skills/bmad-editorial-review-prose/SKILL.md` |
| bmad-editorial-review-structure | 'Structural editor that proposes cuts, reorganization, and simplification while preserving comprehension. Use when user requests structural review or editorial review of structure' | `.claude/skills/bmad-editorial-review-structure/SKILL.md` |
| bmad-help | 'Analyzes current state and user query to answer BMad questions or recommend the next skill(s) to use. Use when user asks for help, bmad help, what to do next, or what to start with in BMad.' | `.claude/skills/bmad-help/SKILL.md` |
| bmad-index-docs | 'Generates or updates an index.md to reference all docs in the folder. Use if user requests to create or update an index of all files in a specific folder' | `.claude/skills/bmad-index-docs/SKILL.md` |
| bmad-party-mode | 'Orchestrates group discussions between installed BMAD agents, enabling natural multi-agent conversations where each agent is a real subagent with independent thinking. Use when user requests party mode, wants multiple agent perspectives, group discussion, roundtable, or multi-agent conversation about their project.' | `.claude/skills/bmad-party-mode/SKILL.md` |
| bmad-review-adversarial-general | 'Perform a Cynical Review and produce a findings report. Use when the user requests a critical review of something' | `.claude/skills/bmad-review-adversarial-general/SKILL.md` |
| bmad-review-edge-case-hunter | 'Walk every branching path and boundary condition in content, report only unhandled edge cases. Orthogonal to adversarial review - method-driven not attitude-driven. Use when you need exhaustive edge-case analysis of code, specs, or diffs.' | `.claude/skills/bmad-review-edge-case-hunter/SKILL.md` |
| bmad-shard-doc | 'Splits large markdown documents into smaller, organized files based on level 2 (default) sections. Use if the user says perform shard document' | `.claude/skills/bmad-shard-doc/SKILL.md` |
<!-- GSD:skills-end -->

<!-- GSD:workflow-start source:GSD defaults -->
## GSD Workflow Enforcement

Before using Edit, Write, or other file-changing tools, start work through a GSD command so planning artifacts and execution context stay in sync.

Use these entry points:
- `/gsd-quick` for small fixes, doc updates, and ad-hoc tasks
- `/gsd-debug` for investigation and bug fixing
- `/gsd-execute-phase` for planned phase work

Do not make direct repo edits outside a GSD workflow unless the user explicitly asks to bypass it.
<!-- GSD:workflow-end -->



<!-- GSD:profile-start -->
## Developer Profile

> Profile not yet configured. Run `/gsd-profile-user` to generate your developer profile.
> This section is managed by `generate-claude-profile` -- do not edit manually.
<!-- GSD:profile-end -->
