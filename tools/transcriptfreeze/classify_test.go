package main

import (
	"errors"
	"os"
	"strings"
	"testing"
)

// TestClassifyFlagsTranscriptPlusServerChange proves the core D-03 shape:
// a transcript path plus an internal/mcp/*.go path together yield a
// violation naming both paths, and the Reason discloses the trigger set as
// a floor (docs/MCP-2026-07-28-SCOPING.md's own wording), never presenting
// a clean pass as a proof of safety.
func TestClassifyFlagsTranscriptPlusServerChange(t *testing.T) {
	changed := []string{
		"testdata/wireoracle/transcripts/handshake-explore.golden",
		"internal/mcp/server.go",
	}
	v := Classify(changed, "")

	if !v.Violation {
		t.Fatalf("Classify(%v, \"\"): Violation = false, want true", changed)
	}
	if len(v.Transcripts) != 1 || v.Transcripts[0] != "testdata/wireoracle/transcripts/handshake-explore.golden" {
		t.Errorf("Verdict.Transcripts = %v, want the transcript path named", v.Transcripts)
	}
	if len(v.ServerChanges) != 1 || v.ServerChanges[0] != "internal/mcp/server.go" {
		t.Errorf("Verdict.ServerChanges = %v, want internal/mcp/server.go named", v.ServerChanges)
	}
	if !strings.Contains(v.Reason, "testdata/wireoracle/transcripts/handshake-explore.golden") {
		t.Errorf("Verdict.Reason does not name the offending transcript path: %q", v.Reason)
	}
	if !strings.Contains(v.Reason, "internal/mcp/server.go") {
		t.Errorf("Verdict.Reason does not name the offending server path: %q", v.Reason)
	}
	if !strings.Contains(v.Reason, "floor") {
		t.Errorf("Verdict.Reason does not state the trigger set is a floor: %q", v.Reason)
	}
	if !strings.Contains(v.Reason, "internal/query") || !strings.Contains(v.Reason, "internal/indexer") {
		t.Errorf("Verdict.Reason does not name internal/query and internal/indexer as recorded residual risk: %q", v.Reason)
	}
}

// TestClassify_TranscriptOnlyIsNotAViolation proves regenerating a
// transcript in its own pull request (the sanctioned Phase 3 path) is not
// blocked.
func TestClassify_TranscriptOnlyIsNotAViolation(t *testing.T) {
	changed := []string{"testdata/wireoracle/transcripts/handshake-explore.golden"}
	v := Classify(changed, "")
	if v.Violation {
		t.Fatalf("Classify(%v, \"\"): Violation = true, want false (transcript-only change)", changed)
	}
}

// TestClassify_ServerOnlyIsNotAViolation proves a change confined to
// internal/mcp with no transcript touched is not blocked.
func TestClassify_ServerOnlyIsNotAViolation(t *testing.T) {
	changed := []string{"internal/mcp/server.go"}
	v := Classify(changed, "")
	if v.Violation {
		t.Fatalf("Classify(%v, \"\"): Violation = true, want false (internal/mcp-only change)", changed)
	}
}

// TestClassify_TranscriptPlusGoModMCPBumpIsError proves the trigger set's
// dependency-line side: a transcript path plus an added/removed go.mod line
// naming an MCP SDK module prefix is a violation, while a go.mod diff
// touching an unrelated dependency (or only a context line mentioning the
// SDK) is not.
func TestClassify_TranscriptPlusGoModMCPBumpIsError(t *testing.T) {
	transcript := "testdata/wireoracle/transcripts/handshake-explore.golden"

	t.Run("added MCP SDK line is a violation", func(t *testing.T) {
		diff := "--- a/go.mod\n+++ b/go.mod\n+\tgithub.com/mark3labs/mcp-go v0.57.0\n"
		v := Classify([]string{transcript}, diff)
		if !v.Violation {
			t.Fatalf("Violation = false, want true for an added MCP SDK go.mod line")
		}
		if !v.DependencyChanged {
			t.Errorf("DependencyChanged = false, want true")
		}
	})

	t.Run("unrelated dependency bump is not a violation", func(t *testing.T) {
		diff := "--- a/go.mod\n+++ b/go.mod\n+\tgithub.com/spf13/cobra v1.11.0\n"
		v := Classify([]string{transcript}, diff)
		if v.Violation {
			t.Fatalf("Violation = true, want false for an unrelated go.mod dependency bump")
		}
	})

	t.Run("context line mentioning the dependency does not trigger it", func(t *testing.T) {
		diff := " \tgithub.com/mark3labs/mcp-go v0.56.0\n" // no leading +/-: a context line
		if MCPDependencyTouched(diff) {
			t.Fatalf("MCPDependencyTouched(%q) = true, want false for a context-only line", diff)
		}
	})
}

