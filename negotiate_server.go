package xmpp

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/meszmate/xmpp-go/internal/ns"
	"github.com/meszmate/xmpp-go/jid"
	"github.com/meszmate/xmpp-go/sasl"
	"github.com/meszmate/xmpp-go/stanza"
	"github.com/meszmate/xmpp-go/stream"
	xmppxml "github.com/meszmate/xmpp-go/xml"
)

// negotiateAndServe runs server-side stream negotiation for an inbound client
// connection and then serves/routes its stanzas until the stream closes. It is
// used when no custom session handler is configured, making xmpp.Server usable
// as a standalone c2s server.
func (s *Server) negotiateAndServe(ctx context.Context, session *Session) {
	defer func() {
		// If the session negotiated Stream Management resumption, park it so its
		// state survives the disconnect for the resumption window instead of
		// tearing it down immediately.
		if s.parkForResume(session) {
			return
		}
		s.closeSessionPlugins(session)
		if !session.RemoteAddr().IsZero() {
			s.router.unregisterIf(session.RemoteAddr(), session)
		}
	}()
	_ = s.serverStream(ctx, session)
}

func (s *Server) serverStream(ctx context.Context, session *Session) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		reader := session.Reader()
		tok, err := reader.Token()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
		start, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}

		// A new stream header (initial or restart): respond with our header and
		// the features available for the current session state. This is
		// <stream:stream> for TCP, or an RFC 7395 <open/> frame for WebSocket.
		if (isStreamNS(start.Name) && start.Name.Local == "stream") ||
			(start.Name.Space == ns.Framing && start.Name.Local == "open") {
			// A jabber:server stream header means an inbound server-to-server
			// connection; hand it to the s2s (dialback) handler.
			if s.s2sEnabled() && attrValue(start.Attr, "xmlns") == ns.Server {
				return s.serveS2SInbound(ctx, session, &start)
			}
			if err := s.writeServerHeader(session); err != nil {
				return err
			}
			if err := s.writeServerFeatures(session); err != nil {
				return err
			}
			continue
		}
		// RFC 7395 stream close.
		if start.Name.Space == ns.Framing && start.Name.Local == "close" {
			return nil
		}

		switch {
		case start.Name.Space == ns.TLS && start.Name.Local == "starttls":
			if err := s.serverStartTLS(ctx, session, reader); err != nil {
				return err
			}
		case start.Name.Space == ns.SASL && start.Name.Local == "auth":
			if err := s.serverAuth(ctx, session, reader, &start); err != nil {
				return err
			}
		case start.Name.Space == ns.SM:
			// XEP-0198 control. enable/resume are server-driven (they issue and
			// consume resumption ids); r/a are handled generically.
			switch start.Name.Local {
			case "enable":
				if err := s.serverEnableSM(ctx, session, &start); err != nil {
					return err
				}
			case "resume":
				if err := s.serverResume(ctx, session, &start); err != nil {
					return err
				}
			default:
				if err := session.handleSMControl(start); err != nil {
					return err
				}
			}
		case start.Name.Local == "iq":
			if err := s.serverIQ(ctx, session, reader, &start); err != nil {
				return err
			}
			session.smCountInbound()
		case start.Name.Local == "message":
			if err := s.serverMessage(ctx, session, reader, &start); err != nil {
				return err
			}
			session.smCountInbound()
		case start.Name.Local == "presence":
			if err := s.serverPresence(ctx, session, reader, &start); err != nil {
				return err
			}
			session.smCountInbound()
		default:
			if err := reader.Skip(); err != nil {
				return err
			}
		}
	}
}

func (s *Server) writeServerHeader(session *Session) error {
	if session.Framing() {
		hdr := fmt.Sprintf(`<open xmlns='%s' from='%s' id='%s' version='1.0'/>`, ns.Framing, s.domain, randomID())
		_, err := session.Writer().WriteRaw([]byte(hdr))
		return err
	}
	from, err := jid.New("", s.domain, "")
	if err != nil {
		return err
	}
	header := stream.Open(stream.Header{
		From:    from,
		ID:      randomID(),
		Lang:    "en",
		Version: stream.DefaultVersion,
		NS:      ns.Client,
	})
	_, err = session.Writer().WriteRaw(header)
	return err
}

