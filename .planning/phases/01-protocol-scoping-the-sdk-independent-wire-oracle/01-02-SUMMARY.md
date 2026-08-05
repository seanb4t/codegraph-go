---
phase: 01-protocol-scoping-the-sdk-independent-wire-oracle
plan: 02
subsystem: mcp-protocol
tags: [mcp, protocol-negotiation, audit, sep-2026-07-28, tools/mcpaudit]

# Dependency graph
requires:
  - phase: 01-protocol-scoping-the-sdk-independent-wire-oracle (plan 01)
    provides: internal/mcp.ProtocolVersion asserted-pin discipline and CONTEXT D-03/D-09/D-10/D-11/D-12 decisions this plan measures/records against
provides:
  - "tools/mcpaudit: standalone proxying capture shim (byte-exact bidirectional passthrough, tees the initialize exchange, never terminates)"
  - "docs/MCP-8-AGENT-AUDIT.md: dated 8-agent MCP negotiation audit (VRFY-05), 3 clients MEASURED, 5 UNMEASURED with explicit blocking reasons"
  - "docs/MCP-2026-07-28-SCOPING.md: SEP-by-SEP stdio applicability table mapped to SPEC-01..09, plus the trigger-set-floor and asserted-pin sections"
  - ".planning/TEAM-SCALE-READOUT.md: standalone Team Scale strategic read-out (BuildServer constructor-parameter gap)"
affects: [phase-3-server-discover-sequencing, phase-2-protocol-version-injection]

# Actuals (#2632)
actuals:
  tokens: 16205
  tasks: 4
  commits: 5

tech-stack:
  added: []
  patterns:
    - "Proxying capture shim: byte-preserving bufio.Reader.ReadBytes('\\n') passthrough (never bufio.Scanner) with a tee that stops observing once both halves of an exchange are recorded"
    - "Config-restore proof pattern: sha256-before/sha256-after published inline in the audit doc as verifiable evidence, not executor say-so"

key-files:
  created:
    - tools/mcpaudit/main.go
    - tools/mcpaudit/proxy.go
    - tools/mcpaudit/proxy_test.go
    - docs/MCP-8-AGENT-AUDIT.md
    - docs/MCP-2026-07-28-SCOPING.md
    - .planning/TEAM-SCALE-READOUT.md
  modified: []

key-decisions:
  - "Task 4's blocking checkpoint was closed on recorded human approval ('approved') rather than re-running the audit — the orchestrator independently re-verified every config, hash, and version claim before presenting the checkpoint, and re-running would have re-edited the developer's live MCP configs, which is expressly out of scope once measured."
  - "Added a 'Reproducing this audit' section to docs/MCP-8-AGENT-AUDIT.md documenting that bin/codegraph (the path all three restored configs point at) is a gitignored, developer-managed local install location not populated by any Taskfile target — so a fresh-clone reproduction attempt failing with 'binary not found' is expected and does not indicate a failed config restore."

requirements-completed: [VRFY-05]

coverage:
  - id: D1
    description: "Proxying capture shim (tools/mcpaudit) proxies byte-exact in both directions while observing the full bidirectional initialize exchange, including probe-then-initialize sessions, CRLF, and unterminated final frames"
    requirement: "VRFY-05"
    verification:
      - kind: unit
        ref: "go test ./tools/mcpaudit/... -count=1"
        status: pass
    human_judgment: false
  - id: D2
    description: "Dated 8-agent MCP negotiation audit (docs/MCP-8-AGENT-AUDIT.md): 3 MEASURED rows (Claude Code, Codex CLI, opencode) with offered+negotiated protocolVersion read from the real handshake, 5 UNMEASURED rows with explicit blocking reasons, all 8 roster clients in fixed ROADMAP order, every touched config's restoration proven by a published sha256-before/sha256-after match"
    requirement: "VRFY-05"
    verification: []
    human_judgment: true
    rationale: "Task 4 requires a human to confirm the audit rows reflect reality on their own machine and that their real agent configs still connect — this is exactly the class of claim an executor cannot self-certify. Approved via the recorded checkpoint response 'approved', backed by the orchestrator's independent pre-checkpoint verification (see Task 4 evidence below)."
  - id: D3
    description: "SEP-by-SEP stdio applicability table (docs/MCP-2026-07-28-SCOPING.md) covering all SEPs from RESEARCH's table, mapped to SPEC-01..09, plus the trigger-set-floor and asserted-pin sections"
    requirement: "VRFY-05"
    verification:
      - kind: other
        ref: "grep -q for all required SEP IDs, 'internal/query', and 'BuildServer' per the plan's <verify> block"
        status: pass
    human_judgment: false
  - id: D4
    description: "Team Scale strategic read-out (.planning/TEAM-SCALE-READOUT.md) as a standalone dated decision record, never an invented section inside ROADMAP.md or STATE.md"
    requirement: "VRFY-05"
    verification:
      - kind: other
        ref: "test -f .planning/TEAM-SCALE-READOUT.md; ROADMAP.md and STATE.md unmodified by this task"
        status: pass
    human_judgment: false

