package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeTemp writes content to dir/name and returns its path, failing the
// test loudly if the write does not succeed.
func writeTemp(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	return path
}

// TestRunReportsCollisionButDoesNotFail proves 03-02's option-advisory
// decision end to end through run()'s CLI entrypoint: a changed-file list
// naming both a transcript path and an internal/mcp Go path, with an empty
// go.mod diff, produces exit 0 — advisory, not blocking — AND a report on
// stderr naming both offending sides. The report is the deliverable now
// that the exit code no longer is.
func TestRunReportsCollisionButDoesNotFail(t *testing.T) {
	dir := t.TempDir()
	changedList := writeTemp(t, dir, "changed",
		"M\ttestdata/wireoracle/transcripts/handshake-explore.golden\nM\tinternal/mcp/server.go\n")
	gomodDiff := writeTemp(t, dir, "gomod", "")

	var stderr bytes.Buffer
	got := runCLI([]string{"-changed-list", changedList, "-gomod-diff", gomodDiff}, &stderr)

	if got != 0 {
		t.Fatalf("runCLI(...) = %d, want 0 (advisory: a detected collision must not fail the build)", got)
	}
	report := stderr.String()
	if report == "" {
		t.Fatal("runCLI(...) wrote nothing to stderr, want a non-empty collision report")
	}
	if !strings.Contains(report, "testdata/wireoracle/transcripts/handshake-explore.golden") {
		t.Errorf("report does not name the offending transcript path: %q", report)
	}
	if !strings.Contains(report, "internal/mcp/server.go") {
		t.Errorf("report does not name the offending internal/mcp path: %q", report)
	}
}

// TestRunTranscriptOnlyIsClean proves a transcript-only change (the
// sanctioned Phase 3 regeneration path) produces exit 0 and no collision
// report — advisory did not become "always report."
func TestRunTranscriptOnlyIsClean(t *testing.T) {
	dir := t.TempDir()
	changedList := writeTemp(t, dir, "changed",
		"M\ttestdata/wireoracle/transcripts/handshake-explore.golden\n")
	gomodDiff := writeTemp(t, dir, "gomod", "")

	var stderr bytes.Buffer
	got := runCLI([]string{"-changed-list", changedList, "-gomod-diff", gomodDiff}, &stderr)

	if got != 0 {
		t.Fatalf("runCLI(...) = %d, want 0", got)
	}
	if strings.Contains(stderr.String(), "D-03") {
		t.Errorf("stderr unexpectedly reports a D-03 collision for a transcript-only change: %q", stderr.String())
	}
}

// TestRunUnreadableInputIsExitTwo pins that the unusable-input code (2) is
// not collapsed into the now-advisory (0) collision code by this change.
func TestRunUnreadableInputIsExitTwo(t *testing.T) {
	dir := t.TempDir()
	gomodDiff := writeTemp(t, dir, "gomod", "")
	missing := filepath.Join(dir, "does-not-exist")

	var stderr bytes.Buffer
	got := runCLI([]string{"-changed-list", missing, "-gomod-diff", gomodDiff}, &stderr)

	if got != 2 {
		t.Fatalf("runCLI(...) = %d, want 2 (unreadable -changed-list)", got)
	}
	if !strings.Contains(stderr.String(), missing) {
		t.Errorf("stderr does not name the unreadable path: %q", stderr.String())
	}
}

// TestRunMalformedRecordIsExitTwo proves a malformed-but-readable
// changed-file record still exits 2, distinct from both the clean and the
// now-non-blocking collision outcomes.
func TestRunMalformedRecordIsExitTwo(t *testing.T) {
	dir := t.TempDir()
	changedList := writeTemp(t, dir, "changed", "garbage-with-no-tab\n")
	gomodDiff := writeTemp(t, dir, "gomod", "")

	var stderr bytes.Buffer
	got := runCLI([]string{"-changed-list", changedList, "-gomod-diff", gomodDiff}, &stderr)

	if got != 2 {
		t.Fatalf("runCLI(...) = %d, want 2 (malformed changed-file record)", got)
	}
	if !strings.Contains(stderr.String(), "garbage-with-no-tab") {
		t.Errorf("stderr does not name the offending line: %q", stderr.String())
	}
}
