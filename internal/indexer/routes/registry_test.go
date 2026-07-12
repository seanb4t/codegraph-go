package routes

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRoute_RegisteredIsDefensiveCopy proves mutating Registered()'s
// returned slice never corrupts the package-level registry.
func TestRoute_RegisteredIsDefensiveCopy(t *testing.T) {
	before := len(Registered())
	got := Registered()
	if len(got) > 0 {
		got[0] = Detector{}
	}
	after := Registered()
	if len(after) != before {
		t.Fatalf("Registered() length changed after mutating a prior result: before=%d after=%d", before, len(after))
	}
	if len(after) > 0 && after[0].ID == "" {
		t.Fatalf("Registered()'s first detector was corrupted by mutating a prior call's slice")
	}
}

// TestRoute_ASTNotRegex proves route detection never falls back to a
// second regex pass over raw source (Pattern 4 / T-05-ReDoS): none of
// this package's detector source files import "regexp". Dynamically
// globs every non-test .go file in this package, so it stays valid as
// later tasks add spring.go/aspnet.go/django.go/express.go.
func TestRoute_ASTNotRegex(t *testing.T) {
	entries, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("Glob: %v", err)
	}
	for _, path := range entries {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile(%s): %v", path, err)
		}
		if strings.Contains(string(data), `"regexp"`) {
			t.Errorf("%s imports \"regexp\" — route detection must be AST-based, never regex over raw source (T-05-ReDoS)", path)
		}
	}
}
