# Roadmap: CodeGraph Go

## Overview

CodeGraph Go is a ground-up Go rewrite of TypeScript CodeGraph — a drop-in, TS-v1.3.x-parity replacement in a single static binary. **v0.1 (Initial Release) shipped 2026-07-14**: the core capabilities (indexing, query, MCP server, sync, migration) work from a signed/attested/SBOM'd release that beats TS 1.3.1 on every measured benchmark — but the CLI/agent surface still diverged *behaviorally* from TS. **v1.0 (Drop-in Parity & Human UX) shipped 2026-08-03**: those gaps are closed. An existing user can now swap binaries with zero change in experience — TS-identical `explore`/`node`/`status` behavior, watcher-on-MCP by default, git/worktree awareness, output hygiene, a human-facing Charm TUI behind a build-enforced rendering seam (the agent/MCP path never sees ANSI), systematic flag reconciliation, fully automated signed releases via release-please + GoReleaser, and contributor-facing local build tooling. **v0.3.0 (MCP Protocol Currency) shipped 2026-08-06**: the stdio MCP server is current with spec revision `2026-07-28` on `modelcontextprotocol/go-sdk@v1.7.0`, proven by a wire-level oracle that never imports the SDK it tests, with every Legacy client still working.

**v0.5.0 (macOS Distribution & Homebrew) shipped 2026-08-11.** The binary is now *installable by convention* on macOS. `spctl -a -vv -t exec` returns **accepted** on both darwin arches, and `brew tap seanb4t/tap && brew install codegraph` works on a clean machine. The release pipeline moved onto a single `goreleaser release` invocation — whose `notarize:` and `homebrew_casks:` blocks never execute under the `goreleaser build --single-target` matrix the pipeline had always used — with both Linux legs `zig cc` cross-compiled from one macOS runner. That cross-compilation was the milestone's single unproven claim; it was spiked first, blocking, and **passed on variation V1 at first dispatch**, so the costed GoReleaser Pro fallback was never bought. `codegraph upgrade` now detects a Homebrew-managed install and steps aside rather than mutating a Caskroom the package manager owns.

**v0.10.0 (Agent Onboarding Skill & MCP Resources) is in progress.** Every prior milestone made the tools better; none of them made an agent *use* the tools. The binary is fast, correct, signed, and installable — and an agent with codegraph configured still reaches for grep first, because the only thing the server ever told it about itself was a marker block and a wire-level `instructions` string that defers full tool guidance to "the MCP initialize response (Phase 3)" — a hand-off that was never built. That stale promise actively misdirected a real debug session on 2026-08-08, which is the incident this milestone exists to close. The fix is three surfaces that reinforce each other: the server serves its own reference documentation over **MCP Resources**, a **SKILL.md** teaches the decision procedure (*which question goes to which tool*) and points at those resources instead of restating them, and a **SessionStart nudge** makes codegraph's availability visible at the moment it matters. `codegraph install` then ships the skill package versioned with the binary, and only then is the `instructions` string rewritten — last, so it can name things that already exist.

**Versioning note:** "v1.0" is a *planning-milestone* name, never a release version. The shipped artifact line reached `v0.2.0` at v1.0's close and has since advanced through `v0.3.0`, `v0.4.0` and — across v0.5.0 — `v0.5.0` … `v0.9.0`, each computed by release-please from Conventional Commits; there is deliberately no `v1.0.0` tag (maintainer directive D-06R, 2026-07-29). Milestone labels track the release line but carry **no git tag**: release-please remains the sole tag authority, pinned by `TestGsdTagCreationIsDisabled`. A hand-created `v*` tag would additionally match `release.yml`'s `push: tags: "v[0-9]*"` trigger and falsely fire the release pipeline. The milestone record lives in `MILESTONES.md` + `milestones/`. (`milestone-v0.1` exists only because it predates release-please.)

## Milestones

