# Coding Conventions

**Analysis Date:** 2026-05-19

## Language Mix

This project spans three languages. Each has distinct conventions:

| Language | Location | Notes |
|----------|----------|-------|
| Go 1.26 | `backend/` | Standard Go idioms; no linter config detected |
| TypeScript 5.2 + React 18 | `frontend/` | Strict TS; ESLint via Vite defaults |
| Python 3.12 | `erp-bridge-aws/bridge.py` | Single script; no formatter config detected |

**Language of human text:** Comments, log messages, user-facing strings, and identifiers are a **bilingual mix of Portuguese (BR) and English**. Technical identifiers (struct fields, function names) lean English; comments and user-facing log messages lean Portuguese. Both languages coexist freely in the same files. When adding code, match the surrounding context:
- Comments in Go handler files: Portuguese for domain logic (`// Obtém o environment_id da empresa`), English for infrastructure patterns (`// Check blacklist before validating signature`)
- Log messages in Go: English prefixes with Portuguese content (`log.Printf("[Login] User not found: %s", req.Email)`)
- Python logging: mostly Portuguese (`log.info("Conectado ao Oracle (thin mode)")`)
- Frontend UI strings: Portuguese

---

## Naming Patterns

### Go (backend/)

**Exported types and functions:** PascalCase
```go
type HealthResponse struct { ... }
func HashPassword(password string) (string, error) { ... }
func GetAllowedOrigins() map[string]bool { ... }
```

**Unexported functions and variables:** camelCase
```go
func getJWTSecret() []byte { ... }
func isSecureCookie(r *http.Request) bool { ... }
var refreshTokenStore sync.Map
```

**Constants:** PascalCase for exported, camelCase for unexported
```go
const BackendVersion = "1.0.0"
const ClaimsKey contextKey = "claims"
```

**Handler naming pattern:** `<Entity><Action>Handler` returning `http.HandlerFunc`
```go
func LoginHandler(db *sql.DB) http.HandlerFunc { ... }
func GetPortalProductsHandler(db *sql.DB) http.HandlerFunc { ... }
```

**Struct fields:** PascalCase with JSON tags using snake_case
```go
type User struct {
    ID          string    `json:"id"`
    FullName    string    `json:"full_name"`
    TrialEndsAt time.Time `json:"trial_ends_at"`
}
```

**File naming:** lowercase with no separator (`auth.go`, `middleware.go`, `portal_products.go`)

### TypeScript/React (frontend/)

**Files:** PascalCase for components (`AuthContext.tsx`, `Login.tsx`, `ProductCard.tsx`); camelCase for utilities (`utils.ts`, `use-mobile.tsx`)

**Components:** PascalCase function declarations exported as default or named
```tsx
const Login = () => { ... }
export default Login;

export function ProductCard({ name, ... }: ProductCardProps) { ... }
```

**Types/Interfaces:** PascalCase
```typescript
interface AuthContextType { ... }
type ProductCardProps = Product & { colorIndex?: number }
```

**Variables/state:** camelCase
```typescript
const [isLoading, setIsLoading] = useState(false)
const [errorMsg, setErrorMsg] = useState<string | null>(null)
```

**Constants (module-level, static config):** SCREAMING_SNAKE_CASE
```typescript
const FEATURES = [ ... ]
const ACCENTS = [ ... ]
const CLIENTE_MAP: Record<string, ...> = { ... }
```

**Path alias:** `@/` maps to `src/` — use exclusively for imports within `frontend/src/`
```typescript
import { useAuth } from "@/contexts/AuthContext"
import { Button } from "@/components/ui/button"
```

### Python (erp-bridge-aws/)

**Functions:** snake_case
```python
def init_tracker() -> sqlite3.Connection:
def ja_enviado(conn, servidor, tipo, chave) -> bool:
def processar_sap(oracle_cfg, data_ini, data_fim, fbtax) -> dict:
```

**Classes:** PascalCase
```python
class FBTaxClient:
```

**Module-level constants:** SCREAMING_SNAKE_CASE
```python
BASE_DIR   = Path(__file__).parent
TRACKER_DB = BASE_DIR / "tracker.db"
BATCH_SIZE = 1000
```

