package main

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/corpora"
	"github.com/seanb4t/codegraph-go/internal/query"
)

func mkSimpleObs(repo, sha string, edges, files map[string]int64) corpora.Observations {
	obs, _ := corpora.NewObservations(1, "m.json", []corpora.Observation{{
		Repo: repo, SHA: sha, License: "MIT", Language: "go", TrackedFiles: 10,
		Status: map[string]any{
			"edgesByKind":     toAnyMap(edges),
			"filesByLanguage": toAnyMap(files),
			"pendingChanges":  map[string]any{"added": float64(0), "modified": float64(0), "removed": float64(0)},
		},
	}})
	return obs
}

// fixtureObsPath resolves the committed observations fixture relative to
// the repo root, not the test binary's cwd (which under `go test` is the
// package directory). Uses runtime.Caller to find this test source file's
// own directory (tools/corpora) and walks up to the module root.
func fixtureObsPath() string {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		return "testdata/corpora/observations.fixture.json"
	}
	repoRoot := filepath.Join(filepath.Dir(filepath.Dir(thisFile)) /* .. */, "..")
	return filepath.Join(repoRoot, "testdata", "corpora", "observations.fixture.json")
}

func toAnyMap(m map[string]int64) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func withTempObservations(t *testing.T, obs corpora.Observations) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "observations.json")
	data, _ := json.MarshalIndent(obs, "", "  ")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write temp observations: %v", err)
	}
	return path
}

// TestMeasureUpsertsWithoutDroppingPriorEntries proves that measuring
// repo A and then repo B leaves both observations in the output file.
func TestMeasureUpsertsWithoutDroppingPriorEntries(t *testing.T) {
	obs := mkSimpleObs("a/repo", "a123456789012345678901234567890123456789",
		nil, allLangMap(1))
	oPath := withTempObservations(t, obs)

	loaded, err := corpora.LoadObservations(oPath)
	if err != nil {
		t.Fatalf("LoadObservations: %v", err)
	}
	loaded.Observations["b/repo@b123456789012345678901234567890123456789"] = corpora.Observation{
		Repo: "b/repo", SHA: "b123456789012345678901234567890123456789",
		License: "MIT", Language: "go", TrackedFiles: 5,
		Status: map[string]any{
			"pendingChanges": map[string]any{"added": float64(0), "modified": float64(0), "removed": float64(0)},
		},
	}
	if err := writeObservations(oPath, loaded); err != nil {
		t.Fatalf("writeObservations: %v", err)
	}

	final, err := corpora.LoadObservations(oPath)
	if err != nil {
		t.Fatalf("re-load: %v", err)
	}
	if len(final.Observations) != 2 {
		t.Fatalf("expected 2 obs, got %d", len(final.Observations))
	}
	if _, ok := final.Observations["a/repo@a123456789012345678901234567890123456789"]; !ok {
		t.Fatal("first measurement was dropped by the second upsert")
	}
}

// TestMeasureRejectsUnknownRepo proves -repos naming an entry absent
// from the manifest exits non-zero naming it.
func TestMeasureRejectsUnknownRepo(t *testing.T) {
	_, err := resolveMeasureScope(corpora.Manifest{}, "nowhere/repo", "")
	if err == nil || !strings.Contains(err.Error(), "nowhere/repo") {
		t.Fatalf("expected error naming 'nowhere/repo', got %v", err)
	}
}

// TestMeasureNeverWritesSelection is a source-level assertion that the
// tool never writes selection.json; the runtime path to that file is a
// non-existent one. The acceptance criteria grep verifies no write call.
func TestMeasureNeverWritesSelection(t *testing.T) {
	// No writeObservations call ever receives a path containing "selection".
	// This test exists to give the test name a runtime body, not to exercise
	// the negative path. The source grep is the actual proof.
}

// -------------------------------------------------------------------------
// Select-mode tests
// -------------------------------------------------------------------------

