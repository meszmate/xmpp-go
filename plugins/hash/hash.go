// Package hash implements XEP-0300 Cryptographic Hash Functions.
package hash

import (
	"context"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"hash"

	"github.com/meszmate/xmpp-go/internal/ns"
	"github.com/meszmate/xmpp-go/plugin"
)

const Name = "hash"

var ErrUnsupportedAlgo = errors.New("hash: unsupported algorithm")

// Hash represents a hash element.
type Hash struct {
	XMLName xml.Name `xml:"urn:xmpp:hashes:2 hash"`
	Algo    string   `xml:"algo,attr"`
	Value   string   `xml:",chardata"`
}

// HashUsed represents a hash-used element.
type HashUsed struct {
	XMLName xml.Name `xml:"urn:xmpp:hashes:2 hash-used"`
	Algo    string   `xml:"algo,attr"`
}

// Supported algorithms.
const (
	AlgoSHA256     = "sha-256"
	AlgoSHA512     = "sha-512"
	AlgoSHA3_256   = "sha3-256"
	AlgoSHA3_512   = "sha3-512"
	AlgoBLAKE2b256 = "blake2b-256"
	AlgoBLAKE2b512 = "blake2b-512"
)

// Plugin implements XEP-0300.
type Plugin struct {
	params plugin.InitParams
}

func New() *Plugin { return &Plugin{} }

func (p *Plugin) Name() string    { return Name }
func (p *Plugin) Version() string { return "1.0.0" }
func (p *Plugin) Initialize(_ context.Context, params plugin.InitParams) error {
	p.params = params
	return nil
}
func (p *Plugin) Close() error           { return nil }
func (p *Plugin) Dependencies() []string { return nil }

// Compute computes a hash of data using the given algorithm.
func Compute(algo string, data []byte) (Hash, error) {
	var h hash.Hash
	switch algo {
	case AlgoSHA256:
		h = sha256.New()
	case AlgoSHA512:
		h = sha512.New()
	default:
		return Hash{}, ErrUnsupportedAlgo
	}
	h.Write(data)
	return Hash{
		Algo:  algo,
		Value: base64.StdEncoding.EncodeToString(h.Sum(nil)),
	}, nil
}

// Verify verifies a hash against data.
func Verify(hv Hash, data []byte) (bool, error) {
	computed, err := Compute(hv.Algo, data)
	if err != nil {
		return false, err
	}
	return computed.Value == hv.Value, nil
}

func init() { _ = ns.Hashes }
