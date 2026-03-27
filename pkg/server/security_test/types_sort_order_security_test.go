package security_test

// Security tests for pkg/server types.go — SortOrder and token count gaps.
//
// These RED tests document vulnerabilities NOT covered by the existing
// types_security_test.go file.

import (
	"math"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"

	server "github.com/JLugagne/agach-mcp/pkg/server"
)

// sortOrderValidator mirrors the shared validator from the existing test file.
var sortOrderValidator = func() *validator.Validate {
	v := validator.New()
	_ = v.RegisterValidation("entity_id", func(fl validator.FieldLevel) bool {
		s := fl.Field().String()
		return len(s) > 0 && len(s) <= 200
	})
	_ = v.RegisterValidation("slug", func(fl validator.FieldLevel) bool {
		s := fl.Field().String()
		return len(s) > 0
	})
	return v
}()

// ─── VULNERABILITY: SortOrder fields have no validation ─────────────────────
// CreateAgentRequest.SortOrder, CreateSkillRequest.SortOrder,
// CreateSpecializedAgentRequest.SortOrder, and CreateDockerfileRequest.SortOrder
// are plain int with no validate tag. An attacker can submit negative values
// or MaxInt to corrupt ordering logic.
//
// File: pkg/server/types.go (multiple locations)

// TestSecurity_RED_CreateAgentRequest_NegativeSortOrderAccepted documents that
// a negative SortOrder is silently accepted.
// TODO(security): add validate:"min=0,max=10000" to SortOrder fields
func TestSecurity_RED_CreateAgentRequest_NegativeSortOrderAccepted(t *testing.T) {
	req := server.CreateAgentRequest{
		Slug:      "test-agent",
		Name:      "Test Agent",
		SortOrder: -999,
	}
	err := sortOrderValidator.Struct(req)
	assert.NoError(t, err, "RED: negative SortOrder should be rejected but currently passes validation")
	t.Log("RED: CreateAgentRequest.SortOrder accepts negative values — no min=0 constraint")
}

// TestSecurity_RED_CreateAgentRequest_MaxIntSortOrderAccepted documents that
// math.MaxInt32 is accepted for SortOrder.
// TODO(security): add validate:"min=0,max=10000" to SortOrder fields
func TestSecurity_RED_CreateAgentRequest_MaxIntSortOrderAccepted(t *testing.T) {
	req := server.CreateAgentRequest{
		Slug:      "test-agent",
		Name:      "Test Agent",
		SortOrder: math.MaxInt32,
	}
	err := sortOrderValidator.Struct(req)
	assert.NoError(t, err, "RED: math.MaxInt32 SortOrder should be rejected but currently passes validation")
	t.Log("RED: CreateAgentRequest.SortOrder accepts MaxInt32 — no upper bound constraint")
}

// TestSecurity_RED_CreateSkillRequest_NegativeSortOrderAccepted documents that
// CreateSkillRequest also lacks SortOrder validation.
// TODO(security): add validate:"min=0,max=10000" to SortOrder fields
func TestSecurity_RED_CreateSkillRequest_NegativeSortOrderAccepted(t *testing.T) {
	req := server.CreateSkillRequest{
		Slug:      "test-skill",
		Name:      "Test Skill",
		SortOrder: -1,
	}
	err := sortOrderValidator.Struct(req)
	assert.NoError(t, err, "RED: negative SortOrder on CreateSkillRequest should be rejected but currently passes")
	t.Log("RED: CreateSkillRequest.SortOrder accepts negative values — no min=0 constraint")
}

// ─── VULNERABILITY: UpdateChatStatsRequest token fields lack min=0 ──────────
// UpdateChatStatsRequest.InputTokens, OutputTokens, CacheReadTokens, and
// CacheWriteTokens are plain int with no validate tag. Negative values can
// corrupt cumulative token statistics.
//
// File: pkg/server/types.go lines 571-577

// TestSecurity_RED_UpdateChatStatsRequest_NegativeTokensAccepted documents that
// negative token counts are silently accepted in chat stats updates.
// TODO(security): add validate:"min=0" to all token count fields
func TestSecurity_RED_UpdateChatStatsRequest_NegativeTokensAccepted(t *testing.T) {
	req := server.UpdateChatStatsRequest{
		InputTokens:      -1000,
		OutputTokens:     -500,
		CacheReadTokens:  -200,
		CacheWriteTokens: -100,
	}
	err := sortOrderValidator.Struct(req)
	assert.NoError(t, err, "RED: negative token counts in UpdateChatStatsRequest should be rejected but currently pass")
	t.Log("RED: UpdateChatStatsRequest accepts negative token values — no min=0 constraint")
}

// ─── VULNERABILITY: GrantUserAccessRequest.UserID has no max length ─────────
// GrantUserAccessRequest.UserID has validate:"required" but no max= constraint.
// An attacker can submit an arbitrarily long UserID string.
//
// File: pkg/server/types.go line 600

// TestSecurity_RED_GrantUserAccessRequest_UnboundedUserID documents that
// an extremely long UserID is accepted without error.
// TODO(security): add validate:"required,max=200" to UserID
func TestSecurity_RED_GrantUserAccessRequest_UnboundedUserID(t *testing.T) {
	longID := make([]byte, 10000)
	for i := range longID {
		longID[i] = 'a'
	}
	req := server.GrantUserAccessRequest{
		UserID: string(longID),
		Role:   "member",
	}
	err := sortOrderValidator.Struct(req)
	assert.NoError(t, err, "RED: a 10KB UserID should be rejected but currently passes validation")
	t.Log("RED: GrantUserAccessRequest.UserID has no max length — accepts arbitrarily long strings")
}

// ─── VULNERABILITY: GrantTeamAccessRequest.TeamID has no max length ─────────
// Same issue as UserID above.
//
// File: pkg/server/types.go line 609

// TestSecurity_RED_GrantTeamAccessRequest_UnboundedTeamID documents that
// an extremely long TeamID is accepted without error.
// TODO(security): add validate:"required,max=200" to TeamID
func TestSecurity_RED_GrantTeamAccessRequest_UnboundedTeamID(t *testing.T) {
	longID := make([]byte, 10000)
	for i := range longID {
		longID[i] = 'b'
	}
	req := server.GrantTeamAccessRequest{
		TeamID: string(longID),
	}
	err := sortOrderValidator.Struct(req)
	assert.NoError(t, err, "RED: a 10KB TeamID should be rejected but currently passes validation")
	t.Log("RED: GrantTeamAccessRequest.TeamID has no max length — accepts arbitrarily long strings")
}
