---
phase: 03-2026-07-28-spec-compliance
verified: 2026-08-06T16:45:00Z
status: passed
score: 5/5 must-haves verified
behavior_unverified: 0
overrides_applied: 0
re_verification:
  previous_status: none
---

# Phase 3: `2026-07-28` Spec Compliance Verification Report

**Phase Goal:** The server answers the `2026-07-28` wire contract correctly for a stdio, tools-only implementation — discovery, per-request `_meta` validation, result metadata, honest cache control, and per-call index detection — while every client still speaking an older revision continues to work.
**Verified:** 2026-08-06
**Status:** passed
**Re-verification:** No — initial verification

All evidence below was produced by running commands against the actual working tree at commit `6c8d1d1` (HEAD), not by reading SUMMARY.md claims. Every command and its output is quoted.

## Goal Achievement

### Observable Truths (ROADMAP § Phase 3 Success Criteria)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | `server/discover` returns capabilities without a prior tool call; `instructions` carries codegraph usage guidance (SPEC-01, SPEC-07) | ✓ VERIFIED | `testdata/wireoracle/transcripts/modern-discover-explore.golden` id=1 (first message in the session, `NoInitialize: true`) returns `capabilities`, `supportedVersions` (5 eras), and a 511-byte `instructions` string. `go test ./test/wireoracle/... -run TestFrozenTranscriptsMatch` passes. |
| 2 | Per-request `_meta` validated: malformed/missing → `-32602`, unsupported version → `-32022` (SPEC-02) | ✓ VERIFIED | `modern-meta-invalid-params.golden`: `{"error":{"code":-32602,"message":"missing or invalid _meta field \"io.modelcontextprotocol/clientCapabilities\""}}`. `modern-meta-unsupported-version.golden`: `{"error":{"code":-32022,"message":"unsupported protocol version",...}}`. Both independently anchored (`TestSpecAnchorsHold/modern-meta-invalid-params`, `TestSpecAnchorsHold/modern-meta-unsupported-version` — both PASS, run live). `git diff --stat b069f09~1..d844e04 -- internal/mcp/` is empty — confirmed zero server code added for this requirement, matching the plan's documented finding that go-sdk already implements this correctly. |
| 3 | Every tool result carries `resultType: "complete"` and `io.modelcontextprotocol/serverInfo` in `_meta` (SPEC-03, SPEC-08) | ✓ VERIFIED | `modern-discover-explore.golden` id=2 (a `tools/call`) carries `"resultType":"complete"` and `"_meta":{"io.modelcontextprotocol/serverInfo":{"name":"codegraph","version":"0.1.0"}}`. Per D-02 (SDK-owned Modern/Legacy gating), Legacy-era results correctly omit these fields (confirmed: `legacy-2024-11-05.golden` id=3 tool-call result has no `resultType`/`_meta.serverInfo`) — this is the spec-correct behavior, not a gap. |
| 4 | `tools/list`/`server/discover` carry `ttlMs: 0` + `cacheScope: "private"`; `hasIndex` re-checked per call so mid-session `codegraph init` makes tools appear (SPEC-04, SPEC-05) | ✓ VERIFIED | `rg -c '"cacheScope":"public"' testdata/wireoracle/transcripts/*.golden` → zero hits anywhere in the 27-transcript corpus. `index-appears-mid-session.golden`: id=2 `tools/list` (before `codegraph init`) returns `"tools":[]`; id=3 `tools/list` (same connection, after a real mid-session `codegraph init` subprocess) returns `"tools":[{"name":"codegraph_explore",...}]`. `TestDefaultToolVisibility`, `TestAllowlist`, `TestNoIndexZeroTools` in `internal/mcp/server_test.go` still assert exact set equality via `equalStrings`/`len(got)!=0` — not relaxed to non-empty checks (read directly, lines 128-180). `toolslist-no-index.golden` still advertises `"capabilities":{"tools":{"listChanged":true}}` at zero registered tools (D-11 intact). |
| 5 | A client speaking `2025-11-25` or earlier completes a session and calls tools, asserted by test (SPEC-06) | ✓ VERIFIED | `legacy-2024-11-05.golden` (the oldest era) now has 3 messages: `initialize` (id=1), `tools/list` (id=2), `tools/call codegraph_explore` (id=3, result carries `content`). Paired with `handshake-explore` proving the same at `2025-11-25` (the newest Legacy era). Confirmed by direct JSON inspection of the golden file, not by SUMMARY claim. |

