# Requirements: FBTax Cloud — Módulo Financeiro (Portal Fortes Bezerra)

**Definido:** 2026-05-19
**Core Value:** O pagador recebe acesso, o inadimplente perde — automaticamente. Token de liberação válido vinculado a contrato ativo é o que mantém os produtos FB funcionando.

## v1 Requirements

### Cadastro de Empresa

- [ ] **EMP-01**: Admin pode cadastrar os dados da empresa Fortes Bezerra (CNPJ, razão social, endereço completo)
- [ ] **EMP-02**: Admin pode cadastrar dados bancários da empresa para recebimento (banco, agência, conta, tipo)
- [ ] **EMP-03**: Admin pode editar os dados da empresa e dados bancários

### Cadastro de Clientes

- [ ] **CLI-01**: Admin pode cadastrar um cliente com razão social e CNPJ principal
- [ ] **CLI-02**: Admin pode associar múltiplos CNPJs ao mesmo cliente (grupo empresarial)
- [ ] **CLI-03**: Admin pode cadastrar dados de contato do cliente (e-mail, telefone, responsável)
- [ ] **CLI-04**: Admin pode editar e inativar clientes existentes
- [ ] **CLI-05**: Admin pode pesquisar e listar clientes com filtros básicos (nome, CNPJ, status)

### Cadastro de Produtos

- [ ] **PROD-01**: Admin pode cadastrar produtos/módulos do portfólio FB (FB_APU02, FB_APU04, FB_SMARTPICK, FB_FAROL)
- [ ] **PROD-02**: Cada produto suporta 5 planos de precificação: Lite, Standard, Premium, Enterprise, Sob Demanda
- [ ] **PROD-03**: Admin pode definir o preço de cada plano por produto
- [ ] **PROD-04**: Admin pode editar preços e planos de produtos existentes

### Cadastro de Contratos

- [ ] **CONT-01**: Admin pode criar um contrato vinculando um cliente a um ou mais produtos com planos específicos
- [ ] **CONT-02**: Um contrato pode abranger múltiplos CNPJs do grupo do cliente sob um único pacote/pagamento
- [ ] **CONT-03**: Cada contrato tem data de início, valor total e periodicidade de renovação
- [ ] **CONT-04**: Admin pode editar contratos ativos e registrar aditivos
- [ ] **CONT-05**: Admin pode visualizar histórico de contratos por cliente

### Motor de Tokens

- [ ] **TOKEN-01**: Sistema gera automaticamente um Token de liberação ao criar ou renovar um contrato
- [ ] **TOKEN-02**: Token tem validade de 45 dias a partir da data de geração
- [ ] **TOKEN-03**: Sistema notifica o admin (e-mail) quando um token está a 15 dias de vencer
- [ ] **TOKEN-04**: Após vencer, o token entra em período de carência de 15 dias (ainda válido, mas em alerta)
- [ ] **TOKEN-05**: Após os 15 dias de carência (total: 60 dias), o token é suspenso automaticamente e os produtos associados perdem acesso
- [ ] **TOKEN-06**: Admin pode reativar manualmente um token suspenso após confirmar o pagamento

### API de Validação de Token

- [ ] **API-01**: Endpoint autenticado que os produtos externos (FB_APU02, FB_APU04, FB_SMARTPICK, FB_FAROL) consultam para verificar se possuem token ativo
- [ ] **API-02**: Endpoint retorna status do token: ativo, em carência, suspenso, não encontrado
- [ ] **API-03**: Endpoint indica a data de vencimento do token para que os produtos possam alertar internamente
- [ ] **API-04**: Endpoint é autenticado por chave de API (API Key) por contrato/produto

### Painel Admin Interno

- [ ] **ADMIN-01**: Admin Fortes Bezerra acessa painel privado com visão geral: total de contratos, tokens ativos, tokens próximos do vencimento, tokens suspensos
- [ ] **ADMIN-02**: Admin pode acessar, criar e editar todos os cadastros (empresa, clientes, produtos, contratos) a partir do painel
- [ ] **ADMIN-03**: Admin pode visualizar e gerenciar tokens: status, histórico, reativação manual
- [ ] **ADMIN-04**: Painel exibe alertas para tokens em carência e suspensos

### Portal do Cliente

- [ ] **PORT-01**: Cliente acessa portal com login próprio e visualiza apenas seus contratos
- [ ] **PORT-02**: Portal exibe status dos tokens ativos, datas de vencimento e alertas de carência
- [ ] **PORT-03**: Portal exibe histórico de contratos e produtos contratados
- [ ] **PORT-04**: Portal exibe instrução de contato com a Fortes Bezerra em caso de dúvidas ou irregularidade

### Navegação

- [ ] **NAV-01**: Interface existente do FBTax Cloud exibe botão "Acessar Fortes Bezerra" que direciona ao portal
- [ ] **NAV-02**: Roteamento separa claramente a área pública do portal cliente da área administrativa interna

## v2 Requirements

### Cobrança Automática (pós-spike de gateway)

