---
gsd_state_version: 1.0
milestone: v1.0
milestone_name: — Drop-in Parity & Human UX
current_phase: 1
current_phase_name: Behavioral Parity — explore & node
status: executing
stopped_at: Completed 01-04-PLAN.md
last_updated: "2026-07-15T13:18:44.389Z"
last_activity: 2026-07-14
last_activity_desc: Phase 1 execution started
progress:
  total_phases: 8
  completed_phases: 0
  total_plans: 17
  completed_plans: 5
  percent: 0
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-07-14)

**Core value:** An agent user can uninstall TS CodeGraph, install the Go binary, migrate their indexes, and everything works the same or better — faster, from a single verifiably-built binary.
**Current focus:** Phase 1 — Behavioral Parity — explore & node

## Current Position

Phase: 1 (Behavioral Parity — explore & node) — EXECUTING
Plan: 6 of 17
Status: Ready to execute
Last activity: 2026-07-14 — Phase 1 execution started

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
| Phase 01-behavioral-parity-explore-node P01 | 55min | 3 tasks | 30 files |
| Phase 01 P02 | 5min | 1 tasks | 1 files |
| Phase 01-behavioral-parity-explore-node P03 | 20min | 2 tasks | 2 files |
| Phase 01-behavioral-parity-explore-node P04 | 25min | 3 tasks | 4 files |
| Phase 01-behavioral-parity-explore-node P05 | 19min | 2 tasks | 9 files |

## Accumulated Context

### Decisions

Full decision log in PROJECT.md Key Decisions. Decisions shaping v1.0:

- [Milestone v1.0]: Scope derived from a live TS 1.3.1 vs codegraph-go bake-off — the command surface is already a superset, so gaps are behavioral (explore/node), watcher-on-MCP default, git/worktree awareness, output hygiene, and human TUI — not missing commands.
- [Milestone v1.0]: Include a bubbletea/Charm human TUI, but the agent/MCP output path stays plain/parseable — enforced by an import-graph archtest (ANSI-isolation guarantee).
- [Milestone v1.0]: Worktree awareness scoped at TS parity (detect+warn+notice); auto-init / git-common-dir sharing deferred (WORK-FUT-01).
- [Milestone v1.0]: No daemon auto-spawn — `serve --mcp` watches in-process (WATCH-01); the daemon is only the explicit shared-writer + picker case.
- [Roadmap v1.0]: Phase numbering reset to 1 (v0.1's 8 phases archived to milestones/v0.1-phases/).
- [Phase 1]: MCP node capture passes includeCode:true explicitly to match the TS CLI's unconditional full-body multi-def rendering (TS MCP tool defaults it false), avoiding an incidental CLI/MCP asymmetry unrelated to NODE-04/EXPL-05
- [Phase 1]: synthetic-parity corpus gets behavioral-only fixtures (no baseline status/query/callers/callees/impact/explore/node) -- it exists solely to drive the D-03 multi-def/multi-word/gate/RWR cases
- [Phase 1]: No SchemaVersion bump, no proto regeneration — Edge.kind is a free-form proto3 string, so the 6 new D-09 edge kinds are additive DATA values, not a schema change
- [Phase 1]: extends/overrides get EdgeKind* constants (Pass-2 synthesis only); references/instantiates/returns/type_of get RefKind* constants (Pass-1 captured)
- [Phase 01-behavioral-parity-explore-node]: Read the live TS 1.3.1 dist source directly (D-01) to correct two RESEARCH.md excerpt gaps in the H1/H2 tokenizer port: STOP_WORDS/commonWords conflation in the plan's illustrative example, and H2's omitted compound-preservation regex
- [Phase 01-behavioral-parity-explore-node]: Deferred getStemVariants() FTS-prefix stem expansion from extractSearchTerms entirely (no stub parameter) per the plan's explicit D-02 divergence allowance
- [Phase 1]: narrowNodeMatches (NODE-03) is fully implemented and tested but not wired into the public Node() signature — no line hint exists in the CLI/MCP surface yet; ready for a future plan to wire it
- [Phase 1]: Fixed a real bug during ad-hoc golden verification: multi-def source blocks rendered the whole file instead of the definition's own line range (renderNumberedSourceRange added)
- [Phase 1]: extends is a 3-way Pass-2 split of the pre-existing embeds-promotion branch (implements/extends/embeds by target+source Kind) — shared code that rippled into every language extractor's class-extends-class regression test, not just Go's
- [Phase 1]: references' Go Pass-1 capture is scoped to a bounded allow-list of unambiguous read positions rather than exhaustive AST coverage, to avoid false same-package-name-collision resolutions
- [Phase 1]: type_of applies only to package-level var declarations for Go (struct-field/local-var type_of is an explicit per-language divergence — this extractor emits no field or local-var nodes to anchor those refs on)

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

Last session: 2026-07-15T13:18:44.383Z
Stopped at: Completed 01-04-PLAN.md
Resume file: None

## Operator Next Steps

- Review the roadmap draft (.planning/ROADMAP.md).
- Plan the first phase with `/gsd-plan-phase 1`.
