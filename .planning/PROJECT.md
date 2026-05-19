# FBTax Cloud — Módulo Financeiro (Portal Fortes Bezerra)

## What This Is

FBTax Cloud é uma plataforma SaaS de gestão fiscal para empresas brasileiras, hospedada em `www.fbtax.cloud`. O próximo milestone adiciona um **Módulo Financeiro** — o "Portal Fortes Bezerra" — que funciona simultaneamente como sistema de gestão de contratos/licenças dos produtos FB e como portal de acesso para clientes. O módulo vive dentro do mesmo repositório e deploy do FBTax Cloud (novas rotas Go + novas páginas React), acessível via botão "Acessar Fortes Bezerra" na interface existente.

## Core Value

**O pagador recebe acesso, o inadimplente perde — automaticamente.** Um token de liberação válido vinculado a um contrato ativo é o que mantém os produtos FB_APU02, FB_APU04, FB_SMARTPICK e FB_FAROL funcionando. Esse fluxo (contrato → pagamento → token → produto) é o coração do módulo.

## Requirements

### Validated

*(O que já existe no FBTax Cloud hoje — inferido do mapeamento de codebase)*

- ✓ Autenticação JWT com refresh tokens — `backend/handlers/auth.go`
- ✓ Multi-tenant: environments → enterprise_groups → companies — PostgreSQL + migrations
- ✓ Backend Go (stdlib `net/http`, PostgreSQL, bcrypt, UUID) — `backend/`
- ✓ Frontend React 18 + TypeScript + shadcn/ui + Tailwind — `frontend/`
- ✓ Migrations SQL auto-aplicadas no boot — `backend/migrations/`
- ✓ Sistema de roles/auth via middleware `withAuth(handler, role)` — `backend/main.go`
- ✓ Integração com Receita Federal (CBS/IBS, débitos fiscais) — `rfb_credentials`, `rfb_debitos`
- ✓ ERP Bridge Python (Oracle → backend via HTTP/API Key) — `erp-bridge-aws/`
- ✓ Geração de PDF via `maroto/v2` — `backend/services/`
- ✓ Envio de e-mail — `backend/services/email.go`
- ✓ Deploy via Coolify + Traefik + Docker — `docker-compose.yml`, `docker-compose.prod.yml`
- ✓ CI/CD GitHub Actions → Coolify webhook — `.github/workflows/deploy-production.yml`

### Active

*(Módulo Financeiro — v1: Contratos, Tokens e Licenciamento)*

**Cadastros base:**
- [ ] Cadastro da empresa Fortes Bezerra: CNPJ, razão social, endereço completo, dados bancários para recebimento
- [ ] Cadastro de clientes: CNPJ(s), razão social, contato, dados do grupo empresarial
- [ ] Cadastro de produtos/módulos com planos: Lite, Standard, Premium, Enterprise, Sob Demanda — com preços por plano
- [ ] Cadastro de contratos: cliente + 1 ou mais produtos + 1 ou mais CNPJs do grupo → único pacote/pagamento

**Motor de tokens e licenciamento:**
- [ ] Geração de Token de liberação por contrato (com validade e vencimento de 45 dias)
- [ ] API pública de validação de token — consultada pelos produtos externos diariamente
- [ ] Lógica de carência: 15 dias após vencimento antes da suspensão automática
- [ ] Suspensão automática de token ao atingir 60 dias sem pagamento (45 + 15)
- [ ] Reativação automática do token ao confirmar pagamento

**Interfaces:**
- [ ] Painel admin interno (Fortes Bezerra): gestão completa de clientes, produtos, contratos e tokens
- [ ] Portal do cliente: visualização de contratos ativos, tokens, datas de vencimento e status de pagamento
- [ ] Acesso via botão "Acessar Fortes Bezerra" na interface existente do FBTax Cloud

### Out of Scope

- NFS-e (Nota Fiscal de Serviço Eletrônico) — v2, após validação do MVP de contratos/tokens
- Boleto bancário automático — v2, após spike de gateway (AbacatePay / Asaas / PlugBoleto)
- PIX automático e baixa automática de pagamento — v2, integrado com gateway escolhido
- Geração automática de token pós-pagamento — v2, depende da integração bancária
- Recorrência / cobrança automática de renovação — v2
- Integração com Abacate Pay, Asaas ou outro gateway — spike antes do v2

