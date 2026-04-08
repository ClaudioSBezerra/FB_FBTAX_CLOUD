---
stepsCompleted: [1, 2, 3, 4]
inputDocuments:
  - FBTAX_CLOUD/_bmad-output/planning-artifacts/prd.md
  - FBTAX_CLOUD/_bmad-output/planning-artifacts/architecture.md
  - FBTAX_CLOUD/_bmad-output/planning-artifacts/project-context.md
---

# FBTAX_CLOUD - Epic Breakdown

## Overview

This document provides the complete epic and story breakdown for FBTAX_CLOUD, decomposing the requirements from the PRD and Architecture into implementable stories.

---

## Requirements Inventory

### Functional Requirements

FR1: Visitante pode visualizar todos os produtos da plataforma fbtax.cloud com nome, ícone e descrição curta
FR2: Visitante pode identificar quais produtos estão ativos (contratados) e quais estão desabilitados para o seu tenant
FR3: Usuário pode acessar um produto ativo clicando no seu ícone (abre em nova aba no subdomínio correspondente)
FR4: Visitante sem tenant identificado pode visualizar todos os produtos como disponíveis para contratação
FR5: Sistema identifica o tenant do usuário via parâmetro de URL (?tenant=)
FR6: Visitante pode visualizar a lista de notificações publicadas (título, tipo, data, texto)
FR7: Sistema registra a visualização de cada notificação por sessão (para fins de contagem)
FR8: Notificações são exibidas em ordem cronológica decrescente (mais recentes primeiro)
FR9: Notificações podem ser do tipo: Aviso de Sistema, Nova Versão, Comunicado Geral
FR10: Visitante pode acessar formulário de contato comercial a partir do portal
FR11: Visitante pode preencher e enviar formulário com nome, e-mail e mensagem
FR12: Sistema envia o conteúdo do formulário por e-mail para o administrador
FR13: Sistema registra cada clique no botão de contato comercial (rastreamento)
FR14: Sistema registra cada envio de formulário concluído
FR15: Administrador pode autenticar-se no painel admin com login e senha
FR16: Administrador pode criar novo produto com nome, descrição, ícone e URL de destino
FR17: Administrador pode editar dados de um produto existente
FR18: Administrador pode ativar ou desativar um produto no portfólio público
FR19: Administrador pode definir quais produtos estão contratados por cada tenant
FR20: Administrador pode criar nova notificação com tipo, título e texto
FR21: Administrador pode editar uma notificação existente
FR22: Administrador pode publicar ou despublicar uma notificação
FR23: Administrador pode excluir uma notificação
FR24: Administrador pode visualizar o número de visualizações por notificação publicada
FR25: Administrador pode visualizar o total de cliques no CTA comercial
FR26: Administrador pode visualizar o total de formulários de contato enviados
FR27: Sistema responde a health check em GET /api/health
FR28: Portal público é acessível sem autenticação em www.fbtax.cloud
FR29: Painel admin é acessível exclusivamente em rota protegida (/admin)
FR30: Sistema serve meta tags de SEO (title, description, og:tags) para indexação

### NonFunctional Requirements

