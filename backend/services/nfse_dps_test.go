package services

import (
	"context"
	"database/sql"
	"software.sslmate.com/src/go-pkcs12"
	"sync"

	"compress/gzip"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	_ "github.com/lib/pq"
	"io"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func dadosNota1() DPSDados {
	brt := time.FixedZone("-03", -3*3600)
	return DPSDados{
		Ambiente:        1,
		Emissao:         time.Date(2026, 11, 2, 14, 58, 49, 0, brt),
		Serie:           "70000",
		NDPS:            4,
		Competencia:     time.Date(2026, 11, 1, 0, 0, 0, 0, brt),
		CodMunEmissor:   "5201405",
		CodMunPrestacao: "2611606",
		Prestador: DPSPrestador{
			CNPJ: "38.149.716/0001-28", Fone: "(62) 99848-3451", Email: "prestador@exemplo.com.br",
			OpSimpNac: 3, RegApTribSN: 1, RegEspTrib: 0,
		},
		Tomador: DPSTomador{
			CNPJ: "10.230.480/0019-60", Nome: "FERREIRA COSTA & CIA LTDA",
			CodMun: "2611606", CEP: "51150-003",
			Logradouro: "MARECHAL MASCARENHAS DE MORAES", Numero: "2967",
			Complemento: "GALPAO 0000               FERREIRA COSTA E          CIA LTDA",
			Bairro:      "IMBIRIBEIRA", Fone: "(81) 0000-0000", Email: "tomador@exemplo.com.br",
		},
		Descricao:     "PRESTAÇÃO DE SERVIÇO PROCESSAMENTO DE DADOS. CODPEDIDOSAP: 4523396011 CHAVE PIX: 38149716000128 VENCIMENTO: 06/11/2026",
		ValorCentavos: 2400000,
	}
}

func lerFixture(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", "nfse_dps_nota1.xml"))
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(string(b))
}

// Matriz: DPS feliz + nome com & + complemento com espaços repetidos.
func TestNFSeGerarDPS_IdenticoNotaReal(t *testing.T) {
	g, err := GerarDPS(dadosNota1())
	if err != nil {
		t.Fatal(err)
	}
	if got, want := g.XMLNaoAssinado(), lerFixture(t); got != want {
		t.Fatalf("DPS difere da nota real.\n got: %s\nwant: %s", got, want)
	}
	if g.ID != "DPS520140523814971600012870000000000000000004" || len(g.ID) != 45 {
		t.Fatalf("Id inesperado: %q", g.ID)
	}
}

// Matriz: valor R$ 1,00.
func TestNFSeGerarDPS_UmReal(t *testing.T) {
	d := dadosNota1()
	d.ValorCentavos = 100
	g, err := GerarDPS(d)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(g.InfDPS, "<vServ>1.00</vServ>") {
		t.Fatalf("vServ esperado 1.00: %s", g.InfDPS)
	}
}

// Matriz: tomador sem IBGE.
func TestNFSeGerarDPS_TomadorSemIBGE(t *testing.T) {
	d := dadosNota1()
	d.Tomador.CodMun = ""
	if _, err := GerarDPS(d); err == nil || !strings.Contains(err.Error(), "IBGE") {
		t.Fatalf("esperava erro de IBGE, veio %v", err)
	}
}

func TestNFSeGerarDPS_Validacoes(t *testing.T) {
	casos := map[string]func(*DPSDados){
		"valor zero":    func(d *DPSDados) { d.ValorCentavos = 0 },
		"sem descrição": func(d *DPSDados) { d.Descricao = "  " },
		"ambiente":      func(d *DPSDados) { d.Ambiente = 3 },
		"nDPS":          func(d *DPSDados) { d.NDPS = 0 },
		"cnpj prest":    func(d *DPSDados) { d.Prestador.CNPJ = "123" },
	}
	for nome, mut := range casos {
		d := dadosNota1()
		mut(&d)
		if _, err := GerarDPS(d); err == nil {
			t.Errorf("%s: esperava erro", nome)
		}
	}
}

func TestNFSeEmpacotarDPS(t *testing.T) {
	g, _ := GerarDPS(dadosNota1())
	xml := g.XMLNaoAssinado()
	b64, err := EmpacotarDPS(xml)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := base64.StdEncoding.DecodeString(b64)
	zr, err := gzip.NewReader(strings.NewReader(string(raw)))
	if err != nil {
		t.Fatal(err)
	}
	out, _ := io.ReadAll(zr)
	if string(out) != xml {
		t.Fatal("round-trip gzip/base64 divergiu")
	}
}

func certificadoDeTeste(t *testing.T) *NFSeCertificadoA1 {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	tpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "FORTES BEZERRA TECNOLOGIA E CONSULTORIA LTDA:38149716000128"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
	}
	der, err := x509.CreateCertificate(rand.Reader, tpl, tpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	cert, _ := x509.ParseCertificate(der)
	return &NFSeCertificadoA1{Chave: key, Cert: cert}
}

// Matriz: assinatura verificável com o certificado.
func TestNFSeAssinarDPS_Verificavel(t *testing.T) {
	a1 := certificadoDeTeste(t)
	g, _ := GerarDPS(dadosNota1())
	xml, err := AssinarDPS(g, a1)
	if err != nil {
		t.Fatal(err)
	}

	if dir := os.Getenv("NFSE_DUMP_DIR"); dir != "" {
		_ = os.WriteFile(filepath.Join(dir, "dps_assinado.xml"), []byte(xml), 0o600)
		_ = os.WriteFile(filepath.Join(dir, "cert.der"), a1.Cert.Raw, 0o600)
	}

	digest := base64.StdEncoding.EncodeToString(func() []byte { h := sha256.Sum256([]byte(g.infDPSCanonico())); return h[:] }())
	if !strings.Contains(xml, "<DigestValue>"+digest+"</DigestValue>") {
		t.Fatal("digest não confere com o infDPS canônico")
	}
	if !strings.Contains(xml, "<Reference URI=\"#"+g.ID+"\">") {
		t.Fatal("Reference deve apontar para o Id do infDPS")
	}

	ini := strings.Index(xml, "<SignedInfo>")
	fim := strings.Index(xml, "</SignedInfo>") + len("</SignedInfo>")
	si := strings.Replace(xml[ini:fim], "<SignedInfo>", "<SignedInfo xmlns=\""+dsigNamespace+"\">", 1)
	ini = strings.Index(xml, "<SignatureValue>") + len("<SignatureValue>")
	sigB64 := xml[ini:strings.Index(xml, "</SignatureValue>")]
	sig, _ := base64.StdEncoding.DecodeString(sigB64)
	h := sha256.Sum256([]byte(si))
	if err := rsa.VerifyPKCS1v15(a1.Cert.PublicKey.(*rsa.PublicKey), crypto.SHA256, h[:], sig); err != nil {
		t.Fatalf("assinatura inválida: %v", err)
	}
	if CNPJDoCertificado(a1.Cert) != "38149716000128" {
		t.Fatal("CNPJ do certificado não extraído")
	}
}

func TestNFSeCarregarCertificadoA1_PFXInvalido(t *testing.T) {
	pfx, _ := EncryptConfig(base64.StdEncoding.EncodeToString([]byte("lixo")))
	senha, _ := EncryptConfig("x")
	if _, err := CarregarCertificadoA1(pfx, senha); err == nil {
		t.Fatal("esperava erro para PFX inválido")
	}
}

// Matriz: contadores concorrentes. Exige TEST_DATABASE_URL com a migration 125 aplicada.
func TestNFSeReservarNDPS_Concorrente(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL não definido")
	}
	db, err := sql.Open("postgres", url)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	var empresaID string
	if err := db.QueryRow(`INSERT INTO financeiro.empresas (razao_social, cnpj, logradouro, numero, bairro, cep, municipio, uf)
		VALUES ('TESTE', '00000000000191', 'RUA', '1', 'B', '74000000', 'X', 'GO') RETURNING id`).Scan(&empresaID); err != nil {
		t.Fatal(err)
	}
	defer db.Exec(`DELETE FROM financeiro.empresas WHERE id = $1`, empresaID)

	const n = 20
	var mu sync.Mutex
	vistos := map[int64]bool{}
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			v, err := ReservarNDPS(context.Background(), db, empresaID, "70000")
			if err != nil {
				t.Error(err)
				return
			}
			mu.Lock()
			defer mu.Unlock()
			if vistos[v] {
				t.Errorf("nDPS %d repetido", v)
			}
			vistos[v] = true
		}()
	}
	wg.Wait()
	for v := int64(101); v < 101+n; v++ {
		if !vistos[v] {
			t.Errorf("faltou nDPS %d (esperado 101..%d sem buracos)", v, 100+n)
		}
	}
}

