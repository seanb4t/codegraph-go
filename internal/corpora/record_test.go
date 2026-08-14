package corpora

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/query"
)

// ---------------------------------------------------------------------
// StripVolatile
// ---------------------------------------------------------------------

// TestStripVolatileRemovesEveryVolatileKey proves StripVolatile removes
// every named volatile key (both the golden suite's base three and this
// package's four additions), the "_at"/"At" suffix rule, and that a
// volatile key nested inside another object is removed too.
func TestStripVolatileRemovesEveryVolatileKey(t *testing.T) {
	named := []string{
		"score", "lastIndexed", "dbSizeBytes",
		"projectPath", "indexPath", "worktreeMismatch", "stale",
	}
	for _, key := range named {
		t.Run(key, func(t *testing.T) {
			in := map[string]any{key: "volatile-value", "backend": "pebble"}
			out := StripVolatile(in)
			if _, present := out[key]; present {
				t.Fatalf("StripVolatile kept volatile key %q: %+v", key, out)
			}
			if out["backend"] != "pebble" {
				t.Fatalf("StripVolatile dropped a non-volatile key: %+v", out)
			}
		})
	}

	t.Run("suffix _at", func(t *testing.T) {
		in := map[string]any{"indexed_at": "2026-01-01", "backend": "pebble"}
		out := StripVolatile(in)
		if _, present := out["indexed_at"]; present {
			t.Fatalf("StripVolatile kept a _at-suffixed key: %+v", out)
		}
	})

	t.Run("suffix At", func(t *testing.T) {
		in := map[string]any{"updatedAt": "2026-01-01", "backend": "pebble"}
		out := StripVolatile(in)
		if _, present := out["updatedAt"]; present {
			t.Fatalf("StripVolatile kept an At-suffixed key: %+v", out)
		}
	})

	t.Run("nested under another object", func(t *testing.T) {
		in := map[string]any{
			"index": map[string]any{
				"state":       "complete",
				"dbSizeBytes": float64(123),
			},
		}
		out := StripVolatile(in)
		index, ok := out["index"].(map[string]any)
		if !ok {
			t.Fatalf("StripVolatile dropped the index object entirely: %+v", out)
		}
		if _, present := index["dbSizeBytes"]; present {
			t.Fatalf("StripVolatile did not recurse: nested dbSizeBytes survived: %+v", index)
		}
		if index["state"] != "complete" {
			t.Fatalf("StripVolatile dropped a non-volatile nested key: %+v", index)
		}
	})

	t.Run("does not mutate input", func(t *testing.T) {
		in := map[string]any{"stale": true, "backend": "pebble"}
		_ = StripVolatile(in)
		if _, present := in["stale"]; !present {
			t.Fatalf("StripVolatile mutated its input map: %+v", in)
		}
	})
}

// TestStripVolatileKeepsMeasurementKeys proves every measurement key
// StripVolatile is required to preserve survives, with pendingChanges
// explicitly asserted (it is a deterministic placeholder, not volatile).
func TestStripVolatileKeepsMeasurementKeys(t *testing.T) {
	in := map[string]any{
		"edgesByKind":     map[string]any{"calls": float64(5)},
		"filesByLanguage": map[string]any{"go": float64(3)},
		"languages":       []any{"go"},
		"fileCount":       float64(3),
		"nodeCount":       float64(10),
		"edgeCount":       float64(5),
		"backend":         "pebble",
		"version":         "1",
		"index":           map[string]any{"state": "complete"},
		"pendingChanges":  map[string]any{"added": float64(0), "modified": float64(0), "removed": float64(0)},
	}
	out := StripVolatile(in)
	for k := range in {
		if _, present := out[k]; !present {
			t.Fatalf("StripVolatile dropped required measurement key %q: %+v", k, out)
		}
	}
	if _, present := out["pendingChanges"]; !present {
		t.Fatalf("StripVolatile must keep pendingChanges: %+v", out)
	}
}

// ---------------------------------------------------------------------
// Observations
// ---------------------------------------------------------------------

func mkObservation(repo, sha string, trackedFiles int64, edges, files map[string]int64) Observation {
	edgesAny := make(map[string]any, len(edges))
	for k, v := range edges {
		edgesAny[k] = v
	}
	filesAny := make(map[string]any, len(files))
	for k, v := range files {
		filesAny[k] = v
	}
	return Observation{
		Repo:         repo,
		SHA:          sha,
		License:      "MIT",
		Language:     "go",
		TrackedFiles: trackedFiles,
		Status: map[string]any{
			"edgesByKind":     edgesAny,
			"filesByLanguage": filesAny,
			"pendingChanges":  map[string]any{"added": 0, "modified": 0, "removed": 0},
		},
	}
}

