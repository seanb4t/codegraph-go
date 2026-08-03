# Pitfalls Research

**Domain:** Adding drop-in TS-parity behavior + a human bubbletea/lipgloss TUI to codegraph-go's existing shared CLI/MCP binary (milestone v1.0)
**Researched:** 2026-07-14
**Confidence:** HIGH (grounded in this repo's actual code — `internal/mcp/server.go`, `internal/cli/serve.go`, `internal/daemon/lock.go`, `internal/query/{explore,render_markdown,node}.go` — and the TS reference's own `worktree.d.ts` / `git-hooks.d.ts` / `watch-policy.d.ts` which document the exact edge cases to match)

---

## Critical Pitfalls

### Pitfall 1: ANSI/TUI styling bleeding into the AGENT path (breaks `codegraph_explore` and every piped consumer)

**What goes wrong:**
A lipgloss-styled string, a `colorprofile` writer, or a bubbletea event loop emits ANSI escape sequences (or, worse, cursor-movement / alt-screen control bytes) onto a stream that an agent is parsing. Two concrete blast zones in *this* codebase:
1. **MCP stdout is the JSON-RPC transport.** `server.ServeStdio(s)` (serve.go:115) frames JSON-RPC over stdout. Any stray byte on stdout — a lipgloss default-writer `Println`, a bubbletea render frame, even a color-reset `\x1b[0m` — corrupts the framing and the agent's connection dies or desyncs. The code already knows this: `WarnUnknownToolsTo` (server.go:64-69) documents "Diagnostics never go to stdout — stdout is reserved for the MCP JSON-RPC transport."
2. **The CLI and MCP share the SAME markdown templates** (`RenderExplore`/`RenderNode` in render_markdown.go). These strings are an agent-facing byte-for-byte contract (the `sourceDisclaimer` is "reproduce it, do not paraphrase it"). If v1.0's "prettify explore/status" work reaches into that shared renderer and wraps a symbol name in `lipgloss.NewStyle().Bold(...)`, every MCP `codegraph_explore` result now carries escape codes and the golden corpus diverges.

**Why it happens:**
lipgloss v2's ergonomics actively invite it: `lipgloss.Println` / `lipgloss.Sprint` use a *global* `lipgloss.Writer` and auto-downsample, so a developer "just styling the status output" reaches for the global and it silently targets stdout process-wide. lipgloss only strips color when it detects no TTY — but inside `serve --mcp` stdout is a pipe (not a TTY), so downsampling to "no color" is the *lucky* case; cursor/layout control bytes from bubbletea are not stripped by color profiling at all. The shared-renderer coupling means the styling entry point is one import statement away from the contract-bearing code.

**How to avoid:**
- **Name the discipline: single-seam TTY gating enforced by an import-graph archtest.** Styling must live in exactly one place — a `internal/tui` (or `internal/render/human`) package that the *CLI human path* imports and that `internal/query` and `internal/mcp` MUST NOT. This repo already has the mechanism: `internal/graphstore/archtest/import_graph_test.go` and `internal/migrate/archtest/modernc_confinement_test.go` are import-graph tests that fail the build if a forbidden dependency edge appears. **Add a third archtest asserting `internal/query` and `internal/mcp` do not (transitively) import `charmbracelet/lipgloss`, `charmbracelet/bubbletea`, or `charmbracelet/bubbles`.** That converts "don't leak ANSI into the agent path" from a code-review hope into a compile-time invariant.
- **The gate location:** styling decision belongs at the CLI command boundary (`internal/cli/*.go`), keyed on `term.IsTerminal(os.Stdout.Fd())` (via `golang.org/x/term`, or the already-present indirect `mattn/go-isatty`) AND honoring `NO_COLOR` / a `CODEGRAPH_NO_COLOR`. The shared `query.Render*` functions stay pure plain-text and are the ONLY thing MCP ever calls.
- **Never use lipgloss global writer/print functions.** Construct an explicit writer bound to the command's `cmd.OutOrStdout()`; never `lipgloss.Println`. For MCP, the plain renderer output is returned as tool-result text — lipgloss is never in that call graph at all (guaranteed by the archtest).
- **bubbletea Program must target the human command's streams explicitly** and never be constructed in any code path reachable from `serve`.

**Warning signs:**
- Golden-corpus diff shows `\x1b[` sequences or byte-count drift in `explore`/`node`/`status` MCP output.
- An agent client (Claude Code, opencode) reports the MCP server "disconnected" or "invalid JSON" shortly after a tool call.
- `go list -deps ./internal/mcp` or `./internal/query` shows a `charmbracelet/*` module.
- `codegraph explore ... | cat` (piped) shows raw escape codes.

