package xmpp

import (
	"context"
	"testing"

	"github.com/meszmate/xmpp-go/stanza"
)

func TestChain(t *testing.T) {
	t.Parallel()
	var order []string

	base := HandlerFunc(func(ctx context.Context, s *Session, st stanza.Stanza) error {
		order = append(order, "base")
		return nil
	})

	mw1 := func(next Handler) Handler {
		return HandlerFunc(func(ctx context.Context, s *Session, st stanza.Stanza) error {
			order = append(order, "mw1")
			return next.HandleStanza(ctx, s, st)
		})
	}

	mw2 := func(next Handler) Handler {
		return HandlerFunc(func(ctx context.Context, s *Session, st stanza.Stanza) error {
			order = append(order, "mw2")
			return next.HandleStanza(ctx, s, st)
		})
	}

	handler := Chain(base, mw1, mw2)
	msg := stanza.NewMessage(stanza.MessageChat)
	handler.HandleStanza(context.Background(), nil, msg)

	// Chain applies in reverse: mw2(mw1(base))
	// Execution: mw1 -> mw2 -> base
	if len(order) != 3 {
		t.Fatalf("order = %v, want 3 entries", order)
	}
	if order[0] != "mw1" || order[1] != "mw2" || order[2] != "base" {
		t.Errorf("order = %v, want [mw1, mw2, base]", order)
	}
}

func TestRecoverMiddleware(t *testing.T) {
	t.Parallel()
	panicker := HandlerFunc(func(ctx context.Context, s *Session, st stanza.Stanza) error {
		panic("test panic")
	})

	handler := RecoverMiddleware()(panicker)
	msg := stanza.NewMessage(stanza.MessageChat)

	// Should not panic
	err := handler.HandleStanza(context.Background(), nil, msg)
	if err != nil {
		t.Errorf("RecoverMiddleware returned error: %v", err)
	}
}
