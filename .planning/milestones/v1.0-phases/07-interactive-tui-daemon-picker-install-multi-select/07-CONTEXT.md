# Phase 7: Interactive TUI — Daemon Picker & Install Multi-Select - Context

**Gathered:** 2026-07-18
**Status:** Ready for planning
**Mode:** --auto --chain (all gray areas auto-selected; recommended options chosen and logged in 07-DISCUSSION-LOG.md; auto-advancing to plan → execute)

<domain>
## Phase Boundary

Add the **interactive bubbletea/bubbles layer** on top of the Phase-6 rendering
seam. Three interactive surfaces plus their supporting daemon lifecycle, every
one of which **auto-falls back to non-interactive behavior when piped and never
hangs**:

- **DMON-01 / TUI-04** — `codegraph daemon` (no args) opens an interactive
  bubbletea **picker** of running daemons (current project first) → stop one /
  stop-all / cancel, resolving the TS name collision (TS `daemon` = picker;
  ours is currently a foreground server).
- **DMON-02** — explicit `daemon start` / `daemon stop` / `daemon stop --all`
  manage the shared background daemon lifecycle, with **no silent auto-spawn**.
- **DMON-03** — a **PPID watchdog** shuts down any daemon / in-process watcher
  when its supervising host or agent process dies (POSIX ppid-reparent +
  Windows liveness poll).
- **DMON-04** — a global **`~/.codegraph/daemons` registry** lets the picker
  list/stop daemons across projects, self-healing stale records.
- **TUI-03** — `install` / `uninstall` present an interactive **bubbles
  multi-select** agent picker by default, with `-y`/`--yes` for
  non-interactive auto/global.
- **TEST-03** — git-hook install→edit→remove is byte-invariant, and every
  interactive component is tested against **piped streams** (never hangs).

**Not in this phase (explicit out-of-scope):**
- **Daemon auto-spawn** — permanently out per PROJECT.md; `serve --mcp`'s
  in-process watcher (WATCH-01) is the zero-config live-sync path.
- **True detached / double-forked per-project daemons + unix-socket sharing** —
  that is **DMON-FUT-01** (a later milestone), not v1.0. v1.0's daemon is an
  explicit foreground process managed by start/stop.
- **The Charm dependency-closure audit** (CGo / govulncheck / SBOM /
  reproducible double-build, **REL-01**) → **Phase 8**.
- **Flag reconciliation** (SURF-01..05) and the signed release (REL-02..04) →
  **Phase 8**.
- Colorizing/interactive-izing any read command beyond what's named here.

</domain>

<decisions>
## Implementation Decisions

### Daemon command shape & lifecycle (DMON-01, DMON-02)
- **D-01:** `codegraph daemon` becomes a **command tree**. No-args → the
  interactive picker on a TTY (plain list off-TTY, D-12); subcommands
  `daemon start`, `daemon stop`, `daemon stop --all`. The current
  foreground-blocking RunE in `internal/cli/daemon.go` (which calls
  `daemon.Run`) **moves to `daemon start`** — same behavior, new name. This
  resolves the TS name collision (bare `daemon` is a picker in TS, was a
  foreground server in ours).
  `[auto] Daemon command shape → Selected: cobra sub-tree (bare=picker, start/stop/stop --all) (recommended)`
- **D-02:** `daemon start` is an **explicit foreground blocking daemon** —
  reuse the existing `daemon.New` / `Run` / `RunWithRetry` + the
  `.codegraph/daemon.lock` single-writer lockfile as-is. **No detached
  double-fork / OS-level backgrounding** in v1.0 (that is DMON-FUT-01).
  "Background" means the user/agent backgrounds it (`&`) or spawns it as a
  child; the point of DMON-02 is *explicit lifecycle*, not process
  daemonization.
  `[auto] daemon start model → Selected: foreground blocking Run, no detached fork (recommended)`
- **D-03:** **No silent auto-spawn anywhere.** `serve --mcp`'s in-process
  watcher (WATCH-01, Phase 3) stays the zero-config live-sync path; the
  standalone daemon is *only* the explicit shared-writer case a user/agent
  starts deliberately. (Locked by PROJECT.md Out-of-Scope + REQUIREMENTS
  Out-of-Scope.)