- ✅ **v0.1 — Initial Release** — Phases 1–8 (shipped 2026-07-14) — core capabilities + signed release; not yet a drop-in parity replacement
- ✅ **v1.0 — Drop-in Parity & Human UX** — Phases 1–10 (shipped 2026-08-03) — behavioral + surface parity with TS 1.3.1, human TUI, automated signed releases, local build tooling
- ✅ **v0.3.0 — MCP Protocol Currency** — Phases 1–5 (shipped 2026-08-06) — official Go SDK adoption, `2026-07-28` spec compliance without breaking Legacy clients, a wire-level verification oracle, tool-modfile vulnerability coverage
- ✅ **v0.5.0 — macOS Distribution & Homebrew** — Phases 1–4 (shipped 2026-08-11) — `goreleaser release` migration with zig cross-compilation, Apple notarization, a Homebrew tap and cask, and an `upgrade` that steps aside under brew. Promoted backlog 999.5, consumed SEED-002. Full detail: [`milestones/v0.5.0-ROADMAP.md`](./milestones/v0.5.0-ROADMAP.md)
- 🚧 **v0.10.0 — Agent Onboarding Skill & MCP Resources** — Phases 5–8 (in progress) — the server documents itself over MCP Resources, a decision-procedure-first SKILL.md plus a SessionStart nudge teach agents when to reach for it, `codegraph install` ships that package with the binary, and the stale `instructions` promise is retired last. Consumes the 2026-08-08 skill todo
- 📋 **Later** — unscoped. Candidates: this milestone's own v2 deferrals (the PreToolUse guard hook GUARD-HOOK-01/02, multi-agent skill+hooks porting AGENT-04…07), the Backlog items below (999.2 tmux TTY harness, 999.4 CheckRegression guard), the v0.5.x deferrals (DIST-06 stapled offline-safe container, BREW-07 homebrew-core), Team Scale (central server, CI-distributed indexes), MRTR/elicitation (MRTR-01), annotations (embeddings/communities/export), local Svelte web UI (SEED-001)

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

### 🚧 v0.10.0 — Agent Onboarding Skill & MCP Resources (In Progress)

**Milestone Goal:** Give agent harnesses a thin, high-signal skill that teaches WHEN and HOW to use codegraph's tools — leading with a decision procedure, not a tool catalog — while pushing detailed reference content into MCP Resources served by the server itself, backed by a soft SessionStart nudge, so an agent stops reaching for grep first.

**Phase numbering continues from v0.5.0** (which ended at Phase 4), so this milestone runs Phases 5–8. That is a deliberate departure from the restart-at-1 convention v0.5.0 used; the sequence is contiguous, and phase directories under `.planning/phases/` sort unambiguously.

**Ordering is load-bearing, not stylistic:**

- **Resources ship before anything points at them (Phase 5).** The SKILL.md's whole design is *point, don't restate* — it names resource URIs instead of embedding tool reference content. A skill naming a URI the server does not serve is the same class of defect as the "Phase 3" promise this milestone was created to retire, just relocated.
- **The drift guard ships in the same phase as the resources, not as a follow-up.** Ungated reference content is worse than none: it is confidently wrong, invisible to the reader, and re-earns SURF-01. Research is explicit on this, and this repo's own history is the evidence — the guard is not a hardening pass, it is half of what makes the resources shippable.
- **Wire-oracle re-capture is folded into Phase 5, not carved out.** Turning on `capabilities.resources` changes the `initialize` result on the wire, so the frozen transcripts go red the moment the capability is live. Re-freezing them is not a separate deliverable with its own user-observable outcome — it is the regression discipline attached to a single change, per `MUTATION-PROOF.md`'s one-deliberate-unit rule. Splitting it would leave main red between two phases.
- **Phase 6 authors both agent-facing artifacts together** — SKILL.md and the SessionStart nudge are one package (`hooks.json` + script + skill), authored and verified as one, and installed as one in Phase 7. The nudge alone is two requirements and a file-existence check; as its own phase it would be a task, not an outcome.
- **Phase 7 is the highest-risk phase in the milestone, and it is not the first.** Writing new artifact types into agent-owned directories is exactly the shape that produced v1.0 Phase 5's two reproduced data-loss Criticals and Phase 6's swallowed-I/O-error findings. It runs after the content it installs has stabilized, so it is never shipping a moving target, and it reuses `internal/fsatomic` + the marker-fence discipline rather than inventing new file-safety primitives.
- **The `instructions` rewrite is genuinely last — after install, not merely after resources (Phase 8).** SUMMARY.md sequenced the rewrite ahead of install distribution; this roadmap deliberately inverts that. WIRE-02 governs the marker block *`codegraph install` writes*, so if the rewrite landed first, that block would name a skill install did not yet place — a fresh broken promise, of precisely the kind being retired, authored inside the milestone that exists to prevent it. Sequenced last, every capability either string names is resolvable at test time.
- **Deferred, and deliberately so:** the PreToolUse guard hook (GUARD-HOOK-01/02) and multi-agent porting (AGENT-04…07) are v2. The guard adds friction and false-positive surface; it is the fallback if skill + resources + nudge prove insufficient, and that evidence does not exist yet. Multi-agent hooks are blocked on per-agent schema differences (Cursor camelCase, Codex/Antigravity different event sets) that are not yet verified — the SKILL.md itself is portable, the hooks are not.
- **`v0.10.0` carries no git tag.** release-please is the sole tag authority (D-06R), and a `v0.10.0` tag would match `release.yml`'s `push: tags: "v[0-9]*"` trigger and falsely fire the release pipeline.

