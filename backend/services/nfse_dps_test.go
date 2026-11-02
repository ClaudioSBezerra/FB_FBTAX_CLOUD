package services

import (
	"compress/gzip"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"database/sql"
	"encoding/base64"
	"fmt"
	"io"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"software.sslmate.com/src/go-pkcs12"
)

// Digest SHA-256 (base64) do infDPS da nota nº 1 calculado FORA do código sob teste, pelo
// c14n exclusivo do lxml (libxml2). Se a serialização do gerador divergir da forma canônica,
// este teste quebra mesmo que o código e o teste concordem entre si.
const digestDourado = "74fki0N174/vtp4y5z7niaC/F5UjAtH0ZW+t9Mogw7A="

var brt = time.FixedZone("-03", -3*3600)

func dadosNota1() DPSDados {
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

func gerar(t *testing.T, d DPSDados) DPSGerado {
	t.Helper()
	g, err := GerarDPS(d)
	if err != nil {
		t.Fatal(err)
	}
	return g
}

// Matriz: DPS feliz + nome com & + complemento com espaços repetidos.
func TestNFSeGerarDPS_IdenticoNotaReal(t *testing.T) {
	g := gerar(t, dadosNota1())
	if got, want := g.XMLNaoAssinado(), lerFixture(t); got != want {
		t.Fatalf("DPS difere da nota real.\n got: %s\nwant: %s", got, want)
	}
	if g.ID != "DPS520140523814971600012870000000000000000004" || len(g.ID) != 45 {
		t.Fatalf("Id inesperado: %q", g.ID)
	}
}

func TestNFSeInfDPSCanonico_DigestIndependente(t *testing.T) {
	h := sha256.Sum256([]byte(gerar(t, dadosNota1()).infDPSCanonico()))
	if got := base64.StdEncoding.EncodeToString(h[:]); got != digestDourado {
		t.Fatalf("digest %s != dourado (lxml) %s", got, digestDourado)
	}
}

// Matriz: valor R$ 1,00.
func TestNFSeGerarDPS_UmReal(t *testing.T) {
	d := dadosNota1()
	d.ValorCentavos = 100
	if g := gerar(t, d); !strings.Contains(g.InfDPS, "<vServ>1.00</vServ>") {
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
	casos := []struct {
		nome string
		mut  func(*DPSDados)
		erro string
	}{
		{"valor zero", func(d *DPSDados) { d.ValorCentavos = 0 }, "valor do serviço"},
		{"sem descrição", func(d *DPSDados) { d.Descricao = "  " }, "descrição do serviço não informada"},
		{"ambiente", func(d *DPSDados) { d.Ambiente = 3 }, "ambiente"},
		{"nDPS", func(d *DPSDados) { d.NDPS = 0 }, "número do DPS"},
		{"cnpj prestador curto", func(d *DPSDados) { d.Prestador.CNPJ = "123" }, "CNPJ do prestador"},
		{"cnpj prestador DV", func(d *DPSDados) { d.Prestador.CNPJ = "38149716000129" }, "dígito verificador"},
		{"cnpj tomador DV", func(d *DPSDados) { d.Tomador.CNPJ = "10230480001961" }, "dígito verificador"},
		{"cnpj alfanumérico", func(d *DPSDados) { d.Tomador.CNPJ = "12ABC34501DE35" }, "alfanumérico"},
		{"série > 5", func(d *DPSDados) { d.Serie = "700001" }, "série"},
		{"descrição > 2000", func(d *DPSDados) { d.Descricao = strings.Repeat("A", 2001) }, "2000"},
		{"complemento > 156", func(d *DPSDados) { d.Tomador.Complemento = strings.Repeat("A", 157) }, "complemento"},
		{"competência futura", func(d *DPSDados) { d.Competencia = d.Emissao.Add(time.Hour) }, "competência posterior"},
		{"regime", func(d *DPSDados) { d.Prestador.OpSimpNac = 7 }, "regime"},
		{"pTotTribSN", func(d *DPSDados) { d.PTotTribSN = "6,0" }, "pTotTribSN"},
		{"telefone curto", func(d *DPSDados) { d.Tomador.Fone = "123" }, "telefone"},
	}
	for _, c := range casos {
		d := dadosNota1()
		c.mut(&d)
		_, err := GerarDPS(d)
		if err == nil || !strings.Contains(err.Error(), c.erro) {
			t.Errorf("%s: esperava erro com %q, veio %v", c.nome, c.erro, err)
		}
	}
}

func TestNFSeGerarDPS_CamposOpcionaisECPF(t *testing.T) {
	d := dadosNota1()
	d.Prestador.IM = "14544632"
	d.Prestador.Fone, d.Prestador.Email = "", ""
	d.Tomador.CNPJ = "529.982.247-25" // CPF válido
	d.Tomador.Complemento, d.Tomador.Fone, d.Tomador.Email = "", "", ""
	g := gerar(t, d)
	for _, esperado := range []string{
		"<prest><CNPJ>38149716000128</CNPJ><IM>14544632</IM><regTrib>",
		"<toma><CPF>52998224725</CPF>",
		"<xLgr>MARECHAL MASCARENHAS DE MORAES</xLgr><nro>2967</nro><xBairro>IMBIRIBEIRA</xBairro></end></toma>",
	} {
		if !strings.Contains(g.InfDPS, esperado) {
			t.Errorf("faltou %q em %s", esperado, g.InfDPS)
		}
	}
	if strings.Contains(g.InfDPS, "<xCpl>") || strings.Contains(g.InfDPS, "<fone>") {
		t.Error("opcionais vazios não devem ser emitidos")
	}
}

func TestNFSeXMLTexto_DescartaCaracteresIlegais(t *testing.T) {
	if got := xmlTexto("A\x00B\x0bC&D<E>\u0001F"); got != "ABC&amp;D&lt;E&gt;F" {
		t.Fatalf("saída inesperada: %q", got)
	}
}

func TestNFSeIDDPS_Entradas(t *testing.T) {
	if IDDPS("123", "38149716000128", "70000", 1) != "" || IDDPS("5201405", "38149716000128", "70000", 0) != "" {
		t.Fatal("IDDPS deve recusar entradas inválidas")
	}
}

func TestNFSeCNPJeCPFValidos(t *testing.T) {
	if !CNPJValido("38149716000128") || !CNPJValido("10230480001960") || CNPJValido("11111111111111") || CNPJValido("38149716000129") {
		t.Fatal("CNPJValido incorreto")
	}
	if !CPFValido("52998224725") || CPFValido("52998224726") || CPFValido("11111111111") {
		t.Fatal("CPFValido incorreto")
	}
}

func TestNFSeEmpacotarDPS(t *testing.T) {
	xml := gerar(t, dadosNota1()).XMLNaoAssinado()
	b64, err := EmpacotarDPS(xml)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		t.Fatal(err)
	}
	zr, err := gzip.NewReader(strings.NewReader(string(raw)))
	if err != nil {
		t.Fatal(err)
	}
	out, err := io.ReadAll(zr)
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != xml {
		t.Fatal("round-trip gzip/base64 divergiu")
	}
}

func certificadoDeTeste(t *testing.T, cn string, notBefore, notAfter time.Time) *NFSeCertificadoA1 {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	tpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: cn},
		NotBefore:    notBefore,
		NotAfter:     notAfter,
	}
	der, err := x509.CreateCertificate(rand.Reader, tpl, tpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	return &NFSeCertificadoA1{Chave: key, Cert: cert}
}

