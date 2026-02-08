package omemo

import (
	"bytes"
	"crypto/ecdh"
	"testing"
)

func TestX3DHWithOneTimePreKey(t *testing.T) {
	// Alice's identity
	alice, err := GenerateIdentityKeyPair()
	if err != nil {
		t.Fatal(err)
	}

	// Bob generates a bundle
	bobStore := NewMemoryStore(1)
	bobBundle, err := GenerateBundle(bobStore, 5)
	if err != nil {
		t.Fatal(err)
	}

	if len(bobBundle.PreKeys) != 5 {
		t.Fatalf("expected 5 pre-keys, got %d", len(bobBundle.PreKeys))
	}

	// Save the first pre-key ID for Bob's side
	usedPreKeyPub := make([]byte, 32)
	copy(usedPreKeyPub, bobBundle.PreKeys[0].PublicKey)
	usedPreKeyID := bobBundle.PreKeys[0].ID

	// Alice initiates X3DH
	result, err := X3DHInitiate(alice, bobBundle)
	if err != nil {
		t.Fatal(err)
	}

	if len(result.SharedSecret) != 32 {
		t.Errorf("shared secret length = %d, want 32", len(result.SharedSecret))
	}
	if len(result.EphemeralPubKey) != 32 {
		t.Errorf("ephemeral pub key length = %d, want 32", len(result.EphemeralPubKey))
	}
	if result.UsedPreKeyID == nil {
		t.Fatal("expected UsedPreKeyID to be set")
	}
	if *result.UsedPreKeyID != usedPreKeyID {
		t.Errorf("UsedPreKeyID = %d, want %d", *result.UsedPreKeyID, usedPreKeyID)
	}

	// Bob responds
	bobIdentity, err := bobStore.GetIdentityKeyPair()
	if err != nil {
		t.Fatal(err)
	}

	// Get Bob's SPK private key
	spkRecord, err := bobStore.GetSignedPreKey(bobBundle.SignedPreKeyID)
	if err != nil {
		t.Fatal(err)
	}
	spkPrivate, err := newX25519PrivateKey(spkRecord.PrivateKey)
	if err != nil {
		t.Fatal(err)
	}

	// Get Bob's OPK private key
	opkRecord, err := bobStore.GetPreKey(usedPreKeyID)
	if err != nil {
		t.Fatal(err)
	}
	opkPrivate, err := newX25519PrivateKey(opkRecord.PrivateKey)
	if err != nil {
		t.Fatal(err)
	}

	bobSS, err := X3DHRespond(bobIdentity, spkPrivate, opkPrivate, alice.PublicKey, result.EphemeralPubKey)
	if err != nil {
		t.Fatal(err)
	}

	// Shared secrets should match
	if !bytes.Equal(result.SharedSecret, bobSS) {
		t.Error("shared secrets do not match")
	}
}

func TestX3DHWithoutOneTimePreKey(t *testing.T) {
	alice, err := GenerateIdentityKeyPair()
	if err != nil {
		t.Fatal(err)
	}

	// Bob generates a bundle with 0 pre-keys
	bobStore := NewMemoryStore(2)
	bobBundle, err := GenerateBundle(bobStore, 0)
	if err != nil {
		t.Fatal(err)
	}

	result, err := X3DHInitiate(alice, bobBundle)
	if err != nil {
		t.Fatal(err)
	}
	if result.UsedPreKeyID != nil {
		t.Error("expected no UsedPreKeyID")
	}

	bobIdentity, err := bobStore.GetIdentityKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	spkRecord, err := bobStore.GetSignedPreKey(bobBundle.SignedPreKeyID)
	if err != nil {
		t.Fatal(err)
	}
	spkPrivate, err := newX25519PrivateKey(spkRecord.PrivateKey)
	if err != nil {
		t.Fatal(err)
	}

	bobSS, err := X3DHRespond(bobIdentity, spkPrivate, nil, alice.PublicKey, result.EphemeralPubKey)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(result.SharedSecret, bobSS) {
		t.Error("shared secrets do not match")
	}
}

func TestX3DHInvalidSignature(t *testing.T) {
	alice, err := GenerateIdentityKeyPair()
	if err != nil {
		t.Fatal(err)
	}

	bobStore := NewMemoryStore(3)
	bobBundle, err := GenerateBundle(bobStore, 1)
	if err != nil {
		t.Fatal(err)
	}

	// Corrupt the signature
	bobBundle.SignedPreKeySignature[0] ^= 0xFF

	_, err = X3DHInitiate(alice, bobBundle)
	if err != ErrInvalidSignature {
		t.Errorf("expected ErrInvalidSignature, got %v", err)
	}
}

func newX25519PrivateKey(data []byte) (*ecdh.PrivateKey, error) {
	return ecdh.X25519().NewPrivateKey(data)
}
