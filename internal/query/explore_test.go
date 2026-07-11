package query

import "testing"

// TestExploreRejectsEmptyQuery pins WR-05 for Explore: an empty or
// whitespace-only query must be rejected, mirroring Query/Search, rather
// than falling through to matchNodes' degenerate "matches everything"
// case.
func TestExploreRejectsEmptyQuery(t *testing.T) {
	dir := copyFixture(t)
	indexFixture(t, dir)

	engine, closer, err := OpenAt(dir)
	if err != nil {
		t.Fatalf("OpenAt: unexpected error: %v", err)
	}
	defer closer.Close()

	if _, err := engine.Explore("", 5); err == nil {
		t.Fatal("Explore: expected error for an empty query, got nil")
	}
	if _, err := engine.Explore("   ", 5); err == nil {
		t.Fatal("Explore: expected error for a whitespace-only query, got nil")
	}
}
