package jid

import (
	"encoding/xml"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		local    string
		domain   string
		resource string
		wantErr  error
	}{
		{"bare", "user", "example.com", "", nil},
		{"full", "user", "example.com", "res", nil},
		{"domain only", "", "example.com", "", nil},
		{"empty domain", "user", "", "", ErrInvalidDomain},
		{"local with @", "us@er", "example.com", "", ErrInvalidLocal},
		{"local with /", "us/er", "example.com", "", ErrInvalidLocal},
		{"domain with @", "", "ex@mple.com", "", ErrInvalidDomain},
		{"too long local", strings.Repeat("a", 1024), "example.com", "", ErrTooLong},
		{"too long domain", "", strings.Repeat("a", 1024), "", ErrTooLong},
		{"too long resource", "user", "example.com", strings.Repeat("a", 1024), ErrTooLong},
		{"max length local", strings.Repeat("a", 1023), "example.com", "", nil},
		{"ip domain", "", "[::1]", "", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := New(tt.local, tt.domain, tt.resource)
			if err != tt.wantErr {
				t.Errorf("New(%q, %q, %q) error = %v, want %v", tt.local, tt.domain, tt.resource, err, tt.wantErr)
			}
		})
	}
}

func TestParse(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		input        string
		wantLocal    string
		wantDomain   string
		wantResource string
		wantErr      bool
	}{
		{"bare", "user@example.com", "user", "example.com", "", false},
		{"full", "user@example.com/res", "user", "example.com", "res", false},
		{"domain only", "example.com", "", "example.com", "", false},
		{"resource with slash", "user@example.com/res/extra", "user", "example.com", "res/extra", false},
		{"empty string", "", "", "", "", true},
		{"just @", "@example.com", "", "example.com", "", false},
		{"domain with resource", "example.com/res", "", "example.com", "res", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			j, err := Parse(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Parse(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if j.Local() != tt.wantLocal {
				t.Errorf("Local() = %q, want %q", j.Local(), tt.wantLocal)
			}
			if j.Domain() != tt.wantDomain {
				t.Errorf("Domain() = %q, want %q", j.Domain(), tt.wantDomain)
			}
			if j.Resource() != tt.wantResource {
				t.Errorf("Resource() = %q, want %q", j.Resource(), tt.wantResource)
			}
		})
	}
}

func TestMustParse(t *testing.T) {
	t.Parallel()
	j := MustParse("user@example.com/res")
	if j.String() != "user@example.com/res" {
		t.Errorf("MustParse got %q", j.String())
	}

	defer func() {
		if r := recover(); r == nil {
			t.Error("MustParse with invalid input did not panic")
		}
	}()
	MustParse("")
}

func TestJIDAccessors(t *testing.T) {
	t.Parallel()
	j, _ := New("user", "example.com", "res")
	if j.Local() != "user" {
		t.Errorf("Local() = %q", j.Local())
	}
	if j.Domain() != "example.com" {
		t.Errorf("Domain() = %q", j.Domain())
	}
	if j.Resource() != "res" {
		t.Errorf("Resource() = %q", j.Resource())
	}
	if j.String() != "user@example.com/res" {
		t.Errorf("String() = %q", j.String())
	}
	bare := j.Bare()
	if bare.String() != "user@example.com" {
		t.Errorf("Bare().String() = %q", bare.String())
	}
}

func TestPredicates(t *testing.T) {
	t.Parallel()
	bare, _ := New("user", "example.com", "")
	full, _ := New("user", "example.com", "res")
	domOnly, _ := New("", "example.com", "")

	if !bare.IsBare() {
		t.Error("bare JID should be bare")
	}
	if bare.IsFull() {
		t.Error("bare JID should not be full")
	}
	if !full.IsFull() {
		t.Error("full JID should be full")
	}
	if full.IsBare() {
		t.Error("full JID should not be bare")
	}
	if !domOnly.IsDomainOnly() {
		t.Error("domain-only JID should be domain-only")
	}
	if bare.IsDomainOnly() {
		t.Error("bare JID with local should not be domain-only")
	}
}

