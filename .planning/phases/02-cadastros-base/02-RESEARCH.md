# Phase 2: Cadastros Base — Research

**Pesquisado:** 2026-05-20
**Domínio:** CRUD multi-entidade, schema relacional PostgreSQL, listas com busca em Go/React
**Confiança geral:** HIGH — baseado em leitura direta do codebase e padrões estabelecidos na Phase 1

---

## Key Findings

1. **Produtos são semi-fixos, não dinâmicos:** Os 4 produtos (FB_APU02, FB_APU04, FB_SMARTPICK, FB_FAROL) já existem em `portal.pt_products` como seed. Em `financeiro.*` devemos criar uma tabela `financeiro.produtos` com os mesmos 4 registros + a estrutura de planos/preços — mas NÃO referenciar `portal.pt_products` via FK (schemas separados, propósitos distintos). Seed na migration.

2. **Multi-CNPJ requer tabela separada, não JSON:** Tabela `financeiro.cliente_cnpjs` com FK para `financeiro.clientes` é o padrão correto para o Go com `lib/pq` (sem ORM). Array JSON em coluna impossibilita busca por CNPJ via `ILIKE` e complica contratos por CNPJ na fase 3.

3. **Contratos precisam de duas tabelas de junção:** `financeiro.contrato_cnpjs` (quais CNPJs do grupo cobertos) e `financeiro.contrato_itens` (quais produto+plano com valor). Ambas têm FK para `financeiro.contratos`. Isso é o que permite fase 3 gerar tokens por item.

4. **`pg_trgm` já está ativa no banco:** Migration 067 ativou `CREATE EXTENSION IF NOT EXISTS pg_trgm`. Busca `ILIKE '%termo%'` com índice GIN é o padrão do projeto. Replicar o padrão para `clientes.razao_social` e `cliente_cnpjs.cnpj`.

5. **Três waves, não duas:** Backend (migrations + handlers) → Frontend (listas) → Frontend (formulários de contrato com lógica de junção). O contrato tem dependências de UX complexas (selecionar CNPJs do grupo, selecionar produtos/planos) que devem ficar em wave separada do CRUD simples de clientes/produtos.

---

## Schema Design

### Clientes e Grupos Multi-CNPJ

**Abordagem recomendada: duas tabelas (`financeiro.clientes` + `financeiro.cliente_cnpjs`)**

Razões contra coluna JSON:
- `lib/pq` exige scan manual de JSON; perde tipagem e dificulta manutenção
- Impossível criar índice GIN trigram em elemento de array JSON para busca por CNPJ
- Fase 3 gera tokens por CNPJ específico dentro do grupo — precisará de `cnpj_id UUID` como FK

**DDL proposto para migration 110:**

```sql
CREATE TABLE IF NOT EXISTS financeiro.clientes (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    razao_social VARCHAR(255) NOT NULL,
    cnpj         VARCHAR(14) NOT NULL UNIQUE,  -- CNPJ principal
    email        VARCHAR(255),
    telefone     VARCHAR(20),
    responsavel  VARCHAR(255),
    ativo        BOOLEAN NOT NULL DEFAULT true,
    created_at   TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at   TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Índice GIN trigram para busca por nome (pg_trgm já ativo via migration 067)
CREATE INDEX IF NOT EXISTS idx_clientes_razao_social_trgm
ON financeiro.clientes USING GIN (razao_social gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_clientes_ativo ON financeiro.clientes(ativo);

CREATE TABLE IF NOT EXISTS financeiro.cliente_cnpjs (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cliente_id UUID NOT NULL REFERENCES financeiro.clientes(id) ON DELETE CASCADE,
    cnpj       VARCHAR(14) NOT NULL UNIQUE,  -- CNPJs adicionais do grupo
    descricao  VARCHAR(255),                 -- nome/filial opcional
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_cliente_cnpjs_cliente_id
ON financeiro.cliente_cnpjs(cliente_id);

CREATE INDEX IF NOT EXISTS idx_cliente_cnpjs_cnpj_trgm
ON financeiro.cliente_cnpjs USING GIN (cnpj gin_trgm_ops);
```

**Impacto na Phase 3:** Token de liberação será vinculado a `financeiro.contratos` — um contrato cobre um ou mais CNPJs. A tabela `contrato_cnpjs` (ver abaixo) fará a ligação. Os CNPJs cobertos pelo contrato são os que têm acesso quando o token está ativo.

