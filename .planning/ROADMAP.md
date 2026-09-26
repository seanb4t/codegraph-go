# Roadmap: CodeGraph Go

## Overview

CodeGraph Go is a Go implementation of a pre-indexed code knowledge graph for coding agents. Nine milestones have shipped: **v0.1** (2026-07-14) landed the core capabilities — indexing, query, MCP server, sync — from a signed/attested/SBOM'd release. **v1.0** (2026-08-03) closed the behavioral and surface gaps, adding a human-facing Charm TUI behind a build-enforced rendering seam that keeps the agent/MCP path free of ANSI, plus fully automated signed releases via release-please + GoReleaser. **v0.3.0** (2026-08-06) brought the stdio MCP server current with spec revision `2026-07-28` on `modelcontextprotocol/go-sdk@v1.7.0`, proven by a wire-level oracle that never imports the SDK it tests. **v0.5.0** (2026-08-11) made the binary installable by convention on macOS — Gatekeeper-accepted on both darwin arches and `brew install`-able from a tap we control. **v0.10.0** (2026-08-13) made agents actually *use* the tools: the server documents itself over MCP Resources, a decision-procedure-first SKILL.md teaches which question goes to which tool, and a SessionStart nudge makes availability visible at the moment it matters. **v0.11.0** (2026-08-16) retired the comparison framing without retiring capability, re-basing the golden suite onto measurement-selected corpora and retiring Compatibility as a constraint. **v0.15.0 (Changie Release Management) is in progress (scoped 2026-09-25):** it replaces release-please with changie so the changelog is authored input — one fragment per user-visible change, written at phase close — and the version is derived from it, proven by cutting a real release through the new chain.

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
- 🚧 **v0.15.0 — Changie Release Management** — Phases 1–6 (in progress) — release-please replaced by changie: `.changie.yaml` with a `v0.14.0` baseline that rewrites no history; a private phase-close capability writing one fragment per user-visible change at `verify:post`; a required fragment gate; a release-PR workflow that tags through the GitHub App into the unchanged `release.yml`; the `query`/`unlock` stubs removed as the milestone's `Breaking` fragment; and a real release cut through the chain. Promotes backlog 999.5 as VERB-09; SEED-005's trigger. Phase numbering restarts at 1 (`--reset-phase-numbers`)
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

### 🚧 v0.15.0 — Changie Release Management (In Progress)

**Milestone Goal:** Replace release-please with changie so the changelog is authored input — one fragment per user-visible change, written at phase close — and the version is derived from it, then prove the chain by cutting a real v0.15.0 release through it.

**Phase numbering restarts at 1** (`--reset-phase-numbers`). v0.14.0 ran Phases 1–7 and its phase directories are archived under `milestones/v0.14.0-phases/`; `.planning/phases/` holds only the three `999.x` backlog directories (kept, annotated, by the 2026-09-08 bookkeeping decision), so `01-*` through `06-*` collide with nothing.

**Ordering is load-bearing, not stylistic:**

