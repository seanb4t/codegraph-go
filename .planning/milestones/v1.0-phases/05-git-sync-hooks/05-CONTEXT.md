# Phase 5: Git Sync Hooks - Context

**Gathered:** 2026-07-16
**Status:** Ready for planning
**Mode:** --auto (all gray areas auto-selected with recommended options; decisions logged in 05-DISCUSSION-LOG.md)

<domain>
## Phase Boundary

Users can install marker-fenced git sync hooks (`post-commit` / `post-merge` /
`post-checkout`) that keep the `.codegraph/` index fresh when the live file
watcher is disabled (WSL2 `/mnt` drives, `CODEGRAPH_NO_WATCH`) — byte-safe
around user-authored hook content, idempotent on re-install, and never
blocking a git operation (the hook backgrounds `codegraph sync`, silenced,
guarded by `command -v codegraph`). Surface: `codegraph githooks install`,
`githooks remove`, `githooks status`, plus the watcher-disabled fallback
advisory that makes HOOK-03's "narrower trigger" real. New
`internal/githooks` package + `internal/fsatomic` extraction from
`internal/agents`.

Requirements: HOOK-01, HOOK-02, HOOK-03.

**Not in this phase:** the formal TEST-03 byte-invariance + piped-stream
harness (Phase 7 — the first phase where hooks and bubbletea components
coexist; HOOK-01/02 still bake idempotency/preservation tests in HERE),
any interactive prompt/select UI (Phase 7 TUI-03/04 — clack's select has no
v1.0-Phase-5 Go analogue), `affected --stdin/--quiet` git-hook scripting
flags (SURF-04, Phase 8), colorized output (Phase 6), and any change to the
watcher/daemon model itself (Phase 3 shipped it; this phase only consumes
`watch.WatchDisabledReason`).

</domain>

<decisions>
## Implementation Decisions

### ★ TS ground truth: `sync/git-hooks.js` — read verbatim this session

- **D-01:** **TS 1.3.1 has NO `githooks` command.** The entire TS surface is
  `sync/git-hooks.js` (exports `isGitRepo` / `installGitSyncHook` /
  `removeGitSyncHook` / `isSyncHookInstalled` / `DEFAULT_SYNC_HOOKS`),
  invoked from exactly two places: `init`'s `offerWatchFallback` (installer,
  interactive) and `uninit`'s best-effort cleanup. Our
  `codegraph githooks install|remove|status` command tree is therefore a
  **Go-only surface extension** locked by the HOOK-01/02 requirement text —
  document it as such (Phase 8 SURF-05 records it alongside `search` /
  `migrate`). The *behavior* underneath is a verbatim TS port.

### Hook file semantics (HOOK-01/02)

- **D-02:** **Port `git-hooks.js` semantics exactly** — do NOT substitute
  `internal/agents`' in-place `replaceOrAppendMarkedSection`:
  - **Markers (verbatim bytes):** `# >>> codegraph sync hook >>>` /
    `# <<< codegraph sync hook <<<`. Marker matching is on **trimmed lines**
    (an indented marker still counts), per TS `stripMarkerBlock`.
  - **Install (per hook file):** existing file → strip any prior marker
    block, trim trailing whitespace; if the remaining base is non-empty,
    write `base + "\n\n" + block + "\n"`; if empty or file absent, write
    `"#!/bin/sh\n" + block + "\n"`. Then chmod `0755` best-effort (no-op
    failure tolerated — Windows). The requirement's "replace-in-place"
    reads as *idempotent replacement, never duplication* — TS's actual
    mechanism is strip-then-re-append-at-end, and TS fidelity is the parity
    bar. Re-running install on an unmodified file MUST be byte-identical.
  - **Remove (per hook file):** only touch files containing the begin
    marker; strip the block; if the remainder is *effectively empty* (every
    line blank or `#!`-prefixed) → **delete the hook file entirely**;
    otherwise write `strippedTrimEnd + "\n"` and re-chmod. User-authored
    content is preserved byte-for-byte.
  - **Status:** installed ⇔ any of the three hook files exists and contains
    the begin marker (TS `isSyncHookInstalled` is `some()`; our `status`
    additionally reports per-hook state — see D-07).
