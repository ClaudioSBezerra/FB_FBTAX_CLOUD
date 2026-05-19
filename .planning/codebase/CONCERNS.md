# Codebase Concerns

**Analysis Date:** 2026-05-19

---

## Tech Debt

**ERP Bridge routes not registered in current backend:**
- Issue: `erp-bridge-aws/bridge.py` calls 10+ endpoints (`/api/erp-bridge/import/batch`, `/api/erp-bridge/runs`, `/api/erp-bridge/config`, `/api/erp-bridge/heartbeat`, `/api/erp-bridge/credentials`, `/api/erp-bridge/parceiros/sync`, etc.) that are defined in database migrations (`backend/migrations/074_erp_bridge.sql` through `091_erp_bridge_only_parceiros.sql`) but are **nowhere registered** in `backend/main.go` (lines 345-354 only register auth and portal routes). The handlers for these routes either live in a separate deployment (the older `fb_apu01` product) or were never ported to this repo.
- Files: `backend/main.go:345-354`, `erp-bridge-aws/bridge.py:302-567`, `backend/migrations/074_erp_bridge.sql`
- Impact: The bridge daemon in `--daemon` mode will silently fail all API calls (401 or 404). The erp_bridge tables are created by migrations but never used by this backend instance.
- Fix approach: Register erp-bridge handlers in `main.go`, or document that the bridge points at a different host.

**Migration runner does not abort on failure:**
- Issue: `backend/main.go:241-254` runs all migrations in a loop and only `continue`s on error. A failed migration is silently skipped (unless it happens to mention "already exists" or "duplicate", in which case it is marked as applied anyway). This means a partially applied migration leaves the schema in an unknown state with no alert.
- Files: `backend/main.go:241-254`
- Impact: Schema corruption can go undetected. Subsequent migrations that depend on the failed one will likely also fail but be silently swallowed.
- Fix approach: Return/panic on migration failure rather than continuing; or use a proper migration library (goose, golang-migrate).

**Duplicate migration numbers (021 and 061):**
- Issue: Two files share the prefix `021` (`021_create_mv_mercadorias.sql` and `021_ensure_admin_user.sql`) and two share `061` (`061_add_filial_to_mv_operacoes_simples.sql` and `061_user_environments_preferred_company.sql`). Because the runner uses the full filename as a primary key, both run — but the ordering between same-prefix files is filesystem-dependent (alphabetical, not temporal intent).
- Files: `backend/migrations/021_create_mv_mercadorias.sql`, `backend/migrations/021_ensure_admin_user.sql`, `backend/migrations/061_add_filial_to_mv_operacoes_simples.sql`, `backend/migrations/061_user_environments_preferred_company.sql`
- Impact: On a fresh deployment the ordering between the two `021` files may differ from the author's intent; if the materialized view depends on admin user data, it could fail.
- Fix approach: Rename to use unique sequential prefixes (e.g., `021b_ensure_admin_user.sql` → `022_ensure_admin_user.sql`).

**Migration sequence has large gaps:**
- Issue: Migrations jump from `002` to `005`, from `010` to `012`, and from `093` to `100`. Files 003, 004, 011, 094-099 do not exist.
- Files: `backend/migrations/`
- Impact: Not immediately dangerous, but indicates migrations were deleted rather than reverted cleanly. Restoring from backup to an earlier point will fail.
- Fix approach: Document why gaps exist; do not delete old migrations.

**`SendAIReportEmail` has no caller in this codebase:**
- Issue: `backend/services/email.go:217` defines `SendAIReportEmail` which accepts AI narrative markdown and tax comparison data, but no handler in `backend/main.go` or any registered handler file calls it. The `coolify-env-template.txt` references `ZAI_API_KEY` for a Z.AI GLM integration, but no code in this repo reads that env var.
- Files: `backend/services/email.go:214-335`, `coolify-env-template.txt:43`
- Impact: Dead code that carries complex dependencies; the feature it was designed to support (AI-generated executive reports) does not exist in the current deployment.
- Fix approach: Either remove the function and env var, or implement the caller.