- **BILL-01**: Sistema emite boleto bancário vinculado ao contrato automaticamente
- **BILL-02**: Sistema gera QR Code PIX para pagamento do contrato
- **BILL-03**: Sistema registra baixa automática ao detectar pagamento via PIX
- **BILL-04**: Sistema baixa boleto automaticamente ao receber confirmação do banco
- **BILL-05**: Ao confirmar pagamento, sistema gera novo Token de liberação automaticamente (sem ação manual)
- **BILL-06**: Gateway: avaliar AbacatePay (PIX nativo simples), Asaas (boleto+PIX+NFS-e integrada), PlugBoleto (multi-banco)

### Nota Fiscal de Serviço (NFS-e)

- **NFSE-01**: Sistema emite NFS-e automaticamente ao faturar um contrato
- **NFSE-02**: NFS-e é emitida para o município de Aparecida de Goiânia — GO (ABRASF 2.04, Série RPS 9)
- **NFSE-03**: Integração via provedor (Focus NFe, Webmania ou NFE.io) com certificado A1
- **NFSE-04**: NFS-e é enviada por e-mail ao cliente automaticamente

### Recorrência

- **REC-01**: Sistema dispara cobrança automática N dias antes do vencimento do contrato
- **REC-02**: Sistema envia e-mail de aviso de vencimento ao cliente
- **REC-03**: Fluxo de inadimplência: aviso → carência → suspensão automática

## Out of Scope

| Feature | Motivo |
|---------|--------|
| Integração OAuth / SSO para login | JWT customizado é suficiente para v1 |
| App mobile | Web-first; portal do cliente via browser |
| Cobrança via cartão de crédito | Fora do modelo de negócio atual (boleto/PIX) |
| Multi-moeda | Operação 100% em BRL |
| Suporte a outros municípios para NFS-e | Apenas Aparecida de Goiânia por ora |
| Chat/suporte in-app | Canal externo (WhatsApp/e-mail) suficiente para v1 |
| Relatórios financeiros avançados | Admin verá dados básicos no v1; BI/dashboard avançado é v3+ |

## Traceability

| Requisito | Fase | Status |
|-----------|------|--------|
| EMP-01 | Phase 1 — Fundação do Módulo | Pendente |
| EMP-02 | Phase 1 — Fundação do Módulo | Pendente |
| EMP-03 | Phase 1 — Fundação do Módulo | Pendente |
| CLI-01 | Phase 2 — Cadastros Base | Pendente |
| CLI-02 | Phase 2 — Cadastros Base | Pendente |
| CLI-03 | Phase 2 — Cadastros Base | Pendente |
| CLI-04 | Phase 2 — Cadastros Base | Pendente |
| CLI-05 | Phase 2 — Cadastros Base | Pendente |
| PROD-01 | Phase 2 — Cadastros Base | Pendente |
| PROD-02 | Phase 2 — Cadastros Base | Pendente |
| PROD-03 | Phase 2 — Cadastros Base | Pendente |
| PROD-04 | Phase 2 — Cadastros Base | Pendente |
| CONT-01 | Phase 2 — Cadastros Base | Pendente |
| CONT-02 | Phase 2 — Cadastros Base | Pendente |
| CONT-03 | Phase 2 — Cadastros Base | Pendente |
| CONT-04 | Phase 2 — Cadastros Base | Pendente |
| CONT-05 | Phase 2 — Cadastros Base | Pendente |
| TOKEN-01 | Phase 3 — Motor de Tokens | Pendente |
| TOKEN-02 | Phase 3 — Motor de Tokens | Pendente |
| TOKEN-03 | Phase 3 — Motor de Tokens | Pendente |
| TOKEN-04 | Phase 3 — Motor de Tokens | Pendente |
| TOKEN-05 | Phase 3 — Motor de Tokens | Pendente |
| TOKEN-06 | Phase 3 — Motor de Tokens | Pendente |
| API-01 | Phase 4 — API de Validação | Pendente |
| API-02 | Phase 4 — API de Validação | Pendente |
| API-03 | Phase 4 — API de Validação | Pendente |
| API-04 | Phase 4 — API de Validação | Pendente |
| ADMIN-01 | Phase 5 — Painel Admin | Pendente |
| ADMIN-02 | Phase 5 — Painel Admin | Pendente |
| ADMIN-03 | Phase 5 — Painel Admin | Pendente |
| ADMIN-04 | Phase 5 — Painel Admin | Pendente |
| NAV-01 | Phase 5 — Painel Admin | Pendente |
| NAV-02 | Phase 5 — Painel Admin | Pendente |
| PORT-01 | Phase 6 — Portal do Cliente | Pendente |
| PORT-02 | Phase 6 — Portal do Cliente | Pendente |
| PORT-03 | Phase 6 — Portal do Cliente | Pendente |
| PORT-04 | Phase 6 — Portal do Cliente | Pendente |

**Cobertura:**
- Requisitos v1: 36 total
- Mapeados para fases: 36/36 ✓
- Não mapeados: 0

---
*Requisitos definidos: 2026-05-19*
*Última atualização: 2026-05-19 — traceability preenchida após criação do roadmap*
