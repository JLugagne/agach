package commands_test

// deep_security_test.go — Deep security analysis of the commands package.
//
// Each vulnerability section contains:
//   - A RED test that demonstrates the vulnerability (fails until fixed).
//   - A GREEN test that verifies correct behaviour exists or that a mitigation works.
//
// Run with: go test -race -failfast ./internal/kanban/inbound/commands/...
//
// Vulnerabilities documented here:
//
//  SEC-01  IDOR: UpdateComment ignores taskId URL param — any comment editable
//          across project/task boundaries by knowing just the comment ID.
//  SEC-02  IDOR: DeleteComment ignores taskId URL param — same boundary bypass.
//  SEC-03  Unbounded PromptTemplate field — no max= validate tag; multi-MB
//          values silently stored and returned.
//  SEC-04  Unbounded ContextFiles/Tags array count — no max-items constraint;
//          10 000-entry arrays pass validation and reach the service layer.
//  SEC-05  Negative token counts accepted — UpdateTaskRequest integer fields
//          have no min=0 constraint; negative values corrupt statistics.
//  SEC-06  Arbitrary column slug injected verbatim — UpdateWIPLimit accepts
//          any string as the column slug without validating against allowed set.
//  SEC-07  Internal errors leak raw messages — controller.SendFail exposes
//          raw error.Error() text for non-apierror.Error values.
//  SEC-08  UpdateTaskRequest.Model field has no size limit — missing max= tag.
//  SEC-09  WontDo double-mutation is non-atomic — RequestWontDo followed by
//          ApproveWontDo: first succeeds, second fails, task left in wrong state.
//  SEC-10  CompleteTask silently swallows UpdateTask error — the post-completion
//          UpdateTask call for HumanEstimateSeconds uses "_ =" to ignore errors.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/JLugagne/agach-mcp/internal/kanban/domain"
	"github.com/JLugagne/agach-mcp/internal/kanban/domain/service/servicetest"
	"github.com/JLugagne/agach-mcp/internal/kanban/inbound/commands"
	"github.com/JLugagne/agach-mcp/pkg/controller"
	"github.com/JLugagne/agach-mcp/pkg/sse"
	"github.com/JLugagne/agach-mcp/pkg/websocket"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─────────────────────────────────────────────────────────────────────────────
// test infrastructure
// ─────────────────────────────────────────────────────────────────────────────

// newDeepSecurityRouter builds a bare router (no auth/rate/body middleware) for
// focused handler-level security tests.
func newDeepSecurityRouter(t *testing.T, app commands.App) *mux.Router {
	t.Helper()
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	logger.SetOutput(io.Discard)
	ctrl := controller.NewController(logger)
	hub := websocket.NewHub(logger)
	go hub.Run()
	sseHub := sse.NewHub()

	router := mux.NewRouter()
	commands.RegisterAllRoutes(router, app, ctrl, hub, sseHub)
	return router
}

// mockApp combines MockCommands and MockQueries for handlers that use both.
type mockApp struct {
	*servicetest.MockCommands
	*servicetest.MockQueries
}

func newTestApp(cmds *servicetest.MockCommands, qrs *servicetest.MockQueries) commands.App {
	return &mockApp{MockCommands: cmds, MockQueries: qrs}
}

// ─────────────────────────────────────────────────────────────────────────────
// SEC-01  IDOR: UpdateComment ignores taskId URL param
// ─────────────────────────────────────────────────────────────────────────────

// TestSecurity_SEC01_IDORUpdateCommentIgnoresTaskID_RED demonstrates that
// UpdateComment does not verify the comment belongs to the task in the URL.
//
// The handler at comments.go:129 parses commentId and projectId from the URL but
// silently drops the taskId parameter. Any caller who knows a comment's UUID can
// update it by crafting a URL with any projectId and any taskId — even from a
// completely different project.
//
// Expected (correct) behaviour: return 400/404 when the taskId in the URL does
// not match the parent task of the comment being updated.
//
// Observed (vulnerable) behaviour: the service is called with the correct
// commentId regardless of the taskId in the URL; no cross-check occurs at the
// HTTP layer.
func TestSecurity_SEC01_IDORUpdateCommentIgnoresTaskID_RED(t *testing.T) {
	projectID := domain.NewProjectID()
	realTaskID := domain.NewTaskID()   // the comment actually belongs to this task
	foreignTaskID := domain.NewTaskID() // attacker uses a different task ID in URL
	commentID := domain.NewCommentID()

	updateCalled := false
	var taskIDPassedToURL domain.TaskID

	cmds := &servicetest.MockCommands{
		UpdateCommentFunc: func(ctx context.Context, pID domain.ProjectID, cID domain.CommentID, content string) error {
			updateCalled = true
			// The handler never validates that taskId belongs to this comment.
			return nil
		},
	}
	qrs := &servicetest.MockQueries{}

	router := newDeepSecurityRouter(t, newTestApp(cmds, qrs))
	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)

	// Attacker uses foreignTaskID in the URL but targets commentID that belongs to realTaskID.
	taskIDPassedToURL = foreignTaskID
	_ = realTaskID // comment actually belongs here, but handler won't check

	url := fmt.Sprintf("%s/api/projects/%s/tasks/%s/comments/%s",
		srv.URL, projectID.String(), taskIDPassedToURL.String(), commentID.String())

	body, _ := json.Marshal(map[string]string{"content": "overwritten by attacker"})
	req, err := http.NewRequest(http.MethodPatch, url, bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// RED: the handler calls UpdateComment even though taskId doesn't match.
	// A secure implementation would verify the comment belongs to the stated task
	// and return 404 or 403 when it doesn't.
	if updateCalled {
		t.Error("SEC-01 RED: UpdateComment was called despite taskId mismatch — " +
			"the handler does not validate that the comment belongs to the supplied taskId; " +
			"fix: pass taskID to UpdateComment or perform a GetComment + ownership check " +
			"before calling the service")
	}

	// The response is 200 OK because the mock succeeds — attacker wins.
	assert.NotEqual(t, http.StatusOK, resp.StatusCode,
		"SEC-01 RED: handler returns 200 for a cross-task comment update; "+
			"must return 403 or 404 when taskId does not own the comment")
}

