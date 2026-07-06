package xmpp

import (
	"context"
	"crypto/tls"
	"encoding/xml"
	"errors"
	"fmt"
	"strings"

	"github.com/meszmate/xmpp-go/internal/ns"
	"github.com/meszmate/xmpp-go/jid"
	"github.com/meszmate/xmpp-go/sasl"
	"github.com/meszmate/xmpp-go/stanza"
	"github.com/meszmate/xmpp-go/stream"
)

// errNoFeatures is returned when the server advertises no feature the client can
// act on and negotiation has not otherwise completed.
var errNoFeatures = errors.New("xmpp: no negotiable stream features offered by server")

// tlsRoleSetter is implemented by transports that can be told, before StartTLS,
// whether to perform the TLS server or client handshake. Setting the role
// explicitly is required for client-certificate (mutual TLS) authentication.
type tlsRoleSetter interface {
	SetTLSRole(server bool)
}

// negotiate drives the full client-side stream negotiation on session:
// stream open -> STARTTLS -> SASL -> resource bind, reaching StateReady.
//
// It loops one feature per round, re-opening the stream after any step that
// alters it (STARTTLS, SASL success), exactly as required by RFC 6120.
func (c *Client) negotiate(ctx context.Context, session *Session, resume bool) error {
	domain := c.addr.Domain()
	domainJID, err := jid.New("", domain, "")
	if err != nil {
		return fmt.Errorf("xmpp: invalid domain %q: %w", domain, err)
	}

	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := c.openStream(session, domainJID); err != nil {
			return fmt.Errorf("xmpp: open stream: %w", err)
		}
		feats, err := c.readFeatures(session)
		if err != nil {
			return err
		}

		state := session.State()
		secure := state&StateSecure != 0
		authed := state&StateAuthenticated != 0
		bound := state&StateBound != 0

		switch {
		case feats.StartTLS != nil && !secure && !c.opts.noTLS:
			if err := c.doStartTLS(session, domain); err != nil {
				return err
			}
			session.SetState(StateSecure)
			session.Restart()

		case feats.Mechanisms != nil && !authed:
			if feats.StartTLS != nil && feats.StartTLS.Required != nil && !secure && !c.opts.noTLS {
				// STARTTLS is required; the STARTTLS branch above handles it
				// first, so reaching here means we cannot proceed securely.
				return errors.New("xmpp: server requires STARTTLS")
			}
			if err := c.doSASL(session, feats.Mechanisms.Mechanism); err != nil {
				return err
			}
			session.SetState(StateAuthenticated)
			session.Restart()

		case feats.Bind != nil && authed && !bound:
			// Resume an earlier session instead of binding a new resource, when
			// requested and the server advertises Stream Management.
			if resume && feats.SM != nil && session.SMPrevID() != "" {
				resumed, rErr := c.doResume(session)
				if rErr != nil {
					return rErr
				}
				if resumed {
					session.SetState(StateBound | StateReady)
					return nil
				}
				// The server declined resumption: discard the stale SM state and
				// fall back to a normal bind on this stream.
				session.resetSM()
			}
			if err := c.doBind(session); err != nil {
				return err
			}
			session.SetState(StateBound | StateReady)
			// Enable Stream Management (XEP-0198) if the server advertised it
			// and the client did not opt out.
			if feats.SM != nil && !c.opts.noSM {
				if err := c.enableSM(session); err != nil {
					return err
				}
			}
			return nil

		default:
			// Nothing left we can negotiate.
			if feats.StartTLS != nil && !secure && c.opts.noTLS {
				return errors.New("xmpp: server offers only STARTTLS but WithNoTLS is set")
			}
			if authed && bound {
				session.SetState(StateReady)
				return nil
			}
			if authed {
				// Authenticated but no bind advertised (some legacy servers):
				// treat the stream as usable.
				session.SetState(StateReady)
				return nil
			}
			return errNoFeatures
		}
	}
}

// openStream writes the initial (or restarted) client stream header. For
// WebSocket sessions this is an RFC 7395 <open/> frame instead of a
// <stream:stream> element.
func (c *Client) openStream(session *Session, to jid.JID) error {
	lang := c.opts.lang
	if lang == "" {
		lang = "en"
	}
	if session.Framing() {
		hdr := fmt.Sprintf(`<open xmlns='%s' to='%s' version='1.0' xml:lang='%s'/>`, ns.Framing, to.String(), lang)
		_, err := session.Writer().WriteRaw([]byte(hdr))
		return err
	}
	header := stream.Open(stream.Header{
		To:      to,
		Version: stream.DefaultVersion,
		Lang:    lang,
		NS:      ns.Client,
	})
	_, err := session.Writer().WriteRaw(header)
	return err
}

