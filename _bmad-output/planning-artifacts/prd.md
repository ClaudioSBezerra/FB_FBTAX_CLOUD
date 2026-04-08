---
stepsCompleted: [1, 2, 3, 4, 7, 8, 9, 10, 11]
inputDocuments:
  - FBTAX_CLOUD/_bmad-output/planning-artifacts/product-brief-FBTAX_CLOUD-2026-04-08.md
workflowType: 'prd'
lastStep: 0
---

# Product Requirements Document - FBTAX_CLOUD

**Author:** Claudio
**Date:** 2026-04-08

---

## Executive Summary

O FBTAX_CLOUD é o portal hub público da plataforma fbtax.cloud — ponto central de entrada para distribuidores brasileiros que utilizam os produtos da FBTax. O portal resolve a fragmentação de acesso: clientes com múltiplos produtos (Apuração Assistida, Simulador Fiscal, SmartPick, Farol) hoje precisam acessar cada sistema individualmente por URLs distintas, sem identidade visual unificada de plataforma.

O portal exibe todos os produtos fbtax.cloud como ícones modernos e clicáveis, diferenciando produtos contratados (ativos) de não contratados (com CTA comercial discreto). Um painel de notificações gerenciado pelo proprietário (Claudio) permite comunicar avisos de sistema, novidades e atualizações a todos os clientes sem necessidade de deploy. Para visitantes externos que chegam via Google ou `fortesbezerra.com.br`, o portal funciona como vitrine institucional da plataforma.

### What Makes This Special

- **Identidade de plataforma:** pela primeira vez, todos os produtos fbtax.cloud têm uma "casa" visual unificada
- **Zero atrito para o cliente ativo:** um único bookmark para todos os sistemas
- **Canal de comunicação próprio:** notificações publicadas sem dependência de e-mail ou WhatsApp
- **Vitrine passiva de vendas:** produtos não contratados geram interesse comercial de forma orgânica
- **Leveza arquitetural:** portal independente, sem banco de dados de usuário final — manutenção mínima
- **Stack familiar:** React + Tailwind + Shadcn herdados do FB_SMARTPICK — desenvolvimento acelerado

## Project Classification

**Technical Type:** Web Application (SPA)
**Domain:** General SaaS
**Complexity:** Baixa-Média
**Project Context:** Greenfield — novo repositório, design system e stack herdados do FB_SMARTPICK

---

## Success Criteria

### User Success

- Cliente ativo acessa qualquer produto fbtax.cloud em ≤ 2 cliques a partir do portal
- Zero necessidade de memorizar URLs individuais dos subdomínios
- Visitante externo encontra o portfólio completo e o CTA comercial sem fricção
- Notificações publicadas são lidas antes de o usuário acessar o produto afetado
- Admin (Claudio) publica notificação em < 5 minutos sem necessidade de deploy

### Business Success

**Curto prazo (0–30 dias após lançamento):**
- 100% dos clientes ativos acessando o portal — **gate para V2**
- Portal comunicado a todos os clientes existentes no dia do lançamento

**Médio prazo (1–6 meses):**
- Canal de notificações ativo com pelo menos 1 comunicado por mês
- CTA comercial rastreando cliques e formulários preenchidos

**Longo prazo (6–12 meses):**
- Portal reconhecido como ponto de entrada padrão da plataforma
- Leads qualificados gerados passivamente via portal

### Technical Success

- Deploy via Coolify na infraestrutura Hostinger — mesmo padrão dos produtos existentes
- Porta e subdomínio próprios: `www.fbtax.cloud` sem conflito com outros serviços
- Health check herdado do padrão FB_SMARTPICK (`GET /api/health`)
- Uptime ≥ 99% (janela 7h–22h dias úteis)
- Tempo de carregamento da página principal < 2s (conexão padrão)
- Zero dependência de banco de dados de usuário final no portal público

### Measurable Outcomes

| Métrica | Alvo | Prazo | Instrumento |
|---|---|---|---|
| Clientes ativos no portal | 100% | 30 dias | Analytics / feedback direto |
| Visualizações por notificação | Rastreado | Contínuo | Contador no painel admin |
| Cliques no CTA comercial | Rastreado | Contínuo | Contador no painel admin |
| Formulários de contato | Rastreado | Contínuo | Registro no painel admin |
| Uptime | ≥ 99% | Contínuo | Coolify health check |
| Page load | < 2s | Lançamento | Lighthouse / browser |

