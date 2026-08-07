---
phase: 01-protocol-scoping-the-sdk-independent-wire-oracle
plan: 01
subsystem: testing
tags: [mcp, jsonrpc, stdio, wire-protocol, golden-transcript, archtest-adjacent, mark3labs-mcp-go]

# Dependency graph
requires: []
provides:
  - "internal/mcp.ProtocolVersion — repo-owned protocol-version literal (asserted compatibility pin, VRFY-02 groundwork)"
  - "internal/mcp.Server seam (SDK-02) — internal/cli/serve.go bootstraps through mcp.NewStdioServer/s.ServeStdio(), no MCP SDK import"
  - "VRFY-03 always-on 'codegraph: mcp-session' stderr line, construction-guaranteed via NewStdioServer's nil-writer panic"
  - "test/wireoracle — the standalone wire-level capture package (spawn/write/scan, named-field normalization with a per-rule hit ledger, human-redirect CLI)"
  - "testdata/wireoracle/fixture — dedicated frozen discovery fixture, distinct from internal/indexer/testdata/gofixture"
  - "testdata/wireoracle/transcripts/handshake-explore.golden — first frozen pre-migration transcript, captured against today's unmodified mark3labs-backed binary"
  - "task test:wireoracle, wired into the test: wrapper and ci.yml's test job"
affects: [02-sdk-migration, 03-04-05-scenario-expansion, 06-anti-regeneration-guard, 07-mutation-matrix]

# Actuals (#2632)
actuals:
  tokens: 14799
  tasks: 1
  commits: 2

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Functional-options BuildServer(..., opts ...Option) — variadic addition keeps all 17 pre-existing positional call sites compiling unchanged"
    - "Named-field placeholder substitution over raw bytes (never JSON decode/re-encode) with a per-rule hit ledger, so a rule that silently stops matching fails loud as 'rule stopped firing' rather than an unexplained byte diff"
    - "Capture() owns its own lifecycle with no *testing.T dependency (kill-on-error/deadline, unconditional Wait via defer) so the same core serves both the in-suite test harness and the human-redirect CLI"

key-files:
  created:
    - internal/mcp/protocol_version.go
    - internal/mcp/session_line.go
    - internal/mcp/session_line_test.go
    - test/wireoracle/capture.go
    - test/wireoracle/normalize.go
    - test/wireoracle/scenarios.go
    - test/wireoracle/main_test.go
    - test/wireoracle/oracle_test.go
    - test/wireoracle/cmd/wireoracle/main.go
    - testdata/wireoracle/fixture/go.mod
    - testdata/wireoracle/fixture/main.go
    - testdata/wireoracle/fixture/pkga/pkga.go
    - testdata/wireoracle/fixture/pkgb/pkgb.go
    - testdata/wireoracle/transcripts/handshake-explore.golden
  modified:
    - internal/mcp/server.go
    - internal/cli/serve.go
    - Taskfile.yml
    - .github/workflows/ci.yml
    - internal/upgrade/taskfile_shape_test.go

key-decisions:
  - "BuildServer kept its return type (*server.MCPServer, unchanged) and only grew a variadic opts... parameter — avoids Pitfall 4's ripple entirely (RESEARCH flagged this as a real, budgeted risk); all 17 existing test call sites compile with zero edits. NewStdioServer is the new, sole production entrypoint returning the SDK-agnostic Server interface."
  - "toolCount for the session line is derived at the AddTool registration seam (incremented next to each call), never independently recomputed from hasIndex/allowlist, per the plan's explicit anti-duplication instruction."
  - "sanitizeClientField processes in this order: ToValidUTF8 (U+FFFD replacement) -> control/space -> '_' -> truncate on rune boundary -> '<unknown>' fallback for empty/all-stripped input. Verified with adversarial-input tests (embedded newlines, invalid UTF-8, 300-byte multi-byte-rune input)."
  - "repoDir rule's ExpectFires was set to the tracer capture's actual ledger value (false), not guessed: the handshake-explore scenario's tools/call result carries no path/file/root/repoPath JSON field (paths only appear inside the markdown text body under the 'text' key), so the rule legitimately does not fire on this capture. serverVersion fires once (every initialize response). timestamp does not fire. TestNormalizeRuleLedgerIsHonest enforces this stays true against the real binary."
  - "Golden transcript was read by a human (this session) before being trusted: the tools/call id=3 response decodes to isError:false with a real explore markdown result over pkga/pkga.go, main.go, pkgb/pkgb.go; no host temp-directory prefix survives into the frozen bytes."

