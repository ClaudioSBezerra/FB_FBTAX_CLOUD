# Testing Patterns

**Analysis Date:** 2026-05-19

## Overview

Testing coverage is minimal across the project. There is one integration test in Go, one unit test file in TypeScript (Vitest), and no tests at all in Python. This section documents what exists and establishes patterns to follow when adding tests.

---

## Test Framework

### Frontend (TypeScript)

**Runner:** Vitest (inferred from `frontend/src/lib/utils.test.ts` — uses `import { expect, test } from 'vitest'`)
- No `vitest.config.*` found in `frontend/`. Vitest likely runs via Vite's default detection.
- No explicit version in `package.json` devDependencies (not yet added as a dependency — see Concerns).

**Assertion Library:** Vitest built-in (`expect`)

**Run Commands:**
```bash
# No test script defined in frontend/package.json
# To add: "test": "vitest run", "test:watch": "vitest"
cd frontend && npx vitest run    # one-shot
cd frontend && npx vitest        # watch mode
cd frontend && npx vitest --coverage  # coverage (requires @vitest/coverage-v8)
```

### Backend (Go)

**Runner:** Go standard `testing` package (`tests/integration_test.go`)

**Run Commands:**
```bash
cd tests && go test ./...        # Run integration tests
go test ./tests/...              # From repo root
go test -v ./tests/...           # Verbose output
```

**Note:** The integration test (`tests/integration_test.go`) requires a running Docker environment. It gracefully skips if the backend is not reachable rather than failing hard.

### Python (erp-bridge-aws/)

No test framework detected. No test files found. No `pytest.ini`, `pyproject.toml`, or test directory exists for `erp-bridge-aws/`.

---

## Test File Organization

### Frontend

**Pattern:** Co-located with source files, same directory, `.test.ts` suffix.

```
frontend/src/
├── lib/
│   ├── utils.ts
│   └── utils.test.ts          ← co-located unit test
```

**Naming:** `<filename>.test.ts` or `<filename>.test.tsx` for component tests.

When adding a new utility at `src/lib/myUtil.ts`, place its test at `src/lib/myUtil.test.ts`.

When adding a new component at `src/components/MyComponent.tsx`, place its test at `src/components/MyComponent.test.tsx`.

### Backend (Go)

**Pattern:** Separate top-level `tests/` directory for integration tests. Go unit tests would follow the standard Go convention of `*_test.go` files co-located with source.

```
tests/
└── integration_test.go         ← package tests, integration-only
backend/
└── handlers/
    └── auth_test.go            ← (would-be) unit test location
```

---

## Test Structure

### Frontend (Vitest)

**Actual pattern from `frontend/src/lib/utils.test.ts`:**
```typescript
import { expect, test } from 'vitest'
import { cn } from './utils'

test('cn merges class names correctly', () => {
  expect(cn('c-1', 'c-2')).toBe('c-1 c-2')
})

test('cn handles conditional classes', () => {
  expect(cn('c-1', true && 'c-2', false && 'c-3')).toBe('c-1 c-2')
})

test('cn merges tailwind classes', () => {
  expect(cn('p-1', 'p-2')).toBe('p-2')
})
```

**Use `describe` for grouping related tests** (not yet used — adopt when adding multiple related tests for one module):
```typescript
import { describe, test, expect } from 'vitest'

describe('formatCurrency', () => {
  test('formats positive BRL values', () => {
    expect(formatCurrency(1234.56)).toBe('R$ 1.234,56')
  })
  test('formats zero', () => {
    expect(formatCurrency(0)).toBe('R$ 0,00')
  })
})
```

### Backend (Go)

**Actual pattern from `tests/integration_test.go`:**
```go
package tests

import (
    "net/http"
    "testing"
    "time"
)

func TestHealthCheck(t *testing.T) {
    baseURL := "http://localhost:8080/api/health"
    client := &http.Client{Timeout: 5 * time.Second}

    resp, err := client.Get(baseURL)
    if err != nil {
        // Graceful skip — not a hard failure
        t.Logf("Aviso: Backend não acessível (%v). Teste ignorado se o ambiente não estiver rodando.", err)
        return
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        t.Errorf("Esperado status 200, recebeu %d", resp.StatusCode)
    }
}
```

**Pattern:** Test function names `Test<DescriptiveName>`, taking `*testing.T`. Comments in Portuguese matching the project's bilingual convention.

---

## Mocking

### Frontend

**No mock framework explicitly installed.** Vitest has built-in `vi.fn()`, `vi.mock()`, `vi.spyOn()`.

**For fetch calls:** The `window.fetch` interceptor in `AuthContext.tsx` patches `window.fetch` globally — in tests, replace with `vi.stubGlobal('fetch', vi.fn())` or use `vi.mock`:
```typescript
import { vi, test, expect, beforeEach } from 'vitest'

beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
        ok: true,
        json: async () => ({ token: 'test-token', user: { id: '1' } }),
    }))
})
```

