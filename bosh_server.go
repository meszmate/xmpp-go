package xmpp

import (
	"context"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/meszmate/xmpp-go/internal/bosh"
	"github.com/meszmate/xmpp-go/transport"
)

// boshCacheSize bounds how many recent responses are cached per session for
// retransmission recovery (XEP-0124 §14.2).
const boshCacheSize = 16

// boshConn is one BOSH (XEP-0124/0206) session on the server: it bridges the
// HTTP <body> request/response protocol to a normal streaming Session running
// the library's standard negotiation and routing over an in-memory pipe.
//
// It supports request acknowledgements (§9), retransmission recovery via a
// per-rid response cache (§14.2), forward-gap holding for out-of-order pipelined
// requests, and hold-release so a newer request frees an older long-polling one
// (honoring the negotiated hold/requests limits).
type boshConn struct {
	sid     string
	bridge  net.Conn
	session *Session

	mu      sync.Mutex
	out     [][]byte
	closed  atomic.Bool
	notify  chan struct{}
	release chan struct{}

	ridMu      sync.Mutex
	ridCond    *sync.Cond
	nextRid    uint64            // next rid expected in sequence
	cache      map[uint64][]byte // rid -> full response body (for retransmission)
	cacheOrder []uint64
}

func newBoshConn(sid string, bridge net.Conn, session *Session, firstRid uint64) *boshConn {
	c := &boshConn{
		sid:     sid,
		bridge:  bridge,
		session: session,
		notify:  make(chan struct{}, 1),
		release: make(chan struct{}, 1),
		nextRid: firstRid,
		cache:   make(map[uint64][]byte),
	}
	c.ridCond = sync.NewCond(&c.ridMu)
	return c
}

// collect drains queued outbound elements, waiting up to d for at least one
// (long-polling) when the queue is empty. It returns early if a newer request
// arrives (hold-release). It reports whether the stream closed.
func (c *boshConn) collect(d time.Duration) ([][]byte, bool) {
	timer := time.NewTimer(d)
	defer timer.Stop()
	for {
		c.mu.Lock()
		if len(c.out) > 0 || c.closed.Load() {
			out := c.out
			c.out = nil
			c.mu.Unlock()
			return out, c.closed.Load()
		}
		c.mu.Unlock()
		select {
		case <-c.notify:
			// data may be available; loop to collect it
		case <-c.release:
			return c.drainOut()
		case <-timer.C:
			return c.drainOut()
		}
	}
}

func (c *boshConn) drainOut() ([][]byte, bool) {
	c.mu.Lock()
	out := c.out
	c.out = nil
	c.mu.Unlock()
	return out, c.closed.Load()
}

// markClosed flags the session closed and wakes any waiters (collectors and
// rid-ordering waiters).
func (c *boshConn) markClosed() {
	c.closed.Store(true)
	c.signal()
	c.ridCond.Broadcast()
}

// releaseHeld wakes a currently-held (long-polling) request so a newer request
// does not exceed the concurrent-request limit.
func (c *boshConn) releaseHeld() {
	select {
	case c.release <- struct{}{}:
	default:
	}
}

// cacheResponse stores a response body under its rid for retransmission,
// evicting the oldest when the cache is full.
func (c *boshConn) cacheResponse(rid uint64, body []byte) {
	c.ridMu.Lock()
	defer c.ridMu.Unlock()
	c.cache[rid] = body
	c.cacheOrder = append(c.cacheOrder, rid)
	if len(c.cacheOrder) > boshCacheSize {
		oldest := c.cacheOrder[0]
		c.cacheOrder = c.cacheOrder[1:]
		delete(c.cache, oldest)
	}
}

