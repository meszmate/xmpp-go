package jid

import "testing"

func TestDomainCaseFolding(t *testing.T) {
	a, err := Parse("user@EXAMPLE.COM")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if a.Domain() != "example.com" {
		t.Errorf("domain not case-folded: %q", a.Domain())
	}
	b := MustParse("user@example.com")
	if !a.Equal(b) {
		t.Errorf("user@EXAMPLE.COM should equal user@example.com after folding")
	}
	if a.String() != "user@example.com" {
		t.Errorf("String() = %q, want user@example.com", a.String())
	}
}

func TestNewDomainCaseFolding(t *testing.T) {
	j, err := New("user", "Example.Com", "Res")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if j.Domain() != "example.com" {
		t.Errorf("domain = %q, want example.com", j.Domain())
	}
	// Resource is case-sensitive and must be preserved.
	if j.Resource() != "Res" {
		t.Errorf("resource = %q, want Res", j.Resource())
	}
}

func TestTrailingSlashResourceRejected(t *testing.T) {
	if _, err := Parse("user@example.com/"); err == nil {
		t.Error("Parse of trailing-slash empty resource should error")
	}
}

func TestControlCharResourceRejected(t *testing.T) {
	if _, err := New("user", "example.com", "bad\x01res"); err == nil {
		t.Error("New with control char in resource should error")
	}
	if _, err := Parse("user@example.com/bad\x00res"); err == nil {
		t.Error("Parse with control char in resource should error")
	}
}

func TestLocalpartCaseFolding(t *testing.T) {
	a := MustParse("Alice@example.com")
	if a.Local() != "alice" {
		t.Errorf("localpart not case-folded: %q", a.Local())
	}
	if !a.Equal(MustParse("alice@example.com")) {
		t.Error("Alice@ should equal alice@ after PRECIS folding")
	}
}

func TestLocalpartSpaceRejected(t *testing.T) {
	if _, err := New("al ice", "example.com", ""); err == nil {
		t.Error("localpart with a space should be rejected")
	}
}

func TestIDNDomainToASCII(t *testing.T) {
	j := MustParse("user@münchen.de")
	if j.Domain() != "xn--mnchen-3ya.de" {
		t.Errorf("IDN domain not converted to A-label: %q", j.Domain())
	}
	// The punycode form must compare equal to the Unicode form.
	if !j.Equal(MustParse("user@xn--mnchen-3ya.de")) {
		t.Error("Unicode and punycode domain forms should be equal")
	}
}

func TestResourceCasePreserved(t *testing.T) {
	j := MustParse("user@example.com/MyPhone")
	if j.Resource() != "MyPhone" {
		t.Errorf("resource case not preserved: %q", j.Resource())
	}
}
