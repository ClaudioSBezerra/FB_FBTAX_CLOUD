# Phase 2: Cadastros Base — Pattern Map

**Mapeado:** 2026-05-20
**Arquivos analisados:** 11 (3 migrations, 3 handlers Go, 2 arquivos modificados Go, 3 páginas React)
**Análogos encontrados:** 11 / 11

---

## File Classification

| Arquivo novo/modificado | Role | Data Flow | Análogo mais próximo | Qualidade |
|-------------------------|------|-----------|----------------------|-----------|
| `backend/migrations/110_financeiro_clientes.sql` | migration | batch | `backend/migrations/109_financeiro_empresas.sql` | exact |
| `backend/migrations/111_financeiro_produtos.sql` | migration | batch | `backend/migrations/109_financeiro_empresas.sql` | exact |
| `backend/migrations/112_financeiro_contratos.sql` | migration | batch | `backend/migrations/109_financeiro_empresas.sql` | exact |
| `backend/handlers/clientes.go` | handler | CRUD | `backend/handlers/financeiro.go` | exact |
| `backend/handlers/produtos.go` | handler | CRUD | `backend/handlers/financeiro.go` | exact |
| `backend/handlers/contratos.go` | handler | CRUD | `backend/handlers/financeiro.go` | exact |
| `backend/main.go` (modificar) | config | request-response | `backend/main.go` linhas 356-358 | exact |
| `frontend/src/pages/ClientesPage.tsx` | page | CRUD | `frontend/src/pages/EmpresaPage.tsx` | role-match |
| `frontend/src/pages/ProdutosPage.tsx` | page | CRUD | `frontend/src/pages/EmpresaPage.tsx` | role-match |
| `frontend/src/pages/ContratosPage.tsx` | page | CRUD | `frontend/src/pages/EmpresaPage.tsx` | role-match |
| `frontend/src/App.tsx` (modificar) | config | request-response | `frontend/src/App.tsx` linhas 36-41 | exact |

---

## Pattern Assignments

### `backend/migrations/110_financeiro_clientes.sql` (migration, batch)

**Análogo:** `backend/migrations/109_financeiro_empresas.sql`

**Padrão de tabela principal com tabela filha via FK** (linhas 1-29):
```sql
-- Tabela principal: UUID PK, campos NOT NULL explícitos, timestamps WITH TIME ZONE
CREATE TABLE IF NOT EXISTS financeiro.empresas (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    razao_social  VARCHAR(255) NOT NULL,
    nome_fantasia VARCHAR(255),
    cnpj          VARCHAR(14) NOT NULL UNIQUE,
    -- ... campos de domínio ...
    created_at    TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at    TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Tabela filha: FK com ON DELETE CASCADE, índice na FK
CREATE TABLE IF NOT EXISTS financeiro.dados_bancarios (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    empresa_id UUID NOT NULL REFERENCES financeiro.empresas(id) ON DELETE CASCADE,
    -- ... campos de domínio ...
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_dados_bancarios_empresa_id ON financeiro.dados_bancarios(empresa_id);
```

**Regras para as novas migrations:**
- Schema sempre `financeiro.` (criado em `108_financeiro_schema.sql`)
- `IF NOT EXISTS` em todas as declarações para idempotência
- UUID com `gen_random_uuid()`, nunca serial
- `VARCHAR(14)` para CNPJ, sem formatação (14 dígitos numéros)
- `ON DELETE CASCADE` nas FKs de tabelas filhas
- Índice explícito em cada FK

---

### `backend/handlers/clientes.go` (handler, CRUD)

**Análogo:** `backend/handlers/financeiro.go`

**Padrão de imports** (financeiro.go linhas 1-7):
```go
package handlers

import (
    "database/sql"
    "encoding/json"
    "net/http"
)
```
Adicionar `"log"` e `"strings"` quando necessário para busca por texto.

**Padrão de struct de request** (financeiro.go linhas 9-21):
```go
type EmpresaRequest struct {
    ID           string `json:"id,omitempty"`
    RazaoSocial  string `json:"razao_social"`
    NomeFantasia string `json:"nome_fantasia,omitempty"`
    CNPJ         string `json:"cnpj"`
    // ... demais campos
}
```

