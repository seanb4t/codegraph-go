package cli

import (
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"github.com/seanb4t/codegraph-go/internal/daemon"
)

// withStubbedDaemonPicker forces the interactiveAllowed/runDaemonPicker
// package-level seams for the duration of one test — interactiveAllowed
// always reports allowed (as if on a real TTY), and picker stands in for
// tui.RunDaemonPicker without ever constructing a real tea.Program. Both
// seams are restored on cleanup. Mirrors install_test.go's
// withStubbedPicker.
func withStubbedDaemonPicker(t *testing.T, picker func(*cobra.Command, string, []daemon.Record) error) {
	t.Helper()
	origAllowed, origPicker := interactiveAllowed, runDaemonPicker
	t.Cleanup(func() {
		interactiveAllowed = origAllowed
		runDaemonPicker = origPicker
	})
	interactiveAllowed = func(*cobra.Command) bool { return true }
	runDaemonPicker = picker
}

// withStubbedDaemonList forces the daemonList package-level seam to return
// a fixed record set for the duration of one test — bypassing daemon.List's
// own self-heal (which would otherwise prune any seeded record whose pid
// isn't a genuinely live OS process in this test binary).
func withStubbedDaemonList(t *testing.T, records []daemon.Record) {
	t.Helper()
	orig := daemonList
	t.Cleanup(func() { daemonList = orig })
	daemonList = func() ([]daemon.Record, error) { return records, nil }
}

// withStubbedDaemonStop forces the daemonStopMatching/daemonStopAll
// package-level seams for the duration of one test, so `daemon stop`
// dispatch assertions never deliver a real OS signal. nil leaves the
// corresponding seam at its real (daemon.StopMatching/StopAll) value.
func withStubbedDaemonStop(t *testing.T, matching func(string) ([]daemon.Record, error), all func() ([]daemon.Record, error)) {
	t.Helper()
	origMatching, origAll := daemonStopMatching, daemonStopAll
	t.Cleanup(func() {
		daemonStopMatching = origMatching
		daemonStopAll = origAll
	})
	if matching != nil {
		daemonStopMatching = matching
	}
	if all != nil {
		daemonStopAll = all
	}
}

// TestDaemonStartCmdPolicyDisabledExitsCleanly pins `daemon start`'s
// policy-disabled branch (03-REVIEW.md IN-06 — the only shipped consumer of
// round-1 WR-01's fix) after this plan moved it verbatim off the bare
// `daemon` command (D-01): with CODEGRAPH_NO_WATCH=1 exported, the command
// must exit cleanly (nil error — a policy-disabled watcher is a deliberate,
// explained state, not a failure) and print the D-12 guidance with the
// verbatim reason to stderr. The reason travels inside daemon.Run's typed
// watch.DisabledError (IN-05), so this also pins the errors.As extraction
// path.
func TestDaemonStartCmdPolicyDisabledExitsCleanly(t *testing.T) {
	dir := copyFixture(t)
	if _, _, err := execCmd("init", "--quiet", dir); err != nil {
		t.Fatalf("init fixture: %v", err)
	}

	t.Setenv("CODEGRAPH_NO_WATCH", "1")

	_, stderr, err := execCmd("daemon", "start", "--path", dir)
	if err != nil {
		t.Fatalf("daemon start with CODEGRAPH_NO_WATCH=1: want nil error (clean exit for a policy-disabled watcher), got: %v", err)
	}
	if want := "File watcher disabled — CODEGRAPH_NO_WATCH=1 is set"; !strings.Contains(stderr, want) {
		t.Fatalf("stderr %q does not contain the verbatim disabled message %q", stderr, want)
	}
	if want := "run `codegraph sync`"; !strings.Contains(stderr, want) {
		t.Fatalf("stderr %q does not contain the %s guidance", stderr, want)
	}
	// IN-05: the standalone daemon command deliberately drops serve's
	// "[CodeGraph MCP]" banner — it is not the MCP server.
	if strings.Contains(stderr, "[CodeGraph MCP]") {
		t.Fatalf("stderr %q carries the [CodeGraph MCP] banner; the standalone daemon command must not", stderr)
	}
}

// TestDaemonBareCmd_NonTTY_EmptyRegistry_PrintsNoRunningDaemons pins D-12's
// non-TTY fallback on an empty registry: a plain "no running daemons"
// notice, exit 0. execCmd's harness always sets stdin to a strings.Reader
// (never os.Stdin), so interactiveAllowed(cmd) is false here without any
// forcing — the SAME non-TTY path a real piped invocation takes.
func TestDaemonBareCmd_NonTTY_EmptyRegistry_PrintsNoRunningDaemons(t *testing.T) {
	fakeHome(t)

	out, _, err := execCmd("daemon")
	if err != nil {
		t.Fatalf("daemon: %v", err)
	}
	if !strings.Contains(out, "no running daemons") {
		t.Fatalf("expected 'no running daemons', got:\n%s", out)
	}
}

