package sasl

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strings"
	"testing"

	"golang.org/x/crypto/pbkdf2"
)

func TestSCRAMConstructors(t *testing.T) {
	t.Parallel()
	creds := Credentials{Username: "user", Password: "pass"}
	tests := []struct {
		name    string
		create  func(Credentials) *SCRAM
		wantName string
	}{
		{"SHA1", NewSCRAMSHA1, "SCRAM-SHA-1"},
		{"SHA1-PLUS", NewSCRAMSHA1Plus, "SCRAM-SHA-1-PLUS"},
		{"SHA256", NewSCRAMSHA256, "SCRAM-SHA-256"},
		{"SHA256-PLUS", NewSCRAMSHA256Plus, "SCRAM-SHA-256-PLUS"},
		{"SHA512", NewSCRAMSHA512, "SCRAM-SHA-512"},
		{"SHA512-PLUS", NewSCRAMSHA512Plus, "SCRAM-SHA-512-PLUS"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := tt.create(creds)
			if s.Name() != tt.wantName {
				t.Errorf("Name() = %q, want %q", s.Name(), tt.wantName)
			}
		})
	}
}

func TestSCRAMStart(t *testing.T) {
	t.Parallel()
	creds := Credentials{Username: "user", Password: "pass"}
	s := NewSCRAMSHA256(creds)

	resp, err := s.Start()
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	str := string(resp)
	// Should start with gs2-header "n,,"
	if !strings.HasPrefix(str, "n,,") {
		t.Errorf("client-first should start with 'n,,', got %q", str)
	}
	// Should contain n=user
	if !strings.Contains(str, "n=user") {
		t.Errorf("missing n=user in: %s", str)
	}
	// Should contain r= (nonce)
	if !strings.Contains(str, "r=") {
		t.Errorf("missing nonce in: %s", str)
	}
	if s.Completed() {
		t.Error("should not be completed after Start")
	}
}

func TestSCRAMStartPlus(t *testing.T) {
	t.Parallel()
	creds := Credentials{
		Username:       "user",
		Password:       "pass",
		ChannelBinding: []byte("binding-data"),
		CBType:         "tls-exporter",
	}
	s := NewSCRAMSHA256Plus(creds)

	resp, err := s.Start()
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	str := string(resp)
	if !strings.HasPrefix(str, "p=tls-exporter,,") {
		t.Errorf("PLUS client-first should start with gs2 cb header, got %q", str)
	}
}

func TestSCRAMStartPlusNoBinding(t *testing.T) {
	t.Parallel()
	creds := Credentials{Username: "user", Password: "pass"}
	s := NewSCRAMSHA256Plus(creds)

	_, err := s.Start()
	if err != ErrChannelBinding {
		t.Errorf("Start without channel binding should return ErrChannelBinding, got %v", err)
	}
}

func TestSCRAMFullExchange(t *testing.T) {
	t.Parallel()
	password := "pencil"
	salt := []byte("salt-value-here!")
	iterations := 4096

	creds := Credentials{Username: "user", Password: password}
	s := NewSCRAMSHA256(creds)

	// Step 1: Client-first
	clientFirst, err := s.Start()
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	// Extract the client nonce from client-first
	cfStr := string(clientFirst)
	// Strip gs2-header to get bare
	bareIdx := strings.Index(cfStr, "n=")
	clientFirstBare := cfStr[bareIdx:]

	// Extract client nonce
	var clientNonce string
	for _, part := range strings.Split(clientFirstBare, ",") {
		if strings.HasPrefix(part, "r=") {
			clientNonce = part[2:]
			break
		}
	}

	// Server creates server-first with extended nonce
	serverNonce := clientNonce + "server-extension"
	saltB64 := base64.StdEncoding.EncodeToString(salt)
	serverFirst := fmt.Sprintf("r=%s,s=%s,i=%d", serverNonce, saltB64, iterations)

	// Step 2: Client processes server-first
	clientFinal, err := s.Next([]byte(serverFirst))
	if err != nil {
		t.Fatalf("Next (server-first): %v", err)
	}

	cfFinalStr := string(clientFinal)
	if !strings.Contains(cfFinalStr, "r="+serverNonce) {
		t.Errorf("client-final missing server nonce: %s", cfFinalStr)
	}
	if !strings.Contains(cfFinalStr, "p=") {
		t.Errorf("client-final missing proof: %s", cfFinalStr)
	}

	// Compute expected server signature for verification
	saltedPwd := pbkdf2.Key([]byte(password), salt, iterations, sha256.Size, sha256.New)
	serverKey := hmacSHA256(saltedPwd, []byte("Server Key"))

	// Reconstruct auth message
	cbData := base64.StdEncoding.EncodeToString([]byte("n,,"))
	clientFinalNoProof := fmt.Sprintf("c=%s,r=%s", cbData, serverNonce)
	authMessage := fmt.Sprintf("%s,%s,%s", clientFirstBare, serverFirst, clientFinalNoProof)

	serverSig := hmacSHA256(serverKey, []byte(authMessage))
	verifier := base64.StdEncoding.EncodeToString(serverSig)

	// Step 3: Server-final
	serverFinal := fmt.Sprintf("v=%s", verifier)
	_, err = s.Next([]byte(serverFinal))
	if err != nil {
		t.Fatalf("Next (server-final): %v", err)
	}

	if !s.Completed() {
		t.Error("should be completed after full exchange")
	}
}

func TestSCRAMServerError(t *testing.T) {
	t.Parallel()
	creds := Credentials{Username: "user", Password: "pass"}
	s := NewSCRAMSHA256(creds)

	clientFirst, _ := s.Start()
	cfStr := string(clientFirst)
	bareIdx := strings.Index(cfStr, "n=")
	clientFirstBare := cfStr[bareIdx:]

	var clientNonce string
	for _, part := range strings.Split(clientFirstBare, ",") {
		if strings.HasPrefix(part, "r=") {
			clientNonce = part[2:]
			break
		}
	}

	salt := base64.StdEncoding.EncodeToString([]byte("salt"))
	serverFirst := fmt.Sprintf("r=%sserver,s=%s,i=4096", clientNonce, salt)
	s.Next([]byte(serverFirst))

	// Server sends error
	_, err := s.Next([]byte("e=invalid-proof"))
	if err == nil {
		t.Error("expected error from server error response")
	}
}

func hmacSHA256(key, data []byte) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write(data)
	return mac.Sum(nil)
}
