---
phase: 2
slug: golden-harness-re-authoring-re-freeze
status: validated
nyquist_compliant: false
wave_0_complete: true
created: 2026-08-14
validated: 2026-08-16
validation_mode: retroactive
automated_rows: 5
manual_only_rows: 1
tests_executed: 79
---

# Phase 2 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go `testing` (stdlib), plus `go-sdk` MCP in-process client for the CLI==MCP trio |
| **Config file** | none — `testdata/golden/*_test.go` are a normal `testing` package reached only via `go test ./testdata/golden/...` (GOLDEN-01) |
| **Quick run command** | `go test -count=1 ./testdata/golden/...` |
| **Full suite command** | `task test:golden` (CI runs this as its own step) |
| **Estimated runtime** | quick ~20s · full golden suite longer (locked-corpus indexing) |

> **`-count=1` is mandatory** — recorded repo gotcha: a cached `ok` can mask a real regression.

---

## Sampling Rate

- **After every task commit:** `go test -count=1 ./testdata/golden/...` (fast; behavioral corpus is in-repo)
- **After every plan wave:** `go test -count=1 ./testdata/golden/...` over the whole re-authored suite
- **Before `/gsd-verify-work`:** `go test -count=1 ./...` AND `go test -count=1 ./testdata/golden/...` both green (CODE-02 criterion 1)
- **Max feedback latency:** ~20s on the per-task loop

---

## Per-Task Verification Map

> Reconciled retroactively by `/gsd-validate-phase 2` on 2026-08-16. Every declared command was
> **executed**; static checks were positive-controlled before their zeros were accepted.

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | Result | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|--------|--------|
| 02-01-T1 | 02-01 | 1 | CODE-02 | — | N/A | static+test | `rg "parity" testdata/golden/` → empty (harness) + `go test -count=1 ./... && go test -count=1 ./testdata/golden/...` | **0 hits**; build green | ✅ green |
| 02-02-T1 | 02-02 | 2 | FIXT-04 | T-02-01 | Do **not** touch `NOTICE`/README origin attribution (licence, not framing) | static | `rg -i "weft\|colbymchenry\|mcp-capture\|capture.sh" testdata/golden/` → empty (harness scope) | **0 hits** in scope; `NOTICE` retains 2 | ✅ green |
| 02-02-T2 | 02-02 | 2 | FIXT-05 | — | Behavioral case map intact; no targeted case lost to the rename | unit+behavioral | `go test -count=1 ./testdata/golden/... -run TestCorpusBehavior` | **24 PASS** | ✅ green |
| 02-03-T1 | 02-03 | 3 | FIXT-06 | T-02-03 | capture-to-temp-then-move; non-empty + `{` marker assertion before install | e2e re-baseline | `go test -count=1 ./testdata/golden/... -run TestReFrozenGoldensValid` | **26/26 goldens verified** | ✅ green |
| 02-04-T1 | 02-04 | 4 | FIXT-06 | T-02-04 | Never index an unverified checkout — `task corpora:assert` (four-part) precedes gocapture reading locked trees | static | `golden:regen` invokes `corpora:assert` before capture | present & invoked; gocapture has **0** raw `CORPUS_*` env resolution | ✅ green |
| 02-04-T2 | 02-04 | 4 | CODE-02 crit. 2 | — | Single-cause re-freeze diff | **manual** | Review judgment — see Manual-Only | performed | ✅ performed |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

**Full-package cross-check:** `go test -count=1 ./testdata/golden/...` → **79 PASS / 0 FAIL**.

**Static-check integrity.** Both `rg` zeros above were positive-controlled before being accepted:
the same patterns return 2,574 (`parity`) and 2 (`colbymchenry`, in `NOTICE`) hits outside the
harness scope, confirming the instrument matches where matches should exist. An unchecked zero
cannot distinguish absence from a misaimed search.

---

## Wave 0 Requirements

All Wave 0 items complete — verified present in the tree at commit `3598bb2`.

- [x] `testdata/golden/gocapture/main.go` — present; locked-corpus + `corpus/behavioral/` capture specs, guarded by `TestGoSideFixturesRegenerated` (2 references)
- [x] `corpus/behavioral/CASES.json` — present; 4 cases, asserted exactly by `TestGoldenScenarioCountIsExact`
- [x] `testdata/golden/behavioral_test.go` — present; `golden_parity_test.go` is **absent**, confirming the rename completed rather than duplicating
- [x] A byte-identity / non-empty assertion over every re-frozen golden — delivered as `TestReFrozenGoldensValid`, which `os.ReadFile`s each enumerated golden and fatals on a missing or empty file, so a bare or absent golden **cannot** read as satisfied

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| One reviewed re-freeze diff, every changed line attributable to one named cause | CODE-02 criterion 2 | Review judgment — a diff must be read, not asserted | Read the re-freeze diff; confirm every golden byte traces to "re-captured from Go output against locked corpora" and no identifier moved |

---

## Validation Sign-Off

- [x] All tasks have automated verify or a documented manual-only rationale
- [x] Sampling continuity: no 3 consecutive tasks without automated verify (5 automated rows precede the single manual row)
- [x] Wave 0 covers all MISSING references — all 4 items complete
- [x] No watch-mode flags
- [x] Every test command forces `-count=1`
- [x] Feedback latency < 30s on the per-task loop
- [x] Every declared command **executed**, not read; every static zero positive-controlled
- [ ] `nyquist_compliant: true` — **not set**, see below

**`nyquist_compliant: false` — PARTIAL, by decision, not by omission.** All four Phase 2
requirements (CODE-02, FIXT-04, FIXT-05, FIXT-06) carry automated verification and all are green.
The single manual-only row is **CODE-02 criterion 2** — *"the rename and the re-freeze land as
separate reviewed diffs, each with every changed line attributable to one named cause."* That is a
claim about how a diff was authored, not a behavior a test can observe: any automated check would
either compare the two commits' file sets (which proves nothing about attribution) or go vacuous.
It was performed as review judgment during 02-04 and is recorded in `02-04-SUMMARY.md`. The phase is
PARTIAL for this reason alone.

## Validation Audit 2026-08-16

| Metric | Count |
|--------|-------|
| Rows in map | 6 |
| Automated & green | 5 |
| Manual-only (by design, performed) | 1 |
| Gaps found | 0 |
| Resolved | 0 |
| Escalated | 0 |
| Tests executed | 79 PASS / 0 FAIL |

No phantom commands in this phase — every seeded `-run` pattern matched real tests, unlike Phase 1
where two matched nothing while exiting 0.

**Approval:** validated 2026-08-16
