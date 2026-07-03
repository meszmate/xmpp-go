package xmpp_test

import (
	"context"
	"testing"
	"time"

	xmpp "github.com/meszmate/xmpp-go"
	"github.com/meszmate/xmpp-go/jid"
	"github.com/meszmate/xmpp-go/plugins/disco"
	"github.com/meszmate/xmpp-go/plugins/roster"
	"github.com/meszmate/xmpp-go/stanza"
	"github.com/meszmate/xmpp-go/storage"
	"github.com/meszmate/xmpp-go/storage/memory"
)

// startExampleServer boots a library server on a random loopback port from the
// external test package (using only exported API).
func startExampleServer(t *testing.T) (addr string, store *memory.Store) {
	t.Helper()
	store = memory.New()
	srv, err := xmpp.NewServer("localhost",
		xmpp.WithServerAddr("127.0.0.1:0"),
		xmpp.WithServerStorage(store),
	)
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	go func() { _ = srv.ListenAndServe(context.Background()) }()
	t.Cleanup(func() { _ = srv.Close() })

	deadline := time.Now().Add(5 * time.Second)
	for srv.Addr() == nil {
		if time.Now().After(deadline) {
			t.Fatal("server did not start listening")
		}
		time.Sleep(2 * time.Millisecond)
	}
	return srv.Addr().String(), store
}

// TestReadmeMessagingExample exercises the exact shape of the README Quick Start
// (NewClient + WithPlugins(disco, roster) + Connect + Send) against a real
// server, proving the documented example now works end to end.
func TestReadmeMessagingExample(t *testing.T) {
	addr, store := startExampleServer(t)
	_ = store.UserStore().CreateUser(context.Background(), &storage.User{Username: "user", Password: "password"})
	_ = store.UserStore().CreateUser(context.Background(), &storage.User{Username: "friend", Password: "pw"})

	// friend logs in to receive the message.
	got := make(chan string, 1)
	friend, err := xmpp.NewClient(
		jid.MustParse("friend@localhost"),
		"pw",
		xmpp.WithConnectAddr(addr),
		xmpp.WithHandler(xmpp.HandlerFunc(func(ctx context.Context, s *xmpp.Session, st stanza.Stanza) error {
			if m, ok := st.(*stanza.Message); ok && m.Body != "" {
				select {
				case got <- m.Body:
				default:
				}
			}
			return nil
		})),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer friend.Close()

	// This mirrors the README example (plus WithConnectAddr for the test).
	client, err := xmpp.NewClient(
		jid.MustParse("user@localhost"),
		"password",
		xmpp.WithConnectAddr(addr),
		xmpp.WithPlugins(disco.New(), roster.New()),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	ctx := context.Background()
	if err := friend.Connect(ctx); err != nil {
		t.Fatalf("friend.Connect: %v", err)
	}
	if err := client.Connect(ctx); err != nil {
		t.Fatalf("client.Connect (README example) failed: %v", err)
	}

	msg := stanza.NewMessage(stanza.MessageChat)
	msg.To = jid.MustParse("friend@localhost")
	msg.Body = "Hello from xmpp-go!"
	if err := client.Send(ctx, msg); err != nil {
		t.Fatalf("client.Send: %v", err)
	}

	select {
	case body := <-got:
		if body != "Hello from xmpp-go!" {
			t.Errorf("friend received %q", body)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("friend never received the README example message")
	}
}
