---
phase: 06-benchmark-de-coupling-memory-sweep
verified: 2026-08-16T20:45:00Z
status: gaps_found
score: 2/5 requirements fully PASS, 3/5 PARTIAL (all three partials are honestly disclosed by the
  phase's own artifacts; one of the three additionally carries a previously-undocumented
  completeness gap found independently by this verification)
behavior_unverified: 0
overrides_applied: 0
gaps:
  - truth: "MEM-01: every superseded record is issued through the plan's own designed acceptance
      record, with the approved-vs-executed supersede sets reconciled by explicit diff and the
      population's completeness stated arithmetically"
    status: partial
    reason: >
      06-06-PLAN.md Task 3 mandates two named sections in 06-MEMORY-SWEEP.md
      (`## Supersedes executed (MEM-01)` and `## Completeness and accepted gaps`) plus an explicit
      approved==executed short_id set-equality reconciliation. None of the three exist in the
      committed 06-MEMORY-SWEEP.md — confirmed by heading grep, zero hits for both. The underlying
      work (4 supersedes, verified two ways: get_memory shows superseded_by stamped and intact;
      search_memory returns only the correcting records) is real and well-evidenced in a
      differently-shaped "### Task 3 (apply the approved corrections) — COMPLETE" subsection, but
      the phase's own defined acceptance artifact for MEM-01 is incomplete.
    artifacts:
      - path: ".planning/phases/06-benchmark-de-coupling-memory-sweep/06-MEMORY-SWEEP.md"
        issue: "Missing '## Supersedes executed (MEM-01)' and '## Completeness and accepted gaps'
          headings; no printed approved/executed set-equality diff"
    missing:
      - "A '## Supersedes executed (MEM-01)' section, one row per executed supersede (short_id
        first column, operation name, returned identifier, response shape, corrected content)"
      - "A '## Completeness and accepted gaps' section stating per-scope totals and the
        supersede-count + leave-historical-count = total arithmetic"
      - "The approved-vs-executed short_id sets extracted and diffed with the count printed
        (APPROVED_SUPERSEDE_IDS / EXECUTED_SUPERSEDE_IDS / SUPERSEDE_SET_EQUALITY)"
  - truth: "MEM-02 store half: the standard actually met (a performed fresh-session recall, or the
      accepted enumeration-plus-returned-identifier evidence) is recorded as an explicit status
      token, never left to a reader's inference"
    status: partial
    reason: >
      06-06-PLAN.md Task 3 mandates a '## MEM-02 — store half' section in
      06-LIVE-VERIFICATIONS.md carrying exactly one of
      `MEM02_STORE_STATUS=verified-fresh-session` or
      `MEM02_STORE_STATUS=accepted-by-d15-evidence-standard`. Neither the heading nor the token
      exists anywhere in the committed 06-LIVE-VERIFICATIONS.md — confirmed by direct grep,
      zero hits. This is the one item in the phase that is not merely partial-and-disclosed but
      silently absent: the plan's own text states "what is not legitimate is claiming MEM-02
      demonstrated when the standard actually met was D-15's" — and right now no standard is
      recorded at all for the store half, so a reader cannot tell whether MEM-02's store half was
      accepted-by-design or simply skipped.
    artifacts:
      - path: ".planning/phases/06-benchmark-de-coupling-memory-sweep/06-LIVE-VERIFICATIONS.md"
        issue: "No '## MEM-02 — store half' section and no MEM02_STORE_STATUS token of either
          value"
    missing:
      - "A '## MEM-02 — store half' section in 06-LIVE-VERIFICATIONS.md carrying
        `MEM02_STORE_STATUS=accepted-by-d15-evidence-standard` (the applicable value, since a
        fresh-session recall was not performed) per the plan's own two-token contract"
