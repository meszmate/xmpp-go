// Package file provides a file-based JSON storage backend for xmpp-go.
// Data is stored as JSON files on disk with file-level locking for concurrent access.
package file

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/meszmate/xmpp-go/storage"
)

// Store implements storage.Storage using JSON files on disk.
type Store struct {
	mu      sync.RWMutex
	baseDir string
}

// New creates a new file-based store rooted at baseDir.
func New(baseDir string) *Store {
	return &Store{baseDir: baseDir}
}

func (s *Store) Init(_ context.Context) error {
	dirs := []string{
		"users", "roster", "roster_versions", "blocking", "vcards",
		"offline", "mam", "muc_rooms", "muc_affiliations",
		"pubsub_nodes", "pubsub_items", "pubsub_subscriptions", "bookmarks",
	}
	for _, d := range dirs {
		if err := os.MkdirAll(filepath.Join(s.baseDir, d), 0o755); err != nil {
			return fmt.Errorf("file: create dir %s: %w", d, err)
		}
	}
	return nil
}

func (s *Store) Close() error { return nil }

func (s *Store) UserStore() storage.UserStore         { return s }
func (s *Store) RosterStore() storage.RosterStore     { return s }
func (s *Store) BlockingStore() storage.BlockingStore { return s }
func (s *Store) VCardStore() storage.VCardStore       { return s }
func (s *Store) OfflineStore() storage.OfflineStore   { return s }
func (s *Store) MAMStore() storage.MAMStore           { return s }
func (s *Store) MUCRoomStore() storage.MUCRoomStore   { return s }
func (s *Store) PubSubStore() storage.PubSubStore     { return s }
func (s *Store) BookmarkStore() storage.BookmarkStore { return s }

// File helpers

func (s *Store) path(parts ...string) string {
	return filepath.Join(append([]string{s.baseDir}, parts...)...)
}

func (s *Store) writeJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func (s *Store) readJSON(path string, v any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return storage.ErrNotFound
		}
		return err
	}
	return json.Unmarshal(data, v)
}

func (s *Store) exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func safeFileName(name string) string {
	// Replace characters that are problematic in filenames.
	result := make([]byte, len(name))
	for i := 0; i < len(name); i++ {
		c := name[i]
		if c == '/' || c == '\\' || c == ':' || c == '*' || c == '?' || c == '"' || c == '<' || c == '>' || c == '|' {
			result[i] = '_'
		} else {
			result[i] = c
		}
	}
	return string(result)
}

// --- UserStore ---

func (s *Store) CreateUser(_ context.Context, user *storage.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	p := s.path("users", safeFileName(user.Username)+".json")
	if s.exists(p) {
		return storage.ErrUserExists
	}
	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now
	return s.writeJSON(p, user)
}

func (s *Store) GetUser(_ context.Context, username string) (*storage.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var user storage.User
	if err := s.readJSON(s.path("users", safeFileName(username)+".json"), &user); err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *Store) UpdateUser(_ context.Context, user *storage.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	p := s.path("users", safeFileName(user.Username)+".json")
	if !s.exists(p) {
		return storage.ErrNotFound
	}
	user.UpdatedAt = time.Now()
	return s.writeJSON(p, user)
}

func (s *Store) DeleteUser(_ context.Context, username string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	p := s.path("users", safeFileName(username)+".json")
	if !s.exists(p) {
		return storage.ErrNotFound
	}
	return os.Remove(p)
}

func (s *Store) UserExists(_ context.Context, username string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.exists(s.path("users", safeFileName(username)+".json")), nil
}

func (s *Store) Authenticate(_ context.Context, username, password string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var user storage.User
	if err := s.readJSON(s.path("users", safeFileName(username)+".json"), &user); err != nil {
		return false, storage.ErrAuthFailed
	}
	if user.Password != password {
		return false, storage.ErrAuthFailed
	}
	return true, nil
}

// --- RosterStore ---

type rosterFile struct {
	Items map[string]*storage.RosterItem `json:"items"`
}

func (s *Store) rosterPath(userJID string) string {
	return s.path("roster", safeFileName(userJID)+".json")
}

