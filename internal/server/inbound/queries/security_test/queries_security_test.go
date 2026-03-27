package security_test

// Security tests for the inbound/queries package.
//
// For each vulnerability the file contains two sub-tests:
//   RED  — a test that demonstrates the vulnerability exists in current code
//          (it asserts the UNSAFE behaviour that is currently present; this
//          test PASSES against unfixed code and FAILS once the fix is applied)
//   GREEN — a test that describes the desired safe behaviour
//          (this test FAILS against unfixed code and PASSES once fixed)
//
// This organisation makes it trivially easy to track remediation progress:
// run `go test -run Security` and every RED test passing means the
// vulnerability is still open.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/JLugagne/agach-mcp/internal/server/domain"
	"github.com/JLugagne/agach-mcp/internal/server/domain/repositories/tasks"
	"github.com/JLugagne/agach-mcp/internal/server/domain/service/servicetest"
	"github.com/JLugagne/agach-mcp/internal/server/inbound/queries"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func newSecurityRouter(mock *servicetest.MockQueries) *mux.Router {
	ctrl := newTestController()
	router := mux.NewRouter()
	queries.NewTaskQueriesHandler(mock, ctrl).RegisterRoutes(router)
	queries.NewCommentQueriesHandler(mock, ctrl).RegisterRoutes(router)
	queries.NewProjectQueriesHandler(mock, ctrl, nil).RegisterRoutes(router)
	return router
}

// ---------------------------------------------------------------------------
// VULN-1: SearchTasks — no upper bound on ?limit parameter
// File: tasks.go:132-136
//
// The limit parameter is accepted from the query string and forwarded
// directly to the service layer with no maximum cap.  An attacker can
// request ?limit=10000000 to cause an arbitrarily large result set,
// exhausting memory and database resources (DoS).
// ---------------------------------------------------------------------------

