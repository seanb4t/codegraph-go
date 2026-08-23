package query_test

import (
	"testing"

	"github.com/seanb4t/codegraph-go/internal/query"
)

// TestExploreResultComponentTypesAreExported proves ExploreFileGroup and
// ExploreBlast are nameable from OUTSIDE internal/query — an in-package
// test would prove nothing about exportedness, since every identifier
// (exported or not) is visible from inside its own declaring package. This
// file compiles only if both types are exported (D-02/ENG-02): a consumer
// outside internal/query (internal/uiserver, from Phase 3 onward) must be
// able to declare a mapper signature over these types and construct one in
// its own tests, not merely read field values off an instance obtained
// elsewhere.
//
// Lives in its own file (detail_external_test.go) rather than inside
// detail_test.go: Go requires one package per file, and detail_test.go's
// other tests are in-package (package query) by necessity — several
// exercise unexported helpers (exploreZeroResult, groupMatchesByFile,
// fileRelevanceGate, fiveTierFileSort, DefaultExploreBFSBounds) that an
// external test package cannot reach.
func TestExploreResultComponentTypesAreExported(t *testing.T) {
	var groups []query.ExploreFileGroup
	var blasts []query.ExploreBlast

	groups = append(groups, query.ExploreFileGroup{Path: "pkg/foo.go"})
	blasts = append(blasts, query.ExploreBlast{CallerCount: 1})

	if len(groups) != 1 {
		t.Fatalf("[]query.ExploreFileGroup: got len %d, want 1", len(groups))
	}
	if len(blasts) != 1 {
		t.Fatalf("[]query.ExploreBlast: got len %d, want 1", len(blasts))
	}
}
