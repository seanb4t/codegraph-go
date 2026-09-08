---
phase: 03-browse-inspect-navigation
plan: 03
subsystem: testing
tags: [mcp, wire-protocol, flake, go-sdk, jsonrpc2, testing, wireoracle]

# Dependency graph
requires:
  - phase: 01-engine-seam-wire-protocol-secure-transport
    provides: the wire-oracle capture/normalize/compare pipeline and FIX-01's pendingWriter fix, both re-verified (not re-derived) as part of this investigation
provides:
  - Root cause of the toolslist-repeat ordering flake, recorded with a live Linux reproduction and a direct SDK-source citation
  - CanonicalizeResponseOrder (test/wireoracle/normalize.go) — response-order canonicalization applied before the frozen-transcript comparison
  - Corrected scenario comments describing what github.com/modelcontextprotocol/go-sdk@v1.7.0 actually guarantees for concurrent dispatch
  - The todo closed with a verifiable resolution record
affects: [any future wire-oracle scenario with more than one pipelined non-initialize call; internal/mcp maintainers reasoning about response ordering]

# Actuals (#2632)
actuals:
  tokens: 6234
  tasks: 3
  commits: 9

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Response-order canonicalization as a distinct post-normalization, pre-comparison step (not folded into the Rules/ledger substitution machinery, which is structurally a different operation — line reordering vs. field substitution)"
    - "RED-first staged as an identity-function stub commit, then a real-implementation GREEN commit, for a non-TDD-typed plan whose task explicitly demanded the discipline"

key-files:
  created:
    - .planning/phases/03-browse-inspect-navigation/03-03-EVIDENCE.md
    - test/wireoracle/capture_test.go
  modified:
    - test/wireoracle/capture.go
    - test/wireoracle/oracle_test.go
    - test/wireoracle/normalize.go
    - test/wireoracle/normalize_test.go
    - test/wireoracle/scenarios.go
    - .planning/todos/completed/2026-08-07-wire-oracle-toolslist-repeat-response-ordering-flake.md (renamed from .planning/todos/pending/, via git mv)

key-decisions:
  - "VERDICT: SERVER-EMITTED-OUT-OF-ORDER, established from a live Linux reproduction (4/60 failures under contention, byte-identical symptom to the CI report) plus a direct read of modelcontextprotocol/go-sdk@v1.7.0's dispatch source — not inferred from timing alone."
  - "Maintainer selected R2 (canonicalize response order by request id before comparison) at the Task 2 blocking-human checkpoint, over R1 (impose in-order emission in production dispatch) and NR (leave open). Rationale: the SDK is not silent about concurrent dispatch, so R1 would knowingly override a documented upstream design decision to preserve a test artifact; NR would leave an intermittently-red required PR leg teaching that a blocking gate means re-run CI."
  - "CanonicalizeResponseOrder is deliberately NOT added to the Rules/NormalizeWithLedger ledger — it reorders whole lines and substitutes nothing, a structurally different operation from the three existing named-field placeholder rules, and folding it in would have broken TestRuleTestCoverageScalesWithRules's 1:1 rule-to-test-case invariant for no benefit."

requirements-completed: []
# TODO-MCP-01 (this plan's own requirements: field) has no entry in
# REQUIREMENTS.md — confirmed via `gsd-tools query requirements.mark-complete
# TODO-MCP-01`, which returned not_found and made no write. Expected: this
# plan is an unrelated todo-fold riding along the phase, per 03-CONTEXT.md's
# own framing ("must not gate this phase's five browse success criteria"),
# not a tracked roadmap requirement.

