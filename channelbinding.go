package xmpp

import (
	"crypto/sha256"
	"crypto/sha512"
	"crypto/tls"
	"crypto/x509"
	"hash"
)

// tlsServerEndpoint computes the RFC 5929 "tls-server-end-point" channel-binding
// data for a certificate: the hash of the certificate's DER encoding, using the
// hash of the certificate's signature algorithm — but substituting SHA-256 when
// that would be MD5 or SHA-1. This value is agreed upon by both peers (the
// server from its own certificate, the client from the one it received) and
// bound into the SCRAM-*-PLUS exchange to defeat man-in-the-middle attacks.
func tlsServerEndpoint(cert *x509.Certificate) []byte {
	if cert == nil {
		return nil
	}
	var h hash.Hash
	switch cert.SignatureAlgorithm {
	case x509.SHA384WithRSA, x509.ECDSAWithSHA384, x509.SHA384WithRSAPSS:
		h = sha512.New384()
	case x509.SHA512WithRSA, x509.ECDSAWithSHA512, x509.SHA512WithRSAPSS:
		h = sha512.New()
	default:
		// Covers SHA-256 signatures and the RFC 5929 rule that MD5/SHA-1
		// signature hashes are upgraded to SHA-256 for channel binding.
		h = sha256.New()
	}
	h.Write(cert.Raw)
	return h.Sum(nil)
}

// serverEndpointFromTLS computes the tls-server-end-point channel-binding data
// for a server's own leaf certificate, parsing it from the DER if the tls
// package did not already populate Leaf. Returns nil when no certificate is
// configured.
func serverEndpointFromTLS(cfg *tls.Config) []byte {
	if cfg == nil || len(cfg.Certificates) == 0 {
		return nil
	}
	c := cfg.Certificates[0]
	leaf := c.Leaf
	if leaf == nil {
		if len(c.Certificate) == 0 {
			return nil
		}
		parsed, err := x509.ParseCertificate(c.Certificate[0])
		if err != nil {
			return nil
		}
		leaf = parsed
	}
	return tlsServerEndpoint(leaf)
}

// clientChannelBinding returns the tls-server-end-point channel-binding data for
// the session's TLS peer (server) certificate, or nil when the session is not
// over real TLS with a server certificate.
func clientChannelBinding(session *Session) []byte {
	cs, ok := session.Transport().ConnectionState()
	if !ok || len(cs.PeerCertificates) == 0 {
		return nil
	}
	return tlsServerEndpoint(cs.PeerCertificates[0])
}
