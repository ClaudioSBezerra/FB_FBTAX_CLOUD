package handlers

import (
	"strings"
	"testing"
)

func TestValidarNovoUsuario(t *testing.T) {
	casos := []struct {
		req      novoUsuarioReq
		erro     string // "" = válido
		emailOut string
	}{
		{novoUsuarioReq{Nome: " Maria ", Email: " Maria@Exemplo.com "}, "", "maria@exemplo.com"},
		{novoUsuarioReq{Nome: "Ana", Email: "ana@exemplo.com", Senha: "12345678"}, "", "ana@exemplo.com"},
		{novoUsuarioReq{Nome: "", Email: "x@exemplo.com"}, "nome", ""},
		{novoUsuarioReq{Nome: "X", Email: "sem-arroba"}, "e-mail", ""},
		{novoUsuarioReq{Nome: "X", Email: "Fulano <f@exemplo.com>"}, "e-mail", ""},
		{novoUsuarioReq{Nome: "X", Email: "f@localhost"}, "e-mail", ""},
		{novoUsuarioReq{Nome: "X", Email: "f@exemplo.com", Senha: "curta"}, "8 caracteres", ""},
	}
	for _, c := range casos {
		req := c.req
		msg := validarNovoUsuario(&req)
		if c.erro == "" {
			if msg != "" {
				t.Errorf("%+v: esperava válido, veio %q", c.req, msg)
			} else if req.Email != c.emailOut {
				t.Errorf("%+v: e-mail normalizado = %q, quer %q", c.req, req.Email, c.emailOut)
			}
			continue
		}
		if !strings.Contains(msg, c.erro) {
			t.Errorf("%+v: erro = %q, esperava conter %q", c.req, msg, c.erro)
		}
	}
}

func TestMesmoAmbienteEscopaPeloAmbiente(t *testing.T) {
	sql := mesmoAmbiente("u.id", "$2")
	if !strings.Contains(sql, "a.user_id = u.id") || !strings.Contains(sql, "b.user_id = $2") || !strings.Contains(sql, "b.environment_id = a.environment_id") {
		t.Errorf("condição de escopo inesperada: %s", sql)
	}
}
