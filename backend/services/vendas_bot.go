package services

// vendas_bot.go — Assistente FbTax, o bot de vendas da landing apuracaoassistida.ia.br (#1).
//
// Conversa com Claude Haiku usando duas ferramentas:
//   - atualizar_lead: grava o que o visitante contou (qualificação e contato);
//   - encaminhar_equipe: marca o lead como quente e dispara o aviso para a equipe.
// Regras fixas no prompt: nunca falar preço, não dar parecer tributário, se apresentar como
// assistente virtual e só pedir contato com consentimento.

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go/v2"
	"github.com/anthropics/anthropic-sdk-go/v2/option"
)

//go:embed vendas_conhecimento.md
var conhecimentoVendas string

const (
	ModeloVendasPadrao   = "claude-haiku-5-5"
	whatsappVendasPadrao = "5562983462443"
)

var ErrVendasNaoConfigurado = errors.New("assistente de vendas não configurado (ANTHROPIC_API_KEY)")

// MensagemVendas é um turno guardado da conversa (papel "visitante" ou "assistente").
type MensagemVendas struct {
	Papel string `json:"papel"`
	Texto string `json:"texto"`
}

// AtualizacaoLead: campos que o modelo pode preencher (vazio = não mudou).
type AtualizacaoLead struct {
	Nome                 string `json:"nome"`
	Empresa              string `json:"empresa"`
	Cargo                string `json:"cargo"`
	Email                string `json:"email"`
	Whatsapp             string `json:"whatsapp"`
	FaixaFaturamento     string `json:"faixa_faturamento"`
	ERP                  string `json:"erp"`
	NumEmpresas          string `json:"num_empresas"`
	Prazo                string `json:"prazo"`
	Resumo               string `json:"resumo"`
	ConsentimentoContato bool   `json:"consentimento_contato"`
}

// ResultadoTurno: resposta ao visitante e o que mudou no lead.
type ResultadoTurno struct {
	Resposta     string
	Atualizacoes []AtualizacaoLead
	Encaminhar   bool
	Motivo       string
}

func ModeloVendas() string {
	if m := strings.TrimSpace(os.Getenv("VENDAS_MODELO")); m != "" {
		return m
	}
	return ModeloVendasPadrao
}

func WhatsappVendas() string {
	if n := strings.TrimSpace(os.Getenv("VENDAS_WHATSAPP")); n != "" {
		return n
	}
	return whatsappVendasPadrao
}

func promptVendas() string {
	return `Você é o Assistente FbTax, o assistente virtual de vendas do FBTax (produto da FBTechIA) no site apuracaoassistida.ia.br.

Objetivo: tirar dúvidas de quem visita o site, entender se o FBTax faz sentido para a empresa e, quando fizer, levar a pessoa a uma demonstração com um especialista.

Como conversar:
- Português do Brasil, cordial e objetivo. Respostas curtas (até 4 frases), sem listas longas. Uma pergunta por vez.
- Na primeira mensagem, apresente-se como assistente virtual do FBTax. Nunca finja ser uma pessoa.
- Use só a base de conhecimento abaixo. Se não souber, diga que o especialista explica na demonstração. Nunca invente funcionalidade, cliente, número ou prazo.
- NUNCA informe preço, faixa de preço, desconto ou estimativa de valor, mesmo se insistirem. Diga que depende do porte, do ERP e do número de empresas do grupo e que o especialista apresenta na demonstração.
- Não dê parecer tributário ou jurídico sobre o caso da pessoa (se tem direito a crédito, quanto vai pagar, como classificar). Explique o conceito em termos gerais e ofereça a conversa com o especialista.
- Nunca cite o nome do cliente em produção (é "um grande varejista de materiais de construção com SAP S/4HANA").

Qualificação (ao longo da conversa, de forma natural, nunca como formulário):
nome, cargo, empresa, faixa de faturamento anual, ERP usado, quantas empresas (CNPJs raiz) no grupo e quando precisam estar prontos.
Sempre que a pessoa contar algo disso, chame a ferramenta atualizar_lead com o que ela disse (só o que ela disse, sem adivinhar).

Contato e privacidade (LGPD):
- Antes de pedir e-mail ou WhatsApp, pergunte se ela autoriza a FBTechIA a entrar em contato para a demonstração (os termos estão em https://apuracaoassistida.ia.br/termos-e-privacidade.html).
- Só grave e-mail/WhatsApp com consentimento_contato verdadeiro, depois que ela autorizar.

Encaminhamento:
- Chame encaminhar_equipe quando a pessoa quiser falar com alguém, pedir demonstração ou preço, ou quando já tiver nome, empresa e um contato autorizado.
- Depois de encaminhar, diga que um especialista da FBTechIA vai chamar e que ela também pode clicar no botão do WhatsApp que aparece abaixo da conversa.

=== BASE DE CONHECIMENTO ===
` + conhecimentoVendas
}