**`coolify-env-template.txt` references vars the backend ignores:**
- Issue: Template lists `REDIS_HOST/PORT/PASSWORD`, `REDIS_ADDR`, `ZAI_API_KEY`, `RFB_API_URL`, `RFB_TOKEN_URL`, `RFB_WEBHOOK_URL`, `RATE_LIMIT_*`, `SESSION_TIMEOUT`, `AUDIT_LOGS`, `METRICS_ENABLED` — none of which are read by the current `backend/main.go` or any handler. The template is inherited from the older `fb_apu01` product.
- Files: `coolify-env-template.txt`
- Impact: Operators setting these vars get a false sense of security (rate limits, audit logs, Redis cache). The feature set described in the template does not match the deployed code.
- Fix approach: Replace template with one that accurately describes only vars this backend reads: `PORT`, `DATABASE_URL`, `JWT_SECRET`, `SMTP_*`, `ALLOWED_ORIGINS`.

**`backup_production.sh` targets wrong database name:**
- Issue: `scripts/backup_production.sh:9` hard-codes `DB_NAME="fiscal_db"` and `BACKUP_DIR="/opt/fb_apu01/backups"` — both are legacy `fb_apu01` values. The current production database is `fbtax_cloud`.
- Files: `scripts/backup_production.sh:9-10`
- Impact: Running this script against the live `fbtax_cloud` deployment produces a backup of `fiscal_db` (which may not exist) or an empty dump.
- Fix approach: Update to read `DB_NAME` from env var and parameterize `BACKUP_DIR`.

**`installer/cliente-aws/setup.sh` and `installer/docker-compose.yml` pull `fb_apu01` images:**
- Issue: `installer/cliente-aws/setup.sh:10-11` and `installer/docker-compose.yml:4,40` reference `ghcr.io/claudiosbezerra/fb_apu01-api:latest` and `fb_apu01-web:latest`, while `installer/cliente-aws/docker-compose.yml` references `fb_apu02-api:latest`. Neither matches the current project (`FB_FBTAX_CLOUD`).
- Files: `installer/cliente-aws/setup.sh:10-11`, `installer/docker-compose.yml:4`, `installer/cliente-aws/docker-compose.yml:4,43`
- Impact: Installing via the `installer/` path deploys the old product, not this one.
- Fix approach: Update image references to match the current project's GHCR path, or archive the installer directory if it targets a separate product.

---

## Known Bugs

**`docker-compose.yml` exposes backend port 8086 to the host:**
- Symptoms: `ports: - "8086:8086"` binds the API directly to the host network, bypassing Traefik TLS termination. Any host-level network access can hit the unauthenticated Go HTTP server.
- Files: `docker-compose.yml:10-11`
- Trigger: Running `docker-compose up` in any environment where external traffic reaches the host.
- Workaround: Use only `expose: - "8086"` (already present at line 8) and remove the `ports` mapping; Traefik can route via Docker network.

**`tunnel-prod-db.sh` uses `StrictHostKeyChecking=no` and binds to `0.0.0.0`:**
- Symptoms: The SSH tunnel (`scripts/tunnel-prod-db.sh:22-25`) binds the local port to all interfaces (`0.0.0.0:5435`) and disables SSH host key verification. Any process on the developer machine — or on the local network — can reach the production database.
- Files: `scripts/tunnel-prod-db.sh:22-25`
- Trigger: Running the tunnel on a shared network or CI environment.
- Workaround: Bind to `127.0.0.1:5435` only; re-enable `StrictHostKeyChecking`.

**`test/integration_test.go` hits wrong port and silently passes:**
- Symptoms: The only integration test (`tests/integration_test.go:14`) hits `http://localhost:8080/api/health`, but the service runs on port 8086. The test swallows connection errors and returns without failure (`t.Logf` + `return`), meaning `go test ./tests/...` always passes regardless of service state.
- Files: `tests/integration_test.go:14,21-23`
- Trigger: Any CI run.
- Workaround: Fix the port; convert the soft return into `t.Skip` and add an explicit check that the port is reachable before running assertions.

---

## Security Considerations

**`erp-bridge-aws/config.yaml` contains real plaintext credentials committed to the repo:**
- Risk: Lines 14-16 contain a real email address (`claudio.bezerra@ferreiracosta.com.br`) and a real password (`Proxy#6939`) for the production FBTax instance at `fctax.fcxlabs.com`. The commented-out legacy `servidores` blocks (lines 33-130) expose real internal Oracle DSN IPs, usernames, and passwords (`fcosta`/`fcosta2013`) for 11 production Oracle databases.
- Files: `erp-bridge-aws/config.yaml:14-16`, `erp-bridge-aws/config.yaml:33-130`
- Current mitigation: None — the file is tracked by git.
- Recommendations: Immediately rotate `Proxy#6939` and all `fcosta2013` Oracle passwords. Remove `config.yaml` from git history (use `git filter-repo`). Add `erp-bridge-aws/config.yaml` to `.gitignore`. Use environment variables or a secrets manager for runtime credentials.

