package main

import (
	"context"
	"encoding/xml"
	"errors"
	"net"
	"strings"
	"sync"
	"time"

	xmpp "github.com/meszmate/xmpp-go"
	"github.com/meszmate/xmpp-go/internal/ns"
	"github.com/meszmate/xmpp-go/plugins/form"
	"github.com/meszmate/xmpp-go/plugins/register"
	"github.com/meszmate/xmpp-go/stanza"
	"github.com/meszmate/xmpp-go/storage"
)

type registrationPolicy string

const (
	registrationOpen   registrationPolicy = "open"
	registrationClosed registrationPolicy = "closed"
	registrationInvite registrationPolicy = "invite"
	registrationAdmin  registrationPolicy = "admin"
)

type registrationConfig struct {
	Policy       registrationPolicy
	Fields       []string
	Invites      map[string]struct{}
	AdminTokens  map[string]struct{}
	RateLimit    int
	RateWindow   time.Duration
	Iterations   int
	DataForm     bool
	Instructions string
}

type rateLimiter struct {
	mu     sync.Mutex
	window time.Duration
	limit  int
	items  map[string][]time.Time
}

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	return &rateLimiter{
		window: window,
		limit:  limit,
		items:  make(map[string][]time.Time),
	}
}

func (r *rateLimiter) Allow(key string) bool {
	if r == nil || r.limit <= 0 {
		return true
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	cutoff := time.Now().Add(-r.window)
	entries := r.items[key]
	out := entries[:0]
	for _, t := range entries {
		if t.After(cutoff) {
			out = append(out, t)
		}
	}
	if len(out) >= r.limit {
		r.items[key] = out
		return false
	}
	out = append(out, time.Now())
	r.items[key] = out
	return true
}

type registrationHandler struct {
	cfg         registrationConfig
	store       storage.Storage
	rateLimiter *rateLimiter
}

func newRegistrationHandler(cfg registrationConfig, store storage.Storage) *registrationHandler {
	return &registrationHandler{
		cfg:         cfg,
		store:       store,
		rateLimiter: newRateLimiter(cfg.RateLimit, cfg.RateWindow),
	}
}

func (h *registrationHandler) Handle(ctx context.Context, session *xmpp.Session, st stanza.Stanza) error {
	iq, ok := st.(*stanza.IQ)
	if !ok {
		return nil
	}
	if iq.Type != stanza.IQGet && iq.Type != stanza.IQSet {
		return nil
	}

	var q register.Query
	if len(iq.Query) == 0 {
		return nil
	}
	if err := xml.NewDecoder(strings.NewReader(string(iq.Query))).Decode(&q); err != nil {
		return nil
	}
	if q.XMLName.Space != ns.Register {
		return nil
	}

	peer := peerKey(session.Transport().Peer())
	if !h.rateLimiter.Allow(peer) {
		return session.Send(ctx, iq.ErrorIQ(stanza.NewStanzaError(stanza.ErrorTypeWait, stanza.ErrorResourceConstraint, "rate limit exceeded")))
	}

	switch iq.Type {
	case stanza.IQGet:
		return h.handleGet(ctx, session, iq)
	case stanza.IQSet:
		return h.handleSet(ctx, session, iq, q)
	default:
		return nil
	}
}

func (h *registrationHandler) handleGet(ctx context.Context, session *xmpp.Session, iq *stanza.IQ) error {
	if h.cfg.Policy == registrationClosed {
		return session.Send(ctx, iq.ErrorIQ(stanza.NewStanzaError(stanza.ErrorTypeCancel, stanza.ErrorServiceUnavailable, "registration disabled")))
	}
	form := h.buildForm()
	payload := &stanza.IQPayload{IQ: *iq.ResultIQ(), Payload: form}
	return session.SendElement(ctx, payload)
}

func (h *registrationHandler) handleSet(ctx context.Context, session *xmpp.Session, iq *stanza.IQ, q register.Query) error {
	if h.cfg.Policy == registrationClosed {
		return session.Send(ctx, iq.ErrorIQ(stanza.NewStanzaError(stanza.ErrorTypeCancel, stanza.ErrorServiceUnavailable, "registration disabled")))
	}
	if h.store == nil || h.store.UserStore() == nil {
		return session.Send(ctx, iq.ErrorIQ(stanza.NewStanzaError(stanza.ErrorTypeCancel, stanza.ErrorServiceUnavailable, "registration unavailable")))
	}

	fields := extractFields(q)
	if h.cfg.Policy == registrationInvite {
		token := fields["invite"]
		if token == "" {
			token = fields["token"]
		}
		if !h.isTokenAllowed(h.cfg.Invites, token) {
			return session.Send(ctx, iq.ErrorIQ(stanza.NewStanzaError(stanza.ErrorTypeAuth, stanza.ErrorNotAllowed, "invalid invite")))
		}
	}
	if h.cfg.Policy == registrationAdmin {
		token := fields["admin_token"]
		if !h.isTokenAllowed(h.cfg.AdminTokens, token) {
			return session.Send(ctx, iq.ErrorIQ(stanza.NewStanzaError(stanza.ErrorTypeAuth, stanza.ErrorNotAllowed, "admin token required")))
		}
	}

	if q.Remove != nil {
		return h.handleRemove(ctx, session, iq, fields)
	}

	username := fields["username"]
	password := fields["password"]
	if username == "" || password == "" {
		return session.Send(ctx, iq.ErrorIQ(stanza.NewStanzaError(stanza.ErrorTypeModify, stanza.ErrorBadRequest, "username and password required")))
	}

	us := h.store.UserStore()
	if exists, err := us.UserExists(ctx, username); err != nil {
		return session.Send(ctx, iq.ErrorIQ(stanza.NewStanzaError(stanza.ErrorTypeWait, stanza.ErrorInternalServerError, "user lookup failed")))
	} else if exists {
		return session.Send(ctx, iq.ErrorIQ(stanza.NewStanzaError(stanza.ErrorTypeCancel, stanza.ErrorConflict, "user already exists")))
	}

	salt, iters, storedKey, serverKey, err := hashPasswordSCRAMSHA256(password, h.cfg.Iterations)
	if err != nil {
		return session.Send(ctx, iq.ErrorIQ(stanza.NewStanzaError(stanza.ErrorTypeWait, stanza.ErrorInternalServerError, "password hashing failed")))
	}

	user := &storage.User{
		Username: username,
		// Keep plaintext populated for backends that still use UserStore.Authenticate.
		Password:   password,
		Salt:       salt,
		Iterations: iters,
		StoredKey:  storedKey,
		ServerKey:  serverKey,
	}
	if err := us.CreateUser(ctx, user); err != nil {
		if errors.Is(err, storage.ErrUserExists) {
			return session.Send(ctx, iq.ErrorIQ(stanza.NewStanzaError(stanza.ErrorTypeCancel, stanza.ErrorConflict, "user already exists")))
		}
		return session.Send(ctx, iq.ErrorIQ(stanza.NewStanzaError(stanza.ErrorTypeWait, stanza.ErrorInternalServerError, "user create failed")))
	}

	resp := iq.ResultIQ()
	payload := &stanza.IQPayload{IQ: *resp, Payload: &register.Query{Registered: &register.Empty{}}}
	return session.SendElement(ctx, payload)
}

func (h *registrationHandler) handleRemove(ctx context.Context, session *xmpp.Session, iq *stanza.IQ, fields map[string]string) error {
	username := fields["username"]
	password := fields["password"]
	if username == "" || password == "" {
		return session.Send(ctx, iq.ErrorIQ(stanza.NewStanzaError(stanza.ErrorTypeModify, stanza.ErrorBadRequest, "username and password required")))
	}
	us := h.store.UserStore()
	ok, err := us.Authenticate(ctx, username, password)
	if err != nil || !ok {
		return session.Send(ctx, iq.ErrorIQ(stanza.NewStanzaError(stanza.ErrorTypeAuth, stanza.ErrorNotAuthorized, "authentication failed")))
	}
	if err := us.DeleteUser(ctx, username); err != nil {
		return session.Send(ctx, iq.ErrorIQ(stanza.NewStanzaError(stanza.ErrorTypeWait, stanza.ErrorInternalServerError, "user delete failed")))
	}
	resp := iq.ResultIQ()
	return session.SendElement(ctx, &stanza.IQPayload{IQ: *resp, Payload: &register.Query{Registered: &register.Empty{}}})
}

func (h *registrationHandler) buildForm() *register.Query {
	query := &register.Query{
		Instructions: h.cfg.Instructions,
	}

	if h.cfg.DataForm {
		dataForm := &form.Form{
			Type:  form.TypeForm,
			Title: "Account Registration",
		}
		for _, field := range h.cfg.Fields {
			if field == "" {
				continue
			}
			label := strings.ToUpper(field[:1]) + field[1:]
			dataForm.Fields = append(dataForm.Fields, form.Field{
				Var:   field,
				Type:  fieldType(field),
				Label: label,
			})
		}
		if h.cfg.Policy == registrationInvite {
			dataForm.Fields = append(dataForm.Fields, form.Field{Var: "invite", Type: form.FieldTextSingle, Label: "Invite Token"})
		}
		if h.cfg.Policy == registrationAdmin {
			dataForm.Fields = append(dataForm.Fields, form.Field{Var: "admin_token", Type: form.FieldTextPrivate, Label: "Admin Token"})
		}
		query.Form = mustMarshal(dataForm)
	}

	for _, field := range h.cfg.Fields {
		switch field {
		case "username":
			query.Username = ""
		case "password":
			query.Password = ""
		case "email":
			query.Email = ""
		}
	}

	return query
}

func fieldType(name string) string {
	switch name {
	case "password":
		return "text-private"
	default:
		return "text-single"
	}
}

func extractFields(q register.Query) map[string]string {
	fields := map[string]string{}
	if q.Username != "" {
		fields["username"] = q.Username
	}
	if q.Password != "" {
		fields["password"] = q.Password
	}
	if q.Email != "" {
		fields["email"] = q.Email
	}
	if q.Form != nil {
		var dataForm form.Form
		if err := xml.NewDecoder(strings.NewReader(string(q.Form))).Decode(&dataForm); err == nil {
			for _, f := range dataForm.Fields {
				if len(f.Values) > 0 {
					fields[f.Var] = f.Values[0]
				}
			}
		}
	}
	return fields
}

func mustMarshal(v any) []byte {
	b, _ := xml.Marshal(v)
	return b
}

func (h *registrationHandler) isTokenAllowed(tokens map[string]struct{}, token string) bool {
	if len(tokens) == 0 {
		return false
	}
	_, ok := tokens[token]
	return ok
}

func peerKey(addr net.Addr) string {
	if addr == nil {
		return "unknown"
	}
	s := addr.String()
	if host, _, err := net.SplitHostPort(s); err == nil {
		return host
	}
	return s
}
