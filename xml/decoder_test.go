package xml

import (
	"encoding/xml"
	"strings"
	"testing"
)

func TestDecoderNextStartElement(t *testing.T) {
	t.Parallel()
	input := `  <!-- comment -->  <element attr="val"/>`
	dec := NewDecoder(strings.NewReader(input))

	se, err := dec.NextStartElement()
	if err != nil {
		t.Fatalf("NextStartElement: %v", err)
	}
	if se.Name.Local != "element" {
		t.Errorf("Name.Local = %q, want %q", se.Name.Local, "element")
	}
}

func TestDecoderSkip(t *testing.T) {
	t.Parallel()
	input := `<root><a><b>text</b></a><c/></root>`
	dec := NewDecoder(strings.NewReader(input))

	// Read <root>
	se, err := dec.NextStartElement()
	if err != nil {
		t.Fatalf("NextStartElement root: %v", err)
	}
	if se.Name.Local != "root" {
		t.Fatalf("expected <root>, got <%s>", se.Name.Local)
	}

	// Read <a>
	se, err = dec.NextStartElement()
	if err != nil {
		t.Fatalf("NextStartElement a: %v", err)
	}
	if se.Name.Local != "a" {
		t.Fatalf("expected <a>, got <%s>", se.Name.Local)
	}

	// Skip <a> and its children
	if err := dec.Skip(); err != nil {
		t.Fatalf("Skip: %v", err)
	}

	// Next should be <c>
	se, err = dec.NextStartElement()
	if err != nil {
		t.Fatalf("NextStartElement after skip: %v", err)
	}
	if se.Name.Local != "c" {
		t.Errorf("expected <c>, got <%s>", se.Name.Local)
	}
}

func TestDecoderMalformedXML(t *testing.T) {
	t.Parallel()
	input := `<broken<>`
	dec := NewDecoder(strings.NewReader(input))

	var v struct{ XMLName xml.Name }
	if err := dec.Decode(&v); err == nil {
		t.Error("expected error for malformed XML")
	}
}

func TestDecoderDecode(t *testing.T) {
	t.Parallel()
	type msg struct {
		XMLName xml.Name `xml:"msg"`
		Text    string   `xml:"text"`
	}

	input := `<msg><text>hello</text></msg>`
	dec := NewDecoder(strings.NewReader(input))

	var m msg
	if err := dec.Decode(&m); err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if m.Text != "hello" {
		t.Errorf("Text = %q, want %q", m.Text, "hello")
	}
}

func TestDecoderToken(t *testing.T) {
	t.Parallel()
	input := `<a>text</a>`
	dec := NewDecoder(strings.NewReader(input))

	tok, err := dec.Token()
	if err != nil {
		t.Fatalf("Token: %v", err)
	}
	se, ok := tok.(xml.StartElement)
	if !ok {
		t.Fatalf("expected StartElement, got %T", tok)
	}
	if se.Name.Local != "a" {
		t.Errorf("Name.Local = %q", se.Name.Local)
	}
}
