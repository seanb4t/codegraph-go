package query

import (
	"testing"
)

func containsPath(paths []string, target string) bool {
	for _, p := range paths {
		if p == target {
			return true
		}
	}
	return false
}

// --- H17: TestFileRelevanceGate ---

// TestFileRelevanceGate_Clause1GraphMassKeepsFile pins clause (1): a file
// whose graph mass is >= 6% of the max survives the gate even with every
// other clause false.
func TestFileRelevanceGate_Clause1GraphMassKeepsFile(t *testing.T) {
	paths := []string{"hot.go", "warm.go", "cold.go"}
	fileScores := map[string]float64{}
	fileGraphScore := map[string]float64{
		"hot.go":  1000, // the max — trivially >= 6% of itself
		"warm.go": 100,  // 100 >= 1000*0.06 (60) — clause 1
		"cold.go": 1,    // 1 < 60, and fails every other clause
	}

	got := fileRelevanceGate(paths, fileScores, fileGraphScore, nil, nil, nil)

	if !containsPath(got, "hot.go") || !containsPath(got, "warm.go") {
		t.Fatalf("expected hot.go and warm.go kept via clause 1, got %v", got)
	}
	if containsPath(got, "cold.go") {
		t.Errorf("expected cold.go dropped (fails all 5 clauses), got %v", got)
	}
}

// TestFileRelevanceGate_Clause2CentralKeepsFile pins clause (2): a
// central file with near-zero graph mass survives.
func TestFileRelevanceGate_Clause2CentralKeepsFile(t *testing.T) {
	paths := []string{"central.go", "hot.go", "cold.go"}
	fileScores := map[string]float64{}
	fileGraphScore := map[string]float64{
		"central.go": 1,    // far below 6% of max
		"hot.go":     1000, // the max — companion clause-1 pass
		"cold.go":    1,    // fails every clause
	}
	centralFiles := map[string]bool{"central.go": true}

	got := fileRelevanceGate(paths, fileScores, fileGraphScore, centralFiles, nil, nil)

	if !containsPath(got, "central.go") {
		t.Errorf("expected central.go kept via clause 2 (central), got %v", got)
	}
	if containsPath(got, "cold.go") {
		t.Errorf("expected cold.go dropped (fails all 5 clauses), got %v", got)
	}
}

// TestFileRelevanceGate_Clause3EntryNamedKeepsFile pins clause (3): a
// file with an entry/named-file score tier (>= fileScoreEntry) but
// near-zero graph mass survives.
func TestFileRelevanceGate_Clause3EntryNamedKeepsFile(t *testing.T) {
	paths := []string{"entry.go", "hot.go", "cold.go"}
	fileScores := map[string]float64{"entry.go": fileScoreEntry}
	fileGraphScore := map[string]float64{
		"entry.go": 1,
		"hot.go":   1000,
		"cold.go":  1,
	}

	got := fileRelevanceGate(paths, fileScores, fileGraphScore, nil, nil, nil)

	if !containsPath(got, "entry.go") {
		t.Errorf("expected entry.go kept via clause 3 (entry/named), got %v", got)
	}
	if containsPath(got, "cold.go") {
		t.Errorf("expected cold.go dropped (fails all 5 clauses), got %v", got)
	}
}

// TestFileRelevanceGate_Clause4ChangeSurfaceRescuedKeepsFile pins clause
// (4): a change-surface-rescued file survives regardless of mass/score.
func TestFileRelevanceGate_Clause4ChangeSurfaceRescuedKeepsFile(t *testing.T) {
	paths := []string{"rescued.go", "hot.go", "cold.go"}
	fileScores := map[string]float64{}
	fileGraphScore := map[string]float64{
		"rescued.go": 1,
		"hot.go":     1000,
		"cold.go":    1,
	}
	rescuedFiles := map[string]bool{"rescued.go": true}

	got := fileRelevanceGate(paths, fileScores, fileGraphScore, nil, rescuedFiles, nil)

	if !containsPath(got, "rescued.go") {
		t.Errorf("expected rescued.go kept via clause 4 (change-surface-rescued), got %v", got)
	}
	if containsPath(got, "cold.go") {
		t.Errorf("expected cold.go dropped (fails all 5 clauses), got %v", got)
	}
}

