package omemo

import (
	"bytes"
	"testing"
)

// TestFullConversation tests a complete Alice<->Bob OMEMO conversation
// including session setup, bidirectional messages, and session persistence.
func TestFullConversation(t *testing.T) {
	// Setup Alice
	aliceStore := NewMemoryStore(1)
	aliceManager := NewManager(aliceStore)
	aliceBundle, err := aliceManager.GenerateBundle(5)
	if err != nil {
		t.Fatal("alice generate bundle:", err)
	}
	aliceAddr := Address{JID: "alice@example.com", DeviceID: 1}

	// Setup Bob
	bobStore := NewMemoryStore(2)
	bobManager := NewManager(bobStore)
	bobBundle, err := bobManager.GenerateBundle(5)
	if err != nil {
		t.Fatal("bob generate bundle:", err)
	}
	bobAddr := Address{JID: "bob@example.com", DeviceID: 2}

	// Exchange bundles
	aliceManager.ProcessBundle(bobAddr, bobBundle)
	bobManager.ProcessBundle(aliceAddr, aliceBundle)

	// Alice sends first message to Bob
	msg1, err := aliceManager.Encrypt([]byte("Hello Bob!"), bobAddr)
	if err != nil {
		t.Fatal("alice encrypt:", err)
	}

	if msg1.SenderDeviceID != 1 {
		t.Errorf("sender device ID = %d, want 1", msg1.SenderDeviceID)
	}
	if len(msg1.Keys) != 1 {
		t.Fatalf("expected 1 key, got %d", len(msg1.Keys))
	}
	if msg1.Keys[0].DeviceID != 2 {
		t.Errorf("key device ID = %d, want 2", msg1.Keys[0].DeviceID)
	}
	if !msg1.Keys[0].IsPreKey {
		t.Error("first message should be a pre-key message")
	}

	// Bob decrypts using the pre-key message flow
	// Extract the pending pre-key info from Alice's session
	aliceSession := aliceManager.sessions[bobAddr]
	if aliceSession == nil {
		t.Fatal("alice should have a session for bob")
	}

	plaintext1, err := bobManager.DecryptPreKeyMessage(
		aliceAddr,
		aliceBundle.IdentityKey,
		aliceSession.PendingPreKey.EphemeralPubKey,
		aliceSession.PendingPreKey.PreKeyID,
		aliceSession.PendingPreKey.SignedPreKeyID,
		msg1,
	)
	if err != nil {
		t.Fatal("bob decrypt pre-key message:", err)
	}

	if string(plaintext1) != "Hello Bob!" {
		t.Errorf("decrypted = %q, want %q", plaintext1, "Hello Bob!")
	}

	// Bob replies to Alice
	msg2, err := bobManager.Encrypt([]byte("Hi Alice!"), aliceAddr)
	if err != nil {
		t.Fatal("bob encrypt:", err)
	}

	// For Bob's reply, Alice needs to process it as a pre-key message too
	// since Alice doesn't have a session from Bob's side yet
	bobSession := bobManager.sessions[aliceAddr]
	if bobSession == nil {
		t.Fatal("bob should have a session for alice")
	}

	// Check if Bob's message is a pre-key message
	if msg2.Keys[0].IsPreKey {
		// Bob should NOT have a pending pre-key since he was the responder
		t.Error("Bob's response should not be a pre-key message")
	}

	// Alice decrypts Bob's reply using the existing session
	plaintext2, err := aliceManager.Decrypt(bobAddr, msg2)
	if err != nil {
		t.Fatal("alice decrypt:", err)
	}

	if string(plaintext2) != "Hi Alice!" {
		t.Errorf("decrypted = %q, want %q", plaintext2, "Hi Alice!")
	}

	// Continue conversation
	messages := []struct {
		from    string
		to      Address
		content string
	}{
		{"alice", bobAddr, "How are you?"},
		{"bob", aliceAddr, "I'm great!"},
		{"alice", bobAddr, "Wonderful"},
		{"alice", bobAddr, "Another message"},
		{"bob", aliceAddr, "Got them all"},
	}

	for _, m := range messages {
		var sender, receiver *Manager
		var senderAddr Address
		if m.from == "alice" {
			sender, receiver = aliceManager, bobManager
			senderAddr = aliceAddr
		} else {
			sender, receiver = bobManager, aliceManager
			senderAddr = bobAddr
		}

		encrypted, err := sender.Encrypt([]byte(m.content), m.to)
		if err != nil {
			t.Fatalf("encrypt %q: %v", m.content, err)
		}

		decrypted, err := receiver.Decrypt(senderAddr, encrypted)
		if err != nil {
			t.Fatalf("decrypt %q: %v", m.content, err)
		}

		if string(decrypted) != m.content {
			t.Errorf("got %q, want %q", decrypted, m.content)
		}
	}
}

