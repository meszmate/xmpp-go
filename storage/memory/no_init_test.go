package memory_test

import (
	"context"
	"testing"
	"time"

	"github.com/meszmate/xmpp-go/storage"
	"github.com/meszmate/xmpp-go/storage/memory"
)

func TestMemoryStorageAllStoresWithoutInit(t *testing.T) {
	ctx := context.Background()
	s := memory.New()

	us := s.UserStore()
	if err := us.CreateUser(ctx, &storage.User{Username: "alice", Password: "secret"}); err != nil {
		t.Fatalf("UserStore.CreateUser without Init: %v", err)
	}

	rs := s.RosterStore()
	if err := rs.UpsertRosterItem(ctx, &storage.RosterItem{
		UserJID:      "alice@example.com",
		ContactJID:   "bob@example.com",
		Name:         "Bob",
		Subscription: "both",
		Groups:       []string{"friends"},
	}); err != nil {
		t.Fatalf("RosterStore.UpsertRosterItem without Init: %v", err)
	}

	bs := s.BlockingStore()
	if err := bs.BlockJID(ctx, "alice@example.com", "spam@example.com"); err != nil {
		t.Fatalf("BlockingStore.BlockJID without Init: %v", err)
	}

	vs := s.VCardStore()
	if err := vs.SetVCard(ctx, "alice@example.com", []byte("<vCard/>")); err != nil {
		t.Fatalf("VCardStore.SetVCard without Init: %v", err)
	}

	os := s.OfflineStore()
	if err := os.StoreOfflineMessage(ctx, &storage.OfflineMessage{
		ID:        "m1",
		UserJID:   "alice@example.com",
		FromJID:   "bob@example.com",
		Data:      []byte("<message/>"),
		CreatedAt: time.Now(),
	}); err != nil {
		t.Fatalf("OfflineStore.StoreOfflineMessage without Init: %v", err)
	}

	ms := s.MAMStore()
	if err := ms.ArchiveMessage(ctx, &storage.ArchivedMessage{
		ID:        "a1",
		UserJID:   "alice@example.com",
		WithJID:   "bob@example.com",
		FromJID:   "bob@example.com",
		Data:      []byte("<message/>"),
		CreatedAt: time.Now(),
	}); err != nil {
		t.Fatalf("MAMStore.ArchiveMessage without Init: %v", err)
	}

	mucs := s.MUCRoomStore()
	if err := mucs.CreateRoom(ctx, &storage.MUCRoom{RoomJID: "room@example.com", Name: "Room"}); err != nil {
		t.Fatalf("MUCRoomStore.CreateRoom without Init: %v", err)
	}
	if err := mucs.SetAffiliation(ctx, &storage.MUCAffiliation{
		RoomJID:     "room@example.com",
		UserJID:     "alice@example.com",
		Affiliation: "member",
	}); err != nil {
		t.Fatalf("MUCRoomStore.SetAffiliation without Init: %v", err)
	}

	ps := s.PubSubStore()
	if err := ps.CreateNode(ctx, &storage.PubSubNode{Host: "example.com", NodeID: "node1"}); err != nil {
		t.Fatalf("PubSubStore.CreateNode without Init: %v", err)
	}
	if err := ps.UpsertItem(ctx, &storage.PubSubItem{
		Host:      "example.com",
		NodeID:    "node1",
		ItemID:    "item1",
		Publisher: "alice@example.com",
		Payload:   []byte("<entry/>"),
		CreatedAt: time.Now(),
	}); err != nil {
		t.Fatalf("PubSubStore.UpsertItem without Init: %v", err)
	}
	if err := ps.Subscribe(ctx, &storage.PubSubSubscription{
		Host:   "example.com",
		NodeID: "node1",
		JID:    "alice@example.com",
	}); err != nil {
		t.Fatalf("PubSubStore.Subscribe without Init: %v", err)
	}

	bms := s.BookmarkStore()
	if err := bms.SetBookmark(ctx, &storage.Bookmark{
		UserJID: "alice@example.com",
		RoomJID: "room@example.com",
		Name:    "Room",
	}); err != nil {
		t.Fatalf("BookmarkStore.SetBookmark without Init: %v", err)
	}
}
