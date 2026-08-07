---
phase: 03-2026-07-28-spec-compliance
plan: 03
subsystem: mcp
tags: [mcp, go-sdk, wire-oracle, sep-2575, meta-validation, error-codes]

# Dependency graph
requires:
  - phase: 03-2026-07-28-spec-compliance
    provides: "03-01's modernMetaParams()/discoverRequest()/modernToolCallRequest() helpers and modern-discover-explore tracer; ExpectedScenarioCount at 24"
provides:
  - "SPEC-02's -32602 half frozen and anchored: a well-formed Modern _meta missing io.modelcontextprotocol/clientCapabilities answers -32602 (modern-meta-invalid-params)"
  - "SPEC-02's -32022 half frozen and anchored: a well-formed Modern _meta offering a supported-shape-but-unrecognized version that sorts lexically after \"2026-07-28\" answers -32022 (modern-meta-unsupported-version)"
  - "modernUnsupportedVersion (\"2099-01-01\"), documenting in full why its lexical relationship to \"2026-07-28\" is load-bearing, and retracting 03-CONTEXT.md's \"SPEC-02 is a real gap\" framing"
  - "codeUnsupportedProtocolVersion = -32022, hand-authored beside codeMethodNotFound/codeInvalidParams"
  - "assertErrorCode's want-id parameter, generalizing it beyond the id=2-after-initialize assumption every pre-existing anchored scenario shared"
  - "ExpectedScenarioCount at 26, the base any later Phase 3 plan's scenario additions build on"
affects: [03-04, 03-05]

# Actuals (#2632)
actuals:
  tokens: 4234
  tasks: 2
  commits: 2

tech-stack:
  added: []
  patterns:
    - "Variation helpers built ON TOP OF an existing base-shape helper (modernMetaMissingCapabilities/modernMetaWithVersion wrap modernMetaParams via delete/overwrite) rather than re-typing its key literals, so the two can never drift apart"
    - "assertErrorCode's want-id is now an explicit parameter, not an assumption baked into the function body — the correct generalization once a NoInitialize sessionless scenario's error lands at a different id than every session-based scenario's id=2"

key-files:
  created:
    - testdata/wireoracle/transcripts/modern-meta-invalid-params.golden
    - testdata/wireoracle/transcripts/modern-meta-unsupported-version.golden
  modified:
    - test/wireoracle/scenarios.go
    - test/wireoracle/anchors.go

key-decisions:
  - "Zero server code: internal/mcp/ is untouched by this plan (git diff --stat -- internal/mcp/ is empty). SPEC-02's -32602 and -32022 halves are already correct in go-sdk@v1.7.0 for every well-formed Modern _meta shape; the plan's entire job was proving that, not building it."
  - "No -32601 scenario or anchor added. 03-CONTEXT.md's original \"SPEC-02 is a real gap\" observation (probed with a lexically-old \"1999-01-01\") is retracted in-file (modernUnsupportedVersion's doc comment, and Anchors()'s doc comment) as go-sdk's own lexical-comparison classification quirk, not a codegraph-go defect and not a shape SPEC-02 asks this suite to prove."
  - "Avoided spelling the literal SDK identifier names (mcp.CodeUnsupportedProtocolVersion, mcp.MetaKeyProtocolVersion) anywhere in my own new prose, even in doc comments explaining why they're not imported — matching 03-01's established precedent and keeping my own additions at zero hits for the plan's whole-tree grep acceptance criteria. (internal/mcp/archtest's own pre-existing self-test files legitimately mention these names in prose as part of testing the VRFY-02 guard itself; those are untouched baseline noise, not something this plan introduced — see Issues Encountered.)"

patterns-established:
  - "A hand-authored error-code anchor for a NoInitialize sessionless scenario must pass its own request/response id explicitly to assertErrorCode, not rely on the id=2-after-initialize assumption every session-based scenario shared."

requirements-completed: [SPEC-02]

