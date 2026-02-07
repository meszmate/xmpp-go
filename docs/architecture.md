# Architecture Overview

## Layers

```
┌─────────────────────────────────────────┐
│          Plugins (XEP implementations)  │
├─────────────────────────────────────────┤
│       Plugin Manager & Registry         │
├─────────────────────────────────────────┤
│         Client / Server / Component     │
├─────────────────────────────────────────┤
│    Session (negotiation, mux, routing)  │
├─────────────────────────────────────────┤
│  SASL │ STARTTLS │ Bind │ Features     │
├─────────────────────────────────────────┤
│    Stanza │ Stream │ XML │ Transport   │
├─────────────────────────────────────────┤
│           Storage (pluggable)           │
├─────────────────────────────────────────┤
│              JID │ Namespaces           │
└─────────────────────────────────────────┘
```

## Core Concepts

### Session
The `Session` is the central type, representing an active XMPP connection. It wraps a transport, manages stream negotiation, and routes stanzas. Both client and server connections use the same Session type, distinguished by `SessionState` flags.

### Transport
Transports abstract the underlying connection (TCP, WebSocket, BOSH). They provide `io.ReadWriteCloser` semantics and handle transport-specific framing.

### Stanza Routing
The `Mux` routes incoming stanzas to registered handlers based on XML name and stanza type. Middleware wraps handlers for cross-cutting concerns (logging, error recovery, etc.).

### Plugin System
Plugins implement the `Plugin` interface and register stream features, stanza handlers, and service discovery information. The `Manager` handles dependency resolution and lifecycle management.

### Storage Layer
The `storage.Storage` interface abstracts persistence. It exposes sub-store accessors (`UserStore()`, `RosterStore()`, `BlockingStore()`, etc.) that return `nil` when the backend does not support that store. The server passes storage into plugins via `InitParams.Storage`. Plugins that receive a non-nil store use it; otherwise they fall back to in-memory maps.

Available backends:
- **Memory** (`storage/memory`) -- in-process maps, no persistence
- **File** (`storage/file`) -- JSON files on disk
- **SQLite** (`storage/sqlite`) -- via shared SQL layer
- **PostgreSQL** (`storage/postgres`) -- via shared SQL layer
- **MySQL** (`storage/mysql`) -- via shared SQL layer
- **MongoDB** (`storage/mongodb`) -- document store
- **Redis** (`storage/redis`) -- key-value store

The shared SQL layer (`storage/sql`) provides dialect-agnostic query code and automatic schema migrations. Each SQL backend supplies a `Dialect` implementation for placeholder syntax, type mappings, and upsert behavior.

### Stream Negotiation
Stream features (STARTTLS, SASL, Bind) are negotiated in order. Each feature declares required/prohibited session states, enabling the negotiator to determine the correct sequence.
