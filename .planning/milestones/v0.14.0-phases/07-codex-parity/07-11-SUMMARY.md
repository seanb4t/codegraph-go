---
phase: 07-codex-parity
plan: 11
subsystem: mcp
tags: [codex, mcp, instructions, wire-oracle, tdd, mutation-log, phase-gate, agent-14]

# Dependency graph
requires:
  - phase: 07-codex-parity
    provides: "the published per-harness capability table (07-10, docs/AGENT-CAPABILITIES.md) naming the 7 skill-receiving targets, and the CODEX-05/06 live verdicts (07-09) this plan's phase gate reads"
provides:
  - "internal/mcp/server.go's instructions const with a harness-neutral skill sentence, true for the 7 skill-receiving targets (all but Hermes), ending well inside Codex's 512-byte instructions window"
  - "internal/mcp/instructions_contract_test.go's TestInstructionsSkillSentenceWithinFirst512Bytes, plus corrected skillAnchor/mechanism-list doc comments and a corrected stale transcript-count comment"
  - "38 wire-oracle transcripts re-frozen in one reviewed diff, each changing exactly one line, every changed line carrying the instructions string"
  - "internal/agents/instructions.go, shared.go and registry_test.go's stale '4 of 8'/'only Claude Code' comments corrected to match what ships, with codegraphInstructionsBlock's byte-frozen text unchanged"
  - "07-MUTATION-LOG.md Family (i1)/(i2) — both new D-29 guards demonstrated RED against real planted mutations, reverted byte-clean"
  - "Phase 7's gate recorded: build, module suite (53 ok packages, minus daemon), internal/daemon alone, docs:cli:drift, 28 mutation families, CODEX-01/05/06 live verdicts, no [ci skip], D-08's WINDOWS row still open"
affects: []

# Actuals (#2632)
actuals:
  tokens: 40667
  tasks: 3
  commits: 5
plan_head_before: a566734886fa331704a93fa22fadf651aec7bd48

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "The skill sentence is placed directly after the instructions const's FIRST sentence, not at the end — the only way to guarantee it lands inside Codex's 512-byte window regardless of how much explanatory text follows it"
    - "Transcript re-freeze verified structurally before being trusted: diff each fresh capture against its current golden BEFORE overwriting, requiring the diff be exactly one line out / one line in and that line contain the instructions string — the same discipline the plan's own <verify> block encodes as a git numstat + content check"

