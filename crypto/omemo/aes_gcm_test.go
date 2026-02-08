package omemo

import (
	"bytes"
	"crypto/rand"
	"testing"
)

func TestAESGCMRoundtrip(t *testing.T) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatal(err)
	}

	plaintext := []byte("Hello, OMEMO!")
	nonce, ciphertext, err := aesGCMEncrypt(key, plaintext)
	if err != nil {
		t.Fatal(err)
	}

	if len(nonce) != aesNonceSize {
		t.Errorf("nonce length = %d, want %d", len(nonce), aesNonceSize)
	}

	// Ciphertext should be longer than plaintext (includes auth tag)
	if len(ciphertext) <= len(plaintext) {
		t.Error("ciphertext should be longer than plaintext")
	}

	decrypted, err := aesGCMDecrypt(key, nonce, ciphertext)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(plaintext, decrypted) {
		t.Errorf("decrypted = %q, want %q", decrypted, plaintext)
	}
}

func TestAESGCMInvalidKey(t *testing.T) {
	_, _, err := aesGCMEncrypt([]byte{1, 2, 3}, []byte("test"))
	if err != ErrInvalidKeyLength {
		t.Errorf("expected ErrInvalidKeyLength, got %v", err)
	}

	_, err = aesGCMDecrypt([]byte{1, 2, 3}, make([]byte, 12), []byte("test"))
	if err != ErrInvalidKeyLength {
		t.Errorf("expected ErrInvalidKeyLength, got %v", err)
	}
}

func TestAESGCMTamper(t *testing.T) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatal(err)
	}

	nonce, ciphertext, err := aesGCMEncrypt(key, []byte("secret"))
	if err != nil {
		t.Fatal(err)
	}

	// Tamper with ciphertext
	ciphertext[0] ^= 0xFF
	_, err = aesGCMDecrypt(key, nonce, ciphertext)
	if err != ErrInvalidMessage {
		t.Errorf("expected ErrInvalidMessage, got %v", err)
	}
}

func TestAESGCMInvalidNonce(t *testing.T) {
	key := make([]byte, 32)
	_, err := aesGCMDecrypt(key, []byte{1, 2, 3}, []byte("test"))
	if err != ErrInvalidMessage {
		t.Errorf("expected ErrInvalidMessage, got %v", err)
	}
}

func TestAESGCMEmptyPlaintext(t *testing.T) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatal(err)
	}

	nonce, ciphertext, err := aesGCMEncrypt(key, []byte{})
	if err != nil {
		t.Fatal(err)
	}

	decrypted, err := aesGCMDecrypt(key, nonce, ciphertext)
	if err != nil {
		t.Fatal(err)
	}

	if len(decrypted) != 0 {
		t.Errorf("decrypted length = %d, want 0", len(decrypted))
	}
}
