# xmpp-go

A comprehensive, production-grade XMPP library for Go supporting both client and server roles with a plugin architecture covering 50+ XEPs.

## Features

- Unified client/server `Session` type
- Plugin architecture with dependency resolution
- Streaming XML parser optimized for XMPP
- Multiple transports: TCP, WebSocket, BOSH
- Full SASL support: PLAIN, SCRAM-SHA-1/256/512 (+PLUS), EXTERNAL, ANONYMOUS
- STARTTLS with certificate verification
- Stanza multiplexer with middleware support
- DNS SRV and host-meta resolution
- Pluggable storage backends: Memory, File, SQLite, PostgreSQL, MySQL, MongoDB, Redis

## Installation

```bash
go get github.com/meszmate/xmpp-go
```

## Quick Start

```go
package main

import (
    "context"
    "log"

    xmpp "github.com/meszmate/xmpp-go"
    "github.com/meszmate/xmpp-go/jid"
    "github.com/meszmate/xmpp-go/stanza"
    "github.com/meszmate/xmpp-go/plugins/disco"
    "github.com/meszmate/xmpp-go/plugins/roster"
)

func main() {
    client, err := xmpp.NewClient(
        jid.MustParse("user@example.com"),
        "password",
        xmpp.WithPlugins(disco.New(), roster.New()),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    ctx := context.Background()
    if err := client.Connect(ctx); err != nil {
        log.Fatal(err)
    }

    msg := stanza.NewMessage(stanza.MessageChat)
    msg.To = jid.MustParse("friend@example.com")
    msg.Body = "Hello from xmpp-go!"
    _ = client.Send(ctx, msg)
}
```

## Feature Checklist

### Core (RFC 6120/6121/7622)
- [x] JID parsing and validation (RFC 7622)
- [x] JID escaping (XEP-0106)
- [x] XML stream reader/writer
- [x] Stream error conditions
- [x] STARTTLS negotiation
- [x] SASL authentication
- [x] Resource binding
- [x] Stanza types: Message, Presence, IQ
- [x] Stanza error conditions
- [x] Roster management (RFC 6121)
- [x] Presence management (RFC 6121)

### Transports
- [x] TCP transport
- [x] WebSocket transport (RFC 7395)
- [x] BOSH transport (XEP-0124/0206)

### SASL Mechanisms
- [x] PLAIN
- [x] SCRAM-SHA-1 / SCRAM-SHA-1-PLUS
- [x] SCRAM-SHA-256 / SCRAM-SHA-256-PLUS
- [x] SCRAM-SHA-512 / SCRAM-SHA-512-PLUS
- [x] EXTERNAL
- [x] ANONYMOUS

### Service Discovery & Capabilities
- [x] XEP-0030: Service Discovery
- [x] XEP-0115: Entity Capabilities

### Messaging
- [x] XEP-0085: Chat State Notifications
- [x] XEP-0184: Message Delivery Receipts
- [x] XEP-0280: Message Carbons
- [x] XEP-0308: Last Message Correction
- [x] XEP-0313: Message Archive Management
- [x] XEP-0333: Chat Markers
- [x] XEP-0334: Message Processing Hints
- [x] XEP-0359: Unique/Stable Stanza IDs
- [x] XEP-0393: Message Styling
- [x] XEP-0424: Message Retraction
- [x] XEP-0444: Message Reactions

### Group Chat
- [x] XEP-0045: Multi-User Chat
- [x] XEP-0249: Direct MUC Invitations
- [x] XEP-0369: MIX Core
- [x] XEP-0403/0405/0406/0407: MIX extensions
- [x] XEP-0425: Message Moderation

### Stream Management
- [x] XEP-0198: Stream Management

### PubSub & Storage
- [x] XEP-0004: Data Forms
- [x] XEP-0060: Publish-Subscribe
- [x] XEP-0163: Personal Eventing Protocol
- [x] XEP-0402: PEP Native Bookmarks