NFR1: Página principal (/) carrega com LCP < 2s em conexão padrão (Lighthouse)
NFR2: Bundle JavaScript do portal público é < 200kb gzipped
NFR3: Respostas da API do painel admin completam em < 1s para operações de leitura
NFR4: Ícones de produtos são servidos em formato SVG ou WebP otimizado
NFR5: Autenticação do painel admin utiliza JWT com expiração configurável (padrão 8h)
NFR6: Rota /admin retorna 401 para qualquer requisição sem token válido
NFR7: Senha do admin é armazenada com hash bcrypt (padrão herdado do FB_SMARTPICK)
NFR8: Todas as queries ao banco de dados utilizam prepared statements (sem concatenação de SQL)
NFR9: Variáveis de ambiente sensíveis (DB_URL, SMTP, JWT_SECRET) nunca são hardcodadas
NFR10: CORS configurado para aceitar apenas origens autorizadas (www.fbtax.cloud)
NFR11: Portal atende WCAG 2.1 nível AA — contraste mínimo 4.5:1 para texto normal
NFR12: Todos os ícones de produto possuem atributo aria-label descritivo
NFR13: Navegação por teclado funciona em todas as ações interativas do portal público
NFR14: Envio de e-mail via SMTP utiliza o serviço herdado do FB_SMARTPICK (services/email.go) sem modificação
NFR15: Links para subdomínios externos abrem em nova aba (target="_blank" com rel="noopener noreferrer")
NFR16: Parâmetro ?tenant= é lido e processado pelo frontend sem chamada de API adicional
NFR17: Sistema mantém uptime ≥ 99% na janela 7h–22h em dias úteis
NFR18: Health check GET /api/health responde em < 200ms
NFR19: Deploy via Coolify com health check no Docker Compose — restart automático em falha
NFR20: Banco de dados PostgreSQL com backup automático via Coolify/Hostinger

### Additional Requirements

- **Starter Template (Epic 1 Story 1):** Clone do repositório FB_SMARTPICK → reinicializar git → renomear módulo Go para `fb_cloud` → remover código de domínio SmartPick
- Módulo Go: `fb_cloud` (go.mod) — nunca fb_smartpick
- Porta API: 8083 (sem conflito com APU02:8081 e SmartPick:8082)
- Schema banco: `portal` com prefixo `pt_` para todas as tabelas novas
- Migrations numeradas a partir de 100 (001–09x herdadas, sem modificação)
- Seed data inicial: 4 produtos (Apuração Assistida, Simulador Fiscal, SmartPick, Farol) com URLs e ícones
- Código herdado intocável: services/email.go, middleware/auth.go, middleware/cors.go, AuthContext.tsx, components/ui/
- Subdomínio: www.fbtax.cloud — configurar DNS + Traefik via Coolify
- Variáveis de ambiente: DB_URL, JWT_SECRET, SMTP_HOST, SMTP_PORT, SMTP_USER, SMTP_PASS, ADMIN_EMAIL, PORT=8083

### FR Coverage Map

| FR | Epic |
|---|---|
| FR1–FR5 | Epic 2 |
| FR6–FR14 | Epic 3 |
| FR15–FR19 | Epic 4 |
| FR20–FR26 | Epic 5 |
| FR27 | Epic 1 |
| FR28 | Epic 1 (parcial), Epic 2 |
| FR29 | Epic 1 (parcial), Epic 4 |
| FR30 | Epic 1 (parcial), Epic 2 |

**NFR Coverage:**
| NFR | Epic |
|---|---|
| NFR1, NFR2, NFR4, NFR11–NFR13, NFR15, NFR16 | Epic 2 |
| NFR3 | Epic 4, Epic 5 |
| NFR5–NFR8 | Epic 4 |
| NFR9, NFR10, NFR17–NFR20 | Epic 1 |
| NFR14 | Epic 3 |

## Epic List

| # | Título | FRs |
|---|---|---|
| 1 | Fundação do Projeto e Deploy | FR27, FR28(p), FR29(p), FR30(p) |
| 2 | Vitrine Pública de Produtos | FR1–FR5, FR28, FR30 |
| 3 | Notificações Públicas e Contato Comercial | FR6–FR14 |
| 4 | Painel Admin: Autenticação e Gestão de Produtos | FR15–FR19, FR29 |
| 5 | Painel Admin: Notificações e Métricas | FR20–FR26 |

---

## Epic 1: Fundação do Projeto e Deploy

**Goal:** Ter o repositório FBTAX_CLOUD funcional, deployado e acessível em `www.fbtax.cloud`, com estrutura base herdada do FB_SMARTPICK e pronta para receber as features dos próximos epics.

**FRs:** FR27, FR28(parcial), FR29(parcial), FR30(parcial)
**NFRs:** NFR9, NFR10, NFR17, NFR18, NFR19, NFR20

---

### Story 1.1: Setup do Repositório Base

