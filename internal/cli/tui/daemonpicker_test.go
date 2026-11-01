package tui

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"charm.land/lipgloss/v2"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/exp/teatest/v2"
	"github.com/spf13/cobra"

	"github.com/seanb4t/codegraph-go/internal/daemon"
)

// rec builds a minimal daemon.Record for tests — a fresh StartedAt is fine
// since none of these tests assert on rendered age text, only on
// ordering/dispatch.
func rec(repoRoot string, pid int) daemon.Record {
	return daemon.Record{PID: pid, StartedAt: time.Now().UTC(), RepoRoot: repoRoot}
}

// downKeyMsg/aKeyMsg build synthetic tea.KeyPressMsg values the same way
// agentpicker_test.go's spaceKeyMsg/enterKeyMsg/quitKeyMsg do (Code set to
// the special key constant or the rune, Text set only for printable keys).
func downKeyMsg() tea.KeyPressMsg { return tea.KeyPressMsg{Code: tea.KeyDown} }
func aKeyMsg() tea.KeyPressMsg    { return tea.KeyPressMsg{Code: 'a', Text: "a"} }

// withStubbedDaemonStop points the package-level stopMatching/stopAll seams
// (Task 1's plan-mandated injection) at capturing/failing stubs for the
// duration of one test, restoring the originals on cleanup — so
// dispatchDaemonAction can be exercised without ever signaling a real
// process.
func withStubbedDaemonStop(t *testing.T, matching func(string) ([]daemon.Record, []string, error), all func() ([]daemon.Record, []string, error)) {
	t.Helper()
	origMatching, origAll := stopMatching, stopAll
	t.Cleanup(func() {
		stopMatching = origMatching
		stopAll = origAll
	})
	if matching != nil {
		stopMatching = matching
	}
	if all != nil {
		stopAll = all
	}
}

// TestDaemonPickerModel_CurrentRepoFirstOrdering asserts newDaemonPickerModel
// sorts the current repo's record(s) first, then a stable (RepoRoot, PID)
// secondary order — the DMON-01/TUI-04 ordering both this Model and
// daemon.go's plain non-TTY list must share.
func TestDaemonPickerModel_CurrentRepoFirstOrdering(t *testing.T) {
	records := []daemon.Record{
		rec("/repo/b", 2),
		rec("/repo/a", 1),
		rec("/repo/current", 3),
	}

	m := newDaemonPickerModel("/repo/current", records)

	if len(m.records) != 3 {
		t.Fatalf("want 3 records, got %d", len(m.records))
	}
	if m.records[0].RepoRoot != "/repo/current" {
		t.Fatalf("records[0] = %q, want the current repo first", m.records[0].RepoRoot)
	}
	if m.records[1].RepoRoot != "/repo/a" || m.records[2].RepoRoot != "/repo/b" {
		t.Fatalf("secondary order not stable by RepoRoot ascending: got %v", m.records)
	}
}

// TestSortRecordsCurrentFirst_NormalizesSymlinks pins WR-03 (07-REVIEW.md):
// the current-repo comparison must resolve symlinks the same way
// internal/daemon/stop.go's StopMatching does, so a record whose RepoRoot
// was recorded through a differently-spelled (but same-target) symlink
// path still sorts first — matching macOS's /tmp -> /private/tmp or any
// project accessed via two symlink paths.
func TestSortRecordsCurrentFirst_NormalizesSymlinks(t *testing.T) {
	real := t.TempDir()
	linkParent := t.TempDir()
	link := filepath.Join(linkParent, "link")
	if err := os.Symlink(real, link); err != nil {
		t.Skipf("symlink unsupported in this environment: %v", err)
	}

	// The daemon's own record was written through the symlinked path;
	// currentRepo is resolved to the real path (or vice versa — either
	// spelling must match the other).
	records := []daemon.Record{
		rec("/other/repo", 9),
		rec(link, 1),
	}

	sorted := SortRecordsCurrentFirst(records, real)
	if sorted[0].RepoRoot != link {
		t.Fatalf("sorted[0].RepoRoot = %q, want %q (current repo first, via symlink normalization)", sorted[0].RepoRoot, link)
	}
}