// TestFileRelevanceGate_Clause5TermHitsKeepsFile pins clause (5): a file
// with >= 2 distinct term hits but near-zero mass survives — this is the
// case RESEARCH's own §4 illustration calls out (a weakly-connected
// Test*-heavy file must still fail ALL 5 clauses to be dropped).
func TestFileRelevanceGate_Clause5TermHitsKeepsFile(t *testing.T) {
	paths := []string{"termy.go", "hot.go", "cold.go"}
	fileScores := map[string]float64{}
	fileGraphScore := map[string]float64{
		"termy.go": 1,
		"hot.go":   1000,
		"cold.go":  1,
	}
	fileTermHits := map[string]int{"termy.go": 2}

	got := fileRelevanceGate(paths, fileScores, fileGraphScore, nil, nil, fileTermHits)

	if !containsPath(got, "termy.go") {
		t.Errorf("expected termy.go kept via clause 5 (>=2 distinct term hits), got %v", got)
	}
	if containsPath(got, "cold.go") {
		t.Errorf("expected cold.go dropped (fails all 5 clauses), got %v", got)
	}
}

// TestFileRelevanceGate_DropsFileFailingAllClauses pins the negative
// case explicitly: a weakly-connected Test*-heavy file failing every
// clause is dropped when >= 2 OTHER files pass.
func TestFileRelevanceGate_DropsFileFailingAllClauses(t *testing.T) {
	paths := []string{"hot.go", "warm.go", "weak_test.go"}
	fileScores := map[string]float64{}
	fileGraphScore := map[string]float64{
		"hot.go":       1000,
		"warm.go":      100,
		"weak_test.go": 0.5,
	}

	got := fileRelevanceGate(paths, fileScores, fileGraphScore, nil, nil, nil)

	if containsPath(got, "weak_test.go") {
		t.Errorf("expected weak_test.go dropped (fails all 5 clauses), got %v", got)
	}
	if len(got) != 2 {
		t.Errorf("expected exactly 2 files kept, got %v", got)
	}
}

// TestFileRelevanceGate_NeverPrunesBelowTwoFiles pins the "never prunes
// below 2 files" guard: only 1 file passes any clause, so gating is not
// applied at all — the pre-gate set is returned unchanged.
func TestFileRelevanceGate_NeverPrunesBelowTwoFiles(t *testing.T) {
	paths := []string{"hot.go", "cold.go"}
	fileScores := map[string]float64{}
	fileGraphScore := map[string]float64{
		"hot.go":  1000, // the max — trivially passes clause 1
		"cold.go": 1,    // fails every clause
	}

	got := fileRelevanceGate(paths, fileScores, fileGraphScore, nil, nil, nil)

	if len(got) != 2 || !containsPath(got, "hot.go") || !containsPath(got, "cold.go") {
		t.Errorf("expected gate NOT applied (would leave <2 files) — pre-gate set returned unchanged, got %v", got)
	}
}

// TestFileRelevanceGate_SkippedWhenMaxGraphZero pins the maxGraph==0
// guard: the gate must not run at all when every file has zero graph
// mass, regardless of how few files would otherwise pass.
func TestFileRelevanceGate_SkippedWhenMaxGraphZero(t *testing.T) {
	paths := []string{"a.go", "b.go", "c.go"}
	fileScores := map[string]float64{}
	fileGraphScore := map[string]float64{}

	got := fileRelevanceGate(paths, fileScores, fileGraphScore, nil, nil, nil)

	if len(got) != 3 {
		t.Errorf("expected gate skipped entirely (maxGraph==0) — all 3 files returned, got %v", got)
	}
}

// --- H19: TestCentralFileSelection ---