**What to mock:**
- `fetch` calls to backend API endpoints
- `localStorage` (use `vi.stubGlobal` or jsdom's built-in)
- `window.location` navigation (use `vi.stubGlobal`)

**What NOT to mock:**
- `cn()` and other pure utility functions — test these directly
- Radix UI primitives — test behavior, not internals

### Backend (Go)

**No mock library detected.** Standard approach is to pass a `*sql.DB` connected to a test database, or use an interface for the DB layer (not currently in use — all handlers take `*sql.DB` directly).

**For unit-testing handlers without a live DB:** Construct an `httptest.ResponseRecorder` and `httptest.NewRequest`, then call the handler directly:
```go
import (
    "net/http/httptest"
    "testing"
)

func TestGetPortalProducts(t *testing.T) {
    // requires a real test DB or mock
    req := httptest.NewRequest(http.MethodGet, "/api/portal/products", nil)
    w := httptest.NewRecorder()
    // handler(db)(w, req)
    // assert w.Code, w.Body
}
```

---

## Fixtures and Factories

**No fixtures or factory helpers currently exist.** When adding them:

**Frontend — test data pattern:**
```typescript
// src/lib/testHelpers.ts (create when needed)
export function makeUser(overrides = {}) {
    return {
        id: 'test-id',
        email: 'test@example.com',
        full_name: 'Test User',
        trial_ends_at: new Date(Date.now() + 14 * 86400000).toISOString(),
        role: 'user',
        ...overrides,
    }
}
```

**Go — test helpers:** Place in `tests/helpers_test.go` (package `tests`) or co-located `backend/handlers/testhelpers_test.go`.

---

## Coverage

**Requirements:** No coverage threshold enforced anywhere.

**View Coverage (Frontend):**
```bash
cd frontend && npx vitest --coverage
# Requires: npm install -D @vitest/coverage-v8
```

**View Coverage (Go):**
```bash
cd tests && go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

---

## Test Types

### Unit Tests

**Frontend:** `frontend/src/lib/utils.test.ts` — tests `cn()` utility function. Pure function, no I/O.

**Backend Go:** None exist. Candidate targets: `HashPassword`/`CheckPasswordHash` in `backend/handlers/auth.go`, `GetClientIP` in `backend/handlers/middleware.go`, `formatEmailBRL` in `backend/services/email.go`.

**Python:** None exist.

### Integration Tests

**Go:** `tests/integration_test.go` — HTTP call to `/api/health`. Requires Docker stack running. Skips gracefully when environment is not available.

### E2E Tests

No E2E test framework detected (no Playwright, Cypress, or Selenium). Not in use.

---

## Common Patterns

### Async Testing (Frontend)

```typescript
import { test, expect, vi } from 'vitest'

test('login calls fetch and sets token', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
        ok: true,
        json: async () => ({ token: 'abc', user: { id: '1' }, environment_name: 'Env', group_name: 'Grp', company_name: 'Co', company_id: 'c1', cnpj: '' }),
    }))

    // ... invoke component behavior ...

    expect(localStorage.getItem('token')).toBe('abc')
})
```

### Error Testing (Frontend)

```typescript
test('login shows error on 401', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
        ok: false,
        json: async () => 'Credenciais inválidas',
    }))

    // ... invoke handleLogin ...
    // assert error state is set
})
```

### Error Testing (Go)

```go
func TestLoginHandlerUnauthorized(t *testing.T) {
    req := httptest.NewRequest(http.MethodPost, "/api/auth/login",
        strings.NewReader(`{"email":"x@y.com","password":"wrong"}`))
    req.Header.Set("Content-Type", "application/json")
    w := httptest.NewRecorder()

    // handlers.LoginHandler(testDB)(w, req)

    if w.Code != http.StatusUnauthorized {
        t.Errorf("expected 401, got %d", w.Code)
    }
}
```

---

## What Is Not Tested (Gaps)

The following critical paths have no tests at all:

- `backend/handlers/auth.go` — `LoginHandler`, `RegisterHandler`, `ResetPasswordHandler`, JWT generation and verification
- `backend/handlers/middleware.go` — `AuthMiddleware`, `SecurityMiddleware`, rate limiter logic
- `backend/handlers/environment.go` — company/environment CRUD
- `erp-bridge-aws/bridge.py` — Oracle connection, XML normalization, batch sending, daemon loop
- `frontend/src/contexts/AuthContext.tsx` — session restore, `login()`, `logout()`, `switchCompany()`
- `frontend/src/pages/` — Login form submission, error display, navigation

These are high-risk areas; any regression goes undetected.

---

*Testing analysis: 2026-05-19*
