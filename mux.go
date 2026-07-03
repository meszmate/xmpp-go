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
//
// A route's name is matched against the stanza element name (message, presence,
// iq) and, for IQ stanzas, against the IQ payload (child) element name. This
// lets plugins register for a payload namespace such as urn:xmpp:ping without
// caring that the wrapping element is <iq>.
func (m *Mux) HandleStanza(ctx context.Context, session *Session, st stanza.Stanza) error {
	if handled, err := m.dispatch(ctx, session, st); handled {
		return err
	}

	m.mu.RLock()
	fallback := m.fallback
	m.mu.RUnlock()

	if fallback != nil {
		return fallback.HandleStanza(ctx, session, st)
	}
	return nil
}

// dispatch routes a stanza to the first matching registered route, reporting
// whether any route matched. Unlike HandleStanza it does not consult the
// fallback handler, so callers can distinguish "a route handled it" from "no
// route matched" — used by the server to try plugin routes before falling back
// to built-in service handling or routing.
func (m *Mux) dispatch(ctx context.Context, session *Session, st stanza.Stanza) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	header := st.GetHeader()

	var payload xml.Name
	if iq, ok := st.(*stanza.IQ); ok {
		payload = iqPayloadName(iq)
	}

	for _, r := range m.routes {
		if r.stanzaType != "" && r.stanzaType != header.Type {
			continue
		}
		if !nameMatches(r.name, header.XMLName) && !nameMatches(r.name, payload) {
			continue
		}

		handler := r.handler
		// Apply middleware in reverse order
		for i := len(m.middleware) - 1; i >= 0; i-- {
			handler = m.middleware[i](handler)
		}
		return true, handler.HandleStanza(ctx, session, st)
	}

	return false, nil
}

// nameMatches reports whether a route name matches a target element name.
// Empty fields on the route name act as wildcards; a fully-empty route name
// matches anything.
func nameMatches(routeName, target xml.Name) bool {
	if routeName.Local == "" && routeName.Space == "" {
		return true
	}
	if routeName.Local != "" && routeName.Local != target.Local {
		return false
	}
	if routeName.Space != "" && routeName.Space != target.Space {
		return false
	}
	return true
}

// WithRoute returns a MuxOption that registers a route.
func WithRoute(name xml.Name, stanzaType string, handler Handler) MuxOption {
	return func(m *Mux) {
		m.Handle(name, stanzaType, handler)
	}
}
