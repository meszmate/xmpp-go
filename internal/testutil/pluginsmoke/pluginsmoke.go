package pluginsmoke

import (
	"context"
	"testing"

	"github.com/meszmate/xmpp-go/plugin"
	"github.com/meszmate/xmpp-go/storage/memory"
)

// Run verifies the basic plugin contract and lifecycle hooks.
func Run(t *testing.T, p plugin.Plugin) {
	t.Helper()

	if p == nil {
		t.Fatal("plugin is nil")
	}
	if p.Name() == "" {
		t.Fatal("Name() returned empty string")
	}
	if p.Version() == "" {
		t.Fatal("Version() returned empty string")
	}
	for i, dep := range p.Dependencies() {
		if dep == "" {
			t.Fatalf("Dependencies()[%d] returned empty string", i)
		}
	}

	params := plugin.InitParams{
		SendRaw: func(context.Context, []byte) error { return nil },
		SendElement: func(context.Context, any) error {
			return nil
		},
		State:     func() uint32 { return 0 },
		LocalJID:  func() string { return "alice@example.com" },
		RemoteJID: func() string { return "bob@example.com" },
		Get:       func(string) (plugin.Plugin, bool) { return nil, false },
		Storage:   memory.New(),
	}

	if err := p.Initialize(context.Background(), params); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	if err := p.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}
