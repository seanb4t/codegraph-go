package present

import "regexp"

// sgrSequence matches an SGR (Select Graphic Rendition) ANSI escape
// sequence — the ONE shared ANSI stripper every renderer contract test in
// this package reuses (plans 04/05/06 run in parallel and must not each
// define their own copy). A local regexp, never charmbracelet/x/ansi, so
// go.mod stays untouched (D-15).
var sgrSequence = regexp.MustCompile("\x1b\\[[0-9;]*m")

// stripANSI removes every SGR escape sequence from s, leaving the
// human-readable text every content/section-order assertion in this
// package's renderer tests checks against.
func stripANSI(s string) string {
	return sgrSequence.ReplaceAllString(s, "")
}
