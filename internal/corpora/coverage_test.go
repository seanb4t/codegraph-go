package corpora

// The tests in this file DELIBERATELY never call the testing.T skip methods —
// the package has none here, and a coverage guard that skips over a missing
// or malformed document is a guard that passes vacuously, exactly the defect
// class the old network-fetched corpus resolvers both exhibited with their
// `if err != nil { t.FatalAndSkip(...) }` bodies. Those resolvers used skip
// because a missing fetchable corpus must not fail `go test ./...` on an
// offline machine; this guard's documents are COMMITTED, so a missing one
// is a real, loud failure, never a reason to go green.

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/query"
)

// ---------------------------------------------------------------------
// Synthetic-document helpers
// ---------------------------------------------------------------------

// allKinds returns a status["edgesByKind"]-shaped map with every
// query.RankEdges kind set to base.
func allKinds(base int64) map[string]any {
	out := make(map[string]any, len(query.RankEdges))
	for k := range query.RankEdges {
		out[k] = int64(base)
	}
	return out
}

// allLangs returns a status["filesByLanguage"]-shaped map with every
// priority-4 language key (plus the TS/JS constituent keys) set to base.
func allLangs(base int64) map[string]any {
	out := make(map[string]any)
	for _, g := range PriorityLanguages {
		for _, k := range g.Keys {
			out[k] = int64(base)
		}
	}
	return out
}

// mkObs builds an Observation whose stripped status carries the given
// edgesByKind and filesByLanguage submaps (both must be map[string]any
// values because Observation.EdgeCount/LanguageFileCount decode through
// numericSubmap, which type-asserts the section to map[string]any).
func mkObs(repo, sha string, edges, langs map[string]any) Observation {
	return Observation{
		Repo:         repo,
		SHA:          sha,
		License:      "MIT",
		Language:     "go",
		TrackedFiles: 100,
		Status: map[string]any{
			"edgesByKind":    edges,
			"filesByLanguage": langs,
		},
	}
}

const testSHAa = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
const testSHAb = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"

// baselineDocs returns a self-consistent (Manifest, Observations, Selection)
// triple that CheckCoverage passes with zero failures: two locked corpora,
// each supplying a base count for every ranked kind and every priority
// language, with every threshold set well below the derived best. Individual
// tests mutate exactly one fact and assert the resulting failure.
func baselineDocs() (Manifest, Observations, Selection) {
	m := Manifest{Corpora: []Entry{
		{Repo: "acme/go", SHA: testSHAa, License: "MIT", Language: "go", Locked: true},
		{Repo: "acme/py", SHA: testSHAb, License: "MIT", Language: "python", Locked: true},
	}}
	obs, err := NewObservations(1, "corpora/observations.json", []Observation{
		mkObs("acme/go", testSHAa, allKinds(10), allLangs(10)),
		mkObs("acme/py", testSHAb, allKinds(8), allLangs(8)),
	})
	if err != nil {
		panic(err)
	}
	th := make(map[string]int64, len(query.RankEdges))
	for k := range query.RankEdges {
		th[k] = 5 // derived best is 10 for both locked corpora — comfortably above
	}
	sel := Selection{
		MinEdgesPerKind: th,
		LockedSet:       []string{ObservationKey("acme/go", testSHAa), ObservationKey("acme/py", testSHAb)},
	}
	return m, obs, sel
}

// kindsExcept returns an allKinds map with kind set to value and every other
// ranked kind left at base.
func kindsExcept(kind string, value, base int64) map[string]any {
	out := allKinds(base)
	out[kind] = int64(value)
	return out
}

// langsZero returns an allLangs map with every key of group set to zero and
// every other language left at base.
func langsZero(group LanguageGroup, base int64) map[string]any {
	out := allLangs(base)
	for _, k := range group.Keys {
		out[k] = int64(0)
	}
	return out
}

// ---------------------------------------------------------------------
// The real committed documents
// ---------------------------------------------------------------------

