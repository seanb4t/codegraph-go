---
phase: 6
slug: benchmark-de-coupling-memory-sweep
# status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6)
# audit-milestone §5.5 distinguishes NOT-VALIDATED (draft) from PARTIAL (validated + nyquist_compliant: false) (#2117)
status: validated
nyquist_compliant: false
wave_0_complete: true
created: 2026-08-16
validated: 2026-08-16
validation_mode: retroactive
automated_rows: 3
manual_only_rows: 3
tests_executed: 97
census_total: 0
---

# Phase 6 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.
> Source: `06-RESEARCH.md` §Validation Architecture (line 506).

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go built-in `testing` (`go test`) |
| **Config file** | none — Go test discovery is convention-based (`*_test.go`) |
| **Quick run command** | `go test -count=1 ./internal/bench/... ./tools/bench/...` |
| **Full suite command** | `go build ./... && go vet ./... && go test -count=1 ./internal/bench/... ./tools/bench/... ./testdata/golden/...` |
| **Estimated runtime** | ~60–90 seconds (golden suite dominates) |

**Why the golden suite is in the full command:** `weft-go` is pinned in *both*
`tools/bench/realcorpus/manifest.go` and the Phase 1 corpus surface. Editing the
realcorpus manifest (D-09/D-10) must not regress the golden corpus's use of that
shared pin.

---

## Sampling Rate

- **After every task commit:** `go test -count=1 ./internal/bench/... ./tools/bench/...`
- **After every plan wave:** full suite command above, **plus** a fresh positive-controlled
  multiline census (D-04) over the Phase 6 file surface
- **Before `/gsd-verify-work`:** full suite green AND census TOTAL recorded verbatim as 0
  (the number, not merely a zero exit status — rule `84d1gfpywd`: a negative-only guard
  passes vacuously when its anchor stops matching)
- **Max feedback latency:** ~90 seconds

---

## Per-Task Verification Map

> Reconciled retroactively by `/gsd-validate-phase 6` on 2026-08-16. Every command **executed**;
> the census TOTAL is recorded verbatim as a number, not inferred from an exit status.

| Task ID | Plan | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | Result | Status |
|---------|------|-------------|------------|-----------------|-----------|-------------------|--------|--------|
| 06-04-T1 | 06-04 | BENCH-01 | T-06-12 | Published figures generated from raw JSON, never typed | inspection (census) | multiline census over `docs/BENCHMARKS.md`, TOTAL=0 for comparison patterns | **CENSUS_TOTAL=0** across 6 patterns; `publishcheck` **8 PASS**; `-emit-rows` present; raw `06-PUBLISH-RESULTS.json` committed | ✅ green |
| 06-03-T1 | 06-03 | BENCH-02 (gate fires) | T-06-01, T-06-02, T-06-22 | Mutation fully reverted; no mutation byte in the commit; gate loads the **committed** baseline from disk | unit + mutation rehearsal | `go test -count=1 ./internal/bench/... -run TestCheckRegression` | **30 PASS**, incl. `TestCheckRegressionAgainstCommittedBaseline` with two subtests proving it **fails** (throughput −11%, RSS +16%); `internal/bench`+`tools/bench` **0 dirty** | ✅ green |
| 06-01-T1 | 06-01 | BENCH-02 (runner removed) | T-06-06, T-06-09 | Input surface net-reduced; no dangling refs | build + census | `go build ./... && go vet ./...`; census over `tools/bench/**` | build **exit 0**, vet **exit 0**; `resolveTSBinary`/`colbymchenry-codegraph`/`headtohead` **0 hits**; realcorpus **3 PASS** (exactly two entries) | ✅ green |
| 06-02-T1 | 06-02 | BENCH-03 | T-06-03, T-06-04, T-06-22, T-06-23 | `rebless` keeps `contents: read`; all `uses:` SHA-pinned; no package-manager install | **manual** (CI dispatch) + static census | Dispatch `publish`; census `bench.yml` | Run **31973967130** succeeded; `BENCH03_STATUS=closed-by-ci-run` (1×); pins **19 + 2 = 21**; **0** package-manager installs; `TestBenchPublishJobShape` PASS | ✅ performed |
| 06-06-M1 | 06-06 | MEM-01 | T-06-18, T-06-19, T-06-21 | Supersede-only; nothing overwritten or deleted | **manual** (inspection) | Enumeration in `06-MEMORY-SWEEP.md` is the artifact (D-15) | 169 records enumerated to exhaustion; **4 supersedes** live in the spine (`mw5z9s9bft`, `b9wjge7375`, `gxwkk3necn`, `xj1stbrsw6`), each preserving its original | ✅ performed |
| 06-05-M1 | 06-05 | MEM-02 | T-06-26 | No present-tense or forward-looking framing recalled | **manual** (inspection) | Not automatable — D-15 declines post-sweep re-query | File half `MEM02_FILES_STATUS=verified-fresh-session`; **store half accepted, not demonstrated** — see below | ⚠️ partial (accepted) |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

