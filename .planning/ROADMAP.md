# Roadmap: FBTax Cloud — Módulo Financeiro (Portal Fortes Bezerra)

## Overview

Este milestone adiciona o Módulo Financeiro ao FBTax Cloud existente. Parte dos cadastros base (empresa, clientes, produtos) para chegar ao coração do produto: o motor de tokens que mantém os produtos FB funcionando. Cada fase entrega uma capacidade end-to-end utilizável antes de avançar. O fluxo central é contrato → token → validação → acesso.

## Phases

**Phase Numbering:**
- Integer phases (1, 2, 3): Planned milestone work
- Decimal phases (2.1, 2.2): Urgent insertions (marked with INSERTED)

- [ ] **Phase 1: Fundação do Módulo** - Migrations, schema `financeiro`, dados da empresa Fortes Bezerra
- [ ] **Phase 2: Cadastros Base** - Clientes (multi-CNPJ), produtos (5 planos) e contratos
- [ ] **Phase 3: Motor de Tokens** - Geração, validade 45 dias, carência 15 dias, suspensão automática, reativação e alertas por e-mail
- [ ] **Phase 4: API de Validação** - Endpoint autenticado por API Key consultado pelos produtos externos
- [ ] **Phase 5: Painel Admin** - Dashboard interno Fortes Bezerra + botão "Acessar Fortes Bezerra" na UI existente
- [ ] **Phase 6: Portal do Cliente** - Login próprio, visão de contratos/tokens/alertas

## Phase Details

### Phase 1: Fundação do Módulo
**Goal**: O banco de dados contém o schema do módulo financeiro e o admin pode cadastrar e editar os dados da empresa Fortes Bezerra
**Mode:** mvp
**Depends on**: Nada (primeira fase — brownfield sobre FBTax Cloud existente)
**Requirements**: EMP-01, EMP-02, EMP-03
**Success Criteria** (what must be TRUE):
  1. Migrations criam as tabelas do módulo financeiro sem quebrar o schema existente
  2. Admin pode salvar CNPJ, razão social e endereço completo da empresa Fortes Bezerra
  3. Admin pode salvar dados bancários (banco, agência, conta, tipo) para recebimento
  4. Admin pode editar empresa e dados bancários e as alterações persistem no banco
**Plans**: 2 plans

Plans:
- [ ] 01-01-PLAN.md — Migrations SQL (schema financeiro + tabelas) + handlers Go CRUD (GET/POST/PUT empresa e dados bancários) + registro de rotas em main.go
- [ ] 01-02-PLAN.md — Página React EmpresaPage com dois formulários independentes + rota protegida em App.tsx + verificação humana end-to-end

### Phase 2: Cadastros Base
**Goal**: Admin pode cadastrar clientes com múltiplos CNPJs, produtos com planos e preços, e criar contratos vinculando cliente a produtos
**Mode:** mvp
**Depends on**: Phase 1
**Requirements**: CLI-01, CLI-02, CLI-03, CLI-04, CLI-05, PROD-01, PROD-02, PROD-03, PROD-04, CONT-01, CONT-02, CONT-03, CONT-04, CONT-05
**Success Criteria** (what must be TRUE):
  1. Admin pode cadastrar, editar e inativar clientes com razão social, CNPJ principal, CNPJs adicionais do grupo e dados de contato
  2. Admin pode pesquisar clientes por nome, CNPJ ou status com resultados corretos
  3. Admin pode cadastrar produtos (FB_APU02, FB_APU04, FB_SMARTPICK, FB_FAROL) com 5 planos de preço (Lite, Standard, Premium, Enterprise, Sob Demanda) e editar preços
  4. Admin pode criar um contrato vinculando um cliente a um ou mais produtos/planos, abrangendo múltiplos CNPJs do grupo, com data de início, valor e periodicidade
  5. Admin pode visualizar o histórico de contratos de um cliente e registrar aditivos em contratos ativos
**Plans**: TBD
**UI hint**: yes

