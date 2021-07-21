package main

import (
	"crypto/sha256"
	"encoding/hex"
)

func hashPassword(password string) string {
	hash := sha256.Sum256([]byte(password))
	return hex.EncodeToString(hash[:])
}

func randomHash() string {
	hash := sha256.Sum256([]byte(randomString(64)))
	return hex.EncodeToString(hash[:])
}
