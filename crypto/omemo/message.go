package omemo

// EncryptedMessage represents an OMEMO encrypted message ready for XML serialization.
type EncryptedMessage struct {
	SenderDeviceID uint32
	Keys           []MessageKey
	IV             []byte // 12 bytes
	Payload        []byte // AES-GCM ciphertext without auth tag
}

// MessageKey holds the encrypted key material for a single recipient device.
type MessageKey struct {
	DeviceID uint32
	Data     []byte // ratchet-encrypted key material (header + ciphertext)
	IsPreKey bool   // true if this is a pre-key message (first message in a session)
}
