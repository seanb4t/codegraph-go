---
phase: "3"
slug: "verb-fold"
# status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6)
# audit-milestone §5.5 distinguishes NOT-VALIDATED (draft) from PARTIAL (validated + nyquist_compliant: false) (#2117)
status: validated
nyquist_compliant: true
wave_0_complete: true
created: "2026-09-16"
---

# Phase 3 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | `go test` (stdlib) — `internal/cli` package tests through the cobra tree (`execCmd`), `test/integration` against the REAL built binary (`TestMain` builds `binPath`), `test/wireoracle` frozen MCP transcripts; `tools/clidoc` + `task docs:cli:drift` for the generated reference; one-shot positive-controlled `rg -nU -w --hidden` census recorded in `03-MUTATION-LOG.md` |
| **Config file** | `Taskfile.yml` (`docs:cli`, `docs:cli:drift`, `test:wireoracle`); `internal/cli/testdata/cli-reference-allowlist.txt` (reason-carrying allowlist the reference guard reads); `go.mod` pins go 1.26.6 — every local Go gate runs under `GOTOOLCHAIN=go1.26.6` |
| **Quick run command** | `GOTOOLCHAIN=go1.26.6 go test -count=1 ./internal/cli/ -run 'TestSearchFullCmd\|TestSearchFlagShortForms\|TestRenderFullLine\|TestSearchCmd\|TestQueryStub\|TestUnlockStub\|TestDaemonUnlockCmd\|TestIndexForceRefusesWhileStoreIsHeld\|TestEveryRegisteredFlagIsAccountedFor\|TestNotice'` |
| **Full suite command** | `GOTOOLCHAIN=go1.26.6 go build ./... && GOTOOLCHAIN=go1.26.6 go test -count=1 ./internal/cli/... && GOTOOLCHAIN=go1.26.6 go test -count=1 ./test/integration/ -run TestRenamedStub && GOTOOLCHAIN=go1.26.6 task docs:cli:drift && GOTOOLCHAIN=go1.26.6 task test:wireoracle` |
| **Estimated runtime** | ~5 seconds (quick); ~2 min (full — the wire oracle alone is ~52 s uncached) |

---

## Sampling Rate

