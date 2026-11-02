package handlers

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"database/sql"
	"encoding/json"
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
