package main

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"encoding/xml"
	"io"
	"log"
	"strings"
	"sync"

	xmpp "github.com/meszmate/xmpp-go"
	"github.com/meszmate/xmpp-go/internal/ns"
	"github.com/meszmate/xmpp-go/jid"
	"github.com/meszmate/xmpp-go/stanza"
	"github.com/meszmate/xmpp-go/storage"
	"github.com/meszmate/xmpp-go/stream"
	xmppxml "github.com/meszmate/xmpp-go/xml"
)

var globalRouter = newSessionRouter()

type sessionRouter struct {
	mu     sync.RWMutex
	byFull map[string]*xmpp.Session
	byBare map[string]map[string]*xmpp.Session
}

func newSessionRouter() *sessionRouter {
	return &sessionRouter{
		byFull: make(map[string]*xmpp.Session),
		byBare: make(map[string]map[string]*xmpp.Session),
	}
}

func (r *sessionRouter) register(full jid.JID, session *xmpp.Session) {
	fullStr := full.String()
	if fullStr == "" {
		return
	}
	bare := full.Bare().String()

	r.mu.Lock()
	defer r.mu.Unlock()
	r.byFull[fullStr] = session
	if r.byBare[bare] == nil {
		r.byBare[bare] = make(map[string]*xmpp.Session)
	}
	r.byBare[bare][fullStr] = session
}

func (r *sessionRouter) unregister(full jid.JID) {
	fullStr := full.String()
	if fullStr == "" {
		return
	}
	bare := full.Bare().String()

	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.byFull, fullStr)
	if sessions, ok := r.byBare[bare]; ok {
		delete(sessions, fullStr)
		if len(sessions) == 0 {
			delete(r.byBare, bare)
		}
	}
}

func (r *sessionRouter) targets(to jid.JID) []*xmpp.Session {
	if to.IsZero() {
		return nil
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	if to.IsFull() {
		if s, ok := r.byFull[to.String()]; ok {
			return []*xmpp.Session{s}
		}
		return nil
	}

	bare := to.Bare().String()
	sessions := r.byBare[bare]
	if len(sessions) == 0 {
		return nil
	}
	out := make([]*xmpp.Session, 0, len(sessions))
	for _, s := range sessions {
		out = append(out, s)
	}
	return out
}

type startTLSRequest struct {
	XMLName xml.Name `xml:"urn:ietf:params:xml:ns:xmpp-tls starttls"`
}

type startTLSProceed struct {
	XMLName xml.Name `xml:"urn:ietf:params:xml:ns:xmpp-tls proceed"`
}

type startTLSFailure struct {
	XMLName xml.Name `xml:"urn:ietf:params:xml:ns:xmpp-tls failure"`
}

type saslAuth struct {
	XMLName   xml.Name `xml:"urn:ietf:params:xml:ns:xmpp-sasl auth"`
	Mechanism string   `xml:"mechanism,attr"`
	Value     string   `xml:",chardata"`
}

type saslSuccess struct {
	XMLName xml.Name `xml:"urn:ietf:params:xml:ns:xmpp-sasl success"`
}

func serveSession(ctx context.Context, session *xmpp.Session, cfg Config, store storage.Storage) {
	regHandler := newRegistrationHandler(cfg.Registration, store)
	tlsConfig, err := buildTLSConfig(cfg)
	if err != nil {
		log.Printf("session tls setup error: %v", err)
		return
	}

	if _, secure := session.Transport().ConnectionState(); secure {
		session.SetState(xmpp.StateSecure)
	}

	var authenticatedUser string
	defer func() {
		globalRouter.unregister(session.RemoteAddr())
	}()

	if err := serveStream(ctx, session, regHandler, cfg, tlsConfig, &authenticatedUser); err != nil {
		log.Printf("session error: %v", err)
	}
}

func serveStream(ctx context.Context, session *xmpp.Session, regHandler *registrationHandler, cfg Config, tlsConfig *tls.Config, authenticatedUser *string) error {
	reader := session.Reader()
	writer := session.Writer()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		tok, err := reader.Token()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		start, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}

		if start.Name.Space == ns.Stream && start.Name.Local == "stream" {
			if err := writeStreamStart(writer, cfg.Domain); err != nil {
				return err
			}
			if err := writeStreamFeatures(writer, cfg, session.State(), tlsConfig); err != nil {
				return err
			}
			continue
		}

		switch {
		case start.Name.Space == ns.TLS && start.Name.Local == "starttls":
			if err := handleStartTLS(ctx, session, tlsConfig, reader); err != nil {
				return err
			}
		case start.Name.Space == ns.SASL && start.Name.Local == "auth":
			if err := handleSASLAuth(ctx, session, storeUserStore(regHandler), cfg, authenticatedUser, reader, &start); err != nil {
				return err
			}
		case start.Name.Local == "message":
			if err := handleMessage(ctx, session, reader, &start); err != nil {
				return err
			}
		case start.Name.Local == "presence":
			if err := handlePresence(ctx, session, reader, &start); err != nil {
				return err
			}
		case start.Name.Local == "iq":
			if err := handleIQ(ctx, session, regHandler, cfg, authenticatedUser, reader, &start); err != nil {
				return err
			}
		default:
			if err := reader.Skip(); err != nil {
				return err
			}
		}
	}
}

