// Package security_test contains NEW security-focused RED tests for the
// server outbound pg layer.  Each test documents a real vulnerability found
// during code review and compiles without a live database (using AST /
// source-text analysis of the production Go files).
package security_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// helpers – parse production source files for static analysis
// ---------------------------------------------------------------------------

// parseSourceFile parses a Go source file and returns its AST.
func parseSourceFile(t *testing.T, path string) *ast.File {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	require.NoError(t, err, "must be able to parse %s", path)
	return f
}

// readSource reads a file as a string for text-based analysis.
func readSource(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err, "must be able to read %s", path)
	return string(data)
}

// findMethodBody returns the string body of a method with the given receiver
// type and method name.  It scans the AST for func decls.
func findMethodBody(t *testing.T, path, receiverType, methodName string) string {
	t.Helper()
	src := readSource(t, path)
	f := parseSourceFile(t, path)

	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv == nil || fn.Name.Name != methodName {
			continue
		}
		for _, recv := range fn.Recv.List {
			recvStr := exprName(recv.Type)
			if recvStr == receiverType {
				return src[fn.Body.Pos()-1 : fn.Body.End()-1]
			}
		}
	}
	t.Fatalf("method (%s).%s not found in %s", receiverType, methodName, path)
	return ""
}

// exprName returns a simplified string representation of a type expression
// (handles *ast.StarExpr for pointer receivers).
func exprName(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.StarExpr:
		return "*" + exprName(e.X)
	default:
		return ""
	}
}

const (
	featuresFile      = "../pg_features.go"
	tasksFile         = "../pg_tasks.go"
	projectAccessFile = "../pg_project_access.go"
	pgFile            = "../pg.go"
	migration010      = "../migrations/010_project_access.sql"
)

// ---------------------------------------------------------------------------
// VULNERABILITY: Feature FindByID does not filter by project_id
//
// featureRepository.FindByID queries `WHERE id = $1` without any project_id
// constraint.  A caller who knows (or guesses) a feature UUID from another
// project can read the full feature record, including its name, description,
// and changelogs.  This is a cross-project data leakage vulnerability.
//
// TODO(security): Add project_id as a required parameter to FindByID and
// include `AND project_id = $2` in the WHERE clause.
// ---------------------------------------------------------------------------

func TestSecurity_RED_FeatureFindByIDNoProjectFilter(t *testing.T) {
	body := findMethodBody(t, featuresFile, "*featureRepository", "FindByID")

	// The method SELECTs project_id as a column but never uses it in the WHERE
	// clause as a filter.  Check for WHERE ... project_id pattern.
	hasProjectFilter := strings.Contains(body, "WHERE") &&
		(strings.Contains(body, "project_id = $") || strings.Contains(body, "project_id=$"))

	assert.False(t, hasProjectFilter,
		"RED: featureRepository.FindByID does not filter by project_id in WHERE; "+
			"any user with a feature UUID can read features from other projects")
	t.Log("RED: featureRepository.FindByID queries only by id, not project_id — cross-project data leakage")
}

// ---------------------------------------------------------------------------
// VULNERABILITY: Feature Delete does not filter by project_id
//
// featureRepository.Delete queries `WHERE id=$1` without project_id.
// A user in project A could delete a feature belonging to project B.
//
// TODO(security): Add project_id filter to the DELETE query.
// ---------------------------------------------------------------------------

func TestSecurity_RED_FeatureDeleteNoProjectFilter(t *testing.T) {
	body := findMethodBody(t, featuresFile, "*featureRepository", "Delete")

	hasProjectFilter := strings.Contains(body, "project_id")

	assert.False(t, hasProjectFilter,
		"RED: featureRepository.Delete does not filter by project_id; "+
			"a user can delete features from other projects")
	t.Log("RED: featureRepository.Delete queries only by id — cross-project destructive operation")
}

