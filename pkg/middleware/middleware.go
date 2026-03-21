package middleware

import (
	"context"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	identityservice "github.com/JLugagne/agach-mcp/internal/identity/domain/service"
	"golang.org/x/time/rate"
)

const maxBodyBytes = 512 * 1024

type contextKey string

const ActorContextKey contextKey = "actor"

func unauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_, _ = w.Write([]byte(`{"status":"fail","error":{"code":"UNAUTHORIZED","message":"authentication required"}}`))
}

func NewRequireAuth(authQueries identityservice.AuthQueries) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("Cache-Control", "no-store, no-cache")

			origin := r.Header.Get("Origin")
			if origin != "" {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Vary", "Origin")
			} else {
				w.Header().Set("Access-Control-Allow-Origin", "null")
			}

			authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
			apiKey := strings.TrimSpace(r.Header.Get("X-Api-Key"))

			if authHeader == "" && apiKey == "" {
				unauthorized(w)
				return
			}

			ctx := r.Context()

			if authHeader != "" {
				lower := strings.ToLower(authHeader)
				if !strings.HasPrefix(lower, "bearer ") {
					unauthorized(w)
					return
				}
				token := strings.TrimSpace(authHeader[len("bearer "):])
				if token == "" {
					unauthorized(w)
					return
				}
				actor, err := authQueries.ValidateJWT(ctx, token)
				if err != nil {
					unauthorized(w)
					return
				}
				r = r.WithContext(context.WithValue(ctx, ActorContextKey, actor))
				next.ServeHTTP(w, r)
				return
			}

			actor, err := authQueries.ValidateAPIKey(ctx, apiKey)
			if err != nil {
				unauthorized(w)
				return
			}
			r = r.WithContext(context.WithValue(ctx, ActorContextKey, actor))
			next.ServeHTTP(w, r)
		})
	}
}

func LimitBodySize(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ContentLength < -1 {
			http.Error(w, `{"status":"fail","error":{"code":"BAD_REQUEST","message":"invalid content-length"}}`, http.StatusBadRequest)
			return
		}
		if r.ContentLength > maxBodyBytes {
			http.Error(w, `{"status":"fail","error":{"code":"BODY_TOO_LARGE","message":"request body exceeds limit"}}`, http.StatusRequestEntityTooLarge)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
		next.ServeHTTP(w, r)
	})
}

type limiterEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type ipRateLimiter struct {
	mu       sync.Mutex
	limiters map[string]*limiterEntry
	r        rate.Limit
	b        int
}

func newIPRateLimiter(r rate.Limit, b int) *ipRateLimiter {
	l := &ipRateLimiter{
		limiters: make(map[string]*limiterEntry),
		r:        r,
		b:        b,
	}
	go l.cleanupLoop()
	return l
}

func (l *ipRateLimiter) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		l.mu.Lock()
		for ip, entry := range l.limiters {
			if time.Since(entry.lastSeen) > 10*time.Minute {
				delete(l.limiters, ip)
			}
		}
		l.mu.Unlock()
	}
}

func (l *ipRateLimiter) getLimiter(ip string) *rate.Limiter {
	l.mu.Lock()
	defer l.mu.Unlock()

	entry, exists := l.limiters[ip]
	if !exists {
		lim := rate.NewLimiter(l.r, l.b)
		l.limiters[ip] = &limiterEntry{limiter: lim, lastSeen: time.Now()}
		return lim
	}
	entry.lastSeen = time.Now()
	return entry.limiter
}

func remoteAddr(r *http.Request) string {
	// Always use the real RemoteAddr — never trust X-Forwarded-For from clients,
	// as it can be spoofed to bypass per-IP rate limiting.
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func RateLimit(next http.Handler) http.Handler {
	limiter := newIPRateLimiter(rate.Limit(5), 10)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !limiter.getLimiter(remoteAddr(r)).Allow() {
			http.Error(w, `{"status":"fail","error":{"code":"RATE_LIMITED","message":"too many requests"}}`, http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}