// TestCorpusCoverageClaim loads the three REAL committed documents and
// requires CheckCoverage to report zero failures and the two derived counts
// (checked kinds == len(query.RankEdges), checked corpora == the manifest's
// locked-entry count). This is the cheap leg's heart: it needs no fetched
// corpora and no indexing.
func TestCorpusCoverageClaim(t *testing.T) {
	m, err := Load(filepath.Join("..", "..", "corpora", "manifest.json"))
	if err != nil {
		t.Fatalf("Load manifest: %v", err)
	}
	obs, err := LoadObservations(filepath.Join("..", "..", "corpora", "observations.json"))
	if err != nil {
		t.Fatalf("LoadObservations: %v", err)
	}
	sel, err := LoadSelection(filepath.Join("..", "..", "corpora", "selection.json"))
	if err != nil {
		t.Fatalf("LoadSelection: %v", err)
	}

	res := CheckCoverage(m, obs, sel)
	if len(res.Failures) != 0 {
		t.Fatalf("CheckCoverage(committed docs) reported failures:\n%s", strings.Join(res.Failures, "\n"))
	}
	wantKinds := len(query.RankEdges)
	if res.CheckedKinds != wantKinds {
		t.Fatalf("CheckedKinds = %d, want %d (len(query.RankEdges))", res.CheckedKinds, wantKinds)
	}
	wantCorpora := len(LockedEntries(m))
	if res.CheckedCorpora != wantCorpora {
		t.Fatalf("CheckedCorpora = %d, want %d (len(LockedEntries(manifest)))", res.CheckedCorpora, wantCorpora)
	}
	if res.CheckedCorpora == 0 {
		t.Fatalf("CheckedCorpora = 0 — the committed manifest has an empty locked set")
	}
}

// ---------------------------------------------------------------------
// Inclusive boundary
// ---------------------------------------------------------------------

// TestCoverageClaimBoundaryIsInclusive proves a kind whose DERIVED best
// count equals its threshold EXACTLY passes, and reducing that same
// observation's count by one fails naming the kind. The comparison is
// at-least, never strictly-greater.
func TestCoverageClaimBoundaryIsInclusive(t *testing.T) {
	m, obs, sel := baselineDocs()
	sel.MinEdgesPerKind["calls"] = 10 // "calls" now has derived best exactly 10 (acme/go)
	res := CheckCoverage(m, obs, sel)
	if len(res.Failures) != 0 {
		t.Fatalf("kind measuring exactly its threshold must pass, got failures:\n%s", strings.Join(res.Failures, "\n"))
	}

	// Reduce the "calls" count of the ONLY supplier above the boundary.
	key := ObservationKey("acme/go", testSHAa)
	obs.Observations[key] = mkObs("acme/go", testSHAa, kindsExcept("calls", 9, 10), allLangs(10))
	res = CheckCoverage(m, obs, sel)
	if len(res.Failures) == 0 {
		t.Fatalf("kind one below its threshold must fail, got zero failures")
	}
	var found bool
	for _, f := range res.Failures {
		if strings.Contains(f, "calls") && strings.Contains(f, "below threshold") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected a below-threshold failure naming kind 'calls', got:\n%s", strings.Join(res.Failures, "\n"))
	}
}

// ---------------------------------------------------------------------
// Supplier derivation
// ---------------------------------------------------------------------

