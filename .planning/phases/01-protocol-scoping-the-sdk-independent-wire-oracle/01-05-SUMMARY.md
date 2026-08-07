---
phase: 01-protocol-scoping-the-sdk-independent-wire-oracle
plan: 05
subsystem: testing
tags: [mcp, jsonrpc, stdio, wire-protocol, golden-transcript, mark3labs-mcp-go, protocol-version, legacy-compatibility]

# Dependency graph
requires:
  - phase: 01-03
    provides: "internal/mcp/archtest's VRFY-02 guard (go test ./internal/mcp/archtest/... -count=1), confirmed still green after this plan's additions"
  - phase: 01-04
    provides: "test/wireoracle/scenarios.go's 17-scenario suite, anchors.go's Anchor infrastructure, and initializeRequest/toolsListRequest helpers this plan reuses"
provides:
  - "Six frozen pre-migration transcripts (testdata/wireoracle/transcripts/legacy-*.golden) — the multi-era Legacy handshake baseline (four supported revisions, one unsupported, one omitted-version) — bringing the suite to exactly 23 scenarios"
  - "TestLegacyEraBaselineIsDocumented — a positive assertion over the frozen era evidence, read from disk, never re-captured"
  - "docs/MCP-2026-07-28-SCOPING.md § Measured pre-migration behavior — the dated, sourced record that today's server silently coerces rather than rejects an unsupported/omitted protocolVersion"
  - "Scenario.EraScenario/EraOfferedVersion/EraNegotiatedVersion (test/wireoracle/capture.go) — the fields any future era-shaped scenario keys its expected offered/negotiated pair off"
affects: [02-sdk-migration, 03-legacy-compatibility-tests, 07-mutation-matrix]

# Actuals (#2632)
actuals:
  tokens: 7765
  tasks: 2
  commits: 1

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "assertSessionLine/assertProtocolVersionAnchor take explicit want-value parameters instead of hardcoding internal/mcp.ProtocolVersion — every one of the 17 pre-existing scenarios passes internal/mcp.ProtocolVersion at both call sites (unchanged behavior), while the six era scenarios pass their own EraOfferedVersion/EraNegotiatedVersion. A scenario whose offered protocol revision differs from the repo-owned pin is now a first-class, structurally distinct case rather than an assumption baked into the assertion helpers."
    - "sanitizedRequestedVersion() mirrors internal/mcp.sanitizeClientField's documented empty-string-to-\"<unknown>\" behavior in the test package, since the production function is unexported — a deliberate, documented duplicate of a published contract, same pattern as the existing codegraphSessionLinePrefix constant."

key-files:
  created:
    - testdata/wireoracle/transcripts/legacy-2025-11-25.golden
    - testdata/wireoracle/transcripts/legacy-2025-06-18.golden
    - testdata/wireoracle/transcripts/legacy-2025-03-26.golden
    - testdata/wireoracle/transcripts/legacy-2024-11-05.golden
    - testdata/wireoracle/transcripts/legacy-unsupported-2026-07-28.golden
    - testdata/wireoracle/transcripts/legacy-omitted-version.golden
  modified:
    - test/wireoracle/scenarios.go
    - test/wireoracle/capture.go
    - test/wireoracle/oracle_test.go
    - docs/MCP-2026-07-28-SCOPING.md