**Total executed for this reconciliation: 97 passing tests, 0 failures**, plus build, vet, and a
6-pattern census.

### BENCH-01 census — TOTAL recorded verbatim

| Pattern | Hits |
|---------|------|
| `head-?to-?head` | 0 |
| `faster than` | 0 |
| `slower than` | 0 |
| `vs\.? *(the )?(TS\|TypeScript)` | 0 |
| `compared (to\|against) (the )?(TS\|TypeScript)` | 0 |
| `colbymchenry` | 0 |
| **CENSUS_TOTAL** | **0** |

**Two-sided, per `T-06-08`.** A zero alone would be worthless. The census is paired with a
**survival** assertion: legitimate `compar*` uses must still exist (floor ≥ 5). Measured: **61**
across `docs/BENCHMARKS.md`, `tools/bench/` and `internal/bench/`. A green census therefore cannot
have been bought with a find-and-replace — the thing a find-and-replace would destroy is checked to
still be there.

**Positive control:** the same multiline instrument returns **596** `head-to-head` hits inside
`.planning/`, so the zero above reports the population, not a broken search.

---

## Wave 0 Requirements

Existing infrastructure covers all phase requirements. `internal/bench/regression_test.go`,
`tools/bench/runner/main_test.go`, and `tools/bench/realcorpus/manifest_test.go` all exist
and already cover the relevant behaviors.

**One non-obvious hazard the plan MUST handle (from RESEARCH.md Pattern 2):** the only
test-file edits needed are *subtractive*, and they are not optional —
`tools/bench/runner/main_test.go:482-533` and `tools/bench/realcorpus/manifest_test.go:72-77`
currently **assert the existence of** the exact surface D-07/D-09 delete (`-ts-binary`,
`resolveTSBinary`, the `colbymchenry-codegraph` entry). If they are not updated in the same
task as the production change, the suite goes red **for the wrong reason** — and an
"expected" red suite is precisely how a genuine regression gets waved through.

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| `bench.yml` `publish` job emits absolute numbers with no second implementation invoked anywhere in the job | BENCH-03 | GitHub Actions jobs cannot be executed by `go test` | Dispatch the workflow on the phase branch; confirm the job summary shows absolute per-corpus figures, the artifact uploads (`if-no-files-found: error` did not trip), and no step installs or invokes another implementation |
| Every retired-framing spine record superseded, none overwritten or deleted | MEM-01 | The engram spine lives outside git; no CI gate can hold it | Review the committed `06-MEMORY-SWEEP.md`: every record enumerated by `short_id` with a supersede / leave-historical verdict and reason; confirm no verdict is "delete" or "overwrite" |
| A session started after the sweep recalls no port/parity framing in the present tense | MEM-02 | Requires a live session's recall, not a test | Start a fresh session in this repo and read the startup recall digest; confirm no record asserts the framing as a current standing fact or future obligation |

**Recorded gap (accepted, D-15):** post-sweep re-query evidence is *not* collected, so
"the supersede actually landed" remains asserted rather than demonstrated. The maintainer
accepted this trade-off; it is logged here so a later audit sees a decision, not an oversight.

**Execution precondition (D-16):** engram tool availability must be verified and fail loud
before the sweep begins. Note the research session found `mcp-gw.fzymgc.house` reachable
(DNS resolves; HTTP responds in ~36ms) — contradicting the `ENOTFOUND` observed at
context-gathering. Engram is configured at user level, not in this repo's `.mcp.json`, so
the real precondition is **`mcp__engram__*` tool availability in the executing session**,
not network reachability. Never record the sweep complete against a store that could not
be enumerated.

