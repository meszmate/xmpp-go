package xmpp

import (
	"context"
	"encoding/xml"
	"errors"
	"io"
	"sync"
	"sync/atomic"

	"github.com/meszmate/xmpp-go/jid"
	"github.com/meszmate/xmpp-go/stanza"
	"github.com/meszmate/xmpp-go/transport"
	xmppxml "github.com/meszmate/xmpp-go/xml"
)

// SessionState represents the state of an XMPP session.
type SessionState uint32

const (
	StateSecure        SessionState = 1 << iota // TLS negotiated
	StateAuthenticated                           // SASL complete
	StateBound                                   // Resource bound
	StateReady                                   // Fully negotiated
	StateServer                                  // Server role
	StateS2S                                     // Server-to-server
)

// Session represents an XMPP session (client or server).
type Session struct {
	state     atomic.Uint32
	mu        sync.Mutex
	trans     transport.Transport
	localJID  jid.JID
	remoteJID jid.JID
	reader    *xmppxml.StreamReader
	writer    *xmppxml.StreamWriter
	mux       *Mux
	closed    chan struct{}
	err       error
}

// NewSession creates a new XMPP session with the given transport and options.
func NewSession(ctx context.Context, trans transport.Transport, opts ...SessionOption) (*Session, error) {
	s := &Session{
		trans:  trans,
		reader: xmppxml.NewStreamReader(trans),
		writer: xmppxml.NewStreamWriter(trans),
		mux:    NewMux(),
		closed: make(chan struct{}),
	}

	for _, opt := range opts {
		opt.apply(s)
	}

	return s, nil
}

// Send sends a stanza through the session.
func (s *Session) Send(ctx context.Context, st stanza.Stanza) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-s.closed:
		return errors.New("xmpp: session closed")
	default:
	}

	return s.writer.Encode(st)
}

// SendRaw writes raw XML to the stream.
func (s *Session) SendRaw(ctx context.Context, r io.Reader) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-s.closed:
		return errors.New("xmpp: session closed")
	default:
	}

	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	_, err = s.writer.WriteRaw(data)
	return err
}

// SendElement encodes an XML element to the stream.
func (s *Session) SendElement(ctx context.Context, v any) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-s.closed:
		return errors.New("xmpp: session closed")
	default:
	}

	return s.writer.Encode(v)
}

// Serve reads stanzas from the stream and dispatches them to the mux.
func (s *Session) Serve(handler Handler) error {
	if handler == nil {
		handler = s.mux
	}
	for {
		select {
		case <-s.closed:
			return s.err
		default:
		}

		tok, err := s.reader.Token()
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

		var st stanza.Stanza
		switch start.Name.Local {
		case "message":
			msg := &stanza.Message{}
			if err := s.reader.DecodeElement(msg, &start); err != nil {
				return err
			}
			st = msg
		case "presence":
			pres := &stanza.Presence{}
			if err := s.reader.DecodeElement(pres, &start); err != nil {
				return err
			}
			st = pres
		case "iq":
			iq := &stanza.IQ{}
			if err := s.reader.DecodeElement(iq, &start); err != nil {
				return err
			}
			st = iq
		default:
			if err := s.reader.Skip(); err != nil {
				return err
			}
			continue
		}

		if err := handler.HandleStanza(context.Background(), s, st); err != nil {
			return err
		}
	}
}

// Close closes the session.
func (s *Session) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	select {
	case <-s.closed:
		return nil
	default:
		close(s.closed)
	}

	return s.trans.Close()
}

// State returns the current session state.
func (s *Session) State() SessionState {
	return SessionState(s.state.Load())
}

// SetState sets session state flags.
func (s *Session) SetState(state SessionState) {
	s.state.Store(uint32(s.State() | state))
}

// LocalAddr returns the local JID.
func (s *Session) LocalAddr() jid.JID {
	return s.localJID
}

// RemoteAddr returns the remote JID.
func (s *Session) RemoteAddr() jid.JID {
	return s.remoteJID
}

// SetLocalAddr sets the local JID.
func (s *Session) SetLocalAddr(j jid.JID) {
	s.localJID = j
}

// SetRemoteAddr sets the remote JID.
func (s *Session) SetRemoteAddr(j jid.JID) {
	s.remoteJID = j
}

// Transport returns the underlying transport.
func (s *Session) Transport() transport.Transport {
	return s.trans
}

// Reader returns the XML stream reader.
func (s *Session) Reader() *xmppxml.StreamReader {
	return s.reader
}

// Writer returns the XML stream writer.
func (s *Session) Writer() *xmppxml.StreamWriter {
	return s.writer
}

// Mux returns the stanza multiplexer.
func (s *Session) Mux() *Mux {
	return s.mux
}
