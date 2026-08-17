# Roadmap: CodeGraph Go

## Overview

CodeGraph Go is a Go implementation of a pre-indexed code knowledge graph for coding agents. Six milestones have shipped: **v0.1** (2026-07-14) landed the core capabilities — indexing, query, MCP server, sync — from a signed/attested/SBOM'd release. **v1.0** (2026-08-03) closed the behavioral and surface gaps, adding a human-facing Charm TUI behind a build-enforced rendering seam that keeps the agent/MCP path free of ANSI, plus fully automated signed releases via release-please + GoReleaser. **v0.3.0** (2026-08-06) brought the stdio MCP server current with spec revision `2026-07-28` on `modelcontextprotocol/go-sdk@v1.7.0`, proven by a wire-level oracle that never imports the SDK it tests. **v0.5.0** (2026-08-11) made the binary installable by convention on macOS — Gatekeeper-accepted on both darwin arches and `brew install`-able from a tap we control. **v0.10.0** (2026-08-13) made agents actually *use* the tools: the server documents itself over MCP Resources, a decision-procedure-first SKILL.md teaches which question goes to which tool, and a SessionStart nudge makes availability visible at the moment it matters.

**v0.11.0 (Standalone Project Identity) shipped 2026-08-16.** Everything the project *did* already stood on its own; the way it *described and tested itself* did not. This milestone retired that framing without retiring capability — with one recorded exception. The origin is acknowledged exactly once, legally and in the past tense, in `NOTICE` plus one clause in README's `## License`; comparison vocabulary is gone from every doc, template, workflow, script, comment, identifier and fixture, closed by a positive-controlled census reporting `TOTAL=0` across 285 Go files. The load-bearing part was never prose: the golden suite derived its oracle from corpora chosen because the origin project used them. Re-basing it meant selecting corpora **by measurement** — recorded per-kind edge counts and per-language file counts from real indexing runs — and then re-proving the re-baselined suite could still fail. The exception is `codegraph migrate`, dropped outright (maintainer ruling D-04) because the migration path *was itself* the parity framing. The **Compatibility constraint is retired**: behavior is now defined by this project's own requirements and its own frozen goldens.

**Versioning note:** "v1.0" is a *planning-milestone* name, never a release version. The shipped artifact line reached `v0.2.0` at v1.0's close and has since advanced through `v0.3.0`, `v0.4.0`, `v0.5.0` … `v0.9.0`, plus `v0.10.0`, each computed by release-please from Conventional Commits; there is deliberately no `v1.0.0` tag (maintainer directive D-06R, 2026-07-29). Milestone labels track the release line but carry **no git tag**: release-please remains the sole tag authority, pinned by `TestGsdTagCreationIsDisabled`. A hand-created `v*` tag would additionally match `release.yml`'s `push: tags: "v[0-9]*"` trigger and falsely fire the release pipeline. The milestone record lives in `MILESTONES.md` + `milestones/`. (`milestone-v0.1` exists only because it predates release-please.)

## Milestones

- ✅ **v0.1 — Initial Release** — Phases 1–8 (shipped 2026-07-14) — core capabilities + signed release
- ✅ **v1.0 — Drop-in Parity & Human UX** — Phases 1–10 (shipped 2026-08-03) — behavioral + surface parity, human TUI, automated signed releases, local build tooling
- ✅ **v0.3.0 — MCP Protocol Currency** — Phases 1–5 (shipped 2026-08-06) — official Go SDK adoption, `2026-07-28` spec compliance without breaking Legacy clients, a wire-level verification oracle, tool-modfile vulnerability coverage
- ✅ **v0.5.0 — macOS Distribution & Homebrew** — Phases 1–4 (shipped 2026-08-11) — `goreleaser release` migration with zig cross-compilation, Apple notarization, a Homebrew tap and cask, and an `upgrade` that steps aside under brew. Promoted backlog 999.5, consumed SEED-002
- ✅ **v0.10.0 — Agent Onboarding Skill & MCP Resources** — Phases 5–8 (shipped 2026-08-13) — the server documents itself over MCP Resources, a decision-procedure-first SKILL.md plus a SessionStart nudge teach agents when to reach for it, `codegraph install` ships that package with the binary, and the stale `instructions` promise was retired last. Consumed the 2026-08-08 skill todo
- ✅ **v0.11.0 — Standalone Project Identity** — Phases 1–6 (shipped 2026-08-16) — origin acknowledged once in `NOTICE` plus one README License clause; comparison framing removed tree-wide to a proven zero; golden corpora re-selected by measurement, re-frozen from codegraph-go's own output, and re-proven non-vacuous; benchmarks published as mechanically-generated absolute numbers; `codegraph migrate` and the `modernc.org/sqlite` dependency removed (D-04); Compatibility retired as a constraint
- 📋 **Later** — unscoped. Candidates: v0.10.0's v2 deferrals (PreToolUse guard hook GUARD-HOOK-01/02, multi-agent skill+hooks porting AGENT-04…07), v0.11.0's v2 deferrals (DOCS-05 self-authored CLI reference; VOCAB-01 was *declined*, not deferred), the Backlog items below (999.2 tmux TTY harness, 999.4 CheckRegression guard), the v0.5.x deferrals (DIST-06 stapled offline-safe container, BREW-07 homebrew-core), Team Scale (central server, CI-distributed indexes), MRTR/elicitation (MRTR-01), annotations (embeddings/communities/export), local Svelte web UI (SEED-001)

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

## Progress

**No active milestone.** v0.11.0 shipped 2026-08-16. Run `/gsd-new-milestone` to scope the next one — it starts with fresh requirements, since `REQUIREMENTS.md` is milestone-scoped and was archived at close.

6 milestones shipped. Backlog below is preserved across milestone closes.

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

