package present

import (
	"fmt"
	"strings"
	"unicode"
)

// dropRune reports whether a display sanitizer removes r: a control rune
// (Unicode category Cc), such as ESC, the C0 and C1 ranges, newline, carriage
// return and DEL. A control rune can start a terminal escape sequence or
// move the cursor, so it is removed and never shown.
func dropRune(r rune) bool {
	return unicode.IsControl(r)
}

// escapeRune reports whether a display sanitizer shows r as a "<U+XXXX>"
// escape instead of printing it: a format rune (category Cf), a line
// separator (Zl) or a paragraph separator (Zp). These reorder or hide text,
// for example a bidi override or isolate, a zero-width space, a soft hyphen,
// a word joiner or a byte order mark. The escape keeps the rune visible, so
// a printed path cannot look like a different path.
//
// Three groups of Cf runes pass through unchanged because they cannot
// reorder text and real scripts need them: ZWNJ (U+200C), ZWJ (U+200D) and
// the tag characters (U+E0020 to U+E007F).
func escapeRune(r rune) bool {
	switch {
	case r == 0x200C, r == 0x200D:
		return false
	case r >= 0xE0020 && r <= 0xE007F:
		return false
	}
	return unicode.In(r, unicode.Cf, unicode.Zl, unicode.Zp)
}

// SanitizeDisplay returns s made safe to print to a terminal: invalid UTF-8
// bytes become U+FFFD, every dropRune rune is removed, and every escapeRune
// rune becomes a "<U+XXXX>" escape. The CLI display sanitizers that cover
// paths, names, prompts and messages call it through sanitizePathForDisplay.
// It removes a tab as well, because a tab inside a one-line value would
// shift the columns that follow it.
func SanitizeDisplay(s string) string {
	return sanitize(s, false)
}

// sanitizeControl is SanitizeDisplay for the text the styled renderers draw
// from the indexed repository: names, paths and verbatim source lines. It
// keeps the tab, because a bare tab cannot start an escape sequence and the
// indentation of tab-indented source depends on it.
//
// It guards styled output only. The plain renderers in internal/query are
// frozen for byte identity and do not call it.
func sanitizeControl(s string) string {
	return sanitize(s, true)
}

// sanitize is the shared body of SanitizeDisplay and sanitizeControl. The
// invalid-UTF-8 replacement runs first: a raw 0x9B byte passes for no rune
// in the fast-path check below, and the colour writer turns it into a real
// escape sequence.
func sanitize(s string, keepTab bool) string {
	s = strings.ToValidUTF8(s, "�")
	drops := func(r rune) bool { return dropRune(r) && (r != '\t' || !keepTab) }
	if !strings.ContainsFunc(s, func(r rune) bool { return drops(r) || escapeRune(r) }) {
		return s // fast path: nothing to change, no allocation
	}
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch {
		case drops(r):
		case escapeRune(r):
			fmt.Fprintf(&b, "<U+%04X>", r)
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}
