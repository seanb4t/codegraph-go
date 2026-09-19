---
phase: 07-codex-parity
plan: 09
subsystem: codex
tags: [codex, live-evidence, mcp, hooks, trust, nudge, subagent, tmux]

# Dependency graph
requires:
  - phase: 07-codex-parity
    provides: "the HEAD binary's post-scope-flip `install --target codex` with `--pretool-nudge` (07-05/07-07/07-08), the CODEX-01 pre-flight baseline and Herdr-orchestrator method (07-04), and the re-anchored TTY-05 picker assertion (07-03)"
provides:
  - "CODEX-06 live verdict: PASS — a genuinely fresh Codex session in an indexed repo reaches for codegraph unprompted (skill listed, codegraph CLI run), entry shown at both project and global scopes"
  - "CODEX-05 live verdict: PASS — the nudge delivers additionalContext once then honors a per-key cooldown, stays silent in an un-indexed repo, and no hook error or block is attributable to codegraph's hook"
  - "A4 settled: Codex subagent PreToolUse stdin carries agent_id (UUIDv7) + agent_type alongside the parent's session_id; the main thread carries neither — D-21 keying (session_id + agent_id, else main) holds unchanged on Codex"
  - "D-23 settled: position-keyed hook trust re-flags a byte-identical foreign hook when its index shifts; codegraph appends its group last so its own removal never shifts a foreign group"
  - "Untrusted-hook behavior settled: Codex skips a new/changed hook until /hooks trust with no exec-side message, only the TUI's 'Continue without trusting' option text"
  - "FIX-03's local tmux re-run recorded as explicitly not-run (maintainer decision, 2026-09-19, issue #75) — the model-level footprint guard (07-03 Family c1/c2) is the FIX-03 evidence at this HEAD"
affects: [07-10, 07-11]

# Actuals (#2632)
actuals:
  tokens: 5986
  tasks: 3
  commits: 3
plan_head_before: 3efb3c1243eef153cbb95a5bd2bc31c50fbba5bc

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "A second, genuinely fresh Codex scratch (HOME/CODEX_HOME under /private/tmp/07-live2) reused the CODEX-01 isolation pattern (auth.json symlinked, never copied) to gather post-scope-flip evidence without any risk to the real ~/.codex"
    - "A foreign probe hook appended as the LAST PreToolUse group (after codegraph's own) doubles as both the A4 raw-stdin capture mechanism and the D-23 position-shift positive control"

key-files:
  created: []
  modified:
    - .planning/phases/07-codex-parity/07-LIVE-SESSIONS.md

key-decisions:
  - "CODEX-06 verdict: PASS — L1 post-flip (both scopes), L5 (fresh session reaches for codegraph unprompted) and L7 (real HOME unchanged) all PASS."
  - "CODEX-05 live verdict: PASS — L6 (one fire, 60s+ cooldown, un-indexed 0 fires, no hook error/block) and L7 both PASS."
  - "A4: Codex subagent PreToolUse stdin DOES carry a subagent identifier — agent_id (a UUIDv7 string) and agent_type ('default') — alongside the parent's session_id; the main thread has neither key. No escalation needed: D-21's session_id+agent_id (else main) keying already matches this shape without change."
  - "D-23: removing codegraph's hook group DOES re-flag a later foreign hook in /hooks when that hook's array index shifts (position-keyed trust: '…:pre_tool_use:1:0' becomes '…:pre_tool_use:0:0' and is shown 'Modified since last trusted' even though its bytes are unchanged). Because codegraph always appends its group LAST, removing codegraph's own group never shifts a foreign group that precedes it — the risk only appears when codegraph's group is removed from a position before an existing foreign group, which the append-last discipline prevents by construction."
  - "Untrusted-hook skip: Codex silently skips a new-or-changed hook until /hooks trust, with no exec-side message anywhere (stderr, JSONL) — the only user-facing statement is the interactive TUI's 'Continue without trusting (hooks won't run)' option text at the trust prompt."
  - "FIX-03's local tmux re-run is explicitly NOT performed (maintainer decision 2026-09-19, verbatim carried from 07-03: tmux was replaced by herdr; skip local tmux evidence, track via GitHub issue #75). Local FIX-03 coverage at this HEAD is the model-level footprint guard (07-03 Family c1/c2); CI's tmux-e2e job (ubuntu-latest, tmux 3.4) remains the only real-PTY run of the re-anchored TTY-05 assertion."
  - "L5 uptake evidence: the agent opened by naming the CodeGraph skill unprompted ('I'm using the CodeGraph skill because this repository is indexed'), read SKILL.md, then ran `codegraph explore` as its FIRST Bash call, followed by `codegraph callers` — zero grep/rg/find calls in the session. The MCP tool (`codegraph_explore`) was loaded into context but the CLI path was what the agent actually chose to use."

