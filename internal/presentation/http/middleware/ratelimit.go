package middleware

import (
	"net/http"
	"sync"
	"time"
)

// RateLimiter implements a simple token bucket rate limiter
type RateLimiter struct {
	mu       sync.Mutex
	clients  map[string]*clientInfo
	rate     int           // requests per interval
	interval time.Duration // time interval
	burst    int           // max burst size
}

type clientInfo struct {
	tokens    int
	lastCheck time.Time
}

// NewRateLimiter creates a new rate limiter
// rate: number of requests allowed per interval
// interval: time window for rate limiting
// burst: max requests allowed in a burst
func NewRateLimiter(rate int, interval time.Duration, burst int) *RateLimiter {
	rl := &RateLimiter{
		clients:  make(map[string]*clientInfo),
		rate:     rate,
		interval: interval,
		burst:    burst,
	}

	// Clean up old entries periodically
	go rl.cleanup()

	return rl
}

func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(10 * time.Minute)
	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for ip, info := range rl.clients {
			if now.Sub(info.lastCheck) > rl.interval*10 {
				delete(rl.clients, ip)
			}
		}
		rl.mu.Unlock()
	}
}

func (rl *RateLimiter) allow(clientIP string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	client, exists := rl.clients[clientIP]

	if !exists {
		rl.clients[clientIP] = &clientInfo{
			tokens:    rl.burst - 1,
			lastCheck: now,
		}
		return true
	}

	// Calculate tokens to add based on elapsed time
	elapsed := now.Sub(client.lastCheck)
	tokensToAdd := int(elapsed.Seconds() / rl.interval.Seconds() * float64(rl.rate))

	client.tokens += tokensToAdd
	if client.tokens > rl.burst {
		client.tokens = rl.burst
	}

	client.lastCheck = now

	if client.tokens > 0 {
		client.tokens--
		return true
	}

	return false
}

// RateLimit middleware limits requests per IP
func RateLimit(limiter *RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			clientIP := getClientIP(r)

			if !limiter.allow(clientIP) {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Retry-After", "60")
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte(`{"error":"Rate limit exceeded. Please try again later."}`))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RateLimitStrict creates a stricter rate limiter for sensitive endpoints
func RateLimitStrict(limiter *RateLimiter) func(http.Handler) http.Handler {
	return RateLimit(limiter)
}

func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first (for proxies)
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		return xff
	}

	// Check X-Real-IP header
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}

	// Fall back to remote address
	return r.RemoteAddr
}

// DefaultRateLimiter creates a rate limiter with sensible defaults
// 100 requests per minute with burst of 20
func DefaultRateLimiter() *RateLimiter {
	return NewRateLimiter(100, time.Minute, 20)
}

// AuthRateLimiter creates a stricter rate limiter for auth endpoints
// 10 requests per minute with burst of 5
func AuthRateLimiter() *RateLimiter {
	return NewRateLimiter(10, time.Minute, 5)
}

// PublicAPIRateLimiter creates a rate limiter for public API endpoints
// 60 requests per minute with burst of 30
func PublicAPIRateLimiter() *RateLimiter {
	return NewRateLimiter(60, time.Minute, 30)
}
