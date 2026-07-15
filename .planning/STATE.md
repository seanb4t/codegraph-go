---
gsd_state_version: 1.0
milestone: v1.0
milestone_name: — Drop-in Parity & Human UX
current_phase: 02
current_phase_name: status Content & Git/Worktree Awareness
status: executing
stopped_at: Completed 02-02-PLAN.md
last_updated: "2026-07-15T23:53:26.886Z"
last_activity: 2026-07-15
last_activity_desc: Phase 02 execution started
progress:
  total_phases: 8
  completed_phases: 1
  total_plans: 24
  completed_plans: 22
  percent: 13
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-07-14)

**Core value:** An agent user can uninstall TS CodeGraph, install the Go binary, migrate their indexes, and everything works the same or better — faster, from a single verifiably-built binary.
**Current focus:** Phase 02 — status Content & Git/Worktree Awareness

## Current Position

Phase: 02 (status Content & Git/Worktree Awareness) — EXECUTING
Plan: 5 of 7
Status: Ready to execute
Last activity: 2026-07-15 — Phase 02 execution started

Progress: [░░░░░░░░░░] 0%

## Performance Metrics

**Velocity (v1.0):**

- Total plans completed: 18 (v0.1 shipped 58+ plans across 8 phases — see milestones/v0.1-*)
- Average duration: — min
- Total execution time: 0.0 hours