deferred: []
human_verification:
  - test: "Start a genuinely fresh session in this repository and read the memory recall it
      surfaces at startup (per .claude/CLAUDE.md plus whatever the harness injects from
      .planning/), plus the engram spine recall for repo:github.com/seanb4t/codegraph-go"
    expected: "Nothing recalled describes codegraph-go as a port, a parity project or a drop-in
      replacement in the present tense, and nothing states a future obligation to another
      implementation's behaviour"
    why_human: "MEM-02's own success criterion is a live session's recall, not a static check. The
      phase's own plans (06-05, 06-06) both declined to perform this and recorded the fact
      explicitly rather than claiming it on silence — this verifier inherits the same limitation
      and cannot spawn an independent fresh session to observe the recall itself."
  - test: "Push branch gsd/v0.11.0-standalone-project-identity (or land it) and dispatch
      .github/workflows/bench.yml with job=publish; inspect the resulting run's job summary and
      the uploaded publish-results artifact"
    expected: "The job summary shows an absolute per-corpus table (files/s, bytes/s, query
      latency, peak RSS, cold start) for weft-go and cockroachdb-pebble; the publish-results
      artifact uploads without if-no-files-found tripping; no step in the job installs or invokes
      any implementation other than the freshly-built Go binary"
    why_human: "BENCH-03 requires an actual bench.yml run. The branch is confirmed unpushed
      (git ls-remote exits non-zero); no CI run exists for it. This is a maintainer action the
      phase's own maintainer explicitly declined to perform in this session (checkpoint decision:
      pending-no-ci-run, not dispatch-now)."
---

# Phase 6: Benchmark De-coupling & Memory Sweep Verification Report

**Phase Goal:** The project publishes its own absolute performance numbers with no second
implementation in the picture, and an agent that starts a session afterward recalls codegraph-go
as a project rather than as a port.

**Verified:** 2026-08-16
**Status:** gaps_found
**Re-verification:** No — initial verification

## Method

Goal-backward verification against the codebase, not the SUMMARY narratives. Every claim below was
independently re-checked with direct file reads, `git log`/`git show`, and fresh `rg -U`
positive-controlled census invocations run by this verifier (not merely re-reading the phase's own
census artifacts). Per the task brief, three items are treated as the deliberately-correct end
state rather than gaps: BENCH-03 recorded pending (no CI run exists for an unpushed branch),
MEM-02's file-half fresh-session check recorded pending, and the engram store's contents being
unverifiable directly (outside git, no tool access here). One additional item — the missing
MEM-02 store-half status token and the missing MEM-01 completeness sections in 06-06's own output
— was found independently during this verification and is **not** one of the pre-cleared items; it
is reported as a genuine gap below.

## Goal Achievement

### Observable Truths (per-requirement)