## Product Scope

### MVP — Minimum Viable Product

- Vitrine de produtos com 4 produtos (Apuração, Simulador, SmartPick, Farol)
- Diferenciação visual: produto contratado (ativo) vs. não contratado (desabilitado + CTA)
- Painel de notificações público (avisos, novidades, comunicados)
- CTA comercial com formulário (nome, e-mail, mensagem) + rastreamento de cliques
- Painel admin protegido por login/senha: CRUD de produtos, CRUD de notificações, métricas básicas
- Identificação de tenant via parâmetro de URL (sem login de usuário final no portal)
- Deploy via Coolify/Hostinger em `www.fbtax.cloud`

### Growth Features (Post-MVP)

- Área do cliente personalizada com status em tempo real dos sistemas contratados
- Notificações por e-mail automáticas para lista de contatos do tenant
- Analytics de uso por produto e por tenant
- Página de roadmap público ("em breve")

### Vision (Future)

- Multi-tenant dashboard: visão consolidada de todos os produtos por cliente
- Integração SSO com `fortesbezerra.com.br`
- Portal como marketplace de módulos adicionais da plataforma

---

## User Journeys

---

**Jornada 1: Ana Santos — A Contadora que Perdeu o Bookmark**

Ana trabalha num escritório de contabilidade em São Paulo e gerencia a apuração fiscal de três distribuidores clientes. Toda semana ela abre o notebook, vai na barra de endereços e digita `apuracao.fbtax.cloud` de memória — mas esta manhã, com o fechamento do mês se aproximando, ela acidentalmente acessa o sistema errado e perde 3 minutos redirecionando.

Quando o escritório adota o FBTAX_CLOUD como ponto de entrada, Ana faz um único bookmark: `www.fbtax.cloud`. Na manhã seguinte, ela abre o portal, vê o ícone da Apuração em destaque (produto contratado, ativo e clicável) e em dois cliques está onde precisa estar. Antes de entrar, nota um aviso de notificação: "Atualização na tabela de CSLL — vigência 05/04." Ela lê antes de começar — algo que nunca aconteceria se a informação viesse só por e-mail.

Um mês depois, Ana nem lembra mais as URLs dos sistemas. O portal virou o começo do dia de trabalho.

**Esta jornada revela:**
- Vitrine de produtos com ícones clicáveis
- Diferenciação visual entre produto ativo e inativo
- Painel de notificações visível antes do acesso ao produto

---

**Jornada 2: Ricardo Almeida — O Controller que Quer Ver Tudo de Uma Vez**

Ricardo é gerente de controladoria do Grupo JC. Ele usa a Apuração para o fechamento fiscal e o SmartPick para acompanhar a eficiência dos CDs. Durante semanas, ele mantinha duas abas abertas no navegador — uma para cada sistema — e precisava lembrar qual empresa tinha acesso a qual produto.

Com o FBTAX_CLOUD, Ricardo acessa `www.fbtax.cloud?tenant=grupojc` e vê um painel limpo: Apuração e SmartPick ativos, Simulador e Farol visíveis mas desabilitados. Em uma tela ele entende exatamente o que a empresa contratou. Quando vê o Farol desabilitado, lê a descrição do produto e decide encaminhar para o CEO como sugestão de expansão.

Três semanas depois, o CEO aprova a contratação do Farol — uma conversa iniciada por Ricardo que nunca teria acontecido sem a vitrine dos produtos não contratados.

**Esta jornada revela:**
- Identificação de tenant via parâmetro de URL
- Produtos não contratados visíveis com descrição e CTA
- Experiência de "catálogo de produtos da empresa"

---

**Jornada 3: Carlos Mendes — O Gestor que Só Quer Chegar no SmartPick**

Carlos gerencia o CD de Ribeirão Preto do Grupo JC. Ele não é o público principal do portal — ele só quer abrir o SmartPick e trabalhar. Numa segunda-feira, recebe um WhatsApp do Ricardo: "O link do portal tá aqui, é por onde a gente acessa agora."

Carlos abre `www.fbtax.cloud?tenant=grupojc`, vê o ícone do SmartPick, clica e está no sistema em segundos. Ele não lê as notificações — até que uma semana depois aparece um banner amarelo no portal: "SmartPick — manutenção programada sexta 22h." Carlos vê, avisa a equipe, e o ciclo de upload daquela semana é antecipado. Problema evitado.

