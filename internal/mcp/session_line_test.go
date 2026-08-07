package mcp

import (
	"fmt"
	"io"
	"strings"
	"testing"
	"unicode/utf8"
)

// TestSessionLineIsAlwaysOnByConstruction is VRFY-03's construction
// guarantee: NewStdioServer must refuse a nil session-log writer rather
// than silently disabling the always-on session line. "Always on" is a
// construction property, not a calling convention — the milestone's only
// mitigation for a spec-sanctioned silent version mismatch must not degrade
// to "on because today's single caller happens to pass cmd.ErrOrStderr()"
// (addresses review concern: "the session line is always-on only by
// convention at one call site").
//
// Renamed from TestNewStdioServerRejectsNilSessionLog, added under
// deviation Rule 2 by the plan-01 tracer before this plan's artifact table
// named it — same behavior, the name this plan's artifact table requires.
func TestSessionLineIsAlwaysOnByConstruction(t *testing.T) {
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
	if len(got) > clientFieldMaxBytes {
		t.Fatalf("sanitizeClientField result is %d bytes, want <= %d", len(got), clientFieldMaxBytes)
	}
	if !utf8.ValidString(got) {
		t.Fatalf("sanitizeClientField truncated mid-rune, producing invalid UTF-8: %q", got)
	}
}

// parseSessionLineFields is a test-only parser proving D-14's parseability
// claim: given a formatted session line, it verifies the fixed shape (one
// trailing newline, the two-token prefix, exactly four space-delimited
// key=value fields) and returns the parsed key->value map. It fails loudly
// — a non-nil error, never a partial or zero-value map — on any malformed
// input (T-03-01's repudiation concern: a test asserting on this parser's
// output must not be able to pass against a line that silently broke the
// four-field shape).
func parseSessionLineFields(line string) (map[string]string, error) {
	if strings.Count(line, "\n") != 1 || !strings.HasSuffix(line, "\n") {
		return nil, fmt.Errorf("parseSessionLineFields: expected exactly one trailing newline, got %q", line)
	}
	trimmed := strings.TrimSuffix(line, "\n")
	if strings.Contains(trimmed, "\n") {
		return nil, fmt.Errorf("parseSessionLineFields: embedded newline before the trailing one: %q", line)
	}

	prefixWithSpace := sessionLinePrefix + " "
	if !strings.HasPrefix(trimmed, prefixWithSpace) {
		return nil, fmt.Errorf("parseSessionLineFields: line does not start with prefix %q: %q", sessionLinePrefix, trimmed)
	}
	rest := strings.TrimPrefix(trimmed, prefixWithSpace)

	tokens := strings.Split(rest, " ")
	if len(tokens) != 4 {
		return nil, fmt.Errorf("parseSessionLineFields: got %d fields after the prefix, want 4: %q", len(tokens), tokens)
	}

	fields := make(map[string]string, 4)
	for _, tok := range tokens {
		key, value, ok := strings.Cut(tok, "=")
		if !ok || key == "" {
			return nil, fmt.Errorf("parseSessionLineFields: token %q is not a key=value pair", tok)
		}
		fields[key] = value
	}
	return fields, nil
}

// TestParseSessionLineFieldsFailLoudly is the fail-loudly companion this
// file introduces (mirroring TestSanitizeClientFieldFailLoudly and
// internal/upgrade/release_workflow_shape_test.go's
// TestWorkflowSourceHelpersFailLoudly table convention): every malformed
// shape below must produce a non-nil error, never a partial map.
func TestParseSessionLineFieldsFailLoudly(t *testing.T) {
	valid := sessionLinePrefix + " requested=a negotiated=a client=a/a tools=1"
	cases := []struct {
		name string
		in   string
	}{
		{name: "no trailing newline", in: valid},
		{name: "two trailing newlines", in: valid + "\n\n"},
		{name: "embedded newline before the end", in: sessionLinePrefix + " requested=a\nnegotiated=a client=a/a tools=1\n"},
		{name: "missing prefix", in: "not-the-prefix requested=a negotiated=a client=a/a tools=1\n"},
		{name: "too few fields", in: sessionLinePrefix + " requested=a negotiated=a tools=1\n"},
		{name: "too many fields", in: sessionLinePrefix + " requested=a negotiated=a client=a/a tools=1 extra=1\n"},
		{name: "field with no equals sign", in: sessionLinePrefix + " requested negotiated=a client=a/a tools=1\n"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fields, err := parseSessionLineFields(c.in)
			if err == nil {
				t.Fatalf("parseSessionLineFields(%q) = %v, nil error, want a non-nil error", c.in, fields)
			}
		})
	}
}

