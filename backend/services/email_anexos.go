package services

// E-mail com anexos pelo SMTP do sistema (Hostinger, SSL na 465).

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"mime"
	"net/mail"
	"net/textproto"
	"strings"
	"time"
)

// AnexoEmail é um arquivo anexado à mensagem.
type AnexoEmail struct {
	Nome  string
	Tipo  string // content-type
	Dados []byte
}

// LimiteAnexosEmail é a soma máxima dos anexos (antes do base64, que incha ~33%): fica folgado
// sob o limite de 35 MB da caixa de saída da Hostinger.
const LimiteAnexosEmail = 15 << 20

var ErrAnexosGrandes = errors.New("os anexos passam de 15 MB: envie os comprovantes à parte")

func enderecoValido(s string) (string, error) {
	if strings.ContainsAny(s, "\r\n") {
		return "", fmt.Errorf("endereço inválido: %q", s)
	}
	a, err := mail.ParseAddress(strings.TrimSpace(s))
	if err != nil {
		return "", fmt.Errorf("e-mail inválido: %q", s)
	}
	return a.Address, nil
}

// ValidarEnderecos devolve os endereços limpos ou o primeiro erro.
func ValidarEnderecos(l []string) ([]string, error) {
	out := make([]string, 0, len(l))
	for _, s := range l {
		if strings.TrimSpace(s) == "" {
			continue
		}
		a, err := enderecoValido(s)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, nil
}

func quebrarBase64(b []byte) string {
	s := base64.StdEncoding.EncodeToString(b)
	var sb strings.Builder
	for len(s) > 76 {
		sb.WriteString(s[:76] + "\r\n")
		s = s[76:]
	}
	sb.WriteString(s + "\r\n")
	return sb.String()
}

// MontarEmailComAnexos monta a mensagem MIME (texto UTF-8 + anexos). Assunto e nomes de arquivo
// seguem RFC 2047, então acentos chegam certos; quebras de linha em campos de cabeçalho são
// recusadas (injeção de cabeçalho).
func MontarEmailComAnexos(from string, para, cc []string, responderPara, assunto, corpo string, anexos []AnexoEmail, agora time.Time) ([]byte, error) {
	if len(para) == 0 {
		return nil, errors.New("informe ao menos um destinatário")
	}
	if strings.ContainsAny(assunto, "\r\n") || strings.ContainsAny(from, "\r\n") {
		return nil, errors.New("assunto inválido")
	}
	var total int
	for _, a := range anexos {
		if strings.ContainsAny(a.Nome, "\r\n\"") {
			return nil, fmt.Errorf("nome de anexo inválido: %q", a.Nome)
		}
		total += len(a.Dados)
	}
	if total > LimiteAnexosEmail {
		return nil, ErrAnexosGrandes
	}
	limite := fmt.Sprintf("fbtax-%d", agora.UnixNano())
	var b bytes.Buffer
	h := func(k, v string) { b.WriteString(textproto.CanonicalMIMEHeaderKey(k) + ": " + v + "\r\n") }
	h("From", from)
	h("To", strings.Join(para, ", "))
	if len(cc) > 0 {
		h("Cc", strings.Join(cc, ", "))
	}
	if responderPara != "" {
		a, err := enderecoValido(responderPara)
		if err != nil {
			return nil, err
		}
		h("Reply-To", a)
	}
	h("Subject", mime.QEncoding.Encode("utf-8", assunto))
	h("Date", agora.Format(time.RFC1123Z))
	h("MIME-Version", "1.0")
	h("Content-Type", `multipart/mixed; boundary="`+limite+`"`)
	b.WriteString("\r\n")

	b.WriteString("--" + limite + "\r\n")
	b.WriteString("Content-Type: text/plain; charset=UTF-8\r\nContent-Transfer-Encoding: base64\r\n\r\n")
	b.WriteString(quebrarBase64([]byte(strings.ReplaceAll(corpo, "\r\n", "\n"))))
	for _, a := range anexos {
		tipo := a.Tipo
		if tipo == "" {
			tipo = "application/octet-stream"
		}
		nome := mime.QEncoding.Encode("utf-8", a.Nome)
		b.WriteString("--" + limite + "\r\n")
		b.WriteString(fmt.Sprintf("Content-Type: %s; name=\"%s\"\r\n", tipo, nome))
		b.WriteString("Content-Transfer-Encoding: base64\r\n")
		b.WriteString(fmt.Sprintf("Content-Disposition: attachment; filename=\"%s\"\r\n\r\n", nome))
		b.WriteString(quebrarBase64(a.Dados))
	}
	b.WriteString("--" + limite + "--\r\n")
	return b.Bytes(), nil
}

// EnviarEmailComAnexos envia pelo SMTP configurado. Os destinatários em "Cc" também entram no
// envelope. A Hostinger só aceita como remetente a conta autenticada (SMTP_USER), então o
// remetente escolhido na tela vai em Reply-To: o cliente responde para ele.
func EnviarEmailComAnexos(para, cc []string, responderPara, assunto, corpo string, anexos []AnexoEmail) error {
	cfg := GetEmailConfig()
	if cfg.Password == "" {
		return ErrSMTPNaoConfigurado
	}
	msg, err := MontarEmailComAnexos(cfg.From, para, cc, responderPara, assunto, corpo, anexos, time.Now())
	if err != nil {
		return err
	}
	return sendMailSSL(cfg, append(append([]string{}, para...), cc...), msg)
}
