package main

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"hash"

	"golang.org/x/crypto/pbkdf2"
)

func hashPasswordSCRAMSHA256(password string, iterations int) (salt string, iters int, storedKey string, serverKey string, err error) {
	saltBytes := make([]byte, 16)
	if _, err = rand.Read(saltBytes); err != nil {
		return
	}

	iters = iterations
	saltedPwd := pbkdf2.Key([]byte(password), saltBytes, iters, sha256.Size, sha256.New)

	clientKey := scramHMAC(sha256.New, saltedPwd, []byte("Client Key"))
	sk := sha256Sum(clientKey)
	srvKey := scramHMAC(sha256.New, saltedPwd, []byte("Server Key"))

	salt = base64.StdEncoding.EncodeToString(saltBytes)
	storedKey = base64.StdEncoding.EncodeToString(sk)
	serverKey = base64.StdEncoding.EncodeToString(srvKey)
	return
}

func scramHMAC(h func() hash.Hash, key, data []byte) []byte {
	mac := hmac.New(h, key)
	mac.Write(data)
	return mac.Sum(nil)
}

func sha256Sum(data []byte) []byte {
	h := sha256.Sum256(data)
	return h[:]
}
