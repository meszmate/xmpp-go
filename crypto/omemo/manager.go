package omemo

import (
	"crypto/ecdh"
	"crypto/ed25519"
	"crypto/rand"
	"fmt"
	"sync"
)

// Manager provides the high-level API for OMEMO encryption and decryption.
type Manager struct {
	mu      sync.Mutex
	store   Store
	bundles map[Address]*Bundle   // cached remote bundles
	sessions map[Address]*Session // active sessions
}

// NewManager creates a new OMEMO Manager.
func NewManager(store Store) *Manager {
	return &Manager{
		store:    store,
		bundles:  make(map[Address]*Bundle),
		sessions: make(map[Address]*Session),
	}
}

// ProcessBundle stores a remote bundle for later X3DH initiation.
func (m *Manager) ProcessBundle(addr Address, bundle *Bundle) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.bundles[addr] = bundle
}

// GenerateBundle generates a new OMEMO bundle for the local device.
func (m *Manager) GenerateBundle(preKeyCount int) (*Bundle, error) {
	return GenerateBundle(m.store, preKeyCount)
}

// Encrypt encrypts plaintext for multiple recipients.
func (m *Manager) Encrypt(plaintext []byte, recipients ...Address) (*EncryptedMessage, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 1. Generate random 32-byte message key
	messageKey := make([]byte, 32)
	if _, err := rand.Read(messageKey); err != nil {
		return nil, err
	}

	// 2. AES-256-GCM encrypt plaintext
	iv, fullCiphertext, err := aesGCMEncrypt(messageKey, plaintext)
	if err != nil {
		return nil, err
	}

	// Separate ciphertext and auth tag
	// fullCiphertext = ciphertext || authTag (16 bytes)
	if len(fullCiphertext) < aesTagSize {
		return nil, ErrInvalidMessage
	}
	ciphertextWithoutTag := fullCiphertext[:len(fullCiphertext)-aesTagSize]
	authTag := fullCiphertext[len(fullCiphertext)-aesTagSize:]

	// 3. key_material = message_key(32) || auth_tag(16) = 48 bytes
	keyMaterial := make([]byte, 48)
	copy(keyMaterial[:32], messageKey)
	copy(keyMaterial[32:], authTag)

	// 4. For each recipient device: ratchet-encrypt key_material
	deviceID, err := m.store.GetLocalDeviceID()
	if err != nil {
		return nil, err
	}

	keys := make([]MessageKey, 0, len(recipients))
	for _, addr := range recipients {
		session, err := m.getOrCreateSession(addr)
		if err != nil {
			return nil, fmt.Errorf("session for %s: %w", addr, err)
		}

		header, ct, isPreKey, err := session.Encrypt(keyMaterial)
		if err != nil {
			return nil, fmt.Errorf("encrypt for %s: %w", addr, err)
		}

		// Serialize header + ciphertext together
		headerBytes, err := header.MarshalBinary()
		if err != nil {
			return nil, err
		}
		data := make([]byte, len(headerBytes)+len(ct))
		copy(data, headerBytes)
		copy(data[len(headerBytes):], ct)

		keys = append(keys, MessageKey{
			DeviceID: addr.DeviceID,
			Data:     data,
			IsPreKey: isPreKey,
		})

		// Save session
		if err := m.saveSession(addr, session); err != nil {
			return nil, err
		}
	}

	return &EncryptedMessage{
		SenderDeviceID: deviceID,
		Keys:           keys,
		IV:             iv,
		Payload:        ciphertextWithoutTag,
	}, nil
}

