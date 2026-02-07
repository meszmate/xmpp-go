# Storage Guide

xmpp-go includes a pluggable storage layer that persists user accounts, rosters, blocked JIDs, vCards, offline messages, message archives (MAM), MUC rooms, PubSub data, and bookmarks.

## Overview

The `storage.Storage` interface is the entry point. It exposes sub-store accessors that return `nil` when the backend does not support that store:

```go
type Storage interface {
    io.Closer
    Init(ctx context.Context) error

    UserStore() UserStore
    RosterStore() RosterStore
    BlockingStore() BlockingStore
    VCardStore() VCardStore
    OfflineStore() OfflineStore
    MAMStore() MAMStore
    MUCRoomStore() MUCRoomStore
    PubSubStore() PubSubStore
    BookmarkStore() BookmarkStore
}
```

All built-in backends implement every sub-store.

## Backends

### Memory

In-process maps protected by a mutex. Data is lost when the process exits. No external dependencies.

```go
import "github.com/meszmate/xmpp-go/storage/memory"

store := memory.New()
```

### File

Stores data as JSON files in a directory on disk. No external dependencies.

```go
import "github.com/meszmate/xmpp-go/storage/file"

store := file.New("/var/lib/xmpp/data")
```

Directory structure created automatically:

```
data/
  users/
  roster/
  blocking/
  vcards/
  offline/
  mam/
  muc_rooms/
  muc_affiliations/
  pubsub_nodes/
  pubsub_items/
  pubsub_subscriptions/
  bookmarks/
```

### SQLite

Uses the shared SQL layer with automatic schema migrations.

```bash
go get github.com/meszmate/xmpp-go/storage/sqlite
```

```go
import "github.com/meszmate/xmpp-go/storage/sqlite"

store, err := sqlite.New("xmpp.db")
```

### PostgreSQL

```bash
go get github.com/meszmate/xmpp-go/storage/postgres
```

```go
import "github.com/meszmate/xmpp-go/storage/postgres"

store, err := postgres.New("postgres://user:pass@localhost/xmpp?sslmode=disable")
```

### MySQL

```bash
go get github.com/meszmate/xmpp-go/storage/mysql
```

```go
import "github.com/meszmate/xmpp-go/storage/mysql"

store, err := mysql.New("user:pass@tcp(localhost:3306)/xmpp")
```

### MongoDB

```bash
go get github.com/meszmate/xmpp-go/storage/mongodb
```

```go
import "github.com/meszmate/xmpp-go/storage/mongodb"

store, err := mongodb.New("mongodb://localhost:27017", "xmpp")
```

### Redis

```bash
go get github.com/meszmate/xmpp-go/storage/redis
```

```go
import xmppredis "github.com/meszmate/xmpp-go/storage/redis"

store := xmppredis.New(&xmppredis.Options{
    Addr: "localhost:6379",
})
```

## Wiring to the Server

Pass the storage backend when creating a server:

```go
server, err := xmpp.NewServer("example.com",
    xmpp.WithServerStorage(store),
    xmpp.WithServerPlugins(
        disco.New(),
        roster.New(),
        blocking.New(),
        vcard.New(),
        muc.New(),
        mam.New(),
        bookmarks.New(),
        pubsub.New(),
    ),
)
```

The server calls `store.Init(ctx)` at startup and `store.Close()` on shutdown. All stateful plugins receive the storage through `InitParams.Storage` and use the relevant sub-store automatically.

### Automatic Authentication

When a storage backend with a `UserStore` is configured and no explicit `WithServerAuth` handler is provided, the server automatically derives authentication from the user store. This means you can manage users through the `UserStore` API and authentication will work out of the box.

## Sub-Stores

### UserStore

Manages user accounts with SCRAM credential support.

| Method | Description |
|--------|-------------|
| `CreateUser(ctx, *User) error` | Create a new account (returns `ErrUserExists` on conflict) |
| `GetUser(ctx, username) (*User, error)` | Retrieve a user (returns `ErrNotFound` if missing) |
| `UpdateUser(ctx, *User) error` | Update account fields |
| `DeleteUser(ctx, username) error` | Delete an account |
| `UserExists(ctx, username) (bool, error)` | Check existence |
| `Authenticate(ctx, username, password) (bool, error)` | Validate credentials |

