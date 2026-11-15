package bfs

import (
	"slices"
	"strconv"
	"testing"

	"pgregory.net/rapid"
)

func levelsOf(start string, adj map[string][]string) [][]string {
	var out [][]string
	Levels(start, func(n string) []string { return adj[n] }, func(level []string) bool {
		out = append(out, slices.Clone(level))
		return false
	})
	return out
}

func TestLevels(t *testing.T) {
	cases := []struct {
		name  string
		adj   map[string][]string
		start string
		want  [][]string
	}{
		{
			name:  "a diamond",
			adj:   map[string][]string{"main.c": {"a.h", "b.h"}, "a.h": {"c.h"}, "b.h": {"c.h"}},
			start: "main.c",
			want:  [][]string{{"a.h", "b.h"}, {"c.h"}},
		},
		{
			name:  "a shortcut",
			adj:   map[string][]string{"main.c": {"a.h", "b.h"}, "a.h": {"b.h"}},
			start: "main.c",
			want:  [][]string{{"a.h", "b.h"}},
		},
		{
			name:  "two nodes out of sort order",
			adj:   map[string][]string{"main.c": {"z.h", "a.h"}},
			start: "main.c",
			want:  [][]string{{"z.h", "a.h"}},
		},
		{
			name:  "a cycle back to the start",
			adj:   map[string][]string{"a": {"b"}, "b": {"a"}},
			start: "a",
			want:  [][]string{{"b"}},
		},
	}
	for _, tc := range cases {
		if got := levelsOf(tc.start, tc.adj); !slices.EqualFunc(got, tc.want, slices.Equal[[]string]) {
			t.Errorf("%s: levels = %q, want %q", tc.name, got, tc.want)
		}
	}
}

// TestLevelsVisitsEachNodeAtItsLeastDepth draws a directed graph with
// cycles and checks each reachable node against the least depth that a
// shortest path gives.
func TestLevelsVisitsEachNodeAtItsLeastDepth(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		n := rapid.IntRange(1, 8).Draw(t, "nodes")
		name := func(i int) string { return "n" + strconv.Itoa(i) }
		adj := make(map[string][]string)
		for i := range n {
			for _, j := range rapid.SliceOfN(rapid.IntRange(0, n-1), 0, 3).Draw(t, "edges") {
				adj[name(i)] = append(adj[name(i)], name(j))
			}
		}
		// The least depth of each node, by relaxing every edge n times.
		depth := map[string]int{name(0): 0}
		for range n {
			for from, tos := range adj {
				d, ok := depth[from]
				if !ok {
					continue
				}
				for _, to := range tos {
					if cur, seen := depth[to]; !seen || d+1 < cur {
						depth[to] = d + 1
					}
				}
			}
		}
		seen := make(map[string]int)
		for d, level := range levelsOf(name(0), adj) {
			for _, node := range level {
				if _, dup := seen[node]; dup {
					t.Fatalf("%s comes twice", node)
				}
				seen[node] = d + 1
			}
		}
		for node, d := range depth {
			if node == name(0) {
				if _, ok := seen[node]; ok {
					t.Fatalf("the start %s comes at depth %d", node, seen[node])
				}
				continue
			}
			if seen[node] != d {
				t.Fatalf("%s comes at depth %d, want %d", node, seen[node], d)
			}
		}
		if len(seen) != len(depth)-1 {
			t.Fatalf("visited %d nodes, want the %d reachable ones", len(seen), len(depth)-1)
		}
	})
}