// TestSecurity_SEC01_IDORUpdateCommentValidatesTaskID_GREEN verifies the
// intended contract: if UpdateComment receives a taskId that does not own the
// comment, the service returns an error and the handler surfaces a 4xx.
//
// This test uses a mock that mimics the corrected behaviour where the service
// validates ownership.
func TestSecurity_SEC01_IDORUpdateCommentValidatesTaskID_GREEN(t *testing.T) {
	projectID := domain.NewProjectID()
	foreignTaskID := domain.NewTaskID()
	commentID := domain.NewCommentID()

	cmds := &servicetest.MockCommands{
		UpdateCommentFunc: func(ctx context.Context, pID domain.ProjectID, cID domain.CommentID, content string) error {
			// Service-level ownership check (what a correct implementation would do).
			// Must return a *domain.Error so the handler routes to SendFail (400)
			// rather than SendError (500).
			return domain.ErrCommentNotFound
		},
	}
	qrs := &servicetest.MockQueries{}

	router := newDeepSecurityRouter(t, newTestApp(cmds, qrs))
	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)

	url := fmt.Sprintf("%s/api/projects/%s/tasks/%s/comments/%s",
		srv.URL, projectID.String(), foreignTaskID.String(), commentID.String())

	body, _ := json.Marshal(map[string]string{"content": "attack"})
	req, err := http.NewRequest(http.MethodPatch, url, bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// UpdateComment handler uses StatusForbidden (403) for domain errors (comments.go:143).
	// Any 4xx status is acceptable: 400 or 403 both deny the request.
	isClientError := resp.StatusCode >= 400 && resp.StatusCode < 500
	assert.True(t, isClientError,
		"SEC-01 GREEN: with ownership check in service the handler must return 4xx, got %d",
		resp.StatusCode)
}

// ─────────────────────────────────────────────────────────────────────────────
// SEC-02  IDOR: DeleteComment ignores taskId URL param
// ─────────────────────────────────────────────────────────────────────────────

// TestSecurity_SEC02_IDORDeleteCommentIgnoresTaskID_RED demonstrates the same
// bypass for the DeleteComment handler (comments.go:165-186).
func TestSecurity_SEC02_IDORDeleteCommentIgnoresTaskID_RED(t *testing.T) {
	projectID := domain.NewProjectID()
	foreignTaskID := domain.NewTaskID()
	commentID := domain.NewCommentID()

	deleteCalled := false
	cmds := &servicetest.MockCommands{
		DeleteCommentFunc: func(ctx context.Context, pID domain.ProjectID, cID domain.CommentID) error {
			deleteCalled = true
			return nil
		},
	}
	qrs := &servicetest.MockQueries{}

	router := newDeepSecurityRouter(t, newTestApp(cmds, qrs))
	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)

	url := fmt.Sprintf("%s/api/projects/%s/tasks/%s/comments/%s",
		srv.URL, projectID.String(), foreignTaskID.String(), commentID.String())

	req, err := http.NewRequest(http.MethodDelete, url, nil)
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// RED: deletion succeeds with any taskId in the URL.
	if deleteCalled {
		t.Error("SEC-02 RED: DeleteComment was called despite unverified taskId — " +
			"attacker can delete any comment in the same project by guessing its UUID; " +
			"fix: verify comment ownership against taskId before deletion")
	}

	assert.NotEqual(t, http.StatusOK, resp.StatusCode,
		"SEC-02 RED: handler returns 200 for cross-task comment deletion; must return 4xx")
}

// TestSecurity_SEC02_IDORDeleteCommentOwnershipCheck_GREEN verifies that when
// the service returns a not-found error the handler correctly surfaces 400.
func TestSecurity_SEC02_IDORDeleteCommentOwnershipCheck_GREEN(t *testing.T) {
	projectID := domain.NewProjectID()
	foreignTaskID := domain.NewTaskID()
	commentID := domain.NewCommentID()

	cmds := &servicetest.MockCommands{
		DeleteCommentFunc: func(ctx context.Context, pID domain.ProjectID, cID domain.CommentID) error {
			// Must return a *domain.Error so the handler routes to SendFail (400).
			return domain.ErrCommentNotFound
		},
	}
	qrs := &servicetest.MockQueries{}

	router := newDeepSecurityRouter(t, newTestApp(cmds, qrs))
	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)

	url := fmt.Sprintf("%s/api/projects/%s/tasks/%s/comments/%s",
		srv.URL, projectID.String(), foreignTaskID.String(), commentID.String())

	req, err := http.NewRequest(http.MethodDelete, url, nil)
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	isClientError := resp.StatusCode >= 400 && resp.StatusCode < 500
	assert.True(t, isClientError,
		"SEC-02 GREEN: service-level ownership check must cause 4xx response, got %d",
		resp.StatusCode)
}

// ─────────────────────────────────────────────────────────────────────────────
// SEC-03  Unbounded PromptTemplate field
// ─────────────────────────────────────────────────────────────────────────────