Como desenvolvedor,
Quero clonar o FB_SMARTPICK e adaptá-lo como base do FBTAX_CLOUD,
Para ter um projeto Go+React funcional na porta 8083 sem código de domínio SmartPick.

**Acceptance Criteria:**

**Given** o repositório FB_SMARTPICK clonado localmente
**When** o setup é executado
**Then** o git history é reinicializado (novo repositório limpo)
**And** o `go.mod` declara o módulo `fb_cloud` (sem nenhuma referência a `fb_smartpick`)
**And** a porta configurada é `8083` (variável `PORT` no `.env.example`)
**And** todos os arquivos de domínio SmartPick são removidos (handlers, models, páginas React específicas do SmartPick)
**And** o código herdado intocável permanece inalterado: `services/email.go`, `middleware/auth.go`, `middleware/cors.go`, `AuthContext.tsx`, `components/ui/`
**And** `go build ./...` e `npm run build` executam sem erros

---

### Story 1.2: Schema e Migrations Iniciais

Como sistema,
Quero criar o schema `portal` e as tabelas base no PostgreSQL,
Para que os dados de produtos e tenants possam ser persistidos e consultados.

**Acceptance Criteria:**

**Given** uma conexão PostgreSQL válida via `DB_URL`
**When** as migrations são executadas
**Then** o schema `portal` existe no banco de dados
**And** a migration 100 cria a tabela `pt_products` com colunas: `id`, `name`, `description`, `icon_url`, `destination_url`, `is_active`, `created_at`, `updated_at`
**And** a migration 101 cria a tabela `pt_tenants` com colunas: `id`, `slug`, `name`, `created_at`
**And** a migration 102 cria a tabela `pt_tenant_products` com colunas: `id`, `tenant_id`, `product_id`, `is_active`, `created_at`
**And** a migration 103 cria a tabela `pt_notifications` com colunas: `id`, `type`, `title`, `body`, `published`, `published_at`, `created_at`, `updated_at`
**And** a migration 104 cria a tabela `pt_events` com colunas: `id`, `type`, `notification_id` (nullable), `session_id`, `created_at`
**And** o seed insere 4 produtos: Apuração Assistida, Simulador Fiscal, SmartPick, Farol — com `is_active = true`
**And** migrations 001–09x herdadas não são modificadas
**And** todas as queries utilizam prepared statements (NFR8)

---

### Story 1.3: Health Check e Configuração de Ambiente

Como operador de infraestrutura,
Quero que o sistema responda ao health check e opere exclusivamente via variáveis de ambiente,
Para garantir observabilidade e segurança operacional.

**Acceptance Criteria:**

**Given** o servidor Go em execução
**When** `GET /api/health` é chamado
**Then** responde `200 OK` com body `{"status":"ok"}` em menos de 200ms (NFR18)
**And** nenhuma variável sensível (`DB_URL`, `JWT_SECRET`, `SMTP_*`) está hardcodada no código-fonte (NFR9)
**And** o arquivo `.env.example` documenta todas as variáveis: `DB_URL`, `JWT_SECRET`, `SMTP_HOST`, `SMTP_PORT`, `SMTP_USER`, `SMTP_PASS`, `ADMIN_EMAIL`, `PORT`
**And** CORS aceita apenas a origem `www.fbtax.cloud` (NFR10)
**And** `GET /` retorna o `index.html` do React (portal público acessível sem autenticação — FR28)

---

### Story 1.4: Deploy Coolify e DNS

Como administrador da plataforma,
Quero o portal deployado via Coolify e acessível em `www.fbtax.cloud`,
Para que clientes possam acessar o sistema em produção com restart automático em caso de falha.

**Acceptance Criteria:**

**Given** o repositório com Dockerfile e docker-compose.yml configurados
**When** o deploy é executado via Coolify
**Then** o serviço sobe na porta 8083 sem erros
**And** o DNS `www.fbtax.cloud` aponta para o servidor Hostinger via Traefik
**And** `GET https://www.fbtax.cloud/api/health` retorna `200 OK`
**And** o Docker Compose tem health check configurado com restart automático em falha (NFR19)
**And** o banco PostgreSQL tem backup automático habilitado via Coolify/Hostinger (NFR20)
**And** o uptime é monitorável via health check (suporte a NFR17)

