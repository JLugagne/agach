package client_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/JLugagne/agach-mcp/internal/daemon/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOnboardingClient_CompleteOnboarding_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/api/onboarding/complete", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		var req map[string]string
		json.NewDecoder(r.Body).Decode(&req)
		assert.Equal(t, "123456", req["code"])
		assert.Equal(t, "my-daemon", req["node_name"])

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"status": "success",
			"data": map[string]any{
				"access_token":  "jwt-token",
				"refresh_token": "refresh-token",
				"node": map[string]any{
					"id":   "node-uuid",
					"name": "my-daemon",
					"mode": "default",
				},
			},
		})
	}))
	defer server.Close()

	c := client.NewOnboardingClient(server.URL)
	result, err := c.CompleteOnboarding(context.Background(), "123456", "my-daemon")

	require.NoError(t, err)
	assert.Equal(t, "jwt-token", result.AccessToken)
	assert.Equal(t, "refresh-token", result.RefreshToken)
	assert.Equal(t, "node-uuid", result.NodeID)
	assert.Equal(t, "my-daemon", result.NodeName)
	assert.Equal(t, "default", result.Mode)
}

func TestOnboardingClient_CompleteOnboarding_CodeNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]any{
			"status": "fail",
			"error": map[string]any{
				"code":    "CODE_NOT_FOUND",
				"message": "onboarding code not found",
			},
		})
	}))
	defer server.Close()

	c := client.NewOnboardingClient(server.URL)
	_, err := c.CompleteOnboarding(context.Background(), "000000", "daemon")

	require.Error(t, err)
	var onboardingErr *client.OnboardingError
	require.ErrorAs(t, err, &onboardingErr)
	assert.True(t, onboardingErr.IsCodeNotFound())
}

func TestOnboardingClient_CompleteOnboarding_CodeExpired(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusGone)
		json.NewEncoder(w).Encode(map[string]any{
			"status": "fail",
			"error": map[string]any{
				"code":    "CODE_EXPIRED",
				"message": "onboarding code has expired",
			},
		})
	}))
	defer server.Close()

	c := client.NewOnboardingClient(server.URL)
	_, err := c.CompleteOnboarding(context.Background(), "123456", "daemon")

	require.Error(t, err)
	var onboardingErr *client.OnboardingError
	require.ErrorAs(t, err, &onboardingErr)
	assert.True(t, onboardingErr.IsCodeExpired())
}

func TestOnboardingClient_CompleteOnboarding_CodeUsed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]any{
			"status": "fail",
			"error": map[string]any{
				"code":    "CODE_ALREADY_USED",
				"message": "onboarding code has already been used",
			},
		})
	}))
	defer server.Close()

	c := client.NewOnboardingClient(server.URL)
	_, err := c.CompleteOnboarding(context.Background(), "123456", "daemon")

	require.Error(t, err)
	var onboardingErr *client.OnboardingError
	require.ErrorAs(t, err, &onboardingErr)
	assert.True(t, onboardingErr.IsCodeUsed())
}

func TestOnboardingClient_CompleteOnboarding_NetworkError(t *testing.T) {
	c := client.NewOnboardingClient("http://localhost:99999")
	_, err := c.CompleteOnboarding(context.Background(), "123456", "daemon")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "send request")
}
