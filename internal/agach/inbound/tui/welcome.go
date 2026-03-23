package tui

import (
	"fmt"
	"sort"

	"github.com/gdamore/tcell/v2"

	"github.com/JLugagne/agach-mcp/internal/agach/inbound/tui/tcellapp"
	pkgkanban "github.com/JLugagne/agach-mcp/pkg/kanban"
)

// projectsLoadedMsg carries the loaded projects list
type projectsLoadedMsg struct {
	projects []pkgkanban.ProjectResponse
	err      error
}

// projectCreatedMsg carries the created project
type projectCreatedMsg struct {
	project *pkgkanban.ProjectResponse
	err     error
}

type welcomeState int

const (
	welcomeStateList    welcomeState = iota
	welcomeStateCreate               // step 1: name/folder/desc
	welcomeStateSetup                // step 2: copy agents/skills options
)

// projectItem is a flat list entry for navigation
type projectItem struct {
	project pkgkanban.ProjectResponse
}

// WelcomeModel is the welcome screen showing the project list
type WelcomeModel struct {
	app     *tuiApp
	items   []projectItem
	cursor  int
	loading bool
	err     string
	state   welcomeState

	// create form step 1: name, desc
	newProjectName string
	newProjectDesc string
	formField      int // 0=name, 1=desc

	// setup step 2: checkboxes
	setupCopyAgents bool
	setupCopySkills bool
	setupSyncRoles  bool
	setupField      int // 0=copyAgents, 1=copySkills, 2=syncRoles, 3=confirm

	// pending project after creation (waiting for setup)
	pendingProject *pkgkanban.ProjectResponse

	// scroll offset for long lists
	scrollOffset int
}

func newWelcomeModel(app *tuiApp) *WelcomeModel {
	return &WelcomeModel{
		app:             app,
		loading:         true,
		setupCopyAgents: true,
		setupCopySkills: true,
		setupSyncRoles:  true,
	}
}

func (m *WelcomeModel) Init() tcellapp.Cmd {
	return m.loadProjects()
}

func (m *WelcomeModel) loadProjects() tcellapp.Cmd {
	return func() tcellapp.Msg {
		projects, err := m.app.kanban.ListProjects()
		return projectsLoadedMsg{projects: projects, err: err}
	}
}

func buildItems(projects []pkgkanban.ProjectResponse) []projectItem {
	// Sort projects by name
	sorted := make([]pkgkanban.ProjectResponse, len(projects))
	copy(sorted, projects)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Name < sorted[j].Name
	})

	items := make([]projectItem, len(sorted))
	for i, p := range sorted {
		items[i] = projectItem{project: p}
	}
	return items
}

func (m *WelcomeModel) HandleMsg(msg tcellapp.Msg) (tcellapp.Screen, tcellapp.Cmd) {
	switch msg := msg.(type) {
	case projectsLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err.Error()
		} else {
			m.items = buildItems(msg.projects)
			m.cursor = 0
		}
		return m, nil

	case projectCreatedMsg:
		if msg.err != nil {
			m.err = msg.err.Error()
			m.state = welcomeStateList
			return m, nil
		}
		m.pendingProject = msg.project
		m.state = welcomeStateSetup
		m.setupField = 0
		return m, nil

	case setupDoneMsg:
		m.pendingProject = nil
		m.state = welcomeStateList
		m.newProjectName = ""
		m.newProjectDesc = ""
		m.formField = 0
		return m, m.loadProjects()

	case tcellapp.KeyMsg:
		switch m.state {
		case welcomeStateCreate:
			return m.updateCreateForm(msg)
		case welcomeStateSetup:
			return m.updateSetup(msg)
		default:
			return m.updateList(msg)
		}
	}
	return m, nil
}

func (m *WelcomeModel) updateList(msg tcellapp.KeyMsg) (tcellapp.Screen, tcellapp.Cmd) {
	switch tcellapp.KeyString(msg) {
	case "up", "k":
		m.moveCursor(-1)
	case "down", "j":
		m.moveCursor(1)
	case "n":
		m.state = welcomeStateCreate
		m.formField = 0
	case "t":
		return m, func() tcellapp.Msg { return launchDiagnosticMsg{} }
	case "r":
		m.loading = true
		return m, m.loadProjects()
	case "enter":
		if len(m.items) > 0 && m.cursor < len(m.items) {
			selected := m.items[m.cursor].project
			return m, func() tcellapp.Msg {
				return projectSelectedMsg{project: selected}
			}
		}
	}
	return m, nil
}

func (m *WelcomeModel) moveCursor(dir int) {
	next := m.cursor + dir
	if next >= 0 && next < len(m.items) {
		m.cursor = next
	}
}

