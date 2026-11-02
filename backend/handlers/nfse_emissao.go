package handlers

// Emissão de NFS-e Nacional (Sefin Nacional) pelo módulo financeiro: certificado A1 do
// prestador, parâmetros fiscais, emissão, consulta e cancelamento.
//
// Salvaguardas (a emissão em produção é um ato fiscal irreversível sem cancelamento):
//   - "enviar" só vale quando true; sem ele a rota devolve a prévia do DPS assinado, sem gastar
//     número nem tocar na Sefin;
//   - em produção (nfse_ambiente = 1) o envio e o cancelamento exigem a frase de confirmação;
//   - o certificado precisa estar válido e pertencer ao CNPJ do prestador antes de reservar
//     o número de DPS;
//   - PFX e senha nunca são devolvidos por nenhuma rota nem escritos em log.

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"fb_cloud/services"

	"github.com/lib/pq"
)

const (
	nfseFraseProducao = "EMITIR EM PRODUCAO"
	nfseMaxPFX        = 256 << 10
)

var (
	nfseBRT   = time.FixedZone("-03", -3*3600)
	reNNFSe   = regexp.MustCompile(`<nNFSe>(\d+)</nNFSe>`)
	reUUIDNFS = regexp.MustCompile(`^[0-9a-fA-F-]{36}$`)
)

func nfseJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func nfseErro(w http.ResponseWriter, status int, msg string) {
	nfseJSON(w, status, map[string]string{"error": msg})
}

type nfseEmpresa struct {
	ID, CNPJ, CodIBGE, Fone, Email, IM string
	OpSimpNac, RegApTribSN, RegEspTrib int
	Ambiente                           int
}

func carregarEmpresaNFSe(ctx context.Context, db *sql.DB) (*nfseEmpresa, error) {
	var e nfseEmpresa
	var ibge, fone, email, im sql.NullString
	err := db.QueryRowContext(ctx, `
		SELECT id, cnpj, cod_ibge, telefone, email, inscricao_municipal,
		       op_simp_nac, reg_ap_trib_sn, reg_esp_trib, nfse_ambiente
		FROM financeiro.empresas ORDER BY created_at LIMIT 1`).Scan(
		&e.ID, &e.CNPJ, &ibge, &fone, &email, &im, &e.OpSimpNac, &e.RegApTribSN, &e.RegEspTrib, &e.Ambiente)
	if err != nil {
		return nil, err
	}
	e.CNPJ = onlyDigits(e.CNPJ)
	e.CodIBGE, e.Fone, e.Email, e.IM = ibge.String, fone.String, email.String, im.String
	return &e, nil
}

func onlyDigits(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func carregarA1(ctx context.Context, db *sql.DB, empresaID string) (*services.NFSeCertificadoA1, error) {
	var pfx, senha string
	err := db.QueryRowContext(ctx, `SELECT certificado_pfx, certificado_senha FROM financeiro.nfse_certificado WHERE empresa_id = $1`, empresaID).Scan(&pfx, &senha)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("nenhum certificado A1 cadastrado: envie o certificado antes de emitir")
	}
	if err != nil {
		return nil, err
	}
	return services.CarregarCertificadoA1(pfx, senha)
}

