---
phase: 08-instructions-marker-block-rewrite
verified: 2026-08-13T21:30:00Z
status: passed
score: 9/9 must-haves verified
behavior_unverified: 0
overrides_applied: 0
---

# Phase 8: Instructions & Marker-Block Rewrite Verification Report

**Phase Goal:** Every statement codegraph makes about itself — on the MCP wire and in the marker block it writes into agent instruction files — is true on the day an agent reads it, and names only capabilities that have already shipped.
**Verified:** 2026-08-13T21:30:00Z
**Status:** passed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths (ROADMAP Success Criteria, cross-referenced with plan must_haves)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | `instructions` string names the skill and the resource surface, contains no forward-looking promise; stale "Phase 3" deferral gone from `internal/agents/instructions.go` (WIRE-01) | VERIFIED | `internal/mcp/server.go:56` const verified by direct read: contains `resources/list` and Claude-Code-scoped `codegraph skill` clause. `rg -o "defers full tool guidance" internal/agents/` → 0 matches (independently re-run). |
| 2 | Marker block `codegraph install` writes matches the rewritten instructions and describes only what that install did; a mutation naming an unshipped capability turns a test red (WIRE-02) | VERIFIED | `internal/agents/instructions.go` `codegraphInstructionsBlock` contains `resources/list`/`resources/read` bullet, shares vocabulary with wire const. `TestInstructionsBlockNamesOnlyShippedCapabilities` PASS (re-run independently); SUMMARY records a live demonstrated-RED mutation (skill sentence appended → test failed, reverted byte-clean). |
| 3 | Every capability either string names is resolvable at test time — named resource URIs resolve via live `resources/read`, named skill path is one `codegraph install` actually writes (WIRE-03) | VERIFIED | `TestInstructionsResourcesClaimIsResolvable` and `TestInstructionsSkillClaimIsResolvable` PASS (re-run independently, no `t.Skip` branches present in source). Both checkers proven non-vacuous by `TestInstructionsClaimGuardsAreNotVacuous` (11 subtests, all PASS, re-run independently). |
| 4 | Frozen wire transcripts carry the new instructions string byte-identically to what a live client receives, re-captured in this change with the diff attributed (WIRE-01) | VERIFIED | `go test ./test/wireoracle/... -count=1` PASS (re-run independently, 24.9s). `rg -l "resources/list" testdata/wireoracle/transcripts/` = 38, `rg -l "optional path argument"` = 0. Commit `329bc0c` message names single cause, 38/4 split, and explicit `toolslist-repeat` inspection outcome. |
| 5 | `session.InitializeResult().Instructions` byte-identical to the `server.go` const (end-to-end wire proof) | VERIFIED | `TestInstructionsReachesTheWireVerbatim` PASS (re-run independently). |
| 6 | No `codegraph://` URI enumerated in the wire const (D-03) | VERIFIED | `rg -n "codegraph://" internal/mcp/server.go` → 0 matches. |
| 7 | Marker fences byte-identical to pre-change values (cross-implementation contract with TS CodeGraph) | VERIFIED | `internal/agents/instructions.go:9-10` — `codegraphSectionStart`/`codegraphSectionEnd` unchanged; `TestInstructionsBlock_ExactMarkerText` PASS. |
| 8 | Marker block names no skill for any of its 4 shared targets (Claude, Codex, opencode, Gemini) (D-04) | VERIFIED | `blockNamesUnshippedCapability` case-insensitive skill check; `TestInstructionsBlockGuardIsNotVacuous` (6 subtests incl. mixed-case row) PASS (re-run independently). |
| 9 | Full downstream test suite is green (no regression introduced by the rewrite) | VERIFIED | `go build ./...`, `go vet ./internal/mcp/... ./internal/agents/...`, `gofmt -l` (empty), `go test ./internal/mcp/... ./internal/agents/... ./test/wireoracle/...` all PASS, re-run independently in this verification session. |