### Phase 3: Motor de Tokens
**Goal**: Contratos ativos geram tokens automaticamente e o sistema gerencia o ciclo de vida completo: validade, carência, suspensão e reativação
**Mode:** mvp
**Depends on**: Phase 2
**Requirements**: TOKEN-01, TOKEN-02, TOKEN-03, TOKEN-04, TOKEN-05, TOKEN-06
**Success Criteria** (what must be TRUE):
  1. Ao criar ou renovar um contrato, o sistema gera automaticamente um token de liberação com validade de 45 dias
  2. Com 15 dias para vencer, o sistema envia e-mail de alerta ao admin e o token aparece em estado "próximo do vencimento"
  3. Após vencer, o token muda para estado "em carência" e permanece tecnicamente válido por mais 15 dias
  4. Após 60 dias totais sem pagamento, o token muda para "suspenso" automaticamente e os produtos associados perdem acesso
  5. Admin pode reativar manualmente um token suspenso, gerando novo token ativo vinculado ao contrato
**Plans**: TBD

### Phase 4: API de Validação
**Goal**: Produtos externos (FB_APU02, FB_APU04, FB_SMARTPICK, FB_FAROL) podem consultar o status do token de um contrato via endpoint autenticado por API Key
**Mode:** mvp
**Depends on**: Phase 3
**Requirements**: API-01, API-02, API-03, API-04
**Success Criteria** (what must be TRUE):
  1. Endpoint retorna status correto do token (ativo, em carência, suspenso, não encontrado) para uma API Key válida
  2. Endpoint retorna a data de vencimento do token para que o produto possa exibir alertas internos
  3. Chamadas com API Key inválida ou ausente recebem HTTP 401
  4. Cada contrato/produto tem sua própria API Key gerada e gerenciável pelo admin
**Plans**: TBD

### Phase 5: Painel Admin
**Goal**: Admin Fortes Bezerra acessa painel privado com visão geral do módulo e gerenciamento completo de cadastros e tokens; botão "Acessar Fortes Bezerra" aparece na interface FBTax Cloud existente
**Mode:** mvp
**Depends on**: Phase 4
**Requirements**: ADMIN-01, ADMIN-02, ADMIN-03, ADMIN-04, NAV-01, NAV-02
**Success Criteria** (what must be TRUE):
  1. Admin vê dashboard com totais de contratos, tokens ativos, tokens próximos do vencimento e tokens suspensos
  2. Admin pode acessar e gerenciar todos os cadastros (empresa, clientes, produtos, contratos) a partir do painel
  3. Admin pode visualizar lista de tokens com status, histórico e executar reativação manual
  4. Painel exibe alertas destacados para tokens em carência e suspensos
  5. Usuário autenticado no FBTax Cloud vê o botão "Acessar Fortes Bezerra" e o roteamento separa claramente a área admin da área do portal cliente
**Plans**: TBD
**UI hint**: yes

### Phase 6: Portal do Cliente
**Goal**: Clientes acessam portal com login próprio e visualizam apenas seus contratos, tokens e status de pagamento
**Mode:** mvp
**Depends on**: Phase 5
**Requirements**: PORT-01, PORT-02, PORT-03, PORT-04
**Success Criteria** (what must be TRUE):
  1. Cliente faz login com credenciais próprias e vê apenas seus contratos (isolamento de dados correto)
  2. Portal exibe status dos tokens ativos, datas de vencimento e alertas de carência em estado visível
  3. Portal exibe histórico de contratos e produtos contratados
  4. Portal exibe instrução clara de contato com a Fortes Bezerra em caso de dúvidas ou irregularidade
**Plans**: TBD
**UI hint**: yes

## Progress

**Execution Order:**
Phases execute in numeric order: 1 → 2 → 3 → 4 → 5 → 6

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 1. Fundação do Módulo | 0/2 | Not started | - |
| 2. Cadastros Base | 0/TBD | Not started | - |
| 3. Motor de Tokens | 0/TBD | Not started | - |
| 4. API de Validação | 0/TBD | Not started | - |
| 5. Painel Admin | 0/TBD | Not started | - |
| 6. Portal do Cliente | 0/TBD | Not started | - |
