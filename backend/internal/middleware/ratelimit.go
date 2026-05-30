package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
)

type ipEntry struct {
	tokens    float64
	lastCheck time.Time
}

type RateLimiter struct {
	mu       sync.Mutex
	ips      map[string]*ipEntry
	rate     float64 // tokens per second
	burst    int
	lastGC   time.Time
}

func NewRateLimiter(requestsPerMinute int, burst int) *RateLimiter {
	return &RateLimiter{
		ips:   make(map[string]*ipEntry),
		rate:  float64(requestsPerMinute) / 60.0,
		burst: burst,
	}
}

func (rl *RateLimiter) allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// GC stale entries every 5 minutes
	if time.Since(rl.lastGC) > 5*time.Minute {
		for k, v := range rl.ips {
			if time.Since(v.lastCheck) > 10*time.Minute {
				delete(rl.ips, k)
			}
		}
		rl.lastGC = time.Now()
	}

	now := time.Now()
	e, ok := rl.ips[ip]
	if !ok {
		rl.ips[ip] = &ipEntry{tokens: float64(rl.burst) - 1, lastCheck: now}
		return true
	}

	elapsed := now.Sub(e.lastCheck).Seconds()
	e.tokens += elapsed * rl.rate
	if e.tokens > float64(rl.burst) {
		e.tokens = float64(rl.burst)
	}
	e.lastCheck = now

	if e.tokens < 1 {
		return false
	}
	e.tokens--
	return true
}

func (rl *RateLimiter) Middleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ip := c.RealIP()
			if !rl.allow(ip) {
				return c.JSON(http.StatusTooManyRequests, map[string]any{
					"error":   true,
					"message": "Too many requests. Please try again later.",
					"code":    "RATE_LIMIT_EXCEEDED",
				})
			}
			return next(c)
		}
	}
}
