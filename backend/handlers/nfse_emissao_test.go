package handlers

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"database/sql"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math/big"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"software.sslmate.com/src/go-pkcs12"
)

func nfseTestDB(t *testing.T) *sql.DB {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL não definido (use dev/test-nfse-db.sh)")
	}
	db, err := sql.Open("postgres", url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func pfxDeTeste(t *testing.T, cn string) []byte {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	tpl := &x509.Certificate{SerialNumber: big.NewInt(7), Subject: pkix.Name{CommonName: cn},
		NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(24 * time.Hour)}
	der, err := x509.CreateCertificate(rand.Reader, tpl, tpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	cert, _ := x509.ParseCertificate(der)
	pfx, err := pkcs12.Modern.Encode(key, cert, nil, "senha")
	if err != nil {
		t.Fatal(err)
	}
	return pfx
}

func chamar(h http.HandlerFunc, metodo, url string, corpo []byte, ct string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(metodo, url, bytes.NewReader(corpo))
	if ct != "" {
		req.Header.Set("Content-Type", ct)
	}
	rr := httptest.NewRecorder()
	h(rr, req)
	return rr
}

func upload(t *testing.T, h http.HandlerFunc, pfx []byte, senha string) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, _ := mw.CreateFormFile("pfx", "cert.pfx")
	fw.Write(pfx)
	mw.WriteField("senha", senha)
	mw.Close()
	return chamar(h, http.MethodPost, "/", buf.Bytes(), mw.FormDataContentType())
}

// Fluxo completo sem falar com a Sefin: cadastro, certificado, prévia e salvaguardas de produção.
func TestNFSeFluxoSemEnvio(t *testing.T) {
	t.Setenv("FINANCEIRO_SECRET", "segredo-de-teste")
	// Certificados de teste não têm raiz ICP-Brasil: o passo de cadeia é trocado por um stub.
	original := completarCadeiaNFSe
	completarCadeiaNFSe = func(_ context.Context, _ *x509.Certificate, i []*x509.Certificate) ([]*x509.Certificate, error) {
		return i, nil
	}
	t.Cleanup(func() { completarCadeiaNFSe = original })
	db := nfseTestDB(t)
	cnpj := "38149716000128"

	db.Exec(`DELETE FROM financeiro.nfse_emitidas`)
	db.Exec(`DELETE FROM financeiro.nfse_certificado`)
	db.Exec(`DELETE FROM financeiro.nfse_dps_contador`)
	db.Exec(`DELETE FROM financeiro.empresas`)
	var empID string
	if err := db.QueryRow(`INSERT INTO financeiro.empresas (razao_social, cnpj, logradouro, numero, bairro, cep, municipio, uf)
		VALUES ('FORTES BEZERRA', $1, 'RUA PLATAO', 'SN', 'VILLA DEIFIORE', '74962320', 'Aparecida de Goiânia', 'GO') RETURNING id`, cnpj).Scan(&empID); err != nil {
		t.Fatal(err)
	}
	var cliID string
	if err := db.QueryRow(`INSERT INTO financeiro.clientes (razao_social, cnpj, cep, logradouro, numero, bairro, municipio, uf)
		VALUES ('JC DISTRIBUICAO LTDA', '06314327000203', '74985240', 'EXPRESSA ANEL VIARIO', 'S/N', 'POLO EMPRESARIAL GOIAS', 'Aparecida de Goiânia', 'GO')
		ON CONFLICT (cnpj) DO UPDATE SET razao_social = EXCLUDED.razao_social RETURNING id`).Scan(&cliID); err != nil {
		t.Fatal(err)
	}

	cert := NFSeCertificadoHandler(db)
	prest := NFSePrestadorHandler(db)
	emitir := NFSeEmitirHandler(db)

	// Sem certificado e sem IBGE, nada de emissão.
	corpo, _ := json.Marshal(map[string]any{"cliente_id": cliID, "valor_centavos": 100, "descricao": "Teste"})
	if rr := chamar(emitir, http.MethodPost, "/", corpo, "application/json"); rr.Code != http.StatusConflict {
		t.Fatalf("sem IBGE esperava 409, veio %d: %s", rr.Code, rr.Body)
	}

	// Parâmetros fiscais + IBGE do cliente.
	pr, _ := json.Marshal(map[string]any{"cod_ibge": "5201405", "telefone": "62998483451", "email": "p@exemplo.com.br",
		"op_simp_nac": 3, "reg_ap_trib_sn": 1, "reg_esp_trib": 0, "nfse_ambiente": 2})
	if rr := chamar(prest, http.MethodPut, "/", pr, "application/json"); rr.Code != 200 {
		t.Fatalf("prestador: %d %s", rr.Code, rr.Body)
	}
	if _, err := db.Exec(`UPDATE financeiro.clientes SET cod_ibge='5201405' WHERE id=$1`, cliID); err != nil {
		t.Fatal(err)
	}

	// Sem certificado cadastrado.
	if rr := chamar(emitir, http.MethodPost, "/", corpo, "application/json"); rr.Code != http.StatusConflict || !strings.Contains(rr.Body.String(), "certificado") {
		t.Fatalf("sem certificado esperava 409: %d %s", rr.Code, rr.Body)
	}

	// Certificado de outro CNPJ é recusado; o certo é aceito e nunca devolvido.
	if rr := upload(t, cert, pfxDeTeste(t, "OUTRA:00000000000191"), "senha"); rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("certificado de outro CNPJ esperava 422: %d %s", rr.Code, rr.Body)
	}
	if rr := upload(t, cert, pfxDeTeste(t, "FORTES BEZERRA:"+cnpj), "senha"); rr.Code != 200 {
		t.Fatalf("upload: %d %s", rr.Code, rr.Body)
	}
	rr := chamar(cert, http.MethodGet, "/", nil, "")
	if !strings.Contains(rr.Body.String(), `"cnpj_confere":true`) || strings.Contains(rr.Body.String(), "pfx") && strings.Contains(rr.Body.String(), "senha") {
		t.Fatalf("status do certificado inesperado: %s", rr.Body)
	}
	var pfxGuardado string
	db.QueryRow(`SELECT certificado_pfx FROM financeiro.nfse_certificado`).Scan(&pfxGuardado)
	if pfxGuardado == "" || strings.HasPrefix(pfxGuardado, "plain:") {
		t.Fatal("PFX deve estar cifrado no banco")
	}

	// Prévia: devolve DPS assinado, não reserva número, não grava nota.
	rr = chamar(emitir, http.MethodPost, "/", corpo, "application/json")
	var previa struct {
		Previa bool   `json:"previa"`
		DPSXML string `json:"dps_xml"`
	}
	if rr.Code != 200 || json.Unmarshal(rr.Body.Bytes(), &previa) != nil || !previa.Previa || !strings.Contains(previa.DPSXML, "<Signature") {
		t.Fatalf("prévia: %d %.300s", rr.Code, rr.Body)
	}
	var n int
	db.QueryRow(`SELECT COUNT(*) FROM financeiro.nfse_dps_contador`).Scan(&n)
	if n != 0 {
		t.Fatal("prévia não pode reservar número de DPS")
	}
	db.QueryRow(`SELECT COUNT(*) FROM financeiro.nfse_emitidas`).Scan(&n)
	if n != 0 {
		t.Fatal("prévia não pode gravar nota")
	}

	// Produção exige a frase de confirmação para enviar — e recusa ANTES de reservar número.
	pr, _ = json.Marshal(map[string]any{"cod_ibge": "5201405", "op_simp_nac": 3, "reg_ap_trib_sn": 1, "nfse_ambiente": 1})
	if rr := chamar(prest, http.MethodPut, "/", pr, "application/json"); rr.Code != 400 {
		t.Fatalf("ativar produção sem frase esperava 400: %d", rr.Code)
	}
	pr, _ = json.Marshal(map[string]any{"cod_ibge": "5201405", "op_simp_nac": 3, "reg_ap_trib_sn": 1, "nfse_ambiente": 1, "confirmar_producao": nfseFraseProducao})
	if rr := chamar(prest, http.MethodPut, "/", pr, "application/json"); rr.Code != 200 {
		t.Fatalf("ativar produção: %d %s", rr.Code, rr.Body)
	}
	envio, _ := json.Marshal(map[string]any{"cliente_id": cliID, "valor_centavos": 100, "descricao": "Teste", "enviar": true})
	if rr := chamar(emitir, http.MethodPost, "/", envio, "application/json"); rr.Code != 400 || !strings.Contains(rr.Body.String(), "PRODUÇÃO") {
		t.Fatalf("envio em produção sem frase esperava 400: %d %s", rr.Code, rr.Body)
	}
	db.QueryRow(`SELECT COUNT(*) FROM financeiro.nfse_dps_contador`).Scan(&n)
	if n != 0 {
		t.Fatal("recusa de produção não pode ter reservado número")
	}
}

