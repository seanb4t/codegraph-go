# 04-MUTATION-LOG — CLI Glow-up

**Phase:** 04-cli-glow-up
**Date:** 2026-09-17
**Scope:** Demonstration families across this phase's plans — (a) D-16/CLI-05 (plan 04-01): a
planted ESC byte in `status.go`'s plain branch turns `TestPlainGolden` and
`TestNoColorNonTTYRegression` RED; (b) D-12/CLI-07 (plan 04-01): a long-only `--limit` flag on
`callers` turns `TestShortFlagsConsistent` RED, naming `callers --limit`; (c) D-15/GRD-13 (plan
04-02): a planted `colorprofile` import in an untracked `internal/query` file turns
`TestNoCharmInServeReachablePackages` RED, naming both the package and the import. (Family (d),
the analogous proof for `charm.land/fang/v2`, does not apply — the 04-02 fang spike verdict is
`declined`; no fang line ever reaches `go.mod`.)

## Pre-mutation cleanliness gate — the convention this log follows

Before every tracked-file mutation, AND before every revert, `git diff --quiet -- <file>` is
asserted to exit clean. This proves no pre-existing tracked edit was overwritten by the
mutation, and no revert was a destructive blind checkout of someone else's in-flight work.
Every family entry below records this gate's result at the point it was checked.

---

## Family (a) — D-16/CLI-05: a planted ESC byte turns TestPlainGolden RED

**Test/guard:** `TestPlainGolden` and `TestNoColorNonTTYRegression`
(`internal/cli/plain_golden_test.go`) — the byte-equality + zero-ESC pair frozen against
`internal/cli/testdata/plain/*.golden` before any renderer, resolver or palette exists this
phase.

**What are we testing, and why?** Whether the golden compare and the zero-ESC assertion are
actually live — i.e. that a styled/ANSI-leaking regression on the plain path would be caught —
before any later plan in this phase relies on `TestPlainGolden` as its own `<verify>`. A gate
that always passes regardless of the property it claims to check is worse than no gate (rule
`84d1gfpywd`).

**Pre-mutation gate:** `git diff --quiet -- internal/cli/status.go` — clean.

**Mutation applied:** via `perl -0pi`, prefixed the plain branch's `query.RenderStatusText(result,
start)` argument with the literal Go string `"\x1b[1m"+` in `internal/cli/status.go`:

```diff
--- a/internal/cli/status.go
+++ b/internal/cli/status.go
@@ -91,7 +91,7 @@ func newStatusCmd() *cobra.Command {
 			// (from result.WorktreeMismatch, live since plan 02-04) at D-09's
 			// structural position — no separate warning print here, which
 			// would double it.
-			fmt.Fprint(cmd.OutOrStdout(), query.RenderStatusText(result, start))
+			fmt.Fprint(cmd.OutOrStdout(), "\x1b[1m"+query.RenderStatusText(result, start))
 			return nil
 		},
 	}
```

**Observed failure** (verbatim, `GOTOOLCHAIN=go1.26.6 go test ./internal/cli/ -count=1 -run
'TestPlainGolden$|TestNoColorNonTTYRegression$' -v`, exit code appended):

```
=== RUN   TestPlainGolden/status
    plain_golden_test.go:483: output does not match golden /Volumes/Code/github.com/seanb4t/codegraph-go/internal/cli/testdata/plain/status.golden
        first differing line 1:
          want: "CodeGraph Status"
          got:  "\x1b[1mCodeGraph Status"
    plain_golden_test.go:486: output contains an ESC byte (0x1b): "\x1b[1mCodeGraph Status\n\nProject: <FIXTURE>\n\nIndex Statistics:\n  Files:     4\n  Nodes:     20\n  Edges:     22\n  DB Size:   <SIZE> MB\n  Backend:   pebble\nNodes by Kind:\n  file            4\n  function        4\n  struct          3\n  interface       2\n  method          2\n  package         2\n  constant        1\n  type_alias      1\n  variable        1\nEdges by Kind:\n  contains        14\n  calls           3\n  imports         2\n  embeds          1\n  extends         1\n  instantiates    1\nFiles by Language:\n  go              4\n\nIndex is up to date.\n"
--- FAIL: TestPlainGolden/status (0.13s)
...
=== RUN   TestNoColorNonTTYRegression/NO_COLOR=1/status
    plain_golden_test.go:556: NO_COLOR=1: output does not match golden /Volumes/Code/github.com/seanb4t/codegraph-go/internal/cli/testdata/plain/status.golden
        first differing line 1:
          want: "CodeGraph Status"
          got:  "\x1b[1mCodeGraph Status"
    plain_golden_test.go:559: NO_COLOR=1: output contains an ESC byte (0x1b): "\x1b[1mCodeGraph Status\n\n...\n"
