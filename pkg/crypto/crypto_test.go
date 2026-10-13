package crypto_test

import (
	"crypto/rand"
	"testing"
	"time"

	"github.com/fransalwan/secure-payment-gateway/pkg/crypto"
)

func TestAESGCMEncryptDecrypt(t *testing.T) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatalf("failed to generate random key: %v", err)
	}

	secretPayload := []byte("Identity: 3171012345670001; Card: 4111-2222-3333-4444")

	encryptedHex, err := crypto.AESGCMEncrypt(secretPayload, key)
	if err != nil {
		t.Fatalf("encryption failed: %v", err)
	}

	if len(encryptedHex) == 0 {
		t.Fatal("encrypted string should not be empty")
	}

	decrypted, err := crypto.AESGCMDecrypt(encryptedHex, key)
	if err != nil {
		t.Fatalf("decryption failed: %v", err)
	}

	if string(decrypted) != string(secretPayload) {
		t.Fatalf("expected '%s', got '%s'", string(secretPayload), string(decrypted))
	}
}

func TestAESGCMTamperDetection(t *testing.T) {
	key := make([]byte, 32)
	rand.Read(key)

	secretPayload := []byte("Sensitive Banking Account Data")
	encryptedHex, err := crypto.AESGCMEncrypt(secretPayload, key)
	if err != nil {
		t.Fatalf("encryption failed: %v", err)
	}

	tamperedHex := encryptedHex[:len(encryptedHex)-1] + "0"
	if tamperedHex == encryptedHex {
		tamperedHex = encryptedHex[:len(encryptedHex)-1] + "1"
	}

	_, err = crypto.AESGCMDecrypt(tamperedHex, key)
	if err == nil {
		t.Fatal("expected decryption failure for tampered ciphertext, got nil")
	}
}

func TestHMACSignatureVerification(t *testing.T) {
	secret := []byte("top-secret-hmac-key")
	message := []byte("{\"amount\":1000000,\"destination\":\"acc_12345\"}")

	sig := crypto.GenerateHMACSignature(message, secret)

	if !crypto.VerifyHMACSignature(message, sig, secret) {
		t.Fatal("valid signature failed verification")
	}

	tamperedMessage := []byte("{\"amount\":9999999,\"destination\":\"acc_12345\"}")
	if crypto.VerifyHMACSignature(tamperedMessage, sig, secret) {
		t.Fatal("tampered message should not pass signature verification")
	}

	if crypto.VerifyHMACSignature(message, sig, []byte("wrong-secret")) {
		t.Fatal("verification should fail with wrong secret")
	}
}

func TestTimestampTolerance(t *testing.T) {
	now := time.Now().Unix()

	if !crypto.VerifyTimestamp(now-100, 300) {
		t.Fatal("timestamp within tolerance should be valid")
	}

	if crypto.VerifyTimestamp(now-400, 300) {
		t.Fatal("expired timestamp should be invalid (anti-replay)")
	}
}