// TestCoverageDerivesSupplierFromObservations proves CoverageResult's
// per-kind supplier is COMPUTED from the observations, not read from any
// stored field: with two locked observations supplying the same kind at
// different counts the reported supplier is the higher one, and flipping
// the counts flips the reported supplier.
func TestCoverageDerivesSupplierFromObservations(t *testing.T) {
	m, obs, sel := baselineDocs()
	// acme/py now supplies calls at 20, acme/go at 10.
	obs.Observations[ObservationKey("acme/py", testSHAb)] = mkObs("acme/py", testSHAb, kindsExcept("calls", 20, 8), allLangs(8))
	res := CheckCoverage(m, obs, sel)
	if got := res.Kinds["calls"].Repo; got != "acme/py" {
		t.Fatalf("derived supplier for calls = %q, want acme/py (the higher observation)", got)
	}

	// Flip: acme/go now supplies calls at 30, new supplier must be acme/go.
	obs.Observations[ObservationKey("acme/go", testSHAa)] = mkObs("acme/go", testSHAa, kindsExcept("calls", 30, 10), allLangs(10))
	obs.Observations[ObservationKey("acme/py", testSHAb)] = mkObs("acme/py", testSHAb, kindsExcept("calls", 20, 8), allLangs(8))
	res = CheckCoverage(m, obs, sel)
	if got := res.Kinds["calls"].Repo; got != "acme/go" {
		t.Fatalf("derived supplier for calls changed to %q, want acme/go (the higher observation)", got)
	}
}

// TestCoverageRejectsUnlockedSupplier proves an observation whose identity is
// NOT in the locked set cannot supply any kind — if only that observation
// could clear a threshold, the kind fails.
func TestCoverageRejectsUnlockedSupplier(t *testing.T) {
	m, obs, sel := baselineDocs()
	// No locked corpus measures calls (set both to 0), and an unlocked
	// candidate measures calls astronomically high.
	obs.Observations[ObservationKey("acme/go", testSHAa)] = mkObs("acme/go", testSHAa, kindsExcept("calls", 0, 10), allLangs(10))
	obs.Observations[ObservationKey("acme/py", testSHAb)] = mkObs("acme/py", testSHAb, kindsExcept("calls", 0, 8), allLangs(8))
	obs.Observations[ObservationKey("acme/unlocked", "cccccccccccccccccccccccccccccccccccccccc")] = mkObs("acme/unlocked", "cccccccccccccccccccccccccccccccccccccccc", kindsExcept("calls", 10000, 0), allLangs(0))
	sel.MinEdgesPerKind["calls"] = 1 // only the unlocked observation exceeds this

	res := CheckCoverage(m, obs, sel)
	var found bool
	for _, f := range res.Failures {
		if strings.Contains(f, "calls") && strings.Contains(f, "below threshold") {
			found = true
		}
	}
	if !found {
		t.Fatalf("kind supplied only by an UNLOCKED candidate must fail, got:\n%s", strings.Join(res.Failures, "\n"))
	}
}

// TestCoverageRejectsRejectedCandidateAsSupplier proves an observation for a
// candidate named in the selection's rejected ledger cannot supply any kind,
// even with a count high enough to clear every threshold — coverage is
// decided entirely by the locked set.
func TestCoverageRejectsRejectedCandidateAsSupplier(t *testing.T) {
	m, obs, sel := baselineDocs()
	obs.Observations[ObservationKey("acme/go", testSHAa)] = mkObs("acme/go", testSHAa, allKinds(0), allLangs(10))
	obs.Observations[ObservationKey("acme/py", testSHAb)] = mkObs("acme/py", testSHAb, allKinds(0), allLangs(8))
	obs.Observations[ObservationKey("acme/rejected", "dddddddddddddddddddddddddddddddddddddddd")] = mkObs("acme/rejected", "dddddddddddddddddddddddddddddddddddddddd", allKinds(10000), allLangs(10))
	sel.Rejected = []RejectedCandidate{{Repo: "acme/rejected", Reason: "named in the rejected ledger"}}

	res := CheckCoverage(m, obs, sel)
	for _, kind := range []string{"calls", "references", "imports"} {
		var found bool
		for _, f := range res.Failures {
			if strings.Contains(f, kind) && strings.Contains(f, "below threshold") {
				found = true
			}
		}
		if !found {
			t.Fatalf("kind %q supplied only by a REJECTED candidate must fail, got:\n%s", kind, strings.Join(res.Failures, "\n"))
		}
	}
}

// ---------------------------------------------------------------------
// Document reconciliation
// ---------------------------------------------------------------------

