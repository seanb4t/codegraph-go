---
phase: 2
slug: status-content-git-worktree-awareness
status: complete
nyquist_compliant: true
wave_0_complete: true
created: 2026-07-15
audited: 2026-07-16
---

# Phase 2 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.
> Derived from `02-RESEARCH.md` § Validation Architecture (commit `0ff9a71`).

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go stdlib `testing` (`go test`) — no third-party test framework in this repo |
| **Config file** | none — standard `go test`, no test-tool config in `go.mod` |
| **Quick run command** | `go test ./internal/gitmeta/... ./internal/query/... ./internal/mcp/...` |
| **Full suite command** | `go test ./...` |
| **Estimated runtime** | ~60–90 seconds (full suite; quick run ~15s) |

**Environment:** `git` 2.55.0 ✓, Go 1.26.5 ✓, `cockroachdb/pebble/v2` v2.1.6 ✓. No missing dependencies. `git`'s absence is already handled by WORK-03's best-effort design (degrade to "no mismatch") and by TEST-02 fixtures using `t.Skip`, never `t.Fatal`.

---

## Sampling Rate

- **After every task commit:** Run the quick run command
- **After every plan wave:** Run `go test ./...`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** ~90 seconds

TDD mode is active (`tdd_mode: true`), so eligible tasks write the failing test first.

---

## Per-Task Verification Map

Reconciled against delivered tests by the 2026-07-16 Nyquist audit (all suites re-run green that day: `internal/gitmeta` 6.0s, `internal/mcp` 2.6s, `internal/query`/`internal/cli`/`testdata/golden` earlier the same session).

