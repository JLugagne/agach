package security_test

// Additional RED security tests for pkg/server/types.go — round 2.
//
// These tests cover vulnerabilities NOT already documented in types_security_test.go.

import (
	"encoding/json"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"

	server "github.com/JLugagne/agach-mcp/pkg/server"
)

// ---- VULNERABILITY 10 -------------------------------------------------------
// SortOrder fields on CreateAgentRequest, CreateSkillRequest,
// CreateSpecializedAgentRequest, CreateDockerfileRequest, etc. are plain int
// with NO validation tag whatsoever. An attacker can submit negative values or
// math.MaxInt64 to corrupt ordering logic or cause integer overflow in
// downstream sort computations.
//
// File: pkg/server/types.go (CreateAgentRequest.SortOrder, CreateSkillRequest.SortOrder, etc.)

// TestSecurity_RED_SortOrderFieldsAcceptNegativeValues documents that negative
// SortOrder values pass validation.
// TODO(security): add `validate:"min=0,max=10000"` (or similar) to all SortOrder fields
func TestSecurity_RED_SortOrderFieldsAcceptNegativeValues(t *testing.T) {
	req := server.CreateAgentRequest{
		Slug:      "test-agent",
		Name:      "Test Agent",
		SortOrder: -999,
	}
	err := sharedValidator.Struct(req)
	assert.NoError(t, err,
		"RED: negative SortOrder is accepted — should be rejected with min=0")
	t.Log("RED: SortOrder fields have no validation — negative values and extreme positives are accepted")
}

// TestSecurity_RED_SortOrderFieldsAcceptMaxInt documents that math.MaxInt
// passes validation for SortOrder.
// TODO(security): add max= constraint to SortOrder fields
func TestSecurity_RED_SortOrderFieldsAcceptMaxInt(t *testing.T) {
	req := server.CreateSkillRequest{
		Slug:      "test-skill",
		Name:      "Test Skill",
		SortOrder: math.MaxInt32,
	}
	err := sharedValidator.Struct(req)
	assert.NoError(t, err,
		"RED: SortOrder of MaxInt32 is accepted — should be bounded to a reasonable max")
	t.Log("RED: SortOrder fields accept arbitrarily large values that can corrupt ordering")
}

// ---- VULNERABILITY 11 -------------------------------------------------------
// UpdateChatStatsRequest has plain int fields (InputTokens, OutputTokens,
// CacheReadTokens, CacheWriteTokens) with NO validation tags. Negative values
// can corrupt aggregated statistics.
//
// File: pkg/server/types.go — UpdateChatStatsRequest

// TestSecurity_RED_UpdateChatStatsNegativeTokensAccepted documents that
// negative token values pass validation on UpdateChatStatsRequest.
// TODO(security): add `validate:"min=0"` to all token count fields
func TestSecurity_RED_UpdateChatStatsNegativeTokensAccepted(t *testing.T) {
	req := server.UpdateChatStatsRequest{
		InputTokens:      -100000,
		OutputTokens:     -1,
		CacheReadTokens:  -50,
		CacheWriteTokens: -999,
	}
	err := sharedValidator.Struct(req)
	assert.NoError(t, err,
		"RED: negative token counts on UpdateChatStatsRequest pass validation — should be rejected")
	t.Log("RED: UpdateChatStatsRequest token fields lack min=0 validation — negative values corrupt statistics")
}

// ---- VULNERABILITY 12 -------------------------------------------------------
// GrantUserAccessRequest.UserID and GrantTeamAccessRequest.TeamID have
// `validate:"required"` but no max length constraint. An attacker can submit
// a multi-megabyte string that is stored in the database and returned in
// every access-list query.
//
// File: pkg/server/types.go — GrantUserAccessRequest, GrantTeamAccessRequest

// TestSecurity_RED_GrantUserAccessUnboundedUserID documents that an oversized
// UserID passes validation.
// TODO(security): add `max=200` to UserID and TeamID fields
func TestSecurity_RED_GrantUserAccessUnboundedUserID(t *testing.T) {
	// 1 MB string
	hugeID := make([]byte, 1024*1024)
	for i := range hugeID {
		hugeID[i] = 'a'
	}
	req := server.GrantUserAccessRequest{
		UserID: string(hugeID),
		Role:   "admin",
	}
	err := sharedValidator.Struct(req)
	assert.NoError(t, err,
		"RED: a 1 MB UserID passes validation — should be bounded by max= constraint")
	t.Log("RED: GrantUserAccessRequest.UserID has no max length — unbounded string accepted")
}

// TestSecurity_RED_GrantTeamAccessUnboundedTeamID documents that an oversized
// TeamID passes validation.
// TODO(security): add `max=200` to TeamID field
func TestSecurity_RED_GrantTeamAccessUnboundedTeamID(t *testing.T) {
	hugeID := make([]byte, 1024*1024)
	for i := range hugeID {
		hugeID[i] = 'b'
	}
	req := server.GrantTeamAccessRequest{
		TeamID: string(hugeID),
	}
	err := sharedValidator.Struct(req)
	assert.NoError(t, err,
		"RED: a 1 MB TeamID passes validation — should be bounded by max= constraint")
	t.Log("RED: GrantTeamAccessRequest.TeamID has no max length — unbounded string accepted")
}

// ---- VULNERABILITY 13 -------------------------------------------------------
// Nil slice fields in response types marshal to JSON `null` instead of `[]`.
// API consumers that do not handle null arrays will crash with nil pointer
// dereferences (especially JavaScript: `null.length` throws TypeError).
// This is an information contract issue that affects all response types with
// slice fields.
//
// File: pkg/server/types.go — TaskResponse.FilesModified, ContextFiles, Tags, etc.

// TestSecurity_RED_NilSliceFieldsMarshalToNull documents that nil slices in
// response types serialize as JSON null, not empty arrays.
// TODO(security): initialize slice fields to empty slices in constructors, or
// use custom MarshalJSON to emit [] for nil slices
func TestSecurity_RED_NilSliceFieldsMarshalToNull(t *testing.T) {
	resp := server.TaskResponse{
		// Leave slice fields at their zero value (nil)
	}
	data, err := json.Marshal(resp)
	assert.NoError(t, err)

	var decoded map[string]any
	assert.NoError(t, json.Unmarshal(data, &decoded))

	// nil slices marshal to null in Go's encoding/json
	assert.Nil(t, decoded["files_modified"],
		"RED: nil FilesModified marshals to null — consumers expecting [] will crash")
	assert.Nil(t, decoded["context_files"],
		"RED: nil ContextFiles marshals to null — consumers expecting [] will crash")
	assert.Nil(t, decoded["tags"],
		"RED: nil Tags marshals to null — consumers expecting [] will crash")
	t.Log("RED: nil slice fields in response types marshal to JSON null instead of [] — breaks API consumers")
}
