package present

import (
	"bytes"
	"regexp"
	"strings"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/gitmeta"
	"github.com/seanb4t/codegraph-go/internal/query"
)

// ansiRE strips SGR escape sequences (\x1b[...m) so section-order and
// content assertions can run against the human-readable text without the
// pretty branch's ANSI bytes getting in the way.
var ansiRE = regexp.MustCompile("\x1b\\[[0-9;]*m")

func stripANSI(s string) string {
	return ansiRE.ReplaceAllString(s, "")
}

func fixtureStatusResult() query.StatusResult {
	return query.StatusResult{
		FileCount:   1234,
		NodeCount:   5678,
		EdgeCount:   9,
		DbSizeBytes: 2 * 1024 * 1024,
		Backend:     "pebble",
		NodesByKind: map[string]int64{
			"function": 10,
			"class":    2,
		},
		FilesByLanguage: map[string]int64{
			"go":     3,
			"python": 1,
		},
	}
}

// TestRenderStatus_ContainsANSI proves the pretty branch styled
// something — output must contain a raw ANSI escape byte sequence.
func TestRenderStatus_ContainsANSI(t *testing.T) {
	var buf bytes.Buffer
	if err := RenderStatus(fixtureStatusResult(), "/tmp/proj", &buf); err != nil {
		t.Fatalf("RenderStatus: %v", err)
	}
	if !bytes.Contains(buf.Bytes(), []byte("\x1b[")) {
		t.Errorf("expected an ANSI escape sequence in output, got:\n%s", buf.String())
	}
}

// TestRenderStatus_SectionOrder asserts, ANSI-stripped, the same section
// headers appear in the same order as query.RenderStatusText.
func TestRenderStatus_SectionOrder(t *testing.T) {
	var buf bytes.Buffer
	if err := RenderStatus(fixtureStatusResult(), "/tmp/proj", &buf); err != nil {
		t.Fatalf("RenderStatus: %v", err)
	}
	stripped := stripANSI(buf.String())

	sections := []string{"CodeGraph Status", "Index Statistics", "Nodes by Kind:", "Files by Language:"}
	last := -1
	for _, section := range sections {
		idx := strings.Index(stripped, section)
		if idx < 0 {
			t.Fatalf("missing section %q in ANSI-stripped output:\n%s", section, stripped)
		}
		if idx <= last {
			t.Fatalf("section %q out of order in ANSI-stripped output:\n%s", section, stripped)
		}
		last = idx
	}
}

// TestRenderStatus_NumericFormatting asserts numeric counts from the
// fixture appear, formatted identically to the plain renderer (thousands
// grouping preserved).
func TestRenderStatus_NumericFormatting(t *testing.T) {
	r := fixtureStatusResult()
	var buf bytes.Buffer
	if err := RenderStatus(r, "/tmp/proj", &buf); err != nil {
		t.Fatalf("RenderStatus: %v", err)
	}
	stripped := stripANSI(buf.String())

	for _, want := range []string{formatNumber(r.FileCount), formatNumber(r.NodeCount), formatNumber(r.EdgeCount)} {
		if !strings.Contains(stripped, want) {
			t.Errorf("expected %q in ANSI-stripped output:\n%s", want, stripped)
		}
	}
}

// TestRenderStatus_WorktreeWarning asserts a WorktreeMismatch warning
// appears — structural parity with the plain path.
func TestRenderStatus_WorktreeWarning(t *testing.T) {
	r := fixtureStatusResult()
	r.WorktreeMismatch = &gitmeta.Mismatch{WorktreeRoot: "/a/main", IndexRoot: "/a/main/.claude/worktrees/probe"}
	var buf bytes.Buffer
	if err := RenderStatus(r, "/tmp/proj", &buf); err != nil {
		t.Fatalf("RenderStatus: %v", err)
	}
	stripped := stripANSI(buf.String())

	warning := r.WorktreeMismatch.Warning()
	if warning == "" {
		t.Fatal("fixture WorktreeMismatch.Warning() unexpectedly empty — test setup is broken")
	}
	if !strings.Contains(stripped, warning) {
		t.Errorf("expected worktree warning %q in ANSI-stripped output:\n%s", warning, stripped)
	}
}

// TestRenderStatus_SanitizesControlChars proves projectPath and the two
// WorktreeMismatch path fields (both of which embed host filesystem paths)
// are passed through sanitizeControl before reaching the pretty sink
// (WR-01) — mirroring sanitize_test.go's ESC/control-byte fixtures. Note
// RenderStatus's own lipgloss styling legitimately emits many unrelated
// ESC sequences (headerStyle, labelStyle, sectionStyle), so this test
// checks for absence of the SPECIFIC injected escape sequences rather than
// absence of ESC bytes generally. The warning's own literal newlines (from
// Warning()'s message template, not attacker-controlled) must survive —
// only the ESC/BEL bytes injected via the path fields are stripped.
func TestRenderStatus_SanitizesControlChars(t *testing.T) {
	r := fixtureStatusResult()
	r.WorktreeMismatch = &gitmeta.Mismatch{
		WorktreeRoot: "/a/main\x1b[31m",
		IndexRoot:    "/a/main/.claude/worktrees/probe\x07",
	}
	dirtyPath := "/tmp/proj\x1b]0;pwned\x07"

	var buf bytes.Buffer
	if err := RenderStatus(r, dirtyPath, &buf); err != nil {
		t.Fatalf("RenderStatus: %v", err)
	}
	out := buf.String()

	for _, forbiddenSeq := range []string{"\x1b[31m", "\x1b]0;pwned\x07"} {
		if strings.Contains(out, forbiddenSeq) {
			t.Errorf("expected injected escape sequence %q to be stripped from output, got:\n%q", forbiddenSeq, out)
		}
	}
	if !strings.Contains(out, "/a/main") || !strings.Contains(out, "/a/main/.claude/worktrees/probe") {
		t.Errorf("expected sanitized worktree paths to survive in output, got:\n%q", out)
	}
	if !strings.Contains(out, "\n  Index from: /a/main/.claude/worktrees/probe\n") {
		t.Errorf("expected the warning's own literal newlines to survive sanitization, got:\n%q", out)
	}
	if !strings.Contains(out, "/tmp/proj") {
		t.Errorf("expected sanitized projectPath to survive in output, got:\n%q", out)
	}
}
