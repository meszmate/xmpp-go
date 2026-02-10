package xmpp

import (
	"crypto/sha1"
	"encoding/hex"
	"testing"
)

func TestComponentHandshakeHash(t *testing.T) {
	t.Parallel()
	c, err := NewComponent("component.example.com", "secret123")
	if err != nil {
		t.Fatalf("NewComponent: %v", err)
	}

	streamID := "abc123"
	got := c.Handshake(streamID)

	// Compute expected: SHA1(streamID + secret)
	h := sha1.New()
	h.Write([]byte(streamID + "secret123"))
	want := hex.EncodeToString(h.Sum(nil))

	if got != want {
		t.Errorf("Handshake(%q) = %q, want %q", streamID, got, want)
	}
}

func TestComponentDomain(t *testing.T) {
	t.Parallel()
	c, err := NewComponent("gateway.example.com", "secret")
	if err != nil {
		t.Fatalf("NewComponent: %v", err)
	}
	if c.Domain() != "gateway.example.com" {
		t.Errorf("Domain() = %q, want %q", c.Domain(), "gateway.example.com")
	}
}

func TestNewComponentDefaults(t *testing.T) {
	t.Parallel()
	c, err := NewComponent("test.example.com", "secret")
	if err != nil {
		t.Fatalf("NewComponent: %v", err)
	}
	if c.addr != "localhost:5275" {
		t.Errorf("default addr = %q, want %q", c.addr, "localhost:5275")
	}
}

func TestComponentWithAddr(t *testing.T) {
	t.Parallel()
	c, err := NewComponent("test.example.com", "secret",
		WithComponentAddr("xmpp.example.com:5275"),
	)
	if err != nil {
		t.Fatalf("NewComponent: %v", err)
	}
	if c.addr != "xmpp.example.com:5275" {
		t.Errorf("addr = %q, want %q", c.addr, "xmpp.example.com:5275")
	}
}

func TestComponentSessionNil(t *testing.T) {
	t.Parallel()
	c, _ := NewComponent("test.example.com", "secret")
	if c.Session() != nil {
		t.Error("Session() should be nil before Connect")
	}
}

func TestComponentCloseBeforeConnect(t *testing.T) {
	t.Parallel()
	c, _ := NewComponent("test.example.com", "secret")
	if err := c.Close(); err != nil {
		t.Errorf("Close before Connect should return nil, got %v", err)
	}
}