**Padrão de handler com method dispatch** (financeiro.go linhas 33-47):
```go
func EmpresaHandler(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        switch r.Method {
        case http.MethodGet:
            handleGetEmpresa(w, r, db)
        case http.MethodPost:
            handlePostEmpresa(w, r, db)
        case http.MethodPut:
            handlePutEmpresa(w, r, db)
        default:
            http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
        }
    }
}
```

**Padrão de GET lista com filtro por query param** (environment.go linhas 156-189):
```go
func GetGroupsHandler(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        envID := r.URL.Query().Get("environment_id")
        query := "SELECT id, environment_id, name, COALESCE(description, ''), created_at FROM enterprise_groups"
        args := []interface{}{}

        if envID != "" {
            query += " WHERE environment_id = $1"
            args = append(args, envID)
        }
        query += " ORDER BY name"

        rows, err := db.Query(query, args...)
        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }
        defer rows.Close()

        var groups []EnterpriseGroup
        for rows.Next() {
            var g EnterpriseGroup
            if err := rows.Scan(&g.ID, &g.EnvironmentID, &g.Name, &g.Description, &g.CreatedAt); err != nil {
                http.Error(w, err.Error(), http.StatusInternalServerError)
                return
            }
            groups = append(groups, g)
        }

        if groups == nil {
            groups = []EnterpriseGroup{}  // nunca retornar null, sempre array vazio
        }
        json.NewEncoder(w).Encode(groups)
    }
}
```

**Extensão do padrão para busca por texto (?q=) e ?status=:**
```go
// Padrão de construção de filtros acumulados (adaptar a partir de environment.go:157-166)
q := r.URL.Query().Get("q")
status := r.URL.Query().Get("status")
query := "SELECT id, nome, cnpj, status FROM financeiro.clientes WHERE 1=1"
args := []interface{}{}
argIdx := 1

if q != "" {
    query += fmt.Sprintf(" AND (nome ILIKE $%d OR cnpj LIKE $%d)", argIdx, argIdx+1)
    like := "%" + q + "%"
    args = append(args, like, like)
    argIdx += 2
}
if status != "" {
    query += fmt.Sprintf(" AND status = $%d", argIdx)
    args = append(args, status)
    argIdx++
}
query += " ORDER BY nome"
```

**Padrão de GET single por query param ?id=** (admin.go linhas 157-159):
```go
userID := r.URL.Query().Get("id")
if userID == "" {
    http.Error(w, "User ID required", http.StatusBadRequest)
    return
}
```

**Padrão de POST** (financeiro.go linhas 76-101):
```go
func handlePostEmpresa(w http.ResponseWriter, r *http.Request, db *sql.DB) {
    var req EmpresaRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "invalid request", http.StatusBadRequest)
        return
    }
    if req.RazaoSocial == "" || req.CNPJ == "" {
        http.Error(w, "razao_social and cnpj are required", http.StatusBadRequest)
        return
    }
    var id string
    err := db.QueryRow(`
        INSERT INTO financeiro.empresas (...)
        VALUES ($1, $2, ...)
        RETURNING id
    `, req.RazaoSocial, ...).Scan(&id)
    if err != nil {
        http.Error(w, "error creating empresa", http.StatusInternalServerError)
        return
    }
    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(map[string]string{"id": id})
}
```

**Padrão de PUT** (financeiro.go linhas 103-127):
```go
func handlePutEmpresa(w http.ResponseWriter, r *http.Request, db *sql.DB) {
    var req EmpresaRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "invalid request", http.StatusBadRequest)
        return
    }
    if req.ID == "" {
        http.Error(w, "id required", http.StatusBadRequest)
        return
    }
    _, err := db.Exec(`
        UPDATE financeiro.empresas
        SET campo = $1, ..., updated_at = CURRENT_TIMESTAMP
        WHERE id = $N
    `, req.Campo, ..., req.ID)
    if err != nil {
        http.Error(w, "error updating empresa", http.StatusInternalServerError)
        return
    }
    json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}
```

