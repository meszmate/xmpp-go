# Client Usage Guide

## Basic Connection

```go
package main

import (
    "context"
    "log"

    xmpp "github.com/meszmate/xmpp-go"
    "github.com/meszmate/xmpp-go/jid"
    "github.com/meszmate/xmpp-go/plugins/disco"
    "github.com/meszmate/xmpp-go/plugins/roster"
    "github.com/meszmate/xmpp-go/plugins/carbons"
)

func main() {
    addr := jid.MustParse("user@example.com")

    client, err := xmpp.NewClient(addr, "password",
        xmpp.WithPlugins(
            disco.New(),
            roster.New(),
            carbons.New(),
        ),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    ctx := context.Background()
    if err := client.Connect(ctx); err != nil {
        log.Fatal(err)
    }

    // Send a message
    // Use plugins
    // Handle incoming stanzas
}
```

## Sending Messages

```go
msg := stanza.NewMessage(stanza.MessageChat)
msg.To = jid.MustParse("friend@example.com")
msg.Body = "Hello!"

if err := client.Send(ctx, msg); err != nil {
    log.Fatal(err)
}
```

## Handling Stanzas

Register handlers via the mux:

```go
client, err := xmpp.NewClient(addr, "password",
    xmpp.WithHandler(xmpp.HandlerFunc(func(ctx context.Context, s *xmpp.Session, st stanza.Stanza) error {
        // Handle incoming stanza
        return nil
    })),
)
```

## Using Plugins

Access plugins by name:

```go
if d, ok := client.Plugin("disco"); ok {
    // Use the disco plugin
    _ = d
}
```
