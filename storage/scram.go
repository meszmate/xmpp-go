package storage

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"hash"

	"golang.org/x/crypto/pbkdf2"
)

// HashPasswordSCRAMSHA256 derives SCRAM-SHA-256 credentials from a plaintext
// password. It returns base64-encoded salt, iteration count, stored key, and
// server key suitable for storing in a User record.
func HashPasswordSCRAMSHA256(password string, iterations int) (salt string, iters int, storedKey string, serverKey string, err error) {
	saltBytes := make([]byte, 16)
	if _, err = rand.Read(saltBytes); err != nil {
		return
	}

	iters = iterations
	saltedPwd := pbkdf2.Key([]byte(password), saltBytes, iters, sha256.Size, sha256.New)

	clientKey := hmacSHA256(saltedPwd, []byte("Client Key"))
	sk := sha256Hash(clientKey)
	srvKey := hmacSHA256(saltedPwd, []byte("Server Key"))

	salt = base64.StdEncoding.EncodeToString(saltBytes)
	storedKey = base64.StdEncoding.EncodeToString(sk)
	serverKey = base64.StdEncoding.EncodeToString(srvKey)
	return
}

func hmacSHA256(key, data []byte) []byte {
	return hmacHash(sha256.New, key, data)
}

func hmacHash(h func() hash.Hash, key, data []byte) []byte {
	mac := hmac.New(h, key)
	mac.Write(data)
	return mac.Sum(nil)
}

func sha256Hash(data []byte) []byte {
	h := sha256.Sum256(data)
	return h[:]
}
