package middleware

import (
	"net/http"
	"strings"
	"sync"
	"time"
)

type rateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*visitor
}

type visitor struct {
	lastSeen time.Time
	count    int
}

var limiter = &rateLimiter{
	visitors: make(map[string]*visitor),
}

const (
	maxReqPerMin = 60
)

func init() {
	go cleanupVisitors()
}

func cleanupVisitors() {
	for {
		time.Sleep(1 * time.Minute)
		limiter.mu.Lock()
		for ip, v := range limiter.visitors {
			if time.Since(v.lastSeen) > 1*time.Minute {
				delete(limiter.visitors, ip)
			}
		}
		limiter.mu.Unlock()
	}
}

func RateLimit(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip := strings.Split(r.RemoteAddr, ":")[0]

		limiter.mu.Lock()
		v, exists := limiter.visitors[ip]
		if !exists {
			limiter.visitors[ip] = &visitor{lastSeen: time.Now(), count: 1}
			limiter.mu.Unlock()
			next.ServeHTTP(w, r)
			return
		}

		if time.Since(v.lastSeen) > 1*time.Minute {
			v.count = 0
		}
		v.count++
		v.lastSeen = time.Now()

		if v.count > maxReqPerMin {
			limiter.mu.Unlock()
			http.Error(w, "Too many requests", http.StatusTooManyRequests)
			return
		}
		limiter.mu.Unlock()

		next.ServeHTTP(w, r)
	}
}
