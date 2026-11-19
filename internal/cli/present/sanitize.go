package present

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// action is what a display sanitizer does with one rune.
type action int

const (
	keep action = iota
	drop
	escape
)

// escapeRune reports whether a display sanitizer shows r as a "<U+XXXX>"
// escape instead of printing it. prev is the rune before r in the text, or 0
// at the start.
//
// The escaped runes are the format runes (category Cf), the line and
// paragraph separators (Zl, Zp), the variation selectors and the other
// default-ignorable runes. A bidi override cannot reorder text and a
// zero-width space cannot hide inside a path, because each shows as an
// escape. The tag characters (U+E0020 to U+E007F) show as escapes too.
//
// Three exemptions pass through, because real text needs them:
//   - U+FE0E and U+FE0F, the text and emoji presentation selectors.
//   - ZWNJ (U+200C) and ZWJ (U+200D) after a non-ASCII rune, which keeps
//     emoji sequences and Persian text intact. After an ASCII rune or at the
//     start they are escaped, so "pro<ZWJ>ject" does not print like "project".
//
// The residual: two texts that differ only in a ZWNJ or ZWJ after a
// non-ASCII rune print the same.
func escapeRune(r, prev rune) bool {
	switch r {
	case 0xFE0E, 0xFE0F:
		return false
	case 0x200C, 0x200D:
		return prev < utf8.RuneSelf
	}
	return unicode.In(r, unicode.Cf, unicode.Zl, unicode.Zp,
		unicode.Variation_Selector, unicode.Other_Default_Ignorable_Code_Point)
}

// classify is the one decision both sanitizer paths make for r. A control
// rune (Unicode category Cc) is escaped when escapeControl is true. When it
// is false the control rune is dropped, except the tab, which stays.
func classify(r, prev rune, escapeControl bool) action {
	switch {
	case unicode.IsControl(r):
		switch {
		case escapeControl:
			return escape
		case r == '\t':
			return keep
		}
		return drop
	case escapeRune(r, prev):
		return escape
	}
	return keep
}

// SanitizeDisplay returns s made safe to print to a terminal: every run of
// invalid UTF-8 bytes becomes one U+FFFD, and every control rune and every
// escapeRune rune becomes a "<U+XXXX>" escape, so two different values never
// print the same, with the residual that escapeRune states. A tab is a
// control rune and shows as an escape, because a tab inside a one-line value
// shifts the columns that follow it. Literal "<U+XXXX>" text in s prints the
// same as an escape.
//
// The CLI uses it, through sanitizePathForDisplay, for paths, names, error
// text and prompts that it prints in styled and plain output alike.
func SanitizeDisplay(s string) string {
	return sanitize(s, true)
}

// sanitizeControl is the sanitizer for text that the styled renderers draw:
// names, paths, query text, messages and verbatim source lines. It does what
// SanitizeDisplay does, except that it drops control runes instead of
// escaping them, and it keeps the tab. A bare tab cannot start an escape
// sequence, and the indentation of tab-indented source depends on it. A CR
// left at the end of a CRLF line is dropped.
//
// It guards styled output only. The plain renderers in internal/query are
// frozen for byte identity and do not call it.
func sanitizeControl(s string) string {
	return sanitize(s, false)
}

// sanitize is the shared body of SanitizeDisplay and sanitizeControl. The
// invalid-UTF-8 replacement runs first. The rune scans below decode a raw
// 0x9B byte as U+FFFD, so without the replacement they would keep it as the
// byte, and the colorprofile writer turns a C1 CSI byte into an escape
// sequence when it downsamples. Both scans call classify, so the fast path
// returns s only when the slow path would change nothing.
func sanitize(s string, escapeControl bool) string {
	s = strings.ToValidUTF8(s, string(utf8.RuneError))
	clean := true
	prev := rune(0)
	for _, r := range s {
		if classify(r, prev, escapeControl) != keep {
			clean = false
			break
		}
		prev = r
	}
	if clean {
		return s // no allocation
	}
	var b strings.Builder
	b.Grow(len(s))
	prev = 0
	for _, r := range s {
		switch classify(r, prev, escapeControl) {
		case escape:
			fmt.Fprintf(&b, "<U+%04X>", r)
		case keep:
			b.WriteRune(r)
		}
		prev = r
	}
	return b.String()
}