// NFSeCertificadoHandler: GET status, POST multipart (campos "pfx" e "senha"), DELETE.
func NFSeCertificadoHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		emp, err := carregarEmpresaNFSe(r.Context(), db)
		if err != nil {
			nfseErro(w, http.StatusConflict, "cadastre a empresa emitente antes (Financeiro > Empresa)")
			return
		}
		switch r.Method {
		case http.MethodGet:
			var nome sql.NullString
			var cnpj sql.NullString
			var validade time.Time
			err := db.QueryRowContext(r.Context(), `SELECT nome_arquivo, cnpj_titular, validade FROM financeiro.nfse_certificado WHERE empresa_id = $1`, emp.ID).Scan(&nome, &cnpj, &validade)
			if err == sql.ErrNoRows {
				nfseJSON(w, http.StatusOK, map[string]any{"cadastrado": false})
				return
			}
			if err != nil {
				nfseErro(w, http.StatusInternalServerError, "erro ao consultar certificado")
				return
			}
			dias := int(time.Until(validade).Hours() / 24)
			nfseJSON(w, http.StatusOK, map[string]any{
				"cadastrado": true, "nome_arquivo": nome.String, "cnpj_titular": cnpj.String,
				"validade": validade.Format("2006-01-02"), "dias_restantes": dias,
				"cnpj_confere": cnpj.String == emp.CNPJ, "vencido": dias < 0,
			})
		case http.MethodPost:
			r.Body = http.MaxBytesReader(w, r.Body, nfseMaxPFX+(8<<10))
			if err := r.ParseMultipartForm(nfseMaxPFX); err != nil {
				nfseErro(w, http.StatusBadRequest, "arquivo muito grande ou multipart inválido")
				return
			}
			f, hdr, err := r.FormFile("pfx")
			if err != nil {
				nfseErro(w, http.StatusBadRequest, "campo 'pfx' obrigatório")
				return
			}
			defer f.Close()
			pfx, err := io.ReadAll(io.LimitReader(f, nfseMaxPFX))
			if err != nil || len(pfx) == 0 {
				nfseErro(w, http.StatusBadRequest, "arquivo vazio")
				return
			}
			senha := r.FormValue("senha")
			// Certificados A1 reais costumam vir em BER, que o Go não lê: reconverte para DER via openssl.
			pfx, _, err = services.NormalizarPKCS12ParaDER(pfx, senha)
			if err != nil {
				log.Printf("[NFSe] normalização do certificado falhou: %v", err)
				switch {
				case errors.Is(err, services.ErrNFSeOpensslIndisponivel), errors.Is(err, services.ErrNFSeAmbienteTemporarioIndisponivel):
					nfseErro(w, http.StatusInternalServerError, "o servidor não consegue converter este certificado (openssl indisponível); avise o suporte")
				default:
					nfseErro(w, http.StatusUnprocessableEntity, "não foi possível abrir o certificado: confira o arquivo e a senha")
				}
				return
			}
			a1, err := services.DecodificarPFX(pfx, senha)
			if err != nil {
				nfseErro(w, http.StatusUnprocessableEntity, err.Error())
				return
			}
			// O .pfx real costuma vir sem as ACs intermediárias e o servidor do governo rejeita o
			// handshake: completa a cadeia via AIA e valida até uma raiz ICP-Brasil.
			cadeia, err := completarCadeiaNFSe(r.Context(), a1.Cert, a1.Cadeia)
			if err != nil {
				log.Printf("[NFSe] cadeia do certificado: %v", err)
				status, msg := mensagemErroCadeiaNFSe(err)
				nfseErro(w, status, msg)
				return
			}
			a1.Cadeia = cadeia
			if pfx, err = services.CodificarPFXComCadeia(a1, senha); err != nil {
				nfseErro(w, http.StatusInternalServerError, "erro ao reempacotar o certificado")
				return
			}
			if err := services.ValidarCertificadoDoPrestador(a1.Cert, emp.CNPJ, time.Now()); err != nil {
				nfseErro(w, http.StatusUnprocessableEntity, err.Error())
				return
			}
			pfxEnc, senhaEnc, validade, err := services.CifrarCertificadoA1(pfx, senha)
			if err != nil {
				nfseErro(w, http.StatusUnprocessableEntity, err.Error())
				return
			}
			_, err = db.ExecContext(r.Context(), `
				INSERT INTO financeiro.nfse_certificado (empresa_id, certificado_pfx, certificado_senha, nome_arquivo, cnpj_titular, validade)
				VALUES ($1,$2,$3,$4,$5,$6)
				ON CONFLICT (empresa_id) DO UPDATE SET certificado_pfx = EXCLUDED.certificado_pfx,
				  certificado_senha = EXCLUDED.certificado_senha, nome_arquivo = EXCLUDED.nome_arquivo,
				  cnpj_titular = EXCLUDED.cnpj_titular, validade = EXCLUDED.validade, updated_at = NOW()`,
				emp.ID, pfxEnc, senhaEnc, hdr.Filename, services.CNPJDoCertificado(a1.Cert), validade)
			if err != nil {
				log.Printf("[NFSe] erro ao gravar certificado: %v", err)
				nfseErro(w, http.StatusInternalServerError, "erro ao gravar certificado")
				return
			}
			log.Printf("[NFSe] certificado A1 atualizado (CNPJ %s, validade %s)", emp.CNPJ, validade.Format("2006-01-02"))
			nfseJSON(w, http.StatusOK, map[string]any{"status": "ok", "validade": validade.Format("2006-01-02")})
		case http.MethodDelete:
			if _, err := db.ExecContext(r.Context(), `DELETE FROM financeiro.nfse_certificado WHERE empresa_id = $1`, emp.ID); err != nil {
				nfseErro(w, http.StatusInternalServerError, "erro ao remover certificado")
				return
			}
			log.Printf("[NFSe] certificado A1 removido")
			nfseJSON(w, http.StatusOK, map[string]string{"status": "removido"})
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

type nfsePrestadorReq struct {
	CodIBGE            string `json:"cod_ibge"`
	Telefone           string `json:"telefone"`
	Email              string `json:"email"`
	InscricaoMunicipal string `json:"inscricao_municipal"`
	OpSimpNac          int    `json:"op_simp_nac"`
	RegApTribSN        int    `json:"reg_ap_trib_sn"`
	RegEspTrib         int    `json:"reg_esp_trib"`
	Ambiente           int    `json:"nfse_ambiente"`
	ConfirmarProducao  string `json:"confirmar_producao,omitempty"`
}

// NFSePrestadorHandler: GET/PUT dos parâmetros fiscais do emitente.
func NFSePrestadorHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		emp, err := carregarEmpresaNFSe(r.Context(), db)
		if err != nil {
			nfseErro(w, http.StatusConflict, "cadastre a empresa emitente antes (Financeiro > Empresa)")
			return
		}
		switch r.Method {
		case http.MethodGet:
			nfseJSON(w, http.StatusOK, map[string]any{
				"cnpj": emp.CNPJ, "cod_ibge": emp.CodIBGE, "telefone": emp.Fone, "email": emp.Email,
				"inscricao_municipal": emp.IM, "op_simp_nac": emp.OpSimpNac, "reg_ap_trib_sn": emp.RegApTribSN,
				"reg_esp_trib": emp.RegEspTrib, "nfse_ambiente": emp.Ambiente,
			})
		case http.MethodPut:
			var req nfsePrestadorReq
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				nfseErro(w, http.StatusBadRequest, "requisição inválida")
				return
			}
			req.CodIBGE = onlyDigits(req.CodIBGE)
			if len(req.CodIBGE) != 7 {
				nfseErro(w, http.StatusBadRequest, "código IBGE do município deve ter 7 dígitos")
				return
			}
			if req.Ambiente != 1 && req.Ambiente != 2 {
				nfseErro(w, http.StatusBadRequest, "ambiente deve ser 1 (produção) ou 2 (produção restrita)")
				return
			}
			if req.Ambiente == 1 && emp.Ambiente != 1 && req.ConfirmarProducao != nfseFraseProducao {
				nfseErro(w, http.StatusBadRequest, "para ativar a produção envie confirmar_producao = \""+nfseFraseProducao+"\"")
				return
			}
			_, err := db.ExecContext(r.Context(), `
				UPDATE financeiro.empresas SET cod_ibge=$1, telefone=$2, email=$3, inscricao_municipal=$4,
				  op_simp_nac=$5, reg_ap_trib_sn=$6, reg_esp_trib=$7, nfse_ambiente=$8, updated_at=CURRENT_TIMESTAMP
				WHERE id=$9`, req.CodIBGE, req.Telefone, req.Email, req.InscricaoMunicipal,
				req.OpSimpNac, req.RegApTribSN, req.RegEspTrib, req.Ambiente, emp.ID)
			if err != nil {
				nfseErro(w, http.StatusBadRequest, "parâmetros fiscais inválidos")
				return
			}
			log.Printf("[NFSe] parâmetros do prestador atualizados (ambiente %d)", req.Ambiente)
			nfseJSON(w, http.StatusOK, map[string]string{"status": "updated"})
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

// NFSeClientesHandler: GET lista clientes ativos com o IBGE; PUT {cliente_id, cod_ibge}.
func NFSeClientesHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			rows, err := db.QueryContext(r.Context(), `
				SELECT id, razao_social, cnpj, COALESCE(cep,''), COALESCE(municipio,''), COALESCE(uf,''), COALESCE(cod_ibge,'')
				FROM financeiro.clientes WHERE ativo ORDER BY razao_social`)
			if err != nil {
				nfseErro(w, http.StatusInternalServerError, "erro ao listar clientes")
				return
			}
			defer rows.Close()
			out := []map[string]string{}
			for rows.Next() {
				var id, rz, cnpj, cep, mun, uf, ibge string
				if rows.Scan(&id, &rz, &cnpj, &cep, &mun, &uf, &ibge) == nil {
					out = append(out, map[string]string{"id": id, "razao_social": rz, "cnpj": cnpj, "cep": cep, "municipio": mun, "uf": uf, "cod_ibge": ibge})
				}
			}
			nfseJSON(w, http.StatusOK, out)
		case http.MethodPut:
			var req struct {
				ClienteID string `json:"cliente_id"`
				CodIBGE   string `json:"cod_ibge"`
			}
			if json.NewDecoder(r.Body).Decode(&req) != nil || !reUUIDNFS.MatchString(req.ClienteID) || len(onlyDigits(req.CodIBGE)) != 7 {
				nfseErro(w, http.StatusBadRequest, "cliente_id e cod_ibge (7 dígitos) são obrigatórios")
				return
			}
			if _, err := db.ExecContext(r.Context(), `UPDATE financeiro.clientes SET cod_ibge = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2`, onlyDigits(req.CodIBGE), req.ClienteID); err != nil {
				nfseErro(w, http.StatusInternalServerError, "erro ao atualizar cliente")
				return
			}
			nfseJSON(w, http.StatusOK, map[string]string{"status": "updated"})
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

type nfseEmitirReq struct {
	ClienteID         string `json:"cliente_id"`
	ContratoID        string `json:"contrato_id,omitempty"`
	ValorCentavos     int64  `json:"valor_centavos"`
	Descricao         string `json:"descricao"`
	Competencia       string `json:"competencia,omitempty"` // AAAA-MM-DD; vazio = 1º dia do mês corrente
	Enviar            bool   `json:"enviar"`
	ConfirmarProducao string `json:"confirmar_producao,omitempty"`
}

// NFSeEmitirHandler trata POST /api/financeiro/nfse/emitir.
func NFSeEmitirHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req nfseEmitirReq
		if err := json.NewDecoder(io.LimitReader(r.Body, 64<<10)).Decode(&req); err != nil || !reUUIDNFS.MatchString(req.ClienteID) {
			nfseErro(w, http.StatusBadRequest, "requisição inválida: cliente_id obrigatório")
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 90*time.Second)
		defer cancel()

		emp, err := carregarEmpresaNFSe(ctx, db)
		if err != nil {
			nfseErro(w, http.StatusConflict, "cadastre a empresa emitente antes (Financeiro > Empresa)")
			return
		}
		if len(emp.CodIBGE) != 7 {
			nfseErro(w, http.StatusConflict, "informe o código IBGE do município do prestador (parâmetros fiscais)")
			return
		}
		var tom services.DPSTomador
		var ibge, cep, logr, num, cpl, bairro, tel, mail sql.NullString
		err = db.QueryRowContext(ctx, `
			SELECT razao_social, cnpj, cod_ibge, cep, logradouro, numero, complemento, bairro, telefone, email
			FROM financeiro.clientes WHERE id = $1`, req.ClienteID).Scan(
			&tom.Nome, &tom.CNPJ, &ibge, &cep, &logr, &num, &cpl, &bairro, &tel, &mail)
		if err == sql.ErrNoRows {
			nfseErro(w, http.StatusNotFound, "cliente não encontrado")
			return
		}
		if err != nil {
			nfseErro(w, http.StatusInternalServerError, "erro ao consultar cliente")
			return
		}
		tom.CodMun, tom.CEP, tom.Logradouro, tom.Numero = ibge.String, cep.String, logr.String, num.String
		tom.Complemento, tom.Bairro, tom.Fone, tom.Email = cpl.String, bairro.String, tel.String, mail.String

		agora := time.Now().In(nfseBRT).Add(-time.Minute) // a Sefin rejeita dhEmi à frente do relógio dela
		competencia := time.Date(agora.Year(), agora.Month(), 1, 0, 0, 0, 0, nfseBRT)
		if req.Competencia != "" {
			c, err := time.ParseInLocation("2006-01-02", req.Competencia, nfseBRT)
			if err != nil {
				nfseErro(w, http.StatusBadRequest, "competencia deve ser AAAA-MM-DD")
				return
			}
			competencia = c
		}
		if req.Enviar && emp.Ambiente == 1 && req.ConfirmarProducao != nfseFraseProducao {
			nfseErro(w, http.StatusBadRequest, "ambiente de PRODUÇÃO: para emitir de verdade envie confirmar_producao = \""+nfseFraseProducao+"\"")
			return
		}

		a1, err := carregarA1(ctx, db, emp.ID)
		if err != nil {
			nfseErro(w, http.StatusConflict, err.Error())
			return
		}

		dados := services.DPSDados{
			Ambiente: emp.Ambiente, Emissao: agora, Serie: services.NFSeSeriePadrao, NDPS: 1,
			Competencia: competencia, CodMunEmissor: emp.CodIBGE, CodMunPrestacao: tom.CodMun,
			Prestador: services.DPSPrestador{CNPJ: emp.CNPJ, IM: emp.IM, Fone: emp.Fone, Email: emp.Email,
				OpSimpNac: emp.OpSimpNac, RegApTribSN: emp.RegApTribSN, RegEspTrib: emp.RegEspTrib},
			Tomador: tom, Descricao: req.Descricao, ValorCentavos: req.ValorCentavos,
		}

		// Prévia: valida tudo e assina com número fictício, sem reservar DPS nem falar com a Sefin.
		if !req.Enviar {
			g, err := services.GerarDPS(dados)
			if err != nil {
				nfseErro(w, http.StatusUnprocessableEntity, err.Error())
				return
			}
			xml, err := services.AssinarDPS(g, a1)
			if err != nil {
				nfseErro(w, http.StatusUnprocessableEntity, err.Error())
				return
			}
			nfseJSON(w, http.StatusOK, map[string]any{"previa": true, "ambiente": emp.Ambiente, "dps_xml": xml,
				"aviso": "prévia: nenhum número de DPS foi reservado e nada foi enviado à Sefin"})
			return
		}

		// Valida com o número fictício antes de gastar um número de verdade.
		if _, err := services.GerarDPS(dados); err != nil {
			nfseErro(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
		nDPS, err := services.ReservarNDPS(ctx, db, emp.ID, emp.Ambiente, services.NFSeSeriePadrao)
		if err != nil {
			nfseErro(w, http.StatusInternalServerError, "erro ao reservar número do DPS")
			return
		}
		dados.NDPS = nDPS
		g, err := services.GerarDPS(dados)
		if err != nil {
			nfseErro(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
		xml, err := services.AssinarDPS(g, a1)
		if err != nil {
			nfseErro(w, http.StatusUnprocessableEntity, err.Error())
			return
		}

		var contrato any
		if req.ContratoID != "" {
			if !reUUIDNFS.MatchString(req.ContratoID) {
				nfseErro(w, http.StatusBadRequest, "contrato_id inválido")
				return
			}
			contrato = req.ContratoID
		}
		var id string
		err = db.QueryRowContext(ctx, `
			INSERT INTO financeiro.nfse_emitidas (empresa_id, cliente_id, contrato_id, ambiente, serie, n_dps, id_dps,
			  competencia, valor, descricao, status, dps_xml)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9::numeric/100,$10,'gerada',$11) RETURNING id`,
			emp.ID, req.ClienteID, contrato, emp.Ambiente, g.Serie, nDPS, g.ID, competencia.Format("2006-01-02"),
			req.ValorCentavos, strings.Join(strings.Fields(req.Descricao), " "), xml).Scan(&id)
		if err != nil {
			log.Printf("[NFSe] erro ao registrar DPS %s: %v", g.ID, err)
			var pqErr *pq.Error
			switch {
			case errors.As(err, &pqErr) && pqErr.Code == "23505" && strings.Contains(pqErr.Constraint, "contrato_competencia"):
				nfseErro(w, http.StatusConflict, "já existe uma nota viva para este contrato e competência")
			case errors.As(err, &pqErr):
				nfseErro(w, http.StatusInternalServerError, "erro ao registrar a nota no banco ("+string(pqErr.Code)+" "+pqErr.Constraint+")")
			default:
				nfseErro(w, http.StatusInternalServerError, "erro ao registrar a nota no banco")
			}
			return
		}

		cli, err := services.NovoClienteSefin(emp.Ambiente, a1)
		if err != nil {
			log.Printf("[NFSe] cliente Sefin: %v", err)
			marcarNFSe(ctx, db, id, "rejeitada", "", "", "", err.Error())
			nfseErro(w, http.StatusBadGateway, err.Error())
			return
		}
		db.ExecContext(ctx, `UPDATE financeiro.nfse_emitidas SET status='enviada', updated_at=NOW() WHERE id=$1`, id)
		log.Printf("[NFSe] enviando DPS %s (ambiente %d, valor %d centavos)", g.ID, emp.Ambiente, req.ValorCentavos)
		resp, err := cli.EmitirNFSe(ctx, xml)
		if err != nil {
			// Resultado desconhecido: a nota fica 'enviada' e deve ser reconciliada em /nfse/consultar.
			db.ExecContext(ctx, `UPDATE financeiro.nfse_emitidas SET erro=$2, updated_at=NOW() WHERE id=$1`, id, "falha de comunicação (resultado desconhecido, use Consultar): "+err.Error())
			log.Printf("[NFSe] falha de comunicação ao enviar DPS %s: %v", g.ID, err)
			nfseErro(w, http.StatusBadGateway, "falha de comunicação com a Sefin ("+err.Error()+"); o resultado é desconhecido — use Consultar antes de reemitir")
			return
		}
		if !resp.Autorizada() {
			marcarNFSe(ctx, db, id, "rejeitada", "", "", resp.Corpo, resp.MensagemErro())
			nfseJSON(w, http.StatusUnprocessableEntity, map[string]any{"id": id, "status": "rejeitada", "erro": resp.MensagemErro()})
			return
		}
		if resp.ChaveAcesso == "" {
			db.ExecContext(ctx, `UPDATE financeiro.nfse_emitidas SET nfse_xml=$2, erro=$3, updated_at=NOW() WHERE id=$1`, id, resp.NFSeXML, "resposta sem chave de acesso: use Consultar. Corpo: "+resp.Corpo)
			nfseErro(w, http.StatusBadGateway, "a Sefin respondeu sem chave de acesso; use Consultar para conferir")
			return
		}
		marcarNFSe(ctx, db, id, "autorizada", resp.ChaveAcesso, resp.NFSeXML, "", "")
		nfseJSON(w, http.StatusCreated, map[string]any{"id": id, "status": "autorizada", "chave_acesso": resp.ChaveAcesso, "n_nfse": numeroNFSe(resp.NFSeXML)})
	}
}

func numeroNFSe(xml string) any {
	if m := reNNFSe.FindStringSubmatch(xml); m != nil {
		return m[1]
	}
	return nil
}

func marcarNFSe(ctx context.Context, db *sql.DB, id, status, chave, nfseXML, corpo, erro string) {
	var chaveV any
	if chave != "" {
		chaveV = chave
	}
	erroFinal := erro
	if corpo != "" && erro != "" {
		erroFinal = erro + " | corpo: " + corpo
	}
	if _, err := db.ExecContext(ctx, `
		UPDATE financeiro.nfse_emitidas SET status=$2, chave_acesso=$3, n_nfse=NULLIF($4,'')::bigint,
		  nfse_xml=NULLIF($5,''), erro=NULLIF($6,''), updated_at=NOW() WHERE id=$1`,
		id, status, chaveV, fmt.Sprint(numeroNFSeStr(nfseXML)), nfseXML, erroFinal); err != nil {
		log.Printf("[NFSe] erro ao atualizar nota %s para %s: %v", id, status, err)
	}
}

func numeroNFSeStr(xml string) string {
	if m := reNNFSe.FindStringSubmatch(xml); m != nil {
		return m[1]
	}
	return ""
}

// NFSeListHandler lista as notas (sem XML) — GET /api/financeiro/nfse.
func NFSeListHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		rows, err := db.QueryContext(r.Context(), `
			SELECT n.id, COALESCE(c.razao_social,''), n.ambiente, n.n_dps, n.competencia, n.valor, n.status,
			       COALESCE(n.chave_acesso,''), COALESCE(n.n_nfse::text,''), COALESCE(n.erro,''), n.created_at
			FROM financeiro.nfse_emitidas n LEFT JOIN financeiro.clientes c ON c.id = n.cliente_id
			ORDER BY n.created_at DESC LIMIT 100`)
		if err != nil {
			nfseErro(w, http.StatusInternalServerError, "erro ao listar notas")
			return
		}
		defer rows.Close()
		out := []map[string]any{}
		for rows.Next() {
			var id, cli, status, chave, nn, erro string
			var amb int
			var ndps int64
			var comp, criada time.Time
			var valor float64
			if rows.Scan(&id, &cli, &amb, &ndps, &comp, &valor, &status, &chave, &nn, &erro, &criada) == nil {
				out = append(out, map[string]any{"id": id, "cliente": cli, "ambiente": amb, "n_dps": ndps,
					"competencia": comp.Format("2006-01-02"), "valor": valor, "status": status,
					"chave_acesso": chave, "n_nfse": nn, "erro": erro, "created_at": criada})
			}
		}
		nfseJSON(w, http.StatusOK, out)
	}
}

// NFSeXMLHandler baixa o XML guardado — GET /api/financeiro/nfse/xml?id=&tipo=dps|nfse|cancelamento.
func NFSeXMLHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, tipo := r.URL.Query().Get("id"), r.URL.Query().Get("tipo")
		col := map[string]string{"dps": "dps_xml", "nfse": "nfse_xml", "cancelamento": "cancel_xml"}[tipo]
		if col == "" || !reUUIDNFS.MatchString(id) {
			nfseErro(w, http.StatusBadRequest, "id e tipo (dps, nfse, cancelamento) são obrigatórios")
			return
		}
		var xml sql.NullString
		var chave sql.NullString
		err := db.QueryRowContext(r.Context(), `SELECT `+col+`, chave_acesso FROM financeiro.nfse_emitidas WHERE id = $1`, id).Scan(&xml, &chave)
		if err != nil || !xml.Valid {
			nfseErro(w, http.StatusNotFound, "XML não disponível")
			return
		}
		nome := tipo + "-" + id
		if chave.Valid {
			nome = tipo + "-" + chave.String
		}
		w.Header().Set("Content-Type", "application/xml; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="`+nome+`.xml"`)
		io.WriteString(w, xml.String)
	}
}

