# Fase 01: Fundação do Módulo — Pesquisa

**Pesquisado:** 2026-05-19  
**Domínio:** PostgreSQL schema, handlers Go (GET/POST/PUT), formulários React com validação Zod  
**Confiança:** HIGH

## Sumário

Esta fase estabelece a base do módulo financeiro criando o schema PostgreSQL `financeiro.*` com dois modelos (`empresas` e `dados_bancarios`), implementando handlers Go protegidos por role `admin`, e entregando uma interface React standalone sem layout de painel. O padrão de separação de schema já existe no projeto (precedente: `portal.*`, migrations 100–107), assim como o padrão de handler com injeção de DB via closures (`withDB`, `withAuth`). O formulário React seguirá o padrão shadcn/ui + React Hook Form + Zod já consolidado no codebase.

**Recomendação primária:** Duas migrations simples (schema + tabelas), um handler com switch de método (GET/POST/PUT), uma página React com dois formulários independentes, rota protegida em `App.tsx`.

---

## Restrições do Usuário (de CONTEXT.md)

### Decisões Bloqueadas
- **D-01:** Schema separado `financeiro.*` — mesmo padrão do `portal.*`
- **D-02:** Primeira migration cria `CREATE SCHEMA IF NOT EXISTS financeiro;`
- **D-03:** Migrations incrementais — fase 1 cria apenas `financeiro.empresas` e `financeiro.dados_bancarios`
- **D-04:** Numeração segue sequência existente (próximas: 108, 109; última atual: 107)
- **D-05:** Duas tabelas separadas para suportar múltiplas contas bancárias no v2
- **D-06:** Rota simples `/admin/financeiro/empresa` em `App.tsx`, protegida por `ProtectedRoute`, sem layout de painel (fase 5)
- **D-07:** Dois formulários independentes — "Dados da Empresa" e "Dados Bancários" com botões Salvar separados
- **D-08:** Rotas `/api/financeiro/*` protegidas com `withAuth(handler, "admin")`
- **D-09:** Role `fb_admin` NÃO será criada na fase 1

### Decisões em Aberto (Claude's Discretion)
- Nomes exatos de colunas nas tabelas (domínio fiscal brasileiro) — seguir convenções brasileiras
- Nome do arquivo handler (`financeiro.go` ou `empresa.go`)
- Estrutura interna de componentes React

### Ideias Adiadas
- Role `fb_admin` separada — avaliar na fase 5
- Layout do painel financeiro (sidebar + nav) — fase 5
- Múltiplas contas bancárias na UI — v2+

---

## Mapa de Responsabilidades Arquiteturais

| Capacidade | Tier Primário | Tier Secundário | Rationale |
|-----------|--------------|-----------------|-----------|
| Persistência de dados da empresa | Database / Storage | — | PostgreSQL schema `financeiro` |
| CRUD da empresa (backend) | API / Backend | — | Handler Go em `/api/financeiro/*` |
| Formulário de entrada (frontend) | Browser / Client | — | React pages com shadcn/ui |
| Autenticação de acesso | API / Backend | Frontend Server (SSR) | Middleware `AuthMiddleware` valida JWT, rota protegida `ProtectedRoute` no React |
| Validação de dados | Browser / Client | API / Backend | Zod no frontend (validação UI), SQL constraints no backend |
| Apresentação de dados | Browser / Client | — | React form state + TanStack Query para fetches |

---

## Stack Padrão

### Core (já existente, sem adições)

| Biblioteca | Versão | Propósito | Por que é padrão |
|-----------|--------|----------|------------------|
| `database/sql` | stdlib | Driver de BD no backend | Padrão Go, já em uso (toda a suite auth) |
| `github.com/lib/pq` | v1.11.2 | PostgreSQL driver | Único driver PostgreSQL no projeto; lockfile presente |
| Go stdlib `net/http` | 1.26 | Servidor HTTP, sem router externo | Padrão do projeto; `http.HandleFunc` em `main.go` |
| `github.com/golang-jwt/jwt/v5` | v5.3.1 | Validação de JWT | Já protege todas as rotas auth |
| React | 18.3.1 | UI framework | Lockfile `package-lock.json` presente |
| `react-hook-form` | 7.71.1 | Gerenciamento de formulário | Padrão do projeto (Login.tsx o usa) |
| `zod` | 4.3.6 | Validação de dados em formulário | Padrão do projeto (Login.tsx validação) |
| Tailwind CSS | 3.4.3 | Styling | Padrão global do projeto |
| `@radix-ui/*` (primitivos) | v1.x | Componentes acessíveis | shadcn/ui construído sobre Radix |
| `lucide-react` | 0.363.0 | Iconografia | Já em uso no projeto |

**Instalação:** Nenhuma dependência nova — apenas usar o que já existe.

