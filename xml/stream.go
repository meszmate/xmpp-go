// Package xml provides streaming XML encoding and decoding for XMPP streams.
package xml

import (
	"encoding/xml"
	"io"
)

// TokenReader reads XML tokens from a stream.
type TokenReader interface {
	Token() (xml.Token, error)
}

// TokenWriter writes XML tokens to a stream.
type TokenWriter interface {
	EncodeToken(t xml.Token) error
	Flush() error
}

// StreamReader wraps an xml.Decoder for reading XMPP streams.
type StreamReader struct {
	d *xml.Decoder
}

// NewStreamReader creates a new StreamReader.
func NewStreamReader(r io.Reader) *StreamReader {
	return &StreamReader{d: xml.NewDecoder(r)}
}

// Token reads the next XML token.
func (sr *StreamReader) Token() (xml.Token, error) {
	return sr.d.Token()
}

// Decode decodes the next element into v.
func (sr *StreamReader) Decode(v interface{}) error {
	return sr.d.Decode(v)
}

// DecodeElement decodes a specific element into v.
func (sr *StreamReader) DecodeElement(v interface{}, start *xml.StartElement) error {
	return sr.d.DecodeElement(v, start)
}

// Skip skips the current element and its children.
func (sr *StreamReader) Skip() error {
	return sr.d.Skip()
}

// Decoder returns the underlying xml.Decoder.
func (sr *StreamReader) Decoder() *xml.Decoder {
	return sr.d
}

// StreamWriter wraps an xml.Encoder for writing XMPP streams.
type StreamWriter struct {
	e *xml.Encoder
	w io.Writer
}

// NewStreamWriter creates a new StreamWriter.
func NewStreamWriter(w io.Writer) *StreamWriter {
	return &StreamWriter{
		e: xml.NewEncoder(w),
		w: w,
	}
}

// EncodeToken writes a single XML token.
func (sw *StreamWriter) EncodeToken(t xml.Token) error {
	if err := sw.e.EncodeToken(t); err != nil {
		return err
	}
	return sw.e.Flush()
}

// Encode encodes a value as XML.
func (sw *StreamWriter) Encode(v interface{}) error {
	if err := sw.e.Encode(v); err != nil {
		return err
	}
	return sw.e.Flush()
}

// Encoder returns the underlying xml.Encoder.
func (sw *StreamWriter) Encoder() *xml.Encoder {
	return sw.e
}

// WriteRaw writes raw bytes to the underlying writer, bypassing XML encoding.
func (sw *StreamWriter) WriteRaw(data []byte) (int, error) {
	return sw.w.Write(data)
}

// Flush flushes the encoder buffer.
func (sw *StreamWriter) Flush() error {
	return sw.e.Flush()
}
