package services

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"database/sql"
	"encoding/json"
	"encoding/pem"
	_ "github.com/lib/pq"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
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

func TestValidarPeriodoExtrato(t *testing.T) {
	hoje := time.Now().Format("2006-01-02")
	ontem := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	ha91 := time.Now().AddDate(0, 0, -91).Format("2006-01-02")
	ha90 := time.Now().AddDate(0, 0, -90).Format("2006-01-02")
	amanha2 := time.Now().AddDate(0, 0, 3).Format("2006-01-02")
	for nome, c := range map[string]struct {
		i, f string
		ok   bool
	}{
		"ok":        {ontem, hoje, true},
		"90 dias":   {ha90, hoje, true},
		"91 dias":   {ha91, hoje, false},
		"invertido": {hoje, ontem, false},
		"futuro":    {hoje, amanha2, false},
		"formato":   {"01/11/2026", hoje, false},
		"vazio":     {"", "", false},
	} {
		if err := ValidarPeriodoExtrato(c.i, c.f); (err == nil) != c.ok {
			t.Errorf("%s: err=%v, esperado ok=%v", nome, err, c.ok)
		}
	}
}

// Sincronização de um período explícito, de ponta a ponta contra um Inter simulado: grava os
// lançamentos, não duplica na segunda execução e devolve o período consultado.
func TestSyncAccountFromDBPeriodo_GravaEDeduplica(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL não definido")
	}
	t.Setenv("FINANCEIRO_SECRET", "segredo-de-teste")
	db, err := sql.Open("postgres", url)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/oauth/v2/token", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"access_token":"t","expires_in":3600,"scope":"extrato.read"}`))
	})
	// Extrato enriquecido em duas páginas, com detalhes por lançamento.
	mux.HandleFunc("/banking/v2/extrato/completo", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("dataInicio") != "2026-10-01" || q.Get("dataFim") != "2026-10-31" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		switch q.Get("pagina") {
		case "0":
			w.Write([]byte(`{"totalPaginas":2,"ultimaPagina":false,"primeiraPagina":true,"transacoes":[
				{"idTransacao":"sync-1","dataTransacao":"2026-10-05","tipoTransacao":"PIX","tipoOperacao":"C","valor":2500,"descricao":"PIX RECEBIDO","detalhes":{"endToEnd":"E123","nomePagador":"JC DISTRIBUICAO","cpfCnpjPagador":"06314327000203"}}]}`))
		case "1":
			w.Write([]byte(`{"totalPaginas":2,"ultimaPagina":true,"primeiraPagina":false,"transacoes":[
				{"idTransacao":"sync-2","dataTransacao":"2026-10-06","tipoTransacao":"PAGAMENTO","tipoOperacao":"D","valor":100,"descricao":"PAGAMENTO"}]}`))
		default:
			w.WriteHeader(http.StatusBadRequest)
		}
	})
	mux.HandleFunc("/banking/v2/saldo", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(`{"disponivel":777.5}`)) })
	srv := httptest.NewTLSServer(mux)
	defer srv.Close()
	pool := x509.NewCertPool()
	pool.AddCert(srv.Certificate())
	mtlsRootCAsTeste = pool
	defer func() { mtlsRootCAsTeste = nil }()
	interTokenCache.Range(func(k, _ any) bool { interTokenCache.Delete(k); return true })

	certPEM, keyPEM := parCertInter(t, time.Now().Add(time.Hour))
	cfgJSON, _ := json.Marshal(ProviderConfig{ClientID: "x", ClientSecret: "y", CertPEM: certPEM, KeyPEM: keyPEM, BaseURL: srv.URL})
	enc, err := EncryptConfig(string(cfgJSON))
	if err != nil {
		t.Fatal(err)
	}
	var contaID string
	if err := db.QueryRow(`INSERT INTO financeiro.contas_financeiras (apelido, banco, provedor, provedor_config) VALUES ('Sync teste','077','inter',$1) RETURNING id`, enc).Scan(&contaID); err != nil {
		t.Fatal(err)
	}
	defer db.Exec(`DELETE FROM financeiro.contas_financeiras WHERE id = $1`, contaID)

	if _, err := SyncAccountFromDBPeriodo(db, contaID, "2026-10-01", "2026-12-31"); err == nil {
		t.Fatal("período maior que 90 dias deveria ser recusado")
	}
	r1, err := SyncAccountFromDBPeriodo(db, contaID, "2026-10-01", "2026-10-31")
	if err != nil {
		t.Fatal(err)
	}
	if r1.Importadas != 2 || r1.Duplicadas != 0 || r1.SaldoFinal != 777.5 || r1.DataInicio != "2026-10-01" || r1.DataFim != "2026-10-31" {
		t.Fatalf("primeira sync: %+v", r1)
	}
	r2, err := SyncAccountFromDBPeriodo(db, contaID, "2026-10-01", "2026-10-31")
	if err != nil || r2.Importadas != 0 || r2.Duplicadas != 2 {
		t.Fatalf("segunda sync deveria só achar duplicadas: %+v %v", r2, err)
	}
	var det string
	if err := db.QueryRow(`SELECT detalhes::text FROM financeiro.transacoes WHERE referencia_ext = 'sync-1'`).Scan(&det); err != nil || !strings.Contains(det, "06314327000203") {
		t.Fatalf("detalhes do lançamento não foram guardados: %q %v", det, err)
	}
	var credito, debito float64
	db.QueryRow(`SELECT COALESCE(SUM(valor) FILTER (WHERE tipo='credito'),0), COALESCE(SUM(valor) FILTER (WHERE tipo='debito'),0) FROM financeiro.transacoes WHERE conta_id=$1`, contaID).Scan(&credito, &debito)
	if credito != 2500 || debito != 100 {
		t.Fatalf("lançamentos gravados errado: crédito %v débito %v", credito, debito)
	}
}

func TestInterContaCorrente_Normaliza(t *testing.T) {
	for in, want := range map[string]string{"": "", "0012345-6": "123456", " 12345678 ": "12345678", "000": "", "ab": ""} {
		if got := interContaCorrente(in); got != want {
			t.Errorf("%q → %q, esperado %q", in, got, want)
		}
	}
}
