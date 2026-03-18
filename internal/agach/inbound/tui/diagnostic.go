package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"

	appagach "github.com/JLugagne/agach-mcp/internal/agach/app"
	"github.com/JLugagne/agach-mcp/internal/agach/domain"
	"github.com/JLugagne/agach-mcp/internal/agach/inbound/tui/tcellapp"
)

// launchDiagnosticMsg triggers the diagnostic screen from the welcome screen
type launchDiagnosticMsg struct{}

// diagnosticUpdateMsg wraps a DiagnosticUpdate for the TUI event loop
type diagnosticUpdateMsg domain.DiagnosticUpdate

// DiagnosticModel shows cold-start token measurements for each agent
type DiagnosticModel struct {
	app     *tuiApp
	results []domain.DiagnosticResult
	done    bool
	cancel  context.CancelFunc
	cursor  int
}

func newDiagnosticModel(app *tuiApp) *DiagnosticModel {
	return &DiagnosticModel{app: app}
}

func (m *DiagnosticModel) Init() tcellapp.Cmd {
	agents := appagach.DiscoverAgents(m.app.workDir)
	tApp := m.app.tcellApp
	workDir := m.app.workDir
	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel
	return func() tcellapp.Msg {
		ch := make(chan domain.DiagnosticUpdate, 8)
		go appagach.RunDiagnostic(ctx, workDir, agents, ch)
		go func() {
			for upd := range ch {
				tApp.Dispatch(diagnosticUpdateMsg(upd))
			}
		}()
		return nil
	}
}

func (m *DiagnosticModel) HandleMsg(msg tcellapp.Msg) (tcellapp.Screen, tcellapp.Cmd) {
	switch msg := msg.(type) {
	case diagnosticUpdateMsg:
		m.results = msg.Results
		m.done = msg.Done
		return m, nil
	case tcellapp.KeyMsg:
		ks := tcellapp.KeyString(msg)
		switch ks {
		case "esc":
			if m.cancel != nil {
				m.cancel()
			}
			return m, func() tcellapp.Msg { return backToWelcomeMsg{} }
		case "q":
			if m.done {
				return m, func() tcellapp.Msg { return backToWelcomeMsg{} }
			}
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.results)-1 {
				m.cursor++
			}
		}
	}
	return m, nil
}

