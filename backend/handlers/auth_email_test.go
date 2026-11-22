package handlers

import "testing"

func TestNormalizarEmail(t *testing.T) {
	casos := map[string]string{
		"Claudio_Bezerra@Hotmail.com ": "claudio_bezerra@hotmail.com",
		"  user@x.com":                 "user@x.com",
		"user@x.com":                   "user@x.com",
	}
	for in, quer := range casos {
		if got := normalizarEmail(in); got != quer {
			t.Errorf("normalizarEmail(%q) = %q, quer %q", in, got, quer)
		}
	}
}
