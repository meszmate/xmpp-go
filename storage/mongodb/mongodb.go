// Package mongodb provides a MongoDB storage backend for xmpp-go.
package mongodb

import (
	"context"
	"fmt"
	"time"

	"github.com/meszmate/xmpp-go/storage"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Store implements storage.Storage using MongoDB.
type Store struct {
	client *mongo.Client
	db     *mongo.Database
}

// New creates a new MongoDB-backed storage.
func New(uri, database string) (*Store, error) {
	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("mongodb: connect: %w", err)
	}
	return &Store{client: client, db: client.Database(database)}, nil
}

func (s *Store) Init(ctx context.Context) error {
	// Create indexes.
	indexes := []struct {
		collection string
		keys       bson.D
		unique     bool
	}{
		{"users", bson.D{{Key: "username", Value: 1}}, true},
		{"roster_items", bson.D{{Key: "user_jid", Value: 1}, {Key: "contact_jid", Value: 1}}, true},
		{"blocked_jids", bson.D{{Key: "user_jid", Value: 1}, {Key: "blocked_jid", Value: 1}}, true},
		{"vcards", bson.D{{Key: "user_jid", Value: 1}}, true},
		{"offline_messages", bson.D{{Key: "user_jid", Value: 1}}, false},
		{"mam_messages", bson.D{{Key: "user_jid", Value: 1}}, false},
		{"mam_messages", bson.D{{Key: "user_jid", Value: 1}, {Key: "with_jid", Value: 1}}, false},
		{"muc_rooms", bson.D{{Key: "room_jid", Value: 1}}, true},
		{"muc_affiliations", bson.D{{Key: "room_jid", Value: 1}, {Key: "user_jid", Value: 1}}, true},
		{"pubsub_nodes", bson.D{{Key: "host", Value: 1}, {Key: "node_id", Value: 1}}, true},
		{"pubsub_items", bson.D{{Key: "host", Value: 1}, {Key: "node_id", Value: 1}, {Key: "item_id", Value: 1}}, true},
		{"pubsub_subscriptions", bson.D{{Key: "host", Value: 1}, {Key: "node_id", Value: 1}, {Key: "jid", Value: 1}}, true},
		{"bookmarks", bson.D{{Key: "user_jid", Value: 1}, {Key: "room_jid", Value: 1}}, true},
	}
	for _, idx := range indexes {
		opts := options.Index().SetUnique(idx.unique)
		_, err := s.db.Collection(idx.collection).Indexes().CreateOne(ctx, mongo.IndexModel{
			Keys:    idx.keys,
			Options: opts,
		})
		if err != nil {
			return fmt.Errorf("mongodb: create index on %s: %w", idx.collection, err)
		}
	}
	return nil
}

func (s *Store) Close() error {
	return s.client.Disconnect(context.Background())
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

func (s *Store) col(name string) *mongo.Collection { return s.db.Collection(name) }

// --- UserStore ---

type userDoc struct {
	Username   string    `bson:"username"`
	Password   string    `bson:"password"`
	Salt       string    `bson:"salt"`
	Iterations int       `bson:"iterations"`
	ServerKey  string    `bson:"server_key"`
	StoredKey  string    `bson:"stored_key"`
	CreatedAt  time.Time `bson:"created_at"`
	UpdatedAt  time.Time `bson:"updated_at"`
}

func (s *Store) CreateUser(ctx context.Context, user *storage.User) error {
	now := time.Now()
	doc := userDoc{
		Username: user.Username, Password: user.Password,
		Salt: user.Salt, Iterations: user.Iterations,
		ServerKey: user.ServerKey, StoredKey: user.StoredKey,
		CreatedAt: now, UpdatedAt: now,
	}
	_, err := s.col("users").InsertOne(ctx, doc)
	if mongo.IsDuplicateKeyError(err) {
		return storage.ErrUserExists
	}
	return err
}

func (s *Store) GetUser(ctx context.Context, username string) (*storage.User, error) {
	var doc userDoc
	err := s.col("users").FindOne(ctx, bson.M{"username": username}).Decode(&doc)
	if err == mongo.ErrNoDocuments {
		return nil, storage.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &storage.User{
		Username: doc.Username, Password: doc.Password,
		Salt: doc.Salt, Iterations: doc.Iterations,
		ServerKey: doc.ServerKey, StoredKey: doc.StoredKey,
		CreatedAt: doc.CreatedAt, UpdatedAt: doc.UpdatedAt,
	}, nil
}

func (s *Store) UpdateUser(ctx context.Context, user *storage.User) error {
	res, err := s.col("users").UpdateOne(ctx,
		bson.M{"username": user.Username},
		bson.M{"$set": bson.M{
			"password": user.Password, "salt": user.Salt,
			"iterations": user.Iterations, "server_key": user.ServerKey,
			"stored_key": user.StoredKey, "updated_at": time.Now(),
		}},
	)
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return storage.ErrNotFound
	}
	return nil
}

func (s *Store) DeleteUser(ctx context.Context, username string) error {
	res, err := s.col("users").DeleteOne(ctx, bson.M{"username": username})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return storage.ErrNotFound
	}
	return nil
}

