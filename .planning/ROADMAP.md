# Roadmap: CodeGraph Go

## Overview

CodeGraph Go is a ground-up Go rewrite of TypeScript CodeGraph — a drop-in, TS-v1.3.x-parity replacement in a single static binary. **v0.1 (Initial Release) shipped 2026-07-14**: the core capabilities (indexing, query, MCP server, sync, migration) work from a signed/attested/SBOM'd release that beats TS 1.3.1 on every measured benchmark — but the CLI/agent surface still diverges *behaviorally* from TS. **v1.0 (Drop-in Parity & Human UX)** closes those gaps so an existing user can swap binaries with zero change in experience: TS-identical `explore`/`node`/`status` behavior, watcher-on-MCP by default, git/worktree awareness, output hygiene, a human-facing Charm TUI (agent path stays plain), systematic flag reconciliation — then the first real signed `v1.0.0`. Work is risk-front-loaded: the load-bearing shared-engine behavioral algorithms land first (Phase 1), the human TUI is added last behind a build-enforced rendering seam so the agent/MCP path never sees ANSI.

## Milestones

- ✅ **v0.1 — Initial Release** — Phases 1–8 (shipped 2026-07-14) — core capabilities + signed release; not yet a drop-in parity replacement
- 🚧 **v1.0 — Drop-in Parity & Human UX** — Phases 1–8 (in progress) — behavioral + surface parity with TS 1.3.1, human TUI, first signed `v1.0.0`
- 📋 **Later** — Team Scale (central server, CI-distributed indexes), annotations (embeddings/communities/export), local Svelte web UI (SEED-001)

## Phases

<details>
<summary>✅ v0.1 — Initial Release (Phases 1–8) — SHIPPED 2026-07-14</summary>

Full phase details archived in [`milestones/v0.1-ROADMAP.md`](milestones/v0.1-ROADMAP.md); phase artifacts in [`milestones/v0.1-phases/`](milestones/v0.1-phases/).

- [x] Phase 1: Foundation — Storage, Schema & Parser Strategy (7/7 plans) — completed 2026-07-10
- [x] Phase 2: Go Indexing Pipeline (6/6 plans) — completed 2026-07-11
- [x] Phase 3: Query Engine & MCP Server (9/9 plans) — completed 2026-07-11
- [x] Phase 4: Incremental Sync & File Watcher (9/9 plans) — completed 2026-07-11
- [x] Phase 5: Language Coverage & Resolution Breadth (14/13 plans) — completed 2026-07-12
- [x] Phase 6: Agent Integrations & CLI Lifecycle (6/6 plans) — completed 2026-07-12
- [x] Phase 7: Migration Tool (7/7 plans) — completed 2026-07-13
- [x] Phase 8: Release Hardening & Benchmarks (9/9 plans) — completed 2026-07-14

**Delivered:** the core capabilities of CodeGraph in a single static Go binary — index/query/MCP/sync/migrate — faster and lighter than TS 1.3.1, from a signed/attested/SBOM'd release verified end-to-end on `v0.0.0-rc.3`. **Known gaps carried into v1.0:** the CLI/agent surface diverges *behaviorally* from TS; DIST-02 (real `v*` tag) and PERF-01 (published numbers) remained pending.

</details>

### 🚧 v1.0 — Drop-in Parity & Human UX (In Progress)

**Milestone Goal:** Close the behavioral + surface gaps against TS CodeGraph v1.3.1 so an existing user swaps binaries with zero change in experience, add a human-facing terminal UI (agent/MCP path stays plain), then cut the first signed `v1.0.0`. Phase numbering resets to 1 for this milestone (v0.1 archived).

