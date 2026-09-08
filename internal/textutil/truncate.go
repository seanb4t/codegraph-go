// Package textutil holds small, security-relevant string/byte helpers
// shared across packages that must never disagree on how they cut text.
// The package exists so a rune-boundary cut has exactly ONE implementation
// in the tree (T-01-37): before this package existed,
// internal/mcp/session_line.go's truncateOnRuneBoundary was unexported and
// therefore not importable, and a second, hand-copied implementation in
// internal/uiserver would have been a second security-relevant copy of a
// cut that must never split a rune.
package textutil

import "unicode/utf8"

// TruncateOnRuneBoundary returns the prefix of s that is at most maxBytes
// long, walking backward from maxBytes to the nearest rune boundary so a
// multi-byte UTF-8 sequence is never split. Moved here, unchanged, from
// internal/mcp/session_line.go (originally unexported as
// truncateOnRuneBoundary) so internal/mcp and internal/uiserver share one
// implementation rather than each carrying their own copy of a cut that is
// security-relevant precisely because it must never split a rune.
//
// The guarantee this function makes is SCOPED, not universal: if s was
// valid UTF-8, the returned prefix is valid UTF-8 too, because the cut
// never lands mid-rune. If s was not valid UTF-8 to begin with, the
// returned prefix is neither repaired nor rejected — it is simply cut at
// the nearest byte that looks like a rune-start byte (or an ASCII byte),
// which is the best a byte-oriented cut can promise over already-invalid
// input.
func TruncateOnRuneBoundary(s string, maxBytes int) string {
	if len(s) <= maxBytes {
		return s
	}
	for maxBytes > 0 && !utf8.RuneStart(s[maxBytes]) {
		maxBytes--
	}
	return s[:maxBytes]
}
