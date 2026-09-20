package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"

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
func (f fakeAgentTarget) Capabilities() agents.Capabilities      { return agents.Capabilities{} }

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

// TestAgentPickerModel_PreChecksCodexAtLocal (D-09, D-26) asserts the
// picker's pre-check detects Codex through the local .codex/ directory,
// using the REAL registry (agents.AllTargets(), agents.DetectAll) in a
// scratch cwd — the picker still lists all 8 targets, and the Codex row
// starts checked.
func TestAgentPickerModel_PreChecksCodexAtLocal(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	if err := os.MkdirAll(filepath.Join(dir, ".codex"), 0o755); err != nil {
		t.Fatalf("mkdir .codex: %v", err)
	}

	all := agents.AllTargets()
	if len(all) != 8 {
		t.Fatalf("agents.AllTargets() returned %d targets, want 8 (D-26)", len(all))
	}
	detection := agents.DetectAll(agents.LocationLocal)

	m := newAgentPickerModel(all, detection)

	codexChecked := false
	for i, target := range all {
		if target.ID() == agents.Codex {
			codexChecked = m.delegate.checked[i]
		}
	}
	if !codexChecked {
		t.Fatalf("expected the Codex row to start checked given a scratch .codex/ dir, checked=%v", m.delegate.checked)
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

// sizeTo sends a tea.WindowSizeMsg{Width, Height} through Update and returns
// the resulting agentPickerModel — a shared helper for the footprint tests
// below, all of which need the list sized before calling View().
func sizeTo(t *testing.T, m agentPickerModel, width, height int) agentPickerModel {
	t.Helper()
	updated, _ := m.Update(tea.WindowSizeMsg{Width: width, Height: height})
	m2, ok := updated.(agentPickerModel)
	if !ok {
		t.Fatalf("Update(WindowSizeMsg) returned %T, want agentPickerModel", updated)
	}
	return m2
}

// viewContentNoPanic calls m.View() under a recover(), failing the test with
// a clear message instead of crashing the test binary if View() panics —
// needed for the height=1/height=2 boundary subtests below (FIX-03
// precision) where the list height floors at 1 and the footer's own
// help string is taller than the window.
func viewContentNoPanic(t *testing.T, m agentPickerModel) string {
	t.Helper()
	var content string
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("View() panicked: %v", r)
			}
		}()
		content = m.View().Content
	}()
	return content
}

// TestAgentPickerFootprintFitsDefaultPane is D-24/D-25's model-level
// footprint guard: at the default 100x30 pane, with the real registry's 8
// targets, the rendered view must fit within 30 lines, show the help
// footer, and list every target's display name as 8 consecutive rows in
// agents.AllTargets() order (D-26 — the picker keeps all 8 rows). Before
// the delegate fix (each row costing 2 lines from the delegate's own
// trailing newline plus bubbles/v2/list's populatedView separator), the
// real height is 35, this test is RED, and the footer never appears.
func TestAgentPickerFootprintFitsDefaultPane(t *testing.T) {
	all := agents.AllTargets()
	if len(all) != 8 {
		t.Fatalf("agents.AllTargets() returned %d targets, want 8 (D-26)", len(all))
	}

	m := sizeTo(t, newAgentPickerModel(all, nil), 100, 30)
	content := viewContentNoPanic(t, m)

	if h := lipgloss.Height(content); h > 30 {
		t.Fatalf("lipgloss.Height(view.Content) = %d, want <= 30 at 100x30 with 8 targets:\n%s", h, content)
	}
	if !strings.Contains(content, "space: toggle") {
		t.Fatalf("view content missing footer text \"space: toggle\":\n%s", content)
	}

	lines := strings.Split(content, "\n")
	rowIdx := make([]int, 0, len(all))
	searchFrom := 0
	for _, target := range all {
		idx := -1
		for j := searchFrom; j < len(lines); j++ {
			if strings.Contains(lines[j], target.DisplayName()) {
				idx = j
				break
			}
		}
		if idx == -1 {
			t.Fatalf("view content missing display name %q in AllTargets() order after line %d:\n%s", target.DisplayName(), searchFrom, content)
		}
		rowIdx = append(rowIdx, idx)
		searchFrom = idx + 1
	}
	for i := 1; i < len(rowIdx); i++ {
		if rowIdx[i] != rowIdx[i-1]+1 {
			t.Fatalf("the 8 target rows are not consecutive lines (row line indices = %v):\n%s", rowIdx, content)
		}
	}
}

// TestAgentPickerFootprint_HeightBoundaries covers FIX-03's explicit height
// boundaries: h29/h30/h31 must each fit within that many lines and still
// show the footer (the boundary case), and h1/h2 must never panic and must
// floor the list height at 1 rather than going to zero or negative (the
// precision case) — View() need not fit the footer at those extremes.
func TestAgentPickerFootprint_HeightBoundaries(t *testing.T) {
	all := agents.AllTargets()
	if len(all) != 8 {
		t.Fatalf("agents.AllTargets() returned %d targets, want 8 (D-26)", len(all))
	}

	boundaries := []struct {
		name        string
		height      int
		checkFooter bool
	}{
		{"h29", 29, true},
		{"h30", 30, true},
		{"h31", 31, true},
		{"h1", 1, false},
		{"h2", 2, false},
	}

	for _, b := range boundaries {
		t.Run(b.name, func(t *testing.T) {
			m := sizeTo(t, newAgentPickerModel(all, nil), 100, b.height)
			content := viewContentNoPanic(t, m)

			if content == "" {
				t.Fatalf("View() at height %d returned empty content", b.height)
			}
			if b.checkFooter {
				if h := lipgloss.Height(content); h > b.height {
					t.Fatalf("lipgloss.Height(view.Content) = %d, want <= %d at height %d:\n%s", h, b.height, b.height, content)
				}
				if !strings.Contains(content, "space: toggle") {
					t.Fatalf("view content at height %d missing footer text \"space: toggle\":\n%s", b.height, content)
				}
			}
		})
	}
}

// TestAgentPickerFootprint_EmptyRoster asserts an empty roster (D-26's
// theoretical zero-target edge — never hit by the real registry, but a
// picker construction the code must not panic on) still renders within the
// default 100x30 pane and still shows the footer.
func TestAgentPickerFootprint_EmptyRoster(t *testing.T) {
	m := sizeTo(t, newAgentPickerModel(nil, nil), 100, 30)
	content := viewContentNoPanic(t, m)

	if h := lipgloss.Height(content); h > 30 {
		t.Fatalf("lipgloss.Height(view.Content) = %d, want <= 30 on empty roster:\n%s", h, content)
	}
	if !strings.Contains(content, "space: toggle") {
		t.Fatalf("view content missing footer text \"space: toggle\" on empty roster:\n%s", content)
	}
}
