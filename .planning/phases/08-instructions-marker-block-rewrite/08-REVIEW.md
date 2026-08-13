---
phase: 08-instructions-marker-block-rewrite
reviewed: 2026-08-13T20:45:53Z
depth: deep
files_reviewed: 42
files_reviewed_list:
  - internal/agents/instructions.go
  - internal/agents/registry_test.go
  - internal/mcp/instructions_contract_test.go
  - internal/mcp/server.go
  - testdata/wireoracle/transcripts/call-callees.golden
  - testdata/wireoracle/transcripts/call-callers.golden
  - testdata/wireoracle/transcripts/call-files.golden
  - testdata/wireoracle/transcripts/call-impact.golden
  - testdata/wireoracle/transcripts/call-node.golden
  - testdata/wireoracle/transcripts/call-search.golden
  - testdata/wireoracle/transcripts/call-status.golden
  - testdata/wireoracle/transcripts/error-confinement-reject.golden
  - testdata/wireoracle/transcripts/error-malformed-args.golden
  - testdata/wireoracle/transcripts/error-unknown-method.golden
  - testdata/wireoracle/transcripts/error-unknown-tool.golden
  - testdata/wireoracle/transcripts/handshake-explore.golden
  - testdata/wireoracle/transcripts/index-appears-mid-session.golden
  - testdata/wireoracle/transcripts/legacy-2024-11-05.golden
  - testdata/wireoracle/transcripts/legacy-2025-03-26.golden
  - testdata/wireoracle/transcripts/legacy-2025-06-18.golden
  - testdata/wireoracle/transcripts/legacy-2025-11-25.golden
  - testdata/wireoracle/transcripts/legacy-omitted-version.golden
  - testdata/wireoracle/transcripts/legacy-unsupported-2026-07-28.golden
  - testdata/wireoracle/transcripts/modern-discover-explore.golden
  - testdata/wireoracle/transcripts/resources-list-no-index.golden
  - testdata/wireoracle/transcripts/resources-list.golden
  - testdata/wireoracle/transcripts/resources-read-callees.golden
  - testdata/wireoracle/transcripts/resources-read-callers.golden
  - testdata/wireoracle/transcripts/resources-read-explore.golden
  - testdata/wireoracle/transcripts/resources-read-files.golden
  - testdata/wireoracle/transcripts/resources-read-impact.golden
  - testdata/wireoracle/transcripts/resources-read-index-state.golden
  - testdata/wireoracle/transcripts/resources-read-node.golden
  - testdata/wireoracle/transcripts/resources-read-search.golden
  - testdata/wireoracle/transcripts/resources-read-status.golden
  - testdata/wireoracle/transcripts/resources-read-tools-filter.golden
  - testdata/wireoracle/transcripts/resources-read-unknown.golden
  - testdata/wireoracle/transcripts/toolslist-default.golden
  - testdata/wireoracle/transcripts/toolslist-filter-empty.golden
  - testdata/wireoracle/transcripts/toolslist-narrowed.golden
  - testdata/wireoracle/transcripts/toolslist-no-index.golden
  - testdata/wireoracle/transcripts/toolslist-repeat.golden
findings:
  critical: 0
  warning: 2
  info: 1
  total: 3
status: issues_found
---

# Phase 8: Code Review Report

**Reviewed:** 2026-08-13T20:45:53Z
**Depth:** deep
**Files Reviewed:** 42 (4 Go source/test files + 38 frozen wire-oracle `.golden` transcripts)
**Status:** issues_found

## Summary

Phase 8 rewrote the MCP `instructions` wire const (`internal/mcp/server.go`) and the shared `codegraphInstructionsBlock` marker-fenced pointer (`internal/agents/instructions.go`) to retire a stale "Phase 3" deferral, and added drift-guard tests (`instructions_contract_test.go`, `registry_test.go`) plus a mechanical re-freeze of 38 wire-oracle golden transcripts.