// readFeatures reads tokens until the peer's <stream:features> is decoded,
// surfacing a <stream:error> as a *StreamError.
func (c *Client) readFeatures(session *Session) (*streamFeatures, error) {
	reader := session.Reader()
	for {
		tok, err := reader.Token()
		if err != nil {
			return nil, fmt.Errorf("xmpp: read stream features: %w", err)
		}
		se, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		switch {
		case isStreamNS(se.Name) && se.Name.Local == "features":
			var f streamFeatures
			if err := reader.DecodeElement(&f, &se); err != nil {
				return nil, fmt.Errorf("xmpp: decode stream features: %w", err)
			}
			return &f, nil
		case isStreamNS(se.Name) && se.Name.Local == "error":
			var e streamErrorElem
			if err := reader.DecodeElement(&e, &se); err != nil {
				return nil, fmt.Errorf("xmpp: decode stream error: %w", err)
			}
			return nil, e.toError()
		case isStreamNS(se.Name) && se.Name.Local == "stream":
			// Peer stream header; features follow.
			continue
		default:
			if err := reader.Skip(); err != nil {
				return nil, err
			}
		}
	}
}

// doStartTLS performs the STARTTLS exchange and upgrades the transport.
func (c *Client) doStartTLS(session *Session, domain string) error {
	if err := session.Writer().Encode(&tlsStartTLS{}); err != nil {
		return fmt.Errorf("xmpp: send starttls: %w", err)
	}

	reader := session.Reader()
	for {
		tok, err := reader.Token()
		if err != nil {
			return fmt.Errorf("xmpp: read starttls response: %w", err)
		}
		se, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		if se.Name.Space != ns.TLS {
			if err := reader.Skip(); err != nil {
				return err
			}
			continue
		}
		switch se.Name.Local {
		case "proceed":
			// Force the TLS client role so that presenting a client certificate
			// (SASL EXTERNAL) is not misread as acting as a TLS server.
			if rs, ok := session.Transport().(tlsRoleSetter); ok {
				rs.SetTLSRole(false)
			}
			if err := session.Transport().StartTLS(c.tlsConfig(domain)); err != nil {
				return fmt.Errorf("xmpp: TLS handshake: %w", err)
			}
			return nil
		case "failure":
			return errors.New("xmpp: server refused STARTTLS")
		default:
			if err := reader.Skip(); err != nil {
				return err
			}
		}
	}
}

// tlsConfig returns the effective TLS config for the client, defaulting the
// ServerName to the target domain for SNI and certificate validation.
func (c *Client) tlsConfig(domain string) *tls.Config {
	var cfg *tls.Config
	if c.opts.tlsConfig != nil {
		cfg = c.opts.tlsConfig.Clone()
	} else {
		cfg = &tls.Config{MinVersion: tls.VersionTLS12}
	}
	if cfg.ServerName == "" {
		cfg.ServerName = domain
	}
	return cfg
}

// doSASL selects and runs a SASL mechanism from the offered list.
func (c *Client) doSASL(session *Session, offered []string) error {
	creds := sasl.Credentials{
		Username: c.addr.Local(),
		Password: c.password,
	}
	// When the connection is over real TLS with a server certificate, enable
	// SCRAM-*-PLUS channel binding (tls-server-end-point) so the SASL exchange
	// is cryptographically bound to this TLS channel.
	cb := clientChannelBinding(session)
	if len(cb) > 0 {
		creds.CBType = "tls-server-end-point"
		creds.ChannelBinding = cb
	}

	mechs := c.clientSASLMechanisms(creds, len(cb) > 0)
	neg := sasl.NewNegotiator(creds, mechs...)
	mech, err := neg.Select(offered)
	if err != nil {
		return fmt.Errorf("xmpp: %w (server offered %v)", err, offered)
	}

	// Never transmit a cleartext password over an unencrypted connection.
	if mech.Name() == "PLAIN" && session.State()&StateSecure == 0 && !c.opts.insecureSASL {
		return errors.New("xmpp: refusing PLAIN over an unencrypted connection; enable TLS or use WithInsecureSASL to override")
	}
	session.SetSASLMechanism(mech.Name())

	writer := session.Writer()
	reader := session.Reader()

	initial, err := mech.Start()
	if err != nil {
		return fmt.Errorf("xmpp: sasl start: %w", err)
	}
	if err := writer.Encode(&saslAuth{Mechanism: mech.Name(), Value: encodeSASLValue(initial)}); err != nil {
		return fmt.Errorf("xmpp: send auth: %w", err)
	}

	for {
		tok, err := reader.Token()
		if err != nil {
			return fmt.Errorf("xmpp: read sasl response: %w", err)
		}
		se, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		if se.Name.Space != ns.SASL {
			if err := reader.Skip(); err != nil {
				return err
			}
			continue
		}
		switch se.Name.Local {
		case "challenge":
			var ch saslChallenge
			if err := reader.DecodeElement(&ch, &se); err != nil {
				return err
			}
			data, err := decodeSASLValue(ch.Value)
			if err != nil {
				return fmt.Errorf("xmpp: decode challenge: %w", err)
			}
			resp, err := mech.Next(data)
			if err != nil {
				return fmt.Errorf("xmpp: sasl step: %w", err)
			}
			if err := writer.Encode(&saslResponse{Value: encodeSASLValue(resp)}); err != nil {
				return fmt.Errorf("xmpp: send response: %w", err)
			}
		case "success":
			var su saslSuccess
			if err := reader.DecodeElement(&su, &se); err != nil {
				return err
			}
			if su.Value != "" && !mech.Completed() {
				data, decErr := decodeSASLValue(su.Value)
				if decErr != nil {
					return fmt.Errorf("xmpp: decode success: %w", decErr)
				}
				if _, err := mech.Next(data); err != nil {
					return fmt.Errorf("xmpp: verify server signature: %w", err)
				}
			}
			return nil
		case "failure":
			var fa saslFailure
			if err := reader.DecodeElement(&fa, &se); err != nil {
				return err
			}
			return &AuthError{Condition: fa.Condition(), Text: trimText(fa.Text)}
		default:
			if err := reader.Skip(); err != nil {
				return err
			}
		}
	}
}

