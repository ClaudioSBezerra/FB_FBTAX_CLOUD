package handlers

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"fb_cloud/services"
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
			Valor                                  float64
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

	// e-mail: prévia sugere assunto/mensagem; envio valida endereços, leva planilha + zip e registra
	originalEnvio := enviarEmailNota
	t.Cleanup(func() { enviarEmailNota = originalEnvio })
	var enviado struct {
		para, cc           []string
		responder, assunto string
		anexos             []string
	}
	enviarEmailNota = func(para, cc []string, responderPara, assunto, corpo string, anexos []services.AnexoEmail) error {
		enviado.para, enviado.cc, enviado.responder, enviado.assunto = para, cc, responderPara, assunto
		enviado.anexos = nil
		for _, a := range anexos {
			enviado.anexos = append(enviado.anexos, a.Nome)
		}
		if !strings.Contains(corpo, "3.887,50") {
			t.Errorf("mensagem deveria trazer o total: %s", corpo)
		}
		return nil
	}
	nid := raw["id"].(string)
	rp := chamar(notas, http.MethodGet, "/?action=email-previa&id="+nid, nil, "")
	var previa map[string]any
	dec(rp, &previa)
	if rp.Code != 200 || !strings.Contains(previa["assunto"].(string), raw["codigo"].(string)) || !strings.Contains(previa["mensagem"].(string), "3.887,50") {
		t.Fatalf("prévia: %d %v", rp.Code, previa)
	}
	if rr := js(notas, http.MethodPost, "/?action=enviar", map[string]any{"id": nid, "para": []string{"sem-arroba"}}); rr.Code != http.StatusBadRequest {
		t.Errorf("e-mail de destino inválido: %d", rr.Code)
	}
	if rr := js(notas, http.MethodPost, "/?action=enviar", map[string]any{"id": nid, "para": []string{"fin@cliente.com.br"}, "remetente": "nao-e-email"}); rr.Code != http.StatusBadRequest {
		t.Errorf("remetente inválido: %d", rr.Code)
	}
	envio := map[string]any{"id": nid, "remetente": "claudio@fortesbezerra.com.br", "para": []string{"fin@cliente.com.br"}, "cc": []string{"eu@fortesbezerra.com.br"}, "incluir_comprovantes": true}
	if rr := js(notas, http.MethodPost, "/?action=enviar", envio); rr.Code != 200 {
		t.Fatalf("enviar: %d %s", rr.Code, rr.Body)
	}
	if enviado.responder != "claudio@fortesbezerra.com.br" || len(enviado.para) != 1 || len(enviado.cc) != 1 || len(enviado.anexos) != 2 {
		t.Errorf("envio: %+v", enviado)
	}
	var enviadaPara string
	db.QueryRow(`SELECT COALESCE(enviada_para,'') FROM financeiro.notas_debito WHERE id=$1`, nid).Scan(&enviadaPara)
	if enviadaPara != "fin@cliente.com.br, eu@fortesbezerra.com.br" {
		t.Errorf("envio não registrado: %q", enviadaPara)
	}
	if rr := js(notas, http.MethodPost, "/?action=enviar", envio); rr.Code != http.StatusConflict {
		t.Errorf("reenvio sem confirmação deve dar 409: %d", rr.Code)
	}
	envio["confirmar_reenvio"] = true
	if rr := js(notas, http.MethodPost, "/?action=enviar", envio); rr.Code != 200 {
		t.Errorf("reenvio confirmado: %d", rr.Code)
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

func TestDespesasLerIA(t *testing.T) {
	db := nfseTestDB(t)
	suf := fmt.Sprintf("%d", time.Now().UnixNano())
	var cli, comp string
	if err := db.QueryRow(`INSERT INTO financeiro.clientes (razao_social, cnpj) VALUES ('LEITURA IA TESTE', $1) RETURNING id`, suf[len(suf)-14:]).Scan(&cli); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`INSERT INTO financeiro.despesas_comprovantes (nome, content_type, dados) VALUES ('teste-despesa-ia.pdf','application/pdf','%PDF-1.4') RETURNING id`).Scan(&comp); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		db.Exec(`DELETE FROM financeiro.despesas_viagem WHERE cliente_id=$1`, cli)
		db.Exec(`DELETE FROM financeiro.despesas_comprovantes WHERE id=$1`, comp)
		db.Exec(`DELETE FROM financeiro.clientes WHERE id=$1`, cli)
	})
	original := lerReciboIA
	t.Cleanup(func() { lerReciboIA = original })
	chamadas := 0
	modeloUsado := ""
	lerReciboIA = func(ctx context.Context, modelo, nome, ct string, dados []byte) ([]services.DespesaLida, services.UsoIA, error) {
		chamadas++
		modeloUsado = modelo
		return []services.DespesaLida{
			{Data: "2026-11-14", TipoDespesa: "ALIMENTAÇÃO - ALMOÇO", Descricao: "Delivery  Com.Pão", Valor: 65.83},
			{Data: "2026-11-14", TipoDespesa: "OUTRAS", Descricao: "Bembom", Valor: 44.95, Obs: "itens pessoais"},
			{Data: "", TipoDespesa: "ALIMENTAÇÃO - JANTAR", Descricao: "Aads Refeições", Valor: 44.30, Obs: "data ilegível"},
			{Data: "2026-11-14", TipoDespesa: "TIPO INVENTADO", Descricao: "x", Valor: 1},
		}, services.UsoIA{Modelo: modelo, EntradaTok: 100, SaidaTok: 50}, nil
	}
	h := DespesasHandler(db)
	ler := func(reler bool) *httptest.ResponseRecorder {
		b, _ := json.Marshal(map[string]any{"cliente_id": cli, "comprovante_id": comp, "reler": reler})
		return chamar(h, http.MethodPost, "/?action=ler", b, "application/json")
	}
	rr := ler(false)
	if rr.Code != 200 {
		t.Fatalf("ler: %d %s", rr.Code, rr.Body)
	}
	var out struct {
		Lidas     int
		Ignoradas []string
	}
	json.Unmarshal(rr.Body.Bytes(), &out)
	if out.Lidas != 2 || len(out.Ignoradas) != 2 {
		t.Fatalf("lidas %d, ignoradas %v", out.Lidas, out.Ignoradas)
	}
	var status, obs, desc string
	db.QueryRow(`SELECT status, obs, descricao FROM financeiro.despesas_viagem WHERE cliente_id=$1 AND valor=44.95`, cli).Scan(&status, &obs, &desc)
	if status != "pendente" || !strings.Contains(obs, "itens pessoais") || !strings.Contains(obs, "lido por IA") {
		t.Errorf("despesa lida: %s %q", status, obs)
	}
	db.QueryRow(`SELECT descricao FROM financeiro.despesas_viagem WHERE cliente_id=$1 AND valor=65.83`, cli).Scan(&desc)
	if desc != "Delivery Com.Pão" {
		t.Errorf("descrição normalizada: %q", desc)
	}
	// ler de novo o mesmo arquivo não duplica sem pedir
	if rr := ler(false); rr.Code != http.StatusConflict || chamadas != 1 {
		t.Errorf("releitura sem pedir: %d (chamadas %d)", rr.Code, chamadas)
	}
	// releitura pedida marca a duplicidade
	if rr := ler(true); rr.Code != 200 {
		t.Fatalf("reler: %d", rr.Code)
	}
	db.QueryRow(`SELECT obs FROM financeiro.despesas_viagem WHERE cliente_id=$1 AND valor=65.83 ORDER BY created_at DESC LIMIT 1`, cli).Scan(&obs)
	if !strings.Contains(obs, "duplicidade") {
		t.Errorf("duplicidade não sinalizada: %q", obs)
	}
	// modelo fora da lista é recusado; simulação não grava nada e devolve o consumo
	lerCom := func(extra map[string]any) *httptest.ResponseRecorder {
		corpo := map[string]any{"cliente_id": cli, "comprovante_id": comp}
		for k, v := range extra {
			corpo[k] = v
		}
		b, _ := json.Marshal(corpo)
		return chamar(h, http.MethodPost, "/?action=ler", b, "application/json")
	}
	if rr := lerCom(map[string]any{"modelo": "gpt-4"}); rr.Code != http.StatusBadRequest {
		t.Errorf("modelo inválido: %d", rr.Code)
	}
	var antes int
	db.QueryRow(`SELECT COUNT(*) FROM financeiro.despesas_viagem WHERE cliente_id=$1`, cli).Scan(&antes)
	rs := lerCom(map[string]any{"modelo": "claude-haiku-5-5", "simular": true})
	var sim struct {
		Simulacao bool
		Despesas  []map[string]any
		Total     float64
		Uso       struct{ Modelo string }
	}
	json.Unmarshal(rs.Body.Bytes(), &sim)
	var depois int
	db.QueryRow(`SELECT COUNT(*) FROM financeiro.despesas_viagem WHERE cliente_id=$1`, cli).Scan(&depois)
	if rs.Code != 200 || !sim.Simulacao || len(sim.Despesas) != 4 || sim.Uso.Modelo != "claude-haiku-5-5" || modeloUsado != "claude-haiku-5-5" || antes != depois {
		t.Errorf("simulação: %d %+v modelo=%s antes=%d depois=%d", rs.Code, sim, modeloUsado, antes, depois)
	}

	// IA sem chave vira 409 com mensagem clara; falha externa vira 424 (nunca 502/504)
	lerReciboIA = func(ctx context.Context, modelo, nome, ct string, dados []byte) ([]services.DespesaLida, services.UsoIA, error) {
		return nil, services.UsoIA{}, services.ErrIANaoConfigurada
	}
	if rr := ler(true); rr.Code != http.StatusConflict {
		t.Errorf("sem chave: %d", rr.Code)
	}
	lerReciboIA = func(ctx context.Context, modelo, nome, ct string, dados []byte) ([]services.DespesaLida, services.UsoIA, error) {
		return nil, services.UsoIA{}, fmt.Errorf("timeout")
	}
	if rr := ler(true); rr.Code != http.StatusFailedDependency {
		t.Errorf("falha externa: %d", rr.Code)
	}
}
