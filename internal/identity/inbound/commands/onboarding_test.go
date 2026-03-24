package commands_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/JLugagne/agach-mcp/internal/identity/domain"
	"github.com/JLugagne/agach-mcp/internal/identity/inbound/commands"
	"github.com/JLugagne/agach-mcp/pkg/controller"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─────────────────────────────────────────────────────────────────────────────
// Mock services
// ─────────────────────────────────────────────────────────────────────────────

type mockOnboardingCommands struct {
	generateCodeFunc       func(ctx context.Context, actor domain.Actor, mode domain.NodeMode, nodeName string) (domain.OnboardingCode, error)
	completeOnboardingFunc func(ctx context.Context, code string, nodeName string) (string, string, domain.Node, error)
}

func (m *mockOnboardingCommands) GenerateCode(ctx context.Context, actor domain.Actor, mode domain.NodeMode, nodeName string) (domain.OnboardingCode, error) {
	if m.generateCodeFunc != nil {
		return m.generateCodeFunc(ctx, actor, mode, nodeName)
	}
	return domain.OnboardingCode{}, nil
}

func (m *mockOnboardingCommands) CompleteOnboarding(ctx context.Context, code string, nodeName string) (string, string, domain.Node, error) {
	if m.completeOnboardingFunc != nil {
		return m.completeOnboardingFunc(ctx, code, nodeName)
	}
	return "", "", domain.Node{}, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────────────────────

func newTestOnboardingHandler(cmds *mockOnboardingCommands, qrs *mockAuthQueries) (*commands.OnboardingHandler, *mux.Router) {
	logger := logrus.New()
	logger.SetLevel(logrus.PanicLevel)
	ctrl := controller.NewController(logger)
	h := commands.NewOnboardingHandler(cmds, qrs, ctrl)
	r := mux.NewRouter()
	h.RegisterRoutes(r)
	return h, r
}

func postJSONOnboarding(router *mux.Router, path string, body interface{}) *httptest.ResponseRecorder {
	b, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr
}

func postJSONOnboardingWithAuth(router *mux.Router, path string, body interface{}, token string) *httptest.ResponseRecorder {
	b, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr
}

// ─────────────────────────────────────────────────────────────────────────────
// Tests for GenerateCode
// ─────────────────────────────────────────────────────────────────────────────

func TestOnboardingHandler_GenerateCode_Success(t *testing.T) {
	expiresAt := time.Now().Add(15 * time.Minute)
	code := domain.OnboardingCode{
		Code:      "123456",
		ExpiresAt: expiresAt,
	}

	cmds := &mockOnboardingCommands{
		generateCodeFunc: func(ctx context.Context, actor domain.Actor, mode domain.NodeMode, nodeName string) (domain.OnboardingCode, error) {
			assert.Equal(t, domain.NodeModeDefault, mode)
			assert.Equal(t, "test-node", nodeName)
			return code, nil
		},
	}
	qrs := &mockAuthQueries{
		validateJWTFunc: func(ctx context.Context, token string) (domain.Actor, error) {
			assert.Equal(t, "valid-token", token)
			return domain.Actor{UserID: domain.NewUserID(), Email: "user@example.com", Role: domain.RoleMember}, nil
		},
		getCurrentUserFunc: func(ctx context.Context, actor domain.Actor) (domain.User, error) {
			return domain.User{}, nil
		},
	}

	_, router := newTestOnboardingHandler(cmds, qrs)

	rr := postJSONOnboardingWithAuth(router, "/api/onboarding/codes", map[string]interface{}{
		"mode":      "default",
		"node_name": "test-node",
	}, "valid-token")

	require.Equal(t, http.StatusOK, rr.Code)

	var resp struct {
		Status string `json:"status"`
		Data   struct {
			Code      string    `json:"code"`
			ExpiresAt time.Time `json:"expires_at"`
		} `json:"data"`
	}
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, "success", resp.Status)
	assert.Equal(t, "123456", resp.Data.Code)
}

