package xml

import (
	"bytes"
	"encoding/xml"
	"strings"
	"testing"
)

func TestStreamReaderDecode(t *testing.T) {
	t.Parallel()
	type item struct {
		XMLName xml.Name `xml:"item"`
		Name    string   `xml:"name"`
		Value   int      `xml:"value"`
	}

	input := `<item><name>test</name><value>42</value></item>`
	sr := NewStreamReader(strings.NewReader(input))

	var got item
	if err := sr.Decode(&got); err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if got.Name != "test" {
		t.Errorf("Name = %q, want %q", got.Name, "test")
	}
	if got.Value != 42 {
		t.Errorf("Value = %d, want %d", got.Value, 42)
	}
}

func TestStreamReaderSkip(t *testing.T) {
	t.Parallel()
	input := `<root><skip><child>deep</child></skip><next>hello</next></root>`
	sr := NewStreamReader(strings.NewReader(input))

	// Read the <root> start element
	tok, err := sr.Token()
	if err != nil {
		t.Fatalf("Token: %v", err)
	}
	if _, ok := tok.(xml.StartElement); !ok {
		t.Fatal("expected StartElement for <root>")
	}

	// Read the <skip> start element
	tok, err = sr.Token()
	if err != nil {
		t.Fatalf("Token: %v", err)
	}
	if se, ok := tok.(xml.StartElement); !ok || se.Name.Local != "skip" {
		t.Fatal("expected StartElement for <skip>")
	}

	// Skip <skip> and its children
	if err := sr.Skip(); err != nil {
		t.Fatalf("Skip: %v", err)
	}

	// Next token should be <next>
	tok, err = sr.Token()
	if err != nil {
		t.Fatalf("Token after skip: %v", err)
	}
	se, ok := tok.(xml.StartElement)
	if !ok || se.Name.Local != "next" {
		t.Errorf("expected <next>, got %T %v", tok, tok)
	}
}

func TestStreamWriterEncode(t *testing.T) {
	t.Parallel()
	type item struct {
		XMLName xml.Name `xml:"item"`
		Value   string   `xml:"value"`
	}

	var buf bytes.Buffer
	sw := NewStreamWriter(&buf)
	if err := sw.Encode(item{Value: "hello"}); err != nil {
		t.Fatalf("Encode: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "<item>") {
		t.Errorf("missing <item> in: %s", out)
	}
	if !strings.Contains(out, "<value>hello</value>") {
		t.Errorf("missing value in: %s", out)
	}
}

func TestStreamWriterWriteRaw(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	sw := NewStreamWriter(&buf)

	raw := []byte(`<raw>data</raw>`)
	n, err := sw.WriteRaw(raw)
	if err != nil {
		t.Fatalf("WriteRaw: %v", err)
	}
	if n != len(raw) {
		t.Errorf("WriteRaw returned %d, want %d", n, len(raw))
	}
	if buf.String() != string(raw) {
		t.Errorf("WriteRaw output = %q, want %q", buf.String(), string(raw))
	}
}
