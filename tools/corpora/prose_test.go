// Package main — tools/corpora/prose_test.go covers the pure function
// renderMeasurementProse from both committed documents.
package main

import (
	"sort"
	"strings"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/corpora"
	"github.com/seanb4t/codegraph-go/internal/query"
)

// fixtureObs returns a small Observations set for prose tests, covering
// all five priority languages and every RankEdges kind collectively.
func fixtureObs() corpora.Observations {
	obs, _ := corpora.NewObservations(1, "m.json", []corpora.Observation{
		{
			Repo: "go/repo", SHA: "0123456789abcdef0123456789abcdef01234567",
			License: "MIT", Language: "go", TrackedFiles: 10,
			Status: map[string]any{
				"edgesByKind":     map[string]any{"calls": float64(100), "imports": float64(50), "overrides": float64(3)},
				"filesByLanguage": map[string]any{"go": float64(10)},
				"pendingChanges":  map[string]any{"added": float64(0), "modified": float64(0), "removed": float64(0)},
			},
		},
		{
			Repo: "java/repo", SHA: "1123456789abcdef0123456789abcdef01234568",
			License: "Apache-2.0", Language: "java", TrackedFiles: 8,
			Status: map[string]any{
				"edgesByKind":     map[string]any{"extends": float64(10), "implements": float64(15), "overrides": float64(2)},
				"filesByLanguage": map[string]any{"java": float64(8)},
				"pendingChanges":  map[string]any{"added": float64(0), "modified": float64(0), "removed": float64(0)},
			},
		},
		{
			Repo: "csharp/repo", SHA: "2123456789abcdef0123456789abcdef01234569",
			License: "MIT", Language: "csharp", TrackedFiles: 6,
			Status: map[string]any{
				"edgesByKind":     map[string]any{"instantiates": float64(12), "overrides": float64(8)},
				"filesByLanguage": map[string]any{"csharp": float64(6)},
				"pendingChanges":  map[string]any{"added": float64(0), "modified": float64(0), "removed": float64(0)},
			},
		},
		{
			Repo: "python/repo", SHA: "3123456789abcdef0123456789abcdef01234560",
			License: "MIT", Language: "python", TrackedFiles: 7,
			Status: map[string]any{
				"edgesByKind":     map[string]any{"references": float64(25)},
				"filesByLanguage": map[string]any{"python": float64(7)},
				"pendingChanges":  map[string]any{"added": float64(0), "modified": float64(0), "removed": float64(0)},
			},
		},
		{
			Repo: "tsjs/repo", SHA: "4123456789abcdef0123456789abcdef01234561",
			License: "MIT", Language: "typescript", TrackedFiles: 5,
			Status: map[string]any{
				"edgesByKind":     map[string]any{"returns": float64(30), "type_of": float64(5)},
				"filesByLanguage": map[string]any{"typescript": float64(5)},
				"pendingChanges":  map[string]any{"added": float64(0), "modified": float64(0), "removed": float64(0)},
			},
		},
	})
	return obs
}

func fixtureSel() corpora.Selection {
	return corpora.Selection{
		SchemaVersion:      1,
		ThresholdRationale: "half the best-observed count, clamped so every kind stays satisfiable",
		MinEdgesPerKind: map[string]int64{
			"calls": 50, "imports": 25, "returns": 15, "type_of": 2, "extends": 5,
			"implements": 7, "overrides": 4, "instantiates": 6, "references": 12,
		},
		LockedSet: []string{
			"go/repo@0123456789abcdef0123456789abcdef01234567",
			"java/repo@1123456789abcdef0123456789abcdef01234568",
			"csharp/repo@2123456789abcdef0123456789abcdef01234569",
			"python/repo@3123456789abcdef0123456789abcdef01234560",
			"tsjs/repo@4123456789abcdef0123456789abcdef01234561",
		},
		Rejected: []corpora.RejectedCandidate{
			{Repo: "old/repo", Reason: "below threshold for multiple edge kinds"},
		},
	}
}

// TestProseRendersWithoutSelection proves the renderer succeeds and says
// so when no selection has been recorded yet.
func TestProseRendersWithoutSelection(t *testing.T) {
	out := renderMeasurementProse(fixtureObs(), corpora.Selection{})
	if out == "" {
		t.Fatal("empty output from renderMeasurementProse with no selection")
	}
	if !strings.Contains(out, "No selection has been recorded") {
		t.Fatal("output lacks the no-selection banner")
	}
}

// TestProseCoversEveryRankEdgesKind proves the coverage table has a row
// for every query.RankEdges member.
func TestProseCoversEveryRankEdgesKind(t *testing.T) {
	out := renderMeasurementProse(fixtureObs(), fixtureSel())
	for kind := range query.RankEdges {
		if !strings.Contains(out, kind) {
			t.Fatalf("coverage table missing RankEdges member %q", kind)
		}
	}
}

// TestProseDerivesSupplierFromObservations proves the supplying
// repository is computed from measured counts.
func TestProseDerivesSupplierFromObservations(t *testing.T) {
	out1 := renderMeasurementProse(fixtureObs(), fixtureSel())
	if !strings.Contains(out1, "go/repo") {
		t.Fatal("go/repo should be the supplier for calls")
	}

	obs2 := fixtureObs()
	o := obs2.Observations["java/repo@1123456789abcdef0123456789abcdef01234568"]
	o.Status["edgesByKind"] = map[string]any{"calls": float64(200)}
	obs2.Observations["java/repo@1123456789abcdef0123456789abcdef01234568"] = o
	out2 := renderMeasurementProse(obs2, fixtureSel())
	if !strings.Contains(out2, "java/repo") {
		t.Fatal("java/repo should be the supplier after swapping counts")
	}
}

// TestProseIsDeterministic proves two renders of the same input match.
func TestProseIsDeterministic(t *testing.T) {
	obs, sel := fixtureObs(), fixtureSel()
	out1 := renderMeasurementProse(obs, sel)
	out2 := renderMeasurementProse(obs, sel)
	if out1 != out2 {
		t.Fatal("prose is not deterministic")
	}
}

// TestProseLabelsSyntheticCoverage proves synthetic kinds get the label.
func TestProseLabelsSyntheticCoverage(t *testing.T) {
	sel := fixtureSel()
	sel.SyntheticKinds = []string{"type_of"}
	out := renderMeasurementProse(fixtureObs(), sel)
	if !strings.Contains(out, "SYNTHETIC") {
		t.Fatal("synthetic coverage not labeled")
	}
}

// TestProseCoversPriorityLanguages proves the language table uses
// corpora.PriorityLanguages for its rows.
func TestProseCoversPriorityLanguages(t *testing.T) {
	out := renderMeasurementProse(fixtureObs(), fixtureSel())
	for _, g := range corpora.PriorityLanguages {
		if !strings.Contains(out, g.Name) {
			t.Fatalf("language table missing priority language %q", g.Name)
		}
	}
}

// sortedKindNames returns the RankEdges kind names sorted, for tests
// that need to iterate the rank set deterministically.
func sortedKindNames() []string {
	out := make([]string, 0, len(query.RankEdges))
	for k := range query.RankEdges {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}