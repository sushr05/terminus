package ratelimit

import (
	"context"
	"errors"
	"testing"
)

type mockLimiter struct {
	allowed bool
	err     error
	calls   int
}

func (m *mockLimiter) Allow(ctx context.Context, clientID string) (bool, error) {
	m.calls++
	return m.allowed, m.err
}

func TestNewFallbackLimiterValidation(t *testing.T) {
	mock := &mockLimiter{allowed: true}

	if _, err := NewFallbackLimiter(nil, mock); err == nil {
		t.Fatal("expected error when primary is nil")
	}

	if _, err := NewFallbackLimiter(mock, nil); err == nil {
		t.Fatal("expected error when fallback is nil")
	}

	fb, err := NewFallbackLimiter(mock, mock)
	if err != nil || fb == nil {
		t.Fatalf("unexpected error creating FallbackLimiter: %v", err)
	}
}

func TestFallbackLimiterUsesPrimaryWhenHealthy(t *testing.T) {
	primary := &mockLimiter{allowed: true, err: nil}
	fallback := &mockLimiter{allowed: false, err: nil}

	limiter, err := NewFallbackLimiter(primary, fallback)
	if err != nil {
		t.Fatalf("failed to create fallback limiter: %v", err)
	}

	allowed, err := limiter.Allow(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !allowed {
		t.Fatal("expected request to be allowed by primary")
	}

	if primary.calls != 1 {
		t.Fatalf("expected primary to be called once, got %d", primary.calls)
	}
	if fallback.calls != 0 {
		t.Fatalf("expected fallback NOT to be called, got %d", fallback.calls)
	}
}

func TestFallbackLimiterDegradesToFallbackOnPrimaryError(t *testing.T) {
	primary := &mockLimiter{allowed: false, err: errors.New("redis connection refused")}
	fallback := &mockLimiter{allowed: true, err: nil}

	limiter, err := NewFallbackLimiter(primary, fallback)
	if err != nil {
		t.Fatalf("failed to create fallback limiter: %v", err)
	}

	allowed, err := limiter.Allow(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !allowed {
		t.Fatal("expected request to be allowed by fallback")
	}

	if primary.calls != 1 {
		t.Fatalf("expected primary to be called, got %d", primary.calls)
	}
	if fallback.calls != 1 {
		t.Fatalf("expected fallback to be called once, got %d", fallback.calls)
	}
}

func TestFallbackLimiterSwitchesBackToPrimaryWhenRecovered(t *testing.T) {
	primary := &mockLimiter{allowed: true, err: errors.New("redis timeout")}
	fallback := &mockLimiter{allowed: true, err: nil}

	limiter, err := NewFallbackLimiter(primary, fallback)
	if err != nil {
		t.Fatalf("failed to create fallback limiter: %v", err)
	}

	// First call: primary errors -> fallback used
	allowed, err := limiter.Allow(context.Background(), "user-1")
	if err != nil || !allowed {
		t.Fatalf("expected allowed through fallback")
	}
	if fallback.calls != 1 {
		t.Fatalf("expected fallback called once, got %d", fallback.calls)
	}

	// Primary recovers
	primary.err = nil
	primary.allowed = false // test rejection from primary

	// Second call: primary is healthy -> primary decision returned directly
	allowed, err = limiter.Allow(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if allowed {
		t.Fatal("expected rejection from recovered primary")
	}

	if primary.calls != 2 {
		t.Fatalf("expected primary called twice, got %d", primary.calls)
	}
	if fallback.calls != 1 {
		t.Fatalf("expected fallback was NOT called when primary recovered, got %d", fallback.calls)
	}
}

