package services

// Gerador do DPS (Declaração de Prestação de Serviços) v1.01 do Sistema Nacional NFS-e.
//
// O leiaute segue o <DPS> das NFS-e reais emitidas pelo Emissor Web nacional para o
// prestador Fortes Bezerra (Simples Nacional, serviço 17.01.01). O XML é montado à mão,
// sem espaços e sem elementos vazios auto-fechados, para que a serialização já seja a
// forma canônica usada na assinatura XMLDSig (ver nfse_assinatura.go).

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	NFSeNamespace   = "http://www.sped.fazenda.gov.br/nfse"
	NFSeVersaoDPS   = "1.01"
	NFSeVerAplic    = "FBTAX-1.0"
	nfseSeriepadrao = "70000"

	// Serviço fixo do prestador: assessoria ou consultoria (LC 116, 17.01.01).
	nfseCTribNac = "170101"
	nfseCNBS     = "114013900"
	nfseCIndOp   = "100301"
)

// DPSPrestador reúne os dados fiscais do emitente (financeiro.empresas).
type DPSPrestador struct {
	CNPJ        string
	IM          string // opcional: o padrão nacional dispensa para o Simples Nacional
	Fone        string
	Email       string
	OpSimpNac   int // 3 = ME/EPP optante
	RegApTribSN int // 1 = tributos federais e municipal pelo SN
	RegEspTrib  int
}

// DPSTomador reúne os dados do contratante (financeiro.clientes).
type DPSTomador struct {
	CNPJ        string // 14 dígitos (CPF com 11 também é aceito)
	Nome        string
	CodMun      string // IBGE, 7 dígitos
	CEP         string
	Logradouro  string
	Numero      string
	Complemento string
	Bairro      string
	Fone        string
	Email       string
}

// DPSDados é a entrada do gerador. Valor está em centavos.
type DPSDados struct {
	Ambiente        int // tpAmb: 1 produção, 2 produção restrita
	Emissao         time.Time
	Serie           string // 5 dígitos; vazio = 70000
	NDPS            int64
	Competencia     time.Time
	CodMunEmissor   string // cLocEmi
	CodMunPrestacao string // cLocPrestacao (município do tomador, como nas notas reais)
	Prestador       DPSPrestador
	Tomador         DPSTomador
	Descricao       string
	ValorCentavos   int64
	PTotTribSN      string // ex.: "6.00"
}

// DPSGerado é o DPS pronto para assinar.
type DPSGerado struct {
	ID      string // Id do infDPS (45 caracteres)
	InfDPS  string // infDPS em forma canônica, sem declaração de namespace
	Versao  string
	Serie   string
	NDPS    int64
	DataXML time.Time
}

