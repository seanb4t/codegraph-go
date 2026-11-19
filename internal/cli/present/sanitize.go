package present

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// action is what a display sanitizer does with one rune. The zero value is
// not a valid action.
type action int

const (
	keep action = iota + 1
	drop
	escape
)

// isJoiner reports whether r is ZWNJ (U+200C) or ZWJ (U+200D).
func isJoiner(r rune) bool { return r == 0x200C || r == 0x200D }

// isPresentationSelector reports whether r is U+FE0E or U+FE0F.
func isPresentationSelector(r rune) bool { return r == 0xFE0E || r == 0xFE0F }

// escapeRune reports whether a display sanitizer shows r as a "<U+XXXX>"
// escape instead of printing it. prev is the rune printed just before r, or 0
// at the start. After an escape, prev is '>', the last rune of the escape.
// After a dropped rune, prev is unchanged.
//
// The escaped runes are the format runes (category Cf), including the tag
// characters, the line and paragraph separators (Zl, Zp), the variation
// selectors and the other default-ignorable runes. A bidi override cannot
// reorder text and a zero-width space cannot hide inside a path, because each
// shows as an escape.
//
// Two exemptions pass through, because emoji sequences and Persian text need
// them:
//   - ZWNJ and ZWJ pass when prev is a printed non-ASCII rune that is not
//     itself a joiner. A presentation selector counts, so the ZWJ in
//     "eye, FE0F, ZWJ, speech bubble" passes.
//   - U+FE0E and U+FE0F pass when prev is a printed non-ASCII rune that is
//     neither a joiner nor a selector. U+FE0F also passes after an ASCII digit,
//     "#" or "*", the start of a keycap sequence.
//
// Everything else in those two groups escapes, so a run of joiners and
// selectors stays at most as long as a real emoji sequence needs, and
// "main<FE0F>.go" or "pro<ZWJ>ject" never prints like the plain name. The
// residual: two texts that differ only in a joiner or selector at an
// allowed position print the same.
func escapeRune(r, prev rune) bool {
	afterBase := prev >= utf8.RuneSelf && !isJoiner(prev)
	switch {
	case isJoiner(r):
		return !afterBase
	case isPresentationSelector(r):
		if r == 0xFE0F && (prev >= '0' && prev <= '9' || prev == '#' || prev == '*') {
			return false
		}
		return !afterBase || isPresentationSelector(prev)
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
// escapeRune rune becomes a "<U+XXXX>" escape. It escapes every rune that can
// move the cursor, reorder text or hide inside a value. A tab is a control
// rune and shows as an escape, because a tab inside a one-line value shifts
// the columns that follow it.
//
// Two different values can still print the same in these cases: literal
// "<U+XXXX>" text and an escape, a run of invalid bytes and a literal U+FFFD,
// and the joiner and selector positions that escapeRune allows.
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
// sequence when it downsamples. Both scans call classify with the same prev
// rule, so the fast path returns s only when the slow path would change
// nothing.
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
		case keep:
			b.WriteRune(r)
			prev = r
		case escape:
			fmt.Fprintf(&b, "<U+%04X>", r)
			prev = '>'
		}
	}
	return b.String()
}
