package realcorpus

import "testing"

// TestCorporaProvenanceComplete asserts every manifest entry carries the
// full provenance record the PERF-01 head-to-head benchmark's
// reproducibility depends on (package doc; plan acceptance criterion:
// "every manifest entry has a non-empty CommitSHA, License, and
// SelectionRule field"). Tag is deliberately excluded — it may be empty
// for a pin taken at a non-tagged commit.
func TestCorporaProvenanceComplete(t *testing.T) {
	corpora := Corpora()
	if len(corpora) == 0 {
		t.Fatal("Corpora() returned no entries")
	}

	seen := make(map[string]bool, len(corpora))
	for _, e := range corpora {
		if e.Name == "" {
			t.Errorf("entry has empty Name (SourceURL=%q)", e.SourceURL)
			continue
		}
		if seen[e.Name] {
			t.Errorf("duplicate entry Name %q", e.Name)
		}
		seen[e.Name] = true

		if e.SourceURL == "" {
			t.Errorf("%s: empty SourceURL", e.Name)
		}
		if e.CommitSHA == "" {
			t.Errorf("%s: empty CommitSHA", e.Name)
		}
		if len(e.CommitSHA) != 40 {
			t.Errorf("%s: CommitSHA %q is not a full 40-char SHA", e.Name, e.CommitSHA)
		}
		if e.License == "" {
			t.Errorf("%s: empty License", e.Name)
		}
		if e.SelectionRule == "" {
			t.Errorf("%s: empty SelectionRule", e.Name)
		}
		if len(e.QueryTerms) == 0 {
			t.Errorf("%s: empty QueryTerms — query-latency metric needs a real fixed query set", e.Name)
		}
	}
}

// TestReferencesExistingGoldenCorpora asserts the manifest reuses the
// same pinned commits testdata/golden/README.md's D-06a corpus table
// already records for weft-go and colbymchenry-codegraph, rather than
// introducing a second, drifting pin for the same repos.
func TestReferencesExistingGoldenCorpora(t *testing.T) {
	const (
		weftPinnedCommit = "f89ae3ea4e4c37509f7302fd4e37986212a72079"
		tscgPinnedCommit = "edb9f2f14cd7394a4d31f94ebc871531ef498ab0"
	)

	byName := make(map[string]Entry)
	for _, e := range Corpora() {
		byName[e.Name] = e
	}

	weft, ok := byName["weft-go"]
	if !ok {
		t.Fatal("manifest missing weft-go entry")
	}
	if weft.CommitSHA != weftPinnedCommit {
		t.Errorf("weft-go CommitSHA = %q, want the golden-corpus pin %q", weft.CommitSHA, weftPinnedCommit)
	}

	tscg, ok := byName["colbymchenry-codegraph"]
	if !ok {
		t.Fatal("manifest missing colbymchenry-codegraph entry")
	}
	if tscg.CommitSHA != tscgPinnedCommit {
		t.Errorf("colbymchenry-codegraph CommitSHA = %q, want the golden-corpus pin %q", tscg.CommitSHA, tscgPinnedCommit)
	}
}
