# Roadmap: CodeGraph Go

## Overview

CodeGraph Go is a Go implementation of a pre-indexed code knowledge graph for coding agents. **v0.1 (Initial Release) shipped 2026-07-14**: the core capabilities (indexing, query, MCP server, sync, migration) work from a signed/attested/SBOM'd release. **v1.0 (Drop-in Parity & Human UX) shipped 2026-08-03**: behavioral gaps closed — `explore`/`node`/`status` behavior, watcher-on-MCP by default, git/worktree awareness, output hygiene, a human-facing Charm TUI behind a build-enforced rendering seam (the agent/MCP path never sees ANSI), systematic flag reconciliation, fully automated signed releases via release-please + GoReleaser, and contributor-facing local build tooling. **v0.3.0 (MCP Protocol Currency) shipped 2026-08-06**: the stdio MCP server is current with spec revision `2026-07-28` on `modelcontextprotocol/go-sdk@v1.7.0`, proven by a wire-level oracle that never imports the SDK it tests, with every Legacy client still working.

**v0.5.0 (macOS Distribution & Homebrew) shipped 2026-08-11.** The binary is now *installable by convention* on macOS. `spctl -a -vv -t exec` returns **accepted** on both darwin arches, and `brew tap seanb4t/tap && brew install codegraph` works on a clean machine. The release pipeline moved onto a single `goreleaser release` invocation — whose `notarize:` and `homebrew_casks:` blocks never execute under the `goreleaser build --single-target` matrix the pipeline had always used — with both Linux legs `zig cc` cross-compiled from one macOS runner. That cross-compilation was the milestone's single unproven claim; it was spiked first, blocking, and **passed on variation V1 at first dispatch**, so the costed GoReleaser Pro fallback was never bought. `codegraph upgrade` now detects a Homebrew-managed install and steps aside rather than mutating a Caskroom the package manager owns.

**v0.10.0 (Agent Onboarding Skill & MCP Resources) shipped 2026-08-13.** Every prior milestone made the tools better; none of them made an agent *use* the tools. The binary was fast, correct, signed, and installable — and an agent with codegraph configured still reached for grep first, because the only thing the server ever told it about itself was a marker block and a wire-level `instructions` string that deferred full tool guidance to "the MCP initialize response (Phase 3)" — a hand-off that was never built. That stale promise actively misdirected a real debug session on 2026-08-08, which is the incident this milestone existed to close. The fix landed as three surfaces that reinforce each other: the server serves its own reference documentation over **MCP Resources**, a **SKILL.md** teaches the decision procedure (*which question goes to which tool*) and points at those resources instead of restating them, and a **SessionStart nudge** makes codegraph's availability visible at the moment it matters. `codegraph install` ships the skill package versioned with the binary, and the `instructions` string was rewritten last, so it names only things that already exist.

**v0.11.0 (Standalone Project Identity) is in progress.** Everything the project *does* now stands on its own; the way it *describes and tests itself* still does not. Its docs, templates, workflows, comments, test identifiers, benchmark tables and golden corpora are all framed against another implementation — a framing that made sense while the goal was to replace that implementation and has outlived it. This milestone retires the framing without retiring any capability: one legally-sufficient acknowledgment in `NOTICE`, one past-tense clause in README's License section, and comparison vocabulary removed from every doc, template, workflow, script, code comment, identifier and test fixture. The load-bearing part is not prose — it is the golden suite, which currently derives its oracle from corpora chosen because the origin project used them. Re-basing it means selecting new corpora *by measurement* and re-proving the re-baselined suite can still fail.

**Versioning note:** "v1.0" is a *planning-milestone* name, never a release version. The shipped artifact line reached `v0.2.0` at v1.0's close and has since advanced through `v0.3.0`, `v0.4.0` and — across v0.5.0 — `v0.5.0` … `v0.9.0`, plus `v0.10.0`, each computed by release-please from Conventional Commits; there is deliberately no `v1.0.0` tag (maintainer directive D-06R, 2026-07-29). Milestone labels track the release line but carry **no git tag**: release-please remains the sole tag authority, pinned by `TestGsdTagCreationIsDisabled`. A hand-created `v*` tag would additionally match `release.yml`'s `push: tags: "v[0-9]*"` trigger and falsely fire the release pipeline. The milestone record lives in `MILESTONES.md` + `milestones/`. (`milestone-v0.1` exists only because it predates release-please.)