func (s *Store) loadRoster(userJID string) (*rosterFile, error) {
	var rf rosterFile
	if err := s.readJSON(s.rosterPath(userJID), &rf); err != nil {
		if err == storage.ErrNotFound {
			return &rosterFile{Items: make(map[string]*storage.RosterItem)}, nil
		}
		return nil, err
	}
	if rf.Items == nil {
		rf.Items = make(map[string]*storage.RosterItem)
	}
	return &rf, nil
}

func (s *Store) UpsertRosterItem(_ context.Context, item *storage.RosterItem) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	rf, err := s.loadRoster(item.UserJID)
	if err != nil {
		return err
	}
	cp := *item
	rf.Items[item.ContactJID] = &cp
	return s.writeJSON(s.rosterPath(item.UserJID), rf)
}

func (s *Store) GetRosterItem(_ context.Context, userJID, contactJID string) (*storage.RosterItem, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rf, err := s.loadRoster(userJID)
	if err != nil {
		return nil, err
	}
	item, ok := rf.Items[contactJID]
	if !ok {
		return nil, storage.ErrNotFound
	}
	cp := *item
	return &cp, nil
}

func (s *Store) GetRosterItems(_ context.Context, userJID string) ([]*storage.RosterItem, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rf, err := s.loadRoster(userJID)
	if err != nil {
		return nil, err
	}
	items := make([]*storage.RosterItem, 0, len(rf.Items))
	for _, item := range rf.Items {
		cp := *item
		items = append(items, &cp)
	}
	return items, nil
}

func (s *Store) DeleteRosterItem(_ context.Context, userJID, contactJID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	rf, err := s.loadRoster(userJID)
	if err != nil {
		return err
	}
	if _, ok := rf.Items[contactJID]; !ok {
		return storage.ErrNotFound
	}
	delete(rf.Items, contactJID)
	return s.writeJSON(s.rosterPath(userJID), rf)
}

func (s *Store) GetRosterVersion(_ context.Context, userJID string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var ver string
	if err := s.readJSON(s.path("roster_versions", safeFileName(userJID)+".json"), &ver); err != nil {
		if err == storage.ErrNotFound {
			return "", nil
		}
		return "", err
	}
	return ver, nil
}

func (s *Store) SetRosterVersion(_ context.Context, userJID, version string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.writeJSON(s.path("roster_versions", safeFileName(userJID)+".json"), version)
}

// --- BlockingStore ---

func (s *Store) blockingPath(userJID string) string {
	return s.path("blocking", safeFileName(userJID)+".json")
}

func (s *Store) loadBlocked(userJID string) (map[string]bool, error) {
	var blocked map[string]bool
	if err := s.readJSON(s.blockingPath(userJID), &blocked); err != nil {
		if err == storage.ErrNotFound {
			return make(map[string]bool), nil
		}
		return nil, err
	}
	return blocked, nil
}

func (s *Store) BlockJID(_ context.Context, userJID, blockedJID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	blocked, err := s.loadBlocked(userJID)
	if err != nil {
		return err
	}
	blocked[blockedJID] = true
	return s.writeJSON(s.blockingPath(userJID), blocked)
}

func (s *Store) UnblockJID(_ context.Context, userJID, blockedJID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	blocked, err := s.loadBlocked(userJID)
	if err != nil {
		return err
	}
	delete(blocked, blockedJID)
	return s.writeJSON(s.blockingPath(userJID), blocked)
}

func (s *Store) IsBlocked(_ context.Context, userJID, blockedJID string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	blocked, err := s.loadBlocked(userJID)
	if err != nil {
		return false, err
	}
	return blocked[blockedJID], nil
}

func (s *Store) GetBlockedJIDs(_ context.Context, userJID string) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	blocked, err := s.loadBlocked(userJID)
	if err != nil {
		return nil, err
	}
	jids := make([]string, 0, len(blocked))
	for jid := range blocked {
		jids = append(jids, jid)
	}
	return jids, nil
}

// --- VCardStore ---

func (s *Store) SetVCard(_ context.Context, userJID string, data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return os.WriteFile(s.path("vcards", safeFileName(userJID)+".xml"), data, 0o644)
}

func (s *Store) GetVCard(_ context.Context, userJID string) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	data, err := os.ReadFile(s.path("vcards", safeFileName(userJID)+".xml"))
	if os.IsNotExist(err) {
		return nil, storage.ErrNotFound
	}
	return data, err
}

