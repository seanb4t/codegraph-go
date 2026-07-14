---
gsd_state_version: 1.0
milestone: v1.0
milestone_name: — Drop-in Parity & Human UX
current_phase: 1
current_phase_name: Behavioral Parity — explore & node
status: executing
stopped_at: Phase 1 context gathered
last_updated: "2026-07-14T21:42:57.508Z"
last_activity: 2026-07-14
last_activity_desc: ROADMAP.md created for v1.0 (8 phases, 45/45 requirements mapped)
progress:
  total_phases: 8
  completed_phases: 0
  total_plans: 0
  completed_plans: 0
  percent: 0
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-07-14)

**Core value:** An agent user can uninstall TS CodeGraph, install the Go binary, migrate their indexes, and everything works the same or better — faster, from a single verifiably-built binary.
**Current focus:** v1.0 (Drop-in Parity & Human UX) — Phase 1 (Behavioral Parity — explore & node)

## Current Position

Phase: 1 of 8 (Behavioral Parity — explore & node)
Plan: — (roadmap created; ready to plan)
Status: Ready to execute
Last activity: 2026-07-14 — ROADMAP.md created for v1.0 (8 phases, 45/45 requirements mapped)

Progress: [░░░░░░░░░░] 0%

## Performance Metrics

**Velocity (v1.0):**

- Total plans completed: 0 (v0.1 shipped 58+ plans across 8 phases — see milestones/v0.1-*)
- Average duration: — min
- Total execution time: 0.0 hours

**By Phase (v1.0):**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| 1 | 0/TBD | - | - |
| 2 | 0/TBD | - | - |
| 3 | 0/TBD | - | - |
| 4 | 0/TBD | - | - |
| 5 | 0/TBD | - | - |
| 6 | 0/TBD | - | - |
| 7 | 0/TBD | - | - |
| 8 | 0/TBD | - | - |

**Recent Trend:** — (no v1.0 plans executed yet)

*Updated after each plan completion*

## Accumulated Context

### Decisions

Full decision log in PROJECT.md Key Decisions. Decisions shaping v1.0:

- [Milestone v1.0]: Scope derived from a live TS 1.3.1 vs codegraph-go bake-off — the command surface is already a superset, so gaps are behavioral (explore/node), watcher-on-MCP default, git/worktree awareness, output hygiene, and human TUI — not missing commands.
- [Milestone v1.0]: Include a bubbletea/Charm human TUI, but the agent/MCP output path stays plain/parseable — enforced by an import-graph archtest (ANSI-isolation guarantee).
- [Milestone v1.0]: Worktree awareness scoped at TS parity (detect+warn+notice); auto-init / git-common-dir sharing deferred (WORK-FUT-01).
- [Milestone v1.0]: No daemon auto-spawn — `serve --mcp` watches in-process (WATCH-01); the daemon is only the explicit shared-writer + picker case.
- [Roadmap v1.0]: Phase numbering reset to 1 (v0.1's 8 phases archived to milestones/v0.1-phases/).

### Pending Todos

[From .planning/todos/pending/ — ideas captured during sessions]

None active for v1.0. Backlog item 999.1 (local build/Taskfile.yml + CONTRIBUTING.md) tracked in ROADMAP.md Backlog.

### Blockers/Concerns

[Issues that affect future work — sourced from research/SUMMARY.md]

- **[Phase 1]** RWR relevance (EXPL-02) is the single highest-risk item — golden-corpus contract at stake. Behavioral fixtures (TEST-01) MUST land before/with the algorithm (template-parity ≠ behavior-parity).
- **[Phase 3]** Watcher default-flip hangs MCP startup on WSL2 without the watch-policy port — bundle them; needs a real WSL2 reproduction to validate.
- **[Phase 2/8]** `files --filter` semantic collision (ours=language, TS=directory) — resolved by SURF-02 (keep language + add directory flag); the #1 silent-failure risk if mishandled.
- **[Phase 6/8]** Charm v2 uses the `charm.land/...` vanity import; audit the full transitive closure for CGo (none expected) + govulncheck + SBOM before the v1.0.0 release.

## Deferred Items

Carried forward from v0.1 close — now scoped into v1.0 Phase 8:

| Category | Item | Status | Deferred At |
|----------|------|--------|-------------|
| Release | DIST-02 — real signed `v*` tag | Scoped into REL-02 (Phase 8) | v0.1 close |
| Perf | PERF-01 — published head-to-head numbers | Scoped into REL-03 (Phase 8) | v0.1 close |

## Session Continuity

Last session: 2026-07-14T20:43:26.635Z
Stopped at: Phase 1 context gathered
Resume file: .planning/phases/01-behavioral-parity-explore-node/01-CONTEXT.md

## Operator Next Steps

- Review the roadmap draft (.planning/ROADMAP.md).
- Plan the first phase with `/gsd-plan-phase 1`.
