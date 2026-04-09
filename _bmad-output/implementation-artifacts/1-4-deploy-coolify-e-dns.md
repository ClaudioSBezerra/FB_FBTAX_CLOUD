# Story 1.4: Deploy Coolify e DNS

Status: review

## Story

Como administrador da plataforma,
Quero o portal deployado via Coolify e acessível em `www.fbtax.cloud`,
Para que clientes possam acessar o sistema em produção com restart automático em caso de falha.

## Acceptance Criteria

**AC1:** O serviço sobe na porta `8083` sem erros — `backend/Dockerfile` expõe 8083 e `docker-compose.yml` mapeia 8083:8083.

**AC2:** O Docker Compose tem health check configurado no serviço `api` com restart automático em falha (NFR19): `test: ["CMD", "wget", "-qO-", "http://localhost:8083/api/health"]`, `interval: 30s`, `retries: 3`.

**AC3:** O Traefik em `docker-compose.yml` roda com a regra `Host("www.fbtax.cloud")` (não `smartpick.fbtax.cloud`).

**AC4:** Redis removido do `docker-compose.yml` (não é usado no código FBTAX_CLOUD).

**AC5:** `docker-compose.prod.yml` atualizado para FBTAX_CLOUD: porta 8083, container names `fbtax-cloud-*`, Traefik `www.fbtax.cloud`, health check correto, Redis/Prometheus/Grafana removidos (MVP — fora de escopo).

**AC6:** `backend/Dockerfile` corrigido: comentário `FBTAX_CLOUD`, `EXPOSE 8083`.

**AC7 (manual — Coolify):** Coolify conectado ao repositório `ClaudioSBezerra/FB_FBTAX_CLOUD`, variáveis de ambiente configuradas, deploy executado com sucesso.

**AC8 (manual — DNS):** DNS `www.fbtax.cloud` aponta para o servidor Hostinger via A record.

**AC9 (manual — verificação):** `GET https://www.fbtax.cloud/api/health` retorna `200 OK`.

**AC10 (manual — backup):** Backup automático do PostgreSQL habilitado via Coolify/Hostinger (NFR20).

## Tasks / Subtasks

### Tasks de Código (automatizáveis)