### User Profile
- [x] XEP-0054: vcard-temp
- [x] XEP-0084: User Avatar
- [x] XEP-0092: Software Version
- [x] XEP-0153: vCard-Based Avatars
- [x] XEP-0292: vCard4 over XMPP

### File Transfer
- [x] XEP-0047: In-Band Bytestreams
- [x] XEP-0065: SOCKS5 Bytestreams
- [x] XEP-0066: Out of Band Data
- [x] XEP-0234: Jingle File Transfer
- [x] XEP-0363: HTTP File Upload
- [x] XEP-0446/0447/0448: Stateless File Sharing

### Encryption
- [x] XEP-0380: Explicit Message Encryption
- [x] XEP-0384: OMEMO Encryption
- [x] XEP-0454: OMEMO Media Sharing

### Jingle (Voice/Video)
- [x] XEP-0166: Jingle
- [x] XEP-0167: Jingle RTP Sessions
- [x] XEP-0176: Jingle ICE-UDP Transport
- [x] XEP-0177: Jingle Raw UDP Transport
- [x] XEP-0320: DTLS-SRTP in Jingle
- [x] XEP-0353: Jingle Message Initiation

### Mobile & Push
- [x] XEP-0352: Client State Indication
- [x] XEP-0357: Push Notifications

### Server Features
- [x] XEP-0012: Last Activity
- [x] XEP-0050: Ad-Hoc Commands
- [x] XEP-0059: Result Set Management
- [x] XEP-0077: In-Band Registration
- [x] XEP-0114: Jabber Component Protocol
- [x] XEP-0191: Blocking Command
- [x] XEP-0215: External Service Discovery
- [x] XEP-0220: Server Dialback
- [x] XEP-0288: Bidirectional Server-to-Server

### Utilities
- [x] XEP-0082: Date/Time Profiles
- [x] XEP-0156: DNS/host-meta resolution
- [x] XEP-0199: XMPP Ping
- [x] XEP-0202: Entity Time
- [x] XEP-0203: Delayed Delivery
- [x] XEP-0231: Bits of Binary
- [x] XEP-0297: Stanza Forwarding
- [x] XEP-0300: Cryptographic Hash Functions
- [x] XEP-0368: SRV records for XMPP over TLS

### Modern Authentication
- [x] XEP-0386: Bind 2
- [x] XEP-0388: SASL2
- [x] XEP-0440: SASL Channel-Binding Type Capability
- [x] XEP-0484: FAST

## Storage Backends

xmpp-go includes a pluggable storage layer. All stateful plugins (roster, blocking, vcard, MUC, MAM, PubSub, bookmarks) automatically use the configured backend, falling back to in-memory storage when none is set.

| Backend | Package | External Dependency |
|---------|---------|-------------------|
| Memory | `storage/memory` | None |
| File (JSON) | `storage/file` | None |
| SQLite | `storage/sqlite` | `github.com/mattn/go-sqlite3` |
| PostgreSQL | `storage/postgres` | `github.com/jackc/pgx/v5` |
| MySQL | `storage/mysql` | `github.com/go-sql-driver/mysql` |
| MongoDB | `storage/mongodb` | `go.mongodb.org/mongo-driver/v2` |
| Redis | `storage/redis` | `github.com/redis/go-redis/v9` |

```go
import (
    xmpp "github.com/meszmate/xmpp-go"
    "github.com/meszmate/xmpp-go/storage/memory"
)

server, _ := xmpp.NewServer("example.com",
    xmpp.WithServerStorage(memory.New()),
    // ...
)
```

Backends with external dependencies live in separate Go modules so the main module stays dependency-free. Install only what you need:

```bash
go get github.com/meszmate/xmpp-go/storage/postgres
```

See the [Storage Guide](docs/storage.md) for full details.

## Documentation

- [Architecture Overview](docs/architecture.md)
- [Plugin Development Guide](docs/plugins.md)
- [Client Usage Guide](docs/client-guide.md)
- [Server Usage Guide](docs/server-guide.md)
- [Storage Guide](docs/storage.md)

## License

MIT License - see [LICENSE](LICENSE) for details.