// TestSortRecordsCurrentFirst_SymlinkEvalErrorDegradesToRawString pins
// WR-03's fallback: resolveRepoRoot's EvalSymlinks failure (e.g. a
// recorded RepoRoot that no longer exists on disk) must degrade to plain
// string comparison rather than panicking or breaking the whole sort.
func TestSortRecordsCurrentFirst_SymlinkEvalErrorDegradesToRawString(t *testing.T) {
	records := []daemon.Record{
		rec("/nonexistent/repo/b", 2),
		rec("/nonexistent/repo/current", 1),
	}

	sorted := SortRecordsCurrentFirst(records, "/nonexistent/repo/current")
	if sorted[0].RepoRoot != "/nonexistent/repo/current" {
		t.Fatalf("sorted[0].RepoRoot = %q, want the current repo first even when EvalSymlinks errors", sorted[0].RepoRoot)
	}
}

// TestDaemonPickerModel_EnterDispatchesStopOne asserts pressing enter on the
// focused row resolves daemonActionStopOne targeting that row's record, and
// that dispatchDaemonAction routes it through the injected stopMatching
// seam with that record's RepoRoot — never touching stopAll.
func TestDaemonPickerModel_EnterDispatchesStopOne(t *testing.T) {
	var calledWith string
	wantSkipped := []string{"/state/daemons/4242.json"}
	withStubbedDaemonStop(t, func(repoRoot string) ([]daemon.Record, []string, error) {
		calledWith = repoRoot
		return []daemon.Record{rec(repoRoot, 1)}, wantSkipped, nil
	}, func() ([]daemon.Record, []string, error) {
		t.Fatal("stopAll must not be called for stop-one")
		return nil, nil, nil
	})

	records := []daemon.Record{rec("/repo/a", 1), rec("/repo/b", 2)}
	m := newDaemonPickerModel("/repo/a", records)

	updated, cmd := m.Update(enterKeyMsg())
	m2, ok := updated.(daemonPickerModel)
	if !ok {
		t.Fatalf("Update returned %T, want daemonPickerModel", updated)
	}
	if cmd == nil {
		t.Fatal("enter: want a non-nil Cmd (tea.Quit), got nil")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatal("enter: want the Cmd to produce tea.QuitMsg")
	}

	action, target := m2.resolvedAction()
	if action != daemonActionStopOne {
		t.Fatalf("action = %v, want daemonActionStopOne", action)
	}
	if target.RepoRoot != "/repo/a" {
		t.Fatalf("target.RepoRoot = %q, want /repo/a (current-repo-first index 0)", target.RepoRoot)
	}

	// WR-01 (07-REVIEW.md): dispatchDaemonAction must surface the
	// signaled records, not just the error, so RunDaemonPicker can print
	// the same confirmation the non-interactive `daemon stop` path gives.
	stopped, skipped, err := dispatchDaemonAction(action, target)
	if err != nil {
		t.Fatalf("dispatchDaemonAction: %v", err)
	}
	if calledWith != "/repo/a" {
		t.Fatalf("stopMatching called with %q, want /repo/a", calledWith)
	}
	if len(stopped) != 1 || stopped[0].RepoRoot != "/repo/a" {
		t.Fatalf("dispatchDaemonAction stopped = %v, want [/repo/a]", stopped)
	}
	if !slices.Equal(skipped, wantSkipped) {
		t.Fatalf("dispatchDaemonAction skipped = %q, want %q", skipped, wantSkipped)
	}
}

