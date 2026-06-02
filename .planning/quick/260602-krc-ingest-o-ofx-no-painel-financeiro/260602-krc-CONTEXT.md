# Quick Task 260602-krc: ingestão OFX no painel financeiro - Context

**Gathered:** 2026-06-02
**Status:** Ready for planning

<domain>
## Task Boundary

Adicionar caminho de ingestão por upload OFX no painel `/admin/financeiro/painel`, alimentando `financeiro.transacoes` para que o agente Z.AI (`chat_financeiro.go`) consiga responder consultas sobre as movimentações. O caminho paralelo à sync automática via Provider (que depende de OAuth2+mTLS do Banco Inter ainda pendente).

Escopo: backend Go (handler + parser OFX + dedup) + frontend (botão "Importar OFX" + dialog + toast de resumo) + integração com cadastro de contas existente em `financeiro.contas_financeiras`.

Fora de escopo: PDF/CSV ingestion (só OFX nesta task), categorização automática via IA na ingestão, edição em massa pós-import.

</domain>

<decisions>
## Implementation Decisions

### Vínculo de conta com financeiro.contas_financeiras

**Auto-detect com fallback manual.**

- Parser extrai `<BANKID>` e `<ACCTID>` do OFX.
- Backend faz match em `financeiro.contas_financeiras` por `(banco, agencia, conta)` derivado.
- **Match encontrado:** importa direto, vinculando à conta achada.
- **Match não encontrado:** retorna 409 com payload `{ detected: { bankid, acctid, branchid, instituicao } }` e o frontend abre dialog perguntando: (a) escolher uma das contas cadastradas no dropdown, (b) criar conta nova pré-preenchida com os dados detectados, ou (c) cancelar.
- Decisão de UX: nunca cria conta nova silenciosamente — sempre confirma com o admin.

### Dedup / re-upload

**FITID + hash fallback.**

- Cada transação OFX traz `<FITID>` que é id externo único — usa como `referencia_ext` na tabela `financeiro.transacoes`.
- **FITID ausente** (alguns bancos omitem): gera `referencia_ext` como hash SHA-256 de `(conta_id|data|valor|descricao_normalizada)`.
- Dedup via `SELECT EXISTS(...) WHERE referencia_ext = $1` antes de cada INSERT — mesma estratégia já usada em `bank_sync.go`.
- Resumo do upload retorna `{ importadas: N, duplicadas: M, erros: K, ids_inseridos: [...] }`.
- **Importante:** garantir que `financeiro.transacoes` tenha índice único em `referencia_ext` (se não tiver, criar migration). Verificar em `108_financeiro_schema.sql`.

### Flag conciliado

**conciliado = false por padrão.**

- Linhas inseridas via upload OFX entram com `conciliado = false`.
- Diferencia origem manual da sync API automática (que usa `true` em `bank_sync.go:54`).
- Força revisão pelo admin no painel — alinha com a expectativa de que upload pode ter ruído (períodos sobrepostos, OFX antigo, etc).
- Adicionar coluna `origem VARCHAR` em `financeiro.transacoes` se ainda não existir (`'api_sync' | 'ofx_upload' | 'manual'`) — verificar no schema atual; se já houver campo similar, reusar.

### Claude's Discretion

Itens não discutidos — usar bom senso no plano:

- **Categorização**: importar sem `categoria` (NULL). Z.AI categoriza on-demand via SQL queries. Não chamar IA na ingestão (latência, custo, escopo).
- **Tratamento de erro parcial**: best-effort por linha. Se 5 de 200 linhas falham parse, importa as 195 OK e retorna `erros: [{linha, motivo}]` no resumo. Não fazer all-or-nothing.
- **Limite de arquivo**: rejeitar OFX > 5 MB. Tipicamente um extrato mensal tem < 100 KB; 5 MB é folga generosa.
- **Encoding**: detectar cp1252 vs utf-8 vs latin-1 com fallback (mesma estratégia validada no MCP local `~/projetos/ofx-mcp/parsers.py`). Inter usa cp1252 declarado mas ASCII real; Itaú usa cp1252 com acentos.
- **Bancos não-mapeados**: parser não precisa conhecer o banco para parsear. Detecção de banco é só para mostrar nome amigável no resumo.
- **UI**: botão "Importar OFX" no card Contas do `PainelFinanceiroPage.tsx` (próximo ao botão "Nova conta"). Dialog com `<input type="file" accept=".ofx,.ofc">` + preview pós-parse antes de confirmar inserção.

</decisions>

<specifics>
## Specific Ideas

- **Lib Go**: usar `github.com/aclindsa/ofxgo` — parser OFX maduro, suporta OFX 1.x SGML e 2.x XML, já existe no ecossistema Go.
- **OFX real validado**: Banco Inter, FID 077, OFX 1.x SGML, charset cp1252 declarado mas conteúdo ASCII, 137 transações no extrato de 3 meses. Arquivo de exemplo em `/home/claudio/Documentos/Extratos/Extrato-04-03-2026-a-02-06-2026-OFX.ofx` para fixture de teste — usar valores anonimizados se virar fixture commitada.
- **Padrão de INSERT existente** (referência): `backend/services/bank_sync.go:54-65` faz exatamente o que precisamos — dedup por `referencia_ext`, INSERT em `financeiro.transacoes`. Reusar o pattern.
- **Padrão de handler multipart**: verificar se já existe upload multipart em outros handlers (`backend/handlers/`) antes de inventar do zero. `http.Request.ParseMultipartForm(5 << 20)` para limite 5 MB.
- **Pattern de dialog upload frontend**: reusar componentes shadcn já presentes — `Dialog`, `Input type=file`, `Button`, `Alert` com `react-hook-form` + `zod` se precisar validação.

</specifics>

<canonical_refs>
## Canonical References

- `backend/services/chat_financeiro.go` — schema que o agente Z.AI conhece (define as colunas que precisam estar populadas em `financeiro.transacoes`).
- `backend/services/bank_sync.go` — padrão de dedup + INSERT a ser replicado.
- `backend/handlers/painel.go`, `backend/handlers/bank_config.go` — handlers vizinhos (estilo de código, validação JWT, withAuth admin).
- `backend/migrations/108_financeiro_schema.sql` — schema de `financeiro.transacoes` e `financeiro.contas_financeiras` (confiar para descobrir colunas/índices existentes; migration nova só se faltar índice em `referencia_ext` ou coluna `origem`).
- `frontend/src/pages/PainelFinanceiroPage.tsx` — componente alvo da mudança no frontend.
- `~/projetos/ofx-mcp/parsers.py` — referência de quirks OFX já testados (BOM, encoding fallback, FID detection); não importar nada, só consultar o conhecimento.
- `CLAUDE.md` — constraints do projeto: Go stdlib (sem framework), schema `financeiro.*` fully-qualified, rotas `/api/financeiro/*` protegidas com `withAuth(..., "admin")`.

</canonical_refs>