func TestOnboardingHandler_GenerateCode_Unauthenticated(t *testing.T) {
	cmds := &mockOnboardingCommands{}
	qrs := &mockAuthQueries{
		validateJWTFunc: func(ctx context.Context, token string) (domain.Actor, error) {
			return domain.Actor{}, nil
		},
		getCurrentUserFunc: func(ctx context.Context, actor domain.Actor) (domain.User, error) {
			return domain.User{}, nil
		},
	}
	_, router := newTestOnboardingHandler(cmds, qrs)

	// Request with no Authorization header
	rr := postJSONOnboarding(router, "/api/onboarding/codes", map[string]interface{}{
		"mode":      "default",
		"node_name": "test-node",
	})

	require.Equal(t, http.StatusUnauthorized, rr.Code)

	var resp struct {
		Status string `json:"status"`
		Error  struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, "fail", resp.Status)
	assert.Equal(t, "UNAUTHORIZED", resp.Error.Code)
	assert.Equal(t, "authentication required", resp.Error.Message)
}

func TestOnboardingHandler_GenerateCode_InvalidToken(t *testing.T) {
	cmds := &mockOnboardingCommands{}
	qrs := &mockAuthQueries{
		validateJWTFunc: func(ctx context.Context, token string) (domain.Actor, error) {
			return domain.Actor{}, domain.ErrUnauthorized
		},
		getCurrentUserFunc: func(ctx context.Context, actor domain.Actor) (domain.User, error) {
			return domain.User{}, nil
		},
	}

	_, router := newTestOnboardingHandler(cmds, qrs)

	rr := postJSONOnboardingWithAuth(router, "/api/onboarding/codes", map[string]interface{}{
		"mode":      "default",
		"node_name": "test-node",
	}, "invalid-token")

	require.Equal(t, http.StatusUnauthorized, rr.Code)

	var resp struct {
		Status string `json:"status"`
		Error  struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, "fail", resp.Status)
	assert.Equal(t, "UNAUTHORIZED", resp.Error.Code)
}

func TestOnboardingHandler_GenerateCode_InvalidMode(t *testing.T) {
	cmds := &mockOnboardingCommands{}
	qrs := &mockAuthQueries{
		validateJWTFunc: func(ctx context.Context, token string) (domain.Actor, error) {
			return domain.Actor{UserID: domain.NewUserID(), Email: "user@example.com", Role: domain.RoleMember}, nil
		},
		getCurrentUserFunc: func(ctx context.Context, actor domain.Actor) (domain.User, error) {
			return domain.User{}, nil
		},
	}

	_, router := newTestOnboardingHandler(cmds, qrs)

	rr := postJSONOnboardingWithAuth(router, "/api/onboarding/codes", map[string]interface{}{
		"mode":      "invalid",
		"node_name": "test-node",
	}, "valid-token")

	require.Equal(t, http.StatusBadRequest, rr.Code)

	var resp struct {
		Status string `json:"status"`
		Error  struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, "fail", resp.Status)
}

func TestOnboardingHandler_GenerateCode_MissingMode(t *testing.T) {
	cmds := &mockOnboardingCommands{}
	qrs := &mockAuthQueries{
		validateJWTFunc: func(ctx context.Context, token string) (domain.Actor, error) {
			return domain.Actor{UserID: domain.NewUserID(), Email: "user@example.com", Role: domain.RoleMember}, nil
		},
		getCurrentUserFunc: func(ctx context.Context, actor domain.Actor) (domain.User, error) {
			return domain.User{}, nil
		},
	}

	_, router := newTestOnboardingHandler(cmds, qrs)

	rr := postJSONOnboardingWithAuth(router, "/api/onboarding/codes", map[string]interface{}{
		"node_name": "test-node",
	}, "valid-token")

	require.Equal(t, http.StatusBadRequest, rr.Code)
}

// ─────────────────────────────────────────────────────────────────────────────
// Tests for CompleteOnboarding
// ─────────────────────────────────────────────────────────────────────────────