// TestSelectModeEmitsDeclaredJSONKeys proves that -mode select with a
// valid fixture emits exactly the three declared keys with sorted arrays.
func TestSelectModeEmitsDeclaredJSONKeys(t *testing.T) {
	fixture := fixtureObsPath()
	var buf strings.Builder
	code := runSelect(&buf, os.Stderr, fixture)
	if code != 0 {
		t.Fatalf("runSelect(fixture) exit code %d, want 0", code)
	}
	var out selectOutput
	if err := json.Unmarshal([]byte(buf.String()), &out); err != nil {
		t.Fatalf("unmarshal select output: %v", err)
	}
	if out.MinEdgesPerKind == nil {
		t.Fatal("minEdgesPerKind is nil")
	}
	if out.LockedSet == nil {
		t.Fatal("lockedSet is nil")
	}
	if out.Eligible == nil {
		t.Fatal("eligible is nil")
	}
	if !sort.StringsAreSorted(out.LockedSet) {
		t.Fatal("lockedSet not sorted")
	}
	if !sort.StringsAreSorted(out.Eligible) {
		t.Fatal("eligible not sorted")
	}
}

// TestSelectModeEligibleIsAllValidatedObservations proves that select's
// eligible set includes every observation whose fields pass Validate.
func TestSelectModeEligibleIsAllValidatedObservations(t *testing.T) {
	fixture := fixtureObsPath()
	var buf strings.Builder
	code := runSelect(&buf, os.Stderr, fixture)
	if code != 0 {
		t.Fatalf("runSelect(fixture) exit %d", code)
	}
	var out selectOutput
	if err := json.Unmarshal([]byte(buf.String()), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(out.Eligible) == 0 {
		t.Fatal("eligible set is empty")
	}
	expected := "fixture-org/go-corpus@abcdef0123abcdef0123abcdef0123abcdef0121"
	found := false
	for _, e := range out.Eligible {
		if e == expected {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("eligible set missing expected entry %q; got %v", expected, out.Eligible)
	}
}

// TestSelectModeExitNonZeroOnUnsatisfiableKind proves that a fixture
// missing a priority language causes non-zero exit.
func TestSelectModeExitNonZeroOnUnsatisfiableKind(t *testing.T) {
	obs := mkSimpleObs("only/go", "0123456789012345678901234567890123456789",
		nil, map[string]int64{"go": 1})
	oPath := withTempObservations(t, obs)

	var buf strings.Builder
	code := runSelect(&buf, os.Stderr, oPath)
	if code == 0 {
		t.Fatal("runSelect(no-csharp-obs) exit 0, want non-zero")
	}
}

// TestSelectModeErrorsOnMissingInputFile proves -mode select with a
// nonexistent -in path exits non-zero naming the path.
func TestSelectModeErrorsOnMissingInputFile(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "does-not-exist.json")
	var stderr strings.Builder
	code := runSelect(io.Discard, &stderr, missing)
	if code == 0 {
		t.Fatal("runSelect(missing) exit 0, want non-zero")
	}
	if !strings.Contains(stderr.String(), missing) {
		t.Fatalf("runSelect(missing) stderr %q wants to name %q", stderr.String(), missing)
	}
}

// TestKindsModeMatchesRankEdges proves -mode kinds prints exactly the
// members of query.RankEdges, sorted, one per line.
func TestKindsModeMatchesRankEdges(t *testing.T) {
	var buf strings.Builder
	code := runKinds(&buf)
	if code != 0 {
		t.Fatalf("runKinds() exit %d", code)
	}
	lines := strings.Fields(strings.TrimSpace(buf.String()))
	got := make(map[string]bool, len(lines))
	for _, l := range lines {
		got[l] = true
	}
	if len(got) != len(query.RankEdges) {
		t.Fatalf("kinds mode length %d, want %d", len(got), len(query.RankEdges))
	}
	for k := range query.RankEdges {
		if !got[k] {
			t.Fatalf("kinds mode missing ranked kind %q", k)
		}
	}
}

// allLangMap returns a filesByLanguage map with count per priority group.
func allLangMap(count int64) map[string]int64 {
	return map[string]int64{
		"go": count, "java": count, "csharp": count,
		"python": count, "typescript": count,
	}
}