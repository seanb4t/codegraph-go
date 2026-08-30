# Roadmap: CodeGraph Go

## Overview

CodeGraph Go is a Go implementation of a pre-indexed code knowledge graph for coding agents. Six milestones have shipped: **v0.1** (2026-07-14) landed the core capabilities — indexing, query, MCP server, sync — from a signed/attested/SBOM'd release. **v1.0** (2026-08-03) closed the behavioral and surface gaps, adding a human-facing Charm TUI behind a build-enforced rendering seam that keeps the agent/MCP path free of ANSI, plus fully automated signed releases via release-please + GoReleaser. **v0.3.0** (2026-08-06) brought the stdio MCP server current with spec revision `2026-07-28` on `modelcontextprotocol/go-sdk@v1.7.0`, proven by a wire-level oracle that never imports the SDK it tests. **v0.5.0** (2026-08-11) made the binary installable by convention on macOS — Gatekeeper-accepted on both darwin arches and `brew install`-able from a tap we control. **v0.10.0** (2026-08-13) made agents actually *use* the tools: the server documents itself over MCP Resources, a decision-procedure-first SKILL.md teaches which question goes to which tool, and a SessionStart nudge makes availability visible at the moment it matters.

**v0.11.0 (Standalone Project Identity) shipped 2026-08-16.** Everything the project *did* already stood on its own; the way it *described and tested itself* did not. This milestone retired that framing without retiring capability — with one recorded exception. The origin is acknowledged exactly once, legally and in the past tense, in `NOTICE` plus one clause in README's `## License`; comparison vocabulary is gone from every doc, template, workflow, script, comment, identifier and fixture, closed by a positive-controlled census reporting `TOTAL=0` across 285 Go files. The load-bearing part was never prose: the golden suite derived its oracle from corpora chosen because the origin project used them. Re-basing it meant selecting corpora **by measurement** — recorded per-kind edge counts and per-language file counts from real indexing runs — and then re-proving the re-baselined suite could still fail. The exception is `codegraph migrate`, dropped outright (maintainer ruling D-04) because the migration path *was itself* the parity framing. The **Compatibility constraint is retired**: behavior is now defined by this project's own requirements and its own frozen goldens.

**v0.12.0 (Local Graph UI) is in progress.** Every consumer of the graph so far has been a program — a CLI invocation or an agent over MCP. This milestone gives the graph a human face: a local, read-only web UI served by the binary itself, so a developer can browse, visualize, query and health-check a `.codegraph/` index without going through an agent. It is deliberately a third consumer of `internal/query.Engine`, never a second implementation of it — the same seam `internal/cli` and `internal/mcp` already share. The wire is ConnectRPC over Protobuf (maintainer directive), the app is a `pnpm`-built Svelte SPA committed and `go:embed`'d so the signed release path stays pure Go, and the whole surface binds loopback with exact-match Origin/Host validation because loopback binding alone has already been walked through by a documented CVE class.

**Versioning note:** "v1.0" is a *planning-milestone* name, never a release version. The shipped artifact line reached `v0.2.0` at v1.0's close and has since advanced through `v0.3.0`, `v0.4.0`, `v0.5.0` … `v0.9.0`, plus `v0.10.0` and `v0.11.0`, each computed by release-please from Conventional Commits; there is deliberately no `v1.0.0` tag (maintainer directive D-06R, 2026-07-29). Milestone labels track the release line but carry **no git tag**: release-please remains the sole tag authority, pinned by `TestGsdTagCreationIsDisabled`. A hand-created `v*` tag would additionally match `release.yml`'s `push: tags: "v[0-9]*"` trigger and falsely fire the release pipeline. The milestone record lives in `MILESTONES.md` + `milestones/`. (`milestone-v0.1` exists only because it predates release-please.)

## Milestones

