# Codebase Structure

**Analysis Date:** 2026-05-19

## Directory Layout

```
FB_FBTAX_CLOUD/
├── backend/                   # Go REST API — the entire server-side service
│   ├── main.go                # Entry point, route registration, migration runner
│   ├── go.mod                 # Module: fb_cloud, Go 1.26
│   ├── go.sum
│   ├── Dockerfile             # Backend-only Docker image (dev)
│   ├── handlers/              # HTTP handlers — one package, no sub-packages
│   │   ├── auth.go            # Login, logout, register, password reset, JWT
│   │   ├── admin.go           # User management (create, list, promote, delete)
│   │   ├── environment.go     # Environments, groups, companies CRUD
│   │   ├── hierarchy.go       # User's env/group/company/branches resolution
│   │   ├── middleware.go      # SecurityMiddleware, CORS, rate limiters, GetClientIP
│   │   └── portal_products.go # Public portal product catalog
│   ├── services/
│   │   └── email.go           # SMTP sending, HTML templates (password reset, AI reports)
│   ├── migrations/            # Sequential SQL migrations, auto-applied on startup
│   │   ├── 001_create_jobs_table.sql
│   │   ├── ...
│   │   └── 107_pt_farol_description.sql   # 107 files total
│   └── tools/                 # Build-tagged one-off debug scripts (not compiled in prod)
│       ├── debug_detailed.go
│       ├── debug_gilson.go
│       ├── debug_query.go
│       ├── debug_stats.go
│       └── verify_data.go
│
├── frontend/                  # React 18 + TypeScript SPA
│   ├── src/
│   │   ├── main.tsx           # React DOM root mount
│   │   ├── App.tsx            # Router, AuthProvider, route definitions
│   │   ├── index.css          # Global Tailwind styles
│   │   ├── vite-env.d.ts
│   │   ├── pages/             # Route-level page components
│   │   │   ├── Login.tsx
│   │   │   ├── PortalPage.tsx
│   │   │   ├── ForgotPassword.tsx
│   │   │   └── ResetPassword.tsx
│   │   ├── components/
│   │   │   ├── ProductCard.tsx   # Portal product display card
│   │   │   ├── Footer.tsx
│   │   │   └── ui/              # shadcn/ui components (Radix + Tailwind wrappers)
│   │   │       └── *.tsx        # button, card, dialog, input, table, etc. (~45 files)
│   │   ├── contexts/
│   │   │   └── AuthContext.tsx  # JWT state, company context, localStorage persistence
│   │   ├── hooks/
│   │   │   └── use-mobile.tsx   # Responsive breakpoint hook
│   │   └── lib/
│   │       ├── utils.ts         # Tailwind cn() utility
│   │       └── utils.test.ts    # Unit test for utils
│   ├── public/                  # Static assets (logos, images)
│   ├── package.json
│   ├── vite.config.ts           # Vite config; dev proxy /api → :8086
│   ├── tailwind.config.js
│   ├── tsconfig.app.json
│   ├── Dockerfile               # Multi-stage: Node build → Nginx serve
│   ├── Dockerfile.dev           # Dev server image
│   └── nginx.conf               # Nginx config — SPA fallback only (no /api proxy)
│
├── erp-bridge-aws/            # Python ERP integration daemon (runs on customer AWS VM)
│   ├── bridge.py              # Full daemon — Oracle queries, FBTaxClient, SAP+XML modes
│   ├── config.yaml            # Customer-specific credentials and server list
│   └── erp-bridge.service     # systemd service unit file
│
├── installer/                 # Installer kit for on-premise / client deployments
│   ├── install.sh             # Main installer script
│   ├── update.sh              # Update script
│   ├── docker-compose.yml     # Installer stack definition
│   ├── .env.template          # Env template for customer
│   ├── QUESTIONARIO_CLIENTE.md
│   ├── README.md
│   └── cliente-aws/           # Variant for customer-side AWS deployment
│       ├── docker-compose.yml # Uses GHCR images (fb_apu02-api, fb_apu02-web) + Redis
│       ├── setup.sh
│       └── .env.template
│
├── scripts/                   # Operational shell/bat scripts
│   ├── backup_production.sh
│   ├── check_materialized_views.sh
│   ├── deploy_production.sh
│   ├── promote_to_prd.bat
│   ├── transport_to_qa.bat
│   └── tunnel-prod-db.sh
│
├── scripts_oracle/            # SQL scripts for Oracle ERP data extraction
│   ├── gera_xmls_nfe.sql
│   ├── gera_xmls_nfse_saidas*.sql
│   ├── importa_cte_entrada.sql
│   └── importa_xmls_entrada.sql
│
├── config/                    # Shared application config
│   └── app.yaml               # Server port, DB pool, Redis, import rules, feature flags
│
├── tests/                     # Integration tests (currently empty placeholder)
│   └── integration_test.go
│
├── docs/                      # Technical documentation
│   └── planning/              # Planning documents
│
├── _bmad/                     # BMAD methodology config and agents (planning tooling)
│   ├── _config/agents/        # BMAD agent definitions
│   └── core/                  # BMAD core resources
│
├── _bmad-output/              # BMAD planning and implementation artifacts
│   ├── planning-artifacts/    # PRD, epics, architecture.md, product-brief
│   └── implementation-artifacts/ # Per-story implementation guides
│
├── .planning/                 # GSD planning documents (this toolchain)
│   └── codebase/              # Codebase map documents
│
├── .claude/                   # Claude agent skills directory
│   └── skills/                # BMAD skill subdirectories
│
├── .github/
│   └── workflows/
│       ├── deploy-production.yml  # Push to main → build GHCR image → trigger Coolify
│       ├── deploy-staging.yml
│       └── deploy-cliente-aws.yml
│
├── docker-compose.yml         # Primary dev/staging compose (Traefik labels, port 8086)
├── docker-compose.prod.yml    # Production-only compose (port 8083, no Traefik labels on api)
├── Dockerfile.production      # 3-stage unified build: frontend + backend + Alpine runtime
├── .env.example               # Environment variable reference (never commit real values)
├── coolify-env-template.txt   # Coolify-specific environment configuration template
├── VERSIONAMENTO.md           # Versioning strategy (Portuguese)
├── VERSIONAMENTO_AUTO.md      # Automated versioning documentation (Portuguese)
├── VALIDATION_REPORT_20260204.md
├── README.md
├── start_dev.bat              # Windows dev startup
├── start_docker.bat           # Windows Docker startup
├── setup_wsl_env.sh           # WSL environment setup
├── cleanup_old_branches.sh
├── delete_branches_with_token.sh
└── git_push_helper.sh
```

