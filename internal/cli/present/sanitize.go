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

// runeState is what the sanitizer remembers of the text before a rune: prev
// is the last printed rune and base is the last printed rune that is not a
// combining mark, joiner or presentation selector. Both are 0 at the start.
// After an escape both are '>', the last rune of the escape. A dropped rune
// changes neither.
type runeState struct {
	prev, base rune
}

// advance updates st after the sanitizer acted on r.
func (st *runeState) advance(r rune, a action) {
	switch a {
	case keep:
		st.prev = r
		if !isMark(r) && !isJoiner(r) && !isPresentationSelector(r) {
			st.base = r
		}
	case escape:
		st.prev, st.base = '>', '>'
	}
}

// isJoiner reports whether r is ZWNJ (U+200C) or ZWJ (U+200D).
func isJoiner(r rune) bool { return r == 0x200C || r == 0x200D }

// isPresentationSelector reports whether r is U+FE0E or U+FE0F.
func isPresentationSelector(r rune) bool { return r == 0xFE0E || r == 0xFE0F }

// isMark reports whether r is a nonspacing or enclosing combining mark. The
// presentation selectors are marks in Unicode but not here.
func isMark(r rune) bool {
	return !isPresentationSelector(r) && unicode.In(r, unicode.Mn, unicode.Me)
}

// afterBase reports whether a joiner or selector may follow the text before
// it. The last printed rune must be non-ASCII and not a joiner. When it is a
// combining mark, the rune that the mark sits on must be non-ASCII too.
func (st runeState) afterBase() bool {
	if st.prev < utf8.RuneSelf || isJoiner(st.prev) {
		return false
	}
	return !isMark(st.prev) || st.base >= utf8.RuneSelf
}

// escapeRune reports whether a display sanitizer shows r as a "<U+XXXX>"
// escape instead of printing it. st describes the text before r and next is
// the rune after r, or 0 at the end.
//
// The escaped runes are the format runes (category Cf), including the tag
// characters, the line and paragraph separators (Zl, Zp), the variation
// selectors and the other default-ignorable runes. A bidi override cannot
// reorder text and a zero-width space cannot hide inside a path, because each
// shows as an escape.
//
// Two exemptions pass through, because emoji sequences, Indic conjuncts and
// Persian text need them:
//   - ZWNJ and ZWJ pass when st.afterBase holds. A presentation selector
//     counts as a base, so the ZWJ in "eye, FE0F, ZWJ, speech bubble" passes.
//     So does a ZWJ after a virama, which is a combining mark on a non-ASCII
//     consonant.
//   - U+FE0E and U+FE0F pass when st.afterBase holds and the last printed
//     rune is not a joiner or a selector. U+FE0F also passes after an ASCII
//     digit, "#" or "*" when next is U+20E3, the keycap sequence.
//
// Everything else in those two groups escapes, so a run of joiners and
// selectors stays at most as long as a real sequence needs, and
// "main<FE0F>.go" or "pro<ZWJ>ject" never prints like the plain name.
func escapeRune(r rune, st runeState, next rune) bool {
	switch {
	case isJoiner(r):
		return !st.afterBase()
	case isPresentationSelector(r):
		if r == 0xFE0F && next == 0x20E3 &&
			(st.prev >= '0' && st.prev <= '9' || st.prev == '#' || st.prev == '*') {
			return false
		}
		return !st.afterBase() || isPresentationSelector(st.prev)
	}
	return unicode.In(r, unicode.Cf, unicode.Zl, unicode.Zp,
		unicode.Variation_Selector, unicode.Other_Default_Ignorable_Code_Point)
}

// classify is the one decision both sanitizer paths make for r. A control
// rune (Unicode category Cc) is escaped when escapeControl is true. When it
// is false the control rune is dropped, except the tab, which stays.
func classify(r rune, st runeState, next rune, escapeControl bool) action {
	switch {
	case unicode.IsControl(r):
		switch {
		case escapeControl:
			return escape
		case r == '\t':
			return keep
		}
		return drop
	case escapeRune(r, st, next):
		return escape
	}
	return keep
}

// nextRune returns the rune of s that starts at byte offset i, or 0 at the
// end of s.
func nextRune(s string, i int) rune {
	if i >= len(s) {
		return 0
	}
	r, _ := utf8.DecodeRuneInString(s[i:])
	return r
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
// a combining mark and the precomposed letter, runes that are unassigned or
// in a private-use area, and the joiner and selector positions that
// escapeRune allows. sanitizeControl adds one more: a dropped control rune,
// so "ma<U+0001>in.go" and "main.go" print the same.
//
// The CLI uses it, directly or through sanitizePathForDisplay, for paths,
// names, error text and prompts that it prints in styled and plain output
// alike.
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
// sequence when it downsamples. Both scans call classify with the same state
// and the same next rune, so the fast path returns s only when the slow path
// would change nothing.
func sanitize(s string, escapeControl bool) string {
	s = strings.ToValidUTF8(s, string(utf8.RuneError))
	clean := true
	var st runeState
	for i, r := range s {
		a := classify(r, st, nextRune(s, i+utf8.RuneLen(r)), escapeControl)
		if a != keep {
			clean = false
			break
		}
		st.advance(r, a)
	}
	if clean {
		return s // no allocation
	}
	var b strings.Builder
	b.Grow(len(s))
	st = runeState{}
	for i, r := range s {
		a := classify(r, st, nextRune(s, i+utf8.RuneLen(r)), escapeControl)
		switch a {
		case keep:
			b.WriteRune(r)
		case escape:
			fmt.Fprintf(&b, "<U+%04X>", r)
		}
		st.advance(r, a)
	}
	return b.String()
}
