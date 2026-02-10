package xmpp

import (
	"context"
	"net"
	"testing"

	"github.com/meszmate/xmpp-go/jid"
	"github.com/meszmate/xmpp-go/stanza"
	"github.com/meszmate/xmpp-go/transport"
)

func newTestSession(t *testing.T, opts ...SessionOption) (*Session, net.Conn) {
	t.Helper()
	c1, c2 := net.Pipe()
	tcp := transport.NewTCP(c1)
	s, err := NewSession(context.Background(), tcp, opts...)
	if err != nil {
		c1.Close()
		c2.Close()
		t.Fatalf("NewSession: %v", err)
	}
	return s, c2
}

func TestNewSession(t *testing.T) {
	t.Parallel()
	s, c2 := newTestSession(t)
	defer s.Close()
	defer c2.Close()

	if s.Transport() == nil {
		t.Error("Transport() should not be nil")
	}
	if s.Reader() == nil {
		t.Error("Reader() should not be nil")
	}
	if s.Writer() == nil {
		t.Error("Writer() should not be nil")
	}
	if s.Mux() == nil {
		t.Error("Mux() should not be nil")
	}
}

func TestSessionStateSetState(t *testing.T) {
	t.Parallel()
	s, c2 := newTestSession(t)
	defer s.Close()
	defer c2.Close()

	s.SetState(StateSecure)
	if s.State()&StateSecure == 0 {
		t.Error("StateSecure should be set")
	}

	s.SetState(StateAuthenticated)
	if s.State()&StateAuthenticated == 0 {
		t.Error("StateAuthenticated should be set")
	}
	// StateSecure should still be set
	if s.State()&StateSecure == 0 {
		t.Error("StateSecure should still be set after adding StateAuthenticated")
	}
}

func TestSessionLocalRemoteAddr(t *testing.T) {
	t.Parallel()
	local := jid.MustParse("alice@example.com/res")
	remote := jid.MustParse("bob@example.com/res")

	s, c2 := newTestSession(t, WithLocalAddr(local), WithRemoteAddr(remote))
	defer s.Close()
	defer c2.Close()

	if !s.LocalAddr().Equal(local) {
		t.Errorf("LocalAddr() = %v, want %v", s.LocalAddr(), local)
	}
	if !s.RemoteAddr().Equal(remote) {
		t.Errorf("RemoteAddr() = %v, want %v", s.RemoteAddr(), remote)
	}

	newLocal := jid.MustParse("carol@example.com")
	s.SetLocalAddr(newLocal)
	if !s.LocalAddr().Equal(newLocal) {
		t.Errorf("after SetLocalAddr: %v", s.LocalAddr())
	}

	newRemote := jid.MustParse("dave@example.com")
	s.SetRemoteAddr(newRemote)
	if !s.RemoteAddr().Equal(newRemote) {
		t.Errorf("after SetRemoteAddr: %v", s.RemoteAddr())
	}
}

func TestSessionSend(t *testing.T) {
	t.Parallel()
	s, c2 := newTestSession(t)
	defer s.Close()
	defer c2.Close()

	msg := stanza.NewMessage(stanza.MessageChat)
	msg.Body = "hello"

	done := make(chan error, 1)
	go func() {
		done <- s.Send(context.Background(), msg)
	}()

	buf := make([]byte, 4096)
	n, err := c2.Read(buf)
	if err != nil {
		t.Fatalf("pipe Read: %v", err)
	}
	got := string(buf[:n])
	if len(got) == 0 {
		t.Error("expected non-empty XML output")
	}

	if err := <-done; err != nil {
		t.Fatalf("Send: %v", err)
	}
}

func TestSessionSendClosed(t *testing.T) {
	t.Parallel()
	s, c2 := newTestSession(t)
	c2.Close()
	s.Close()

	msg := stanza.NewMessage(stanza.MessageChat)
	err := s.Send(context.Background(), msg)
	if err == nil {
		t.Error("Send on closed session should return error")
	}
}

func TestSessionCloseIdempotent(t *testing.T) {
	t.Parallel()
	s, c2 := newTestSession(t)
	defer c2.Close()

	if err := s.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}
	// Second close should not panic
	if err := s.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
}

func TestSessionOptions(t *testing.T) {
	t.Parallel()
	local := jid.MustParse("user@example.com")
	remote := jid.MustParse("server.example.com")
	mux := NewMux()

	s, c2 := newTestSession(t,
		WithLocalAddr(local),
		WithRemoteAddr(remote),
		WithState(StateSecure|StateAuthenticated),
		WithMux(mux),
	)
	defer s.Close()
	defer c2.Close()

	if !s.LocalAddr().Equal(local) {
		t.Error("WithLocalAddr not applied")
	}
	if !s.RemoteAddr().Equal(remote) {
		t.Error("WithRemoteAddr not applied")
	}
	if s.State()&StateSecure == 0 || s.State()&StateAuthenticated == 0 {
		t.Error("WithState not applied")
	}
	if s.Mux() != mux {
		t.Error("WithMux not applied")
	}
}
