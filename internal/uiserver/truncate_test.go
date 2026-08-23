package uiserver

import (
	"bytes"
	"strings"
	"testing"
	"unicode/utf8"
)

// TestCountLinesSemantics proves countLines' locked rule (D-11, the
// checkpoint) for all five named cases: empty input, a lone newline, two
// newline-terminated lines, two lines with an unterminated final line, and
// content with no newline at all.
func TestCountLinesSemantics(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want int
	}{
		{"empty", "", 0},
		{"lone-newline", "\n", 1},
		{"two-terminated-lines", "a\nb\n", 2},
		{"two-lines-unterminated-final", "a\nb", 2},
		{"no-newline-at-all", "abc", 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := countLines([]byte(c.in)); got != c.want {
				t.Fatalf("countLines(%q) = %d, want %d", c.in, got, c.want)
			}
		})
	}
}

// TestTruncateSourceUnderBothCaps proves a source under both the line cap
// and the byte cap returns unchanged content, the truncated flag unset,
// and totals equal to returned counts.
func TestTruncateSourceUnderBothCaps(t *testing.T) {
	src := []byte(strings.Repeat("line\n", 100))

	got := truncateSource(src)
	if got.Truncated {
		t.Fatal("truncateSource: truncated = true, want false for a source under both caps")
	}
	if !bytes.Equal(got.Content, src) {
		t.Fatalf("truncateSource: content changed for a source under both caps: got %d bytes, want %d bytes", len(got.Content), len(src))
	}
	if got.TotalLines != 100 || got.ReturnedLines != 100 {
		t.Fatalf("truncateSource: total_lines=%d returned_lines=%d, want both 100", got.TotalLines, got.ReturnedLines)
	}
	if got.TotalBytes != len(src) || got.ReturnedBytes != len(src) {
		t.Fatalf("truncateSource: total_bytes=%d returned_bytes=%d, want both %d", got.TotalBytes, got.ReturnedBytes, len(src))
	}
}

// TestTruncateSourceExceedsLineCap proves a source exceeding sourceLineCap
// returns exactly sourceLineCap lines, the truncated flag set, a total
// equal to the real line count, and a returned count equal to
// sourceLineCap.
func TestTruncateSourceExceedsLineCap(t *testing.T) {
	const extraLines = 50
	realLines := sourceLineCap + extraLines
	src := []byte(strings.Repeat("line\n", realLines))

	got := truncateSource(src)
	if !got.Truncated {
		t.Fatal("truncateSource: truncated = false, want true for a source exceeding sourceLineCap")
	}
	if got.TotalLines != realLines {
		t.Fatalf("truncateSource: total_lines = %d, want %d (the real line count)", got.TotalLines, realLines)
	}
	if got.ReturnedLines != sourceLineCap {
		t.Fatalf("truncateSource: returned_lines = %d, want sourceLineCap (%d)", got.ReturnedLines, sourceLineCap)
	}
	if gotLines := countLines(got.Content); gotLines != sourceLineCap {
		t.Fatalf("truncateSource: content actually has %d lines, want exactly sourceLineCap (%d)", gotLines, sourceLineCap)
	}
}

// TestTruncateSourceExceedsByteCap proves a single-line source exceeding
// sourceByteCap returns at most sourceByteCap bytes, the truncated flag
// set, a total equal to the real byte length, and a returned byte count at
// most sourceByteCap.
func TestTruncateSourceExceedsByteCap(t *testing.T) {
	const extraBytes = 10
	src := []byte(strings.Repeat("a", sourceByteCap+extraBytes))

	got := truncateSource(src)
	if !got.Truncated {
		t.Fatal("truncateSource: truncated = false, want true for a source exceeding sourceByteCap")
	}
	if got.TotalBytes != len(src) {
		t.Fatalf("truncateSource: total_bytes = %d, want %d (the real byte length)", got.TotalBytes, len(src))
	}
	if got.ReturnedBytes > sourceByteCap {
		t.Fatalf("truncateSource: returned_bytes = %d, want at most sourceByteCap (%d)", got.ReturnedBytes, sourceByteCap)
	}
	if len(got.Content) > sourceByteCap {
		t.Fatalf("truncateSource: content is %d bytes, want at most sourceByteCap (%d)", len(got.Content), sourceByteCap)
	}
}

