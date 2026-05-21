package httpserver

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type tokenBucket struct {
	mu       sync.Mutex
	tokens   float64
	last     time.Time
	capacity float64
	refill   float64 // tokens per second
}

func newTokenBucket(capacity int, perMinute int) *tokenBucket {
	return &tokenBucket{
		tokens:   float64(capacity),
		last:     time.Now(),
		capacity: float64(capacity),
		refill:   float64(perMinute) / 60.0,
	}
}

func (b *tokenBucket) allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	now := time.Now()
	elapsed := now.Sub(b.last).Seconds()
	b.last = now
	b.tokens += elapsed * b.refill
	if b.tokens > b.capacity {
		b.tokens = b.capacity
	}
	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

// memoryRateLimiter is an in-memory per-IP token bucket (single replica).
type memoryRateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*tokenBucket
}

func NewMemoryRateLimiter() *memoryRateLimiter {
	return &memoryRateLimiter{buckets: make(map[string]*tokenBucket)}
}

func (rl *memoryRateLimiter) Allow(key string, capacity, perMinute int) bool {
	rl.mu.Lock()
	b, ok := rl.buckets[key]
	if !ok {
		b = newTokenBucket(capacity, perMinute)
		rl.buckets[key] = b
	}
	rl.mu.Unlock()
	return b.allow()
}

func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return strings.TrimSpace(strings.Split(xff, ",")[0])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func withRateLimit(rl RequestLimiter, path string, capacity, perMinute int, next http.Handler) http.Handler {
	if rl == nil {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !rateLimitMatches(path, r.URL.Path, r.Method) {
			next.ServeHTTP(w, r)
			return
		}
		key := clientIP(r) + ":" + path
		if !rl.Allow(key, capacity, perMinute) {
			writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "rate limit exceeded"})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func rateLimitMatches(prefix, path, method string) bool {
	if method != http.MethodPost {
		return false
	}
	if path == prefix {
		return true
	}
	switch prefix {
	case "/repositories":
		return path == "/repositories" || strings.HasSuffix(path, "/reindex")
	case "/ai/chat":
		return path == "/ai/chat"
	}
	return false
}
