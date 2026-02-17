package memory_test

import (
	"context"
	"testing"

	"github.com/meszmate/xmpp-go/storage"
	"github.com/meszmate/xmpp-go/storage/memory"
	"github.com/meszmate/xmpp-go/storage/storagetest"
)

func TestMemoryStorage(t *testing.T) {
	storagetest.TestStorage(t, func() storage.Storage {
		return memory.New()
	})
}

func TestMemoryStorageWithoutInit(t *testing.T) {
	ctx := context.Background()
	s := memory.New()
	rs := s.RosterStore()

	item := &storage.RosterItem{
		UserJID:      "alice@example.com",
		ContactJID:   "bob@example.com",
		Name:         "Bob",
		Subscription: "both",
		Groups:       []string{"friends"},
	}
	if err := rs.UpsertRosterItem(ctx, item); err != nil {
		t.Fatalf("UpsertRosterItem without Init: %v", err)
	}

	got, err := rs.GetRosterItem(ctx, "alice@example.com", "bob@example.com")
	if err != nil {
		t.Fatalf("GetRosterItem without Init: %v", err)
	}
	if got.Name != "Bob" {
		t.Fatalf("GetRosterItem without Init: got name %q, want Bob", got.Name)
	}
}
