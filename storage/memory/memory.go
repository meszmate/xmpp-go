// Package memory provides an in-memory implementation of the storage interfaces.
package memory

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/meszmate/xmpp-go/storage"
)

// Store is an in-memory implementation of storage.Storage.
type Store struct {
	mu sync.RWMutex

	// users
	users map[string]*storage.User

	// roster
	rosterItems    map[string]map[string]*storage.RosterItem // userJID -> contactJID -> item
	rosterVersions map[string]string                         // userJID -> version

	// blocking
	blocked map[string]map[string]bool // userJID -> blockedJID -> true

	// vcards
	vcards map[string][]byte // userJID -> raw XML

	// offline messages
	offlineMsgs map[string][]*storage.OfflineMessage // userJID -> messages

	// MAM
	mamMessages map[string][]*storage.ArchivedMessage // userJID -> messages
	mamIDCounter int64

	// MUC rooms
	mucRooms        map[string]*storage.MUCRoom                    // roomJID -> room
	mucAffiliations map[string]map[string]*storage.MUCAffiliation  // roomJID -> userJID -> aff

	// PubSub
	pubsubNodes         map[string]map[string]*storage.PubSubNode            // host -> nodeID -> node
	pubsubItems         map[string]map[string]map[string]*storage.PubSubItem // host -> nodeID -> itemID -> item
	pubsubSubscriptions map[string]map[string]map[string]*storage.PubSubSubscription // host -> nodeID -> jid -> sub

	// Bookmarks
	bookmarks map[string]map[string]*storage.Bookmark // userJID -> roomJID -> bookmark
}

// New creates a new in-memory store.
func New() *Store {
	return &Store{}
}

func (s *Store) Init(_ context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.users = make(map[string]*storage.User)
	s.rosterItems = make(map[string]map[string]*storage.RosterItem)
	s.rosterVersions = make(map[string]string)
	s.blocked = make(map[string]map[string]bool)
	s.vcards = make(map[string][]byte)
	s.offlineMsgs = make(map[string][]*storage.OfflineMessage)
	s.mamMessages = make(map[string][]*storage.ArchivedMessage)
	s.mucRooms = make(map[string]*storage.MUCRoom)
	s.mucAffiliations = make(map[string]map[string]*storage.MUCAffiliation)
	s.pubsubNodes = make(map[string]map[string]*storage.PubSubNode)
	s.pubsubItems = make(map[string]map[string]map[string]*storage.PubSubItem)
	s.pubsubSubscriptions = make(map[string]map[string]map[string]*storage.PubSubSubscription)
	s.bookmarks = make(map[string]map[string]*storage.Bookmark)
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

// --- UserStore ---

func (s *Store) CreateUser(_ context.Context, user *storage.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.users[user.Username]; ok {
		return storage.ErrUserExists
	}
	now := time.Now()
	u := *user
	u.CreatedAt = now
	u.UpdatedAt = now
	s.users[user.Username] = &u
	return nil
}

func (s *Store) GetUser(_ context.Context, username string) (*storage.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	u, ok := s.users[username]
	if !ok {
		return nil, storage.ErrNotFound
	}
	cp := *u
	return &cp, nil
}

func (s *Store) UpdateUser(_ context.Context, user *storage.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.users[user.Username]; !ok {
		return storage.ErrNotFound
	}
	u := *user
	u.UpdatedAt = time.Now()
	s.users[user.Username] = &u
	return nil
}

func (s *Store) DeleteUser(_ context.Context, username string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.users[username]; !ok {
		return storage.ErrNotFound
	}
	delete(s.users, username)
	return nil
}

func (s *Store) UserExists(_ context.Context, username string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.users[username]
	return ok, nil
}

func (s *Store) Authenticate(_ context.Context, username, password string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	u, ok := s.users[username]
	if !ok {
		return false, storage.ErrAuthFailed
	}
	if u.Password != password {
		return false, storage.ErrAuthFailed
	}
	return true, nil
}

// --- RosterStore ---

func (s *Store) UpsertRosterItem(_ context.Context, item *storage.RosterItem) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.rosterItems[item.UserJID] == nil {
		s.rosterItems[item.UserJID] = make(map[string]*storage.RosterItem)
	}
	cp := *item
	cp.Groups = append([]string(nil), item.Groups...)
	s.rosterItems[item.UserJID][item.ContactJID] = &cp
	return nil
}

