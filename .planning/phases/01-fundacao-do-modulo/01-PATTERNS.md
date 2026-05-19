# Fase 1: Fundação do Módulo - Mapa de Padrões

**Mapeado:** 2026-05-19
**Arquivos analisados:** 6 (novos/modificados)
**Análogos encontrados:** 6 / 6

---

## Classificação de Arquivos

| Arquivo Novo/Modificado | Role | Data Flow | Análogo Mais Próximo | Qualidade da Correspondência |
|--------------------------|------|-----------|----------------------|------------------------------|
| `backend/migrations/108_financeiro_schema.sql` | migration | N/A | `backend/migrations/100_pt_products.sql` | exact |
| `backend/migrations/109_financeiro_empresas.sql` | migration | N/A | `backend/migrations/100_pt_products.sql` | exact |
| `backend/handlers/financeiro.go` | controller | request-response | `backend/handlers/portal_products.go` | exact |
| `backend/main.go` (atualizar rutas) | config | request-response | `backend/main.go:322–342` (closures) | exact |
| `frontend/src/pages/EmpresaPage.tsx` | page-component | request-response | `frontend/src/pages/Login.tsx` | role-match |
| `frontend/src/App.tsx` (atualizar rotas) | router-config | request-response | `frontend/src/App.tsx:10–15` (ProtectedRoute) | exact |

---

## Mapeamento de Padrões

### `backend/migrations/108_financeiro_schema.sql` (migration)

**Análogo:** `backend/migrations/100_pt_products.sql`

**Padrão de criar schema** (linhas 1-4):
```sql
-- Source: backend/migrations/100_pt_products.sql (lines 1-4)
CREATE SCHEMA IF NOT EXISTS portal;

CREATE TABLE IF NOT EXISTS portal.pt_products (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
```

**Aplicar:** Copiar o padrão `CREATE SCHEMA IF NOT EXISTS` para criar `financeiro` schema na migration 108:
```sql
CREATE SCHEMA IF NOT EXISTS financeiro;
```

**Notas:**
- Usar `IF NOT EXISTS` para idempotência (re-aplicável sem erro em migrations repetidas)
- O schema é o primeiro passo antes de criar tabelas que dependem dele

---

### `backend/migrations/109_financeiro_empresas.sql` (migration)

**Análogo:** `backend/migrations/100_pt_products.sql` (estrutura de tabelas)

**Padrão UUID primary key + timestamps** (linhas 6-15):
```sql
-- Source: backend/migrations/100_pt_products.sql (lines 6-15)
CREATE TABLE IF NOT EXISTS portal.pt_products (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            VARCHAR(255) NOT NULL,
    description     TEXT,
    icon_url        VARCHAR(500),
    destination_url VARCHAR(500),
    is_active       BOOLEAN NOT NULL DEFAULT true,
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_pt_products_is_active ON portal.pt_products(is_active);
```

**Aplicar:** 
- Tabela `financeiro.empresas` com UUID PK, timestamps, colunas fully-qualified
- Tabela `financeiro.dados_bancarios` com UUID PK, FK para `financeiro.empresas`, timestamps
- Criar índice para `empresa_id` na tabela de dados bancários

```sql
-- Migration 109: Criar tabelas do módulo financeiro
CREATE TABLE IF NOT EXISTS financeiro.empresas (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    razao_social    VARCHAR(255) NOT NULL,
    nome_fantasia   VARCHAR(255),
    cnpj            VARCHAR(14) NOT NULL UNIQUE,
    logradouro      VARCHAR(255) NOT NULL,
    numero          VARCHAR(10) NOT NULL,
    complemento     VARCHAR(255),
    bairro          VARCHAR(100) NOT NULL,
    cep             VARCHAR(8) NOT NULL,
    municipio       VARCHAR(100) NOT NULL,
    uf              VARCHAR(2) NOT NULL,
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS financeiro.dados_bancarios (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    empresa_id      UUID NOT NULL REFERENCES financeiro.empresas(id) ON DELETE CASCADE,
    banco           VARCHAR(100) NOT NULL,
    agencia         VARCHAR(10) NOT NULL,
    conta           VARCHAR(20) NOT NULL,
    tipo_conta      VARCHAR(20) NOT NULL,
    titular         VARCHAR(255),
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_dados_bancarios_empresa_id ON financeiro.dados_bancarios(empresa_id);
```