**Phase to address:**
Earliest TUI phase — introduce the `internal/tui` package + the import-graph archtest *in the same phase as, or before, the first lipgloss import*. The archtest is cheap and must predate any styling code.

---

### Pitfall 2: Regressing proven output parity while "improving" explore relevance / node disambiguation (v0.1's golden test has a blind spot)

**What goes wrong:**
v1.0's headline parity work changes the *selection and ranking algorithms* — `explore` semantic-relevance selection, `node` multi-definition disambiguation, multi-word `<query...>` arity. But v0.1's golden-parity test only fed **single, unambiguous symbols** through the frozen templates. So the test proves *template shape* (the markdown skeleton `RenderExplore`/`RenderNode` produce) and is **blind to which symbols get selected, in what order, and how ties break**. A developer "improving relevance" can:
- Change `matchNodes` ranking (explore.go:114) and pass every existing golden test while silently returning different symbols than TS for an ambiguous query.
- "Fix" `node` to disambiguate multiple definitions and break the single-def golden because the disambiguation header now prints even for the one-def case.
- Add multi-word arity and have `strings.TrimSpace(query)` (explore.go:106) or the matcher treat `"foo bar"` as one token vs. TS's two-term semantics — a divergence no single-token fixture exercises.

Template-parity ≠ behavior-parity. The proven templates can stay byte-perfect while the *answers* diverge from TS.

**Why it happens:**
The golden corpus was built from cases that were easy to make deterministic (one symbol, one file). Ambiguity, ranking, and multi-word queries are exactly the cases that are *hard* to fixture, so they were omitted — and the omission is invisible: the suite is green, so "parity" feels done. The shared renderer makes it worse: because MCP and CLI share `Render*`, a green golden run feels like it covers both surfaces, when in fact neither surface's *selection* logic is covered for the hard inputs.

**How to avoid:**
- **Build BEHAVIORAL fixtures, not just template fixtures, before touching the algorithms.** Capture TS CodeGraph v1.3.1 output for: (a) an **ambiguous name** with 2+ definitions across files (`node <name>` disambiguation), (b) a **multi-word query** (`explore user auth token`), (c) a **relevance-ordering** case where several symbols match and order matters, (d) a **no-covering-tests** symbol (the `⚠️ no covering tests` warning). Dual-index the same tree in both binaries (the milestone was already scoped from a live bake-off — reuse that setup) and snapshot both.
- **Golden test asserts selection + order + warnings, not just the skeleton.** Assert *which* symbols appear and *in what order*, and that the disambiguation/warning lines appear only when they should.
- **Run the same behavioral fixtures through BOTH front-ends** (CLI and MCP tool handler) so a divergence can't hide on one surface — cheap here because both call the same `query.Engine`.
- **Change the algorithm behind the frozen render contract, never the contract.** Relevance/disambiguation changes belong in `explore.go`/`node.go` (selection); `render_markdown.go` (shape) should change only if TS's shape genuinely changes.

**Warning signs:**
- A relevance/disambiguation PR touches only `*_test.go` fixtures that contain a single symbol.
- Someone edits `render_markdown.go` to add a conditional header for the multi-def case (shape change driven by selection change — smell).
- No fixture in the repo contains two nodes with the same `Name` and different `FilePath`.
- The bake-off tree isn't re-diffed after the algorithm change.

**Phase to address:**
The behavioral-parity phase — and the fixture-harness work must land *first within that phase*, gating the algorithm changes. (PROJECT.md already lists "Behavioral parity test harness" as a target feature; treat it as a prerequisite, not a sibling deliverable.)

---

### Pitfall 3: Watcher-default flip breaks the daemon lockfile interplay / double-watches / stalls MCP startup

