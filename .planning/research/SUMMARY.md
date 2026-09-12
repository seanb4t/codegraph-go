# Project Research Summary

**Project:** CodeGraph Go — v0.13.0 "Guard Hardening & UI Follow-through"
**Domain:** Subsequent (brownfield) milestone on an established Go + Svelte 5 + ConnectRPC + Pebble codebase — seven scoped capabilities: a tmux real-PTY e2e harness (999.2), five "guards that cannot fire" hardening fixes (999.4, T-01-18, dry-run-signed, post-release-verify, tap-secret), four UI follow-through items (BRW-11, BRW-10, HLT-04, GRF-06), and a CLI reference with drift guard (DOCS-05)
**Researched:** 2026-09-08
**Confidence:** HIGH (STACK/ARCHITECTURE grounded directly in this repo's own source at HEAD; FEATURES/PITFALLS MEDIUM-HIGH, cross-checked against comparable tools and this repo's own retrospective)

## Executive Summary

This milestone adds no new architecture — it is entirely integration work against an already-stable seam (`internal/query.Engine`, `internal/uiserver`'s ConnectRPC surface, the additive-only D-02a proto discipline) plus a disciplined pass at making five existing CI/regression guards actually able to fail. Five of the seven capabilities need **zero new dependencies**: the tmux harness wraps the `tmux` CLI directly via `os/exec`, the archtest and workflow-YAML guards reuse dependencies already in `go.mod` (`golang.org/x/tools/go/packages`, `go.yaml.in/yaml/v3`), and the editor-handoff and breadcrumb UI items are thin wrappers over native browser APIs (`<a href>`, `IntersectionObserver`). The only genuinely new dependency across the whole milestone is `gonum.org/v1/gonum/graph/community` (pure Go, no cgo) for GRF-06's Louvain clustering.

The recommended approach, confirmed by direct code reads at HEAD: BRW-10 (breadcrumb) is pure frontend reuse of the already-shipped `FileSymbols` rpc with zero backend changes, smaller than BRW-11 despite the roadmap listing it second. BRW-11 (editor handoff) extends the existing `GetPermalink` precedent with one additive proto field. HLT-04 (coverage denominator) is the one item needing genuinely new Engine surface, split into two sub-problems: an already-persisted-but-unsurfaced category (`schema.File.errors`) and a genuinely-new discovery-time collection with no schema change needed if computed query-time or recorded via the existing per-file secondary index. GRF-06 (community clustering) has an unresolved design question, discussed below, but the schema's own reserved field range 50-59 is prior art for the shape, not necessarily the mechanism, of the answer.

The dominant risk across the milestone is guards that look green but verify nothing. This is the milestone's own named theme, and pitfalls research finds the same failure shape recurring in the new guards this milestone will write if not deliberately avoided: a community-detection test that checks "ran, returned non-empty" instead of "stable across N runs"; a coverage-denominator that infers reasons after the fact instead of recording them at the decision point; a CLI-reference drift guard built on Cobras own doc generator, which silently excludes hidden flags by design. The mitigation pattern is uniform and already established in this repo (rule 84d1gfpywd): demonstrate every guard RED against the real historical failure condition before trusting it green, with a positive control proving the guard can actually detect the thing it claims to check.

## Key Findings

### Recommended Stack

Nearly the entire milestone rides on dependencies already in `go.mod`: `golang.org/x/tools/go/packages` for T-01-18's archtest, following the exact pattern already used by `internal/graphstore/archtest/import_graph_test.go`; `go.yaml.in/yaml/v3` for the workflow `if:` guard, following the existing typed-struct pattern in `internal/upgrade/bench_workflow_shape_test.go`; and Cobras own bundled `doc` subpackage for DOCS-05. The tmux harness (999.2) deliberately avoids any Go tmux client library. `os/exec` wrapping the `tmux` binary directly is recommended, matching this projects existing `os/exec`-based style for git, brew, and task interop, with a new `tmux` build tag (the first feature build tag in this repo) and an explicit tmux-install CI step, since availability on hosted runners is not confirmed.

Core new technologies:
- `gonum.org/v1/gonum/graph/community`, v0.17.0, for GRF-06 Louvain community detection. The only actively-maintained pure-Go graph library with Louvain built in; verified to import no BLAS/LAPACK, no cgo.
- tmux (external CLI, not a Go module) for 999.2's real-PTY harness driver. No viable Go client library exists; wrap the stable CLI contract directly.
- Native IntersectionObserver Web API (zero package) for BRW-10's breadcrumb. A hand-written Svelte 5 action outperforms any npm wrapper package for this size of need.
- Native anchor/`window.location.assign()` (zero package) for BRW-11's editor handoff. The browsers own external-protocol prompt is the safety mechanism; no JS library adds meaningful value.

Total new go.mod requires: 1. Total new package.json entries: 0.

### Expected Features

Must have, per comparable-tool research:
- BRW-11: one-click "open in editor" from the node-detail/source view, with a configurable URI template rather than a hardcoded vscode scheme. Sourcegraphs own settings precedent exists specifically because no single scheme wins across JetBrains, Zed, and remote VS Code users.
- BRW-10: a single-line "you are here" symbol indicator while scrolling. The converged industry answer (VS Code sticky scroll, JetBrains Rider Sticky Lines) to "where am I in this file," scoped to the containing symbol only, not a full nested scope stack (that is an explicitly deferred enhancement).
- HLT-04: a real discovered-vs-indexed denominator with a reason string per skipped file. The schemas own File.errors field, already on disk, was built for exactly this transparency need.
- GRF-06: Louvain-class community detection over the file/package graph, rendered as per-cluster node coloring on the existing directory-structural layout. Never a switch to force-directed layout, which this project has already explicitly and repeatedly rejected.
- DOCS-05: every registered command/flag documented, backed by a non-vacuous drift guard, not a substring match, that structurally verifies presence.

Anti-features, explicitly rejected by comparable-tool research and this repos own architecture:
- Server-side shell-out to launch an editor. Breaks read-only-by-construction (SRV-03); use the browsers own URI-handler dispatch instead.
- Auto-remediation ("re-index this file" button) on the coverage-denominator view. Same SRV-03 violation; show the reason and remedy as text only.
- Persisted cluster assignments written into the graph store at index time using the reserved field range. Violates ENG-03's fresh-per-call discipline (contested, see disagreement below, not settled).
- Full mutation-testing framework adoption project-wide for the guard-hardening theme. Heavyweight for five specifically-diagnosed instances; this repos own lighter-weight hand-authored RED-demonstration convention already works at this scale.

### Architecture Approach

Every one of the seven capabilities maps onto an existing, already-established extension point rather than requiring new architecture: `internal/query.Engine` gains at most one new method for HLT-04, following Status()'s live-filesystem-alongside-store-reads precedent; `internal/uiserver`'s ConnectRPC surface is extended additively (new proto fields on existing messages, never a new rpc where an existing one already carries the needed data); and the two pure-frontend items touch only SourcePane.svelte and sibling routes. The readonly guard in `internal/uiserver/readonly_test.go` has one concrete trap discovered: it forbids the substring "Index" in any method name, so a naive GetIndexCoverage name for HLT-04 fails this guard outright despite being read-only; use GetCoverage instead.

Major components, new or modified:
1. `internal/query.Engine`: one new method for HLT-04's coverage surface; GRF-06's community detection folded into the existing FileGraph() call, following the CycleID precedent exactly.
2. `internal/uiserver`: additive proto/handler extensions only, using the existing mapper-function convention; zero new rpcs across the whole milestone.
3. `internal/tmuxtest/` (new package, recommended location): the tmux harness, gated behind a new build tag, first feature build tag in this repo.
4. Five independent guard-hardening fixes, each confined to its existing file, fully parallelizable with no cross-dependency.
5. `web/src/lib/components/browse/SourcePane.svelte`: mount point for both BRW-10 and BRW-11.

### Critical Pitfalls

1. Community-detection guard asserts "ran" instead of "is stable." Louvain/Leiden are non-deterministic unless seeded and iterated in fixed order; the guard must run the algorithm at least three times on identical input and assert label-canonicalized equality, never just a non-empty-result check.
2. HLT-04's skip reasons become plausible-sounding lies if inferred after the fact. The milestone context names this exact failure mode explicitly; reasons must be captured at the real pipeline decision point, never reconstructed by a query-time re-walk that only has access to static, present-tense file properties.
3. The five "fixed" guards get re-broken by a vacuous repair, for example CheckRegression's zero-metric bypass "fixed" with a bare positivity floor that is itself trivially satisfiable by a different measurement bug. Every fix needs a RED-then-revert demonstration against the actual historical failure condition, not just a green run.
4. tmux capture-pane races the TUI's own render. Never use a fixed sleep; poll with a capture-twice-and-compare stability check. This is the single highest-flake-risk item in the whole milestone.
5. CSP is mistaken for the security boundary on editor links. The actual served CSP has no directive governing top-level navigation to a custom scheme; the real boundary is the browsers native prompt, which "always allow" permanently removes, plus server-side path validation. This needs its own SECURITY.md, since v0.12.0 Phase 1 already had exactly one prior gap of this shape.

## Known Disagreements

### Disagreement 1: GRF-06 clustering, compute fresh per call or persist at index time

Position A, from FEATURES.md and ARCHITECTURE.md: compute fresh, inside FileGraph(), following the CycleID precedent exactly. FileGraph() is documented as built fresh inside every call, with no package-level cache; the FileGraphNode/FileGraphEdge messages carry no reserved clause, meaning the established extension pattern is a plain additive field, exactly how CycleID and InCycle were added. Persisting would mean writing derived analytics into the graph store, reopening the additive-only schema-versioning discipline for a feature that architecturally does not need persistence, and the UI's read-only-by-construction rule (SRV-03) forbids the UI server from writing anything at all.

Position B, from PITFALLS.md: persist at index time, into the schemas already-reserved field range on Node. Louvain/Leiden are non-deterministic by construction (iteration order, tie-breaking, randomness); a fresh-per-call recomputation risks user-visible cluster reshuffling between page loads or across incremental sync runs, eroding the stable, deep-linkable navigation model v0.12.0 established. Persisting once at index time, keyed to a deterministic algorithm run, sidesteps the query-time-determinism problem entirely.

Reconciling note: a fresh-per-call computation using a genuinely deterministic algorithm, fixed sorted iteration order, constant seed, deterministic tie-breaking, and canonical label relabeling, can satisfy both positions' underlying concerns simultaneously: same graph state in, identical output out, every time, with no persistence and no SRV-03 conflict. The persistence question then reduces to a performance decision, whether per-call Louvain is cheap enough on a guava-scale corpus given GRF-01 already found the render pipeline expensive at 3,233 nodes, rather than a correctness one, and that performance question needs its own committed-threshold-before-measuring benchmark before either approach is locked in.

Recommended resolution: default to fresh-per-call with full determinism discipline as the design, since it requires no schema change and matches the established CycleID architecture pattern exactly. Budget an explicit clustering-time-in-isolation benchmark against the guava corpus before committing; if that benchmark shows unacceptable per-request cost, fall back to index-time persistence into the reserved field range as the documented, deliberate escape hatch, not the default. Either way, ship the repeated-run determinism test in the same phase/commit that introduces clustering.

### Disagreement 2: UI item ordering, BRW-11 first or BRW-10

Position A, milestone scope: ordered BRW-11, then BRW-10, then HLT-04, then GRF-06, by increasing scope.

Position B, ARCHITECTURE.md: BRW-10 requires zero backend changes, pure frontend reuse of the already-shipped FileSymbols rpc (confirmed via search returning zero hits for FileSymbols consumption outside the graph route today), no proto edit, no new Engine method, no new fixture entry. BRW-11 touches the server Options struct (new field), a new CLI flag, and an additive proto field even under the cheapest reuse-GetPermalink design. On pure engineering-surface-area grounds, BRW-10 is architecturally the smallest item in the whole follow-through set, smaller than BRW-11, not just smaller than HLT-04/GRF-06.

Recommended resolution: read the milestones "increasing scope" framing as applying to the pair {BRW-11, BRW-10} versus {HLT-04, GRF-06}, small UI polish versus new-surface items, not as a strict internal ordering within the small pair. The roadmapper should sequence BRW-10 before BRW-11 if minimizing time-to-first-demonstrable-item is the goal, since it has no risk of blocking anything else and no proto/fixture surface to review, but either order is defensible since the two items are otherwise independent.

## Open Questions Carried Forward, by source file

From STACK.md:
- Editor URI scheme confidence varies sharply by editor: VS Code/Cursor/VSCodium are high-to-medium confidence (documented or clearly-forked convention); JetBrains is low-to-medium (explicitly undocumented per JetBrains' own community forum); Zed is low, file+line open via URL is an open, unresolved upstream feature request as of this research. BRW-11's config surface should be a template string, not a hardcoded per-editor enum, and must degrade gracefully, not promise support, for Zed.
- Whether GitHub-hosted Ubuntu runners ship tmux preinstalled could not be conclusively verified; treat as absent-by-default and add an explicit install step.
- Whether gonum's module-level supply-chain surface is acceptable is a review-time judgment call; a documented zero-dependency fallback (hand-rolled label propagation) exists if gonum is rejected.

From FEATURES.md:
- Sourcegraph's per-file skip-reason documentation could not be located; HLT-04's design leans more on this repo's own File.errors precedent than on a confirmed external Sourcegraph feature.
- Public documentation of tmux-driven TUI e2e testing for comparable tools (gh, lazygit, k9s, bubbletea) is thin to absent; the assertion classes in this research are reconstructed from tmux's own scripting primitives and this repo's own prior incident record, not an externally-documented convention.
- Whether DOCS-05 should be Cobra-generated or hand-authored-with-guard was explicitly resolved toward hand-authored, but the lower-effort fully-generated alternative remains documented as a question a future maintainer may reasonably re-raise.

From ARCHITECTURE.md:
- Whether the T-01-18 archtest's scope should extend to also forbid a query-to-indexer dependency edge, not just query-to-connectrpc, is unresolved and directly affects how HLT-04's discovery-exclusion helper may be wired; verify at plan time before writing that helper.
- Whether BRW-11 should extend GetPermalink, reusing its existing availability enum which has no real analog for an editor link's binary buildable/not-buildable state, or use a small new purpose-built message, is an open design question to resolve at discuss-phase time.
- Whether the tmux harness lives under an internal-only test package (favored, matches this repo's convention) or a new top-level test directory (would be a first-of-its-kind top-level test directory) is unresolved; no existing sibling precedent either way.

From PITFALLS.md:
- Whether packages.Load's cold-cache latency in CI is a real, measurable cost across the now-five archtest packages is flagged as an inference, not a measured figure; requires a wall-clock measurement before deciding whether to share one load across all five archtests.
- Whether clusters should recompute on every sync or only on a full re-index, directly entangled with Disagreement 1 above, is named as a decision that must be made explicitly and recorded as a generation/version stamp, not left implicit.
- The exact recovery/warm-start strategy for minimizing GRF-06 reshuffle on trivial one-line edits, warm-starting Louvain/Leiden from the prior partition, is proposed but unverified against this project's actual sync-path incremental-recompute machinery; flagged as needing a real test scenario that does not exist today.

## Implications for Roadmap

Based on combined research, the natural phase structure follows the dependency graph ARCHITECTURE.md already derives, organized into five phases.

### Phase 1: Guard Hardening
Rationale: all five items, CheckRegression positivity, T-01-18 archtest, dry-run-signed diff guard, post-release-verify conclusion guard, and the tap-secret test, touch fully disjoint files, have zero dependency on each other or on anything else in the milestone, and are each small, test/CI-config only. Landing first matches this repo's own guards-that-cannot-fire priority and de-risks T-01-18 specifically, since it governs how HLT-04's discovery-exclusion helper may be wired without violating the archtest it establishes.
Delivers: five demonstrably non-vacuous guards, each with a documented RED-then-GREEN demonstration.
Addresses: the milestone's own named guard-hardening theme.
Avoids: guard fixes that repeat the original vacuous-guard defect one layer deeper.

### Phase 2: tmux Real-PTY E2E Harness
Rationale: genuinely new infrastructure, new package, first feature build tag in this repo, new CI job, with no dependency on Phase 1 or on any UI follow-on. It exercises the terminal UI (daemon/install pickers), not the web UI, so it can run in parallel with Phase 1 but is sequenced before UI work per the milestone's own stated rationale.
Delivers: a tmux test harness with alt-screen, escape-sequence-hygiene, keyboard-interaction, and flicker-proxy assertion classes; a CI job installing and version-pinning tmux.
Uses: os/exec wrapping tmux directly; no new go.mod dependency.
Implements: the missing rung between the piped never-hang integration tests and manual human UAT.
Avoids: capture-pane races, DECRQM leakage, OS/version drift, and alt-screen-restore-by-absence pitfalls.

### Phase 3: UI Follow-Through, BRW-10 and BRW-11
Rationale: both are confirmed zero-or-minimal backend surface, independent of each other and of Phase 4's larger items. BRW-10 is the smaller of the two by ARCHITECTURE.md's direct analysis and should be sequenced first within this phase regardless of the roadmap's listed order, per Disagreement 2's resolution.
Delivers: a scroll-tracking breadcrumb and a configurable open-in-editor affordance on the source/node-detail view.
Addresses: FEATURES.md's table-stakes findings for both items.
Avoids: CSP mistaken for the editor-link security boundary, prompt-fatigue steady state, effect double-invocation recurrence, and layout thrash. BRW-11 needs its own SECURITY.md; BRW-10 needs a live-browser UAT gate, not just jsdom.

### Phase 4: UI Follow-Through, HLT-04 and GRF-06
Rationale: both need genuinely new backend/algorithmic surface. HLT-04 depends on Phase 1's T-01-18 archtest scope decision. GRF-06 is the highest-risk item in the milestone, unresolved persistence-vs-fresh-compute design question, and the item most likely to hit GRF-01's already-measured rendering ceiling; it should carry its own measured-threshold checkpoint before UI wiring begins.
Delivers: a discovered-vs-indexed coverage view with per-file reasons; community-clustered coloring on the existing file/package graph view.
Uses: gonum's graph/community package; extends the health response and FileGraph() additively.
Implements: ENG-03's FileGraph() extension pattern; the File.errors read-surfacing pattern.
Avoids: non-deterministic clustering, sync-staleness reshuffle, rendering-wall compounding, lying skip reasons, and divergent discovery-count definitions.

### Phase 5: DOCS-05 CLI Reference and Drift Guard
Rationale: independent of everything above but logically last, since it should document the final flag surface including any new flags Phase 3's BRW-11 introduces.
Delivers: a CLI reference doc plus a completeness guard walking the live Cobra flag-set tree, not diffing Cobra's own generated docs, which silently excludes hidden flags.
Addresses: the deferred DOCS-05 work named at v0.11.0's flag-parity file deletion.
Avoids: a drift guard inheriting Cobra doc-gen's hidden/inherited/conditional/deprecated-flag blind spots.

### Phase Ordering Rationale

- Guard hardening first because it is fully parallelizable, cheapest, and gates how later phases may structure new package dependencies via T-01-18.
- The tmux harness is sequenced before UI work per the milestone's own explicit rationale, though this applies to the terminal UI, not the web UI; the two can genuinely run in parallel if resourcing allows.
- UI follow-through splits into two phases by real dependency weight, not the roadmap's raw listed order: near-zero new surface first, new Engine/algorithmic surface second.
- DOCS-05 last because it documents the flag surface every other phase may have changed.

### Research Flags

Phases likely needing deeper research during planning:
- Phase 4, GRF-06: the persistence-vs-fresh-compute disagreement is unresolved and has real downstream consequences; needs a dedicated measurement pass against the guava-scale corpus before implementation begins.
- Phase 2, tmux harness: thin external precedent; the assertion classes and flake-avoidance primitives are reconstructed first-principles work, not a documented pattern to follow.

Phases with standard, well-documented patterns, skip research-phase:
- Phase 1, guard hardening: every fix has an exact source location and an exact reusable structural precedent already in this repo.
- Phase 5, DOCS-05: a fully proven, previously-shipped pattern exists and is retargetable verbatim; the lowest-risk item in the whole milestone.
- Phase 3, BRW-10 and BRW-11: both have direct, confirmed architectural precedents already shipped; the main open item is the SECURITY.md for BRW-11, a known, named pattern, not novel research.

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | 6 of 7 capabilities verified directly against this repo's own go.mod and source; editor URI schemes are the one medium/low-confidence sub-area, explicitly flagged and mitigated via a configurable-template design |
| Features | MEDIUM | Cross-checked against multiple comparable tools; two areas, Sourcegraph per-file skip reasons and tmux TUI test harnesses in comparable projects, have thin public documentation and are explicitly marked low/inferred |
| Architecture | HIGH | Every claim is either a direct quote from a file at this repository's HEAD, verified via direct reads, or explicitly labelled as inference; no external documentation lookup needed, pure codebase-integration research |
| Pitfalls | MEDIUM-HIGH | Grounded directly in this repo's own code and retrospective for the repo-specific guard-hardening pitfalls; cross-checked against current external sources for genuinely ecosystem-level claims at medium confidence |

Overall confidence: HIGH. This is integration research against a codebase whose own source is the primary and most authoritative source available, and the research consistently cites exact file locations rather than general claims.

### Gaps to Address

- GRF-06's persistence-vs-fresh-compute question is not settled by this research and should be resolved explicitly at discuss-phase or plan-phase time for Phase 4, informed by an actual clustering-time benchmark against the guava corpus; do not let implementation begin without a committed threshold.
- Whether GitHub-hosted CI runners ship tmux preinstalled needs a direct check rather than continued assumption; cheap to verify, should happen early in Phase 2.
- The query-to-indexer dependency-direction question for T-01-18 needs explicit scoping before Phase 4's HLT-04 discovery-exclusion helper is written, since the helper's placement depends on the answer.
- BRW-11's GetPermalink-extension-vs-new-message design question should be resolved at Phase 3 discuss-phase time, since it changes the shape of the change materially.

## Sources

Primary, HIGH confidence:
- This repository at HEAD, tag v0.12.0, commit 17e88d67: the query engine, ui server, schema and wire proto files, indexer, bench regression code, existing archtests, upgrade shape tests, Taskfile, workflow files, and go.mod, all read directly across all four research passes.
- The project's own PROJECT.md, ROADMAP.md, RETROSPECTIVE.md, and pending todos: milestone scope, prior guard-hardening rule, cross-milestone defect-pattern analysis.
- pkg.go.dev and the Go module proxy: gonum import-graph and version verification.

Secondary, MEDIUM confidence:
- Context7 gonum documentation: Louvain algorithm API surface.
- Cobra's official docs and a relevant GitHub issue: doc.GenMarkdownTree hidden-flag exclusion.
- VS Code sticky scroll, JetBrains Rider Sticky Lines, Sourcegraph's open-in-editor feature: cross-vendor UX convergence for BRW-10/BRW-11.
- Gephi and Cytoscape community-detection tooling: GRF-06 rendering convention.
- Relevant Svelte GitHub issues: effect double-invocation, consistent with this repo's own prior findings.
- Web search on tmux DECSET 2026/DECRQM and terminal support: current as of research date.

Tertiary, LOW confidence:
- A Zed GitHub issue: file+line-via-URL is an open, unresolved feature request; do not promise Zed support.
- JetBrains community forum and issue tracker: the JetBrains URI scheme is explicitly undocumented officially.
- Community tmux-wrapper projects: precedent only for the thin-CLI-wrapper pattern.
- GitHub's own runner-images documentation: could not confirm tmux preinstalled on hosted Ubuntu runners; treated as absent-by-default.
- Sourcegraph documentation: could not locate a page specifically enumerating per-file skip reasons; HLT-04 design leans on this repo's own precedent instead.

---
Research completed: 2026-09-08
Ready for roadmap: yes