- **The changie baseline (Phase 1) comes first because every later phase writes fragments against it.** The kind vocabulary, the required `PR` field and the `v0.14.0` baseline are fixed before a single fragment exists, so no fragment is written against a config that later changes under it. `changie merge --dry-run` reproducing today's `CHANGELOG.md` byte-for-byte is the proof that adopting changie rewrote no history.
- **The phase-close capability (Phase 2) lands second so that phases close after it is installed.** `CAP-05` accepts only evidence of the `verify:post` step firing at a real phase close inside this milestone; installing it late would leave nothing to observe. Phase 3's close is the first one it must fire at, and Phases 4–5 are further chances if it does not — a missed firing is re-attempted at the next close, never papered over by a hand-written fragment presented as the capability's. The capability repository `seanb4t/gsd-capability-changie` is private and separate: its commits happen outside this roadmap's commit trail, and what this repository's requirements ask for is "installed and firing".
- **The stub removal (Phase 3) is small, early and deliberately its own phase.** It is the milestone's only `Breaking` change and therefore the change that makes `changie batch auto` derive a minor (`Breaking → minor` until 1.0). It must be closed before Phase 6 batches the release PR, or the derived version is wrong. It is also the cleanest user-visible change to be the capability's first real input.
- **The fragment gate (Phase 4) lands before the release chain (Phase 5)** so that `CONTRIBUTING.md`'s "how to exempt" (`DOCS-12`) documents a gate that exists, and so `scripts/pr_template_policy.py` is edited by the gate before `GRD-15` re-points its release-please exemptions — one file, two edits, in a fixed order. `GATE-05` is a repository-settings action the maintainer performs; the `.github/required-status-checks.txt` fixture and `TestRequiredCheckNamesPreserved` change in the same commit as the ruleset edit or that test goes red, so the phase halts at a blocking-human checkpoint for it (the v0.14.0 `GRD-12` precedent).
- **The replacement and the re-pointed guards land in one phase (Phase 5), and `REL-14` closes it.** Every test that pins release-please as an authority goes RED against the replacement and is re-pointed in the same phase that introduces it — never left broken across a phase boundary. The docs (`DOCS-12…14`) are in this phase too, because `REL-14`'s zero-reference census covers `CONTRIBUTING.md` and `docs/RELEASE-PROCEDURES.md`: the census cannot read zero until their references are resolved, so deleting the release-please files is the phase's last change.
- **The first real release (Phase 6) is last and is the milestone's definition of done.** Before merge, `REL-10`/`REL-11` can be proven only by shape tests, actionlint and a dry-run; the chain's live behaviour exists only on `main`. Phase 6 therefore runs after the milestone PR merges and carries three human checkpoints: merging the release PR, verifying the App-authored tag, and running `codegraph upgrade` from v0.14.0 on two platforms.
- **Locked contracts bind every phase.** `release.yml`'s `on: push: tags: v[0-9]*` trigger and `internal/upgrade/verify.go`'s cosign identity `release.yml@refs/tags/v[0-9]*` are byte-locked; only who pushes the tag changes. Every action is SHA-pinned with a version comment. `Taskfile.yml` is the single definition of every CI job body (`TestWorkflowRunBodiesInvokeTask`). No human `git tag`: D-06R's tag authority transfers from release-please to the changie release workflow; it is not repealed. Never `[ci skip]` in a commit message (rule `f18zrdsgx5`).
- **A guard is not trusted until demonstrated RED** against a confirmed-applied, byte-cleanly-reverted mutation (rule `84d1gfpywd`): the `CHG-03` byte-reproduction check against a one-byte `CHANGELOG.md` edit, the fragment gate against a mutated path rule, every re-pointed release guard against the replacement, the tag-authority assertion against a planted second tag push. Each phase commits its mutation log rather than asserting it.
- **`v0.15.0` is a prediction, not a tag.** changie derives the real version from the fragments present at batch time; no phase schedules a `git tag` step.

- [x] **Phase 1: Changie Baseline** - `.changie.yaml` and a `v0.14.0` baseline that reproduce today's `CHANGELOG.md` byte-for-byte, with fragments written non-interactively and malformed ones refused (completed 2026-09-25)
- [ ] **Phase 2: Phase-Close Fragment Capability** - A private `gsd-capability-changie` feature capability writes a phase's fragments at `verify:post`, installed and configured in this repository at project scope
- [ ] **Phase 3: Rename-Stub Removal** - The hidden `query`/`unlock` stubs are gone, and this phase's close is the capability's first observed firing, writing the milestone's `Breaking` fragment
- [ ] **Phase 4: Fragment-Required Gate** - A pull request that changes shipped code cannot merge without a fragment or a reasoned exemption, and the check is required on `main`
- [ ] **Phase 5: Release Chain & release-please Retirement** - A changie release-PR workflow tags through the GitHub App into the locked `release.yml`, every release-please guard is re-pointed, the docs describe the chain, and release-please is deleted
- [ ] **Phase 6: First Release Through the Chain** - A real release is cut through the new chain end to end, and an installed v0.14.0 upgrades to it with verification