func (s *Server) writeServerFeatures(session *Session) error {
	state := session.State()
	secure := state&StateSecure != 0
	authed := state&StateAuthenticated != 0
	bound := state&StateBound != 0

	w := session.Writer()
	start := xml.StartElement{Name: xml.Name{Space: ns.Stream, Local: "features"}}
	if err := w.EncodeToken(start); err != nil {
		return err
	}

	switch {
	case !secure && s.tlsConfig != nil:
		if err := encodeStartTLSFeature(w); err != nil {
			return err
		}
	case !authed:
		if err := encodeMechanismsFeature(w, s.offeredMechanisms(session)); err != nil {
			return err
		}
	case !bound:
		if err := encodeBindFeature(w); err != nil {
			return err
		}
		if err := encodeSMFeature(w); err != nil {
			return err
		}
	}

	return w.EncodeToken(xml.EndElement{Name: start.Name})
}

func (s *Server) serverStartTLS(ctx context.Context, session *Session, reader *xmppxml.StreamReader) error {
	if err := reader.Skip(); err != nil {
		return err
	}
	if s.tlsConfig == nil || session.State()&StateSecure != 0 {
		return session.SendElement(ctx, &tlsFailure{})
	}
	if err := session.SendElement(ctx, &tlsProceed{}); err != nil {
		return err
	}
	if rs, ok := session.Transport().(tlsRoleSetter); ok {
		rs.SetTLSRole(true)
	}
	if err := session.Transport().StartTLS(s.tlsConfig); err != nil {
		return err
	}
	session.SetState(StateSecure)
	session.Restart()
	return nil
}

func (s *Server) serverAuth(ctx context.Context, session *Session, reader *xmppxml.StreamReader, start *xml.StartElement) error {
	if session.State()&StateAuthenticated != 0 {
		_ = reader.Skip()
		return s.sendSASLFailure(ctx, session, "not-authorized")
	}

	var auth saslAuth
	if err := reader.DecodeElement(&auth, start); err != nil {
		return err
	}
	mech := strings.ToUpper(strings.TrimSpace(auth.Mechanism))

	switch {
	case mech == "PLAIN":
		return s.serverAuthPlain(ctx, session, auth.Value)
	case strings.HasPrefix(mech, "SCRAM-"):
		return s.serverAuthSCRAM(ctx, session, reader, mech, auth.Value)
	case mech == "EXTERNAL":
		return s.serverAuthExternal(ctx, session, auth.Value)
	case mech == "ANONYMOUS":
		if !s.opts.allowAnonymous {
			return s.sendSASLFailure(ctx, session, "invalid-mechanism")
		}
		return s.completeAuth(ctx, session, "anon-"+randomID()[:12])
	default:
		return s.sendSASLFailure(ctx, session, "invalid-mechanism")
	}
}

// serverAuthExternal authenticates a client via its TLS client certificate
// (SASL EXTERNAL, RFC 6120 §6). The certificate must have been verified during
// the TLS handshake and map to a local username via the configured identity
// function. An optional authorization identity in the request must match the
// derived identity.
func (s *Server) serverAuthExternal(ctx context.Context, session *Session, value string) error {
	if s.opts.externalAuth == nil || session.State()&StateSecure == 0 {
		return s.sendSASLFailure(ctx, session, "invalid-mechanism")
	}
	cs, ok := session.Transport().ConnectionState()
	if !ok || len(cs.PeerCertificates) == 0 {
		return s.sendSASLFailure(ctx, session, "not-authorized")
	}
	// Require that the presented certificate chained to a configured client CA.
	if len(cs.VerifiedChains) == 0 {
		return s.sendSASLFailure(ctx, session, "not-authorized")
	}
	username, ok := s.opts.externalAuth(cs.PeerCertificates[0])
	if !ok || username == "" {
		return s.sendSASLFailure(ctx, session, "not-authorized")
	}
	// An empty authzid ("=" / "") means "use the identity from the certificate".
	// A non-empty one must match the derived identity (bare or full).
	if reqID, err := decodeSASLValue(value); err == nil && len(reqID) > 0 {
		got := string(reqID)
		if got != username && got != username+"@"+s.domain {
			return s.sendSASLFailure(ctx, session, "invalid-authzid")
		}
	}
	return s.completeAuth(ctx, session, username)
}

func (s *Server) serverAuthPlain(ctx context.Context, session *Session, value string) error {
	payload, err := decodeSASLValue(value)
	if err != nil {
		return s.sendSASLFailure(ctx, session, "incorrect-encoding")
	}
	parts := strings.SplitN(string(payload), "\x00", 3)
	if len(parts) != 3 || strings.TrimSpace(parts[1]) == "" {
		return s.sendSASLFailure(ctx, session, "malformed-request")
	}
	username := strings.TrimSpace(parts[1])
	password := parts[2]

	if s.opts.authFunc == nil {
		return s.sendSASLFailure(ctx, session, "temporary-auth-failure")
	}
	ok, err := s.opts.authFunc(username, password)
	if err != nil || !ok {
		return s.sendSASLFailure(ctx, session, "not-authorized")
	}
	return s.completeAuth(ctx, session, username)
}

