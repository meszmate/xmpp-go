package transport

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/meszmate/xmpp-go/internal/bosh"
)

// BOSHClient is a working client-side XMPP-over-BOSH (XEP-0124/0206) transport.
// It presents the streaming Transport interface to the negotiation layer while
// tunneling the XMPP stream over HTTP <body/> long-polling: outbound stream
// elements are wrapped into <body> POSTs, and response bodies are unwrapped into
// a synthesized stream the reader consumes.
type BOSHClient struct {
	url    string
	to     string
	client *http.Client

	rid atomic.Uint64
	sid string

	// Outbound: the negotiation layer writes into writePipe; parseLoop splits the
	// stream into events delivered on outCh.
	writePipe *io.PipeWriter
	outCh     chan bosh.Event

	// Inbound: response payloads are appended to inBuf for Read.
	inMu     sync.Mutex
	inBuf    bytes.Buffer
	inSignal chan struct{}

	// feedResponse ordering: responses may complete out of rid order (a data
	// request can return before a concurrent long-poll), so they are reassembled
	// into the reader's byte stream in strict rid order.
	feedMu      sync.Mutex
	expectFeed  uint64
	pendingFeed map[uint64][]byte
	pollStarted atomic.Bool

	reqCtx    context.Context
	reqCancel context.CancelFunc
	closeOnce sync.Once
	closed    chan struct{}
	errMu     sync.Mutex
	err       error
}

// DialBOSHClient creates a BOSH transport targeting the given connection-manager
// URL for the XMPP domain. No HTTP request is made until stream negotiation
// begins writing.
func DialBOSHClient(url, domain string) (*BOSHClient, error) {
	pr, pw := io.Pipe()
	b := &BOSHClient{
		url:         url,
		to:          domain,
		client:      &http.Client{Timeout: 70 * time.Second},
		writePipe:   pw,
		outCh:       make(chan bosh.Event, 16),
		inSignal:    make(chan struct{}, 1),
		pendingFeed: make(map[uint64][]byte),
		closed:      make(chan struct{}),
	}
	b.rid.Store(randomRID())
	b.reqCtx, b.reqCancel = context.WithCancel(context.Background())
	go b.parseLoop(pr)
	go b.pumpLoop()
	return b, nil
}

func randomRID() uint64 {
	n, err := rand.Int(rand.Reader, big.NewInt(1<<31))
	if err != nil {
		return 1
	}
	return n.Uint64() + 1
}

// parseLoop splits the outbound XMPP byte stream into events for the pump.
func (b *BOSHClient) parseLoop(r io.Reader) {
	parser := bosh.NewParser(r)
	for {
		ev, err := parser.Next()
		if err != nil {
			return
		}
		select {
		case b.outCh <- ev:
		case <-b.closed:
			return
		}
	}
}

// pumpLoop dispatches outbound stream events (create/restart/data/terminate).
// Inbound long-polling runs concurrently in pollLoop once a session exists, so a
// send is never stalled behind an idle poll (the server frees held polls when a
// new request arrives — XEP-0124 hold/requests semantics).
func (b *BOSHClient) pumpLoop() {
	for {
		select {
		case ev := <-b.outCh:
			b.handleEvent(ev)
		case <-b.closed:
			return
		}
	}
}

func (b *BOSHClient) handleEvent(ev bosh.Event) {
	switch ev.Kind {
	case bosh.Open:
		if b.sid == "" {
			b.create()
		} else {
			b.restart()
		}
	case bosh.Element:
		b.sendData(ev.Raw)
	case bosh.Close:
		b.terminate()
	}
}

func (b *BOSHClient) nextRID() uint64 { return b.rid.Add(1) }

// post performs one BOSH HTTP round-trip, retransmitting the identical request
// (same rid) on transport failure. Because the connection manager caches
// responses by rid (§14.2), a retransmission returns the original response
// rather than reprocessing the request.
func (b *BOSHClient) post(attrs map[string]string, payload []byte) (*bosh.Body, error) {
	reqBody := bosh.BuildBody(attrs, payload)
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		select {
		case <-b.closed:
			return nil, errors.New("transport: BOSH connection closed")
		default:
		}
		req, err := http.NewRequestWithContext(b.reqCtx, http.MethodPost, b.url, bytes.NewReader(reqBody))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "text/xml; charset=utf-8")
		resp, err := b.client.Do(req)
		if err != nil {
			lastErr = err
			time.Sleep(time.Duration(attempt+1) * 100 * time.Millisecond)
			continue
		}
		data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		_ = resp.Body.Close()
		if err != nil {
			lastErr = err
			continue
		}
		return bosh.ParseBody(data)
	}
	return nil, lastErr
}

func (b *BOSHClient) create() {
	rid := b.nextRID()
	attrs := map[string]string{
		"rid":                        itoa(rid),
		"to":                         b.to,
		"ver":                        "1.6",
		"wait":                       "60",
		"hold":                       "1",
		"requests":                   "2",
		"content":                    "text/xml; charset=utf-8",
		bosh.XMPPPrefix + ":version": "1.0",
	}
	body, err := b.post(attrs, nil)
	if err != nil {
		b.fail(err)
		return
	}
	b.sid = body.Attrs["sid"]
	if b.sid == "" {
		b.fail(errors.New("bosh: server did not return a sid"))
		return
	}
	from := body.Attrs["from"]
	if from == "" {
		from = b.to
	}
	b.feedMu.Lock()
	b.expectFeed = rid
	b.feedMu.Unlock()
	// Synthesize the responding stream header so the reader sees a normal stream.
	b.feedResponse(rid, append(bosh.ServerHeader(from, b.sid), body.Payload...))
	b.startPolling()
}