// TestSessionPersistence tests that sessions survive serialization/deserialization.
func TestSessionPersistence(t *testing.T) {
	// Setup Alice and Bob
	aliceStore := NewMemoryStore(1)
	aliceManager := NewManager(aliceStore)
	aliceBundle, err := aliceManager.GenerateBundle(5)
	if err != nil {
		t.Fatal(err)
	}
	aliceAddr := Address{JID: "alice@example.com", DeviceID: 1}

	bobStore := NewMemoryStore(2)
	bobManager := NewManager(bobStore)
	bobBundle, err := bobManager.GenerateBundle(5)
	if err != nil {
		t.Fatal(err)
	}
	bobAddr := Address{JID: "bob@example.com", DeviceID: 2}

	// Exchange bundles and send first message
	aliceManager.ProcessBundle(bobAddr, bobBundle)
	bobManager.ProcessBundle(aliceAddr, aliceBundle)

	msg, err := aliceManager.Encrypt([]byte("persist test"), bobAddr)
	if err != nil {
		t.Fatal(err)
	}

	aliceSession := aliceManager.sessions[bobAddr]
	plaintext, err := bobManager.DecryptPreKeyMessage(
		aliceAddr, aliceBundle.IdentityKey,
		aliceSession.PendingPreKey.EphemeralPubKey,
		aliceSession.PendingPreKey.PreKeyID,
		aliceSession.PendingPreKey.SignedPreKeyID,
		msg,
	)
	if err != nil {
		t.Fatal(err)
	}
	if string(plaintext) != "persist test" {
		t.Fatalf("got %q, want %q", plaintext, "persist test")
	}

	// Create new managers with the same stores (simulating restart)
	aliceManager2 := NewManager(aliceStore)
	bobManager2 := NewManager(bobStore)

	// Bob sends with new manager (should load session from store)
	msg2, err := bobManager2.Encrypt([]byte("after restart"), aliceAddr)
	if err != nil {
		t.Fatal("bob encrypt after restart:", err)
	}

	decrypted, err := aliceManager2.Decrypt(bobAddr, msg2)
	if err != nil {
		t.Fatal("alice decrypt after restart:", err)
	}

	if string(decrypted) != "after restart" {
		t.Errorf("got %q, want %q", decrypted, "after restart")
	}
}

// TestMultiDevice tests encryption to multiple devices.
func TestMultiDevice(t *testing.T) {
	// Alice
	aliceStore := NewMemoryStore(1)
	aliceManager := NewManager(aliceStore)
	aliceBundle, err := aliceManager.GenerateBundle(5)
	if err != nil {
		t.Fatal(err)
	}

	// Bob device 1
	bobStore1 := NewMemoryStore(2)
	bobManager1 := NewManager(bobStore1)
	bobBundle1, err := bobManager1.GenerateBundle(5)
	if err != nil {
		t.Fatal(err)
	}
	bobAddr1 := Address{JID: "bob@example.com", DeviceID: 2}

	// Bob device 2
	bobStore2 := NewMemoryStore(3)
	bobManager2 := NewManager(bobStore2)
	bobBundle2, err := bobManager2.GenerateBundle(5)
	if err != nil {
		t.Fatal(err)
	}
	bobAddr2 := Address{JID: "bob@example.com", DeviceID: 3}

	// Alice processes both of Bob's bundles
	aliceManager.ProcessBundle(bobAddr1, bobBundle1)
	aliceManager.ProcessBundle(bobAddr2, bobBundle2)

	// Alice encrypts for both devices
	msg, err := aliceManager.Encrypt([]byte("multi-device"), bobAddr1, bobAddr2)
	if err != nil {
		t.Fatal(err)
	}

	if len(msg.Keys) != 2 {
		t.Fatalf("expected 2 keys, got %d", len(msg.Keys))
	}

	// Both Bob devices should be able to decrypt
	aliceAddr := Address{JID: "alice@example.com", DeviceID: 1}

	aliceSession := aliceManager.sessions[bobAddr1]
	bobManager1.ProcessBundle(aliceAddr, aliceBundle)
	pt1, err := bobManager1.DecryptPreKeyMessage(
		aliceAddr, aliceBundle.IdentityKey,
		aliceSession.PendingPreKey.EphemeralPubKey,
		aliceSession.PendingPreKey.PreKeyID,
		aliceSession.PendingPreKey.SignedPreKeyID,
		msg,
	)
	if err != nil {
		t.Fatal("bob device 1 decrypt:", err)
	}
	if string(pt1) != "multi-device" {
		t.Errorf("device 1: got %q, want %q", pt1, "multi-device")
	}

	aliceSession2 := aliceManager.sessions[bobAddr2]
	bobManager2.ProcessBundle(aliceAddr, aliceBundle)
	pt2, err := bobManager2.DecryptPreKeyMessage(
		aliceAddr, aliceBundle.IdentityKey,
		aliceSession2.PendingPreKey.EphemeralPubKey,
		aliceSession2.PendingPreKey.PreKeyID,
		aliceSession2.PendingPreKey.SignedPreKeyID,
		msg,
	)
	if err != nil {
		t.Fatal("bob device 2 decrypt:", err)
	}
	if string(pt2) != "multi-device" {
		t.Errorf("device 2: got %q, want %q", pt2, "multi-device")
	}
}

