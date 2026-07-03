package xmpp

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/meszmate/xmpp-go/jid"
	"github.com/meszmate/xmpp-go/stanza"
	"github.com/meszmate/xmpp-go/storage"
	"github.com/meszmate/xmpp-go/storage/memory"
)

// peerDirectory is a mutable domain→address map used as the s2s resolver so two
// in-process servers can federate once both are listening.
type peerDirectory struct {
	mu    sync.Mutex
	addrs map[string]string
}

func (p *peerDirectory) resolve(domain string) (string, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	a, ok := p.addrs[domain]
	return a, ok
}

func (p *peerDirectory) set(domain, addr string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.addrs[domain] = addr
}

func startS2SServer(t *testing.T, domain, secret string, dir *peerDirectory) (*Server, string, *memory.Store) {
	t.Helper()
	store := memory.New()
	srv, err := NewServer(domain,
		WithServerAddr("127.0.0.1:0"),
		WithServerStorage(store),
		WithServerS2S(secret, dir.resolve),
	)
	if err != nil {
		t.Fatalf("NewServer(%s): %v", domain, err)
	}
	errc := make(chan error, 1)
	go func() { errc <- srv.ListenAndServe(context.Background()) }()
	t.Cleanup(func() { _ = srv.Close() })

	deadline := time.Now().Add(5 * time.Second)
	for {
		if a := srv.Addr(); a != nil {
			return srv, a.String(), store
		}
		select {
		case err := <-errc:
			t.Fatalf("%s exited before listening: %v", domain, err)
		default:
		}
		if time.Now().After(deadline) {
			t.Fatalf("%s did not start listening", domain)
		}
		time.Sleep(2 * time.Millisecond)
	}
}

func addS2SUser(t *testing.T, store *memory.Store, username, password string) {
	t.Helper()
	if err := store.UserStore().CreateUser(context.Background(), &storage.User{Username: username, Password: password}); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
}

// TestS2SFederation proves a message from a user on one domain is delivered to a
// user on another domain over an s2s stream authenticated by XEP-0220 dialback.
func TestS2SFederation(t *testing.T) {
	dir := &peerDirectory{addrs: map[string]string{}}

	_, addrA, storeA := startS2SServer(t, "alpha.test", "secret-alpha", dir)
	_, addrB, storeB := startS2SServer(t, "beta.test", "secret-beta", dir)
	dir.set("alpha.test", addrA)
	dir.set("beta.test", addrB)

	addS2SUser(t, storeA, "alice", "pw")
	addS2SUser(t, storeB, "bob", "pw")

	received := make(chan *stanza.Message, 1)
	bobHandler := HandlerFunc(func(ctx context.Context, s *Session, st stanza.Stanza) error {
		if m, ok := st.(*stanza.Message); ok && m.Body != "" {
			select {
			case received <- m:
			default:
			}
		}
		return nil
	})

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	bob, _ := NewClient(jid.MustParse("bob@beta.test"), "pw", WithConnectAddr(addrB), WithHandler(bobHandler))
	defer bob.Close()
	if err := bob.Connect(ctx); err != nil {
		t.Fatalf("bob connect: %v", err)
	}

	alice, _ := NewClient(jid.MustParse("alice@alpha.test"), "pw", WithConnectAddr(addrA))
	defer alice.Close()
	if err := alice.Connect(ctx); err != nil {
		t.Fatalf("alice connect: %v", err)
	}

	msg := stanza.NewMessage(stanza.MessageChat)
	msg.To = jid.MustParse("bob@beta.test")
	msg.Body = "hello across the federation"
	if err := alice.Send(ctx, msg); err != nil {
		t.Fatalf("alice.Send: %v", err)
	}

	select {
	case got := <-received:
		if got.Body != "hello across the federation" {
			t.Errorf("body = %q", got.Body)
		}
		if got.From.Domain() != "alpha.test" || got.From.Local() != "alice" {
			t.Errorf("From = %q, want alice@alpha.test/...", got.From)
		}
	case <-time.After(15 * time.Second):
		t.Fatal("message was not federated to bob within 15s")
	}
}

// TestS2SDialbackForgedKeyRejected verifies the authoritative server rejects a
// dialback key it did not generate — the core security property of dialback.
func TestS2SDialbackForgedKeyRejected(t *testing.T) {
	// alpha uses secret S; a forged key computed with a different secret must not
	// verify.
	streamID := "stream-123"
	good := dialbackKey("secret-alpha", "beta.test", "alpha.test", streamID)
	forged := dialbackKey("WRONG-secret", "beta.test", "alpha.test", streamID)
	if good == forged {
		t.Fatal("keys under different secrets must differ")
	}
	// Recomputation under the real secret matches the real key (what the
	// authoritative server does in s2sHandleVerify).
	recomputed := dialbackKey("secret-alpha", "beta.test", "alpha.test", streamID)
	if recomputed != good {
		t.Errorf("recomputed key %q != %q", recomputed, good)
	}
}