func (m *DiagnosticModel) Draw(s tcell.Screen, w, h int) {
	tcellapp.Fill(s, 0, 0, w, h, tcell.StyleDefault.Background(tcellapp.ColorSurface))
	surfBg := tcell.StyleDefault.Background(tcellapp.ColorSurface)

	cy := 1
	titleStyle := surfBg.Bold(true).Foreground(tcellapp.ColorPrimary)
	tcellapp.DrawCenteredText(s, 0, cy, w, titleStyle, "Token Diagnostic")
	cy++
	tcellapp.DrawCenteredText(s, 0, cy, w, tcellapp.StyleDim(), "Cold-start cost per agent")
	cy += 2

	if len(m.results) == 0 {
		tcellapp.DrawCenteredText(s, 0, cy, w, tcellapp.StyleDim(), "Discovering agents...")
		return
	}

	// Layout: left table | right detail panel
	detailW := 40
	tableW := w - detailW - 3 // 3 for separator + margins
	if tableW < 60 {
		// Not enough space for detail panel — full-width table
		detailW = 0
		tableW = w - 4
	}
	tableX := 2

	// ── Table ──────────────────────────────
	colAgent := 18
	colTokens := 8
	colDelta := 9
	colTime := 7

	headerStyle := surfBg.Bold(true).Foreground(tcellapp.ColorAccent)
	x := tableX
	x = diagDrawCell(s, x, cy, colAgent, headerStyle, "Agent")
	x = diagDrawCell(s, x, cy, colTokens, headerStyle, "Input")
	x = diagDrawCell(s, x, cy, colDelta, headerStyle, "Δ Base")
	x = diagDrawCell(s, x, cy, colTokens, headerStyle, "Output")
	x = diagDrawCell(s, x, cy, colTokens, headerStyle, "Cache R")
	x = diagDrawCell(s, x, cy, colTokens, headerStyle, "Cache W")
	diagDrawCell(s, x, cy, colTime, headerStyle, "Time")
	cy++

	sepStyle := surfBg.Foreground(tcellapp.ColorDimmer)
	tcellapp.DrawText(s, tableX, cy, sepStyle, strings.Repeat("─", tableW))
	cy++

	var baselineInput int
	if len(m.results) > 0 && m.results[0].Status == domain.DiagnosticDone {
		baselineInput = m.results[0].InputTokens
	}

	tableStartY := cy
	for i, r := range m.results {
		if cy >= h-2 {
			break
		}

		x = tableX
		name := r.AgentSlug
		if name == "" {
			name = "(baseline)"
		}

		isFocused := i == m.cursor

		// Row highlight
		if isFocused {
			hlBg := tcell.StyleDefault.Background(tcellapp.ColorCardFocused)
			for col := tableX; col < tableX+tableW; col++ {
				s.SetContent(col, cy, ' ', nil, hlBg)
			}
		}

		rowBg := surfBg
		if isFocused {
			rowBg = tcell.StyleDefault.Background(tcellapp.ColorCardFocused)
		}

		switch r.Status {
		case domain.DiagnosticPending:
			dimStyle := rowBg.Foreground(tcellapp.ColorDimmer)
			x = diagDrawCell(s, x, cy, colAgent, dimStyle, name)
			diagDrawCell(s, x, cy, colTokens, dimStyle, "...")
		case domain.DiagnosticRunning:
			runStyle := rowBg.Foreground(tcellapp.ColorWarning)
			x = diagDrawCell(s, x, cy, colAgent, runStyle, name)
			diagDrawCell(s, x, cy, colTokens*4+colDelta+colTime, runStyle, "running...")
		case domain.DiagnosticError:
			errStyle := rowBg.Foreground(tcellapp.ColorError)
			x = diagDrawCell(s, x, cy, colAgent, errStyle, name)
			diagDrawCell(s, x, cy, tableW-colAgent, errStyle, "err: "+tcellapp.Truncate(r.Error, 40))
		case domain.DiagnosticDone:
			nameStyle := rowBg.Foreground(tcellapp.ColorNormal)
			valStyle := rowBg.Foreground(tcellapp.ColorNormal)

			x = diagDrawCell(s, x, cy, colAgent, nameStyle, name)
			x = diagDrawCell(s, x, cy, colTokens, valStyle, tcellapp.FormatTokens(r.InputTokens))

			if r.AgentSlug == "" {
				x = diagDrawCell(s, x, cy, colDelta, rowBg.Foreground(tcellapp.ColorDimmer), "—")
			} else {
				delta := r.InputTokens - baselineInput
				var deltaStr string
				deltaStyle := valStyle
				if delta >= 0 {
					deltaStr = "+" + tcellapp.FormatTokens(delta)
					if delta > 5000 {
						deltaStyle = rowBg.Foreground(tcellapp.ColorWarning)
					}
					if delta > 10000 {
						deltaStyle = rowBg.Foreground(tcellapp.ColorError)
					}
				} else {
					deltaStr = "-" + tcellapp.FormatTokens(-delta)
					deltaStyle = rowBg.Foreground(tcellapp.ColorSuccess)
				}
				x = diagDrawCell(s, x, cy, colDelta, deltaStyle, deltaStr)
			}

			x = diagDrawCell(s, x, cy, colTokens, valStyle, tcellapp.FormatTokens(r.OutputTokens))
			x = diagDrawCell(s, x, cy, colTokens, valStyle, tcellapp.FormatTokens(r.CacheReadInputTokens))
			x = diagDrawCell(s, x, cy, colTokens, valStyle, tcellapp.FormatTokens(r.CacheCreationInputTokens))
			diagDrawCell(s, x, cy, colTime, valStyle, diagFormatDuration(r.Duration))
		}
		cy++
	}

	// ── Detail panel ──────────────────────────
	if detailW > 0 && m.cursor < len(m.results) {
		detailX := tableX + tableW + 1
		// Vertical separator
		for row := tableStartY - 2; row < h-1; row++ {
			s.SetContent(detailX-1, row, '│', nil, surfBg.Foreground(tcellapp.ColorDimmer))
		}
		m.drawDetail(s, detailX, tableStartY-2, detailW, h-tableStartY, m.results[m.cursor])
	}

	// Footer
	if m.done {
		tcellapp.DrawFooterBar(s, h-1, w, "[j/k] navigate  [esc/q] back")
	} else {
		tcellapp.DrawFooterBar(s, h-1, w, "[j/k] navigate  [esc] cancel  ·  running probes...")
	}
}