- **D-03:** **Marker block content is verbatim TS bytes**, all 7 inner lines
  including the comment `# Managed by codegraph; remove with \`codegraph
  uninit\` or delete this block.` and the exact sync invocation
  `( codegraph sync >/dev/null 2>&1 & ) >/dev/null 2>&1` inside the
  `command -v codegraph` guard. Rationale: (a) verbatim-TS-strings-on-parity
  -surfaces convention (Phase 3 D-12/D-13); (b) byte-identical markers mean
  hooks installed by TS CodeGraph are **recognized and managed by the Go
  binary** — `githooks status` detects them, `remove`/re-`install` operate
  on them — a genuine drop-in-swap win; (c) the `codegraph uninit` advice
  stays truthful because D-06 wires uninit cleanup. Do not add a
  `githooks remove` mention to the block in v1.0 (it would break byte
  parity with TS-installed hooks for zero functional gain).
- **D-04:** **Hooks dir resolution = `git rev-parse --git-path hooks`**
  (honors `core.hooksPath` and linked worktrees — worktrees share the
  common hooks dir), cwd = project root, relative output resolved against
  the project root, absolute passed through. Repo probe =
  `git rev-parse --is-inside-work-tree` → literal `true`. Both under the
  established gitmeta exec contract: `exec.CommandContext` with 5s timeout,
  stderr discarded, trimmed stdout, any error/empty → null-equivalent
  (not-a-repo / no hooks dir → clean skip message, never an error that
  blocks). `githooks install` in a non-repo reports TS's
  `skipped: 'not a git repository'` shape as a friendly message, exit 0.
- **D-05:** **All hook-file writes go through the atomic-write primitive**
  (temp file + rename, D-08) — a deliberate Go improvement over TS's plain
  `writeFileSync`; on-disk result bytes are identical, only crash behavior
  differs. File deletion on remove is plain `os.Remove`.

### Surfacing & lifecycle integration (HOOK-03)

- **D-06:** **`codegraph uninit` gains TS-parity best-effort hook cleanup**:
  after removing `.codegraph/`, strip codegraph's marker blocks from the
  three hooks (non-fatal, no-op when none / not a repo), reporting
  `Removed git post-commit, post-merge, post-checkout sync hooks`-style
  info on success — mirroring `bin/codegraph.js` ~629-636.
- **D-07:** **HOOK-03's fallback surfacing = a non-interactive plain-text
  port of TS `offerWatchFallback`, wired into `init`'s success path** (and
  ONLY there this phase). Logic gate-for-gate from `installer/index.js`
  ~476-525:
  1. `watch.WatchDisabledReason(projectRoot, …)` empty → print nothing
     (the not-always-on guarantee).
  2. Reason non-empty → warn `Live file watching is disabled here —
     {reason}.` + the frozen-index explanation line.
  3. Not a git repo → `Run \`codegraph sync\` after changing files to
     refresh the index.` and stop.
  4. Hooks already installed → the "already installed" info line and stop.
  5. Otherwise → point at `codegraph githooks install` (Go adaptation of
     TS's interactive select — clack prompts have no Phase-5 analogue;
     the bubbletea select is Phase 7 territory, see Deferred). No
     auto-install without explicit user action in v1.0.
  Output is plain text to the command's stdout writer (Phase 6 owns color).
  Exact Go phrasing of step-5's pointer line is Claude's discretion; steps
  2-4 reuse TS wording adapted only where it names clack UI affordances.
- **D-08:** **The shipped Phase-3 D-12 stderr message stays byte-untouched**
  (`… or install the git sync hooks via \`codegraph init\` …`). It is a
  locked verbatim-TS parity string pinned for log-driven dashboards.
  Known residual: Go's `init` on an already-initialized project errors
  ("already exists — use `codegraph index --force`") instead of re-running
  the fallback offer like TS's re-runnable init — so the message's advice
  is imperfect on an existing index. Do NOT reword it here; record the
  residual for Phase 8 surface reconciliation (SURF-05 divergence table).

### Package layout & extraction

- **D-09:** **`internal/fsatomic` extracts `atomicWriteFile` ONLY** (temp
  file in target dir → fsync-safe rename, preserve existing file mode,
  0644 default for new files — behavior byte-identical to today's
  `internal/agents/shared.go:327`). `internal/agents` is rewired to consume
  it with zero behavior change (its install/uninstall byte-invariance tests
  must stay green unmodified). **The marker-splice helpers are NOT
  extracted** — this narrows the ROADMAP note's "atomic-write /
  marker-fenced splice" scope deliberately: agents' splice is in-place
  replacement of `<!-- -->` HTML markers with append-on-missing; hooks'
  splice is trimmed-line `#` marker stripping + re-append-at-end +
  delete-when-effectively-empty + shebang seeding + chmod. Forcing one
  abstraction would distort one side or the other; two small correct
  implementations beat one wrong shared one. Document the narrowing in the
  fsatomic package comment.