// O código IBGE do município precisa ser gravado e devolvido pelo cadastro de clientes.
func TestNFSeClientes_CodIBGEGravaEDevolve(t *testing.T) {
	db := nfseTestDB(t)
	h := ClientesHandler(db)
	cnpj := "10230480001960"
	db.Exec(`DELETE FROM financeiro.clientes WHERE cnpj = $1`, cnpj)
	t.Cleanup(func() { db.Exec(`DELETE FROM financeiro.clientes WHERE cnpj = $1`, cnpj) })

	novo := func(ibge string) []byte {
		b, _ := json.Marshal(map[string]any{"razao_social": "CLIENTE TESTE", "cnpj": cnpj, "ativo": true,
			"cep": "74985240", "logradouro": "RUA A", "numero": "1", "bairro": "B", "municipio": "Aparecida", "uf": "GO", "cod_ibge": ibge})
		return b
	}
	if rr := chamar(h, http.MethodPost, "/", novo("52014"), "application/json"); rr.Code != http.StatusBadRequest {
		t.Fatalf("IBGE com 5 dígitos deveria dar 400, veio %d", rr.Code)
	}
	if rr := chamar(h, http.MethodPost, "/", novo("5201405"), "application/json"); rr.Code >= 300 {
		t.Fatalf("criar: %d %s", rr.Code, rr.Body)
	}
	var lista []struct {
		ID      string `json:"id"`
		CodIBGE string `json:"cod_ibge"`
		CNPJ    string `json:"cnpj"`
	}
	rr := chamar(h, http.MethodGet, "/?q="+cnpj, nil, "")
	if err := json.Unmarshal(rr.Body.Bytes(), &lista); err != nil || len(lista) != 1 || lista[0].CodIBGE != "5201405" {
		t.Fatalf("listagem sem IBGE: %d %s", rr.Code, rr.Body)
	}
	id := lista[0].ID
	var um struct {
		CodIBGE string `json:"cod_ibge"`
	}
	rr = chamar(h, http.MethodGet, "/?id="+id, nil, "")
	if json.Unmarshal(rr.Body.Bytes(), &um) != nil || um.CodIBGE != "5201405" {
		t.Fatalf("busca por id sem IBGE: %s", rr.Body)
	}
	// Edição: troca e depois limpa.
	b, _ := json.Marshal(map[string]any{"id": id, "razao_social": "CLIENTE TESTE", "ativo": true, "cod_ibge": "2611606"})
	if rr := chamar(h, http.MethodPut, "/", b, "application/json"); rr.Code >= 300 {
		t.Fatalf("editar: %d %s", rr.Code, rr.Body)
	}
	var ibge sql.NullString
	db.QueryRow(`SELECT cod_ibge FROM financeiro.clientes WHERE id=$1`, id).Scan(&ibge)
	if ibge.String != "2611606" {
		t.Fatalf("IBGE não foi atualizado: %q", ibge.String)
	}
}

