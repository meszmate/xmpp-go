package xmpp

import (
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/meszmate/xmpp-go/internal/ns"
)

// streamFeatures is the <stream:features> element advertised by the peer during
// stream negotiation (RFC 6120 §4.3.2).
type streamFeatures struct {
	XMLName    xml.Name           `xml:"http://etherx.jabber.org/streams features"`
	StartTLS   *tlsFeature        `xml:"urn:ietf:params:xml:ns:xmpp-tls starttls"`
	Mechanisms *mechanismsFeature `xml:"urn:ietf:params:xml:ns:xmpp-sasl mechanisms"`
	Bind       *bindFeature       `xml:"urn:ietf:params:xml:ns:xmpp-bind bind"`
	SM         *smFeature         `xml:"urn:xmpp:sm:3 sm"`
	// Raw preserves any additional advertised features for inspection.
	Raw []byte `xml:",innerxml"`
}

// smFeature is the XEP-0198 Stream Management feature advertisement.
type smFeature struct{}

// smResumeTimeout is the default resumption window (seconds) the server offers
// and holds a dropped session's state for.
const smResumeTimeout = 120

// attrValue returns the value of the named attribute (any namespace), or "".
func attrValue(attrs []xml.Attr, name string) string {
	for _, a := range attrs {
		if a.Name.Local == name {
			return a.Value
		}
	}
	return ""
}

// attrTrue reports whether the named attribute is present and truthy
// (xs:boolean "true" or "1").
func attrTrue(attrs []xml.Attr, name string) bool {
	v := strings.TrimSpace(attrValue(attrs, name))
	return v == "true" || v == "1"
}

// parseUintAttr parses the named attribute as a uint32, defaulting to 0.
func parseUintAttr(attrs []xml.Attr, name string) uint32 {
	v := strings.TrimSpace(attrValue(attrs, name))
	var n uint64
	for _, c := range v {
		if c < '0' || c > '9' {
			break
		}
		n = n*10 + uint64(c-'0')
	}
	return uint32(n)
}

// tlsFeature is the STARTTLS feature advertisement.
type tlsFeature struct {
	Required *struct{} `xml:"required"`
}

// mechanismsFeature is the SASL mechanisms advertisement.
type mechanismsFeature struct {
	Mechanism []string `xml:"mechanism"`
}

// bindFeature is the resource-binding feature advertisement.
type bindFeature struct {
	Required *struct{} `xml:"required"`
}

// STARTTLS negotiation elements (RFC 6120 §5).

type tlsStartTLS struct {
	XMLName xml.Name `xml:"urn:ietf:params:xml:ns:xmpp-tls starttls"`
}

type tlsProceed struct {
	XMLName xml.Name `xml:"urn:ietf:params:xml:ns:xmpp-tls proceed"`
}

type tlsFailure struct {
	XMLName xml.Name `xml:"urn:ietf:params:xml:ns:xmpp-tls failure"`
}

// SASL negotiation elements (RFC 6120 §6).

type saslAuth struct {
	XMLName   xml.Name `xml:"urn:ietf:params:xml:ns:xmpp-sasl auth"`
	Mechanism string   `xml:"mechanism,attr"`
	Value     string   `xml:",chardata"`
}

type saslResponse struct {
	XMLName xml.Name `xml:"urn:ietf:params:xml:ns:xmpp-sasl response"`
	Value   string   `xml:",chardata"`
}

type saslChallenge struct {
	XMLName xml.Name `xml:"urn:ietf:params:xml:ns:xmpp-sasl challenge"`
	Value   string   `xml:",chardata"`
}

type saslSuccess struct {
	XMLName xml.Name `xml:"urn:ietf:params:xml:ns:xmpp-sasl success"`
	Value   string   `xml:",chardata"`
}

type saslAbort struct {
	XMLName xml.Name `xml:"urn:ietf:params:xml:ns:xmpp-sasl abort"`
}

// namedElement captures only the XML name of an element, used to read SASL
// failure condition child elements.
type namedElement struct {
	XMLName xml.Name
}

type saslFailure struct {
	XMLName    xml.Name       `xml:"urn:ietf:params:xml:ns:xmpp-sasl failure"`
	Text       string         `xml:"text"`
	Conditions []namedElement `xml:",any"`
}

// Condition returns the SASL failure condition local name (e.g. "not-authorized").
func (f *saslFailure) Condition() string {
	for _, c := range f.Conditions {
		if c.XMLName.Local != "" {
			return c.XMLName.Local
		}
	}
	return ""
}

// encodeSASLValue base64-encodes a SASL payload for the wire. A nil payload
// yields an empty element body; an empty (non-nil) payload yields "=" per the
// XMPP convention for an empty response (RFC 6120 §6.4.2).
func encodeSASLValue(data []byte) string {
	if data == nil {
		return ""
	}
	if len(data) == 0 {
		return "="
	}
	return base64.StdEncoding.EncodeToString(data)
}

// decodeSASLValue decodes a base64 SASL payload from the wire. "=" and empty
// strings decode to a nil payload.
func decodeSASLValue(v string) ([]byte, error) {
	v = strings.TrimSpace(v)
	if v == "" || v == "=" {
		return nil, nil
	}
	return base64.StdEncoding.DecodeString(v)
}

// AuthError is returned when SASL authentication fails. It exposes the SASL
// failure condition (e.g. "not-authorized") so callers can distinguish a bad
// password from other failures.
type AuthError struct {
	Condition string // SASL failure condition local name
	Text      string // optional human-readable description
}

func (e *AuthError) Error() string {
	switch {
	case e.Text != "" && e.Condition != "":
		return fmt.Sprintf("xmpp: authentication failed: %s: %s", e.Condition, e.Text)
	case e.Condition != "":
		return fmt.Sprintf("xmpp: authentication failed: %s", e.Condition)
	case e.Text != "":
		return fmt.Sprintf("xmpp: authentication failed: %s", e.Text)
	default:
		return "xmpp: authentication failed"
	}
}

// streamErrorElem represents a received <stream:error> (RFC 6120 §4.9).
type streamErrorElem struct {
	XMLName    xml.Name       `xml:"http://etherx.jabber.org/streams error"`
	Text       string         `xml:"urn:ietf:params:xml:ns:xmpp-streams text"`
	Conditions []namedElement `xml:",any"`
}

// StreamError represents an XMPP stream-level error condition.
type StreamError struct {
	Condition string
	Text      string
}

func (e *StreamError) Error() string {
	if e.Text != "" {
		return fmt.Sprintf("xmpp: stream error: %s: %s", e.Condition, e.Text)
	}
	if e.Condition != "" {
		return fmt.Sprintf("xmpp: stream error: %s", e.Condition)
	}
	return "xmpp: stream error"
}

func (e *streamErrorElem) toError() *StreamError {
	cond := ""
	for _, c := range e.Conditions {
		if c.XMLName.Local != "" && c.XMLName.Local != "text" {
			cond = c.XMLName.Local
			break
		}
	}
	return &StreamError{Condition: cond, Text: strings.TrimSpace(e.Text)}
}

// isStreamNS reports whether name belongs to the XMPP stream namespace.
func isStreamNS(name xml.Name) bool { return name.Space == ns.Stream }
