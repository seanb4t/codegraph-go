---
phase: 05-agent-reach-capability-model-skill-in-every-harness
plan: 06
subsystem: agents
tags: [live-sessions, herdr, opencode, antigravity, agy, cursor, gemini-md, evidence]

# Dependency graph
requires:
  - phase: 05-05
    provides: "every skill write path (shared .agents/skills for Cursor/opencode, harness dirs for Gemini/Kiro/Antigravity) in place to be probed live"
provides:
  - "05-LIVE-SESSIONS.md: live evidence and fixed verdict lines for Cursor (not probed), D-11 (not probed), opencode (read) and Antigravity (not read; GEMINI.md read)"
  - "D-11 verdict: not probed — 05-07 takes the no-change branch for Cursor's instructions target"
  - "Maintainer decision 1A — Antigravity's written skill dir moves to ~/.gemini/config/skills/codegraph (implemented in 05-07)"
affects: [05-07]

# Actuals (#2632) — chars/4 over the realized diff
actuals:
  tokens: 10300
  tasks: 3
  commits: 1
plan_head_before: cd9307819b8c8044bbab2d61c740dd1eed9ba473

tech-stack:
  added: []
  patterns:
    - "Harness discovery evidence comes from the harness's own loader when one exists (`opencode debug skill` JSON with per-skill `location`), not the agent's self-report of an already-deduplicated list"
    - "A directory-read claim is proven with a uniquely named probe skill placed in that directory, then removed"
    - "Cleanup on a real $HOME is judged on codegraph-owned paths plus recorded sha256 sums; the harness's own runtime state is classified, not reverted"

key-files:
  created:
    - .planning/phases/05-agent-reach-capability-model-skill-in-every-harness/05-LIVE-SESSIONS.md
  modified: []

key-decisions:
  - "No Cursor account: Cursor and the D-11 AGENTS.md probe are recorded `not probed` and Cursor stays [ASSUMED]; 05-07 takes the D-11 no-change branch"
  - "2A: opencode's duplicate-skill-name WARN (Claude project package + shared package) is accepted as an advisory — all copies are byte-identical; no guard on the shared write"
  - "1A: agy 1.2.6 reads user skills only from ~/.gemini/config/skills/, so Antigravity's written skill dir moves there in 05-07; ~/.gemini/antigravity-cli/skills/ stops being written"
  - "3A: the Antigravity cleanup gate is judged on codegraph-owned paths and recorded hashes; agy runtime state in the raw tree diff is classified, not reverted"

patterns-established:
  - "Live-session verdict lines use a fixed vocabulary that the plan gates grep; `not probed` is an allowed value only when a Maintainer decision line records why"

requirements-completed: [AGENT-09, AGENT-04, AGENT-06, AGENT-07, AGENT-10, AGENT-11]

coverage:
  - id: D1
    description: "opencode reads the shared project skill .agents/skills/codegraph without a frontmatter error; the skill is absent in the negative control; duplicate-name WARN recorded"
    requirement: AGENT-06
    verification:
      - kind: manual_procedural
        ref: "05-LIVE-SESSIONS.md ## opencode (opencode debug skill runs ×12 installed, ×8 control; probe skills)"
        status: pass
    human_judgment: true
    rationale: "Live-harness evidence recorded by the orchestrator; no automated test can claim a harness read (D-00)"
  - id: D2
    description: "Antigravity: codegraph's ~/.gemini/antigravity-cli/skills/codegraph is not read by agy 1.2.6; ~/.gemini/GEMINI.md is read; $HOME restored"
    requirement: AGENT-07
    verification:
      - kind: manual_procedural
        ref: "05-LIVE-SESSIONS.md ## Antigravity (negative control, positive session, GEMINI.md probe, cleanup)"
        status: pass
    human_judgment: true
    rationale: "Live-harness evidence against the real $HOME; the not-read outcome is resolved in 05-07 (1A), not here"
  - id: D3
    description: "Cursor, the D-11 AGENTS.md probe, Gemini CLI, Kiro and the global shared path are recorded [ASSUMED] / not probed with doc URLs and fetch date"
    requirement: AGENT-09
    verification:
      - kind: other
        ref: "Task 2/Task 3 <automated> verify gates in 05-06-PLAN.md (exit 0)"
        status: pass
    human_judgment: true
    rationale: "An [ASSUMED] row is an honest absence of evidence; the maintainer owns whether it is acceptable at phase close"

