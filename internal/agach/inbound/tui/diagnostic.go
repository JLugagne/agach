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
	app        *tuiApp
	results    []domain.DiagnosticResult
	done       bool
	cancel     context.CancelFunc
	cursor     int
	detailScroll int // vertical scroll offset in detail panel
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
				m.detailScroll = 0
			}
		case "down", "j":
			if m.cursor < len(m.results)-1 {
				m.cursor++
				m.detailScroll = 0
			}
		case "pgup":
			if m.detailScroll > 0 {
				m.detailScroll -= 10
				if m.detailScroll < 0 {
					m.detailScroll = 0
				}
			}
		case "pgdn":
			m.detailScroll += 10
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

	// Layout: left table | right detail panel (fixed width)
	detailW := 50
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
		tcellapp.DrawFooterBar(s, h-1, w, "[j/k] navigate  [pgup/pgdn] scroll  [esc/q] back")
	} else {
		tcellapp.DrawFooterBar(s, h-1, w, "[j/k] navigate  [pgup/pgdn] scroll  [esc] cancel  ·  running probes...")
	}
}

func (m *DiagnosticModel) drawDetail(s tcell.Screen, x, y, w, maxH int, r domain.DiagnosticResult) {
	surfBg := tcell.StyleDefault.Background(tcellapp.ColorSurface)
	headerStyle := surfBg.Bold(true).Foreground(tcellapp.ColorAccent)
	labelStyle := surfBg.Foreground(tcellapp.ColorMuted)
	valStyle := surfBg.Foreground(tcellapp.ColorNormal)

	cy := y

	// Title
	name := r.AgentSlug
	if name == "" {
		name = "(baseline)"
	}
	tcellapp.DrawText(s, x, cy, headerStyle, name)
	cy++

	if r.Status != domain.DiagnosticDone {
		tcellapp.DrawText(s, x, cy, labelStyle, string(r.Status))
		return
	}

	if r.ContextRaw == "" {
		tcellapp.DrawText(s, x, cy, labelStyle, "fetching context...")
		return
	}

	// Parse markdown tables from /context output into display lines
	lines := diagParseContext(r.ContextRaw)

	// Clamp scroll
	visibleLines := maxH
	maxScroll := len(lines) - visibleLines
	if maxScroll < 0 {
		maxScroll = 0
	}
	if m.detailScroll > maxScroll {
		m.detailScroll = maxScroll
	}

	start := m.detailScroll
	for i := start; i < len(lines) && cy < y+maxH; i++ {
		dl := lines[i]
		switch dl.kind {
		case diagLineHeader:
			tcellapp.DrawText(s, x, cy, headerStyle, tcellapp.Truncate(dl.text, w-1))
		case diagLineLabel:
			tcellapp.DrawText(s, x, cy, labelStyle, tcellapp.Truncate(dl.text, w-1))
		case diagLineKV:
			tcellapp.DrawText(s, x, cy, labelStyle, dl.text)
			if dl.value != "" {
				vx := x + w - len([]rune(dl.value)) - 1
				if vx > x+len([]rune(dl.text)) {
					tcellapp.DrawText(s, vx, cy, valStyle, dl.value)
				}
			}
		case diagLineEmpty:
			// blank line
		}
		cy++
	}
}

type diagLineKind int

const (
	diagLineEmpty  diagLineKind = iota
	diagLineHeader              // section header (bold accent)
	diagLineLabel               // bold key text (e.g. "Model:")
	diagLineKV                  // left-aligned label, right-aligned value
)

type diagLine struct {
	kind  diagLineKind
	text  string
	value string // right-aligned (for diagLineKV)
}

// diagParseContext converts the markdown /context output into display lines.
func diagParseContext(raw string) []diagLine {
	var out []diagLine
	lines := strings.Split(raw, "\n")

	var currentSection string // track which section we're in
	var mcpTotal int          // accumulate MCP tool token total

	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])

		// Skip empty
		if line == "" {
			// Emit MCP total before the blank line after MCP Tools section
			if currentSection == "mcp" && mcpTotal > 0 {
				out = append(out, diagLine{kind: diagLineKV, text: "  Total", value: tcellapp.FormatTokens(mcpTotal)})
				mcpTotal = 0
				currentSection = ""
			}
			out = append(out, diagLine{kind: diagLineEmpty})
			continue
		}

		// Section headers: ### or ##
		if strings.HasPrefix(line, "##") {
			// Emit MCP total if switching away
			if currentSection == "mcp" && mcpTotal > 0 {
				out = append(out, diagLine{kind: diagLineKV, text: "  Total", value: tcellapp.FormatTokens(mcpTotal)})
				mcpTotal = 0
			}
			title := strings.TrimLeft(line, "# ")
			if strings.Contains(strings.ToLower(title), "mcp tool") {
				currentSection = "mcp"
			} else {
				currentSection = ""
			}
			out = append(out, diagLine{kind: diagLineHeader, text: title})
			continue
		}

		// Bold key-value: **Key:** value
		if strings.HasPrefix(line, "**") {
			line = strings.ReplaceAll(line, "**", "")
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				out = append(out, diagLine{kind: diagLineKV, text: strings.TrimSpace(parts[0]), value: strings.TrimSpace(parts[1])})
			} else {
				out = append(out, diagLine{kind: diagLineLabel, text: line})
			}
			continue
		}

		// Table rows: | col1 | col2 | col3 |
		if strings.HasPrefix(line, "|") {
			// Skip separator rows (|---|---|)
			if strings.Contains(line, "---") {
				continue
			}
			cells := diagParseTableRow(line)
			if len(cells) == 0 {
				continue
			}
			// Skip header rows (detected by matching known headers)
			lower0 := strings.ToLower(cells[0])
			if lower0 == "category" || lower0 == "tool" || lower0 == "agent type" || lower0 == "type" || lower0 == "skill" {
				continue
			}
			label := "  " + cells[0]
			value := ""
			if len(cells) >= 3 {
				last := cells[len(cells)-1]
				if strings.Contains(last, "%") {
					// Category table: Category | Tokens | Percentage → use tokens (col 1)
					value = cells[1]
				} else {
					// MCP/agent/memory tables: Name | Server/Type | Tokens → use last
					value = last
				}
			} else if len(cells) == 2 {
				value = cells[1]
			}

			// Accumulate MCP total
			if currentSection == "mcp" {
				if n := diagParseTokenCount(value); n > 0 {
					mcpTotal += n
				}
			}

			out = append(out, diagLine{kind: diagLineKV, text: label, value: value})
			continue
		}
	}

	// Emit trailing MCP total if file ends without blank line
	if currentSection == "mcp" && mcpTotal > 0 {
		out = append(out, diagLine{kind: diagLineKV, text: "  Total", value: tcellapp.FormatTokens(mcpTotal)})
	}

	return out
}

// diagParseTokenCount parses a token count string like "107", "4.1k", "19.4k"
func diagParseTokenCount(s string) int {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, ",", "")
	s = strings.ReplaceAll(s, "%", "")

	multiplier := 1.0
	if strings.HasSuffix(s, "k") {
		multiplier = 1000
		s = s[:len(s)-1]
	}

	var f float64
	if _, err := fmt.Sscanf(s, "%f", &f); err != nil {
		return 0
	}
	return int(f * multiplier)
}

func diagParseTableRow(line string) []string {
	parts := strings.Split(line, "|")
	var cells []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			cells = append(cells, p)
		}
	}
	return cells
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