func TestOnboardingHandler_CompleteOnboarding_Success(t *testing.T) {
	nodeID := domain.NewNodeID()
	now := time.Now()
	node := domain.Node{
		ID:        nodeID,
		Name:      "test-node",
		Mode:      domain.NodeModeDefault,
		Status:    domain.NodeStatusActive,
		CreatedAt: now,
	}

	cmds := &mockOnboardingCommands{
		completeOnboardingFunc: func(ctx context.Context, code string, nodeName string) (string, string, domain.Node, error) {
			assert.Equal(t, "123456", code)
			assert.Equal(t, "test-node", nodeName)
			return "access-token", "refresh-token", node, nil
		},
	}
	qrs := &mockAuthQueries{
		validateJWTFunc: func(ctx context.Context, token string) (domain.Actor, error) {
			return domain.Actor{}, nil
		},
		getCurrentUserFunc: func(ctx context.Context, actor domain.Actor) (domain.User, error) {
			return domain.User{}, nil
		},
	}

	_, router := newTestOnboardingHandler(cmds, qrs)

	rr := postJSONOnboarding(router, "/api/onboarding/complete", map[string]interface{}{
		"code":      "123456",
		"node_name": "test-node",
	})

	require.Equal(t, http.StatusOK, rr.Code)

	var resp struct {
		Status string `json:"status"`
		Data   struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
			Node         struct {
				ID        string `json:"id"`
				Name      string `json:"name"`
				Mode      string `json:"mode"`
				Status    string `json:"status"`
				CreatedAt time.Time `json:"created_at"`
			} `json:"node"`
		} `json:"data"`
	}
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, "success", resp.Status)
	assert.Equal(t, "access-token", resp.Data.AccessToken)
	assert.Equal(t, "refresh-token", resp.Data.RefreshToken)
	assert.Equal(t, nodeID.String(), resp.Data.Node.ID)
	assert.Equal(t, "test-node", resp.Data.Node.Name)
	assert.Equal(t, "default", resp.Data.Node.Mode)
	assert.Equal(t, "active", resp.Data.Node.Status)
}

func TestOnboardingHandler_CompleteOnboarding_InvalidCode_NotFound(t *testing.T) {
	cmds := &mockOnboardingCommands{
		completeOnboardingFunc: func(ctx context.Context, code string, nodeName string) (string, string, domain.Node, error) {
			return "", "", domain.Node{}, domain.ErrOnboardingCodeNotFound
		},
	}
	qrs := &mockAuthQueries{
		validateJWTFunc: func(ctx context.Context, token string) (domain.Actor, error) {
			return domain.Actor{}, nil
		},
		getCurrentUserFunc: func(ctx context.Context, actor domain.Actor) (domain.User, error) {
			return domain.User{}, nil
		},
	}

	_, router := newTestOnboardingHandler(cmds, qrs)

	rr := postJSONOnboarding(router, "/api/onboarding/complete", map[string]interface{}{
		"code":      "999999",
		"node_name": "test-node",
	})

	require.Equal(t, http.StatusNotFound, rr.Code)

	var resp struct {
		Status string `json:"status"`
		Error  struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, "fail", resp.Status)
	assert.Equal(t, "CODE_NOT_FOUND", resp.Error.Code)
	assert.Equal(t, "onboarding code not found", resp.Error.Message)
}

func TestOnboardingHandler_CompleteOnboarding_ExpiredCode(t *testing.T) {
	cmds := &mockOnboardingCommands{
		completeOnboardingFunc: func(ctx context.Context, code string, nodeName string) (string, string, domain.Node, error) {
			return "", "", domain.Node{}, domain.ErrOnboardingCodeExpired
		},
	}
	qrs := &mockAuthQueries{
		validateJWTFunc: func(ctx context.Context, token string) (domain.Actor, error) {
			return domain.Actor{}, nil
		},
		getCurrentUserFunc: func(ctx context.Context, actor domain.Actor) (domain.User, error) {
			return domain.User{}, nil
		},
	}

	_, router := newTestOnboardingHandler(cmds, qrs)

	rr := postJSONOnboarding(router, "/api/onboarding/complete", map[string]interface{}{
		"code":      "123456",
		"node_name": "test-node",
	})

	require.Equal(t, http.StatusGone, rr.Code)

	var resp struct {
		Status string `json:"status"`
		Error  struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, "fail", resp.Status)
	assert.Equal(t, "CODE_EXPIRED", resp.Error.Code)
	assert.Equal(t, "onboarding code has expired", resp.Error.Message)
}

