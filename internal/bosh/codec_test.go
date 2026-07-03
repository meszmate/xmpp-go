package bosh

import (
	"io"
	"strings"
	"testing"
)

func TestParserBasicStream(t *testing.T) {
	stream := `<?xml version='1.0'?><stream:stream to='localhost' xmlns='jabber:client' xmlns:stream='http://etherx.jabber.org/streams'>` +
		`<stream:features><mechanisms xmlns='urn:ietf:params:xml:ns:xmpp-sasl'><mechanism>SCRAM-SHA-256</mechanism></mechanisms></stream:features>` +
		`<challenge xmlns='urn:ietf:params:xml:ns:xmpp-sasl'>abc</challenge>`
	p := NewParser(strings.NewReader(stream))

	ev, err := p.Next()
	if err != nil || ev.Kind != Open {
		t.Fatalf("event 1 = %+v, %v; want Open", ev, err)
	}
	if ev.Attrs["to"] != "localhost" {
		t.Errorf("open to = %q", ev.Attrs["to"])
	}

	ev, err = p.Next()
	if err != nil || ev.Kind != Element || !strings.Contains(string(ev.Raw), "mechanisms") {
		t.Fatalf("event 2 = %+v, %v; want features Element", ev, err)
	}

	ev, err = p.Next()
	if err != nil || ev.Kind != Element || !strings.Contains(string(ev.Raw), "challenge") {
		t.Fatalf("event 3 = %+v, %v; want challenge Element", ev, err)
	}
}

// TestParserHandlesRestart is the critical case: a stream restart re-opens
// <stream:stream> without closing the prior stream, and the parser must treat it
// as a fresh Open (not a nested element it waits forever to close).
func TestParserHandlesRestart(t *testing.T) {
	stream := `<stream:stream xmlns:stream='http://etherx.jabber.org/streams'>` +
		`<auth xmlns='urn:ietf:params:xml:ns:xmpp-sasl'>x</auth>` +
		`<?xml version='1.0'?><stream:stream xmlns:stream='http://etherx.jabber.org/streams'>` +
		`<iq type='result' id='1'/>`
	p := NewParser(strings.NewReader(stream))

	kinds := []EventKind{Open, Element, Open, Element}
	for i, want := range kinds {
		ev, err := p.Next()
		if err != nil {
			t.Fatalf("event %d: %v", i, err)
		}
		if ev.Kind != want {
			t.Fatalf("event %d kind = %v, want %v (raw=%s)", i, ev.Kind, want, ev.Raw)
		}
	}
}

func TestParserStreamClose(t *testing.T) {
	stream := `<stream:stream xmlns:stream='http://etherx.jabber.org/streams'><message/></stream:stream>`
	p := NewParser(strings.NewReader(stream))
	want := []EventKind{Open, Element, Close}
	for i, w := range want {
		ev, err := p.Next()
		if err != nil {
			t.Fatalf("event %d: %v", i, err)
		}
		if ev.Kind != w {
			t.Fatalf("event %d = %v, want %v", i, ev.Kind, w)
		}
	}
	if _, err := p.Next(); err != io.EOF {
		t.Errorf("expected EOF after close, got %v", err)
	}
}

func TestParserNestedElement(t *testing.T) {
	// A stanza with nested children and an attribute containing '>' must be
	// captured whole.
	stream := `<stream:stream xmlns:stream='http://etherx.jabber.org/streams'>` +
		`<iq type='set'><query xmlns='jabber:iq:roster'><item jid='a@b' name='x&gt;y'/></query></iq>`
	p := NewParser(strings.NewReader(stream))
	if ev, _ := p.Next(); ev.Kind != Open {
		t.Fatal("want Open")
	}
	ev, err := p.Next()
	if err != nil || ev.Kind != Element {
		t.Fatalf("want Element, got %+v %v", ev, err)
	}
	raw := string(ev.Raw)
	if !strings.HasPrefix(raw, "<iq") || !strings.HasSuffix(raw, "</iq>") {
		t.Errorf("element not captured whole: %s", raw)
	}
	if !strings.Contains(raw, "jabber:iq:roster") {
		t.Errorf("nested content lost: %s", raw)
	}
}

func TestBuildParseBodyRoundTrip(t *testing.T) {
	attrs := map[string]string{"sid": "s1", "rid": "42", "type": "terminate"}
	payload := []byte(`<message to='a@b'><body>hi</body></message>`)
	body := BuildBody(attrs, payload)

	parsed, err := ParseBody(body)
	if err != nil {
		t.Fatalf("ParseBody: %v", err)
	}
	for k, v := range attrs {
		if parsed.Attrs[k] != v {
			t.Errorf("attr %q = %q, want %q", k, parsed.Attrs[k], v)
		}
	}
	if string(parsed.Payload) != string(payload) {
		t.Errorf("payload = %q, want %q", parsed.Payload, payload)
	}
}

func TestParseEmptyBody(t *testing.T) {
	body := BuildBody(map[string]string{"sid": "s1"}, nil)
	parsed, err := ParseBody(body)
	if err != nil {
		t.Fatalf("ParseBody: %v", err)
	}
	if len(parsed.Payload) != 0 {
		t.Errorf("expected empty payload, got %q", parsed.Payload)
	}
	if parsed.Attrs["sid"] != "s1" {
		t.Errorf("sid = %q", parsed.Attrs["sid"])
	}
}