- **D-10:** **`IsGitRepo` and `HooksDir` live in `internal/gitmeta`** (the
  package Phase 2 D-04 explicitly kept free of query concerns "so Phase 5's
  git sync hooks can reuse it"), following its existing
  `worktree.go` exec contract verbatim. `internal/githooks` consumes
  gitmeta for probes and owns everything hook-specific (marker block,
  splice, install/remove/status, result types). `internal/cli` gains
  `githooks.go` registering the parent + 3 subcommands in root.go's
  AddCommand list.
- **D-11:** **Command shapes:** `githooks install [path]` /
  `githooks remove [path]` / `githooks status [path]`, `[path]` resolved
  via the existing `targetRoot` pattern (Args: MaximumNArgs(1)), matching
  `init`/`sync`/`uninit`. Fixed hook trio (TS `DEFAULT_SYNC_HOOKS`) — no
  hook-selection flags in v1.0. `status` exits 0 whether or not hooks are
  installed (it reports, it doesn't probe); per-hook lines + hooks-dir path,
  plain text. Success/skip wording mirrors TS's installer messages where
  one exists (`Installed git post-commit, post-merge, post-checkout hooks —
  the index refreshes in the background after each.` / `Run \`codegraph
  sync\` anytime to refresh immediately.`); `status` output shape is
  Claude's discretion (TS has no analogue).

### Verification (all three requirements)

- **D-12:** **Real-git fixtures in `t.TempDir()`** (Phase 2 D-15 pattern —
  never fake `.git` layouts; deterministic `GIT_*` env; `t.Skip` when git
  absent). Required cases: fresh install into a bare-hooks repo (file =
  `#!/bin/sh` + block, mode 0755); install over an existing user hook
  (user content preserved, block appended after blank separator);
  re-install idempotency (second run byte-identical); install after a
  prior-version block (strip + re-append, no duplication); remove with
  user content (only our block stripped, byte-preserved remainder); remove
  when effectively empty (file deleted); remove when never installed
  (untouched, no error); `core.hooksPath` honored; linked-worktree resolves
  to the shared common hooks dir; a TS-installed block (verbatim fixture
  string) is detected by `status` and removable. HOOK-01/02's
  idempotency/preservation contracts are proven HERE; TEST-03's formal
  install→edit→remove byte-invariance + piped-stream harness lands Phase 7.
- **D-13:** **Mutation-proof the reachability** (9-recurrence lesson): CLI
  tests drive the real cobra command tree (`githooks install` end-to-end in
  a real fixture repo), not package functions alone; the `init` advisory
  test asserts init PERFORMS the policy check (injectable probe forcing
  "disabled" — reverting the init wiring must turn it red); the uninit
  cleanup test asserts hooks vanish through the real `uninit --force` path.
  No new CI steps expected: everything is normal `go test ./...` packages
  (D-10 Phase-4 precedent) — verify, don't add.
- **D-14:** **Do not assert hook *execution* end-to-end in v1.0** (spawning
  git commits and racing a backgrounded, silenced `codegraph sync` is
  inherently flaky; TS ships zero execution tests). The never-blocks
  property is by construction (`( … & )` subshell + `command -v` guard);
  content-level tests + one optional `sh -n` syntax check on the written
  hook file are sufficient. If the planner wants an execution smoke case,
  gate it `testing.Short()` in `test/integration/` — Claude's discretion.

### Claude's Discretion
- File layout inside `internal/githooks` (single file vs split); result
  struct shape (mirror TS's `{installed, hooksDir, skipped}` or Go-idiomatic
  equivalent); exact `githooks status` output lines; the step-5 pointer
  wording in D-07; whether `sh -n` validation runs in tests; goleak wiring
  if any goroutines appear (none expected — this phase is synchronous file
  I/O).

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase scope & requirements
- `.planning/ROADMAP.md` § "Phase 5: Git Sync Hooks" (lines ~198-210) —
  goal, 3 success criteria, and the Notes line this context narrows (D-09):
  "`internal/fsatomic` extracted from `internal/agents/shared.go`".
- `.planning/REQUIREMENTS.md` § HOOK-01/02/03 (lines ~53-55) — requirement
  text; § TEST-03 (line ~83) + the "TEST-03 → Phase 7" mapping note
  (line ~196) — why the formal byte-invariance harness is NOT here while
  idempotency/preservation tests ARE.
- `.planning/PROJECT.md` § Current Milestone "Git/worktree awareness"
  bullet — "opt-in git sync hooks (post-commit/merge/checkout)".

### TS 1.3.1 reference implementation (white-box ground truth)
**ABSOLUTE external paths — top-level `…/codegraph/dist/` is `.d.ts` stubs
only; the real `.js` lives under the platform sub-package:**
- `/opt/homebrew/lib/node_modules/@colbymchenry/codegraph/node_modules/@colbymchenry/codegraph-darwin-arm64/lib/dist/sync/git-hooks.js`
  — the COMPLETE HOOK-01/02 ground truth (226 lines, read in full this
  session): `MARKER_BEGIN`/`MARKER_END`, `DEFAULT_SYNC_HOOKS`,
  `isGitRepo` (`--is-inside-work-tree`, 5s timeout), `gitHooksDir`
  (`--git-path hooks`, relative→resolve), `markerBlock()` (the verbatim
  7-line block), `stripMarkerBlock` (trimmed-line matching),
  `isEffectivelyEmpty` (blank or `#!` lines only), `chmodExecutable`
  (0755, failure swallowed), `installGitSyncHook` (strip → re-append /
  shebang-seed), `removeGitSyncHook` (delete-when-effectively-empty),
  `isSyncHookInstalled` (`some(contains MARKER_BEGIN)`).
- `…/codegraph-darwin-arm64/lib/dist/installer/index.js` ~lines 467-525 —
  `offerWatchFallback`: the HOOK-03 trigger logic D-07 ports
  (watchDisabledReason gate → warn + frozen-index line → non-repo manual
  hint → already-installed info → select/install), plus the exact success
  (`Installed git … hook(s) — the index refreshes in the background after
  each.`) and failure (`Could not install git hooks(…). Run \`codegraph
  sync\` after changes instead.`) strings.
- `…/codegraph-darwin-arm64/lib/dist/bin/codegraph.js` ~lines 520-546 +
  ~585-588 — both `offerWatchFallback` call sites live inside the
  `init [path]` command (TS surfaces hooks ONLY through init); ~lines
  629-636 — `uninit`'s best-effort `removeGitSyncHook` cleanup + its
  `Removed git … sync hook(s)` info line (D-06's model).

### Current implementation (the extension points)
- `internal/agents/shared.go` — `atomicWriteFile` (~312-345, the D-09
  extraction target: temp-in-dir + rename, mode preservation, 0644 new-file
  default) and `replaceOrAppendMarkedSection`/`removeMarkedSection`
  (~194-295, the helpers that are deliberately NOT reused — read them to
  understand why the semantics differ).
- `internal/gitmeta/worktree.go` — `WorktreeRoot`/`CommonDir`: the exec
  contract (CommandContext 5s, stderr discarded, trim, error→"") that
  D-10's `IsGitRepo`/`HooksDir` must follow; `internal/gitmeta/detect.go`
  for package tone/shape.
- `internal/watch/policy.go` — `WatchDisabledReason(projectRoot, Probe)`
  + injectable `Probe` (Phase 3 D-09/D-10): the D-07 gate; its Probe
  injection is how the init-advisory test forces "disabled"
  deterministically.
- `internal/cli/root.go` (~45-50) — the AddCommand list `newGithooksCmd()`
  joins; `internal/cli/init.go` — `targetRoot`, the success path D-07's
  advisory lands on, and the already-initialized error branch D-08 flags;
  `internal/cli/uninit.go` — the `--force`/confirm flow D-06 extends;
  `internal/cli/sync.go` — the command the hook block invokes (no flag
  changes here; the block calls bare `codegraph sync`).
- `internal/cli/install_test.go`, `internal/agents/*_test.go` — the
  byte-invariance tests that must stay green unmodified through the D-09
  fsatomic rewiring.

### Prior-phase context that carries forward
- `.planning/phases/03-watcher-on-mcp-default/03-CONTEXT.md` § D-11/D-12
  (the verbatim disabled message whose hooks clause this phase makes true;
  the message itself stays untouched per D-08) and § D-09/D-10 (the
  watch-policy port D-07 consumes).
- `.planning/phases/02-status-content-git-worktree-awareness/02-CONTEXT.md`
  § D-03 (the gitmeta git-exec contract D-10 extends), § D-04 (gitmeta
  designed for this phase's reuse), § D-15 (real-git fixture pattern D-12
  follows).

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/agents/shared.go` `atomicWriteFile`: production-hardened
  atomic write with mode preservation — becomes `internal/fsatomic`, shared
  by agents (rewired) and githooks (new consumer).
- `internal/gitmeta`: the established 5s-timeout `git rev-parse` seam —
  gains `IsGitRepo` + `HooksDir`, no new exec conventions invented.
- `internal/watch.WatchDisabledReason` + injectable `Probe`: the HOOK-03
  gate exists since Phase 3; the init advisory only consumes it.
- `internal/cli`'s `targetRoot` + cobra command patterns (`init`/`uninit`/
  `sync`): the `githooks` command tree copies these shapes.
- Phase-2 D-15 real-git fixture helpers (`internal/gitmeta` tests):
  deterministic-git-in-t.TempDir setup to mirror for hook fixtures.

### Established Patterns
- **Verbatim TS strings on parity surfaces** with documented allowed
  divergences at the code — the marker block and installer messages follow
  it; the `githooks` command surface itself is the documented divergence.
- **Best-effort, never-block**: hook install/remove failures degrade with
  a message, never fail init/uninit; the hook script itself backgrounds
  and silences sync so git is never delayed.
- **Mutation-proof the gates** (9 recurrences): reachability tests drive
  the real cobra tree; wiring reverts must turn tests red.
- **Single-seam confinement**: git exec stays in gitmeta; atomic writes in
  fsatomic; nothing else shells out to git or hand-rolls temp-file writes.
- **Explicit CI steps for anything `go test ./...` skips** — nothing new
  needed here (all normal packages); verify existing steps stay green.

### Integration Points
- `internal/cli/root.go` → `newGithooksCmd()` (install/remove/status).
- `internal/cli/init.go` success path → D-07 advisory
  (`watch.WatchDisabledReason` → `gitmeta.IsGitRepo` →
  `githooks.IsInstalled` → pointer line).
- `internal/cli/uninit.go` → post-removal best-effort
  `githooks.Remove` (D-06).
- `internal/agents/shared.go` → imports `internal/fsatomic` (behavioral
  no-op rewiring).
- `internal/githooks` (new) → consumes `internal/gitmeta` probes +
  `internal/fsatomic` writes.

</code_context>

<specifics>
## Specific Ideas

- **Byte-compatibility with TS-installed hooks is a feature, not an
  accident** (D-03): a user swapping binaries mid-project has TS's marker
  blocks in `.git/hooks/` today — the Go binary must detect, replace, and
  remove them seamlessly. Encode this as an explicit fixture (paste TS's
  exact block bytes into a hook file, then drive Go `status`/`remove`).
- **Watch the strip-then-append subtlety**: after a user edits *around* an
  existing block, re-install moves codegraph's block to the end of the
  file. That is TS behavior, not a bug — encode it in a test with an
  explanatory comment so a future "simplification" to in-place replacement
  doesn't silently diverge.
- **`--git-path hooks` already answers the worktree question**: linked
  worktrees share the common hooks dir, `core.hooksPath` overrides
  everything — no hand-rolled `.git/hooks` path joining anywhere.

</specifics>

<deferred>
## Deferred Ideas

- **Interactive "How should CodeGraph keep its index fresh?" select in
  `init`** (TS clack UI, `hook` vs `manual`) — Phase 7 (TUI-03/04
  bubbletea territory); D-07 ships the non-interactive pointer this phase.
- **TEST-03 formal byte-invariance + piped-stream harness** — Phase 7 (the
  first phase where hooks and bubbletea components coexist; requirement
  mapping note).
- **D-12 message wording residual** (serve's verbatim "via `codegraph
  init`" advice vs Go's non-re-runnable init) — Phase 8 SURF-05 divergence
  table (D-08).
- **`affected --stdin/--depth/--filter/--quiet` for git-hook/CI scripting**
  (SURF-04) — Phase 8; the hook block only ever calls bare
  `codegraph sync`.
- **`codegraph install` (agent installer) offering hooks** — TS only
  surfaces hooks from `init`/`uninit`; keep that scoping unless Phase 8's
  flag audit says otherwise.

### Reviewed Todos (not folded)
- `2026-07-14-document-release-cut-procedures-runbook.md` (match score 0.6,
  generic keyword overlap only) — release/maintainer runbook docs belong
  with Phase 8 (release hardening); fifth consecutive review with the
  identical call (Phases 1-4 precedent). The ≥0.4 auto-fold default is
  overridden by the scope guardrail again; retitling the todo so the
  matcher stops flagging it remains recommended.

</deferred>

---

*Phase: 05-git-sync-hooks*
*Context gathered: 2026-07-16*