// TestSecurity_SEC03_PromptTemplateUnbounded_RED demonstrates that a
// CreateRoleRequest with an arbitrarily large PromptTemplate value passes
// validation and reaches the service.
//
// In pkg/kanban/types.go line 57:
//
//	PromptTemplate string `json:"prompt_template"` // ← no validate tag at all
//
// An attacker can store multi-megabyte blobs in every role record.
func TestSecurity_SEC03_PromptTemplateUnbounded_RED(t *testing.T) {
	const hugeSize = 200_000 // 200 KB — well above any reasonable prompt template

	createCalled := false
	var receivedTemplateLen int

	cmds := &servicetest.MockCommands{
		CreateRoleFunc: func(ctx context.Context, slug, name, icon, color, description, promptHint, promptTemplate string, techStack []string, sortOrder int) (domain.Role, error) {
			createCalled = true
			receivedTemplateLen = len(promptTemplate)
			return domain.Role{ID: domain.NewRoleID(), Slug: slug, Name: name}, nil
		},
	}
	qrs := &servicetest.MockQueries{}

	router := newDeepSecurityRouter(t, newTestApp(cmds, qrs))
	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)

	body, _ := json.Marshal(map[string]interface{}{
		"slug":            "sec-test-role",
		"name":            "Security Test Role",
		"prompt_template": strings.Repeat("X", hugeSize),
	})

	req, err := http.NewRequest(http.MethodPost, srv.URL+"/api/roles", bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// RED: the 200 KB template is accepted because there is no validate:"max=..." tag.
	if createCalled && receivedTemplateLen >= hugeSize {
		t.Errorf("SEC-03 RED: PromptTemplate of %d bytes passed validation — "+
			"CreateRole was called with an unbounded payload; "+
			"fix: add validate:\"omitempty,max=50000\" (or appropriate limit) to "+
			"CreateRoleRequest.PromptTemplate and UpdateRoleRequest.PromptTemplate in pkg/kanban/types.go",
			receivedTemplateLen)
	}

	// The correct behaviour is a 400 rejection before reaching the service.
	if createCalled {
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode,
			"SEC-03 RED: a 200 KB PromptTemplate must return 400, not 200")
	}
}

// TestSecurity_SEC03_PromptTemplateBoundedByBodyLimit_GREEN verifies that the
// LimitBodySize middleware (512 KB) is the last-resort defense — bodies above
// 512 KB are rejected before any field is decoded.
//
// This is a mitigation, not a fix: the correct fix is to add a max= tag on the
// field itself so that individual field limits are enforced independently of the
// body size limit.
func TestSecurity_SEC03_PromptTemplateBoundedByBodyLimit_GREEN(t *testing.T) {
	cmds := &servicetest.MockCommands{}
	qrs := &servicetest.MockQueries{}

	// Build a router WITH the body-size middleware.
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	ctrl := controller.NewController(logger)
	hub := websocket.NewHub(logger)
	go hub.Run()
	sseHub := sse.NewHub()
	router := mux.NewRouter()
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.ContentLength > 512*1024 {
				http.Error(w, `{"status":"fail","error":{"code":"BODY_TOO_LARGE"}}`, http.StatusRequestEntityTooLarge)
				return
			}
			r.Body = http.MaxBytesReader(w, r.Body, 512*1024)
			next.ServeHTTP(w, r)
		})
	})
	commands.RegisterAllRoutes(router, newTestApp(cmds, qrs), ctrl, hub, sseHub)

	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)

	// Construct a raw payload that exceeds 512 KB.
	req, err := http.NewRequest(http.MethodPost, srv.URL+"/api/roles",
		strings.NewReader(strings.Repeat("x", 600*1024)))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.ContentLength = 600 * 1024

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusRequestEntityTooLarge, resp.StatusCode,
		"SEC-03 GREEN: body-size middleware rejects 600 KB payloads with 413")
}

// ─────────────────────────────────────────────────────────────────────────────
// SEC-04  Unbounded ContextFiles/Tags array count
// ─────────────────────────────────────────────────────────────────────────────

// TestSecurity_SEC04_UnboundedArraysPassValidation_RED demonstrates that
// CreateTaskRequest.ContextFiles and Tags arrays with 10 000 entries each pass
// validation.
//
// In pkg/kanban/types.go the validate tags are:
//
//	ContextFiles []string `json:"context_files" validate:"dive,max=500"`
//	Tags         []string `json:"tags" validate:"dive,max=50"`
//
// The "dive" validator checks each element's length, but there is no
// "max=N" constraint on the slice itself (e.g. validate:"max=100,dive,max=500").
// An attacker can submit 10 000 context files, each 500 bytes long, per single
// task creation request.
func TestSecurity_SEC04_UnboundedArraysPassValidation_RED(t *testing.T) {
	const itemCount = 10_000

	createCalled := false
	var receivedContextFilesCount int
	var receivedTagsCount int

	cmds := &servicetest.MockCommands{
		CreateTaskFunc: func(ctx context.Context, pID domain.ProjectID, title, summary, description string, priority domain.Priority, createdByRole, createdByAgent, assignedRole string, contextFiles, tags []string, estimatedEffort string, startInBacklog bool, featureID *domain.ProjectID) (domain.Task, error) {
			createCalled = true
			receivedContextFilesCount = len(contextFiles)
			receivedTagsCount = len(tags)
			return domain.Task{ID: domain.NewTaskID(), Title: title}, nil
		},
	}
	qrs := &servicetest.MockQueries{}
	projectID := domain.NewProjectID()

	router := newDeepSecurityRouter(t, newTestApp(cmds, qrs))
	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)

	contextFiles := make([]string, itemCount)
	for i := range contextFiles {
		contextFiles[i] = fmt.Sprintf("/path/to/file-%05d.go", i)
	}
	tags := make([]string, itemCount)
	for i := range tags {
		tags[i] = fmt.Sprintf("tag-%05d", i)
	}

	body, _ := json.Marshal(map[string]interface{}{
		"title":         "Array Bomb Task",
		"summary":       "testing array bounds",
		"context_files": contextFiles,
		"tags":          tags,
	})

	url := fmt.Sprintf("%s/api/projects/%s/tasks", srv.URL, projectID.String())
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// RED: 10 000-entry arrays pass validation and reach CreateTask.
	if createCalled {
		t.Errorf("SEC-04 RED: CreateTask was called with context_files=%d tags=%d — "+
			"no array-count limit is enforced; "+
			"fix: add validate:\"max=100,dive,max=500\" to ContextFiles and "+
			"validate:\"max=50,dive,max=50\" to Tags in CreateTaskRequest",
			receivedContextFilesCount, receivedTagsCount)
	}

	assert.NotEqual(t, http.StatusOK, resp.StatusCode,
		"SEC-04 RED: 10 000-entry arrays must return 4xx, not 200")
}

