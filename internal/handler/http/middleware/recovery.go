package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/fransalwan/secure-payment-gateway/pkg/response"
)

func RecovererMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rvr := recover(); rvr != nil {
				slog.Error("panic recovered in HTTP request",
					"error", rvr,
					"stack", string(debug.Stack()),
					"path", r.URL.Path,
				)
				response.InternalServerError(w, "internal server error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}
