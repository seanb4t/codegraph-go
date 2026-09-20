---
phase: 06-claude-code-pretooluse-nudge
plan: 06
subsystem: testing
tags: [claude-code, hooks, pretooluse, nudge, live-session, d-18, herdr, cooldown, subagent]
status: complete

requires:
  - phase: 06-claude-code-pretooluse-nudge
    provides: 06-01..06-05 guard, cooldown, local install with --pretool-nudge, six single-handler PreToolUse blocks
provides:
  - 06-LIVE-SESSIONS.md with D-18 evidence: C1-C7 PASS, D-18 verdict PASS
  - recorded fire rate, true-positive share, uptake and hook wall time from fresh live sessions
  - the five hooks-reference points settled live (input to CODEX-05 in Phase 7)
affects: [06-07, 07-codex]

actuals:
  tokens: 8600
  tasks: 3
  commits: 2
plan_head_before: d3201ba96dbb6d7774ed93a4b112f802e299bcd0

tech-stack:
  added: []
  patterns:
    - "Live-session evidence: executor scaffolds, orchestrator drives Herdr panes, executor gates completeness and commits"
    - "Every absence claim is backed by a search first shown to find the thing when present"

key-files:
  created:
    - .planning/phases/06-claude-code-pretooluse-nudge/06-LIVE-SESSIONS.md
  modified: []

key-decisions:
  - "Phase 6 06-06: D-18 verdict PASS (C1-C7 all PASS) in Claude Code 2.1.278; fire rate 9/18, 7/9 fires true positives"
  - "Phase 6 06-06: Claude Code does NOT deduplicate same-command PreToolUse handlers that differ only in `if`; each block is evaluated independently, validating the 06-05 one-handler-per-block shape"
  - "Phase 6 06-06: subagent hook calls carry the parent's session_id; the per-(session, agent) cooldown key is what separates them (D-06)"

patterns-established:
  - "Hook wall time is measured directly outside Claude Code; the in-session tool_use-to-fire gap includes the maintainer's other hooks"

requirements-completed: []

coverage:
  - id: L1
    description: "C1-C5 live in the indexed repo: first matched call fires once, no same-key fire within 60 s, fires again after the gap, subagent fires independently, fires after /clear"
    requirement: NUDGE-04
    verification:
      - kind: manual_procedural
        ref: "06-LIVE-SESSIONS.md C1-C5 lines + Task 2 automated verify (exit 0)"
        status: pass
    human_judgment: true
    rationale: "Live Claude Code behaviour (D-00) cannot be asserted by a unit test; excerpts were judged by the orchestrator"
  - id: L2
    description: "C6 un-indexed control 0 fires; C7 no hook error, prompt, deny or block from the hook across both sessions"
    requirement: NUDGE-03
    verification:
      - kind: manual_procedural
        ref: "06-LIVE-SESSIONS.md C6/C7 lines + Task 3 automated verify (exit 0)"
        status: pass
    human_judgment: true
    rationale: "Attribution of denials to the hook command versus the maintainer's global hooks is a judgment over the debug log"
  - id: L3
    description: "Fire rate against matched calls measured in a fresh live session (9/18; 7/9 true positives)"
    requirement: NUDGE-05
    verification:
      - kind: manual_procedural
        ref: "06-LIVE-SESSIONS.md Matched calls / Fires / Fire rate / True-positive fires lines"
        status: pass
    human_judgment: true
    rationale: "Recorded, not gated (D-18); a measurement of a live agent"

duration: ~4h (including orchestrator-run sessions)
completed: 2026-09-19
---

# Phase 6 Plan 06: D-18 Live Sessions Summary

**D-18 passed in fresh Claude Code 2.1.278 sessions: all seven criteria PASS, 9 fires over 18 matched calls (7 true positives), zero fires in the un-indexed control, no hook errors or prompts, and the global config unchanged.**

## Performance

- **Duration:** about 4 h, most of it the orchestrator-run sessions and their 65 s waits
- **Started:** 2026-09-19T10:40:53Z (scaffold commit)
- **Completed:** 2026-09-19T10:57Z
- **Tasks:** 3 (Task 1 executor; Tasks 2 and 3 orchestrator, then the executor completeness gate)
- **Files modified:** 1

## Verdict

| Line | Result |
|------|--------|
| C1 first matched call fires once | PASS |
| C2 no same-key fire within 60 s | PASS |
| C3 fires again after a >= 60 s gap | PASS |
| C4 subagent first matched call fires once | PASS |
| C5 fires after /clear | PASS |
| C6 un-indexed control | PASS (0 fires) |
| C7 no hook error, prompt, deny or block from the hook | PASS |
| **D-18 verdict** | **PASS** |
| Matched calls / Fires / Fire rate | 18 / 9 / 9/18 |
| True-positive fires | 7/9 |
| Bash rg path fired | yes (the `find` path fired too) |
| FP check (git log \| rg) | 0 fires |
| Global config unchanged | yes (settings.json sha256 identical; no global guard) |

