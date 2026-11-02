package phpextract

import (
	"testing"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
)

// TestGuardedFunctionIsDeclared checks design.md D16 for PHP: a function
// under a `function_exists` guard is a module-level declaration, not a
// nested one, and two guarded copies are two declarations.
func TestGuardedFunctionIsDeclared(t *testing.T) {
	src := `<?php
if (!function_exists('slugify')) {
    function slugify($s) { return $s; }
}
if (!function_exists('slugify')) {
    function slugify($s) { return strtolower($s); }
}
function useIt() { return slugify("A"); }
`
	p := newTestParser(t)
	res, err := Extract(p, "app", "helpers.php", []byte(src))
	if err != nil || res.Err != nil {
		t.Fatalf("Extract: %v / %v", err, res.Err)
	}
	var ids []string
	for _, en := range res.Nodes {
		if en.Node.Kind == goextract.KindFunction && en.Node.Name == "slugify" {
			ids = append(ids, en.Node.Id)
			if en.Node.Nested {
				t.Errorf("slugify at line %d is marked nested, want module-level", en.Node.StartLine)
			}
		}
	}
	if len(ids) != 2 || ids[0] == ids[1] {
		t.Fatalf("slugify ids = %v, want two distinct ids", ids)
	}
}
