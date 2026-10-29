package main

import (
	"log"
	"net/http"

	"github.com/sushr05/terminus/internal/config"
	"github.com/sushr05/terminus/middleware"
	"github.com/sushr05/terminus/ratelimit"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load configuration:%v", err)
	}

	limiter, err := ratelimit.NewLimiter(cfg)
	if err != nil {
		log.Fatalf("failed to create limiter instance:%v", err)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	apiHandler := http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("API request successful"))
	})

	mux.Handle(
		"GET /api/test",
		middleware.RateLimitMiddleware(limiter, apiHandler),
	)

	server := &http.Server{
		Addr:    cfg.ServerAddr,
		Handler: mux,
	}

	log.Printf(
		"terminus listening on %s using %s rate limiter!",
		cfg.ServerAddr,
		cfg.Algorithm,
	)

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server failed: %v", err)
	}
}
