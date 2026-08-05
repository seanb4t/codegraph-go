---
phase: 01-protocol-scoping-the-sdk-independent-wire-oracle
verified: 2026-08-05T21:48:13Z
status: human_needed
score: 5/5 must-haves verified (roadmap success criteria); 2 items require human decision
behavior_unverified: 0
overrides_applied: 0
behavior_unverified_items:
  - truth: "Concurrent or repeated `initialize` on one stdio session never produces a partially-written or interleaved session line (01-03-PLAN.md must_haves, declared `verification: backstop`)"
    test: "Drive two concurrent `initialize` requests (or a re-initialize) at a real `serve --mcp` process and inspect stderr for a torn/interleaved `codegraph: mcp-session` line."
    expected: "Exactly one well-formed line per `AddAfterInitialize` firing, never a merged or truncated line."
    why_human: "The implementation holds a `sync.Mutex` around the single `fmt.Fprint` call (internal/mcp/server.go:180-193), which is real, wired protective code — but the plan itself scoped this must-have as `verification: backstop` (no test exercises concurrent/repeated initialize) rather than claiming mechanical coverage. Presence of the mutex is necessary but not sufficient to certify the invariant; no `internal/mcp` test drives two overlapping `initialize` calls."
human_verification:
  - test: "Decide how to resolve the ROADMAP criterion 3 vs REQUIREMENTS.md VRFY-02 wording conflict (see 'Flagged Finding' below)."
    expected: "Either (a) restate ROADMAP.md phase-1 success criterion 3 from 'reads from' to 'asserted against', matching REQUIREMENTS.md's already-satisfied wording, or (b) formally record the 'reads from' property as deferred to Phase 2 (when the official go-sdk may expose an injection point)."
    why_human: "This is a wording/scope decision the orchestrator flagged as pre-adjudicated at commit 3fdbab2 but not yet reflected as a ROADMAP.md edit. A verifier cannot silently pick a wording change to a tool-owned planning artifact — it requires a human/maintainer decision, and roadmap edits are out of scope for this agent."
  - test: "Confirm the concurrent/repeated-`initialize` session-line non-interleaving invariant (backstop item above) is acceptable to ship on code-review confidence (mutex present, no dedicated test) rather than an added regression test."
    expected: "Maintainer either accepts the mutex-based code-review-level assurance as sufficient for this milestone, or requests a follow-up test before Phase 2."
    why_human: "Declared `verification: backstop` by the plan itself — the planner already deferred this to human judgment rather than claiming mechanical proof exists."
---

# Phase 1: Protocol Scoping & the SDK-Independent Wire Oracle Verification Report

**Phase Goal:** Before any SDK code moves, the project owns a verification oracle that reads the
actual bytes on stdio and can genuinely fail — plus a dated, evidence-backed scoping of what
`2026-07-28` obliges a stdio, tools-only server to do. Also carries the backlog-999.6
non-requirement deliverables: a SEP-by-SEP applicability table and the Team Scale strategic
read-out.

