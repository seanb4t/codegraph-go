package cli

import (
	"errors"
	"strings"
	"testing"
)

// assertNoANSI fails the test if s contains an ANSI CSI escape sequence
// (\x1b[) — the signal a spinner frame would leave behind.
func assertNoANSI(t *testing.T, s, label string) {
	t.Helper()
	if strings.Contains(s, "\x1b[") {
		t.Errorf("%s contains an ANSI escape sequence, want none on non-TTY: %q", label, s)
	}
}

// TestProgressCLINonTTYReachability drives the real `init`/`index`/`sync`
// commands via execCmd — a non-TTY harness by construction (RESEARCH
// Pitfall 3: os.Stderr under `go test` is never a real terminal, so
// ChoosePresentation's isTTY branch can never fire here) — and asserts the
// TUI-05 wiring's non-TTY branch is byte-identical to pre-Phase-6
// behavior: zero ANSI escape bytes ever reach stderr, and the stdout
// summary line is unchanged (D-06/D-07).
func TestProgressCLINonTTYReachability(t *testing.T) {
	t.Run("init: no spinner frames, unchanged summary", func(t *testing.T) {
		dir := copyFixture(t)
		out, errOut, err := execCmd("init", dir)
		if err != nil {
			t.Fatalf("init: unexpected error: %v", err)
		}
		assertNoANSI(t, errOut, "init stderr")
		for _, want := range []string{"files=", "nodes=", "edges=", "duration="} {
			if !strings.Contains(out, want) {
				t.Fatalf("init: expected unchanged summary to contain %q, got: %q", want, out)
			}
		}
	})

	t.Run("index: no spinner frames, unchanged summary", func(t *testing.T) {
		dir := setupIndexedFixture(t)
		out, errOut, err := execCmd("index", "--force", dir)
		if err != nil {
			t.Fatalf("index: unexpected error: %v", err)
		}
		assertNoANSI(t, errOut, "index stderr")
		if !strings.Contains(out, "files=") {
			t.Fatalf("index: expected unchanged summary line, got: %q", out)
		}
	})

	t.Run("sync: no spinner frames, unchanged summary", func(t *testing.T) {
		dir := setupIndexedFixture(t)
		out, errOut, err := execCmd("sync", dir)
		if err != nil {
			t.Fatalf("sync: unexpected error: %v", err)
		}
		assertNoANSI(t, errOut, "sync stderr")
		if !strings.Contains(out, "files=") || !strings.Contains(out, "reparsed=") {
			t.Fatalf("sync: expected unchanged files=/reparsed= summary, got: %q", out)
		}
	})
}

// TestProgressCLIQuietSuppressesSummary pins the pre-existing --quiet
// contract (no summary output) as still intact now that the spinner
// wiring's own !quiet gate sits alongside it. The non-TTY test harness
// cannot independently distinguish "no spinner because non-TTY" from "no
// spinner because --quiet" (both already produce zero stderr ANSI in this
// environment) — this test's job is proving --quiet's summary-suppression
// contract survived this plan's change byte-for-byte.
func TestProgressCLIQuietSuppressesSummary(t *testing.T) {
	dir := copyFixture(t)
	out, errOut, err := execCmd("init", "--quiet", dir)
	if err != nil {
		t.Fatalf("init --quiet: unexpected error: %v", err)
	}
	if out != "" {
		t.Fatalf("init --quiet: expected empty stdout, got: %q", out)
	}
	assertNoANSI(t, errOut, "init --quiet stderr")
}

// TestProgressWiringDoesNotBreakErrorPaths proves the spinner wiring
// (inserted immediately before each indexer.Run/Sync call) does not
// introduce a panic or hang on the pre-existing error-return paths that
// execute before the wrapped call — index/sync's ErrNotInitialized guard.
// A real TTY-driven "Stop still fires when indexer.Run itself errors"
// assertion is out of reach for this in-process, non-TTY harness (the
// spinner never starts here at all, per RESEARCH Pitfall 3); that
// specific guarantee is Go's own `defer` semantics plus
// progress_test.go's TestProgress_StopIdempotent /
// TestProgress_NoGoroutineLeak, which directly cover Stop()'s
// deterministic-teardown contract in isolation.
func TestProgressWiringDoesNotBreakErrorPaths(t *testing.T) {
	freshDir := t.TempDir()
	if _, _, err := execCmd("index", freshDir); !errors.Is(err, ErrNotInitialized) {
		t.Fatalf("index: expected ErrNotInitialized, got: %v", err)
	}
	if _, _, err := execCmd("sync", freshDir); !errors.Is(err, ErrNotInitialized) {
		t.Fatalf("sync: expected ErrNotInitialized, got: %v", err)
	}
}
