package tui

import (
	"fmt"
	"io"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"

	"github.com/seanb4t/codegraph-go/internal/agents"
)

// agentItem adapts an agents.AgentTarget into bubbles/v2/list's Item
// interface (D-14).
type agentItem struct {
	target agents.AgentTarget
}

func (i agentItem) FilterValue() string { return i.target.DisplayName() }

// checkboxDelegate is the hand-rolled multi-select ItemDelegate
// bubbles/v2/list has no built-in equivalent for (RESEARCH.md Pattern 2).
//
// TODO(RED): Update does not yet toggle checked — GREEN wires this up.
type checkboxDelegate struct {
	checked map[int]bool
}

func (d *checkboxDelegate) Height() int  { return 1 }
func (d *checkboxDelegate) Spacing() int { return 0 }

func (d *checkboxDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd {
	return nil
}

func (d *checkboxDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	ai, ok := item.(agentItem)
	if !ok {
		return
	}
	fmt.Fprintf(w, "%s\n", ai.target.DisplayName())
}

// agentPickerModel is the bubbletea Model backing RunAgentPicker.
//
// TODO(RED): confirmed/resolvedTargets are not yet wired to Enter/quit.
type agentPickerModel struct {
	list     list.Model
	delegate *checkboxDelegate
	targets  []agents.AgentTarget
}

func newAgentPickerModel(all []agents.AgentTarget, _ map[agents.TargetID]agents.DetectionResult) agentPickerModel {
	checked := make(map[int]bool, len(all))
	items := make([]list.Item, len(all))
	for i, t := range all {
		items[i] = agentItem{target: t}
	}
	delegate := &checkboxDelegate{checked: checked}
	l := list.New(items, delegate, 0, 0)
	return agentPickerModel{list: l, delegate: delegate, targets: all}
}

func (m agentPickerModel) Init() tea.Cmd { return nil }

func (m agentPickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m agentPickerModel) View() tea.View {
	return tea.NewView(m.list.View())
}

func (m agentPickerModel) resolvedTargets() []agents.AgentTarget {
	return nil
}
