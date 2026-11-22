package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	_ "github.com/lib/pq"
)

// #1: rota pública inteira (Anthropic e Telegram falsos). Rode com TEST_DATABASE_URL apontando para um
// banco com a migration 133.
func TestVendasChat_Integracao(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL não definido")
	}
	db, err := sql.Open("postgres", url)
	if err != nil || db.Ping() != nil {
		t.Skip("banco de teste indisponível")
	}
	defer db.Close()
	db.Exec(`DELETE FROM vendas_leads`)

	var turno int32
	ia := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		switch atomic.AddInt32(&turno, 1) {
		case 1: // 1ª mensagem: só texto
			w.Write([]byte(`{"id":"a","type":"message","role":"assistant","model":"m","stop_reason":"end_turn","content":[{"type":"text","text":"Olá! Sou o assistente virtual do FBTax."}],"usage":{"input_tokens":1,"output_tokens":1}}`))
		case 2: // 2ª: grava e encaminha
			w.Write([]byte(`{"id":"b","type":"message","role":"assistant","model":"m","stop_reason":"tool_use","content":[{"type":"tool_use","id":"t1","name":"atualizar_lead","input":{"nome":"Ana","empresa":"Empresa X","consentimento_contato":true,"whatsapp":"62999990000"}},{"type":"tool_use","id":"t2","name":"encaminhar_equipe","input":{"motivo":"pediu demonstração"}}],"usage":{"input_tokens":1,"output_tokens":1}}`))
		default:
			w.Write([]byte(`{"id":"c","type":"message","role":"assistant","model":"m","stop_reason":"end_turn","content":[{"type":"text","text":"Um especialista vai chamar você."}],"usage":{"input_tokens":1,"output_tokens":1}}`))
		}
	}))
	defer ia.Close()
	var avisos int32
	tg := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&avisos, 1)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer tg.Close()
	t.Setenv("ANTHROPIC_BASE_URL", ia.URL)
	t.Setenv("ANTHROPIC_API_KEY", "teste")
	t.Setenv("TELEGRAM_API_BASE", tg.URL)
	t.Setenv("TELEGRAM_BOT_TOKEN", "1:a")
	t.Setenv("TELEGRAM_CHAT_ID_VENDAS", "-1")
	t.Setenv("SMTP_PASSWORD", "")
	t.Setenv("TURNSTILE_SECRET", "")

	enviar := func(corpo string) map[string]any {
		r := httptest.NewRequest(http.MethodPost, "/api/vendas/chat", bytes.NewBufferString(corpo))
		r.Header.Set("CF-Connecting-IP", "203.0.113.9")
		w := httptest.NewRecorder()
		VendasChatHandler(db)(w, r)
		if w.Code != 200 {
			t.Fatalf("HTTP %d: %s", w.Code, w.Body)
		}
		var m map[string]any
		json.Unmarshal(w.Body.Bytes(), &m)
		return m
	}
	r1 := enviar(`{"mensagem":"Oi, o que é o FBTax?","origem":"utm_source=instagram"}`)
	sessao, _ := r1["sessao"].(string)
	if sessao == "" || r1["whatsapp_url"] != nil {
		t.Fatalf("1º turno: %v", r1)
	}
	r2 := enviar(`{"sessao":"` + sessao + `","mensagem":"Quero uma demonstração, sou a Ana da Empresa X, pode me chamar no 62999990000"}`)
	wa, _ := r2["whatsapp_url"].(string)
	if !strings.Contains(wa, "wa.me/") || !strings.Contains(wa, "Empresa+X") {
		t.Errorf("2º turno deveria trazer o WhatsApp: %v", r2)
	}
	var st, nome, zap, origem string
	var msgs int
	var enc bool
	db.QueryRow(`SELECT status, nome, whatsapp, origem, mensagens, encaminhado_em IS NOT NULL FROM vendas_leads WHERE sessao = $1::uuid`, sessao).
		Scan(&st, &nome, &zap, &origem, &msgs, &enc)
	if st != "quente" || nome != "Ana" || zap != "62999990000" || origem != "utm_source=instagram" || msgs != 2 || !enc {
		t.Errorf("lead gravado: status=%s nome=%s zap=%s origem=%s msgs=%d enc=%v", st, nome, zap, origem, msgs, enc)
	}
	for i := 0; i < 50 && atomic.LoadInt32(&avisos) == 0; i++ {
		time.Sleep(20 * time.Millisecond)
	}
	if atomic.LoadInt32(&avisos) != 1 {
		t.Errorf("Telegram deveria receber 1 aviso, recebeu %d", avisos)
	}
}

