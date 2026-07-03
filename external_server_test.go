package xmpp

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"testing"
	"time"

	"github.com/meszmate/xmpp-go/jid"
	"github.com/meszmate/xmpp-go/stanza"
)

// testCA is a throwaway certificate authority for exercising mutual-TLS
// (SASL EXTERNAL) in tests.
type testCA struct {
	cert *x509.Certificate
	key  *ecdsa.PrivateKey
	pool *x509.CertPool
}

func newTestCA(t *testing.T) *testCA {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("CA key: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "xmpp-go test CA"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("CA cert: %v", err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("parse CA cert: %v", err)
	}
	pool := x509.NewCertPool()
	pool.AddCert(cert)
	return &testCA{cert: cert, key: key, pool: pool}
}

// issue mints a leaf certificate signed by the CA. serial must be unique.
func (ca *testCA) issue(t *testing.T, serial int64, cn string, dnsNames []string, server bool) tls.Certificate {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("leaf key: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(serial),
		Subject:      pkix.Name{CommonName: cn},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		DNSNames:     dnsNames,
	}
	if server {
		tmpl.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}
	} else {
		tmpl.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, ca.cert, &key.PublicKey, ca.key)
	if err != nil {
		t.Fatalf("leaf cert: %v", err)
	}
	leaf, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("parse leaf: %v", err)
	}
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key, Leaf: leaf}
}

// TestSASLExternalClientCert proves end-to-end mutual-TLS authentication: the
// client presents a certificate over STARTTLS and the server maps it to a local
// identity via SASL EXTERNAL — no password involved.
func TestSASLExternalClientCert(t *testing.T) {
	ca := newTestCA(t)
	serverCert := ca.issue(t, 2, testDomain, []string{testDomain}, true)
	clientCert := ca.issue(t, 3, "alice", nil, false)

	identity := func(cert *x509.Certificate) (string, bool) {
		return cert.Subject.CommonName, cert.Subject.CommonName != ""
	}

	_, addr, _ := startServer(t,
		WithServerTLSConfig(&tls.Config{Certificates: []tls.Certificate{serverCert}}),
		WithServerClientCertAuth(ca.pool, identity),
	)

	clientTLS := &tls.Config{
		Certificates: []tls.Certificate{clientCert},
		RootCAs:      ca.pool,
		ServerName:   testDomain,
		MinVersion:   tls.VersionTLS12,
	}
	c, err := NewClient(
		jid.MustParse("alice@"+testDomain),
		"", // no password: authenticate purely by certificate
		WithConnectAddr(addr),
		WithClientTLS(clientTLS),
		WithSASLMechanisms("EXTERNAL"),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := c.Connect(ctx); err != nil {
		t.Fatalf("EXTERNAL Connect failed: %v", err)
	}
	if c.Session().State()&StateReady == 0 {
		t.Fatalf("session not Ready after EXTERNAL auth: state=%b", c.Session().State())
	}
	if got := c.Session().LocalAddr(); got.Local() != "alice" || got.Domain() != testDomain {
		t.Errorf("bound JID = %q, want alice@%s/<resource>", got, testDomain)
	}
	if c.Session().LocalAddr().Resource() == "" {
		t.Error("expected a bound resource after EXTERNAL auth")
	}
}

// TestSASLExternalNoCertRejected verifies that selecting EXTERNAL without having
// presented a client certificate is refused (the server never advertises it and
// rejects the auth).
func TestSASLExternalNoCertRejected(t *testing.T) {
	ca := newTestCA(t)
	serverCert := ca.issue(t, 2, testDomain, []string{testDomain}, true)
	identity := func(cert *x509.Certificate) (string, bool) {
		return cert.Subject.CommonName, true
	}

	_, addr, store := startServer(t,
		WithServerTLSConfig(&tls.Config{Certificates: []tls.Certificate{serverCert}}),
		WithServerClientCertAuth(ca.pool, identity),
	)
	addUser(t, store, "alice", "pw") // password auth still available

	// Client trusts the server but presents NO client certificate, yet demands
	// EXTERNAL. The server must not authenticate it.
	clientTLS := &tls.Config{RootCAs: ca.pool, ServerName: testDomain, MinVersion: tls.VersionTLS12}
	c, err := NewClient(
		jid.MustParse("alice@"+testDomain),
		"pw",
		WithConnectAddr(addr),
		WithClientTLS(clientTLS),
		WithSASLMechanisms("EXTERNAL"),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := c.Connect(ctx); err == nil {
		t.Fatal("EXTERNAL without a client certificate must fail, got nil")
	}
}

// TestSASLExternalCoexistsWithPassword confirms a client-cert server still lets
// ordinary password clients (SCRAM) log in over the same STARTTLS endpoint.
func TestSASLExternalCoexistsWithPassword(t *testing.T) {
	ca := newTestCA(t)
	serverCert := ca.issue(t, 2, testDomain, []string{testDomain}, true)
	identity := func(cert *x509.Certificate) (string, bool) { return cert.Subject.CommonName, true }

	_, addr, store := startServer(t,
		WithServerTLSConfig(&tls.Config{Certificates: []tls.Certificate{serverCert}}),
		WithServerClientCertAuth(ca.pool, identity),
	)
	addUser(t, store, "bob", "hunter2")

	clientTLS := &tls.Config{RootCAs: ca.pool, ServerName: testDomain, MinVersion: tls.VersionTLS12}
	c, err := NewClient(
		jid.MustParse("bob@"+testDomain),
		"hunter2",
		WithConnectAddr(addr),
		WithClientTLS(clientTLS),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := c.Connect(ctx); err != nil {
		t.Fatalf("password Connect over client-cert server failed: %v", err)
	}
	if c.Session().State()&StateReady == 0 {
		t.Error("password client not Ready")
	}
	// Sanity: the authenticated session can still send.
	msg := stanza.NewMessage(stanza.MessageChat)
	msg.To = jid.MustParse("bob@" + testDomain)
	msg.Body = "self"
	if err := c.Send(ctx, msg); err != nil {
		t.Errorf("Send after password auth: %v", err)
	}
}
