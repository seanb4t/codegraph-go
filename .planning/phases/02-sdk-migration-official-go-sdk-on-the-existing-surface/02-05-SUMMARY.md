---
phase: 02-sdk-migration-official-go-sdk-on-the-existing-surface
plan: 05
subsystem: testing
tags: [mcp, go-sdk, wire-oracle, protocol-negotiation, jsonschema, migration]

# Dependency graph
requires:
  - phase: 02-sdk-migration-official-go-sdk-on-the-existing-surface
    provides: "02-01 through 02-04 — internal/mcp migrated to modelcontextprotocol/go-sdk@v1.7.0, mark3labs/mcp-go removed from go.mod"
provides:
  - "23 wire-oracle transcripts re-frozen against the go-sdk-backed binary, every changed line attributed to one of nine named causes"
  - "legacyOmittedVersionCoercion constant and edge-call-before-initialize doc comment updated as the two permitted harness-code exceptions"
  - "Wire oracle demonstrated RED against a confirmed-applied mutation on the go-sdk backend, then green after revert"
  - "check:transcript-freeze's self-expiring SDK-01 exemption confirmed firing on the real branch diff"
affects: [phase-03-2026-07-28-obligations, phase-04-maintenance]

# Actuals (#2632)
actuals:
  tokens: 14490
  tasks: 3
  commits: 1

tech-stack:
  added: []
  patterns:
    - "D-03 divergence-record-in-commit-message mechanism: no expected-diff file, no ledger — the re-freeze commit message enumerates every named cause with counts"
    - "Human-diff-read checkpoint precedes any golden-file regeneration (T-02-20 mitigation)"

key-files:
  created: []
  modified:
    - test/wireoracle/scenarios.go
    - testdata/wireoracle/transcripts/call-callees.golden
    - testdata/wireoracle/transcripts/call-callers.golden
    - testdata/wireoracle/transcripts/call-files.golden
    - testdata/wireoracle/transcripts/call-impact.golden
    - testdata/wireoracle/transcripts/call-node.golden
    - testdata/wireoracle/transcripts/call-search.golden
    - testdata/wireoracle/transcripts/call-status.golden
    - testdata/wireoracle/transcripts/edge-call-before-initialize.golden
    - testdata/wireoracle/transcripts/error-confinement-reject.golden
    - testdata/wireoracle/transcripts/error-malformed-args.golden
    - testdata/wireoracle/transcripts/error-unknown-method.golden
    - testdata/wireoracle/transcripts/error-unknown-tool.golden
    - testdata/wireoracle/transcripts/handshake-explore.golden
    - testdata/wireoracle/transcripts/legacy-2024-11-05.golden
    - testdata/wireoracle/transcripts/legacy-2025-03-26.golden
    - testdata/wireoracle/transcripts/legacy-2025-06-18.golden
    - testdata/wireoracle/transcripts/legacy-2025-11-25.golden
    - testdata/wireoracle/transcripts/legacy-omitted-version.golden
    - testdata/wireoracle/transcripts/legacy-unsupported-2026-07-28.golden
    - testdata/wireoracle/transcripts/toolslist-allowlist.golden
    - testdata/wireoracle/transcripts/toolslist-default.golden
    - testdata/wireoracle/transcripts/toolslist-no-index.golden
    - testdata/wireoracle/transcripts/toolslist-repeat.golden

key-decisions:
  - "All 23 transcripts differ from frozen; every changed line attributed to one of nine named causes (seven cosmetic/additive, two semantic: #2 legacy-omitted-version's value change, #9 edge-call-before-initialize's success-to-rejection flip)."
  - "Cause #9 (edge-call-before-initialize now rejects a tools/call sent before initialize, code 0) was NOT predicted by 02-RESEARCH.md or any of the eight causes it enumerated — discovered only by the full transcript diff read, exactly the failure mode the checkpoint exists to catch. Escalated and accepted by the maintainer at the Task 1 checkpoint: go-sdk's rejection is spec-correct (MCP requires initialize precede other requests); mark3labs' permissiveness was the deviation. Zero known blast radius (VRFY-05 audit surfaced no client calling tools before initializing)."
  - "Error code 0 on the edge-call-before-initialize rejection is recorded as an upstream go-sdk defect (github.com/modelcontextprotocol/go-sdk#976, predicted by PITFALLS.md) — not worked around, not anchored in anchors.go, not codegraph-go's to fix. Phase 3 input at most."
  - "Two harness-code exceptions to ROADMAP criterion 2's 'harness code unmodified' bar, both hand-authored mirrors of an observed transcript truth rather than mechanism changes: legacyOmittedVersionCoercion's value+doc-comment (cause #2), and edge-call-before-initialize's doc comment retracting Phase 1's now-false 'accidental asset for Phase 3's statelessness work' framing (cause #9)."
  - "legacy-unsupported-2026-07-28's protocolVersion did NOT move to 2026-07-28 — D-05 CORRECTED confirmed empirically a second time (02-RESEARCH.md already proved this; this plan re-confirmed against the real re-frozen bytes)."
  - "toolslist-no-index still advertises tools:{listChanged:true} with no logging key — D-11's regression fix holds at zero registered tools."
  - "Both hand-authored spec anchors (-32601, -32602) held; only their message text changed (cause #8). Neither anchor required editing."