**Notas:**
- Todos os nomes de schema no SQL são fully-qualified (`financeiro.empresas`, `financeiro.dados_bancarios`)
- FK usa `ON DELETE CASCADE` para manter integridade relacional
- Índices criados para performance em queries de busca

---

### `backend/handlers/financeiro.go` (controller, request-response)

**Análogo:** `backend/handlers/portal_products.go`

**Padrão handler factory com method dispatch** (linhas 21-26 + 23-24):
```go
// Source: backend/handlers/portal_products.go (lines 21-26)
func GetPortalProductsHandler(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodGet {
            http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
            return
        }
        // ... query logic
    }
}
```

**Padrão struct JSON com tags snake_case** (linhas 10-17):
```go
// Source: backend/handlers/portal_products.go (lines 10-17)
type ProductResponse struct {
    ID             string `json:"id"`
    Name           string `json:"name"`
    Description    string `json:"description"`
    IconURL        string `json:"icon_url"`
    DestinationURL string `json:"destination_url"`
    Contracted     bool   `json:"contracted"`
}
```

**Padrão query com error handling** (linhas 37-47):
```go
// Source: backend/handlers/portal_products.go (lines 37-47)
rows, err = db.Query(`
    SELECT
        id,
        name,
        COALESCE(description, ''),
        COALESCE(icon_url, ''),
        COALESCE(destination_url, '')
    FROM portal.pt_products
    WHERE is_active = true
    ORDER BY name
`)
```

**Padrão header + JSON encoding** (linhas 97-98):
```go
// Source: backend/handlers/portal_products.go (lines 97-98)
w.Header().Set("Content-Type", "application/json")
json.NewEncoder(w).Encode(products)
```

**Aplicar:** Criar `backend/handlers/financeiro.go` com padrão switch de métodos (GET/POST/PUT) para ambos handlers:

