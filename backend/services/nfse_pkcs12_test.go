package services

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// pfxViaOpenssl gera um .pfx de teste com o openssl local, nas cifras pedidas.
func pfxViaOpenssl(t *testing.T, extra ...string) []byte {
	t.Helper()
	if _, err := exec.LookPath("openssl"); err != nil {
		t.Skip("openssl não disponível")
	}
	dir := t.TempDir()
	run := func(args ...string) {
		if out, err := exec.Command("openssl", args...).CombinedOutput(); err != nil {
			t.Skipf("openssl %v: %v: %s", args[:1], err, out)
		}
	}
	run("req", "-x509", "-newkey", "rsa:2048", "-nodes", "-keyout", filepath.Join(dir, "k.pem"), "-out", filepath.Join(dir, "c.pem"),
		"-days", "2", "-subj", "/CN=TESTE:38149716000128")
	args := append([]string{"pkcs12", "-export", "-in", filepath.Join(dir, "c.pem"), "-inkey", filepath.Join(dir, "k.pem"), "-out", filepath.Join(dir, "a.pfx"), "-passout", "pass:senha"}, extra...)
	run(args...)
	b, err := os.ReadFile(filepath.Join(dir, "a.pfx"))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestNFSeNormalizarPKCS12_Variantes(t *testing.T) {
	for nome, extra := range map[string][]string{"moderno": nil, "legado": {"-legacy"}} {
		t.Run(nome, func(t *testing.T) {
			pfx := pfxViaOpenssl(t, extra...)
			norm, _, err := NormalizarPKCS12ParaDER(pfx, "senha")
			if err != nil {
				t.Fatalf("normalizar: %v", err)
			}
			a1, err := DecodificarPFX(norm, "senha")
			if err != nil {
				t.Fatalf("decodificar: %v", err)
			}
			if CNPJDoCertificado(a1.Cert) != "38149716000128" {
				t.Fatal("CNPJ do certificado não confere")
			}
			if _, _, err := NormalizarPKCS12ParaDER(pfx, "errada"); err == nil {
				t.Fatal("senha errada deveria falhar")
			}
		})
	}
}

func TestNFSeDecodificarPFX_SenhaIncorreta(t *testing.T) {
	pfx := pfxViaOpenssl(t)
	if _, err := DecodificarPFX(pfx, "errada"); err != ErrNFSeSenhaIncorreta {
		t.Fatalf("esperava ErrNFSeSenhaIncorreta, veio %v", err)
	}
}
