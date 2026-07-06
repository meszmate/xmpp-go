package xmpp

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/meszmate/xmpp-go/internal/bosh"
)

// postBoshRaw sends a BOSH body and returns the raw response bytes.
func postBoshRaw(t *testing.T, url string, attrs map[string]string, payload []byte) []byte {
	t.Helper()
	resp, err := http.Post(url, "text/xml", bytes.NewReader(bosh.BuildBody(attrs, payload)))
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	return data
}

func boshCreateSession(t *testing.T, url string) (sid string, nextRid uint64) {
	t.Helper()
	raw := postBoshRaw(t, url, map[string]string{
		"rid": "100", "to": testDomain, "ver": "1.6", "wait": "60", "hold": "1",
	}, nil)
	body, err := bosh.ParseBody(raw)
	if err != nil {
		t.Fatalf("parse create response: %v", err)
	}
	sid = body.Attrs["sid"]
	if sid == "" {
		t.Fatalf("no sid in create response: %s", raw)
	}
	return sid, 101
}

// TestBOSHAckAndRetransmit verifies request acknowledgements (§9) and that a
// retransmitted rid returns the byte-identical cached response (§14.2) without
// reprocessing.
func TestBOSHAckAndRetransmit(t *testing.T) {
	srv, _, store := startServer(t)
	addUser(t, store, "alice", "pw")
	httpSrv := httptest.NewServer(srv.BOSHHandler())
	t.Cleanup(httpSrv.Close)

	sid, rid := boshCreateSession(t, httpSrv.URL)

	// Send an empty request that will hold; release it with a following request.
	first := make(chan []byte, 1)
	go func() {
		first <- postBoshRaw(t, httpSrv.URL, map[string]string{"rid": itoa(rid), "sid": sid}, nil)
	}()
	time.Sleep(100 * time.Millisecond) // let the poll register as held
	// A newer request releases the held one (hold-release).
	_ = postBoshRaw(t, httpSrv.URL, map[string]string{"rid": itoa(rid + 1), "sid": sid}, nil)

	var firstResp []byte
	select {
	case firstResp = <-first:
	case <-time.After(5 * time.Second):
		t.Fatal("held request never returned")
	}

	body, err := bosh.ParseBody(firstResp)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if body.Attrs["ack"] != itoa(rid) {
		t.Errorf("ack = %q, want %d", body.Attrs["ack"], rid)
	}

	// Retransmit the same rid: must return the identical cached bytes.
	retr := postBoshRaw(t, httpSrv.URL, map[string]string{"rid": itoa(rid), "sid": sid}, nil)
	if !bytes.Equal(retr, firstResp) {
		t.Errorf("retransmit response differs from cached original\n first=%s\n retr =%s", firstResp, retr)
	}
}

// TestBOSHPipelineGapHold verifies that a pipelined request arriving before its
// predecessor is held and processed in rid order rather than rejected.
func TestBOSHPipelineGapHold(t *testing.T) {
	srv, _, store := startServer(t)
	addUser(t, store, "alice", "pw")
	httpSrv := httptest.NewServer(srv.BOSHHandler())
	t.Cleanup(httpSrv.Close)

	sid, rid := boshCreateSession(t, httpSrv.URL)

	// Fire rid+1 (out of order) first; it must be held until rid arrives.
	ahead := make(chan []byte, 1)
	go func() {
		ahead <- postBoshRaw(t, httpSrv.URL, map[string]string{"rid": itoa(rid + 1), "sid": sid}, nil)
	}()
	time.Sleep(150 * time.Millisecond)

	// Now send rid; both should complete in order (and release each other).
	inOrder := make(chan []byte, 1)
	go func() {
		inOrder <- postBoshRaw(t, httpSrv.URL, map[string]string{"rid": itoa(rid), "sid": sid}, nil)
	}()
	// A third request releases any lingering held poll.
	time.Sleep(150 * time.Millisecond)
	_ = postBoshRaw(t, httpSrv.URL, map[string]string{"rid": itoa(rid + 2), "sid": sid}, nil)

	for _, ch := range []chan []byte{ahead, inOrder} {
		select {
		case raw := <-ch:
			body, err := bosh.ParseBody(raw)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			if strings.Contains(string(raw), "item-not-found") {
				t.Errorf("pipelined request was rejected instead of held: %s", raw)
			}
			_ = body
		case <-time.After(8 * time.Second):
			t.Fatal("pipelined request did not complete")
		}
	}
}