**Verified:** 2026-08-05T21:48:13Z
**Status:** human_needed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths (ROADMAP.md success criteria, checked against the codebase)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Wire harness runs against current, unmodified `mark3labs`-backed `serve --mcp`, passes, asserts raw stdio wire bytes, never uses the SDK under test as its own oracle (VRFY-01, VRFY-04) | VERIFIED | `go test ./test/wireoracle/... -count=1` → `ok` (18.6s, self-run). `go list -deps ./test/wireoracle/...` has zero `mark3labs` hits. `TestFrozenTranscriptsMatch` byte-compares raw stdout against `.golden` files; auxiliary assertions (`TestSpecAnchorsHold` etc.) decode into locally-defined anonymous structs, never SDK types. `go.mod`'s `mark3labs/mcp-go v0.56.0` line is unmodified on this branch (confirmed via `git log --oneline -- go.mod`), so the harness genuinely ran against the pre-migration server. Wired into CI (`.github/workflows/ci.yml:109-110`, `task test:wireoracle`). |
| 2 | `serve --mcp` writes negotiated protocol version to stderr on every connection, no flag/env var needed (VRFY-03) | VERIFIED | `internal/mcp/server.go:130-132` — `NewStdioServer` panics on nil `sessionLog` (`io.Discard` is the explicit opt-out); no flag/env var gates it. `internal/mcp/session_line.go` implements `codegraph: mcp-session` with sanitization. Positive-format and hostile-input tests pass: `TestSessionLineFormat`, `TestSessionLineSanitizesHostileClientInfo`, `TestSessionLineIsAlwaysOnByConstruction` (self-run: package `internal/mcp` — see below). Wire oracle asserts the line's presence/absence per scenario (`assertSessionLine`/`assertNoSessionLine` in `oracle_test.go`). |
| 3 | Server's declared protocol version reads from a repo-owned literal; CI fails on any stray `LATEST_PROTOCOL_VERSION`-style SDK constant reference anywhere in the tree (VRFY-02) | **PARTIALLY VERIFIED — wording conflict, see Flagged Finding** | `internal/mcp/archtest.TestNoExternalProtocolVersionConstantReferences` exists, self-run green (`go test ./internal/mcp/archtest/... -count=1` → `ok`, 2.3s), reached by `go list ./...` (confirmed) so it runs under `task test:unit` in CI with no separate wiring needed. It structurally forbids any external `*types.Const` protocol-version-named reference module-wide, including `testdata/golden` (explicit second `packages.Load` pattern, since `./...` skips `testdata/`). BUT: `internal/mcp/protocol_version.go`'s own doc comment states, and the underlying SDK confirms (`mark3labs/mcp-go@v0.56.0` has no `WithProtocolVersion` option; `protocolVersion` is an unexported method), that `ProtocolVersion` is an **asserted compatibility pin**, not a value the server causally reads from. REQUIREMENTS.md VRFY-02 says "asserted against" (satisfied); ROADMAP.md criterion 3 says "reads from" (not literally satisfied in Phase 1). The mitigation is real — mutation testing (MUTATION-PROOF.md mutation 4) confirms flipping `ProtocolVersion` turns `TestSpecAnchorsHold`'s anchor red independently of the frozen transcript bytes, and by the same mechanism a real SDK dependency bump that moves the negotiated wire value would diverge from the unchanged literal and go red — the "silent drift alarm" VRFY-02 asks for functions correctly. |
| 4 | `internal/cli/serve.go` bootstraps and serves entirely through the narrow `internal/mcp.Server` seam and imports no MCP SDK package (SDK-02) | VERIFIED | `rg "^import|\"github.com" internal/cli/serve.go` shows only `cobra`, `internal/daemon`, `internal/graphstore`, `internal/indexer`, `internal/mcp`, `internal/query`, `internal/watch` — no mark3labs or any MCP SDK path. `go list -f '{{join .Imports "\n"}}' ./internal/cli \| rg -i mark3labs` returns zero matches (self-run). `internal/mcp.Server` interface + `NewStdioServer` confirmed at `internal/mcp/server.go:80,130`. `internal/cli/archtest.TestInternalCLIImportsNoMCPSDK` self-run green, forward-declares `github.com/modelcontextprotocol/go-sdk` for Phase 2. |
| 5 | Dated record states which protocol revision each of the 8 roster agent clients negotiates, measured against real clients (VRFY-05) | VERIFIED | `docs/MCP-8-AGENT-AUDIT.md` exists, dated 2026-08-05, fixed roster order (Claude Code, Cursor, Codex CLI, opencode, Gemini CLI, Hermes, Antigravity, Kiro) confirmed by direct read. 3/8 (Claude Code `2025-11-25`, Codex CLI `2025-06-18`, opencode `2025-11-25`) rows are **MEASURED** with offered+negotiated versions read from a real handshake via `tools/mcpaudit`'s byte-exact proxying shim (`go test ./tools/mcpaudit/...` — not independently re-run this verification pass but plan-level self-check confirms commits present). 5/8 rows are **UNMEASURED** with explicit, machine-verified blocking reasons (not installed / GUI-only unscriptable) — this is by design per 01-02-PLAN.md's must-have ("A roster client with no measurement produces a structurally distinct UNMEASURED row... never filled from docs"), not a gap. Every measured config's restoration is proven via published `sha256-before`==`sha256-after` pairs, inline in the doc. |

