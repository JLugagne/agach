package onboardingcodestest

import (
	"context"

	"github.com/JLugagne/agach-mcp/internal/identity/domain"
)

type MockOnboardingCodeRepository struct {
	CreateFunc        func(ctx context.Context, code domain.OnboardingCode) error
	FindByCodeFunc    func(ctx context.Context, code string) (domain.OnboardingCode, error)
	MarkUsedFunc      func(ctx context.Context, codeID domain.OnboardingCodeID, nodeID domain.NodeID) error
	DeleteExpiredFunc func(ctx context.Context) (int64, error)
}

func (m *MockOnboardingCodeRepository) Create(ctx context.Context, code domain.OnboardingCode) error {
	if m.CreateFunc == nil {
		panic("called not defined CreateFunc")
	}
	return m.CreateFunc(ctx, code)
}

func (m *MockOnboardingCodeRepository) FindByCode(ctx context.Context, code string) (domain.OnboardingCode, error) {
	if m.FindByCodeFunc == nil {
		panic("called not defined FindByCodeFunc")
	}
	return m.FindByCodeFunc(ctx, code)
}

func (m *MockOnboardingCodeRepository) MarkUsed(ctx context.Context, codeID domain.OnboardingCodeID, nodeID domain.NodeID) error {
	if m.MarkUsedFunc == nil {
		panic("called not defined MarkUsedFunc")
	}
	return m.MarkUsedFunc(ctx, codeID, nodeID)
}

func (m *MockOnboardingCodeRepository) DeleteExpired(ctx context.Context) (int64, error) {
	if m.DeleteExpiredFunc == nil {
		panic("called not defined DeleteExpiredFunc")
	}
	return m.DeleteExpiredFunc(ctx)
}
