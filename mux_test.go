package xmpp

import (
	"context"
	"encoding/xml"
	"errors"
	"testing"

	"github.com/meszmate/xmpp-go/stanza"
)

func TestMuxHandleDispatch(t *testing.T) {
	t.Parallel()
	mux := NewMux()
	var called bool

	mux.Handle(xml.Name{Local: "message"}, "chat", HandlerFunc(
		func(ctx context.Context, s *Session, st stanza.Stanza) error {
			called = true
			return nil
		},
	))

	msg := stanza.NewMessage(stanza.MessageChat)
	err := mux.HandleStanza(context.Background(), nil, msg)
	if err != nil {
		t.Fatalf("HandleStanza: %v", err)
	}
	if !called {
		t.Error("handler was not called")
	}
}

func TestMuxNoMatchNilFallback(t *testing.T) {
	t.Parallel()
	mux := NewMux()
	mux.Handle(xml.Name{Local: "iq"}, "get", HandlerFunc(
		func(ctx context.Context, s *Session, st stanza.Stanza) error {
			t.Error("should not be called")
			return nil
		},
	))

	msg := stanza.NewMessage(stanza.MessageChat)
	err := mux.HandleStanza(context.Background(), nil, msg)
	if err != nil {
		t.Fatalf("HandleStanza: %v", err)
	}
}

func TestMuxFallback(t *testing.T) {
	t.Parallel()
	mux := NewMux()
	var fallbackCalled bool

	mux.SetFallback(HandlerFunc(
		func(ctx context.Context, s *Session, st stanza.Stanza) error {
			fallbackCalled = true
			return nil
		},
	))

	msg := stanza.NewMessage(stanza.MessageChat)
	err := mux.HandleStanza(context.Background(), nil, msg)
	if err != nil {
		t.Fatalf("HandleStanza: %v", err)
	}
	if !fallbackCalled {
		t.Error("fallback was not called")
	}
}

func TestMuxRouteByType(t *testing.T) {
	t.Parallel()
	mux := NewMux()
	var handledType string

	// Match by stanza type only (empty name)
	mux.Handle(xml.Name{}, "chat", HandlerFunc(
		func(ctx context.Context, s *Session, st stanza.Stanza) error {
			handledType = st.GetHeader().Type
			return nil
		},
	))

	msg := stanza.NewMessage(stanza.MessageChat)
	mux.HandleStanza(context.Background(), nil, msg)
	if handledType != "chat" {
		t.Errorf("handledType = %q, want %q", handledType, "chat")
	}
}

func TestMuxMiddlewareOrder(t *testing.T) {
	t.Parallel()
	mux := NewMux()
	var order []string

	mux.Use(func(next Handler) Handler {
		return HandlerFunc(func(ctx context.Context, s *Session, st stanza.Stanza) error {
			order = append(order, "mw1")
			return next.HandleStanza(ctx, s, st)
		})
	})
	mux.Use(func(next Handler) Handler {
		return HandlerFunc(func(ctx context.Context, s *Session, st stanza.Stanza) error {
			order = append(order, "mw2")
			return next.HandleStanza(ctx, s, st)
		})
	})

	mux.Handle(xml.Name{Local: "message"}, "", HandlerFunc(
		func(ctx context.Context, s *Session, st stanza.Stanza) error {
			order = append(order, "handler")
			return nil
		},
	))

	msg := stanza.NewMessage(stanza.MessageChat)
	mux.HandleStanza(context.Background(), nil, msg)

	// Middleware applied in reverse: mw1(mw2(handler))
	// Execution: mw1 -> mw2 -> handler
	if len(order) != 3 {
		t.Fatalf("order = %v, want 3 entries", order)
	}
	if order[0] != "mw1" || order[1] != "mw2" || order[2] != "handler" {
		t.Errorf("order = %v, want [mw1, mw2, handler]", order)
	}
}

func TestMuxWithRouteOption(t *testing.T) {
	t.Parallel()
	var called bool
	mux := NewMux(
		WithRoute(xml.Name{Local: "iq"}, "get", HandlerFunc(
			func(ctx context.Context, s *Session, st stanza.Stanza) error {
				called = true
				return nil
			},
		)),
	)

	iq := stanza.NewIQ(stanza.IQGet)
	mux.HandleStanza(context.Background(), nil, iq)
	if !called {
		t.Error("handler registered via WithRoute was not called")
	}
}

func TestMuxHandlerError(t *testing.T) {
	t.Parallel()
	mux := NewMux()
	wantErr := errors.New("handler error")

	mux.Handle(xml.Name{Local: "message"}, "", HandlerFunc(
		func(ctx context.Context, s *Session, st stanza.Stanza) error {
			return wantErr
		},
	))

	msg := stanza.NewMessage(stanza.MessageChat)
	err := mux.HandleStanza(context.Background(), nil, msg)
	if err != wantErr {
		t.Errorf("error = %v, want %v", err, wantErr)
	}
}
