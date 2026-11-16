package symbols

import (
	"slices"
	"testing"
)

func TestWalkSupertypes(t *testing.T) {
	t.Run("a cycle never visits the start", func(t *testing.T) {
		var got []string
		WalkSupertypes("A", map[string][]string{"A": {"B"}, "B": {"A"}}, func(id string) bool {
			got = append(got, id)
			return false
		})
		if !slices.Equal(got, []string{"B"}) {
			t.Errorf("visited %q, want [B]", got)
		}
	})
	t.Run("a visitor that stops on the first of two at one depth", func(t *testing.T) {
		var got []string
		stopped := WalkSupertypes("A", map[string][]string{"A": {"B", "C"}}, func(id string) bool {
			got = append(got, id)
			return id == "B"
		})
		if !stopped || !slices.Equal(got, []string{"B"}) {
			t.Errorf("visited %q, stopped %v; want [B], true", got, stopped)
		}
	})
}