- [ ] **Phase 5: MCP Resources Capability & Claims Drift Guard** - The server documents itself: `resources/list` + `resources/read` serve tool-by-tool reference over the wire in any repo, indexed or not, with every stated fact derived from source or pinned by a test that can fire
- [ ] **Phase 6: Agent Skill Package — SKILL.md & SessionStart Nudge** - An agent with the skill installed picks `codegraph_explore` over grep on a where-is-X prompt, shown by transcript diff, and learns codegraph exists at session start in an indexed repo — and only there
- [ ] **Phase 7: `codegraph install` Skill + Hooks Distribution (Claude Code)** - Installing codegraph installs the skill package too, versioned with the binary and refreshed by `upgrade`; uninstall removes exactly what it wrote and nothing a user authored
- [ ] **Phase 8: Instructions & Marker-Block Rewrite** - Everything codegraph tells an agent about itself is true on the day it is read — the "Phase 3" deferral is gone, and each surface names only capabilities that already shipped

## Phase Details

### Phase 5: MCP Resources Capability & Claims Drift Guard

**Goal**: An agent connected to `codegraph serve --mcp` can ask the server itself how its tools work — in any repository, whether or not an index exists — and no fact in that reference can drift away from the binary without a test going red.
**Depends on**: Nothing (first phase of v0.10.0)
**Requirements**: RSRC-01, RSRC-02, RSRC-03, GUARD-01, GUARD-02
**Success Criteria** (what must be TRUE):

  1. A live client against a real `serve --mcp` subprocess sees `capabilities.resources` in the `initialize` result, receives from `resources/list` one entry for each of the 8 tools plus `CODEGRAPH_MCP_TOOLS` semantics and index-state preconditions, and gets non-empty `text/markdown` back from `resources/read` on every URI that list advertised — observed on the wire by the SDK-independent oracle, never by calling the server's own Go API as its own witness (RSRC-01, RSRC-02)
  2. In a directory with no `.codegraph/`, where `tools/list` returns an empty set, `resources/list` still returns the full catalog and `resources/read` still serves content — so an agent in an unindexed repository can learn what codegraph is and that `codegraph init` is the next step (RSRC-03)
  3. Adding, removing, or renaming a tool turns a test red until the resource content matches — demonstrated by applying that mutation, watching it fail, and reverting byte-clean, rather than asserted from reading the guard (GUARD-02)
  4. Every tool count, default value, and env var name appearing in the resource docs is derived from a source constant or pinned by a test: mutating a stated default in the code (or in the prose) turns the guard red instead of shipping a document that contradicts the binary it describes (GUARD-01)
  5. The frozen wire transcripts are re-captured in the same change that makes `capabilities.resources` live, every changed transcript's diff attributed to a named cause with a count, and the oracle re-proved non-vacuous — main is green at the phase boundary, with no re-freeze deferred to a later phase

**Notes**: `Server.AddResource` is stable in the pinned `modelcontextprotocol/go-sdk@v1.7.0`; no new module dependency is needed anywhere in this milestone. Resource content is `go:embed`'d markdown under `internal/mcp/`, and the drift guard extends the existing `tools_schema_drift_test.go` / `instructions_contract_test.go` patterns rather than introducing a new mechanism. Resource `subscribe`/`listChanged` is explicitly out of scope — the tool roster is static per-process today.

