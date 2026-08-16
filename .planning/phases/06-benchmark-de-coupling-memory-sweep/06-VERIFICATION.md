---
phase: 06-benchmark-de-coupling-memory-sweep
verified: 2026-08-16T21:20:00Z
status: human_needed
score: 3/5 requirements fully PASS, 2/5 PARTIAL (both partials are honestly disclosed by the
  phase's own status tokens and require a maintainer/live-session action outside this phase's
  automatable scope)
behavior_unverified: 0
overrides_applied: 0
re_verification:
  previous_status: gaps_found
  previous_score: 2/5
  gaps_closed:
    - "MEM-01: '## Supersedes executed (MEM-01)' and '## Completeness and accepted gaps' sections
      now exist in 06-MEMORY-SWEEP.md with printed approved/executed set-equality and per-scope
      arithmetic (supersede + leave-historical = total)"
    - "MEM-02 store half: '## MEM-02 — store half' section now exists in
      06-LIVE-VERIFICATIONS.md carrying MEM02_STORE_STATUS=accepted-by-d15-evidence-standard"
  gaps_remaining: []
  regressions: []
deferred: []
human_verification:
  - test: "Start a genuinely fresh session in this repository and read the memory recall it
      surfaces at startup (per .claude/CLAUDE.md plus whatever the harness injects from
      .planning/), plus the engram spine recall for repo:github.com/seanb4t/codegraph-go"
    expected: "Nothing recalled describes codegraph-go as a port, a parity project or a drop-in
      replacement in the present tense, and nothing states a future obligation to another
      implementation's behaviour"
    why_human: "MEM-02's own success criterion is a live session's recall, not a static check. The
      phase's own artifacts (06-05, 06-06, and this closure commit) all decline to perform this and
      record the fact explicitly (MEM02_FILES_STATUS=pending-fresh-session-not-performed,
      MEM02_STORE_STATUS=accepted-by-d15-evidence-standard) rather than claiming it on silence —
      this verifier inherits the same limitation and cannot spawn an independent fresh session to
      observe the recall itself."
  - test: "Push branch gsd/v0.11.0-standalone-project-identity (or land it) and dispatch
      .github/workflows/bench.yml with job=publish; inspect the resulting run's job summary and
      the uploaded publish-results artifact"
    expected: "The job summary shows an absolute per-corpus table (files/s, bytes/s, query
      latency, peak RSS, cold start) for weft-go and cockroachdb-pebble; the publish-results
      artifact uploads without if-no-files-found tripping; no step in the job installs or invokes
      any implementation other than the freshly-built Go binary"
    why_human: "BENCH-03 requires an actual bench.yml run. The branch is confirmed unpushed
      (git ls-remote --exit-code --heads origin gsd/v0.11.0-standalone-project-identity exits 2,
      re-verified independently in this re-verification pass at commit d3b4f37); no CI run exists
      for it. This is a maintainer action the phase's own maintainer explicitly declined to perform
      in this session (checkpoint decision: pending-no-ci-run, not dispatch-now)."
---

# Phase 6: Benchmark De-coupling & Memory Sweep Verification Report

**Phase Goal:** The project publishes its own absolute performance numbers with no second
implementation in the picture, and an agent that starts a session afterward recalls codegraph-go
as a project rather than as a port.

**Verified:** 2026-08-16 (re-verification, second pass)
**Status:** human_needed
**Re-verification:** Yes — after gap closure at commit d3b4f37 (previous pass: 91354f7, gaps_found)

## Method

This is a re-verification pass. Per the task brief, BENCH-01 and BENCH-02's prior VERIFIED
findings and the BENCH-03 / MEM-02-file-half deferral assessments from the 91354f7 pass are
accepted and spot-checked rather than fully re-derived (`git diff --stat 91354f7 d3b4f37` confirms
the closure commit touched only `06-MEMORY-SWEEP.md` and `06-LIVE-VERIFICATIONS.md` — no code or
`docs/BENCHMARKS.md` changed, so nothing underlying BENCH-01/BENCH-02 could have regressed). The
primary work this pass: independently confirm or refute that commit d3b4f37 genuinely closes the
one gap the prior pass found — 06-06-PLAN.md Task 3's missing artifact-contract sections — rather
than merely satisfying its grep gate cosmetically.

