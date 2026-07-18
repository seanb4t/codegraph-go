package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/seanb4t/codegraph-go/internal/agents"
)

// fakeAgentTarget is a minimal agents.AgentTarget stub for driving
// agentPickerModel without depending on the real registry — same shape as
// internal/agents/registry_test.go's fakeTarget, redefined here since tui
// cannot import internal/agents' unexported test helpers and importing the
// real registry would couple this Model test to whichever real agents
// happen to be registered.
type fakeAgentTarget struct {
	id agents.TargetID
}

func (f fakeAgentTarget) ID() agents.TargetID                   { return f.id }
func (f fakeAgentTarget) DisplayName() string                   { return string(f.id) }
func (f fakeAgentTarget) SupportsLocation(agents.Location) bool { return true }
func (f fakeAgentTarget) Detect(agents.Location) agents.DetectionResult {
	return agents.DetectionResult{}
}
func (f fakeAgentTarget) Install(agents.Location, agents.InstallOptions) agents.WriteResult {
	return agents.WriteResult{}
}
func (f fakeAgentTarget) Uninstall(agents.Location) agents.WriteResult {
	return agents.WriteResult{}
}
func (f fakeAgentTarget) DescribePaths(agents.Location) []string { return nil }

// fakeTargets builds a []agents.AgentTarget from bare ids, in the given
// order — agentPickerModel's index space is this slice's index space.
func fakeTargets(ids ...string) []agents.AgentTarget {
	out := make([]agents.AgentTarget, len(ids))
	for i, id := range ids {
		out[i] = fakeAgentTarget{id: agents.TargetID(id)}
	}
	return out
}

// spaceKeyMsg/enterKeyMsg/quitKeyMsg build synthetic tea.KeyPressMsg values
// the way bubbletea's own real input driver would (Code set to the special
// key constant; Text set only for printable keys) — confirmed against
// charm.land/bubbletea/v2's own key_test.go fixtures (Code: KeySpace,
// Text: " "), not guessed.
func spaceKeyMsg() tea.KeyPressMsg { return tea.KeyPressMsg{Code: tea.KeySpace, Text: " "} }
func enterKeyMsg() tea.KeyPressMsg { return tea.KeyPressMsg{Code: tea.KeyEnter} }
func quitKeyMsg() tea.KeyPressMsg  { return tea.KeyPressMsg{Code: 'q', Text: "q"} }

// TestAgentPickerModel_PreChecksDetectedTargets asserts newAgentPickerModel
// seeds checked[i]=true for exactly the indices whose DetectAll(loc)
// entry reports Installed — the D-14 pre-check contract.
func TestAgentPickerModel_PreChecksDetectedTargets(t *testing.T) {
	all := fakeTargets("a", "b", "c")
	detection := map[agents.TargetID]agents.DetectionResult{
		"b": {Installed: true},
	}

	m := newAgentPickerModel(all, detection)

	if m.delegate.checked[0] {
		t.Errorf("index 0 (a): want unchecked, got checked")
	}
	if !m.delegate.checked[1] {
		t.Errorf("index 1 (b): want pre-checked (DetectAll reports installed), got unchecked")
	}
	if m.delegate.checked[2] {
		t.Errorf("index 2 (c): want unchecked, got checked")
	}
}

// TestAgentPickerModel_SpaceTogglesFocusedRow asserts a space
// tea.KeyPressMsg flips the checked state of the currently-focused row
// (list.Model.Index()), and flips it back on a second press — the
// checkboxDelegate.Update toggle (RESEARCH.md Pitfall 5).
func TestAgentPickerModel_SpaceTogglesFocusedRow(t *testing.T) {
	all := fakeTargets("a", "b")
	m := newAgentPickerModel(all, nil)

	if m.delegate.checked[0] {
		t.Fatalf("precondition: index 0 should start unchecked")
	}

	updated, _ := m.Update(spaceKeyMsg())
	m2, ok := updated.(agentPickerModel)
	if !ok {
		t.Fatalf("Update returned %T, want agentPickerModel", updated)
	}
	if !m2.delegate.checked[0] {
		t.Fatalf("after one space press: want index 0 checked, got unchecked")
	}

	updated2, _ := m2.Update(spaceKeyMsg())
	m3 := updated2.(agentPickerModel)
	if m3.delegate.checked[0] {
		t.Fatalf("after two space presses: want index 0 unchecked again, got checked")
	}
}

// TestAgentPickerModel_EnterResolvesCheckedSetInAscendingOrder asserts
// Enter resolves the checked index set through selectByIndices — dedup +
// ascending index order over the ORIGINAL target slice, regardless of the
// map's own (non-deterministic) iteration order — matching the legacy
// promptAgentMultiSelect's resolution semantics (D-14).
func TestAgentPickerModel_EnterResolvesCheckedSetInAscendingOrder(t *testing.T) {
	all := fakeTargets("a", "b", "c")
	m := newAgentPickerModel(all, nil)
	// Check c (index 2) then a (index 0), out of order, to prove the
	// resolved order is index-ascending, not toggle-order.
	m.delegate.checked[2] = true
	m.delegate.checked[0] = true

	updated, cmd := m.Update(enterKeyMsg())
	m2, ok := updated.(agentPickerModel)
	if !ok {
		t.Fatalf("Update returned %T, want agentPickerModel", updated)
	}
	if cmd == nil {
		t.Fatalf("Enter: want a non-nil Cmd (tea.Quit), got nil")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatalf("Enter: want the Cmd to produce tea.QuitMsg")
	}

	got := m2.resolvedTargets()
	if len(got) != 2 || got[0].ID() != "a" || got[1].ID() != "c" {
		t.Fatalf("resolvedTargets() = %v, want [a, c] in ascending order", got)
	}
}

// TestAgentPickerModel_QuitYieldsEmptySelection asserts q/esc/ctrl+c
// terminate the Program WITHOUT resolving the checked set — quit is
// "no agents", not an error and not whatever was checked at the time.
func TestAgentPickerModel_QuitYieldsEmptySelection(t *testing.T) {
	all := fakeTargets("a", "b")
	m := newAgentPickerModel(all, nil)
	m.delegate.checked[0] = true // prove this gets ignored on quit

	updated, cmd := m.Update(quitKeyMsg())
	m2, ok := updated.(agentPickerModel)
	if !ok {
		t.Fatalf("Update returned %T, want agentPickerModel", updated)
	}
	if cmd == nil {
		t.Fatalf("quit: want a non-nil Cmd (tea.Quit), got nil")
	}

	got := m2.resolvedTargets()
	if len(got) != 0 {
		t.Fatalf("resolvedTargets() after quit = %v, want empty", got)
	}
}