// TestCoverageRejectsLockedSetMismatch proves the guard fails on divergence
// in EITHER direction between the manifest's locked entries and the
// selection's locked set.
func TestCoverageRejectsLockedSetMismatch(t *testing.T) {
	t.Run("manifest locks a repo the selection omits", func(t *testing.T) {
		m, obs, sel := baselineDocs()
		sel.LockedSet = []string{ObservationKey("acme/py", testSHAb)} // drop acme/go
		res := CheckCoverage(m, obs, sel)
		var found bool
		for _, f := range res.Failures {
			if strings.Contains(f, "manifest locks") && strings.Contains(f, "acme/go") {
				found = true
			}
		}
		if !found {
			t.Fatalf("expected a manifest-locks-but-selection-omits failure, got:\n%s", strings.Join(res.Failures, "\n"))
		}
	})

	t.Run("selection locks a repo the manifest does not", func(t *testing.T) {
		m, obs, sel := baselineDocs()
		sel.LockedSet = append(sel.LockedSet, ObservationKey("acme/nothing", "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"))
		res := CheckCoverage(m, obs, sel)
		var found bool
		for _, f := range res.Failures {
			if strings.Contains(f, "selection locks") && strings.Contains(f, "acme/nothing") {
				found = true
			}
		}
		if !found {
			t.Fatalf("expected a selection-locks-but-manifest-does-not failure, got:\n%s", strings.Join(res.Failures, "\n"))
		}
	})
}

// TestCoverageRejectsMissingObservation proves a locked identity with no
// observation fails loudly, never skipping.
func TestCoverageRejectsMissingObservation(t *testing.T) {
	m, obs, sel := baselineDocs()
	delete(obs.Observations, ObservationKey("acme/go", testSHAa))
	res := CheckCoverage(m, obs, sel)
	var found bool
	for _, f := range res.Failures {
		if strings.Contains(f, "no observation") && strings.Contains(f, "acme/go") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected a missing-observation failure naming acme/go, got:\n%s", strings.Join(res.Failures, "\n"))
	}
}

// TestCoverageRejectsSyntheticClaim proves a non-empty syntheticKinds fails
// the guard, naming the kinds.
func TestCoverageRejectsSyntheticClaim(t *testing.T) {
	m, obs, sel := baselineDocs()
	sel.SyntheticKinds = []string{"calls", "imports"}
	res := CheckCoverage(m, obs, sel)
	var found bool
	for _, f := range res.Failures {
		if strings.Contains(f, "synthetic coverage") && strings.Contains(f, "calls") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected a synthetic-coverage failure naming kinds, got:\n%s", strings.Join(res.Failures, "\n"))
	}
}

// TestCoverageRejectsThresholdMapEdgeCases proves the selection's threshold
// map must cover every ranked kind and carry no kind outside the ranked set.
func TestCoverageRejectsThresholdMapEdgeCases(t *testing.T) {
	t.Run("omits a ranked kind", func(t *testing.T) {
		m, obs, sel := baselineDocs()
		delete(sel.MinEdgesPerKind, "calls")
		res := CheckCoverage(m, obs, sel)
		var found bool
		for _, f := range res.Failures {
			if strings.Contains(f, "omits a threshold") && strings.Contains(f, "calls") {
				found = true
			}
		}
		if !found {
			t.Fatalf("expected an omitted-threshold failure naming calls, got:\n%s", strings.Join(res.Failures, "\n"))
		}
	})

	t.Run("carries a kind outside the ranked set", func(t *testing.T) {
		m, obs, sel := baselineDocs()
		sel.MinEdgesPerKind["contains"] = 5
		res := CheckCoverage(m, obs, sel)
		var found bool
		for _, f := range res.Failures {
			if strings.Contains(f, "outside the ranked set") && strings.Contains(f, "contains") {
				found = true
			}
		}
		if !found {
			t.Fatalf("expected an outside-ranked-set failure naming contains, got:\n%s", strings.Join(res.Failures, "\n"))
		}
	})
}