patterns-established:
  - "Capture(ctx, binPath, fixtureSrc, workDir, Scenario) (Transcript, error) — the reusable spawn/write/scan core for every later wireoracle scenario; plans 03/04/05 add to Scenarios(), not to this function."
  - "Rule{Name, Placeholder, ExpectFires, Why} + NormalizeWithLedger — the pattern every future normalization rule must follow: field-name-anchored, ledger-honest, forward-compatible (a false ExpectFires needs a real Why, not a guess)."

requirements-completed: [VRFY-01, VRFY-03, VRFY-04, SDK-02]

coverage:
  - id: D1
    description: "Wire oracle spawns the real binary, drives a scripted initialize -> tools/list -> tools/call codegraph_explore session over real stdio, normalizes by named-field substitution, and byte-compares against a frozen transcript — green against today's unmodified mark3labs-backed serve --mcp"
    requirement: "VRFY-01"
    verification:
      - kind: integration
        ref: "go test ./test/wireoracle/... -run TestFrozenTranscriptsMatch"
        status: pass
    human_judgment: false
  - id: D2
    description: "The tracer's tools/call codegraph_explore supplies a real query argument and the captured result carries isError:false with non-empty content — the flagship tool's pre-migration baseline is a real result, not a rejected-argument error"
    requirement: "VRFY-04"
    verification:
      - kind: integration
        ref: "go test ./test/wireoracle/... -run TestTracerExploreCallSucceeds"
        status: pass
    human_judgment: false
  - id: D3
    description: "serve --mcp emits exactly one always-on 'codegraph: mcp-session' stderr line with requested/negotiated/client/tools keys in fixed order, with no flag or env var needed to enable it, and NewStdioServer refuses a nil session-log writer by construction"
    requirement: "VRFY-03"
    verification:
      - kind: integration
        ref: "go test ./test/wireoracle/... -run TestFrozenTranscriptsMatch (stderr session-line assertion)"
        status: pass
      - kind: unit
        ref: "go test ./internal/mcp/... -run TestNewStdioServerRejectsNilSessionLog"
        status: pass
    human_judgment: false
  - id: D4
    description: "internal/cli/serve.go bootstraps and serves entirely through mcp.NewStdioServer returning the mcp.Server interface; its direct import list names no MCP SDK package"
    requirement: "SDK-02"
    verification:
      - kind: other
        ref: "go list -f '{{join .Imports \"\\n\"}}' github.com/seanb4t/codegraph-go/internal/cli | grep -i mark3labs (zero matches)"
        status: pass
    human_judgment: false

duration: ~40min
completed: 2026-08-05
status: complete
---

# Phase 1 Plan 1: End-to-End Wire Oracle Tracer Summary

**One real `initialize` -> `tools/list` -> `tools/call codegraph_explore` MCP session captured over real stdio against the unmodified mark3labs-backed binary, normalized by named-field placeholder substitution, and frozen to a byte-compared golden transcript — plus the `mcp.Server` seam and the always-on `codegraph: mcp-session` stderr line it now observes.**

## Performance

- **Duration:** ~40 min
- **Completed:** 2026-08-05
- **Tasks:** 1 (tracer, tdd=true)
- **Files modified:** 19 (14 created, 5 modified)

## Accomplishments

