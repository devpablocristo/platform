package ginmw

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

)

const maxRateLimitKeys = 50_000

type rateLimiter struct {
	mu    sync.Mutex
	hits  map[string][]time.Time
	limit int
}

// NewRateLimit crea un middleware de rate limiting por IP (sliding window de 1 minuto).
// limit es la cantidad máxima de requests por minuto por IP (default 30).
func NewRateLimit(limit int) gin.HandlerFunc {
	if limit <= 0 {
		limit = 30
	}
	rl := &rateLimiter{
		hits:  make(map[string][]time.Time),
		limit: limit,
	}
	go rl.evictLoop()

	return func(c *gin.Context) {
		key := ClientIP(c)
		now := time.Now().UTC()

		rl.mu.Lock()
		defer rl.mu.Unlock()

		windowStart := now.Add(-1 * time.Minute)
		history := rl.hits[key]
		filtered := make([]time.Time, 0, len(history)+1)
		for _, ts := range history {
			if ts.After(windowStart) {
				filtered = append(filtered, ts)
			}
		}

		if len(filtered) >= rl.limit {
			rl.hits[key] = filtered
			c.AbortWithStatusJSON(http.StatusTooManyRequests, SimpleErrorResponse{Error: "rate limit exceeded"})
			return
		}

		filtered = append(filtered, now)
		rl.hits[key] = filtered
		c.Next()
	}
}

func (l *rateLimiter) evictLoop() {
	ticker := time.NewTicker(2 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		l.mu.Lock()
		now := time.Now().UTC()
		windowStart := now.Add(-1 * time.Minute)
		for key, history := range l.hits {
			if len(history) == 0 || history[len(history)-1].Before(windowStart) {
				delete(l.hits, key)
			}
		}
		// Hard cap para proteger contra IP rotation attacks.
		if len(l.hits) > maxRateLimitKeys {
			for key := range l.hits {
				delete(l.hits, key)
				if len(l.hits) <= maxRateLimitKeys/2 {
					break
				}
			}
		}
		l.mu.Unlock()
	}
}

// ClientIP detecta la IP del cliente con soporte para CloudFlare y proxies.
func ClientIP(c *gin.Context) string {
	for _, header := range []string{"CF-Connecting-IP", "X-Forwarded-For", "X-Real-IP"} {
		if raw := strings.TrimSpace(c.GetHeader(header)); raw != "" {
			if header == "X-Forwarded-For" {
				parts := strings.Split(raw, ",")
				if len(parts) > 0 {
					return strings.TrimSpace(parts[0])
				}
			}
			return raw
		}
	}
	host, _, err := net.SplitHostPort(strings.TrimSpace(c.Request.RemoteAddr))
	if err == nil && host != "" {
		return host
	}
	if c.ClientIP() != "" {
		return c.ClientIP()
	}
	return "unknown"
}
