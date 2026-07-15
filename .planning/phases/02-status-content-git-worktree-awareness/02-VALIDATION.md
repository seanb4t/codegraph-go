---
phase: 2
slug: status-content-git-worktree-awareness
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-07-15
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

Task IDs are assigned by the planner; this maps **requirements → the test that proves them**. Every requirement below must trace to at least one plan task.

| Req | Behavior | Test Type | Automated Command | File Exists |
|-----|----------|-----------|-------------------|-------------|
| STAT-01 | `status` reports Pebble on-disk DB size (`dbSizeBytes` raw in `--json`, `X.XX MB` rendered) | unit | `go test ./internal/query/... -run TestStatus` | ❌ W0 — extend `internal/query/files_status_test.go` |
| STAT-02 | `status` reports nodes-by-kind + files-by-language (count>0, sorted count DESC) | unit | `go test ./internal/query/... -run TestStatus` | ❌ W0 — same file |
| STAT-03 | `status` reports live stale / reindexRecommended, **reachable on both CLI and MCP** | unit + integration | `go test ./internal/query/... -run TestStatusStaleness` | ✅ `internal/query/status_staleness_test.go` — extend for sectioned-output reachability |
| WORK-01 | Mismatch detected via the 4-gate cascade (incl. gate-4 suppression polarity) | unit | `go test ./internal/gitmeta/...` | ❌ W0 — package does not exist |
| WORK-02 | Verbose warning (`status`) + compact notice (7 other read tools, CLI+MCP) | integration | `go test ./internal/mcp/... -run TestNotice` + `go test ./internal/cli/... -run TestNotice` | ❌ W0 |
| WORK-03 | Best-effort, never blocking, no false positives | unit | `go test ./internal/gitmeta/... -run TestNoFalsePositive` | ❌ W0 |
| TEST-02 | Six fixture layouts, real `git` in `t.TempDir()` | integration | `go test ./internal/gitmeta/... -run TestFixture` | ❌ W0 — no repo-*building* helper exists anywhere yet |
| SURF-06 | The 5 `Render*` funcs emit markdown (renderer in isolation) | unit | `go test ./internal/query/... -run TestRender` | ❌ W0 |
| SURF-06 | Each of the 5 MCP tools' **success payload** is markdown end-to-end | integration | `go test ./internal/mcp/... -run TestMarkdownOutput` | ❌ W0 — **zero existing coverage (blind spot)** |
| SURF-06 | CLI `--json` still emits valid JSON for all 5 commands (**regression guard**) | integration | `go test ./internal/cli/... -run "TestSearchCmd\|TestCallersCalleesCmd\|TestImpactCmd\|TestFilesCmd"` | ✅ `internal/cli/query_cli_test.go` — **must keep passing untouched** |
| D-17 | MCP `status` renders markdown (bolded-key bullets), CLI renders padded columns | unit + integration | `go test ./internal/query/... -run TestRenderStatus` + `go test ./internal/mcp/... -run TestStatusMarkdown` | ❌ W0 — MCP status is JSON today |

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

- [ ] `internal/gitmeta/` — new package + its test file (WORK-01/03); nothing exists yet
- [ ] `internal/gitmeta/fixtures_test.go` (or similar) — the real-`git` repo-**building** helper for TEST-02's six layouts. **No precedent exists in this codebase**; the closest (`testdata/golden/golden_parity_test.go`'s `resolveColbymchenryCorpus`) only *clones*, never *builds*. Must `t.Skip` when `git` is absent and set deterministic `GIT_*` env for hermetic commits.
- [ ] `internal/mcp/` markdown-output test — the red test for SURF-06's blind spot (above)
- [ ] `internal/query/` — `Render*` unit tests for the 5 new renderers + MCP status renderer (D-17)
- [ ] Extend `internal/query/files_status_test.go` — STAT-01/02; note its existing assertions that `WorktreeMismatch` is nil must flip to live (`PendingChanges` stays zero per D-06)
- [ ] `testdata/golden/golden_parity_test.go` ~line 651 — narrow `dbSizeBytes` exemption (D-08). **Do NOT touch the shared `volatileKeys` map in `golden_test.go`** — it correctly still gates the frozen TS oracle fixtures.

*No framework install needed — Go stdlib `testing` is already the convention.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Real borrowed-index warning in a live agent session | WORK-02 | End-to-end MCP handshake through a real agent client can't be asserted in `go test` | From a `.claude/worktrees/<name>/` worktree of this repo, run a `codegraph_explore` MCP call and confirm the compact `⚠` notice prefixes the result |
| `dbSizeBytes` plausibility across a real re-index | STAT-01 | Byte value is intentionally non-deterministic (Pebble LSM compaction) — only shape is asserted | Run `codegraph status --json` before/after `index --force`; confirm both are integers > 0 and the MB rendering matches `^\d+\.\d{2} MB$` |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all ❌ MISSING references above
- [ ] No watch-mode flags
- [ ] Feedback latency < 90s
- [ ] MCP success-payload assertions exist for all 5 SURF-06 tools (closes the blind spot)
- [ ] CLI `--json` regression guard still green and unmodified
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
</content>
