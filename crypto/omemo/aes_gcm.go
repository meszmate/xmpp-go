package omemo

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
)

const (
	aesKeySize   = 32 // AES-256
	aesNonceSize = 12 // GCM standard nonce
	aesTagSize   = 16 // GCM auth tag
)

// aesGCMEncrypt encrypts plaintext with AES-256-GCM.
// Returns (nonce, ciphertext || authTag).
func aesGCMEncrypt(key, plaintext []byte) (nonce, ciphertext []byte, err error) {
	if len(key) != aesKeySize {
		return nil, nil, ErrInvalidKeyLength
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, err
	}

	nonce = make([]byte, aesNonceSize)
	if _, err := rand.Read(nonce); err != nil {
		return nil, nil, err
	}

	ciphertext = gcm.Seal(nil, nonce, plaintext, nil)
	return nonce, ciphertext, nil
}

// aesGCMDecrypt decrypts ciphertext with AES-256-GCM.
// ciphertext must include the auth tag appended.
func aesGCMDecrypt(key, nonce, ciphertext []byte) ([]byte, error) {
	if len(key) != aesKeySize {
		return nil, ErrInvalidKeyLength
	}
	if len(nonce) != aesNonceSize {
		return nil, ErrInvalidMessage
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, ErrInvalidMessage
	}

	return plaintext, nil
}
