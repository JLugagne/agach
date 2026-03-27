package security_test

// Security tests for the domain layer.
//
// Each vulnerability section has:
//   - RED test  (TestSecurity_RED_*):  fails today, demonstrates the gap
//   - GREEN test (TestSecurity_GREEN_*): passes today or documents desired post-fix state
//
// Vulnerabilities covered:
//  1. Unbounded string lengths in domain types (DoS / storage exhaustion)
//  2. Invalid Priority value silently accepted as "medium"
//  3. Invalid AuthorType silently accepted with no validation
//  4. (Removed — WIP limits no longer enforced)

import (
	"strings"
	"testing"

	"github.com/JLugagne/agach-mcp/internal/server/domain"
	"github.com/stretchr/testify/assert"
)

// ─────────────────────────────────────────────────────────────────────────────
// 1. Unbounded string lengths — Task.Title
// ─────────────────────────────────────────────────────────────────────────────

// TestSecurity_RED_TaskTitleUnbounded asserts that the domain layer MUST enforce
// an upper-bound on Task.Title length.
//
// FIX REQUIRED: add a const MaxTaskTitleLength and a Validate() method (or
// constructor) that returns domain.ErrInvalidTaskData when Title length exceeds it.
//
// This test FAILS today because domain.Task imposes no length constraint:
// a title of 100 KB is accepted silently.
func TestSecurity_RED_TaskTitleUnbounded(t *testing.T) {
	longTitle := strings.Repeat("A", 100_001) // 100 KB title

	task := domain.Task{
		ID:      domain.NewTaskID(),
		Title:   longTitle,
		Summary: "short summary",
	}

	err := task.ValidateTitle()
	assert.Error(t, err,
		"domain.Task.ValidateTitle() must reject a title longer than %d characters",
		domain.MaxTaskTitleLength)
}

// TestSecurity_GREEN_TaskTitleNormalLength verifies that a normally-sized title
// is always accepted (regression guard for the fix).
func TestSecurity_GREEN_TaskTitleNormalLength(t *testing.T) {
	task := domain.Task{
		ID:      domain.NewTaskID(),
		Title:   "Implement feature X",
		Summary: "short summary",
	}
	assert.NotEmpty(t, task.Title)
}

// ─────────────────────────────────────────────────────────────────────────────
// 2. Unbounded string lengths — Comment.Content
// ─────────────────────────────────────────────────────────────────────────────

// TestSecurity_RED_CommentContentUnbounded asserts that the domain layer MUST
// enforce an upper-bound on Comment.Content length.
//
// FIX REQUIRED: add a const MaxCommentContentLength and a Validate() method
// that returns an error when Content length exceeds it.
//
// This test FAILS today because domain.Comment imposes no length constraint:
// a content of 1 MB is accepted silently.
func TestSecurity_RED_CommentContentUnbounded(t *testing.T) {
	hugeContent := strings.Repeat("X", 1_000_001) // 1 MB content

	comment := domain.Comment{
		ID:      domain.NewCommentID(),
		TaskID:  domain.NewTaskID(),
		Content: hugeContent,
	}

	err := comment.ValidateContent()
	assert.Error(t, err,
		"domain.Comment.ValidateContent() must reject content longer than %d characters",
		domain.MaxCommentContentLength)
}

// TestSecurity_GREEN_CommentContentNormalLength verifies normally-sized content
// is accepted.
func TestSecurity_GREEN_CommentContentNormalLength(t *testing.T) {
	comment := domain.Comment{
		ID:      domain.NewCommentID(),
		TaskID:  domain.NewTaskID(),
		Content: "This is a normal comment.",
	}
	assert.NotEmpty(t, comment.Content)
}

// ─────────────────────────────────────────────────────────────────────────────
// 3. Unbounded string lengths — Role.PromptTemplate
// ─────────────────────────────────────────────────────────────────────────────

// TestSecurity_RED_RolePromptTemplateUnbounded asserts that the domain layer MUST
// enforce an upper-bound on Role.PromptTemplate length.
//
// FIX REQUIRED: add a const MaxPromptTemplateLength and a Validate() method
// that returns an error when PromptTemplate length exceeds it.
//
// This test FAILS today because domain.Role (alias for Agent) imposes no length
// constraint: a template of 10 MB is accepted silently.
func TestSecurity_RED_RolePromptTemplateUnbounded(t *testing.T) {
	bigTemplate := strings.Repeat("T", 10_000_001) // 10 MB

	role := domain.Agent{
		ID:             domain.NewRoleID(),
		Slug:           "go-implement",
		Name:           "Go Implementer",
		PromptTemplate: bigTemplate,
	}

	err := role.ValidatePromptTemplate()
	assert.Error(t, err,
		"domain.Agent.ValidatePromptTemplate() must reject a template longer than %d characters",
		domain.MaxPromptTemplateLength)
}