**`delete_branches_with_token.sh` encodes GitHub token in the remote URL:**
- Risk: `delete_branches_with_token.sh:24` passes the GitHub token inline in the remote URL as `https://ClaudioSBezerra:$TOKEN@github.com/...`. Tokens passed this way can appear in process lists (`ps aux`), shell history, and git reflog.
- Files: `delete_branches_with_token.sh:24`
- Current mitigation: Token is a CLI argument, not hardcoded.
- Recommendations: Use `GH_TOKEN` env var with `gh auth login` instead; or pass via credential helper.

**Login failure logs the email address:**
- Risk: `backend/handlers/auth.go:635` logs `[Login] Invalid password for: <email>` at `Printf` level, exposing authenticated email addresses in plain-text logs on every failed attempt.
- Files: `backend/handlers/auth.go:635`
- Current mitigation: Rate limiter restricts attempts per IP.
- Recommendations: Log a hash or partial address, or reduce log level to DEBUG; avoid logging PII in production.

**CSP header allows `script-src 'unsafe-inline'`:**
- Risk: `backend/handlers/middleware.go:69` sets `script-src 'self' 'unsafe-inline'`, which defeats XSS protection for inline scripts. The React bundle is entirely `'self'`-served, so `'unsafe-inline'` provides no benefit and is unnecessary.
- Files: `backend/handlers/middleware.go:66-74`
- Current mitigation: `X-XSS-Protection: 1; mode=block` is set.
- Recommendations: Remove `'unsafe-inline'` from `script-src`; add a nonce or hash if any inline scripts are required.

**Database connection uses `sslmode=disable` in production compose:**
- Risk: `docker-compose.yml:14` sets `sslmode=disable` in the `DATABASE_URL` default. Traffic between the `api` container and the `db` container is unencrypted.
- Files: `docker-compose.yml:14`, `backend/main.go:118`
- Current mitigation: Both services are on the same Docker bridge network `fb_net`.
- Recommendations: Acceptable for intra-container on the same host; document this explicitly. If PostgreSQL is ever moved to a managed service (e.g., Hostinger external DB), the connection string must be updated to `sslmode=require`.

**`backend/tools/` debug files contain hardcoded credentials:**
- Risk: `backend/tools/debug_stats.go:16`, `debug_query.go:17`, `debug_gilson.go:26`, `debug_detailed.go:17` all hardcode `postgres://postgres:postgres@localhost:5432/fiscal_db?sslmode=disable`. `verify_data.go:14` uses `fb_apu01` as the database name. These files are part of the build path (build tag `//go:build scripts` on some but not all).
- Files: `backend/tools/debug_stats.go:16`, `backend/tools/debug_query.go:17`, `backend/tools/verify_data.go:14`
- Current mitigation: `debug_stats.go` and `debug_query.go` use `//go:build scripts` so they are excluded from normal builds. `verify_data.go` does not have a build tag.
- Recommendations: Add `//go:build scripts` to all tool files; never ship credentials in source — always fall back to env var only.

---

## Performance Bottlenecks