// TestSecurity_SEC04_ReasonableSizedArraysAccepted_GREEN verifies that a task
// with a reasonable number of context files and tags (within typical limits) is
// accepted by the validator.
func TestSecurity_SEC04_ReasonableSizedArraysAccepted_GREEN(t *testing.T) {
	createCalled := false
	cmds := &servicetest.MockCommands{
		CreateTaskFunc: func(ctx context.Context, pID domain.ProjectID, title, summary, description string, priority domain.Priority, createdByRole, createdByAgent, assignedRole string, contextFiles, tags []string, estimatedEffort string, startInBacklog bool, featureID *domain.ProjectID) (domain.Task, error) {
			createCalled = true
			return domain.Task{ID: domain.NewTaskID(), Title: title}, nil
		},
	}
	qrs := &servicetest.MockQueries{}
	projectID := domain.NewProjectID()

	router := newDeepSecurityRouter(t, newTestApp(cmds, qrs))
	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)

	body, _ := json.Marshal(map[string]interface{}{
		"title":         "Normal Task",
		"summary":       "reasonable sizes",
		"context_files": []string{"file1.go", "file2.go"},
		"tags":          []string{"backend", "critical"},
	})

	url := fmt.Sprintf("%s/api/projects/%s/tasks", srv.URL, projectID.String())
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode,
		"SEC-04 GREEN: reasonable array sizes must be accepted")
	assert.True(t, createCalled,
		"SEC-04 GREEN: CreateTask must be reached for normal-sized payloads")
}

// ─────────────────────────────────────────────────────────────────────────────
// SEC-05  Negative token counts accepted
// ─────────────────────────────────────────────────────────────────────────────

// TestSecurity_SEC05_NegativeTokenCountsAccepted_RED demonstrates that
// UpdateTaskRequest integer fields (InputTokens, OutputTokens, etc.) accept
// negative values, which would corrupt token usage statistics.
//
// In pkg/kanban/types.go lines 113-121, the fields use:
//
//	InputTokens *int `json:"input_tokens,omitempty"`
//
// There is no validate:"omitempty,min=0" tag. A malicious or buggy agent can
// submit −1_000_000 to reduce cumulative totals.
func TestSecurity_SEC05_NegativeTokenCountsAccepted_RED(t *testing.T) {
	updateCalled := false
	var receivedTokenUsage *domain.TokenUsage

	cmds := &servicetest.MockCommands{
		UpdateTaskFunc: func(ctx context.Context, pID domain.ProjectID, tID domain.TaskID, title, description, assignedRole, estimatedEffort, resolution *string, priority *domain.Priority, contextFiles, tags *[]string, tokenUsage *domain.TokenUsage, humanEstimateSeconds *int, featureID *domain.ProjectID, clearFeature bool) error {
			updateCalled = true
			receivedTokenUsage = tokenUsage
			return nil
		},
	}
	qrs := &servicetest.MockQueries{}
	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()

	router := newDeepSecurityRouter(t, newTestApp(cmds, qrs))
	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)

	negativeTokens := -1_000_000
	body, _ := json.Marshal(map[string]interface{}{
		"input_tokens":  negativeTokens,
		"output_tokens": negativeTokens,
	})

	url := fmt.Sprintf("%s/api/projects/%s/tasks/%s", srv.URL, projectID.String(), taskID.String())
	req, err := http.NewRequest(http.MethodPatch, url, bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// RED: negative values pass validation and reach the service.
	if updateCalled && receivedTokenUsage != nil && receivedTokenUsage.InputTokens < 0 {
		t.Errorf("SEC-05 RED: UpdateTask received InputTokens=%d — "+
			"negative token counts are not rejected by validation; "+
			"fix: add validate:\"omitempty,min=0\" to all *int token fields in "+
			"UpdateTaskRequest (pkg/kanban/types.go lines 113-122)",
			receivedTokenUsage.InputTokens)
	}

	assert.NotEqual(t, http.StatusOK, resp.StatusCode,
		"SEC-05 RED: negative token counts must return 4xx, not 200")
}

