package sasl

import (
	"testing"
)

func TestExternalName(t *testing.T) {
	t.Parallel()
	e := NewExternal("admin@example.com")
	if e.Name() != "EXTERNAL" {
		t.Errorf("Name() = %q, want %q", e.Name(), "EXTERNAL")
	}
}

func TestExternalStart(t *testing.T) {
	t.Parallel()
	e := NewExternal("admin@example.com")
	resp, err := e.Start()
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if string(resp) != "admin@example.com" {
		t.Errorf("Start() = %q, want %q", string(resp), "admin@example.com")
	}
}

func TestExternalStartEmpty(t *testing.T) {
	t.Parallel()
	e := NewExternal("")
	resp, err := e.Start()
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if len(resp) != 0 {
		t.Errorf("Start() = %q, want empty", string(resp))
	}
}

func TestExternalCompleted(t *testing.T) {
	t.Parallel()
	e := NewExternal("user")
	if e.Completed() {
		t.Error("should not be completed before Start")
	}
	e.Start()
	if !e.Completed() {
		t.Error("should be completed after Start")
	}
}
