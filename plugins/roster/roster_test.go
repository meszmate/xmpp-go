package roster

import (
	"context"
	"testing"

	"github.com/meszmate/xmpp-go/plugin"
	"github.com/meszmate/xmpp-go/storage/memory"
)

func TestRosterPluginWithMemoryStoreWithoutInit(t *testing.T) {
	ctx := context.Background()
	p := New()

	if err := p.Initialize(ctx, plugin.InitParams{
		LocalJID: func() string { return "alice@example.com" },
		Storage:  memory.New(),
	}); err != nil {
		t.Fatalf("Initialize: %v", err)
	}

	if err := p.Set(ctx, Item{
		JID:          "bob@example.com",
		Name:         "Bob",
		Subscription: SubBoth,
	}); err != nil {
		t.Fatalf("Set: %v", err)
	}

	item, ok, err := p.Get(ctx, "bob@example.com")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !ok {
		t.Fatal("Get: expected item to exist")
	}
	if item.Name != "Bob" {
		t.Fatalf("Get: got name %q, want Bob", item.Name)
	}
}
