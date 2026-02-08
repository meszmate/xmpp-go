package omemo

import (
	"bytes"
	"crypto/rand"
	"testing"
)

func TestRatchetHeaderMarshalRoundtrip(t *testing.T) {
	pub := make([]byte, 32)
	if _, err := rand.Read(pub); err != nil {
		t.Fatal(err)
	}
	h := &RatchetHeader{
		DHPub: pub,
		N:     42,
		PN:    10,
	}

	data, err := h.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}

	var h2 RatchetHeader
	if err := h2.UnmarshalBinary(data); err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(h.DHPub, h2.DHPub) {
		t.Error("DHPub mismatch")
	}
	if h.N != h2.N {
		t.Errorf("N = %d, want %d", h2.N, h.N)
	}
	if h.PN != h2.PN {
		t.Errorf("PN = %d, want %d", h2.PN, h.PN)
	}
}

func TestRatchetHeaderInvalidSize(t *testing.T) {
	var h RatchetHeader
	if err := h.UnmarshalBinary([]byte{1, 2, 3}); err == nil {
		t.Error("expected error for invalid size")
	}
}

func setupAliceBobRatchets(t *testing.T) (*RatchetState, *RatchetState) {
	t.Helper()

	// Generate shared secret (simulating X3DH output)
	sharedSecret := make([]byte, 32)
	if _, err := rand.Read(sharedSecret); err != nil {
		t.Fatal(err)
	}

	// Bob's SPK
	bobSPK, err := GenerateX25519KeyPair()
	if err != nil {
		t.Fatal(err)
	}

	alice, err := InitRatchetAsAlice(sharedSecret, bobSPK.PublicKey().Bytes())
	if err != nil {
		t.Fatal(err)
	}

	bob := InitRatchetAsBob(sharedSecret, bobSPK)

	return alice, bob
}

func TestRatchetBasicExchange(t *testing.T) {
	alice, bob := setupAliceBobRatchets(t)

	// Alice sends to Bob
	plaintext := []byte("Hello Bob!")
	header, ct, err := alice.RatchetEncrypt(plaintext)
	if err != nil {
		t.Fatal(err)
	}

	decrypted, err := bob.RatchetDecrypt(header, ct)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(plaintext, decrypted) {
		t.Errorf("decrypted = %q, want %q", decrypted, plaintext)
	}
}

func TestRatchetBidirectional(t *testing.T) {
	alice, bob := setupAliceBobRatchets(t)

	messages := []struct {
		from    string
		content string
	}{
		{"alice", "Hello Bob!"},
		{"bob", "Hi Alice!"},
		{"alice", "How are you?"},
		{"bob", "Great, thanks!"},
		{"alice", "Message 5"},
		{"alice", "Message 6"},
		{"bob", "Message 7"},
		{"bob", "Message 8"},
		{"alice", "Message 9"},
	}

	for _, msg := range messages {
		plaintext := []byte(msg.content)
		var sender, receiver *RatchetState
		if msg.from == "alice" {
			sender, receiver = alice, bob
		} else {
			sender, receiver = bob, alice
		}

		header, ct, err := sender.RatchetEncrypt(plaintext)
		if err != nil {
			t.Fatalf("encrypt %q: %v", msg.content, err)
		}

		decrypted, err := receiver.RatchetDecrypt(header, ct)
		if err != nil {
			t.Fatalf("decrypt %q: %v", msg.content, err)
		}

		if !bytes.Equal(plaintext, decrypted) {
			t.Errorf("decrypted = %q, want %q", decrypted, plaintext)
		}
	}
}

func TestRatchetOutOfOrder(t *testing.T) {
	alice, bob := setupAliceBobRatchets(t)

	// Alice sends 3 messages
	var headers [3]*RatchetHeader
	var cts [3][]byte
	for i := range 3 {
		h, ct, err := alice.RatchetEncrypt([]byte("message " + string(rune('A'+i))))
		if err != nil {
			t.Fatal(err)
		}
		headers[i] = h
		cts[i] = ct
	}

	// Bob decrypts in reverse order
	for i := 2; i >= 0; i-- {
		decrypted, err := bob.RatchetDecrypt(headers[i], cts[i])
		if err != nil {
			t.Fatalf("decrypt message %d: %v", i, err)
		}
		expected := "message " + string(rune('A'+i))
		if string(decrypted) != expected {
			t.Errorf("message %d: got %q, want %q", i, decrypted, expected)
		}
	}
}