coverage:
  - id: D1
    description: "Root cause of the toolslist-repeat ordering flake established from captured/reproduced evidence and recorded in 03-03-EVIDENCE.md before any behavior change"
    requirement: "TODO-MCP-01"
    verification:
      - kind: unit
        ref: "test/wireoracle/capture_test.go#TestCaptureArrivalLedgerPreservesWireOrder"
        status: pass
      - kind: other
        ref: "go test ./test/wireoracle/ -run '^TestCaptureArrivalLedgerPreservesWireOrder$' -count=1 -v; grep VERDICT: 03-03-EVIDENCE.md"
        status: pass
    human_judgment: false
  - id: D2
    description: "R2 resolution (response-order canonicalization) implemented RED-first and proven GREEN"
    requirement: "TODO-MCP-01"
    verification:
      - kind: unit
        ref: "test/wireoracle/normalize_test.go#TestToolsListRepeatOrderingResolution"
        status: pass
    human_judgment: false
  - id: D3
    description: "Oracle's content-discrimination power proven unchanged after canonicalization — a planted content mutation is still caught"
    requirement: "TODO-MCP-01"
    verification:
      - kind: unit
        ref: "test/wireoracle/normalize_test.go#TestFrozenTranscriptComparisonDetectsContentMutation"
        status: pass
      - kind: manual_procedural
        ref: "manual plant against testdata/wireoracle/transcripts/toolslist-repeat.golden: FAIL observed, reverted, PASS observed (recorded below)"
        status: pass
    human_judgment: false
  - id: D4
    description: "Both false 'handled synchronously in request order' claim sites corrected in scenarios.go with the property actually enforced"
    requirement: "TODO-MCP-01"
    verification:
      - kind: other
        ref: "rg -o 'handled synchronously in' test/wireoracle/scenarios.go | wc -l (3 before, 1 after)"
        status: pass
    human_judgment: false
  - id: D5
    description: "Todo moved from pending/ to completed/ via git mv with a verifiable resolution record"
    requirement: "TODO-MCP-01"
    verification:
      - kind: other
        ref: "ls .planning/todos/completed/ | rg toolslist-repeat -> 1; ls .planning/todos/pending/ | rg toolslist-repeat -> 0"
        status: pass
    human_judgment: false
  - id: D6
    description: "Full-suite backstop green under the same contention that originally produced the failure"
    requirement: "TODO-MCP-01"
    verification:
      - kind: integration
        ref: "task test:wireoracle (fresh), task test:unit (fresh, -count=1), and a 60-attempt Linux-container re-run of the exact contention method that failed 4/60 before the fix"
        status: pass
    human_judgment: false

