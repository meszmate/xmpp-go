package xmpp

import (
	"context"

	"github.com/meszmate/xmpp-go/stanza"
)

// Handler handles incoming XMPP stanzas.
type Handler interface {
	HandleStanza(ctx context.Context, session *Session, st stanza.Stanza) error
}

// HandlerFunc is an adapter to allow ordinary functions as handlers.
type HandlerFunc func(ctx context.Context, session *Session, st stanza.Stanza) error

// HandleStanza calls f(ctx, session, st).
func (f HandlerFunc) HandleStanza(ctx context.Context, session *Session, st stanza.Stanza) error {
	return f(ctx, session, st)
}
