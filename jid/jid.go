// Package jid implements XMPP JID (Jabber ID) parsing, validation, and escaping
// per RFC 7622 and XEP-0106.
package jid

import (
	"errors"
	"strings"
	"unicode/utf8"
)

var (
	ErrEmptyJID     = errors.New("jid: empty JID")
	ErrInvalidJID   = errors.New("jid: invalid JID format")
	ErrInvalidLocal = errors.New("jid: invalid localpart")
	ErrInvalidDomain = errors.New("jid: invalid domainpart")
	ErrInvalidResource = errors.New("jid: invalid resourcepart")
	ErrTooLong      = errors.New("jid: part exceeds maximum length")
)

const maxPartLen = 1023

// JID represents an XMPP address (localpart@domainpart/resourcepart).
type JID struct {
	local    string
	domain   string
	resource string
}

// New creates a new JID from its parts.
func New(local, domain, resource string) (JID, error) {
	if domain == "" {
		return JID{}, ErrInvalidDomain
	}
	if len(local) > maxPartLen {
		return JID{}, ErrTooLong
	}
	if len(domain) > maxPartLen {
		return JID{}, ErrTooLong
	}
	if len(resource) > maxPartLen {
		return JID{}, ErrTooLong
	}
	if local != "" && !validLocal(local) {
		return JID{}, ErrInvalidLocal
	}
	if !validDomain(domain) {
		return JID{}, ErrInvalidDomain
	}
	return JID{local: local, domain: domain, resource: resource}, nil
}

// Parse parses a JID string into a JID.
func Parse(s string) (JID, error) {
	if s == "" {
		return JID{}, ErrEmptyJID
	}

	var local, domain, resource string

	// Extract resource
	if slashIdx := strings.IndexByte(s, '/'); slashIdx != -1 {
		resource = s[slashIdx+1:]
		s = s[:slashIdx]
	}

	// Extract local and domain
	if atIdx := strings.IndexByte(s, '@'); atIdx != -1 {
		local = s[:atIdx]
		domain = s[atIdx+1:]
	} else {
		domain = s
	}

	if domain == "" {
		return JID{}, ErrInvalidDomain
	}

	return New(local, domain, resource)
}

// MustParse parses a JID string and panics on error.
func MustParse(s string) JID {
	j, err := Parse(s)
	if err != nil {
		panic(err)
	}
	return j
}

// Local returns the localpart.
func (j JID) Local() string { return j.local }

// Domain returns the domainpart.
func (j JID) Domain() string { return j.domain }

// Resource returns the resourcepart.
func (j JID) Resource() string { return j.resource }

// Bare returns a copy of the JID without the resource part.
func (j JID) Bare() JID {
	return JID{local: j.local, domain: j.domain}
}

// IsBare returns true if the JID has no resource part.
func (j JID) IsBare() bool {
	return j.resource == ""
}

// IsFull returns true if the JID has a resource part.
func (j JID) IsFull() bool {
	return j.resource != ""
}

// IsDomainOnly returns true if the JID has no local part.
func (j JID) IsDomainOnly() bool {
	return j.local == ""
}

// Equal returns true if two JIDs are equal.
func (j JID) Equal(other JID) bool {
	return j.local == other.local && j.domain == other.domain && j.resource == other.resource
}

// String returns the string representation of the JID.
func (j JID) String() string {
	if j.domain == "" {
		return ""
	}
	var b strings.Builder
	if j.local != "" {
		b.WriteString(j.local)
		b.WriteByte('@')
	}
	b.WriteString(j.domain)
	if j.resource != "" {
		b.WriteByte('/')
		b.WriteString(j.resource)
	}
	return b.String()
}

// IsZero returns true if the JID is the zero value.
func (j JID) IsZero() bool {
	return j.domain == ""
}

// WithResource returns a copy of the JID with the given resource.
func (j JID) WithResource(resource string) JID {
	return JID{local: j.local, domain: j.domain, resource: resource}
}

// MarshalText implements encoding.TextMarshaler.
func (j JID) MarshalText() ([]byte, error) {
	return []byte(j.String()), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (j *JID) UnmarshalText(data []byte) error {
	parsed, err := Parse(string(data))
	if err != nil {
		return err
	}
	*j = parsed
	return nil
}

// XEP-0106 escaping replacements.
var escapeReplacer = strings.NewReplacer(
	`\`, `\5c`,
	` `, `\20`,
	`"`, `\22`,
	`&`, `\26`,
	`'`, `\27`,
	`/`, `\2f`,
	`:`, `\3a`,
	`<`, `\3c`,
	`>`, `\3e`,
	`@`, `\40`,
)

var unescapeReplacer = strings.NewReplacer(
	`\20`, ` `,
	`\22`, `"`,
	`\26`, `&`,
	`\27`, `'`,
	`\2f`, `/`,
	`\3a`, `:`,
	`\3c`, `<`,
	`\3e`, `>`,
	`\40`, `@`,
	`\5c`, `\`,
)

// EscapeLocal escapes a localpart per XEP-0106.
func EscapeLocal(s string) string {
	return escapeReplacer.Replace(s)
}

// UnescapeLocal unescapes a localpart per XEP-0106.
func UnescapeLocal(s string) string {
	return unescapeReplacer.Replace(s)
}

func validLocal(s string) bool {
	if s == "" {
		return true
	}
	if !utf8.ValidString(s) {
		return false
	}
	for _, r := range s {
		if r == '@' || r == '/' {
			return false
		}
	}
	return true
}

func validDomain(s string) bool {
	if s == "" {
		return false
	}
	if !utf8.ValidString(s) {
		return false
	}
	// Allow IP addresses in brackets
	if strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]") {
		return true
	}
	for _, r := range s {
		if r == '@' || r == '/' {
			return false
		}
	}
	return true
}