// xmlTexto escapa texto de elemento como a canonicalização C14N manda: só & < > e CR.
func xmlTexto(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '&':
			b.WriteString("&amp;")
		case '<':
			b.WriteString("&lt;")
		case '>':
			b.WriteString("&gt;")
		case '\r':
			b.WriteString("&#xD;")
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// xmlAttr escapa valor de atributo na forma C14N.
func xmlAttr(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '&':
			b.WriteString("&amp;")
		case '<':
			b.WriteString("&lt;")
		case '"':
			b.WriteString("&quot;")
		case '\t':
			b.WriteString("&#x9;")
		case '\n':
			b.WriteString("&#xA;")
		case '\r':
			b.WriteString("&#xD;")
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func elem(nome, conteudo string) string {
	return "<" + nome + ">" + conteudo + "</" + nome + ">"
}

func elemTxt(nome, valor string) string { return elem(nome, xmlTexto(valor)) }

// elemOpc devolve vazio quando o valor está em branco (campo opcional do leiaute).
func elemOpc(nome, valor string) string {
	if strings.TrimSpace(valor) == "" {
		return ""
	}
	return elemTxt(nome, valor)
}

// normalizarTexto apara e colapsa espaços repetidos (o endereço dos clientes vem do ERP com
// campos preenchidos à direita por espaços).
func normalizarTexto(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func soDigitos(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func formatarValor(centavos int64) string {
	return fmt.Sprintf("%d.%02d", centavos/100, centavos%100)
}

func tpInscricao(doc string) string {
	if len(doc) == 14 {
		return "2"
	}
	return "1"
}

// IDDPS monta o identificador do DPS: DPS + cLocEmi(7) + tpInsc(1) + CNPJ(14) + série(5) + nDPS(15).
func IDDPS(codMunEmissor, cnpj, serie string, nDPS int64) string {
	return fmt.Sprintf("DPS%s%s%014s%05s%015d", codMunEmissor, tpInscricao(cnpj), cnpj, serie, nDPS)
}

func (d *DPSDados) validar() error {
	var erros []string
	if d.Ambiente != 1 && d.Ambiente != 2 {
		erros = append(erros, "ambiente deve ser 1 ou 2")
	}
	if len(soDigitos(d.Prestador.CNPJ)) != 14 {
		erros = append(erros, "CNPJ do prestador inválido")
	}
	if doc := soDigitos(d.Tomador.CNPJ); len(doc) != 14 && len(doc) != 11 {
		erros = append(erros, "CNPJ/CPF do tomador inválido")
	}
	if len(soDigitos(d.CodMunEmissor)) != 7 {
		erros = append(erros, "código IBGE do município emissor inválido")
	}
	if len(soDigitos(d.Tomador.CodMun)) != 7 {
		erros = append(erros, "código IBGE do município do tomador não informado")
	}
	if len(soDigitos(d.CodMunPrestacao)) != 7 {
		erros = append(erros, "código IBGE do local da prestação inválido")
	}
	if len(soDigitos(d.Tomador.CEP)) != 8 {
		erros = append(erros, "CEP do tomador inválido")
	}
	if normalizarTexto(d.Tomador.Nome) == "" {
		erros = append(erros, "nome do tomador não informado")
	}
	if normalizarTexto(d.Tomador.Logradouro) == "" || normalizarTexto(d.Tomador.Numero) == "" || normalizarTexto(d.Tomador.Bairro) == "" {
		erros = append(erros, "endereço do tomador incompleto (logradouro, número e bairro)")
	}
	if d.NDPS <= 0 {
		erros = append(erros, "número do DPS inválido")
	}
	if d.ValorCentavos <= 0 {
		erros = append(erros, "valor do serviço deve ser maior que zero")
	}
	if desc := normalizarTexto(d.Descricao); desc == "" {
		erros = append(erros, "descrição do serviço não informada")
	} else if utf8.RuneCountInString(desc) > 2000 {
		erros = append(erros, "descrição do serviço excede 2000 caracteres")
	}
	if d.Competencia.IsZero() || d.Emissao.IsZero() {
		erros = append(erros, "datas de emissão e competência obrigatórias")
	}
	if len(erros) > 0 {
		return errors.New("DPS inválido: " + strings.Join(erros, "; "))
	}
	return nil
}

// GerarDPS valida os dados e devolve o infDPS canônico e o Id.
func GerarDPS(d DPSDados) (DPSGerado, error) {
	if err := d.validar(); err != nil {
		return DPSGerado{}, err
	}
	serie := soDigitos(d.Serie)
	if serie == "" {
		serie = nfseSeriepadrao
	}
	if len(serie) > 5 {
		return DPSGerado{}, errors.New("DPS inválido: série excede 5 dígitos")
	}
	serie = fmt.Sprintf("%05s", serie)

	cnpj := soDigitos(d.Prestador.CNPJ)
	docTomador := soDigitos(d.Tomador.CNPJ)
	tagTomador := "CNPJ"
	if len(docTomador) == 11 {
		tagTomador = "CPF"
	}
	ptot := d.PTotTribSN
	if ptot == "" {
		ptot = "6.00"
	}
	id := IDDPS(soDigitos(d.CodMunEmissor), cnpj, serie, d.NDPS)

	prest := elem("prest",
		elemTxt("CNPJ", cnpj)+
			elemOpc("IM", soDigitos(d.Prestador.IM))+
			elemOpc("fone", soDigitos(d.Prestador.Fone))+
			elemOpc("email", d.Prestador.Email)+
			elem("regTrib",
				elemTxt("opSimpNac", fmt.Sprint(d.Prestador.OpSimpNac))+
					elemTxt("regApTribSN", fmt.Sprint(d.Prestador.RegApTribSN))+
					elemTxt("regEspTrib", fmt.Sprint(d.Prestador.RegEspTrib))))

	toma := elem("toma",
		elemTxt(tagTomador, docTomador)+
			elemTxt("xNome", normalizarTexto(d.Tomador.Nome))+
			elem("end",
				elem("endNac",
					elemTxt("cMun", soDigitos(d.Tomador.CodMun))+
						elemTxt("CEP", soDigitos(d.Tomador.CEP)))+
					elemTxt("xLgr", normalizarTexto(d.Tomador.Logradouro))+
					elemTxt("nro", normalizarTexto(d.Tomador.Numero))+
					elemOpc("xCpl", normalizarTexto(d.Tomador.Complemento))+
					elemTxt("xBairro", normalizarTexto(d.Tomador.Bairro)))+
			elemOpc("fone", soDigitos(d.Tomador.Fone))+
			elemOpc("email", strings.TrimSpace(d.Tomador.Email)))

	serv := elem("serv",
		elem("locPrest", elemTxt("cLocPrestacao", soDigitos(d.CodMunPrestacao)))+
			elem("cServ",
				elemTxt("cTribNac", nfseCTribNac)+
					elemTxt("xDescServ", normalizarTexto(d.Descricao))+
					elemTxt("cNBS", nfseCNBS)))

	valores := elem("valores",
		elem("vServPrest", elemTxt("vServ", formatarValor(d.ValorCentavos)))+
			elem("trib",
				elem("tribMun",
					elemTxt("tribISSQN", "1")+
						elemTxt("tpRetISSQN", "1"))+
					elem("tribFed",
						elem("piscofins",
							elemTxt("CST", "00")+
								elemTxt("tpRetPisCofins", "0")))+
					elem("totTrib", elemTxt("pTotTribSN", ptot))))

	ibscbs := elem("IBSCBS",
		elemTxt("finNFSe", "0")+
			elemTxt("indFinal", "0")+
			elemTxt("cIndOp", nfseCIndOp)+
			elemTxt("indDest", "0")+
			elem("valores",
				elem("trib",
					elem("gIBSCBS",
						elemTxt("CST", "000")+
							elemTxt("cClassTrib", "000001")))))

	corpo := elemTxt("tpAmb", fmt.Sprint(d.Ambiente)) +
		elemTxt("dhEmi", d.Emissao.Format("2006-01-02T15:04:05-07:00")) +
		elemTxt("verAplic", NFSeVerAplic) +
		elemTxt("serie", serie) +
		elemTxt("nDPS", fmt.Sprint(d.NDPS)) +
		elemTxt("dCompet", d.Competencia.Format("2006-01-02")) +
		elemTxt("tpEmit", "1") +
		elemTxt("cLocEmi", soDigitos(d.CodMunEmissor)) +
		prest + toma + serv + valores + ibscbs

	return DPSGerado{
		ID:      id,
		InfDPS:  "<infDPS Id=\"" + xmlAttr(id) + "\">" + corpo + "</infDPS>",
		Versao:  NFSeVersaoDPS,
		Serie:   serie,
		NDPS:    d.NDPS,
		DataXML: d.Emissao,
	}, nil
}

// infDPSCanonico é o infDPS como nó apex da canonicalização exclusiva: o namespace padrão
// é declarado no próprio elemento, antes dos demais atributos.
func (g DPSGerado) infDPSCanonico() string {
	return strings.Replace(g.InfDPS, "<infDPS Id=", "<infDPS xmlns=\""+NFSeNamespace+"\" Id=", 1)
}

// XMLNaoAssinado devolve o <DPS> completo, sem <Signature>.
func (g DPSGerado) XMLNaoAssinado() string {
	return g.montarXML("")
}

func (g DPSGerado) montarXML(assinatura string) string {
	return "<DPS xmlns=\"" + NFSeNamespace + "\" versao=\"" + g.Versao + "\">" + g.InfDPS + assinatura + "</DPS>"
}

// EmpacotarDPS comprime (gzip) e codifica em base64 o XML, como exige o POST /nfse.
func EmpacotarDPS(xml string) (string, error) {
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	if _, err := zw.Write([]byte(xml)); err != nil {
		return "", fmt.Errorf("gzip: %w", err)
	}
	if err := zw.Close(); err != nil {
		return "", fmt.Errorf("gzip: %w", err)
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}
