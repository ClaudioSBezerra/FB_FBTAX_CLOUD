---
stepsCompleted: [1, 2, 3, 4, 5, 6]
inputDocuments:
  - projetos/_bmad-output/planning-artifacts/product-brief-FB_SMARTPICK-2026-04-06.md
  - projetos/_bmad-output/planning-artifacts/project-context.md
  - projetos/_bmad-output/planning-artifacts/architecture.md (parcial)
date: 2026-04-08
author: Claudio
---

# Product Brief: FBTAX_CLOUD

<!-- Content will be appended sequentially through collaborative workflow steps -->

## Executive Summary

O FBTAX_CLOUD é o portal hub público da plataforma fbtax.cloud — ponto central de entrada para distribuidores brasileiros que utilizam os produtos da FBTax (Apuração Assistida, Simulador Fiscal, SmartPick, Farol e futuros módulos).

Para clientes ativos, o portal exibe os produtos contratados como ícones modernos com acesso direto a cada subdomínio (`apuracao.fbtax.cloud`, `smartpick.fbtax.cloud` etc.). Para visitantes que chegam via Google ou pelo site `fortesbezerra.com.br`, o portal é a vitrine institucional da plataforma — com CTA comercial discreto para produtos não contratados.

O portal não gerencia autenticação de usuário final (cada produto mantém seu próprio login). Um painel administrativo restrito ao proprietário (Claudio) permite publicar notificações, novidades e avisos do sistema sem necessidade de deploy.

---

## Core Vision

### Problem Statement

Clientes que contratam mais de um produto fbtax.cloud não têm um ponto central de navegação — cada sistema possui URL e tela de login independentes, criando uma experiência fragmentada sem identidade unificada de plataforma.

Paralelamente, visitantes que chegam via Google ou pelo site institucional `fortesbezerra.com.br` não encontram uma vitrine profissional dos produtos — a percepção de "plataforma" não se concretiza visualmente.

### Problem Impact

- Clientes com múltiplos produtos precisam lembrar e acessar cada URL individualmente
- Sem ponto central, a percepção de "plataforma integrada" não existe para o cliente
- Novos potenciais clientes chegam sem uma vitrine clara dos produtos disponíveis
- A equipe FBTax não tem canal centralizado para comunicar novidades e avisos a todos os clientes

### Why Existing Solutions Fall Short

Cada produto fbtax.cloud foi desenvolvido de forma independente — repositório, deploy e login próprios. Não existe hoje nenhum ponto de agregação visual ou comercial entre eles. O site `fortesbezerra.com.br` é institucional mas não é o hub dos produtos SaaS em si.

### Proposed Solution

Um portal web público em `www.fbtax.cloud` com:

- **Vitrine de produtos:** ícones modernos dos sistemas fbtax.cloud, cada um linkando para seu subdomínio
- **Diferenciação por status:** produtos contratados pelo tenant ativos e clicáveis; não contratados visíveis com CTA comercial discreto ("Saiba mais / Falar com comercial")
- **Notificações e novidades:** painel de comunicados gerenciado pelo admin (Claudio) — avisos de sistema, novas versões, comunicados ao cliente
- **Painel admin protegido:** acesso exclusivo via login/senha para gerenciar conteúdo sem deploy
- **Porta de entrada comercial:** visitantes externos (Google / fortesbezerra.com.br) encontram a plataforma completa com CTA para contato

### Key Differentiators

- **Identidade de plataforma:** pela primeira vez, os produtos fbtax.cloud têm uma "casa" visual unificada em `www.fbtax.cloud`
- **Zero atrito para o cliente:** um bookmark, todos os produtos — sem gerenciar múltiplas URLs
- **Canal de comunicação próprio:** notificações e novidades publicadas sem dependência de e-mail ou WhatsApp
- **Vitrine passiva de vendas:** produtos não contratados geram interesse comercial sem esforço ativo
- **Leveza arquitetural:** portal independente, sem banco de dados de usuário final — manutenção mínima

---

## Target Users

### Primary Users

---

#### Persona 1 — Ana, Contadora / Profissional Fiscal

**Perfil:**
Contadora responsável pela apuração fiscal de um ou mais distribuidores clientes. Acessa `apuracao.fbtax.cloud` e `simulador.fbtax.cloud` regularmente. Trabalha em escritório de contabilidade ou no departamento fiscal da própria empresa. Acessa pelo notebook, em horário comercial.

**Como usa o portal:**
- Acessa `www.fbtax.cloud` como ponto de partida do dia
- Navega diretamente para o produto que precisa (Apuração ou Simulador)
- Consulta avisos do sistema antes de processar apurações críticas
- Vê novidades da plataforma (ex: atualizações da reforma tributária)

**Momento de valor:**
Quando percebe que todas as ferramentas fiscais que usa estão num único lugar, atualizadas, com avisos sobre mudanças regulatórias relevantes.

