package omemo

import (
	"crypto/ecdh"
	"crypto/ed25519"
)

// Session wraps a Double Ratchet state with session metadata.
type Session struct {
	Ratchet          *RatchetState
	RemoteIdentity   ed25519.PublicKey
	PendingPreKey    *PendingPreKey // set until the first reply is received
}

// PendingPreKey tracks pre-key info for the initial message.
type PendingPreKey struct {
	PreKeyID        *uint32
	SignedPreKeyID  uint32
	EphemeralPubKey []byte // 32 bytes, X25519
}

// InitSessionAsAlice creates a new session as the initiator using X3DH.
func InitSessionAsAlice(
	localIdentity *IdentityKeyPair,
	remoteBundle *Bundle,
) (*Session, error) {
	x3dhResult, err := X3DHInitiate(localIdentity, remoteBundle)
	if err != nil {
		return nil, err
	}

	ratchet, err := InitRatchetAsAlice(x3dhResult.SharedSecret, remoteBundle.SignedPreKey)
	if err != nil {
		return nil, err
	}

	return &Session{
		Ratchet:        ratchet,
		RemoteIdentity: remoteBundle.IdentityKey,
		PendingPreKey: &PendingPreKey{
			PreKeyID:        x3dhResult.UsedPreKeyID,
			SignedPreKeyID:  remoteBundle.SignedPreKeyID,
			EphemeralPubKey: x3dhResult.EphemeralPubKey,
		},
	}, nil
}

// InitSessionAsBob creates a new session as the responder using X3DH.
func InitSessionAsBob(
	localIdentity *IdentityKeyPair,
	localSPK *ecdh.PrivateKey,
	localOPK *ecdh.PrivateKey,
	remoteIdentityKey ed25519.PublicKey,
	ephemeralPubKey []byte,
) (*Session, error) {
	sharedSecret, err := X3DHRespond(localIdentity, localSPK, localOPK, remoteIdentityKey, ephemeralPubKey)
	if err != nil {
		return nil, err
	}

	ratchet := InitRatchetAsBob(sharedSecret, localSPK)

	return &Session{
		Ratchet:        ratchet,
		RemoteIdentity: remoteIdentityKey,
	}, nil
}

// Encrypt encrypts plaintext using this session's ratchet.
func (s *Session) Encrypt(plaintext []byte) (*RatchetHeader, []byte, bool, error) {
	header, ciphertext, err := s.Ratchet.RatchetEncrypt(plaintext)
	if err != nil {
		return nil, nil, false, err
	}
	isPreKey := s.PendingPreKey != nil
	return header, ciphertext, isPreKey, nil
}

// Decrypt decrypts a message using this session's ratchet.
func (s *Session) Decrypt(header *RatchetHeader, ciphertext []byte) ([]byte, error) {
	plaintext, err := s.Ratchet.RatchetDecrypt(header, ciphertext)
	if err != nil {
		return nil, err
	}
	// Clear pending pre-key after first successful decrypt (means we got a reply)
	s.PendingPreKey = nil
	return plaintext, nil
}

// MarshalBinary serializes the session state.
func (s *Session) MarshalBinary() ([]byte, error) {
	ratchetData, err := s.Ratchet.MarshalBinary()
	if err != nil {
		return nil, err
	}

	// Format: [remoteIdentity(32)] [pendingPreKeyFlag(1)] [pendingPreKey...] [ratchetData...]
	size := 32 + 1 + len(ratchetData)
	hasPending := s.PendingPreKey != nil
	if hasPending {
		// preKeyID flag(1) + optional preKeyID(4) + signedPreKeyID(4) + ephemeralPubKey(32)
		size += 1 + 4 + 32
		if s.PendingPreKey.PreKeyID != nil {
			size += 4
		}
	}

	buf := make([]byte, 0, size)
	buf = append(buf, s.RemoteIdentity...)

	if hasPending {
		buf = append(buf, 1)
		ppk := s.PendingPreKey
		if ppk.PreKeyID != nil {
			buf = append(buf, 1)
			buf = appendUint32(buf, *ppk.PreKeyID)
		} else {
			buf = append(buf, 0)
		}
		buf = appendUint32(buf, ppk.SignedPreKeyID)
		buf = append(buf, ppk.EphemeralPubKey...)
	} else {
		buf = append(buf, 0)
	}

	buf = append(buf, ratchetData...)
	return buf, nil
}

// UnmarshalBinary deserializes a session from bytes.
func (s *Session) UnmarshalBinary(data []byte) error {
	if len(data) < 33 {
		return ErrInvalidMessage
	}

	s.RemoteIdentity = make(ed25519.PublicKey, 32)
	copy(s.RemoteIdentity, data[:32])
	pos := 32

	pendingFlag := data[pos]
	pos++

	if pendingFlag == 1 {
		s.PendingPreKey = &PendingPreKey{}
		preKeyFlag := data[pos]
		pos++

		if preKeyFlag == 1 {
			if pos+4 > len(data) {
				return ErrInvalidMessage
			}
			id := readUint32(data[pos:])
			s.PendingPreKey.PreKeyID = &id
			pos += 4
		}

		if pos+4 > len(data) {
			return ErrInvalidMessage
		}
		s.PendingPreKey.SignedPreKeyID = readUint32(data[pos:])
		pos += 4

		if pos+32 > len(data) {
			return ErrInvalidMessage
		}
		s.PendingPreKey.EphemeralPubKey = make([]byte, 32)
		copy(s.PendingPreKey.EphemeralPubKey, data[pos:pos+32])
		pos += 32
	}

	s.Ratchet = &RatchetState{}
	return s.Ratchet.UnmarshalBinary(data[pos:])
}

func appendUint32(buf []byte, v uint32) []byte {
	return append(buf, byte(v>>24), byte(v>>16), byte(v>>8), byte(v))
}

func readUint32(b []byte) uint32 {
	return uint32(b[0])<<24 | uint32(b[1])<<16 | uint32(b[2])<<8 | uint32(b[3])
}