**Busca unificada por CNPJ (nome OU cnpj adicional):** A query de listagem de clientes deve fazer UNION ou subquery para verificar tanto `clientes.cnpj` quanto `cliente_cnpjs.cnpj`, retornando o `cliente_id` sem duplicatas. Ver padrão em Handler Patterns.

---

### Produtos e Planos

**Abordagem recomendada: duas tabelas (`financeiro.produtos` + `financeiro.planos`)**

Os 4 produtos são semi-fixos (PROD-01 implica CRUD para edição de nome/descrição), mas os planos e preços são mutáveis (PROD-03, PROD-04). Estrutura limpa:

```sql
-- Migration 111
CREATE TABLE IF NOT EXISTS financeiro.produtos (
    id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    codigo    VARCHAR(50) NOT NULL UNIQUE,  -- FB_APU02, FB_APU04, FB_SMARTPICK, FB_FAROL
    nome      VARCHAR(255) NOT NULL,
    descricao TEXT,
    ativo     BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Seed dos 4 produtos
INSERT INTO financeiro.produtos (codigo, nome, descricao) VALUES
    ('FB_APU02',      'Apuração Assistida',    'Ferramenta de apuração fiscal assistida'),
    ('FB_APU04',      'Simulador Fiscal',      'Simulador de cenários fiscais'),
    ('FB_SMARTPICK',  'SmartPick',             'Inteligência para picking logístico'),
    ('FB_FAROL',      'Farol',                 'Monitoramento e alertas fiscais')
ON CONFLICT (codigo) DO NOTHING;

CREATE TABLE IF NOT EXISTS financeiro.planos (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    produto_id UUID NOT NULL REFERENCES financeiro.produtos(id) ON DELETE CASCADE,
    nome       VARCHAR(50) NOT NULL,   -- Lite, Standard, Premium, Enterprise, Sob Demanda
    preco      NUMERIC(12, 2),         -- NULL = "a negociar" para Sob Demanda
    ativo      BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (produto_id, nome)
);

-- Seed dos 5 planos para cada produto
INSERT INTO financeiro.planos (produto_id, nome, preco)
SELECT id, unnest(ARRAY['Lite','Standard','Premium','Enterprise','Sob Demanda']), NULL
FROM financeiro.produtos
ON CONFLICT (produto_id, nome) DO NOTHING;
```

**Relação com `portal.pt_products`:** NÃO usar FK entre schemas. Os dois sistemas servem propósitos distintos:
- `portal.pt_products` = catálogo público para o portal de clientes externos (portal page)
- `financeiro.produtos` = portfólio interno para contratos e tokens

Manter independentes. No v2, se necessário sincronizar, usar `codigo` como chave de reconciliação.

**Por que não JSON para planos:** Plano individual precisa de `plano_id` como FK em `financeiro.contrato_itens` (fase 3 referencia qual plano está em vigor). JSON inviabiliza FKs.

---

### Contratos

**Estrutura de três tabelas: `financeiro.contratos` + `financeiro.contrato_cnpjs` + `financeiro.contrato_itens`**

```sql
-- Migration 112
CREATE TABLE IF NOT EXISTS financeiro.contratos (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cliente_id     UUID NOT NULL REFERENCES financeiro.clientes(id),
    data_inicio    DATE NOT NULL,
    periodicidade  VARCHAR(20) NOT NULL DEFAULT 'mensal',  -- mensal, trimestral, anual
    valor_total    NUMERIC(12, 2) NOT NULL,
    status         VARCHAR(20) NOT NULL DEFAULT 'ativo',    -- ativo, encerrado, suspenso
    observacoes    TEXT,
    created_at     TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at     TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_contratos_cliente_id ON financeiro.contratos(cliente_id);
CREATE INDEX IF NOT EXISTS idx_contratos_status ON financeiro.contratos(status);

-- Quais CNPJs do grupo estão cobertos por este contrato
CREATE TABLE IF NOT EXISTS financeiro.contrato_cnpjs (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    contrato_id UUID NOT NULL REFERENCES financeiro.contratos(id) ON DELETE CASCADE,
    cnpj_id     UUID NOT NULL REFERENCES financeiro.cliente_cnpjs(id),
    UNIQUE (contrato_id, cnpj_id)
);

-- Importante: o CNPJ principal do cliente não está em cliente_cnpjs,
-- então para contratos que cobrem o CNPJ principal usar coluna nullable:
-- Alternativa: incluir o CNPJ principal em cliente_cnpjs automaticamente no INSERT de cliente.

-- Quais produtos/planos estão no contrato (um item = um produto + um plano + valor negociado)
CREATE TABLE IF NOT EXISTS financeiro.contrato_itens (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    contrato_id UUID NOT NULL REFERENCES financeiro.contratos(id) ON DELETE CASCADE,
    plano_id    UUID NOT NULL REFERENCES financeiro.planos(id),
    valor_item  NUMERIC(12, 2),  -- pode diferir do plano.preco (negociado)
    UNIQUE (contrato_id, plano_id)
);

CREATE INDEX IF NOT EXISTS idx_contrato_itens_contrato_id ON financeiro.contrato_itens(contrato_id);
```