func certValido(t *testing.T) *NFSeCertificadoA1 {
	return certificadoDeTeste(t, "FORTES BEZERRA TECNOLOGIA E CONSULTORIA LTDA:38149716000128", time.Now().Add(-time.Hour), time.Now().Add(24*time.Hour))
}

// extrairAssinatura devolve o SignedInfo como o verificador o canonicaliza e a assinatura.
func extrairAssinatura(t *testing.T, xml string) (signedInfo string, sig []byte) {
	t.Helper()
	ini := strings.Index(xml, "<SignedInfo>")
	fim := strings.Index(xml, "</SignedInfo>") + len("</SignedInfo>")
	signedInfo = strings.Replace(xml[ini:fim], "<SignedInfo>", "<SignedInfo xmlns=\""+dsigNamespace+"\">", 1)
	i := strings.Index(xml, "<SignatureValue>") + len("<SignatureValue>")
	var err error
	sig, err = base64.StdEncoding.DecodeString(xml[i:strings.Index(xml, "</SignatureValue>")])
	if err != nil {
		t.Fatal(err)
	}
	return
}

// Matriz: assinatura verificável com o certificado (RSA-SHA256 sobre o SignedInfo canônico).
func TestNFSeAssinarDPS_Verificavel(t *testing.T) {
	a1 := certValido(t)
	g := gerar(t, dadosNota1())
	xml, err := AssinarDPS(g, a1)
	if err != nil {
		t.Fatal(err)
	}
	h := sha256.Sum256([]byte(g.infDPSCanonico()))
	if !strings.Contains(xml, "<DigestValue>"+base64.StdEncoding.EncodeToString(h[:])+"</DigestValue>") {
		t.Fatal("digest não confere com o infDPS canônico")
	}
	if !strings.Contains(xml, "<Reference URI=\"#"+g.ID+"\">") {
		t.Fatal("Reference deve apontar para o Id do infDPS")
	}
	si, sig := extrairAssinatura(t, xml)
	hs := sha256.Sum256([]byte(si))
	if err := rsa.VerifyPKCS1v15(a1.Cert.PublicKey.(*rsa.PublicKey), crypto.SHA256, hs[:], sig); err != nil {
		t.Fatalf("assinatura inválida: %v", err)
	}
	// Adulterar o SignedInfo (ex.: trocar o digest) deve invalidar a assinatura.
	adulterado := strings.Replace(si, "<DigestValue>", "<DigestValue>A", 1)
	ha := sha256.Sum256([]byte(adulterado))
	if rsa.VerifyPKCS1v15(a1.Cert.PublicKey.(*rsa.PublicKey), crypto.SHA256, ha[:], sig) == nil {
		t.Fatal("assinatura aceitou SignedInfo adulterado")
	}
	if !strings.HasPrefix(xml, "<DPS xmlns=") || !strings.HasSuffix(xml, "</Signature></DPS>") {
		t.Fatal("estrutura do DPS assinado inesperada")
	}
}