- **After every task commit:** Run the quick run command above (every 03-02 task ran it RED-first, then GREEN)
- **After every plan wave:** Run the full suite command above (the orchestrator ran `go build ./...` + `go test ./internal/cli/... ./internal/daemon/...` after wave 2 and the regression gate after wave 4)
- **Before `/gsd-verify-work`:** Full suite must be green (it was: 03-04's final gate and the verifier's independent re-run)
- **Max feedback latency:** 5 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 03-01.1 | 03-01 | 1 | VERB-05 | T-03-01-01 / T-03-01-02 | Planted `.github/__census_control__.md` is found on 3 lines, then removed byte-clean and proven absent from disk AND index; the clean census reads 14 lines / 6 files (Family (a)) | one-shot census (positive-controlled) | `rg -nU -w --hidden 'codegraph\s+(query\|unlock)' --glob '!.planning/**' --glob '!CHANGELOG.md' --glob '!web/build/**' --glob '!.git/**' .` with the EXIT-trap control | ✅ `03-MUTATION-LOG.md` § Family (a) | ✅ recorded |
| 03-01.2 | 03-01 | 1 | VERB-01, VERB-06, VERB-07 (baseline) | T-03-01-03…05 | Pre-fold `search` digests (9 invocations), `__complete`/man listings, the 8 MCP names and a green uncached wire oracle are captured before any `internal/cli/` edit | baseline capture | `GOTOOLCHAIN=go1.26.6 go build -o /tmp/03-01-prefold-bin ./cmd/codegraph` + `shasum -a 256`; `GOTOOLCHAIN=go1.26.6 task test:wireoracle` | ✅ `03-MUTATION-LOG.md` § Baseline | ✅ recorded |
| 03-02.1 | 03-02 | 2 | VERB-01, VERB-02 | T-03-02-02 / T-03-02-04 | `search --full` = `Engine.Query` + `MarshalQueryJSON` envelope under `--json` + two-line human render + WORK-02 notice; `-k/-l/-j` accepted; default `search`/`search --json` byte-identical | unit (tdd, RED `f6bd1ffb`) | `GOTOOLCHAIN=go1.26.6 go test -count=1 ./internal/cli/ -run 'TestSearchFullCmd\|TestSearchFlagShortForms\|TestRenderFullLine\|TestSearchCmd\|TestNotice'` | ✅ `internal/cli/query_cli_test.go`, `notice_test.go` | ✅ green |
| 03-02.2 | 03-02 | 2 | VERB-04 | T-03-02-03 / T-03-02-06 | `daemon unlock [path]` is `unlock.go` moved verbatim (`MaximumNArgs(1)`, zero own flags, absent-lock / dead-pid-lock behaviour); `ErrStoreLocked`/`ErrLockLive` messages name `codegraph daemon unlock` | unit (tdd, RED `4215e42f`) | `GOTOOLCHAIN=go1.26.6 go test -count=1 ./internal/cli/ -run 'TestDaemonUnlockCmd\|TestIndexForceRefusesWhileStoreIsHeld'` | ✅ `internal/cli/daemon_test.go`, `index_lock_test.go` | ✅ green |
| 03-02.3 | 03-02 | 2 | VERB-03, VERB-04, VERB-06, VERB-08 | T-03-02-01 / T-03-02-05 / T-03-02-07…09 | Hidden `DisableFlagParsing` stubs execute nothing (no engine open, stale lock survives), stdout empty, non-nil error → exit 1; `--help` of each stub accounted for by a reasoned allowlist line; reference regenerated as a reviewed diff; ONE regex-conformant `feat(cli)!:` commit with a `BREAKING CHANGE:` footer | unit (tdd, RED `4da74784`) + generated-reference guard + commit-shape gate | `GOTOOLCHAIN=go1.26.6 go test -count=1 ./internal/cli/ -run 'TestQueryStub\|TestUnlockStub\|TestEveryRegisteredFlagIsAccountedFor'`; `GOTOOLCHAIN=go1.26.6 task docs:cli:drift`; `git log --grep='^feat(cli)!: fold query' -1` + footer/regex asserts | ✅ `internal/cli/renamed_test.go`, `cli_reference_test.go`, `testdata/cli-reference-allowlist.txt`; feat `5d69ee2e` | ✅ green |
| 03-02.3 (post-review) | review fix | — | VERB-03, VERB-04 | T-03-02-01 | The compiled binary prints EXACTLY the two D-06 lines once (via `main.go`'s single exit path), stdout empty, exit 1 — the third duplicated line WR-01 found is gone | integration (real binary; RED against the pre-fix binary, then GREEN `fa81672c`) | `GOTOOLCHAIN=go1.26.6 go test -count=1 ./test/integration/ -run TestRenamedStubsPrintExactlyOnce -v` | ✅ `test/integration/renamed_stubs_test.go` | ✅ green |
| 03-03.1 | 03-03 | 3 | VERB-07 | T-03-03-01 / T-03-03-02 | A re-visible `query` stub (`Hidden: false`, one token) turns `task docs:cli:drift` RED (exit 201, `+## codegraph query`) and `TestEveryRegisteredFlagIsAccountedFor` RED (`stale allowlist entry: codegraph query`); revert byte-clean; two regenerations `cmp`-identical | RED demonstration (one-shot, recorded) — reproduced independently by the verifier | mutation + `GOTOOLCHAIN=go1.26.6 task docs:cli:drift` + `go test -run TestEveryRegisteredFlagIsAccountedFor` + `git diff --quiet -- internal/cli/renamed.go` | ✅ `03-MUTATION-LOG.md` § Family (b) | ✅ recorded (durable guard = the two gates themselves, green at HEAD) |
| 03-03.2 | 03-03 | 3 | VERB-06, VERB-07 | T-03-03-03…06 | `__complete ""` lists neither stub (hidden `man` = positive control); `codegraph man <dir>` has `codegraph-daemon-unlock.1` and no stub page; zero diff under `internal/mcp`/`testdata/wireoracle`/`test/wireoracle` across the phase; 8 MCP tool names unchanged; wire oracle green | generated-surface probes + frozen-transcript test | `<bin> __complete ""`, `<bin> __complete daemon ""`, `<bin> man <tmpdir>`; `git diff --quiet 5d69ee2e^ HEAD -- internal/mcp testdata/wireoracle test/wireoracle`; `GOTOOLCHAIN=go1.26.6 task test:wireoracle` | ✅ `03-MUTATION-LOG.md` § Generated-surface and MCP proofs; `test/wireoracle` | ✅ green |
| 03-04.1 | 03-04 | 4 | VERB-08 | T-03-04-01…03 | Backlog row `### Phase 999.5: remove the query/unlock rename stubs` written ONLY by `gsd-tools phase add --id 999.5` (additions only, no version token in the heading, milestone phase filter unchanged, `roadmap validate` clean) | tool-verb gate | `node -e '…getMilestonePhaseFilter…'` (three milestone dirs, no `999.`); `gsd-tools roadmap validate` | ✅ `.planning/ROADMAP.md` § Backlog; `ef3cf0c4` | ✅ green |
| 03-04.2 | 03-04 | 4 | VERB-05, VERB-08 | T-03-04-04…06 | After-census with the identical instrument and control: 4 lines / 2 files, all inside `internal/cli/renamed.go`'s doc comment and the two allowlist lines; commit audit `5d69ee2e^..HEAD`: one `feat(`, one `!:`, rest `test(`/`docs(`/`chore(`, no CI-skip marker | one-shot census + commit audit | the Family (a) `rg` invocation; `git log --format=%s F^..HEAD` regex audit | ✅ `03-MUTATION-LOG.md` § Family (c); `8adea168` | ✅ recorded |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [x] `internal/cli/query_cli_test.go` — `TestSearchFullCmd`, `TestSearchFlagShortForms`, `TestRenderFullLine` (VERB-01/VERB-02; RED `f6bd1ffb` → GREEN `5d69ee2e`)
- [x] `internal/cli/daemon_test.go` `TestDaemonUnlockCmd` + `index_lock_test.go` assertion flip (VERB-04; RED `4215e42f` → GREEN `5d69ee2e`)
- [x] `internal/cli/renamed_test.go` — `TestQueryStub`, `TestUnlockStub` (VERB-03/VERB-04; RED `4da74784` → GREEN `5d69ee2e`, tightened to exact error text at `fa81672c`)
- [x] `test/integration/renamed_stubs_test.go` — `TestRenamedStubsPrintExactlyOnce` against the real binary (WR-01; RED against the pre-fix binary → GREEN `fa81672c`)
- [x] `03-MUTATION-LOG.md` Families (a)/(b)/(c) + Baseline — VERB-05/VERB-07 RED demonstrations and the byte-identity reference digests
- [x] No framework install needed — every gap closed inside the existing Go conventions; the RED-first discipline was satisfied with `test(03-02):` commits (the TDD runtime gate's contract), not with the TAP-only `check tdd-red-evidence` verb (no `go test` support — Go RED verified by `--- FAIL:` transcript, as in every earlier phase)

*If none: "Existing infrastructure covers all phase requirements."*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| The re-frozen `docs/CLI-REFERENCE.md` diff (+28/−49, three change groups) and the two reasoned allowlist lines are the intended surface | VERB-06, VERB-07 | D-11/D-13 reserve the re-freeze review for the executor and then the maintainer at end of phase; the drift gate proves "generator output", not "intended surface" | Accepted by the maintainer 2026-09-16 (`03-UAT.md` test 1) |
| The stubs' stderr shape after WR-01 (one error carrying both D-06 lines; no third line) satisfies D-05/D-06 | VERB-03, VERB-04 | Supersedes 03-02's literal "two `Fprintln` + one error" wording — a maintainer call on the contract, pinned afterwards by `TestRenamedStubsPrintExactlyOnce` | Accepted by the maintainer 2026-09-16 (`03-UAT.md` test 2) |
| The release-notes half of VERB-08 (the CHANGELOG's BREAKING CHANGES entry) | VERB-08 | Written by release-please from the squash-merged PR title at ship time, outside this phase; the in-phase half (the `BREAKING CHANGE:` footer on `5d69ee2e`) is asserted by the commit audit | Inspect the CHANGELOG after release-please runs (`/gsd-ship`); the PR title must be the same `feat(cli)!:` subject |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies (31/31 verify commands resolved and carry a stated failing direction — plan-checker probes, re-run after the 03-02 amendment)
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags (`go test -count=1` one-shot; `task test:wireoracle` one-shot)
- [x] Feedback latency: quick command ~5 s; full suite ~2 min, run per wave
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** validated 2026-09-16 by /gsd-validate-phase (autonomous run — no gaps; manual-only rows are by explicit decision)

## Validation Audit 2026-09-16

| Metric | Count |
|--------|-------|
| Gaps found | 0 — every automatable requirement has a durable guard (VERB-01/02 unit tests through the cobra tree; VERB-03/04 unit tests + a real-binary integration test; VERB-06 the reference drift gate + `TestEveryRegisteredFlagIsAccountedFor`; VERB-07 the wire-oracle transcripts + the two gates the RED demonstration exercised); VERB-05 and the commit-shape half of VERB-08 are one-shot, positive-controlled records by design (D-10, D-15); the release-notes half of VERB-08 is manual-only (post-merge) |
| Resolved | 0 (nothing to add) |
| Escalated | 0 |

Post-execution note: the code-review fix loop changed the stubs to return one two-line error (WR-01, `fa81672c`) after the plan's literal shape printed a duplicated third line through `main.go`; the new integration test asserts the exact two lines on the compiled binary. Pre-existing, unrelated: `internal/daemon` is load-flaky under cross-package `go test` (WINDOWS #37 — reproduced on the pre-fold commit) and `tests/browse-page.test.ts` hit its 15 s vitest timeout under machine load ~35 while passing alone; neither test is in this phase's scope.