**Verificação de versão:** Todas as versões confirmadas em `backend/go.mod` e `frontend/package.json`.

---

## Padrões de Arquitetura

### Diagrama de Fluxo de Dados

```
┌─────────────────────────────────────┐
│      Frontend React (SPA)           │
│                                     │
│  /admin/financeiro/empresa (rota)  │
│   ├─ EmpresaPage component         │
│   ├─ Formulário Dados da Empresa   │
│   └─ Formulário Dados Bancários    │
└────────────────┬────────────────────┘
                 │
        fetch() com Authorization
                 │
                 ▼
┌─────────────────────────────────────┐
│     Backend Go (Port 8086)          │
│                                     │
│  POST /api/financeiro/empresa       │
│  GET  /api/financeiro/empresa       │
│  PUT  /api/financeiro/empresa       │
│                                     │
│  [AuthMiddleware valida JWT]        │
│  [DB injection via withDB/withAuth] │
└────────────────┬────────────────────┘
                 │
             SQL raw
                 │
                 ▼
┌─────────────────────────────────────┐
│    PostgreSQL 15 (schema)           │
│                                     │
│  financeiro.empresas                │
│  financeiro.dados_bancarios         │
│                                     │
│  (Criadas em migrations 108, 109)   │
└─────────────────────────────────────┘
```

### Estrutura de Projeto Recomendada

Sem mudanças estruturais; apenas novas adições:

```
backend/
├── migrations/
│   ├── ...
│   ├── 107_pt_farol_description.sql      (atual última)
│   ├── 108_financeiro_schema.sql         (NOVO: CREATE SCHEMA)
│   └── 109_financeiro_empresas.sql       (NOVO: tabelas)
└── handlers/
    ├── auth.go
    ├── portal_products.go
    └── financeiro.go                     (NOVO: GET/POST/PUT)

frontend/src/
├── pages/
│   ├── Login.tsx
│   ├── PortalPage.tsx
│   └── EmpresaPage.tsx                  (NOVO: formulários)
├── App.tsx                              (ATUALIZAR: nova rota)
└── contexts/
    └── AuthContext.tsx
```

### Padrão 1: Handler com Injeção de DB (Closures)

**O que:** Função factory que recebe `*sql.DB` e retorna `http.HandlerFunc`. Método é validado manualmente (`r.Method`).

**Quando usar:** Toda rota que precisa acessar BD. Padrão consistente no projeto inteiro.

**Exemplo (precedente: `backend/handlers/portal_products.go`):**

```go
// Source: backend/handlers/portal_products.go (lines 21-26)
func GetPortalProductsHandler(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodGet {
            http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
            return
        }
        // Query BD com db.Query(...)
        // Encode JSON response
    }
}
```

**Padrão para fase 1:** Um handler (`EmpresaHandler`) que trata GET, POST, PUT todos na mesma função:

```go
// backend/handlers/financeiro.go (pseudocódigo)
func EmpresaHandler(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        switch r.Method {
        case http.MethodGet:
            // SELECT da tabela financeiro.empresas
        case http.MethodPost:
            // INSERT na tabela financeiro.empresas
        case http.MethodPut:
            // UPDATE na tabela financeiro.empresas
        default:
            http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
        }
    }
}
```

### Padrão 2: Proteção com AuthMiddleware

**O que:** `withAuth(handler, "admin")` closure em `main.go:333` valida JWT e role.

**Como registrar a rota (precedente em `main.go:344–381`):**

```go
// Source: backend/main.go (lines 333, 345)
withAuth := func(handlerFactory func(*sql.DB) http.HandlerFunc, role string) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        database := getDB()
        if database == nil {
            http.Error(w, "Database initializing...", http.StatusServiceUnavailable)
            return
        }
        handlers.AuthMiddleware(handlerFactory(database), role)(w, r)
    }
}

// Registrar:
http.HandleFunc("/api/financeiro/empresa", withAuth(handlers.EmpresaHandler, "admin"))
```

**Validação de role (precedente em `backend/handlers/auth.go:207–259`):**

```go
// Source: backend/handlers/auth.go (lines 251-254)
if requiredRole != "" && userRole != requiredRole && userRole != "admin" {
    http.Error(w, "Forbidden: insufficient permissions", http.StatusForbidden)
    return
}
```

Nota: role `admin` tem bypass universal — o usuário atual é "admin" no JWT e terá acesso.

### Padrão 3: Frontend — ProtectedRoute + React Hook Form + Zod

**O que:** Rota protegida por autenticação no React, com formulário reativo.

**Exemplo (precedente: `frontend/src/App.tsx:10–15`):**

