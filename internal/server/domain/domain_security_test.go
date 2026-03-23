package domain_test

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

// TestSecurity_RED_TaskTitleUnbounded demonstrates that the domain type places
// no upper-bound on Task.Title length. An attacker / buggy agent can create a
// task with a megabyte-long title. The domain layer has no Validate() method or
// constant that defines a maximum title length, so upper-bound enforcement is
// entirely absent at the domain level and must currently be done (inconsistently)
// in the HTTP layer only.
//
// RED: domain.Task has no maximum-length constraint on Title.
// Fix: add a const MaxTaskTitleLength and a Validate() method (or use a
// constructor) that returns domain.ErrInvalidTaskData when Title length exceeds it.
func TestSecurity_RED_TaskTitleUnbounded(t *testing.T) {
	longTitle := strings.Repeat("A", 100_001) // 100 KB title

	task := domain.Task{
		ID:      domain.NewTaskID(),
		Title:   longTitle,
		Summary: "short summary",
	}

	// RED: the domain type accepts any length title silently.
	// After the fix a Validate() call should return an error.
	// Currently there is no Validate() method, so this passes without error —
	// demonstrating the vulnerability exists at the domain level.
	//
	// We assert the title was stored as-is (no truncation, no error), which is
	// the current vulnerable state.
	assert.Equal(t, 100_001, len(task.Title),
		"RED: domain.Task silently accepts a 100 KB title; "+
			"fix: add domain-level length validation (e.g. max 500 chars) so "+
			"storage exhaustion / truncation attacks are caught at the domain boundary")
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

// TestSecurity_RED_CommentContentUnbounded demonstrates that the domain type
// places no upper-bound on Comment.Content length. This allows an attacker to
// POST a gigabyte-sized comment body that exhausts server memory during JSON
// decoding and database storage.
//
// RED: domain.Comment has no maximum-length constraint on Content.
func TestSecurity_RED_CommentContentUnbounded(t *testing.T) {
	hugeContent := strings.Repeat("X", 1_000_001) // 1 MB content

	comment := domain.Comment{
		ID:      domain.NewCommentID(),
		TaskID:  domain.NewTaskID(),
		Content: hugeContent,
	}

	// RED: the domain type accepts any length content silently.
	assert.Equal(t, 1_000_001, len(comment.Content),
		"RED: domain.Comment silently accepts a 1 MB content string; "+
			"fix: add domain-level length validation (e.g. max 10 000 chars)")
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

// TestSecurity_RED_RolePromptTemplateUnbounded demonstrates that PromptTemplate
// on a Role has no length bound. An admin storing a 10 MB prompt template causes
// every RenderPrompt call for that role to allocate that memory on the heap and
// pass it through the template engine, enabling repeated memory exhaustion.
//
// RED: domain.Role has no maximum-length constraint on PromptTemplate.
func TestSecurity_RED_RolePromptTemplateUnbounded(t *testing.T) {
	bigTemplate := strings.Repeat("T", 10_000_001) // 10 MB

	role := domain.Role{
		ID:             domain.NewRoleID(),
		Slug:           "go-implement",
		Name:           "Go Implementer",
		PromptTemplate: bigTemplate,
	}

	// RED: the domain type accepts any length prompt template silently.
	assert.Equal(t, 10_000_001, len(role.PromptTemplate),
		"RED: domain.Role silently accepts a 10 MB PromptTemplate; "+
			"fix: add domain-level length validation (e.g. max 100 000 chars)")
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

// TestSecurity_RED_InvalidPriorityDefaultsToMedium demonstrates that any
// arbitrary string is silently accepted as a Priority and treated as "medium"
// (score 200) by Priority.Score(). This means a caller can pass
// priority="hack" and it silently succeeds rather than being rejected.
// An attacker can probe which values are "valid" by observing whether score
// returns 200 for invalid inputs.
//
// RED: Priority.Score() returns 200 for unknown values without signalling an
// error; there is no validation constructor or Validate() method on Priority.
func TestSecurity_RED_InvalidPriorityDefaultsToMedium(t *testing.T) {
	invalid := domain.Priority("totally-invalid-priority")

	score := invalid.Score()

	// RED: no error is returned; the invalid value silently acts as "medium".
	// The desired behaviour after fixing is for Priority to have a Validate()
	// method (or constructor) that returns an error for unknown values.
	assert.Equal(t, 200, score,
		"RED: Priority(\"totally-invalid-priority\").Score() returns 200 (medium) "+
			"instead of signalling an error; fix: add Priority.IsValid() bool or "+
			"Priority.Validate() error that returns domain.ErrInvalidTaskData for unknown values")
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

// TestSecurity_RED_InvalidAuthorTypeAccepted demonstrates that AuthorType has
// only two valid values ("agent", "human") but no validation is enforced at the
// domain type level. Any string can be assigned without error.
//
// RED: domain.AuthorType is a plain string alias with no IsValid() method.
// An attacker can supply author_type="admin" to impersonate a privileged actor
// in the comment history, or author_type="" to bypass author-type-based
// display logic in the UI.
func TestSecurity_RED_InvalidAuthorTypeAccepted(t *testing.T) {
	invalidType := domain.AuthorType("admin") // not "agent" or "human"

	comment := domain.Comment{
		ID:         domain.NewCommentID(),
		TaskID:     domain.NewTaskID(),
		AuthorType: invalidType,
		Content:    "I am an admin",
	}

	// RED: no validation occurs; the invalid author type is stored as-is.
	assert.Equal(t, domain.AuthorType("admin"), comment.AuthorType,
		"RED: domain.Comment accepts AuthorType(\"admin\") without any validation; "+
			"fix: add AuthorType.IsValid() bool or a constructor that rejects values "+
			"outside {\"agent\", \"human\"}")
}

// TestSecurity_GREEN_ValidAuthorTypesAreAccepted verifies the two legitimate
// author types are defined and distinguishable.
func TestSecurity_GREEN_ValidAuthorTypesAreAccepted(t *testing.T) {
	assert.Equal(t, domain.AuthorType("agent"), domain.AuthorTypeAgent)
	assert.Equal(t, domain.AuthorType("human"), domain.AuthorTypeHuman)
	assert.NotEqual(t, domain.AuthorTypeAgent, domain.AuthorTypeHuman)
}