**Padrão sql.NullString** (financeiro.go linhas 50-73):
```go
var nomeFantasia, complemento sql.NullString
err := db.QueryRow(`SELECT ...`).Scan(&emp.ID, ..., &nomeFantasia, &complemento)
// após scan:
emp.NomeFantasia = nomeFantasia.String
emp.Complemento = complemento.String
```

---

### `backend/handlers/produtos.go` (handler, CRUD)

**Análogo:** `backend/handlers/financeiro.go` + `backend/handlers/portal_products.go`

**Padrão GET lista com struct própria** (portal_products.go linhas 10-99):
```go
type ProductResponse struct {
    ID   string `json:"id"`
    Name string `json:"name"`
    // ...
}

// produtos := []ProductResponse{}   // inicializar como slice vazio, não nil
for rows.Next() {
    var p ProductResponse
    if scanErr = rows.Scan(&p.ID, &p.Name, ...); scanErr != nil {
        http.Error(w, "internal server error", http.StatusInternalServerError)
        return
    }
    products = append(products, p)
}

if err = rows.Err(); err != nil {  // sempre verificar rows.Err() após loop
    http.Error(w, "internal server error", http.StatusInternalServerError)
    return
}
```

**Nota para produtos:** planos/preços são tabela filha. O GET de produto deve usar LEFT JOIN ou query separada para trazer planos agregados. Usar o padrão de LEFT JOIN de `portal_products.go` linhas 50-67 como referência estrutural.

---

### `backend/handlers/contratos.go` (handler, CRUD)

**Análogo:** `backend/handlers/admin.go` (padrão com transação) + `backend/handlers/financeiro.go`

**Padrão de transação (para POST de contrato com items)** (admin.go linhas 230-260):
```go
tx, err := db.Begin()
if err != nil {
    http.Error(w, "Internal server error", http.StatusInternalServerError)
    return
}
defer tx.Rollback()

// ... operações no tx ...

if err := tx.Commit(); err != nil {
    http.Error(w, "Failed to commit changes", http.StatusInternalServerError)
    return
}
```

**Padrão GET lista filtrada por cliente** — mesma abordagem de `environment.go` linhas 156-166:
```go
clienteID := r.URL.Query().Get("cliente_id")
query := "SELECT id, cliente_id, status, ... FROM financeiro.contratos WHERE 1=1"
args := []interface{}{}
if clienteID != "" {
    query += " AND cliente_id = $1"
    args = append(args, clienteID)
}
query += " ORDER BY created_at DESC"
```

**Log de operações críticas** (admin.go linha 261):
```go
log.Printf("[Contratos] Contrato %s criado para cliente %s", contratoID, req.ClienteID)
```

---

### `backend/main.go` (modificar — registrar rotas)

**Análogo:** `backend/main.go` linhas 356-358

**Padrão de registro de rota com auth** (main.go linhas 344-358):
```go
// ── Financeiro ────────────────────────────────────────────────────────────
http.HandleFunc("/api/financeiro/empresa",         withAuth(handlers.EmpresaHandler, "admin"))
http.HandleFunc("/api/financeiro/dados-bancarios", withAuth(handlers.DadosBancariosHandler, "admin"))
```

**Novas rotas a adicionar no mesmo bloco `// ── Financeiro`:**
```go
http.HandleFunc("/api/financeiro/clientes",  withAuth(handlers.ClientesHandler, "admin"))
http.HandleFunc("/api/financeiro/produtos",  withAuth(handlers.ProdutosHandler, "admin"))
http.HandleFunc("/api/financeiro/contratos", withAuth(handlers.ContratosHandler, "admin"))
```

Regras:
- Todas as rotas financeiras usam `withAuth(..., "admin")` — nunca `withDB` sem auth
- Nome da rota: kebab-case plural em português (`/clientes`, `/produtos`, `/contratos`)
- Handler name: PascalCase + sufixo `Handler` (`ClientesHandler`, `ProdutosHandler`, `ContratosHandler`)
- Inserir no bloco `// ── Financeiro ──` já existente (linha 356), mantendo ordenação alfabética dentro do bloco