duration: 1h32m
completed: 2026-09-18
status: complete
---

# Phase 5 Plan 6: Live-Session Evidence Summary

**opencode reads the shared `.agents/skills/codegraph` package (with a log-only duplicate-name WARN). Antigravity 1.2.6 does not read the directory codegraph writes but does read `~/.gemini/GEMINI.md`. Cursor and D-11 could not be probed because there is no Cursor account. Every result is recorded in fixed verdict lines, and the maintainer's `$HOME` was restored.**

## Performance

- **Duration:** ~1h32m (end of 05-05 at 18:32 to the evidence commit at 20:04, including checkpoint escalations)
- **Completed:** 2026-09-18
- **Tasks:** 3 (Task 1 executor scaffold; Tasks 2 and 3 run by the orchestrator)
- **Files modified:** 1 (05-LIVE-SESSIONS.md), plus this SUMMARY

## Verdict lines (as recorded in 05-LIVE-SESSIONS.md)

| Line | Value |
|---|---|
| Cursor verdict | not probed |
| Cursor negative control | not probed |
| Cursor duplicate | not probed |
| Cursor where-is-X | not probed |
| **D-11 verdict** (consumed by 05-07) | **not probed** |
| opencode verdict | read |
| opencode negative control | skill absent |
| opencode duplicate | warning: duplicate skill name (log-level WARN, one entry kept, winner varies per run, all copies byte-identical) |
| opencode frontmatter | loaded without error |
| opencode where-is-X | codegraph first |
| Antigravity pre-flight readlink | ~/.claude/skills/codegraph -> ../../.agents/skills/codegraph; ~/.agents/skills/codegraph is a real directory |
| Antigravity negative control | skill absent |
| Antigravity verdict | not read |
| Antigravity where-is-X | codegraph first |
| GEMINI.md read | yes |
| GEMINI.md restored | byte-identical |
| Antigravity cleanup | before/after identical (codegraph-owned paths; raw tree diff is agy runtime state only) |
| Global shared path, Gemini CLI, Kiro | [ASSUMED] (doc URLs, fetched 2026-09-18) |

Maintainer decisions (five `Maintainer decision:` lines in the document): no Cursor account, so Cursor is not probed; D-11 is not probed for the same reason; 2A; 1A; 3A. See key-decisions.

## Key live findings

- **opencode's same-name dedup only appears in the log.** With Claude's project package, the shared project package and the maintainer's global `~/.agents/skills/codegraph` all present, opencode keeps one `codegraph` entry. The winning copy changes from run to run, and the only signal is a WARN at log level. The agent's self-report cannot show which directory was read, because it sees the list after dedup. `opencode debug skill` prints each resolved skill's `location` without a model call, so it became the discovery instrument. Two uniquely named probe skills proved that opencode reads both the project `.agents/skills/` and `.claude/skills/`. All three `codegraph` copies have the same sha256 (`e711379d…`).
- **agy 1.2.6 reads user skills only from `~/.gemini/config/skills/`.** It does not read `~/.gemini/antigravity-cli/skills/`, where codegraph currently writes. A renamed probe copy of SKILL.md (`cgprobe-agy-1789774582`) placed under `~/.gemini/config/skills/` was listed, which proved the point. The probe was then removed. This is the reverse of the D-06 correction (b) assumption that this path was "IDE only".
- **GEMINI.md is read live.** The QUINCE-2208 sentinel appended to `~/.gemini/GEMINI.md` came back from a fresh `agy`. The file was then restored and `cmp`-verified.
- **agy's runtime state means a raw tree diff is never empty.** Session brain/conversation dirs, log rotation, refreshed built-in skills and an MCP schema cache all change the tree. The cleanup is therefore judged on codegraph-owned paths and sha256 sums (3A). The sums diffs are empty, and `~/.claude/skills/codegraph` is still the same symlink.