// NFSeConsultarHandler reconcilia uma nota 'enviada' com a Sefin — POST ?id=.
func NFSeConsultarHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		id := r.URL.Query().Get("id")
		if !reUUIDNFS.MatchString(id) {
			nfseErro(w, http.StatusBadRequest, "id inválido")
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
		defer cancel()
		var empID, idDPS string
		var amb int
		var chave sql.NullString
		if err := db.QueryRowContext(ctx, `SELECT empresa_id, id_dps, ambiente, chave_acesso FROM financeiro.nfse_emitidas WHERE id=$1`, id).Scan(&empID, &idDPS, &amb, &chave); err != nil {
			nfseErro(w, http.StatusNotFound, "nota não encontrada")
			return
		}
		a1, err := carregarA1(ctx, db, empID)
		if err != nil {
			nfseErro(w, http.StatusConflict, err.Error())
			return
		}
		cli, err := services.NovoClienteSefin(amb, a1)
		if err != nil {
			nfseErro(w, http.StatusBadGateway, err.Error())
			return
		}
		ch := chave.String
		if ch == "" {
			resp, err := cli.ConsultarDPS(ctx, idDPS)
			if err != nil {
				nfseErro(w, http.StatusBadGateway, err.Error())
				return
			}
			if !resp.Autorizada() || resp.ChaveAcesso == "" {
				nfseJSON(w, http.StatusOK, map[string]any{"encontrada": false, "detalhe": resp.MensagemErro()})
				return
			}
			ch = resp.ChaveAcesso
		}
		resp, err := cli.ConsultarNFSe(ctx, ch)
		if err != nil {
			nfseErro(w, http.StatusBadGateway, err.Error())
			return
		}
		if !resp.Autorizada() {
			nfseJSON(w, http.StatusOK, map[string]any{"encontrada": false, "detalhe": resp.MensagemErro()})
			return
		}
		marcarNFSe(ctx, db, id, "autorizada", ch, resp.NFSeXML, "", "")
		nfseJSON(w, http.StatusOK, map[string]any{"encontrada": true, "chave_acesso": ch})
	}
}

