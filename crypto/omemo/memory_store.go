package omemo

import (
	"bytes"
	"crypto/ed25519"
	"sync"
)

// MemoryStore is an in-memory Store implementation for testing.
// It uses a Trust On First Use (TOFU) model for identity trust.
type MemoryStore struct {
	mu            sync.RWMutex
	identityKey   *IdentityKeyPair
	deviceID      uint32
	remoteKeys    map[Address]ed25519.PublicKey
	preKeys       map[uint32]*PreKeyRecord
	signedPreKeys map[uint32]*SignedPreKeyRecord
	sessions      map[Address][]byte
}

// NewMemoryStore creates a new in-memory store with the given device ID.
func NewMemoryStore(deviceID uint32) *MemoryStore {
	return &MemoryStore{
		deviceID:      deviceID,
		remoteKeys:    make(map[Address]ed25519.PublicKey),
		preKeys:       make(map[uint32]*PreKeyRecord),
		signedPreKeys: make(map[uint32]*SignedPreKeyRecord),
		sessions:      make(map[Address][]byte),
	}
}

func (s *MemoryStore) GetIdentityKeyPair() (*IdentityKeyPair, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.identityKey, nil
}

func (s *MemoryStore) SaveIdentityKeyPair(ikp *IdentityKeyPair) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.identityKey = ikp
	return nil
}

func (s *MemoryStore) GetLocalDeviceID() (uint32, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.deviceID, nil
}

func (s *MemoryStore) GetRemoteIdentity(addr Address) (ed25519.PublicKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	key, ok := s.remoteKeys[addr]
	if !ok {
		return nil, nil
	}
	return key, nil
}

func (s *MemoryStore) SaveRemoteIdentity(addr Address, key ed25519.PublicKey) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.remoteKeys[addr] = key
	return nil
}

// IsTrusted implements TOFU: trust on first use, reject on change.
func (s *MemoryStore) IsTrusted(addr Address, key ed25519.PublicKey) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	existing, ok := s.remoteKeys[addr]
	if !ok {
		return true, nil // first use: trust
	}
	return bytes.Equal(existing, key), nil
}

func (s *MemoryStore) GetPreKey(id uint32) (*PreKeyRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	pk, ok := s.preKeys[id]
	if !ok {
		return nil, ErrNoPreKey
	}
	return pk, nil
}

func (s *MemoryStore) SavePreKey(record *PreKeyRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.preKeys[record.ID] = record
	return nil
}

func (s *MemoryStore) RemovePreKey(id uint32) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.preKeys, id)
	return nil
}

func (s *MemoryStore) GetSignedPreKey(id uint32) (*SignedPreKeyRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	spk, ok := s.signedPreKeys[id]
	if !ok {
		return nil, ErrNoPreKey
	}
	return spk, nil
}

func (s *MemoryStore) SaveSignedPreKey(record *SignedPreKeyRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.signedPreKeys[record.ID] = record
	return nil
}

func (s *MemoryStore) GetSession(addr Address) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	data, ok := s.sessions[addr]
	if !ok {
		return nil, ErrNoSession
	}
	cp := make([]byte, len(data))
	copy(cp, data)
	return cp, nil
}

func (s *MemoryStore) SaveSession(addr Address, data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := make([]byte, len(data))
	copy(cp, data)
	s.sessions[addr] = cp
	return nil
}

func (s *MemoryStore) ContainsSession(addr Address) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.sessions[addr]
	return ok, nil
}