### RosterStore

Manages contact lists.

| Method | Description |
|--------|-------------|
| `UpsertRosterItem(ctx, *RosterItem) error` | Add or update a contact |
| `GetRosterItem(ctx, userJID, contactJID) (*RosterItem, error)` | Get one contact |
| `GetRosterItems(ctx, userJID) ([]*RosterItem, error)` | Get all contacts |
| `DeleteRosterItem(ctx, userJID, contactJID) error` | Remove a contact |
| `GetRosterVersion(ctx, userJID) (string, error)` | Get roster version |
| `SetRosterVersion(ctx, userJID, version) error` | Set roster version |

### BlockingStore

Manages JID block lists (XEP-0191).

| Method | Description |
|--------|-------------|
| `BlockJID(ctx, userJID, blockedJID) error` | Block a JID |
| `UnblockJID(ctx, userJID, blockedJID) error` | Unblock a JID |
| `IsBlocked(ctx, userJID, blockedJID) (bool, error)` | Check if blocked |
| `GetBlockedJIDs(ctx, userJID) ([]string, error)` | List all blocked JIDs |

### VCardStore

Stores vCard XML data (XEP-0054).

| Method | Description |
|--------|-------------|
| `SetVCard(ctx, userJID, data) error` | Store vCard XML |
| `GetVCard(ctx, userJID) ([]byte, error)` | Retrieve vCard XML |
| `DeleteVCard(ctx, userJID) error` | Remove vCard |

### OfflineStore

Queues messages for offline users.

| Method | Description |
|--------|-------------|
| `StoreOfflineMessage(ctx, *OfflineMessage) error` | Queue a message |
| `GetOfflineMessages(ctx, userJID) ([]*OfflineMessage, error)` | Get all queued messages |
| `DeleteOfflineMessages(ctx, userJID) error` | Clear the queue |
| `CountOfflineMessages(ctx, userJID) (int, error)` | Count queued messages |

### MAMStore

Message Archive Management (XEP-0313).

| Method | Description |
|--------|-------------|
| `ArchiveMessage(ctx, *ArchivedMessage) error` | Store a message |
| `QueryMessages(ctx, *MAMQuery) (*MAMResult, error)` | Query with filters and RSM |
| `DeleteMessageArchive(ctx, userJID) error` | Delete all archived messages |

`MAMQuery` supports filtering by correspondent (`WithJID`), time range (`Start`/`End`), Result Set Management (`AfterID`/`BeforeID`), and page size (`Max`).

### MUCRoomStore

Multi-User Chat rooms (XEP-0045).

| Method | Description |
|--------|-------------|
| `CreateRoom(ctx, *MUCRoom) error` | Create a room |
| `GetRoom(ctx, roomJID) (*MUCRoom, error)` | Get room details |
| `UpdateRoom(ctx, *MUCRoom) error` | Update room config |
| `DeleteRoom(ctx, roomJID) error` | Delete room (and affiliations) |
| `ListRooms(ctx) ([]*MUCRoom, error)` | List all rooms |
| `SetAffiliation(ctx, *MUCAffiliation) error` | Set user affiliation |
| `GetAffiliation(ctx, roomJID, userJID) (*MUCAffiliation, error)` | Get user affiliation |
| `GetAffiliations(ctx, roomJID) ([]*MUCAffiliation, error)` | List room affiliations |
| `RemoveAffiliation(ctx, roomJID, userJID) error` | Remove affiliation |

### PubSubStore

Publish-Subscribe (XEP-0060).