**Esta jornada revela:**
- Acesso rápido (≤ 2 cliques) ao produto desejado
- Notificações de aviso de manutenção com destaque visual
- Portal funcional mesmo para usuário sem interesse no hub

---

**Jornada 4: Felipe Costa — O TI que Avalia Antes de Recomendar**

Felipe é analista de TI de um distribuidor médio em Minas Gerais. O diretor financeiro pediu para ele avaliar "aquele sistema de apuração fiscal que o pessoal de SP tá usando." Felipe vai ao Google, digita "apuração fiscal distribuidor SaaS" e um dos resultados é `www.fbtax.cloud`.

Ele abre o portal e em 90 segundos entende o portfólio completo: não é só apuração — é uma plataforma com simulador fiscal, gestão logística de picking e painel de performance comercial. Mais do que esperava. Ele vê o botão "Fale Conosco" no rodapé, clica, preenche nome, e-mail e uma mensagem técnica sobre integração com o Winthor. Claudio recebe o formulário no mesmo dia.

Duas semanas depois, o distribuidor está em processo de contratação do SmartPick — uma venda que começou porque o portal comunicou mais do que o prospect esperava encontrar.

**Esta jornada revela:**
- Portal acessível e indexável por Google (SEO básico)
- Portfólio completo visível para visitantes não autenticados
- CTA comercial com formulário simples e rastreamento de cliques
- Descrição clara de cada produto para avaliação rápida

---

**Jornada 5: Claudio Bezerra — O Admin que Controla sem Deploy**

É quinta-feira à noite. Claudio identifica que o servidor do SmartPick passará por manutenção na madrugada de sexta. Antes, ele teria mandado um WhatsApp para cada cliente — processo manual, esquecível, sem histórico.

Agora ele abre `www.fbtax.cloud/admin`, faz login com suas credenciais, cria uma notificação: tipo "Aviso de Sistema", título "Manutenção SmartPick — Sexta 01h às 03h", texto explicativo. Publica. Em menos de 3 minutos, todos os clientes que acessarem o portal na sexta verão o aviso antes de tentar entrar no SmartPick.

Na manhã seguinte, Claudio acessa o painel de métricas: 12 visualizações da notificação, 2 cliques no CTA comercial. Ele sabe que a comunicação funcionou — sem precisar perguntar para ninguém.

**Esta jornada revela:**
- Painel admin com login/senha exclusivo
- CRUD de notificações com tipo, título e texto
- Contador de visualizações por notificação
- Métricas de CTA no painel admin

---

### Journey Requirements Summary

| Capacidade | Jornadas que exigem |
|---|---|
| Vitrine de produtos com ícones | Ana, Ricardo, Carlos, Felipe |
| Diferenciação ativo/inativo por tenant | Ana, Ricardo, Carlos |
| Identificação de tenant via URL | Ricardo, Carlos |
| Painel de notificações público | Ana, Carlos |
| CTA comercial + formulário | Felipe |
| Rastreamento de cliques e visualizações | Felipe, Claudio |
| Painel admin protegido (CRUD) | Claudio |
| Descrição de produtos para visitantes | Felipe, Ricardo |
| SEO / indexabilidade | Felipe |

---

## Web Application Specific Requirements

### Project-Type Overview

O FBTAX_CLOUD é uma SPA (Single Page Application) React, seguindo exatamente o mesmo padrão tecnológico do FB_SMARTPICK. Foco em simplicidade máxima: carregamento estático, sem real-time, sem complexidade desnecessária.

### Technical Architecture Considerations

- **Rendering:** SPA com React 18 + Vite — build estático servido via Nginx (padrão Coolify)
- **Roteamento:** `react-router-dom` — rotas: `/` (portal público), `/admin` (painel admin), `/admin/login`
- **Estado:** mínimo — sem estado global complexo; dados do admin via TanStack Query
- **Backend:** Go `net/http` padrão — apenas endpoints necessários para o painel admin e formulário de contato
- **Banco de dados:** PostgreSQL — apenas para admin (produtos, notificações, métricas de cliques/visualizações)
- **Sem WebSockets / SSE** — nenhum dado em tempo real necessário

### Browser Matrix

- Chrome, Firefox, Safari, Edge — versões modernas (últimas 2)
- Responsivo para desktop e mobile (visualização, não app)
- Sem suporte a IE

