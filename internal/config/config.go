package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	ServerAddr string
	Algorithm  string

	FixedWindowLimit    int
	FixedWindowDuration time.Duration

	SlidingLogLimit  int
	SlidingLogWindow time.Duration

	SlidingCounterLimit  int
	SlidingCounterWindow time.Duration

	TokenBucketCapacity   int
	TokenBucketRefillRate float64

	LeakyBucketCapacity int
	LeakyBucketLeakRate float64
}

func Load() (Config, error) {
	cfg := Config{
		ServerAddr: getEnv("SERVER_ADDR", ":8081"),
		Algorithm:  getEnv("RATE_LIMIT_ALGORITHM", "token_bucket"),

		FixedWindowLimit:    10,
		FixedWindowDuration: time.Minute,

		SlidingLogLimit:  10,
		SlidingLogWindow: time.Minute,

		SlidingCounterLimit:  10,
		SlidingCounterWindow: time.Minute,

		TokenBucketCapacity:   5,
		TokenBucketRefillRate: 1,

		LeakyBucketCapacity: 5,
		LeakyBucketLeakRate: 1,
	}

	var err error

	if cfg.FixedWindowLimit, err = getEnvInt(
		"FIXED_WINDOW_LIMIT", cfg.FixedWindowLimit,
	); err != nil {
		return Config{}, err
	}

	if cfg.FixedWindowDuration, err = getEnvDuration(
		"FIXED_WINDOW_DURATION", cfg.FixedWindowDuration,
	); err != nil {
		return Config{}, err
	}

	if cfg.SlidingLogLimit, err = getEnvInt(
		"SLIDING_LOG_LIMIT", cfg.SlidingLogLimit,
	); err != nil {
		return Config{}, err
	}

	if cfg.SlidingLogWindow, err = getEnvDuration(
		"SLIDING_LOG_WINDOW", cfg.SlidingLogWindow,
	); err != nil {
		return Config{}, err
	}

	if cfg.SlidingCounterLimit, err = getEnvInt(
		"SLIDING_COUNTER_LIMIT", cfg.SlidingCounterLimit,
	); err != nil {
		return Config{}, err
	}

	if cfg.SlidingCounterWindow, err = getEnvDuration(
		"SLIDING_COUNTER_WINDOW", cfg.SlidingCounterWindow,
	); err != nil {
		return Config{}, err
	}

	if cfg.TokenBucketCapacity, err = getEnvInt(
		"TOKEN_BUCKET_CAPACITY", cfg.TokenBucketCapacity,
	); err != nil {
		return Config{}, err
	}

	if cfg.TokenBucketRefillRate, err = getEnvFloat(
		"TOKEN_BUCKET_REFILL_RATE", cfg.TokenBucketRefillRate,
	); err != nil {
		return Config{}, err
	}

	if cfg.LeakyBucketCapacity, err = getEnvInt(
		"LEAKY_BUCKET_CAPACITY", cfg.LeakyBucketCapacity,
	); err != nil {
		return Config{}, err
	}

	if cfg.LeakyBucketLeakRate, err = getEnvFloat(
		"LEAKY_BUCKET_LEAK_RATE", cfg.LeakyBucketLeakRate,
	); err != nil {
		return Config{}, err
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func (c Config) Validate() error {
	if c.ServerAddr == "" {
		return fmt.Errorf("SERVER_ADDR cannot be empty")
	}

	switch c.Algorithm {
	case "fixed_window":
		if c.FixedWindowLimit <= 0 {
			return fmt.Errorf("FIXED_WINDOW_LIMIT must be positive")
		}
		if c.FixedWindowDuration <= 0 {
			return fmt.Errorf("FIXED_WINDOW_DURATION must be positive")
		}

	case "sliding_log":
		if c.SlidingLogLimit <= 0 {
			return fmt.Errorf("SLIDING_LOG_LIMIT must be positive")
		}
		if c.SlidingLogWindow <= 0 {
			return fmt.Errorf("SLIDING_LOG_WINDOW must be positive")
		}

	case "sliding_counter":
		if c.SlidingCounterLimit <= 0 {
			return fmt.Errorf("SLIDING_COUNTER_LIMIT must be positive")
		}
		if c.SlidingCounterWindow <= 0 {
			return fmt.Errorf("SLIDING_COUNTER_WINDOW must be positive")
		}

	case "token_bucket":
		if c.TokenBucketCapacity <= 0 {
			return fmt.Errorf("TOKEN_BUCKET_CAPACITY must be positive")
		}
		if c.TokenBucketRefillRate <= 0 {
			return fmt.Errorf("TOKEN_BUCKET_REFILL_RATE must be positive")
		}

	case "leaky_bucket":
		if c.LeakyBucketCapacity <= 0 {
			return fmt.Errorf("LEAKY_BUCKET_CAPACITY must be positive")
		}
		if c.LeakyBucketLeakRate <= 0 {
			return fmt.Errorf("LEAKY_BUCKET_LEAK_RATE must be positive")
		}

	default:
		return fmt.Errorf("unsupported rate limit algorithm: %q", c.Algorithm)
	}

	return nil
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) (int, error) {
	value, exists := os.LookupEnv(key)
	if !exists {
		return fallback, nil
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer: %w", key, err)
	}

	return parsed, nil
}

func getEnvFloat(key string, fallback float64) (float64, error) {
	value, exists := os.LookupEnv(key)
	if !exists {
		return fallback, nil
	}

	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, fmt.Errorf("%s must be a number: %w", key, err)
	}

	return parsed, nil
}

func getEnvDuration(key string, fallback time.Duration) (time.Duration, error) {
	value, exists := os.LookupEnv(key)
	if !exists {
		return fallback, nil
	}

	parsed, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf(
			"%s must be a valid duration (e.g. 30s, 2m): %w",
			key, err,
		)
	}

	return parsed, nil
}

//right now its a flat eager approach where we send all to factory and it chooses what it needs. but in future can make it lazy loaded ie we load the config only when we need it.
