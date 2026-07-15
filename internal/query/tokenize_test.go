package query

import (
	"reflect"
	"testing"
)

// contains reports whether s is present in tokens.
func contains(tokens []string, s string) bool {
	for _, t := range tokens {
		if t == s {
			return true
		}
	}
	return false
}

// TestExtractSearchTerms exercises H2 (extractSearchTerms + STOP_WORDS,
// search/query-utils.js:189-242, RESEARCH §2) — EXPL-01's literal
// "stopword-filtered" tokenizer feeding the FTS gather channel (plan 07).
func TestExtractSearchTerms(t *testing.T) {
	t.Run("multi-word query splits and preserves compounds", func(t *testing.T) {
		got := extractSearchTerms("getUserName by id")
		for _, want := range []string{"get", "user", "name", "getusername"} {
			if !contains(got, want) {
				t.Errorf("extractSearchTerms(%q) = %v, want to contain %q", "getUserName by id", got, want)
			}
		}
		for _, notWant := range []string{"by", "id"} {
			if contains(got, notWant) {
				t.Errorf("extractSearchTerms(%q) = %v, want NOT to contain %q (stopword or len<3)", "getUserName by id", got, notWant)
			}
		}
	})

	t.Run("all-stopword query yields empty", func(t *testing.T) {
		got := extractSearchTerms("the fix bug")
		if len(got) != 0 {
			t.Errorf("extractSearchTerms(%q) = %v, want empty", "the fix bug", got)
		}
	})

	t.Run("empty and whitespace-only query never becomes match-all (WR-05/V5)", func(t *testing.T) {
		for _, q := range []string{"", "   "} {
			got := extractSearchTerms(q)
			if len(got) != 0 {
				t.Errorf("extractSearchTerms(%q) = %v, want empty slice (WR-05 guard)", q, got)
			}
		}
	})

	t.Run("acronym/camel boundary split", func(t *testing.T) {
		got := extractSearchTerms("HTTPServer")
		for _, want := range []string{"http", "server", "httpserver"} {
			if !contains(got, want) {
				t.Errorf("extractSearchTerms(%q) = %v, want to contain %q", "HTTPServer", got, want)
			}
		}
	})

	t.Run("deterministic order across repeated calls", func(t *testing.T) {
		const q = "getUserName by id HTTPServer"
		first := extractSearchTerms(q)
		second := extractSearchTerms(q)
		if !reflect.DeepEqual(first, second) {
			t.Errorf("extractSearchTerms(%q) not deterministic: %v vs %v", q, first, second)
		}
	})
}
