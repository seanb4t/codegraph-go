# Roadmap: CodeGraph Go

## Overview

CodeGraph Go is a Go implementation of a pre-indexed code knowledge graph for coding agents. Seven milestones have shipped: **v0.1** (2026-07-14) landed the core capabilities — indexing, query, MCP server, sync — from a signed/attested/SBOM'd release. **v1.0** (2026-08-03) closed the behavioral and surface gaps, adding a human-facing Charm TUI behind a build-enforced rendering seam that keeps the agent/MCP path free of ANSI, plus fully automated signed releases via release-please + GoReleaser. **v0.3.0** (2026-08-06) brought the stdio MCP server current with spec revision `2026-07-28` on `modelcontextprotocol/go-sdk@v1.7.0`, proven by a wire-level oracle that never imports the SDK it tests. **v0.5.0** (2026-08-11) made the binary installable by convention on macOS — Gatekeeper-accepted on both darwin arches and `brew install`-able from a tap we control. **v0.10.0** (2026-08-13) made agents actually *use* the tools: the server documents itself over MCP Resources, a decision-procedure-first SKILL.md teaches which question goes to which tool, and a SessionStart nudge makes availability visible at the moment it matters. **v0.11.0** (2026-08-16) retired the comparison framing without retiring capability, re-basing the golden suite onto measurement-selected corpora and retiring Compatibility as a constraint.

**v0.12.0 (Local Graph UI) shipped 2026-09-07.** Every consumer of the graph until now was a program — a CLI invocation or an agent over MCP. This milestone gave the graph a human face: `codegraph ui` serves a local, read-only web UI from the binary itself, so a developer can browse, visualize, query and health-check a `.codegraph/` index without going through an agent. It is deliberately a **third consumer** of `internal/query.Engine`, never a second implementation — the cross-phase integration check found no duplicated traversal logic in `internal/uiserver`, with every handler routing through one `withEngine` seam. The wire is ConnectRPC over Protobuf (maintainer directive), the app is a `pnpm`-built Svelte SPA committed and `go:embed`'d so the signed release path stays pure Go, and the whole surface binds loopback with exact-match Origin/Host validation because loopback binding alone has already been walked through by a documented CVE class. 51/51 requirements, 6/6 phases verified *and* validated, six per-phase SECURITY.md files all at `threats_open: 0`.

**v0.13.0 (Guard Hardening & UI Follow-through) is in progress, scoped 2026-09-08.** A deferral burn-down in two coherent sets. First, every known guard that cannot fire — this repo's own recurring defect shape is an assertion that is true but non-discriminating, so it passes whether the property holds or not (rule `84d1gfpywd`), and five diagnosed instances plus the real-PTY testing gap are closed here, each demonstrated RED against the actual failure condition before it is called fixed. Second, the four contained UI follow-ons v0.12.0 deliberately left: a scroll breadcrumb, an editor handoff, the coverage denominator that answers "why is my file missing", and community clustering on the graph view. A self-authored `docs/CLI-REFERENCE.md` with its own live-Cobra-tree drift guard is the documentation tail. **Phase numbering continues from v0.12.0 at Phase 7.** Promotes backlog **999.2** and **999.4**.

**Versioning note:** "v1.0" is a *planning-milestone* name, never a release version. The shipped artifact line reached `v0.2.0` at v1.0's close and has since advanced through `v0.3.0`, `v0.4.0`, `v0.5.0` … `v0.9.0`, plus `v0.10.0` and `v0.11.0`, each computed by release-please from Conventional Commits; there is deliberately no `v1.0.0` tag (maintainer directive D-06R, 2026-07-29). Milestone labels track the release line but carry **no git tag**: release-please remains the sole tag authority, pinned by `TestGsdTagCreationIsDisabled`. A hand-created `v*` tag would additionally match `release.yml`'s `push: tags: "v[0-9]*"` trigger and falsely fire the release pipeline. The milestone record lives in `MILESTONES.md` + `milestones/`. (`milestone-v0.1` exists only because it predates release-please.)

## Milestones