| Method | Description |
|--------|-------------|
| `CreateNode(ctx, *PubSubNode) error` | Create a node |
| `GetNode(ctx, host, nodeID) (*PubSubNode, error)` | Get node details |
| `DeleteNode(ctx, host, nodeID) error` | Delete node (and items/subscriptions) |
| `ListNodes(ctx, host) ([]*PubSubNode, error)` | List all nodes |
| `UpsertItem(ctx, *PubSubItem) error` | Publish or update an item |
| `GetItem(ctx, host, nodeID, itemID) (*PubSubItem, error)` | Get one item |
| `GetItems(ctx, host, nodeID) ([]*PubSubItem, error)` | Get all items |
| `DeleteItem(ctx, host, nodeID, itemID) error` | Delete an item |
| `Subscribe(ctx, *PubSubSubscription) error` | Subscribe to a node |
| `Unsubscribe(ctx, host, nodeID, jid) error` | Unsubscribe |
| `GetSubscription(ctx, host, nodeID, jid) (*PubSubSubscription, error)` | Get subscription |
| `GetSubscriptions(ctx, host, nodeID) ([]*PubSubSubscription, error)` | List node subscriptions |
| `GetUserSubscriptions(ctx, host, jid) ([]*PubSubSubscription, error)` | List user subscriptions |

### BookmarkStore

Conference bookmarks (XEP-0402).

| Method | Description |
|--------|-------------|
| `SetBookmark(ctx, *Bookmark) error` | Add or update a bookmark |
| `GetBookmark(ctx, userJID, roomJID) (*Bookmark, error)` | Get one bookmark |
| `GetBookmarks(ctx, userJID) ([]*Bookmark, error)` | Get all bookmarks |
| `DeleteBookmark(ctx, userJID, roomJID) error` | Remove a bookmark |

## Sentinel Errors

All backends return consistent sentinel errors:

| Error | Meaning |
|-------|---------|
| `storage.ErrNotFound` | Requested entity does not exist |
| `storage.ErrUserExists` | User already exists (on `CreateUser`) |
| `storage.ErrAuthFailed` | Invalid credentials (on `Authenticate`) |

Use `errors.Is` to check:

```go
user, err := store.UserStore().GetUser(ctx, "alice")
if errors.Is(err, storage.ErrNotFound) {
    // user does not exist
}
```

## Module Structure

Backends with external dependencies are separate Go modules to keep the main module dependency-free:

```
Main module (github.com/meszmate/xmpp-go):
  storage/            Interfaces only
  storage/memory/     In-memory backend
  storage/file/       File backend (JSON on disk)
  storage/sql/        Shared SQL layer (database/sql from stdlib)
  storage/storagetest/ Conformance test suite

Sub-modules (separate go.mod):
  storage/sqlite/     github.com/mattn/go-sqlite3
  storage/postgres/   github.com/jackc/pgx/v5
  storage/mysql/      github.com/go-sql-driver/mysql
  storage/mongodb/    go.mongodb.org/mongo-driver/v2
  storage/redis/      github.com/redis/go-redis/v9
```

## Implementing a Custom Backend

To create your own storage backend, implement the `storage.Storage` interface:

```go
package mystore

import (
    "context"

    "github.com/meszmate/xmpp-go/storage"
)

type Store struct {
    // your fields
}

func New() *Store { return &Store{} }

func (s *Store) Init(ctx context.Context) error { return nil }
func (s *Store) Close() error                   { return nil }

func (s *Store) UserStore() storage.UserStore         { return s }
func (s *Store) RosterStore() storage.RosterStore     { return s }
func (s *Store) BlockingStore() storage.BlockingStore { return s }
func (s *Store) VCardStore() storage.VCardStore       { return s }
func (s *Store) OfflineStore() storage.OfflineStore   { return s }
func (s *Store) MAMStore() storage.MAMStore           { return s }
func (s *Store) MUCRoomStore() storage.MUCRoomStore   { return s }
func (s *Store) PubSubStore() storage.PubSubStore     { return s }
func (s *Store) BookmarkStore() storage.BookmarkStore { return s }

// Implement all sub-store methods...
```

Return `nil` from any accessor for stores you don't support -- plugins will fall back to in-memory.

### Conformance Testing

Validate your backend with the shared test suite:

```go
package mystore_test

import (
    "testing"

    "github.com/meszmate/xmpp-go/storage"
    "github.com/meszmate/xmpp-go/storage/storagetest"
    "mymodule/mystore"
)

func TestMyStore(t *testing.T) {
    storagetest.TestStorage(t, func() storage.Storage {
        return mystore.New()
    })
}
```

The test suite covers CRUD operations, edge cases (not found, duplicates), and all 9 sub-stores.
