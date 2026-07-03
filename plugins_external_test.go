package xmpp_test

import (
	"bytes"
	"context"
	"testing"
	"time"

	xmpp "github.com/meszmate/xmpp-go"
	"github.com/meszmate/xmpp-go/jid"
	"github.com/meszmate/xmpp-go/plugins/disco"
	"github.com/meszmate/xmpp-go/plugins/ping"
	"github.com/meszmate/xmpp-go/plugins/roster"
	"github.com/meszmate/xmpp-go/plugins/version"
	"github.com/meszmate/xmpp-go/stanza"
	"github.com/meszmate/xmpp-go/storage"
	"github.com/meszmate/xmpp-go/storage/memory"
)

func mkUser(t *testing.T, store *memory.Store, u, p string) {
	t.Helper()
	if err := store.UserStore().CreateUser(context.Background(), &storage.User{Username: u, Password: p}); err != nil {
		t.Fatalf("CreateUser(%s): %v", u, err)
	}
}

func connectClient(t *testing.T, addr, u, p string, opts ...xmpp.ClientOption) *xmpp.Client {
	t.Helper()
	all := append([]xmpp.ClientOption{xmpp.WithConnectAddr(addr)}, opts...)
	c, err := xmpp.NewClient(jid.MustParse(u+"@localhost"), p, all...)
	if err != nil {
		t.Fatalf("NewClient(%s): %v", u, err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := c.Connect(ctx); err != nil {
		t.Fatalf("%s.Connect: %v", u, err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return c
}

// TestDiscoPluginRespondsToQuery proves a client-side disco plugin now actually
// answers an incoming disco#info query (previously it was send-only/dead).
func TestDiscoPluginRespondsToQuery(t *testing.T) {
	addr, store := startExampleServer(t)
	mkUser(t, store, "alice", "pw")
	mkUser(t, store, "bob", "pw")

	d := disco.New()
	d.AddIdentity(disco.Identity{Category: "client", Type: "pc", Name: "TestClient"})
	d.AddFeature("urn:xmpp:ping")

	alice := connectClient(t, addr, "alice", "pw", xmpp.WithPlugins(d))
	bob := connectClient(t, addr, "bob", "pw")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req := stanza.NewIQ(stanza.IQGet)
	req.To = alice.Session().LocalAddr()
	req.Query = []byte(`<query xmlns='http://jabber.org/protocol/disco#info'/>`)

	res, err := bob.SendIQ(ctx, req)
	if err != nil {
		t.Fatalf("disco query: %v", err)
	}
	if res.Type != stanza.IQResult {
		t.Fatalf("disco reply type = %q, want result", res.Type)
	}
	if !bytes.Contains(res.Query, []byte(`category="client"`)) {
		t.Errorf("disco result missing identity: %s", res.Query)
	}
	if !bytes.Contains(res.Query, []byte("urn:xmpp:ping")) {
		t.Errorf("disco result missing added feature: %s", res.Query)
	}
}

func TestVersionPluginRespondsToQuery(t *testing.T) {
	addr, store := startExampleServer(t)
	mkUser(t, store, "alice", "pw")
	mkUser(t, store, "bob", "pw")

	alice := connectClient(t, addr, "alice", "pw", xmpp.WithPlugins(version.New("TestApp", "9.9")))
	bob := connectClient(t, addr, "bob", "pw")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req := stanza.NewIQ(stanza.IQGet)
	req.To = alice.Session().LocalAddr()
	req.Query = []byte(`<query xmlns='jabber:iq:version'/>`)

	res, err := bob.SendIQ(ctx, req)
	if err != nil {
		t.Fatalf("version query: %v", err)
	}
	if !bytes.Contains(res.Query, []byte("TestApp")) || !bytes.Contains(res.Query, []byte("9.9")) {
		t.Errorf("version result missing app info: %s", res.Query)
	}
}

// TestPingPluginRoundTrip uses the ping plugin's client API to ping another
// client, whose built-in auto-pong (and ping plugin) replies.
func TestPingPluginRoundTrip(t *testing.T) {
	addr, store := startExampleServer(t)
	mkUser(t, store, "alice", "pw")
	mkUser(t, store, "bob", "pw")

	pp := ping.New()
	_ = connectClient(t, addr, "alice", "pw", xmpp.WithPlugins(pp))
	bob := connectClient(t, addr, "bob", "pw", xmpp.WithPlugins(ping.New()))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := pp.Ping(ctx, bob.Session().LocalAddr()); err != nil {
		t.Fatalf("ping plugin round trip failed: %v", err)
	}
}

// TestRosterPluginFetch proves the roster plugin can now fetch the server roster
// (previously it was a local-only shell).
func TestRosterPluginFetch(t *testing.T) {
	addr, store := startExampleServer(t)
	mkUser(t, store, "alice", "pw")

	rp := roster.New()
	alice := connectClient(t, addr, "alice", "pw", xmpp.WithPlugins(rp))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Add a contact via a roster set to the server.
	set := stanza.NewIQ(stanza.IQSet)
	set.Query = []byte(`<query xmlns='jabber:iq:roster'><item jid='carol@localhost' name='Carol'/></query>`)
	if _, err := alice.SendIQ(ctx, set); err != nil {
		t.Fatalf("roster set: %v", err)
	}

	items, err := rp.Fetch(ctx)
	if err != nil {
		t.Fatalf("roster fetch: %v", err)
	}
	found := false
	for _, it := range items {
		if it.JID == "carol@localhost" {
			found = true
		}
	}
	if !found {
		t.Errorf("roster fetch did not return the added contact: %+v", items)
	}
}
