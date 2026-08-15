// Package golden contains a fast smoke test asserting the golden fixtures
// (the legacy schema DDL + the committed golden JSON tool outputs) exist,
// parse, and remain stripped of the non-deterministic fields identified in
// 01-RESEARCH.md's Pitfall 1 (FTS `score` floats, `*_at`/`*At` timestamps).
//
// This test does not validate fixture *content* against a live TS install —
// it only guards that a future re-capture didn't forget to strip volatile
// fields, which would silently reintroduce spurious failures in
// later phases.
package golden

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// volatileKeys are the exact JSON key names the capture path stripped
// because they are non-deterministic across reindex runs (Pitfall 1) or
// machine-local (projectPath/indexPath are normalized rather than removed,
// so they are not checked here).
var volatileKeys = map[string]bool{
	"score":       true,
	"lastIndexed": true,
	"dbSizeBytes": true,
}

// isVolatileKey reports whether a JSON key matches one of the volatile
// patterns the capture path strips: the exact keys in volatileKeys, or any
// key ending in "_at" or "At" (updatedAt, indexed_at, applied_at,
// modified_at, ...).
func isVolatileKey(key string) bool {
	if volatileKeys[key] {
		return true
	}
	if strings.HasSuffix(key, "_at") || strings.HasSuffix(key, "At") {
		return true
	}
	return false
}

// findVolatileKeys recursively walks a decoded JSON value (as produced by
// encoding/json's default decoding into interface{}) and returns every
// volatile key path found, for a descriptive test failure message.
func findVolatileKeys(v interface{}, path string) []string {
	var found []string
	switch val := v.(type) {
	case map[string]interface{}:
		for k, child := range val {
			childPath := path + "." + k
			if isVolatileKey(k) {
				found = append(found, childPath)
			}
			found = append(found, findVolatileKeys(child, childPath)...)
		}
	case []interface{}:
		for _, child := range val {
			found = append(found, findVolatileKeys(child, path+"[]")...)
		}
	}
	return found
}

func TestGoldenFixturesExist(t *testing.T) {
	t.Run("ts-schema.sql exists and is non-empty", func(t *testing.T) {
		info, err := os.Stat("ts-schema.sql")
		if err != nil {
			t.Fatalf("ts-schema.sql: %v", err)
		}
		if info.Size() == 0 {
			t.Fatal("ts-schema.sql is empty")
		}
	})

	t.Run("ts-version.txt exists, non-empty, and records a version", func(t *testing.T) {
		data, err := os.ReadFile("ts-version.txt")
		if err != nil {
			t.Fatalf("ts-version.txt: %v", err)
		}
		if len(data) == 0 {
			t.Fatal("ts-version.txt is empty")
		}
		if !strings.Contains(string(data), "codegraph_version=") {
			t.Fatalf("ts-version.txt does not record a codegraph_version: %q", string(data))
		}
	})

	t.Run("at least one corpus JSON fixture exists and parses", func(t *testing.T) {
		localMatches, _ := filepath.Glob(filepath.Join("corpus", "*", "*.json"))
		behavioralMatches, _ := filepath.Glob(filepath.Join("..", "..", "corpus", "behavioral", "*.json"))
		matches := append(localMatches, behavioralMatches...)
		if len(matches) == 0 {
			t.Fatal("no corpus JSON fixtures found")
		}

		for _, m := range matches {
			data, err := os.ReadFile(m)
			if err != nil {
				t.Errorf("%s: read error: %v", m, err)
				continue
			}
			if len(data) == 0 {
				t.Errorf("%s: fixture is empty", m)
				continue
			}
			var parsed interface{}
			if err := json.Unmarshal(data, &parsed); err != nil {
				t.Errorf("%s: does not parse as JSON: %v", m, err)
			}
		}
	})

	t.Run("corpus JSON fixtures are stripped of volatile fields", func(t *testing.T) {
		localMatches, _ := filepath.Glob(filepath.Join("corpus", "*", "*.json"))
		behavioralMatches, _ := filepath.Glob(filepath.Join("..", "..", "corpus", "behavioral", "*.json"))
		matches := append(localMatches, behavioralMatches...)
		if len(matches) == 0 {
			t.Fatal("no corpus JSON fixtures found")
		}

		for _, m := range matches {
			data, err := os.ReadFile(m)
			if err != nil {
				t.Errorf("%s: read error: %v", m, err)
				continue
			}
			var parsed interface{}
			if err := json.Unmarshal(data, &parsed); err != nil {
				// Already reported by the parse test above; skip here.
				continue
			}
			if volatile := findVolatileKeys(parsed, m); len(volatile) > 0 {
				t.Errorf("%s: contains volatile field(s) that should have been stripped: %v", m, volatile)
			}
		}
	})
}

