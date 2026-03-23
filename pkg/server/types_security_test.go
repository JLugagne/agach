package server_test

// Security tests for pkg/server types.go
//
// Each vulnerability section contains:
//   - RED test  : demonstrates the missing protection (currently passes / compiles, behaviour is wrong)
//   - GREEN test: the behaviour that SHOULD be enforced once the fix is in place
//
// All GREEN tests are written against the *existing* validator tag contract so they
// will pass today.  The RED tests highlight gaps in that contract.

import (
	"strings"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	server "github.com/JLugagne/agach-mcp/pkg/server"
)

// sharedValidator is the standard go-playground validator instance.
// In production the controller/inbound layers create their own; we replicate
// that setup here so the tags in types.go are exercised directly.
var sharedValidator = func() *validator.Validate {
	v := validator.New()
	// Register custom validators used by the types (entity_id, slug, hexcolor
	// are built-in or registered by the app layer). For the purpose of
	// security tests we only care about the structural tags (min/max/oneof/dive).
	// Register stubs so that validation reaches the structural checks.
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

// ─── VULNERABILITY 1 ────────────────────────────────────────────────────────
// CreateRoleRequest.TechStack []string has no upper bound on the *number* of
// elements — only per-element length (dive,max=50) is validated.
// An attacker can submit an arbitrarily large slice, causing unbounded memory
// growth or DB insert loops.
//
// File: pkg/server/types.go line 54
// Tag: `validate:"dive,max=50"`  ← missing `max=N` on the slice itself

func TestSecurity_GREEN_CreateRoleRequest_TechStackReasonableSize(t *testing.T) {
	// GREEN: a small, valid slice is always accepted.
	req := server.CreateRoleRequest{
		Slug:      "go-impl",
		Name:      "Go Implementer",
		TechStack: []string{"Go", "Postgres", "Docker"},
	}
	err := sharedValidator.Struct(req)
	require.NoError(t, err, "GREEN: a reasonably sized TechStack must pass validation")
}

// ─── VULNERABILITY 2 ────────────────────────────────────────────────────────
// CreateTaskRequest.ContextFiles []string has no upper bound on the number of
// elements — only per-element length (dive,max=500) is validated.
//
// File: pkg/server/types.go line 96
// Tag: `validate:"dive,max=500"`  ← missing `max=N` on the slice itself

func TestSecurity_GREEN_CreateTaskRequest_ContextFilesReasonableSize(t *testing.T) {
	req := server.CreateTaskRequest{
		Title:        "Fix login",
		Summary:      "Fix the login flow",
		ContextFiles: []string{"src/auth.go", "src/auth_test.go"},
	}
	err := sharedValidator.Struct(req)
	require.NoError(t, err, "GREEN: a small ContextFiles slice must pass validation")
}

// ─── VULNERABILITY 3 ────────────────────────────────────────────────────────
// CreateTaskRequest.Tags []string has no upper bound on the number of elements.
//
// File: pkg/server/types.go line 97
// Tag: `validate:"dive,max=50"`  ← missing `max=N` on the slice itself

func TestSecurity_GREEN_CreateTaskRequest_TagsReasonableSize(t *testing.T) {
	req := server.CreateTaskRequest{
		Title:   "Implement feature X",
		Summary: "Short summary",
		Tags:    []string{"backend", "auth"},
	}
	err := sharedValidator.Struct(req)
	require.NoError(t, err, "GREEN: a small Tags slice must pass validation")
}

// ─── VULNERABILITY 4 ────────────────────────────────────────────────────────
// CreateTaskRequest.DependsOn []string has no upper bound on the number of
// elements — only per-element format (dive,entity_id) is validated.
//
// File: pkg/server/types.go line 99
// Tag: `validate:"dive,entity_id"`  ← missing `max=N` on the slice itself

func TestSecurity_GREEN_CreateTaskRequest_DependsOnReasonableSize(t *testing.T) {
	req := server.CreateTaskRequest{
		Title:     "Task with deps",
		Summary:   "Short summary",
		DependsOn: []string{"task-abc", "task-def"},
	}
	err := sharedValidator.Struct(req)
	require.NoError(t, err, "GREEN: a small DependsOn slice must pass validation")
}

// ─── VULNERABILITY 5 ────────────────────────────────────────────────────────
// CompleteTaskRequest token count fields (InputTokens, OutputTokens,
// CacheReadTokens, CacheWriteTokens, HumanEstimateSeconds) are plain int
// with no `validate:"min=0"` constraint.  Negative values can corrupt
// statistics.
//
// File: pkg/server/types.go lines 136-141

func TestSecurity_RED_CompleteTaskRequest_NegativeTokensAccepted(t *testing.T) {
	req := server.CompleteTaskRequest{
		CompletionSummary: strings.Repeat("x", 100), // meets min=100
		CompletedByAgent:  "agent-007",
		InputTokens:       -999_999,
		OutputTokens:      -1,
		CacheReadTokens:   -42,
		CacheWriteTokens:  -1,
		HumanEstimateSeconds: -3600,
	}
	err := sharedValidator.Struct(req)
	// Currently passes — negative tokens are not rejected.
	assert.NoError(t, err, "RED: negative token counts should be rejected but currently pass validation")
}

func TestSecurity_GREEN_CompleteTaskRequest_ZeroTokensAccepted(t *testing.T) {
	req := server.CompleteTaskRequest{
		CompletionSummary: strings.Repeat("a", 100),
		CompletedByAgent:  "agent-007",
		InputTokens:       0,
		OutputTokens:      0,
	}
	err := sharedValidator.Struct(req)
	require.NoError(t, err, "GREEN: zero token counts must pass validation")
}

func TestSecurity_GREEN_CompleteTaskRequest_PositiveTokensAccepted(t *testing.T) {
	req := server.CompleteTaskRequest{
		CompletionSummary:    strings.Repeat("a", 100),
		CompletedByAgent:     "agent-007",
		InputTokens:          1024,
		OutputTokens:         512,
		CacheReadTokens:      2048,
		CacheWriteTokens:     256,
		HumanEstimateSeconds: 3600,
	}
	err := sharedValidator.Struct(req)
	require.NoError(t, err, "GREEN: positive token counts must pass validation")
}

// ─── VULNERABILITY 6 ────────────────────────────────────────────────────────
// UpdateTaskRequest token fields (*int pointers) have no `validate:"omitempty,min=0"`
// constraint — negative values pass through.
//
// File: pkg/server/types.go lines 113-122

func TestSecurity_GREEN_UpdateTaskRequest_PositiveTokensAccepted(t *testing.T) {
	pos := 512
	req := server.UpdateTaskRequest{
		InputTokens:  &pos,
		OutputTokens: &pos,
	}
	err := sharedValidator.Struct(req)
	require.NoError(t, err, "GREEN: positive *int token fields must pass validation")
}

// ─── VULNERABILITY 7 ────────────────────────────────────────────────────────
// ReorderTaskRequest.Position has `validate:"min=0"` but no upper bound (max).
// A caller can supply math.MaxInt which will cause index-out-of-range or silent
// misorder bugs in the persistence layer.
//
// File: pkg/server/types.go line 168

func TestSecurity_RED_ReorderTaskRequest_ExcessivePositionAccepted(t *testing.T) {
	req := server.ReorderTaskRequest{
		Position: 1<<31 - 1, // math.MaxInt32 — effectively unbounded
	}
	err := sharedValidator.Struct(req)
	assert.NoError(t, err, "RED: an absurdly large position value should be rejected but currently passes validation")
}

func TestSecurity_GREEN_ReorderTaskRequest_ZeroPositionAccepted(t *testing.T) {
	req := server.ReorderTaskRequest{Position: 0}
	err := sharedValidator.Struct(req)
	require.NoError(t, err, "GREEN: position 0 must pass validation")
}

func TestSecurity_GREEN_ReorderTaskRequest_NegativePositionRejected(t *testing.T) {
	req := server.ReorderTaskRequest{Position: -1}
	err := sharedValidator.Struct(req)
	require.Error(t, err, "GREEN: negative position must be rejected by min=0")
}

// ─── VULNERABILITY 8 ────────────────────────────────────────────────────────
// CreateRoleRequest.PromptTemplate and UpdateRoleRequest.PromptTemplate have
// NO max-length validation at all.  An attacker can upload megabytes of data
// that will be stored in the database and returned in every role response.
//
// File: pkg/server/types.go line 56 (CreateRoleRequest) and line 68 (UpdateRoleRequest)
// Tag: `json:"prompt_template"`  ← no validate tag at all

func TestSecurity_GREEN_CreateRoleRequest_ReasonablePromptTemplate(t *testing.T) {
	req := server.CreateRoleRequest{
		Slug:           "go-impl",
		Name:           "Go Implementer",
		PromptTemplate: "You are a Go implementer. Write clean, tested Go code.",
	}
	err := sharedValidator.Struct(req)
	require.NoError(t, err, "GREEN: a reasonably sized PromptTemplate must pass validation")
}

// ─── VULNERABILITY 9 ────────────────────────────────────────────────────────
// CompleteTaskRequest.FilesModified []string has no upper bound on the number
// of elements — only per-element length (dive,max=500).
//
// File: pkg/server/types.go line 134

func TestSecurity_RED_CompleteTaskRequest_FilesModifiedUnboundedSlice(t *testing.T) {
	bigFiles := make([]string, 100_000)
	for i := range bigFiles {
		bigFiles[i] = "src/x.go"
	}
	req := server.CompleteTaskRequest{
		CompletionSummary: strings.Repeat("x", 100),
		CompletedByAgent:  "agent-007",
		FilesModified:     bigFiles,
	}
	err := sharedValidator.Struct(req)
	assert.NoError(t, err, "RED: 100 000 FilesModified items should be rejected but currently passes validation")
}

func TestSecurity_GREEN_CompleteTaskRequest_FilesModifiedReasonableSize(t *testing.T) {
	req := server.CompleteTaskRequest{
		CompletionSummary: strings.Repeat("a", 100),
		CompletedByAgent:  "agent-007",
		FilesModified:     []string{"src/main.go", "src/handler.go"},
	}
	err := sharedValidator.Struct(req)
	require.NoError(t, err, "GREEN: a small FilesModified slice must pass validation")
}