Byte-level and structural verification against the four focus areas:

- **`instructions` const**: 554/600 bytes, single line, pure ASCII (byte-for-byte verified), and contains all five required anchors (`default`, `CODEGRAPH_MCP_TOOLS`, `codegraph init`, `resources/list`, `codegraph skill`). Confirmed via direct `len()` measurement, not just test-suite trust.
- **Claude-Code scoping**: the skill sentence is prefixed `"in Claude Code, codegraph install also adds the codegraph skill"` — correctly scoped, and the shared `codegraphInstructionsBlock` (used identically by all 4 marker-block targets: Claude, Codex, opencode, Gemini) contains no skill reference at all, enforced by `blockNamesUnshippedCapability`'s case-insensitive `"skill"` substring check. Verified all 4 per-target files (`claude.go`, `codex.go`, `opencode.go`, `gemini.go`) route through the identical `instructionsBody()`/marker-fence machinery with no per-target text divergence.
- **Marker-fence integrity**: `git diff` on `internal/agents/instructions.go` confirms `codegraphSectionStart`/`codegraphSectionEnd` are byte-unchanged — only the doc comment and one new bullet inside the block body changed.
- **Golden-file diff scope**: scripted verification (`git diff --numstat` per file) confirms all 38 changed `.golden` files have exactly one line removed and one line added each — the `instructions` JSON field and nothing else. The 4 frozen transcripts NOT re-touched (`edge-call-before-initialize`, `modern-listen-catalog-change`, `modern-meta-invalid-params`, `modern-meta-unsupported-version`) were confirmed to never carry an `instructions` field at all (pre-`initialize`-completion or meta-error paths), so their exclusion from the re-freeze is correct, not a gap.

The drift-guard tests (`TestInstructionsBlockNamesOnlyShippedCapabilities`, `TestInstructionsBlockGuardIsNotVacuous`, `resourcesClaimResolves`/`skillClaimResolves` + their non-vacuity table tests) were traced end-to-end and do discriminate: each was checked against a synthetic "would this catch a reversion" case, and the checkers take their subject as a parameter rather than reading package state, so a doc comment or the checker's own source text cannot make the gate pass vacuously.

One pre-existing issue was found via full-suite execution (not part of this phase's diff, but directly relevant to a focus area this review was asked to check): the `toolslist-repeat` wire-oracle scenario is measurably flaky, reproduced on this HEAD and independently reproduced on the pre-phase-8 base commit (`fa87a7a`) — see WR-01. This is not a regression introduced by Phase 8 (the const-only diff has no bearing on response ordering), but it was surfaced by the very risk area the review brief called out ("previously-flaky toolslist-repeat response-arrival-order scenario") and remains open.

## Warnings

### WR-01: `toolslist-repeat` wire-oracle scenario is flakingly order-dependent (pre-existing, not introduced by this phase, still open)

**File:** `testdata/wireoracle/transcripts/toolslist-repeat.golden` (comparison logic: `test/wireoracle/oracle_test.go`, response dispatch: `internal/mcp/server.go` `ServeStdio`/go-sdk `handleAsync`)
**Issue:** `go test ./test/wireoracle/... -run 'TestFrozenTranscriptsMatch/toolslist-repeat$' -count=1`, run repeatedly, fails intermittently:

```
oracle_test.go:130: scenario "toolslist-repeat": normalized transcript differs at line 2:
 got:  {"jsonrpc":"2.0","id":3,"result":{...tools/list result...}}
 want: {"jsonrpc":"2.0","id":2,"result":{...tools/list result...}}
```

The scenario sends `initialize(1)`, `toolsListRequest(2)`, `toolsListRequest(3)` back-to-back. `oracle_test.go`'s transcript comparison is line-positional (no per-response `id` re-sorting), but `go-sdk` dispatches each accepted request to its own handler goroutine independent of arrival order (`server.go`'s own `ServeStdio` doc comment describes this dispatch model in detail for a different problem). Two structurally-identical `tools/list` calls issued in immediate succession therefore have no guaranteed completion order, so their responses can legitimately reach stdout as `(id=3, id=2)` instead of the golden's `(id=2, id=3)`.