---

## Epic 2: Vitrine Pública de Produtos

**Goal:** Visitante acessa `www.fbtax.cloud` e visualiza todos os produtos da plataforma, com diferenciação visual entre ativos e inativos para o seu tenant, podendo navegar para o produto ativo em um clique.

**FRs:** FR1–FR5, FR28, FR30
**NFRs:** NFR1, NFR2, NFR4, NFR11–NFR13, NFR15, NFR16

---

### Story 2.1: Endpoint de Produtos Públicos

Como visitante do portal,
Quero que a API retorne a lista de produtos com indicação de quais estão contratados pelo meu tenant,
Para que o frontend possa renderizar a vitrine corretamente.

**Acceptance Criteria:**

**Given** que existem produtos cadastrados na tabela `pt_products` com `is_active = true`
**When** `GET /api/portal/products` é chamado sem parâmetro de tenant
**Then** retorna `200 OK` com array de todos os produtos ativos: `id`, `name`, `description`, `icon_url`, `destination_url`
**And** cada produto inclui o campo `contracted: false` por padrão

**Given** que o parâmetro `?tenant=slug` é fornecido e o tenant existe em `pt_tenants`
**When** `GET /api/portal/products?tenant=slug` é chamado
**Then** produtos presentes em `pt_tenant_products` para esse tenant com `is_active = true` retornam com `contracted: true`
**And** os demais produtos retornam com `contracted: false`
**And** a query utiliza prepared statements (NFR8)

---

### Story 2.2: PortalPage e ProductCard

Como visitante do portal,
Quero ver os produtos em uma grade visual com ícone, nome e descrição,
Para identificar rapidamente o portfólio da plataforma fbtax.cloud.

**Acceptance Criteria:**

**Given** que `GET /api/portal/products` retorna a lista de produtos
**When** a página `/` é carregada
**Then** a `PortalPage` exibe uma grade de `ProductCard` com ícone, nome e descrição de cada produto
**And** o ícone é renderizado em formato SVG ou WebP otimizado (NFR4)
**And** o LCP da página é < 2s em conexão padrão (NFR1)
**And** o bundle JavaScript é < 200kb gzipped (NFR2)
**And** a página é acessível sem autenticação (FR28)

---

### Story 2.3: Lógica de Tenant e Navegação

Como cliente ativo do portal,
Quero que meus produtos contratados sejam destacados e clicáveis para acesso direto,
E que produtos não contratados exibam um CTA comercial discreto.

**Acceptance Criteria:**

**Given** que a URL contém `?tenant=slug`
**When** a `PortalPage` é carregada
**Then** o parâmetro `?tenant=` é lido pelo frontend sem chamada de API adicional (NFR16)
**And** produtos com `contracted: true` são exibidos com estilo ativo e link clicável para `destination_url`
**And** o link abre em nova aba com `target="_blank" rel="noopener noreferrer"` (NFR15)
**And** produtos com `contracted: false` são exibidos com estilo inativo (opacidade reduzida ou badge) e CTA comercial discreto (ex: "Saiba mais")

**Given** que a URL não contém `?tenant=`
**When** a `PortalPage` é carregada
**Then** todos os produtos são exibidos como disponíveis para contratação (FR4)
**And** nenhum produto é destacado como ativo (FR4)

---

### Story 2.4: SEO e Acessibilidade

Como visitante externo que chegou via busca orgânica,
Quero que o portal tenha meta tags corretas e seja navegável por teclado,
Para que o portal seja indexado e acessível a todos os usuários.

**Acceptance Criteria:**

**Given** que a página `/` é carregada
**When** o HTML é inspecionado
**Then** o `<head>` contém `<title>`, `<meta name="description">`, `og:title`, `og:description` e `og:url` com valores do portal (FR30)
**And** cada ícone de produto tem `aria-label` descritivo (ex: `aria-label="Acessar Apuração Assistida"`) (NFR12)
**And** o contraste de texto atende WCAG 2.1 AA — mínimo 4.5:1 (NFR11)
**And** todos os botões e links são acessíveis via tecla Tab e ativáveis via Enter/Space (NFR13)

