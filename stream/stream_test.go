package stream

import (
	"strings"
	"testing"

	"github.com/meszmate/xmpp-go/jid"
)

func TestOpen(t *testing.T) {
	t.Parallel()
	to := jid.MustParse("example.com")
	from := jid.MustParse("client.example.com")

	data := Open(Header{
		To:   to,
		From: from,
		ID:   "abc123",
		Lang: "en",
	})
	s := string(data)

	if !strings.Contains(s, "<?xml version='1.0'?>") {
		t.Error("missing XML declaration")
	}
	if !strings.Contains(s, "<stream:stream") {
		t.Error("missing stream:stream opening")
	}
	if !strings.Contains(s, "to='example.com'") {
		t.Errorf("missing to attr in: %s", s)
	}
	if !strings.Contains(s, "from='client.example.com'") {
		t.Errorf("missing from attr in: %s", s)
	}
	if !strings.Contains(s, "id='abc123'") {
		t.Errorf("missing id attr in: %s", s)
	}
	if !strings.Contains(s, "version='1.0'") {
		t.Error("missing default version")
	}
	if !strings.Contains(s, "xml:lang='en'") {
		t.Error("missing lang attr")
	}
	if !strings.Contains(s, "xmlns='jabber:client'") {
		t.Error("missing default xmlns")
	}
	if !strings.HasSuffix(s, ">") {
		t.Error("should end with >")
	}
}

func TestOpenCustomVersion(t *testing.T) {
	t.Parallel()
	data := Open(Header{
		To:      jid.MustParse("example.com"),
		Version: "2.0",
	})
	if !strings.Contains(string(data), "version='2.0'") {
		t.Error("custom version not used")
	}
}

func TestOpenCustomNS(t *testing.T) {
	t.Parallel()
	data := Open(Header{
		To: jid.MustParse("example.com"),
		NS: "jabber:component:accept",
	})
	if !strings.Contains(string(data), "xmlns='jabber:component:accept'") {
		t.Error("custom namespace not used")
	}
}

func TestClose(t *testing.T) {
	t.Parallel()
	data := Close()
	if string(data) != "</stream:stream>" {
		t.Errorf("Close() = %q, want %q", string(data), "</stream:stream>")
	}
}
