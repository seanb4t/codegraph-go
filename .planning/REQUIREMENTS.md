# Requirements: CodeGraph Go — Milestone v1.0 (Drop-in Parity & Human UX)

**Defined:** 2026-07-14
**Core Value:** An agent user can uninstall TypeScript CodeGraph, install the Go binary, migrate their indexes, and everything works the same or better — faster, from a single verifiably-built binary.

**Milestone goal:** Close the behavioral + surface gaps against TS CodeGraph v1.3.1 so an existing user swaps binaries with zero change in experience, add a human-facing terminal UI, then cut the first signed `v1.0.0`. Scope is evidence-based (a live dual-indexed bake-off + reverse-engineering the installed TS dist), not docs.

## v1.0 Requirements

### Behavioral Parity — explore (EXPL)

- [x] **EXPL-01**: User can run `explore` with a multi-word query (variadic `<query...>`), tokenized (CamelCase/snake_case/acronym/dot-notation/plain) with stopword filtering, matching TS (ours currently takes a single `<query>` and returns 0 on multi-word)
- [x] **EXPL-02**: `explore` ranks results by graph relevance (Random-Walk-with-Restart, α=0.25, ~25 iterations, TS's 9 edge kinds + type-hierarchy/BFS/glue-node expansion), not lexical name-match
- [x] **EXPL-03**: `explore` applies a file-level relevance gate so weakly-connected symbols (e.g. `Test*` funcs with no graph connectivity) no longer surface as top results, matching TS selection
- [x] **EXPL-04**: `explore` emits a per-root "⚠️ no covering tests" warning when a symbol has direct callers but no covering test files, matching TS
- [x] **EXPL-05**: `explore` output is identical across the CLI command and the `codegraph_explore` MCP tool (shared engine), verified against TS on behavioral fixtures

### Behavioral Parity — node (NODE)

- [x] **NODE-01**: `node` enumerates ALL exact-name definitions of an overloaded symbol (not just one), sorted generated-files-last, matching TS (ours currently returns a single definition)
- [x] **NODE-02**: `node` renders multiple definitions with a "N definitions named X — returning M in full" header, full bodies up to TS's budget (≤16 defs / 12,000 chars), and an overflow list of the rest
- [x] **NODE-03**: `node` accepts optional file/line narrowing that never empties the result set (best-effort hints)
- [x] **NODE-04**: single-definition `node` output stays byte-comparable to TS (CLI + MCP)

### Behavioral Parity — status content (STAT)

- [x] **STAT-01**: `status` reports DB size (Pebble on-disk bytes), reversing the Phase-3 golden-corpus strip
- [x] **STAT-02**: `status` reports nodes-by-kind and files-by-language breakdowns (data already computed in `StatusResult` — surface it)
- [x] **STAT-03**: `status` reports a live pending-changes / reindex-recommended state (the top-level `stale` signal), not the inert placeholder

### Watcher-on-MCP (WATCH)

- [x] **WATCH-01**: `serve --mcp` runs the file watcher by default (live in-process auto-sync) with `--no-watch` to opt out, matching TS's default-on behavior (ours is currently opt-in via `--watch`); `install` already writes the byte-identical `serve --mcp` invocation, so this restores live sync with zero config change
- [x] **WATCH-02**: watcher startup never delays the MCP handshake or first-tool availability (watcher started off the handshake path)
- [x] **WATCH-03**: a WSL2 / slow-filesystem watch-policy auto-disables the watcher (env precedence: `CODEGRAPH_NO_WATCH` / force-on), matching TS's escape hatch
- [x] **WATCH-04**: concurrent `serve --mcp` sessions on one repo converge to a single writer (no double-watching), goleak-clean

### Daemon Model (DMON)

- [x] **DMON-01**: `codegraph daemon` (no args) opens an interactive picker listing running daemons (current project first) to stop one / stop-all / cancel, resolving the TS name-collision (TS `daemon` = picker; ours was a foreground server)
- [x] **DMON-02**: explicit `daemon start` / `daemon stop` / `daemon stop --all` manage the shared background daemon lifecycle (no silent auto-spawn — `serve --mcp` watches in-process per WATCH-01, so a separate daemon is only for the explicit shared-writer case)
- [x] **DMON-03**: a PPID watchdog shuts down any daemon / in-process watcher when its supervising host or agent process dies (POSIX ppid-reparent + Windows liveness poll), preventing leaked daemons
- [x] **DMON-04**: a global daemon registry (`~/.codegraph/daemons`) lets the picker list/stop daemons across projects, self-healing stale records

### Git / Worktree Awareness (WORK)

- [x] **WORK-01**: a query run from a git worktree whose resolved index belongs to a DIFFERENT working tree is detected (`git rev-parse --show-toplevel` vs `--git-common-dir`), computing the now-live `worktreeMismatch` (fixes the silent "worktree queries the main branch's graph" bug)
- [x] **WORK-02**: `status` prints a verbose borrowed-index warning, and every other read tool (CLI + MCP) prefixes a compact single-line notice via a shared `withWorktreeNotice` wrapper
- [x] **WORK-03**: worktree detection is best-effort and never blocks queries — no false positive on submodules, nested clones, monorepo subdirs, non-git trees, or symlinked paths (EvalSymlinks both sides)

### Git Sync Hooks (HOOK)

- [x] **HOOK-01**: `codegraph githooks install` writes marker-fenced `post-commit`/`post-merge`/`post-checkout` hooks that background-run `codegraph sync`, guarded by `command -v codegraph`, idempotent (replace-in-place), preserving any user hook content
- [x] **HOOK-02**: `codegraph githooks remove` strips only codegraph's marker block (preserving user content); `githooks status` reports install state
- [x] **HOOK-03**: hooks are surfaced as the fallback for when the watcher is disabled (WSL2 / `CODEGRAPH_NO_WATCH`), matching TS's narrower trigger — not an always-on feature

### Output Hygiene (HYG)

- [x] **HYG-01**: Pebble's internal WAL/INFO log noise no longer prints on any command (explicit `pebble.Options.Logger` routing INFO→discard) while real errors are preserved
- [x] **HYG-02**: no library log output ever reaches MCP stdout — JSON-RPC framing stays clean; diagnostics go to stderr only

### Human TUI / Rendering (TUI)

- [x] **TUI-01**: an import-graph archtest fails the build if `charm.land/lipgloss`/`bubbletea`/`bubbles` are reachable from `internal/query` or `internal/mcp` (the ANSI-isolation guarantee; mirrors the existing graphstore/migrate archtests)
- [x] **TUI-02**: `status` and `files` render colorized, sectioned output on a TTY (lipgloss) and byte-identical plain output when piped or non-TTY
- [x] **TUI-03**: `install`/`uninstall` present an interactive multi-select agent picker by default (bubbles), with `-y`/`--yes` for non-interactive auto/global — matching TS (per decision this session)
- [x] **TUI-04**: the `daemon` picker (DMON-01) is a bubbletea UI, and every interactive component auto-falls back to non-interactive behavior when stdin/stdout is not a TTY (never hangs)
- [x] **TUI-05**: `init`/`index`/`sync` show progress feedback (spinner/progress) on a TTY, plain otherwise

### Surface Reconciliation (SURF)

- [x] **SURF-01**: `impact` default depth changes to 2 (from 5), matching TS
- [x] **SURF-02**: `files` gains a directory filter matching TS's `--filter` semantics as a NEW flag, while our existing language `--filter` is retained (documented intentional divergence — per decision this session: keep ours + add TS's)
- [x] **SURF-03**: missing short-flag aliases (`-l`, `-k`, `-j`, `-d`, etc.) are added across commands to match TS
- [x] **SURF-04**: `affected` gains `--stdin`, `--depth`, `--filter <glob>`, and `--quiet` for git-hook/CI scripting, matching TS
- [x] **SURF-05**: a systematic per-command flag audit confirms every TS flag name + default is present or a documented divergence; `search` retained as a documented Go-only extension; `migrate` documented as an accepted divergence
- [x] **SURF-06**: the 5 JSON-shaped MCP read tools (`callers`/`callees`/`impact`/`search`/`files`) return human/agent-readable markdown instead of raw `json.Marshal` output, matching TS (which returns markdown from every MCP tool) and the 2 already-markdown Go tools (`explore`/`node`) — closing a silent Go-vs-TS MCP surface divergence. The CLI `--json` flag keeps emitting JSON (a genuinely different consumer: jq/scripts/CI), and **no `Marshal*JSON` helper body is modified** — every one is shared with the CLI path, so the change is additive (sibling `Render*` funcs + the six `tools.go` call sites only). Measured on this repo's own index: `files` drops 28,505 → 17,471 bytes (**-38.7%**), the saving being ~11KB of JSON keys repeated once per record across 308 records. (Corrected 2026-07-15 during execution: the -41% / ~16,835 B figure cited at planning time was an estimate from a hand-built table; 17,471 B is the real renderer's measured output — plan 02-03.) (MCP `codegraph_status` is a 6th JSON tool and also gains a markdown renderer, but under WORK-02/STAT — D-12's blockquote warning requires it. See CONTEXT D-17.)

### Behavioral Parity Test Harness (TEST)

- [x] **TEST-01**: a behavioral fixture harness diffs `explore`/`node`/`status` against TS 1.3.1 for ambiguous names, multi-word queries, relevance ordering, and coverage warnings — on BOTH the CLI and MCP surfaces — closing v0.1's single-symbol golden blind spot
- [x] **TEST-02**: worktree detection has fixtures for linked-worktree, submodule, nested-clone, monorepo-subdir, `.claude/worktrees/` layout, and symlinked paths
- [x] **TEST-03**: git-hook install→edit→remove is byte-invariant, and interactive TUI components are tested against piped streams (never hang)
- [x] **TEST-04**: an integration harness **spawns the release binary as a subprocess** and drives its real transports end-to-end — the CLI via argv and the MCP server via a real stdio JSON-RPC session (`initialize` → `tools/call`) — asserting reachability that in-process/unit tests structurally cannot. Motivated by Phase-2 CR-01, where the MCP worktree notice was dead code in production for the whole phase yet every in-process `BuildServer`→`CallTool` test passed (in-process bypasses `serve.go`'s cwd/argv → path-derivation → handler wiring, the exact seam that broke). Anchor case: from inside a `.claude/worktrees/<name>/` borrowed worktree, a real `codegraph serve --mcp` session's `codegraph_explore` payload carries the `⚠` notice, with a main-checkout control run showing none. Wired into CI **alongside** `go test ./testdata/golden/...` (which `go test ./...` silently skips — the go tool ignores `testdata/`). Mechanizes the session-long "drive the real entry point, don't infer reachability" lesson. *(Mapped to Phase 3 because it first expands the `serve --mcp` surface; movable to Phase 8 to bundle with the REL-04 drop-in gate.)*

### v1.0.0 Release (REL)

- [ ] **REL-01**: the new Charm/TUI dependency closure is audited — no new CGo, `govulncheck` clean, SBOM regenerated, reproducible double-build still passes
- [ ] **REL-02**: a real signed `v1.0.0` release is cut (per-binary cosign keyless + SLSA provenance + SBOM), closing v0.1's pending DIST-02
- [ ] **REL-03**: head-to-head benchmarks vs TS 1.3.1 are re-run and published, closing v0.1's pending PERF-01
- [ ] **REL-04**: the "drop-in parity" claim is validated against the real TS CLI (behavioral fixtures + flag audit green) and PROJECT.md's "not yet drop-in" caveat is retired

## Future Requirements

Deferred to later releases. Tracked, not in this roadmap.

### Worktree (beyond parity)

- **WORK-FUT-01**: auto-`init` a borrowed-index worktree, or share the parent index via `git-common-dir`, instead of only warning (v1.0 ships TS-parity detect+warn+notice; "make worktree support better later" — user-confirmed)

### Daemon (beyond explicit model)

- **DMON-FUT-01**: full TS auto-spawn parity (detached per-project daemon + unix-socket sharing) if the in-process watcher model proves insufficient for the shared-across-many-agents case

### Human Web UI

- **UI-FUT-01**: local Svelte + shadcn-svelte web UI for browsing/querying the graph (SEED-001 — triggers once parity lands; 1.1; distinct from the v1.0 terminal TUI)

### Team Scale / Annotations (later milestones, per PROJECT.md)

- **TEAM-FUT-01**: central graph server (multi-user, remote queries, auth)
- **TEAM-FUT-02**: CI-built shared index distribution / caching
- **ANNO-FUT-01**: embedding vectors, community detection, bulk export for visualization (local-model-first)

## Out of Scope

Explicitly excluded from v1.0. Documented to prevent scope creep.

| Feature | Reason |
|---------|--------|
| Repointing `files --filter` to directory (breaking our language filter) | Per decision: keep language `--filter` + add a separate directory flag; do not break the existing capability |
| Auto-spawning daemons on `serve --mcp` | Explicit-lifecycle model chosen; the in-process watcher (WATCH-01) already delivers live MCP sync without daemon sprawl |
| Local Svelte web UI (SEED-001) | Triggers after parity lands (1.1); distinct from the v1.0 terminal TUI |
| `pendingChanges` exact added/modified/removed COUNT at `status`-time | Requires re-running Sync's diff on every `status` call (expensive, RESEARCH A2); the live `stale`/reindex signal (STAT-03) suffices for v1.0 |
| Cloud-API embeddings, vector search, community detection, graph-viz UI | Permanently or later-milestone out of scope per PROJECT.md (embeddings will be local-model-first) |
| Central graph server / CI index distribution | Team Scale — a later milestone (architecture already accommodates it) |

## Traceability

Each requirement maps to exactly one phase. Phase numbering is scoped to milestone v1.0 (v0.1 phases archived).

| Requirement | Phase | Status |
|-------------|-------|--------|
| EXPL-01 | Phase 1 | Complete |
| EXPL-02 | Phase 1 | Complete |
| EXPL-03 | Phase 1 | Complete |
| EXPL-04 | Phase 1 | Complete |
| EXPL-05 | Phase 1 | Complete |
| NODE-01 | Phase 1 | Complete |
| NODE-02 | Phase 1 | Complete |
| NODE-03 | Phase 1 | Complete |
| NODE-04 | Phase 1 | Complete |
| TEST-01 | Phase 1 | Complete |
| STAT-01 | Phase 2 | Complete |
| STAT-02 | Phase 2 | Complete |
| STAT-03 | Phase 2 | Complete |
| WORK-01 | Phase 2 | Complete |
| WORK-02 | Phase 2 | Complete |
| WORK-03 | Phase 2 | Complete |
| TEST-02 | Phase 2 | Complete |
| SURF-06 | Phase 2 | Complete |
| TEST-04 | Phase 3 | Complete |
| WATCH-01 | Phase 3 | Complete |
| WATCH-02 | Phase 3 | Complete |
| WATCH-03 | Phase 3 | Complete |
| WATCH-04 | Phase 3 | Complete |
| HYG-01 | Phase 4 | Complete |
| HYG-02 | Phase 4 | Complete |
| HOOK-01 | Phase 5 | Complete |
| HOOK-02 | Phase 5 | Complete |
| HOOK-03 | Phase 5 | Complete |
| TUI-01 | Phase 6 | Complete |
| TUI-02 | Phase 6 | Complete |
| TUI-05 | Phase 6 | Complete |
| DMON-01 | Phase 7 | Complete |
| DMON-02 | Phase 7 | Complete |
| DMON-03 | Phase 7 | Complete |
| DMON-04 | Phase 7 | Complete |
| TUI-03 | Phase 7 | Complete |
| TUI-04 | Phase 7 | Complete |
| TEST-03 | Phase 7 | Complete |
| SURF-01 | Phase 8 | Complete |
| SURF-02 | Phase 8 | Complete |
| SURF-03 | Phase 8 | Complete |
| SURF-04 | Phase 8 | Complete |
| SURF-05 | Phase 8 | Complete |
| REL-01 | Phase 8 | Pending |
| REL-02 | Phase 8 | Pending |
| REL-03 | Phase 8 | Pending |
| REL-04 | Phase 8 | Pending |

**Coverage:**

- v1.0 requirements: 47 total (EXPL 5, NODE 4, STAT 3, WATCH 4, DMON 4, WORK 3, HOOK 3, HYG 2, TUI 5, SURF 6, TEST 4, REL 4)
- Mapped to phases: 47 / 47 ✓
- Unmapped: 0
- Per phase: P1=10, P2=8, P3=5, P4=2, P5=3, P6=3, P7=7, P8=9

**TEST-04 note (added 2026-07-16):** the integration-test harness was surfaced during Phase-2 execution — three consecutive Criticals (CR-01/CR-02/BL-01) were reachability/composition failures a 100%-green unit + in-process suite could not see, each caught only by driving the real binary. Filed as a new requirement (user decision) rather than retrofitting the closed Phase 2. Mapped to Phase 3 (first phase to expand the `serve --mcp` surface); can move to Phase 8 with the release gate.

**SURF-06 note (added 2026-07-15, during Phase-2 planning):** SURF-01..05 are Phase 8; SURF-06 is mapped to **Phase 2** by explicit user decision. It landed there because Phase 2 already rewires every MCP read tool's result path for the WORK-02 worktree notice — doing the format change in the same pass avoids touching those 5 handlers twice and dissolves the "text-prefix a JSON payload" question entirely (with all 7 tools on markdown, one uniform notice mechanism works). Discovered while resolving 02-RESEARCH.md Open Question #3.

**Mapping notes:**

- **TEST-01** (explore/node/status behavioral harness) → Phase 1: the harness lands with the hardest algorithm (RWR) so parity is validated as it's built. Phase 8's REL-04 re-runs it green as the drop-in gate but does not own it.
- **TEST-03** (hook byte-invariance + TUI piped-stream safety) → Phase 7: the only phase where both hooks (built Phase 5) and the new bubbletea components (built Phase 7) coexist, so both halves of the assertion are satisfiable. HOOK-01/02 already bake idempotency/marker-preservation into Phase 5.
- **TUI** split: TUI-01/02/05 (rendering seam + non-interactive pretty output) → Phase 6; TUI-03/04 (interactive components) → Phase 7.

---
*Requirements defined: 2026-07-14*
*Last updated: 2026-07-14 — traceability populated during roadmap creation (45/45 mapped, 8 phases)*
