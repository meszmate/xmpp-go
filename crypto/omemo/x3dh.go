package omemo

import (
	"crypto/ecdh"
	"crypto/ed25519"
)

var (
	x3dhSalt = make([]byte, 32) // 32 zero bytes
	x3dhPad  []byte             // 32 0xFF bytes
)

func init() {
	x3dhPad = make([]byte, 32)
	for i := range x3dhPad {
		x3dhPad[i] = 0xFF
	}
}

// X3DHResult holds the result of an X3DH key agreement.
type X3DHResult struct {
	SharedSecret    []byte
	EphemeralPubKey []byte // X25519 public key used by initiator
	UsedPreKeyID    *uint32
}

// X3DHInitiate performs the X3DH key agreement as the initiator (Alice).
func X3DHInitiate(localIdentity *IdentityKeyPair, remoteBundle *Bundle) (*X3DHResult, error) {
	// 1. Verify SPK signature
	if !ed25519.Verify(remoteBundle.IdentityKey, remoteBundle.SignedPreKey, remoteBundle.SignedPreKeySignature) {
		return nil, ErrInvalidSignature
	}

	// 2. Generate ephemeral X25519 key pair
	ephemeralKey, err := GenerateX25519KeyPair()
	if err != nil {
		return nil, err
	}

	// 3. Convert identity keys to X25519
	localX25519, err := Ed25519PrivateKeyToX25519(localIdentity.PrivateKey)
	if err != nil {
		return nil, err
	}

	remoteX25519Pub, err := Ed25519PublicKeyToX25519(remoteBundle.IdentityKey)
	if err != nil {
		return nil, err
	}

	// 4. DH calculations
	// DH1 = DH(IK_A_x25519, SPK_B)
	dh1, err := x25519DH(localX25519, remoteBundle.SignedPreKey)
	if err != nil {
		return nil, err
	}

	// DH2 = DH(EK_A, IK_B_x25519)
	dh2, err := x25519DH(ephemeralKey, remoteX25519Pub)
	if err != nil {
		return nil, err
	}

	// DH3 = DH(EK_A, SPK_B)
	dh3, err := x25519DH(ephemeralKey, remoteBundle.SignedPreKey)
	if err != nil {
		return nil, err
	}

	// Concatenate: 0xFF*32 || DH1 || DH2 || DH3
	ikm := make([]byte, 0, 32+32*3+32)
	ikm = append(ikm, x3dhPad...)
	ikm = append(ikm, dh1...)
	ikm = append(ikm, dh2...)
	ikm = append(ikm, dh3...)

	var usedPreKeyID *uint32

	// DH4 = DH(EK_A, OPK_B) if available
	if len(remoteBundle.PreKeys) > 0 {
		opk := remoteBundle.PreKeys[0]
		dh4, err := x25519DH(ephemeralKey, opk.PublicKey)
		if err != nil {
			return nil, err
		}
		ikm = append(ikm, dh4...)
		usedPreKeyID = &opk.ID
	}

	// 5. SK = HKDF(salt=0x00*32, ikm, info="OMEMO X3DH")
	sk, err := hkdfSHA256(x3dhSalt, ikm, []byte("OMEMO X3DH"), 32)
	if err != nil {
		return nil, err
	}

	return &X3DHResult{
		SharedSecret:    sk,
		EphemeralPubKey: ephemeralKey.PublicKey().Bytes(),
		UsedPreKeyID:    usedPreKeyID,
	}, nil
}

// X3DHRespond performs the X3DH key agreement as the responder (Bob).
func X3DHRespond(
	localIdentity *IdentityKeyPair,
	localSPK *ecdh.PrivateKey,
	localOPK *ecdh.PrivateKey,
	remoteIdentityKey ed25519.PublicKey,
	ephemeralPubKey []byte,
) ([]byte, error) {
	// Convert remote identity key to X25519
	remoteX25519Pub, err := Ed25519PublicKeyToX25519(remoteIdentityKey)
	if err != nil {
		return nil, err
	}

	// Convert local identity key to X25519
	localX25519, err := Ed25519PrivateKeyToX25519(localIdentity.PrivateKey)
	if err != nil {
		return nil, err
	}

	// DH1 = DH(SPK_B, IK_A_x25519)
	dh1, err := x25519DH(localSPK, remoteX25519Pub)
	if err != nil {
		return nil, err
	}

	// DH2 = DH(IK_B_x25519, EK_A)
	dh2, err := x25519DH(localX25519, ephemeralPubKey)
	if err != nil {
		return nil, err
	}

	// DH3 = DH(SPK_B, EK_A)
	dh3, err := x25519DH(localSPK, ephemeralPubKey)
	if err != nil {
		return nil, err
	}

	// Concatenate: 0xFF*32 || DH1 || DH2 || DH3
	ikm := make([]byte, 0, 32+32*3+32)
	ikm = append(ikm, x3dhPad...)
	ikm = append(ikm, dh1...)
	ikm = append(ikm, dh2...)
	ikm = append(ikm, dh3...)

	// DH4 = DH(OPK_B, EK_A) if OPK was used
	if localOPK != nil {
		dh4, err := x25519DH(localOPK, ephemeralPubKey)
		if err != nil {
			return nil, err
		}
		ikm = append(ikm, dh4...)
	}

	// SK = HKDF(salt=0x00*32, ikm, info="OMEMO X3DH")
	return hkdfSHA256(x3dhSalt, ikm, []byte("OMEMO X3DH"), 32)
}