func (s *Store) GetRosterItem(_ context.Context, userJID, contactJID string) (*storage.RosterItem, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items, ok := s.rosterItems[userJID]
	if !ok {
		return nil, storage.ErrNotFound
	}
	item, ok := items[contactJID]
	if !ok {
		return nil, storage.ErrNotFound
	}
	cp := *item
	cp.Groups = append([]string(nil), item.Groups...)
	return &cp, nil
}

func (s *Store) GetRosterItems(_ context.Context, userJID string) ([]*storage.RosterItem, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := s.rosterItems[userJID]
	result := make([]*storage.RosterItem, 0, len(items))
	for _, item := range items {
		cp := *item
		cp.Groups = append([]string(nil), item.Groups...)
		result = append(result, &cp)
	}
	return result, nil
}

func (s *Store) DeleteRosterItem(_ context.Context, userJID, contactJID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	items, ok := s.rosterItems[userJID]
	if !ok {
		return storage.ErrNotFound
	}
	if _, ok := items[contactJID]; !ok {
		return storage.ErrNotFound
	}
	delete(items, contactJID)
	return nil
}

func (s *Store) GetRosterVersion(_ context.Context, userJID string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.rosterVersions[userJID], nil
}

func (s *Store) SetRosterVersion(_ context.Context, userJID, version string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rosterVersions[userJID] = version
	return nil
}

// --- BlockingStore ---

func (s *Store) BlockJID(_ context.Context, userJID, blockedJID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.blocked[userJID] == nil {
		s.blocked[userJID] = make(map[string]bool)
	}
	s.blocked[userJID][blockedJID] = true
	return nil
}

func (s *Store) UnblockJID(_ context.Context, userJID, blockedJID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.blocked[userJID] != nil {
		delete(s.blocked[userJID], blockedJID)
	}
	return nil
}

func (s *Store) IsBlocked(_ context.Context, userJID, blockedJID string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.blocked[userJID] == nil {
		return false, nil
	}
	return s.blocked[userJID][blockedJID], nil
}

func (s *Store) GetBlockedJIDs(_ context.Context, userJID string) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	blocked := s.blocked[userJID]
	result := make([]string, 0, len(blocked))
	for jid := range blocked {
		result = append(result, jid)
	}
	return result, nil
}

// --- VCardStore ---

func (s *Store) SetVCard(_ context.Context, userJID string, data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.vcards[userJID] = append([]byte(nil), data...)
	return nil
}

func (s *Store) GetVCard(_ context.Context, userJID string) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	data, ok := s.vcards[userJID]
	if !ok {
		return nil, storage.ErrNotFound
	}
	return append([]byte(nil), data...), nil
}

func (s *Store) DeleteVCard(_ context.Context, userJID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.vcards[userJID]; !ok {
		return storage.ErrNotFound
	}
	delete(s.vcards, userJID)
	return nil
}

// --- OfflineStore ---

func (s *Store) StoreOfflineMessage(_ context.Context, msg *storage.OfflineMessage) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := *msg
	cp.Data = append([]byte(nil), msg.Data...)
	if cp.CreatedAt.IsZero() {
		cp.CreatedAt = time.Now()
	}
	s.offlineMsgs[msg.UserJID] = append(s.offlineMsgs[msg.UserJID], &cp)
	return nil
}