// TestObservationsRoundTrip proves an Observation round-trips through
// JSON with its dense edgesByKind intact, including explicit zeros.
func TestObservationsRoundTrip(t *testing.T) {
	obs, err := NewObservations(1, "corpora/manifest.json", []Observation{
		mkObservation("org/repo", validSHA, 42,
			map[string]int64{"calls": 5, "overrides": 0},
			map[string]int64{"go": 42}),
	})
	if err != nil {
		t.Fatalf("NewObservations: %v", err)
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "observations.json")
	writeJSON(t, path, obs)

	loaded, err := LoadObservations(path)
	if err != nil {
		t.Fatalf("LoadObservations: %v", err)
	}
	key := ObservationKey("org/repo", validSHA)
	o, ok := loaded.Observations[key]
	if !ok {
		t.Fatalf("loaded observations missing key %q: %+v", key, loaded.Observations)
	}
	if o.EdgeCount("calls") != 5 {
		t.Fatalf("EdgeCount(calls) = %d, want 5", o.EdgeCount("calls"))
	}
	if o.EdgeCount("overrides") != 0 {
		t.Fatalf("EdgeCount(overrides) = %d, want 0 (explicit zero must survive)", o.EdgeCount("overrides"))
	}
	edges, _ := o.Status["edgesByKind"].(map[string]any)
	if _, present := edges["overrides"]; !present {
		t.Fatalf("explicit-zero overrides key vanished on round trip: %+v", edges)
	}
	if o.TrackedFiles != 42 {
		t.Fatalf("TrackedFiles = %d, want 42", o.TrackedFiles)
	}
}

// TestObservationsRejectDuplicateKey proves NewObservations refuses two
// entries that share an ObservationKey.
func TestObservationsRejectDuplicateKey(t *testing.T) {
	dup := []Observation{
		mkObservation("org/repo", validSHA, 1, nil, nil),
		mkObservation("org/repo", validSHA, 2, nil, nil),
	}
	_, err := NewObservations(1, "corpora/manifest.json", dup)
	if !errors.Is(err, ErrDuplicateObservationKey) {
		t.Fatalf("NewObservations(dup) err = %v, want ErrDuplicateObservationKey", err)
	}
}

// TestLoadObservationsFailsOnMissingFile proves absence of the
// observations file is a loud, path-naming error, never an empty
// document a downstream guard would vacuously pass over.
func TestLoadObservationsFailsOnMissingFile(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "does-not-exist.json")
	_, err := LoadObservations(missing)
	if err == nil {
		t.Fatal("LoadObservations(missing) = nil error, want an error naming the path")
	}
	if !contains(err.Error(), missing) {
		t.Fatalf("LoadObservations error %q does not name the missing path %q", err.Error(), missing)
	}
}

// ---------------------------------------------------------------------
// Selection
// ---------------------------------------------------------------------

func validSelectionDoc() Selection {
	return Selection{
		SchemaVersion:      1,
		MinEdgesPerKind:    map[string]int64{"calls": 3},
		ThresholdRationale: "half the best-observed count, clamped so every kind stays satisfiable",
		LockedSet:          []string{"org/repo@" + validSHA},
		Rejected:           []RejectedCandidate{{Repo: "org/other", Reason: "missed the overrides bar"}},
		SyntheticKinds:     nil,
	}
}

// TestSelectionRoundTrip proves a well-formed Selection decodes and
// validates cleanly, with every field surviving the JSON round trip.
func TestSelectionRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "selection.json")
	writeJSON(t, path, validSelectionDoc())

	s, err := LoadSelection(path)
	if err != nil {
		t.Fatalf("LoadSelection: %v", err)
	}
	if s.ThresholdRationale == "" || s.MinEdgesPerKind["calls"] != 3 || len(s.LockedSet) != 1 || len(s.Rejected) != 1 {
		t.Fatalf("round-tripped selection mismatch: %+v", s)
	}
}

// TestSelectionRequiresThresholdRationale proves an empty (or
// whitespace-only) threshold rationale is rejected at load time (D-15).
func TestSelectionRequiresThresholdRationale(t *testing.T) {
	for name, rationale := range map[string]string{"empty": "", "whitespace": "   "} {
		t.Run(name, func(t *testing.T) {
			doc := validSelectionDoc()
			doc.ThresholdRationale = rationale
			dir := t.TempDir()
			path := filepath.Join(dir, "selection.json")
			writeJSON(t, path, doc)

			if _, err := LoadSelection(path); err == nil {
				t.Fatal("LoadSelection accepted an empty threshold rationale, want an error (D-15)")
			}
		})
	}
}

