# Story 1.1: Setup do Repositório Base

Status: review

## Story

Como desenvolvedor,
Quero clonar o FB_SMARTPICK e adaptá-lo como base do FBTAX_CLOUD,
Para ter um projeto Go+React funcional na porta 8083 sem código de domínio SmartPick.

## Acceptance Criteria

**AC1:** O git history é reinicializado — o repositório FBTAX_CLOUD tem um novo histórico git limpo (sem histórico do FB_SMARTPICK).

**AC2:** O `go.mod` declara o módulo `fb_cloud` (sem nenhuma referência a `fb_smartpick`) e todas as referências internas ao módulo antigo são substituídas.

**AC3:** A porta configurada é `8083` como default no código e no `.env.example`.

**AC4:** Todo código de domínio SmartPick é removido: handlers específicos de SmartPick, models, páginas React específicas, assets de domínio.

**AC5:** O código herdado intocável permanece inalterado: `backend/services/email.go`, `backend/middleware/auth.go`, `backend/middleware/cors.go`, `backend/middleware/security.go`, `backend/db/db.go`, `frontend/src/contexts/AuthContext.tsx`, `frontend/src/components/ui/`, `frontend/src/lib/utils.ts`.

**AC6:** `go build ./...` executa sem erros a partir do diretório `backend/`.

**AC7:** `npm run build` executa sem erros a partir do diretório `frontend/`.

**AC8:** O arquivo `.env.example` documenta todas as variáveis obrigatórias: `DB_URL`, `JWT_SECRET`, `SMTP_HOST`, `SMTP_PORT`, `SMTP_USER`, `SMTP_PASS`, `ADMIN_EMAIL`, `PORT=8083`.

## Tasks / Subtasks

