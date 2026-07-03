package sasl

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"errors"
	"fmt"
	"hash"
	"strings"

	"golang.org/x/crypto/pbkdf2"
)

// PasswordLookup returns the plaintext password for a username and whether the
// user exists. It backs server-side SCRAM authentication.
type PasswordLookup func(username string) (password string, ok bool)

// SCRAMServer implements the server side of the SCRAM-SHA-* family (RFC 5802).
//
// Usage:
//
//	srv := NewSCRAMServerSHA256(lookup)
//	serverFirst, _ := srv.Start(clientFirst)   // -> send as <challenge>
//	serverFinal, err := srv.Finish(clientFinal) // -> send as <success> on nil err
//	if err != nil { /* send <failure> */ }
type SCRAMServer struct {
	name       string
	hashFunc   func() hash.Hash
	lookup     PasswordLookup
	iterations int
	plus       bool
	cbData     []byte // channel-binding data (excludes the gs2 header) for -PLUS

	step            int
	username        string
	clientNonce     string
	serverNonce     string
	salt            []byte
	saltedPwd       []byte
	gs2Header       string
	clientFirstBare string
	serverFirst     string
	userExists      bool
	completed       bool
}

// NewSCRAMServerSHA1 creates a server-side SCRAM-SHA-1 mechanism.
func NewSCRAMServerSHA1(lookup PasswordLookup) *SCRAMServer {
	return newSCRAMServer("SCRAM-SHA-1", sha1.New, lookup)
}

// NewSCRAMServerSHA256 creates a server-side SCRAM-SHA-256 mechanism.
func NewSCRAMServerSHA256(lookup PasswordLookup) *SCRAMServer {
	return newSCRAMServer("SCRAM-SHA-256", sha256.New, lookup)
}

// NewSCRAMServerSHA512 creates a server-side SCRAM-SHA-512 mechanism.
func NewSCRAMServerSHA512(lookup PasswordLookup) *SCRAMServer {
	return newSCRAMServer("SCRAM-SHA-512", sha512.New, lookup)
}

// NewSCRAMServer creates a server-side SCRAM mechanism for the named variant,
// including the -PLUS channel-binding forms ("SCRAM-SHA-256", "SCRAM-SHA-256-PLUS",
// etc.). It returns nil for an unknown name. For -PLUS variants the caller must
// supply the channel-binding data via SetChannelBinding before Start.
func NewSCRAMServer(name string, lookup PasswordLookup) *SCRAMServer {
	switch strings.ToUpper(name) {
	case "SCRAM-SHA-1":
		return NewSCRAMServerSHA1(lookup)
	case "SCRAM-SHA-256":
		return NewSCRAMServerSHA256(lookup)
	case "SCRAM-SHA-512":
		return NewSCRAMServerSHA512(lookup)
	case "SCRAM-SHA-1-PLUS":
		return newSCRAMServerPlus("SCRAM-SHA-1-PLUS", sha1.New, lookup)
	case "SCRAM-SHA-256-PLUS":
		return newSCRAMServerPlus("SCRAM-SHA-256-PLUS", sha256.New, lookup)
	case "SCRAM-SHA-512-PLUS":
		return newSCRAMServerPlus("SCRAM-SHA-512-PLUS", sha512.New, lookup)
	default:
		return nil
	}
}

func newSCRAMServer(name string, h func() hash.Hash, lookup PasswordLookup) *SCRAMServer {
	return &SCRAMServer{
		name:       name,
		hashFunc:   h,
		lookup:     lookup,
		iterations: 4096,
	}
}

func newSCRAMServerPlus(name string, h func() hash.Hash, lookup PasswordLookup) *SCRAMServer {
	s := newSCRAMServer(name, h, lookup)
	s.plus = true
	return s
}

// SetChannelBinding provides the TLS channel-binding data (for tls-server-
// end-point, the certificate hash) that a -PLUS exchange must match. It has no
// effect on non-PLUS mechanisms.
func (s *SCRAMServer) SetChannelBinding(data []byte) { s.cbData = data }

// Name returns the mechanism name.
func (s *SCRAMServer) Name() string { return s.name }

// Username returns the authenticating username parsed from the client-first
// message. It is valid after Start.
func (s *SCRAMServer) Username() string { return s.username }

// Completed reports whether the exchange has finished (successfully or not).
func (s *SCRAMServer) Completed() bool { return s.completed }

