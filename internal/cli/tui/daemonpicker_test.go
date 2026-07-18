package tui

import (
	"bytes"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

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
func withStubbedDaemonStop(t *testing.T, matching func(string) ([]daemon.Record, error), all func() ([]daemon.Record, error)) {
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

// TestDaemonPickerModel_EnterDispatchesStopOne asserts pressing enter on the
// focused row resolves daemonActionStopOne targeting that row's record, and
// that dispatchDaemonAction routes it through the injected stopMatching
// seam with that record's RepoRoot — never touching stopAll.
func TestDaemonPickerModel_EnterDispatchesStopOne(t *testing.T) {
	var calledWith string
	withStubbedDaemonStop(t, func(repoRoot string) ([]daemon.Record, error) {
		calledWith = repoRoot
		return []daemon.Record{rec(repoRoot, 1)}, nil
	}, func() ([]daemon.Record, error) {
		t.Fatal("stopAll must not be called for stop-one")
		return nil, nil
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
	stopped, err := dispatchDaemonAction(action, target)
	if err != nil {
		t.Fatalf("dispatchDaemonAction: %v", err)
	}
	if calledWith != "/repo/a" {
		t.Fatalf("stopMatching called with %q, want /repo/a", calledWith)
	}
	if len(stopped) != 1 || stopped[0].RepoRoot != "/repo/a" {
		t.Fatalf("dispatchDaemonAction stopped = %v, want [/repo/a]", stopped)
	}
}

// TestDaemonPickerModel_DownNavigatesThenEnterTargetsFocusedRow proves
// navigation (list.Model's own CursorDown, forwarded by Update's default
// case) actually changes which record enter subsequently targets.
func TestDaemonPickerModel_DownNavigatesThenEnterTargetsFocusedRow(t *testing.T) {
	var calledWith string
	withStubbedDaemonStop(t, func(repoRoot string) ([]daemon.Record, error) {
		calledWith = repoRoot
		return nil, nil
	}, func() ([]daemon.Record, error) { return nil, nil })

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

	if _, err := dispatchDaemonAction(action, target); err != nil {
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
	withStubbedDaemonStop(t, func(string) ([]daemon.Record, error) {
		t.Fatal("stopMatching must not be called for stop-all")
		return nil, nil
	}, func() ([]daemon.Record, error) {
		stopAllCalled = true
		return nil, nil
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

	if _, err := dispatchDaemonAction(action, target); err != nil {
		t.Fatalf("dispatchDaemonAction: %v", err)
	}
	if !stopAllCalled {
		t.Fatal("expected stopAll to be called")
	}
}

// TestDaemonPickerModel_CancelSignalsNothing asserts q/esc/ctrl+c resolve
// daemonActionNone — dispatchDaemonAction must be a pure no-op, never
// calling stopMatching or stopAll.
func TestDaemonPickerModel_CancelSignalsNothing(t *testing.T) {
	withStubbedDaemonStop(t, func(string) ([]daemon.Record, error) {
		t.Fatal("stopMatching must not be called on cancel")
		return nil, nil
	}, func() ([]daemon.Record, error) {
		t.Fatal("stopAll must not be called on cancel")
		return nil, nil
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
	if _, err := dispatchDaemonAction(action, target); err != nil {
		t.Fatalf("dispatchDaemonAction: %v", err)
	}
}

// TestDaemonPickerModel_EmptyRecordsQuitsWithoutDispatch asserts an empty
// record set's Init() resolves immediately (tea.Quit) without ever
// signaling — the DMON-01/DMON-04 empty-registry edge.
func TestDaemonPickerModel_EmptyRecordsQuitsWithoutDispatch(t *testing.T) {
	withStubbedDaemonStop(t, func(string) ([]daemon.Record, error) {
		t.Fatal("stopMatching must not be called for an empty registry")
		return nil, nil
	}, func() ([]daemon.Record, error) {
		t.Fatal("stopAll must not be called for an empty registry")
		return nil, nil
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
	if _, err := dispatchDaemonAction(action, target); err != nil {
		t.Fatalf("dispatchDaemonAction: %v", err)
	}
}

// TestPrintDaemonPickerResult asserts WR-01's (07-REVIEW.md) confirmation
// output: a "stopped pid %d (%s)" line per signaled record for stop-one/
// stop-all, a "nothing stopped" notice when the action attempted a stop but
// nothing matched, and complete silence for daemonActionNone (cancel/quit/
// empty-registry already print nothing, by design).
func TestPrintDaemonPickerResult(t *testing.T) {
	tests := []struct {
		name    string
		action  daemonAction
		stopped []daemon.Record
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
			name:    "cancel prints nothing",
			action:  daemonActionNone,
			stopped: nil,
			want:    "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			printDaemonPickerResult(&buf, tc.action, tc.stopped)
			if got := buf.String(); got != tc.want {
				t.Fatalf("printDaemonPickerResult() = %q, want %q", got, tc.want)
			}
		})
	}
}
