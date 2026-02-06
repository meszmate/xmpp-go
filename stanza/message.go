package stanza

import (
	"encoding/xml"

	"github.com/meszmate/xmpp-go/internal/ns"
)

// Message type constants.
const (
	MessageChat      = "chat"
	MessageError     = "error"
	MessageGroupchat = "groupchat"
	MessageHeadline  = "headline"
	MessageNormal    = "normal"
)

// Message represents an XMPP message stanza.
type Message struct {
	Header
	XMLName    xml.Name    `xml:"message"`
	Subject    string      `xml:"subject,omitempty"`
	Body       string      `xml:"body,omitempty"`
	Thread     string      `xml:"thread,omitempty"`
	Error      *StanzaError `xml:"error,omitempty"`
	Extensions []Extension `xml:",any,omitempty"`
}

// NewMessage creates a new Message with the given type and a random ID.
func NewMessage(typ string) *Message {
	return &Message{
		Header: Header{
			XMLName: xml.Name{Space: ns.Client, Local: "message"},
			ID:      GenerateID(),
			Type:    typ,
		},
	}
}

// StanzaType returns "message".
func (m *Message) StanzaType() string {
	return "message"
}
