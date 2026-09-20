# Roadmap: CodeGraph Go

## Overview

CodeGraph Go is a Go implementation of a pre-indexed code knowledge graph for coding agents. Eight milestones have shipped: **v0.1** (2026-07-14) landed the core capabilities — indexing, query, MCP server, sync — from a signed/attested/SBOM'd release. **v1.0** (2026-08-03) closed the behavioral and surface gaps, adding a human-facing Charm TUI behind a build-enforced rendering seam that keeps the agent/MCP path free of ANSI, plus fully automated signed releases via release-please + GoReleaser. **v0.3.0** (2026-08-06) brought the stdio MCP server current with spec revision `2026-07-28` on `modelcontextprotocol/go-sdk@v1.7.0`, proven by a wire-level oracle that never imports the SDK it tests. **v0.5.0** (2026-08-11) made the binary installable by convention on macOS — Gatekeeper-accepted on both darwin arches and `brew install`-able from a tap we control. **v0.10.0** (2026-08-13) made agents actually *use* the tools: the server documents itself over MCP Resources, a decision-procedure-first SKILL.md teaches which question goes to which tool, and a SessionStart nudge makes availability visible at the moment it matters. **v0.11.0** (2026-08-16) retired the comparison framing without retiring capability, re-basing the golden suite onto measurement-selected corpora and retiring Compatibility as a constraint.

**v0.12.0 (Local Graph UI) shipped 2026-09-07.** Every consumer of the graph until now was a program — a CLI invocation or an agent over MCP. This milestone gave the graph a human face: `codegraph ui` serves a local, read-only web UI from the binary itself, so a developer can browse, visualize, query and health-check a `.codegraph/` index without going through an agent. It is deliberately a **third consumer** of `internal/query.Engine`, never a second implementation — the cross-phase integration check found no duplicated traversal logic in `internal/uiserver`, with every handler routing through one `withEngine` seam. The wire is ConnectRPC over Protobuf (maintainer directive), the app is a `pnpm`-built Svelte SPA committed and `go:embed`'d so the signed release path stays pure Go, and the whole surface binds loopback with exact-match Origin/Host validation because loopback binding alone has already been walked through by a documented CVE class. 51/51 requirements, 6/6 phases verified *and* validated, six per-phase SECURITY.md files all at `threats_open: 0`.

**v0.13.0 (Guard Hardening & UI Follow-through) shipped 2026-09-13.** A deferral burn-down in two coherent sets. First, every known guard that cannot fire — this repo's own recurring defect shape is an assertion that is true but non-discriminating, so it passes whether the property holds or not (rule `84d1gfpywd`) — closed with a recorded RED demonstration each, plus the real-PTY testing gap: a tmux harness that drives the release binary in a genuine pane and gates CI on an exact executed count. Second, the four contained UI follow-ons v0.12.0 deliberately left — scroll breadcrumb and editor handoff, the coverage denominator that answers "why is my file missing", and community clustering on the graph view — all on the unchanged 16-rpc read-only wire with every proto change additive. The documentation tail became a `cobra/doc`-generated `docs/CLI-REFERENCE.md` under a drift gate plus an allowlist guard for the generator's one blind spot, after the maintainer reset that phase's scope at discuss time. 26/26 requirements, 6/6 phases verified (re-verified to canonical `passed` at close), 21 mutation families, six SECURITY.md files at `threats_open: 0`, a `verified_closeout`. Promoted and delivered backlog **999.2** and **999.4**.

**v0.14.0 (Polish & Agent Reach) shipped 2026-09-20.** Three milestones of feature work had left a ledger of small, known, individually-cheap defects that were expensive together. This milestone burned them down and, on the same pass, gave the CLI a colour-and-structure glow-up on a folded verb surface and took agent reach from Claude-Code-shaped to harness-shaped: one capability table drives install/uninstall for all 8 targets, the skill package and instructions block go wherever each harness reads them, an opt-in PreToolUse nudge serves Claude Code and Codex from one core, and Codex reached parity at both scopes — each mechanism live-verified against codex-cli 0.155.0 before any code changed. Two data-loss-class bugs surfaced and were fixed on the way: the Codex TOML splice and a terminal escape injection through install notes.

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
- ✅ **v0.14.0 — Polish & Agent Reach** — Phases 1–7 (shipped 2026-09-20) — every known defect, vacuous guard, flake and doc drift burned down; the verb surface folded and every human-output verb styled through `present`; the skill, capability model and an opt-in PreToolUse nudge delivered across all 8 harnesses with a published drift-tested table; Codex brought to parity at both scopes, live-verified before any `codex.go` change
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

<details>
<summary>✅ v0.14.0 Polish &amp; Agent Reach (Phases 1–7) — SHIPPED 2026-09-20</summary>

Archived: [`milestones/v0.14.0-ROADMAP.md`](./milestones/v0.14.0-ROADMAP.md) · audit: [`milestones/v0.14.0-MILESTONE-AUDIT.md`](./milestones/v0.14.0-MILESTONE-AUDIT.md) · phase directories: `milestones/v0.14.0-phases/`

7 phases, 53 plans, 55/55 requirements. All phases verified, validated and security-audited (0 threats open at or above `high`). Phase numbering restarted at 1 for this milestone.

</details>

## Progress

8 milestones shipped through v0.13.0; **v0.14.0 shipped 2026-09-20** — 7 phases (1–7), 53 plans, 55/55 requirements,
all phases verified, validated and security-audited. Per-phase detail lives in the archive
([`milestones/v0.14.0-ROADMAP.md`](./milestones/v0.14.0-ROADMAP.md)) and the audit
([`milestones/v0.14.0-MILESTONE-AUDIT.md`](./milestones/v0.14.0-MILESTONE-AUDIT.md)).

No milestone is currently scoped. Start the next one with `/gsd-new-milestone`.

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

### Phase 999.5: remove the query/unlock rename stubs

**Goal:** Remove the hidden `query` and `unlock` rename stubs registered by Phase 3 (VERB-03/VERB-04, D-05/D-09; feat commit 5d69ee2ea3c6276ed73c946b24be93612fae1698) in v0.15.0, the minor after the fold: delete `internal/cli/renamed.go` and `internal/cli/renamed_test.go`, drop `newQueryCmd()` and `newUnlockCmd()` from `internal/cli/root.go`'s AddCommand list, remove the two `codegraph query` / `codegraph unlock` lines from `internal/cli/testdata/cli-reference-allowlist.txt`, run `task docs:cli` (no reference change expected — the stubs are hidden), and re-run the Phase 3 census instrument (03-MUTATION-LOG.md Family (c)) expecting zero hits.
**Requirements**: TBD
**Depends on:** Phase TBD
**Plans:** 0 plans

Plans:

- [ ] TBD (run /gsd-plan-phase 999.5 to break down)
