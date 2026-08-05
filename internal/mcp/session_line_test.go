package mcp

import (
	"io"
	"strings"
	"testing"
	"unicode/utf8"
)

// TestNewStdioServerRejectsNilSessionLog is VRFY-03's construction
// guarantee: NewStdioServer must refuse a nil session-log writer rather
// than silently disabling the always-on session line. Not in this plan's
// files_modified list, but required by its own acceptance criteria — added
// per deviation Rule 2 (auto-add missing critical functionality).
func TestNewStdioServerRejectsNilSessionLog(t *testing.T) {
	dir := t.TempDir()

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("NewStdioServer(..., nil) did not panic")
		}
		msg, ok := r.(string)
		if !ok || !strings.Contains(msg, "io.Discard") {
			t.Fatalf("panic value = %v, want a string naming io.Discard as the opt-out", r)
		}
	}()

	NewStdioServer(false, map[string]bool{}, dir, dir, nil)
}

// TestNewStdioServerAcceptsDiscard confirms io.Discard is the sanctioned,
// explicit opt-out named in the panic message above.
func TestNewStdioServerAcceptsDiscard(t *testing.T) {
	dir := t.TempDir()

	s := NewStdioServer(false, map[string]bool{}, dir, dir, io.Discard)
	if s == nil {
		t.Fatal("NewStdioServer(..., io.Discard) returned nil")
	}
}

func TestSanitizeClientFieldFailLoudly(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{name: "empty becomes unknown", in: "", want: "<unknown>"},
		{name: "control chars and spaces become underscores", in: "claude code\n1.2.3\r\t", want: "claude_code_1.2.3__"},
		{name: "plain value passes through", in: "claude-code/1.2.3", want: "claude-code/1.2.3"},
		{name: "invalid utf-8 is replaced", in: "bad\xffbytes", want: "bad�bytes"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := sanitizeClientField(c.in)
			if got != c.want {
				t.Fatalf("sanitizeClientField(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

func TestSanitizeClientFieldTruncatesOnRuneBoundary(t *testing.T) {
	// A rune that is 3 bytes wide (e.g. U+4E2D "中"), repeated so the
	// truncation boundary lands mid-rune if truncation were byte-naive.
	long := strings.Repeat("中", 100) // 300 bytes
	got := sanitizeClientField(long)
	if len(got) > 256 {
		t.Fatalf("sanitizeClientField result is %d bytes, want <= 256", len(got))
	}
	if !utf8.ValidString(got) {
		t.Fatalf("sanitizeClientField truncated mid-rune, producing invalid UTF-8: %q", got)
	}
}
