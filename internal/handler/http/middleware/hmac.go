package middleware

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/fransalwan/secure-payment-gateway/internal/config"
	"github.com/fransalwan/secure-payment-gateway/pkg/crypto"
	"github.com/fransalwan/secure-payment-gateway/pkg/response"
)

func HMACAuthMiddleware(cfg *config.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			signature := r.Header.Get("X-Signature")
			timestampStr := r.Header.Get("X-Timestamp")

			if signature == "" || timestampStr == "" {
				response.Unauthorized(w, "missing security headers", "X-Signature and X-Timestamp headers are required")
				return
			}

			timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
			if err != nil {
				response.BadRequest(w, "invalid timestamp format", "INVALID_TIMESTAMP", "X-Timestamp must be a Unix timestamp in seconds")
				return
			}

			if !crypto.VerifyTimestamp(timestamp, cfg.HMACTimestampTolerance) {
				response.Unauthorized(w, "request timestamp expired or out of bounds", "request expired to prevent replay attacks")
				return
			}

			bodyBytes, err := io.ReadAll(r.Body)
			if err != nil {
				response.BadRequest(w, "failed to read request body", "BODY_READ_ERROR", err.Error())
				return
			}

			r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

			messageToVerify := []byte(fmt.Sprintf("%d.%s", timestamp, string(bodyBytes)))

			if !crypto.VerifyHMACSignature(messageToVerify, signature, cfg.HMACSecretKey) {
				response.Unauthorized(w, "invalid request signature", "HMAC signature mismatch")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