- ✅ **v0.1 — Initial Release** — Phases 1–8 (shipped 2026-07-14) — core capabilities + signed release
- ✅ **v1.0 — Drop-in Parity & Human UX** — Phases 1–10 (shipped 2026-08-03) — behavioral + surface parity, human TUI, automated signed releases, local build tooling
- ✅ **v0.3.0 — MCP Protocol Currency** — Phases 1–5 (shipped 2026-08-06) — official Go SDK adoption, `2026-07-28` spec compliance without breaking Legacy clients, a wire-level verification oracle, tool-modfile vulnerability coverage
- ✅ **v0.5.0 — macOS Distribution & Homebrew** — Phases 1–4 (shipped 2026-08-11) — `goreleaser release` migration with zig cross-compilation, Apple notarization, a Homebrew tap and cask, and an `upgrade` that steps aside under brew. Promoted backlog 999.5, consumed SEED-002
- ✅ **v0.10.0 — Agent Onboarding Skill & MCP Resources** — Phases 5–8 (shipped 2026-08-13) — the server documents itself over MCP Resources, a decision-procedure-first SKILL.md plus a SessionStart nudge teach agents when to reach for it, `codegraph install` ships that package with the binary, and the stale `instructions` promise was retired last. Consumed the 2026-08-08 skill todo
- ✅ **v0.11.0 — Standalone Project Identity** — Phases 1–6 (shipped 2026-08-16) — origin acknowledged once in `NOTICE` plus one README License clause; comparison framing removed tree-wide to a proven zero; golden corpora re-selected by measurement, re-frozen from codegraph-go's own output, and re-proven non-vacuous; benchmarks published as mechanically-generated absolute numbers; `codegraph migrate` and the `modernc.org/sqlite` dependency removed (D-04); Compatibility retired as a constraint
- 🚧 **v0.12.0 — Local Graph UI** — Phases 1–6 (in progress) — `codegraph ui` serves a read-only, loopback-bound, ConnectRPC-backed Svelte SPA embedded in the binary: browse and inspect symbols with verbatim source and blast radius, a deep-linkable navigation model, an interactive query workbench, a visual index-health verdict, a file/package graph view, and live push from the watcher. Consumes SEED-001; folds in the CR-01 `pendingWriter` fix
- 📋 **Later** — unscoped. Candidates: v0.10.0's v2 deferrals (PreToolUse guard hook GUARD-HOOK-01/02, multi-agent skill+hooks porting AGENT-04…07), v0.11.0's v2 deferrals (DOCS-05 self-authored CLI reference; VOCAB-01 was *declined*, not deferred), v0.12.0's v2 deferrals (HLT-04 coverage denominator, BRW-10 breadcrumb, BRW-11 editor handoff, GRF-06 community clustering, GRF-07 whole-symbol graph), the Backlog items below (999.2 tmux TTY harness, 999.4 CheckRegression guard), the v0.5.x deferrals (DIST-06 stapled offline-safe container, BREW-07 homebrew-core), Team Scale (central server, CI-distributed indexes), MRTR/elicitation (MRTR-01), annotations (embeddings/communities/export)

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

### 🚧 v0.12.0 — Local Graph UI (In Progress)

**Milestone Goal:** Give the graph a human face — a local, read-only web UI served by the binary itself, so a developer can browse, visualize, query and health-check a `.codegraph/` index without going through an agent.

**Phase numbering restarts at 1.** v0.11.0 also ran Phases 1–6; this milestone restarts rather than continuing to 7. `.planning/phases/` holds only backlog directories (`999.x`) at the milestone boundary, so Phases 1–6 sort unambiguously and collide with nothing.

**Ordering is load-bearing, not stylistic:**

