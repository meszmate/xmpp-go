package stanza

import (
	"encoding/xml"

	"github.com/meszmate/xmpp-go/internal/ns"
)

// IQ type constants.
const (
	IQGet    = "get"
	IQSet    = "set"
	IQResult = "result"
	IQError  = "error"
)

// IQ represents an XMPP IQ (Info/Query) stanza.
type IQ struct {
	Header
	XMLName xml.Name    `xml:"iq"`
	Query   []byte      `xml:",innerxml"`
	Error   *StanzaError `xml:"error,omitempty"`
}

// NewIQ creates a new IQ stanza with the given type.
func NewIQ(typ string) *IQ {
	return &IQ{
		Header: Header{
			XMLName: xml.Name{Space: ns.Client, Local: "iq"},
			ID:      GenerateID(),
			Type:    typ,
		},
	}
}

// StanzaType returns "iq".
func (iq *IQ) StanzaType() string {
	return "iq"
}

// ResultIQ creates a result IQ in response to this IQ.
func (iq *IQ) ResultIQ() *IQ {
	return &IQ{
		Header: Header{
			XMLName: xml.Name{Space: ns.Client, Local: "iq"},
			ID:      iq.ID,
			Type:    IQResult,
			From:    iq.To,
			To:      iq.From,
		},
	}
}

// ErrorIQ creates an error IQ in response to this IQ.
func (iq *IQ) ErrorIQ(err *StanzaError) *IQ {
	return &IQ{
		Header: Header{
			XMLName: xml.Name{Space: ns.Client, Local: "iq"},
			ID:      iq.ID,
			Type:    IQError,
			From:    iq.To,
			To:      iq.From,
		},
		Error: err,
	}
}

// IQPayload wraps an IQ with a typed payload for marshaling.
type IQPayload struct {
	IQ
	Payload interface{}
}

// MarshalXML implements xml.Marshaler for IQPayload.
func (iq *IQPayload) MarshalXML(enc *xml.Encoder, start xml.StartElement) error {
	start.Name = xml.Name{Space: ns.Client, Local: "iq"}
	if iq.ID != "" {
		start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: "id"}, Value: iq.ID})
	}
	if iq.Type != "" {
		start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: "type"}, Value: iq.Type})
	}
	if !iq.To.IsZero() {
		start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: "to"}, Value: iq.To.String()})
	}
	if !iq.From.IsZero() {
		start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: "from"}, Value: iq.From.String()})
	}

	if err := enc.EncodeToken(start); err != nil {
		return err
	}
	if iq.Payload != nil {
		if err := enc.Encode(iq.Payload); err != nil {
			return err
		}
	}
	return enc.EncodeToken(xml.EndElement{Name: start.Name})
}