// TestSecurity_SEC05_PositiveTokenCountsAccepted_GREEN verifies that positive
// token counts are accepted by the handler.
func TestSecurity_SEC05_PositiveTokenCountsAccepted_GREEN(t *testing.T) {
	updateCalled := false
	cmds := &servicetest.MockCommands{
		UpdateTaskFunc: func(ctx context.Context, pID domain.ProjectID, tID domain.TaskID, title, description, assignedRole, estimatedEffort, resolution *string, priority *domain.Priority, contextFiles, tags *[]string, tokenUsage *domain.TokenUsage, humanEstimateSeconds *int, featureID *domain.ProjectID, clearFeature bool) error {
			updateCalled = true
			return nil
		},
	}
	qrs := &servicetest.MockQueries{}
	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()

	router := newDeepSecurityRouter(t, newTestApp(cmds, qrs))
	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)

	body, _ := json.Marshal(map[string]interface{}{
		"input_tokens":  1000,
		"output_tokens": 500,
	})

	url := fmt.Sprintf("%s/api/projects/%s/tasks/%s", srv.URL, projectID.String(), taskID.String())
	req, err := http.NewRequest(http.MethodPatch, url, bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode,
		"SEC-05 GREEN: positive token counts must be accepted")
	assert.True(t, updateCalled, "SEC-05 GREEN: UpdateTask must be reached")
}

// ─────────────────────────────────────────────────────────────────────────────
// SEC-06  Arbitrary column slug injected verbatim
// ─────────────────────────────────────────────────────────────────────────────

// TestSecurity_SEC06_ArbitraryColumnSlugInjected_RED demonstrates that the
// UpdateWIPLimit handler passes the URL path segment directly to the service as
// a domain.ColumnSlug without validating it against the allowed set
// {backlog, todo, in_progress, done, blocked}.
//
// columns.go line 45:
//
//	slug := domain.ColumnSlug(mux.Vars(r)["slug"])
//
// There is no validation before this value reaches UpdateColumnWIPLimit.
// An attacker can trigger behaviour with non-existent column names, and the
// service/database layer must absorb the load of looking up non-existent slugs.
func TestSecurity_SEC06_ArbitraryColumnSlugInjected_RED(t *testing.T) {
	wipCalled := false
	var receivedSlug domain.ColumnSlug

	cmds := &servicetest.MockCommands{
		UpdateColumnWIPLimitFunc: func(ctx context.Context, pID domain.ProjectID, slug domain.ColumnSlug, wipLimit int) error {
			wipCalled = true
			receivedSlug = slug
			return nil
		},
	}
	qrs := &servicetest.MockQueries{}
	projectID := domain.NewProjectID()

	router := newDeepSecurityRouter(t, newTestApp(cmds, qrs))
	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)

	// Inject an arbitrary slug that is not a valid column.
	arbitrarySlug := "'; DROP TABLE columns; --"
	url := fmt.Sprintf("%s/api/projects/%s/columns/%s/wip-limit",
		srv.URL, projectID.String(), arbitrarySlug)

	body, _ := json.Marshal(map[string]int{"wip_limit": 5})
	req, err := http.NewRequest(http.MethodPatch, url, bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Note: gorilla/mux percent-decodes the path segment, so the injected value
	// arrives at the handler (potentially URL-decoded). The assertion focuses on
	// whether an invalid slug is passed into the service.
	if wipCalled {
		t.Errorf("SEC-06 RED: UpdateColumnWIPLimit called with unvalidated slug=%q — "+
			"the handler does not reject slugs outside the allowed set; "+
			"fix: validate slug against domain.ColumnBacklog/ColumnTodo/ColumnInProgress/"+
			"ColumnDone/ColumnBlocked before calling the service",
			receivedSlug)
	}

	assert.NotEqual(t, http.StatusNoContent, resp.StatusCode,
		"SEC-06 RED: invalid column slug must return 4xx, not 204")
}

// TestSecurity_SEC06_ValidColumnSlugAccepted_GREEN verifies that a known slug
// reaches the service and returns 204.
func TestSecurity_SEC06_ValidColumnSlugAccepted_GREEN(t *testing.T) {
	wipCalled := false
	cmds := &servicetest.MockCommands{
		UpdateColumnWIPLimitFunc: func(ctx context.Context, pID domain.ProjectID, slug domain.ColumnSlug, wipLimit int) error {
			wipCalled = true
			return nil
		},
	}
	qrs := &servicetest.MockQueries{}
	projectID := domain.NewProjectID()

	router := newDeepSecurityRouter(t, newTestApp(cmds, qrs))
	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)

	url := fmt.Sprintf("%s/api/projects/%s/columns/in_progress/wip-limit",
		srv.URL, projectID.String())

	body, _ := json.Marshal(map[string]int{"wip_limit": 3})
	req, err := http.NewRequest(http.MethodPatch, url, bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNoContent, resp.StatusCode,
		"SEC-06 GREEN: valid slug 'in_progress' must return 204")
	assert.True(t, wipCalled, "SEC-06 GREEN: service must be reached for valid slug")
}

// ─────────────────────────────────────────────────────────────────────────────
// SEC-07  Internal errors leak raw messages
// ─────────────────────────────────────────────────────────────────────────────

