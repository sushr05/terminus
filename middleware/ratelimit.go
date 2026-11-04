package middleware

import (
	"log"
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

		allowed, err := limiter.Allow(r.Context(), host)
		if err != nil {
			log.Printf("rate limiter error: %v", err)
			http.Error(
				w,
				"internal server error",
				http.StatusInternalServerError,
			)
			return
		}

		if !allowed {
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
