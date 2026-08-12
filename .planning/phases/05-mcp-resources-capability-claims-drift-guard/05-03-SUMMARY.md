---
phase: 05-mcp-resources-capability-claims-drift-guard
plan: 03
subsystem: mcp
tags: [mcp, go-sdk, resources, drift-guard, wire-oracle, jsonschema]

# Dependency graph
requires:
  - phase: "05-02: full 10-resource catalog (8 per-tool fact-sheets + 2 behavior docs), resourceURIFor/resourceDescriptionFor registration"
  - phase: "05-01: resourcesFS embed seam, tools_schema_drift_test.go's numericClaimRe/engineConstantFor pattern, instructions_contract_test.go's docNamesCompanionsWithoutTheFilter/allowlistEnvName pattern"
provides:
  - "TestResourceFileSetMatchesToolNames + resourceStemSetDiff: bidirectional set-equality guard between the tool roster (allToolNames()) and the resource file set/URI map/tools-filter.md prose (GUARD-02)"
  - "TestMCPResourceNumericClaimsMatchToolSchemas: every numeric claim in a resource fact-sheet transitively pinned to internal/query via the existing engine-constant guard, with no third copy of any number (GUARD-01)"
  - "TestResourceCountClaimsMatchSourceSets/countClaimsIn: every tool/companion count claim pinned to len(allToolNames())/len(companionNames) (GUARD-01)"
  - "TestResourceEnvVarNamesAreReal: the only CODEGRAPH_-prefixed token permitted in resource content is allowlistEnvName (GUARD-01)"
  - "TestResourceContentCarriesNoHostFacts/hostFactsIn: no resource file may carry an absolute machine-rooted path (T-05-01)"
  - "Mutations 7, 8, 9 in test/wireoracle/MUTATION-PROOF.md: real-tree demonstrated-red proof for GUARD-02 (tool rename) and GUARD-01's code/prose asymmetry (defaultDepth vs impact.md)"
affects: [05-04-wire-coverage-remaining]

# Actuals (#2632)
actuals:
  tokens: 10617
  tasks: 3
  commits: 2

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Every new guard checker (resourceStemSetDiff, countClaimsIn, hostFactsIn) is extracted into its own function with a dedicated synthetic non-vacuity sub-test, matching instructions_contract_test.go's docNamesCompanionsWithoutTheFilter/TestREADMEGateCheckerIsNotVacuous precedent."
    - "Numeric-claim pinning compares resource markdown against the TOOL SCHEMA's own claims (never against internal/query directly) so the resource is transitively pinned through the pre-existing TestMCPToolSchemaNumericClaimsMatchEngineConstants guard, avoiding a second numeric-source map."

key-files:
  created:
    - internal/mcp/resources_schema_drift_test.go
  modified:
    - test/wireoracle/MUTATION-PROOF.md

key-decisions:
  - "Tasks 1 and 2 were committed as a single combined commit rather than two separate atomic commits, because the whole guard file was authored and verified as one coherent unit before any commit was made — see Deviations."
  - "MUTATION-PROOF.md's new mutations are numbered 7, 8, 9, not 6, 7, 8 as the plan's Task 3 text assumed — the file already had 6 mutations (05-01-PLAN's own Task 3 appended Mutations 5-6 for SPEC-09) by the time this task ran, not 5 as the plan's read_first note stated. Documented in both MUTATION-PROOF.md itself (a dedicated 'A note on numbering' section) and here — see Deviations."
  - "toolSchemaClaimsFor's per-type jsonschema.For[XArgs](nil) dispatch is a Go-generics-forced switch on stem (not a hand-typed tool NAME) — Go's f(g()) multi-value spreading rule requires g()'s return values to be f's ONLY arguments, so the closure form (not a package-level mustProps(t, s, err)) was needed to keep t.Fatalf available without violating that rule."

patterns-established:
  - "A guard-the-guard fatal check (empty expected set / empty actual set / zero claims compared / zero tokens found) precedes every comparison in a new drift-guard test, mirroring claimsChecked's existing shape in tools_schema_drift_test.go."