**What goes wrong:**
v1.0 flips `serve --mcp` to watch **by default** (with `--no-watch` opt-out). Today watching is opt-in via `--watch` and runs "under the SAME lockfile `internal/daemon` uses… mutually exclusive with a standalone daemon" (serve.go:79-81); a live daemon makes `d.Run` return `ErrLockLive` and serve defers. Flipping the default multiplies the exposure:
- **Every** MCP session now tries to acquire the writer lock. If a user also runs `codegraph daemon`, that's fine (serve defers). But if two agents each spawn `serve --mcp` (common — Claude Code + Cursor on the same repo), the second logs "deferring" and runs watcher-less; its offline-reconcile-on-connect (serve.go:68-73) is the only thing keeping it current, which is correct — but only if that reconcile path stays wired.
- **Startup latency / hang on slow FS.** The TS project disables the watcher on WSL2 `/mnt/*` precisely because "recursive `fs.watch`… stalls the event loop during startup long enough to blow past host handshake timeouts (opencode's 30s), so the tools never appear" (watch-policy.d.ts, issue #199). Flipping watch-on-by-default without porting that policy means codegraph-go will hang MCP init on WSL2/9p/network mounts and the agent will see zero tools.
- **Double-watching / leaked goroutines** if the default watcher and an explicit `--watch` or a daemon both start.

**Why it happens:**
"Match TS's live auto-sync" reads as a one-line default flip, but TS pairs the default with a `watch-policy` module (`CODEGRAPH_NO_WATCH` > `CODEGRAPH_FORCE_WATCH` > WSL2-`/mnt` auto-off precedence) that codegraph-go doesn't have yet. The lockfile mutual-exclusion already works for the opt-in case, so it *looks* safe to flip — but the failure mode (startup hang) only appears on the exact slow filesystems that don't show up in the developer's local test.

**How to avoid:**
- **Port `watch-policy` before flipping the default.** Implement the same precedence: `--no-watch` / `CODEGRAPH_NO_WATCH=1` (explicit off wins) → `CODEGRAPH_FORCE_WATCH=1` (on) → WSL2 + `/mnt/*` auto-off. Detect WSL via env vars then `/proc/version` "microsoft", cached.
- **Start the watcher AFTER MCP init completes / tools are advertised**, or in a goroutine that cannot block `ServeStdio`. The offline reconcile (serve.go:68-73) already runs before serving — keep that, but ensure the *watcher setup* (recursive add walk) never sits on the handshake path.
- **Keep the shared-lockfile deferral** exactly as-is (serve.go:96-104) and add a test that N concurrent `serve --mcp` processes → exactly one watcher, N-1 deferrals, zero errors surfaced to the agent.
- **goleak the flipped default**, not just the opt-in path (Phase-4 already has a goleak soak — extend it).

**Warning signs:**
- MCP tools never appear on a WSL2 or network-mounted repo; works fine locally.
- `daemon.lock` churn or `ErrLockLive` surfacing to the agent instead of being swallowed as a deferral.
- Two `serve` processes both logging watcher activity for the same repo.
- goleak reports a lingering fsnotify goroutine after serve shutdown.

**Phase to address:**
Watcher-default-flip phase — bundle the `watch-policy` port + WSL2 detection into the *same* phase as the default change; they are not separable.

---

### Pitfall 4: Worktree detection FALSE POSITIVES (submodules, nested clones, monorepo subdirs, symlinks)

**What goes wrong:**
The borrowed-index warning fires when it shouldn't, training the user to ignore it (or, worse, it fires in Sean's GSD worktree workflow inconsistently). The naive implementation ("is the resolved `.codegraph/` in a different directory than where I'm standing?") false-positives on:
- **Submodules and embedded/nested clones** — a different *repository* whose parent index legitimately doesn't cover it. TS distinguishes this with **`git rev-parse --git-common-dir`**: linked worktrees of one repo share the SAME common dir; a submodule reports `…/.git/modules/<name>` and a nested clone reports its own `.git` (worktree.d.ts). Comparing `--show-toplevel` alone can't tell a borrowed worktree from a nested repo.
- **Monorepo subdirs / non-git trees** — an unrelated parent dir that merely happens to contain a `.codegraph/`. TS guards this: it returns "no mismatch" unless `indexRoot` is *itself a working-tree root*, "which keeps non-git and monorepo-subdir layouts from producing false warnings."
- **Symlinks** — macOS `/tmp -> /private/tmp` and symlinked worktree paths. Comparisons must be on **absolute, symlink-resolved** paths (both `gitWorktreeRoot` and `gitCommonDir` in the TS reference are documented as "Absolute, symlink-resolved"). The repo already learned this lesson in `resolveSourcePath` (node.go:63-77, WR-03: EvalSymlinks both sides).

Getting this wrong specifically bites **Sean's** workflow: GSD spins a worktree per phase under gitignored paths (`.claude/worktrees/...`), which is the *exact* borrowed-index scenario TS was built to catch — so both a false negative (miss it) and a false positive (warn on a submodule inside that worktree) are real daily-driver bugs.

