package stanza

import (
	"bytes"
	"encoding/hex"
	"encoding/xml"
	"strings"
	"testing"

	"github.com/meszmate/xmpp-go/jid"
)

func TestGenerateID(t *testing.T) {
	t.Parallel()
	id := GenerateID()
	if len(id) != 32 {
		t.Errorf("GenerateID() length = %d, want 32", len(id))
	}
	if _, err := hex.DecodeString(id); err != nil {
		t.Errorf("GenerateID() not valid hex: %v", err)
	}
	id2 := GenerateID()
	if id == id2 {
		t.Error("two GenerateID() calls returned the same value")
	}
}

func TestNewMessage(t *testing.T) {
	t.Parallel()
	m := NewMessage(MessageChat)
	if m.Type != MessageChat {
		t.Errorf("Type = %q, want %q", m.Type, MessageChat)
	}
	if m.ID == "" {
		t.Error("ID should not be empty")
	}
	if m.StanzaType() != "message" {
		t.Errorf("StanzaType() = %q, want %q", m.StanzaType(), "message")
	}
	if m.GetHeader().XMLName.Local != "message" {
		t.Errorf("Header.XMLName.Local = %q, want %q", m.GetHeader().XMLName.Local, "message")
	}
}

func TestNewPresence(t *testing.T) {
	t.Parallel()
	p := NewPresence(PresenceUnavailable)
	if p.Type != PresenceUnavailable {
		t.Errorf("Type = %q, want %q", p.Type, PresenceUnavailable)
	}
	if p.ID == "" {
		t.Error("ID should not be empty")
	}
	if p.StanzaType() != "presence" {
		t.Errorf("StanzaType() = %q", p.StanzaType())
	}
}

func TestNewIQ(t *testing.T) {
	t.Parallel()
	iq := NewIQ(IQGet)
	if iq.Type != IQGet {
		t.Errorf("Type = %q, want %q", iq.Type, IQGet)
	}
	if iq.ID == "" {
		t.Error("ID should not be empty")
	}
	if iq.StanzaType() != "iq" {
		t.Errorf("StanzaType() = %q", iq.StanzaType())
	}
}

func TestIQMarshalOmitsEmptyJIDAttrs(t *testing.T) {
	t.Parallel()

	iq := NewIQ(IQSet)
	out, err := xml.Marshal(iq)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	s := string(out)
	if strings.Contains(s, `from=""`) || strings.Contains(s, `to=""`) {
		t.Fatalf("empty from/to attrs must be omitted, got: %s", s)
	}
}

func TestIQResultIQ(t *testing.T) {
	t.Parallel()
	iq := NewIQ(IQGet)
	iq.From = jid.MustParse("alice@example.com")
	iq.To = jid.MustParse("bob@example.com")

	result := iq.ResultIQ()
	if result.Type != IQResult {
		t.Errorf("result Type = %q, want %q", result.Type, IQResult)
	}
	if result.ID != iq.ID {
		t.Errorf("result ID = %q, want %q", result.ID, iq.ID)
	}
	if result.To.String() != "alice@example.com" {
		t.Errorf("result To = %q, want %q", result.To.String(), "alice@example.com")
	}
	if result.From.String() != "bob@example.com" {
		t.Errorf("result From = %q, want %q", result.From.String(), "bob@example.com")
	}
}

func TestIQErrorIQ(t *testing.T) {
	t.Parallel()
	iq := NewIQ(IQGet)
	iq.From = jid.MustParse("alice@example.com")
	iq.To = jid.MustParse("bob@example.com")

	se := NewStanzaError(ErrorTypeCancel, ErrorItemNotFound, "not found")
	errIQ := iq.ErrorIQ(se)

	if errIQ.Type != IQError {
		t.Errorf("error IQ Type = %q, want %q", errIQ.Type, IQError)
	}
	if errIQ.To.String() != "alice@example.com" {
		t.Error("To/From not swapped correctly")
	}
	if errIQ.Error == nil {
		t.Fatal("Error should not be nil")
	}
	if errIQ.Error.Condition != ErrorItemNotFound {
		t.Errorf("error Condition = %q", errIQ.Error.Condition)
	}
}

func TestNewStanzaError(t *testing.T) {
	t.Parallel()
	se := NewStanzaError(ErrorTypeAuth, ErrorNotAuthorized, "go away")
	if se.Type != ErrorTypeAuth {
		t.Errorf("Type = %q", se.Type)
	}
	if se.Condition != ErrorNotAuthorized {
		t.Errorf("Condition = %q", se.Condition)
	}
	if se.Text != "go away" {
		t.Errorf("Text = %q", se.Text)
	}
}

func TestStanzaErrorString(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		err  *StanzaError
		want string
	}{
		{
			"with text",
			NewStanzaError(ErrorTypeCancel, ErrorItemNotFound, "missing"),
			"stanza error: item-not-found (cancel: missing)",
		},
		{
			"without text",
			NewStanzaError(ErrorTypeAuth, ErrorForbidden, ""),
			"stanza error: forbidden (auth)",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.err.Error()
			if got != tt.want {
				t.Errorf("Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestStanzaErrorMarshalXML(t *testing.T) {
	t.Parallel()
	se := NewStanzaError(ErrorTypeCancel, ErrorItemNotFound, "not found")

	var buf bytes.Buffer
	enc := xml.NewEncoder(&buf)
	if err := enc.Encode(se); err != nil {
		t.Fatalf("Encode: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, `type="cancel"`) {
		t.Errorf("missing type attr in: %s", out)
	}
	if !strings.Contains(out, "item-not-found") {
		t.Errorf("missing condition in: %s", out)
	}
	if !strings.Contains(out, "not found") {
		t.Errorf("missing text in: %s", out)
	}
}