// TestGoSideFixturesRegenerated pins F5 (plan 17): the Go-side EXPECTED
// fixtures (go-explore-multi.json/go-node-multi.json, produced by
// `go run ./testdata/golden/gocapture` running the CURRENT Go explore/node
// pipeline against the re-indexed corpora) must exist and be non-empty for
// the behavioral corpus (corpus/behavioral/, always available in-repo).
// This guards against F5 silently going stale again the way the
// PRE-plan-17 explore.json/node.json fixtures did after the D-09 re-index
// (01-15-SUMMARY.md) — a future contributor who changes the explore/node
// pipeline without re-running gocapture will at least not have a MISSING
// go-*.json fixture slip past review, even though this test cannot by
// itself detect a STALE (present but outdated) one.
func TestGoSideFixturesRegenerated(t *testing.T) {
	for _, name := range []string{"go-explore-multi.json", "go-node-multi.json"} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join("..", "..", "corpus", "behavioral", name)
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("%s: %v (run `go run ./testdata/golden/gocapture` to regenerate — F5, plan 17)", path, err)
			}
			if len(data) == 0 {
				t.Fatalf("%s is empty", path)
			}
			var parsed goldenCapture
			if err := json.Unmarshal(data, &parsed); err != nil {
				t.Fatalf("%s: does not parse as golden capture: %v", path, err)
			}
			if parsed.Output == "" {
				t.Fatalf("%s has an empty \"output\" field", path)
			}
		})
	}
}

// ============================================================================
// Re-frozen golden guard (02-03, FIXT-06)
// ============================================================================
//
// TestReFrozenGoldensValid enumerates the EXPECTED golden set from the
// gocapture spec table (the per-corpus expected filenames), requires each
// to exist AND be non-empty AND carry the envelope marker AND parse, and
// positively asserts the verified count (rule 84d1gfpywd). This guard
// enumerates from an AUTHORITATIVE source — NEVER from filepath.Glob of
// existing files — so a missing expected golden fails the suite.

// expectedGoldenFiles defines the per-corpus expected golden filenames, in
// the same structure as gocapture's spec table (one entry per corpus slug,
// listing the 6 expected golden filenames per locked corpus).
type expectedCorpusSet struct {
	slug   string
	files  []string
}

var expectedGoCaptures = []expectedCorpusSet{
	{
		slug: "behavioral",
		files: []string{
			"go-explore-multi.json",
			"go-node-multi.json",
		},
	},
	{
		slug: "hugo",
		files: []string{
			"go-explore.json",
			"go-node.json",
			"go-explore-multi.json",
			"go-node-multi.json",
			"go-explore-mcp.json",
			"go-node-mcp.json",
		},
	},
	{
		slug: "guava",
		files: []string{
			"go-explore.json",
			"go-node.json",
			"go-explore-multi.json",
			"go-node-multi.json",
			"go-explore-mcp.json",
			"go-node-mcp.json",
		},
	},
	{
		slug: "serilog",
		files: []string{
			"go-explore.json",
			"go-node.json",
			"go-explore-multi.json",
			"go-node-multi.json",
			"go-explore-mcp.json",
			"go-node-mcp.json",
		},
	},
	{
		slug: "requests",
		files: []string{
			"go-explore.json",
			"go-node.json",
			"go-explore-multi.json",
			"go-node-multi.json",
			"go-explore-mcp.json",
			"go-node-mcp.json",
		},
	},
}

// ExpectedGoldenScenarioCount is the declared total the golden suite's
// executed-scenario self-assertion must prove (D-02, wire-oracle
// TestScenarioCountIsExact precedent): 26 goldens enumerated from
// expectedGoCaptures (2 behavioral + 6 each for the four locked corpora)
// plus the 4 behavioral property cases from corpus/behavioral/CASES.json
// = 30. The constant is placed beside the authoritative derivation, never
// derived from a hand-maintained literal — TestGoldenScenarioCountIsExact
// proves it from the tables. A shrinking count is the failure mode this
// constant exists to catch.
const ExpectedGoldenScenarioCount = 30