func (m *WelcomeModel) updateCreateForm(msg tcellapp.KeyMsg) (tcellapp.Screen, tcellapp.Cmd) {
	const numFields = 2
	switch tcellapp.KeyString(msg) {
	case "esc":
		m.state = welcomeStateList
		m.newProjectName = ""
		m.newProjectDesc = ""
		m.formField = 0
	case "tab", "down":
		m.formField = (m.formField + 1) % numFields
	case "shift+tab", "up":
		m.formField = (m.formField - 1 + numFields) % numFields
	case "enter":
		if m.formField < numFields-1 {
			m.formField++
			return m, nil
		}
		if m.newProjectName == "" {
			return m, nil
		}
		name := m.newProjectName
		desc := m.newProjectDesc
		return m, func() tcellapp.Msg {
			project, err := m.app.kanban.CreateProject(pkgkanban.CreateProjectRequest{
				Name:        name,
				Description: desc,
			})
			return projectCreatedMsg{project: project, err: err}
		}
	case "backspace":
		switch m.formField {
		case 0:
			if len(m.newProjectName) > 0 {
				m.newProjectName = m.newProjectName[:len(m.newProjectName)-1]
			}
		case 1:
			if len(m.newProjectDesc) > 0 {
				m.newProjectDesc = m.newProjectDesc[:len(m.newProjectDesc)-1]
			}
		}
	default:
		if ch := tcellapp.KeyString(msg); len(ch) == 1 {
			switch m.formField {
			case 0:
				m.newProjectName += ch
			case 1:
				m.newProjectDesc += ch
			}
		}
	}
	return m, nil
}

func (m *WelcomeModel) updateSetup(msg tcellapp.KeyMsg) (tcellapp.Screen, tcellapp.Cmd) {
	const numSetupFields = 4 // copyAgents, copySkills, syncRoles, confirm
	switch tcellapp.KeyString(msg) {
	case "esc":
		return m, func() tcellapp.Msg { return setupDoneMsg{} }
	case "tab", "down", "j":
		m.setupField = (m.setupField + 1) % numSetupFields
	case "shift+tab", "up", "k":
		m.setupField = (m.setupField - 1 + numSetupFields) % numSetupFields
	case " ":
		switch m.setupField {
		case 0:
			m.setupCopyAgents = !m.setupCopyAgents
		case 1:
			m.setupCopySkills = !m.setupCopySkills
		case 2:
			m.setupSyncRoles = !m.setupSyncRoles
		}
	case "enter":
		if m.setupField < 3 {
			switch m.setupField {
			case 0:
				m.setupCopyAgents = !m.setupCopyAgents
			case 1:
				m.setupCopySkills = !m.setupCopySkills
			case 2:
				m.setupSyncRoles = !m.setupSyncRoles
			}
			m.setupField++
			return m, nil
		}
		project := m.pendingProject
		copyAgents := m.setupCopyAgents
		copySkills := m.setupCopySkills
		syncRoles := m.setupSyncRoles
		workDir := m.app.workDir
		agachApp := m.app.agach
		return m, func() tcellapp.Msg {
			agachApp.SetupProject(project.ID, workDir, appSetupOptions(copyAgents, copySkills, syncRoles))
			return setupDoneMsg{}
		}
	}
	return m, nil
}

