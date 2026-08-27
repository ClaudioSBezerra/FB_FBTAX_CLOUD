package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/johnfercher/maroto/v2/pkg/consts/align"

	"fb_cloud/services"
)

// ── Estruturas de detalhe do contrato ────────────────────────────────────────

type ContratoDetalhe struct {
	ID            string          `json:"id"`
	Numero        string          `json:"numero"`
	DataInicio    string          `json:"data_inicio"`
	Periodicidade string          `json:"periodicidade"`
	ValorTotal    float64         `json:"valor_total"`
	Status        string          `json:"status"`
	Observacoes   string          `json:"observacoes"`
	CriadoEm      string          `json:"criado_em"`
	AssinadoEm    *string         `json:"assinado_em"`
	AssinadoNome  *string         `json:"assinado_nome"`
	Cliente       ContratoCliente `json:"cliente"`
	Empresa       ContratoEmpresa `json:"empresa"`
	CNPJs         []ContratoCNPJ  `json:"cnpjs"`
	Itens         []ContratoItem  `json:"itens"`
}

type ContratoCliente struct {
	RazaoSocial string `json:"razao_social"`
	CNPJ        string `json:"cnpj"`
	Email       string `json:"email"`
	Fone        string `json:"fone"`
	Municipio   string `json:"municipio"`
	UF          string `json:"uf"`
	Responsavel string `json:"responsavel"`
	Logradouro  string `json:"logradouro"`
	NumeroEnd   string `json:"numero_end"`
}

type ContratoEmpresa struct {
	RazaoSocial  string `json:"razao_social"`
	NomeFantasia string `json:"nome_fantasia"`
	CNPJ         string `json:"cnpj"`
	Logradouro   string `json:"logradouro"`
	Numero       string `json:"numero"`
	Municipio    string `json:"municipio"`
	UF           string `json:"uf"`
}

type ContratoCNPJ struct {
	CNPJ      string `json:"cnpj"`
	Descricao string `json:"descricao"`
	Principal bool   `json:"principal"`
}

type ContratoItem struct {
	Produto   string   `json:"produto"`
	Plano     string   `json:"plano"`
	ValorItem *float64 `json:"valor_item"`
}

// ── GET /api/financeiro/contratos/detalhe?id=xxx ─────────────────────────────

func ContratoDetalheHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		id := r.URL.Query().Get("id")
		if id == "" {
			http.Error(w, "id obrigatório", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")

		det, err := carregarDetalheContrato(db, id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(det)
	}
}

func carregarDetalheContrato(db *sql.DB, id string) (*ContratoDetalhe, error) {
	var d ContratoDetalhe
	var obs, assinadoNome sql.NullString
	var assinadoEmTime sql.NullTime

	err := db.QueryRow(`
		SELECT c.id, COALESCE(c.numero,''), c.data_inicio, c.periodicidade,
		       c.valor_total, c.status, COALESCE(c.observacoes,''), c.created_at,
		       c.assinado_em, c.assinado_nome,
		       cl.razao_social, COALESCE(cl.cnpj,''), COALESCE(cl.email,''),
		       COALESCE(cl.telefone,''), COALESCE(cl.municipio,''), COALESCE(cl.uf,''),
		       COALESCE(cl.responsavel,''), COALESCE(cl.logradouro,''), COALESCE(cl.numero,'')
		FROM financeiro.contratos c
		JOIN financeiro.clientes cl ON cl.id = c.cliente_id
		WHERE c.id = $1`, id,
	).Scan(
		&d.ID, &d.Numero, &d.DataInicio, &d.Periodicidade,
		&d.ValorTotal, &d.Status, &obs, &d.CriadoEm,
		&assinadoEmTime, &assinadoNome,
		&d.Cliente.RazaoSocial, &d.Cliente.CNPJ, &d.Cliente.Email,
		&d.Cliente.Fone, &d.Cliente.Municipio, &d.Cliente.UF,
		&d.Cliente.Responsavel, &d.Cliente.Logradouro, &d.Cliente.NumeroEnd,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("contrato não encontrado")
	}
	if err != nil {
		return nil, fmt.Errorf("erro ao carregar contrato: %w", err)
	}
	d.Observacoes = obs.String
	if assinadoEmTime.Valid {
		s := assinadoEmTime.Time.Format("02/01/2006 15:04")
		d.AssinadoEm = &s
	}
	if assinadoNome.Valid {
		d.AssinadoNome = &assinadoNome.String
	}

	// Empresa contratante
	db.QueryRow(`
		SELECT COALESCE(razao_social,''), COALESCE(nome_fantasia,''), COALESCE(cnpj,''),
		       COALESCE(logradouro,''), COALESCE(numero,''), COALESCE(municipio,''), COALESCE(uf,'')
		FROM financeiro.empresas LIMIT 1`,
	).Scan(
		&d.Empresa.RazaoSocial, &d.Empresa.NomeFantasia, &d.Empresa.CNPJ,
		&d.Empresa.Logradouro, &d.Empresa.Numero, &d.Empresa.Municipio, &d.Empresa.UF,
	)

	// CNPJs
	cnpjRows, _ := db.Query(`
		SELECT cc.cnpj, COALESCE(cc.descricao,''), cc.is_principal
		FROM financeiro.contrato_cnpjs ccj
		JOIN financeiro.cliente_cnpjs cc ON cc.id = ccj.cnpj_id
		WHERE ccj.contrato_id = $1
		ORDER BY cc.is_principal DESC, cc.cnpj`, id)
	if cnpjRows != nil {
		defer cnpjRows.Close()
		for cnpjRows.Next() {
			var c ContratoCNPJ
			cnpjRows.Scan(&c.CNPJ, &c.Descricao, &c.Principal)
			d.CNPJs = append(d.CNPJs, c)
		}
	}

	// Itens
	itemRows, _ := db.Query(`
		SELECT p.nome AS produto, pl.nome AS plano, ci.valor_item
		FROM financeiro.contrato_itens ci
		JOIN financeiro.planos pl ON pl.id = ci.plano_id
		JOIN financeiro.produtos p ON p.id = pl.produto_id
		WHERE ci.contrato_id = $1
		ORDER BY p.nome, pl.nome`, id)
	if itemRows != nil {
		defer itemRows.Close()
		for itemRows.Next() {
			var item ContratoItem
			var vi sql.NullFloat64
			itemRows.Scan(&item.Produto, &item.Plano, &vi)
			if vi.Valid {
				item.ValorItem = &vi.Float64
			}
			d.Itens = append(d.Itens, item)
		}
	}

	return &d, nil
}

// ── GET /api/financeiro/contratos/pdf?id=xxx ─────────────────────────────────

func ContratoPDFHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		id := r.URL.Query().Get("id")
		if id == "" {
			http.Error(w, "id obrigatório", http.StatusBadRequest)
			return
		}

		det, err := carregarDetalheContrato(db, id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		pdfBytes, err := gerarContratoPDF(det)
		if err != nil {
			http.Error(w, "erro gerando PDF: "+err.Error(), http.StatusInternalServerError)
			return
		}

		nome := fmt.Sprintf("Contrato_%s.pdf", det.Numero)
		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`inline; filename="%s"`, nome))
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(pdfBytes)))
		w.Write(pdfBytes)
	}
}

func formatarCNPJ(cnpj string) string {
	d := []rune{}
	for _, c := range cnpj {
		if c >= '0' && c <= '9' {
			d = append(d, c)
		}
	}
	if len(d) == 14 {
		return fmt.Sprintf("%s.%s.%s/%s-%s",
			string(d[0:2]), string(d[2:5]), string(d[5:8]), string(d[8:12]), string(d[12:14]))
	}
	return cnpj
}

var formatsData = []string{
	"2006-01-02",
	time.RFC3339,
	"2006-01-02T15:04:05Z",
	"2006-01-02 15:04:05",
}