**Sucesso:**
Zero tempo perdido procurando URLs. Informada sobre atualizações antes que impactem o trabalho.

---

#### Persona 2 — Ricardo, Profissional de Controladoria

**Perfil:**
Analista ou gerente de controladoria de um distribuidor de médio/grande porte. Usa múltiplos produtos fbtax.cloud — Apuração para fechamento fiscal e SmartPick para acompanhar eficiência logística do CD. Apresenta resultados consolidados para a diretoria.

**Como usa o portal:**
- Usa o portal como dashboard de acesso rápido entre produtos
- Valoriza a visão de "quais produtos minha empresa contratou"
- Lê notificações de novas funcionalidades para avaliar expansão da licença
- Eventualmente encaminha o link do portal para colegas que precisam de acesso a um produto específico

**Momento de valor:**
Quando consegue alternar entre Apuração e SmartPick sem precisar lembrar URLs separadas — tudo numa tela limpa e profissional.

**Sucesso:**
Plataforma percebida como solução integrada, não como produtos avulsos.

---

#### Persona 3 — Carlos, Gestor de CD (usuário ocasional do portal)

**Perfil:**
Gestor logístico que usa o SmartPick no dia a dia. Acessa o portal esporadicamente — principalmente quando vê uma notificação de nova versão ou é orientado pelo Ricardo/CEO a verificar algo. Não é o público principal do portal, mas passa por ele.

**Como usa o portal:**
- Clica no link do SmartPick a partir do portal
- Lê avisos de manutenção ou atualizações que afetam o sistema que usa
- Pode ver outros produtos disponíveis mas raramente interage com eles

---

#### Persona 4 — Felipe, Profissional de TI

**Perfil:**
Analista de TI responsável pela infraestrutura e ferramentas digitais do distribuidor. Avalia novas soluções, coordena onboarding de sistemas e é o primeiro ponto de contato técnico para questões de acesso e integração.

**Como usa o portal:**
- Chega via Google pesquisando solução fiscal/logística SaaS para distribuidores
- Avalia a completude da plataforma: quais produtos existem, como funcionam, como são integrados
- Preenche o formulário de contato comercial após avaliar o portfólio
- Após contratação, é responsável por comunicar as URLs corretas para os usuários da empresa

**Momento de valor:**
Quando encontra no portal uma visão clara de toda a plataforma — produtos, descrições e contato — sem precisar ligar ou mandar e-mail para entender o que a FBTax oferece.

---

### Secondary Users

#### Persona 5 — Claudio, Admin da Plataforma (proprietário FBTax)

**Perfil:**
Proprietário da FBTax, único com acesso ao painel de configurações do portal. Gerencia notificações, avisos de sistema, novidades de produto e o portfólio de produtos visíveis no portal. Não precisa de deploy para atualizar o conteúdo.

**Responsabilidades:**
- Publicar avisos de manutenção programada
- Anunciar novas versões de produtos
- Atualizar portfólio quando novos produtos forem lançados
- Monitorar acessos ao CTA comercial

---

### User Journey

#### Jornada — Cliente Ativo (Ana / Ricardo)

| Etapa | Ação | Resultado esperado |
|---|---|---|
| **Acesso** | Abre `www.fbtax.cloud` | Vê ícones dos produtos contratados, ativos e clicáveis |
| **Navegação** | Clica no ícone do produto desejado | Redirecionado para o subdomínio (`apuracao.fbtax.cloud`) |
| **Informação** | Lê painel de notificações | Vê avisos relevantes antes de trabalhar |
| **Descoberta** | Vê produto não contratado com CTA | Considera expansão ou encaminha para decisor |

#### Jornada — Visitante Externo (Felipe / TI)

| Etapa | Ação | Resultado esperado |
|---|---|---|
| **Chegada** | Google → `www.fbtax.cloud` | Vitrine profissional com todos os produtos |
| **Avaliação** | Navega pelos produtos disponíveis | Entende o portfólio completo da plataforma |
| **Interesse** | Clica em CTA comercial discreto | Formulário ou link de contato (`fortesbezerra.com.br`) |
| **Follow-up** | Recebe contato da equipe FBTax | Início do processo comercial |

---

## Success Metrics

### Sucesso do Usuário

**Cliente ativo (Ana / Ricardo / Carlos):**
- Usa `www.fbtax.cloud` como ponto de entrada para os produtos — não precisa memorizar URLs individuais
- Acessa o produto desejado em ≤ 2 cliques a partir do portal
- Lê notificações relevantes antes de usar o sistema (ex: aviso de manutenção, nova versão)
- Percebe o fbtax.cloud como plataforma integrada, não como produtos avulsos

**Visitante externo (Felipe / TI):**
- Encontra o portfólio completo de produtos em uma única página
- Clica no CTA comercial após avaliar os produtos disponíveis

