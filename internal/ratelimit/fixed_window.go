package ratelimit

import (
	"sync"
	"time"
)

type FixedWindow struct {
	mu      sync.Mutex
	limit   int
	window  time.Duration
	buckets map[string]bucket
}

type bucket struct {
	start time.Time
	count int
}

func NewFixedWindow(limit int, window time.Duration) *FixedWindow {
	if window <= 0 {
		window = time.Second
	}
	return &FixedWindow{
		limit:   limit,
		window:  window,
		buckets: map[string]bucket{},
	}
}

func (l *FixedWindow) Allow(key string, now time.Time) bool {
	if l == nil || l.limit <= 0 || key == "" {
		return true
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	current := l.buckets[key]
	if current.start.IsZero() || now.Sub(current.start) >= l.window {
		l.buckets[key] = bucket{start: now, count: 1}
		return true
	}
	if current.count >= l.limit {
		return false
	}
	current.count++
	l.buckets[key] = current
	return true
}