coverage:
  - id: D1
    description: "A well-formed Modern _meta whose io.modelcontextprotocol/clientCapabilities key is absent is answered -32602, frozen and independently anchored against a fresh capture"
    requirement: "SPEC-02"
    verification:
      - kind: integration
        ref: "test/wireoracle TestFrozenTranscriptsMatch/modern-meta-invalid-params"
        status: pass
      - kind: integration
        ref: "test/wireoracle TestSpecAnchorsHold/modern-meta-invalid-params"
        status: pass
    human_judgment: false
  - id: D2
    description: "A well-formed Modern _meta offering a supported-shape-but-unrecognized protocol version that sorts lexically after \"2026-07-28\" is answered -32022, frozen and independently anchored against a fresh capture"
    requirement: "SPEC-02"
    verification:
      - kind: integration
        ref: "test/wireoracle TestFrozenTranscriptsMatch/modern-meta-unsupported-version"
        status: pass
      - kind: integration
        ref: "test/wireoracle TestSpecAnchorsHold/modern-meta-unsupported-version"
        status: pass
    human_judgment: false
  - id: D3
    description: "ExpectedScenarioCount moved 24 -> 26 in the same commit as the two new scenarios and their transcripts"
    verification:
      - kind: unit
        ref: "test/wireoracle TestScenarioCountIsExact"
        status: pass
      - kind: unit
        ref: "test/wireoracle TestTranscriptSetMatchesScenarioSet"
        status: pass
    human_judgment: false
  - id: D4
    description: "Both new anchors were demonstrated RED against confirmed-applied, individually-reverted mutations, and the VRFY-02 archtest guard remains unaffected"
    verification:
      - kind: unit
        ref: "test/wireoracle TestSpecAnchorsHold (post-revert green run)"
        status: pass
      - kind: unit
        ref: "internal/mcp/archtest full suite"
        status: pass
    human_judgment: false

duration: 5min
completed: 2026-08-06
status: complete
---

# Phase 3 Plan 3: SPEC-02 `_meta` Validation Proof Summary

**SPEC-02's `-32602`/`-32022` per-request `_meta` failure codes frozen and anchored against go-sdk's already-correct behavior — zero server code added, and 03-CONTEXT.md's original `-32601` "gap" observation retracted in-file as go-sdk's own lexical version-string comparison, not a codegraph-go defect.**

## Performance

- **Duration:** ~5 min (commit-to-commit; investigation/reading time not separately tracked)
- **Started:** 2026-08-06T11:40:23-04:00 (first task commit)
- **Completed:** 2026-08-06T11:45:27-04:00 (final task commit)
- **Tasks:** 2
- **Files modified:** 4 (2 created, 2 modified)

## Accomplishments

