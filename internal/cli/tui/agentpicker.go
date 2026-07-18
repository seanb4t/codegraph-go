package tui

import (
	"fmt"
	"io"
	"sort"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"

	"github.com/seanb4t/codegraph-go/internal/agents"
)

// agentItem adapts an agents.AgentTarget into bubbles/v2/list's Item
// interface (D-14) — FilterValue is DisplayName even though the list's
// built-in filter is disabled (newAgentPickerModel), satisfying the
// interface with the same text the row renders.
type agentItem struct {
	target agents.AgentTarget
}

func (i agentItem) FilterValue() string { return i.target.DisplayName() }

// checkboxDelegate is the hand-rolled multi-select ItemDelegate
// bubbles/v2/list has no built-in equivalent for (RESEARCH.md Pattern 2):
// it tracks a checked[index] map and toggles it in ITS OWN Update — not the
// outer Model's — so the toggle can never collide with list.Model's own
// KeyMap dispatch (RESEARCH.md Pitfall 5).
type checkboxDelegate struct {
	checked map[int]bool
}

func (d *checkboxDelegate) Height() int  { return 1 }
func (d *checkboxDelegate) Spacing() int { return 0 }

// Update toggles the focused row's checked state on a space key press.
// list.Model's own Update calls this for every message except while the
// user is filtering (bubbles/v2/list's ItemDelegate contract) — the
// sanctioned per-item hook for delegate-owned key handling.
func (d *checkboxDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd {
	if key, ok := msg.(tea.KeyPressMsg); ok && key.String() == "space" {
		i := m.Index()
		d.checked[i] = !d.checked[i]
	}
	return nil
}

// Render draws one row: a "> " cursor on the focused row, a "[x]"/"[ ]"
// checkbox reflecting d.checked, then the target's display name.
func (d *checkboxDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	ai, ok := item.(agentItem)
	if !ok {
		return
	}
	box := "[ ]"
	if d.checked[index] {
		box = "[x]"
	}
	cursor := "  "
	if index == m.Index() {
		cursor = "> "
	}
	fmt.Fprintf(w, "%s%s %s\n", cursor, box, ai.target.DisplayName())
}

// agentPickerModel is the bubbletea Model backing RunAgentPicker: a
// bubbles/v2/list.Model over agents.AllTargets(), rendered by
// checkboxDelegate. confirmed distinguishes Enter (resolve the checked set)
// from quit/cancel (resolve to no selection) — both terminate the Program;
// only Enter treats the checked map as meaningful.
type agentPickerModel struct {
	list      list.Model
	delegate  *checkboxDelegate
	targets   []agents.AgentTarget
	confirmed bool
}

// newAgentPickerModel builds the Model from all (agents.AllTargets(), or a
// test's fake roster) and detection (agents.DetectAll(loc), or a test's
// fake result set) — a plain constructor-argument seam (no package var
// needed) so agentpicker_test.go can drive the Model without depending on
// the real registry. checked[i] is pre-seeded true for every target
// detection reports Installed (D-14's pre-check contract).
func newAgentPickerModel(all []agents.AgentTarget, detection map[agents.TargetID]agents.DetectionResult) agentPickerModel {
	checked := make(map[int]bool, len(all))
	items := make([]list.Item, len(all))
	for i, t := range all {
		items[i] = agentItem{target: t}
		if detection[t.ID()].Installed {
			checked[i] = true
		}
	}
	delegate := &checkboxDelegate{checked: checked}
	l := list.New(items, delegate, 0, 0)
	l.Title = "Select agents to configure"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)
	l.DisableQuitKeybindings()
	return agentPickerModel{list: l, delegate: delegate, targets: all}
}

func (m agentPickerModel) Init() tea.Cmd { return nil }

// Update intercepts enter/quit keys itself — BEFORE forwarding to
// list.Model.Update — so RunAgentPicker's caller can distinguish "confirm"
// from "cancel" deterministically, rather than relying on list.Model's own
// Quit keymap (bound to the same q/esc keys, but with no way for Update to
// tell confirmed apart from canceled). list.Model's own quit keybindings are
// disabled (newAgentPickerModel) so q/esc always reach this switch instead
// of being swallowed by list.Model first. Every other message (including
// the space toggle) is forwarded to list.Model.Update, which itself
// dispatches to checkboxDelegate.Update per bubbles/v2/list's own contract.
func (m agentPickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetSize(msg.Width, msg.Height)
		return m, nil
	case tea.KeyPressMsg:
		switch msg.String() {
		case "enter":
			m.confirmed = true
			return m, tea.Quit
		case "q", "esc", "ctrl+c":
			m.confirmed = false
			return m, tea.Quit
		}
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m agentPickerModel) View() tea.View {
	help := "\nspace: toggle  enter: confirm  q/esc: cancel\n"
	return tea.NewView(m.list.View() + help)
}

// resolvedTargets maps the model's final checked-index set through
// selectByIndices — the SAME dedup+ascending-order semantics the legacy
// promptAgentMultiSelect used (D-14) — after Enter; an empty slice after
// quit/cancel, never an error.
func (m agentPickerModel) resolvedTargets() []agents.AgentTarget {
	if !m.confirmed {
		return nil
	}
	var indices []int
	for i, checked := range m.delegate.checked {
		if checked {
			indices = append(indices, i)
		}
	}
	return selectByIndices(m.targets, indices)
}

// selectByIndices mirrors internal/cli/install.go's selectByIndices (dedup
// + ascending index order over the target slice) verbatim. Duplicated
// rather than imported: internal/cli imports internal/cli/tui
// (RunAgentPicker's caller), so importing internal/cli back from here would
// be a cycle.
func selectByIndices(all []agents.AgentTarget, indices []int) []agents.AgentTarget {
	sort.Ints(indices)
	out := make([]agents.AgentTarget, 0, len(indices))
	seen := make(map[int]bool, len(indices))
	for _, i := range indices {
		if seen[i] {
			continue
		}
		seen[i] = true
		out = append(out, all[i])
	}
	return out
}

// RunAgentPicker constructs and runs the checkbox multi-select Program over
// agents.AllTargets(), pre-checked from agents.DetectAll(loc), wiring cmd's
// own stdin/stdout so both the real CLI and any test harness capture the
// same I/O every other command uses. RunAgentPicker does NOT re-check
// tui.InteractiveAllowed itself — the caller (install.go/uninstall.go) MUST
// have already gated on it (D-10); calling this off a non-TTY is the
// caller's bug, not this function's job to detect.
func RunAgentPicker(cmd *cobra.Command, loc agents.Location) ([]agents.AgentTarget, error) {
	all := agents.AllTargets()
	detection := agents.DetectAll(loc)
	m := newAgentPickerModel(all, detection)

	p := tea.NewProgram(m, tea.WithInput(cmd.InOrStdin()), tea.WithOutput(cmd.OutOrStdout()))
	final, err := p.Run()
	if err != nil {
		return nil, err
	}
	fm, ok := final.(agentPickerModel)
	if !ok {
		return nil, fmt.Errorf("tui: RunAgentPicker: unexpected final model type %T", final)
	}
	return fm.resolvedTargets(), nil
}