No maintainer decision was needed: nothing failed.

## Key Live Findings

- **The maintainer's global config exposes no Grep tool.** Every search in Session A went through Bash `rg`, so the `Bash(rg *)` handler carried the evidence. The `Grep` and `Glob` blocks were not exercised live; their registration rests on `TestPreToolUseRegistrationShape`.
- **Uptake:** before `/clear` the agent kept using `rg` after each fire (it named `codegraph explore` in its commentary but did not call it). After `/clear` it answered the where-is-X prompt by calling `mcp__codegraph__codegraph_explore` directly, with no search.
- **Hook wall time:** median 3.2 ms (2.8-9.0) for the un-indexed guard, which only checks the directory; median 12.2 ms (11.4-13.0) for the indexed guard running the binary and firing. 20 runs each, measured outside Claude Code.
- **Same-command handlers are not deduplicated.** The three Bash blocks share a command and differ only in `if`. The debug log shows each evaluated on its own (`Skipping hook due to if condition "Bash(grep *)" not matching` while the `rg` handler ran; the `find` handler fired on the `find` call). This validates the 06-05 one-handler-per-block amendment.
- **Five points (feed CODEX-05):**
  - `additionalContext` without a decision: delivered as a `hook_additional_context` attachment and the tool call is unchanged.
  - Subagent session_id: subagents carry the parent's `session_id` and have their own `agentId`.
  - `CLAUDE_CODE_SESSION_ID`: exported in 2.1.278 and equal to the transcript's session id.
  - Same-command handlers with different `if`: not deduplicated, as described above.
  - stdin key order: not relied on, because `encoding/json` parses it.

## Task Commits

1. **Task 1: Scaffold scratch repos, pre-flight, hooks-reference quotes and evidence skeleton** - `537898f3` (docs)
2. **Task 2: ORCHESTRATOR Session A (indexed), C1-C5, metrics, five points** - recorded in `c1ef2baf`
3. **Task 3: ORCHESTRATOR Session B (un-indexed), C6, C7, verdict; executor completeness gate** - `c1ef2baf` (docs)

## Files Created/Modified

- `.planning/phases/06-claude-code-pretooluse-nudge/06-LIVE-SESSIONS.md` - D-18 evidence: pre-flight, hooks-reference quotes, scaffold, protocols, fire log, C1-C7 with excerpts, metrics, five points, global-config check

## Decisions Made

None beyond recording the verdict. See key-decisions for the facts the sessions settled.

## Deviations from Plan

These are protocol deviations recorded in 06-LIVE-SESSIONS.md. None weakens a criterion.

1. **Timing adjustments.** Prompt 2 ran 55-73 s after prompt 1's fire, so it straddled the cooldown boundary, which makes it stronger C2 evidence. Prompt 3 ran about 40 s after the previous fire, inside the window, so it added C2 evidence instead of C3; C3 was established by a later fire after a gap of more than 60 s.
2. **C4 and C5 re-run to be discriminating.** The first C4 attempt could not tell the subagent key from the main key, so it was re-run while the main key was still inside its window. C5 was re-run with an explicit `rg`, because after the first `/clear` the agent used `codegraph_explore` without searching.
3. **Workspace-trust entry.** Accepting the workspace-trust prompt (and the project MCP prompt) wrote a trust entry to `~/.claude.json`. `~/.claude/settings.json` was not changed.
4. **Grep/Glob not exercised live.** The prompts asked for the Grep tool, but this environment has none, so every search went through the Bash `rg` handler (see Key Live Findings).

The executor changed no content in 06-LIVE-SESSIONS.md; every verify regex passed unchanged.

## Verification

- Task 2 automated verify: exit 0
- Task 3 automated verify without the commit-subject check: exit 0 (before the commit)
- Task 3 full automated verify with the commit-subject check: exit 0 (after `c1ef2baf`)

## Requirements Touched

NUDGE-03, NUDGE-04 and NUDGE-05 now have live halves. They are not marked complete here; 06-07's phase gate closes them.

## Next Phase Readiness

06-07 can read `D-18 verdict: PASS` as its precondition.

## Self-Check: PASSED

- FOUND: .planning/phases/06-claude-code-pretooluse-nudge/06-LIVE-SESSIONS.md
- FOUND: 537898f3
- FOUND: c1ef2baf
