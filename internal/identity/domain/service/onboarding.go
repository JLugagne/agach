package service

import (
	"context"

	"github.com/JLugagne/agach-mcp/internal/identity/domain"
)

type OnboardingCommands interface {
	GenerateCode(ctx context.Context, actor domain.Actor, mode domain.NodeMode, nodeName string) (domain.OnboardingCode, error)
	CompleteOnboarding(ctx context.Context, code string, nodeName string) (accessToken, refreshToken string, node domain.Node, err error)
}