- Built `test/wireoracle`, a standalone Go package that spawns the real `codegraph` binary, drives a scripted JSON-RPC session over real stdin/stdout, and never imports `mark3labs/mcp-go` or decodes into an SDK type (VRFY-01).
- Froze the first pre-migration transcript, `testdata/wireoracle/transcripts/handshake-explore.golden`, captured against today's unmodified `mark3labs`-backed `serve --mcp`, with a `tools/call codegraph_explore` result that succeeds (`isError:false`, real markdown content) rather than freezing an argument-validation error (VRFY-04).
- Added `internal/mcp.ProtocolVersion`, documented honestly as a Phase-1 asserted compatibility pin (mark3labs v0.56.0 exposes no `WithProtocolVersion` option), and a hand-authored spec anchor asserting the wire's negotiated `protocolVersion` equals it (VRFY-02 groundwork).
- Added the always-on `codegraph: mcp-session` stderr line (`requested=`, `negotiated=`, `client=`, `tools=` in fixed order), wired via `server.Hooks.AddAfterInitialize`, with all four client-supplied fields sanitized against log injection, and made "always on" a construction guarantee via `NewStdioServer`'s nil-writer panic rather than a calling convention (VRFY-03).
- Landed the `internal/mcp.Server` seam: `internal/cli/serve.go` now bootstraps entirely through `mcp.NewStdioServer(...).ServeStdio()` and no longer imports `github.com/mark3labs/mcp-go/server` (SDK-02).
- Wired `task test:wireoracle` into the `test:` wrapper and `ci.yml`'s `test` job, measured at ~5.8s wall-clock.

## Task Commits

Single tracer task, TDD-gated (RED -> GREEN):

1. **Task 1 (RED): wire oracle capture harness + `mcp.Server` seam** - `caafd20` (test) — production code, harness, and tests land together (tests need the production symbols to compile); `TestFrozenTranscriptsMatch` confirmed RED, failing by naming the missing golden file, never skipping. `TestTracerExploreCallSucceeds`, `TestNormalizeRuleLedgerIsHonest`, `TestCaptureIsDeterministic`, and `TestNewStdioServerRejectsNilSessionLog` already passed against the real binary at this commit.
2. **Task 1 (GREEN): freeze transcript + wire CI** - `df07ff3` (feat) — captured and froze `handshake-explore.golden` via the human-redirect CLI, confirmed `TestFrozenTranscriptsMatch` RED->GREEN with no test edit, added `task test:wireoracle` + CI step + `taskWrapperExpectedLegs` entry.

**Plan metadata:** (this commit, pending)

## Files Created/Modified

- `internal/mcp/protocol_version.go` - repo-owned `ProtocolVersion` literal, documented as an asserted compatibility pin
- `internal/mcp/session_line.go` - `formatSessionLine` + `sanitizeClientField` (VRFY-03/D-14, log-injection defense)
- `internal/mcp/session_line_test.go` - `TestNewStdioServerRejectsNilSessionLog` + `sanitizeClientField` adversarial-input tests (deviation, Rule 2 — see below)
- `internal/mcp/server.go` - `Server` interface, `mark3labsServer`, `buildConfig`/`Option`/`WithSessionLog`, `NewStdioServer`; `BuildServer` gains variadic `opts...`, wires the session-line hook, derives `toolCount` at the registration seam
- `internal/cli/serve.go` - bootstraps through `mcp.NewStdioServer(...).ServeStdio()`, drops the `mark3labs/mcp-go/server` import
- `test/wireoracle/capture.go` - `Capture`/`Transcript`/`Scenario`, the spawn/write/scan core, `*testing.T`-free lifecycle
- `test/wireoracle/normalize.go` - `Rule`/`Rules`/`NormalizeWithLedger`/`Normalize`, three field-anchored rules (repoDir, serverVersion, timestamp)
- `test/wireoracle/scenarios.go` - the `handshake-explore` scenario, `ScenarioByName`, `TranscriptPath`
- `test/wireoracle/main_test.go` - `TestMain` builds the real binary once
- `test/wireoracle/oracle_test.go` - `TestFrozenTranscriptsMatch`, `TestTracerExploreCallSucceeds`, `TestNormalizeRuleLedgerIsHonest`, `TestCaptureIsDeterministic`
- `test/wireoracle/cmd/wireoracle/main.go` - human-redirect capture CLI (`-bin`/`-fixture`/`-scenario`, ledger to stderr, transcript to stdout, no regeneration flag)
- `testdata/wireoracle/fixture/{go.mod,main.go,pkga/pkga.go,pkgb/pkgb.go}` - dedicated frozen discovery fixture (D-08)
- `testdata/wireoracle/transcripts/handshake-explore.golden` - the first frozen pre-migration transcript
- `Taskfile.yml` - `test:wireoracle` target + `test:` wrapper entry
- `.github/workflows/ci.yml` - "Test wire oracle (test/wireoracle)" step in the `test` job
- `internal/upgrade/taskfile_shape_test.go` - `taskWrapperExpectedLegs` gains `"test:wireoracle"`