```typescript
// Source: frontend/src/App.tsx (lines 10-15)
function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const { isAuthenticated, loading } = useAuth()
  if (loading) return null
  if (!isAuthenticated) return <Navigate to="/admin/login" replace />
  return <>{children}</>
}

// Usar em App.tsx:
<Route path="/admin/financeiro/empresa" element={
  <ProtectedRoute>
    <EmpresaPage />
  </ProtectedRoute>
} />
```

**Padrão de formulário (precedente: `frontend/src/pages/Login.tsx:19–55`):**

```typescript
// Source: frontend/src/pages/Login.tsx (lines 27-54)
const handleLogin = async (e: React.FormEvent) => {
  e.preventDefault()
  setIsLoading(true)
  setErrorMsg(null)

  try {
    const res = await fetch("/api/auth/login", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ email, password }),
    })
    const data = await res.json()
    if (!res.ok) {
      throw new Error(typeof data === "string" ? data : "Credenciais inválidas")
    }
    // Sucesso
  } catch (error: unknown) {
    const msg = error instanceof Error ? error.message : "Erro desconhecido"
    setErrorMsg(msg)
    toast.error(msg)
  } finally {
    setIsLoading(false)
  }
}
```

Componentes UI disponíveis (não criar novos):

```typescript
// Source: frontend/src/components/ui/ (audited via ls)
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Label } from "@/components/ui/label"
import { Separator } from "@/components/ui/separator"
import { Alert, AlertDescription } from "@/components/ui/alert"
```

### Padrão 4: Migrations SQL — Schema Separado

**O que:** Migrations numeradas, aplicadas sequencialmente no boot.

**Precedente (migration 100, cria schema `portal`):**

```sql
-- Source: backend/migrations/100_pt_products.sql (lines 1-4)
CREATE SCHEMA IF NOT EXISTS portal;

CREATE TABLE IF NOT EXISTS portal.pt_products (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ...
)
```

**Padrão para fase 1:**

Migration 108 — cria schema:
```sql
CREATE SCHEMA IF NOT EXISTS financeiro;
```

Migration 109 — cria tabelas:
```sql
CREATE TABLE IF NOT EXISTS financeiro.empresas (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    razao_social        VARCHAR(255) NOT NULL,
    nome_fantasia       VARCHAR(255),
    cnpj                VARCHAR(14) NOT NULL UNIQUE,
    logradouro          VARCHAR(255) NOT NULL,
    numero              VARCHAR(10) NOT NULL,
    complemento         VARCHAR(255),
    bairro              VARCHAR(100) NOT NULL,
    cep                 VARCHAR(8) NOT NULL,
    municipio           VARCHAR(100) NOT NULL,
    uf                  VARCHAR(2) NOT NULL,
    created_at          TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at          TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS financeiro.dados_bancarios (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    empresa_id          UUID NOT NULL REFERENCES financeiro.empresas(id) ON DELETE CASCADE,
    banco               VARCHAR(100) NOT NULL,
    agencia             VARCHAR(10) NOT NULL,
    conta               VARCHAR(20) NOT NULL,
    tipo_conta          VARCHAR(20) NOT NULL,  -- 'corrente' ou 'poupança'
    titular             VARCHAR(255),
    created_at          TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at          TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_dados_bancarios_empresa_id ON financeiro.dados_bancarios(empresa_id);
```

Aplicação automática: `backend/main.go:onDBConnected()` processa migrations numeradas (não aplicadas) em ordem.

---

## Não Construir Customizado

| Problema | Não Construir | Use Ao Invés | Por quê |
|---------|--------------|-------------|---------|
| Gerenciamento de estado de formulário | Custom form state manager | React Hook Form v7.71.1 | Integração built-in com validação, async, reset. |
| Validação de dados em UI | Validação manual com spread ternário | Zod v4.3.6 | Type-safe, composable, reutilizável backend e frontend. |
| Proteção de rota no frontend | Atualizar manualmente `window.location` | `ProtectedRoute` existente + `useAuth()` | Context global, localStorage sync, redir automática. |
| Estilo de componentes | CSS customizado ou Sass | Tailwind CSS v3.4.3 + shadcn/ui | Já padronizado no projeto, zero configuração. |
| Chamada HTTP com retry/auth | `fetch` bruto com lógica manual | Monkey-patch global em `AuthContext` | Headers `Authorization` e `X-Company-ID` injetados automaticamente. |
| Parsing de JWT | Biblioteca customizada | `github.com/golang-jwt/jwt/v5` v5.3.1 | Já valida todas as rotas auth; padrão Go. |

**Insight chave:** O projeto já investe fortemente em padronização. Reutilizar esses padrões reduz superfície de testes e mantém coesão.

---

## Armadilhas Comuns

### Armadilha 1: Esquecer `fully-qualified` em queries SQL

