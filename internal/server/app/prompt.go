package app

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"github.com/JLugagne/agach-mcp/internal/server/domain"
)

// FileReader abstracts filesystem reads for prompt rendering (testability).
type FileReader interface {
	ReadFile(path string) ([]byte, error)
}

type osFileReader struct{}

func (osFileReader) ReadFile(path string) ([]byte, error) { return os.ReadFile(path) }

func resolveFileReader(readers []FileReader) FileReader {
	if len(readers) > 0 && readers[0] != nil {
		return readers[0]
	}
	return osFileReader{}
}

// RenderPrompt renders the assigned role's prompt_template for the given task.
// It resolves all {{slot}} variables with live data from the project, task, role,
// dependencies, and optionally reads context_files from disk.
func (a *App) RenderPrompt(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) (string, error) {
	logger := a.TaskService.logger.WithContext(ctx).WithFields(map[string]interface{}{
		"projectID": projectID,
		"taskID":    taskID,
	})

	// Load task
	task, err := a.TaskService.tasks.FindByID(ctx, projectID, taskID)
	if err != nil || task == nil {
		return "", errors.Join(domain.ErrTaskNotFound, err)
	}

	// Load project
	project, err := a.ProjectService.projects.FindByID(ctx, projectID)
	if err != nil || project == nil {
		return "", errors.Join(domain.ErrProjectNotFound, err)
	}

	// Load role template — try project-scoped first, fall back to global
	var role *domain.Agent
	if task.AssignedRole != "" {
		role, _ = a.AgentService.agents.FindBySlugInProject(ctx, projectID, task.AssignedRole)
		if role == nil {
			role, _ = a.AgentService.agents.FindBySlug(ctx, task.AssignedRole)
		}
	}

	promptTemplate := ""
	if role != nil {
		promptTemplate = role.PromptTemplate
	}

	if promptTemplate == "" {
		logger.Warn("role has no prompt_template")
		return "", nil
	}

	// Load dependency tasks
	depTasks, err := a.DependencyService.dependencies.GetDependencyContext(ctx, projectID, taskID)
	if err != nil {
		depTasks = nil
	}

	// Build dependency maps keyed by assigned_role
	depByRole := make(map[string]domain.DependencyContext)
	var depSummaries []string
	for _, dep := range depTasks {
		depSummaries = append(depSummaries, dep.CompletionSummary)
	}

	// We need the full Task objects for role-keyed access — fetch them
	fullDepTasks, err := a.DependencyService.dependencies.List(ctx, projectID, taskID)
	if err == nil {
		for _, depRef := range fullDepTasks {
			depTask, ferr := a.TaskService.tasks.FindByID(ctx, projectID, depRef.DependsOnTaskID)
			if ferr == nil && depTask != nil && depTask.AssignedRole != "" {
				depByRole[depTask.AssignedRole] = domain.DependencyContext{
					TaskID:            depTask.ID,
					Title:             depTask.Title,
					CompletionSummary: depTask.CompletionSummary,
					FilesModified:     depTask.FilesModified,
				}
			}
		}
	}

	// Read context files from disk (path traversal protected)
	contextFilesContent := readContextFiles(task.ContextFiles, a.fileReader)
	contextFilesSignatures := extractSignatures(task.ContextFiles, a.fileReader)

	// Infer test command from role tech stack
	testCommand := inferTestCommand(role)

	// Extract test info from red dependency (assumes red agent wrote failure_output in completion_summary)
	testFile, testName, failureOutput := extractTestInfo(depByRole)

	// Build template data — only expose safe, allow-listed keys
	data := map[string]interface{}{
		"task": map[string]interface{}{
			"title":            task.Title,
			"description":      task.Description,
			"summary":          task.Summary,
			"priority":         string(task.Priority),
			"assigned_role":    task.AssignedRole,
			"estimated_effort": task.EstimatedEffort,
			"tags":             task.Tags,
			"test_command":     testCommand,
			"test_file":        testFile,
			"test_name":        testName,
		},
		"project": map[string]interface{}{
			"name": project.Name,
		},
		"dependencies": map[string]interface{}{
			"all":   strings.Join(depSummaries, "\n\n"),
			"red":   getDepField(depByRole, "red", "completion_summary"),
			"green": getDepField(depByRole, "green", "completion_summary"),
		},
		"dependency":               buildDepAccessor(depByRole),
		"context_files":            contextFilesContent,
		"context_files_signatures": contextFilesSignatures,
	}

	// Add failure_output as a special dependency field
	if failureOutput != "" {
		if depMap, ok := data["dependency"].(map[string]map[string]string); ok {
			if _, ok := depMap["red"]; !ok {
				depMap["red"] = make(map[string]string)
			}
			depMap["red"]["failure_output"] = failureOutput
		}
	}

	// Parse and execute Go template with restricted data access
	rendered, err := renderTemplate(promptTemplate, data)
	if err != nil {
		logger.WithError(err).Error("failed to render prompt template")
		return "", fmt.Errorf("prompt template error: %w", err)
	}

	return rendered, nil
}