**Decisão crítica sobre CNPJ principal:** O `clientes.cnpj` (principal) NÃO está em `cliente_cnpjs`. Para simplificar a cobertura de contratos, **inserir o CNPJ principal em `cliente_cnpjs` automaticamente no POST de cliente** (como registro "principal"). Isso unifica o modelo: todos os CNPJs do grupo, incluindo o principal, ficam em `cliente_cnpjs`. O `clientes.cnpj` permanece como UNIQUE constraint para busca rápida e display.

**Status para suportar Phase 3:** O campo `status` em `financeiro.contratos` deve contemplar `ativo`, `encerrado` e `suspenso`. Phase 3 adicionará lógica de token que muda status de token (tabela separada), não do contrato. Contrato ativo pode ter token suspenso — o contrato não muda de status por inadimplência; apenas o token é suspenso. Separação limpa das responsabilidades.

**Aditivos (CONT-04):** Implementar como novos registros em `contrato_itens` (adicionando produtos) ou como UPDATE de `valor_total` + `observacoes` com auditoria. MVP mais simples: campo `observacoes TEXT` no contrato principal para registrar o texto do aditivo. Não criar tabela de aditivos na fase 2 — complexidade desnecessária para MVP.

---

## Handler Patterns (Go)

### Lista com Busca — Padrão do Projeto

O projeto usa `r.URL.Query().Get("param")` para query params (ver `portal_products.go` linha 28, `admin.go` linha 158). O padrão para lista com filtro é:

```go
// Em financeiro.go — função handleListClientes
func handleListClientes(w http.ResponseWriter, r *http.Request, db *sql.DB) {
    q := r.URL.Query().Get("q")        // busca livre
    status := r.URL.Query().Get("status") // "ativo", "inativo", "" (todos)

    query := `
        SELECT DISTINCT c.id, c.razao_social, c.cnpj, c.email, c.telefone,
               c.responsavel, c.ativo, c.created_at
        FROM financeiro.clientes c
        LEFT JOIN financeiro.cliente_cnpjs cc ON cc.cliente_id = c.id
        WHERE 1=1
    `
    args := []interface{}{}
    argN := 1

    if q != "" {
        query += fmt.Sprintf(` AND (c.razao_social ILIKE $%d OR c.cnpj ILIKE $%d OR cc.cnpj ILIKE $%d)`, argN, argN, argN)
        args = append(args, "%"+q+"%")
        argN++
    }
    if status == "ativo" {
        query += fmt.Sprintf(` AND c.ativo = $%d`, argN)
        args = append(args, true)
        argN++
    } else if status == "inativo" {
        query += fmt.Sprintf(` AND c.ativo = $%d`, argN)
        args = append(args, false)
        argN++
    }
    query += ` ORDER BY c.razao_social`

    rows, err := db.Query(query, args...)
    // ... scan e encode
}
```

**Paginação:** O projeto NÃO tem paginação em nenhum handler existente. `ListUsersHandler` retorna lista completa. Para MVP da fase 2, retornar lista completa (sem paginação). Clientes são poucos (dezenas no v1). Adicionar `LIMIT/OFFSET` via query params é trivial se necessário.

### Handler Factory para múltiplas entidades

Cada entidade fica em seu próprio grupo de funções no mesmo arquivo `financeiro.go` (ou arquivos separados por entidade: `financeiro_clientes.go`, `financeiro_produtos.go`, `financeiro_contratos.go`). **Recomendação: usar arquivos separados** para manter cada arquivo < 200 linhas — o projeto já tem `handlers/` com múltiplos arquivos pequenos por domínio.