// TestSessionSerialization tests session marshal/unmarshal roundtrip.
func TestSessionSerialization(t *testing.T) {
	aliceStore := NewMemoryStore(1)
	aliceManager := NewManager(aliceStore)
	aliceBundle, err := aliceManager.GenerateBundle(5)
	if err != nil {
		t.Fatal(err)
	}

	bobStore := NewMemoryStore(2)
	bobBundle, err := GenerateBundle(bobStore, 5)
	if err != nil {
		t.Fatal(err)
	}
	bobAddr := Address{JID: "bob@example.com", DeviceID: 2}

	aliceManager.ProcessBundle(bobAddr, bobBundle)

	// Create session by encrypting
	_, err = aliceManager.Encrypt([]byte("test"), bobAddr)
	if err != nil {
		t.Fatal(err)
	}

	session := aliceManager.sessions[bobAddr]
	if session == nil {
		t.Fatal("expected session")
	}

	data, err := session.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}

	var restored Session
	if err := restored.UnmarshalBinary(data); err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(session.RemoteIdentity, restored.RemoteIdentity) {
		t.Error("remote identity mismatch")
	}

	// Both should have PendingPreKey since this was the first message
	if session.PendingPreKey == nil || restored.PendingPreKey == nil {
		t.Fatal("expected pending pre-key")
	}

	_ = aliceBundle // used for setup
}

// TestMalformedInputs tests handling of various malformed inputs.
func TestMalformedInputs(t *testing.T) {
	t.Run("empty key data", func(t *testing.T) {
		store := NewMemoryStore(1)
		manager := NewManager(store)
		if _, err := manager.GenerateBundle(1); err != nil {
			t.Fatal(err)
		}

		msg := &EncryptedMessage{
			SenderDeviceID: 99,
			Keys: []MessageKey{
				{DeviceID: 1, Data: []byte{}, IsPreKey: false},
			},
			IV:      make([]byte, 12),
			Payload: []byte("test"),
		}

		_, err := manager.Decrypt(Address{JID: "x@y", DeviceID: 99}, msg)
		if err == nil {
			t.Error("expected error for empty key data")
		}
	})

	t.Run("wrong device id", func(t *testing.T) {
		store := NewMemoryStore(1)
		manager := NewManager(store)

		msg := &EncryptedMessage{
			SenderDeviceID: 99,
			Keys: []MessageKey{
				{DeviceID: 999, Data: []byte("test"), IsPreKey: false},
			},
			IV:      make([]byte, 12),
			Payload: []byte("test"),
		}

		_, err := manager.Decrypt(Address{JID: "x@y", DeviceID: 99}, msg)
		if err == nil {
			t.Error("expected error for wrong device id")
		}
	})

	t.Run("corrupted session data", func(t *testing.T) {
		var s Session
		err := s.UnmarshalBinary([]byte{1, 2, 3})
		if err == nil {
			t.Error("expected error for corrupted session data")
		}
	})

	t.Run("corrupted ratchet header", func(t *testing.T) {
		var h RatchetHeader
		err := h.UnmarshalBinary([]byte{1, 2, 3})
		if err == nil {
			t.Error("expected error for corrupted ratchet header")
		}
	})
}

// TestBundleGeneration tests bundle generation and key properties.
func TestBundleGeneration(t *testing.T) {
	store := NewMemoryStore(1)
	bundle, err := GenerateBundle(store, 10)
	if err != nil {
		t.Fatal(err)
	}

	if len(bundle.IdentityKey) != 32 {
		t.Errorf("identity key length = %d, want 32", len(bundle.IdentityKey))
	}
	if len(bundle.SignedPreKey) != 32 {
		t.Errorf("signed pre-key length = %d, want 32", len(bundle.SignedPreKey))
	}
	if len(bundle.SignedPreKeySignature) != 64 {
		t.Errorf("signature length = %d, want 64", len(bundle.SignedPreKeySignature))
	}
	if len(bundle.PreKeys) != 10 {
		t.Errorf("pre-keys count = %d, want 10", len(bundle.PreKeys))
	}

	// All pre-key IDs should be unique
	ids := make(map[uint32]bool)
	for _, pk := range bundle.PreKeys {
		if ids[pk.ID] {
			t.Errorf("duplicate pre-key ID: %d", pk.ID)
		}
		ids[pk.ID] = true

		if len(pk.PublicKey) != 32 {
			t.Errorf("pre-key %d public key length = %d, want 32", pk.ID, len(pk.PublicKey))
		}
	}

	// Identity key should be saved in store
	ikp, err := store.GetIdentityKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	if ikp == nil {
		t.Fatal("expected identity key pair in store")
	}
	if !bytes.Equal(ikp.PublicKey, bundle.IdentityKey) {
		t.Error("stored identity key doesn't match bundle")
	}

	// Generating a second bundle should reuse the same identity key
	bundle2, err := GenerateBundle(store, 5)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(bundle.IdentityKey, bundle2.IdentityKey) {
		t.Error("second bundle should reuse same identity key")
	}
}