// Sem certificado cadastrado o handler não consegue falar com o ADN e gera o DANFSe do XML.
func TestNFSeDANFSeHandler_GeraLocalDoXML(t *testing.T) {
	db := nfseTestDB(t)
	xml, err := os.ReadFile("../services/testdata/nfse_nota1_real.xml")
	if err != nil {
		t.Fatal(err)
	}
	db.Exec(`DELETE FROM financeiro.nfse_emitidas`)
	db.Exec(`DELETE FROM financeiro.nfse_certificado`)
	db.Exec(`DELETE FROM financeiro.empresas`)
	var empID, id string
	if err := db.QueryRow(`INSERT INTO financeiro.empresas (razao_social, cnpj, logradouro, numero, bairro, cep, municipio, uf)
		VALUES ('FORTES BEZERRA', '38149716000128', 'RUA', 'SN', 'B', '74962320', 'Aparecida', 'GO') RETURNING id`).Scan(&empID); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`INSERT INTO financeiro.nfse_emitidas (empresa_id, ambiente, serie, n_dps, id_dps, competencia, valor, descricao, status, chave_acesso, nfse_xml)
		VALUES ($1, 1, '00001', 1, 'DPS-TESTE-DANFSE', '2026-11-01', 24000, 'x', 'autorizada', '52014052238149716000128000000000000126119042182716', $2) RETURNING id`, empID, string(xml)).Scan(&id); err != nil {
		t.Fatal(err)
	}
	rr := chamar(NFSeDANFSeHandler(db), http.MethodGet, "/?id="+id, nil, "")
	if rr.Code != 200 || rr.Header().Get("X-DANFSe-Origem") != "local" || !bytes.HasPrefix(rr.Body.Bytes(), []byte("%PDF")) {
		t.Fatalf("DANFSe local: %d origem=%q %.120s", rr.Code, rr.Header().Get("X-DANFSe-Origem"), rr.Body)
	}
	db.Exec(`UPDATE financeiro.nfse_emitidas SET status = 'rejeitada' WHERE id = $1`, id)
	if rr := chamar(NFSeDANFSeHandler(db), http.MethodGet, "/?id="+id, nil, ""); rr.Code != http.StatusConflict {
		t.Fatalf("nota rejeitada não tem DANFSe: %d", rr.Code)
	}
}

