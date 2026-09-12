package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
)

// AEAD encrypter usando AES-256-GCM. Key debe ser 32 bytes.
type Cryptor struct {
	gcm cipher.AEAD
}

func NewCryptor(key string) (*Cryptor, error) {
	b := []byte(key)
	if len(b) != 32 {
		return nil, errors.New("CREDENTIAL_KEY debe tener exactamente 32 bytes")
	}
	block, err := aes.NewCipher(b)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &Cryptor{gcm: gcm}, nil
}

func (c *Cryptor) Encrypt(plaintext string) (string, error) {
	nonce := make([]byte, c.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ct := c.gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ct), nil
}

func (c *Cryptor) Decrypt(ciphertext string) (string, error) {
	b, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}
	if len(b) < c.gcm.NonceSize() {
		return "", errors.New("ciphertext too short")
	}
	nonce, ct := b[:c.gcm.NonceSize()], b[c.gcm.NonceSize():]
	pt, err := c.gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return "", err
	}
	return string(pt), nil
}