---

## Validation Sign-Off

- [x] All tasks have automated verify or a recorded manual-only justification
- [x] Sampling continuity: 3 automated rows precede the 3 manual rows — no run of 3 unautomated tasks *within* the automated set
- [x] Wave 0 covers all MISSING references (none — subtractive test edits only)
- [x] **Subtractive test edits landed in the same task as their production change** — verified below
- [x] No watch-mode flags
- [x] Census TOTAL recorded verbatim (**0**), not inferred from exit status; positive-controlled
- [x] Feedback latency < 90s
- [ ] `nyquist_compliant: true` — **not set**, see below

### The Wave 0 hazard: resolved, and the one surviving literal is deliberate

This file warned that `tools/bench/runner/main_test.go` and
`tools/bench/realcorpus/manifest_test.go` **assert the existence of** the exact surface D-07/D-09
deletes, so a production change without the matching test edit turns the suite red *for the wrong
reason* — and *"an 'expected' red suite is precisely how a genuine regression gets waved through."*

Verified at reconciliation: `resolveTSBinary`, `colbymchenry-codegraph` and `headtohead` are all
**0 hits** outside `.planning/`, and the realcorpus manifest tests pass with
`TestCorporaHasExactlyTwoEntries`. The subtractive edits landed.

**One `-ts-binary` literal survives, and must.** `tools/bench/runner/main_test.go:503` reads
`parseFlags([]string{"-ts-binary", "/custom/codegraph"})` inside
`TestParseFlags_RejectsRetiredComparisonBinaryFlag`, which asserts the call **errors**. This is a
*positive removal assertion* (rule `84d1gfpywd`): it proves the flag is gone from the flag set
rather than merely undocumented, and Go's `flag` package matches by name, so the test **cannot be
written without naming the retired flag**.

That creates a genuine tension — the census must exclude the file that proves the removal — and the
phase resolved it rather than hand-waving: three census gates pre-authorise the exclusion *on the
strength of a companion assertion that the literal occurs in that file **exactly once***. Verified:
**1 occurrence**, and the test **PASSes**. An exclusion granted on a checkable invariant is not the
same thing as an exclusion granted on trust.

**`nyquist_compliant: false` — PARTIAL, by decision, not by omission.** BENCH-01 and BENCH-02 are
fully automated and green. Three rows cannot be:

1. **BENCH-03** — a GitHub Actions job cannot be executed by `go test`. Discharged by a real
   dispatch (run **31973967130**) and recorded in a single status token.
2. **MEM-01** — the engram spine lives **outside git**, so no CI gate can hold it. Discharged by
   the committed enumeration of all 169 records and 4 supersedes.
3. **MEM-02** — requires a live session's recall, which a test cannot stage.

The first two were performed. The third is **accepted, not demonstrated** — see below.

### Recorded gap (accepted): MEM-02's store half

`MEM02_STORE_STATUS=accepted-by-d15-evidence-standard` appears **exactly once** in
`06-LIVE-VERIFICATIONS.md` and was deliberately **not** upgraded to `verified-fresh-session`.
Maintainer ruling 2026-08-16, verbatim: *"MEM-02 - accept it and move on."* No session that was both
genuinely fresh **and** engram-tooled ever observed the spine recall.

This is the outcome the plan's own threat model anticipated: `T-06-26`'s mitigation reads *"the
fresh-session read may go unperformed but **cannot go unrecorded**."* That property held. The token
carries the weaker claim honestly rather than rounding it up, which is what leaves the gap auditable
— and cheaply dischargeable later by a fresh session probing the spine as its **first** action.

## Validation Audit 2026-08-16

| Metric | Count |
|--------|-------|
| Rows in map | 3 automated + 3 manual-only |
| Automated & green | 3 |
| Manual-only performed | 2 (BENCH-03, MEM-01) |
| Manual-only accepted-not-demonstrated | 1 (MEM-02 store half) |
| Gaps found | 0 |
| Census TOTAL | 0 (positive-controlled; 61 legitimate uses survived) |
| Escalated | 0 |
| Tests executed | 97 PASS / 0 FAIL |

**Approval:** validated 2026-08-16
