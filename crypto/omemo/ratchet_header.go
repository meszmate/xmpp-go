package omemo

import (
	"encoding/binary"
	"fmt"
)

// RatchetHeader contains the public information sent with each ratchet message.
type RatchetHeader struct {
	DHPub []byte // 32 bytes, X25519 public ratchet key
	N     uint32 // message number in sending chain
	PN    uint32 // previous chain length
}

const ratchetHeaderSize = 32 + 4 + 4 // 40 bytes

// MarshalBinary encodes a RatchetHeader to bytes.
func (h *RatchetHeader) MarshalBinary() ([]byte, error) {
	if len(h.DHPub) != 32 {
		return nil, ErrInvalidKeyLength
	}
	buf := make([]byte, ratchetHeaderSize)
	copy(buf[:32], h.DHPub)
	binary.BigEndian.PutUint32(buf[32:36], h.N)
	binary.BigEndian.PutUint32(buf[36:40], h.PN)
	return buf, nil
}

// UnmarshalBinary decodes a RatchetHeader from bytes.
func (h *RatchetHeader) UnmarshalBinary(data []byte) error {
	if len(data) != ratchetHeaderSize {
		return fmt.Errorf("%w: header size %d, expected %d", ErrInvalidMessage, len(data), ratchetHeaderSize)
	}
	h.DHPub = make([]byte, 32)
	copy(h.DHPub, data[:32])
	h.N = binary.BigEndian.Uint32(data[32:36])
	h.PN = binary.BigEndian.Uint32(data[36:40])
	return nil
}