## Directory Purposes

**`backend/`:**
- Purpose: Complete Go server — HTTP API, DB migrations, static file server
- Contains: Single Go package `main` + `handlers` package + `services` package
- Key files: `main.go` (routes + migration runner), `handlers/auth.go` (largest handler file)

**`backend/handlers/`:**
- Purpose: All HTTP handler functions — single flat package, no sub-grouping
- Contains: 6 files mapping roughly to domain areas (auth, admin, environment, hierarchy, middleware, portal_products)
- Key files: `auth.go` (1250+ lines, JWT, password flow), `middleware.go` (CORS, security headers, rate limiters)

**`backend/migrations/`:**
- Purpose: SQL migration scripts executed sequentially at backend startup
- Contains: 107 numbered SQL files; some are `.sql.disabled` (skipped); migrations cover public schema and `portal` schema
- Key files: `013_create_environment_hierarchy.sql` (tenant model), `015_create_auth_system.sql` (users/tokens), `074_erp_bridge.sql` (bridge run tracking), `100_pt_products.sql` (portal schema)

**`backend/services/`:**
- Purpose: Business logic callable from multiple handlers
- Contains: `email.go` — SMTP configuration, HTML email rendering for password reset and AI report emails
- Key files: `email.go` (single large file, 36k characters)

**`backend/tools/`:**
- Purpose: One-off debug/admin scripts with `//go:build scripts` build tag — NOT compiled into the main binary
- Contains: Debug queries and data verification scripts
- Generated: No. Committed: Yes (developer utilities)

**`frontend/src/pages/`:**
- Purpose: Route-level React components; each file corresponds to a URL path
- Contains: `Login.tsx`, `PortalPage.tsx`, `ForgotPassword.tsx`, `ResetPassword.tsx`

