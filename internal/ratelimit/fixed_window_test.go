package ratelimit

import (
	"testing"
	"time"
)

func TestFixedWindow(t *testing.T) {
	limiter := NewFixedWindow(2, time.Second)
	now := time.Date(2026, 5, 21, 1, 0, 0, 0, time.UTC)
	if !limiter.Allow("agent:a", now) {
		t.Fatal("first request should pass")
	}
	if !limiter.Allow("agent:a", now.Add(100*time.Millisecond)) {
		t.Fatal("second request should pass")
	}
	if limiter.Allow("agent:a", now.Add(200*time.Millisecond)) {
		t.Fatal("third request in same window should be denied")
	}
	if !limiter.Allow("agent:a", now.Add(time.Second)) {
		t.Fatal("next window should pass")
	}
}
