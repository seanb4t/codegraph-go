---
phase: 01-protocol-scoping-the-sdk-independent-wire-oracle
plan: 03
subsystem: testing
tags: [go-packages, go-types, archtest, mcp, protocol-version, import-confinement, mark3labs-mcp-go]

# Dependency graph
requires:
  - phase: 01-protocol-scoping-the-sdk-independent-wire-oracle
    provides: "internal/mcp.ProtocolVersion (plan 01), internal/mcp.Server/NewStdioServer seam (plan 01), session_line.go's formatSessionLine/sanitizeClientField (plan 01)"
provides:
  - "internal/mcp/archtest — VRFY-02's identity-agnostic, non-vacuous go/packages+go/types guard forbidding any externally-defined protocol-version constant anywhere in the module, including testdata/golden"
  - "internal/cli/archtest — SDK-02's permanent, non-vacuous direct-import guard keeping internal/cli free of any MCP SDK import"
  - "internal/mcp/session_line_test.go's full D-14/D-16 contract suite (format, hostile-input sanitization, always-on-by-construction)"
  - "The six pre-migration sites (server_test.go, golden_parity_test.go, mcp_stdout_purity_test.go, watch_live_sync_test.go, watch_default_test.go, worktree_notice_test.go) now read from internal/mcp.ProtocolVersion instead of the SDK's LATEST_PROTOCOL_VERSION"
affects: [02-sdk-migration]

# Actuals (#2632)
actuals:
  tokens: 13064
  tasks: 3
  commits: 3

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Identity-agnostic archtest predicate: key on identifier NAME (regex) + external-package boundary check + a types.Const type-switch, not an import-path allowlist — forbids the CLASS of reference so the guard survives a future dependency swap without maintenance (VRFY-02)"
    - "Boundary-aware module-ownership check (exact match OR prefix+\"/\", never a raw strings.HasPrefix) — rejects a lookalike module path (github.com/seanb4t/codegraph-go-fork) that a naive prefix match would incorrectly admit"
    - "Explicit testdata/<pkg> load pattern alongside modulePathPrefix+\"/...\" — go's own wildcard expansion silently skips any directory named testdata; internal/mcp/archtest's loadWholeModule and internal/cli/archtest's assertMCPSDKImporterExists both structurally close this blind spot rather than relying on convention"
    - "Hand-constructed go/types objects (types.NewConst/NewFunc/NewVar/NewField/NewTypeName + a bare *ast.SelectorExpr/*types.Info pair) to unit-test an identifier-classification predicate directly, bypassing packages.Load/go list entirely — used for TestIsExternalProtocolVersionConstantMatrix's 8-row false-positive boundary table, including a lookalike-module-path case no real go.mod fixture could cheaply construct"
    - "Overlay self-defeat test inserts BOTH the import AND a syntactically-used package-level declaration referencing it — a bare unused import fails the load for an unrelated reason (Go rejects unused imports at type-check time) and the self-test would 'pass' because compilation broke, proving nothing"

key-files:
  created:
    - internal/mcp/archtest/protocol_version_test.go
    - internal/mcp/archtest/protocol_version_selftest_test.go
    - internal/cli/archtest/mcp_sdk_confinement_test.go
    - internal/cli/archtest/mcp_sdk_selftest_test.go
  modified:
    - internal/mcp/session_line.go
    - internal/mcp/session_line_test.go
    - internal/mcp/server_test.go
    - testdata/golden/golden_parity_test.go
    - test/integration/mcp_stdout_purity_test.go
    - test/integration/watch_live_sync_test.go
    - test/integration/watch_default_test.go
    - test/integration/worktree_notice_test.go