// TestReFrozenGoldensValid enumerates the EXPECTED golden set from the
// gocapture spec table (not a glob), and for each expected golden:
//  1. asserts the file EXISTS (missing expected golden fails the suite);
//  2. asserts it is non-empty and byte-prefixed with the `{` envelope marker;
//  3. parses it as the goldenCapture{Command, Output} envelope and asserts a
//     non-empty Output for command-invocation goldens;
//  4. positively asserts a COUNT of goldens verified, so a guard that ran over
//     zero expected goldens fails instead of passing vacuously (H5).
func TestReFrozenGoldensValid(t *testing.T) {
	expectedTotal := 0
	verified := 0

	for _, cs := range expectedGoCaptures {
		for _, name := range cs.files {
			expectedTotal++
			t.Run(cs.slug+"/"+name, func(t *testing.T) {
				// Resolve the golden path. Behavioral goldens live at
				// repo-root corpus/behavioral/, locked goldens at
				// testdata/golden/corpus/<slug>/.
				var path string
				if cs.slug == "behavioral" {
					path = filepath.Join("..", "..", "corpus", "behavioral", name)
				} else {
					path = filepath.Join("corpus", cs.slug, name)
				}

				// 1. Assert file exists.
				data, err := os.ReadFile(path)
				if err != nil {
					t.Fatalf("expected golden %s not found (run `task golden:regen` to regenerate): %v", path, err)
				}

				// 2. Assert non-empty and marker.
				if len(data) == 0 {
					t.Fatalf("expected golden %s is empty", path)
				}
				if data[0] != '{' {
					t.Fatalf("expected golden %s first byte is %q (want '{') — not a valid JSON envelope", path, data[0])
				}

				// 3. Parse and assert non-empty output.
				var capture goldenCapture
				if err := json.Unmarshal(data, &capture); err != nil {
					t.Fatalf("expected golden %s does not parse as goldenCapture: %v", path, err)
				}
				if capture.Output == "" {
					t.Fatalf("expected golden %s has an empty output field", path)
				}

				verified++
			})
		}
	}

	t.Logf("TestReFrozenGoldensValid: %d/%d goldens verified", verified, expectedTotal)
	if expectedTotal == 0 {
		t.Fatal("expected golden list is empty — guard ran over zero expected goldens (H5)")
	}
	if verified != expectedTotal {
		t.Fatalf("verified %d of %d expected goldens — the executed-and-verified count must EXACTLY equal the enumerated total (a count short by even one means a golden was dropped or a subtest loop never ran; missing entries must be regenerated via `task golden:regen`)", verified, expectedTotal)
	}
}

// TestGoldenScenarioCountIsExact is FIXT-03's central non-shrinkage guard
// (D-02), mirroring the wire-oracle TestScenarioCountIsExact precedent:
// the golden suite's scenario total, DERIVED from the authoritative tables
// (the expectedGoCaptures file lists + loadBehavioralCases's committed
// CASES.json), must equal ExpectedGoldenScenarioCount with EXACT equality —
// never a lower bound, because a lower bound cannot detect a scenario
// silently disappearing. The just-touching adjacency of the exact
// coincidence (26 goldens + 4 cases lands exactly on 30) is enforced
// individually on each leg before the combined check, and a zero on either
// leg is fatal before the sum check so a suite that never derived a
// scenario is red by construction.
func TestGoldenScenarioCountIsExact(t *testing.T) {
	goldenTotal := 0
	for _, cs := range expectedGoCaptures {
		goldenTotal += len(cs.files)
	}
	if goldenTotal == 0 {
		t.Fatal("goldenTotal is 0 — expectedGoCaptures enumerates no goldens; the derivation cannot be trusted")
	}
	if goldenTotal != 26 {
		t.Fatalf("goldenTotal = %d, want exactly 26 — either a golden silently disappeared from expectedGoCaptures or one was added without updating the derivation", goldenTotal)
	}

	caseTotal := len(loadBehavioralCases(t))
	if caseTotal == 0 {
		t.Fatal("caseTotal is 0 — loadBehavioralCases returned no cases; the derivation cannot be trusted")
	}
	if caseTotal != 4 {
		t.Fatalf("caseTotal = %d, want exactly 4 — either a CASES.json case silently disappeared or one was added without updating the derivation", caseTotal)
	}

	total := goldenTotal + caseTotal
	if total != ExpectedGoldenScenarioCount {
		t.Fatalf("golden suite scenario total = %d, want exactly %d (ExpectedGoldenScenarioCount) — either a scenario silently disappeared or one was added without updating the constant beside expectedGoCaptures", total, ExpectedGoldenScenarioCount)
	}
	t.Logf("golden suite scenario count: goldens: %d/26, CASES cases: %d/4, total: %d/%d", goldenTotal, caseTotal, total, ExpectedGoldenScenarioCount)
}