## Rule Ledger (recorded verbatim, per plan Layer 11)

Captured with `go run ./test/wireoracle/cmd/wireoracle -bin <built binary> -fixture testdata/wireoracle/fixture -scenario handshake-explore`:

```
rule=repoDir hits=0
rule=serverVersion hits=1
rule=timestamp hits=0
```

`test/wireoracle/normalize.go`'s `Rules` declares `ExpectFires` matching this exactly: `serverVersion: true` (fires on every `initialize`); `repoDir: false` and `timestamp: false`, each with a `Why` explaining no current scenario exercises it — `repoDir` because the handshake-explore `tools/call` result carries no `path`/`file`/`root`/`repoPath` JSON field (the matched file paths appear only inside the markdown `text` body, which this rule deliberately does not touch — D-04's field-name anchoring), `timestamp` because no response in this scenario carries a `timestamp`/`time`/`ts` field. `TestNormalizeRuleLedgerIsHonest` enforces this stays true against the real binary going forward.

**Query used:** `"Alpha"` (the exported function in `testdata/wireoracle/fixture/pkga/pkga.go`).

**First 200 bytes of the explore result content:**
```
**Exploration: Alpha**\n\nFound 3 symbols across 3 files.\n\n**Blast radius — what depends on these (update/verify before editing)**\n\n- `Alpha` (pkga/pkga.go:15) — 1 caller in `pkga/pkga.go`; ⚠️
```

## Decisions Made

- **`BuildServer`'s return type unchanged, only `opts ...Option` added.** RESEARCH flagged the return-type-change ripple (Pitfall 4) as a real, budgeted risk across `server_test.go`/`markdown_test.go`/`reconnect_test.go`/`golden_parity_test.go` (17 call sites). Keeping the concrete `*server.MCPServer` return type and adding only a variadic option parameter avoided that ripple entirely — all 17 sites compile unmodified. `NewStdioServer` is the new, sole production entrypoint, returning the SDK-agnostic `Server` interface.
- **`toolCount` derived at the registration seam**, incremented next to each `AddTool` call, never independently recomputed — per the plan's explicit instruction to avoid duplicated state that drifts on a future registration-condition change.
- **`sanitizeClientField` order**: `strings.ToValidUTF8` (U+FFFD) -> control/space -> `_` -> rune-boundary truncation to 256 bytes -> `<unknown>` fallback. Verified against adversarial inputs (embedded `\n`/`\r`/`\t`, invalid UTF-8 bytes, a 300-byte multi-byte-rune string) to confirm the truncation never splits a rune.
- **`repoDir`'s `ExpectFires: false` was set from the tracer capture's actual ledger, not guessed** — matching the plan's explicit "do not guess" instruction and the acceptance criteria's note that a successful explore result is not guaranteed to carry a path-family field.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] Added `internal/mcp/session_line_test.go`**
- **Found during:** Task 1 implementation
- **Issue:** The plan's `<behavior>` block and acceptance criteria explicitly require `TestNewStdioServerRejectsNilSessionLog` to exist and pass (`go test ./internal/mcp/... -run 'TestNewStdioServerRejectsNilSessionLog' -count=1`), but this file is not listed in the plan frontmatter's `files_modified`.
- **Fix:** Added `internal/mcp/session_line_test.go` with `TestNewStdioServerRejectsNilSessionLog`, a companion `TestNewStdioServerAcceptsDiscard`, and `sanitizeClientField` unit tests (fail-loudly table + rune-boundary truncation case) — co-located with `session_line.go` per Go convention.
- **Files modified:** `internal/mcp/session_line_test.go`
- **Verification:** `go test ./internal/mcp/... -run 'TestNewStdioServerRejectsNilSessionLog' -count=1` passes.
- **Committed in:** `caafd20` (Task 1 RED commit)