```go
package handlers

import (
    "database/sql"
    "encoding/json"
    "net/http"
)

type EmpresaRequest struct {
    ID           string `json:"id,omitempty"`
    RazaoSocial  string `json:"razao_social"`
    NomeFantasia string `json:"nome_fantasia,omitempty"`
    CNPJ         string `json:"cnpj"`
    Logradouro   string `json:"logradouro"`
    Numero       string `json:"numero"`
    Complemento  string `json:"complemento,omitempty"`
    Bairro       string `json:"bairro"`
    CEP          string `json:"cep"`
    Municipio    string `json:"municipio"`
    UF           string `json:"uf"`
}

type DadosBancariosRequest struct {
    ID        string `json:"id,omitempty"`
    EmpresaID string `json:"empresa_id"`
    Banco     string `json:"banco"`
    Agencia   string `json:"agencia"`
    Conta     string `json:"conta"`
    TipoConta string `json:"tipo_conta"`
    Titular   string `json:"titular,omitempty"`
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

// handleGetEmpresa retorna o primeiro registro da tabela (singleton v1)
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

// handlePostEmpresa insere novo registro de empresa
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

// handlePutEmpresa atualiza registro existente de empresa
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

// handleGetDadosBancarios retorna o primeiro registro (1:1 com empresa v1)
func handleGetDadosBancarios(w http.ResponseWriter, r *http.Request, db *sql.DB) {
    var dados DadosBancariosRequest
    err := db.QueryRow(`
        SELECT id, empresa_id, banco, agencia, conta, tipo_conta, titular
        FROM financeiro.dados_bancarios
        LIMIT 1
    `).Scan(
        &dados.ID, &dados.EmpresaID, &dados.Banco, &dados.Agencia, &dados.Conta,
        &dados.TipoConta, &dados.Titular,
    )
    if err == sql.ErrNoRows {
        w.WriteHeader(http.StatusNotFound)
        json.NewEncoder(w).Encode(map[string]string{"error": "dados_bancarios not found"})
        return
    }
    if err != nil {
        http.Error(w, "internal server error", http.StatusInternalServerError)
        return
    }
    json.NewEncoder(w).Encode(dados)
}

// handlePostDadosBancarios insere novo registro
func handlePostDadosBancarios(w http.ResponseWriter, r *http.Request, db *sql.DB) {
    var req DadosBancariosRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "invalid request", http.StatusBadRequest)
        return
    }
    
    var dadosID string
    err := db.QueryRow(`
        INSERT INTO financeiro.dados_bancarios
        (empresa_id, banco, agencia, conta, tipo_conta, titular)
        VALUES ($1, $2, $3, $4, $5, $6)
        RETURNING id
    `,
        req.EmpresaID, req.Banco, req.Agencia, req.Conta, req.TipoConta, req.Titular,
    ).Scan(&dadosID)
    if err != nil {
        http.Error(w, "error creating dados_bancarios", http.StatusInternalServerError)
        return
    }
    
    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(map[string]string{"id": dadosID})
}

// handlePutDadosBancarios atualiza registro existente
func handlePutDadosBancarios(w http.ResponseWriter, r *http.Request, db *sql.DB) {
    var req DadosBancariosRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "invalid request", http.StatusBadRequest)
        return
    }
    if req.ID == "" {
        http.Error(w, "id required", http.StatusBadRequest)
        return
    }
    
    _, err := db.Exec(`
        UPDATE financeiro.dados_bancarios
        SET empresa_id = $1, banco = $2, agencia = $3, conta = $4, tipo_conta = $5,
            titular = $6, updated_at = CURRENT_TIMESTAMP
        WHERE id = $7
    `,
        req.EmpresaID, req.Banco, req.Agencia, req.Conta, req.TipoConta, req.Titular, req.ID,
    )
    if err != nil {
        http.Error(w, "error updating dados_bancarios", http.StatusInternalServerError)
        return
    }
    
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}
```

---

### `backend/main.go` — Registro de Rotas (atualizar)

**Análogo:** `backend/main.go:322–342` (closures) + `backend/main.go:344–355` (registros)

**Padrão closure withAuth para proteção de rota** (linhas 333-342):
```go
// Source: backend/main.go (lines 333-342)
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
```

**Padrão registrar rotas com withAuth** (linhas 344-351):
```go
// Source: backend/main.go (lines 344-351)
// ── Auth ──────────────────────────────────────────────────────────────────
http.HandleFunc("/api/auth/login",           withDB(handlers.LoginHandler))
http.HandleFunc("/api/auth/me",              withAuth(handlers.GetMeHandler, ""))
http.HandleFunc("/api/auth/forgot-password", withDB(handlers.ForgotPasswordHandler))
http.HandleFunc("/api/auth/reset-password",  withDB(handlers.ResetPasswordHandler))
http.HandleFunc("/api/auth/change-password", withAuth(handlers.ChangePasswordHandler, ""))
http.HandleFunc("/api/auth/refresh",         withDB(handlers.RefreshHandler))
http.HandleFunc("/api/auth/logout",          withDB(handlers.LogoutHandler))
```

**Aplicar:** Adicionar duas linhas de registro no bloco `http.HandleFunc` do `backend/main.go`, logo após a seção Portal:

```go
// ── Financeiro ────────────────────────────────────────────────────────────
http.HandleFunc("/api/financeiro/empresa",         withAuth(handlers.EmpresaHandler, "admin"))
http.HandleFunc("/api/financeiro/dados-bancarios", withAuth(handlers.DadosBancariosHandler, "admin"))
```

**Notas:**
- Ambas rotas protegidas com `withAuth(..., "admin")` — requer role "admin"
- Pattern comentário antes de grupo de rotas (ex: `// ── Financeiro ──`)

