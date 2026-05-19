# Phase 1: Fundação do Módulo - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured em CONTEXT.md — este log preserva as alternativas consideradas.

**Date:** 2026-05-19
**Phase:** 1-Fundação do Módulo
**Areas discussed:** Namespace do schema, Escopo das migrations, Ponto de entrada da UI, Role do admin financeiro

---

## Namespace do Schema

| Option | Description | Selected |
|--------|-------------|----------|
| `financeiro.*` separado | Schema PostgreSQL separado, seguindo precedente do `portal.*`. Queries com nomes fully-qualified. | ✓ |
| `public.*` com prefixo `fin_` | Tabelas no schema public com prefixo, sem gerenciar schema separado. | |

**User's choice:** `financeiro.*` schema separado
**Notes:** Usuário aprovou o preview da migration com `CREATE SCHEMA IF NOT EXISTS financeiro`.

---

## Escopo das Migrations

| Option | Description | Selected |
|--------|-------------|----------|
| Apenas o que a fase usa | Migrations incrementais — fase 1 cria só `empresas` + `dados_bancarios`. Fases futuras criam suas tabelas. | ✓ |
| Esqueleto completo agora | Fase 1 cria todas as tabelas do módulo (clientes, produtos, contratos, tokens). Fases seguintes só adicionam lógica. | |

**User's choice:** Apenas o que a fase usa
**Notes:** Preferência por incrementalidade — cada fase é responsável por suas próprias tabelas.

| Option | Description | Selected |
|--------|-------------|----------|
| Tabelas separadas (`empresas` + `dados_bancarios`) | Separação clara — facilita múltiplas contas bancárias no v2. | ✓ |
| Uma única tabela | Todos os dados em `financeiro.empresas` com colunas de dados bancários. | |

**User's choice:** Tabelas separadas
**Notes:** Separação de tabelas reflete separação de domínio — dados cadastrais vs dados financeiros.

---

## Ponto de Entrada da UI

| Option | Description | Selected |
|--------|-------------|----------|
| Rota simples `/admin/financeiro/empresa` | Página standalone com `ProtectedRoute`, sem layout de painel. O painel vem na fase 5. | ✓ |
| Layout mínimo do painel já na fase 1 | Sidebar + main layout para `/admin/financeiro/*` que a fase 5 expande. | |

**User's choice:** Rota simples `/admin/financeiro/empresa`
**Notes:** Preferência por mínimo viável na fase 1 — o layout do painel é responsabilidade da fase 5.

| Option | Description | Selected |
|--------|-------------|----------|
| Dois formulários separados | Seção "Dados da Empresa" + seção "Dados Bancários" com botões Salvar independentes. | ✓ |
| Uma página, duas seções | Um único botão Salvar para ambos. | |

**User's choice:** Dois formulários separados
**Notes:** Admin pode editar dados cadastrais sem afetar dados bancários e vice-versa.

---

## Role do Admin Financeiro

| Option | Description | Selected |
|--------|-------------|----------|
| Mesma pessoa, mesmo login | Admin FBTax = Admin financeiro. Sem separação de usuários no v1. | ✓ |
| Usuário distinto | Equipe FB com login próprio sem acesso ao FBTax Cloud. | |

**User's choice:** Mesma pessoa, mesmo login
**Notes:** No v1 há um único admin — o Cláudio. Separação de identidades é questão do v2+.

| Option | Description | Selected |
|--------|-------------|----------|
| Reusar `admin` existente | `withAuth(handler, "admin")`. Sem nova migration de role. | ✓ |
| Criar `fb_admin` desde agora | Nova role — mais granular, mas sem benefício real no v1. | |

**User's choice:** Reusar `admin` existente
**Notes:** `fb_admin` pode ser avaliado na fase 5 se a separação de painel exigir.

---

## Claude's Discretion

- Nomes exatos das colunas de endereço (`logradouro`, `numero`, `bairro`, `cep`, `municipio`, `uf`)
- Nome do arquivo handler no backend (`financeiro.go` ou `empresa.go`)
- Granularidade dos arquivos de migration (1 arquivo por tabela ou agrupados)
- Estrutura interna dos componentes React

## Deferred Ideas

- **Role `fb_admin`**: Criar role separada para o painel do módulo financeiro — avaliar na fase 5
- **Layout do painel financeiro**: Sidebar + nav — fase 5 (Painel Admin)
- **Múltiplas contas bancárias**: Modelo de dados já suporta (tabela separada), mas UI é v2+
