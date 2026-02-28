package main

import (
	"bytes"
	"strings"
	"testing"

	xmppxml "github.com/meszmate/xmpp-go/xml"
)

func TestWriteStreamStartHeader(t *testing.T) {
	var buf bytes.Buffer
	writer := xmppxml.NewStreamWriter(&buf)

	if err := writeStreamStart(writer, "example.com"); err != nil {
		t.Fatalf("writeStreamStart failed: %v", err)
	}

	s := buf.String()
	if !strings.Contains(s, "<stream:stream") {
		t.Fatalf("expected stream prefix in header, got %q", s)
	}
	if strings.Count(s, "xmlns=") != 1 {
		t.Fatalf("expected exactly one default xmlns declaration, got %q", s)
	}
	if !strings.Contains(s, "xmlns='jabber:client'") {
		t.Fatalf("expected jabber:client namespace, got %q", s)
	}
	if !strings.Contains(s, "xmlns:stream='http://etherx.jabber.org/streams'") {
		t.Fatalf("expected stream namespace declaration, got %q", s)
	}
	if !strings.Contains(s, "id='") {
		t.Fatalf("expected stream id attribute, got %q", s)
	}
	if !strings.Contains(s, "xml:lang='en'") {
		t.Fatalf("expected xml:lang attribute, got %q", s)
	}
}