**Plans**: 4 plans

Plans:
**Wave 1**

- [ ] 05-01-PLAN.md — Tracer: `codegraph://tools/explore` registered, listed, read and wire-observed end-to-end; 25 frozen transcripts re-captured under one named cause

**Wave 2** *(blocked on Wave 1 completion)*

- [ ] 05-02-PLAN.md — The remaining 9 resource files: 7 per-tool fact-sheets plus the `CODEGRAPH_MCP_TOOLS` filter and index-state behavior docs

**Wave 3** *(blocked on Wave 2 completion)*

- [ ] 05-03-PLAN.md — Claims drift guard: bidirectional tool/file set equality and numeric, count, env-var and host-fact claim pinning, each demonstrated RED

**Wave 4** *(blocked on Wave 3 completion)*

- [ ] 05-04-PLAN.md — Wire fan-out: a read scenario per advertised URI, the unindexed catalog, the unknown-URI error shape, and the oracle re-proved non-vacuous

### Phase 6: Agent Skill Package — SKILL.md & SessionStart Nudge

**Goal**: An agent that has the codegraph skill installed answers a "where is X" question by calling codegraph instead of grepping — and, in an indexed repository, is told codegraph is available at the moment a session starts.
**Depends on**: Phase 5 to ship (the skill names resource URIs that must resolve against a running server). Authoring can proceed in parallel with Phase 5 — the packages are disjoint — but this phase does not close until those URIs are live.
**Requirements**: SKILL-01, SKILL-02, SKILL-03, NUDGE-01, NUDGE-02
**Success Criteria** (what must be TRUE):

  1. Given a fresh session, the skill installed, and a "where is X" prompt, the agent's first code-search action is `codegraph_explore` (or `codegraph explore`) rather than grep/find/Read — shown by a recorded transcript diff against the same prompt without the skill, not asserted from the skill's own text (SKILL-03)
  2. SKILL.md opens with a decision table mapping question shapes to tools *before* any per-tool catalog, stays within the ~500-line / <5k-token budget, and defers detail to the Phase-5 resource URIs instead of restating it — so the skill stays a decision procedure rather than becoming a second, drifting copy of the reference (SKILL-01)
  3. The skill carries 2–3 worked examples, one of which reproduces the 2026-08-08 misdirection incident end to end: the question that was asked, the grep path that misled the session, and the codegraph call that answers it correctly (SKILL-02)
  4. Starting a session in a `.codegraph/`-indexed repository produces one short nudge toward codegraph's tools, decided by a file-existence check alone — no MCP round-trip, no index read, and no repeat later in the same session (NUDGE-01)
  5. Starting a session in a repository with no `.codegraph/` produces no nudge output whatsoever and does no filesystem work beyond that single existence check — established by executing the shipped hook script in both trees and diffing what it emitted, not by reading it (NUDGE-02)

**Notes**: The SKILL.md is portable across Claude Code / Cursor / Codex CLI / opencode with zero changes (shared `agentskills.io` frontmatter); the `hooks.json` is not — schemas diverge per agent, which is why distribution is Claude-Code-only in v1 (Phase 7) and porting is v2 (AGENT-04…07). Frontmatter `description` is the only field always in context, so it must be trigger-shaped ("use when the user asks …"), not a summary. Every factual claim in the skill body is subject to the Phase-5 GUARD-01 discipline — the guard covers skill, resources, and instructions alike, not resources alone.

**Plans**: TBD

Plans:

- [ ] TBD (run `/gsd-plan-phase 6`)

### Phase 7: `codegraph install` Skill + Hooks Distribution (Claude Code)