patterns-established: []

requirements-completed: [SDK-01, SDK-05]

coverage:
  - id: D1
    description: "All 23 wire-oracle transcripts re-frozen against the go-sdk-backed binary; task test:wireoracle exits 0."
    requirement: "SDK-01"
    verification:
      - kind: integration
        ref: "go test ./test/wireoracle/... -run TestFrozenTranscriptsMatch (23/23 pass)"
        status: pass
    human_judgment: false
  - id: D2
    description: "Every changed transcript line attributed to a named cause (nine total); three must-not-have-changed properties confirmed with observed values; no unexplained line survives."
    requirement: "SDK-01"
    verification: []
    human_judgment: true
    rationale: "This is precisely the class of judgment call D-01's human-diff-read checkpoint exists for — a comparator cannot distinguish a cosmetic reorder from a silent regression. The maintainer performed this review live at the Task 1 checkpoint and accepted cause #9 explicitly; recorded here as already adjudicated, not deferred."
  - id: D3
    description: "Wire oracle demonstrated RED against a confirmed-applied mutation on the go-sdk backend (Mutation 1, stray stdout line), then green after revert; internal/mcp/server.go byte-identical post-revert."
    requirement: "SDK-01"
    verification:
      - kind: integration
        ref: "go test ./test/wireoracle/... -count=1 (mutation applied: 23/23 TestFrozenTranscriptsMatch fail + 23/23 TestSpecAnchorsHold framing fail; reverted: ok)"
        status: pass
    human_judgment: false
  - id: D4
    description: "Harness code (test/wireoracle/ minus scenarios.go) unmodified — git diff --name-only against the phase's starting commit lists only scenarios.go."
    requirement: "SDK-01"
    verification:
      - kind: other
        ref: "git diff --name-only 38a8486 -- test/wireoracle/ -> test/wireoracle/scenarios.go (only)"
        status: pass
    human_judgment: false
  - id: D5
    description: "SDK-05 schema audit: zero enum constraints existed to lose; only constraint-semantics deltas are number->integer (7 params) and additionalProperties:false (8 tools, accepted improvement)."
    requirement: "SDK-05"
    verification:
      - kind: integration
        ref: "transcript diff review, causes #4/#5/#7 in the re-freeze commit message"
        status: pass
    human_judgment: false

duration: ~2h active (spanning a human checkpoint pause between Task 1 and Tasks 2-3)
completed: 2026-08-06
status: complete
---

# Phase 2 Plan 5: Wire Oracle Re-freeze Against go-sdk Summary

**All 23 wire-oracle transcripts re-frozen against the go-sdk@v1.7.0-backed binary — nine named causes, including one maintainer-adjudicated semantic regression (session-ordering enforcement) that no prior research document predicted.**

## Performance

- **Duration:** ~2h active work (Task 1's attribution + Tasks 2-3's re-freeze/mutation-proof), spanning a human checkpoint pause for the Task 1 diff review
- **Tasks:** 3 (Task 1 checkpoint:human-verify, Task 2 auto, Task 3 auto)
- **Files modified:** 24 (`test/wireoracle/scenarios.go` + 23 `.golden` transcripts)
- **Commits:** 1 (`f4c9052`)

## Accomplishments