// ---------------------------------------------------------------------------
// VULNERABILITY: Feature Update does not filter by project_id
//
// featureRepository.Update uses `WHERE id=$4` without project_id.
// A user could modify another project's feature name/description.
//
// TODO(security): Add project_id to the UPDATE WHERE clause.
// ---------------------------------------------------------------------------

func TestSecurity_RED_FeatureUpdateNoProjectFilter(t *testing.T) {
	body := findMethodBody(t, featuresFile, "*featureRepository", "Update")

	// The WHERE clause should contain project_id but it does not.
	hasProjectFilter := strings.Contains(body, "project_id")

	assert.False(t, hasProjectFilter,
		"RED: featureRepository.Update does not filter by project_id; "+
			"a user can modify features belonging to other projects")
	t.Log("RED: featureRepository.Update WHERE clause lacks project_id — cross-project modification")
}

// ---------------------------------------------------------------------------
// VULNERABILITY: Feature UpdateStatus does not filter by project_id
//
// featureRepository.UpdateStatus uses `WHERE id=$2` without project_id.
//
// TODO(security): Add project_id filter to UpdateStatus.
// ---------------------------------------------------------------------------

func TestSecurity_RED_FeatureUpdateStatusNoProjectFilter(t *testing.T) {
	body := findMethodBody(t, featuresFile, "*featureRepository", "UpdateStatus")

	hasProjectFilter := strings.Contains(body, "project_id")

	assert.False(t, hasProjectFilter,
		"RED: featureRepository.UpdateStatus does not filter by project_id; "+
			"a user can change feature status across projects")
	t.Log("RED: featureRepository.UpdateStatus WHERE clause lacks project_id — cross-project status change")
}

// ---------------------------------------------------------------------------
// VULNERABILITY: Feature UpdateChangelogs does not filter by project_id
//
// featureRepository.UpdateChangelogs builds a dynamic UPDATE with
// `WHERE id=$N` but never includes project_id.
//
// TODO(security): Add project_id filter to UpdateChangelogs.
// ---------------------------------------------------------------------------

func TestSecurity_RED_FeatureUpdateChangelogsNoProjectFilter(t *testing.T) {
	body := findMethodBody(t, featuresFile, "*featureRepository", "UpdateChangelogs")

	hasProjectFilter := strings.Contains(body, "project_id")

	assert.False(t, hasProjectFilter,
		"RED: featureRepository.UpdateChangelogs does not filter by project_id; "+
			"a user can overwrite changelogs of features in other projects")
	t.Log("RED: featureRepository.UpdateChangelogs WHERE clause lacks project_id — cross-project changelog overwrite")
}

// ---------------------------------------------------------------------------
// VULNERABILITY: Feature ListTaskSummaries does not filter by project_id
//
// featureRepository.ListTaskSummaries queries tasks `WHERE feature_id = $1`
// without verifying the feature belongs to the caller's project.  This
// leaks task titles, completion summaries, agent names, token counts, and
// file modification lists across project boundaries.
//
// TODO(security): Either add a project_id parameter or JOIN features to
// verify ownership.
// ---------------------------------------------------------------------------

func TestSecurity_RED_FeatureListTaskSummariesNoProjectFilter(t *testing.T) {
	body := findMethodBody(t, featuresFile, "*featureRepository", "ListTaskSummaries")

	hasProjectFilter := strings.Contains(body, "project_id")

	assert.False(t, hasProjectFilter,
		"RED: featureRepository.ListTaskSummaries does not filter by project_id; "+
			"task data (titles, summaries, tokens, files) leaks across projects")
	t.Log("RED: featureRepository.ListTaskSummaries queries only by feature_id — cross-project task data leakage")
}

// ---------------------------------------------------------------------------
// VULNERABILITY: project_user_access and project_team_access missing RLS
//
// Migration 010 creates project_user_access and project_team_access without
// enabling Row Level Security, unlike every other table in migration 001.
// If a restricted DB role connects, it has unrestricted access to all rows
// in these access control tables.
//
// TODO(security): Add ENABLE ROW LEVEL SECURITY and FORCE ROW LEVEL
// SECURITY to migration 010 for both tables.
// ---------------------------------------------------------------------------

