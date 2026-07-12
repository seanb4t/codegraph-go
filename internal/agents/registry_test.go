package agents

import (
	"reflect"
	"strings"
	"testing"
)

// fakeTarget is a minimal AgentTarget stub used to exercise the registry
// without depending on any real per-agent implementation (those land in
// 06-02/06-03).
type fakeTarget struct {
	id        TargetID
	installed bool
}

func (f fakeTarget) ID() TargetID                   { return f.id }
func (f fakeTarget) DisplayName() string            { return string(f.id) }
func (f fakeTarget) SupportsLocation(Location) bool { return true }
func (f fakeTarget) Detect(Location) DetectionResult {
	return DetectionResult{Installed: f.installed}
}
func (f fakeTarget) Install(Location, InstallOptions) WriteResult { return WriteResult{} }
func (f fakeTarget) Uninstall(Location) WriteResult               { return WriteResult{} }
func (f fakeTarget) DescribePaths(Location) []string              { return nil }

// resetRegistryForTest swaps in a fresh, empty registry for the duration
// of one test, restoring the previous global registry on cleanup — keeps
// each test's fake registrations from leaking into its siblings (or into
// future per-agent test files that self-register real targets).
func resetRegistryForTest(t *testing.T) {
	t.Helper()
	saved := registry
	registry = map[TargetID]AgentTarget{}
	t.Cleanup(func() { registry = saved })
}

func TestRegistry_RegisterAndGet(t *testing.T) {
	resetRegistryForTest(t)
	want := fakeTarget{id: TargetID("fake-a")}
	registerTarget(want)

	got, ok := GetTarget(TargetID("fake-a"))
	if !ok {
		t.Fatalf("GetTarget: not found")
	}
	if got.ID() != want.ID() {
		t.Fatalf("GetTarget: got %v, want %v", got, want)
	}
}

func TestRegistry_GetUnknownReturnsFalse(t *testing.T) {
	resetRegistryForTest(t)

	_, ok := GetTarget(TargetID("does-not-exist"))
	if ok {
		t.Fatalf("GetTarget: want not-found for an unregistered id")
	}
}

func TestRegistry_AllTargetIDsSorted(t *testing.T) {
	resetRegistryForTest(t)
	registerTarget(fakeTarget{id: TargetID("zebra")})
	registerTarget(fakeTarget{id: TargetID("alpha")})
	registerTarget(fakeTarget{id: TargetID("mid")})

	got := AllTargetIDs()
	want := []TargetID{"alpha", "mid", "zebra"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("AllTargetIDs() = %v, want %v (sorted ascending)", got, want)
	}
}

func TestResolveTargetFlag_All(t *testing.T) {
	resetRegistryForTest(t)
	registerTarget(fakeTarget{id: TargetID("a")})
	registerTarget(fakeTarget{id: TargetID("b")})

	got, err := ResolveTargetFlag("all", LocationGlobal)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("want 2 targets, got %d: %v", len(got), got)
	}
}

func TestResolveTargetFlag_None(t *testing.T) {
	resetRegistryForTest(t)
	registerTarget(fakeTarget{id: TargetID("a")})

	got, err := ResolveTargetFlag("none", LocationGlobal)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("want 0 targets, got %d: %v", len(got), got)
	}
}

func TestResolveTargetFlag_CSVSelectsExactlyListedTargets(t *testing.T) {
	resetRegistryForTest(t)
	registerTarget(fakeTarget{id: TargetID("claude")})
	registerTarget(fakeTarget{id: TargetID("cursor")})
	registerTarget(fakeTarget{id: TargetID("codex")})

	got, err := ResolveTargetFlag("claude,cursor", LocationGlobal)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("want 2 targets, got %d: %v", len(got), got)
	}
	ids := map[TargetID]bool{}
	for _, t := range got {
		ids[t.ID()] = true
	}
	if !ids["claude"] || !ids["cursor"] {
		t.Fatalf("want exactly claude+cursor, got %v", got)
	}
}

func TestResolveTargetFlag_CSVUnknownIDErrors(t *testing.T) {
	resetRegistryForTest(t)
	registerTarget(fakeTarget{id: TargetID("claude")})

	_, err := ResolveTargetFlag("claude,nonexistent", LocationGlobal)
	if err == nil {
		t.Fatalf("want an error for an unknown target id in the csv")
	}
}

func TestResolveTargetFlag_AutoReturnsOnlyDetectedTargets(t *testing.T) {
	resetRegistryForTest(t)
	registerTarget(fakeTarget{id: TargetID("claude"), installed: true})
	registerTarget(fakeTarget{id: TargetID("cursor"), installed: false})

	got, err := ResolveTargetFlag("auto", LocationGlobal)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].ID() != TargetID("claude") {
		t.Fatalf("want only the detected [claude] target, got %v", got)
	}
}

func TestResolveTargetFlag_AutoFallsBackToClaudeWhenNoneDetected(t *testing.T) {
	resetRegistryForTest(t)
	registerTarget(fakeTarget{id: Claude, installed: false})
	registerTarget(fakeTarget{id: TargetID("cursor"), installed: false})

	got, err := ResolveTargetFlag("auto", LocationGlobal)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].ID() != Claude {
		t.Fatalf("want fallback to just [claude], got %v", got)
	}
}

func TestInstructionsBlock_HasMarkersAndCodegraphExploreReference(t *testing.T) {
	if !strings.HasPrefix(codegraphInstructionsBlock, codegraphSectionStart) {
		t.Fatalf("codegraphInstructionsBlock does not start with %q", codegraphSectionStart)
	}
	if !strings.HasSuffix(codegraphInstructionsBlock, codegraphSectionEnd) {
		t.Fatalf("codegraphInstructionsBlock does not end with %q", codegraphSectionEnd)
	}
	inner := strings.TrimSuffix(
		strings.TrimPrefix(codegraphInstructionsBlock, codegraphSectionStart),
		codegraphSectionEnd,
	)
	if strings.TrimSpace(inner) == "" {
		t.Fatalf("codegraphInstructionsBlock has no content between the markers")
	}
	if !strings.Contains(codegraphInstructionsBlock, "codegraph_explore") {
		t.Fatalf("codegraphInstructionsBlock does not reference codegraph_explore")
	}
}

func TestInstructionsBlock_ExactMarkerText(t *testing.T) {
	// D-01a: hard cross-implementation parity contract — do not deviate
	// from these exact strings.
	if codegraphSectionStart != "<!-- CODEGRAPH_START -->" {
		t.Fatalf("codegraphSectionStart = %q, want the exact TS marker text", codegraphSectionStart)
	}
	if codegraphSectionEnd != "<!-- CODEGRAPH_END -->" {
		t.Fatalf("codegraphSectionEnd = %q, want the exact TS marker text", codegraphSectionEnd)
	}
}