func TestBancosConfig_SegredosNuncaVoltamEBrancoPreserva(t *testing.T) {
	t.Setenv("FINANCEIRO_SECRET", "segredo-de-teste")
	db := nfseTestDB(t)
	var contaID string
	if err := db.QueryRow(`INSERT INTO financeiro.contas_financeiras (apelido, banco, provedor) VALUES ('Inter teste', '077 - INTER', 'inter') RETURNING id`).Scan(&contaID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Exec(`DELETE FROM financeiro.contas_financeiras WHERE id = $1`, contaID) })

	// Certificado e chave de teste (par válido).
	key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	tpl := &x509.Certificate{SerialNumber: big.NewInt(3), Subject: pkix.Name{CommonName: "t"}, NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(200 * 24 * time.Hour)}
	der, _ := x509.CreateCertificate(rand.Reader, tpl, tpl, &key.PublicKey, key)
	kb, _ := x509.MarshalECPrivateKey(key)
	certPEM := string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
	keyPEM := string(pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: kb}))

	h := BancosConfigHandler(db)
	put := func(cfg map[string]string) *httptest.ResponseRecorder {
		b, _ := json.Marshal(map[string]any{"conta_id": contaID, "provedor": "inter", "config": cfg})
		return chamar(h, http.MethodPut, "/", b, "application/json")
	}
	if rr := put(map[string]string{"client_id": "abc-1234", "client_secret": "SEGREDO-SUPER", "cert_pem": certPEM, "key_pem": keyPEM, "conta_corrente": "999"}); rr.Code != 200 {
		t.Fatalf("salvar: %d %s", rr.Code, rr.Body)
	}
	// Segunda gravação só com a conta corrente: o resto precisa ser preservado.
	if rr := put(map[string]string{"conta_corrente": "888"}); rr.Code != 200 {
		t.Fatalf("salvar parcial: %d %s", rr.Code, rr.Body)
	}
	rr := chamar(h, http.MethodGet, "/?conta_id="+contaID, nil, "")
	corpo := rr.Body.String()
	for _, proibido := range []string{"SEGREDO-SUPER", "BEGIN", "abc-1234"} {
		if strings.Contains(corpo, proibido) {
			t.Fatalf("a resposta não pode conter %q: %s", proibido, corpo)
		}
	}
	for _, esperado := range []string{`"tem_secret":true`, `"tem_certificado":true`, `"tem_chave":true`, `"client_id_final":"1234"`, `"conta_corrente":"888"`} {
		if !strings.Contains(corpo, esperado) {
			t.Fatalf("faltou %s em %s", esperado, corpo)
		}
	}
	var guardado string
	db.QueryRow(`SELECT provedor_config FROM financeiro.contas_financeiras WHERE id=$1`, contaID).Scan(&guardado)
	if strings.Contains(guardado, "SEGREDO") || strings.HasPrefix(guardado, "plain:") {
		t.Fatal("config deve estar cifrada no banco")
	}
	// Par inválido (chave de outro certificado) é recusado.
	key2, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	kb2, _ := x509.MarshalECPrivateKey(key2)
	outraChave := string(pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: kb2}))
	if rr := put(map[string]string{"key_pem": outraChave}); rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("par inválido deveria dar 422, veio %d", rr.Code)
	}
	// Sem FINANCEIRO_SECRET não grava.
	t.Setenv("FINANCEIRO_SECRET", "")
	if rr := put(map[string]string{"client_id": "x"}); rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("sem chave de criptografia deveria dar 422, veio %d", rr.Code)
	}
}

// Manutenção de contas, histórico de sincronização, filtros de lançamentos e validação do período.
func TestBancosManutencao_ContaSyncLogETransacoes(t *testing.T) {
	db := nfseTestDB(t)
	contas := ContasFinanceirasHandler(db)

	rr := chamar(contas, http.MethodPost, "/", []byte(`{"apelido":"Inter manut","banco":"077 - INTER","agencia":"0001","conta":"123","tipo":"corrente","provedor":"inter"}`), "application/json")
	var criada struct {
		ID string `json:"id"`
	}
	if rr.Code != http.StatusCreated || json.Unmarshal(rr.Body.Bytes(), &criada) != nil || criada.ID == "" {
		t.Fatalf("criar conta: %d %s", rr.Code, rr.Body)
	}
	t.Cleanup(func() {
		db.Exec(`DELETE FROM financeiro.inter_sync_log WHERE conta_id = $1`, criada.ID)
		db.Exec(`DELETE FROM financeiro.contas_financeiras WHERE id = $1`, criada.ID)
	})
	// Editar.
	b, _ := json.Marshal(map[string]any{"id": criada.ID, "apelido": "Inter renomeada", "banco": "077 - INTER", "agencia": "0001", "conta": "999", "tipo": "corrente", "ativa": true})
	if rr := chamar(contas, http.MethodPut, "/", b, "application/json"); rr.Code != 200 {
		t.Fatalf("editar conta: %d %s", rr.Code, rr.Body)
	}
	var apelido, conta string
	db.QueryRow(`SELECT apelido, conta FROM financeiro.contas_financeiras WHERE id=$1`, criada.ID).Scan(&apelido, &conta)
	if apelido != "Inter renomeada" || conta != "999" {
		t.Fatalf("edição não gravou: %q %q", apelido, conta)
	}
	if rr := chamar(contas, http.MethodPut, "/", []byte(`{"id":"`+criada.ID+`","apelido":"","banco":""}`), "application/json"); rr.Code != http.StatusBadRequest {
		t.Fatalf("PUT sem apelido/banco deveria dar 400: %d", rr.Code)
	}

	// Sincronização: período inválido (400), sem credenciais (422), e uma em andamento bloqueia outra (409).
	sync := BancosSyncHandler(db)
	post := func(corpo string) *httptest.ResponseRecorder {
		return chamar(sync, http.MethodPost, "/", []byte(corpo), "application/json")
	}
	if rr := post(`{"conta_id":"` + criada.ID + `","data_inicio":"2026-01-01","data_fim":"2026-12-31"}`); rr.Code != http.StatusBadRequest {
		t.Fatalf("período > 90 dias: %d %s", rr.Code, rr.Body)
	}
	if rr := post(`{"conta_id":"` + criada.ID + `"}`); rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("sem credenciais: %d %s", rr.Code, rr.Body)
	}
	db.Exec(`UPDATE financeiro.contas_financeiras SET provedor_config = 'x' WHERE id = $1`, criada.ID)
	db.Exec(`INSERT INTO financeiro.inter_sync_log (conta_id, status, data_inicio, data_fim) VALUES ($1, 'running', '2026-10-01', '2026-10-31')`, criada.ID)
	if rr := post(`{"conta_id":"` + criada.ID + `"}`); rr.Code != http.StatusConflict {
		t.Fatalf("sync em andamento: %d %s", rr.Code, rr.Body)
	}

	// Histórico.
	db.Exec(`INSERT INTO financeiro.inter_sync_log (conta_id, status, data_inicio, data_fim, tx_importadas, tx_duplicadas, encerrado_em) VALUES ($1, 'ok', '2026-09-01', '2026-09-30', 7, 3, NOW())`, criada.ID)
	rr = chamar(BancosSyncLogHandler(db), http.MethodGet, "/?conta_id="+criada.ID, nil, "")
	var logs []map[string]any
	if json.Unmarshal(rr.Body.Bytes(), &logs) != nil || len(logs) != 2 {
		t.Fatalf("histórico: %d %s", rr.Code, rr.Body)
	}

	// Lançamentos: filtros por período, tipo e texto.
	for _, l := range []struct{ data, desc, tipo, cat, ref string }{
		{"2026-10-05", "PIX RECEBIDO JC", "credito", "PIX", "ft-1"},
		{"2026-10-20", "BOLETO LIQUIDADO", "credito", "BOLETO_COBRANCA", "ft-2"},
		{"2026-10-21", "PAGAMENTO CONTA", "debito", "PAGAMENTO", "ft-3"},
		{"2026-09-10", "PIX ANTIGO", "credito", "PIX", "ft-4"},
	} {
		db.Exec(`INSERT INTO financeiro.transacoes (conta_id, data_transacao, descricao, valor, tipo, categoria, referencia_ext) VALUES ($1,$2,$3,10,$4,$5,$6) ON CONFLICT DO NOTHING`, criada.ID, l.data, l.desc, l.tipo, l.cat, l.ref)
	}
	lista := func(q string) int {
		rr := chamar(TransacoesHandler(db), http.MethodGet, "/?conta_id="+criada.ID+q, nil, "")
		var out []map[string]any
		json.Unmarshal(rr.Body.Bytes(), &out)
		return len(out)
	}
	if n := lista("&data_inicio=2026-10-01&data_fim=2026-10-31"); n != 3 {
		t.Errorf("período de outubro: %d, esperado 3", n)
	}
	if n := lista("&data_inicio=2026-10-01&data_fim=2026-10-31&tipo=debito"); n != 1 {
		t.Errorf("só saídas: %d, esperado 1", n)
	}
	if n := lista("&q=pix"); n != 2 {
		t.Errorf("busca por pix: %d, esperado 2", n)
	}
	if n := lista("&q=BOLETO_COB"); n != 1 {
		t.Errorf("busca pela categoria: %d, esperado 1", n)
	}
	db.Exec(`DELETE FROM financeiro.transacoes WHERE conta_id = $1`, criada.ID)
}