func (s *Store) UserExists(ctx context.Context, username string) (bool, error) {
	count, err := s.col("users").CountDocuments(ctx, bson.M{"username": username})
	return count > 0, err
}

func (s *Store) Authenticate(ctx context.Context, username, password string) (bool, error) {
	var doc userDoc
	err := s.col("users").FindOne(ctx, bson.M{"username": username}).Decode(&doc)
	if err == mongo.ErrNoDocuments {
		return false, storage.ErrAuthFailed
	}
	if err != nil {
		return false, err
	}
	if doc.Password != password {
		return false, storage.ErrAuthFailed
	}
	return true, nil
}

// --- RosterStore ---

type rosterDoc struct {
	UserJID      string   `bson:"user_jid"`
	ContactJID   string   `bson:"contact_jid"`
	Name         string   `bson:"name"`
	Subscription string   `bson:"subscription"`
	Ask          string   `bson:"ask"`
	Groups       []string `bson:"groups"`
}

func (s *Store) UpsertRosterItem(ctx context.Context, item *storage.RosterItem) error {
	_, err := s.col("roster_items").UpdateOne(ctx,
		bson.M{"user_jid": item.UserJID, "contact_jid": item.ContactJID},
		bson.M{"$set": rosterDoc{
			UserJID: item.UserJID, ContactJID: item.ContactJID,
			Name: item.Name, Subscription: item.Subscription,
			Ask: item.Ask, Groups: item.Groups,
		}},
		options.UpdateOne().SetUpsert(true),
	)
	return err
}