#### Phase 1: Changie Baseline

**Goal**: The repository has a changie configuration and baseline that answers `v0.14.0` as the latest version and reproduces today's `CHANGELOG.md` exactly, so every later phase writes fragments against a fixed vocabulary and no history is rewritten.
**Depends on**: Nothing (first phase of the milestone).
**Requirements**: CHG-01, CHG-02, CHG-03, CHG-04
**Success Criteria** (what must be TRUE):

  1. `changie latest` answers `v0.14.0`, `changie next auto` with an empty `.changes/unreleased/` fails with the expected "nothing to release" outcome, and `changie merge --dry-run` reproduces the current `CHANGELOG.md` byte-for-byte — all three run by a committed test or Taskfile target rather than a one-off shell transcript, and the byte-reproduction check is demonstrated RED against a one-byte `CHANGELOG.md` mutation, byte-cleanly reverted (CHG-03)
  2. `.changie.yaml` declares exactly the five kinds with their `auto` bumps (`Breaking → minor`, carrying the flip-to-major-at-1.0 note in the file), the required integer `PR` field with `minInt: 1`, and the version/kind/change formats from `notes/changie-release-management.md`; `changie` is pinned at v1.26.0 or later with one recorded install path for CI and one for contributors (CHG-01)
  3. `.changes/header.tpl.md`, `.changes/unreleased/.gitkeep` and `.changes/v0.14.0.md` exist, and every existing `CHANGELOG.md` entry below the changie header is byte-identical to `main` — baseline only, no historical entry rewritten (CHG-02)
  4. `CI=true changie new -k <Kind> -b "<sentence>" -m PR=<n>` writes a `.changes/unreleased/*.yaml` without prompting, while a fragment missing `PR` and a fragment naming an undeclared kind are each refused — both directions shown, with the refusal cases confirmed to have executed rather than inferred from a green exit (CHG-04)

**Notes**: `changie merge` regenerates `CHANGELOG.md` from the header plus every `.changes/v*.md` file, so byte-reproducing today's history is not automatic when only `.changes/v0.14.0.md` is seeded — how the pre-v0.14.0 entries survive a merge (where they live, and whether that stays inside "no historical rewrite") is this phase's first research question, answered by a real `changie merge --dry-run`, not by reading the docs. The uniform regeneration of historical entries from GitHub Release notes is `CHG-05`, deferred to v2. `CHANGELOG.md` is tool-owned (today by release-please, from this phase by changie); no entry is hand-edited. The two install paths are recorded where contributors and CI will find them — `CONTRIBUTING.md`'s prose rewrite belongs to `DOCS-12` in Phase 5.
**Plans**: 2/2 plans executed

Plans:
**Wave 1**

- [x] 01-01-PLAN.md — Pinned changie renders the seeded `v0.14.0` baseline byte-for-byte (tracer): D-06 shape guards RED-first, `.changie.yaml`, 14 verbatim seeds, isolated `go.tool-changie.mod`, `task changie` wrapper, CONTRIBUTING install path (CHG-01, CHG-02)

**Wave 2**

- [x] 01-02-PLAN.md — `check:changie` 11-leg live-tool guard wired into ci.yml after `docs:cli:drift`, changie added to `task vuln`, and the RED mutation log for the byte-reproduction, missing-PR, collision and seed-set guards (CHG-01, CHG-03, CHG-04)

#### Phase 2: Phase-Close Fragment Capability