key-decisions:
  - "isExternalProtocolVersionConstant's true-arm is a type-switch on *types.Const, not a negative 'external and not a struct field' filter — the pre-review draft's broader shape admitted functions, types, methods, and package-level vars, which both cross-AI reviewers flagged as a materially wider false-positive surface than 'an SDK-owned protocol-version constant' means. req.Params.ProtocolVersion (a *types.Var struct field) keeps compiling at every test/integration site unchanged."
  - "isModuleOwnedPath is boundary-aware (pkgPath == modulePathPrefix OR strings.HasPrefix(pkgPath, modulePathPrefix+\"/\")), never a raw prefix check — proven against the lookalike-module-path matrix row (github.com/seanb4t/codegraph-go-fork), which a naive strings.HasPrefix would incorrectly admit as module-owned."
  - "The package doc comment documents the (?i)protocol.?version heuristic's limit honestly rather than overstating it: it cannot catch an SDK constant spelled LatestVersion or CurrentRevision. internal/cli/archtest's forbiddenMCPSDKPrefixes import-path list is named explicitly as the spelling-independent complement, and Phase 2 must confirm the replacement SDK's constant is caught by one guard or the other."
  - "TestIsExternalProtocolVersionConstantMatrix constructs go/types objects directly (types.NewConst/NewFunc/NewVar/NewField/NewTypeName) rather than via packages.Load+Overlay against real fixture packages — isExternalProtocolVersionConstant only inspects info.Uses[sel.Sel], never sel.X's structure, so a hand-built object exercises the identical code path without needing a real second module for the lookalike-path case."
  - "mcp_stdout_purity_test.go drops its SDK mcp import entirely (its only use was LATEST_PROTOCOL_VERSION) and imports internal/mcp unaliased instead; the other three test/integration sites keep the SDK import (still needed for InitializeRequest/Implementation/CallToolRequest/etc.) and add a codegraphmcp alias for internal/mcp; golden_parity_test.go already had an internalmcp alias from plan 01, so only the reference itself changed."
  - "SDK-02's guard is scoped to internal/cli's DIRECT imports only (Tests: false, exactly one package), not a transitive closure — internal/cli legitimately reaches the SDK through internal/mcp, and SDK-02's criterion is specifically that serve.go itself names no SDK package. This is simpler than the six-package reachability closure the charm/stdout guards use, and deliberately so."
  - "forbiddenMCPSDKPrefixes forward-declares github.com/modelcontextprotocol/go-sdk alongside today's github.com/mark3labs/mcp-go — a guard naming only today's dependency would go silently vacuous at exactly the moment a Phase 2 SDK swap matters most."

patterns-established:
  - "Both archtests measured and documented their packages.Load wall-clock cost in the file's own doc comment (indicative only, dated) with the authoritative figure recorded here in the plan summary, per the plan's explicit instruction not to pin a bare number in source as a threshold."
  - "Every mutation/RED proof in this plan was performed as a real, observed test run (not asserted) and reverted before the corresponding task's commit — see 'Mutation Proofs' below."

requirements-completed: [VRFY-02, VRFY-03, SDK-02]

coverage:
  - id: D1
    description: "internal/mcp/session_line_test.go pins D-14's fixed format and key order with a positional assertion, and proves nine hostile clientInfo shapes (including a client name embedding the session-line prefix itself) can never break the four-field parseable shape or inject a second diagnostic line"
    requirement: "VRFY-03"
    verification:
      - kind: unit
        ref: "go test ./internal/mcp/... -run 'TestSessionLineFormat|TestSessionLineSanitizesHostileClientInfo|TestSessionLineIsAlwaysOnByConstruction' -count=1"
        status: pass
    human_judgment: false
  - id: D2
    description: "internal/mcp/archtest.TestNoExternalProtocolVersionConstantReferences forbids any externally-defined protocol-version constant anywhere in the module (including testdata/golden), proven non-vacuous by an Overlay-injected synthetic violation observed RED-then-GREEN, and proven not to false-positive on legitimate struct-field/func/type/var usage by an 8-row matrix"
    requirement: "VRFY-02"
    verification:
      - kind: unit
        ref: "go test ./internal/mcp/archtest/... -count=1"
        status: pass
      - kind: unit
        ref: "go test ./internal/mcp/archtest/... -count=2 (repeatability/read-only proof)"
        status: pass
    human_judgment: false
  - id: D3
    description: "internal/cli/archtest.TestInternalCLIImportsNoMCPSDK permanently forbids any direct MCP SDK import in internal/cli, proven non-vacuous by both an Overlay-injected planted import and a manual on-disk mutation restoring the SDK import by hand"
    requirement: "SDK-02"
    verification:
      - kind: unit
        ref: "go test ./internal/cli/archtest/... -count=1"
        status: pass
    human_judgment: false
  - id: D4
    description: "Full plan-level verification: go build ./... && go vet ./... clean; task test:unit, test:golden, test:integration, and test:wireoracle all green after the six-site migration"
    verification:
      - kind: other
        ref: "go build ./... && go vet ./... && task test:unit && task test:golden && task test:integration && task test:wireoracle"
        status: pass
    human_judgment: false