func (s *Store) GetRosterItem(ctx context.Context, userJID, contactJID string) (*storage.RosterItem, error) {
	var doc rosterDoc
	err := s.col("roster_items").FindOne(ctx, bson.M{"user_jid": userJID, "contact_jid": contactJID}).Decode(&doc)
	if err == mongo.ErrNoDocuments {
		return nil, storage.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &storage.RosterItem{
		UserJID: doc.UserJID, ContactJID: doc.ContactJID,
		Name: doc.Name, Subscription: doc.Subscription,
		Ask: doc.Ask, Groups: doc.Groups,
	}, nil
}

func (s *Store) GetRosterItems(ctx context.Context, userJID string) ([]*storage.RosterItem, error) {
	cursor, err := s.col("roster_items").Find(ctx, bson.M{"user_jid": userJID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var items []*storage.RosterItem
	for cursor.Next(ctx) {
		var doc rosterDoc
		if err := cursor.Decode(&doc); err != nil {
			return nil, err
		}
		items = append(items, &storage.RosterItem{
			UserJID: doc.UserJID, ContactJID: doc.ContactJID,
			Name: doc.Name, Subscription: doc.Subscription,
			Ask: doc.Ask, Groups: doc.Groups,
		})
	}
	return items, cursor.Err()
}

func (s *Store) DeleteRosterItem(ctx context.Context, userJID, contactJID string) error {
	res, err := s.col("roster_items").DeleteOne(ctx, bson.M{"user_jid": userJID, "contact_jid": contactJID})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return storage.ErrNotFound
	}
	return nil
}

type rosterVersionDoc struct {
	UserJID string `bson:"user_jid"`
	Version string `bson:"version"`
}

func (s *Store) GetRosterVersion(ctx context.Context, userJID string) (string, error) {
	var doc rosterVersionDoc
	err := s.col("roster_versions").FindOne(ctx, bson.M{"user_jid": userJID}).Decode(&doc)
	if err == mongo.ErrNoDocuments {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return doc.Version, nil
}

func (s *Store) SetRosterVersion(ctx context.Context, userJID, version string) error {
	_, err := s.col("roster_versions").UpdateOne(ctx,
		bson.M{"user_jid": userJID},
		bson.M{"$set": bson.M{"version": version}},
		options.UpdateOne().SetUpsert(true),
	)
	return err
}

// --- BlockingStore ---

func (s *Store) BlockJID(ctx context.Context, userJID, blockedJID string) error {
	_, err := s.col("blocked_jids").UpdateOne(ctx,
		bson.M{"user_jid": userJID, "blocked_jid": blockedJID},
		bson.M{"$set": bson.M{"user_jid": userJID, "blocked_jid": blockedJID}},
		options.UpdateOne().SetUpsert(true),
	)
	return err
}

func (s *Store) UnblockJID(ctx context.Context, userJID, blockedJID string) error {
	_, err := s.col("blocked_jids").DeleteOne(ctx, bson.M{"user_jid": userJID, "blocked_jid": blockedJID})
	return err
}

func (s *Store) IsBlocked(ctx context.Context, userJID, blockedJID string) (bool, error) {
	count, err := s.col("blocked_jids").CountDocuments(ctx, bson.M{"user_jid": userJID, "blocked_jid": blockedJID})
	return count > 0, err
}

func (s *Store) GetBlockedJIDs(ctx context.Context, userJID string) ([]string, error) {
	cursor, err := s.col("blocked_jids").Find(ctx, bson.M{"user_jid": userJID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var jids []string
	for cursor.Next(ctx) {
		var doc struct {
			BlockedJID string `bson:"blocked_jid"`
		}
		if err := cursor.Decode(&doc); err != nil {
			return nil, err
		}
		jids = append(jids, doc.BlockedJID)
	}
	return jids, cursor.Err()
}

// --- VCardStore ---

func (s *Store) SetVCard(ctx context.Context, userJID string, data []byte) error {
	_, err := s.col("vcards").UpdateOne(ctx,
		bson.M{"user_jid": userJID},
		bson.M{"$set": bson.M{"user_jid": userJID, "data": data}},
		options.UpdateOne().SetUpsert(true),
	)
	return err
}

func (s *Store) GetVCard(ctx context.Context, userJID string) ([]byte, error) {
	var doc struct {
		Data []byte `bson:"data"`
	}
	err := s.col("vcards").FindOne(ctx, bson.M{"user_jid": userJID}).Decode(&doc)
	if err == mongo.ErrNoDocuments {
		return nil, storage.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return doc.Data, nil
}

func (s *Store) DeleteVCard(ctx context.Context, userJID string) error {
	res, err := s.col("vcards").DeleteOne(ctx, bson.M{"user_jid": userJID})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return storage.ErrNotFound
	}
	return nil
}

// --- OfflineStore ---

type offlineDoc struct {
	ID        string    `bson:"id"`
	UserJID   string    `bson:"user_jid"`
	FromJID   string    `bson:"from_jid"`
	Data      []byte    `bson:"data"`
	CreatedAt time.Time `bson:"created_at"`
}

func (s *Store) StoreOfflineMessage(ctx context.Context, msg *storage.OfflineMessage) error {
	createdAt := msg.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now()
	}
	_, err := s.col("offline_messages").InsertOne(ctx, offlineDoc{
		ID: msg.ID, UserJID: msg.UserJID, FromJID: msg.FromJID,
		Data: msg.Data, CreatedAt: createdAt,
	})
	return err
}

func (s *Store) GetOfflineMessages(ctx context.Context, userJID string) ([]*storage.OfflineMessage, error) {
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: 1}})
	cursor, err := s.col("offline_messages").Find(ctx, bson.M{"user_jid": userJID}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var msgs []*storage.OfflineMessage
	for cursor.Next(ctx) {
		var doc offlineDoc
		if err := cursor.Decode(&doc); err != nil {
			return nil, err
		}
		msgs = append(msgs, &storage.OfflineMessage{
			ID: doc.ID, UserJID: doc.UserJID, FromJID: doc.FromJID,
			Data: doc.Data, CreatedAt: doc.CreatedAt,
		})
	}
	return msgs, cursor.Err()
}

func (s *Store) DeleteOfflineMessages(ctx context.Context, userJID string) error {
	_, err := s.col("offline_messages").DeleteMany(ctx, bson.M{"user_jid": userJID})
	return err
}

func (s *Store) CountOfflineMessages(ctx context.Context, userJID string) (int, error) {
	count, err := s.col("offline_messages").CountDocuments(ctx, bson.M{"user_jid": userJID})
	return int(count), err
}

// --- MAMStore ---

type mamDoc struct {
	ID        string    `bson:"id"`
	UserJID   string    `bson:"user_jid"`
	WithJID   string    `bson:"with_jid"`
	FromJID   string    `bson:"from_jid"`
	Data      []byte    `bson:"data"`
	CreatedAt time.Time `bson:"created_at"`
}

func (s *Store) ArchiveMessage(ctx context.Context, msg *storage.ArchivedMessage) error {
	createdAt := msg.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now()
	}
	_, err := s.col("mam_messages").InsertOne(ctx, mamDoc{
		ID: msg.ID, UserJID: msg.UserJID, WithJID: msg.WithJID,
		FromJID: msg.FromJID, Data: msg.Data, CreatedAt: createdAt,
	})
	return err
}

func (s *Store) QueryMessages(ctx context.Context, query *storage.MAMQuery) (*storage.MAMResult, error) {
	filter := bson.M{"user_jid": query.UserJID}
	if query.WithJID != "" {
		filter["with_jid"] = query.WithJID
	}
	if !query.Start.IsZero() {
		filter["created_at"] = bson.M{"$gte": query.Start}
	}
	if !query.End.IsZero() {
		if existing, ok := filter["created_at"]; ok {
			existing.(bson.M)["$lte"] = query.End
		} else {
			filter["created_at"] = bson.M{"$lte": query.End}
		}
	}
	if query.AfterID != "" {
		filter["id"] = bson.M{"$gt": query.AfterID}
	}
	if query.BeforeID != "" {
		if existing, ok := filter["id"]; ok {
			existing.(bson.M)["$lt"] = query.BeforeID
		} else {
			filter["id"] = bson.M{"$lt": query.BeforeID}
		}
	}

	max := query.Max
	if max <= 0 {
		max = 100
	}

	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: 1}}).SetLimit(int64(max + 1))
	cursor, err := s.col("mam_messages").Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var msgs []*storage.ArchivedMessage
	for cursor.Next(ctx) {
		var doc mamDoc
		if err := cursor.Decode(&doc); err != nil {
			return nil, err
		}
		msgs = append(msgs, &storage.ArchivedMessage{
			ID: doc.ID, UserJID: doc.UserJID, WithJID: doc.WithJID,
			FromJID: doc.FromJID, Data: doc.Data, CreatedAt: doc.CreatedAt,
		})
	}
	if err := cursor.Err(); err != nil {
		return nil, err
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
	_, err := s.col("mam_messages").DeleteMany(ctx, bson.M{"user_jid": userJID})
	return err
}