func (s *Store) DeleteVCard(_ context.Context, userJID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	p := s.path("vcards", safeFileName(userJID)+".xml")
	if !s.exists(p) {
		return storage.ErrNotFound
	}
	return os.Remove(p)
}

// --- OfflineStore ---

func (s *Store) offlinePath(userJID string) string {
	return s.path("offline", safeFileName(userJID)+".json")
}

func (s *Store) loadOffline(userJID string) ([]*storage.OfflineMessage, error) {
	var msgs []*storage.OfflineMessage
	if err := s.readJSON(s.offlinePath(userJID), &msgs); err != nil {
		if err == storage.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}
	return msgs, nil
}

func (s *Store) StoreOfflineMessage(_ context.Context, msg *storage.OfflineMessage) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	msgs, err := s.loadOffline(msg.UserJID)
	if err != nil {
		return err
	}
	cp := *msg
	if cp.CreatedAt.IsZero() {
		cp.CreatedAt = time.Now()
	}
	msgs = append(msgs, &cp)
	return s.writeJSON(s.offlinePath(msg.UserJID), msgs)
}

func (s *Store) GetOfflineMessages(_ context.Context, userJID string) ([]*storage.OfflineMessage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	msgs, err := s.loadOffline(userJID)
	if err != nil {
		return nil, err
	}
	return msgs, nil
}

func (s *Store) DeleteOfflineMessages(_ context.Context, userJID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	p := s.offlinePath(userJID)
	if !s.exists(p) {
		return nil
	}
	return os.Remove(p)
}

func (s *Store) CountOfflineMessages(_ context.Context, userJID string) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	msgs, err := s.loadOffline(userJID)
	if err != nil {
		return 0, err
	}
	return len(msgs), nil
}

// --- MAMStore ---

func (s *Store) mamPath(userJID string) string {
	return s.path("mam", safeFileName(userJID)+".json")
}

func (s *Store) loadMAM(userJID string) ([]*storage.ArchivedMessage, error) {
	var msgs []*storage.ArchivedMessage
	if err := s.readJSON(s.mamPath(userJID), &msgs); err != nil {
		if err == storage.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}
	return msgs, nil
}

var mamCounter int64

func (s *Store) ArchiveMessage(_ context.Context, msg *storage.ArchivedMessage) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	msgs, err := s.loadMAM(msg.UserJID)
	if err != nil {
		return err
	}
	cp := *msg
	if cp.CreatedAt.IsZero() {
		cp.CreatedAt = time.Now()
	}
	if cp.ID == "" {
		mamCounter++
		cp.ID = fmt.Sprintf("%d", mamCounter)
	}
	msgs = append(msgs, &cp)
	return s.writeJSON(s.mamPath(msg.UserJID), msgs)
}

func (s *Store) QueryMessages(_ context.Context, query *storage.MAMQuery) (*storage.MAMResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	msgs, err := s.loadMAM(query.UserJID)
	if err != nil {
		return nil, err
	}

	var filtered []*storage.ArchivedMessage
	afterIDFound := query.AfterID == ""
	for _, msg := range msgs {
		if !afterIDFound {
			if msg.ID == query.AfterID {
				afterIDFound = true
			}
			continue
		}
		if query.BeforeID != "" && msg.ID == query.BeforeID {
			break
		}
		if query.WithJID != "" && msg.WithJID != query.WithJID {
			continue
		}
		if !query.Start.IsZero() && msg.CreatedAt.Before(query.Start) {
			continue
		}
		if !query.End.IsZero() && msg.CreatedAt.After(query.End) {
			continue
		}
		filtered = append(filtered, msg)
	}

	max := query.Max
	if max <= 0 {
		max = 100
	}
	complete := len(filtered) <= max
	if len(filtered) > max {
		filtered = filtered[:max]
	}

	result := &storage.MAMResult{
		Messages: filtered, Complete: complete, Count: len(filtered),
	}
	if len(filtered) > 0 {
		result.First = filtered[0].ID
		result.Last = filtered[len(filtered)-1].ID
	}
	return result, nil
}

