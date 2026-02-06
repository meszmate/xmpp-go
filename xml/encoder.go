package xml

import (
	"encoding/xml"
	"io"
)

// Encoder is a streaming XMPP XML encoder.
type Encoder struct {
	w   io.Writer
	enc *xml.Encoder
}

// NewEncoder creates a new XMPP XML encoder.
func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{
		w:   w,
		enc: xml.NewEncoder(w),
	}
}

// Encode encodes a value as XML and flushes.
func (e *Encoder) Encode(v interface{}) error {
	if err := e.enc.Encode(v); err != nil {
		return err
	}
	return e.enc.Flush()
}

// EncodeToken writes a single XML token and flushes.
func (e *Encoder) EncodeToken(t xml.Token) error {
	if err := e.enc.EncodeToken(t); err != nil {
		return err
	}
	return e.enc.Flush()
}

// EncodeElement encodes a value as a specific XML element.
func (e *Encoder) EncodeElement(v interface{}, start xml.StartElement) error {
	if err := e.enc.EncodeElement(v, start); err != nil {
		return err
	}
	return e.enc.Flush()
}

// WriteRaw writes raw bytes directly to the writer.
func (e *Encoder) WriteRaw(data []byte) (int, error) {
	return e.w.Write(data)
}

// Flush flushes any buffered XML.
func (e *Encoder) Flush() error {
	return e.enc.Flush()
}

// Underlying returns the underlying xml.Encoder.
func (e *Encoder) Underlying() *xml.Encoder {
	return e.enc
}
