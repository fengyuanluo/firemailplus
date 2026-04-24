package models

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
	"os"
	"strings"
)

const encryptedValuePrefix = "enc:v1:"

func credentialKey() ([]byte, error) {
	secret := os.Getenv("FIREMAIL_CREDENTIAL_KEY")
	if secret == "" {
		secret = os.Getenv("CREDENTIAL_ENCRYPTION_KEY")
	}
	if secret == "" {
		secret = os.Getenv("JWT_SECRET")
	}
	if len(secret) < 24 {
		return nil, errors.New("FIREMAIL_CREDENTIAL_KEY/CREDENTIAL_ENCRYPTION_KEY must be set and at least 24 characters")
	}
	sum := sha256.Sum256([]byte(secret))
	return sum[:], nil
}

func isEncryptedValue(value string) bool {
	return strings.HasPrefix(value, encryptedValuePrefix)
}

func EncryptCredential(value string) (string, error) {
	if value == "" || isEncryptedValue(value) {
		return value, nil
	}
	key, err := credentialKey()
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	sealed := gcm.Seal(nonce, nonce, []byte(value), nil)
	return encryptedValuePrefix + base64.RawURLEncoding.EncodeToString(sealed), nil
}

func DecryptCredential(value string) (string, error) {
	if value == "" || !isEncryptedValue(value) {
		return value, nil
	}
	key, err := credentialKey()
	if err != nil {
		return "", err
	}
	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(value, encryptedValuePrefix))
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(raw) < gcm.NonceSize() {
		return "", errors.New("encrypted credential is too short")
	}
	nonce, ciphertext := raw[:gcm.NonceSize()], raw[gcm.NonceSize():]
	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}