| # | Requirement | Truth | Status | Evidence |
|---|---|---|---|---|
| 1 | BENCH-01 | `docs/BENCHMARKS.md` publishes absolute throughput/query-latency/peak-RSS figures with methodology and the regression-gate description, no head-to-head table | ✓ VERIFIED | Read the full file. §1 Methodology (median-of-5, OS-level `getrusage` peak RSS, pinned-SHA corpora), §2 two-row absolute table generated mechanically by `publishcheck -emit-rows` (byte-matched against `06-PUBLISH-RESULTS.json`), §3 regression gate (PERF-02/INDX-06). No comparison table, no ratio/multiplier, no second implementation named anywhere in the file (independent `rg` census, see below). |
| 2 | BENCH-02 | `tools/bench` contains no comparison runner; `CheckRegression` against the committed `baseline.json` still fires on a real regression, demonstrated RED against a confirmed-applied, byte-cleanly-reverted mutation | ✓ VERIFIED | `ts-binary` occurs exactly once repo-wide, inside the intentional removal-proof test (`tools/bench/runner/main_test.go:503`); `headtohead` occurs zero times in `.github/workflows/bench.yml` or `tools/bench/`; the six `headtohead-*.json` captures are untracked (`git ls-files` → 0), recoverable from history (`git log -1` resolves at `2572a67`). `06-MUTATION-LOG.md` reviewed in full: two independent tolerance-constant mutations, each with a pre-mutation `git diff --quiet` gate, a pasted non-empty `git diff --stat` confirming the mutation applied, a pasted RED transcript naming the correct subtest in **both** `TestCheckRegression` (pre-existing table) and the new `TestCheckRegressionAgainstCommittedBaseline` (loads the real committed `baseline.json`), a byte-clean revert (`git diff --stat` empty), and a green re-run. No category-error RED in either family (the failure mode that would prove nothing). |
| 3 | BENCH-03 | `bench.yml` runs and publishes the absolute numbers without invoking another implementation | ⚠️ PARTIAL — deliberately deferred | The `publish` job exists, is structurally verified against all four D-06 properties by `TestBenchPublishJobShape` (runner pinning, no-Taskfile exception, non-blocking publish-not-gate, artifact upload + `GITHUB_STEP_SUMMARY`), and `rebless` is proven untouched by a parsed-subtree fixture plus a SHA-256 run-body digest. But **no `bench.yml` run has actually executed the `publish` job** — the branch is confirmed unpushed (`git ls-remote --exit-code --heads origin gsd/v0.11.0-standalone-project-identity` → exit 2, re-verified independently by this report), `06-LIVE-VERIFICATIONS.md` carries exactly one `BENCH03_STATUS=pending-no-ci-run` token, no fabricated run URL exists, and the maintainer explicitly declined to push/dispatch at a blocking checkpoint. This is the correct honest state per the task brief, not a defect — but the roadmap's literal success criterion ("a `bench.yml` run publishes...") is not yet true. |
| 4 | MEM-01 | Every engram spine record whose durable content asserts retired framing is superseded — none overwritten, none deleted | ⚠️ PARTIAL | Substance strongly evidenced: 169 records enumerated to exhaustion across 5 pages (`next_cursor` empty on the last); classification criterion applied and put to a blocking maintainer checkpoint; 4 supersedes approved and applied (`3ekc84hbqt→xj1stbrsw6`, `gw79qy2a9z→gxwkk3necn`, `agggksad53→b9wjge7375`, `7f0pq2wepv→mw5z9s9bft`); verified two ways per 06-06-SUMMARY.md (`get_memory` shows the original intact with `superseded_by` stamped; `search_memory` on the superseded records' own terms returns only the correcting records). The engram store itself lives outside git and is not independently checkable by this verifier — per the task brief, its unverifiability is not counted as a gap. **However**, 06-06-PLAN.md's own Task 3 acceptance criteria mandate a `## Supersedes executed (MEM-01)` section and a `## Completeness and accepted gaps` section in `06-MEMORY-SWEEP.md`, with an explicit approved-vs-executed short_id set-equality reconciliation. Neither section exists (confirmed by heading grep — zero hits for both). This is a genuine, previously-undocumented gap against the phase's own designed acceptance record, independent of the pre-cleared items. |
| 5 | MEM-02 | A session started after the sweep recalls no memory describing codegraph-go as a port or parity project in the present tense | ⚠️ PARTIAL | **File half:** independently re-verified clean. `.claude/CLAUDE.md`'s "ground-up Go rewrite" opener is gone; its Core Value line matches `.planning/PROJECT.md`'s byte-for-byte; the Compatibility/Licensing lines are re-synced and correctly retired; `.planning/STATE.md`'s Core value line now matches `.planning/PROJECT.md` verbatim (both read the clean pre-indexed-code-knowledge-graph statement). Live fresh-session recall check honestly recorded as `MEM02_FILES_STATUS=pending-fresh-session-not-performed` — not claimed on silence; this is the correct disclosed state per the task brief. **Store half:** the engram spine was swept (see MEM-01 above), but 06-06-PLAN.md's own Task 3 mandates a `## MEM-02 — store half` section in `06-LIVE-VERIFICATIONS.md` carrying exactly one of `MEM02_STORE_STATUS=verified-fresh-session` / `MEM02_STORE_STATUS=accepted-by-d15-evidence-standard`. **Neither the heading nor the token exists anywhere in the committed file** (confirmed by direct grep — zero hits). Unlike the file-half token, which is present and honest, the store-half token is simply absent — the one item in this phase that reads as silently skipped rather than disclosed. |

**Score:** 2/5 requirements fully VERIFIED (BENCH-01, BENCH-02); 3/5 PARTIAL (BENCH-03, MEM-01,
MEM-02) — all three partial for reasons rooted in the phase's own design (deferred CI dispatch,
declined fresh-session check, unverifiable-outside-git store), but MEM-01 and MEM-02 additionally
carry one real, independently-found completeness gap each (the missing 06-06 Task 3 record
sections and the missing MEM02_STORE_STATUS token) that the phase's own artifacts do not disclose.

### Independent Re-verification (spot checks run by this verifier, not copied from phase artifacts)

- **Positive control proven live** before trusting any zero: planted a wrapped-phrase fixture
  (`// the retired harness invoked a second (TS\n// codegraph@1.3.1) binary here...`), multiline
  `rg -U` returned 3, line-based `rg` returned 1 (3 > 1 > 0) — confirms the multiline instrument
  catches what line-based misses, matching the same shape the phase's own census documents.
