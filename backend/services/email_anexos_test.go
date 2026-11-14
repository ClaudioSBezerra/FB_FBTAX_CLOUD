package services

import (
	"bytes"
	"io"
	"mime"
	"mime/multipart"
	"net/mail"
	"strings"
	"testing"
	"time"
)

func TestMontarEmailComAnexos(t *testing.T) {
	agora := time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC)
	planilha := []byte("PK\x03\x04conteudo-da-planilha")
	msg, err := MontarEmailComAnexos("FBTax <contato@fortesbezerra.com.br>", []string{"a@cliente.com.br"}, []string{"eu@fortesbezerra.com.br"}, "claudio@fortesbezerra.com.br",
		"Nota de Débito FOR.FIN.0002-112026 — Reembolso", "Prezados,\nSegue a nota.\n", []AnexoEmail{{Nome: "FOR.FIN.0002-112026.xlsx", Tipo: "application/vnd.ms-excel", Dados: planilha}}, agora)
	if err != nil {
		t.Fatal(err)
	}
	m, err := mail.ReadMessage(bytes.NewReader(msg))
	if err != nil {
		t.Fatal(err)
	}
	dec := new(mime.WordDecoder)
	if assunto, _ := dec.DecodeHeader(m.Header.Get("Subject")); assunto != "Nota de Débito FOR.FIN.0002-112026 — Reembolso" {
		t.Errorf("assunto: %q", assunto)
	}
	if m.Header.Get("Reply-To") != "claudio@fortesbezerra.com.br" {
		t.Errorf("reply-to: %q", m.Header.Get("Reply-To"))
	}
	if m.Header.Get("To") != "a@cliente.com.br" || m.Header.Get("Cc") != "eu@fortesbezerra.com.br" {
		t.Errorf("destinatários: %q %q", m.Header.Get("To"), m.Header.Get("Cc"))
	}
	tipo, params, err := mime.ParseMediaType(m.Header.Get("Content-Type"))
	if err != nil || tipo != "multipart/mixed" {
		t.Fatalf("content-type: %v %v", tipo, err)
	}
	mr := multipart.NewReader(m.Body, params["boundary"])
	p1, _ := mr.NextPart()
	corpo, _ := io.ReadAll(p1)
	_ = corpo
	if !strings.HasPrefix(p1.Header.Get("Content-Type"), "text/plain") {
		t.Errorf("1ª parte: %s", p1.Header.Get("Content-Type"))
	}
	p2, err := mr.NextPart()
	if err != nil {
		t.Fatal(err)
	}
	if _, par, _ := mime.ParseMediaType(p2.Header.Get("Content-Disposition")); par["filename"] != "FOR.FIN.0002-112026.xlsx" {
		t.Errorf("anexo: %v", par)
	}
	if !strings.Contains(string(msg), "Content-Transfer-Encoding: base64") {
		t.Error("anexo deve ir em base64")
	}
}

func TestMontarEmailComAnexos_Recusas(t *testing.T) {
	agora := time.Now()
	if _, err := MontarEmailComAnexos("a@b.com", nil, nil, "", "x", "y", nil, agora); err == nil {
		t.Error("sem destinatário deve falhar")
	}
	if _, err := MontarEmailComAnexos("a@b.com", []string{"c@d.com"}, nil, "", "x\r\nBcc: ladrao@x.com", "y", nil, agora); err == nil {
		t.Error("injeção de cabeçalho no assunto deve falhar")
	}
	grande := []AnexoEmail{{Nome: "g.zip", Dados: make([]byte, LimiteAnexosEmail+1)}}
	if _, err := MontarEmailComAnexos("a@b.com", []string{"c@d.com"}, nil, "", "x", "y", grande, agora); err != ErrAnexosGrandes {
		t.Errorf("anexo grande: %v", err)
	}
	if _, err := MontarEmailComAnexos("a@b.com", []string{"c@d.com"}, nil, "x\r\nBcc: z@z.com", "x", "y", nil, agora); err == nil {
		t.Error("injeção de cabeçalho no reply-to deve falhar")
	}
	if _, err := MontarEmailComAnexos("a@b.com", []string{"c@d.com"}, nil, "", "x", "y", []AnexoEmail{{Nome: "a\".exe", Dados: []byte("x")}}, agora); err == nil {
		t.Error("nome de anexo com aspas deve falhar")
	}
}

func TestValidarEnderecos(t *testing.T) {
	ok, err := ValidarEnderecos([]string{" Fulano <a@b.com.br> ", "", "c@d.com"})
	if err != nil || len(ok) != 2 || ok[0] != "a@b.com.br" {
		t.Errorf("%v %v", ok, err)
	}
	for _, ruim := range []string{"sem-arroba", "a@b.com\r\nBcc: x@y.com", "<>"} {
		if _, err := ValidarEnderecos([]string{ruim}); err == nil {
			t.Errorf("deveria recusar %q", ruim)
		}
	}
}
