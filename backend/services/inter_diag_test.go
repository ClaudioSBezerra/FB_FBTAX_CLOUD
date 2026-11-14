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
	mux.HandleFunc("/oauth/v2/token", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		if r.Form.Get("client_id") != "id-teste" || r.Form.Get("client_secret") != "segredo-teste" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if !concedidos[r.Form.Get("scope")] {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":"invalid_scope"}`))
			return
		}
		w.Write([]byte(`{"access_token":"tok-` + r.Form.Get("scope") + `","expires_in":3600,"scope":"` + r.Form.Get("scope") + `"}`))
	})
	mux.HandleFunc("/banking/v2/saldo", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNotFound) })
	mux.HandleFunc("/banking/v3/saldo", func(w http.ResponseWriter, r *http.Request) {
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

	certPEM, keyPEM := parCertInter(t, time.Now().Add(40*24*time.Hour))
	cfg := ProviderConfig{ClientID: "id-teste", ClientSecret: "segredo-teste", CertPEM: certPEM, KeyPEM: keyPEM, ContaCorrente: "12345678", BaseURL: srv.URL}

	d := DiagnosticarInter(cfg)
	if !d.TokenOK || !d.SaldoOK || d.SaldoDisponivel != 1234.56 || d.BankingVersao != "v3" {
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