func (m *WelcomeModel) Draw(s tcell.Screen, w, h int) {
	tcellapp.Fill(s, 0, 0, w, h, tcell.StyleDefault.Background(tcellapp.ColorSurface))

	switch m.state {
	case welcomeStateCreate:
		m.drawCreateForm(s, 2, w, h)
		return
	case welcomeStateSetup:
		m.drawSetupForm(s, 2, w, h)
		return
	}

	if m.loading {
		tcellapp.DrawCenteredText(s, 0, h/2, w, tcellapp.StyleDim(), "loading projects...")
		return
	}
	if m.err != "" {
		tcellapp.DrawCenteredText(s, 0, h/2, w, tcellapp.StyleError(), "error: "+m.err)
		tcellapp.DrawCenteredText(s, 0, h/2+1, w, tcellapp.StyleDim(), "[r] retry")
		return
	}

	surfBg := tcell.StyleDefault.Background(tcellapp.ColorSurface)

	projectCount := len(m.items)

	// Layout constants — use a wider panel
	panelW := min(80, w-4)
	panelX := (w - panelW) / 2
	if panelX < 2 {
		panelX = 2
	}

	maxVisible := 10
	displayItems := min(projectCount, maxVisible)

	// Calculate block height:
	// title(1) + subtitle(1) + blank(2) + shortcuts(5) + blank(1) + separator(1) + blank(1) + projects
	shortcutLines := 5
	projectLines := displayItems
	if projectCount == 0 {
		projectLines = 1
	}
	totalBlock := 1 + 1 + 2 + shortcutLines + 1 + 1 + 1 + projectLines
	startY := max(2, (h-totalBlock)/2)
	cy := startY

	// ── Title ──────────────────────────────────────────────────
	titleStyle := surfBg.Bold(true).Foreground(tcellapp.ColorPrimary)
	tcellapp.DrawCenteredText(s, 0, cy, w, titleStyle, "╺  agach  ╸")
	cy++

	// Subtitle
	tcellapp.DrawCenteredText(s, 0, cy, w, surfBg.Foreground(tcellapp.ColorMuted), "multi-agent orchestrator")
	cy += 3

	// ── Shortcuts panel ────────────────────────────────────────
	type shortcut struct {
		key  string
		desc string
	}
	shortcuts := []shortcut{
		{"n", "New project"},
		{"↵", "Open project"},
		{"t", "Token diagnostic"},
		{"r", "Refresh"},
		{"q", "Quit"},
	}

	// Draw shortcuts left-aligned within panel
	shortcutX := panelX + 2
	for _, sc := range shortcuts {
		// Key badge with subtle bg
		keyBg := tcell.StyleDefault.Background(tcellapp.ColorSurfaceHL).Bold(true).Foreground(tcellapp.ColorPrimary)
		tcellapp.DrawText(s, shortcutX, cy, keyBg, " "+sc.key+" ")
		// Description
		tcellapp.DrawText(s, shortcutX+3+len([]rune(sc.key)), cy, surfBg.Foreground(tcellapp.ColorNormal), "  "+sc.desc)
		cy++
	}
	cy++

	// ── Separator ──────────────────────────────────────────────
	for col := panelX; col < panelX+panelW; col++ {
		s.SetContent(col, cy, '─', nil, surfBg.Foreground(tcellapp.ColorDimmer))
	}
	cy++

	// ── Project list ───────────────────────────────────────────
	if projectCount == 0 {
		cy++
		tcellapp.DrawCenteredText(s, 0, cy, w, surfBg.Foreground(tcellapp.ColorMuted), "no projects yet — press n to create one")
		tcellapp.DrawFooterBar(s, h-1, w, "[n] new project  [r] refresh  [q] quit")
		return
	}

	shown := 0
	for i, item := range m.items {
		if cy >= h-2 {
			break
		}
		if shown >= maxVisible {
			break
		}

		isFocused := i == m.cursor

		p := item.project
		name := tcellapp.Truncate(p.Name, 36)

		// Build description snippet
		desc := ""
		if p.Description != "" {
			desc = tcellapp.Truncate(p.Description, panelW-len([]rune(name))-10)
		}

		if isFocused {
			// Highlighted row — use ColorCardFocused for a more visible selection
			hlBg := tcell.StyleDefault.Background(tcellapp.ColorCardFocused)
			for col := panelX; col < panelX+panelW; col++ {
				s.SetContent(col, cy, ' ', nil, hlBg)
			}
			tcellapp.DrawText(s, panelX+1, cy, hlBg.Bold(true).Foreground(tcellapp.ColorPrimary), " ▶ ")
			x := tcellapp.DrawText(s, panelX+4, cy, hlBg.Bold(true).Foreground(tcellapp.ColorNormal), name)
			if desc != "" {
				tcellapp.DrawText(s, x+2, cy, hlBg.Foreground(tcellapp.ColorMuted), desc)
			}
		} else {
			tcellapp.DrawText(s, panelX+1, cy, surfBg.Foreground(tcellapp.ColorDimmer), "   ")
			x := tcellapp.DrawText(s, panelX+4, cy, surfBg.Foreground(tcellapp.ColorNormal), name)
			if desc != "" {
				tcellapp.DrawText(s, x+2, cy, surfBg.Foreground(tcellapp.ColorDimmer), desc)
			}
		}
		cy++
		shown++
	}

	// Footer
	tcellapp.DrawFooterBar(s, h-1, w, "[j/k] navigate  [enter] open  [n] new  [t] diagnostic  [r] refresh  [q] quit")
}