func (s *Store) GetOfflineMessages(_ context.Context, userJID string) ([]*storage.OfflineMessage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	msgs := s.offlineMsgs[userJID]
	result := make([]*storage.OfflineMessage, len(msgs))
	for i, msg := range msgs {
		cp := *msg
		cp.Data = append([]byte(nil), msg.Data...)
		result[i] = &cp
	}
	return result, nil
}

func (s *Store) DeleteOfflineMessages(_ context.Context, userJID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.offlineMsgs, userJID)
	return nil
}

func (s *Store) CountOfflineMessages(_ context.Context, userJID string) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.offlineMsgs[userJID]), nil
}

// --- MAMStore ---

func (s *Store) ArchiveMessage(_ context.Context, msg *storage.ArchivedMessage) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := *msg
	cp.Data = append([]byte(nil), msg.Data...)
	if cp.CreatedAt.IsZero() {
		cp.CreatedAt = time.Now()
	}
	if cp.ID == "" {
		s.mamIDCounter++
		cp.ID = fmt.Sprintf("%d", s.mamIDCounter)
	}
	s.mamMessages[msg.UserJID] = append(s.mamMessages[msg.UserJID], &cp)
	return nil
}

func (s *Store) QueryMessages(_ context.Context, query *storage.MAMQuery) (*storage.MAMResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	msgs := s.mamMessages[query.UserJID]
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
		cp := *msg
		cp.Data = append([]byte(nil), msg.Data...)
		filtered = append(filtered, &cp)
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
		Messages: filtered,
		Complete: complete,
		Count:    len(filtered),
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
	delete(s.mamMessages, userJID)
	return nil
}

// --- MUCRoomStore ---

func (s *Store) CreateRoom(_ context.Context, room *storage.MUCRoom) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.mucRooms[room.RoomJID]; ok {
		return storage.ErrUserExists // room already exists
	}
	cp := *room
	s.mucRooms[room.RoomJID] = &cp
	return nil
}

func (s *Store) GetRoom(_ context.Context, roomJID string) (*storage.MUCRoom, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	room, ok := s.mucRooms[roomJID]
	if !ok {
		return nil, storage.ErrNotFound
	}
	cp := *room
	return &cp, nil
}

func (s *Store) UpdateRoom(_ context.Context, room *storage.MUCRoom) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.mucRooms[room.RoomJID]; !ok {
		return storage.ErrNotFound
	}
	cp := *room
	s.mucRooms[room.RoomJID] = &cp
	return nil
}

func (s *Store) DeleteRoom(_ context.Context, roomJID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.mucRooms[roomJID]; !ok {
		return storage.ErrNotFound
	}
	delete(s.mucRooms, roomJID)
	delete(s.mucAffiliations, roomJID)
	return nil
}

func (s *Store) ListRooms(_ context.Context) ([]*storage.MUCRoom, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*storage.MUCRoom, 0, len(s.mucRooms))
	for _, room := range s.mucRooms {
		cp := *room
		result = append(result, &cp)
	}
	return result, nil
}

func (s *Store) SetAffiliation(_ context.Context, aff *storage.MUCAffiliation) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.mucAffiliations[aff.RoomJID] == nil {
		s.mucAffiliations[aff.RoomJID] = make(map[string]*storage.MUCAffiliation)
	}
	cp := *aff
	s.mucAffiliations[aff.RoomJID][aff.UserJID] = &cp
	return nil
}

func (s *Store) GetAffiliation(_ context.Context, roomJID, userJID string) (*storage.MUCAffiliation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	affs, ok := s.mucAffiliations[roomJID]
	if !ok {
		return nil, storage.ErrNotFound
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
	affs := s.mucAffiliations[roomJID]
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
	if affs, ok := s.mucAffiliations[roomJID]; ok {
		delete(affs, userJID)
	}
	return nil
}

// --- PubSubStore ---

func (s *Store) CreateNode(_ context.Context, node *storage.PubSubNode) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.pubsubNodes[node.Host] == nil {
		s.pubsubNodes[node.Host] = make(map[string]*storage.PubSubNode)
	}
	if _, ok := s.pubsubNodes[node.Host][node.NodeID]; ok {
		return storage.ErrUserExists // node already exists
	}
	cp := *node
	if node.Config != nil {
		cp.Config = make(map[string]string, len(node.Config))
		for k, v := range node.Config {
			cp.Config[k] = v
		}
	}
	s.pubsubNodes[node.Host][node.NodeID] = &cp
	return nil
}