**The plan's own acceptance command (06-06-PLAN.md Task 3, `<automated>` block, verbatim) was run
from the repo root and is the authority here:**

```
SUPERSEDE_VERDICTS=4 DELETE_OR_OVERWRITE_VERDICTS=0
APPROVED_SECTION_RAW_LINES=122 EXECUTED_SECTION_RAW_LINES=28
APPROVED_SUPERSEDE_IDS=4 APPROVED_UNIQUE=4 EXECUTED_SUPERSEDE_IDS=4 EXECUTED_UNIQUE=4
SUPERSEDE_SET_EQUALITY=true
MEM02_STORE_STATUS_TOKENS=1
SWEEP_RECORD_CLOSED=true
EXIT_CODE=0
```

This matches the expected shape exactly (4/4 supersede verdicts, 0 delete/overwrite, both raw-line
counts > 0, approved and executed sets each size 4 with 4 unique IDs, set equality true, exactly
one MEM02_STORE_STATUS token, gate closed).

**Substantive-vs-cosmetic check beyond the gate itself**, per the task brief's explicit instruction
not to stop at a green grep:

- **`## Supersedes executed (MEM-01)` rows are real, not grep-shaped filler.** Each of the 4 rows
  names the actual operation called (`mcp__engram__supersede_memory`), a returned identifier in
  both short and full UUID form (e.g. `xj1stbrsw6` / `fc4e8512-70a3-4af9-9bcc-5368eb09cd18`), and
  the response shape (`{id, short_id}`). This satisfies the plan's own bar — "a row with no
  returned identifier is an unproven write" and "an identifier with no operation name does not
  distinguish a supersede from an update" — both conditions are met by every row. The section also
  independently re-states the two-way re-query proof (`get_memory` shows the original intact with
  `superseded_by` stamped; `search_memory` on the superseded terms returns only the correcting
  records), matching what the prior pass already found credible in 06-06-SUMMARY.md.
- **`## Completeness and accepted gaps` prints real arithmetic, not implied totals.** The section
  states `supersede 4  +  leave-historical 165  =  169 total enumerated` for the repo scope and
  `supersede 0  +  leave-historical 3  =  3 total enumerated` for the rule scope, explicitly
  invoking rule `84d1gfpywd` ("Arithmetic printed, not implied"). It also states the workspace
  overlay scope was empty by non-existence, not by having been skipped, and names the two accepted
  gaps (165 leave-historical records classified by criterion and sampled rather than individually
  listed; MEM-02's fresh-session recall not performed) rather than burying them.
- **`MEM02_STORE_STATUS=accepted-by-d15-evidence-standard` is the honest token, not the weaker
  choice dressed up.** The section states plainly that the fresh-session recall check was NOT
  performed (an orchestrator session cannot spawn an independently fresh session of itself, and a
  worktree-spawned executor carries no `mcp__engram__*` tools) and grounds the acceptance instead
  in the same re-query evidence used for the MEM-01 rows. Choosing
  `accepted-by-d15-evidence-standard` over `verified-fresh-session` here is correct behavior, not
  a misrepresentation — the stronger token was not earned and the artifact does not claim it was.

**Verdict on the closure: substantive, not cosmetic.** The gate's automated shape (`SUPERSEDE_SET_EQUALITY=true`,
`MEM02_STORE_STATUS_TOKENS=1`) is backed by content that actually satisfies the underlying
intent — named operations, real returned identifiers, printed per-scope arithmetic, and an honestly
chosen (not inflated) status token. The gap the prior pass found is closed.

## Goal Achievement

### Observable Truths (per-requirement)