duration: ~55min
completed: 2026-08-05
status: complete
---

# Phase 1 Plan 3: Protocol-Version Guard, SDK-Import Confinement Guard, and Session-Line Contract Summary

**Two go/packages+go/types archtests (VRFY-02's identity-agnostic protocol-version guard and SDK-02's permanent internal/cli SDK-import confinement) plus internal/mcp/session_line_test.go's full D-14/D-16 contract suite — all four's non-vacuity proven by observed RED-then-GREEN mutation runs, not asserted.**

## Performance

- **Duration:** ~55 min
- **Completed:** 2026-08-05
- **Tasks:** 3 (all tdd=true)
- **Files modified:** 12 (4 created, 8 modified)

## Accomplishments

- **VRFY-03 (Task 1):** Added `TestSessionLineFormat` (positional key-order pin, equal-versions adjacency, `tools=0`-not-dropped), `TestSessionLineSanitizesHostileClientInfo` (9 hostile shapes including a client name embedding the session-line prefix itself), and `parseSessionLineFields` + `TestParseSessionLineFieldsFailLoudly` (a test-only parser proving D-14's parseability claim, fails loudly on malformed input). Renamed the tracer-era `TestNewStdioServerRejectsNilSessionLog` to `TestSessionLineIsAlwaysOnByConstruction` per this plan's artifact table. Added the named `clientFieldMaxBytes` constant to `session_line.go`, replacing the repeated `256` literal.
- **VRFY-02 (Task 2):** Added `internal/mcp/archtest`, a go/packages+go/types archtest (never a text search) forbidding any externally-defined protocol-version constant anywhere in the module. `isExternalProtocolVersionConstant`'s true-arm is a `*types.Const` type-switch, not a negative filter — functions, types, methods, and package-level vars correctly return `false`. `isModuleOwnedPath` is boundary-aware, rejecting a lookalike module path. `loadWholeModule` explicitly loads `testdata/golden` as a second pattern, closing the GOLDEN-01 blind spot where `go`'s own `./...` expansion silently skips any directory named `testdata`. Migrated all six known pre-migration sites onto `internal/mcp.ProtocolVersion`.
- **SDK-02 (Task 3):** Added `internal/cli/archtest`, permanently forbidding `internal/cli` from directly importing any MCP SDK package (`github.com/mark3labs/mcp-go` today, `github.com/modelcontextprotocol/go-sdk` forward-declared for Phase 2). Scoped to direct imports only (not a transitive closure) — `internal/cli` legitimately reaches the SDK through `internal/mcp`.
- Every guard's non-vacuity was proven with a real, observed mutation run (see "Mutation Proofs" below), not merely asserted in a doc comment.

## Task Commits

Three tasks, all TDD-gated:

1. **Task 1: Session-line contract and adversarial sanitization tests** - `d9d9855` (test) — new tests added against the already-implemented (Wave 1 tracer) `sanitizeClientField`; RED-then-GREEN proof performed by temporarily removing space-replacement from `sanitizeClientField` (confirmed the embedded-space and prefix-injection hostile cases fail with "got 5/9 fields, want 4"), then reverted.
2. **Task 2: VRFY-02 protocol-version guard + six-site migration** - `61969c3` (feat) — archtest package, matrix test, overlay self-defeat test, and the six-site migration landed together (the guard needs the migration to be green, and the migration needs the guard to prove it was necessary).
3. **Task 3: SDK-02 import confinement guard** - `25fb9c0` (test) — `internal/cli/archtest` package with its own overlay self-defeat companion.

**Plan metadata:** (this commit, pending)

## Files Created/Modified

- `internal/mcp/archtest/protocol_version_test.go` - VRFY-02's guard: `loadWholeModule`, `isExternalProtocolVersionConstant`, `isModuleOwnedPath`, `scanForProtocolVersionRefs`, `TestNoExternalProtocolVersionConstantReferences`, `TestIsExternalProtocolVersionConstantMatrix`, `TestProtocolVersionGuardHelpersFailLoudly`
- `internal/mcp/archtest/protocol_version_selftest_test.go` - `TestProtocolVersionGuardCatchesOverlaidViolation`, the Overlay-injected non-vacuity proof
- `internal/cli/archtest/mcp_sdk_confinement_test.go` - SDK-02's guard: `forbiddenMCPSDKPrefixes`, `hasForbiddenPrefix`, `assertMCPSDKImporterExists`, `TestInternalCLIImportsNoMCPSDK`
- `internal/cli/archtest/mcp_sdk_selftest_test.go` - `TestInternalCLIImportsNoMCPSDK_PlantedImportIsError`, the Overlay-injected non-vacuity proof
- `internal/mcp/session_line.go` - added named `clientFieldMaxBytes` constant, referenced by `sanitizeClientField` instead of a repeated literal
- `internal/mcp/session_line_test.go` - added `TestSessionLineFormat`, `TestSessionLineSanitizesHostileClientInfo`, `parseSessionLineFields`, `TestParseSessionLineFieldsFailLoudly`; renamed `TestNewStdioServerRejectsNilSessionLog` to `TestSessionLineIsAlwaysOnByConstruction`
- `internal/mcp/server_test.go` - migrated to unqualified `ProtocolVersion` (same package)
- `testdata/golden/golden_parity_test.go` - migrated to the pre-existing `internalmcp.ProtocolVersion` alias
- `test/integration/mcp_stdout_purity_test.go` - dropped the SDK `mcp` import (its only use), now imports `internal/mcp` unaliased as `mcp.ProtocolVersion`
- `test/integration/watch_live_sync_test.go`, `test/integration/watch_default_test.go`, `test/integration/worktree_notice_test.go` - added a `codegraphmcp` alias for `internal/mcp`, migrated to `codegraphmcp.ProtocolVersion`

## Mutation Proofs (recorded verbatim per plan instruction)

**Task 1 — `sanitizeClientField` weakened (space-replacement removed), then reverted:**
```
=== RUN   TestSessionLineSanitizesHostileClientInfo/embedded_space
    session_line_test.go:235: parseSessionLineFields(...): parseSessionLineFields: got 5 fields after the prefix, want 4: [...]
=== RUN   TestSessionLineSanitizesHostileClientInfo/embeds_the_session-line_prefix_itself
    session_line_test.go:235: parseSessionLineFields(...): parseSessionLineFields: got 9 fields after the prefix, want 4: [...]
--- FAIL: TestSessionLineSanitizesHostileClientInfo (0.00s)
```
Reverted via restoring the backed-up file; re-run confirmed green.

**Task 2a — `TestNoExternalProtocolVersionConstantReferences` run BEFORE the six-site migration** (proves the guard actually catches all six sites, including the testdata/golden one — the GOLDEN-01 case):
```
--- FAIL: TestNoExternalProtocolVersionConstantReferences (1.09s)
    VRFY-02: ... internal/mcp/server_test.go:81: references mcp.LATEST_PROTOCOL_VERSION ...
    VRFY-02: ... test/integration/mcp_stdout_purity_test.go:120: references mcp.LATEST_PROTOCOL_VERSION ...
    VRFY-02: ... test/integration/watch_default_test.go:119: references mcp.LATEST_PROTOCOL_VERSION ...
    VRFY-02: ... test/integration/watch_live_sync_test.go:47: references mcp.LATEST_PROTOCOL_VERSION ...
    VRFY-02: ... test/integration/worktree_notice_test.go:44: references mcp.LATEST_PROTOCOL_VERSION ...
    VRFY-02: ... testdata/golden/golden_parity_test.go:1477: references mcp.LATEST_PROTOCOL_VERSION ...
```
All six sites named, confirming loadWholeModule's explicit `testdata/golden` pattern reaches the one site `./...` would have missed.

**Task 2b — `TestProtocolVersionGuardCatchesOverlaidViolation` run against a neutered `scanForProtocolVersionRefs` (temporarily hardcoded `return nil`), then reverted:**
```
--- FAIL: TestProtocolVersionGuardCatchesOverlaidViolation (0.92s)
    protocol_version_selftest_test.go:107: planted a reference to mcp.LATEST_PROTOCOL_VERSION in
    github.com/seanb4t/codegraph-go/internal/mcp (an in-memory overlay only — the real file on disk
    is untouched) but scanForProtocolVersionRefs did not flag it — got violations: []
```
Reverted via restoring the backed-up file; re-run confirmed green (`go test ./internal/mcp/archtest/... -count=2` also confirmed identical/repeatable results).

**Task 3 — manual on-disk restoration of the SDK import into `internal/cli/serve.go` (`import "github.com/mark3labs/mcp-go/server"` plus a referencing `var`), then reverted:**
```
--- FAIL: TestInternalCLIImportsNoMCPSDK (0.19s)
    mcp_sdk_confinement_test.go:115: package github.com/seanb4t/codegraph-go/internal/cli directly
    imports github.com/mark3labs/mcp-go/server (forbidden prefix "github.com/mark3labs/mcp-go") —
    internal/cli must bootstrap entirely through the internal/mcp.Server seam (mcp.NewStdioServer),
    never an MCP SDK package itself (SDK-02)
```
Reverted via restoring the backed-up file (`git status --short internal/cli/serve.go` clean before the commit); re-run confirmed green.

## Measured `packages.Load` Wall-Clock Cost (authoritative, dated — per plan instruction)

- `go test ./internal/mcp/archtest/... -count=1`: ~1.7s (Apple Silicon, darwin/arm64, go1.26.5, 2026-08-05). `-count=2` confirmed identical/repeatable (2.9s total, read-only scan).
- `go test ./internal/cli/archtest/... -count=1`: ~1.2s (same machine/date), including the overlay self-defeat test.
- Both are well under any plausible CI-leg budget for this repository's existing `task test:unit` wrapper (whole-suite `task test:unit` measured ~3.5 min wall-clock in this same run, dominated by indexer/gitmeta/githooks integration tests, not these two archtest packages).

## Decisions Made

See `key-decisions` in frontmatter for the full list. The most consequential:
- `isExternalProtocolVersionConstant`'s `*types.Const` type-switch (not a negative filter) is what keeps `req.Params.ProtocolVersion` (a struct field) compiling at every `test/integration` site while still catching `mcp.LATEST_PROTOCOL_VERSION` (a const) — this was the specific false-positive surface both cross-AI reviewers flagged in the pre-review draft.
- The VRFY-02 guard's heuristic limitation (cannot catch `LatestVersion`/`CurrentRevision`-spelled constants) is documented honestly in the package doc comment rather than claimed as zero-maintenance forever; `internal/cli/archtest`'s import-path list is named as the explicit spelling-independent complement.

## Deviations from Plan

None — plan executed exactly as written. All six known pre-migration sites, both archtest packages, and the session-line contract suite matched the plan's `<behavior>`/`<action>`/`<acceptance_criteria>` blocks without requiring an architectural change or an out-of-scope fix.

## Issues Encountered

None — all builds, vets, and test runs succeeded; every mutation proof reproduced the expected RED on the first attempt.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Both archtests (`internal/mcp/archtest`, `internal/cli/archtest`) are ordinary packages reached by `go list ./...` (confirmed via `task test:unit`'s full run above) — no separate CI wiring is needed for them to run on every PR.
- Phase 2's SDK swap has two independent guards to satisfy: `internal/mcp/archtest`'s name-pattern guard (must confirm the replacement SDK's protocol-version constant matches `(?i)protocol.?version`, or add its spelling deliberately) and `internal/cli/archtest`'s `forbiddenMCPSDKPrefixes` list (already forward-declares `github.com/modelcontextprotocol/go-sdk`, so `assertMCPSDKImporterExists`'s self-defeat check will keep passing once the swap lands).
- `internal/mcp.ProtocolVersion` remains, as plan 01 documented, an asserted compatibility pin in Phase 1 — this plan did not change that mechanism, only made the pin load-bearing by forbidding every alternative source.
- No blockers.

---
*Phase: 01-protocol-scoping-the-sdk-independent-wire-oracle*
*Completed: 2026-08-05*

## Self-Check: PASSED

All 12 created/modified files verified present via `git ls-files --error-unmatch`.
All three task commits (`d9d9855`, `61969c3`, `25fb9c0`) verified present in
`git log --oneline --all`.