// Decrypt decrypts an OMEMO encrypted message.
func (m *Manager) Decrypt(sender Address, msg *EncryptedMessage) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 1. Find our MessageKey by device ID
	deviceID, err := m.store.GetLocalDeviceID()
	if err != nil {
		return nil, err
	}

	var ourKey *MessageKey
	for i := range msg.Keys {
		if msg.Keys[i].DeviceID == deviceID {
			ourKey = &msg.Keys[i]
			break
		}
	}
	if ourKey == nil {
		return nil, fmt.Errorf("%w: no key for device %d", ErrInvalidMessage, deviceID)
	}

	// Parse header from the key data
	if len(ourKey.Data) < ratchetHeaderSize {
		return nil, ErrInvalidMessage
	}
	var header RatchetHeader
	if err := header.UnmarshalBinary(ourKey.Data[:ratchetHeaderSize]); err != nil {
		return nil, err
	}
	ratchetCiphertext := ourKey.Data[ratchetHeaderSize:]

	// 2. Get or create session
	session, err := m.getOrCreateSessionForDecrypt(sender, ourKey.IsPreKey)
	if err != nil {
		return nil, err
	}

	// 3. Ratchet-decrypt → 48-byte key_material
	keyMaterial, err := session.Decrypt(&header, ratchetCiphertext)
	if err != nil {
		return nil, err
	}

	if len(keyMaterial) != 48 {
		return nil, fmt.Errorf("%w: key material length %d, expected 48", ErrInvalidMessage, len(keyMaterial))
	}

	// 4. Split: message_key = [:32], auth_tag = [32:48]
	messageKey := keyMaterial[:32]
	authTag := keyMaterial[32:48]

	// 5. AES-GCM decrypt payload||authTag with messageKey and IV
	fullCiphertext := make([]byte, len(msg.Payload)+len(authTag))
	copy(fullCiphertext, msg.Payload)
	copy(fullCiphertext[len(msg.Payload):], authTag)

	plaintext, err := aesGCMDecrypt(messageKey, msg.IV, fullCiphertext)
	if err != nil {
		return nil, err
	}

	// Save session
	if err := m.saveSession(sender, session); err != nil {
		return nil, err
	}

	return plaintext, nil
}

func (m *Manager) getOrCreateSession(addr Address) (*Session, error) {
	// Check in-memory sessions first
	if session, ok := m.sessions[addr]; ok {
		return session, nil
	}

	// Try loading from store
	data, err := m.store.GetSession(addr)
	if err == nil {
		session := &Session{}
		if err := session.UnmarshalBinary(data); err == nil {
			m.sessions[addr] = session
			return session, nil
		}
	}

	// No existing session; try to create one from a cached bundle
	bundle, ok := m.bundles[addr]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrNoSession, addr)
	}

	ikp, err := m.store.GetIdentityKeyPair()
	if err != nil {
		return nil, err
	}
	if ikp == nil {
		return nil, fmt.Errorf("no local identity key pair")
	}

	session, err := InitSessionAsAlice(ikp, bundle)
	if err != nil {
		return nil, err
	}

	// Save remote identity (TOFU)
	if err := m.store.SaveRemoteIdentity(addr, bundle.IdentityKey); err != nil {
		return nil, err
	}

	// Consume the used pre-key from the cached bundle
	if len(bundle.PreKeys) > 0 {
		bundle.PreKeys = bundle.PreKeys[1:]
	}

	m.sessions[addr] = session
	return session, nil
}

func (m *Manager) getOrCreateSessionForDecrypt(sender Address, isPreKey bool) (*Session, error) {
	// Try existing session first
	if session, ok := m.sessions[sender]; ok {
		return session, nil
	}

	// Try loading from store
	data, err := m.store.GetSession(sender)
	if err == nil {
		session := &Session{}
		if err := session.UnmarshalBinary(data); err == nil {
			m.sessions[sender] = session
			return session, nil
		}
	}

	// If this is a pre-key message, create session as Bob
	if !isPreKey {
		return nil, fmt.Errorf("%w: %s", ErrNoSession, sender)
	}

	return m.createSessionFromPreKeyMessage(sender)
}

func (m *Manager) createSessionFromPreKeyMessage(sender Address) (*Session, error) {
	bundle, ok := m.bundles[sender]
	if !ok {
		return nil, fmt.Errorf("%w: no bundle for %s", ErrNoSession, sender)
	}

	ikp, err := m.store.GetIdentityKeyPair()
	if err != nil {
		return nil, err
	}
	if ikp == nil {
		return nil, fmt.Errorf("no local identity key pair")
	}

	// We need to find the SPK that was used and the OPK
	// For the responder side, we need the actual private keys from our store
	spkRecord, err := m.store.GetSignedPreKey(bundle.SignedPreKeyID)
	if err != nil {
		return nil, fmt.Errorf("getting signed pre-key: %w", err)
	}

	spkPrivate, err := ecdh.X25519().NewPrivateKey(spkRecord.PrivateKey)
	if err != nil {
		return nil, err
	}

	// For Bob, we need to find the ephemeral key and remote identity from the incoming message
	// The bundle here is the *sender's* bundle that we previously processed
	var opkPrivate *ecdh.PrivateKey

	session, err := InitSessionAsBob(ikp, spkPrivate, opkPrivate, bundle.IdentityKey, nil)
	if err != nil {
		return nil, err
	}

	// Save remote identity (TOFU)
	if err := m.store.SaveRemoteIdentity(sender, bundle.IdentityKey); err != nil {
		return nil, err
	}

	m.sessions[sender] = session
	return session, nil
}

