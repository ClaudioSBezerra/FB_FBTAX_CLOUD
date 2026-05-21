package services

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"os"
)

// masterKey deriva uma chave AES-256 de FINANCEIRO_SECRET.
// Se a variável não estiver configurada, retorna nil e as funções
// usam base64 simples com aviso — suficiente para dev/staging.
func masterKey() []byte {
	secret := os.Getenv("FINANCEIRO_SECRET")
	if secret == "" {
		return nil
	}
	sum := sha256.Sum256([]byte(secret))
	return sum[:]
}

// EncryptConfig criptografa texto com AES-256-GCM.
// Retorna base64(nonce + ciphertext). Se FINANCEIRO_SECRET não estiver
// configurado, retorna base64 simples com prefixo "plain:" para
// distinguir dos dados criptografados.
func EncryptConfig(plaintext string) (string, error) {
	key := masterKey()
	if key == nil {
		log.Println("[crypto] FINANCEIRO_SECRET não configurado — config salvo sem criptografia")
		return "plain:" + base64.StdEncoding.EncodeToString([]byte(plaintext)), nil
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("gcm: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("nonce: %w", err)
	}
	sealed := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(sealed), nil
}

// DecryptConfig desfaz EncryptConfig.
func DecryptConfig(encoded string) (string, error) {
	// Dados não criptografados (fallback sem secret)
	const plainPrefix = "plain:"
	if len(encoded) > len(plainPrefix) && encoded[:len(plainPrefix)] == plainPrefix {
		decoded, err := base64.StdEncoding.DecodeString(encoded[len(plainPrefix):])
		if err != nil {
			return "", fmt.Errorf("decode plain: %w", err)
		}
		return string(decoded), nil
	}

	key := masterKey()
	if key == nil {
		return "", fmt.Errorf("FINANCEIRO_SECRET não configurado e dados estão criptografados")
	}

	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("base64: %w", err)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("gcm: %w", err)
	}
	ns := gcm.NonceSize()
	if len(data) < ns {
		return "", fmt.Errorf("dados corrompidos")
	}
	plain, err := gcm.Open(nil, data[:ns], data[ns:], nil)
	if err != nil {
		return "", fmt.Errorf("decrypt: %w", err)
	}
	return string(plain), nil
}
