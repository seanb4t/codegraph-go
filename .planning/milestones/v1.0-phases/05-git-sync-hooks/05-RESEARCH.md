# Phase 5: Git Sync Hooks - Research

**Researched:** 2026-07-16
**Domain:** Git hook file management (marker-fenced shell script splicing), stdlib `os/exec` git introspection
**Confidence:** HIGH

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

- **D-01:** TS 1.3.1 has NO `githooks` command. The entire TS surface is `sync/git-hooks.js` (exports `isGitRepo`/`installGitSyncHook`/`removeGitSyncHook`/`isSyncHookInstalled`/`DEFAULT_SYNC_HOOKS`), invoked from exactly two places: `init`'s `offerWatchFallback` (installer, interactive) and `uninit`'s best-effort cleanup. Our `codegraph githooks install|remove|status` command tree is a **Go-only surface extension** locked by the HOOK-01/02 requirement text — document it as such (Phase 8 SURF-05 records it alongside `search`/`migrate`). The *behavior* underneath is a verbatim TS port.
- **D-02:** Port `git-hooks.js` semantics exactly — do NOT substitute `internal/agents`' in-place `replaceOrAppendMarkedSection`. Markers (verbatim bytes): `# >>> codegraph sync hook >>>` / `# <<< codegraph sync hook <<<`. Marker matching is on trimmed lines. Install (per hook file): existing file → strip any prior marker block, trim trailing whitespace; if the remaining base is non-empty, write `base + "\n\n" + block + "\n"`; if empty or file absent, write `"#!/bin/sh\n" + block + "\n"`. Then chmod `0755` best-effort (no-op failure tolerated). Re-running install on an unmodified file MUST be byte-identical. Remove (per hook file): only touch files containing the begin marker; strip the block; if the remainder is effectively empty (every line blank or `#!`-prefixed) → delete the hook file entirely; otherwise write `strippedTrimEnd + "\n"` and re-chmod. Status: installed ⇔ any of the three hook files exists and contains the begin marker; our `status` additionally reports per-hook state.
- **D-03:** Marker block content is verbatim TS bytes, all 7 inner lines including the comment `# Managed by codegraph; remove with \`codegraph uninit\` or delete this block.` and the exact sync invocation `( codegraph sync >/dev/null 2>&1 & ) >/dev/null 2>&1` inside the `command -v codegraph` guard. Byte-identical markers mean hooks installed by TS CodeGraph are recognized/managed by the Go binary. Do not add a `githooks remove` mention to the block in v1.0.
- **D-04:** Hooks dir resolution = `git rev-parse --git-path hooks` (honors `core.hooksPath` and linked worktrees), cwd = project root, relative output resolved against the project root, absolute passed through. Repo probe = `git rev-parse --is-inside-work-tree` → literal `true`. Both under the established gitmeta exec contract: `exec.CommandContext` with 5s timeout, stderr discarded, trimmed stdout, any error/empty → null-equivalent (not-a-repo/no hooks dir → clean skip message, never an error that blocks). `githooks install` in a non-repo reports TS's `skipped: 'not a git repository'` shape as a friendly message, exit 0.
- **D-05:** All hook-file writes go through the atomic-write primitive (temp file + rename, D-08/D-09) — a deliberate Go improvement over TS's plain `writeFileSync`; on-disk result bytes are identical, only crash behavior differs. File deletion on remove is plain `os.Remove`.
- **D-06:** `codegraph uninit` gains TS-parity best-effort hook cleanup: after removing `.codegraph/`, strip codegraph's marker blocks from the three hooks (non-fatal, no-op when none/not a repo), reporting `Removed git post-commit, post-merge, post-checkout sync hooks`-style info on success — mirroring `bin/codegraph.js` ~629-636.
- **D-07:** HOOK-03's fallback surfacing = a non-interactive plain-text port of TS `offerWatchFallback`, wired into `init`'s success path (and ONLY there this phase). Logic gate-for-gate from `installer/index.js` ~476-525: (1) `watch.WatchDisabledReason(projectRoot, …)` empty → print nothing; (2) reason non-empty → warn `Live file watching is disabled here — {reason}.` + frozen-index explanation line; (3) not a git repo → `Run \`codegraph sync\` after changing files to refresh the index.` and stop; (4) hooks already installed → "already installed" info line and stop; (5) otherwise → point at `codegraph githooks install` (no auto-install without explicit user action in v1.0). Output is plain text to the command's stdout writer. Exact Go phrasing of step-5's pointer line is Claude's discretion; steps 2-4 reuse TS wording adapted only where it names clack UI affordances.
- **D-08:** The shipped Phase-3 D-12 stderr message stays byte-untouched (`… or install the git sync hooks via \`codegraph init\` …`). It is a locked verbatim-TS parity string pinned for log-driven dashboards. Known residual: Go's `init` on an already-initialized project errors instead of re-running the fallback offer — do NOT reword the message here; record the residual for Phase 8.
- **D-09:** `internal/fsatomic` extracts `atomicWriteFile` ONLY (temp file in target dir → fsync-safe rename, preserve existing file mode, 0644 default for new files — behavior byte-identical to today's `internal/agents/shared.go:327`). `internal/agents` is rewired to consume it with zero behavior change (its install/uninstall byte-invariance tests must stay green unmodified). **The marker-splice helpers are NOT extracted** — this narrows the ROADMAP note's scope deliberately. Document the narrowing in the fsatomic package comment.
- **D-10:** `IsGitRepo` and `HooksDir` live in `internal/gitmeta`, following its existing `worktree.go` exec contract verbatim. `internal/githooks` consumes gitmeta for probes and owns everything hook-specific (marker block, splice, install/remove/status, result types). `internal/cli` gains `githooks.go` registering the parent + 3 subcommands in root.go's AddCommand list.
- **D-11:** Command shapes: `githooks install [path]` / `githooks remove [path]` / `githooks status [path]`, `[path]` resolved via the existing `targetRoot` pattern (Args: MaximumNArgs(1)), matching `init`/`sync`/`uninit`. Fixed hook trio (TS `DEFAULT_SYNC_HOOKS`) — no hook-selection flags in v1.0. `status` exits 0 whether or not hooks are installed; per-hook lines + hooks-dir path, plain text. Success/skip wording mirrors TS's installer messages where one exists; `status` output shape is Claude's discretion.
- **D-12:** Real-git fixtures in `t.TempDir()` (never fake `.git` layouts; deterministic `GIT_*` env; `t.Skip` when git absent). Required cases: fresh install into a bare-hooks repo; install over an existing user hook; re-install idempotency; install after a prior-version block; remove with user content; remove when effectively empty (file deleted); remove when never installed; `core.hooksPath` honored; linked-worktree resolves to the shared common hooks dir; a TS-installed block (verbatim fixture string) is detected by `status` and removable.
- **D-13:** Mutation-proof the reachability (9-recurrence lesson): CLI tests drive the real cobra command tree, not package functions alone; the `init` advisory test asserts init PERFORMS the policy check (injectable probe forcing "disabled" — reverting the init wiring must turn it red); the uninit cleanup test asserts hooks vanish through the real `uninit --force` path. No new CI steps expected.
- **D-14:** Do not assert hook *execution* end-to-end in v1.0 (spawning git commits and racing a backgrounded, silenced `codegraph sync` is inherently flaky; TS ships zero execution tests). Content-level tests + one optional `sh -n` syntax check are sufficient. If the planner wants an execution smoke case, gate it `testing.Short()` in `test/integration/` — Claude's discretion.

### Claude's Discretion

- File layout inside `internal/githooks` (single file vs split); result struct shape (mirror TS's `{installed, hooksDir, skipped}` or Go-idiomatic equivalent); exact `githooks status` output lines; the step-5 pointer wording in D-07; whether `sh -n` validation runs in tests; goleak wiring if any goroutines appear (none expected — this phase is synchronous file I/O).

### Deferred Ideas (OUT OF SCOPE)

- Interactive "How should CodeGraph keep its index fresh?" select in `init` (TS clack UI, `hook` vs `manual`) — Phase 7 (TUI-03/04 bubbletea territory); D-07 ships the non-interactive pointer this phase.
- TEST-03 formal byte-invariance + piped-stream harness — Phase 7 (the first phase where hooks and bubbletea components coexist).
- D-12 message wording residual (serve's verbatim "via `codegraph init`" advice vs Go's non-re-runnable init) — Phase 8 SURF-05 divergence table (D-08).
- `affected --stdin/--depth/--filter/--quiet` for git-hook/CI scripting (SURF-04) — Phase 8; the hook block only ever calls bare `codegraph sync`.
- `codegraph install` (agent installer) offering hooks — TS only surfaces hooks from `init`/`uninit`; keep that scoping unless Phase 8's flag audit says otherwise.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| HOOK-01 | `codegraph githooks install` writes marker-fenced `post-commit`/`post-merge`/`post-checkout` hooks that background-run `codegraph sync`, guarded by `command -v codegraph`, idempotent (replace-in-place), preserving any user hook content | Verbatim TS `installGitSyncHook`/`stripMarkerBlock`/`markerBlock` transcribed in Code Examples; strip-then-append semantics documented in Architecture Pattern 1 and Common Pitfall 2; `HooksDir`/`IsGitRepo` exec-contract pattern in Pattern 4, empirically verified against `core.hooksPath` and linked worktrees |
| HOOK-02 | `codegraph githooks remove` strips only codegraph's marker block (preserving user content); `githooks status` reports install state | Verbatim TS `removeGitSyncHook`/`isEffectivelyEmpty`/`isSyncHookInstalled` transcribed in Code Examples; Pitfall 1 covers TS-installed-block detection fidelity; Open Question 2 covers `status`'s (TS-less) output shape |
| HOOK-03 | Hooks are surfaced as the fallback for when the watcher is disabled (WSL2 / `CODEGRAPH_NO_WATCH`), matching TS's narrower trigger — not an always-on feature | Verbatim TS `offerWatchFallback` transcribed in Code Examples; System Architecture Diagram traces the full `init`-success-path gate cascade against `watch.WatchDisabledReason`/`gitmeta.IsGitRepo`/`githooks` install-state; `uninit` cleanup call site transcribed for D-06 |

</phase_requirements>

## Summary

Phase 5 is a **verbatim port** of a single, small (226-line) TS module —
`sync/git-hooks.js` — plus a thin non-interactive rewrite of one function
from `installer/index.js` (`offerWatchFallback`, ~60 lines) and two small
call-site insertions in `bin/codegraph.js` (`init`'s success path, `uninit`'s
cleanup path). Every byte of the source has been read this session; there is
no ambiguity left in "what does TS do" — the only real design work is the Go
package shape, which CONTEXT.md's D-01 through D-14 already lock down in
detail. This research adds no new decisions; it verifies the CONTEXT.md
claims against the actual source (all confirmed) and empirically verifies
the two riskiest runtime-behavior claims (`--git-path hooks` across a linked
worktree, and `core.hooksPath` honoring) directly against a real local git
binary.

No new external dependencies are introduced. The whole phase is `os`,
`os/exec`, `path/filepath`, `strings`, `context`, `time` — all stdlib,
already vendored patterns (`internal/gitmeta`, `internal/agents`). There is
nothing to run through the Package Legitimacy Gate.

**Primary recommendation:** Port `git-hooks.js` byte-for-byte into a new
`internal/githooks` package (consuming two new `internal/gitmeta` functions,
`IsGitRepo`/`HooksDir`, and a new `internal/fsatomic.WriteFile` extracted
from `internal/agents/shared.go`'s `atomicWriteFile`). Do NOT reuse
`internal/agents`' `replaceOrAppendMarkedSection`/`removeMarkedSection` — the
splice semantics are different in three concrete ways (see Don't Hand-Roll).
Wire `githooks status`'s detection logic into `init`'s success path (D-07)
and `uninit`'s cleanup path (D-06), both non-fatal / best-effort.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Git repo/hooks-dir probing (`git rev-parse`) | Backend (internal/gitmeta) | — | Single-seam git-exec convention (Phase 2 D-03/D-04); githooks must not open its own exec.Command call sites |
| Marker-block compute/strip/splice | Backend (internal/githooks) | — | Pure string logic, hook-specific semantics, not shared with agents' splice |
| Atomic file write (temp+rename) | Backend (internal/fsatomic) | — | Shared primitive; agents package is rewired to consume it too (D-09) |
| `githooks install/remove/status` CLI surface | Backend (internal/cli) | — | Cobra command tree, mirrors init/uninit/sync shape |
| `init` watcher-fallback advisory | Backend (internal/cli, init.go) | internal/watch (policy gate), internal/gitmeta (repo probe), internal/githooks (install-state probe) | Plain-text, non-interactive; TS's interactive select has no Phase-5 analogue (Phase 7 territory) |
| `uninit` hook cleanup | Backend (internal/cli, uninit.go) | internal/githooks | Best-effort, mirrors TS `bin/codegraph.js` ~629-636 |
| Hook script itself (`post-commit` etc.) | OS-level (git-invoked shell script) | — | Not a Go process; a `/bin/sh` script git executes directly, backgrounding `codegraph sync` |

This phase touches only the backend/CLI tier — no browser, no SSR, no CDN. It is entirely local filesystem + subprocess orchestration.

## Standard Stack

### Core

No new libraries. Every dependency below is stdlib, already imported by adjacent packages in this repo.

| Package | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `os` | stdlib (go1.26.5) | File read/write/stat/remove/mkdir/chmod | Matches TS's `fs.*Sync` calls 1:1 |
| `os/exec` | stdlib | `git rev-parse` invocations | Existing `internal/gitmeta` convention (`exec.CommandContext`, 5s timeout) |
| `path/filepath` | stdlib | Hooks-dir path resolution (relative→absolute against project root) | Matches TS's `path.isAbsolute`/`path.resolve`/`path.join` |
| `context` | stdlib | Timeout propagation into `exec.CommandContext` | `internal/gitmeta`'s existing contract |
| `strings` | stdlib | Marker-line trimming/splitting/joining | Matches TS's `.split('\n')`/`.trim()`/`.join('\n')` |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `github.com/spf13/cobra` | already in go.mod | `githooks install/remove/status` subcommand tree | Matches every other CLI command in this repo |

### Alternatives Considered

None — this phase is a literal port of existing, working TS logic using stdlib. There is no library-selection question here (no YAML/TOML parsing, no templating, no third-party git binding needed — `git rev-parse` via `exec.CommandContext` is the established pattern).

**Installation:**

No `go get` / `npm install` needed — zero new module dependencies.

**Version verification:** N/A — no new external packages.

## Package Legitimacy Audit

**Not applicable this phase.** No external packages are introduced. `internal/githooks` and `internal/fsatomic` are new *internal* packages, not third-party dependencies, and every import they need (`os`, `os/exec`, `path/filepath`, `strings`, `context`, `time`) is Go stdlib. `github.com/spf13/cobra` is an existing, already-audited dependency (Phase 1+ stack, no version bump required for this phase).

**Packages removed due to [SLOP] verdict:** none
**Packages flagged as suspicious [SUS]:** none

## Architecture Patterns

### System Architecture Diagram

```
                    ┌─────────────────────────────┐
                    │   codegraph init [path]      │
                    │  (existing success path)     │
                    └──────────────┬────────────────┘
                                   │ D-07: after index completes
                                   ▼
                    ┌─────────────────────────────┐
                    │ watch.WatchDisabledReason()   │◄── injectable Probe (test seam)
                    └──────────────┬────────────────┘
                          reason==""│           │reason!=""
                          (print    │           ▼
                           nothing) │  print warn + frozen-index line
                                    │           │
                                    │           ▼
                                    │  gitmeta.IsGitRepo(root)?
                                    │     no │      │ yes
                                    │        ▼      ▼
                                    │  "run sync   githooks.IsInstalled(root)?
                                    │   manually"   already │      │ not yet
                                    │   + stop      "already   point at
                                    │               installed"  `githooks install`
                                    │               + stop      + stop
                                    ▼
                              (nothing printed)


      ┌────────────────────┐        ┌────────────────────┐        ┌────────────────────┐
      │ githooks install    │        │ githooks remove      │        │ githooks status      │
      │  [path]              │        │  [path]               │        │  [path]               │
      └──────────┬───────────┘        └──────────┬────────────┘        └──────────┬────────────┘
                 │                               │                               │
                 ▼                               ▼                               ▼
      gitmeta.HooksDir(root)          gitmeta.HooksDir(root)          gitmeta.HooksDir(root)
      (git rev-parse --git-path        (same probe)                  (same probe)
       hooks; not-a-repo → skip)
                 │                               │                               │
                 ▼                               ▼                               ▼
      for each of [post-commit,        for each hook file present      for each hook file:
       post-merge, post-checkout]:      containing MARKER_BEGIN:        exists && contains
        read existing (if any) →         stripMarkerBlock →              MARKER_BEGIN?
        stripMarkerBlock →                effectively-empty? delete     report per-hook state
        non-empty base? append           file : rewrite remainder
        block after "\n\n" :              + chmod 0755
        seed "#!/bin/sh\n"+block
        → fsatomic.WriteFile
        → chmod 0755 (best-effort)
                 │                               │
                 ▼                               ▼
      report installed hooks           report removed hooks


                    ┌─────────────────────────────┐
                    │  codegraph uninit [path]      │
                    │  (after .codegraph/ removal)  │
                    └──────────────┬────────────────┘
                                   │ D-06: best-effort, non-fatal
                                   ▼
                       githooks.Remove(root)  → "Removed git … sync hook(s)"
                          (no-op if none / not a repo)
```

The hook **script itself** runs entirely outside this Go process — git invokes `/bin/sh path/to/post-commit` as a subprocess after `commit`/`merge`/`checkout`; the script's own `( codegraph sync >/dev/null 2>&1 & ) >/dev/null 2>&1` backgrounds and silences a *separate* `codegraph sync` invocation. No file in `internal/githooks` ever executes a hook — it only writes/reads/deletes the script files.

### Recommended Project Structure

```
internal/
├── fsatomic/
│   └── fsatomic.go       # WriteFile (extracted atomicWriteFile, D-09 — atomic-write ONLY)
├── gitmeta/
│   ├── worktree.go        # existing: WorktreeRoot, CommonDir
│   ├── githooks.go        # NEW: IsGitRepo, HooksDir (D-10) — same exec contract as worktree.go
│   └── githooks_test.go
├── githooks/
│   ├── githooks.go        # markers, DefaultHooks, markerBlock(), Install/Remove/Status, result types
│   ├── splice.go          # stripMarkerBlock, isEffectivelyEmpty (or inline in githooks.go — Claude's discretion, D-decisions)
│   └── githooks_test.go
internal/cli/
├── githooks.go             # NEW: newGithooksCmd() + install/remove/status subcommands
├── init.go                 # gains D-07 advisory call in success path
└── uninit.go                # gains D-06 cleanup call after .codegraph/ removal
internal/agents/
└── shared.go                # atomicWriteFile body replaced with a call into internal/fsatomic (D-09 rewiring, zero behavior change)
```

### Pattern 1: Marker-fenced strip-then-append (NOT in-place replacement)

**What:** Re-running install does NOT edit the block in place. It strips any
existing marker block (wherever it is in the file), trims trailing
whitespace off what remains, then re-appends the current block at the
*end* of the file (with a blank-line separator if the remaining base is
non-empty).

**When to use:** Every `githooks install` call, first-time or re-run.

**Example (verbatim TS, `sync/git-hooks.js:155-186`):**
```javascript
function installGitSyncHook(projectRoot, hooks = exports.DEFAULT_SYNC_HOOKS) {
    const hooksDir = gitHooksDir(projectRoot);
    if (!hooksDir) {
        return { installed: [], hooksDir: null, skipped: 'not a git repository' };
    }
    try { fs.mkdirSync(hooksDir, { recursive: true }); }
    catch { return { installed: [], hooksDir, skipped: 'could not access the git hooks directory' }; }

    const block = markerBlock();
    const installed = [];
    for (const hook of hooks) {
        const file = path.join(hooksDir, hook);
        let content;
        if (fs.existsSync(file)) {
            const base = stripMarkerBlock(fs.readFileSync(file, 'utf8')).replace(/\s*$/, '');
            content = base.length > 0 ? `${base}\n\n${block}\n` : `#!/bin/sh\n${block}\n`;
        } else {
            content = `#!/bin/sh\n${block}\n`;
        }
        fs.writeFileSync(file, content);
        chmodExecutable(file);
        installed.push(hook);
    }
    return { installed, hooksDir };
}
```

**Go port shape (illustrative, not literal):**
```go
func Install(ctx context.Context, projectRoot string, hooks []string) Result {
    hooksDir, err := gitmeta.HooksDir(ctx, projectRoot)
    if hooksDir == "" || err != nil {
        return Result{Skipped: "not a git repository"}
    }
    if err := os.MkdirAll(hooksDir, 0o755); err != nil {
        return Result{HooksDir: hooksDir, Skipped: "could not access the git hooks directory"}
    }
    block := markerBlock()
    var installed []string
    for _, hook := range hooks {
        file := filepath.Join(hooksDir, hook)
        var content string
        if existing, err := os.ReadFile(file); err == nil {
            base := strings.TrimRight(stripMarkerBlock(string(existing)), " \t\n")
            if base != "" {
                content = base + "\n\n" + block + "\n"
            } else {
                content = "#!/bin/sh\n" + block + "\n"
            }
        } else {
            content = "#!/bin/sh\n" + block + "\n"
        }
        if err := fsatomic.WriteFile(file, content); err != nil {
            continue // or accumulate error — Claude's discretion on result shape
        }
        _ = os.Chmod(file, 0o755) // best-effort, TS swallows chmod errors too
        installed = append(installed, hook)
    }
    return Result{Installed: installed, HooksDir: hooksDir}
}
```

### Pattern 2: Trimmed-line marker matching

**What:** `stripMarkerBlock` compares `line.trim() === MARKER_BEGIN` /
`=== MARKER_END`, not the raw line — an indented marker (e.g. `  # >>> ...`)
still matches and is stripped.

**Example (verbatim TS, `sync/git-hooks.js:116-134`):**
```javascript
function stripMarkerBlock(content) {
    const lines = content.split('\n');
    const kept = [];
    let inBlock = false;
    for (const line of lines) {
        const trimmed = line.trim();
        if (trimmed === MARKER_BEGIN) { inBlock = true; continue; }
        if (trimmed === MARKER_END) { inBlock = false; continue; }
        if (!inBlock) kept.push(line);
    }
    return kept.join('\n');
}
```
Go: `strings.TrimSpace(line) == MARKER_BEGIN`, iterate over `strings.Split(content, "\n")`.

### Pattern 3: Effectively-empty detection gates deletion, not just stripping

**What:** After removal, if every remaining line is blank or starts with
`#!` (any shebang line, not just the exact one this tool wrote), the whole
file is deleted rather than left as a bare shebang. This means removing a
hook this tool never actually customized beyond the shebang cleans up fully.

**Example (verbatim TS, `sync/git-hooks.js:136-141`):**
```javascript
function isEffectivelyEmpty(content) {
    return content.split('\n').map((l) => l.trim())
        .every((l) => l.length === 0 || l.startsWith('#!'));
}
```

### Pattern 4: gitmeta exec contract for hook-dir/repo probes

**What:** `IsGitRepo`/`HooksDir` must follow `internal/gitmeta/worktree.go`'s
established shape exactly: `exec.CommandContext(ctx, "git", ...)`, `cmd.Dir =
projectRoot`, `cmd.Stdin = nil`, 5s timeout via `context.WithTimeout`, any
error → degrade to zero value (`false` / `""`), never propagate the error.

**Example (existing pattern in this repo, `internal/gitmeta/worktree.go:35-51`):**
```go
func WorktreeRoot(ctx context.Context, dir string) string {
    ctx, cancel := context.WithTimeout(ctx, gitTimeout)
    defer cancel()
    cmd := exec.CommandContext(ctx, "git", "rev-parse", "--show-toplevel")
    cmd.Dir = dir
    cmd.Stdin = nil
    out, err := cmd.Output()
    if err != nil {
        return ""
    }
    trimmed := strings.TrimSpace(string(out))
    if trimmed == "" {
        return ""
    }
    return realpath(trimmed)
}
```
`IsGitRepo`/`HooksDir` should be new siblings in `internal/gitmeta` (a new file, e.g. `githooks.go`, per D-10) using this identical shape:
- `IsGitRepo`: `git rev-parse --is-inside-work-tree`, success + trimmed output `== "true"` → `true`, else `false`.
- `HooksDir`: `git rev-parse --git-path hooks`, empty/error → `""`; if `filepath.IsAbs(out)` return as-is, else `filepath.Join(projectRoot, out)` (mirrors TS's `path.resolve(projectRoot, out)`).

**Empirically verified this session** (see Common Pitfalls #3 for why this mattered):
- Plain repo: `git rev-parse --git-path hooks` → `.git/hooks` (relative, must resolve against `projectRoot`).
- Linked worktree (`git worktree add`): `git rev-parse --git-path hooks` run *from the worktree* → resolves to the **main checkout's** `.git/hooks` as an **absolute path** (already resolved, no join needed — `filepath.IsAbs` check correctly short-circuits).
- `core.hooksPath` set to a relative custom dir: `git rev-parse --git-path hooks` → the custom relative path, must still resolve against `projectRoot`.

### Anti-Patterns to Avoid

- **In-place block replacement:** Do NOT implement install as "find the
  block, replace its contents where it sits." TS's actual behavior is
  strip-then-append-at-end — if you "fix" this to be in-place, a hook with
  user content *after* the codegraph block will end up with codegraph's
  block moved after that content on the next re-install under TS's real
  semantics, but an in-place implementation would silently diverge from a
  TS-installed hook on first Go-side re-install. D-05/D-12 explicitly call
  this out as a locked contract, not a bug to fix.
- **Reusing `internal/agents`' `replaceOrAppendMarkedSection`:** its markers
  are HTML comments (`<!-- -->`), it does in-place replacement when markers
  exist, and it never deletes the file on empty-after-strip. All three
  diverge from git-hooks semantics. See Don't Hand-Roll below.
- **Hand-joining `.git/hooks`:** never construct the hooks path via
  `filepath.Join(projectRoot, ".git", "hooks")` — this breaks on
  `core.hooksPath` and on linked worktrees (which need the shared common
  hooks dir, not a per-worktree `.git/hooks` that may not even exist as a
  directory in a linked worktree's `.git` file-based layout).

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Marker-fenced file splicing | A generic "upsert marked block" helper shared between agents and githooks | Two separate small implementations (`internal/agents`'s existing helpers, unchanged; `internal/githooks`'s new strip-then-append logic) | D-09 explicitly narrows scope: agents' splice is in-place-replace-or-append-once with HTML markers; githooks' splice is trimmed-line `#`-marker stripping + always-re-append-at-end + delete-when-effectively-empty + shebang-seeding + chmod. Forcing a shared abstraction distorts one side (agents would need to accept "always move to end" semantics it never wanted, or githooks would need to accept "never delete file" semantics that break TS parity). |
| Atomic file write (temp+rename) | A second `atomicWriteFile` copy inside `internal/githooks` | `internal/fsatomic.WriteFile` (the D-09 extraction of `internal/agents/shared.go`'s existing `atomicWriteFile`) | Already production-hardened (mode preservation, MkdirAll, temp-in-same-dir for atomic rename); extracting once and importing from both packages avoids drift between two copies of the same crash-safety logic. |
| Git introspection (`rev-parse` calls) | A second `exec.Command("git", ...)` call site inside `internal/githooks` | `internal/gitmeta.IsGitRepo` / `internal/gitmeta.HooksDir` (new, following the existing exec contract) | Single-seam confinement (established pattern this session, D-10): all git shelling-out lives in `internal/gitmeta`, nowhere else. A second ad-hoc exec site would duplicate the 5s-timeout/stdin-nil/degrade-on-error contract and risk drifting from it. |

**Key insight:** This entire phase is small enough (226 lines of TS + ~60
lines of a second TS function) that "hand-rolling" isn't really the risk —
the risk is *inventing new semantics* where TS's exact semantics are
available and already correct. The Don't-Hand-Roll items above are about
reusing this codebase's own already-built primitives (fsatomic, gitmeta),
not about pulling in a third-party library.

## Runtime State Inventory

> Not applicable — this is a greenfield feature phase (new command surface,
> new files git hooks write), not a rename/refactor/migration phase. No
> existing stored data, service config, OS registrations, or secrets are
> being renamed or moved.

One adjacent note for completeness, not a rename concern: this phase
introduces a **cross-tool compatibility surface** deliberately (D-03) — TS
CodeGraph, if previously used on this project, may have already written its
own marker-fenced hook blocks to `.git/hooks/{post-commit,post-merge,post-checkout}`
using byte-identical markers. That is not "runtime state to migrate" so much
as "runtime state the Go implementation must recognize and manage
correctly" — covered under Common Pitfalls #1 and the D-12 test-fixture
requirement (a verbatim TS-block fixture must be detected/removable by Go).

## Common Pitfalls

### Pitfall 1: Treating a TS-installed hook as foreign

**What goes wrong:** If Go's marker constants (bytes, whitespace, comment
wording) drift even slightly from TS's `MARKER_BEGIN`/`MARKER_END` strings,
`githooks status`/`remove` will silently fail to recognize hooks a user
installed with the TS CLI — `status` reports "not installed" on a hook that
actually IS installed, and `remove` treats the file as having no marker at
all (no-op, leaving TS's block orphaned forever).

**Why it happens:** Marker strings are easy to "clean up" or reformat during
a port (e.g. adding/removing a trailing space, changing comment phrasing)
without realizing the string is a *detection key*, not just documentation.

**How to avoid:** Copy `MARKER_BEGIN`/`MARKER_END` and all 7 lines of
`markerBlock()` byte-for-byte from `sync/git-hooks.js:58-59,102-114` (already
transcribed verbatim in this document's Code Examples section). Write the
D-12 fixture test that pastes TS's *exact* block bytes into a hook file and
asserts Go's `status`/`remove` detect and operate on it correctly.

**Warning signs:** A hand-typed marker constant instead of a copy-paste from
the source; a test suite with no TS-block fixture case.

### Pitfall 2: In-place edit instead of strip-then-append

**What goes wrong:** A "simpler" or "more surgical" implementation that
finds the existing block and replaces its contents in place (like
`internal/agents`'s `replaceOrAppendMarkedSection` does) silently diverges
from TS behavior the moment a user has content both before AND after the
codegraph block, or re-installs after editing around the block. TS always
moves the block to the end of the file on re-install; an in-place
implementation does not.

**Why it happens:** In-place replacement feels like better design (minimal
diff, preserves position) and is the pattern already established in
`internal/agents` for a superficially similar problem — natural to reach for
without re-reading the TS source line-by-line.

**How to avoid:** Follow the literal TS control flow: `stripMarkerBlock` →
trim trailing whitespace → `base.length > 0 ? base + "\n\n" + block + "\n" :
"#!/bin/sh\n" + block + "\n"`. Encode a test with an explanatory comment
(per CONTEXT.md § Specific Ideas) so this doesn't get "simplified" later.

**Warning signs:** A diff that only replaces the byte range between markers
instead of rewriting the whole file content.

### Pitfall 3: Hand-joining `.git/hooks` instead of `git rev-parse --git-path hooks`

**What goes wrong:** A naive `filepath.Join(projectRoot, ".git", "hooks")`
breaks in two real scenarios verified this session:
1. `core.hooksPath` set to a custom directory — hooks silently install to
   the wrong (unused) location, so they never fire.
2. Linked worktrees — a linked worktree's `.git` is a *file* (not a
   directory) containing a `gitdir:` pointer; there is no
   `<worktree>/.git/hooks` directory to write into at all. `git rev-parse
   --git-path hooks` correctly resolves to the shared common hooks
   directory (confirmed empirically this session: returns the **main
   checkout's absolute** `.git/hooks` path when run from inside a linked
   worktree).

**Why it happens:** `.git/hooks` is the "obvious" naive path and works on
the simplest case (a plain repo with default settings), so the bug is
invisible until someone with a customized or worktree-based setup hits it.

**How to avoid:** Always resolve via `git rev-parse --git-path hooks` (D-04)
— never construct the path by string-joining `.git`.

**Warning signs:** Any `filepath.Join(..., ".git", "hooks")` or
`filepath.Join(..., ".git", "hooks", hookName)` literal in the codebase.

### Pitfall 4: chmod failure treated as fatal

**What goes wrong:** On Windows (or any filesystem/mount where chmod is a
no-op or unsupported), a naive Go port that propagates `os.Chmod`'s error
will make `githooks install` fail entirely on platforms where TS's
`chmodSync` inside a `try {} catch {}` silently swallows the same failure.

**Why it happens:** Go's idiomatic "check every error" instinct conflicts
with TS's deliberate best-effort swallow here.

**How to avoid:** `_ = os.Chmod(file, 0o755)` — discard the error explicitly
(with a comment noting why), matching `chmodExecutable`'s empty catch block
(`sync/git-hooks.js:142-149`).

**Warning signs:** `if err := os.Chmod(...); err != nil { return err }`
anywhere in the install/remove path.

### Pitfall 5: Blocking git operations on a slow/hung `codegraph sync`

**What goes wrong:** If the hook script's shell snippet is written without
the subshell-backgrounding syntax, `git commit`/`merge`/`checkout` will
block on `codegraph sync` completing — defeating the entire "never blocks a
commit" success criterion (HOOK-01).

**Why it happens:** It's tempting to "simplify" the shell snippet to just
`codegraph sync >/dev/null 2>&1 &` (a single `&`) instead of TS's `(
codegraph sync >/dev/null 2>&1 & ) >/dev/null 2>&1` — the outer subshell
wrapping matters: without it, some shells still keep the hook's own process
group tied to the backgrounded job in a way that can delay git's perceived
completion (job-control / wait semantics vary by shell).

**How to avoid:** Copy the exact 2-line body from `markerBlock()` verbatim —
do not "clean up" the parens or redirection. This is exactly what D-03 locks
down.

**Warning signs:** Any deviation from `if command -v codegraph >/dev/null
2>&1; then\n  ( codegraph sync >/dev/null 2>&1 & ) >/dev/null 2>&1\nfi`.

## Code Examples

### Verbatim marker constants and block (source: `sync/git-hooks.js:58-59,102-114`)

```javascript
const MARKER_BEGIN = '# >>> codegraph sync hook >>>';
const MARKER_END = '# <<< codegraph sync hook <<<';

function markerBlock() {
    return [
        MARKER_BEGIN,
        '# Keeps the CodeGraph index fresh while the live file watcher is off',
        '# (e.g. WSL2 /mnt drives). Runs in the background so it never blocks git.',
        '# Managed by codegraph; remove with `codegraph uninit` or delete this block.',
        'if command -v codegraph >/dev/null 2>&1; then',
        '  ( codegraph sync >/dev/null 2>&1 & ) >/dev/null 2>&1',
        'fi',
        MARKER_END,
    ].join('\n');
}
```

### DEFAULT_SYNC_HOOKS (source: `sync/git-hooks.js:60-61`)

```javascript
exports.DEFAULT_SYNC_HOOKS = ['post-commit', 'post-merge', 'post-checkout'];
```

### Verbatim `isGitRepo`/`gitHooksDir` (source: `sync/git-hooks.js:66-101`)

```javascript
function isGitRepo(projectRoot) {
    try {
        const out = execFileSync('git', ['rev-parse', '--is-inside-work-tree'], {
            cwd: projectRoot, encoding: 'utf8',
            stdio: ['ignore', 'pipe', 'ignore'], windowsHide: true, timeout: 5000,
        }).trim();
        return out === 'true';
    } catch { return false; }
}

function gitHooksDir(projectRoot) {
    try {
        const out = execFileSync('git', ['rev-parse', '--git-path', 'hooks'], {
            cwd: projectRoot, encoding: 'utf8',
            stdio: ['ignore', 'pipe', 'ignore'], windowsHide: true, timeout: 5000,
        }).trim();
        if (!out) return null;
        return path.isAbsolute(out) ? out : path.resolve(projectRoot, out);
    } catch { return null; }
}
```

### Verbatim `isSyncHookInstalled` (source: `sync/git-hooks.js:218-226`)

```javascript
function isSyncHookInstalled(projectRoot, hooks = exports.DEFAULT_SYNC_HOOKS) {
    const hooksDir = gitHooksDir(projectRoot);
    if (!hooksDir) return false;
    return hooks.some((hook) => {
        const file = path.join(hooksDir, hook);
        return fs.existsSync(file) && fs.readFileSync(file, 'utf8').includes(MARKER_BEGIN);
    });
}
```

### Verbatim `removeGitSyncHook` (source: `sync/git-hooks.js:192-216`)

```javascript
function removeGitSyncHook(projectRoot, hooks = exports.DEFAULT_SYNC_HOOKS) {
    const hooksDir = gitHooksDir(projectRoot);
    if (!hooksDir) return { installed: [], hooksDir: null, skipped: 'not a git repository' };
    const removed = [];
    for (const hook of hooks) {
        const file = path.join(hooksDir, hook);
        if (!fs.existsSync(file)) continue;
        const original = fs.readFileSync(file, 'utf8');
        if (!original.includes(MARKER_BEGIN)) continue;
        const stripped = stripMarkerBlock(original);
        if (isEffectivelyEmpty(stripped)) {
            fs.unlinkSync(file);
        } else {
            fs.writeFileSync(file, `${stripped.replace(/\s*$/, '')}\n`);
            chmodExecutable(file);
        }
        removed.push(hook);
    }
    return { installed: removed, hooksDir };
}
```

Note: TS reuses the `{installed, hooksDir}` result shape for BOTH install
and remove — the `removed` array is stored under the key `installed` in the
returned object (`return { installed: removed, hooksDir }`). This is a TS
naming quirk, not a semantic requirement — D-decisions leave the Go result
struct shape to Claude's discretion ("mirror TS's `{installed, hooksDir,
skipped}` or Go-idiomatic equivalent"). A Go port SHOULD use a clearer field
name (e.g. `Removed []string` on a `RemoveResult`, `Installed []string` on
an `InstallResult`) rather than copying this naming oddity.

### Verbatim `offerWatchFallback` (source: `installer/index.js:476-525`)

```javascript
async function offerWatchFallback(clack, projectPath, opts = {}) {
    const reason = watchDisabledReason(projectPath);
    if (!reason) return; // Watcher runs normally — nothing to set up.
    clack.log.warn(`Live file watching is disabled here — ${reason}.`);
    clack.log.info('Until you re-sync, the CodeGraph index stays frozen — it will not pick up edits on its own.');
    if (!isGitRepo(projectPath)) {
        clack.log.info('Run `codegraph sync` after changing files to refresh the index.');
        return;
    }
    if (isSyncHookInstalled(projectPath)) {
        clack.log.info('Git sync hooks are already installed — the index refreshes after commit / pull / checkout.');
        return;
    }
    // ... interactive clack.select() branch — Phase 7 territory, not ported ...
    const result = installGitSyncHook(projectPath);
    if (result.installed.length > 0) {
        clack.log.success(`Installed git ${result.installed.join(', ')} hook${result.installed.length > 1 ? 's' : ''} — ` +
            'the index refreshes in the background after each.');
        clack.log.info('Run `codegraph sync` anytime to refresh immediately.');
    } else {
        clack.log.warn(`Could not install git hooks${result.skipped ? ` (${result.skipped})` : ''}. ` +
            'Run `codegraph sync` after changes instead.');
    }
}
```

D-07's Go port stops at "already installed" or points at `githooks install`
(no interactive select) — the success/failure strings above (`Installed git
… — the index refreshes in the background after each.` / `Could not install
git hooks(…). Run \`codegraph sync\` after changes instead.`) are still
useful reference strings if the planner wants `githooks install`'s own
output to echo them (D-11 already specifies near-identical wording for the
command's own success line).

### uninit cleanup call site (source: `bin/codegraph.js:629-637`)

```javascript
try {
    const { removeGitSyncHook } = await import('../sync/git-hooks');
    const removed = removeGitSyncHook(projectPath);
    if (removed.installed.length > 0) {
        info(`Removed git ${removed.installed.join(', ')} sync hook${removed.installed.length > 1 ? 's' : ''}`);
    }
} catch { /* non-fatal */ }
```

### init advisory call sites (source: `bin/codegraph.js:544-548, 586-590`)

Two call sites inside `init`: one on the "already initialized" early-return
branch (line ~544-548), one on the normal success path after indexing
(~586-590). Both wrapped in `try { } catch { /* non-fatal */ }`. D-07 wires
Go's version into `init`'s **success path only** ("and ONLY there this
phase") — the already-initialized branch is explicitly out of scope this
phase (D-08's documented residual for Phase 8 SURF-05).

### Existing Go exec-contract pattern to follow (source: `internal/gitmeta/worktree.go:35-51`, this repo)

```go
func WorktreeRoot(ctx context.Context, dir string) string {
    ctx, cancel := context.WithTimeout(ctx, gitTimeout)
    defer cancel()
    cmd := exec.CommandContext(ctx, "git", "rev-parse", "--show-toplevel")
    cmd.Dir = dir
    cmd.Stdin = nil
    out, err := cmd.Output()
    if err != nil {
        return ""
    }
    trimmed := strings.TrimSpace(string(out))
    if trimmed == "" {
        return ""
    }
    return realpath(trimmed)
}
```

### Existing real-git fixture helper to reuse (source: `internal/gitmeta/fixtures_test.go:20-54`, this repo)

```go
func runGit(t *testing.T, dir string, args ...string) string {
    t.Helper()
    base := []string{
        "-c", "init.defaultBranch=main",
        "-c", "user.name=codegraph-test",
        "-c", "user.email=test@example.invalid",
        "-c", "commit.gpgsign=false",
        "-c", "protocol.file.allow=always",
    }
    full := append(append([]string{}, base...), args...)
    cmd := exec.Command("git", full...)
    cmd.Dir = dir
    out, err := cmd.CombinedOutput()
    if err != nil {
        t.Skipf("git %v failed (git missing or fixture unsupported here): %v: %s", args, err, string(out))
    }
    return string(out)
}

func initRepo(t *testing.T, dir string) string {
    t.Helper()
    os.MkdirAll(dir, 0o755)
    runGit(t, dir, "init")
    os.WriteFile(filepath.Join(dir, "README.md"), []byte("fixture\n"), 0o644)
    runGit(t, dir, "add", "-A")
    runGit(t, dir, "commit", "-m", "init")
    return dir
}
```
D-12's required fixture cases (fresh install, install-over-user-hook,
re-install idempotency, TS-block detection/removal, `core.hooksPath`,
linked-worktree resolution) should build on `initRepo` plus a
`runGit(t, main, "worktree", "add", "-b", "feature", wt)` call for the
worktree case — mirroring `newLinkedWorktreeFixture` in the same file.

### Existing CLI reachability test pattern to follow (source: `internal/cli/install_test.go:66-79`, `cli_test.go:55`, this repo)

```go
func TestInstall_TargetAll_WritesAndReportsPerAgent(t *testing.T) {
    home := fakeHome(t)
    out, _, err := execCmd("install", "--target", "all", "--location", "global")
    if err != nil {
        t.Fatalf("install --target all: %v", err)
    }
    // assert on out ...
}
```
`execCmd` drives the real cobra command tree (D-13's "not package functions
alone" requirement). For githooks tests this needs a real-git fixture
directory as the target path argument rather than (or in addition to)
`fakeHome`'s isolated project dir.

### The Go extraction target (source: `internal/agents/shared.go:312-360`, this repo — D-09)

```go
func atomicWriteFile(path, content string) error {
    dir := filepath.Dir(path)
    if err := os.MkdirAll(dir, 0o755); err != nil {
        return err
    }
    tmp, err := os.CreateTemp(dir, ".codegraph-write-*")
    if err != nil {
        return err
    }
    tmpPath := tmp.Name()
    if _, err := tmp.WriteString(content); err != nil {
        tmp.Close()
        os.Remove(tmpPath)
        return err
    }
    if err := tmp.Close(); err != nil {
        os.Remove(tmpPath)
        return err
    }
    mode := os.FileMode(0o644)
    if info, statErr := os.Stat(path); statErr == nil {
        mode = info.Mode().Perm()
    }
    if err := os.Chmod(tmpPath, mode); err != nil {
        os.Remove(tmpPath)
        return err
    }
    if err := os.Rename(tmpPath, path); err != nil {
        os.Remove(tmpPath)
        return err
    }
    return nil
}
```
D-09: extract this exact function body into `internal/fsatomic` (e.g. as
`fsatomic.WriteFile`), byte-identical behavior, and rewire
`internal/agents/shared.go` to call it — the agents package's existing
byte-invariance tests (`internal/agents/*_test.go`) must stay green
unmodified through this rewiring. **Explicitly NOT extracted**:
`replaceOrAppendMarkedSection`/`removeMarkedSection` (lines 194-295) — those
are the agents-specific splice helpers this phase must NOT reuse (see Don't
Hand-Roll).

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| N/A | N/A | N/A | This phase ports a stable, unversioned TS module (no upstream API churn to track — `git-hooks.js` has no changelog entries in the frozen TS 1.3.1 capture) |

**Deprecated/outdated:** None applicable — `git rev-parse --git-path` and
`--is-inside-work-tree` are long-stable, low-level git plumbing commands
(available since early git 2.x), not subject to deprecation churn.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Go's `os.Chmod(file, 0o755)` on Windows is a safe no-op (does not error in a way that would need special handling) rather than TS's `fs.chmodSync` behavior on Windows, which the TS source's own comment says is "unsupported" and silently caught | Common Pitfalls #4 | Low — both TS and the recommended Go pattern discard the error either way (`_ = os.Chmod(...)`), so even if Windows behavior differs slightly (error vs. silent success), the code path is identical: ignore and move on. No test currently runs on Windows CI for this repo (unverified this session) — flag for the planner's Windows note (D-11 mentions "Windows behavior notes" as an open question) |
| A2 | `git rev-parse --git-path hooks` behaves identically across the git versions likely to be on a user's machine (the CONTEXT.md open question "whether output differs across git versions" is not fully resolved — only the *current, locally installed* git version was tested this session) | Architecture Patterns, Pattern 4 | Low-Medium — `--git-path` is old, stable plumbing; a git version old enough to lack it would also lack `--is-inside-work-tree`'s modern behavior and other things this codebase already depends on via `internal/gitmeta`. If a plan wants certainty, add a `git --version` floor check or note it as an accepted risk matching existing gitmeta assumptions |

**If this table is empty:** N/A — two low-risk assumptions logged above; neither blocks planning.

## Open Questions

1. **Exact Go signatures for githooks package result types**
   - What we know: TS returns `{installed: string[], hooksDir: string|null, skipped?: string}` from both install and remove (reusing the same shape, remove's removed-list stored under the `installed` key — a TS naming quirk documented in Code Examples above).
   - What's unclear: CONTEXT.md explicitly leaves this to Claude's discretion ("result struct shape (mirror TS's `{installed, hooksDir, skipped}` or Go-idiomatic equivalent)").
   - Recommendation: Two distinct result types are clearer Go style —
     `InstallResult{Installed []string; HooksDir string; Skipped string}` and
     `RemoveResult{Removed []string; HooksDir string; Skipped string}` — the
     planner should pick field names once and use them consistently across
     `internal/githooks` and the `internal/cli/githooks.go` output
     formatting.

2. **`githooks status` output shape**
   - What we know: TS has no `status` analogue at all (`isSyncHookInstalled` returns a single bool, no CLI surface). D-11 confirms `status` output is "Claude's discretion (TS has no analogue)" but specifies content: per-hook install state + hooks-dir path, plain text, exit 0 regardless.
   - What's unclear: exact line format.
   - Recommendation: Something like:
     ```
     hooks dir: /path/to/.git/hooks
     post-commit:    installed
     post-merge:     not installed
     post-checkout:  installed
     ```
     Left fully to the planner/executor; no TS byte-parity constraint applies here.

3. **`sh -n` syntax validation in tests**
   - What we know: D-14 marks this "Claude's discretion."
   - What's unclear: whether it adds meaningful coverage beyond content-string assertions.
   - Recommendation: Low-cost, low-value addition (the block is a fixed, already-correct 7-line snippet with no interpolation) — optional, not blocking. If added, gate behind `t.Skip` when `sh` is unavailable, mirroring the `git`-absence skip pattern.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| `git` CLI | `gitmeta.IsGitRepo`/`HooksDir`, all `githooks` operations, all D-12 fixture tests | ✓ (verified this session) | not pinned; empirically tested against the locally installed version | `IsGitRepo`/`HooksDir` degrade to `false`/`""` on any exec failure (missing git included) per the existing gitmeta "never block" contract (D-04); `githooks install/remove` report `skipped: "not a git repository"` and exit 0 |
| `sh` (POSIX shell) | Hook script execution (not this Go process — git invokes it) | not applicable to Go build/test, only to end-user runtime | — | N/A — hooks target `#!/bin/sh`; on Windows, Git for Windows ships its own `sh.exe` that git's hook-invocation mechanism uses (see Open Question re: Windows, left to planner) |
| Go toolchain | Build/test | ✓ | go1.26.5 (this session's `go version`) | — |

**Missing dependencies with no fallback:** none.

**Missing dependencies with fallback:** `git` — already has an established, tested fallback path (degrade to no-op, never block/error) reused directly from `internal/gitmeta`'s existing WORK-03 contract.

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Go stdlib `testing` (existing repo convention — no third-party test framework anywhere in this codebase) |
| Config file | none — plain `go test ./...` |
| Quick run command | `go test ./internal/githooks/... ./internal/gitmeta/... ./internal/fsatomic/... ./internal/cli/...` |
| Full suite command | `go test ./...` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| HOOK-01 | Fresh install writes marker-fenced hooks with correct content, mode 0755, guarded by `command -v codegraph` | unit | `go test ./internal/githooks/... -run TestInstall -v` | ❌ Wave 0 (new package) |
| HOOK-01 | Install over existing user hook preserves user content, appends block after blank line | unit | `go test ./internal/githooks/... -run TestInstall_PreservesUserContent -v` | ❌ Wave 0 |
| HOOK-01 | Re-install is idempotent (byte-identical second run) | unit | `go test ./internal/githooks/... -run TestInstall_Idempotent -v` | ❌ Wave 0 |
| HOOK-01 | Install after a prior-version block strips+re-appends (no duplication) | unit | `go test ./internal/githooks/... -run TestInstall_ReplacesPriorBlock -v` | ❌ Wave 0 |
| HOOK-01 | `install` in a non-repo reports friendly skip message, exit 0 | CLI reachability | `go test ./internal/cli/... -run TestGithooksInstall_NotAGitRepo -v` | ❌ Wave 0 |
| HOOK-01 | `core.hooksPath` honored | unit | `go test ./internal/gitmeta/... -run TestHooksDir_HonorsCoreHooksPath -v` | ❌ Wave 0 |
| HOOK-01 | Linked worktree resolves to shared common hooks dir | unit | `go test ./internal/gitmeta/... -run TestHooksDir_LinkedWorktree -v` | ❌ Wave 0 |
| HOOK-02 | Remove strips only codegraph's block, byte-preserves remainder | unit | `go test ./internal/githooks/... -run TestRemove_PreservesUserContent -v` | ❌ Wave 0 |
| HOOK-02 | Remove deletes file when effectively empty | unit | `go test ./internal/githooks/... -run TestRemove_DeletesWhenEmpty -v` | ❌ Wave 0 |
| HOOK-02 | Remove when never installed is a no-op, no error | unit | `go test ./internal/githooks/... -run TestRemove_NeverInstalled -v` | ❌ Wave 0 |
| HOOK-02 | A TS-installed block (verbatim fixture bytes) is detected by `status` and removable | unit | `go test ./internal/githooks/... -run TestStatus_DetectsTSInstalledBlock -v` | ❌ Wave 0 |
| HOOK-02 | `githooks status` reports per-hook state, exits 0 either way | CLI reachability | `go test ./internal/cli/... -run TestGithooksStatus -v` | ❌ Wave 0 |
| HOOK-03 | `init` advisory prints nothing when watcher is enabled | CLI reachability | `go test ./internal/cli/... -run TestInitAdvisory_WatcherEnabled -v` | ❌ Wave 0 |
| HOOK-03 | `init` advisory triggers on injected "disabled" Probe (reachability — reverting the wiring must turn this red per D-13) | CLI reachability | `go test ./internal/cli/... -run TestInitAdvisory_WatcherDisabled -v` | ❌ Wave 0 |
| HOOK-03 | `uninit --force` removes hooks through the real cobra path | CLI reachability | `go test ./internal/cli/... -run TestUninit_RemovesGitHooks -v` | ❌ Wave 0 |
| — (D-09) | `internal/agents` byte-invariance tests stay green after `fsatomic` rewiring | regression | `go test ./internal/agents/...` | ✅ existing |

### Sampling Rate
- **Per task commit:** `go test ./internal/githooks/... ./internal/gitmeta/... ./internal/fsatomic/... ./internal/cli/...`
- **Per wave merge:** `go test ./...`
- **Phase gate:** Full suite green before `/gsd-verify-work`

### Wave 0 Gaps
- [ ] `internal/fsatomic/fsatomic.go` + `fsatomic_test.go` — new package, D-09 extraction
- [ ] `internal/gitmeta/githooks.go` + `githooks_test.go` — `IsGitRepo`/`HooksDir` additions, D-10
- [ ] `internal/githooks/githooks.go` + `githooks_test.go` — the core port, D-01 through D-05
- [ ] `internal/cli/githooks.go` + `githooks_test.go` — command tree, D-11
- [ ] `internal/cli/init_test.go` (or new file) — D-07 advisory reachability test with injectable `watch.Probe`
- [ ] `internal/cli/uninit_test.go` (or new file) — D-06 cleanup reachability test
- [ ] Framework install: none — `testing` stdlib already used throughout

Per D-14, hook *execution* end-to-end is explicitly NOT required this phase (content-level tests suffice; the never-blocks property is by construction). No `testing.Short()`-gated execution smoke test is required, though the planner may add one at their discretion.

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | No auth surface — local CLI + local filesystem |
| V3 Session Management | no | N/A |
| V4 Access Control | no | Operates only within the resolved project root's git hooks dir; no privilege boundary crossed |
| V5 Input Validation | yes | `[path]` argument is resolved via the existing `targetRoot` helper (`filepath.Abs`); hook names are a fixed internal constant list (`DEFAULT_SYNC_HOOKS`), never user-supplied — no injection surface for hook *names*. The written *content* is a fixed template string with zero interpolation of untrusted input (no project name, no path, no env var is spliced into the shell block) |
| V6 Cryptography | no | N/A — no crypto operations |
| V12 File/Resource | yes | Atomic writes via `fsatomic.WriteFile` (temp+rename) prevent partial/corrupted hook files on crash; `os.Remove` only ever targets a path already resolved from `HooksDir(root) + hookName`, one of the 3 fixed hook names — never an arbitrary user-controlled path |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Command injection via hook content | Tampering | Not applicable — the shell snippet is a fixed constant string (`markerBlock()`), never built via string interpolation of any runtime value (no project path, no username, no env var appears inside the block). Copy the block byte-for-byte; do not add `fmt.Sprintf`-based customization to the shell body in a future change without re-threat-modeling |
| Path traversal via `[path]` argument | Tampering | Already mitigated by the existing `targetRoot(args)` pattern (`filepath.Abs`) shared with `init`/`uninit`/`sync` — no new path-handling code introduced by this phase |
| TOCTOU on hook file between exists-check and write | Tampering | Low severity (single-user local CLI, no concurrent-writer threat model established elsewhere in this codebase); `fsatomic.WriteFile`'s temp+rename already limits the window where a partial write could be observed, matching the existing `internal/agents` risk posture (a materially higher-risk third-party-config surface than this phase's own git-owned hooks dir) |
| Overwriting an unrelated file at a git-supplied path | Spoofing (of `--git-path` output) | `HooksDir` degrades to `""` on any error/empty output (never assumes a value); every write is `filepath.Join(hooksDir, hook)` where `hook` is one of exactly 3 fixed constants — there is no code path where an attacker-controlled string reaches the write-target path |
| A malicious hook file already present, install "poisoning" it further | Tampering | Out of scope — a compromised `.git/hooks/post-commit` is a pre-existing local-attacker-with-repo-write-access scenario this tool does not attempt to detect or sandbox against (git hooks are inherently trusted local scripts; this is git's own threat model, not a codegraph-specific gap) |

No new attack surface is introduced beyond what `internal/agents` (arbitrary third-party config file writes) and `internal/gitmeta` (git subprocess invocation) already carry and have already been threat-modeled for in prior phases.

## Sources

### Primary (HIGH confidence)
- `/opt/homebrew/lib/node_modules/@colbymchenry/codegraph/node_modules/@colbymchenry/codegraph-darwin-arm64/lib/dist/sync/git-hooks.js` — read in full (226 lines), the complete HOOK-01/02 ground truth. `[VERIFIED: local file, read in full this session]`
- `…/codegraph-darwin-arm64/lib/dist/installer/index.js` lines 460-526 — `offerWatchFallback`, the HOOK-03 trigger logic. `[VERIFIED: local file, read this session]`
- `…/codegraph-darwin-arm64/lib/dist/bin/codegraph.js` lines 510-640 — `init`/`uninit` call sites for git-hooks integration. `[VERIFIED: local file, read this session]`
- `internal/agents/shared.go` (this repo) — `atomicWriteFile`, `replaceOrAppendMarkedSection`, `removeMarkedSection`. `[VERIFIED: local file, read this session]`
- `internal/gitmeta/worktree.go`, `internal/gitmeta/detect.go`, `internal/gitmeta/fixtures_test.go` (this repo) — exec contract, fixture-building pattern. `[VERIFIED: local file, read this session]`
- `internal/watch/policy.go` (this repo) — `WatchDisabledReason`/`Probe`, the D-07 gate. `[VERIFIED: local file, read this session]`
- `internal/cli/root.go`, `init.go`, `uninit.go`, `sync.go`, `serve.go`, `daemon.go`, `install_test.go`, `cli_test.go` (this repo) — command registration pattern, `targetRoot`, `confirm`, the verbatim D-08 stderr message, `execCmd` test helper. `[VERIFIED: local file, read this session]`
- Empirical `git rev-parse --git-path hooks` behavior across a plain repo, a linked worktree, and a `core.hooksPath`-customized repo — ad hoc shell verification this session against the locally installed git binary. `[VERIFIED: command output, this session]`

### Secondary (MEDIUM confidence)
None — every claim in this document traces to a directly-read local file or an empirically-run command this session. No web search was performed (unnecessary: the entire domain is internal ground truth + Go stdlib).

### Tertiary (LOW confidence)
None.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — stdlib only, no library-selection ambiguity
- Architecture: HIGH — CONTEXT.md's D-01 through D-14 already fully specify the port; this research independently verified every claim against the actual TS source and empirically confirmed the two riskiest runtime assumptions (worktree hooks-dir resolution, `core.hooksPath` honoring)
- Pitfalls: HIGH — all five pitfalls are drawn from literal TS/Go source comparison, not speculation

**Research date:** 2026-07-16
**Valid until:** No expiry driver — this research is a snapshot of a frozen TS 1.3.1 dist and this repo's own current code; it does not track a moving external API. Re-verify only if the TS ground-truth path changes (new TS CodeGraph version installed) or if `internal/gitmeta`/`internal/agents`'s referenced functions are refactored before this phase executes.