**`frontend/src/components/ui/`:**
- Purpose: shadcn/ui component library — Radix UI primitives wrapped with Tailwind styling
- Contains: ~45 component files; do not modify these manually — regenerate via shadcn CLI

**`frontend/src/contexts/`:**
- Purpose: React Context providers for cross-component state
- Contains: `AuthContext.tsx` — JWT token, user profile, environment/group/company selection

**`erp-bridge-aws/`:**
- Purpose: Python 3 daemon that bridges customer Oracle ERP databases to the FBTAX API
- Contains: Single large `bridge.py` file (1000+ lines), YAML config, systemd service definition
- Not containerized in the main compose; deployed independently on customer AWS VMs

**`installer/`:**
- Purpose: Turn-key install kit for deploying the `fb_apu02` product on customer infrastructure
- Contains: Shell scripts, Docker Compose file, environment template, customer questionnaire
- Note: References a different GHCR image (`fb_apu02-api`) — this is for a different product variant

**`_bmad/` and `_bmad-output/`:**
- Purpose: BMAD (AI-assisted planning) methodology artifacts — PRD, epics, implementation stories
- Contains: Planning documents consumed during feature design, not runtime code
- Generated: Partially (by BMAD agents). Committed: Yes

**`config/`:**
- Purpose: Static application configuration file (`app.yaml`) — server, DB pool, import rules, feature flags
- Note: Redis is configured here but not present in the main `docker-compose.yml` — treat as aspirational config

## Key File Locations

**Entry Points:**
- `backend/main.go`: Go server entry — `func main()` at line 259
- `frontend/src/main.tsx`: React entry — `ReactDOM.createRoot`
- `erp-bridge-aws/bridge.py`: ERP bridge entry — `if __name__ == "__main__":`

**Configuration:**
- `docker-compose.yml`: Primary compose, Traefik labels, dev ports
- `docker-compose.prod.yml`: Production compose
- `Dockerfile.production`: Unified production image (frontend + backend)
- `frontend/vite.config.ts`: Vite dev server proxy configuration
- `frontend/nginx.conf`: Nginx SPA serving (production)
- `config/app.yaml`: Application-level settings
- `.env.example`: All required environment variables listed
- `erp-bridge-aws/config.yaml`: Customer-specific Oracle + FBTax credentials

**Core Logic:**
- `backend/handlers/auth.go`: Authentication flows (login, register, password reset, JWT)
- `backend/handlers/middleware.go`: CORS, security headers, rate limiting
- `backend/handlers/environment.go`: Tenant hierarchy management (environments/groups/companies)
- `backend/handlers/portal_products.go`: Portal product catalog serving
- `backend/services/email.go`: Email service
- `frontend/src/contexts/AuthContext.tsx`: Frontend auth state management
- `frontend/src/pages/PortalPage.tsx`: Public portal landing page
- `erp-bridge-aws/bridge.py`: Complete ERP sync logic

**Testing:**
- `frontend/src/lib/utils.test.ts`: Only existing test file (unit test for Tailwind `cn()` utility)
- `tests/integration_test.go`: Placeholder — empty

**CI/CD:**
- `.github/workflows/deploy-production.yml`: Push to main → GHCR image build → Coolify trigger
- `.github/workflows/deploy-staging.yml`: Staging pipeline
- `.github/workflows/deploy-cliente-aws.yml`: Customer AWS deployment pipeline

## Naming Conventions

**Files (Backend):**
- Go handler files: domain name, lowercase, `.go` (e.g., `auth.go`, `environment.go`, `portal_products.go`)
- Migration files: `NNN_snake_case_description.sql` where NNN is a 3-digit zero-padded number (e.g., `013_create_environment_hierarchy.sql`)
- Disabled migrations: append `.disabled` suffix (e.g., `000_reset_db.sql.disabled`)

**Files (Frontend):**
- Page components: PascalCase `.tsx` (e.g., `Login.tsx`, `PortalPage.tsx`)
- UI components: kebab-case `.tsx` (e.g., `alert-dialog.tsx`, `dropdown-menu.tsx`)
- Contexts: PascalCase + `Context` suffix (e.g., `AuthContext.tsx`)
- Hooks: `use-` prefix, kebab-case (e.g., `use-mobile.tsx`)
- Utilities: `utils.ts`