func parseData(iso string) (time.Time, bool) {
	for _, f := range formatsData {
		if t, err := time.Parse(f, iso); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

func formatarData(iso string) string {
	t, ok := parseData(iso)
	if !ok {
		return iso
	}
	meses := []string{"", "janeiro", "fevereiro", "março", "abril", "maio", "junho",
		"julho", "agosto", "setembro", "outubro", "novembro", "dezembro"}
	return fmt.Sprintf("%d de %s de %d", t.Day(), meses[int(t.Month())], t.Year())
}

func formatarDataCurta(iso string) string {
	t, ok := parseData(iso)
	if !ok {
		return iso
	}
	return t.Format("02/01/2006")
}

func formatarMoeda(v float64) string {
	cents := int64(math.Round(v * 100))
	dec := cents % 100
	inteiro := cents / 100
	s := strconv.FormatInt(inteiro, 10)
	result := make([]byte, 0, len(s)+5)
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result = append(result, '.')
		}
		result = append(result, byte(c))
	}
	return fmt.Sprintf("R$ %s,%02d", string(result), dec)
}

// gerarContratoPDF monta o Contrato de Prestação de Serviços no Padrão de
// Documentos FBTECH (services/fbdoc.go).
//
// As cláusulas conservam a numeração própria do instrumento ("CLÁUSULA 1ª"),
// por isso são abertas com Bloco e não com Secao: a numeração automática do kit
// duplicaria a que já consta do texto jurídico.
func gerarContratoPDF(d *ContratoDetalhe) ([]byte, error) {
	// ── dados da empresa (fallback) ───────────────────────────────────────────
	nomeEmpresa := d.Empresa.RazaoSocial
	if nomeEmpresa == "" {
		nomeEmpresa = "FORTES BEZERRA TECNOLOGIA E CONSULTORIA LTDA"
	}
	cnpjEmpresa := formatarCNPJ(d.Empresa.CNPJ)
	if cnpjEmpresa == "" {
		cnpjEmpresa = "38.149.716/0001-28"
	}
	endEmpresa := ""
	if d.Empresa.Logradouro != "" {
		endEmpresa = fmt.Sprintf("%s, nº %s, %s/%s", d.Empresa.Logradouro, d.Empresa.Numero, d.Empresa.Municipio, d.Empresa.UF)
	} else {
		endEmpresa = "Aparecida de Goiânia - GO"
	}
	municipioEmpresa := d.Empresa.Municipio
	if municipioEmpresa == "" {
		municipioEmpresa = "Aparecida de Goiânia"
	}
	ufEmpresa := d.Empresa.UF
	if ufEmpresa == "" {
		ufEmpresa = "GO"
	}

	// ── dados do cliente ──────────────────────────────────────────────────────
	cnpjCliente := formatarCNPJ(d.Cliente.CNPJ)
	endCliente := ""
	if d.Cliente.Logradouro != "" {
		endCliente = d.Cliente.Logradouro
		if d.Cliente.NumeroEnd != "" {
			endCliente += ", nº " + d.Cliente.NumeroEnd
		}
		if d.Cliente.Municipio != "" {
			endCliente += fmt.Sprintf(", %s/%s", d.Cliente.Municipio, d.Cliente.UF)
		}
	} else if d.Cliente.Municipio != "" {
		endCliente = fmt.Sprintf("%s/%s", d.Cliente.Municipio, d.Cliente.UF)
	}

	dataDoc, ok := parseData(d.DataInicio)
	if !ok {
		dataDoc = time.Now()
	}
	cidade := municipioEmpresa + " – " + ufEmpresa

	doc := services.NovoDoc(services.DocMeta{
		Sigla:        "CT",
		Numero:       strings.TrimPrefix(d.Numero, "FB-"),
		Versao:       "1.0",
		Titulo:       "Contrato de Prestação de Serviços de Tecnologia",
		Subtitulo:    d.Cliente.RazaoSocial,
		Cliente:      d.Cliente.RazaoSocial,
		Cidade:       cidade,
		Data:         dataDoc,
		Emitente:     nomeEmpresa,
		EmitCNPJ:     cnpjEmpresa,
		Confidencial: true,
	})

	doc.Capa()

	// ── PREÂMBULO ─────────────────────────────────────────────────────────────
	doc.Bloco("QUALIFICAÇÃO DAS PARTES")
	doc.Paragrafo(fmt.Sprintf(
		"%s, pessoa jurídica de direito privado, inscrita no CNPJ sob o nº %s, com sede na %s, doravante denominada simplesmente CONTRATANTE;",
		nomeEmpresa, cnpjEmpresa, endEmpresa,
	))
	doc.Paragrafo("E")

	clienteDesc := d.Cliente.RazaoSocial
	if cnpjCliente != "" {
		clienteDesc += fmt.Sprintf(", pessoa jurídica de direito privado, inscrita no CNPJ sob o nº %s", cnpjCliente)
	}
	if endCliente != "" {
		clienteDesc += fmt.Sprintf(", com sede na %s", endCliente)
	}
	clienteDesc += ", doravante denominada simplesmente CONTRATADA;"
	doc.Paragrafo(clienteDesc)
	doc.Paragrafo("As partes acima qualificadas, em conjunto denominadas PARTES, têm entre si justo e acordado o presente Contrato de Prestação de Serviços de Tecnologia, que se regerá pelas cláusulas e condições a seguir estipuladas.")

	// ── CLÁUSULA 1ª ───────────────────────────────────────────────────────────
	doc.Bloco("CLÁUSULA 1ª – DO OBJETO")
	doc.Paragrafo("O presente Contrato tem por objeto a prestação de serviços de tecnologia pela CONTRATANTE à CONTRATADA, consistindo no acesso e utilização dos sistemas e plataformas digitais integrantes do ecossistema da Fortes Bezerra Tecnologia, nos termos e condições estabelecidos neste instrumento.")

	// ── CLÁUSULA 2ª ───────────────────────────────────────────────────────────
	doc.Bloco("CLÁUSULA 2ª – DOS SERVIÇOS CONTRATADOS")
	doc.Paragrafo("Os serviços objeto deste Contrato compreendem o acesso às seguintes soluções e respectivos planos:")

	linhasItens := make([][]string, 0, len(d.Itens))
	for _, item := range d.Itens {
		valorStr := "Sob consulta"
		if item.ValorItem != nil {
			valorStr = formatarMoeda(*item.ValorItem)
		}
		linhasItens = append(linhasItens, []string{item.Produto, item.Plano, valorStr})
	}
	doc.Tabela([]services.Coluna{
		{Titulo: "Produto / Solução", Largura: 5},
		{Titulo: "Plano", Largura: 4},
		{Titulo: "Valor Mensal (R$)", Largura: 3, Align: align.Right},
	}, linhasItens)

	if len(d.CNPJs) > 0 {
		doc.Paragrafo("Parágrafo único. Os serviços acima abrangem os seguintes CNPJs da CONTRATADA:")
		for _, c := range d.CNPJs {
			label := formatarCNPJ(c.CNPJ)
			if c.Principal {
				label += " (Estabelecimento Principal)"
			} else if c.Descricao != "" {
				label += " — " + c.Descricao
			}
			doc.Marcador(label)
		}
		doc.Esp(3)
	}

	// ── CLÁUSULA 3ª ───────────────────────────────────────────────────────────
	doc.Bloco("CLÁUSULA 3ª – DO VALOR E DA FORMA DE PAGAMENTO")
	doc.Campo("Valor total:", fmt.Sprintf("%s (%s)", formatarMoeda(d.ValorTotal), valorPorExtenso(d.ValorTotal)))
	doc.Campo("Periodicidade:", capitalizar(d.Periodicidade))
	doc.Campo("Vigência a partir de:", formatarDataCurta(d.DataInicio))
	doc.Esp(3)
	doc.Paragrafo("3.1. O pagamento deverá ser realizado na data de vencimento acordada entre as PARTES, mediante boleto bancário, transferência bancária (TED/PIX) ou outra forma previamente convencionada.")
	doc.Paragrafo("3.2. O inadimplemento por prazo superior a 15 (quinze) dias corridos implicará a suspensão automática do acesso aos serviços contratados, sem prejuízo da cobrança de multa de 2% (dois por cento) sobre o valor em atraso, acrescida de juros de mora de 1% (um por cento) ao mês e correção monetária pelo IGPM/FGV.")

	// ── CLÁUSULA 4ª ───────────────────────────────────────────────────────────
	doc.Bloco("CLÁUSULA 4ª – DAS OBRIGAÇÕES DAS PARTES")
	doc.Paragrafo("4.1. Compete à CONTRATANTE: (i) disponibilizar o acesso às plataformas contratadas em ambiente operacional; (ii) prestar suporte técnico nos termos do plano contratado; (iii) manter os sistemas atualizados conforme a legislação fiscal e tributária vigente; (iv) garantir a segurança e disponibilidade dos dados no ambiente de nuvem.")
	doc.Paragrafo("4.2. Compete à CONTRATADA: (i) efetuar os pagamentos nas datas acordadas; (ii) utilizar os serviços de forma lícita e de acordo com a legislação vigente; (iii) manter atualizados seus dados cadastrais junto à CONTRATANTE; (iv) não ceder, sublicenciar ou compartilhar credenciais de acesso a terceiros.")

	// ── CLÁUSULA 5ª ───────────────────────────────────────────────────────────
	doc.Bloco("CLÁUSULA 5ª – DA VIGÊNCIA E DA RESCISÃO")
	doc.Paragrafo(fmt.Sprintf("5.1. O presente Contrato entra em vigor na data de sua assinatura, com início da prestação dos serviços em %s, e permanecerá em vigor por prazo indeterminado, renovando-se automaticamente a cada período de cobrança.", formatarDataCurta(d.DataInicio)))
	doc.Paragrafo("5.2. Qualquer das PARTES poderá rescindir o presente Contrato mediante notificação prévia por escrito com antecedência mínima de 30 (trinta) dias, sem a incidência de multas rescisórias, desde que não haja débitos pendentes.")
	doc.Paragrafo("5.3. A rescisão imotivada pela CONTRATADA dentro dos primeiros 12 (doze) meses de vigência implicará o pagamento de multa equivalente a 2 (duas) mensalidades, a título de compensação pelos investimentos realizados pela CONTRATANTE.")

	// ── CLÁUSULA 6ª ───────────────────────────────────────────────────────────
	doc.Bloco("CLÁUSULA 6ª – DA PROPRIEDADE INTELECTUAL")
	doc.Paragrafo("6.1. Todos os sistemas, softwares, algoritmos, bases de dados, interfaces, documentações e demais ativos tecnológicos disponibilizados pela CONTRATANTE são de sua exclusiva propriedade intelectual, protegidos pela Lei nº 9.279/1996 e pela Lei nº 9.609/1998 (Lei do Software), sendo vedada qualquer reprodução, cópia, engenharia reversa, adaptação ou uso não autorizado, sob pena de responsabilização civil e criminal.")
	doc.Paragrafo("6.2. A CONTRATADA reconhece que os sistemas disponibilizados constituem segredo industrial e comercial da CONTRATANTE, comprometendo-se a não desenvolver, direta ou indiretamente, produto ou serviço concorrente com base em informações obtidas por meio deste Contrato.")

	// ── CLÁUSULA 7ª – NDA ─────────────────────────────────────────────────────
	doc.Bloco("CLÁUSULA 7ª – DO SIGILO E CONFIDENCIALIDADE (NDA)")
	doc.Paragrafo("7.1. As PARTES reconhecem que, em razão da execução deste Contrato, terão acesso a informações confidenciais da outra parte, incluindo, mas não se limitando a: dados técnicos, estratégias comerciais, listas de clientes, metodologias, precificação, planos de negócio, código-fonte, arquiteturas de sistemas e quaisquer outras informações designadas como confidenciais ou que, pela sua natureza, devam ser tratadas como tal.")
	doc.Paragrafo("7.2. Cada PARTE obriga-se a: (i) manter em estrito sigilo todas as Informações Confidenciais recebidas; (ii) utilizar tais informações exclusivamente para os fins previstos neste Contrato; (iii) não divulgar, reproduzir ou transferir as Informações Confidenciais a terceiros sem autorização prévia e escrita da outra PARTE; (iv) adotar medidas de segurança adequadas para proteger as informações, no mínimo equivalentes às que adota para proteger suas próprias informações confidenciais.")
	doc.Paragrafo("7.3. As obrigações de confidencialidade não se aplicam a informações que: (i) sejam ou se tornem públicas sem culpa da PARTE receptora; (ii) já eram de conhecimento da PARTE receptora antes da divulgação; (iii) sejam recebidas de terceiros sem restrição de sigilo; ou (iv) devam ser divulgadas por determinação legal ou judicial, desde que a PARTE afetada seja imediatamente notificada.")
	doc.Paragrafo("7.4. As obrigações estabelecidas nesta Cláusula permanecerão em vigor durante toda a vigência do Contrato e pelo prazo de 5 (cinco) anos após sua extinção, independentemente do motivo. O descumprimento sujeitará a PARTE infratora ao pagamento de indenização por perdas e danos, sem prejuízo das medidas cautelares cabíveis.")

	// ── CLÁUSULA 8ª ───────────────────────────────────────────────────────────
	doc.Bloco("CLÁUSULA 8ª – DA PROTEÇÃO DE DADOS PESSOAIS (LGPD)")
	doc.Paragrafo("As PARTES declaram estar cientes e em conformidade com a Lei Geral de Proteção de Dados Pessoais — LGPD (Lei nº 13.709/2018). A CONTRATANTE atuará como Operadora dos dados pessoais eventualmente processados nos sistemas em nome da CONTRATADA (Controladora), comprometendo-se a adotar medidas técnicas e administrativas adequadas para garantir a segurança, confidencialidade e integridade das informações tratadas.")

	// ── CLÁUSULA 9ª ───────────────────────────────────────────────────────────
	doc.Bloco("CLÁUSULA 9ª – DO FORO")
	doc.Paragrafo(fmt.Sprintf("Fica eleito o foro da Comarca de %s, Estado de %s, com exclusão de qualquer outro, por mais privilegiado que seja, para dirimir quaisquer litígios decorrentes deste Contrato.", municipioEmpresa, ufEmpresa))

	if d.Observacoes != "" {
		doc.Bloco("DISPOSIÇÕES ADICIONAIS")
		doc.Paragrafo(d.Observacoes)
	}

	// ── ENCERRAMENTO E ASSINATURAS ────────────────────────────────────────────
	doc.Bloco("ENCERRAMENTO")
	doc.Paragrafo("E, por estarem assim justas e acordadas, as PARTES assinam o presente Contrato em 2 (duas) vias de igual teor e forma, na presença das testemunhas abaixo identificadas.")
	doc.LocalEData(cidade)

	doc.BlocoAssinaturas([]services.Signatario{
		{Papel: "CONTRATANTE", Org: nomeEmpresa, Doc: cnpjEmpresa, RotuloDoc: "CNPJ"},
		{Papel: "CONTRATADA", Org: d.Cliente.RazaoSocial, Doc: cnpjCliente, RotuloDoc: "CNPJ"},
	})

	doc.Esp(8)
	doc.Bloco("TESTEMUNHAS")
	doc.BlocoAssinaturas([]services.Signatario{
		{Papel: "TESTEMUNHA 1"},
		{Papel: "TESTEMUNHA 2"},
	})

	doc.Esp(6)
	doc.Nota(fmt.Sprintf("Contrato %s gerado pelo FBTax Cloud em %s.",
		d.Numero, time.Now().Format("02/01/2006 às 15:04")))

	return doc.Bytes()
}