// drain reads the session's outbound XMPP stream, queuing top-level elements for
// delivery in HTTP responses. Stream headers are absorbed (BOSH conveys stream
// state via <body> attributes).
func (c *boshConn) drain() {
	parser := bosh.NewParser(c.bridge)
	for {
		ev, err := parser.Next()
		if err != nil {
			c.markClosed()
			return
		}
		switch ev.Kind {
		case bosh.Element:
			c.mu.Lock()
			c.out = append(c.out, ev.Raw)
			c.mu.Unlock()
			c.signal()
		case bosh.Close:
			c.markClosed()
			return
		}
	}
}

func (c *boshConn) signal() {
	select {
	case c.notify <- struct{}{}:
	default:
	}
}

func (c *boshConn) feed(payload []byte) error {
	if len(payload) == 0 {
		return nil
	}
	_, err := c.bridge.Write(payload)
	return err
}

// BOSHHandler returns an http.Handler implementing the server side of XMPP over
// BOSH (XEP-0124/0206). Mount it on an HTTP(S) endpoint (typically /http-bind):
//
//	http.Handle("/http-bind", srv.BOSHHandler())
//
// Each BOSH session is bridged to the library's standard negotiation and
// routing, so SASL authentication, resource binding, presence, and message
// delivery all work identically to the TCP and WebSocket transports.
func (s *Server) BOSHHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "BOSH requires POST", http.StatusMethodNotAllowed)
			return
		}
		data, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			http.Error(w, "read body", http.StatusBadRequest)
			return
		}
		body, err := bosh.ParseBody(data)
		if err != nil {
			http.Error(w, "malformed body", http.StatusBadRequest)
			return
		}

		sid := body.Attrs["sid"]
		if sid == "" {
			s.boshCreate(w, body)
			return
		}
		s.boshRequest(w, sid, body)
	})
}

// boshCreate handles a session-creation request: it stands up a bridged Session,
// feeds the initiating stream header, and returns the negotiated features.
func (s *Server) boshCreate(w http.ResponseWriter, body *bosh.Body) {
	to := body.Attrs["to"]
	if to == "" {
		to = s.domain
	}

	serverConn, bridgeConn := net.Pipe()
	trans := transport.NewTCP(serverConn)
	session, err := NewSession(context.Background(), trans,
		WithState(StateServer|StateSecure),
	)
	if err != nil {
		_ = serverConn.Close()
		_ = bridgeConn.Close()
		http.Error(w, "session", http.StatusInternalServerError)
		return
	}

	sid := "bosh-" + randomID()
	conn := newBoshConn(sid, bridgeConn, session, parseBoshRid(body.Attrs["rid"])+1)

	s.mu.Lock()
	s.sessions[sid] = session
	if s.boshConns == nil {
		s.boshConns = make(map[string]*boshConn)
	}
	s.boshConns[sid] = conn
	s.mu.Unlock()

	go func() {
		s.negotiateAndServe(context.Background(), session)
		conn.markClosed()
		s.mu.Lock()
		delete(s.sessions, sid)
		delete(s.boshConns, sid)
		s.mu.Unlock()
	}()
	go conn.drain()

	// Feed the initiating stream header (and any create-time payload) so the
	// session emits its response header and features.
	_ = conn.feed(bosh.ClientHeader(to))
	_ = conn.feed(body.Payload)

	elems, closed := conn.collect(s.boshWait())
	attrs := map[string]string{
		"sid":        sid,
		"from":       s.domain,
		"wait":       "60",
		"requests":   "2",
		"ver":        "1.6",
		"inactivity": "60",
	}
	if closed {
		attrs["type"] = "terminate"
	}
	writeBoshResponse(w, attrs, elems)
}