// #1 (caso real, lead #2): encaminhou no 1º turno sem contato; o WhatsApp só chegou depois. A equipe
// recebe uma atualização com o contato (📇), uma única vez.
func TestVendasChat_ContatoDepoisDoEncaminhamento(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL não definido")
	}
	db, err := sql.Open("postgres", url)
	if err != nil || db.Ping() != nil {
		t.Skip("banco de teste indisponível")
	}
	defer db.Close()
	db.Exec(`DELETE FROM vendas_leads`)

	var turno int32
	var estados []string
	ia := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		var req struct {
			System []struct {
				Text string `json:"text"`
			} `json:"system"`
		}
		json.Unmarshal(b, &req)
		if len(req.System) > 1 {
			estados = append(estados, req.System[1].Text)
		}
		w.Header().Set("Content-Type", "application/json")
		switch atomic.AddInt32(&turno, 1) {
		case 1: // 1º turno: encaminha sem dados
			w.Write([]byte(`{"id":"a","type":"message","role":"assistant","model":"m","stop_reason":"tool_use","content":[{"type":"tool_use","id":"t1","name":"encaminhar_equipe","input":{"motivo":"quer falar com vendedor"}}],"usage":{"input_tokens":1,"output_tokens":1}}`))
		case 3: // 2º turno: chega nome + contato autorizado
			w.Write([]byte(`{"id":"b","type":"message","role":"assistant","model":"m","stop_reason":"tool_use","content":[{"type":"tool_use","id":"t2","name":"atualizar_lead","input":{"nome":"Claudio","empresa":"JC","consentimento_contato":true,"whatsapp":"62999990000"}}],"usage":{"input_tokens":1,"output_tokens":1}}`))
		default:
			w.Write([]byte(`{"id":"c","type":"message","role":"assistant","model":"m","stop_reason":"end_turn","content":[{"type":"text","text":"Certo!"}],"usage":{"input_tokens":1,"output_tokens":1}}`))
		}
	}))
	defer ia.Close()
	var avisos int32
	var textos []string
	tg := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		textos = append(textos, string(b))
		atomic.AddInt32(&avisos, 1)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer tg.Close()
	t.Setenv("ANTHROPIC_BASE_URL", ia.URL)
	t.Setenv("ANTHROPIC_API_KEY", "teste")
	t.Setenv("TELEGRAM_API_BASE", tg.URL)
	t.Setenv("TELEGRAM_BOT_TOKEN", "1:a")
	t.Setenv("TELEGRAM_CHAT_ID_VENDAS", "-1")
	t.Setenv("SMTP_PASSWORD", "")
	t.Setenv("TURNSTILE_SECRET", "")

	enviar := func(corpo string) map[string]any {
		r := httptest.NewRequest(http.MethodPost, "/api/vendas/chat", bytes.NewBufferString(corpo))
		r.Header.Set("CF-Connecting-IP", "203.0.113.10")
		w := httptest.NewRecorder()
		VendasChatHandler(db)(w, r)
		if w.Code != 200 {
			t.Fatalf("HTTP %d: %s", w.Code, w.Body)
		}
		var m map[string]any
		json.Unmarshal(w.Body.Bytes(), &m)
		return m
	}
	esperar := func(n int32) {
		for i := 0; i < 50 && atomic.LoadInt32(&avisos) < n; i++ {
			time.Sleep(20 * time.Millisecond)
		}
	}
	sessao, _ := enviar(`{"mensagem":"Quero que um vendedor fale comigo"}`)["sessao"].(string)
	esperar(1)
	enviar(`{"sessao":"` + sessao + `","mensagem":"Claudio, da JC, autorizo, 62999990000"}`)
	esperar(2)
	enviar(`{"sessao":"` + sessao + `","mensagem":"obrigado"}`)
	time.Sleep(200 * time.Millisecond)

	if n := atomic.LoadInt32(&avisos); n != 2 {
		t.Fatalf("esperava 2 avisos (encaminhamento + contato), veio %d", n)
	}
	if !strings.Contains(textos[1], "📇") || !strings.Contains(textos[1], "62999990000") || !strings.Contains(textos[1], "Claudio") {
		t.Errorf("atualização deveria trazer o contato: %s", textos[1])
	}
	if len(estados) < 3 || !strings.Contains(estados[2], "Equipe já avisada (encaminhar_equipe já foi executada): sim") {
		t.Errorf("o 2º turno deveria informar ao modelo que a equipe já foi avisada: %v", estados)
	}
}
