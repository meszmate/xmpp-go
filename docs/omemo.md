# OMEMO Encryption Guide

OMEMO v2 ([XEP-0384](https://xmpp.org/extensions/xep-0384.html)) provides end-to-end encrypted messaging over XMPP using the Signal protocol. This guide explains how OMEMO works across the server and client sides of xmpp-go, and how to wire everything together.

## How OMEMO Works

OMEMO encrypts messages so that only the intended recipient devices can decrypt them. Even the XMPP server cannot read the message content. It uses three cryptographic building blocks:

1. **X3DH** (Extended Triple Diffie-Hellman) -- establishes a shared secret between two devices that have never communicated before
2. **Double Ratchet** -- derives a unique encryption key for every single message, providing forward secrecy
3. **AES-256-GCM** -- encrypts the actual message content

To make this work, devices need to discover each other and exchange public key material. This happens through PEP (Personal Eventing Protocol), which is server-side PubSub storage.

## Architecture

OMEMO spans three packages, two of which run on the client and one on the server:

**`plugins/omemo`** (client-side) -- XML types (`Encrypted`, `DeviceList`, `Bundle`, `Key`, `Payload`) and an in-memory device cache (`SetDevices()`/`GetDevices()`). This package converts between XMPP XML and Go types. It communicates with the server via PubSub IQ stanzas to publish and fetch device lists and bundles.

**`crypto/omemo`** (client-side) -- Signal protocol implementation: X3DH key agreement, Double Ratchet, AES-256-GCM, and the high-level `Manager` API (`Encrypt`/`Decrypt`). Its `Store` interface holds private cryptographic state locally: identity key pair, pre-key private keys, Double Ratchet sessions, and remote identity trust. The server never sees any of this data.

**`plugins/pubsub` + `storage.PubSubStore`** (server-side) -- Persists public data only: device lists and bundles (public keys). Backed by any configured storage backend (SQLite, Postgres, MySQL, MongoDB, Redis, etc.). The server has no access to private keys or session state.

### What the Server Stores (Public)

The XMPP server persists two PEP nodes per user account via `storage.PubSubStore`:

| PEP Node | Content | Item ID |
|----------|---------|---------|
| `urn:xmpp:omemo:2:devices` | List of device IDs for this account | `"current"` (single item, overwritten each update) |
| `urn:xmpp:omemo:2:bundles` | Key bundles with **public** keys only | Device ID as string (e.g. `"12345"`) |

When a client sends a PubSub IQ `<set>` to publish its device list or bundle, the server calls `PubSubStore.UpsertItem()` to save it to the database. When another client sends an IQ `<get>` to fetch a contact's devices, the server calls `PubSubStore.GetItems()` and returns the result.

The server never sees private keys or message plaintext. It only stores and relays public information.

### What the Client Stores (Private)

The client persists its cryptographic state locally via `crypto/omemo.Store`:

| Data | Purpose |
|------|---------|
| Identity key pair (Ed25519) | Your long-term signing key. The private half never leaves the client. |
| Pre-key private keys (X25519) | Needed to complete X3DH when someone messages you for the first time. Only the public halves are published to the server in the bundle. |
| Signed pre-key (X25519) | Medium-term key, rotated periodically. Also published as public only. |
| Double Ratchet session state | Per-device encryption state (root keys, chain keys, message counters, skipped keys). Completely client-side. |
| Remote identity keys | Public keys of contacts, stored locally for trust verification (TOFU). |

If the client loses this store, all active sessions are lost and must be re-established.

## Server Setup

### No OMEMO Plugin Needed on the Server

The server does **not** need any OMEMO-specific plugin or module. OMEMO is
end-to-end encryption -- the server never encrypts or decrypts anything. It only
needs standard PubSub/PEP support to store and relay public key material.

If you've seen other XMPP servers (Prosody, ejabberd) mention "OMEMO modules",
here's what's actually going on:

- **Prosody** had `mod_omemo_all_access` -- it did zero cryptography. All it did
  was change the PubSub access model on OMEMO nodes from `presence` (contacts
  only) to `open` (anyone can read). This is now built into `mod_pep` since
  Prosody 0.11 and the module is obsolete.
- **ejabberd** has no `mod_omemo`. It just needs `mod_pubsub`/`mod_pep` with
  support for changing PEP node access models (added in ejabberd 17.12).

The actual server-side requirements for OMEMO are all standard PubSub features:

| Requirement | Why |
|-------------|-----|
| Persistent PEP items | Device lists and bundles must survive server restarts |
| Open access model | Anyone must be able to fetch your bundle, not just contacts (needed for first-contact messaging) |
| Multiple items per node | The bundles node stores one item per device |
| Publish-options | Clients set the access model when publishing |

In xmpp-go, the `plugins/pubsub` plugin + any storage backend already provides
all of this. Just enable PubSub with a persistent store:

```go
import (
    xmpp "github.com/meszmate/xmpp-go"
    "github.com/meszmate/xmpp-go/plugins/pubsub"
    "github.com/meszmate/xmpp-go/storage/sqlite"
)

store, _ := sqlite.New("xmpp.db")

server, _ := xmpp.NewServer("example.com",
    xmpp.WithServerStorage(store),
    xmpp.WithServerPlugins(
        pubsub.New(),
        // ... other plugins
    ),
)
```

That's it. The `PubSubStore` handles OMEMO data automatically -- device lists
and bundles are just regular PubSub items stored in your database. The server
is a dumb pipe for public key material.

## Client Setup

The client is where all the OMEMO logic lives. You need three things:

1. **`plugins/omemo`** -- XML stanza types and a device list cache
2. **`plugins/pubsub`** -- stanza types for publishing and fetching PEP nodes
3. **`crypto/omemo`** -- the actual Signal protocol encryption

### Install the Crypto Module

The crypto module is a separate Go module (no dependency on the main library):

```bash
go get github.com/meszmate/xmpp-go/crypto/omemo
```

### Create the Client with Plugins

Register the OMEMO and PubSub plugins when creating the client via `xmpp.WithPlugins()`:

```go
import (
    xmpp "github.com/meszmate/xmpp-go"
    "github.com/meszmate/xmpp-go/crypto/omemo"
    "github.com/meszmate/xmpp-go/jid"
    omemoplugin "github.com/meszmate/xmpp-go/plugins/omemo"
    "github.com/meszmate/xmpp-go/plugins/pubsub"
)

// Your device ID -- generate once, persist, reuse across restarts.
var myDeviceID uint32 = 12345

addr := jid.MustParse("alice@example.com")
client, _ := xmpp.NewClient(addr, "password",
    xmpp.WithPlugins(
        omemoplugin.New(myDeviceID),
        pubsub.New(),
    ),
)
client.Connect(ctx) // plugins are initialized automatically

// Retrieve the OMEMO plugin by name
p, _ := client.Plugin("omemo")
omemoPlugin := p.(*omemoplugin.Plugin)

// Client-side crypto store for private keys and sessions.
// Use omemo.NewMemoryStore(myDeviceID) for testing.
// For production, implement omemo.Store backed by a local database.
store := omemo.NewMemoryStore(myDeviceID)
manager := omemo.NewManager(store)
```

### Generate and Publish Your Bundle

```go
import (
    "encoding/base64"
    "encoding/xml"

    "github.com/meszmate/xmpp-go/crypto/omemo"
    omemoplugin "github.com/meszmate/xmpp-go/plugins/omemo"
)

// Generate key material (private keys stay in the local store)
bundle, _ := manager.GenerateBundle(25)

// Publish device list to PEP
iq := omemoplugin.PublishDeviceListIQ(myDeviceID)
client.Send(ctx, &iq.IQ)

// Publish bundle (public keys only) to PEP
bundleXML, _ := xml.Marshal(&omemoplugin.Bundle{
    SPK:     omemoplugin.SPK{ID: bundle.SignedPreKeyID, Value: base64.StdEncoding.EncodeToString(bundle.SignedPreKey)},
    SPKS:    base64.StdEncoding.EncodeToString(bundle.SignedPreKeySignature),
    IK:      base64.StdEncoding.EncodeToString(bundle.IdentityKey),
    Prekeys: toXMLPreKeys(bundle.PreKeys),
})
iq2 := omemoplugin.PublishBundleIQ(myDeviceID, bundleXML)
client.Send(ctx, &iq2.IQ)
```

### Fetch a Contact's Devices and Bundles

```go
// Fetch device list from server
bob := jid.MustParse("bob@example.com")
iq := omemoplugin.FetchDeviceListIQ(bob)
client.Send(ctx, &iq.IQ)
// Parse the IQ result -> omemoplugin.DeviceList -> list of device IDs

// Cache devices locally via the plugin
omemoPlugin.SetDevices("bob@example.com", devices)

// Fetch all bundles for Bob's devices
iq2 := omemoplugin.FetchBundlesIQ(bob)
client.Send(ctx, &iq2.IQ)
// Parse the IQ result -> []omemoplugin.Bundle, one per device

for _, dev := range omemoPlugin.GetDevices("bob@example.com") {
    // Convert XML bundle to crypto bundle and process it
    cryptoBundle := parseBundleToCrypto(xmlBundle)
    addr := omemo.Address{JID: "bob@example.com", DeviceID: dev.ID}
    manager.ProcessBundle(addr, cryptoBundle)
}
```

### Encrypt and Send a Message

```go
// Build recipient list from cached devices
bobDevices := omemoPlugin.GetDevices("bob@example.com")
recipients := make([]omemo.Address, len(bobDevices))
for i, dev := range bobDevices {
    recipients[i] = omemo.Address{JID: "bob@example.com", DeviceID: dev.ID}
}

// Encrypt
encMsg, err := manager.Encrypt([]byte("Hello Bob!"), recipients...)
if err != nil {
    log.Fatal(err)
}

// Convert to XML
encrypted := cryptoToXMLEncrypted(encMsg)

// Send as XMPP message
msg := stanza.NewMessage(stanza.MessageChat)
msg.To = jid.MustParse("bob@example.com")
msg.Extensions = append(msg.Extensions, marshalToExtension(encrypted))
msg.Extensions = append(msg.Extensions, marshalToExtension(omemoplugin.NewEME()))
client.Send(ctx, msg)
```

### Decrypt an Incoming Message

```go
// When you receive a <message> with an <encrypted xmlns='urn:xmpp:omemo:2'> child:

func handleEncryptedMessage(
    manager *omemo.Manager,
    myDeviceID uint32,
    senderJID string,
    enc *omemoplugin.Encrypted,
) ([]byte, error) {
    // Convert XML to crypto types
    msg := &omemo.EncryptedMessage{
        SenderDeviceID: enc.Header.SID,
        IV:             mustBase64Decode(enc.Header.IV),
    }
    if enc.Payload != nil {
        msg.Payload = mustBase64Decode(enc.Payload.Value)
    }
    for _, k := range enc.Header.Keys {
        msg.Keys = append(msg.Keys, omemo.MessageKey{
            DeviceID: k.RID,
            Data:     mustBase64Decode(k.Value),
            IsPreKey: k.Prekey,
        })
    }

    senderAddr := omemo.Address{JID: senderJID, DeviceID: enc.Header.SID}

    // Check if this is a pre-key message (first message, session setup)
    var isPreKey bool
    for _, k := range msg.Keys {
        if k.DeviceID == myDeviceID {
            isPreKey = k.IsPreKey
            break
        }
    }

    if isPreKey {
        // First message from this device.
        // The pre-key message embeds X3DH key exchange data:
        // sender's identity key, ephemeral key, used pre-key ID, signed pre-key ID.
        // Extract these from the key exchange portion of the message.
        return manager.DecryptPreKeyMessage(
            senderAddr,
            senderIdentityKey,  // Ed25519 public key from key exchange data
            ephemeralPubKey,    // X25519 public key from key exchange data
            usedPreKeyID,       // *uint32, may be nil if no OPK was used
            signedPreKeyID,     // uint32
            msg,
        )
    }

    // Subsequent message -- existing session
    return manager.Decrypt(senderAddr, msg)
}
```

## Converting Between XML and Crypto Types

The `plugins/omemo` package defines XMPP XML types (base64-encoded strings). The `crypto/omemo` package works with raw byte slices. You convert between them:

```go
import (
    "encoding/base64"

    "github.com/meszmate/xmpp-go/crypto/omemo"
    omemoplugin "github.com/meszmate/xmpp-go/plugins/omemo"
)

// XML bundle -> crypto bundle
func parseBundleToCrypto(b *omemoplugin.Bundle) *omemo.Bundle {
    ik, _ := base64.StdEncoding.DecodeString(b.IK)
    spk, _ := base64.StdEncoding.DecodeString(b.SPK.Value)
    spks, _ := base64.StdEncoding.DecodeString(b.SPKS)

    preKeys := make([]omemo.BundlePreKey, len(b.Prekeys))
    for i, pk := range b.Prekeys {
        pub, _ := base64.StdEncoding.DecodeString(pk.Value)
        preKeys[i] = omemo.BundlePreKey{ID: pk.ID, PublicKey: pub}
    }

    return &omemo.Bundle{
        IdentityKey:           ik,
        SignedPreKey:          spk,
        SignedPreKeyID:        b.SPK.ID,
        SignedPreKeySignature: spks,
        PreKeys:               preKeys,
    }
}

// Crypto EncryptedMessage -> XML Encrypted element
func cryptoToXMLEncrypted(enc *omemo.EncryptedMessage) *omemoplugin.Encrypted {
    keys := make([]omemoplugin.Key, len(enc.Keys))
    for i, k := range enc.Keys {
        keys[i] = omemoplugin.Key{
            RID:    k.DeviceID,
            Prekey: k.IsPreKey,
            Value:  base64.StdEncoding.EncodeToString(k.Data),
        }
    }
    result := &omemoplugin.Encrypted{
        Header: omemoplugin.Header{
            SID:  enc.SenderDeviceID,
            Keys: keys,
            IV:   base64.StdEncoding.EncodeToString(enc.IV),
        },
    }
    if len(enc.Payload) > 0 {
        result.Payload = &omemoplugin.Payload{
            Value: base64.StdEncoding.EncodeToString(enc.Payload),
        }
    }
    return result
}

// Crypto pre-keys -> XML pre-keys
func toXMLPreKeys(pks []omemo.BundlePreKey) []omemoplugin.Prekey {
    result := make([]omemoplugin.Prekey, len(pks))
    for i, pk := range pks {
        result[i] = omemoplugin.Prekey{
            ID:    pk.ID,
            Value: base64.StdEncoding.EncodeToString(pk.PublicKey),
        }
    }
    return result
}
```

## Implementing a Persistent Client Store

The `omemo.MemoryStore` loses all state when the process exits. For a real client, implement the `omemo.Store` interface backed by a local database:

```go
type Store interface {
    // Your identity -- generated once, reused forever
    GetIdentityKeyPair() (*IdentityKeyPair, error)
    SaveIdentityKeyPair(ikp *IdentityKeyPair) error
    GetLocalDeviceID() (uint32, error)

    // Trust management for contacts
    GetRemoteIdentity(addr Address) (ed25519.PublicKey, error)
    SaveRemoteIdentity(addr Address, key ed25519.PublicKey) error
    IsTrusted(addr Address, key ed25519.PublicKey) (bool, error)

    // One-time pre-keys (consumed when someone messages you)
    GetPreKey(id uint32) (*PreKeyRecord, error)
    SavePreKey(record *PreKeyRecord) error
    RemovePreKey(id uint32) error

    // Signed pre-key (rotated periodically, e.g. weekly)
    GetSignedPreKey(id uint32) (*SignedPreKeyRecord, error)
    SaveSignedPreKey(record *SignedPreKeyRecord) error

    // Double Ratchet sessions (one per remote device)
    GetSession(addr Address) ([]byte, error)
    SaveSession(addr Address, data []byte) error
    ContainsSession(addr Address) (bool, error)
}
```

This is separate from the server's `storage.PubSubStore`. The server stores public PEP data; the client stores private cryptographic state. They never overlap.

## The Complete OMEMO Flow

### One-Time Setup (per device)

1. Generate a device ID (random `uint32`).
2. Call `manager.GenerateBundle(25)` -- this creates an Ed25519 identity key pair, a signed X25519 pre-key (signed with the identity key), and 25 one-time X25519 pre-keys. All private keys are stored in the local `Store`.
3. Publish the device list to the server via a PubSub IQ set to the PEP node. The server saves it via `PubSubStore.UpsertItem()`.
4. Publish the bundle (public keys only) to the server the same way.

### Sending a Message

1. **Alice** fetches Bob's device list from the server (PubSub IQ get, served by `PubSubStore.GetItems()`).
2. **Alice** fetches Bob's bundle for each device (PubSub IQ get per device, served by `PubSubStore.GetItem()`).
3. **Alice** calls `manager.ProcessBundle()` for each device, then `manager.Encrypt(plaintext, recipients...)`. On first contact this performs X3DH key agreement, then Double Ratchet encrypts the key material per device, and AES-256-GCM encrypts the payload.
4. **Alice** sends a `<message>` containing the `<encrypted>` element. The server routes it to Bob.
5. **Bob** parses the `<encrypted>` XML and calls `manager.DecryptPreKeyMessage()` (first message, sets up session via X3DH respond) or `manager.Decrypt()` (existing session). This performs Double Ratchet decryption followed by AES-256-GCM decryption to recover the plaintext.

### Ongoing

- **PEP notifications**: The server pushes device list changes to subscribed contacts. Update your cache when you receive them.
- **Pre-key replenishment**: After one-time pre-keys are consumed, generate and publish new ones.
- **Signed pre-key rotation**: Rotate periodically (e.g. weekly) and publish the new one.
- **Multi-device**: Encrypt for all of a recipient's devices, including your own other devices.

## Crypto Module API Reference

```go
// Manager is the high-level API
manager := omemo.NewManager(store)

// Generate bundle (stores private keys locally, returns public bundle)
bundle, err := manager.GenerateBundle(preKeyCount)

// Cache a remote bundle for future session creation
manager.ProcessBundle(addr, bundle)

// Encrypt plaintext for multiple recipient devices
encMsg, err := manager.Encrypt(plaintext, recipients...)

// Decrypt a message from an existing session
plaintext, err := manager.Decrypt(senderAddr, encMsg)

// Decrypt first message from a new device (X3DH + session setup)
plaintext, err := manager.DecryptPreKeyMessage(
    senderAddr, identityKey, ephemeralKey, preKeyID, signedPreKeyID, encMsg,
)
```

See the [crypto/omemo README](../crypto/omemo/README.md) for additional details and the complete Store interface.