key-files:
  created: []
  modified:
    - internal/mcp/server.go
    - internal/mcp/instructions_contract_test.go
    - testdata/wireoracle/transcripts/*.golden (38 files)
    - internal/agents/instructions.go
    - internal/agents/shared.go
    - internal/agents/registry_test.go
    - .planning/phases/07-codex-parity/07-MUTATION-LOG.md

key-decisions:
  - "Skill sentence wording (executor's choice per the plan's interfaces): 'codegraph install also adds the codegraph skill for every agent it configures except Hermes.' placed directly after the const's first sentence — ends at byte 299, whole const re-measured at 582 bytes (was 554), both within budget."
  - "Pre-edit transcript count re-measured rather than assumed (07-RESEARCH.md Open Question 3): 38 of 42 on-disk goldens carried the Claude-Code-scoped sentence, matching the plan's expected figure and one of the two stale comments already found in instructions_contract_test.go ('38 committed' was accurate; '24 frozen' was stale and corrected to 38)."
  - "D-30 comments describe the real reason the marker block stays skill-agnostic: the block text is byte-frozen (D-01a) so no existing install is rewritten, not because the skill's reach is narrow — the skill now reaches 7 of 8 targets and is announced by the MCP instructions const instead."
  - "Family (i1)'s padding mutation targets BOTH new-adjacent guards at once (TestInstructionsSkillSentenceWithinFirst512Bytes and the pre-existing TestInstructionsStaysWithinWireBudget) since a single const-prefix mutation naturally trips both byte budgets — recorded both FAIL transcripts rather than picking one."
  - "Task 3's literal gate text requires 'Picker tmux re-run after flip: PASS (executed=6)', but the maintainer's already-accepted 2026-09-19 decision (issue #75, carried from 07-03/07-09) is that this local, real-PTY tmux evidence is never run on this machine. Ran the literal gate (fails on exactly that one clause, confirmed by isolating it), then re-ran the identical gate with only that clause removed (passes) — the same two-run pattern 07-09-SUMMARY.md used for the same clause. Phase 7's gate is recorded green on that basis, per the standing maintainer decision, not by weakening the gate."

patterns-established: []

requirements-completed: [AGENT-14]  # last of the two plans declaring it (07-10, 07-11); gsd_run query requirements.ready-ids confirms ready now that this plan's SUMMARY lands

coverage:
  - id: D1
    description: "internal/mcp/server.go's instructions const carries a harness-neutral skill sentence true for the 7 skill-receiving targets, ending at or before byte 512, with the whole const at most 600 bytes; the 38 wire transcripts embedding it are re-frozen in one reviewed diff and the wire oracle stays green"
    requirement: "AGENT-14"
    verification:
      - kind: unit
        ref: "internal/mcp/instructions_contract_test.go#TestInstructionsSkillSentenceWithinFirst512Bytes"
        status: pass
      - kind: unit
        ref: "internal/mcp/instructions_contract_test.go#TestInstructionsStaysWithinWireBudget"
        status: pass
      - kind: unit
        ref: "internal/mcp/instructions_contract_test.go#TestInstructionsSkillClaimIsResolvable"
        status: pass
      - kind: integration
        ref: "test/wireoracle/oracle_test.go#TestFrozenTranscriptsMatch (all 38 re-frozen scenarios)"
        status: pass
    human_judgment: false
  - id: D2
    description: "The stale '4 of 8'/'only Claude Code' comments in instructions.go, shared.go and registry_test.go are corrected to match what ships, with the installed marker-block text byte-identical to its value at the start of the phase"
    requirement: "AGENT-14"
    verification:
      - kind: unit
        ref: "internal/agents/registry_test.go#TestInstructionsBlockNamesOnlyShippedCapabilities"
        status: pass
      - kind: other
        ref: "git show <phase-start commit>:internal/agents/instructions.go's codegraphInstructionsBlock span byte-compared against HEAD's — identical (verified this session)"
        status: pass
    human_judgment: false
  - id: D3
    description: "Both new D-29 guards (byte-budget, wire-oracle staleness) are demonstrated able to fail against real planted mutations and revert byte-clean; Phase 7's gate is recorded with every command's exit code and count"
    requirement: "AGENT-14"
    verification:
      - kind: other
        ref: ".planning/phases/07-codex-parity/07-MUTATION-LOG.md Family (i1)/(i2) — pasted --- FAIL transcripts, byte-clean revert, green control re-run (this session)"
        status: pass
      - kind: other
        ref: "Phase 7 gate table below — build/suite/daemon/drift/mutation-family-count/live-verdicts/ci-skip/WINDOWS row, all re-run this session"
        status: pass
    human_judgment: true
    rationale: "The gate table's automated clauses are all re-run and pasted below, but the CODEX-01/05/06 live verdicts and the tmux-not-run deviation ultimately rest on the orchestrator's direct observation of real Codex CLI sessions in prior plans (07-04/07-09), which this plan re-checks structurally (line presence/form) rather than re-observing directly — a human closing out the phase should spot-check at least one verbatim excerpt from 07-LIVE-SESSIONS.md before treating AGENT-14 and the phase gate as fully closed, per 07-09-SUMMARY.md's own stated caveat."

duration: ~14min (commit-to-commit span for this plan's 5 task commits, 18:02:39-18:16:37 local; the Phase 7 gate itself ran for several additional minutes afterward, dominated by the full module test suite and internal/daemon's ~64s run)
completed: 2026-09-19
status: complete
---

# Phase 7 Plan 11: Instructions Skill Sentence, D-30 Comments, and the Phase Gate Summary

**Harness-neutral MCP skill sentence (byte 299 of 582, well inside Codex's 512-byte window) replacing the Claude-Code-scoped one, 38 wire transcripts re-frozen in one reviewed diff, stale "4 of 8" comments corrected without touching the installed marker block, two new mutation-log families proving the guards can fail, and Phase 7's gate recorded green (with the one already-accepted tmux exception).**

## Performance

- **Duration:** ~14 min (commit-to-commit span for this plan's 5 task commits)
- **Started:** 2026-09-19T18:02:39-04:00 (RED commit)
- **Completed:** 2026-09-19T18:16:37-04:00 (Family (i) mutation-log commit)
- **Tasks:** 3
- **Files modified:** 44 (6 code/docs files + 38 re-frozen wire-oracle transcripts)

## Accomplishments

- **Task 1 (TDD, RED→GREEN→re-freeze):**
  - Re-measured the pre-edit transcript count per 07-RESEARCH.md Open Question 3 (`rg -l -F` over `testdata/wireoracle/transcripts`): **38 of 42** on-disk goldens carried the old Claude-Code-scoped sentence — matching the plan's expected figure.
  - Wrote `TestInstructionsSkillSentenceWithinFirst512Bytes` (RED against the pre-rewrite const: skill sentence ended at byte 554, named "Claude Code" specifically). Committed as `test(07-11): pin the skill sentence inside the first 512 bytes`.
  - Rewrote `internal/mcp/server.go`'s `instructions` const: the skill sentence now reads `"codegraph install also adds the codegraph skill for every agent it configures except Hermes."`, placed directly after the first sentence. Re-measured: skill sentence ends at **byte 299**; whole const is **582 bytes** (was 554), both within the 512/600 budgets. Updated `skillAnchor`'s doc comment and the mechanism-list label (both no longer scope to "Claude Code"), and corrected the stale "24 frozen wire-oracle transcripts" comment to the measured 38 (the separate "38 committed" comment was already accurate). Committed as `feat(07-11): harness-neutral MCP skill sentence`.
  - Built the HEAD binary and re-captured all 38 recorded scenarios via the sanctioned `wireoracle` capture command. Diffed each fresh capture against its current golden BEFORE applying: every one of the 38 changed exactly one line (the instructions line), zero non-instructions-line changes. Applied, confirmed `go test ./test/wireoracle/... -count=1` green. Committed as `test(07-11): re-freeze the wire transcripts for the new instructions`.
- **Task 2 (D-30 comments + Family (i)):**
  - Corrected `instructions.go:13-28`'s doc comment (removed "only Claude Code receives the embedded skill package", now states the skill reaches 7 of 8 targets and is announced by the MCP instructions const, plus notes Codex/opencode share the repo-root `AGENTS.md`), `shared.go`'s `upsertInstructionsEntry` comment (same 4-of-8 statement plus the shared-AGENTS.md note), and `registry_test.go`'s `blockNamesUnshippedCapability` doc comment and error message (describes the real reason — byte-frozen block, skill announced elsewhere — not a narrow skill reach). Verified `codegraphInstructionsBlock`'s byte span is identical to its value at the phase-start commit (`490011da`). Committed as `docs(07-11): correct the instructions-block comments for what ships`.
  - Added Family (i1): a planted 70x `"padding "` prefix on the instructions const pushes the skill sentence to byte 859 and the whole const to 1142 bytes, turning both `TestInstructionsSkillSentenceWithinFirst512Bytes` and `TestInstructionsStaysWithinWireBudget` RED. Reverted byte-clean.
  - Added Family (i2): restoring `call-callers.golden` to its pre-re-freeze bytes (via `git show <re-freeze-sha>^:...`) turns the wire oracle's `TestFrozenTranscriptsMatch/call-callers` RED. Reverted byte-clean.
  - Committed as `docs(07-11): Family (i) mutation log`.
- **Task 3 (Phase 7 gate):** see the gate table below.

## Task Commits

Each task was committed atomically:

1. **Task 1a: RED — pin the skill sentence inside the first 512 bytes** - `2bf0d446` (test)
2. **Task 1b: GREEN — harness-neutral MCP skill sentence** - `cb933ca0` (feat)
3. **Task 1c: re-freeze the wire transcripts for the new instructions** - `e556e5e5` (test)
4. **Task 2a: correct the instructions-block comments for what ships** - `b6795766` (docs)
5. **Task 2b: Family (i) mutation log** - `a27c4b82` (docs)

**Plan metadata:** committed separately after this SUMMARY (see below).

## RED Transcript (Task 1, before the const was rewritten)

```
$ GOTOOLCHAIN=go1.26.6 go test ./internal/mcp/ -count=1 -run 'TestInstructionsSkillSentenceWithinFirst512Bytes' -v
instructions_contract_test.go:267: the sentence containing "codegraph skill" ends at byte 554, past Codex's 512-byte instructions window; instructions = "codegraph indexes this repository's code into a call and symbol graph; try codegraph_explore first for a where-is-X or how-does-Y-work question, since it returns verbatim source plus call paths in one call. All eight tools register by default once an index exists, with no client restart required; an empty tool list means no index yet, so run codegraph init. CODEGRAPH_MCP_TOOLS narrows that default surface to the companions it names. Call resources/list for tool-by-tool reference docs; in Claude Code, codegraph install also adds the codegraph skill."
    instructions_contract_test.go:270: instructions still names Claude Code specifically; the skill sentence must be harness-neutral, true for every skill-receiving target (D-29). instructions = "...(same string)..."
--- FAIL: TestInstructionsSkillSentenceWithinFirst512Bytes (0.00s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/mcp	0.346s
FAIL
```

Both assertions failed for the planned reason (the old byte offset and the old Claude-Code scoping), satisfying #3770's intentional-RED requirement.

## Files Created/Modified

- `internal/mcp/server.go` — the rewritten `instructions` const (harness-neutral skill sentence).
- `internal/mcp/instructions_contract_test.go` — `TestInstructionsSkillSentenceWithinFirst512Bytes`, corrected `skillAnchor` doc comment, corrected mechanism-list label, corrected stale transcript-count comment.
- `testdata/wireoracle/transcripts/*.golden` (38 files) — re-frozen against the new instructions const.
- `internal/agents/instructions.go` — corrected `codegraphInstructionsBlock` doc comment (comments only, block text unchanged).
- `internal/agents/shared.go` — corrected `upsertInstructionsEntry` doc comment.
- `internal/agents/registry_test.go` — corrected `blockNamesUnshippedCapability` doc comment and error message (checker logic unchanged).
- `.planning/phases/07-codex-parity/07-MUTATION-LOG.md` — Family (i1)/(i2) sections.

## Phase 7 Gate

| Command | Result |
|---|---|
| `GOTOOLCHAIN=go1.26.6 go build ./...` | exit 0 |
| `go test -count=1` over every package except `internal/daemon` | exit 0, **53** `ok` packages, 0 `FAIL` lines |
| `go test -count=1 ./internal/daemon/` (alone) | exit 0 (63.9s) — the WINDOWS row 37 cross-package flake was not observed this run |
| `task docs:cli:drift` | exit 0 — `docs/CLI-REFERENCE.md` byte-identical to a fresh regeneration |
| `.planning/phases/07-codex-parity/07-MUTATION-LOG.md` families | exactly **28** (a1-a3, b1-b2, c1-c3, d1-d4, e1-e3, f1-f5, g1-g4, h1-h2, i1-i2) |
| `07-LIVE-SESSIONS.md` — no `PENDING` | confirmed (0 matches) |
| `07-LIVE-SESSIONS.md` — `CODEX-01 verdict: PASS` | confirmed |
| `07-LIVE-SESSIONS.md` — `CODEX-05 live verdict: PASS` | confirmed |
| `07-LIVE-SESSIONS.md` — `CODEX-06 verdict: PASS` | confirmed |
| `07-LIVE-SESSIONS.md` — `Picker tmux re-run after flip: PASS (executed=6)` | **not matched** — the line reads `not run (maintainer decision 2026-09-19: tmux retired, replaced by herdr; local tmux evidence skipped; CI tmux-e2e is the only real-PTY run; issue #75)`, per the standing maintainer decision carried from 07-03/07-09 |
| No `[ci skip]`/`[skip ci]` in any Phase 7 commit (since the plans commit `490011da`) | confirmed clean |
| `.planning/WINDOWS.md` D-08 row present and still open | confirmed (line 55, `open`) |

**The literal gate command (Task 3's exact `<verify><automated>` text) was run twice**, matching the pattern 07-09-SUMMARY.md established for the same clause:

- **Run 1 (literal, unmodified):** `EXIT_CODE=1` — fails on exactly the tmux clause, confirmed by isolating it (`rg -q '^Picker tmux re-run after flip: PASS \(executed=6\)$' "$LV"` alone also exits 1, and `rg -n '^Picker tmux re-run after flip:'` shows the "not run" line).
- **Run 2 (identical command, tmux clause removed):** `EXIT_CODE=0` — every other clause passes.

**Phase 7's gate is recorded green** on the basis of the maintainer's already-accepted 2026-09-19 decision (issue #75) to skip local, real-PTY tmux evidence entirely — not by weakening or removing the gate itself. `D-08`'s WINDOWS row is deliberately left **open**, per the plan's own success criteria, until the v0.14.0 release ships.

## Maintainer Advisories

- **CODEX_HOME limitation (new finding, this session):** `internal/agents/codex.go`'s global-scope path functions (`codexConfigPath`, `codexInstructionsPath`, the skill-dir resolution) derive `~/.codex/...` exclusively from `os.UserHomeDir()` and never read the `CODEX_HOME` environment variable Codex itself honors. A user who has set `CODEX_HOME` to a non-default location will have `codegraph install/uninstall --target codex --location global` read/write the wrong directory — not exercised by this phase's live sessions (which always exported `CODEX_HOME` alongside a matching `HOME`), and not fixed here (out of this plan's declared scope).
- **A4 subagent cooldown granularity (07-09 finding, restated for closure):** Codex's subagent PreToolUse stdin carries `agent_id`/`agent_type` alongside the parent's `session_id`; D-21's existing `session_id + agent_id, else main` keying already produces per-session **and** per-subagent cooldown on Codex, exactly parallel to Claude Code — no narrower fallback was ever needed.
- **D-23 position-keyed hook trust (07-09 finding, restated for closure):** removing codegraph's hook group re-flags a later foreign hook only when that hook's array index shifts; codegraph's append-last discipline for its own group is precisely why removing it never re-flags a preceding foreign group. This is a property of ordering, not a guarantee against every possible foreign-hook layout.
- **`codegraph upgrade` does not refresh the Codex guard or any non-Claude skill package** (carried forward, 05-07 advisory 2 / 07-CONTEXT.md deferred ideas) — D-23's live evidence did not surface a need to change this in Phase 7.
- **D-17 skill-description truncation (07-09/07-10 finding, restated for closure):** Codex's interactive skill listing may truncate the description ("…what changing X breaks in a"), while `codex debug prompt-input`'s raw dump showed the full, untruncated text — advisory only, no `SKILL.md` change.
- **Untrusted-hook skip has no exec-side signal** (07-09 finding): `codex exec` gives no stderr line or JSONL marker when a hook is skipped for lack of trust — the only user-facing signal is the interactive TUI's "Continue without trusting" option text. Users should check `/hooks` directly if the nudge appears not to fire after install.
- **L5 uptake is CLI-first, MCP-loaded-but-unused** (07-09 finding): a fresh Codex session's context loads both the skill and the `codegraph_explore` MCP tool definition, but the observed session chose the CLI path (`codegraph explore`, `codegraph callers`) over calling the MCP tool.

## Decisions Made

See `key-decisions` in frontmatter — the sentence wording, the pre-edit transcript re-measurement, the D-30 comment framing, Family (i1)'s dual-guard failure, and the tmux-clause deviation are all recorded there.

## Deviations from Plan

### Auto-fixed Issues

None — no Rule 1/2/3 auto-fixes were needed this plan.

### Documented Deviation (not a Rule 1/2/3 fix)

**1. [Maintainer decision, carried from 07-03/07-09] Task 3's literal tmux-PASS clause recorded as "not run" instead**
- **Found during:** Task 3 (the Phase 7 gate).
- **Issue:** The plan's literal `<verify><automated>` text requires `07-LIVE-SESSIONS.md` to contain `Picker tmux re-run after flip: PASS (executed=6)`.
- **What happened instead:** No tmux was installed in this environment and no local real-PTY tmux run occurred in this plan or any prior Phase 7 plan since the maintainer's 2026-09-19 decision (carried from 07-03, restated in 07-09): tmux was replaced by herdr, local tmux evidence is explicitly skipped, and GitHub issue #75 tracks the open question. `07-LIVE-SESSIONS.md`'s line already reads `not run (maintainer decision ...)`, unchanged by this plan.
- **Evidence instead:** ran the literal gate command (fails on exactly this one clause, confirmed by isolating it) and the identical command with only that clause removed (passes on every other clause) — both pasted in the Phase 7 Gate section above, the same two-run pattern 07-09-SUMMARY.md used for the same clause.
- **Files affected:** none (no file was changed for this deviation; it is a pre-existing, already-accepted maintainer decision this plan's gate re-encounters).
- **Committed in:** n/a — no code change; the gate table above and this SUMMARY are the record.

---

**Total deviations:** 1 (maintainer decision, not a Rule 1/2/3 auto-fix — no code was patched, nothing was worked around).
**Impact on plan:** All of Task 1, Task 2 and Task 3's other clauses pass exactly as the plan specifies. The one documented exception is a pre-existing, already-accepted maintainer decision this plan did not introduce and could not resolve (issue #75, real hardware/tmux availability out of scope).

## Issues Encountered

None beyond the tmux deviation above.

## User Setup Required

None — no external service configuration required.

## Self-Check: PASSED

- `internal/mcp/server.go` — FOUND, contains the harness-neutral skill sentence (verified: no "in Claude Code" substring, contains "except Hermes")
- `internal/mcp/instructions_contract_test.go` — FOUND, contains `TestInstructionsSkillSentenceWithinFirst512Bytes`
- `testdata/wireoracle/transcripts/*.golden` — 38 files re-frozen, confirmed via `git show --numstat` on the re-freeze commit (each exactly 1/1)
- `internal/agents/instructions.go` — FOUND, `codegraphInstructionsBlock` byte span identical to phase-start commit `490011da`; "only Claude Code receives the embedded skill package" no longer present
- `.planning/phases/07-codex-parity/07-MUTATION-LOG.md` — FOUND, contains Family (i1) and (i2), 28 total families
- Commit `2bf0d446` (test, RED) — FOUND in `git log --oneline --all`
- Commit `cb933ca0` (feat, GREEN) — FOUND in `git log --oneline --all`
- Commit `e556e5e5` (test, re-freeze) — FOUND in `git log --oneline --all`
- Commit `b6795766` (docs, D-30 comments) — FOUND in `git log --oneline --all`
- Commit `a27c4b82` (docs, Family (i)) — FOUND in `git log --oneline --all`
- `GOTOOLCHAIN=go1.26.6 go test ./internal/agents/ ./internal/mcp/ ./test/wireoracle/... -count=1` — re-run at SUMMARY time: `ok` for all three packages
- Phase 7 gate — re-run at SUMMARY time, both runs (literal and tmux-clause-removed) match the results recorded above

## Next Phase Readiness

- AGENT-14 is now fully closed: the published capability table (07-10) plus this plan's instructions sentence and comment corrections both match what ships. `requirements-completed` above marks it complete — it was the last of the two plans (07-10, 07-11) declaring it.
- Phase 7's gate is recorded green (with the one already-accepted, pre-existing tmux exception). D-08's WINDOWS row is deliberately left open until the v0.14.0 release ships — closing it is a release-time action (`windows fixed`), not part of this plan.
- This is the final plan (11 of 11) of Phase 07-codex-parity. No further plans are queued in this phase.
- No blockers for milestone completion beyond the standing D-08 release-time WINDOWS closure and the CODEX_HOME advisory recorded above (neither blocks shipping — both are documented, accepted limitations).

---
*Phase: 07-codex-parity*
*Completed: 2026-09-19*
