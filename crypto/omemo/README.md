# crypto/omemo -- OMEMO v2 Signal Protocol Implementation

This package implements the cryptographic layer of [OMEMO v2 (XEP-0384)](https://xmpp.org/extensions/xep-0384.html)
for end-to-end encrypted messaging over XMPP. It provides X3DH key agreement,
Double Ratchet message encryption, and a high-level Manager API.

This is a **standalone Go module** with no dependency on the main xmpp-go library.
You wire the two together on the client side.

For the full integration guide covering both server and client setup, see
[docs/omemo.md](../../docs/omemo.md).

## Install

```
go get github.com/meszmate/xmpp-go/crypto/omemo
```

## Where This Fits

This package is **client-side only**. The server stores public data (device lists, bundles with public keys) via `storage.PubSubStore`. This package stores private data (private keys, sessions, trust) locally. The server never sees private keys or session state -- that's what makes it end-to-end encrypted.

## Quick Start

```go
import "github.com/meszmate/xmpp-go/crypto/omemo"

// 1. Create a client-side store (use a real DB for production)
store := omemo.NewMemoryStore(myDeviceID)
manager := omemo.NewManager(store)

// 2. Generate bundle (private keys stay local)
bundle, _ := manager.GenerateBundle(25)
// Publish bundle.IdentityKey, bundle.SignedPreKey, bundle.PreKeys
// to the server via PEP (see docs/omemo.md)

// 3. Process a remote bundle fetched from the server
manager.ProcessBundle(addr, remoteBundleParsedFromXML)

// 4. Encrypt
encMsg, _ := manager.Encrypt([]byte("Hello!"), recipientAddr1, recipientAddr2)

// 5. Decrypt (existing session)
plaintext, _ := manager.Decrypt(senderAddr, encMsg)

// 6. Decrypt first message (pre-key message, creates session as Bob)
plaintext, _ := manager.DecryptPreKeyMessage(
    senderAddr, identityKey, ephemeralKey, preKeyID, signedPreKeyID, encMsg,
)
```

## API

### Manager

```go
manager := omemo.NewManager(store)

manager.GenerateBundle(preKeyCount int) (*Bundle, error)
manager.ProcessBundle(addr Address, bundle *Bundle)
manager.Encrypt(plaintext []byte, recipients ...Address) (*EncryptedMessage, error)
manager.Decrypt(sender Address, msg *EncryptedMessage) ([]byte, error)
manager.DecryptPreKeyMessage(sender Address, identityKey ed25519.PublicKey,
    ephemeralPubKey []byte, preKeyID *uint32, signedPreKeyID uint32,
    msg *EncryptedMessage) ([]byte, error)
```

### Types

```go
// Address uniquely identifies an OMEMO device
type Address struct {
    JID      string
    DeviceID uint32
}

// Bundle holds public key material for X3DH
type Bundle struct {
    IdentityKey           ed25519.PublicKey
    SignedPreKey          []byte   // X25519 public
    SignedPreKeyID        uint32
    SignedPreKeySignature []byte   // Ed25519 signature
    PreKeys               []BundlePreKey
}

// EncryptedMessage is the output of Encrypt / input to Decrypt
type EncryptedMessage struct {
    SenderDeviceID uint32
    Keys           []MessageKey  // one per recipient device
    IV             []byte        // 12 bytes
    Payload        []byte        // AES-GCM ciphertext (without auth tag)
}
```

### Store Interface

The `Store` is **client-side** storage for private cryptographic state. Implement
it backed by a local database for production. `MemoryStore` is for testing only.

```go
type Store interface {
    GetIdentityKeyPair() (*IdentityKeyPair, error)
    SaveIdentityKeyPair(ikp *IdentityKeyPair) error
    GetLocalDeviceID() (uint32, error)

    GetRemoteIdentity(addr Address) (ed25519.PublicKey, error)
    SaveRemoteIdentity(addr Address, key ed25519.PublicKey) error
    IsTrusted(addr Address, key ed25519.PublicKey) (bool, error)

    GetPreKey(id uint32) (*PreKeyRecord, error)
    SavePreKey(record *PreKeyRecord) error
    RemovePreKey(id uint32) error

    GetSignedPreKey(id uint32) (*SignedPreKeyRecord, error)
    SaveSignedPreKey(record *SignedPreKeyRecord) error

    GetSession(addr Address) ([]byte, error)
    SaveSession(addr Address, data []byte) error
    ContainsSession(addr Address) (bool, error)
}
```

## Cryptographic Primitives

| Component | Algorithm | Purpose |
|-----------|-----------|---------|
| Identity keys | Ed25519 | Long-term signing, converted to X25519 for DH |
| Key agreement | X3DH (X25519) | Establish shared secret between strangers |
| Message encryption | Double Ratchet | Per-message forward secrecy |
| Payload encryption | AES-256-GCM | Authenticated encryption of message content |
| Key derivation | HKDF-SHA-256 | Derive keys from shared secrets |

## Dependencies

Only `golang.org/x/crypto` (for HKDF). Everything else uses Go stdlib.
