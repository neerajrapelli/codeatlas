package vcsauth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
)

// Cipher encrypts provider tokens at rest (AES-256-GCM).
type Cipher struct {
	gcm cipher.AEAD
}

func NewCipher(keyB64 string) (*Cipher, error) {
	if keyB64 == "" {
		return nil, errors.New("TOKEN_ENCRYPTION_KEY is required for provider token storage")
	}
	raw, err := base64.StdEncoding.DecodeString(keyB64)
	if err != nil {
		return nil, fmt.Errorf("decode TOKEN_ENCRYPTION_KEY: %w", err)
	}
	if len(raw) != 32 {
		return nil, fmt.Errorf("TOKEN_ENCRYPTION_KEY must be 32 bytes (base64-encoded), got %d", len(raw))
	}
	block, err := aes.NewCipher(raw)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &Cipher{gcm: gcm}, nil
}

func (c *Cipher) Encrypt(plaintext []byte) ([]byte, error) {
	nonce := make([]byte, c.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return c.gcm.Seal(nonce, nonce, plaintext, nil), nil
}

func (c *Cipher) Decrypt(ciphertext []byte) ([]byte, error) {
	ns := c.gcm.NonceSize()
	if len(ciphertext) < ns {
		return nil, errors.New("ciphertext too short")
	}
	return c.gcm.Open(nil, ciphertext[:ns], ciphertext[ns:], nil)
}
