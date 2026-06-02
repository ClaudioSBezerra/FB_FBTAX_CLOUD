---
phase: quick-260602-krc
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - backend/migrations/123_ofx_origem.sql
  - backend/services/ofx_ingest.go
  - backend/handlers/ofx_upload.go
  - backend/main.go
  - frontend/src/pages/PainelFinanceiroPage.tsx
autonomous: true
requirements: [OFX-INGEST]
must_haves:
  truths:
    - "Admin pode clicar em 'Importar OFX' no card Contas e fazer upload de um arquivo .ofx"
    - "Backend detecta a conta por BANKID+ACCTID; se não encontrar retorna 409 com payload detectado"
    - "Frontend abre dialog de fallback ao receber 409, permitindo escolher conta existente ou cancelar"
    - "Transações são inseridas em financeiro.transacoes com conciliado=false e origem='ofx_upload'"
    - "Dedup funciona: re-upload do mesmo OFX não duplica registros"
    - "Resposta final informa importadas/duplicadas/erros; Z.AI pode consultá-las via SQL"
  artifacts:
    - path: "backend/migrations/123_ofx_origem.sql"
      provides: "Coluna origem em financeiro.transacoes e índice único em referencia_ext"
      contains: "ALTER TABLE financeiro.transacoes"
    - path: "backend/services/ofx_ingest.go"
      provides: "Parser OFX + lógica de dedup + INSERT best-effort"
      exports: ["IngestOFX", "OFXIngestResult", "OFXDetectedAccount"]
    - path: "backend/handlers/ofx_upload.go"
      provides: "POST /api/financeiro/ofx/upload multipart handler"
      exports: ["OFXUploadHandler"]
    - path: "frontend/src/pages/PainelFinanceiroPage.tsx"
      provides: "Botão Importar OFX + dialog 409 + toast de resumo"
  key_links:
    - from: "backend/handlers/ofx_upload.go"
      to: "backend/services/ofx_ingest.go"
      via: "services.IngestOFX(db, contaID, reader)"
    - from: "frontend/src/pages/PainelFinanceiroPage.tsx"
      to: "/api/financeiro/ofx/upload"
      via: "fetch POST multipart/form-data"
    - from: "backend/services/ofx_ingest.go"
      to: "financeiro.transacoes"
      via: "INSERT com referencia_ext, origem, conciliado=false"
---

<objective>
Implementar ingestão de extratos OFX no painel financeiro: migration de schema, serviço de parse/dedup, handler HTTP multipart, registro de rota e UI com dialog de fallback.

Purpose: Permitir que o admin alimente financeiro.transacoes via upload OFX enquanto a integração OAuth2 do Banco Inter está pendente, habilitando o agente Z.AI a responder consultas sobre movimentações.

Output: Rota POST /api/financeiro/ofx/upload funcional + botão "Importar OFX" no card Contas do painel.
</objective>

<execution_context>
@$HOME/.claude/get-shit-done/workflows/execute-plan.md
@$HOME/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/quick/260602-krc-ingest-o-ofx-no-painel-financeiro/260602-krc-CONTEXT.md
@backend/migrations/118_financeiro_painel.sql
@backend/services/bank_sync.go
@backend/handlers/bank_config.go
@backend/main.go
@frontend/src/pages/PainelFinanceiroPage.tsx

<interfaces>
<!-- Contratos existentes que o executor deve usar diretamente. -->

De backend/migrations/118_financeiro_painel.sql:
- financeiro.transacoes: id UUID PK, conta_id UUID FK, data_transacao DATE,
  descricao VARCHAR(500), valor NUMERIC(15,2), tipo VARCHAR(10) CHECK('credito'|'debito'),
  categoria VARCHAR(100), referencia_ext VARCHAR(255), conciliado BOOLEAN DEFAULT false,
  created_at TIMESTAMPTZ
- financeiro.contas_financeiras: id UUID PK, apelido, banco VARCHAR(100),
  agencia VARCHAR(20), conta VARCHAR(30), tipo, provedor, provedor_id, saldo, ultima_sync, ativa

De backend/services/bank_sync.go (padrão de dedup a replicar, linhas 51-67):
  SELECT EXISTS(SELECT 1 FROM financeiro.transacoes WHERE referencia_ext = $1)
  INSERT INTO financeiro.transacoes
      (conta_id, data_transacao, descricao, valor, tipo, categoria, referencia_ext, conciliado)
  VALUES ($1, $2, $3, $4, $5, $6, $7, true)   -- OFX usa false + adiciona origem

De backend/handlers/bank_config.go (padrão de handler):
  func XHandler(db *sql.DB) http.HandlerFunc { return func(w http.ResponseWriter, r *http.Request) {...} }
  Retornos: http.Error(w, "msg", statusCode) ou json.NewEncoder(w).Encode(payload)