func TestRatchetStateSerialization(t *testing.T) {
	alice, bob := setupAliceBobRatchets(t)

	// Exchange a message to establish chains
	h, ct, err := alice.RatchetEncrypt([]byte("test"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := bob.RatchetDecrypt(h, ct); err != nil {
		t.Fatal(err)
	}

	// Serialize and deserialize Alice's state
	data, err := alice.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}

	var restored RatchetState
	if err := restored.UnmarshalBinary(data); err != nil {
		t.Fatal(err)
	}

	// Send a message with restored state
	h2, ct2, err := restored.RatchetEncrypt([]byte("after restore"))
	if err != nil {
		t.Fatal(err)
	}

	decrypted, err := bob.RatchetDecrypt(h2, ct2)
	if err != nil {
		t.Fatal(err)
	}

	if string(decrypted) != "after restore" {
		t.Errorf("decrypted = %q, want %q", decrypted, "after restore")
	}
}

func TestRatchetSkippedKeyLimit(t *testing.T) {
	alice, bob := setupAliceBobRatchets(t)

	// Alice sends first message so Bob has a receiving chain
	h, ct, err := alice.RatchetEncrypt([]byte("init"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := bob.RatchetDecrypt(h, ct); err != nil {
		t.Fatal(err)
	}

	// Simulate a header claiming message number way beyond limit
	fakeHeader := &RatchetHeader{
		DHPub: alice.DHs.PublicKey().Bytes(),
		N:     maxSkippedKeys + bob.Nr + 1,
		PN:    0,
	}
	err = bob.skipMessageKeys(fakeHeader.N)
	if err != ErrSkippedKeyLimit {
		t.Errorf("expected ErrSkippedKeyLimit, got %v", err)
	}
}

func TestRatchetStateMarshalWithSkippedKeys(t *testing.T) {
	alice, bob := setupAliceBobRatchets(t)

	// Alice sends 3 messages
	var headers [3]*RatchetHeader
	var cts [3][]byte
	for i := range 3 {
		h, ct, err := alice.RatchetEncrypt([]byte("msg"))
		if err != nil {
			t.Fatal(err)
		}
		headers[i] = h
		cts[i] = ct
	}

	// Bob only decrypts the 3rd, skipping 0 and 1
	if _, err := bob.RatchetDecrypt(headers[2], cts[2]); err != nil {
		t.Fatal(err)
	}

	if len(bob.MKSkipped) != 2 {
		t.Fatalf("expected 2 skipped keys, got %d", len(bob.MKSkipped))
	}

	// Serialize with skipped keys
	data, err := bob.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}

	var restored RatchetState
	if err := restored.UnmarshalBinary(data); err != nil {
		t.Fatal(err)
	}

	if len(restored.MKSkipped) != 2 {
		t.Fatalf("restored: expected 2 skipped keys, got %d", len(restored.MKSkipped))
	}

	// Now decrypt the skipped messages with restored state
	for i := range 2 {
		decrypted, err := restored.RatchetDecrypt(headers[i], cts[i])
		if err != nil {
			t.Fatalf("decrypt skipped message %d: %v", i, err)
		}
		if string(decrypted) != "msg" {
			t.Errorf("message %d: got %q, want %q", i, decrypted, "msg")
		}
	}
}

func TestRatchetDuplicateMessage(t *testing.T) {
	alice, bob := setupAliceBobRatchets(t)

	h, ct, err := alice.RatchetEncrypt([]byte("one-time"))
	if err != nil {
		t.Fatal(err)
	}

	// First decrypt succeeds
	if _, err := bob.RatchetDecrypt(h, ct); err != nil {
		t.Fatal(err)
	}

	// Second decrypt should fail (key consumed)
	_, err = bob.RatchetDecrypt(h, ct)
	if err == nil {
		t.Error("expected error for duplicate message")
	}
}

