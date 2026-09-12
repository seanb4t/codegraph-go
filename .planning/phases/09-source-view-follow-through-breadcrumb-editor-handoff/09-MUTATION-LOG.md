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