func TestTransacoes_DevolveDetalhesDoExtratoEnriquecido(t *testing.T) {
	db := nfseTestDB(t)
	var contaID string
	if err := db.QueryRow(`INSERT INTO financeiro.contas_financeiras (apelido, banco) VALUES ('Detalhes teste','077') RETURNING id`).Scan(&contaID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		db.Exec(`DELETE FROM financeiro.transacoes WHERE conta_id = $1`, contaID)
		db.Exec(`DELETE FROM financeiro.contas_financeiras WHERE id = $1`, contaID)
	})
	db.Exec(`INSERT INTO financeiro.transacoes (conta_id, data_transacao, descricao, valor, tipo, categoria, referencia_ext, detalhes)
		VALUES ($1,'2026-11-11','Jc Distrib',4500,'credito','PIX','det-1','{"cpfCnpjPagador":"06314327000203","nomePagador":"Jc Distrib","endToEndId":"E123"}')`, contaID)
	db.Exec(`INSERT INTO financeiro.transacoes (conta_id, data_transacao, descricao, valor, tipo, categoria, referencia_ext) VALUES ($1,'2026-11-12','Sem detalhe',1,'debito','TARIFA','det-2')`, contaID)
	rr := chamar(TransacoesHandler(db), http.MethodGet, "/?conta_id="+contaID, nil, "")
	var out []struct {
		Descricao string         `json:"descricao"`
		Detalhes  map[string]any `json:"detalhes"`
	}
	if json.Unmarshal(rr.Body.Bytes(), &out) != nil || len(out) != 2 {
		t.Fatalf("lista: %d %s", rr.Code, rr.Body)
	}
	for _, o := range out {
		if o.Descricao == "Jc Distrib" && o.Detalhes["cpfCnpjPagador"] != "06314327000203" {
			t.Errorf("detalhes do Pix recebido não vieram: %v", o.Detalhes)
		}
		if o.Descricao == "Sem detalhe" && o.Detalhes != nil {
			t.Errorf("lançamento sem detalhes deveria vir sem o campo: %v", o.Detalhes)
		}
	}
}

