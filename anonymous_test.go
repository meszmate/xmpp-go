package xmpp

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/meszmate/xmpp-go/jid"
)

func TestAnonymousAuth(t *testing.T) {
	_, addr, _ := startServer(t, WithServerAnonymous())

	// Anonymous clients connect with just a domain and no credentials.
	c, err := NewClient(
		jid.MustParse(testDomain),
		"",
		WithConnectAddr(addr),
		WithSASLMechanisms("ANONYMOUS"),
	)
	if err != nil {
		t.Fatalf("NewClient (anonymous, domain-only JID): %v", err)
	}
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := c.Connect(ctx); err != nil {
		t.Fatalf("anonymous Connect failed: %v", err)
	}

	local := c.Session().LocalAddr()
	if !strings.HasPrefix(local.Local(), "anon-") {
		t.Errorf("expected an anon- localpart, got %q", local)
	}
	if local.Resource() == "" {
		t.Errorf("expected a bound resource, got %q", local)
	}
	if c.Session().State()&StateReady == 0 {
		t.Error("session not Ready after anonymous auth")
	}
}

func TestAnonymousRefusedWhenDisabled(t *testing.T) {
	// Server does NOT enable anonymous; the client only offers ANONYMOUS.
	_, addr, store := startServer(t)
	addUser(t, store, "alice", "pw")

	c, _ := NewClient(
		jid.MustParse(testDomain),
		"",
		WithConnectAddr(addr),
		WithSASLMechanisms("ANONYMOUS"),
	)
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := c.Connect(ctx); err == nil {
		t.Fatal("anonymous Connect should fail when the server disables ANONYMOUS")
	}
}
