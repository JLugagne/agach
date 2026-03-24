package onboardingcodes

import (
	"context"

	"github.com/JLugagne/agach-mcp/internal/identity/domain"
)

type OnboardingCodeRepository interface {
	Create(ctx context.Context, code domain.OnboardingCode) error
	FindByCode(ctx context.Context, code string) (domain.OnboardingCode, error)
	MarkUsed(ctx context.Context, codeID domain.OnboardingCodeID, nodeID domain.NodeID) error
	DeleteExpired(ctx context.Context) (int64, error)
}