- Added `modernMetaMissingCapabilities()`, `modernMetaWithVersion(version)`, and `discoverRequestWithMeta(id, meta)` to `test/wireoracle/scenarios.go` — variation helpers built on top of 03-01's `modernMetaParams()` (delete one key / overwrite one key) rather than re-typing its literals.
- Added `modernUnsupportedVersion = "2099-01-01"`, with a doc comment stating in full why its lexical relationship to `"2026-07-28"` is load-bearing (go-sdk's `validateRequestMeta` reclassifies any lexically-smaller `_meta.protocolVersion` as not-modern before the unsupported-version check runs) and retracting 03-CONTEXT.md's `"SPEC-02 is a real gap"` framing of the resulting `-32601` observation.
- Added two new scenarios — `modern-meta-invalid-params` (`-32602`, missing `clientCapabilities`) and `modern-meta-unsupported-version` (`-32022`, offering `modernUnsupportedVersion`) — both `NoInitialize: true`, `Index: true`, one request each. Froze both transcripts via the oracle's own capture CLI against a freshly rebuilt `bin/codegraph`, never hand-written.
- Moved `ExpectedScenarioCount` from 24 to 26 in the same commit, extending its doc comment's arithmetic.
- Added `codeUnsupportedProtocolVersion = -32022` to `test/wireoracle/anchors.go`, hand-authored beside `codeMethodNotFound`/`codeInvalidParams`, following their exact doc-comment style.
- Registered two new `Anchor` entries against the two new scenarios, both at response id=1 (these are `NoInitialize` sessionless requests, not id=2-after-initialize like every pre-existing anchored scenario).
- Gave `assertErrorCode` an explicit `wantID float64` parameter (previously hardcoded to 2) and updated its two pre-existing call sites to pass `2` explicitly. Its "no error response found" fatal is unchanged.
- Rewrote the `Anchors()` doc-comment paragraph that used to explain why no unsupported-version anchor existed at all — that was only ever true for `legacy-unsupported-2026-07-28`'s classic-`initialize` silent-coercion path (still true, still unanchored); the Modern `_meta` path now does answer an error and is now anchored. Made the distinction between the two scenarios' outcomes explicit so a future reader doesn't conflate them.
- Demonstrated both new anchors RED against confirmed-applied, individually-reverted mutations (see below), then confirmed `git diff -- test/wireoracle/scenarios.go` empty and the full wire-oracle suite green.

## Task Commits

Each task was committed atomically:

1. **Task 1: Freeze both `_meta` failure answers, and record why the third shape is not a gap** - `5a6bf07` (test)
2. **Task 2: Hand-author the `-32022` anchor and demonstrate both new anchors RED** - `8760d2e` (test)

_No plan-metadata commit yet — this SUMMARY plus STATE.md/ROADMAP.md/REQUIREMENTS.md updates land in the final `docs(03-03)` commit below._

## Files Created/Modified

- `test/wireoracle/scenarios.go` - added `modernMetaMissingCapabilities`, `modernMetaWithVersion`, `discoverRequestWithMeta`, `modernUnsupportedVersion`, the two new scenarios, and bumped `ExpectedScenarioCount` to 26
- `test/wireoracle/anchors.go` - added `codeUnsupportedProtocolVersion`, two new `Anchor` entries, `assertErrorCode`'s explicit `wantID` parameter, and rewrote the stale no-anchor paragraph
- `testdata/wireoracle/transcripts/modern-meta-invalid-params.golden` - new frozen transcript (created via `test/wireoracle/cmd/wireoracle`, never hand-written)
- `testdata/wireoracle/transcripts/modern-meta-unsupported-version.golden` - new frozen transcript (same mechanism)

## Decisions Made

- Kept the two `_meta` failure proofs as two separate scenarios (not folded into one), matching the plan's explicit design — each isolates exactly one failure mode (missing key vs. unsupported value) so a future regression in one path can't hide behind the other passing.
- Avoided spelling the literal SDK identifier names (`mcp.CodeUnsupportedProtocolVersion`, `mcp.MetaKeyProtocolVersion`) anywhere in my own new prose. The plan's Task 2 `<action>` text originally modeled this doc comment as explicitly naming `mcp.CodeUnsupportedProtocolVersion`; I reworded it to describe the SDK's constant functionally instead ("the SDK's own equivalently-named exported error-code constant") after noticing this would otherwise violate the plan's own whole-tree grep acceptance criterion for that identifier. This is a wording change only — the substance (why it's hand-authored, why importing it would trip VRFY-02) is unchanged.

## Deviations from Plan

None requiring a stop or architectural discussion. One self-correction during Task 2 worth recording:

**Wording adjustment, not a Rule 1-4 deviation:** The plan's Task 2 `<action>` block describes `codeUnsupportedProtocolVersion`'s doc comment as stating "that importing `mcp.CodeUnsupportedProtocolVersion` would additionally trip internal/mcp/archtest's VRFY-02 name heuristic" — i.e., it directs writing that literal identifier name into anchors.go's prose. Task 2's own acceptance criteria separately require `rg -n 'CodeUnsupportedProtocolVersion' --glob '!.planning/**' .` to return zero hits across the whole tree. Following the `<action>` text literally would have made the acceptance criterion fail (against my own file, not baseline noise). I resolved this by rewording the doc comment to convey the same substance without spelling the literal token, consistent with how Task 1's `modernUnsupportedVersion` doc comment (also written this plan) already avoided it. No functional behavior changed; the VRFY-02 guard is AST/structural (confirmed by `go test ./internal/mcp/archtest/...` passing both before and after), so this was a documentation-wording choice, not a guard-evasion concern.