---

### `frontend/src/pages/ClientesPage.tsx` (page, CRUD)

**Análogo:** `frontend/src/pages/EmpresaPage.tsx`

**Padrão de imports** (EmpresaPage.tsx linhas 1-8):
```tsx
import { useState, useEffect } from 'react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Label } from '@/components/ui/label'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { toast } from 'sonner'
```

**Adicionais para páginas com lista:**
```tsx
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Badge } from '@/components/ui/badge'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog'
```

**Padrão de interface TypeScript** (EmpresaPage.tsx linhas 10-22):
```tsx
interface Cliente {
  id?: string
  nome: string
  cnpj: string
  status: string
  // campos opcionais com ?
  email?: string
  telefone?: string
}
```

**Padrão de estado da página** (EmpresaPage.tsx linhas 34-55):
```tsx
const [items, setItems] = useState<Cliente[]>([])
const [selected, setSelected] = useState<Cliente | null>(null)
const [loading, setLoading] = useState(true)
const [submitting, setSubmitting] = useState(false)
const [error, setError] = useState<string | null>(null)
const [search, setSearch] = useState('')
```

**Padrão de fetch inicial** (EmpresaPage.tsx linhas 57-78):
```tsx
useEffect(() => {
  const fetchData = async () => {
    try {
      const res = await fetch('/api/financeiro/clientes')
      if (res.ok) {
        const data = await res.json()
        setItems(data)
      }
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Erro ao carregar dados'
      setError(msg)
    } finally {
      setLoading(false)
    }
  }
  fetchData()
}, [])
```

**Padrão de submit (POST/PUT)** (EmpresaPage.tsx linhas 80-106):
```tsx
const handleSave = async (e: React.FormEvent) => {
  e.preventDefault()
  setSubmitting(true)
  setError(null)
  try {
    const method = selected?.id ? 'PUT' : 'POST'
    const res = await fetch('/api/financeiro/clientes', {
      method,
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(selected),
    })
    const data = await res.json()
    if (!res.ok) {
      throw new Error(typeof data === 'string' ? data : 'Erro ao salvar')
    }
    toast.success('Salvo com sucesso')
    // recarregar lista
  } catch (err) {
    const msg = err instanceof Error ? err.message : 'Erro desconhecido'
    setError(msg)
    toast.error(msg)
  } finally {
    setSubmitting(false)
  }
}
```

**Padrão de loading state** (EmpresaPage.tsx linhas 140-146):
```tsx
if (loading) {
  return (
    <div className="min-h-screen p-8 flex items-center justify-center">
      Carregando...
    </div>
  )
}
```

**Padrão de layout de página** (EmpresaPage.tsx linhas 148-157):
```tsx
return (
  <div className="min-h-screen bg-gray-50 p-8">
    <div className="max-w-2xl mx-auto space-y-8">
      {error && (
        <Alert variant="destructive">
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}
      {/* conteúdo */}
    </div>
  </div>
)
```

**Padrão de formulário com campos obrigatórios vs opcionais** (EmpresaPage.tsx linhas 162-256):
```tsx
<form onSubmit={handleSave} className="space-y-4">
  <div>
    <Label htmlFor="nome">Nome *</Label>
    <Input
      id="nome"
      value={selected?.nome ?? ''}
      onChange={e => setSelected(prev => ({ ...prev!, nome: e.target.value }))}
      required
    />
  </div>
  {/* campo opcional sem required */}
  <div>
    <Label htmlFor="email">E-mail</Label>
    <Input
      id="email"
      value={selected?.email ?? ''}
      onChange={e => setSelected(prev => ({ ...prev!, email: e.target.value }))}
    />
  </div>
  <Button type="submit" disabled={submitting} className="w-full">
    {submitting ? 'Salvando...' : 'Salvar'}
  </Button>
</form>
```

