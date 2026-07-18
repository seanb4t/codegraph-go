package tui

import (
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"time"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"

	"github.com/seanb4t/codegraph-go/internal/daemon"
)

// stopMatching/stopAll are daemon.StopMatching/daemon.StopAll indirected
// behind package-level func vars — the same injectable-seam convention as
// tty.go's stdinIsInteractive/stdoutIsTTY/noColor — so daemonpicker_test.go
// can drive dispatchDaemonAction without ever delivering a real OS signal.
var stopMatching = daemon.StopMatching
var stopAll = daemon.StopAll

// daemonItem adapts a daemon.Record into bubbles/v2/list's Item interface.
type daemonItem struct {
	record daemon.Record
}

func (i daemonItem) FilterValue() string { return i.record.RepoRoot }

// daemonDelegate renders one row: a "> " cursor on the focused row, the
// repo's basename, its pid, and its age (time.Since(StartedAt)). Unlike
// agentpicker's checkboxDelegate, the daemon picker is single-select (no
// per-item toggle state), so Update is a no-op — all key handling happens
// in daemonPickerModel.Update before ever reaching list.Model.
type daemonDelegate struct{}

func (d daemonDelegate) Height() int  { return 1 }
func (d daemonDelegate) Spacing() int { return 0 }

func (d daemonDelegate) Update(tea.Msg, *list.Model) tea.Cmd { return nil }

func (d daemonDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	di, ok := item.(daemonItem)
	if !ok {
		return
	}
	cursor := "  "
	if index == m.Index() {
		cursor = "> "
	}
	age := time.Since(di.record.StartedAt).Round(time.Second)
	fmt.Fprintf(w, "%s%s (pid %d, up %s)\n", cursor, filepath.Base(di.record.RepoRoot), di.record.PID, age)
}

// SortRecordsCurrentFirst orders records with the current repo's own
// record(s) first, then a stable secondary order by (RepoRoot, PID) — the
// DMON-01/TUI-04 ordering both this Model and daemon.go's plain non-TTY
// list (D-12) must share, so the two presentations never diverge. Exported
// so internal/cli's bare `daemon` RunE can reuse the exact same ordering
// for its plain-list fallback. currentRepo is compared by exact string
// equality — callers pass an already-absolutized path, matching how
// daemon.Run itself records RepoRoot (filepath.Abs'd in daemon.New).
func SortRecordsCurrentFirst(records []daemon.Record, currentRepo string) []daemon.Record {
	sorted := make([]daemon.Record, len(records))
	copy(sorted, records)
	sort.SliceStable(sorted, func(i, j int) bool {
		iCur := sorted[i].RepoRoot == currentRepo
		jCur := sorted[j].RepoRoot == currentRepo
		if iCur != jCur {
			return iCur
		}
		if sorted[i].RepoRoot != sorted[j].RepoRoot {
			return sorted[i].RepoRoot < sorted[j].RepoRoot
		}
		return sorted[i].PID < sorted[j].PID
	})
	return sorted
}

// daemonAction distinguishes the three terminal outcomes of the daemon
// picker (D-01/DMON-01): stop the focused daemon, stop every daemon, or
// leave everything running (quit/cancel/empty). Unexported — RunDaemonPicker
// resolves this from the Program's final Model and dispatches via
// dispatchDaemonAction; daemonpicker_test.go drives both directly with
// synthetic tea.Msg values, without a real tea.Program.
type daemonAction int

const (
	daemonActionNone daemonAction = iota
	daemonActionStopOne
	daemonActionStopAll
)

// daemonPickerModel is the bubbletea Model backing RunDaemonPicker: a
// bubbles/v2/list.Model over daemon.Record (sorted current-repo-first),
// rendered by daemonDelegate. action/target capture the terminal outcome
// resolved by Update — dispatchDaemonAction (called by RunDaemonPicker
// after the Program exits, and directly by tests) performs it.
type daemonPickerModel struct {
	list    list.Model
	records []daemon.Record
	action  daemonAction
	target  daemon.Record
}

// newDaemonPickerModel builds the Model from records (sorted
// current-repo-first via SortRecordsCurrentFirst) — a plain
// constructor-argument seam, mirroring newAgentPickerModel, so
// daemonpicker_test.go can drive the Model without a real registry.
func newDaemonPickerModel(currentRepo string, records []daemon.Record) daemonPickerModel {
	sorted := SortRecordsCurrentFirst(records, currentRepo)
	items := make([]list.Item, len(sorted))
	for i, r := range sorted {
		items[i] = daemonItem{record: r}
	}
	l := list.New(items, daemonDelegate{}, 0, 0)
	l.Title = "Running daemons"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)
	return daemonPickerModel{list: l, records: sorted}
}

// Init resolves an empty record set immediately (tea.Quit, no dispatch) —
// there is nothing to pick and nothing to render interaction for (the
// DMON-01/DMON-04 empty-registry edge).
func (m daemonPickerModel) Init() tea.Cmd {
	if len(m.records) == 0 {
		return tea.Quit
	}
	return nil
}

