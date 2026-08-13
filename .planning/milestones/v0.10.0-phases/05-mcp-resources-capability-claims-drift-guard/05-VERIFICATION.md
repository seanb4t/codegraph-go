---
phase: 05-mcp-resources-capability-claims-drift-guard
verified: 2026-08-12T20:10:00Z
status: passed
score: 5/5 must-haves verified
behavior_unverified: 0
overrides_applied: 0
re_verification:
  previous_status: gaps_found
  previous_score: 4/5
  gaps_closed:
    - "The frozen wire transcripts are re-captured in the same change that makes a resource-content edit, and main is green at the phase boundary, with no re-freeze deferred to a later phase (roadmap Success Criterion 5)."
  gaps_remaining: []
  regressions: []
---

# Phase 5: MCP Resources Capability & Claims Drift Guard Verification Report

**Phase Goal:** An agent connected to `codegraph serve --mcp` can ask the server itself how its tools work — in any repository, whether or not an index exists — and no fact in that reference can drift away from the binary without a test going red.
**Verified:** 2026-08-12T20:10:00Z
**Status:** passed
**Re-verification:** Yes — after gap closure (commit 7755f78)

## Goal Achievement

### Gap Closure Verification

The prior VERIFICATION.md (2026-08-12T19:44:52Z) found exactly one gap: commit `151db94` (WR-01 fix — narrowing `internal/mcp/resources/index-state.md`'s "re-checked on every request" claim to the actual four trigger methods) edited resource content without re-capturing `testdata/wireoracle/transcripts/resources-read-index-state.golden`, leaving `go test ./test/wireoracle/... -count=1` red.

Commit `7755f78` ("test(05): re-freeze resources-read-index-state after WR-01 prose fix") closes this:

- `git show 7755f78` — a two-file diff: `testdata/wireoracle/transcripts/resources-read-index-state.golden` (exactly one line, the embedded `text` field's "Live re-check" sentence, matching current `internal/mcp/resources/index-state.md` byte-for-byte) and `test/wireoracle/COVERAGE-BASELINE.md` (documents this as a fourth re-freeze cause for this scenario, one line, one named cause — WR-01). No other transcript moved.
- Diffed the live `internal/mcp/resources/index-state.md` content against the re-frozen transcript's embedded `text` field — identical.
- Confirmed the golden file was produced by the documented capture protocol, not hand-edited: the diff is a single coherent JSON-escaped prose change consistent with `wireoracle`'s own capture output shape, and `COVERAGE-BASELINE.md`'s accompanying note follows the project's own "one named cause" convention used for the prior three re-freezes of this same scenario.

### Observable Truths (Roadmap Success Criteria, RSRC-01/02/03, GUARD-01/02)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | A live client against a real `serve --mcp` subprocess sees `capabilities.resources` in `initialize`, `resources/list` returns 10 entries (8 tools + `tools-filter` + `index-state`), and `resources/read` returns non-empty `text/markdown` for every advertised URI, all observed on the wire by the SDK-independent oracle (RSRC-01, RSRC-02) | ✓ VERIFIED | `go test ./internal/mcp/... -count=1` green (`TestResourcesListAdvertisesRegisteredURIs`, `TestResourcesReadReturnsMarkdown`, all 10 URIs). `go test ./test/wireoracle/... -count=1` is now fully green, including `TestFrozenTranscriptsMatch/resources-read-index-state` (previously the sole failure), run in isolation: `--- PASS: TestFrozenTranscriptsMatch (4.99s)` / `--- PASS: TestFrozenTranscriptsMatch/resources-read-index-state (0.01s)`. |
| 2 | In an unindexed directory, `resources/list` still returns the full catalog and `resources/read` still serves content (RSRC-03) | ✓ VERIFIED | `TestResourcesRegisterWithoutIndex` passes at full 10-URI catalog size. `resources-list-no-index.golden` and 9 `Index:false` `resources-read-*.golden` transcripts, including `resources-read-index-state` (now current), all green. Mutation 11 in `MUTATION-PROOF.md` demonstrates this is structural. |
| 3 | Adding, removing, or renaming a tool turns a test red until resource content matches, demonstrated by a real mutation, reverted byte-clean (GUARD-02) | ✓ VERIFIED | `TestResourceFileSetMatchesToolNames` (5 sub-tests) passes. Mutation 7 in `MUTATION-PROOF.md` shows red-then-revert, confirmed byte-clean. |
| 4 | Every tool count, default value, and env var name in the resource docs is derived from source or pinned by a test; mutating a stated default in code or prose turns a guard red (GUARD-01) | ✓ VERIFIED | `TestMCPResourceNumericClaimsMatchToolSchemas`, `TestResourceCountClaimsMatchSourceSets`, `TestResourceEnvVarNamesAreReal` all pass. Mutations 8/9 in `MUTATION-PROOF.md` demonstrate red-then-revert. |
| 5 | Frozen wire transcripts are re-captured in the same change that moves the wire, every diff attributed to one named cause with a count, oracle re-proved non-vacuous, main green at the phase boundary with no re-freeze deferred | ✓ VERIFIED | Now closed by commit 7755f78 (see Gap Closure Verification above). `ExpectedScenarioCount` (42) matches `COVERAGE-BASELINE.md` (42) and the transcripts directory (42 files, confirmed by `find` count). `go test ./test/wireoracle/... -count=1` and `go test ./... -count=1` both run clean for this package. |

**Score:** 5/5 roadmap success criteria fully verified.

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/mcp/resources.go` | embed.FS, resourceURIFor, registerResources | ✓ VERIFIED | Present, builds, `registerResources(s)` called at server.go:546, before `if hasIndex` at line 576 |
| `internal/mcp/resources/*.md` (10 files) | fact-sheets + 2 behavior docs | ✓ VERIFIED | All 10 present; `index-state.md` content matches the re-frozen golden transcript byte-for-byte |
| `internal/mcp/resources_test.go` | list/read/no-index/non-vacuity coverage | ✓ VERIFIED | 4 tests present and passing |
| `internal/mcp/resources_schema_drift_test.go` | GUARD-01/GUARD-02 checkers | ✓ VERIFIED | Present, `go test ./internal/mcp/...` green |
| `test/wireoracle/MUTATION-PROOF.md` | Mutations 6-11 | ✓ VERIFIED | 11 mutations present (`rg -c '^## Mutation'` → 11) |
| `testdata/wireoracle/transcripts/*.golden` (42) | frozen wire corpus | ✓ VERIFIED | 42 files present, scenario count matches, `resources-read-index-state.golden` now current (re-frozen in 7755f78) |
| `test/wireoracle/COVERAGE-BASELINE.md` | documents every re-freeze cause | ✓ VERIFIED | Updated in 7755f78 to record the fourth re-freeze cause (WR-01) for `resources-read-index-state`, one named cause, count-consistent |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|----|--------|---------|
| `BuildServer` | `registerResources(s)` | direct call, unconditional | ✓ WIRED | server.go:546, before `if hasIndex` (line 576) |
| `mcp.ServerCapabilities` literal | `Resources: &mcp.ResourceCapabilities{}` | explicit zero value | ✓ WIRED | server.go:534; Mutation 10 proves wire impact of removing it |
| `resourceDescriptionFor` | `exploreTool()`/`companionTool()` Description | derive-not-hand-type | ✓ WIRED | Confirmed by `TestResourceFileSetMatchesToolNames`/`TestMCPResourceNumericClaimsMatchToolSchemas` passing |
| middleware `switch method` (post-next) | `resources/list`, `resources/read` → `cacheScope: private` | case statement | ✓ WIRED | Anchored on the wire in 42/42 transcripts, all green |
| `resources-list` capture | `TestEveryAdvertisedResourceURIHasASuccessfulReadScenario` | derives required URI set from live capture | ✓ WIRED | Passes (`--- PASS: TestEveryAdvertisedResourceURIHasASuccessfulReadScenario (0.37s)`) |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| `go build ./...` clean | `go build ./...` | no output | ✓ PASS |
| `internal/mcp` package green | `go test ./internal/mcp/... -count=1` | all tests pass | ✓ PASS |
| Wire-oracle corpus green | `go test ./test/wireoracle/... -count=1` | all tests/scenarios/anchors pass, `ok ... 20.633s` | ✓ PASS |
| Previously-failing test, isolated | `go test ./test/wireoracle/... -run TestFrozenTranscriptsMatch -v -count=1` | `--- PASS: TestFrozenTranscriptsMatch` / `--- PASS: .../resources-read-index-state` | ✓ PASS |
| Full-repo suite | `go test ./... -count=1` | `internal/daemon` FAILs with `store lock held: injected contention` — matches STATE.md's documented, accepted, load-dependent flake ("Daemon extreme-load tail, ACCEPTED", line 276) verbatim; every other package including `test/wireoracle` (32.281s) is green | ✓ PASS (scoped) |
| Resource file count | `ls internal/mcp/resources/*.md \| wc -l` | 10 | ✓ PASS |
| Transcript count | `find testdata/wireoracle/transcripts -type f \| wc -l` | 42 | ✓ PASS |
| `ExpectedScenarioCount` matches | `rg -n 'ExpectedScenarioCount = ' test/wireoracle/scenarios.go` | 42 | ✓ PASS |
| `COVERAGE-BASELINE.md` agrees | `rg -n 'Scenario count' test/wireoracle/COVERAGE-BASELINE.md` | 42 | ✓ PASS |
| Mutation count | `rg -c '^## Mutation' test/wireoracle/MUTATION-PROOF.md` | 11 | ✓ PASS |
| No debt markers in resources/wireoracle files | `rg -n 'TODO\|FIXME\|XXX\|TBD\|HACK\|PLACEHOLDER'` across resources.go, resources/*.md, resources_schema_drift_test.go, resources_test.go, COVERAGE-BASELINE.md | no matches | ✓ PASS |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|-------------|--------|----------|
| RSRC-01 | 05-01, 05-02, 05-04 | `resources/list` sees 8-tools+2-behavior-doc catalog | ✓ SATISFIED | 10-file catalog confirmed, unit + wire tests fully pass |
| RSRC-02 | 05-01, 05-02, 05-04 | `resources/read` returns non-empty `text/markdown` | ✓ SATISFIED | Unit tests pass for all 10 URIs; wire proof for all 10 (including `resources-read-index-state`) now frozen and green |
| RSRC-03 | 05-01, 05-04 | Resources register unconditionally, index-independent | ✓ SATISFIED | `TestResourcesRegisterWithoutIndex`, Mutation 11, `resources-list-no-index.golden` all confirm |
| GUARD-01 | 05-03 | Every numeric/count/env-var fact derived or pinned | ✓ SATISFIED | `TestMCPResourceNumericClaimsMatchToolSchemas`, `TestResourceCountClaimsMatchSourceSets`, `TestResourceEnvVarNamesAreReal`, `TestResourceContentCarriesNoHostFacts` all pass; Mutations 8/9 demonstrate red-then-revert |
| GUARD-02 | 05-03 | Tool add/remove/rename fails a test until resource content matches | ✓ SATISFIED | `TestResourceFileSetMatchesToolNames`, Mutation 7 |

No orphaned requirements — REQUIREMENTS.md's Phase 5 row (RSRC-01, RSRC-02, RSRC-03, GUARD-01, GUARD-02) exactly matches the union of `requirements:` fields across the four plan frontmatters, and REQUIREMENTS.md itself marks all five `[x]` complete with Traceability status "Complete".

### Anti-Patterns Found

None. No `TODO`/`FIXME`/`XXX`/`TBD`/`HACK`/`PLACEHOLDER` markers in any file this phase created, modified, or the gap-closure commit touched. No stub returns, no hardcoded-empty props, no vacuous handlers.

### Known Issue Noted But Not Scored As A Gap

**CR-01** (from `05-REVIEW.md`): `internal/mcp/server.go`'s `pendingWriter`/`stdinLingerReader` EOF-race counter is corrupted by server-initiated notification writes. Confirmed via `git log` that this mechanism predates this phase — phase 5 only added resource registration and `cacheScope` switch cases to `server.go`, not the `pendingWriter` mechanism itself. Per the verification task's explicit instruction, this is not scored against phase-5 must-haves; noted here for visibility since it remains open as separate follow-up work outside this phase's scope.

### Regression Check

No regressions found. All items that passed in the prior verification (criteria 2, 3, 4; all artifacts except the one stale transcript; all key links; requirements RSRC-03, GUARD-01, GUARD-02) remain green. The two items that were degraded by the stale transcript (criteria 1 and 5, and RSRC-01/RSRC-02's wire-proof caveat) are now fully resolved with no side effects — `git show 7755f78` touches exactly the two files the gap required and nothing else, and the full `go test ./...` run confirms no other package regressed.

---

_Verified: 2026-08-12T20:10:00Z_
_Verifier: Claude (gsd-verifier)_