func storeUserStore(regHandler *registrationHandler) storage.UserStore {
	if regHandler == nil || regHandler.store == nil {
		return nil
	}
	return regHandler.store.UserStore()
}

func handleStartTLS(ctx context.Context, session *xmpp.Session, tlsConfig *tls.Config, reader *xmppxml.StreamReader) error {
	if err := reader.Skip(); err != nil {
		return err
	}
	if tlsConfig == nil || session.State()&xmpp.StateSecure != 0 {
		return session.SendElement(ctx, startTLSFailure{})
	}
	if err := session.SendElement(ctx, startTLSProceed{}); err != nil {
		return err
	}
	if err := session.Transport().StartTLS(tlsConfig); err != nil {
		return err
	}
	session.SetState(xmpp.StateSecure)
	return nil
}

func handleSASLAuth(ctx context.Context, session *xmpp.Session, userStore storage.UserStore, cfg Config, authenticatedUser *string, reader *xmppxml.StreamReader, start *xml.StartElement) error {
	if session.State()&xmpp.StateAuthenticated != 0 {
		if err := reader.Skip(); err != nil {
			return err
		}
		return sendSASLFailure(ctx, session, "not-authorized")
	}

	var auth saslAuth
	if err := reader.DecodeElement(&auth, start); err != nil {
		return err
	}

	if strings.ToUpper(strings.TrimSpace(auth.Mechanism)) != "PLAIN" {
		return sendSASLFailure(ctx, session, "invalid-mechanism")
	}

	payload, err := base64.StdEncoding.DecodeString(strings.TrimSpace(auth.Value))
	if err != nil {
		return sendSASLFailure(ctx, session, "malformed-request")
	}
	parts := strings.SplitN(string(payload), "\x00", 3)
	if len(parts) != 3 || strings.TrimSpace(parts[1]) == "" {
		return sendSASLFailure(ctx, session, "malformed-request")
	}

	username := strings.TrimSpace(parts[1])
	password := parts[2]
	if userStore == nil {
		return sendSASLFailure(ctx, session, "temporary-auth-failure")
	}

	ok, err := userStore.Authenticate(ctx, username, password)
	if err != nil {
		log.Printf("auth lookup failed for %s: %v", username, err)
		return sendSASLFailure(ctx, session, "temporary-auth-failure")
	}
	if !ok {
		return sendSASLFailure(ctx, session, "not-authorized")
	}

	j, err := jid.New(username, cfg.Domain, "")
	if err != nil {
		return sendSASLFailure(ctx, session, "not-authorized")
	}
	*authenticatedUser = username
	session.SetRemoteAddr(j)
	session.SetState(xmpp.StateAuthenticated)
	return session.SendElement(ctx, saslSuccess{})
}

func handleIQ(ctx context.Context, session *xmpp.Session, regHandler *registrationHandler, cfg Config, authenticatedUser *string, reader *xmppxml.StreamReader, start *xml.StartElement) error {
	var iq stanza.IQ
	if err := reader.DecodeElement(&iq, start); err != nil {
		return err
	}

	if isBindRequestIQ(&iq) {
		return handleBindIQ(ctx, session, cfg, authenticatedUser, &iq)
	}

	if err := regHandler.Handle(ctx, session, &iq); err != nil {
		return err
	}

	if session.State()&xmpp.StateReady == 0 {
		if iq.Type == stanza.IQGet || iq.Type == stanza.IQSet {
			return session.Send(ctx, iq.ErrorIQ(stanza.NewStanzaError(stanza.ErrorTypeAuth, stanza.ErrorNotAuthorized, "authenticate and bind first")))
		}
		return nil
	}

	return routeIQ(ctx, session, &iq)
}

func isBindRequestIQ(iq *stanza.IQ) bool {
	if iq == nil || iq.Type != stanza.IQSet || len(iq.Query) == 0 {
		return false
	}
	var req xmpp.BindRequest
	if err := xml.Unmarshal(iq.Query, &req); err != nil {
		return false
	}
	return req.XMLName.Space == ns.Bind && req.XMLName.Local == "bind"
}