key-decisions:
  - "Task 1 blocking checkpoint (resumed from a prior halt): human selected six-era — the four D-06 revisions plus 2026-07-28 (unsupported) plus a handshake omitting protocolVersion entirely. 'more' and 'five-era' were not selected. Six scenarios added, phase total goes from 17 to 23."
  - "Deviation (Rule 1 — bug): oracle_test.go's assertSessionLine and assertProtocolVersionAnchor hardcoded internal/mcp.ProtocolVersion as the expected requested/negotiated value for every scenario, an assumption valid only because all 17 pre-existing scenarios happen to offer that same literal. The six era scenarios deliberately offer other revisions, so both helpers were parameterized with explicit want values (defaulting to internal/mcp.ProtocolVersion at the 17 existing call sites, using each era scenario's own EraOfferedVersion/EraNegotiatedVersion at the six new ones) rather than special-casing scenario names."
  - "Deviation (Rule 1 — bug, discovered during first verify run): legacy-omitted-version's session line reports requested=<unknown>, not requested= — internal/mcp.sanitizeClientField (session_line.go) converts an empty client-supplied field to the literal \"<unknown>\" before formatting, a T-01-01 injection defense. Added sanitizedRequestedVersion() to the test package (a documented duplicate of the unexported production behavior) rather than weakening the production sanitizer or hand-waving the assertion."
  - "legacy-unsupported-2026-07-28 and legacy-omitted-version carry no error-code anchor in anchors.go (unchanged) — both are frozen as the SUCCESSFUL initialize results they measured, per RESEARCH Pitfall 1; asserting an error there would assert a behavior that never fires."

patterns-established:
  - "Era-shaped scenarios (offering a protocol revision other than the repo-owned pin) declare EraScenario: true and carry their own EraOfferedVersion/EraNegotiatedVersion pair; TestFrozenTranscriptsMatch branches on this field to select the correct expected-value pair for the D-02 spec anchor and VRFY-03 session-line check."
  - "initializeRequestWithVersion(id, version) and initializeRequestOmittingVersion(id) in scenarios.go give any future era-shaped scenario a ready request-shape constructor without re-deriving the omitted-vs-empty-string distinction."

requirements-completed: [VRFY-01, VRFY-04]

coverage:
  - id: D1
    description: "Six-era Legacy handshake baseline frozen against the pre-migration mark3labs v0.56.0 binary — four supported revisions echoed back verbatim, one unsupported revision silently coerced to the server's own latest (success, no error), one omitted-version request silently coerced to the server's older backwards-compat default (success, no error) — suite at exactly 23 scenarios"
    requirement: "VRFY-04"
    verification:
      - kind: unit
        ref: "test/wireoracle/oracle_test.go#TestFrozenTranscriptsMatch"
        status: pass
      - kind: unit
        ref: "test/wireoracle/oracle_test.go#TestLegacyEraBaselineIsDocumented"
        status: pass
      - kind: unit
        ref: "test/wireoracle/oracle_test.go#TestSpecAnchorsHold"
        status: pass
    human_judgment: false
  - id: D2
    description: "No SDK-owned protocol-version constant crept into the era matrix — every offered revision is a hand-authored literal in scenarios.go"
    requirement: "VRFY-01"
    verification:
      - kind: unit
        ref: "internal/mcp/archtest/protocol_version_test.go#TestNoExternalProtocolVersionConstantReferences"
        status: pass
    human_judgment: false
  - id: D3
    description: "docs/MCP-2026-07-28-SCOPING.md records the measured silent-coercion behavior, dated, pointing at the six frozen transcripts, cross-referenced against the error-code allocation policy row Phase 3 must satisfy with -32022"
    verification: []
    human_judgment: true
    rationale: "Documentation content quality (accuracy of the cross-reference, dated claim) is not mechanically verifiable beyond the file existing and containing the required heading — a human should confirm the prose reads correctly."

# Metrics
duration: ~35min (continuation from Task 1's checkpoint halt; timing covers only this continuation's work)
completed: 2026-08-05
status: complete
---

# Phase 1 Plan 05: Six-Era Legacy Handshake Baseline Summary

**Froze six pre-migration transcripts (four supported protocol revisions plus one unsupported plus one omitted) proving mark3labs v0.56.0 silently coerces rather than rejects, with a parameterized D-02 spec anchor so the assertion machinery no longer assumes every scenario shares one protocol-version literal.**

## Performance