## Milestones

- ✅ **v0.1 — Initial Release** — Phases 1–8 (shipped 2026-07-14) — core capabilities + signed release
- ✅ **v1.0 — Drop-in Parity & Human UX** — Phases 1–10 (shipped 2026-08-03) — behavioral + surface parity with TS 1.3.1, human TUI, automated signed releases, local build tooling
- ✅ **v0.3.0 — MCP Protocol Currency** — Phases 1–5 (shipped 2026-08-06) — official Go SDK adoption, `2026-07-28` spec compliance without breaking Legacy clients, a wire-level verification oracle, tool-modfile vulnerability coverage
- ✅ **v0.5.0 — macOS Distribution & Homebrew** — Phases 1–4 (shipped 2026-08-11) — `goreleaser release` migration with zig cross-compilation, Apple notarization, a Homebrew tap and cask, and an `upgrade` that steps aside under brew. Promoted backlog 999.5, consumed SEED-002. Full detail: [`milestones/v0.5.0-ROADMAP.md`](./milestones/v0.5.0-ROADMAP.md)
- ✅ **v0.10.0 — Agent Onboarding Skill & MCP Resources** — Phases 5–8 (shipped 2026-08-13) — the server documents itself over MCP Resources, a decision-procedure-first SKILL.md plus a SessionStart nudge teach agents when to reach for it, `codegraph install` ships that package with the binary, and the stale `instructions` promise was retired last. Consumed the 2026-08-08 skill todo. Full detail: [`milestones/v0.10.0-ROADMAP.md`](./milestones/v0.10.0-ROADMAP.md) · requirements: [`milestones/v0.10.0-REQUIREMENTS.md`](./milestones/v0.10.0-REQUIREMENTS.md)
- 🚧 **v0.11.0 — Standalone Project Identity** — Phases 1–6 (in progress) — origin acknowledged once in `NOTICE` and one README License clause; comparison framing removed from docs, templates, workflows, comments, identifiers and fixtures; golden corpora re-selected by measurement and re-frozen from codegraph-go's own output; benchmarks published as absolute numbers
- 📋 **Later** — unscoped. Candidates: v0.10.0's own v2 deferrals (the PreToolUse guard hook GUARD-HOOK-01/02, multi-agent skill+hooks porting AGENT-04…07), v0.11.0's v2 deferrals (DOCS-05 self-authored CLI reference, VOCAB-01 vocabulary drift guard), the Backlog items below (999.2 tmux TTY harness, 999.4 CheckRegression guard), the v0.5.x deferrals (DIST-06 stapled offline-safe container, BREW-07 homebrew-core), Team Scale (central server, CI-distributed indexes), MRTR/elicitation (MRTR-01), annotations (embeddings/communities/export), local Svelte web UI (SEED-001)

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

### 🚧 v0.11.0 — Standalone Project Identity (In Progress)

**Milestone Goal:** Make codegraph-go read, test, and benchmark as a project in its own right — one legally-sufficient acknowledgment in `NOTICE` plus a single clause in README's License section, with *parity*, *upstream*, *drop-in* and origin-derived framing removed from every doc, template, workflow, script, code comment, identifier, and test fixture.

**Phase numbering restarts at 1.** v0.10.0 continued v0.5.0's sequence (Phases 5–8); this milestone restarts, matching the convention v0.5.0 itself used. `.planning/phases/` holds only backlog directories (`999.x`) at the milestone boundary, so Phases 1–6 sort unambiguously and collide with nothing.

**Ordering is load-bearing, not stylistic:**

