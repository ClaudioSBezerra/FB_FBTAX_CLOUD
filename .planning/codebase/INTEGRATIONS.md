# External Integrations

**Analysis Date:** 2026-05-19

## APIs & External Services

**Oracle ERP (ERP Bridge):**
- Oracle Database (SAP S/4HANA via FCCORP schema, or legacy per-branch Oracle XML) — source of NF-e/CT-e fiscal documents
  - SDK/Client: `python-oracledb` thin mode (no Instant Client required) in `erp-bridge-aws/bridge.py`
  - Auth: `oracle.usuario` / `oracle.senha` / `oracle.dsn` in `erp-bridge-aws/config.yaml`
  - Two modes: `oracle_xml` (per-branch, legacy Totvs/Protheus) and `sap_s4hana` (centralized SAP FCCORP schema using `s4i_nfe` + `s4i_nfe_impostos` tables)

**Receita Federal do Brasil (RFB):**
- Brazilian Federal Tax Authority API — CBS/IBS data fetch and fiscal debt queries
  - Endpoint: `RFB_API_URL` (template: `https://api.receitafederal.gov.br`)
  - OAuth token endpoint: `RFB_TOKEN_URL` (`https://api.receitafederal.gov.br/token`)
  - Credentials stored per-company in `rfb_credentials` table (migration `051_create_rfb_credentials.sql`): `client_id`, `client_secret`, `cnpj_matriz`
  - Scheduled import: `agendamento_ativo` + `horario_agendamento` columns in `rfb_credentials` (migration `069_rfb_agendamento.sql`)
  - RFB data stored in: `rfb_debitos`, `rfb_requests`, `rfb_resumo` tables

**Z.AI GLM API (AI Executive Reports):**
- AI-generated executive fiscal summaries
  - API key: `ZAI_API_KEY` env var (see `coolify-env-template.txt`)
  - Reports stored in: `ai_reports` table (migration `048_create_ai_reports_table.sql`)
  - Schema comment on `ai_reports` references "Claude" as the AI, but env template references `ZAI_API_KEY` (GLM API); actual API call code not found in current handler files — may exist in a product-specific service not yet committed
  - Frontend references Claude/Anthropic (Claude sun logo SVG in `frontend/src/pages/PortalPage.tsx`)

**Coolify PaaS:**
- Self-hosted deployment platform triggering redeploy via webhook
  - Webhook: `http://76.13.171.196:8000/api/v1/deploy?uuid=u4k84840408sc4wgogcw4c04` (triggered from GitHub Actions `deploy-production.yml`)

## Data Storage

**Databases:**
- PostgreSQL 15 (primary)
  - Connection: `DATABASE_URL` env var (format: `postgres://user:pass@host:5432/dbname?sslmode=disable`)
  - Client: `github.com/lib/pq` — raw SQL, no ORM
  - In Docker: service named `db`, data persisted in `postgres_data` Docker volume
  - 107+ SQL migrations in `backend/migrations/`

- SQLite (ERP Bridge only)
  - File: `erp-bridge-aws/tracker.db`
  - Purpose: deduplication tracker (tracks already-sent documents) and SAP watermark (last successful import date)
  - Client: Python stdlib `sqlite3`

**File Storage:**
- Local filesystem only — uploaded files stored in `/app/uploads` inside the container
- No S3 or cloud object storage detected

**Caching:**
- Redis referenced in `coolify-env-template.txt` (`REDIS_HOST`, `REDIS_ADDR`) but not found in backend Go code — not currently implemented
- In-memory token stores only: `refreshTokenStore` and `tokenBlacklist` (`sync.Map` in `backend/handlers/auth.go`)

## Authentication & Identity

**Auth Provider:**
- Custom JWT-based auth — no OAuth providers or third-party auth services
  - Implementation: `backend/handlers/auth.go`
  - Access tokens: JWT signed with `JWT_SECRET` env var (`github.com/golang-jwt/jwt/v5`)
  - Refresh tokens: stored in-memory `sync.Map` (not database-persisted — lost on restart)
  - Password hashing: bcrypt via `golang.org/x/crypto/bcrypt`
  - Password reset: email token flow, tokens in `verification_tokens` table
  - Tenant isolation: `X-Company-ID` header injected by frontend auth interceptor (`frontend/src/contexts/AuthContext.tsx`)

**ERP Bridge Auth:**
- FBTax API login: email/password → JWT Bearer token (standard auth flow)
- Alternative: `X-API-Key` header for daemon/batch endpoints (`/api/erp-bridge/import/batch`, `/api/erp-bridge/parceiros/sync`, `/api/erp-bridge/heartbeat`)
- Credentials can be stored encrypted in DB and fetched by daemon at startup via `GET /api/erp-bridge/credentials` (migration `079_erp_bridge_credentials.sql` — AES-256-GCM encrypted fields)

