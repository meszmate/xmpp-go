package xmpp

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/meszmate/xmpp-go/jid"
	"github.com/meszmate/xmpp-go/stanza"
)

// stanzaCapture collects inbound stanzas for assertions in presence tests.
type stanzaCapture struct {
	ch chan stanza.Stanza
}

func newCapture() *stanzaCapture { return &stanzaCapture{ch: make(chan stanza.Stanza, 64)} }

func (c *stanzaCapture) handler() Handler {
	return HandlerFunc(func(ctx context.Context, s *Session, st stanza.Stanza) error {
		select {
		case c.ch <- st:
		default:
		}
		return nil
	})
}

func (c *stanzaCapture) wait(t *testing.T, what string, pred func(stanza.Stanza) bool) stanza.Stanza {
	t.Helper()
	deadline := time.After(5 * time.Second)
	for {
		select {
		case st := <-c.ch:
			if pred(st) {
				return st
			}
		case <-deadline:
			t.Fatalf("timed out waiting for %s", what)
			return nil
		}
	}
}

// expectAll waits until every named predicate has matched at least one inbound
// stanza, in any order (stanzas can race).
func (c *stanzaCapture) expectAll(t *testing.T, checks map[string]func(stanza.Stanza) bool) {
	t.Helper()
	seen := make(map[string]bool, len(checks))
	remaining := len(checks)
	deadline := time.After(5 * time.Second)
	for remaining > 0 {
		select {
		case st := <-c.ch:
			for name, pred := range checks {
				if !seen[name] && pred(st) {
					seen[name] = true
					remaining--
				}
			}
		case <-deadline:
			var missing []string
			for name := range checks {
				if !seen[name] {
					missing = append(missing, name)
				}
			}
			t.Fatalf("timed out; missing: %v", missing)
		}
	}
}

func connectCap(t *testing.T, addr, user, pass string) (*Client, *stanzaCapture) {
	t.Helper()
	cap := newCapture()
	c, err := NewClient(jid.MustParse(user+"@"+testDomain), pass,
		WithConnectAddr(addr), WithHandler(cap.handler()))
	if err != nil {
		t.Fatalf("NewClient(%s): %v", user, err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := c.Connect(ctx); err != nil {
		t.Fatalf("%s.Connect: %v", user, err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return c, cap
}

// TestPresenceSubscriptionFlow exercises the full RFC 6121 subscribe/approve
// flow: routing, roster updates, roster pushes, and presence broadcast.
func TestPresenceSubscriptionFlow(t *testing.T) {
	_, addr, store := startServer(t)
	addUser(t, store, "alice", "pw")
	addUser(t, store, "bob", "pw")

	alice, aliceCap := connectCap(t, addr, "alice", "pw")
	bob, bobCap := connectCap(t, addr, "bob", "pw")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	isPresence := func(typ string, fromLocal string) func(stanza.Stanza) bool {
		return func(st stanza.Stanza) bool {
			p, ok := st.(*stanza.Presence)
			return ok && p.Type == typ && p.From.Local() == fromLocal
		}
	}
	isRosterPush := func(contact, sub string) func(stanza.Stanza) bool {
		return func(st stanza.Stanza) bool {
			iq, ok := st.(*stanza.IQ)
			if !ok || iq.Type != stanza.IQSet {
				return false
			}
			return bytes.Contains(iq.Query, []byte("jabber:iq:roster")) &&
				bytes.Contains(iq.Query, []byte(contact)) &&
				bytes.Contains(iq.Query, []byte(`subscription="`+sub+`"`))
		}
	}

	// 1. alice subscribes to bob.
	sub := stanza.NewPresence(stanza.PresenceSubscribe)
	sub.To = jid.MustParse("bob@" + testDomain)
	if err := alice.Send(ctx, sub); err != nil {
		t.Fatal(err)
	}
	bobCap.wait(t, "subscribe from alice", isPresence(stanza.PresenceSubscribe, "alice"))

	// 2. bob approves.
	subd := stanza.NewPresence(stanza.PresenceSubscribed)
	subd.To = jid.MustParse("alice@" + testDomain)
	if err := bob.Send(ctx, subd); err != nil {
		t.Fatal(err)
	}
	// alice must receive both the subscribed presence and a roster push showing
	// bob with subscription=to (these two race, so check order-independently).
	aliceCap.expectAll(t, map[string]func(stanza.Stanza) bool{
		"subscribed from bob": isPresence(stanza.PresenceSubscribed, "bob"),
		"roster push bob=to":  isRosterPush("bob@"+testDomain, "to"),
	})

	// 3. bob broadcasts available presence; alice (subscribed) must receive it.
	avail := stanza.NewPresence(stanza.PresenceAvailable)
	avail.Status = "online"
	if err := bob.Send(ctx, avail); err != nil {
		t.Fatal(err)
	}
	got := aliceCap.wait(t, "bob available presence", isPresence(stanza.PresenceAvailable, "bob"))
	if p := got.(*stanza.Presence); p.Status != "online" {
		t.Errorf("presence status = %q, want online", p.Status)
	}
}
