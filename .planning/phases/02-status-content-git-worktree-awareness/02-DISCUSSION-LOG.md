# Phase 2: status Content & Git/Worktree Awareness - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-07-15
**Phase:** 2-status-content-git-worktree-awareness
**Mode:** `--auto` (Claude auto-selected the recommended option for every question; no interactive prompts)
**Areas discussed:** TS ground-truth availability, Worktree detection algorithm, Status layout, DB size semantics, files-by-language, Live signals (STAT-03), Notice delivery, Detection wiring & caching, Test fixtures

---

## TS Ground-Truth Availability

| Option | Description | Selected |
|--------|-------------|----------|
| Frozen goldens only | Treat the TS dist as gone (per memory `9zt8afrs8k`) and use `testdata/golden/corpus/*/status.json` as the sole oracle | |
| White-box from live dist | Locate and read the live TS 1.3.1 implementation for verbatim constants/strings, as Phase 1's D-01 did | ✓ |

**Choice:** White-box from live dist (D-01).
**Notes:** Memory `9zt8afrs8k` recorded "the live TS 1.3.1 dist VANISHED mid-execution (only .d.ts left)". Verified false this session: the top-level `dist/` holds only `.d.ts` stubs, but the real 195-file `.js` implementation was **relocated** to the platform sub-package `node_modules/@colbymchenry/codegraph-darwin-arm64/lib/dist/`, and `codegraph --version` → `1.3.1` still works. This changes the phase's evidence base from archaeology to direct source extraction. Memory superseded.

---

## Worktree Detection Algorithm

| Option | Description | Selected |
|--------|-------------|----------|
| Literal requirement reading | Compare `--show-toplevel` vs `--git-common-dir`; flag when they differ | |
| Verbatim 4-gate cascade port | Port `sync/worktree.js`'s full `detectWorktreeIndexMismatch` cascade, including the index-root-is-a-worktree-root gate and the inverted common-dir suppression | ✓ |

**Choice:** Verbatim 4-gate cascade (D-02).
**Notes:** The requirement text's shorthand is materially misleading. Reading the TS source showed gate 4's polarity is **inverted** from the naive reading: differing common dirs ⇒ *suppress* the warning (submodule/embedded clone = a different repo the parent index already covers, TS #1031/#1033); a genuine borrowed worktree **shares** a common dir. Gate 3 (`gitWorktreeRoot(indexRoot) != indexRoot ⇒ no mismatch`) is what actually kills the monorepo-subdir and non-git false positives that WORK-03 demands. Auto-selected the faithful port because the literal reading would produce both false positives and false negatives.

---

## Status Human-Output Layout

| Option | Description | Selected |
|--------|-------------|----------|
| Adopt TS's sectioned plain-text layout now | Port sections/ordering/wording; leave only ANSI color for Phase 6 | ✓ |
| Extend the terse key=value line | Append `dbSize=… nodesByKind=…` to the existing single-line output | |
| Defer layout entirely to Phase 6 | Land only the JSON/data changes now | |

**Choice:** Adopt TS's sectioned layout now (D-09).
**Notes:** Our current `backend=… files=… stale=…` one-liner cannot carry two multi-row breakdowns. Extending it would guarantee a second rewrite at Phase 6 ("Rendering Seam & Pretty status/files"). Phase 6 owns *color*; Phase 2 owns *content and structure* — this split lets Phase 6 paint an already-correct layout. `Journal:` dropped (no Pebble analog); `Backend:` renders the Go-truthful `pebble`.

---

## DB Size Semantics (STAT-01)

| Option | Description | Selected |
|--------|-------------|----------|
| Recursive store-dir byte sum, presence+plausibility assertion | Walk `.codegraph/store/`; assert key present / int / >0 / MB well-formed as a documented divergence | ✓ |
| Byte-stable assertion against a golden | Pin an exact dbSizeBytes value in the fixtures | |
| Keep it stripped | Leave the volatile-field strip in place | |

**Choice:** Recursive sum + presence/plausibility (D-07, D-08).
**Notes:** `testdata/golden/README.md` strips `dbSizeBytes` as volatile ("not guaranteed byte-stable across reindexes even of identical source"), and `golden_test.go` encodes that as an invariant. That rationale is *stronger* for Pebble — LSM compaction makes the on-disk total genuinely nondeterministic. So the TS golden's strip stays (the frozen oracle can't supply a stable value) and the Go side asserts shape, not bytes, under Phase 1's D-02 allowed-divergence regime. STAT-01's "reverse the strip" is therefore scoped to the Go assertion + the README table, not to the TS oracle.

---

## files-by-language (STAT-02)

| Option | Description | Selected |
|--------|-------------|----------|
| Render-only (per requirement text) | Surface data "already computed in StatusResult" | |
| Add `FilesByLanguage map[string]int64` | New aggregation; derive the existing `Languages []string` from it | ✓ |

**Choice:** New field + derive the list (D-05).
**Notes:** The requirement's parenthetical ("data already computed — surface it") is **half wrong**. True for `NodesByKind` (real counts, already scanned); false for languages — `StatusResult.Languages` is a bare `[]string` with no counts. TS does the same in JSON: it derives the flat list *from* `filesByLanguage` and discards the counts (`Object.entries(...).filter(([,c]) => c>0).map(([lang]) => lang)`); the counts survive only in rendered output. So this is a real scan/aggregation task, not a render tweak — flagged so the planner sizes it correctly.

