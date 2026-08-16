---
phase: 06-benchmark-de-coupling-memory-sweep
verified: 2026-08-16T23:20:00Z
status: passed
score: 5/5 requirements closed — 4/5 fully VERIFIED (BENCH-01, BENCH-02, BENCH-03, MEM-01); MEM-02's
  file half live-verified and its store half ACCEPTED at the D-15 evidence standard by explicit
  maintainer ruling (2026-08-16, "MEM-02 - accept it and move on") rather than demonstrated by a
  fresh-session read. See ## Acknowledged Gaps.
behavior_unverified: 0
overrides_applied: 0
re_verification:
  previous_status: human_needed
  previous_score: 3/5
  gaps_closed:
    - "BENCH-03: closed by a real CI run — bench.yml run 31973967130 (workflow_dispatch, branch
      gsd/v0.11.0-standalone-project-identity) concluded success with the publish job green and
      all six other jobs skipped; artifact independently downloaded and read"
    - "MEM-02 file half: the live fresh-session read the prior pass could not perform was
      performed during UAT (06-UAT.md test 1, pass) in a genuinely fresh post-/clear session;
      MEM02_FILES_STATUS updated pending-fresh-session-not-performed -> verified-fresh-session"
  gaps_remaining: []
  regressions: []
  corrections:
    - "STALE ASSERTION CORRECTED: the prior pass (written at d3b4f37) stated in both its
      frontmatter human_verification block and its Observable Truths table that the branch was
      'confirmed unpushed (git ls-remote --exit-code --heads origin
      gsd/v0.11.0-standalone-project-identity exits 2) ... no CI run exists for it.' That was true
      when written and is now false. Re-derived this pass: git ls-remote exits 0 and resolves to
      9f763cae17e372c5100d814d010a1a07f1de42a6, and bench.yml run 31973967130 exists and
      succeeded. BENCH-03 moves PARTIAL -> VERIFIED and is removed from human_verification."
deferred: []
human_verification: []
---

# Phase 6: Benchmark De-coupling & Memory Sweep Verification Report

**Phase Goal:** The project publishes its own absolute performance numbers with no second
implementation in the picture, and an agent that starts a session afterward recalls codegraph-go
as a project rather than as a port.

**Verified:** 2026-08-16 (re-verification, third pass — post-UAT)
**Status:** human_needed
**Re-verification:** Yes — after the BENCH-03 CI run (9f763ca) and human UAT (86650e1). Prior
passes: 91354f7 (gaps_found, 2/5), d3b4f37 (human_needed, 3/5).

## Correction to the Prior Pass

