# Roadmap: CodeGraph Go

## Overview

CodeGraph Go is a Go implementation of a pre-indexed code knowledge graph for coding agents. Eight milestones have shipped: **v0.1** (2026-07-14) landed the core capabilities — indexing, query, MCP server, sync — from a signed/attested/SBOM'd release. **v1.0** (2026-08-03) closed the behavioral and surface gaps, adding a human-facing Charm TUI behind a build-enforced rendering seam that keeps the agent/MCP path free of ANSI, plus fully automated signed releases via release-please + GoReleaser. **v0.3.0** (2026-08-06) brought the stdio MCP server current with spec revision `2026-07-28` on `modelcontextprotocol/go-sdk@v1.7.0`, proven by a wire-level oracle that never imports the SDK it tests. **v0.5.0** (2026-08-11) made the binary installable by convention on macOS — Gatekeeper-accepted on both darwin arches and `brew install`-able from a tap we control. **v0.10.0** (2026-08-13) made agents actually *use* the tools: the server documents itself over MCP Resources, a decision-procedure-first SKILL.md teaches which question goes to which tool, and a SessionStart nudge makes availability visible at the moment it matters. **v0.11.0** (2026-08-16) retired the comparison framing without retiring capability, re-basing the golden suite onto measurement-selected corpora and retiring Compatibility as a constraint.

**v0.12.0 (Local Graph UI) shipped 2026-09-07.** Every consumer of the graph until now was a program — a CLI invocation or an agent over MCP. This milestone gave the graph a human face: `codegraph ui` serves a local, read-only web UI from the binary itself, so a developer can browse, visualize, query and health-check a `.codegraph/` index without going through an agent. It is deliberately a **third consumer** of `internal/query.Engine`, never a second implementation — the cross-phase integration check found no duplicated traversal logic in `internal/uiserver`, with every handler routing through one `withEngine` seam. The wire is ConnectRPC over Protobuf (maintainer directive), the app is a `pnpm`-built Svelte SPA committed and `go:embed`'d so the signed release path stays pure Go, and the whole surface binds loopback with exact-match Origin/Host validation because loopback binding alone has already been walked through by a documented CVE class. 51/51 requirements, 6/6 phases verified *and* validated, six per-phase SECURITY.md files all at `threats_open: 0`.

**v0.13.0 (Guard Hardening & UI Follow-through) shipped 2026-09-13.** A deferral burn-down in two coherent sets. First, every known guard that cannot fire — this repo's own recurring defect shape is an assertion that is true but non-discriminating, so it passes whether the property holds or not (rule `84d1gfpywd`) — closed with a recorded RED demonstration each, plus the real-PTY testing gap: a tmux harness that drives the release binary in a genuine pane and gates CI on an exact executed count. Second, the four contained UI follow-ons v0.12.0 deliberately left — scroll breadcrumb and editor handoff, the coverage denominator that answers "why is my file missing", and community clustering on the graph view — all on the unchanged 16-rpc read-only wire with every proto change additive. The documentation tail became a `cobra/doc`-generated `docs/CLI-REFERENCE.md` under a drift gate plus an allowlist guard for the generator's one blind spot, after the maintainer reset that phase's scope at discuss time. 26/26 requirements, 6/6 phases verified (re-verified to canonical `passed` at close), 21 mutation families, six SECURITY.md files at `threats_open: 0`, a `verified_closeout`. Promoted and delivered backlog **999.2** and **999.4**.

**v0.14.0 (Polish & Agent Reach) is in progress (scoped 2026-09-14).** Three milestones of feature work left a ledger of small, known, individually-cheap defects that no phase owned — a stock Svelte favicon that also violates CSP, a picker footer that never renders with all eight targets, a drift gate whose two halves enumerate different sets, a watchdog test that flakes only under full-suite load — beside a CLI whose styling seam has three monochrome styles used by three of 24 verbs and two verbs (`query`, `search`) that print byte-identical human lines, and an agent-reach backlog (AGENT-04…07, GUARD-HOOK-01/02) deferred twice for want of evidence that was never collected, while Codex — a harness the maintainer actually uses — stayed a global-only MCP+`AGENTS.md` target with no skill and no nudge. This milestone burns the ledger down in two early phases, folds the verb surface before styling it (both touch `search.go`/`daemon.go`), gives every human-output verb one semantic palette while holding the agent/MCP, `--json` and piped paths byte-identical, builds a per-target capability table before the first harness consumes it, reframes the guard hook as a nudge that adds context and never denies, and brings Codex to parity last — each mechanism verified in a live session before it is written. 55 requirements across 7 phases. **Phase numbering restarts at 1** (`--reset-phase-numbers`): v0.13.0's Phases 7–12 are archived under `milestones/v0.13.0-phases/`, so `01-*` directories collide with nothing.

**Versioning note:** "v1.0" is a *planning-milestone* name, never a release version. The shipped artifact line reached `v0.2.0` at v1.0's close and has since advanced through `v0.3.0`, `v0.4.0`, `v0.5.0` … `v0.9.0`, plus `v0.10.0` and `v0.11.0`, each computed by release-please from Conventional Commits; there is deliberately no `v1.0.0` tag (maintainer directive D-06R, 2026-07-29). Milestone labels track the release line but carry **no git tag**: release-please remains the sole tag authority, pinned by `TestGsdTagCreationIsDisabled`. A hand-created `v*` tag would additionally match `release.yml`'s `push: tags: "v[0-9]*"` trigger and falsely fire the release pipeline. The milestone record lives in `MILESTONES.md` + `milestones/`. (`milestone-v0.1` exists only because it predates release-please.)

## Milestones