## Task Commits

1. **Task 1: scaffold** — no commit. See Deviations.
2. **Task 2: Cursor/opencode/D-11 (orchestrator)** and **Task 3: Antigravity + completeness gate (orchestrator/executor)** — `ed52f925` docs(05-06): record live-session evidence for Cursor, opencode and Antigravity

Related orchestrator commits made at the checkpoints, before this plan's ledger started: `f83724ee` (no-Cursor decision, --yes todo), `abacd116` (gates accept `not probed`), `498172b4` (CONTEXT records 1A/2A/3A), `cd930781` (05-07 gains the 1A TDD task).

## Files Created/Modified

- `.planning/phases/05-agent-reach-capability-model-skill-in-every-harness/05-LIVE-SESSIONS.md`: per-harness commands, verbatim excerpts, negative controls, verdict lines, maintainer decisions, [ASSUMED] rows, summary table

## Decisions Made

See key-decisions. No code was changed in this plan. The 1A skill-dir move is recorded here and implemented in 05-07.

## Deviations from Plan

1. **Task 1 scaffold commit lost.** The first executor was killed by an API 429 after it wrote the scaffold and before it committed. The scaffold content survived in the untracked `05-LIVE-SESSIONS.md` and is included in `ed52f925`. No backdated `docs(05-06): scaffold …` commit was made, so Task 1's commit-subject check was never satisfied as written.
2. **Cursor not run.** The maintainer has no Cursor account, so the Cursor installed/control sessions and the D-11 AGENTS.md probe were not run. They are recorded as `not probed` with Maintainer decision lines, and Cursor stays [ASSUMED].
3. **Plan gates amended at the checkpoint.** `abacd116` changed the Task 2 and Task 3 verifies to accept `not probed` for Cursor/D-11. It also changed the cleanup line to the scoped form `before/after identical (codegraph-owned paths…)`, per 3A.
4. **opencode discovery instrument changed.** The orchestrator switched from the agent's self-report to `opencode debug skill` (see Key live findings). It also added two probe skills inside the scratch project and one temporary probe skill under `~/.gemini/config/skills/` for agy. All three were removed afterwards.
5. **agy version.** The plan names agy 1.1.11. The sessions ran on agy 1.2.6.

**Impact:** The evidence standard held: every verdict has a transcript or loader output plus a negative control, or an explicit `not probed`/[ASSUMED] with a recorded reason. Antigravity's `not read` was recorded and escalated, not patched (1A goes to 05-07).

## Advisories

- `codegraph install --yes` ignores an explicit `--target` and resolves to auto. Filed as `.planning/todos/pending/2026-09-18-install-yes-discards-explicit-target.md`. The scaffold avoided `--yes`.
- `codegraph uninstall --target antigravity --location global` left behind the now-empty `~/.gemini/antigravity-cli/skills/` that the install had created. It was removed by hand. Uninstall also deletes `mcp_config.json` once it becomes empty (remove-when-empty), so a file that previously held only the codegraph entry disappears and had to be restored from backup.

## Issues Encountered

None beyond the deviations above. Both verify gates (Task 2, and Task 3 including the commit-subject check) exit 0.

## Next Phase Readiness

05-07 can proceed. It consumes `D-11 verdict: not probed` (no Cursor instructions-target change) and implements the 1A Antigravity skill-dir move as a TDD task.

---
*Phase: 05-agent-reach-capability-model-skill-in-every-harness*
*Completed: 2026-09-18*

## Self-Check: PASSED

05-LIVE-SESSIONS.md, this SUMMARY and the --yes todo exist; commits ed52f925, f83724ee, abacd116, 498172b4, cd930781 exist; `git rev-list --count cd930781..HEAD` = 1 at SUMMARY write.