**The prior report was factually stale on BENCH-03 and this pass corrects it.** At d3b4f37 the
report asserted, in both its frontmatter `human_verification` block and its Observable Truths
table, that the branch was "confirmed unpushed (`git ls-remote --exit-code --heads origin
gsd/v0.11.0-standalone-project-identity` exits 2) ... no CI run exists for it." That was true when
written. Commit 9f763ca landed afterward.

Re-derived independently this pass, not transcribed from the ledger or from UAT:

```
$ git ls-remote --exit-code --heads origin gsd/v0.11.0-standalone-project-identity; echo "EXIT=$?"
9f763cae17e372c5100d814d010a1a07f1de42a6	refs/heads/gsd/v0.11.0-standalone-project-identity
EXIT=0
```

The branch resolves. The prior assertion is reversed, and BENCH-03 moves ⚠️ PARTIAL → ✓ VERIFIED.

## Method

BENCH-01 and BENCH-02 were VERIFIED in the prior pass and are spot-checked rather than
re-derived. `git diff --stat d3b4f37 HEAD` shows only three files changed
(`06-LIVE-VERIFICATIONS.md`, `06-UAT.md`, `06-VERIFICATION.md`) — all planning artifacts.
`git diff --name-only d3b4f37 HEAD -- docs/BENCHMARKS.md tools/bench .github/workflows/bench.yml
internal/bench` returns **zero files**. **No file underlying BENCH-01 or BENCH-02 changed**, so
neither can have regressed. Both were nonetheless re-censused with a positive-controlled
instrument (18 `bench` hits in `docs/BENCHMARKS.md` proving the instrument is live; 0 `headtohead`
hits across `docs/BENCHMARKS.md`, `tools/bench/` and `.github/workflows/bench.yml`).

**06-06-PLAN.md Task 3's `<automated>` acceptance gate, run VERBATIM from the repo root** (not
reformulated), literal output:

```
SUPERSEDE_VERDICTS=4 DELETE_OR_OVERWRITE_VERDICTS=0
APPROVED_SECTION_RAW_LINES=122 EXECUTED_SECTION_RAW_LINES=28
APPROVED_SUPERSEDE_IDS=4 APPROVED_UNIQUE=4 EXECUTED_SUPERSEDE_IDS=4 EXECUTED_UNIQUE=4
SUPERSEDE_SET_EQUALITY=true
MEM02_STORE_STATUS_TOKENS=1
SWEEP_RECORD_CLOSED=true
EXIT_CODE=0
```

Re-run after this pass's edit to `06-LIVE-VERIFICATIONS.md`: still `SWEEP_RECORD_CLOSED=true`,
exit 0, with `BENCH03_STATUS`, `MEM02_FILES_STATUS` and `MEM02_STORE_STATUS` each carrying
exactly one token.

## Goal Achievement

### Observable Truths (per-requirement)

| # | Requirement | Truth | Status | Evidence |
|---|---|---|---|---|
| 1 | BENCH-01 | `docs/BENCHMARKS.md` publishes absolute throughput / query-latency / peak-RSS figures with methodology and the regression-gate description, and carries no head-to-head table or multipliers | ✓ VERIFIED (spot-checked, file unchanged) | `git diff --name-only d3b4f37 HEAD -- docs/BENCHMARKS.md` → empty. Census re-run positive-controlled: 18 `bench` hits (instrument live), 0 `head.?to.?head` hits. Corroborated by UAT test 18 (pass). |
| 2 | BENCH-02 | `tools/bench` contains no comparison runner; `CheckRegression` against the committed `baseline.json` still fires on a real regression, demonstrated RED against a reverted mutation | ✓ VERIFIED (spot-checked, files unchanged) | `git diff --name-only d3b4f37 HEAD -- tools/bench .github/workflows/bench.yml internal/bench` → empty; nothing could have regressed. 0 `headtohead` hits. Prior pass's mutation-log review (`06-MUTATION-LOG.md`) stands. Corroborated by UAT tests 4–8, 15–17 (all pass). |
| 3 | BENCH-03 | A `bench.yml` run publishes the absolute numbers without invoking another implementation anywhere in the job | ✓ VERIFIED (**was ⚠️ PARTIAL — prior assertion was stale**) | Run 31973967130 re-derived directly: `conclusion=success`, `event=workflow_dispatch`, `headBranch=gsd/v0.11.0-standalone-project-identity`. Publish job `success`; all 6 other jobs `skipped`. Artifact downloaded and read: exactly 2 records, repos `{weft-go, cockroachdb-pebble}`, both `"subject": "go"`, all five metrics positive, no comparison column / second subject / ratio. Full 13-step list read — no Node/npm/npx/pnpm/yarn step; full-log scan for `npm\|npx\|pnpm\|yarn\|node_modules` returns 0 against a positive-controlled instrument (49 `codegraph` hits in the same log). `if-no-files-found: error` configured at `bench.yml:154` and untripped. Corroborated by UAT test 3 (pass). |
| 4 | MEM-01 | Every engram spine record whose durable content asserts retired framing is superseded by a corrected record — none overwritten, none deleted, no record of real history removed | ✓ VERIFIED | 06-06 Task 3's acceptance gate run verbatim passes (`SWEEP_RECORD_CLOSED=true`, exit 0): 4 supersede verdicts, 0 delete/overwrite, approved and executed short_id sets set-equal (empty `diff`), both sections non-empty. Substance judged real in the prior pass and unchanged since — named operations, real returned identifiers, printed per-scope arithmetic (`4 + 165 = 169`; `0 + 3 = 3`). |
| 5 | MEM-02 | A session started after the sweep recalls no memory describing codegraph-go as a port or parity project in the present tense | ✓ CLOSED — file half VERIFIED, store half ACCEPTED by maintainer ruling (not demonstrated) | **File half (newly closed):** UAT test 1 `pass` — a genuinely fresh post-`/clear` session assembled and read its own startup context and found no present-tense port/parity/drop-in framing. Independently re-derived: `CLAUDE_MD_FRAMING_TOTAL=0`, `STATE_MD_RETIRED_CORE_VALUE=0`, instrument positive-controlled against a planted wrapped fixture (returns 2 there, so the zero is real). `MEM02_FILES_STATUS` updated to `verified-fresh-session`. **Store half (unchanged by ruling):** `MEM02_STORE_STATUS=accepted-by-d15-evidence-standard` stays exactly as is. UAT test 2 is `skipped`, not passed — the maintainer cannot query engram directly, so the human judgment was never rendered. Routed to human verification below. |

**Score:** 5/5 requirements closed. 4/5 VERIFIED by evidence (BENCH-01, BENCH-02, BENCH-03,
MEM-01). MEM-02's file half is VERIFIED; its store half is CLOSED BY MAINTAINER ACCEPTANCE at the
D-15 evidence standard (ruling 2026-08-16) — a verified-by-inspection item the spine's out-of-git
location makes unautomatable, never observed from a fresh engram-tooled session, and carrying an
explicit and deliberately un-upgraded status token. See `## Acknowledged Gaps`.

