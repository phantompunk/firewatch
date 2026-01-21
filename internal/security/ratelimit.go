package security

import (
	"sync"
	"time"
)

// RateLimiter implements a simple global rate limiter.
// It uses a sliding window approach without tracking individual IPs.
type RateLimiter struct {
	mu           sync.Mutex
	timestamps   []time.Time
	maxPerMinute int
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(maxPerMinute int) *RateLimiter {
	return &RateLimiter{
		timestamps:   make([]time.Time, 0),
		maxPerMinute: maxPerMinute,
	}
}

// Allow checks if a request should be allowed
func (r *RateLimiter) Allow() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-time.Minute)

	// Remove old timestamps
	valid := r.timestamps[:0]
	for _, ts := range r.timestamps {
		if ts.After(cutoff) {
			valid = append(valid, ts)
		}
	}
	r.timestamps = valid

	// Check if under limit
	if len(r.timestamps) >= r.maxPerMinute {
		return false
	}

	// Add new timestamp
	r.timestamps = append(r.timestamps, now)
	return true
}