**Padrão de tabela com componentes UI disponíveis** (`frontend/src/components/ui/table.tsx`):
```tsx
<Table>
  <TableHeader>
    <TableRow>
      <TableHead>Nome</TableHead>
      <TableHead>CNPJ</TableHead>
      <TableHead>Status</TableHead>
      <TableHead>Ações</TableHead>
    </TableRow>
  </TableHeader>
  <TableBody>
    {items.map(item => (
      <TableRow key={item.id}>
        <TableCell>{item.nome}</TableCell>
        <TableCell>{item.cnpj}</TableCell>
        <TableCell><Badge>{item.status}</Badge></TableCell>
        <TableCell>
          <Button variant="outline" size="sm" onClick={() => setSelected(item)}>
            Editar
          </Button>
        </TableCell>
      </TableRow>
    ))}
  </TableBody>
</Table>
```

---

### `frontend/src/pages/ProdutosPage.tsx` (page, CRUD)

**Análogo:** `frontend/src/pages/EmpresaPage.tsx`

Mesmo padrão de `ClientesPage.tsx` acima. Diferença: planos/preços são sub-seção da página, similar ao padrão de seção `DadosBancarios` em `EmpresaPage.tsx` (linhas 260-327) — seção separada dentro da mesma página, habilitada apenas após o produto estar salvo (`disabled={!produto.id}`).

---

### `frontend/src/pages/ContratosPage.tsx` (page, CRUD)

**Análogo:** `frontend/src/pages/EmpresaPage.tsx`

Mesmo padrão base. Diferença: a lista de contratos deve aceitar filtro por cliente. Usar `useSearchParams` do React Router (já presente em `PortalPage.tsx` linha 3) para ler `?cliente_id=` da URL:

```tsx
// Padrão PortalPage.tsx linhas 1-3 e 63-66
import { useSearchParams } from 'react-router-dom'

const [searchParams] = useSearchParams()
const clienteId = searchParams.get('cliente_id') ?? ''
```

---

### `frontend/src/App.tsx` (modificar — adicionar rotas)

**Análogo:** `frontend/src/App.tsx` linhas 36-41

**Padrão de rota protegida existente** (App.tsx linhas 36-41):
```tsx
{/* Financeiro */}
<Route path="/admin/financeiro/empresa" element={
  <ProtectedRoute>
    <EmpresaPage />
  </ProtectedRoute>
} />
```

**Novas rotas a adicionar no mesmo bloco `{/* Financeiro */}`:**
```tsx
<Route path="/admin/financeiro/clientes" element={
  <ProtectedRoute>
    <ClientesPage />
  </ProtectedRoute>
} />
<Route path="/admin/financeiro/produtos" element={
  <ProtectedRoute>
    <ProdutosPage />
  </ProtectedRoute>
} />
<Route path="/admin/financeiro/contratos" element={
  <ProtectedRoute>
    <ContratosPage />
  </ProtectedRoute>
} />
```

**Imports a adicionar no topo** (App.tsx linhas 6-8):
```tsx
import ClientesPage from './pages/ClientesPage'
import ProdutosPage from './pages/ProdutosPage'
import ContratosPage from './pages/ContratosPage'
```

---

## Shared Patterns

### Autenticação nos handlers Go
**Fonte:** `backend/main.go` linhas 333-342
**Aplicar a:** todos os handlers (`ClientesHandler`, `ProdutosHandler`, `ContratosHandler`)

O handler em si não precisa verificar auth — a camada `withAuth` faz isso. Porém quando o handler precisa saber quem é o usuário (ex.: auditoria), extrair claims assim:
```go
// Fonte: backend/handlers/environment.go linhas 41-47
claims, ok := r.Context().Value(ClaimsKey).(jwt.MapClaims)
if !ok {
    http.Error(w, "Unauthorized", http.StatusUnauthorized)
    return
}
userID := claims["user_id"].(string)
```
Para os handlers de cadastros base (Fase 2), `withAuth(..., "admin")` é suficiente — não é necessário extrair claims.

### Tratamento de erro Go — padrão consistente
**Fonte:** `backend/handlers/financeiro.go` (padrão completo)
**Aplicar a:** todos os handlers