// --- MUCRoomStore ---

type mucRoomDoc struct {
	RoomJID     string `bson:"room_jid"`
	Name        string `bson:"name"`
	Description string `bson:"description"`
	Subject     string `bson:"subject"`
	Password    string `bson:"password"`
	Public      bool   `bson:"is_public"`
	Persistent  bool   `bson:"is_persistent"`
	MaxUsers    int    `bson:"max_users"`
}

func (s *Store) CreateRoom(ctx context.Context, room *storage.MUCRoom) error {
	_, err := s.col("muc_rooms").InsertOne(ctx, mucRoomDoc{
		RoomJID: room.RoomJID, Name: room.Name, Description: room.Description,
		Subject: room.Subject, Password: room.Password,
		Public: room.Public, Persistent: room.Persistent, MaxUsers: room.MaxUsers,
	})
	if mongo.IsDuplicateKeyError(err) {
		return storage.ErrUserExists
	}
	return err
}

func (s *Store) GetRoom(ctx context.Context, roomJID string) (*storage.MUCRoom, error) {
	var doc mucRoomDoc
	err := s.col("muc_rooms").FindOne(ctx, bson.M{"room_jid": roomJID}).Decode(&doc)
	if err == mongo.ErrNoDocuments {
		return nil, storage.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &storage.MUCRoom{
		RoomJID: doc.RoomJID, Name: doc.Name, Description: doc.Description,
		Subject: doc.Subject, Password: doc.Password,
		Public: doc.Public, Persistent: doc.Persistent, MaxUsers: doc.MaxUsers,
	}, nil
}

func (s *Store) UpdateRoom(ctx context.Context, room *storage.MUCRoom) error {
	res, err := s.col("muc_rooms").UpdateOne(ctx,
		bson.M{"room_jid": room.RoomJID},
		bson.M{"$set": bson.M{
			"name": room.Name, "description": room.Description,
			"subject": room.Subject, "password": room.Password,
			"is_public": room.Public, "is_persistent": room.Persistent,
			"max_users": room.MaxUsers,
		}},
	)
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return storage.ErrNotFound
	}
	return nil
}