---

### Business Objectives

**Curto prazo (0–3 meses):**
- Portal publicado e acessível em `www.fbtax.cloud`
- Todos os clientes ativos usando o portal como ponto de entrada
- Painel admin funcional — Claudio publica notificações sem deploy

**Médio prazo (3–12 meses):**
- Portal consolidado como ponto focal da plataforma fbtax.cloud
- Canal de comunicação ativo com clientes — novidades e avisos publicados regularmente
- Geração passiva de leads qualificados via CTA comercial

**Longo prazo (12+ meses):**
- Cada novo produto lançado aparece automaticamente no portal
- Portal reconhecido por prospectos como vitrine profissional da plataforma

---

### Key Performance Indicators

| KPI | Métrica | Alvo | Prazo |
|---|---|---|---|
| **Adoção pelos clientes** | % clientes ativos que acessam o portal mensalmente | ≥ 80% | 3 meses |
| **Engajamento com notificações** | % de acessos onde o usuário visualiza pelo menos 1 notificação | ≥ 60% | Contínuo |
| **Leitura de notificações** | Contador de visualizações por notificação publicada | Visibilidade total | Contínuo |
| **CTA comercial** | Contador de cliques no botão de contato comercial | Rastreado | Contínuo |
| **Leads via formulário** | Número de formulários de contato preenchidos | Rastreado | Contínuo |
| **Disponibilidade** | Uptime do portal | ≥ 99% | Contínuo |
| **Velocidade** | Tempo de carregamento da página principal | < 2s | Lançamento |

---

## MVP Scope

### Core Features

**1. Vitrine de produtos**
- Página principal pública em `www.fbtax.cloud`
- Ícones modernos de cada produto com nome e descrição curta
- Produtos ativos: Apuração Assistida, Simulador Fiscal, SmartPick, Farol
- Cada ícone linka para o subdomínio correspondente (nova aba)
- Design herdado do sistema visual FB_SMARTPICK (React + Tailwind + Shadcn)

**2. Diferenciação por status de contratação**
- Produtos contratados pelo tenant: ícone ativo e clicável
- Produtos não contratados: ícone visível porém desabilitado (visual cinza/opaco) com CTA discreto "Saiba mais"
- Identificação do tenant: via parâmetro de URL ou configuração simples — sem login de usuário final no portal

**3. Painel de notificações e novidades**
- Lista de comunicados publicados pelo admin (título + texto + data)
- Tipos: aviso de sistema, nova versão, comunicado geral
- Contador de visualizações por notificação (para rastrear leituras)
- Exibido na página principal — visível a todos os visitantes

**4. CTA comercial**
- Botão "Fale Conosco" discreto em posição fixa (rodapé ou canto)
- Formulário simples: nome, e-mail, mensagem
- Rastreamento de cliques no botão (contador)
- Envio de formulário por e-mail para Claudio

**5. Painel admin protegido**
- Acesso via login/senha exclusivo para Claudio
- CRUD de produtos: adicionar, editar, ativar/desativar produto no portfólio
- CRUD de notificações: criar, editar, publicar/despublicar comunicados
- Visualização de métricas: cliques no CTA, visualizações por notificação

---

### Out of Scope para MVP

| Feature | Motivo | Versão |
|---|---|---|
| Login de usuário final no portal | Auth fica em cada produto | Fora do escopo |
| SSO entre produtos | Complexidade desnecessária | V2+ |
| App mobile nativo | Web responsivo suficiente | Não planejado |
| Dashboard analytics avançado | Google Analytics cobre o básico | V2 |
| Chat/suporte em tempo real | Formulário de contato suficiente | V2 |
| Blog ou conteúdo editorial | Fora do foco do portal | V2 |
| Multi-idioma | Base de clientes BR | Não planejado |
| Notificações por e-mail automáticas | Painel visual suficiente no MVP | V2 |

---

### MVP Success Criteria

- ✅ Portal publicado e acessível em `www.fbtax.cloud`
- ✅ Todos os 4 produtos visíveis com links funcionais para subdomínios
- ✅ Admin publica notificação sem necessidade de deploy
- ✅ Formulário de contato envia e-mail e registra clique
- ✅ Página carrega em < 2s
- ✅ 100% dos clientes ativos informados do novo portal no lançamento

---

### Future Vision (V2+)

- **Área do cliente personalizada:** após identificação do tenant, exibe apenas produtos contratados em destaque e painel de status dos sistemas
- **Notificações por e-mail:** comunicados enviados automaticamente para lista de contatos do tenant
- **Analytics de uso:** quais produtos são mais acessados por tenant
- **Portal de novos produtos:** página de roadmap pública com produtos "em breve"
- **Integração com `fortesbezerra.com.br`:** SSO institucional entre o site e o portal
- **Multi-tenant dashboard:** visão consolidada de status de todos os produtos por cliente