| Situação | Resposta |
|----------|----------|
| Body inválido / não decodificável | `http.Error(w, "invalid request", http.StatusBadRequest)` |
| Campo obrigatório ausente | `http.Error(w, "campo_x is required", http.StatusBadRequest)` |
| Registro não encontrado | `http.StatusNotFound` + JSON `{"error": "..."}` |
| Erro de DB genérico | `http.Error(w, "internal server error", http.StatusInternalServerError)` |
| Método não permitido | `http.Error(w, "method not allowed", http.StatusMethodNotAllowed)` |

### Resposta JSON com array vazio vs null
**Fonte:** `backend/handlers/environment.go` linhas 183-186 e `backend/handlers/admin.go` linhas 139-141
**Aplicar a:** todos os handlers GET lista

```go
// Correto: nunca retornar null para arrays
if groups == nil {
    groups = []EnterpriseGroup{}
}
json.NewEncoder(w).Encode(groups)
```

### Content-Type no handler
**Fonte:** `backend/handlers/financeiro.go` linha 35
**Aplicar a:** todos os handlers

```go
// Setar no início do handler pai, antes do switch
w.Header().Set("Content-Type", "application/json")
```

### Fetch com auth no frontend (automático)
**Fonte:** `frontend/src/contexts/AuthContext.tsx` — `window.fetch` é monkey-patched globalmente
**Aplicar a:** todos os `fetch('/api/financeiro/...')` nas páginas

Não é necessário adicionar `Authorization` manualmente. O `AuthContext` injeta o header automaticamente em todas as chamadas `fetch`. Basta usar `fetch('/api/financeiro/clientes')` sem configuração extra.

### Toast de feedback
**Fonte:** `frontend/src/pages/EmpresaPage.tsx` linhas 98, 104
**Aplicar a:** todos os handlers de submit nas páginas

```tsx
import { toast } from 'sonner'  // já instalado via shadcn/ui

toast.success('Salvo com sucesso')
toast.error(msg)
```

---

## Respostas às Perguntas Específicas

1. **Handler que retorna LISTA (não singleton)?**
   Sim. `ListUsersHandler` em `admin.go` (linhas 103-146) e `GetGroupsHandler`/`GetCompaniesHandler` em `environment.go` (linhas 156-266) são os melhores análogos — retornam slice com guard `if nil { = []T{} }`.

2. **Handler com query params para filtros (?status=, ?q=)?**
   Parcialmente. `GetGroupsHandler` (environment.go linha 158) filtra por `?environment_id=` e `GetCompaniesHandler` filtra por `?group_id=`. Filtro por texto livre (`?q=` com `ILIKE`) não existe ainda — usar padrão de construção de query dinâmica descrito acima.

3. **Página React com lista/tabela de registros + busca?**
   Não existe ainda no projeto. `EmpresaPage.tsx` é o único análogo de página financeira, mas é singleton (não lista). O componente `Table` do shadcn/ui está disponível em `frontend/src/components/ui/table.tsx`. A página `ClientesPage.tsx` será a primeira do projeto a combinar lista tabelada + busca.

4. **Formulário React com campos condicionais ou multi-seção?**
   Sim. `EmpresaPage.tsx` mostra padrão de multi-seção: seção "Dados da Empresa" + seção "Dados Bancários" com campos `disabled={!empresa.id}` — campos da seção filha desabilitados até o pai ser salvo. Copiar esse padrão para `ProdutosPage.tsx` (seção de planos desabilitada até o produto ser salvo).

5. **Migração com FK para outra tabela no mesmo schema?**
   Sim. `109_financeiro_empresas.sql` linha 19: `empresa_id UUID NOT NULL REFERENCES financeiro.empresas(id) ON DELETE CASCADE`. Copiar exatamente esse padrão nas migrations 110-112.

---

## No Analog Found

Nenhum arquivo sem análogo — todos os padrões necessários existem no codebase.

---

## Metadata

**Escopo de busca de análogos:** `backend/handlers/`, `backend/migrations/`, `frontend/src/pages/`, `frontend/src/components/ui/`, `backend/main.go`, `frontend/src/App.tsx`
**Arquivos lidos:** 12
**Data de extração de padrões:** 2026-05-20
