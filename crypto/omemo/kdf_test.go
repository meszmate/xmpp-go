package omemo

import (
	"bytes"
	"crypto/rand"
	"testing"
)

func TestChainKDF(t *testing.T) {
	ck := make([]byte, 32)
	if _, err := rand.Read(ck); err != nil {
		t.Fatal(err)
	}

	mk, nextCK := chainKDF(ck)

	if len(mk) != 32 {
		t.Errorf("message key length = %d, want 32", len(mk))
	}
	if len(nextCK) != 32 {
		t.Errorf("next chain key length = %d, want 32", len(nextCK))
	}

	// Message key and next chain key should be different
	if bytes.Equal(mk, nextCK) {
		t.Error("message key and next chain key should differ")
	}

	// Same input should produce same output (deterministic)
	mk2, nextCK2 := chainKDF(ck)
	if !bytes.Equal(mk, mk2) {
		t.Error("chainKDF is not deterministic for message key")
	}
	if !bytes.Equal(nextCK, nextCK2) {
		t.Error("chainKDF is not deterministic for next chain key")
	}
}

func TestRootKDF(t *testing.T) {
	rk := make([]byte, 32)
	dhOut := make([]byte, 32)
	if _, err := rand.Read(rk); err != nil {
		t.Fatal(err)
	}
	if _, err := rand.Read(dhOut); err != nil {
		t.Fatal(err)
	}

	newRK, newCK, err := rootKDF(rk, dhOut)
	if err != nil {
		t.Fatal(err)
	}

	if len(newRK) != 32 {
		t.Errorf("new root key length = %d, want 32", len(newRK))
	}
	if len(newCK) != 32 {
		t.Errorf("new chain key length = %d, want 32", len(newCK))
	}
	if bytes.Equal(newRK, newCK) {
		t.Error("new root key and chain key should differ")
	}

	// Deterministic
	newRK2, newCK2, err := rootKDF(rk, dhOut)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(newRK, newRK2) {
		t.Error("rootKDF is not deterministic for root key")
	}
	if !bytes.Equal(newCK, newCK2) {
		t.Error("rootKDF is not deterministic for chain key")
	}
}

func TestHKDFSHA256(t *testing.T) {
	salt := make([]byte, 32)
	ikm := []byte("input keying material")
	info := []byte("context info")

	out, err := hkdfSHA256(salt, ikm, info, 64)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 64 {
		t.Errorf("output length = %d, want 64", len(out))
	}

	// Same inputs should produce same output
	out2, err := hkdfSHA256(salt, ikm, info, 64)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out, out2) {
		t.Error("hkdfSHA256 is not deterministic")
	}
}