// boshRequest handles a data/restart/terminate request on an existing session,
// with acknowledgement, retransmission recovery, and pipelined-request ordering.
func (s *Server) boshRequest(w http.ResponseWriter, sid string, body *bosh.Body) {
	s.mu.Lock()
	conn := s.boshConns[sid]
	s.mu.Unlock()
	if conn == nil {
		writeBoshResponse(w, map[string]string{"type": "terminate", "condition": "item-not-found"}, nil)
		return
	}
	rid := parseBoshRid(body.Attrs["rid"])

	conn.ridMu.Lock()
	// Retransmission recovery (§14.2): a request whose rid we already answered
	// returns the cached response rather than being reprocessed.
	if rid != 0 && rid < conn.nextRid {
		cached := conn.cache[rid]
		conn.ridMu.Unlock()
		if cached != nil {
			w.Header().Set("Content-Type", "text/xml; charset=utf-8")
			_, _ = w.Write(cached)
			return
		}
		writeBoshResponse(w, map[string]string{"sid": sid, "ack": itoa(rid)}, nil)
		return
	}
	// Forward gap: a pipelined request that arrived before its predecessor is
	// held until it is this rid's turn (bounded), so requests are processed in
	// rid order.
	if rid != 0 && rid > conn.nextRid {
		if !conn.waitForTurn(rid, s.boshWait()+5*time.Second) {
			conn.ridMu.Unlock()
			writeBoshResponse(w, map[string]string{"sid": sid, "type": "terminate", "condition": "item-not-found"}, nil)
			return
		}
	}

	terminate := body.Attrs["type"] == "terminate"
	switch {
	case terminate:
		_ = conn.feed([]byte("</stream:stream>"))
		_ = conn.bridge.Close()
	case body.Attrs[bosh.XMPPPrefix+":restart"] == "true" || body.Attrs["restart"] == "true":
		_ = conn.feed(bosh.ClientHeader(s.domain))
	default:
		_ = conn.feed(body.Payload)
	}
	if rid != 0 {
		conn.nextRid = rid + 1
		conn.ridCond.Broadcast()
	}
	conn.ridMu.Unlock()

	if terminate {
		writeBoshResponse(w, map[string]string{"sid": sid, "type": "terminate", "ack": itoa(rid)}, nil)
		return
	}

	// A newer request frees an older long-polling one (hold/requests limit).
	conn.releaseHeld()

	elems, closed := conn.collect(s.boshWait())
	attrs := map[string]string{"sid": sid, "from": s.domain}
	if rid != 0 {
		attrs["ack"] = itoa(rid)
	}
	if closed {
		attrs["type"] = "terminate"
	}
	resp := bosh.BuildBody(attrs, joinElems(elems))
	if rid != 0 {
		conn.cacheResponse(rid, resp)
	}
	w.Header().Set("Content-Type", "text/xml; charset=utf-8")
	_, _ = w.Write(resp)
}

// waitForTurn blocks until this request's rid becomes the next expected rid, the
// session closes, or the deadline elapses. Callers hold ridMu; it is held again
// on return. Returns true if it is now this rid's turn.
func (c *boshConn) waitForTurn(rid uint64, timeout time.Duration) bool {
	timedOut := false
	timer := time.AfterFunc(timeout, func() {
		c.ridMu.Lock()
		timedOut = true
		c.ridCond.Broadcast()
		c.ridMu.Unlock()
	})
	defer timer.Stop()
	for rid > c.nextRid && !c.closed.Load() && !timedOut {
		c.ridCond.Wait()
	}
	return rid <= c.nextRid
}

// boshWait bounds how long a request is held when no data is available. Queued
// stanzas are returned immediately (via notify), and a held long-poll is freed
// as soon as a newer request arrives (hold-release), so this only caps the
// truly-idle case.
func (s *Server) boshWait() time.Duration {
	return 20 * time.Second
}

func writeBoshResponse(w http.ResponseWriter, attrs map[string]string, elems [][]byte) {
	w.Header().Set("Content-Type", "text/xml; charset=utf-8")
	_, _ = w.Write(bosh.BuildBody(attrs, joinElems(elems)))
}

func joinElems(elems [][]byte) []byte {
	var payload []byte
	for _, e := range elems {
		payload = append(payload, e...)
	}
	return payload
}

func parseBoshRid(s string) uint64 {
	n, _ := strconv.ParseUint(strings.TrimSpace(s), 10, 64)
	return n
}

func itoa(n uint64) string { return strconv.FormatUint(n, 10) }