func (s *Store) DeleteMessageArchive(_ context.Context, userJID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	p := s.mamPath(userJID)
	if !s.exists(p) {
		return nil
	}
	return os.Remove(p)
}

// --- MUCRoomStore ---

func (s *Store) mucRoomPath(roomJID string) string {
	return s.path("muc_rooms", safeFileName(roomJID)+".json")
}

func (s *Store) CreateRoom(_ context.Context, room *storage.MUCRoom) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	p := s.mucRoomPath(room.RoomJID)
	if s.exists(p) {
		return storage.ErrUserExists
	}
	return s.writeJSON(p, room)
}

func (s *Store) GetRoom(_ context.Context, roomJID string) (*storage.MUCRoom, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var room storage.MUCRoom
	if err := s.readJSON(s.mucRoomPath(roomJID), &room); err != nil {
		return nil, err
	}
	return &room, nil
}

func (s *Store) UpdateRoom(_ context.Context, room *storage.MUCRoom) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	p := s.mucRoomPath(room.RoomJID)
	if !s.exists(p) {
		return storage.ErrNotFound
	}
	return s.writeJSON(p, room)
}

func (s *Store) DeleteRoom(_ context.Context, roomJID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	p := s.mucRoomPath(roomJID)
	if !s.exists(p) {
		return storage.ErrNotFound
	}
	os.Remove(s.path("muc_affiliations", safeFileName(roomJID)+".json"))
	return os.Remove(p)
}

func (s *Store) ListRooms(_ context.Context) ([]*storage.MUCRoom, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	entries, err := os.ReadDir(s.path("muc_rooms"))
	if err != nil {
		return nil, err
	}
	var rooms []*storage.MUCRoom
	for _, e := range entries {
		var room storage.MUCRoom
		if err := s.readJSON(filepath.Join(s.path("muc_rooms"), e.Name()), &room); err != nil {
			continue
		}
		rooms = append(rooms, &room)
	}
	return rooms, nil
}

func (s *Store) mucAffPath(roomJID string) string {
	return s.path("muc_affiliations", safeFileName(roomJID)+".json")
}

func (s *Store) loadAffs(roomJID string) (map[string]*storage.MUCAffiliation, error) {
	var affs map[string]*storage.MUCAffiliation
	if err := s.readJSON(s.mucAffPath(roomJID), &affs); err != nil {
		if err == storage.ErrNotFound {
			return make(map[string]*storage.MUCAffiliation), nil
		}
		return nil, err
	}
	return affs, nil
}

func (s *Store) SetAffiliation(_ context.Context, aff *storage.MUCAffiliation) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	affs, err := s.loadAffs(aff.RoomJID)
	if err != nil {
		return err
	}
	cp := *aff
	affs[aff.UserJID] = &cp
	return s.writeJSON(s.mucAffPath(aff.RoomJID), affs)
}

func (s *Store) GetAffiliation(_ context.Context, roomJID, userJID string) (*storage.MUCAffiliation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	affs, err := s.loadAffs(roomJID)
	if err != nil {
		return nil, err
	}
	aff, ok := affs[userJID]
	if !ok {
		return nil, storage.ErrNotFound
	}
	cp := *aff
	return &cp, nil
}

func (s *Store) GetAffiliations(_ context.Context, roomJID string) ([]*storage.MUCAffiliation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	affs, err := s.loadAffs(roomJID)
	if err != nil {
		return nil, err
	}
	result := make([]*storage.MUCAffiliation, 0, len(affs))
	for _, aff := range affs {
		cp := *aff
		result = append(result, &cp)
	}
	return result, nil
}

func (s *Store) RemoveAffiliation(_ context.Context, roomJID, userJID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	affs, err := s.loadAffs(roomJID)
	if err != nil {
		return err
	}
	delete(affs, userJID)
	return s.writeJSON(s.mucAffPath(roomJID), affs)
}

// --- PubSubStore ---

func (s *Store) pubsubNodePath(host, nodeID string) string {
	return s.path("pubsub_nodes", safeFileName(host)+"_"+safeFileName(nodeID)+".json")
}

func (s *Store) CreateNode(_ context.Context, node *storage.PubSubNode) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	p := s.pubsubNodePath(node.Host, node.NodeID)
	if s.exists(p) {
		return storage.ErrUserExists
	}
	return s.writeJSON(p, node)
}

