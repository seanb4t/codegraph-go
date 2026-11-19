package present

import (
	"strings"
	"unicode"
)

// UnsafeDisplayRune reports whether r must not reach a terminal: a control
// rune (Unicode category Cc) or a format rune (category Cf), such as a bidi
// override or isolate, a zero-width character or a byte order mark. Every
// display sanitizer in the cli packages decides through this predicate.
func UnsafeDisplayRune(r rune) bool {
	return unicode.IsControl(r) || unicode.Is(unicode.Cf, r)
}

// sanitizeControl strips terminal control characters from s so that
// filesystem-derived names (directory names and file paths from the indexed
// repository, which may be adversarial) and verbatim source-code lines
// cannot inject ANSI/OSC escape sequences into the styled TTY output
// (CR-01, and narrowed by CR-02/04-REVIEW.md). Every rune for which
// UnsafeDisplayRune reports true — ESC (0x1b), the C0/C1 control ranges,
// embedded newline/carriage-return that would otherwise break the line
// structure, and format runes that reorder or hide text — is dropped, EXCEPT the literal tab (0x09): a bare tab cannot
// start an OSC/CSI/DCS escape sequence and cannot overwrite prior output
// the way CR/backspace can, so it poses no injection risk, and stripping it
// was destroying the indentation of tab-indented source (e.g. gofmt'd Go)
// shown by the styled `explore`/`node` renderers.
//
// This guards the pretty (TTY) renderer only. The plain (piped/non-TTY)
// renderer in internal/query is intentionally NOT changed: it is frozen for
// byte-identity (TUI-02), and a non-TTY sink does not interpret escapes the
// way an interactive terminal does — the pretty path is the injection vector
// this closes.
func sanitizeControl(s string) string {
	isDangerous := func(r rune) bool { return UnsafeDisplayRune(r) && r != '\t' }
	if !strings.ContainsFunc(s, isDangerous) {
		return s // fast path: nothing to strip, no allocation
	}
	return strings.Map(func(r rune) rune {
		if isDangerous(r) {
			return -1 // drop
		}
		return r
	}, s)
}