**Score:** 5/5 truths verified (0 present-but-behavior-unverified)

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/mcp/server.go` | discover cacheScope fix, `instructions` constant, per-request re-check | ✓ VERIFIED | `case "server/discover"` branch, `registerTools`/`unregisterTools`/`recheckCatalog`, `instructions` const all present and wired into `AddReceivingMiddleware` and `ServerOptions` |
| `internal/mcp/server_test.go` | 5 new SPEC-05 tests, 3 pre-existing set-equality tests unmodified | ✓ VERIFIED | `TestIndexAppearingMidSessionRegistersTools`, `TestIndexAppearingMidSessionHonorsAllowlist`, `TestIndexDisappearingMidSessionUnregistersTools`, `TestRepeatedListsDoNotDuplicateTools`, `TestSessionLineReflectsPostAppearanceToolCount` all present; `go test ./internal/mcp/... -race -count=1` passes |
| `test/wireoracle/scenarios.go` | `ExpectedScenarioCount` progression 23→24→26→27 | ✓ VERIFIED | `rg -o 'ExpectedScenarioCount = [0-9]+'` → `27` |
| `test/wireoracle/anchors.go` | `assertDiscoverCacheControl`, `codeUnsupportedProtocolVersion` anchors | ✓ VERIFIED | Both present, both PASS in a live `TestSpecAnchorsHold` run |
| `testdata/wireoracle/transcripts/*.golden` (27 files) | Frozen wire evidence for every new scenario | ✓ VERIFIED | `ls *.golden \| wc -l` → 27, matches `ExpectedScenarioCount` |
| `tools/transcriptfreeze/{classify,main}.go` | D-03 guard made advisory | ✓ VERIFIED | Live run (`TRANSCRIPT_FREEZE_BASE=be53a2e task check:transcript-freeze`) reports the full collision list and **exits 0** |
| `test/wireoracle/COVERAGE-BASELINE.md` | Rewritten for 27-scenario corpus | ✓ VERIFIED | Header states "Scenario count: 27", Total line sums to 27, History section documents phase-by-phase growth |

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| `AddReceivingMiddleware` switch | `mcp.DiscoverResult.CacheScope` | `case "server/discover"` branch | ✓ WIRED | `modern-discover-explore.golden` shows `cacheScope:"private"` on the discover result |
| Per-request re-check | `registerTools`/`unregisterTools` | `recheckCatalog()` called before `next()` for `initialize`/`tools/list`/`tools/call`/`server/discover` | ✓ WIRED | Live wire transcript (`index-appears-mid-session.golden`) shows the transition on the very next call, no restart |
| `ServerOptions.Instructions` | both `initialize` and `discover` results | single SDK field | ✓ WIRED | Identical `instructions` string present on `legacy-2024-11-05.golden` (`initialize`) and `modern-discover-explore.golden` (`discover`) |
| `go-sdk`'s `validateRequestMeta` | `-32602`/`-32022` wire responses | no codegraph-go code (D-02/SPEC-02 finding) | ✓ VERIFIED (inherited, asserted) | `git diff --stat b069f09~1..d844e04 -- internal/mcp/` empty; frozen transcripts prove the codes fire correctly |

### T-03-19 Threat Verification (instructions constant, no interpolation)

- Unique `instructions` value across the entire 27-transcript corpus: `rg -o '"instructions":"[^"]*"' testdata/wireoracle/transcripts/*.golden | sort -u | wc -l` → **1**
- Host-path leakage: `rg -c '/Users/|/home/|/tmp/|/var/folders' testdata/wireoracle/transcripts/*.golden | rg -v ':0$'` → **zero matches** (rg exit 1, meaning no line had a non-zero count)
- Source: `instructions` is declared as a Go string `const` in `internal/mcp/server.go` (not a `var`, not built via `Sprintf`/concatenation) — verified by reading the declaration directly.

### Behavioral Spot-Checks / Automated Test Runs

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Wire oracle full suite | `go test ./test/wireoracle/... -count=1` | `ok ... 18.520s` | ✓ PASS |
| Scenario/transcript count agreement | `ExpectedScenarioCount` vs `ls *.golden \| wc -l` | 27 == 27 | ✓ PASS |
| `internal/mcp` package + archtest | `go test ./internal/mcp/... -count=1 -race` | `ok` (both subpackages) | ✓ PASS |
| Anchors re-captured fresh | `go test ./test/wireoracle/... -run TestSpecAnchorsHold -v` | all 27 scenario groups PASS, including the two SPEC-02 anchors and `assertDiscoverCacheControl` | ✓ PASS |
| Build/vet | `go build ./...` / `go vet ./...` | clean, exit 0 | ✓ PASS |
| VRFY-02 guard unaffected | `go test ./internal/mcp/archtest/... -run TestNoExternalProtocolVersionConstantReferences -v` | PASS | ✓ PASS |
| SPEC-02 zero-server-code claim | `git diff --stat b069f09~1..d844e04 -- internal/mcp/` | empty | ✓ PASS |
| SPEC-04 zero-public-cacheScope | `rg -c '"cacheScope":"public"' testdata/wireoracle/transcripts/*.golden \| rg -v ':0$'` | empty | ✓ PASS |
| D-03 guard advisory + still detects | `TRANSCRIPT_FREEZE_BASE=be53a2e task check:transcript-freeze` | reports the full 26-transcript / 2-source-file collision, **EXIT_CODE=0** | ✓ PASS (matches intended design — not a failure) |
| Re-freeze commit names causes | `git show -s --format=%B 0d3e448` | 3 named additive causes with per-cause transcript counts (23 / 1 / 1), zero unattributed lines, "Zero semantic changes" stated | ✓ PASS |
| Full workspace suite (single run) | `go test ./... -count=1` | All packages pass except `internal/daemon` (4 tests: `TestRunWatchdogCancelsRunOnSimulatedReparent`, `TestDaemonSharedWriter`, `TestDaemonRunWaitsForInFlightFlushBeforeReleasingLock`, `TestSoak`) | ⚠️ EXCLUDED — see below |

### Known and Excluded Condition — Observed, Not Silent

`go test ./... -count=1` (run once, this session) reproduced the documented pre-existing `internal/daemon` flake — 4 tests failed this run under full-suite load (a different subset than the "3-4" range description names, consistent with "a DIFFERENT subset each run"). Confirmed excluded: `git diff --stat be53a2e..HEAD -- internal/daemon/` is **empty** — no Phase 3 commit touches `internal/daemon`, and `internal/mcp` (the package Phase 3 modifies) has no dependency on it. This matches issue #17 / MAINT-02, explicitly scheduled for Phase 4. Not scored against Phase 3.

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|-------------|--------|----------|
| SPEC-01 | 03-01 | discover answers with capabilities, no tool call first | ✓ SATISFIED | `modern-discover-explore.golden` id=1 |
| SPEC-02 | 03-03 | `-32602`/`-32022` `_meta` validation | ✓ SATISFIED | Two frozen+anchored transcripts, zero server code |
| SPEC-03 | 03-01 | `resultType:"complete"` on every Modern tool result | ✓ SATISFIED | `modern-discover-explore.golden` id=2 |
| SPEC-04 | 03-01 | `ttlMs:0` + `cacheScope:"private"` | ✓ SATISFIED | Zero `"public"` hits corpus-wide |
| SPEC-05 | 03-04 | per-call `hasIndex` re-check | ✓ SATISFIED | `index-appears-mid-session.golden`, 5 new race-tested unit tests |
| SPEC-06 | 03-05 | Legacy client session + tool call | ✓ SATISFIED | `legacy-2024-11-05.golden` now 3 messages incl. `tools/call` |
| SPEC-07 | 03-05 | `instructions` on discover | ✓ SATISFIED | present on both `initialize` and `discover`, constant, no interpolation |
| SPEC-08 | 03-01 | `serverInfo` in `_meta` | ✓ SATISFIED | `modern-discover-explore.golden` ids 1 and 2 |

All 8 SPEC-01…08 requirements are marked `Complete` in `.planning/REQUIREMENTS.md`'s traceability table, and every one is independently confirmed above by live command output, not by trusting that table.

### Anti-Patterns Found

Scanned every file modified in this phase (`internal/mcp/server.go`, `internal/mcp/server_test.go`, `test/wireoracle/{scenarios,anchors,capture}.go`, `tools/transcriptfreeze/{classify,main,main_test}.go`, `Taskfile.yml`, `.github/workflows/ci.yml`) for `TBD|FIXME|XXX|TODO|HACK|PLACEHOLDER|not yet implemented|coming soon`.

Two "placeholder" hits, both explanatory prose about deliberate design decisions, not stubs:
- `internal/mcp/server.go:38` — "a literal placeholder is fine here" — refers to the pre-existing `version = "0.1.0"` constant (unrelated to Phase 3's scope; a real version isn't needed since clients don't gate on it).
- `test/wireoracle/scenarios.go:924` — "`ExpectTools: 0` is deliberate, not a placeholder" — a comment actively denying stub status for a scenario field.

No debt markers requiring follow-up references. No blockers found.

### Human Verification Required

None. Every must-have in this phase is either directly observable in frozen wire transcripts (which this verification re-ran and re-inspected independently) or in passing, live-executed Go tests. No visual, real-time, or external-service-dependent behavior is in scope for a stdio JSON-RPC server.

### Gaps Summary

None. All 5 ROADMAP success criteria and all 8 SPEC-01…08 requirements are verified against live command output and direct file inspection — not SUMMARY.md narrative. The D-03 guard's advisory exit-0 behavior and the `internal/daemon` full-suite flake are both intentional/pre-existing conditions correctly excluded per the verification brief, not scored as gaps.

---

_Verified: 2026-08-06_
_Verifier: Claude (gsd-verifier)_
