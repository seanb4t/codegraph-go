// Package golden contains a fast smoke test asserting the D-06 golden
// fixtures (TS CodeGraph v1.3.1 schema DDL + golden JSON tool outputs) exist,
// parse, and remain stripped of the non-deterministic fields identified in
// 01-RESEARCH.md's Pitfall 1 (FTS `score` floats, `*_at`/`*At` timestamps).
//
// This test does not validate fixture *content* against a live TS install —
// it only guards that a future re-capture (via capture.sh) didn't forget to
// strip volatile fields, which would silently reintroduce spurious parity
// failures in Phases 3/4/7.
package golden

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// volatileKeys are the exact JSON key names capture.sh strips because they
// are non-deterministic across reindex runs (Pitfall 1) or machine-local
// (projectPath/indexPath are normalized rather than removed, so they are not
// checked here).
var volatileKeys = map[string]bool{
	"score":       true,
	"lastIndexed": true,
	"dbSizeBytes": true,
}

// isVolatileKey reports whether a JSON key matches one of the volatile
// patterns capture.sh strips: the exact keys in volatileKeys, or any key
// ending in "_at" or "At" (updatedAt, indexed_at, applied_at, modified_at, ...).
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
		matches, err := filepath.Glob(filepath.Join("corpus", "*", "*.json"))
		if err != nil {
			t.Fatalf("glob corpus/*/*.json: %v", err)
		}
		if len(matches) == 0 {
			t.Fatal("no corpus/*/*.json fixtures found")
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
		matches, err := filepath.Glob(filepath.Join("corpus", "*", "*.json"))
		if err != nil {
			t.Fatalf("glob corpus/*/*.json: %v", err)
		}
		if len(matches) == 0 {
			t.Fatal("no corpus/*/*.json fixtures found")
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
				t.Errorf("%s: contains volatile field(s) that should have been stripped by capture.sh: %v", m, volatile)
			}
		}
	})
}