func ferramentasVendas() []anthropic.ToolUnionParam {
	texto := func(desc string) map[string]any { return map[string]any{"type": "string", "description": desc} }
	atualizar := anthropic.ToolParam{
		Name:        "atualizar_lead",
		Description: anthropic.String("Grava o que o visitante contou sobre ele e a empresa. Envie só os campos que ele informou nesta conversa."),
		InputSchema: anthropic.ToolInputSchemaParam{Properties: map[string]any{
			"nome":                  texto("Nome da pessoa"),
			"empresa":               texto("Nome da empresa"),
			"cargo":                 texto("Cargo ou área"),
			"email":                 texto("E-mail (só com consentimento)"),
			"whatsapp":              texto("WhatsApp/telefone (só com consentimento)"),
			"faixa_faturamento":     texto("Faturamento anual como a pessoa disse (ex.: 'entre 100 e 500 milhões')"),
			"erp":                   texto("ERP usado (SAP, Oracle, Totvs...)"),
			"num_empresas":          texto("Quantas empresas/CNPJs raiz no grupo"),
			"prazo":                 texto("Quando precisam estar prontos"),
			"resumo":                texto("Uma frase sobre o interesse ou a dor principal"),
			"consentimento_contato": map[string]any{"type": "boolean", "description": "true se a pessoa autorizou o contato da FBTechIA"},
		}},
	}
	encaminhar := anthropic.ToolParam{
		Name:        "encaminhar_equipe",
		Description: anthropic.String("Avisa a equipe de vendas da FBTechIA que este lead quer demonstração ou falar com alguém."),
		InputSchema: anthropic.ToolInputSchemaParam{Properties: map[string]any{
			"motivo": texto("Por que encaminhar (ex.: 'pediu demonstração', 'perguntou preço')"),
		}},
	}
	return []anthropic.ToolUnionParam{{OfTool: &atualizar}, {OfTool: &encaminhar}}
}

// mensagensVendas converte o histórico guardado em mensagens alternadas (começa pelo visitante).
func mensagensVendas(historico []MensagemVendas, nova string) []anthropic.MessageParam {
	if len(historico) > 24 {
		historico = historico[len(historico)-24:]
	}
	msgs := []anthropic.MessageParam{}
	ultimo := ""
	for _, h := range historico {
		t := strings.TrimSpace(h.Texto)
		papel := "user"
		if h.Papel == "assistente" {
			papel = "assistant"
		}
		if t == "" || papel == ultimo || (len(msgs) == 0 && papel == "assistant") {
			continue
		}
		if papel == "user" {
			msgs = append(msgs, anthropic.NewUserMessage(anthropic.NewTextBlock(t)))
		} else {
			msgs = append(msgs, anthropic.NewAssistantMessage(anthropic.NewTextBlock(t)))
		}
		ultimo = papel
	}
	if ultimo == "user" {
		msgs = msgs[:len(msgs)-1]
	}
	return append(msgs, anthropic.NewUserMessage(anthropic.NewTextBlock(nova)))
}

