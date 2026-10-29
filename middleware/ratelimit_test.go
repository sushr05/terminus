package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type fakeLimiter struct {
	allowed  bool
	called   bool
	clientID string
}

func (f *fakeLimiter) Allow(clientID string) bool {
	f.called = true
	f.clientID = clientID
	return f.allowed
}

func TestRateLimitMiddlewareAllowsRequest(t *testing.T) {
	limiter := &fakeLimiter{allowed: true}

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("handler executed"))
	})

	handler := RateLimitMiddleware(limiter, next)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("got status %d, want %d", rec.Code, http.StatusOK)
	}

	if rec.Body.String() != "handler executed" {
		t.Fatalf("unexpected response body: %q", rec.Body.String())
	}
}

func TestRateLimitMiddlewareRejectsRequest(t *testing.T) {
	limiter := &fakeLimiter{allowed: false}

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler should not execute for a rejected request")
	})

	handler := RateLimitMiddleware(limiter, next)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf(
			"got status %d, want %d",
			rec.Code,
			http.StatusTooManyRequests,
		)
	}

	if !strings.Contains(rec.Body.String(), "rate limit exceeded") {
		t.Fatalf("unexpected response body: %q", rec.Body.String())
	}
}

func TestRateLimitMiddlewareCallsLimiter(t *testing.T) {
	limiter := &fakeLimiter{allowed: true}

	handler := RateLimitMiddleware(
		limiter,
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
	)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if !limiter.called {
		t.Fatal("expected middleware to call limiter.Allow")
	}
}

func TestRateLimitMiddlewarePassesClientIP(t *testing.T) {
	limiter := &fakeLimiter{allowed: true}

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	handler := RateLimitMiddleware(limiter, next)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.168.1.25:54321"

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if limiter.clientID != "192.168.1.25" {
		t.Fatalf("got client ID %q, want %q", limiter.clientID, "192.168.1.25")
	}
}

func TestRateLimitMiddlewareMakesDecisionPerRequest(t *testing.T) {
	limiter := &fakeLimiter{allowed: true}
	handlerCalls := 0

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalls++
	})

	handler := RateLimitMiddleware(limiter, next)

	// First request is allowed.
	req1 := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)

	if handlerCalls != 1 {
		t.Fatalf("after allowed request, handler calls = %d, want 1", handlerCalls)
	}

	// Change the fake limiter's decision for the second request.
	limiter.allowed = false

	req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusTooManyRequests {
		t.Fatalf("got status %d, want %d", rec2.Code, http.StatusTooManyRequests)
	}

	if handlerCalls != 1 {
		t.Fatalf("after rejected request, handler calls = %d, want 1", handlerCalls)
	}
}
