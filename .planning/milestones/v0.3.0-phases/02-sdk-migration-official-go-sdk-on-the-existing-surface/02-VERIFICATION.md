---
phase: 02-sdk-migration-official-go-sdk-on-the-existing-surface
verified: 2026-08-06T12:00:00Z
status: passed
score: 5/5 must-haves verified
behavior_unverified: 0
overrides_applied: 0
---

# Phase 2: SDK Migration — official go-sdk on the existing surface — Verification Report

**Phase Goal:** `internal/mcp` runs on `modelcontextprotocol/go-sdk@v1.7.0` with the agent-facing surface unchanged — same tools, same behavior, no semantic change on the wire — and `mark3labs/mcp-go` is gone.
**Verified:** 2026-08-06
**Status:** passed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

All five ROADMAP.md success criteria for Phase 2 were checked against the current repo text (edited 2026-08-05: criterion 1 relaxed from byte-identity to semantic equivalence + human diff read; criterion 2 split into harness-code-unmodified vs. transcript-bytes-may-move).

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Every existing MCP tool semantically unchanged from pre-migration across the wire-oracle corpus on go-sdk v1.7.0; full transcript diff read line by line, every changed line with a recorded cause (SDK-01) | ✓ VERIFIED | `go test ./test/wireoracle/... -count=1` → PASS, 23/23 (ran directly, not from SUMMARY). Re-freeze commit `f4c9052` message enumerates all nine named causes verbatim (read in full via `git show f4c9052 -s`), two flagged `[SEMANTIC]` (#2 `legacy-omitted-version` value change, #9 `edge-call-before-initialize` success→rejection flip). Independently spot-checked golden files confirm the described content: `legacy-unsupported-2026-07-28.golden`'s `protocolVersion` is `"2025-11-25"` (did not move — matches the claimed non-event); `cacheScope` is `"private"` in every one of 11 `tools/list`-bearing transcripts, zero `"public"` occurrences anywhere in `testdata/wireoracle/transcripts/`; `toolslist-no-index.golden` still advertises `"tools":{"listChanged":true}` at zero registered tools (D-11 regression fix holds). |
| 2 | Phase 1's harness *code* unmodified and still runs; transcript *bytes* move only through the reviewed pass with causes named — never regenerated wholesale, never relaxed to pass (SDK-01) | ✓ VERIFIED | `git diff --name-only abea7ea..HEAD -- test/wireoracle/` → exactly `test/wireoracle/scenarios.go`, nothing else (ran directly). The two argued exceptions inside that one file — `legacyOmittedVersionCoercion`'s value move and `edge-call-before-initialize`'s doc-comment retraction — are named as exceptions in both `02-05-SUMMARY.md`'s frontmatter and the `f4c9052` commit message, not silently taken. `rg -n 'ExpectedScenarioCount = ' test/wireoracle/scenarios.go` → `23`, unchanged. |
| 3 | `mark3labs/mcp-go` absent from `go.mod`, closure re-audited through `govulncheck` and SBOM paths (SDK-03) | ✓ VERIFIED | `rg -n 'mark3labs' go.mod go.sum` → zero matches in both (ran directly). `rg -n '"github.com/mark3labs' --type go` → only two hits, both string literals in forbidden-module-prefix lists (`internal/cli/archtest/mcp_sdk_confinement_test.go`, `tools/transcriptfreeze/classify.go`), not imports — confirmed by `go build ./...` exiting 0. `task vuln` (the real `govulncheck` CI target) run directly → "No vulnerabilities found. Your code is affected by 0 vulnerabilities." SBOM claim (local `syft` run naming go-sdk, zero mark3labs) taken from `02-04-SUMMARY.md` verbatim output — not independently re-run (no tagged release build available in this environment; the real SBOM path only runs in `release.yml` against a tag), but consistent with the go.mod/go.sum state independently confirmed. |
| 4 | A handler returning a plain Go `error` produces a known, asserted wire shape — covered by an explicit test, not inferred from the type signature (SDK-04) | ✓ VERIFIED | `internal/mcp/error_mapping_test.go` exists with exactly four test functions (`TestHandlerErrorIsToolResultNotProtocolError`, `TestMissingRequiredArgumentIsToolVisibleError`, `TestUnknownArgumentIsRejected`, `TestEngineErrorIsToolResult`), confirmed by `rg -n '^func Test'`. Ran all four directly: `go test ./internal/mcp/... -run '...' -v -count=1` → 4/4 PASS. `02-03-SUMMARY.md` records a full mutation cycle (mutation applied via `*jsonrpc.Error` on `codegraph_status`'s error path, confirmed applied via `git diff`, observed FAIL with the exact expected message, reverted, confirmed green via `git status --porcelain` empty) — this is the demonstrated-RED gate proof the honesty contract asked to check for; format matches `MUTATION-PROOF.md`'s established convention. |
| 5 | Tool input schemas keep constraint semantics, or each loss written down as a deliberate divergence (SDK-05) | ✓ VERIFIED | `f4c9052` commit message's SDK-05 finding section (read directly): zero enum constraints existed pre-migration to lose (`codegraph_files`' `format` values live only in a description string — matches `02-CONTEXT.md` D-08's prediction); the only constraint-semantics deltas are `number`→`integer` on 7 numeric params and `additionalProperties:false` on all 8 tools, the latter explicitly recorded as an accepted improvement (D-10) rather than a silent loss. Independently confirmed via golden-file grep: `"type":"integer"` and `"additionalProperties":false` both present in the re-frozen transcripts inspected above. |

**Score:** 5/5 truths verified (0 present-but-behavior-unverified)

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/mcp/server.go` | go-sdk-backed `Server` implementation behind the unchanged seam | ✓ VERIFIED | Builds (`go build ./...` exit 0); `internal/mcp` test suite passes. |
| `internal/mcp/tools.go` | 8 tools re-registered via `mcp.AddTool` with typed `*Args` structs | ✓ VERIFIED | Compiles; error-mapping tests exercise handler paths through it directly. |
| `internal/mcp/error_mapping_test.go` | Four SDK-04 assertions | ✓ VERIFIED | Present, 4 test funcs, all pass when run directly. |
| `go.mod` / `go.sum` | `mark3labs/mcp-go` absent, `modelcontextprotocol/go-sdk v1.7.0` present | ✓ VERIFIED | Confirmed via `rg` (zero mark3labs matches, one go-sdk match at v1.7.0). |
| `test/wireoracle/scenarios.go` | Only harness file touched by the phase | ✓ VERIFIED | `git diff --name-only` confirms sole change. |
| `testdata/wireoracle/transcripts/*.golden` (23 files) | Re-frozen against go-sdk backend | ✓ VERIFIED | `TestFrozenTranscriptsMatch` passes; spot-checked 3 golden files' content directly. |
| `tools/transcriptfreeze/classify.go` | Self-expiring SDK-01 exemption | ✓ VERIFIED | `TRANSCRIPT_FREEZE_BASE=main task check:transcript-freeze` run directly → exit 0, exemption notice printed naming all 23 transcripts and the mark3labs→go-sdk transition, matching the claimed shape exactly. |

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| `internal/cli/serve.go` | `internal/mcp.NewStdioServer` | the `Server` seam, no MCP SDK import in `internal/cli` | ✓ WIRED | `rg -n 'mcp\.\|modelcontextprotocol\|mark3labs' internal/cli/serve.go` shows only calls to the package-local `mcp` alias (`internal/mcp`), no SDK import; `internal/cli/archtest` guard passes. |
| `.github/workflows/ci.yml` `transcript-freeze` job | `tools/transcriptfreeze` classifier | blocking PR gate | ✓ WIRED | Ran the real command against `main`; exemption fires correctly, not vacuously (only fires on the real mark3labs→go-sdk go.mod transition). |
| Wire oracle capture CLI | 23 `.golden` transcripts | `task test:wireoracle` / `go test ./test/wireoracle/...` | ✓ WIRED | Ran directly, 23/23 pass. |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Wire oracle passes on go-sdk backend | `go test ./test/wireoracle/... -count=1` | `ok ... 16.784s` (re-run also `ok ... 31.006s` in full-suite context) | ✓ PASS |
| Build succeeds | `go build ./...` | exit 0 | ✓ PASS |
| SDK-04 error-mapping assertions | `go test ./internal/mcp/... -run '...' -v -count=1` | 4/4 PASS | ✓ PASS |
| `internal/mcp` + both archtest packages | `go test ./internal/mcp/... ./internal/cli/archtest/... -count=1` | all `ok` | ✓ PASS |
| `mark3labs` absent from go.mod/go.sum | `rg -n 'mark3labs' go.mod go.sum` | zero matches | ✓ PASS |
| `govulncheck` clean | `task vuln` | "0 vulnerabilities" | ✓ PASS |
| Transcript-freeze exemption fires correctly | `TRANSCRIPT_FREEZE_BASE=main task check:transcript-freeze` | exit 0, correct exemption message | ✓ PASS |
| Full workspace suite | `go test ./... -count=1` (run once) | all packages `ok` except `internal/daemon` | ⚠️ see Excluded Condition below |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|--------------|--------|----------|
| SDK-01 | 02-01, 02-02, 02-05 | go-sdk migration, semantic equivalence proven via reviewed diff | ✓ SATISFIED | Wire oracle green on go-sdk backend; harness-code-unmodified confirmed; nine-cause divergence record confirmed present and matches golden-file content. |
| SDK-03 | 02-04 | mark3labs removed, closure re-audited | ✓ SATISFIED | Confirmed absent from go.mod/go.sum; govulncheck confirmed clean directly. |
| SDK-04 | 02-03 | error-to-wire mapping asserted | ✓ SATISFIED | 4/4 tests pass directly; mutation-cycle record present in SUMMARY. |
| SDK-05 | 02-05 (audit folded into re-freeze) | schema constraint audit | ✓ SATISFIED | Zero enum constraints existed to lose; deltas (`number`→`integer`, `additionalProperties:false`) explicitly recorded, not silently discovered. |

No orphaned requirements — REQUIREMENTS.md maps exactly SDK-01/SDK-03/SDK-04/SDK-05 to Phase 2, and all four appear in at least one plan's `requirements` field (SDK-02 was completed in Phase 1).

### Anti-Patterns Found

`rg -n 'TBD|FIXME|XXX|TODO|HACK|PLACEHOLDER' internal/mcp/error_mapping_test.go internal/mcp/tools.go internal/mcp/server.go internal/mcp/protocol_version.go test/wireoracle/scenarios.go` and a broader sweep of `internal/mcp/`, `test/wireoracle/`, `tools/transcriptfreeze/` (Go files) → zero matches. No debt markers found in phase-modified files.

### Excluded Condition (observed, out of scope per verification brief)

Running the full workspace suite once (`go test ./... -count=1`) reproduced the known `internal/daemon` flake: `TestDaemonFlushLockRequeueGivesUpPerEpisode` failed under full-suite load (a lock-requeue timing race). This is the pre-existing condition described in the verification brief (issue #17 / MAINT-02, scheduled for Phase 4) — independently confirmed excluded from this phase's scope:
- `git diff --name-only abea7ea..HEAD | rg -i daemon` → empty (zero daemon files touched by any Phase 2 commit)
- `go list -deps ./internal/daemon | rg 'internal/mcp'` → empty (no dependency on the migrated package)

Not scored against Phase 2; noted here per the verification brief's instruction to keep the exclusion visible.

## Deviations / Notes

- **`02-VALIDATION.md` frontmatter `status: draft`** — this is the document's own lifecycle field (set to `validated` by a separate `/gsd-validate-phase` pass), not evidence of incomplete verification. Not counted as a gap.
- **SBOM claim (SDK-03, criterion 3)** — taken from `02-04-SUMMARY.md`'s recorded `syft` output rather than independently re-run, since the real SBOM assembly path (`release.yml`) only executes against a tagged release build, which this environment cannot produce. The independently-verified `go.mod`/`go.sum` state and `govulncheck` result are consistent with the SBOM claim; this is disclosed as "asserted in SUMMARY, not independently re-run" per the honesty contract rather than silently accepted as fact.

## Gaps Summary

None. All five ROADMAP success criteria, all four requirements (SDK-01, SDK-03, SDK-04, SDK-05), and every claim listed in the verification brief's `<verify_these_specific_claims_independently>` section were checked directly against the repository (not taken from SUMMARY prose) and confirmed accurate. The one SUMMARY-sourced claim not independently re-run (the SBOM's package listing) is disclosed above rather than silently trusted, and is corroborated by independently-verified adjacent evidence (go.mod state, govulncheck).

---

*Verified: 2026-08-06*
*Verifier: Claude (gsd-verifier)*