func (s *Store) GetNode(_ context.Context, host, nodeID string) (*storage.PubSubNode, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	nodes, ok := s.pubsubNodes[host]
	if !ok {
		return nil, storage.ErrNotFound
	}
	node, ok := nodes[nodeID]
	if !ok {
		return nil, storage.ErrNotFound
	}
	cp := *node
	if node.Config != nil {
		cp.Config = make(map[string]string, len(node.Config))
		for k, v := range node.Config {
			cp.Config[k] = v
		}
	}
	return &cp, nil
}

func (s *Store) DeleteNode(_ context.Context, host, nodeID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if nodes, ok := s.pubsubNodes[host]; ok {
		if _, ok := nodes[nodeID]; !ok {
			return storage.ErrNotFound
		}
		delete(nodes, nodeID)
	} else {
		return storage.ErrNotFound
	}
	if items, ok := s.pubsubItems[host]; ok {
		delete(items, nodeID)
	}
	if subs, ok := s.pubsubSubscriptions[host]; ok {
		delete(subs, nodeID)
	}
	return nil
}

func (s *Store) ListNodes(_ context.Context, host string) ([]*storage.PubSubNode, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	nodes := s.pubsubNodes[host]
	result := make([]*storage.PubSubNode, 0, len(nodes))
	for _, node := range nodes {
		cp := *node
		result = append(result, &cp)
	}
	return result, nil
}

func (s *Store) UpsertItem(_ context.Context, item *storage.PubSubItem) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.pubsubItems[item.Host] == nil {
		s.pubsubItems[item.Host] = make(map[string]map[string]*storage.PubSubItem)
	}
	if s.pubsubItems[item.Host][item.NodeID] == nil {
		s.pubsubItems[item.Host][item.NodeID] = make(map[string]*storage.PubSubItem)
	}
	cp := *item
	cp.Payload = append([]byte(nil), item.Payload...)
	if cp.CreatedAt.IsZero() {
		cp.CreatedAt = time.Now()
	}
	s.pubsubItems[item.Host][item.NodeID][item.ItemID] = &cp
	return nil
}

func (s *Store) GetItem(_ context.Context, host, nodeID, itemID string) (*storage.PubSubItem, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	hostItems, ok := s.pubsubItems[host]
	if !ok {
		return nil, storage.ErrNotFound
	}
	nodeItems, ok := hostItems[nodeID]
	if !ok {
		return nil, storage.ErrNotFound
	}
	item, ok := nodeItems[itemID]
	if !ok {
		return nil, storage.ErrNotFound
	}
	cp := *item
	cp.Payload = append([]byte(nil), item.Payload...)
	return &cp, nil
}

func (s *Store) GetItems(_ context.Context, host, nodeID string) ([]*storage.PubSubItem, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	hostItems := s.pubsubItems[host]
	if hostItems == nil {
		return nil, nil
	}
	nodeItems := hostItems[nodeID]
	result := make([]*storage.PubSubItem, 0, len(nodeItems))
	for _, item := range nodeItems {
		cp := *item
		cp.Payload = append([]byte(nil), item.Payload...)
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
	if hostItems, ok := s.pubsubItems[host]; ok {
		if nodeItems, ok := hostItems[nodeID]; ok {
			if _, ok := nodeItems[itemID]; !ok {
				return storage.ErrNotFound
			}
			delete(nodeItems, itemID)
			return nil
		}
	}
	return storage.ErrNotFound
}

func (s *Store) Subscribe(_ context.Context, sub *storage.PubSubSubscription) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.pubsubSubscriptions[sub.Host] == nil {
		s.pubsubSubscriptions[sub.Host] = make(map[string]map[string]*storage.PubSubSubscription)
	}
	if s.pubsubSubscriptions[sub.Host][sub.NodeID] == nil {
		s.pubsubSubscriptions[sub.Host][sub.NodeID] = make(map[string]*storage.PubSubSubscription)
	}
	cp := *sub
	s.pubsubSubscriptions[sub.Host][sub.NodeID][sub.JID] = &cp
	return nil
}