- **The Engine seam is extracted first, and it is a pure extraction (Phase 1).** `Node()` and `Explore()` already compute structured intermediate data before rendering — `fetchCalls`/`fetchCalledBy` and `groups`/`blasts`/`sources` respectively. `ENG-01`/`ENG-02` expose those pieces as new methods that stop *before* the existing render call, leaving `Node()`/`Explore()` byte-identical and the frozen golden suite untouched. Almost every browse and inspect requirement depends on them, and the extraction is only cheap while it stays an extraction — a rewrite that re-derives the same data is how a second implementation of the query engine gets born.
- **Origin/Host validation lands in the transport phase, before any RPC handler exists (Phase 1, `SRV-02`).** Retrofitting a DNS-rebinding defense after handlers are already serving is precisely how CVE-2024-28224 (Ollama) and CVE-2025-66414/66416 (MCP SDKs) happened. It is one control, not three: exact-match allowlist, no `Contains`/`HasPrefix`/wildcard regex, and `localhost` / `127.0.0.1` / `[::1]` each admitted explicitly rather than treated as interchangeable. Loopback binding is not the boundary and is not treated as one.
- **`FIX-01` (CR-01) lands early and deliberately out of order (Phase 1).** The `internal/mcp/server.go` `pendingWriter` bug is independent of everything else here, but its root cause — server-initiated writes decrementing a counter only client-initiated requests increment — is exactly the shape the new ConnectRPC server-streaming surface can reinvent. Fixing it first makes it a worked example the live-push design reviews against (Phase 6), rather than a rushed fix landing beside integration-heavy UI work at the end.
- **Two distinct findings are both named CR-01, and they must never be conflated.** This milestone's is `internal/mcp/server.go`'s `pendingWriter` corruption — **open**, scoped as `FIX-01`. The CR-01 cited throughout `internal/graphstore/pebble_store.go` is a v0.5.0 Phase-3 finding about the Pebble directory lock — **already fixed**; the bounded retry loop *is* that fix.
- **No IPC-to-daemon architecture is needed, and none is scoped.** `pebble.Open` takes an exclusive directory lock regardless of `ReadOnly` (cockroachdb/pebble#1583), but nothing in this module holds the store long-term: `graphstore.Open` already wraps `pebble.Open` in a bounded retry (5 × 100ms) classifying lock-held failures as the exported `ErrStoreLocked`, precisely because `query.OpenAt`, `indexer.Sync` and the startup reconcile collide by design. `codegraph ui` opens-snapshots-closes per RPC exactly as `internal/mcp`'s `openEngine` already does (`SRV-04`), and never caches an `Engine`. The research pitfall claiming this needs an IPC design was investigated and is wrong.
- **Both drift guards ship in the same phase as the artifact they guard,** per this repo's established v0.10.0 pattern ("the claims-drift guard ships in the same phase as the resources it gates"), not deferred to a cleanup phase at the end. `BLD-04` (protobuf codegen) ships in Phase 1 with the new `.proto`; `BLD-03` (the committed SPA build output) ships in Phase 2 with the committed `web/build/`. `BLD-04` also closes a **pre-existing** gap: there is no proto codegen drift guard in this repo today and no regeneration task in `Taskfile.yml` — `roundtrip_test.go` tests serialization round-tripping, not codegen currency — so the guard must cover the pre-existing `internal/schema/graph.proto` as well as the new UI schema.
- **Every guard in this milestone must carry a positive assertion that it did its work** (rule `84d1gfpywd`). A negative-only guard passes vacuously, and this repo already carries that defect class more than once. `BLD-03`, `BLD-04`, `BLD-05` and `BLD-06` each report a count of what they actually inspected, and each is demonstrated RED against a confirmed-applied, byte-cleanly-reverted staleness before it is trusted green. `pnpm audit` is the sharpest case: its own documented exit-code behavior makes "scanned and clean" and "the registry call failed" indistinguishable, so `BLD-06` needs a sibling assertion that does not read that exit code at all.
- **`ENG-04` is schema work, not UI work, and it gates `BRW-09`.** `schema.Meta` records no commit SHA today (`SchemaVersion`, `NodeCount`, `EdgeCount`, `LastSyncUnixMs`, `Healthy`, `HealthMessage`, `HasFileIndex`). A GitHub permalink that matches what the UI just showed must point at the *indexed* commit, which needs a new additive `Meta` field following the `HasFileIndex` precedent (absent ⇒ pre-upgrade graph, degrade gracefully). It is sequenced with the protobuf work in Phase 1 so it inherits the same additive-only discipline (D-02a) rather than being invented alongside a UI feature.
- **The SPA toolchain is stood up before any view is built (Phase 2), not after.** Research's suggested ordering put the SPA build last; that is only correct for a milestone that ships RPCs. This one ships *views*, and Phases 3–6 cannot begin without a real app to build them in. `pnpm`'s strict non-hoisted `node_modules` and its default blocking of dependency lifecycle scripts are both live ways a developer's `web/build/` can diverge from CI's — which is exactly what a *committed* build output cannot tolerate, and exactly what `BLD-03`/`BLD-05` exist to catch.
- **Browse, navigation, the workbench and index health ship before the graph view (Phases 3–4, then 5).** Eight of the Engine's ten read methods already return structured results; those four capability groups need almost no new backend and are the milestone's actual table stakes. The graph view is **must-ship** for this milestone (maintainer directive) but must not *block* them.
- **`GRF-01` is a blocking measurement spike whose pass condition is locked before dispatch, and nothing in the graph view may be planned before it resolves (Phase 5).** It measures real file/package rollup node and edge counts against indexed repositories and its result — not a preference — selects the renderer between Cytoscape.js and Sigma.js + graphology. This roadmap deliberately names neither as chosen. The precedent for a blocking spike with pre-agreed exit criteria is v0.5.0's zig-cross spike; the failure it prevents is a rendering choice defended after the fact.
- **The graph view is never force-directed, by design decision rather than by later discovery.** This is the best-documented failure mode in the surveyed literature (dependency-cruiser's FAQ, the MSR "Trimming the Hairball" paper, `aryx/codegraph` abandoning node-link entirely, SourceTrail shipping it in v1 and retreating in v2 after user studies). File/package aggregation with directory-structural grouping and a hierarchical layout is the v1 design; recovery from shipping the hairball is HIGH cost because layout choice reaches into drag, zoom and click-target code throughout.
- **The rollup is computed on demand, never precomputed (Phase 5, `ENG-03`).** A cached file/package projection would invent a second staleness mechanism this project does not need and would contradict the maintainer's own stated rationale for choosing the rollup. `Engine.FileGraph()` follows `BuildReverseAdjacency`'s fresh-per-call full-scan discipline — no new record kind, no re-indexing.
- **Live push comes last, after the base views are stable (Phase 6).** "Apply this delta to a view" is not designable before the view exists. Its integration test is non-negotiable and must run against a *real* `codegraph daemon` / `serve --mcp` holding the same store — the property it must not violate (never holding the store open) is only observable under genuine concurrent multi-process use, and "run `codegraph ui` alone" is the dev workflow that hides it.
- **`go test -run PATTERN` exits 0 when the pattern matches nothing.** No phase may treat a green exit as evidence that a test ran; where a phase claims a test executed, it reports a count.
- **Out of scope by construction, and recorded so the decisions stay visible:** in-browser editing, multi-user auth, server-persisted bookmarks, a force-directed whole-graph landing view, collaborative presence, a free-form graph query console, a churn/complexity dashboard, and hosted platform features. Each is a trap specifically *because* this UI is local, read-only, single-binary and cloud-free.
- **`v0.12.0` carries no git tag.** release-please is the sole tag authority (D-06R). The label is a prediction that holds if this milestone lands `feat:` commits. No phase schedules a `git tag` step.

- [x] **Phase 1: Engine Seam, Wire Protocol & Secure Transport** - `codegraph ui` runs as its own process, serving typed, bounded, read-only RPCs over a loopback listener that refuses a rebinding request — with every existing CLI and MCP byte unchanged (completed 2026-08-23)
- [x] **Phase 2: SPA Toolchain, Embedded App Shell & JS Supply Chain** - The browser gets a real pnpm-built Svelte app served from inside the binary, committed, drift-guarded, and covered by a JS vulnerability gate the Go tooling cannot see (completed 2026-08-24)
- [x] **Phase 3: Browse, Inspect & Navigation** - A developer finds any symbol or file, reads its verbatim source with callers, callees and blast radius, keeps clicking outward, and can hand someone a URL that lands them exactly where they were (completed 2026-08-29)
- [ ] **Phase 4: Query Workbench & Index Health** - A developer runs the four graph analyses interactively with their own knobs and can tell at a glance whether the index they are reading is worth trusting
- [ ] **Phase 5: File/Package Graph View** - A developer sees the whole repository as one readable picture at file/package granularity and drills into any file — with the renderer chosen by measurement, not assumption
- [ ] **Phase 6: Live Push** - Open views stop going quietly stale: a re-index reaches the browser over the same schema and the same client as every other call, and updates what is on screen in place

## Phase Details

### Phase 1: Engine Seam, Wire Protocol & Secure Transport

**Goal**: `codegraph ui` runs as its own process, serving typed, bounded, read-only RPCs over a loopback listener that refuses a rebinding request — backed by structured Engine results and a commit-aware graph schema, with every existing CLI and MCP byte unchanged.
**Depends on**: Nothing (first phase). **Blocking**: no RPC handler may ship before `SRV-02`'s Origin/Host control exists, and no UI view work may be planned before `ENG-01`/`ENG-02` land and are independently proven golden-clean.
**Requirements**: ENG-01, ENG-02, ENG-04, RPC-01, RPC-02, RPC-05, SRV-01, SRV-02, SRV-03, SRV-04, BLD-04, FIX-01
**Success Criteria** (what must be TRUE):

  1. A developer runs `codegraph ui`, sees a printed loopback URL, and a Connect client at that URL returns real search / callers / callees / impact / affected / files / status / node-detail / explore results from the repository's own index — including the commit SHA the index was built at — while `codegraph node` and `codegraph explore` stay byte-identical and every frozen golden covering them passes unchanged (SRV-01, RPC-01, RPC-02, ENG-01, ENG-02, ENG-04)
  2. A request carrying a foreign `Host` or `Origin` against the live listener is rejected before any handler runs — demonstrated RED against the pre-middleware build — with `localhost`, `127.0.0.1` and `[::1]` each admitted by exact match rather than treated as interchangeable; no v1 flag exposes a bind address or an auth credential, and no RPC can mutate the index (SRV-02, SRV-03)
  3. With `codegraph daemon` / `serve --mcp` already holding the store, `codegraph ui` still serves every RPC and holds no handle between calls; a re-index that outlasts `graphstore.Open`'s retry budget renders as "indexing in progress" rather than as an error; and an oversized source response comes back bounded and explicitly marked truncated rather than unbounded (SRV-04, RPC-05)
  4. Regenerating both protobuf surfaces — the new UI schema and the pre-existing `internal/schema/graph.proto` — produces no diff, the guard reports how many generated files it actually compared, and it has been watched fail against a deliberately stale checked-in file (BLD-04)
  5. An MCP session no longer loses an in-flight response when a server-initiated notification is written concurrently, proven against a reproduction that showed the loss (FIX-01)

**Notes**: `ENG-01`/`ENG-02` are extractions that stop before the existing `RenderNode`/`RenderNodeMultiDef`/`RenderExplore` call — not rewrites. `ENG-04` follows the `HasFileIndex` precedent: absent ⇒ pre-upgrade graph, degrade gracefully. `BLD-04` closes a pre-existing gap; `Taskfile.yml` has no proto regeneration task today. The two CR-01s are distinct (see the ordering notes above) — this phase's is `internal/mcp/server.go`'s `pendingWriter`, not the already-fixed Pebble lock finding. **Planning correction (2026-08-22):** research verified that nothing in the repo byte-diffs live `Engine.Node`/`Explore` output against the 26 frozen goldens today — `TestReFrozenGoldensValid` checks envelope shape only, and the byte-diff capture path was retired in FIXT-04. A new byte-identity oracle is therefore Wave-1 must-add scope (plan 01-02) and lands before the extraction, or criterion 1 is satisfiable without the output being unchanged.
**Plans**: 11/11 plans executed

Plans:
**Wave 1**

- [x] 01-01-PLAN.md — Tracer: `codegraph ui` binds, publishes its URL, then serves one origin-guarded RPC end-to-end (SRV-02, SRV-01, SRV-03, RPC-01, RPC-02)
- [x] 01-02-PLAN.md — Shared capture spec plus the golden byte-identity oracle over all 26 frozen pairs, proven non-vacuous (ENG-01, ENG-02)
- [x] 01-03-PLAN.md — FIX-01 `pendingWriter` counter, plus the `toolslist-repeat` separability disproof (FIX-01)

**Wave 2** *(blocked on Wave 1 completion)*

- [x] 01-04-PLAN.md — Engine seam: `NodeDetail` sum shape with a lazy multi-definition protocol (ENG-01)
- [x] 01-06-PLAN.md — Commit-aware `Meta` field 8 and the `(*Engine).IndexMeta` data path (ENG-04)

**Wave 3** *(blocked on Wave 2 completion)*

- [x] 01-05-PLAN.md — Engine seam: `ExploreResult`, exported component types, typed query errors, and the D-04 mutation proof (ENG-02, ENG-01)
- [x] 01-07-PLAN.md — Two-surface proto codegen drift guard, temp-tree generated and RED-proven (BLD-04)

**Wave 4** *(blocked on Wave 3 completion)*

- [x] 01-08-PLAN.md — Shared wire types plus the seven structured reads, including Status's commit SHA (RPC-01, RPC-02, ENG-04)

**Wave 5** *(blocked on Wave 4 completion)*

- [x] 01-09-PLAN.md — `GetNodeDetail` and `Explore` over the extracted seams, plus the read-only method-set guard (RPC-01, RPC-02, SRV-03)

**Wave 6** *(blocked on Wave 5 completion)*

- [x] 01-10-PLAN.md — Bounded source responses: shared rune-boundary cut, two-tier truncation, transport backstop (RPC-05)

**Wave 7** *(blocked on Wave 6 completion)*

- [x] 01-11-PLAN.md — Degraded state: typed `IndexingInProgress`, a `Status` that answers without the store (SRV-04)

### Phase 2: SPA Toolchain, Embedded App Shell & JS Supply Chain

**Goal**: The browser gets a real, pnpm-built Svelte app served from inside the binary — committed, embedded, drift-guarded, and covered by a JS vulnerability gate the Go tooling cannot see — with no Node or pnpm invocation reachable anywhere in the signed release path.
**Depends on**: Phase 1 (the SPA is served by the same mux the Connect handlers mount on, and its generated TS client comes from Phase 1's proto surface)
**Requirements**: RPC-03, BLD-01, BLD-02, BLD-03, BLD-05, BLD-06, BLD-07
**Success Criteria** (what must be TRUE):

  1. A developer with only Go on `PATH` builds the binary, runs `codegraph ui`, and gets the real app in the browser from the embedded assets — including Vite's underscore-prefixed chunks — with no JS toolchain installed anywhere on the machine (BLD-02)
  2. Opening a client-side route directly loads the app, while RPC paths and hashed asset URLs resolve to their real handlers rather than falling back to `index.html` (RPC-03)
  3. A clean checkout runs `pnpm install --frozen-lockfile` at the Corepack-pinned pnpm version and rebuilds a `web/build/` the drift guard finds matching; the guard reports how many source files it hashed and how many committed output files it manifested, and it has been watched fail against a deliberately stale committed `web/build/` and against a hand-edited byte inside it (BLD-01, BLD-03)
  4. CI fails loudly when an unapproved dependency lifecycle script appears, rather than skipping it and quietly producing a different `web/build/` than the committed one (BLD-05)
  5. `pnpm audit` runs as a named CI gate that distinguishes "scanned and clean" from "the scan itself failed" — proven by a sibling assertion that does not read `pnpm audit`'s exit code — and no `pnpm`/`node`/`npx`/`npm` invocation is reachable anywhere in `.goreleaser.yaml` or `release.yml`, checked structurally rather than by design intent (BLD-06, BLD-07)

**Notes**: `//go:embed all:build` — the committed output directory is `web/build/`, `adapter-static`'s own default (D-02/D-03 in `02-CONTEXT.md`); this text previously said `dist/`, which the phase never creates. Without the `all:` prefix, Go's default dotfile/underscore exclusion silently drops SvelteKit's `_app`-prefixed output and the build still succeeds. Criterion 1's verification is a file-list diff between the embedded FS and the on-disk tree, not "the build succeeded". Criterion 3's guard binds two things, not one: a source-tree digest AND a manifest of the committed output tree's own file list and per-file content hashes — a source-only marker passes when someone hand-edits the shipped bytes. Approval state belongs in committed config, never in per-machine interactive state.
**Plans**: 7/7 plans executed
**UI hint**: yes

Plans:

**Wave 1**

- [x] 02-01-PLAN.md — Tracer: pnpm-built SvelteKit at `web/`, committed `web/build/`, `go:embed all:build`, SPA handler on Phase 1's mux (BLD-01, BLD-02)

**Wave 2** *(blocked on Wave 1 completion)*

- [x] 02-02-PLAN.md — SPA fallback routing in full: immutable-prefix 404, client-route fallback, two-class cache policy, RPC precedence, plus `nosniff` and a same-origin CSP whose script hashes derive from the embedded `index.html` (RPC-03, BLD-02)
- [x] 02-03-PLAN.md — Scoped `protoc-gen-es` template, committed TS Connect client, `proto:drift` floor 3 → 4 (BLD-01, BLD-02)
- [x] 02-04-PLAN.md — Transitive structural release-path purity scan over `.goreleaser.yaml`, `release.yml` and the local actions and Taskfile targets they execute, positive-controlled; plus the no-JS-cache invariant on the CI install path (BLD-07, BLD-01)

**Wave 3** *(blocked on Wave 2 completion)*

- [x] 02-05-PLAN.md — App shell: Tailwind v4 + shadcn-svelte init, four named nav slots, live `GetStatus` render (BLD-02, RPC-03)

**Wave 4** *(blocked on Wave 3 completion)*

- [x] 02-06-PLAN.md — `web:deps`/`web:build`/`web:build:verify`/`web:drift`: a two-part marker binding both the source tree and the committed output bytes, RED-proven six ways, plus a scratch rebuild check and the Node 24 CI fold-in (BLD-01, BLD-03)

**Wave 5** *(blocked on Wave 4 completion)*

- [x] 02-07-PLAN.md — `strictDepBuilds` positive assertion, a lockfile-shape guard, and the `pnpm audit` gate with its exit-code-independent sibling count; closes with `task web:drift` green (BLD-05, BLD-06, BLD-03)

### Phase 3: Browse, Inspect & Navigation

**Goal**: A developer can find any symbol or file, read its verbatim source alongside its callers, callees and blast radius, keep clicking outward without losing their place, and hand someone a URL that lands them exactly where they were.
**Depends on**: Phase 2 (needs a real app shell to build views in) and Phase 1's `ENG-01`/`ENG-04`
**Requirements**: BRW-01, BRW-02, BRW-03, BRW-04, BRW-05, BRW-06, BRW-07, BRW-08, BRW-09, NAV-01, NAV-02, NAV-03, NAV-04, SRV-05
**Success Criteria** (what must be TRUE):

  1. A developer types a partial symbol or file name, sees results as they type, opens one, and reads syntax-highlighted verbatim source alongside its callers, callees and blast radius — then clicks a neighbor and keeps navigating from there (BRW-01, BRW-02, BRW-03, BRW-06)
  2. Clicking a symbol reference inside rendered source jumps to its definition; a bare name resolving to several definitions offers a picker instead of guessing; and a file path or symbol name is copyable in one action (BRW-04, BRW-05, BRW-07)
  3. A natural-language question returns `Explore`'s relevance-selected results alongside exact-name search, and the current file and line open on GitHub permalinked to the commit the index was built at, so the remote view matches what the UI just showed (BRW-08, BRW-09)
  4. Every view, symbol and query state has a shareable URL encoding view, target, depth and limit; browser back and forward walk that history correctly; and search and result selection are drivable from the keyboard, including a focus shortcut and `Esc` to dismiss (NAV-01, NAV-02, NAV-03)
  5. No index, a stale index, and a symbol that does not exist each render an explicit state naming what happened rather than an empty pane; and a request for source outside the repository root is refused by the same confinement the MCP path already uses, proven by a regression test aimed at the new endpoint (NAV-04, SRV-05)

**Notes**: `SRV-05` reuses the existing MCP path-confinement fix rather than reimplementing it — this is the same threat class this project already closed once, cheap to prevent by reuse and expensive to rediscover after ship. `BRW-06`'s highlighter registers only the indexed languages, not a full grammar bundle. `NAV-01`'s depth/limit encoding is what Phase 4's workbench deep-links against, so its shape is settled here. **Planning correction (2026-08-28):** the indexed set is **14** `LanguageSpec.ID` values, not the 12 named during discussion (`languages_typescript.go` registers three IDs from one extractor), covered by **13** highlight.js modules because that library's typescript grammar already declares `tsx` as its own alias — a Go set-equality guard binds the two. Criterion 5's "new endpoint" resolves to `GetPermalink`, the one method this phase adds, whose `path` field goes through the same confinement gate; `GetNodeDetail`'s boundary, which had zero confinement coverage in `internal/uiserver`, gets its own regression test with a passing in-repo control so neither guard can pass vacuously. Two unrelated todos ride along in their own plans (03-03 wire-oracle ordering flake, 03-10 golangci-lint — the third fold of that one) and **neither gates these five criteria**.
**Plans**: 10/10 plans executed
**UI hint**: yes

Plans:

- [x] 03-01-PLAN.md — JS test harness (vitest + jsdom + Testing Library), highlight.js install, `task web:test` and its CI step
- [x] 03-02-PLAN.md — SRV-05 confinement regression test at the `GetNodeDetail` RPC boundary, with a passing in-repo control
- [x] 03-03-PLAN.md — wire-oracle `toolslist-repeat` ordering flake (folded todo; does not gate the phase criteria)
- [x] 03-04-PLAN.md — TRACER: URL → `GetNodeDetail` → error classification → highlighted source, end to end, plus the highlighter coverage guard
- [x] 03-05-PLAN.md — BRW-09 server side: additive `GetPermalink` RPC, git remote/pushed derivation, confinement reuse
- [x] 03-06-PLAN.md — search surface: live `Search`/`Files`, `Explore` on Enter, keyboard drivability
- [x] 03-07-PLAN.md — node detail, callers/callees/blast radius, click-through, and URL push-vs-replace history
- [x] 03-08-PLAN.md — click-to-definition, disambiguation picker, copy affordance, truncation notice and permalink surface
- [x] 03-09-PLAN.md — shared status gate, the three explicit degrade states, and the rebuilt committed bundle
- [x] 03-10-PLAN.md — golangci-lint gate (folded todo, third fold; does not gate the phase criteria)

### Phase 4: Query Workbench & Index Health

**Goal**: A developer can run the four graph analyses interactively with their own knobs, read the results as real tables, and tell at a glance whether the index they are reading is worth trusting.
**Depends on**: Phase 3 (the workbench deep-links through `NAV-01`'s URL model and opens results in Phase 3's node view)
**Requirements**: WRK-01, WRK-02, WRK-03, WRK-04, HLT-01, HLT-02, HLT-03
**Success Criteria** (what must be TRUE):

  1. A developer runs `Impact` on a symbol, moves the depth control, and watches the blast radius change without leaving the page (WRK-01)
  2. A developer selects several files at once and sees what they affect, and runs `Callers`/`Callees` with an adjustable result limit (WRK-02, WRK-03)
  3. Workbench results render as structured tables the developer can sort by column, and a failed query says which kind of failure it was — not found, index stale, or server error — rather than collapsing every case into one message (WRK-04)
  4. Index freshness, coverage, per-language file counts, and node and edge counts are all readable from one health view (HLT-01)
  5. Staleness reads as a verdict about whether to trust what is shown, placed above the raw numbers rather than buried among them, and a worktree mismatch is impossible to miss (HLT-02, HLT-03)

**Notes**: Depth and limit are passed straight through to the existing Engine methods — `validateDepth`/`clampDepth`/`clampAffectedDepth`/`validateLimit`/`MaxLimit` already enforce bounds server-side for every caller, and a second copy in the UI would be a driftable duplicate. Criterion 3's error distinction exists because ConnectRPC's typed error model is easy to flatten into a single catch-all on the client, which is exactly where the difference between "not found" and "server crashed" matters to the user.
**Plans**: 5/7 plans executed
**UI hint**: yes

Plans:

- [x] 04-01-PLAN.md — tracer: table dependency intake, the Workbench URL grammar, and one end-to-end Callers slice
- [x] 04-02-PLAN.md — `Files` recursive glob via `doublestar/v4`, fixing CLI, MCP and UI at once
- [x] 04-03-PLAN.md — `GetHealth`, the eleventh read-only rpc, and its handler and mapper
- [x] 04-04-PLAN.md — four-tab Workbench shell, Impact depth control, Callees limit
- [x] 04-05-PLAN.md — the health view: trust verdict above the numbers, loud worktree warning
- [ ] 04-06-PLAN.md — multi-file selection with removable chips driving Affected
- [ ] 04-07-PLAN.md — `web:components:drift`, render-cost measurement, committed-bundle refresh

### Phase 5: File/Package Graph View

**Goal**: A developer can see the whole repository as one readable picture at file/package granularity, drill into any file for its symbols, and spot dependency cycles — with the renderer chosen by recorded measurement rather than by assumption.
**Depends on**: Phase 4 (the graph view is sequenced after the core views so it cannot block them). **Blocking within the phase**: no renderer may be chosen and no `GRF` implementation work may be planned before `GRF-01` resolves against a pass condition locked before dispatch.
**Requirements**: GRF-01, GRF-02, GRF-03, GRF-04, GRF-05, ENG-03
**Success Criteria** (what must be TRUE):

  1. A committed measurement records real file/package rollup node and edge counts against this repository's own index and at least one large external corpus, checked against a pass condition written down before the measurement ran — and that recorded result, not a preference, names the selected renderer (GRF-01)
  2. A developer opens the graph view on a real repository and can read it: files and packages grouped by directory structure, edges aggregated with per-kind counts, laid out hierarchically — never as a force-directed whole-graph view (GRF-02)
  3. A developer clicks a file in the graph and sees the symbols inside it (GRF-03)
  4. Dependency cycles are visually distinguished from ordinary edges without the developer having to hunt for them (GRF-04)
  5. The rollup is computed fresh per request from edges that already exist — no precomputed projection, no new record kind, no re-indexing — and the rendering library sits behind a component seam that a swap would not reach past (ENG-03, GRF-05)

**Notes**: `GRF-01`'s result selects between Cytoscape.js and Sigma.js + graphology; this roadmap deliberately pre-commits to neither. `ENG-03` follows `BuildReverseAdjacency`'s fresh-per-call full-scan discipline. Criterion 2 must be demonstrated against the project's own largest real corpus, not a toy repository — a graph view that only reads well on a small demo is the documented failure mode this phase exists to avoid. The exact aggregation semantics (edge counts by kind over distinct source-file/target-file/kind tuples) are pinned during planning, informed by `GRF-01`'s measurement.
**Plans**: TBD
**UI hint**: yes

Plans:

- [ ] TBD (run `/gsd-plan-phase 5`)

### Phase 6: Live Push

**Goal**: Open views stop going quietly stale — a watcher re-index reaches the browser over the same schema and the same client as every other call, and updates what is on screen in place.
**Depends on**: Phase 5 (every view that live push updates, including the graph, must exist and be stable first) and Phase 1's `FIX-01`, whose root cause this phase's streaming lifecycle is reviewed against
**Requirements**: RPC-04, LIV-01, LIV-02, LIV-03, LIV-04
**Success Criteria** (what must be TRUE):

  1. A developer edits a file, the watcher re-indexes, and the open view updates in place — with the staleness and health chrome updating by the same mechanism rather than staying stale itself while the data around it moves (LIV-01, LIV-02)
  2. Events arrive message by message as they happen rather than in a burst at the end, over plain HTTP/1.1, using the same Protobuf schema and the same Connect client as every other call — no separate SSE or WebSocket transport (RPC-04)
  3. Three or more browser tabs each keep receiving updates; a slow client gets backpressure instead of blocking the others; dropping a connection and coming back reconnects cleanly with backoff rather than a reconnect storm; and the stream's lifecycle bookkeeping is checked against Phase 1's `pendingWriter` root cause — server-initiated writes counted separately from client-initiated pending state — with the verdict recorded rather than assumed (LIV-03)
  4. Graph nodes stay where they were across a live update: the layout is updated incrementally rather than re-run from scratch, and nodes do not jump (LIV-04)
  5. With `codegraph daemon` and `serve --mcp` running against the same store, a live-push session survives repeated real re-index flushes without starving a sync or holding the store open — verified against the real processes, not a stub (LIV-01)

**Notes**: Criterion 5 is non-negotiable and is the reason this phase carries a research flag: the property it must not violate is only observable under genuine concurrent multi-process use, and running `codegraph ui` alone is the dev workflow that hides it. Criterion 2's message-by-message assertion measures per-message delivery latency, not eventual arrival — streaming that is silently buffered still passes an "it all arrived" test. Criterion 4 is verified against the graph view specifically; list and table views do not exhibit this failure.
**Plans**: TBD
**UI hint**: yes

Plans:

- [ ] TBD (run `/gsd-plan-phase 6`)

## Progress

**Execution Order:**
Phases execute in numeric order: 1 → 2 → 3 → 4 → 5 → 6. The chain is genuinely sequential — Phase 2 needs Phase 1's proto surface and mux, Phases 3–6 each need a working app shell, and Phase 6 needs every view it updates to already exist. Within Phase 1, `FIX-01` is independent of the rest and may be planned first; within Phase 5, `GRF-01` blocks everything else in the phase.

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 1. Engine Seam, Wire Protocol & Secure Transport | 11/11 | Complete    | 2026-08-23 |
| 2. SPA Toolchain, Embedded App Shell & JS Supply Chain | 7/7 | Complete    | 2026-08-24 |
| 3. Browse, Inspect & Navigation | 10/10 | Complete    | 2026-08-29 |
| 4. Query Workbench & Index Health | 5/7 | In Progress|  |
| 5. File/Package Graph View | 0/TBD | Not started | - |
| 6. Live Push | 0/TBD | Not started | - |

6 milestones shipped. v0.12.0 scoped: 6 phases, 51 requirements, 0/6 phases complete (0%). Backlog below is preserved across milestone closes.

## Backlog

### Phase 999.2: tmux e2e/UAT test harness and suite (BACKLOG)

**Goal:** [Captured for future planning] A real-PTY end-to-end test harness that drives the interactive TUI through **tmux** (send-keys + capture-pane) so the terminal actually replies to escape queries and actually scrolls — the exact conditions the current piped/non-TTY suite can never reproduce. Motivation: v1.0 Phase 7's human UAT caught two user-visible TUI bugs that BOTH the full piped automated suite AND a deep multi-agent code review missed, because they only manifest on a live TTY — G-07-1 (bare `daemon` on a TTY with an empty registry leaked the terminal's DECRQM capability-probe responses `^[[?2026;2$y^[[?2027;0$y`) and G-07-2 (both bubbletea pickers rendered inline without alt-screen → heavy flicker + blank list). bubbletea Models are unit-testable via synthetic `tea.Msg` (state transitions) but that path never renders. Scope a suite that spawns the release binary inside a tmux pane and asserts on `capture-pane` output: (a) bare `daemon` empty-registry prints ONLY `no running daemons` with no leaked escape sequences; (b) the daemon picker enters the alternate screen, renders `Running daemons` + a seeded record, and restores the main buffer on quit (no residual escapes in scrollback); (c) the install/uninstall checkbox picker renders `[x]`/`[ ]` glyphs, `space` toggles, `q`/`esc` cancels with zero config writes; (d) no flicker proxy (stable capture across N frames). Reuse the `tmux` skill's send-keys/capture-pane idioms; gate the suite behind a build tag / CI job that has tmux available (skip cleanly where it isn't). This is the missing rung between the piped never-hang/byte-identity integration tests (necessary, TTY-blind) and manual human UAT (thorough, unautomated).
**Requirements:** TBD
**Plans:** 0 plans

Plans:

- [ ] TBD (promote with /gsd-review-backlog when ready)

### Phase 999.4: CheckRegression current-metrics positivity guard (BACKLOG)

**Goal:** [Captured for future planning] Close the degenerate-input bypass in `internal/bench.CheckRegression`, surfaced and **reproduced** during the Phase 10 security audit (recorded in `10-SECURITY.md` → "Advisory — Unregistered Surface"; also code-review finding WR-06). Calling `CheckRegression(baseline, current, ceiling=1)` with `current.PeakRSSBytes = 0` and an otherwise-matching frame returns `nil` — **both** the relative RSS regression check and the absolute INDX-06 memory ceiling silently pass. The function already validates that the *baseline* metrics are positive; it never validates the *current* ones, so a zero or negative current reading reads as "no regression" instead of "unusable measurement". This is unreachable through today's only caller because `internal/bench.PeakRSSBytes` returns an error rather than a zero on failure, but `CheckRegression` is exported, its doc comment claims it "never misleads", and the phase-10 audit already showed how easily a frame-descriptor blind spot becomes a live gate failure. Scope: add a positivity/sanity check on `current` mirroring the existing baseline check, refusing rather than passing on a non-positive throughput or RSS reading, with an error naming which field was degenerate. This belongs to the repo's documented class of **gates that cannot fire** (the retracted 10.6% perf claim, the inverted `rg -qv` gate, the 51.5%-stale baseline) — so the fix must be demonstrated RED with a degenerate-input test, not merely added.
**Requirements:** TBD
**Plans:** 0 plans

Plans:

- [ ] TBD (promote with /gsd-review-backlog when ready)
