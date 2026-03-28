package httpserver

import (
	"crypto/subtle"
	"net/http"
)

// Middleware wraps an http.Handler.
type Middleware func(http.Handler) http.Handler

// BasicAuth returns middleware that requires HTTP Basic Authentication
// on health endpoints. Use for external-facing deployments.
func BasicAuth(username, password string) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			u, p, ok := r.BasicAuth()
			if !ok ||
				subtle.ConstantTimeCompare([]byte(u), []byte(username)) != 1 ||
				subtle.ConstantTimeCompare([]byte(p), []byte(password)) != 1 {
				w.Header().Set("WWW-Authenticate", `Basic realm="health"`)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