func (m *Manager) saveSession(addr Address, session *Session) error {
	data, err := session.MarshalBinary()
	if err != nil {
		return err
	}
	return m.store.SaveSession(addr, data)
}

// DecryptPreKeyMessage handles the full pre-key message decryption flow.
// It takes the sender's identity key, ephemeral key, and optionally the used pre-key ID
// to establish a new session as Bob and decrypt the message.
func (m *Manager) DecryptPreKeyMessage(
	sender Address,
	senderIdentityKey ed25519.PublicKey,
	ephemeralPubKey []byte,
	usedPreKeyID *uint32,
	signedPreKeyID uint32,
	msg *EncryptedMessage,
) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Get our identity key pair
	ikp, err := m.store.GetIdentityKeyPair()
	if err != nil {
		return nil, err
	}
	if ikp == nil {
		return nil, fmt.Errorf("no local identity key pair")
	}

	// Get our signed pre-key
	spkRecord, err := m.store.GetSignedPreKey(signedPreKeyID)
	if err != nil {
		return nil, fmt.Errorf("getting signed pre-key %d: %w", signedPreKeyID, err)
	}
	spkPrivate, err := ecdh.X25519().NewPrivateKey(spkRecord.PrivateKey)
	if err != nil {
		return nil, err
	}

	// Get our one-time pre-key if used
	var opkPrivate *ecdh.PrivateKey
	if usedPreKeyID != nil {
		opkRecord, err := m.store.GetPreKey(*usedPreKeyID)
		if err != nil {
			return nil, fmt.Errorf("getting pre-key %d: %w", *usedPreKeyID, err)
		}
		opkPrivate, err = ecdh.X25519().NewPrivateKey(opkRecord.PrivateKey)
		if err != nil {
			return nil, err
		}
		// Remove the used one-time pre-key
		_ = m.store.RemovePreKey(*usedPreKeyID)
	}

	// Create session as Bob
	session, err := InitSessionAsBob(ikp, spkPrivate, opkPrivate, senderIdentityKey, ephemeralPubKey)
	if err != nil {
		return nil, err
	}

	// Save remote identity
	if err := m.store.SaveRemoteIdentity(sender, senderIdentityKey); err != nil {
		return nil, err
	}

	m.sessions[sender] = session

	// Find our key
	deviceID, err := m.store.GetLocalDeviceID()
	if err != nil {
		return nil, err
	}

	var ourKey *MessageKey
	for i := range msg.Keys {
		if msg.Keys[i].DeviceID == deviceID {
			ourKey = &msg.Keys[i]
			break
		}
	}
	if ourKey == nil {
		return nil, fmt.Errorf("%w: no key for device %d", ErrInvalidMessage, deviceID)
	}

	// Parse header
	if len(ourKey.Data) < ratchetHeaderSize {
		return nil, ErrInvalidMessage
	}
	var header RatchetHeader
	if err := header.UnmarshalBinary(ourKey.Data[:ratchetHeaderSize]); err != nil {
		return nil, err
	}
	ratchetCiphertext := ourKey.Data[ratchetHeaderSize:]

	// Ratchet-decrypt
	keyMaterial, err := session.Decrypt(&header, ratchetCiphertext)
	if err != nil {
		return nil, err
	}

	if len(keyMaterial) != 48 {
		return nil, fmt.Errorf("%w: key material length %d, expected 48", ErrInvalidMessage, len(keyMaterial))
	}

	messageKey := keyMaterial[:32]
	authTag := keyMaterial[32:48]

	fullCiphertext := make([]byte, len(msg.Payload)+len(authTag))
	copy(fullCiphertext, msg.Payload)
	copy(fullCiphertext[len(msg.Payload):], authTag)

	plaintext, err := aesGCMDecrypt(messageKey, msg.IV, fullCiphertext)
	if err != nil {
		return nil, err
	}

	// Save session
	if err := m.saveSession(sender, session); err != nil {
		return nil, err
	}

	return plaintext, nil
}
