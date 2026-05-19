<!-- refreshed: 2026-05-19 -->
# Architecture

**Analysis Date:** 2026-05-19

## System Overview

```text
┌─────────────────────────────────────────────────────────────────────────┐
│                     Internet / Browser                                  │
│                  https://www.fbtax.cloud                                │
└───────────────────────────────┬─────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                    Traefik Reverse Proxy (Coolify)                      │
│  Host: www.fbtax.cloud | fbtax.cloud                                    │
│  PathPrefix(/api/) → fbtax-cloud-api (port 8086)                        │
│  All other paths   → fbtax-cloud (port 80)                              │
└─────────────┬───────────────────────────────────┬───────────────────────┘
              │                                   │
              ▼                                   ▼
┌─────────────────────────┐          ┌────────────────────────────────────┐
│  Go Backend (api)       │          │  React/Nginx Frontend (web)        │
│  `backend/`             │          │  `frontend/`                       │
│  port 8086 (dev/prod)   │◄─────────│  nginx:80 — SPA only               │
│  port 8083 (prod.yml)   │  fetch   │  Vite proxy → :8086 in dev        │
│  `backend/main.go`      │  /api/*  │  `frontend/src/main.tsx`           │
└───────────┬─────────────┘          └────────────────────────────────────┘
            │ sql
            ▼
┌─────────────────────────┐
│  PostgreSQL 15 (db)     │
│  postgres_data volume   │
│  public schema          │
│  portal schema          │
└─────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────┐
│  ERP Bridge (external process — NOT a container in main compose)        │
│  `erp-bridge-aws/bridge.py`  — runs as Linux daemon on customer AWS VM │
│                                                                         │
│  Oracle DB ──────────────────► FBTaxClient.enviar_batch()              │
│  (SAP S/4HANA FCCORP)           POST /api/erp-bridge/import/batch      │
│  or Oracle XML (legado)         X-API-Key header                        │
│                                 → Go backend                            │
└─────────────────────────────────────────────────────────────────────────┘
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

**Overall:** Monolith-with-satellite. A single Go binary serves the entire backend — no internal microservices. A separate Python process (`erp-bridge-aws`) operates as an optional external satellite that pushes data to the backend via HTTP.

**Key Characteristics:**
- Single Go binary handles API, migration runner, and static file serving (SPA fallback)
- No router framework — `net/http` standard library `http.HandleFunc` only, routes registered in `backend/main.go`
- DB injection pattern: every route is `withDB(handlers.SomeHandler)` or `withAuth(handlers.SomeHandler, role)` — closures inject `*sql.DB`
- Migration auto-apply on startup: `onDBConnected()` in `backend/main.go` scans `migrations/*.sql` and executes unapplied files tracked in `schema_migrations`
- Multi-tenant data model: `environments` → `enterprise_groups` → `companies` → users/data
- Portal schema (`portal.*`) is a separate PostgreSQL schema for the public product catalog and tenant subscriptions
- ERP Bridge is deployed independently on customer-side AWS, authenticates with `X-API-Key` header
- Frontend in dev proxies `/api/*` via Vite to Go backend; in production Traefik routes `/api/*` directly to the Go container and all other traffic to the Nginx container

## Layers

**Routing Layer:**
- Purpose: Route HTTP requests, inject DB, enforce auth
- Location: `backend/main.go` (lines 344–381)
- Contains: `http.HandleFunc` registrations, `withDB` and `withAuth` closures, static file server
- Depends on: `handlers` package
- Used by: Traefik (production), Vite dev proxy (development)

**Handlers Layer:**
- Purpose: HTTP request parsing, validation, DB interaction, JSON responses
- Location: `backend/handlers/`
- Contains: `auth.go`, `admin.go`, `environment.go`, `hierarchy.go`, `middleware.go`, `portal_products.go`
- Depends on: `database/sql`, `services` package
- Used by: routing layer via `withDB` / `withAuth` closures

**Services Layer:**
- Purpose: Business logic reused across handlers
- Location: `backend/services/`
- Contains: `email.go` — SMTP sending, HTML email templates for password reset, AI reports
- Depends on: OS environment variables (`SMTP_*`)
- Used by: `handlers/auth.go` (password reset emails), future report handlers

**Migrations Layer:**
- Purpose: Database schema versioning, applied automatically at startup
- Location: `backend/migrations/*.sql`
- Contains: 107 numbered SQL files (000 through 107)
- Depends on: `schema_migrations` table in PostgreSQL
- Used by: `onDBConnected()` in `backend/main.go`

**Frontend Layer:**
- Purpose: React SPA — public portal and admin interface
- Location: `frontend/src/`
- Contains: Pages, components, contexts, hooks
- Depends on: Go backend via `/api/*` fetch calls
- Used by: Nginx container (production), Vite dev server (development)

**ERP Bridge Layer:**
- Purpose: Extract fiscal data from Oracle ERP and push to FBTAX backend
- Location: `erp-bridge-aws/bridge.py`
- Contains: Oracle query logic, XML normalization, SAP S/4HANA batch mode, FBTaxClient HTTP wrapper, SQLite tracker
- Depends on: Go backend `/api/erp-bridge/*` endpoints, Oracle DB
- Used by: Customer AWS VMs running the bridge as a Linux daemon (`erp-bridge.service`)

## Data Flow

### Portal Page (Public)

1. Browser loads React SPA from Nginx — `frontend/src/main.tsx`
2. `PortalPage.tsx` mounts; `useQuery` fires `GET /api/portal/products?tenant=<slug>`
3. Traefik routes `/api/*` → Go backend container
4. `GetPortalProductsHandler` (`backend/handlers/portal_products.go`) queries `portal.pt_products` LEFT JOIN `portal.pt_tenant_products`
5. JSON returned, products rendered as `ProductCard` components

### Authenticated Admin Login

1. User submits credentials in `frontend/src/pages/Login.tsx`
2. `POST /api/auth/login` → `LoginHandler` (`backend/handlers/auth.go:599`)
3. Handler bcrypt-verifies password, queries `users` + `user_environments` + `enterprise_groups` + `companies`
4. Returns JWT access token + refresh token cookie + company context (`environment_name`, `group_name`, `company_name`)
5. `AuthContext.login()` persists to `localStorage`; `window.fetch` interceptor auto-injects `Authorization: Bearer <token>` and `X-Company-ID: <id>` on all subsequent requests

### ERP Bridge Import (SAP Mode)

1. Bridge daemon on customer AWS VM reads `config.yaml` for Oracle DSN + FBTax API key
2. Calls `fbtax.login()` → `POST /api/auth/login` with service credentials
3. Creates a run record via `POST /api/erp-bridge/runs`
4. Executes `SAP_QUERY` against Oracle FCCORP (`s4i_nfe`, `s4i_nfe_impostos`)
5. Syncs trading partners via `POST /api/erp-bridge/parceiros/sync` (X-API-Key)
6. Sends document batch via `POST /api/erp-bridge/import/batch` (X-API-Key)
7. Reports item stats via `POST /api/erp-bridge/runs/{id}/items`
8. Finalizes run via `PATCH /api/erp-bridge/runs/{id}` with status/totals
9. SQLite `tracker.db` (`erp-bridge-aws/tracker.db`) prevents duplicate sends

### ERP Bridge Import (Oracle XML Legacy Mode)

1. Same bridge, `erp_type: oracle_xml` in `config.yaml`
2. Iterates per-server entries from `servidores` list
3. Per document type (`nfe_saidas`, `nfe_entradas`, `cte_entradas`): queries XML CLOB from Oracle table, normalizes encoding, POSTs to respective endpoint (`/api/nfe-saidas/upload`, `/api/nfe-entradas/upload`, `/api/cte-entradas/upload`)

### DB Migration on Startup

1. `initDBAsync()` spawns goroutine — `backend/main.go:112`
2. Retries `sql.Open` + `Ping` every 5 seconds until PostgreSQL responds
3. On first connection: `ensureDatabaseExists()` creates target DB if missing (connects to `postgres` admin DB)
4. `onDBConnected()` — `backend/main.go:161` — scans `backend/migrations/*.sql` alphabetically
5. Files not in `schema_migrations` table are executed in order; failures on `already exists` are skipped with a soft-insert

**State Management (Frontend):**
- JWT token stored in `localStorage`; company/environment context also in `localStorage`
- `AuthContext` (`frontend/src/contexts/AuthContext.tsx`) is the single source of truth
- `window.fetch` is monkey-patched by `AuthContext` to inject `Authorization` and `X-Company-ID` headers globally

## Key Abstractions

**`withDB` / `withAuth` closures (Route wrappers):**
- Purpose: Inject `*sql.DB` into handlers and optionally enforce JWT authentication
- Location: `backend/main.go:322–342`
- Pattern: Higher-order functions returning `http.HandlerFunc`, avoiding global state for DB

**`AuthMiddleware`:**
- Purpose: Parse and validate JWT from `Authorization: Bearer` header; store claims in `context.Value(ClaimsKey)`
- Location: `backend/handlers/auth.go` (exported as `handlers.AuthMiddleware`)
- Pattern: Standard Go middleware wrapping `http.HandlerFunc`

**`SecurityMiddleware`:**
- Purpose: CORS enforcement, CSP headers, X-Frame-Options, HSTS, preflight handling
- Location: `backend/handlers/middleware.go:90`
- Pattern: Wraps the entire `http.DefaultServeMux`; uses `secureResponseWriter` to intercept and override headers

**Environment Hierarchy (Tenant Model):**
- Purpose: Multi-tenant data isolation
- Tables: `environments` → `enterprise_groups` → `companies` → `users` (via `user_environments`)
- Examples: `backend/migrations/013_create_environment_hierarchy.sql`, `backend/handlers/environment.go`
- Pattern: UUID primary keys, `ON DELETE CASCADE` throughout

**Portal Schema (`portal.*`):**
- Purpose: Public product catalog and tenant subscriptions for `fbtax.cloud` portal page
- Tables: `portal.pt_products`, `portal.pt_tenants`, `portal.pt_tenant_products`, `portal.pt_notifications`, `portal.pt_events`, `portal.pt_product_urls`
- Location: `backend/migrations/100_pt_products.sql` through `107_pt_farol_description.sql`
- Pattern: Separate PostgreSQL schema; queries use `portal.pt_*` fully qualified names

**ERP Bridge `FBTaxClient`:**
- Purpose: HTTP client wrapper that handles authentication, token refresh, and all FBTax API calls
- Location: `erp-bridge-aws/bridge.py:291`
- Pattern: Session-based with auto-retry on 401; supports both user credentials (login flow) and `X-API-Key` (batch endpoints)

## Entry Points

**Go Backend:**
- Location: `backend/main.go:259` — `func main()`
- Triggers: Docker CMD (`./fb_fbtax_cloud`), or `go run main.go` locally
- Responsibilities: Load `.env`, validate JWT secret, start async DB init, register all HTTP routes, start HTTP server

**React Frontend:**
- Location: `frontend/src/main.tsx`
- Triggers: Loaded by Nginx (production) or Vite dev server
- Responsibilities: Mount React root, wrap in `QueryClientProvider`, delegate to `App.tsx`

**App Routing:**
- Location: `frontend/src/App.tsx`
- Responsibilities: Define routes — `/` → `PortalPage`, `/admin/login` → `Login`, `/admin` → protected dashboard stub, `*` → redirect to `/`

**ERP Bridge:**
- Location: `erp-bridge-aws/bridge.py` — `if __name__ == "__main__":` block (end of file)
- Triggers: CLI invocation or `erp-bridge.service` systemd unit (`erp-bridge-aws/erp-bridge.service`)
- Responsibilities: Parse args (`--data`, `--mes`, `--daemon`, `--servidor`), load `config.yaml`, run import cycle

**Docker Compose (development):**
- Location: `docker-compose.yml`
- Services: `api` (Go), `web` (React/Nginx), `db` (PostgreSQL 15), `db-init` (one-shot init container)

**Docker Compose (production variant):**
- Location: `docker-compose.prod.yml`
- Services: same minus `db-init`; uses port 8083 instead of 8086

**Unified Production Image:**
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

**What happens:** Many handlers in `backend/handlers/` (e.g., `CreateUserHandler`, `ListUsersHandler`, `GetEnvironmentsHandler`, `GetUserHierarchyHandler`) are implemented but have no corresponding `http.HandleFunc` in `backend/main.go`.
**Why it's wrong:** The handlers are dead code from the client's perspective — they cannot be called.
**Do this instead:** Register each handler in `backend/main.go` with the appropriate `withDB` or `withAuth` wrapper. See the existing auth routes at `backend/main.go:345–351` as the model.

### In-Memory Token State

**What happens:** `refreshTokenStore` and `tokenBlacklist` are `sync.Map` instances in `backend/handlers/auth.go:68–71` — process-local memory.
**Why it's wrong:** Restarting the container invalidates all existing refresh tokens and clears the logout blacklist, forcing all users to re-login.
**Do this instead:** Persist refresh tokens and blacklist entries to the `verification_tokens` table (already exists in `backend/migrations/015_create_auth_system.sql`) or a dedicated `refresh_tokens` table.

### Monkey-Patched `window.fetch`

**What happens:** `AuthContext` replaces `window.fetch` with a wrapper that injects auth headers (`frontend/src/contexts/AuthContext.tsx:48–60`).
**Why it's wrong:** Breaks third-party libraries that use `fetch` directly; side effects are invisible to callers; cleanup on unmount is unreliable.
**Do this instead:** Use a centralized API client module (e.g., `src/lib/api.ts`) that wraps `fetch` and accepts token/companyId as arguments, injected from context only at call site.

## Error Handling

**Strategy:** HTTP status codes returned directly from handlers with `http.Error()`; no centralized error type or wrapper.

**Patterns:**
- DB errors return `http.StatusInternalServerError` with a plain text message
- Auth failures return `http.StatusUnauthorized` or `http.StatusForbidden`
- Validation failures return `http.StatusBadRequest`
- ERP Bridge logs errors to file (`erp-bridge-aws/logs/`) and continues processing remaining items; run status is set to `partial` or `error` in `erp_bridge_runs`

## Cross-Cutting Concerns

**Logging:** `log.Printf` / `fmt.Printf` to stdout; ERP Bridge uses `logging` module writing to `erp-bridge-aws/logs/bridge_<timestamp>.log`
**Validation:** Per-handler; no shared validation layer
**Authentication:** JWT HS256 (`github.com/golang-jwt/jwt/v5`); secret from `JWT_SECRET` env var; `handlers.AuthMiddleware` validates and injects claims into request context
**Rate Limiting:** In-memory per-IP sliding window; `LoginRL` (5 req/15 min), `RegisterRL` (10 req/hr), `ForgotPasswordRL` (3 req/hr) — defined in `backend/handlers/middleware.go:137–140`
**CORS:** Origin whitelist from `ALLOWED_ORIGINS` env var; enforced by `SecurityMiddleware` wrapping entire mux

---

*Architecture analysis: 2026-05-19*