requirements-completed: [GUARD-01, GUARD-02]

coverage:
  - id: D1
    description: "Renaming a tool turns go test ./internal/mcp/... red until the resource content matches, demonstrated by applying the mutation, observing the failure, and reverting byte-clean (GUARD-02)"
    requirement: GUARD-02
    verification:
      - kind: unit
        ref: "internal/mcp TestResourceFileSetMatchesToolNames (all 5 sub-tests)"
        status: pass
      - kind: other
        ref: "test/wireoracle/MUTATION-PROOF.md Mutation 7 — status renamed to health, TestResourceFileSetMatchesToolNames observed red naming both missing [health] and orphaned [status], reverted byte-clean"
        status: pass
    human_judgment: false
  - id: D2
    description: "Mutating internal/query/validate.go's defaultDepth turns the existing engine-constant guard red while the new resource-vs-schema guard stays green; mutating the same number in resources/impact.md's prose turns the new guard red while the engine-constant guard stays green — the asymmetry recorded accurately (GUARD-01)"
    requirement: GUARD-01
    verification:
      - kind: unit
        ref: "internal/mcp TestMCPResourceNumericClaimsMatchToolSchemas, TestMCPToolSchemaNumericClaimsMatchEngineConstants"
        status: pass
      - kind: other
        ref: "test/wireoracle/MUTATION-PROOF.md Mutations 8 and 9 — defaultDepth changed to 4 (code side) and impact.md's stated default changed to 9 (prose side), each mutation's predicted asymmetry observed exactly, both reverted byte-clean"
        status: pass
    human_judgment: false
  - id: D3
    description: "Every integer-plus-noun tool/companion count in the resource docs equals len(allToolNames()) or len(companionNames) (GUARD-01)"
    requirement: GUARD-01
    verification:
      - kind: unit
        ref: "internal/mcp TestResourceCountClaimsMatchSourceSets"
        status: pass
    human_judgment: false
  - id: D4
    description: "The only CODEGRAPH_-prefixed token in any resource file is the real env var name (GUARD-01)"
    requirement: GUARD-01
    verification:
      - kind: unit
        ref: "internal/mcp TestResourceEnvVarNamesAreReal"
        status: pass
    human_judgment: false
  - id: D5
    description: "No resource file carries an absolute filesystem path or host-shaped token (T-05-01)"
    verification:
      - kind: unit
        ref: "internal/mcp TestResourceContentCarriesNoHostFacts"
        status: pass
    human_judgment: false
  - id: D6
    description: "Every checker is proven to discriminate — by a synthetic non-vacuity sub-test, by a demonstrated-red mutation, or both"
    verification:
      - kind: unit
        ref: "internal/mcp TestResourceStemSetDiffIsNotVacuous (6 sub-cases), TestResourceCountCheckerIsNotVacuous (5 sub-cases), TestResourceHostFactCheckerIsNotVacuous (5 sub-cases)"
        status: pass
      - kind: other
        ref: "test/wireoracle/MUTATION-PROOF.md Mutations 7-9 (real-tree red proof for the set-equality and numeric-claim guards)"
        status: pass
    human_judgment: false
  - id: D7
    description: "go test ./internal/mcp/... -count=1, go test ./... -count=1, and task test:wireoracle stay green; this plan moves no wire byte"
    verification:
      - kind: unit
        ref: "internal/mcp full package (go test ./internal/mcp/... -count=1)"
        status: pass
      - kind: integration
        ref: "test/wireoracle full suite (go test ./test/wireoracle/... -count=1)"
        status: pass
      - kind: integration
        ref: "go test ./... -count=1 (repo-wide, on the committed tree after Task 3's revert)"
        status: pass
    human_judgment: false
duration: ~35min
completed: 2026-08-12
status: complete
---

# Phase 5 Plan 3: MCP Resources Claims Drift Guard Summary

