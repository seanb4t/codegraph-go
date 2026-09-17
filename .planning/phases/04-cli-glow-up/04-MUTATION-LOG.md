# 04-MUTATION-LOG — CLI Glow-up

**Phase:** 04-cli-glow-up
**Date:** 2026-09-17
**Scope:** Two demonstration families for plan 04-01's plain-golden-freeze work — (a) D-16/
CLI-05: a planted ESC byte in `status.go`'s plain branch turns `TestPlainGolden` and
`TestNoColorNonTTYRegression` RED; (b) D-12/CLI-07: a long-only `--limit` flag on `callers`
turns `TestShortFlagsConsistent` RED, naming `callers --limit`.

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

## Summary

Both guards this plan introduces were demonstrated RED against a confirmed-applied,
byte-cleanly-reverted mutation before being trusted:

| Family | Guard | Mutation | RED confirmed | Reverted clean |
|---|---|---|---|---|
| (a) | `TestPlainGolden` / `TestNoColorNonTTYRegression` | ESC byte in `status.go` | yes (3 subtests, 2 assertions each) | yes |
| (b) | `TestShortFlagsConsistent` | long-only `--limit` on `callers.go` | yes (names `callers --limit`) | yes |

`git status --porcelain internal/ cmd/ docs/ go.mod` is empty at the end of this plan — no
production source file was left modified; the only production-file edits were the two planted
mutations above, each reverted byte-clean before the plan's single commit.
