package xmpp

import (
	"context"
	"testing"
	"time"

	"github.com/meszmate/xmpp-go/jid"
	"github.com/meszmate/xmpp-go/stanza"
)

// TestStreamManagementAck verifies XEP-0198 is negotiated and that a client can
// obtain a server acknowledgement (h) covering the stanzas it sent.
func TestStreamManagementAck(t *testing.T) {
	_, addr, store := startServer(t)
	addUser(t, store, "alice", "pw")

	c := mustConnect(t, addr, "alice", "pw")
	if !c.Session().SMEnabled() {
		t.Fatal("Stream Management should be enabled after connect")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	const n = 3
	for i := 0; i < n; i++ {
		m := stanza.NewMessage(stanza.MessageChat)
		m.To = jid.MustParse("bob@" + testDomain)
		m.Body = "x"
		if err := c.Send(ctx, m); err != nil {
			t.Fatalf("send %d: %v", i, err)
		}
	}

	h, err := c.Session().RequestSMAck(ctx)
	if err != nil {
		t.Fatalf("RequestSMAck: %v", err)
	}
	if h < n {
		t.Errorf("server acknowledged h=%d, want >= %d", h, n)
	}
}

func TestStreamManagementCanBeDisabled(t *testing.T) {
	_, addr, store := startServer(t)
	addUser(t, store, "alice", "pw")

	c, err := NewClient(jid.MustParse("alice@"+testDomain), "pw",
		WithConnectAddr(addr), WithoutStreamManagement())
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := c.Connect(ctx); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	if c.Session().SMEnabled() {
		t.Error("SM should be disabled with WithoutStreamManagement")
	}
}