// TestDaemonBareCmd_NonTTY_PrintsSeededRecordsCurrentRepoFirst pins the
// DMON-01/TUI-04 ordering truth: the plain non-TTY list orders the current
// repo's record first, exactly like the picker's Model would.
func TestDaemonBareCmd_NonTTY_PrintsSeededRecordsCurrentRepoFirst(t *testing.T) {
	fakeHome(t)
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}

	other := daemon.Record{PID: 111, StartedAt: time.Now().UTC(), RepoRoot: "/other/repo"}
	current := daemon.Record{PID: 222, StartedAt: time.Now().UTC(), RepoRoot: cwd}
	withStubbedDaemonList(t, []daemon.Record{other, current})

	out, _, err := execCmd("daemon")
	if err != nil {
		t.Fatalf("daemon: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected a header line + 2 records, got %d lines:\n%s", len(lines), out)
	}
	if !strings.Contains(lines[1], cwd) {
		t.Fatalf("expected the current-repo record first, got:\n%s", out)
	}
	if !strings.Contains(lines[2], "/other/repo") {
		t.Fatalf("expected the other-repo record second, got:\n%s", out)
	}
}

// TestDaemonBareCmd_InteractiveAllowed_CallsRunDaemonPicker asserts that,
// with interactiveAllowed forced true (as if on a real TTY), the bare
// `daemon` RunE calls runDaemonPicker with the resolved current repo and
// daemon.List()'s records — proving the TTY-gate wiring without a real
// tea.Program (D-10).
func TestDaemonBareCmd_InteractiveAllowed_CallsRunDaemonPicker(t *testing.T) {
	fakeHome(t)
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}

	rec := daemon.Record{PID: 333, StartedAt: time.Now().UTC(), RepoRoot: cwd}
	withStubbedDaemonList(t, []daemon.Record{rec})

	var gotRepo string
	var gotRecords []daemon.Record
	withStubbedDaemonPicker(t, func(cmd *cobra.Command, currentRepo string, records []daemon.Record) error {
		gotRepo = currentRepo
		gotRecords = records
		return nil
	})

	if _, _, err := execCmd("daemon"); err != nil {
		t.Fatalf("daemon: %v", err)
	}
	if gotRepo != cwd {
		t.Fatalf("runDaemonPicker currentRepo = %q, want %q", gotRepo, cwd)
	}
	if len(gotRecords) != 1 || gotRecords[0].PID != 333 {
		t.Fatalf("runDaemonPicker records = %v, want exactly [pid 333]", gotRecords)
	}
}

// TestDaemonBareCmd_InteractiveAllowed_EmptyRegistry_NoPicker pins G-07-1
// (07-UAT test 1): with interactiveAllowed forced true (as if on a real
// TTY) but an EMPTY registry, the bare `daemon` RunE must NOT open the
// bubbletea picker — it must fall through to the plain "no running daemons"
// notice. Opening a Program for the empty set (which immediately tea.Quits)
// leaked the terminal's DECRQM capability-probe responses
// (\e[?2026;2$y \e[?2027;0$y) to stdout on a real TTY, garbage the piped
// (non-TTY) tests never saw because the terminal never replies off a TTY.
func TestDaemonBareCmd_InteractiveAllowed_EmptyRegistry_NoPicker(t *testing.T) {
	fakeHome(t)
	withStubbedDaemonList(t, nil)
	withStubbedDaemonPicker(t, func(*cobra.Command, string, []daemon.Record) error {
		t.Fatal("runDaemonPicker must NOT be called for an empty registry — nothing to pick; opening a Program leaks terminal capability-probe responses on a TTY (G-07-1)")
		return nil
	})

	out, _, err := execCmd("daemon")
	if err != nil {
		t.Fatalf("daemon: %v", err)
	}
	if !strings.Contains(out, "no running daemons") {
		t.Fatalf("expected 'no running daemons', got:\n%s", out)
	}
}

// TestDaemonStopCmd_All_EmptyRegistry_CleanNoOp pins DMON-02's empty edge:
// `daemon stop --all` against an empty registry is a clean no-op notice,
// exit 0 — not an error. Uses the real daemon.StopAll (empty registry is
// safe: no pid is ever signaled).
func TestDaemonStopCmd_All_EmptyRegistry_CleanNoOp(t *testing.T) {
	fakeHome(t)

	out, _, err := execCmd("daemon", "stop", "--all")
	if err != nil {
		t.Fatalf("daemon stop --all: %v", err)
	}
	if !strings.Contains(out, "no running daemons") {
		t.Fatalf("expected 'no running daemons', got:\n%s", out)
	}
}