// ConversarVendas roda um turno: chama o modelo, executa as ferramentas (no máximo 3 voltas) e
// devolve a resposta final. Quem chama grava o lead e dispara os avisos.
func ConversarVendas(ctx context.Context, historico []MensagemVendas, nova string) (ResultadoTurno, error) {
	var res ResultadoTurno
	if strings.TrimSpace(os.Getenv("ANTHROPIC_API_KEY")) == "" {
		return res, ErrVendasNaoConfigurado
	}
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	cli := anthropic.NewClient()
	msgs := mensagensVendas(historico, nova)
	sistema := []anthropic.TextBlockParam{{Text: promptVendas(), CacheControl: anthropic.NewCacheControlEphemeralParam()}}

	var texto strings.Builder
	for volta := 0; volta < 3; volta++ {
		resp, err := cli.Messages.New(ctx, anthropic.MessageNewParams{
			Model:        anthropic.Model(ModeloVendas()),
			MaxTokens:    700,
			System:       sistema,
			Messages:     msgs,
			Tools:        ferramentasVendas(),
			OutputConfig: anthropic.OutputConfigParam{Effort: anthropic.OutputConfigEffortLow},
		}, option.WithMaxRetries(1))
		if err != nil {
			return res, fmt.Errorf("falha ao consultar o modelo: %w", err)
		}
		if resp.StopReason == anthropic.StopReasonRefusal {
			res.Resposta = "Não consigo ajudar com isso por aqui. Se quiser, um especialista da FBTechIA conversa com você pelo WhatsApp."
			return res, nil
		}
		msgs = append(msgs, resp.ToParam())
		resultados := []anthropic.ContentBlockParamUnion{}
		for _, b := range resp.Content {
			switch v := b.AsAny().(type) {
			case anthropic.TextBlock:
				texto.WriteString(v.Text)
			case anthropic.ToolUseBlock:
				switch v.Name {
				case "atualizar_lead":
					var a AtualizacaoLead
					if err := json.Unmarshal([]byte(v.JSON.Input.Raw()), &a); err != nil {
						resultados = append(resultados, anthropic.NewToolResultBlock(b.ID, "dados inválidos", true))
						continue
					}
					res.Atualizacoes = append(res.Atualizacoes, a)
					resultados = append(resultados, anthropic.NewToolResultBlock(b.ID, "gravado", false))
				case "encaminhar_equipe":
					var e struct {
						Motivo string `json:"motivo"`
					}
					json.Unmarshal([]byte(v.JSON.Input.Raw()), &e)
					res.Encaminhar, res.Motivo = true, e.Motivo
					resultados = append(resultados, anthropic.NewToolResultBlock(b.ID, "equipe avisada; o botão de WhatsApp vai aparecer para o visitante", false))
				default:
					resultados = append(resultados, anthropic.NewToolResultBlock(b.ID, "ferramenta desconhecida", true))
				}
			}
		}
		if resp.StopReason != anthropic.StopReasonToolUse || len(resultados) == 0 {
			break
		}
		msgs = append(msgs, anthropic.NewUserMessage(resultados...))
	}
	res.Resposta = strings.TrimSpace(texto.String())
	if res.Resposta == "" {
		res.Resposta = "Certo! Um especialista da FBTechIA vai falar com você. Se preferir, use o botão do WhatsApp abaixo."
	}
	return res, nil
}

// LinkWhatsappLead monta o link de WhatsApp da equipe com a mensagem preenchida.
func LinkWhatsappLead(id int64, nome, empresa string) string {
	t := "Olá! Conversei com o Assistente FbTax no site e quero agendar uma demonstração."
	if nome != "" {
		t = "Olá! Sou " + nome
		if empresa != "" {
			t += ", da " + empresa
		}
		t += ". Conversei com o Assistente FbTax no site e quero agendar uma demonstração."
	}
	t += fmt.Sprintf(" (lead #%d)", id)
	return "https://wa.me/" + WhatsappVendas() + "?text=" + url.QueryEscape(t)
}

// EnviarTelegramVendas manda o aviso para o grupo de vendas (TELEGRAM_BOT_TOKEN + TELEGRAM_CHAT_ID_VENDAS).
// Sem as variáveis, não faz nada (o e-mail continua indo).
func EnviarTelegramVendas(ctx context.Context, texto string) error {
	token := strings.TrimSpace(os.Getenv("TELEGRAM_BOT_TOKEN"))
	chat := strings.TrimSpace(os.Getenv("TELEGRAM_CHAT_ID_VENDAS"))
	if token == "" || chat == "" {
		return nil
	}
	base := strings.TrimRight(os.Getenv("TELEGRAM_API_BASE"), "/")
	if base == "" {
		base = "https://api.telegram.org"
	}
	corpo, _ := json.Marshal(map[string]any{"chat_id": chat, "text": texto, "disable_web_page_preview": true})
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/bot"+token+"/sendMessage", bytes.NewReader(corpo))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("telegram: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram: HTTP %d", resp.StatusCode)
	}
	log.Printf("[Vendas] aviso enviado ao Telegram")
	return nil
}