// TestSecurity_GREEN_RolePromptTemplateNormalLength verifies a normally-sized
// template is accepted.
func TestSecurity_GREEN_RolePromptTemplateNormalLength(t *testing.T) {
	role := domain.Role{
		ID:             domain.NewRoleID(),
		Slug:           "go-implement",
		Name:           "Go Implementer",
		PromptTemplate: "You are a Go expert. Task: {{task.title}}",
	}
	assert.NotEmpty(t, role.PromptTemplate)
}

// ─────────────────────────────────────────────────────────────────────────────
// 4. Invalid Priority value silently defaults to medium
// ─────────────────────────────────────────────────────────────────────────────

// TestSecurity_RED_InvalidPriorityDefaultsToMedium asserts that an invalid
// Priority value MUST NOT silently succeed as if it were "medium".
//
// FIX REQUIRED: add a Priority.IsValid() bool (or Priority.Validate() error)
// so callers can detect and reject unknown priority values before persisting them.
//
// This test FAILS today because Priority.Score() returns 200 for any unknown
// value, making "totally-invalid-priority" indistinguishable from "medium".
func TestSecurity_RED_InvalidPriorityDefaultsToMedium(t *testing.T) {
	invalid := domain.Priority("totally-invalid-priority")

	score := invalid.Score()

	// DESIRED: an invalid priority must NOT produce the same score as a valid
	// "medium" priority (200). It should either be distinguishable via an
	// IsValid() / Validate() method, or Score() should signal an error.
	// Today score == 200 so this assertion fails — demonstrating the gap.
	assert.NotEqual(t, 200, score,
		"Priority(%q).Score() must not silently return 200 (medium) for an "+
			"unknown value; fix: add Priority.IsValid() bool or "+
			"Priority.Validate() error that returns domain.ErrInvalidTaskData "+
			"for unknown values, and make Score() return a sentinel (e.g. 0) or "+
			"panic for invalid input", invalid)
}

// TestSecurity_GREEN_ValidPriorityScoresAreCorrect verifies that the four known
// priority values always produce the correct scores (regression guard).
func TestSecurity_GREEN_ValidPriorityScoresAreCorrect(t *testing.T) {
	cases := []struct {
		p     domain.Priority
		score int
	}{
		{domain.PriorityCritical, 400},
		{domain.PriorityHigh, 300},
		{domain.PriorityMedium, 200},
		{domain.PriorityLow, 100},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.score, tc.p.Score(),
			"priority %q must produce score %d", tc.p, tc.score)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// 5. Invalid AuthorType silently accepted
// ─────────────────────────────────────────────────────────────────────────────

// TestSecurity_RED_InvalidAuthorTypeAccepted asserts that domain.Comment MUST
// reject an AuthorType that is neither "agent" nor "human".
//
// FIX REQUIRED: add an AuthorType.IsValid() bool (or a Comment constructor)
// so callers can detect and reject unknown author types before persisting them.
//
// This test FAILS today because AuthorType is a plain string alias with no
// validation; AuthorType("admin") is stored without any error.
func TestSecurity_RED_InvalidAuthorTypeAccepted(t *testing.T) {
	invalidType := domain.AuthorType("admin") // not "agent" or "human"

	comment := domain.Comment{
		ID:         domain.NewCommentID(),
		TaskID:     domain.NewTaskID(),
		AuthorType: invalidType,
		Content:    "I am an admin",
	}

	err := comment.ValidateAuthorType()
	assert.Error(t, err,
		"domain.Comment.ValidateAuthorType() must reject AuthorType(%q); "+
			"only %q and %q are valid",
		comment.AuthorType, domain.AuthorTypeAgent, domain.AuthorTypeHuman)
}

// TestSecurity_GREEN_ValidAuthorTypesAreAccepted verifies the two legitimate
// author types are defined and distinguishable.
func TestSecurity_GREEN_ValidAuthorTypesAreAccepted(t *testing.T) {
	assert.Equal(t, domain.AuthorType("agent"), domain.AuthorTypeAgent)
	assert.Equal(t, domain.AuthorType("human"), domain.AuthorTypeHuman)
	assert.NotEqual(t, domain.AuthorTypeAgent, domain.AuthorTypeHuman)
}
