package tui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"

	"github.com/JLugagne/agach-mcp/internal/agach/domain"
	"github.com/JLugagne/agach-mcp/internal/agach/inbound/tui/tcellapp"
)

const maxMessages = 500

type messagesPanel struct {
	workerID     int
	messages     []domain.LiveMessage
	width        int
	height       int
	autoScroll   bool
	scrollOffset int
}

func newMessagesPanel(workerID, width, height int) messagesPanel {
	return messagesPanel{
		workerID:   workerID,
		width:      width,
		height:     height,
		autoScroll: true,
	}
}

func (m *messagesPanel) addMessage(msg domain.LiveMessage) {
	m.messages = append(m.messages, msg)
	if len(m.messages) > maxMessages {
		m.messages = m.messages[len(m.messages)-maxMessages:]
	}
	if m.autoScroll {
		lines := m.renderLines()
		visibleH := m.height - 1 // reserve 1 row for header
		if len(lines) > visibleH {
			m.scrollOffset = len(lines) - visibleH
		} else {
			m.scrollOffset = 0
		}
	}
}

func (m messagesPanel) Update(msg tcellapp.Msg) (messagesPanel, tcellapp.Cmd) {
	km, ok := msg.(tcellapp.KeyMsg)
	if !ok {
		return m, nil
	}
	switch tcellapp.KeyString(km) {
	case "s":
		m.autoScroll = !m.autoScroll
		if m.autoScroll {
			lines := m.renderLines()
			visibleH := m.height - 1
			if len(lines) > visibleH {
				m.scrollOffset = len(lines) - visibleH
			} else {
				m.scrollOffset = 0
			}
		}
		return m, nil
	case "up", "k":
		if m.scrollOffset > 0 {
			m.scrollOffset--
		}
		return m, nil
	case "down", "j":
		lines := m.renderLines()
		visibleH := m.height - 1
		maxOff := len(lines) - visibleH
		if maxOff < 0 {
			maxOff = 0
		}
		if m.scrollOffset < maxOff {
			m.scrollOffset++
		}
		return m, nil
	}
	return m, nil
}

type renderedLine struct {
	timestamp string
	prefix    string
	content   string
	style     tcell.Style
}

func (m messagesPanel) renderLines() []renderedLine {
	maxContentWidth := m.width - 14 // timestamp(8) + space + prefix(~4) + space
	if maxContentWidth < 10 {
		maxContentWidth = 10
	}

	lines := make([]renderedLine, 0, len(m.messages))
	for _, msg := range m.messages {
		ts := msg.At.Format("15:04:05")

		var prefix string
		var style tcell.Style

		switch msg.Kind {
		case domain.MessageKindAssistant:
			prefix = " ▶ "
			style = tcell.StyleDefault.Foreground(tcellapp.ColorAssistant)
		case domain.MessageKindToolUse:
			prefix = " ⚙ tool: "
			style = tcell.StyleDefault.Foreground(tcellapp.ColorToolUse)
		case domain.MessageKindToolResult:
			prefix = " ← "
			style = tcell.StyleDefault.Foreground(tcellapp.ColorToolResult)
		case domain.MessageKindSystem:
			prefix = " · "
			style = tcellapp.StyleDim()
		case domain.MessageKindResult:
			prefix = " ✓ "
			style = tcell.StyleDefault.Bold(true).Foreground(tcellapp.ColorResult)
		default:
			prefix = " ? "
			style = tcellapp.StyleDim()
		}

		content := msg.Content
		if len(content) > maxContentWidth {
			content = content[:maxContentWidth-3] + "..."
		}

		lines = append(lines, renderedLine{
			timestamp: ts,
			prefix:    prefix,
			content:   content,
			style:     style,
		})
	}
	return lines
}

func (m messagesPanel) Draw(s tcell.Screen, x, y, w, h int) {
	bg := tcell.StyleDefault.Background(tcellapp.ColorSurface)

	// Fill background
	for row := y; row < y+h; row++ {
		for col := x; col < x+w; col++ {
			s.SetContent(col, row, ' ', nil, bg)
		}
	}

	// Header bar
	headerBg := tcellapp.DrawHeaderBar(s, y, w)
	autoStr := "on"
	autoColor := tcellapp.ColorSuccess
	if !m.autoScroll {
		autoStr = "off"
		autoColor = tcellapp.ColorDimmer
	}
	hx := tcellapp.DrawText(s, x+2, y, headerBg.Bold(true).Foreground(tcellapp.ColorAccent),
		fmt.Sprintf("MESSAGES  Worker %d", m.workerID))
	hx = tcellapp.DrawText(s, hx+3, y, headerBg.Foreground(tcellapp.ColorDimmer), "auto-scroll ")
	tcellapp.DrawText(s, hx, y, headerBg.Foreground(autoColor), autoStr)

	// Separator
	for col := x; col < x+w; col++ {
		s.SetContent(col, y+1, '─', nil, bg.Foreground(tcellapp.ColorDimmer))
	}

	// Message lines
	lines := m.renderLines()
	visibleH := h - 2
	for row := 0; row < visibleH; row++ {
		lineY := y + 2 + row
		idx := m.scrollOffset + row
		if idx < len(lines) {
			rl := lines[idx]
			cx := tcellapp.DrawText(s, x+1, lineY, bg.Foreground(tcellapp.ColorDimmer), rl.timestamp)
			cx = tcellapp.DrawText(s, cx, lineY, rl.style, rl.prefix+rl.content)
			for col := cx; col < x+w; col++ {
				s.SetContent(col, lineY, ' ', nil, bg)
			}
		}
	}
}
