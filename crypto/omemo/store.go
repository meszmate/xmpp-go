package omemo

import "crypto/ed25519"

// PreKeyRecord holds a one-time pre-key pair.
type PreKeyRecord struct {
	ID         uint32
	PrivateKey []byte // 32 bytes, X25519
	PublicKey  []byte // 32 bytes, X25519
}

// SignedPreKeyRecord holds a signed pre-key pair with its signature.
type SignedPreKeyRecord struct {
	ID         uint32
	PrivateKey []byte // 32 bytes, X25519
	PublicKey  []byte // 32 bytes, X25519
	Signature  []byte // Ed25519 signature over PublicKey
}

// Store defines the persistence interface for OMEMO state.
type Store interface {
	// GetIdentityKeyPair returns the local identity key pair.
	GetIdentityKeyPair() (*IdentityKeyPair, error)

	// SaveIdentityKeyPair stores the local identity key pair.
	SaveIdentityKeyPair(ikp *IdentityKeyPair) error

	// GetLocalDeviceID returns the local device ID.
	GetLocalDeviceID() (uint32, error)

	// GetRemoteIdentity returns the stored identity public key for an address.
	GetRemoteIdentity(addr Address) (ed25519.PublicKey, error)

	// SaveRemoteIdentity stores a remote identity public key.
	SaveRemoteIdentity(addr Address, key ed25519.PublicKey) error

	// IsTrusted returns whether the identity key for an address is trusted.
	IsTrusted(addr Address, key ed25519.PublicKey) (bool, error)

	// GetPreKey returns a pre-key by ID.
	GetPreKey(id uint32) (*PreKeyRecord, error)

	// SavePreKey stores a pre-key.
	SavePreKey(record *PreKeyRecord) error

	// RemovePreKey removes a pre-key by ID.
	RemovePreKey(id uint32) error

	// GetSignedPreKey returns a signed pre-key by ID.
	GetSignedPreKey(id uint32) (*SignedPreKeyRecord, error)

	// SaveSignedPreKey stores a signed pre-key.
	SaveSignedPreKey(record *SignedPreKeyRecord) error

	// GetSession returns the serialized session state for an address.
	GetSession(addr Address) ([]byte, error)

	// SaveSession stores the serialized session state for an address.
	SaveSession(addr Address, data []byte) error

	// ContainsSession returns whether a session exists for an address.
	ContainsSession(addr Address) (bool, error)
}
