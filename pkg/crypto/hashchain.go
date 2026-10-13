package crypto

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// CalculateAuditHash computes SHA-256(previousHash + ":" + payload) for tamper-evident ledger log chaining.
func CalculateAuditHash(previousHash string, payload []byte) string {
	hasher := sha256.New()
	data := fmt.Sprintf("%s:%s", previousHash, string(payload))
	hasher.Write([]byte(data))
	return hex.EncodeToString(hasher.Sum(nil))
}
