package middleware

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"net/http"
)

const contextKeyNonce contextKey = "nonce"

// NonceFromContext returns the CSP nonce for the current request.
func NonceFromContext(ctx context.Context) string {
	v, _ := ctx.Value(contextKeyNonce).(string)
	return v
}

// CSP generates a per-request nonce and sets the Content-Security-Policy header.
// Alpine.js requires 'unsafe-eval' because it uses new Function() for expression
// evaluation internally. Nonces still protect against injected script tags.
func CSP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b := make([]byte, 16)
		_, _ = rand.Read(b)
		nonce := base64.RawURLEncoding.EncodeToString(b)

		csp := "default-src 'self'; " +
			"script-src 'self' 'nonce-" + nonce + "' 'unsafe-eval'; " +
			"style-src 'self'; " +
			"img-src 'self'; " +
			"font-src 'self'; " +
			"connect-src 'self'; " +
			"frame-ancestors 'none'; " +
			"form-action 'self'; " +
			"base-uri 'self'; " +
			"object-src 'none'"
		w.Header().Set("Content-Security-Policy", csp)

		ctx := context.WithValue(r.Context(), contextKeyNonce, nonce)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