**Goal**: Installing codegraph installs its skill package too — carried inside the binary, so it moves with `codegraph upgrade` — and uninstalling takes back exactly what install wrote, leaving user-authored content byte-untouched.
**Depends on**: Phase 6 (the artifacts being installed must be final; shipping a moving target through install is what makes install bugs expensive) and Phase 5 (the skill's resource URIs must resolve)
**Requirements**: AGENT-01, AGENT-02, AGENT-03
**Success Criteria** (what must be TRUE):

  1. `codegraph install` on a Claude Code setup places SKILL.md and `hooks.json` in the locations Claude Code actually reads, and a second `install` produces byte-identical files — idempotent by measurement, not by intent (AGENT-01)
  2. `codegraph uninstall` after `install` returns the tree to its pre-install bytes, including a `hooks.json` that already carried unrelated user-authored entries before codegraph ever touched it — the round-trip is byte-invariant against a fixture that has such entries, not only against an empty tree (AGENT-02)
  3. The installed package is the running binary's own embedded copy: after `codegraph upgrade` (or a re-install from a newer binary), the on-disk skill is the newer content, and which version is installed is observable from the installed files rather than inferred (AGENT-03)
  4. A read error or unparseable existing `hooks.json` makes install and uninstall surface the error instead of overwriting or deleting content they could not read and parse — the invariant v1.0 Phase 5 converged on after two reproduced data-loss Criticals, re-proven here for this new artifact type across the {install, uninstall} × {read-error, malformed} matrix

**Notes**: Highest-risk phase of the milestone; a deep-review pass is warranted. Reuse `internal/fsatomic` and the existing `AgentTarget` / `recordFile` machinery — do not invent new file-safety primitives. Scope is Claude Code only: per-agent `hooks.json` schemas differ (Cursor camelCase, Codex CLI and Antigravity carry different event sets, Antigravity has no `SessionStart`/`UserPromptSubmit` at all), and porting is deferred to v2 as AGENT-04…07 rather than guessed at here.

**Plans**: TBD

Plans:

- [ ] TBD (run `/gsd-plan-phase 7`)

### Phase 8: Instructions & Marker-Block Rewrite

**Goal**: Every statement codegraph makes about itself — on the MCP wire and in the marker block it writes into agent instruction files — is true on the day an agent reads it, and names only capabilities that have already shipped.
**Depends on**: Phase 7 (the marker block describes what that same `install` invocation actually did), Phase 6 (the skill it names), Phase 5 (the resource URIs it names). This phase is last by requirement, not by convenience — WIRE-03 makes "everything named already exists" the acceptance condition.
**Requirements**: WIRE-01, WIRE-02, WIRE-03
**Success Criteria** (what must be TRUE):

  1. The `instructions` string a client receives from `initialize` (and `server/discover`) points at the skill and the resource URIs the running server actually serves, and contains no forward-looking promise — the "defers full tool guidance to the MCP initialize response (Phase 3)" deferral is gone from `internal/agents/instructions.go`, along with the behavior of promising a hand-off nothing delivers (WIRE-01)
  2. The marker block `codegraph install` writes matches the rewritten instructions and describes only what that same install did; a mutation making it name an unshipped capability turns a test red rather than shipping (WIRE-02)
  3. Every capability either string names is resolvable at test time — each named resource URI returns content from a live `resources/read`, and the named skill path is one `codegraph install` actually writes — so a promise can no longer outlive the thing it promised (WIRE-03)
  4. The frozen wire transcripts carry the new instructions string byte-identically to what a live client receives, re-captured in this same change with the diff attributed — no transcript is left describing the retired text

**Notes**: Closes the pending todo `2026-08-08-author-a-codegraph-usage-skill-for-agents.md` and the never-completed hand-off at `internal/agents/instructions.go:17-18`. The marker fences (`<!-- CODEGRAPH_START/END -->`) are a byte-exact cross-implementation contract with TS CodeGraph and must not change — only the content between them. `instructions_contract_test.go` gains a fourth anchor.

**Plans**: TBD

Plans:

- [ ] TBD (run `/gsd-plan-phase 8`)

## Progress

**Execution Order:**
Phases execute in numeric order: 5 → 6 → 7 → 8

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 5. MCP Resources Capability & Claims Drift Guard | 0/TBD | Not started | - |
| 6. Agent Skill Package — SKILL.md & SessionStart Nudge | 0/TBD | Not started | - |
| 7. `codegraph install` Skill + Hooks Distribution | 0/TBD | Not started | - |
| 8. Instructions & Marker-Block Rewrite | 0/TBD | Not started | - |

4 milestones shipped. v0.10.0 scoped: 4 phases, 16 requirements, 0/4 phases complete (0%).

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
