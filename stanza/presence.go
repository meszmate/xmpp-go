package stanza

import (
	"encoding/xml"

	"github.com/meszmate/xmpp-go/internal/ns"
)

// Presence type constants.
const (
	PresenceAvailable    = ""
	PresenceUnavailable  = "unavailable"
	PresenceSubscribe    = "subscribe"
	PresenceSubscribed   = "subscribed"
	PresenceUnsubscribe  = "unsubscribe"
	PresenceUnsubscribed = "unsubscribed"
	PresenceProbe        = "probe"
	PresenceError        = "error"
)

// Show values for presence.
const (
	ShowAway = "away"
	ShowChat = "chat"
	ShowDND  = "dnd"
	ShowXA   = "xa"
)

// Presence represents an XMPP presence stanza.
type Presence struct {
	Header
	XMLName    xml.Name    `xml:"presence"`
	Show       string      `xml:"show,omitempty"`
	Status     string      `xml:"status,omitempty"`
	Priority   int8        `xml:"priority,omitempty"`
	Error      *StanzaError `xml:"error,omitempty"`
	Extensions []Extension `xml:",any,omitempty"`
}

// NewPresence creates a new Presence with the given type.
func NewPresence(typ string) *Presence {
	return &Presence{
		Header: Header{
			XMLName: xml.Name{Space: ns.Client, Local: "presence"},
			ID:      GenerateID(),
			Type:    typ,
		},
	}
}

// StanzaType returns "presence".
func (p *Presence) StanzaType() string {
	return "presence"
}
