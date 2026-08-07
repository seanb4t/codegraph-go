# v1.0 Research Summary: Drop-in Parity & Human UX

**Milestone:** v1.0 (Drop-in Parity & Human UX)  
**Researched:** 2026-07-14  
**Confidence:** HIGH (all findings grounded in TS v1.3.1 live behavior, installed codegraph-go v0.1 source, and this repo's own prior architecture decisions)

---

## Executive Summary

CodeGraph Go v1.0 closes the gap between the v0.1 reference implementation and TS CodeGraph v1.3.1 on three fronts: **behavioral parity** (the algorithms that make `explore` and `node` work as advertised), **structural parity** (flag names, defaults, command signatures), and **human UX** (a bubbletea/lipgloss TUI that can be dropped into the same binary without affecting agent-facing behavior).

The core insight is that behavioral work (relevance ranking, multi-definition disambiguation, worktree detection) lives in the *shared* `internal/query` layer — both CLI and MCP benefit simultaneously by construction, and there is no risk of them drifting. The human UX is added as a *parallel* rendering seam (`internal/cli/present`) that **never** reaches `internal/query` or `internal/mcp` — enforced by an import-graph archtest that fails the build if styling imports ever leak into those packages. This keeps MCP's JSON-RPC transport clean and preserves the golden-corpus byte-invariance contract.

The project already has architectural precedent for this pattern (the Pebble/SQLite confinement tests), and will use the exact same mechanism for the Charm TUI confinement. The payoff is that v1.0 can ship a human-friendly colored/interactive experience *without* creating two code paths or regressing agent reliability.

---

## Key Findings

### From STACK.md (Technology Stack)

**v0.1 core unchanged.** Go 1.24+, tree-sitter (CGo, justified exception), Pebble (pure-Go LSM, handles concurrent access), mark3labs/mcp-go, Cobra CLI, fsnotify file watching, cosign keyless signing.

**v1.0 addendum – Charm v2 TUI stack (pure-Go, ~20 transitive deps, zero new CGo):**
- `charm.land/bubbletea/v2@v2.0.8` — interactive event-loop framework, CLI-only
- `charm.land/lipgloss/v2@v2.0.5` — declarative styling with per-stream TTY auto-downsampling (fixes v1's stdout/stdin conflation bug)
- `charm.land/bubbles/v2@v2.1.1` — pre-built interactive components (list, spinner, progress)

**Critical:** v2 moved to vanity imports (`charm.land/...` not `github.com/charmbracelet/...`). TTY-gating uses `golang.org/x/term.IsTerminal` (already an indirect dep via sigstore-go) for interactive checks; `lipgloss` v2's own `colorprofile` owns color-downsampling decisions. Git/worktree detection uses **stdlib `os/exec`** only (two `git rev-parse` calls) — no pure-Go git library needed.

**Audit posture:** Charm modules are production-maintained (wide adoption in `gh`/K8s tooling); peripheral libs are lower-profile but low-scope. Must audit the full transitive closure for CGo (none expected), run `govulncheck` and regenerate SBOM as explicit v1.0 gates.

---

### From FEATURES.md (Behavioral Parity)

**Three headline gaps closed:**

1. **`explore` relevance algorithm** (TABLE STAKES) — currently lexical-only, leaks `Test*` funcs, misses call-connected graph neighbors. TS uses a **6-stage pipeline:**
   - Multi-word query tokenization (CamelCase, snake_case, acronyms, dot-notation, plain; ~90-word stopword list filtering)
   - Hybrid search: exact-name + prefix-match + FTS + co-occurrence re-ranking + compound-term LIKE
   - **Type-hierarchy traversal** (extends/implements neighbors)
   - **BFS expansion** (depth=3, both directions)
   - **"Glue node" injection** (direct callers/callees in already-included files)
   - **Random-Walk-with-Restart (α=0.25, 25 iterations, 9 edge kinds)** — THE mechanism that makes structural relevance beat lexical relevance
   - **File-level relevance gate** (keep files ≥6% of max mass, OR top-2 central files with text hit, OR explicitly named, OR buried type defs force-rescued) — this is the concrete gate that stops Test files with zero graph connectivity from surfacing
   - Per-root **"no covering tests" warning** (⚠️, direct callers only, skips if root has zero callers)

   **Implementation risk:** HIGH (load-bearing, golden-corpus contract at stake). Must be sequenced before any `explore` output changes.

2. **`node` multi-definition disambiguation** (TABLE STAKES) — currently returns exactly one definition (lowest ID tie-break), silently dropping N-1 of N same-named definitions.
   - Enumerate **all exact-name defs** via direct name index (not FTS, which rank-cuts overloaded names), sort **generated-files-last**
   - Optional file/line narrowing, never empties the set (hints are best-effort)
   - Rendering: 1 def → direct; >1 def → header + full bodies up to 16 within 12,000-char budget (generated-last sort order, NOT relevance), overflow listed below with hint to re-query with file/line

3. **`status` richer content** (TABLE STAKES for data; differentiator for pretty rendering) — data mostly already computed in `StatusResult`:
   - Nodes-by-kind (already scaffolded)
   - Files-by-language (already scaffolded)
   - DB size in MB (Pebble disk usage — **currently stripped by golden-corpus rule, needs revisit**)
   - Worktree-mismatch warning (verbose in status; compact prefix in other read tools)
   - Pending-changes state + reindex-recommended flag (live, not inert)

**Silent divergences requiring explicit fixes:**

| Gap | Current | TS | Classification |
|-----|---------|----|----|
| `files --filter` | language filter | **directory filter** (semantic collision) | TABLE STAKES |
| `impact` default depth | 5 | **2** | TABLE STAKES |
| `install`/`uninstall` defaults | auto/global (no prompt) | **interactive prompt** (default to auto/global only in `-y` mode) | TABLE STAKES |
| `daemon` command | foreground watch/index server | **interactive picker over running daemons** (different product, same name) | TABLE STAKES — resolved by TUI daemon picker |
| Flag short-aliases | `-l`, `-k`, `-j`, `-d` mostly missing | all present in TS | TABLE STAKES |
| `affected` flags | only `--path`/`--json` | `--stdin`, `--depth`, `--filter <glob>`, `--quiet` (git-hook/CI scripting) | TABLE STAKES |

**Worktree detection + notices** (TABLE STAKES for correctness — fixes silent "queries wrong branch" bug):
- `git rev-parse --show-toplevel` (per-worktree root) vs `--git-common-dir` (shared `.git` across worktrees)
- Verbose warning in `status` output; compact single-line notice prefixed to every other read-tool result (CLI + MCP)
- Best-effort, never blocks queries (git missing/not-a-repo → no warning)

**Git sync hooks** (TABLE STAKES for mechanism, **narrower scope than PROJECT.md**):
- TS only offers hooks when watcher is **disabled** (WSL2 `/mnt/*` or `CODEGRAPH_NO_WATCH=1`), not always-on
- Three fixed hooks: `post-commit`, `post-merge`, `post-checkout`
- Marker-fenced (`# >>> codegraph sync hook >>>` ... `# <<< codegraph sync hook <<<`)
- Background + silenced (`( codegraph sync >/dev/null 2>&1 & )`)
- Guarded by `command -v codegraph` (no-ops cleanly off-PATH)
- Idempotent replace-in-place; preserve user content on removal

**Watcher-on-MCP default** (TABLE STAKES — already-decided in PROJECT.md):
- TS: `--no-watch` opt-out (on by default)
- Go: currently `--watch` opt-in (off by default)
- Fix: flip flag default + add `--no-watch` option

---

### From ARCHITECTURE.md (Integration Approach)

**Standard architecture preserved.** CLI and MCP both call the same `internal/query.Engine`; output divergence via rendering, not behavior.

**Four new components for v1.0:**

1. **`internal/cli/present`** — TTY-gated lipgloss/bubbletea presenter wrapping `StatusResult`/`FilesResult`. **Never imported by `internal/query` or `internal/mcp`** — enforced by archtest.

2. **`internal/gitmeta`** (new package) — worktree detection, consumed by `internal/query` (to populate `WorktreeMismatch`/live staleness fields) and by `internal/cli` (pretty status banner).

3. **`internal/githooks`** (new package, separate from `internal/agents`) — install/remove git sync hooks. Reuses `internal/fsatomic` (extracted from `internal/agents/shared.go`) for atomic-write/marker-fenced plumbing.

4. **`internal/fsatomic`** (new, extracted) — `atomicWriteFile`, `replaceOrAppendMarkedSection`, `removeMarkedSection` — shared by both `internal/agents` and `internal/githooks`.

**Rendering seam pattern (proven by `internal/graphstore/archtest`):**

The CLI branches on TTY for rendering, but `internal/query.Engine` returns plain data and plain-text markup. All styling lives in `internal/cli/present` only. The MCP path uses the same plain-text output, never touches presentation code — enforced by import-graph archtest that fails the build if `charmbracelet/*` appears anywhere `internal/mcp` or `internal/query` can reach.

**Worktree detection lives in query layer** — both CLI and MCP get `WorktreeMismatch` in the same commit, zero drift risk.

**Watcher-on-MCP** — existing `--watch` plumbing already correct; only the flag default flips + `--no-watch` added.

**Suggested build order** (front-loads highest-risk, shared-engine work):
1. **Phase A:** `explore`/`node` relevance + disambiguation (internal/query only)
2. **Phase B:** `status` richer content + worktree awareness (internal/gitmeta + Engine)
3. **Phase C:** Watcher-on-MCP default flip + watch-policy port
4. **Phase D:** Output hygiene (Pebble logger, stderr discipline)
5. **Phase E:** Git hooks (internal/githooks, opt-in surface)
6. **Phase F:** Rendering seam + pretty `status`/`files` (archtest-first)
7. **Phase G:** Interactive daemon picker + install multi-select (bubbletea)
8. **Phase H:** Parity test harness + surface reconciliation + v1.0.0 release

---

### From PITFALLS.md (Load-Bearing Risks)

**Critical pitfalls must be prevented by design, not code-review:**

1. **ANSI/TUI bleeding into MCP path** — lipgloss on stdout corrupts MCP JSON-RPC; styling in shared `query.Render*` diverges golden corpus. **Prevention:** Import-graph archtest (identical to existing Pebble/SQLite confinement) asserting `internal/query` and `internal/mcp` do NOT import `charmbracelet/*`.

2. **Template-parity ≠ behavior-parity** — v0.1's golden test only fed single unambiguous symbols; proves shape but is blind to selection/ranking/tie-breaking. **Prevention:** Build behavioral fixtures for ambiguous names, multi-word queries, relevance ordering, "no covering tests" warnings; diff both surfaces against TS 1.3.1.

3. **Watcher-default flip breaks daemon lockfile + MCP hangs** — startup hangs on WSL2 without watch-policy port. **Prevention:** Port `watch-policy` (WSL2 auto-off, env precedence) in same phase; start watcher off MCP handshake path; test N concurrent → 1 watcher.

4. **Worktree false positives** — naive implementation warns on submodules/monorepo subdirs/symlinks. **Prevention:** Port TS logic (use `--git-common-dir`, only warn when indexRoot is itself worktree root, EvalSymlinks both sides). Fixture edge cases.

5. **Git-hook corruption** — can silently block `git commit`/`checkout` if not idempotent or foreground. **Prevention:** Marker-fenced splice; idempotent; preserve user content on removal; background sync; `command -v` guard; fixture byte-invariance.

6. **Interactive TUI in non-TTY context** — bubbletea hangs when stdin/stdout piped. **Prevention:** TTY-gate before `tea.NewProgram()` using `term.IsTerminal()`; deterministic non-interactive fallback; test with piped streams (never hang).

7. **Pebble WAL log noise hides corruption** — wholesale silencing loses error diagnostics. **Prevention:** Explicit `Options.Logger` routing INFO→debug/discard, preserving errors; never to stdout.

8. **Charm supply-chain inflation** — TUI deps risk CGo regression, reproducibility churn, vuln exposure. **Prevention:** Audit transitive closure for CGo; re-run reproducible double-build + govulncheck + SBOM before merge.

---

## Implications for Roadmap

### Suggested Phase Structure

**Phase A** — `explore`/`node` Relevance & Disambiguation (internal/query only)
- **Deliverables:** RWR ranking (9 edge kinds, α=0.25, 25 iters), file-level gate, multi-word tokenization + stopword filter, multi-definition disambiguation, "no covering tests" warning
- **Validation:** Behavioral fixtures vs. TS 1.3.1 for ambiguous names, multi-word queries, relevance ordering, warnings; both CLI and MCP surfaces
- **Rationale:** Highest risk to golden-corpus; shared engine means both surfaces improve in same commit

**Phase B** — `status` Richer Content + Worktree Awareness (internal/gitmeta + Engine)
- **Deliverables:** Pebble disk-size, gitmeta detection (show-toplevel vs git-common-dir, symlinks, edge-case fixtures), wire into StatusResult
- **Validation:** Golden status output includes DB size, nodes-by-kind, languages, pending-changes, worktree warning; linked/submodule/monorepo-subdir/`.claude/worktrees/` fixtures correct
- **Rationale:** Both surfaces get worktree awareness in one commit; validate edge cases before rendering

**Phase C** — Watcher-on-MCP Default Flip + watch-policy Port (internal/cli/daemon)
- **Deliverables:** WSL2 detection (CODEGRAPH_NO_WATCH/FORCE_WATCH precedence, /proc/version), --no-watch flag, watcher off handshake, goleak test
- **Validation:** MCP tools appear within timeout on WSL2/slow FS; 2 concurrent serve → 1 watcher; goleak clean
- **Rationale:** Bundle watch-policy + default flip; inseparable

**Phase D** — Output Hygiene (internal/graphstore, internal/mcp)
- **Deliverables:** Explicit Pebble logger (INFO→debug/discard, preserve errors), no bytes on MCP stdout
- **Validation:** CLI stderr clean; corrupt store still errors; MCP stdout valid JSON-RPC

**Phase E** — Git Hooks (internal/githooks, internal/fsatomic extraction, opt-in CLI)
- **Deliverables:** Extract fsatomic from agents; new githooks package; `codegraph githooks install|remove|status` command; marker-fenced + idempotent + background + PATH guard
- **Validation:** install→edit→remove byte-invariant; off-PATH no-op; commits don't block

**Phase F** — Rendering Seam + Pretty `status`/`files` (internal/cli/present + archtest)
- **Deliverables:** Archtest FIRST (fails immediately if lipgloss/bubbletea leak to query/mcp), then add present package with RenderStatus/RenderFiles, lipgloss/bubbles deps
- **Validation:** Archtest passes; golden MCP output byte-identical; CLI plain fallback unchanged; TTY gates color

**Phase G** — Interactive Daemon Picker + Install Multi-Select (bubbletea)
- **Deliverables:** daemon.Status() read API, bubbletea daemon picker, replace bufio multi-select, --no-tui override
- **Validation:** Picker TTY-only; non-TTY doesn't hang; install with pipes requires flags or auto defaults

**Phase H** — Parity Test Harness + Surface Reconciliation + v1.0.0 Release
- **Deliverables:** Behavioral fixtures validated vs. TS; all flag short-aliases; `files --filter` collision resolved; `daemon` identity resolved (it's the picker); `impact` depth→2; govulncheck + SBOM + reproducible double-build + SLSA provenance
- **Validation:** Flag audit vs. live TS binary; golden parity both surfaces; security gates pass

---

## Confidence Assessment

| Area | Confidence | Caveats |
|------|-----------|---------|
| Stack (core) | HIGH | v0.1 ships; active libraries |
| Stack (Charm v2) | HIGH | GitHub go.mod direct reads; must audit CGo + govulncheck before merge |
| Features (TS parity) | HIGH | Behavioral fixtures for hard cases must be built BEFORE algorithm changes |
| Architecture | HIGH | Patterns match existing precedent (archtest, confinement) |
| Worktree detection | MEDIUM-HIGH | Edge cases need live testing (symlinks, submodules, GSD-layout) |
| Watch-policy port | MEDIUM | WSL2 startup-hang needs reproduction; MCP timeout sensitivity untested |
| Git hooks | MEDIUM | User-hook-preservation and Husky/Lefthook interaction untested |
| Pitfalls | HIGH | Prevention strategies proven; risks remain MEDIUM for phase-specific details |

### Gaps Requiring Later Attention

1. **Behavioral fixture harness** — v0.1 only pins template shape. Ambiguous/multi-word/relevance cases must be diffed vs. TS 1.3.1 before Phase B ships.

2. **WSL2 live testing** — watch-policy port assumes issue #199 root cause. Needs real WSL2 reproduction before Phase C.

3. **`files --filter` resolution** — semantic collision (ours=language; TS=directory) is #1 silent-failure risk. Decide fix early.

4. **Daemon command identity** — current "server" vs. required "picker" resolved by TUI daemon picker, but marks a breaking change.

---

## Sources

- TS CodeGraph v1.3.1 installed dist JS (HIGH: direct reads)
- Live `codegraph <cmd> --help` output (HIGH: v1.3.1 binary)
- This repo's v0.1 source code (HIGH: direct inspection)
- Charm repositories GitHub tags + raw go.mod files (HIGH: verified vanity import paths)
- Context7 library docs (MEDIUM: supplementary)
- Existing archtest precedent in this repo (HIGH: model for new gates)
- Git documentation + general shell/git knowledge (MEDIUM)

---

**Research completed:** 2026-07-14  
**For:** v1.0 milestone roadmapping

