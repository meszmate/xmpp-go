package xmpp

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/meszmate/xmpp-go/jid"
	"github.com/meszmate/xmpp-go/stanza"
)

// TestBOSHConnect proves a client can complete full negotiation (SASL + bind)
// over the BOSH HTTP transport.
func TestBOSHConnect(t *testing.T) {
	srv, _, memStore := startServer(t)
	addUser(t, memStore, "alice", "pw")

	httpSrv := httptest.NewServer(srv.BOSHHandler())
	t.Cleanup(httpSrv.Close)

	c, err := NewClient(jid.MustParse("alice@"+testDomain), "pw", WithBOSH(httpSrv.URL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := c.Connect(ctx); err != nil {
		t.Fatalf("BOSH Connect failed: %v", err)
	}
	if c.Session().State()&StateReady == 0 {
		t.Errorf("session not Ready after BOSH negotiation: state=%b", c.Session().State())
	}
	if c.Session().LocalAddr().Resource() == "" {
		t.Errorf("expected a bound full JID, got %q", c.Session().LocalAddr())
	}
	if c.Session().LocalAddr().Local() != "alice" {
		t.Errorf("bound localpart = %q, want alice", c.Session().LocalAddr().Local())
	}
}

// TestBOSHMessaging proves stanzas flow both directions over BOSH: one client
// sends, another (long-polling) receives.
func TestBOSHMessaging(t *testing.T) {
	srv, _, memStore := startServer(t)
	addUser(t, memStore, "alice", "pw")
	addUser(t, memStore, "bob", "pw")

	httpSrv := httptest.NewServer(srv.BOSHHandler())
	t.Cleanup(httpSrv.Close)

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

	bob, _ := NewClient(jid.MustParse("bob@"+testDomain), "pw", WithBOSH(httpSrv.URL), WithHandler(bobHandler))
	defer bob.Close()
	alice, _ := NewClient(jid.MustParse("alice@"+testDomain), "pw", WithBOSH(httpSrv.URL))
	defer alice.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := bob.Connect(ctx); err != nil {
		t.Fatalf("bob BOSH connect: %v", err)
	}
	if err := alice.Connect(ctx); err != nil {
		t.Fatalf("alice BOSH connect: %v", err)
	}

	msg := stanza.NewMessage(stanza.MessageChat)
	msg.To = jid.MustParse("bob@" + testDomain)
	msg.Body = "hello over bosh"
	if err := alice.Send(ctx, msg); err != nil {
		t.Fatalf("alice.Send: %v", err)
	}

	select {
	case body := <-received:
		if body != "hello over bosh" {
			t.Errorf("bob got %q", body)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("message did not arrive over BOSH")
	}
}

// TestBOSHServiceIQ proves request/response IQs (disco) work over BOSH.
func TestBOSHServiceIQ(t *testing.T) {
	srv, _, memStore := startServer(t)
	addUser(t, memStore, "alice", "pw")

	httpSrv := httptest.NewServer(srv.BOSHHandler())
	t.Cleanup(httpSrv.Close)

	c, _ := NewClient(jid.MustParse("alice@"+testDomain), "pw", WithBOSH(httpSrv.URL))
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := c.Connect(ctx); err != nil {
		t.Fatalf("Connect: %v", err)
	}

	to, _ := jid.New("", testDomain, "")
	req := stanza.NewIQ(stanza.IQGet)
	req.To = to
	req.Query = []byte(`<query xmlns='http://jabber.org/protocol/disco#info'/>`)

	res, err := c.SendIQ(ctx, req)
	if err != nil {
		t.Fatalf("disco over BOSH: %v", err)
	}
	if res.Type != stanza.IQResult {
		t.Fatalf("disco reply type = %q, want result", res.Type)
	}
}