De backend/main.go (linhas 381-384 — padrão de registro de rota):
  http.HandleFunc("/api/financeiro/bancos/sync", withAuth(handlers.BancosSyncHandler, "admin"))
  -- Novo registro vai logo após linha 384 no bloco financeiro-admin
</interfaces>
</context>

<tasks>

<task type="auto">
  <name>Task 1: Migration — coluna origem + índice único em referencia_ext</name>
  <files>backend/migrations/123_ofx_origem.sql</files>
  <action>
Criar migration SQL `backend/migrations/123_ofx_origem.sql` com dois ALTER:

1. Adicionar coluna `origem VARCHAR(20) DEFAULT 'api_sync'` em `financeiro.transacoes` (IF NOT EXISTS via DO block ou COLUMN check — usar pattern de migration idempotente com `DO $$ BEGIN IF NOT EXISTS (...) THEN ALTER TABLE ... END IF; END $$`).

   Valores válidos: 'api_sync' | 'ofx_upload' | 'manual'. Não adicionar CHECK constraint agora — evitar lock e manter flexibilidade.

2. Criar índice único em `referencia_ext` com UNIQUE e WHERE referencia_ext IS NOT NULL (índice parcial — evita conflito nos NULLs de transações sem ID externo):
   `CREATE UNIQUE INDEX IF NOT EXISTS idx_transacoes_referencia_ext_uq ON financeiro.transacoes(referencia_ext) WHERE referencia_ext IS NOT NULL;`

O índice parcial é correto aqui: múltiplas transações podem ter referencia_ext NULL (manuais sem dedup), mas dois registros com o mesmo referencia_ext não-nulo são sempre duplicata.
  </action>
  <verify>
    <automated>cd /home/claudio/projetos/FB_FBTAX_CLOUD && grep -c "origem" backend/migrations/123_ofx_origem.sql && grep -c "idx_transacoes_referencia_ext_uq" backend/migrations/123_ofx_origem.sql</automated>
  </verify>
  <done>Arquivo existe com ALTER TABLE adicionando coluna origem e CREATE UNIQUE INDEX IF NOT EXISTS parcial em referencia_ext.</done>
</task>

<task type="auto" tdd="true">
  <name>Task 2: Serviço ofx_ingest.go — parse OFX + dedup + INSERT</name>
  <files>backend/services/ofx_ingest.go, backend/go.mod, backend/go.sum</files>
  <behavior>
    - IngestOFX com conta não encontrada retorna erro "conta não encontrada"
    - FITID presente vira referencia_ext diretamente
    - FITID ausente gera SHA-256 de "contaID|data|valor|descricaoNorm"
    - Transação já existente (referencia_ext duplicada) incrementa Duplicadas, não insere
    - Transação com erro de parse (campo obrigatório vazio) incrementa Erros com motivo, não interrompe loop
    - Todas as inserções têm conciliado=false e origem='ofx_upload'
    - OFX com BANKID+ACCTID sem match na tabela retorna ErrContaNaoDetectada com OFXDetectedAccount preenchido
  </behavior>
  <action>
Adicionar `github.com/aclindsa/ofxgo` ao go.mod via `go get github.com/aclindsa/ofxgo` dentro de `backend/`. Esta lib suporta OFX 1.x SGML e 2.x XML e resolve encoding automaticamente.

Criar `backend/services/ofx_ingest.go` no package `services`:

Tipos exportados:
  - `OFXDetectedAccount` struct com campos BankID, AcctID, BranchID string
  - `OFXIngestResult` struct com Importadas, Duplicadas, Erros int, IdsInseridos []string, DetalhesErros []OFXErroDetalhe, ContaApelido string
  - `OFXErroDetalhe` struct com Linha int, FITID, Motivo string
  - `ErrContaNaoDetectada` como tipo de erro customizado com campo Detected OFXDetectedAccount

Função principal: `IngestOFX(db *sql.DB, contaID string, r io.Reader) (*OFXIngestResult, error)`