## Issues Encountered

- The Task 1 and Task 2 acceptance criteria's whole-tree grep for `CodeUnsupportedProtocolVersion`/`MetaKeyProtocolVersion` (`--glob '!.planning/**'`) returns non-zero even on a clean tree with none of this plan's changes applied: `internal/mcp/archtest/protocol_version_test.go` and `internal/mcp/archtest/protocol_version_selftest_test.go` (both pre-existing, from Phase 2's `02-04` commit `0e96d30`) legitimately mention both names in prose as part of documenting and self-testing the VRFY-02 guard itself. This is baseline noise unrelated to this plan's scope (I did not touch `internal/mcp/archtest/`, per the plan's explicit "Do NOT touch `internal/mcp/`" instruction) — confirmed via `git status --short internal/mcp/archtest/` (empty) and `git log -1 -- <those files>` (last touched by Phase 2). Verified instead that my own new code (`test/wireoracle/scenarios.go`, `test/wireoracle/anchors.go`) contributes zero hits to that grep, and that the functional guard (`go test ./internal/mcp/archtest/... -count=1`) passes both before and after this plan's changes.
- Mutation 1's observed failure mode differed slightly from the plan's stated prediction. The plan's `<action>` text predicted swapping `modernUnsupportedVersion` to a supported era string would make "the unsupported-version anchor FAIL (the request now succeeds, so no error response exists at id 1)". The actual observed failure was different and, on reflection, more informative: `"2025-11-25"` is itself lexically SMALLER than `"2026-07-28"`, so the mutated request falls into the exact lexical-reclassification trap `modernUnsupportedVersion`'s own doc comment describes — it does NOT succeed; it fails with `-32601` (`method not found: "server/discover"`), the very mechanism this plan documents. The anchor test still correctly FAILED (asserting `-32022`, got `-32601`), satisfying the acceptance criterion's actual requirement ("Confirm the unsupported-version anchor FAILS"), just via a more instructive failure mode than the plan anticipated. Quoted observation: `error.code = -32601, want -32022: "{\"jsonrpc\":\"2.0\",\"id\":1,\"error\":{\"code\":-32601,\"message\":\"method not found: \\\"server/discover\\\"\"}}"`.
- Mutation 2 (restoring `modernMetaParams()` in place of `modernMetaMissingCapabilities()`) matched the plan's prediction exactly: `no error response (id=1) found in captured stdout — a missing response must never read as a pass`.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- `ExpectedScenarioCount` is now 26 and `Scenarios()`/`testdata/wireoracle/transcripts/` agree — the base later Phase 3 plans (03-04, 03-05) build their own scenario additions on top of.
- `modernMetaMissingCapabilities()`, `modernMetaWithVersion(version)`, and `discoverRequestWithMeta(id, meta)` are now available in `test/wireoracle/scenarios.go` for reuse by any later plan needing to construct a variant Modern `_meta`-bearing request.
- `assertErrorCode`'s explicit `wantID` parameter is the established pattern for any future anchored error scenario that isn't a classic id=2-after-initialize shape.
- SPEC-02 is closed. `git diff --stat -- internal/mcp/` is empty for this entire plan — no server code was added, matching the plan's central premise that SPEC-02 was already satisfied by go-sdk@v1.7.0 for every well-formed Modern client shape.
- No blockers. `internal/mcp/archtest`'s VRFY-02 guard was not triggered by any new code this plan added (confirmed both by the passing archtest suite and by a direct grep of my own new files).

---
*Phase: 03-2026-07-28-spec-compliance*
*Completed: 2026-08-06*

## Self-Check: PASSED

All created/modified files found on disk (`test/wireoracle/scenarios.go`, `test/wireoracle/anchors.go`, `testdata/wireoracle/transcripts/modern-meta-invalid-params.golden`, `testdata/wireoracle/transcripts/modern-meta-unsupported-version.golden`, this SUMMARY). Both task commits (`5a6bf07`, `8760d2e`) found in git log.
