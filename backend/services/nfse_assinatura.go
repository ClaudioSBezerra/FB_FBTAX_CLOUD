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

	"software.sslmate.com/src/go-pkcs12"
)

const (
	dsigNamespace = "http://www.w3.org/2000/09/xmldsig#"
	algC14NExc    = "http://www.w3.org/2001/10/xml-exc-c14n#"
	algEnveloped  = "http://www.w3.org/2000/09/xmldsig#enveloped-signature"
	algRSASHA256  = "http://www.w3.org/2001/04/xmldsig-more#rsa-sha256"
	algSHA256     = "http://www.w3.org/2001/04/xmlenc#sha256"
)

// NFSeCertificadoA1 é o par chave/certificado já decodificado do PFX.
type NFSeCertificadoA1 struct {
	Chave crypto.Signer
	Cert  *x509.Certificate
}

// CarregarCertificadoA1 decifra e decodifica o PFX guardado no cofre. Os dois argumentos
// são os valores cifrados por EncryptConfig (o PFX em base64 antes de cifrar).
func CarregarCertificadoA1(pfxCifrado, senhaCifrada string) (*NFSeCertificadoA1, error) {
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
	chave, cert, _, err := pkcs12.DecodeChain(pfx, senha)
	if err != nil {
		return nil, fmt.Errorf("certificado inválido ou senha incorreta: %w", err)
	}
	rsaKey, ok := chave.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("o certificado não contém chave RSA")
	}
	return &NFSeCertificadoA1{Chave: rsaKey, Cert: cert}, nil
}

var reCNPJ14 = regexp.MustCompile(`\d{14}`)

// CNPJDoCertificado extrai o CNPJ do CN do certificado ICP-Brasil ("RAZAO:00000000000000").
func CNPJDoCertificado(cert *x509.Certificate) string {
	return reCNPJ14.FindString(cert.Subject.CommonName)
}

// AssinarDPS devolve o XML completo do DPS com o <Signature> sobre o infDPS.
func AssinarDPS(g DPSGerado, a1 *NFSeCertificadoA1) (string, error) {
	if a1 == nil || a1.Chave == nil || a1.Cert == nil {
		return "", errors.New("certificado A1 não informado")
	}
	if g.ID == "" || g.InfDPS == "" {
		return "", errors.New("DPS vazio")
	}

	resumo := sha256.Sum256([]byte(g.infDPSCanonico()))
	digest := base64.StdEncoding.EncodeToString(resumo[:])

	// Corpo do SignedInfo: elementos expandidos, sem espaços (já canônico).
	corpoSI := "<CanonicalizationMethod Algorithm=\"" + algC14NExc + "\"></CanonicalizationMethod>" +
		"<SignatureMethod Algorithm=\"" + algRSASHA256 + "\"></SignatureMethod>" +
		"<Reference URI=\"#" + xmlAttr(g.ID) + "\">" +
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

	assinatura := "<Signature xmlns=\"" + dsigNamespace + "\">" +
		"<SignedInfo>" + corpoSI + "</SignedInfo>" +
		"<SignatureValue>" + base64.StdEncoding.EncodeToString(sig) + "</SignatureValue>" +
		"<KeyInfo><X509Data><X509Certificate>" + base64.StdEncoding.EncodeToString(a1.Cert.Raw) +
		"</X509Certificate></X509Data></KeyInfo></Signature>"

	return g.montarXML(assinatura), nil
}

// CifrarCertificadoA1 prepara o PFX e a senha para o cofre. Diferente de EncryptConfig, recusa
// gravar quando FINANCEIRO_SECRET não está configurado: um certificado A1 nunca pode ir para
// o banco só em base64.
func CifrarCertificadoA1(pfx []byte, senha string) (pfxCifrado, senhaCifrada string, err error) {
	if masterKey() == nil {
		return "", "", errors.New("FINANCEIRO_SECRET não configurado: o certificado não pode ser guardado sem criptografia")
	}
	if _, err := DecodificarPFX(pfx, senha); err != nil {
		return "", "", err
	}
	if pfxCifrado, err = EncryptConfig(base64.StdEncoding.EncodeToString(pfx)); err != nil {
		return "", "", err
	}
	if senhaCifrada, err = EncryptConfig(senha); err != nil {
		return "", "", err
	}
	return pfxCifrado, senhaCifrada, nil
}
