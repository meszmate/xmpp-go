package transport

import (
	"net"
	"testing"
)

func TestTCPReadWrite(t *testing.T) {
	t.Parallel()
	c1, c2 := net.Pipe()
	defer c1.Close()
	defer c2.Close()

	tcp1 := NewTCP(c1)
	tcp2 := NewTCP(c2)

	msg := []byte("hello xmpp")
	go func() {
		tcp1.Write(msg)
	}()

	buf := make([]byte, 64)
	n, err := tcp2.Read(buf)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if string(buf[:n]) != "hello xmpp" {
		t.Errorf("Read = %q, want %q", string(buf[:n]), "hello xmpp")
	}
}

func TestTCPClose(t *testing.T) {
	t.Parallel()
	c1, c2 := net.Pipe()
	tcp1 := NewTCP(c1)
	tcp2 := NewTCP(c2)

	tcp1.Close()

	buf := make([]byte, 64)
	_, err := tcp2.Read(buf)
	if err == nil {
		t.Error("expected error reading from closed peer")
	}
}

func TestTCPConnectionState(t *testing.T) {
	t.Parallel()
	c1, c2 := net.Pipe()
	defer c1.Close()
	defer c2.Close()

	tcp := NewTCP(c1)
	_, ok := tcp.ConnectionState()
	if ok {
		t.Error("plain connection should return false")
	}
}

func TestTCPPeerLocalAddress(t *testing.T) {
	t.Parallel()
	c1, c2 := net.Pipe()
	defer c1.Close()
	defer c2.Close()

	tcp := NewTCP(c1)
	peer := tcp.Peer()
	if peer == nil {
		t.Error("Peer() should not be nil for net.Pipe")
	}
	local := tcp.LocalAddress()
	if local == nil {
		t.Error("LocalAddress() should not be nil for net.Pipe")
	}
}

func TestTCPConn(t *testing.T) {
	t.Parallel()
	c1, c2 := net.Pipe()
	defer c1.Close()
	defer c2.Close()

	tcp := NewTCP(c1)
	if tcp.Conn() != c1 {
		t.Error("Conn() should return the underlying connection")
	}
}