patterns-established: []

requirements-completed: [CODEX-05, CODEX-06, FIX-03]  # 07-09 is the last plan declaring all three (frontmatter requirements: fields across 07-01/07-03/07-07/07-08/07-09) — gsd_run query requirements.ready-ids reports all 3 ready now that this plan's SUMMARY lands

coverage:
  - id: D1
    description: "CODEX-06 verdict PASS: a genuinely fresh Codex session in an indexed repo reaches for codegraph unprompted (skill listed AND a codegraph call made), with the entry shown at both project and global scopes"
    requirement: "CODEX-06"
    verification:
      - kind: other
        ref: "07-09-PLAN.md Task 3 <verify><automated> gate — re-run at SUMMARY time with the single expected tmux-clause exception documented below (both runs pasted); L1-post/L5/L7 all PASS, verdict internally consistent"
        status: pass
    human_judgment: true
    rationale: "The automated gate verifies structural completeness of the recorded lines (no PENDING, correct forms, verdict consistent with its L-lines), not the truth of what Codex CLI actually did in the live interactive TUI and codex exec sessions. Evidentiary correctness rests on the orchestrator's direct observation of Herdr pane output and the session JSONL transcripts under /private/tmp/07-live2/, which a human consuming this for 07-10/07-11 should spot-check against at least one verbatim excerpt (e.g. the B5 L5 rollout quoted in 07-LIVE-SESSIONS.md) before treating it as final."
  - id: D2
    description: "CODEX-05 live verdict PASS: the nudge fires once then cools down for 60s+ per session/subagent key, stays silent in an un-indexed repo, and no hook error or block is attributable to codegraph's hook; A4 and D-23 settled from raw probe evidence"
    requirement: "CODEX-05"
    verification:
      - kind: other
        ref: "07-09-PLAN.md Task 3 <verify><automated> gate — same re-run as D1; L6/L7 PASS, A4 and D-23 lines present in allowed form with no Maintainer-decision escalation required (A4=yes)"
        status: pass
    human_judgment: true
    rationale: "Same reasoning as D1 — the gate checks line format and internal consistency, not the ground truth of the live fire-log timestamps, probe stdin captures, or /hooks TUI screens. A human should spot-check the Fire log table and at least one raw a4-*.json excerpt before relying on this for the docs plan."
  - id: D3
    description: "FIX-03's post-flip TTY-05 tmux re-run recorded as explicitly not-run per maintainer decision, with the model-level footprint guard (07-03) standing as the local evidence at this HEAD"
    requirement: "FIX-03"
    verification:
      - kind: other
        ref: "07-MUTATION-LOG.md Family (c1)/(c2) (07-03); this plan's Task 3 gate treats the tmux line's 'not run (maintainer decision …)' form as the sole documented deviation from the plan's literal PASS (executed=6) requirement, isolated and proven below"
        status: pass
    human_judgment: true
    rationale: "The literal plan text required a real-PTY PASS (executed=6); the maintainer explicitly waived local tmux evidence (2026-09-19, issue #75) before this plan's Task 3 ran. A human should confirm issue #75's resolution before treating FIX-03 as fully closed against a real local PTY — CI's tmux-e2e job is the only remaining real-PTY confirmation."

duration: ~15min (Task 1 scaffold commit to Task 3 evidence commit; orchestrator-driven Herdr/Codex sessions for Tasks 2-3 ran outside this timing)
completed: 2026-09-19
status: complete
---

# Phase 7 Plan 9: CODEX-05/06 Live Evidence (Post-Scope-Flip) Summary

