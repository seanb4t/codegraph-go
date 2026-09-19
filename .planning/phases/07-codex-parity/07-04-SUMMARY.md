---
phase: 07-codex-parity
plan: 04
subsystem: codex
tags: [codex, live-evidence, mcp, hooks, trust, skills, codex-cli]

# Dependency graph
requires:
  - phase: 07-codex-parity
    provides: "wave-4 sequencing after 07-03 (unrelated TUI picker fix); the fixed HEAD binary's `install --target codex` (07-01/07-02) used for the global-install evidence"
provides:
  - "CODEX-01 live evidence: pass bar L1-L7 all PASS with dated Codex CLI documentation citations, gathered in an isolated scratch HOME (D-02) before any codex.go change (D-01)"
  - "A1/A2/D-15/D-16/D-17 and the trust-override question answered from real Codex sessions (interactive TUI + `codex exec`), each with a positive-controlled absence search"
  - "Real `~/.codex` and `~/.agents` files verified byte-unchanged (sha256 before/after, L7)"
affects: [07-05, 07-07, 07-09, 07-10, 07-11]

# Actuals (#2632)
actuals:
  tokens: 8103
  tasks: 3
  commits: 3
plan_head_before: 3096959f3fb5962701123e058e2e1323106ff248

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Live-evidence plans run in a genuinely isolated `HOME`/`CODEX_HOME` scratch (never the maintainer's real Codex config), with sha256 checksums of the real files taken before and after as the tamper proof (L7)"
    - "Every absence claim is backed by a positive-controlled search — the same search that proves absence must first be shown to find the thing when it is present, in a trusted session"

key-files:
  created: []
  modified:
    - .planning/phases/07-codex-parity/07-LIVE-SESSIONS.md

key-decisions:
  - "CODEX-01 verdict: PASS — all six pass-bar L-lines (L1 project, L1 global, L2, L3, L4, L7) PASS in an isolated Codex scratch HOME; real ~/.codex and ~/.agents files unchanged (sha256 before/after, L7)."
  - "D-15: both `.codex/skills` (project) and `$CODEX_HOME/skills` are read by Codex (prompt-input roots r0/r1) — 07-05 MAY list them as read-only `SkillDirs[1:]` entries; D-14 still forbids writing a second copy there (same-name skills are not merged)."
  - "A2/D-16: Codex trust-gates project MCP servers (config.toml) and hooks.json, but NOT the repo `AGENTS.md` block or project `.agents/skills` — both were visible and listed in a real untrusted session. 07-05's D-10 trust Note must say the MCP server (and hooks) need trust; the skill/AGENTS.md content works untrusted."
  - "Trust override (`-c projects.\"<path>\".trust_level=\"trusted\"`) does NOT grant trust to the untrusted project — MCP servers stayed `[]` under the override; only the real TUI trust prompt loads the project layer. 07-05's Note must not suggest the override as a substitute for trust."
  - "A1: both the local D-20 command form (`\"$(git rev-parse --show-toplevel)/.codex/hooks/<script>\"`) and the single-quoted absolute global form (verified with a space in the path) are shell-expanded and executed by Codex hooks."
  - "Hook trust is keyed per `<hooks.json path>:pre_tool_use:<group>:<handler>` with a `trusted_hash` — adding a new hook in the global `hooks.json` did not re-flag the already-trusted project hook. 07-07 must append new hook entries (D-23) rather than reordering/rewriting the file, or it will re-trigger trust review for every existing hook."
  - "`features.hooks=false` silences hooks end-to-end (marker log stayed 1->1); hooks-on fires normally (1->2) — confirms the feature flag is a genuine kill switch, not cosmetic."
  - "The global install splice (internal/agents/codex.go, fixed in 07-01/07-02) preserved Codex's own `[projects...]`/`[hooks.state...]` tables byte-for-byte (5 lines before, 5 after) while appending the `[mcp_servers.codegraph]` table."

patterns-established: []

requirements-completed: []  # CODEX-01 is shared with 07-01, 07-02, 07-05, 07-09, 07-10, 07-11 (gsd_run query requirements.ready-ids reports 0/1 ready) — this plan's live evidence settles the requirement's factual content, but the ID stays unmarked until every declaring plan lands. Left for the last plan to close.

coverage:
  - id: D1
    description: "CODEX-01 verdict PASS: pass bar L1 (project), L1 (global), L2, L3, L4, L7 all PASS, gathered in a genuinely isolated Codex scratch HOME before any codex.go change, with dated Codex CLI documentation citations"
    requirement: "CODEX-01"
    verification:
      - kind: other
        ref: "07-04-PLAN.md Task 3 <verify><automated> gate — re-run at SUMMARY time: no PENDING, exactly 6 L-lines in allowed form, verdict consistent with all 6 PASS, real-HOME checksums (L7) PASS, no internal/agents/codex.go commit since the 07-01 plans commit, evidence commit subject present"
        status: pass
    human_judgment: true
    rationale: "The automated gate verifies structural completeness and internal consistency of the recorded lines, but the underlying claims (what Codex CLI actually did in the interactive TUI and codex exec sessions) were observed live by the orchestrator via Herdr and cannot be replayed by an automated test here. A human consuming this evidence for 07-05/07-07 should spot-check at least one verbatim excerpt (e.g. the C2 mcp list --json outputs) against the raw transcripts under $S/transcripts/ before treating it as final."
  - id: D2
    description: "A1 (local + global hook command forms), A2, D-15 (both skill roots), D-16, D-17 and the trust-override question answered from real session JSONL and marker-log evidence (not model paraphrase), each absence backed by a positive-controlled search in the trusted session"
    requirement: "CODEX-01"
    verification:
      - kind: other
        ref: "07-LIVE-SESSIONS.md CODEX-01 verdicts section (Protocol A/B/C evidence blocks); same automated gate as D1 for line-form and PENDING checks"
        status: pass
    human_judgment: true
    rationale: "Same reasoning as D1 — the gate checks line format and internal consistency (e.g. any FAIL or A1 'no' requires a Maintainer decision line), not the truth of what Codex CLI did during the live session. Evidentiary correctness rests on the orchestrator's direct observation of Herdr pane output and marker-log files."

duration: ~15min (Task 1 scaffold to Task 3 evidence commit; orchestrator-driven Herdr/Codex sessions for Tasks 2-3 ran outside this timing)
completed: 2026-09-19
status: complete
---

# Phase 7 Plan 4: CODEX-01 Live Evidence Summary

**Settled CODEX-01 live, before any `codex.go` change: Codex CLI loads a project-local `.codex/config.toml` only when trusted, reads skills from both `.codex/skills` and `$CODEX_HOME/skills`, trust-gates MCP servers and hooks but NOT `AGENTS.md` or project skills, and shell-expands both the local and global hook command forms — verdict PASS on all six locked pass-bar lines, with the maintainer's real `~/.codex`/`~/.agents` files verified byte-unchanged.**

## Performance

- **Duration:** ~15 min (commit-to-commit span for this plan's three task commits)
- **Started:** 2026-09-19T12:59:12-04:00 (Task 1 scaffold commit)
- **Completed:** 2026-09-19T13:13:39-04:00 (Task 3 evidence commit)
- **Tasks:** 3 (Task 1 executor-scaffolded; Tasks 2-3 orchestrator-driven via Herdr against a real Codex CLI, then executor-gated for completeness)
- **Files modified:** 1

## Accomplishments

- Built a genuinely isolated Codex scratch (`HOME`/`CODEX_HOME` under `/private/tmp/07-live`, `auth.json` symlinked — never copied — from the real `~/.codex`), with `trusted`/`untrusted`/`bare` repos, project-layer `.codex/config.toml` hand-planted with the exact `codexTableBody` bytes (since `codex.go` cannot write this yet), and D-15/A1 probe skills and hooks.
- Ran Protocol A (model-free): `bare` and `untrusted` `codex mcp list --json` / `debug prompt-input` confirmed no project MCP surface before trust, and the `-c projects...trust_level=trusted` override does NOT grant trust (`[]` stayed empty).
- Ran Protocol B (real interactive TUI + minimal model turns): trusted `$S/trusted` through the real TUI prompt, confirmed L1 (project) and L3 (skill + AGENTS.md block listed), and captured the A1 local hook probe's raw PreToolUse stdin field list and `tool_input.command` type. Confirmed via real untrusted/trusted session JSONL that `AGENTS.md` and project `.agents/skills` are NOT trust-gated (A2=no, D-16=no) even though the MCP server is.
- Ran Protocol C (global install + close-out): installed the HEAD binary's global Codex config into the scratch HOME, confirmed L1 (global) shows the global binary while the project layer still wins in `trusted`, confirmed the global quoted hook form runs even with a space in its path, re-ran the four L7 checksums (unchanged), and recorded `CODEX-01 verdict: PASS`.
- Recorded all 18 CODEX-01 verdict lines in `07-LIVE-SESSIONS.md` with verbatim excerpts and dated Codex CLI documentation citations; re-ran Task 3's automated completeness gate (exit 0) before writing this SUMMARY.

## Task Commits

Each task was committed atomically:

1. **Task 1: Scaffold, pre-flight, citations, pass bar, 18-PENDING skeleton** - `429abc02` (docs)
2. **Task 2 (orchestrator): Protocols A and B — 14 of 18 lines** - `d13d7f26` (docs)
3. **Task 3 (orchestrator + executor gate): Protocol C — remaining 4 lines, verdict, completeness gate** - `97c82219` (docs)

**Plan metadata:** committed separately after this SUMMARY (see below).

## Files Created/Modified

- `.planning/phases/07-codex-parity/07-LIVE-SESSIONS.md` — CODEX-01 section: pass bar, pre-flight, dated Codex CLI reference citations, scaffold, Protocol A/B/C evidence blocks, and the 18-line verdict table (all settled, `CODEX-01 verdict: PASS`)

## Task 3 Completeness Gate — Re-run at SUMMARY Time

Re-ran the plan's exact `<verify><automated>` command from `07-04-PLAN.md` Task 3 before writing this SUMMARY:

```
$ cd /Volumes/Code/github.com/seanb4t/codegraph-go && D=.planning/phases/07-codex-parity/07-LIVE-SESSIONS.md && \
  ! rg -q 'PENDING' "$D" && \
  test "$(rg -c '^L[1-7] [^:]+: (PASS|FAIL \(.+\))$' "$D")" = "6" && \
  rg -q '^A1 global quoted command form runs: (yes|no|not observed)( \(.+\))?$' "$D" && \
  rg -q '^L7 real HOME unchanged \(CODEX-01\): PASS$' "$D" && \
  rg -q '^CODEX-01 verdict: (PASS|FAIL)$' "$D" && \
  if rg -q '^CODEX-01 verdict: PASS$' "$D"; then test "$(rg -c '^L[1-7] [^:]+: PASS$' "$D")" = "6"; else rg -q '^Maintainer decision: ' "$D"; fi && \
  L=$(git log --diff-filter=A --format=%H -- .planning/phases/07-codex-parity/07-01-PLAN.md) && B=$(printf "%s\n" "$L" | tail -1) && \
  test -n "$B" && test -z "$(git log --format=%H "$B"..HEAD -- internal/agents/codex.go)" && \
  S=$(git log --format=%s -5) && printf '%s\n' "$S" | rg -q '^docs\(07-04\): record CODEX-01 live evidence'

EXIT_CODE: 0
```

Result: **PASS.** No `PENDING` remains, exactly 6 L-lines in allowed form, the A1-global and L7 lines are present and PASS, the verdict is `PASS` with all 6 L-lines `PASS`, no `internal/agents/codex.go` commit exists since the 07-01 plans commit, and the evidence commit subject matches.

## CODEX-01 Verdicts (verbatim from 07-LIVE-SESSIONS.md)

```
L1 project config loads when trusted: PASS
L1 global entry shown (global install): PASS
L2 untrusted project layer not loaded: PASS
L3 prompt-input lists skill and AGENTS.md block: PASS
L4 uninstalled repo shows no codegraph surface: PASS
L7 real HOME unchanged (CODEX-01): PASS
Untrusted warning: none observed (mcp list / debug prompt-input / exec stderr carry no warning; the only trust messaging is the TUI prompt quoted in B4)
Trust override (-c projects trust_level) grants trust: no (A3: `[]` under the override; the same key written by the real TUI trust loads the layer, B6)
A1 local command form shell-expanded: yes (B5: a1-local 2026-09-19T17:02:54Z /private/tmp/07-live/trusted)
A1 global quoted command form runs: yes (C3: a1-global 2026-09-19T17:11:22Z /private/tmp/07-live/trusted, single-quoted absolute path containing a space)
A2 AGENTS.md trust-gated in a real session: no (B8: "## CodeGraph" injected in the untrusted read-only exec session, 2 vs 2)
D-16 project .agents/skills trust-gated: no (B8: "- codegraph: Use when" listed in the untrusted session, 2 vs 2)
D-15 .codex/skills read: yes (B7: r0 = trusted/.codex/skills lists cgprobe-dotcodex; also listed untrusted, B8)
D-15 CODEX_HOME/skills read: yes (B7: r1 = home/.codex/skills lists cgprobe-codexhome)
D-17 skill description as listed: "Use when asked where X is defined, how Y works, what calls X, or what changing X breaks in a .codegraph/ repo." (untruncated, byte-identical to SKILL.md frontmatter; B7)
Hooks.json runs behind features.hooks: yes (B5: marker 1->2 with hooks on, 1->1 with -c features.hooks=false; `hooks stable true` in features list)
PreToolUse stdin fields (main thread): cwd, hook_event_name, model, permission_mode, session_id, tool_input, tool_name, tool_use_id, transcript_path, turn_id; tool_name "Bash"; tool_input.command is a string; no agent_id/agent_type on the main thread (B5)
CODEX-01 verdict: PASS
```

## What Each Settled Line Means for 07-05 (Scope Flip)

- **D-15 (`.codex/skills` AND `$CODEX_HOME/skills` are both read, roots r0/r1):** per D-15, 07-05 MAY list these as read-only `SkillDirs[1:]` entries in `internal/agents/codex.go` — the planner/executor of 07-05 decides the exact wiring. D-14 still forbids WRITING a second copy of the skill there; same-name skills across roots are not merged by Codex, so codegraph must keep writing to exactly one location.
- **A2=no, D-16=no (AGENTS.md and project `.agents/skills` are NOT trust-gated):** Codex trust-gates project config (MCP servers) and hooks, but not the repo `AGENTS.md` block or project skills — both render for an untrusted project. 07-05's D-10 trust Note MUST say the MCP server (and hooks) need trust; it must NOT imply the skill or AGENTS.md content is gated too.
- **Trust override does NOT grant trust:** `-c projects."<path>".trust_level="trusted"` left the untrusted project's MCP servers empty (`[]`); only the real TUI trust prompt loaded the project layer. The D-10 Note must not suggest the override as a substitute for actual trust.
- **Global install splice preserved Codex's own tables:** the HEAD binary's `install --target codex --location global` appended only the `[mcp_servers.codegraph]` table; Codex's own `[projects...]`/`[hooks.state...]` tables were unchanged (5 lines before, 5 after) — confirms the splice fix from 07-01/07-02 is safe for repeat global installs.

## What Each Settled Line Means for 07-07 (Nudge)

- **A1 local form IS shell-expanded and run:** the exact D-20 form `"$(git rev-parse --show-toplevel)/.codex/hooks/<script>"` fired on the probe turn (marker log line captured). The single-quoted absolute global form also runs, verified even with a space in the script's directory path (AR-06-08 case) — 07-07's hook command form can rely on both.
- **PreToolUse stdin (main thread) fields:** `cwd, hook_event_name, model, permission_mode, session_id, tool_input, tool_name, tool_use_id, transcript_path, turn_id`; `tool_name` is `"Bash"`; `tool_input.command` is a **string**, not an array; no `agent_id`/`agent_type` key is present on the main thread (the subagent case with those keys remains open for 07-09 to settle).
- **Hook trust is keyed per file+group+handler with a `trusted_hash`:** adding the global hook in a *different* hooks.json file did not re-flag the already-trusted project hook (`1 hook is new or changed` — only the new one). D-23's append-last discipline is required: rewriting or reordering an existing hooks.json risks re-triggering trust review for every hook in that file, not just the new one.
- **`features.hooks=false` is a genuine kill switch:** the marker log stayed at its prior count (1->1) with hooks disabled via `-c features.hooks=false`, and fired normally (1->2) with hooks on — 07-07 can rely on this flag to fully suppress hook execution, not just hide it from `features list`.

## Advisory Notes

- `codex-cli` offered an update to 0.155.1 during the live TUI session; the update was declined ("Skip"), so all evidence in this SUMMARY is against **codex-cli 0.155.0**, the version recorded at research time.
- `codex exec` blocks on "Reading additional input from stdin..." unless stdin is explicitly closed (`< /dev/null`) or otherwise redirected — a pitfall for any future non-interactive `codex exec` invocation in this repo's scripts or CI.

## Decisions Made

See `key-decisions` in frontmatter — CODEX-01 verdict PASS, D-15/D-16/A2/trust-override answers, the A1 command forms, hook-trust keying, and the global-install splice safety are all settled as live facts, not assumptions.

## Deviations from Plan

None - plan executed exactly as written. Task 1 was scaffolded by the executor per plan; Tasks 2 and 3 (checkpoint:human-action, gate=blocking-human) were driven by the orchestrator exactly per their `<instructions>`, with all 18 verdict lines recorded in their fixed forms and no FAIL or "A1 ... no" requiring a `Maintainer decision:` escalation.

## Issues Encountered

None beyond the two advisory notes above (codex-cli update offer declined; `codex exec` stdin-blocking behavior) — neither affected the evidence gathered.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- CODEX-01's live evidence is fully settled with `CODEX-01 verdict: PASS`. `internal/agents/codex.go` remains untouched by this plan (verified by the gate), so 07-05 is the first plan permitted to change it.
- `requirements-completed` is deliberately left empty in this SUMMARY's frontmatter: CODEX-01 is declared by six plans in this phase (07-01, 07-02, 07-04, 07-05, 07-09, 07-10, 07-11) and `gsd_run query requirements.ready-ids` reports 0/1 ready — the requirement will be marked complete automatically when the last declaring plan lands its SUMMARY.
- 07-05 has everything it needs to proceed on the scope flip: D-15 SkillDirs guidance, the D-10 trust Note wording constraint (MCP server and hooks need trust; AGENTS.md/skills do not), and confirmation the trust override must not be presented as a trust substitute.
- 07-07 has everything it needs for the nudge: both hook command forms are confirmed shell-expanded, the full PreToolUse stdin field list and `tool_input.command` type are recorded, and the per-file hook-trust keying behavior (D-23 append-last) is confirmed live.
- No blockers. `07-LIVE-SESSIONS.md`'s CODEX-01 section is closed; only 07-VALIDATION row `07-LIVE1` and downstream plans (07-05, 07-07, 07-09) remain to consume it.

## Self-Check: PASSED

- `.planning/phases/07-codex-parity/07-LIVE-SESSIONS.md` — FOUND, contains `CODEX-01 verdict: PASS`
- Commit `429abc02` (docs, Task 1 scaffold) — FOUND in `git log --oneline --all`
- Commit `d13d7f26` (docs, Task 2 Protocols A/B) — FOUND in `git log --oneline --all`
- Commit `97c82219` (docs, Task 3 Protocol C + verdict) — FOUND in `git log --oneline --all`
- Task 3's `<verify><automated>` gate re-run at SUMMARY time — exit 0 (pasted above)
- No `internal/agents/codex.go` commit since the 07-01 plans commit — confirmed by the gate

---
*Phase: 07-codex-parity*
*Completed: 2026-09-19*