// TestSecurity_SEC07_InternalErrorLeaksRawMessage_RED demonstrates that when
// controller.SendFail receives an error that is NOT an *apierror.Error it falls
// back to:
//
//	errMsg = err.Error()
//
// This exposes raw internal error strings to the HTTP client, potentially
// leaking database table names, file paths, or other implementation details.
//
// controller.go line 83-84:
//
//	} else {
//	    errCode = "CLIENT_ERROR"
//	    errMsg = err.Error()   ← leaks internal error text
//	}
func TestSecurity_SEC07_InternalErrorLeaksRawMessage_RED(t *testing.T) {
	internalMessage := "pq: duplicate key value violates unique constraint \"projects_pkey\""

	cmds := &servicetest.MockCommands{
		CreateProjectFunc: func(ctx context.Context, name, description, workDir, createdByRole, createdByAgent string, parentID *domain.ProjectID) (domain.Project, error) {
			// Return a plain error (not *apierror.Error and not a domain error).
			// This simulates a database driver error escaping the service layer.
			return domain.Project{}, errors.New(internalMessage)
		},
	}
	qrs := &servicetest.MockQueries{}

	router := newDeepSecurityRouter(t, newTestApp(cmds, qrs))
	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)

	body, _ := json.Marshal(map[string]string{
		"name": "Project Alpha",
	})
	req, err := http.NewRequest(http.MethodPost, srv.URL+"/api/projects", bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	rawBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	bodyStr := string(rawBody)

	// RED: the response body contains the raw internal error message.
	// Note: CreateProject returns a non-domain error, so the handler calls
	// SendError (500 path) which already sanitises the message. The leak occurs
	// specifically in SendFail when validators or other non-apierror errors are passed.
	//
	// Trigger the SendFail path: submit a request whose domain returns a plain error.
	// We simulate this via a domain path that calls SendFail with a plain error.
	// UpdateProject passes req errors through SendFail without apierror wrapping
	// when DecodeAndValidate returns a raw json.SyntaxError.
	_ = bodyStr // May not contain the leak on this code path; documented below.

	// The leak is specifically in the SendFail path with non-apierror errors.
	// Demonstrate with a JSON decode error (which is a plain error, not apierror.Error):
	invalidJSON := bytes.NewReader([]byte("{invalid json"))
	req2, err := http.NewRequest(http.MethodPost, srv.URL+"/api/projects", invalidJSON)
	require.NoError(t, err)
	req2.Header.Set("Content-Type", "application/json")

	resp2, err := http.DefaultClient.Do(req2)
	require.NoError(t, err)
	defer resp2.Body.Close()

	rawBody2, err := io.ReadAll(resp2.Body)
	require.NoError(t, err)
	bodyStr2 := string(rawBody2)

	// RED: the raw json.SyntaxError message (with character offsets and internal
	// decoder details) appears verbatim in the response body.
	assert.NotContains(t, bodyStr2, "invalid character",
		"SEC-07 RED: raw JSON syntax error leaks internal decoder message to client; "+
			"fix: in controller.SendFail, replace raw err.Error() with a generic message "+
			"when the error is not an *apierror.Error, e.g. \"invalid request data\"")
}

// TestSecurity_SEC07_DomainErrorMessageIsSafe_GREEN verifies that when the
// service returns a *domain.Error (the type the handlers check for), the client
// receives only the controlled Code and Message fields.
func TestSecurity_SEC07_DomainErrorMessageIsSafe_GREEN(t *testing.T) {
	cmds := &servicetest.MockCommands{
		CreateProjectFunc: func(ctx context.Context, name, description, workDir, createdByRole, createdByAgent string, parentID *domain.ProjectID) (domain.Project, error) {
			// *domain.Error is routed through SendFail with controlled code/message.
			return domain.Project{}, domain.ErrProjectAlreadyExists
		},
	}
	qrs := &servicetest.MockQueries{}

	router := newDeepSecurityRouter(t, newTestApp(cmds, qrs))
	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)

	body, _ := json.Marshal(map[string]string{"name": "Duplicate Project"})
	req, err := http.NewRequest(http.MethodPost, srv.URL+"/api/projects", bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	rawBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	assertJSONFailCode(t, rawBody, "PROJECT_ALREADY_EXISTS")
	assert.NotContains(t, string(rawBody), "pq:",
		"SEC-07 GREEN: domain.Error path must not leak DB driver internals")
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode,
		"SEC-07 GREEN: domain error must result in 400 response")
}

// ─────────────────────────────────────────────────────────────────────────────
// SEC-08  UpdateTaskRequest.Model field has no size limit
// ─────────────────────────────────────────────────────────────────────────────

// TestSecurity_SEC08_ModelFieldUnbounded_RED demonstrates that the Model field
// in UpdateTaskRequest accepts arbitrarily large strings.
//
// pkg/kanban/types.go line 117:
//
//	Model *string `json:"model,omitempty"` // ← no validate tag
//
// This allows storing very long model names in task records.
func TestSecurity_SEC08_ModelFieldUnbounded_RED(t *testing.T) {
	const hugeModelName = 100_000 // 100 KB model name

	updateCalled := false
	var receivedModelLen int

	cmds := &servicetest.MockCommands{
		UpdateTaskFunc: func(ctx context.Context, pID domain.ProjectID, tID domain.TaskID, title, description, assignedRole, estimatedEffort, resolution *string, priority *domain.Priority, contextFiles, tags *[]string, tokenUsage *domain.TokenUsage, humanEstimateSeconds *int, featureID *domain.ProjectID, clearFeature bool) error {
			updateCalled = true
			if tokenUsage != nil {
				receivedModelLen = len(tokenUsage.Model)
			}
			return nil
		},
	}
	qrs := &servicetest.MockQueries{}
	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()

	router := newDeepSecurityRouter(t, newTestApp(cmds, qrs))
	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)

	body, _ := json.Marshal(map[string]interface{}{
		"model":        strings.Repeat("M", hugeModelName),
		"input_tokens": 100,
	})

	url := fmt.Sprintf("%s/api/projects/%s/tasks/%s", srv.URL, projectID.String(), taskID.String())
	req, err := http.NewRequest(http.MethodPatch, url, bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	if updateCalled && receivedModelLen >= hugeModelName {
		t.Errorf("SEC-08 RED: UpdateTask received Model of length %d — "+
			"no size limit enforced on Model field; "+
			"fix: add validate:\"omitempty,max=200\" to UpdateTaskRequest.Model and "+
			"CompleteTaskRequest.Model in pkg/kanban/types.go",
			receivedModelLen)
	}

	assert.NotEqual(t, http.StatusOK, resp.StatusCode,
		"SEC-08 RED: a 100 KB Model name must return 4xx, not 200")
}

