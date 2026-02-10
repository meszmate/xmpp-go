package plugin

import (
	"context"
	"errors"
	"testing"
)

type mockPlugin struct {
	name    string
	version string
	deps    []string
	initLog *[]string
	closeLog *[]string
	initErr error
	closeErr error
}

func newMockPlugin(name string, deps []string, initLog, closeLog *[]string) *mockPlugin {
	return &mockPlugin{
		name:     name,
		version:  "1.0",
		deps:     deps,
		initLog:  initLog,
		closeLog: closeLog,
	}
}

func (m *mockPlugin) Name() string    { return m.name }
func (m *mockPlugin) Version() string { return m.version }
func (m *mockPlugin) Dependencies() []string { return m.deps }

func (m *mockPlugin) Initialize(_ context.Context, _ InitParams) error {
	if m.initLog != nil {
		*m.initLog = append(*m.initLog, m.name)
	}
	return m.initErr
}

func (m *mockPlugin) Close() error {
	if m.closeLog != nil {
		*m.closeLog = append(*m.closeLog, m.name)
	}
	return m.closeErr
}

func TestManagerRegisterGet(t *testing.T) {
	t.Parallel()
	mgr := NewManager()
	p := newMockPlugin("test", nil, nil, nil)

	if err := mgr.Register(p); err != nil {
		t.Fatalf("Register: %v", err)
	}

	got, ok := mgr.Get("test")
	if !ok {
		t.Fatal("Get returned false")
	}
	if got.Name() != "test" {
		t.Errorf("Name() = %q", got.Name())
	}
}

func TestManagerDuplicateError(t *testing.T) {
	t.Parallel()
	mgr := NewManager()
	p1 := newMockPlugin("dup", nil, nil, nil)
	p2 := newMockPlugin("dup", nil, nil, nil)

	if err := mgr.Register(p1); err != nil {
		t.Fatalf("first Register: %v", err)
	}
	err := mgr.Register(p2)
	if err == nil {
		t.Fatal("expected error for duplicate plugin")
	}
	if !errors.Is(err, ErrDuplicatePlugin) {
		t.Errorf("error = %v, want ErrDuplicatePlugin", err)
	}
}

func TestManagerGetNotFound(t *testing.T) {
	t.Parallel()
	mgr := NewManager()
	p, ok := mgr.Get("nonexistent")
	if ok {
		t.Error("Get should return false for missing plugin")
	}
	if p != nil {
		t.Error("Get should return nil for missing plugin")
	}
}

func TestManagerInitOrder(t *testing.T) {
	t.Parallel()
	var initLog []string
	mgr := NewManager()

	// B has no deps, A depends on B
	a := newMockPlugin("A", []string{"B"}, &initLog, nil)
	b := newMockPlugin("B", nil, &initLog, nil)

	mgr.Register(a)
	mgr.Register(b)

	if err := mgr.Initialize(context.Background(), InitParams{}); err != nil {
		t.Fatalf("Initialize: %v", err)
	}

	// B must be initialized before A
	if len(initLog) != 2 {
		t.Fatalf("initLog length = %d, want 2", len(initLog))
	}
	bIdx, aIdx := -1, -1
	for i, name := range initLog {
		if name == "B" {
			bIdx = i
		}
		if name == "A" {
			aIdx = i
		}
	}
	if bIdx >= aIdx {
		t.Errorf("B (idx=%d) should be initialized before A (idx=%d)", bIdx, aIdx)
	}
}

func TestManagerCyclicDep(t *testing.T) {
	t.Parallel()
	mgr := NewManager()

	a := newMockPlugin("A", []string{"B"}, nil, nil)
	b := newMockPlugin("B", []string{"A"}, nil, nil)

	mgr.Register(a)
	mgr.Register(b)

	err := mgr.Initialize(context.Background(), InitParams{})
	if err == nil {
		t.Fatal("expected error for cyclic dependency")
	}
	if !errors.Is(err, ErrCyclicDep) {
		t.Errorf("error = %v, want ErrCyclicDep", err)
	}
}

func TestManagerCloseOrder(t *testing.T) {
	t.Parallel()
	var initLog, closeLog []string
	mgr := NewManager()

	a := newMockPlugin("A", []string{"B"}, &initLog, &closeLog)
	b := newMockPlugin("B", nil, &initLog, &closeLog)

	mgr.Register(a)
	mgr.Register(b)

	if err := mgr.Initialize(context.Background(), InitParams{}); err != nil {
		t.Fatalf("Initialize: %v", err)
	}

	if err := mgr.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Close order should be reverse of init order
	if len(closeLog) != 2 {
		t.Fatalf("closeLog length = %d, want 2", len(closeLog))
	}
	// If init was [B, A], close should be [A, B]
	for i := range initLog {
		if initLog[i] != closeLog[len(closeLog)-1-i] {
			t.Errorf("close order %v is not reverse of init order %v", closeLog, initLog)
			break
		}
	}
}

func TestManagerMissingDep(t *testing.T) {
	t.Parallel()
	mgr := NewManager()

	a := newMockPlugin("A", []string{"Missing"}, nil, nil)
	mgr.Register(a)

	err := mgr.Initialize(context.Background(), InitParams{})
	if err == nil {
		t.Fatal("expected error for missing dependency")
	}
	if !errors.Is(err, ErrMissingDep) {
		t.Errorf("error = %v, want ErrMissingDep", err)
	}
}