**Padrão de método dispatch** (já estabelecido na fase 1):
```go
func ClienteHandler(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        switch r.Method {
        case http.MethodGet:
            id := r.URL.Query().Get("id")
            if id != "" {
                handleGetCliente(w, r, db, id)
            } else {
                handleListClientes(w, r, db)
            }
        case http.MethodPost:
            handlePostCliente(w, r, db)
        case http.MethodPut:
            handlePutCliente(w, r, db)
        default:
            http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
        }
    }
}
```

**GET por ID vs. lista:** Sem URL params, usar `?id=UUID` para buscar item único (padrão confirmado em `admin.go:158` e `environment.go:139`).

### Registro de rotas em main.go

Seguir o bloco `// ── Financeiro ──` existente em `main.go`:
```go
// Clientes
http.HandleFunc("/api/financeiro/clientes",  withAuth(handlers.ClienteHandler, "admin"))

// Produtos e Planos
http.HandleFunc("/api/financeiro/produtos",  withAuth(handlers.ProdutoHandler, "admin"))
http.HandleFunc("/api/financeiro/planos",    withAuth(handlers.PlanoHandler, "admin"))

// Contratos
http.HandleFunc("/api/financeiro/contratos", withAuth(handlers.ContratoHandler, "admin"))
```

**Importante:** Rotas separadas para planos (vs. aninhar em `/produtos/:id/planos`) porque o projeto não suporta path params — tudo via query string.

---

## Frontend Patterns (React)

### Lista com busca — padrão híbrido TanStack Query + estado local

`PortalPage.tsx` usa `useQuery` para fetching (padrão do projeto). `EmpresaPage.tsx` usa `useEffect + fetch` manual (padrão de singleton). Para listas com busca, usar `useQuery` com `queryKey` reativo ao termo de busca:

```tsx
// ClientesPage.tsx
const [busca, setBusca] = useState('')
const [status, setStatus] = useState<'todos' | 'ativo' | 'inativo'>('todos')

const { data: clientes, isPending, refetch } = useQuery({
    queryKey: ['financeiro-clientes', busca, status],
    queryFn: () => {
        const params = new URLSearchParams()
        if (busca) params.set('q', busca)
        if (status !== 'todos') params.set('status', status)
        return fetch(`/api/financeiro/clientes?${params}`).then(r => r.json())
    }
})
```

**Debounce da busca:** Input de busca atualiza `busca` com debounce de 300ms para não disparar uma query a cada tecla. Implementar via `setTimeout` + `clearTimeout` em `onChange` — sem biblioteca extra.

### Componentes de tabela — shadcn/ui Table

O componente `table.tsx` já está disponível (confirmado em `frontend/src/components/ui/`). Padrão para listagem:

```tsx
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'

// Dentro do JSX:
<Input placeholder="Buscar por nome ou CNPJ..." value={busca}
    onChange={e => setBusca(e.target.value)} className="max-w-sm" />

<Table>
    <TableHeader>
        <TableRow>
            <TableHead>Razão Social</TableHead>
            <TableHead>CNPJ</TableHead>
            <TableHead>Status</TableHead>
            <TableHead>Ações</TableHead>
        </TableRow>
    </TableHeader>
    <TableBody>
        {clientes?.map(c => (
            <TableRow key={c.id}>
                <TableCell>{c.razao_social}</TableCell>
                <TableCell>{c.cnpj}</TableCell>
                <TableCell>
                    <Badge variant={c.ativo ? 'default' : 'secondary'}>
                        {c.ativo ? 'Ativo' : 'Inativo'}
                    </Badge>
                </TableCell>
                <TableCell>
                    <Button size="sm" variant="outline" onClick={() => setEditando(c)}>
                        Editar
                    </Button>
                </TableCell>
            </TableRow>
        ))}
    </TableBody>
</Table>
```

**`badge.tsx`** já disponível. `select.tsx` disponível para filtro de status (usar `<Select>` do shadcn/ui em vez de `<select>` nativo).

### Formulários — estado controlado simples (manter padrão da fase 1)

A fase 1 usou **estado controlado simples** sem react-hook-form. O projeto tem `react-hook-form` instalado mas nunca usado em nenhuma página existente. **Recomendação: manter estado controlado simples** para consistência com EmpresaPage e Login — o padrão está estabelecido.

Exceção: o formulário de contrato tem seleção múltipla de CNPJs e produtos/planos, que exige state mais complexo — arrays de IDs selecionados. Usar `useState<string[]>` para arrays de seleção, sem necessidade de react-hook-form.

### Dialog/Sheet para formulários de edição

