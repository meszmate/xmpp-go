// Package redis provides a Redis storage backend for xmpp-go.
package redis

import (
	"context"
	"encoding/json"
	"time"

	"github.com/meszmate/xmpp-go/storage"

	"github.com/redis/go-redis/v9"
)

// Store implements storage.Storage using Redis.
type Store struct {
	rdb *redis.Client
}

// New creates a new Redis-backed storage.
func New(opts *redis.Options) *Store {
	return &Store{rdb: redis.NewClient(opts)}
}

func (s *Store) Init(ctx context.Context) error {
	return s.rdb.Ping(ctx).Err()
}

func (s *Store) Close() error {
	return s.rdb.Close()
}

func (s *Store) UserStore() storage.UserStore         { return s }
func (s *Store) RosterStore() storage.RosterStore     { return s }
func (s *Store) BlockingStore() storage.BlockingStore { return s }
func (s *Store) VCardStore() storage.VCardStore       { return s }
func (s *Store) OfflineStore() storage.OfflineStore   { return s }
func (s *Store) MAMStore() storage.MAMStore           { return s }
func (s *Store) MUCRoomStore() storage.MUCRoomStore   { return s }
func (s *Store) PubSubStore() storage.PubSubStore     { return s }
func (s *Store) BookmarkStore() storage.BookmarkStore { return s }

// Key helpers
func userKey(username string) string                  { return "xmpp:user:" + username }
func rosterKey(userJID string) string                 { return "xmpp:roster:" + userJID }
func rosterVerKey(userJID string) string              { return "xmpp:roster_ver:" + userJID }
func blockedKey(userJID string) string                { return "xmpp:blocked:" + userJID }
func vcardKey(userJID string) string                  { return "xmpp:vcard:" + userJID }
func offlineKey(userJID string) string                { return "xmpp:offline:" + userJID }
func mamKey(userJID string) string                    { return "xmpp:mam:" + userJID }
func mamMsgKey(userJID, id string) string             { return "xmpp:mam_msg:" + userJID + ":" + id }
func mucRoomKey(roomJID string) string                { return "xmpp:muc_room:" + roomJID }
func mucRoomsSetKey() string                          { return "xmpp:muc_rooms" }
func mucAffKey(roomJID string) string                 { return "xmpp:muc_aff:" + roomJID }
func pubsubNodeKey(host, nodeID string) string        { return "xmpp:ps_node:" + host + ":" + nodeID }
func pubsubNodesKey(host string) string               { return "xmpp:ps_nodes:" + host }
func pubsubItemKey(host, nodeID, itemID string) string { return "xmpp:ps_item:" + host + ":" + nodeID + ":" + itemID }
func pubsubItemsKey(host, nodeID string) string       { return "xmpp:ps_items:" + host + ":" + nodeID }
func pubsubSubsKey(host, nodeID string) string        { return "xmpp:ps_subs:" + host + ":" + nodeID }
func pubsubUserSubsKey(host, jid string) string       { return "xmpp:ps_usubs:" + host + ":" + jid }
func bookmarkKey(userJID string) string               { return "xmpp:bookmarks:" + userJID }