// Start processes the client-first message and returns the server-first message
// (r=,s=,i=). It never reveals whether the user exists: an unknown user is
// processed with a random salt so that failure only surfaces at Finish.
func (s *SCRAMServer) Start(clientFirst []byte) ([]byte, error) {
	if s.step != 0 {
		return nil, errors.New("sasl: SCRAM server: Start called out of order")
	}
	str := string(clientFirst)

	// gs2-header = gs2-cbind-flag "," [ authzid ] ","
	firstComma := strings.IndexByte(str, ',')
	if firstComma < 0 {
		return nil, ErrInvalidResponse
	}
	rest := str[firstComma+1:]
	secondRel := strings.IndexByte(rest, ',')
	if secondRel < 0 {
		return nil, ErrInvalidResponse
	}
	secondComma := firstComma + 1 + secondRel
	s.gs2Header = str[:secondComma+1]
	s.clientFirstBare = str[secondComma+1:]

	// Enforce channel-binding flag consistency with the mechanism variant:
	//   - a -PLUS mechanism MUST carry "p=" (channel binding in use);
	//   - a non-PLUS mechanism MUST NOT carry "p=".
	usesCB := strings.HasPrefix(str, "p=")
	if usesCB != s.plus {
		return nil, ErrChannelBinding
	}

	attrs := parseSCRAMAttributes(s.clientFirstBare)
	user, ok := attrs["n"]
	if !ok {
		return nil, ErrInvalidResponse
	}
	cnonce, ok := attrs["r"]
	if !ok || cnonce == "" {
		return nil, ErrInvalidResponse
	}
	s.username = unescapeSCRAM(user)
	s.clientNonce = cnonce
	s.serverNonce = cnonce + generateNonce()

	s.salt = make([]byte, 16)
	if _, err := rand.Read(s.salt); err != nil {
		return nil, err
	}

	password := ""
	if s.lookup != nil {
		password, s.userExists = s.lookup(s.username)
	}
	if !s.userExists {
		// Derive a stable-but-useless password so timing/shape does not leak
		// user existence; Finish will reject the proof.
		password = base64.StdEncoding.EncodeToString(s.salt)
	}

	s.saltedPwd = pbkdf2.Key([]byte(password), s.salt, s.iterations, s.hashFunc().Size(), s.hashFunc)
	s.serverFirst = fmt.Sprintf("r=%s,s=%s,i=%d", s.serverNonce, base64.StdEncoding.EncodeToString(s.salt), s.iterations)
	s.step = 1
	return []byte(s.serverFirst), nil
}

// Finish processes the client-final message. On success it returns the
// server-final message (v=...) to send in <success>. On failure it returns
// ErrAuthFailed (bad password / unknown user) or a protocol error.
func (s *SCRAMServer) Finish(clientFinal []byte) ([]byte, error) {
	if s.step != 1 {
		return nil, errors.New("sasl: SCRAM server: Finish called out of order")
	}
	s.completed = true
	str := string(clientFinal)
	attrs := parseSCRAMAttributes(str)

	cb, ok := attrs["c"]
	if !ok {
		return nil, ErrInvalidResponse
	}
	gotGS2, err := base64.StdEncoding.DecodeString(cb)
	if err != nil {
		return nil, ErrInvalidResponse
	}
	// The c= attribute is base64(gs2-header [|| channel-binding-data]). For a
	// -PLUS exchange the client appends the TLS channel-binding data; verifying
	// it binds the SASL exchange to this exact TLS channel (anti-MITM).
	expectedCB := []byte(s.gs2Header)
	if s.plus {
		expectedCB = append(expectedCB, s.cbData...)
	}
	if !bytes.Equal(gotGS2, expectedCB) {
		return nil, ErrChannelBinding
	}

	rnonce, ok := attrs["r"]
	if !ok || rnonce != s.serverNonce {
		return nil, ErrInvalidResponse
	}
	proofB64, ok := attrs["p"]
	if !ok {
		return nil, ErrInvalidResponse
	}
	proof, err := base64.StdEncoding.DecodeString(proofB64)
	if err != nil {
		return nil, ErrInvalidResponse
	}

	clientFinalNoProof := clientFinalWithoutProof(str)
	authMessage := s.clientFirstBare + "," + s.serverFirst + "," + clientFinalNoProof

	clientKey := hmacHash(s.hashFunc, s.saltedPwd, []byte("Client Key"))
	storedKey := hashBytes(s.hashFunc, clientKey)
	clientSig := hmacHash(s.hashFunc, storedKey, []byte(authMessage))
	if len(proof) != len(clientSig) {
		return nil, ErrAuthFailed
	}
	recoveredClientKey := xorBytes(proof, clientSig)
	recoveredStoredKey := hashBytes(s.hashFunc, recoveredClientKey)

	if !s.userExists || !hmac.Equal(recoveredStoredKey, storedKey) {
		return nil, ErrAuthFailed
	}

	serverKey := hmacHash(s.hashFunc, s.saltedPwd, []byte("Server Key"))
	serverSig := hmacHash(s.hashFunc, serverKey, []byte(authMessage))
	return []byte("v=" + base64.StdEncoding.EncodeToString(serverSig)), nil
}

// clientFinalWithoutProof strips the trailing ",p=<proof>" from the client-final
// message, yielding the client-final-message-without-proof.
func clientFinalWithoutProof(s string) string {
	if idx := strings.Index(s, ",p="); idx >= 0 {
		return s[:idx]
	}
	return s
}

// unescapeSCRAM reverses escapeSCRAM (=2C -> ",", =3D -> "=").
func unescapeSCRAM(s string) string {
	s = strings.ReplaceAll(s, "=2C", ",")
	s = strings.ReplaceAll(s, "=3D", "=")
	return s
}
