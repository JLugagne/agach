package tui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"

	appagach "github.com/JLugagne/agach-mcp/internal/agach/app"
	"github.com/JLugagne/agach-mcp/internal/agach/inbound/tui/tcellapp"
	pkgkanban "github.com/JLugagne/agach-mcp/pkg/kanban"
)

// setupDoneMsg signals setup is complete (skip or applied)
type setupDoneMsg struct{}

// syncRolesRequestMsg triggers the sync roles screen from config
type syncRolesRequestMsg struct {
	project pkgkanban.ProjectResponse
}

// syncPreviewMsg carries the diff before applying
type syncPreviewMsg struct {
	toAdd    []appagach.AgentDef
	toRemove []pkgkanban.RoleResponse
	err      error
}

// syncAppliedMsg signals the sync has been applied
type syncAppliedMsg struct {
	added   int
	removed int
	err     error
}

// backToConfigMsg signals the sync screen should go back to config
type backToConfigMsg struct{}

// SyncRolesModel shows a diff of roles to add/remove and asks for confirmation
type SyncRolesModel struct {
	app     *tuiApp
	project pkgkanban.ProjectResponse
	workDir string

	loading  bool
	toAdd    []appagach.AgentDef
	toRemove []pkgkanban.RoleResponse
	err      string

	// confirmation state
	confirmed bool
	applying  bool
	result    *syncAppliedMsg
}

func newSyncRolesModel(app *tuiApp, project pkgkanban.ProjectResponse, workDir string) SyncRolesModel {
	return SyncRolesModel{
		app:     app,
		project: project,
		workDir: workDir,
		loading: true,
	}
}

func (m SyncRolesModel) Init() tcellapp.Cmd {
	return m.loadPreview()
}

func (m SyncRolesModel) loadPreview() tcellapp.Cmd {
	return func() tcellapp.Msg {
		// Get current project roles
		current, err := m.app.kanban.ListProjectRoles(m.project.ID)
		if err != nil {
			return syncPreviewMsg{err: err}
		}

		// Discover all available agents (local + global)
		available := appagach.DiscoverAgents(m.workDir)

		// Build sets
		currentSlugs := map[string]bool{}
		for _, r := range current {
			currentSlugs[r.Slug] = true
		}
		availableSlugs := map[string]bool{}
		for _, a := range available {
			if a.Name != "" {
				availableSlugs[a.Slug] = true
			}
		}

		// toAdd: agents not yet in project roles
		var toAdd []appagach.AgentDef
		for _, a := range available {
			if a.Name != "" && !currentSlugs[a.Slug] {
				toAdd = append(toAdd, a)
			}
		}

		// toRemove: project roles with no matching agent
		var toRemove []pkgkanban.RoleResponse
		for _, r := range current {
			if !availableSlugs[r.Slug] {
				toRemove = append(toRemove, r)
			}
		}

		return syncPreviewMsg{toAdd: toAdd, toRemove: toRemove}
	}
}

func (m SyncRolesModel) applySync() tcellapp.Cmd {
	return func() tcellapp.Msg {
		added := 0
		removed := 0

		for _, ag := range m.toAdd {
			name := ag.Name
			desc := ag.Description
			_, err := m.app.kanban.CreateProjectAgent(m.project.ID, pkgkanban.CreateRoleRequest{
				Slug:        ag.Slug,
				Name:        name,
				Description: desc,
			})
			if err == nil {
				added++
			}
		}

		for _, r := range m.toRemove {
			err := m.app.kanban.DeleteProjectAgent(m.project.ID, r.Slug)
			if err == nil {
				removed++
			}
		}

		return syncAppliedMsg{added: added, removed: removed}
	}
}

func (m SyncRolesModel) HandleMsg(msg tcellapp.Msg) (SyncRolesModel, tcellapp.Cmd) {
	switch msg := msg.(type) {
	case syncPreviewMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err.Error()
		} else {
			m.toAdd = msg.toAdd
			m.toRemove = msg.toRemove
		}

	case syncAppliedMsg:
		m.applying = false
		m.result = &msg

	case tcellapp.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m SyncRolesModel) handleKey(msg tcellapp.KeyMsg) (SyncRolesModel, tcellapp.Cmd) {
	if m.result != nil {
		// Done — any key goes back
		return m, func() tcellapp.Msg { return backToConfigMsg{} }
	}

	ks := tcellapp.KeyString(msg)
	switch ks {
	case "esc", "q":
		return m, func() tcellapp.Msg { return backToConfigMsg{} }
	case "enter", "y":
		if !m.loading && m.err == "" {
			m.applying = true
			return m, m.applySync()
		}
	case "n":
		return m, func() tcellapp.Msg { return backToConfigMsg{} }
	}
	return m, nil
}