func (s *Server) completeAuth(ctx context.Context, session *Session, username string) error {
	j, err := jid.New(username, s.domain, "")
	if err != nil {
		return s.sendSASLFailure(ctx, session, "not-authorized")
	}
	session.SetRemoteAddr(j)
	session.SetState(StateAuthenticated)
	if err := session.SendElement(ctx, &saslSuccess{}); err != nil {
		return err
	}
	session.Restart()
	return nil
}

// serverAuthSCRAM runs the multi-round server side of a SCRAM-SHA-* exchange.
func (s *Server) serverAuthSCRAM(ctx context.Context, session *Session, reader *xmppxml.StreamReader, mech, initialValue string) error {
	srv := sasl.NewSCRAMServer(mech, s.passwordLookup(ctx))
	if srv == nil {
		return s.sendSASLFailure(ctx, session, "invalid-mechanism")
	}
	// For a -PLUS exchange, bind to this TLS channel via tls-server-end-point.
	if strings.HasSuffix(mech, "-PLUS") {
		if len(s.tlsServerEndpoint) == 0 {
			return s.sendSASLFailure(ctx, session, "invalid-mechanism")
		}
		srv.SetChannelBinding(s.tlsServerEndpoint)
	}

	clientFirst, err := decodeSASLValue(initialValue)
	if err != nil || len(clientFirst) == 0 {
		return s.sendSASLFailure(ctx, session, "malformed-request")
	}
	serverFirst, err := srv.Start(clientFirst)
	if err != nil {
		return s.sendSASLFailure(ctx, session, scramFailureCondition(err))
	}
	if err := session.SendElement(ctx, &saslChallenge{Value: encodeSASLValue(serverFirst)}); err != nil {
		return err
	}

	clientFinal, err := readSASLResponse(reader)
	if err != nil {
		return err
	}
	serverFinal, err := srv.Finish(clientFinal)
	if err != nil {
		return s.sendSASLFailure(ctx, session, scramFailureCondition(err))
	}

	j, err := jid.New(srv.Username(), s.domain, "")
	if err != nil {
		return s.sendSASLFailure(ctx, session, "not-authorized")
	}
	if err := session.SendElement(ctx, &saslSuccess{Value: encodeSASLValue(serverFinal)}); err != nil {
		return err
	}
	session.SetRemoteAddr(j)
	session.SetSASLMechanism(mech)
	session.SetState(StateAuthenticated)
	session.Restart()
	return nil
}

// readSASLResponse reads the next <response> (or <abort>) element and returns
// its decoded payload.
func readSASLResponse(reader *xmppxml.StreamReader) ([]byte, error) {
	for {
		tok, err := reader.Token()
		if err != nil {
			return nil, err
		}
		se, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		if se.Name.Space == ns.SASL && se.Name.Local == "response" {
			var r saslResponse
			if err := reader.DecodeElement(&r, &se); err != nil {
				return nil, err
			}
			return decodeSASLValue(r.Value)
		}
		if se.Name.Space == ns.SASL && se.Name.Local == "abort" {
			return nil, errors.New("xmpp: client aborted SASL")
		}
		if err := reader.Skip(); err != nil {
			return nil, err
		}
	}
}

func scramFailureCondition(err error) string {
	switch {
	case errors.Is(err, sasl.ErrAuthFailed):
		return "not-authorized"
	case errors.Is(err, sasl.ErrChannelBinding):
		return "not-authorized"
	default:
		return "malformed-request"
	}
}

func encodeStartTLSFeature(w *xmppxml.StreamWriter) error {
	feat := xml.StartElement{Name: xml.Name{Space: ns.TLS, Local: "starttls"}}
	if err := w.EncodeToken(feat); err != nil {
		return err
	}
	req := xml.StartElement{Name: xml.Name{Local: "required"}}
	if err := w.EncodeToken(req); err != nil {
		return err
	}
	if err := w.EncodeToken(xml.EndElement{Name: req.Name}); err != nil {
		return err
	}
	return w.EncodeToken(xml.EndElement{Name: feat.Name})
}

