package queries

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"strings"

	"github.com/JLugagne/agach-mcp/internal/pkg/controller"
	"github.com/JLugagne/agach-mcp/internal/server/domain"
	"github.com/JLugagne/agach-mcp/internal/server/domain/service"
	"github.com/gorilla/mux"
)

// AgentDownloadHandler serves a multipart bundle of project agents and skills.
type AgentDownloadHandler struct {
	queries    service.Queries
	controller *controller.Controller
}

func NewAgentDownloadHandler(queries service.Queries, ctrl *controller.Controller) *AgentDownloadHandler {
	return &AgentDownloadHandler{queries: queries, controller: ctrl}
}

func (h *AgentDownloadHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/api/projects/{projectId}/agents/download", h.Download).Methods("GET")
}

// manifestEntry is one row in the final manifest part.
type manifestEntry struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

// Download streams a multipart response containing every agent and skill
// assigned to the project.  Each MIME part carries:
//
//	Content-Disposition: attachment; filename="agents/<slug>.md"
//	Content-Type: text/markdown
//
// Agent files include YAML frontmatter (name, description, model, thinking,
// skills list).  Skill files include frontmatter (name, description).
// The last part is the manifest (application/json) listing every file with
// its SHA-256 checksum.  Skills shared by multiple agents are sent only once.
func (h *AgentDownloadHandler) Download(w http.ResponseWriter, r *http.Request) {
	rawID := mux.Vars(r)["projectId"]
	projectID, err := domain.ParseProjectID(rawID)
	if err != nil {
		h.controller.SendFail(w, r, nil, domain.ErrProjectNotFound)
		return
	}

	// Fetch project agents
	agents, err := h.queries.ListProjectAgents(r.Context(), projectID)
	if err != nil {
		if domain.IsDomainError(err) {
			h.controller.SendFail(w, r, nil, err)
		} else {
			h.controller.SendError(w, r, err)
		}
		return
	}

	// Collect all unique skills across agents
	type fileEntry struct {
		path    string
		content []byte
	}
	var files []fileEntry
	seenSkills := make(map[string]bool)

	for _, agent := range agents {
		skills, err := h.queries.ListAgentSkills(r.Context(), agent.Slug)
		if err != nil {
			skills = nil
		}

		// Build agent .md with frontmatter
		agentMD := buildAgentMarkdown(agent, skills)
		path := fmt.Sprintf("agents/%s.md", agent.Slug)
		files = append(files, fileEntry{path: path, content: []byte(agentMD)})

		for _, skill := range skills {
			if seenSkills[skill.Slug] {
				continue
			}
			seenSkills[skill.Slug] = true
			skillMD := buildSkillMarkdown(skill)
			skillPath := fmt.Sprintf("skills/%s.md", skill.Slug)
			files = append(files, fileEntry{path: skillPath, content: []byte(skillMD)})
		}
	}

	// Write multipart response
	mw := multipart.NewWriter(w)
	w.Header().Set("Content-Type", "multipart/mixed; boundary="+mw.Boundary())

	var manifest []manifestEntry

	for _, f := range files {
		checksum := fmt.Sprintf("%x", sha256.Sum256(f.content))
		manifest = append(manifest, manifestEntry{Path: f.path, SHA256: checksum})

		hdr := make(textproto.MIMEHeader)
		hdr.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, f.path))
		hdr.Set("Content-Type", "text/markdown")

		part, err := mw.CreatePart(hdr)
		if err != nil {
			return
		}
		if _, err := part.Write(f.content); err != nil {
			return
		}
	}

	// Write manifest as last part
	manifestJSON, _ := json.Marshal(manifest)

	hdr := make(textproto.MIMEHeader)
	hdr.Set("Content-Disposition", `attachment; filename="manifest.json"`)
	hdr.Set("Content-Type", "application/json")
	part, err := mw.CreatePart(hdr)
	if err != nil {
		return
	}
	part.Write(manifestJSON)

	mw.Close()
}

// buildAgentMarkdown produces the full .md content for an agent, with YAML
// frontmatter followed by the agent's Content body.
func buildAgentMarkdown(agent domain.Agent, skills []domain.Skill) string {
	var b strings.Builder
	b.WriteString("---\n")
	b.WriteString(fmt.Sprintf("name: %s\n", agent.Name))
	b.WriteString(fmt.Sprintf("description: %s\n", agent.Description))
	if agent.Model != "" {
		b.WriteString(fmt.Sprintf("model: %s\n", agent.Model))
	}
	if agent.Thinking != "" {
		b.WriteString(fmt.Sprintf("thinking: %s\n", agent.Thinking))
	}
	if len(skills) > 0 {
		b.WriteString("skills:\n")
		for _, s := range skills {
			b.WriteString(fmt.Sprintf("  - %s\n", s.Slug))
		}
	}
	b.WriteString("---\n\n")
	b.WriteString(agent.Content)
	return b.String()
}

// buildSkillMarkdown produces the full .md content for a skill, with YAML
// frontmatter followed by the skill's Content body.
func buildSkillMarkdown(skill domain.Skill) string {
	var b strings.Builder
	b.WriteString("---\n")
	b.WriteString(fmt.Sprintf("name: %s\n", skill.Name))
	b.WriteString(fmt.Sprintf("description: %s\n", skill.Description))
	b.WriteString("---\n\n")
	b.WriteString(skill.Content)
	return b.String()
}
