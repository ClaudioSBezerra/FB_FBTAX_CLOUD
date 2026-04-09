# Story 2.1: Endpoint de Produtos Públicos

Status: review

## Story

Como visitante do portal,
Quero que a API retorne a lista de produtos com indicação de quais estão contratados pelo meu tenant,
Para que o frontend possa renderizar a vitrine corretamente.

## Acceptance Criteria

**AC1:** `GET /api/portal/products` (sem `?tenant=`) retorna `200 OK` com array JSON de todos os produtos ativos (`is_active = true`) contendo: `id`, `name`, `description`, `icon_url`, `destination_url`, `contracted: false`.

**AC2:** `GET /api/portal/products?tenant=slug` — se o tenant existir em `pt_tenants`, produtos presentes em `pt_tenant_products` para esse tenant com `is_active = true` retornam `contracted: true`; os demais retornam `contracted: false`.

**AC3:** `GET /api/portal/products?tenant=slug-inexistente` retorna `200 OK` com todos os produtos e `contracted: false` (tenant não encontrado não é erro).

**AC4:** Todas as queries utilizam prepared statements (NFR8) — nenhuma interpolação de string SQL.

**AC5:** A rota é registrada em `main.go` como `/api/portal/products` via `withDB(handlers.GetPortalProductsHandler)`.

**AC6:** `go build ./...` passa sem erros.

## Tasks / Subtasks

- [x] Task 1: Criar `backend/handlers/portal_products.go` (AC: #1, #2, #3, #4)
  - [x] 1.1: Struct `ProductResponse` com `id`, `name`, `description`, `icon_url`, `destination_url`, `contracted`
  - [x] 1.2: `GetPortalProductsHandler(db *sql.DB) http.HandlerFunc` implementado
  - [x] 1.3: `r.URL.Query().Get("tenant")` para extrair o param
  - [x] 1.4: Query simples sem tenant — todos com `contracted: false`
  - [x] 1.5: Query com LEFT JOIN + subquery `pt_tenants.slug = $1` (prepared statement)
  - [x] 1.6: `w.Header().Set("Content-Type", "application/json")` + `json.NewEncoder(w).Encode(products)`

- [x] Task 2: Registrar rota em `backend/main.go` (AC: #5)
  - [x] 2.1: `/api/portal/products` registrado via `withDB(handlers.GetPortalProductsHandler)`

- [x] Task 3: Validar (AC: #6)
  - [x] 3.1: `go build ./...` → BUILD OK

## Dev Notes

### Handler pattern obrigatório (baseado em auth.go e admin.go)

```go
package handlers

import (
    "database/sql"
    "encoding/json"
    "net/http"
)

type ProductResponse struct {
    ID             string `json:"id"`
    Name           string `json:"name"`
    Description    string `json:"description"`
    IconURL        string `json:"icon_url"`
    DestinationURL string `json:"destination_url"`
    Contracted     bool   `json:"contracted"`
}

func GetPortalProductsHandler(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // apenas GET
        if r.Method != http.MethodGet {
            http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
            return
        }
        // ...
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(products)
    }
}
```

### Query sem tenant (AC1)

```sql
SELECT id, name, description, COALESCE(icon_url,''), COALESCE(destination_url,'')
FROM portal.pt_products
WHERE is_active = true
ORDER BY name
```
Cada produto retorna com `contracted: false`.

### Query com tenant (AC2) — prepared statement com $1

```sql
SELECT
    p.id,
    p.name,
    p.description,
    COALESCE(p.icon_url, ''),
    COALESCE(p.destination_url, ''),
    CASE WHEN tp.id IS NOT NULL THEN true ELSE false END AS contracted
FROM portal.pt_products p
LEFT JOIN portal.pt_tenant_products tp
    ON tp.product_id = p.id
    AND tp.tenant_id = (
        SELECT id FROM portal.pt_tenants WHERE slug = $1 LIMIT 1
    )
    AND tp.is_active = true
WHERE p.is_active = true
ORDER BY p.name
```

Se o tenant não existir, a subquery retorna NULL, o LEFT JOIN não encontra match, e todos os produtos ficam com `contracted: false` — AC3 satisfeito sem código extra.

### Registro da rota em main.go

Adicionar após a linha `http.HandleFunc("/api/auth/logout", ...)`:

```go
// Portal — público (sem auth)
http.HandleFunc("/api/portal/products", withDB(handlers.GetPortalProductsHandler))
```

### Estrutura real do projeto (learnings das stories anteriores)

- Handlers em `backend/handlers/` (NÃO em `backend/middleware/`)
- Routing em `backend/main.go` (NÃO em arquivo separado)
- `withDB` wrapper já existe em main.go — use, não recrie
- `go build ./...` a partir de `backend/`

### Rota pública (sem auth)

`GetPortalProductsHandler` usa `withDB` (não `withAuth`) — não requer JWT.
CORS já está configurado no `SecurityMiddleware` herdado.

### Referências

- [Source: epics.md#Story-2.1] — ACs completos
- [Source: backend/handlers/auth.go] — padrão de handler a seguir
- [Source: backend/main.go#266-271] — padrão de registro de rotas
- [Source: backend/migrations/100_pt_products.sql] — schema da tabela
- [Source: backend/migrations/102_pt_tenant_products.sql] — schema do join
- [Source: architecture.md#API-Communication-Patterns] — `GET /api/portal/products?tenant=`

## Dev Agent Record

### Agent Model Used

Claude Sonnet 4.6 (claude-sonnet-4-6)

### Debug Log References

Sem issues. Implementação direta.

### Completion Notes List

- AC1 ✅: GET sem tenant → todos os produtos com `contracted: false`
- AC2 ✅: GET com `?tenant=slug` → LEFT JOIN determina `contracted: true/false`
- AC3 ✅: tenant inexistente → subquery retorna NULL, LEFT JOIN sem match, todos `contracted: false`
- AC4 ✅: prepared statement com `$1` para o slug do tenant
- AC5 ✅: rota `/api/portal/products` registrada em main.go
- AC6 ✅: `go build ./...` → BUILD OK

### File List

**Criados:**
- `backend/handlers/portal_products.go` — handler GET /api/portal/products com suporte a ?tenant=

**Modificados:**
- `backend/main.go` — rota `/api/portal/products` registrada

### Change Log

| Data | Alteração |
|---|---|
| 2026-04-09 | Handler GetPortalProductsHandler criado; rota /api/portal/products registrada |