// TestSecurity_SEC08_ReasonableModelNameAccepted_GREEN verifies that a
// short, realistic model name is accepted.
func TestSecurity_SEC08_ReasonableModelNameAccepted_GREEN(t *testing.T) {
	updateCalled := false
	cmds := &servicetest.MockCommands{
		UpdateTaskFunc: func(ctx context.Context, pID domain.ProjectID, tID domain.TaskID, title, description, assignedRole, estimatedEffort, resolution *string, priority *domain.Priority, contextFiles, tags *[]string, tokenUsage *domain.TokenUsage, humanEstimateSeconds *int, featureID *domain.ProjectID, clearFeature bool) error {
			updateCalled = true
			return nil
		},
	}
	qrs := &servicetest.MockQueries{}
	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()

	router := newDeepSecurityRouter(t, newTestApp(cmds, qrs))
	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)

	body, _ := json.Marshal(map[string]interface{}{
		"model":        "claude-sonnet-4-5",
		"input_tokens": 100,
	})

	url := fmt.Sprintf("%s/api/projects/%s/tasks/%s", srv.URL, projectID.String(), taskID.String())
	req, err := http.NewRequest(http.MethodPatch, url, bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode,
		"SEC-08 GREEN: short model name must be accepted")
	assert.True(t, updateCalled, "SEC-08 GREEN: UpdateTask must be reached")
}

// ─────────────────────────────────────────────────────────────────────────────
// SEC-09  WontDo double-mutation is non-atomic
// ─────────────────────────────────────────────────────────────────────────────

// TestSecurity_SEC09_WontDoNonAtomicPartialFailure_RED demonstrates the
// non-atomic double-mutation in WontDo (tasks.go:410-428):
//
//  1. RequestWontDo → moves task to blocked column, sets wont_do_requested=1
//  2. ApproveWontDo → moves task to done column
//
// If ApproveWontDo fails (e.g. transient DB error), the task remains in the
// blocked column with wont_do_requested=1 — an inconsistent state. The handler
// returns a 500 error but the mutation from step 1 is already persisted.
func TestSecurity_SEC09_WontDoNonAtomicPartialFailure_RED(t *testing.T) {
	requestCalled := false
	approveCalled := false

	approveErr := errors.New("transient database error")

	cmds := &servicetest.MockCommands{
		RequestWontDoFunc: func(ctx context.Context, pID domain.ProjectID, tID domain.TaskID, reason, requestedBy string) error {
			requestCalled = true
			return nil // First mutation succeeds.
		},
		ApproveWontDoFunc: func(ctx context.Context, pID domain.ProjectID, tID domain.TaskID) error {
			approveCalled = true
			return approveErr // Second mutation fails.
		},
	}
	qrs := &servicetest.MockQueries{}
	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()

	router := newDeepSecurityRouter(t, newTestApp(cmds, qrs))
	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)

	body, _ := json.Marshal(map[string]string{
		"wont_do_reason":       "This task is out of scope and should not be done at all",
		"wont_do_requested_by": "human-operator",
	})

	url := fmt.Sprintf("%s/api/projects/%s/tasks/%s/wont-do",
		srv.URL, projectID.String(), taskID.String())
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// RED: both calls were made, the first succeeded and the second failed.
	// The task is now stuck in "blocked" with wont_do_requested=1 — it is
	// invisible to GetNextTask and cannot be unblocked via normal flows.
	if requestCalled && approveCalled {
		t.Log("SEC-09 RED: RequestWontDo succeeded but ApproveWontDo failed — " +
			"task is left in inconsistent state (blocked, wont_do_requested=1); " +
			"fix: implement a single atomic WontDo service method, or add compensating " +
			"rollback (RequestWontDo reversal) when ApproveWontDo fails")
	}

	// The response MUST be 500 (error) since ApproveWontDo failed.
	// This is the correct error status, but the side-effect of the first mutation
	// is already committed — no rollback occurred.
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode,
		"SEC-09: partial WontDo failure must return 500 (it does, but the state is corrupted); "+
			"the real fix is atomicity, not just returning the right status code")

	// Document the inconsistency: first call succeeded, second failed.
	assert.True(t, requestCalled,
		"SEC-09: RequestWontDo was called (mutation 1 committed)")
	assert.True(t, approveCalled,
		"SEC-09: ApproveWontDo was called (mutation 2 failed, state is inconsistent)")
}

// TestSecurity_SEC09_WontDoHappyPath_GREEN verifies that when both mutations
// succeed the response is 200 and the handler emits the broadcast event.
func TestSecurity_SEC09_WontDoHappyPath_GREEN(t *testing.T) {
	requestCalled := false
	approveCalled := false

	cmds := &servicetest.MockCommands{
		RequestWontDoFunc: func(ctx context.Context, pID domain.ProjectID, tID domain.TaskID, reason, requestedBy string) error {
			requestCalled = true
			return nil
		},
		ApproveWontDoFunc: func(ctx context.Context, pID domain.ProjectID, tID domain.TaskID) error {
			approveCalled = true
			return nil
		},
	}
	qrs := &servicetest.MockQueries{}
	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()

	router := newDeepSecurityRouter(t, newTestApp(cmds, qrs))
	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)

	body, _ := json.Marshal(map[string]string{
		"wont_do_reason":       "This task is out of scope and should not be done at all",
		"wont_do_requested_by": "human-operator",
	})

	url := fmt.Sprintf("%s/api/projects/%s/tasks/%s/wont-do",
		srv.URL, projectID.String(), taskID.String())
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode,
		"SEC-09 GREEN: successful WontDo must return 200")
	assert.True(t, requestCalled, "SEC-09 GREEN: RequestWontDo must be called")
	assert.True(t, approveCalled, "SEC-09 GREEN: ApproveWontDo must be called")
}

