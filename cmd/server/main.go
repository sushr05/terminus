package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sushr05/terminus/internal/config"
	"github.com/sushr05/terminus/middleware"
	"github.com/sushr05/terminus/ratelimit"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load configuration: %v", err)
	}

	var redisClient *redis.Client
	if cfg.Mode == "distributed" {
		redisClient = redis.NewClient(&redis.Options{
			Addr:     cfg.RedisAddr,
			Password: cfg.RedisPassword,
			DB:       cfg.RedisDB,
		})
		defer func() {
			if closeErr := redisClient.Close(); closeErr != nil {
				log.Printf("failed to close redis client: %v", closeErr)
			}
		}()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := redisClient.Ping(ctx).Err(); err != nil {
			log.Fatalf("failed to connect to redis at %s: %v", cfg.RedisAddr, err)
		}
		log.Printf("connected to redis at %s", cfg.RedisAddr)
	}

	limiter, err := ratelimit.NewLimiter(cfg, redisClient)
	if err != nil {
		log.Fatalf("failed to create limiter instance: %v", err)
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
		"terminus listening on %s using %s rate limiter in %s mode!",
		cfg.ServerAddr,
		cfg.Algorithm,
		cfg.Mode,
	)

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server failed: %v", err)
	}
}
