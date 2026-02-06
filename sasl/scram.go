package sasl

import (
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

// SCRAM implements SCRAM-SHA-* SASL mechanisms (RFC 5802).
type SCRAM struct {
	creds       Credentials
	hashFunc    func() hash.Hash
	name        string
	plus        bool
	step        int
	clientNonce string
	serverNonce string
	salt        []byte
	iterations  int
	authMessage string
	saltedPwd   []byte
	gs2Header   string
	clientFirst string
}

// NewSCRAMSHA1 creates a SCRAM-SHA-1 mechanism.
func NewSCRAMSHA1(creds Credentials) *SCRAM {
	return newSCRAM(creds, "SCRAM-SHA-1", sha1.New, false)
}

// NewSCRAMSHA1Plus creates a SCRAM-SHA-1-PLUS mechanism.
func NewSCRAMSHA1Plus(creds Credentials) *SCRAM {
	return newSCRAM(creds, "SCRAM-SHA-1-PLUS", sha1.New, true)
}

// NewSCRAMSHA256 creates a SCRAM-SHA-256 mechanism.
func NewSCRAMSHA256(creds Credentials) *SCRAM {
	return newSCRAM(creds, "SCRAM-SHA-256", sha256.New, false)
}

// NewSCRAMSHA256Plus creates a SCRAM-SHA-256-PLUS mechanism.
func NewSCRAMSHA256Plus(creds Credentials) *SCRAM {
	return newSCRAM(creds, "SCRAM-SHA-256-PLUS", sha256.New, true)
}

// NewSCRAMSHA512 creates a SCRAM-SHA-512 mechanism.
func NewSCRAMSHA512(creds Credentials) *SCRAM {
	return newSCRAM(creds, "SCRAM-SHA-512", sha512.New, false)
}

// NewSCRAMSHA512Plus creates a SCRAM-SHA-512-PLUS mechanism.
func NewSCRAMSHA512Plus(creds Credentials) *SCRAM {
	return newSCRAM(creds, "SCRAM-SHA-512-PLUS", sha512.New, true)
}

func newSCRAM(creds Credentials, name string, h func() hash.Hash, plus bool) *SCRAM {
	return &SCRAM{
		creds:    creds,
		hashFunc: h,
		name:     name,
		plus:     plus,
	}
}

// Name returns the mechanism name.
func (s *SCRAM) Name() string { return s.name }

// Completed returns true after all steps are done.
func (s *SCRAM) Completed() bool { return s.step >= 3 }

// Start creates the client-first message.
func (s *SCRAM) Start() ([]byte, error) {
	s.clientNonce = generateNonce()

	if s.plus {
		if len(s.creds.ChannelBinding) == 0 {
			return nil, ErrChannelBinding
		}
		s.gs2Header = fmt.Sprintf("p=%s,,", s.creds.CBType)
	} else {
		s.gs2Header = "n,,"
	}

	s.clientFirst = fmt.Sprintf("n=%s,r=%s", escapeSCRAM(s.creds.Username), s.clientNonce)
	s.step = 1
	return []byte(s.gs2Header + s.clientFirst), nil
}

// Next processes server challenges.
func (s *SCRAM) Next(challenge []byte) ([]byte, error) {
	switch s.step {
	case 1:
		return s.processServerFirst(challenge)
	case 2:
		return s.processServerFinal(challenge)
	default:
		return nil, errors.New("sasl: SCRAM unexpected step")
	}
}

func (s *SCRAM) processServerFirst(data []byte) ([]byte, error) {
	parts := parseSCRAMAttributes(string(data))

	nonce, ok := parts["r"]
	if !ok || !strings.HasPrefix(nonce, s.clientNonce) {
		return nil, ErrInvalidResponse
	}
	s.serverNonce = nonce

	saltB64, ok := parts["s"]
	if !ok {
		return nil, ErrInvalidResponse
	}
	salt, err := base64.StdEncoding.DecodeString(saltB64)
	if err != nil {
		return nil, err
	}
	s.salt = salt

	iterStr, ok := parts["i"]
	if !ok {
		return nil, ErrInvalidResponse
	}
	fmt.Sscanf(iterStr, "%d", &s.iterations)
	if s.iterations <= 0 {
		return nil, errors.New("sasl: invalid iteration count")
	}

	// Channel binding data
	var cbData []byte
	cbData = append(cbData, []byte(s.gs2Header)...)
	if s.plus {
		cbData = append(cbData, s.creds.ChannelBinding...)
	}
	cbB64 := base64.StdEncoding.EncodeToString(cbData)

	clientFinalNoProof := fmt.Sprintf("c=%s,r=%s", cbB64, s.serverNonce)

	s.saltedPwd = pbkdf2.Key([]byte(s.creds.Password), s.salt, s.iterations, s.hashFunc().Size(), s.hashFunc)

	clientKey := hmacHash(s.hashFunc, s.saltedPwd, []byte("Client Key"))
	storedKey := hashBytes(s.hashFunc, clientKey)

	s.authMessage = fmt.Sprintf("%s,%s,%s", s.clientFirst, string(data), clientFinalNoProof)
	clientSig := hmacHash(s.hashFunc, storedKey, []byte(s.authMessage))
	proof := xorBytes(clientKey, clientSig)
	proofB64 := base64.StdEncoding.EncodeToString(proof)

	s.step = 2
	return []byte(fmt.Sprintf("%s,p=%s", clientFinalNoProof, proofB64)), nil
}

func (s *SCRAM) processServerFinal(data []byte) ([]byte, error) {
	parts := parseSCRAMAttributes(string(data))

	if errMsg, ok := parts["e"]; ok {
		return nil, fmt.Errorf("sasl: server error: %s", errMsg)
	}

	verifier, ok := parts["v"]
	if !ok {
		return nil, ErrInvalidResponse
	}

	serverKey := hmacHash(s.hashFunc, s.saltedPwd, []byte("Server Key"))
	expected := hmacHash(s.hashFunc, serverKey, []byte(s.authMessage))
	expectedB64 := base64.StdEncoding.EncodeToString(expected)

	if verifier != expectedB64 {
		return nil, ErrAuthFailed
	}

	s.step = 3
	return nil, nil
}

func parseSCRAMAttributes(s string) map[string]string {
	attrs := make(map[string]string)
	for _, part := range strings.Split(s, ",") {
		if idx := strings.IndexByte(part, '='); idx > 0 {
			attrs[part[:idx]] = part[idx+1:]
		}
	}
	return attrs
}

func escapeSCRAM(s string) string {
	s = strings.ReplaceAll(s, "=", "=3D")
	s = strings.ReplaceAll(s, ",", "=2C")
	return s
}

func hmacHash(h func() hash.Hash, key, data []byte) []byte {
	mac := hmac.New(h, key)
	mac.Write(data)
	return mac.Sum(nil)
}

func hashBytes(h func() hash.Hash, data []byte) []byte {
	hasher := h()
	hasher.Write(data)
	return hasher.Sum(nil)
}

func xorBytes(a, b []byte) []byte {
	result := make([]byte, len(a))
	for i := range a {
		result[i] = a[i] ^ b[i]
	}
	return result
}

func generateNonce() string {
	b := make([]byte, 24)
	_, _ = rand.Read(b)
	return base64.StdEncoding.EncodeToString(b)
}
