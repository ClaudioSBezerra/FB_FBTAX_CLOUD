package handlers

// Consulta de CNPJ (dados cadastrais públicos) para preencher o destino da nota de débito.
//
//	GET /api/financeiro/cnpj?cnpj=00000000000000

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"fb_cloud/services"
)

// urlConsultaCNPJ é variável só para os testes apontarem para um servidor local.
var urlConsultaCNPJ = "https://brasilapi.com.br/api/cnpj/v1/"

type brasilAPICNPJ struct {
	RazaoSocial     string `json:"razao_social"`
	NomeFantasia    string `json:"nome_fantasia"`
	TipoLogradouro  string `json:"descricao_tipo_de_logradouro"`
	Logradouro      string `json:"logradouro"`
	Numero          string `json:"numero"`
	Complemento     string `json:"complemento"`
	Bairro          string `json:"bairro"`
	Municipio       string `json:"municipio"`
	UF              string `json:"uf"`
	CEP             string `json:"cep"`
	SituacaoCadastr string `json:"descricao_situacao_cadastral"`
}

func limpo(s string) string { return strings.Join(strings.Fields(s), " ") }

// CNPJConsultaHandler devolve razão social e endereço de um CNPJ, já montados para o destino.
func CNPJConsultaHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		cnpj := onlyDigits(r.URL.Query().Get("cnpj"))
		if len(cnpj) != 14 || !services.CNPJValido(cnpj) {
			nfseErro(w, http.StatusBadRequest, "CNPJ inválido")
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 12*time.Second)
		defer cancel()
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlConsultaCNPJ+cnpj, nil)
		if err != nil {
			nfseErro(w, http.StatusInternalServerError, "erro interno")
			return
		}
		req.Header.Set("User-Agent", "fbtax-cloud")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			// 424 e não 502/504: o Cloudflare troca o corpo desses dois e a mensagem some.
			nfseErro(w, http.StatusFailedDependency, "não consegui consultar o CNPJ agora; preencha o destino à mão")
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusBadRequest {
			nfseErro(w, http.StatusNotFound, "CNPJ não encontrado na base da Receita")
			return
		}
		if resp.StatusCode != http.StatusOK {
			nfseErro(w, http.StatusFailedDependency, "a consulta de CNPJ está indisponível; preencha o destino à mão")
			return
		}
		var d brasilAPICNPJ
		if json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&d) != nil || d.RazaoSocial == "" {
			nfseErro(w, http.StatusFailedDependency, "resposta inesperada da consulta de CNPJ")
			return
		}
		rua := limpo(d.TipoLogradouro + " " + d.Logradouro)
		if n := limpo(d.Numero); n != "" {
			rua += ", " + n
		}
		if c := limpo(d.Complemento); c != "" {
			rua += " - " + c
		}
		partes := []string{}
		for _, p := range []string{rua, limpo(d.Bairro)} {
			if p != "" {
				partes = append(partes, p)
			}
		}
		if m := limpo(d.Municipio); m != "" {
			if uf := limpo(d.UF); uf != "" {
				m += " - " + uf
			}
			partes = append(partes, m)
		}
		endereco := strings.Join(partes, " - ")
		if cep := onlyDigits(d.CEP); len(cep) == 8 {
			endereco += ". CEP " + cep[:5] + "-" + cep[5:]
		}
		nfseJSON(w, http.StatusOK, map[string]string{
			"cnpj": cnpj, "razao_social": limpo(d.RazaoSocial), "nome_fantasia": limpo(d.NomeFantasia),
			"endereco": endereco, "situacao": limpo(d.SituacaoCadastr),
			"destino": limpo(d.RazaoSocial) + " — " + endereco,
		})
	}
}