func TestEqual(t *testing.T) {
	t.Parallel()
	a, _ := New("user", "example.com", "res")
	b, _ := New("user", "example.com", "res")
	c, _ := New("other", "example.com", "res")

	if !a.Equal(b) {
		t.Error("identical JIDs should be equal")
	}
	if a.Equal(c) {
		t.Error("different JIDs should not be equal")
	}
	var zero JID
	if a.Equal(zero) {
		t.Error("JID should not equal zero value")
	}
}

func TestIsZero(t *testing.T) {
	t.Parallel()
	var zero JID
	if !zero.IsZero() {
		t.Error("zero JID should be zero")
	}
	j, _ := New("user", "example.com", "")
	if j.IsZero() {
		t.Error("non-zero JID should not be zero")
	}
}

func TestWithResource(t *testing.T) {
	t.Parallel()
	j, _ := New("user", "example.com", "old")
	j2 := j.WithResource("new")
	if j2.Resource() != "new" {
		t.Errorf("WithResource() resource = %q, want %q", j2.Resource(), "new")
	}
	if j2.Local() != "user" || j2.Domain() != "example.com" {
		t.Error("WithResource should preserve local and domain")
	}
}

func TestMarshalUnmarshalText(t *testing.T) {
	t.Parallel()
	orig, _ := New("user", "example.com", "res")
	data, err := orig.MarshalText()
	if err != nil {
		t.Fatalf("MarshalText: %v", err)
	}
	if string(data) != "user@example.com/res" {
		t.Errorf("MarshalText = %q", string(data))
	}

	var j JID
	if err := j.UnmarshalText(data); err != nil {
		t.Fatalf("UnmarshalText: %v", err)
	}
	if !j.Equal(orig) {
		t.Errorf("round-trip failed: got %v, want %v", j, orig)
	}
}

func TestEscapeLocal(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input string
		want  string
	}{
		{`space test`, `space\20test`},
		{`"quoted"`, `\22quoted\22`},
		{`&amp`, `\26amp`},
		{`'apos`, `\27apos`},
		{`path/to`, `path\2fto`},
		{`:colon`, `\3acolon`},
		{`<less`, `\3cless`},
		{`>greater`, `\3egreater`},
		{`@at`, `\40at`},
		{`back\slash`, `back\5cslash`},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			got := EscapeLocal(tt.input)
			if got != tt.want {
				t.Errorf("EscapeLocal(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestUnescapeLocal(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input string
		want  string
	}{
		{`space\20test`, `space test`},
		{`\22quoted\22`, `"quoted"`},
		{`\26amp`, `&amp`},
		{`\27apos`, `'apos`},
		{`path\2fto`, `path/to`},
		{`\3acolon`, `:colon`},
		{`\3cless`, `<less`},
		{`\3egreater`, `>greater`},
		{`\40at`, `@at`},
		{`back\5cslash`, `back\slash`},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			got := UnescapeLocal(tt.input)
			if got != tt.want {
				t.Errorf("UnescapeLocal(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestEscapeRoundtrip(t *testing.T) {
	t.Parallel()
	inputs := []string{
		`user@host`,
		`"hello" <world>`,
		`a/b:c&d'e f`,
		`back\slash`,
	}
	for _, s := range inputs {
		got := UnescapeLocal(EscapeLocal(s))
		if got != s {
			t.Errorf("roundtrip(%q) = %q", s, got)
		}
	}
}

func TestMarshalXMLAttrOmitsZeroJID(t *testing.T) {
	t.Parallel()

	type holder struct {
		XMLName xml.Name `xml:"stanza"`
		From    JID      `xml:"from,attr,omitempty"`
		To      JID      `xml:"to,attr,omitempty"`
	}

	out, err := xml.Marshal(holder{})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	s := string(out)
	if strings.Contains(s, `from=""`) || strings.Contains(s, `to=""`) {
		t.Fatalf("zero JID attrs must be omitted, got: %s", s)
	}
}

func TestMarshalXMLAttrIncludesNonZeroJID(t *testing.T) {
	t.Parallel()

	j := MustParse("alice@example.com/res")

	type holder struct {
		XMLName xml.Name `xml:"stanza"`
		From    JID      `xml:"from,attr,omitempty"`
	}

	out, err := xml.Marshal(holder{From: j})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	s := string(out)
	if !strings.Contains(s, `from="alice@example.com/res"`) {
		t.Fatalf("expected serialized JID attr, got: %s", s)
	}
}