func TestOnboardingHandler_CompleteOnboarding_UsedCode(t *testing.T) {
	cmds := &mockOnboardingCommands{
		completeOnboardingFunc: func(ctx context.Context, code string, nodeName string) (string, string, domain.Node, error) {
			return "", "", domain.Node{}, domain.ErrOnboardingCodeUsed
		},
	}
	qrs := &mockAuthQueries{
		validateJWTFunc: func(ctx context.Context, token string) (domain.Actor, error) {
			return domain.Actor{}, nil
		},
		getCurrentUserFunc: func(ctx context.Context, actor domain.Actor) (domain.User, error) {
			return domain.User{}, nil
		},
	}

	_, router := newTestOnboardingHandler(cmds, qrs)

	rr := postJSONOnboarding(router, "/api/onboarding/complete", map[string]interface{}{
		"code":      "123456",
		"node_name": "test-node",
	})

	require.Equal(t, http.StatusConflict, rr.Code)

	var resp struct {
		Status string `json:"status"`
		Error  struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, "fail", resp.Status)
	assert.Equal(t, "CODE_ALREADY_USED", resp.Error.Code)
	assert.Equal(t, "onboarding code has already been used", resp.Error.Message)
}

func TestOnboardingHandler_CompleteOnboarding_InvalidCodeFormat(t *testing.T) {
	cmds := &mockOnboardingCommands{}
	qrs := &mockAuthQueries{
		validateJWTFunc: func(ctx context.Context, token string) (domain.Actor, error) {
			return domain.Actor{}, nil
		},
		getCurrentUserFunc: func(ctx context.Context, actor domain.Actor) (domain.User, error) {
			return domain.User{}, nil
		},
	}

	_, router := newTestOnboardingHandler(cmds, qrs)

	// Code with wrong length (should be exactly 6)
	rr := postJSONOnboarding(router, "/api/onboarding/complete", map[string]interface{}{
		"code":      "12345",
		"node_name": "test-node",
	})

	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestOnboardingHandler_CompleteOnboarding_NonNumericCode(t *testing.T) {
	cmds := &mockOnboardingCommands{}
	qrs := &mockAuthQueries{
		validateJWTFunc: func(ctx context.Context, token string) (domain.Actor, error) {
			return domain.Actor{}, nil
		},
		getCurrentUserFunc: func(ctx context.Context, actor domain.Actor) (domain.User, error) {
			return domain.User{}, nil
		},
	}

	_, router := newTestOnboardingHandler(cmds, qrs)

	// Code with non-numeric characters
	rr := postJSONOnboarding(router, "/api/onboarding/complete", map[string]interface{}{
		"code":      "abc123",
		"node_name": "test-node",
	})

	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestOnboardingHandler_CompleteOnboarding_MissingCode(t *testing.T) {
	cmds := &mockOnboardingCommands{}
	qrs := &mockAuthQueries{
		validateJWTFunc: func(ctx context.Context, token string) (domain.Actor, error) {
			return domain.Actor{}, nil
		},
		getCurrentUserFunc: func(ctx context.Context, actor domain.Actor) (domain.User, error) {
			return domain.User{}, nil
		},
	}

	_, router := newTestOnboardingHandler(cmds, qrs)

	rr := postJSONOnboarding(router, "/api/onboarding/complete", map[string]interface{}{
		"node_name": "test-node",
	})

	require.Equal(t, http.StatusBadRequest, rr.Code)
}