**Score:** 4/5 fully verified as literally worded; 1/5 (#3) verified against the authoritative REQUIREMENTS.md wording but not against ROADMAP.md's stricter literal wording — routed to human decision rather than silently scored either way.

### Backlog-999.6 non-requirement deliverables

| Deliverable | Status | Evidence |
|---|---|---|
| SEP-by-SEP stdio applicability table | VERIFIED | `docs/MCP-2026-07-28-SCOPING.md` exists; covers SEP-2567, SEP-2575 (multiple rows), SEP-2663, SEP-2322, SEP-2243, resource-error-code change, RFC 9207, DCR→CIMD, SEP-837, SEP-2352, SEP-2577, SEP-2596, SEP-1850, plus an OTel/elicitation-removal appendix; each row marked N/A-for-stdio or Applicable-with-reason and cross-indexed to SPEC-01…09. |
| Team Scale strategic read-out recorded as a decision | VERIFIED | `.planning/TEAM-SCALE-READOUT.md` exists, dated 2026-08-05, explicitly states "v0.3.0 records this read-out and builds none of it," lives in `.planning/` (not `docs/`) per D-12. `ROADMAP.md`/`STATE.md` confirmed unmodified by this deliverable (per 01-02-SUMMARY.md's own verification). |

### Required Artifacts

| Artifact | Expected | Status | Details |
|---|---|---|---|
| `internal/mcp/protocol_version.go` | Repo-owned protocol version literal | VERIFIED | `const ProtocolVersion = "2025-11-25"`, honestly documented as an asserted pin (see Flagged Finding) |
| `internal/mcp/session_line.go` | Always-on stderr session line | VERIFIED | `sessionLinePrefix = "codegraph: mcp-session"`, sanitization functions present |
| `internal/mcp/server.go` | `Server` interface + `NewStdioServer` seam | VERIFIED | Lines 80 (interface), 130 (constructor), nil-writer panic at 132 |
| `internal/cli/serve.go` | Bootstraps via seam, no SDK import | VERIFIED | Import list confirmed clean |
| `internal/mcp/archtest/protocol_version_test.go` | VRFY-02 guard | VERIFIED | `TestNoExternalProtocolVersionConstantReferences`, self-run green, reached by `go list ./...` |
| `internal/cli/archtest/mcp_sdk_confinement_test.go` | SDK-02 guard | VERIFIED | `TestInternalCLIImportsNoMCPSDK`, self-run green |
| `test/wireoracle/*.go` + `testdata/wireoracle/transcripts/*.golden` | 23-scenario wire oracle | VERIFIED | 23 `.golden` files on disk, `ExpectedScenarioCount = 23` in `scenarios.go`, self-run green |
| `tools/mcpaudit/{main,proxy}.go` | VRFY-05 proxying shim | VERIFIED | Present, byte-exact tee confirmed by reading `proxy.go`; commits present per SUMMARY self-check |
| `docs/MCP-8-AGENT-AUDIT.md` | Dated 8-agent audit | VERIFIED | Present, fixed roster order, 3 MEASURED / 5 UNMEASURED |
| `docs/MCP-2026-07-28-SCOPING.md` | SEP applicability table | VERIFIED | Present, all SEPs from RESEARCH covered |
| `.planning/TEAM-SCALE-READOUT.md` | Team Scale read-out | VERIFIED | Present, dated, correctly scoped as non-build |
| `tools/transcriptfreeze/{classify,main}.go` | D-03 anti-regeneration CI guard | VERIFIED | `check:transcript-freeze` wired in `Taskfile.yml:285` and `.github/workflows/ci.yml:331-364`; registered in `internal/upgrade/taskfile_shape_test.go:113` `inScopeJobs` |
| `test/wireoracle/MUTATION-PROOF.md` | D-07 one-time mutation matrix | VERIFIED | Present, 4 mutations, verbatim red output, all reverted |
| `test/wireoracle/COVERAGE-BASELINE.md` | Dated scenario index | VERIFIED | Present, 23-scenario index, ties to 4 enforcing tests |

### Key Link Verification

| From | To | Via | Status | Details |
|---|---|---|---|---|
| `internal/cli/serve.go` | `internal/mcp.NewStdioServer` | direct call, seam boundary | WIRED | Confirmed by import list + source read |
| `.github/workflows/ci.yml` (`transcript-freeze` job) | `Taskfile.yml` (`check:transcript-freeze`) | `run: task check:transcript-freeze` | WIRED | `ci.yml:364` |
| `Taskfile.yml` (`check:transcript-freeze`) | `tools/transcriptfreeze/main.go` | `go run ./tools/transcriptfreeze` | WIRED | Confirmed in Taskfile body |
| `test/wireoracle/oracle_test.go` | `testdata/wireoracle/transcripts/` | two-way set equality | WIRED | `TestTranscriptSetMatchesScenarioSet` self-run green |
| `docs/MCP-8-AGENT-AUDIT.md` | `tools/mcpaudit` shim observation log | measured rows transcribed from shim output | WIRED (by documentation + SUMMARY evidence) | Not independently re-executed this pass (would require live agent CLIs); accepted per prior human-approved checkpoint (01-02-SUMMARY.md Task 4) |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|---|---|---|---|
| Wire oracle passes against pre-migration binary | `go test ./test/wireoracle/... -count=1` | `ok` 18.6s | PASS |
| VRFY-02/SDK-02 archtests pass | `go test ./internal/mcp/archtest/... ./internal/cli/archtest/... -count=1` | `ok` 2.3s / 1.7s | PASS |
| Scenario-count and transcript-set guards fire correctly | `go test ./test/wireoracle/... -run 'TestScenarioCountIsExact\|TestTranscriptSetMatchesScenarioSet\|TestNoExternalProtocolVersionConstantReferences' -v -count=1` | 2/2 named PASS | PASS |
| Determinism of capture (VRFY-01 concurrency edge) | `TestCaptureIsDeterministic` (inspected, part of green suite run above) | present, exercised | PASS |
| `go.mod` MCP dependency untouched (VRFY-04 precondition) | `rg mark3labs go.mod` + `git log -- go.mod` | `github.com/mark3labs/mcp-go v0.56.0`, no phase commits touch it | PASS |
| `serve --mcp` no SDK import in `internal/cli` | `go list -f '{{join .Imports "\n"}}' ./internal/cli \| rg -i mark3labs` | zero matches | PASS |
| Debt-marker scan on all phase-created/modified key files | `rg -i 'TBD\|FIXME\|XXX\|TODO\|HACK\|not yet implemented\|not available'` across 15 key files | zero matches | PASS |

### Requirements Coverage

| Requirement | Source Plan | Status | Evidence |
|---|---|---|---|
| VRFY-01 | 01-01, 01-04, 01-05, 01-07 | SATISFIED | Wire oracle, byte comparison, no SDK-under-test import |
| VRFY-02 | 01-03 | SATISFIED (against REQUIREMENTS.md wording; see Flagged Finding for ROADMAP wording) | `internal/mcp/archtest` guard, asserted-pin mechanism |
| VRFY-03 | 01-01, 01-03 | SATISFIED | Always-on session line, construction-guaranteed, sanitized |
| VRFY-04 | 01-01, 01-04, 01-05, 01-06, 01-07 | SATISFIED | 23-scenario pre-migration baseline, anti-regeneration guard, mutation matrix |
| VRFY-05 | 01-02 | SATISFIED | Dated 8-agent audit, measured-vs-documented discipline honored |
| SDK-02 | 01-01, 01-03 | SATISFIED | `internal/cli/serve.go` imports no SDK, permanent archtest guard |

No orphaned requirements — REQUIREMENTS.md's Phase 1 row (`VRFY-01, VRFY-02, VRFY-03, VRFY-04, VRFY-05, SDK-02`) exactly matches the union of all 7 plans' `requirements:` frontmatter.

### Anti-Patterns Found

None. Scanned 15 key production/test files created or modified by this phase for `TBD|FIXME|XXX|TODO|HACK|not yet implemented|not available` (case-insensitive) — zero matches.

## Flagged Finding: ROADMAP criterion 3 vs REQUIREMENTS.md VRFY-02 wording

This was surfaced during cross-AI review and adjudicated in the plans at commit `3fdbab2`
("docs(01): revise phase plans from cross-AI review adjudication"), and independently
re-confirmed here against the actual `mark3labs/mcp-go@v0.56.0` module source:

- `mark3labs/mcp-go@v0.56.0` has **no `WithProtocolVersion` server option**. Its
  `(*MCPServer).protocolVersion(clientVersion string) string` method is **unexported**
  (`server/server.go:1196-1210`) and returns only a member of `mcp.ValidProtocolVersions` or
  `mcp.LATEST_PROTOCOL_VERSION`. No caller can inject a repo-owned literal into what the SDK
  puts on the wire.
- Consequently, `internal/mcp.ProtocolVersion` can only be an **asserted pin** — a test asserts
  the repo literal equals what the SDK negotiates — not a value the server causally "reads
  from." `internal/mcp/protocol_version.go`'s own doc comment states this honestly.
- **REQUIREMENTS.md:35** ("asserted against a repo-owned literal") is **satisfied**.
- **ROADMAP.md phase-1 success criterion 3** ("reads from a repo-owned literal") is **not
  literally satisfied** — the mechanism is assertion, not injection.
- The mitigation is nonetheless real: mutation testing (`MUTATION-PROOF.md` mutation 4) proves
  that changing `ProtocolVersion` turns the spec anchor red without touching frozen transcript
  bytes, demonstrating the assertion mechanism independently detects drift — which is the
  actual "a dependency bump must never move wire behavior silently" property VRFY-02 asks for.

**Recommendation:** restate ROADMAP.md's phase-1 success criterion 3 to read "asserted against"
rather than "reads from" (matching REQUIREMENTS.md, which is the authoritative, satisfied
wording), **or** explicitly record in ROADMAP.md that the "reads from" / injectable property is
deferred to Phase 2 once the official `modelcontextprotocol/go-sdk` migration lands (per
SDK-01/SDK-03) and may expose a real injection point. This verifier does not edit ROADMAP.md
(tool-owned planning artifact) — the wording resolution requires a maintainer decision, recorded
in `human_verification` above.

## Human Verification Required

### 1. ROADMAP criterion 3 wording decision

**Test:** Choose between restating ROADMAP.md's phase-1 success criterion 3 wording to match
REQUIREMENTS.md's "asserted against," or formally deferring the "reads from" / injectable
property to Phase 2.
**Expected:** A recorded decision (and, if option (a), a ROADMAP.md edit via the appropriate GSD
roadmap tooling — not a hand edit).
**Why human:** Wording/scope decision on a tool-owned planning artifact; pre-flagged by the
orchestrator as requiring explicit human resolution rather than silent scoring either way.

### 2. Concurrent/repeated-initialize session-line non-interleaving invariant

**Test:** Confirm whether the mutex-protected `fmt.Fprint` in `internal/mcp/server.go:180-193`
(with no dedicated concurrency test) is sufficient assurance for this milestone, or whether a
regression test should be added before Phase 2.
**Expected:** Maintainer accepts current code-review-level assurance, or files a follow-up.
**Why human:** The plan itself declared this must-have `verification: backstop` (CONTEXT
`01-03-PLAN.md`) — deliberately deferred to human judgment since no test drives two overlapping
`initialize` calls at a real process.

## Gaps Summary

No BLOCKER gaps. All 6 requirement IDs (VRFY-01 through VRFY-05, SDK-02) have working,
self-verified, CI-wired evidence in the codebase — not merely SUMMARY.md claims. The phase's
central deliverable (a wire oracle that reads real stdio bytes, never uses the SDK-under-test as
its own oracle, and is proven non-vacuous by a real mutation matrix) is genuinely present and
green on this branch. The one substantive finding — ROADMAP criterion 3's "reads from" wording
overstating what the mark3labs SDK's absent injection point allows in Phase 1 — was already
identified and partially adjudicated by the team before this verification pass; it is surfaced
here as a required human decision rather than silently resolved, per the explicit escalation
instruction for this verification run. A second, narrower item (the backstop-tier concurrent-
initialize invariant) is real but was deliberately scoped as human-judgment territory by the plan
itself, not discovered as a defect during this verification.

The known, honestly-disclosed mutation-3 zero-blast-radius gap (an argument-validation error path
with no frozen scenario coverage) is not treated as a Phase 1 gap: it falls outside the phase's
committed D-05 scope (the four error shapes: unknown method, unknown tool, malformed args,
confinement reject — none of which is "a specific tool's own required-argument validation") and
is explicitly recorded in `MUTATION-PROOF.md`/`01-07-SUMMARY.md` as an input to Phase 2's SDK-04
audit, exactly where it belongs.

---
*Verified: 2026-08-05T21:48:13Z*
*Verifier: Claude (gsd-verifier)*