func TestNFSeCifrarCertificadoA1_ExigeSegredo(t *testing.T) {
	t.Setenv("FINANCEIRO_SECRET", "")
	if _, _, err := CifrarCertificadoA1([]byte("x"), "y"); err == nil || !strings.Contains(err.Error(), "FINANCEIRO_SECRET") {
		t.Fatalf("esperava recusa sem segredo, veio %v", err)
	}
}

func TestNFSeCifrarCertificadoA1_RoundTrip(t *testing.T) {
	t.Setenv("FINANCEIRO_SECRET", "segredo-de-teste")
	a1 := certificadoDeTeste(t)
	pfx, err := pkcs12.Modern.Encode(a1.Chave, a1.Cert, nil, "senha123")
	if err != nil {
		t.Fatal(err)
	}
	pfxEnc, senhaEnc, err := CifrarCertificadoA1(pfx, "senha123")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(pfxEnc, base64.StdEncoding.EncodeToString(pfx)) || strings.HasPrefix(pfxEnc, "plain:") {
		t.Fatal("PFX guardado sem criptografia")
	}
	got, err := CarregarCertificadoA1(pfxEnc, senhaEnc)
	if err != nil {
		t.Fatal(err)
	}
	if CNPJDoCertificado(got.Cert) != "38149716000128" {
		t.Fatal("certificado carregado não confere")
	}
}
