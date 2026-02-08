package omemo

import (
	"bytes"
	"crypto/ecdh"
	"encoding/binary"
	"fmt"
)

const maxSkippedKeys = 1000

// skippedKey identifies a skipped message key by ratchet public key and message number.
type skippedKey struct {
	dhPub [32]byte
	n     uint32
}

// RatchetState holds the state of a Double Ratchet session.
type RatchetState struct {
	DHs *ecdh.PrivateKey // our current ratchet key pair
	DHr []byte           // their current ratchet public key (32 bytes)

	RK  []byte // root key (32 bytes)
	CKs []byte // sending chain key (32 bytes)
	CKr []byte // receiving chain key (32 bytes)

	Ns  uint32 // sending message number
	Nr  uint32 // receiving message number
	PN  uint32 // previous sending chain length

	MKSkipped map[skippedKey][]byte // skipped message keys
}

// InitRatchetAsAlice initializes a Double Ratchet as Alice (initiator).
// Alice generates a new DH pair and derives the first sending chain from DH with Bob's SPK.
func InitRatchetAsAlice(sharedSecret, remoteSPK []byte) (*RatchetState, error) {
	dhs, err := GenerateX25519KeyPair()
	if err != nil {
		return nil, err
	}

	// Perform DH with Bob's signed pre-key
	dhOut, err := x25519DH(dhs, remoteSPK)
	if err != nil {
		return nil, err
	}

	rk, cks, err := rootKDF(sharedSecret, dhOut)
	if err != nil {
		return nil, err
	}

	return &RatchetState{
		DHs:       dhs,
		DHr:       remoteSPK,
		RK:        rk,
		CKs:       cks,
		CKr:       nil,
		Ns:        0,
		Nr:        0,
		PN:        0,
		MKSkipped: make(map[skippedKey][]byte),
	}, nil
}

// InitRatchetAsBob initializes a Double Ratchet as Bob (responder).
// Bob uses SPK as initial ratchet key, waits for Alice's first message to complete DH ratchet.
func InitRatchetAsBob(sharedSecret []byte, localSPK *ecdh.PrivateKey) *RatchetState {
	return &RatchetState{
		DHs:       localSPK,
		DHr:       nil,
		RK:        sharedSecret,
		CKs:       nil,
		CKr:       nil,
		Ns:        0,
		Nr:        0,
		PN:        0,
		MKSkipped: make(map[skippedKey][]byte),
	}
}

// RatchetEncrypt encrypts plaintext using the Double Ratchet.
func (s *RatchetState) RatchetEncrypt(plaintext []byte) (*RatchetHeader, []byte, error) {
	mk, nextCK := chainKDF(s.CKs)
	s.CKs = nextCK

	header := &RatchetHeader{
		DHPub: s.DHs.PublicKey().Bytes(),
		N:     s.Ns,
		PN:    s.PN,
	}
	s.Ns++

	nonce, ciphertext, err := aesGCMEncrypt(mk, plaintext)
	if err != nil {
		return nil, nil, err
	}

	// Prepend nonce to ciphertext
	out := make([]byte, len(nonce)+len(ciphertext))
	copy(out, nonce)
	copy(out[len(nonce):], ciphertext)

	return header, out, nil
}

// RatchetDecrypt decrypts a message using the Double Ratchet.
func (s *RatchetState) RatchetDecrypt(header *RatchetHeader, ciphertext []byte) ([]byte, error) {
	// 1. Try skipped keys
	if plaintext, err := s.trySkippedKeys(header, ciphertext); err == nil {
		return plaintext, nil
	}

	// 2. If new DH key, perform DH ratchet step
	if s.DHr == nil || !bytes.Equal(header.DHPub, s.DHr) {
		if err := s.skipMessageKeys(header.PN); err != nil {
			return nil, err
		}
		if err := s.dhRatchetStep(header.DHPub); err != nil {
			return nil, err
		}
	}

	// 3. Skip any messages in the current receiving chain
	if err := s.skipMessageKeys(header.N); err != nil {
		return nil, err
	}

	// 4. Derive message key and decrypt
	mk, nextCK := chainKDF(s.CKr)
	s.CKr = nextCK
	s.Nr++

	return decryptWithNonce(mk, ciphertext)
}

