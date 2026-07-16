---
phase: 02-status-content-git-worktree-awareness
verified: 2026-07-16T12:53:44Z
status: passed
score: 6/6 must-haves verified
behavior_unverified: 0
overrides_applied: 0
re_verification: No — initial verification
---

# Phase 2: status Content & Git/Worktree Awareness Verification Report

**Phase Goal:** `status` reports the full TS-parity content (DB size, nodes-by-kind, files-by-language, live staleness), and every read tool — CLI and MCP — detects a borrowed worktree index and warns, closing the silent "worktree queries the main branch's graph" correctness bug. Every MCP read tool settles on one markdown output shape, so the worktree notice rides a single uniform mechanism.

**Verified:** 2026-07-16T12:53:44Z
**Status:** passed
**Re-verification:** No — initial verification

This phase went through two full code-review + fix cycles (02-REVIEW.md/02-REVIEW-FIX.md and 02-REVIEW-2.md/02-REVIEW-FIX-2.md), which surfaced and fixed 2 Critical + 5 Warning findings, including a serious silent-correctness bug (BL-01: one cancelled MCP call could permanently disable the worktree notice for the whole server's life). This verification does not take those review reports as evidence — every claim below was independently re-driven against the real CLI binary and a real stdio MCP session, plus one independent mutation test of the highest-risk regression guard.

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | `codegraph status` shows the full TS-parity sectioned content (DB size, nodes-by-kind, files-by-language, comma-grouped numbers, `Backend: pebble`, no `Journal:` line) | ✓ VERIFIED | Live run on this repo's own index — see "Check 1" below |
| 2 | `status` surfaces a live staleness/reindex advisory derived from `stale`, not a `PendingChanges` count | ✓ VERIFIED | `render_status.go:139` calls `writeStatusAdvisories(&b, r, ...)`, gated on `r.Stale`; live output showed the advisory sentence with `pendingChanges` JSON staying `{0,0,0}` — see "Check 2" |
| 3 | `status --json` carries `dbSizeBytes` (int > 0), omits `filesByLanguage`, and `worktreeMismatch` is `null` on a clean checkout | ✓ VERIFIED | Live `--json` capture, parsed with Python `json` — see "Check 3" |
| 4 | A query run from a borrowed git worktree is detected and warned on BOTH the CLI (verbose form on `status`, compact form on other commands) AND the MCP surface (verbose blockquote on `codegraph_status`, compact prefix on the other 6 tools), with a control run from the main checkout emitting no notice | ✓ VERIFIED | Real `git worktree add --detach`, real CLI runs, and a real stdio JSON-RPC MCP session (`serve --mcp`) driven from Python — see "Check 4" (CLI) and "Check 5" (MCP) |
| 5 | `--json` never carries the worktree notice and stays valid JSON, even from inside a borrowed worktree | ✓ VERIFIED | `codegraph callers ... --json` from inside the probe worktree parsed cleanly with `json.load` — see "Check 4" |
| 6 | The 5 SURF-06 MCP tools (`callers`/`callees`/`impact`/`search`/`files`) plus `status` emit markdown (not JSON) on MCP, while CLI `--json` still emits JSON via the untouched `Marshal*JSON` helpers | ✓ VERIFIED | Live MCP `codegraph_callers`/`codegraph_status` payloads showed markdown tables/bullets; `internal/mcp/tools.go` call-site grep shows zero `json.Marshal`/`Marshal*JSON` calls (all replaced by `Render*Markdown`); CLI `internal/cli/*.go` still calls the same `Marshal*JSON` functions — see "Check 6" |

**Score:** 6/6 truths verified (0 present-but-behavior-unverified)

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/gitmeta/detect.go`, `worktree.go`, `cache.go` | 4-gate worktree-mismatch detection cascade + caching detector | ✓ VERIFIED | Exists, substantive, wired into `internal/query/engine.go`; live-tested (see Check 4/5) |
| `internal/gitmeta/*_test.go` (TEST-02 fixtures) | 7 real-`git` fixture layouts with correct verdicts | ✓ VERIFIED | `TestFixtureVerdicts` — linked-worktree, `.claude/worktrees/`, submodule, nested-clone, monorepo-subdir (+ plain-ancestor variant), symlinked, non-git — all pass against real `git`, not faked `.git` dirs (see Check 7) |
| `internal/query/render_status.go` (`RenderStatusText`, `RenderStatusMarkdown`) | D-09 CLI sectioned layout + D-17 MCP bolded-bullet layout | ✓ VERIFIED | Both functions exist; live CLI and MCP output matches both target shapes byte-for-byte against the CONTEXT.md D-09/D-17 templates |
| `internal/query/render_results.go` (`RenderCallersMarkdown`, `RenderCalleesMarkdown`, `RenderImpactMarkdown`, `RenderSearchMarkdown`, `RenderFilesMarkdown`) | SURF-06's 5 markdown sibling renderers | ✓ VERIFIED | All 5 functions exist and are the only call at their respective `internal/mcp/tools.go` call sites |
| `internal/query/worktree_notice.go` (`WorktreeNotice`) | Shared compact-notice text-prefix mechanism (WORK-02/D-12) | ✓ VERIFIED | Called from all 9 CLI read commands and all 7 non-status MCP tools |
| `internal/query/status.go` (`StatusResult.FilesByLanguage`, `DbSizeBytes`, live `WorktreeMismatch`) | STAT-01/02, WORK-01 data model | ✓ VERIFIED | Live `--json` output carries all three; `MarshalStatusJSON` body untouched (git diff confirms only ctx param + doc comment changed per 02-REVIEW-2.md, independently spot-checked by grep) |
| `internal/mcp/tools.go` (6 call sites → `Render*Markdown`) | SURF-06 wiring | ✓ VERIFIED | grep shows all 6 (`search`/`callers`/`callees`/`impact`/`files`/`status`) call the matching `Render*Markdown` function, zero `json.Marshal` |
| `internal/cli/status.go`, `internal/cli/serve.go` (`serveServerPaths`) | Sectioned CLI layout + CR-01-safe start-path plumbing | ✓ VERIFIED | Live CLI output; `serve.go:38-46` confirmed to keep `repoPath` (resolved index root) and `start` (caller cwd) distinct, matching `BuildServer(hasIndex, allowlist, repoPath, start)` at `serve.go:142` |

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| `internal/cli/serve.go` (`RunE`) | `internal/mcp.BuildServer` | `mcp.BuildServer(hasIndex, allowlist, repoPath, start)` | ✓ WIRED | Read source directly; `start` (not `repoPath` twice) is the 4th arg — confirms CR-01's fix held |
| `internal/mcp/tools.go` (`exploreHandler`/`companionHandler`) | `internal/query.WorktreeNotice` | text-prefix onto success-only `mcp.NewToolResultText` | ✓ WIRED | 7 call sites found (`explore`, `node`, `search`, `callers`, `callees`, `impact`, `files`); `status` deliberately excluded (own blockquote) |
| `internal/cli/{query,search,callers,callees,impact,affected,files,explore,node}.go` | `internal/query.WorktreeNotice` | `fmt.Fprint(out, query.WorktreeNotice(eng.WorktreeMismatch(cmd.Context())))` | ✓ WIRED | All 9 non-status read commands call it; live-verified on `callers` from a real borrowed worktree |
| `internal/gitmeta.CachingDetector.Detect` | cache write | `if ctx.Err() != nil { return v }` before `d.cache[key] = v` | ✓ WIRED | Live-read at `cache.go:104`; independently mutation-tested (removed the guard, confirmed `TestCachingDetectorCancelledContextNotCached` fails, restored, confirmed clean) |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|---------------|--------|---------------------|--------|
| `RenderStatusText`/`RenderStatusMarkdown` | `StatusResult.{DbSizeBytes,NodesByKind,FilesByLanguage,Stale}` | `Engine.Status()` — real `filepath.WalkDir` byte sum, real `IterateNodes`/file scan, real `.sync-pending`/mtime check | Yes | ✓ FLOWING — live run on this repo's real 326-file, 3,239-node index produced correct non-static counts that changed between two `status` calls a minute apart (327/3,247/7,883 after a background sync) |
| `RenderCallersMarkdown` | `CallersResult` | `Engine.Callers` reading the real Pebble graph | Yes | ✓ FLOWING — live MCP + CLI calls for `RenderStatusText` returned 9 real callers with real file:line locations |
| `worktreeMismatch` (JSON) / verbose+compact notices | `Engine.WorktreeMismatch(ctx)` | `gitmeta.DetectIndexMismatch` — real `git rev-parse` subprocess calls | Yes | ✓ FLOWING — real `git worktree add`, real detection, real notice text; control run from main checkout correctly produced no notice |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| `status` shows TS-parity sectioned content | `codegraph status` (real binary, real index) | See Check 1 output below | ✓ PASS |
| `status --json` STAT/WORK content | `codegraph status --json` \| `python3 -m json.tool` | See Check 3 output below | ✓ PASS |
| Borrowed-worktree CLI detection (verbose) | `git worktree add --detach .claude/worktrees/verify-probe HEAD` then `codegraph status` from inside it | Verbose warning printed, matches D-11 verbatim | ✓ PASS |
| Borrowed-worktree CLI detection (compact) | `codegraph callers RenderStatusText` from inside the worktree | Compact `⚠` notice printed, `e2 9a a0 20` byte-confirmed (no U+FE0F) | ✓ PASS |
| CLI control (no false positive from main checkout) | `codegraph callers RenderStatusText` from main checkout | No notice | ✓ PASS |
| Borrowed-worktree MCP detection | Real stdio JSON-RPC session (`serve --mcp`, cwd = worktree) calling `codegraph_explore`/`codegraph_callers`/`codegraph_status` | All 3 emit the notice (compact for explore/callers, blockquote for status) | ✓ PASS |
| MCP control (no false positive from main checkout) | Same session, cwd = main checkout | None of the 3 tools emit a notice | ✓ PASS |
| `--json` never carries the notice, stays parseable | `codegraph callers RenderStatusText --json` from inside the worktree | `json.load()` succeeds, no notice text present | ✓ PASS |
| TEST-02 fixture matrix | `go test ./internal/gitmeta/... -run TestFixtureVerdicts -v` | 8/8 subtests pass (7 layouts + plain-ancestor variant) | ✓ PASS |
| BL-01 regression guard (independent mutation test) | Remove `ctx.Err()` guard in `cache.go`, re-run `TestCachingDetectorCancelledContextNotCached` | Test FAILS as expected (`BL-01 REGRESSION: a verdict computed under a cancelled context was written to the cache`); guard restored, `git diff` empty | ✓ PASS |
| Full non-daemon suite | `go test $(go list ./... | grep -v /internal/daemon)` | All packages `ok` | ✓ PASS |
| Isolated daemon suite | `go test ./internal/daemon/ -count=1` | `ok` | ✓ PASS |
| Golden parity suite (NOT covered by `go test ./...`) | `go test ./testdata/golden/...` | `ok` | ✓ PASS |
| Static analysis | `go vet ./...` | clean | ✓ PASS |

### Probe Execution

No `scripts/*/tests/probe-*.sh` convention exists in this repository and none is referenced by this phase's PLAN/SUMMARY files — Step 7c: SKIPPED (no probe scripts declared or found).

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|--------------|------------|--------------|--------|----------|
| STAT-01 | 02-02, 02-05, 02-07 | `status` reports Pebble on-disk DB size | ✓ SATISFIED | Live `status --json`: `dbSizeBytes: 1253427`; live `status` text: `DB Size: 1.20 MB` |
| STAT-02 | 02-02, 02-05, 02-07 | `status` reports nodes-by-kind + files-by-language | ✓ SATISFIED | Live output shows both sections, sorted DESC, `count > 0` filtered, `padEnd(15)`-equivalent alignment |
| STAT-03 | 02-05, 02-07 | Live pending-changes / reindex-recommended signal, not inert placeholder | ✓ SATISFIED | Advisory text driven by `r.Stale` (code-read); `pendingChanges` count stays inert `{0,0,0}` per explicit out-of-scope decision (D-06) |
| WORK-01 | 02-01, 02-04 | Worktree mismatch detected via 4-gate cascade | ✓ SATISFIED | Real `git worktree add` + live detection; `TestFixtureVerdicts` green against real git for all layouts |
| WORK-02 | 02-04, 02-06, 02-07 | Verbose warning on `status`, compact notice on other read tools (CLI+MCP), shared `withWorktreeNotice`-equivalent | ✓ SATISFIED | Live-verified on both CLI (9 commands) and MCP (7 tools + status blockquote) |
| WORK-03 | 02-01 | Best-effort, never blocking, no false positives on submodule/nested-clone/monorepo-subdir/non-git/symlinked | ✓ SATISFIED | `TestFixtureVerdicts` covers all 5 negative cases + linked-worktree/`.claude/worktrees` positive cases, all pass against real git |
| TEST-02 | 02-01 | Fixtures for linked-worktree, submodule, nested-clone, monorepo-subdir, `.claude/worktrees/`, symlinked | ✓ SATISFIED | `internal/gitmeta/detect_test.go` `TestFixtureVerdicts` — 8 subtests, all pass, driven via real `os/exec` git in `t.TempDir()` |
| SURF-06 | 02-03, 02-06 | 5 JSON-shaped MCP tools + status switch to markdown; CLI `--json` and `Marshal*JSON` bodies untouched | ✓ SATISFIED | Live MCP payloads are markdown; grep confirms zero `Marshal*JSON` calls remain in `tools.go`; CLI still calls the same `Marshal*JSON` functions |

No orphaned requirements: REQUIREMENTS.md maps exactly STAT-01..03, WORK-01..03, TEST-02, SURF-06 to Phase 2, and all 8 are claimed by at least one of the 7 plans' `requirements:` frontmatter.

### Anti-Patterns Found

None. Scanned `internal/gitmeta/*.go`, `internal/query/status.go`, `internal/query/render_status.go`, `internal/query/render_results.go`, `internal/query/worktree_notice.go`, `internal/query/engine.go`, `internal/mcp/tools.go`, `internal/mcp/server.go`, `internal/cli/status.go`, `internal/cli/serve.go` for `TBD|FIXME|XXX|TODO|HACK|PLACEHOLDER`. One match ("TODO, it is v1.0's deliberate bar" in `render_status.go:134`) is an explanatory comment stating the inert `PendingChanges` placeholder is *not* a TODO — not a debt marker. No color/ANSI/lipgloss imports found (only explanatory comments naming `lipgloss` for Phase 6). No `Journal:` line in actual rendered output (only in comments/test names explaining its absence).

### Anti-Goal Checks (intentionally NOT gaps)

| Anti-goal | Verified absent/present as expected |
|-----------|--------------------------------------|
| Color/lipgloss/ANSI anywhere | Confirmed absent — grep found zero ANSI escape sequences, zero `lipgloss` imports |
| `PendingChanges` all-zero | Confirmed — live `--json` shows `{"added":0,"modified":0,"removed":0}` |
| `Journal:` line / watcher sections | Confirmed absent from rendered output; explicit test (`TestRenderStatusTextNoJournalLine`) guards it |
| CLI ≠ MCP output for the 5 SURF-06 tools | Confirmed intentional — CLI emits JSON via `Marshal*JSON`, MCP emits markdown via `Render*Markdown`, verified live on both surfaces |
| en-US comma grouping | Confirmed — live output shows `3,239`, `7,867` |

### Human Verification Required

None. Every must-have in this phase was directly, live-verified against the real CLI binary and a real stdio MCP JSON-RPC session — no truth in this phase required subjective/visual/real-time judgment beyond what was already exercised.

### Gaps Summary

No gaps. All 6 derived observable truths (mapped 1:1 to the ROADMAP's 6 success criteria for this phase) verified against live command execution, not SUMMARY.md claims. Both prior code-review cycles' findings (2 Critical, 9 Warning total across both iterations) were independently spot-checked here — most notably BL-01 (the cancelled-context cache-poisoning bug), whose regression guard was independently mutation-tested in this verification pass (guard removed → test fails as predicted → guard restored → tree clean) rather than trusted from the fix report alone.

One process note, not a phase gap: `go test ./...` alone does **not** run the golden parity suite (the Go tool ignores `testdata/` directories) — this was itself a finding fixed within this phase (GOLDEN-01, now wired into `.github/workflows/ci.yml`). This verification ran `go test ./testdata/golden/...` explicitly, per the instructions, and it is green.

---

## Evidence Appendix

### Check 1: `codegraph status` — full sectioned content

```
$ /tmp/codegraph-verify status
CodeGraph Status

Project: /Volumes/Code/github.com/seanb4t/codegraph-go

Index Statistics:
  Files:     326
  Nodes:     3,239
  Edges:     7,867
  DB Size:   1.20 MB
  Backend:   pebble
Nodes by Kind:
  function        1,764
  method          597
  file            326
  constant        217
  struct          193
  variable        74
  package         32
  type_alias      17
  interface       15
  route           4
Files by Language:
  go              308
  python          8
  typescript      4
  java            3
  csharp          2
  javascript      1

Pending Changes: a sync is recommended — this index may be stale. Run "codegraph sync" to update.
```

No `Journal:` line present. Comma-grouped numbers confirmed (`3,239`, `7,867`). `Backend: pebble` confirmed.

### Check 3: `codegraph status --json`

```
$ /tmp/codegraph-verify status --json | python3 -c "
import json,sys
d=json.load(sys.stdin)
print('dbSizeBytes' in d, d.get('dbSizeBytes'))
print('filesByLanguage' in d)
print('worktreeMismatch' in d, d.get('worktreeMismatch'))
"
True 1253427
False
True None
```

### Check 4: CLI worktree detection (real `git worktree add`)

```
$ git worktree add --detach .claude/worktrees/verify-probe HEAD
Preparing worktree (detached HEAD 1a01c38)

$ cd .claude/worktrees/verify-probe && codegraph status
CodeGraph Status

Project: .../codegraph-go/.claude/worktrees/verify-probe
This CodeGraph index belongs to a different git working tree.
  Running in: .../codegraph-go/.claude/worktrees/verify-probe
  Index from: .../codegraph-go
Results reflect that tree's code (often a different branch), not this worktree — symbols changed only here are missing. Run "codegraph init -i" in this worktree for a worktree-local index.
...

$ codegraph callers RenderStatusText
⚠ CodeGraph results below come from a different git worktree (.../codegraph-go), not where you're working (.../verify-probe) — they may reflect another branch, and symbols changed only here are missing. Run "codegraph init -i" here for a worktree-local index.

RenderStatusText has 9 caller(s): ...
```

`xxd` on the notice's first bytes: `e2 9a a0 20` — bare U+26A0, no U+FE0F variation selector, confirming D-11's glyph requirement.

Control (main checkout, same command): `codegraph callers RenderStatusText` → no notice, direct results only.

`--json` from inside the worktree: `codegraph callers RenderStatusText --json | python3 -c "import json,sys; json.load(sys.stdin); print('VALID JSON')"` → `VALID JSON`, no notice text present.

### Check 5: MCP worktree detection (real stdio JSON-RPC session)

Driven via a hand-written Python client speaking the mcp-go newline-delimited JSON-RPC framing (`initialize` → `notifications/initialized` → `tools/call`), with `CODEGRAPH_MCP_TOOLS=status,callers` to register the companion tools beyond the default `codegraph_explore`.

From inside `.claude/worktrees/verify-probe`:

```
codegraph_explore → "⚠ CodeGraph results below come from a different git worktree ..."
codegraph_callers → "⚠ CodeGraph results below come from a different git worktree ...\n\n**Callers of `RenderStatusText`** — 9 callers\n\n| Name | Kind | Location |\n|---|---|---|\n..."
codegraph_status  → "**CodeGraph Status**\n\n> ⚠ This CodeGraph index belongs to a different git working tree.\n>   Running in: .../verify-probe\n>   Index from: .../codegraph-go\n> Results reflect ...\n\n**Files indexed:** 327\n**Total nodes:** 3,247\n..."
```

Control (same session shape, cwd = main checkout): all 3 tools return results with no notice prefix (`codegraph_status` has no blockquote at all).

Probe worktree removed afterward (`git worktree remove --force .claude/worktrees/verify-probe`); `git status` confirms working tree clean, `git worktree list` confirms only the main checkout remains.

### Check 6: SURF-06 markdown/JSON split

```
$ grep -n "RenderCallersMarkdown\|RenderCalleesMarkdown\|RenderImpactMarkdown\|RenderSearchMarkdown\|RenderFilesMarkdown\|RenderStatusMarkdown\|Marshal.*JSON\|json.Marshal" internal/mcp/tools.go
# (comments only, plus 6 Render*Markdown call sites — zero live json.Marshal/Marshal*JSON calls)

$ grep -n "MarshalCallersJSON\|MarshalCalleesJSON\|MarshalImpactJSON\|MarshalFilesJSON\|MarshalStatusJSON\|MarshalQueryJSON" internal/cli/*.go
# all 6 CLI --json paths still call the shared Marshal*JSON functions
```

### Check 7: TEST-02 fixture matrix

```
$ go test ./internal/gitmeta/... -v -run TestFixtureVerdicts
=== RUN   TestFixtureVerdicts
--- PASS: TestFixtureVerdicts (0.52s)
    --- PASS: TestFixtureVerdicts/linked-worktree (0.08s)
    --- PASS: TestFixtureVerdicts/claude-worktrees (0.06s)
    --- PASS: TestFixtureVerdicts/submodule (0.20s)
    --- PASS: TestFixtureVerdicts/nested-clone (0.07s)
    --- PASS: TestFixtureVerdicts/monorepo-subdir (0.03s)
    --- PASS: TestFixtureVerdicts/symlinked (0.03s)
    --- PASS: TestFixtureVerdicts/non-git (0.00s)
    --- PASS: TestFixtureVerdicts/monorepo-subdir-plain-ancestor (0.04s)
PASS
```

### Check 8: Independent BL-01 mutation test

```
$ cp internal/gitmeta/cache.go /tmp/cache.go.bak
# removed: if ctx.Err() != nil { return v }
$ go test ./internal/gitmeta/... -run TestCachingDetectorCancelledContextNotCached -v
    cache_test.go:114: BL-01 REGRESSION: a verdict computed under a cancelled context was written to the cache
--- FAIL: TestCachingDetectorCancelledContextNotCached
$ cp /tmp/cache.go.bak internal/gitmeta/cache.go
$ git diff --stat internal/gitmeta/cache.go
# (empty — restore confirmed clean)
```

### Check 9: Full suites (build/vet/test)

```
$ go build ./...           # clean
$ go vet ./...              # clean
$ go test $(go list ./... | grep -v /internal/daemon)   # all ok
$ go test ./internal/daemon/ -count=1                    # ok
$ go test ./testdata/golden/...                          # ok (explicit — go test ./... does NOT cover this)
$ git status                                              # clean, no leftover probe files
```

---

_Verified: 2026-07-16T12:53:44Z_
_Verifier: Claude (gsd-verifier)_