**Why it happens:**
`git rev-parse --show-toplevel` is the obvious call and it's *almost* right — it returns the per-worktree root, which is what you want for the primary detection. The subtlety is that toplevel alone can't classify *why* two roots differ (linked worktree vs. separate repo); that requires `--git-common-dir`. Skipping the common-dir check is the single most likely mistake.

**How to avoid:**
- **Port the TS three-way logic exactly:** use `git rev-parse --show-toplevel` for the worktree root AND `git rev-parse --git-common-dir` to decide whether `indexRoot` and `startPath` are the *same repository's* worktrees (share common dir → genuine borrow) vs. different repos (submodule/nested clone → no warning).
- **Only warn when `indexRoot` is itself a working-tree root** (guards monorepo-subdir / plain-parent-dir cases).
- **Resolve to absolute + EvalSymlinks on every path** before comparing; reuse the WR-03 pattern already in `resolveSourcePath`.
- **Best-effort/fail-open:** when git is missing or the path isn't a repo, report "no mismatch" and carry on (matches TS: "Detection is best-effort").
- **Fixture the edge cases:** a linked worktree (warn), a submodule (no warn), a nested clone (no warn), a monorepo subdir with a parent `.codegraph/` (no warn), a symlinked worktree path (warn, resolved correctly), a non-git tree (no warn). Include a `.claude/worktrees/<name>/` layout to mirror Sean's real setup.

**Warning signs:**
- The mismatch notice appears when running inside a submodule or a freshly-cloned nested repo.
- The notice does NOT appear inside a GSD phase worktree under `.claude/worktrees/`.
- Path comparisons use raw `os.Getwd()` output without `EvalSymlinks` (works on Linux, fails on macOS `/tmp`).

**Phase to address:**
Git/worktree-awareness phase. Port `worktree.ts` semantics wholesale; the compact inline MCP notice + `status` warning are the two surfaces (PROJECT.md scopes this as detect+warn+notice only — do not over-build).

---

### Pitfall 5: Git-hook install/removal that corrupts user hooks, duplicates, blocks git, or fails when codegraph isn't on PATH

**What goes wrong:**
The opt-in sync hooks (`post-commit`/`post-merge`/`post-checkout`) are the classic footgun surface. Failure modes the TS reference explicitly guards (git-hooks.d.ts):
- **Non-idempotent install** — re-running appends a second copy of the snippet. Prevented by a **marker-delimited block** ("delimited by marker comments so install is idempotent and removal preserves any user-authored hook content"). The repo already has the exact pattern for instruction files: `<!-- CODEGRAPH_START/END -->` fences (validated in Phase 6) — reuse the marker-block discipline.
- **Clobbering user hook content** — writing the whole file instead of splicing. Removal must "strip only our marker block; delete the hook file entirely when nothing but a shebang remains, otherwise rewrite the user's content untouched."
- **Blocking git** — a synchronous `codegraph sync` in `post-commit` makes every commit wait for a reindex. Must "run `codegraph sync` in the background so they never block git."
- **Failing when codegraph isn't on PATH** — a fresh checkout on another machine whose hooks reference a missing binary errors on every commit. Must be "guarded by `command -v codegraph` so they no-op cleanly when the CLI isn't on PATH."

**Why it happens:**
Hook files are shell scripts users may already own; the tempting implementation is `os.WriteFile(hookPath, snippet)`, which silently destroys a pre-existing hook. And "run sync" reads as a foreground call because that's how you'd write it in a test.

**How to avoid:**
- **Marker-fenced splice, idempotent by replace-in-place** — reuse the `CODEGRAPH_START/END` marker discipline from the Phase-6 instruction-injection work.
- **Preserve/rewrite user content on removal**; delete the file only when nothing but a shebang remains.
- **Background the sync** (`codegraph sync &` / `nohup`, detached) so git never blocks.
- **Guard with `command -v codegraph`** so the hook is a clean no-op off-PATH.
- **`isGitRepo` gate + best-effort skip** with a reason (matches `GitHookResult.skipped`).
- **Round-trip test:** install → user-edits-around-the-block → install again (no dup) → uninstall (user content intact, byte-identical). Mirror the Phase-6 install→uninstall byte-invariance test.

**Warning signs:**
- A user's existing `post-commit` hook vanishes after `codegraph` hook install.
- Two identical codegraph blocks in one hook file after two installs.
- Commits noticeably slower after enabling hooks (foreground sync).
- A teammate without codegraph on PATH sees `codegraph: command not found` on every commit.

**Phase to address:**
Git/worktree-awareness phase (same phase as worktree detection — both are the "git sync" feature in PROJECT.md). It is opt-in; ensure the default is off.