func (s *RatchetState) trySkippedKeys(header *RatchetHeader, ciphertext []byte) ([]byte, error) {
	var k skippedKey
	copy(k.dhPub[:], header.DHPub)
	k.n = header.N

	mk, ok := s.MKSkipped[k]
	if !ok {
		return nil, ErrInvalidMessage
	}

	delete(s.MKSkipped, k)
	return decryptWithNonce(mk, ciphertext)
}

func (s *RatchetState) skipMessageKeys(until uint32) error {
	if s.CKr == nil {
		return nil
	}
	if until > s.Nr+uint32(maxSkippedKeys) {
		return ErrSkippedKeyLimit
	}
	for s.Nr < until {
		mk, nextCK := chainKDF(s.CKr)
		s.CKr = nextCK

		var k skippedKey
		copy(k.dhPub[:], s.DHr)
		k.n = s.Nr
		s.MKSkipped[k] = mk
		s.Nr++

		if len(s.MKSkipped) > maxSkippedKeys {
			return ErrSkippedKeyLimit
		}
	}
	return nil
}

func (s *RatchetState) dhRatchetStep(newDHr []byte) error {
	s.PN = s.Ns
	s.Ns = 0
	s.Nr = 0
	s.DHr = make([]byte, 32)
	copy(s.DHr, newDHr)

	// DH with new remote key and our current key
	dhOut, err := x25519DH(s.DHs, s.DHr)
	if err != nil {
		return err
	}

	rk, ckr, err := rootKDF(s.RK, dhOut)
	if err != nil {
		return err
	}
	s.RK = rk
	s.CKr = ckr

	// Generate new DH key pair
	s.DHs, err = GenerateX25519KeyPair()
	if err != nil {
		return err
	}

	// DH with new remote key and our new key
	dhOut, err = x25519DH(s.DHs, s.DHr)
	if err != nil {
		return err
	}

	rk, cks, err := rootKDF(s.RK, dhOut)
	if err != nil {
		return err
	}
	s.RK = rk
	s.CKs = cks

	return nil
}

// decryptWithNonce extracts the 12-byte nonce from the front of ciphertext and decrypts.
func decryptWithNonce(mk, data []byte) ([]byte, error) {
	if len(data) < aesNonceSize {
		return nil, ErrInvalidMessage
	}
	return aesGCMDecrypt(mk, data[:aesNonceSize], data[aesNonceSize:])
}

// MarshalBinary serializes the RatchetState to bytes.
func (s *RatchetState) MarshalBinary() ([]byte, error) {
	var buf bytes.Buffer

	// DHs private key (32 bytes)
	dhsBytes := s.DHs.Bytes()
	buf.Write(dhsBytes)

	// DHr (1 byte flag + optional 32 bytes)
	if s.DHr != nil {
		buf.WriteByte(1)
		buf.Write(s.DHr)
	} else {
		buf.WriteByte(0)
	}

	// RK (32 bytes)
	buf.Write(s.RK)

	// CKs (1 byte flag + optional 32 bytes)
	writeOptionalKey(&buf, s.CKs)

	// CKr (1 byte flag + optional 32 bytes)
	writeOptionalKey(&buf, s.CKr)

	// Counters (4 bytes each)
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, s.Ns)
	buf.Write(b)
	binary.BigEndian.PutUint32(b, s.Nr)
	buf.Write(b)
	binary.BigEndian.PutUint32(b, s.PN)
	buf.Write(b)

	// Skipped keys: count (4 bytes), then each entry
	binary.BigEndian.PutUint32(b, uint32(len(s.MKSkipped)))
	buf.Write(b)
	for k, v := range s.MKSkipped {
		buf.Write(k.dhPub[:])
		binary.BigEndian.PutUint32(b, k.n)
		buf.Write(b)
		buf.Write(v) // 32 bytes
	}

	return buf.Bytes(), nil
}