- Captured all 23 scenarios against the real go-sdk-backed `codegraph` binary using the oracle's own `wireoracle` capture CLI, diffed against the Phase 1 frozen baseline, and read every changed line.
- Attributed every changed line across all 23 transcripts to one of nine named causes (seven cosmetic/additive from `02-RESEARCH.md`, two semantic — one predicted, one discovered live during the diff review).
- Discovered and escalated an unpredicted semantic regression: `edge-call-before-initialize` flips from a successful `tools/call` result to a rejection (`{"code":0,...}`) because go-sdk enforces MCP's session-initialization ordering, which mark3labs never did. The maintainer adjudicated this at the checkpoint: accepted as spec-correct, zero known blast radius, recorded as cause #9.
- Re-froze all 23 `.golden` transcripts and moved the one required harness constant (`legacyOmittedVersionCoercion`), plus retracted a now-false Phase 1 doc comment (`edge-call-before-initialize`) — the two permitted exceptions to ROADMAP criterion 2's harness-code-unmodified bar, both argued and proven confined via `git diff --name-only`.
- Committed the full nine-cause divergence record, both non-events, and the SDK-05 audit finding in a single commit message per D-03's mechanism — no ledger file.
- Re-proved the wire oracle's non-vacuity on the new backend: re-applied Phase 1's Mutation 1 (stray stdout line), confirmed both `TestFrozenTranscriptsMatch` and `TestSpecAnchorsHold`'s framing invariant go RED across all 23 scenarios, reverted, confirmed green and byte-identical.
- Confirmed `check:transcript-freeze`'s self-expiring SDK-01 exemption fires correctly on the real branch diff against `main` (exit 0, exemption notice on stderr).

## Task Commits

1. **Task 1: The diff review — the phase's actual acceptance mechanism (D-01)** — checkpoint, no commit (read-only diff review; findings recorded in this SUMMARY per the task's own instruction)
2. **Task 2: Re-freeze the 23 transcripts and move the one expected-value constant** — `f4c9052` (test)
3. **Task 3: Prove the oracle still fails, and that only one harness line moved** — folded into `f4c9052`'s working tree (mutation applied/reverted, zero residual diff; no separate commit since `internal/mcp/server.go` ends byte-identical to its pre-mutation state, so there is nothing to commit for this task)

**Plan metadata:** this SUMMARY's own commit (pending, alongside STATE.md/ROADMAP.md updates)

## Files Created/Modified

- `test/wireoracle/scenarios.go` — `legacyOmittedVersionCoercion` moved `"2025-03-26"` → `"2025-11-25"` with doc comment re-cited to go-sdk's `negotiatedVersion` fallback; `edge-call-before-initialize`'s doc comment retracted Phase 1's "accidental asset" framing and records go-sdk's session-ordering enforcement instead
- 23× `testdata/wireoracle/transcripts/*.golden` — re-frozen from the go-sdk-backed binary via the oracle's own capture CLI, no hand edits

## Decisions Made

