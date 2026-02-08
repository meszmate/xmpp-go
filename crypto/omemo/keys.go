package omemo

import (
	"crypto/ecdh"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha512"
	"math/big"
)

// IdentityKeyPair holds an Ed25519 identity key pair.
type IdentityKeyPair struct {
	PrivateKey ed25519.PrivateKey
	PublicKey  ed25519.PublicKey
}

// GenerateIdentityKeyPair generates a new Ed25519 identity key pair.
func GenerateIdentityKeyPair() (*IdentityKeyPair, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	return &IdentityKeyPair{
		PrivateKey: priv,
		PublicKey:  pub,
	}, nil
}

// GenerateX25519KeyPair generates a new X25519 key pair.
func GenerateX25519KeyPair() (*ecdh.PrivateKey, error) {
	return ecdh.X25519().GenerateKey(rand.Reader)
}

// p is the field prime 2^255 - 19.
var p = func() *big.Int {
	p := new(big.Int).SetBit(new(big.Int), 255, 1)
	p.Sub(p, big.NewInt(19))
	return p
}()

// Ed25519PrivateKeyToX25519 converts an Ed25519 private key to an X25519 private key.
// It hashes the 32-byte seed with SHA-512, clamps the first 32 bytes, and uses
// the result as the X25519 scalar.
func Ed25519PrivateKeyToX25519(edPriv ed25519.PrivateKey) (*ecdh.PrivateKey, error) {
	seed := edPriv.Seed()
	h := sha512.Sum512(seed)
	// Clamp
	h[0] &= 248
	h[31] &= 127
	h[31] |= 64
	return ecdh.X25519().NewPrivateKey(h[:32])
}

// Ed25519PublicKeyToX25519 converts an Ed25519 public key to an X25519 public key
// using the birational map u = (1+y)/(1-y) mod p.
func Ed25519PublicKeyToX25519(edPub ed25519.PublicKey) ([]byte, error) {
	if len(edPub) != ed25519.PublicKeySize {
		return nil, ErrInvalidKeyLength
	}

	// Ed25519 public key is a compressed Edwards point; the 255 bits encode the
	// y-coordinate (little-endian) and the high bit is the sign of x.
	// For the Montgomery conversion we only need y.
	yBytes := make([]byte, 32)
	copy(yBytes, edPub)
	yBytes[31] &= 0x7F // clear sign bit

	// Convert from little-endian to big.Int
	// big.Int expects big-endian, so reverse
	reversed := make([]byte, 32)
	for i := 0; i < 32; i++ {
		reversed[i] = yBytes[31-i]
	}
	y := new(big.Int).SetBytes(reversed)

	// u = (1 + y) / (1 - y) mod p
	one := big.NewInt(1)
	numerator := new(big.Int).Add(one, y)
	numerator.Mod(numerator, p)

	denominator := new(big.Int).Sub(new(big.Int).Set(one), y)
	denominator.Mod(denominator, p)

	// Modular inverse of denominator
	denomInv := new(big.Int).ModInverse(denominator, p)
	if denomInv == nil {
		return nil, ErrInvalidKeyLength
	}

	u := new(big.Int).Mul(numerator, denomInv)
	u.Mod(u, p)

	// Convert u to 32-byte little-endian
	uBytes := make([]byte, 32)
	uBig := u.Bytes() // big-endian
	for i, b := range uBig {
		uBytes[len(uBig)-1-i] = b
	}

	return uBytes, nil
}

// x25519DH performs an X25519 Diffie-Hellman exchange.
func x25519DH(privateKey *ecdh.PrivateKey, publicKeyBytes []byte) ([]byte, error) {
	pub, err := ecdh.X25519().NewPublicKey(publicKeyBytes)
	if err != nil {
		return nil, err
	}
	return privateKey.ECDH(pub)
}