// defaultSASLOrder is the client's mechanism preference when none is configured.
var defaultSASLOrder = []string{"SCRAM-SHA-512", "SCRAM-SHA-256", "SCRAM-SHA-1", "PLAIN"}

// defaultSASLOrderPlus prefers channel-binding (-PLUS) variants first, used when
// TLS channel binding is available.
var defaultSASLOrderPlus = []string{
	"SCRAM-SHA-512-PLUS", "SCRAM-SHA-256-PLUS", "SCRAM-SHA-1-PLUS",
	"SCRAM-SHA-512", "SCRAM-SHA-256", "SCRAM-SHA-1", "PLAIN",
}

// hasMechanism reports whether name (case-insensitive) is in the list.
func hasMechanism(list []string, name string) bool {
	for _, m := range list {
		if strings.EqualFold(strings.TrimSpace(m), name) {
			return true
		}
	}
	return false
}

// clientSASLMechanisms returns candidate mechanisms in preference order,
// honoring WithSASLMechanisms when set. SCRAM is preferred over PLAIN, and
// channel-binding (-PLUS) variants are preferred when cbAvailable.
func (c *Client) clientSASLMechanisms(creds sasl.Credentials, cbAvailable bool) []sasl.Mechanism {
	build := map[string]func() sasl.Mechanism{
		"SCRAM-SHA-512":      func() sasl.Mechanism { return sasl.NewSCRAMSHA512(creds) },
		"SCRAM-SHA-256":      func() sasl.Mechanism { return sasl.NewSCRAMSHA256(creds) },
		"SCRAM-SHA-1":        func() sasl.Mechanism { return sasl.NewSCRAMSHA1(creds) },
		"SCRAM-SHA-512-PLUS": func() sasl.Mechanism { return sasl.NewSCRAMSHA512Plus(creds) },
		"SCRAM-SHA-256-PLUS": func() sasl.Mechanism { return sasl.NewSCRAMSHA256Plus(creds) },
		"SCRAM-SHA-1-PLUS":   func() sasl.Mechanism { return sasl.NewSCRAMSHA1Plus(creds) },
		"PLAIN":              func() sasl.Mechanism { return sasl.NewPlain(creds) },
		"EXTERNAL":           func() sasl.Mechanism { return sasl.NewExternal(creds.AuthzID) },
		"ANONYMOUS":          func() sasl.Mechanism { return sasl.NewAnonymous("") },
	}
	order := c.opts.saslMechanisms
	if len(order) == 0 {
		if cbAvailable {
			order = defaultSASLOrderPlus
		} else {
			order = defaultSASLOrder
		}
	}
	out := make([]sasl.Mechanism, 0, len(order))
	for _, name := range order {
		if f, ok := build[strings.ToUpper(strings.TrimSpace(name))]; ok {
			out = append(out, f())
		}
	}
	return out
}