- [x] **Phase 1: Behavioral Parity — explore & node** - TS-identical relevance-ranked exploration + full multi-definition disambiguation, validated by a behavioral fixture harness (completed 2026-07-15)
- [ ] **Phase 2: status Content & Git/Worktree Awareness** - Rich `status` content + borrowed-index detection across every read tool
- [ ] **Phase 3: Watcher-on-MCP Default** - `serve --mcp` runs live auto-sync by default with a WSL2/slow-FS escape hatch
- [ ] **Phase 4: Output Hygiene** - Silence Pebble WAL noise; keep MCP stdout clean JSON-RPC
- [ ] **Phase 5: Git Sync Hooks** - Marker-fenced, idempotent post-commit/merge/checkout sync hooks as the watcher-disabled fallback
- [ ] **Phase 6: Rendering Seam & Pretty status/files** - Build-enforced ANSI isolation + lipgloss-styled `status`/`files` (plain when piped)
- [ ] **Phase 7: Interactive TUI — Daemon Picker & Install Multi-Select** - bubbletea daemon picker, explicit lifecycle, install multi-select; never hangs when piped
- [ ] **Phase 8: Surface Reconciliation & Signed v1.0.0 Release** - Flag parity + Charm-dep audit + first signed `v1.0.0`, retiring the "not yet drop-in" caveat

## Phase Details

### Phase 1: Behavioral Parity — explore & node

**Goal**: Agents and users get TS-identical `explore` and `node` results — graph-relevance-ranked exploration and full multi-definition disambiguation — proven equivalent on both the CLI and the `codegraph_explore` MCP surface by a behavioral fixture harness that lands with the algorithm.
**Depends on**: Nothing (first phase of v1.0)
**Requirements**: EXPL-01, EXPL-02, EXPL-03, EXPL-04, EXPL-05, NODE-01, NODE-02, NODE-03, NODE-04, TEST-01
**Success Criteria** (what must be TRUE):

  1. `explore` accepts a multi-word `<query...>` (tokenized CamelCase/snake_case/acronym/dot-notation/plain, stopword-filtered) and ranks results by graph relevance (Random-Walk-with-Restart, α=0.25, ~25 iters, 9 edge kinds) with a file-level relevance gate, so structurally-connected symbols outrank lexical name-matches and weakly-connected `Test*` funcs no longer top results (EXPL-01/02/03)
  2. `explore` emits a per-root "⚠️ no covering tests" warning when a symbol has direct callers but no covering test files, matching TS (EXPL-04)
  3. `node` enumerates ALL exact-name definitions of an overloaded symbol (generated-files-last) with the "N definitions named X — returning M in full" header, full bodies to TS's budget (≤16 defs / 12,000 chars), and an overflow list of the rest; optional file/line narrowing never empties the set (NODE-01/02/03)
  4. `explore` and `node` output is byte-identical across the CLI command and the MCP tool (shared engine), and single-definition `node` stays byte-comparable to TS (EXPL-05/NODE-04)
  5. A behavioral fixture harness diffs `explore`/`node` against TS 1.3.1 for ambiguous names, multi-word queries, relevance ordering, and coverage warnings on BOTH the CLI and MCP surfaces — closing v0.1's single-symbol golden blind spot (TEST-01)

**Plans**: 17/17 plans complete

Plans:
**Wave 1**

- [x] 01-01-PLAN.md — Behavioral fixture harness + TS 1.3.1 golden capture (CLI + MCP, synthetic corpus)
- [x] 01-02-PLAN.md — F1: 6 new edge-kind constants (additive string, no schema bump)
- [x] 01-03-PLAN.md — Query tokenizers (H1 extractSymbolsFromQuery + H2 extractSearchTerms/STOP_WORDS)
- [x] 01-04-PLAN.md — node multi-def enumeration + budget/overflow + never-empty narrowing (NODE-01/02/03/04)

**Wave 2** *(blocked on Wave 1 completion)*

- [x] 01-05-PLAN.md — Go edge-kind extraction: resolve.go Pass-2 (extends/overrides) + Pass-1 (references/instantiates/type_of/returns)
- [x] 01-06-PLAN.md — RWR core: computeGraphRelevance + 9-kind RankEdges (deterministic, α=0.25, 25 iters)
- [x] 01-07-PLAN.md — Hybrid gather channels 1-3 + merge (H3–H6) + shared isTestFile

**Wave 3** *(blocked on Wave 2 completion)*

- [x] 01-08-PLAN.md — Java + C# extractor edge-kind emission
- [x] 01-09-PLAN.md — Python + TS/JS extractor edge-kind emission
- [x] 01-10-PLAN.md — Gather re-rankers: test-dampen + core-dir + multi-term (H7–H9)
- [x] 01-11-PLAN.md — Subgraph expansion: type-hierarchy + BFS bounds + glue-node (H10–H12)
- [x] 01-12-PLAN.md — Named-symbol seeding + per-overload disambiguation tiers (H13)
- [x] 01-13-PLAN.md — Per-file score tiers + hard exclusion + buried-rescue (H14–H16)