func capitalizar(s string) string {
	if len(s) == 0 {
		return s
	}
	return string(s[0]-32) + s[1:]
}

// ── POST /api/financeiro/contratos/upload-assinado ───────────────────────────

func ContratoUploadAssinadoHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")

		// Limita a 20MB
		r.ParseMultipartForm(20 << 20)

		contratoID := r.FormValue("contrato_id")
		if contratoID == "" {
			http.Error(w, "contrato_id obrigatório", http.StatusBadRequest)
			return
		}

		file, header, err := r.FormFile("arquivo")
		if err != nil {
			http.Error(w, "arquivo obrigatório", http.StatusBadRequest)
			return
		}
		defer file.Close()

		data, err := io.ReadAll(file)
		if err != nil {
			http.Error(w, "erro lendo arquivo", http.StatusInternalServerError)
			return
		}

		_, err = db.Exec(`
			UPDATE financeiro.contratos
			SET assinado_data = $1, assinado_nome = $2, assinado_em = NOW()
			WHERE id = $3`,
			data, header.Filename, contratoID,
		)
		if err != nil {
			http.Error(w, "db error: "+err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"mensagem": "Contrato assinado enviado com sucesso"})
	}
}

// ── GET /api/financeiro/contratos/download-assinado?id=xxx ──────────────────

func ContratoDownloadAssinadoHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		id := r.URL.Query().Get("id")
		if id == "" {
			http.Error(w, "id obrigatório", http.StatusBadRequest)
			return
		}

		var data []byte
		var nome sql.NullString
		err := db.QueryRow(`SELECT assinado_data, assinado_nome FROM financeiro.contratos WHERE id = $1`, id).
			Scan(&data, &nome)
		if err != nil || data == nil {
			http.Error(w, "arquivo não encontrado", http.StatusNotFound)
			return
		}

		filename := "contrato_assinado.pdf"
		if nome.Valid && nome.String != "" {
			filename = nome.String
		}
		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`inline; filename="%s"`, filename))
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
		w.Write(data)
	}
}

func valorPorExtenso(v float64) string {
	inteiro := int64(v)
	centavos := int64((v - float64(inteiro)) * 100)

	unidades := []string{"", "um", "dois", "três", "quatro", "cinco", "seis", "sete", "oito", "nove",
		"dez", "onze", "doze", "treze", "quatorze", "quinze", "dezesseis", "dezessete", "dezoito", "dezenove"}
	dezenas := []string{"", "", "vinte", "trinta", "quarenta", "cinquenta", "sessenta", "setenta", "oitenta", "noventa"}
	centenas := []string{"", "cem", "duzentos", "trezentos", "quatrocentos", "quinhentos",
		"seiscentos", "setecentos", "oitocentos", "novecentos"}

	var escrever func(n int64) string
	escrever = func(n int64) string {
		switch {
		case n == 0:
			return "zero"
		case n < 20:
			return unidades[n]
		case n < 100:
			r := dezenas[n/10]
			if n%10 != 0 {
				r += " e " + unidades[n%10]
			}
			return r
		case n < 1000:
			if n == 100 {
				return "cem"
			}
			r := centenas[n/100]
			if n%100 != 0 {
				r += " e " + escrever(n%100)
			}
			return r
		case n < 1000000:
			mil := n / 1000
			resto := n % 1000
			r := escrever(mil) + " mil"
			if resto != 0 {
				r += " e " + escrever(resto)
			}
			return r
		default:
			return fmt.Sprintf("%d", n)
		}
	}

	resultado := escrever(inteiro) + " reais"
	if centavos > 0 {
		resultado += " e " + escrever(centavos) + " centavos"
	}
	return resultado
}