func (s *Store) GetNode(_ context.Context, host, nodeID string) (*storage.PubSubNode, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var node storage.PubSubNode
	if err := s.readJSON(s.pubsubNodePath(host, nodeID), &node); err != nil {
		return nil, err
	}
	return &node, nil
}

func (s *Store) DeleteNode(_ context.Context, host, nodeID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	p := s.pubsubNodePath(host, nodeID)
	if !s.exists(p) {
		return storage.ErrNotFound
	}
	os.Remove(s.path("pubsub_items", safeFileName(host)+"_"+safeFileName(nodeID)+".json"))
	os.Remove(s.path("pubsub_subscriptions", safeFileName(host)+"_"+safeFileName(nodeID)+".json"))
	return os.Remove(p)
}

func (s *Store) ListNodes(_ context.Context, host string) ([]*storage.PubSubNode, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	entries, err := os.ReadDir(s.path("pubsub_nodes"))
	if err != nil {
		return nil, err
	}
	prefix := safeFileName(host) + "_"
	var nodes []*storage.PubSubNode
	for _, e := range entries {
		if len(e.Name()) > len(prefix) && e.Name()[:len(prefix)] == prefix {
			var node storage.PubSubNode
			if err := s.readJSON(filepath.Join(s.path("pubsub_nodes"), e.Name()), &node); err != nil {
				continue
			}
			nodes = append(nodes, &node)
		}
	}
	return nodes, nil
}

func (s *Store) pubsubItemsPath(host, nodeID string) string {
	return s.path("pubsub_items", safeFileName(host)+"_"+safeFileName(nodeID)+".json")
}

func (s *Store) loadPubsubItems(host, nodeID string) (map[string]*storage.PubSubItem, error) {
	var items map[string]*storage.PubSubItem
	if err := s.readJSON(s.pubsubItemsPath(host, nodeID), &items); err != nil {
		if err == storage.ErrNotFound {
			return make(map[string]*storage.PubSubItem), nil
		}
		return nil, err
	}
	return items, nil
}

func (s *Store) UpsertItem(_ context.Context, item *storage.PubSubItem) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	items, err := s.loadPubsubItems(item.Host, item.NodeID)
	if err != nil {
		return err
	}
	cp := *item
	if cp.CreatedAt.IsZero() {
		cp.CreatedAt = time.Now()
	}
	items[item.ItemID] = &cp
	return s.writeJSON(s.pubsubItemsPath(item.Host, item.NodeID), items)
}

func (s *Store) GetItem(_ context.Context, host, nodeID, itemID string) (*storage.PubSubItem, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items, err := s.loadPubsubItems(host, nodeID)
	if err != nil {
		return nil, err
	}
	item, ok := items[itemID]
	if !ok {
		return nil, storage.ErrNotFound
	}
	cp := *item
	return &cp, nil
}

func (s *Store) GetItems(_ context.Context, host, nodeID string) ([]*storage.PubSubItem, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items, err := s.loadPubsubItems(host, nodeID)
	if err != nil {
		return nil, err
	}
	result := make([]*storage.PubSubItem, 0, len(items))
	for _, item := range items {
		cp := *item
		result = append(result, &cp)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.Before(result[j].CreatedAt)
	})
	return result, nil
}

func (s *Store) DeleteItem(_ context.Context, host, nodeID, itemID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	items, err := s.loadPubsubItems(host, nodeID)
	if err != nil {
		return err
	}
	if _, ok := items[itemID]; !ok {
		return storage.ErrNotFound
	}
	delete(items, itemID)
	return s.writeJSON(s.pubsubItemsPath(host, nodeID), items)
}

func (s *Store) pubsubSubsPath(host, nodeID string) string {
	return s.path("pubsub_subscriptions", safeFileName(host)+"_"+safeFileName(nodeID)+".json")
}

func (s *Store) loadPubsubSubs(host, nodeID string) (map[string]*storage.PubSubSubscription, error) {
	var subs map[string]*storage.PubSubSubscription
	if err := s.readJSON(s.pubsubSubsPath(host, nodeID), &subs); err != nil {
		if err == storage.ErrNotFound {
			return make(map[string]*storage.PubSubSubscription), nil
		}
		return nil, err
	}
	return subs, nil
}