**Wave 4** *(blocked on Wave 3 completion)*

- [x] 01-14-PLAN.md — 5-way relevance gate + central-file + 5-tier sort (H17–H19, EXPL-03 core)
- [x] 01-15-PLAN.md — F4: force re-index repo + golden corpora (9-kind graph)

**Wave 5** *(blocked on Wave 4 completion)*

- [x] 01-16-PLAN.md — Explore orchestration + EXPL-04 warning + skeletonization/adaptive budget (H20/H21) + CLI variadic

**Wave 6** *(blocked on Wave 5 completion)*

- [x] 01-17-PLAN.md — F5 golden regen + behavioral parity harness (Go vs TS, CLI==MCP)

**Notes**: Highest-risk, load-bearing work. EXPL-02's RWR relevance algorithm is the single hardest item — it lives in the shared `internal/query.Engine` (CLI + MCP improve in the same commit) and puts the golden-corpus contract at stake. Fixtures MUST exist before/with the algorithm change (template-parity ≠ behavior-parity). No styling anywhere in this phase — plain-text output only (the archtest lands in Phase 6 but the constraint holds from day one).

### Phase 2: status Content & Git/Worktree Awareness

**Goal**: `status` reports the full TS-parity content (DB size, nodes-by-kind, files-by-language, live staleness), and every read tool — CLI and MCP — detects a borrowed worktree index and warns, closing the silent "worktree queries the main branch's graph" correctness bug. Every MCP read tool settles on one markdown output shape, so the worktree notice rides a single uniform mechanism.
**Depends on**: Phase 1
**Requirements**: STAT-01, STAT-02, STAT-03, WORK-01, WORK-02, WORK-03, TEST-02, SURF-06
**Success Criteria** (what must be TRUE):

  1. `status` reports Pebble on-disk DB size, nodes-by-kind and files-by-language breakdowns, and a live pending-changes / reindex-recommended signal instead of the Phase-3 inert placeholders (STAT-01/02/03)
  2. A query run from a git worktree whose resolved index belongs to a DIFFERENT working tree is detected (`git rev-parse --show-toplevel` vs `--git-common-dir`), computing the now-live `worktreeMismatch` (WORK-01)
  3. `status` prints a verbose borrowed-index warning, and every other read tool (CLI + MCP) prefixes a compact single-line notice via a shared `withWorktreeNotice` wrapper (WORK-02)
  4. Worktree detection is best-effort and never blocks queries — no false positive on submodules, nested clones, monorepo subdirs, non-git trees, or symlinked paths (EvalSymlinks both sides) (WORK-03)
  5. Worktree detection has passing fixtures for linked-worktree, submodule, nested-clone, monorepo-subdir, `.claude/worktrees/`, and symlinked layouts (TEST-02)
  6. The 5 JSON-shaped MCP read tools (`callers`/`callees`/`impact`/`search`/`files`) emit markdown like `explore`/`node` already do, so all 7 non-status read tools take the same text-prefix notice; MCP `status` also gains a markdown renderer (D-12's blockquote warning requires it — it emits JSON today); CLI `--json` still emits JSON on every command, and no `Marshal*JSON` helper body changes (SURF-06)

**Plans**: 5/7 plans executed

Plans:
**Wave 1**

- [x] 02-01-PLAN.md — `internal/gitmeta`: the 4-gate detection cascade, verbatim TS notice/warning strings, `CachingDetector`, and the six real-`git` fixture layouts (wave 1)
- [x] 02-02-PLAN.md — `StatusResult` gains `dbSizeBytes` + `filesByLanguage`; the golden `dbSizeBytes` strip is reversed Go-side as a documented divergence (wave 1)
- [x] 02-03-PLAN.md — the 5 SURF-06 markdown renderers, added as siblings of the untouched `Marshal*JSON` helpers (wave 1)

**Wave 2** *(blocked on Wave 1 completion)*

- [x] 02-04-PLAN.md — `OpenAt` retains `startPath`; `worktreeMismatch` goes live in TS's `{worktreeRoot, indexRoot}` shape; the two notice helpers (wave 2)
- [x] 02-05-PLAN.md — the two status renderers TS ships: CLI padded columns (D-09) and MCP bolded-key bullets (D-17), plus the hand-rolled comma grouper (wave 2)

**Wave 3** *(blocked on Wave 2 completion)*

- [ ] 02-06-PLAN.md — MCP wiring: six call sites to markdown, one server-scoped detector, the notice on 7 tools; closes the zero-coverage blind spot (wave 3)
- [ ] 02-07-PLAN.md — CLI wiring: the sectioned `status` layout replaces the terse one-liner; the notice on 7 read commands, human-output only (wave 3)

**Notes**: New `internal/gitmeta` package (stdlib `os/exec` only — two `git rev-parse` calls, no pure-Go git lib), consumed by `internal/query` so both CLI and MCP get worktree awareness in one commit. Validate the edge-case fixtures before any pretty rendering. SURF-06 was pulled in from Phase 8 (user decision, 2026-07-15): Phase 2 already rewires all 7 MCP read-tool result paths for WORK-02, so changing the output shape in the same pass avoids double-touching them and removes the "prefix text onto a JSON payload" problem. Still plain-text-only — markdown here means structure/wording, NOT color (Phase 6 owns TUI-02).

### Phase 3: Watcher-on-MCP Default

**Goal**: `serve --mcp` runs live in-process auto-sync by default (matching TS's auto-sync), with a `--no-watch` opt-out and a WSL2/slow-filesystem auto-off policy — restoring the live-sync experience with zero config change and without ever delaying the MCP handshake.
**Depends on**: Phase 2
**Requirements**: WATCH-01, WATCH-02, WATCH-03, WATCH-04
**Success Criteria** (what must be TRUE):

  1. `serve --mcp` runs the file watcher by default with `--no-watch` to opt out (flipping the current opt-in `--watch`); `install` already writes the byte-identical `serve --mcp` invocation, so live sync returns with no config change (WATCH-01)
  2. Watcher startup never delays the MCP handshake or first-tool availability — the watcher is started off the handshake path (WATCH-02)
  3. A WSL2 / slow-filesystem watch-policy auto-disables the watcher, honoring env precedence (`CODEGRAPH_NO_WATCH` / force-on), matching TS's escape hatch (WATCH-03)
  4. Concurrent `serve --mcp` sessions on one repo converge to a single writer (no double-watching), goleak-clean (WATCH-04)

**Plans**: TBD
**Notes**: The default flip is ~2 lines but MUST be bundled with the watch-policy port — a naive flip hangs MCP startup on WSL2. Reuses v0.1's `--watch` plumbing + daemon lockfile. This phase's watcher model is a prerequisite for the Phase 7 daemon picker.

### Phase 4: Output Hygiene

**Goal**: No library log noise ever pollutes command output or the MCP transport — Pebble's WAL/INFO chatter is routed away while real errors survive, and MCP stdout stays clean JSON-RPC.
**Depends on**: Phase 3
**Requirements**: HYG-01, HYG-02
**Success Criteria** (what must be TRUE):

  1. Pebble's internal WAL/INFO log noise no longer prints on any command (explicit `pebble.Options.Logger` routing INFO→discard), while real error diagnostics are preserved and still surface (HYG-01)
  2. No library log output ever reaches MCP stdout — JSON-RPC framing stays clean; all diagnostics go to stderr only (HYG-02)

**Plans**: TBD
**Notes**: Small and mechanical, but do NOT wholesale-silence — route INFO→discard explicitly so store-corruption errors are never hidden.

### Phase 5: Git Sync Hooks

**Goal**: Users can install marker-fenced git sync hooks that keep the index fresh when the watcher is disabled (WSL2 / `CODEGRAPH_NO_WATCH`), byte-invariantly and without ever blocking a commit.
**Depends on**: Phase 3
**Requirements**: HOOK-01, HOOK-02, HOOK-03
**Success Criteria** (what must be TRUE):

  1. `codegraph githooks install` writes marker-fenced `post-commit`/`post-merge`/`post-checkout` hooks that background-run `codegraph sync`, guarded by `command -v codegraph`, idempotent (replace-in-place), preserving any existing user hook content (HOOK-01)
  2. `codegraph githooks remove` strips only codegraph's marker block (preserving user content), and `githooks status` reports install state (HOOK-02)
  3. Hooks are surfaced as the fallback for when the watcher is disabled (WSL2 / `CODEGRAPH_NO_WATCH`), matching TS's narrower trigger — not an always-on feature (HOOK-03)

**Plans**: TBD
**Notes**: New `internal/githooks` package + `internal/fsatomic` extracted from `internal/agents/shared.go` (atomic-write / marker-fenced splice, shared with the agent installer). Background + silenced sync; `command -v` guard so it no-ops cleanly off-PATH. The formal byte-invariance/pipe-safety harness (TEST-03) lands in Phase 7 alongside the interactive TUI, since that is the first phase where both hooks and the new TUI components coexist.

### Phase 6: Rendering Seam & Pretty status/files

**Goal**: A build-enforced rendering seam permanently isolates all Charm styling from the agent/MCP path, and `status`/`files` render colorized, sectioned output on a TTY while staying byte-identical plain when piped.
**Depends on**: Phase 4
**Requirements**: TUI-01, TUI-02, TUI-05
**Success Criteria** (what must be TRUE):

  1. An import-graph archtest fails the build if `charm.land/lipgloss`/`bubbletea`/`bubbles` are reachable from `internal/query` or `internal/mcp` — the ANSI-isolation guarantee, mirroring the existing graphstore/migrate archtests (TUI-01)
  2. `status` and `files` render colorized, sectioned output on a TTY (lipgloss) and byte-identical plain output when piped or non-TTY (TUI-02)
  3. `init`/`index`/`sync` show progress feedback (spinner/progress) on a TTY, and plain output otherwise (TUI-05)
  4. The golden/MCP output path stays byte-identical — no ANSI ever reaches the agent surface (archtest green + golden-corpus unchanged)

**Plans**: TBD
**Notes**: Archtest FIRST (fails the build immediately if styling leaks into query/mcp), THEN add the `internal/cli/present` package + the Charm v2 deps. Charm v2 uses the `charm.land/...` vanity import (not `github.com/charmbracelet/...`); TTY-gating via `golang.org/x/term.IsTerminal`. This establishes the rendering seam the Phase 7 interactive components build on.
**UI hint**: yes

### Phase 7: Interactive TUI — Daemon Picker & Install Multi-Select

**Goal**: The `daemon` command becomes an interactive picker (resolving the TS name collision) backed by explicit start/stop lifecycle and a PPID watchdog, `install`/`uninstall` present a multi-select, and every interactive surface auto-falls back to non-interactive behavior when piped — never hanging.
**Depends on**: Phase 3 (watcher model), Phase 6 (rendering seam / TTY-gate)
**Requirements**: DMON-01, DMON-02, DMON-03, DMON-04, TUI-03, TUI-04, TEST-03
**Success Criteria** (what must be TRUE):

  1. `codegraph daemon` (no args) opens an interactive bubbletea picker listing running daemons (current project first) to stop one / stop-all / cancel, resolving the TS name collision (DMON-01, TUI-04)
  2. Explicit `daemon start` / `daemon stop` / `daemon stop --all` manage the shared background daemon lifecycle with no silent auto-spawn (`serve --mcp` already watches in-process per WATCH-01) (DMON-02)
  3. A PPID watchdog shuts down any daemon / in-process watcher when its supervising host or agent process dies (POSIX ppid-reparent + Windows liveness poll), and a global `~/.codegraph/daemons` registry lets the picker list/stop daemons across projects, self-healing stale records (DMON-03/04)
  4. `install`/`uninstall` present an interactive multi-select agent picker by default (bubbles), with `-y`/`--yes` for non-interactive auto/global (TUI-03)
  5. Every interactive component auto-falls back to non-interactive behavior when stdin/stdout is not a TTY (never hangs), and git-hook install→edit→remove is byte-invariant — both tested against piped streams (TUI-04, TEST-03)

**Plans**: TBD
**Notes**: bubbletea/bubbles interactive layer. Daemon = explicit start/stop + picker + PPID watchdog + registry — NO auto-spawn (the user chose this explicitly; the in-process watcher from Phase 3 already delivers live MCP sync). TTY-gate before `tea.NewProgram()`; test with piped streams so nothing ever hangs. TEST-03 lives here because it is the first phase where both hooks (Phase 5) and the new bubbletea components coexist, so its byte-invariance + piped-stream assertions can both be satisfied.
**UI hint**: yes

### Phase 8: Surface Reconciliation & Signed v1.0.0 Release

**Goal**: Every TS flag name and default is present or a documented divergence, then the new Charm dependency closure is audited and the first real signed `v1.0.0` is cut — retiring v0.1's "not yet drop-in" caveat and closing its pending DIST-02/PERF-01.
**Depends on**: Phase 7
**Requirements**: SURF-01, SURF-02, SURF-03, SURF-04, SURF-05, REL-01, REL-02, REL-03, REL-04
**Success Criteria** (what must be TRUE):

  1. `impact` default depth is 2, `files` gains a directory filter alongside the retained language `--filter`, missing short-flag aliases (`-l`/`-k`/`-j`/`-d`, etc.) are added across commands, and `affected` gains `--stdin`/`--depth`/`--filter <glob>`/`--quiet` for git-hook/CI scripting (SURF-01/02/03/04)
  2. A systematic per-command flag audit confirms every TS flag name + default is present or a documented divergence; `search` is retained as a documented Go-only extension and `migrate` is documented as an accepted divergence (SURF-05)
  3. The new Charm/TUI dependency closure is audited — no new CGo, `govulncheck` clean, SBOM regenerated, reproducible double-build still passes (REL-01)
  4. A real signed `v1.0.0` release is cut (per-binary cosign keyless + SLSA provenance + SBOM), closing v0.1's pending DIST-02, and head-to-head benchmarks vs TS 1.3.1 are re-run and published, closing v0.1's pending PERF-01 (REL-02/03)
  5. The "drop-in parity" claim is validated against the real TS CLI (behavioral fixtures + flag audit green) and PROJECT.md's "not yet drop-in" caveat is retired (REL-04)

**Plans**: TBD
**Notes**: Mechanical flag/default reconciliation first, then the Charm-dep supply-chain audit and the first real signed `v1.0.0` tag. REL-04 re-runs the Phase-1 TEST-01 behavioral harness + the SURF-05 flag audit green as the drop-in gate — it consumes the harness, it doesn't rebuild it.

## Progress

**Execution Order:** Phases execute in numeric order: 1 → 2 → 3 → 4 → 5 → 6 → 7 → 8

| Phase | Milestone | Plans Complete | Status | Completed |
|-------|-----------|----------------|--------|-----------|
| 1. Behavioral Parity — explore & node | v1.0 | 18/17 | Complete    | 2026-07-15 |
| 2. status Content & Git/Worktree Awareness | v1.0 | 5/7 | In Progress|  |
| 3. Watcher-on-MCP Default | v1.0 | 0/TBD | Not started | - |
| 4. Output Hygiene | v1.0 | 0/TBD | Not started | - |
| 5. Git Sync Hooks | v1.0 | 0/TBD | Not started | - |
| 6. Rendering Seam & Pretty status/files | v1.0 | 0/TBD | Not started | - |
| 7. Interactive TUI — Daemon Picker & Install Multi-Select | v1.0 | 0/TBD | Not started | - |
| 8. Surface Reconciliation & Signed v1.0.0 Release | v1.0 | 0/TBD | Not started | - |

## Backlog

### Phase 999.1: local build/contribution and taskfile.yml setup (BACKLOG)

**Goal:** [Captured for future planning] Contributor-facing local dev tooling — the repo currently has no `Makefile`/`Taskfile`/`scripts/`. Add a `Taskfile.yml` (go-task) wrapping the common local workflows (build with the release ldflags, test with the daemon-flake isolation, `-race`, `go vet`, `govulncheck`, `actionlint`, `goreleaser check`, the bench runner modes, cross-`GOOS` `go list -mod=readonly` pre-tag check) plus a `CONTRIBUTING.md` documenting the CGo toolchain prerequisites (zig for cross-builds) so a new contributor can build/test/lint from a clean checkout without reverse-engineering the CI workflows.
**Requirements:** TBD
**Plans:** 18/17 plans complete

Plans:

- [ ] TBD (promote with /gsd-review-backlog when ready)