- **Duration:** ~35 min (this continuation; Task 1's original checkpoint halt happened in a prior, unbilled session)
- **Completed:** 2026-08-05T21:03:42Z
- **Tasks:** 2 (Task 1 closed on the human's recorded `six-era` selection; Task 2 executed and committed)
- **Files modified:** 4 modified, 6 created

## Accomplishments

- Task 1 (`checkpoint:decision`, gate=`blocking`) closed on the human's `six-era` selection: capture all four protocol revisions today's server recognizes, plus `2026-07-28` (unsupported), plus a handshake omitting `protocolVersion` entirely.
- Six new scenarios added to `Scenarios()` (`legacy-2025-11-25`, `legacy-2025-06-18`, `legacy-2025-03-26`, `legacy-2024-11-05`, `legacy-unsupported-2026-07-28`, `legacy-omitted-version`), suite goes from 17 to **exactly 23**.
- Six transcripts frozen via the `wireoracle` CLI redirect (D-03 — no in-suite write path), one per era scenario.
- `TestLegacyEraBaselineIsDocumented` added: a positive assertion over the six frozen transcripts (read from disk, never re-captured) checking the era-scenario count is exactly 6 and that the four supported revisions negotiate to themselves.
- `docs/MCP-2026-07-28-SCOPING.md` gets a dated **Measured pre-migration behavior** subsection with the per-era offered/negotiated table and a cross-reference to the error-code allocation policy row Phase 3 must satisfy.
- `go test ./internal/mcp/archtest/... -count=1` still exits 0 — no SDK-owned protocol-version constant crept into the era matrix.

### Measured offered vs. negotiated (per era, as required by the plan's verification step)

| Scenario | Offered | Negotiated | Result |
|---|---|---|---|
| `legacy-2025-11-25` | `2025-11-25` | `2025-11-25` | supported, echoed back |
| `legacy-2025-06-18` | `2025-06-18` | `2025-06-18` | supported, echoed back |
| `legacy-2025-03-26` | `2025-03-26` | `2025-03-26` | supported, echoed back |
| `legacy-2024-11-05` | `2024-11-05` | `2024-11-05` | supported, echoed back |
| `legacy-unsupported-2026-07-28` | `2026-07-28` | `2025-11-25` | **silent coercion to server's own latest — SUCCESS, no `error` object** |
| `legacy-omitted-version` | *(no key)* | `2025-03-26` | **silent coercion to server's older backwards-compat default — SUCCESS, distinct from the row above** |

Every negotiated value matched the orchestrator's SDK-source-verified predictions exactly on the first capture — no surprise requiring a loud flag per the plan's instructions.

## Task Commits

Task 1 and Task 2 were closed together in one commit (Task 1 made no code changes — it is a decision checkpoint; the human's `six-era` selection is recorded here and in this SUMMARY, not in a separate commit):

1. **Task 1 (checkpoint:decision) + Task 2 (capture and freeze)** - `acdad68` (feat)

_Note: no plan-metadata commit is included in this list — per the objective, this executor does not update STATE.md/ROADMAP.md; the orchestrator owns those writes and the metadata commit that follows._

## Files Created/Modified

- `test/wireoracle/scenarios.go` - six new era scenarios, `legacyEraVersions`/`legacyUnsupportedVersion`/`legacyOmittedVersionCoercion` literals, `initializeRequestWithVersion`/`initializeRequestOmittingVersion` helpers
- `test/wireoracle/capture.go` - `Scenario.EraScenario`/`EraOfferedVersion`/`EraNegotiatedVersion` fields
- `test/wireoracle/oracle_test.go` - parameterized `assertSessionLine`/`assertProtocolVersionAnchor`, `sanitizedRequestedVersion` helper, `TestLegacyEraBaselineIsDocumented` + `decodeFrozenInitializeProtocolVersion`
- `docs/MCP-2026-07-28-SCOPING.md` - dated "Measured pre-migration behavior" subsection
- `testdata/wireoracle/transcripts/legacy-*.golden` (6 files) - the frozen multi-era baseline

## Decisions Made

See `key-decisions` in frontmatter — summarized: the `six-era` checkpoint selection, and two Rule-1 bug fixes to the generalized-but-untested-for-this-case oracle assertion helpers (hardcoded protocol-version literal, and the `<unknown>` sanitization of an empty client-supplied field).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Parameterized `assertSessionLine`/`assertProtocolVersionAnchor` instead of hardcoding `internal/mcp.ProtocolVersion`**
- **Found during:** Task 2, first `go test` run against the new era scenarios
- **Issue:** Both helpers asserted `requested`/`negotiated`/`result.protocolVersion` against the single fixed literal `internal/mcp.ProtocolVersion` — correct for all 17 pre-existing scenarios (which all happen to offer that same value) but wrong by construction for five of the six new era scenarios, which deliberately offer a different revision.
- **Fix:** Both functions now take explicit `want` (or `wantRequested`/`wantNegotiated`) parameters. The 17 existing call sites pass `internal/mcp.ProtocolVersion` (unchanged assertion, identical behavior). The six era scenarios pass their own `EraOfferedVersion`/`EraNegotiatedVersion`.
- **Files modified:** `test/wireoracle/oracle_test.go`
- **Verification:** `go test ./test/wireoracle/... -count=1` — all 23 scenarios pass, including the 17 unaffected ones.
- **Committed in:** `acdad68`

**2. [Rule 1 - Bug] Added `sanitizedRequestedVersion()` for the omitted-version scenario's session line**
- **Found during:** Task 2, after the fix above — `legacy-omitted-version` still failed with `requested="<unknown>", want ""`
- **Issue:** `internal/mcp.sanitizeClientField` (a T-01-01 injection defense) converts an empty client-supplied field to the literal `"<unknown>"` before writing the stderr session line. `EraOfferedVersion` for the omitted-version scenario is correctly `""` (matching the raw wire shape), but the *session line's* `requested=` value is the sanitized `"<unknown>"`, not the raw `""`.
- **Fix:** Added `sanitizedRequestedVersion(offered string) string` to `oracle_test.go`, mirroring the unexported production function's documented empty-string behavior, applied only at the session-line comparison call site (not at `assertProtocolVersionAnchor`, which reads the raw JSON `result.protocolVersion` field and is unaffected).
- **Files modified:** `test/wireoracle/oracle_test.go`
- **Verification:** `go test ./test/wireoracle/... -count=1 -run 'TestFrozenTranscriptsMatch|TestLegacyEraBaselineIsDocumented|TestSpecAnchorsHold' -v` — all green, including `legacy-omitted-version`.
- **Committed in:** `acdad68`

---

**Total deviations:** 2 auto-fixed (both Rule 1 — bugs in the oracle's own assertion helpers, surfaced only once scenarios stopped sharing one hardcoded protocol-version literal)
**Impact on plan:** Both fixes were necessary for the plan's own stated verify command to pass; no scope creep, no change to `internal/mcp` production code, no change to the 17 pre-existing scenarios' expected values.

## Issues Encountered

None beyond the two deviations above, both caught and fixed on the first verification run.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- The suite is at exactly 23 scenarios with 23 frozen transcripts on disk — plan 07 (`ExpectedScenarioCount`, `TestScenarioCountIsExact`) can set its exact-equality assertion to 23 in the same commit that reads this SUMMARY.
- `docs/MCP-2026-07-28-SCOPING.md`'s Measured pre-migration behavior table is the honest pre-migration baseline Phase 3's SPEC-06 (Legacy compatibility) needs to compare against, rather than writing its first multi-era test against an SDK that already claims five-era support.
- No blockers. `go.mod`'s MCP dependency line is unchanged (per this plan's threat register, T-05-SC).
- This SUMMARY does not update `.planning/STATE.md` or `.planning/ROADMAP.md` — per the objective, that is the orchestrator's responsibility after this plan's completion is reported.

---
*Phase: 01-protocol-scoping-the-sdk-independent-wire-oracle*
*Completed: 2026-08-05*

## Self-Check: PASSED

All 11 files created/modified this plan verified present on disk; commit `acdad68` verified present in `git log --oneline --all`.