func encodeMechanismsFeature(w *xmppxml.StreamWriter, mechs []string) error {
	root := xml.StartElement{Name: xml.Name{Space: ns.SASL, Local: "mechanisms"}}
	if err := w.EncodeToken(root); err != nil {
		return err
	}
	for _, m := range mechs {
		el := xml.StartElement{Name: xml.Name{Space: ns.SASL, Local: "mechanism"}}
		if err := w.EncodeToken(el); err != nil {
			return err
		}
		if err := w.EncodeToken(xml.CharData(m)); err != nil {
			return err
		}
		if err := w.EncodeToken(xml.EndElement{Name: el.Name}); err != nil {
			return err
		}
	}
	return w.EncodeToken(xml.EndElement{Name: root.Name})
}

func encodeBindFeature(w *xmppxml.StreamWriter) error {
	feat := xml.StartElement{Name: xml.Name{Space: ns.Bind, Local: "bind"}}
	if err := w.EncodeToken(feat); err != nil {
		return err
	}
	return w.EncodeToken(xml.EndElement{Name: feat.Name})
}

func encodeSMFeature(w *xmppxml.StreamWriter) error {
	feat := xml.StartElement{Name: xml.Name{Space: ns.SM, Local: "sm"}}
	if err := w.EncodeToken(feat); err != nil {
		return err
	}
	return w.EncodeToken(xml.EndElement{Name: feat.Name})
}

func (s *Server) serverIQ(ctx context.Context, session *Session, reader *xmppxml.StreamReader, start *xml.StartElement) error {
	var iq stanza.IQ
	if err := reader.DecodeElement(&iq, start); err != nil {
		return err
	}

	if isBindRequest(&iq) {
		return s.serverBind(ctx, session, &iq)
	}

	if session.State()&StateReady == 0 {
		if iq.Type == stanza.IQGet || iq.Type == stanza.IQSet {
			return session.Send(ctx, iq.ErrorIQ(stanza.NewStanzaError(stanza.ErrorTypeAuth, stanza.ErrorNotAuthorized, "authenticate and bind first")))
		}
		return nil
	}

	// Per-session plugins get first refusal on IQs directed at the server or the
	// sender's own account, so they can extend the service surface (custom
	// namespaces) or override built-ins. Routed (peer-directed) IQs are not
	// intercepted.
	if s.isServiceDirected(session, &iq) {
		if handled, err := s.dispatchSessionPlugin(ctx, session, &iq); handled {
			return err
		}
		return s.serverServiceIQ(ctx, session, &iq)
	}
	return s.routeIQ(ctx, session, &iq)
}

func isBindRequest(iq *stanza.IQ) bool {
	if iq == nil || iq.Type != stanza.IQSet || len(iq.Query) == 0 {
		return false
	}
	var req BindRequest
	if err := xml.Unmarshal(iq.Query, &req); err != nil {
		return false
	}
	return req.XMLName.Space == ns.Bind && req.XMLName.Local == "bind"
}

func (s *Server) serverBind(ctx context.Context, session *Session, iq *stanza.IQ) error {
	if session.State()&StateAuthenticated == 0 {
		return session.Send(ctx, iq.ErrorIQ(stanza.NewStanzaError(stanza.ErrorTypeAuth, stanza.ErrorNotAuthorized, "not authenticated")))
	}
	username := session.RemoteAddr().Local()
	if username == "" {
		return session.Send(ctx, iq.ErrorIQ(stanza.NewStanzaError(stanza.ErrorTypeAuth, stanza.ErrorNotAuthorized, "not authenticated")))
	}

	var req BindRequest
	_ = xml.Unmarshal(iq.Query, &req)
	resource := strings.TrimSpace(req.Resource)
	if resource == "" {
		resource = randomID()
	}

	full, err := jid.New(username, s.domain, resource)
	if err != nil {
		return session.Send(ctx, iq.ErrorIQ(stanza.NewStanzaError(stanza.ErrorTypeModify, stanza.ErrorJIDMalformed, "invalid jid")))
	}

	session.SetRemoteAddr(full)
	session.SetState(StateBound | StateReady)
	s.router.register(full, session)

	// Activate per-session plugins now that the session has an identity and is
	// ready to exchange stanzas.
	s.initSessionPlugins(ctx, session)

	result := iq.ResultIQ()
	payload := &stanza.IQPayload{IQ: *result, Payload: &BindResult{JID: full.String()}}
	return session.SendElement(ctx, payload)
}

func (s *Server) serverMessage(ctx context.Context, session *Session, reader *xmppxml.StreamReader, start *xml.StartElement) error {
	var msg stanza.Message
	if err := reader.DecodeElement(&msg, start); err != nil {
		return err
	}
	if session.State()&StateReady == 0 {
		return nil
	}
	if msg.From.IsZero() {
		msg.From = session.RemoteAddr()
	}
	// A message to a remote domain is federated over server-to-server.
	if !msg.To.IsZero() && !s.isLocalDomain(msg.To) && s.s2sEnabled() {
		s.deliverRemote(ctx, &msg, msg.To)
		return nil
	}
	for _, dst := range s.router.targets(msg.To) {
		if dst == session {
			continue
		}
		_ = dst.Send(ctx, &msg)
	}
	return nil
}

