package debug

import (
	"crypto/sha256"
	"encoding/hex"
)

func CheckSum(file string) string {
	sum := sha256.Sum256([]byte(file))
	return hex.EncodeToString(sum[:])
}