// TestParseChangedListDistinguishesUnreadableFromEmpty proves the three
// outcomes ParseChangedList and main.go's file-read step together must
// never conflate: a clean empty diff, a malformed-but-successfully-read
// record, and a file that could not be read at all (ErrNoInput, main.go's
// own sentinel — see its doc comment).
func TestParseChangedListDistinguishesUnreadableFromEmpty(t *testing.T) {
	t.Run("empty input is clean, not an error", func(t *testing.T) {
		got, err := ParseChangedList("")
		if err != nil {
			t.Fatalf("ParseChangedList(\"\"): err = %v, want nil", err)
		}
		if len(got) != 0 {
			t.Fatalf("ParseChangedList(\"\"): got %v, want an empty slice", got)
		}
	})

	t.Run("whitespace/blank-lines-only input is clean, not an error", func(t *testing.T) {
		got, err := ParseChangedList("\n\n  \n")
		if err != nil {
			t.Fatalf("ParseChangedList(\"\\n\\n  \\n\"): err = %v, want nil", err)
		}
		if len(got) != 0 {
			t.Fatalf("ParseChangedList(\"\\n\\n  \\n\"): got %v, want an empty slice", got)
		}
	})

	t.Run("malformed non-empty input is a parse error naming the line, not ErrNoInput", func(t *testing.T) {
		_, err := ParseChangedList("garbage-with-no-tab")
		if err == nil {
			t.Fatalf("ParseChangedList(\"garbage-with-no-tab\"): err = nil, want a non-nil error")
		}
		if !strings.Contains(err.Error(), "garbage-with-no-tab") {
			t.Fatalf("ParseChangedList error does not name the offending line: %v", err)
		}
		if errors.Is(err, ErrNoInput) {
			t.Fatalf("a parse error over successfully-read malformed content must not be ErrNoInput — ErrNoInput is main.go's own file-unreadable sentinel, a categorically different failure")
		}
	})

	t.Run("ErrNoInput is reserved for a file that could not be read at all", func(t *testing.T) {
		missing := "/nonexistent/does-not-exist-transcriptfreeze-test"
		if _, statErr := os.Stat(missing); statErr == nil {
			t.Fatalf("test precondition violated: %q unexpectedly exists", missing)
		}
		_, readErr := os.ReadFile(missing)
		if readErr == nil {
			t.Fatalf("os.ReadFile(%q): err = nil, want a non-nil error", missing)
		}
		// main.go wraps exactly this shape of failure with ErrNoInput; prove
		// the sentinel itself is distinguishable via errors.Is once wrapped,
		// so the "unreadable" signal main.go produces is never confused with
		// ParseChangedList's own clean-empty or malformed-parse results.
		wrapped := errorsJoin(ErrNoInput, readErr)
		if !errors.Is(wrapped, ErrNoInput) {
			t.Fatalf("a wrapped file-read failure must remain distinguishable as ErrNoInput via errors.Is")
		}
	})
}

// errorsJoin mirrors the minimal wrapping shape main.go uses when a
// changed-list or go.mod-diff file cannot be read at all — kept local to
// the test so it exercises exactly the errors.Is contract ErrNoInput
// promises, without depending on main.go's own control flow.
func errorsJoin(sentinel, cause error) error {
	return &wrappedError{sentinel: sentinel, cause: cause}
}

type wrappedError struct {
	sentinel error
	cause    error
}

