package tui

import "strings"

// sanitiseTitle strips bytes that could terminate or escape an OSC-0 terminal
// title sequence: BEL (0x07), ESC (0x1b), and C1 ST (0x9c).
func sanitiseTitle(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r == 0x07 || r == 0x1b || r == 0x9c {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// sanitiseANSI removes ANSI/VT escape sequences from s so that server-
// controlled strings cannot manipulate the terminal state when displayed.
func sanitiseANSI(s string) string {
	var b strings.Builder
	inEsc := false
	for _, r := range s {
		switch {
		case r == '\x1b':
			inEsc = true
		case inEsc && ((r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || r == 'm'):
			inEsc = false
		case inEsc:
			// still consuming escape sequence
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// sanitiseCell strips control characters (except tab) from a table cell value
// returned by the server, preventing ANSI injection through diagnostic output.
func sanitiseCell(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r < 0x20 && r != '\t' {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// isShellSafeSessionID reports whether s consists only of alphanumerics,
// hyphens, and underscores — the characters present in normal UUID-format
// session identifiers produced by the server.  Any other character could act
// as a shell meta-character and enable command injection when the value is
// embedded in a string passed to sh -c.
func isShellSafeSessionID(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '-' || r == '_' {
			continue
		}
		return false
	}
	return true
}

const (
	maxProjectNameRunes = 200
	maxProjectDescRunes = 1000
)

// appendCappedRune appends ch to current only when the rune count of current
// is below max, enforcing an upper bound on field lengths.
func appendCappedRune(current, ch string, max int) string {
	if len([]rune(current)) >= max {
		return current
	}
	return current + ch
}