| Req | Behavior | Test Type | Automated Command / Evidence | File Exists | Status |
|-----|----------|-----------|------------------------------|-------------|--------|
| STAT-01 | `status` reports Pebble on-disk DB size (`dbSizeBytes` raw in `--json`, `X.XX MB` rendered) | unit + CLI + golden | `go test ./internal/query/ -run 'TestStatus\|TestDbSizeBytes\|TestFormatMB' -count=1` + `go test ./internal/cli/ -run TestStatusCmdSections -count=1` + golden `TestGoldenParity/status` (plausibility + MB-shape) | ✅ `files_status_test.go:303,492`, `render_status_test.go`, `status_cli_test.go` | ✅ green |
| STAT-02 | `status` reports nodes-by-kind + files-by-language (count>0, sorted count DESC) | unit + CLI | `go test ./internal/query/ -run 'TestStatus\|TestSortedCounts\|TestRenderStatusTextBreakdownFilterSortPad' -count=1` (+ filesByLanguage json:"-" absence subtest) | ✅ same files | ✅ green |
| STAT-03 | `status` reports live stale / reindexRecommended, reachable on both CLI and MCP | unit + CLI + MCP | `go test ./internal/query/ -run 'TestRenderStatusTextStaleAdvisory\|TestRenderStatusTextReindexAdvisory' -count=1` + `status_cli_test.go#TestStatusCmdSections/Test_3` + `internal/mcp/markdown_test.go#TestStatusMarkdownOutput` | ✅ `render_status_test.go`, `status_cli_test.go`, `markdown_test.go` | ✅ green |
| WORK-01 | Mismatch detected via the 4-gate cascade (incl. gate-4 suppression polarity) | unit + engine wiring | `go test ./internal/gitmeta/ -run TestFixtureVerdicts -count=1` (8 real-git layouts) + `go test ./internal/query/ -run 'TestEngineWorktreeMismatch\|TestOpenAtAbsolutizesStartPath\|TestStatusWorktreeMismatch' -count=1` | ✅ `detect_test.go:20`, `engine_worktree_test.go` | ✅ green |
| WORK-02 | Verbose warning (`status`) + compact notice (7 other read tools, CLI+MCP) | integration | `go test ./internal/mcp/ -run 'TestWorktreeNotice' -count=1` (mismatch/clean/error-path/consistency) + `go test ./internal/cli/ -run 'TestNotice' -count=1` (mismatch/clean/JSON-suppressed) + verbatim strings in `gitmeta/notice_test.go` — plus, since Phase 3, the SUBPROCESS anchor `test/integration/worktree_notice_test.go#TestWorktreeNoticeReachesServeMCPExplore` (real spawned binary, mutation-proof vs CR-01) | ✅ `markdown_test.go:318`, `notice_test.go:152` | ✅ green |
| WORK-03 | Best-effort, never blocking, no false positives | unit | `go test ./internal/gitmeta/ -count=1` (TestFixtureVerdicts negative layouts: submodule/nested-clone/monorepo-subdir/non-git/symlinked + nil-receiver + memoization) + `engine_worktree_test.go#TestEngineWorktreeMismatchDegradesSafely` | ✅ `detect_test.go`, `cache_test.go` | ✅ green |
| TEST-02 | Six fixture layouts, real `git` in `t.TempDir()` | integration | `go test ./internal/gitmeta/ -run TestFixtureVerdicts -count=1` — 8 subtests (7 layouts + plain-ancestor variant), real `os/exec` git, `t.Skip` on git absence | ✅ `fixtures_test.go` (seven builders) | ✅ green |
| SURF-06 | The 5 `Render*` funcs emit markdown (renderer in isolation) | unit | `go test ./internal/query/ -run 'TestRenderCallersMarkdown\|TestRenderCalleesMarkdown\|TestRenderImpactMarkdown\|TestRenderSearchMarkdown\|TestRenderFilesMarkdown\|TestRenderLocationTable' -count=1` | ✅ `render_results_test.go` | ✅ green |
| SURF-06 | Each of the 5 MCP tools' **success payload** is markdown end-to-end | integration | `go test ./internal/mcp/ -run TestMarkdownOutput -count=1` — asserts BOTH `json.Unmarshal` FAILS AND markdown marker present, per the blind-spot contract below | ✅ `markdown_test.go:203` — **blind spot CLOSED** | ✅ green |
| SURF-06 | CLI `--json` still emits valid JSON for all 5 commands (**regression guard**) | integration | `go test ./internal/cli/ -run 'TestSearchCmd\|TestCallersCalleesCmd\|TestImpactCmd\|TestFilesCmd' -count=1`; `query_cli_test.go` confirmed untouched (`git diff --stat` empty per 02-06/02-07 SUMMARYs) | ✅ `query_cli_test.go` | ✅ green |
| D-17 | MCP `status` renders markdown (bolded-key bullets), CLI renders padded columns | unit + integration | `go test ./internal/query/ -run TestRenderStatusMarkdownShape -count=1` + `go test ./internal/mcp/ -run TestStatusMarkdownOutput -count=1` | ✅ `render_status_test.go:241`, `markdown_test.go:253` | ✅ green |
| BL-01 (review fix) | A verdict computed under a cancelled ctx is returned but NEVER cached (cache-poisoning guard) | unit regression | `go test ./internal/gitmeta/ -run TestCachingDetectorCancelledContextNotCached -count=1` — independently mutation-tested during phase verification (guard removed → test fails → restored) | ✅ `cache_test.go:99` | ✅ green |

*Status legend: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## ★ The Blind Spot (drives Wave 0 ordering)

> **Not one existing test asserts the MCP success-payload text of ANY of the 5 tools SURF-06 changes.** MCP coverage today = `explore` (markdown) + `status` (**error path only**).

**Consequence:** SURF-06 could be implemented wrong — or skipped entirely — and `go test ./...` stays fully green. This is the same "implemented + marked complete ≠ delivered" failure that produced Phase 1's CR-02. There is no failing test to drive the change and none to catch a regression back to JSON.

**The risk is inverted from intuition.** The feared hazard ("we mutate a shared `Marshal*JSON` helper and break the CLI") is *loud* — 7 tests across 3 files catch it instantly. The genuinely silent failure is the MCP side going unverified in either direction.

**Required red test (Wave 0, TDD):** a table-driven `internal/mcp` test that `CallTool`s each of the 5 tools against the existing `copyFixture` + `indexFixture` harness (`server_test.go`) and asserts the text **is NOT valid JSON** (`json.Unmarshal` *fails*) **AND** contains the expected markdown marker (e.g. a `| Name | Kind |` header row). **Both assertions are required** — the negative makes a silent regression to `json.Marshal` impossible; the positive makes an empty/garbage render impossible. Either alone is defeatable.