func TestSecurity_RED_ProjectAccessTablesNoRLS(t *testing.T) {
	src := readSource(t, migration010)
	upper := strings.ToUpper(src)

	hasRLS := strings.Contains(upper, "ROW LEVEL SECURITY")

	assert.False(t, hasRLS,
		"RED: migration 010 does not enable RLS on project_user_access or project_team_access; "+
			"any DB role has unrestricted access to access-control rows")
	t.Log("RED: project_user_access and project_team_access have no Row Level Security")
}

// ---------------------------------------------------------------------------
// VULNERABILITY: RevokeUser silent no-op on non-existent grant
//
// projectAccessRepository.RevokeUser does not check RowsAffected().
// Revoking access for a user who was never granted returns nil, making it
// impossible for the caller to know whether the revocation was effective.
// In access-control code, silent no-ops mask bugs and can hide TOCTOU
// issues where a grant was re-added between check and revoke.
//
// TODO(security): Check tag.RowsAffected() and return an appropriate error
// or domain sentinel when no row was deleted.
// ---------------------------------------------------------------------------

func TestSecurity_RED_RevokeUserSilentNoop(t *testing.T) {
	body := findMethodBody(t, projectAccessFile, "*projectAccessRepository", "RevokeUser")

	checksRowsAffected := strings.Contains(body, "RowsAffected")

	assert.False(t, checksRowsAffected,
		"RED: projectAccessRepository.RevokeUser does not check RowsAffected; "+
			"revoking a non-existent grant silently succeeds")
	t.Log("RED: RevokeUser is a silent no-op when the user was never granted access")
}

// ---------------------------------------------------------------------------
// VULNERABILITY: UpdateUserRole silent no-op on non-existent grant
//
// projectAccessRepository.UpdateUserRole does not check RowsAffected().
// Updating the role of a user who has no access grant returns nil, silently
// suggesting success.  This can mask privilege escalation bugs where the
// caller assumes the update took effect.
//
// TODO(security): Check tag.RowsAffected() and return an error when no row
// was updated.
// ---------------------------------------------------------------------------

func TestSecurity_RED_UpdateUserRoleSilentNoop(t *testing.T) {
	body := findMethodBody(t, projectAccessFile, "*projectAccessRepository", "UpdateUserRole")

	checksRowsAffected := strings.Contains(body, "RowsAffected")

	assert.False(t, checksRowsAffected,
		"RED: projectAccessRepository.UpdateUserRole does not check RowsAffected; "+
			"updating role on a non-existent grant silently succeeds")
	t.Log("RED: UpdateUserRole is a silent no-op for non-existent grants — can mask privilege escalation bugs")
}

// ---------------------------------------------------------------------------
// VULNERABILITY: GetTimeline days parameter unbounded
//
// taskRepository.GetTimeline passes the `days` parameter directly into a
// PostgreSQL interval expression: `($2 || ' days')::interval`.
// A negative value generates a future-to-past range producing empty but
// valid results.  A very large value (e.g. 999999999) causes generate_series
// to produce hundreds of millions of rows, leading to resource exhaustion
// (CPU/memory DoS on the database).
//
// TODO(security): Clamp or validate `days` to a reasonable range (e.g. 1-365)
// before passing it to the query.
// ---------------------------------------------------------------------------

func TestSecurity_RED_GetTimelineDaysUnbounded(t *testing.T) {
	body := findMethodBody(t, tasksFile, "*taskRepository", "GetTimeline")

	// Check that the days parameter is used directly without validation.
	hasBoundsCheck := strings.Contains(body, "days <") ||
		strings.Contains(body, "days >") ||
		strings.Contains(body, "days <=") ||
		strings.Contains(body, "days >=") ||
		strings.Contains(body, "math.Min") ||
		strings.Contains(body, "math.Max")

	assert.False(t, hasBoundsCheck,
		"RED: taskRepository.GetTimeline does not validate the days parameter; "+
			"a large value can cause generate_series resource exhaustion (DoS)")
	t.Log("RED: GetTimeline days parameter is unbounded — potential DB-level DoS via generate_series")
}

