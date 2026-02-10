package xml

import (
	"bytes"
	"encoding/xml"
	"strings"
	"testing"
)

func TestEncoderEncode(t *testing.T) {
	t.Parallel()
	type msg struct {
		XMLName xml.Name `xml:"message"`
		Body    string   `xml:"body"`
	}

	var buf bytes.Buffer
	enc := NewEncoder(&buf)
	if err := enc.Encode(msg{Body: "hello"}); err != nil {
		t.Fatalf("Encode: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "<message>") {
		t.Errorf("missing <message> in: %s", out)
	}
	if !strings.Contains(out, "<body>hello</body>") {
		t.Errorf("missing body in: %s", out)
	}
}

func TestEncoderEncodeToken(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	enc := NewEncoder(&buf)

	start := xml.StartElement{Name: xml.Name{Local: "test"}}
	if err := enc.EncodeToken(start); err != nil {
		t.Fatalf("EncodeToken start: %v", err)
	}
	if err := enc.EncodeToken(xml.CharData("content")); err != nil {
		t.Fatalf("EncodeToken chardata: %v", err)
	}
	if err := enc.EncodeToken(xml.EndElement{Name: start.Name}); err != nil {
		t.Fatalf("EncodeToken end: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "<test>content</test>") {
		t.Errorf("unexpected output: %s", out)
	}
}

func TestEncoderWriteRaw(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	enc := NewEncoder(&buf)

	raw := []byte(`<raw/>`)
	n, err := enc.WriteRaw(raw)
	if err != nil {
		t.Fatalf("WriteRaw: %v", err)
	}
	if n != len(raw) {
		t.Errorf("WriteRaw returned %d, want %d", n, len(raw))
	}
	if buf.String() != "<raw/>" {
		t.Errorf("output = %q", buf.String())
	}
}

func TestEncoderDecodeRoundtrip(t *testing.T) {
	t.Parallel()
	type item struct {
		XMLName xml.Name `xml:"item"`
		Name    string   `xml:"name"`
		Count   int      `xml:"count"`
	}

	original := item{Name: "widget", Count: 5}

	var buf bytes.Buffer
	enc := NewEncoder(&buf)
	if err := enc.Encode(original); err != nil {
		t.Fatalf("Encode: %v", err)
	}

	dec := NewDecoder(&buf)
	var decoded item
	if err := dec.Decode(&decoded); err != nil {
		t.Fatalf("Decode: %v", err)
	}

	if decoded.Name != original.Name || decoded.Count != original.Count {
		t.Errorf("roundtrip: got %+v, want %+v", decoded, original)
	}
}
