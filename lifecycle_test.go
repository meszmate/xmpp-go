package xmpp

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/meszmate/xmpp-go/jid"
	"github.com/meszmate/xmpp-go/stanza"
)

func TestClientCloseStopsReceiveLoop(t *testing.T) {
	_, addr, store := startServer(t)
	addUser(t, store, "alice", "pw")

	client, _ := NewClient(jid.MustParse("alice@"+testDomain), "pw", WithConnectAddr(addr))
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Connect: %v", err)
	}

	done := client.Done()
	if err := client.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// The receive loop must terminate promptly after Close.
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("receive loop did not stop within 3s of Close")
	}
}

func TestClientDoneReportsServerDisconnect(t *testing.T) {
	srv, addr, store := startServer(t)
	addUser(t, store, "alice", "pw")

	client, _ := NewClient(jid.MustParse("alice@"+testDomain), "pw", WithConnectAddr(addr))
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer client.Close()

	done := client.Done()
	// Tear down the server; the client's receive loop should observe the drop.
	_ = srv.Close()

	select {
	case <-done:
		// Either nil (clean EOF) or an error is acceptable; the point is it
		// terminates rather than hanging forever.
	case <-time.After(3 * time.Second):
		t.Fatal("receive loop did not observe server disconnect within 3s")
	}
}

func TestConnectContextCancellation(t *testing.T) {
	_, addr, store := startServer(t)
	addUser(t, store, "alice", "pw")

	client, _ := NewClient(jid.MustParse("alice@"+testDomain), "pw", WithConnectAddr(addr))
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already cancelled

	err := client.Connect(ctx)
	if err == nil {
		t.Fatal("Connect with a cancelled context should fail")
	}
}

func TestDoubleConnectFails(t *testing.T) {
	_, addr, store := startServer(t)
	addUser(t, store, "alice", "pw")

	client, _ := NewClient(jid.MustParse("alice@"+testDomain), "pw", WithConnectAddr(addr))
	defer client.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := client.Connect(ctx); err != nil {
		t.Fatalf("first Connect: %v", err)
	}
	if err := client.Connect(ctx); err == nil {
		t.Error("second Connect on a connected client should fail")
	}
}

func TestSendAfterServerCloseErrors(t *testing.T) {
	srv, addr, store := startServer(t)
	addUser(t, store, "alice", "pw")

	client, _ := NewClient(jid.MustParse("alice@"+testDomain), "pw", WithConnectAddr(addr))
	defer client.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Connect: %v", err)
	}

	done := client.Done()
	_ = srv.Close()
	<-done // wait for the receive loop to observe the drop and close the session

	msg := stanza.NewMessage(stanza.MessageChat)
	msg.To = jid.MustParse("bob@" + testDomain)
	msg.Body = "into the void"
	if err := client.Send(ctx, msg); err == nil {
		t.Error("Send after server disconnect must return an error, not fake success")
	}
}

func TestHandlerPanicDoesNotKillReceiveLoop(t *testing.T) {
	_, addr, store := startServer(t)
	addUser(t, store, "alice", "pw")
	addUser(t, store, "bob", "pw")

	got := make(chan string, 2)
	bobHandler := HandlerFunc(func(ctx context.Context, s *Session, st stanza.Stanza) error {
		m, ok := st.(*stanza.Message)
		if !ok || m.Body == "" {
			return nil
		}
		if m.Body == "boom" {
			panic("handler panic on boom")
		}
		got <- m.Body
		return nil
	})

	bob, _ := NewClient(jid.MustParse("bob@"+testDomain), "pw", WithConnectAddr(addr), WithHandler(bobHandler))
	defer bob.Close()
	alice, _ := NewClient(jid.MustParse("alice@"+testDomain), "pw", WithConnectAddr(addr))
	defer alice.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := bob.Connect(ctx); err != nil {
		t.Fatal(err)
	}
	if err := alice.Connect(ctx); err != nil {
		t.Fatal(err)
	}

	send := func(body string) {
		m := stanza.NewMessage(stanza.MessageChat)
		m.To = jid.MustParse("bob@" + testDomain)
		m.Body = body
		if err := alice.Send(ctx, m); err != nil {
			t.Fatalf("Send(%s): %v", body, err)
		}
	}
	send("boom")  // panics in bob's handler
	send("hello") // must still be delivered

	select {
	case body := <-got:
		if body != "hello" {
			t.Errorf("got %q, want hello", body)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("receive loop died after a handler panic — 'hello' never arrived")
	}
}

func TestConnectDeadlineOnStalledServer(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	// Accept the connection but never send the stream header.
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		time.Sleep(5 * time.Second)
		_ = conn.Close()
	}()

	client, _ := NewClient(jid.MustParse("alice@"+testDomain), "pw", WithConnectAddr(ln.Addr().String()))
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	start := time.Now()
	err = client.Connect(ctx)
	if err == nil {
		t.Fatal("Connect to a stalled server must fail")
	}
	if elapsed := time.Since(start); elapsed > 3*time.Second {
		t.Errorf("Connect ignored the context deadline (took %v)", elapsed)
	}
}

func TestNewClientValidation(t *testing.T) {
	if _, err := NewClient(jid.JID{}, "pw"); err == nil {
		t.Error("NewClient with zero JID should fail")
	}
	if _, err := NewClient(jid.MustParse("example.com"), "pw"); err == nil {
		t.Error("NewClient with domain-only JID (no localpart) should fail")
	}
	if _, err := NewClient(jid.MustParse("alice@example.com"), "pw"); err != nil {
		t.Errorf("NewClient with valid JID should succeed, got %v", err)
	}
}
