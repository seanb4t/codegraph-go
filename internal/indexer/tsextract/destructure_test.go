package tsextract

import (
	"slices"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
)

// TestDestructuringAssignmentWrites covers an assignment whose target is
// a destructuring pattern, in TypeScript, JavaScript and TSX. Each name
// the pattern binds is a write by the enclosing function: a shorthand
// property, a renamed property, a shorthand property with a default, a
// rest element, and the elements of an array pattern.
func TestDestructuringAssignmentWrites(t *testing.T) {
	const src = `let a, c, d, e, f, g;
function assign(obj, list) {
  ({ a, b: c, d = 1, ...e } = obj);
  [f, ...g] = list;
}
`
	for _, tc := range []struct{ grammar, path string }{{"ts", "m.ts"}, {"js", "m.js"}, {"tsx", "m.tsx"}} {
		t.Run(tc.grammar, func(t *testing.T) {
			result, err := Extract(newTestParser(t, tc.grammar), "m", tc.path, []byte(src))
			if err != nil {
				t.Fatalf("Extract: %v", err)
			}
			var writes []string
			for _, ref := range result.Unresolved {
				if ref.Kind == goextract.RefKindReferences && ref.Access == goextract.AccessWrite {
					writes = append(writes, ref.Name)
				}
			}
			slices.Sort(writes)
			if want := []string{"a", "c", "d", "e", "f", "g"}; !slices.Equal(writes, want) {
				t.Errorf("writes = %v, want %v", writes, want)
			}
		})
	}
}
