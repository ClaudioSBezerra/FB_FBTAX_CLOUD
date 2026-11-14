package services

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func parCertInter(t *testing.T, validoAte time.Time) (certPEM, keyPEM string) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tpl := &x509.Certificate{SerialNumber: big.NewInt(9), Subject: pkix.Name{CommonName: "inter-teste"},
		NotBefore: time.Now().Add(-time.Hour), NotAfter: validoAte, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}}
	der, err := x509.CreateCertificate(rand.Reader, tpl, tpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	kb, _ := x509.MarshalECPrivateKey(key)
	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})),
		string(pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: kb}))
}

func TestDiagnosticarInter_DescobreEscoposESaldo(t *testing.T) {
	concedidos := map[string]bool{"extrato.read": true, "cobv.read": true, "cobv.write": true}
	mux := http.NewServeMux()
	var chamadas, limitarAte int
	estrito := false // true: recusa o conjunto se QUALQUER escopo não for permitido (fluxo de recuo)
	mux.HandleFunc("/oauth/v2/token", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		chamadas++
		if chamadas <= limitarAte {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		if r.Form.Get("client_id") != "id-teste" || r.Form.Get("client_secret") != "segredo-teste" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		var ok []string
		pedidos := strings.Fields(r.Form.Get("scope"))
		for _, e := range pedidos {
			if concedidos[e] {
				ok = append(ok, e)
			}
		}
		if len(ok) == 0 || (estrito && len(ok) != len(pedidos)) {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":"invalid_scope"}`))
			return
		}
		w.Write([]byte(`{"access_token":"tok-` + ok[0] + `","expires_in":3600,"scope":"` + strings.Join(ok, " ") + `"}`))
	})
	mux.HandleFunc("/banking/v2/saldo", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-conta-corrente") != "12345678" || !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer tok-") {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		w.Write([]byte(`{"disponivel":1234.56,"bloqueado":0}`))
	})
	srv := httptest.NewTLSServer(mux)
	defer srv.Close()
	pool := x509.NewCertPool()
	pool.AddCert(srv.Certificate())
	mtlsRootCAsTeste = pool
	defer func() { mtlsRootCAsTeste = nil }()
	antes := interTokenEspaco
	interTokenEspaco = time.Millisecond
	defer func() { interTokenEspaco = antes }()

	certPEM, keyPEM := parCertInter(t, time.Now().Add(40*24*time.Hour))
	cfg := ProviderConfig{ClientID: "id-teste", ClientSecret: "segredo-teste", CertPEM: certPEM, KeyPEM: keyPEM, ContaCorrente: "12345678", BaseURL: srv.URL}

	d := DiagnosticarInter(cfg)
	if !d.TokenOK || !d.SaldoOK || d.SaldoDisponivel != 1234.56 || d.BankingVersao != "v2" {
		t.Fatalf("diagnóstico inesperado: %+v", d)
	}
	got := map[string]bool{}
	for _, e := range d.Escopos {
		got[e.Escopo] = e.Concedido
	}
	for esc, ok := range map[string]bool{"extrato.read": true, "cobv.write": true, "cobv.read": true, "pix.write": false, "pagamento-pix.write": false, "webhook.write": false} {
		if got[esc] != ok {
			t.Errorf("escopo %s: concedido=%v, esperado %v", esc, got[esc], ok)
		}
	}
	if d.CertDiasRestantes < 39 || d.CertDiasRestantes > 40 {
		t.Errorf("dias restantes do certificado: %d", d.CertDiasRestantes)
	}
	if len(d.Avisos) == 0 || !strings.Contains(d.Avisos[0], "vence em") {
		t.Errorf("deveria avisar do vencimento próximo (≤ 90 dias): %v", d.Avisos)
	}

	// Escopo perigoso concedido gera aviso; credenciais erradas não emitem token.
	concedidos["pagamento-pix.write"] = true
	interTokenCache.Range(func(k, _ any) bool { interTokenCache.Delete(k); return true })
	if d := DiagnosticarInter(cfg); !strings.Contains(strings.Join(d.Avisos, " "), "pagamento-pix.write") {
		t.Errorf("deveria alertar escopo que movimenta dinheiro: %v", d.Avisos)
	}
	// Recuo: servidor que recusa o conjunto inteiro (estrito) → um token por escopo, resultado igual.
	estrito = true
	chamadas = 0
	concedidos["pagamento-pix.write"] = false
	interTokenCache.Range(func(k, _ any) bool { interTokenCache.Delete(k); return true })
	d = DiagnosticarInter(cfg)
	got = map[string]bool{}
	for _, e := range d.Escopos {
		got[e.Escopo] = e.Concedido
	}
	if !got["extrato.read"] || !got["cobv.write"] || got["pix.write"] || chamadas < len(interEscoposSonda) {
		t.Errorf("recuo para token por escopo falhou: %v (chamadas=%d)", got, chamadas)
	}
	// 429: as primeiras chamadas são limitadas; o cliente espera e tenta de novo, sem marcar falta de permissão.
	estrito, chamadas, limitarAte = false, 0, 2
	interTokenCache.Range(func(k, _ any) bool { interTokenCache.Delete(k); return true })
	if d := DiagnosticarInter(cfg); !d.TokenOK || !d.SaldoOK {
		t.Errorf("deveria superar o 429 com nova tentativa: %+v", d)
	}
	limitarAte = 0
	cfg.ClientSecret = "errado"
	interTokenCache.Range(func(k, _ any) bool { interTokenCache.Delete(k); return true })
	if d := DiagnosticarInter(cfg); d.TokenOK || !strings.Contains(strings.Join(d.Avisos, " "), "MESMA integração") {
		t.Errorf("credenciais erradas: %+v", d)
	}
}

