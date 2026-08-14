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
			// goldenCapture (behavioral_test.go, same package) mirrors
			// gocapture/main.go's own envelope shape.
			var parsed goldenCapture
			if err := json.Unmarshal(data, &parsed); err != nil {
				t.Fatalf("%s: does not parse as JSON: %v", path, err)
			}
			if parsed.Output == "" {
				t.Fatalf("%s has an empty \"output\" field", path)
			}
		})
	}
}
