package xml

import (
	"encoding/xml"
	"io"
)

// Decoder is a streaming XMPP XML decoder.
type Decoder struct {
	dec *xml.Decoder
}

// NewDecoder creates a new XMPP XML decoder.
func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{dec: xml.NewDecoder(r)}
}

// Token reads the next XML token.
func (d *Decoder) Token() (xml.Token, error) {
	return d.dec.Token()
}

// Decode decodes the next XML element into v.
func (d *Decoder) Decode(v interface{}) error {
	return d.dec.Decode(v)
}

// DecodeElement decodes a specific XML element into v.
func (d *Decoder) DecodeElement(v interface{}, start *xml.StartElement) error {
	return d.dec.DecodeElement(v, start)
}

// Skip skips the current element including its children.
func (d *Decoder) Skip() error {
	return d.dec.Skip()
}

// NextStartElement reads tokens until it finds a StartElement or EOF.
func (d *Decoder) NextStartElement() (*xml.StartElement, error) {
	for {
		tok, err := d.dec.Token()
		if err != nil {
			return nil, err
		}
		if se, ok := tok.(xml.StartElement); ok {
			return &se, nil
		}
	}
}

// Underlying returns the underlying xml.Decoder.
func (d *Decoder) Underlying() *xml.Decoder {
	return d.dec
}