- ✅ **v0.1 — Initial Release** — Phases 1–8 (shipped 2026-07-14) — core capabilities + signed release
- ✅ **v1.0 — Drop-in Parity & Human UX** — Phases 1–10 (shipped 2026-08-03) — behavioral + surface parity, human TUI, automated signed releases, local build tooling
- ✅ **v0.3.0 — MCP Protocol Currency** — Phases 1–5 (shipped 2026-08-06) — official Go SDK adoption, `2026-07-28` spec compliance without breaking Legacy clients, a wire-level verification oracle, tool-modfile vulnerability coverage
- ✅ **v0.5.0 — macOS Distribution & Homebrew** — Phases 1–4 (shipped 2026-08-11) — `goreleaser release` migration with zig cross-compilation, Apple notarization, a Homebrew tap and cask, and an `upgrade` that steps aside under brew. Promoted backlog 999.5, consumed SEED-002
- ✅ **v0.10.0 — Agent Onboarding Skill & MCP Resources** — Phases 5–8 (shipped 2026-08-13) — the server documents itself over MCP Resources, a decision-procedure-first SKILL.md plus a SessionStart nudge teach agents when to reach for it, `codegraph install` ships that package with the binary, and the stale `instructions` promise was retired last. Consumed the 2026-08-08 skill todo
- ✅ **v0.11.0 — Standalone Project Identity** — Phases 1–6 (shipped 2026-08-16) — origin acknowledged once in `NOTICE` plus one README License clause; comparison framing removed tree-wide to a proven zero; golden corpora re-selected by measurement, re-frozen from codegraph-go's own output, and re-proven non-vacuous; benchmarks published as mechanically-generated absolute numbers; `codegraph migrate` and the `modernc.org/sqlite` dependency removed (D-04); Compatibility retired as a constraint
- ✅ **v0.12.0 — Local Graph UI** — Phases 1–6 (shipped 2026-09-07) — `codegraph ui` serves a read-only, loopback-bound, ConnectRPC-backed Svelte SPA embedded in the binary: browse and inspect symbols with verbatim source and blast radius, a deep-linkable navigation model, an interactive query workbench, a visual index-health verdict, a file/package graph view, and live push from the watcher over a server-streaming rpc on the same schema and client as every other call. Consumed SEED-001; folded in the CR-01 `pendingWriter` fix
- 🚧 **v0.13.0 — Guard Hardening & UI Follow-through** — Phases 7–12 (in progress) — every known guard that cannot fire closed with a recorded RED demonstration; a tmux real-PTY harness giving the terminal UI its missing rung between the piped TTY-blind suite and manual human UAT; then v0.12.0's four contained UI follow-ons (scroll breadcrumb, editor handoff, coverage denominator, community clustering); with a self-authored CLI reference and its live-tree drift guard as the docs tail. Promotes backlog 999.2 and 999.4; consumes four pending todos, the T-01-18 archtest item, and DOCS-05 (declined at v0.11.0)
- 📋 **Later** — unscoped. Candidates: v0.10.0's v2 deferrals (PreToolUse guard hook GUARD-HOOK-01/02, multi-agent skill+hooks porting AGENT-04…07), GRF-07 (opt-in whole-symbol graph — parked on the v0.12.0 Phase 5 evidence that a 3,233-node file view never converged; revisit only with a measured budget), v0.13.0's own v2 deferrals (BRW-14 nested scope-stack breadcrumb; GRD-07 `requiredCheckNames` vs the live `protect-main` ruleset and GRD-08 root `SECURITY.md`'s govulncheck claim, both *declined for this milestone* by the maintainer rather than forgotten; VOCAB-01 was *declined* at v0.11.0, not deferred), the v0.5.x deferrals (DIST-06 stapled offline-safe container, BREW-07 homebrew-core), Team Scale (central server, CI-distributed indexes), SEED-003 (markdown in the index), MRTR/elicitation (MRTR-01), annotations (embeddings/export)

## Phases

<details>
<summary>✅ v0.1 Initial Release (Phases 1–8) — SHIPPED 2026-07-14</summary>

Archived: [`milestones/v0.1-ROADMAP.md`](./milestones/v0.1-ROADMAP.md)

</details>

<details>
<summary>✅ v1.0 Drop-in Parity & Human UX (Phases 1–10) — SHIPPED 2026-08-03</summary>

Archived: [`milestones/v1.0-ROADMAP.md`](./milestones/v1.0-ROADMAP.md)

</details>

<details>
<summary>✅ v0.3.0 MCP Protocol Currency (Phases 1–5) — SHIPPED 2026-08-06</summary>

Archived: [`milestones/v0.3.0-ROADMAP.md`](./milestones/v0.3.0-ROADMAP.md)

</details>

<details>
<summary>✅ v0.5.0 macOS Distribution & Homebrew (Phases 1–4) — SHIPPED 2026-08-11</summary>

- [x] Phase 1: Cross-Compile Spike & `goreleaser release` Migration (6/6 plans) — REL-05…09
- [x] Phase 2: Apple Signing & Notarization (7/7 plans) — SIGN-01…04
- [x] Phase 3: Homebrew Tap & Cask (5/5 plans) — BREW-01…06
- [x] Phase 4: `codegraph upgrade` × Homebrew (6/6 plans) — UPGR-01…03

Archived: [`milestones/v0.5.0-ROADMAP.md`](./milestones/v0.5.0-ROADMAP.md) · requirements: [`milestones/v0.5.0-REQUIREMENTS.md`](./milestones/v0.5.0-REQUIREMENTS.md) · audit: [`milestones/v0.5.0-MILESTONE-AUDIT.md`](./milestones/v0.5.0-MILESTONE-AUDIT.md)

</details>

<details>
<summary>✅ v0.10.0 Agent Onboarding Skill & MCP Resources (Phases 5–8) — SHIPPED 2026-08-13</summary>

- [x] Phase 5: MCP Resources Capability & Claims Drift Guard (4/4 plans) — RSRC-01…03, GUARD-01…02
- [x] Phase 6: Agent Skill Package — SKILL.md & SessionStart Nudge (4/4 plans) — SKILL-01…03, NUDGE-01…02
- [x] Phase 7: `codegraph install` Skill + Hooks Distribution (Claude Code) (4/4 plans) — AGENT-01…03
- [x] Phase 8: Instructions & Marker-Block Rewrite (3/3 plans) — WIRE-01…03

Archived: [`milestones/v0.10.0-ROADMAP.md`](./milestones/v0.10.0-ROADMAP.md) · requirements: [`milestones/v0.10.0-REQUIREMENTS.md`](./milestones/v0.10.0-REQUIREMENTS.md)

</details>

<details>
<summary>✅ v0.11.0 Standalone Project Identity (Phases 1–6) — SHIPPED 2026-08-16</summary>

- [x] Phase 1: Corpus Selection by Measurement (7/7 plans) — FIXT-01, FIXT-02 — completed 2026-08-14
- [x] Phase 2: Golden Harness Re-authoring & Re-freeze (4/4 plans) — CODE-02, FIXT-04…06 — completed 2026-08-14
- [x] Phase 3: Non-Vacuity Proof & Unconditional CI Execution (2/2 plans) — FIXT-03, FIXT-07 — completed 2026-08-15
- [x] Phase 4: Attribution & Documentation Sweep (3/3 plans) — ATTR-01…03, DOCS-01…04 — completed 2026-08-15
- [x] Phase 5: Process, CI & In-Tree Sweep (8/8 plans) — PROC-01…03, CODE-01, CODE-03 (removed by D-04) — completed 2026-08-16
- [x] Phase 6: Benchmark De-coupling & Memory Sweep (6/6 plans) — BENCH-01…03, MEM-01, MEM-02 — completed 2026-08-16

Archived: [`milestones/v0.11.0-ROADMAP.md`](./milestones/v0.11.0-ROADMAP.md) · requirements: [`milestones/v0.11.0-REQUIREMENTS.md`](./milestones/v0.11.0-REQUIREMENTS.md) · audit: [`milestones/v0.11.0-MILESTONE-AUDIT.md`](./milestones/v0.11.0-MILESTONE-AUDIT.md)

</details>

<details>
<summary>✅ v0.12.0 Local Graph UI (Phases 1–6) — SHIPPED 2026-09-07</summary>

- [x] Phase 1: Engine Seam, Wire Protocol & Secure Transport (11/11 plans) — ENG-01…04, RPC-01/02/05, SRV-01…04, BLD-04, FIX-01 — completed 2026-08-23
- [x] Phase 2: SPA Toolchain, Embedded App Shell & JS Supply Chain (7/7 plans) — RPC-03, BLD-01…03, BLD-05…07 — completed 2026-08-24
- [x] Phase 3: Browse, Inspect & Navigation (10/10 plans) — BRW-01…09, NAV-01…04, SRV-05 — completed 2026-08-29
- [x] Phase 4: Query Workbench & Index Health (7/7 plans) — WRK-01…04, HLT-01…03 — completed 2026-08-30
- [x] Phase 5: File/Package Graph View (8/8 plans, 05-08 added mid-phase as the GRF-01 remedy) — GRF-01…05, ENG-03 — completed 2026-08-31
- [x] Phase 6: Live Push (8/8 plans, 06-08 added post-verification as the criterion-1 gap closure) — RPC-04, LIV-01…04 — completed 2026-09-07

**What the milestone proved, beyond the features:** GRF-01's render threshold was committed *alone, before any measurement* (that file has exactly one commit, an ancestor of both observation commits), then genuinely FAILED at guava scale and was remedied from a pre-written list without touching the bars. Criterion 5's real three-process gate was demonstrated RED on command and its bound is measured, not hardcoded — proven when an independent re-run produced a different baseline and still passed. The read-only guard's 19-verb set is byte-identical from introduction to the milestone's final commit. And a live browser pass found real defects in **four consecutive phases** that headless tests structurally could not — most sharply the root Status route rendering `Stale: no` over a generation-old count, found *after* that phase had verified 5/5.

Archived: [`milestones/v0.12.0-ROADMAP.md`](./milestones/v0.12.0-ROADMAP.md) · requirements: [`milestones/v0.12.0-REQUIREMENTS.md`](./milestones/v0.12.0-REQUIREMENTS.md) · audit: [`milestones/v0.12.0-MILESTONE-AUDIT.md`](./milestones/v0.12.0-MILESTONE-AUDIT.md) · phases: [`milestones/v0.12.0-phases/`](./milestones/v0.12.0-phases/)

</details>

### 🚧 v0.13.0 — Guard Hardening & UI Follow-through (In Progress)

**Milestone Goal:** Burn down the deferral backlog in two coherent sets — close every known guard that cannot fire, then finish the contained UI follow-ons v0.12.0 deliberately left — with a self-authored CLI reference as the documentation tail.

**Phase numbering continues at 7.** v0.12.0 ran Phases 1–6 and its phase directories are archived under `milestones/v0.12.0-phases/`; `.planning/phases/` holds only the two backlog directories (`999.2`, `999.4`), both of which this milestone promotes. Continuing to 7 rather than restarting keeps the promoted backlog work traceable to a phase number that has never been used in this project.

**Ordering is load-bearing, not stylistic:**

- **Guard hardening lands first (Phase 7), and it is the cheapest phase in the milestone.** All six items touch disjoint files, depend on nothing else here, and are test/CI-config only — so the phase most likely to expose a wrong assumption about this repo's own gates runs before any feature work is committed to. It is also load-bearing for Phase 10: `GRD-02`'s archtest fixes the dependency-direction rule `internal/query` must obey, and `HLT-05`'s discovery-exclusion helper has to be wired without violating it. Writing the helper first and the archtest second is how an archtest gets quietly weakened to fit the code it was supposed to constrain.
- **Every guard in this milestone must carry a positive assertion that it did its work** (rule `84d1gfpywd`). A negative-only guard passes vacuously, and this repo has now diagnosed that shape five separate times — in v0.12.0 Phase 5 the same threat survived three green guards. Each of `GRD-01`…`GRD-04` reports a count, or a matched-anchor confirmation, of what it actually inspected, and each is demonstrated RED against a confirmed-applied, byte-cleanly-reverted mutation before it is trusted green. `GRD-06` is that proof committed rather than asserted — four demonstrations with pasted failing output, following `03-MUTATION-LOG.md`. The fifth diagnosed instance (the tap-secret test) is deleted rather than repaired: a low-value property does not earn a rewrite, and a lying test earns removal.
- **The tmux harness lands before the UI work (Phase 8), per the milestone's own stated reason: it gives UI UAT a real-terminal rung.** v0.12.0's most transferable lesson was that a live pass found real defects in *four consecutive phases* that headless tests structurally could not. The harness exercises the *terminal* UI, not the web UI, so it does not technically block Phases 9–11 — but sequencing it first means the milestone's remaining UI work happens in a repo that already has a rung between the TTY-blind piped suite and manual human UAT, rather than one still promising to build one.
- **`TTY-02`'s stability poll is a precondition, not a detail.** `capture-pane` races the TUI's own render, and a fixed sleep is the failure mode; this is the single highest flake risk in the milestone. No assertion in Phase 8 may run against a single capture.
- **The UI follow-through splits by real dependency weight (Phases 9, 10, 11), not by the order the milestone scope lists.** `BRW-10` and `BRW-11`/`12`/`13` are confirmed zero-or-minimal backend surface sharing one mount point (`SourcePane.svelte`), so they go together in Phase 9. `HLT-04`…`06` needs a new Engine surface *and* a write into the indexer's discovery path — the only write-path change in the whole milestone — so it is its own phase. `GRF-06`/`08`/`09`/`10` carries the milestone's only new `go.mod` dependency, its only committed-threshold benchmark, and its only unresolved design fallback, so it is its own phase too. Research proposed folding HLT and GRF into a single phase; they are separated here because their risks are disjoint and neither is verifiable through the other — a green coverage view says nothing about clustering determinism, and a passing clustering benchmark says nothing about whether a skip reason is true.
- **Within Phase 9, `BRW-10` is sequenced before `BRW-11`.** The milestone scope lists `BRW-11` first on an increasing-scope reading, but `BRW-10` is pure frontend reuse of the already-shipped `FileSymbols` rpc — no proto edit, no new Engine method, no CLI flag — while `BRW-11`/`12` touch the server `Options` struct, a new flag, and an additive proto field. "Increasing scope" applies to the pair {BRW-10, BRW-11} versus {HLT, GRF}, not to an internal ordering within the pair.
- **`GRF-09`'s threshold is committed BEFORE its measurement, in its own commit, following `GRF-01`'s precedent exactly.** v0.12.0's signature was that its gates were *allowed to fail*: `GRF-01`'s render threshold was committed alone, ahead of any measurement — that file has exactly one commit, an ancestor of both observation commits, so the ordering is **checkable** rather than assertable — and then genuinely FAILED at guava scale, with the remedy chosen from a list written before measuring and the failing observation preserved. `GRF-09` inherits the whole shape. A failing clustering-time measurement triggers the documented fallback (index-time persistence into the schema's reserved 50-59 field range), never a raised bar.
- **`GRF-06` computes clustering fresh per call, deterministically, inside `FileGraph()`** (maintainer, 2026-09-08) — following the `CycleID` precedent and `ENG-03`'s fresh-per-call discipline. Persistence is `GRF-09`'s documented escape hatch, not the default. `GRF-08`'s determinism test ships in the same phase and the same commit as the clustering itself: a clustering that reshuffles between page loads erodes the deep-linkable navigation model v0.12.0 established, and "it ran and returned something non-empty" is precisely the vacuous assertion this milestone exists to eliminate.
- **`HLT-05` records exclusion reasons at the discovery decision point and persists them additively** (maintainer, 2026-09-08) — never inferred by a query-time re-walk. A reason reconstructed after the fact from static, present-tense file properties is a plausible-sounding lie, and a coverage view whose reasons are lies is worse than no coverage view at all.
- **`HLT-06` must clear the read-only method-set guard by name.** `internal/uiserver`'s `mutatingVerbs` set forbids the substring "Index" in any method name, so a naive `GetIndexCoverage` fails the guard outright despite being read-only. That 19-verb set has been byte-identical since its introduction and is not weakened here; `wantUIServiceMethods` is updated by set-equality in both directions, with the method count asserted from both sides.
- **The docs tail lands last (Phase 12) because it documents the final flag surface,** including `BRW-12`'s new `codegraph ui --editor-url`. `DOCS-06`'s guard walks the *live* Cobra tree — hidden, inherited persistent and deprecated flags included — rather than diffing `cobra/doc`'s generated output, which excludes hidden flags by design and would hand the guard exactly the blind spot it exists to close.
- **`DOCS-07` (the brew-trust note) stays with the docs tail, deliberately.** It arrived as a pending todo alongside the guard todos and carries security framing, so folding it into Phase 7 was considered and rejected: it is a documentation wording change with no guard and no test of its own, and PROJECT.md's own milestone scope groups it under "Docs tail". Keeping it with `DOCS-05`/`06` means one phase owns the docs tree.
- **Backlog 999.2 and 999.4 are promoted, not deleted.** Both entries stay in `## Backlog` below, annotated with the phase that consumed them. This closes the standing "backlog bookkeeping inconsistency" recorded in STATE.md → Blockers: prior promotions (`999.1`, `999.3`, `999.5`, `999.6`) left entries either silently removed or unmarked, and the resulting ambiguity has already cost real time.
- **`v0.13.0` carries no git tag.** release-please is the sole tag authority (D-06R). The label is a prediction that holds because the UI follow-through lands `feat:` commits; a fixes-only outcome would cut `v0.12.1` instead. No phase schedules a `git tag` step.

- [ ] **Phase 7: Guards That Cannot Fire** - Every guard in the known-vacuous set now fails when the property it claims to check is violated, each proven by a recorded RED demonstration rather than a green run
- [ ] **Phase 8: tmux Real-PTY Harness** - The interactive TUI is finally exercised on a terminal that answers escape queries and actually scrolls, closing the gap between the TTY-blind piped suite and manual human UAT
- [ ] **Phase 9: Source View Follow-Through — Breadcrumb & Editor Handoff** - A developer reading a long file always knows which symbol they are inside, and can jump from any node or line straight into their own editor
- [ ] **Phase 10: Index Health — The Coverage Denominator** - "Why is my file missing" is answered by the index itself: how many files were discovered, how many were indexed, and a recorded reason for every gap
- [ ] **Phase 11: Graph View — Community Clustering** - The file/package graph reads as groups rather than a flat mesh, coloured by communities computed deterministically on the layout already shipped
- [ ] **Phase 12: CLI Reference & Docs Tail** - Every flag the binary actually registers is documented in a reference this project authored, kept honest by a walk of the live command tree

## Phase Details

### Phase 7: Guards That Cannot Fire

**Goal**: Every guard in this repo's known cannot-fire set now fails when the property it claims to check is violated — each one proven by a RED demonstration against the actual failure condition, with the proof committed rather than asserted.
**Depends on**: Nothing (first phase of the milestone). **Blocking**: `GRD-02` fixes the dependency-direction rule `internal/query` must obey and therefore gates Phase 10's discovery-exclusion helper — the archtest lands before the code it constrains, never after.
**Requirements**: GRD-01, GRD-02, GRD-03, GRD-04, GRD-06
**Success Criteria** (what must be TRUE):

  1. `internal/bench.CheckRegression` refuses a frame whose *current* throughput or peak-RSS reading is non-positive, with an error naming the degenerate field — proven by a test passing `current.PeakRSSBytes = 0` against an otherwise-matching frame, and watched fail against the pre-fix build where both the relative RSS check and the absolute INDX-06 ceiling silently return `nil` (GRD-01)
  2. An archtest asserts `internal/query` imports no wire-layer package (`internal/uiserver`, `connectrpc.com/connect`, `internal/mcp`), reports the number of packages it actually loaded, and carries a positive control that fails when the expected importer disappears — so a load returning zero packages cannot read as a pass (GRD-02)
  3. `release:dry-run-signed`'s additions-only diff guard asserts that its awk anchor matched and that the `--key=` injection actually occurred, reporting what it matched; deleting or renaming the anchor turns the guard red instead of leaving it green (GRD-03)
  4. A test parses `post-release-verify.yml`, reports how many jobs it inspected, and fails when the event-aware conclusion guard is removed or inverted on any one of them — it cannot pass by comparing constants it wrote itself (GRD-04)
  5. A committed mutation log carries, for each of the four guards, the pasted failing output from its RED demonstration plus evidence of a byte-clean revert, following `03-MUTATION-LOG.md`'s precedent — four demonstrations, none of them summarised away (GRD-06)

**Notes**: Every fix has an exact source location and an exact reusable structural precedent already in-tree — `internal/graphstore/archtest/import_graph_test.go` for `GRD-02`, `internal/upgrade/bench_workflow_shape_test.go` for `GRD-04` — so this phase should skip the research pass. `GRD-01`'s fix must not itself be vacuous: a bare positivity floor is trivially satisfiable by a different measurement bug, so the test exercises the historical degenerate frame rather than a synthetic one. A project-wide mutation-testing framework is out of scope by construction; the hand-authored RED-demonstration convention already works at this scale. Consumes backlog **999.4**, three pending todos (dry-run-signed, post-release-verify, tap secret — the last resolved by deleting the tautological test, not rewriting it) and the T-01-18 archtest item. `GRD-07`/`GRD-08` were declined for this milestone by the maintainer and are recorded in REQUIREMENTS.md → v2, not silently dropped; `GRD-05` joined them at the Phase 7 discussion (2026-09-08). The `query`→`indexer` scoping question was resolved at that discussion: the archtest forbids the `internal/indexer` root and allows the `goextract`/`nodeid` leaves (07-CONTEXT.md D-01), which fixes where Phase 10's helper may live.
**Plans**: 3/4 plans executed

Plans:
**Wave 1**

- [x] 07-01-PLAN.md — TRACER: GRD-01 `CheckRegression` current-metrics positivity, watched RED against the pre-fix build, and `07-MUTATION-LOG.md` created with family (a)

**Wave 2** *(blocked on Wave 1 completion)*

- [x] 07-02-PLAN.md — GRD-02 `internal/query` dependency-direction archtest over the resolved transitive set, RED-proven on both the wire-layer and indexer-root rules, family (b)

**Wave 3** *(blocked on Wave 2 completion)*

- [x] 07-03-PLAN.md — GRD-03 `scripts/inject-cosign-key.sh` extraction plus the exactly-one injected-key count assertion, both Task targets rewired, family (c)

**Wave 4** *(blocked on Wave 3 completion)*

- [ ] 07-04-PLAN.md — GRD-04 per-job conclusion-guard test (removed and inverted both RED-proven), the tautological tap test deleted per D-09, family (d) and the log closed

### Phase 8: tmux Real-PTY Harness

**Goal**: The release binary's interactive TUI is exercised inside a real tmux pane that answers escape queries and actually scrolls — the missing rung between the piped, TTY-blind integration suite and manual human UAT.
**Depends on**: Nothing in this milestone; fully independent of Phase 7 and could run beside it. Sequenced second by the milestone's own rationale — it lands before the UI phases so that work has a real-terminal rung.
**Requirements**: TTY-01, TTY-02, TTY-03, TTY-04, TTY-05, TTY-06, TTY-07
**Success Criteria** (what must be TRUE):

  1. A build-tagged harness builds the binary, spawns it in a tmux pane, sends keys and captures the pane; with `tmux` absent from `PATH` the suite reports a skip with a stated reason and a skipped-case count, never a silent pass — demonstrated by running it on a `PATH` with tmux removed (TTY-01)
  2. Every frame assertion polls until two consecutive `capture-pane` results are byte-identical before asserting, with no fixed sleep anywhere in the assertion path — proven against a case that races and fails under a single naive capture (TTY-02)
  3. Bare `codegraph daemon` on a real TTY with an empty registry renders only the `no running daemons` line with zero DECRQM/mode-query response bytes in pane or scrollback; and the daemon picker enters the alternate screen, renders `Running daemons` plus a seeded record, and on quit restores the main buffer with no residual escape sequences — both watched fail against the historical G-07-1 / G-07-2 behaviour (TTY-03, TTY-04)
  4. The install/uninstall checkbox picker renders `[x]`/`[ ]` glyphs, `space` flips the glyph, and `q`/`esc` cancel with a before/after hash of the config tree proving zero writes; an idle picker holds frame-stable across N captures, with N reported (TTY-05, TTY-06)
  5. A CI job installs a pinned tmux version, runs the suite, and asserts a positive count of *executed* — not skipped — test cases, so a runner without tmux fails the job rather than passing it empty (TTY-07)

**Notes**: Research flag — external precedent for tmux-driven TUI e2e testing is thin to absent, so the assertion classes are reconstructed from tmux's own scripting primitives plus this repo's incident record rather than copied from a documented convention. `os/exec` wraps the stable tmux CLI directly, matching this repo's existing git/brew interop style; a Go tmux client library is out of scope by construction because no viable one exists. This introduces the first *feature* build tag in the repo. Whether GitHub-hosted runners ship tmux could not be confirmed and is treated as absent-by-default — verify early, since `TTY-07` depends on the answer. Consumes backlog **999.2** and closes the G-07-1 / G-07-2 classes that v1.0 Phase 7's human UAT caught after both the full piped suite and a deep multi-agent code review had missed them.
**Plans**: TBD

### Phase 9: Source View Follow-Through — Breadcrumb & Editor Handoff

**Goal**: A developer reading a long file always knows which symbol they are inside, and can hand any node or line off to their own editor in one click — with the security boundary named correctly and the absolute path never weakening repo-root confinement at the RPC boundary.
**Depends on**: Phase 8 (sequencing only — the real-terminal rung exists before the UI work; the two phases touch disjoint trees and share no code)
**Requirements**: BRW-10, BRW-11, BRW-12, BRW-13
**Success Criteria** (what must be TRUE):

  1. Scrolling a long file's source updates a single-line breadcrumb naming the innermost containing symbol, computed client-side from `FileSymbols` line ranges — verified in a live browser against a real index, not only in jsdom, and demonstrated wrong (stale or empty) against the pre-fix build (BRW-10)
  2. Node detail and source views offer an "open in editor" link built from a `{path}`/`{line}`/`{col}` template resolved server-side, and a traversal-shaped or out-of-root path is refused at the RPC boundary before it ever reaches the template — asserted by a test that names the rejected input and fails if the request succeeds (BRW-11)
  3. `codegraph ui --editor-url <template>` sets the default and a per-browser override replaces it for that browser only; presets exist for VS Code, Cursor and JetBrains, and a check reports the preset count and finds Zed in neither the preset list nor any claim the UI makes (BRW-12)
  4. A phase `SECURITY.md` names the browser's external-protocol prompt — not CSP — as the security boundary and covers validation of the template's inputs, with every named threat carrying a test or a recorded verdict rather than prose (BRW-13)

**Notes**: `BRW-10` is sequenced first within the phase despite the milestone scope listing `BRW-11` first: it is pure frontend reuse of the already-shipped `FileSymbols` rpc with no proto edit, no new Engine method and no CLI flag, so it is the smallest demonstrable item in the whole follow-through set. Whether `BRW-11` extends `GetPermalink` — whose availability enum has no real analog for an editor link's buildable/not-buildable state — or uses a small purpose-built message is an open design question to settle at discuss-phase time. Zed is deliberately neither a preset nor a claimed target: file+line open via URL is an open upstream request, and a preset would be a promise this project cannot keep. Server-side shell-out to launch an editor is out of scope by construction — it breaks read-only-by-construction (`SRV-03`); the browser's own URI-handler dispatch is the mechanism. `BRW-13`'s SECURITY.md is not optional paperwork: v0.12.0 Phase 1 shipped with no `01-SECURITY.md` despite 11 plans carrying threat models, the only such omission in project history, and it was caught only by the milestone audit.
**Plans**: TBD
**UI hint**: yes

### Phase 10: Index Health — The Coverage Denominator

**Goal**: The index answers "why is my file missing" itself — a real discovered-versus-indexed denominator with a recorded reason for every gap, distinguishing extraction failures from pre-extraction exclusions.
**Depends on**: Phase 7 (`GRD-02`'s archtest fixes the dependency-direction rule the discovery-exclusion helper must be wired within) and Phase 9 (sequencing by increasing dependency weight — the near-zero-surface UI items land first)
**Requirements**: HLT-04, HLT-05, HLT-06
**Success Criteria** (what must be TRUE):

  1. Index health shows discovered-versus-indexed counts and lists each unindexed file with a reason, visibly distinguishing extraction failures already persisted in `File.errors` from pre-extraction exclusions — demonstrated against a fixture repo containing at least one file of each kind, with the counts reported rather than implied (HLT-04)
  2. Exclusion reasons (vendor/dot-dir, unsupported extension, build tag, size limit) are written at the discovery decision point and read back from the store, surviving a process restart with no re-walk — proven by a test that fails if the reason is reconstructed at query time, since a query-time re-walk cannot distinguish a file excluded by build tag from one that simply is not there (HLT-05)
  3. The persisted shape is additive within SchemaVersion 1: a graph written before this phase still opens and reports its coverage as unknown rather than erroring or claiming zero, following the `HasFileIndex` precedent (HLT-05)
  4. The coverage surface extends `GetHealthResponse` additively, or adds an rpc whose name clears every `mutatingVerbs` substring including "Index", with `wantUIServiceMethods` updated by set-equality in both directions and the method count asserted from both sides — so neither an added nor a removed method can slip through (HLT-06)

**Notes**: This is the milestone's only write-path change, and its central risk is named explicitly: reasons inferred after the fact are plausible-sounding lies, so they are captured at the real pipeline decision point and never reconstructed by a query-time walk that only sees static, present-tense file properties. The discovered-count definition must be pinned once and used by both the denominator and the reason list, or the two disagree in the user's face. "Re-index this file" auto-remediation on the coverage view is out of scope by construction — the same `SRV-03` violation as an editor shell-out; show reason and remedy as text only. Where the discovery-exclusion helper may live depends on Phase 7's landed archtest scope; check it against the archtest as written, not as remembered.
**Plans**: TBD
**UI hint**: yes

### Phase 11: Graph View — Community Clustering

**Goal**: The file/package graph reads as groups rather than a flat mesh — nodes coloured by community, computed fresh and deterministically inside `FileGraph()`, on the layered layout that already shipped.
**Depends on**: Phase 10 (sequencing — last of the UI follow-ons by dependency weight and risk). **Blocking within the phase**: `GRF-09`'s pass threshold is committed in its own commit before any measurement runs, and no clustering may be wired into the UI before that measurement resolves.
**Requirements**: GRF-06, GRF-08, GRF-09, GRF-10
**Success Criteria** (what must be TRUE):

  1. A threshold for clustering time in isolation against the guava corpus exists as its own commit, an ancestor of every measurement commit — so the before/after ordering is checkable from git rather than asserted — and the recorded measurement's verdict is preserved whichever way it falls; a failing verdict triggers the documented index-time-persistence fallback into the reserved 50-59 field range, never a raised bar (GRF-09)
  2. Opening the graph view on a real repository shows nodes coloured by community on the existing layered layout, with the layout algorithm itself unchanged — asserted by a check that reports the number of distinct communities rendered and finds no force-directed mode reachable from any code path (GRF-06)
  3. Running the clustering three or more times on identical input yields label-canonicalized-equal assignments, with the test reporting how many runs it compared; a deliberately perturbed iteration order or seed turns the test red rather than leaving it green (GRF-08)
  4. `gonum.org/v1/gonum` passes govulncheck, appears by name in the generated SBOM, and a check over its import closure reports the number of packages it inspected and finds zero cgo — so an empty closure cannot read as a clean one (GRF-10)

**Notes**: Research flag — this is the highest-risk item in the milestone and the persistence-versus-fresh-compute question is settled by maintainer decision (2026-09-08: fresh per call, deterministically, inside `FileGraph()`, following the `CycleID` precedent and `ENG-03`'s discipline) but its *performance* premise is not, which is exactly what `GRF-09` measures. Persisted cluster assignments as the default are out of scope by construction; they are retained only as `GRF-09`'s documented fallback. Force-directed layout is out of scope by construction, repeatedly rejected across v0.12.0 — clustering is rendered as colouring on the layered layout, never as a layout change. `gonum` is the milestone's only new `go.mod` require; a documented zero-dependency fallback (hand-rolled label propagation) exists if the supply-chain review rejects it. Whether clusters recompute on every sync or only on a full re-index must be decided explicitly and recorded, not left implicit.
**Plans**: TBD
**UI hint**: yes

### Phase 12: CLI Reference & Docs Tail

**Goal**: Every command and flag the binary actually registers is documented in a reference this project authored, kept honest by a walk of the live Cobra tree rather than by a generator that shares the reference's blind spots — and the brew-trust instructions recommend the narrow grant with security framing.
**Depends on**: Phase 9 (`BRW-12` adds `codegraph ui --editor-url`, which the reference must document) and, in practice, every earlier phase, since the reference describes the final flag surface
**Requirements**: DOCS-05, DOCS-06, DOCS-07
**Success Criteria** (what must be TRUE):

  1. `docs/CLI-REFERENCE.md` documents every command and flag in the Cobra tree — including this milestone's new `--editor-url` — replacing what `docs/FLAG-PARITY.md` used to carry before it was deleted at v0.11.0 (DOCS-05)
  2. A drift guard walks the live Cobra tree including hidden, inherited persistent and deprecated flags, reports the number of flags it inspected, and fails when any registered flag is missing from the reference — demonstrated RED by registering a throwaway *hidden* flag, which `cobra/doc`'s own generated output would silently omit, then byte-cleanly reverting it (DOCS-06)
  3. The brew-trust instructions recommend the narrow grant with security framing rather than the broader `--tap` grant, and the previously-recommended broader form appears nowhere in the docs tree — asserted by a check that reports its match count and is proven able to find the old form when it is present (DOCS-07)

**Notes**: A Cobra-generated reference as the source of truth is out of scope by construction: `cobra/doc` silently excludes hidden flags by design, so a guard built on it inherits precisely the blind spot it exists to close. `DOCS-05` was deliberately *declined* at v0.11.0 — recorded, not forgotten — when `docs/FLAG-PARITY.md` and its drift guard `internal/cli/flag_parity_test.go` were deleted; this phase is the deferred replacement, not a new idea. Lowest-risk phase in the milestone: a fully proven, previously-shipped guard pattern is retargetable here rather than invented. `DOCS-07` was considered for Phase 7 (it arrived as a pending todo beside the guard todos and carries security framing) and deliberately kept here so one phase owns the docs tree.
**Plans**: TBD

## Progress

**Execution Order:**
Phases execute in numeric order: 7 → 8 → 9 → 10 → 11 → 12. Only two edges are genuine code dependencies — Phase 10 needs Phase 7's archtest to exist before its discovery-exclusion helper is wired, and Phase 12 needs Phase 9's `--editor-url` flag to exist before it documents the final flag surface. The rest is deliberate sequencing: guard hardening first because it is the cheapest phase and the one most likely to expose a wrong assumption about this repo's own gates; the tmux harness before the UI work because it gives UI UAT a real-terminal rung (the milestone's own stated reason); and the UI follow-ons ordered by increasing dependency weight rather than by the order the milestone scope lists them. Phases 7 and 8 are fully independent of each other and could run in parallel if resourcing allows. Within Phase 9, `BRW-10` precedes `BRW-11`; within Phase 11, `GRF-09`'s threshold commit precedes every measurement and blocks all UI wiring.

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 7. Guards That Cannot Fire | 3/4 | In Progress|  |
| 8. tmux Real-PTY Harness | 0/TBD | Not started | - |
| 9. Source View Follow-Through — Breadcrumb & Editor Handoff | 0/TBD | Not started | - |
| 10. Index Health — The Coverage Denominator | 0/TBD | Not started | - |
| 11. Graph View — Community Clustering | 0/TBD | Not started | - |
| 12. CLI Reference & Docs Tail | 0/TBD | Not started | - |

7 milestones shipped. v0.13.0 scoped: 6 phases (7–12), 27 requirements, 0/6 phases complete (0%). Backlog below is preserved across milestone closes.

## Backlog

### Phase 999.2: tmux e2e/UAT test harness and suite (BACKLOG)

**PROMOTED 2026-09-08 → v0.13.0 Phase 8 (tmux Real-PTY Harness), scoped as TTY-01…TTY-07.** Kept here rather than deleted: this project's own record shows removed or unmarked backlog entries have cost real time (STATE.md → Blockers, "Backlog bookkeeping inconsistency"). The captured goal below stands as written and is the direct source of Phase 8's four assertion classes.

**Goal:** [Captured for future planning] A real-PTY end-to-end test harness that drives the interactive TUI through **tmux** (send-keys + capture-pane) so the terminal actually replies to escape queries and actually scrolls — the exact conditions the current piped/non-TTY suite can never reproduce. Motivation: v1.0 Phase 7's human UAT caught two user-visible TUI bugs that BOTH the full piped automated suite AND a deep multi-agent code review missed, because they only manifest on a live TTY — G-07-1 (bare `daemon` on a TTY with an empty registry leaked the terminal's DECRQM capability-probe responses `^[[?2026;2$y^[[?2027;0$y`) and G-07-2 (both bubbletea pickers rendered inline without alt-screen → heavy flicker + blank list). bubbletea Models are unit-testable via synthetic `tea.Msg` (state transitions) but that path never renders. Scope a suite that spawns the release binary inside a tmux pane and asserts on `capture-pane` output: (a) bare `daemon` empty-registry prints ONLY `no running daemons` with no leaked escape sequences; (b) the daemon picker enters the alternate screen, renders `Running daemons` + a seeded record, and restores the main buffer on quit (no residual escapes in scrollback); (c) the install/uninstall checkbox picker renders `[x]`/`[ ]` glyphs, `space` toggles, `q`/`esc` cancels with zero config writes; (d) no flicker proxy (stable capture across N frames). Reuse the `tmux` skill's send-keys/capture-pane idioms; gate the suite behind a build tag / CI job that has tmux available (skip cleanly where it isn't). This is the missing rung between the piped never-hang/byte-identity integration tests (necessary, TTY-blind) and manual human UAT (thorough, unautomated).
**Requirements:** TBD
**Plans:** 0 plans

Plans:

- [ ] TBD (promote with /gsd-review-backlog when ready)

### Phase 999.4: CheckRegression current-metrics positivity guard (BACKLOG)

**PROMOTED 2026-09-08 → v0.13.0 Phase 7 (Guards That Cannot Fire), scoped as GRD-01 with its RED demonstration recorded under GRD-06.** Kept here rather than deleted, for the same reason as 999.2 above. The captured goal below stands as written — in particular its insistence that the fix be demonstrated RED with a degenerate-input test rather than merely added.

**Goal:** [Captured for future planning] Close the degenerate-input bypass in `internal/bench.CheckRegression`, surfaced and **reproduced** during the Phase 10 security audit (recorded in `10-SECURITY.md` → "Advisory — Unregistered Surface"; also code-review finding WR-06). Calling `CheckRegression(baseline, current, ceiling=1)` with `current.PeakRSSBytes = 0` and an otherwise-matching frame returns `nil` — **both** the relative RSS regression check and the absolute INDX-06 memory ceiling silently pass. The function already validates that the *baseline* metrics are positive; it never validates the *current* ones, so a zero or negative current reading reads as "no regression" instead of "unusable measurement". This is unreachable through today's only caller because `internal/bench.PeakRSSBytes` returns an error rather than a zero on failure, but `CheckRegression` is exported, its doc comment claims it "never misleads", and the phase-10 audit already showed how easily a frame-descriptor blind spot becomes a live gate failure. Scope: add a positivity/sanity check on `current` mirroring the existing baseline check, refusing rather than passing on a non-positive throughput or RSS reading, with an error naming which field was degenerate. This belongs to the repo's documented class of **gates that cannot fire** (the retracted 10.6% perf claim, the inverted `rg -qv` gate, the 51.5%-stale baseline) — so the fix must be demonstrated RED with a degenerate-input test, not merely added.
**Requirements:** TBD
**Plans:** 0 plans

Plans:

- [ ] TBD (promote with /gsd-review-backlog when ready)