func marshal(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func unmarshal(data string, v any) error {
	return json.Unmarshal([]byte(data), v)
}

// --- UserStore ---

func (s *Store) CreateUser(ctx context.Context, user *storage.User) error {
	key := userKey(user.Username)
	ok, err := s.rdb.SetNX(ctx, key, marshal(user), 0).Result()
	if err != nil {
		return err
	}
	if !ok {
		return storage.ErrUserExists
	}
	return nil
}

func (s *Store) GetUser(ctx context.Context, username string) (*storage.User, error) {
	data, err := s.rdb.Get(ctx, userKey(username)).Result()
	if err == redis.Nil {
		return nil, storage.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	var user storage.User
	if err := unmarshal(data, &user); err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *Store) UpdateUser(ctx context.Context, user *storage.User) error {
	key := userKey(user.Username)
	exists, err := s.rdb.Exists(ctx, key).Result()
	if err != nil {
		return err
	}
	if exists == 0 {
		return storage.ErrNotFound
	}
	user.UpdatedAt = time.Now()
	return s.rdb.Set(ctx, key, marshal(user), 0).Err()
}

func (s *Store) DeleteUser(ctx context.Context, username string) error {
	n, err := s.rdb.Del(ctx, userKey(username)).Result()
	if err != nil {
		return err
	}
	if n == 0 {
		return storage.ErrNotFound
	}
	return nil
}

func (s *Store) UserExists(ctx context.Context, username string) (bool, error) {
	n, err := s.rdb.Exists(ctx, userKey(username)).Result()
	return n > 0, err
}

func (s *Store) Authenticate(ctx context.Context, username, password string) (bool, error) {
	user, err := s.GetUser(ctx, username)
	if err != nil {
		if err == storage.ErrNotFound {
			return false, storage.ErrAuthFailed
		}
		return false, err
	}
	if user.Password != password {
		return false, storage.ErrAuthFailed
	}
	return true, nil
}

// --- RosterStore ---

func (s *Store) UpsertRosterItem(ctx context.Context, item *storage.RosterItem) error {
	return s.rdb.HSet(ctx, rosterKey(item.UserJID), item.ContactJID, marshal(item)).Err()
}

func (s *Store) GetRosterItem(ctx context.Context, userJID, contactJID string) (*storage.RosterItem, error) {
	data, err := s.rdb.HGet(ctx, rosterKey(userJID), contactJID).Result()
	if err == redis.Nil {
		return nil, storage.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	var item storage.RosterItem
	if err := unmarshal(data, &item); err != nil {
		return nil, err
	}
	return &item, nil
}

func (s *Store) GetRosterItems(ctx context.Context, userJID string) ([]*storage.RosterItem, error) {
	data, err := s.rdb.HGetAll(ctx, rosterKey(userJID)).Result()
	if err != nil {
		return nil, err
	}
	items := make([]*storage.RosterItem, 0, len(data))
	for _, v := range data {
		var item storage.RosterItem
		if err := unmarshal(v, &item); err != nil {
			return nil, err
		}
		items = append(items, &item)
	}
	return items, nil
}

func (s *Store) DeleteRosterItem(ctx context.Context, userJID, contactJID string) error {
	n, err := s.rdb.HDel(ctx, rosterKey(userJID), contactJID).Result()
	if err != nil {
		return err
	}
	if n == 0 {
		return storage.ErrNotFound
	}
	return nil
}

func (s *Store) GetRosterVersion(ctx context.Context, userJID string) (string, error) {
	ver, err := s.rdb.Get(ctx, rosterVerKey(userJID)).Result()
	if err == redis.Nil {
		return "", nil
	}
	return ver, err
}

func (s *Store) SetRosterVersion(ctx context.Context, userJID, version string) error {
	return s.rdb.Set(ctx, rosterVerKey(userJID), version, 0).Err()
}

// --- BlockingStore ---

func (s *Store) BlockJID(ctx context.Context, userJID, blockedJID string) error {
	return s.rdb.SAdd(ctx, blockedKey(userJID), blockedJID).Err()
}

func (s *Store) UnblockJID(ctx context.Context, userJID, blockedJID string) error {
	return s.rdb.SRem(ctx, blockedKey(userJID), blockedJID).Err()
}

func (s *Store) IsBlocked(ctx context.Context, userJID, blockedJID string) (bool, error) {
	return s.rdb.SIsMember(ctx, blockedKey(userJID), blockedJID).Result()
}

func (s *Store) GetBlockedJIDs(ctx context.Context, userJID string) ([]string, error) {
	return s.rdb.SMembers(ctx, blockedKey(userJID)).Result()
}

// --- VCardStore ---

func (s *Store) SetVCard(ctx context.Context, userJID string, data []byte) error {
	return s.rdb.Set(ctx, vcardKey(userJID), data, 0).Err()
}

func (s *Store) GetVCard(ctx context.Context, userJID string) ([]byte, error) {
	data, err := s.rdb.Get(ctx, vcardKey(userJID)).Bytes()
	if err == redis.Nil {
		return nil, storage.ErrNotFound
	}
	return data, err
}

func (s *Store) DeleteVCard(ctx context.Context, userJID string) error {
	n, err := s.rdb.Del(ctx, vcardKey(userJID)).Result()
	if err != nil {
		return err
	}
	if n == 0 {
		return storage.ErrNotFound
	}
	return nil
}

// --- OfflineStore ---

func (s *Store) StoreOfflineMessage(ctx context.Context, msg *storage.OfflineMessage) error {
	return s.rdb.RPush(ctx, offlineKey(msg.UserJID), marshal(msg)).Err()
}

func (s *Store) GetOfflineMessages(ctx context.Context, userJID string) ([]*storage.OfflineMessage, error) {
	data, err := s.rdb.LRange(ctx, offlineKey(userJID), 0, -1).Result()
	if err != nil {
		return nil, err
	}
	msgs := make([]*storage.OfflineMessage, 0, len(data))
	for _, v := range data {
		var msg storage.OfflineMessage
		if err := unmarshal(v, &msg); err != nil {
			return nil, err
		}
		msgs = append(msgs, &msg)
	}
	return msgs, nil
}

func (s *Store) DeleteOfflineMessages(ctx context.Context, userJID string) error {
	return s.rdb.Del(ctx, offlineKey(userJID)).Err()
}

func (s *Store) CountOfflineMessages(ctx context.Context, userJID string) (int, error) {
	n, err := s.rdb.LLen(ctx, offlineKey(userJID)).Result()
	return int(n), err
}

// --- MAMStore ---

func (s *Store) ArchiveMessage(ctx context.Context, msg *storage.ArchivedMessage) error {
	if msg.CreatedAt.IsZero() {
		msg.CreatedAt = time.Now()
	}
	score := float64(msg.CreatedAt.UnixNano())
	pipe := s.rdb.Pipeline()
	pipe.ZAdd(ctx, mamKey(msg.UserJID), redis.Z{Score: score, Member: msg.ID})
	pipe.Set(ctx, mamMsgKey(msg.UserJID, msg.ID), marshal(msg), 0)
	_, err := pipe.Exec(ctx)
	return err
}

func (s *Store) QueryMessages(ctx context.Context, query *storage.MAMQuery) (*storage.MAMResult, error) {
	max := query.Max
	if max <= 0 {
		max = 100
	}

	// Get all message IDs sorted by time.
	ids, err := s.rdb.ZRangeByScore(ctx, mamKey(query.UserJID), &redis.ZRangeBy{
		Min: "-inf", Max: "+inf",
	}).Result()
	if err != nil {
		return nil, err
	}

	// Filter and collect messages.
	var msgs []*storage.ArchivedMessage
	afterIDFound := query.AfterID == ""

	for _, id := range ids {
		if !afterIDFound {
			if id == query.AfterID {
				afterIDFound = true
			}
			continue
		}
		if query.BeforeID != "" && id == query.BeforeID {
			break
		}

		data, err := s.rdb.Get(ctx, mamMsgKey(query.UserJID, id)).Result()
		if err == redis.Nil {
			continue
		}
		if err != nil {
			return nil, err
		}

		var msg storage.ArchivedMessage
		if err := unmarshal(data, &msg); err != nil {
			return nil, err
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

		msgs = append(msgs, &msg)
		if len(msgs) > max {
			break
		}
	}

	complete := len(msgs) <= max
	if len(msgs) > max {
		msgs = msgs[:max]
	}

	result := &storage.MAMResult{
		Messages: msgs, Complete: complete, Count: len(msgs),
	}
	if len(msgs) > 0 {
		result.First = msgs[0].ID
		result.Last = msgs[len(msgs)-1].ID
	}
	return result, nil
}

func (s *Store) DeleteMessageArchive(ctx context.Context, userJID string) error {
	ids, err := s.rdb.ZRange(ctx, mamKey(userJID), 0, -1).Result()
	if err != nil {
		return err
	}
	pipe := s.rdb.Pipeline()
	for _, id := range ids {
		pipe.Del(ctx, mamMsgKey(userJID, id))
	}
	pipe.Del(ctx, mamKey(userJID))
	_, err = pipe.Exec(ctx)
	return err
}

// --- MUCRoomStore ---

func (s *Store) CreateRoom(ctx context.Context, room *storage.MUCRoom) error {
	ok, err := s.rdb.SetNX(ctx, mucRoomKey(room.RoomJID), marshal(room), 0).Result()
	if err != nil {
		return err
	}
	if !ok {
		return storage.ErrUserExists
	}
	s.rdb.SAdd(ctx, mucRoomsSetKey(), room.RoomJID)
	return nil
}

func (s *Store) GetRoom(ctx context.Context, roomJID string) (*storage.MUCRoom, error) {
	data, err := s.rdb.Get(ctx, mucRoomKey(roomJID)).Result()
	if err == redis.Nil {
		return nil, storage.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	var room storage.MUCRoom
	if err := unmarshal(data, &room); err != nil {
		return nil, err
	}
	return &room, nil
}

func (s *Store) UpdateRoom(ctx context.Context, room *storage.MUCRoom) error {
	key := mucRoomKey(room.RoomJID)
	exists, err := s.rdb.Exists(ctx, key).Result()
	if err != nil {
		return err
	}
	if exists == 0 {
		return storage.ErrNotFound
	}
	return s.rdb.Set(ctx, key, marshal(room), 0).Err()
}

func (s *Store) DeleteRoom(ctx context.Context, roomJID string) error {
	n, err := s.rdb.Del(ctx, mucRoomKey(roomJID)).Result()
	if err != nil {
		return err
	}
	if n == 0 {
		return storage.ErrNotFound
	}
	s.rdb.SRem(ctx, mucRoomsSetKey(), roomJID)
	s.rdb.Del(ctx, mucAffKey(roomJID))
	return nil
}

func (s *Store) ListRooms(ctx context.Context) ([]*storage.MUCRoom, error) {
	jids, err := s.rdb.SMembers(ctx, mucRoomsSetKey()).Result()
	if err != nil {
		return nil, err
	}
	var rooms []*storage.MUCRoom
	for _, jid := range jids {
		room, err := s.GetRoom(ctx, jid)
		if err != nil {
			continue
		}
		rooms = append(rooms, room)
	}
	return rooms, nil
}

func (s *Store) SetAffiliation(ctx context.Context, aff *storage.MUCAffiliation) error {
	return s.rdb.HSet(ctx, mucAffKey(aff.RoomJID), aff.UserJID, marshal(aff)).Err()
}

func (s *Store) GetAffiliation(ctx context.Context, roomJID, userJID string) (*storage.MUCAffiliation, error) {
	data, err := s.rdb.HGet(ctx, mucAffKey(roomJID), userJID).Result()
	if err == redis.Nil {
		return nil, storage.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	var aff storage.MUCAffiliation
	if err := unmarshal(data, &aff); err != nil {
		return nil, err
	}
	return &aff, nil
}

func (s *Store) GetAffiliations(ctx context.Context, roomJID string) ([]*storage.MUCAffiliation, error) {
	data, err := s.rdb.HGetAll(ctx, mucAffKey(roomJID)).Result()
	if err != nil {
		return nil, err
	}
	affs := make([]*storage.MUCAffiliation, 0, len(data))
	for _, v := range data {
		var aff storage.MUCAffiliation
		if err := unmarshal(v, &aff); err != nil {
			return nil, err
		}
		affs = append(affs, &aff)
	}
	return affs, nil
}

func (s *Store) RemoveAffiliation(ctx context.Context, roomJID, userJID string) error {
	return s.rdb.HDel(ctx, mucAffKey(roomJID), userJID).Err()
}

// --- PubSubStore ---

func (s *Store) CreateNode(ctx context.Context, node *storage.PubSubNode) error {
	key := pubsubNodeKey(node.Host, node.NodeID)
	ok, err := s.rdb.SetNX(ctx, key, marshal(node), 0).Result()
	if err != nil {
		return err
	}
	if !ok {
		return storage.ErrUserExists
	}
	s.rdb.SAdd(ctx, pubsubNodesKey(node.Host), node.NodeID)
	return nil
}

func (s *Store) GetNode(ctx context.Context, host, nodeID string) (*storage.PubSubNode, error) {
	data, err := s.rdb.Get(ctx, pubsubNodeKey(host, nodeID)).Result()
	if err == redis.Nil {
		return nil, storage.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	var node storage.PubSubNode
	if err := unmarshal(data, &node); err != nil {
		return nil, err
	}
	return &node, nil
}

func (s *Store) DeleteNode(ctx context.Context, host, nodeID string) error {
	n, err := s.rdb.Del(ctx, pubsubNodeKey(host, nodeID)).Result()
	if err != nil {
		return err
	}
	if n == 0 {
		return storage.ErrNotFound
	}
	s.rdb.SRem(ctx, pubsubNodesKey(host), nodeID)
	// Clean up items
	itemIDs, _ := s.rdb.SMembers(ctx, pubsubItemsKey(host, nodeID)).Result()
	pipe := s.rdb.Pipeline()
	for _, itemID := range itemIDs {
		pipe.Del(ctx, pubsubItemKey(host, nodeID, itemID))
	}
	pipe.Del(ctx, pubsubItemsKey(host, nodeID))
	// Clean up subscriptions
	subJIDs, _ := s.rdb.HKeys(ctx, pubsubSubsKey(host, nodeID)).Result()
	for _, jid := range subJIDs {
		pipe.SRem(ctx, pubsubUserSubsKey(host, jid), nodeID)
	}
	pipe.Del(ctx, pubsubSubsKey(host, nodeID))
	_, err = pipe.Exec(ctx)
	return err
}

func (s *Store) ListNodes(ctx context.Context, host string) ([]*storage.PubSubNode, error) {
	nodeIDs, err := s.rdb.SMembers(ctx, pubsubNodesKey(host)).Result()
	if err != nil {
		return nil, err
	}
	var nodes []*storage.PubSubNode
	for _, nodeID := range nodeIDs {
		node, err := s.GetNode(ctx, host, nodeID)
		if err != nil {
			continue
		}
		nodes = append(nodes, node)
	}
	return nodes, nil
}

func (s *Store) UpsertItem(ctx context.Context, item *storage.PubSubItem) error {
	if item.CreatedAt.IsZero() {
		item.CreatedAt = time.Now()
	}
	pipe := s.rdb.Pipeline()
	pipe.Set(ctx, pubsubItemKey(item.Host, item.NodeID, item.ItemID), marshal(item), 0)
	pipe.SAdd(ctx, pubsubItemsKey(item.Host, item.NodeID), item.ItemID)
	_, err := pipe.Exec(ctx)
	return err
}

func (s *Store) GetItem(ctx context.Context, host, nodeID, itemID string) (*storage.PubSubItem, error) {
	data, err := s.rdb.Get(ctx, pubsubItemKey(host, nodeID, itemID)).Result()
	if err == redis.Nil {
		return nil, storage.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	var item storage.PubSubItem
	if err := unmarshal(data, &item); err != nil {
		return nil, err
	}
	return &item, nil
}

func (s *Store) GetItems(ctx context.Context, host, nodeID string) ([]*storage.PubSubItem, error) {
	itemIDs, err := s.rdb.SMembers(ctx, pubsubItemsKey(host, nodeID)).Result()
	if err != nil {
		return nil, err
	}
	var items []*storage.PubSubItem
	for _, itemID := range itemIDs {
		item, err := s.GetItem(ctx, host, nodeID, itemID)
		if err != nil {
			continue
		}
		items = append(items, item)
	}
	return items, nil
}

func (s *Store) DeleteItem(ctx context.Context, host, nodeID, itemID string) error {
	n, err := s.rdb.Del(ctx, pubsubItemKey(host, nodeID, itemID)).Result()
	if err != nil {
		return err
	}
	if n == 0 {
		return storage.ErrNotFound
	}
	s.rdb.SRem(ctx, pubsubItemsKey(host, nodeID), itemID)
	return nil
}

func (s *Store) Subscribe(ctx context.Context, sub *storage.PubSubSubscription) error {
	pipe := s.rdb.Pipeline()
	pipe.HSet(ctx, pubsubSubsKey(sub.Host, sub.NodeID), sub.JID, marshal(sub))
	pipe.SAdd(ctx, pubsubUserSubsKey(sub.Host, sub.JID), sub.NodeID)
	_, err := pipe.Exec(ctx)
	return err
}

func (s *Store) Unsubscribe(ctx context.Context, host, nodeID, jid string) error {
	pipe := s.rdb.Pipeline()
	pipe.HDel(ctx, pubsubSubsKey(host, nodeID), jid)
	pipe.SRem(ctx, pubsubUserSubsKey(host, jid), nodeID)
	_, err := pipe.Exec(ctx)
	return err
}

func (s *Store) GetSubscription(ctx context.Context, host, nodeID, jid string) (*storage.PubSubSubscription, error) {
	data, err := s.rdb.HGet(ctx, pubsubSubsKey(host, nodeID), jid).Result()
	if err == redis.Nil {
		return nil, storage.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	var sub storage.PubSubSubscription
	if err := unmarshal(data, &sub); err != nil {
		return nil, err
	}
	return &sub, nil
}

func (s *Store) GetSubscriptions(ctx context.Context, host, nodeID string) ([]*storage.PubSubSubscription, error) {
	data, err := s.rdb.HGetAll(ctx, pubsubSubsKey(host, nodeID)).Result()
	if err != nil {
		return nil, err
	}
	subs := make([]*storage.PubSubSubscription, 0, len(data))
	for _, v := range data {
		var sub storage.PubSubSubscription
		if err := unmarshal(v, &sub); err != nil {
			return nil, err
		}
		subs = append(subs, &sub)
	}
	return subs, nil
}

func (s *Store) GetUserSubscriptions(ctx context.Context, host, jid string) ([]*storage.PubSubSubscription, error) {
	nodeIDs, err := s.rdb.SMembers(ctx, pubsubUserSubsKey(host, jid)).Result()
	if err != nil {
		return nil, err
	}
	var subs []*storage.PubSubSubscription
	for _, nodeID := range nodeIDs {
		sub, err := s.GetSubscription(ctx, host, nodeID, jid)
		if err != nil {
			continue
		}
		subs = append(subs, sub)
	}
	return subs, nil
}

// --- BookmarkStore ---

func (s *Store) SetBookmark(ctx context.Context, bm *storage.Bookmark) error {
	return s.rdb.HSet(ctx, bookmarkKey(bm.UserJID), bm.RoomJID, marshal(bm)).Err()
}

func (s *Store) GetBookmark(ctx context.Context, userJID, roomJID string) (*storage.Bookmark, error) {
	data, err := s.rdb.HGet(ctx, bookmarkKey(userJID), roomJID).Result()
	if err == redis.Nil {
		return nil, storage.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	var bm storage.Bookmark
	if err := unmarshal(data, &bm); err != nil {
		return nil, err
	}
	return &bm, nil
}

func (s *Store) GetBookmarks(ctx context.Context, userJID string) ([]*storage.Bookmark, error) {
	data, err := s.rdb.HGetAll(ctx, bookmarkKey(userJID)).Result()
	if err != nil {
		return nil, err
	}
	bms := make([]*storage.Bookmark, 0, len(data))
	for _, v := range data {
		var bm storage.Bookmark
		if err := unmarshal(v, &bm); err != nil {
			return nil, err
		}
		bms = append(bms, &bm)
	}
	return bms, nil
}

func (s *Store) DeleteBookmark(ctx context.Context, userJID, roomJID string) error {
	n, err := s.rdb.HDel(ctx, bookmarkKey(userJID), roomJID).Result()
	if err != nil {
		return err
	}
	if n == 0 {
		return storage.ErrNotFound
	}
	return nil
}