### Deferred Items

None — no gap in this phase maps to work explicitly scheduled in a later milestone phase.

### Required Artifacts

| Artifact | Expected | Status | Details |
|---|---|---|---|
| `docs/BENCHMARKS.md` | Absolute-only, no comparison table | ✓ VERIFIED (unchanged) | Untouched since d3b4f37; 0 `headtohead` hits, instrument positive-controlled |
| `tools/bench/publishcheck/{main,main_test}.go` | Pure-Go publish-JSON verifier | ✓ VERIFIED (unchanged) | Untouched; its pure-Go nature is why the CI publish job needs no Node step |
| `.github/workflows/bench.yml` | `publish` job, no `headtohead`, `if-no-files-found: error` | ✓ VERIFIED | Untouched; `publish:` job at line 97, `if-no-files-found: error` at line 154, both exercised live by run 31973967130 |
| `06-MEMORY-SWEEP.md` | Full D-12/D-13/D-14 verdict record, MEM-01 completeness sections | ✓ VERIFIED | Gate passes; 17-row agent-facing framing table enumerates every occurrence with a verdict, including the `PROJECT.md:31,40,…,299` keep-historical range row |
| `06-LIVE-VERIFICATIONS.md` | BENCH-03 + MEM-02 (both halves) status tokens | ✓ VERIFIED (updated this pass) | `BENCH03_STATUS=closed-by-ci-run …/31973967130`; `MEM02_FILES_STATUS=verified-fresh-session` (upgraded on UAT test 1 + independent re-derivation); `MEM02_STORE_STATUS=accepted-by-d15-evidence-standard` (unchanged by ruling). Exactly one token each |
| `.claude/CLAUDE.md`, `.planning/PROJECT.md`, `.planning/STATE.md` | Present-tense / forward-looking framing swept | ✓ VERIFIED | `CLAUDE_MD_FRAMING_TOTAL=0`, `STATE_MD_RETIRED_CORE_VALUE=0`, both positive-controlled |
| `06-UAT.md` | Human UAT record | ✓ VERIFIED | status `complete`; 23 tests, 22 pass, 0 issues, 1 skipped-with-reason, 0 blocked |

### Key Link Verification

| From | To | Via | Status | Details |
|---|---|---|---|---|
| `.github/workflows/bench.yml` `publish` job | `tools/bench/runner -mode publish` | inline `go run` (bench.yml:135) | ✓ WIRED (**now live-proven, not just structural**) | Run 31973967130 executed it end-to-end and produced the 2-record absolute artifact. The prior pass could only prove this structurally |
| `publish` job | `$GITHUB_STEP_SUMMARY` | `cat publish-results.json` in a fenced block (bench.yml:139-147) | ✓ WIRED | Step concluded `success`; summary content is byte-identical to the artifact that was downloaded and read |
| Approved MEM-01 supersedes | Executed engram writes | `supersede_memory` calls | ✓ WIRED | Set-equality between the approved and executed short_id sets proven by the gate's own `diff` (empty), `SUPERSEDE_SET_EQUALITY=true` |
| Superseded originals | Correcting records | `superseded_by` stamp | ✓ WIRED (evidence at D-15 standard) | `get_memory("3ekc84hbqt")` returns the original intact with `superseded_by` stamped — history preserved, nothing deleted |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|---|---|---|---|---|
| `publish-results.json` (CI artifact) | `files_per_sec`, `bytes_per_sec`, `query_latency_median_ms`, `peak_rss_bytes`, `cold_start_ms` | live `go run ./tools/bench/runner -mode publish` over the pinned real-repo corpora on `namespace-profile-linux-amd64-4x8` | ✓ — weft-go 3357.77 files/s, 29802491 bytes/s, 10.596 ms p50, 57020416 B peak RSS, 6.498 ms cold start; cockroachdb-pebble 1244.30 / 19607524 / 25.467 / 186265600 / 6.746. All positive, no hardcoded or placeholder values | ✓ FLOWING |
| Job summary | same five metrics | `cat publish-results.json` into `$GITHUB_STEP_SUMMARY` | ✓ — byte-identical to the artifact | ✓ FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|---|---|---|---|
| Phase branch exists on origin | `git ls-remote --exit-code --heads origin gsd/v0.11.0-standalone-project-identity` | resolves to `9f763cae…`, exit 0 | ✓ PASS |
| bench.yml run succeeded on the phase branch | `gh run view 31973967130 --json conclusion,event,headBranch` | `success` / `workflow_dispatch` / phase branch | ✓ PASS |
| Publish job green, all others skipped | `gh run view 31973967130 --json jobs` | 1 success + 6 skipped | ✓ PASS |
| Artifact shape | `gh run download 31973967130 -n publish-results` then read JSON | 2 records, both `subject: go`, 5 positive metrics, no ratio | ✓ PASS |
| No Node toolchain in publish job | full step list + log scan for `npm\|npx\|pnpm\|yarn\|node_modules` | 0 hits; instrument positive-controlled (49 `codegraph` hits same log) | ✓ PASS |
| `.claude/CLAUDE.md` framing census | 06-05 Task 3 gate, verbatim | `CLAUDE_MD_FRAMING_TOTAL=0` (control fixture returns 2) | ✓ PASS |
| Engram spine recall cleanliness | `search_memory` / `get_memory` | performed by an agent during UAT, not by the maintainer | ? SKIP — routed to human verification |