func TestReceber_APIDeTitulos(t *testing.T) {
	db := nfseTestDB(t)
	suf := fmt.Sprintf("%d", time.Now().UnixNano())
	var cli, contrato, prod, plano, conta string
	must := func(dest *string, q string, args ...any) {
		if err := db.QueryRow(q, args...).Scan(dest); err != nil {
			t.Fatalf("%s: %v", q, err)
		}
	}
	must(&cli, `INSERT INTO financeiro.clientes (razao_social, cnpj) VALUES ('CLIENTE API', '06314327000121') RETURNING id`)
	must(&contrato, `INSERT INTO financeiro.contratos (cliente_id, data_inicio, valor_total) VALUES ($1, CURRENT_DATE, 100) RETURNING id`, cli)
	must(&prod, `INSERT INTO financeiro.produtos (codigo, nome) VALUES ($1,'P') RETURNING id`, "R"+suf)
	must(&plano, `INSERT INTO financeiro.planos (produto_id, nome) VALUES ($1,'PL') RETURNING id`, prod)
	must(&conta, `INSERT INTO financeiro.contas_financeiras (apelido, banco) VALUES ('Receber API','077') RETURNING id`)
	db.Exec(`INSERT INTO financeiro.tokens (contrato_id, plano_id, token, valid_from, valid_until, status) VALUES ($1,$2,$3, CURRENT_DATE-60, CURRENT_DATE-15,'em_carencia')`, contrato, plano, "t"+suf)
	t.Cleanup(func() {
		db.Exec(`DELETE FROM financeiro.titulos_receber WHERE cliente_id=$1`, cli)
		db.Exec(`DELETE FROM financeiro.tokens WHERE contrato_id=$1`, contrato)
		db.Exec(`DELETE FROM financeiro.contratos WHERE id=$1`, contrato)
		db.Exec(`DELETE FROM financeiro.clientes WHERE id=$1`, cli)
		db.Exec(`DELETE FROM financeiro.planos WHERE id=$1`, plano)
		db.Exec(`DELETE FROM financeiro.produtos WHERE id=$1`, prod)
		db.Exec(`DELETE FROM financeiro.transacoes WHERE conta_id=$1`, conta)
		db.Exec(`DELETE FROM financeiro.contas_financeiras WHERE id=$1`, conta)
	})

	h := TitulosHandler(db)
	post := func(q, corpo string) *httptest.ResponseRecorder {
		return chamar(h, http.MethodPost, "/?"+q, []byte(corpo), "application/json")
	}
	venc := time.Now().AddDate(0, 0, 9).Format("2006-01-02")
	comp := time.Now().Format("2006-01-02")

	// validações
	for nome, corpo := range map[string]string{
		"sem valor":                   fmt.Sprintf(`{"cliente_id":"%s","tipo":"adesao","descricao":"x","valor":0,"vencimento":"%s"}`, cli, venc),
		"sem vencimento":              fmt.Sprintf(`{"cliente_id":"%s","tipo":"adesao","descricao":"x","valor":10}`, cli),
		"mensalidade sem contrato":    fmt.Sprintf(`{"cliente_id":"%s","tipo":"mensalidade","competencia":"%s","descricao":"x","valor":10,"vencimento":"%s"}`, cli, comp, venc),
		"mensalidade sem competência": fmt.Sprintf(`{"cliente_id":"%s","contrato_id":"%s","tipo":"mensalidade","descricao":"x","valor":10,"vencimento":"%s"}`, cli, contrato, venc),
		"tipo inválido":               fmt.Sprintf(`{"cliente_id":"%s","tipo":"multa","descricao":"x","valor":10,"vencimento":"%s"}`, cli, venc),
	} {
		if rr := post("", corpo); rr.Code != http.StatusBadRequest {
			t.Errorf("%s: esperava 400, veio %d %s", nome, rr.Code, rr.Body)
		}
	}

	mens := fmt.Sprintf(`{"cliente_id":"%s","contrato_id":"%s","tipo":"mensalidade","competencia":"%s","descricao":"Mensalidade SmartPick e Farol","valor":4500,"vencimento":"%s"}`, cli, contrato, comp, venc)
	rr := post("", mens)
	var criado struct {
		ID string `json:"id"`
	}
	if rr.Code != http.StatusCreated || json.Unmarshal(rr.Body.Bytes(), &criado) != nil || criado.ID == "" {
		t.Fatalf("criar mensalidade: %d %s", rr.Code, rr.Body)
	}
	if rr := post("", mens); rr.Code != http.StatusConflict {
		t.Fatalf("mensalidade duplicada deveria dar 409: %d %s", rr.Code, rr.Body)
	}
	adesao := post("", fmt.Sprintf(`{"cliente_id":"%s","tipo":"adesao","descricao":"Taxa de adesão","valor":3000,"vencimento":"%s"}`, cli, venc))
	var adesaoID struct {
		ID string `json:"id"`
	}
	json.Unmarshal(adesao.Body.Bytes(), &adesaoID)

	// lista
	rr = chamar(h, http.MethodGet, "/?cliente_id="+cli, nil, "")
	var lista []map[string]any
	if json.Unmarshal(rr.Body.Bytes(), &lista) != nil || len(lista) != 2 {
		t.Fatalf("lista: %d %s", rr.Code, rr.Body)
	}

	// candidatos + vincular: crédito com valor igual aparece primeiro; baixa renova o token.
	var tx string
	must(&tx, `INSERT INTO financeiro.transacoes (conta_id, data_transacao, descricao, valor, tipo, categoria, referencia_ext)
		VALUES ($1, CURRENT_DATE, 'TED sem detalhe', 4500, 'credito', 'TRANSFERENCIA', $2) RETURNING id`, conta, "api-"+suf)
	rr = chamar(h, http.MethodGet, "/?action=candidatos&id="+criado.ID, nil, "")
	var cands []map[string]any
	if json.Unmarshal(rr.Body.Bytes(), &cands) != nil || len(cands) == 0 || cands[0]["transacao_id"] != tx || cands[0]["valor_igual"] != true {
		t.Fatalf("candidatos: %d %s", rr.Code, rr.Body)
	}
	rr = post("action=vincular", fmt.Sprintf(`{"id":"%s","transacao_id":"%s"}`, criado.ID, tx))
	if rr.Code != 200 || !strings.Contains(rr.Body.String(), `"tokens_renovados":1`) {
		t.Fatalf("vincular: %d %s", rr.Code, rr.Body)
	}
	if rr := post("action=vincular", fmt.Sprintf(`{"id":"%s","transacao_id":"%s"}`, adesaoID.ID, tx)); rr.Code != http.StatusConflict {
		t.Fatalf("o mesmo lançamento não baixa dois títulos: %d %s", rr.Code, rr.Body)
	}
	var ativos int
	db.QueryRow(`SELECT COUNT(*) FROM financeiro.tokens WHERE contrato_id=$1 AND status='ativo'`, contrato).Scan(&ativos)
	if ativos != 1 {
		t.Errorf("a mensalidade paga deveria ter ativado o token (ativos=%d)", ativos)
	}

	// baixa manual da adesão (não renova token), edição bloqueada depois de paga, cancelamento só em aberto.
	rr = post("action=baixar", fmt.Sprintf(`{"id":"%s","data":"%s","valor":3000,"obs":"recebido em dinheiro"}`, adesaoID.ID, comp))
	if rr.Code != 200 || !strings.Contains(rr.Body.String(), `"tokens_renovados":0`) {
		t.Fatalf("baixa manual: %d %s", rr.Code, rr.Body)
	}
	put := chamar(h, http.MethodPut, "/", []byte(fmt.Sprintf(`{"id":"%s","cliente_id":"%s","tipo":"adesao","descricao":"x","valor":1,"vencimento":"%s"}`, adesaoID.ID, cli, venc)), "application/json")
	if put.Code != http.StatusConflict {
		t.Fatalf("editar título pago deveria dar 409: %d", put.Code)
	}
	if rr := post("action=cancelar", fmt.Sprintf(`{"id":"%s"}`, adesaoID.ID)); rr.Code != http.StatusConflict {
		t.Fatalf("cancelar título pago deveria dar 409: %d", rr.Code)
	}
	novo := post("", fmt.Sprintf(`{"cliente_id":"%s","tipo":"projeto","descricao":"Projeto X","valor":10,"vencimento":"%s"}`, cli, venc))
	var novoID struct {
		ID string `json:"id"`
	}
	json.Unmarshal(novo.Body.Bytes(), &novoID)
	if rr := post("action=cancelar", fmt.Sprintf(`{"id":"%s","obs":"cliente desistiu"}`, novoID.ID)); rr.Code != 200 {
		t.Fatalf("cancelar: %d %s", rr.Code, rr.Body)
	}
}

