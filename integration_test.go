package xmpp

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/meszmate/xmpp-go/jid"
	"github.com/meszmate/xmpp-go/stanza"
	"github.com/meszmate/xmpp-go/storage"
	"github.com/meszmate/xmpp-go/storage/memory"
)

const testDomain = "localhost"

// startServer boots a library-managed XMPP server on a random loopback port and
// returns it together with its dial address and backing store. Users must be
// added after the server is listening (Init resets the store).
func startServer(t *testing.T, opts ...ServerOption) (*Server, string, *memory.Store) {
	t.Helper()
	store := memory.New()
	baseOpts := []ServerOption{
		WithServerAddr("127.0.0.1:0"),
		WithServerStorage(store),
	}
	srv, err := NewServer(testDomain, append(baseOpts, opts...)...)
	if err != nil {
		t.Fatalf("NewServer: %v", err)
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
			t.Fatalf("server exited before listening: %v", err)
		default:
		}
		if time.Now().After(deadline) {
			t.Fatal("server did not start listening within 5s")
		}
		time.Sleep(2 * time.Millisecond)
	}
}

func addUser(t *testing.T, store *memory.Store, username, password string) {
	t.Helper()
	err := store.UserStore().CreateUser(context.Background(), &storage.User{
		Username: username,
		Password: password,
	})
	if err != nil {
		t.Fatalf("CreateUser(%s): %v", username, err)
	}
}

// TestConnectAuthenticatesAndBinds is the core regression test for the reported
// bug: correct JID + password must fully log in (SASL + bind), not silently do
// nothing.
func TestConnectAuthenticatesAndBinds(t *testing.T) {
	_, addr, store := startServer(t)
	addUser(t, store, "alice", "s3cret")

	client, err := NewClient(
		jid.MustParse("alice@"+testDomain),
		"s3cret",
		WithConnectAddr(addr),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Connect with correct credentials failed: %v", err)
	}

	sess := client.Session()
	if sess == nil {
		t.Fatal("Session() is nil after Connect")
	}
	if sess.State()&StateReady == 0 {
		t.Errorf("session not Ready after Connect: state=%b", sess.State())
	}
	bound := client.JID()
	if sess.LocalAddr().IsZero() || sess.LocalAddr().Resource() == "" {
		t.Errorf("expected a bound full JID with a resource, got %q", sess.LocalAddr())
	}
	if sess.LocalAddr().Local() != "alice" || sess.LocalAddr().Domain() != testDomain {
		t.Errorf("bound JID has wrong bare part: %q (client JID %q)", sess.LocalAddr(), bound)
	}
}

// TestConnectWrongPassword covers the reported symptom: correct JID + wrong
// password must return a clear *AuthError, not silently succeed.
func TestConnectWrongPassword(t *testing.T) {
	_, addr, store := startServer(t)
	addUser(t, store, "alice", "s3cret")

	client, err := NewClient(
		jid.MustParse("alice@"+testDomain),
		"WRONG",
		WithConnectAddr(addr),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = client.Connect(ctx)
	if err == nil {
		t.Fatal("Connect with wrong password must fail, but returned nil")
	}
	var authErr *AuthError
	if !errors.As(err, &authErr) {
		t.Fatalf("expected *AuthError, got %T: %v", err, err)
	}
	if authErr.Condition != "not-authorized" {
		t.Errorf("expected condition not-authorized, got %q", authErr.Condition)
	}
}

func TestConnectUnknownUser(t *testing.T) {
	_, addr, store := startServer(t)
	addUser(t, store, "alice", "s3cret")

	client, _ := NewClient(
		jid.MustParse("ghost@"+testDomain),
		"whatever",
		WithConnectAddr(addr),
	)
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err := client.Connect(ctx)
	var authErr *AuthError
	if !errors.As(err, &authErr) {
		t.Fatalf("expected *AuthError for unknown user, got %T: %v", err, err)
	}
}

func TestConnectDialErrorIsReported(t *testing.T) {
	// Nothing is listening on this port.
	client, _ := NewClient(
		jid.MustParse("alice@"+testDomain),
		"s3cret",
		WithConnectAddr("127.0.0.1:1"),
	)
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := client.Connect(ctx); err == nil {
		t.Fatal("Connect to a dead address must return a dial error, got nil")
	}
}

func TestConnectRequestedResource(t *testing.T) {
	_, addr, store := startServer(t)
	addUser(t, store, "alice", "s3cret")

	client, _ := NewClient(
		jid.MustParse("alice@"+testDomain),
		"s3cret",
		WithConnectAddr(addr),
		WithResource("phone"),
	)
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	if got := client.Session().LocalAddr().Resource(); got != "phone" {
		t.Errorf("requested resource 'phone', bound resource is %q", got)
	}
}

// TestMessageDeliveryBetweenClients proves the full path works: two authenticated
// clients exchange a real message through the server's router.
func TestMessageDeliveryBetweenClients(t *testing.T) {
	_, addr, store := startServer(t)
	addUser(t, store, "alice", "pw1")
	addUser(t, store, "bob", "pw2")

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

	bob, _ := NewClient(jid.MustParse("bob@"+testDomain), "pw2",
		WithConnectAddr(addr), WithHandler(bobHandler))
	defer bob.Close()
	alice, _ := NewClient(jid.MustParse("alice@"+testDomain), "pw1",
		WithConnectAddr(addr))
	defer alice.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := bob.Connect(ctx); err != nil {
		t.Fatalf("bob.Connect: %v", err)
	}
	if err := alice.Connect(ctx); err != nil {
		t.Fatalf("alice.Connect: %v", err)
	}

	msg := stanza.NewMessage(stanza.MessageChat)
	msg.To = jid.MustParse("bob@" + testDomain)
	msg.Body = "hello bob"
	if err := alice.Send(ctx, msg); err != nil {
		t.Fatalf("alice.Send: %v", err)
	}

	select {
	case got := <-received:
		if got.Body != "hello bob" {
			t.Errorf("bob received body %q, want %q", got.Body, "hello bob")
		}
		if got.From.Local() != "alice" {
			t.Errorf("message From = %q, want alice@...", got.From)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("bob did not receive alice's message within 5s")
	}
}

// TestConcurrentClients stresses the negotiation path with many simultaneous
// logins to surface races (run with -race).
func TestConcurrentClients(t *testing.T) {
	_, addr, store := startServer(t)
	const n = 20
	for i := 0; i < n; i++ {
		addUser(t, store, userName(i), "pw")
	}

	var wg sync.WaitGroup
	errs := make(chan error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			c, _ := NewClient(jid.MustParse(userName(i)+"@"+testDomain), "pw",
				WithConnectAddr(addr))
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if err := c.Connect(ctx); err != nil {
				errs <- err
				return
			}
			_ = c.Close()
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Errorf("concurrent connect failed: %v", err)
	}
}

func userName(i int) string {
	return "user" + string(rune('a'+i%26)) + string(rune('0'+i/26))
}