func (s *Store) Unsubscribe(_ context.Context, host, nodeID, jid string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if hostSubs, ok := s.pubsubSubscriptions[host]; ok {
		if nodeSubs, ok := hostSubs[nodeID]; ok {
			delete(nodeSubs, jid)
		}
	}
	return nil
}

func (s *Store) GetSubscription(_ context.Context, host, nodeID, jid string) (*storage.PubSubSubscription, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	hostSubs, ok := s.pubsubSubscriptions[host]
	if !ok {
		return nil, storage.ErrNotFound
	}
	nodeSubs, ok := hostSubs[nodeID]
	if !ok {
		return nil, storage.ErrNotFound
	}
	sub, ok := nodeSubs[jid]
	if !ok {
		return nil, storage.ErrNotFound
	}
	cp := *sub
	return &cp, nil
}

func (s *Store) GetSubscriptions(_ context.Context, host, nodeID string) ([]*storage.PubSubSubscription, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	hostSubs := s.pubsubSubscriptions[host]
	if hostSubs == nil {
		return nil, nil
	}
	nodeSubs := hostSubs[nodeID]
	result := make([]*storage.PubSubSubscription, 0, len(nodeSubs))
	for _, sub := range nodeSubs {
		cp := *sub
		result = append(result, &cp)
	}
	return result, nil
}

func (s *Store) GetUserSubscriptions(_ context.Context, host, jid string) ([]*storage.PubSubSubscription, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	hostSubs := s.pubsubSubscriptions[host]
	if hostSubs == nil {
		return nil, nil
	}
	var result []*storage.PubSubSubscription
	for _, nodeSubs := range hostSubs {
		if sub, ok := nodeSubs[jid]; ok {
			cp := *sub
			result = append(result, &cp)
		}
	}
	return result, nil
}

// --- BookmarkStore ---

func (s *Store) SetBookmark(_ context.Context, bm *storage.Bookmark) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.bookmarks[bm.UserJID] == nil {
		s.bookmarks[bm.UserJID] = make(map[string]*storage.Bookmark)
	}
	cp := *bm
	s.bookmarks[bm.UserJID][bm.RoomJID] = &cp
	return nil
}

func (s *Store) GetBookmark(_ context.Context, userJID, roomJID string) (*storage.Bookmark, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	userBm, ok := s.bookmarks[userJID]
	if !ok {
		return nil, storage.ErrNotFound
	}
	bm, ok := userBm[roomJID]
	if !ok {
		return nil, storage.ErrNotFound
	}
	cp := *bm
	return &cp, nil
}

func (s *Store) GetBookmarks(_ context.Context, userJID string) ([]*storage.Bookmark, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	userBm := s.bookmarks[userJID]
	result := make([]*storage.Bookmark, 0, len(userBm))
	for _, bm := range userBm {
		cp := *bm
		result = append(result, &cp)
	}
	return result, nil
}

func (s *Store) DeleteBookmark(_ context.Context, userJID, roomJID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if userBm, ok := s.bookmarks[userJID]; ok {
		if _, ok := userBm[roomJID]; !ok {
			return storage.ErrNotFound
		}
		delete(userBm, roomJID)
		return nil
	}
	return storage.ErrNotFound
}
