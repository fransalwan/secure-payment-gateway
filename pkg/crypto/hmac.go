package crypto

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"math"
	"time"
)

// GenerateHMACSignature creates a hex-encoded HMAC-SHA256 signature of the payload.
func GenerateHMACSignature(message []byte, secret []byte) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write(message)
	return hex.EncodeToString(mac.Sum(nil))
}

// VerifyHMACSignature verifies if the provided signature matches the message using constant-time comparison.
func VerifyHMACSignature(message []byte, signatureHex string, secret []byte) bool {
	expectedSig := GenerateHMACSignature(message, secret)

	expectedBytes, err := hex.DecodeString(expectedSig)
	if err != nil {
		return false
	}

	actualBytes, err := hex.DecodeString(signatureHex)
	if err != nil {
		return false
	}

	return hmac.Equal(expectedBytes, actualBytes)
}

// VerifyTimestamp checks if the request timestamp is within tolerance window to prevent replay attacks.
func VerifyTimestamp(requestTimestampUnix int64, toleranceSeconds int64) bool {
	now := time.Now().Unix()
	diff := math.Abs(float64(now - requestTimestampUnix))
	return diff <= float64(toleranceSeconds)
}
