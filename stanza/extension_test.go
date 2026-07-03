package stanza

import (
	"encoding/xml"
	"strings"
	"testing"
)

// TestExtensionNoDuplicateXmlns is a regression test: routing a message that
// carries a namespaced child extension must not emit duplicate xmlns
// attributes, which corrupts the element for the recipient.
func TestExtensionNoDuplicateXmlns(t *testing.T) {
	raw := `<message xmlns="jabber:client" type="chat" to="bob@x" from="alice@x">` +
		`<body>hi</body>` +
		`<active xmlns="http://jabber.org/protocol/chatstates"/>` +
		`</message>`

	var m Message
	if err := xml.Unmarshal([]byte(raw), &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	out, err := xml.Marshal(&m)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	s := string(out)

	if n := strings.Count(s, `xmlns="http://jabber.org/protocol/chatstates"`); n != 1 {
		t.Fatalf("expected exactly one chatstates xmlns, got %d in: %s", n, s)
	}
	// The re-marshaled output must itself be well-formed and re-parseable.
	var m2 Message
	if err := xml.Unmarshal(out, &m2); err != nil {
		t.Fatalf("re-marshaled output is not well-formed: %v\n%s", err, s)
	}
	if len(m2.Extensions) != 1 || m2.Extensions[0].XMLName.Local != "active" {
		t.Errorf("extension not preserved through round trip: %+v", m2.Extensions)
	}
}

func TestExtensionWithInnerAndAttrs(t *testing.T) {
	raw := `<message xmlns="jabber:client" type="chat">` +
		`<stanza-id xmlns="urn:xmpp:sid:0" id="abc" by="room@x"/>` +
		`<reactions xmlns="urn:xmpp:reactions:0" id="msg1"><reaction>👍</reaction></reactions>` +
		`</message>`
	var m Message
	if err := xml.Unmarshal([]byte(raw), &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	out, err := xml.Marshal(&m)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	s := string(out)
	// Each extension namespace must appear exactly once (no duplicates).
	if n := strings.Count(s, `xmlns="urn:xmpp:sid:0"`); n != 1 {
		t.Errorf("stanza-id xmlns appears %d times: %s", n, s)
	}
	if n := strings.Count(s, `xmlns="urn:xmpp:reactions:0"`); n != 1 {
		t.Errorf("reactions xmlns appears %d times: %s", n, s)
	}
	// Inner content and attributes preserved.
	if !strings.Contains(s, `id="abc"`) || !strings.Contains(s, "👍") {
		t.Errorf("inner/attrs not preserved: %s", s)
	}
	var m2 Message
	if err := xml.Unmarshal(out, &m2); err != nil {
		t.Fatalf("not well-formed after round trip: %v\n%s", err, s)
	}
}