Fluxo interno:
1. Ler r integralmente em []byte (já limitado a 5 MB no handler — não re-limitar aqui).
2. Tentar `ofxgo.ParseResponse(bytes.NewReader(data))`. Se falhar, tentar recodificar de cp1252→utf-8 com `golang.org/x/text/encoding/charmap` (já no go.sum via dependência indireta de maroto). Se ainda falhar, retornar erro de parse.
3. Iterar sobre `resp.Bank[i].(*ofxgo.StatementResponse).BankTranList.Transactions`.
4. Se `contaID == ""` (auto-detect): extrair BANKID e ACCTID do OFX. Fazer SELECT em `financeiro.contas_financeiras` por `(banco ILIKE $1 OR provedor_id = $1) AND conta = $2` onde $1=BANKID e $2=ACCTID. Se não encontrar: retornar `&ErrContaNaoDetectada{Detected: OFXDetectedAccount{...}}`. Se encontrar: usar o id encontrado como contaID.
5. Para cada transação:
   a. Normalizar descricao: strings.TrimSpace do campo MEMO ou NAME.
   b. Determinar referencia_ext: se FITID não vazio → usar FITID; senão → SHA-256 de fmt.Sprintf("%s|%s|%s|%s", contaID, data, valor, descricaoNorm).
   c. Determinar tipo: se TRNTYPE == "CREDIT" → "credito"; senão → "debito".
   d. Converter DTPOSTED para DATE (formato OFX: "20260402000000[-3:BRT]" → time.Parse com layout "20060102150405").
   e. Checar EXISTS por referencia_ext no DB. Se existir: result.Duplicadas++, continue.
   f. INSERT com: conta_id, data_transacao, descricao, valor ABS, tipo, categoria=NULL, referencia_ext, conciliado=false, origem='ofx_upload'.
   g. Se INSERT falhar: append em DetalhesErros, Erros++, continue (best-effort).
   h. Caso contrário: Importadas++, append id em IdsInseridos.
6. Retornar result, nil.

Nota sobre encoding: `golang.org/x/text` já está no go.sum como dependência indireta. Importar `golang.org/x/text/encoding/charmap` e `golang.org/x/text/transform` para a tentativa cp1252→utf-8.

NUNCA usar fenced code blocks dentro desta action — nomes de funções e campos são suficientes.
  </action>
  <verify>
    <automated>cd /home/claudio/projetos/FB_FBTAX_CLOUD/backend && go build ./services/... 2>&1</automated>
  </verify>
  <done>go build ./services/... passa sem erros. Tipos IngestOFX, OFXIngestResult, OFXDetectedAccount, ErrContaNaoDetectada exportados e compilando.</done>
</task>

<task type="auto">
  <name>Task 3: Handler ofx_upload.go + rota + UI dialog</name>
  <files>backend/handlers/ofx_upload.go, backend/main.go, frontend/src/pages/PainelFinanceiroPage.tsx</files>
  <action>
--- BACKEND ---

Criar `backend/handlers/ofx_upload.go` no package `handlers`.

Handler `OFXUploadHandler(db *sql.DB) http.HandlerFunc`:
- Aceita apenas POST; rejeita outros métodos com 405.
- `r.ParseMultipartForm(5 << 20)` para limitar a 5 MB. Erro neste passo → 400 "arquivo muito grande ou multipart inválido".
- Ler campo `file` do form: `r.FormFile("file")`. Erro → 400 "campo 'file' obrigatório".
- Ler `conta_id` de `r.FormValue("conta_id")` (pode vir vazio — auto-detect mode).
- Chamar `services.IngestOFX(db, contaID, file)`.
- Se err é `*services.ErrContaNaoDetectada`:
    w.WriteHeader(409)
    json.NewEncoder(w).Encode(map[string]interface{}{"detected": err.Detected})
    return
- Se outro erro: 500 com mensagem de erro.
- Caso sucesso: 200 com json do OFXIngestResult.
- Header Content-Type: application/json em todos os caminhos.

Registrar rota em `backend/main.go` logo após a linha do `bancos/providers`:
  http.HandleFunc("/api/financeiro/ofx/upload", withAuth(handlers.OFXUploadHandler, "admin"))

--- FRONTEND ---

Em `frontend/src/pages/PainelFinanceiroPage.tsx`:

1. Adicionar interfaces locais:
   - `OFXImportResult` com campos: importadas, duplicadas, erros, ids_inseridos (string[]), conta_apelido (string)
   - `OFXDetected` com campos: bankid, acctid, branchid (string)

2. Adicionar estados (junto aos outros useState no topo do componente principal, após os existentes):
   - `ofxDialogOpen: boolean` — controla abertura do dialog de upload
   - `ofx409Open: boolean` — controla dialog de fallback 409
   - `ofxDetected: OFXDetected | null` — payload recebido no 409
   - `ofxFile: File | null` — arquivo retido para retry após escolha de conta
   - `ofxContaSelecionada: string` — conta escolhida no fallback
   - `ofxImporting: boolean` — estado de loading

3. Função `handleOFXUpload(file: File, contaId?: string)`:
   - Cria FormData com `file` e opcionalmente `conta_id`.
   - Faz fetch POST para `/api/financeiro/ofx/upload`.
   - Se res.status === 409: parse do JSON, setar ofxDetected + ofxFile, abrir ofx409Open, fechar ofxDialogOpen.
   - Se res.ok: parse do OFXImportResult, fechar ofxDialogOpen, toast.success com mensagem "N transações importadas, M duplicadas". Se erros > 0: toast.warning "K linhas ignoradas". Chamar carregarTransacoes() para atualizar a lista.
   - Se outro erro: toast.error com mensagem do servidor.