## Context

**Stack atual:**
- Backend: Go 1.26, stdlib `net/http`, PostgreSQL 15, JWT (`golang-jwt/jwt/v5`), bcrypt, UUID, maroto (PDF)
- Frontend: React 18, TypeScript 5.2, Vite 5.2, shadcn/ui (Radix + Tailwind CSS), React Router v6
- Deploy: Docker (Coolify + Traefik), PostgreSQL em container, GitHub Actions CI/CD
- Sem ORM — SQL raw com `lib/pq`. Padrão de injeção via closures `withDB(handler)` / `withAuth(handler, role)`

**Modelo de dados relevante:**
- Multi-tenant existente (`environments`, `enterprise_groups`, `companies`) provavelmente será reaproveitado para representar clientes do módulo financeiro
- 107+ migrations SQL — o módulo financeiro adicionará novas migrations para as entidades de contratos, tokens e licenças

**Produtos externos que consumirão a API de tokens:**
- FB_APU02, FB_APU04, FB_SMARTPICK, FB_FAROL
- Consultam diariamente para verificar tokens novos com vencimento de 45 dias
- Precisam de endpoint autenticado (API Key ou JWT) acessível por esses processos externos

**Licenciamento hoje:** nenhum. Este módulo é o primeiro sistema de controle de acesso/licença dos produtos FB.

**v2 — Integrações a pesquisar (spike antes de implementar):**
- Gateway de cobranças: AbacatePay (mais simples, PIX nativo), Asaas (boleto + PIX + NFS-e integrada), PlugBoleto/TecnoSpeed
- NFS-e: município Aparecida de Goiânia (GO), padrão ABRASF 2.04, Série RPS 9 — provedores: Focus NFe, Webmania, NFE.io

## Constraints

- **Tech stack**: Go (backend) + React/TypeScript (frontend) — sem adicionar novos runtimes no v1
- **Deploy**: mesma infraestrutura Coolify/Docker — módulo financeiro não cria novo container separado no v1
- **Banco de dados**: PostgreSQL existente — novas entidades via migrations SQL
- **Autenticação**: aproveitar o sistema JWT/roles existente — admin interno usará role específica
- **Sem gateway externo no v1**: cobranças manuais até o v2 resolver o spike de gateway

## Key Decisions

| Decisão | Rationale | Outcome |
|----------|-----------|---------|
| Módulo dentro do FBTax Cloud (não serviço separado) | Reutiliza auth, deploy, DB e infra existentes — menos overhead no v1 | — Pending |
| v1 sem boleto/PIX | Validar o fluxo de contratos e tokens primeiro; gateway requer spike e decisão de parceiro bancário | — Pending |
| NFS-e no v2 | Emissão manual enquanto MVP valida o modelo de negócio; integração ABRASF requer homologação | — Pending |
| Gateway de pagamento: spike antes de comprometer | AbacatePay x Asaas x PlugBoleto — analisar custo/complexidade/suporte antes de implementar | — Pending |
| Token com 45 dias de validade + 15 de carência | Modelo de negócio definido: cliente tem 60 dias no total para regularizar antes da suspensão | — Pending |

## Evolution

Este documento evolui a cada transição de fase e milestone.

**Após cada fase (via `/gsd-transition`):**
1. Requisitos invalidados? → Mover para Out of Scope com motivo
2. Requisitos validados (shipped)? → Mover para Validated com referência de fase
3. Novos requisitos emergiram? → Adicionar em Active
4. Decisões a registrar? → Adicionar em Key Decisions
5. "What This Is" ainda preciso? → Atualizar se houver drift

**Após cada milestone (via `/gsd:complete-milestone`):**
1. Revisão completa de todas as seções
2. Core Value ainda é a prioridade certa?
3. Auditoria de Out of Scope — motivos ainda válidos?
4. Atualizar Context com o estado atual

---
*Last updated: 2026-05-19 após inicialização do projeto*
