// Package secretbox provides AES-256-GCM authenticated encryption with a
// per-record random salt. The data-encryption key is derived from a master
// secret (typically the configured JWT secret or a dedicated env var) using
// SHA-256(master || salt).
//
// Storage layout (per record):
//   - salt:        16 random bytes
//   - nonce:       12 random bytes (GCM standard nonce size)
//   - ciphertext:  AES-256-GCM(plaintext) including auth tag
package secretbox

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"io"
)

const (
	saltLen  = 16
	nonceLen = 12
)

// Box encrypts and decrypts secrets using a master key.
type Box struct {
	master []byte
}

// New returns a Box. The master key may be any non-empty byte slice; longer is
// fine. An empty master returns nil so callers can detect "encryption disabled".
func New(master []byte) *Box {
	if len(master) == 0 {
		return nil
	}
	cp := make([]byte, len(master))
	copy(cp, master)
	return &Box{master: cp}
}

func (b *Box) deriveKey(salt []byte) []byte {
	h := sha256.New()
	h.Write(b.master)
	h.Write(salt)
	return h.Sum(nil)
}

// Seal encrypts plaintext and returns (ciphertext, salt, nonce). All three
// must be stored together to allow later decryption.
func (b *Box) Seal(plaintext []byte) (ct, salt, nonce []byte, err error) {
	if b == nil {
		return nil, nil, nil, errors.New("secretbox: not initialized")
	}
	salt = make([]byte, saltLen)
	if _, err = io.ReadFull(rand.Reader, salt); err != nil {
		return nil, nil, nil, err
	}
	nonce = make([]byte, nonceLen)
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, nil, nil, err
	}
	block, err := aes.NewCipher(b.deriveKey(salt))
	if err != nil {
		return nil, nil, nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, nil, err
	}
	ct = gcm.Seal(nil, nonce, plaintext, nil)
	return ct, salt, nonce, nil
}

// Open decrypts ciphertext using the stored salt and nonce.
func (b *Box) Open(ct, salt, nonce []byte) ([]byte, error) {
	if b == nil {
		return nil, errors.New("secretbox: not initialized")
	}
	if len(salt) != saltLen || len(nonce) != nonceLen {
		return nil, errors.New("secretbox: invalid salt/nonce length")
	}
	block, err := aes.NewCipher(b.deriveKey(salt))
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return gcm.Open(nil, nonce, ct, nil)
}
