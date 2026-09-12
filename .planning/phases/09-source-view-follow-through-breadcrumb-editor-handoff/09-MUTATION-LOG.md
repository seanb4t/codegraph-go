# 09-MUTATION-LOG — Source View Follow-Through: Breadcrumb & Editor Handoff

**Phase:** 09-source-view-follow-through-breadcrumb-editor-handoff
**Date:** 2026-09-12
**Scope:** Four RED demonstrations, one per gate family this phase introduces: (a) BRW-10's breadcrumb, demonstrated empty/absent against the PRE-FIX build [this plan, 09-03] — (b) the breadcrumb going STALE, demonstrated via a nearest-preceding-symbol mutation — (c) the rejected-path test for BRW-11's editor-link handoff — (d) the editor-link scheme allowlist. Families (b)-(d) are recorded by plan 09-05.

## Pre-mutation cleanliness gate — the convention this log follows

Before every tracked-file mutation, AND before every revert, `git diff --quiet -- <file>` is asserted to exit 0. This proves no pre-existing tracked edit was overwritten by the mutation, and no revert was a destructive blind checkout of someone else's in-flight work. Every family entry below records this gate's result at the point it was checked. Family (a) is an exception per the note in its own section: its RED is the ABSENCE of the fix on a pre-fix commit, not a mutation of otherwise-correct tracked code, so there is no tracked-file mutation and therefore no revert step (07-MUTATION-LOG.md family (a) precedent).

---

## Family (a) — BRW-10: the breadcrumb against the pre-fix build

