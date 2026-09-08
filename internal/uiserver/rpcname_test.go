package uiserver

import (
	"strings"
	"testing"
)

// wantRPCName is the fourteenth UIService rpc's chosen name, verified
// CLEAN this session against the live mutatingVerbs fixture declared in
// readonly_test.go (never re-transcribed here — this test reads that
// same package-level var). WatchIndex, IndexEvents and StreamIndex are
// all BANNED: each contains "Index", one of mutatingVerbs' 19 members,
// the exact trap that forced Phase 4 to abandon GetIndexHealth. LiveUpdates
// is also banned (contains "Update").
const wantRPCName = "WatchGraph"

// TestRPCNameIsCleanAgainstMutatingVerbs asserts the chosen fourteenth
// rpc name contains none of mutatingVerbs' members, and pairs that
// negative assertion with a POSITIVE CONTROL — GetIndexHealth, the name
// Phase 4 had to abandon, still collides on "Index" — so this test
// cannot pass by mutatingVerbs having gone empty or by the containment
// check being broken (rule 84d1gfpywd: an assertion that cannot fail
// proves nothing).
func TestRPCNameIsCleanAgainstMutatingVerbs(t *testing.T) {
	if len(mutatingVerbs) == 0 {
		t.Fatal("mutatingVerbs is empty — this guard would pass vacuously")
	}

	foundReindex := false
	for _, verb := range mutatingVerbs {
		if verb == "Reindex" {
			foundReindex = true
			break
		}
	}
	if !foundReindex {
		t.Fatalf("mutatingVerbs no longer contains %q — the fixture was emptied or edited; this test forbids that (rule 84d1gfpywd)", "Reindex")
	}
	t.Logf("mutatingVerbs has %d members and still includes Reindex", len(mutatingVerbs))

	// Positive control: a KNOWN-colliding name must still collide, or the
	// containment check itself is broken and every other assertion in
	// this test proves nothing.
	const knownColliding = "GetIndexHealth"
	controlCollided := false
	for _, verb := range mutatingVerbs {
		if strings.Contains(knownColliding, verb) {
			controlCollided = true
			break
		}
	}
	if !controlCollided {
		t.Fatalf("positive control failed: %q no longer collides with any mutatingVerbs member — the discriminator is broken", knownColliding)
	}

	// The actual assertion: the chosen name is clean.
	for _, verb := range mutatingVerbs {
		if strings.Contains(wantRPCName, verb) {
			t.Fatalf("chosen rpc name %q contains mutating verb %q — pick a different name", wantRPCName, verb)
		}
	}
}
