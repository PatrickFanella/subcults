package middleware

import "net/http"

// SecurityHeaders returns middleware that sets defense-in-depth security
// headers on every response. These duplicate what the external Caddy sets,
// protecting direct-access scenarios (dev, tests, misconfigured proxy).
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		h.Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains; preload")
		h.Set("Permissions-Policy", "camera=(), microphone=(self), geolocation=(self)")
		h.Set("X-XSS-Protection", "1; mode=block")
		h.Set("Content-Security-Policy",
			"default-src 'self'; "+
				"script-src 'self'; "+
				"style-src 'self' 'unsafe-inline'; "+
				"img-src 'self' https://*.maptiler.com data:; "+
				"connect-src 'self' https://*.maptiler.com wss:; "+
				"font-src 'self'; "+
				"frame-ancestors 'none'; "+
				"report-uri /api/csp-report")
		next.ServeHTTP(w, r)
	})
}
