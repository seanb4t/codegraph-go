package present

import (
	"strings"
	"unicode"
)

// sanitizeControl strips terminal control characters from s so that
// filesystem-derived names (directory names and file paths from the indexed
// repository, which may be adversarial) cannot inject ANSI/OSC escape
// sequences into the styled TTY output (CR-01). Every rune for which
// unicode.IsControl reports true — ESC (0x1b), the C0/C1 control ranges, and
// embedded tab/newline/carriage-return that would otherwise break the line
// structure — is dropped.
//
// This guards the pretty (TTY) renderer only. The plain (piped/non-TTY)
// renderer in internal/query is intentionally NOT changed: it is frozen for
// byte-identity (TUI-02), and a non-TTY sink does not interpret escapes the
// way an interactive terminal does — the pretty path is the injection vector
// this closes.
func sanitizeControl(s string) string {
	if !strings.ContainsFunc(s, unicode.IsControl) {
		return s // fast path: nothing to strip, no allocation
	}
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1 // drop
		}
		return r
	}, s)
}
