package services

// Cliente da API Sefin Nacional (Sistema Nacional NFS-e): emissão síncrona (POST /nfse),
// consulta (GET /nfse/{chave}) e cancelamento por evento (POST /nfse/{chave}/eventos).
// Autenticação por mTLS com o A1 do prestador. Manual: "Guia para utilização das API's do
// Emissor Público Nacional" (gov.br/nfse). O formato exato das respostas é tratado de forma
// tolerante e o corpo bruto é sempre devolvido para ser guardado em nfse_emitidas.erro.

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	SefinBaseProducao         = "https://sefin.nfse.gov.br/SefinNacional"
	SefinBaseProducaoRestrita = "https://sefin.producaorestrita.nfse.gov.br/API/SefinNacional"

	// API de parametrização (parâmetros municipais e convênio) fica no ADN, não na Sefin.
	ParametrizacaoBaseProducao         = "https://adn.nfse.gov.br/parametrizacao"
	ParametrizacaoBaseProducaoRestrita = "https://adn.producaorestrita.nfse.gov.br/parametrizacao"

	sefinTimeout   = 40 * time.Second
	sefinMaxCorpo  = 4 << 20
	nfseCaFileEnv  = "NFSE_TLS_CA_FILE"
	NFSeMotivoErro = 1 // cMotivo: 1 = erro na emissão
)

// SefinBaseURL devolve a base da API conforme tpAmb (1 produção, 2 produção restrita).
func SefinBaseURL(ambiente int) (string, error) {
	switch ambiente {
	case 1:
		return SefinBaseProducao, nil
	case 2:
		return SefinBaseProducaoRestrita, nil
	}
	return "", fmt.Errorf("ambiente inválido: %d", ambiente)
}

// ClienteSefin fala com a API em um ambiente, autenticado com o A1.
type ClienteSefin struct {
	Base      string
	BaseParam string // API de parametrização (ADN)
	HTTP      *http.Client
}

// NovoClienteSefin monta o cliente mTLS. NFSE_TLS_CA_FILE (opcional) acrescenta CAs ao pool do
// sistema, necessário quando a imagem não traz a cadeia ICP-Brasil dos servidores do governo.
func NovoClienteSefin(ambiente int, a1 *NFSeCertificadoA1) (*ClienteSefin, error) {
	base, err := SefinBaseURL(ambiente)
	if err != nil {
		return nil, err
	}
	pool, err := x509.SystemCertPool()
	if err != nil || pool == nil {
		pool = x509.NewCertPool()
	}
	if f := os.Getenv(nfseCaFileEnv); f != "" {
		pem, err := os.ReadFile(f)
		if err != nil {
			return nil, fmt.Errorf("ler %s: %w", nfseCaFileEnv, err)
		}
		if !pool.AppendCertsFromPEM(pem) {
			return nil, fmt.Errorf("%s não contém certificados PEM válidos", nfseCaFileEnv)
		}
	}
	cadeia := [][]byte{a1.Cert.Raw}
	for _, ac := range a1.Cadeia {
		cadeia = append(cadeia, ac.Raw)
	}
	tlsCert := tls.Certificate{Certificate: cadeia, PrivateKey: a1.Chave, Leaf: a1.Cert}
	// GetClientCertificate (não Certificates): o servidor lista só as raízes ICP-Brasil como CAs
	// aceitas, e o Go não envia o certificado quando o emissor dele não casa com a lista. Aqui o
	// certificado, com a cadeia completada no upload, é sempre enviado e o servidor decide.
	tr := &http.Transport{TLSClientConfig: &tls.Config{
		MinVersion: tls.VersionTLS12,
		RootCAs:    pool,
		// O IIS da Sefin negocia TLS 1.2 e só pede o certificado de cliente depois, por
		// renegociação iniciada pelo servidor; o Go a recusa por padrão ("tls: no renegotiation").
		// Uma renegociação por conexão basta. Não existe em TLS 1.3, só no 1.2.
		Renegotiation:        tls.RenegotiateOnceAsClient,
		GetClientCertificate: func(*tls.CertificateRequestInfo) (*tls.Certificate, error) { return &tlsCert, nil },
	}}
	baseParam := ParametrizacaoBaseProducaoRestrita
	if ambiente == 1 {
		baseParam = ParametrizacaoBaseProducao
	}
	return &ClienteSefin{Base: base, BaseParam: baseParam, HTTP: &http.Client{Transport: tr, Timeout: sefinTimeout}}, nil
}

// SefinErro é uma rejeição da API (código e descrição).
type SefinErro struct {
	Codigo    string
	Descricao string
}