**Private/internal identifiers:** single-underscore prefix
```python
_DECL_RE     = re.compile(r'<\?xml[^?]*\?>', re.IGNORECASE)
_file_handler = logging.FileHandler(log_file, encoding="utf-8")
```

---

## Code Style

### Go

**No automatic formatter config detected** — standard `gofmt` conventions are followed implicitly. Alignment of struct tags and multi-column variable assignments use manual tab-alignment:
```go
var (
    db      *sql.DB
    dbMutex sync.RWMutex
    dbErr   error
)
```

**No ESLint/golangci-lint config file found** in `backend/`. Linting is informal.

### TypeScript/React

**Formatter:** No `.prettierrc` or `biome.json` found. Formatting is not enforced via tooling.

**Linting:** ESLint available via `npm run lint` (`frontend/package.json`). No custom eslint config file found — relies on Vite's default ESLint setup.

**TypeScript strictness:** `strict: true`, `noUnusedLocals: true`, `noUnusedParameters: true` enforced via `frontend/tsconfig.app.json`.

**Semicolons:** Omitted in React component files (`frontend/src/pages/`, `frontend/src/components/`); present in utility files (`frontend/src/lib/utils.ts`). This is inconsistent — match the surrounding file.

**Quotes:** Single quotes in imports (React files), double quotes mixed in some places.

### Python

**No `pyproject.toml`, `ruff.toml`, `.flake8`, or `black` config detected.** Code in `erp-bridge-aws/bridge.py` follows PEP 8 broadly but is not enforced by tooling.

---

## Import Organization

### Go

Standard Go grouping: stdlib → third-party → internal, separated by blank lines:
```go
import (
    "context"
    "database/sql"
    "encoding/json"
    "net/http"

    "fb_cloud/services"
    "github.com/golang-jwt/jwt/v5"
    "golang.org/x/crypto/bcrypt"
)
```

### TypeScript/React

No enforced import order. Observed pattern in page components:
1. React/framework imports
2. UI component library imports (`@/components/ui/...`)
3. External packages (icons, routing, query)
4. Internal contexts/hooks (`@/contexts/...`, `@/hooks/...`)

```typescript
import { useState } from "react"
import { Button } from "@/components/ui/button"
import { toast } from "sonner"
import { useNavigate } from "react-router-dom"
import { useAuth } from "@/contexts/AuthContext"
```

---

## Error Handling

### Go

**Pattern:** Immediate return with `http.Error()` on every error path. No centralized error wrapper. Errors are checked explicitly at every step.

```go
if err != nil {
    log.Printf("[Register] Error creating user: %v", err)
    http.Error(w, "Database error", http.StatusInternalServerError)
    return
}
```

**`sql.ErrNoRows` is handled explicitly** with distinct responses (404 vs 500):
```go
if err == sql.ErrNoRows {
    http.Error(w, "User not found", http.StatusNotFound)
    return
} else if err != nil {
    http.Error(w, "Database error", http.StatusInternalServerError)
    return
}
```

**Transactions:** Always `defer tx.Rollback()` immediately after `db.Begin()`, with explicit `tx.Commit()` at success path. `tx.Rollback()` is also called manually before early returns in some cases (inconsistency — see `backend/handlers/auth.go:509`).

**Error wrapping:** Go's `%w` verb used in `backend/services/email.go` for wrapping:
```go
return fmt.Errorf("TLS dial failed: %w", err)
```

Not consistently used in handler code.

### TypeScript/React

**Pattern:** `try/catch` with typed error extraction in async handlers:
```typescript
try {
    const res = await fetch("/api/auth/login", { ... })
    const data = await res.json()
    if (!res.ok) {
        throw new Error(typeof data === "string" ? data : "Credenciais inválidas")
    }
    login(data)
} catch (error: unknown) {
    const msg = error instanceof Error ? error.message : "Erro desconhecido"
    setErrorMsg(msg)
    toast.error(msg)
} finally {
    setIsLoading(false)
}
```