### SEO Strategy

- Meta tags básicas: `title`, `description`, `og:title`, `og:description`, `og:image`
- `sitemap.xml` estático gerado no build
- `robots.txt` permitindo indexação completa
- URLs limpas e semânticas

### Performance Targets

- LCP (Largest Contentful Paint) < 2s em conexão padrão
- Bundle JS < 200kb gzipped (portal público — página leve)
- Imagens dos ícones de produtos: SVG ou WebP otimizado

### Accessibility Level

- WCAG 2.1 AA — padrão: contraste adequado, navegação por teclado, atributos `aria` nos ícones de produto

### Implementation Considerations

- **Zero autenticação de usuário final no portal público** — página completamente estática para visitantes
- **Painel admin:** autenticação simples com JWT (padrão herdado do FB_SMARTPICK)
- **Formulário de contato:** POST para endpoint Go → envio de e-mail via SMTP herdado + registro no banco
- **Deploy:** Coolify/Hostinger, porta dedicada, subdomínio `www.fbtax.cloud`
- **Health check:** `GET /api/health` (padrão herdado)

---

## Project Scoping & Phased Development

### MVP Strategy & Philosophy

**MVP Approach:** Platform MVP — construir a fundação que unifica os produtos existentes e suporta expansão futura sem sobrecarga técnica
**Resource Requirements:** 1 desenvolvedor (Claudio), estimativa 3–4 semanas, stack conhecida (clone FB_SMARTPICK)

### MVP Feature Set (Phase 1)

**Core User Journeys Suportadas:**
- Ana / Ricardo / Carlos — acesso direto aos produtos via portal
- Felipe — vitrine pública + CTA comercial
- Claudio — painel admin com publicação de notificações e métricas

**Must-Have Capabilities:**

| Capacidade | Justificativa |
|---|---|
| Vitrine de 4 produtos com ícones | Jornadas Ana, Ricardo, Carlos, Felipe — core do portal |
| Diferenciação ativo/inativo por tenant via URL | Jornadas Ricardo e Carlos — identidade de plataforma |
| Painel de notificações público | Jornadas Ana e Carlos — canal de comunicação |
| CTA comercial + formulário + rastreamento | Jornada Felipe — geração passiva de leads |
| Painel admin (login + CRUD produtos + CRUD notificações + métricas) | Jornada Claudio — controle sem deploy |
| Deploy `www.fbtax.cloud` via Coolify | Infraestrutura base — tudo depende disso |
| SEO básico (meta tags + sitemap) | Jornada Felipe — chegada via Google |

### Post-MVP Features

**Phase 2 (Growth):**
- Área do cliente personalizada com status em tempo real dos sistemas
- Notificações por e-mail automáticas para lista de contatos do tenant
- Analytics de uso por produto e por tenant
- Página de roadmap público ("em breve")

**Phase 3 (Expansion):**
- Multi-tenant dashboard consolidado
- Integração SSO com `fortesbezerra.com.br`
- Portal como marketplace de módulos adicionais

### Risk Mitigation Strategy

**Risco técnico:** Stack 100% conhecida (clone FB_SMARTPICK) — risco mínimo. Único ponto de atenção é a lógica de identificação de tenant via parâmetro de URL (sem banco de usuários no portal público).

**Risco de mercado:** Portal depende de adoção pelos clientes ativos. Mitigação: comunicação direta no lançamento + URL simples de memorizar (`www.fbtax.cloud`).

**Risco de recurso:** Projeto solo. Contingência: reduzir painel admin ao essencial (publicar notificação + ver clique no CTA) e entregar vitrine primeiro.

---

## Functional Requirements

### 1. Vitrine de Produtos

- **FR1:** Visitante pode visualizar todos os produtos da plataforma fbtax.cloud com nome, ícone e descrição curta
- **FR2:** Visitante pode identificar quais produtos estão ativos (contratados) e quais estão desabilitados para o seu tenant
- **FR3:** Usuário pode acessar um produto ativo clicando no seu ícone (abre em nova aba no subdomínio correspondente)
- **FR4:** Visitante sem tenant identificado pode visualizar todos os produtos como disponíveis para contratação
- **FR5:** Sistema identifica o tenant do usuário via parâmetro de URL (`?tenant=`)

### 2. Notificações e Comunicados