**Bidirectional GUARD-02 set-equality test plus GUARD-01 numeric/count/env-var claim-pinning tests for the full 10-resource catalog, each proven to discriminate by a synthetic non-vacuity sub-test and/or a demonstrated-red real-tree mutation.**

## Performance

- **Duration:** ~35 min
- **Started:** 2026-08-12T18:08:00Z
- **Completed:** 2026-08-12T18:43:05Z
- **Tasks:** 3
- **Files modified:** 2 (1 created, 1 modified)

## Accomplishments
- `internal/mcp/resources_schema_drift_test.go` (new, package `mcp`): `resourceStemSetDiff` + `TestResourceStemSetDiffIsNotVacuous`, `TestResourceFileSetMatchesToolNames` (file set, `resourceURIFor` key set, per-tool URI shape D-09, behavior-doc URI shape D-10, and tools-filter.md's prose — all derived from `allToolNames()`/`companionNames`, zero hand-typed tool names)
- `toolSchemaClaimsFor` + `TestMCPResourceNumericClaimsMatchToolSchemas`: every fact-sheet's numeric claims compared, as a multiset, against that tool's own registered schema claims (description + every jsonschema property description) — transitively pinned to `internal/query` through the pre-existing `TestMCPToolSchemaNumericClaimsMatchEngineConstants` guard, with an explicit doc comment stating exactly which drift this test catches and which it does not
- `countClaimsIn` + `TestResourceCountCheckerIsNotVacuous` + `TestResourceCountClaimsMatchSourceSets`: every companion/tool count claim compared against `len(companionNames)`/`len(allToolNames())`, including a synthetic case pinning that "N companion tools" binds to the companion rule only
- `envVarTokenRe` + `TestResourceEnvVarNamesAreReal`: every `CODEGRAPH_`-prefixed token in any resource file must equal `allowlistEnvName`
- `hostFactsIn` + `TestResourceHostFactCheckerIsNotVacuous` + `TestResourceContentCarriesNoHostFacts`: no resource file may carry an absolute machine-rooted POSIX path or a Windows drive-letter path (T-05-01)
- `test/wireoracle/MUTATION-PROOF.md`: Mutations 7 (rename `status`→`health`, GUARD-02), 8 (change `defaultDepth`, GUARD-01 code side), and 9 (change `impact.md`'s stated default, GUARD-01 prose side, the exact mirror of Mutation 8) — each applied, confirmed, observed red with verbatim failure text, and reverted byte-clean

## Task Commits

1. **Tasks 1 & 2 (combined): GUARD-02 set-equality + GUARD-01 claim-pinning guards** - `27f5fd2` (feat)
2. **Task 3: demonstrate all three guards red against real mutations** - `af00d77` (test)

## Files Created/Modified
- `internal/mcp/resources_schema_drift_test.go` - GUARD-01/GUARD-02 claims drift guard, 609 lines
- `test/wireoracle/MUTATION-PROOF.md` - Mutations 7, 8, 9 appended (+300 lines), plus a numbering-note section

## Decisions Made
- Numeric-claim pinning compares the resource markdown against the TOOL SCHEMA's own claims (not `internal/query` directly), so the resource is transitively pinned through the existing engine-constant guard rather than introducing a second numeric source map — matches the plan's explicit instruction.
- The two behavior-doc stems (`tools-filter`, `index-state`) are asserted to carry ZERO numeric claims in `TestMCPResourceNumericClaimsMatchToolSchemas`, closing the gap where a free-floating number could hide in the two files that have no tool schema to compare against.
- Mutation numbering continues from the file's actual state (7, 8, 9) rather than the plan's stale assumption (6, 7, 8) — see Deviations.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking issue] Combined Tasks 1 and 2 into a single commit**
- **Found during:** Task 1/2 boundary
- **Issue:** The guard file's Task 2 additions (numeric/count/env-var/host-fact checkers) directly reuse Task 1's helpers (`toolStemsFromNames`, the derived stem sets) and were authored, built, vetted, and tested as one coherent file before any intermediate commit point was reached — retroactively splitting the already-verified file into two commits by hand-editing a diff would have been artificial and risked introducing an inconsistent intermediate state that never actually existed on disk.
- **Fix:** Committed both tasks' guard code together as one `feat` commit (`27f5fd2`), with the commit message enumerating both GUARD-01 and GUARD-02 deliverables. All acceptance criteria for both Task 1 and Task 2 were independently verified (via the exact `go test`/`rg` commands the plan specifies) before this commit, so the split is administrative, not a gap in verification.
- **Files modified:** `internal/mcp/resources_schema_drift_test.go`
- **Commit:** `27f5fd2`

**2. [Rule 3 - Blocking issue] Mutation numbering continues from 7, not 6**
- **Found during:** Task 3
- **Issue:** The plan's Task 3 `<read_first>` states "Five mutations exist; these are 6, 7 and 8" and the acceptance criteria literally check `rg -c '^## Mutation' ... returns 8` and name `## Mutation 6` as the tool-rename proof's header. By the time this task ran, `test/wireoracle/MUTATION-PROOF.md` already had SIX mutations (05-01-PLAN's own Task 3 appended Mutations 5 and 6 for SPEC-09 before this plan executed) — the plan's assumption was stale, and `## Mutation 6` was already occupied by an unrelated, already-committed, already-reverted proof (the SPEC-09 acknowledgment-echo anchor).
- **Fix:** Appended the three new mutations as Mutations 7, 8, and 9 instead, preserving the existing Mutation 6 untouched. Added a dedicated "A note on numbering" section to `MUTATION-PROOF.md` immediately before the new mutations, stating the discrepancy plainly and giving the corrected verification commands (`rg -c '^## Mutation' ... returns 9`, with the three proofs at headers 7, 8, 9). All three guards named in the plan's success criteria (GUARD-02 criterion 3, GUARD-01 criterion 4 with its asymmetry) are demonstrated regardless of the number attached to each.
- **Files modified:** `test/wireoracle/MUTATION-PROOF.md`
- **Commit:** `af00d77`

---

**Total deviations:** 2 auto-fixed (both Rule 3 — blocking issues caused by the plan's assumptions not matching the actual repository state at execution time)
**Impact on plan:** No scope change. All three tasks' behavior, acceptance criteria (re-verified against the actual mutation numbers), and success criteria are fully satisfied; only commit granularity and mutation numbering differ from the plan's literal text.

## Issues Encountered
None beyond the two deviations above. The pre-existing, documented, load-dependent `internal/daemon` flake (`TestDaemonRunWaitsForInFlightFlushBeforeReleasingLock`-family, "Daemon extreme-load tail (ACCEPTED, not a gap)" in STATE.md) was NOT observed to fire during this plan's `go test ./...` runs — `internal/mcp` and `test/wireoracle` (the only packages this plan touches) both passed clean on every run, including the final run on the fully-reverted, committed tree.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Plan 05-04 (wire-coverage-remaining) can proceed: this plan added zero wire-visible behavior and zero new resource content — only test-time guards and a mutation-proof record — so no golden transcript was touched and `resources-list.golden`/`resources-read-*.golden` states from 05-02 remain exactly as 05-02 left them.
- The claims-drift guard is now live on every `go test ./internal/mcp/...` run: any future tool rename, numeric-default change, count drift, unauthorized env-var mention, or host-path leak in `internal/mcp/resources/*.md` will fail the build before it can ship — closing the SURF-01 recurrence risk this phase exists to prevent.
- `test/wireoracle/MUTATION-PROOF.md` now carries 9 mutations total; any future contributor extending this file should continue the sequence from 10, not reuse 7/8/9.

---
*Phase: 05-mcp-resources-capability-claims-drift-guard*
*Completed: 2026-08-12*

## Self-Check: PASSED

- FOUND: `internal/mcp/resources_schema_drift_test.go`
- FOUND: `test/wireoracle/MUTATION-PROOF.md`
- FOUND commit `27f5fd2` (Tasks 1+2, feat)
- FOUND commit `af00d77` (Task 3, test)