// TestCentralFileSelection_PicksTopTwoByMassWithTermHit pins H19: among
// files with >=1 term hit, the top 2 by graph mass are selected — a
// higher-mass file with ZERO term hits is excluded regardless of mass.
func TestCentralFileSelection_PicksTopTwoByMassWithTermHit(t *testing.T) {
	paths := []string{"top.go", "second.go", "third.go", "fourth.go", "massive_no_terms.go"}
	fileGraphScore := map[string]float64{
		"top.go":              100,
		"second.go":           90,
		"third.go":            80,
		"fourth.go":           70,
		"massive_no_terms.go": 500,
	}
	fileTermHits := map[string]int{
		"top.go":              1,
		"second.go":           1,
		"third.go":            1,
		"fourth.go":           1,
		"massive_no_terms.go": 0,
	}

	got := centralFileSelection(paths, fileGraphScore, fileTermHits)

	if len(got) != 2 {
		t.Fatalf("expected exactly 2 central files, got %v", got)
	}
	if !got["top.go"] || !got["second.go"] {
		t.Errorf("expected top.go and second.go selected (highest mass with a term hit), got %v", got)
	}
	if got["massive_no_terms.go"] {
		t.Errorf("expected massive_no_terms.go excluded (0 term hits despite highest mass), got %v", got)
	}
	if got["third.go"] || got["fourth.go"] {
		t.Errorf("expected third.go/fourth.go excluded by the 2-file cap, got %v", got)
	}
}

// TestCentralFileSelection_ExcludesFilesWithZeroTermHits pins the >=1
// term-hit requirement as a hard filter, not a tie-break.
func TestCentralFileSelection_ExcludesFilesWithZeroTermHits(t *testing.T) {
	paths := []string{"lonely.go"}
	fileGraphScore := map[string]float64{"lonely.go": 1000}
	fileTermHits := map[string]int{"lonely.go": 0}

	got := centralFileSelection(paths, fileGraphScore, fileTermHits)

	if len(got) != 0 {
		t.Errorf("expected no central files (0 term hits), got %v", got)
	}
}

// TestCentralFileSelection_ReturnsFewerThanTwoWhenNotEnoughQualify pins
// the "1-2 files" phrasing: selection is not padded to 2 when fewer
// files qualify.
func TestCentralFileSelection_ReturnsFewerThanTwoWhenNotEnoughQualify(t *testing.T) {
	paths := []string{"only.go", "excluded.go"}
	fileGraphScore := map[string]float64{"only.go": 50, "excluded.go": 999}
	fileTermHits := map[string]int{"only.go": 1, "excluded.go": 0}

	got := centralFileSelection(paths, fileGraphScore, fileTermHits)

	if len(got) != 1 || !got["only.go"] {
		t.Errorf("expected exactly {only.go}, got %v", got)
	}
}

// --- H18: TestFiveTierFileSort ---

// TestFiveTierFileSort_NamedSeedFirst pins tier (1): a named-seed file
// sorts above a merely-corroborated file even when the corroborated file
// has vastly higher graph mass and term hits.
func TestFiveTierFileSort_NamedSeedFirst(t *testing.T) {
	paths := []string{"corroborated.go", "seed.go"}
	fileScores := map[string]float64{
		"seed.go":         fileScoreNamedSeed,
		"corroborated.go": fileScoreEntry,
	}
	fileGraphScore := map[string]float64{
		"seed.go":         1,
		"corroborated.go": 1000,
	}
	fileTermHits := map[string]int{
		"seed.go":         0,
		"corroborated.go": 5,
	}

	got := fiveTierFileSort(paths, fileScores, fileGraphScore, nil, fileTermHits, nil)

	if got[0] != "seed.go" {
		t.Errorf("expected seed.go (named-seed tier) first, got %v", got)
	}
}