func (m SyncRolesModel) Draw(s tcell.Screen, w, h int) {
	tcellapp.Fill(s, 0, 0, w, h, tcell.StyleDefault.Background(tcellapp.ColorSurface))

	bg := tcellapp.DrawHeaderBar(s, 0, w)
	x := tcellapp.DrawText(s, 2, 0, bg.Bold(true).Foreground(tcellapp.ColorPrimary), "  Sync Roles")
	tcellapp.DrawText(s, x+2, 0, bg.Foreground(tcellapp.ColorDimmer), m.project.Name)

	boxW := min(60, w-4)
	boxX := max(1, (w-boxW)/2)
	innerX := boxX + 2

	row := 2

	if m.loading {
		tcellapp.DrawText(s, innerX, row, tcellapp.StyleDim(), "computing diff...")
		return
	}

	if m.err != "" {
		tcellapp.DrawText(s, innerX, row, tcellapp.StyleError(), "error: "+m.err)
		row++
		tcellapp.DrawText(s, innerX, row, tcellapp.StyleDim(), "[esc] back")
		return
	}

	if m.result != nil {
		tcellapp.DrawText(s, innerX, row, tcellapp.StyleSuccess(),
			fmt.Sprintf("✓ Done: +%d added, -%d removed", m.result.added, m.result.removed))
		row += 2
		tcellapp.DrawText(s, innerX, row, tcellapp.StyleDim(), "[any key] back")
		return
	}

	if m.applying {
		tcellapp.DrawText(s, innerX, row, tcellapp.StyleDim(), "applying...")
		return
	}

	if len(m.toAdd) == 0 && len(m.toRemove) == 0 {
		tcellapp.DrawText(s, innerX, row, tcellapp.StyleSuccess(), "✓ Roles are already in sync")
		row += 2
		tcellapp.DrawText(s, innerX, row, tcellapp.StyleDim(), "[esc] back")
		return
	}

	innerH := 0
	if len(m.toAdd) > 0 {
		innerH += 1 + len(m.toAdd) + 1
	}
	if len(m.toRemove) > 0 {
		innerH += 1 + len(m.toRemove) + 1
	}
	boxH := innerH + 2

	cardBg := tcell.StyleDefault.Background(tcellapp.ColorCardBg)

	tcellapp.DrawBoxWithTitle(s, boxX, row, boxW, boxH,
		tcell.StyleDefault.Background(tcellapp.ColorSurface).Foreground(tcellapp.ColorCardBorder),
		"Changes", tcellapp.StyleSubtitle().Background(tcellapp.ColorSurface))
	tcellapp.FillInner(s, boxX, row, boxW, boxH, cardBg)

	iy := row + 1

	if len(m.toAdd) > 0 {
		tcellapp.DrawText(s, innerX, iy, cardBg.Foreground(tcellapp.ColorSuccess),
			fmt.Sprintf("+ %d role(s) to add:", len(m.toAdd)))
		iy++
		for _, a := range m.toAdd {
			src := "global"
			if a.IsLocal {
				src = "local"
			}
			tcellapp.DrawText(s, innerX, iy, cardBg.Foreground(tcellapp.ColorSuccess), "+ ")
			tcellapp.DrawText(s, innerX+2, iy, cardBg.Foreground(tcellapp.ColorNormal),
				fmt.Sprintf("%-20s %s (%s)", a.Slug, a.Name, src))
			iy++
		}
		iy++
	}

	if len(m.toRemove) > 0 {
		tcellapp.DrawText(s, innerX, iy, cardBg.Foreground(tcellapp.ColorError),
			fmt.Sprintf("- %d role(s) to remove:", len(m.toRemove)))
		iy++
		for _, r := range m.toRemove {
			tcellapp.DrawText(s, innerX, iy, cardBg.Foreground(tcellapp.ColorError), "- ")
			tcellapp.DrawText(s, innerX+2, iy, cardBg.Foreground(tcellapp.ColorNormal),
				fmt.Sprintf("%-20s %s", r.Slug, r.Name))
			iy++
		}
	}

	tcellapp.DrawFooterBar(s, h-1, w, "[enter/y] apply  [n/esc] cancel")
}

// appSetupOptions converts booleans to app.SetupOptions (used from welcome.go)
func appSetupOptions(copyAgents, copySkills, syncRoles bool) appagach.SetupOptions {
	return appagach.SetupOptions{
		CopyAgents: copyAgents,
		CopySkills: copySkills,
		SyncRoles:  syncRoles,
	}
}