// TestDaemonPickerModel_DownNavigatesThenEnterTargetsFocusedRow proves
// navigation (list.Model's own CursorDown, forwarded by Update's default
// case) actually changes which record enter subsequently targets.
func TestDaemonPickerModel_DownNavigatesThenEnterTargetsFocusedRow(t *testing.T) {
	var calledWith string
	withStubbedDaemonStop(t, func(repoRoot string) ([]daemon.Record, []string, error) {
		calledWith = repoRoot
		return nil, nil, nil
	}, func() ([]daemon.Record, []string, error) { return nil, nil, nil })

	records := []daemon.Record{rec("/repo/current", 1), rec("/repo/other", 2)}
	m := newDaemonPickerModel("/repo/current", records)

	updated, _ := m.Update(downKeyMsg())
	m2, ok := updated.(daemonPickerModel)
	if !ok {
		t.Fatalf("Update returned %T, want daemonPickerModel", updated)
	}
	if m2.list.Index() != 1 {
		t.Fatalf("after down: list.Index() = %d, want 1", m2.list.Index())
	}

	updated2, _ := m2.Update(enterKeyMsg())
	m3, ok := updated2.(daemonPickerModel)
	if !ok {
		t.Fatalf("Update returned %T, want daemonPickerModel", updated2)
	}
	action, target := m3.resolvedAction()
	if action != daemonActionStopOne || target.RepoRoot != "/repo/other" {
		t.Fatalf("after down+enter: action=%v target=%v, want stop-one on /repo/other", action, target)
	}

	if _, _, err := dispatchDaemonAction(action, target); err != nil {
		t.Fatalf("dispatchDaemonAction: %v", err)
	}
	if calledWith != "/repo/other" {
		t.Fatalf("stopMatching called with %q, want /repo/other", calledWith)
	}
}

// TestDaemonPickerModel_StopAllDispatches asserts the "a" key resolves
// daemonActionStopAll (no per-record target) and dispatchDaemonAction
// routes it through the injected stopAll seam — never stopMatching.
func TestDaemonPickerModel_StopAllDispatches(t *testing.T) {
	var stopAllCalled bool
	wantSkipped := []string{"/state/daemons/4242.json", "/state/daemons/4343.json"}
	withStubbedDaemonStop(t, func(string) ([]daemon.Record, []string, error) {
		t.Fatal("stopMatching must not be called for stop-all")
		return nil, nil, nil
	}, func() ([]daemon.Record, []string, error) {
		stopAllCalled = true
		return nil, wantSkipped, nil
	})

	records := []daemon.Record{rec("/repo/a", 1), rec("/repo/b", 2)}
	m := newDaemonPickerModel("/repo/a", records)

	updated, cmd := m.Update(aKeyMsg())
	m2, ok := updated.(daemonPickerModel)
	if !ok {
		t.Fatalf("Update returned %T, want daemonPickerModel", updated)
	}
	if cmd == nil {
		t.Fatal("a: want a non-nil Cmd (tea.Quit), got nil")
	}

	action, target := m2.resolvedAction()
	if action != daemonActionStopAll {
		t.Fatalf("action = %v, want daemonActionStopAll", action)
	}

	_, skipped, err := dispatchDaemonAction(action, target)
	if err != nil {
		t.Fatalf("dispatchDaemonAction: %v", err)
	}
	if !stopAllCalled {
		t.Fatal("expected stopAll to be called")
	}
	if !slices.Equal(skipped, wantSkipped) {
		t.Fatalf("dispatchDaemonAction skipped = %q, want %q", skipped, wantSkipped)
	}
}