---

### `frontend/src/pages/EmpresaPage.tsx` (page-component, request-response)

**Análogo:** `frontend/src/pages/Login.tsx`

**Padrão imports e estrutura** (linhas 1-10):
```typescript
// Source: frontend/src/pages/Login.tsx (lines 1-10)
import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Label } from "@/components/ui/label";
import { toast } from "sonner";
import { useNavigate, Link } from "react-router-dom";
import { useAuth } from "@/contexts/AuthContext";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { AlertCircle, BarChart3, FileUp, Zap, ShieldCheck } from "lucide-react";
```

**Padrão handleSubmit com try/catch** (linhas 27-55):
```typescript
// Source: frontend/src/pages/Login.tsx (lines 27-55)
const handleLogin = async (e: React.FormEvent) => {
  e.preventDefault();
  setIsLoading(true);
  setErrorMsg(null);

  try {
    const res = await fetch("/api/auth/login", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ email, password }),
    });

    const data = await res.json();

    if (!res.ok) {
      throw new Error(typeof data === "string" ? data : "Credenciais inválidas");
    }

    login(data);
    toast.success("Login realizado com sucesso!");
    navigate("/dashboard/urgencia/falta");
  } catch (error: unknown) {
    const msg = error instanceof Error ? error.message : "Erro desconhecido";
    setErrorMsg(msg);
    toast.error(msg);
  } finally {
    setIsLoading(false);
  }
};
```

**Aplicar:** Criar `frontend/src/pages/EmpresaPage.tsx` com dois formulários independentes:

```typescript
import { useState, useEffect } from 'react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Label } from '@/components/ui/label'
import { Separator } from '@/components/ui/separator'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { toast } from 'sonner'

interface Empresa {
  id?: string
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
  id?: string
  empresa_id: string
  banco: string
  agencia: string
  conta: string
  tipo_conta: string
  titular?: string
}

export default function EmpresaPage() {
  const [empresa, setEmpresa] = useState<Empresa>({
    razao_social: '',
    cnpj: '',
    logradouro: '',
    numero: '',
    bairro: '',
    cep: '',
    municipio: '',
    uf: '',
  })
  const [dadosBancarios, setDadosBancarios] = useState<DadosBancarios>({
    empresa_id: '',
    banco: '',
    agencia: '',
    conta: '',
    tipo_conta: '',
  })
  const [loading, setLoading] = useState(true)
  const [submittingEmpresa, setSubmittingEmpresa] = useState(false)
  const [submittingDados, setSubmittingDados] = useState(false)
  const [error, setError] = useState<string | null>(null)

  // Carregar dados ao montar
  useEffect(() => {
    const fetchData = async () => {
      try {
        const resEmp = await fetch('/api/financeiro/empresa')
        if (resEmp.ok) {
          const data = await resEmp.json()
          setEmpresa(data)
        }
        const resDad = await fetch('/api/financeiro/dados-bancarios')
        if (resDad.ok) {
          const data = await resDad.json()
          setDadosBancarios(data)
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

  const handleSaveEmpresa = async (e: React.FormEvent) => {
    e.preventDefault()
    setSubmittingEmpresa(true)
    setError(null)
    try {
      const method = empresa.id ? 'PUT' : 'POST'
      const res = await fetch('/api/financeiro/empresa', {
        method,
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(empresa),
      })

      if (!res.ok) {
        const data = await res.json()
        throw new Error(typeof data === 'string' ? data : 'Erro ao salvar empresa')
      }

      const data = await res.json()
      if (!empresa.id) {
        setEmpresa(prev => ({ ...prev, id: data.id }))
      }
      toast.success('Empresa salva com sucesso')
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Erro desconhecido'
      setError(msg)
      toast.error(msg)
    } finally {
      setSubmittingEmpresa(false)
    }
  }

  const handleSaveDadosBancarios = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!empresa.id) {
      toast.error('Salve os dados da empresa primeiro')
      return
    }
    setSubmittingDados(true)
    setError(null)
    try {
      const method = dadosBancarios.id ? 'PUT' : 'POST'
      const res = await fetch('/api/financeiro/dados-bancarios', {
        method,
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ ...dadosBancarios, empresa_id: empresa.id }),
      })

      if (!res.ok) {
        const data = await res.json()
        throw new Error(typeof data === 'string' ? data : 'Erro ao salvar dados bancários')
      }

      const data = await res.json()
      if (!dadosBancarios.id) {
        setDadosBancarios(prev => ({ ...prev, id: data.id }))
      }
      toast.success('Dados bancários salvos com sucesso')
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Erro desconhecido'
      setError(msg)
      toast.error(msg)
    } finally {
      setSubmittingDados(false)
    }
  }

  if (loading) {
    return <div className="min-h-screen p-8 flex items-center justify-center">Carregando...</div>
  }

  return (
    <div className="min-h-screen bg-gray-50 p-8">
      <div className="max-w-2xl mx-auto space-y-8">
        {error && (
          <Alert variant="destructive">
            <AlertDescription>{error}</AlertDescription>
          </Alert>
        )}

        {/* Seção 1: Dados da Empresa */}
        <Card>
          <CardHeader>
            <CardTitle>Dados da Empresa</CardTitle>
          </CardHeader>
          <CardContent>
            <form onSubmit={handleSaveEmpresa} className="space-y-4">
              <div>
                <Label htmlFor="razao_social">Razão Social *</Label>
                <Input
                  id="razao_social"
                  value={empresa.razao_social}
                  onChange={(e) => setEmpresa(prev => ({ ...prev, razao_social: e.target.value }))}
                  required
                />
              </div>
              <div>
                <Label htmlFor="nome_fantasia">Nome Fantasia</Label>
                <Input
                  id="nome_fantasia"
                  value={empresa.nome_fantasia || ''}
                  onChange={(e) => setEmpresa(prev => ({ ...prev, nome_fantasia: e.target.value }))}
                />
              </div>
              <div>
                <Label htmlFor="cnpj">CNPJ *</Label>
                <Input
                  id="cnpj"
                  value={empresa.cnpj}
                  onChange={(e) => setEmpresa(prev => ({ ...prev, cnpj: e.target.value }))}
                  required
                />
              </div>
              <div>
                <Label htmlFor="logradouro">Logradouro *</Label>
                <Input
                  id="logradouro"
                  value={empresa.logradouro}
                  onChange={(e) => setEmpresa(prev => ({ ...prev, logradouro: e.target.value }))}
                  required
                />
              </div>
              <div>
                <Label htmlFor="numero">Número *</Label>
                <Input
                  id="numero"
                  value={empresa.numero}
                  onChange={(e) => setEmpresa(prev => ({ ...prev, numero: e.target.value }))}
                  required
                />
              </div>
              <div>
                <Label htmlFor="complemento">Complemento</Label>
                <Input
                  id="complemento"
                  value={empresa.complemento || ''}
                  onChange={(e) => setEmpresa(prev => ({ ...prev, complemento: e.target.value }))}
                />
              </div>
              <div>
                <Label htmlFor="bairro">Bairro *</Label>
                <Input
                  id="bairro"
                  value={empresa.bairro}
                  onChange={(e) => setEmpresa(prev => ({ ...prev, bairro: e.target.value }))}
                  required
                />
              </div>
              <div>
                <Label htmlFor="cep">CEP *</Label>
                <Input
                  id="cep"
                  value={empresa.cep}
                  onChange={(e) => setEmpresa(prev => ({ ...prev, cep: e.target.value }))}
                  required
                />
              </div>
              <div>
                <Label htmlFor="municipio">Município *</Label>
                <Input
                  id="municipio"
                  value={empresa.municipio}
                  onChange={(e) => setEmpresa(prev => ({ ...prev, municipio: e.target.value }))}
                  required
                />
              </div>
              <div>
                <Label htmlFor="uf">UF *</Label>
                <Input
                  id="uf"
                  maxLength={2}
                  value={empresa.uf}
                  onChange={(e) => setEmpresa(prev => ({ ...prev, uf: e.target.value.toUpperCase() }))}
                  required
                />
              </div>
              <Button type="submit" disabled={submittingEmpresa} className="w-full">
                {submittingEmpresa ? 'Salvando...' : 'Salvar Empresa'}
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
              <div>
                <Label htmlFor="banco">Banco *</Label>
                <Input
                  id="banco"
                  value={dadosBancarios.banco}
                  onChange={(e) => setDadosBancarios(prev => ({ ...prev, banco: e.target.value }))}
                  disabled={!empresa.id}
                  required
                />
              </div>
              <div>
                <Label htmlFor="agencia">Agência *</Label>
                <Input
                  id="agencia"
                  value={dadosBancarios.agencia}
                  onChange={(e) => setDadosBancarios(prev => ({ ...prev, agencia: e.target.value }))}
                  disabled={!empresa.id}
                  required
                />
              </div>
              <div>
                <Label htmlFor="conta">Conta *</Label>
                <Input
                  id="conta"
                  value={dadosBancarios.conta}
                  onChange={(e) => setDadosBancarios(prev => ({ ...prev, conta: e.target.value }))}
                  disabled={!empresa.id}
                  required
                />
              </div>
              <div>
                <Label htmlFor="tipo_conta">Tipo de Conta *</Label>
                <Input
                  id="tipo_conta"
                  value={dadosBancarios.tipo_conta}
                  onChange={(e) => setDadosBancarios(prev => ({ ...prev, tipo_conta: e.target.value }))}
                  disabled={!empresa.id}
                  required
                  placeholder="Ex: corrente, poupança"
                />
              </div>
              <div>
                <Label htmlFor="titular">Titular</Label>
                <Input
                  id="titular"
                  value={dadosBancarios.titular || ''}
                  onChange={(e) => setDadosBancarios(prev => ({ ...prev, titular: e.target.value }))}
                  disabled={!empresa.id}
                />
              </div>
              <Button
                type="submit"
                disabled={submittingDados || !empresa.id}
                className="w-full"
              >
                {submittingDados ? 'Salvando...' : 'Salvar Dados Bancários'}
              </Button>
            </form>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
```