---

## Epic 3: Notificações Públicas e Contato Comercial

**Goal:** Visitante vê notificações publicadas pelo admin e pode enviar formulário de contato. Sistema registra visualizações, cliques e envios para métricas.

**FRs:** FR6–FR14
**NFRs:** NFR14

---

### Story 3.1: Endpoint de Notificações Públicas

Como visitante do portal,
Quero acessar as notificações publicadas pelo administrador via API,
Para que o frontend possa exibi-las sem necessidade de autenticação.

**Acceptance Criteria:**

**Given** que existem notificações com `published = true` na tabela `pt_notifications`
**When** `GET /api/portal/notifications` é chamado
**Then** retorna `200 OK` com array de notificações em ordem cronológica decrescente (FR8)
**And** cada notificação contém: `id`, `type`, `title`, `body`, `published_at`
**And** o campo `type` é um dos valores: `system_alert`, `new_version`, `general` (FR9)
**And** notificações com `published = false` não são retornadas
**And** a query utiliza prepared statements (NFR8)

---

### Story 3.2: NotificationList no Frontend

Como visitante do portal,
Quero ver a lista de notificações na página principal com título, tipo, data e texto,
Para me manter informado sobre avisos e novidades da plataforma.

**Acceptance Criteria:**

**Given** que `GET /api/portal/notifications` retorna notificações
**When** a `PortalPage` é carregada
**Then** o componente `NotificationList` exibe cada notificação com título, tipo (label legível), data formatada e texto (FR6)
**And** notificações são exibidas em ordem decrescente — mais recente primeiro (FR8)
**And** tipos são exibidos com labels: "Aviso de Sistema", "Nova Versão", "Comunicado Geral" (FR9)
**And** caso não haja notificações, exibe mensagem vazia amigável

---

### Story 3.3: Rastreamento de Visualizações de Notificação

Como administrador,
Quero que o sistema registre automaticamente quando uma notificação é visualizada,
Para poder acompanhar o alcance de cada comunicado.

**Acceptance Criteria:**

**Given** que o visitante carrega a página com notificações visíveis
**When** `POST /api/portal/events` é chamado com `{"type": "notification_view", "notification_id": N}`
**Then** o sistema registra o evento na tabela `pt_events` com `type`, `notification_id` e `session_id` (gerado no frontend)
**And** o endpoint retorna `200 OK`
**And** o frontend envia o evento apenas uma vez por notificação por sessão de navegador (sem duplicação via sessionStorage) (FR7)
**And** a query utiliza prepared statements (NFR8)

---

### Story 3.4: Formulário de Contato Comercial

Como visitante interessado em contratar um produto,
Quero preencher um formulário de contato e receber confirmação de envio,
Para que o administrador seja notificado por e-mail e meu interesse seja registrado.

**Acceptance Criteria:**

**Given** que o visitante clica no botão de contato comercial
**When** o clique ocorre
**Then** o sistema registra o evento via `POST /api/portal/events` com `{"type": "cta_click"}` (FR13)
**And** o componente `ContactForm` é exibido com campos: nome, e-mail e mensagem (FR10, FR11)

**Given** que o visitante preenche e submete o formulário com dados válidos
**When** `POST /api/portal/contact` é chamado com `{name, email, message}`
**Then** o sistema envia e-mail ao `ADMIN_EMAIL` via `services/email.go` sem modificar o serviço herdado (FR12, NFR14)
**And** o sistema registra o envio via `POST /api/portal/events` com `{"type": "contact_submit"}` (FR14)
**And** o frontend exibe mensagem de sucesso ao visitante
**And** campos obrigatórios (nome, e-mail, mensagem) são validados antes do envio
**And** e-mail inválido exibe erro de validação sem submeter o formulário

---

## Epic 4: Painel Admin — Autenticação e Gestão de Produtos

