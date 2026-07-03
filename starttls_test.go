package xmpp

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/meszmate/xmpp-go/jid"
)

// selfSignedCert writes a throwaway self-signed certificate/key for localhost to
// temp files and returns their paths.
func selfSignedCert(t *testing.T) (certPath, keyPath string) {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	tmpl := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "localhost"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     []string{"localhost"},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
	}
	der, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &priv.PublicKey, priv)
	if err != nil {
		t.Fatalf("CreateCertificate: %v", err)
	}
	dir := t.TempDir()
	certPath = filepath.Join(dir, "cert.pem")
	keyPath = filepath.Join(dir, "key.pem")

	certOut, err := os.Create(certPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: der}); err != nil {
		t.Fatal(err)
	}
	certOut.Close()

	keyOut, err := os.Create(keyPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)}); err != nil {
		t.Fatal(err)
	}
	keyOut.Close()
	return certPath, keyPath
}

func TestConnectSTARTTLS(t *testing.T) {
	certPath, keyPath := selfSignedCert(t)
	_, addr, store := startServer(t, WithServerTLS(certPath, keyPath))
	addUser(t, store, "alice", "s3cret")

	client, _ := NewClient(
		jid.MustParse("alice@"+testDomain),
		"s3cret",
		WithConnectAddr(addr),
		WithClientTLS(&tls.Config{InsecureSkipVerify: true, ServerName: "localhost"}),
	)
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Connect over STARTTLS failed: %v", err)
	}
	st := client.Session().State()
	if st&StateSecure == 0 {
		t.Error("session should be StateSecure after STARTTLS")
	}
	if st&StateReady == 0 {
		t.Error("session should be StateReady after full negotiation")
	}
}

func TestConnectPlainMechanism(t *testing.T) {
	_, addr, store := startServer(t)
	addUser(t, store, "alice", "s3cret")

	client, _ := NewClient(
		jid.MustParse("alice@"+testDomain),
		"s3cret",
		WithConnectAddr(addr),
		WithSASLMechanisms("PLAIN"),
		WithInsecureSASL(), // PLAIN over the test's non-TLS connection
	)
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Connect with PLAIN failed: %v", err)
	}
	if client.Session().State()&StateReady == 0 {
		t.Error("session not Ready after PLAIN auth")
	}
}

func TestPlainRefusedOverCleartext(t *testing.T) {
	_, addr, store := startServer(t)
	addUser(t, store, "alice", "s3cret")

	// Force PLAIN but do NOT opt in to insecure SASL: the client must refuse to
	// send the cleartext password over the unencrypted connection.
	client, _ := NewClient(
		jid.MustParse("alice@"+testDomain),
		"s3cret",
		WithConnectAddr(addr),
		WithSASLMechanisms("PLAIN"),
	)
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err := client.Connect(ctx)
	if err == nil {
		t.Fatal("expected Connect to refuse PLAIN over cleartext, got nil")
	}
	if !strings.Contains(err.Error(), "PLAIN") {
		t.Errorf("expected a PLAIN-refusal error, got: %v", err)
	}
}

func TestConnectScramWrongPassword(t *testing.T) {
	_, addr, store := startServer(t)
	addUser(t, store, "alice", "right")

	client, _ := NewClient(
		jid.MustParse("alice@"+testDomain),
		"wrong",
		WithConnectAddr(addr),
		WithSASLMechanisms("SCRAM-SHA-256"),
	)
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err := client.Connect(ctx)
	var authErr *AuthError
	if !errors.As(err, &authErr) {
		t.Fatalf("expected *AuthError from SCRAM with wrong password, got %T: %v", err, err)
	}
}

func TestConnectScramSHA1(t *testing.T) {
	_, addr, store := startServer(t)
	addUser(t, store, "alice", "s3cret")

	client, _ := NewClient(
		jid.MustParse("alice@"+testDomain),
		"s3cret",
		WithConnectAddr(addr),
		WithSASLMechanisms("SCRAM-SHA-1"),
	)
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Connect with SCRAM-SHA-1 failed: %v", err)
	}
	if client.Session().State()&StateReady == 0 {
		t.Error("session not Ready after SCRAM-SHA-1 auth")
	}
}
