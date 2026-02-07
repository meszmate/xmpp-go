// Package storagetest provides a conformance test suite for storage backends.
// Any backend can use TestStorage(t, newStore) to verify it implements the
// storage.Storage interface correctly.
package storagetest

import (
	"context"
	"testing"
	"time"

	"github.com/meszmate/xmpp-go/storage"
)

// TestStorage runs the full conformance test suite against a storage backend.
func TestStorage(t *testing.T, newStore func() storage.Storage) {
	t.Run("UserStore", func(t *testing.T) { testUserStore(t, newStore) })
	t.Run("RosterStore", func(t *testing.T) { testRosterStore(t, newStore) })
	t.Run("BlockingStore", func(t *testing.T) { testBlockingStore(t, newStore) })
	t.Run("VCardStore", func(t *testing.T) { testVCardStore(t, newStore) })
	t.Run("OfflineStore", func(t *testing.T) { testOfflineStore(t, newStore) })
	t.Run("MAMStore", func(t *testing.T) { testMAMStore(t, newStore) })
	t.Run("MUCRoomStore", func(t *testing.T) { testMUCRoomStore(t, newStore) })
	t.Run("PubSubStore", func(t *testing.T) { testPubSubStore(t, newStore) })
	t.Run("BookmarkStore", func(t *testing.T) { testBookmarkStore(t, newStore) })
}

