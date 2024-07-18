package utils

import (
	"crypto/sha256"
	"encoding/hex"
)

func Hash(data string) string {
	hash := sha256.New()
	// Write data to the hash object
	hash.Write([]byte(data))
	// Get the resulting hash value
	hashValue := hash.Sum(nil)
	// Convert the hash value to a hexadecimal string
	return hex.EncodeToString(hashValue)
}