// UnmarshalBinary deserializes a RatchetState from bytes.
func (s *RatchetState) UnmarshalBinary(data []byte) error {
	r := bytes.NewReader(data)

	// DHs
	dhsBytes := make([]byte, 32)
	if _, err := r.Read(dhsBytes); err != nil {
		return fmt.Errorf("%w: reading DHs: %v", ErrInvalidMessage, err)
	}
	var err error
	s.DHs, err = ecdh.X25519().NewPrivateKey(dhsBytes)
	if err != nil {
		return fmt.Errorf("%w: parsing DHs: %v", ErrInvalidMessage, err)
	}

	// DHr
	flag, err := r.ReadByte()
	if err != nil {
		return fmt.Errorf("%w: reading DHr flag: %v", ErrInvalidMessage, err)
	}
	if flag == 1 {
		s.DHr = make([]byte, 32)
		if _, err := r.Read(s.DHr); err != nil {
			return fmt.Errorf("%w: reading DHr: %v", ErrInvalidMessage, err)
		}
	}

	// RK
	s.RK = make([]byte, 32)
	if _, err := r.Read(s.RK); err != nil {
		return fmt.Errorf("%w: reading RK: %v", ErrInvalidMessage, err)
	}

	// CKs
	s.CKs, err = readOptionalKey(r)
	if err != nil {
		return fmt.Errorf("%w: reading CKs: %v", ErrInvalidMessage, err)
	}

	// CKr
	s.CKr, err = readOptionalKey(r)
	if err != nil {
		return fmt.Errorf("%w: reading CKr: %v", ErrInvalidMessage, err)
	}

	// Counters
	b := make([]byte, 4)
	if _, err := r.Read(b); err != nil {
		return fmt.Errorf("%w: reading Ns: %v", ErrInvalidMessage, err)
	}
	s.Ns = binary.BigEndian.Uint32(b)

	if _, err := r.Read(b); err != nil {
		return fmt.Errorf("%w: reading Nr: %v", ErrInvalidMessage, err)
	}
	s.Nr = binary.BigEndian.Uint32(b)

	if _, err := r.Read(b); err != nil {
		return fmt.Errorf("%w: reading PN: %v", ErrInvalidMessage, err)
	}
	s.PN = binary.BigEndian.Uint32(b)

	// Skipped keys
	if _, err := r.Read(b); err != nil {
		return fmt.Errorf("%w: reading skipped count: %v", ErrInvalidMessage, err)
	}
	count := binary.BigEndian.Uint32(b)
	s.MKSkipped = make(map[skippedKey][]byte, count)

	for range count {
		var k skippedKey
		if _, err := r.Read(k.dhPub[:]); err != nil {
			return fmt.Errorf("%w: reading skipped dhPub: %v", ErrInvalidMessage, err)
		}
		if _, err := r.Read(b); err != nil {
			return fmt.Errorf("%w: reading skipped n: %v", ErrInvalidMessage, err)
		}
		k.n = binary.BigEndian.Uint32(b)
		mk := make([]byte, 32)
		if _, err := r.Read(mk); err != nil {
			return fmt.Errorf("%w: reading skipped mk: %v", ErrInvalidMessage, err)
		}
		s.MKSkipped[k] = mk
	}

	return nil
}

func writeOptionalKey(buf *bytes.Buffer, key []byte) {
	if key != nil {
		buf.WriteByte(1)
		buf.Write(key)
	} else {
		buf.WriteByte(0)
	}
}

func readOptionalKey(r *bytes.Reader) ([]byte, error) {
	flag, err := r.ReadByte()
	if err != nil {
		return nil, err
	}
	if flag == 0 {
		return nil, nil
	}
	key := make([]byte, 32)
	if _, err := r.Read(key); err != nil {
		return nil, err
	}
	return key, nil
}