// RespostaSefin é o resultado normalizado de uma chamada.
type RespostaSefin struct {
	Status      int
	Corpo       string // corpo bruto (limitado)
	ChaveAcesso string
	NFSeXML     string // XML da NFS-e já descompactado, quando vier
	IDDPS       string
	Erros       []SefinErro
}

// Autorizada indica HTTP 2xx sem erros.
func (r *RespostaSefin) Autorizada() bool {
	return r.Status >= 200 && r.Status < 300 && len(r.Erros) == 0
}

// MensagemErro resume as rejeições para exibição e gravação.
func (r *RespostaSefin) MensagemErro() string {
	if len(r.Erros) == 0 {
		if r.Autorizada() {
			return ""
		}
		return fmt.Sprintf("HTTP %d: %s", r.Status, truncarTexto(r.Corpo, 500))
	}
	var p []string
	for _, e := range r.Erros {
		p = append(p, strings.TrimSpace(e.Codigo+" "+e.Descricao))
	}
	return fmt.Sprintf("HTTP %d: %s", r.Status, strings.Join(p, " | "))
}

// EmitirNFSe envia o DPS assinado (POST /nfse).
func (c *ClienteSefin) EmitirNFSe(ctx context.Context, dpsAssinadoXML string) (*RespostaSefin, error) {
	b64, err := EmpacotarDPS(dpsAssinadoXML)
	if err != nil {
		return nil, err
	}
	return c.postJSON(ctx, "/nfse", map[string]string{"dpsXmlGZipB64": b64})
}

// ConsultarNFSe busca uma NFS-e pela chave de acesso (GET /nfse/{chave}).
func (c *ClienteSefin) ConsultarNFSe(ctx context.Context, chave string) (*RespostaSefin, error) {
	return c.fazer(ctx, http.MethodGet, "/nfse/"+chave, nil)
}

// CancelarNFSe registra o evento e101101 (POST /nfse/{chave}/eventos). pedidoAssinado é o XML
// montado por PedidoCancelamento + AssinarPedidoCancelamento.
func (c *ClienteSefin) CancelarNFSe(ctx context.Context, chave, pedidoAssinado string) (*RespostaSefin, error) {
	b64, err := EmpacotarDPS(pedidoAssinado)
	if err != nil {
		return nil, err
	}
	return c.postJSON(ctx, "/nfse/"+chave+"/eventos", map[string]string{"pedidoRegistroEventoXmlGZipB64": b64})
}

func (c *ClienteSefin) postJSON(ctx context.Context, caminho string, corpo any) (*RespostaSefin, error) {
	b, err := json.Marshal(corpo)
	if err != nil {
		return nil, err
	}
	return c.fazer(ctx, http.MethodPost, caminho, b)
}

func (c *ClienteSefin) fazer(ctx context.Context, metodo, caminho string, corpo []byte) (*RespostaSefin, error) {
	return c.fazerEm(ctx, c.Base, metodo, caminho, corpo)
}

func (c *ClienteSefin) fazerEm(ctx context.Context, base, metodo, caminho string, corpo []byte) (*RespostaSefin, error) {
	req, err := http.NewRequestWithContext(ctx, metodo, base+caminho, bytes.NewReader(corpo))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	if corpo != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sefin: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, sefinMaxCorpo))
	if err != nil {
		return nil, fmt.Errorf("sefin: ler resposta: %w", err)
	}
	return ParseRespostaSefin(resp.StatusCode, raw), nil
}

// ParseRespostaSefin extrai, de forma tolerante, chave, XML e erros do JSON da Sefin.
func ParseRespostaSefin(status int, raw []byte) *RespostaSefin {
	r := &RespostaSefin{Status: status, Corpo: string(raw)}
	var m map[string]any
	if json.Unmarshal(raw, &m) != nil {
		return r
	}
	get := func(chaves ...string) any {
		for k, v := range m {
			for _, want := range chaves {
				if strings.EqualFold(k, want) {
					return v
				}
			}
		}
		return nil
	}
	if s, ok := get("chaveAcesso").(string); ok {
		r.ChaveAcesso = s
	}
	if s, ok := get("idDps", "idDPS").(string); ok {
		r.IDDPS = s
	}
	if s, ok := get("nfseXmlGZipB64").(string); ok && s != "" {
		if x, err := DescompactarXML(s); err == nil {
			r.NFSeXML = x
		}
	}
	for _, chave := range []string{"erros", "erro", "mensagens"} {
		lista, ok := get(chave).([]any)
		if !ok {
			continue
		}
		for _, it := range lista {
			if e, ok := it.(map[string]any); ok {
				r.Erros = append(r.Erros, SefinErro{Codigo: strVal(e, "codigo"), Descricao: strings.TrimSpace(strVal(e, "descricao") + " " + strVal(e, "complemento"))})
			}
		}
	}
	return r
}

