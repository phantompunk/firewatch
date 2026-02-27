package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"log/slog"
	"strings"
)

// Crypter encrypts and decrypts data using AES-256-GCM.
type Crypter struct {
	key []byte
}

// New creates a Crypter. key must be exactly 32 bytes.
func New(key []byte) *Crypter {
	if len(key) != 32 {
		panic("crypto: key must be 32 bytes")
	}
	return &Crypter{key: key}
}

// Encrypt encrypts plaintext using AES-256-GCM and returns ciphertext with
// the nonce prepended.
func (c *Crypter) Encrypt(plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(c.key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// EmailHMAC normalises the email address (lowercase, trimmed) and returns its
// HMAC-SHA256 hex digest using the provided key.
func EmailHMAC(key []byte, email string) string {
	normalised := strings.ToLower(strings.TrimSpace(email))
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(normalised))
	return hex.EncodeToString(mac.Sum(nil))
}

// Decrypt decrypts ciphertext produced by Encrypt.
func (c *Crypter) Decrypt(ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(c.key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(ciphertext) < gcm.NonceSize() {
		slog.Error("crypto: ciphertext too short", "length", len(ciphertext), "nonce_size", gcm.NonceSize())
		return nil, errors.New("crypto: ciphertext too short")
	}
	nonce, ciphertext := ciphertext[:gcm.NonceSize()], ciphertext[gcm.NonceSize():]
	return gcm.Open(nil, nonce, ciphertext, nil)
}
