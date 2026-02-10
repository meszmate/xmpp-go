package sasl

import (
	"bytes"
	"testing"
)

func TestPlainName(t *testing.T) {
	t.Parallel()
	p := NewPlain(Credentials{Username: "user", Password: "pass"})
	if p.Name() != "PLAIN" {
		t.Errorf("Name() = %q, want %q", p.Name(), "PLAIN")
	}
}

func TestPlainStart(t *testing.T) {
	t.Parallel()
	p := NewPlain(Credentials{Username: "user", Password: "pass"})
	resp, err := p.Start()
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	// Format: [authzid]\0authcid\0passwd
	expected := []byte("\x00user\x00pass")
	if !bytes.Equal(resp, expected) {
		t.Errorf("Start() = %q, want %q", resp, expected)
	}
}

func TestPlainStartWithAuthzID(t *testing.T) {
	t.Parallel()
	p := NewPlain(Credentials{Username: "user", Password: "pass", AuthzID: "admin"})
	resp, err := p.Start()
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	expected := []byte("admin\x00user\x00pass")
	if !bytes.Equal(resp, expected) {
		t.Errorf("Start() = %q, want %q", resp, expected)
	}
}

func TestPlainCompleted(t *testing.T) {
	t.Parallel()
	p := NewPlain(Credentials{Username: "user", Password: "pass"})
	if p.Completed() {
		t.Error("should not be completed before Start")
	}
	p.Start()
	if !p.Completed() {
		t.Error("should be completed after Start")
	}
}