**Go Functions:**
- Exported handlers: `<Verb><Domain>Handler` (e.g., `LoginHandler`, `GetEnvironmentsHandler`, `CreateUserHandler`)
- Exported middleware: `<Name>Middleware` (e.g., `SecurityMiddleware`, `AuthMiddleware`)
- Package-level helpers: camelCase (e.g., `withDB`, `withAuth`, `getDB`, `initDBAsync`)

**Database:**
- Public schema: `snake_case` tables (e.g., `import_jobs`, `enterprise_groups`, `user_environments`)
- Portal schema: `pt_` prefix in `portal` schema (e.g., `portal.pt_products`, `portal.pt_tenants`)
- ERP bridge tables: `erp_bridge_` prefix (e.g., `erp_bridge_config`, `erp_bridge_runs`)
- Indexes: `idx_<table>_<column>` convention (e.g., `idx_users_email`, `idx_pt_tenants_slug`)

## Where to Add New Code

**New API Endpoint:**
1. Implement handler function in `backend/handlers/<domain>.go` (or create new file if domain is new)
2. Register route in `backend/main.go` using `withDB(handlers.YourHandler)` or `withAuth(handlers.YourHandler, "role")`
3. Follow existing pattern: handler func signature `func YourHandler(db *sql.DB) http.HandlerFunc`

**New Database Table or Schema Change:**
1. Create `backend/migrations/<NNN>_<description>.sql` where NNN follows the highest existing number
2. Write idempotent SQL using `CREATE TABLE IF NOT EXISTS`, `CREATE INDEX IF NOT EXISTS`
3. No manual migration execution needed — backend applies it automatically at next startup

**New Frontend Page:**
1. Create `frontend/src/pages/YourPage.tsx`
2. Add route in `frontend/src/App.tsx` using `<Route path="..." element={<YourPage />} />`
3. Use `useAuth()` from `frontend/src/contexts/AuthContext.tsx` for auth state
4. Wrap in `<ProtectedRoute>` if authentication is required

**New Shared UI Component:**
- Custom components: `frontend/src/components/<ComponentName>.tsx`
- shadcn/ui primitives: `frontend/src/components/ui/<component-name>.tsx` (generate with `npx shadcn@latest add`)

**New Hook:**
- Location: `frontend/src/hooks/use-<name>.tsx`

**New Business Logic (Backend):**
- Location: `backend/services/<domain>.go`
- Called from handlers; do not import from `handlers` package (one-way dependency)

**New ERP Data Source or Query:**
- Add SQL query constant and FONTES entry in `erp-bridge-aws/bridge.py`
- Add corresponding endpoint registration in `erp-bridge-aws/bridge.py` FONTES dict

**New Operational Script:**
- Shell scripts: `scripts/<purpose>.sh`
- Oracle-specific SQL: `scripts_oracle/<purpose>.sql`

## Special Directories

**`_bmad/`:**
- Purpose: BMAD AI planning methodology configuration and agent definitions
- Generated: Partially (by Claude/BMAD agents)
- Committed: Yes — contains planning tooling configuration

**`_bmad-output/`:**
- Purpose: Output artifacts from BMAD planning sessions — PRD, epics, implementation stories, architecture design
- Generated: Yes (by BMAD agents via Claude)
- Committed: Yes — serves as project documentation and implementation guide

**`.planning/`:**
- Purpose: GSD codebase map and planning documents for this toolchain
- Generated: Yes (by GSD map-codebase command)
- Committed: Yes — consumed by `/gsd:plan-phase` and `/gsd:execute-phase`

**`.claude/skills/`:**
- Purpose: Claude agent skill definitions for BMAD and editorial workflows
- Generated: No (manually maintained)
- Committed: Yes

**`backend/tools/`:**
- Purpose: Developer debug scripts with `//go:build scripts` — excluded from normal `go build`
- Generated: No
- Committed: Yes — useful during debugging

---

*Structure analysis: 2026-05-19*