**Settled live, post-scope-flip, in a second genuinely fresh Codex scratch: a fresh indexed-repo session reaches for codegraph unprompted at both scopes (CODEX-06 PASS), the PreToolUse nudge fires once then cools down and stays silent un-indexed with no hook error (CODEX-05 live PASS), Codex subagent PreToolUse stdin carries an agent_id/agent_type pair the main thread lacks (A4=yes), position-keyed hook trust re-flags a foreign hook when its index shifts (D-23=yes), and FIX-03's local tmux re-run is recorded as explicitly waived by the maintainer (issue #75) rather than silently skipped.**

## Performance

- **Duration:** ~15 min (commit-to-commit span for this plan's three task commits)
- **Started:** 2026-09-19T17:07:41-04:00 (Task 1 scaffold commit)
- **Completed:** 2026-09-19T17:22:38-04:00 (Task 3 evidence commit)
- **Tasks:** 3 (Task 1 executor-scaffolded; Tasks 2-3 orchestrator-driven via Herdr against a real Codex CLI, then executor-gated for completeness by this SUMMARY)
- **Files modified:** 1

## Accomplishments

- Built a second, genuinely fresh Codex scratch (`/private/tmp/07-live2`, `HOME`/`CODEX_HOME` isolated, `auth.json` symlinked never copied) with `indexed`/`unindexed`/`bare` repos, the HEAD binary's real post-scope-flip installs (`--pretool-nudge` local in `indexed`/`unindexed`, no-nudge global from `bare`), and an A4/D-23 probe hook appended as the LAST PreToolUse group in both repos' `hooks.json`.
- Ran the untrusted-hook check (`codex exec` before `/hooks` trust): 0 pinned-substring matches and no probe file written for that turn — Codex skips a new/changed hook silently pre-trust, with only the TUI's "Continue without trusting" option text as the user-facing statement.
- Trusted both repos through the real `/hooks` TUI, confirmed L1 post-flip at both project and global scopes via `codex mcp list --json` plus both `config.toml` files, and ran a fresh `codex exec` L5 session whose prompt never named codegraph: the injected context listed the `codegraph` skill and the `codegraph_explore` MCP tool definition, and the agent's first Bash call ran `codegraph explore` (zero grep/rg/find calls).
- Ran the L6 fire/cooldown protocol in ONE TUI session (fire at 21:16:51Z, silent 18s later, second fire at 21:18:10Z — 79.4s gap), the un-indexed control (0 fires while the probe proved hooks ran), the A4 subagent probe (raw stdin: `agent_id` UUIDv7 + `agent_type "default"` alongside the parent's `session_id`; absent on the main thread), the D-23 position-shift check (removing codegraph's group re-flagged the byte-identical foreign probe once its index shifted from 1 to 0), the uninstall (scratch repo left clean, probe group intact), and the L7 real-HOME checksums (all four unchanged).
- Recorded FIX-03's tmux re-run as **not run** per the maintainer's 2026-09-19 decision (tmux retired, replaced by herdr; issue #75) rather than the plan's literal `PASS (executed=6)` line — the sole documented deviation, isolated and proven below.
- Recorded all 16 CODEX-05/06 verdict lines in `07-LIVE-SESSIONS.md`; ran Task 3's exact `<verify><automated>` gate at SUMMARY time (fails, as expected, on exactly the tmux clause) and the same gate with only that clause removed (passes on every other clause).

## Task Commits

Each task was committed atomically:

1. **Task 1: Scaffold second scratch, real installs, A4/D-23 probe, pre-flight, 16-PENDING skeleton** - `13c67e71` (docs)
2. **Task 2 (orchestrator): trust, untrusted-hook check, L1 post-flip (both scopes), L5 uptake** - `5cafacd8` (docs)
3. **Task 3 (orchestrator + executor gate): L6 fire/cooldown, un-indexed control, A4, D-23, uninstall, tmux (not run), L7, verdicts** - `4819b6ec` (docs)

**Plan metadata:** committed separately after this SUMMARY (see below).

## Files Created/Modified

- `.planning/phases/07-codex-parity/07-LIVE-SESSIONS.md` — CODEX-05/06 section: scaffold, Protocol L1/L5 and L6/A4/D-23/uninstall/tmux evidence, the Fire log, and the 16-line verdict table (all settled, `CODEX-05 live verdict: PASS`, `CODEX-06 verdict: PASS`)

## Task 3 Completeness Gate — Re-run at SUMMARY Time

Re-ran the plan's exact `<verify><automated>` command from `07-09-PLAN.md` Task 3 before writing this SUMMARY. **This gate is expected to fail on exactly one clause** — `rg -q '^Picker tmux re-run after flip: PASS \(executed=6\)$' "$D"` — because the maintainer decided (2026-09-19) to skip local tmux evidence entirely (tmux retired, replaced by herdr; GitHub issue #75). The evidence line reads `Picker tmux re-run after flip: not run (maintainer decision 2026-09-19: tmux retired, replaced by herdr; local tmux evidence skipped; CI tmux-e2e is the only real-PTY run; issue #75)`, not the literal `PASS (executed=6)` the plan's gate demands.

**Run 1 — exact plan text, unmodified:**

```
$ cd /Volumes/Code/github.com/seanb4t/codegraph-go && D=.planning/phases/07-codex-parity/07-LIVE-SESSIONS.md && \
  ! rg -q 'PENDING' "$D" && rg -q '^L6 nudge fires once then cools down; un-indexed 0 fires: (PASS|FAIL \(.+\))$' "$D" && \
  rg -q '^L7 real HOME unchanged \(CODEX-05/06\): PASS$' "$D" && \
  rg -q '^A4 PreToolUse stdin carries a subagent id: (yes|no|not observed)( \(.+\))?$' "$D" && \
  rg -q '^tool_input\.command type: (string|array)( \(.+\))?$' "$D" && \
  rg -q '^D-23 removing our group re-flags later foreign hooks: (yes|no|not observed)( \(.+\))?$' "$D" && \
  rg -q '^Matched Bash search calls: [0-9]+$' "$D" && rg -q '^Fires: [0-9]+$' "$D" && \
  rg -q '^Uninstall leaves the scratch repo clean: (PASS|FAIL \(.+\))$' "$D" && \
  rg -q '^Picker tmux re-run after flip: PASS \(executed=6\)$' "$D" && \
  rg -q '^CODEX-05 live verdict: (PASS|FAIL)$' "$D" && rg -q '^CODEX-06 verdict: (PASS|FAIL)$' "$D" && \
  if rg -q '^CODEX-05 live verdict: PASS$' "$D"; then rg -q '^L6 [^:]+: PASS$' "$D"; else rg -q '^Maintainer decision: ' "$D"; fi && \
  if rg -q '^CODEX-06 verdict: PASS$' "$D"; then rg -q '^L1 post-flip [^:]+: PASS$' "$D" && rg -q '^L5 [^:]+: PASS$' "$D"; else rg -q '^Maintainer decision: ' "$D"; fi && \
  if rg -q '^A4 PreToolUse stdin carries a subagent id: (no|not observed)' "$D"; then rg -q '^Maintainer decision: ' "$D"; fi && \
  S=$(git log --format=%s -5) && printf '%s\n' "$S" | rg -q '^docs\(07-09\): record CODEX-05/06 live evidence'

EXIT_CODE: 1
```

Isolating the failing clause directly confirms it is the tmux line and nothing else:

```
$ rg -q '^Picker tmux re-run after flip: PASS \(executed=6\)$' "$D"; echo $?
1
$ rg -n '^Picker tmux re-run after flip:' "$D"
940:Picker tmux re-run after flip: not run (maintainer decision 2026-09-19: tmux retired, replaced by herdr; local tmux evidence skipped; CI tmux-e2e is the only real-PTY run; issue #75)
```

**Run 2 — same command with only the tmux clause removed:**

```
$ cd /Volumes/Code/github.com/seanb4t/codegraph-go && D=.planning/phases/07-codex-parity/07-LIVE-SESSIONS.md && \
  ! rg -q 'PENDING' "$D" && rg -q '^L6 nudge fires once then cools down; un-indexed 0 fires: (PASS|FAIL \(.+\))$' "$D" && \
  rg -q '^L7 real HOME unchanged \(CODEX-05/06\): PASS$' "$D" && \
  rg -q '^A4 PreToolUse stdin carries a subagent id: (yes|no|not observed)( \(.+\))?$' "$D" && \
  rg -q '^tool_input\.command type: (string|array)( \(.+\))?$' "$D" && \
  rg -q '^D-23 removing our group re-flags later foreign hooks: (yes|no|not observed)( \(.+\))?$' "$D" && \
  rg -q '^Matched Bash search calls: [0-9]+$' "$D" && rg -q '^Fires: [0-9]+$' "$D" && \
  rg -q '^Uninstall leaves the scratch repo clean: (PASS|FAIL \(.+\))$' "$D" && \
  rg -q '^CODEX-05 live verdict: (PASS|FAIL)$' "$D" && rg -q '^CODEX-06 verdict: (PASS|FAIL)$' "$D" && \
  if rg -q '^CODEX-05 live verdict: PASS$' "$D"; then rg -q '^L6 [^:]+: PASS$' "$D"; else rg -q '^Maintainer decision: ' "$D"; fi && \
  if rg -q '^CODEX-06 verdict: PASS$' "$D"; then rg -q '^L1 post-flip [^:]+: PASS$' "$D" && rg -q '^L5 [^:]+: PASS$' "$D"; else rg -q '^Maintainer decision: ' "$D"; fi && \
  if rg -q '^A4 PreToolUse stdin carries a subagent id: (no|not observed)' "$D"; then rg -q '^Maintainer decision: ' "$D"; fi && \
  S=$(git log --format=%s -5) && printf '%s\n' "$S" | rg -q '^docs\(07-09\): record CODEX-05/06 live evidence'

EXIT_CODE: 0
```

Result: the gate passes on **every clause other than the tmux line**, and the tmux line's failure is exactly and only the documented maintainer decision, not an unrecorded gap. Recorded as a deviation below, following the same pattern 07-03-SUMMARY.md used for Family (c3).

## CODEX-05/06 Verdicts (verbatim from 07-LIVE-SESSIONS.md)

```
L1 post-flip both scopes (codegraph-installed): PASS
L5 fresh session reaches for codegraph unprompted: PASS
L6 nudge fires once then cools down; un-indexed 0 fires: PASS
L7 real HOME unchanged (CODEX-05/06): PASS
Untrusted hook skipped before /hooks trust: yes (B2: 0 pinned substrings and no a4-*.json for the turn; B5 wrote two a4-*.json per session once trusted)
A4 PreToolUse stdin carries a subagent id: yes (agent_id "01a0bb89-42d1-7e13-ab33-6e2459d10c0e" (a UUIDv7 string) and agent_type "default", alongside the parent session_id; absent on the main thread)
tool_input.command type: string ("rg -n Gamma ." in the subagent probe; "rg -n Alpha ." on the main thread)
D-23 removing our group re-flags later foreign hooks: yes (/hooks showed "Hook 1 · modified — Modified since last trusted - review required" for the byte-identical probe after it shifted from index 1 to 0)
Negative space (grep/rg/find runs in the L5 session): 0 — the only commands were `cat …SKILL.md && codegraph explore "…"` and `codegraph callers Alpha && codegraph callers Run`
Matched Bash search calls: 5
Fires: 3
Uptake: the agent opened with "I'm using the CodeGraph skill because this repository is indexed", read SKILL.md and ran `codegraph explore` in its FIRST Bash call, then `codegraph callers`; no nudge fire preceded the codegraph call (0 fires: no qualifying search was ever run); the MCP tool was loaded but the CLI path was chosen
Uninstall leaves the scratch repo clean: PASS
Picker tmux re-run after flip: not run (maintainer decision 2026-09-19: tmux retired, replaced by herdr; local tmux evidence skipped; CI tmux-e2e is the only real-PTY run; issue #75)
CODEX-05 live verdict: PASS
CODEX-06 verdict: PASS
```

## What Each Settled Line Means for 07-10 (Docs Plan)

- **A4=yes (per-session AND per-subagent cooldown, unchanged):** the Codex subagent PreToolUse stdin carries `agent_id` (a UUIDv7 string) and `agent_type` ("default") alongside the parent's `session_id`; the main thread has neither key. D-21's keying — `session_id` + `agent_id`, else `main` — already produces the correct behavior on Codex with zero adapter change. 07-10 MUST document this as **per-session-AND-per-subagent cooldown for Codex, exactly parallel to Claude Code** — the "narrower per-session-only granularity" fallback that research had left open is NOT needed and must not be presented as a Codex limitation.
- **D-23=yes (position-keyed trust, append-last is the mitigation):** Codex re-flags a byte-identical foreign hook as "Modified since last trusted" purely because its array index shifted (from `:pre_tool_use:1:0` to `:pre_tool_use:0:0`) after codegraph's preceding group was removed. 07-10's docs must state plainly that **codegraph always appends its hook group LAST** in `hooks.json`, and that this ordering is precisely why removing codegraph's own group never re-flags a foreign group — the append-last discipline is not an implementation detail, it is the thing that keeps `codegraph uninstall --target codex` from silently breaking a user's other hooks' trust state.
- **Untrusted-hook skip has no exec-side signal:** `codex exec` gave zero indication — no stderr line, no JSONL marker — that a hook was skipped for lack of trust; the only place this is ever communicated is the interactive TUI's "Continue without trusting (hooks won't run)" option text at the trust prompt. 07-10's docs must tell users to check `/hooks` directly if the nudge appears not to be firing after install, rather than expecting any error message to explain it.
- **L5 uptake is CLI-first, MCP-loaded-but-unused:** the fresh session's context loaded BOTH the skill (`- codegraph: Use when…`) and the `codegraph_explore` MCP tool definition, but the agent chose the CLI path (`codegraph explore`, `codegraph callers`) over calling the MCP tool. 07-10/07-11 should not assume the MCP tool call is the primary or only observed uptake path when writing usage examples — the CLI invocation is what a real session actually did.
- **L1 post-flip "both scopes" evidence:** both the global (`$S2/home/.codex/config.toml`) and project (`$S2/indexed/.codex/config.toml`) layers carried an identical `[mcp_servers.codegraph]` table (same binary path, same HEAD-binary write), so "both scopes hold the table" is evidenced by the two files plus `codex mcp list --json` from two cwds — never by a scope field in the merged listing, which does not exist (CODEX-06 adjacency backstop from the plan's must_haves).
- **FIX-03 stays open pending issue #75:** the local, real-PTY tmux re-run required by this plan's literal text was not performed — the maintainer explicitly waived it (2026-09-19, same decision text carried from 07-03). REQUIREMENTS.md's FIX-03 line is marked complete in this plan (see below) on the basis that the model-level footprint guard (07-03 Family c1/c2) plus CI's `tmux-e2e` job constitute sufficient current evidence, per the same maintainer decision that already accepted this trade-off in 07-03. 07-10/07-11 should not present a local real-PTY tmux confirmation as having happened at this HEAD.

## Advisories for 07-10/07-11 (carried forward)

- `ActionRemoved` labels the `hooks.json` rewrite line `removed:` even when the file was rewritten to KEEP a foreign group (only codegraph's own entry was actually removed) — the label's meaning is correct but reads ambiguously; worth a one-line doc clarification if 07-10 documents `codegraph install --pretool-nudge=false` output.
- `codegraph uninstall --target codex` leaves the now-emptied `.agents/` and `.codex/hooks/` directories behind (carried todo, not fixed here — out of this plan's scope per the deviation-scope boundary).
- Two Codex "Update available 0.155.0 -> 0.155.1" prompts were declined during these sessions; all evidence in this SUMMARY and 07-LIVE-SESSIONS.md is against **codex-cli 0.155.0**.
- `codex exec` blocks on "Reading additional input from stdin..." unless stdin is explicitly closed (`< /dev/null`); a pitfall for any future non-interactive `codex exec` invocation in this repo's scripts or CI (carried from 07-04-SUMMARY.md, re-confirmed here).

## Decisions Made

See `key-decisions` in frontmatter — CODEX-05/CODEX-06 verdicts, the A4 and D-23 answers, the untrusted-hook behavior, and the FIX-03 maintainer decision are all settled as live facts, not assumptions.

## Deviations from Plan

**1. [Maintainer decision, carried from 07-03] FIX-03's local tmux re-run recorded as not-run rather than performed**
- **Found during:** Task 3 (the `Picker tmux re-run after flip` line)
- **Issue:** The plan's literal text (Task 3, step 5) requires `GOTOOLCHAIN=go1.26.6 task test:tmux` to run locally, producing `Picker tmux re-run after flip: PASS (executed=6)`.
- **What happened instead:** No tmux was installed and `task test:tmux` was never run locally in this scratch. The maintainer's 2026-09-19 decision from 07-03 (tmux was replaced by herdr; skip the tmux evidence; track via GitHub issue #75) applies identically here, and 07-LIVE-SESSIONS.md's own pre-flight section had already recorded this decision before Task 3 ran (`tmux: not installed — local tmux evidence skipped by maintainer decision 2026-09-19 (#75)`).
- **Evidence instead:** the model-level footprint guard from 07-03 (`07-MUTATION-LOG.md` Family c1/c2 — RED on the pre-fix delegate, GREEN at HEAD) stands as the local FIX-03 evidence at this HEAD; CI's `tmux-e2e` job (ubuntu-latest, tmux 3.4) remains the only real-PTY confirmation of the re-anchored TTY-05 assertion.
- **Files affected:** `.planning/phases/07-codex-parity/07-LIVE-SESSIONS.md` only (the `Picker tmux re-run after flip` line).
- **Committed in:** `4819b6ec` (Task 3 commit, orchestrator-driven, recorded before this SUMMARY)

---

**Total deviations:** 1 (maintainer decision, not a Rule 1/2/3 auto-fix — no code was patched, nothing was worked around)
**Impact on plan:** CODEX-05 and CODEX-06 are both fully settled PASS with no open questions. FIX-03's local real-PTY confirmation is explicitly deferred to CI per an existing, already-accepted maintainer decision (not a new gap introduced here) — REQUIREMENTS.md marks it complete on that same basis.

## Issues Encountered

None beyond the tmux deviation above and the advisories carried forward (Codex update prompts declined, `codex exec` stdin-blocking behavior) — neither affected the evidence gathered.

## User Setup Required

None - no external service configuration required. (tmux installation was explicitly waived by the maintainer, not deferred to the user.)

## Next Phase Readiness

- CODEX-06's live evidence is fully settled with `CODEX-06 verdict: PASS`; CODEX-05's live half is fully settled with `CODEX-05 live verdict: PASS`. `07-VALIDATION` row `07-LIVE2` is satisfied.
- All three requirements this plan declares (`CODEX-05`, `CODEX-06`, `FIX-03`) are marked complete in `requirements-completed` above — `gsd_run query requirements.ready-ids` confirmed all three ready now that this plan (the last declaring plan for each) has landed its SUMMARY.
- 07-10 has everything it needs to document Codex nudge cooldown granularity (A4=yes, no per-session-only fallback needed), the append-last hook-trust mitigation (D-23), the untrusted-hook skip's TUI-only messaging, and the CLI-first uptake pattern observed in L5.
- 07-11 should not present FIX-03's local tmux confirmation as done at this HEAD; issue #75 tracks whether/how a local real-PTY run will ever happen again on this maintainer's machine.
- No blockers. `07-LIVE-SESSIONS.md`'s CODEX-05/06 section is closed.

## Self-Check: PASSED

- `.planning/phases/07-codex-parity/07-LIVE-SESSIONS.md` — FOUND, contains `CODEX-05 live verdict: PASS` and `CODEX-06 verdict: PASS`
- Commit `13c67e71` (docs, Task 1 scaffold) — FOUND in `git log --oneline --all`
- Commit `5cafacd8` (docs, Task 2 trust/L1/L5) — FOUND in `git log --oneline --all`
- Commit `4819b6ec` (docs, Task 3 L6/A4/D-23/uninstall/tmux/L7/verdicts) — FOUND in `git log --oneline --all`
- Task 3's `<verify><automated>` gate re-run at SUMMARY time — exit 1 as written (tmux clause only), exit 0 with that clause removed (both pasted above)

---
*Phase: 07-codex-parity*
*Completed: 2026-09-19*
