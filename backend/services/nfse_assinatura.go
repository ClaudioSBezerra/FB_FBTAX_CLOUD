package services

// Assinatura XMLDSig do DPS (enveloped, RSA-SHA256, canonicalização exclusiva) com o A1 do
// prestador, e carga do certificado a partir do cofre cifrado (financeiro.nfse_certificado).
//
// O DPS é montado já em forma canônica (nfse_dps.go): sem espaços entre elementos, sem
// elementos auto-fechados, texto escapado como o C14N manda. Por isso o digest é calculado
// direto sobre a string do infDPS, sem biblioteca de canonicalização.

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"software.sslmate.com/src/go-pkcs12"
)

const (
	dsigNamespace = "http://www.w3.org/2000/09/xmldsig#"
	algC14NExc    = "http://www.w3.org/2001/10/xml-exc-c14n#"
	algEnveloped  = "http://www.w3.org/2000/09/xmldsig#enveloped-signature"
	algRSASHA256  = "http://www.w3.org/2001/04/xmldsig-more#rsa-sha256"
	algSHA256     = "http://www.w3.org/2001/04/xmlenc#sha256"
)

// ErrNFSeSenhaIncorreta indica que a senha informada não abre o .pfx.
var ErrNFSeSenhaIncorreta = errors.New("senha do certificado incorreta")

// NFSeCertificadoA1 é o par chave/certificado já decodificado do PFX.
type NFSeCertificadoA1 struct {
	Chave  crypto.Signer
	Cert   *x509.Certificate
	Cadeia []*x509.Certificate // ACs intermediárias (completadas no upload, ver nfse_cadeia.go)
}

// CarregarCertificadoA1 decifra e decodifica o PFX guardado no cofre. Os dois argumentos
// são os valores cifrados por EncryptConfig (o PFX em base64 antes de cifrar).
func CarregarCertificadoA1(pfxCifrado, senhaCifrada string) (*NFSeCertificadoA1, error) {
	if strings.HasPrefix(pfxCifrado, "plain:") || strings.HasPrefix(senhaCifrada, "plain:") {
		return nil, errors.New("certificado guardado sem criptografia: reenvie o certificado com FINANCEIRO_SECRET configurado")
	}
	pfxB64, err := DecryptConfig(pfxCifrado)
	if err != nil {
		return nil, fmt.Errorf("decifrar pfx: %w", err)
	}
	senha, err := DecryptConfig(senhaCifrada)
	if err != nil {
		return nil, fmt.Errorf("decifrar senha: %w", err)
	}
	pfx, err := base64.StdEncoding.DecodeString(pfxB64)
	if err != nil {
		return nil, fmt.Errorf("pfx não é base64 válido: %w", err)
	}
	return DecodificarPFX(pfx, senha)
}

// DecodificarPFX lê um PKCS#12. Só chaves RSA são aceitas (exigência do algoritmo rsa-sha256).
func DecodificarPFX(pfx []byte, senha string) (*NFSeCertificadoA1, error) {
	chave, cert, cadeia, err := pkcs12.DecodeChain(pfx, senha)
	if errors.Is(err, pkcs12.ErrIncorrectPassword) {
		return nil, ErrNFSeSenhaIncorreta
	}
	if err != nil {
		return nil, fmt.Errorf("arquivo de certificado não pôde ser lido: %w", err)
	}
	rsaKey, ok := chave.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("o certificado não contém chave RSA")
	}
	return &NFSeCertificadoA1{Chave: rsaKey, Cert: cert, Cadeia: cadeia}, nil
}

var reCNPJCN = regexp.MustCompile(`:(\d{14})\s*$`)

// CNPJDoCertificado extrai o CNPJ do CN do certificado e-CNPJ ICP-Brasil ("RAZAO SOCIAL:00000000000000").
// Vazio quando o CN não segue o padrão.
func CNPJDoCertificado(cert *x509.Certificate) string {
	if m := reCNPJCN.FindStringSubmatch(cert.Subject.CommonName); m != nil {
		return m[1]
	}
	return ""
}