**Goal**: Closing a GSD phase in this repository writes that phase's changelog fragments without a human remembering to: a private `role: "feature"` capability, `seanb4t/gsd-capability-changie`, owns a `changie-fragments` skill dispatched at `verify:post`, and codegraph-go installs it at project scope. The capability repository is separate and private — its commits happen outside this roadmap's commit trail — so what this phase delivers *here* is the capability installed, configured and documented; its first real firing is proven at Phase 3's close (`CAP-05`).
**Depends on**: Phase 1 (the skill reads the kinds declared in `.changie.yaml` and writes through `changie new`).
**Requirements**: CAP-01, CAP-02, CAP-03, CAP-04
**Success Criteria** (what must be TRUE):

  1. `seanb4t/gsd-capability-changie` exists as a private repository with a README, a LICENSE and a tagged release, and its `capability.json` (`role: "feature"`, an id outside the reserved `gsd-` prefix, `engines.gsd` pinned) installs cleanly on a scratch project under gsd-core 1.14.0 with `gsd_run capability install ./ --scope project` (CAP-01)
  2. The manifest registers exactly one `verify:post` step (`ref.skill`, `onError: skip`) gated on the federated `workflow.changie_fragments` key (`boolean`, default `true`): on a scratch project the step is dispatched with the key `true` and never dispatched with it `false` — both observed (CAP-02)
  3. Given a phase number in a scratch project, the skill writes one fragment per user-visible change from that phase's `*-SUMMARY.md` files through `CI=true changie new`, using only the host `.changie.yaml`'s declared kinds, and commits them under the phase's commit conventions; in each of the three skip cases — no `changie` on `PATH`, no `.changie.yaml`, no user-visible change — it prints a reason and returns without prompting or blocking the workflow (CAP-03)
  4. codegraph-go has the capability installed at project scope from `https://github.com/seanb4t/gsd-capability-changie.git#<tag>`, `workflow.changie_fragments` is set in `.planning/config.json`, `.gsd/` stays gitignored, and `CONTRIBUTING.md` documents the one-time per-clone install (CAP-04)

**Notes**: `CHG-01` makes `PR` a required field (`minInt: 1`), but at phase close on the milestone branch no milestone PR number may exist yet — where the skill gets `<n>` (a draft milestone PR opened early, a lookup, or a config value) is an open design question for discuss-phase, not something to paper over with a placeholder number. The capability manifest is the sanctioned extension vocabulary; no gsd-core workflow file is edited (out of scope by requirement), and `.planning/config.json` is written through the tool's own config verb, never by hand. A `contribution` or `ref.command` step is out of scope — a skill step is the least-privilege sanctioned shape. The scratch-project proofs in criteria 1–3 are the capability's own tests; the in-repo proof that it fires on this repository is `CAP-05` at Phase 3's close.
**Plans**: 4/5 plans executed

Plans:
**Wave 1**

- [x] 02-01-PLAN.md — Capability repo tracer, RED first: manifest installs on a scratch project, gated `verify:post` step, fragment writer with every named skip, idempotency trailers, rollback and lock, proven by a 29-leg `test/run.sh` (CAP-01, CAP-02, CAP-03)

**Wave 2**

- [x] 02-02-PLAN.md — README, verbatim MIT LICENSE, a pre-publication judgment rehearsal of SKILL.md, and three RED mutation families (CAP-01, CAP-03)

**Wave 3**

- [x] 02-03-PLAN.md — Maintainer-approved publish: private repo, annotated tag `v0.1.0` proven from a clean clone, draft milestone PR, `/gsd-ship` hazard recorded (CAP-01, CAP-04)

**Wave 4**

- [x] 02-04-PLAN.md — Project-scope install from the tag, config-set, ledger ignored, live preflight; maintainer-decided Claude skill materialization (CAP-04)

**Wave 5**

- [ ] 02-05-PLAN.md — Real `Skill(gsd-changie-fragments)` dispatch proof, and the CONTRIBUTING D-12 subsection (CAP-03, CAP-04)

#### Phase 3: Rename-Stub Removal