- **Phase 1 is a blocking measurement spike, and nothing that re-freezes a golden may be planned before it resolves.** Corpus selection decides what the golden suite is *capable of* catching. The trap this spike exists to catch is already documented in this repo's own history: v1.0 Phase 1 found that codegraph-go's own idiomatic Go source legitimately produces **zero** `overrides` and `type_of` edges. A corpus set can therefore under-cover the 9-kind `RANK_EDGES` vocabulary while every test stays green — a silently narrowed oracle, which is the exact failure mode a re-baseline invites. The set is locked by recorded per-kind edge counts and per-language file counts from real indexing runs, not by reputation. The shortlist to measure is not yet locked: gohugoio/hugo (Apache-2.0, Go), nestjs/nest (MIT, TypeScript), google/guava (Apache-2.0, Java), apache/arrow (Apache-2.0, mixed monorepo incl. C#).
- **The rename pass and the re-freeze pass are separate reviewed diffs (Phase 2).** `CODE-02` renames `parity_*_test.go` / `TestGoldenParity*`; `FIXT-06` re-freezes every golden from codegraph-go's own output. Doing both in one diff makes any resulting regression un-attributable — was it the rename, or the new corpus? The repo's established discipline is one reviewed-diff pass with every changed transcript attributable to a single named cause (v0.3.0 Phase 3's consolidated fourth pass; v0.10.0 Phase 8's 38-transcript re-freeze). The identifier pass changes no golden byte; the re-freeze pass changes no identifier.
- **Non-vacuity proof is its own phase, never folded into the re-freeze (Phase 3).** The standing repo rule is that a gate is not trusted until demonstrated RED against a confirmed-applied, byte-cleanly-reverted mutation. A re-baseline that also authors its own proof is the same author certifying their own oracle in the same breath. `FIXT-07` runs after `FIXT-06` and in a different phase so the proof is applied to a suite that already exists in its final form.
- **`FIXT-03`'s "no self-skips" criterion needs a positive assertion, not a passing exit.** A negative-only guard passes vacuously — this repo already carries that class of defect twice (rule `84d1gfpywd`; the `dry-run-signed` additions-only diff guard). The job must report how many golden scenarios it executed and assert that count, so a suite that ran zero scenarios fails instead of reporting success.
- **The attribution, documentation and process sweeps are independent of the fixture work (Phases 4 and 5).** They touch disjoint files and carry no dependency on corpus selection, so they can run in parallel with Phases 1–3. Phase 5's in-tree comment sweep is sequenced after the fixture work only so a comment change and a golden change never share a diff.
- **The sweep removes framing, never capability — with one recorded exception.** `codegraph migrate` is **dropped entirely** (maintainer ruling 2026-08-15): the command, the `internal/migrate` package, and the `modernc.org/sqlite` sole-use dependency are removed. The migration path was itself parity framing (a drop-in-competitor posture), inconsistent with the milestone's standalone-identity goal. Amended deliberately, not dropped silently; the archived build phase that produced it stays in history. `internal/indexer/tsextract`, the language registry and the capability matrix are product surface — TypeScript-the-indexed-language, not TypeScript-the-origin-project. Every retained use is resolved term-by-term with a recorded reason, never by regex.
- **The memory sweep is last, and is verified by inspection rather than by a test (Phase 6).** The engram spine lives outside git, so no CI gate can hold it. It should describe the end state, which makes it most accurate after the code and documentation sweeps have landed. `MEM-01` supersedes rather than overwrites: nothing that records real history is deleted.
- **Deliberately declined, and recorded so the decision stays visible:** a build-time vocabulary drift guard (`VOCAB-01`). A term blocklist either goes vacuous or fights legitimate uses like `tsextract`. One-time sweep plus review discipline is the chosen posture. Also deferred: `DOCS-05`, a self-authored `docs/CLI-REFERENCE.md` — this milestone deletes the comparison matrix; authoring its replacement is separate work.
- **Out of scope by construction:** `.planning/` archives and `CHANGELOG.md`. The first is an append-only record of what actually happened and is parsed by scope-sensitive tooling; the second is release-please-owned and hand-editing it breaks the tool that both writes and re-reads it.
- **`v0.11.0` carries no git tag.** release-please is the sole tag authority (D-06R). The label is a prediction that holds if this milestone lands `feat:` commits; a fixes-only outcome cuts a patch version instead. No phase schedules a `git tag` step.

- [x] **Phase 1: Corpus Selection by Measurement** - The project knows from recorded measurement, not assumption, which third-party repositories at which commits its golden suite exercises — and can fetch that tree reproducibly without vendoring it (completed 2026-08-14)
- [x] **Phase 2: Golden Harness Re-authoring & Re-freeze** - The golden suite reads as codegraph-go's own regression suite: named for what it asserts, frozen from codegraph-go's own output, with the origin-driving capture path gone (completed 2026-08-14)
- [x] **Phase 3: Non-Vacuity Proof & Unconditional CI Execution** - The re-baselined suite is trusted because it has been watched fail, and CI cannot silently stop running it (completed 2026-08-15)
- [x] **Phase 4: Attribution & Documentation Sweep** - A reader finds the origin acknowledged once, legally and in the past tense, and meets no comparison framing anywhere else in the project's documentation (completed 2026-08-15)
- [ ] **Phase 5: Process, CI & In-Tree Sweep** - A contributor filing an issue or opening a PR, and an agent reading the source, meet a project described on its own terms — with `tsextract` intact and `codegraph migrate` removed (maintainer ruling 2026-08-15)
- [ ] **Phase 6: Benchmark De-coupling & Memory Sweep** - The project publishes its own absolute performance numbers with no second implementation in the picture, and a fresh session recalls it as a project rather than a port

## Phase Details

### Phase 1: Corpus Selection by Measurement

**Goal**: The project knows, from recorded measurement rather than assumption, exactly which third-party repositories at which commits its golden suite will exercise — and can fetch that tree reproducibly without vendoring it.
**Depends on**: Nothing (first phase). **Blocking**: no phase that re-freezes a golden may be planned until this one resolves.
**Requirements**: FIXT-01, FIXT-02
**Success Criteria** (what must be TRUE):

  1. A committed measurement record names every candidate repository that was actually indexed and reports its per-kind edge counts and per-language file counts — real indexing output, not estimates — and names the locked final set (FIXT-01)
  2. Across the locked set, each of the 9 `RANK_EDGES` kinds (`calls`, `implements`, `imports`, `extends`, `overrides`, `references`, `instantiates`, `returns`, `type_of`) has a non-zero measured count attributed to a named repository, and each of the 5 priority-4 languages (Go, Java, C#, Python, TS/JS) has a non-zero measured file count (FIXT-01)
  3. `overrides` and `type_of` — the two kinds this repository's own idiomatic Go source produces **zero** of — are each covered by a named repository in the locked set, cited by measured count rather than by expectation (FIXT-01)
  4. One Taskfile target fetches the locked corpora at pinned commit SHAs; running it twice from clean produces the same tree, and no corpus source is committed to this repository (FIXT-02)
  5. A CI run restores the corpora from cache; a cache miss falls through to a fetch at the pinned SHAs rather than to a skip (FIXT-02)

**Notes**: Every candidate must be MIT or Apache-2.0. Fetch-at-pinned-SHA was chosen over vendoring specifically to avoid adding redistribution obligations to the `NOTICE` file this milestone is trimming. The shortlist to measure — gohugoio/hugo, nestjs/nest, google/guava, apache/arrow — is a starting point, not a decision; the spike may keep, drop, or add to it based on what it measures.

**Plans**: 7 plans

Plans:
**Wave 1**

- [x] 01-01-PLAN.md — Tracer: `edgesByKind` tally and un-suppressed `filesByLanguage` on the `status --json` contract
- [x] 01-04-PLAN.md — Corpora manifest as sole pin authority, strict validation, atomic out-of-tree fetch

**Wave 2** *(blocked on Wave 1 completion)*

- [x] 01-02-PLAN.md — Dense `--all-kinds` opt-in derived from `RankEdges`, across all four status render surfaces

**Wave 3** *(blocked on Wave 2 completion)*

- [x] 01-03-PLAN.md — Re-freeze `call-status.golden`; assert MCP sparsity and `--json` key presence
- [x] 01-05-PLAN.md — Measurement record pipeline: typed record, in-process measure mode, generated prose

**Wave 4** *(blocked on Wave 3 completion)*

- [x] 01-06-PLAN.md — The spike: score candidates, freeze the per-kind threshold, lock the corpus set

**Wave 5** *(blocked on Wave 4 completion)*

- [x] 01-07-PLAN.md — Coverage drift guard and CI wiring: per-corpus cache, miss falls through to a real fetch

### Phase 2: Golden Harness Re-authoring & Re-freeze

**Goal**: The golden suite reads as codegraph-go's own behavioral regression suite — its files, tests and fixtures named for what they assert, its goldens frozen from codegraph-go's own output against the locked corpora, and the origin-driving capture path gone.
**Depends on**: Phase 1 (the locked corpora and the fetch target are what the goldens are frozen against)
**Requirements**: CODE-02, FIXT-04, FIXT-05, FIXT-06
**Success Criteria** (what must be TRUE):

  1. No file, test function or fixture directory under `testdata/golden/` carries comparison framing in its name — `parity_*_test.go` and `TestGoldenParity*` are renamed or removed — and `go test ./...` plus `go test ./testdata/golden/...` both pass (CODE-02)
  2. The rename and the re-freeze land as separate reviewed diffs, each with every changed line attributable to one named cause: the rename pass changes no golden byte, and the re-freeze pass changes no identifier (CODE-02, FIXT-06)
  3. `capture.sh`, `mcp-capture.mjs`, and the `weft-go` and `colbymchenry-codegraph` corpora and their captured fixtures are absent from the tree, with nothing left referencing them (FIXT-04)
  4. Every targeted behavioral case the purpose-built corpus encodes — overloaded same-named symbols, multi-word queries, the `Test*`-heavy weakly-connected cluster, structural-beats-lexical ranking — is still exercised by a named test under `corpus/behavioral/`, with its case map intact (FIXT-05)
  5. Every golden under `testdata/golden/` was produced by the Go-side capture path (`testdata/golden/gocapture`) from codegraph-go's own output against the locked corpora (FIXT-06)

**Notes**: The `synthetic-parity` corpus is the one that must *survive* the rename, not be replaced — it is purpose-built and encodes deliberate behavioral cases that no third-party repository reproduces. Losing a case to the rename is the failure mode FIXT-05 exists to prevent, so the case map is the acceptance artifact, not the directory name. Both `explore` and `node`, on both the CLI and MCP surfaces, are in the re-freeze.

**Plans**: 4 plans

Plans:
**Diff A — rename (changes no golden byte)**
**Wave 1**

- [x] 02-01-PLAN.md — Rename golden harness identifiers to behavioral vocabulary, carrying the matrix/doc-mirror gates in one atomic diff (CODE-02 tracer)

**Wave 2** *(blocked on Wave 1)*

- [x] 02-02-PLAN.md — Delete TS-era capture path + weft/colbymchenry corpora, move behavioral corpus to corpus/behavioral with CASES.json, re-author synthetic test to D-09 property assertions (FIXT-04, FIXT-05)

**Diff B — re-freeze (changes no identifier)**

**Wave 3** *(blocked on Wave 2)*

- [x] 02-03-PLAN.md — Extend gocapture (locked-corpus + behavioral specs, temp-then-move, CLI+MCP), hermetic fail-loud locked-corpus resolution, golden:regen target, byte-identity guard (FIXT-06)

**Wave 4** *(blocked on Wave 3)*

- [x] 02-04-PLAN.md — Run the re-freeze capture, review single-cause attribution + zero identifier change, commit the reviewed diff (FIXT-06)

**Goal**: The re-baselined golden suite is trusted because it has been watched fail, and CI cannot silently stop running it.
**Depends on**: Phase 2. This phase is deliberately separate from the re-freeze — a re-baseline that authors its own proof in the same change certifies its own oracle.
**Requirements**: FIXT-03, FIXT-07
**Success Criteria** (what must be TRUE):

  1. Each assertion family in the re-baselined suite has been demonstrated RED against a confirmed-applied mutation that was afterwards reverted byte-clean, with the mutation, the observed failure and the revert recorded per family (FIXT-07)
  2. A CI run reports the number of golden scenarios it executed and asserts that count against an expected value — a run that executed zero scenarios fails the job rather than passing it (FIXT-03)
  3. A corpus fetch failure or cache failure fails the CI job loudly; no golden test in the suite can self-skip in CI (FIXT-03)
  4. Removing a locked corpus from the CI environment turns the job red — demonstrated by doing it, not argued from reading the guard (FIXT-03)

**Notes**: The scenario-count assertion is the positive half FIXT-03 requires; a non-failing exit is not evidence the suite ran. This repo already carries two negative-only guards that pass vacuously (rule `84d1gfpywd`; the `dry-run-signed` additions-only diff guard), and the wire-oracle's `ExpectedScenarioCount` is the working precedent for the shape that does not.

**Plans**: 2 plans

Plans:
**Wave 1**

- [ ] 03-01-PLAN.md — Tracer: executed-scenario-count self-assertion (26 goldens + 4 CASES cases = 30, exact equality) + unconditional `golden` job in corpora.yml with widened path filter, inScopeJobs entry, -count=1, and the ci.yml D-04 removal (FIXT-03)

**Wave 2** *(blocked on Wave 1)*

- [ ] 03-02-PLAN.md — FIXT-07 mutation rehearsals: all five assertion families demonstrated RED (mutation applied → observed failure → byte-clean revert) and recorded in 03-MUTATION-LOG.md (FIXT-07)

### Phase 4: Attribution & Documentation Sweep

**Goal**: A reader of this project's own documentation finds the origin acknowledged exactly once — legally, and in the past tense — and encounters no comparison framing anywhere else.
**Depends on**: Nothing. Independent of the fixture work (Phases 1–3) and may run in parallel with it.
**Requirements**: ATTR-01, ATTR-02, ATTR-03, DOCS-01, DOCS-02, DOCS-03, DOCS-04
**Success Criteria** (what must be TRUE):

  1. `NOTICE` contains the MIT copyright transcription plus one sentence of origin and nothing else — the drop-in / ported-heuristics / flag-parity argument it currently makes is gone (ATTR-01)
  2. README has no `## Relationship to the original` section, and its only origin mention is one past-tense clause inside `## License` that links to `NOTICE` (ATTR-02)
  3. `LICENSE` is still verbatim MIT text and `gh api repos/seanb4t/codegraph-go/license` returns `MIT` — run live against the repository after the change, not assumed (ATTR-03)
  4. A reader of `README.md`, `CONTRIBUTING.md`, `SECURITY.md`, `CODE_OF_CONDUCT.md`, `PARSER-DECISION.md`, `docs/RELEASE.md`, `docs/RELEASE-PROCEDURES.md`, `docs/MCP-2026-07-28-SCOPING.md` or `docs/MCP-8-AGENT-AUDIT.md` encounters no comparison framing, and `docs/LANGUAGE-CAPABILITY-MATRIX.md` states per-language capability on codegraph-go's own terms without reference to another implementation's coverage (DOCS-01, DOCS-03, DOCS-04)
  5. `docs/FLAG-PARITY.md` and `internal/cli/flag_parity_test.go` are both deleted, `go test ./...` still passes, and nothing in the tree references either (DOCS-02)

**Notes**: Deleting `flag_parity_test.go` removes a live drift guard. Its replacement (`DOCS-05`, a self-authored CLI reference with its own guard) is deliberately deferred to a later milestone — this is a knowing, recorded reduction in coverage, not an oversight. `docs/BENCHMARKS.md` belongs to Phase 6, not here, because its rewrite is coupled to removing the comparison runner.

**Plans**: 3 plans

Plans:
**Wave 1** *(parallel — zero file overlap)*

- [x] 04-01-PLAN.md — NOTICE attribution trim (tracer) + README Relationship removal / License clause / Performance + Status sweep; live license check + LICENSE byte-identity (ATTR-01, ATTR-02, ATTR-03)
- [x] 04-03-PLAN.md — CONTRIBUTING + capability matrix reword; verify-only sweep of SECURITY/CoC/PARSER-DECISION and remaining docs/* with KEEP reasons recorded (DOCS-01, DOCS-03, DOCS-04)

**Wave 2** *(blocked on Wave 1 — the phase-final gate re-asserts the other plans' end-state)*

- [x] 04-02-PLAN.md — Delete docs/FLAG-PARITY.md + internal/cli/flag_parity_test.go and sweep all reference sites incl. the man.go:27 census correction (DOCS-02) + the executable phase-final gate (13/13 positive count, cycle-1 review finding 3)

### Phase 5: Process, CI & In-Tree Sweep

**Goal**: A contributor filing an issue or opening a PR, and an agent reading the source, meet a project described on its own terms — with TypeScript-the-indexed-language intact and `codegraph migrate` removed (maintainer ruling 2026-08-15).
**Depends on**: Phase 3, so that no in-tree comment change shares a diff with a golden change. `PROC-01…03` carries no fixture dependency and may start earlier.
**Requirements**: PROC-01, PROC-02, PROC-03, CODE-01, CODE-03
**Success Criteria** (what must be TRUE):

  1. A contributor filing an issue sees no comparison framing in any of the 5 issue templates (`bug_report`, `chore`, `config`, `enhancement`, `feature_request`) (PROC-01)
  2. A contributor opening a PR sees none in `pull_request_template.md` or any of the 3 `PULL_REQUEST_TEMPLATE/*` variants (PROC-02)
  3. Every workflow still passes `actionlint` and its own required status checks after the sweep, with job names, step names and comments carrying no retired framing (PROC-03)
  4. Comments across `internal/`, `tools/` and `test/` carry no comparison framing, while `internal/indexer/tsextract`, the language registry and the capability matrix are preserved as product surface — every retained use resolved term-by-term with a recorded reason, never by regex (CODE-01)
  5. `codegraph migrate` is removed entirely (maintainer ruling 2026-08-15): the command, the `internal/migrate` package, its fixture, and the `modernc.org/sqlite` sole-use dependency are gone, with nothing left referencing it. **The project no longer carries a TS→Go index migration path** — a TS user installs the Go binary and re-indexes from source (CODE-03 amended)

**Notes**: Workflows carrying the vocabulary include `ci`, `release`, `bench`, `post-release-verify`, `linux-cross-canary`, `require-issue-link` and `auto-close-unsolicited-prs`. `bench.yml`'s own de-coupling is Phase 6's; this phase touches only its framing. The naive-regex failure mode is real and named in the requirements: a find-and-replace over "TypeScript" breaks `tsextract` and de-lists a supported language.

**Plans**: TBD

### Phase 6: Benchmark De-coupling & Memory Sweep

**Goal**: The project publishes its own absolute performance numbers with no second implementation in the picture, and an agent that starts a session afterward recalls codegraph-go as a project rather than as a port.
**Depends on**: Phase 4 and Phase 5 — the memory sweep describes the end state, so it is most accurate once the documentation and code sweeps have landed.
**Requirements**: BENCH-01, BENCH-02, BENCH-03, MEM-01, MEM-02
**Success Criteria** (what must be TRUE):

  1. `docs/BENCHMARKS.md` publishes absolute throughput, query-latency and peak-RSS figures with their methodology and the regression-gate description, and carries no head-to-head comparison table or multipliers (BENCH-01)
  2. `tools/bench` contains no comparison runner, and `internal/bench.CheckRegression` against the committed `baseline.json` still fires on a real regression — demonstrated RED against a confirmed-applied, byte-cleanly-reverted mutation, not asserted from reading the gate (BENCH-02)
  3. A `bench.yml` run publishes the absolute numbers without invoking another implementation anywhere in the job (BENCH-03)
  4. Every engram spine record for `repo:github.com/seanb4t/codegraph-go` whose durable content asserts retired framing has been superseded by a corrected record — none overwritten, none deleted, and no record of real history removed (MEM-01)
  5. A session started after the sweep recalls no memory describing codegraph-go as a port or parity project in the present tense (MEM-02)

**Notes**: MEM-01/02 are verified by inspection, not by a test — the spine lives outside git and no CI gate can hold it. "Supersede, don't overwrite" is the point: the historical records are true statements about what happened and must survive; only records asserting the framing *in the present tense* are corrected. The head-to-head JSON captures already committed under `tools/bench/headtohead-*.json` are historical measurement records; whether they are removed or retained as dated artifacts is a scoping call for the phase plan, but `docs/BENCHMARKS.md` must not publish their multipliers either way.

**Plans**: TBD

## Progress

**Execution Order:**
Phases execute in numeric order: 1 → 2 → 3 → 4 → 5 → 6. Phase 4 has no dependency on Phases 1–3 and may run in parallel with them.

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 1. Corpus Selection by Measurement | 7/7 | Complete    | 2026-08-14 |
| 2. Golden Harness Re-authoring & Re-freeze | 4/4 | Complete    | 2026-08-14 |
| 3. Non-Vacuity Proof & Unconditional CI Execution | 2/2 | Complete    | 2026-08-15 |
| 4. Attribution & Documentation Sweep | 3/3 | Complete    | 2026-08-15 |
| 5. Process, CI & In-Tree Sweep | 0/? | Not started | - |
| 6. Benchmark De-coupling & Memory Sweep | 0/? | Not started | - |

5 milestones shipped. v0.11.0 scoped: 6 phases, 25 requirements, 0/6 phases complete (0%).

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