// TestSelectionRejectsUnreasonedRejection proves a rejected-ledger entry
// with an empty reason is rejected at load time.
func TestSelectionRejectsUnreasonedRejection(t *testing.T) {
	doc := validSelectionDoc()
	doc.Rejected = []RejectedCandidate{{Repo: "org/other", Reason: ""}}
	dir := t.TempDir()
	path := filepath.Join(dir, "selection.json")
	writeJSON(t, path, doc)

	if _, err := LoadSelection(path); err == nil {
		t.Fatal("LoadSelection accepted a rejected entry with an empty reason, want an error")
	}
}

// ---------------------------------------------------------------------
// ComputeThresholds
// ---------------------------------------------------------------------

// TestComputeThresholdsUsesAllEligibleNotLocked proves the derived
// threshold tracks the best count across the FULL eligible slice handed
// to it, not some other narrower notion — shrinking the eligible slice
// changes the result.
func TestComputeThresholdsUsesAllEligibleNotLocked(t *testing.T) {
	obs, err := NewObservations(1, "m.json", []Observation{
		mkObservation("a/repo", validSHA, 1, map[string]int64{"calls": 10}, nil),
		mkObservation("b/repo", validSHA, 1, map[string]int64{"calls": 2}, nil),
	})
	if err != nil {
		t.Fatalf("NewObservations: %v", err)
	}
	all := []string{ObservationKey("a/repo", validSHA), ObservationKey("b/repo", validSHA)}
	thAll := ComputeThresholds(obs, all)
	if thAll["calls"] != 5 { // clamp(max(2, 10/2), 10) = 5
		t.Fatalf("thAll[calls] = %d, want 5 (derived from the full eligible slice's best=10)", thAll["calls"])
	}

	narrow := []string{ObservationKey("b/repo", validSHA)}
	thNarrow := ComputeThresholds(obs, narrow)
	if thNarrow["calls"] != 2 { // clamp(max(2, 2/2), 2) = 2
		t.Fatalf("thNarrow[calls] = %d, want 2 (derived from only b/repo's best=2)", thNarrow["calls"])
	}
}

// TestComputeThresholdsClampsToBestSoEveryThresholdIsSatisfiable proves
// every derived threshold is <= that kind's best observed count,
// including the best==1 boundary case, and that best==0 yields 0.
func TestComputeThresholdsClampsToBestSoEveryThresholdIsSatisfiable(t *testing.T) {
	cases := []struct {
		name string
		best int64
		want int64
	}{
		{"best zero", 0, 0},
		{"best one", 1, 1},
		{"best ten", 10, 5},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			obs, err := NewObservations(1, "m.json", []Observation{
				mkObservation("a/repo", validSHA, 1, map[string]int64{"calls": c.best}, nil),
			})
			if err != nil {
				t.Fatalf("NewObservations: %v", err)
			}
			eligible := []string{ObservationKey("a/repo", validSHA)}
			th := ComputeThresholds(obs, eligible)
			if th["calls"] != c.want {
				t.Fatalf("threshold(best=%d) = %d, want %d", c.best, th["calls"], c.want)
			}
			if th["calls"] > c.best {
				t.Fatalf("threshold %d exceeds best %d — unsatisfiable by construction", th["calls"], c.best)
			}
		})
	}

	t.Run("every RankEdges kind is present", func(t *testing.T) {
		obs, _ := NewObservations(1, "m.json", []Observation{
			mkObservation("a/repo", validSHA, 1, nil, nil),
		})
		th := ComputeThresholds(obs, []string{ObservationKey("a/repo", validSHA)})
		if len(th) != len(query.RankEdges) {
			t.Fatalf("len(th) = %d, want %d (one entry per query.RankEdges kind)", len(th), len(query.RankEdges))
		}
		for kind := range query.RankEdges {
			if _, present := th[kind]; !present {
				t.Fatalf("threshold map missing RankEdges member %q", kind)
			}
		}
	})
}

// ---------------------------------------------------------------------
// SelectLockedSet
// ---------------------------------------------------------------------

// allLanguages is a filesByLanguage map covering every PriorityLanguages
// group with a single file each, for tests that need one observation to
// independently satisfy the language bar.
func allLanguages(n int64) map[string]int64 {
	return map[string]int64{
		"go": n, "java": n, "csharp": n, "python": n, "typescript": n,
	}
}