**By Phase (v1.0):**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| 1 | 18 | - | - |
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
| Phase 01-behavioral-parity-explore-node P06 | 15min | 2 tasks | 2 files |
| Phase 01-behavioral-parity-explore-node P07 | 15min | 2 tasks | 2 files |
| Phase 01-behavioral-parity-explore-node P11 | 5min | 2 tasks | 2 files |
| Phase 01-behavioral-parity-explore-node P12 | 35min | 2 tasks | 2 files |
| Phase 01-behavioral-parity-explore-node P08 | 20min | 2 tasks | 4 files |
| Phase 01-behavioral-parity-explore-node P09 | 24min | 2 tasks | 4 files |
| Phase 01-behavioral-parity-explore-node P10 | 20min | 2 tasks | 2 files |
| Phase 01-behavioral-parity-explore-node P13 | 25min | 2 tasks | 2 files |
| Phase 01-behavioral-parity-explore-node P14 | 12min | 2 tasks | 2 files |
| Phase 01-behavioral-parity-explore-node PP15 | 12min | 2 tasks | 0 files |
| Phase 01-behavioral-parity-explore-node P16 | 55min | 3 tasks | 5 files |
| Phase 01-behavioral-parity-explore-node P17 | 25min | 3 tasks | 13 files |
| Phase 02 P01 | 22min | 3 tasks | 8 files |
| Phase 02 P02 | 28min | 3 tasks | 4 files |
| Phase 02 P03 | 48min | 3 tasks | 2 files |
| Phase 02 P04 | 35min | 3 tasks | 5 files |

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
- [Phase 01-behavioral-parity-explore-node]: Task 1/2 combined into one feat commit (shared file, no independent dependency); Task 2's tests committed separately to lock in D-04's determinism contract
- [Phase 01-behavioral-parity-explore-node]: T-01-10 DoS mitigation for rwr.go is documentation-only (precondition doc comment) — maxNodes=200/GLUE_NODE_CAP=60 enforcement deferred to upstream subgraph-gathering plans 11/16 per this plan's threat_model disposition
- [Phase 1]: 01-07 gather channels (H3-H6) shipped without matchNodes wiring by design -- plan 10 wires this into Explore() and adds H7-H21 rerankers
- [Phase 01-behavioral-parity-explore-node]: H10-H12 subgraph-construction primitives (expandTypeHierarchy/expandBFS/expandGlueNodes) close the T-01-18 DoS mitigation gap plan 06 deliberately deferred; every cited constant (ceil(maxNodes/4), maxNodes=200/traversalDepth=3/minScore=0.2/searchLimit=8, GLUE_NODE_CAP=60) ported verbatim from RESEARCH; traversal mechanics (RankEdges-walk, sorted-Id tie-break) documented as this plan's own design since no verbatim TS source for H10-H12 survives in the frozen capture
- [Phase 1]: H13 named seeding: body-substance measure (line span) and type-token corroboration mechanism (owning-type-name match, not self-name match) are this plan's own documented D-02 design since no verbatim TS source for these specifics survives the frozen RESEARCH capture
- [Phase 1]: Java/C# field type_of is anchored on the ENCLOSING TYPE id (not a field node, since neither extractor emits one) — a documented D-02 divergence distinct from Go's package-level-var-anchored type_of
- [Phase 1]: Local-variable type_of added beyond the plan's literal 'field' behavior spec for Java/C#, anchored at the enclosing method — RESEARCH §B describes the missing kind as 'field/local declared type', and the existing method-body walk made this near-zero-cost
- [Phase 1]: Python's instantiates capture is folded directly into recordCall (not a parallel object-creation walk) since Foo() is syntactically identical to any function call in Python; a PascalCase gate trims candidate volume, resolve.go's unchanged Kind-check disambiguates the real edge
- [Phase 1]: All five priority-4 languages (Go/Java/C#/Python/TS-JS) now emit the full 9-member RANK_EDGES set -- D-09's extraction scope is complete; F4 (re-index) and F5 (regenerate golden corpus) are unblocked for a downstream plan
- [Phase 1]: H8's per-file edge count implemented as candidate-count proxy (pure function, no reader access) — documented substitution
- [Phase 1]: H9 stem grouping uses a new lightweight suffix-stripping stemTerm(), not TS's deferred getStemVariants() or a full Porter stemmer
- [Phase 1]: isDistinctiveIdentifier is a documented heuristic (>=6 runes + underscore/case-transition) substituting for TS's uncited algorithm
- [Phase 1]: applyPostMergeRerankers computes the H9 exemption set BEFORE H7 runs — deliberate inversion of the naive pipeline order so the exemption actually gates the dampening
- [Phase 01-behavioral-parity-explore-node]: H17's entry/named-file gate clause (3) derives from plan 13's fileScores tier (score>=fileScoreEntry) rather than a separate caller-supplied entryFiles set -- plan 13's own readiness note names only fileScores/fileGraphScore/rescued as this plan's inputs
- [Phase 01-behavioral-parity-explore-node]: H18's 5-tier sort uses ascending file path as the final deterministic tie-break -- files have no stable Id field (unlike nodes), so path is the documented D-04 substitute
- [Phase 1]: D-09 F4 re-index complete: this repo + weft-go + colbymchenry-codegraph + synthetic-parity force-re-indexed (index --force, never sync); colbymchenry-codegraph independently confirms all 6 new edge kinds fire correctly; this repo's own zero overrides/type_of count is a documented corpus-content fact, not a missing emit site
- [Phase 01-behavioral-parity-explore-node]: Wired the full H1-H21 explore pipeline; 'matched' symbols shown per file are gather+seed candidates (RWR-ordered), not the whole bounded subgraph, with a single-node structural fallback for files selected purely via connectivity
- [Phase 01-behavioral-parity-explore-node]: computeFileTermHits is a new wiring-level primitive this plan added — plans 13/14 both documented fileTermHits as unwired/caller-supplied
- [Phase 01-behavioral-parity-explore-node]: H21's getExploreOutputBudget is a documented D-02 monotonic step function substitute (TS dist unreadable); only overrides the maxFiles default, never an explicit --max-files
- [Phase 01-behavioral-parity-explore-node]: F5 Go-side fixture regeneration lands under a distinct go-*.json naming convention (via a new gocapture tool), never overwriting the TS 1.3.1 frozen oracle fixtures from plan 01
- [Phase 01-behavioral-parity-explore-node]: D-02's oracle applies at two tiers: full ordering+membership+warning+header+counts on synthetic-parity (purpose-built, tractable); shape-only plus four newly-documented allowed divergences (AD-01..04: TS stemming precision, TS/JS object-literal-method extraction gap, a weft-go def-count delta, file-selection/bullet-scope breadth) on the real-world weft-go/colbymchenry-codegraph corpora
- [Phase 02-status-content-git-worktree-awareness]: Ported TS sync/worktree.js's 4-gate cascade verbatim (gate order + gate-4 polarity: differing common dirs SUPPRESS the warning, not trigger it)
- [Phase 02-status-content-git-worktree-awareness]: CachingDetector lives in internal/gitmeta, not on query.Engine, since MCP rebuilds Engine per call and an Engine-scoped cache would give zero cross-call benefit
- [Phase 02-status-content-git-worktree-awareness]: T-02-04 accepted: Mismatch.WorktreeRoot/IndexRoot intentionally carry absolute host paths in the warning/notice text, a scoped exception to the codebase's usual no-host-paths-in-MCP-output stance
- [Phase 02-status-content-git-worktree-awareness]: dbSizeBytes' WalkDir callback distinguishes root-level failure (propagated as error, caller degrades) from per-entry failure deeper in the tree (skipped) — a refinement over RESEARCH's literal sample, needed to satisfy both the RED test's (0,err) contract and Status()'s degrade-to-0-without-erroring contract
- [Phase 02-status-content-git-worktree-awareness]: Languages re-derivation from FilesByLanguage produced no value-set change on the weft golden corpus — wantLanguages assertion passes unmodified
- [Phase ?]: renderFileTreeMarkdown duplicates internal/cli/files.go's printFileTree rather than importing it, to avoid an internal/query -> internal/cli dependency edge and a wave conflict with plan 02-07
- [Phase 02-status-content-git-worktree-awareness]: OpenAt retains an absolutized start path as Engine.startPath (D-14) — the plumbing fix that delivers CLI+MCP worktree awareness through the single shared read seam
- [Phase 02-status-content-git-worktree-awareness]: WorktreeMismatch cross-call caching lives via an injectable gitmeta.CachingDetector (UseDetector), not solely on Engine's own sync.Once, since MCP's openEngine rebuilds a fresh Engine per tool call (D-13 corrected)
- [Phase 02-status-content-git-worktree-awareness]: StatusResult.WorktreeMismatch type-changed from *string to *gitmeta.Mismatch to match TS's {worktreeRoot,indexRoot}/null shape; T-02-14 accepted as a deliberate, documented host-path-disclosure exception

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

Last session: 2026-07-15T23:51:56.145Z
Stopped at: Completed 02-02-PLAN.md
Resume file: None

## Operator Next Steps

- Review the roadmap draft (.planning/ROADMAP.md).
- Plan the first phase with `/gsd-plan-phase 1`.