Verified not a phase-8 regression: reproduced independently on a worktree checked out at `fa87a7a` (pre-phase-8 base) — 2 failures in 15 runs at base, 1 failure in 17 runs at HEAD — consistent flake rate at both commits, and the phase's only change to this scenario's golden file is the `instructions` string (confirmed via `git diff --numstat`, exactly 1 line changed). This pre-dates the phase and was not fixed by it, despite the phase's own context notes explicitly flagging it as "the previously-flaky toolslist-repeat response-arrival-order scenario."

**Fix:** Two independent fixes address this at different layers — pick one:
1. In `test/wireoracle/oracle_test.go`'s comparison, treat a scenario's back-to-back-identical-method-pair responses as an unordered set keyed by `id` rather than a strict line sequence, when the scenario is explicitly testing repeat-call idempotency rather than ordering.
2. Or, if response ordering for same-connection sequential requests is meant to be a guaranteed MCP/JSON-RPC property, serialize dispatch of consecutive same-method calls (or at minimum document this as a known non-guarantee and mark the scenario `t.Skip`/retry-tolerant) rather than leaving a red/flaky assertion in the suite.

Given this is a CI-flakiness risk (test suite fails ~10% of the time on an unrelated, unmodified scenario) it should be tracked and fixed independently of this phase, but it is not blocking for Phase 8's own changes since Phase 8 did not touch the code paths responsible.

### WR-02: `instructionsMaxBytes`/anchor absolute-path denylist in `TestInstructionsCarriesNoWireContractViolation` is an incomplete token set

**File:** `internal/mcp/instructions_contract_test.go:261`
**Issue:** The denylist guarding against a future `const`→`var` downgrade that admits a host path is `{"/Users/", "/home/", "/private/", `C:\`}`. This misses other common absolute-path prefixes that would equally leak host filesystem layout into a committed wire-oracle transcript if ever introduced by mistake — e.g. `/tmp/`, `/var/`, `/root/`, `/opt/`, `\\` (UNC), or a bare drive-letter pattern like `D:\`. Since `instructions` is a compile-time string literal today, this guard is pure defense-in-depth against a hypothetical future refactor, so the gap is low-severity — but an incomplete denylist gives a false sense of completeness once (if) the const is ever loosened to a `var`.
**Fix:** Either broaden the token set (`/tmp/`, `/var/`, `/root/`, `/opt/`, a regex for `^[A-Za-z]:\\`) or, more robustly, replace the literal-token denylist with a `filepath.IsAbs`-style scan over whitespace-delimited tokens in `instructions`, which generalizes past any specific host's directory layout instead of enumerating known ones.

## Info

### IN-01: "optional path argument" guidance was dropped from the wire `instructions` const to make byte-budget room (deliberate, but worth confirming it isn't a net information loss)

**File:** `internal/mcp/server.go:56`
**Issue:** The rewrite deleted the sentence "Every tool accepts an optional path argument; omitting it uses this server's own working directory." to free ~101 bytes for the new `resources/list`/skill clauses (documented rationale: `08-RESEARCH.md` Pitfall 2, `08-01-SUMMARY.md` D-02). Confirmed this is not a net information loss: every individual tool's `inputSchema` in the `tools/list` response already documents `"path":{"type":"string","description":"Repo path (default: server cwd)"}` per-tool (visible in `handshake-explore.golden`), so an agent inspecting the tool catalog still learns the same fact, just from a different (arguably more precise, per-tool) location rather than the server-level prose. No fix needed — recorded for completeness since the trade-off wasn't self-evident from the diff alone.

---

_Reviewed: 2026-08-13T20:45:53Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep_