func (s *Store) DeleteRoom(ctx context.Context, roomJID string) error {
	res, err := s.col("muc_rooms").DeleteOne(ctx, bson.M{"room_jid": roomJID})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return storage.ErrNotFound
	}
	_, _ = s.col("muc_affiliations").DeleteMany(ctx, bson.M{"room_jid": roomJID})
	return nil
}

func (s *Store) ListRooms(ctx context.Context) ([]*storage.MUCRoom, error) {
	cursor, err := s.col("muc_rooms").Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var rooms []*storage.MUCRoom
	for cursor.Next(ctx) {
		var doc mucRoomDoc
		if err := cursor.Decode(&doc); err != nil {
			return nil, err
		}
		rooms = append(rooms, &storage.MUCRoom{
			RoomJID: doc.RoomJID, Name: doc.Name, Description: doc.Description,
			Subject: doc.Subject, Password: doc.Password,
			Public: doc.Public, Persistent: doc.Persistent, MaxUsers: doc.MaxUsers,
		})
	}
	return rooms, cursor.Err()
}

type mucAffDoc struct {
	RoomJID     string `bson:"room_jid"`
	UserJID     string `bson:"user_jid"`
	Affiliation string `bson:"affiliation"`
	Reason      string `bson:"reason"`
}

func (s *Store) SetAffiliation(ctx context.Context, aff *storage.MUCAffiliation) error {
	_, err := s.col("muc_affiliations").UpdateOne(ctx,
		bson.M{"room_jid": aff.RoomJID, "user_jid": aff.UserJID},
		bson.M{"$set": mucAffDoc{
			RoomJID: aff.RoomJID, UserJID: aff.UserJID,
			Affiliation: aff.Affiliation, Reason: aff.Reason,
		}},
		options.UpdateOne().SetUpsert(true),
	)
	return err
}

