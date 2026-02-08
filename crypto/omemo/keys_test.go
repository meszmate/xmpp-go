package omemo

import (
	"crypto/ed25519"
	"testing"
)

func TestGenerateIdentityKeyPair(t *testing.T) {
	ikp, err := GenerateIdentityKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	if len(ikp.PublicKey) != ed25519.PublicKeySize {
		t.Errorf("public key length = %d, want %d", len(ikp.PublicKey), ed25519.PublicKeySize)
	}
	if len(ikp.PrivateKey) != ed25519.PrivateKeySize {
		t.Errorf("private key length = %d, want %d", len(ikp.PrivateKey), ed25519.PrivateKeySize)
	}

	// Sign and verify roundtrip
	msg := []byte("test message")
	sig := ed25519.Sign(ikp.PrivateKey, msg)
	if !ed25519.Verify(ikp.PublicKey, msg, sig) {
		t.Error("signature verification failed")
	}
}

func TestGenerateX25519KeyPair(t *testing.T) {
	key, err := GenerateX25519KeyPair()
	if err != nil {
		t.Fatal(err)
	}
	if len(key.Bytes()) != 32 {
		t.Errorf("private key length = %d, want 32", len(key.Bytes()))
	}
	if len(key.PublicKey().Bytes()) != 32 {
		t.Errorf("public key length = %d, want 32", len(key.PublicKey().Bytes()))
	}
}

func TestEd25519ToX25519Roundtrip(t *testing.T) {
	ikp, err := GenerateIdentityKeyPair()
	if err != nil {
		t.Fatal(err)
	}

	// Convert private key
	x25519Priv, err := Ed25519PrivateKeyToX25519(ikp.PrivateKey)
	if err != nil {
		t.Fatal("private key conversion:", err)
	}

	// Convert public key
	x25519Pub, err := Ed25519PublicKeyToX25519(ikp.PublicKey)
	if err != nil {
		t.Fatal("public key conversion:", err)
	}

	// The X25519 public key derived from the private key should match
	// the X25519 public key derived from the Ed25519 public key
	derivedPub := x25519Priv.PublicKey().Bytes()
	if len(derivedPub) != len(x25519Pub) {
		t.Fatalf("derived public key length %d != converted public key length %d", len(derivedPub), len(x25519Pub))
	}
	for i := range derivedPub {
		if derivedPub[i] != x25519Pub[i] {
			t.Fatalf("derived public key differs from converted public key at byte %d", i)
		}
	}
}

func TestEd25519ToX25519DH(t *testing.T) {
	// Generate two identity key pairs
	alice, err := GenerateIdentityKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	bob, err := GenerateIdentityKeyPair()
	if err != nil {
		t.Fatal(err)
	}

	// Convert to X25519
	alicePriv, err := Ed25519PrivateKeyToX25519(alice.PrivateKey)
	if err != nil {
		t.Fatal(err)
	}
	bobPriv, err := Ed25519PrivateKeyToX25519(bob.PrivateKey)
	if err != nil {
		t.Fatal(err)
	}
	bobPub, err := Ed25519PublicKeyToX25519(bob.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	alicePub, err := Ed25519PublicKeyToX25519(alice.PublicKey)
	if err != nil {
		t.Fatal(err)
	}

	// DH should produce the same shared secret from both sides
	ss1, err := x25519DH(alicePriv, bobPub)
	if err != nil {
		t.Fatal(err)
	}
	ss2, err := x25519DH(bobPriv, alicePub)
	if err != nil {
		t.Fatal(err)
	}

	if len(ss1) != 32 || len(ss2) != 32 {
		t.Fatal("unexpected shared secret length")
	}
	for i := range ss1 {
		if ss1[i] != ss2[i] {
			t.Fatalf("shared secrets differ at byte %d", i)
		}
	}
}

func TestEd25519PublicKeyToX25519InvalidLength(t *testing.T) {
	_, err := Ed25519PublicKeyToX25519([]byte{1, 2, 3})
	if err != ErrInvalidKeyLength {
		t.Errorf("expected ErrInvalidKeyLength, got %v", err)
	}
}