Usar `<Dialog>` (já disponível) para formulários de criação/edição inline, em vez de páginas separadas. Padrão: lista na página principal → botão "Novo" ou "Editar" → Dialog com formulário. Isso evita criar 6+ rotas separadas.

**Rotas de frontend necessárias:**
- `/admin/financeiro/clientes` — lista + dialog de criação/edição
- `/admin/financeiro/produtos` — lista de produtos + edição de preços por plano
- `/admin/financeiro/contratos` — lista + formulário de criação de contrato (wizard simples ou página separada, dado a complexidade)

Para **contrato**, recomendo rota separada `/admin/financeiro/contratos/novo` em vez de Dialog — o formulário tem 3 seções (cliente, CNPJs, produtos/planos) e não cabe bem em modal.

---

## Wave Structure Recommendation

**3 waves, 4 planos:**

### Wave 1 — Backend (bloqueante)
**Plano 02-01:** Migrations + Handlers Go + Rotas

- Migrations 110 (clientes + cliente_cnpjs), 111 (produtos + planos + seed), 112 (contratos + contrato_cnpjs + contrato_itens)
- Handlers: `financeiro_clientes.go` (CRUD + listagem com busca), `financeiro_produtos.go` (GET lista, PUT preço de plano), `financeiro_contratos.go` (GET lista por cliente, POST criar, PUT editar)
- Registro de rotas em `main.go`
- Verificação: `go build ./...` + testes de endpoint manual

**Por que tudo em um plano:** As migrations são pré-requisito de todos os handlers; colocá-las em planos separados cria dependências internas desnecessárias.

### Wave 2 — Frontend básico (paralelo internamente, sequencial após Wave 1)
**Plano 02-02:** Clientes + Produtos (páginas de lista simples)

- `ClientesPage.tsx` — lista com busca/filtro, Dialog para criar/editar cliente, gerenciar CNPJs adicionais
- `ProdutosPage.tsx` — lista fixa de 4 produtos com tabela de planos/preços editável por produto
- Rotas em `App.tsx`: `/admin/financeiro/clientes` e `/admin/financeiro/produtos`

**Por que junto:** Clientes e Produtos são formulários simples sem dependências entre si. Ambos podem ser implementados simultaneamente.

### Wave 3 — Frontend avançado (sequencial após Wave 2)
**Plano 02-03:** Contratos

- `ContratosPage.tsx` — lista de contratos com histórico por cliente
- `ContratoNovoPage.tsx` (ou modal pesado) — formulário em 3 passos: (1) selecionar cliente, (2) selecionar CNPJs do grupo, (3) selecionar produto(s)/plano(s) + valor + periodicidade
- Rota em `App.tsx`: `/admin/financeiro/contratos` e `/admin/financeiro/contratos/novo`
- Checkpoint humano end-to-end: criar cliente → criar contrato → verificar histórico

**Por que separado:** O formulário de contrato consome dados de clientes E produtos (seleção cruzada) — esses endpoints devem estar funcionando antes de construir o formulário. Testar o formulário de contrato exige que clientes e produtos existam no banco.

**Resumo:**
```
Wave 1:  [02-01] Migrations + Handlers Go + Rotas         ← bloqueante
Wave 2:  [02-02] ClientesPage + ProdutosPage               ← bloqueado em Wave 1
Wave 3:  [02-03] ContratosPage + ContratoNovoPage          ← bloqueado em Wave 2
```

---

## Risk Flags

### RF-01: CNPJ principal vs. lista de CNPJs
**Problema:** `clientes.cnpj` (principal) e `cliente_cnpjs.cnpj` (adicionais) são entidades separadas. O formulário de contrato precisa mostrar TODOS os CNPJs do grupo (principal + adicionais). A decisão de inserir o CNPJ principal automaticamente em `cliente_cnpjs` no POST simplifica isso — verificar que o handler de criação de cliente faça os dois INSERTs em transação (`db.Begin()`).

### RF-02: Seed de planos com preço NULL
**Problema:** PROD-02 exige 5 planos por produto. Seed cria planos com `preco = NULL` (a negociar). A UI deve tratar `null` como "—" e não exibir "R$ 0,00". Handler deve retornar `preco: null` (não `preco: 0`). Em Go, usar `sql.NullFloat64` para o campo preço.