func (s *Store) GetAffiliation(ctx context.Context, roomJID, userJID string) (*storage.MUCAffiliation, error) {
	var doc mucAffDoc
	err := s.col("muc_affiliations").FindOne(ctx, bson.M{"room_jid": roomJID, "user_jid": userJID}).Decode(&doc)
	if err == mongo.ErrNoDocuments {
		return nil, storage.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &storage.MUCAffiliation{
		RoomJID: doc.RoomJID, UserJID: doc.UserJID,
		Affiliation: doc.Affiliation, Reason: doc.Reason,
	}, nil
}

func (s *Store) GetAffiliations(ctx context.Context, roomJID string) ([]*storage.MUCAffiliation, error) {
	cursor, err := s.col("muc_affiliations").Find(ctx, bson.M{"room_jid": roomJID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var affs []*storage.MUCAffiliation
	for cursor.Next(ctx) {
		var doc mucAffDoc
		if err := cursor.Decode(&doc); err != nil {
			return nil, err
		}
		affs = append(affs, &storage.MUCAffiliation{
			RoomJID: doc.RoomJID, UserJID: doc.UserJID,
			Affiliation: doc.Affiliation, Reason: doc.Reason,
		})
	}
	return affs, cursor.Err()
}

func (s *Store) RemoveAffiliation(ctx context.Context, roomJID, userJID string) error {
	_, err := s.col("muc_affiliations").DeleteOne(ctx, bson.M{"room_jid": roomJID, "user_jid": userJID})
	return err
}

// --- PubSubStore ---

type pubsubNodeDoc struct {
	Host    string `bson:"host"`
	NodeID  string `bson:"node_id"`
	Name    string `bson:"name"`
	Type    string `bson:"type"`
	Creator string `bson:"creator"`
}

type pubsubItemDoc struct {
	Host      string    `bson:"host"`
	NodeID    string    `bson:"node_id"`
	ItemID    string    `bson:"item_id"`
	Publisher string    `bson:"publisher"`
	Payload   []byte    `bson:"payload"`
	CreatedAt time.Time `bson:"created_at"`
}

type pubsubSubDoc struct {
	Host   string `bson:"host"`
	NodeID string `bson:"node_id"`
	JID    string `bson:"jid"`
	SubID  string `bson:"sub_id"`
	State  string `bson:"state"`
}

func (s *Store) CreateNode(ctx context.Context, node *storage.PubSubNode) error {
	_, err := s.col("pubsub_nodes").InsertOne(ctx, pubsubNodeDoc{
		Host: node.Host, NodeID: node.NodeID, Name: node.Name,
		Type: node.Type, Creator: node.Creator,
	})
	if mongo.IsDuplicateKeyError(err) {
		return storage.ErrUserExists
	}
	return err
}

func (s *Store) GetNode(ctx context.Context, host, nodeID string) (*storage.PubSubNode, error) {
	var doc pubsubNodeDoc
	err := s.col("pubsub_nodes").FindOne(ctx, bson.M{"host": host, "node_id": nodeID}).Decode(&doc)
	if err == mongo.ErrNoDocuments {
		return nil, storage.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &storage.PubSubNode{
		Host: doc.Host, NodeID: doc.NodeID, Name: doc.Name,
		Type: doc.Type, Creator: doc.Creator,
	}, nil
}

func (s *Store) DeleteNode(ctx context.Context, host, nodeID string) error {
	res, err := s.col("pubsub_nodes").DeleteOne(ctx, bson.M{"host": host, "node_id": nodeID})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return storage.ErrNotFound
	}
	_, _ = s.col("pubsub_items").DeleteMany(ctx, bson.M{"host": host, "node_id": nodeID})
	_, _ = s.col("pubsub_subscriptions").DeleteMany(ctx, bson.M{"host": host, "node_id": nodeID})
	return nil
}

func (s *Store) ListNodes(ctx context.Context, host string) ([]*storage.PubSubNode, error) {
	cursor, err := s.col("pubsub_nodes").Find(ctx, bson.M{"host": host})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var nodes []*storage.PubSubNode
	for cursor.Next(ctx) {
		var doc pubsubNodeDoc
		if err := cursor.Decode(&doc); err != nil {
			return nil, err
		}
		nodes = append(nodes, &storage.PubSubNode{
			Host: doc.Host, NodeID: doc.NodeID, Name: doc.Name,
			Type: doc.Type, Creator: doc.Creator,
		})
	}
	return nodes, cursor.Err()
}

func (s *Store) UpsertItem(ctx context.Context, item *storage.PubSubItem) error {
	createdAt := item.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now()
	}
	_, err := s.col("pubsub_items").UpdateOne(ctx,
		bson.M{"host": item.Host, "node_id": item.NodeID, "item_id": item.ItemID},
		bson.M{"$set": pubsubItemDoc{
			Host: item.Host, NodeID: item.NodeID, ItemID: item.ItemID,
			Publisher: item.Publisher, Payload: item.Payload, CreatedAt: createdAt,
		}},
		options.UpdateOne().SetUpsert(true),
	)
	return err
}

func (s *Store) GetItem(ctx context.Context, host, nodeID, itemID string) (*storage.PubSubItem, error) {
	var doc pubsubItemDoc
	err := s.col("pubsub_items").FindOne(ctx, bson.M{"host": host, "node_id": nodeID, "item_id": itemID}).Decode(&doc)
	if err == mongo.ErrNoDocuments {
		return nil, storage.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &storage.PubSubItem{
		Host: doc.Host, NodeID: doc.NodeID, ItemID: doc.ItemID,
		Publisher: doc.Publisher, Payload: doc.Payload, CreatedAt: doc.CreatedAt,
	}, nil
}

func (s *Store) GetItems(ctx context.Context, host, nodeID string) ([]*storage.PubSubItem, error) {
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: 1}})
	cursor, err := s.col("pubsub_items").Find(ctx, bson.M{"host": host, "node_id": nodeID}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var items []*storage.PubSubItem
	for cursor.Next(ctx) {
		var doc pubsubItemDoc
		if err := cursor.Decode(&doc); err != nil {
			return nil, err
		}
		items = append(items, &storage.PubSubItem{
			Host: doc.Host, NodeID: doc.NodeID, ItemID: doc.ItemID,
			Publisher: doc.Publisher, Payload: doc.Payload, CreatedAt: doc.CreatedAt,
		})
	}
	return items, cursor.Err()
}

func (s *Store) DeleteItem(ctx context.Context, host, nodeID, itemID string) error {
	res, err := s.col("pubsub_items").DeleteOne(ctx, bson.M{"host": host, "node_id": nodeID, "item_id": itemID})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return storage.ErrNotFound
	}
	return nil
}

