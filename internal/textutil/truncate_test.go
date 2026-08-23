package textutil

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// TestTruncateOnRuneBoundary proves TruncateOnRuneBoundary's rune-safety
// guarantee for the package that now owns the implementation (moved,
// unchanged, from internal/mcp/session_line.go's unexported
// truncateOnRuneBoundary): built from a repeated multi-byte rune so a
// byte-naive cut would land mid-rune if it were going to, mirroring
// internal/mcp/session_line_test.go's
// TestSanitizeClientFieldTruncatesOnRuneBoundary shape exactly — a length
// assertion plus a UTF-8 validity assertion with a failure message
// explaining why it matters.
func TestTruncateOnRuneBoundary(t *testing.T) {
	const maxBytes = 50

	// A rune that is 3 bytes wide (e.g. U+4E2D "中"), repeated so the
	// truncation boundary lands mid-rune if truncation were byte-naive.
	long := strings.Repeat("中", 100) // 300 bytes

	got := TruncateOnRuneBoundary(long, maxBytes)
	if len(got) > maxBytes {
		t.Fatalf("TruncateOnRuneBoundary result is %d bytes, want <= %d", len(got), maxBytes)
	}
	if !utf8.ValidString(got) {
		t.Fatalf("TruncateOnRuneBoundary truncated mid-rune, producing invalid UTF-8: %q", got)
	}

	// A string already within the limit is returned unchanged.
	short := "hello"
	if got := TruncateOnRuneBoundary(short, maxBytes); got != short {
		t.Fatalf("TruncateOnRuneBoundary(%q, %d) = %q, want unchanged input", short, maxBytes, got)
	}
}