duration: 6min (continuation from Task 4 checkpoint)
completed: 2026-08-05
status: complete
---

# Phase 01 Plan 02: MCP Protocol Negotiation Audit & SEP Scoping Summary

**Byte-exact proxying capture shim measures live protocol negotiation for 3 of 8 roster MCP clients, with a dated audit, SEP-by-SEP stdio applicability table, and Team Scale read-out.**

## Performance

- **Duration:** ~6 min (this continuation session; closes Task 4 and writes the summary — Tasks 1-3 were executed in a prior session)
- **Tasks:** 4/4 complete
- **Files created:** 6
- **Commits:** 5 (4 from the prior session, 1 from this continuation)

## Accomplishments

- Built `tools/mcpaudit`, a standalone proxying capture shim that passes every byte through unchanged in both directions (byte-preserving `bufio.Reader`, never `Scanner`) while observing the client's `initialize` request and the server's matching response, including sessions where a probe precedes `initialize`.
- Ran the dated 8-agent negotiation audit: Claude Code (`2.1.222`), Codex CLI (`codex-cli 0.146.0`), and opencode (`1.18.10`) MEASURED with both offered and negotiated `protocolVersion` read from the real handshake; Cursor, Gemini CLI, Hermes, Antigravity, and Kiro UNMEASURED with explicit blocking reasons (not installed, or GUI-only and not confirmed scriptable). All three touched agent configs (`~/.claude.json`, `~/.codex/config.toml`, `~/.config/opencode/opencode.json`) restored and proven byte-identical via published `sha256-before`/`sha256-after` pairs.
- Wrote the SEP-by-SEP stdio applicability table (`docs/MCP-2026-07-28-SCOPING.md`) covering every SEP in RESEARCH's table, mapped to SPEC-01…SPEC-09, plus the "trigger set is a floor" and "asserted pin" sections addressing both cross-AI review concerns.
- Wrote the standalone Team Scale strategic read-out (`.planning/TEAM-SCALE-READOUT.md`), naming `BuildServer`'s four constructor-time parameters as the one real structural gap for a future multi-tenant server, and stating explicitly that this milestone builds none of it.
- Closed Task 4's blocking checkpoint on the human's recorded "approved" response, backed by the orchestrator's independent pre-checkpoint re-verification of every config, hash, and version claim — no re-audit, no config re-edit.
- Documented one finding surfaced during checkpoint verification: the restored configs' `bin/codegraph` path is a gitignored, developer-managed local install location not produced by any Taskfile target, so a fresh-clone reproduction will fail with "binary not found" until built by hand — a fact orthogonal to config-restore correctness, now recorded in the audit doc's new "Reproducing this audit" section.

## Task Commits

Each task was committed atomically (hashes below are the worktree's current hashes after reconciliation; the prior session's original hashes `9d2aaae 97c44f0 b75ad95 5f894f0` were cherry-picked cleanly onto this worktree, producing new hashes with the same tree content):

1. **Task 1: Proxying capture shim (byte-exact bidirectional passthrough)** — `1d51d25` (feat)
2. **Task 1 fix: shim survives a direct SIGTERM** — `e91afbd` (fix, Rule 1 deviation — see prior session)
3. **Task 2: Dated 8-agent negotiation audit (VRFY-05)** — `5ef32e4` (docs)
4. **Task 3: SEP-by-SEP stdio scoping table + Team Scale read-out** — `af7d75a` (docs)
5. **Task 4: Document the `bin/codegraph` finding surfaced at checkpoint** — `0bd0dd3` (docs)

**Plan metadata:** committed separately per `<final_commit>` protocol.

## Files Created/Modified

