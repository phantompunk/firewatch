package middleware

import (
	"net"
	"net/http"
	"strings"
	"sync"

	"golang.org/x/time/rate"
)

type ipLimiter struct {
	mu       sync.Mutex
	limiters map[string]*rate.Limiter
	rate     rate.Limit
	burst    int
}

func newIPLimiter(r rate.Limit, burst int) *ipLimiter {
	return &ipLimiter{
		limiters: make(map[string]*rate.Limiter),
		rate:     r,
		burst:    burst,
	}
}

func (ipl *ipLimiter) get(ip string) *rate.Limiter {
	ipl.mu.Lock()
	defer ipl.mu.Unlock()

	l, ok := ipl.limiters[ip]
	if !ok {
		l = rate.NewLimiter(ipl.rate, ipl.burst)
		ipl.limiters[ip] = l
	}
	return l
}

// clientIP returns the IP address to use for rate limiting.
//
// The raw TCP connection address (r.RemoteAddr) is always used as the default.
// Forwarded headers (X-Real-IP, X-Forwarded-For) are only trusted when
// trustedProxy is non-nil and the connecting address falls within that CIDR.
// This prevents clients from spoofing their IP to bypass rate limiting.
func clientIP(r *http.Request, trustedProxy *net.IPNet) string {
	connHost, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		// r.RemoteAddr has no port (shouldn't happen with net/http, but be safe)
		connHost = r.RemoteAddr
	}

	if trustedProxy != nil {
		connIP := net.ParseIP(connHost)
		if connIP != nil && trustedProxy.Contains(connIP) {
			// Connection is from the trusted proxy — trust forwarded headers.
			if xri := r.Header.Get("X-Real-IP"); xri != "" {
				if ip := net.ParseIP(strings.TrimSpace(xri)); ip != nil {
					return ip.String()
				}
			}
			if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
				// X-Forwarded-For may be a comma-separated list; leftmost is the client.
				first, _, _ := strings.Cut(xff, ",")
				if ip := net.ParseIP(strings.TrimSpace(first)); ip != nil {
					return ip.String()
				}
			}
		}
	}

	return connHost
}

// RateLimit returns middleware that limits requests per client IP.
// trustedProxy may be nil; when non-nil, forwarded IP headers are trusted only
// from connections originating within that CIDR.
func RateLimit(r rate.Limit, burst int, trustedProxy *net.IPNet) func(http.Handler) http.Handler {
	il := newIPLimiter(r, burst)
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ip := clientIP(req, trustedProxy)
			if !il.get(ip).Allow() {
				http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
				return
			}
			h.ServeHTTP(w, req)
		})
	}
}
