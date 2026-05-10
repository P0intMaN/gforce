package middleware

import (
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// RateLimit returns middleware that enforces a per-IP token-bucket rate limit.
// reqPerMin is the sustained request rate allowed per IP address.
// A burst of reqPerMin/5 (minimum 10) is allowed to handle short spikes.
func RateLimit(reqPerMin int) func(http.Handler) http.Handler {
	rl := newIPLimiter(rate.Limit(float64(reqPerMin)/60.0), max(reqPerMin/5, 10))
	go rl.cleanupLoop()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := realIP(r)
			if !rl.allow(ip) {
				w.Header().Set("X-RateLimit-Limit", "100")
				w.Header().Set("Retry-After", "60")
				http.Error(w, `{"error":"rate limit exceeded"}`, http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

type ipEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type ipLimiter struct {
	mu    sync.Mutex
	ips   map[string]*ipEntry
	r     rate.Limit
	burst int
}

func newIPLimiter(r rate.Limit, burst int) *ipLimiter {
	return &ipLimiter{ips: make(map[string]*ipEntry), r: r, burst: burst}
}

func (l *ipLimiter) allow(ip string) bool {
	l.mu.Lock()
	entry, ok := l.ips[ip]
	if !ok {
		entry = &ipEntry{limiter: rate.NewLimiter(l.r, l.burst)}
		l.ips[ip] = entry
	}
	entry.lastSeen = time.Now()
	allowed := entry.limiter.Allow()
	l.mu.Unlock()
	return allowed
}

// cleanupLoop removes IP entries not seen in the last 5 minutes to prevent unbounded growth.
func (l *ipLimiter) cleanupLoop() {
	t := time.NewTicker(5 * time.Minute)
	defer t.Stop()
	for range t.C {
		l.mu.Lock()
		for ip, entry := range l.ips {
			if time.Since(entry.lastSeen) > 5*time.Minute {
				delete(l.ips, ip)
			}
		}
		l.mu.Unlock()
	}
}

// realIP extracts the client IP from the request, preferring X-Real-IP over RemoteAddr.
func realIP(r *http.Request) string {
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		return ip
	}
	return r.RemoteAddr
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