---

### Pitfall 6: Interactive TUI launched in a non-interactive context (CI, piped, agent-driven) — hangs or errors

**What goes wrong:**
The `daemon` picker, `install`/`uninstall` multi-select, and `init`/`index`/`sync` progress are interactive bubbletea programs. If launched when stdin/stdout isn't a TTY they either **block forever waiting for keypresses that never come** (a hung CI job, a hung agent tool call) or crash trying to enter raw mode / alt-screen on a pipe. Because the same binary is what agents drive, an agent invoking `codegraph install` non-interactively must get a deterministic non-interactive result, not a picker.

**Why it happens:**
bubbletea's `tea.NewProgram(...).Run()` assumes a controlling terminal; the interactive path is the "happy path" a developer tests by hand, and the non-TTY branch is easy to forget. There's an open upstream issue (bubbletea #860) about exactly this ambiguity — the framework does not auto-fallback for you.

**How to avoid:**
- **TTY-gate every interactive entry point.** Check `term.IsTerminal(os.Stdin.Fd())` **and** `os.Stdout` before constructing the Program; if either is not a terminal, take a deterministic non-interactive path (e.g. `install` with no explicit target → error listing required flags, or a sensible default; `daemon` picker → print the list as plain text or require an argument).
- **Honor an explicit override** (`--no-tui` / `CI` env / `NO_COLOR` adjacency) so agents and scripts can force non-interactive without TTY heuristics.
- **Same gate function as the color gate** (Pitfall 1) — one `isInteractive()` helper feeds both styling and interactivity decisions, so they can't disagree.
- **Test with piped stdin/stdout** in CI to prove no interactive command hangs.

**Warning signs:**
- A CI job or agent tool call on `install`/`daemon` hangs with no output.
- `codegraph daemon < /dev/null` blocks instead of erroring/printing.
- bubbletea "open /dev/tty" or raw-mode errors in non-TTY logs.

**Phase to address:**
The TUI phase — every interactive component ships with its non-TTY fallback in the same commit; add a piped-context test alongside each.

---

### Pitfall 7: Pebble WAL replay noise on stderr misclassified / suppressed too broadly

**What goes wrong:**
Pebble is opened with `pebble.Open(dir, &pebble.Options{})` (pebble_store.go:68) — a zero-value Options means Pebble's default logger writes WAL-replay and other info lines to stderr on **every command**. PROJECT.md lists "silence Pebble WAL log noise on stderr" as a v1.0 hygiene item. Three ways to get the fix wrong:
- **Over-suppression:** globally redirecting stderr or swallowing all Pebble output also hides genuine corruption/error diagnostics — a silent data-loss risk.
- **Under-scoping:** silencing only in `serve` but leaving the noise in CLI commands, so piped CLI consumers still get polluted stderr; or vice-versa.
- **Confusing the stream:** the noise is on **stderr**, which does NOT corrupt MCP stdout framing — but a careless "route Pebble logs somewhere" fix that sends them to stdout would newly break MCP. Fixing hygiene must not create Pitfall 1.

**Why it happens:**
The default `pebble.Options{}` logger is implicit; nobody chose stderr, it's just the default. The fix ("make it quiet") is one-line and tempting to do with a blunt instrument.

**How to avoid:**
- **Set an explicit `Options.Logger`** that routes Pebble INFO/WAL-replay to the codegraph structured logger at debug level (or discards INFO) while **preserving error-level messages**. Never `io.Discard` unconditionally.
- **Never route Pebble logs to stdout** in any path reachable from `serve` (protects MCP framing).
- **Verify both surfaces:** CLI stderr clean on a normal command; MCP stdout still valid JSON-RPC (unchanged); a deliberately-corrupted store still surfaces an error.

**Warning signs:**
- `codegraph status 2>/dev/null` needed to get clean output (noise still present).
- After the fix, a corrupt store fails silently with no diagnostic.
- Any Pebble log line appears on stdout under `serve --mcp`.

**Phase to address:**
Output-hygiene phase (pairs with TTY-gating). Small, but sequence it with the color gate so both stream-hygiene fixes are validated together against the MCP golden.

---

### Pitfall 8: Charm dependency tree inflating the audited-deps / reproducible-build / govulncheck story

**What goes wrong:**
The project's differentiator is "minimal, audited dependencies; reproducible/signed/SBOM'd releases," with **tree-sitter CGo as the single documented exception**. Pulling in `bubbletea` + `lipgloss` + `bubbles` drags a transitive fan-out (`muesli/termenv`, `mattn/go-runewidth`, `charmbracelet/x/*`, `colorprofile`, `ansi`, etc.). Risks specific to *this* project's guarantees:
- **New CGo** — must verify none of the Charm chain introduces a CGo dependency (would violate the "tree-sitter is the sole CGo exception" invariant and break `CGO_ENABLED=0` cross-builds for the non-parser paths).
- **Reproducible double-build breakage** — the Phase-8 `ci.yml` runs a `-mod=readonly` reproducible double-build; a Charm dep that embeds build-time timestamps, or version drift across the two builds, breaks bit-for-bit reproducibility.
- **govulncheck surface** — PROJECT.md explicitly requires auditing the new Charm deps via govulncheck/SBOM as part of v1.0. A vuln in the TUI chain that's reachable only from the human path still needs triage.
- **SBOM/attestation churn** — each new module widens the syft SBOM and the SLSA provenance material.

**Why it happens:**
TUI deps feel "cosmetic" so their supply-chain weight is underestimated; they're added late (polish phase) after the supply-chain gates were designed around a lean tree.

**How to avoid:**
- **Audit the full transitive closure before adopting:** `go mod graph` / `go list -deps` on the Charm imports; confirm **zero new CGo** (`CGO_ENABLED=0 go build` of the human-path package must succeed).
- **Pin exact versions** and run the reproducible double-build in CI *with the Charm deps in* before merging the TUI phase, not at release time.
- **Run govulncheck on the new closure** (call-graph-aware, so human-path-only vulns are visible) and regenerate the SBOM as an explicit v1.0 gate (matches the milestone's stated "audits the new Charm deps via govulncheck/SBOM").
- **Isolate the imports** to the `internal/tui` package so the archtest (Pitfall 1) also bounds the supply-chain blast radius to one package — query/mcp/graphstore stay Charm-free.

**Warning signs:**
- `CGO_ENABLED=0 go build ./...` starts failing after adding a Charm dep.
- The reproducible double-build diffs after the TUI phase.
- govulncheck reports a new advisory in a `charmbracelet/*` or `muesli/*` module.
- SBOM diff shows an unexpectedly large module fan-out.

**Phase to address:**
Split across the TUI phase (adopt + isolate + CGo check) and the final `v1.0.0` release-hardening phase (govulncheck/SBOM/reproducibility gate on the finished tree).

---

## Technical Debt Patterns

| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
|----------|-------------------|----------------|-----------------|
| Style directly in the shared `query.Render*` templates instead of a separate `internal/tui` package | One fewer package; "just wrap the name in bold" | ANSI leaks into MCP + golden corpus; the load-bearing contract now has a styling dependency | **Never** — this is the #1 pitfall |
| Add relevance/disambiguation with only single-symbol golden fixtures | Green suite fast | Silent behavioral divergence from TS on the exact hard cases parity is about | **Never** — behavioral fixtures are the whole point |
| Flip watch-on-by-default without the `watch-policy` port | One-line change, matches TS headline | MCP startup hangs on WSL2/slow FS; #199 reproduced | **Never** — port the policy in the same phase |
| Foreground `codegraph sync` in git hooks | Simplest to write/test | Every commit blocks on reindex | Only in a throwaway spike; never shipped |
| `io.Discard` the Pebble logger wholesale | Silences noise in one line | Hides corruption/error diagnostics | Never — gate by level, preserve errors |
| Adopt Charm deps in the polish phase without re-running the reproducible build | Ships the TUI faster | Reproducibility/CGo/vuln regression discovered at release | Only if the supply-chain gate runs before merge |

## Integration Gotchas

| Integration | Common Mistake | Correct Approach |
|-------------|----------------|------------------|
| MCP stdio (`server.ServeStdio`) | Any byte on stdout (lipgloss global print, bubbletea frame, Pebble log) corrupts JSON-RPC framing | Stdout is JSON-RPC ONLY; all diagnostics/styling to stderr or the human-CLI path; enforce via archtest |
| lipgloss v2 | Using the global `lipgloss.Writer` / `lipgloss.Println` (process-wide, targets stdout) | Construct an explicit writer bound to `cmd.OutOrStdout()`; never touch the global in a shared/MCP-reachable path |
| bubbletea | `tea.NewProgram().Run()` in a non-TTY context hangs/crashes | TTY-gate the entry point; deterministic non-interactive fallback (upstream #860 won't do it for you) |
| git worktrees | Comparing `--show-toplevel` alone; can't tell linked worktree from submodule/nested clone | Also use `--git-common-dir`; only warn when index root is itself a worktree root; EvalSymlinks both sides |
| git hooks | `os.WriteFile` whole hook file; foreground sync; no PATH guard | Marker-fenced splice (reuse `CODEGRAPH_START/END`); background sync; `command -v codegraph` guard |
| Pebble | Zero-value `Options{}` → default stderr logger noise | Explicit `Options.Logger` routing INFO→debug/discard, preserving errors; never to stdout |
| Charm transitive deps | Assuming TUI deps are supply-chain-free | Full `go list -deps` audit; verify zero CGo; re-run reproducible double-build + govulncheck + SBOM |

## Performance Traps

| Trap | Symptoms | Prevention | When It Breaks |
|------|----------|------------|----------------|
| Recursive watcher setup on the MCP handshake path | Tools never appear; agent times out | Port `watch-policy` (WSL2 `/mnt` auto-off); start watcher off the handshake path | WSL2 `/mnt/*`, 9p/drvfs, network mounts, very large trees |
| Foreground `sync` in git hooks | Commits/checkouts noticeably slow | Background/detached sync | Any repo big enough for sync to take >100ms |
| N concurrent `serve --mcp` each acquiring the writer lock after the default flip | Lock churn, spurious deferrals | Keep shared-lockfile deferral; test N-process → 1 watcher | Multi-agent (Claude Code + Cursor) on one repo — Sean's setup |
| `explore` verbatim-source reads uncapped | Huge payloads / slow calls | `groupMatchesByFile` already caps distinct files (`maxFiles`) — preserve when adding relevance | Ambiguous multi-word queries matching many files |

## Security Mistakes

| Mistake | Risk | Prevention |
|---------|------|------------|
| Styling/path code following symlinks out of the repo | Info disclosure (read outside repo) | The WR-03 EvalSymlinks-both-sides gate in `resolveSourcePath` already handles source reads; reuse the same discipline for any new path handling in worktree/hook code |
| Git hook that runs a binary resolved from a relative/attacker-controlled PATH | Arbitrary command execution on commit | Guard with `command -v codegraph`; bind to the resolved absolute path (matches the Phase-6 `os.Executable()` install discipline) |
| Wholesale stderr suppression hiding corruption | Silent data loss goes unnoticed | Level-gated Pebble logger; preserve error-level output |
| New Charm deps introducing an unaudited/vulnerable transitive module | Supply-chain regression against the project's core promise | govulncheck the full closure; SBOM diff; pin versions |

## UX Pitfalls

| Pitfall | User Impact | Better Approach |
|---------|-------------|-----------------|
| Borrowed-index warning false-positives on submodules/monorepo subdirs | Alarm fatigue; user ignores the real warning | Full TS worktree logic (`--git-common-dir` + index-root-is-worktree-root guard) |
| Interactive picker hangs when piped/CI/agent-driven | Hung job, no feedback | TTY-gate + deterministic non-interactive fallback |
| Colorized output dumped into a file/pager as raw escapes | Unreadable logs, broken grep | TTY-gate color at the CLI boundary; honor `NO_COLOR` |
| Git hooks silently blocking commits | User blames git, not codegraph | Background sync; opt-in and clearly announced |

## "Looks Done But Isn't" Checklist

- [ ] **Explore/node "improvements":** Often missing behavioral fixtures — verify ambiguous-name, multi-word, and relevance-ordering cases are diffed against TS 1.3.1, not just single-symbol templates.
- [ ] **TUI styling:** Often missing the archtest — verify `go list -deps ./internal/mcp ./internal/query` shows zero `charmbracelet/*`, and MCP golden output is byte-identical.
- [ ] **Watcher default flip:** Often missing `watch-policy` — verify WSL2/`/mnt` auto-off and that MCP tools still appear on a slow FS within the handshake timeout.
- [ ] **Worktree detection:** Often missing the submodule/nested-clone/monorepo-subdir negative cases — verify no false warning; verify it DOES fire in a `.claude/worktrees/` layout.
- [ ] **Git hooks:** Often missing the preserve-user-content removal path — verify install→edit-around→install→uninstall is byte-invariant and no-ops off-PATH.
- [ ] **Interactive commands:** Often missing the non-TTY branch — verify each hangs-never when stdin/stdout piped.
- [ ] **Pebble hygiene:** Often missing error-preservation — verify a corrupt store still surfaces a diagnostic, and nothing lands on MCP stdout.
- [ ] **Charm supply chain:** Often missing the re-run of the reproducible double-build + govulncheck + SBOM on the final tree.

## Recovery Strategies

| Pitfall | Recovery Cost | Recovery Steps |
|---------|---------------|----------------|
| ANSI leaked into MCP path | MEDIUM | Add the import-graph archtest (fails immediately at the leak), move styling to `internal/tui`, re-diff golden corpus |
| Behavioral divergence shipped | HIGH | Build the behavioral fixtures retroactively, diff against TS, patch selection logic; every shipped agent got wrong answers in the interim |
| Watcher-flip MCP hang on WSL2 | MEDIUM | Ship `watch-policy` port + `CODEGRAPH_NO_WATCH` escape hatch; users unblock via env var immediately |
| Worktree false positives | LOW | Add `--git-common-dir` + index-root guard + negative fixtures |
| Git hook clobbered user content | HIGH | Data already lost on that machine; restore from git reflog/backup; ship marker-fenced splice + preserve-on-removal |
| Interactive hang in CI/agent | LOW | Add TTY gate + fallback; users unblock via `--no-tui`/piped detection |
| Charm supply-chain regression | MEDIUM | Pin/rollback the offending module; re-run reproducible build + govulncheck before re-releasing |

## Pitfall-to-Phase Mapping

| Pitfall | Prevention Phase | Verification |
|---------|------------------|--------------|
| ANSI/TUI bleed into agent path | TUI phase (archtest lands first) | `go list -deps ./internal/mcp ./internal/query` has zero `charmbracelet/*`; MCP golden byte-identical |
| Behavior-vs-template parity gap | Behavioral-parity phase (harness first) | Ambiguous/multi-word/relevance fixtures diff clean vs TS 1.3.1 on BOTH front-ends |
| Watcher-default flip hazards | Watcher-flip phase | WSL2/`/mnt` auto-off test; N-process → 1-watcher test; goleak clean |
| Worktree false positives | Git/worktree-awareness phase | Submodule/nested/monorepo/symlink/`.claude/worktrees` fixture matrix |
| Git-hook corruption/blocking | Git/worktree-awareness phase | install→edit→install→uninstall byte-invariant; off-PATH no-op; background sync |
| Interactive TUI in non-interactive context | TUI phase | Every interactive command tested with piped stdin/stdout (hangs-never) |
| Pebble stderr hygiene | Output-hygiene phase | CLI stderr clean; MCP stdout valid JSON-RPC; corrupt store still errors |
| Charm supply-chain inflation | TUI phase + release-hardening phase | Zero new CGo; reproducible double-build passes; govulncheck + SBOM regenerated |

## Sources

- This repo's code (HIGH): `internal/mcp/server.go` (stdout-is-transport discipline, conditional tool registration), `internal/cli/serve.go` (`--watch`/daemon shared-lockfile interplay, offline reconcile-on-connect), `internal/daemon/lock.go` (lockfile mutual-exclusion, `ErrLockLive`), `internal/query/{explore,render_markdown,node}.go` (shared agent-facing templates, `maxFiles` cap, WR-03 EvalSymlinks path confinement), `internal/graphstore/pebble_store.go` (`pebble.Open` zero-Options default logger)
- Existing archtest precedent (HIGH): `internal/graphstore/archtest/import_graph_test.go`, `internal/migrate/archtest/modernc_confinement_test.go` — the enforcement mechanism to reuse for the Charm/query/mcp import boundary
- TS CodeGraph v1.3.x reference typedefs (HIGH, directly authoritative for parity): `sync/worktree.d.ts` (`--show-toplevel` vs `--git-common-dir`, submodule/nested/monorepo false-positive guards, symlink resolution, best-effort), `sync/git-hooks.d.ts` (marker-block idempotency, preserve-user-content removal, background sync, `command -v codegraph` guard), `sync/watch-policy.d.ts` (WSL2 `/mnt` auto-off, `CODEGRAPH_NO_WATCH`/`FORCE_WATCH` precedence, issue #199 handshake-timeout rationale)
- lipgloss v2 docs (MEDIUM, Context7 `/charmbracelet/lipgloss`): global `lipgloss.Writer`/print functions target stdout process-wide; color auto-downsamples on no-TTY but control bytes do not — motivates the single-seam gate
- bubbletea non-interactive handling (MEDIUM, web): TTY-detection-before-`NewProgram` is the required pattern; framework does not auto-fallback (issue #860); `term.IsTerminal(os.Stdin.Fd())` gate

---
*Pitfalls research for: v1.0 drop-in parity + human TUI on codegraph-go's shared CLI/MCP binary*
*Researched: 2026-07-14*
