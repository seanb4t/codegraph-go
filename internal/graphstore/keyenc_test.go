package graphstore

import (
	"bytes"
	"sort"
	"testing"
)

// adversarialSegments are candidate id/path values crafted to try to
// escape their segment: the namespace delimiter byte, NUL, the 0xFF
// boundary byte, an empty string, and a multi-byte unicode value.
var adversarialSegments = []string{
	"",
	"a",
	"a/b",
	"a/b/c",
	"a\x00b",
	"a\xffb",
	"/",
	"\x00",
	"\xff",
	"héllo/wörld",
	"a//b",
}

// TestKeyEncodingRejectsDelimiterInjection proves that distinct logical
// (namespace, id) inputs — including ones containing the namespace
// delimiter ('/') or control/boundary bytes (0x00, 0xFF) — never produce
// equal or prefix-overlapping keys, and that no crafted node id can
// forge a key that sorts inside another namespace's range (V5 / T-01-02).
func TestKeyEncodingRejectsDelimiterInjection(t *testing.T) {
	t.Run("distinct node ids never collide", func(t *testing.T) {
		seen := map[string]string{}
		for _, id := range adversarialSegments {
			k := nodeKey(id)
			ks := string(k)
			if other, ok := seen[ks]; ok && other != id {
				t.Fatalf("nodeKey(%q) collides with nodeKey(%q): both encode to %x", id, other, k)
			}
			seen[ks] = id
		}
	})

	t.Run("no naive-concat aliasing across segment boundaries", func(t *testing.T) {
		// A naive "prefix + sep + a + sep + b" scheme would let
		// nodeKey("a/b") collide with a hypothetical two-segment
		// encoding of ("a","b"). Our nodeKey has only one segment, so
		// assert instead that splitting a delimiter-containing id does
		// not produce the same bytes as the whole id encoded as one
		// segment — i.e. the delimiter is not treated as structural.
		whole := nodeKey("a/b")
		asTwoNodes := append(append([]byte{}, nodeKey("a")...), nodeKey("b")...)
		if bytes.Equal(whole, asTwoNodes) {
			t.Fatalf("nodeKey(%q) aliases nodeKey(%q)+nodeKey(%q)", "a/b", "a", "b")
		}
	})

	t.Run("no crafted node key falls inside another namespace's range", func(t *testing.T) {
		for _, id := range adversarialSegments {
			nk := nodeKey(id)
			if nk[0] != prefixNode {
				t.Fatalf("nodeKey(%q) does not start with prefixNode: %x", id, nk)
			}
			// A key can only ever sort "inside" a foreign namespace's
			// range if its leading byte matches that namespace's
			// prefix. Since every builder emits its own fixed prefix
			// byte first and no segment content can alter that first
			// byte, cross-namespace containment is structurally
			// impossible — assert the invariant explicitly here so a
			// future refactor that broke it would fail this test.
			for _, foreign := range []byte{prefixMeta, prefixEdge, prefixFile, prefixAnnotation} {
				if nk[0] == foreign {
					t.Fatalf("nodeKey(%q) leading byte %q collides with foreign namespace %q", id, nk[0], foreign)
				}
			}
		}
	})

	t.Run("edge keys with adversarial segments stay distinct and ordered by src", func(t *testing.T) {
		seen := map[string]struct{ src, kind, dst string }{}
		for _, src := range adversarialSegments {
			for _, kind := range []string{"calls", "a/b"} {
				for _, dst := range adversarialSegments {
					k := edgeKey(src, kind, dst)
					ks := string(k)
					if prev, ok := seen[ks]; ok {
						if prev.src != src || prev.kind != kind || prev.dst != dst {
							t.Fatalf("edgeKey(%q,%q,%q) collides with edgeKey(%q,%q,%q)",
								src, kind, dst, prev.src, prev.kind, prev.dst)
						}
					}
					seen[ks] = struct{ src, kind, dst string }{src, kind, dst}
					if k[0] != prefixEdge {
						t.Fatalf("edgeKey(%q,%q,%q) does not start with prefixEdge: %x", src, kind, dst, k)
					}
				}
			}
		}
	})

	t.Run("file keys with adversarial paths never collide", func(t *testing.T) {
		seen := map[string]string{}
		for _, path := range adversarialSegments {
			k := fileKey(path)
			ks := string(k)
			if other, ok := seen[ks]; ok && other != path {
				t.Fatalf("fileKey(%q) collides with fileKey(%q)", path, other)
			}
			seen[ks] = path
		}
	})
}

