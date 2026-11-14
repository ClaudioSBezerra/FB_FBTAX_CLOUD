package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCNPJConsulta(t *testing.T) {
	var chamado string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		chamado = r.URL.Path
		switch r.URL.Path {
		case "/11222333000181":
			w.Write([]byte(`{"razao_social":"EMPRESA  TESTE LTDA","nome_fantasia":"","descricao_tipo_de_logradouro":"AVENIDA","logradouro":"MASCARENHAS DE MORAES","numero":"2967","complemento":"","bairro":"IMBIRIBEIRA","municipio":"RECIFE","uf":"PE","cep":"51150000","descricao_situacao_cadastral":"ATIVA"}`))
		case "/00000000000191":
			w.WriteHeader(http.StatusNotFound)
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer srv.Close()
	original := urlConsultaCNPJ
	urlConsultaCNPJ = srv.URL + "/"
	t.Cleanup(func() { urlConsultaCNPJ = original })
	h := CNPJConsultaHandler()

	// CNPJ com dígito errado nem chega à consulta externa
	chamado = ""
	if rr := chamar(h, http.MethodGet, "/?cnpj=11222333000182", nil, ""); rr.Code != http.StatusBadRequest || chamado != "" {
		t.Errorf("DV inválido: %d (chamou %q)", rr.Code, chamado)
	}
	if rr := chamar(h, http.MethodGet, "/?cnpj=123", nil, ""); rr.Code != http.StatusBadRequest {
		t.Errorf("curto: %d", rr.Code)
	}

	rr := chamar(h, http.MethodGet, "/?cnpj=11.222.333/0001-81", nil, "")
	if rr.Code != 200 {
		t.Fatalf("consulta: %d %s", rr.Code, rr.Body)
	}
	var j map[string]string
	json.Unmarshal(rr.Body.Bytes(), &j)
	want := "EMPRESA TESTE LTDA — AVENIDA MASCARENHAS DE MORAES, 2967 - IMBIRIBEIRA - RECIFE - PE. CEP 51150-000"
	if j["destino"] != want || chamado != "/11222333000181" {
		t.Errorf("destino %q (chamou %q)", j["destino"], chamado)
	}

	if rr := chamar(h, http.MethodGet, "/?cnpj=00000000000191", nil, ""); rr.Code != http.StatusNotFound {
		t.Errorf("não encontrado: %d", rr.Code)
	}
	// falha da consulta externa vira 424 (nunca 502/504, que o Cloudflare mascara)
	srv.Close()
	if rr := chamar(h, http.MethodGet, "/?cnpj=11222333000181", nil, ""); rr.Code != http.StatusFailedDependency {
		t.Errorf("serviço fora do ar: %d", rr.Code)
	}
}
