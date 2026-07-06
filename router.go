package xmpp

import (
	"sync"

	"github.com/meszmate/xmpp-go/jid"
)

// localRouter tracks bound client sessions by JID for local stanza delivery.
// It supports delivery to a specific full JID or fan-out to all resources of a
// bare JID.
type localRouter struct {
	mu     sync.RWMutex
	byFull map[string]*Session            // full JID -> session
	byBare map[string]map[string]*Session // bare JID -> full JID -> session
}

func newLocalRouter() *localRouter {
	return &localRouter{
		byFull: make(map[string]*Session),
		byBare: make(map[string]map[string]*Session),
	}
}

// register records a bound session under its full JID.
func (r *localRouter) register(full jid.JID, s *Session) {
	fullStr := full.String()
	if fullStr == "" {
		return
	}
	bare := full.Bare().String()

	r.mu.Lock()
	defer r.mu.Unlock()
	r.byFull[fullStr] = s
	if r.byBare[bare] == nil {
		r.byBare[bare] = make(map[string]*Session)
	}
	r.byBare[bare][fullStr] = s
}

// unregister removes a session's full JID from the router.
func (r *localRouter) unregister(full jid.JID) {
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

// unregisterIf removes a session's full JID only if the currently-registered
// session is exactly s. This avoids a resume race where an expiring parked
// session would otherwise evict the freshly-resumed session that replaced it.
func (r *localRouter) unregisterIf(full jid.JID, s *Session) {
	fullStr := full.String()
	if fullStr == "" {
		return
	}
	bare := full.Bare().String()

	r.mu.Lock()
	defer r.mu.Unlock()
	if r.byFull[fullStr] != s {
		return
	}
	delete(r.byFull, fullStr)
	if sessions, ok := r.byBare[bare]; ok {
		delete(sessions, fullStr)
		if len(sessions) == 0 {
			delete(r.byBare, bare)
		}
	}
}

// targets returns the sessions a stanza addressed to `to` should be delivered
// to. A full JID resolves to at most one session; a bare JID fans out to every
// connected resource.
func (r *localRouter) targets(to jid.JID) []*Session {
	if to.IsZero() {
		return nil
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	if to.IsFull() {
		if s, ok := r.byFull[to.String()]; ok {
			return []*Session{s}
		}
		return nil
	}

	bare := to.Bare().String()
	sessions := r.byBare[bare]
	if len(sessions) == 0 {
		return nil
	}
	out := make([]*Session, 0, len(sessions))
	for _, s := range sessions {
		out = append(out, s)
	}
	return out
}

// online reports whether any resource of the given bare JID is connected.
func (r *localRouter) online(bare jid.JID) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.byBare[bare.Bare().String()]) > 0
}
