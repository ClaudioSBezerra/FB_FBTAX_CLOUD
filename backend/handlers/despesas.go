package handlers

// Reembolso de despesas de viagem.
//
//	/api/financeiro/despesas
//	  GET   ?cliente_id=&status=         lista (com comprovante e nota)
//	  POST                               cria despesa
//	  PUT                                edita despesa que ainda não está em nota
//	  DELETE ?id=                        exclui despesa que ainda não está em nota
//	  POST  ?action=importar             importa lista (JSON) ligando comprovantes por nome de arquivo
//	  POST  ?action=status               {ids, status}: aprova/descarta/volta a pendente
//	/api/financeiro/despesas/comprovantes
//	  POST  multipart (campo "arquivo", vários)   guarda comprovantes (PDF/imagem)
//	  GET   ?id=                          baixa/mostra o comprovante
//	/api/financeiro/notas-debito
//	  GET                                lista notas
//	  GET   ?id=&action=xlsx|zip         baixa a planilha preenchida / os comprovantes da nota
//	  POST                               fecha a nota com as despesas aprovadas e gera o título 'reembolso'
//	  POST  ?action=cancelar             cancela a nota (e o título em aberto), liberando as despesas

import (
	"archive/zip"
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"fb_cloud/services"

	"github.com/lib/pq"
)

const (
	maxComprovanteBytes = 12 << 20
	maxDespesasImport   = 300
)

type despesaReq struct {
	ID            string  `json:"id,omitempty"`
	ClienteID     string  `json:"cliente_id"`
	Data          string  `json:"data"`
	TipoDespesa   string  `json:"tipo_despesa"`
	Descricao     string  `json:"descricao"`
	Valor         float64 `json:"valor"`
	ComprovanteID string  `json:"comprovante_id,omitempty"`
	Comprovante   string  `json:"comprovante,omitempty"` // nome do arquivo já enviado (importação)
	Status        string  `json:"status,omitempty"`
	Obs           string  `json:"obs,omitempty"`
}

func validarDespesa(r *despesaReq) string {
	if !reUUIDNFS.MatchString(r.ClienteID) {
		return "cliente obrigatório"
	}
	if _, err := time.Parse("2006-01-02", r.Data); err != nil {
		return "data deve ser AAAA-MM-DD"
	}
	if !services.TipoDespesaValido(r.TipoDespesa) {
		return "tipo de despesa fora da lista do modelo"
	}
	r.Descricao = strings.Join(strings.Fields(r.Descricao), " ")
	if r.Descricao == "" {
		return "descrição obrigatória"
	}
	if r.Valor <= 0 {
		return "valor deve ser maior que zero"
	}
	switch r.Status {
	case "":
		r.Status = "pendente"
	case "pendente", "aprovada", "descartada":
	default:
		return "status inválido"
	}
	if r.ComprovanteID != "" && !reUUIDNFS.MatchString(r.ComprovanteID) {
		return "comprovante inválido"
	}
	return ""
}