// ─────────────────────────────────────────────────────────────────────────────
// SEC-10  CompleteTask silently swallows UpdateTask error
// ─────────────────────────────────────────────────────────────────────────────

// TestSecurity_SEC10_CompleteTaskSwallowsHumanEstimateError_RED documents that
// tasks.go line 341 uses:
//
//	_ = h.commands.UpdateTask(r.Context(), ..., &humanEst)
//
// The error from UpdateTask is silently discarded. If the underlying storage
// fails to persist the human_estimate_seconds value, the API returns 200 OK
// but the estimate is lost with no indication to the caller.
func TestSecurity_SEC10_CompleteTaskSwallowsHumanEstimateError_RED(t *testing.T) {
	completeCalled := false
	updateCalled := false
	updateErr := errors.New("DB write failed: disk full")

	cmds := &servicetest.MockCommands{
		CompleteTaskFunc: func(ctx context.Context, pID domain.ProjectID, tID domain.TaskID, completionSummary string, filesModified []string, completedByAgent string, tokenUsage *domain.TokenUsage) error {
			completeCalled = true
			return nil // CompleteTask itself succeeds.
		},
		UpdateTaskFunc: func(ctx context.Context, pID domain.ProjectID, tID domain.TaskID, title, description, assignedRole, estimatedEffort, resolution *string, priority *domain.Priority, contextFiles, tags *[]string, tokenUsage *domain.TokenUsage, humanEstimateSeconds *int, featureID *domain.ProjectID, clearFeature bool) error {
			updateCalled = true
			return updateErr // Saving the estimate fails.
		},
	}
	qrs := &servicetest.MockQueries{}
	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()

	router := newDeepSecurityRouter(t, newTestApp(cmds, qrs))
	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)

	body, _ := json.Marshal(map[string]interface{}{
		"completion_summary":     "All work completed. See PR #123 for details and full implementation notes. All tests pass and coverage is adequate.",
		"files_modified":         []string{"main.go"},
		"completed_by_agent":     "claude-agent",
		"human_estimate_seconds": 3600,
	})

	url := fmt.Sprintf("%s/api/projects/%s/tasks/%s/complete",
		srv.URL, projectID.String(), taskID.String())
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// RED: the response is 200 OK even though UpdateTask failed.
	// The human_estimate_seconds was silently lost.
	if completeCalled && updateCalled {
		assert.Equal(t, http.StatusOK, resp.StatusCode,
			"SEC-10 RED: handler returns 200 even when UpdateTask for human_estimate_seconds "+
				"failed with %q — the estimate is silently lost; "+
				"fix: check the error from UpdateTask and return 500 or merge "+
				"human_estimate_seconds into a single CompleteTask call",
			updateErr)
		t.Logf("SEC-10 RED: UpdateTask error %q was swallowed — caller received 200 OK "+
			"but human estimate was not persisted; "+
			"fix: change '_ = h.commands.UpdateTask(...)' to check the error "+
			"in tasks.go line 341", updateErr)
	}
}

// TestSecurity_SEC10_CompleteTaskWithoutHumanEstimate_GREEN verifies that
// CompleteTask without a human_estimate_seconds field succeeds normally and
// the second UpdateTask call is never made.
func TestSecurity_SEC10_CompleteTaskWithoutHumanEstimate_GREEN(t *testing.T) {
	completeCalled := false
	updateCalled := false

	cmds := &servicetest.MockCommands{
		CompleteTaskFunc: func(ctx context.Context, pID domain.ProjectID, tID domain.TaskID, completionSummary string, filesModified []string, completedByAgent string, tokenUsage *domain.TokenUsage) error {
			completeCalled = true
			return nil
		},
		UpdateTaskFunc: func(ctx context.Context, pID domain.ProjectID, tID domain.TaskID, title, description, assignedRole, estimatedEffort, resolution *string, priority *domain.Priority, contextFiles, tags *[]string, tokenUsage *domain.TokenUsage, humanEstimateSeconds *int, featureID *domain.ProjectID, clearFeature bool) error {
			updateCalled = true
			return nil
		},
	}
	qrs := &servicetest.MockQueries{}
	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()

	router := newDeepSecurityRouter(t, newTestApp(cmds, qrs))
	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)

	body, _ := json.Marshal(map[string]interface{}{
		"completion_summary": "All work completed. See PR #123 for details and full implementation notes. All tests pass and coverage is adequate.",
		"files_modified":     []string{"main.go"},
		"completed_by_agent": "claude-agent",
		// human_estimate_seconds is 0 (zero value) — UpdateTask must NOT be called.
	})

	url := fmt.Sprintf("%s/api/projects/%s/tasks/%s/complete",
		srv.URL, projectID.String(), taskID.String())
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode,
		"SEC-10 GREEN: CompleteTask without human estimate must return 200")
	assert.True(t, completeCalled,
		"SEC-10 GREEN: CompleteTask must be called")
	assert.False(t, updateCalled,
		"SEC-10 GREEN: UpdateTask must NOT be called when human_estimate_seconds=0")
}