// Contrato de serviço (sem itens de produto) pode ser cadastrado e não gera token.
func TestContrato_ServicoSemItensNaoGeraToken(t *testing.T) {
	db := nfseTestDB(t)
	var cli string
	if err := db.QueryRow(`INSERT INTO financeiro.clientes (razao_social, cnpj) VALUES ('FERREIRA TESTE', '10230480001960') ON CONFLICT (cnpj) DO UPDATE SET razao_social = EXCLUDED.razao_social RETURNING id`).Scan(&cli); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		db.Exec(`DELETE FROM financeiro.titulos_receber WHERE cliente_id = $1`, cli)
		db.Exec(`DELETE FROM financeiro.contratos WHERE cliente_id = $1`, cli)
		db.Exec(`DELETE FROM financeiro.clientes WHERE id = $1`, cli)
	})
	corpo := fmt.Sprintf(`{"cliente_id":"%s","data_inicio":"2026-11-01","periodicidade":"mensal","valor_total":24000,"observacoes":"Processamento de dados","cnpj_ids":[],"itens":[]}`, cli)
	rr := chamar(ContratosHandler(db), http.MethodPost, "/", []byte(corpo), "application/json")
	var criado struct {
		ID string `json:"id"`
	}
	if rr.Code != http.StatusCreated || json.Unmarshal(rr.Body.Bytes(), &criado) != nil || criado.ID == "" {
		t.Fatalf("contrato de serviço: %d %s", rr.Code, rr.Body)
	}
	var tokens, itens int
	db.QueryRow(`SELECT COUNT(*) FROM financeiro.tokens WHERE contrato_id = $1`, criado.ID).Scan(&tokens)
	db.QueryRow(`SELECT COUNT(*) FROM financeiro.contrato_itens WHERE contrato_id = $1`, criado.ID).Scan(&itens)
	if tokens != 0 || itens != 0 {
		t.Errorf("contrato de serviço não deveria ter itens nem tokens: itens=%d tokens=%d", itens, tokens)
	}
	// E a mensalidade desse contrato baixada não tenta renovar token.
	var titulo string
	if err := db.QueryRow(`INSERT INTO financeiro.titulos_receber (cliente_id, contrato_id, tipo, competencia, descricao, valor, vencimento)
		VALUES ($1,$2,'mensalidade','2026-11-01','Serviço',24000,'2026-11-06') RETURNING id`, cli, criado.ID).Scan(&titulo); err != nil {
		t.Fatal(err)
	}
	rr = chamar(TitulosHandler(db), http.MethodPost, "/?action=baixar", []byte(fmt.Sprintf(`{"id":"%s","data":"2026-11-06","valor":24000}`, titulo)), "application/json")
	if rr.Code != 200 || !strings.Contains(rr.Body.String(), `"tokens_renovados":0`) {
		t.Fatalf("baixa de mensalidade sem token: %d %s", rr.Code, rr.Body)
	}
}