4. No card Contas (após o Button "Nova conta", linha ~541 do original):
   - Adicionar Button variant="ghost" size="sm" com ícone Upload (lucide-react) e texto "Importar OFX", onClick={() => setOfxDialogOpen(true)}.

5. Dialog principal de upload (ofxDialogOpen):
   - DialogContent com DialogHeader "Importar extrato OFX".
   - Input type="file" accept=".ofx,.ofc" onChange que chama handleOFXUpload(file) imediatamente ao selecionar (sem preview — simplifica UX).
   - Enquanto ofxImporting: mostrar Loader2 girando e texto "Processando...".
   - Botão Cancelar que fecha o dialog e limpa estados.

6. Dialog de fallback 409 (ofx409Open):
   - DialogContent com DialogHeader "Conta não identificada automaticamente".
   - Exibir os dados detectados (ofxDetected.bankid, ofxDetected.acctid) em texto informativo.
   - Select (usando elemento nativo html select com className Tailwind ou Select do Radix se já importado) listando `contas` do estado existente — mostra apelido, value=id.
   - Botão "Importar para esta conta" que chama handleOFXUpload(ofxFile!, ofxContaSelecionada) e fecha o dialog.
   - Botão "Cancelar" que limpa e fecha.

Importar Upload de lucide-react (adicionar ao import existente da linha ~11).
  </action>
  <verify>
    <automated>cd /home/claudio/projetos/FB_FBTAX_CLOUD/backend && go build ./... 2>&1</automated>
  </verify>
  <done>go build ./... passa. Handler OFXUploadHandler registrado em main.go. PainelFinanceiroPage.tsx compilável com botão "Importar OFX" e os dois dialogs.</done>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| browser -> POST /api/financeiro/ofx/upload | Arquivo enviado por admin autenticado — conteúdo não confiável |
| services.IngestOFX -> financeiro.transacoes | Dados parseados do OFX injetados via query parametrizada |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-ofx-01 | Tampering | OFX file upload | mitigate | ParseMultipartForm(5<<20) limita a 5 MB; io.LimitReader não necessário pois ParseMultipartForm já trunca |
| T-ofx-02 | Information Disclosure | ofxgo parsing | mitigate | Erros de parse retornam mensagem genérica ao cliente; detalhe vai para log interno |
| T-ofx-03 | Denial of Service | arquivo com 100k transações | accept | 5 MB cap limita a ~50k linhas OFX; processamento síncrono aceitável para extrato mensal (<500 tx) |
| T-ofx-04 | Injection | referencia_ext, descricao | mitigate | Queries parametrizadas ($1, $2, ...) em todos os INSERTs/SELECTs — sem concatenação de string SQL |
| T-ofx-05 | Elevation of Privilege | rota /api/financeiro/ofx/upload | mitigate | withAuth(..., "admin") — mesmo padrão de todas as rotas financeiro-admin |
| T-ofx-SC | Tampering | go get github.com/aclindsa/ofxgo | mitigate | Verificar no pkg.go.dev antes de adicionar — pacote maduro com histórico; confirmar checksum em go.sum |
</threat_model>

<verification>
Sequência de verificação manual pós-execução:

1. `cd backend && go build ./...` — zero erros de compilação
2. Subir dev server (`go run main.go`) e confirmar no log de startup que migration 123 foi aplicada
3. `curl -X POST http://localhost:8086/api/financeiro/ofx/upload` sem auth → 401
4. Fazer upload de `/home/claudio/Documentos/Extratos/Extrato-04-03-2026-a-02-06-2026-OFX.ofx` autenticado:
   - Primeira vez: verificar resposta 200 ou 409 (se conta não mapeada)
   - Segunda vez com mesma conta: importadas=0, duplicadas=N (zero reinserções)
5. Query de confirmação: `SELECT origem, conciliado, COUNT(*) FROM financeiro.transacoes GROUP BY 1,2` — linhas OFX aparecem com origem='ofx_upload', conciliado=false
6. Frontend: botão "Importar OFX" visível no card Contas; dialog abre e fecha sem erros de console
</verification>

<success_criteria>
- Migration 123 aplicada: coluna `origem` existe e índice único parcial em `referencia_ext` criado
- POST /api/financeiro/ofx/upload aceita multipart, retorna JSON com importadas/duplicadas/erros
- Re-upload do mesmo OFX retorna importadas=0, duplicadas=N (dedup funciona)
- Conta não detectada retorna 409 com `{ detected: { bankid, acctid } }`
- Frontend exibe botão, dialog de upload e dialog 409 de fallback
- go build ./... sem erros
</success_criteria>

<output>
Criar `.planning/quick/260602-krc-ingest-o-ofx-no-painel-financeiro/260602-krc-SUMMARY.md` ao concluir.
</output>