**Score:** 9/9 truths verified (0 present-but-behavior-unverified)

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/mcp/server.go` | Rewritten `instructions` const naming `resources/list` and Claude-Code-scoped skill | VERIFIED | Const value confirmed byte-for-byte (554 bytes, measured independently with Python), single literal, pure ASCII |
| `internal/mcp/instructions_contract_test.go` | WIRE-03 resolvability guards, non-vacuity table test, host-value guard | VERIFIED | All 8 relevant tests present and passing (`TestInstructionsDescribesEveryVisibilityMechanism`, `TestInstructionsResourcesClaimIsResolvable`, `TestInstructionsSkillClaimIsResolvable`, `TestInstructionsCarriesNoWireContractViolation`, `TestInstructionsReachesTheWireVerbatim`, `TestInstructionsClaimGuardsAreNotVacuous`, plus 2 pre-existing) |
| `internal/agents/instructions.go` | Rewritten block body + corrected doc comment | VERIFIED | New "Reference docs" bullet present, doc comment references Pitfall 1, no "Phase 3" deferral text remains |
| `internal/agents/registry_test.go` | `blockNamesUnshippedCapability` checker + non-vacuity test | VERIFIED | `TestInstructionsBlockNamesOnlyShippedCapabilities`, `TestInstructionsBlockGuardIsNotVacuous` present and passing |
| `testdata/wireoracle/transcripts/` | 38 re-frozen golden transcripts | VERIFIED | 38 files contain `resources/list`, 0 contain retired `optional path argument` text; `TestFrozenTranscriptsMatch` green |

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| `instructions_contract_test.go` | `claudeassets.go` | `SkillMarkdown()`/`SkillMarkdownPath` | WIRED | `skillClaimResolves` takes `claudeassets.SkillMarkdown` as a function value; no second hand-typed path found (`rg -v '^\s*//' ... rg -o 'skills/codegraph'` = 0 outside comments, per SUMMARY, spot-checked) |
| `instructions_contract_test.go` | `resources.go` | Live `ListResources`/`ReadResource` round-trip | WIRED | `TestInstructionsResourcesClaimIsResolvable` uses `newTestSession`, confirmed PASS with a real session, not a hardcoded URI list |
| `server.go` | `BuildServer` `ServerOptions.Instructions` | Const carried to both classic initialize and server/discover | WIRED | `TestInstructionsReachesTheWireVerbatim` proves this end-to-end |
| `registry_test.go` | `instructions.go` | `blockNamesUnshippedCapability(codegraphInstructionsBlock)` reads the const value | WIRED | Confirmed: checker signature takes block as parameter, applied to real const in `TestInstructionsBlockNamesOnlyShippedCapabilities` |
| `instructions.go` (agents) | `server.go` (mcp) | Shared capability vocabulary (`resources/list`) | WIRED | Both files independently confirmed to contain `resources/list` |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|-------------|--------|----------|
| WIRE-01 | 08-01, 08-03 | Wire instructions string retires stale promise, points at shipped capabilities; transcripts re-frozen | SATISFIED | Verified truths 1, 4, 5, 6, 9 above |
| WIRE-02 | 08-02 | Marker block matches rewritten instructions, promises nothing unshipped | SATISFIED | Verified truths 2, 7, 8 above |
| WIRE-03 | 08-01, 08-02 | Every named capability resolvable at test time | SATISFIED | Verified truth 3 above |

No orphaned requirements — `.planning/REQUIREMENTS.md` maps exactly WIRE-01/02/03 to Phase 8, all three appear in plan frontmatter `requirements:` fields, and the roadmap phase table confirms "3/3 plans executed."

### Anti-Patterns Found

None. `rg -n "TODO|FIXME|XXX|TBD|HACK|PLACEHOLDER"` against all four phase-touched Go source files returned zero matches. No `t.Skip` branches in any resolvability guard (confirmed by direct inspection of `instructions_contract_test.go` behavior and independent test run — no SKIP lines in verbose output).

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Wire const reaches a live client verbatim | `go test ./internal/mcp/ -run TestInstructionsReachesTheWireVerbatim -v -count=1` | PASS | PASS |
| Resources claim resolves via live round-trip | `go test ./internal/mcp/ -run TestInstructionsResourcesClaimIsResolvable -v -count=1` | PASS | PASS |
| Skill claim resolves to embedded asset | `go test ./internal/mcp/ -run TestInstructionsSkillClaimIsResolvable -v -count=1` | PASS | PASS |
| Marker block honesty guard discriminates | `go test ./internal/agents/ -run TestInstructionsBlockGuardIsNotVacuous -v -count=1` | 6/6 subtests PASS | PASS |
| Full wire-oracle corpus green after re-freeze | `go test ./test/wireoracle/... -count=1` | PASS (24.9s) | PASS |
| No regression in touched packages | `go test ./internal/mcp/... ./internal/agents/... -count=1` | PASS | PASS |
| Build/vet/fmt clean | `go build ./...`, `go vet ./internal/mcp/... ./internal/agents/...`, `gofmt -l` | all clean | PASS |

### Probe Execution

Not applicable — this phase is a wire-contract/const rewrite with unit and golden-transcript test coverage, not a probe-based migration/tooling phase. No `scripts/*/tests/probe-*.sh` files referenced by PLAN/SUMMARY/REVIEW.

### Human Verification Required

None. All must-haves are mechanically verifiable (const content, byte counts, test pass/fail, golden-file diffs) and were independently re-verified by running the actual test suite and reading actual file contents in this session, not by trusting SUMMARY.md claims.

### Gaps Summary

No gaps. All 9 observable truths derived from ROADMAP.md success criteria and PLAN must_haves are VERIFIED against the actual codebase:

- Independently re-read `internal/mcp/server.go`'s `instructions` const and confirmed its byte length (554) and content match the SUMMARY's claims exactly.
- Independently re-read `internal/agents/instructions.go` and confirmed the marker block body, fences, and doc comment match the SUMMARY's claims.
- Independently re-ran (not trusted) `go build ./...`, `go vet`, `gofmt -l`, and the full `internal/mcp`, `internal/agents`, and `test/wireoracle` test suites — all green.
- Independently confirmed the stale "defers full tool guidance" deferral text is fully absent from `internal/agents/`.
- Independently confirmed no `codegraph://` URI is enumerated in the wire const (D-03) and marker fences are byte-unchanged.
- Independently confirmed all 38 re-frozen transcripts carry `resources/list` and none carry the retired `optional path argument` text.
- Independently confirmed no phase-8 commit touches `internal/daemon`, corroborating the orchestration note that `TestConvergenceTwoSessions`'s flake is unrelated to this phase's goal.
- 08-REVIEW.md's two warnings (WR-01 pre-existing `toolslist-repeat` flake reproduced at both HEAD and pre-phase-8 base commit `fa87a7a`; WR-02 incomplete absolute-path denylist, defense-in-depth only) are both correctly framed as pre-existing/low-severity and not phase regressions — confirmed by re-reading the review's own reproduction evidence, which is not itself something this verifier could independently re-run without seeding additional flakiness, but the review's methodology (base-commit comparison) is sound on its face and the phase's own commit history shows zero touches to the response-dispatch code path WR-01 implicates.

---

_Verified: 2026-08-13T21:30:00Z_
_Verifier: Claude (gsd-verifier)_
