package tcellapp

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
)

// ── Color palette ───────────────────────────────────────────────────────────

var (
	// Core brand — aquarium dark theme
	ColorPrimary    = tcell.NewRGBColor(0x6C, 0x8E, 0xC4) // #6C8EC4 muted blue (keywords)
	ColorPrimaryDim = tcell.NewRGBColor(0x50, 0x6A, 0x95) // #506A95 darker blue
	ColorAccent     = tcell.NewRGBColor(0x78, 0x9A, 0xB8) // #789AB8 desaturated blue

	// Semantic
	ColorSuccess = tcell.NewRGBColor(0x8A, 0xA8, 0x7A) // #8AA87A olive green
	ColorError   = tcell.NewRGBColor(0xC4, 0x7A, 0x8A) // #C47A8A muted rose
	ColorWarning = tcell.NewRGBColor(0xB8, 0x9E, 0x6C) // #B89E6C muted gold
	ColorRunning = tcell.NewRGBColor(0x6C, 0x8E, 0xC4) // #6C8EC4 blue
	ColorInfo    = tcell.NewRGBColor(0x78, 0x9A, 0xB8) // #789AB8 steel blue

	// Text
	ColorNormal = tcell.NewRGBColor(0xC5, 0xCD, 0xD9) // #C5CDD9 blueish gray fg
	ColorMuted  = tcell.NewRGBColor(0x4A, 0x55, 0x68) // #4A5568 comment gray
	ColorDimmer = tcell.NewRGBColor(0x38, 0x40, 0x4E) // #38404E dark comment

	// Surfaces — aquarium dark (very subtle blue tint)
	ColorSurface     = tcell.NewRGBColor(0x20, 0x22, 0x2B) // #20222B bg
	ColorSurfaceAlt  = tcell.NewRGBColor(0x24, 0x27, 0x30) // #242730 slightly lighter
	ColorSurfaceHL   = tcell.NewRGBColor(0x2A, 0x2E, 0x38) // #2A2E38 highlight
	ColorHeaderBg    = tcell.NewRGBColor(0x1C, 0x1E, 0x26) // #1C1E26 darkest
	ColorCardBg      = tcell.NewRGBColor(0x24, 0x27, 0x30) // #242730 card bg
	ColorCardBorder  = tcell.NewRGBColor(0x2F, 0x33, 0x3D) // #2F333D subtle border
	ColorCardFocused = tcell.NewRGBColor(0x2A, 0x38, 0x54) // #2A3854 focused (blue tint, more visible)

	// Message colors
	ColorAssistant  = ColorNormal
	ColorToolUse    = tcell.NewRGBColor(0x78, 0x9A, 0xB8) // #789AB8 steel blue
	ColorToolResult = tcell.NewRGBColor(0xB8, 0x9E, 0x6C) // #B89E6C gold
	ColorResult     = ColorSuccess
)

// ── Styles ──────────────────────────────────────────────────────────────────

func StyleTitle() tcell.Style   { return tcell.StyleDefault.Bold(true).Foreground(ColorPrimary) }
func StyleSubtitle() tcell.Style { return tcell.StyleDefault.Foreground(ColorAccent) }
func StyleSelected() tcell.Style { return tcell.StyleDefault.Bold(true).Foreground(ColorPrimary) }
func StyleNormal() tcell.Style  { return tcell.StyleDefault.Foreground(ColorNormal) }
func StyleDim() tcell.Style     { return tcell.StyleDefault.Foreground(ColorMuted) }
func StyleDimmer() tcell.Style  { return tcell.StyleDefault.Foreground(ColorDimmer) }
func StyleError() tcell.Style   { return tcell.StyleDefault.Foreground(ColorError) }
func StyleSuccess() tcell.Style { return tcell.StyleDefault.Foreground(ColorSuccess) }
func StyleWarning() tcell.Style { return tcell.StyleDefault.Foreground(ColorWarning) }
func StyleInfo() tcell.Style    { return tcell.StyleDefault.Foreground(ColorInfo) }
func StyleToken() tcell.Style   { return tcell.StyleDefault.Foreground(ColorInfo) }

func StyleWorkerRunning() tcell.Style { return tcell.StyleDefault.Foreground(ColorRunning) }
func StyleWorkerIdle() tcell.Style    { return tcell.StyleDefault.Foreground(ColorMuted) }
func StyleWorkerDone() tcell.Style    { return tcell.StyleDefault.Foreground(ColorSuccess) }
func StyleWorkerError() tcell.Style   { return tcell.StyleDefault.Foreground(ColorError) }

