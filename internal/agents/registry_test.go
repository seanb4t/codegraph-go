package agents

import (
	"fmt"
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

// blockNamesUnshippedCapability is WIRE-02/D-04's honesty checker: the
// marker block is shared across all 4 marker-block agent targets (Claude,
// Codex, opencode, Gemini), but only Claude Code ever receives the
// embedded skill package (Phase 7, AGENT-01, v1-scoped). A block naming a
// skill would send the other 3 targets after a file install never wrote
// for them.
//
// It takes the block string as a parameter and never reads
// codegraphInstructionsBlock from package scope — so it stays
// table-testable against synthetic inputs (Task 2) and so a doc comment
// that discusses this very rule cannot make the gate vacuous by accident:
// the checker inspects the const's VALUE, never this source file's bytes.
func blockNamesUnshippedCapability(block string) error {
	if !strings.Contains(block, "codegraph_explore") {
		return fmt.Errorf("marker block %q never mentions codegraph_explore, so an agent reading it has no pointer to the MCP tool that answers most code questions in one call", block)
	}
	if !strings.Contains(block, "resources/list") {
		return fmt.Errorf("marker block %q never mentions resources/list, so an agent reading it has no pointer to the per-tool reference docs (WIRE-02)", block)
	}
	if strings.Contains(strings.ToLower(block), "skill") {
		return fmt.Errorf("marker block %q names a skill, but 3 of its 4 targets (Codex, opencode, Gemini) never receive one (D-04) — this block is shared across all 4 and must stay skill-agnostic", block)
	}
	return nil
}

// TestInstructionsBlockNamesOnlyShippedCapabilities applies
// blockNamesUnshippedCapability to the real codegraphInstructionsBlock
// const value — the direct proof of ROADMAP success criterion 2. RED
// before the resources/list bullet is added to the block body.
func TestInstructionsBlockNamesOnlyShippedCapabilities(t *testing.T) {
	if err := blockNamesUnshippedCapability(codegraphInstructionsBlock); err != nil {
		t.Fatalf("codegraphInstructionsBlock %v", err)
	}
}

// TestInstructionsBlockGuardIsNotVacuous proves
// blockNamesUnshippedCapability actually discriminates, across all three
// failure classes plus their passing neighbours — including a mixed-case
// skill row, since a case-sensitive check would let "Skill"/"SKILL"
// through. Without this, a checker that returned nil unconditionally
// would pass TestInstructionsBlockNamesOnlyShippedCapabilities forever
// and prove nothing (T-08-06).
func TestInstructionsBlockGuardIsNotVacuous(t *testing.T) {
	cases := []struct {
		name    string
		block   string
		wantErr bool
	}{
		{
			name:    "explore + resources/list, no skill word",
			block:   "Use codegraph_explore first. Call resources/list for more.",
			wantErr: false,
		},
		{
			name:    "same block plus a sentence naming an installed skill file",
			block:   "Use codegraph_explore first. Call resources/list for more. See the installed skill file at .claude/skills/codegraph/SKILL.md.",
			wantErr: true,
		},
		{
			name:    "same block, skill word in mixed case",
			block:   "Use codegraph_explore first. Call resources/list for more. See the installed Skill file.",
			wantErr: true,
		},
		{
			name:    "resources/list present, codegraph_explore missing",
			block:   "Call resources/list for more.",
			wantErr: true,
		},
		{
			name:    "codegraph_explore present, resources/list missing",
			block:   "Use codegraph_explore first.",
			wantErr: true,
		},
		{
			name:    "empty block",
			block:   "",
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := blockNamesUnshippedCapability(tc.block)
			if tc.wantErr && err == nil {
				t.Fatalf("blockNamesUnshippedCapability(%q) = nil, want an error", tc.block)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("blockNamesUnshippedCapability(%q) = %v, want nil", tc.block, err)
			}
		})
	}
}
