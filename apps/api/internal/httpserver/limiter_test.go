package httpserver

import "testing"

func TestMemoryRateLimiterAllowsBurst(t *testing.T) {
	rl := NewMemoryRateLimiter()
	key := "test:1"
	if !rl.Allow(key, 3, 60) {
		t.Fatal("first request should pass")
	}
	if !rl.Allow(key, 3, 60) {
		t.Fatal("second request should pass")
	}
	if !rl.Allow(key, 3, 60) {
		t.Fatal("third request should pass")
	}
	if rl.Allow(key, 3, 60) {
		t.Fatal("fourth request should be rate limited")
	}
}
