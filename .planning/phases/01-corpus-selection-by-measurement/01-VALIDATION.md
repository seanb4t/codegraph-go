---
phase: 1
slug: corpus-selection-by-measurement
# status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6)
# audit-milestone §5.5 distinguishes NOT-VALIDATED (draft) from PARTIAL (validated + nyquist_compliant: false) (#2117)
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-08-14
---

# Phase 1 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go standard `testing` package (`go test`) |
| **Config file** | none — `Taskfile.yml` is the single definition of every suite grouping (`test:unit`, `test:golden`, `test:integration`, `test:wireoracle`, `test:daemon`, `test:race`) |
| **Quick run command** | `go test -count=1 ./internal/query/... ./internal/cli/...` |
| **Full suite command** | `task test` |
| **Estimated runtime** | quick ~15s · full suite several minutes (serial by `TestTaskfileWrapperIsSerial`) |

> **`-count=1` is mandatory, not stylistic.** Recorded gotcha from v0.10.0 Phase 8: `go test` without `-count=1` can report a stale cached `ok` and mask a real regression. Every command in this file forces it.

---

## Sampling Rate

- **After every task commit:** Run `go test -count=1 ./internal/query/... ./internal/cli/...` (fast; covers the measurement-instrument half without needing fetched corpora)
- **After every plan wave:** Run `task test:golden && task test:wireoracle` (covers the golden-parity `status` subtest and the re-frozen `call-status` transcript)
- **Before `/gsd-verify-work`:** Full `task test` green, **plus** a real (not simulated) CI run in which the cache-miss→fetch path and the cache-hit→skip-fetch path have each fired at least once
- **Max feedback latency:** ~15 seconds on the per-task loop

---

## Per-Task Verification Map

> Task IDs are assigned by the planner; this table is seeded from the requirement→test map and is completed by `/gsd-validate-phase`.

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| TBD | TBD | TBD | FIXT-01 | — | N/A | unit | `go test -count=1 ./internal/query/... -run TestRenderStatus` | ✅ (`render_status_test.go`; new subtests, not a new file) | ⬜ pending |
| TBD | TBD | TBD | FIXT-01 | — | N/A | unit | `go test -count=1 ./internal/query/... -run TestEdgesByKindDense` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | FIXT-01 | — | N/A | unit (drift guard) | `go test -count=1 ./... -run TestCorpusCoverageClaim` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | FIXT-01 | — | N/A | integration | `go test -count=1 ./test/wireoracle/... -run 'TestFrozenTranscriptsMatch/call-status'` | ✅ (scenario exists; golden needs re-freeze) | ⬜ pending |
| TBD | TBD | TBD | FIXT-01 | — | N/A | integration | `go test -count=1 ./internal/cli/present/... -run TestStatus` | ✅ (4th TTY render surface — must not diverge from piped output) | ⬜ pending |
| TBD | TBD | TBD | FIXT-02 | T-01-01 | Manifest `sha` matches `^[0-9a-f]{40}$` and `repo` matches a strict `org/name` allowlist **before** any shell interpolation | unit | `go test -count=1 ./... -run TestCorpusManifestRejectsMalformedEntry` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | FIXT-02 | — | N/A | integration/manual | `task corpora:fetch` twice from clean; identical tree hash | ❌ W0 (no target exists) | ⬜ pending |
| TBD | TBD | TBD | FIXT-02 | — | N/A | CI-only | Cold-cache CI run shows the fetch step **executing**, with a positive assertion of corpus count/SHA — never a skip | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] Key-set-equality test asserting dense-mode `edgesByKind`'s key set is **exactly** `query.RankEdges`'s key set (D-04; follows the existing `TestRankEdges` precedent in `internal/query/rwr_test.go`) — covers FIXT-01 criteria 2 and 3
- [ ] Corpus-coverage drift guard: reads the committed measurement record and asserts every one of the 9 `RANK_EDGES` kinds and all 5 priority-4 languages clears its bar, **naming the repository** that supplies each — carries a positive assertion of the count it checked (rule `84d1gfpywd`)
- [ ] Manifest validation test proving malformed `repo`/`sha` entries are rejected before shell interpolation (threat T-01-01)
- [ ] `Taskfile.yml` target for corpus fetch (none exists today)
- [ ] The corpora manifest file itself (none exists today) — sole pin authority per D-09
- [ ] Measurement-record generator (Go program, shaped after `testdata/golden/gocapture/main.go`) driving `codegraph status --json` in dense mode against each fetched corpus, writing JSON keyed by `repo@SHA`
- [ ] Re-freeze mechanic for `test/wireoracle/transcripts/call-status.golden` — **no `-update`/`UPDATE_GOLDEN` flag exists anywhere in `test/wireoracle/*.go`**; either confirm the historical manual mechanic via `git log` on prior `.golden` re-freezes, or add a small capture helper

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Cache **miss** falls through to a real fetch rather than a skipped job | FIXT-02 | Inherently CI-observed — there is no local equivalent of `actions/cache`'s restore semantics | Push a commit changing a pinned SHA (new cache key ⇒ guaranteed miss); confirm in the run log that the fetch step executed and reported a non-zero corpus count |
| Cache **hit** restores corpora and skips the fetch | FIXT-02 | Same | Re-run the identical workflow; confirm the restore reported a hit and the fetch step was skipped |

> Both rows are a matched pair on purpose. A guard that only ever observes the hit path passes vacuously the day the fetch breaks (rule `84d1gfpywd`).

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Every test command forces `-count=1`
- [ ] Feedback latency < 30s on the per-task loop
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