func TestNFSeAssinarDPS_RecusaCertificadoErrado(t *testing.T) {
	g := gerar(t, dadosNota1())
	casos := map[string]*NFSeCertificadoA1{
		"outro CNPJ": certificadoDeTeste(t, "OUTRA EMPRESA:00000000000191", time.Now().Add(-time.Hour), time.Now().Add(time.Hour)),
		"vencido":    certificadoDeTeste(t, "X:38149716000128", time.Now().Add(-48*time.Hour), time.Now().Add(-time.Hour)),
		"sem CNPJ":   certificadoDeTeste(t, "SEM CNPJ NO CN", time.Now().Add(-time.Hour), time.Now().Add(time.Hour)),
	}
	for nome, a1 := range casos {
		if _, err := AssinarDPS(g, a1); err == nil {
			t.Errorf("%s: esperava recusa", nome)
		}
	}
	if _, err := AssinarDPS(DPSGerado{}, certValido(t)); err == nil {
		t.Error("DPS vazio deve ser recusado")
	}
}

func TestNFSeCarregarCertificadoA1_Invalidos(t *testing.T) {
	t.Setenv("FINANCEIRO_SECRET", "segredo-de-teste")
	pfx, _ := EncryptConfig(base64.StdEncoding.EncodeToString([]byte("lixo")))
	senha, _ := EncryptConfig("x")
	if _, err := CarregarCertificadoA1(pfx, senha); err == nil {
		t.Fatal("esperava erro para PFX inválido")
	}
	if _, err := CarregarCertificadoA1("plain:abc", senha); err == nil || !strings.Contains(err.Error(), "sem criptografia") {
		t.Fatalf("esperava recusa de linha em texto claro, veio %v", err)
	}
}