**Goal**: The hidden `query` and `unlock` rename stubs v0.14.0 promised to remove in the following minor are gone, and this phase's close is the first observed firing of the phase-close capability in this repository — the milestone's `Breaking` fragment is the one the capability wrote.
**Depends on**: Phase 2 (`CAP-05` needs the capability installed before this phase closes). Must be closed before Phase 6 batches the release PR, so the derived version is a minor.
**Requirements**: VERB-09, CAP-05
**Success Criteria** (what must be TRUE):

  1. `codegraph query` and `codegraph unlock` exit non-zero as unknown commands rather than printing a rename message; `internal/cli/renamed.go` and `renamed_test.go` are deleted, `newQueryCmd()`/`newUnlockCmd()` are gone from `root.go`, and the two `cli-reference-allowlist.txt` lines are removed (VERB-09)
  2. `task docs:cli` produces no change to `docs/CLI-REFERENCE.md`, `task docs:cli:drift` and `TestEveryRegisteredFlagIsAccountedFor` are green, and the v0.14.0 Phase 3 census instrument (`03-MUTATION-LOG.md` family (c)) re-run with its positive control first finding a planted reference reports zero `codegraph query` / `codegraph unlock` hits (VERB-09)
  3. At this phase's close the `verify:post` step dispatches and writes the stub removal's `Breaking` fragment into `.changes/unreleased/`; that fragment's commit is on the milestone branch, and the phase's VERIFICATION or SUMMARY records the dispatch with the commit SHA — evidence, not assertion (CAP-05, VERB-09)
  4. With that fragment present, `changie next auto` on the milestone branch answers a minor bump over `v0.14.0` (predicted `v0.15.0`), not a major (VERB-09)

