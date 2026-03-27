package middleware

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
)

// AuthValidator is the interface the auth middleware needs. It is intentionally
// narrow so that pkg/middleware does not depend on internal/identity.
type AuthValidator interface {
	ValidateJWT(ctx context.Context, token string) (any, error)
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hj, ok := rw.ResponseWriter.(http.Hijacker); ok {
		return hj.Hijack()
	}
	return nil, nil, fmt.Errorf("upstream ResponseWriter does not implement http.Hijacker")
}

func RequestLogger(logger *logrus.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rw, r)
			logger.WithFields(logrus.Fields{
				"method":   r.Method,
				"path":     r.URL.Path,
				"status":   rw.status,
				"duration": time.Since(start).Round(time.Millisecond).String(),
				"ip":       remoteAddr(r),
			}).Info("http")
		})
	}
}

const maxBodyBytes = 512 * 1024

type contextKey string

const ActorContextKey contextKey = "actor"

func unauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_, _ = w.Write([]byte(`{"status":"fail","error":{"code":"UNAUTHORIZED","message":"authentication required"}}`))
}

var allowedOrigins = []string{
	"http://localhost",
	"http://localhost:3000",
	"http://localhost:8080",
}

func isAllowedOrigin(origin string) bool {
	for _, allowed := range allowedOrigins {
		if origin == allowed {
			return true
		}
	}
	return false
}

func NewRequireAuth(authQueries AuthValidator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("Cache-Control", "no-store, no-cache")
			w.Header().Set("Vary", "Origin")

			origin := r.Header.Get("Origin")
			if origin != "" && isAllowedOrigin(origin) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
			} else {
				w.Header().Set("Access-Control-Allow-Origin", allowedOrigins[0])
			}

			authHeader := strings.TrimSpace(r.Header.Get("Authorization"))

			if authHeader == "" {
				unauthorized(w)
				return
			}

			ctx := r.Context()

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
		})
	}
}

func jsonError(w http.ResponseWriter, body string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_, _ = w.Write([]byte(body))
}

func LimitBodySize(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ContentLength < -1 {
			jsonError(w, `{"status":"fail","error":{"code":"BAD_REQUEST","message":"invalid content-length"}}`, http.StatusBadRequest)
			return
		}
		if r.ContentLength > maxBodyBytes {
			jsonError(w, `{"status":"fail","error":{"code":"BODY_TOO_LARGE","message":"request body exceeds limit"}}`, http.StatusRequestEntityTooLarge)
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
	calls    int
}

const cleanupInterval = 100

func newIPRateLimiter(r rate.Limit, b int) *ipRateLimiter {
	return &ipRateLimiter{
		limiters: make(map[string]*limiterEntry),
		r:        r,
		b:        b,
	}
}

func (l *ipRateLimiter) cleanup() {
	for ip, entry := range l.limiters {
		if time.Since(entry.lastSeen) > 10*time.Minute {
			delete(l.limiters, ip)
		}
	}
}

func (l *ipRateLimiter) getLimiter(ip string) *rate.Limiter {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.calls++
	if l.calls%cleanupInterval == 0 {
		l.cleanup()
	}

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
			jsonError(w, `{"status":"fail","error":{"code":"RATE_LIMITED","message":"too many requests"}}`, http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}