**Do NOT extend `TestExploreCLIMatchesMCP`/`TestNodeCLIMatchesMCP` to these 5** — that harness asserts CLI output == MCP output, which SURF-06 makes *intentionally false* for them.

---

## Wave 0 Requirements

- [x] `internal/gitmeta/` — new package + its test files (WORK-01/03) — plan 02-01
- [x] `internal/gitmeta/fixtures_test.go` — the real-`git` repo-building helper, seven layout builders, `t.Skip` on git absence, hermetic `-c` config — plan 02-01
- [x] `internal/mcp/markdown_test.go#TestMarkdownOutput` — the red test for SURF-06's blind spot (both assertions: NOT-valid-JSON + markdown marker) — plan 02-06
- [x] `internal/query/render_results_test.go` + `render_status_test.go` — the 5 renderer unit tests + MCP status renderer (D-17) — plans 02-03/02-05
- [x] Extended `internal/query/files_status_test.go` — STAT-01/02 (dbSizeBytes, filesByLanguage internal-only); `WorktreeMismatch` flipped to live in `engine_worktree_test.go` — plans 02-02/02-04
- [x] `testdata/golden/golden_parity_test.go` — `dbSizeBytes` exemption narrowed (D-08); shared `volatileKeys` map untouched — plan 02-02

*No framework install needed — Go stdlib `testing` is already the convention.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| ~~Real borrowed-index warning in a live agent session~~ **AUTOMATED since Phase 3** | WORK-02 | No longer manual: `test/integration/worktree_notice_test.go#TestWorktreeNoticeReachesServeMCPExplore` spawns the real binary with real process cwd inside `.claude/worktrees/probe` and asserts the `⚠` notice in a live `codegraph_explore` payload over real stdio JSON-RPC (mutation-proof vs CR-01) | `go test ./test/integration/... -run TestWorktreeNoticeReachesServeMCPExplore -count=1` |
| `dbSizeBytes` plausibility across a real re-index | STAT-01 | Byte value is intentionally non-deterministic (Pebble LSM compaction) — only shape is asserted automatically (golden `TestGoldenParity/status`) | Run `codegraph status --json` before/after `index --force`; confirm both are integers > 0 and the MB rendering matches `^\d+\.\d{2} MB$` |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies — every requirement row carries a runnable command, re-executed green 2026-07-16
- [x] Sampling continuity: no 3 consecutive tasks without automated verify — 7 plans, per-task verify blocks throughout (per SUMMARY coverage entries)
- [x] Wave 0 covers all ❌ MISSING references above — all 6 Wave 0 items delivered
- [x] No watch-mode flags — no build tags or skip-by-default gating; git-absence uses `t.Skip` only
- [x] Feedback latency < 90s — gitmeta 6s + mcp 2.6s + query ~9s; full quick lane well under budget
- [x] MCP success-payload assertions exist for all 5 SURF-06 tools — `TestMarkdownOutput` closes the blind spot with BOTH required assertions
- [x] CLI `--json` regression guard still green and unmodified — `query_cli_test.go` untouched, guards pass
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** auto-validated 2026-07-16 (/gsd-validate-phase audit — zero gaps found)

---

## Validation Audit 2026-07-16

| Metric | Count |
|--------|-------|
| Gaps found | 0 |
| Resolved | 0 |
| Escalated | 0 |

Audit method: reconciled the planner's draft map (all rows `❌ W0`) against delivered tests using the 7 SUMMARY coverage blocks + 02-VERIFICATION.md (passed 6/6, with the verifier independently re-driving CLI + a real stdio MCP session and mutation-testing the BL-01 guard), confirmed all key test functions exist, and re-ran `go test ./internal/gitmeta/... ./internal/mcp/... -count=1` live (query/cli/golden suites re-run green earlier the same session). Additions beyond the draft: a BL-01 cache-poisoning regression row (review-cycle fix, `cache_test.go:99`), and WORK-02's row now cross-references Phase 3's subprocess anchor — the former "live agent session" manual-only item is fully automated by `TestWorktreeNoticeReachesServeMCPExplore`, so it was struck from Manual-Only. The draft's blind-spot contract (NOT-valid-JSON AND markdown-marker, both required) was implemented verbatim in `TestMarkdownOutput`.
</content>
