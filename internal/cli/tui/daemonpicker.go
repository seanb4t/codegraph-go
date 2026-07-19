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

// helpFooterLines is the rendered height of the help footer View() appends
// below the list (a leading blank line + the key-hint line). Update reserves
// this many rows when sizing the list so list + footer fit the window
// exactly instead of overflowing and flickering (07-UAT test 1).
const helpFooterLines = 2

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

// resolveRepoRoot normalizes p via filepath.EvalSymlinks for comparison —
// duplicated from internal/daemon/stop.go's identically-named, unexported
// helper (WR-03, 07-REVIEW.md) rather than imported, since it isn't
// exported and this project's existing cross-package-duplication
// precedent (e.g. internal/gitmeta/worktree.go's realpath,
// internal/agents/cursor.go's inline EvalSymlinks) already normalizes
// symlinked paths this same way per-package rather than sharing one
// helper. Best-effort: if EvalSymlinks errors (e.g. the path no longer
// exists on disk), the original string is returned as-is, so a genuine
// mismatch degrades to plain string comparison instead of breaking
// ordering.
func resolveRepoRoot(p string) string {
	if resolved, err := filepath.EvalSymlinks(p); err == nil {
		return resolved
	}
	return p
}

// SortRecordsCurrentFirst orders records with the current repo's own
// record(s) first, then a stable secondary order by (RepoRoot, PID) — the
// DMON-01/TUI-04 ordering both this Model and daemon.go's plain non-TTY
// list (D-12) must share, so the two presentations never diverge. Exported
// so internal/cli's bare `daemon` RunE can reuse the exact same ordering
// for its plain-list fallback. currentRepo is compared via resolveRepoRoot
// (WR-03, 07-REVIEW.md) — the same filepath.EvalSymlinks normalization
// internal/daemon/stop.go's StopMatching uses to target a daemon — so the
// "is this my repo" ordering answer never diverges from the "is this my
// repo" targeting answer just because the two sides of a symlinked path
// (e.g. macOS's /tmp -> /private/tmp) were spelled differently. Secondary
// ordering (RepoRoot, PID) intentionally still compares the raw RepoRoot
// strings, not resolved ones — records within the same actual repo could
// have been recorded through different symlink spellings, and stable raw
// string ordering is enough to keep the secondary sort deterministic
// without adding another resolveRepoRoot call per comparison.
func SortRecordsCurrentFirst(records []daemon.Record, currentRepo string) []daemon.Record {
	resolvedCurrent := resolveRepoRoot(currentRepo)
	// Pair each record with its resolved RepoRoot up front, rather than
	// re-resolving inside the comparator — sort.SliceStable's comparator
	// runs O(n log n) times, and re-resolving per-comparison would
	// redundantly re-stat the same handful of paths via EvalSymlinks. The
	// pairing is sorted as one unit (not two parallel slices) specifically
	// so the resolved-path index never desyncs from its record across the
	// swaps sort.SliceStable performs.
	type entry struct {
		record   daemon.Record
		resolved string
	}
	entries := make([]entry, len(records))
	for i, r := range records {
		entries[i] = entry{record: r, resolved: resolveRepoRoot(r.RepoRoot)}
	}
	sort.SliceStable(entries, func(i, j int) bool {
		iCur := entries[i].resolved == resolvedCurrent
		jCur := entries[j].resolved == resolvedCurrent
		if iCur != jCur {
			return iCur
		}
		if entries[i].record.RepoRoot != entries[j].record.RepoRoot {
			return entries[i].record.RepoRoot < entries[j].record.RepoRoot
		}
		return entries[i].record.PID < entries[j].record.PID
	})
	sorted := make([]daemon.Record, len(entries))
	for i, e := range entries {
		sorted[i] = e.record
	}
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
		// Reserve rows for the help footer View() appends below the list so
		// the list + footer fit the window exactly (07-UAT test 1). Sizing
		// the list to the full window height overflows by the footer's lines,
		// which — combined with alt-screen off — scrolled the frame and
		// flickered; the list must own only (height - footer) rows. Floor at
		// 1 so a tiny window never sizes the list to a non-positive height.
		listHeight := msg.Height - helpFooterLines
		if listHeight < 1 {
			listHeight = 1
		}
		m.list.SetSize(msg.Width, listHeight)
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
	v := tea.NewView(m.list.View() + help)
	// bubbletea v2 makes alt-screen a per-View field (there is no
	// WithAltScreen Program option). Render the picker in the alternate
	// screen buffer (07-UAT test 1): without it bubbletea renders inline
	// below the prompt, and a full-height list that doesn't fit the remaining
	// space scrolls the main buffer every frame — heavy flicker, title/rows
	// pushed out of view (only the footer visible). The alt-screen is a
	// dedicated, frame-diffed full-window canvas, auto-restored on quit, so
	// the picker leaves the scrollback untouched.
	v.AltScreen = true
	return v
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
	// Defense-in-depth (07-UAT test 1 / G-07-1): never construct a Program
	// for an empty record set. The caller (daemon.go) already gates on
	// len(records) > 0, but opening a Program just to immediately tea.Quit
	// on the empty set leaks the terminal's DECRQM capability-probe
	// responses (\e[?2026;2$y \e[?2027;0$y) to stdout — the probes are sent
	// at Program start but the event loop exits before consuming the
	// replies. Nothing to pick => the plain notice, no Program, so this
	// function can never be the source of that leak regardless of caller.
	if len(records) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "no running daemons")
		return nil
	}

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
