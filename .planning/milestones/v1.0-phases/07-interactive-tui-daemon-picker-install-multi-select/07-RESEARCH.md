# Phase 7: Interactive TUI — Daemon Picker & Install Multi-Select - Research

**Researched:** 2026-07-18
**Domain:** Terminal UI (bubbletea/bubbles v2), process lifecycle (PPID watchdog, cross-platform signaling), file-based service registry
**Confidence:** HIGH (stack/versions/API verified via `go list -m -versions` + Context7 official docs + direct code reading; a small number of design-detail gaps are explicitly flagged ASSUMED below)

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**Daemon command shape & lifecycle (DMON-01, DMON-02)**
- **D-01:** `codegraph daemon` becomes a **command tree**. No-args → the interactive picker on a TTY (plain list off-TTY, D-12); subcommands `daemon start`, `daemon stop`, `daemon stop --all`. The current foreground-blocking RunE in `internal/cli/daemon.go` (which calls `daemon.Run`) **moves to `daemon start`** — same behavior, new name. This resolves the TS name collision (bare `daemon` is a picker in TS, was a foreground server in ours).
- **D-02:** `daemon start` is an **explicit foreground blocking daemon** — reuse the existing `daemon.New` / `Run` / `RunWithRetry` + the `.codegraph/daemon.lock` single-writer lockfile as-is. **No detached double-fork / OS-level backgrounding** in v1.0 (that is DMON-FUT-01). "Background" means the user/agent backgrounds it (`&`) or spawns it as a child; the point of DMON-02 is *explicit lifecycle*, not process daemonization.
- **D-03:** **No silent auto-spawn anywhere.** `serve --mcp`'s in-process watcher (WATCH-01, Phase 3) stays the zero-config live-sync path; the standalone daemon is *only* the explicit shared-writer case a user/agent starts deliberately. (Locked by PROJECT.md Out-of-Scope + REQUIREMENTS Out-of-Scope.)