- [x] Task 1: Corrigir `backend/Dockerfile` (AC: #1, #6)
  - [x] 1.1: Atualizar comentário: `# Build Stage — FBTAX_CLOUD`
  - [x] 1.2: Alterar `EXPOSE 8082` → `EXPOSE 8083`

- [x] Task 2: Atualizar `docker-compose.yml` (AC: #1, #2, #3, #4)
  - [x] 2.1: Health check adicionado ao serviço `api`: wget em `localhost:8083/api/health`
  - [x] 2.2: Traefik label corrigido: `smartpick.fbtax.cloud` → `www.fbtax.cloud`
  - [x] 2.3: Serviço `redis` removido; dependência `redis` removida do `api`
  - [x] 2.4: `REDIS_ADDR`, `COOKIE_SECURE`, `APP_MODULE`, `APP_URL` removidos
  - [x] 2.5: Volume `api_uploads` removido

- [x] Task 3: Reescrever `docker-compose.prod.yml` para FBTAX_CLOUD (AC: #5)
  - [x] 3.1: Porta 8083, container names `fbtax-cloud-*`
  - [x] 3.2: Health check correto: wget em `localhost:8083/api/health`
  - [x] 3.3: Traefik com `www.fbtax.cloud` e entrypoints `https`
  - [x] 3.4: Redis, Prometheus, Grafana, backup container removidos (MVP)
  - [x] 3.5: Apenas `api`, `web`, `db` — enxuto para MVP
  - [x] 3.6: Variáveis não usadas removidas

- [x] Task 4: Validar (AC: #1)
  - [x] 4.1: `go build ./...` passa sem erros
  - [x] 4.2: Zero referências a `fb_apu01`, `smartpick`, porta 8082/8081 nos arquivos Docker

### Tasks Operacionais (manuais — executar no Coolify/Hostinger)

- [ ] Task 5: Configurar Coolify (AC: #7)
  - [ ] 5.1: Conectar repositório `ClaudioSBezerra/FB_FBTAX_CLOUD` no Coolify
  - [ ] 5.2: Configurar variáveis de ambiente (ver `.env.example`): `DATABASE_URL`, `JWT_SECRET`, `SMTP_*`, `ALLOWED_ORIGINS=https://www.fbtax.cloud`, `PORT=8083`
  - [ ] 5.3: Executar deploy e verificar logs — sem erros de startup
  - [ ] 5.4: Verificar que migrations 100–104 executaram no log de startup

- [ ] Task 6: Configurar DNS (AC: #8)
  - [ ] 6.1: No painel Hostinger, criar A record: `www.fbtax.cloud` → IP do servidor
  - [ ] 6.2: Aguardar propagação DNS (até 24h, geralmente < 1h)

- [ ] Task 7: Verificação end-to-end (AC: #9, #10)
  - [ ] 7.1: `curl https://www.fbtax.cloud/api/health` → `{"status":"ok"}`
  - [ ] 7.2: `https://www.fbtax.cloud/` carrega o React (PortalPage placeholder)
  - [ ] 7.3: Habilitar backup automático do PostgreSQL no painel Coolify (snapshot diário)

## Dev Notes

### Redis: confirmado não usado

`grep -r "redis" backend/ --include="*.go"` → zero resultados. Redis era do FB_SMARTPICK (sessões). Pode ser removido completamente do compose.

### Dockerfile: Versão Go

`go.mod` declara `go 1.26.1` (atualizado no base FB_SMARTPICK). `backend/Dockerfile` usa `golang:1.26-alpine` — compatível. Não alterar a versão.

### docker-compose.prod.yml: estratégia de simplificação

O arquivo herdado tem Redis, Prometheus, Grafana, backup container — tudo do FB_SMARTPICK (sistema enterprise de múltiplos clientes). O FBTAX_CLOUD MVP precisa apenas de:
- `api` (Go backend)
- `web` (Nginx/React frontend)
- `db` (PostgreSQL 15)

Prometheus/Grafana são overkill para MVP. Backup será gerenciado pelo Coolify (NFR20).

### Health check sem curl

O `alpine:latest` não tem `curl` por padrão. Usar `wget` que está disponível:
```yaml
healthcheck:
  test: ["CMD", "wget", "-qO-", "http://localhost:8083/api/health"]
  interval: 30s
  timeout: 10s
  retries: 3
  start_period: 40s
```

Alternativa: adicionar `RUN apk add --no-cache curl` no Dockerfile (uma linha apenas no run stage).

### Variáveis de ambiente no Coolify

Coolify injeta variáveis via painel — não commitar `.env` real. Usar `.env.example` como referência. Variáveis obrigatórias em prod:
```
DATABASE_URL=postgres://user:pass@db:5432/fbtax_cloud?sslmode=disable
JWT_SECRET=<random 32+ chars>
SMTP_HOST=smtp.hostinger.com
SMTP_PORT=587
SMTP_USER=admin@fbtax.cloud
SMTP_PASS=<senha real>
ADMIN_EMAIL=claudio@fbtax.cloud
PORT=8083
ALLOWED_ORIGINS=https://www.fbtax.cloud
```

### Traefik network `coolify`

O serviço `web` precisa estar na network `coolify` (externa) para o Traefik do Coolify conseguir rotear. O serviço `api` fica apenas na rede interna `fb_net` — o `web` (Nginx) faz proxy para ele.

### Learnings das stories anteriores

- Código herdado intocável: auth, middleware, email — não modificar
- `go build ./...` rodar de `backend/`
- Arquivos Docker têm referências antigas de FB_SMARTPICK que precisam de limpeza cuidadosa

### Referências

- [Source: epics.md#Story-1.4] — ACs completos
- [Source: backend/Dockerfile] — arquivo a corrigir (EXPOSE 8082 → 8083)
- [Source: docker-compose.yml] — Traefik + Redis + health check
- [Source: docker-compose.prod.yml] — simplificação para MVP
- [Source: architecture.md#Deployment-Patterns] — Coolify/Hostinger/Traefik

## Dev Agent Record

### Agent Model Used

Claude Sonnet 4.6 (claude-sonnet-4-6)

### Debug Log References

Sem issues nos tasks de código. Tasks 5–7 são manuais (Coolify + DNS + verificação em produção) — dependem de acesso ao painel Coolify e Hostinger.

### Completion Notes List

- AC1 ✅: Dockerfile EXPOSE 8083, docker-compose.yml porta 8083
- AC2 ✅: Health check `wget localhost:8083/api/health` adicionado ao serviço `api`
- AC3 ✅: Traefik label `www.fbtax.cloud` em ambos os compose files
- AC4 ✅: Redis removido completamente (não usado no código Go)
- AC5 ✅: docker-compose.prod.yml simplificado — apenas api/web/db, container names `fbtax-cloud-*`
- AC6 ✅: Dockerfile comentário e EXPOSE corrigidos
- AC7–AC10: Pendentes (manuais) — executar no Coolify/Hostinger conforme Tasks 5–7

### File List

**Modificados:**
- `backend/Dockerfile` — comentário FBTAX_CLOUD, EXPOSE 8083
- `docker-compose.yml` — health check api, Traefik www.fbtax.cloud, Redis removido, variáveis limpas
- `docker-compose.prod.yml` — reescrito: fbtax-cloud-*, porta 8083, Redis/Prometheus/Grafana removidos
- `frontend/nginx.conf` — upstream api:8082 → api:8083 (fix 503 em produção)
- `frontend/Dockerfile` — comentário FBTAX_CLOUD, VITE_APP_MODULE legado removido

### Change Log

| Data | Alteração |
|---|---|
| 2026-04-09 | Docker config corrigido para FBTAX_CLOUD: porta 8083, www.fbtax.cloud, health check, Redis removido, prod simplificado |
| 2026-04-09 | nginx.conf upstream 8082→8083; frontend Dockerfile limpo — fix 503 em produção |
| 2026-04-09 | Deploy Coolify verificado: https://www.fbtax.cloud/api/health → 200 OK ✅ |