- ✅ **v0.1 — Initial Release** — Phases 1–8 (shipped 2026-07-14) — core capabilities + signed release
- ✅ **v1.0 — Drop-in Parity & Human UX** — Phases 1–10 (shipped 2026-08-03) — behavioral + surface parity, human TUI, automated signed releases, local build tooling
- ✅ **v0.3.0 — MCP Protocol Currency** — Phases 1–5 (shipped 2026-08-06) — official Go SDK adoption, `2026-07-28` spec compliance without breaking Legacy clients, a wire-level verification oracle, tool-modfile vulnerability coverage
- ✅ **v0.5.0 — macOS Distribution & Homebrew** — Phases 1–4 (shipped 2026-08-11) — `goreleaser release` migration with zig cross-compilation, Apple notarization, a Homebrew tap and cask, and an `upgrade` that steps aside under brew. Promoted backlog 999.5, consumed SEED-002
- ✅ **v0.10.0 — Agent Onboarding Skill & MCP Resources** — Phases 5–8 (shipped 2026-08-13) — the server documents itself over MCP Resources, a decision-procedure-first SKILL.md plus a SessionStart nudge teach agents when to reach for it, `codegraph install` ships that package with the binary, and the stale `instructions` promise was retired last. Consumed the 2026-08-08 skill todo
- ✅ **v0.11.0 — Standalone Project Identity** — Phases 1–6 (shipped 2026-08-16) — origin acknowledged once in `NOTICE` plus one README License clause; comparison framing removed tree-wide to a proven zero; golden corpora re-selected by measurement, re-frozen from codegraph-go's own output, and re-proven non-vacuous; benchmarks published as mechanically-generated absolute numbers; `codegraph migrate` and the `modernc.org/sqlite` dependency removed (D-04); Compatibility retired as a constraint
- ✅ **v0.12.0 — Local Graph UI** — Phases 1–6 (shipped 2026-09-07) — `codegraph ui` serves a read-only, loopback-bound, ConnectRPC-backed Svelte SPA embedded in the binary: browse and inspect symbols with verbatim source and blast radius, a deep-linkable navigation model, an interactive query workbench, a visual index-health verdict, a file/package graph view, and live push from the watcher over a server-streaming rpc on the same schema and client as every other call. Consumed SEED-001; folded in the CR-01 `pendingWriter` fix
- ✅ **v0.13.0 — Guard Hardening & UI Follow-through** — Phases 7–12 (shipped 2026-09-13) — every known guard that cannot fire closed with a recorded RED demonstration (21 mutation families across six phases); a tmux real-PTY harness gating CI on an exact executed count; v0.12.0's four UI follow-ons (breadcrumb + editor handoff, coverage denominator, community clustering) on the unchanged 16-rpc read-only wire; `docs/CLI-REFERENCE.md` generated from the live Cobra tree under a drift gate plus an allowlist guard for hidden flags; brew-trust wording narrowed. Verified closeout after re-verifying all six phases at HEAD. Archived to `milestones/v0.13.0-*`
- 🚧 **v0.14.0 — Polish & Agent Reach** — Phases 1–7 (in progress) — every known defect, vacuous guard, flake and doc drift burned down at its cause; `query` → `search --full` and `unlock` → `daemon unlock` with loud one-release stubs; one semantic colour palette across every human-output verb behind `--color`/`NO_COLOR` with the agent/MCP path byte-identical; a per-target capability table, the skill in every harness that can read one, a Claude Code PreToolUse nudge that adds context and never denies, and Codex at parity — each verified in a live session. Promotes AGENT-04/06/07 and GRD-07/08; supersedes AGENT-05 and GUARD-HOOK-01/02. Phase numbering restarts at 1 (`--reset-phase-numbers`)
- 📋 **Later** — unscoped. Still parked: GRF-07 (opt-in whole-symbol graph — on the v0.12.0 Phase 5 evidence that a 3,233-node file view never converged; revisit only with a measured budget), Team Scale (central server, CI-distributed indexes), SEED-003 (markdown in the index), DIST-06 (stapled offline-safe container), BREW-07 (homebrew-core), MRTR-01 (elicitation), GH #23 (gsd-pi install target), GH #9 (open-gsd workflow chore), the v0.14.0 v2 deferrals (NUDGE-07…10 nudge hooks for Cursor, Kiro, Gemini CLI, opencode and Antigravity — deferred by maintainer decision 2026-09-14; AGENT-12 Hermes — no public documentation found, nothing claimed until verified by non-web means), BRW-14 (nested scope-stack breadcrumb), annotations (embeddings/export). VOCAB-01 was *declined* at v0.11.0, not deferred

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

<details>
<summary>✅ v0.13.0 Guard Hardening & UI Follow-through (Phases 7–12) — SHIPPED 2026-09-13</summary>

- [x] Phase 7: Guards That Cannot Fire (4/4 plans) — GRD-01…04, GRD-06 (GRD-05 declined by deletion) — completed 2026-09-08
- [x] Phase 8: tmux Real-PTY Harness (5/5 plans, 08-05 added as the G-08-1 gap closure) — TTY-01…07 — completed 2026-09-11
- [x] Phase 9: Source View Follow-Through — Breadcrumb & Editor Handoff (6/6 plans, 09-06 added as the svelte-check gap closure) — BRW-10…13 — completed 2026-09-12
- [x] Phase 10: Index Health — The Coverage Denominator (6/6 plans) — HLT-04…06 — completed 2026-09-13
- [x] Phase 11: Graph View — Community Clustering (5/5 plans) — GRF-06, GRF-08…10 — completed 2026-09-13
- [x] Phase 12: CLI Reference & Docs Tail (3/3 plans; scope reset at discuss — `cobra/doc`-generated reference + drift gate, allowlist guard, no DOCS-07 test) — DOCS-05…07 — completed 2026-09-13

**What the milestone proved, beyond the features:** a guard that cannot fire is found only by making it fail — every phase committed a mutation log of RED transcripts against applied, byte-cleanly-reverted mutations, and the discipline itself found two more vacuous guards (CR-01 in Phase 7's review; the graphstore archtest's partial-load hole, fixed at the close as quick task `260913-pkp`). GRF-09 repeated GRF-01's threshold-before-measurement protocol with a persisted ancestry test (threshold `698235a2`, measured 106 ms vs 500 ms). The milestone closed as a `verified_closeout`: Phases 7–11 were re-verified at HEAD so every canonical status reads `passed` (the #4155 `REQUIREMENTS.md`-in-`covered_files` trap is the lesson), and the two pre-close open artefacts were resolved rather than acknowledged. Phases 7–8 merged mid-milestone as PR #69; phases 9–12 were unpushed at close. Carried out: CI wiring for `check:gonum` / `check:no-force-layout`, the `tmux e2e` required-check setting, WINDOWS #35/#36, four pre-existing `/graph` console entries.

Archived: [`milestones/v0.13.0-ROADMAP.md`](./milestones/v0.13.0-ROADMAP.md) · requirements: [`milestones/v0.13.0-REQUIREMENTS.md`](./milestones/v0.13.0-REQUIREMENTS.md) · audit: [`milestones/v0.13.0-MILESTONE-AUDIT.md`](./milestones/v0.13.0-MILESTONE-AUDIT.md) · phases: `milestones/v0.13.0-phases/`

</details>

### 🚧 v0.14.0 — Polish & Agent Reach (In Progress)

