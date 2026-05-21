package services

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"strings"
	"time"
	"encoding/base64"
)

// buildMTLSClient cria um http.Client com mTLS se certPEM/keyPEM fornecidos,
// ou um client HTTPS simples caso contrário.
// Aceita PEM bruto, PEM com \n literal (para env vars de linha única), ou base64.
func buildMTLSClient(certPEM, keyPEM string) (*http.Client, error) {
	if certPEM == "" || keyPEM == "" {
		return &http.Client{Timeout: 30 * time.Second}, nil
	}
	cert, err := tls.X509KeyPair(decodePEM(certPEM), decodePEM(keyPEM))
	if err != nil {
		return nil, fmt.Errorf("certificado mTLS inválido: %w", err)
	}
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{Certificates: []tls.Certificate{cert}},
	}
	return &http.Client{Transport: transport, Timeout: 30 * time.Second}, nil
}

// decodePEM aceita:
// 1. PEM com headers (-----BEGIN...) — retorna como-está
// 2. PEM com \n literal (env var em linha única) — substitui por \n real
// 3. Base64 puro — decodifica
func decodePEM(s string) []byte {
	s = strings.TrimSpace(s)
	if strings.Contains(s, "-----BEGIN") {
		return []byte(strings.ReplaceAll(s, `\n`, "\n"))
	}
	if decoded, err := base64.StdEncoding.DecodeString(s); err == nil {
		return decoded
	}
	// Fallback: trata como PEM com \n literal
	return []byte(strings.ReplaceAll(s, `\n`, "\n"))
}