- **Independent census over the bench surface** (`docs/BENCHMARKS.md`, `tools/bench/`,
  `internal/bench/`, `internal/corpora/manifest.go`, `corpora/manifest.json`,
  `.github/workflows/bench.yml`, `internal/upgrade/taskfile_shape_test.go`), excluding only the two
  pre-authorised files, bounded pattern set (`head-to-head`, `headtohead`, `colbymchenry`,
  `codegraph-ts`, `comparator`, `1.3.1`, `ts-binary`): **0 hits.**
- `internal/migrate` does not exist on disk; `rg -n "migrate" internal/cli` returns zero hits — the
  CLI registers no `migrate` command. Confirms CODE-03's Phase-5 removal, which `.planning/STATE.md`
  and `.planning/PROJECT.md` now correctly describe.
- `git show 61f7975` reviewed directly: the exact two stale lines the task brief named
  (STATE.md:113, PROJECT.md:72, both previously claiming `codegraph migrate` "keeps working
  end-to-end" / "stays as a capability") are corrected to state the maintainer-ruled removal, with
  genuinely historical entries elsewhere in both files left untouched.
- `.planning/phases/.../06-PUBLISH-RESULTS.json` byte-cross-checked against the data rows rendered
  in `docs/BENCHMARKS.md` §2 — both rows (`cockroachdb-pebble`, `weft-go`) match exactly on every
  published field.
- Git history reviewed for the full phase: `70848f7` (tracking update) back through `899565a`,
  `b4dd057`, `a69c4dd`, `f2460aa`, `1a3ab37` (merge), `61f7975` (the maintainer-approved
  out-of-plan fix), `6e9c482` (the halt), plus the earlier wave commits — the narrative in
  `06-MEMORY-SWEEP.md`'s "HALTED" / "RESUMED AND COMPLETED" sections matches the actual commit
  sequence. Working tree is clean at HEAD (`70848f7`).

### Required Artifacts

| Artifact | Expected | Status | Details |
|---|---|---|---|
| `docs/BENCHMARKS.md` | Absolute-only rewrite | ✓ VERIFIED | Full rewrite confirmed; no comparison content |
| `tools/bench/publishcheck/{main,main_test}.go` | Pure-Go publish-JSON verifier | ✓ VERIFIED | Exists, `TestPublishCheck` is the single top-level symbol with 7 subtests |
| `tools/bench/runner/main.go` | `runPublish`, no `runHeadToHead`/`resolveTSBinary` | ✓ VERIFIED | Confirmed via census + build |
| `.github/workflows/bench.yml` | `publish` job, no `headtohead` | ✓ VERIFIED | `publish:` present at line 97; zero `headtohead` hits |
| `internal/upgrade/bench_workflow_shape_test.go` | D-06 structural verifier | ✓ VERIFIED (existence + plan-documented RED/GREEN cycle; not independently re-run by this verifier) | File exists; SUMMARY documents a genuine pre-existence RED and a post-GREEN discriminating perturbation |
| `internal/bench/baseline_gate_test.go` | Drives gate with committed `baseline.json` | ✓ VERIFIED (existence + plan-documented mutation rehearsal reviewed in full) | File exists; `06-MUTATION-LOG.md` reviewed line-by-line, both oracles reddened correctly |
| `06-MEMORY-SWEEP.md` | Full D-13/D-14 verdict record, MEM-01 completeness sections | ⚠️ INCOMPLETE | Present but missing two plan-mandated sections (see gap above) |
| `06-LIVE-VERIFICATIONS.md` | BENCH-03 + MEM-02 (both halves) status tokens | ⚠️ INCOMPLETE | BENCH-03 and MEM-02 file-half tokens present and correct; MEM-02 store-half section/token entirely absent |
| `.claude/CLAUDE.md`, `.planning/PROJECT.md`, `.planning/STATE.md` | Present-tense/forward-looking framing swept | ✓ VERIFIED | Independently re-read; matches the sweep record |
| `.planning/STATE.md`, `.planning/PROJECT.md` (61f7975) | Stale migrate claims corrected | ✓ VERIFIED | `git show` reviewed directly |

### Key Link Verification

| From | To | Via | Status | Details |
|---|---|---|---|---|
| `tools/bench/runner/main.go` (`runPublish`) | `internal/bench.Metrics` JSON on stdout | `measureSubject("go", ...)` | ✓ WIRED | Confirmed by `06-PUBLISH-RESULTS.json`'s shape and `docs/BENCHMARKS.md`'s rendered rows matching it |
| `realcorpus.Corpora()` | `tools/bench/runner/main.go` | consumption at the publish loop | ✓ WIRED | Two-entry corpus (`weft-go`, `cockroachdb-pebble`) confirmed in both `manifest.go` and the emitted results |
| `.github/workflows/bench.yml` `publish` job | `tools/bench/runner -mode publish` | inline `go run` invocation | ✓ WIRED (structurally proven; not live-run) | Confirmed via `bench_workflow_shape_test.go`'s assertions, reviewed in the SUMMARY; no independent CI run exists (BENCH-03 gap above) |
| Approved MEM-01 supersedes | Executed engram writes | `supersede_memory` calls | ⚠️ WIRED but under-documented | Work is done (per 06-06-SUMMARY.md's two independent verifications), but the plan's own reconciliation record connecting "approved" to "executed" is missing from the committed artifact |

### Anti-Patterns Found

No `TODO`/`FIXME`/`XXX`/`HACK`/`PLACEHOLDER` debt markers found in any file this phase modified
(spot-checked `docs/BENCHMARKS.md`, `tools/bench/runner/main.go`, `tools/bench/publishcheck/*.go`,
`internal/bench/*.go`, `.github/workflows/bench.yml`). No stub returns, no hardcoded-empty data
flowing to a rendered surface. The phase's own census discipline (positive-controlled multiline
`rg`, bounded patterns, per-exclusion citations) is unusually rigorous and was independently spot
re-verified above rather than merely trusted.

### Requirements Coverage

| Requirement | Source Plan | Status | Evidence |
|---|---|---|---|
| BENCH-01 | 06-04 | ✓ SATISFIED | `docs/BENCHMARKS.md` |
| BENCH-02 | 06-01, 06-02 (partial), 06-03 | ✓ SATISFIED | Runner removal + mutation rehearsal |
| BENCH-03 | 06-02, 06-04 | ⚠️ NOT YET SATISFIED (by design, disclosed) | No CI run exists |
| MEM-01 | 06-06 | ⚠️ PARTIALLY SATISFIED | Substance done; own acceptance record incomplete |
| MEM-02 | 06-05, 06-06 | ⚠️ PARTIALLY SATISFIED | File half clean, live check pending; store half token missing |

No orphaned requirements — all five BENCH/MEM requirements are claimed by a phase-6 plan.

## Gaps Summary

Two categories of incompleteness, and they should be weighed differently:

1. **Deliberately deferred, honestly disclosed (not phase defects):** BENCH-03's CI run, and MEM-02's
   file-half live fresh-session recall. Both are recorded with explicit status tokens
   (`BENCH03_STATUS=pending-no-ci-run`, `MEM02_FILES_STATUS=pending-fresh-session-not-performed`),
   both require a maintainer/session action outside this phase's own automatable scope, and both were
   explicit checkpoint decisions the maintainer made deliberately. These are not gaps in execution —
   they are the phase correctly declining to fabricate evidence it does not have.

2. **A genuine, previously-undocumented completeness gap, found independently by this
   verification:** 06-06's Task 3 (the MEM-01 supersede-execution / MEM-02 store-half closure task)
   did not produce two of its own plan-mandated artifacts — the `## Supersedes executed (MEM-01)` /
   `## Completeness and accepted gaps` sections in `06-MEMORY-SWEEP.md`, and the
   `## MEM-02 — store half` section with its `MEM02_STORE_STATUS` token in
   `06-LIVE-VERIFICATIONS.md`. The underlying work (4 supersedes, verified two ways) is real and
   substantively satisfies MEM-01's requirement text. But unlike every other partial item in this
   phase, this one carries no disclosure token at all — nothing in the committed artifacts states
   that these sections are missing. A closure plan should add them; this is bookkeeping, not
   re-doing the memory-store work itself.

Everything else independently checked in this verification — the runner/CI/document rewrite, the
mutation-rehearsal proof, the agent-facing framing sweep, and the STATE.md/PROJECT.md stale-claim
fix — held up under direct re-inspection and matches what the SUMMARY files claim.

---

*Verified: 2026-08-16*
*Verifier: Claude (gsd-verifier)*
