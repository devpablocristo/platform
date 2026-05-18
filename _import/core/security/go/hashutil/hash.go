package hashutil

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
)

func SHA256Hex(input string) string {
	sum := sha256.Sum256([]byte(input))
	return hex.EncodeToString(sum[:])
}

// SHA256BytesHex calcula SHA-256 de bytes y devuelve el hex.
func SHA256BytesHex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func GenerateAPIKey() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return "psk_" + hex.EncodeToString(buf), nil
}