| # | Requirement | Truth | Status | Evidence |
|---|---|---|---|---|
| 1 | BENCH-01 | `docs/BENCHMARKS.md` publishes absolute throughput/query-latency/peak-RSS figures with methodology and the regression-gate description, no head-to-head table | ✓ VERIFIED (spot-checked, unchanged since 91354f7) | `git diff --stat 91354f7 d3b4f37` confirms `docs/BENCHMARKS.md` untouched by the closure commit; file still present with no `headtohead` occurrences (independent `rg -o` re-check, 0 hits). |
| 2 | BENCH-02 | `tools/bench` contains no comparison runner; `CheckRegression` against the committed `baseline.json` still fires on a real regression | ✓ VERIFIED (spot-checked, unchanged since 91354f7) | `git diff --stat 91354f7 d3b4f37` confirms `tools/bench/`, `.github/workflows/bench.yml` untouched by the closure commit; `headtohead` census re-run, 0 hits. Prior pass's mutation-log review stands — nothing in it could have regressed since no relevant file changed. |
| 3 | BENCH-03 | `bench.yml` runs and publishes the absolute numbers without invoking another implementation | ⚠️ PARTIAL — deliberately deferred (unchanged) | `git ls-remote --exit-code --heads origin gsd/v0.11.0-standalone-project-identity` re-run this pass, exit 2 — branch still unpushed, no CI run exists. Correct disclosed state, not a defect; routed to human verification below. |
| 4 | MEM-01 | Every engram spine record whose durable content asserts retired framing is superseded — none overwritten, none deleted — reconciled by the phase's own designed acceptance record | ✓ VERIFIED (gap closed) | The plan's own acceptance command (`SWEEP_RECORD_CLOSED=true`, full output above) passes. `## Supersedes executed (MEM-01)` and `## Completeness and accepted gaps` sections now exist in `06-MEMORY-SWEEP.md`, independently read in full: 4 named-operation rows with returned identifiers, printed per-scope arithmetic (`4 + 165 = 169`, `0 + 3 = 3`), and the approved/executed short_id sets verified set-equal by the gate's own diff (empty diff, `SUPERSEDE_SET_EQUALITY=true`). Substance judged real, not grep-shaped — see Method section above. |
| 5 | MEM-02 | A session started after the sweep recalls no memory describing codegraph-go as a port or parity project in the present tense | ⚠️ PARTIAL — store-half closure verified, file-half live check still deferred | **Store half (gap closed):** `## MEM-02 — store half` section now exists in `06-LIVE-VERIFICATIONS.md` with `MEM02_STORE_STATUS=accepted-by-d15-evidence-standard`, independently read in full; the token is honestly chosen (fresh-session check explicitly stated as not performed, acceptance grounded in the same re-query evidence as MEM-01). **File half (unchanged from 91354f7, not part of this closure):** `MEM02_FILES_STATUS=pending-fresh-session-not-performed` still stands; no fresh session has been spawned to observe actual recall. Routed to human verification below. |

**Score:** 3/5 requirements fully VERIFIED (BENCH-01, BENCH-02, MEM-01); 2/5 PARTIAL (BENCH-03,
MEM-02) — both partials are for reasons rooted in the phase's own design (deferred CI dispatch,
live fresh-session recall not spawnable by this verifier), each carrying an explicit, honest status
token rather than a silent gap.

### Deferred Items

None — no gap in this phase maps to work explicitly scheduled in a later milestone phase. BENCH-03
and MEM-02's file-half live check are maintainer/session actions outside any phase's automatable
scope, not deferred-to-a-later-phase items; they are routed to human verification instead.

### Required Artifacts

| Artifact | Expected | Status | Details |
|---|---|---|---|
| `docs/BENCHMARKS.md` | Absolute-only rewrite | ✓ VERIFIED (unchanged) | Untouched by closure commit; no comparison content |
| `tools/bench/publishcheck/{main,main_test}.go` | Pure-Go publish-JSON verifier | ✓ VERIFIED (unchanged) | Untouched by closure commit |
| `.github/workflows/bench.yml` | `publish` job, no `headtohead` | ✓ VERIFIED (unchanged) | Untouched by closure commit; `publish:` present, zero `headtohead` hits |
| `06-MEMORY-SWEEP.md` | Full D-13/D-14 verdict record, MEM-01 completeness sections | ✓ VERIFIED (gap closed) | `## Supersedes executed (MEM-01)` and `## Completeness and accepted gaps` now present with substantive rows and printed arithmetic |
| `06-LIVE-VERIFICATIONS.md` | BENCH-03 + MEM-02 (both halves) status tokens | ✓ VERIFIED (gap closed) | `## MEM-02 — store half` section now present with honest `MEM02_STORE_STATUS=accepted-by-d15-evidence-standard` token; BENCH-03 and MEM-02 file-half tokens unchanged and still correct |
| `.claude/CLAUDE.md`, `.planning/PROJECT.md`, `.planning/STATE.md` | Present-tense/forward-looking framing swept | ✓ VERIFIED (unchanged, spot-checked) | Untouched by closure commit |