// TestSelectLockedSetIsMinimumCardinality proves SelectLockedSet returns
// the smallest satisfying subset even when a larger, cheaper-by-total-
// tracked-files combination also satisfies — cardinality is the primary
// sort key, not total tracked files.
func TestSelectLockedSetIsMinimumCardinality(t *testing.T) {
	solo := mkObservation("solo/repo", validSHA, 1000, map[string]int64{"calls": 5}, allLanguages(1))
	pairA := mkObservation("pair/a", validSHA, 1, map[string]int64{"calls": 5}, map[string]int64{"go": 1})
	pairB := mkObservation("pair/b", validSHA, 1, nil, map[string]int64{"java": 1, "csharp": 1, "python": 1, "typescript": 1})

	obs, err := NewObservations(1, "m.json", []Observation{solo, pairA, pairB})
	if err != nil {
		t.Fatalf("NewObservations: %v", err)
	}
	eligible := []string{
		ObservationKey("solo/repo", validSHA),
		ObservationKey("pair/a", validSHA),
		ObservationKey("pair/b", validSHA),
	}
	th := map[string]int64{"calls": 3}

	got, err := SelectLockedSet(obs, th, eligible)
	if err != nil {
		t.Fatalf("SelectLockedSet: %v", err)
	}
	want := []string{ObservationKey("solo/repo", validSHA)}
	if len(got) != 1 || got[0] != want[0] {
		t.Fatalf("SelectLockedSet = %v, want %v (the size-1 solution, despite pair/a+pair/b costing fewer total tracked files)", got, want)
	}
}

// TestSelectLockedSetTieBreakIsDeterministic proves that among equally
// satisfying, equally-sized, equal-total-tracked-files subsets, the
// sorted-repository-name lexicographic tie-break decides the winner.
func TestSelectLockedSetTieBreakIsDeterministic(t *testing.T) {
	first := mkObservation("a/first", validSHA, 5, map[string]int64{"calls": 5}, allLanguages(1))
	second := mkObservation("b/second", validSHA, 5, map[string]int64{"calls": 5}, allLanguages(1))

	obs, err := NewObservations(1, "m.json", []Observation{first, second})
	if err != nil {
		t.Fatalf("NewObservations: %v", err)
	}
	eligible := []string{ObservationKey("a/first", validSHA), ObservationKey("b/second", validSHA)}
	th := map[string]int64{"calls": 3}

	got, err := SelectLockedSet(obs, th, eligible)
	if err != nil {
		t.Fatalf("SelectLockedSet: %v", err)
	}
	want := []string{ObservationKey("a/first", validSHA)}
	if len(got) != 1 || got[0] != want[0] {
		t.Fatalf("SelectLockedSet = %v, want %v (lexicographically smaller repo name wins an equal-size, equal-cost tie)", got, want)
	}

	got2, err := SelectLockedSet(obs, th, eligible)
	if err != nil {
		t.Fatalf("SelectLockedSet (rerun): %v", err)
	}
	if got2[0] != got[0] {
		t.Fatalf("SelectLockedSet is non-deterministic across runs: %v then %v", got, got2)
	}
}

// TestSelectLockedSetErrorsOnUnsatisfiableKind proves SelectLockedSet
// returns ErrNoQualifyingSubset, naming the unsatisfiable kind, when no
// subset of eligible can clear a threshold.
func TestSelectLockedSetErrorsOnUnsatisfiableKind(t *testing.T) {
	solo := mkObservation("solo/repo", validSHA, 10, map[string]int64{"calls": 5}, allLanguages(1))
	obs, err := NewObservations(1, "m.json", []Observation{solo})
	if err != nil {
		t.Fatalf("NewObservations: %v", err)
	}
	eligible := []string{ObservationKey("solo/repo", validSHA)}
	th := map[string]int64{"calls": 100} // unreachable: best observed is 5

	_, err = SelectLockedSet(obs, th, eligible)
	if !errors.Is(err, ErrNoQualifyingSubset) {
		t.Fatalf("SelectLockedSet err = %v, want ErrNoQualifyingSubset", err)
	}
	if !contains(err.Error(), "calls") {
		t.Fatalf("SelectLockedSet error %q does not name the unsatisfiable kind %q", err.Error(), "calls")
	}
}

// ---------------------------------------------------------------------
// test helpers
// ---------------------------------------------------------------------

func writeJSON(t *testing.T, path string, v any) {
	t.Helper()
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write fixture %s: %v", path, err)
	}
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