// TestTruncateSourceLineCapBoundary asserts the line-cap boundary is
// INCLUSIVE, at both edges: exactly sourceLineCap lines is not truncated,
// and one line more sets the truncated flag.
func TestTruncateSourceLineCapBoundary(t *testing.T) {
	t.Run("exactly-at-cap-not-truncated", func(t *testing.T) {
		src := []byte(strings.Repeat("line\n", sourceLineCap))

		got := truncateSource(src)
		if got.Truncated {
			t.Fatal("truncateSource: truncated = true at exactly sourceLineCap lines, want false (boundary is inclusive)")
		}
		if got.TotalLines != sourceLineCap || got.ReturnedLines != sourceLineCap {
			t.Fatalf("truncateSource: total_lines=%d returned_lines=%d, want both sourceLineCap (%d)", got.TotalLines, got.ReturnedLines, sourceLineCap)
		}
	})

	t.Run("one-line-over-truncated", func(t *testing.T) {
		src := []byte(strings.Repeat("line\n", sourceLineCap+1))

		got := truncateSource(src)
		if !got.Truncated {
			t.Fatal("truncateSource: truncated = false at sourceLineCap+1 lines, want true")
		}
		if got.ReturnedLines != sourceLineCap {
			t.Fatalf("truncateSource: returned_lines = %d, want sourceLineCap (%d)", got.ReturnedLines, sourceLineCap)
		}
	})
}

// TestTruncateSourceNeverSplitsARune builds its input from a repeated
// multi-byte rune so a byte-naive cut would land mid-rune if it were going
// to, mirroring internal/mcp/session_line_test.go's
// TestSanitizeClientFieldTruncatesOnRuneBoundary shape: a length assertion
// plus a UTF-8 validity assertion.
func TestTruncateSourceNeverSplitsARune(t *testing.T) {
	// "中" is 3 bytes wide; repeated enough times to exceed sourceByteCap
	// without landing on an exact multiple of 3 relative to the cap,
	// which is exactly the condition that exposes a byte-naive cut.
	runeCount := sourceByteCap/3 + 100
	src := []byte(strings.Repeat("中", runeCount))

	got := truncateSource(src)
	if len(got.Content) > sourceByteCap {
		t.Fatalf("truncateSource: content is %d bytes, want at most sourceByteCap (%d)", len(got.Content), sourceByteCap)
	}
	if !utf8.Valid(got.Content) {
		t.Fatalf("truncateSource: content is not valid UTF-8 — truncation split a rune: %q", got.Content)
	}
}

// TestTruncateSourceEmptyInput proves an empty source is not an error and
// not a truncation: zero-length content, the truncated flag unset, and
// all four counts zero.
func TestTruncateSourceEmptyInput(t *testing.T) {
	got := truncateSource(nil)
	if len(got.Content) != 0 {
		t.Fatalf("truncateSource(nil): content has length %d, want 0", len(got.Content))
	}
	if got.Truncated {
		t.Fatal("truncateSource(nil): truncated = true, want false")
	}
	if got.TotalLines != 0 || got.TotalBytes != 0 || got.ReturnedLines != 0 || got.ReturnedBytes != 0 {
		t.Fatalf("truncateSource(nil): counts = %+v, want all zero", got)
	}
}

// TestTruncateSourceTotalsAreExact asserts the line and byte totals equal
// the untruncated input's real counts, computed independently in this
// test, for both a truncated and an untruncated input.
func TestTruncateSourceTotalsAreExact(t *testing.T) {
	t.Run("untruncated", func(t *testing.T) {
		src := []byte("alpha\nbeta\ngamma\n")
		wantLines := strings.Count(string(src), "\n")
		wantBytes := len(src)

		got := truncateSource(src)
		if got.TotalLines != wantLines {
			t.Fatalf("truncateSource: total_lines = %d, want %d (independently counted)", got.TotalLines, wantLines)
		}
		if got.TotalBytes != wantBytes {
			t.Fatalf("truncateSource: total_bytes = %d, want %d (independently counted)", got.TotalBytes, wantBytes)
		}
	})

	t.Run("truncated", func(t *testing.T) {
		const extraLines = 25
		realLines := sourceLineCap + extraLines
		src := []byte(strings.Repeat("line\n", realLines))
		wantBytes := len(src)

		got := truncateSource(src)
		if !got.Truncated {
			t.Fatal("test fixture assumption broken: expected this input to be truncated")
		}
		if got.TotalLines != realLines {
			t.Fatalf("truncateSource: total_lines = %d, want %d (independently computed, unaffected by truncation)", got.TotalLines, realLines)
		}
		if got.TotalBytes != wantBytes {
			t.Fatalf("truncateSource: total_bytes = %d, want %d (independently computed, unaffected by truncation)", got.TotalBytes, wantBytes)
		}
	})
}

// TestTransportBackstopSitsAboveApplicationCap asserts the D-13 ordering
// directly, so a later edit cannot silently invert it: the transport
// backstop must sit strictly above the application byte cap, or correct
// truncation would surface to the user as a resource_exhausted error.
func TestTransportBackstopSitsAboveApplicationCap(t *testing.T) {
	if transportSendMaxBytes <= sourceByteCap {
		t.Fatalf("transportSendMaxBytes (%d) is not strictly greater than sourceByteCap (%d) — a correctly-truncated response could be rejected as resource_exhausted", transportSendMaxBytes, sourceByteCap)
	}
}