func StyleHeaderBg() tcell.Style { return tcell.StyleDefault.Background(ColorHeaderBg) }
func StyleCardBg() tcell.Style   { return tcell.StyleDefault.Background(ColorCardBg) }

// ── Text rendering ──────────────────────────────────────────────────────────

func DrawText(s tcell.Screen, x, y int, style tcell.Style, text string) int {
	for _, r := range text {
		s.SetContent(x, y, r, nil, style)
		x++
	}
	return x
}

func DrawTextf(s tcell.Screen, x, y int, style tcell.Style, format string, args ...any) int {
	return DrawText(s, x, y, style, fmt.Sprintf(format, args...))
}

func DrawCenteredText(s tcell.Screen, x, y, w int, style tcell.Style, text string) int {
	runes := []rune(text)
	offset := (w - len(runes)) / 2
	if offset < 0 {
		offset = 0
	}
	return DrawText(s, x+offset, y, style, text)
}

func DrawRightAlignedText(s tcell.Screen, x, y, w int, style tcell.Style, text string) int {
	runes := []rune(text)
	offset := w - len(runes)
	if offset < 0 {
		offset = 0
	}
	return DrawText(s, x+offset, y, style, text)
}

// ── Fill & shapes ───────────────────────────────────────────────────────────

func Fill(s tcell.Screen, x, y, w, h int, style tcell.Style) {
	for row := y; row < y+h; row++ {
		for col := x; col < x+w; col++ {
			s.SetContent(col, row, ' ', nil, style)
		}
	}
}

func FillInner(s tcell.Screen, x, y, w, h int, style tcell.Style) {
	for row := y + 1; row < y+h-1; row++ {
		for col := x + 1; col < x+w-1; col++ {
			s.SetContent(col, row, ' ', nil, style)
		}
	}
}

// DrawBox draws a bordered box using Unicode box-drawing characters.
func DrawBox(s tcell.Screen, x, y, w, h int, style tcell.Style) {
	if w < 2 || h < 2 {
		return
	}
	// Rounded corners
	s.SetContent(x, y, '╭', nil, style)
	s.SetContent(x+w-1, y, '╮', nil, style)
	s.SetContent(x, y+h-1, '╰', nil, style)
	s.SetContent(x+w-1, y+h-1, '╯', nil, style)

	for col := x + 1; col < x+w-1; col++ {
		s.SetContent(col, y, '─', nil, style)
		s.SetContent(col, y+h-1, '─', nil, style)
	}
	for row := y + 1; row < y+h-1; row++ {
		s.SetContent(x, row, '│', nil, style)
		s.SetContent(x+w-1, row, '│', nil, style)
	}
}

// DrawBoxWithTitle draws a bordered box with a title in the top border.
func DrawBoxWithTitle(s tcell.Screen, x, y, w, h int, style tcell.Style, title string, titleStyle tcell.Style) {
	DrawBox(s, x, y, w, h, style)
	if title != "" && w > 4 {
		tx := x + 2
		s.SetContent(tx-1, y, '┤', nil, style)
		tx = DrawText(s, tx, y, titleStyle, " "+title+" ")
		if tx < x+w-1 {
			s.SetContent(tx, y, '├', nil, style)
		}
	}
}

func DrawHLine(s tcell.Screen, x, y, w int, style tcell.Style) {
	for col := x; col < x+w; col++ {
		s.SetContent(col, y, '─', nil, style)
	}
	// Tee connectors for box integration
	s.SetContent(x, y, '├', nil, style)
	s.SetContent(x+w-1, y, '┤', nil, style)
}

// DrawHLinePlain draws a plain horizontal line without tee connectors.
func DrawHLinePlain(s tcell.Screen, x, y, w int, style tcell.Style) {
	for col := x; col < x+w; col++ {
		s.SetContent(col, y, '─', nil, style)
	}
}

// DrawVLine draws a vertical line.
func DrawVLine(s tcell.Screen, x, y, h int, style tcell.Style) {
	for row := y; row < y+h; row++ {
		s.SetContent(x, row, '│', nil, style)
	}
}

// ── Header / Footer bars ────────────────────────────────────────────────────

// DrawHeaderBar fills a row with the header background and returns the style.
func DrawHeaderBar(s tcell.Screen, y, w int) tcell.Style {
	style := tcell.StyleDefault.Background(ColorHeaderBg)
	for col := 0; col < w; col++ {
		s.SetContent(col, y, ' ', nil, style)
	}
	return style
}