// Update intercepts enter/"a"/quit keys itself BEFORE ever forwarding a
// message to list.Model.Update — mirroring agentPickerModel's own
// enter-vs-quit interception — so RunDaemonPicker's caller can distinguish
// stop-one from stop-all from cancel deterministically. Everything else
// (navigation) reaches list.Model.Update unchanged.
func (m daemonPickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetSize(msg.Width, msg.Height)
		return m, nil
	case tea.KeyPressMsg:
		if len(m.records) == 0 {
			return m, tea.Quit
		}
		switch msg.String() {
		case "enter":
			m.action = daemonActionStopOne
			m.target = m.records[m.list.Index()]
			return m, tea.Quit
		case "a":
			m.action = daemonActionStopAll
			return m, tea.Quit
		case "q", "esc", "ctrl+c":
			m.action = daemonActionNone
			return m, tea.Quit
		}
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m daemonPickerModel) View() tea.View {
	if len(m.records) == 0 {
		return tea.NewView("no running daemons\n")
	}
	help := "\nenter: stop selected  a: stop all  q/esc: cancel\n"
	return tea.NewView(m.list.View() + help)
}

// resolvedAction returns the terminal action + target record
// RunDaemonPicker (or a test) dispatches on, from the Program's final
// Model.
func (m daemonPickerModel) resolvedAction() (daemonAction, daemon.Record) {
	return m.action, m.target
}

// dispatchDaemonAction performs the terminal action a completed daemon
// picker resolved to: stop-one/stop-all signal through the injected
// stopMatching/stopAll seams, returning every record actually signaled
// alongside the error; cancel (and the empty-registry case) is a pure
// no-op, returning (nil, nil). Unexported and called both by
// RunDaemonPicker and directly by daemonpicker_test.go, so Update's
// transitions are proven to drive the right dispatch without ever
// constructing a real tea.Program. The []daemon.Record return (WR-01,
// 07-REVIEW.md) lets RunDaemonPicker report the same per-record
// confirmation the non-interactive `daemon stop` path already gives via
// printStoppedDaemons — previously this discarded the return, leaving a
// successful stop-one/stop-all silent.
func dispatchDaemonAction(action daemonAction, target daemon.Record) ([]daemon.Record, error) {
	switch action {
	case daemonActionStopOne:
		return stopMatching(target.RepoRoot)
	case daemonActionStopAll:
		return stopAll()
	default:
		return nil, nil
	}
}

// printDaemonPickerResult prints confirmation lines for the picker's
// dispatch (WR-01, 07-REVIEW.md), mirroring internal/cli/daemon.go's
// printStoppedDaemons/newDaemonStopCmd output shape ("stopped pid %d (%s)"
// per record, a "nothing stopped" notice when the action attempted a stop
// but nothing matched) — duplicated here rather than imported from
// internal/cli to avoid an internal/cli/tui -> internal/cli import cycle
// (internal/cli already imports internal/cli/tui). Never called for
// daemonActionNone (cancel/quit/empty-registry): those already print
// nothing, by design, and dispatchDaemonAction returns (nil, nil) for them
// which would otherwise read as an indistinguishable "attempted, matched
// nothing" case.
func printDaemonPickerResult(w io.Writer, action daemonAction, stopped []daemon.Record) {
	if action == daemonActionNone {
		return
	}
	if len(stopped) == 0 {
		fmt.Fprintln(w, "nothing stopped")
		return
	}
	for _, rec := range stopped {
		fmt.Fprintf(w, "stopped pid %d (%s)\n", rec.PID, rec.RepoRoot)
	}
}

// RunDaemonPicker constructs and runs the daemon picker Program over
// records (sorted current-repo-first), wiring cmd's own stdin/stdout so
// both the real CLI and any test harness capture the same I/O every other
// command uses, then dispatches the resolved action and prints a
// confirmation of what was actually stopped (WR-01, 07-REVIEW.md) — giving
// the interactive path the same on-screen feedback `daemon stop [--all]`
// already gives non-interactively. RunDaemonPicker does NOT re-check
// tui.InteractiveAllowed itself — the caller (daemon.go) MUST have already
// gated on it (D-10); calling this off a non-TTY is the caller's bug, not
// this function's job to detect.
func RunDaemonPicker(cmd *cobra.Command, currentRepo string, records []daemon.Record) error {
	m := newDaemonPickerModel(currentRepo, records)

	p := tea.NewProgram(m, tea.WithInput(cmd.InOrStdin()), tea.WithOutput(cmd.OutOrStdout()))
	final, err := p.Run()
	if err != nil {
		return err
	}
	fm, ok := final.(daemonPickerModel)
	if !ok {
		return fmt.Errorf("tui: RunDaemonPicker: unexpected final model type %T", final)
	}
	action, target := fm.resolvedAction()
	stopped, err := dispatchDaemonAction(action, target)
	printDaemonPickerResult(cmd.OutOrStdout(), action, stopped)
	return err
}
