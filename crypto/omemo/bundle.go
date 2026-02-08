package omemo

import "crypto/ed25519"

// Bundle holds the public key material needed for X3DH key agreement.
type Bundle struct {
	IdentityKey         ed25519.PublicKey
	SignedPreKey        []byte // 32 bytes, X25519 public key
	SignedPreKeyID      uint32
	SignedPreKeySignature []byte // Ed25519 signature over SignedPreKey
	PreKeys             []BundlePreKey
}

// BundlePreKey is a one-time pre-key in a bundle.
type BundlePreKey struct {
	ID        uint32
	PublicKey []byte // 32 bytes, X25519
}

// GenerateBundle generates a new OMEMO bundle, storing keys in the provided store.
func GenerateBundle(store Store, preKeyCount int) (*Bundle, error) {
	// Load or generate identity key pair
	ikp, err := store.GetIdentityKeyPair()
	if err != nil {
		return nil, err
	}
	if ikp == nil {
		ikp, err = GenerateIdentityKeyPair()
		if err != nil {
			return nil, err
		}
		if err := store.SaveIdentityKeyPair(ikp); err != nil {
			return nil, err
		}
	}

	// Generate signed pre-key
	spk, err := generateSignedPreKey(ikp, 1)
	if err != nil {
		return nil, err
	}
	if err := store.SaveSignedPreKey(spk.record); err != nil {
		return nil, err
	}

	// Generate one-time pre-keys
	preKeys := make([]BundlePreKey, 0, preKeyCount)
	for i := range preKeyCount {
		pk, err := generatePreKey(uint32(i + 1))
		if err != nil {
			return nil, err
		}
		if err := store.SavePreKey(pk); err != nil {
			return nil, err
		}
		preKeys = append(preKeys, BundlePreKey{
			ID:        pk.ID,
			PublicKey: pk.PublicKey,
		})
	}

	return &Bundle{
		IdentityKey:           ikp.PublicKey,
		SignedPreKey:          spk.record.PublicKey,
		SignedPreKeyID:        spk.record.ID,
		SignedPreKeySignature: spk.record.Signature,
		PreKeys:               preKeys,
	}, nil
}

type signedPreKeyResult struct {
	record *SignedPreKeyRecord
}

func generateSignedPreKey(ikp *IdentityKeyPair, id uint32) (*signedPreKeyResult, error) {
	key, err := GenerateX25519KeyPair()
	if err != nil {
		return nil, err
	}

	pubBytes := key.PublicKey().Bytes()

	// Sign the public key with the Ed25519 identity key
	sig := ed25519.Sign(ikp.PrivateKey, pubBytes)

	return &signedPreKeyResult{
		record: &SignedPreKeyRecord{
			ID:         id,
			PrivateKey: key.Bytes(),
			PublicKey:  pubBytes,
			Signature:  sig,
		},
	}, nil
}

func generatePreKey(id uint32) (*PreKeyRecord, error) {
	key, err := GenerateX25519KeyPair()
	if err != nil {
		return nil, err
	}
	return &PreKeyRecord{
		ID:         id,
		PrivateKey: key.Bytes(),
		PublicKey:  key.PublicKey().Bytes(),
	}, nil
}