### Global daemon registry (DMON-04)
- **D-04:** A global registry directory **`~/.codegraph/daemons/`**, with **one
  record file per running daemon** (not a single shared JSON file). Each record
  is written **atomically via `internal/fsatomic.WriteFile`** (Phase 5). Fields:
  at minimum `pid`, `startedAt`, `repoRoot` (exact filename scheme + any extra
  fields are Claude's discretion). One-file-per-daemon avoids a global
  write-lock and mirrors the per-project lockfile's create+liveness model.
  Home dir via `os.UserHomeDir()`.
  `[auto] Registry format → Selected: dir of atomic per-daemon record files (recommended)`
- **D-05:** **Self-healing on read**, no background reaper. Any scan (picker
  open, the plain list, `stop --all`) stats each record's pid liveness by
  **reusing `internal/daemon/lock.go`'s `isProcessLive` / `isStale`** (including
  the Linux `/proc` start-time corroboration already there) and **prunes dead /
  stale records** in place. This is the same "detect+clear stale on every
  independent call" discipline `acquire()` already uses (D-16 of Phase 4).
- **D-06:** Each daemon **registers its record at `Run` start** and
  **best-effort removes it on clean shutdown** (a `defer`, exactly like
  `release()` for the lockfile). Records orphaned by a crash are cleaned by
  D-05's next scan — never by a long-lived reaper.

