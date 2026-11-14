package handlers

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestDespesasReembolso_FluxoCompleto(t *testing.T) {
	db := nfseTestDB(t)
	db.Exec(`DELETE FROM financeiro.empresas`)
	if _, err := db.Exec(`INSERT INTO financeiro.empresas (razao_social, cnpj, logradouro, numero, bairro, cep, municipio, uf)
		VALUES ('FORTES BEZERRA TESTE','38149716000128','Rua Patricia','20','Jardim Progresso','74911745','Aparecida de Goiânia','GO')`); err != nil {
		t.Fatal(err)
	}
	suf := fmt.Sprintf("%d", time.Now().UnixNano())
	var cli string
	if err := db.QueryRow(`INSERT INTO financeiro.clientes (razao_social, cnpj, logradouro, numero, bairro, municipio, uf)
		VALUES ('FERREIRA COSTA TESTE', $1, 'Av. Mascarenhas de Moraes','2967','Imbiribeira','Recife','PE') RETURNING id`, suf[len(suf)-14:]).Scan(&cli); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		db.Exec(`UPDATE financeiro.notas_debito SET titulo_id=NULL WHERE cliente_id=$1`, cli)
		db.Exec(`DELETE FROM financeiro.titulos_receber WHERE cliente_id=$1`, cli)
		db.Exec(`DELETE FROM financeiro.despesas_viagem WHERE cliente_id=$1`, cli)
		db.Exec(`DELETE FROM financeiro.notas_debito WHERE cliente_id=$1`, cli)
		db.Exec(`DELETE FROM financeiro.despesas_comprovantes WHERE nome LIKE 'teste-despesa-%'`)
		db.Exec(`DELETE FROM financeiro.clientes WHERE id=$1`, cli)
	})
	desp, comp, notas, tit := DespesasHandler(db), DespesasComprovantesHandler(db), NotasDebitoHandler(db), TitulosHandler(db)
	js := func(h http.HandlerFunc, m, url string, v any) *httptest.ResponseRecorder {
		b, _ := json.Marshal(v)
		return chamar(h, m, url, b, "application/json")
	}
	dec := func(rr *httptest.ResponseRecorder, v any) {
		t.Helper()
		if err := json.Unmarshal(rr.Body.Bytes(), v); err != nil {
			t.Fatalf("json: %v: %s", err, rr.Body)
		}
	}

	// comprovante: só PDF/imagem
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	fw, _ := mw.CreateFormFile("arquivo", "teste-despesa-uber.pdf")
	fw.Write([]byte("%PDF-1.4\n%teste\n"))
	fx, _ := mw.CreateFormFile("arquivo", "teste-despesa-script.pdf")
	fx.Write([]byte("<html><script>alert(1)</script>"))
	mw.Close()
	if rr := chamar(comp, http.MethodPost, "/", body.Bytes(), mw.FormDataContentType()); rr.Code != http.StatusBadRequest {
		t.Fatalf("arquivo que não é PDF/imagem deve ser recusado: %d %s", rr.Code, rr.Body)
	}
	body.Reset()
	mw = multipart.NewWriter(&body)
	fw, _ = mw.CreateFormFile("arquivo", "teste-despesa-uber.pdf")
	fw.Write([]byte("%PDF-1.4\n%teste\n"))
	mw.Close()
	rr := chamar(comp, http.MethodPost, "/", body.Bytes(), mw.FormDataContentType())
	if rr.Code != http.StatusCreated {
		t.Fatalf("upload: %d %s", rr.Code, rr.Body)
	}

	// importação: liga o comprovante pelo nome do arquivo e rejeita tipo fora da lista
	ruim := map[string]any{"cliente_id": cli, "despesas": []map[string]any{{"data": "2026-11-08", "tipo_despesa": "VIAGEM A MARTE", "descricao": "x", "valor": 1}}}
	if rr := js(desp, http.MethodPost, "/?action=importar", ruim); rr.Code != http.StatusBadRequest {
		t.Fatalf("tipo inválido: %d", rr.Code)
	}
	imp := map[string]any{"cliente_id": cli, "despesas": []map[string]any{
		{"data": "2026-11-08", "tipo_despesa": "TÁXI / APLICATIVO", "descricao": "Uber", "valor": 57.06, "comprovante": "teste-despesa-uber.pdf"},
		{"data": "2026-11-10", "tipo_despesa": "TÁXI / APLICATIVO", "descricao": "Uber", "valor": 17.98, "comprovante": "teste-despesa-uber.pdf"},
		{"data": "2026-11-08", "tipo_despesa": "HOSPEDAGEM", "descricao": "Hospedagem AIRBNB", "valor": 3812.46},
		{"data": "2026-11-14", "tipo_despesa": "ALIMENTAÇÃO - ALMOÇO", "descricao": "Delivery Com.Pão", "valor": 65.83, "obs": "revisar"},
	}}
	rr = js(desp, http.MethodPost, "/?action=importar", imp)
	if rr.Code != http.StatusCreated {
		t.Fatalf("importar: %d %s", rr.Code, rr.Body)
	}

	// pendentes não entram na nota
	var lista struct {
		Despesas []struct {
			ID, Status, Descricao, ComprovanteNome string
			Valor                                   float64
		} `json:"despesas"`
	}
	rr = chamar(desp, http.MethodGet, "/?cliente_id="+cli, nil, "")
	dec(rr, &lista)
	if len(lista.Despesas) != 4 {
		t.Fatalf("despesas: %d", len(lista.Despesas))
	}
	var ids []string
	for _, d := range lista.Despesas {
		if d.Status != "pendente" {
			t.Fatalf("importadas devem nascer pendentes: %s", d.Status)
		}
		ids = append(ids, d.ID)
	}
	nota := map[string]any{"cliente_id": cli, "despesa_ids": ids, "destino": "Recife - PE", "solicitante": "Fulano"}
	if rr := js(notas, http.MethodPost, "/", nota); rr.Code != http.StatusConflict {
		t.Fatalf("nota com despesa pendente deve dar 409: %d %s", rr.Code, rr.Body)
	}

	// aprova três (o almoço fica de fora) e fecha a nota
	aprovar := ids[:3]
	if rr := js(desp, http.MethodPost, "/?action=status", map[string]any{"ids": aprovar, "status": "aprovada"}); rr.Code != 200 {
		t.Fatalf("aprovar: %d %s", rr.Code, rr.Body)
	}
	nota["despesa_ids"] = aprovar
	rr = js(notas, http.MethodPost, "/", nota)
	if rr.Code != http.StatusCreated {
		t.Fatalf("criar nota: %d %s", rr.Code, rr.Body)
	}
	var criada struct {
		ID, Codigo, TituloID, Vencimento string
		Total                            float64
		Linhas                           int `json:"linhas"`
	}
	dec(rr, &criada)
	criada.TituloID = strings.TrimSpace(criada.TituloID)
	var raw map[string]any
	dec(rr, &raw)
	if raw["total"].(float64) != 3887.50 || raw["linhas"].(float64) != 2 {
		t.Errorf("total/linhas: %v %v", raw["total"], raw["linhas"])
	}
	hoje := time.Now().In(nfseBRT)
	if want := time.Date(hoje.Year(), hoje.Month(), hoje.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, 10).Format("2006-01-02"); raw["vencimento"] != want {
		t.Errorf("vencimento %v, esperado %s (emissão + 10 dias)", raw["vencimento"], want)
	}

	// o título de reembolso nasceu em aberto, sem contrato
	var tipo, status string
	var valor float64
	if err := db.QueryRow(`SELECT tipo, status, valor::float8 FROM financeiro.titulos_receber WHERE id=$1`, raw["titulo_id"]).Scan(&tipo, &status, &valor); err != nil || tipo != "reembolso" || status != "aberto" || valor != 3887.50 {
		t.Fatalf("título: %s %s %v %v", tipo, status, valor, err)
	}
	_ = tit

	// despesas da nota ficam travadas; o almoço continua editável
	if rr := js(desp, http.MethodPut, "/", map[string]any{"id": aprovar[0], "cliente_id": cli, "data": "2026-11-08", "tipo_despesa": "TÁXI / APLICATIVO", "descricao": "x", "valor": 1, "status": "aprovada"}); rr.Code != http.StatusConflict {
		t.Errorf("editar despesa de nota fechada: %d", rr.Code)
	}
	if rr := chamar(desp, http.MethodDelete, "/?id="+aprovar[0], nil, ""); rr.Code != http.StatusConflict {
		t.Errorf("excluir despesa de nota fechada: %d", rr.Code)
	}
	// mesma despesa não entra em duas notas
	if rr := js(notas, http.MethodPost, "/", nota); rr.Code != http.StatusConflict {
		t.Errorf("reuso de despesa: %d", rr.Code)
	}

	// downloads: xlsx e zip com o comprovante (uma vez só, mesmo ligado a 2 despesas)
	rr = chamar(notas, http.MethodGet, "/?action=xlsx&id="+raw["id"].(string), nil, "")
	if rr.Code != 200 || !strings.Contains(rr.Header().Get("Content-Disposition"), ".xlsx") || rr.Body.Len() < 1000 {
		t.Fatalf("xlsx: %d %d", rr.Code, rr.Body.Len())
	}
	rr = chamar(notas, http.MethodGet, "/?action=zip&id="+raw["id"].(string), nil, "")
	zr, err := zip.NewReader(bytes.NewReader(rr.Body.Bytes()), int64(rr.Body.Len()))
	if err != nil || len(zr.File) != 1 || zr.File[0].Name != "teste-despesa-uber.pdf" {
		t.Fatalf("zip: %v", err)
	}

	// cancelar a nota cancela o título e libera as despesas
	if rr := js(notas, http.MethodPost, "/?action=cancelar", map[string]any{"id": raw["id"]}); rr.Code != 200 {
		t.Fatalf("cancelar: %d %s", rr.Code, rr.Body)
	}
	db.QueryRow(`SELECT status FROM financeiro.titulos_receber WHERE id=$1`, raw["titulo_id"]).Scan(&status)
	var presas int
	db.QueryRow(`SELECT COUNT(*) FROM financeiro.despesas_viagem WHERE nota_debito_id IS NOT NULL AND cliente_id=$1`, cli).Scan(&presas)
	if status != "cancelado" || presas != 0 {
		t.Errorf("após cancelar: título %s, despesas presas %d", status, presas)
	}
	// nota de mesmo mês ganha sufixo
	rr = js(notas, http.MethodPost, "/", nota)
	if rr.Code != http.StatusCreated {
		t.Fatalf("nova nota: %d %s", rr.Code, rr.Body)
	}
	dec(rr, &raw)
	if !strings.HasSuffix(raw["codigo"].(string), "-2") {
		t.Errorf("segunda nota do mês deveria ter sufixo -2: %v", raw["codigo"])
	}
	// nota com título pago não cancela
	db.Exec(`UPDATE financeiro.titulos_receber SET status='pago', pago_em=CURRENT_DATE, valor_pago=valor WHERE id=$1`, raw["titulo_id"])
	if rr := js(notas, http.MethodPost, "/?action=cancelar", map[string]any{"id": raw["id"]}); rr.Code != http.StatusConflict {
		t.Errorf("cancelar nota paga: %d", rr.Code)
	}
}
