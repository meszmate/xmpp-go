// Package stanza defines XMPP stanza types: Message, Presence, and IQ.
package stanza

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/xml"

	"github.com/meszmate/xmpp-go/jid"
)

// Stanza is the interface implemented by all XMPP stanza types.
type Stanza interface {
	StanzaType() string
	GetHeader() *Header
}

// Header contains the common attributes of all stanzas.
type Header struct {
	XMLName xml.Name `xml:"-"`
	ID      string   `xml:"id,attr,omitempty"`
	From    jid.JID  `xml:"from,attr,omitempty"`
	To      jid.JID  `xml:"to,attr,omitempty"`
	Type    string   `xml:"type,attr,omitempty"`
	Lang    string   `xml:"xml:lang,attr,omitempty"`
}

// GetHeader returns the stanza header.
func (h *Header) GetHeader() *Header {
	return h
}

// GenerateID generates a random stanza ID.
func GenerateID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// Extension represents an arbitrary XML extension element in a stanza.
type Extension struct {
	XMLName xml.Name
	Inner   []byte `xml:",innerxml"`
	Attrs   []xml.Attr `xml:",any,attr"`
}
