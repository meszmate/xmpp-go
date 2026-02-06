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

### Stream Negotiation
Stream features (STARTTLS, SASL, Bind) are negotiated in order. Each feature declares required/prohibited session states, enabling the negotiator to determine the correct sequence.
