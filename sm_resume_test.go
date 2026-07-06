package xmpp

import (
	"context"
	"testing"
	"time"

	"github.com/meszmate/xmpp-go/jid"
	"github.com/meszmate/xmpp-go/stanza"
)

// waitFor polls cond until it is true or the deadline elapses.
func waitFor(t *testing.T, d time.Duration, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatalf("condition not met within %s", d)
}

// TestStreamResumptionNegotiated verifies the client obtains a resumption id.
func TestStreamResumptionNegotiated(t *testing.T) {
	_, addr, store := startServer(t)
	addUser(t, store, "alice", "pw")

	c, _ := NewClient(jid.MustParse("alice@"+testDomain), "pw", WithConnectAddr(addr))
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := c.Connect(ctx); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	if !c.Session().SMEnabled() {
		t.Fatal("stream management not enabled")
	}
	if c.Session().SMPrevID() == "" {
		t.Fatal("no resumption id negotiated")
	}
}

// TestStreamResumptionReplaysBufferedStanza is the core resumption test: a
// client drops, a message sent while it is gone is buffered by the server, and
// on resume the same bound resource is restored and the buffered message is
// replayed.
func TestStreamResumptionReplaysBufferedStanza(t *testing.T) {
	srv, addr, store := startServer(t)
	addUser(t, store, "alice", "pw")
	addUser(t, store, "bob", "pw")

	received := make(chan string, 4)
	bobHandler := HandlerFunc(func(ctx context.Context, s *Session, st stanza.Stanza) error {
		if m, ok := st.(*stanza.Message); ok && m.Body != "" {
			received <- m.Body
		}
		return nil
	})

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	bob, _ := NewClient(jid.MustParse("bob@"+testDomain), "pw", WithConnectAddr(addr), WithHandler(bobHandler))
	defer bob.Close()
	if err := bob.Connect(ctx); err != nil {
		t.Fatalf("bob.Connect: %v", err)
	}
	bobJID := bob.Session().LocalAddr()
	if bob.Session().SMPrevID() == "" {
		t.Fatal("bob did not negotiate resumption")
	}

	alice, _ := NewClient(jid.MustParse("alice@"+testDomain), "pw", WithConnectAddr(addr))
	defer alice.Close()
	if err := alice.Connect(ctx); err != nil {
		t.Fatalf("alice.Connect: %v", err)
	}

	// Abruptly drop bob's connection (as if the network died) without a clean
	// close, so the server parks his session for resumption.
	_ = bob.Session().Transport().Close()

	// The server should park bob's session.
	waitFor(t, 5*time.Second, func() bool { return srv.numDetached() == 1 })

	// Alice messages bob while he is disconnected: the server buffers it.
	msg := stanza.NewMessage(stanza.MessageChat)
	msg.To = bobJID
	msg.Body = "while you were gone"
	if err := alice.Send(ctx, msg); err != nil {
		t.Fatalf("alice.Send during bob outage: %v", err)
	}

	// Give the buffered delivery a moment to land in the holder's queue, then
	// resume bob.
	waitFor(t, 5*time.Second, func() bool { return srv.numDetached() == 1 })
	if err := bob.Resume(ctx); err != nil {
		t.Fatalf("bob.Resume: %v", err)
	}

	// Resumption must preserve bob's bound resource.
	if got := bob.Session().LocalAddr(); !got.Equal(bobJID) {
		t.Errorf("resumed JID = %q, want %q", got, bobJID)
	}
	// The parked session is consumed by the resume.
	waitFor(t, 5*time.Second, func() bool { return srv.numDetached() == 0 })

	// The buffered message must have been replayed to bob.
	select {
	case body := <-received:
		if body != "while you were gone" {
			t.Errorf("replayed body = %q", body)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("buffered message was not replayed after resume")
	}

	// And the resumed session keeps working for new traffic.
	msg2 := stanza.NewMessage(stanza.MessageChat)
	msg2.To = bobJID
	msg2.Body = "after resume"
	if err := alice.Send(ctx, msg2); err != nil {
		t.Fatalf("alice.Send after resume: %v", err)
	}
	select {
	case body := <-received:
		if body != "after resume" {
			t.Errorf("post-resume body = %q", body)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("post-resume message not delivered")
	}
}

// TestStreamResumptionExpires verifies a parked session is torn down after the
// resumption window elapses, freeing the resource and dropping buffered stanzas.
func TestStreamResumptionExpires(t *testing.T) {
	srv, addr, store := startServer(t, WithServerResumeTimeout(150*time.Millisecond))
	addUser(t, store, "alice", "pw")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	c, _ := NewClient(jid.MustParse("alice@"+testDomain), "pw", WithConnectAddr(addr))
	if err := c.Connect(ctx); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	aliceJID := c.Session().LocalAddr()
	_ = c.Session().Transport().Close()
	waitFor(t, 5*time.Second, func() bool { return srv.numDetached() == 1 })

	// After the (short) resumption window, the parked session is reaped.
	waitFor(t, 5*time.Second, func() bool { return srv.numDetached() == 0 })

	// The resource is no longer routable.
	if srv.router.online(aliceJID.Bare()) {
		t.Error("resource still routable after resumption window expired")
	}

	// Resuming after expiry must fail (server no longer has the state), and the
	// client can fall back to a fresh connect.
	if err := c.Resume(ctx); err == nil {
		// Resume may return nil if the server declined and a fresh bind
		// succeeded; in that case the JID should have a (new) resource.
		if c.Session() == nil || c.Session().LocalAddr().Resource() == "" {
			t.Error("post-expiry Resume returned nil without a bound session")
		}
	}
	_ = c.Close()
}
