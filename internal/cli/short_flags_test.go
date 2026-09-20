package cli

import (
	"testing"

	"github.com/spf13/cobra"
)

// queryVerbNames is exactly D-13's Query-the-graph group (04-CONTEXT.md).
var queryVerbNames = []string{"explore", "search", "node", "callers", "callees", "impact", "affected", "files", "status"}

// shortFlagPairs maps each long flag name CLI-07 cares about to its
// required shorthand.
var shortFlagPairs = map[string]string{
	"json":  "j",
	"limit": "l",
	"kind":  "k",
	"path":  "p",
}

// shortFlagsMinPairs is the positive floor (rule 84d1gfpywd): a walk that
// inspects nothing must never read as green. D-12's discovery counted 20
// (verb, flag) pairs on today's tree — search 4, callers 3, callees 3,
// impact 2, affected 2, files 2, status 2, node 1, explore 0 — so the
// floor is set comfortably below that observed count, not equal to it: it
// exists to catch a walk that silently finds nothing, not to pin the exact
// number (which this test logs, but does not assert as a ceiling).
const shortFlagsMinPairs = 18

// TestShortFlagsConsistent pins D-12's already-true CLI-07 invariant: every
// query verb (D-13's Query-the-graph group) that registers one of
// --json/--limit/--kind/--path also carries its short form. It never
// re-derives or reimplements cobra's own flag parsing (D-00) — it only
// walks the real, already-built command tree (cliReferenceTree,
// cli_reference_test.go) and reads pflag.Flag.Shorthand.
func TestShortFlagsConsistent(t *testing.T) {
	root := cliReferenceTree()

	byName := make(map[string]*cobra.Command, len(queryVerbNames))
	for _, sub := range root.Commands() {
		byName[sub.Name()] = sub
	}

	var missing []string
	for _, verb := range queryVerbNames {
		if _, ok := byName[verb]; !ok {
			missing = append(missing, verb)
		}
	}
	if len(missing) > 0 {
		t.Fatalf("query verb(s) missing from the command tree (the tree changed under this test): %v", missing)
	}

	checked := 0
	for _, verb := range queryVerbNames {
		cmd := byName[verb]
		for _, long := range []string{"json", "limit", "kind", "path"} {
			f := cmd.Flags().Lookup(long)
			if f == nil {
				// The long form does not exist on this verb (explore has
				// none of the four; node has --line, not --limit) —
				// contributes zero pairs by design (D-12).
				continue
			}
			checked++
			want := shortFlagPairs[long]
			if f.Shorthand != want {
				t.Errorf("%s --%s has shorthand %q, want -%s", verb, long, f.Shorthand, want)
			}
		}
	}

	t.Logf("inspected %d (verb, flag) pairs across %d query verbs", checked, len(queryVerbNames))
	if checked < shortFlagsMinPairs {
		t.Fatalf("inspected only %d (verb, flag) pairs — expected at least %d (D-12 floor, rule 84d1gfpywd)", checked, shortFlagsMinPairs)
	}
}