func TestNFSeCifrarCertificadoA1_ExigeSegredo(t *testing.T) {
	t.Setenv("FINANCEIRO_SECRET", "")
	if _, _, _, err := CifrarCertificadoA1([]byte("x"), "y"); err == nil || !strings.Contains(err.Error(), "FINANCEIRO_SECRET") {
		t.Fatalf("esperava recusa sem segredo, veio %v", err)
	}
}

func TestNFSeCifrarCertificadoA1_RoundTripEValidade(t *testing.T) {
	t.Setenv("FINANCEIRO_SECRET", "segredo-de-teste")
	a1 := certValido(t)
	pfx, err := pkcs12.Modern.Encode(a1.Chave, a1.Cert, nil, "senha123")
	if err != nil {
		t.Fatal(err)
	}
	pfxEnc, senhaEnc, validade, err := CifrarCertificadoA1(pfx, "senha123")
	if err != nil {
		t.Fatal(err)
	}
	if !validade.Equal(a1.Cert.NotAfter) {
		t.Fatalf("validade %v != NotAfter %v", validade, a1.Cert.NotAfter)
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

	vencido := certificadoDeTeste(t, "X:38149716000128", time.Now().Add(-48*time.Hour), time.Now().Add(-time.Hour))
	pfxV, err := pkcs12.Modern.Encode(vencido.Chave, vencido.Cert, nil, "s")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := CifrarCertificadoA1(pfxV, "s"); err == nil || !strings.Contains(err.Error(), "vencido") {
		t.Fatalf("esperava recusa de certificado vencido, veio %v", err)
	}
}

func TestNFSeCNPJDoCertificado_Ancorado(t *testing.T) {
	c := func(cn string) *x509.Certificate { return &x509.Certificate{Subject: pkix.Name{CommonName: cn}} }
	if CNPJDoCertificado(c("EMPRESA 12345678901234 LTDA:38149716000128")) != "38149716000128" {
		t.Fatal("deveria pegar o CNPJ após os dois pontos")
	}
	if CNPJDoCertificado(c("EMPRESA 12345678901234 LTDA")) != "" {
		t.Fatal("sem sufixo :CNPJ deve devolver vazio")
	}
}

func TestNFSeParseRespostaSefin(t *testing.T) {
	xml := "<NFSe>ok</NFSe>"
	b64, err := EmpacotarDPS(xml)
	if err != nil {
		t.Fatal(err)
	}
	r := ParseRespostaSefin(201, []byte(fmt.Sprintf(`{"chaveAcesso":"123","idDps":"DPS1","nfseXmlGZipB64":%q}`, b64)))
	if !r.Autorizada() || r.ChaveAcesso != "123" || r.NFSeXML != xml {
		t.Fatalf("resposta autorizada mal interpretada: %+v", r)
	}
	r = ParseRespostaSefin(400, []byte(`{"erros":[{"codigo":"E0001","descricao":"falhou","complemento":"detalhe"}]}`))
	if r.Autorizada() || len(r.Erros) != 1 || !strings.Contains(r.MensagemErro(), "E0001") {
		t.Fatalf("rejeição mal interpretada: %+v", r)
	}
	if r = ParseRespostaSefin(500, []byte("<html>")); r.Autorizada() || !strings.Contains(r.MensagemErro(), "HTTP 500") {
		t.Fatal("erro HTTP sem JSON deve ser relatado")
	}
}

func TestNFSePedidoCancelamento(t *testing.T) {
	chave := "52014052238149716000128000000000000126119042182716"
	p, err := NovoPedidoCancelamento(1, "38.149.716/0001-28", chave, 1, "Nota de teste emitida para validação do sistema", time.Date(2026, 11, 2, 15, 0, 0, 0, brt))
	if err != nil {
		t.Fatal(err)
	}
	if p.ID != "PRE"+chave+"101101" || len(p.ID) != 59 {
		t.Fatalf("Id do pedido inesperado: %s", p.ID)
	}
	xml, err := AssinarPedidoCancelamento(p, certValido(t))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(xml, "<e101101><xDesc>Cancelamento de NFS-e</xDesc><cMotivo>1</cMotivo>") || !strings.Contains(xml, "<Reference URI=\"#"+p.ID+"\">") {
		t.Fatalf("pedido mal formado: %s", xml)
	}
	for nome, f := range map[string]func() error{
		"justificativa curta": func() error {
			_, e := NovoPedidoCancelamento(1, "38149716000128", chave, 1, "curta", time.Now())
			return e
		},
		"chave curta": func() error {
			_, e := NovoPedidoCancelamento(1, "38149716000128", "123", 1, strings.Repeat("x", 20), time.Now())
			return e
		},
		"motivo": func() error {
			_, e := NovoPedidoCancelamento(1, "38149716000128", chave, 5, strings.Repeat("x", 20), time.Now())
			return e
		},
	} {
		if f() == nil {
			t.Errorf("%s: esperava erro", nome)
		}
	}
}

func TestNFSeNormalizarSerieDPS(t *testing.T) {
	for in, want := range map[string]string{"": "70000", "70000": "70000", " 70000": "70000", "7": "00007", "070000": ""} {
		got, err := NormalizarSerieDPS(in)
		if want == "" && err == nil || want != "" && (err != nil || got != want) {
			t.Errorf("série %q: got %q err %v, want %q", in, got, err, want)
		}
	}
}

// Matriz: contadores concorrentes. Exige TEST_DATABASE_URL com a migration 125 aplicada
// (make test-nfse-db sobe um banco descartável e roda este teste).
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

	cnpj := fmt.Sprintf("%014d", time.Now().UnixNano()%100000000000000)
	var empresaID string
	if err := db.QueryRow(`INSERT INTO financeiro.empresas (razao_social, cnpj, logradouro, numero, bairro, cep, municipio, uf)
		VALUES ('TESTE', $1, 'RUA', '1', 'B', '74000000', 'X', 'GO') RETURNING id`, cnpj).Scan(&empresaID); err != nil {
		t.Fatal(err)
	}
	defer func() {
		db.Exec(`DELETE FROM financeiro.nfse_dps_contador WHERE empresa_id = $1`, empresaID)
		if _, err := db.Exec(`DELETE FROM financeiro.empresas WHERE id = $1`, empresaID); err != nil {
			t.Errorf("limpeza: %v", err)
		}
	}()

	const n = 20
	var mu sync.Mutex
	vistos := map[int64]bool{}
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			v, err := ReservarNDPS(context.Background(), db, empresaID, 1, " 70000")
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
	// Outra série começa do 1 e outro ambiente tem contador próprio.
	if v, err := ReservarNDPS(context.Background(), db, empresaID, 1, "70001"); err != nil || v != 1 {
		t.Errorf("série nova deveria começar em 1: %d %v", v, err)
	}
	if v, err := ReservarNDPS(context.Background(), db, empresaID, 2, "70000"); err != nil || v != 101 {
		t.Errorf("ambiente 2 deveria ter contador próprio (101): %d %v", v, err)
	}
}
