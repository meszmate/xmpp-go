package xmpp

import (
	"context"
	"log"

	"github.com/meszmate/xmpp-go/stanza"
)

// Middleware wraps a Handler to add cross-cutting behavior.
type Middleware func(Handler) Handler

// Chain applies a series of middleware to a handler.
func Chain(handler Handler, middleware ...Middleware) Handler {
	for i := len(middleware) - 1; i >= 0; i-- {
		handler = middleware[i](handler)
	}
	return handler
}

// LogMiddleware logs incoming stanzas.
func LogMiddleware() Middleware {
	return func(next Handler) Handler {
		return HandlerFunc(func(ctx context.Context, session *Session, st stanza.Stanza) error {
			header := st.GetHeader()
			log.Printf("xmpp: %s from=%s to=%s id=%s type=%s",
				st.StanzaType(), header.From, header.To, header.ID, header.Type)
			return next.HandleStanza(ctx, session, st)
		})
	}
}

// RecoverMiddleware recovers from panics in handlers.
func RecoverMiddleware() Middleware {
	return func(next Handler) Handler {
		return HandlerFunc(func(ctx context.Context, session *Session, st stanza.Stanza) error {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("xmpp: recovered from panic: %v", r)
				}
			}()
			return next.HandleStanza(ctx, session, st)
		})
	}
}