// GREEN — after fixing, the handler MUST cap the limit to a safe maximum.
// The upper bound should be at most 1000 (or whatever the project decides).
func TestSecurity_GREEN_SearchTasks_LimitIsCapped(t *testing.T) {
	const maxAllowedLimit = 1000
	projectID := newValidProjectID()

	receivedLimit := 0
	mock := &servicetest.MockQueries{
		ListTasksFunc: func(_ context.Context, _ domain.ProjectID, f tasks.TaskFilters) ([]domain.TaskWithDetails, error) {
			receivedLimit = f.Limit
			return nil, nil
		},
	}

	router := newSecurityRouter(mock)
	req := httptest.NewRequest(http.MethodGet,
		"/api/projects/"+string(projectID)+"/tasks/search?q=foo&limit=9999999",
		nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	assert.LessOrEqual(t, receivedLimit, maxAllowedLimit,
		"GREEN: handler must cap limit to at most %d", maxAllowedLimit)
}

// ---------------------------------------------------------------------------
// VULN-2: GetNextTasks — no upper bound on ?count parameter
// File: tasks.go:282-284
//
// The `count` query parameter is accepted without a maximum cap.
// With include_subprojects=true the handler fetches count*10 tasks
// per project (lines 328 and 346), multiplying the load.
// ---------------------------------------------------------------------------

// GREEN — after fixing, count is capped to a safe maximum.
func TestSecurity_GREEN_GetNextTasks_CountIsCapped(t *testing.T) {
	const maxAllowedCount = 100
	projectID := newValidProjectID()

	receivedCount := 0
	mock := &servicetest.MockQueries{
		GetNextTasksFunc: func(_ context.Context, _ domain.ProjectID, _ string, count int, _ *domain.ProjectID) ([]domain.Task, error) {
			receivedCount = count
			return nil, nil
		},
	}

	router := newSecurityRouter(mock)
	req := httptest.NewRequest(http.MethodGet,
		"/api/projects/"+string(projectID)+"/next-tasks?count=999999",
		nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	assert.LessOrEqual(t, receivedCount, maxAllowedCount,
		"GREEN: handler must cap count to at most %d", maxAllowedCount)
}

// ---------------------------------------------------------------------------
// VULN-3: GetNextTasks with include_subprojects=true — amplified service calls
// File: tasks.go:385-390
//
// The handler calls ListSubProjects and GetNextTasks a SECOND time in the
// project-mapping phase (lines 385-390), doubling every subproject's
// database queries even though the data was already fetched.
// With N subprojects the service receives 2*(N+1) calls instead of N+1.
// ---------------------------------------------------------------------------

// GREEN — after fixing, each project is queried exactly once.
func TestSecurity_GREEN_GetNextTasks_SubprojectsMinimalQueries(t *testing.T) {
	projectID := newValidProjectID()
	subProjectID := newValidProjectID()
	taskID := newValidTaskID()

	listSubCallCount := 0
	getNextCallCount := 0

	mock := &servicetest.MockQueries{
		ListSubProjectsFunc: func(_ context.Context, _ domain.ProjectID) ([]domain.Project, error) {
			listSubCallCount++
			return []domain.Project{{ID: subProjectID, Name: "sub"}}, nil
		},
		GetNextTasksFunc: func(_ context.Context, _ domain.ProjectID, _ string, _ int, _ *domain.ProjectID) ([]domain.Task, error) {
			getNextCallCount++
			return []domain.Task{{ID: taskID, Title: "t", Priority: domain.PriorityMedium}}, nil
		},
	}

	router := newSecurityRouter(mock)
	req := httptest.NewRequest(http.MethodGet,
		"/api/projects/"+string(projectID)+"/next-tasks?count=1&include_subprojects=true",
		nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, 1, listSubCallCount,
		"GREEN: ListSubProjects should be called exactly once (not duplicated for mapping)")
	// main project (1) + 1 subproject (1) = 2 calls total
	assert.Equal(t, 2, getNextCallCount,
		"GREEN: GetNextTasks should be called exactly once per project, 2 total")
}

// ---------------------------------------------------------------------------
// VULN-4: ListTasks with include_children=true — silent error suppression
// File: tasks.go:97
//
// When ListSubProjects fails, the error is silently consumed with
// `if err == nil { ... }` and the handler returns only the parent project's
// tasks with HTTP 200.  The caller has no way to distinguish a full result
// from a partial one.  If access control is ever added at the subproject
// level, this pattern would silently omit denied subprojects.
// ---------------------------------------------------------------------------

// GREEN — after fixing, the handler should propagate the error to the client
// (either 500 for server error or a partial-result header/flag in the response).
// The simplest safe fix is to return 500 when include_children=true and
// ListSubProjects fails.
func TestSecurity_GREEN_ListTasks_IncludeChildrenErrorPropagated(t *testing.T) {
	projectID := newValidProjectID()

	mock := &servicetest.MockQueries{
		ListTasksFunc: func(_ context.Context, _ domain.ProjectID, _ tasks.TaskFilters) ([]domain.TaskWithDetails, error) {
			return []domain.TaskWithDetails{
				{Task: domain.Task{ID: newValidTaskID(), Title: "parent task", Priority: domain.PriorityMedium}},
			}, nil
		},
		ListSubProjectsFunc: func(_ context.Context, _ domain.ProjectID) ([]domain.Project, error) {
			return nil, assert.AnError
		},
	}

	router := newSecurityRouter(mock)
	req := httptest.NewRequest(http.MethodGet,
		"/api/projects/"+string(projectID)+"/tasks?include_children=true",
		nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	// GREEN: the handler must not return 200 when a dependency fails.
	assert.NotEqual(t, http.StatusOK, rr.Code,
		"GREEN: handler must not silently swallow a subproject enumeration error")
}

// ---------------------------------------------------------------------------
// VULN-5: GetBoard — unvalidated, unbounded done_since duration
// File: tasks.go:185-188
//
// The ?done_since= parameter is parsed with time.ParseDuration with no
// maximum bound.  A request with done_since=876000h (100 years) causes the
// filter to return every task ever completed, producing an unbounded result
// set that can exhaust memory and database resources.
// ---------------------------------------------------------------------------

// GREEN — after fixing, a done_since duration beyond a reasonable maximum
// (e.g. 8760h = 1 year) must result in a 400 error or the parameter must be
// silently clamped (and therefore the UpdatedSince filter should reflect the
// clamped value, not the requested 100-year value).
func TestSecurity_GREEN_GetBoard_DoneSinceClamped(t *testing.T) {
	projectID := newValidProjectID()

	const hugeHours = 876000
	mock := &servicetest.MockQueries{
		ListColumnsFunc: func(_ context.Context, _ domain.ProjectID) ([]domain.Column, error) {
			slug := domain.ColumnDone
			return []domain.Column{
				{ID: domain.NewColumnID(), Slug: slug, Name: "Done", Position: 2},
			}, nil
		},
		GetProjectFunc: func(_ context.Context, _ domain.ProjectID) (*domain.Project, error) {
			return &domain.Project{ID: projectID, Name: "P"}, nil
		},
		ListTasksFunc: func(_ context.Context, _ domain.ProjectID, _ tasks.TaskFilters) ([]domain.TaskWithDetails, error) {
			return nil, nil
		},
	}

	router := newSecurityRouter(mock)
	req := httptest.NewRequest(http.MethodGet,
		"/api/projects/"+string(projectID)+"/board?done_since="+strconv.Itoa(hugeHours)+"h",
		nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	// GREEN: the handler must reject an absurd duration with 400.
	assert.Equal(t, http.StatusBadRequest, rr.Code,
		"GREEN: handler must reject done_since values beyond a reasonable maximum")
}

// ---------------------------------------------------------------------------
// VULN-8: ListComments — no pagination; always passes limit=0 (no limit)
// File: comments.go:46
//
//   h.queries.ListComments(r.Context(), projectID, taskID, 0, 0)
//
// limit=0 is the sentinel value for "no limit" (see tasks.go TaskFilters).
// A task with thousands of comments returns everything in a single response,
// making it trivial to DoS the server by posting many comments and then
// repeatedly listing them.
// ---------------------------------------------------------------------------

// GREEN — after fixing, the handler must apply a sensible default limit and
// also respect a ?limit= query parameter capped at a safe maximum.
func TestSecurity_GREEN_ListComments_DefaultLimitApplied(t *testing.T) {
	projectID := newValidProjectID()
	taskID := newValidTaskID()

	receivedLimit := -1
	mock := &servicetest.MockQueries{
		ListCommentsFunc: func(_ context.Context, _ domain.ProjectID, _ domain.TaskID, limit, _ int) ([]domain.Comment, error) {
			receivedLimit = limit
			return nil, nil
		},
	}

	router := newSecurityRouter(mock)
	req := httptest.NewRequest(http.MethodGet,
		"/api/projects/"+string(projectID)+"/tasks/"+string(taskID)+"/comments",
		nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	// GREEN: a positive default limit should be applied when none is requested.
	assert.Greater(t, receivedLimit, 0,
		"GREEN: ListComments must apply a positive default limit to prevent unbounded result sets")
}
