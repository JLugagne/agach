package app

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	pkgserver "github.com/JLugagne/agach-mcp/pkg/server"
)

const maxCopyBytes = 512 * 1024 // 512 KB cap for agent/skill .md files

var invalidSlugCharRe = regexp.MustCompile(`[^a-zA-Z0-9_-]+`)

// AgentDef holds the parsed frontmatter of a claude agent file
type AgentDef struct {
	Slug        string // filename without .md
	Name        string // from frontmatter "name:"
	Description string // from frontmatter "description:"
	SourcePath  string // absolute path to the .md file
	IsLocal     bool   // true if found in workdir/.claude/agents/
}

// SetupOptions controls what gets copied/synced at project creation
type SetupOptions struct {
	CopyAgents bool
	CopySkills bool
	SyncRoles  bool
}

// SetupResult holds the outcome of a project setup
type SetupResult struct {
	AgentsCopied int
	SkillsCopied int
	RolesSynced  int
	Errors       []string
}

// DiscoverAgents returns all agent definitions available:
// first local (workdir/.claude/agents/), then global (~/.claude/agents/).
// Local agents override global ones with the same slug.
// workDir must be an absolute canonical path (filepath.Clean(workDir) == workDir).
func DiscoverAgents(workDir string) []AgentDef {
	if !filepath.IsAbs(workDir) || filepath.Clean(workDir) != workDir {
		return nil
	}
	globalDir := filepath.Join(userHomeClaudeDir(), "agents")
	localDir := filepath.Join(workDir, ".claude", "agents")

	seen := map[string]bool{}
	var agents []AgentDef

	// Local first (higher priority)
	for _, def := range readAgentsDir(localDir, true) {
		seen[def.Slug] = true
		agents = append(agents, def)
	}
	// Global — skip slugs already seen locally
	for _, def := range readAgentsDir(globalDir, false) {
		if !seen[def.Slug] {
			seen[def.Slug] = true
			agents = append(agents, def)
		}
	}
	return agents
}

// SetupProject copies agents/skills into workDir and syncs project roles.
func (a *App) SetupProject(projectID, workDir string, opts SetupOptions) SetupResult {
	result := SetupResult{}

	globalDir := userHomeClaudeDir()
	localAgentsDir := filepath.Join(workDir, ".claude", "agents")
	localSkillsDir := filepath.Join(workDir, ".claude", "skills")

	if opts.CopyAgents {
		src := filepath.Join(globalDir, "agents")
		n, errs := copyDir(src, localAgentsDir, "*.md")
		result.AgentsCopied = n
		result.Errors = append(result.Errors, errs...)
	}

	if opts.CopySkills {
		src := filepath.Join(globalDir, "skills")
		n, errs := copySkillsDir(src, localSkillsDir)
		result.SkillsCopied = n
		result.Errors = append(result.Errors, errs...)
	}

	if opts.SyncRoles {
		agents := DiscoverAgents(workDir)
		for _, ag := range agents {
			if ag.Name == "" {
				continue
			}
			// Try to create; if already exists, update
			_, err := a.client.CreateProjectAgent(projectID, pkgserver.CreateRoleRequest{
				Slug:        ag.Slug,
				Name:        ag.Name,
				Description: ag.Description,
			})
			if err != nil {
				// May already exist — try update
				_ = a.client.UpdateProjectAgent(projectID, ag.Slug, pkgserver.UpdateRoleRequest{
					Name:        &ag.Name,
					Description: &ag.Description,
				})
			}
			result.RolesSynced++
		}
	}

	return result
}

// userHomeClaudeDir returns ~/.claude
func userHomeClaudeDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude")
}

// readAgentsDir parses all .md files in dir and returns their AgentDef.
// Filenames with invalid slug characters are sanitised: each run of
// non-alphanumeric/underscore/hyphen characters is replaced with a hyphen.
// If sanitisation produces an empty or invalid slug the file is skipped.
func readAgentsDir(dir string, isLocal bool) []AgentDef {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var defs []AgentDef
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		raw := strings.TrimSuffix(e.Name(), ".md")
		slug := invalidSlugCharRe.ReplaceAllString(raw, "-")
		slug = strings.Trim(slug, "-")
		if !isValidSlug(slug) {
			continue
		}
		path := filepath.Join(dir, e.Name())
		name, desc := parseAgentFrontmatter(path)
		defs = append(defs, AgentDef{
			Slug:        slug,
			Name:        name,
			Description: desc,
			SourcePath:  path,
			IsLocal:     isLocal,
		})
	}
	return defs
}

// parseAgentFrontmatter reads the YAML frontmatter block (between ---) and extracts name/description.
func parseAgentFrontmatter(path string) (name, description string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	inFrontmatter := false
	lineNum := 0
	for scanner.Scan() {
		line := scanner.Text()
		lineNum++
		if lineNum == 1 {
			if strings.TrimSpace(line) == "---" {
				inFrontmatter = true
			}
			continue
		}
		if inFrontmatter {
			if strings.TrimSpace(line) == "---" {
				break
			}
			if after, ok := strings.CutPrefix(line, "name:"); ok {
				name = strings.TrimSpace(after)
			} else if after, ok := strings.CutPrefix(line, "description:"); ok {
				description = strings.TrimSpace(after)
			}
		}
	}
	return
}

// copyDir copies files matching glob from src to dst directory.
func copyDir(src, dst, glob string) (int, []string) {
	matches, err := filepath.Glob(filepath.Join(src, glob))
	if err != nil || len(matches) == 0 {
		return 0, nil
	}
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return 0, []string{fmt.Sprintf("mkdir %s: %v", dst, err)}
	}
	var errs []string
	count := 0
	for _, src := range matches {
		dst := filepath.Join(dst, filepath.Base(src))
		if err := copyFile(src, dst); err != nil {
			errs = append(errs, err.Error())
		} else {
			count++
		}
	}
	return count, errs
}

// copySkillsDir copies each skill subdirectory (skill/SKILL.md) from src to dst.
func copySkillsDir(src, dst string) (int, []string) {
	entries, err := os.ReadDir(src)
	if err != nil {
		return 0, nil
	}
	var errs []string
	count := 0
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		skillFile := filepath.Join(src, e.Name(), "SKILL.md")
		if _, err := os.Stat(skillFile); err != nil {
			continue
		}
		dstDir := filepath.Join(dst, e.Name())
		if err := os.MkdirAll(dstDir, 0o755); err != nil {
			errs = append(errs, err.Error())
			continue
		}
		if err := copyFile(skillFile, filepath.Join(dstDir, "SKILL.md")); err != nil {
			errs = append(errs, err.Error())
		} else {
			count++
		}
	}
	return count, errs
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, io.LimitReader(in, maxCopyBytes))
	return err
}