func (e *wrappedError) Error() string {
	return e.sentinel.Error() + ": " + e.cause.Error()
}

func (e *wrappedError) Unwrap() error {
	return e.sentinel
}

// TestClassifyDetectsRenamedTranscript proves a renamed transcript (or a
// renamed transcripts directory) is still detected: git diff --name-status
// -M -C reports a rename as one record carrying BOTH paths, and
// ParseChangedList must contribute both to the transcript set.
// --name-only would report only one side depending on diff filtering,
// which is why the guard's Taskfile body uses --name-status.
func TestClassifyDetectsRenamedTranscript(t *testing.T) {
	renameLine := "R100\ttestdata/wireoracle/transcripts/a.golden\ttestdata/wireoracle/transcripts/b.golden"
	changed, err := ParseChangedList(renameLine)
	if err != nil {
		t.Fatalf("ParseChangedList(%q): err = %v, want nil", renameLine, err)
	}

	v := Classify(changed, "")
	if len(v.Transcripts) != 2 {
		t.Fatalf("Verdict.Transcripts = %v, want both sides of the rename record", v.Transcripts)
	}
	want := map[string]bool{
		"testdata/wireoracle/transcripts/a.golden": true,
		"testdata/wireoracle/transcripts/b.golden": true,
	}
	for _, p := range v.Transcripts {
		if !want[p] {
			t.Errorf("Verdict.Transcripts contains unexpected path %q", p)
		}
		delete(want, p)
	}
	if len(want) != 0 {
		t.Errorf("Verdict.Transcripts missing path(s): %v", want)
	}
}

// TestTranscriptFreezeHelpersFailLoudly is this file's table-of-{name, fn}
// repudiation guard, following the convention established in
// internal/upgrade/taskfile_shape_test.go and
// internal/upgrade/release_workflow_shape_test.go's
// TestWorkflowSourceHelpersFailLoudly: every parser helper must return a
// non-nil error — never a usable zero value — on malformed input.
func TestTranscriptFreezeHelpersFailLoudly(t *testing.T) {
	cases := []struct {
		name string
		fn   func() error
	}{
		{
			name: "ParseChangedList: line with no tab",
			fn: func() error {
				_, err := ParseChangedList("no-tab-here")
				return err
			},
		},
		{
			name: "ParseChangedList: unrecognized status letter",
			fn: func() error {
				_, err := ParseChangedList("Z\tfoo.go")
				return err
			},
		},
		{
			name: "ParseChangedList: rename record missing its second path",
			fn: func() error {
				_, err := ParseChangedList("R100\tonly-one-path.go")
				return err
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if err := c.fn(); err == nil {
				t.Fatalf("%s: expected a non-nil error, got nil", c.name)
			}
		})
	}
}

// TestGuardPatternsMatchRealTree proves the guard's path predicates are not
// stale: at least one real file under testdata/wireoracle/transcripts/
// satisfies the transcript predicate, and the real internal/mcp/server.go
// satisfies the server-change predicate. If the transcripts directory is
// ever moved or renamed, this test fails loudly instead of leaving the
// guard silently unable to match anything.
func TestGuardPatternsMatchRealTree(t *testing.T) {
	const transcriptsDir = "../../testdata/wireoracle/transcripts"
	entries, err := os.ReadDir(transcriptsDir)
	if err != nil {
		t.Fatalf("read %s: %v", transcriptsDir, err)
	}

	found := false
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		relPath := transcriptDirPrefix + e.Name()
		if isTranscriptPath(relPath) {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("no real file under %s satisfies isTranscriptPath — the guard's transcriptDirPrefix (%q) may be stale relative to the real tree", transcriptsDir, transcriptDirPrefix)
	}

	const serverFile = "../../internal/mcp/server.go"
	if _, statErr := os.Stat(serverFile); statErr != nil {
		t.Fatalf("stat %s: %v", serverFile, statErr)
	}
	if !isServerChangePath(serverDirPrefix + "server.go") {
		t.Fatalf("internal/mcp/server.go does not satisfy isServerChangePath — the guard's serverDirPrefix (%q) may be stale", serverDirPrefix)
	}
}