func (s *Server) serverPresence(ctx context.Context, session *Session, reader *xmppxml.StreamReader, start *xml.StartElement) error {
	var pres stanza.Presence
	if err := reader.DecodeElement(&pres, start); err != nil {
		return err
	}
	if session.State()&StateReady == 0 {
		return nil
	}
	return s.dispatchPresence(ctx, session, &pres)
}

func (s *Server) routeIQ(ctx context.Context, session *Session, iq *stanza.IQ) error {
	// Federate IQs addressed to a remote domain over server-to-server.
	if !iq.To.IsZero() && !s.isLocalDomain(iq.To) && s.s2sEnabled() {
		if iq.From.IsZero() {
			iq.From = session.RemoteAddr()
		}
		s.deliverRemote(ctx, iq, iq.To)
		return nil
	}
	if iq.To.IsZero() || iq.To.IsDomainOnly() {
		if iq.Type == stanza.IQGet || iq.Type == stanza.IQSet {
			return session.Send(ctx, iq.ErrorIQ(stanza.NewStanzaError(stanza.ErrorTypeCancel, stanza.ErrorServiceUnavailable, "unsupported iq")))
		}
		return nil
	}
	if iq.From.IsZero() {
		iq.From = session.RemoteAddr()
	}
	targets := s.router.targets(iq.To)
	if len(targets) == 0 {
		if iq.Type == stanza.IQGet || iq.Type == stanza.IQSet {
			return session.Send(ctx, iq.ErrorIQ(stanza.NewStanzaError(stanza.ErrorTypeCancel, stanza.ErrorItemNotFound, "recipient not found")))
		}
		return nil
	}
	for _, dst := range targets {
		if dst == session {
			continue
		}
		_ = dst.Send(ctx, iq)
		if iq.To.IsFull() {
			break
		}
	}
	return nil
}

func (s *Server) sendSASLFailure(ctx context.Context, session *Session, condition string) error {
	payload := "<failure xmlns='" + ns.SASL + "'><" + condition + "/></failure>"
	return session.SendRaw(ctx, strings.NewReader(payload))
}

// offeredMechanisms returns the SASL mechanisms the server advertises for the
// current session state and configuration.
func (s *Server) offeredMechanisms(session *Session) []string {
	var mechs []string
	// EXTERNAL is offered first (strongest) once the client has presented a
	// certificate over a secure channel that we can map to an identity.
	if s.opts.externalAuth != nil && session.State()&StateSecure != 0 {
		if cs, ok := session.Transport().ConnectionState(); ok && len(cs.PeerCertificates) > 0 {
			mechs = append(mechs, "EXTERNAL")
		}
	}
	// SCRAM-*-PLUS (channel binding) is offered when we are over real TLS with a
	// server certificate, binding the SASL exchange to this TLS channel.
	if s.canSCRAM() && len(s.tlsServerEndpoint) > 0 && session.State()&StateSecure != 0 {
		if cs, ok := session.Transport().ConnectionState(); ok && cs.Version != 0 {
			mechs = append(mechs, "SCRAM-SHA-256-PLUS", "SCRAM-SHA-1-PLUS")
		}
	}
	if s.canSCRAM() {
		mechs = append(mechs, "SCRAM-SHA-256", "SCRAM-SHA-1")
	}
	if s.opts.authFunc != nil {
		mechs = append(mechs, "PLAIN")
	}
	if s.opts.allowAnonymous {
		mechs = append(mechs, "ANONYMOUS")
	}
	return mechs
}

func (s *Server) canSCRAM() bool {
	if s.opts.storage == nil {
		return false
	}
	return s.opts.storage.UserStore() != nil
}

// passwordLookup returns a sasl.PasswordLookup backed by the configured user
// store, used for server-side SCRAM.
func (s *Server) passwordLookup(ctx context.Context) sasl.PasswordLookup {
	if s.opts.storage == nil {
		return nil
	}
	us := s.opts.storage.UserStore()
	if us == nil {
		return nil
	}
	return func(username string) (string, bool) {
		u, err := us.GetUser(ctx, username)
		if err != nil || u == nil {
			return "", false
		}
		return u.Password, true
	}
}

func randomID() string {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return "id-fallback"
	}
	return hex.EncodeToString(b)
}
