---
project_name: 'FBTAX_CLOUD'
user_name: 'Claudio'
date: '2026-04-08'
sections_completed: ['technology_stack', 'language_rules', 'framework_rules', 'database_rules', 'api_rules', 'inherited_code', 'anti_patterns', 'env_vars']
status: 'complete'
optimized_for_llm: true
---

# Project Context for AI Agents

_Este arquivo contém regras críticas que agentes de IA devem seguir ao implementar código no FBTAX_CLOUD. Foco em detalhes não-óbvios que agentes poderiam deixar passar._

---

## Technology Stack & Versions

- **Backend:** Go 1.22 — `net/http` stdlib, sem framework externo
- **Frontend:** React 18.3.1 + TypeScript 5.2.2 (strict) + Vite 5.2.0
- **Styling:** Tailwind CSS 3.4.3 + Shadcn/UI (Radix UI)
- **Data Fetching:** TanStack Query v5
- **Database:** PostgreSQL 15
- **Module Go:** `fb_cloud` (não fb_smartpick)
- **Port:** 8083
- **Deploy:** Docker Compose + Coolify/Hostinger

---

## Critical Implementation Rules

### Go (Backend)

- **Módulo:** `module fb_cloud` em go.mod — nunca fb_smartpick ou outro
- **Handlers:** `func NomeHandler(db *sql.DB) http.HandlerFunc { ... }` — sempre recebem `*sql.DB`
- **Registro:** em `main.go` via `mux.HandleFunc("/api/...", NomeHandler(db))`
- **Respostas JSON:** `json.NewEncoder(w).Encode(...)` com `Content-Type: application/json`
- **SQL:** 100% prepared statements — `db.Query("... WHERE x = $1", valor)` — NUNCA `fmt.Sprintf`
- **Erros para o cliente:** `http.Error(w, "mensagem", statusCode)` — log interno separado com `log.Printf`
- **Structs JSON:** tags sempre snake_case — `json:"field_name"`, `json:"is_active"`

### TypeScript (Frontend)

- **Modo strict:** ativo — sem `any` implícito, sem `!` desnecessário
- **Path alias:** `@/` aponta para `src/` (configurado no vite.config.ts)
- **Imports:** usar `@/` para imports absolutos dentro de `src/`
- **Tipos:** definir em `src/types/index.ts` — `Product`, `Notification`, `TenantProduct`, `Event`

### React

- **Dados do servidor:** sempre via TanStack Query — `useQuery` / `useMutation`
- **Estado local:** `useState` apenas para UI (modais, campos de form) — nunca para dados do servidor
- **Query keys** (hierarquia obrigatória):
  ```typescript
  ["products"]                 // lista pública
  ["products", tenantSlug]     // filtrada por tenant
  ["notifications"]            // lista pública
  ["admin", "products"]        // admin
  ["admin", "notifications"]   // admin
  ["admin", "metrics"]         // admin
  ```
- **Invalidação após mutação:** `queryClient.invalidateQueries({ queryKey: [...] })`
- **Formulários:** botão com `disabled + isPending` durante submissão
- **Loading:** Skeleton do Shadcn durante carregamento de listas

### Banco de Dados

- **Schema:** todas as tabelas novas no schema `portal` com prefixo `pt_`
- **PKs:** `uuid DEFAULT gen_random_uuid()`
- **Datas:** `TIMESTAMPTZ` no banco; ISO 8601 string na API (`"2026-04-08T14:30:00Z"`)
- **Migrations:** numeradas a partir de **100** (001–09x são herdadas do FB_SMARTPICK — não modificar)

### API

- **Formato de sucesso (lista):** `{"data": [...], "total": N}`
- **Formato de sucesso (item):** `{"data": {...}}`
- **Formato de erro:** `{"error": "mensagem legível"}`
- **Health check:** `{"status": "ok"}`
- **Auth header:** `Authorization: Bearer <token>` (JWT, expiração 8h)
- **CORS:** apenas `www.fbtax.cloud` como origem autorizada

---

## Código Herdado — NÃO MODIFICAR

Estes arquivos são herdados do FB_SMARTPICK e **não devem ser alterados**:

- `backend/services/email.go` — serviço SMTP completo
- `backend/middleware/auth.go` — `AuthMiddleware` JWT
- `backend/middleware/cors.go` — CORS configurado
- `backend/middleware/security.go` — `SecurityMiddleware`
- `backend/db/db.go` — `onDBConnected()` executa migrations automaticamente
- `frontend/src/contexts/AuthContext.tsx` — não alterar interface pública
- `frontend/src/components/ui/` — componentes Shadcn base — não modificar
- `frontend/src/lib/utils.ts` — `cn()` e helpers base

**Migrations herdadas (001–09x):** não renumerar, não modificar.

---

## Anti-Patterns — PROIBIDO

```
❌ fmt.Sprintf para montar queries SQL (SQL injection)
❌ Criar tabelas no schema portal sem prefixo pt_
❌ Numerar migrations com número < 100
❌ Duplicar lógica de auth fora do AuthMiddleware / AuthContext
❌ fetch() direto em vez de TanStack Query para dados do servidor
❌ Modificar components/ui/ (quebra Shadcn)
❌ Usar porta diferente de 8083
❌ Alterar go.mod para módulo diferente de fb_cloud
❌ Criar novo schema de banco fora de "portal" ou "public"
```

---

## Variáveis de Ambiente Obrigatórias

```bash
DB_URL=postgres://...
JWT_SECRET=...
SMTP_HOST=...
SMTP_PORT=...
SMTP_USER=...
SMTP_PASS=...
ADMIN_EMAIL=...
PORT=8083
```

---

## Usage Guidelines

**Para Agentes de IA:**
- Ler este arquivo antes de implementar qualquer código
- Seguir TODAS as regras exatamente como documentadas
- Em caso de dúvida, preferir a opção mais restritiva
- Consultar `_bmad-output/planning-artifacts/architecture.md` para decisões arquiteturais completas

**Para Humanos:**
- Manter este arquivo enxuto e focado nas necessidades dos agentes
- Atualizar quando o stack de tecnologia mudar
- Remover regras que se tornarem óbvias com o tempo

_Last Updated: 2026-04-08_
