// Package omemo implements the Signal protocol cryptographic primitives for
// OMEMO v2 (XEP-0384) encryption in XMPP.
//
// This package provides X3DH key agreement, Double Ratchet message encryption,
// and a high-level Manager API for encrypting and decrypting OMEMO messages.
// It is a standalone cryptographic module with no dependency on the main
// xmpp-go module -- users wire the two together on the client side.
package omemo