func initStore(t *testing.T, newStore func() storage.Storage) storage.Storage {
	t.Helper()
	s := newStore()
	ctx := context.Background()
	if err := s.Init(ctx); err != nil {
		t.Fatalf("Init: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func testUserStore(t *testing.T, newStore func() storage.Storage) {
	s := initStore(t, newStore)
	us := s.UserStore()
	if us == nil {
		t.Skip("UserStore not supported")
	}
	ctx := context.Background()

	// Create
	user := &storage.User{Username: "alice", Password: "secret"}
	if err := us.CreateUser(ctx, user); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	// Duplicate
	if err := us.CreateUser(ctx, user); err != storage.ErrUserExists {
		t.Fatalf("CreateUser duplicate: got %v, want ErrUserExists", err)
	}

	// Get
	got, err := us.GetUser(ctx, "alice")
	if err != nil {
		t.Fatalf("GetUser: %v", err)
	}
	if got.Username != "alice" || got.Password != "secret" {
		t.Fatalf("GetUser: got %+v", got)
	}

	// Not found
	_, err = us.GetUser(ctx, "bob")
	if err != storage.ErrNotFound {
		t.Fatalf("GetUser not found: got %v, want ErrNotFound", err)
	}

	// Exists
	exists, err := us.UserExists(ctx, "alice")
	if err != nil || !exists {
		t.Fatalf("UserExists: %v, %v", exists, err)
	}
	exists, err = us.UserExists(ctx, "bob")
	if err != nil || exists {
		t.Fatalf("UserExists bob: %v, %v", exists, err)
	}

	// Authenticate
	ok, err := us.Authenticate(ctx, "alice", "secret")
	if err != nil || !ok {
		t.Fatalf("Authenticate: %v, %v", ok, err)
	}
	_, err = us.Authenticate(ctx, "alice", "wrong")
	if err != storage.ErrAuthFailed {
		t.Fatalf("Authenticate wrong: got %v, want ErrAuthFailed", err)
	}

	// Update
	user.Password = "newsecret"
	if err := us.UpdateUser(ctx, user); err != nil {
		t.Fatalf("UpdateUser: %v", err)
	}
	ok, err = us.Authenticate(ctx, "alice", "newsecret")
	if err != nil || !ok {
		t.Fatalf("Authenticate after update: %v, %v", ok, err)
	}

	// Delete
	if err := us.DeleteUser(ctx, "alice"); err != nil {
		t.Fatalf("DeleteUser: %v", err)
	}
	if err := us.DeleteUser(ctx, "alice"); err != storage.ErrNotFound {
		t.Fatalf("DeleteUser again: got %v, want ErrNotFound", err)
	}
}

func testRosterStore(t *testing.T, newStore func() storage.Storage) {
	s := initStore(t, newStore)
	rs := s.RosterStore()
	if rs == nil {
		t.Skip("RosterStore not supported")
	}
	ctx := context.Background()

	item := &storage.RosterItem{
		UserJID: "alice@example.com", ContactJID: "bob@example.com",
		Name: "Bob", Subscription: "both", Groups: []string{"friends"},
	}

	// Upsert
	if err := rs.UpsertRosterItem(ctx, item); err != nil {
		t.Fatalf("UpsertRosterItem: %v", err)
	}

	// Get
	got, err := rs.GetRosterItem(ctx, "alice@example.com", "bob@example.com")
	if err != nil {
		t.Fatalf("GetRosterItem: %v", err)
	}
	if got.Name != "Bob" || got.Subscription != "both" {
		t.Fatalf("GetRosterItem: got %+v", got)
	}

	// Not found
	_, err = rs.GetRosterItem(ctx, "alice@example.com", "charlie@example.com")
	if err != storage.ErrNotFound {
		t.Fatalf("GetRosterItem not found: got %v", err)
	}

	// List
	items, err := rs.GetRosterItems(ctx, "alice@example.com")
	if err != nil || len(items) != 1 {
		t.Fatalf("GetRosterItems: %d, %v", len(items), err)
	}

	// Version
	if err := rs.SetRosterVersion(ctx, "alice@example.com", "v1"); err != nil {
		t.Fatalf("SetRosterVersion: %v", err)
	}
	ver, err := rs.GetRosterVersion(ctx, "alice@example.com")
	if err != nil || ver != "v1" {
		t.Fatalf("GetRosterVersion: %q, %v", ver, err)
	}

	// Delete
	if err := rs.DeleteRosterItem(ctx, "alice@example.com", "bob@example.com"); err != nil {
		t.Fatalf("DeleteRosterItem: %v", err)
	}
	items, _ = rs.GetRosterItems(ctx, "alice@example.com")
	if len(items) != 0 {
		t.Fatalf("GetRosterItems after delete: %d", len(items))
	}
}

func testBlockingStore(t *testing.T, newStore func() storage.Storage) {
	s := initStore(t, newStore)
	bs := s.BlockingStore()
	if bs == nil {
		t.Skip("BlockingStore not supported")
	}
	ctx := context.Background()

	// Block
	if err := bs.BlockJID(ctx, "alice@example.com", "spam@example.com"); err != nil {
		t.Fatalf("BlockJID: %v", err)
	}

	// IsBlocked
	blocked, err := bs.IsBlocked(ctx, "alice@example.com", "spam@example.com")
	if err != nil || !blocked {
		t.Fatalf("IsBlocked: %v, %v", blocked, err)
	}
	blocked, _ = bs.IsBlocked(ctx, "alice@example.com", "friend@example.com")
	if blocked {
		t.Fatal("IsBlocked false positive")
	}

	// List
	jids, err := bs.GetBlockedJIDs(ctx, "alice@example.com")
	if err != nil || len(jids) != 1 {
		t.Fatalf("GetBlockedJIDs: %d, %v", len(jids), err)
	}

	// Unblock
	if err := bs.UnblockJID(ctx, "alice@example.com", "spam@example.com"); err != nil {
		t.Fatalf("UnblockJID: %v", err)
	}
	blocked, _ = bs.IsBlocked(ctx, "alice@example.com", "spam@example.com")
	if blocked {
		t.Fatal("IsBlocked after unblock")
	}
}

func testVCardStore(t *testing.T, newStore func() storage.Storage) {
	s := initStore(t, newStore)
	vs := s.VCardStore()
	if vs == nil {
		t.Skip("VCardStore not supported")
	}
	ctx := context.Background()

	data := []byte("<vCard><FN>Alice</FN></vCard>")

	// Set
	if err := vs.SetVCard(ctx, "alice@example.com", data); err != nil {
		t.Fatalf("SetVCard: %v", err)
	}

	// Get
	got, err := vs.GetVCard(ctx, "alice@example.com")
	if err != nil || string(got) != string(data) {
		t.Fatalf("GetVCard: %q, %v", got, err)
	}

	// Not found
	_, err = vs.GetVCard(ctx, "bob@example.com")
	if err != storage.ErrNotFound {
		t.Fatalf("GetVCard not found: got %v", err)
	}

	// Delete
	if err := vs.DeleteVCard(ctx, "alice@example.com"); err != nil {
		t.Fatalf("DeleteVCard: %v", err)
	}
	_, err = vs.GetVCard(ctx, "alice@example.com")
	if err != storage.ErrNotFound {
		t.Fatalf("GetVCard after delete: got %v", err)
	}
}

func testOfflineStore(t *testing.T, newStore func() storage.Storage) {
	s := initStore(t, newStore)
	os := s.OfflineStore()
	if os == nil {
		t.Skip("OfflineStore not supported")
	}
	ctx := context.Background()

	msg := &storage.OfflineMessage{
		ID: "msg1", UserJID: "alice@example.com",
		FromJID: "bob@example.com", Data: []byte("<message/>"),
		CreatedAt: time.Now(),
	}

	// Store
	if err := os.StoreOfflineMessage(ctx, msg); err != nil {
		t.Fatalf("StoreOfflineMessage: %v", err)
	}

	// Count
	count, err := os.CountOfflineMessages(ctx, "alice@example.com")
	if err != nil || count != 1 {
		t.Fatalf("CountOfflineMessages: %d, %v", count, err)
	}

	// Get
	msgs, err := os.GetOfflineMessages(ctx, "alice@example.com")
	if err != nil || len(msgs) != 1 {
		t.Fatalf("GetOfflineMessages: %d, %v", len(msgs), err)
	}

	// Delete
	if err := os.DeleteOfflineMessages(ctx, "alice@example.com"); err != nil {
		t.Fatalf("DeleteOfflineMessages: %v", err)
	}
	count, _ = os.CountOfflineMessages(ctx, "alice@example.com")
	if count != 0 {
		t.Fatalf("CountOfflineMessages after delete: %d", count)
	}
}

func testMAMStore(t *testing.T, newStore func() storage.Storage) {
	s := initStore(t, newStore)
	ms := s.MAMStore()
	if ms == nil {
		t.Skip("MAMStore not supported")
	}
	ctx := context.Background()

	now := time.Now()
	msg1 := &storage.ArchivedMessage{
		ID: "1", UserJID: "alice@example.com", WithJID: "bob@example.com",
		FromJID: "bob@example.com", Data: []byte("<msg1/>"), CreatedAt: now,
	}
	msg2 := &storage.ArchivedMessage{
		ID: "2", UserJID: "alice@example.com", WithJID: "charlie@example.com",
		FromJID: "charlie@example.com", Data: []byte("<msg2/>"), CreatedAt: now.Add(time.Second),
	}

	// Archive
	if err := ms.ArchiveMessage(ctx, msg1); err != nil {
		t.Fatalf("ArchiveMessage: %v", err)
	}
	if err := ms.ArchiveMessage(ctx, msg2); err != nil {
		t.Fatalf("ArchiveMessage: %v", err)
	}

	// Query all
	result, err := ms.QueryMessages(ctx, &storage.MAMQuery{UserJID: "alice@example.com"})
	if err != nil {
		t.Fatalf("QueryMessages: %v", err)
	}
	if len(result.Messages) != 2 {
		t.Fatalf("QueryMessages: got %d messages", len(result.Messages))
	}

	// Query with filter
	result, err = ms.QueryMessages(ctx, &storage.MAMQuery{
		UserJID: "alice@example.com", WithJID: "bob@example.com",
	})
	if err != nil || len(result.Messages) != 1 {
		t.Fatalf("QueryMessages with filter: %d, %v", len(result.Messages), err)
	}

	// Delete
	if err := ms.DeleteMessageArchive(ctx, "alice@example.com"); err != nil {
		t.Fatalf("DeleteMessageArchive: %v", err)
	}
	result, _ = ms.QueryMessages(ctx, &storage.MAMQuery{UserJID: "alice@example.com"})
	if len(result.Messages) != 0 {
		t.Fatalf("QueryMessages after delete: %d", len(result.Messages))
	}
}

func testMUCRoomStore(t *testing.T, newStore func() storage.Storage) {
	s := initStore(t, newStore)
	ms := s.MUCRoomStore()
	if ms == nil {
		t.Skip("MUCRoomStore not supported")
	}
	ctx := context.Background()

	room := &storage.MUCRoom{RoomJID: "room@conference.example.com", Name: "Test Room", Public: true}

	// Create
	if err := ms.CreateRoom(ctx, room); err != nil {
		t.Fatalf("CreateRoom: %v", err)
	}

	// Duplicate
	if err := ms.CreateRoom(ctx, room); err != storage.ErrUserExists {
		t.Fatalf("CreateRoom duplicate: got %v", err)
	}

	// Get
	got, err := ms.GetRoom(ctx, room.RoomJID)
	if err != nil || got.Name != "Test Room" {
		t.Fatalf("GetRoom: %+v, %v", got, err)
	}

	// Update
	room.Name = "Updated Room"
	if err := ms.UpdateRoom(ctx, room); err != nil {
		t.Fatalf("UpdateRoom: %v", err)
	}

	// List
	rooms, err := ms.ListRooms(ctx)
	if err != nil || len(rooms) != 1 {
		t.Fatalf("ListRooms: %d, %v", len(rooms), err)
	}

	// Affiliations
	aff := &storage.MUCAffiliation{
		RoomJID: room.RoomJID, UserJID: "alice@example.com",
		Affiliation: "owner",
	}
	if err := ms.SetAffiliation(ctx, aff); err != nil {
		t.Fatalf("SetAffiliation: %v", err)
	}
	gotAff, err := ms.GetAffiliation(ctx, room.RoomJID, "alice@example.com")
	if err != nil || gotAff.Affiliation != "owner" {
		t.Fatalf("GetAffiliation: %+v, %v", gotAff, err)
	}
	affs, err := ms.GetAffiliations(ctx, room.RoomJID)
	if err != nil || len(affs) != 1 {
		t.Fatalf("GetAffiliations: %d, %v", len(affs), err)
	}

	// Delete room
	if err := ms.DeleteRoom(ctx, room.RoomJID); err != nil {
		t.Fatalf("DeleteRoom: %v", err)
	}
	_, err = ms.GetRoom(ctx, room.RoomJID)
	if err != storage.ErrNotFound {
		t.Fatalf("GetRoom after delete: got %v", err)
	}
}

func testPubSubStore(t *testing.T, newStore func() storage.Storage) {
	s := initStore(t, newStore)
	ps := s.PubSubStore()
	if ps == nil {
		t.Skip("PubSubStore not supported")
	}
	ctx := context.Background()

	node := &storage.PubSubNode{Host: "pubsub.example.com", NodeID: "news", Name: "News", Type: "leaf"}

	// Create node
	if err := ps.CreateNode(ctx, node); err != nil {
		t.Fatalf("CreateNode: %v", err)
	}

	// Get node
	got, err := ps.GetNode(ctx, "pubsub.example.com", "news")
	if err != nil || got.Name != "News" {
		t.Fatalf("GetNode: %+v, %v", got, err)
	}

	// List nodes
	nodes, err := ps.ListNodes(ctx, "pubsub.example.com")
	if err != nil || len(nodes) != 1 {
		t.Fatalf("ListNodes: %d, %v", len(nodes), err)
	}

	// Publish item
	item := &storage.PubSubItem{
		Host: "pubsub.example.com", NodeID: "news", ItemID: "item1",
		Publisher: "alice@example.com", Payload: []byte("<entry/>"),
	}
	if err := ps.UpsertItem(ctx, item); err != nil {
		t.Fatalf("UpsertItem: %v", err)
	}

	// Get items
	items, err := ps.GetItems(ctx, "pubsub.example.com", "news")
	if err != nil || len(items) != 1 {
		t.Fatalf("GetItems: %d, %v", len(items), err)
	}

	// Subscribe
	sub := &storage.PubSubSubscription{
		Host: "pubsub.example.com", NodeID: "news",
		JID: "bob@example.com", State: "subscribed",
	}
	if err := ps.Subscribe(ctx, sub); err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	// Get subscriptions
	subs, err := ps.GetSubscriptions(ctx, "pubsub.example.com", "news")
	if err != nil || len(subs) != 1 {
		t.Fatalf("GetSubscriptions: %d, %v", len(subs), err)
	}

	// User subscriptions
	userSubs, err := ps.GetUserSubscriptions(ctx, "pubsub.example.com", "bob@example.com")
	if err != nil || len(userSubs) != 1 {
		t.Fatalf("GetUserSubscriptions: %d, %v", len(userSubs), err)
	}

	// Delete node (should clean up items and subs)
	if err := ps.DeleteNode(ctx, "pubsub.example.com", "news"); err != nil {
		t.Fatalf("DeleteNode: %v", err)
	}
	_, err = ps.GetNode(ctx, "pubsub.example.com", "news")
	if err != storage.ErrNotFound {
		t.Fatalf("GetNode after delete: got %v", err)
	}
}

func testBookmarkStore(t *testing.T, newStore func() storage.Storage) {
	s := initStore(t, newStore)
	bs := s.BookmarkStore()
	if bs == nil {
		t.Skip("BookmarkStore not supported")
	}
	ctx := context.Background()

	bm := &storage.Bookmark{
		UserJID: "alice@example.com", RoomJID: "room@conference.example.com",
		Name: "My Room", Nick: "alice", Autojoin: true,
	}

	// Set
	if err := bs.SetBookmark(ctx, bm); err != nil {
		t.Fatalf("SetBookmark: %v", err)
	}

	// Get
	got, err := bs.GetBookmark(ctx, "alice@example.com", "room@conference.example.com")
	if err != nil || got.Name != "My Room" || !got.Autojoin {
		t.Fatalf("GetBookmark: %+v, %v", got, err)
	}

	// List
	bms, err := bs.GetBookmarks(ctx, "alice@example.com")
	if err != nil || len(bms) != 1 {
		t.Fatalf("GetBookmarks: %d, %v", len(bms), err)
	}

	// Update
	bm.Name = "Updated Room"
	if err := bs.SetBookmark(ctx, bm); err != nil {
		t.Fatalf("SetBookmark update: %v", err)
	}
	got, _ = bs.GetBookmark(ctx, "alice@example.com", "room@conference.example.com")
	if got.Name != "Updated Room" {
		t.Fatalf("GetBookmark after update: %+v", got)
	}

	// Delete
	if err := bs.DeleteBookmark(ctx, "alice@example.com", "room@conference.example.com"); err != nil {
		t.Fatalf("DeleteBookmark: %v", err)
	}
	_, err = bs.GetBookmark(ctx, "alice@example.com", "room@conference.example.com")
	if err != storage.ErrNotFound {
		t.Fatalf("GetBookmark after delete: got %v", err)
	}
}
