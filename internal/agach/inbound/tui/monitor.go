package tui

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"

	appagach "github.com/JLugagne/agach-mcp/internal/agach/app"
	"github.com/JLugagne/agach-mcp/internal/agach/domain"
	"github.com/JLugagne/agach-mcp/internal/agach/inbound/tui/tcellapp"
	pkgkanban "github.com/JLugagne/agach-mcp/pkg/kanban"
	"github.com/JLugagne/agach-mcp/pkg/kanban/client"
)

// workerUpdateMsg wraps a WorkerUpdate for the TUI
type workerUpdateMsg appagach.WorkerUpdate

// stopRunMsg signals the run has ended
type stopRunMsg struct{}

// columnCountsMsg delivers refreshed column counts
type columnCountsMsg struct {
	counts client.ColumnCounts
}

// refreshColumnCountsMsg triggers a column count refresh
type refreshColumnCountsMsg struct{}


// terminalResultMsg carries the result of opening a terminal
type terminalResultMsg struct {
	err string
}

// MonitorModel shows real-time worker status during a run.
// It starts idle until the user presses [r] to start.
type MonitorModel struct {
	app     *tuiApp
	project pkgkanban.ProjectResponse
	config  domain.RunConfig
	workers []domain.WorkerState
	started time.Time
	running bool // true after user starts the run

	// which worker is "focused" for details/console
	focusedWorker int
	showingPast   bool
	pastCursor    int

	// right panel scroll for completed tasks list
	completedScroll int

	paused       bool
	columnCounts client.ColumnCounts
	stopped      bool

	// terminal dimensions
	width  int
	height int

	// messages panel (shown below workers)
	msgBuffers      map[int][]domain.LiveMessage
	msgPanel        *messagesPanel
	viewingMessages bool

	// console overlay
	showConsole bool

	// done tasks panel (full screen with resume support)
	showDone   bool
	doneCursor int
	doneErr    string // last error from opening terminal

	// settings popup
	showSettings  bool
	roles         []pkgkanban.RoleResponse
	subProjects   []pkgkanban.ProjectResponse
	maxWorkers    int
	roleCursor    int // -1 = all roles
	scopeChoice   int // 0=main, 1=all, 2=specific
	subProjCursor int
	autoStart     bool
	settingsField int // which field is focused in the popup

	// sync roles sub-screen
	showSync bool
	sync     SyncRolesModel
}

func newMonitorModel(app *tuiApp, project pkgkanban.ProjectResponse) *MonitorModel {
	return &MonitorModel{
		app:        app,
		project:    project,
		maxWorkers: 3,
		roleCursor: -1,
		width:      80,
		height:     24,
		msgBuffers: make(map[int][]domain.LiveMessage),
	}
}

// configLoadedMsg signals that config data was loaded
type configLoadedMsg struct {
	configInitDoneMsg
}

func (m *MonitorModel) Init() tcellapp.Cmd {
	// Load roles, sub-projects, columns, then auto-start the run
	return func() tcellapp.Msg {
		roles, rolesErr := m.app.kanban.ListProjectRoles(m.project.ID)
		all, subsErr := m.app.kanban.ListProjects()
		var subs []pkgkanban.ProjectResponse
		if subsErr == nil {
			for _, p := range all {
				if p.ParentID != nil && *p.ParentID == m.project.ID {
					subs = append(subs, p)
				}
			}
		}
		return configLoadedMsg{configInitDoneMsg{
			roles:    roles,
			rolesErr: rolesErr,
			subs:     subs,
			subsErr:  subsErr,
		}}
	}
}

func (m *MonitorModel) startRun() tcellapp.Cmd {
	cfg := m.buildRunConfig()
	m.config = cfg
	workers := make([]domain.WorkerState, cfg.MaxWorkers)
	for i := range workers {
		workers[i] = domain.WorkerState{ID: i, Status: domain.WorkerIdle}
	}
	m.workers = workers
	m.started = time.Now()
	m.running = true
	m.stopped = false
	m.msgBuffers = make(map[int][]domain.LiveMessage)

	ctx, cancel := context.WithCancel(context.Background())
	m.app.runCtx = ctx
	m.app.cancelRun = cancel

	return func() tcellapp.Msg {
		updates := make(chan appagach.WorkerUpdate, 64)
		m.app.runUpdates = updates
		m.app.agach.Run(m.app.runCtx, cfg, updates)

		go func() {
			for {
				update, ok := <-m.app.runUpdates
				if !ok {
					m.app.tcellApp.Dispatch(stopRunMsg{})
					return
				}
				m.app.tcellApp.Dispatch(workerUpdateMsg(update))
			}
		}()

		go func() {
			m.app.tcellApp.Dispatch(refreshColumnCountsMsg{})
			for {
				select {
				case <-m.app.runCtx.Done():
					return
				case <-time.After(15 * time.Second):
					m.app.tcellApp.Dispatch(refreshColumnCountsMsg{})
						}
			}
		}()

		return nil
	}
}

func (m *MonitorModel) buildRunConfig() domain.RunConfig {
	cfg := domain.RunConfig{
		ProjectID:   m.project.ID,
		ProjectName: m.project.Name,
		MaxWorkers:  m.maxWorkers,
		ServerURL:   m.app.serverURL,
		AutoStart:   m.autoStart,
	}
	if m.roleCursor >= 0 && m.roleCursor < len(m.roles) {
		cfg.RoleSlug = m.roles[m.roleCursor].Slug
	}
	switch m.scopeChoice {
	case 0:
		cfg.Scope = domain.RunScopeMain
	case 1:
		cfg.Scope = domain.RunScopeAll
	case 2:
		cfg.Scope = domain.RunScopeSpecific
		if m.subProjCursor < len(m.subProjects) {
			cfg.SubProjectID = m.subProjects[m.subProjCursor].ID
		}
	}
	return cfg
}

func (m *MonitorModel) refreshColumnCounts() tcellapp.Cmd {
	return func() tcellapp.Msg {
		counts, err := m.app.kanban.GetColumnCounts(m.config.ProjectID)
		if err != nil {
			return nil
		}
		return columnCountsMsg{counts: counts}
	}
}