// TestDaemonStopCmd_NoMatch_Notice pins DMON-02's no-match edge: `daemon
// stop` (no --all) against a registry with nothing for the current repo is
// a clean notice, exit 0.
func TestDaemonStopCmd_NoMatch_Notice(t *testing.T) {
	fakeHome(t)
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}

	out, _, err := execCmd("daemon", "stop")
	if err != nil {
		t.Fatalf("daemon stop: %v", err)
	}
	if !strings.Contains(out, "no running daemon for "+cwd) {
		t.Fatalf("expected a no-match notice for %s, got:\n%s", cwd, out)
	}
}

// TestDaemonStopCmd_DispatchesToStopMatching asserts `daemon stop` (no
// --all) resolves -p/--path to the current repo (A2) and dispatches
// through daemonStopMatching, never daemonStopAll — using the injected
// seam so no real process is signaled.
func TestDaemonStopCmd_DispatchesToStopMatching(t *testing.T) {
	fakeHome(t)
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}

	var gotRepo string
	withStubbedDaemonStop(t, func(repoRoot string) ([]daemon.Record, error) {
		gotRepo = repoRoot
		return []daemon.Record{{PID: 42, RepoRoot: repoRoot}}, nil
	}, func() ([]daemon.Record, error) {
		t.Fatal("daemonStopAll must not be called without --all")
		return nil, nil
	})

	out, _, err := execCmd("daemon", "stop")
	if err != nil {
		t.Fatalf("daemon stop: %v", err)
	}
	if gotRepo != cwd {
		t.Fatalf("daemonStopMatching called with %q, want %q", gotRepo, cwd)
	}
	if !strings.Contains(out, "stopped pid 42") {
		t.Fatalf("expected a stopped-pid line, got:\n%s", out)
	}
}

// TestDaemonStopCmd_All_DispatchesToStopAll asserts `daemon stop --all`
// dispatches through daemonStopAll, never daemonStopMatching.
func TestDaemonStopCmd_All_DispatchesToStopAll(t *testing.T) {
	fakeHome(t)

	var called bool
	withStubbedDaemonStop(t, func(string) ([]daemon.Record, error) {
		t.Fatal("daemonStopMatching must not be called with --all")
		return nil, nil
	}, func() ([]daemon.Record, error) {
		called = true
		return []daemon.Record{{PID: 7, RepoRoot: "/repo"}}, nil
	})

	out, _, err := execCmd("daemon", "stop", "--all")
	if err != nil {
		t.Fatalf("daemon stop --all: %v", err)
	}
	if !called {
		t.Fatal("expected daemonStopAll to be called")
	}
	if !strings.Contains(out, "stopped pid 7") {
		t.Fatalf("expected a stopped-pid line, got:\n%s", out)
	}
}

// TestDaemonStopCmd_AggregatedError_ExitsNonZero asserts a genuine stop
// error (e.g. errors.Join from 07-04's stopTargets) is surfaced as a
// non-zero exit, never swallowed like the clean no-op/no-match cases.
func TestDaemonStopCmd_AggregatedError_ExitsNonZero(t *testing.T) {
	fakeHome(t)
	wantErr := errors.New("boom")
	withStubbedDaemonStop(t, nil, func() ([]daemon.Record, error) {
		return nil, wantErr
	})

	if _, _, err := execCmd("daemon", "stop", "--all"); err == nil {
		t.Fatal("expected daemon stop --all to return a non-nil error when daemonStopAll errors")
	}
}

// TestDaemonStopCmd_AllAndPath_MutuallyExclusive pins WR-02 (07-REVIEW.md):
// before this, `daemon stop --all --path <p>` silently ignored --path (the
// RunE's `if all { ...; return }` branch never looked at path) with no
// error. cmd.MarkFlagsMutuallyExclusive must now reject the combination
// before RunE ever dispatches to daemonStopMatching/daemonStopAll.
func TestDaemonStopCmd_AllAndPath_MutuallyExclusive(t *testing.T) {
	fakeHome(t)

	withStubbedDaemonStop(t, func(string) ([]daemon.Record, error) {
		t.Fatal("daemonStopMatching must not be called when --all/--path conflict")
		return nil, nil
	}, func() ([]daemon.Record, error) {
		t.Fatal("daemonStopAll must not be called when --all/--path conflict")
		return nil, nil
	})

	if _, _, err := execCmd("daemon", "stop", "--all", "--path", "/repo"); err == nil {
		t.Fatal("expected daemon stop --all --path <p> to return a non-nil error (mutually exclusive flags)")
	}
}