---

## Live Signals (STAT-03)

| Option | Description | Selected |
|--------|-------------|----------|
| Verify + surface existing live signals | `stale`/`reindexRecommended` already live from v0.1 D-04a; render them and prove reachability on both surfaces | ✓ |
| Rebuild the staleness computation | Write a fresh pending-changes computation | |
| Compute exact pendingChanges counts | Re-run Sync's diff per status call | |

**Choice:** Verify + surface (D-06).
**Notes:** `computeStale` (`.sync-pending` sidecar OR newest-mtime > `last_sync_unix_ms`) and `!IsCurrentSchemaVersion(meta)` are already live and already printed by the terse line. Remaining work is the sectioned rendering + TS's advisory lines. Exact `pendingChanges` counts are an explicit REQUIREMENTS.md Out-of-Scope row — rejected. Flagged the CR-02 trap from memory `9zt8afrs8k`: "implemented + unit-tested + marked complete ≠ delivered" — trace `stale` end-to-end through CLI *and* MCP.

---

## Notice Delivery & Verbatim Strings (WORK-02)

| Option | Description | Selected |
|--------|-------------|----------|
| Mirror the `staleBanner` prepend precedent | `worktreeNotice()` alongside `staleBanner()`; verbose for status, compact for the other 7 read tools | ✓ |
| Wrap at each MCP handler | Duplicate the wrapper per tool | |

**Choice:** Mirror `staleBanner` (D-11, D-12).
**Notes:** TS's `withWorktreeNotice` prefixes `notice\n\n`, no-ops on `isError`, and excludes `codegraph_status` (which embeds its own verbose blockquote form). Our `staleBanner` already establishes exactly this pattern and already uses the correct glyph. **Hexdump-verified:** the glyph is U+26A0 `⚠` (`e2 9a a0`) with **no** U+FE0F variation selector — *not* the `⚠️` from Phase 1's covering-tests warning. Also verified TS's CLI `warn()` writes to `console.log` = **stdout**, not stderr — matters for CLI byte-parity.

---

## Detection Wiring & Caching (WORK-01/02)

| Option | Description | Selected |
|--------|-------------|----------|
| Plumb `startPath` onto Engine; cache once via `sync.Once` | Detect in the shared engine so CLI+MCP both gain awareness in one commit | ✓ |
| Detect at each call site | Duplicate detection in CLI and MCP separately | |
| Detect per tool call, uncached | Re-probe git on every request | |

**Choice:** Engine plumbing + cached-once (D-13, D-14).
**Notes:** `OpenAt(start)` currently **discards** `start` after `ResolveCodegraphDir` — `Engine{reader, repoRoot}` holds only the resolved *index* root, and the `StatusResult` doc comment even records "Engine carries no path context". Detection needs both sides, and TS deliberately captures `startPath` *before* the walk-up. Since `OpenAt` is "the single read seam CLI commands and MCP tool handlers both" use, plumbing it there is what delivers the ROADMAP's "one commit" promise. Caching mirrors TS (#926: 2 git subprocesses/call would regress long-lived MCP latency); negative results must cache too, so nil ≠ unchecked.

---

## Test Fixtures (TEST-02)

| Option | Description | Selected |
|--------|-------------|----------|
| Real `git` subprocess fixtures in `t.TempDir()` | Build all six layouts with actual git; skip when git absent | ✓ |
| Hand-built fake `.git` directories | Fabricate the metadata | |

**Choice:** Real git (D-15).
**Notes:** The submodule-vs-linked-worktree distinction lives entirely in git's real `--git-common-dir` semantics; a fabricated `.git` would not reproduce gate 4 and would test nothing. Expected verdicts pinned per layout so the inverted gate-4 polarity is locked by assertion. `.claude/worktrees/<name>/` is a first-class fixture — it is the motivating true positive for Sean's GSD worktree-per-phase flow (memory `76t84ynav5`).

---

## Claude's Discretion

- File layout within `internal/query` for the status sections and notice helper (extend `status.go`/`render_markdown.go` vs a new `render_status.go`).
- The internal shape of the `FilesByLanguage` aggregation.
- Exact `internal/gitmeta` function signatures.
- The fixture-builder helper structure.

All bounded by the plain-text-only, shared-engine, and never-blocking constraints.

## Deferred Ideas

- Colorized/TTY-gated `status` + the ASCII-vs-Unicode glyph fallback — Phase 6 (TUI-02).
- Exact `pendingChanges` counts — out of scope for v1.0 (REQUIREMENTS.md).
- `Journal:`/journalMode line — permanently dropped (no Pebble analog).
- Auto-init / index-sharing for worktrees — deferred past v1.0 (PROJECT.md Key Decisions).
- Reusing `internal/gitmeta` for git sync hooks — Phase 5 (HOOK-*).
- TS's `getPendingFiles()` freshness + `isWatcherDegraded()` sections — depend on a live watcher, Phase 3 (WATCH-01).
- **Reviewed todo, not folded:** "Document release procedures (maintainer runbook)" (score 0.40) — release-engineering docs, belongs to Phase 8 (REL). The `--auto` ≥0.4 auto-fold default was overridden by the scope guardrail, consistent with Phase 1's identical call.
</content>