### RF-03: Contrato precisa de transação multi-tabela
**Problema:** Criar um contrato envolve INSERT em 3 tabelas: `contratos`, `contrato_cnpjs` (N registros), `contrato_itens` (M registros). Obrigatoriamente em `db.Begin()` / `tx.Rollback()` / `tx.Commit()`. Qualquer falha parcial deixaria o banco inconsistente. Este é o handler mais complexo da fase 2.

**Padrão confirmado no projeto:** O projeto usa `db.Exec` e `db.QueryRow` direto (sem transações explícitas na fase 1). Para a fase 2, o handler de POST contrato DEVE introduzir o primeiro uso de `tx, err := db.Begin()` — isso é novo no codebase e precisa ser documentado no plano.

### RF-04: Busca unificada por CNPJ
**Problema:** Buscar cliente por CNPJ deve encontrar tanto o CNPJ principal (`clientes.cnpj`) quanto CNPJs adicionais (`cliente_cnpjs.cnpj`). A query com `DISTINCT` + `LEFT JOIN` + `ILIKE` pode retornar duplicatas se não usar `DISTINCT ON (c.id)` ou subquery. Testar com cliente que tem 3+ CNPJs.

### RF-05: Sem `data_fim` no contrato — OK para MVP
**Observação:** O schema proposto não tem `data_fim` em contratos (CONT-03 fala em periodicidade, não em data de término). A renovação é gerenciada pelo token (fase 3). Para MVP, `periodicidade` (mensal/trimestral/anual) é suficiente — a data de expiração é calculada a partir do token, não do contrato. O planejador DEVE confirmar que a ausência de `data_fim` no contrato é intencional para o v1.

### RF-06: Portal schema e financeiro schema coexistem sem conflito
**Verificado:** `portal.*` (migrations 100-107) e `financeiro.*` (migrations 108+) são schemas separados. Não há FK entre eles. O projeto confirma que `CREATE SCHEMA IF NOT EXISTS` é idempotente. Não há risco de colisão.

---

## Canonical References

**O planejador DEVE ler estes arquivos antes de criar os planos:**

| Arquivo | Por que é obrigatório |
|---------|----------------------|
| `backend/handlers/financeiro.go` | Padrão exato de handler factory, switch de método, sql.NullString, CRUD base |
| `backend/handlers/admin.go` (linhas 102-146) | Único exemplo de `db.Query` com `rows.Next()` + scan para lista — padrão para handlers de listagem |
| `backend/handlers/portal_products.go` | Padrão de query param (`r.URL.Query().Get`), lista com rows, DISTINCT/JOIN |
| `backend/main.go` (linhas 344-359) | Bloco de registro de rotas — onde inserir novas rotas financeiro |
| `backend/migrations/067_add_trgm_indexes.sql` | Confirma pg_trgm ativo + padrão de índice GIN para colunas de texto |
| `backend/migrations/109_financeiro_empresas.sql` | Template de migration para financeiro.* com padrões de colunas |
| `frontend/src/pages/EmpresaPage.tsx` | Padrão de formulário controlado, fetch manual, toast, Card/Label/Input |
| `frontend/src/pages/PortalPage.tsx` (linhas 1-75) | Padrão de useQuery com queryKey + fetch |
| `frontend/src/components/ui/table.tsx` | Componente Table disponível, confirmar imports |
| `frontend/src/App.tsx` | Onde registrar novas rotas React + padrão ProtectedRoute |
| `.planning/phases/01-fundacao-do-modulo/01-CONTEXT.md` | Decisões bloqueadas da fase anterior que se aplicam aqui |

---

## Perguntas em Aberto para o Planejador

1. **CNPJ principal em `cliente_cnpjs`?** A decisão de inserir o CNPJ principal automaticamente em `cliente_cnpjs` no POST do cliente simplifica a cobertura de contratos. O planejador deve confirmar e tornar isso explícito no plano do handler.

2. **`data_fim` no contrato?** CONT-03 diz "periodicidade de renovação" mas não "data de término". Confirmar que o contrato não precisa de `data_fim` no v1 — a expiração é responsabilidade do token (fase 3).

3. **Aditivos (CONT-04):** A proposta é campo `observacoes TEXT` no contrato para MVP. É suficiente ou é necessária uma tabela de aditivos?

4. **Rota do formulário de novo contrato:** Dialog pesado ou página separada (`/admin/financeiro/contratos/novo`)? A complexidade do formulário (3 seções, múltipla seleção) sugere página separada.

---

*Pesquisa concluída: 2026-05-20*
*Baseada em: leitura direta do codebase (HIGH confidence — sem assumidos externos)*