func (s *Store) Subscribe(_ context.Context, sub *storage.PubSubSubscription) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	subs, err := s.loadPubsubSubs(sub.Host, sub.NodeID)
	if err != nil {
		return err
	}
	cp := *sub
	subs[sub.JID] = &cp
	return s.writeJSON(s.pubsubSubsPath(sub.Host, sub.NodeID), subs)
}

func (s *Store) Unsubscribe(_ context.Context, host, nodeID, jid string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	subs, err := s.loadPubsubSubs(host, nodeID)
	if err != nil {
		return err
	}
	delete(subs, jid)
	return s.writeJSON(s.pubsubSubsPath(host, nodeID), subs)
}

func (s *Store) GetSubscription(_ context.Context, host, nodeID, jid string) (*storage.PubSubSubscription, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	subs, err := s.loadPubsubSubs(host, nodeID)
	if err != nil {
		return nil, err
	}
	sub, ok := subs[jid]
	if !ok {
		return nil, storage.ErrNotFound
	}
	cp := *sub
	return &cp, nil
}

func (s *Store) GetSubscriptions(_ context.Context, host, nodeID string) ([]*storage.PubSubSubscription, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	subs, err := s.loadPubsubSubs(host, nodeID)
	if err != nil {
		return nil, err
	}
	result := make([]*storage.PubSubSubscription, 0, len(subs))
	for _, sub := range subs {
		cp := *sub
		result = append(result, &cp)
	}
	return result, nil
}

func (s *Store) GetUserSubscriptions(_ context.Context, host, jid string) ([]*storage.PubSubSubscription, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	entries, err := os.ReadDir(s.path("pubsub_subscriptions"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	prefix := safeFileName(host) + "_"
	var result []*storage.PubSubSubscription
	for _, e := range entries {
		if len(e.Name()) > len(prefix) && e.Name()[:len(prefix)] == prefix {
			var subs map[string]*storage.PubSubSubscription
			if err := s.readJSON(filepath.Join(s.path("pubsub_subscriptions"), e.Name()), &subs); err != nil {
				continue
			}
			if sub, ok := subs[jid]; ok {
				cp := *sub
				result = append(result, &cp)
			}
		}
	}
	return result, nil
}

// --- BookmarkStore ---

func (s *Store) bookmarkPath(userJID string) string {
	return s.path("bookmarks", safeFileName(userJID)+".json")
}

func (s *Store) loadBookmarks(userJID string) (map[string]*storage.Bookmark, error) {
	var bms map[string]*storage.Bookmark
	if err := s.readJSON(s.bookmarkPath(userJID), &bms); err != nil {
		if err == storage.ErrNotFound {
			return make(map[string]*storage.Bookmark), nil
		}
		return nil, err
	}
	return bms, nil
}

func (s *Store) SetBookmark(_ context.Context, bm *storage.Bookmark) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	bms, err := s.loadBookmarks(bm.UserJID)
	if err != nil {
		return err
	}
	cp := *bm
	bms[bm.RoomJID] = &cp
	return s.writeJSON(s.bookmarkPath(bm.UserJID), bms)
}

func (s *Store) GetBookmark(_ context.Context, userJID, roomJID string) (*storage.Bookmark, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	bms, err := s.loadBookmarks(userJID)
	if err != nil {
		return nil, err
	}
	bm, ok := bms[roomJID]
	if !ok {
		return nil, storage.ErrNotFound
	}
	cp := *bm
	return &cp, nil
}

func (s *Store) GetBookmarks(_ context.Context, userJID string) ([]*storage.Bookmark, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	bms, err := s.loadBookmarks(userJID)
	if err != nil {
		return nil, err
	}
	result := make([]*storage.Bookmark, 0, len(bms))
	for _, bm := range bms {
		cp := *bm
		result = append(result, &cp)
	}
	return result, nil
}

func (s *Store) DeleteBookmark(_ context.Context, userJID, roomJID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	bms, err := s.loadBookmarks(userJID)
	if err != nil {
		return err
	}
	if _, ok := bms[roomJID]; !ok {
		return storage.ErrNotFound
	}
	delete(bms, roomJID)
	return s.writeJSON(s.bookmarkPath(userJID), bms)
}
