package omemo

import (
	"bytes"
	"crypto/ed25519"
	"testing"
)

func TestMemoryStoreIdentityKey(t *testing.T) {
	store := NewMemoryStore(1)

	ikp, err := store.GetIdentityKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	if ikp != nil {
		t.Error("expected nil before save")
	}

	newIKP, err := GenerateIdentityKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveIdentityKeyPair(newIKP); err != nil {
		t.Fatal(err)
	}

	got, err := store.GetIdentityKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got.PublicKey, newIKP.PublicKey) {
		t.Error("identity key pair mismatch")
	}
}

func TestMemoryStoreDeviceID(t *testing.T) {
	store := NewMemoryStore(42)
	id, err := store.GetLocalDeviceID()
	if err != nil {
		t.Fatal(err)
	}
	if id != 42 {
		t.Errorf("device ID = %d, want 42", id)
	}
}

func TestMemoryStoreTOFU(t *testing.T) {
	store := NewMemoryStore(1)
	addr := Address{JID: "bob@example.com", DeviceID: 1}

	pub, _, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}

	// First use: should be trusted
	trusted, err := store.IsTrusted(addr, pub)
	if err != nil {
		t.Fatal(err)
	}
	if !trusted {
		t.Error("expected trust on first use")
	}

	// Save and verify same key is trusted
	if err := store.SaveRemoteIdentity(addr, pub); err != nil {
		t.Fatal(err)
	}
	trusted, err = store.IsTrusted(addr, pub)
	if err != nil {
		t.Fatal(err)
	}
	if !trusted {
		t.Error("expected same key to be trusted")
	}

	// Different key should not be trusted
	pub2, _, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	trusted, err = store.IsTrusted(addr, pub2)
	if err != nil {
		t.Fatal(err)
	}
	if trusted {
		t.Error("expected different key to be untrusted")
	}
}

func TestMemoryStorePreKeys(t *testing.T) {
	store := NewMemoryStore(1)

	// Non-existent key
	_, err := store.GetPreKey(1)
	if err != ErrNoPreKey {
		t.Errorf("expected ErrNoPreKey, got %v", err)
	}

	pk := &PreKeyRecord{ID: 1, PrivateKey: make([]byte, 32), PublicKey: make([]byte, 32)}
	if err := store.SavePreKey(pk); err != nil {
		t.Fatal(err)
	}

	got, err := store.GetPreKey(1)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != 1 {
		t.Errorf("pre-key ID = %d, want 1", got.ID)
	}

	if err := store.RemovePreKey(1); err != nil {
		t.Fatal(err)
	}
	_, err = store.GetPreKey(1)
	if err != ErrNoPreKey {
		t.Errorf("expected ErrNoPreKey after remove, got %v", err)
	}
}

func TestMemoryStoreSessions(t *testing.T) {
	store := NewMemoryStore(1)
	addr := Address{JID: "bob@example.com", DeviceID: 1}

	exists, err := store.ContainsSession(addr)
	if err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Error("expected no session initially")
	}

	_, err = store.GetSession(addr)
	if err != ErrNoSession {
		t.Errorf("expected ErrNoSession, got %v", err)
	}

	data := []byte("session data")
	if err := store.SaveSession(addr, data); err != nil {
		t.Fatal(err)
	}

	exists, err = store.ContainsSession(addr)
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Error("expected session to exist")
	}

	got, err := store.GetSession(addr)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, data) {
		t.Error("session data mismatch")
	}

	// Verify it's a copy
	got[0] = 0xFF
	got2, _ := store.GetSession(addr)
	if got2[0] == 0xFF {
		t.Error("session data should be independent copies")
	}
}
