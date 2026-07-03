package xmpp

import (
	"context"
	"encoding/xml"
	"sync/atomic"
	"testing"
	"time"

	"github.com/meszmate/xmpp-go/internal/ns"
	"github.com/meszmate/xmpp-go/jid"
	"github.com/meszmate/xmpp-go/plugin"
	"github.com/meszmate/xmpp-go/stanza"
)

const echoNS = "urn:xmpp:echo:test"

// echoQuery is the payload the echo plugin answers.
type echoQuery struct {
	XMLName xml.Name `xml:"urn:xmpp:echo:test query"`
	Data    string   `xml:",chardata"`
}

// echoPlugin is a per-session server plugin that replies to <query
// xmlns='urn:xmpp:echo:test'> IQ gets with "pong:<data>". It records how many
// requests this instance handled to prove per-session isolation.
type echoPlugin struct {
	params   plugin.InitParams
	handled  int32
	instance int
}

func (p *echoPlugin) Name() string           { return "echo" }
func (p *echoPlugin) Version() string        { return "1.0" }
func (p *echoPlugin) Dependencies() []string { return nil }
func (p *echoPlugin) Close() error           { return nil }
func (p *echoPlugin) Handled() int32         { return atomic.LoadInt32(&p.handled) }

func (p *echoPlugin) Initialize(ctx context.Context, params plugin.InitParams) error {
	p.params = params
	params.Handle(xml.Name{Space: echoNS, Local: "query"}, "", func(ctx context.Context, st stanza.Stanza) error {
		iq, ok := st.(*stanza.IQ)
		if !ok || iq.Type != stanza.IQGet {
			return nil
		}
		atomic.AddInt32(&p.handled, 1)
		var q echoQuery
		_ = xml.Unmarshal(iq.Query, &q)
		res := stanza.IQ{Header: stanza.Header{ID: iq.ID, Type: stanza.IQResult, To: iq.From}}
		payload := &stanza.IQPayload{IQ: res, Payload: &echoQuery{Data: "pong:" + q.Data}}
		return params.SendElement(ctx, payload)
	})
	return nil
}

// TestServerPluginPerSessionDispatch proves a factory-provided plugin handles an
// inbound custom IQ on the server and that each session gets its own instance.
func TestServerPluginPerSessionDispatch(t *testing.T) {
	var instances int32
	var lastPlugin atomic.Value // *echoPlugin

	factory := func() []plugin.Plugin {
		n := atomic.AddInt32(&instances, 1)
		p := &echoPlugin{instance: int(n)}
		lastPlugin.Store(p)
		return []plugin.Plugin{p}
	}

	_, addr, store := startServer(t, WithServerPluginFactory(factory))
	addUser(t, store, "alice", "pw")

	alice, err := NewClient(jid.MustParse("alice@"+testDomain), "pw", WithConnectAddr(addr))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer alice.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := alice.Connect(ctx); err != nil {
		t.Fatalf("Connect: %v", err)
	}

	to, _ := jid.New("", testDomain, "")
	req := stanza.NewIQ(stanza.IQGet)
	req.To = to
	req.Query = []byte(`<query xmlns='` + echoNS + `'>hello</query>`)

	res, err := alice.SendIQ(ctx, req)
	if err != nil {
		t.Fatalf("SendIQ to echo plugin: %v", err)
	}
	if res.Type != stanza.IQResult {
		t.Fatalf("echo reply type = %q, want result", res.Type)
	}
	var got echoQuery
	if err := xml.Unmarshal(res.Query, &got); err != nil {
		t.Fatalf("unmarshal echo result: %v", err)
	}
	if got.Data != "pong:hello" {
		t.Errorf("echo returned %q, want %q", got.Data, "pong:hello")
	}
	if n := atomic.LoadInt32(&instances); n != 1 {
		t.Errorf("factory called %d times for 1 session, want 1", n)
	}
}

// TestServerPluginDoesNotBreakBuiltins confirms built-in service IQs (disco)
// still work when a plugin factory is configured and the plugin does not claim
// that namespace.
func TestServerPluginDoesNotBreakBuiltins(t *testing.T) {
	factory := func() []plugin.Plugin { return []plugin.Plugin{&echoPlugin{}} }
	_, addr, store := startServer(t, WithServerPluginFactory(factory))
	addUser(t, store, "alice", "pw")

	alice, _ := NewClient(jid.MustParse("alice@"+testDomain), "pw", WithConnectAddr(addr))
	defer alice.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := alice.Connect(ctx); err != nil {
		t.Fatalf("Connect: %v", err)
	}

	to, _ := jid.New("", testDomain, "")
	req := stanza.NewIQ(stanza.IQGet)
	req.To = to
	req.Query = []byte(`<query xmlns='` + ns.DiscoInfo + `'/>`)

	res, err := alice.SendIQ(ctx, req)
	if err != nil {
		t.Fatalf("disco SendIQ: %v", err)
	}
	if res.Type != stanza.IQResult {
		t.Fatalf("disco reply type = %q, want result", res.Type)
	}
	if name := iqPayloadName(res); name.Space != ns.DiscoInfo {
		t.Errorf("disco reply payload ns = %q, want %q", name.Space, ns.DiscoInfo)
	}
}

// TestServerPluginIsolationAcrossSessions verifies two concurrent sessions each
// receive an independent plugin instance (fresh state per session).
func TestServerPluginIsolationAcrossSessions(t *testing.T) {
	var instances int32
	factory := func() []plugin.Plugin {
		atomic.AddInt32(&instances, 1)
		return []plugin.Plugin{&echoPlugin{}}
	}
	_, addr, store := startServer(t, WithServerPluginFactory(factory))
	addUser(t, store, "alice", "pw")
	addUser(t, store, "bob", "pw")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	connect := func(user string) *Client {
		c, _ := NewClient(jid.MustParse(user+"@"+testDomain), "pw", WithConnectAddr(addr))
		if err := c.Connect(ctx); err != nil {
			t.Fatalf("%s connect: %v", user, err)
		}
		return c
	}
	alice := connect("alice")
	defer alice.Close()
	bob := connect("bob")
	defer bob.Close()

	// Each session must have triggered its own factory call.
	if n := atomic.LoadInt32(&instances); n != 2 {
		t.Errorf("factory called %d times for 2 sessions, want 2", n)
	}

	// Both sessions can independently reach their own plugin.
	to, _ := jid.New("", testDomain, "")
	for _, c := range []*Client{alice, bob} {
		req := stanza.NewIQ(stanza.IQGet)
		req.To = to
		req.Query = []byte(`<query xmlns='` + echoNS + `'>x</query>`)
		res, err := c.SendIQ(ctx, req)
		if err != nil {
			t.Fatalf("SendIQ: %v", err)
		}
		var got echoQuery
		_ = xml.Unmarshal(res.Query, &got)
		if got.Data != "pong:x" {
			t.Errorf("echo = %q, want pong:x", got.Data)
		}
	}
}