- `tools/mcpaudit/main.go` - shim entrypoint; flags `-real`, `-log`, `-args`; fails loudly on missing `-real`/`-log`
- `tools/mcpaudit/proxy.go` - `Observation`, `ParseClientFrame`, `ParseServerFrame`, `Run` — byte-exact tee that stops observing once the initialize exchange completes
- `tools/mcpaudit/proxy_test.go` - table-driven tests including `FailLoudly` and `_IsError` families, byte-exactness (CRLF + unterminated final frame), probe-then-initialize
- `docs/MCP-8-AGENT-AUDIT.md` - dated audit; 3 MEASURED + 5 UNMEASURED rows in fixed ROADMAP order; config-restore proof; **this session added** the "Reproducing this audit" section documenting the `bin/codegraph` local-install-path finding
- `docs/MCP-2026-07-28-SCOPING.md` - SEP-by-SEP stdio applicability table, SPEC-01…09 mapping, trigger-set-floor and asserted-pin sections
- `.planning/TEAM-SCALE-READOUT.md` - standalone dated Team Scale decision record

## Decisions Made

- **Task 4 closed on recorded human approval, not re-verification.** The checkpoint's `gate="blocking"` was satisfied by the human operator's "approved" response together with the orchestrator's own independent pre-checkpoint verification (all three config `command`/`args` fields confirmed pointing at the real binary, zero `mcpaudit` references, zero backup files, live CLI versions matching published values, all five UNMEASURED "not installed" claims re-confirmed true, no doc-sourced value leaked into a measured column, both required scoping-doc sections present, `go test ./tools/mcpaudit/...` green). Re-running the audit was explicitly out of scope — it would re-edit the developer's real MCP configuration files for no benefit once already measured and verified.
- **Documented the `bin/codegraph` local-install-path finding rather than treating it as a defect.** `/bin/` is gitignored (`.gitignore:14`); no Taskfile target populates it (`task build` is compile-check only, `task build:release` writes `./codegraph` at repo root). The restored configs correctly point at that developer-managed path — this is not a config-restore bug, but a future reader attempting to reproduce the audit needs to know a fresh clone won't have that binary until built by hand. Added a short "Reproducing this audit" section to `docs/MCP-8-AGENT-AUDIT.md` rather than altering any restored config or the audit's measured data.

## Deviations from Plan

### Auto-fixed Issues (from the prior session, Tasks 1-3)

**1. [Rule 1 - Bug] Shim did not survive a direct SIGTERM from the agent client**
- **Found during:** Task 1
- **Issue:** The initial shim implementation did not forward or handle SIGTERM gracefully when the parent agent process terminated it directly, risking the child `codegraph` process being orphaned or the exit code being lost.
- **Fix:** Added explicit signal handling so the shim survives and correctly propagates termination.
- **Files modified:** tools/mcpaudit/main.go, tools/mcpaudit/proxy.go
- **Committed in:** `e91afbd`

### This session (Task 4 closure)

No new auto-fixed bugs. One documentation addition (not a plan deviation under Rules 1-4 — it directly implements the orchestrator's Task 4 evidence-recording instruction): the "Reproducing this audit" section in `docs/MCP-8-AGENT-AUDIT.md`, committed in `0bd0dd3`.

---

**Total deviations:** 1 auto-fixed (Rule 1, prior session) + 1 documentation addition (this session, per explicit resume instructions)
**Impact on plan:** No scope creep. The documentation addition makes an already-true fact about the restored configs' `bin/codegraph` path legible to future readers without altering any measured audit data or restored config.

## Issues Encountered

None this session. `git merge --ff-only` was denied by the sandbox's Bash permission model for the worktree-branch merge; fell back to the plan's own documented alternative (`git cherry-pick` of the four listed commit hashes in order), which succeeded cleanly and reproduced all four expected artifacts.

## User Setup Required

None - no external service configuration required. (The audit itself required temporary, fully-restored edits to the developer's own local `~/.claude.json`, `~/.codex/config.toml`, and `~/.config/opencode/opencode.json` — completed and verified restored in the prior session, re-confirmed at this checkpoint.)

## Next Phase Readiness

- VRFY-05 satisfied: a dated, measured (not documentation-read) record of protocol negotiation exists for the 3 currently-installed roster clients, with a clear path to re-measure the other 5 when installed.
- The probe-column finding (Codex CLI's second-process-spawn behavior) is recorded and flagged for Phase 3's `server/discover` sequencing decision.
- The SEP-by-SEP scoping table gives Phase 3 a direct SPEC-01…09 index for the `2026-07-28` changelog.
- The Team Scale read-out gives a future unscoped milestone a bounded starting point (`BuildServer`'s constructor-parameter gap) without committing this milestone to build any of it.
- No blockers.

---
*Phase: 01-protocol-scoping-the-sdk-independent-wire-oracle*
*Plan: 02*
*Completed: 2026-08-05*