---

**Total deviations:** 1 auto-fixed (1 missing critical — a required test file omitted from `files_modified`)
**Impact on plan:** Necessary to satisfy the plan's own acceptance criteria; no scope creep beyond what the plan already specified as required.

## Bootstrap Exception to D-03 (recorded per plan instruction)

This task creates `testdata/wireoracle/transcripts/handshake-explore.golden` in the same change that modifies `internal/mcp/server.go`, `internal/mcp/session_line.go`, and `internal/mcp/protocol_version.go` — the cross-change shape plan 06's anti-regeneration guard later declares unsafe. This is unavoidable and intended: the first transcript cannot pre-date the seam it captures, and the guard does not exist yet when this task runs. Plan 06 lands the guard afterward; plan 07's mutation matrix independently proves this bootstrap transcript is not vacuous. No exemption, allowlist, or bypass flag was added anywhere for this case.

## Flagged Planner Assumption Honored (VRFY-02 mechanism)

Per the plan's `<verification>` section, `internal/mcp.ProtocolVersion` is implemented and documented strictly as an **asserted compatibility pin** in Phase 1, not an injected value — `mark3labs/mcp-go v0.56.0` exposes no `WithProtocolVersion`-style option, confirmed by inspection of the vendored module source (`server/server.go`'s unexported `protocolVersion` method). The ROADMAP's stricter "reads from" wording is explicitly deferred to Phase 2 and not scored as met here; this plan does not edit ROADMAP.md or REQUIREMENTS.md.

## Issues Encountered

None — all builds, vets, and test runs succeeded on first or second attempt (one test-expectation typo in `session_line_test.go`, corrected before commit).

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- `test/wireoracle`'s `Capture`/`Normalize`/`Scenarios` core is ready for plans 03/04/05 to extend with the remaining ~22 scenarios (D-05) — no phase-conditional branching exists in the harness, matching the must-have that the Phase 2 oracle suite is the same file with the same scenario list.
- The frozen `handshake-explore.golden` and dedicated `testdata/wireoracle/fixture/` tree are the pre-migration baseline VRFY-04 requires exist before Phase 2 touches `go.mod` — captured and verified in this plan, `mark3labs/mcp-go v0.56.0` is still the pinned dependency.
- `internal/mcp.Server`/`NewStdioServer` is the seam Phase 2's SDK swap will add a second implementation behind; `internal/cli` needs no further changes to accept it.
- Plan 06 (anti-regeneration CI guard) and plan 07 (mutation matrix + scenario-count assertions) are the follow-on work this plan's bootstrap exception explicitly defers to.
- No blockers.

---
*Phase: 01-protocol-scoping-the-sdk-independent-wire-oracle*
*Completed: 2026-08-05*

## Self-Check: PASSED

All 14 created files verified present via `git ls-files --error-unmatch`. Both
commits (`caafd20`, `df07ff3`) verified present in `git log --oneline --all`.