Toast notifications (`sonner`) used for user-facing error feedback.

### Python

**Pattern:** `try/except Exception as exc` with `log.error()` or `log.warning()`. Functions return early with error-state dict rather than raising:
```python
except Exception as exc:
    log.error("Falha ao conectar em %s: %s", nome, exc)
    stats["sap_batch"]["erros"] = 1
    stats["sap_batch"]["erro_msg"] = str(exc)
    return stats
```

Non-critical errors in daemon mode swallow exceptions and continue:
```python
except Exception as exc:
    log.warning("Nao foi possivel registrar servidores: %s", exc)
```

---

## Logging

### Go

**Library:** Standard `log` package (`log.Printf`, `log.Println`, `log.Fatal`)

**Pattern:** Prefixed with `[HandlerName]` in square brackets for structured searchability:
```go
log.Printf("[Login] Attempting login for: %s", req.Email)
log.Printf("[Register] Error creating user: %v", err)
log.Printf("[Email Service] Sending password reset email to %s via %s:%d", ...)
```

`fmt.Println` / `fmt.Printf` used for startup/infrastructure messages (DB connection, migration status). `log.Printf` used for request-scoped events.

### Python

**Library:** `logging` module with dual output (file + stdout). Module-level `log = logging.getLogger(__name__)`.

File handler: DEBUG level, timestamped.
Screen handler: INFO level, timestamped.

```python
log.info("Conectado ao Oracle (thin mode)")
log.warning("Token expirado — renovando sessao...")
log.error("Falha ao conectar em %s: %s", nome, exc)
```

Log file created per-run: `erp-bridge-aws/logs/bridge_YYYYMMDD_HHMMSS.log`

### Frontend

`console.error` used sparingly for session errors (`frontend/src/contexts/AuthContext.tsx:100`). No structured frontend logging.

---

## Comments

**Go:** Comments use full sentences for exported symbols (godoc-style). Inline comments use `//` for blocks of logic:
```go
// GetMeHandler returns the current authenticated user's details
func GetMeHandler(db *sql.DB) http.HandlerFunc { ... }

// Check blacklist before validating signature
if _, revoked := tokenBlacklist.Load(tokenString); revoked { ... }
```

Section dividers use Unicode box-drawing lines for visual grouping:
```go
// ─── Rate Limiter ─────────────────────────────────────────────────────────────
```

**TypeScript/React:** JSDoc not used. Inline comments in Portuguese explain UI intent:
```typescript
// Refs para o interceptor de fetch (sem stale closure)
// Interceptor global de fetch: injeta Authorization e X-Company-ID
```

**Python:** Module docstring at top of file. Section headers use Unicode dividers matching Go style. Inline comments explain business domain logic in Portuguese.

---

## Function Design

### Go

**Handler factory pattern:** All handlers return `http.HandlerFunc` closures that capture a `*sql.DB`:
```go
func LoginHandler(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // ...
    }
}
```

Functions are large by convention — `LoginHandler` is 194 lines; `auth.go` is 1139 lines. No enforced size limit.

### TypeScript/React

Components use arrow function assignment (`const Login = () => { ... }`), except named component exports which use `function` keyword (`export function ProductCard(...)`). Either pattern is acceptable — match the file style.

### Python

Functions are procedural and large (`processar_sap` is 178 lines, `run_daemon` is 157 lines). No max size enforced.

---

## Module Design

### Go

**Package structure:** One package per directory. All handler code lives in `package handlers` (`backend/handlers/`). Services in `package services` (`backend/services/`). No barrel files.

**Exports:** Exported only when consumed from another package (e.g., `handlers.LoginHandler` called from `main.go`). Internal helpers kept unexported.

### TypeScript/React

**No barrel files** (`index.ts`) in use. Import components directly by path:
```typescript
import { ProductCard } from "@/components/ProductCard"
import { useAuth } from "@/contexts/AuthContext"
```

UI primitives in `frontend/src/components/ui/` are thin wrappers over Radix UI — do not modify them; extend via composition.

---

*Convention analysis: 2026-05-19*