// ---------------------------------------------------------------------
// Positive derived counts
// ---------------------------------------------------------------------

// TestCoverageCheckedCountsMatchManifest proves the two positive assertions
// are DERIVED (from query.RankEdges and LockedEntries(manifest)) and that an
// empty locked set fails rather than passing vacuously.
func TestCoverageCheckedCountsMatchManifest(t *testing.T) {
	m, obs, sel := baselineDocs()
	res := CheckCoverage(m, obs, sel)
	if res.CheckedKinds != len(query.RankEdges) {
		t.Fatalf("CheckedKinds = %d, want %d", res.CheckedKinds, len(query.RankEdges))
	}
	if res.CheckedCorpora != len(LockedEntries(m)) {
		t.Fatalf("CheckedCorpora = %d, want %d", res.CheckedCorpora, len(LockedEntries(m)))
	}
	if res.CheckedCorpora == 0 {
		t.Fatalf("CheckedCorpora must be > 0 over a non-empty locked set")
	}

	// Empty locked set: the selection should still reconcile (both sets empty
	// — exact equality holds), but the checked-corpus positive assertion must
	// fail.
	m2 := Manifest{Corpora: []Entry{{Repo: "acme/x", SHA: testSHAa, License: "MIT", Locked: false}}}
	sel2 := sel
	sel2.LockedSet = nil
	res2 := CheckCoverage(m2, obs, sel2)
	if res2.CheckedCorpora != 0 {
		t.Fatalf("empty locked set must yield CheckedCorpora = 0, got %d", res2.CheckedCorpora)
	}
	var found bool
	for _, f := range res2.Failures {
		if strings.Contains(f, "locked corpus set is empty") {
			found = true
		}
	}
	if !found {
		t.Fatalf("an empty locked set must fail, got:\n%s", strings.Join(res2.Failures, "\n"))
	}
}

// TestPriorityLanguagesNonZero proves a priority language summed to zero
// across the locked observations fails, naming the language.
func TestPriorityLanguagesNonZero(t *testing.T) {
	m, obs, sel := baselineDocs()
	// Zero out java (priority language group name "java", key "java") on both
	// locked corpora.
	java := PriorityLanguages[1]
	for _, key := range []string{ObservationKey("acme/go", testSHAa), ObservationKey("acme/py", testSHAb)} {
		o := obs.Observations[key]
		obs.Observations[key] = mkObs(o.Repo, o.SHA, allKinds(10), langsZero(java, 10))
	}
	res := CheckCoverage(m, obs, sel)
	var found bool
	for _, f := range res.Failures {
		if strings.Contains(f, "priority language java") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected a zero-file failure naming java, got:\n%s", strings.Join(res.Failures, "\n"))
	}
}

// TestCoverageGuardFailsOnMissingRecord proves the LOADING layer errors,
// naming the path, for each of the three documents — never a skip.
// CheckCoverage itself performs no I/O; these loaders are its contract's
// other half.
func TestCoverageGuardFailsOnMissingRecord(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "does-not-exist.json")

	if _, err := Load(missing); err == nil {
		t.Fatalf("Load(%s) must error on a missing manifest", missing)
	} else if !strings.Contains(err.Error(), "does-not-exist.json") {
		t.Fatalf("Load error must name the path, got: %v", err)
	}

	if _, err := LoadObservations(missing); err == nil {
		t.Fatalf("LoadObservations(%s) must error on a missing observations file", missing)
	} else if !strings.Contains(err.Error(), "does-not-exist.json") {
		t.Fatalf("LoadObservations error must name the path, got: %v", err)
	}

	if _, err := LoadSelection(missing); err == nil {
		t.Fatalf("LoadSelection(%s) must error on a missing selection file", missing)
	} else if !strings.Contains(err.Error(), "does-not-exist.json") {
		t.Fatalf("LoadSelection error must name the path, got: %v", err)
	}
}