// Editar o CNPJ do cliente grava em clientes E na linha principal de cliente_cnpjs (os contratos
// apontam para ela), recusa CNPJ inválido ou de outro cliente e preserva o atual se vier em branco.
func TestClientes_AlterarCNPJ(t *testing.T) {
	db := nfseTestDB(t)
	h := ClientesHandler(db)
	const matriz, filial, outro = "06314327000203", "38149716000128", "10230480001960"
	limpar := func() {
		db.Exec(`DELETE FROM financeiro.titulos_receber WHERE cliente_id IN (SELECT id FROM financeiro.clientes WHERE cnpj IN ($1,$2,$3))`, matriz, filial, outro)
		db.Exec(`DELETE FROM financeiro.contratos WHERE cliente_id IN (SELECT id FROM financeiro.clientes WHERE cnpj IN ($1,$2,$3))`, matriz, filial, outro)
		db.Exec(`DELETE FROM financeiro.clientes WHERE cnpj IN ($1,$2,$3)`, matriz, filial, outro)
	}
	limpar()
	t.Cleanup(limpar)

	criar := func(razao, cnpj string) string {
		b, _ := json.Marshal(map[string]any{"razao_social": razao, "cnpj": cnpj, "ativo": true})
		rr := chamar(h, http.MethodPost, "/", b, "application/json")
		var c struct {
			ID string `json:"id"`
		}
		if rr.Code != http.StatusCreated || json.Unmarshal(rr.Body.Bytes(), &c) != nil {
			t.Fatalf("criar %s: %d %s", razao, rr.Code, rr.Body)
		}
		return c.ID
	}
	jc := criar("JC TESTE", filial)
	criar("OUTRO CLIENTE", outro)

	// Contrato vinculado ao CNPJ principal.
	var cnpjID string
	db.QueryRow(`SELECT id FROM financeiro.cliente_cnpjs WHERE cliente_id = $1`, jc).Scan(&cnpjID)
	var contrato string
	if err := db.QueryRow(`INSERT INTO financeiro.contratos (cliente_id, data_inicio, valor_total) VALUES ($1, CURRENT_DATE, 4500) RETURNING id`, jc).Scan(&contrato); err != nil {
		t.Fatal(err)
	}
	db.Exec(`INSERT INTO financeiro.contrato_cnpjs (contrato_id, cnpj_id) VALUES ($1,$2)`, contrato, cnpjID)

	put := func(cnpj string) *httptest.ResponseRecorder {
		b, _ := json.Marshal(map[string]any{"id": jc, "razao_social": "JC TESTE", "cnpj": cnpj, "ativo": true})
		return chamar(h, http.MethodPut, "/", b, "application/json")
	}
	// 1) muda para a outra filial
	if rr := put("06.314.327/0002-03"); rr.Code != 200 {
		t.Fatalf("alterar CNPJ: %d %s", rr.Code, rr.Body)
	}
	var atual, principal, idDepois string
	db.QueryRow(`SELECT cnpj FROM financeiro.clientes WHERE id = $1`, jc).Scan(&atual)
	db.QueryRow(`SELECT id, cnpj FROM financeiro.cliente_cnpjs WHERE cliente_id = $1`, jc).Scan(&idDepois, &principal)
	if atual != matriz || principal != matriz {
		t.Fatalf("CNPJ não foi gravado: clientes=%s cliente_cnpjs=%s", atual, principal)
	}
	if idDepois != cnpjID {
		t.Errorf("a linha do CNPJ principal deve ser a mesma (contratos apontam para ela): %s vs %s", idDepois, cnpjID)
	}
	var vinculos int
	db.QueryRow(`SELECT COUNT(*) FROM financeiro.contrato_cnpjs WHERE contrato_id = $1 AND cnpj_id = $2`, contrato, cnpjID).Scan(&vinculos)
	if vinculos != 1 {
		t.Errorf("o vínculo do contrato com o CNPJ deveria continuar: %d", vinculos)
	}
	// 2) inválido, curto e de outro cliente
	if rr := put("06314327000204"); rr.Code != http.StatusBadRequest {
		t.Errorf("DV inválido: %d", rr.Code)
	}
	if rr := put("123"); rr.Code != http.StatusBadRequest {
		t.Errorf("CNPJ curto: %d", rr.Code)
	}
	if rr := put(outro); rr.Code != http.StatusConflict {
		t.Errorf("CNPJ de outro cliente deveria dar 409: %d %s", rr.Code, rr.Body)
	}
	// 3) em branco mantém
	if rr := put(""); rr.Code != 200 {
		t.Fatalf("PUT sem CNPJ: %d", rr.Code)
	}
	db.QueryRow(`SELECT cnpj FROM financeiro.clientes WHERE id = $1`, jc).Scan(&atual)
	if atual != matriz {
		t.Errorf("CNPJ em branco não pode apagar nem trocar: %s", atual)
	}
}
