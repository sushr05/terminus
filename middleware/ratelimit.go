package middleware

import (
	"net"
	"net/http"

	"github.com/sushr05/terminus/ratelimit"
)

func RateLimitMiddleware(
	limiter ratelimit.RateLimiter,
	next http.Handler,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			host = r.RemoteAddr
		}

		if !limiter.Allow(host) {
			http.Error(
				w,
				"rate limit exceeded",
				http.StatusTooManyRequests,
			)
			return
		}

		next.ServeHTTP(w, r)
	})
}