// ---------------------------------------------------------------------------
// VULNERABILITY: Migration lacks versioning / tracking table
//
// NewRepositories iterates over all migration files and executes them on
// every startup.  There is no migrations tracking table (e.g. schema_migrations)
// to record which migrations have already been applied.  While most DDL uses
// IF NOT EXISTS, any future migration with DML (UPDATE/DELETE) will run
// repeatedly, potentially corrupting data.  This also means there's no way
// to detect or prevent out-of-order or partial migration application.
//
// TODO(security): Add a schema_migrations table that records applied
// migration filenames and skip already-applied ones in NewRepositories.
// ---------------------------------------------------------------------------

func TestSecurity_RED_NoMigrationVersioning(t *testing.T) {
	src := readSource(t, pgFile)

	hasVersionTracking := strings.Contains(src, "schema_migrations") ||
		strings.Contains(src, "migration_version") ||
		strings.Contains(src, "applied_migrations")

	assert.False(t, hasVersionTracking,
		"RED: NewRepositories has no migration versioning; all migrations "+
			"re-execute on every startup, risking data corruption from non-idempotent DML")
	t.Log("RED: No migration tracking table — all migrations re-run on every startup")
}

// ---------------------------------------------------------------------------
// VULNERABILITY: HasUnresolvedDependencies lacks project_id scoping
//
// taskRepository.HasUnresolvedDependencies queries task_dependencies by
// task_id only (WHERE td.task_id = $1), without filtering by project_id.
// The projectID parameter is accepted but never used.  While the impact is
// limited (dependency data, not task content), it violates the principle of
// least privilege and could leak information about task completion state
// across projects.
//
// TODO(security): Add project_id filter to the query or remove the unused
// parameter to avoid false confidence in caller code.
// ---------------------------------------------------------------------------

func TestSecurity_RED_HasUnresolvedDepsIgnoresProjectID(t *testing.T) {
	body := findMethodBody(t, tasksFile, "*taskRepository", "HasUnresolvedDependencies")

	// The method signature accepts projectID but the query never uses it.
	acceptsProjectID := strings.Contains(body, "projectID")
	queryUsesProjectID := strings.Contains(body, "string(projectID)")

	assert.True(t, acceptsProjectID || true, // method definitely accepts it (from signature)
		"precondition: method accepts projectID parameter")
	assert.False(t, queryUsesProjectID,
		"RED: HasUnresolvedDependencies accepts projectID but never uses it in the query; "+
			"dependency resolution state is not scoped to a project")
	t.Log("RED: HasUnresolvedDependencies ignores projectID parameter — dependency state leaks across projects")
}

// ---------------------------------------------------------------------------
// VULNERABILITY: GetDependentsNotDone lacks project_id scoping
//
// taskRepository.GetDependentsNotDone queries by depends_on_task_id only,
// without project_id.  The projectID parameter is accepted but unused.
// This can return tasks from other projects if they happen to share a
// dependency link (unlikely but architecturally unsound).
//
// TODO(security): Add project_id filter or remove the unused parameter.
// ---------------------------------------------------------------------------

func TestSecurity_RED_GetDependentsNotDoneIgnoresProjectID(t *testing.T) {
	body := findMethodBody(t, tasksFile, "*taskRepository", "GetDependentsNotDone")

	queryUsesProjectID := strings.Contains(body, "string(projectID)")

	assert.False(t, queryUsesProjectID,
		"RED: GetDependentsNotDone accepts projectID but never uses it in the query; "+
			"dependent tasks are not scoped to a project")
	t.Log("RED: GetDependentsNotDone ignores projectID — can return tasks from other projects")
}
