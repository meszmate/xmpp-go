package xmpp

import (
	"context"
	"encoding/xml"
	"sync"

	"github.com/meszmate/xmpp-go/stanza"
)

// MuxOption configures the Mux.
type MuxOption func(*Mux)

// route represents a registered handler with matching criteria.
type route struct {
	name       xml.Name
	stanzaType string
	handler    Handler
}

// Mux is a stanza multiplexer that routes stanzas to handlers.
type Mux struct {
	mu         sync.RWMutex
	routes     []route
	middleware []Middleware
	fallback   Handler
}

// NewMux creates a new Mux.
func NewMux(opts ...MuxOption) *Mux {
	m := &Mux{}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

// Handle registers a handler for stanzas matching the given XML name and type.
func (m *Mux) Handle(name xml.Name, stanzaType string, handler Handler) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.routes = append(m.routes, route{
		name:       name,
		stanzaType: stanzaType,
		handler:    handler,
	})
}

// HandleFunc registers a handler function.
func (m *Mux) HandleFunc(name xml.Name, stanzaType string, f HandlerFunc) {
	m.Handle(name, stanzaType, f)
}

// Use adds middleware to the mux.
func (m *Mux) Use(mw ...Middleware) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.middleware = append(m.middleware, mw...)
}

// SetFallback sets the fallback handler for unmatched stanzas.
func (m *Mux) SetFallback(h Handler) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.fallback = h
}

// HandleStanza routes a stanza to the appropriate handler.
func (m *Mux) HandleStanza(ctx context.Context, session *Session, st stanza.Stanza) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	header := st.GetHeader()

	for _, r := range m.routes {
		if r.stanzaType != "" && r.stanzaType != header.Type {
			continue
		}
		if r.name.Local != "" && r.name.Local != header.XMLName.Local {
			continue
		}
		if r.name.Space != "" && r.name.Space != header.XMLName.Space {
			continue
		}

		handler := r.handler
		// Apply middleware in reverse order
		for i := len(m.middleware) - 1; i >= 0; i-- {
			handler = m.middleware[i](handler)
		}
		return handler.HandleStanza(ctx, session, st)
	}

	if m.fallback != nil {
		return m.fallback.HandleStanza(ctx, session, st)
	}

	return nil
}

// WithRoute returns a MuxOption that registers a route.
func WithRoute(name xml.Name, stanzaType string, handler Handler) MuxOption {
	return func(m *Mux) {
		m.Handle(name, stanzaType, handler)
	}
}