func (m *MonitorModel) HandleMsg(msg tcellapp.Msg) (tcellapp.Screen, tcellapp.Cmd) {
	// Sync sub-screen (from settings)
	if m.showSync {
		switch msg.(type) {
		case backToConfigMsg:
			m.showSync = false
			return m, func() tcellapp.Msg {
				roles, _ := m.app.kanban.ListProjectRoles(m.project.ID)
				return rolesLoadedMsg{roles: roles}
			}
		}
		var cmd tcellapp.Cmd
		m.sync, cmd = m.sync.HandleMsg(msg)
		return m, cmd
	}

	switch msg := msg.(type) {
	case configLoadedMsg:
		if msg.rolesErr == nil {
			m.roles = msg.roles
		}
		if msg.subsErr == nil {
			m.subProjects = msg.subs
		}
		cmd := m.startRun()
		m.app.agach.Pause()
		m.paused = true
		return m, cmd

	case configInitDoneMsg:
		if msg.rolesErr == nil {
			m.roles = msg.roles
		}
		if msg.subsErr == nil {
			m.subProjects = msg.subs
		}
		return m, nil

	case rolesLoadedMsg:
		if msg.err == nil {
			m.roles = msg.roles
		}
		return m, nil

	case workerUpdateMsg:
		if msg.WorkerID < len(m.workers) {
			m.workers[msg.WorkerID] = msg.State
		}
		for _, lm := range msg.NewMessages {
			buf := m.msgBuffers[msg.WorkerID]
			buf = append(buf, lm)
			if len(buf) > 500 {
				buf = buf[len(buf)-500:]
			}
			m.msgBuffers[msg.WorkerID] = buf
			if m.viewingMessages && m.msgPanel != nil && m.msgPanel.workerID == msg.WorkerID {
				m.msgPanel.addMessage(lm)
			}
		}
		return m, nil

	case stopRunMsg:
		m.stopped = true
		return m, nil

	case columnCountsMsg:
		m.columnCounts = msg.counts
		return m, nil

	case refreshColumnCountsMsg:
		return m, m.refreshColumnCounts()

	case terminalResultMsg:
		m.doneErr = msg.err
		return m, nil

	case tcellapp.ResizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.viewingMessages && m.msgPanel != nil {
			m.msgPanel.width = m.width
			_, msgH := m.splitHeights(msg.Height)
			m.msgPanel.height = msgH
		}
		return m, nil

	case tcellapp.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m *MonitorModel) handleKey(msg tcellapp.KeyMsg) (tcellapp.Screen, tcellapp.Cmd) {
	ks := tcellapp.KeyString(msg)

	// Settings popup
	if m.showSettings {
		return m.handleSettingsKey(ks)
	}

	// When viewing messages, forward scroll keys to the panel
	if m.viewingMessages && m.msgPanel != nil {
		switch ks {
		case "esc":
			m.viewingMessages = false
			m.msgPanel = nil
			return m, nil
		case "up", "k":
			updated, cmd := m.msgPanel.Update(msg)
			m.msgPanel = &updated
			return m, cmd
		case "down", "j":
			updated, cmd := m.msgPanel.Update(msg)
			m.msgPanel = &updated
			return m, cmd
		case "s":
			updated, cmd := m.msgPanel.Update(msg)
			m.msgPanel = &updated
			return m, cmd
		case "tab":
			if m.focusedWorker < len(m.workers)-1 {
				m.focusedWorker++
			} else {
				m.focusedWorker = 0
			}
			m.openMessagesPanel()
			return m, nil
		case "shift+tab":
			if m.focusedWorker > 0 {
				m.focusedWorker--
			} else {
				m.focusedWorker = len(m.workers) - 1
			}
			m.openMessagesPanel()
			return m, nil
		}
		return m, nil
	}

	// Console overlay
	if m.showConsole {
		switch ks {
		case "esc", "c":
			m.showConsole = false
		}
		return m, nil
	}

	// Done tasks panel
	if m.showDone {
		return m.handleDoneKey(ks)
	}

	switch ks {
	case "esc":
		if m.showingPast {
			m.showingPast = false
			return m, nil
		}
		if m.running {
			m.app.cancelRun()
		}
		return m, func() tcellapp.Msg { return backToWelcomeMsg{} }
	case "up", "k":
		if !m.running {
			return m, nil
		}
		if m.showingPast {
			if m.pastCursor > 0 {
				m.pastCursor--
			}
		} else if m.focusedWorker > 0 {
			m.focusedWorker--
		}
	case "down", "j":
		if !m.running {
			return m, nil
		}
		if m.showingPast {
			past := m.workers[m.focusedWorker].Past
			if m.pastCursor < len(past)-1 {
				m.pastCursor++
			}
		} else if m.focusedWorker < len(m.workers)-1 {
			m.focusedWorker++
		}
	case " ":
		if m.running && !m.stopped {
			if m.app.agach.IsPaused() {
				m.app.agach.Resume()
			} else {
				m.app.agach.Pause()
			}
			m.paused = m.app.agach.IsPaused()
		}
	case "r", "enter":
		if !m.running {
			return m, m.startRun()
		}
	case "s":
		m.showSettings = true
		m.settingsField = 0
		return m, nil
	case "p":
		if m.running && len(m.workers[m.focusedWorker].Past) > 0 {
			m.showingPast = !m.showingPast
			m.pastCursor = 0
		}
	case "m":
		if m.running {
			m.openMessagesPanel()
			m.viewingMessages = true
		}
	case "c":
		if m.running {
			m.showConsole = !m.showConsole
		}
	case "d":
		if m.running {
			m.showDone = true
			m.doneCursor = 0
		}
	}
	return m, nil
}

// ── Settings popup key handling ─────────────────────────────────────────────

const (
	settingsWorkers    = 0
	settingsRole       = 1
	settingsScope      = 2
	settingsSubProject = 3
	settingsAutoStart  = 4
	settingsSyncRoles  = 5
	settingsStart      = 6
)

func (m *MonitorModel) settingsFieldCount() int {
	if m.scopeChoice == 2 {
		return 8 // workers, role, scope, subproject, autostart, sync, start (index 0-6, but 7 is max+1)
	}
	return 7 // workers, role, scope, autostart, sync, start (index 0-6, settingsStart=6 reachable)
}

func (m *MonitorModel) settingsNextField() {
	m.settingsField++
	if m.scopeChoice != 2 && m.settingsField == settingsSubProject {
		m.settingsField++
	}
	max := m.settingsFieldCount()
	if m.settingsField >= max {
		m.settingsField = max - 1
	}
}

func (m *MonitorModel) settingsPrevField() {
	m.settingsField--
	if m.scopeChoice != 2 && m.settingsField == settingsSubProject {
		m.settingsField--
	}
	if m.settingsField < 0 {
		m.settingsField = 0
	}
}

func (m *MonitorModel) handleSettingsKey(ks string) (tcellapp.Screen, tcellapp.Cmd) {
	switch ks {
	case "esc":
		m.showSettings = false
	case "tab", "down", "j":
		m.settingsNextField()
	case "shift+tab", "up", "k":
		m.settingsPrevField()
	case "left", "h":
		m.settingsDecrement()
	case "right", "l":
		m.settingsIncrement()
	case " ":
		if m.settingsField == settingsAutoStart {
			m.autoStart = !m.autoStart
		}
	case "enter":
		actualField := m.settingsField
		switch actualField {
		case settingsAutoStart:
			m.autoStart = !m.autoStart
			m.settingsNextField()
		case settingsSyncRoles:
			m.sync = newSyncRolesModel(m.app, m.project, m.app.workDir)
			m.showSync = true
			m.showSettings = false
			return m, m.sync.Init()
		case settingsStart:
			m.showSettings = false
			if !m.running {
				return m, m.startRun()
			}
			// Apply updated settings to running workers immediately.
			newCfg := m.buildRunConfig()
			m.config = newCfg
			m.app.agach.UpdateConfig(newCfg)
		default:
			m.settingsNextField()
		}
	}
	return m, nil
}

func (m *MonitorModel) settingsIncrement() {
	switch m.settingsField {
	case settingsWorkers:
		if m.maxWorkers < 10 {
			m.maxWorkers++
		}
	case settingsRole:
		if m.roleCursor < len(m.roles)-1 {
			m.roleCursor++
		}
	case settingsScope:
		if m.scopeChoice < 2 {
			m.scopeChoice++
		}
	case settingsSubProject:
		if m.subProjCursor < len(m.subProjects)-1 {
			m.subProjCursor++
		}
	}
}

func (m *MonitorModel) settingsDecrement() {
	switch m.settingsField {
	case settingsWorkers:
		if m.maxWorkers > 1 {
			m.maxWorkers--
		}
	case settingsRole:
		if m.roleCursor > -1 {
			m.roleCursor--
		}
	case settingsScope:
		if m.scopeChoice > 0 {
			m.scopeChoice--
		}
	case settingsSubProject:
		if m.subProjCursor > 0 {
			m.subProjCursor--
		}
	}
}

func (m *MonitorModel) settingsRoleValue() string {
	if m.roleCursor < 0 || len(m.roles) == 0 {
		return "(all roles)"
	}
	r := m.roles[m.roleCursor]
	if r.Icon != "" {
		return r.Icon + " " + r.Name
	}
	return r.Name
}

func (m *MonitorModel) settingsScopeValue() string {
	switch m.scopeChoice {
	case 0:
		return "main project only"
	case 1:
		return "all (including subprojects)"
	case 2:
		return "specific subproject"
	}
	return ""
}

func (m *MonitorModel) settingsSubProjValue() string {
	if len(m.subProjects) == 0 {
		return "(no subprojects)"
	}
	return m.subProjects[m.subProjCursor].Name
}

// openMessagesPanel creates/resets the messages panel for the focused worker
func (m *MonitorModel) openMessagesPanel() {
	_, msgH := m.splitHeights(m.height)
	panel := newMessagesPanel(m.focusedWorker, m.width, msgH)
	for _, lm := range m.msgBuffers[m.focusedWorker] {
		panel.addMessage(lm)
	}
	m.msgPanel = &panel
	m.viewingMessages = true
}

// splitHeights returns (workersH, messagesH) for the split layout.
// contentH = total rows between stats bar and footer.
// Layout: workersH rows + 1 separator row + msgH rows = contentH.
func (m *MonitorModel) splitHeights(totalH int) (int, int) {
	contentH := totalH - 5 - 1 // minus header(1) + stats(4) + footer(1)
	if contentH < 4 {
		contentH = 4
	}

	// Workers panel gets enough for all workers (1 row each in compact mode)
	workersNeeded := len(m.workers)
	if workersNeeded > contentH/2 {
		workersNeeded = contentH / 2
	}
	if workersNeeded < 1 {
		workersNeeded = 1
	}

	msgH := contentH - workersNeeded - 1 // -1 for separator
	if msgH < 3 {
		msgH = 3
		workersNeeded = contentH - msgH - 1
		if workersNeeded < 1 {
			workersNeeded = 1
		}
	}
	return workersNeeded, msgH
}

// ── Drawing ──────────────────────────────────────────────────────────────────

func (m *MonitorModel) Draw(s tcell.Screen, w, h int) {
	// Background
	tcellapp.Fill(s, 0, 0, w, h, tcell.StyleDefault.Background(tcellapp.ColorSurface))

	if m.showSync {
		m.sync.Draw(s, w, h)
		return
	}

	if !m.running {
		// auto-started in paused mode on configLoadedMsg — this state is transient
		return
	}

	// Settings popup overlay (while running)
	if m.showSettings {
		m.drawSettingsPopup(s, w, h)
		return
	}

	// Header (row 0)
	m.drawHeader(s, w)

	// Stats bar (rows 1-4)
	m.drawStatsBar(s, w)

	contentY := 5
	contentH := h - contentY - 1 // minus footer

	if m.showDone {
		m.drawDonePanel(s, contentY, w, contentH)
	} else if m.showConsole {
		m.drawConsoleOverlay(s, contentY, w, contentH)
	} else if m.showingPast {
		m.drawPastTasks(s, contentY, w, contentH)
	} else if m.viewingMessages && m.msgPanel != nil {
		// Split layout: workers on top, messages on bottom
		workersH, msgH := m.splitHeights(h)

		m.drawWorkersCompact(s, 0, contentY, w, workersH)

		// Separator
		sepY := contentY + workersH
		sepStyle := tcell.StyleDefault.Background(tcellapp.ColorSurface).Foreground(tcellapp.ColorDimmer)
		for col := 0; col < w; col++ {
			s.SetContent(col, sepY, '─', nil, sepStyle)
		}

		// Messages panel below
		m.msgPanel.width = w
		m.msgPanel.height = msgH
		m.msgPanel.Draw(s, 0, sepY+1, w, msgH)
	} else {
		// Normal layout: workers + optional completed panel
		rightPanelW := 40
		if w < 100 {
			rightPanelW = 0
		} else if w > 160 {
			rightPanelW = 50
		}

		leftW := w
		if rightPanelW > 0 {
			leftW = w - rightPanelW
		}

		m.drawWorkers(s, 0, contentY, leftW, contentH)

		if rightPanelW > 0 {
			m.drawCompletedPanel(s, leftW, contentY, rightPanelW, contentH)
		}
	}

	// Footer
	var help string
	if m.showDone {
		help = "[up/dn] navigate  [enter] open terminal  [esc] close"
	} else if m.viewingMessages {
		help = "[up/dn] scroll  [tab] switch worker  [s] auto-scroll  [esc] close"
	} else {
		help = "[up/dn] worker  [m] messages  [d] done  [c] console  [p] past  [s] settings  [space] pause  [esc] stop"
	}
	tcellapp.DrawFooterBar(s, h-1, w, help)
}

func (m *MonitorModel) drawIdleScreen(s tcell.Screen, w, h int) {
	surfBg := tcell.StyleDefault.Background(tcellapp.ColorSurface)

	if m.showSettings {
		m.drawSettingsPopup(s, w, h)
		return
	}

	// NvChad-style centered idle screen
	totalBlock := 8 // title + blank + 3 shortcuts + blank + project info + status
	startY := max(1, (h-totalBlock)/2)
	cy := startY

	// Title
	tcellapp.DrawCenteredText(s, 0, cy, w, surfBg.Bold(true).Foreground(tcellapp.ColorPrimary), "agach")
	cy += 2

	// Project name
	tcellapp.DrawCenteredText(s, 0, cy, w, surfBg.Foreground(tcellapp.ColorNormal), m.project.Name)
	if m.project.Description != "" {
		cy++
		tcellapp.DrawCenteredText(s, 0, cy, w, surfBg.Foreground(tcellapp.ColorDimmer), m.project.Description)
	}
	cy += 2

	// Shortcuts
	type shortcut struct {
		key  string
		desc string
	}
	shortcuts := []shortcut{
		{"r", "Start run"},
		{"s", "Settings"},
		{"esc", "Back"},
	}

	for _, sc := range shortcuts {
		left := w/2 - 10
		if left < 2 {
			left = 2
		}
		tcellapp.DrawText(s, left, cy, surfBg.Bold(true).Foreground(tcellapp.ColorAccent), sc.key)
		tcellapp.DrawText(s, left+5, cy, surfBg.Foreground(tcellapp.ColorMuted), sc.desc)
		cy++
	}

	// Summary line at bottom
	cy += 2
	summary := fmt.Sprintf("%d workers · %s", m.maxWorkers, m.settingsRoleValue())
	tcellapp.DrawCenteredText(s, 0, cy, w, surfBg.Foreground(tcellapp.ColorDimmer), summary)

	tcellapp.DrawFooterBar(s, h-1, w, "[r/enter] start  [s] settings  [esc] back")
}

func (m *MonitorModel) drawSettingsPopup(s tcell.Screen, w, h int) {
	surfBg := tcell.StyleDefault.Background(tcellapp.ColorSurface)
	_ = surfBg

	boxW := min(56, w-4)
	boxH := 12
	boxX := max(1, (w-boxW)/2)
	boxY := max(1, (h-boxH)/2)

	cardBg := tcell.StyleDefault.Background(tcellapp.ColorCardBg)

	tcellapp.DrawBoxWithTitle(s, boxX, boxY, boxW, boxH,
		tcell.StyleDefault.Background(tcellapp.ColorSurface).Foreground(tcellapp.ColorCardBorder),
		"Settings", tcellapp.StyleSubtitle().Background(tcellapp.ColorSurface))
	tcellapp.FillInner(s, boxX, boxY, boxW, boxH, cardBg)

	innerX := boxX + 2

	type fieldEntry struct {
		field int
		label string
		value string
	}

	fields := []fieldEntry{
		{settingsWorkers, "Workers", fmt.Sprintf("%d", m.maxWorkers)},
		{settingsRole, "Role", m.settingsRoleValue()},
		{settingsScope, "Scope", m.settingsScopeValue()},
	}
	if m.scopeChoice == 2 {
		fields = append(fields, fieldEntry{settingsSubProject, "Sub-project", m.settingsSubProjValue()})
	}

	autoVal := "off"
	if m.autoStart {
		autoVal = "on"
	}
	fields = append(fields, fieldEntry{settingsAutoStart, "Auto-start", autoVal})

	row := boxY + 1
	for _, f := range fields {
		focused := m.settingsField == f.field
		if focused {
			selBg := tcell.StyleDefault.Background(tcellapp.ColorSurfaceHL)
			for col := innerX - 1; col < boxX+boxW-1; col++ {
				s.SetContent(col, row, ' ', nil, selBg)
			}
			lbl := fmt.Sprintf("▸ %-14s", f.label)
			x := tcellapp.DrawText(s, innerX, row, selBg.Bold(true).Foreground(tcellapp.ColorPrimary), lbl)
			valStyle := selBg.Foreground(tcellapp.ColorSuccess)
			if f.field == settingsAutoStart && !m.autoStart {
				valStyle = selBg.Foreground(tcellapp.ColorMuted)
			}
			x = tcellapp.DrawText(s, x+1, row, valStyle, f.value)
			hint := "◂ ▸"
			if f.field == settingsAutoStart {
				hint = "space"
			}
			tcellapp.DrawText(s, x+2, row, selBg.Foreground(tcellapp.ColorDimmer), hint)
		} else {
			lbl := fmt.Sprintf("  %-14s", f.label)
			x := tcellapp.DrawText(s, innerX, row, cardBg.Foreground(tcellapp.ColorNormal), lbl)
			valStyle := cardBg.Foreground(tcellapp.ColorNormal)
			if f.field == settingsAutoStart {
				if m.autoStart {
					valStyle = cardBg.Foreground(tcellapp.ColorSuccess)
				} else {
					valStyle = cardBg.Foreground(tcellapp.ColorDimmer)
				}
			}
			tcellapp.DrawText(s, x+1, row, valStyle, f.value)
		}
		row++
	}

	row++

	// Sync roles action
	if m.settingsField == settingsSyncRoles {
		selBg := tcell.StyleDefault.Background(tcellapp.ColorSurfaceHL)
		for col := innerX - 1; col < boxX+boxW-1; col++ {
			s.SetContent(col, row, ' ', nil, selBg)
		}
		tcellapp.DrawText(s, innerX, row, selBg.Foreground(tcellapp.ColorPrimary), "▸ Sync roles")
	} else {
		tcellapp.DrawText(s, innerX, row, cardBg.Foreground(tcellapp.ColorDimmer), "  Sync roles")
	}
	row += 2

	// Start Run / Close button
	startLabel := "Start Run"
	if m.running {
		startLabel = "Apply & Close"
	}
	if m.settingsField == settingsStart {
		selBg := tcell.StyleDefault.Background(tcell.NewRGBColor(0x1A, 0x33, 0x2A))
		for col := innerX - 1; col < boxX+boxW-1; col++ {
			s.SetContent(col, row, ' ', nil, selBg)
		}
		tcellapp.DrawText(s, innerX, row, selBg.Bold(true).Foreground(tcellapp.ColorSuccess), "▸ "+startLabel)
	} else {
		tcellapp.DrawText(s, innerX, row, cardBg.Foreground(tcellapp.ColorNormal), "  "+startLabel)
	}

	tcellapp.DrawFooterBar(s, h-1, w, "[tab/dn] next  [◂/▸] change  [enter] select  [esc] close")
}

func (m *MonitorModel) drawHeader(s tcell.Screen, w int) {
	bg := tcellapp.DrawHeaderBar(s, 0, w)

	// Logo
	x := tcellapp.DrawText(s, 2, 0, bg.Bold(true).Foreground(tcellapp.ColorPrimary), "agach")
	x = tcellapp.DrawText(s, x+1, 0, bg.Foreground(tcellapp.ColorDimmer), "│")
	x = tcellapp.DrawText(s, x+1, 0, bg.Foreground(tcellapp.ColorNormal), m.project.Name)

	// Status pill
	x += 2
	status := "RUNNING"
	statusColor := tcellapp.ColorSuccess
	if m.stopped {
		status = "STOPPED"
		statusColor = tcellapp.ColorError
	} else if m.paused {
		status = "PAUSED"
		statusColor = tcellapp.ColorWarning
	}
	x = tcellapp.DrawStatusPill(s, x, 0, status, statusColor)

	// Right side: elapsed + workers
	elapsed := time.Since(m.started).Round(time.Second)
	rightText := fmt.Sprintf("%d workers  %s", m.config.MaxWorkers, formatDuration(elapsed))
	tcellapp.DrawRightAlignedText(s, 0, 0, w-2, bg.Foreground(tcellapp.ColorMuted), rightText)
}

func (m *MonitorModel) drawStatsBar(s tcell.Screen, w int) {
	barBg := tcell.StyleDefault.Background(tcellapp.ColorSurfaceAlt)
	for row := 1; row < 5; row++ {
		for col := 0; col < w; col++ {
			s.SetContent(col, row, ' ', nil, barBg)
		}
	}

	// Compute totals
	var totalIn, totalOut, totalCacheR, totalCacheW int
	for _, wk := range m.workers {
		if wk.Current != nil {
			totalIn += wk.Current.InputTokens
			totalOut += wk.Current.OutputTokens
			totalCacheR += wk.Current.CacheReadInputTokens
			totalCacheW += wk.Current.CacheCreationInputTokens
		}
		for _, p := range wk.Past {
			totalIn += p.InputTokens
			totalOut += p.OutputTokens
			totalCacheR += p.CacheReadInputTokens
			totalCacheW += p.CacheCreationInputTokens
		}
	}

	// Thin separator
	for col := 0; col < w; col++ {
		s.SetContent(col, 1, '─', nil, barBg.Foreground(tcellapp.ColorDimmer))
	}

	// Token stats - row 2
	y := 2
	x := 3

	labelStyle := barBg.Foreground(tcellapp.ColorNormal)
	valStyle := barBg.Foreground(tcellapp.ColorPrimary)
	dimValStyle := barBg.Foreground(tcellapp.ColorMuted)

	x = tcellapp.DrawText(s, x, y, barBg.Bold(true).Foreground(tcellapp.ColorAccent), "TOKENS")
	x += 2

	x = tcellapp.DrawText(s, x, y, labelStyle, "in ")
	x = tcellapp.DrawText(s, x, y, valStyle, tcellapp.FormatTokens(totalIn))
	x += 2
	x = tcellapp.DrawText(s, x, y, labelStyle, "out ")
	x = tcellapp.DrawText(s, x, y, valStyle, tcellapp.FormatTokens(totalOut))
	x += 2
	x = tcellapp.DrawText(s, x, y, labelStyle, "cache_r ")
	x = tcellapp.DrawText(s, x, y, dimValStyle, tcellapp.FormatTokens(totalCacheR))
	x += 2
	x = tcellapp.DrawText(s, x, y, labelStyle, "cache_w ")
	_ = tcellapp.DrawText(s, x, y, dimValStyle, tcellapp.FormatTokens(totalCacheW))

	// Board stats - row 3
	y = 3
	x = 3
	c := m.columnCounts

	x = tcellapp.DrawText(s, x, y, barBg.Bold(true).Foreground(tcellapp.ColorAccent), "BOARD ")
	x += 2

	x = drawBoardStat(s, x, y, barBg, "todo", c.Todo, tcellapp.ColorNormal)
	x += 2
	x = drawBoardStat(s, x, y, barBg, "in_progress", c.InProgress, tcellapp.ColorRunning)
	x += 2
	x = drawBoardStat(s, x, y, barBg, "done", c.Done, tcellapp.ColorSuccess)
	x += 2
	_ = drawBoardStat(s, x, y, barBg, "blocked", c.Blocked, tcellapp.ColorError)

	// Thin separator bottom
	for col := 0; col < w; col++ {
		s.SetContent(col, 4, '─', nil, tcell.StyleDefault.Background(tcellapp.ColorSurface).Foreground(tcellapp.ColorDimmer))
	}
}

func drawBoardStat(s tcell.Screen, x, y int, bg tcell.Style, label string, count int, color tcell.Color) int {
	x = tcellapp.DrawText(s, x, y, bg.Foreground(tcellapp.ColorNormal), label+" ")
	return tcellapp.DrawText(s, x, y, bg.Bold(true).Foreground(color), fmt.Sprintf("%d", count))
}

// drawWorkersCompact draws workers as single-line rows (used in split layout)
func (m *MonitorModel) drawWorkersCompact(s tcell.Screen, ox, oy, w, h int) {
	bg := tcell.StyleDefault.Background(tcellapp.ColorSurface)

	row := oy
	for i, wk := range m.workers {
		if row >= oy+h {
			break
		}
		focused := i == m.focusedWorker

		rowBg := bg
		if focused {
			rowBg = tcell.StyleDefault.Background(tcellapp.ColorSurfaceHL)
		}

		// Clear row
		for col := ox; col < ox+w; col++ {
			s.SetContent(col, row, ' ', nil, rowBg)
		}

		x := ox + 2

		// Focus indicator
		if focused {
			tcellapp.DrawText(s, x, row, rowBg.Bold(true).Foreground(tcellapp.ColorPrimary), ">")
		}
		x += 2

		// Status icon
		icon, iconStyle := workerIcon(wk.Status)
		_, bgColor, _ := rowBg.Decompose()
		x = tcellapp.DrawText(s, x, row, iconStyle.Background(bgColor), icon)
		x++

		// Worker label
		x = tcellapp.DrawText(s, x, row, rowBg.Bold(true).Foreground(tcellapp.ColorNormal),
			fmt.Sprintf("W%d", wk.ID))
		x++

		if wk.Current != nil {
			t := wk.Current

			// Right-align token summary
			tokenSummary := fmt.Sprintf("↑%s ↓%s x%d",
				tcellapp.FormatTokens(t.InputTokens),
				tcellapp.FormatTokens(t.OutputTokens),
				t.Exchanges)
			elapsed := time.Since(t.StartedAt).Round(time.Second)
			rightInfo := fmt.Sprintf("%s  %s", tokenSummary, formatDuration(elapsed))
			rightW := len([]rune(rightInfo))
			tokensX := ox + w - rightW - 3
			if tokensX > x+2 {
				tcellapp.DrawText(s, tokensX, row, rowBg.Foreground(tcellapp.ColorInfo), tokenSummary)
				durationX := tokensX + len([]rune(tokenSummary)) + 2
				tcellapp.DrawText(s, durationX, row, rowBg.Foreground(tcellapp.ColorDimmer), formatDuration(elapsed))
			}

			// Task title
			titleMaxW := tokensX - x - 2
			if titleMaxW < 0 {
				titleMaxW = 0
			}
			title := tcellapp.Truncate(t.TaskTitle, titleMaxW)
			tcellapp.DrawText(s, x+1, row, rowBg.Foreground(tcellapp.ColorNormal), title)
		} else if wk.Status == domain.WorkerIdle {
			tcellapp.DrawText(s, x+1, row, rowBg.Foreground(tcellapp.ColorMuted), "waiting for task...")
		}

		row++
	}
}

func (m *MonitorModel) drawWorkers(s tcell.Screen, ox, oy, w, h int) {
	row := oy
	for i, wk := range m.workers {
		if row >= oy+h-1 {
			break
		}
		focused := i == m.focusedWorker
		row = m.drawWorkerCard(s, wk, focused, ox, row, w)
	}
}

func (m *MonitorModel) drawWorkerCard(s tcell.Screen, wk domain.WorkerState, focused bool, ox, row, w int) int {
	cardX := ox + 1
	cardW := w - 2
	cardH := 3

	borderColor := tcellapp.ColorCardBorder
	if focused {
		borderColor = tcellapp.ColorCardFocused
	}
	borderStyle := tcell.StyleDefault.Foreground(borderColor).Background(tcellapp.ColorSurface)

	cardBg := tcell.StyleDefault.Background(tcellapp.ColorCardBg)

	tcellapp.DrawBox(s, cardX, row, cardW, cardH, borderStyle)
	tcellapp.FillInner(s, cardX, row, cardW, cardH, cardBg)

	contentRow := row + 1
	x := cardX + 2

	// Focus indicator
	if focused {
		tcellapp.DrawText(s, x, contentRow, cardBg.Bold(true).Foreground(tcellapp.ColorPrimary), ">")
		x += 2
	} else {
		x += 2
	}

	// Status icon
	icon, iconStyle := workerIcon(wk.Status)
	iconStyle = addBg(iconStyle, tcellapp.ColorCardBg)
	x = tcellapp.DrawText(s, x, contentRow, iconStyle, icon)
	x++

	// Worker label
	x = tcellapp.DrawText(s, x, contentRow, cardBg.Bold(true).Foreground(tcellapp.ColorNormal),
		fmt.Sprintf("W%d", wk.ID))
	x++

	if wk.Current != nil {
		t := wk.Current

		// Right-align token summary
		tokenSummary := fmt.Sprintf("↑%s ↓%s x%d",
			tcellapp.FormatTokens(t.InputTokens),
			tcellapp.FormatTokens(t.OutputTokens),
			t.Exchanges)
		elapsed := time.Since(t.StartedAt).Round(time.Second)
		rightInfo := fmt.Sprintf("%s  %s", tokenSummary, formatDuration(elapsed))
		rightW := len([]rune(rightInfo))
		tokensX := cardX + cardW - rightW - 3
		if tokensX > x+2 {
			tcellapp.DrawText(s, tokensX, contentRow, cardBg.Foreground(tcellapp.ColorInfo), tokenSummary)
			durationX := tokensX + len([]rune(tokenSummary)) + 2
			tcellapp.DrawText(s, durationX, contentRow, cardBg.Foreground(tcellapp.ColorDimmer), formatDuration(elapsed))
		}

		// Task title
		titleMaxW := tokensX - x - 2
		if titleMaxW < 0 {
			titleMaxW = 0
		}
		title := tcellapp.Truncate(t.TaskTitle, titleMaxW)
		x = tcellapp.DrawText(s, x+1, contentRow, cardBg.Foreground(tcellapp.ColorNormal), title)
		if t.SessionID != "" && x+10 < tokensX {
			tcellapp.DrawText(s, x+1, contentRow, cardBg.Foreground(tcellapp.ColorDimmer), t.SessionID[:min(8, len(t.SessionID))])
		}
	} else if wk.Status == domain.WorkerIdle {
		tcellapp.DrawText(s, x+1, contentRow, cardBg.Foreground(tcellapp.ColorMuted), "waiting for task...")
	}

	return row + cardH
}

func (m *MonitorModel) drawCompletedPanel(s tcell.Screen, ox, oy, w, h int) {
	// Vertical separator
	sepStyle := tcell.StyleDefault.Background(tcellapp.ColorSurface).Foreground(tcellapp.ColorDimmer)
	for row := oy; row < oy+h; row++ {
		s.SetContent(ox, row, '|', nil, sepStyle)
	}

	panelX := ox + 1
	panelW := w - 1

	// Panel title
	titleBg := tcell.StyleDefault.Background(tcellapp.ColorSurface)
	for col := panelX; col < panelX+panelW; col++ {
		s.SetContent(col, oy, ' ', nil, titleBg)
	}
	tcellapp.DrawText(s, panelX+1, oy, titleBg.Bold(true).Foreground(tcellapp.ColorAccent), "COMPLETED TASKS")

	// Separator under title
	for col := panelX; col < panelX+panelW; col++ {
		s.SetContent(col, oy+1, '─', nil, tcell.StyleDefault.Background(tcellapp.ColorSurface).Foreground(tcellapp.ColorDimmer))
	}

	// Collect all past tasks across all workers
	type pastEntry struct {
		workerID int
		task     domain.TaskRun
	}
	var allPast []pastEntry
	for _, wk := range m.workers {
		for _, t := range wk.Past {
			allPast = append(allPast, pastEntry{workerID: wk.ID, task: t})
		}
	}

	// Sort by completion time (most recent first)
	for i := 0; i < len(allPast); i++ {
		for j := i + 1; j < len(allPast); j++ {
			ti := allPast[i].task.CompletedAt
			tj := allPast[j].task.CompletedAt
			if ti != nil && tj != nil && tj.After(*ti) {
				allPast[i], allPast[j] = allPast[j], allPast[i]
			}
		}
	}

	if len(allPast) == 0 {
		tcellapp.DrawText(s, panelX+2, oy+3, tcell.StyleDefault.Background(tcellapp.ColorSurface).Foreground(tcellapp.ColorDimmer), "No completed tasks yet")
		return
	}

	row := oy + 2
	maxRow := oy + h
	bg := tcell.StyleDefault.Background(tcellapp.ColorSurface)

	for _, entry := range allPast {
		if row >= maxRow-1 {
			break
		}

		// Clear row
		for col := panelX; col < panelX+panelW; col++ {
			s.SetContent(col, row, ' ', nil, bg)
		}

		t := entry.task
		x := panelX + 1

		// Status badge
		statusIcon, statusColor := taskStatusDisplay(t)
		x = tcellapp.DrawText(s, x, row, bg.Foreground(statusColor), statusIcon)
		x++

		// Worker tag
		wTag := fmt.Sprintf("W%d", entry.workerID)
		x = tcellapp.DrawText(s, x, row, bg.Foreground(tcellapp.ColorDimmer), wTag)
		x++

		// Duration
		if t.CompletedAt != nil {
			dur := t.CompletedAt.Sub(t.StartedAt).Round(time.Second)
			durStr := formatDuration(dur)
			x = tcellapp.DrawText(s, x, row, bg.Foreground(tcellapp.ColorMuted), durStr)
			x++
		}

		// Task title (remaining space)
		titleW := panelW - (x - panelX) - 1
		if titleW > 0 {
			title := tcellapp.Truncate(t.TaskTitle, titleW)
			tcellapp.DrawText(s, x, row, bg.Foreground(tcellapp.ColorNormal), title)
		}

		row++

		// Token detail line
		if row < maxRow-1 {
			for col := panelX; col < panelX+panelW; col++ {
				s.SetContent(col, row, ' ', nil, bg)
			}
			tokenLine := fmt.Sprintf("    ↑%s ↓%s  r:%s w:%s  x%d",
				tcellapp.FormatTokens(t.InputTokens),
				tcellapp.FormatTokens(t.OutputTokens),
				tcellapp.FormatTokens(t.CacheReadInputTokens),
				tcellapp.FormatTokens(t.CacheCreationInputTokens),
				t.Exchanges)
			tcellapp.DrawText(s, panelX+1, row, bg.Foreground(tcellapp.ColorDimmer), tokenLine)
			row++
		}
	}
}

func (m *MonitorModel) drawConsoleOverlay(s tcell.Screen, oy, w, h int) {
	bg := tcell.StyleDefault.Background(tcellapp.ColorSurface)

	// Fill background
	for row := oy; row < oy+h; row++ {
		for col := 0; col < w; col++ {
			s.SetContent(col, row, ' ', nil, bg)
		}
	}

	row := oy + 1
	tcellapp.DrawText(s, 3, row, bg.Bold(true).Foreground(tcellapp.ColorAccent), "RESUME SESSIONS")
	row += 2

	wk := m.workers[m.focusedWorker]

	// Current task
	if wk.Current != nil && wk.Current.SessionID != "" {
		tcellapp.DrawText(s, 3, row, bg.Bold(true).Foreground(tcellapp.ColorNormal),
			fmt.Sprintf("Worker %d  (current)", wk.ID))
		row++
		row = m.drawConsoleEntry(s, row, w, wk.Current, bg)
		row++
	}

	// Past tasks with sessions
	for _, t := range wk.Past {
		if t.SessionID == "" || row >= oy+h-2 {
			continue
		}
		statusIcon, statusColor := taskStatusDisplay(t)
		tcellapp.DrawText(s, 3, row, bg.Foreground(statusColor), statusIcon)
		tcellapp.DrawText(s, 5, row, bg.Foreground(tcellapp.ColorNormal),
			tcellapp.Truncate(t.TaskTitle, w-8))
		row++
		row = m.drawConsoleEntry(s, row, w, &t, bg)
		row++
	}

	if (wk.Current == nil || wk.Current.SessionID == "") && len(wk.Past) == 0 {
		tcellapp.DrawText(s, 3, row, bg.Foreground(tcellapp.ColorDimmer),
			"No sessions available for this worker")
	}
}

func (m *MonitorModel) drawConsoleEntry(s tcell.Screen, row, w int, t *domain.TaskRun, bg tcell.Style) int {
	workDir := m.app.workDir
	codeBg := tcell.StyleDefault.Background(tcellapp.ColorSurfaceAlt)

	// Work directory
	tcellapp.DrawText(s, 5, row, bg.Foreground(tcellapp.ColorDimmer), "cd "+workDir)
	row++

	// Command
	cmd := fmt.Sprintf("claude --resume %s", t.SessionID)
	for col := 5; col < w-2; col++ {
		s.SetContent(col, row, ' ', nil, codeBg)
	}
	tcellapp.DrawText(s, 6, row, codeBg.Foreground(tcellapp.ColorInfo), "$ "+cmd)
	row++

	return row
}

func (m *MonitorModel) drawPastTasks(s tcell.Screen, oy, w, h int) {
	wk := m.workers[m.focusedWorker]
	if len(wk.Past) == 0 {
		return
	}

	bg := tcell.StyleDefault.Background(tcellapp.ColorSurface)

	// Fill background
	for row := oy; row < oy+h; row++ {
		for col := 0; col < w; col++ {
			s.SetContent(col, row, ' ', nil, bg)
		}
	}

	title := fmt.Sprintf("WORKER %d - PAST TASKS (%d)", wk.ID, len(wk.Past))
	tcellapp.DrawText(s, 3, oy, bg.Bold(true).Foreground(tcellapp.ColorAccent), title)

	// Separator
	for col := 1; col < w-1; col++ {
		s.SetContent(col, oy+1, '─', nil, bg.Foreground(tcellapp.ColorDimmer))
	}

	row := oy + 2
	endRow := oy + h

	for i, t := range wk.Past {
		if row >= endRow-1 {
			break
		}

		selected := i == m.pastCursor
		rowBg := bg
		if selected {
			rowBg = tcell.StyleDefault.Background(tcellapp.ColorSurfaceHL)
			for col := 1; col < w-1; col++ {
				s.SetContent(col, row, ' ', nil, rowBg)
			}
		}

		x := 3

		// Selection indicator
		if selected {
			tcellapp.DrawText(s, x, row, rowBg.Bold(true).Foreground(tcellapp.ColorPrimary), ">")
		}
		x += 2

		// Status
		statusIcon, statusColor := taskStatusDisplay(t)
		x = tcellapp.DrawText(s, x, row, rowBg.Foreground(statusColor), statusIcon)
		x++

		// Duration
		if t.CompletedAt != nil {
			dur := t.CompletedAt.Sub(t.StartedAt).Round(time.Second)
			durStr := formatDuration(dur)
			x = tcellapp.DrawText(s, x, row, rowBg.Foreground(tcellapp.ColorMuted), durStr)
			x++
		}

		// Task title
		titleW := w - x - 4
		if titleW > 0 {
			title := tcellapp.Truncate(t.TaskTitle, titleW)
			tcellapp.DrawText(s, x, row, rowBg.Foreground(tcellapp.ColorNormal), title)
		}
		row++

		// Expanded detail for selected task
		if selected {
			if row < endRow {
				detailBg := rowBg
				for col := 1; col < w-1; col++ {
					s.SetContent(col, row, ' ', nil, detailBg)
				}
				detail := fmt.Sprintf("     in: %s  out: %s  cache_r: %s  cache_w: %s  exchanges: %d",
					tcellapp.FormatTokens(t.InputTokens),
					tcellapp.FormatTokens(t.OutputTokens),
					tcellapp.FormatTokens(t.CacheReadInputTokens),
					tcellapp.FormatTokens(t.CacheCreationInputTokens),
					t.Exchanges)
				tcellapp.DrawText(s, 5, row, detailBg.Foreground(tcellapp.ColorInfo), detail)
				row++
			}
			if t.ColdStartCaptured && row < endRow {
				for col := 1; col < w-1; col++ {
					s.SetContent(col, row, ' ', nil, rowBg)
				}
				coldLine := fmt.Sprintf("     cold_start  in: %s  out: %s  cache_r: %s  cache_w: %s",
					tcellapp.FormatTokens(t.ColdStartInputTokens),
					tcellapp.FormatTokens(t.ColdStartOutputTokens),
					tcellapp.FormatTokens(t.ColdStartCacheReadInputTokens),
					tcellapp.FormatTokens(t.ColdStartCacheCreationInputTokens))
				tcellapp.DrawText(s, 5, row, rowBg.Foreground(tcellapp.ColorDimmer), coldLine)
				row++
			}
			if t.SessionID != "" && row < endRow {
				for col := 1; col < w-1; col++ {
					s.SetContent(col, row, ' ', nil, rowBg)
				}
				tcellapp.DrawText(s, 5, row, rowBg.Foreground(tcellapp.ColorDimmer),
					"session: "+t.SessionID)
				row++
			}
			if t.Error != "" && row < endRow {
				for col := 1; col < w-1; col++ {
					s.SetContent(col, row, ' ', nil, rowBg)
				}
				tcellapp.DrawText(s, 5, row, rowBg.Foreground(tcellapp.ColorError), "error: "+t.Error)
				row++
			}
		}
	}
}

// ── Done tasks panel ─────────────────────────────────────────────────────────

type doneEntry struct {
	workerID int
	task     domain.TaskRun
}

func (m *MonitorModel) collectAllPast() []doneEntry {
	var all []doneEntry
	for _, wk := range m.workers {
		for _, t := range wk.Past {
			all = append(all, doneEntry{workerID: wk.ID, task: t})
		}
	}
	// Sort by completion time (most recent first)
	for i := 0; i < len(all); i++ {
		for j := i + 1; j < len(all); j++ {
			ti := all[i].task.CompletedAt
			tj := all[j].task.CompletedAt
			if ti != nil && tj != nil && tj.After(*ti) {
				all[i], all[j] = all[j], all[i]
			}
		}
	}
	return all
}

func (m *MonitorModel) handleDoneKey(ks string) (tcellapp.Screen, tcellapp.Cmd) {
	allPast := m.collectAllPast()
	switch ks {
	case "esc", "d":
		m.showDone = false
		m.doneErr = ""
	case "up", "k":
		if m.doneCursor > 0 {
			m.doneCursor--
			m.doneErr = ""
		}
	case "down", "j":
		if m.doneCursor < len(allPast)-1 {
			m.doneCursor++
			m.doneErr = ""
		}
	case "enter":
		if m.doneCursor < len(allPast) {
			entry := allPast[m.doneCursor]
			if entry.task.SessionID == "" {
				m.doneErr = "no session ID on this task"
				return m, nil
			}
			sessionID := entry.task.SessionID
			workDir := m.app.workDir
			return m, func() tcellapp.Msg {
				err := openTerminalForSession(workDir, sessionID)
				return terminalResultMsg{err: err}
			}
		}
	}
	return m, nil
}

// resolveTerminal returns the short name of the terminal emulator from env vars.
// Handles full paths (/usr/bin/kitty -> kitty) and TERM values (xterm-256color -> xterm).
func resolveTerminal() string {
	for _, env := range []string{"TERM_PROGRAM", "TERMINAL"} {
		v := os.Getenv(env)
		if v == "" {
			continue
		}
		// Extract basename and strip extension: /usr/bin/kitty -> kitty
		base := filepath.Base(v)
		return strings.TrimSuffix(base, filepath.Ext(base))
	}
	// Fallback: parse TERM (e.g. "xterm-256color" -> "xterm", "kitty" -> "kitty")
	t := os.Getenv("TERM")
	if t != "" {
		for _, known := range []string{"kitty", "alacritty", "wezterm"} {
			if strings.HasPrefix(t, known) {
				return known
			}
		}
		// "xterm-256color" -> "xterm"
		if i := strings.IndexByte(t, '-'); i > 0 {
			return t[:i]
		}
		return t
	}
	return ""
}

func openTerminalForSession(workDir, sessionID string) string {
	resumeShell := "claude --resume " + sessionID

	type termEntry struct {
		name string
		args []string
	}
	terminals := []termEntry{
		{"kitty", []string{"--directory", workDir, "--hold", "sh", "-c", resumeShell}},
		{"alacritty", []string{"--working-directory", workDir, "-e", "sh", "-c", resumeShell}},
		{"wezterm", []string{"start", "--cwd", workDir, "--", "sh", "-c", resumeShell}},
		{"konsole", []string{"--workdir", workDir, "-e", "sh", "-c", resumeShell}},
		{"gnome-terminal", []string{"--working-directory=" + workDir, "--", "sh", "-c", resumeShell}},
		{"xterm", []string{"-e", "sh", "-c", resumeShell}},
	}

	detected := resolveTerminal()

	// If detected terminal is not in our list, add it with generic -e flag
	if detected != "" {
		found := false
		for i, t := range terminals {
			if t.name == detected {
				terminals = append([]termEntry{t}, append(terminals[:i], terminals[i+1:]...)...)
				found = true
				break
			}
		}
		if !found {
			terminals = append([]termEntry{
				{detected, []string{"-e", "sh", "-c", resumeShell}},
			}, terminals...)
		}
	}

	var tried []string
	for _, t := range terminals {
		path, err := exec.LookPath(t.name)
		if err != nil {
			tried = append(tried, fmt.Sprintf("%s: not found", t.name))
			continue
		}
		cmd := exec.Command(path, t.args...)
		cmd.Dir = workDir
		if err := cmd.Start(); err != nil {
			tried = append(tried, fmt.Sprintf("%s: start failed: %v", t.name, err))
			continue
		}
		return "" // success
	}

	envInfo := fmt.Sprintf("TERM_PROGRAM=%q TERMINAL=%q TERM=%q detected=%q",
		os.Getenv("TERM_PROGRAM"), os.Getenv("TERMINAL"), os.Getenv("TERM"), detected)
	return fmt.Sprintf("no terminal found. %s | tried: %s", envInfo, strings.Join(tried, "; "))
}

func (m *MonitorModel) drawDonePanel(s tcell.Screen, oy, w, h int) {
	bg := tcell.StyleDefault.Background(tcellapp.ColorSurface)

	// Fill background
	for row := oy; row < oy+h; row++ {
		for col := 0; col < w; col++ {
			s.SetContent(col, row, ' ', nil, bg)
		}
	}

	allPast := m.collectAllPast()

	tcellapp.DrawText(s, 3, oy, bg.Bold(true).Foreground(tcellapp.ColorAccent), "COMPLETED TASKS")

	if len(allPast) > 0 {
		countText := fmt.Sprintf("(%d)", len(allPast))
		tcellapp.DrawText(s, 20, oy, bg.Foreground(tcellapp.ColorDimmer), countText)
	}

	// Separator
	for col := 1; col < w-1; col++ {
		s.SetContent(col, oy+1, '─', nil, bg.Foreground(tcellapp.ColorDimmer))
	}

	startRow := oy + 2

	// Show error from terminal launch attempt
	if m.doneErr != "" {
		errText := tcellapp.Truncate(m.doneErr, w-6)
		tcellapp.DrawText(s, 3, startRow, bg.Foreground(tcellapp.ColorError), errText)
		startRow++
	}

	if len(allPast) == 0 {
		tcellapp.DrawText(s, 3, startRow+1, bg.Foreground(tcellapp.ColorDimmer), "No completed tasks yet")
		return
	}

	row := startRow
	endRow := oy + h

	for i, entry := range allPast {
		if row >= endRow-1 {
			break
		}

		selected := i == m.doneCursor
		rowBg := bg
		if selected {
			rowBg = tcell.StyleDefault.Background(tcellapp.ColorSurfaceHL)
			for col := 1; col < w-1; col++ {
				s.SetContent(col, row, ' ', nil, rowBg)
			}
		}

		x := 3

		// Selection indicator
		if selected {
			tcellapp.DrawText(s, x, row, rowBg.Bold(true).Foreground(tcellapp.ColorPrimary), ">")
		}
		x += 2

		t := entry.task

		// Status
		statusIcon, statusColor := taskStatusDisplay(t)
		x = tcellapp.DrawText(s, x, row, rowBg.Foreground(statusColor), statusIcon)
		x++

		// Worker tag
		x = tcellapp.DrawText(s, x, row, rowBg.Foreground(tcellapp.ColorDimmer), fmt.Sprintf("W%d", entry.workerID))
		x++

		// Duration
		if t.CompletedAt != nil {
			dur := t.CompletedAt.Sub(t.StartedAt).Round(time.Second)
			x = tcellapp.DrawText(s, x, row, rowBg.Foreground(tcellapp.ColorMuted), formatDuration(dur))
			x++
		}

		// Task title
		titleW := w - x - 4
		if titleW > 0 {
			title := tcellapp.Truncate(t.TaskTitle, titleW)
			tcellapp.DrawText(s, x, row, rowBg.Foreground(tcellapp.ColorNormal), title)
		}
		row++

		// Expanded detail for selected task
		if selected {
			// Token details
			if row < endRow {
				for col := 1; col < w-1; col++ {
					s.SetContent(col, row, ' ', nil, rowBg)
				}
				detail := fmt.Sprintf("     in: %s  out: %s  cache_r: %s  cache_w: %s  exchanges: %d",
					tcellapp.FormatTokens(t.InputTokens),
					tcellapp.FormatTokens(t.OutputTokens),
					tcellapp.FormatTokens(t.CacheReadInputTokens),
					tcellapp.FormatTokens(t.CacheCreationInputTokens),
					t.Exchanges)
				tcellapp.DrawText(s, 5, row, rowBg.Foreground(tcellapp.ColorInfo), detail)
				row++
			}
			// Session + resume hint
			if t.SessionID != "" && row < endRow {
				for col := 1; col < w-1; col++ {
					s.SetContent(col, row, ' ', nil, rowBg)
				}
				tcellapp.DrawText(s, 5, row, rowBg.Foreground(tcellapp.ColorDimmer),
					"session: "+t.SessionID)
				row++
				if row < endRow {
					for col := 1; col < w-1; col++ {
						s.SetContent(col, row, ' ', nil, rowBg)
					}
					tcellapp.DrawText(s, 5, row, rowBg.Foreground(tcellapp.ColorAccent),
						"[enter] open terminal with claude --resume")
					row++
				}
			}
			// Error
			if t.Error != "" && row < endRow {
				for col := 1; col < w-1; col++ {
					s.SetContent(col, row, ' ', nil, rowBg)
				}
				tcellapp.DrawText(s, 5, row, rowBg.Foreground(tcellapp.ColorError), "error: "+t.Error)
				row++
			}
		}
	}
}

// ── Helpers ──────────────────────────────────────────────────────────────────

func workerIcon(status domain.WorkerStatus) (string, tcell.Style) {
	switch status {
	case domain.WorkerRunning:
		return "●", tcellapp.StyleWorkerRunning()
	case domain.WorkerDone:
		return "+", tcellapp.StyleWorkerDone()
	case domain.WorkerError:
		return "x", tcellapp.StyleWorkerError()
	default:
		return "o", tcellapp.StyleWorkerIdle()
	}
}

func taskStatusDisplay(t domain.TaskRun) (string, tcell.Color) {
	switch t.Status {
	case domain.WorkerDone:
		return "+", tcellapp.ColorSuccess
	case domain.WorkerError:
		return "x", tcellapp.ColorError
	default:
		return "-", tcellapp.ColorMuted
	}
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		m := int(d.Minutes())
		s := int(d.Seconds()) % 60
		return fmt.Sprintf("%dm%02ds", m, s)
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	return fmt.Sprintf("%dh%02dm", h, m)
}

func addBg(style tcell.Style, bg tcell.Color) tcell.Style {
	return style.Background(bg)
}