### Probe / Acceptance Command Execution

| Command | Source | Result | Status |
|---|---|---|---|
| 06-06-PLAN.md Task 3 `<automated>` gate (verbatim) | `06-06-PLAN.md:370` | `SWEEP_RECORD_CLOSED=true`, exit 0 — full literal output in Method above | PASS |
| 06-05-PLAN.md Task 3 `<automated>` framing gate (verbatim) | `06-05-PLAN.md:376` | `CLAUDE_MD_FRAMING_TOTAL=0`, `STATE_MD_RETIRED_CORE_VALUE=0` | PASS |

### Requirements Coverage

| Requirement | Source Plan | Status | Evidence |
|---|---|---|---|
| BENCH-01 | 06-04 | ✓ SATISFIED | `docs/BENCHMARKS.md`, unchanged; CI run corroborates the numbers on different hardware |
| BENCH-02 | 06-01, 06-02, 06-03 | ✓ SATISFIED | Comparison runner removed; `CheckRegression` proven RED against reverted mutations |
| BENCH-03 | 06-02, 06-04 | ✓ SATISFIED (**newly closed**) | Live run 31973967130 |
| MEM-01 | 06-06 | ✓ SATISFIED | 4/4 supersedes executed and reconciled set-equal; nothing deleted |
| MEM-02 | 06-05, 06-06 | ✓ SATISFIED (store half by acceptance) | File half live-verified via UAT test 1; store half closed by maintainer ruling 2026-08-16 at the D-15 evidence standard — accepted, not demonstrated. See `## Acknowledged Gaps` |

No orphaned requirements — all five BENCH/MEM requirements are claimed by a phase-6 plan.

### Anti-Patterns Found

No `TODO`/`FIXME`/`XXX`/`HACK`/`PLACEHOLDER` debt markers in the files changed since d3b4f37
(`06-LIVE-VERIFICATIONS.md`, `06-UAT.md`, `06-VERIFICATION.md` — all planning artifacts). No stub
content and no grep-satisfying filler.

**One wording imprecision recorded rather than smoothed over (not a defect).** The prior pass's
BENCH-03 human-check, and `06-UAT.md` test 3's `expected` field, both say the job summary shows an
absolute per-corpus **table**. The `Publish results to job summary` step (`bench.yml:139-147`)
emits a fenced JSON block, not a rendered markdown table. The substance the requirement asks for is
published — absolute per-corpus figures for both corpora with no second implementation — and
neither BENCH-03's requirement text nor Phase 6's Success Criterion 3 uses the word "table". The
checklist phrasing was imprecise; the artifact is correct. Noted in `06-LIVE-VERIFICATIONS.md`.

## Acknowledged Gaps

### MEM-02 store half — accepted, not demonstrated

**Maintainer ruling, 2026-08-16, verbatim: "MEM-02 - accept it and move on."**

This closes the phase's last outstanding human-verification item by **acceptance at the D-15
evidence standard**, not by the fresh-session observation the checkpoint originally asked for. The
distinction is recorded rather than smoothed over, and the durable status token is deliberately
left unchanged:

- `MEM02_STORE_STATUS=accepted-by-d15-evidence-standard` in `06-LIVE-VERIFICATIONS.md` **stays
  exactly as is.** It was not upgraded to `verified-fresh-session` and must not be read as such.