func (b *BOSHClient) restart() {
	rid := b.nextRID()
	attrs := map[string]string{
		"rid":                        itoa(rid),
		"sid":                        b.sid,
		"to":                         b.to,
		bosh.XMPPPrefix + ":restart": "true",
	}
	body, err := b.post(attrs, nil)
	if err != nil {
		b.fail(err)
		return
	}
	from := body.Attrs["from"]
	if from == "" {
		from = b.to
	}
	b.feedResponse(rid, append(bosh.ServerHeader(from, b.sid), body.Payload...))
}

func (b *BOSHClient) sendData(raw []byte) {
	rid := b.nextRID()
	body, err := b.post(map[string]string{"rid": itoa(rid), "sid": b.sid}, raw)
	if err != nil {
		b.fail(err)
		return
	}
	b.feedResponse(rid, body.Payload)
	if body.Attrs["type"] == "terminate" {
		b.Close()
	}
}

// startPolling launches the single background long-poll loop (once).
func (b *BOSHClient) startPolling() {
	if b.pollStarted.CompareAndSwap(false, true) {
		go b.pollLoop()
	}
}

// pollLoop keeps one long-poll request outstanding to receive server-pushed
// stanzas, concurrent with data sends from pumpLoop.
func (b *BOSHClient) pollLoop() {
	for {
		select {
		case <-b.closed:
			return
		default:
		}
		rid := b.nextRID()
		body, err := b.post(map[string]string{"rid": itoa(rid), "sid": b.sid}, nil)
		if err != nil {
			b.fail(err)
			return
		}
		b.feedResponse(rid, body.Payload)
		if body.Attrs["type"] == "terminate" {
			b.Close()
			return
		}
	}
}

func (b *BOSHClient) terminate() {
	if b.sid == "" {
		return
	}
	_, _ = b.post(map[string]string{"rid": itoa(b.nextRID()), "sid": b.sid, "type": "terminate"}, nil)
	b.Close()
}

// feedResponse delivers a request's response payload into the reader's byte
// stream in strict rid order, buffering any that arrive ahead of their
// predecessors.
func (b *BOSHClient) feedResponse(rid uint64, payload []byte) {
	b.feedMu.Lock()
	if b.expectFeed == 0 {
		b.expectFeed = rid
	}
	b.pendingFeed[rid] = payload
	for {
		p, ok := b.pendingFeed[b.expectFeed]
		if !ok {
			break
		}
		delete(b.pendingFeed, b.expectFeed)
		b.expectFeed++
		b.feedMu.Unlock()
		b.feedIn(p)
		b.feedMu.Lock()
	}
	b.feedMu.Unlock()
}

func itoa(n uint64) string { return fmt.Sprintf("%d", n) }

// feedIn appends inbound payload bytes and wakes any blocked Read.
func (b *BOSHClient) feedIn(payload []byte) {
	if len(payload) == 0 {
		return
	}
	b.inMu.Lock()
	b.inBuf.Write(payload)
	b.inMu.Unlock()
	select {
	case b.inSignal <- struct{}{}:
	default:
	}
}

func (b *BOSHClient) fail(err error) {
	b.errMu.Lock()
	if b.err == nil {
		b.err = err
	}
	b.errMu.Unlock()
	b.Close()
}

// Read returns tunneled stream bytes, blocking until data is available or the
// transport is closed.
func (b *BOSHClient) Read(p []byte) (int, error) {
	for {
		b.inMu.Lock()
		if b.inBuf.Len() > 0 {
			n, _ := b.inBuf.Read(p)
			b.inMu.Unlock()
			return n, nil
		}
		b.inMu.Unlock()

		select {
		case <-b.inSignal:
		case <-b.closed:
			b.inMu.Lock()
			if b.inBuf.Len() > 0 {
				n, _ := b.inBuf.Read(p)
				b.inMu.Unlock()
				return n, nil
			}
			b.inMu.Unlock()
			if err := b.readErr(); err != nil {
				return 0, err
			}
			return 0, io.EOF
		}
	}
}

// Write hands outbound stream bytes to the reframer.
func (b *BOSHClient) Write(p []byte) (int, error) {
	select {
	case <-b.closed:
		return 0, errors.New("transport: BOSH connection closed")
	default:
	}
	return b.writePipe.Write(p)
}

func (b *BOSHClient) readErr() error {
	b.errMu.Lock()
	defer b.errMu.Unlock()
	return b.err
}

// Close terminates the BOSH session and releases resources.
func (b *BOSHClient) Close() error {
	b.closeOnce.Do(func() {
		close(b.closed)
		if b.reqCancel != nil {
			b.reqCancel()
		}
		_ = b.writePipe.Close()
	})
	return nil
}

// StartTLS is not supported: BOSH runs over HTTPS at the transport URL.
func (b *BOSHClient) StartTLS(_ *tls.Config) error {
	return errors.New("transport: BOSH does not support STARTTLS; use an https:// endpoint")
}

// ConnectionState reports whether the BOSH endpoint is secured (HTTPS). BOSH
// security is provided by the HTTP/TLS layer rather than in-stream STARTTLS, so
// the returned tls.ConnectionState is empty; only the secure flag is meaningful.
func (b *BOSHClient) ConnectionState() (tls.ConnectionState, bool) {
	return tls.ConnectionState{}, b.Secure()
}

// Secure reports whether the BOSH endpoint uses HTTPS.
func (b *BOSHClient) Secure() bool {
	return strings.HasPrefix(strings.ToLower(b.url), "https")
}

// Peer returns nil; BOSH has no single persistent peer connection.
func (b *BOSHClient) Peer() net.Addr { return nil }

// LocalAddress returns nil for BOSH.
func (b *BOSHClient) LocalAddress() net.Addr { return nil }
