---
phase: 06-claude-code-pretooluse-nudge
plan: 07
subsystem: cli
tags: [claude-code, hooks, pretooluse, nudge, help-text, cli-reference, phase-gate]
status: complete

requires:
  - phase: 06-claude-code-pretooluse-nudge
    provides: 06-01..06-05 guard, classifier, cooldown, sticky opt-in, CLI flag; 06-06 D-18 verdict PASS
provides:
  - install/uninstall Long help describing the --pretool-nudge opt-in and its removal
  - docs/CLI-REFERENCE.md regenerated through task docs:cli (drift gate green)
  - Phase 6 gate record (build, suite, daemon alone, drift, mutation log, live verdict, untouched surfaces)
affects: [07-codex]

actuals:
  tokens: 5500
  tasks: 2
  commits: 1
plan_head_before: ff4c174c29d2d6dddb5fdc7265c805119b1d6088

tech-stack:
  added: []
  patterns:
    - "Help text that the generated reference greps for keeps each checked phrase on one source line"

key-files:
  created:
    - .planning/phases/06-claude-code-pretooluse-nudge/06-07-SUMMARY.md
  modified:
    - internal/cli/install.go
    - internal/cli/uninstall.go
    - docs/CLI-REFERENCE.md

key-decisions:
  - "Phase 6 06-07: install/uninstall help describe only what shipped and passed live (D-18 PASS); CLI-REFERENCE.md regenerated only via task docs:cli"
  - "Phase 6 06-07: the phase gate's mutation-family count is 19 (a1-a2, b1-b3, c1-c6, d1-d4, e1-e4) after the 06-05 amendment added (e4); every family carries a RED observation and a revert proof"

patterns-established: []

requirements-completed: [NUDGE-03, NUDGE-04, NUDGE-05, NUDGE-06]

duration: 5min
completed: 2026-09-19
---

# Phase 6 Plan 07: Help Text, Reference Regeneration and Phase Gate Summary

`install --help` now documents the Claude-only `--pretool-nudge` PreToolUse pointer: it never blocks a tool call, and the choice sticks until `--pretool-nudge=false` or uninstall. `uninstall --help` documents removing the hook and its guard. The CLI reference was regenerated through the drift gate, and the full Phase 6 gate is green.

## Performance

- **Duration:** about 5 min
- **Started:** 2026-09-19T10:58:12Z
- **Completed:** 2026-09-19T11:03Z
- **Tasks:** 2
- **Files modified:** 3 (plus this SUMMARY)

## Accomplishments

- `install` Long help: after the `--print-config-style` sentence, a new sentence describes the opt-in. The existing sentences are kept, and one Example line was added: `codegraph install --target claude --pretool-nudge`.
- `uninstall` Long help: after the skill-package sentence, it now says: "It also removes the Claude Code PreToolUse nudge hook and its guard script when present."
- `docs/CLI-REFERENCE.md` was regenerated with `task docs:cli`. Only the install and uninstall sections changed: 15 changed lines, within the plan's bound of 16.
- The phase gate is green. Exit codes are listed below.

## Task Commits

1. **Task 1: help text and reference regeneration** — `d01844c7` (docs)
2. **Task 2: phase gate and advisories.** No source change was needed, so the gate record is in this SUMMARY (metadata commit).

## Phase Gate (transcripts in /tmp/06-07-*.txt)

