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

## Authentication

Provide a custom authentication handler:

```go
xmpp.WithServerAuth(func(username, password string) (bool, error) {
    // Verify credentials against your database
    return true, nil
})
```

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
