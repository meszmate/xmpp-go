package xmpp

import (
	"encoding/xml"
	"errors"
	"testing"
)

func TestEncodeDecodeSASLValue(t *testing.T) {
	// nil -> "" -> nil
	if got := encodeSASLValue(nil); got != "" {
		t.Errorf("encodeSASLValue(nil) = %q, want empty", got)
	}
	// empty non-nil -> "="
	if got := encodeSASLValue([]byte{}); got != "=" {
		t.Errorf("encodeSASLValue([]) = %q, want =", got)
	}
	// round-trip a payload
	payload := []byte("\x00alice\x00pw")
	enc := encodeSASLValue(payload)
	dec, err := decodeSASLValue(enc)
	if err != nil {
		t.Fatalf("decodeSASLValue: %v", err)
	}
	if string(dec) != string(payload) {
		t.Errorf("round trip mismatch: %q != %q", dec, payload)
	}
	// "=" and "" decode to nil
	for _, v := range []string{"", "=", "  "} {
		d, err := decodeSASLValue(v)
		if err != nil || d != nil {
			t.Errorf("decodeSASLValue(%q) = (%v, %v), want (nil, nil)", v, d, err)
		}
	}
}

func TestSASLFailureCondition(t *testing.T) {
	raw := `<failure xmlns="urn:ietf:params:xml:ns:xmpp-sasl"><not-authorized/><text>bad password</text></failure>`
	var f saslFailure
	if err := xml.Unmarshal([]byte(raw), &f); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if f.Condition() != "not-authorized" {
		t.Errorf("Condition() = %q, want not-authorized", f.Condition())
	}
	if f.Text != "bad password" {
		t.Errorf("Text = %q, want 'bad password'", f.Text)
	}
}

func TestAuthErrorMessage(t *testing.T) {
	cases := []struct {
		err  *AuthError
		want string
	}{
		{&AuthError{Condition: "not-authorized"}, "xmpp: authentication failed: not-authorized"},
		{&AuthError{Condition: "not-authorized", Text: "bad"}, "xmpp: authentication failed: not-authorized: bad"},
		{&AuthError{}, "xmpp: authentication failed"},
	}
	for _, tc := range cases {
		if got := tc.err.Error(); got != tc.want {
			t.Errorf("Error() = %q, want %q", got, tc.want)
		}
	}
	// errors.As unwrapping works.
	var target *AuthError
	if !errors.As(error(&AuthError{Condition: "x"}), &target) {
		t.Error("errors.As should match *AuthError")
	}
}

func TestStreamErrorParsing(t *testing.T) {
	raw := `<error xmlns="http://etherx.jabber.org/streams">` +
		`<host-unknown xmlns="urn:ietf:params:xml:ns:xmpp-streams"/>` +
		`<text xmlns="urn:ietf:params:xml:ns:xmpp-streams">no such host</text></error>`
	var e streamErrorElem
	if err := xml.Unmarshal([]byte(raw), &e); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	se := e.toError()
	if se.Condition != "host-unknown" {
		t.Errorf("Condition = %q, want host-unknown", se.Condition)
	}
	if se.Text != "no such host" {
		t.Errorf("Text = %q, want 'no such host'", se.Text)
	}
	if se.Error() == "" {
		t.Error("StreamError.Error() should be non-empty")
	}
}

func TestStreamFeaturesDecoding(t *testing.T) {
	raw := `<stream:features xmlns:stream="http://etherx.jabber.org/streams" xmlns="jabber:client">` +
		`<starttls xmlns="urn:ietf:params:xml:ns:xmpp-tls"><required/></starttls>` +
		`<mechanisms xmlns="urn:ietf:params:xml:ns:xmpp-sasl">` +
		`<mechanism>SCRAM-SHA-256</mechanism><mechanism>PLAIN</mechanism></mechanisms>` +
		`</stream:features>`
	var f streamFeatures
	if err := xml.Unmarshal([]byte(raw), &f); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if f.StartTLS == nil {
		t.Fatal("StartTLS feature not decoded")
	}
	if f.StartTLS.Required == nil {
		t.Error("STARTTLS required flag not decoded")
	}
	if f.Mechanisms == nil || len(f.Mechanisms.Mechanism) != 2 {
		t.Fatalf("mechanisms not decoded: %+v", f.Mechanisms)
	}
	if f.Mechanisms.Mechanism[0] != "SCRAM-SHA-256" {
		t.Errorf("first mechanism = %q", f.Mechanisms.Mechanism[0])
	}
}