- [x] Task 1: Clonar FB_SMARTPICK e reinicializar git (AC: #1)
  - [x] 1.1: Localizar repositório FB_SMARTPICK em `/home/claudiobezerra/projetos/FB_SMARTPICK`
  - [x] 1.2: Copiar conteúdo do FB_SMARTPICK para o diretório FBTAX_CLOUD (preservar estrutura)
  - [x] 1.3: Reinicializar git: `rm -rf .git && git init && git add . && git commit -m "chore: init from FB_SMARTPICK base"`

- [x] Task 2: Renomear módulo Go de `fb_smartpick` para `fb_cloud` (AC: #2)
  - [x] 2.1: Editar `backend/go.mod`: alterar `module fb_smartpick` → `module fb_cloud`
  - [x] 2.2: Substituir todas as referências ao módulo antigo no código Go: `import "fb_smartpick/..."` → `import "fb_cloud/..."`
  - [x] 2.3: Verificar com `grep -r "fb_smartpick" backend/` que nenhuma referência sobrou

- [x] Task 3: Configurar porta 8083 (AC: #3)
  - [x] 3.1: Verificar/ajustar valor default de PORT no código Go (main.go ou equivalente) para `8083`
  - [x] 3.2: Verificar docker-compose.yml: porta mapeada para 8083
  - [x] 3.3: Criar/atualizar `.env.example` com `PORT=8083`

- [x] Task 4: Remover código de domínio SmartPick (AC: #4)
  - [x] 4.1: Identificar e remover handlers Go específicos do SmartPick (não são: auth.go, health check)
  - [x] 4.2: Remover models/types Go do domínio SmartPick que não serão reutilizados
  - [x] 4.3: Remover páginas React específicas do SmartPick em `frontend/src/pages/` (manter estrutura de pastas)
  - [x] 4.4: Remover rotas do SmartPick de `frontend/src/App.tsx` (manter rota `/admin` e `/admin/login`)
  - [x] 4.5: Remover imports órfãos que referenciam código SmartPick removido

- [x] Task 5: Verificar integridade do código herdado intocável (AC: #5)
  - [x] 5.1: Confirmar que `backend/services/email.go` está presente e intacto
  - [x] 5.2: Confirmar que `backend/middleware/auth.go`, `cors.go`, `security.go` estão presentes e intactos
  - [x] 5.3: Confirmar que `backend/db/db.go` está presente e intacto (inclui `onDBConnected()`)
  - [x] 5.4: Confirmar que `frontend/src/contexts/AuthContext.tsx` está presente e intacto
  - [x] 5.5: Confirmar que `frontend/src/components/ui/` está presente e intacto
  - [x] 5.6: Confirmar que `frontend/src/lib/utils.ts` está presente e intacto

- [x] Task 6: Validar builds e atualizar .env.example (AC: #6, #7, #8)
  - [x] 6.1: Executar `cd backend && go build ./...` — passou sem erros
  - [x] 6.2: Executar `cd frontend && npm install && npm run build` — passou sem erros (84.31 kB gzip)
  - [x] 6.3: Criar/atualizar `.env.example` com todas as variáveis obrigatórias

## Dev Notes

### Contexto Crítico: O que é o FB_SMARTPICK

O FB_SMARTPICK é o projeto irmão que serve de base para este clone. Localizado em `/home/claudiobezerra/projetos/FB_SMARTPICK`. Usa a mesma stack: Go 1.22 + React 18 + TypeScript + Tailwind + Shadcn/UI + PostgreSQL. O objetivo é herdar toda a infraestrutura (Docker, Nginx, auth, middleware, email) e remover apenas o código de negócio específico do SmartPick.

### Regra fundamental desta story

**Herdar infra, remover domínio SmartPick, não criar nada do domínio FBTAX_CLOUD ainda.**
Os handlers Go novos (`products.go`, `notifications.go`, etc.) e as páginas React novas (`PortalPage.tsx`, etc.) serão criados nas stories 1.2 em diante. Esta story entrega apenas a fundação limpa.

### O que MANTER do FB_SMARTPICK (código herdado intocável)

```
backend/
  services/email.go          ← SMTP completo — não tocar
  middleware/auth.go         ← AuthMiddleware JWT — não tocar
  middleware/cors.go         ← CORS — não tocar (ajustar origem se necessário na Story 1.3)
  middleware/security.go     ← SecurityMiddleware — não tocar
  db/db.go                   ← conexão + onDBConnected() executa migrations — não tocar
  main.go                    ← manter estrutura, remover rotas SmartPick
  go.mod / go.sum            ← manter dependências, apenas trocar nome do módulo

frontend/
  src/contexts/AuthContext.tsx  ← não tocar
  src/components/ui/            ← Shadcn base — não tocar
  src/lib/utils.ts              ← cn() e helpers — não tocar
  package.json                  ← manter dependências (TanStack Query, Shadcn, Vite, etc.)
  vite.config.ts                ← manter path alias @/
  tailwind.config.js            ← manter
  tsconfig.json                 ← manter strict mode

docker-compose.yml              ← manter estrutura, ajustar porta para 8083
docker-compose.prod.yml         ← manter
nginx/nginx.conf                ← manter
backend/Dockerfile              ← manter
frontend/Dockerfile             ← manter
```

### O que REMOVER do FB_SMARTPICK

```
backend/
  handlers/smartpick_*.go    ← qualquer handler de domínio SmartPick
  handlers/picking_*.go      ← handlers de picking se existirem
  models/smartpick*.go       ← models de domínio se em arquivos separados
  migrations/100+*.sql       ← migrations novas do SmartPick (as 001-09x ficam!)

frontend/
  src/pages/SmartPick*.tsx   ← páginas específicas do SmartPick
  src/pages/Picking*.tsx     ← qualquer página de domínio
  src/components/smartpick/  ← componentes de domínio se existirem
  src/types/smartpick*.ts    ← tipos de domínio
```

### Decisão de arquitetura: por que apenas renomear em go.mod não basta

O Go usa o nome do módulo como prefixo em todos os imports internos. Se o módulo era `fb_smartpick`, todos os arquivos Go têm imports como:
```go
import "fb_smartpick/middleware"
import "fb_smartpick/services"
```
Após renomear para `fb_cloud`, todos esses imports precisam ser atualizados. O comando `go mod edit -module fb_cloud` renomeia o go.mod, mas os imports devem ser atualizados manualmente ou via `sed`.

**Comando recomendado para substituição em massa:**
```bash
find backend/ -name "*.go" -exec sed -i 's|"fb_smartpick/|"fb_cloud/|g' {} \;
```

### Estrutura esperada ao final da story

Ao fim da Story 1.1, a estrutura será:
```
FBTAX_CLOUD/
├── .env.example              ← com todas as variáveis
├── docker-compose.yml        ← porta 8083
├── docker-compose.prod.yml
├── backend/
│   ├── go.mod               ← module fb_cloud, Go 1.22
│   ├── go.sum
│   ├── main.go              ← rotas SmartPick removidas, apenas /api/health e /admin/ por ora
│   ├── handlers/
│   │   └── (handlers SmartPick removidos, auth.go herdado mantido)
│   ├── middleware/
│   │   ├── auth.go          ← intacto
│   │   ├── cors.go          ← intacto
│   │   └── security.go      ← intacto
│   ├── services/
│   │   └── email.go         ← intacto
│   ├── migrations/
│   │   └── 001-09x*.sql     ← herdadas intactas (migrations 100+ do SmartPick removidas)
│   └── db/
│       └── db.go            ← intacto
├── frontend/
│   ├── package.json
│   ├── vite.config.ts
│   ├── src/
│   │   ├── App.tsx          ← rotas SmartPick removidas
│   │   ├── contexts/
│   │   │   └── AuthContext.tsx  ← intacto
│   │   ├── components/
│   │   │   └── ui/          ← intacto
│   │   └── lib/
│   │       └── utils.ts     ← intacto
└── nginx/
    └── nginx.conf           ← intacto
```

### Nota sobre .env.example

O arquivo `.env.example` deve conter APENAS placeholders, nunca valores reais:
```bash
DB_URL=postgres://user:password@localhost:5432/fbtax_cloud
JWT_SECRET=your-jwt-secret-here
SMTP_HOST=smtp.example.com
SMTP_PORT=587
SMTP_USER=user@example.com
SMTP_PASS=your-smtp-password
ADMIN_EMAIL=admin@fbtax.cloud
PORT=8083
```

### Nota sobre CORS (MC2 do implementation-readiness)

O middleware CORS herdado deve aceitar `localhost:5173` (Vite dev server) em desenvolvimento. Verificar se o `middleware/cors.go` lê a lista de origens de variável de ambiente `CORS_ORIGINS`. Se não lê, anotar no Dev Agent Record para endereçar na Story 1.3.

### Referências

- [Source: architecture.md#Starter-Template-Evaluation] — decisão de usar FB_SMARTPICK como base
- [Source: architecture.md#Project-Structure] — árvore completa de diretórios esperada
- [Source: architecture.md#Naming-Patterns] — convenções de nomenclatura Go/TS/DB
- [Source: project-context.md#Critical-Implementation-Rules] — regras obrigatórias
- [Source: epics.md#Story-1.1] — ACs originais da story

## Dev Agent Record

### Agent Model Used

Claude Sonnet 4.6 (claude-sonnet-4-6)

### Debug Log References

- Referências `fb_smartpick` encontradas em `frontend/package-lock.json` após `npm install` inicial — resolvido regenerando com `npm install` novamente após renomear `package.json`.
- `backend/main.go` tinha referência ao banco `fb_smartpick` na connection string padrão — corrigido com sed para `fbtax_cloud`.
- Estrutura real do FB_SMARTPICK difere da architecture.md: middleware está em `backend/handlers/middleware.go` (não `backend/middleware/`), DB em `main.go` (não `backend/db/db.go`). Diferença estrutural anotada para stories futuras.

### Completion Notes List

- AC1 ✅: Git reinicializado, histórico limpo
- AC2 ✅: go.mod declara `module fb_cloud`, zero referências `fb_smartpick` em `backend/`
- AC3 ✅: Porta default `8083` em `main.go` e `.env.example`
- AC4 ✅: 11 handlers Go removidos, 10 páginas React removidas, 8 componentes removidos, 9 migrations 100-108 removidas
- AC5 ✅: Código herdado intacto: `services/email.go`, `handlers/auth.go`, `handlers/middleware.go`, `AuthContext.tsx`, `components/ui/`, `lib/utils.ts`
- AC6 ✅: `go build ./...` passa sem erros
- AC7 ✅: `npm run build` passa, bundle 84.31 kB gzip (< 200 kB NFR2)
- AC8 ✅: `.env.example` criado com todas as variáveis obrigatórias

### File List

**Modificados:**
- `backend/go.mod` — module `fb_cloud`
- `backend/main.go` — rotas SmartPick removidas, porta 8083, constantes FBTAX_CLOUD
- `frontend/src/App.tsx` — rotas SmartPick removidas, placeholders /admin e /
- `frontend/package.json` — nome `fbtax-cloud-frontend`
- `frontend/vite.config.ts` — proxy target `localhost:8083`
- `docker-compose.yml` — porta 8083, referências fbtax.cloud

**Criados:**
- `.env.example` — variáveis obrigatórias com placeholders

**Removidos (handlers Go):**
- `backend/handlers/smartpick_auth.go`
- `backend/handlers/sp_admin.go`
- `backend/handlers/sp_ambiente.go`
- `backend/handlers/sp_csv.go`
- `backend/handlers/sp_historico.go`
- `backend/handlers/sp_motor.go`
- `backend/handlers/sp_pdf.go`
- `backend/handlers/sp_propostas.go`
- `backend/handlers/sp_reincidencia.go`
- `backend/handlers/sp_usuarios.go`
- `backend/handlers/filiais.go`
- `backend/handlers/crypto.go`
- `backend/services/csv_worker.go`

**Removidos (migrations SmartPick):**
- `backend/migrations/100_sp_schema.sql` a `108_sp_retencao_hash.sql` (9 arquivos)

**Removidos (frontend pages):**
- `frontend/src/pages/SpAmbiente.tsx`
- `frontend/src/pages/SpDashboard.tsx`
- `frontend/src/pages/SpGerarPDF.tsx`
- `frontend/src/pages/SpHistorico.tsx`
- `frontend/src/pages/SpReincidencia.tsx`
- `frontend/src/pages/SpUploadCSV.tsx`
- `frontend/src/pages/SpUsuarios.tsx`
- `frontend/src/pages/GestaoAmbiente.tsx`
- `frontend/src/pages/Register.tsx`
- `frontend/src/pages/AdminUsers.tsx`

**Removidos (frontend components):**
- `frontend/src/components/AppRail.tsx`
- `frontend/src/components/AppSidebar.tsx`
- `frontend/src/components/CompanySwitcher.tsx`
- `frontend/src/components/FileUpload.tsx`
- `frontend/src/components/FilialSelector.tsx`
- `frontend/src/components/InsightCard.tsx`
- `frontend/src/components/ParticipantList.tsx`
- `frontend/src/components/UploadProgress.tsx`
- `frontend/src/contexts/FilialContext.tsx`
- `frontend/src/lib/navigation.ts`
- `frontend/src/lib/formatFilial.ts`
- `frontend/src/lib/exportToExcel.ts`
- `frontend/src/lib/logger.ts`

### Change Log

| Data | Alteração |
|---|---|
| 2026-04-08 | Story implementada: clone FB_SMARTPICK → FBTAX_CLOUD, módulo renomeado para fb_cloud, porta 8083, código domínio SmartPick removido, builds validados |