| Check | Result |
|-------|--------|
| `go build ./...` | exit 0 |
| Module suite minus `internal/daemon` (`/tmp/06-07-suite.txt`) | exit 0; 53 `ok` packages (bar ≥ 53); no `--- FAIL`/`FAIL` line; `internal/nudge` ok |
| `internal/daemon` alone (WINDOWS #37, `/tmp/06-07-daemon.txt`) | exit 0 (65.0 s); no goleak flake on this run |
| `task docs:cli:drift` (Task 1 `/tmp/06-07-drift.txt`, Task 2 `/tmp/06-07-drift2.txt`) | exit 0 both times: "byte-identical to a fresh regeneration" |
| `TestEveryRegisteredFlagIsAccountedFor`, `TestPlainGolden` (`/tmp/06-07-t1.txt`) | 2/2 PASS |
| Long-help width (≤ 80 columns) | pass |
| 06-MUTATION-LOG.md families | 19 sections (a1-a2, b1-b3, c1-c6, d1-d4, e1-e4). Each has a Pre-mutation gate, Mutation applied, Observed failure (RED), Pre-revert gate, Revert and Green control (19 of each heading) |
| 06-LIVE-SESSIONS.md | 0 `PENDING`; `D-18 verdict: PASS` |
| Untouched surfaces since the phase base `603efc95` | `internal/mcp`: only `skill_claims_drift_test.go` changed (a test, so the diff is not vacuous); `internal/agents/instructions.go`, `.claude/hooks/session-nudge.sh`, and both print-config-style goldens: empty diff |
| No `[ci skip]`/`[skip ci]` in any Phase 6 commit | pass |

## Deviations from Plan

**1. [Plan gate text] The mutation-family count is 19, not 18**
- The Task 2 verify asserts `rg -c '^## Family \([a-e][1-6]\)' = 18`. The 06-05 amendment (commit `6935aa3e`, one handler per block) added family (e4), so the log now has 19 sections. The literal `= 18` check fails for that reason alone.
- The property the check guards still holds. All 18 families the plan lists are present, (e4) is also present, and every section has its RED line and its revert proof. `d3201ba9` made the same post-amendment adjustment to the 06-06 gate. The check was not weakened: this SUMMARY records the count of 19.

**2. [Plan gate text] The untouched-surface base lookup resolves to an earlier milestone**
- `git log --grep='^test\(06-01\): ' | tail -1` resolves to `7095f81a` (2026-07-12, an earlier milestone's 06-01), which is the known pitfall from 06-04 and 06-05. Against `7095f81a^`, `internal/mcp` shows many production changes from earlier milestones, so the literal command fails.
- The same property was verified against this phase's base `603efc95` (the docs(06) plans commit) and holds. Only the drift test changed under `internal/mcp`, and the other four surfaces have an empty diff. The other checks were not weakened.

Every other check in both tasks passed as written.

## Advisories for the Maintainer (recorded, not changed here)

1. **CODEX-05 / ROADMAP Phase 7 criterion 3 wording.** The plan expected both to still say "once-per-session contract". Both already carry the D-05 wording: REQUIREMENTS.md CODEX-05 and ROADMAP.md Phase 7 criterion 3 read "first matched call, then at most once a minute per session and per subagent — the 2026-09-19 amendment of NUDGE-04". No `once-per-session` text remains in either file. No action is needed before Phase 7 plans.
2. **Unquoted hook command paths (D-12).** The mirrored shell form leaves the global command as an unquoted absolute path and the local one as an unquoted `${CLAUDE_PROJECT_DIR}` expansion. A path with spaces therefore produces a hook error for BOTH SessionStart and PreToolUse. The deferred exec-form/quoting idea should cover PreToolUse too.
3. **A committed local-scope guard carries the installing machine's ExecPath.** This is the same exposure as the local `.mcp.json`.
4. **Never run `install --location local --pretool-nudge` inside this repository.** It rewrites the embedded template. `TestRenderPreToolGuard/template_token_exactly_once` catches a committed rendering.
5. **Foreign (kept) Claude skill directory.** An opt-in written there is not recorded in a manifest, so Keep neither refreshes nor removes it. Uninstall still removes it.
6. **D-01a's "one allowlist line" became two.** D-01's two-level command gives both hidden commands a `--help` flag.
7. **The D-18 findings on the five open points (from 06-LIVE-SESSIONS.md) and what each means for CODEX-05:**
   - *additionalContext without a decision:* confirmed in Claude Code 2.1.278. Output with only `hookSpecificOutput.additionalContext` is delivered as a `hook_additional_context` attachment and does not change the tool call. For CODEX-05, the Claude side of the contract is proven. Codex's own hooks reference must be checked for the same "context-only" semantics; this finding cannot be assumed to carry over.
   - *Subagent session_id:* subagent hook calls carry the PARENT's `session_id`, and the per-(session, agent) key separates them (D-06). For CODEX-05, the Codex nudge needs its own subagent identity source, or it must state that a per-subagent cooldown does not apply there.
   - *CLAUDE_CODE_SESSION_ID export:* confirmed. The in-session value equals the transcript's session id. This is Claude-specific, and CODEX-05 must find Codex's equivalent.
   - *Same-command handlers with different `if`:* NOT deduplicated. Each block is evaluated independently, which validates the 06-05 one-handler-per-block shape. For CODEX-05, if Codex's `hooks.json` has a similar matcher/condition model, prefer one handler per block for the same ownership reason.
   - *stdin key order:* not relied on, because the subcommand parses with `encoding/json`. For CODEX-05, the same parser is reusable whatever key order Codex uses.
8. **`install --yes` discards `--target` is still open** (`.planning/todos/pending/2026-09-18-install-yes-discards-explicit-target.md`). It affects which targets the `--pretool-nudge` note (D-09) considers.

Carried forward from earlier plans:

9. **(06-04) `isOwned` is block-granular.** The PreToolUse registration was amended to one handler per block (06-05, `6935aa3e`), so a hand-edit of a single handler now duplicates the block instead of being overwritten. The 06-06 live session confirmed that same-command handlers with different `if` are not deduplicated.
10. **(06-04) A corrupt or unreadable Claude manifest loses the PreToolUse opt-in record** on the self-healing rewrite. Later plain installs then stop refreshing the guard. Uninstall still removes it, and re-running with `--pretool-nudge` re-records it.
11. **(06-05) Family (e4) goes RED at the shape/block-count assertion** before the hand-edit step. The overwrite behaviour itself was recorded by 06-04.
12. **(06-06) Grep/Glob blocks were not exercised live.** The maintainer's global Claude Code config exposes no Grep tool, so every live search went through Bash `rg`. The Grep/Glob blocks rest on `TestPreToolUseRegistrationShape`.
13. **Tooling.** `state.update-progress` warns that STATE.md has no `Progress:` body line (frontmatter progress is fine). Stale `.git/gsd-plan-head-before-*` markers from earlier milestones had to be removed. Plan-gate base lookups of the form `git log --grep … | tail -1` collide with earlier milestones' same-numbered plans (Deviation 2 above).

## Issues Encountered

None. The known `internal/cli` load flake and the `internal/daemon` goleak flake did not occur.

## Known Stubs

None.

## Threat Flags

None. The change is help text and the generated reference, both covered by T-06-31. T-06-32 is discharged by the gate table above.

## Next Phase Readiness

Phase 6 is closed: NUDGE-03..06 are complete. Phase 7 (Codex parity, CODEX-05) can reuse the proven contract and the D-18 findings above.

## Self-Check: PASSED

- FOUND: internal/cli/install.go, internal/cli/uninstall.go, docs/CLI-REFERENCE.md
- FOUND: d01844c7