func (s *Store) Subscribe(ctx context.Context, sub *storage.PubSubSubscription) error {
	_, err := s.col("pubsub_subscriptions").UpdateOne(ctx,
		bson.M{"host": sub.Host, "node_id": sub.NodeID, "jid": sub.JID},
		bson.M{"$set": pubsubSubDoc{
			Host: sub.Host, NodeID: sub.NodeID, JID: sub.JID,
			SubID: sub.SubID, State: sub.State,
		}},
		options.UpdateOne().SetUpsert(true),
	)
	return err
}

func (s *Store) Unsubscribe(ctx context.Context, host, nodeID, jid string) error {
	_, err := s.col("pubsub_subscriptions").DeleteOne(ctx, bson.M{"host": host, "node_id": nodeID, "jid": jid})
	return err
}

func (s *Store) GetSubscription(ctx context.Context, host, nodeID, jid string) (*storage.PubSubSubscription, error) {
	var doc pubsubSubDoc
	err := s.col("pubsub_subscriptions").FindOne(ctx, bson.M{"host": host, "node_id": nodeID, "jid": jid}).Decode(&doc)
	if err == mongo.ErrNoDocuments {
		return nil, storage.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &storage.PubSubSubscription{
		Host: doc.Host, NodeID: doc.NodeID, JID: doc.JID,
		SubID: doc.SubID, State: doc.State,
	}, nil
}

func (s *Store) GetSubscriptions(ctx context.Context, host, nodeID string) ([]*storage.PubSubSubscription, error) {
	cursor, err := s.col("pubsub_subscriptions").Find(ctx, bson.M{"host": host, "node_id": nodeID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var subs []*storage.PubSubSubscription
	for cursor.Next(ctx) {
		var doc pubsubSubDoc
		if err := cursor.Decode(&doc); err != nil {
			return nil, err
		}
		subs = append(subs, &storage.PubSubSubscription{
			Host: doc.Host, NodeID: doc.NodeID, JID: doc.JID,
			SubID: doc.SubID, State: doc.State,
		})
	}
	return subs, cursor.Err()
}

func (s *Store) GetUserSubscriptions(ctx context.Context, host, jid string) ([]*storage.PubSubSubscription, error) {
	cursor, err := s.col("pubsub_subscriptions").Find(ctx, bson.M{"host": host, "jid": jid})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var subs []*storage.PubSubSubscription
	for cursor.Next(ctx) {
		var doc pubsubSubDoc
		if err := cursor.Decode(&doc); err != nil {
			return nil, err
		}
		subs = append(subs, &storage.PubSubSubscription{
			Host: doc.Host, NodeID: doc.NodeID, JID: doc.JID,
			SubID: doc.SubID, State: doc.State,
		})
	}
	return subs, cursor.Err()
}

// --- BookmarkStore ---

type bookmarkDoc struct {
	UserJID  string `bson:"user_jid"`
	RoomJID  string `bson:"room_jid"`
	Name     string `bson:"name"`
	Nick     string `bson:"nick"`
	Password string `bson:"password"`
	Autojoin bool   `bson:"autojoin"`
}

func (s *Store) SetBookmark(ctx context.Context, bm *storage.Bookmark) error {
	_, err := s.col("bookmarks").UpdateOne(ctx,
		bson.M{"user_jid": bm.UserJID, "room_jid": bm.RoomJID},
		bson.M{"$set": bookmarkDoc{
			UserJID: bm.UserJID, RoomJID: bm.RoomJID, Name: bm.Name,
			Nick: bm.Nick, Password: bm.Password, Autojoin: bm.Autojoin,
		}},
		options.UpdateOne().SetUpsert(true),
	)
	return err
}

func (s *Store) GetBookmark(ctx context.Context, userJID, roomJID string) (*storage.Bookmark, error) {
	var doc bookmarkDoc
	err := s.col("bookmarks").FindOne(ctx, bson.M{"user_jid": userJID, "room_jid": roomJID}).Decode(&doc)
	if err == mongo.ErrNoDocuments {
		return nil, storage.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &storage.Bookmark{
		UserJID: doc.UserJID, RoomJID: doc.RoomJID, Name: doc.Name,
		Nick: doc.Nick, Password: doc.Password, Autojoin: doc.Autojoin,
	}, nil
}

func (s *Store) GetBookmarks(ctx context.Context, userJID string) ([]*storage.Bookmark, error) {
	cursor, err := s.col("bookmarks").Find(ctx, bson.M{"user_jid": userJID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var bms []*storage.Bookmark
	for cursor.Next(ctx) {
		var doc bookmarkDoc
		if err := cursor.Decode(&doc); err != nil {
			return nil, err
		}
		bms = append(bms, &storage.Bookmark{
			UserJID: doc.UserJID, RoomJID: doc.RoomJID, Name: doc.Name,
			Nick: doc.Nick, Password: doc.Password, Autojoin: doc.Autojoin,
		})
	}
	return bms, cursor.Err()
}

func (s *Store) DeleteBookmark(ctx context.Context, userJID, roomJID string) error {
	res, err := s.col("bookmarks").DeleteOne(ctx, bson.M{"user_jid": userJID, "room_jid": roomJID})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return storage.ErrNotFound
	}
	return nil
}