// assinarXMLDSig assina o elemento apex (já canônico, com o namespace declarado) referenciado
// por #id e devolve o bloco <Signature>. Serve ao DPS e ao pedido de registro de evento.
func assinarXMLDSig(id, apexCanonico string, a1 *NFSeCertificadoA1) (string, error) {
	if a1 == nil || a1.Chave == nil || a1.Cert == nil {
		return "", errors.New("certificado A1 não informado")
	}
	if id == "" || apexCanonico == "" {
		return "", errors.New("elemento a assinar vazio")
	}
	resumo := sha256.Sum256([]byte(apexCanonico))
	digest := base64.StdEncoding.EncodeToString(resumo[:])

	// Corpo do SignedInfo: elementos expandidos, sem espaços (já canônico).
	corpoSI := "<CanonicalizationMethod Algorithm=\"" + algC14NExc + "\"></CanonicalizationMethod>" +
		"<SignatureMethod Algorithm=\"" + algRSASHA256 + "\"></SignatureMethod>" +
		"<Reference URI=\"#" + xmlAttr(id) + "\">" +
		"<Transforms>" +
		"<Transform Algorithm=\"" + algEnveloped + "\"></Transform>" +
		"<Transform Algorithm=\"" + algC14NExc + "\"></Transform>" +
		"</Transforms>" +
		"<DigestMethod Algorithm=\"" + algSHA256 + "\"></DigestMethod>" +
		"<DigestValue>" + digest + "</DigestValue>" +
		"</Reference>"

	// O verificador canonicaliza o SignedInfo com o xmlns da dsig declarado no próprio apex.
	hash := sha256.Sum256([]byte("<SignedInfo xmlns=\"" + dsigNamespace + "\">" + corpoSI + "</SignedInfo>"))
	sig, err := a1.Chave.Sign(rand.Reader, hash[:], crypto.SHA256)
	if err != nil {
		return "", fmt.Errorf("assinar: %w", err)
	}
	return "<Signature xmlns=\"" + dsigNamespace + "\">" +
		"<SignedInfo>" + corpoSI + "</SignedInfo>" +
		"<SignatureValue>" + base64.StdEncoding.EncodeToString(sig) + "</SignatureValue>" +
		"<KeyInfo><X509Data><X509Certificate>" + base64.StdEncoding.EncodeToString(a1.Cert.Raw) +
		"</X509Certificate></X509Data></KeyInfo></Signature>", nil
}

// AssinarDPS devolve o XML completo do DPS com o <Signature> sobre o infDPS. Recusa assinar
// se o certificado estiver fora da validade ou não pertencer ao prestador do DPS (o CNPJ do
// emitente está no Id): assim o número de DPS não é gasto com uma nota fadada à rejeição.
func AssinarDPS(g DPSGerado, a1 *NFSeCertificadoA1) (string, error) {
	canon := g.infDPSCanonico()
	if a1 == nil || a1.Cert == nil || g.ID == "" || canon == "" {
		return "", errors.New("DPS ou certificado vazio")
	}
	if err := ValidarCertificadoDoPrestador(a1.Cert, g.ID[11:25], time.Now()); err != nil {
		return "", err
	}
	sig, err := assinarXMLDSig(g.ID, canon, a1)
	if err != nil {
		return "", err
	}
	return g.montarXML(sig), nil
}

// ValidarCertificadoDoPrestador confere validade e titularidade (CNPJ) do certificado.
func ValidarCertificadoDoPrestador(cert *x509.Certificate, cnpjPrestador string, agora time.Time) error {
	if agora.Before(cert.NotBefore) || agora.After(cert.NotAfter) {
		return fmt.Errorf("certificado fora da validade (%s a %s)", cert.NotBefore.Format("02/01/2006"), cert.NotAfter.Format("02/01/2006"))
	}
	if got := CNPJDoCertificado(cert); got != cnpjPrestador {
		return fmt.Errorf("o certificado pertence ao CNPJ %q, não ao prestador %s", got, cnpjPrestador)
	}
	return nil
}

// CifrarCertificadoA1 prepara o PFX e a senha para o cofre. Diferente de EncryptConfig, recusa
// gravar quando FINANCEIRO_SECRET não está configurado: um certificado A1 nunca pode ir para
// o banco só em base64.
func CifrarCertificadoA1(pfx []byte, senha string) (pfxCifrado, senhaCifrada string, validade time.Time, err error) {
	if masterKey() == nil {
		return "", "", time.Time{}, errors.New("FINANCEIRO_SECRET não configurado: o certificado não pode ser guardado sem criptografia")
	}
	a1, err := DecodificarPFX(pfx, senha)
	if err != nil {
		return "", "", time.Time{}, err
	}
	if time.Now().After(a1.Cert.NotAfter) {
		return "", "", time.Time{}, fmt.Errorf("certificado vencido em %s", a1.Cert.NotAfter.Format("02/01/2006"))
	}
	if pfxCifrado, err = EncryptConfig(base64.StdEncoding.EncodeToString(pfx)); err != nil {
		return "", "", time.Time{}, err
	}
	if senhaCifrada, err = EncryptConfig(senha); err != nil {
		return "", "", time.Time{}, err
	}
	return pfxCifrado, senhaCifrada, a1.Cert.NotAfter, nil
}

// CodificarPFXComCadeia reempacota chave, certificado e ACs intermediárias num .pfx moderno
// (AES-256/PBKDF2), que o próprio Go consegue ler de volta.
func CodificarPFXComCadeia(a1 *NFSeCertificadoA1, senha string) ([]byte, error) {
	return pkcs12.Modern.Encode(a1.Chave, a1.Cert, a1.Cadeia, senha)
}