// TestFinishDaemonPicker_StopAllReturnsSkippedPaths: a picker that finished
// on stop-all dispatches through stopAll, prints what it stopped, and returns
// the registry paths the stop's listing skipped, which RunDaemonPicker hands
// to the bare `daemon` command.
func TestFinishDaemonPicker_StopAllReturnsSkippedPaths(t *testing.T) {
	wantSkipped := []string{"/state/daemons/4242.json"}
	withStubbedDaemonStop(t, func(string) ([]daemon.Record, []string, error) {
		t.Fatal("stopMatching must not be called for stop-all")
		return nil, nil, nil
	}, func() ([]daemon.Record, []string, error) {
		return []daemon.Record{rec("/repo/a", 7)}, wantSkipped, nil
	})

	m := newDaemonPickerModel("/repo/a", []daemon.Record{rec("/repo/a", 7)})
	final, _ := m.Update(aKeyMsg())

	var out bytes.Buffer
	skipped, err := finishDaemonPicker(&out, final)
	if err != nil {
		t.Fatalf("finishDaemonPicker: %v", err)
	}
	if !slices.Equal(skipped, wantSkipped) {
		t.Fatalf("finishDaemonPicker skipped = %q, want %q", skipped, wantSkipped)
	}
	if want := "stopped pid 7 (/repo/a)\n"; out.String() != want {
		t.Fatalf("output = %q, want %q", out.String(), want)
	}
}

// TestFinishDaemonPicker_FailedStopAllPrintsNoNotice: a picker whose stop-all
// failed before stopping anything returns the error and prints nothing.
func TestFinishDaemonPicker_FailedStopAllPrintsNoNotice(t *testing.T) {
	wantErr := errors.New("boom")
	withStubbedDaemonStop(t, func(string) ([]daemon.Record, []string, error) {
		t.Fatal("stopMatching must not be called for stop-all")
		return nil, nil, nil
	}, func() ([]daemon.Record, []string, error) {
		return nil, nil, wantErr
	})

	m := newDaemonPickerModel("/repo/a", []daemon.Record{rec("/repo/a", 7)})
	final, _ := m.Update(aKeyMsg())

	var out bytes.Buffer
	_, err := finishDaemonPicker(&out, final)
	if !errors.Is(err, wantErr) {
		t.Fatalf("finishDaemonPicker: err = %v, want %v", err, wantErr)
	}
	if out.String() != "" {
		t.Fatalf("output = %q, want none", out.String())
	}
}

// TestDaemonPickerModel_CancelSignalsNothing asserts q/esc/ctrl+c resolve
// daemonActionNone — dispatchDaemonAction must be a pure no-op, never
// calling stopMatching or stopAll.
func TestDaemonPickerModel_CancelSignalsNothing(t *testing.T) {
	withStubbedDaemonStop(t, func(string) ([]daemon.Record, []string, error) {
		t.Fatal("stopMatching must not be called on cancel")
		return nil, nil, nil
	}, func() ([]daemon.Record, []string, error) {
		t.Fatal("stopAll must not be called on cancel")
		return nil, nil, nil
	})

	records := []daemon.Record{rec("/repo/a", 1)}
	m := newDaemonPickerModel("/repo/a", records)

	updated, cmd := m.Update(quitKeyMsg())
	m2, ok := updated.(daemonPickerModel)
	if !ok {
		t.Fatalf("Update returned %T, want daemonPickerModel", updated)
	}
	if cmd == nil {
		t.Fatal("quit: want a non-nil Cmd (tea.Quit), got nil")
	}

	action, target := m2.resolvedAction()
	if action != daemonActionNone {
		t.Fatalf("action = %v, want daemonActionNone", action)
	}
	if _, _, err := dispatchDaemonAction(action, target); err != nil {
		t.Fatalf("dispatchDaemonAction: %v", err)
	}
}