### Key Link Verification

| From | To | Via | Status | Details |
|---|---|---|---|---|
| Approved MEM-01 supersedes | Executed engram writes | `supersede_memory` calls | ✓ WIRED (gap closed) | `## Supersedes executed (MEM-01)` now connects "approved" (Engram enumeration section) to "executed" (this section) with an explicit set-equality diff run by the plan's own gate — previously this reconciliation record was missing from the committed artifact; now present |
| `.github/workflows/bench.yml` `publish` job | `tools/bench/runner -mode publish` | inline `go run` invocation | ✓ WIRED (structurally proven; not live-run, unchanged) | No independent CI run exists (BENCH-03 gap, routed to human verification) |

### Probe / Acceptance Command Execution

| Command | Source | Result | Status |
|---|---|---|---|
| 06-06-PLAN.md Task 3 `<automated>` gate (verbatim) | `.planning/phases/06-benchmark-de-coupling-memory-sweep/06-06-PLAN.md:370` | `SWEEP_RECORD_CLOSED=true`, exit 0, full output matches expected shape (see Method section) | PASS |

### Anti-Patterns Found

No `TODO`/`FIXME`/`XXX`/`HACK`/`PLACEHOLDER` debt markers found in the files touched by the closure
commit (`06-MEMORY-SWEEP.md`, `06-LIVE-VERIFICATIONS.md`). No stub content, no grep-satisfying
filler — see the substantive-vs-cosmetic assessment in Method above.

### Requirements Coverage

| Requirement | Source Plan | Status | Evidence |
|---|---|---|---|
| BENCH-01 | 06-04 | ✓ SATISFIED (unchanged) | `docs/BENCHMARKS.md` |
| BENCH-02 | 06-01, 06-02 (partial), 06-03 | ✓ SATISFIED (unchanged) | Runner removal + mutation rehearsal |
| BENCH-03 | 06-02, 06-04 | ⚠️ NOT YET SATISFIED (by design, disclosed) | No CI run exists |
| MEM-01 | 06-06 | ✓ SATISFIED (gap closed) | Complete acceptance record now present in `06-MEMORY-SWEEP.md` |
| MEM-02 | 06-05, 06-06 | ⚠️ PARTIALLY SATISFIED | Store-half token now honestly recorded; file-half live check still pending |

No orphaned requirements — all five BENCH/MEM requirements are claimed by a phase-6 plan.

## Gaps Summary

The one gap from the prior verification pass (91354f7) — 06-06's Task 3 missing its own
plan-mandated artifact-contract sections in `06-MEMORY-SWEEP.md` and
`06-LIVE-VERIFICATIONS.md` — is **closed** by commit d3b4f37. This was independently confirmed two
ways: (1) running the plan's own acceptance gate verbatim, which passes cleanly
(`SWEEP_RECORD_CLOSED=true`), and (2) reading the added sections in full to confirm they are
substantive rather than shaped only to satisfy the grep — real operation names, real returned
identifiers, printed per-scope arithmetic, and an honestly chosen (not inflated) MEM02 status
token.

What remains, unchanged from the prior pass and not part of this closure's scope, are the two
deliberately-deferred items that require an action outside any phase's automatable reach:

1. **BENCH-03's CI run.** The branch is still unpushed; no `bench.yml` run has executed the
   `publish` job. Routed to human verification.
2. **MEM-02's file-half live fresh-session recall check.** No independently fresh session has been
   spawned to observe actual startup recall. Routed to human verification.

Both are recorded with explicit, honest status tokens in the phase's own artifacts
(`BENCH03_STATUS=pending-no-ci-run`, `MEM02_FILES_STATUS=pending-fresh-session-not-performed`), and
both are maintainer/session actions the phase's own checkpoints explicitly declined to perform
rather than silently omitted work.

---

*Verified: 2026-08-16 (re-verification)*
*Verifier: Claude (gsd-verifier)*