duration: ~90min (active work; excludes the maintainer's Task 2 decision wait, which is not execution time)
completed: 2026-08-28
status: complete
---

# Phase 3 Plan 03: Wire-oracle toolslist-repeat ordering flake — root-caused and resolved (R2) Summary

**Root-caused a real, reproducible SDK dispatch race (not a harness bug) by reading `modelcontextprotocol/go-sdk@v1.7.0` source directly, then resolved it via response-order canonicalization (R2) rather than a production dispatch change — proven RED-first, proven not to weaken content-discrimination, and re-verified 60/60 under the exact Linux contention that previously failed 4/60.**

## Performance

- **Duration:** ~90 min active work (excludes the Task 2 checkpoint wait for the maintainer's decision)
- **Tasks:** 3 (Task 1: reproduce and root-cause; Task 2: checkpoint:decision, answered R2; Task 3: implement and prove)
- **Files modified:** 7 (2 created, 5 modified, 1 renamed)
- **Commits:** 9 (5 code/docs commits, a self-caught staging fix, and 3 SUMMARY/metadata commits)

## Accomplishments

- **Reproduced the flake live** on Linux (Docker `golang:1.26.5`, `--cpus=4`, `GOMAXPROCS=4`): 4/60 attempts (~6.7%) failed under contention, with the identical symptom to the original CI report (line 2 of the normalized transcript held the id-3 `tools/list` response where id-2 belonged).
- **Established root cause from source, not inference:** `modelcontextprotocol/go-sdk@v1.7.0`'s `ServerSession.handle` (`mcp/server.go:1908-1914`) calls `jsonrpc2.Async(ctx)` unconditionally for every call except `initialize`; `internal/jsonrpc2/conn.go`'s `handleAsync` (652-687) dequeues requests sequentially but only blocks until `Async()` fires or the handler completes — so two consecutive `tools/list` calls run in independently scheduled goroutines with no ordering guarantee. VERDICT: SERVER-EMITTED-OUT-OF-ORDER, recorded in `03-03-EVIDENCE.md`.
- **Landed a durable, named arrival-ledger test** (`TestCaptureArrivalLedgerPreservesWireOrder`) proving the capture layer itself never reorders anything, ruling out the harness as an alternative explanation. Instrumented `Capture()` to record a timestamped `ArrivalLedger` unconditionally, and wired it to dump via `t.Logf` whenever `TestFrozenTranscriptsMatch` is about to fail.
- **Maintainer selected R2** at the Task 2 `checkpoint:decision` (`gate="blocking-human"`, never auto-approved): canonicalize response order by request id before comparison, over R1 (impose in-order emission in production dispatch — rejected, the SDK is not silent) and NR (leave open — rejected, an intermittently-red required PR leg teaches "re-run CI").
- **Implemented `CanonicalizeResponseOrder`** (`test/wireoracle/normalize.go`), RED-first: a stub identity function landed first with two failing tests (exit code 1, failure text recorded below), then the real stable-sort-by-request-id implementation landed and both tests went GREEN. Wired into `TestFrozenTranscriptsMatch` on both the captured and frozen side.
- **Proved the oracle's discrimination power is unchanged:** `TestFrozenTranscriptComparisonDetectsContentMutation` (automated, paired positive+negative case) plus a manual live plant against the real gate — both catch a one-character content mutation even in an out-of-order (and therefore canonicalized) transcript.
- **Corrected both false-claim comment sites** in `scenarios.go` (`:659-666` and `:1100-1101`) that asserted the old `mark3labs/mcp-go` transport's synchronous-ordering behavior, never updated after Phase 2's SDK-01 migration. `rg -o 'handled synchronously in' scenarios.go` count: 3 → 1 (only the correctly-scoped mark3labs `tools/call` worker-pool note at `:461` remains).
- **Closed the todo** with `git mv` (recorded as a rename) plus a resolution record in its frontmatter and body.
- **Re-verified the fix under the exact same contention method that originally reproduced the failure:** 60/60 pass (vs. 56/60 before the fix).

## Task Commits

Each task was committed atomically (2-4 commits per task where RED/GREEN discipline applied):

1. **Task 1: Reproduce under contention and separate server emission from harness consumption** — `7b4b4f52` (test)
2. **Task 3, RED:** `c22660c` (test) — `CanonicalizeResponseOrder` identity stub + two failing tests
3. **Task 3, GREEN:** `7197083` (feat) — real canonicalization implementation, wired into the oracle
4. **Task 3, comment correction:** `14fd221` (docs) — both false-claim sites corrected in `scenarios.go`
5. **Task 3, todo closure:** `e8138c0` (docs) — todo moved to `completed/` via `git mv` (this commit landed the rename only; see the staging-bug note below)
6. **Follow-up fix:** `7ad65cb` (docs) — lands the resolution frontmatter/section content that `e8138c0`'s `git add` silently failed to stage (see Issues Encountered)

**Plan metadata:** `eaf241e` (SUMMARY), `55f4a8d` (self-check), `baea65e0` (STATE/ROADMAP)

_Note: Task 2 was a `checkpoint:decision` answered by the maintainer directly — no separate commit; its answer (R2) drove Task 3._

## Files Created/Modified

- `.planning/phases/03-browse-inspect-navigation/03-03-EVIDENCE.md` — Task 1's full verdict, reproduction record, and SDK citation
- `test/wireoracle/capture_test.go` — `TestCaptureArrivalLedgerPreservesWireOrder`
- `test/wireoracle/capture.go` — `ArrivalLine`/`scanArrivalLines` + `Transcript.ArrivalLedger`, populated unconditionally by `Capture()`
- `test/wireoracle/oracle_test.go` — dumps the arrival ledger on impending failure; wires `CanonicalizeResponseOrder` into `TestFrozenTranscriptsMatch` on both sides before comparison
- `test/wireoracle/normalize.go` — `CanonicalizeResponseOrder`, the R2 resolution
- `test/wireoracle/normalize_test.go` — `TestToolsListRepeatOrderingResolution`, `TestFrozenTranscriptComparisonDetectsContentMutation`
- `test/wireoracle/scenarios.go` — both false-claim comment sites corrected
- `.planning/todos/completed/2026-08-07-wire-oracle-toolslist-repeat-response-ordering-flake.md` — renamed from `pending/`, resolution frontmatter + section added

## Decisions Made

See `key-decisions` in frontmatter. In full: the maintainer's stated rationale for R2 (verbatim, from the Task 2 answer):

> The oracle was freezing response arrival order, a property the MCP SDK explicitly declines to provide — `mcp/server.go:1911-1914` calls `jsonrpc2.Async(ctx)` for every call except `initialize`, citing `modelcontextprotocol/go-sdk#26`, and `internal/jsonrpc2/conn.go`'s `handleAsync` dequeues sequentially without awaiting completion. Freezing that property does not catch regressions; it measures goroutine scheduling, at a measured ~6.7% failure rate. Canonicalizing by request id narrows the oracle to the property that actually has a contract — response CONTENT, still frozen byte-exactly — rather than abandoning strictness. R1 was rejected specifically because the SDK is NOT silent: imposing in-order emission would knowingly override a documented upstream design decision and surrender concurrent tool-call dispatch in production to preserve a test artifact. NR was rejected because an intermittently-red REQUIRED PR leg teaches that a blocking gate means re-run CI, which erodes every other gate on the branch.

The orchestrator independently verified both halves of the Task 1 SDK citation before putting the decision to the human (source lines, the `#26` comment, and both scenario comment sites) — confirmed as accurate.

## RED-first observation (Task 3, verbatim)

Command: `go test ./test/wireoracle/ -run '^(TestToolsListRepeatOrderingResolution|TestFrozenTranscriptComparisonDetectsContentMutation)$' -count=1 -v`, run against the identity-stub `CanonicalizeResponseOrder`.

**Exit code: 1**

```
    normalize_test.go:243: CanonicalizeResponseOrder did not resequence out-of-order responses:
         got:  {"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"x"}}
        {"jsonrpc":"2.0","id":3,"result":{"tools":["c"]}}
        {"jsonrpc":"2.0","id":2,"result":{"tools":["c"]}}
        {"jsonrpc":"2.0","method":"notifications/tools/list_changed"}
        {"jsonrpc":"2.0","id":5,"result":"e"}
        want: {"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"x"}}
        {"jsonrpc":"2.0","id":2,"result":{"tools":["c"]}}
        {"jsonrpc":"2.0","id":3,"result":{"tools":["c"]}}
        {"jsonrpc":"2.0","method":"notifications/tools/list_changed"}
        {"jsonrpc":"2.0","id":5,"result":"e"}
--- FAIL: TestToolsListRepeatOrderingResolution (0.00s)
    normalize_test.go:292: canonicalized reorder-only transcript must compare EQUAL (no content changed): normalized transcript differs at line 2:
         got: "{\"jsonrpc\":\"2.0\",\"id\":3,...\"description\":\"List a symbol's reverse callers\"...}"
        want: "{\"jsonrpc\":\"2.0\",\"id\":2,...\"description\":\"List a symbol's reverse callers\"...}"
--- FAIL: TestFrozenTranscriptComparisonDetectsContentMutation (0.00s)
FAIL
```

After implementing the real logic: both tests PASS (GREEN), verified this session.

## Manual planted-content-mutation observation (live gate)

Mutated `testdata/wireoracle/transcripts/toolslist-repeat.golden` temporarily (one character: `"List a symbol's forward call targets"` → `"...targetz"`), never committed:

- **Before revert:** `go test ./test/wireoracle/ -run '^TestFrozenTranscriptsMatch$/^toolslist-repeat$' -count=1 -v` → exit code 1, `--- FAIL: TestFrozenTranscriptsMatch/toolslist-repeat (0.66s)`, failure text quotes the mutated `"targetz"` vs. the unmutated `"targets"` on line 2 of the comparison.
- **After `cp` restore of the original bytes:** same command → exit code 0, `--- PASS: TestFrozenTranscriptsMatch/toolslist-repeat (0.64s)`.
- `git diff --stat -- testdata/wireoracle/transcripts/toolslist-repeat.golden` confirmed empty both before the plant and after the revert — the golden file was never left dirty and nothing was committed mid-plant.

## `rg` count before/after (scenarios.go claim correction)

`rg -o 'handled synchronously in' test/wireoracle/scenarios.go | wc -l`: **3 before, 1 after.** The single remaining occurrence (`:461`, inside `"...method) is handled synchronously inline before the next stdin line is even read."`) is the correctly-scoped mark3labs `tools/call` worker-pool note, out of scope for this plan (verified, not assumed — that paragraph is historically accurate for the OLD transport it describes and was not touched).

## Backstop verification (full-suite, under contention)

- `task test:wireoracle` (fresh): `ok github.com/seanb4t/codegraph-go/test/wireoracle 47.324s`
- `task test:unit`, fresh (`-count=1`, all non-daemon packages): exit 0, 0 `FAIL` lines
- **The decisive proof:** re-ran the EXACT contention method from Task 1's reproduction (Linux/arm64 container, `--cpus=4`, `GOMAXPROCS=4`, 6 parallel `go test -run toolslist-repeat -count=10` processes, 60 total attempts) against the fixed tree: **60/60 pass**, versus 56/60 (4 failures) before the fix.

## Deviations from Plan

None — plan executed exactly as written. Task 2's `checkpoint:decision` (`gate="blocking-human"`) was answered by the maintainer as designed; the executor did not auto-select an option, consistent with this project's rule that a `blocking-human` gate is never bypassed even under `mode: yolo` / `auto_advance: true`.

One non-blocking observation: this plan's `requirements: [TODO-MCP-01]` frontmatter field has no corresponding entry in `.planning/REQUIREMENTS.md` — `gsd-tools query requirements.mark-complete TODO-MCP-01` returned `not_found` and made no write (`.planning/REQUIREMENTS.md` diff confirmed empty). This is expected: `03-CONTEXT.md`'s own framing states this plan is unrelated work riding along the phase ("must not gate this phase's five browse success criteria"), not a tracked roadmap requirement — so `TODO-MCP-01` was never meant to resolve to a REQUIREMENTS.md row.

## Issues Encountered

- **Self-caught commit-staging bug.** `e8138c0`'s `git add` call passed both the new `completed/` path and the already-moved-away `pending/` path in one invocation with stderr suppressed (`2>/dev/null`); git errored on the nonexistent `pending/` pathspec and staged nothing beyond the automatic rename `git mv` had already recorded, so the resolution frontmatter/section never actually landed in that commit despite the commit succeeding. Caught via a routine `git status --short` check before the final metadata commit (the working tree was not clean, as expected after a completed plan); fixed in a new follow-up commit (`7ad65cb`), not an amend. Lesson: never suppress `git add`'s stderr with multiple pathspecs, and always check `git status --short` is clean immediately before the final metadata commit, not just after individual task commits.
- **Reproduction platform caveat.** The reproducing Docker container is linux/**arm64** (Apple Silicon host resolved `golang:1.26.5` to its native variant), not linux/amd64 like the CI runner (`namespace-profile-linux-amd64-4x8`). Two follow-up attempts to build a standalone repro binary inside the same container hit resource limits unrelated to this investigation (CGo compilation of `tree-sitter-c-sharp` OOM-killed under constrained container memory; a `cp -r /repo` staging step timed out on this repo's large `web/node_modules`/`graphify-out` trees). The successful 60-attempt reproduction (both before and after the fix) used `go test` directly against the read-only bind-mounted repo, which avoided both issues. Recorded in `03-03-EVIDENCE.md` as an assumption, not fully verified: kernel-family (Linux goroutine/thread scheduling) rather than CPU architecture is the more likely factor in why darwin never reproduced it, since the original CI report was also Linux-specific on an unconstrained amd64 host.
- **First-differing-line limitation.** `compareBytesLineByLine`/`assertBytesEqualLineByLine` report only the FIRST differing line by design, so the original live-reproduced failure (captured before the fix) did not directly show whether the id-2 response appeared later in the transcript (reordered) or never arrived (dropped). The original CI report's "tool payloads match exactly" is consistent with a pure reorder; this session did not independently re-derive that via a full raw dump of a live-reproduced failure — noted as a gap in `03-03-EVIDENCE.md` rather than papered over. It does not change the verdict or the fix: either outcome is still the server writing responses in an order other than request order.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

This plan's outcome is unrelated to and does not gate any of Phase 3's five Browse success criteria, per its own objective statement. `task test:wireoracle` and `task test:unit` are both green; the wire-oracle required PR leg is no longer intermittently red for this scenario. No frozen transcript was regenerated at any point (`git diff --stat -- test/wireoracle/testdata testdata/wireoracle` empty throughout, verified repeatedly). Ready for the next Browse-view plan in this phase.

---

*Phase: 03-browse-inspect-navigation*
*Completed: 2026-08-28*

## Self-Check: PASSED

- All 9 claimed files verified present on disk (`[ -f ]`), including the todo's new location and its confirmed absence from `pending/`.
- All 6 claimed commit hashes (`7b4b4f52`, `c22660c`, `7197083`, `14fd221`, `e8138c0`, `eaf241e`) verified present in `git log --oneline --all`.
