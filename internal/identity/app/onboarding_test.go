package app

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/JLugagne/agach-mcp/internal/identity/domain"
	"github.com/JLugagne/agach-mcp/internal/identity/domain/repositories/nodes/nodestest"
	"github.com/JLugagne/agach-mcp/internal/identity/domain/repositories/onboardingcodes/onboardingcodestest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestOnboardingService(codes *onboardingcodestest.MockOnboardingCodeRepository, nodeRepo *nodestest.MockNodeRepository) *onboardingService {
	return NewOnboardingService(codes, nodeRepo, []byte("secret-that-is-at-least-32-bytes!"), 0).(*onboardingService)
}

func TestOnboardingService_GenerateCode_Success(t *testing.T) {
	var created domain.OnboardingCode
	codes := &onboardingcodestest.MockOnboardingCodeRepository{
		FindByCodeFunc: func(_ context.Context, _ string) (domain.OnboardingCode, error) {
			return domain.OnboardingCode{}, domain.ErrOnboardingCodeNotFound
		},
		CreateFunc: func(_ context.Context, code domain.OnboardingCode) error {
			created = code
			return nil
		},
	}
	nodeRepo := &nodestest.MockNodeRepository{}

	svc := newTestOnboardingService(codes, nodeRepo)
	actor := domain.Actor{UserID: domain.NewUserID()}

	oc, err := svc.GenerateCode(context.Background(), actor, domain.NodeModeDefault, "my-node")
	require.NoError(t, err)

	sixDigits := regexp.MustCompile(`^\d{6}$`)
	assert.True(t, sixDigits.MatchString(oc.Code), "code must be 6 digits, got %q", oc.Code)
	assert.Equal(t, actor.UserID, oc.CreatedByUserID)
	assert.Equal(t, domain.NodeModeDefault, oc.NodeMode)
	assert.Equal(t, "my-node", oc.NodeName)
	assert.WithinDuration(t, time.Now().Add(15*time.Minute), oc.ExpiresAt, 5*time.Second)
	assert.Equal(t, created.Code, oc.Code)
}

func TestOnboardingService_GenerateCode_UniquePerCall(t *testing.T) {
	codes := &onboardingcodestest.MockOnboardingCodeRepository{
		FindByCodeFunc: func(_ context.Context, _ string) (domain.OnboardingCode, error) {
			return domain.OnboardingCode{}, domain.ErrOnboardingCodeNotFound
		},
		CreateFunc: func(_ context.Context, _ domain.OnboardingCode) error {
			return nil
		},
	}
	nodeRepo := &nodestest.MockNodeRepository{}

	svc := newTestOnboardingService(codes, nodeRepo)
	actor := domain.Actor{UserID: domain.NewUserID()}

	seen := make(map[string]bool)
	for i := 0; i < 10; i++ {
		oc, err := svc.GenerateCode(context.Background(), actor, domain.NodeModeDefault, "")
		require.NoError(t, err)
		seen[oc.Code] = true
	}
	assert.Greater(t, len(seen), 1, "expected multiple different codes across 10 calls")
}

func TestOnboardingService_CompleteOnboarding_Success(t *testing.T) {
	userID := domain.NewUserID()
	storedCode := domain.OnboardingCode{
		ID:              domain.NewOnboardingCodeID(),
		Code:            "123456",
		CreatedByUserID: userID,
		NodeMode:        domain.NodeModeDefault,
		NodeName:        "stored-name",
		ExpiresAt:       time.Now().Add(15 * time.Minute),
	}

	var createdNode domain.Node
	var markUsedCalled bool

	codes := &onboardingcodestest.MockOnboardingCodeRepository{
		FindByCodeFunc: func(_ context.Context, _ string) (domain.OnboardingCode, error) {
			return storedCode, nil
		},
		MarkUsedFunc: func(_ context.Context, _ domain.OnboardingCodeID, _ domain.NodeID) error {
			markUsedCalled = true
			return nil
		},
	}
	nodeRepo := &nodestest.MockNodeRepository{
		CreateFunc: func(_ context.Context, node domain.Node) error {
			createdNode = node
			return nil
		},
	}

	svc := newTestOnboardingService(codes, nodeRepo)

	accessToken, refreshToken, node, err := svc.CompleteOnboarding(context.Background(), "123456", "my-daemon")
	require.NoError(t, err)

	assert.NotEmpty(t, accessToken)
	assert.NotEmpty(t, refreshToken)
	assert.Equal(t, userID, node.OwnerUserID)
	assert.Equal(t, "my-daemon", node.Name)
	assert.Equal(t, domain.NodeStatusActive, node.Status)
	assert.NotEmpty(t, node.RefreshTokenHash)
	assert.True(t, markUsedCalled)
	assert.Equal(t, createdNode.ID, node.ID)
}

func TestOnboardingService_CompleteOnboarding_ExpiredCode(t *testing.T) {
	expired := domain.OnboardingCode{
		ID:        domain.NewOnboardingCodeID(),
		Code:      "123456",
		ExpiresAt: time.Now().Add(-1 * time.Minute),
	}

	codes := &onboardingcodestest.MockOnboardingCodeRepository{
		FindByCodeFunc: func(_ context.Context, _ string) (domain.OnboardingCode, error) {
			return expired, nil
		},
	}
	nodeRepo := &nodestest.MockNodeRepository{}

	svc := newTestOnboardingService(codes, nodeRepo)

	_, _, _, err := svc.CompleteOnboarding(context.Background(), "123456", "daemon")
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrOnboardingCodeExpired))
}

func TestOnboardingService_CompleteOnboarding_UsedCode(t *testing.T) {
	usedAt := time.Now().Add(-1 * time.Minute)
	used := domain.OnboardingCode{
		ID:        domain.NewOnboardingCodeID(),
		Code:      "123456",
		ExpiresAt: time.Now().Add(15 * time.Minute),
		UsedAt:    &usedAt,
	}

	codes := &onboardingcodestest.MockOnboardingCodeRepository{
		FindByCodeFunc: func(_ context.Context, _ string) (domain.OnboardingCode, error) {
			return used, nil
		},
	}
	nodeRepo := &nodestest.MockNodeRepository{}

	svc := newTestOnboardingService(codes, nodeRepo)

	_, _, _, err := svc.CompleteOnboarding(context.Background(), "123456", "daemon")
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrOnboardingCodeUsed))
}

func TestOnboardingService_CompleteOnboarding_InvalidCode(t *testing.T) {
	codes := &onboardingcodestest.MockOnboardingCodeRepository{
		FindByCodeFunc: func(_ context.Context, _ string) (domain.OnboardingCode, error) {
			return domain.OnboardingCode{}, domain.ErrOnboardingCodeNotFound
		},
	}
	nodeRepo := &nodestest.MockNodeRepository{}

	svc := newTestOnboardingService(codes, nodeRepo)

	_, _, _, err := svc.CompleteOnboarding(context.Background(), "000000", "daemon")
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrOnboardingCodeNotFound))
}
