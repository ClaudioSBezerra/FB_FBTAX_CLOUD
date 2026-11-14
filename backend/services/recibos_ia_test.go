package services

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLerReciboIA_EnviaArquivoELeEstrutura(t *testing.T) {
	var corpo map[string]any
	var chave string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		chave = strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		b, _ := io.ReadAll(r.Body)
		json.Unmarshal(b, &corpo)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id":"msg_1","type":"message","role":"assistant","model":"claude-opus-5-5","stop_reason":"end_turn",
		 "content":[{"type":"text","text":"{\"despesas\":[{\"data\":\"2026-11-14\",\"tipo_despesa\":\"ALIMENTAÇÃO - ALMOÇO\",\"descricao\":\"Delivery Com.Pão\",\"valor\":65.83,\"obs\":\"\"},{\"data\":\"\",\"tipo_despesa\":\"OUTRAS\",\"descricao\":\"Bembom\",\"valor\":44.95,\"obs\":\"itens pessoais\"}]}"}],
		 "usage":{"input_tokens":10,"output_tokens":10}}`))
	}))
	defer srv.Close()
	t.Setenv("ANTHROPIC_BASE_URL", srv.URL)
	t.Setenv("ANTHROPIC_API_KEY", "chave-de-teste")

	ds, uso, err := LerReciboIA(context.Background(), "claude-haiku-5-5", "Arq1.pdf", "application/pdf", []byte("%PDF-1.4 teste"))
	if err != nil {
		t.Fatal(err)
	}
	if len(ds) != 2 || ds[0].Valor != 65.83 || ds[0].TipoDespesa != "ALIMENTAÇÃO - ALMOÇO" || ds[1].Obs != "itens pessoais" {
		t.Errorf("despesas: %+v", ds)
	}
	if chave != "chave-de-teste" {
		t.Errorf("api key enviada: %q", chave)
	}
	if uso.Modelo != "claude-haiku-5-5" || uso.EntradaTok != 10 || uso.SaidaTok != 10 {
		t.Errorf("uso: %+v", uso)
	}
	if corpo["model"] != "claude-haiku-5-5" {
		t.Errorf("modelo: %v", corpo["model"])
	}
	// o arquivo vai como documento base64 e a saída é estruturada com a lista de tipos do modelo
	msg := corpo["messages"].([]any)[0].(map[string]any)["content"].([]any)[0].(map[string]any)
	if msg["type"] != "document" {
		t.Errorf("bloco do arquivo: %v", msg["type"])
	}
	oc := corpo["output_config"].(map[string]any)
	if oc["format"].(map[string]any)["type"] != "json_schema" {
		t.Errorf("output_config: %v", oc)
	}
	if !strings.Contains(string(mustJSON(oc)), "TÁXI / APLICATIVO") {
		t.Error("o esquema deveria listar os tipos de despesa do modelo")
	}
	// não manda parâmetros removidos no Opus 5.5
	for _, k := range []string{"temperature", "top_p", "top_k", "budget_tokens"} {
		if _, ok := corpo[k]; ok {
			t.Errorf("não deve enviar %s", k)
		}
	}
}

func mustJSON(v any) []byte { b, _ := json.Marshal(v); return b }

func TestLerReciboIA_Recusas(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "")
	if _, _, err := LerReciboIA(context.Background(), "", "a.pdf", "application/pdf", []byte("x")); err != ErrIANaoConfigurada {
		t.Errorf("sem chave: %v", err)
	}
	t.Setenv("ANTHROPIC_API_KEY", "k")
	if _, _, err := LerReciboIA(context.Background(), "", "a.png", "image/png", make([]byte, limiteImagemIA+1)); err == nil || !strings.Contains(err.Error(), "5 MB") {
		t.Errorf("imagem grande: %v", err)
	}
	if _, _, err := LerReciboIA(context.Background(), "", "a.gif", "image/gif", []byte("x")); err == nil {
		t.Error("tipo não suportado deve falhar")
	}
}

func TestModeloLeituraValido(t *testing.T) {
	for _, id := range []string{"claude-haiku-5-5", "claude-sonnet-5-5", "claude-opus-5-5"} {
		if !ModeloLeituraValido(id) {
			t.Errorf("%s deveria ser permitido", id)
		}
	}
	if ModeloLeituraValido("gpt-4") || ModeloLeituraValido("") || ModeloLeituraValido("claude-opus-5-5-20260401") {
		t.Error("só os ids exatos da lista")
	}
	t.Setenv("ANTHROPIC_API_KEY", "k")
	if _, _, err := LerReciboIA(context.Background(), "modelo-invalido", "a.pdf", "application/pdf", []byte("x")); err == nil {
		t.Error("modelo fora da lista deve ser recusado antes de chamar a API")
	}
}
