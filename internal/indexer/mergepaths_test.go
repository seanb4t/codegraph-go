package indexer

import (
	"slices"
	"testing"
)

func TestMergeSortedPaths(t *testing.T) {
	cases := []struct {
		name string
		a    []string
		more [][]string
		want []string
	}{
		{"no other list", []string{"a", "c"}, nil, []string{"a", "c"}},
		{"only empty lists", []string{"a", "c"}, [][]string{nil, {}}, []string{"a", "c"}},
		{"an empty first list", nil, [][]string{{"b", "a"}}, []string{"a", "b"}},
		{"a path in several lists", []string{"b"}, [][]string{{"a", "b"}, {"b", "c"}, {"a"}}, []string{"a", "b", "c"}},
		{"lists that interleave", []string{"a", "d"}, [][]string{{"c"}, {"b", "e"}}, []string{"a", "b", "c", "d", "e"}},
	}
	for _, tc := range cases {
		if got := mergeSortedPaths(tc.a, tc.more...); !slices.Equal(got, tc.want) {
			t.Errorf("%s: mergeSortedPaths(%q, %q) = %q, want %q", tc.name, tc.a, tc.more, got, tc.want)
		}
	}
}