--- FAIL: TestNoColorNonTTYRegression/NO_COLOR=1/status (0.09s)
...
=== RUN   TestNoColorNonTTYRegression/NO_COLOR=banana/status
    plain_golden_test.go:556: NO_COLOR=banana: output does not match golden /Volumes/Code/github.com/seanb4t/codegraph-go/internal/cli/testdata/plain/status.golden
        first differing line 1:
          want: "CodeGraph Status"
          got:  "\x1b[1mCodeGraph Status"
    plain_golden_test.go:559: NO_COLOR=banana: output contains an ESC byte (0x1b): "\x1b[1mCodeGraph Status\n\n...\n"
--- FAIL: TestNoColorNonTTYRegression/NO_COLOR=banana/status (0.13s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/cli	6.xxs
FAIL
exit=1
```

Both the byte-mismatch message (`output does not match golden …`) and the ESC-present message
(`output contains an ESC byte …`) appear for `status` in all three environments (NO_COLOR
unset, `=1`, `=banana`) — the two assertions were deliberately made independent (`t.Errorf`,
never a short-circuiting `t.Fatalf`) precisely so a mutation tripping both is reported with both
messages, not whichever check happened to run first.

**Revert:** `git checkout -- internal/cli/status.go`.

**Byte-clean proof:** `git diff --quiet -- internal/cli/status.go` — holds; `git status
--porcelain internal/` — reports only the new, untracked test files this plan adds
(`plain_golden_test.go`, `short_flags_test.go`, `testdata/plain/`), no modified tracked file.

**Green re-run** (verbatim, exit code appended):

```
$ GOTOOLCHAIN=go1.26.6 go test ./internal/cli/ -count=1 -run 'TestPlainGolden$|TestNoColorNonTTYRegression$'
ok  	github.com/seanb4t/codegraph-go/internal/cli	6.258s
exit=0
```

**Verdict:** The golden compare and the zero-ESC assertion are both live and independently
discriminating — a single planted ESC byte on `status`'s plain branch was caught by both, under
`NO_COLOR` unset, `=1`, and `=banana` alike, and the mutation reverted byte-clean.

---

## Family (b) — D-12/CLI-07: a long-only flag turns TestShortFlagsConsistent RED

**Test/guard:** `TestShortFlagsConsistent` (`internal/cli/short_flags_test.go`) — the
positive-counted walk-the-tree table pinning D-12's already-true short-flag invariant for the
nine D-13 Query-the-graph verbs.

**What are we testing, and why?** Whether the walk actually inspects real flags and would catch
a regression that dropped a short form — not merely that it returns green on today's tree,
which a walk that silently inspects zero pairs would also do (rule `84d1gfpywd`).

**Pre-mutation gate:** `git diff --quiet -- internal/cli/callers.go` — clean.

**Mutation applied:** via `perl -0pi -e 's/IntVarP\(&limit, "limit", "l", 0,/IntVar(&limit,
"limit", 0,/'`, replacing `callers`'s `IntVarP` (with shorthand `"l"`) registration with a
shorthand-less `IntVar`:

```diff
--- a/internal/cli/callers.go
+++ b/internal/cli/callers.go
@@ -59,7 +59,7 @@ func newCallersCmd() *cobra.Command {
 	}
 
 	cmd.Flags().StringVarP(&path, "path", "p", "", "repo path (default: cwd)")
-	cmd.Flags().IntVarP(&limit, "limit", "l", 0, "cap on results returned")
+	cmd.Flags().IntVar(&limit, "limit", 0, "cap on results returned")
 	cmd.Flags().BoolVarP(&jsonOut, "json", "j", false, "emit JSON output")
 
 	return cmd
```

**Observed failure** (verbatim, `GOTOOLCHAIN=go1.26.6 go test ./internal/cli/ -count=1 -run
'TestShortFlagsConsistent$' -v`, exit code appended):

```
    short_flags_test.go:68: callers --limit has shorthand "", want -l
    short_flags_test.go:73: inspected 20 (verb, flag) pairs across 9 query verbs
--- FAIL: TestShortFlagsConsistent (0.00s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/cli	0.449s
FAIL
exit=1
```

**Revert:** `git checkout -- internal/cli/callers.go`.

**Byte-clean proof:** `git diff --quiet -- internal/cli/callers.go` — holds; `git status
--porcelain internal/` — reports only the new, untracked test files this plan adds, no modified
tracked file.

**Green re-run** (verbatim):

```
$ GOTOOLCHAIN=go1.26.6 go test ./internal/cli/ -run TestShortFlagsConsistent -count=1 -v
    short_flags_test.go:73: inspected 20 (verb, flag) pairs across 9 query verbs
--- PASS: TestShortFlagsConsistent (0.00s)
ok  	github.com/seanb4t/codegraph-go/internal/cli	0.421s
```

**Verdict:** The walk inspected a positive, non-trivial count (20 pairs across 9 verbs — above
the 18-pair floor and matching D-12's own discovery figure) and caught the exact regression
named — a long-only `--limit` on `callers` — with the failure message naming the verb and flag,
then returned to green after a single-token, byte-clean revert. The walk is not vacuous.

---

## Family (c) — D-15/GRD-13: a planted `colorprofile` import turns TestNoCharmInServeReachablePackages RED

**Test/guard:** `TestNoCharmInServeReachablePackages`
(`internal/cli/present/archtest/import_graph_test.go`) — widened this plan (04-02) from an
exact-match `forbiddenImportPaths` literal list to a `forbiddenImportPathPrefixes` prefix walk
over `charm.land/` and `github.com/charmbracelet/`, in the same commit as `go.mod`'s promotion
of `github.com/charmbracelet/colorprofile` from `// indirect` to direct.

**What are we testing, and why?** Whether the widened prefix-match guard actually catches a
newly-required module (`colorprofile`) hosted under the second, newly-added prefix
(`github.com/charmbracelet/`) reaching a serve-reachable package — not merely that the guard
returns green on today's tree, which a walk with a typo'd or unreachable prefix would also do
(rule `84d1gfpywd`). This is the D-15 discipline: the denylist widening and the `go.mod` change
land in the same commit, and each newly-required module is proven RED before that commit exists.

**Pre-mutation gate:** `git diff --quiet -- internal/query` — clean; `git status --porcelain
internal/query` — empty (no untracked files).

**Mutation applied:** created an UNTRACKED file `internal/query/zz_planted_charm_probe.go`:

```go
package query

import _ "github.com/charmbracelet/colorprofile"
```

**Observed failure** (verbatim, `GOTOOLCHAIN=go1.26.6 go test ./internal/cli/present/archtest/
-count=1 -run 'TestNoCharmInServeReachablePackages$' -v`, exit code appended):

```
    import_graph_test.go:202: package github.com/seanb4t/codegraph-go/internal/query imports github.com/charmbracelet/colorprofile — charm styling must never reach the serve-reachable closure (TUI-01); charm-family usage must be confined to internal/cli/present
--- FAIL: TestNoCharmInServeReachablePackages (0.19s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/cli/present/archtest	0.253s
FAIL
exit=1
```

The failure names both the offending package (`internal/query`) and the exact forbidden import
(`github.com/charmbracelet/colorprofile`) — the prefix walk correctly matched an import under the
newly-added `github.com/charmbracelet/` root, which the pre-widening exact-match list (scoped
only to three `charm.land/...` literals) could never have caught.

**Revert:** `rm -f internal/query/zz_planted_charm_probe.go` (untracked file — no `git checkout`
needed).

**Byte-clean proof:** `git status --porcelain internal/query` — empty.

**Green re-run** (verbatim):

```
$ GOTOOLCHAIN=go1.26.6 go test ./internal/cli/present/archtest/... -count=1 -run 'TestNoCharmInServeReachablePackages$' -v
--- PASS: TestNoCharmInServeReachablePackages (0.17s)
ok  	github.com/seanb4t/codegraph-go/internal/cli/present/archtest	0.240s
```

**Verdict:** The widened prefix-match guard is live and discriminating for the
`github.com/charmbracelet/` root specifically — a planted `colorprofile` import in a guarded
package was caught, named precisely, and the plant reverted byte-clean before the D-15 commit.

**Family (d) — not applicable this plan.** D-15 also calls for the same proof against
`charm.land/fang/v2` "if adopted." The 04-02 fang spike (`04-FANG-VERDICT.md`) concluded
**declined** (`fang.Execute`'s `DefaultErrorHandler` cannot satisfy D-03's exact-once plain
stderr contract on a non-TTY pipe). No `charm.land/fang/v2` line ever reaches `go.mod`, so there
is no newly-required fang module for a Family (d) proof to cover.

---

## Family (e) — CLI-06/D-13: a planted ungrouped command turns TestEveryCommandHasGroupID RED

**Test/guard:** `TestEveryCommandHasGroupID` (`internal/cli/cli_reference_test.go`) — the
positive-counted walk over `root.Commands()` asserting the four D-13 groups are registered in
order and every visible command carries one of their IDs (hidden commands stay groupless).

**What are we testing, and why?** Whether the walk actually inspects real GroupID data and would
catch a regression that dropped a command's group assignment — not merely that it returns green
on today's tree, which a walk with a typo'd map key or an early `return` would also do (rule
`84d1gfpywd`). This is also the guard's own RED-first history: it failed on the ungrouped tree
before `root.AddGroup`/`commandGroups` existed (`root.Groups() has 0 groups, want 4 [query build
agents maintenance]`, captured in the `test(04-08):` commit) — this family proves the SECOND,
narrower failure mode: a single command silently losing its group after the feature already
ships.

**Pre-mutation gate:** `git diff --quiet -- internal/cli/root.go` — clean.

**Mutation applied:** via `perl -0pi -e 's/"telemetry": groupMaintenance,\n//'`, removing
`telemetry`'s entry from the `commandGroups` map in `internal/cli/root.go`:

```diff
--- a/internal/cli/root.go
+++ b/internal/cli/root.go
@@ -83,7 +83,6 @@ var commandGroups = map[string]string{
 	"version":   groupMaintenance,
 	"upgrade":   groupMaintenance,
-	"telemetry": groupMaintenance,
 }
```

**Observed failure** (verbatim, `GOTOOLCHAIN=go1.26.6 go test ./internal/cli/ -count=1 -run
'TestEveryCommandHasGroupID$' -v`, exit code appended):

```
cli_reference_test.go:360: visible command "telemetry" has GroupID "", which is not one of the four registered groups [query build agents maintenance]
    cli_reference_test.go:378: inspected 24 visible commands across 4 groups
--- FAIL: TestEveryCommandHasGroupID (0.00s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/cli	0.459s
FAIL
exit=1
```

The failure names the exact command (`telemetry`) that lost its group assignment, and the
`inspected 24 visible commands` line confirms the walk still traversed the full tree (not a
silently-truncated one) even while one command failed its assertion.

**Revert:** `git checkout -- internal/cli/root.go`.

**Byte-clean proof:** `git diff --quiet -- internal/cli/root.go` — holds.

**Green re-run** (verbatim):

```
$ GOTOOLCHAIN=go1.26.6 go test ./internal/cli/ -count=1 -run 'TestEveryCommandHasGroupID$' -v
    cli_reference_test.go:378: inspected 24 visible commands across 4 groups
--- PASS: TestEveryCommandHasGroupID (0.00s)
ok  	github.com/seanb4t/codegraph-go/internal/cli	0.417s
```

**Verdict:** The walk is live and discriminating for a single command silently losing its
`commandGroups` entry — the planted removal of `telemetry`'s mapping was caught, named
precisely, and the plant reverted byte-clean. Note (per this family's own design instruction):
cobra's `checkCommandGroups` panics at `Execute()` on a `GroupID` that names an UNREGISTERED
group — the opposite direction from this family's proof (a command with NO group at all) — and
is cited here as the backstop for that other direction, not tested (D-00: never test cobra's own
behavior).

---

## Summary

Every guard this phase has introduced or widened so far was demonstrated RED against a
confirmed-applied, byte-cleanly-reverted mutation before being trusted:

| Family | Guard | Mutation | RED confirmed | Reverted clean |
|---|---|---|---|---|
| (a) | `TestPlainGolden` / `TestNoColorNonTTYRegression` | ESC byte in `status.go` | yes (3 subtests, 2 assertions each) | yes |
| (b) | `TestShortFlagsConsistent` | long-only `--limit` on `callers.go` | yes (names `callers --limit`) | yes |
| (c) | `TestNoCharmInServeReachablePackages` | untracked `colorprofile` import in `internal/query` | yes (names `internal/query` and `github.com/charmbracelet/colorprofile`) | yes |
| (e) | `TestEveryCommandHasGroupID` | removed `telemetry`'s entry from `commandGroups` in `root.go` | yes (names `telemetry`) | yes |

`git status --porcelain internal/ cmd/ docs/ go.mod` is empty at the end of plan 04-01 and,
after Family (c)'s revert, immediately before plan 04-02's D-15 commit — no production source
file was left modified by any planted mutation; every plant above was reverted byte-clean before
its respective plan's commit. Family (e) (plan 04-08) is likewise reverted byte-clean before the
`feat(04-08): group the command tree...` commit's follow-on work.
