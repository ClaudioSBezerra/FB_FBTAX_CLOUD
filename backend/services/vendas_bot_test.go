package services

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

// #1: turno com ferramentas — 1ª volta chama atualizar_lead e encaminhar_equipe, 2ª devolve o texto.
func TestConversarVendas_FerramentasEDepoisTexto(t *testing.T) {
	var chamadas int32
	var primeiro map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&chamadas, 1)
		b, _ := io.ReadAll(r.Body)
		if n == 1 {
			json.Unmarshal(b, &primeiro)
		}
		w.Header().Set("Content-Type", "application/json")
		if n == 1 {
			w.Write([]byte(`{"id":"m1","type":"message","role":"assistant","model":"claude-haiku-5-5","stop_reason":"tool_use",
			 "content":[{"type":"tool_use","id":"tu1","name":"atualizar_lead","input":{"nome":"Ana","empresa":"Empresa X","erp":"SAP","consentimento_contato":true,"email":"ana@x.com"}},
			            {"type":"tool_use","id":"tu2","name":"encaminhar_equipe","input":{"motivo":"pediu demonstração"}}],
			 "usage":{"input_tokens":10,"output_tokens":10}}`))
			return
		}
		w.Write([]byte(`{"id":"m2","type":"message","role":"assistant","model":"claude-haiku-5-5","stop_reason":"end_turn",
		 "content":[{"type":"text","text":"Combinado, Ana! Um especialista vai chamar você."}],"usage":{"input_tokens":10,"output_tokens":10}}`))
	}))
	defer srv.Close()
	t.Setenv("ANTHROPIC_BASE_URL", srv.URL)
	t.Setenv("ANTHROPIC_API_KEY", "teste")
	t.Setenv("VENDAS_MODELO", "")

	res, err := ConversarVendas(context.Background(), []MensagemVendas{{Papel: "assistente", Texto: "oi"}}, "Quero uma demonstração")
	if err != nil {
		t.Fatal(err)
	}
	if chamadas != 2 || !res.Encaminhar || res.Motivo != "pediu demonstração" {
		t.Errorf("chamadas=%d encaminhar=%v motivo=%q", chamadas, res.Encaminhar, res.Motivo)
	}
	if len(res.Atualizacoes) != 1 || res.Atualizacoes[0].Empresa != "Empresa X" || !res.Atualizacoes[0].ConsentimentoContato {
		t.Errorf("atualizações: %+v", res.Atualizacoes)
	}
	if !strings.Contains(res.Resposta, "especialista") {
		t.Errorf("resposta: %q", res.Resposta)
	}
	if primeiro["model"] != "claude-haiku-5-5" {
		t.Errorf("modelo: %v", primeiro["model"])
	}
	sys := primeiro["system"].([]any)[0].(map[string]any)
	if sys["cache_control"] == nil || !strings.Contains(sys["text"].(string), "NUNCA informe preço") {
		t.Error("system deveria trazer a regra de preço e cache")
	}
	if len(primeiro["tools"].([]any)) != 2 {
		t.Error("deveria enviar as duas ferramentas")
	}
	// o histórico começava pelo assistente: descartado para a API aceitar
	if msgs := primeiro["messages"].([]any); len(msgs) != 1 {
		t.Errorf("mensagens enviadas: %d", len(msgs))
	}
}

func TestConversarVendas_SemChave(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "")
	if _, err := ConversarVendas(context.Background(), nil, "oi"); err != ErrVendasNaoConfigurado {
		t.Errorf("sem chave: %v", err)
	}
}

func TestLinkWhatsappLead(t *testing.T) {
	t.Setenv("VENDAS_WHATSAPP", "")
	l := LinkWhatsappLead(7, "Ana", "Empresa X")
	if !strings.HasPrefix(l, "https://wa.me/5562983462443?text=") || !strings.Contains(l, "lead+%237") || !strings.Contains(l, "Empresa+X") {
		t.Errorf("link: %s", l)
	}
}

func TestEnviarTelegramVendas(t *testing.T) {
	var corpo map[string]any
	var caminho string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		caminho = r.URL.Path
		b, _ := io.ReadAll(r.Body)
		json.Unmarshal(b, &corpo)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()
	t.Setenv("TELEGRAM_API_BASE", srv.URL)
	t.Setenv("TELEGRAM_BOT_TOKEN", "123:abc")
	t.Setenv("TELEGRAM_CHAT_ID_VENDAS", "-100999")
	if err := EnviarTelegramVendas(context.Background(), "Lead #1"); err != nil {
		t.Fatal(err)
	}
	if caminho != "/bot123:abc/sendMessage" || corpo["chat_id"] != "-100999" || corpo["text"] != "Lead #1" {
		t.Errorf("caminho=%s corpo=%v", caminho, corpo)
	}
	// sem configuração: não faz nada nem dá erro
	t.Setenv("TELEGRAM_BOT_TOKEN", "")
	if err := EnviarTelegramVendas(context.Background(), "x"); err != nil {
		t.Errorf("sem token deveria ignorar: %v", err)
	}
}
