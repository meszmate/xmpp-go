package omemo

import (
	"crypto/hmac"
	"crypto/sha256"
	"io"

	"golang.org/x/crypto/hkdf"
)

// hkdfSHA256 derives a key of the given length using HKDF-SHA-256.
func hkdfSHA256(salt, ikm, info []byte, length int) ([]byte, error) {
	r := hkdf.New(sha256.New, ikm, salt, info)
	out := make([]byte, length)
	if _, err := io.ReadFull(r, out); err != nil {
		return nil, err
	}
	return out, nil
}

// chainKDF derives a message key and next chain key from a chain key.
// messageKey = HMAC-SHA256(CK, 0x01)
// nextChainKey = HMAC-SHA256(CK, 0x02)
func chainKDF(chainKey []byte) (messageKey, nextChainKey []byte) {
	mk := hmac.New(sha256.New, chainKey)
	mk.Write([]byte{0x01})
	messageKey = mk.Sum(nil)

	ck := hmac.New(sha256.New, chainKey)
	ck.Write([]byte{0x02})
	nextChainKey = ck.Sum(nil)

	return messageKey, nextChainKey
}

// rootKDF derives a new root key and chain key from the current root key and DH output.
// Uses HKDF with salt=RK, ikm=DHOutput, info="OMEMO Root Chain", producing 64 bytes
// split into new RK (first 32) and new CK (last 32).
func rootKDF(rootKey, dhOutput []byte) (newRootKey, newChainKey []byte, err error) {
	out, err := hkdfSHA256(rootKey, dhOutput, []byte("OMEMO Root Chain"), 64)
	if err != nil {
		return nil, nil, err
	}
	return out[:32], out[32:], nil
}