// TestEdgeKeyRangeScanOrdering asserts that, when byte-sorted (the order
// Pebble stores keys in), all edges sharing a source form one contiguous
// range under edgeSrcPrefix(src) — the property callers/callees/impact
// queries depend on (D-03).
func TestEdgeKeyRangeScanOrdering(t *testing.T) {
	type edge struct{ src, kind, dst string }
	edges := []edge{
		{"alpha", "calls", "beta"},
		{"alpha", "calls", "gamma"},
		{"alpha", "imports", "delta"},
		{"beta", "calls", "gamma"},
		{"beta", "implements", "alpha"},
		{"gamma", "calls", "alpha"},
		// adversarial sources thrown into the same scan
		{"a/b", "calls", "x"},
		{"a/b", "calls", "y"},
		{"a", "calls", "z"},
	}

	keys := make([][]byte, len(edges))
	for i, e := range edges {
		keys[i] = edgeKey(e.src, e.kind, e.dst)
	}
	sort.Slice(keys, func(i, j int) bool { return bytes.Compare(keys[i], keys[j]) < 0 })

	sources := map[string][]bool{}
	for _, e := range edges {
		sources[e.src] = nil
	}

	for src := range sources {
		prefix := edgeSrcPrefix(src)
		upper := rangeUpperBound(prefix)

		inRange := false
		sawGapAfterRange := false
		for _, k := range keys {
			within := bytes.HasPrefix(k, prefix) && (upper == nil || bytes.Compare(k, upper) < 0)
			if within {
				if sawGapAfterRange {
					t.Fatalf("src %q: keys matching prefix are not contiguous in sorted order", src)
				}
				inRange = true
			} else if inRange {
				sawGapAfterRange = true
			}
		}
		if !inRange {
			t.Fatalf("src %q: no keys found in its own prefix range", src)
		}
	}

	// Explicitly verify the "alpha" range contains exactly its own three
	// edges and none belonging to "a" or "a/b" despite the shared byte
	// "a" prefix textually.
	alphaPrefix := edgeSrcPrefix("alpha")
	alphaUpper := rangeUpperBound(alphaPrefix)
	count := 0
	for _, e := range edges {
		k := edgeKey(e.src, e.kind, e.dst)
		within := bytes.HasPrefix(k, alphaPrefix) && (alphaUpper == nil || bytes.Compare(k, alphaUpper) < 0)
		if within {
			if e.src != "alpha" {
				t.Fatalf("edge with src %q incorrectly falls inside alpha's range", e.src)
			}
			count++
		}
	}
	if count != 3 {
		t.Fatalf("expected 3 edges in alpha's range, got %d", count)
	}
}

// TestFileSubgraphRangeDeleteBoundary asserts that the range-delete window
// [fileSubgraphPrefix(path), rangeUpperBound(prefix)) covers exactly that
// file's own key and excludes a lexicographically adjacent file whose path
// is a naive prefix or suffix of it.
func TestFileSubgraphRangeDeleteBoundary(t *testing.T) {
	cases := []struct {
		name          string
		target        string
		adjacentPaths []string
	}{
		{
			name:          "prefix collision: foo vs foobar",
			target:        "foo",
			adjacentPaths: []string{"foobar", "foo2", "foo/bar.go"},
		},
		{
			name:          "suffix collision: bar.go vs foobar.go",
			target:        "bar.go",
			adjacentPaths: []string{"foobar.go", "bar.go2"},
		},
		{
			name:          "delimiter-bearing path",
			target:        "src/pkg/a",
			adjacentPaths: []string{"src/pkg/ab", "src/pkg/a/b.go"},
		},
		{
			name:          "empty path is its own boundary",
			target:        "",
			adjacentPaths: []string{"a", "\x00"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			targetKey := fileKey(tc.target)
			prefix := fileSubgraphPrefix(tc.target)
			upper := rangeUpperBound(prefix)

			if !bytes.HasPrefix(targetKey, prefix) {
				t.Fatalf("fileKey(%q) does not start with its own fileSubgraphPrefix", tc.target)
			}
			if upper != nil && bytes.Compare(targetKey, upper) >= 0 {
				t.Fatalf("fileKey(%q)=%x is not below its own upper bound %x", tc.target, targetKey, upper)
			}

			for _, adj := range tc.adjacentPaths {
				adjKey := fileKey(adj)
				within := bytes.HasPrefix(adjKey, prefix) && (upper == nil || bytes.Compare(adjKey, upper) < 0)
				if within {
					t.Fatalf("adjacent path %q (key %x) incorrectly falls inside %q's range-delete window [%x, %x)",
						adj, adjKey, tc.target, prefix, upper)
				}
			}
		})
	}
}