- **What was never done:** no session that is both (a) genuinely fresh and (b) carrying
  `mcp__engram__*` tools has observed the spine's startup recall. Worktree-spawned executors carry
  no engram tools (engram is installed at user scope, not in this repo's `.mcp.json`) — the
  condition that halted 06-06's first execution attempt and that no phase artifact ever satisfied.
- **What was done, and what the acceptance rests on:** the D-15 re-query evidence — four
  corrections applied via `supersede_memory`, `get_memory("3ekc84hbqt")` returning the original
  intact with `superseded_by` stamped (history preserved, nothing deleted or overwritten), and a
  `search_memory` phrased in the superseded records' own vocabulary returning all four correcting
  records and none of the four originals. Re-run independently during UAT and clean both times.
- **UAT test 2 remains `skipped` in `06-UAT.md`**, with the agent-run probe attached as evidence.
  It was deliberately not converted to a `pass` at the time, because no human judgment had been
  rendered. This ruling *is* that judgment — rendered as acceptance of the substitute evidence,
  which is a weaker claim than observation and is labelled as such here.

**If this is ever revisited:** the cheapest discharge is a fresh session that runs the adversarial
`search_memory` probe as its first action, before any other context accumulates. Nothing about the
sweep needs redoing — only the observation.

### Human Verification — discharged

#### 1. MEM-02 store half — engram spine recall (ACCEPTED per ruling above)

**Test:** From a session carrying `mcp__engram__*` tools, query the engram spine for
`repo:github.com/seanb4t/codegraph-go` and confirm the four correcting records (`xj1stbrsw6`,
`gxwkk3necn`, `b9wjge7375`, `mw5z9s9bft`) surface while none of the four superseded originals
(`3ekc84hbqt`, `gw79qy2a9z`, `agggksad53`, `7f0pq2wepv`) do.

**Expected:** No recalled memory asserts a binding parity floor, a functionality baseline, or a
migration read contract in the present tense. The superseded originals remain retrievable by id
with `superseded_by` stamped — history preserved, nothing deleted.

**Why human:** MEM-02's store half is verified by inspection, not by a test — the spine lives
outside git and no CI gate can hold it (ROADMAP Phase 6 Notes). UAT test 2 is recorded `skipped`,
NOT passed: the maintainer cannot query engram directly, so the human judgment this checkpoint asks
for was never rendered. Per explicit maintainer ruling, `MEM02_STORE_STATUS` stays at
`accepted-by-d15-evidence-standard` and is not upgraded. The agent-run adversarial probe recorded
in `06-UAT.md` (a `search_memory` phrased in the superseded records' own vocabulary returning all
four correcting records and none of the four originals, plus `get_memory("3ekc84hbqt")` showing the
original intact with `superseded_by` stamped) re-confirms the same D-15 re-query evidence the phase
already accepted. It is corroboration at the existing standard, never an upgrade.

## Gaps Summary

**No gaps.** All five requirements are either VERIFIED or, in MEM-02's case, half-verified with the
remaining half carrying an honest, deliberately un-upgraded status token rather than a silent
omission.

Two items that the prior pass routed to human verification are now closed:

1. **BENCH-03 — closed by a real CI run.** The prior report's claim that the branch was unpushed
   and no run existed was correct when written and is now stale; it is corrected above. Run
   31973967130 succeeded on the phase branch with the publish job green, all six other jobs
   skipped, a 2-record absolute-only artifact independently downloaded and read, and no Node
   toolchain anywhere in the job.
2. **MEM-02 file half — closed by the UAT session's live read.** The prior pass could not spawn an
   independently fresh session; the UAT session *was* one (post-`/clear`), assembled its own
   startup context, and observed no present-tense port/parity/drop-in framing. Independently
   corroborated here by re-running the phase's own census gate with a positive-controlled
   instrument.

The third and last item — MEM-02's store half — is closed by **maintainer acceptance** rather than
by observation: ruling of 2026-08-16, "MEM-02 - accept it and move on." The phase's own design
(ROADMAP Phase 6 Notes: "MEM-01/02 are verified by inspection, not by a test") places it outside
any automatable gate, and no session in this phase was ever both fresh and engram-tooled. The
acceptance, what it rests on, and what was never demonstrated are recorded in full under
`## Acknowledged Gaps` above; `MEM02_STORE_STATUS` is deliberately left at
`accepted-by-d15-evidence-standard` so the artifact never reads stronger than the evidence.

Phase 6 status is therefore `passed` — with that one requirement closed by an explicit, recorded
judgment call rather than by a demonstration.

---

*Verified: 2026-08-16 (re-verification, third pass — post-UAT)*
*Verifier: Claude (gsd-verifier)*