func nullUUID(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// DespesasHandler trata /api/financeiro/despesas.
func DespesasHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		action := r.URL.Query().Get("action")
		switch {
		case r.Method == http.MethodGet:
			despesasListar(w, r, db)
		case r.Method == http.MethodPost && action == "importar":
			despesasImportar(w, r, db)
		case r.Method == http.MethodPost && action == "status":
			despesasStatus(w, r, db)
		case r.Method == http.MethodPost && action == "ler":
			despesasLerIA(w, r, db)
		case r.Method == http.MethodPost:
			despesasCriar(w, r, db)
		case r.Method == http.MethodPut:
			despesasEditar(w, r, db)
		case r.Method == http.MethodDelete:
			despesasExcluir(w, r, db)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

func despesasListar(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	q := r.URL.Query()
	args := []any{}
	where := []string{"1=1"}
	if c := q.Get("cliente_id"); c != "" {
		if !reUUIDNFS.MatchString(c) {
			nfseErro(w, http.StatusBadRequest, "cliente inválido")
			return
		}
		args = append(args, c)
		where = append(where, fmt.Sprintf("d.cliente_id = $%d", len(args)))
	}
	if s := q.Get("status"); s != "" {
		args = append(args, s)
		where = append(where, fmt.Sprintf("d.status = $%d", len(args)))
	}
	rows, err := db.QueryContext(r.Context(), `
		SELECT d.id, d.cliente_id, d.data::text, d.tipo_despesa, d.descricao, d.valor::float8, d.status, COALESCE(d.obs,''),
		       COALESCE(d.comprovante_id::text,''), COALESCE(c.nome,''), COALESCE(d.nota_debito_id::text,''), COALESCE(n.codigo,'')
		FROM financeiro.despesas_viagem d
		LEFT JOIN financeiro.despesas_comprovantes c ON c.id = d.comprovante_id
		LEFT JOIN financeiro.notas_debito n ON n.id = d.nota_debito_id
		WHERE `+strings.Join(where, " AND ")+` ORDER BY d.data, d.created_at`, args...)
	if err != nil {
		log.Printf("[Despesas] listar: %v", err)
		nfseErro(w, http.StatusInternalServerError, "erro ao listar despesas")
		return
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id, cli, data, tipo, desc, status, obs, comp, compNome, nota, notaCod string
		var valor float64
		if err := rows.Scan(&id, &cli, &data, &tipo, &desc, &valor, &status, &obs, &comp, &compNome, &nota, &notaCod); err != nil {
			nfseErro(w, http.StatusInternalServerError, "erro ao ler despesas")
			return
		}
		out = append(out, map[string]any{"id": id, "cliente_id": cli, "data": data, "tipo_despesa": tipo, "descricao": desc,
			"valor": valor, "status": status, "obs": obs, "comprovante_id": comp, "comprovante_nome": compNome,
			"nota_debito_id": nota, "nota_codigo": notaCod})
	}
	// comprovantes enviados nos últimos 60 dias que ainda não geraram despesa (candidatos à leitura por IA)
	pend := []map[string]string{}
	pr, err := db.QueryContext(r.Context(), `SELECT c.id, c.nome FROM financeiro.despesas_comprovantes c
		WHERE c.created_at > NOW() - INTERVAL '60 days'
		  AND NOT EXISTS (SELECT 1 FROM financeiro.despesas_viagem d WHERE d.comprovante_id = c.id)
		ORDER BY c.nome`)
	if err == nil {
		for pr.Next() {
			var id, nome string
			if pr.Scan(&id, &nome) == nil {
				pend = append(pend, map[string]string{"id": id, "nome": nome})
			}
		}
		pr.Close()
	}
	nfseJSON(w, http.StatusOK, map[string]any{"despesas": out, "tipos": services.TiposDespesaViagem,
		"comprovantes_sem_despesa": pend, "ia_configurada": os.Getenv("ANTHROPIC_API_KEY") != ""})
}

func despesasCriar(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	var req despesaReq
	if json.NewDecoder(io.LimitReader(r.Body, 64<<10)).Decode(&req) != nil {
		nfseErro(w, http.StatusBadRequest, "requisição inválida")
		return
	}
	if msg := validarDespesa(&req); msg != "" {
		nfseErro(w, http.StatusBadRequest, msg)
		return
	}
	var id string
	err := db.QueryRowContext(r.Context(), `
		INSERT INTO financeiro.despesas_viagem (cliente_id, data, tipo_despesa, descricao, valor, comprovante_id, status, obs)
		VALUES ($1,$2,$3,$4,$5,$6,$7,NULLIF($8,'')) RETURNING id`,
		req.ClienteID, req.Data, req.TipoDespesa, req.Descricao, req.Valor, nullUUID(req.ComprovanteID), req.Status, req.Obs).Scan(&id)
	if err != nil {
		log.Printf("[Despesas] criar: %v", err)
		nfseErro(w, http.StatusInternalServerError, "erro ao criar despesa")
		return
	}
	nfseJSON(w, http.StatusCreated, map[string]string{"id": id})
}

func despesasEditar(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	var req despesaReq
	if json.NewDecoder(io.LimitReader(r.Body, 64<<10)).Decode(&req) != nil || !reUUIDNFS.MatchString(req.ID) {
		nfseErro(w, http.StatusBadRequest, "requisição inválida")
		return
	}
	if msg := validarDespesa(&req); msg != "" {
		nfseErro(w, http.StatusBadRequest, msg)
		return
	}
	res, err := db.ExecContext(r.Context(), `
		UPDATE financeiro.despesas_viagem SET data=$2, tipo_despesa=$3, descricao=$4, valor=$5, comprovante_id=$6,
		       status=$7, obs=NULLIF($8,''), updated_at=NOW()
		WHERE id=$1 AND nota_debito_id IS NULL`,
		req.ID, req.Data, req.TipoDespesa, req.Descricao, req.Valor, nullUUID(req.ComprovanteID), req.Status, req.Obs)
	if err != nil {
		nfseErro(w, http.StatusInternalServerError, "erro ao editar despesa")
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		nfseErro(w, http.StatusConflict, "despesa não encontrada ou já incluída em uma nota")
		return
	}
	nfseJSON(w, http.StatusOK, map[string]string{"id": req.ID})
}

func despesasExcluir(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	id := r.URL.Query().Get("id")
	if !reUUIDNFS.MatchString(id) {
		nfseErro(w, http.StatusBadRequest, "id inválido")
		return
	}
	res, err := db.ExecContext(r.Context(), `DELETE FROM financeiro.despesas_viagem WHERE id=$1 AND nota_debito_id IS NULL`, id)
	if err != nil {
		nfseErro(w, http.StatusInternalServerError, "erro ao excluir")
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		nfseErro(w, http.StatusConflict, "despesa não encontrada ou já incluída em uma nota")
		return
	}
	nfseJSON(w, http.StatusOK, map[string]string{"id": id})
}

func despesasStatus(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	var req struct {
		IDs    []string `json:"ids"`
		Status string   `json:"status"`
	}
	if json.NewDecoder(io.LimitReader(r.Body, 256<<10)).Decode(&req) != nil || len(req.IDs) == 0 || len(req.IDs) > 500 {
		nfseErro(w, http.StatusBadRequest, "requisição inválida")
		return
	}
	if req.Status != "pendente" && req.Status != "aprovada" && req.Status != "descartada" {
		nfseErro(w, http.StatusBadRequest, "status inválido")
		return
	}
	for _, id := range req.IDs {
		if !reUUIDNFS.MatchString(id) {
			nfseErro(w, http.StatusBadRequest, "id inválido")
			return
		}
	}
	res, err := db.ExecContext(r.Context(), `UPDATE financeiro.despesas_viagem SET status=$2, updated_at=NOW()
		WHERE id = ANY($1::uuid[]) AND nota_debito_id IS NULL`, pq.Array(req.IDs), req.Status)
	if err != nil {
		nfseErro(w, http.StatusInternalServerError, "erro ao atualizar")
		return
	}
	n, _ := res.RowsAffected()
	nfseJSON(w, http.StatusOK, map[string]int64{"atualizadas": n})
}

func despesasImportar(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	var req struct {
		ClienteID string       `json:"cliente_id"`
		Despesas  []despesaReq `json:"despesas"`
	}
	if json.NewDecoder(io.LimitReader(r.Body, 2<<20)).Decode(&req) != nil || !reUUIDNFS.MatchString(req.ClienteID) ||
		len(req.Despesas) == 0 || len(req.Despesas) > maxDespesasImport {
		nfseErro(w, http.StatusBadRequest, "informe cliente_id e de 1 a 300 despesas")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		nfseErro(w, http.StatusInternalServerError, "erro interno")
		return
	}
	defer tx.Rollback()
	semComprovante := []string{}
	for i := range req.Despesas {
		d := &req.Despesas[i]
		d.ClienteID = req.ClienteID
		if d.Comprovante != "" {
			var cid string
			err := tx.QueryRowContext(ctx, `SELECT id FROM financeiro.despesas_comprovantes WHERE nome = $1 ORDER BY created_at DESC LIMIT 1`, d.Comprovante).Scan(&cid)
			if err == nil {
				d.ComprovanteID = cid
			} else if err == sql.ErrNoRows {
				semComprovante = append(semComprovante, d.Comprovante)
			} else {
				nfseErro(w, http.StatusInternalServerError, "erro ao buscar comprovante")
				return
			}
		}
		if msg := validarDespesa(d); msg != "" {
			nfseErro(w, http.StatusBadRequest, fmt.Sprintf("despesa %d: %s", i+1, msg))
			return
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO financeiro.despesas_viagem (cliente_id, data, tipo_despesa, descricao, valor, comprovante_id, status, obs)
			VALUES ($1,$2,$3,$4,$5,$6,$7,NULLIF($8,''))`,
			d.ClienteID, d.Data, d.TipoDespesa, d.Descricao, d.Valor, nullUUID(d.ComprovanteID), d.Status, d.Obs); err != nil {
			log.Printf("[Despesas] importar %d: %v", i+1, err)
			nfseErro(w, http.StatusInternalServerError, "erro ao importar despesas")
			return
		}
	}
	if err := tx.Commit(); err != nil {
		nfseErro(w, http.StatusInternalServerError, "erro ao importar despesas")
		return
	}
	nfseJSON(w, http.StatusCreated, map[string]any{"importadas": len(req.Despesas), "comprovantes_nao_encontrados": uniq(semComprovante)})
}

// lerReciboIA é variável só para os testes não chamarem a API de verdade.
var lerReciboIA = services.LerReciboIA

// despesasLerIA lê UM comprovante com IA e grava as despesas encontradas como pendentes. A tela
// chama um arquivo por vez: uma leitura leva dezenas de segundos e o proxy corta pedidos longos.
func despesasLerIA(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	var req struct {
		ClienteID     string `json:"cliente_id"`
		ComprovanteID string `json:"comprovante_id"`
		Reler         bool   `json:"reler"`
	}
	if json.NewDecoder(io.LimitReader(r.Body, 8<<10)).Decode(&req) != nil || !reUUIDNFS.MatchString(req.ClienteID) || !reUUIDNFS.MatchString(req.ComprovanteID) {
		nfseErro(w, http.StatusBadRequest, "informe o cliente e o comprovante")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 100*time.Second)
	defer cancel()
	var nome, ct string
	var dados []byte
	if err := db.QueryRowContext(ctx, `SELECT nome, content_type, dados FROM financeiro.despesas_comprovantes WHERE id=$1`, req.ComprovanteID).Scan(&nome, &ct, &dados); err != nil {
		nfseErro(w, http.StatusNotFound, "comprovante não encontrado")
		return
	}
	if !req.Reler {
		var ja int
		db.QueryRowContext(ctx, `SELECT COUNT(*) FROM financeiro.despesas_viagem WHERE comprovante_id=$1`, req.ComprovanteID).Scan(&ja)
		if ja > 0 {
			nfseErro(w, http.StatusConflict, nome+": já foi lido ("+fmt.Sprint(ja)+" despesa(s) ligada(s) a ele)")
			return
		}
	}
	lidas, err := lerReciboIA(ctx, nome, ct, dados)
	if err != nil {
		log.Printf("[Despesas] leitura IA de %s: %v", nome, err)
		if errors.Is(err, services.ErrIANaoConfigurada) {
			nfseErro(w, http.StatusConflict, err.Error())
			return
		}
		// 424 e não 502/504: o Cloudflare troca o corpo desses dois e a mensagem some.
		nfseErro(w, http.StatusFailedDependency, nome+": não consegui ler agora ("+err.Error()+")")
		return
	}
	ignoradas := []string{}
	gravadas := 0
	for _, d := range lidas {
		desc := strings.Join(strings.Fields(d.Descricao), " ")
		if _, err := time.Parse("2006-01-02", d.Data); err != nil || d.Valor <= 0 || desc == "" || !services.TipoDespesaValido(d.TipoDespesa) {
			ignoradas = append(ignoradas, fmt.Sprintf("%s (R$ %.2f): data ou valor ilegível, cadastre à mão", desc, d.Valor))
			continue
		}
		obs := strings.TrimSpace(d.Obs)
		var igual int
		db.QueryRowContext(ctx, `SELECT COUNT(*) FROM financeiro.despesas_viagem WHERE cliente_id=$1 AND data=$2 AND valor=$3::numeric AND tipo_despesa=$4`,
			req.ClienteID, d.Data, d.Valor, d.TipoDespesa).Scan(&igual)
		if igual > 0 {
			obs = strings.TrimSpace(obs + " · possível duplicidade: já há despesa igual")
		}
		obs = strings.TrimSpace(obs + " · lido por IA, confira")
		if _, err := db.ExecContext(ctx, `
			INSERT INTO financeiro.despesas_viagem (cliente_id, data, tipo_despesa, descricao, valor, comprovante_id, status, obs)
			VALUES ($1,$2,$3,$4,$5,$6,'pendente',$7)`, req.ClienteID, d.Data, d.TipoDespesa, desc, d.Valor, req.ComprovanteID, strings.TrimPrefix(obs, " · ")); err != nil {
			log.Printf("[Despesas] gravar leitura: %v", err)
			nfseErro(w, http.StatusInternalServerError, "erro ao gravar as despesas lidas")
			return
		}
		gravadas++
	}
	nfseJSON(w, http.StatusOK, map[string]any{"arquivo": nome, "lidas": gravadas, "ignoradas": ignoradas})
}

func uniq(s []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, x := range s {
		if !seen[x] {
			seen[x] = true
			out = append(out, x)
		}
	}
	return out
}

// DespesasComprovantesHandler trata /api/financeiro/despesas/comprovantes.
func DespesasComprovantesHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			r.Body = http.MaxBytesReader(w, r.Body, 64<<20)
			if err := r.ParseMultipartForm(16 << 20); err != nil {
				nfseErro(w, http.StatusBadRequest, "envio inválido ou maior que 64 MB")
				return
			}
			files := r.MultipartForm.File["arquivo"]
			if len(files) == 0 || len(files) > 40 {
				nfseErro(w, http.StatusBadRequest, "envie de 1 a 40 arquivos no campo \"arquivo\"")
				return
			}
			out := []map[string]string{}
			for _, fh := range files {
				f, err := fh.Open()
				if err != nil {
					nfseErro(w, http.StatusBadRequest, "arquivo ilegível")
					return
				}
				dados, err := io.ReadAll(io.LimitReader(f, maxComprovanteBytes+1))
				f.Close()
				if err != nil || len(dados) == 0 || len(dados) > maxComprovanteBytes {
					nfseErro(w, http.StatusBadRequest, fh.Filename+": vazio ou maior que 12 MB")
					return
				}
				ct := http.DetectContentType(dados)
				if ct != "application/pdf" && ct != "image/png" && ct != "image/jpeg" && ct != "image/webp" {
					nfseErro(w, http.StatusBadRequest, fh.Filename+": só PDF, PNG, JPEG ou WebP")
					return
				}
				nome := strings.ReplaceAll(strings.ReplaceAll(fh.Filename, "/", "_"), "\\", "_")
				var id string
				if err := db.QueryRowContext(r.Context(), `INSERT INTO financeiro.despesas_comprovantes (nome, content_type, dados) VALUES ($1,$2,$3) RETURNING id`, nome, ct, dados).Scan(&id); err != nil {
					log.Printf("[Despesas] comprovante: %v", err)
					nfseErro(w, http.StatusInternalServerError, "erro ao guardar comprovante")
					return
				}
				out = append(out, map[string]string{"id": id, "nome": nome})
			}
			nfseJSON(w, http.StatusCreated, map[string]any{"comprovantes": out})
		case http.MethodGet:
			id := r.URL.Query().Get("id")
			if !reUUIDNFS.MatchString(id) {
				nfseErro(w, http.StatusBadRequest, "id inválido")
				return
			}
			var nome, ct string
			var dados []byte
			if err := db.QueryRowContext(r.Context(), `SELECT nome, content_type, dados FROM financeiro.despesas_comprovantes WHERE id=$1`, id).Scan(&nome, &ct, &dados); err != nil {
				nfseErro(w, http.StatusNotFound, "comprovante não encontrado")
				return
			}
			w.Header().Set("Content-Type", ct)
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=%q", nome))
			w.Write(dados)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

type notaDebitoReq struct {
	ClienteID         string   `json:"cliente_id"`
	DespesaIDs        []string `json:"despesa_ids"`
	Destino           string   `json:"destino"`
	CentroCustoDesc   string   `json:"centro_custo_desc"`
	CentroCustoNum    string   `json:"centro_custo_num"`
	ContaContabilDesc string   `json:"conta_contabil_desc"`
	ContaContabilNum  string   `json:"conta_contabil_num"`
	Solicitante       string   `json:"solicitante"`
	Emissao           string   `json:"emissao,omitempty"` // AAAA-MM-DD; vazio = hoje
}

// NotasDebitoHandler trata /api/financeiro/notas-debito.
func NotasDebitoHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		action := r.URL.Query().Get("action")
		switch {
		case r.Method == http.MethodGet && (action == "xlsx" || action == "zip"):
			notaDebitoBaixar(w, r, db, action)
		case r.Method == http.MethodGet && action == "email-previa":
			notaDebitoEmailPrevia(w, r, db)
		case r.Method == http.MethodPost && action == "enviar":
			notaDebitoEnviar(w, r, db)
		case r.Method == http.MethodGet:
			notaDebitoListar(w, r, db)
		case r.Method == http.MethodPost && action == "cancelar":
			notaDebitoCancelar(w, r, db)
		case r.Method == http.MethodPost:
			notaDebitoCriar(w, r, db)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

func notaDebitoListar(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	rows, err := db.QueryContext(r.Context(), `
		SELECT n.id, n.cliente_id, c.razao_social, n.codigo, n.emissao::text, n.vencimento::text, n.periodo_ini::text, n.periodo_fim::text,
		       n.total::float8, n.status, COALESCE(n.titulo_id::text,''), COALESCE(t.status,''), COALESCE(n.enviada_em::text,''),
		       (SELECT COUNT(*) FROM financeiro.despesas_viagem d WHERE d.nota_debito_id = n.id)
		FROM financeiro.notas_debito n
		JOIN financeiro.clientes c ON c.id = n.cliente_id
		LEFT JOIN financeiro.titulos_receber t ON t.id = n.titulo_id
		ORDER BY n.created_at DESC LIMIT 200`)
	if err != nil {
		nfseErro(w, http.StatusInternalServerError, "erro ao listar notas")
		return
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id, cli, razao, cod, em, venc, ini, fim, status, tit, titStatus, enviada string
		var total float64
		var n int
		if err := rows.Scan(&id, &cli, &razao, &cod, &em, &venc, &ini, &fim, &total, &status, &tit, &titStatus, &enviada, &n); err != nil {
			nfseErro(w, http.StatusInternalServerError, "erro ao ler notas")
			return
		}
		out = append(out, map[string]any{"id": id, "cliente_id": cli, "cliente": razao, "codigo": cod, "emissao": em, "vencimento": venc,
			"periodo_ini": ini, "periodo_fim": fim, "total": total, "status": status, "titulo_id": tit, "titulo_status": titStatus, "enviada_em": enviada, "despesas": n})
	}
	nfseJSON(w, http.StatusOK, map[string]any{"notas": out})
}

func enderecoCompleto(logr, num, cpl, bairro, mun, uf, cep sql.NullString) string {
	partes := []string{}
	if logr.String != "" {
		l := logr.String
		if num.String != "" {
			l += ", " + num.String
		}
		if cpl.String != "" {
			l += " - " + cpl.String
		}
		partes = append(partes, l)
	}
	if bairro.String != "" {
		partes = append(partes, bairro.String)
	}
	if mun.String != "" {
		m := mun.String
		if uf.String != "" {
			m += " - " + uf.String
		}
		partes = append(partes, m)
	}
	s := strings.Join(partes, " - ")
	if cep.String != "" {
		s += ". CEP " + cep.String
	}
	return s
}

func notaDebitoCriar(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	var req notaDebitoReq
	if json.NewDecoder(io.LimitReader(r.Body, 256<<10)).Decode(&req) != nil || !reUUIDNFS.MatchString(req.ClienteID) ||
		len(req.DespesaIDs) == 0 || len(req.DespesaIDs) > 500 {
		nfseErro(w, http.StatusBadRequest, "informe o cliente e as despesas da nota")
		return
	}
	for _, id := range req.DespesaIDs {
		if !reUUIDNFS.MatchString(id) {
			nfseErro(w, http.StatusBadRequest, "despesa inválida")
			return
		}
	}
	emissao := time.Now().In(nfseBRT)
	emissao = time.Date(emissao.Year(), emissao.Month(), emissao.Day(), 0, 0, 0, 0, time.UTC)
	if req.Emissao != "" {
		e, err := time.Parse("2006-01-02", req.Emissao)
		if err != nil {
			nfseErro(w, http.StatusBadRequest, "emissao deve ser AAAA-MM-DD")
			return
		}
		emissao = e
	}
	vencimento := emissao.AddDate(0, 0, services.NotaDebitoPrazoDias)
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		nfseErro(w, http.StatusInternalServerError, "erro interno")
		return
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx, `SELECT id, data, tipo_despesa, descricao, valor::float8, status, nota_debito_id IS NOT NULL
		FROM financeiro.despesas_viagem WHERE id = ANY($1::uuid[]) AND cliente_id = $2 FOR UPDATE`, pq.Array(req.DespesaIDs), req.ClienteID)
	if err != nil {
		nfseErro(w, http.StatusInternalServerError, "erro ao ler despesas")
		return
	}
	var ds []services.DespesaNota
	var ini, fim time.Time
	for rows.Next() {
		var id, tipo, desc, status string
		var data time.Time
		var valor float64
		var emNota bool
		if err := rows.Scan(&id, &data, &tipo, &desc, &valor, &status, &emNota); err != nil {
			rows.Close()
			nfseErro(w, http.StatusInternalServerError, "erro ao ler despesas")
			return
		}
		if status != "aprovada" || emNota {
			rows.Close()
			nfseErro(w, http.StatusConflict, "só despesas aprovadas e ainda fora de nota entram na nota: confira "+desc)
			return
		}
		ds = append(ds, services.DespesaNota{Data: data, Tipo: tipo, Descricao: desc, Valor: valor})
		if ini.IsZero() || data.Before(ini) {
			ini = data
		}
		if data.After(fim) {
			fim = data
		}
	}
	rows.Close()
	if len(ds) != len(req.DespesaIDs) {
		nfseErro(w, http.StatusConflict, "alguma despesa não pertence ao cliente ou não existe")
		return
	}
	linhas := services.ConsolidarDespesas(ds)

	var emp struct{ nome, cnpj, fone, ie, banco, agencia, conta string }
	var fone, im, logr, num, cpl, bairro, mun, uf, cep, banco, agencia, conta sql.NullString
	if err := tx.QueryRowContext(ctx, `SELECT razao_social, cnpj, telefone, inscricao_municipal, logradouro, numero, complemento, bairro, municipio, uf, cep
		FROM financeiro.empresas ORDER BY created_at LIMIT 1`).Scan(&emp.nome, &emp.cnpj, &fone, &im, &logr, &num, &cpl, &bairro, &mun, &uf, &cep); err != nil {
		nfseErro(w, http.StatusConflict, "cadastre a empresa emitente antes (Financeiro > Empresa)")
		return
	}
	enderecoEmp := enderecoCompleto(logr, num, cpl, bairro, mun, uf, cep)
	tx.QueryRowContext(ctx, `SELECT banco, agencia, conta FROM financeiro.dados_bancarios ORDER BY created_at LIMIT 1`).Scan(&banco, &agencia, &conta)

	var cli struct{ nome, cnpj string }
	var ie sql.NullString
	if err := tx.QueryRowContext(ctx, `SELECT razao_social, cnpj, logradouro, numero, complemento, bairro, municipio, uf, cep
		FROM financeiro.clientes WHERE id = $1`, req.ClienteID).Scan(&cli.nome, &cli.cnpj, &logr, &num, &cpl, &bairro, &mun, &uf, &cep); err != nil {
		nfseErro(w, http.StatusNotFound, "cliente não encontrado")
		return
	}
	enderecoCli := enderecoCompleto(logr, num, cpl, bairro, mun, uf, cep)

	var existentes []string
	cr, err := tx.QueryContext(ctx, `SELECT codigo FROM financeiro.notas_debito WHERE codigo LIKE $1`, "FOR.FIN.0002-"+emissao.Format("012006")+"%")
	if err == nil {
		for cr.Next() {
			var c string
			cr.Scan(&c)
			existentes = append(existentes, c)
		}
		cr.Close()
	}
	codigo := services.CodigoNotaDebito(emissao, existentes)

	dados := services.DadosNotaDebito{
		Emissao: emissao, Vencimento: vencimento, PeriodoIni: ini, PeriodoFim: fim, Destino: strings.TrimSpace(req.Destino),
		EmissorNome: emp.nome, EmissorCNPJ: emp.cnpj, EmissorFone: fone.String, EmissorIE: im.String, EmissorEndereco: enderecoEmp,
		Banco: banco.String, Agencia: agencia.String, Conta: conta.String,
		ReceptorNome: cli.nome, ReceptorCNPJ: cli.cnpj, ReceptorIE: ie.String, ReceptorEndereco: enderecoCli,
		CentroCustoDesc: req.CentroCustoDesc, CentroCustoNum: req.CentroCustoNum,
		ContaContabilDesc: req.ContaContabilDesc, ContaContabilNum: req.ContaContabilNum,
		Solicitante: strings.TrimSpace(req.Solicitante), Linhas: linhas,
	}
	xlsx, err := services.GerarNotaDebitoXLSX(dados)
	if err != nil {
		nfseErro(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	total := dados.Total()

	var notaID, tituloID string
	if err := tx.QueryRowContext(ctx, `
		INSERT INTO financeiro.notas_debito (cliente_id, codigo, emissao, vencimento, destino, periodo_ini, periodo_fim, solicitante, total, xlsx)
		VALUES ($1,$2,$3,$4,NULLIF($5,''),$6,$7,NULLIF($8,''),$9::numeric,$10) RETURNING id`,
		req.ClienteID, codigo, emissao, vencimento, dados.Destino, ini, fim, dados.Solicitante, total, xlsx).Scan(&notaID); err != nil {
		log.Printf("[NotaDebito] inserir: %v", err)
		nfseErro(w, http.StatusInternalServerError, "erro ao registrar a nota")
		return
	}
	if err := tx.QueryRowContext(ctx, `
		INSERT INTO financeiro.titulos_receber (cliente_id, tipo, descricao, valor, vencimento)
		VALUES ($1,'reembolso',$2,$3::numeric,$4) RETURNING id`,
		req.ClienteID, "Reembolso de despesas de viagem — nota de débito "+codigo, total, vencimento).Scan(&tituloID); err != nil {
		log.Printf("[NotaDebito] título: %v", err)
		nfseErro(w, http.StatusInternalServerError, "erro ao criar o título do reembolso")
		return
	}
	if _, err := tx.ExecContext(ctx, `UPDATE financeiro.notas_debito SET titulo_id=$2 WHERE id=$1`, notaID, tituloID); err != nil {
		nfseErro(w, http.StatusInternalServerError, "erro ao ligar o título")
		return
	}
	if _, err := tx.ExecContext(ctx, `UPDATE financeiro.despesas_viagem SET nota_debito_id=$2, updated_at=NOW() WHERE id = ANY($1::uuid[])`, pq.Array(req.DespesaIDs), notaID); err != nil {
		nfseErro(w, http.StatusInternalServerError, "erro ao fechar as despesas")
		return
	}
	if err := tx.Commit(); err != nil {
		nfseErro(w, http.StatusInternalServerError, "erro ao gravar a nota")
		return
	}
	nfseJSON(w, http.StatusCreated, map[string]any{"id": notaID, "codigo": codigo, "total": total, "titulo_id": tituloID,
		"vencimento": vencimento.Format("2006-01-02"), "linhas": len(linhas)})
}

func notaDebitoBaixar(w http.ResponseWriter, r *http.Request, db *sql.DB, action string) {
	id := r.URL.Query().Get("id")
	if !reUUIDNFS.MatchString(id) {
		nfseErro(w, http.StatusBadRequest, "id inválido")
		return
	}
	var codigo string
	var xlsx []byte
	if err := db.QueryRowContext(r.Context(), `SELECT codigo, xlsx FROM financeiro.notas_debito WHERE id=$1`, id).Scan(&codigo, &xlsx); err != nil {
		nfseErro(w, http.StatusNotFound, "nota não encontrada")
		return
	}
	if action == "xlsx" {
		w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", codigo+".xlsx"))
		w.Write(xlsx)
		return
	}
	zipBytes, err := zipComprovantesNota(r.Context(), db, id)
	if err != nil {
		nfseErro(w, http.StatusInternalServerError, "erro ao montar o zip dos comprovantes")
		return
	}
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", codigo+"-comprovantes.zip"))
	w.Write(zipBytes)
}

// zipComprovantesNota reúne os comprovantes das despesas da nota (cada arquivo uma vez só).
func zipComprovantesNota(ctx context.Context, db *sql.DB, notaID string) ([]byte, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT DISTINCT c.id, c.nome, c.dados FROM financeiro.despesas_viagem d
		JOIN financeiro.despesas_comprovantes c ON c.id = d.comprovante_id WHERE d.nota_debito_id = $1 ORDER BY c.nome, c.id`, notaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	usados := map[string]bool{}
	n := 0
	for rows.Next() {
		var cid, nome string
		var dados []byte
		if err := rows.Scan(&cid, &nome, &dados); err != nil {
			return nil, err
		}
		if usados[nome] {
			nome = cid[:8] + "_" + nome
		}
		usados[nome] = true
		fw, err := zw.Create(nome)
		if err != nil {
			return nil, err
		}
		fw.Write(dados)
		n++
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, nil
	}
	return buf.Bytes(), nil
}

// enviarEmailNota é variável só para os testes não falarem com o SMTP de verdade.
var enviarEmailNota = services.EnviarEmailComAnexos

type notaEmailInfo struct {
	codigo, status, solicitante, cliente, clienteEmail, empresa, banco, agencia, conta string
	emissao, vencimento, ini, fim                                                      time.Time
	total                                                                              float64
	enviadaEm                                                                          sql.NullTime
	enviadaPara                                                                        sql.NullString
}

func carregarNotaEmail(ctx context.Context, db *sql.DB, id string) (*notaEmailInfo, error) {
	var n notaEmailInfo
	var sol, mail, banco, ag, conta, empresa sql.NullString
	err := db.QueryRowContext(ctx, `
		SELECT n.codigo, n.status, n.solicitante, c.razao_social, c.email, n.emissao, n.vencimento, n.periodo_ini, n.periodo_fim,
		       n.total::float8, n.enviada_em, n.enviada_para
		FROM financeiro.notas_debito n JOIN financeiro.clientes c ON c.id = n.cliente_id WHERE n.id = $1`, id).Scan(
		&n.codigo, &n.status, &sol, &n.cliente, &mail, &n.emissao, &n.vencimento, &n.ini, &n.fim, &n.total, &n.enviadaEm, &n.enviadaPara)
	if err != nil {
		return nil, err
	}
	n.solicitante, n.clienteEmail = sol.String, mail.String
	db.QueryRowContext(ctx, `SELECT razao_social FROM financeiro.empresas ORDER BY created_at LIMIT 1`).Scan(&empresa)
	db.QueryRowContext(ctx, `SELECT banco, agencia, conta FROM financeiro.dados_bancarios ORDER BY created_at LIMIT 1`).Scan(&banco, &ag, &conta)
	n.empresa, n.banco, n.agencia, n.conta = empresa.String, banco.String, ag.String, conta.String
	return &n, nil
}

func (n *notaEmailInfo) assuntoPadrao() string {
	return "Nota de Débito " + n.codigo + " — Reembolso de despesas de viagem"
}

func (n *notaEmailInfo) mensagemPadrao(comComprovantes bool) string {
	var b strings.Builder
	b.WriteString("Prezados,\n\n")
	b.WriteString(fmt.Sprintf("Segue em anexo a Nota de Débito %s, referente ao reembolso das despesas de viagem do período de %s a %s, no valor total de %s, com vencimento em %s.\n\n",
		n.codigo, n.ini.Format("02/01/2006"), n.fim.Format("02/01/2006"), brlTexto(n.total), n.vencimento.Format("02/01/2006")))
	if comComprovantes {
		b.WriteString("Os comprovantes das despesas seguem no arquivo ZIP anexo.\n\n")
	}
	if n.banco != "" || n.conta != "" {
		b.WriteString(fmt.Sprintf("Dados para pagamento: banco %s, agência %s, conta corrente %s.\n\n", n.banco, n.agencia, n.conta))
	}
	b.WriteString("Atenciosamente,\n")
	if n.solicitante != "" {
		b.WriteString(n.solicitante + "\n")
	}
	b.WriteString(n.empresa + "\n")
	return b.String()
}

func brlTexto(v float64) string {
	s := fmt.Sprintf("%.2f", v)
	inteiro, dec := s[:len(s)-3], s[len(s)-2:]
	var out []byte
	for i, c := range []byte(inteiro) {
		if i > 0 && (len(inteiro)-i)%3 == 0 {
			out = append(out, '.')
		}
		out = append(out, c)
	}
	return "R$ " + string(out) + "," + dec
}

func notaDebitoEmailPrevia(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	id := r.URL.Query().Get("id")
	if !reUUIDNFS.MatchString(id) {
		nfseErro(w, http.StatusBadRequest, "id inválido")
		return
	}
	n, err := carregarNotaEmail(r.Context(), db, id)
	if err != nil {
		nfseErro(w, http.StatusNotFound, "nota não encontrada")
		return
	}
	out := map[string]any{"assunto": n.assuntoPadrao(), "mensagem": n.mensagemPadrao(true), "para": n.clienteEmail,
		"remetente": os.Getenv("SMTP_USER"), "cc": os.Getenv("SMTP_USER"), "smtp_configurado": services.GetEmailConfig().Password != "", "status": n.status}
	if n.enviadaEm.Valid {
		out["enviada_em"], out["enviada_para"] = n.enviadaEm.Time.In(nfseBRT).Format("02/01/2006 15:04"), n.enviadaPara.String
	}
	nfseJSON(w, http.StatusOK, out)
}

func notaDebitoEnviar(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	var req struct {
		ID                 string   `json:"id"`
		Remetente          string   `json:"remetente"` // vira Reply-To: o envio sai pela conta do SMTP
		Para               []string `json:"para"`
		Cc                 []string `json:"cc"`
		Assunto            string   `json:"assunto"`
		Mensagem           string   `json:"mensagem"`
		IncluirComprovante bool     `json:"incluir_comprovantes"`
		ConfirmarReenvio   bool     `json:"confirmar_reenvio"`
	}
	if json.NewDecoder(io.LimitReader(r.Body, 128<<10)).Decode(&req) != nil || !reUUIDNFS.MatchString(req.ID) {
		nfseErro(w, http.StatusBadRequest, "requisição inválida")
		return
	}
	para, err := services.ValidarEnderecos(req.Para)
	if err != nil || len(para) == 0 {
		nfseErro(w, http.StatusBadRequest, "informe um e-mail de destino válido")
		return
	}
	cc, err := services.ValidarEnderecos(req.Cc)
	if err != nil {
		nfseErro(w, http.StatusBadRequest, err.Error())
		return
	}
	remetente, err := services.ValidarEnderecos([]string{req.Remetente})
	if err != nil {
		nfseErro(w, http.StatusBadRequest, "remetente inválido")
		return
	}
	responderPara := ""
	if len(remetente) == 1 {
		responderPara = remetente[0]
	}
	if len(para)+len(cc) > 10 {
		nfseErro(w, http.StatusBadRequest, "no máximo 10 destinatários")
		return
	}
	n, err := carregarNotaEmail(r.Context(), db, req.ID)
	if err != nil {
		nfseErro(w, http.StatusNotFound, "nota não encontrada")
		return
	}
	if n.status != "emitida" {
		nfseErro(w, http.StatusConflict, "a nota está cancelada: não pode ser enviada")
		return
	}
	if n.enviadaEm.Valid && !req.ConfirmarReenvio {
		nfseErro(w, http.StatusConflict, "esta nota já foi enviada em "+n.enviadaEm.Time.In(nfseBRT).Format("02/01/2006 15:04")+" para "+n.enviadaPara.String+": confirme o reenvio")
		return
	}
	assunto := strings.TrimSpace(req.Assunto)
	if assunto == "" {
		assunto = n.assuntoPadrao()
	}
	mensagem := strings.TrimSpace(req.Mensagem)
	if mensagem == "" {
		mensagem = n.mensagemPadrao(req.IncluirComprovante)
	}
	var xlsx []byte
	if err := db.QueryRowContext(r.Context(), `SELECT xlsx FROM financeiro.notas_debito WHERE id=$1`, req.ID).Scan(&xlsx); err != nil {
		nfseErro(w, http.StatusInternalServerError, "erro ao ler a planilha")
		return
	}
	anexos := []services.AnexoEmail{{Nome: n.codigo + ".xlsx", Tipo: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", Dados: xlsx}}
	if req.IncluirComprovante {
		z, err := zipComprovantesNota(r.Context(), db, req.ID)
		if err != nil {
			nfseErro(w, http.StatusInternalServerError, "erro ao montar o zip dos comprovantes")
			return
		}
		if z != nil {
			anexos = append(anexos, services.AnexoEmail{Nome: n.codigo + "-comprovantes.zip", Tipo: "application/zip", Dados: z})
		}
	}
	if err := enviarEmailNota(para, cc, responderPara, assunto, mensagem, anexos); err != nil {
		log.Printf("[NotaDebito] envio de %s falhou: %v", n.codigo, err)
		switch {
		case errors.Is(err, services.ErrSMTPNaoConfigurado):
			nfseErro(w, http.StatusConflict, "o e-mail do sistema não está configurado (SMTP_PASSWORD)")
		case errors.Is(err, services.ErrAnexosGrandes):
			nfseErro(w, http.StatusRequestEntityTooLarge, err.Error())
		default:
			nfseErro(w, http.StatusFailedDependency, "o servidor de e-mail recusou o envio; confira as configurações de SMTP")
		}
		return
	}
	db.ExecContext(r.Context(), `UPDATE financeiro.notas_debito SET enviada_em=NOW(), enviada_para=$2 WHERE id=$1`, req.ID, strings.Join(append(append([]string{}, para...), cc...), ", "))
	nfseJSON(w, http.StatusOK, map[string]any{"enviada": true, "remetente": responderPara, "para": para, "cc": cc, "anexos": len(anexos)})
}

func notaDebitoCancelar(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	var req struct {
		ID string `json:"id"`
	}
	if json.NewDecoder(io.LimitReader(r.Body, 8<<10)).Decode(&req) != nil || !reUUIDNFS.MatchString(req.ID) {
		nfseErro(w, http.StatusBadRequest, "requisição inválida")
		return
	}
	tx, err := db.BeginTx(r.Context(), nil)
	if err != nil {
		nfseErro(w, http.StatusInternalServerError, "erro interno")
		return
	}
	defer tx.Rollback()
	var status string
	var titulo sql.NullString
	if err := tx.QueryRowContext(r.Context(), `SELECT status, titulo_id FROM financeiro.notas_debito WHERE id=$1 FOR UPDATE`, req.ID).Scan(&status, &titulo); err != nil {
		nfseErro(w, http.StatusNotFound, "nota não encontrada")
		return
	}
	if status != "emitida" {
		nfseErro(w, http.StatusConflict, "a nota já está cancelada")
		return
	}
	if titulo.Valid {
		var ts string
		tx.QueryRowContext(r.Context(), `SELECT status FROM financeiro.titulos_receber WHERE id=$1 FOR UPDATE`, titulo.String).Scan(&ts)
		if ts == "pago" {
			nfseErro(w, http.StatusConflict, "o título desta nota já foi pago: não é possível cancelar")
			return
		}
		tx.ExecContext(r.Context(), `UPDATE financeiro.titulos_receber SET status='cancelado', baixa_obs='Nota de débito cancelada', updated_at=NOW() WHERE id=$1 AND status='aberto'`, titulo.String)
	}
	tx.ExecContext(r.Context(), `UPDATE financeiro.despesas_viagem SET nota_debito_id=NULL, updated_at=NOW() WHERE nota_debito_id=$1`, req.ID)
	if _, err := tx.ExecContext(r.Context(), `UPDATE financeiro.notas_debito SET status='cancelada' WHERE id=$1`, req.ID); err != nil {
		nfseErro(w, http.StatusInternalServerError, "erro ao cancelar")
		return
	}
	if err := tx.Commit(); err != nil {
		nfseErro(w, http.StatusInternalServerError, "erro ao cancelar")
		return
	}
	nfseJSON(w, http.StatusOK, map[string]string{"status": "cancelada"})
}
