# Technology Stack

**Analysis Date:** 2026-05-19

## Languages

**Primary:**
- Go 1.26 — Backend API server (`backend/main.go`, `backend/handlers/`, `backend/services/`)
- TypeScript 5.2 — Frontend SPA (`frontend/src/`)
- Python 3.x — ERP Bridge daemon (`erp-bridge-aws/bridge.py`)

**Secondary:**
- SQL — PostgreSQL migrations (`backend/migrations/*.sql`) and Oracle ERP queries (`scripts_oracle/*.sql`)

## Runtime

**Environment:**
- Go runtime 1.26 (backend, compiled CGO_ENABLED=0 for static binary)
- Node.js 18 (frontend build stage only, Alpine image)
- Python 3 with `venv` (ERP Bridge on AWS EC2, run via systemd)

**Package Manager:**
- Go: Go modules — lockfile `backend/go.sum` present
- Frontend: npm — lockfile `frontend/package-lock.json` present
- Python: pip (dependencies: `oracledb`, `requests`, `pyyaml`) — no `requirements.txt`, install instructions inline in `erp-bridge-aws/bridge.py`

## Frameworks

**Backend:**
- No web framework — uses Go stdlib `net/http` exclusively (`backend/main.go`)
- `github.com/golang-jwt/jwt/v5` v5.3.1 — JWT token auth
- `github.com/lib/pq` v1.11.2 — PostgreSQL driver
- `github.com/joho/godotenv` v1.5.1 — `.env` loading in development
- `golang.org/x/crypto` v0.49.0 — bcrypt password hashing
- `github.com/johnfercher/maroto/v2` v2.4.0 — PDF generation
- `github.com/google/uuid` v1.6.0 — UUID generation

**Frontend:**
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

**Testing:**
- Go standard `testing` package — integration test in `tests/integration_test.go`
- No frontend testing framework detected

**Build/Dev:**
- Docker Buildx — multi-stage builds
- Vite dev server with `/api` proxy to backend on port 8086
- `start_dev.bat`, `start_docker.bat` — Windows dev helpers

## Key Dependencies

**Critical:**
- `github.com/lib/pq` v1.11.2 — sole PostgreSQL driver; no ORM, raw SQL throughout
- `github.com/golang-jwt/jwt/v5` v5.3.1 — all auth flows depend on this
- `golang.org/x/crypto` — bcrypt for password storage
- `@tanstack/react-query` v5 — all API data fetching on frontend
- `python-oracledb` — Oracle thin client (no Oracle Instant Client needed), required for ERP Bridge

**Infrastructure:**
- `github.com/johnfercher/maroto/v2` v2.4.0 — PDF report generation (stored as `ai_reports`)
- `recharts` v3.7 — fiscal comparison charts on frontend
- `zod` v4 — runtime validation on frontend forms

## Configuration

**Environment:**
- Backend reads env vars at startup via `godotenv.Load()` (dev) or Docker environment block (prod)
- Key vars: `DATABASE_URL`, `JWT_SECRET`, `SMTP_HOST`, `SMTP_PORT`, `SMTP_USER`, `SMTP_PASSWORD`, `SMTP_FROM`, `ALLOWED_ORIGINS`, `PORT`, `APP_URL`
- Template documented in `coolify-env-template.txt`
- ERP Bridge reads `erp-bridge-aws/config.yaml` (YAML config file, not env vars)

**Build:**
- Production: `Dockerfile.production` — 3-stage (Node build → Go build → Alpine runtime)
- Backend only: `backend/Dockerfile` — 2-stage (Go builder → Alpine runtime)
- Frontend only: `frontend/Dockerfile` — 2-stage (Node builder → nginx:stable-alpine)
- Frontend build env var: `VITE_APP_MODULE=fbtax_cloud` (passed as ARG)
- Frontend dev proxy: `VITE_API_TARGET` (defaults to `http://localhost:8086`)

## Database

**Engine:** PostgreSQL 15 Alpine (Docker image `postgres:15-alpine`)

**Migration system:** Custom hand-rolled — Go reads `migrations/*.sql` files sorted alphabetically at startup, tracks execution in `schema_migrations` table (`backend/main.go` `onDBConnected()`)

**Schema features used:**
- `pg_trgm` extension (trigram search on NF-e names, migration 067)
- `gen_random_uuid()` — primary keys throughout
- `portal` schema namespace for portal-related tables (migration 100+)
- Materialized views for fiscal summary data

**Connection pool:**
- MaxOpenConns: 50
- MaxIdleConns: 15
- ConnMaxLifetime: 15 min

## Platform Requirements

**Development:**
- Docker + Docker Compose
- Node.js 18+ (frontend dev server)
- Go 1.26+ (backend dev)
- Python 3.11+ with `oracledb`, `requests`, `pyyaml` (ERP Bridge)
- Windows dev scripts: `start_dev.bat`, `start_docker.bat`

**Production:**
- Deployed on Coolify (self-hosted PaaS) via Docker Compose
- Traefik as reverse proxy — routes `/api/*` to Go backend (port 8086), all other paths to Nginx frontend (port 80)
- TLS via Let's Encrypt (`certresolver=letsencrypt`)
- Domain: `fbtax.cloud` and `www.fbtax.cloud`
- Container image pushed to GHCR (`ghcr.io/claudiosbezerra/...`) via GitHub Actions
- Staging environment: Azure (SSH deploy, no Coolify)
- ERP Bridge: runs on client AWS EC2 as a systemd service (`erp-bridge-aws/erp-bridge.service`)

---

*Stack analysis: 2026-05-19*
