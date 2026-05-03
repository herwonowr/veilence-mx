// Package crypto provides AES-256-GCM encryption and decryption utilities.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
)

// AESEncryptor implements AES-256-GCM encrypt/decrypt for secret storage at rest.
type AESEncryptor struct {
	key []byte
}

// NewAESEncryptor creates a new AES-256-GCM encryptor from a 32-byte hex-encoded key (64 hex chars).
func NewAESEncryptor(hexKey string) (*AESEncryptor, error) {
	key, err := hex.DecodeString(hexKey)
	if err != nil {
		return nil, fmt.Errorf("crypto.NewAESEncryptor: invalid hex key: %w", err)
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("crypto.NewAESEncryptor: key must be 32 bytes (64 hex chars), got %d bytes", len(key))
	}
	return &AESEncryptor{key: key}, nil
}

// Encrypt encrypts plaintext using AES-256-GCM and returns a base64-encoded ciphertext
// (nonce prepended to the ciphertext).
func (e *AESEncryptor) Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", fmt.Errorf("crypto.Encrypt: creating cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("crypto.Encrypt: creating GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("crypto.Encrypt: generating nonce: %w", err)
	}

	// Seal appends the ciphertext to nonce, so result = nonce || ciphertext || tag.
	sealed := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(sealed), nil
}

// Decrypt decrypts a base64-encoded ciphertext produced by Encrypt.
func (e *AESEncryptor) Decrypt(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}

	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("crypto.Decrypt: decoding base64: %w", err)
	}

	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", fmt.Errorf("crypto.Decrypt: creating cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("crypto.Decrypt: creating GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("crypto.Decrypt: ciphertext too short")
	}

	nonce, sealed := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, sealed, nil)
	if err != nil {
		return "", fmt.Errorf("crypto.Decrypt: decryption failed: %w", err)
	}

	return string(plaintext), nil
}