// TestSessionLineFormat pins D-14's fixed format and key order
// (requested, negotiated, client, tools) with a positional assertion, not
// a set-membership check — a reordering of the keys must fail this test
// even though every key is still present.
func TestSessionLineFormat(t *testing.T) {
	got := formatSessionLine("2025-11-25", "2025-11-25", "claude-code", "1.2.3", 1)
	want := "codegraph: mcp-session requested=2025-11-25 negotiated=2025-11-25 client=claude-code/1.2.3 tools=1\n"
	if got != want {
		t.Fatalf("formatSessionLine(...) = %q, want %q", got, want)
	}

	t.Run("equal requested and negotiated versions are not collapsed into one key", func(t *testing.T) {
		line := formatSessionLine("2025-06-18", "2025-06-18", "c", "v", 0)
		fields, err := parseSessionLineFields(line)
		if err != nil {
			t.Fatalf("parseSessionLineFields(%q): %v", line, err)
		}
		if fields["requested"] != "2025-06-18" {
			t.Fatalf("requested field = %q, want %q", fields["requested"], "2025-06-18")
		}
		if fields["negotiated"] != "2025-06-18" {
			t.Fatalf("negotiated field = %q, want %q", fields["negotiated"], "2025-06-18")
		}
	})

	t.Run("tools=0 is emitted, the key is never dropped", func(t *testing.T) {
		line := formatSessionLine("a", "a", "c", "v", 0)
		fields, err := parseSessionLineFields(line)
		if err != nil {
			t.Fatalf("parseSessionLineFields(%q): %v", line, err)
		}
		got, ok := fields["tools"]
		if !ok {
			t.Fatalf("tools key missing from %q", line)
		}
		if got != "0" {
			t.Fatalf("tools field = %q, want %q", got, "0")
		}
	})
}

// TestSessionLineSanitizesHostileClientInfo is T-03-01's mitigation proof:
// nine hostile clientInfo shapes, each of which must produce a line that
// still contains exactly one newline, still starts with the fixed prefix,
// and still parses into exactly four key=value fields — D-14's
// parseability guarantee must survive a hostile or malformed clientInfo.
// The prefix-injection case (a client name embedding the session-line
// prefix itself) is the specific shape a hostile client would use to try
// to synthesize a fabricated second diagnostic line in the same stream.
func TestSessionLineSanitizesHostileClientInfo(t *testing.T) {
	longASCII := strings.Repeat("A", 1024)
	longMultiByte := strings.Repeat("中", 100) // 300 bytes, each rune 3 bytes wide

	cases := []struct {
		name          string
		clientName    string
		clientVersion string
	}{
		{name: "embedded line feed", clientName: "evil\nname", clientVersion: "1.0.0"},
		{name: "embedded carriage return", clientName: "evil\rname", clientVersion: "1.0.0"},
		{name: "embedded NUL", clientName: "evil\x00name", clientVersion: "1.0.0"},
		{name: "embedded space", clientName: "evil name", clientVersion: "1.0.0"},
		{name: "embeds the session-line prefix itself", clientName: sessionLinePrefix + " requested=fake negotiated=fake client=x/x tools=99", clientVersion: "1.0.0"},
		{name: "1024 bytes of ASCII", clientName: longASCII, clientVersion: "1.0.0"},
		{name: "300 bytes of multi-byte runes", clientName: longMultiByte, clientVersion: "1.0.0"},
		{name: "invalid UTF-8 byte sequence", clientName: "bad\xffname", clientVersion: "1.0.0"},
		{name: "empty client name and empty client version", clientName: "", clientVersion: ""},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			line := formatSessionLine("2025-11-25", "2025-11-25", c.clientName, c.clientVersion, 1)

			if n := strings.Count(line, "\n"); n != 1 {
				t.Fatalf("formatSessionLine output has %d newlines, want exactly 1: %q", n, line)
			}
			if !strings.HasPrefix(line, sessionLinePrefix+" ") {
				t.Fatalf("formatSessionLine output does not start with the prefix %q: %q", sessionLinePrefix, line)
			}

			fields, err := parseSessionLineFields(line)
			if err != nil {
				t.Fatalf("parseSessionLineFields(%q): %v (a hostile clientInfo must never break the four-field shape)", line, err)
			}

			clientField := fields["client"]

			switch c.name {
			case "300 bytes of multi-byte runes":
				name, _, ok := strings.Cut(clientField, "/")
				if !ok {
					t.Fatalf("client field %q missing the '/' separator", clientField)
				}
				if !utf8.ValidString(name) {
					t.Fatalf("truncated client name %q is not valid UTF-8 — truncation split a rune", name)
				}
			case "empty client name and empty client version":
				name, version, ok := strings.Cut(clientField, "/")
				if !ok {
					t.Fatalf("client field %q missing the '/' separator", clientField)
				}
				if name != "<unknown>" || version != "<unknown>" {
					t.Fatalf("client field for empty name/version = %q, want \"<unknown>/<unknown>\"", clientField)
				}
			}
		})
	}
}
