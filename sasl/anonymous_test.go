package sasl

import (
	"testing"
)

func TestAnonymousName(t *testing.T) {
	t.Parallel()
	a := NewAnonymous("trace123")
	if a.Name() != "ANONYMOUS" {
		t.Errorf("Name() = %q, want %q", a.Name(), "ANONYMOUS")
	}
}

func TestAnonymousStart(t *testing.T) {
	t.Parallel()
	a := NewAnonymous("trace-info")
	resp, err := a.Start()
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if string(resp) != "trace-info" {
		t.Errorf("Start() = %q, want %q", string(resp), "trace-info")
	}
}

func TestAnonymousStartEmpty(t *testing.T) {
	t.Parallel()
	a := NewAnonymous("")
	resp, err := a.Start()
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if string(resp) != "" {
		t.Errorf("Start() = %q, want empty", string(resp))
	}
}

func TestAnonymousCompleted(t *testing.T) {
	t.Parallel()
	a := NewAnonymous("trace")
	if a.Completed() {
		t.Error("should not be completed before Start")
	}
	a.Start()
	if !a.Completed() {
		t.Error("should be completed after Start")
	}
}