**Notes**: The census instrument is v0.14.0's own, reused verbatim, including its `--hidden` correction and positive control — a zero-hit result from an instrument not first shown able to find a planted hit is not evidence. If the step does not fire, `CAP-05` stays open and is re-attempted at Phase 4's close (and Phase 5's after that); VERB-09's `Breaking` fragment is still required, but a hand-written one is recorded as hand-written and does not satisfy `CAP-05`. The capability skipping at a later close that has no user-visible change (Phase 4's gate is CI-only) is a useful second observation of `CAP-03`'s skip path, not a failure.
**Plans**: TBD

Plans:

- [ ] TBD (run /gsd-plan-phase 3 to break down)

#### Phase 4: Fragment-Required Gate

**Goal**: A pull request that changes shipped code cannot merge without a changelog fragment or an explicit, reasoned exemption: a `fragment-required` check runs on every pull request, fails closed, and is a required status check on `main`.
**Depends on**: Phase 1 (`.changes/unreleased/` exists). Independent of Phases 2–3 in code; sequenced after them so the capability's first firing lands on a phase with a real user-visible change.
**Requirements**: GATE-01, GATE-02, GATE-03, GATE-04, GATE-05
**Success Criteria** (what must be TRUE):

  1. A pull request whose changed files touch `cmd/**` or `internal/**` (excluding `*_test.go` and `testdata/`) and add nothing under `.changes/unreleased/` fails `fragment-required`, and the identical shape plus a fragment passes — demonstrated in both directions against real fixtures, with the output recorded and the executed-case count asserted; the path rule lives in one place shared by the workflow and its tests (GATE-01, GATE-04)
  2. A body carrying `<!-- changelog-exempt: <reason> -->` with a non-empty reason passes, while an empty reason does not exempt — the marker grammar mirrors `pr-template-exempt` (GATE-02)
  3. An empty changed-file list, an API error and an unreadable body each fail with a distinct message; none of them passes (GATE-03)
  4. The gate is demonstrated RED against a confirmed-applied, byte-cleanly-reverted mutation of its path rule, recorded in the phase's mutation log, and its job body is a `task` invocation so `TestWorkflowRunBodiesInvokeTask` covers it (GATE-04)
  5. **Checkpoint (repository-settings action, maintainer):** `fragment-required` is in ruleset `protect-main`'s required status checks, with the ruleset edit's timestamp recorded; `.github/required-status-checks.txt` and `TestRequiredCheckNamesPreserved` change in the same commit as that edit, and `TestRequiredCheckNamesPreserved` plus `scripts/check-ruleset-drift.sh` pass against the live ruleset (GATE-05)

**Notes**: `GATE-05` says "7 contexts", but `.github/required-status-checks.txt` already lists 8 (v0.14.0 `GRD-12` grew it from 7 to 8 with `tmux e2e`), so adding `fragment-required` makes 9 — re-verify the live count read-only before the edit rather than trusting either number. Timing hazard for discuss-phase: a required check whose workflow exists only on the milestone branch never reports on any other pull request to `main`, so marking it required before the workflow is on `main` blocks every other PR until the milestone merges; the ruleset edit, the fixture commit and the workflow's arrival on `main` need one deliberate sequence (landing the gate on `main` mid-milestone, as v0.13.0 merged Phases 7–8 as PR #69, is one option). Precedents to reuse rather than reinvent: `require-issue-link.yml`'s fail-closed empty-list handling, `scripts/pr_template_policy.py`'s path-aware exemption, and FIX-10's per-run `$GITHUB_OUTPUT` delimiter for any fork-controlled file list. Planning-only pull requests touch neither `cmd/**` nor `internal/**` and pass without a marker.
**Plans**: TBD

Plans:

- [ ] TBD (run /gsd-plan-phase 4 to break down)

#### Phase 5: Release Chain & release-please Retirement

**Goal**: release-please is replaced end to end: a changie release workflow batches fragments into a reviewable release PR and, on its merge, pushes the tag with the GitHub App token into the unchanged `release.yml`; every guard that pinned release-please now pins the changie chain; the contributor and release docs describe the new chain; and release-please's files are gone.
**Depends on**: Phase 1 (`.changie.yaml` and the baseline) and Phase 4 (`DOCS-12` documents the gate's exemption; `GRD-15`'s `pr_template_policy.py` edit follows the gate's).
**Requirements**: REL-10, REL-11, REL-12, REL-13, REL-14, GRD-15, GRD-16, DOCS-12, DOCS-13, DOCS-14
**Success Criteria** (what must be TRUE):

  1. A new release workflow — actionlint-clean, every action SHA-pinned with a version comment, job bodies in `Taskfile.yml` — on a push to `main` with unreleased fragments runs `pretag-gate` (`task check:cross`, moved verbatim) then `changie batch auto && changie merge` and opens or updates one `chore(main): release vX.Y.Z` PR carrying the rendered `.changes/vX.Y.Z.md` and `CHANGELOG.md`; with no unreleased fragments it does nothing; on a new `.changes/vX.Y.Z.md` reaching `main` it mints an App installation token from `APP_ID`/`APP_PRIVATE_KEY` and pushes tag `vX.Y.Z` with it, never with `GITHUB_TOKEN` — each branch pinned by a shape test (REL-10, REL-11)
  2. `release.yml` differs from `main` by exactly one change — `goreleaser release` receives `--release-notes .changes/${TAG}.md` — with its trigger, SLSA/cosign/notarization/SBOM steps and job shape byte-identical, the shape tests asserting `changelog.disable: true` is absent, and `task release:dry-run` rendering a non-empty release body equal to a `.changes/vX.Y.Z.md` file, captured before any real tag exists (REL-12, REL-13)
  3. Every test that pinned release-please as an authority went RED against the replacement and was re-pointed within this phase: `TestReleasePleaseStaysPreMajor` reads `.changie.yaml`'s `Breaking → minor`, the Taskfile job registry names the new workflow's `pretag-gate`, the forbidden-`release:`-keys rationale is restated for GoReleaser-authored bodies, and `pr_template_policy.py` swaps the release-please exemptions for the release PR's own; `TestGsdTagCreationIsDisabled` passes unchanged, and a new assertion that the App-token step is the only `git tag`/`git push --tags` in `.github/workflows/` goes RED against a planted second tag push — no guard is left red when the phase closes (GRD-15, GRD-16)
  4. A contributor reading `CONTRIBUTING.md` §Pull requests learns to add a fragment per user-visible change, never to hand-write `.changes/v*.md`, how to exempt, and how the version is derived; `docs/RELEASE-PROCEDURES.md` describes fragment → release PR → merge → App-token tag → locked `release.yml`, places `pretag-gate` in the new workflow, and states that a release is cut by merging the release PR, with each of its 45 release-please references resolved one by one under a recorded verdict; `pr-title.yml`'s header says it is hygiene only, and `Taskfile.yml`'s `release:dry-run-signed` comment and `docs/RELEASE.md`'s pipeline paragraph name the changie workflow (DOCS-12, DOCS-13, DOCS-14)
  5. As the phase's last change, `release-please.yml`, `release-please-config.json` and `.release-please-manifest.json` are deleted, `changie latest` is the version source of truth, and a positive-controlled census — first shown to find a planted reference — reports zero `release-please` references outside `.planning/`, `CHANGELOG.md` history and this milestone's own notes (REL-14)

**Notes**: Two token hazards shape the workflow, and both are why the App token exists: a tag pushed with `GITHUB_TOKEN` never fires `release.yml`, and a pull request opened or updated with `GITHUB_TOKEN` fires no `pull_request` workflows — so the release PR's required checks would never report and it could never merge. Which token authors the release PR is a discuss-phase decision that must answer the second hazard, not only the first. The workflow must also not loop: the release PR's own merge push carries a new `.changes/vX.Y.Z.md` and an empty `.changes/unreleased/`, so the tag path fires and the batch path does not. GitHub runs the workflow file from the pushed commit, so the milestone merge commit — which deletes `release-please.yml` — runs the new workflow and not the old one; that merge is Phase 6's entry. The census scope already excludes `.planning/`, so the Overview's versioning note and STATE.md's standing decisions naming release-please are updated at milestone close, not counted here. `release-please` references in `ci.yml`, `require-issue-link.yml`, `post-release-verify.yml`, `release.yml` comments and `.goreleaser.yaml` are part of the census and each gets a verdict. Before merge, criteria 1–2 are proven by shape tests, actionlint and the dry-run; the live proof is Phase 6.
**Plans**: TBD

Plans:

- [ ] TBD (run /gsd-plan-phase 5 to break down)

#### Phase 6: First Release Through the Chain

**Goal**: A real release ships through the new chain end to end — the workflow opens the release PR from this milestone's fragments, the maintainer merges it, the GitHub App pushes the tag, the locked `release.yml` builds, signs and attests with the changie file as the Release body, and an installed v0.14.0 upgrades to it with verification. This is the milestone's definition of done.
**Depends on**: Phases 1–5, and the milestone PR merged to `main` (its merge push is what fires the release workflow).
**Requirements**: SHIP-01, SHIP-02, SHIP-03, SHIP-04
**Success Criteria** (what must be TRUE):

  1. **Checkpoint (human):** after the milestone PR merges, the workflow opens one `chore(main): release vX.Y.Z` PR from this milestone's fragments; the maintainer reviews the rendered notes and version and merges it; `changie latest` on `main` then answers the derived version (predicted `v0.15.0`) (SHIP-01)
  2. **Checkpoint (human):** the resulting tag is authored by the GitHub App — not by a human and not by `GITHUB_TOKEN` — `release.yml` ran on that tag push, and the GitHub Release body equals `.changes/v0.15.0.md` byte-for-byte (SHIP-02)
  3. **Checkpoint (human):** `post-release-verify` is green against the published assets, and `codegraph upgrade` from an installed v0.14.0 verifies the new artifact against the unchanged cosign identity on darwin/arm64 and on linux/amd64 (SHIP-03)
  4. SEED-005 is annotated with the verified trigger (this release) so the other repositories can pick up the template, and the capability tag installed by `CAP-04` is recorded as the one this release ran with (SHIP-04)

**Notes**: This phase executes after the milestone PR has merged, so its own planning artifacts cannot ride on that PR — where they land (a planning-only follow-up PR, which the fragment gate passes without a marker) and when the milestone audit and close run relative to this phase are sequencing decisions to settle at discuss time. No human `git tag`, no `[ci skip]`, no re-run of a failed step to make a check green: a defect found here is fixed forward through a `Fixes` fragment and the chain, never by hand-tagging. The version label is a prediction — criterion 2 names `v0.15.0` only because that is what the fragments are expected to derive; the file compared is whatever `.changes/vX.Y.Z.md` the release actually produced.
**Plans**: TBD

Plans:

- [ ] TBD (run /gsd-plan-phase 6 to break down)

## Progress

**Execution Order:**
Phases execute in numeric order: 1 → 2 → 3 → 4 → 5 → 6. The genuine dependencies are few: every phase needs Phase 1's `.changie.yaml`; Phase 3's close needs Phase 2's capability installed (`CAP-05`); Phase 5's docs and its `pr_template_policy.py` edit follow Phase 4's gate; Phase 6 needs all five phases and the milestone PR merged to `main`. Phase 4 is otherwise independent of Phases 2–3 and could run beside them, but is kept after Phase 3 so the capability's first firing lands on a close carrying a real user-visible change. `REL-14` is Phase 5's last change because its zero-reference census cannot pass until `DOCS-12…14` are done. Phase numbering restarted at 1 by `--reset-phase-numbers`; `.planning/phases/` held only the three `999.x` backlog directories at the start. Backlog below is preserved across milestone closes; `999.5` is promoted into Phase 3.

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 1. Changie Baseline | 2/2 | Complete    | 2026-09-25 |
| 2. Phase-Close Fragment Capability | 4/5 | In Progress|  |
| 3. Rename-Stub Removal | 0/TBD | Not started | - |
| 4. Fragment-Required Gate | 0/TBD | Not started | - |
| 5. Release Chain & release-please Retirement | 0/TBD | Not started | - |
| 6. First Release Through the Chain | 0/TBD | Not started | - |

9 milestones shipped (v0.1, v1.0, v0.3.0, v0.5.0, v0.10.0, v0.11.0, v0.12.0, v0.13.0, v0.14.0). v0.15.0 scoped: 6 phases (1–6), 29 requirements, 0/6 phases complete (0%).

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

**PROMOTED 2026-09-25 → v0.15.0 Phase 3 (Rename-Stub Removal), scoped as VERB-09.** Kept here rather than deleted, for the same reason as 999.2 above. The captured goal below stands as written and is the direct source of VERB-09's deletion list and its zero-hit census.

**Goal:** Remove the hidden `query` and `unlock` rename stubs registered by Phase 3 (VERB-03/VERB-04, D-05/D-09; feat commit 5d69ee2ea3c6276ed73c946b24be93612fae1698) in v0.15.0, the minor after the fold: delete `internal/cli/renamed.go` and `internal/cli/renamed_test.go`, drop `newQueryCmd()` and `newUnlockCmd()` from `internal/cli/root.go`'s AddCommand list, remove the two `codegraph query` / `codegraph unlock` lines from `internal/cli/testdata/cli-reference-allowlist.txt`, run `task docs:cli` (no reference change expected — the stubs are hidden), and re-run the Phase 3 census instrument (03-MUTATION-LOG.md Family (c)) expecting zero hits.
**Requirements**: TBD
**Depends on:** Phase TBD
**Plans:** 0 plans

Plans:

- [ ] TBD (run /gsd-plan-phase 999.5 to break down)