func (m *WelcomeModel) drawCreateForm(s tcell.Screen, _, w, h int) {
	boxW := min(60, w-4)
	boxH := 9
	boxX := max(1, (w-boxW)/2)
	boxY := max(2, (h-boxH)/2)

	tcellapp.DrawBoxWithTitle(s, boxX, boxY, boxW, boxH,
		tcell.StyleDefault.Foreground(tcellapp.ColorCardBorder).Background(tcellapp.ColorSurface),
		"New Project", tcellapp.StyleTitle())
	tcellapp.FillInner(s, boxX, boxY, boxW, boxH, tcell.StyleDefault.Background(tcellapp.ColorCardBg))

	innerX := boxX + 2
	innerW := boxW - 4
	iy := boxY + 2

	fields := []struct{ label, value string }{
		{"Name", m.newProjectName},
		{"Description", m.newProjectDesc},
	}
	for i, f := range fields {
		active := m.formField == i
		labelStyle := tcellapp.StyleDim()
		if active {
			labelStyle = tcellapp.StyleSelected()
		}
		tcellapp.DrawText(s, innerX, iy, labelStyle, f.label)
		iy++
		fieldStyle := tcellapp.StyleInputFieldInactive()
		if active {
			fieldStyle = tcellapp.StyleInputField()
		}
		tcellapp.DrawInputField(s, innerX, iy, innerW, fieldStyle, f.value, active)
		iy++
	}

	tcellapp.DrawFooterBar(s, h-1, w, "[tab/dn] next field  [enter] next/submit  [esc] cancel")
}

func (m *WelcomeModel) drawSetupForm(s tcell.Screen, _, w, h int) {
	name := ""
	if m.pendingProject != nil {
		name = m.pendingProject.Name
	}

	boxW := min(58, w-4)
	boxH := 10
	boxX := max(1, (w-boxW)/2)
	boxY := max(2, (h-boxH)/2)

	tcellapp.DrawBoxWithTitle(s, boxX, boxY, boxW, boxH,
		tcell.StyleDefault.Foreground(tcellapp.ColorCardBorder).Background(tcellapp.ColorSurface),
		"Project Setup", tcellapp.StyleTitle())
	tcellapp.FillInner(s, boxX, boxY, boxW, boxH, tcell.StyleDefault.Background(tcellapp.ColorCardBg))

	innerX := boxX + 2
	cardBg := tcell.StyleDefault.Background(tcellapp.ColorCardBg)
	iy := boxY + 1

	tcellapp.DrawText(s, innerX, iy, cardBg.Foreground(tcellapp.ColorSuccess), fmt.Sprintf("Project \"%s\" created", name))
	iy += 2

	checkboxes := []struct {
		label   string
		checked bool
		field   int
	}{
		{"Copy global agents to .claude/agents/", m.setupCopyAgents, 0},
		{"Copy global skills to .claude/skills/", m.setupCopySkills, 1},
		{"Sync project roles from agents", m.setupSyncRoles, 2},
	}

	for _, cb := range checkboxes {
		check := "[ ]"
		checkStyle := tcellapp.StyleDim()
		if cb.checked {
			check = "[+]"
			checkStyle = tcellapp.StyleSuccess()
		}
		focused := m.setupField == cb.field
		if focused {
			selBg := tcell.StyleDefault.Background(tcellapp.ColorSurfaceHL)
			for col := innerX - 1; col < boxX+boxW-1; col++ {
				s.SetContent(col, iy, ' ', nil, selBg)
			}
			tcellapp.DrawText(s, innerX, iy, selBg.Foreground(tcellapp.ColorPrimary), "> ")
			x := tcellapp.DrawText(s, innerX+2, iy, checkStyle, check)
			tcellapp.DrawText(s, x+1, iy, selBg.Foreground(tcellapp.ColorNormal), cb.label)
		} else {
			x := tcellapp.DrawText(s, innerX, iy, cardBg.Foreground(tcellapp.ColorNormal), "  ")
			x = tcellapp.DrawText(s, x, iy, checkStyle, check)
			tcellapp.DrawText(s, x+1, iy, cardBg.Foreground(tcellapp.ColorNormal), cb.label)
		}
		iy++
	}

	iy++
	if m.setupField == 3 {
		selBg := tcell.StyleDefault.Background(tcellapp.ColorSurfaceHL)
		for col := innerX - 1; col < boxX+boxW-1; col++ {
			s.SetContent(col, iy, ' ', nil, selBg)
		}
		tcellapp.DrawText(s, innerX, iy, selBg.Bold(true).Foreground(tcellapp.ColorSuccess), "> Apply & continue")
	} else {
		tcellapp.DrawText(s, innerX, iy, cardBg.Foreground(tcellapp.ColorDimmer), "  Apply & continue")
	}

	tcellapp.DrawFooterBar(s, h-1, w, "[up/dn/tab] navigate  [space/enter] toggle  [esc] skip")
}