- **FR6:** Visitante pode visualizar a lista de notificações publicadas (título, tipo, data, texto)
- **FR7:** Sistema registra a visualização de cada notificação por sessão (para fins de contagem)
- **FR8:** Notificações são exibidas em ordem cronológica decrescente (mais recentes primeiro)
- **FR9:** Notificações podem ser do tipo: Aviso de Sistema, Nova Versão, Comunicado Geral

### 3. Contato Comercial

- **FR10:** Visitante pode acessar formulário de contato comercial a partir do portal
- **FR11:** Visitante pode preencher e enviar formulário com nome, e-mail e mensagem
- **FR12:** Sistema envia o conteúdo do formulário por e-mail para o administrador
- **FR13:** Sistema registra cada clique no botão de contato comercial (rastreamento)
- **FR14:** Sistema registra cada envio de formulário concluído

### 4. Administração de Produtos

- **FR15:** Administrador pode autenticar-se no painel admin com login e senha
- **FR16:** Administrador pode criar novo produto com nome, descrição, ícone e URL de destino
- **FR17:** Administrador pode editar dados de um produto existente
- **FR18:** Administrador pode ativar ou desativar um produto no portfólio público
- **FR19:** Administrador pode definir quais produtos estão contratados por cada tenant

### 5. Administração de Notificações

- **FR20:** Administrador pode criar nova notificação com tipo, título e texto
- **FR21:** Administrador pode editar uma notificação existente
- **FR22:** Administrador pode publicar ou despublicar uma notificação
- **FR23:** Administrador pode excluir uma notificação

### 6. Métricas e Visibilidade

- **FR24:** Administrador pode visualizar o número de visualizações por notificação publicada
- **FR25:** Administrador pode visualizar o total de cliques no CTA comercial
- **FR26:** Administrador pode visualizar o total de formulários de contato enviados

### 7. Infraestrutura e Operação

- **FR27:** Sistema responde a health check em `GET /api/health`
- **FR28:** Portal público é acessível sem autenticação em `www.fbtax.cloud`
- **FR29:** Painel admin é acessível exclusivamente em rota protegida (`/admin`)
- **FR30:** Sistema serve meta tags de SEO (title, description, og:tags) para indexação

---

## Non-Functional Requirements

### Performance

- **NFR1:** A página principal (`/`) carrega com LCP < 2s em conexão padrão (Lighthouse score)
- **NFR2:** O bundle JavaScript do portal público é < 200kb gzipped
- **NFR3:** Respostas da API do painel admin completam em < 1s para operações de leitura
- **NFR4:** Ícones de produtos são servidos em formato SVG ou WebP otimizado

### Security

- **NFR5:** Autenticação do painel admin utiliza JWT com expiração configurável (padrão 8h)
- **NFR6:** Rota `/admin` retorna 401 para qualquer requisição sem token válido
- **NFR7:** Senha do admin é armazenada com hash bcrypt (padrão herdado do FB_SMARTPICK)
- **NFR8:** Todas as queries ao banco de dados utilizam prepared statements (sem concatenação de SQL)
- **NFR9:** Variáveis de ambiente sensíveis (DB_URL, SMTP, JWT_SECRET) nunca são hardcodadas
- **NFR10:** CORS configurado para aceitar apenas origens autorizadas (`www.fbtax.cloud`)

### Accessibility

- **NFR11:** Portal atende WCAG 2.1 nível AA — contraste mínimo 4.5:1 para texto normal
- **NFR12:** Todos os ícones de produto possuem atributo `aria-label` descritivo
- **NFR13:** Navegação por teclado funciona em todas as ações interativas do portal público

### Integration

- **NFR14:** Envio de e-mail via SMTP utiliza o serviço herdado do FB_SMARTPICK (`services/email.go`) sem modificação
- **NFR15:** Links para subdomínios externos abrem em nova aba (`target="_blank"` com `rel="noopener noreferrer"`)
- **NFR16:** Parâmetro `?tenant=` é lido e processado pelo frontend sem chamada de API (configuração estática)

### Reliability

- **NFR17:** Sistema mantém uptime ≥ 99% na janela 7h–22h em dias úteis
- **NFR18:** Health check `GET /api/health` responde em < 200ms
- **NFR19:** Deploy via Coolify com health check no Docker Compose — restart automático em falha
- **NFR20:** Banco de dados PostgreSQL com backup automático via Coolify/Hostinger