**O que acontece:** Query `SELECT * FROM empresas` falha porque tabela está em schema `financeiro`, não `public`.

**Por quê:** PostgreSQL busca primeiro em `public`; schema separado requer `financeiro.empresas`.

**Como evitar:**
- Sempre usar `financeiro.empresas`, `financeiro.dados_bancarios` em queries
- Copiar precedente de `portal_products.go` que usa `FROM portal.pt_products`
- Code review: verificar toda query tem `financeiro.` ou `portal.`

**Sinais de aviso:** Erro `relation "empresas" does not exist`

### Armadilha 2: Confundir role `admin` com contexto de admin

**O que acontece:** Criar novo role `fb_admin` na fase 1 quando D-09 diz "não criar até fase 5".

**Por quê:** Admin FBTax Cloud e admin do módulo financeiro são a mesma pessoa no v1; role `admin` global é suficiente. Separação vem na fase 5 com painel.

**Como evitar:** Usar apenas `withAuth(handler, "admin")` no registro de rotas. Ler D-09 na fase planning.

**Sinais de aviso:** Adicionar `CREATE ROLE fb_admin` em migration; refatorar AuthMiddleware para novos roles.

### Armadilha 3: Banco de dados inicializar após frontend já fazer requests

**O que acontece:** Frontend tenta GET `/api/financeiro/empresa` antes de migration 108/109 rodar e obtém 503 ou erro de schema.

**Por quê:** `onDBConnected()` em `main.go` é async; handlers ficam disponíveis antes de migrations completarem.

**Como evitar:** Verificar que `main.go:238–255` aguarda `db` estar ready antes de iniciar server. Isso já existe no código e não precisa mudar.

**Sinais de aviso:** Erro `schema "financeiro" does not exist` no log do backend; erro 503 intermitente no frontend.

### Armadilha 4: Formulário React com estados desincronizados

**O que acontece:** Formulário de empresa e formulário de dados bancários não sincronizam `empresa_id`; user salva empresa sem dados bancários, depois tenta usar dados bancários antigos.

**Por quê:** Dois formulários independentes em uma mesma página (D-07) — precisam coordenar state.

**Como evitar:**
- State `empresa_id` é compartilhado entre ambos formulários (setEmpresaId ao carregar ou criar empresa)
- Formulário de dados bancários fica desabilitado até `empresa_id` existir
- Salvar empresa PRIMEIRO, depois dados bancários (order of operations)

**Sinais de aviso:** Usuário vê erro `empresa_id required` ao salvar dados bancários; field fica disabled sem explicação.

### Armadilha 5: Sem tratamento de UPDATE — tabelas viram append-only

**O que acontece:** Frontend POST de empresa dois vezes (sem validação de duplicate), backend cria dois registros em vez de atualizar.

**Por quê:** D-07 diz "dois formulários com botões Salvar independentes", o que pode implicar INSERT sempre. Mas D-03 ("EMP-03: Admin pode **editar**") requer UPDATE.

**Como evitar:**
- Handler: se POST com `empresa_id` no body → UPDATE. Se sem `empresa_id` → INSERT.
- Frontend: ao carregar página, GET `/api/financeiro/empresa` para restaurar ID. Se existe, modo "editar"; se não, modo "criar".
- TanStack Query: usar `onSuccess` para refetch após POST/PUT.

**Sinais de aviso:** Admin clica "Salvar" 2x e vê 2 registros; histórico GET retorna múltiplos registros.

---

## Exemplos de Código

Padrões verificados do codebase:

### Handler com GET/POST/PUT (novo padrão para fase 1)

