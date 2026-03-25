package app

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"math/big"
	"time"

	"github.com/JLugagne/agach-mcp/internal/identity/domain"
	"github.com/JLugagne/agach-mcp/internal/identity/domain/repositories/nodes"
	"github.com/JLugagne/agach-mcp/internal/identity/domain/repositories/onboardingcodes"
	"github.com/JLugagne/agach-mcp/internal/identity/domain/service"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

const defaultDaemonJWTTTL = domain.DefaultDaemonJWTTTL

type onboardingService struct {
	codes        onboardingcodes.OnboardingCodeRepository
	nodes        nodes.NodeRepository
	jwtSecret    []byte
	daemonJWTTTL time.Duration
}

func NewOnboardingService(
	codes onboardingcodes.OnboardingCodeRepository,
	nodes nodes.NodeRepository,
	jwtSecret []byte,
	daemonJWTTTL time.Duration,
) service.OnboardingCommands {
	if daemonJWTTTL == 0 {
		daemonJWTTTL = defaultDaemonJWTTTL
	}
	return &onboardingService{
		codes:        codes,
		nodes:        nodes,
		jwtSecret:    jwtSecret,
		daemonJWTTTL: daemonJWTTTL,
	}
}

var _ service.OnboardingCommands = (*onboardingService)(nil)

func (s *onboardingService) GenerateCode(ctx context.Context, actor domain.Actor, mode domain.NodeMode, nodeName string) (domain.OnboardingCode, error) {
	var code string
	var err error

	for attempt := 0; attempt < 3; attempt++ {
		code, err = generateSixDigitCode()
		if err != nil {
			return domain.OnboardingCode{}, fmt.Errorf("generate code: %w", err)
		}

		_, findErr := s.codes.FindByCode(ctx, code)
		if findErr != nil {
			break
		}
	}

	now := time.Now()
	oc := domain.OnboardingCode{
		ID:              domain.NewOnboardingCodeID(),
		Code:            code,
		CreatedByUserID: actor.UserID,
		NodeMode:        mode,
		NodeName:        nodeName,
		ExpiresAt:       now.Add(15 * time.Minute),
		CreatedAt:       now,
	}

	if err := s.codes.Create(ctx, oc); err != nil {
		return domain.OnboardingCode{}, err
	}
	return oc, nil
}

func (s *onboardingService) CompleteOnboarding(ctx context.Context, code string, nodeName string) (string, string, domain.Node, error) {
	oc, err := s.codes.FindByCode(ctx, code)
	if err != nil {
		return "", "", domain.Node{}, err
	}

	if oc.IsExpired() {
		return "", "", domain.Node{}, domain.ErrOnboardingCodeExpired
	}
	if oc.IsUsed() {
		return "", "", domain.Node{}, domain.ErrOnboardingCodeUsed
	}

	rawRefresh := make([]byte, 32)
	if _, err := rand.Read(rawRefresh); err != nil {
		return "", "", domain.Node{}, fmt.Errorf("generate refresh token: %w", err)
	}
	refreshToken := base64.StdEncoding.EncodeToString(rawRefresh)

	hash, err := bcrypt.GenerateFromPassword([]byte(refreshToken), bcryptCost)
	if err != nil {
		return "", "", domain.Node{}, fmt.Errorf("hash refresh token: %w", err)
	}

	name := nodeName
	if name == "" {
		name = oc.NodeName
	}

	now := time.Now()
	node := domain.Node{
		ID:               domain.NewNodeID(),
		OwnerUserID:      oc.CreatedByUserID,
		Name:             name,
		Mode:             oc.NodeMode,
		Status:           domain.NodeStatusActive,
		RefreshTokenHash: string(hash),
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	if err := s.nodes.Create(ctx, node); err != nil {
		return "", "", domain.Node{}, err
	}

	if err := s.codes.MarkUsed(ctx, oc.ID, node.ID); err != nil {
		return "", "", domain.Node{}, err
	}

	accessToken, err := s.issueDaemonToken(node)
	if err != nil {
		return "", "", domain.Node{}, fmt.Errorf("issue daemon token: %w", err)
	}

	return accessToken, refreshToken, node, nil
}

func (s *onboardingService) issueDaemonToken(node domain.Node) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"sub":        node.ID.String(),
		"token_type": "daemon",
		"owner_id":   node.OwnerUserID.String(),
		"mode":       string(node.Mode),
		"iat":        now.Unix(),
		"exp":        now.Add(s.daemonJWTTTL).Unix(),
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString(s.jwtSecret)
}

func generateSixDigitCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}
