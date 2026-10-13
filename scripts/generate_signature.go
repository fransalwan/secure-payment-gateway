package main

import (
	"fmt"
	"os"
	"time"

	"github.com/fransalwan/secure-payment-gateway/pkg/crypto"
)

func main() {
	secretKey := "spg-secret-hmac-key-for-signatures"
	if len(os.Args) > 1 {
		secretKey = os.Args[1]
	}

	payload := `{"reference_id":"REF-2026-001","source_account_id":"PASTE_SOURCE_UUID_HERE","destination_account_id":"PASTE_DEST_UUID_HERE","amount":50000,"currency":"IDR","description":"Payment for invoice"}`
	now := time.Now().Unix()

	message := fmt.Sprintf("%d.%s", now, payload)
	sig := crypto.GenerateHMACSignature([]byte(message), []byte(secretKey))

	fmt.Println("=== Generated Security Headers ===")
	fmt.Printf("X-Timestamp: %d\n", now)
	fmt.Printf("X-Signature: %s\n", sig)
	fmt.Printf("X-Idempotency-Key: %s\n", fmt.Sprintf("idemp-%d", now))
	fmt.Println("\n=== Sample Payload ===")
	fmt.Println(payload)
}