**Global daemon registry (DMON-04)**
- **D-04:** A global registry directory **`~/.codegraph/daemons/`**, with **one record file per running daemon** (not a single shared JSON file). Each record is written **atomically via `internal/fsatomic.WriteFile`** (Phase 5). Fields: at minimum `pid`, `startedAt`, `repoRoot` (exact filename scheme + any extra fields are Claude's discretion). One-file-per-daemon avoids a global write-lock and mirrors the per-project lockfile's create+liveness model. Home dir via `os.UserHomeDir()`.
- **D-05:** **Self-healing on read**, no background reaper. Any scan (picker open, the plain list, `stop --all`) stats each record's pid liveness by **reusing `internal/daemon/lock.go`'s `isProcessLive` / `isStale`** (including the Linux `/proc` start-time corroboration already there) and **prunes dead / stale records** in place. This is the same "detect+clear stale on every independent call" discipline `acquire()` already uses (D-16 of Phase 4).
- **D-06:** Each daemon **registers its record at `Run` start** and **best-effort removes it on clean shutdown** (a `defer`, exactly like `release()` for the lockfile). Records orphaned by a crash are cleaned by D-05's next scan — never by a long-lived reaper.

**PPID watchdog (DMON-03)**
- **D-07:** A **background goroutine polls parent liveness** on a modest interval (~1–2s, exact value Claude's discretion) and **cancels the daemon/watcher `ctx`** when the supervising process dies. POSIX: capture the original ppid at start; treat **reparenting** (ppid changed away from the captured value, e.g. → 1/init or a subreaper) as parent death. Windows: poll the parent process handle's liveness. Implemented as **build-tag-split files (`watchdog_posix.go` / `watchdog_windows.go`)**, mirroring the existing `procstart_linux.go` / `procstart_other.go` platform-split precedent.
- **D-08:** Wire the watchdog into **both** `daemon start`/`Run` **and** `serve --mcp`'s in-process watcher (both run as children of a host/agent). It lives in **`internal/daemon` (charm-free)** and does nothing but cancel the ctx — which already triggers clean teardown (release lock, join goroutines, prune the registry record). No new shutdown path.

**Charm isolation & interactive seam (TUI-01, TUI-03, TUI-04)**
- **D-09:** **bubbletea + bubbles live ONLY in the `internal/cli` presentation layer** — extend `internal/cli/present` (or a sibling `internal/cli/tui`). `internal/daemon`, `internal/agents`, and the whole guarded engine set stay charm-free. **The Phase-6 TUI-01 archtest already forbids `charm.land/bubbletea/v2` + `charm.land/bubbles/v2` from the six guarded packages — including `internal/daemon`** — so it stays green *by construction* once the deps are added. Data layers produce plain structs the UI consumes: the daemon **registry lister** (charm-free, in `internal/daemon`) and `agents.AllTargets()` / `agents.DetectAll()` feed the picker views. This is the exact Phase-6 "producers plain, `present` renders" seam extended to interactive components.
- **D-10:** Every interactive entry point **TTY-gates via the SAME `present.ChoosePresentation` / `term.IsTerminal` seam BEFORE calling `tea.NewProgram()`**. Not a TTY (piped / CI / `NO_COLOR`) ⇒ the non-interactive fallback (D-12/D-13) — never a blocked stdin read. This is the literal "TTY-gate before `tea.NewProgram()`" the ROADMAP Notes mandate.
- **D-11:** Add **`charm.land/bubbletea/v2` + `charm.land/bubbles/v2`** to `go.mod` (pinned exact versions) — the interactive layer Phase 6 deliberately deferred (`charm.land/lipgloss/v2` is already present). Prefer pure-Go/no-CGo by construction; the full REL-01 closure audit is Phase 8.

**Non-interactive fallbacks (TUI-04)**
- **D-12:** **daemon picker no-TTY fallback:** `codegraph daemon` (no args, non-TTY) prints the **plain-text list of running daemons** (current project first) and exits 0 — a read-only, script-safe listing that never blocks. (This implicitly provides a "list" behavior; whether to also expose an explicit `daemon list` alias is Claude's discretion.)
- **D-13:** **install/uninstall no-TTY fallback:** keep today's behavior — resolve straight to `auto` (install.go's existing default branch), never prompt. `-y`/`--yes` (D-15) forces this same non-interactive path even on a TTY. Empty/EOF stdin degrades to `auto`, never hangs (already true today).

**install / uninstall multi-select (TUI-03)**
- **D-14:** Replace install's current **plain numbered-line prompt** (`promptAgentMultiSelect` in `internal/cli/install.go`) with a **bubbles multi-select (checkbox list)** on a TTY, pre-checking the targets `agents.DetectAll(loc)` reports installed. Do the same for `uninstall` (pre-check installed). The non-TTY / `-y` path reuses `agents.ResolveTargetFlag("auto", loc)`. Keep `--target` / `--location` / `--auto-allow` unchanged. The bubbles UI lives in the cli layer (D-09), so `internal/agents` stays charm-free.
- **D-15:** Add **`-y` / `--yes`** to **both** `install` and `uninstall`: skip the picker and use the non-interactive default set (`auto`), for scripts/CI. Matches TS.

**TEST-03 (byte-invariance + piped never-hang)**
- **D-16:** **githooks byte-invariance** — a focused `internal/githooks` test: install → simulate a user edit *outside* the marker block → remove ⇒ the file returns **byte-identical to the pre-install original** (marker block fully stripped, user content intact). Exercises the Phase-5 strip-then-restore semantics; no new mechanism.
- **D-17:** **piped-stream never-hang** — assertions in the existing **`test/integration/` subprocess harness** (TEST-04's home): spawn the real binary for `daemon` (no args) and `install` with **piped / closed stdin+stdout under a timeout**; assert it exits promptly with the non-interactive output and never blocks. Adding bubbletea must not regress this.

### Claude's Discretion
- Exact bubbles list styling, key bindings, and checkbox glyphs; picker column layout (e.g. repo / pid / age).
- Watchdog poll interval within ~1–2s; the exact reparent-detection predicate (ppid==1 vs ppid≠original) — pick the more robust given subreapers.
- Registry record filename scheme (pid- vs hash-based) and any fields beyond `{pid, startedAt, repoRoot}`.
- Whether to expose an explicit `daemon list` alias (D-12 already covers the behavior).
- Stop signal policy: **recommend graceful `SIGTERM` only** (the daemon's existing signal handling cancels ctx → releases lock → prunes registry; stale records self-heal via D-05). A grace-timeout + `SIGKILL` escalation is discretion only if a hung daemon proves to be a real problem.

### Deferred Ideas (OUT OF SCOPE)
- **DMON-FUT-01** — true detached / double-forked per-project daemons + unix-socket sharing (full TS auto-spawn parity). A later milestone; v1.0 ships the explicit foreground start/stop + picker + registry + watchdog model only.
- **Explicit `daemon list` subcommand alias** — if not adopted as discretion under D-12, it is a natural low-cost follow-up.
- **SIGKILL grace-timeout escalation** for `stop` — v1.0 recommends graceful SIGTERM only; escalation only if hung daemons prove real.
- **Charm dependency-closure audit** (CGo / govulncheck / SBOM / reproducible double-build, REL-01) → **Phase 8**.
- **`--color` / `--no-color` explicit override flag** — carried from Phase 6; v1.0 relies on TTY-detection + `NO_COLOR`.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| DMON-01 | `codegraph daemon` (no args) opens an interactive picker listing running daemons (current project first) to stop one / stop-all / cancel | Architecture Patterns #1/#3, Code Examples "Daemon picker Model skeleton", Standard Stack (bubbles list.Model + custom delegate) |
| DMON-02 | Explicit `daemon start` / `daemon stop` / `daemon stop --all` manage the shared background daemon lifecycle, no silent auto-spawn | Architecture Patterns "Daemon lifecycle diagram", Don't Hand-Roll (reuse daemon.New/Run/RunWithRetry), Open Question #3 (stop target flag shape) |
| DMON-03 | PPID watchdog shuts down any daemon/in-process watcher when its supervising host/agent dies (POSIX reparent + Windows liveness poll) | Architecture Patterns #5/#6, Code Examples "PPID reparent watchdog" + "Windows parent liveness", Common Pitfalls #2/#4, Environment Availability |
| DMON-04 | Global `~/.codegraph/daemons` registry lets the picker list/stop daemons across projects, self-healing stale records | Architecture Patterns #3 (registry self-heal), Don't Hand-Roll (reuse lock.go's isProcessLive/isStale), Open Question #2 (filename scheme) |
| TUI-03 | `install`/`uninstall` present an interactive multi-select agent picker by default (bubbles), `-y`/`--yes` for non-interactive | Architecture Patterns #2, Code Examples "Checkbox ItemDelegate", Common Pitfalls #5/#6 |
| TUI-04 | Every interactive component auto-falls back to non-interactive when stdin/stdout is not a TTY, never hangs | Architecture Patterns #1 (TTY-gate-before-NewProgram), Common Pitfalls #1, Validation Architecture (piped-stream harness) |
| TEST-03 | git-hook install→edit→remove is byte-invariant; interactive components tested against piped streams | Validation Architecture (Wave 0 gaps: githooks byte-invariance test, integration piped-never-hang cases) |
</phase_requirements>

## Project Constraints (from CLAUDE.md)

- Go (latest stable), single static binary per platform — no change to this constraint; bubbletea v2/bubbles v2 are pure Go (confirmed below), so the single-binary story is unaffected.
- Minimal, audited dependencies; CGo only for the already-justified tree-sitter exception — bubbletea/bubbles add **zero** new CGo surface.
- Charm v2 vanity import is `charm.land/...`, **not** `github.com/charmbracelet/...` — verified below via `go list -m -versions`; the bare (non-`/v2`) path resolves to a different, wrong module (same Phase-6 finding, now applying to bubbletea/bubbles too).
- Full CGo/govulncheck/SBOM/reproducible-build audit for the new Charm closure is explicitly **Phase 8 (REL-01)** — this phase only adds the dependency and keeps the archtest green; it does not run that audit.

## Summary

Phase 7 is almost entirely **composition**, not new invention: every hard mechanism it needs already exists in the codebase from Phases 3–6 — `daemon.New`/`Run`/`RunWithRetry` and the lockfile's `isProcessLive`/`isStale` (Phase 3/4), `fsatomic.WriteFile` (Phase 5), and `present.ChoosePresentation` + `term.IsTerminal` + the TUI-01 archtest (Phase 6). This phase's genuinely new work is: (1) a bubbletea/bubbles-based interactive layer confined to `internal/cli` per the archtest's existing (and already-passing-by-construction) guard on `internal/daemon`; (2) a charm-free global daemon registry that is a straightforward multi-file generalization of the single-lockfile self-heal pattern Phase 4 already established; (3) a PPID watchdog that is a small polling goroutine following the exact `wg.Wait()`-joined-goroutine discipline `daemon.Run` already uses for its watch loop; and (4) restructuring one cobra command (`daemon`) into a tree.

The two libraries this phase actually adds — `charm.land/bubbletea/v2` (currently v2.0.8) and `charm.land/bubbles/v2` (currently v2.1.1) — are verified-current via the Go module proxy and are the same charmbracelet org that already ships the pinned `charm.land/lipgloss/v2 v2.0.5` Phase 6 vetted; they are pure Go (no CGo), and bubbles has **no built-in multi-select/checkbox component** — the "bubbles checkbox list" D-14 specifies is a small custom `list.ItemDelegate` rendering `[x]`/`[ ]` glyphs on top of `list.Model`, not an off-the-shelf widget. The single highest-value architectural fact this research surfaces beyond the CONTEXT.md decisions: **Windows has no POSIX-signal-delivery equivalent for `daemon stop`** (`os.Process.Signal` on Windows only supports `os.Kill`), which the CONTEXT.md's "graceful SIGTERM only" discretion note does not address — see Open Question #1.

**Primary recommendation:** Add `charm.land/bubbletea/v2@v2.0.8` + `charm.land/bubbles/v2@v2.1.1`; put the two bubbletea Models (daemon picker, agent checkbox picker) in a new sibling package `internal/cli/tui/` (keeps `present`'s existing pure-function TTY-gate scope untouched); put the registry (`registry.go`) and watchdog (`watchdog.go` + `watchdog_posix.go`/`watchdog_windows.go`) as new charm-free files inside `internal/daemon`, reusing `lock.go`'s unexported `isProcessLive`/`isStale` directly (same package); gate every `tea.NewProgram()` call on **both** stdin and stdout being real TTYs (not just stdout, which is all the existing `ChoosePresentation` call sites check today) before ever constructing a Program.

## Architectural Responsibility Map

This is a CLI/daemon tool, not a client-server web app — the template's browser/SSR/API/CDN/DB tiers don't apply. The project's own natural tiers are used instead: **CLI Presentation** (cobra RunE + bubbletea Models, `internal/cli`), **Domain/Process Data** (charm-free business logic, `internal/daemon`, `internal/agents`), **OS/Kernel** (signals, ppid, process handles — no Go abstraction, direct syscalls), and **Filesystem** (registry records, lockfile — durable state).

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Daemon picker rendering (list/select/stop/cancel) | CLI Presentation (`internal/cli/tui`) | Domain/Process Data (`internal/daemon` registry reader) | The picker only renders; all pid liveness/staleness logic must stay in the archtest-guarded `internal/daemon` package |
| Install/uninstall checkbox picker rendering | CLI Presentation (`internal/cli/tui`) | Domain/Process Data (`internal/agents`) | Same seam Phase 6 established: producers plain, `present`/`tui` renders |
| Daemon start/stop/stop-all lifecycle | Domain/Process Data (`internal/daemon`) | OS/Kernel (signal delivery) | `daemon.Run`/`RunWithRetry` already own this; `stop` adds a new OS/Kernel-tier signal-send, not new domain logic |
| PPID watchdog (parent-death detection) | OS/Kernel (`Getppid`/parent handle) | Domain/Process Data (`internal/daemon` — ctx cancellation) | The liveness *check* is a pure OS query; the *consequence* (cancel ctx → clean teardown) is domain-owned, reusing the existing shutdown path |
| Global daemon registry (`~/.codegraph/daemons/`) | Filesystem (durable records) | Domain/Process Data (`internal/daemon` register/list/self-heal) | Mirrors the existing per-repo `daemon.lock` split: Filesystem holds state, `internal/daemon` owns the read/write/self-heal logic |
| TTY-gate / non-interactive fallback | CLI Presentation (`internal/cli/present` + `internal/cli` RunE) | — | Pure decision logic already lives in `present.ChoosePresentation`; extending it to also check stdin is still CLI-tier |
| Signal delivery to a remote daemon pid (`stop`) | OS/Kernel (`os.FindProcess`+`Signal`, or Windows equivalent) | CLI Presentation (command surface, target resolution) | The syscall itself is OS/Kernel; deciding *which* pid(s) to target is a CLI-tier concern (current-project default vs `--all`) |

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `charm.land/bubbletea/v2` | v2.0.8 [VERIFIED: Go module proxy + Context7] | TUI event loop (`Model`/`Update`/`View`, `tea.NewProgram`) for the daemon picker and install/uninstall checkbox picker | The org (charmbracelet) already supplies this project's pinned `lipgloss/v2` (Phase 6); same vanity-import migration, same maintainers, dominant Go TUI framework |
| `charm.land/bubbles/v2` | v2.1.1 [VERIFIED: Go module proxy + Context7] | Reusable component primitives — `list.Model` is the base for both pickers | Same org/ecosystem as bubbletea/lipgloss; `list.Model` already solves viewport sizing, keybinding, pagination, filtering — only the item-render/toggle logic needs to be custom (see Don't Hand-Roll) |
| `charm.land/lipgloss/v2` | v2.0.5 (already in go.mod, Phase 6) [VERIFIED: existing go.mod] | Styling for the new interactive views (borders, selected-row highlight) | Already vetted and pinned; reused, not re-audited |
| `golang.org/x/term` | v0.45.0 (already in go.mod) [VERIFIED: existing go.mod] | `term.IsTerminal(fd)` — the TTY-gate every interactive entry point checks BEFORE `tea.NewProgram()` | Already the project's sole TTY-detection dependency (Phase 6 D-03); no new library needed |
| `golang.org/x/sys` | v0.47.0 (already indirect in go.mod) [VERIFIED: existing go.mod] | Promote to **direct** for the `windows` build-tagged watchdog/stop files (`windows.OpenProcess`, `windows.WaitForSingleObject`) | Already transitively required (via charmbracelet/x/term etc.); promoting to direct is a `go.mod` metadata change, not a new supply-chain dependency — mirrors `internal/graphstore/locked_windows.go`'s existing precedent of hand-rolling minimal Windows syscalls rather than pulling in a heavier wrapper |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `go.uber.org/goleak` | v1.3.0 (already in go.mod, gates `internal/daemon`'s `TestMain`) | Keeps the new watchdog goroutine leak-free under test | Every new goroutine in `internal/daemon` (the watchdog's poll loop) must join via a `stop()`/`Wait()` before test teardown — reuse the exact discipline `daemon.Run`'s `wg.Wait()` already enforces |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Hand-rolled `list.ItemDelegate` checkbox multi-select on `bubbles/v2/list` | `github.com/charmbracelet/huh` (forms library, has a native multi-select field) | `huh` would give a ready-made multi-select widget with less custom code, but it is a **separate, additional charm dependency** beyond what CONTEXT.md D-11/D-14 names ("bubbles multi-select (checkbox list)"); introducing it would need its own legitimacy/CGo check and isn't what the locked decision specifies. Reasonable to reconsider in a later phase if the hand-rolled delegate proves fiddly. |
| POSIX `SIGTERM` via `os.FindProcess(pid).Signal(...)` for `daemon stop` | `golang.org/x/sys/unix.Kill(pid, syscall.SIGTERM)` directly | Functionally identical on POSIX; stdlib `os.FindProcess`+`Signal` is what `lock.go`'s existing `isProcessLive` already uses (Signal(0)) — reusing the same call shape for a real SIGTERM keeps one signaling idiom in the package instead of two. |
| Poll-based PPID watchdog (D-07, locked) | Linux `prctl(PR_SET_PDEATHSIG)` (signal-on-parent-death, no polling) | `PR_SET_PDEATHSIG` is Linux-only (no macOS/Windows equivalent — this project's `procstart_*` split already treats "linux" vs "everything else" as materially different, and macOS is a first-class release target per `release.yml`'s 3-OS matrix), and has a well-documented startup race (parent can die between fork and the `prctl` call, before the signal arm is set) that a periodic reparent-check does not have. CONTEXT.md already locked the poll-based approach (D-07); this alternative is documented for completeness, not as a live option. |
| `internal/cli/tui` as a new sibling package (recommended) | Extend `internal/cli/present` in place | Both are explicitly allowed by D-09. `present.go`'s `ChoosePresentation` is a small, pure, side-effect-free function today; folding bubbletea Models into the same package would mix "pure decision logic" with "stateful interactive rendering" in one package's public surface. A sibling `internal/cli/tui` (still excluded from the archtest's guarded closure, since the exclusion is `internal/cli` and everything under it) keeps that separation without weakening any guarantee. |

**Installation:**
```bash
go get charm.land/bubbletea/v2@v2.0.8
go get charm.land/bubbles/v2@v2.1.1
go mod tidy -e   # -e required: pre-existing tree-sitter-swift resolution error, unrelated (see STATE.md Phase 6 note)
```

**Version verification:** confirmed via `go list -m -versions charm.land/bubbletea/v2` (latest: v2.0.8, stable — no `-alpha`/`-beta`/`-rc` suffix) and `go list -m -versions charm.land/bubbles/v2` (latest: v2.1.1). Both queried directly against the Go module proxy in this research session — not training-data versions. Context7 (`/charmbracelet/bubbletea`, `/charmbracelet/bubbles`) cross-confirms the v2 API shape (`tea.KeyPressMsg`, `list.Model`, `list.ItemDelegate`) documented below.

## Package Legitimacy Audit

> The `gsd-tools package-legitimacy check` seam only supports `--ecosystem npm|pypi|crates` — Go modules are out of scope for that tool. The Go-ecosystem equivalent verification below uses `go list -m -versions` against the real Go module proxy (the authoritative registry for this ecosystem) plus Context7's official-docs cross-check, which is the same evidentiary bar the seam would apply.

| Package | Registry | Age / Provenance | Downloads/Adoption | Source Repo | Verdict | Disposition |
|---------|----------|-----|-----------|-------------|---------|-------------|
| `charm.land/bubbletea/v2` (→ `github.com/charmbracelet/bubbletea`) | Go module proxy [VERIFIED] | v2 line GA'd 2025 (post alpha/beta/rc series visible in `go list -m -versions`); bubbletea itself is a multi-year, org-maintained project | One of the most widely adopted Go TUI frameworks; same org already vetted for `lipgloss/v2` in Phase 6 | github.com/charmbracelet/bubbletea | OK | Approved |
| `charm.land/bubbles/v2` (→ `github.com/charmbracelet/bubbles`) | Go module proxy [VERIFIED] | Same org/release cadence as bubbletea v2 | Standard companion library to bubbletea; ships alongside it in virtually every charm-based Go TUI | github.com/charmbracelet/bubbles | OK | Approved |

**Packages removed due to [SLOP] verdict:** none.
**Packages flagged as suspicious [SUS]:** none. Both packages are from the same already-vetted charmbracelet org (Phase 6 approved `lipgloss/v2` from this org) and are verified current via the Go module proxy directly in this session — no unverified WebSearch/training-data package names are introduced by this phase.

## Architecture Patterns

### System Architecture Diagram

```
                    ┌─────────────────────────────┐
                    │  cobra RunE (internal/cli)   │
                    └──────────────┬───────────────┘
                                   │
                                   ▼
              ┌─────────────────────────────────────────┐
              │  TTY gate (BOTH stdin AND stdout):       │
              │  term.IsTerminal(stdin fd) &&            │
              │  term.IsTerminal(stdout fd)               │
              └──────────────┬───────────────┬───────────┘
                    TTY       │               │  non-TTY / -y / NO_COLOR
                              ▼               ▼
              ┌───────────────────────┐   ┌─────────────────────────────┐
              │ tea.NewProgram(Model) │   │ Plain fallback:              │
              │  (internal/cli/tui)   │   │  - daemon: print list, exit0 │
              └──────────┬────────────┘   │  - install/uninstall: auto   │
                         │                 └─────────────────────────────┘
                         ▼
      ┌──────────────────────────────────────────────────┐
      │  Model.Init / Update / View loop                  │
      │  reads plain data from charm-free producers:       │
      │    internal/daemon.ListRegistry()  (daemon picker) │
      │    agents.AllTargets()+DetectAll() (agent picker)  │
      └──────────────────────┬─────────────────────────────┘
                              │ user selects: stop-one / stop-all / cancel
                              │               or: toggle checkboxes / confirm
                              ▼
      ┌──────────────────────────────────────────────────┐
      │  Charm-free action (internal/daemon, internal/agents):│
      │    daemon.Stop(pid) / daemon.StopAll()             │
      │    agents.Install(target, opts) / Uninstall(target)│
      └──────────────────────┬─────────────────────────────┘
                              ▼
                    cmd.OutOrStdout()/Stderr — result printed
```

Daemon lifecycle (DMON-01/02/03/04), separate from the picker's rendering concern:

```
codegraph daemon start [-p path]
  daemon.New(repoRoot) → daemon.Run(ctx)
    watch.WatchDisabledReason gate (existing, unchanged)
    acquire() lockfile (existing, unchanged)
    registry.Register({pid, startedAt, repoRoot})        ← NEW (D-04/D-06)
    watchdog.Start(ctx, cancel, interval)                 ← NEW (D-07/D-08), joined goroutine
    watch.Open + debounce + flush loop (existing, unchanged)
    on ctx.Done(): release() lock; registry.Deregister()  ← NEW, deferred like release()

codegraph daemon stop [-p path | --all]
  registry.List()  → self-heals stale records (D-05, reuses isStale/isProcessLive)
  resolve target pid(s): current-project match, or every live record if --all
  send graceful stop signal per target (POSIX: SIGTERM; Windows: see Open Question #1)
  → target's own Run(ctx) sees ctx cancel via its signal.NotifyContext, tears down cleanly

codegraph daemon   (bare, TTY)
  registry.List() → self-heal → sort current-repo-first
  tea.NewProgram(daemonPickerModel) → user picks stop-one/stop-all/cancel → same stop path above

codegraph daemon   (bare, non-TTY)
  registry.List() → self-heal → print plain "pid  repo  started" table → exit 0
```

### Recommended Project Structure

```
internal/daemon/
├── daemon.go            # unchanged: New/Run/RunWithRetry/flush
├── lock.go              # unchanged: acquire/release/isProcessLive/isStale — registry.go calls these directly (same package)
├── procstart_linux.go   # unchanged
├── procstart_other.go   # unchanged
├── registry.go          # NEW: Record{PID,StartedAt,RepoRoot}; Register/Deregister/List (self-heals via isStale)
├── watchdog.go           # NEW: Start(ctx, cancel, interval) — spawns+joins the poll goroutine; platform-independent orchestration
├── watchdog_posix.go     # NEW: build linux,darwin,... — parentAlive() via captured-original-ppid + os.Getppid()
├── watchdog_windows.go   # NEW: build windows — parentAlive() via windows.OpenProcess+WaitForSingleObject
├── stop_posix.go         # NEW: sendStop(pid) — os.FindProcess(pid).Signal(syscall.SIGTERM)
└── stop_windows.go       # NEW: sendStop(pid) — documented-divergence hard-kill (see Open Question #1)

internal/cli/
├── daemon.go             # RESTRUCTURED: newDaemonCmd() returns a cobra tree — bare RunE (TTY-gate→picker/plain), + AddCommand(newDaemonStartCmd(), newDaemonStopCmd())
├── install.go            # promptAgentMultiSelect's *input* mechanism replaced by tui.RunAgentPicker; resolution/reporting pipeline (selectByIndices, printAgentResults) UNCHANGED per D-14; new -y/--yes flag
├── uninstall.go          # same picker reused (pre-checked from DetectAll); new -y/--yes flag

internal/cli/tui/         # NEW sibling package (D-09) — the ONLY place bubbletea/bubbles are imported
├── daemonpicker.go       # bubbletea Model: list of daemon.Record, stop-one/stop-all/cancel actions
└── agentpicker.go        # bubbletea Model: list.Model + custom checkbox ItemDelegate over agents.AgentTarget

internal/cli/present/     # UNCHANGED — stays pure TTY-gate + non-interactive renderers (status/files/progress)
```

### Pattern 1: TTY-gate before `tea.NewProgram()` — gate BOTH stdin and stdout

**What:** Every call site that might construct a `tea.Program` must first prove both the input and output file descriptors are real terminals — not just stdout (which is all `present.ChoosePresentation`'s existing call sites check today, since they only ever *write*).
**When to use:** `codegraph daemon` (bare), `install`/`uninstall` (unless `-y`).
**Example:**
```go
// internal/cli/present/tty.go already has:
//   func ChoosePresentation(isTTY bool, noColor string) bool
// Interactive components need an analogous, equally pure gate that
// additionally requires STDIN to be a real, readable terminal — reusing
// install.go's existing installStdinIsInteractive shape (os.ModeCharDevice)
// alongside status.go's term.IsTerminal(fd) call for stdout.

func stdinIsRealTTY(cmd *cobra.Command) bool {
	if cmd.InOrStdin() != os.Stdin {
		return false
	}
	fi, err := os.Stdin.Stat()
	return err == nil && fi.Mode()&os.ModeCharDevice != 0
}

func interactiveAllowed(cmd *cobra.Command) bool {
	stdoutFd := int(os.Stdout.Fd())
	return stdinIsRealTTY(cmd) && term.IsTerminal(stdoutFd) && os.Getenv("NO_COLOR") == ""
}
```
Call this BEFORE `tea.NewProgram(...)`, never after — `Program.Run()`'s own error return is not a substitute (see Common Pitfalls #1).

### Pattern 2: Custom checkbox `list.ItemDelegate` for multi-select

**What:** `bubbles/v2/list` ships a single-select-oriented `list.Model` (one "cursor" index) — there is no built-in multi-select/checkbox delegate. D-14's "bubbles multi-select (checkbox list)" is achieved by a small custom `ItemDelegate` that tracks a `map[int]bool` of checked indices and renders `[x]`/`[ ]`.
**When to use:** install/uninstall's agent picker.
**Example:**
```go
// Source: charm.land/bubbles/v2/list ItemDelegate interface (Context7,
// github.com/charmbracelet/bubbles/list/list.go) — the interface this
// delegate implements is verbatim from the official docs; the
// checkbox-toggle logic itself is this project's own addition (no
// off-the-shelf bubbles multi-select delegate exists).
type agentItem struct {
	target agents.AgentTarget
}

func (i agentItem) FilterValue() string { return i.target.DisplayName() }

type checkboxDelegate struct {
	checked map[int]bool // index -> selected
}

func (d checkboxDelegate) Height() int  { return 1 }
func (d checkboxDelegate) Spacing() int { return 0 }

// Update is list.Model's per-item message hook (called for every message
// except while the user is typing into the filter) — the sanctioned place
// to intercept "space toggles this row" without fighting list.Model's own
// KeyMap (see Common Pitfalls #5).
func (d *checkboxDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd {
	if key, ok := msg.(tea.KeyPressMsg); ok && key.String() == "space" {
		i := m.Index()
		d.checked[i] = !d.checked[i]
	}
	return nil
}

func (d checkboxDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	ai := item.(agentItem)
	box := "[ ]"
	if d.checked[index] {
		box = "[x]"
	}
	cursor := "  "
	if index == m.Index() {
		cursor = "> "
	}
	fmt.Fprintf(w, "%s%s %s\n", cursor, box, ai.target.DisplayName())
}
```
Pre-check state comes from `agents.DetectAll(loc)` exactly as `promptAgentMultiSelect` already does — only the *rendering/input* mechanism changes (D-14 is explicit about this: "replace the input mechanism, keep the resolution + reporting").

### Pattern 3: Charm-free registry self-heal (generalizes the existing single-lockfile pattern to many files)

**What:** `~/.codegraph/daemons/` holds one record file per live daemon. `List()` reads every file, applies `lock.go`'s existing `isStale`/`isProcessLive` (same package, unexported functions — no new liveness logic), and deletes any record found stale — mirroring `acquire()`'s existing "detect+clear stale on every independent call" discipline (Phase 4 D-16), just applied per-file instead of to a single lockfile.
**When to use:** every read of the registry — picker open, plain-list fallback, `stop --all` target resolution.
**Example:**
```go
// internal/daemon/registry.go (new file, same package as lock.go —
// isProcessLive/isStale are called directly, no export needed)

type Record struct {
	PID       int       `json:"pid"`
	StartedAt time.Time `json:"startedAt"`
	RepoRoot  string    `json:"repoRoot"`
}

func registryDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".codegraph", "daemons"), nil
}

// List reads every record file, self-heals (removes) any that fail the
// SAME isStale check acquire() already applies to the per-repo lockfile —
// no second liveness implementation. Returns only live records.
func List() ([]Record, error) {
	dir, err := registryDir()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var live []Record
	for _, e := range entries {
		path := filepath.Join(dir, e.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue // vanished between ReadDir and ReadFile — another scan raced us
		}
		var rec Record
		if err := json.Unmarshal(data, &rec); err != nil {
			continue // malformed record — treat as unreadable, not live
		}
		if isStale(lockInfo{PID: rec.PID, StartedAt: rec.StartedAt}) {
			_ = os.Remove(path) // self-heal; best-effort, matches release()'s style
			continue
		}
		live = append(live, rec)
	}
	return live, nil
}

// Register writes this daemon's record via the same atomic-write
// primitive D-04 names — no new write mechanism.
func Register(rec Record) error {
	dir, err := registryDir()
	if err != nil {
		return err
	}
	data, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	path := filepath.Join(dir, fmt.Sprintf("%d.json", rec.PID))
	return fsatomic.WriteFile(path, string(data))
}
```
Filename `<pid>.json` is safe against collision because the OS guarantees pid uniqueness among *live* processes at any instant — see Open Question #2 for the full argument and the alternative discretion CONTEXT.md leaves open.

### Pattern 4: Build-tag platform split (mirrors `procstart_linux.go`/`procstart_other.go` and `locked_windows.go`)

**What:** Both the watchdog's liveness check and the stop signal delivery need one POSIX implementation and one Windows implementation, sharing a common exported entry point.
**When to use:** `watchdog_posix.go`/`watchdog_windows.go`, `stop_posix.go`/`stop_windows.go`.
**Example:** the existing `internal/graphstore/locked_windows.go` precedent — a `//go:build windows` file with a minimal, direct syscall wrapper (no heavyweight abstraction library) — is the exact shape to replicate; see Code Examples below for the watchdog/stop bodies themselves.

### Pattern 5: Reparent detection with a captured baseline (not a bare `ppid == 1` check)

**What:** capture `os.Getppid()` once at daemon start; on every poll tick, treat *any* change away from that captured value as parent death — not specifically `ppid == 1`.
**Why:** confirmed via research (see Sources) that on Linux, when a process's immediate parent dies, `getppid()` returns the pid of the **nearest living subreaper ancestor** (if one exists, e.g. `tini`, `docker --init`, a supervisor using `PR_SET_CHILD_SUBREAPER`) — **not** always `1`. A bare `ppid == 1` check would miss parent death inside any container/supervisor that registers as a subreaper. D-07 already specifies this correctly ("reparenting... e.g. → 1/init or a subreaper"); this research confirms the specific mechanism that makes the captured-baseline form the robust choice over the naive `ppid==1` form.
**When to use:** `watchdog_posix.go`.

### Anti-Patterns to Avoid

- **Calling `tea.NewProgram()` unconditionally and trusting its own error return to handle a non-TTY gracefully:** bubbletea's input driver negotiates terminal capabilities assuming a real terminal; behavior off a genuine TTY is not a guaranteed-fast error across all platforms/redirects (see Common Pitfalls #1). Always gate BEFORE construction — this is the literal reason D-10 exists.
- **Putting registry/watchdog logic behind (or inside) a bubbletea Model:** the registry and watchdog must be plain, independently testable, charm-free functions the picker Model merely calls — this is what keeps `internal/daemon` archtest-green *and* keeps them testable without a pty.
- **Re-implementing process liveness in `registry.go`:** `lock.go`'s `isProcessLive`/`isStale` already handle PID-reuse corroboration via Linux `/proc` start-time comparison — a second implementation in the same package would silently drift from it over time.
- **Treating `ppid == 1` as the only "parent died" signal:** see Pattern 5 — use the captured-original-ppid comparison instead.
- **Assuming `SIGTERM` is deliverable cross-platform:** Windows has no POSIX signal delivery to an arbitrary external process via `os.Process.Signal` (only `os.Kill` is supported; anything else returns an error) — see Open Question #1. Don't silently no-op or panic on Windows; document the divergence explicitly.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|--------------|-----|
| Multi-select checkbox terminal UI | A hand-rolled ANSI checkbox renderer + raw-mode terminal handling | `bubbles/v2/list.Model` + a small custom `ItemDelegate` | `list.Model` already solves viewport sizing, keybindings, pagination, and filtering; only the render+toggle logic (≈20 lines, Pattern 2) needs to be custom |
| TUI event loop / raw-mode terminal handling | Custom termios raw-mode + ANSI escape-sequence parser | `bubbletea/v2`'s `Program` | Cross-platform raw-mode entry/exit, resize handling, and signal-safe teardown (including on panic, per Context7's documented `Program.Run` recover behavior) are already solved and battle-tested |
| Process liveness check for the registry | A second pid-liveness probe inside `registry.go` | `internal/daemon/lock.go`'s unexported `isProcessLive`/`isStale` (same package, direct call) | Already handles PID-reuse corroboration via `/proc` start-time comparison (WR-02's mitigation) — a second implementation is a correctness/drift risk for zero benefit |
| Atomic registry record writes | Custom temp-file-then-rename | `internal/fsatomic.WriteFile` | Exactly the primitive CONTEXT.md D-04 names by name — already crash-safe, already mode-preserving |
| Windows parent-liveness polling / stop signal | A CGo-wrapped Win32 call, or a heavyweight process-management library | `golang.org/x/sys/windows` (`OpenProcess`/`WaitForSingleObject`) — pure-Go syscall wrapper, already an indirect module in `go.mod` | Zero new CGo, zero new supply-chain surface (promotion from indirect→direct only); mirrors the existing minimal-syscall style of `internal/graphstore/locked_windows.go` |

**Key insight:** this phase's actual engineering surface is small. The registry is a ~40-line generalization of a pattern (`acquire`'s stale-detection) the codebase already has; the watchdog is a ~20-line goroutine following the exact join discipline `daemon.Run`'s watch loop already uses; the only place with real design freedom is the two bubbletea Models, and even those are thin renderers over data the domain layer already exposes (`agents.DetectAll`, and the new `registry.List`).

## Common Pitfalls

### Pitfall 1: `tea.NewProgram` on a non-TTY input can misbehave instead of erroring cleanly
**What goes wrong:** bubbletea's default input driver assumes an interactive terminal and negotiates capabilities (bracketed paste, focus reporting) via escape sequences that expect a terminal to respond. Piped/closed stdin does not behave identically to a real TTY across platforms.
**Why it happens:** the raw-mode/terminal-capability negotiation is inherently terminal-specific; there is no universal "this isn't a terminal, fail fast" contract guaranteed by every OS/redirect combination.
**How to avoid:** NEVER call `tea.NewProgram` unless the TTY-gate (Pattern 1, checking BOTH stdin and stdout) has already passed. Do not rely on `Program.Run()`'s own error return as the safety net — the gate must run first, unconditionally.
**Warning signs:** `codegraph daemon | cat` or a CI job invoking `codegraph install` hangs instead of exiting immediately with plain output.

### Pitfall 2: PPID watchdog is fundamentally incompatible with a future detached daemon
**What goes wrong:** if DMON-FUT-01 (deferred, out of scope) ever double-forks/`setsid`s the daemon to detach it from its launching process, `os.Getppid()` at daemon start would already return `1` (or a subreaper) immediately — the watchdog's captured baseline would be meaningless from t=0.
**Why it happens:** detaching a process is *specifically* severing the parent-child liveness link the watchdog depends on.
**How to avoid:** not a concern for this phase (D-02 explicitly forbids detaching in v1.0) — document this as the reason DMON-FUT-01, when it lands, will need a different liveness mechanism (heartbeat, socket, or an explicit `stop` as the only teardown path).

### Pitfall 3: Windows has no `SIGTERM` — `daemon stop` cannot signal-deliver gracefully to an arbitrary remote pid
**What goes wrong:** `os.Process.Signal(sig)` on Windows only supports `os.Kill` (`TerminateProcess`); any other signal value returns an error ("not supported by windows").
**Why it happens:** Windows has no POSIX signal-delivery model for cross-process signals.
**How to avoid:** see Open Question #1 for the concrete recommendation (accept a hard-kill on Windows for v1.0, self-healing registry tolerates it) — but this MUST be an explicit, documented decision in the plan, not a silent gap discovered during implementation.

### Pitfall 4: watchdog goroutine leak under `goleak`
**What goes wrong:** `internal/daemon` already gates its whole test package on `go.uber.org/goleak`'s `TestMain` (`soak_test.go`) — a watchdog poll goroutine that isn't joined before `Run` returns will intermittently fail that gate.
**Why it happens:** easy to forget when adding a "just poll in the background" goroutine — the existing watch-loop goroutine already had to be retrofitted with exactly this discipline (the `wg.Wait()` in `daemon.Run`).
**How to avoid:** `watchdog.Start` must return a `stop func()` that blocks on a `done` channel closing before returning, exactly mirroring the watch loop's own join pattern; `daemon.Run` must call that `stop()` on every teardown path (ctx cancel AND the `ErrWatcherClosed` abnormal path).

### Pitfall 5: `bubbles/v2/list`'s default keybindings collide with checkbox-toggle intent
**What goes wrong:** `list.Model` binds `/` (filter), arrow/`j`/`k` (navigation), and its own `enter` semantics for the *default* delegate's single-select behavior — none of which is "toggle this row's checkbox."
**Why it happens:** `list.Model` is single-select-oriented by default; a naive `space` binding added to the outer `Model.Update` (rather than the delegate's `Update`) risks double-handling or ordering conflicts with the list's own message routing.
**How to avoid:** implement the toggle inside `ItemDelegate.Update` (Pattern 2) — this is the interface's sanctioned per-item-message hook specifically for delegate-owned key handling, confirmed via Context7's official `ItemDelegate` interface documentation.

### Pitfall 6: `-y`/`--yes` must bypass the TTY check itself, not just skip the render
**What goes wrong:** if `-y` only skips *calling* the picker but the surrounding code still runs `installStdinIsInteractive`-style detection for logging/behavior branching, a CI runner that happens to have a real pty attached (common on some CI providers) could still take an unexpected branch.
**Why it happens:** `-y` is a semantic override ("never interactive"), not merely "the picker isn't needed this time" — the two are easy to conflate when wiring the flag into the existing `switch` in `install.go`'s `RunE`.
**How to avoid:** check `--yes` FIRST in the target-resolution switch, before the TTY branch, so it always short-circuits straight to `agents.ResolveTargetFlag("auto", loc)` regardless of what stdin/stdout actually are.

### Pitfall 7: an accidental charm import from `internal/daemon` breaks the TUI-01 archtest build
**What goes wrong:** any debug/logging convenience (e.g., rendering a colored status line inside `registry.go` or `watchdog.go`) that imports `charm.land/lipgloss/v2` or the new bubbletea/bubbles packages fails `TestNoCharmInServeReachablePackages` (`internal/cli/present/archtest/import_graph_test.go`) — it already lists `internal/daemon` in `guardedPackages`.
**How to avoid:** keep `registry.go`/`watchdog.go`/`stop_*.go` producing only plain Go values (structs, strings, errors); all rendering happens exclusively in the new `internal/cli/tui` package.

## Code Examples

### PPID reparent watchdog (POSIX)
```go
// internal/daemon/watchdog_posix.go
//go:build !windows

package daemon

import (
	"os"
	"time"
)

func parentChanged(original int) bool {
	return os.Getppid() != original
}
```

```go
// internal/daemon/watchdog.go — platform-independent orchestration,
// calling the build-tag-split parentChanged/parentAlive predicate.
func startWatchdog(ctx context.Context, cancel context.CancelFunc, interval time.Duration) (stop func()) {
	original := os.Getppid()
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if parentChanged(original) { // POSIX predicate (this file's sibling)
					cancel()
					return
				}
			}
		}
	}()
	return func() { <-done } // join, mirroring daemon.Run's wg.Wait() discipline
}
```

### Windows parent liveness (build-tag sibling, not a reparent check — Windows has no `getppid` reparent semantics)
```go
// internal/daemon/watchdog_windows.go
//go:build windows

package daemon

import "golang.org/x/sys/windows"

// parentAlive probes whether the process identified by originalParentPID
// (captured once, at daemon start, via a Windows-specific parent-pid
// lookup — Windows has no os.Getppid()) is still running, via
// OpenProcess+WaitForSingleObject(0) — WAIT_TIMEOUT means still running,
// WAIT_OBJECT_0 means it has exited. Source: golang.org/x/sys/windows
// (already an indirect go.mod dependency); pattern confirmed against
// community/Microsoft documentation of the OpenProcess+WaitForSingleObject
// idiom (Sources section) — GetExitCodeProcess's STILL_ACTIVE sentinel is
// explicitly NOT used here per Microsoft's own documented caveat that it
// is unreliable for liveness checks.
func parentAlive(pid int) bool {
	h, err := windows.OpenProcess(windows.SYNCHRONIZE, false, uint32(pid))
	if err != nil {
		return false // can't open — treat as gone
	}
	defer windows.CloseHandle(h)
	ev, err := windows.WaitForSingleObject(h, 0)
	return err == nil && ev == uint32(windows.WAIT_TIMEOUT)
}
```

### Checkbox `ItemDelegate` — see Architecture Patterns #2 above (full example inline there).

### Registry self-heal `List()` — see Architecture Patterns #3 above (full example inline there).

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|---------------|--------|
| `codegraph daemon` = single foreground `RunE` calling `daemon.Run` directly | `codegraph daemon` = cobra sub-tree (bare picker + `start`/`stop`/`stop --all`) | This phase (D-01) | Resolves the TS `daemon` name collision (TS: picker; ours: was foreground server); matches TS's command surface expectation |
| `install`'s plain numbered-line prompt (`promptAgentMultiSelect`) reading a raw line from stdin | `bubbles/v2/list.Model` + custom checkbox `ItemDelegate` on a TTY | This phase (D-14) | Matches TS's bubbles-based interactive picker; the underlying resolution/reporting pipeline (`selectByIndices`, `printAgentResults`) is unchanged |
| No cross-project daemon visibility — each repo's `daemon.lock` is invisible outside that repo | `~/.codegraph/daemons/` global registry, self-healing on read | This phase (D-04/D-05) | Enables the picker's "current project first" cross-repo listing (DMON-01) |
| A daemon persists until explicitly stopped or it crashes — no automatic cleanup if its supervising host/agent process exits | PPID watchdog cancels ctx (clean teardown: release lock, deregister, join goroutines) on parent death | This phase (D-07/D-08) | Prevents leaked daemons from an exited host/agent session — the motivating DMON-03 problem |

**Deprecated/outdated:**
- `promptAgentMultiSelect`'s *input-reading* mechanism (raw `bufio.Reader` line parse) is superseded by the bubbles checkbox picker, but per D-14 the function's surrounding resolution logic (`selectByIndices`) and the shared `printAgentResults` reporting loop are explicitly retained, not replaced.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|----------------|
| A1 | Registry record filename scheme is `<pid>.json` (relying on OS pid-uniqueness-among-live-processes to rule out write-time collisions) | Architecture Patterns #3, Open Question #2 | LOW — CONTEXT.md explicitly leaves this to Claude's discretion; if a different scheme is preferred during plan review, it's a small, isolated change confined to `registry.go`'s two functions |
| A2 | `daemon stop`'s non-interactive target-selection flag is `-p`/`--path` (mirroring `daemon start`) plus `--all`, with no explicit `--pid` flag | Architecture Patterns "daemon lifecycle diagram", Open Question #3 | LOW — not locked by CONTEXT.md's decisions; a CLI ergonomics choice easily adjusted during plan review without touching any locked behavior |
| A3 | New sibling package `internal/cli/tui` (rather than extending `internal/cli/present` in place) | Recommended Project Structure, Alternatives Considered | LOW — D-09 explicitly allows either; purely organizational, no behavioral risk |
| A4 | Windows `daemon stop` accepts a hard-kill (`TerminateProcess`) divergence from POSIX's graceful `SIGTERM`, tolerated because the registry/lockfile self-heal on the next scan regardless | Open Question #1, Common Pitfalls #3 | MEDIUM — this is a genuine behavioral gap CONTEXT.md's "graceful SIGTERM only" discretion note does not address for Windows; if the plan wants a softer Windows story (e.g., a control-channel), that's materially more implementation work than this research assumes, and should be explicitly confirmed rather than silently built to the cheaper option |

**On package versions:** `charm.land/bubbletea/v2@v2.0.8` and `charm.land/bubbles/v2@v2.1.1` are NOT in this table — they were confirmed via a direct `go list -m -versions` tool call against the Go module proxy in this session (an authoritative source), satisfying the VERIFIED bar rather than ASSUMED.

## Open Questions

1. **Windows `daemon stop` graceful-shutdown semantics**
   - What we know: `os.Process.Signal` on Windows only supports `os.Kill`; there is no POSIX `SIGTERM` delivery to an arbitrary external process. `golang.org/x/sys/windows` offers `TerminateProcess` (hard-kill, immediate) but nothing that triggers the target's own `signal.NotifyContext`-based graceful teardown remotely without extra plumbing the target process must itself set up (e.g., listening on a named pipe/local socket, or being started in its own process group so `GenerateConsoleCtrlEvent(CTRL_BREAK_EVENT)` can reach it).
   - What's unclear: whether v1.0 should accept the hard-kill divergence (simple, self-healing via the existing stale-lock/stale-registry detection on the next scan) or invest in a softer mechanism now.
   - Recommendation: **accept the hard-kill divergence for v1.0** and document it explicitly in the plan/user-facing help text — the registry and lockfile already tolerate an ungracefully-killed daemon (next scan's `isStale`/`isProcessLive` check cleans it up), matching the same risk tolerance CONTEXT.md's own SIGKILL-escalation discretion note accepts for the POSIX case. Revisit only if this proves to be a real problem in practice (mirrors DMON-FUT-01's own "later, if needed" framing).

2. **Registry record filename scheme**
   - What we know: CONTEXT.md D-04 explicitly defers this to Claude's discretion. A pid-keyed filename (`<pid>.json`) is simple and correct because the OS never assigns the same pid to two simultaneously-live processes — a new daemon writing `<pid>.json` can only ever "collide" with a record for a process that is, by definition, no longer alive (since this new process now legitimately holds that pid), which the very next self-heal scan would have pruned anyway as stale.
   - What's unclear: whether a hash-based scheme (e.g., `sha256(repoRoot)[:12].json`) might be preferred for easier manual inspection/debugging (`ls ~/.codegraph/daemons/` grouped by repo rather than by pid).
   - Recommendation: use `<pid>.json` (Pattern 3's example) — simplest, and the collision argument above is airtight; flag for plan-review sign-off since it is a concrete design commitment this research makes on the user's behalf.

3. **`daemon stop`'s explicit target-selection flag surface**
   - What we know: CONTEXT.md locks the existence of `daemon stop` / `daemon stop --all` (D-01/D-02) but not how a non-interactive `daemon stop` (no `--all`) picks its single target.
   - What's unclear: whether it should default to the current working directory's repo (mirroring `daemon start`'s `-p/--path` convention) or require an explicit `--pid`/positional argument.
   - Recommendation: default to the current-directory repo via the same `-p/--path` flag `daemon start` already has (consistency with the existing command); this is a low-risk assumption (A2) easily corrected in plan review if a different UX is preferred.

4. **Explicit `daemon list` alias**
   - What we know: CONTEXT.md D-12 already delivers this behavior implicitly (the non-TTY fallback of bare `daemon`); an explicit alias is explicitly named as discretion/deferred-idea territory.
   - What's unclear: whether adding the alias in this phase is worth the (small) surface growth.
   - Recommendation: do NOT add it in this phase — it is not in REQUIREMENTS.md's DMON-01..04/TUI-03/04/TEST-03 set, and CONTEXT.md's Deferred Ideas section already frames it as a natural low-cost follow-up, not something this phase needs to ship.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| `charm.land/bubbletea/v2` module | DMON-01, TUI-03/04 | ✓ (resolvable via Go module proxy, confirmed this session) | v2.0.8 | — |
| `charm.land/bubbles/v2` module | DMON-01, TUI-03 | ✓ | v2.1.1 | — |
| A real TTY for interactive testing | Manual verification of the picker's actual look/feel | ✗ in this research session's non-interactive shell, and ✗ in CI (all CI jobs run non-interactively per `ci.yml`) | — | Automated tests exercise only the TTY-gate branch selection and the non-TTY fallback paths (this is already TUI-02's precedent, per `status_files_plain_test.go`'s doc comment: "a subprocess piped into a bytes.Buffer can never satisfy [isTTY]"); the interactive rendering itself is necessarily a manual/`checkpoint:human-verify` concern, not an automatable CI assertion |
| A Windows runner for testing `watchdog_windows.go`/`stop_windows.go` | DMON-03 (Windows liveness poll) | ✗ — `ci.yml` runs exclusively on `ubuntu-latest`; there is no Windows test runner in CI (confirmed: only `release.yml`'s cross-compile matrix touches `GOOS=windows`, and that only *builds*, never *runs*, Windows binaries) | — | Existing project precedent (`internal/graphstore/locked_windows.go`) is compiled-but-untested-in-CI via a dedicated `GOOS=windows GOARCH=amd64 go vet ./internal/graphstore/` typecheck-only CI step (`ci.yml` line ~124, "Typecheck windows lock classifier"); the plan should add `internal/daemon` to that same `go vet` line (or a sibling line) so the new Windows-only files are at least compile-checked in CI, with actual runtime behavior left to a documented manual-verification step |
| A macOS test runner | DMON-03 (POSIX watchdog — `!windows` build tag covers darwin too) | ✓ — this research session's own environment is `darwin/arm64`; `release.yml`'s build matrix includes darwin targets | go1.26.5 | — |

**Missing dependencies with no fallback:** none — every dependency this phase needs is either already resolvable (bubbletea/bubbles via the module proxy) or has an established fallback pattern (Windows compile-only CI check).

**Missing dependencies with fallback:**
- Windows runtime test coverage → compile-only `go vet` CI gate (existing precedent) + manual verification, not full CI automation.
- Live-terminal interactive-rendering verification → non-TTY-path automated tests (TTY-gate branch selection, piped never-hang) cover everything CI can assert; actual visual/keybinding UX needs a manual pass.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go stdlib `testing` + `go.uber.org/goleak` (already gating `internal/daemon`) |
| Config file | none — plain `go test`; CI orchestration lives in `.github/workflows/ci.yml` |
| Quick run command | `go test ./internal/daemon/... ./internal/cli/... -run <Test>` |
| Full suite command | `go test ./... && go test ./testdata/golden/... && go test ./test/integration/...` (mirrors `ci.yml`'s existing three-part invocation, since `testdata/` and `test/integration/` are both excluded from a bare `go test ./...` per this project's own documented `GOLDEN-01`/TEST-04 lessons) |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|---------------------|--------------|
| DMON-01 | Bare `daemon` on a TTY opens the bubbletea picker (current project first) | unit (Model.Update state transitions) + manual (visual) | `go test ./internal/cli/tui/... -run TestDaemonPicker` | ❌ Wave 0 |
| DMON-01 | Bare `daemon` off-TTY prints a plain list, exit 0 | integration (piped subprocess) | `go test ./test/integration/... -run TestDaemonBarePlainList` | ❌ Wave 0 |
| DMON-02 | `daemon start`/`stop`/`stop --all` manage lifecycle, no auto-spawn | unit (`internal/daemon` — registry register/deregister across a Run cycle) | `go test ./internal/daemon/... -run TestDaemonStartStop` | ❌ Wave 0 |
| DMON-03 | PPID watchdog cancels ctx on captured-ppid change (POSIX) | unit (fake/injectable ppid source, mirrors `onSyncStart`-style test seams) | `go test ./internal/daemon/... -run TestWatchdogCancelsOnReparent` | ❌ Wave 0 |
| DMON-03 | Windows liveness poll compiles and typechecks | compile-only (no Windows runner) | `GOOS=windows GOARCH=amd64 go vet ./internal/daemon/` | ❌ Wave 0 (extend existing `ci.yml` vet line) |
| DMON-04 | Registry self-heals a stale record on `List()` | unit (`internal/daemon` — write a record for a dead pid, assert it's pruned) | `go test ./internal/daemon/... -run TestRegistryListPrunesStale` | ❌ Wave 0 |
| TUI-03 | install/uninstall checkbox picker pre-checks detected agents, toggles, resolves to the same install/uninstall pipeline | unit (`internal/cli/tui` — delegate toggle logic) + existing `install_test.go` resolution tests (unchanged) | `go test ./internal/cli/tui/... ./internal/cli/... -run TestAgentPicker` | ❌ Wave 0 (delegate); ✓ existing (resolution pipeline, `install_test.go`) |
| TUI-04 | Every interactive component falls back cleanly off-TTY, never hangs | integration (piped subprocess under a timeout) | `go test ./test/integration/... -run TestPipedNeverHang` | ❌ Wave 0 (new cases for `daemon` bare + `install`) |
| TEST-03 | githooks install→edit→remove is byte-invariant | unit (`internal/githooks`) | `go test ./internal/githooks/... -run TestInstall_EditThenRemove_ByteInvariant` | ❌ Wave 0 — closest existing test (`TestRemove_WithUserContent_PreservesRemainderBytes`) covers *remove* preserving remainder, not the full install→edit-outside-marker→remove == pre-install-original round trip D-16 specifies |
| TEST-03 | interactive components tested against piped streams under a timeout | integration | same `TestPipedNeverHang` as TUI-04 above | ❌ Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./internal/daemon/... ./internal/cli/... ./internal/cli/tui/... ./internal/githooks/...`
- **Per wave merge:** `go build ./... && GOOS=windows GOARCH=amd64 go vet ./internal/daemon/ ./internal/graphstore/ && go test ./... && go test ./testdata/golden/... && go test ./test/integration/...`
- **Phase gate:** Full suite green before `/gsd-verify-work`, including the extended Windows `go vet` line covering the new `internal/daemon` Windows-tagged files.

### Wave 0 Gaps
- [ ] `internal/daemon/registry_test.go` — DMON-04 register/list/self-heal
- [ ] `internal/daemon/watchdog_test.go` — DMON-03 POSIX reparent-cancel behavior, injectable ppid source for determinism (mirror the existing `onSyncStart`/`onWatchOpen` test-seam convention in `daemon.go`)
- [ ] `internal/daemon/stop_test.go` (POSIX) — signal delivery to a real short-lived test process
- [ ] `internal/cli/tui/daemonpicker_test.go` — DMON-01 Model.Update transitions (stop-one/stop-all/cancel), no real pty needed (bubbletea Models are directly unit-testable by feeding `Update` synthetic `tea.Msg` values)
- [ ] `internal/cli/tui/agentpicker_test.go` — TUI-03 checkbox delegate toggle + pre-check-from-DetectAll logic
- [ ] `internal/cli/daemon_test.go` additions — cobra tree wiring (`daemon start`/`stop`/`stop --all` route to the right functions; bare `daemon` TTY-gates correctly)
- [ ] `internal/githooks/githooks_test.go` addition — `TestInstall_EditThenRemove_ByteInvariant` (D-16, the genuine gap identified above)
- [ ] `test/integration/piped_never_hang_test.go` (new file) — D-17: spawn the real binary for `daemon` (no args) and `install`, with closed/piped stdin AND stdout, under a bounded `context.WithTimeout` (mirrors `watch_default_test.go`'s existing 30s-timeout convention), asserting prompt exit + non-interactive output
- [ ] Extend `.github/workflows/ci.yml`'s existing `GOOS=windows GOARCH=amd64 go vet ./internal/graphstore/` line (or add a sibling line) to also cover `./internal/daemon/` once `watchdog_windows.go`/`stop_windows.go` exist — Framework install: none needed, this is a one-line CI config change, not a new framework

*(Wave 0 gaps are non-trivial for this phase — DMON-03/04 and TUI-03/04 are all genuinely new test surfaces; TEST-03's githooks half has a real, specific gap identified above rather than being fully covered by existing tests.)*

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|----------------|---------|--------------------|
| V2 Authentication | no | No new authentication surface — the daemon registry and picker operate entirely within the invoking user's own OS session/filesystem permissions, same trust boundary as the existing per-repo lockfile |
| V3 Session Management | no | N/A — no sessions introduced |
| V4 Access Control | partial | `~/.codegraph/daemons/` is a per-user directory (created via `os.UserHomeDir()`, standard `0o755`/`0o644` file modes matching `fsatomic.WriteFile`'s existing defaults) — the OS's own filesystem permissions are the access-control boundary, same trust model as `.codegraph/daemon.lock` today. `daemon stop`'s signal delivery is bounded by the OS's own cross-process-signal permission model (a user can only signal their own processes without elevated privilege) — no new privilege escalation surface. |
| V5 Input Validation | yes | Registry record JSON (`Record{PID, StartedAt, RepoRoot}`) is parsed via `encoding/json.Unmarshal` into a fixed struct — malformed/malicious record content on disk is not evaluated as code and, per Pattern 3's `List()` example, a decode failure is treated as "unreadable, skip" rather than propagated as a fault. The picker's `RepoRoot` field is rendered as plain text (never shell-interpolated or passed to `exec.Command` as anything other than an opaque display string) — no injection surface. |
| V6 Cryptography | no | No new cryptographic operations — signing/SBOM/provenance for the new Charm dependency closure is explicitly Phase 8 (REL-01), not this phase |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|-----------------------|
| A malicious/corrupted registry record file (e.g., a crafted `pid` value pointing at an unrelated live process, or a `repoRoot` containing control characters) causing `daemon stop`/`stop --all` to signal an unintended process | Tampering / Elevation of Privilege | The OS's own signal-permission model already prevents signaling a process the invoking user doesn't own — a crafted `pid` can at worst target another of the SAME user's own processes, no different from the existing risk surface of running `kill <pid>` manually. `List()`'s `isStale` check (Pattern 3) additionally requires the target pid's OS-observed start time to plausibly match the record's `StartedAt` (the same `/proc` corroboration `lock.go` already applies) before treating a record as live — reducing (not eliminating) the chance a stale, tampered, or reused-pid record is acted on. Registry files live under `~/.codegraph/` (0600-ish home-directory permissions, same trust boundary as SSH keys/other per-user dotfiles), not a world-writable location. |
| Denial of service via an unbounded registry directory (an attacker or bug spamming `~/.codegraph/daemons/` with thousands of records) slowing every `List()` scan | Denial of Service | Out of scope for v1.0's threat model — this directory is only ever written by `codegraph daemon start` itself (one record per invocation), under the same user account; this is not an externally-reachable or multi-tenant surface. Documented here for completeness, not flagged as requiring a mitigation in this phase. |
| Terminal-escape-sequence injection via a crafted `repoRoot`/agent display name rendered inside the bubbletea picker | Tampering (of terminal state) | `repoRoot` values are self-authored by this project's own `daemon.New`/`filepath.Abs` (never user-supplied free text at render time) and agent display names come from the hardcoded `internal/agents` registry — neither is attacker-controlled input in the threat-model sense; lipgloss/bubbletea's own rendering does not interpolate raw ANSI from data values into the terminal without going through the library's own escaping. No additional mitigation needed beyond "don't render raw untrusted strings," which this phase's data sources already satisfy. |

## Sources

### Primary (HIGH confidence)
- Direct code reading: `internal/cli/daemon.go`, `internal/daemon/daemon.go`, `internal/daemon/lock.go`, `internal/daemon/procstart_linux.go`/`procstart_other.go`, `internal/cli/install.go`, `internal/cli/uninstall.go`, `internal/cli/present/tty.go`, `internal/cli/present/progress.go`, `internal/cli/present/archtest/import_graph_test.go`, `internal/agents/registry.go`, `internal/fsatomic/fsatomic.go`, `internal/githooks/githooks.go`, `test/integration/main_test.go` + siblings, `internal/graphstore/locked_windows.go`, `.github/workflows/ci.yml`, `.github/workflows/release.yml`, `go.mod` — all read verbatim in this session.
- `go list -m -versions charm.land/bubbletea/v2` / `charm.land/bubbles/v2` — direct Go module proxy query, confirming v2.0.8 / v2.1.1 as current stable.

### Secondary (MEDIUM confidence)
- Context7 `/charmbracelet/bubbletea` — `tea.NewProgram`/`Program.Run` API shape, v1→v2 `UPGRADE_GUIDE_V2.md` key-event changes (`tea.KeyPressMsg`, `msg.Code`/`msg.Text`/`msg.Mod`).
- Context7 `/charmbracelet/bubbles` — `list.Model`/`list.ItemDelegate` interface and the confirmed absence of a built-in multi-select/checkbox delegate (verified by the delegate example shown requiring manual selection-state tracking).
- WebSearch (cross-checked against Microsoft's own `GetExitCodeProcess` documentation caveat re: `STILL_ACTIVE` unreliability) — the `OpenProcess`+`WaitForSingleObject(0)` idiom for Windows process-liveness polling.
- WebSearch (cross-checked against `PR_SET_CHILD_SUBREAPER(2)` man page semantics) — confirmation that Linux reparents an orphan to the nearest living subreaper ancestor (not unconditionally to pid 1), the basis for Pattern 5's captured-baseline recommendation.

### Tertiary (LOW confidence)
- None used directly as a basis for a recommendation in this document — every WebSearch finding above was cross-checked against an official/authoritative secondary source (Microsoft docs, the Linux man page) before being relied upon, per the classify-confidence seam's `--verified` uplift to MEDIUM.

## Metadata

**Confidence breakdown:**
- Standard stack (bubbletea/bubbles versions + API shape): HIGH — verified via direct Go module proxy query + Context7 official docs, not training-data recall
- Architecture (registry/watchdog/picker composition): HIGH — every mechanism reuses an existing, already-tested pattern in this codebase (lock.go, fsatomic, present.ChoosePresentation, the wg.Wait() join discipline); only the composition is new
- Windows stop/watchdog specifics: MEDIUM — cross-checked against authoritative docs (Microsoft, man7.org) but not executable-verified on an actual Windows machine in this session (no Windows runner available, consistent with the project's own existing `locked_windows.go` precedent of compile-only CI verification)
- Pitfalls: HIGH for the ones grounded in this codebase's own established conventions (goleak, archtest scope); MEDIUM for the bubbletea-specific non-TTY behavior claim (grounded in the library's documented terminal-capability-negotiation design, not a directly observed hang in this session)

**Research date:** 2026-07-18
**Valid until:** ~30 days for the architecture/pattern content (stable, codebase-internal); ~14 days for the exact bubbletea/bubbles pinned versions (an active, frequently-released project — re-run `go list -m -versions` at `go get` time rather than trusting this document's version numbers if more than a couple weeks have passed)