**Goal:** Admin (Claudio) pode autenticar-se e gerenciar o catálogo de produtos e quais tenants têm acesso a cada um.

**FRs:** FR15–FR19, FR29
**NFRs:** NFR3, NFR5–NFR8

---

### Story 4.1: Login e Autenticação JWT

Como administrador,
Quero autenticar-me com login e senha e receber um token JWT,
Para acessar o painel admin de forma segura.

**Acceptance Criteria:**

**Given** que o admin navega para `/admin`
**When** não há token JWT válido no localStorage
**Then** o frontend redireciona para a tela `AdminLogin` (FR29, NFR6)
**And** a rota `/api/admin/*` retorna `401 Unauthorized` para qualquer requisição sem token (NFR6)

**Given** que o admin submete credenciais corretas em `POST /api/admin/login`
**When** o servidor valida login e senha (bcrypt — NFR7)
**Then** retorna `200 OK` com `{"token": "<jwt>"}` com expiração padrão de 8h (NFR5)
**And** o frontend armazena o token e redireciona para `AdminDashboard`

**Given** que o admin submete credenciais incorretas
**When** `POST /api/admin/login` é chamado
**Then** retorna `401 Unauthorized` sem revelar qual campo está errado

---

### Story 4.2: AdminDashboard e Listagem de Produtos

Como administrador autenticado,
Quero ver o painel principal com a lista de todos os produtos cadastrados,
Para ter uma visão geral do catálogo e partir para as ações de gestão.

**Acceptance Criteria:**

**Given** que o admin está autenticado com token válido
**When** acessa `/admin`
**Then** a tela `AdminDashboard` é exibida com lista de produtos via `GET /api/admin/products` (FR16 base)
**And** a resposta inclui todos os produtos (ativos e inativos) com: `id`, `name`, `description`, `icon_url`, `destination_url`, `is_active`
**And** a resposta completa em < 1s (NFR3)
**And** cada produto exibe seu estado ativo/inativo visualmente na listagem

**Given** que o token JWT expirou ou é inválido
**When** qualquer requisição autenticada é feita
**Then** o servidor retorna `401` e o frontend redireciona para `AdminLogin`

---

### Story 4.3: CRUD de Produtos

Como administrador autenticado,
Quero criar, editar e ativar/desativar produtos no catálogo,
Para manter o portfólio público sempre atualizado.

**Acceptance Criteria:**

**Given** que o admin preenche o formulário `AdminProductForm` com nome, descrição, `icon_url` e `destination_url`
**When** submete `POST /api/admin/products`
**Then** o produto é criado com `is_active = true` por padrão (FR16)
**And** retorna `201 Created` com o produto criado
**And** o produto aparece imediatamente na listagem do `AdminDashboard`

**Given** que o admin edita um produto existente
**When** submete `PUT /api/admin/products/:id`
**Then** os campos nome, descrição, `icon_url` e `destination_url` são atualizados (FR17)
**And** retorna `200 OK` com o produto atualizado
**And** prepared statements são usados em todas as queries (NFR8)

**Given** que o admin clica em ativar/desativar um produto
**When** `PATCH /api/admin/products/:id/toggle` é chamado
**Then** o campo `is_active` é alternado (FR18)
**And** retorna `200 OK` com o novo estado do produto
**And** produto desativado deixa de aparecer no `GET /api/portal/products`

---

### Story 4.4: Gestão de Tenant-Products

Como administrador autenticado,
Quero definir quais produtos cada tenant tem contratados,
Para que o portal público reflita corretamente o acesso de cada cliente.

**Acceptance Criteria:**

**Given** que o admin acessa a tela `AdminTenants`
**When** `GET /api/admin/tenants` é chamado
**Then** retorna lista de tenants cadastrados com seus produtos contratados (FR19)
**And** a resposta completa em < 1s (NFR3)

**Given** que o admin cria um novo tenant
**When** submete `POST /api/admin/tenants` com `{name, slug}`
**Then** o tenant é criado e aparece na listagem
**And** o `slug` deve ser único — retorna `409 Conflict` se duplicado