func TestValidarParCertificadoInter(t *testing.T) {
	c1, k1 := parCertInter(t, time.Now().Add(time.Hour))
	_, k2 := parCertInter(t, time.Now().Add(time.Hour))
	if _, err := ValidarParCertificadoInter(c1, k1); err != nil {
		t.Fatalf("par válido: %v", err)
	}
	if _, err := ValidarParCertificadoInter(c1, k2); err == nil {
		t.Fatal("certificado e chave de pares diferentes deveriam ser recusados")
	}
	if _, err := ValidarParCertificadoInter("lixo", "lixo"); err == nil {
		t.Fatal("lixo deveria ser recusado")
	}
}

func TestInterExtrato_ClassificaCreditoDebitoPeloSentidoEDeduplica(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/oauth/v2/token", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"access_token":"t","expires_in":3600,"scope":"extrato.read"}`))
	})
	mux.HandleFunc("/banking/v2/extrato", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("dataInicio") != "2026-11-01" || r.URL.Query().Get("dataFim") != "2026-11-14" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.Write([]byte(`{"transacoes":[
			{"idTransacao":"a1","dataTransacao":"2026-11-10","tipoTransacao":"PIX","tipoOperacao":"C","valor":2500.00,"descricao":"PIX RECEBIDO JC DISTRIBUICAO"},
			{"idTransacao":"a2","dataTransacao":"2026-11-11T10:00:00","tipoTransacao":"BOLETO_COBRANCA","tipoOperacao":"C","valor":4500,"titulo":"Boleto liquidado"},
			{"idTransacao":"a3","dataTransacao":"2026-11-12","tipoTransacao":"PAGAMENTO","tipoOperacao":"D","valor":-120.5,"descricao":"Pagamento de conta"},
			{"idTransacao":"a4","dataTransacao":"2026-11-12","tipoTransacao":"CREDITO","tipoOperacao":"","valor":10,"descricao":"formato antigo"},
			{"dataTransacao":"2026-11-13","tipoTransacao":"PIX","tipoOperacao":"C","valor":99,"descricao":"sem id"},
			{"idTransacao":"a6","dataTransacao":"2026-11-13","tipoTransacao":"OUTROS","tipoOperacao":"?","valor":5,"descricao":"sentido desconhecido"}
		]}`))
	})
	srv := httptest.NewTLSServer(mux)
	defer srv.Close()
	pool := x509.NewCertPool()
	pool.AddCert(srv.Certificate())
	mtlsRootCAsTeste = pool
	defer func() { mtlsRootCAsTeste = nil }()
	interTokenCache.Range(func(k, _ any) bool { interTokenCache.Delete(k); return true })

	certPEM, keyPEM := parCertInter(t, time.Now().Add(time.Hour))
	cfg := ProviderConfig{ClientID: "x", ClientSecret: "y", CertPEM: certPEM, KeyPEM: keyPEM, BaseURL: srv.URL}
	txs, err := (&InterProvider{}).GetExtrato(cfg, "2026-11-01", "2026-11-14")
	if err != nil {
		t.Fatal(err)
	}
	if len(txs) != 5 {
		t.Fatalf("esperava 5 lançamentos (o de sentido desconhecido é ignorado), vieram %d: %+v", len(txs), txs)
	}
	esperado := map[string]struct {
		tipo, cat, data string
		valor           float64
	}{
		"a1": {"credito", "PIX", "2026-11-10", 2500},
		"a2": {"credito", "BOLETO_COBRANCA", "2026-11-11", 4500},
		"a3": {"debito", "PAGAMENTO", "2026-11-12", 120.5},
		"a4": {"credito", "", "2026-11-12", 10},
	}
	semID := 0
	for _, x := range txs {
		if e, ok := esperado[x.IDExterno]; ok {
			if x.Tipo != e.tipo || x.Categoria != e.cat || x.DataTransacao != e.data || x.Valor != e.valor {
				t.Errorf("%s: %+v, esperado %+v", x.IDExterno, x, e)
			}
		} else if strings.HasPrefix(x.IDExterno, "h-") {
			semID++
		} else {
			t.Errorf("lançamento inesperado: %+v", x)
		}
	}
	if semID != 1 {
		t.Errorf("o lançamento sem id deveria ganhar um identificador estável (h-…)")
	}
	again, _ := (&InterProvider{}).GetExtrato(cfg, "2026-11-01", "2026-11-14")
	if len(again) != 5 || again[4].IDExterno != txs[4].IDExterno {
		t.Errorf("identificador sintético precisa ser estável entre sincronizações")
	}
}
