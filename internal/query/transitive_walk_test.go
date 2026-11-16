package query

import (
	"slices"
	"testing"
)

func TestTransitiveWalk(t *testing.T) {
	cases := []struct {
		name  string
		index map[string][]string
		want  []string
	}{
		{"a cycle back to the root", map[string][]string{"root": {"a"}, "a": {"root"}}, []string{"a"}},
		{
			"a diamond",
			map[string][]string{"root": {"c", "b"}, "b": {"d"}, "c": {"d", "a"}},
			[]string{"b", "c", "a", "d"},
		},
	}
	for _, tc := range cases {
		if got := transitiveWalk("root", tc.index); !slices.Equal(got, tc.want) {
			t.Errorf("%s: transitiveWalk = %q, want %q", tc.name, got, tc.want)
		}
	}
}