func handleBindIQ(ctx context.Context, session *xmpp.Session, cfg Config, authenticatedUser *string, iq *stanza.IQ) error {
	if session.State()&xmpp.StateAuthenticated == 0 {
		return session.Send(ctx, iq.ErrorIQ(stanza.NewStanzaError(stanza.ErrorTypeAuth, stanza.ErrorNotAuthorized, "not authenticated")))
	}

	username := strings.TrimSpace(*authenticatedUser)
	if username == "" {
		username = strings.TrimSpace(session.RemoteAddr().Local())
	}
	if username == "" {
		return session.Send(ctx, iq.ErrorIQ(stanza.NewStanzaError(stanza.ErrorTypeAuth, stanza.ErrorNotAuthorized, "not authenticated")))
	}

	var bindReq xmpp.BindRequest
	if err := xml.Unmarshal(iq.Query, &bindReq); err != nil {
		return session.Send(ctx, iq.ErrorIQ(stanza.NewStanzaError(stanza.ErrorTypeModify, stanza.ErrorBadRequest, "invalid bind payload")))
	}

	resource := strings.TrimSpace(bindReq.Resource)
	if resource == "" {
		resource = randomResource()
	}

	full, err := jid.New(username, cfg.Domain, resource)
	if err != nil {
		return session.Send(ctx, iq.ErrorIQ(stanza.NewStanzaError(stanza.ErrorTypeModify, stanza.ErrorJIDMalformed, "invalid jid")))
	}

	session.SetRemoteAddr(full)
	session.SetState(xmpp.StateBound | xmpp.StateReady)
	globalRouter.register(full, session)

	result := iq.ResultIQ()
	payload := &stanza.IQPayload{
		IQ:      *result,
		Payload: &xmpp.BindResult{JID: full.String()},
	}
	return session.SendElement(ctx, payload)
}

func handleMessage(ctx context.Context, session *xmpp.Session, reader *xmppxml.StreamReader, start *xml.StartElement) error {
	var msg stanza.Message
	if err := reader.DecodeElement(&msg, start); err != nil {
		return err
	}
	if session.State()&xmpp.StateReady == 0 {
		return nil
	}
	return routeMessage(ctx, session, &msg)
}

func handlePresence(ctx context.Context, session *xmpp.Session, reader *xmppxml.StreamReader, start *xml.StartElement) error {
	var pres stanza.Presence
	if err := reader.DecodeElement(&pres, start); err != nil {
		return err
	}
	if session.State()&xmpp.StateReady == 0 {
		return nil
	}
	return routePresence(ctx, session, &pres)
}

func routeMessage(ctx context.Context, source *xmpp.Session, msg *stanza.Message) error {
	if msg.From.IsZero() {
		msg.From = source.RemoteAddr()
	}
	targets := globalRouter.targets(msg.To)
	for _, dst := range targets {
		if dst == source {
			continue
		}
		if err := dst.Send(ctx, msg); err != nil {
			log.Printf("message route error to %s: %v", dst.RemoteAddr(), err)
		}
	}
	return nil
}

func routePresence(ctx context.Context, source *xmpp.Session, pres *stanza.Presence) error {
	if pres.From.IsZero() {
		pres.From = source.RemoteAddr()
	}
	if pres.To.IsZero() {
		return nil
	}
	targets := globalRouter.targets(pres.To)
	for _, dst := range targets {
		if dst == source {
			continue
		}
		if err := dst.Send(ctx, pres); err != nil {
			log.Printf("presence route error to %s: %v", dst.RemoteAddr(), err)
		}
	}
	return nil
}

func routeIQ(ctx context.Context, source *xmpp.Session, iq *stanza.IQ) error {
	if iq.To.IsZero() || iq.To.IsDomainOnly() {
		if iq.Type == stanza.IQGet || iq.Type == stanza.IQSet {
			return source.Send(ctx, iq.ErrorIQ(stanza.NewStanzaError(stanza.ErrorTypeCancel, stanza.ErrorServiceUnavailable, "unsupported server iq")))
		}
		return nil
	}
	if iq.From.IsZero() {
		iq.From = source.RemoteAddr()
	}

	targets := globalRouter.targets(iq.To)
	if len(targets) == 0 {
		if iq.Type == stanza.IQGet || iq.Type == stanza.IQSet {
			return source.Send(ctx, iq.ErrorIQ(stanza.NewStanzaError(stanza.ErrorTypeCancel, stanza.ErrorItemNotFound, "recipient not found")))
		}
		return nil
	}

	for _, dst := range targets {
		if dst == source {
			continue
		}
		if err := dst.Send(ctx, iq); err != nil {
			log.Printf("iq route error to %s: %v", dst.RemoteAddr(), err)
		}
		if iq.To.IsFull() {
			break
		}
	}
	return nil
}