func strVal(m map[string]any, k string) string {
	for kk, v := range m {
		if strings.EqualFold(kk, k) {
			if s, ok := v.(string); ok {
				return s
			}
		}
	}
	return ""
}

// DescompactarXML desfaz base64 + gzip.
func DescompactarXML(b64 string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return "", err
	}
	zr, err := gzip.NewReader(bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	defer zr.Close()
	out, err := io.ReadAll(io.LimitReader(zr, sefinMaxCorpo))
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// PedidoCancelamento é o pedido de registro do evento e101101 pronto para assinar.
type PedidoCancelamento struct {
	ID      string
	InfPed  string
	Versao  string
	DhEvent time.Time
}

// NovoPedidoCancelamento monta o infPedReg do evento de cancelamento.
func NovoPedidoCancelamento(ambiente int, cnpjAutor, chave string, cMotivo int, xMotivo string, quando time.Time) (PedidoCancelamento, error) {
	cnpj := soDigitos(cnpjAutor)
	xm := normalizarTexto(xMotivo)
	switch {
	case len(cnpj) != 14:
		return PedidoCancelamento{}, errors.New("CNPJ do autor inválido")
	case len(chave) != 50 || soDigitos(chave) != chave:
		return PedidoCancelamento{}, errors.New("chave de acesso deve ter 50 dígitos")
	case ambiente != 1 && ambiente != 2:
		return PedidoCancelamento{}, errors.New("ambiente inválido")
	case cMotivo != 1 && cMotivo != 2 && cMotivo != 9:
		return PedidoCancelamento{}, errors.New("código de motivo inválido (1, 2 ou 9)")
	}
	if n := len([]rune(xm)); n < 15 || n > 255 {
		return PedidoCancelamento{}, errors.New("a justificativa deve ter entre 15 e 255 caracteres")
	}
	id := "PRE" + chave + "101101"
	inf := "<infPedReg Id=\"" + id + "\">" +
		elemTxt("tpAmb", fmt.Sprint(ambiente)) +
		elemTxt("verAplic", NFSeVerAplic) +
		elemTxt("dhEvento", quando.Format("2006-01-02T15:04:05-07:00")) +
		elemTxt("CNPJAutor", cnpj) +
		elemTxt("chNFSe", chave) +
		elem("e101101",
			elemTxt("xDesc", "Cancelamento de NFS-e")+
				elemTxt("cMotivo", fmt.Sprint(cMotivo))+
				elemTxt("xMotivo", xm)) +
		"</infPedReg>"
	return PedidoCancelamento{ID: id, InfPed: inf, Versao: NFSeVersaoDPS, DhEvent: quando}, nil
}

// AssinarPedidoCancelamento devolve o <pedRegEvento> assinado.
func AssinarPedidoCancelamento(p PedidoCancelamento, a1 *NFSeCertificadoA1) (string, error) {
	canon := strings.Replace(p.InfPed, "<infPedReg Id=", "<infPedReg xmlns=\""+NFSeNamespace+"\" Id=", 1)
	sig, err := assinarXMLDSig(p.ID, canon, a1)
	if err != nil {
		return "", err
	}
	return XMLDeclaracao + "<pedRegEvento xmlns=\"" + NFSeNamespace + "\" versao=\"" + p.Versao + "\">" + p.InfPed + sig + "</pedRegEvento>", nil
}

func truncarTexto(s string, n int) string {
	if r := []rune(s); len(r) > n {
		return string(r[:n]) + "…"
	}
	return s
}

// ConsultarDPS recupera a chave de acesso a partir do Id do DPS (GET /dps/{id}). Só responde
// ao prestador, tomador ou intermediário da nota (o certificado da conexão é conferido).
func (c *ClienteSefin) ConsultarDPS(ctx context.Context, idDPS string) (*RespostaSefin, error) {
	return c.fazer(ctx, http.MethodGet, "/dps/"+idDPS, nil)
}

// ConsultarConvenio lê o cadastro de convênio do município no ambiente (GET
// /parametros_municipais/{cod}/convenio). Somente leitura: serve para saber se o município
// está habilitado a emitir pelo padrão nacional naquele ambiente.
func (c *ClienteSefin) ConsultarConvenio(ctx context.Context, codMunicipio string) (*RespostaSefin, error) {
	if len(codMunicipio) != 7 || soDigitos(codMunicipio) != codMunicipio {
		return nil, errors.New("código IBGE deve ter 7 dígitos")
	}
	return c.fazerEm(ctx, c.BaseParam, http.MethodGet, "/parametros_municipais/"+codMunicipio+"/convenio", nil)
}