// TestDaemonPickerModel_EmptyRecordsQuitsWithoutDispatch asserts an empty
// record set's Init() resolves immediately (tea.Quit) without ever
// signaling — the DMON-01/DMON-04 empty-registry edge.
func TestDaemonPickerModel_EmptyRecordsQuitsWithoutDispatch(t *testing.T) {
	withStubbedDaemonStop(t, func(string) ([]daemon.Record, []string, error) {
		t.Fatal("stopMatching must not be called for an empty registry")
		return nil, nil, nil
	}, func() ([]daemon.Record, []string, error) {
		t.Fatal("stopAll must not be called for an empty registry")
		return nil, nil, nil
	})

	m := newDaemonPickerModel("/repo/a", nil)

	cmd := m.Init()
	if cmd == nil {
		t.Fatal("Init on empty records: want a non-nil Cmd (tea.Quit), got nil")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatal("Init on empty records: want the Cmd to produce tea.QuitMsg")
	}

	action, target := m.resolvedAction()
	if action != daemonActionNone {
		t.Fatalf("action = %v, want daemonActionNone", action)
	}
	if _, _, err := dispatchDaemonAction(action, target); err != nil {
		t.Fatalf("dispatchDaemonAction: %v", err)
	}
}

// TestPrintDaemonPickerResult asserts WR-01's (07-REVIEW.md) confirmation
// output: a "stopped pid %d (%s)" line per signaled record for stop-one/
// stop-all, a "nothing stopped" notice when the action attempted a stop but
// nothing matched, nothing at all when a stop-one or stop-all failed before
// stopping anything, and complete silence for daemonActionNone (cancel/quit/
// empty-registry already print nothing, by design).
func TestPrintDaemonPickerResult(t *testing.T) {
	boom := errors.New("boom")
	tests := []struct {
		name    string
		action  daemonAction
		stopped []daemon.Record
		err     error
		want    string
	}{
		{
			name:    "stop-one with a signaled record",
			action:  daemonActionStopOne,
			stopped: []daemon.Record{rec("/repo/a", 42)},
			want:    "stopped pid 42 (/repo/a)\n",
		},
		{
			name:    "stop-all with multiple signaled records",
			action:  daemonActionStopAll,
			stopped: []daemon.Record{rec("/repo/a", 1), rec("/repo/b", 2)},
			want:    "stopped pid 1 (/repo/a)\nstopped pid 2 (/repo/b)\n",
		},
		{
			name:    "stop-one with nothing matched",
			action:  daemonActionStopOne,
			stopped: nil,
			want:    "nothing stopped\n",
		},
		{
			name:    "stop-one that failed prints no notice",
			action:  daemonActionStopOne,
			stopped: nil,
			err:     boom,
			want:    "",
		},
		{
			name:    "stop-all that failed prints no notice",
			action:  daemonActionStopAll,
			stopped: nil,
			err:     boom,
			want:    "",
		},
		{
			name:    "stop-all that failed part-way names what it stopped",
			action:  daemonActionStopAll,
			stopped: []daemon.Record{rec("/repo/a", 1)},
			err:     boom,
			want:    "stopped pid 1 (/repo/a)\n",
		},
		{
			name:    "cancel prints nothing",
			action:  daemonActionNone,
			stopped: nil,
			want:    "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			printDaemonPickerResult(&buf, tc.action, tc.stopped, tc.err)
			if got := buf.String(); got != tc.want {
				t.Fatalf("printDaemonPickerResult() = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestDaemonPickerFootprintFitsDefaultPane is the daemon-picker sibling of
// TestAgentPickerFootprintFitsDefaultPane (D-24/D-25): 8 records at the
// default 100x30 pane must fit within 30 lines and still show the picker's
// own footer text. Before the delegate fix (daemonDelegate.Render's own
// trailing newline, same defect as checkboxDelegate), each row costs 2
// lines and the footer overflows out of the pane.
func TestDaemonPickerFootprintFitsDefaultPane(t *testing.T) {
	records := make([]daemon.Record, 8)
	for i := range records {
		records[i] = rec(fmt.Sprintf("/repo/r%d", i), 1000+i)
	}

	m := newDaemonPickerModel("/repo/r0", records)
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m2, ok := updated.(daemonPickerModel)
	if !ok {
		t.Fatalf("Update(WindowSizeMsg) returned %T, want daemonPickerModel", updated)
	}

	content := m2.View().Content
	if h := lipgloss.Height(content); h > 30 {
		t.Fatalf("lipgloss.Height(view.Content) = %d, want <= 30 at 100x30 with 8 records:\n%s", h, content)
	}
	if !strings.Contains(content, "enter: stop selected") {
		t.Fatalf("view content missing footer text \"enter: stop selected\":\n%s", content)
	}
}

// TestDaemonPickerLeavesAltScreenLast is TTY-04 at the model level: the
// picker draws its frame, including the seeded record, inside the alternate
// screen. Once it leaves the alternate screen it writes only DEC private
// mode resets: no text in the restored main buffer and no mode query whose
// reply could reach the shell after exit. The test quits only after the
// first frame, as a user does; a Program that quits before its first render
// tick writes its startup queries after the leave sequence, which this
// model cannot control.
func TestDaemonPickerLeavesAltScreenLast(t *testing.T) {
	records := []daemon.Record{rec("/repo/alpha", 101), rec("/repo/bravo", 102)}
	tm := teatest.NewTestModel(t, newDaemonPickerModel("/repo/alpha", records), teatest.WithInitialTermSize(100, 30))

	first := waitForFrame(t, tm, "enter: stop selected")
	tm.Send(quitKeyMsg())
	out := slices.Concat(first, finalOutput(t, tm))

	if len(out) == 0 {
		t.Fatal("the program wrote 0 bytes")
	}
	if n := bytes.Count(out, []byte(altScreenEnter)); n != 1 {
		t.Fatalf("alt-screen enter written %d times, want 1:\n%q", n, out)
	}
	if n := bytes.Count(out, []byte(altScreenLeave)); n != 1 {
		t.Fatalf("alt-screen leave written %d times, want 1:\n%q", n, out)
	}
	enter := bytes.Index(out, []byte(altScreenEnter))
	leave := bytes.Index(out, []byte(altScreenLeave))
	if leave < enter {
		t.Fatalf("alt-screen leave at byte %d comes before the enter at byte %d:\n%q", leave, enter, out)
	}

	if text := textOf(out[:enter]); text != "" {
		t.Fatalf("text written to the main buffer before the alt screen: %q", text)
	}
	inside := out[enter:leave]
	for _, want := range []string{"Running daemons", "alpha", "bravo"} {
		if !bytes.Contains(inside, []byte(want)) {
			t.Fatalf("the alt-screen frame does not contain %q:\n%q", want, inside)
		}
	}
	tail := out[leave+len(altScreenLeave):]
	if !privateModesOnly.Match(tail) {
		t.Fatalf("TTY-04: after leaving the alt screen the picker wrote more than mode resets:\n%q", tail)
	}
}

// TestDaemonEmptyRegistryWritesNoModeQuery is TTY-03: on an empty registry
// RunDaemonPicker prints the plain notice and starts no Program. A Program
// writes its DECRQM mode queries at startup; one that quits at once lets the
// terminal's replies reach the shell (G-07-1). So the output must be the
// notice alone, with no mode query and no control sequence at all.
func TestDaemonEmptyRegistryWritesNoModeQuery(t *testing.T) {
	var out bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetIn(strings.NewReader(""))
	cmd.SetOut(&out)

	if _, err := RunDaemonPicker(cmd, "/repo/a", nil); err != nil {
		t.Fatalf("RunDaemonPicker: %v", err)
	}

	if out.Len() == 0 {
		t.Fatal("RunDaemonPicker wrote 0 bytes on an empty registry")
	}
	if loc := modeQuery.FindIndex(out.Bytes()); loc != nil {
		t.Fatalf("TTY-03: empty registry wrote a mode query at byte %d:\n%q", loc[0], out.Bytes())
	}
	if got, want := out.String(), "no running daemons\n"; got != want {
		t.Fatalf("empty registry output = %q, want %q", got, want)
	}
}