func sendSASLFailure(ctx context.Context, session *xmpp.Session, condition string) error {
	xmlPayload := "<failure xmlns='" + ns.SASL + "'><" + condition + "/></failure>"
	return session.SendRaw(ctx, strings.NewReader(xmlPayload))
}

func randomStreamID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "stream-" + randomResource()
	}
	return hex.EncodeToString(b)
}

func randomResource() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "roster"
	}
	return "roster-" + hex.EncodeToString(b)
}

func buildTLSConfig(cfg Config) (*tls.Config, error) {
	if cfg.TLSCert == "" || cfg.TLSKey == "" {
		return nil, nil
	}
	cert, err := tls.LoadX509KeyPair(cfg.TLSCert, cfg.TLSKey)
	if err != nil {
		return nil, err
	}
	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}, nil
}

func writeStreamStart(writer *xmppxml.StreamWriter, domain string) error {
	from, err := jid.New("", domain, "")
	if err != nil {
		return err
	}

	header := stream.Open(stream.Header{
		From:    from,
		ID:      randomStreamID(),
		Lang:    "en",
		Version: stream.DefaultVersion,
		NS:      ns.Client,
	})
	_, err = writer.WriteRaw(header)
	return err
}

func writeStreamFeatures(writer *xmppxml.StreamWriter, cfg Config, state xmpp.SessionState, tlsConfig *tls.Config) error {
	start := xml.StartElement{Name: xml.Name{Space: ns.Stream, Local: "features"}}
	if err := writer.EncodeToken(start); err != nil {
		return err
	}

	secure := state&xmpp.StateSecure != 0
	authenticated := state&xmpp.StateAuthenticated != 0
	bound := state&xmpp.StateBound != 0

	if !secure && tlsConfig != nil {
		if err := writeStartTLSFeature(writer); err != nil {
			return err
		}
		if cfg.Registration.Policy != registrationClosed {
			if err := writeRegistrationFeature(writer); err != nil {
				return err
			}
		}
		return writer.EncodeToken(xml.EndElement{Name: start.Name})
	}

	if !authenticated {
		if err := writeSASLMechanisms(writer, []string{"PLAIN"}); err != nil {
			return err
		}
		if cfg.Registration.Policy != registrationClosed {
			if err := writeRegistrationFeature(writer); err != nil {
				return err
			}
		}
		return writer.EncodeToken(xml.EndElement{Name: start.Name})
	}

	if !bound {
		if err := writeBindFeature(writer); err != nil {
			return err
		}
	}

	return writer.EncodeToken(xml.EndElement{Name: start.Name})
}

func writeStartTLSFeature(writer *xmppxml.StreamWriter) error {
	feature := xml.StartElement{Name: xml.Name{Space: ns.TLS, Local: "starttls"}}
	if err := writer.EncodeToken(feature); err != nil {
		return err
	}
	required := xml.StartElement{Name: xml.Name{Local: "required"}}
	if err := writer.EncodeToken(required); err != nil {
		return err
	}
	if err := writer.EncodeToken(xml.EndElement{Name: required.Name}); err != nil {
		return err
	}
	return writer.EncodeToken(xml.EndElement{Name: feature.Name})
}

func writeSASLMechanisms(writer *xmppxml.StreamWriter, mechanisms []string) error {
	mechs := xml.StartElement{Name: xml.Name{Space: ns.SASL, Local: "mechanisms"}}
	if err := writer.EncodeToken(mechs); err != nil {
		return err
	}
	for _, mechanism := range mechanisms {
		mech := xml.StartElement{Name: xml.Name{Space: ns.SASL, Local: "mechanism"}}
		if err := writer.EncodeToken(mech); err != nil {
			return err
		}
		if err := writer.EncodeToken(xml.CharData(mechanism)); err != nil {
			return err
		}
		if err := writer.EncodeToken(xml.EndElement{Name: mech.Name}); err != nil {
			return err
		}
	}
	return writer.EncodeToken(xml.EndElement{Name: mechs.Name})
}

func writeBindFeature(writer *xmppxml.StreamWriter) error {
	feature := xml.StartElement{Name: xml.Name{Space: ns.Bind, Local: "bind"}}
	if err := writer.EncodeToken(feature); err != nil {
		return err
	}
	return writer.EncodeToken(xml.EndElement{Name: feature.Name})
}

func writeRegistrationFeature(writer *xmppxml.StreamWriter) error {
	feature := xml.StartElement{Name: xml.Name{Space: ns.Register, Local: "register"}}
	if err := writer.EncodeToken(feature); err != nil {
		return err
	}
	return writer.EncodeToken(xml.EndElement{Name: feature.Name})
}