**Given** que o admin define os produtos de um tenant
**When** submete `PUT /api/admin/tenant-products` com `{tenant_id, product_ids[]}`
**Then** os registros em `pt_tenant_products` são atualizados para o tenant (FR19)
**And** `GET /api/portal/products?tenant=slug` passa a refletir os produtos contratados

---

## Epic 5: Painel Admin — Notificações e Métricas

**Goal:** Admin gerencia notificações (criar, editar, publicar, despublicar, excluir) e visualiza métricas de uso do portal (visualizações, cliques no CTA, formulários enviados).

**FRs:** FR20–FR26
**NFRs:** NFR3

---

### Story 5.1: CRUD de Notificações

Como administrador autenticado,
Quero criar, editar e excluir notificações no painel admin,
Para gerenciar os comunicados exibidos publicamente no portal.

**Acceptance Criteria:**

**Given** que o admin acessa a tela `AdminNotifications`
**When** `GET /api/admin/notifications` é chamado
**Then** retorna todas as notificações (publicadas e não publicadas) com: `id`, `type`, `title`, `body`, `published`, `published_at`, `created_at`
**And** a resposta completa em < 1s (NFR3)

**Given** que o admin preenche o formulário com tipo, título e texto
**When** submete `POST /api/admin/notifications`
**Then** a notificação é criada com `published = false` por padrão (FR20)
**And** retorna `201 Created` com a notificação criada
**And** prepared statements são usados (NFR8)

**Given** que o admin edita uma notificação existente
**When** submete `PUT /api/admin/notifications/:id`
**Then** os campos tipo, título e texto são atualizados (FR21)
**And** retorna `200 OK` com a notificação atualizada

**Given** que o admin clica em excluir uma notificação
**When** `DELETE /api/admin/notifications/:id` é chamado
**Then** a notificação é removida do banco (FR23)
**And** retorna `204 No Content`
**And** a notificação não aparece mais na listagem admin nem no endpoint público

---

### Story 5.2: Publicar e Despublicar Notificações

Como administrador autenticado,
Quero publicar e despublicar notificações com um único clique,
Para controlar o que é visível publicamente sem necessidade de excluir o conteúdo.

**Acceptance Criteria:**

**Given** que uma notificação tem `published = false`
**When** o admin clica em publicar e `PATCH /api/admin/notifications/:id/toggle` é chamado
**Then** `published` é alterado para `true` e `published_at` é registrado com o timestamp atual (FR22)
**And** retorna `200 OK` com o novo estado
**And** a notificação passa a aparecer em `GET /api/portal/notifications`

**Given** que uma notificação tem `published = true`
**When** o admin clica em despublicar e `PATCH /api/admin/notifications/:id/toggle` é chamado
**Then** `published` é alterado para `false` (FR22)
**And** a notificação deixa de aparecer em `GET /api/portal/notifications` imediatamente
**And** o conteúdo da notificação é preservado no banco

---

### Story 5.3: Métricas no Painel Admin

Como administrador autenticado,
Quero visualizar métricas de uso do portal (visualizações, cliques no CTA e formulários enviados),
Para acompanhar o alcance das notificações e o interesse comercial gerado.

**Acceptance Criteria:**

**Given** que o admin acessa a tela `AdminMetrics`
**When** `GET /api/admin/metrics/notifications` é chamado
**Then** retorna contagem de visualizações por notificação: `[{notification_id, title, view_count}]` (FR24)
**And** a resposta completa em < 1s (NFR3)

**When** `GET /api/admin/metrics/cta` é chamado
**Then** retorna o total de cliques no botão de contato comercial: `{"cta_clicks": N}` (FR25)

**When** `GET /api/admin/metrics/contacts` é chamado
**Then** retorna o total de formulários de contato enviados: `{"contact_submits": N}` (FR26)

**And** todas as métricas são calculadas a partir da tabela `pt_events` com prepared statements (NFR8)
**And** a tela `AdminMetrics` exibe os três indicadores de forma clara e legível