// TestFiveTierFileSort_CorroboratedAboveGraphMass pins tier (2):
// corroborated (entry/central AND >=2 terms) outranks a plain
// graph-mass-dominant file that is not entry/central/corroborated.
func TestFiveTierFileSort_CorroboratedAboveGraphMass(t *testing.T) {
	paths := []string{"massive.go", "corroborated.go"}
	fileScores := map[string]float64{
		"corroborated.go": fileScoreEntry,
		"massive.go":      fileScoreOther,
	}
	fileGraphScore := map[string]float64{
		"corroborated.go": 1,
		"massive.go":      1000,
	}
	fileTermHits := map[string]int{
		"corroborated.go": 2,
		"massive.go":      0,
	}

	got := fiveTierFileSort(paths, fileScores, fileGraphScore, nil, fileTermHits, nil)

	if got[0] != "corroborated.go" {
		t.Errorf("expected corroborated.go (tier 2) before massive.go (tier 3 only), got %v", got)
	}
}

// TestFiveTierFileSort_GraphMassEpsilonTie pins tier (3)'s 1%-of-max
// epsilon: two files within epsilon of each other tie on graph mass and
// fall through to term hits (tier 4).
func TestFiveTierFileSort_GraphMassEpsilonTie(t *testing.T) {
	paths := []string{"lowterm.go", "hiterm.go"}
	fileScores := map[string]float64{"lowterm.go": 0, "hiterm.go": 0}
	fileGraphScore := map[string]float64{
		"hiterm.go":  100,   // max
		"lowterm.go": 100.5, // within 1% of 100.5 (epsilon ~1.005), diff 0.5
	}
	fileTermHits := map[string]int{
		"lowterm.go": 1,
		"hiterm.go":  5,
	}

	got := fiveTierFileSort(paths, fileScores, fileGraphScore, nil, fileTermHits, nil)

	if got[0] != "hiterm.go" {
		t.Errorf("expected hiterm.go first (mass tie within epsilon falls to term hits), got %v", got)
	}
}

// TestFiveTierFileSort_GraphMassBeyondEpsilonWins is the epsilon test's
// control: when the mass gap exceeds 1% of max, graph mass (tier 3)
// decides directly, even against a higher term-hit count.
func TestFiveTierFileSort_GraphMassBeyondEpsilonWins(t *testing.T) {
	paths := []string{"low.go", "high.go"}
	fileScores := map[string]float64{"low.go": 0, "high.go": 0}
	fileGraphScore := map[string]float64{
		"high.go": 100, // max
		"low.go":  50,  // well beyond the ~1-unit epsilon
	}
	fileTermHits := map[string]int{
		"low.go":  5,
		"high.go": 1,
	}

	got := fiveTierFileSort(paths, fileScores, fileGraphScore, nil, fileTermHits, nil)

	if got[0] != "high.go" {
		t.Errorf("expected high.go first (mass gap beyond epsilon decides directly), got %v", got)
	}
}

// TestFiveTierFileSort_LowValueSortsAfter pins tier (5): a low-value
// (test) file sorts after an otherwise-equivalent non-low-value file.
func TestFiveTierFileSort_LowValueSortsAfter(t *testing.T) {
	paths := []string{"foo_test.go", "foo.go"}
	fileScores := map[string]float64{"foo_test.go": 0, "foo.go": 0}
	fileGraphScore := map[string]float64{"foo_test.go": 10, "foo.go": 10}
	fileTermHits := map[string]int{"foo_test.go": 0, "foo.go": 0}

	got := fiveTierFileSort(paths, fileScores, fileGraphScore, nil, fileTermHits, nil)

	if got[0] != "foo.go" {
		t.Errorf("expected foo.go (non-low-value) before foo_test.go, got %v", got)
	}
}

// TestFiveTierFileSort_GeneratedFileSortsAfterEquivalent pins the tail's
// !generated key: a generated file sorts after an equivalent
// non-generated file once every earlier tier ties.
func TestFiveTierFileSort_GeneratedFileSortsAfterEquivalent(t *testing.T) {
	paths := []string{"foo.pb.go", "foo.go"}
	fileScores := map[string]float64{"foo.pb.go": 0, "foo.go": 0}
	fileGraphScore := map[string]float64{"foo.pb.go": 10, "foo.go": 10}
	fileTermHits := map[string]int{"foo.pb.go": 0, "foo.go": 0}

	got := fiveTierFileSort(paths, fileScores, fileGraphScore, nil, fileTermHits, nil)

	if got[0] != "foo.go" {
		t.Errorf("expected foo.go (non-generated) before foo.pb.go, got %v", got)
	}
}