## Monitoring & Observability

**Error Tracking:**
- None detected (no Sentry, Datadog, etc.)

**Logs:**
- Backend: Go stdlib `log` package, writes to stdout/stderr (captured by Docker/systemd)
- ERP Bridge: Python `logging` module, dual output — rotating file logs in `erp-bridge-aws/logs/bridge_YYYYMMDD_HHMMSS.log` and stdout
- Health endpoint: `GET /api/health` returns JSON with DB connection status and connection pool stats

## CI/CD & Deployment

**Hosting:**
- Production: Coolify (self-hosted) at `fbtax.cloud` / `www.fbtax.cloud`
- Staging: Azure VM (SSH-based deploy), triggered by `v*-rc*` tags
- Client installs: AWS EC2 via `installer/cliente-aws/setup.sh` (pulls images from GHCR)

**CI Pipeline:**
- GitHub Actions (`.github/workflows/`)
  - `deploy-production.yml` — triggers on push to `main` or `v*` tags; builds Docker image, pushes to GHCR (`ghcr.io/claudiosbezerra/...`), triggers Coolify webhook
  - `deploy-staging.yml` — triggers on `v*-rc*` tags; builds separate api/web images, deploys to Azure via SSH
  - `deploy-cliente-aws.yml` — separate workflow for client AWS deployment

**Container Registry:**
- GitHub Container Registry (GHCR) — `ghcr.io/claudiosbezerra/fb_fbtax_cloud`
- Auth: `GITHUB_TOKEN` secret (automatic in Actions)

## Webhooks & Callbacks

**Incoming:**
- `GET /api/rfb/webhook` — referenced as `RFB_WEBHOOK_URL` in env template (`https://fbtax.cloud/api/rfb/webhook`); handler not found in current `backend/handlers/` files — likely planned or in another product module

**Outgoing:**
- Coolify redeploy webhook: `http://76.13.171.196:8000/api/v1/deploy?uuid=...` called from GitHub Actions

## Email

**Provider:** SMTP (Hostinger by default — `smtp.hostinger.com:465`)
  - Implementation: `backend/services/email.go`
  - Auth: `SMTP_USER` / `SMTP_PASSWORD` env vars
  - Transport: Implicit TLS (port 465) via `tls.Dial`, or STARTTLS fallback for other ports
  - Use cases:
    1. Password reset emails (`SendPasswordResetEmail`)
    2. AI-generated executive report emails with full fiscal KPI layout (`SendAIReportEmail`)
  - From address: `SMTP_FROM` env var (default: `FBTax Cloud <noreply@fbtax.cloud>`)

## Environment Configuration

**Required env vars (backend):**
- `DATABASE_URL` — PostgreSQL connection string
- `JWT_SECRET` — JWT signing secret (must be set; validated at startup via `handlers.ValidateJWTSecret()`)
- `SMTP_HOST` / `SMTP_PORT` / `SMTP_USER` / `SMTP_PASSWORD` / `SMTP_FROM` — email delivery
- `ALLOWED_ORIGINS` — comma-separated CORS origins (defaults to `https://www.fbtax.cloud`)
- `PORT` — HTTP listen port (defaults to `8083`)
- `APP_URL` — base URL for password reset links (defaults to `https://fbtax.cloud`)

**Optional env vars (backend):**
- `ZAI_API_KEY` — AI report generation
- `RFB_API_URL` / `RFB_TOKEN_URL` — Receita Federal API
- `ADMIN_EMAIL` — initial admin user seeding

**ERP Bridge config (`erp-bridge-aws/config.yaml`):**
- `erp_type` — `oracle_xml` or `sap_s4hana`
- `fbtax.url` / `fbtax.email` / `fbtax.password` / `fbtax.company_id` — FBTax API connection
- `fbtax.api_key` — optional; if set, daemon fetches encrypted credentials from server
- `oracle.dsn` / `oracle.usuario` / `oracle.senha` — Oracle DB connection (sap_s4hana mode)
- `servidores[]` — list of per-branch Oracle connections (oracle_xml mode)

**Secrets location:**
- Production secrets in Coolify environment variables panel
- `erp-bridge-aws/config.yaml` on client AWS EC2 (sensitive — contains credentials)
- GitHub Secrets: `STAGING_SSH_KEY`, `STAGING_HOST`, `STAGING_USER`

---

*Integration audit: 2026-05-19*