---

### `frontend/src/App.tsx` — Registrar Rota (atualizar)

**Análogo:** `frontend/src/App.tsx:1–44` (estrutura de rotas)

**Padrão ProtectedRoute** (linhas 10-15):
```typescript
// Source: frontend/src/App.tsx (lines 10-15)
function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const { isAuthenticated, loading } = useAuth()
  if (loading) return null
  if (!isAuthenticated) return <Navigate to="/admin/login" replace />
  return <>{children}</>
}
```

**Padrão registrar rota** (linhas 22-32):
```typescript
// Source: frontend/src/App.tsx (lines 22-32)
<Routes>
  {/* Portal público */}
  <Route path="/" element={<PortalPage />} />

  {/* Admin */}
  <Route path="/admin/login" element={<Login />} />
  <Route path="/admin" element={
    <ProtectedRoute>
      <div>AdminDashboard — em breve</div>
    </ProtectedRoute>
  } />

  {/* Fallback */}
  <Route path="*" element={<Navigate to="/" replace />} />
</Routes>
```

**Aplicar:** Adicionar import e nova rota em `frontend/src/App.tsx`:

```typescript
// Adicionar import no topo:
import EmpresaPage from './pages/EmpresaPage'

// Adicionar rota dentro de <Routes>:
<Route path="/admin/financeiro/empresa" element={
  <ProtectedRoute>
    <EmpresaPage />
  </ProtectedRoute>
} />
```

Localização: inserir depois da rota `/admin` e antes do fallback `*`.

---

## Padrões Compartilhados