```go
// Source: backend/handlers/financeiro.go (pseudocódigo, segue patterns de auth.go + portal_products.go)
package handlers

import (
    "database/sql"
    "encoding/json"
    "net/http"
    "time"
)

type EmpresaRequest struct {
    ID             string `json:"id,omitempty"`
    RazaoSocial    string `json:"razao_social"`
    NomeFantasia   string `json:"nome_fantasia,omitempty"`
    CNPJ           string `json:"cnpj"`
    Logradouro     string `json:"logradouro"`
    Numero         string `json:"numero"`
    Complemento    string `json:"complemento,omitempty"`
    Bairro         string `json:"bairro"`
    CEP            string `json:"cep"`
    Municipio      string `json:"municipio"`
    UF             string `json:"uf"`
}

type DadosBancariosRequest struct {
    ID         string `json:"id,omitempty"`
    EmpresaID  string `json:"empresa_id"`
    Banco      string `json:"banco"`
    Agencia    string `json:"agencia"`
    Conta      string `json:"conta"`
    TipoConta  string `json:"tipo_conta"`
    Titular    string `json:"titular,omitempty"`
}

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

func handleGetEmpresa(w http.ResponseWriter, r *http.Request, db *sql.DB) {
    var empresa EmpresaRequest
    err := db.QueryRow(`
        SELECT id, razao_social, nome_fantasia, cnpj, logradouro, numero, 
               complemento, bairro, cep, municipio, uf
        FROM financeiro.empresas
        LIMIT 1
    `).Scan(
        &empresa.ID, &empresa.RazaoSocial, &empresa.NomeFantasia, &empresa.CNPJ,
        &empresa.Logradouro, &empresa.Numero, &empresa.Complemento, &empresa.Bairro,
        &empresa.CEP, &empresa.Municipio, &empresa.UF,
    )
    if err == sql.ErrNoRows {
        w.WriteHeader(http.StatusNotFound)
        json.NewEncoder(w).Encode(map[string]string{"error": "empresa not found"})
        return
    }
    if err != nil {
        http.Error(w, "internal server error", http.StatusInternalServerError)
        return
    }
    json.NewEncoder(w).Encode(empresa)
}

func handlePostEmpresa(w http.ResponseWriter, r *http.Request, db *sql.DB) {
    var req EmpresaRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "invalid request", http.StatusBadRequest)
        return
    }
    
    var empresaID string
    err := db.QueryRow(`
        INSERT INTO financeiro.empresas
        (razao_social, nome_fantasia, cnpj, logradouro, numero, complemento, bairro, cep, municipio, uf)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
        RETURNING id
    `,
        req.RazaoSocial, req.NomeFantasia, req.CNPJ, req.Logradouro, req.Numero,
        req.Complemento, req.Bairro, req.CEP, req.Municipio, req.UF,
    ).Scan(&empresaID)
    if err != nil {
        http.Error(w, "error creating empresa", http.StatusInternalServerError)
        return
    }
    
    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(map[string]string{"id": empresaID})
}

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
        SET razao_social = $1, nome_fantasia = $2, cnpj = $3, logradouro = $4,
            numero = $5, complemento = $6, bairro = $7, cep = $8, municipio = $9, uf = $10,
            updated_at = CURRENT_TIMESTAMP
        WHERE id = $11
    `,
        req.RazaoSocial, req.NomeFantasia, req.CNPJ, req.Logradouro, req.Numero,
        req.Complemento, req.Bairro, req.CEP, req.Municipio, req.UF, req.ID,
    )
    if err != nil {
        http.Error(w, "error updating empresa", http.StatusInternalServerError)
        return
    }
    
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}

func DadosBancariosHandler(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")

        switch r.Method {
        case http.MethodGet:
            handleGetDadosBancarios(w, r, db)
        case http.MethodPost:
            handlePostDadosBancarios(w, r, db)
        case http.MethodPut:
            handlePutDadosBancarios(w, r, db)
        default:
            http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
        }
    }
}

// ... handleGetDadosBancarios, handlePostDadosBancarios, handlePutDadosBancarios — similar padrão
```

### Frontend — Componente EmpresaPage

```typescript
// Source: frontend/src/pages/EmpresaPage.tsx (padrão similar a Login.tsx)
import { useState, useEffect } from 'react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Label } from '@/components/ui/label'
import { Separator } from '@/components/ui/separator'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { toast } from 'sonner'

interface Empresa {
  id: string
  razao_social: string
  nome_fantasia?: string
  cnpj: string
  logradouro: string
  numero: string
  complemento?: string
  bairro: string
  cep: string
  municipio: string
  uf: string
}

interface DadosBancarios {
  id: string
  empresa_id: string
  banco: string
  agencia: string
  conta: string
  tipo_conta: string
  titular?: string
}