// DrawFooterBar fills the bottom row with footer background and draws help text.
func DrawFooterBar(s tcell.Screen, y, w int, help string) {
	style := tcell.StyleDefault.Background(ColorHeaderBg).Foreground(ColorDimmer)
	for col := 0; col < w; col++ {
		s.SetContent(col, y, ' ', nil, style)
	}
	// Parse help string to highlight keys in brackets
	x := 2
	i := 0
	runes := []rune(help)
	for i < len(runes) {
		if runes[i] == '[' {
			// Find closing bracket
			j := i + 1
			for j < len(runes) && runes[j] != ']' {
				j++
			}
			if j < len(runes) {
				keyStr := string(runes[i : j+1])
				x = DrawText(s, x, y, style.Foreground(ColorAccent), keyStr)
				i = j + 1
				continue
			}
		}
		s.SetContent(x, y, runes[i], nil, style)
		x++
		i++
	}
}

// ── Badges & labels ─────────────────────────────────────────────────────────

// DrawBadge draws a colored badge with text, returns x after badge.
func DrawBadge(s tcell.Screen, x, y int, text string, fg, bg tcell.Color) int {
	style := tcell.StyleDefault.Foreground(fg).Background(bg)
	x = DrawText(s, x, y, style, " "+text+" ")
	return x
}

// DrawStatusPill draws a status indicator pill.
func DrawStatusPill(s tcell.Screen, x, y int, text string, color tcell.Color) int {
	dimBg := tcell.NewRGBColor(0x1A, 0x1C, 0x30)
	style := tcell.StyleDefault.Foreground(color).Background(dimBg)
	x = DrawText(s, x, y, style.Foreground(color), " ")
	x = DrawText(s, x, y, style, text+" ")
	return x
}

// ── Progress bar ────────────────────────────────────────────────────────────

// DrawProgressBar draws a horizontal progress bar.
func DrawProgressBar(s tcell.Screen, x, y, w int, pct float64, fg, bg tcell.Color) {
	filled := int(float64(w) * pct)
	if filled > w {
		filled = w
	}
	fgStyle := tcell.StyleDefault.Foreground(fg).Background(fg)
	bgStyle := tcell.StyleDefault.Foreground(bg).Background(bg)
	for col := 0; col < w; col++ {
		if col < filled {
			s.SetContent(x+col, y, '█', nil, fgStyle)
		} else {
			s.SetContent(x+col, y, '░', nil, bgStyle)
		}
	}
}

// ── String helpers ──────────────────────────────────────────────────────────

func Truncate(s string, maxWidth int) string {
	runes := []rune(s)
	if len(runes) <= maxWidth {
		return s
	}
	if maxWidth <= 3 {
		return string(runes[:maxWidth])
	}
	return string(runes[:maxWidth-3]) + "..."
}

func PadRight(s string, width int) string {
	runes := []rune(s)
	if len(runes) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(runes))
}

// FormatTokens formats a token count for display (e.g. 1.2M, 456K, 789).
func FormatTokens(n int) string {
	if n >= 1_000_000 {
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	}
	if n >= 10_000 {
		return fmt.Sprintf("%.0fK", float64(n)/1_000)
	}
	if n >= 1_000 {
		return fmt.Sprintf("%.1fK", float64(n)/1_000)
	}
	return fmt.Sprintf("%d", n)
}

// ── Input fields ────────────────────────────────────────────────────────────

func StyleInputField() tcell.Style {
	return tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.NewRGBColor(0x2A, 0x2D, 0x44))
}

func StyleInputFieldInactive() tcell.Style {
	return tcell.StyleDefault.Foreground(ColorNormal).Background(tcell.NewRGBColor(0x1A, 0x1C, 0x30))
}

func DrawInputField(s tcell.Screen, x, y, fieldW int, style tcell.Style, value string, showCursor bool) int {
	for col := x; col < x+fieldW; col++ {
		s.SetContent(col, y, ' ', nil, style)
	}
	runes := []rune(value)
	visible := runes
	if len(visible) > fieldW-1 {
		visible = visible[len(visible)-(fieldW-1):]
	}
	DrawText(s, x, y, style, string(visible))
	if showCursor {
		cursorX := x + len(visible)
		if cursorX < x+fieldW {
			s.SetContent(cursorX, y, '▎', nil, style.Foreground(ColorPrimary))
		}
	}
	return x + fieldW
}