func (m *DiagnosticModel) drawDetail(s tcell.Screen, x, y, w, maxH int, r domain.DiagnosticResult) {
	surfBg := tcell.StyleDefault.Background(tcellapp.ColorSurface)
	labelStyle := surfBg.Foreground(tcellapp.ColorMuted)
	valStyle := surfBg.Foreground(tcellapp.ColorNormal)
	headerStyle := surfBg.Bold(true).Foreground(tcellapp.ColorAccent)
	dimStyle := surfBg.Foreground(tcellapp.ColorDimmer)

	cy := y

	// Title
	name := r.AgentSlug
	if name == "" {
		name = "(baseline)"
	}
	tcellapp.DrawText(s, x, cy, headerStyle, name)
	cy += 2

	if r.Status != domain.DiagnosticDone {
		tcellapp.DrawText(s, x, cy, dimStyle, string(r.Status))
		return
	}

	// Token breakdown
	tcellapp.DrawText(s, x, cy, labelStyle, "Tokens")
	cy++
	nonCached := r.InputTokens - r.CacheReadInputTokens - r.CacheCreationInputTokens
	lines := []struct{ label, value string }{
		{"  Total input", tcellapp.FormatTokens(r.InputTokens)},
		{"  Non-cached", tcellapp.FormatTokens(nonCached)},
		{"  Cache read", tcellapp.FormatTokens(r.CacheReadInputTokens)},
		{"  Cache create", tcellapp.FormatTokens(r.CacheCreationInputTokens)},
		{"  Output", tcellapp.FormatTokens(r.OutputTokens)},
	}
	for _, l := range lines {
		if cy >= y+maxH {
			return
		}
		tcellapp.DrawText(s, x, cy, dimStyle, l.label)
		tcellapp.DrawText(s, x+w-len([]rune(l.value))-1, cy, valStyle, l.value)
		cy++
	}
	cy++

	// Cost & model
	if r.CostUSD > 0 {
		if cy >= y+maxH {
			return
		}
		tcellapp.DrawText(s, x, cy, labelStyle, "Cost")
		cost := fmt.Sprintf("$%.4f", r.CostUSD)
		tcellapp.DrawText(s, x+w-len(cost)-1, cy, valStyle, cost)
		cy++
	}
	if r.Model != "" {
		if cy >= y+maxH {
			return
		}
		tcellapp.DrawText(s, x, cy, labelStyle, "Model")
		model := tcellapp.Truncate(r.Model, w-8)
		tcellapp.DrawText(s, x+w-len([]rune(model))-1, cy, valStyle, model)
		cy++
	}
	if r.Duration > 0 {
		if cy >= y+maxH {
			return
		}
		tcellapp.DrawText(s, x, cy, labelStyle, "Duration")
		dur := diagFormatDuration(r.Duration)
		tcellapp.DrawText(s, x+w-len(dur)-1, cy, valStyle, dur)
		cy++
	}
	cy++

	// Tools breakdown
	if cy >= y+maxH {
		return
	}
	tcellapp.DrawText(s, x, cy, labelStyle, "Tools")
	cy++
	toolLines := []struct{ label, value string }{
		{"  System", fmt.Sprintf("%d", r.SystemToolCount)},
		{"  MCP", fmt.Sprintf("%d", r.MCPToolCount)},
	}
	for _, l := range toolLines {
		if cy >= y+maxH {
			return
		}
		tcellapp.DrawText(s, x, cy, dimStyle, l.label)
		tcellapp.DrawText(s, x+w-len(l.value)-1, cy, valStyle, l.value)
		cy++
	}

	// MCP tool list
	if len(r.MCPTools) > 0 {
		cy++
		if cy >= y+maxH {
			return
		}
		tcellapp.DrawText(s, x, cy, labelStyle, "MCP Tools")
		cy++
		for _, t := range r.MCPTools {
			if cy >= y+maxH {
				return
			}
			// Strip mcp__ prefix for readability
			short := t
			if idx := strings.Index(t, "__"); idx >= 0 {
				parts := strings.SplitN(t[idx+2:], "__", 2)
				if len(parts) == 2 {
					short = parts[0] + "/" + parts[1]
				} else {
					short = t[idx+2:]
				}
			}
			tcellapp.DrawText(s, x+2, cy, dimStyle, tcellapp.Truncate(short, w-3))
			cy++
		}
	}

	// Agent list
	if len(r.Agents) > 0 {
		cy++
		if cy >= y+maxH {
			return
		}
		tcellapp.DrawText(s, x, cy, labelStyle, fmt.Sprintf("Agents (%d)", r.AgentCount))
		cy++
		for _, a := range r.Agents {
			if cy >= y+maxH {
				return
			}
			tcellapp.DrawText(s, x+2, cy, dimStyle, tcellapp.Truncate(a, w-3))
			cy++
		}
	}

	// Skills list
	if len(r.Skills) > 0 {
		cy++
		if cy >= y+maxH {
			return
		}
		tcellapp.DrawText(s, x, cy, labelStyle, fmt.Sprintf("Skills (%d)", r.SkillCount))
		cy++
		for _, sk := range r.Skills {
			if cy >= y+maxH {
				return
			}
			tcellapp.DrawText(s, x+2, cy, dimStyle, tcellapp.Truncate(sk, w-3))
			cy++
		}
	}
}

func diagDrawCell(s tcell.Screen, x, y, width int, style tcell.Style, text string) int {
	tcellapp.DrawText(s, x, y, style, tcellapp.Truncate(text, width-1))
	return x + width
}

func diagFormatDuration(d time.Duration) string {
	secs := d.Seconds()
	if secs < 1 {
		return fmt.Sprintf("%dms", int(secs*1000))
	}
	return fmt.Sprintf("%.1fs", secs)
}