// doBind performs resource binding and records the assigned full JID.
func (c *Client) doBind(session *Session) error {
	iq := stanza.NewIQ(stanza.IQSet)
	iq.ID = "bind_" + stanza.GenerateID()[:12]
	payload := &stanza.IQPayload{IQ: *iq, Payload: &BindRequest{Resource: c.opts.resource}}
	if err := session.SendElement(context.Background(), payload); err != nil {
		return fmt.Errorf("xmpp: send bind: %w", err)
	}

	reader := session.Reader()
	for {
		tok, err := reader.Token()
		if err != nil {
			return fmt.Errorf("xmpp: read bind result: %w", err)
		}
		se, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		if se.Name.Local != "iq" {
			if err := reader.Skip(); err != nil {
				return err
			}
			continue
		}
		var res stanza.IQ
		if err := reader.DecodeElement(&res, &se); err != nil {
			return fmt.Errorf("xmpp: decode bind result: %w", err)
		}
		if res.ID != iq.ID {
			continue
		}
		if res.Type == stanza.IQError {
			if res.Error != nil {
				return fmt.Errorf("xmpp: bind rejected: %s", res.Error.Condition)
			}
			return errors.New("xmpp: bind rejected")
		}
		var result BindResult
		if err := xml.Unmarshal(res.Query, &result); err != nil {
			return fmt.Errorf("xmpp: invalid bind payload: %w", err)
		}
		full, err := jid.Parse(result.JID)
		if err != nil {
			return fmt.Errorf("xmpp: invalid bound JID %q: %w", result.JID, err)
		}
		session.SetLocalAddr(full)
		return nil
	}
}

// enableSM performs the XEP-0198 <enable/> handshake, requesting resumption
// support. On <enabled/> it marks the session SM-enabled and, if the server
// granted a resumption id, records it so the session can later be resumed. On
// <failed/> it silently proceeds without SM.
func (c *Client) enableSM(session *Session) error {
	enable := "<enable xmlns='" + ns.SM + "' resume='true'/>"
	if c.opts.noSMResume {
		enable = "<enable xmlns='" + ns.SM + "'/>"
	}
	if err := session.writeSMRaw(enable); err != nil {
		return fmt.Errorf("xmpp: send sm enable: %w", err)
	}
	reader := session.Reader()
	for {
		tok, err := reader.Token()
		if err != nil {
			return fmt.Errorf("xmpp: read sm enabled: %w", err)
		}
		se, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		if se.Name.Space != ns.SM {
			if err := reader.Skip(); err != nil {
				return err
			}
			continue
		}
		if err := reader.Skip(); err != nil {
			return err
		}
		if se.Name.Local == "enabled" {
			session.EnableSM()
			if id := attrValue(se.Attr, "id"); id != "" && attrTrue(se.Attr, "resume") {
				session.SetSMResume(id, int(parseUintAttr(se.Attr, "max")))
			}
		}
		return nil
	}
}

// doResume attempts XEP-0198 §5 resumption on an authenticated stream: it sends
// <resume/> with the prior resumption id and this session's handled count, and
// on <resumed/> replays any unacknowledged outbound stanzas. It reports whether
// the server accepted the resumption; a <failed/> reply returns (false, nil).
func (c *Client) doResume(session *Session) (bool, error) {
	previd := session.SMPrevID()
	req := fmt.Sprintf("<resume xmlns='%s' previd='%s' h='%d'/>", ns.SM, previd, session.SMHandled())
	if err := session.writeSMRaw(req); err != nil {
		return false, fmt.Errorf("xmpp: send sm resume: %w", err)
	}
	reader := session.Reader()
	for {
		tok, err := reader.Token()
		if err != nil {
			return false, fmt.Errorf("xmpp: read sm resume result: %w", err)
		}
		se, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		if se.Name.Space != ns.SM {
			if err := reader.Skip(); err != nil {
				return false, err
			}
			continue
		}
		switch se.Name.Local {
		case "resumed":
			serverH := parseUintAttr(se.Attr, "h")
			if err := reader.Skip(); err != nil {
				return false, err
			}
			// The server acknowledged serverH of our stanzas; resend the rest.
			session.mu.Lock()
			replayErr := session.replayUnacked(serverH)
			session.mu.Unlock()
			if replayErr != nil {
				return false, fmt.Errorf("xmpp: replay on resume: %w", replayErr)
			}
			return true, nil
		case "failed":
			if err := reader.Skip(); err != nil {
				return false, err
			}
			return false, nil
		default:
			if err := reader.Skip(); err != nil {
				return false, err
			}
		}
	}
}

func trimText(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\n' || s[0] == '\t' || s[0] == '\r') {
		s = s[1:]
	}
	for len(s) > 0 {
		last := s[len(s)-1]
		if last == ' ' || last == '\n' || last == '\t' || last == '\r' {
			s = s[:len(s)-1]
			continue
		}
		break
	}
	return s
}