// NFSeCancelarHandler trata POST /api/financeiro/nfse/cancelar.
func NFSeCancelarHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			ID                string `json:"id"`
			Motivo            int    `json:"motivo"` // 1 erro na emissão, 2 serviço não prestado, 9 outros
			Justificativa     string `json:"justificativa"`
			ConfirmarProducao string `json:"confirmar_producao,omitempty"`
		}
		if json.NewDecoder(io.LimitReader(r.Body, 16<<10)).Decode(&req) != nil || !reUUIDNFS.MatchString(req.ID) {
			nfseErro(w, http.StatusBadRequest, "requisição inválida")
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
		defer cancel()
		var empID, status string
		var amb int
		var chave sql.NullString
		if err := db.QueryRowContext(ctx, `SELECT empresa_id, ambiente, status, chave_acesso FROM financeiro.nfse_emitidas WHERE id=$1`, req.ID).Scan(&empID, &amb, &status, &chave); err != nil {
			nfseErro(w, http.StatusNotFound, "nota não encontrada")
			return
		}
		if status != "autorizada" || !chave.Valid {
			nfseErro(w, http.StatusConflict, "só é possível cancelar nota autorizada")
			return
		}
		if amb == 1 && req.ConfirmarProducao != nfseFraseProducao {
			nfseErro(w, http.StatusBadRequest, "nota de PRODUÇÃO: envie confirmar_producao = \""+nfseFraseProducao+"\"")
			return
		}
		emp, err := carregarEmpresaNFSe(ctx, db)
		if err != nil {
			nfseErro(w, http.StatusConflict, "empresa emitente não encontrada")
			return
		}
		a1, err := carregarA1(ctx, db, empID)
		if err != nil {
			nfseErro(w, http.StatusConflict, err.Error())
			return
		}
		if err := services.ValidarCertificadoDoPrestador(a1.Cert, emp.CNPJ, time.Now()); err != nil {
			nfseErro(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
		ped, err := services.NovoPedidoCancelamento(amb, emp.CNPJ, chave.String, req.Motivo, req.Justificativa, time.Now().In(nfseBRT).Add(-time.Minute))
		if err != nil {
			nfseErro(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
		xml, err := services.AssinarPedidoCancelamento(ped, a1)
		if err != nil {
			nfseErro(w, http.StatusInternalServerError, err.Error())
			return
		}
		cli, err := services.NovoClienteSefin(amb, a1)
		if err != nil {
			nfseErro(w, http.StatusBadGateway, err.Error())
			return
		}
		log.Printf("[NFSe] cancelando nota %s (ambiente %d)", chave.String, amb)
		resp, err := cli.CancelarNFSe(ctx, chave.String, xml)
		if err != nil {
			nfseErro(w, http.StatusBadGateway, "falha de comunicação com a Sefin; use Consultar antes de tentar de novo")
			return
		}
		if !resp.Autorizada() {
			db.ExecContext(ctx, `UPDATE financeiro.nfse_emitidas SET erro=$2, updated_at=NOW() WHERE id=$1`, req.ID, "cancelamento: "+resp.MensagemErro())
			nfseJSON(w, http.StatusUnprocessableEntity, map[string]any{"id": req.ID, "status": "autorizada", "erro": resp.MensagemErro()})
			return
		}
		db.ExecContext(ctx, `UPDATE financeiro.nfse_emitidas SET status='cancelada', cancel_xml=$2, erro=NULL, updated_at=NOW() WHERE id=$1`, req.ID, xml)
		nfseJSON(w, http.StatusOK, map[string]string{"id": req.ID, "status": "cancelada"})
	}
}

// completarCadeiaNFSe é o ponto de chamada de services.CompletarCadeiaICPBrasil no upload.
// Variável não exportada só para os testes do pacote validarem contra uma raiz de teste (não
// existe chave de raiz ICP-Brasil real para gerar certificados de teste); em produção é sempre
// a função real.
var completarCadeiaNFSe = services.CompletarCadeiaICPBrasil

// mensagemErroCadeiaNFSe mapeia os erros da cadeia para HTTP. As mensagens nunca trazem detalhe
// técnico (URL, IP, saída do openssl): isso vai só para o log.
func mensagemErroCadeiaNFSe(err error) (int, string) {
	switch {
	case errors.Is(err, services.ErrNFSeCadeiaAmbienteIndisponivel):
		return http.StatusInternalServerError, "Não foi possível validar a cadeia do certificado — ambiente do servidor incompleto, contate o suporte"
	case errors.Is(err, services.ErrNFSeCadeiaDownloadFalhou):
		return http.StatusBadGateway, "Não foi possível baixar a cadeia de certificação da Autoridade Certificadora — tente novamente em alguns minutos"
	case errors.Is(err, services.ErrNFSeCadeiaVencida):
		return http.StatusBadRequest, "O certificado ou uma Autoridade Certificadora da cadeia está vencido — envie um certificado válido"
	case errors.Is(err, services.ErrNFSeCadeiaAIAEsquemaNaoSuportado):
		return http.StatusBadRequest, "O endereço da AC indicado no certificado usa um esquema não suportado (só http/https) — verifique com a sua certificadora"
	case errors.Is(err, services.ErrNFSeCadeiaAIAInvalida):
		return http.StatusBadRequest, "Não foi possível obter as ACs intermediárias pelo endereço indicado no certificado — verifique com a sua certificadora"
	case errors.Is(err, services.ErrNFSeCadeiaAIABloqueada):
		return http.StatusBadRequest, "O endereço da Autoridade Certificadora informado no certificado não é permitido"
	case errors.Is(err, services.ErrNFSeCadeiaNaoICPBrasil):
		return http.StatusBadRequest, "O certificado não é ICP-Brasil (a cadeia não termina numa raiz ICP-Brasil) — envie um certificado A1 e-CNPJ de AC credenciada"
	default:
		return http.StatusBadRequest, "O certificado não pôde ter a cadeia de certificação validada até uma raiz ICP-Brasil"
	}
}

// NFSeConvenioHandler consulta, nos dois ambientes, se o município do prestador está no cadastro
// de convênio do Sistema Nacional — GET /api/financeiro/nfse/convenio. Somente leitura.
func NFSeConvenioHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
		defer cancel()
		emp, err := carregarEmpresaNFSe(ctx, db)
		if err != nil || len(emp.CodIBGE) != 7 {
			nfseErro(w, http.StatusConflict, "cadastre a empresa e o IBGE do município antes")
			return
		}
		a1, err := carregarA1(ctx, db, emp.ID)
		if err != nil {
			nfseErro(w, http.StatusConflict, err.Error())
			return
		}
		out := map[string]any{"municipio": emp.CodIBGE}
		for amb, nome := range map[int]string{1: "producao", 2: "producao_restrita"} {
			cli, err := services.NovoClienteSefin(amb, a1)
			if err != nil {
				out[nome] = map[string]any{"erro": err.Error()}
				continue
			}
			resp, err := cli.ConsultarConvenio(ctx, emp.CodIBGE)
			if err != nil {
				out[nome] = map[string]any{"erro": err.Error()}
				continue
			}
			corpo := resp.Corpo
			if len([]rune(corpo)) > 1500 {
				corpo = string([]rune(corpo)[:1500]) + "…"
			}
			out[nome] = map[string]any{"http": resp.Status, "resposta": corpo}
		}
		nfseJSON(w, http.StatusOK, out)
	}
}