// renderTemplate converts {{slot.field}} notation to Go template syntax and executes it.
// Only allow-listed top-level keys are accessible in the template.
func renderTemplate(tmplStr string, data map[string]interface{}) (string, error) {
	converted := convertDotNotation(tmplStr)

	funcMap := template.FuncMap{
		"join": strings.Join,
	}

	tmpl, err := template.New("prompt").Funcs(funcMap).Parse(converted)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// convertDotNotation converts {{key.subkey.field}} to Go template .key.subkey.field
// for map access. For deeply nested dynamic keys (dependency.{role}.{field}),
// we use index calls.
var dotDepPattern = regexp.MustCompile(`\{\{(dependency\.([^.}\s]+)\.([^}\s]+))\}\}`)

func convertDotNotation(s string) string {
	// Convert {{dependency.role.field}} → {{index (index .dependency "role") "field"}}
	s = dotDepPattern.ReplaceAllStringFunc(s, func(match string) string {
		sub := dotDepPattern.FindStringSubmatch(match)
		if len(sub) < 4 {
			return match
		}
		role := sub[2]
		field := sub[3]
		return fmt.Sprintf(`{{index (index .dependency "%s") "%s"}}`, role, field)
	})
	// Convert top-level {{key.subkey}} → {{.key.subkey}} (standard Go template)
	s = regexp.MustCompile(`\{\{([a-zA-Z_][a-zA-Z0-9_]*\.[a-zA-Z_][a-zA-Z0-9_.]*)\}\}`).ReplaceAllString(s, `{{.$1}}`)
	// Convert top-level bare slot {{context_files}} → {{.context_files}}
	s = regexp.MustCompile(`\{\{([a-zA-Z_][a-zA-Z0-9_]*)\}\}`).ReplaceAllString(s, `{{.$1}}`)
	return s
}

// buildDepAccessor builds a map[role]map[field]string for template access.
func buildDepAccessor(depByRole map[string]domain.DependencyContext) map[string]map[string]string {
	result := make(map[string]map[string]string)
	for role, dep := range depByRole {
		result[role] = map[string]string{
			"title":              dep.Title,
			"completion_summary": dep.CompletionSummary,
			"task_id":            string(dep.TaskID),
			"files_modified":     strings.Join(dep.FilesModified, ", "),
			"failure_output":     "",
		}
	}
	return result
}

func getDepField(depByRole map[string]domain.DependencyContext, role, field string) string {
	dep, ok := depByRole[role]
	if !ok {
		return ""
	}
	switch field {
	case "completion_summary":
		return dep.CompletionSummary
	case "title":
		return dep.Title
	}
	return ""
}

// readContextFiles reads file contents from disk, returning a combined string.
// Absolute paths and paths with traversal sequences are rejected to prevent path traversal attacks.
func readContextFiles(paths []string, readers ...FileReader) string {
	fr := resolveFileReader(readers)
	var parts []string
	for _, p := range paths {
		if filepath.IsAbs(p) {
			continue
		}
		if strings.Contains(p, "..") {
			continue
		}
		content, err := fr.ReadFile(p)
		if err != nil {
			parts = append(parts, fmt.Sprintf("// [could not read %s: %v]\n", filepath.Base(p), err))
			continue
		}
		parts = append(parts, fmt.Sprintf("// File: %s\n%s", p, string(content)))
	}
	return strings.Join(parts, "\n\n")
}

// extractSignatures reads Go files and extracts function/type signatures only.
// Absolute paths and paths with traversal sequences are rejected to prevent path traversal attacks.
func extractSignatures(paths []string, readers ...FileReader) string {
	fr := resolveFileReader(readers)
	var sigs []string
	for _, p := range paths {
		if !strings.HasSuffix(p, ".go") {
			continue
		}
		if filepath.IsAbs(p) {
			continue
		}
		if strings.Contains(p, "..") {
			continue
		}
		content, err := fr.ReadFile(p)
		if err != nil {
			continue
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, p, content, 0)
		if err != nil {
			continue
		}
		var fileSigs []string
		for _, decl := range f.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				fileSigs = append(fileSigs, fmt.Sprintf("func %s", d.Name.Name))
			case *ast.GenDecl:
				for _, spec := range d.Specs {
					if ts, ok := spec.(*ast.TypeSpec); ok {
						fileSigs = append(fileSigs, fmt.Sprintf("type %s", ts.Name.Name))
					}
				}
			}
		}
		if len(fileSigs) > 0 {
			sigs = append(sigs, fmt.Sprintf("// %s\n%s", filepath.Base(p), strings.Join(fileSigs, "\n")))
		}
	}
	return strings.Join(sigs, "\n\n")
}

// inferTestCommand tries to determine the test command from the role tech stack.
func inferTestCommand(role *domain.Agent) string {
	if role != nil {
		for _, tech := range role.TechStack {
			switch strings.ToLower(tech) {
			case "go", "golang":
				return "go test ./..."
			case "node", "typescript", "javascript":
				return "npm test"
			case "python":
				return "pytest"
			case "rust":
				return "cargo test"
			}
		}
	}
	return "go test ./..."
}

// extractTestInfo tries to parse test file/name from red dependency completion summary.
func extractTestInfo(depByRole map[string]domain.DependencyContext) (testFile, testName, failureOutput string) {
	red, ok := depByRole["red"]
	if !ok {
		return "", "", ""
	}
	summary := red.CompletionSummary

	fileRe := regexp.MustCompile(`(?i)(?:file:|test file:)\s*(\S+_test\.go)`)
	if m := fileRe.FindStringSubmatch(summary); len(m) > 1 {
		testFile = m[1]
	}

	funcRe := regexp.MustCompile(`(?i)(?:function:|test:|func:)\s*(Test\w+)`)
	if m := funcRe.FindStringSubmatch(summary); len(m) > 1 {
		testName = m[1]
	}

	failRe := regexp.MustCompile(`(?is)(?:failure output:|--- fail)(.+)`)
	if m := failRe.FindStringSubmatch(summary); len(m) > 1 {
		failureOutput = strings.TrimSpace(m[1])
	}

	return testFile, testName, failureOutput
}