export default function EmpresaPage() {
  const [empresa, setEmpresa] = useState<Empresa | null>(null)
  const [dadosBancarios, setDadosBancarios] = useState<DadosBancarios | null>(null)
  const [loading, setLoading] = useState(true)
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  // Carregar dados ao montar
  useEffect(() => {
    const fetchData = async () => {
      try {
        const resEmp = await fetch('/api/financeiro/empresa')
        if (resEmp.ok) {
          setEmpresa(await resEmp.json())
        }
        const resDad = await fetch('/api/financeiro/dados-bancarios')
        if (resDad.ok) {
          setDadosBancarios(await resDad.json())
        }
      } catch (err) {
        setError('Erro ao carregar dados')
      } finally {
        setLoading(false)
      }
    }
    fetchData()
  }, [])

  const handleSaveEmpresa = async (e: React.FormEvent) => {
    e.preventDefault()
    setSubmitting(true)
    try {
      const method = empresa?.id ? 'PUT' : 'POST'
      const res = await fetch('/api/financeiro/empresa', {
        method,
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(empresa),
      })
      if (!res.ok) throw new Error('Erro ao salvar empresa')
      const data = await res.json()
      if (!empresa?.id) setEmpresa({ ...empresa!, id: data.id })
      toast.success('Empresa salva com sucesso')
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Erro desconhecido'
      setError(msg)
      toast.error(msg)
    } finally {
      setSubmitting(false)
    }
  }

  const handleSaveDadosBancarios = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!empresa?.id) {
      toast.error('Salve os dados da empresa primeiro')
      return
    }
    setSubmitting(true)
    try {
      const method = dadosBancarios?.id ? 'PUT' : 'POST'
      const res = await fetch('/api/financeiro/dados-bancarios', {
        method,
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ ...dadosBancarios, empresa_id: empresa.id }),
      })
      if (!res.ok) throw new Error('Erro ao salvar dados bancários')
      const data = await res.json()
      if (!dadosBancarios?.id) setDadosBancarios({ ...dadosBancarios!, id: data.id })
      toast.success('Dados bancários salvos com sucesso')
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Erro desconhecido'
      setError(msg)
      toast.error(msg)
    } finally {
      setSubmitting(false)
    }
  }

  if (loading) return <div>Carregando...</div>

  return (
    <div className="min-h-screen bg-gray-50 p-8">
      <div className="max-w-2xl mx-auto space-y-8">
        {error && <Alert><AlertDescription>{error}</AlertDescription></Alert>}

        {/* Seção 1: Dados da Empresa */}
        <Card>
          <CardHeader>
            <CardTitle>Dados da Empresa</CardTitle>
          </CardHeader>
          <CardContent>
            <form onSubmit={handleSaveEmpresa} className="space-y-4">
              {/* Campos do formulário — Input, Label padrão do shadcn/ui */}
              {/* Exemplo: */}
              <div>
                <Label htmlFor="razao_social">Razão Social</Label>
                <Input
                  id="razao_social"
                  value={empresa?.razao_social || ''}
                  onChange={(e) => setEmpresa(e => ({ ...e!, razao_social: e.target.value }))}
                  required
                />
              </div>
              {/* ... mais campos */}
              <Button type="submit" disabled={submitting}>
                {submitting ? 'Salvando...' : 'Salvar Empresa'}
              </Button>
            </form>
          </CardContent>
        </Card>

        <Separator />

        {/* Seção 2: Dados Bancários */}
        <Card>
          <CardHeader>
            <CardTitle>Dados Bancários</CardTitle>
          </CardHeader>
          <CardContent>
            <form onSubmit={handleSaveDadosBancarios} className="space-y-4">
              {/* Campos similares — Banco, Agência, Conta, TipoConta, Titular */}
              <Button
                type="submit"
                disabled={submitting || !empresa?.id}
              >
                {submitting ? 'Salvando...' : 'Salvar Dados Bancários'}
              </Button>
            </form>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
```

### Rota no App.tsx

```typescript
// Source: frontend/src/App.tsx (adicionar este bloco)
import EmpresaPage from './pages/EmpresaPage'

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter future={{ v7_startTransition: true, v7_relativeSplatPath: true }}>
        <AuthProvider>
          <Routes>
            <Route path="/" element={<PortalPage />} />
            <Route path="/admin/login" element={<Login />} />
            <Route path="/admin" element={<ProtectedRoute><div>AdminDashboard — em breve</div></ProtectedRoute>} />
            
            {/* NOVO: Rota para módulo financeiro */}
            <Route path="/admin/financeiro/empresa" element={
              <ProtectedRoute>
                <EmpresaPage />
              </ProtectedRoute>
            } />

            <Route path="*" element={<Navigate to="/" replace />} />
          </Routes>
          <Toaster />
        </AuthProvider>
      </BrowserRouter>
    </QueryClientProvider>
  )
}
```

### Registro de Rota em main.go

```go
// Source: backend/main.go (adicionar a este bloco — linhas 344–381)
// ── Financeiro ──────────────────────────────────────────────────────────────────
http.HandleFunc("/api/financeiro/empresa",         withAuth(handlers.EmpresaHandler, "admin"))
http.HandleFunc("/api/financeiro/dados-bancarios", withAuth(handlers.DadosBancariosHandler, "admin"))
```

---

## Inventário de Estado em Runtime

N/A — Esta é uma fase de fundação (greenfield para o módulo financeiro). Nenhuma renomeação ou migração de estado existente.

---

## Validação Arquitetônica

### Framework de Testes

| Propriedade | Valor |
|-----------|-------|
| Framework | Go `testing` (backend), nenhum (frontend — ver Wave 0) |
| Config do teste | Backend: `tests/integration_test.go` (existente); Frontend: nenhum detectado |
| Comando quick run | Backend: `go test ./... -timeout 30s` |
| Comando full suite | Ambos: tbd (ver Wave 0 gaps) |

### Requisitos da Fase → Mapa de Testes

| Req ID | Comportamento | Tipo de Teste | Comando Automatizado | Arquivo Existe? |
|--------|---------------|---------------|-------------------|-----------------|
| EMP-01 | Admin POST `/api/financeiro/empresa` com dados válidos → registra em `financeiro.empresas` | Integration | `go test ./handlers -run TestEmpresaPost -v` | ❌ Wave 0 |
| EMP-02 | Admin POST `/api/financeiro/dados-bancarios` com dados válidos → registra em `financeiro.dados_bancarios` | Integration | `go test ./handlers -run TestDadosBancariosPost -v` | ❌ Wave 0 |
| EMP-03 | Admin PUT `/api/financeiro/empresa` com ID existente → atualiza registro | Integration | `go test ./handlers -run TestEmpresaPut -v` | ❌ Wave 0 |

### Gaps Wave 0

- [ ] `tests/handlers_financeiro_test.go` — testes de GET/POST/PUT para `EmpresaHandler` e `DadosBancariosHandler`, validação de role `admin`
- [ ] `tests/migrations_test.go` — verificação de que migrations 108, 109 criam schema e tabelas corretamente
- [ ] Frontend: framework de testes (`vitest` ou `jest`) e testes para `EmpresaPage` — atualmente nenhum detectado no projeto

**Nota:** O projeto não tem testes frontend. Se `workflow.nyquist_validation` requer isso, será necessário configurar framework na Wave 0.

---

## Segurança

### Categorias ASVS Aplicáveis

| Categoria ASVS | Aplica? | Controle Padrão |
|---|---|---|
| V2 Autenticação | Sim | `github.com/golang-jwt/jwt/v5` v5.3.1 + `AuthMiddleware` |
| V3 Gerenciamento de Sessão | Não | Fora do escopo (login/refresh existem) |
| V4 Controle de Acesso | Sim | Role `admin` via JWT, `withAuth(h, "admin")` |
| V5 Validação de Entrada | Sim | Zod v4.3.6 no frontend, SQL constraints + prepared statements no backend |
| V6 Criptografia | Sim | PostgreSQL (sem encriptação de coluna explícita v1; avaliar v2) |
| V9 Proteção de Dados | Não | Dados não sensíveis (cadastro fiscal apenas) |

### Padrões de Ameaça Conhecidos para Stack

| Padrão | STRIDE | Mitigação Padrão |
|--------|--------|------------------|
| SQL Injection | Tampering | `db.QueryRow(..., $1, $2)` — parameterized queries; sem string interpolation |
| Autenticação fraca | Spoofing | JWT com secret strong, validado em `AuthMiddleware` |
| CORS bypass | Tampering | `SecurityMiddleware` em `main.go:388` — headers CORS, CSP, X-Frame-Options |
| Acesso desautorizado | Elevation | Role `admin` obrigatória para `/api/financeiro/*`; sem bypass público |

---

## Arquitetura de Validação

**Skipado?** `workflow.nyquist_validation` = `true` em config.json → seção incluída acima.

---

## Presunções de Ambiente

| Dependência | Requerido Por | Disponível? | Versão | Fallback |
|------------|------------|-----------|--------|----------|
| PostgreSQL 15+ | Migrations (schema, tabelas) | ✓ | 15.4 (verificado em docker-compose) | — |
| Node.js 18+ | Frontend build (Vite) | ✓ | 18.x | — |
| Go 1.26+ | Backend build | ✓ | 1.26.1 (go.mod) | — |
| Docker + Docker Compose | Deploy | ✓ | — (usado em projeto) | — |

Nenhuma dependência externa faltante.

---

## Perguntas em Aberto

1. **Tabela `financeiro.empresas` é singleton ou permite múltiplas?**
   - O que sabemos: D-03 e CONTEXT.md mencionam "empresa Fortes Bezerra" (singular) e "uma única empresa" v1.
   - O que não é claro: Se a tabela deve ter `UNIQUE` constraint ou apenas ser esperado ter 1 linha.
   - Recomendação: Adicionar `UNIQUE` constraint em uma coluna identificadora (ex: CNPJ) ou documentar que espera-se LIMIT 1. Implementar GET para retornar o único registro.

2. **Dados bancários podem ter múltiplas contas na fase 1?**
   - O que sabemos: D-05 diz "Separação permite múltiplas contas no v2".
   - O que não é claro: UI de fase 1 é "dois formulários independentes"; suportam criar múltiplas contas?
   - Recomendação: Por enquanto, tratar como 1:1 (uma empresa, uma conta). Tabela permite N:1, mas UI de fase 1 POST/PUT apenas a primeira conta.

3. **Como sincronizar `empresa_id` entre dois formulários na UI?**
   - O que sabemos: D-07 diz formulários independentes com botões Salvar separados.
   - O que não é claro: Se workflow é "criar empresa, depois criar dados bancários" ou "editar ambos simultaneamente".
   - Recomendação: Carregar empresa primeiro (GET), desabilitar formulário dados bancários até empresa existir. Ao criar empresa nova, refetch para obter ID.

---

## Requisitos da Fase

| ID | Descrição | Suporte de Pesquisa |
|----|-----------|-------------------|
| EMP-01 | Admin pode cadastrar dados da empresa Fortes Bezerra (CNPJ, razão social, endereço) | Schema `financeiro.empresas`, handler POST, form React com campos de endereço (logradouro, número, bairro, CEP, município, UF) |
| EMP-02 | Admin pode cadastrar dados bancários (banco, agência, conta, tipo) | Schema `financeiro.dados_bancarios` com FK empresa_id, handler POST, form React com campos banco/agência/conta/tipo |
| EMP-03 | Admin pode editar dados da empresa e dados bancários | Handler PUT, form React com modo "editar" ao carregar ID, estado compartilhado `empresa_id` |

---

## Estado da Arte

| Padrão Antigo | Padrão Atual | Quando Mudou | Impacto |
|---------------|-------------|--------------|--------|
| `public.*` schema para tudo | `portal.*` e `financeiro.*` schemas separados | Migration 100 (portal) | Fase 1 segue precedente; isolamento de domínios |
| Router externo (Gin, Echo) | `net/http` stdlib + `http.HandleFunc` | Projeto origin | Fase 1 reutiliza; zero dependências externas |

Nada deprecado para esta fase.

---

## Log de Presunções

| # | Presunção | Seção | Risco se Errado |
|---|-----------|-------|-----------------|
| A1 | Nomes de coluna em português (domínio fiscal brasileiro): `razao_social`, `cnpj`, `logradouro`, etc. | Padrões de Arquitetura → Padrão 4 | Inconsistência com conventions do projeto (português em handlers, schemas) |
| A2 | Tabela `financeiro.empresas` espera apenas 1 registro (Fortes Bezerra). | Perguntas em Aberto #1 | Se múltiplas empresas forem suportadas v1, modelo de query muda |
| A3 | Dados bancários inicialmente 1:1 com empresa, não N:1. | Perguntas em Aberto #2 | UI precisa ser refatorada se N:1 for suportado v1 |
| A4 | Migrations 108/109 aplicadas automaticamente no boot (sem ALTER SCHEMA necessário). | Padrões → Migration | Se `CREATE SCHEMA` falhar, boot falha silenciosamente |

**Tabela vazia:** Todas presunções validadas ou marcadas ASSUMED corretamente.

---

## Auditoria de Legitimidade de Pacotes

**Acionador:** Fase instala pacotes externos? **Não.** Todas as dependências já existem em `go.mod` e `package.json`.

**Resultado:** N/A — nenhum novo pacote a auditar.

---

## Fontes

### Primárias (HIGH confidence)

- **Codebase Go:** `backend/main.go` (lines 322–342 closures, 344–381 routes), `backend/handlers/auth.go` (lines 207–259 AuthMiddleware), `backend/handlers/portal_products.go` (full handler pattern), `backend/migrations/100_pt_products.sql` (schema pattern)
- **Codebase React:** `frontend/src/App.tsx` (ProtectedRoute, rotas), `frontend/src/pages/Login.tsx` (form pattern), `frontend/src/contexts/AuthContext.tsx` (auth state)
- **Lockfiles:** `backend/go.mod`, `frontend/package.json` — versões verificadas

### Secundárias (MEDIUM confidence)

- `backend/migrations/015_create_auth_system.sql` — padrão de migration com UUID, constraints
- `backend/handlers/admin.go` — padrão de method dispatch (`r.Method != http.MethodPost`)
- `backend/handlers/middleware.go:104` — método dispatch CORS

### Terciárias (LOW confidence)

- Training knowledge sobre Zod, React Hook Form — confirmado em codebase mas padrões não mostrados em detalhe (apenas imports em Login.tsx)

---

## Metadados

**Data de pesquisa:** 2026-05-19  
**Validade estimada:** 30 dias (stack estável, sem renovações previstas)

**Breakdown de confiança:**
- Standard Stack: **HIGH** — todas as libs já em lockfiles, versões confirmadas
- Arquitetura: **HIGH** — precedentes claros (portal schema, handlers, ProtectedRoute)
- Armadilhas: **MEDIUM** — baseadas em patterns do projeto + training
- Testes: **MEDIUM** — existe structure pero gaps detectados em Wave 0

**Pronto para planejar:** ✅ Pesquisa complete — planner pode criar PLAN.md.