### Autenticação (Authorization)

**Fonte:** `backend/handlers/auth.go:207–259`

**Aplicar em:** Todos os handlers do módulo financeiro

**Padrão de validação de role** (linhas 251-254):
```go
// Source: backend/handlers/auth.go (lines 251-254)
if requiredRole != "" && userRole != requiredRole && userRole != "admin" {
    http.Error(w, "Forbidden: insufficient permissions", http.StatusForbidden)
    return
}
```

**Notas:**
- Role `admin` tem bypass universal
- Rotas financeiro usam `withAuth(handler, "admin")` — obrigam role admin
- Validação feita em `AuthMiddleware` antes do handler executar

### Tratamento de Erros

**Fonte:** `backend/handlers/portal_products.go:70–98` (error handling)

**Padrão:**
```go
// Source: backend/handlers/portal_products.go (lines 70-98)
if err != nil {
    http.Error(w, "internal server error", http.StatusInternalServerError)
    return
}
defer rows.Close()

// ... scan loop ...

if err = rows.Err(); err != nil {
    http.Error(w, "internal server error", http.StatusInternalServerError)
    return
}

w.Header().Set("Content-Type", "application/json")
json.NewEncoder(w).Encode(products)
```

**Aplicar em:** Todos os handlers (GET/POST/PUT) para `EmpresaHandler` e `DadosBancariosHandler`

- Query errors → `http.StatusInternalServerError`
- Validation errors → `http.StatusBadRequest`
- JSON encoding always after all checks
- Header `Content-Type: application/json` set before encoding

### Validação de JSON no Frontend

**Fonte:** `frontend/src/pages/Login.tsx:27–55` (try/catch pattern)

**Padrão:**
```typescript
try {
  const res = await fetch("/api/auth/login", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email, password }),
  });

  const data = await res.json();

  if (!res.ok) {
    throw new Error(typeof data === "string" ? data : "Credenciais inválidas");
  }
  // ... success logic
} catch (error: unknown) {
  const msg = error instanceof Error ? error.message : "Erro desconhecido";
  setErrorMsg(msg);
  toast.error(msg);
} finally {
  setIsLoading(false);
}
```

**Aplicar em:** Ambos os handlers de formulário (EmpresaPage)
- Always check `res.ok` after fetch
- Use `try/catch` com type guard `error instanceof Error`
- Call `toast.error()` para feedback ao usuário
- Set `isLoading` no finally block

---

## Arquivos Sem Análogo

Nenhum arquivo classificado como "sem análogo". Todos os padrões têm precedentes diretos no codebase:

- Migrations: precedente em `migration 100`
- Handlers: precedente em `portal_products.go`
- Routes: precedente em `main.go` e `App.tsx`
- Pages: precedente em `Login.tsx`

---

## Metadados

**Escopo de busca de análogos:** `backend/migrations/*.sql`, `backend/handlers/*.go`, `backend/main.go`, `frontend/src/App.tsx`, `frontend/src/pages/*.tsx`

**Arquivos escaneados:** 12 referências analisadas

**Data de extração:** 2026-05-19

---

## Checklist de Implementação

Ao implementar os novos arquivos, validar contra estes padrões:

- [ ] `financeiro.sql` migrations usam fully-qualified schema names (`financeiro.*`)
- [ ] Handler factory retorna `http.HandlerFunc` com switch de métodos (GET/POST/PUT)
- [ ] Structs JSON com tags `snake_case`
- [ ] Todas queries usam parameterized statements (`$1, $2`, etc)
- [ ] `Content-Type: application/json` set antes de encoding
- [ ] Error handling com `http.Error()` para erros internos, `http.StatusBadRequest` para validation
- [ ] Routes registradas com `withAuth(handler, "admin")` em `main.go`
- [ ] Frontend: `ProtectedRoute` wrapper em `App.tsx`
- [ ] Frontend: try/catch + `toast.error()` em handlers
- [ ] Dois formulários independentes com state separado em `EmpresaPage`