### PPID watchdog (DMON-03)
- **D-07:** A **background goroutine polls parent liveness** on a modest
  interval (~1–2s, exact value Claude's discretion) and **cancels the
  daemon/watcher `ctx`** when the supervising process dies. POSIX: capture the
  original ppid at start; treat **reparenting** (ppid changed away from the
  captured value, e.g. → 1/init or a subreaper) as parent death. Windows: poll
  the parent process handle's liveness. Implemented as **build-tag-split files
  (`watchdog_posix.go` / `watchdog_windows.go`)**, mirroring the existing
  `procstart_linux.go` / `procstart_other.go` platform-split precedent.
  `[auto] Watchdog mechanism → Selected: poll ppid/parent-liveness, reparent=death, build-split (recommended)`
- **D-08:** Wire the watchdog into **both** `daemon start`/`Run` **and**
  `serve --mcp`'s in-process watcher (both run as children of a host/agent).
  It lives in **`internal/daemon` (charm-free)** and does nothing but cancel the
  ctx — which already triggers clean teardown (release lock, join goroutines,
  prune the registry record). No new shutdown path.

### Charm isolation & interactive seam (TUI-01, TUI-03, TUI-04)
- **D-09:** **bubbletea + bubbles live ONLY in the `internal/cli` presentation
  layer** — extend `internal/cli/present` (or a sibling `internal/cli/tui`).
  `internal/daemon`, `internal/agents`, and the whole guarded engine set stay
  charm-free. **The Phase-6 TUI-01 archtest already forbids
  `charm.land/bubbletea/v2` + `charm.land/bubbles/v2` from the six guarded
  packages — including `internal/daemon`** — so it stays green *by
  construction* once the deps are added. Data layers produce plain structs the
  UI consumes: the daemon **registry lister** (charm-free, in `internal/daemon`)
  and `agents.AllTargets()` / `agents.DetectAll()` feed the picker views. This
  is the exact Phase-6 "producers plain, `present` renders" seam extended to
  interactive components.
  `[auto] Charm seam → Selected: bubbletea/bubbles confined to internal/cli, data layers charm-free (recommended)`
- **D-10:** Every interactive entry point **TTY-gates via the SAME
  `present.ChoosePresentation` / `term.IsTerminal` seam BEFORE calling
  `tea.NewProgram()`**. Not a TTY (piped / CI / `NO_COLOR`) ⇒ the
  non-interactive fallback (D-12/D-13) — never a blocked stdin read. This is the
  literal "TTY-gate before `tea.NewProgram()`" the ROADMAP Notes mandate.
- **D-11:** Add **`charm.land/bubbletea/v2` + `charm.land/bubbles/v2`** to
  `go.mod` (pinned exact versions) — the interactive layer Phase 6 deliberately
  deferred (`charm.land/lipgloss/v2` is already present). Prefer pure-Go/no-CGo
  by construction; the full REL-01 closure audit is Phase 8.

### Non-interactive fallbacks (TUI-04)
- **D-12:** **daemon picker no-TTY fallback:** `codegraph daemon` (no args,
  non-TTY) prints the **plain-text list of running daemons** (current project
  first) and exits 0 — a read-only, script-safe listing that never blocks.
  (This implicitly provides a "list" behavior; whether to also expose an
  explicit `daemon list` alias is Claude's discretion.)
  `[auto] daemon no-TTY fallback → Selected: print plain running-daemon list, exit 0 (recommended)`
- **D-13:** **install/uninstall no-TTY fallback:** keep today's behavior —
  resolve straight to `auto` (install.go's existing default branch), never
  prompt. `-y`/`--yes` (D-15) forces this same non-interactive path even on a
  TTY. Empty/EOF stdin degrades to `auto`, never hangs (already true today).

### install / uninstall multi-select (TUI-03)
- **D-14:** Replace install's current **plain numbered-line prompt**
  (`promptAgentMultiSelect` in `internal/cli/install.go`) with a **bubbles
  multi-select (checkbox list)** on a TTY, pre-checking the targets
  `agents.DetectAll(loc)` reports installed. Do the same for `uninstall`
  (pre-check installed). The non-TTY / `-y` path reuses
  `agents.ResolveTargetFlag("auto", loc)`. Keep `--target` / `--location` /
  `--auto-allow` unchanged. The bubbles UI lives in the cli layer (D-09), so
  `internal/agents` stays charm-free.
  `[auto] install multi-select → Selected: bubbles checkbox list (TTY), auto fallback off-TTY (recommended)`
- **D-15:** Add **`-y` / `--yes`** to **both** `install` and `uninstall`: skip
  the picker and use the non-interactive default set (`auto`), for scripts/CI.
  Matches TS.

### TEST-03 (byte-invariance + piped never-hang)
- **D-16:** **githooks byte-invariance** — a focused `internal/githooks` test:
  install → simulate a user edit *outside* the marker block → remove ⇒ the file
  returns **byte-identical to the pre-install original** (marker block fully
  stripped, user content intact). Exercises the Phase-5 strip-then-restore
  semantics; no new mechanism.
- **D-17:** **piped-stream never-hang** — assertions in the existing
  **`test/integration/` subprocess harness** (TEST-04's home): spawn the real
  binary for `daemon` (no args) and `install` with **piped / closed
  stdin+stdout under a timeout**; assert it exits promptly with the
  non-interactive output and never blocks. Adding bubbletea must not regress
  this.
  `[auto] TEST-03 harness → Selected: githooks unit byte-invariance + integration piped never-hang (recommended)`

### Claude's Discretion
- Exact bubbles list styling, key bindings, and checkbox glyphs; picker column
  layout (e.g. repo / pid / age).
- Watchdog poll interval within ~1–2s; the exact reparent-detection predicate
  (ppid==1 vs ppid≠original) — pick the more robust given subreapers.
- Registry record filename scheme (pid- vs hash-based) and any fields beyond
  `{pid, startedAt, repoRoot}`.
- Whether to expose an explicit `daemon list` alias (D-12 already covers the
  behavior).
- Stop signal policy: **recommend graceful `SIGTERM` only** (the daemon's
  existing signal handling cancels ctx → releases lock → prunes registry;
  stale records self-heal via D-05). A grace-timeout + `SIGKILL` escalation is
  discretion only if a hung daemon proves to be a real problem.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase scope (locked)
- `.planning/ROADMAP.md` §"Phase 7: Interactive TUI — Daemon Picker & Install
  Multi-Select" — goal, 5 success criteria, and the Notes (bubbletea/bubbles
  layer; no auto-spawn; TTY-gate before `tea.NewProgram()`; test with piped
  streams; TEST-03 rationale).
- `.planning/REQUIREMENTS.md` — DMON-01, DMON-02, DMON-03, DMON-04, TUI-03,
  TUI-04, TEST-03 (plus the DMON-FUT-01 deferral and the "no daemon auto-spawn"
  Out-of-Scope row).

### Cross-phase decisions carried forward (MUST honor)
- `.planning/phases/06-rendering-seam-pretty-status-files/06-CONTEXT.md` —
  D-01/D-04 (the `internal/cli/present` seam + `present.ChoosePresentation`),
  D-10/D-11/D-12 (the TUI-01 archtest mechanics, forbidden `charm.land/…/v2`
  paths, guarded set, self-defeat guard). Phase 7 extends this seam to
  interactive components; the archtest must stay green.
- `.planning/phases/03-watcher-on-mcp-default/03-CONTEXT.md` — the daemon /
  watcher lifecycle + policy-gate model the watchdog and start/stop build on.
- `.planning/phases/05-git-sync-hooks/05-CONTEXT.md` — the githooks
  strip/restore semantics TEST-03's byte-invariance asserts.

### Daemon lifecycle & lock (DMON-01..04)
- `internal/cli/daemon.go` — the command to transform into the start/stop/picker
  tree (currently a single foreground `RunE`; note the `watch.DisabledError`
  friendly-exit branch to preserve).
- `internal/daemon/daemon.go` — `New` / `Run` / `RunWithRetry` / `flush`; the
  single-writer model + WATCH-03 policy gate + `WithProbe` Option the watchdog
  and registry hook into (register on Run start, cancel ctx on parent death).
- `internal/daemon/lock.go` — `acquire` / `release` / `Unlock` / `isStale` /
  `isProcessLive` / `startTimesCorroborate`; **reuse `isProcessLive`/`isStale`
  for registry self-heal (D-05)**.
- `internal/daemon/procstart_linux.go` / `procstart_other.go` — the build-tag
  platform-split precedent the watchdog files mirror.
- `internal/cli/unlock.go` — the `codegraph unlock` thin CLI over
  `daemon.Unlock`, an existing lifecycle command to stay consistent with.

### Interactive components (TUI-03/04) & the charm seam (TUI-01)
- `internal/cli/present/tty.go` — `ChoosePresentation(isTTY, noColor)`, the
  shared branch selector every interactive entry gates on (D-10).
- `internal/cli/present/progress.go` — the Phase-6 charm-in-cli, TTY-gated,
  stop-deterministic precedent (lipgloss-only today; bubbletea/bubbles are the
  new additions).
- `internal/cli/present/archtest/import_graph_test.go` — the TUI-01 archtest:
  `guardedPackages` (mcp/graphstore/daemon/watch/indexer/query),
  `forbiddenImportPaths` (`charm.land/{lipgloss,bubbletea,bubbles}/v2`),
  self-defeat guard. **Already forbids bubbletea/bubbles from `internal/daemon`
  — the constraint that forces the picker UI into the cli layer.**
- `internal/cli/install.go` — the multi-select integration point:
  `installStdinIsInteractive`, `promptAgentMultiSelect` (to be replaced by
  bubbles), `printAgentResults`, `--target/--location/--auto-allow` flags.
- `internal/cli/uninstall.go` — the uninstall command (same multi-select +
  `-y`/`--yes` treatment).
- `internal/agents/registry.go` — `AllTargets()` / `DetectAll(loc)` /
  `ResolveTargetFlag(spec, loc)`; `internal/agents/types.go` — the
  `AgentTarget` interface. The charm-free data the picker consumes.

### Registry writes & TEST-03 harness
- `internal/fsatomic/fsatomic.go` — `WriteFile(path, content)` for atomic
  registry record writes (D-04).
- `internal/githooks/githooks.go` — the Install/Remove/Status semantics
  TEST-03's byte-invariance test drives (D-16).
- `test/integration/main_test.go` (+ siblings `watch_default_test.go`,
  `worktree_notice_test.go`, `status_files_plain_test.go`) — the subprocess
  harness the piped-stream never-hang assertions extend (D-17).

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `daemon.New`/`Run`/`RunWithRetry` + `.codegraph/daemon.lock` — `daemon start`
  is a rename of the current foreground command; no new lifecycle engine.
- `lock.go`'s `isProcessLive` / `isStale` (incl. Linux `/proc` start-time
  corroboration) — the registry's self-heal predicate, verbatim reuse.
- `procstart_linux.go` / `procstart_other.go` — the exact build-tag split shape
  for `watchdog_posix.go` / `watchdog_windows.go`.
- `present.ChoosePresentation` + `term.IsTerminal(fd)` call sites
  (status.go:61, files.go:68, progress_cli.go:31) — the copy-paste TTY-gate for
  each interactive entry, applied *before* `tea.NewProgram()`.
- `install.go`'s `promptAgentMultiSelect` / `selectByIndices` /
  `printAgentResults` — the selection→install pipeline the bubbles picker slots
  into (replace the *input* mechanism, keep the resolution + reporting).
- `fsatomic.WriteFile` — atomic registry record writes.
- `test/integration/` subprocess harness (real binary, piped transports) —
  TEST-03's piped never-hang home.

### Established Patterns
- **"Producers plain, `present` renders"** (Phase 6) — data layers
  (daemon registry, agents registry) stay charm-free; only the cli presentation
  layer imports bubbletea/bubbles.
- **Build-enforced ANSI isolation** — the TUI-01 archtest fails the build if
  charm reaches the guarded engine set (incl. `internal/daemon`); it is the
  structural reason the picker UI must live in cli.
- **Detect-and-clear-stale on every independent call** (`acquire`, Phase 4
  D-16) — the registry reuses it instead of a background reaper.
- **Build-tag platform split** (`procstart_*`) — reused for the watchdog.
- **Injectable-seam testing** (`installStdinIsInteractive` var,
  `daemon.onSync*`/`onWatchOpen` seams) — the pattern for making the
  interactive/watchdog paths unit-testable without a real pty.
- **TTY-gate + never-block-on-stdin** — install already degrades EOF/non-TTY to
  `auto`; extend the same guarantee to the daemon picker.

### Integration Points
- `internal/cli/daemon.go` `RunE` → split into a picker (bare) + `start`/`stop`/
  `stop --all` subcommands; register the watchdog + registry in `daemon.Run`.
- `daemon.Run` (`internal/daemon/daemon.go`) → register a registry record on
  start (best-effort remove on shutdown, like `release()`); start the PPID
  watchdog goroutine that cancels ctx on parent death.
- `serve --mcp`'s in-process watcher (Phase 3 `serveWatchStart`) → same
  watchdog wiring.
- `internal/cli/install.go` + `uninstall.go` `RunE` → bubbles multi-select on
  TTY (D-14), `-y`/`--yes` flag (D-15).
- New `go.mod` requires: `charm.land/bubbletea/v2`, `charm.land/bubbles/v2`.
- New registry package/type (charm-free, likely in `internal/daemon` or a new
  `internal/daemon/registry.go`): list / register / deregister / prune-stale.

</code_context>

<specifics>
## Specific Ideas

- Charm **v2** vanity imports are `charm.land/bubbletea/v2` +
  `charm.land/bubbles/v2` (NOT `github.com/charmbracelet/…`, NOT bare non-`/v2`)
  — the exact paths the TUI-01 archtest's `forbiddenImportPaths` names.
- TTY gate is `golang.org/x/term.IsTerminal`, applied at the cli RunE boundary
  before `tea.NewProgram()` — ROADMAP-locked.
- The daemon registry lives at `~/.codegraph/daemons/` (global, cross-project) —
  distinct from the per-project `.codegraph/daemon.lock`.
- No-auto-spawn is a hard, user-locked constraint — the daemon is explicit
  start/stop only.

</specifics>

<deferred>
## Deferred Ideas

- **DMON-FUT-01** — true detached / double-forked per-project daemons +
  unix-socket sharing (full TS auto-spawn parity). A later milestone; v1.0 ships
  the explicit foreground start/stop + picker + registry + watchdog model only.
- **Explicit `daemon list` subcommand alias** — if not adopted as discretion
  under D-12, it is a natural low-cost follow-up.
- **SIGKILL grace-timeout escalation** for `stop` — v1.0 recommends graceful
  SIGTERM only; escalation only if hung daemons prove real.
- **Charm dependency-closure audit** (CGo / govulncheck / SBOM / reproducible
  double-build, REL-01) → **Phase 8**.
- **`--color` / `--no-color` explicit override flag** — carried from Phase 6;
  v1.0 relies on TTY-detection + `NO_COLOR`.

### Reviewed Todos (not folded)
- **"Document release procedures (maintainer runbook)"** (score 0.4) —
  reviewed, **NOT folded**. It matched only on generic keywords (phase/user)
  and is a release-process runbook belonging to **Phase 8** (Signed v1.0.0
  Release / REL-02), not the interactive TUI. Same disposition Phase 6 gave the
  same todo. Deferred to Phase 8.

</deferred>

---

*Phase: 7-Interactive TUI — Daemon Picker & Install Multi-Select*
*Context gathered: 2026-07-18*