**All nine causes, with counts** (full detail in commit `f4c9052`'s message):

1. `initialize` key order `capabilities, protocolVersion, serverInfo` — 22/23 transcripts (all but `edge-call-before-initialize`, which has no `initialize` response)
2. **[SEMANTIC]** `legacy-omitted-version` value `"2025-03-26"` → `"2025-11-25"` — 1 transcript
3. `ttlMs:0,cacheScope:"private"` on `tools/list` — 11 transcripts (`cacheScope` confirmed `"private"` everywhere, never `"public"`)
4. `additionalProperties:false` + schema key reorder — same 10 transcripts as #3 minus `toolslist-no-index`
5. `number`→`integer` on 7 numeric params — same set as #4
6. `annotations` key reorder, all 4 hint values survive — same set as #4
7. `required:[]` omitted for zero-required tools (`codegraph_node`, `codegraph_status`) — `toolslist-allowlist`, `toolslist-repeat`
8. Error message wording changed, codes unchanged — `error-unknown-method`, `error-malformed-args`, `error-unknown-tool` (3 transcripts)
9. **[SEMANTIC]** `edge-call-before-initialize` flips success → rejection (`code:0`) — 1 transcript, maintainer-adjudicated

**Two non-events confirmed with observed values:** `legacy-unsupported-2026-07-28`'s `protocolVersion` stayed `"2025-11-25"` (did not move to `"2026-07-28"`); `toolslist-no-index` kept `tools:{listChanged:true}` with no `logging` key.

**Both spec anchors held:** `error-unknown-method` → `-32601`; `error-malformed-args` → `-32602`.

**SDK-05 finding:** zero enum constraints existed to lose (`codegraph_files`' `format` values live only in its description string); every description and `required` set survives; only deltas are `number`→`integer` (7 params) and `additionalProperties:false` (accepted improvement).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 4 - Architectural/semantic, escalated to maintainer] `edge-call-before-initialize` behavioral flip not covered by the plan's eight predicted causes**
- **Found during:** Task 1 (diff review checkpoint)
- **Issue:** Pre-migration, a `tools/call` sent with no prior `initialize` succeeds (Phase 1 explicitly locked this in as "an accidental but real asset for Phase 3's statelessness work"). Post-migration, go-sdk rejects it with `{"code":0,"message":"method \"tools/call\" is invalid during session initialization"}`. This is a full success-to-rejection flip, not a cosmetic reorder, and matches none of `02-RESEARCH.md`'s eight predicted causes.
- **Resolution:** Per Rule 4 (architectural/semantic change requiring a decision, not an auto-fixable bug), this was surfaced via the `checkpoint:human-verify` gate rather than silently attributed. The maintainer reviewed and accepted it as cause #9: go-sdk's enforcement is spec-correct (MCP requires `initialize` precede other requests); mark3labs' permissiveness was the deviation, not this. Zero known blast radius per VRFY-05's client audit.
- **Files modified:** `test/wireoracle/scenarios.go` (doc comment retraction), `testdata/wireoracle/transcripts/edge-call-before-initialize.golden` (re-frozen)
- **Committed in:** `f4c9052`

---

**Total deviations:** 1 escalated-and-resolved (Rule 4, maintainer decision required and obtained)
**Impact on plan:** The plan's own design worked exactly as intended — the checkpoint caught a real, unpredicted semantic change that no automated comparator or prior research would have found, and routed it to a human decision rather than silently absorbing or blocking on it.

## Issues Encountered

- **Go test cache anomaly (operational note, not a defect):** one `task test:wireoracle` invocation returned `ok (cached)` immediately after editing `internal/mcp/server.go` for Mutation 1, despite the source change. Re-running with `go test ./test/wireoracle/... -count=1` produced the correct, fresh RED result matching the expected mutation signature exactly. All subsequent verification in this plan used explicit `-count=1` to avoid relying on ambiguous cache behavior. Root cause not investigated (out of scope, did not block the task).
- **`internal/daemon` full-suite flake:** `task test` failed once on `TestConvergenceTwoSessions` under full-suite parallel load — this is the documented, pre-existing flake (issue #17/MAINT-02, scheduled Phase 4). Confirmed `internal/daemon` does not import `internal/mcp` (`go list -deps` — zero hits) and passes isolated (`go test ./internal/daemon/... -count=1` — ok). Per this plan's execution instructions, not investigated or fixed. `task test:wireoracle` itself and every other package in the full suite passed.
- **Worktree discontinuity between Task 1's checkpoint and Task 2's resume:** the isolated worktree used for Task 1 no longer existed when work resumed after the human's approval; the working directory had reset to the main repo checkout, which was confirmed to be on the correct integration branch (`gsd/v0.3.0-mcp-protocol-currency`) at the exact expected starting commit. Verified via `.git` being a directory (not a linked worktree) and `git rev-parse --show-toplevel` before proceeding, consistent with this being the final wave of the phase (no further merge-back needed).

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Phase 2 (SDK Migration) is fully green: `internal/mcp` runs on `modelcontextprotocol/go-sdk@v1.7.0`, `mark3labs/mcp-go` is out of `go.mod` (02-04), and the wire oracle — Phase 1's acceptance mechanism — is re-frozen and demonstrated non-vacuous against the new backend.
- Phase 3 inherits this 23-transcript corpus as its comparison baseline, plus two explicit carry-forwards: (1) the deferred `legacy-unsupported-2026-07-28` scenario rename (cosmetic, deliberately not done here per this plan's objective), and (2) the upstream go-sdk error-code-0 defect (`modelcontextprotocol/go-sdk#976`) as an input, not a requirement, to any future error-shape work.
- No blockers for Phase 3.

---
*Phase: 02-sdk-migration-official-go-sdk-on-the-existing-surface*
*Completed: 2026-08-06*
