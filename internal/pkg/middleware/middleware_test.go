package middleware_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/JLugagne/agach-mcp/internal/pkg/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errUnauthorized = errors.New("authentication required")

// okHandler is a simple next handler that always responds 200.
var okHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
})

// mockAuthQueries is a test double for AuthValidator that validates tokens by a
// simple lookup map. Any token not in the map returns errUnauthorized.
type mockAuthQueries struct {
	validJWTs map[string]any
}

func (m *mockAuthQueries) ValidateJWT(_ context.Context, token string) (any, error) {
	if a, ok := m.validJWTs[token]; ok {
		return a, nil
	}
	return nil, errUnauthorized
}

type testActor struct {
	Email string
}

var validActor = testActor{Email: "test@example.com"}

func newTestAuthMiddleware() http.Handler {
	mock := &mockAuthQueries{
		validJWTs: map[string]any{"valid-jwt": validActor},
	}
	return middleware.NewRequireAuth(mock)(okHandler)
}

// TestRequireAuth verifies the RequireAuth middleware.
func TestRequireAuth(t *testing.T) {
	handler := newTestAuthMiddleware()

	t.Run("Passes through when Authorization header has valid JWT", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer valid-jwt")
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Equal(t, "ok", rr.Body.String())
	})

	t.Run("Rejects request with neither header with 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
		assert.Contains(t, rr.Body.String(), "UNAUTHORIZED")
	})

	t.Run("Rejects request with empty Authorization header with 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "")
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	t.Run("Rejects invalid JWT with 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer fake-token")
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})

}

// TestLimitBodySize verifies the LimitBodySize middleware.
func TestLimitBodySize(t *testing.T) {
	handler := middleware.LimitBodySize(okHandler)

	t.Run("Passes through request without Content-Length", func(t *testing.T) {
		body := strings.NewReader("small body")
		req := httptest.NewRequest(http.MethodPost, "/", body)
		// Do not set Content-Length; httptest.NewRequest sets it from body but
		// we can clear it to simulate chunked transfer.
		req.ContentLength = -1
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("Passes through request with small Content-Length", func(t *testing.T) {
		body := strings.NewReader("hello")
		req := httptest.NewRequest(http.MethodPost, "/", body)
		req.ContentLength = int64(len("hello"))
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("Rejects request with Content-Length exceeding 512KB with 413", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.ContentLength = 512*1024 + 1
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusRequestEntityTooLarge, rr.Code)
		assert.Contains(t, rr.Body.String(), "BODY_TOO_LARGE")
	})

	t.Run("Passes through request with Content-Length exactly at the limit", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.ContentLength = 512 * 1024
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		// Should not be rejected based on Content-Length alone (exactly at limit)
		assert.Equal(t, http.StatusOK, rr.Code)
	})
}

// TestRateLimit verifies the RateLimit middleware.
func TestRateLimit(t *testing.T) {
	handler := middleware.RateLimit(okHandler)

	t.Run("Allows requests within rate limit", func(t *testing.T) {
		// The global limiter allows burst of 10, so the first 10 requests from a
		// fresh IP should succeed. We use a unique RemoteAddr to avoid pollution
		// from other tests.
		for i := 0; i < 5; i++ {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = "192.0.2.100:12345"
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			require.Equal(t, http.StatusOK, rr.Code, "request %d should pass", i)
		}
	})

	t.Run("Returns 429 when rate limit is exceeded", func(t *testing.T) {
		// Use a dedicated IP with a fresh burst bucket and exhaust it.
		// Burst is 10, so send 11 requests rapidly.
		ip := "192.0.2.200:99999"
		var lastCode int
		for i := 0; i < 15; i++ {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = ip
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)
			lastCode = rr.Code
		}

		// After exhausting the burst bucket the last request must have been rejected.
		assert.Equal(t, http.StatusTooManyRequests, lastCode)
	})

	t.Run("429 response body contains RATE_LIMITED code", func(t *testing.T) {
		// Exhaust the limiter for this IP.
		ip := "192.0.2.201:11111"
		var body string
		for i := 0; i < 15; i++ {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = ip
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)
			if rr.Code == http.StatusTooManyRequests {
				body = rr.Body.String()
				break
			}
		}

		assert.Contains(t, body, "RATE_LIMITED")
	})
}
