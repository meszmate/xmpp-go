package xmpp

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/meszmate/xmpp-go/jid"
	"github.com/meszmate/xmpp-go/stanza"
)

func TestWebSocketConnect(t *testing.T) {
	srv, _, memStore := startServer(t)
	addUser(t, memStore, "alice", "pw")

	httpSrv := httptest.NewServer(srv.WebSocketHandler())
	t.Cleanup(httpSrv.Close)
	wsURL := "ws" + strings.TrimPrefix(httpSrv.URL, "http")

	c, err := NewClient(jid.MustParse("alice@"+testDomain), "pw", WithWebSocket(wsURL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := c.Connect(ctx); err != nil {
		t.Fatalf("WebSocket Connect failed: %v", err)
	}
	if c.Session().State()&StateReady == 0 {
		t.Error("session not Ready after WebSocket negotiation")
	}
	if c.Session().LocalAddr().Resource() == "" {
		t.Errorf("expected a bound full JID, got %q", c.Session().LocalAddr())
	}
}

// TestWebSocketMessaging proves stanzas flow over the WebSocket transport.
func TestWebSocketMessaging(t *testing.T) {
	srv, _, memStore := startServer(t)
	addUser(t, memStore, "alice", "pw")
	addUser(t, memStore, "bob", "pw")

	httpSrv := httptest.NewServer(srv.WebSocketHandler())
	t.Cleanup(httpSrv.Close)
	wsURL := "ws" + strings.TrimPrefix(httpSrv.URL, "http")

	received := make(chan string, 1)
	bobHandler := HandlerFunc(func(ctx context.Context, s *Session, st stanza.Stanza) error {
		if m, ok := st.(*stanza.Message); ok && m.Body != "" {
			select {
			case received <- m.Body:
			default:
			}
		}
		return nil
	})

	bob, _ := NewClient(jid.MustParse("bob@"+testDomain), "pw", WithWebSocket(wsURL), WithHandler(bobHandler))
	defer bob.Close()
	alice, _ := NewClient(jid.MustParse("alice@"+testDomain), "pw", WithWebSocket(wsURL))
	defer alice.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := bob.Connect(ctx); err != nil {
		t.Fatalf("bob WS connect: %v", err)
	}
	if err := alice.Connect(ctx); err != nil {
		t.Fatalf("alice WS connect: %v", err)
	}

	msg := stanza.NewMessage(stanza.MessageChat)
	msg.To = jid.MustParse("bob@" + testDomain)
	msg.Body = "hello over websocket"
	if err := alice.Send(ctx, msg); err != nil {
		t.Fatalf("alice.Send: %v", err)
	}

	select {
	case body := <-received:
		if body != "hello over websocket" {
			t.Errorf("bob got %q", body)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("message did not arrive over WebSocket")
	}
}
