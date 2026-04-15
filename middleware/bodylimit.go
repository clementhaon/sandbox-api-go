package middleware

import "net/http"

// DefaultMaxBodySize is the default maximum request body size (1 MB).
const DefaultMaxBodySize = 1 << 20

// MaxBytesMiddleware limits the size of incoming request bodies.
func MaxBytesMiddleware(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			next.ServeHTTP(w, r)
		})
	}
}
