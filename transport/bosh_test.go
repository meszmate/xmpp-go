package transport

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBOSHSetSID(t *testing.T) {
	t.Parallel()
	b := NewBOSH("http://localhost")
	if b.SID() != "" {
		t.Error("initial SID should be empty")
	}
	b.SetSID("session123")
	if b.SID() != "session123" {
		t.Errorf("SID() = %q, want %q", b.SID(), "session123")
	}
}

func TestBOSHStartTLSError(t *testing.T) {
	t.Parallel()
	b := NewBOSH("http://localhost")
	if err := b.StartTLS(nil); err == nil {
		t.Error("StartTLS should return error for BOSH")
	}
}

func TestBOSHConnectionState(t *testing.T) {
	t.Parallel()
	b := NewBOSH("http://localhost")
	_, ok := b.ConnectionState()
	if ok {
		t.Error("BOSH ConnectionState should return false")
	}
}

func TestBOSHPeerNil(t *testing.T) {
	t.Parallel()
	b := NewBOSH("http://localhost")
	if b.Peer() != nil {
		t.Error("BOSH Peer() should return nil")
	}
}

func TestBOSHLocalAddressNil(t *testing.T) {
	t.Parallel()
	b := NewBOSH("http://localhost")
	if b.LocalAddress() != nil {
		t.Error("BOSH LocalAddress() should return nil")
	}
}

func TestBOSHWriteRead(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "text/xml")
		w.Write([]byte("<response>" + string(body) + "</response>"))
	}))
	defer srv.Close()

	b := NewBOSH(srv.URL)
	payload := []byte("<body/>")
	n, err := b.Write(payload)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if n != len(payload) {
		t.Errorf("Write returned %d, want %d", n, len(payload))
	}

	buf := make([]byte, 256)
	n, err = b.Read(buf)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	got := string(buf[:n])
	if got != "<response><body/></response>" {
		t.Errorf("Read = %q", got)
	}
}

func TestBOSHClose(t *testing.T) {
	t.Parallel()
	b := NewBOSH("http://localhost")
	if err := b.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	_, err := b.Write([]byte("data"))
	if err == nil {
		t.Error("Write after Close should return error")
	}

	_, err = b.Read(make([]byte, 64))
	if err != io.EOF {
		t.Errorf("Read after Close should return EOF, got %v", err)
	}
}
