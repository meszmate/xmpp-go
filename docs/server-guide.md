# Server Usage Guide

## Basic Server

```go
package main

import (
    "context"
    "log"

    xmpp "github.com/meszmate/xmpp-go"
    "github.com/meszmate/xmpp-go/plugins/disco"
    "github.com/meszmate/xmpp-go/plugins/roster"
    "github.com/meszmate/xmpp-go/plugins/ping"
)

func main() {
    server, err := xmpp.NewServer("example.com",
        xmpp.WithServerPlugins(
            disco.New(),
            roster.New(),
            ping.New(),
        ),
        xmpp.WithServerTLS("cert.pem", "key.pem"),
        xmpp.WithServerAddr(":5222"),
    )
    if err != nil {
        log.Fatal(err)
    }

    ctx := context.Background()
    if err := server.ListenAndServe(ctx); err != nil {
        log.Fatal(err)
    }
}
```

## Storage

Configure a storage backend to persist user data, rosters, messages, and more. Without storage, all stateful plugins use in-memory fallbacks.

### Memory (no persistence)

```go
import "github.com/meszmate/xmpp-go/storage/memory"

xmpp.WithServerStorage(memory.New())
```

### File (JSON on disk)

```go
import "github.com/meszmate/xmpp-go/storage/file"

xmpp.WithServerStorage(file.New("/var/lib/xmpp/data"))
```

### SQLite

```go
import "github.com/meszmate/xmpp-go/storage/sqlite"

store, err := sqlite.New("xmpp.db")
if err != nil {
    log.Fatal(err)
}
xmpp.WithServerStorage(store)
```

### PostgreSQL

```go
import "github.com/meszmate/xmpp-go/storage/postgres"

store, err := postgres.New("postgres://user:pass@localhost/xmpp?sslmode=disable")
if err != nil {
    log.Fatal(err)
}
xmpp.WithServerStorage(store)
```

### MySQL

```go
import "github.com/meszmate/xmpp-go/storage/mysql"

store, err := mysql.New("user:pass@tcp(localhost:3306)/xmpp")
if err != nil {
    log.Fatal(err)
}
xmpp.WithServerStorage(store)
```

### MongoDB

```go
import "github.com/meszmate/xmpp-go/storage/mongodb"

store, err := mongodb.New("mongodb://localhost:27017", "xmpp")
if err != nil {
    log.Fatal(err)
}
xmpp.WithServerStorage(store)
```

### Redis

```go
import "github.com/meszmate/xmpp-go/storage/redis"

store := redis.New(&redis.Options{
    Addr: "localhost:6379",
})
xmpp.WithServerStorage(store)
```

When a storage backend with a `UserStore` is configured and no explicit `WithServerAuth` is set, the server automatically derives authentication from the storage layer.

## Authentication

Provide a custom authentication handler:

```go
xmpp.WithServerAuth(func(username, password string) (bool, error) {
    // Verify credentials against your database
    return true, nil
})
```

When using a storage backend with a `UserStore`, you can skip `WithServerAuth` -- the server will authenticate against the stored user accounts automatically.

## Session Handling

Register a handler for new sessions:

```go
xmpp.WithServerSessionHandler(func(ctx context.Context, session *xmpp.Session) {
    log.Printf("New session: %s", session.LocalAddr())
})
```

## Component Protocol (XEP-0114)

```go
comp, err := xmpp.NewComponent("gateway.example.com", "secret",
    xmpp.WithComponentAddr("localhost:5275"),
)
if err != nil {
    log.Fatal(err)
}

if err := comp.Connect(ctx); err != nil {
    log.Fatal(err)
}
```
