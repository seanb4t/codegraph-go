---
phase: 1
slug: corpus-selection-by-measurement
# status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6)
# audit-milestone §5.5 distinguishes NOT-VALIDATED (draft) from PARTIAL (validated + nyquist_compliant: false) (#2117)
status: validated
nyquist_compliant: false
wave_0_complete: true
created: 2026-08-14
validated: 2026-08-16
validation_mode: retroactive
automated_rows: 6
manual_only_rows: 2
tests_executed: 127
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

> Reconciled retroactively by `/gsd-validate-phase 1` on 2026-08-16. The commands below were seeded
> at plan time as *predictions of test names*; where the executor chose a different name, the row
> records the *corrected* command and the prediction is footnoted. Every command was **executed**,
> and its passing-test count recorded — never inferred from exit status.

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | Tests | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------|--------|
| 01-01-T1 | 01-01 | 1 | FIXT-01 | T-01-01-01 | Status projection emits counts only; path fields stay blank | unit | `go test -count=1 ./internal/query/... -run TestRenderStatus` | 13 | ✅ green |
| 01-02-T1 | 01-02 | 2 | FIXT-01 | T-01-02-02 | Dense `edgesByKind` key set is **exactly** `query.RankEdges` (D-04) | unit | `go test -count=1 ./internal/query/... -run 'TestDenseEdgesByKind'` ¹ | 4 | ✅ green |
| 01-01-T2 | 01-01 | 1 | FIXT-01 | — | Sparse tally omits absent kinds, keeps unranked kinds | unit | `go test -count=1 ./internal/query/... -run 'TestStatusEdgesByKind'` | 4 | ✅ green |
| 01-06-T1 | 01-06 | 4 | FIXT-01 | T-01-06-01, T-01-07-03 | Coverage claim derived from measurement, with a positive checked-count assertion | unit (drift guard) | `go test -count=1 ./internal/corpora/... -run TestCorpusCoverageClaim` | 1 | ✅ green |
| 01-03-T1 | 01-03 | 3 | FIXT-01 | T-01-03-01 | Re-frozen `call-status` transcript matches byte-for-byte | integration | `go test -count=1 ./test/wireoracle/... -run 'TestFrozenTranscriptsMatch/call-status'` | 2 | ✅ green |
| 01-02-T2 | 01-02 | 2 | FIXT-01 | T-01-02-01 | 4th TTY render surface must not diverge from piped output | integration | `go test -count=1 ./internal/cli/present/... -run 'TestRenderStatus_'` ² | 7 | ✅ green |
| 01-04-T1 | 01-04 | 1 | FIXT-02 | T-01-04-01 | Manifest `sha` matches `^[0-9a-f]{40}$` and `repo` a strict `org/name` allowlist **before** shell interpolation | unit | `go test -count=1 ./internal/corpora/... -run 'TestValidate\|TestManifest'` ³ | 23 | ✅ green |
| 01-04-T2 | 01-04 | 1 | FIXT-02 | T-01-04-02, T-01-04-04 | Fetch is claim → verify → promote; integrity failure exits non-zero | integration/**manual** | `task corpora:fetch` twice from clean; identical tree hash | — | ✅ performed 2026-08-14 (see Manual-Only) |
| 01-07-T1 | 01-07 | 5 | FIXT-02 | T-01-07-01 | Cold-cache CI run shows the fetch step **executing**, never skipping | **CI-only** | Observed in a real CI run | — | ✅ performed (see Manual-Only) |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

¹ Seeded as `-run TestEdgesByKindDense`, which matches **zero** tests (`go test -run` exits 0 on no
match — the command read green while running nothing). Real names are
`TestDenseEdgesByKindKeySetEqualsRankEdges` / `…ExplicitZero` / `…PreservesUnrankedKind` /
`…DoesNotMutateInput` at `internal/query/render_status_test.go:323-385`.
² Seeded as `-run TestStatus`, which matches **zero** tests in that package. Real names are
`TestRenderStatus_*` at `internal/cli/present/status_test.go:51-196`, including
`TestRenderStatus_MatchesPipedSectionOrder` — the non-divergence assertion this row exists for.
³ Seeded as `-run TestCorpusManifestRejectsMalformedEntry`; the implemented equivalents live in
`internal/corpora/manifest_test.go`. Full-package run: **76 tests, 0 failures.**

**Full-package cross-check:** `go test -count=1 ./internal/corpora/...` → 76 PASS / 0 FAIL.
**Total executed for this reconciliation:** 127 passing tests, 0 failures.

---

## Wave 0 Requirements

All Wave 0 items are complete — verified present in the tree at commit `69159a3`.

- [x] Key-set-equality test asserting dense-mode `edgesByKind`'s key set is **exactly** `query.RankEdges`'s key set (D-04) — delivered as `TestDenseEdgesByKindKeySetEqualsRankEdges`, `internal/query/render_status_test.go:323`
- [x] Corpus-coverage drift guard naming the repository supplying each kind, with a positive assertion of the count checked (rule `84d1gfpywd`) — delivered as `CheckCoverage` in `internal/corpora/coverage.go`, asserting `CheckedKinds: len(query.RankEdges)` and `CheckedCorpora`, with a `checkedCorpora == 0` guard (lines 216-224)
- [x] Manifest validation test proving malformed `repo`/`sha` entries are rejected before shell interpolation — `internal/corpora/manifest_test.go`, plus the Taskfile's own manifest-anchoring refusal
- [x] `Taskfile.yml` target for corpus fetch — `corpora:fetch` and `corpora:fetch-one` (claim → verify → promote)
- [x] The corpora manifest file — `corpora/manifest.json`, sole pin authority per D-09
- [x] Measurement-record generator — `tools/corpora` (`-mode measure`), with **no write path** to `corpora/selection.json` (asserted at `tools/corpora/main.go:17`, tested at `measure_test.go:105`)
- [x] Re-freeze mechanic for `test/wireoracle/transcripts/call-status.golden` — resolved; the transcript is frozen and `TestFrozenTranscriptsMatch/call-status` passes

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Cache **miss** falls through to a real fetch rather than a skipped job | FIXT-02 | Inherently CI-observed — there is no local equivalent of `actions/cache`'s restore semantics | Push a commit changing a pinned SHA (new cache key ⇒ guaranteed miss); confirm in the run log that the fetch step executed and reported a non-zero corpus count |
| Cache **hit** restores corpora and skips the fetch | FIXT-02 | Same | Re-run the identical workflow; confirm the restore reported a hit and the fetch step was skipped |

> Both rows are a matched pair on purpose. A guard that only ever observes the hit path passes vacuously the day the fetch breaks (rule `84d1gfpywd`).

---

## Validation Audit 2026-08-16

| Metric | Count |
|--------|-------|
| Rows in map | 9 |
| Automated & green | 7 |
| Manual-only (by design, performed) | 2 |
| Gaps found | 2 (both **phantom commands**, not missing coverage) |
| Resolved | 2 (corrected to the real test names) |
| Escalated | 0 |
| Tests executed | 127 PASS / 0 FAIL |

**Two phantom commands found and corrected.** `-run TestEdgesByKindDense` and `-run TestStatus`
each matched **zero** tests while exiting 0 — `go test -run` reports success when its pattern
selects nothing. Both were scored by counting `--- PASS` lines rather than trusting exit status,
which is the only way this class is visible. In both cases the *behavior* was fully covered under a
different test name, so these were map-accuracy defects, not coverage gaps. Recorded rather than
silently rewritten, because a validation map whose commands run nothing is the same vacuous-pass
family as rule `84d1gfpywd`.

---

## Validation Sign-Off

- [x] All tasks have automated verify or a documented manual-only rationale
- [x] Sampling continuity: no 3 consecutive tasks without automated verify (the 2 manual-only rows are adjacent but not a run of 3)
- [x] Wave 0 covers all MISSING references — all 7 items complete
- [x] No watch-mode flags
- [x] Every test command forces `-count=1`
- [x] Feedback latency < 30s on the per-task loop
- [x] Every declared command **executed**, not read; passing-test counts recorded

**`nyquist_compliant: false` — PARTIAL, by decision, not by omission.** Two FIXT-02 behaviors have
no automated local command: the cache-**miss**→fetch path and the cache-**hit**→skip path are
inherently CI-observed (there is no local equivalent of the runner's cache restore semantics). Both
were performed and recorded — the twice-from-clean reproducibility check on 2026-08-14 produced
4/4 byte-identical trees across `hugo`, `guava`, `requests` and `serilog`. They remain manual-only
by design, which is why this phase is PARTIAL rather than COMPLIANT.

**Approval:** validated 2026-08-16
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