// TestFiveTierFileSort_DeterministicTailOnFullTie pins the final
// deterministic tail: when named-seed/corroborated/mass/term-hits/
// low-value/generated all tie, score then node count then path (D-04)
// break the tie, in that order.
func TestFiveTierFileSort_DeterministicTailOnFullTie(t *testing.T) {
	// Score differs: b.go has a higher score, so it must sort first even
	// though its path is lexicographically later.
	paths := []string{"a.go", "b.go"}
	fileScores := map[string]float64{"a.go": 1, "b.go": 3}
	fileGraphScore := map[string]float64{"a.go": 10, "b.go": 10}
	fileTermHits := map[string]int{"a.go": 0, "b.go": 0}

	got := fiveTierFileSort(paths, fileScores, fileGraphScore, nil, fileTermHits, nil)
	if got[0] != "b.go" {
		t.Errorf("expected b.go first (higher score tie-break), got %v", got)
	}

	// Now score AND node count tie too — path (ascending) is the final
	// tie-break, independent of input order.
	paths2 := []string{"z.go", "a.go"}
	fileScores2 := map[string]float64{"z.go": 1, "a.go": 1}
	fileGraphScore2 := map[string]float64{"z.go": 10, "a.go": 10}
	fileTermHits2 := map[string]int{"z.go": 0, "a.go": 0}
	fileNodeCounts2 := map[string]int{"z.go": 4, "a.go": 4}

	got2 := fiveTierFileSort(paths2, fileScores2, fileGraphScore2, nil, fileTermHits2, fileNodeCounts2)
	if got2[0] != "a.go" {
		t.Errorf("expected a.go first (final deterministic path tie-break), got %v", got2)
	}

	// Node count differs, score ties — node count (descending) breaks it
	// before path is ever consulted.
	paths3 := []string{"z.go", "a.go"}
	fileScores3 := map[string]float64{"z.go": 1, "a.go": 1}
	fileGraphScore3 := map[string]float64{"z.go": 10, "a.go": 10}
	fileTermHits3 := map[string]int{"z.go": 0, "a.go": 0}
	fileNodeCounts3 := map[string]int{"z.go": 9, "a.go": 1}

	got3 := fiveTierFileSort(paths3, fileScores3, fileGraphScore3, nil, fileTermHits3, fileNodeCounts3)
	if got3[0] != "z.go" {
		t.Errorf("expected z.go first (higher node count tie-break), got %v", got3)
	}
}

// TestFiveTierFileSort_UsesSliceStable is a light sanity check that the
// sort is deterministic across repeated invocations regardless of input
// slice order (D-04) — guards against an accidental map-iteration
// dependency creeping into a future edit.
func TestFiveTierFileSort_UsesSliceStable(t *testing.T) {
	paths := []string{"c.go", "a.go", "b.go"}
	fileScores := map[string]float64{"a.go": 1, "b.go": 1, "c.go": 1}
	fileGraphScore := map[string]float64{"a.go": 10, "b.go": 10, "c.go": 10}
	fileTermHits := map[string]int{"a.go": 0, "b.go": 0, "c.go": 0}
	fileNodeCounts := map[string]int{"a.go": 1, "b.go": 1, "c.go": 1}

	first := fiveTierFileSort(paths, fileScores, fileGraphScore, nil, fileTermHits, fileNodeCounts)
	shuffled := []string{"b.go", "c.go", "a.go"}
	second := fiveTierFileSort(shuffled, fileScores, fileGraphScore, nil, fileTermHits, fileNodeCounts)

	if len(first) != 3 || len(second) != 3 {
		t.Fatalf("expected 3 files in both results, got %v / %v", first, second)
	}
	for i := range first {
		if first[i] != second[i] {
			t.Errorf("expected deterministic ordering regardless of input order: %v vs %v", first, second)
		}
	}
}