**Milestone Goal:** Burn down every known defect, vacuous guard, flake and doc drift; give the CLI a colour-and-structure glow-up on a streamlined verb surface; and make the codegraph skill, instructions and nudge present and discoverable in every supported agent harness, with Codex brought to parity with Claude Code.

**Phase numbering restarts at 1** (`--reset-phase-numbers`). v0.13.0 ran Phases 7–12 and its phase directories are archived under `milestones/v0.13.0-phases/`; `.planning/phases/` holds only the two `999.x` backlog directories (kept, annotated, by the 2026-09-08 bookkeeping decision), so `01-*` through `07-*` collide with nothing.

**Ordering is load-bearing, not stylistic:**

- **The burn-down lands first, in two phases split by what each item is (Phases 1–2), not scattered across the features.** Every item is orthogonal to the three medium features — `web/`, `internal/daemon`, `internal/bench`, `.github/workflows/`, `docs/`, `.planning/` — so it is the cheapest work in the milestone and the work most likely to expose a wrong assumption about this repo's own gates before any feature is committed to. Phase 1 is the `FIX-` set: code defects and flakes, each fixed at its cause (the asset, not the CSP; the time source, not the timeout; the seam, not a sleep). Phase 2 is the `GRD-`/`DOCS-` set: guards proven RED against their incident shape, CI wiring for checks that today run only locally, and doc claims brought back to what ships. Two exceptions are deliberate: **`FIX-03` (picker footer) is held to Phase 7** because its final verification must run after the milestone's last change to the install target count, which is `CODEX-02`; **`GRD-13` (the `present` archtest denylist) rides with Phase 4** because it must ship in the same commit as the first new charm-family import, and that import is the glow-up's.
- **The verb fold (Phase 3) lands before the glow-up (Phase 4) because both touch `search.go` and `daemon.go`.** A `present.RenderSearch` written against the pre-fold two-verb shape is deleted and rewritten one phase later when `--full` lands; writing it once against the final surface is the whole reason for the order. `VERB-05`'s census runs *before* `internal/cli/` is edited and again after, so the fold cannot leave a stale `codegraph query` in a doc, a hook script or a test.
- **The fang verdict (`CLI-08`) is Phase 4's first plan and gates every renderer.** Adopting fang after a hand-rolled grouped-help template throws that work away; hand-rolling before the verdict is the out-of-scope entry it names. The spike's bar is written down before it runs: `WithoutManpage()`/`WithoutCompletions()` compose with the existing hidden `man` and cobra completions, `serve --mcp`'s transcript is byte-identical under the wrapper, and the SBOM/govulncheck delta is acceptable. This is the one place the CLI/MCP boundary genuinely widens — the TUI-01 archtest excludes `internal/cli` from its closure by design, so the proof is a transcript re-capture, not an amendment to the archtest.
- **The capability model (`AGENT-08`, Phase 5) lands before Codex parity (Phase 7) because Codex is its first real consumer.** Building `Capabilities()` and wiring `install`/`uninstall`/`Detect`/`--print-config-style` to one table *before* writing Codex's skill-dir, `AGENTS.md` and local-scope code is the "seam before second consumer" discipline v0.12.0 Phase 1 established — otherwise Codex becomes the ninth bespoke re-implementation. The shared `.agents/skills/` path is written once and each harness is verified live to read it; a harness-specific directory is written only where a live session shows the shared path is not read.
- **The Claude Code nudge (Phase 6) lands before Codex parity because `CODEX-05` carries `NUDGE-03/04`'s contract verbatim.** The hook emits only `additionalContext`, exits 0 unconditionally, never emits `permissionDecision`, and fires at most once per session — the 2026-09-14 reframe from "redirect" to "add context, never deny". That contract is proven once, against the current hooks reference and a fresh live session, and then reused; it is not derived twice.
- **`CODEX-01`'s live verification is Phase 7's first task, before any `codex.go` change.** `codex.go`'s "no per-project config" comment encodes a 2026-era assumption research found stale; whether Codex parity is "add a new write path" or "confirm global-only" is decided by a scratch trusted project with dated citations, not by memory. Within Phase 7, the `FIX-03` fix lands *before* `CODEX-02` flips Codex to local scope (so the newly selectable row cannot make the overflow worse mid-phase) and its tmux assertion is re-run *after* it.
- **Two invariants bind every phase.** The agent/MCP output path stays byte-identical and ANSI-free — golden oracle, wire oracle and the TUI-01 archtest unchanged, the 8-tool MCP set frozen — so `--json` and non-TTY output are gated before any styled branch and the fold has zero MCP wire surface. And every new per-harness write uses exact-identity ownership (commit `242ec0a`): a planted-foreign-entry test per harness proves an unrelated sibling entry is never overwritten, and `uninstall` leaves unrelated content byte-identical.
- **A guard is not trusted until demonstrated RED** (rule `84d1gfpywd`). `web:drift` is replayed against commit `98cd41dd`'s exact incident; the `present` archtest against a planted charm import; the re-frozen goldens against a reintroduced old verb; the store-lock fix against a held lock across `index --force`. Each phase commits its mutation log rather than asserting it.
- **"Verified in a genuinely fresh live session" is the evidence standard** (v0.10.0's), and it is hardest for the CLI-only harnesses. Each harness verification is a real transcript, not a summary, with the negative-space check recorded — did the agent list the skill, and did it reach for codegraph rather than grep. Where live verification is impossible the capability table says `[ASSUMED]` rather than claiming.
- **The two feature tracks are independent.** Phases 3→4 (`internal/cli`) and 5→6→7 (`internal/agents`, hook assets) share no files; they are numbered sequentially but could run as parallel workstreams once each track's own prerequisite lands.
- **`v0.14.0` carries no git tag** (D-06R). The `feat!:` verb rename cuts a *minor* under `release-please-config.json`'s `bump-minor-pre-major`, so the label holds; the stubs' removal in the following minor is recorded in the release notes and `docs/CLI-REFERENCE.md`, never scheduled as a tag.

- [x] **Phase 1: Defect & Flake Burn-down** - Every known user-facing defect, store-lock hole, race and load-sensitive flake is fixed at its cause or closed with the measurement that justifies closing (completed 2026-09-15)
- [ ] **Phase 2: Guards, CI Wiring & Docs Burn-down** - Every guard that could pass vacuously fails against its incident shape, CI runs the checks that exist only locally, and every doc claim about dependencies, provenance, scanners and planning state matches what ships
- [ ] **Phase 3: Verb Fold** - `query` folds into `search --full` and `unlock` into `daemon unlock`, every consumer of the old names found, the removed verbs failing loudly for one release
- [ ] **Phase 4: CLI Glow-up** - Every human-output verb renders through one shared semantic palette, adaptive to the terminal and the user's colour preferences, with the agent/MCP, `--json` and piped paths byte-identical
- [ ] **Phase 5: Agent Reach — Capability Model & Skill in Every Harness** - One capability table drives all eight targets, and every harness with a skill mechanism receives the codegraph skill, verified live, with every write exact-identity-owned and reversible
- [ ] **Phase 6: Claude Code PreToolUse Nudge** - In an indexed repo, Claude Code is pointed at `codegraph_explore` the first time it reaches for grep/find/Read — as added context, never a denial — and stays silent everywhere else
- [ ] **Phase 7: Codex Parity** - Codex receives everything Claude Code does — project-local MCP config, skill, repo-root instructions, opt-in nudge — each mechanism verified live before it is written, and a fresh Codex session reaches for codegraph unprompted

#### Phase 1: Defect & Flake Burn-down

**Goal**: Every known user-facing defect, store-lock hole, race and load-sensitive flake in the ledger is fixed at its cause — never masked by a wider policy, a wider timeout or a sleep — or closed with the measurement that justifies closing it.
**Depends on**: Nothing (first phase of the milestone). Independent of every feature phase; touches `web/`, `internal/cli/index.go`, `internal/daemon`, `internal/bench` and two `pull_request_target` workflows only.
**Requirements**: FIX-02, FIX-04, FIX-05, FIX-06, FIX-07, FIX-08, FIX-09, FIX-10, FIX-11
**Success Criteria** (what must be TRUE):

  1. Every UI route serves a codegraph favicon that loads under the unchanged `default-src 'self'` CSP — the browser console shows no CSP-blocked favicon entry, the asset is a static file, and `spa_test.go`'s CSP assertion is untouched (FIX-02)
  2. `/graph` loads with zero uncaught page errors on this repo's index and on guava, and the cytoscape "invalid endpoints" warnings at guava scale are root-caused and either fixed or waived with the cause on record (FIX-04, FIX-05)
  3. `codegraph index --force` against a store another process holds locked refuses (or warns) before `RemoveAll` instead of reading the lock as "never indexed" — proven by a regression test that holds the lock across the call and was watched fail against the pre-fix build (FIX-06)
  4. `go test -race ./internal/daemon/...` is clean and `TestRunWatchdogCancelsRunOnSimulatedReparent` passes deterministically under full-suite parallel load, with the fix in time-source injection or isolation and the getppid seam per-instance or synchronized — no timeout widened (FIX-07, FIX-08)
  5. `CheckRegression` refuses a baseline/current corpus-identity mismatch without going red on a legitimate corpus rename; neither `pull_request_target` workflow expands fork-controlled paths through a fixed heredoc delimiter, exercised against a path containing the old delimiter; and GH #20's two perf-gate follow-ups each end in a recorded fixed-or-closed decision (FIX-09, FIX-10, FIX-11)

**Notes**: Research pitfalls attached: 14 (watchdog flake — WINDOWS #12 already diagnosed load-sensitivity, so a wider constant is the failure mode; the fix touches the time source or test isolation and the comment cites #12), 15 (`var getppid = os.Getppid` is a package-level mutable global — the race is inherent to the pattern, not incidental), 16 (`Metrics.Repo` — investigate what the write sites actually populate before choosing the key, or a path-representation difference reads as a mismatch), 17 (GH #15 — the vulnerability class is multi-line-output injection; the delimiter is per-run generated, not a longer literal, and the fix lands in *both* `require-issue-link.yml` and `pr-template-format.yml`), 18 (the favicon fix touches the asset, never `spa.go`'s CSP string — `img-src data:` is the out-of-scope entry). FIX-06 is WINDOWS #36 / T-10-16, accepted below the `high` gate at v0.13.0 Phase 10 with the fix named; the regression test is the lock-collision shape it describes. FIX-04/05 are the four pre-existing `/graph` console entries v0.13.0 Phase 11 reproduced on a pre-phase build and left unowned; a live Chromium pass is the gate, per v0.12.0's lesson. `FIX-03` is deliberately *not* here — see Phase 7.
**Plans**: 9/9 plans executed
**UI hint**: yes

Plans:
**Wave 1**

- [x] 01-01-PLAN.md — Live-Chromium `/graph` console gate (tracer): `graph-console-check.mjs`, Taskfile target, pre-fix RED verdict and the guava overlap diagnosis (FIX-04, FIX-05)
- [x] 01-02-PLAN.md — Per-instance watchdog parent-pid seam and injected tick source, with an AST shape guard and GH #13's second half settled (FIX-07, FIX-08)
- [x] 01-03-PLAN.md — `CheckRegression` corpus-identity guard on `Metrics.Repo`, strict equality, empty means unrecorded (FIX-09)
- [x] 01-04-PLAN.md — `index --force` refuses a held store before `RemoveAll`, warns on a corrupt one, with the hold-the-lock regression test (FIX-06)
- [x] 01-05-PLAN.md — GH #20's two perf-gate follow-ups each end in a recorded decision; the issue is closed (FIX-11)

**Wave 2** *(blocked on Wave 1 completion)*

- [x] 01-06-PLAN.md — codegraph mark shipped as static `web/static/` icons under the unchanged CSP; Svelte logo deleted (FIX-02)
- [x] 01-07-PLAN.md — Per-run `$GITHUB_OUTPUT` delimiter in both `pull_request_target` workflows, with a harness over the shipped shell (FIX-10)

**Wave 3** *(blocked on Wave 2 completion)*

- [x] 01-08-PLAN.md — Stylesheet alignment fix and the deferred-`cy.destroy()` teardown guard, after a live reproduction (FIX-04, FIX-05)

**Wave 4** *(blocked on Wave 3 completion)*

- [x] 01-09-PLAN.md — ELK option change separating the coincident collapsed pair, plus the committed green two-corpus verdict (FIX-05, FIX-04)

#### Phase 2: Guards, CI Wiring & Docs Burn-down

**Goal**: Every guard in the ledger that could pass vacuously now fails against its recorded incident shape, CI runs on every pull request the checks that today exist only as local Taskfile targets, and every documentation and planning claim — dependency counts, provenance scope, scanner coverage, window status, todo table, seed status — states what is true today.
**Depends on**: Nothing (independent of Phase 1 and of every feature phase; touches `Taskfile.yml`, `ci.yml`, `release.yml`, `internal/upgrade` shape tests, `docs/`, `SECURITY.md`, `WINDOWS.md`, `STATE.md` and `seeds/` only). `GRD-12`'s `tmux e2e` context is added when the maintainer's repository-settings action lands; the live-ruleset comparison itself does not wait for it.
**Requirements**: GRD-09, GRD-10, GRD-11, GRD-12, GRD-14, DOCS-08, DOCS-09, DOCS-10, DOCS-11
**Success Criteria** (what must be TRUE):

  1. `web:drift` is demonstrated RED against commit `98cd41dd`'s exact incident shape — build output present on disk but unstaged — on a clean checkout, closing WINDOWS #29 with the recorded cause; the local `find` enumeration of `web/build` is retained by design (GRD-09)
  2. The vendored `button.svelte` drift is isolated by a `workflow_dispatch` run under Corepack-pinned pnpm 11.23.0 and the component is re-vendored or the window waived with the cause recorded (GRD-10)
  3. A pull request that breaks `check:gonum` or `check:no-force-layout` fails `ci.yml`, and a test compares `requiredCheckNames` against the live `protect-main` ruleset and fails when they diverge (GRD-11, GRD-12)
  4. `docs/RELEASE.md`'s dependency paragraph states counts derived from `go.mod` and credits `modelcontextprotocol/go-sdk`; SLSA provenance is described as attested over the binaries, not the checksums file, in `release.yml` and both docs; root `SECURITY.md` states govulncheck's and `pnpm audit`'s disjoint scope with the advisory caveat (DOCS-08, DOCS-09, DOCS-10)
  5. WINDOWS #16 and #33 read fixed via `gsd-tools windows fixed` with the verification recorded and #20/#21/#34 annotated record-only; STATE.md's Pending Todos table matches `.planning/todos/`, and SEED-001's frontmatter records its consumption by v0.12.0 — every write through a tool-sanctioned writer where one exists, the SEED-002 precedent where none does (GRD-14, DOCS-11)

**Notes**: Research pitfall 13 is the whole of `GRD-09`: the paired-assertion form pitfall 13 proposed (unifying both halves onto git-tree enumeration, or asserting `find` equals `git ls-files`) was declined at discuss time (D-01) once CI's clean-checkout `find` was established to already fail on the incident — the mutation log replays it instead. `GRD-12` promotes GRD-07 and `DOCS-10` promotes GRD-08, both *declined* at v0.13.0 rather than forgotten; `GRD-14` closes the stale-open windows the v0.13.0 audit listed and leaves #20/#21/#34 open as record-only deviations, in the same pass. `DOCS-11` is bounded by the planning-artifacts rule: `STATE.md`'s todo table and `ROADMAP.md` are tool-owned, so values are filled in existing shapes and no heading is invented; the `pendingWriter`/CR-01 row and three others predate v0.13.0 and were never reconciled when their files moved. `DOCS-09` is GH #14; the two docs carrying the claim are located by census, not assumed.
**Plans**: 7/7 plans executed

Plans:
**Wave 1**

- [x] 02-01-PLAN.md — `web:drift` replayed RED against a clean worktree of `98cd41dd` (tracer) with a green control on HEAD, recorded as `02-MUTATION-LOG.md` Family (a); GRD-09 requirement and criterion 1 reworded to the proof statement (GRD-09)
- [x] 02-03-PLAN.md — Shared `.github/required-status-checks.txt`, the hard-failing CI-only ruleset-drift script/step wired RED against today's 6-vs-7 divergence (tracer), and the TDD data-file loader replacing the `requiredCheckNames` literal (GRD-12)
- [x] 02-05-PLAN.md — Fresh drift measurement under pinned pnpm 11.23.0 (tracer), then all eight shadcn-svelte families re-vendored to one registry snapshot with every D-12 gate green and `web/build/**` in the same commit (GRD-10)
- [x] 02-06-PLAN.md — Positive-controlled provenance census closing GH #14 (tracer); count-free `docs/RELEASE.md` dependency paragraph crediting `modelcontextprotocol/go-sdk`; one `tool-vuln` sentence in `SECURITY.md` (DOCS-08, DOCS-09, DOCS-10)

**Wave 2** *(blocked on Wave 1 completion)*

- [x] 02-02-PLAN.md — `Install syft` + `check:gonum` + `check:no-force-layout` steps in `ci.yml`'s `test` job (tracer), proven able to fail by a planted layout literal recorded as Family (b) (GRD-11)
- [x] 02-07-PLAN.md — WINDOWS #13/#16/#29/#31/#33 closed through `gsd-tools windows fixed` with recorded evidence; #20/#21/#34 record-only; STATE.md Pending Todos regenerated from `init todos` with the bench `pinnedAt` todo filed; SEED-001 consumption fields; tool gaps reported (GRD-14, DOCS-11)

**Wave 3** *(blocked on Wave 2 completion; halts `blocking-human` for the D-08 ruleset change)*

- [x] 02-04-PLAN.md — Maintainer's one-glance ruleset update package (tracer, RED re-captured), then — after the live `protect-main` set reads 8 — the fixture grows to 8, the step goes GREEN as Family (c), and the Phase 08-03 blocker is resolved by verb (GRD-12)

#### Phase 3: Verb Fold

**Goal**: The CLI verb surface is streamlined — `query` folds into `search --full` on the same ranking function and `unlock` becomes `daemon unlock` — with every consumer of the old names found before and after the edit, the removed verbs exiting non-zero with a rename message for one release, and the generated reference, completions, man pages and goldens re-frozen to the new surface in a reviewed diff.
**Depends on**: Nothing in code. Sequenced before Phase 4 because both touch `search.go` and `daemon.go`, so the styled renderers are written once against the final flag shape.
**Requirements**: VERB-01, VERB-02, VERB-03, VERB-04, VERB-05, VERB-06, VERB-07, VERB-08
**Success Criteria** (what must be TRUE):

  1. `codegraph search --full <term>` prints each hit's signature and qualified name in the human branch and emits the `MarshalQueryJSON` envelope under `--json`, while default `search` output and `search --json` are byte-identical to before the fold (VERB-01)
  2. `search` accepts the removed `query`'s flag superset — `-j`, `-l`, `-k`, `-p` short forms — with a flag-parse test covering both long and short forms on `search` and `search --full` (VERB-02)
  3. `codegraph query …` and `codegraph unlock …` print `"query" has been renamed to "search --full"` (respectively `daemon unlock`) to stderr, exit non-zero and execute nothing, as hidden dedicated stubs that do not use cobra's `Deprecated`; `codegraph daemon unlock` has `unlock`'s identical flags and behaviour (VERB-03, VERB-04)
  4. A positive-controlled, word-boundary, multiline census run before `internal/cli/` is edited and again after finds zero references to `codegraph query` / `codegraph unlock` outside the stubs across README, docs, SKILL.md, `internal/mcp/resources/*.md`, hook scripts, Taskfile, CI and tests; `docs/CLI-REFERENCE.md` is regenerated with both stubs allowlisted by reason, `task docs:cli:drift` and `TestEveryRegisteredFlagIsAccountedFor` are green, and shell completions and man pages reflect the new surface (VERB-05, VERB-06)
  5. Every golden the fold touches is re-frozen in a reviewed diff with a RED demonstration against a reintroduced old verb name; the 8-tool MCP set and the wire oracle's transcripts are unchanged; the rename lands as a `feat!:` conventional commit with the stubs' removal in the following minor recorded in the release notes and `docs/CLI-REFERENCE.md` (VERB-07, VERB-08)

**Notes**: Research pitfalls attached: 4 (the merge loses flags or changes shape — keep the superset, gate the JSON envelope behind `--full`; `search.go`'s `json.Marshal(locs)` and `query.go`'s `MarshalQueryJSON` are *not* the same shape today and both must survive), 5 (the census is whole-repo and word-boundary, excluding the unrelated `internal/query` package identifier — the positive control is a planted old-verb string that the census must find), 6 (`feat!:` under `bump-minor-pre-major` cuts a minor, not a major; the CHANGELOG's BREAKING CHANGES section is inspected post-merge), 7 (the re-freeze is a deliberate reviewed diff, never a blanket regenerate). The stubs need one command-level allowlist line each in `testdata/cli-reference-allowlist.txt` because every hidden command still registers a `help` flag. `daemon unlock` is `unlock.go`'s body moved verbatim — the same move `daemon start` already made once. Neither verb ever had an MCP tool, so the fold has zero wire surface; SKILL.md and the MCP resources name neither today, and the census proves it rather than assuming it.
**Plans**: TBD

Plans:

- [ ] TBD

#### Phase 4: CLI Glow-up

**Goal**: Every human-output verb renders through a `present` renderer using one shared semantic palette — adaptive to light and dark backgrounds, downsampled to what the terminal can show, overridable by `--color` and the standard environment variables — with grouped, consistently styled help and consistent short flags, while the agent/MCP path, `--json` and piped output stay byte-identical and the archtest that keeps charm out of the serve-reachable closure catches every import path added this milestone.
**Depends on**: Phase 3 (the renderers for `search` and `daemon` are written against the folded surface).
**Requirements**: CLI-01, CLI-02, CLI-03, CLI-04, CLI-05, CLI-06, CLI-07, CLI-08, GRD-13
**Success Criteria** (what must be TRUE):

  1. A recorded fang verdict precedes every renderer: `charm.land/fang/v2` is adopted only if `WithoutManpage()`/`WithoutCompletions()` compose with the existing hidden `man` and cobra completions, `serve --mcp`'s transcript is byte-identical under the wrapper, and the SBOM/govulncheck delta is acceptable — otherwise the help template is hand-rolled, and either way no renderer or help template lands before the decision (CLI-08)
  2. Every human-output verb — `explore`, `search`, `node`, `callers`, `callees`, `impact`, `affected`, `files`, `status`, `init`, `index`, `sync`, `daemon`, `install`, `uninstall`, `upgrade`, `serve` (non-MCP), `githooks`, `uninit`, `version`, `ui` — renders through `present` with header, label, value, path, count, warning and error as distinct hues, consuming the existing plain-struct seams with zero changes to `internal/query` or `internal/mcp`; `codegraph --help` groups commands into Query the graph / Build the index / Agents & serving / Maintenance with `help` and `completion` filed, `<verb> --help` matches root help's styling, and `-j`, `-l`, `-k`, `-p` exist on every query verb whose long form exists (CLI-01, CLI-06, CLI-07)
  3. `TERM=dumb`, 16-colour, 256-colour and truecolor terminals each render correctly with downsampling done via `colorprofile` at the RunE boundary and never inside `present`; the palette reads `HasDarkBackground` once at the call site and every hue is readable on Solarized Light (or macOS light Terminal) and a dark theme; `--color=auto|always|never` exists on every styled verb with `--color` > `NO_COLOR` > `CLICOLOR_FORCE` > `CLICOLOR=0` > TTY auto-detect, covered by a unit-test matrix over every combination (CLI-02, CLI-03, CLI-04)
  4. The golden oracle, the wire oracle and the TUI-01 archtest are unchanged; a regression test pins `NO_COLOR` + non-TTY plain output; and the `present` archtest goes RED against a planted import of every charm-family path added this milestone (`colorprofile`, `fang/v2`, `x/ansi`, …), the denylist updated in the same commit as the `go.mod` change (CLI-05, GRD-13)

**Notes**: Research pitfalls attached: 1 (lipgloss v2 removed the renderer — nothing downsamples automatically; route through `colorprofile` at the RunE boundary and check manually in a non-256-colour `TERM` and over SSH), 2 (this repo honours neither `CLICOLOR` nor `CLICOLOR_FORCE` today; `NO_COLOR` any-non-empty wins), 3 (`forbiddenImportPaths` is a fixed literal list — prefix-match on `charm.land/` or exact paths in the same commit, proven RED). Architecture anti-patterns: never widen `present.ChoosePresentation`'s two-arg signature — resolve `--color` + TTY + `NO_COLOR` into it from one shared `internal/cli` helper, with `--color` a persistent flag on root (the `inheritedFromAncestor` guard in `cli_reference_test.go` already handles it); and never colourise `Engine.Explore`/`Engine.Node`'s markdown strings — the styled branch calls `ExploreDetail`/`NodeDetail`, the same plain structs the UI consumes. `install`/`uninstall` branch inside `printAgentResults` over `agents.WriteResult`. Fang is a process-level wrapper at the `Execute()` boundary and `internal/cli` is outside the TUI-01 closure by design, so the `serve --mcp` transcript proof is a *new* guard, not an archtest amendment. The gh #13335 lesson is the `NO_COLOR` + non-TTY regression test. `progress_cli.go`'s stderr-fd variant has different TTY semantics and is not unified with the stdout resolver.
**Plans**: TBD
**UI hint**: yes

Plans:

- [ ] TBD

#### Phase 5: Agent Reach — Capability Model & Skill in Every Harness

**Goal**: One per-target capability table drives `install`, `uninstall`, `Detect` and `--print-config-style` for all eight targets, and every harness that has a skill mechanism receives the codegraph skill package — written once to the shared `.agents/skills/` path and verified live to be read, with a harness-specific directory only where a live session shows it is needed — every write exact-identity-owned and reversed by `uninstall` leaving unrelated content byte-identical.
**Depends on**: Nothing in code (disjoint from Phases 3–4: `internal/agents` and the embedded skill assets only). Sequenced before Phases 6–7 because Codex is the capability model's first consumer and the nudge's install path derives from the same table.
**Requirements**: AGENT-08, AGENT-09, AGENT-04, AGENT-06, AGENT-07, AGENT-10, AGENT-11, AGENT-13
**Success Criteria** (what must be TRUE):

  1. `AgentTarget` exposes per-target capabilities — skill directories, instructions path, hook mechanism, supported scopes — and `install`, `uninstall`, `Detect` and `--print-config-style` derive from that one table rather than eight re-implementations; `--print-config-style` for each of the eight targets reports what the table says (AGENT-08)
  2. The skill package with its sidecar manifest is written once to `.agents/skills/codegraph/` (project) and `~/.agents/skills/codegraph/` (global), and each harness documented as reading that path is verified live to discover it — a harness-specific directory exists only where a live session showed the shared path is not read (AGENT-09)
  3. Cursor's repo-root `AGENTS.md` pickup is probed in a live Cursor session before any Cursor-specific instructions target is written, with the outcome recorded either way; opencode reads the skill at a path it actually reads with SKILL.md frontmatter compatibility verified and its instructions block retained; Antigravity gets the skill via `.agents/skills/` and the instructions block in `AGENTS.md`; Gemini CLI gets the skill at `.gemini/skills/` and its global equivalent, discoverable via `activate_skill`, with its instructions block retained; Kiro gets the skill at `.kiro/skills/` and `~/.kiro/skills/` with `AGENTS.md` retained as a steering source (AGENT-04, AGENT-06, AGENT-07, AGENT-10, AGENT-11)
  4. A planted-foreign-entry test per harness proves an unrelated sibling entry is never overwritten, and `uninstall` reverses every write leaving unrelated content byte-identical — with commit `242ec0a` cited in review (AGENT-13)

**Notes**: Research pitfalls attached: 10 (shape/position ownership is a *reverted* vulnerability in this repo — write the ownership-identity test first per harness), 11 (per-harness paths and mechanisms are verified against current docs, not memory; every `[ASSUMED]` in research — Cursor `AGENTS.md` auto-pickup, opencode's plugin mechanism, Antigravity's hook location — is resolved by a live session or stays labelled), 12 (fresh-session evidence for CLI-only harnesses is a real transcript with the negative-space check, never a summary). `Capabilities()` is additive — one literal per target file, mirroring `SupportsLocation` — with no change to `Install`/`Uninstall` semantics. The skill content is the one embedded `SKILL.md`; no second skill file is authored. Hermes is out of scope (AGENT-12, v2): no public documentation was found, and nothing beyond the existing MCP config is claimed. Nudge hooks for these harnesses are v2 by maintainer decision (NUDGE-07…10); this phase ships skill and instructions only. `AGENT-14` (the published capability table) is deliberately in Phase 7 so it describes what actually ships after the last harness write.
**Plans**: TBD

Plans:

- [ ] TBD

#### Phase 6: Claude Code PreToolUse Nudge

**Goal**: In a `.codegraph/`-indexed repo, Claude Code is pointed at `codegraph_explore` the first time it reaches for grep/find/Read — as added context that never denies or blocks a tool call — at most once per session, with zero overhead in an un-indexed repo, and registered and removed by `codegraph install`/`uninstall` as an opt-in through the existing exact-identity hook writer.
**Depends on**: Phase 5 (the hook's install path derives from the capability table; the `writeHookEntry`/`removeHookEntry` mechanism is unchanged). Sequenced before Phase 7 because `CODEX-05` reuses this phase's contract.
**Requirements**: NUDGE-03, NUDGE-04, NUDGE-05, NUDGE-06
**Success Criteria** (what must be TRUE):

  1. A PreToolUse hook on `Bash` (grep/rg/find as the first binary in the pipe), `Grep`, `Glob` and `Read` in an indexed repo returns only `hookSpecificOutput.additionalContext` pointing at `codegraph_explore`, exits 0 unconditionally — every internal error path forced — and never emits `permissionDecision`, with the matcher grammar and output shape verified against the current hooks reference (NUDGE-03)
  2. The nudge fires at most once per session via a session-scoped sentinel, and in a repo without `.codegraph/` it is silent with zero overhead — one directory check, no binary invocation, no index read (NUDGE-04)
  3. The hook is validated against a false-positive corpus (legitimate grep use) and a true-positive corpus (where-is-X patterns), and its fire rate against matched-tool-call count is measured in a genuinely fresh live session (NUDGE-05)
  4. `codegraph install`/`uninstall` register and remove the hook through `writeHookEntry`/`removeHookEntry` as an opt-in alongside the default SessionStart nudge; a hand-edited own entry duplicates rather than overwrites, and an unrelated `PreToolUse` entry under the same event is untouched (NUDGE-06)

**Notes**: Research pitfalls attached: 8 (a hook that "redirects" reintroduces the friction GUARD-HOOK-01/02 was deferred to avoid — the out-of-scope entry is explicit: no `permissionDecision: deny|ask`, no exit 2), 9 (a blanket tool-name match is noise that trains the agent to ignore it — gate on `.codegraph/` presence AND a content heuristic). The mechanism is already event-generic: a new `PreToolUse` array in the same embedded `hooks.json` fragment, a new script beside `session-nudge.sh`, a `claudePreToolUseBlocks(loc)` mirroring `claudeSessionStartBlocks(loc)`, and a second `writeHookEntry(settingsPath, "PreToolUse", …)` call alongside the existing one — `shared.go`'s exact-command-string matching is unchanged. Carries GUARD-HOOK-02's live fire-rate measurement. Supersedes GUARD-HOOK-01/02 by the 2026-09-14 reframe.
**Plans**: TBD

Plans:

- [ ] TBD

#### Phase 7: Codex Parity

**Goal**: Codex receives everything Claude Code does — project-local MCP config through the existing TOML splice, the skill package at the path(s) Codex actually reads, a repo-root `AGENTS.md` block, and an opt-in PreToolUse nudge carrying Phase 6's contract — with every mechanism verified in a live scratch project before a line of `codex.go` changes, the picker footer verified after the target count settles, the published capability table honest about what ships, and a genuinely fresh Codex session reaching for codegraph unprompted.
**Depends on**: Phase 5 (first consumer of the capability model) and Phase 6 (`CODEX-05` reuses the nudge contract). `FIX-03`'s fix lands before `CODEX-02` and its tmux assertion is re-run after it.
**Requirements**: CODEX-01, CODEX-02, CODEX-03, CODEX-04, CODEX-05, CODEX-06, FIX-03, AGENT-14
**Success Criteria** (what must be TRUE):

  1. Before any `codex.go` change, a live verification in a scratch trusted project records — with dated citations — whether the current Codex CLI loads a project-scoped `.codex/config.toml`, which skill path(s) it reads (`.agents/skills/` vs `.codex/skills/`), and whether `hooks.json` works behind `features.hooks`; `codex.go`'s "no per-project config" comment is corrected in the same commit (CODEX-01)
  2. `SupportsLocation(LocationLocal)` is true; project-local `install` writes `.codex/config.toml` through the TOML splice with existing tables, inline tables, comments and CRLF preserved and tells the user the project must be trusted; `--target auto` detection and the agent picker reflect the new scope; the skill package is installed to the verified Codex path(s) at both scopes idempotently and byte-invariant against sibling content; project-local install writes the marker-fenced block into the repo-root `AGENTS.md` with global `~/.codex/AGENTS.md` unchanged, and `uninstall` removes it leaving the rest of the file byte-identical (CODEX-02, CODEX-03, CODEX-04)
  3. A Codex PreToolUse nudge in `hooks.json` carries the same additionalContext-only, once-per-session contract as NUDGE-03/04; it is installed only when Codex's hooks feature is enabled (opt-in), skipped with a message otherwise, and its experimental, Windows-unsupported status is stated in the docs (CODEX-05)
  4. A genuinely fresh Codex session in an indexed repo reaches for codegraph unprompted — the skill is listed, and the MCP tool is called or `codegraph explore` is run — with the evidence recorded to the v0.10.0 live-session standard, and `codex mcp list` shows the entry at both scopes (CODEX-06)
  5. The install/uninstall agent picker renders its help footer in a 100×30 pane with every registered target listed, the height budget accounting for bubbles v2 list pagination, asserted by the tmux harness *after* `CODEX-02`'s scope flip — the milestone's last change to the target count; and the published per-harness capability table (MCP config, instructions, skill, nudge, scopes for all 8 targets) matches what ships, `[ASSUMED]` where live verification was not possible, with `instructions.go`'s "4 of 8" comment and the MCP `instructions` skill sentence updated to match (FIX-03, AGENT-14)

**Notes**: Research pitfalls attached: 11 (Codex's config surface changed during 2026 — project-scoped `.codex/config.toml` is trust-gated and silently falls back when untrusted, the exact "looks configured, does nothing" shape rule `84d1gfpywd` exists to catch; the doc comment and the code change together with a dated citation), 12 (a Codex transcript is the evidence, not a summary; the negative-space check — did it grep instead — is recorded), 19 (the footer fix is measured against the real 100×30 budget with the *final* target count, not at default terminal size). Codex hooks are experimental, disabled by default and unsupported on Windows — `CODEX-05` is opt-in and says so. Required companion changes when `SupportsLocation(LocationLocal)` flips: `Detect(LocationLocal)` must check the local `.codex/` path or `--target auto --location local` treats Codex as never installed; `DescribePaths(loc)` must add the local paths or `--print-config-style` under-reports; `agentpicker_test.go` gains the newly selectable row. `spliceTOMLTable`/`stripTOMLTable` operate on content, not paths, so a second file needs no `toml.go` change, and `findTOMLTableRange` already confines edits to `[mcp_servers.codegraph]`'s own range. `AGENT-14` closes the milestone's docs so the table describes the shipped surface, never the plan.
**Plans**: TBD

Plans:

- [ ] TBD

## Progress

**Execution Order:**
Phases execute in numeric order: 1 → 2 → 3 → 4 → 5 → 6 → 7. Only four edges are genuine code dependencies — Phase 4's `search`/`daemon` renderers need Phase 3's folded surface to exist; Phase 6's hook install and Phase 7's Codex writes need Phase 5's capability table; and Phase 7's `CODEX-05` reuses Phase 6's nudge contract. The rest is deliberate sequencing: the burn-down first because it is the cheapest work and the most likely to expose a wrong assumption about this repo's own gates, split into a `FIX-` phase and a `GRD-`/`DOCS-` phase so the fixes are not scattered; the fang verdict (`CLI-08`) as Phase 4's first plan so no renderer or help template is written and then thrown away; `CODEX-01`'s live verification as Phase 7's first task so no `codex.go` line is written against a stale assumption. Two requirements sit outside their prefix's phase on purpose — `GRD-13` rides with Phase 4's first charm import, and `FIX-03` is fixed before and re-verified after Phase 7's `CODEX-02`, the milestone's last change to the install target count. The two feature tracks (3→4 in `internal/cli`, 5→6→7 in `internal/agents`) share no files and could run as parallel workstreams. Phase numbering restarted at 1 by `--reset-phase-numbers`; `.planning/phases/` held only the two `999.x` backlog directories at the start (both consumed by v0.13.0 and kept, annotated, by the 2026-09-08 bookkeeping decision). Backlog below is preserved across milestone closes.

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 1. Defect & Flake Burn-down | 9/9 | Complete    | 2026-09-15 |
| 2. Guards, CI Wiring & Docs Burn-down | 7/7 | In Progress|  |
| 3. Verb Fold | 0/TBD | Not started | - |
| 4. CLI Glow-up | 0/TBD | Not started | - |
| 5. Agent Reach — Capability Model & Skill in Every Harness | 0/TBD | Not started | - |
| 6. Claude Code PreToolUse Nudge | 0/TBD | Not started | - |
| 7. Codex Parity | 0/TBD | Not started | - |

8 milestones shipped (v0.1, v1.0, v0.3.0, v0.5.0, v0.10.0, v0.11.0, v0.12.0, v0.13.0). v0.14.0 scoped: 7 phases (1–7), 55 requirements, 0/7 phases complete (0%).

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