**Instrument:** `web/scripts/breadcrumb-check.mjs` — a real-Chromium, real-index live gate that:
- Spawns `<binary> ui --no-open --path <repo>` itself (the same lifecycle shape `graph-live-update-check.mjs`'s `startCodegraphUi` establishes).
- Fetches `FileSymbols` for the target file over HTTP directly (`POST <url>/codegraph.ui.v1.UIService/FileSymbols`), independent of the SPA's own client code, as the oracle's data source.
- Re-implements the "innermost containing range" derivation independently in the script itself (`oracleInnermost`, a sort-based algorithm — deliberately a different shape from the SPA's own iterative min-tracking implementation), so a wrong derivation in the SPA cannot agree with itself.
- Navigates to `/browse?file=<path>`, polls for `[data-testid="source-breadcrumb"]`, then drives REAL `page.mouse.wheel` input while reading `data-current-line` / `source-breadcrumb-symbol` (with a poll-until-STABLE read — two consecutive identical reads — to avoid catching a torn intermediate DOM state mid-Svelte-flush).
- Writes a diagnostic record (`corpora/breadcrumb-check.json`) on every exit path via try/finally.

**Target file used for both runs:** `internal/query/node.go` (13 real symbols returned by `FileSymbols`, confirmed via a direct probe of the live endpoint before writing the script — lines 1-32 are outside any symbol, `resolveSourcePath` spans lines 33-79).

**Pre-fix commit:** `aafee950ad35260c29f9d42666d5d5ea374293f9` (`docs(09-02): complete startup editor discovery plan`) — the last commit before this plan's first SPA commit (`test(09-03): add failing source-line splitter and breadcrumb derivation tests`, `9af6f095`). At this commit, `SourcePane.svelte` still renders the single `{@html}` blob with no per-line DOM and no breadcrumb bar at all (09-RESEARCH.md Pitfall 3).

**No tracked-file mutation, no revert step.** Unlike families (b)-(d), this RED is the ABSENCE of the fix on a prior commit, not a mutation applied to otherwise-correct code on the current tree — there is nothing to mutate and nothing to revert (07-MUTATION-LOG.md family (a) precedent, cited by this plan's own Task 3 action text).

**Exact commands run:**

```
$ PREFIX=aafee950ad35260c29f9d42666d5d5ea374293f9
$ mkdir -p /tmp/gsd-09-03-prefix
$ git worktree add /tmp/gsd-09-03-prefix/prefix-09 "$PREFIX"
$ cd /tmp/gsd-09-03-prefix/prefix-09
$ GOTOOLCHAIN=go1.26.6 go build -o /tmp/codegraph-prefix ./cmd/codegraph
$ cd /Volumes/Code/github.com/seanb4t/codegraph-go
$ node web/scripts/breadcrumb-check.mjs --binary /tmp/codegraph-prefix --out /tmp/breadcrumb-prefix.json
$ echo "EXIT_CODE=$?"
```

**RED — pasted verbatim (exit 1):**

```
breadcrumb-check: oracle has 13 symbols for internal/query/node.go
breadcrumb-check: wrote /tmp/breadcrumb-prefix.json — success=false
breadcrumb-check: fatal — breadcrumb-check: FAILED — breadcrumbPresent=false error=pollUntil: condition did not become true within 15000ms
Error: breadcrumb-check: FAILED — breadcrumbPresent=false error=pollUntil: condition did not become true within 15000ms
    at main (file:///Volumes/Code/github.com/seanb4t/codegraph-go/web/scripts/breadcrumb-check.mjs:372:9)
    at process.processTicksAndRejections (node:internal/process/task_queues:104:5)
EXIT_CODE=1
```

**RED — recorded diagnostic (`/tmp/breadcrumb-prefix.json`, pasted verbatim):**

```json
{
  "schemaVersion": 1,
  "file": "internal/query/node.go",
  "binaryPath": "/tmp/codegraph-prefix",
  "repoPath": "/Volumes/Code/github.com/seanb4t/codegraph-go",
  "symbolCount": 13,
  "breadcrumbPresent": false,
  "observations": [],
  "nonEmptyObservations": 0,
  "emptyObservations": 0,
  "pageErrorCount": null,
  "pageErrors": [],
  "browserIdentity": {
    "name": "chromium",
    "version": "151.0.7922.34"
  },
  "error": "pollUntil: condition did not become true within 15000ms",
  "success": false,
  "errorStack": "Error: pollUntil: condition did not become true within 15000ms\n    at pollUntil (file:///Volumes/Code/github.com/seanb4t/codegraph-go/web/scripts/breadcrumb-check.mjs:52:10)\n    at async main (file:///Volumes/Code/github.com/seanb4t/codegraph-go/web/scripts/breadcrumb-check.mjs:251:4)"
}
```

`breadcrumbPresent: false` — the pre-fix build has no `source-breadcrumb` element at all; the poll for it exhausts its 15s budget and the script exits 1, exactly the "stale or empty against the pre-fix build" RED success criterion 1 requires.

**Worktree cleanup confirmed:**

```
$ git worktree remove --force /tmp/gsd-09-03-prefix/prefix-09
$ git worktree list
/Volumes/Code/github.com/seanb4t/codegraph-go                                           ff2e5a64 [gsd/v0.13.0-guard-hardening-ui-follow-through]
/Volumes/Code/github.com/seanb4t/codegraph-go/.claude/worktrees/agent-aebfa7de95041ec86 524fe615 [worktree-agent-aebfa7de95041ec86]
```

(The second entry is a pre-existing, unrelated agent worktree from parallel phase execution — no `prefix-09` entry remains.)

**GREEN — re-run at HEAD against the real `./codegraph` binary (built from the committed `web/build/` via `task web:build && task web:drift && GOTOOLCHAIN=go1.26.6 task build:release`), pasted verbatim (exit 0):**

```
breadcrumb-check: oracle has 13 symbols for internal/query/node.go
breadcrumb-check: source-breadcrumb element present
breadcrumb-check: observation {"line":1,"shown":null,"expected":null,"empty":true,"agrees":true}
breadcrumb-check: observation {"line":1,"shown":null,"expected":null,"empty":true,"agrees":true}
breadcrumb-check: observation {"line":11,"shown":null,"expected":null,"empty":true,"agrees":true}
breadcrumb-check: observation {"line":26,"shown":null,"expected":null,"empty":true,"agrees":true}
breadcrumb-check: observation {"line":41,"shown":"resolveSourcePath","expected":"resolveSourcePath","empty":false,"agrees":true}
breadcrumb-check: wrote /Volumes/Code/github.com/seanb4t/codegraph-go/corpora/breadcrumb-check.json — success=true
```

**GREEN — recorded diagnostic (`corpora/breadcrumb-check.json`, committed alongside this plan):**

```json
{
  "schemaVersion": 1,
  "file": "internal/query/node.go",
  "binaryPath": "/Volumes/Code/github.com/seanb4t/codegraph-go/codegraph",
  "repoPath": "/Volumes/Code/github.com/seanb4t/codegraph-go",
  "symbolCount": 13,
  "breadcrumbPresent": true,
  "observations": [
    { "line": 1, "shown": null, "expected": null, "empty": true, "agrees": true },
    { "line": 1, "shown": null, "expected": null, "empty": true, "agrees": true },
    { "line": 11, "shown": null, "expected": null, "empty": true, "agrees": true },
    { "line": 26, "shown": null, "expected": null, "empty": true, "agrees": true },
    { "line": 41, "shown": "resolveSourcePath", "expected": "resolveSourcePath", "empty": false, "agrees": true }
  ],
  "nonEmptyObservations": 1,
  "emptyObservations": 4,
  "pageErrorCount": 0,
  "pageErrors": [],
  "browserIdentity": { "name": "chromium", "version": "151.0.7922.34" },
  "error": null,
  "success": true
}
```

The breadcrumb correctly names the innermost `FileSymbols` range containing the first fully visible line at real scroll positions (an honest empty state for lines 1/11/26, which are genuinely outside any symbol — `resolveSourcePath` starts at line 33 — and the correct symbol name once scrolled to line 41, inside `resolveSourcePath`'s 33-79 range), verified against an oracle computed independently of the SPA's own code, at both states (empty and non-empty), with zero page errors.

**Debugging note, recorded honestly.** The first live-gate implementation used a fixed two-`requestAnimationFrame` settle after detecting `data-current-line` change, on the theory that `data-current-line` and the crumb-derived `data-empty`/button text (two separate template bindings sharing the `currentLine` dependency) commit within the same Svelte flush. This was insufficient — reproduced twice: a tight poll caught `data-current-line="41"` already updated while `data-empty` was still `"true"` (a torn read of a flush still in progress), most likely because that flush is itself scheduled inside the scroll-tracking effect's own `requestAnimationFrame` throttle, adding at least one more frame of latency than the fixed count assumed. The instrument was hardened to poll-until-STABLE (read the full observation tuple repeatedly until two consecutive reads agree) rather than poll-until-changed-plus-fixed-wait — a torn read is by construction unstable, since the second binding's update changes the tuple again on the very next poll. Confirmed stable across 3 consecutive full runs after the fix.

---

## Family (b) — BRW-10: a STALE breadcrumb (nearest-preceding-symbol lie)

**Instrument:** `web/scripts/breadcrumb-check.mjs` (the same live gate as family (a)), driving the real `innermostSymbolAt` function shipped in `web/src/lib/breadcrumb.ts` against the real, independent `oracleInnermost` the script itself implements.

**Pre-mutation cleanliness gate:**
```
$ git diff --quiet -- web/src/lib/breadcrumb.ts web/build; echo $?
0
```

**Mutation applied.** `innermostSymbolAt` rewritten to ignore `endLine` entirely and return, among symbols with `startLine <= line`, the one with the greatest `startLine` — the nearest-preceding-symbol lie D-03 forbids:

```diff
 export function innermostSymbolAt<T extends SymbolRange>(
 	symbols: readonly T[],
 	line: number
 ): T | null {
+	// MUTATION (09-05 family (b), BRW-10 STALE demonstration): ignore
+	// endLine entirely and return the nearest-preceding symbol — the
+	// exact lie D-03 forbids. Reverted via `git checkout --` after the
+	// live gate observes it fail.
 	let best: T | null = null;
-	let bestSpan = Infinity;
 	let bestStart = -Infinity;

 	for (const s of symbols) {
 		const start = s.startLine;
-		const end = Math.max(start, s.endLine);
-		if (line < start || line > end) continue;
-
-		const span = end - start;
-		if (best === null || span < bestSpan || (span === bestSpan && start >= bestStart)) {
+		if (start > line) continue;
+		if (best === null || start >= bestStart) {
 			best = s;
-			bestSpan = span;
 			bestStart = start;
 		}
 	}

 	return best;
 }
```

**Confirmed applied** (grep, before running):
```
$ rg -n 'nearest-preceding symbol' web/src/lib/breadcrumb.ts
44:	// endLine entirely and return the nearest-preceding symbol — the
```

**Rebuild** (per this family's own key-link: rebuild the SPA and the binary before driving the live gate):
```
$ task web:build            # wrote web/build/.build-manifest (source-files=114, output-files=32)
$ GOTOOLCHAIN=go1.26.6 task build:release
```

**Why `internal/query/node.go` (family (a)'s default target) does NOT expose this mutation.** The live gate's scroll loop stops (`observations.length >= 3 && nonEmptyObservations >= 1 && emptyObservations >= 1`) the instant it first records a TRUE (oracle) non-empty observation — which, for `node.go`, happens at line 41, squarely inside `resolveSourcePath`'s wide 33-79 range. Both the oracle and the mutated function agree there (41 is genuinely inside that range), so the run completes green without ever reaching `resolveSourcePath`'s trailing gap (lines 80-90, before `ValidateRepoRelativePath` starts at 91) where the lie would surface. Retargeting the SAME script at `internal/cli/editorurl.go` (`--file`, no other change) exposes it: that file's first two symbols are single-line (`editorURLEnvVar` line 22, `noEditorURLEnvVar` line 23) followed by a real gap (24-28) before `discoveredEditor` (29-33) — the scroll step from line 11 to line 26 skips clean over both single-line symbols without ever landing inside either, so at line 26 the oracle correctly says "no symbol contains line 26" (`null`) while the mutated function, evaluating the full symbol list fresh at every read (it is not incremental/stateful), finds `noEditorURLEnvVar` (`startLine=23 <= 26`, the latest-starting symbol satisfying that condition) and reports it — the nearest-preceding lie, live, in a real browser.

**Exact command run:**
```
$ node web/scripts/breadcrumb-check.mjs --file internal/cli/editorurl.go --out "$TMPDIR/breadcrumb-stale.json"
$ echo "EXIT_CODE=$?"
```

**RED — pasted verbatim (exit 1):**
```
breadcrumb-check: oracle has 6 symbols for internal/cli/editorurl.go
breadcrumb-check: source-breadcrumb element present
breadcrumb-check: observation {"line":1,"shown":null,"expected":null,"empty":true,"agrees":true}
breadcrumb-check: observation {"line":1,"shown":null,"expected":null,"empty":true,"agrees":true}
breadcrumb-check: observation {"line":11,"shown":null,"expected":null,"empty":true,"agrees":true}
breadcrumb-check: observation {"line":26,"shown":"noEditorURLEnvVar","expected":null,"empty":false,"agrees":false}
breadcrumb-check: observation {"line":41,"shown":"editorResolveInputs","expected":"editorResolveInputs","empty":false,"agrees":true}
breadcrumb-check: editor-link href=vscode://file//Volumes/Code/github.com/seanb4t/codegraph-go/internal/cli/editorurl.go:1:1
breadcrumb-check: gutterLinkCount=131
breadcrumb-check: gutterClickIssuedRpc=true
breadcrumb-check: wrote /var/folders/_b/3hyf5qvs62q0wh2vyh856z580000gn/T/gsd-09-05/breadcrumb-stale.json — success=false
breadcrumb-check: fatal — breadcrumb-check: FAILED — breadcrumbPresent=true error=null
Error: breadcrumb-check: FAILED — breadcrumbPresent=true error=null
    at main (file:///Volumes/Code/github.com/seanb4t/codegraph-go/web/scripts/breadcrumb-check.mjs:449:9)
    at process.processTicksAndRejections (node:internal/process/task_queues:104:5)
EXIT_CODE=1
```

At line 26 the oracle expects `null` (no symbol contains it) and the bar shows `"noEditorURLEnvVar"` — a symbol that ENDED at line 23, three lines earlier — with `"agrees": false`, `"empty": false` (the mutated code claims a non-empty state where the true answer is empty), and the script's own overall-success check fails (`success=false`, exit 1). This is success criterion 1's "stale" half; family (a) was the "empty" half.

**Revert:**
```
$ git checkout -- web/src/lib/breadcrumb.ts web/build
```

Vite's content-hashed output filenames mean the mutated rebuild's `web:build` run produced several NEWLY-hashed, untracked chunk files that `git checkout --` (which only restores TRACKED file contents) does not remove; these were deleted individually with `rm -f` (never `git clean`, which is forbidden inside this repo's git-integration discipline) before the post-revert gate below.

**Post-revert gate:**
```
$ git diff --quiet -- web/src/lib/breadcrumb.ts web/build; echo $?
0
$ git status --porcelain --untracked-files=all
(empty)
```

**Rebuild the binary from the restored tree** (`web/build` is already the committed bytes — no `web:build` re-run needed):
```
$ GOTOOLCHAIN=go1.26.6 task build:release
```

**GREEN — re-run against the committed record path (default `--file`/`--out`), pasted verbatim (exit 0):**
```
breadcrumb-check: oracle has 13 symbols for internal/query/node.go
breadcrumb-check: source-breadcrumb element present
breadcrumb-check: observation {"line":1,"shown":null,"expected":null,"empty":true,"agrees":true}
breadcrumb-check: observation {"line":1,"shown":null,"expected":null,"empty":true,"agrees":true}
breadcrumb-check: observation {"line":11,"shown":null,"expected":null,"empty":true,"agrees":true}
breadcrumb-check: observation {"line":26,"shown":null,"expected":null,"empty":true,"agrees":true}
breadcrumb-check: observation {"line":41,"shown":"resolveSourcePath","expected":"resolveSourcePath","empty":false,"agrees":true}
breadcrumb-check: editor-link href=vscode://file//Volumes/Code/github.com/seanb4t/codegraph-go/internal/query/node.go:1:1
breadcrumb-check: gutterLinkCount=418
breadcrumb-check: gutterClickIssuedRpc=true
breadcrumb-check: wrote /Volumes/Code/github.com/seanb4t/codegraph-go/corpora/breadcrumb-check.json — success=true
```
`git status --porcelain corpora/breadcrumb-check.json` reported no diff — the re-run reproduced the exact committed record byte-for-byte.

---

## Family (c) — BRW-11: the rejected-path test with the confinement gate removed

**Test name:** `TestGetEditorLinkPathConfinementAtRPCBoundary` (`internal/uiserver/editorlink_test.go`).

**Pre-mutation cleanliness gate:**
```
$ git diff --quiet -- internal/uiserver/editorlink.go; echo $?
0
```

**Mutation applied.** The `eng.ValidateRepoRelativePath(path)` call and its `if verr != nil { … }` block deleted from `GetEditorLink`'s `withEngine` closure, so the caller-supplied path flows straight into the `filepath.Join`. The now-unused `"io/fs"` import (its only use was `fs.ErrNotExist` inside the deleted block) was removed in the SAME mutation so the package still compiles — a build-breaking unused import would produce a compile error rather than the intended assertion failures, which this family's own convention (RED must be an assertion failure, never a compile error) forbids:

```diff
 import (
 	"context"
 	"errors"
 	"fmt"
-	"io/fs"
 	"net/url"
 ...
 		path := req.Msg.GetPath()
-		if verr := eng.ValidateRepoRelativePath(path); verr != nil {
-			if errors.Is(verr, fs.ErrNotExist) {
-				classifiedErr = connect.NewError(connect.CodeInvalidArgument, fmt.Errorf(
-					"path %q does not exist in the working tree — it may have existed at the indexed commit and been deleted or renamed since; re-index or check the commit history",
-					path,
-				))
-				return nil
-			}
-			return verr
-		}
 
 		root, err := filepath.Abs(s.repoPath)
```

**Confirmed applied** (grep, before running): zero occurrences of the actual call site, `go build` still succeeds:
```
$ rg -c 'eng\.ValidateRepoRelativePath\(' internal/uiserver/editorlink.go
(no match — exit 1, count 0)
$ GOTOOLCHAIN=go1.26.6 go build ./internal/uiserver/...
(no output — success)
```
(Three remaining `ValidateRepoRelativePath` hits are prose in doc comments naming the design, not the deleted call — confirmed by line inspection.)

**Observed failure (pasted verbatim, `GOTOOLCHAIN=go1.26.6 go test -count=1 -v -run TestGetEditorLinkPathConfinementAtRPCBoundary ./internal/uiserver/...`, exit 1):**
```
    editorlink_test.go:112: GetEditorLink(path="../outside.txt") succeeded, want a refusal
    editorlink_test.go:112: GetEditorLink(path="/etc/passwd") succeeded, want a refusal
    editorlink_test.go:112: GetEditorLink(path="") succeeded, want a refusal
    editorlink_test.go:133: GetEditorLink(path="escape-link/secret.txt") succeeded, want a refusal
--- FAIL: TestGetEditorLinkPathConfinementAtRPCBoundary (0.28s)
    --- PASS: TestGetEditorLinkPathConfinementAtRPCBoundary/in-repo_control (0.04s)
    --- FAIL: TestGetEditorLinkPathConfinementAtRPCBoundary/escape (0.04s)
    --- FAIL: TestGetEditorLinkPathConfinementAtRPCBoundary/absolute (0.03s)
    --- FAIL: TestGetEditorLinkPathConfinementAtRPCBoundary/empty (0.03s)
    --- FAIL: TestGetEditorLinkPathConfinementAtRPCBoundary/symlink-escape (0.04s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/uiserver	0.683s
FAIL
```

Every path-rejection subtest (`escape`, `absolute`, `empty`, `symlink-escape`) fails on "succeeded, want a refusal" — exactly the clause this family exists to watch fail — while `in-repo_control` still passes, proving the failure is specific to the confinement gate's removal, not a broken test harness.

**Revert:**
```
$ git checkout -- internal/uiserver/editorlink.go
```

**Post-revert gate:**
```
$ git diff --quiet -- internal/uiserver/editorlink.go; echo $?
0
```

**Green re-run (pasted verbatim, same command, exit 0):**
```
--- PASS: TestGetEditorLinkPathConfinementAtRPCBoundary (0.29s)
    --- PASS: TestGetEditorLinkPathConfinementAtRPCBoundary/in-repo_control (0.03s)
    --- PASS: TestGetEditorLinkPathConfinementAtRPCBoundary/escape (0.03s)
    --- PASS: TestGetEditorLinkPathConfinementAtRPCBoundary/absolute (0.03s)
    --- PASS: TestGetEditorLinkPathConfinementAtRPCBoundary/empty (0.03s)
    --- PASS: TestGetEditorLinkPathConfinementAtRPCBoundary/symlink-escape (0.04s)
ok  	github.com/seanb4t/codegraph-go/internal/uiserver	0.657s
```

---

## Family (d) — BRW-11/BRW-12: the scheme allowlist poisoned (`javascript` admitted)

**Test names:** `TestEditorTemplateSchemeAllowlist`, `TestGetEditorLinkAvailabilityStatesAreAnswersNotErrors` (`internal/uiserver/editorlink_test.go`).

**Pre-mutation cleanliness gate (freshly re-checked here, after family (c)'s revert, not assumed from it):**
```
$ git diff --quiet -- internal/uiserver/editorlink.go; echo $?
0
```

**Mutation applied.** `"javascript": {}` added to `editorURLSchemes` (D-13's positive allowlist), growing it from 15 to 16 members:
```diff
 	"jetbrains":       {},
 	"http":            {},
 	"https":           {},
+	"javascript":      {},
 }
```

**Confirmed applied** (grep, before running):
```
$ rg -n '"javascript":' internal/uiserver/editorlink.go
114:	"javascript":      {},
```

**Observed failure (pasted verbatim, `GOTOOLCHAIN=go1.26.6 go test -count=1 -v -run 'TestEditorTemplateSchemeAllowlist|TestGetEditorLinkAvailabilityStatesAreAnswersNotErrors' ./internal/uiserver/...`, exit 1):**
```
    editorlink_test.go:202: len(editorURLSchemes) = 16, want exactly 15
--- FAIL: TestEditorTemplateSchemeAllowlist (0.00s)
    editorlink_test.go:338: availability = EDITOR_LINK_AVAILABILITY_BUILDABLE, want TEMPLATE_INVALID
--- FAIL: TestGetEditorLinkAvailabilityStatesAreAnswersNotErrors (0.32s)
    --- PASS: TestGetEditorLinkAvailabilityStatesAreAnswersNotErrors/unconfigured (0.09s)
    --- PASS: TestGetEditorLinkAvailabilityStatesAreAnswersNotErrors/disabled_by_operator (0.07s)
    --- FAIL: TestGetEditorLinkAvailabilityStatesAreAnswersNotErrors/template_invalid_via_override (0.08s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/uiserver	0.677s
FAIL
```

Both independent guards fire, as the plan requires: the count guard (`TestEditorTemplateSchemeAllowlist`, line 201's `t.Fatalf` on `len(editorURLSchemes) != 15`) fails FIRST and — being a `t.Fatalf` at the top of the test function, before any `t.Run` subtest — short-circuits that same test's own `"javascript"` membership subtest, so that specific subtest is never reached in this run (recorded honestly, not papered over). The membership guard's failure is nonetheless directly demonstrated by the SECOND test: `TestGetEditorLinkAvailabilityStatesAreAnswersNotErrors/template_invalid_via_override` (which sets `override := "javascript:{path}"`, `editorlink_test.go:332`) fails with `availability = EDITOR_LINK_AVAILABILITY_BUILDABLE, want TEMPLATE_INVALID` — i.e. `javascript:{path}` is now BUILDABLE, precisely the fact the plan's action text names. Both failure lines are pasted above and both independently prove D-13's allowlist was defeated.

**Revert:**
```
$ git checkout -- internal/uiserver/editorlink.go
```

**Post-revert gate:**
```
$ git diff --quiet -- internal/uiserver/editorlink.go; echo $?
0
```

**Green re-run (pasted verbatim, same command plus the confinement test for the phase-wide check below, exit 0):**
```
--- PASS: TestGetEditorLinkPathConfinementAtRPCBoundary (0.28s)
    --- PASS: TestGetEditorLinkPathConfinementAtRPCBoundary/in-repo_control (0.04s)
    --- PASS: TestGetEditorLinkPathConfinementAtRPCBoundary/escape (0.03s)
    --- PASS: TestGetEditorLinkPathConfinementAtRPCBoundary/absolute (0.03s)
    --- PASS: TestGetEditorLinkPathConfinementAtRPCBoundary/empty (0.03s)
    --- PASS: TestGetEditorLinkPathConfinementAtRPCBoundary/symlink-escape (0.03s)
--- PASS: TestEditorTemplateSchemeAllowlist (0.00s)
    --- PASS: TestEditorTemplateSchemeAllowlist/javascript (0.00s)
    --- PASS: TestEditorTemplateSchemeAllowlist/javascript#01 (0.00s)
    --- PASS: TestEditorTemplateSchemeAllowlist/data (0.00s)
    --- PASS: TestEditorTemplateSchemeAllowlist/blob (0.00s)
    --- PASS: TestEditorTemplateSchemeAllowlist/file (0.00s)
    --- PASS: TestEditorTemplateSchemeAllowlist/ftp (0.00s)
    --- PASS: TestEditorTemplateSchemeAllowlist/no_path_placeholder (0.00s)
    --- PASS: TestEditorTemplateSchemeAllowlist/unknown_placeholder (0.00s)
    --- PASS: TestEditorTemplateSchemeAllowlist/unbalanced_brace (0.00s)
    --- PASS: TestEditorTemplateSchemeAllowlist/space (0.00s)
    --- PASS: TestEditorTemplateSchemeAllowlist/control_byte (0.00s)
    --- PASS: TestEditorTemplateSchemeAllowlist/too_long (0.00s)
ok  	github.com/seanb4t/codegraph-go/internal/uiserver	0.647s
```

---

## Closing

### Phase-wide byte-clean proof

**Porcelain status across the three mutated paths, at the point all four families' reverts are complete (pasted verbatim):**
```
$ git status --porcelain --untracked-files=all
(empty)
$ git diff --quiet -- web/src/lib/breadcrumb.ts web/build internal/uiserver/editorlink.go; echo $?
0
```

**Confinement call and poisoned-scheme key both absent at HEAD (pasted verbatim):**
```
$ rg -o 'ValidateRepoRelativePath' internal/uiserver/editorlink.go | wc -l | tr -d ' '
4
$ rg -o '"javascript"' internal/uiserver/editorlink.go | wc -l | tr -d ' '
0
```
The `ValidateRepoRelativePath` count of 4 is the restored real call plus its surrounding doc-comment mentions — not zero, because the gate is genuinely back in place (the task's own post-revert `<verify>` requires `-ge 1`, precisely to prove restoration rather than deletion).

**Phase-close line (recorded by Task 3):** see below.

### Non-vacuity assertion

Three instruments were watched fail on the specific assertion each exists for, none rewritten to force a pass: (b) `breadcrumb-check.mjs`'s live-Chromium observation loop disagreed with its own independent oracle at a real post-symbol gap (`shown: "noEditorURLEnvVar"` vs `expected: null`), demonstrating the STALE half of BRW-10 success criterion 1 that family (a) (the EMPTY half) did not cover; (c) `TestGetEditorLinkPathConfinementAtRPCBoundary` failed on exactly the clause success criterion 2 names ("fails if the request succeeds") for all four rejection cases while its in-repo control kept passing; (d) `TestEditorTemplateSchemeAllowlist` and `TestGetEditorLinkAvailabilityStatesAreAnswersNotErrors` independently proved D-13's allowlist defeated — the count guard and the membership guard, as two separately-failing tests since a `t.Fatalf` prevented both guards from firing within the SAME test run (recorded honestly above, per 08-MUTATION-LOG.md family (d)'s precedent for reporting exactly what was observed).

### Family-to-requirement table

| Family | Requirement | Test/instrument | Mutated file(s) | ROADMAP criterion |
|--------|-------------|------------------|------------------|--------------------|
| (a) | BRW-10 SC1 (empty) | `breadcrumb-check.mjs` vs pre-fix binary `aafee950` | none (pre-fix commit, no mutation) | Criterion 1, empty half — **discharged** |
| (b) | BRW-10 SC1 (stale) | `breadcrumb-check.mjs` vs `oracleInnermost`, `--file internal/cli/editorurl.go` | `web/src/lib/breadcrumb.ts`, `web/build` | Criterion 1, stale half — **discharged** |
| (c) | BRW-11 SC2 | `TestGetEditorLinkPathConfinementAtRPCBoundary` | `internal/uiserver/editorlink.go` | Criterion 2 — **discharged** |
| (d) | BRW-11/BRW-12 D-13 | `TestEditorTemplateSchemeAllowlist`, `TestGetEditorLinkAvailabilityStatesAreAnswersNotErrors` | `internal/uiserver/editorlink.go` | D-13's allowlist guard — **discharged** |

All four families this phase committed to are discharged: BRW-10's success criterion 1 now has BOTH its empty (a) and stale (b) halves watched fail; BRW-11's rejected-path test (c) and BRW-11/BRW-12's scheme allowlist (d) are both watched fail against the real, shipped implementation.

### Phase-close run (Task 3)

Recorded 2026-09-12 (phase-close date) at HEAD `416b69551068e40c9bb44fd550371fdcd0f809f8` (the commit adding `09-SECURITY.md`, immediately before this line's own commit): `proto:drift`, `web:drift`, `test:unit`, `web:test`, the live gate, and `TestEditorPresetsAreExactlyThreeAndNameNoZed` all green on a clean tree; `web:components:drift` failed for a pre-existing, already-tracked, out-of-scope reason (`.planning/WINDOWS.md` entry 31 — local pnpm/Corepack toolchain mismatch against 8 vendored shadcn-svelte files this phase never touched). Full transcript in `09-05-SUMMARY.md`'s `## Phase close` section.

### Gate-hygiene addendum (09-06)

`pnpm -C web check` (svelte-check, strict TS) reported 2 errors at `web/src/lib/components/browse/SourcePane.svelte:446:31` ("'activeClient.getEditorLink' is possibly 'undefined'") from commit `28d5d795` (code-review iteration 1, CR-01 corrective re-probe) through the verified HEAD `3c0f4bc7`, and was fixed in commit `1e92d0d6` by capturing the guard-narrowed `getEditorLink` once (`const getEditorLink = activeClient.getEditorLink.bind(activeClient)`, declared immediately after the effect's top-of-body guard) and calling it from both the initial probe and the CR-01 re-probe — a narrowing-hygiene change with no behavioural delta. The CR-01 regression test ("sends a pre-seeded PRESET override as the corrected template via a re-probe (CR-01)" in `web/tests/source-pane-editor-link.test.ts`) passed unchanged before and after, and `task web:build` + `task web:drift` were re-run green in the same commit: `web:drift: source half MATCH (114 files, fa0d79480ce1f3849faabeec63c72fe12227ab0ffaa7a9c6a2b8bfc815d3125e)` and `web:drift: output half MATCH (32 files, d772f7b81e4f6bb8c316f7707c0eed884f24dcf4841296bcc61796530e08856b)`.

Why nobody noticed: 09-05 Task 3's phase-close gate list (`proto:drift`, `web:drift`, `web:components:drift`, `test:unit`, `web:test`, the live gate, `TestEditorPresetsAreExactlyThreeAndNameNoZed`) did not include `pnpm -C web check`; the code-review re-verification passes (09-REVIEW.md iterations 2-3, which landed WR-01..WR-05 on top of CR-01) re-ran `go build`/`go vet`/`go test`/`pnpm -C web test`/`task web:drift` but never svelte-check; and 09-04-SUMMARY.md's pasted `0 ERRORS` transcript predates `28d5d795`, so no SUMMARY was wrong on its own terms — the gate was only re-executed by 09-VERIFICATION.md. Note in passing the casing landmine: svelte-check prints its MACHINE summary (`… 0 ERRORS 0 WARNINGS`) when piped, so a case-sensitive `rg -q '0 errors'` gate cannot match it — gates must be case-insensitive and anchored on the exit code.

Forward rule for this phase's record (not a plan edit — 09-05-PLAN.md stays frozen): `pnpm -C web check` belongs in every phase-close gate enumeration for any phase that touches `web/src`, run alongside `task web:test` and `task web:drift`, and it must be re-run after every late-landing code-review fix commit, not only at each plan's own GREEN capture. Dated 2026-09-12 at HEAD `1e92d0d6` (the Task 1 fix commit; this addendum's own commit follows immediately after).

