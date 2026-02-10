package transport

import (
	"net"
	"testing"
)

func TestWebSocketReadWrite(t *testing.T) {
	t.Parallel()
	c1, c2 := net.Pipe()
	defer c1.Close()
	defer c2.Close()

	ws1 := NewWebSocket(c1)
	ws2 := NewWebSocket(c2)

	msg := []byte("<message>hello</message>")
	go func() {
		ws1.Write(msg)
	}()

	buf := make([]byte, 128)
	n, err := ws2.Read(buf)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if string(buf[:n]) != string(msg) {
		t.Errorf("Read = %q, want %q", string(buf[:n]), string(msg))
	}
}

func TestWebSocketStartTLSError(t *testing.T) {
	t.Parallel()
	c1, c2 := net.Pipe()
	defer c1.Close()
	defer c2.Close()

	ws := NewWebSocket(c1)
	if err := ws.StartTLS(nil); err == nil {
		t.Error("StartTLS should return error for WebSocket")
	}
}

func TestWebSocketConnectionState(t *testing.T) {
	t.Parallel()
	c1, c2 := net.Pipe()
	defer c1.Close()
	defer c2.Close()

	ws := NewWebSocket(c1)
	_, ok := ws.ConnectionState()
	if ok {
		t.Error("plain connection should return false")
	}
}

func TestWebSocketPeerLocalAddress(t *testing.T) {
	t.Parallel()
	c1, c2 := net.Pipe()
	defer c1.Close()
	defer c2.Close()

	ws := NewWebSocket(c1)
	if ws.Peer() == nil {
		t.Error("Peer() should not be nil")
	}
	if ws.LocalAddress() == nil {
		t.Error("LocalAddress() should not be nil")
	}
}

func TestWebSocketClose(t *testing.T) {
	t.Parallel()
	c1, c2 := net.Pipe()
	defer c2.Close()

	ws := NewWebSocket(c1)
	if err := ws.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	buf := make([]byte, 64)
	_, err := c2.Read(buf)
	if err == nil {
		t.Error("expected error reading from closed peer")
	}
}
