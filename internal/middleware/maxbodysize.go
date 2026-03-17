package middleware

import (
	"net/http"
	"strings"
)

// MaxBodySize returns middleware that limits request body size.
// JSON endpoints get jsonLimit bytes; upload paths get uploadLimit bytes.
// Exceeding the limit causes http.MaxBytesReader to return an error
// which surfaces as a 413 Request Entity Too Large.
func MaxBodySize(jsonLimit, uploadLimit int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			limit := jsonLimit
			if strings.HasPrefix(r.URL.Path, "/uploads/") {
				limit = uploadLimit
			}
			r.Body = http.MaxBytesReader(w, r.Body, limit)
			next.ServeHTTP(w, r)
		})
	}
}