**`time.Sleep(200ms)` per 1000 lines in SPED parser (legacy concern):**
- Problem: `VALIDATION_REPORT_20260204.md:36` documents an intentional `time.Sleep(200ms)` inserted every 1000 lines during SPED file processing to avoid CPU saturation on a shared VPS.
- Files: `backend/` (the worker referenced in the report; the sleep is in the older `fb_apu01` product, not visible in this repo's current handlers)
- Cause: VPS is Hostinger shared hosting with limited CPU quota; processing large SPED batches (12 months) saturates the scheduler.
- Improvement path: Move to a dedicated worker process; use PostgreSQL `COPY` for bulk inserts; or switch to a larger VPS tier.

**In-memory rate limiter not cleaned up:**
- Problem: `backend/handlers/middleware.go:120-193` stores rate limit entries in a `map[string][]time.Time` that is never evicted. Under sustained abuse (many unique IPs), the map grows without bound for the process lifetime.
- Files: `backend/handlers/middleware.go:120-133`
- Cause: No background goroutine to expire old buckets; `Allow()` and `IsLimited()` prune only the specific key being checked.
- Improvement path: Add a periodic cleanup goroutine, or replace with a Redis-backed rate limiter.

**ERP Bridge loads all Oracle results into memory before sending:**
- Problem: `erp-bridge-aws/bridge.py:614-617` fetches all rows into a Python list before chunking. For large Oracle databases with millions of NF-e records, this materializes the entire result set in RAM.
- Files: `erp-bridge-aws/bridge.py:614-617`
- Cause: `fetchall()` used instead of iterating the cursor.
- Improvement path: Use cursor iteration with server-side batching; process and send each chunk before fetching the next.

---

## Fragile Areas

**Nginx DNS caching causing 502s (recently fixed, but the workaround is fragile):**
- Files: `docker-compose.yml` (Traefik labels), commit history (`fix(nginx): resolver 127.0.0.11 + set $upstream api:8086`)
- Why fragile: The resolver workaround (`set $upstream api:8086`) depends on the Docker embedded DNS being available at startup. If the `coolify` external network changes or Traefik restarts before DNS is ready, the `api` upstream becomes unresolvable and Nginx returns 502 until restart.
- Safe modification: Any change to service names or port numbers in `docker-compose.yml` must also update the Traefik labels (`traefik.http.services.fbtax-cloud-api.loadbalancer.server.port`).
- Test coverage: No automated test verifies the routing chain end-to-end.

**`db-init` service race condition:**
- Files: `docker-compose.yml:89-105`
- Why fragile: The `db-init` one-shot container uses `service_healthy` on `db`, but the `pg_isready` healthcheck only verifies the port is open — not that the `postgres` superuser has finished initializing. On fast hardware the `CREATE DATABASE` can still fail with "role does not exist" if Postgres is mid-init.
- Safe modification: Add `sleep 2` before the `psql` call, or verify with an actual connection rather than `pg_isready`.
- Test coverage: None.

**Materialized view refresh can still silently fall back:**
- Files: `VALIDATION_REPORT_20260204.md:14-18`, migration `034_add_unique_index_mv.sql`
- Why fragile: `worker.go` (in the older product) implements fallback from `CONCURRENTLY` to blocking refresh. If the unique index `idx_mv_unique_concurrent` is ever dropped (e.g., by a botched migration), the fallback silently blocks all reads during the refresh window.
- Safe modification: Do not drop or alter the unique index on `mv_mercadorias_agregada` without also removing the `CONCURRENTLY` codepath.
- Test coverage: No test validates the index exists post-migration.

**`ensureDatabaseExists` uses string parsing to extract DB name:**
- Files: `backend/main.go:57-110`
- Why fragile: The function manually parses the `DATABASE_URL` connection string using `strings.Index` to extract the database name. If the URL contains URL-encoded characters, IPv6 hosts, or non-standard syntax, the parser can silently extract the wrong name and attempt to connect to the wrong database.
- Safe modification: Use `url.Parse` or the `pq.ParseURL` helper from `lib/pq` instead of manual string slicing.
- Test coverage: No unit tests for this function.

**Auto-provisioning at login creates shared "Ambiente de Testes":**
- Files: `backend/handlers/auth.go:715-750`
- Why fragile: When a user has no company link, the login handler creates (or reuses) a single shared environment named `"Ambiente de Testes"` and a shared group `"Grupo de Empresas Testes"`. Multiple orphaned users end up in the same test environment, meaning they can potentially see each other's data if row-level company isolation is not enforced everywhere.
- Safe modification: Audit all report/data queries to confirm they filter by `company_id` before enabling public registration.
- Test coverage: No test validates tenant isolation for auto-provisioned users.

---

## Scaling Limits

**Single PostgreSQL container, no read replicas:**
- Current capacity: One `postgres:15-alpine` container on a single VPS.
- Limit: Concurrent SPED imports + dashboard queries + ERP bridge writes share one connection pool (max 50 connections set in `backend/main.go:136`). Under a batch import of 12 months of SPED data, the worker pool saturates connections and dashboard queries queue.
- Scaling path: Add a read replica for report queries; move bulk imports to a background queue with its own connection pool.

**Rate limiter is single-process, not cluster-aware:**
- Current capacity: In-memory map in one Go process.
- Limit: If multiple backend replicas are deployed behind a load balancer, each has its own independent rate limiter, effectively multiplying the allowed request rate by the number of replicas.
- Scaling path: Replace with Redis-backed rate limiting using `INCR`/`EXPIRE`.

---

## Dependencies at Risk

**`lib/pq` PostgreSQL driver (deprecated):**
- Risk: `backend/go.mod` uses `github.com/lib/pq`. This driver is in maintenance-only mode; the community has moved to `pgx`. `lib/pq` does not support newer PostgreSQL features (batch protocol, pipeline mode).
- Impact: No immediate breakage, but migrating later will require rewriting all `database/sql` scanning code.
- Migration plan: Replace with `pgx/v5` with `pgx/stdlib` compatibility shim; no query changes required in the first pass.

**`python-oracledb` thin mode (no Oracle Client installed):**
- Risk: `erp-bridge-aws/bridge.py` uses `oracledb` in thin mode (no Oracle Instant Client). Thin mode does not support all Oracle features (e.g., `DBMS_LOB`, advanced security options, some data types). Large CLOBs fetched via `clob_para_str` may fail on thin mode with certain Oracle 12c servers.
- Files: `erp-bridge-aws/bridge.py:592-599`, `erp-bridge-aws/bridge.py:153-158`
- Impact: Silent data truncation or connection failures against older Oracle versions.
- Migration plan: Install Oracle Instant Client on the AWS host to enable thick mode; call `oracledb.init_oracle_client()` at startup.

---

## Missing Critical Features

**No automated backup of production PostgreSQL:**
- Problem: `scripts/backup_production.sh` targets the wrong database name (`fiscal_db`) and must be run manually. There is no cron job, no S3 upload, and no retention policy enforced.
- Blocks: Point-in-time recovery for the production database.

**No structured logging or log aggregation:**
- Problem: All backend logging uses `log.Printf` (stdlib) to stdout. There is no structured format (JSON), no correlation IDs per request, and no centralized log sink (Loki, CloudWatch, Datadog).
- Blocks: Post-incident investigation, request tracing, and alerting on error rates.

**No health check on the `api` service in `docker-compose.yml`:**
- Problem: `docker-compose.yml` defines a healthcheck only for the `db` service (line 83). The `api` service has no `healthcheck` block, so Docker/Coolify cannot automatically restart it on application-level failure. (`docker-compose.prod.yml` has one at line 27, but the production deployment uses `docker-compose.yml`.)
- Files: `docker-compose.yml:7-38`, `docker-compose.prod.yml:27-32`
- Blocks: Automatic self-healing on silent backend failure.

---

## Test Coverage Gaps

**Backend: zero unit or integration tests for business logic:**
- What's not tested: All of `backend/handlers/auth.go` (1139 lines), `backend/handlers/environment.go` (326 lines), `backend/handlers/admin.go` (294 lines), the entire migration runner, the `ensureDatabaseExists` function.
- Files: `tests/integration_test.go` (1 test, wrong port, always passes)
- Risk: Regressions in auth, auto-provisioning, or migration logic go undetected until production.
- Priority: High

**Frontend: only a trivial utility test exists:**
- What's not tested: `Login.tsx`, `PortalPage.tsx`, `AuthContext.tsx`, all hooks. The only test is `frontend/src/lib/utils.test.ts` which tests 3 CSS class-merge cases.
- Files: `frontend/src/lib/utils.test.ts`
- Risk: Login flow, password reset, and portal rendering regressions are invisible.
- Priority: High

**ERP Bridge: no tests at all:**
- What's not tested: Oracle connection logic, XML normalization (`normalizar_xml`), SAP S4/HANA query processing, daemon loop, watermark management, tracker deduplication.
- Files: `erp-bridge-aws/bridge.py` (1292 lines, 0 tests)
- Risk: Silent data loss or double-send on any change to the bridge.
- Priority: High

**Destructive scripts not tested and have no dry-run guards:**
- What's not tested: `cleanup_old_branches.sh`, `delete_branches_with_token.sh` — both permanently delete remote git branches. The only guard is a `read -p` prompt in `cleanup_old_branches.sh`; `delete_branches_with_token.sh` has none.
- Files: `cleanup_old_branches.sh:20-28`, `delete_branches_with_token.sh:22-29`
- Risk: Accidental deletion of branches if the script is run non-interactively (e.g., in a CI context).
- Priority: Medium

---

*Concerns audit: 2026-05-19*
