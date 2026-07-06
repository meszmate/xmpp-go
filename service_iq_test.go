package xmpp

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/meszmate/xmpp-go/jid"
	"github.com/meszmate/xmpp-go/stanza"
)

func mustConnect(t *testing.T, addr, user, pass string, opts ...ClientOption) *Client {
	t.Helper()
	all := append([]ClientOption{WithConnectAddr(addr)}, opts...)
	c, err := NewClient(jid.MustParse(user+"@"+testDomain), pass, all...)
	if err != nil {
		t.Fatalf("NewClient(%s): %v", user, err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := c.Connect(ctx); err != nil {
		t.Fatalf("%s.Connect: %v", user, err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return c
}

func TestServerAnswersPing(t *testing.T) {
	_, addr, store := startServer(t)
	addUser(t, store, "alice", "pw")
	c := mustConnect(t, addr, "alice", "pw")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ping := stanza.NewIQ(stanza.IQGet)
	ping.To = jid.MustParse(testDomain)
	ping.Query = []byte(`<ping xmlns='urn:xmpp:ping'/>`)

	res, err := c.SendIQ(ctx, ping)
	if err != nil {
		t.Fatalf("ping: %v", err)
	}
	if res.Type != stanza.IQResult {
		t.Errorf("ping reply type = %q, want result", res.Type)
	}
}

func TestServerAnswersDiscoInfo(t *testing.T) {
	_, addr, store := startServer(t)
	addUser(t, store, "alice", "pw")
	c := mustConnect(t, addr, "alice", "pw")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	q := stanza.NewIQ(stanza.IQGet)
	q.To = jid.MustParse(testDomain)
	q.Query = []byte(`<query xmlns='http://jabber.org/protocol/disco#info'/>`)

	res, err := c.SendIQ(ctx, q)
	if err != nil {
		t.Fatalf("disco#info: %v", err)
	}
	if res.Type != stanza.IQResult {
		t.Fatalf("disco#info reply type = %q, want result", res.Type)
	}
	if !bytes.Contains(res.Query, []byte("urn:xmpp:ping")) {
		t.Errorf("disco#info result missing ping feature: %s", res.Query)
	}
	if !bytes.Contains(res.Query, []byte(`category="server"`)) {
		t.Errorf("disco#info result missing server identity: %s", res.Query)
	}
}

func TestServerAnswersVersion(t *testing.T) {
	_, addr, store := startServer(t)
	addUser(t, store, "alice", "pw")
	c := mustConnect(t, addr, "alice", "pw")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	q := stanza.NewIQ(stanza.IQGet)
	q.To = jid.MustParse(testDomain)
	q.Query = []byte(`<query xmlns='jabber:iq:version'/>`)

	res, err := c.SendIQ(ctx, q)
	if err != nil {
		t.Fatalf("version: %v", err)
	}
	if !bytes.Contains(res.Query, []byte("xmpp-go")) {
		t.Errorf("version result missing name: %s", res.Query)
	}
}

func TestServerRosterSetAndGet(t *testing.T) {
	_, addr, store := startServer(t)
	addUser(t, store, "alice", "pw")
	c := mustConnect(t, addr, "alice", "pw")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Roster set: add a contact.
	set := stanza.NewIQ(stanza.IQSet)
	set.Query = []byte(`<query xmlns='jabber:iq:roster'><item jid='bob@localhost' name='Bob'/></query>`)
	res, err := c.SendIQ(ctx, set)
	if err != nil {
		t.Fatalf("roster set: %v", err)
	}
	if res.Type != stanza.IQResult {
		t.Fatalf("roster set reply = %q, want result", res.Type)
	}

	// Roster get: the contact must come back.
	get := stanza.NewIQ(stanza.IQGet)
	get.Query = []byte(`<query xmlns='jabber:iq:roster'/>`)
	res, err = c.SendIQ(ctx, get)
	if err != nil {
		t.Fatalf("roster get: %v", err)
	}
	if !bytes.Contains(res.Query, []byte("bob@localhost")) {
		t.Errorf("roster get missing the added contact: %s", res.Query)
	}
}

// TestClientAutoRespondsToPing routes a ping from alice to bob's full JID; bob's
// client must auto-answer with a pong, delivered back to alice.
func TestClientAutoRespondsToPing(t *testing.T) {
	_, addr, store := startServer(t)
	addUser(t, store, "alice", "pw")
	addUser(t, store, "bob", "pw")

	alice := mustConnect(t, addr, "alice", "pw")
	bob := mustConnect(t, addr, "bob", "pw")

	bobJID := bob.Session().LocalAddr()
	if bobJID.Resource() == "" {
		t.Fatal("bob not bound to a full JID")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ping := stanza.NewIQ(stanza.IQGet)
	ping.To = bobJID
	ping.Query = []byte(`<ping xmlns='urn:xmpp:ping'/>`)

	res, err := alice.SendIQ(ctx, ping)
	if err != nil {
		t.Fatalf("client-to-client ping: %v", err)
	}
	if res.Type != stanza.IQResult {
		t.Errorf("pong type = %q, want result", res.Type)
	}
}
