# Phase 3: Watcher-on-MCP Default - Context

**Gathered:** 2026-07-16
**Status:** Ready for planning
**Mode:** --auto (all gray areas auto-selected with recommended options; decisions logged in 03-DISCUSSION-LOG.md)

<domain>
## Phase Boundary

`serve --mcp` runs live in-process auto-sync **by default** (matching TS 1.3.1's
auto-sync), with a `--no-watch` opt-out and a WSL2/slow-filesystem watch-policy
auto-off honoring env precedence (`CODEGRAPH_NO_WATCH` / force-on) — restoring
the live-sync experience with **zero config change** (`install` already writes
the byte-identical bare `serve --mcp` invocation for all 8 agents) and without
ever delaying the MCP handshake or first-tool availability. Concurrent
`serve --mcp` sessions on one repo converge to a single writer, goleak-clean.
Plus TEST-04: a subprocess integration harness that spawns the real binary and
drives its real transports (CLI argv + MCP stdio JSON-RPC), anchored on the
Phase-2 CR-01 worktree-notice case.

Requirements: WATCH-01, WATCH-02, WATCH-03, WATCH-04, TEST-04.

**Not in this phase:** the standalone daemon picker / lifecycle UX (DMON-01..04,
Phase 7), git sync hooks (HOOK reqs, later phase), TS's multi-session pool/proxy
sharing (DMON-FUT-01, explicitly out of v1.0), the TS liveness watchdog (#850 —
Phase 7 DMON-03 territory), `CODEGRAPH_WATCH_DEBOUNCE_MS` parity (see Deferred).

</domain>

<decisions>
## Implementation Decisions

### Flag surface & precedence (WATCH-01)
- **D-01:** Watcher defaults **ON** for `serve --mcp` whenever an index exists
  (`hasIndex`); the absent-index MCP-03 case skips the watcher exactly as today.
- **D-02:** Add `--no-watch` bool flag (opt-out), help text mirroring TS's:
  "Disable the file watcher (no auto-sync; useful on slow filesystems like WSL2
  /mnt drives)". TS ships ONLY `--no-watch` (bin/codegraph.js ~1570).
- **D-03:** The existing v0.1 `--watch` flag is **repurposed, not removed**: it
  becomes the explicit force-on — the flag analogue of `CODEGRAPH_FORCE_WATCH=1`
  (overrides the WSL2 auto-off; a plain default-on run does NOT override it).
  This keeps v0.1 invocations working (`--watch` still yields a watcher) while
  giving the force-on escape hatch a discoverable CLI form TS lacks.
- **D-04:** Precedence, first match wins (mirrors TS watch-policy.js exactly,
  with flags folded in at the strength of their env twins):
  1. `--no-watch` flag OR `CODEGRAPH_NO_WATCH=1` → off (opt-out always wins)
  2. `--watch` flag OR `CODEGRAPH_FORCE_WATCH=1` → on (beats auto-detection)
  3. WSL2 + `/mnt/[a-z]` drive → off (auto-detection)
  4. default → on
  `--no-watch --watch` together is a hard flag error (contradiction), not a
  silent precedence pick.
- **D-05:** Unlike TS (which routes `--no-watch` by literally setting
  `process.env.CODEGRAPH_NO_WATCH='1'`), we pass explicit values into the policy
  function and **never mutate the process env** — the watcher is in-process; no
  subprocess needs to inherit it. (TS's env-routing trick is a Commander
  artifact, not a behavior to port.)

### Handshake-path budget (WATCH-02)
- **D-06:** ALL watcher startup moves inside the background goroutine:
  `daemon.New`, the watch-policy check, lock acquisition, and `watch.Open`'s
  recursive fsnotify walk. After this phase the code path between process start
  and `server.ServeStdio(s)` contains **zero watcher code by construction** —
  that is the WATCH-02 guarantee, provable structurally. (Today `daemon.New`
  runs on-path and only `d.Run` is in the goroutine; on WSL2 the recursive walk
  would burn on-path I/O in TS — Go's goroutine sidesteps the event-loop stall,
  but the policy still gates it for parity and resource sanity.)
- **D-07:** The reconcile `indexer.Sync` (serve.go ~88–93) **stays synchronous
  pre-handshake**. It is NOT watcher startup — it is the shipped D-06/SYNC-03
  guarantee ("the first codegraph_explore reads a current graph"), stat-prefiltered
  cheap by construction. Changing its ordering is out of scope for this phase;
  WATCH-02's criterion targets watcher startup only.
- **D-08:** WATCH-02 is verified two ways: (a) a structural unit test pinning
  serve.go's real wiring via an extracted seam function (the WR-01
  `serveServerPaths` precedent — extract e.g. a `serveWatchStart`-shaped helper
  so the test asserts serve.go PERFORMS the deferred start, mutation-testable:
  moving watcher code back on-path must turn it red); (b) the TEST-04 subprocess
  harness asserts `initialize` → `tools/list` completes promptly with the
  default-on watcher.

### Watch-policy port (WATCH-03)
- **D-09:** New `internal/watch/policy.go` — the policy lives with the watcher,
  mirroring TS's centralization ("the watcher, the MCP server (for diagnostics),
  and the installer all agree"). One exported function, TS-shaped:
  `WatchDisabledReason(projectRoot string, …) string` — empty string = watch
  runs; non-empty = short human-readable reason. Probes (env lookup, WSL
  detection) injectable for tests, with the cached-WSL-detection reset analogue
  of TS's `__resetWslCacheForTests`.
- **D-10:** Detection semantics ported exactly from
  `sync/watch-policy.js` (read it, don't paraphrase):
  - WSL detection: `GOOS == "linux"` only; `WSL_DISTRO_NAME` or `WSL_INTEROP`
    env → true (no I/O); else `/proc/version` lowercased contains `microsoft`
    or `wsl`; read failure → false. Result cached (sync.Once-style).
  - Windows-drive mount: regex `^/mnt/[a-z](/|$)` case-insensitive against the
    normalized project root — deliberately single-letter only, so `/mnt/wsl/…`
    stays fast-path.
  - Env values: TS checks `=== '1'` exactly — port that strictness (only the
    string "1" triggers), don't invent truthiness.
- **D-11:** Enforcement point is the shared watcher-driving seam
  (`internal/daemon.Run`, which drives `watch.Open`) so BOTH the in-process
  `serve --mcp` watcher AND the standalone `codegraph daemon` honor the policy
  — exactly as TS enforces inside `watcher.start()` (sync/watcher.js ~301).
  `serve --mcp` additionally emits the human-visible disabled message; the
  standalone daemon logs it at its own verbosity.
- **D-12:** Disabled-reason strings and the MCP stderr message are **verbatim
  TS** (mcp/engine.js ~256): `[CodeGraph MCP] File watcher disabled — {reason}.
  The graph will not auto-update; run ` + backtick-`codegraph sync`-backtick +
  ` (or install the git sync hooks via ` + backtick-`codegraph init`-backtick +
  `) to refresh.` — TS's own comment pins this wording for log-driven
  dashboards. The hooks clause references functionality landing later in v1.0
  (HOOK reqs); acceptable because v1.0.0 ships them together. Message goes to
  **stderr** (model-invisible; MCP hosts surface it in logs).
- **D-13:** Reason strings verbatim: `CODEGRAPH_NO_WATCH=1 is set` and
  `project is on a WSL2 /mnt/ drive, where recursive fs.watch is too slow to be
  reliable` — keep TS's `fs.watch` wording? **No** — this one names a Node API.
  Use the TS sentence with `fs.watch` → `file watching` (documented allowed
  divergence, D-02-style: the sentence is human-facing stderr, never parsed;
  naming a Node API in a Go binary is wrong). Everything else byte-identical.

### Concurrent-session convergence (WATCH-04)
- **D-14:** Replace defer-once with **defer-and-retry**: on `ErrLockLive` the
  session logs the existing "deferring to it" line once, then retries lock
  acquisition on a slow jittered cadence (~30s ± jitter) inside the same
  background goroutine. If the lock-holding writer exits, a surviving session
  acquires and starts watching — true convergence: at most one writer always
  (lock), exactly one writer eventually while ≥1 session lives (retry). Today's
  code gives up forever after one attempt, leaving zero watchers after the
  holder exits.
- **D-15:** goleak coverage: the retry ticker + goroutine tear down cleanly via
  the existing `cancelWatch`/`watchDone` teardown on ServeStdio return; extend
  the existing goleak soak pattern (internal/daemon/soak_test.go,
  internal/watch/soak_test.go) to the retry loop and to two-concurrent-sessions
  convergence.
- **D-16:** **Researcher must confirm `acquire()`'s stale-lock/liveness
  semantics** (internal/daemon/lock.go; v0.1 shipped `codegraph unlock` and
  lock liveness) before finalizing the retry design — retrying into a stale
  lock must self-heal, not spin; a crashed holder must not wedge every
  surviving session.

### Subprocess integration harness (TEST-04)
- **D-17:** Location: `test/integration/` — a **normal Go package**, so
  `go test ./...` includes it (NEVER under `testdata/` — GOLDEN-01: the go tool
  silently ignores testdata dirs; that lie cost Phase 2 a Critical). Guard slow
  cases with `testing.Short()`. CI gets an **explicit named step**
  `go test ./test/integration/...` alongside the existing explicit
  `go test ./testdata/golden/...` step — belt and braces, so a future refactor
  cannot silently drop it.
- **D-18:** `TestMain` builds the real release binary once per test run
  (`go build -o <t.TempDir or shared tmp>/codegraph .`) — hermetic, no reliance
  on a pre-built artifact or PATH.
- **D-19:** The MCP session drives the **spawned binary over real stdio
  JSON-RPC** using mcp-go's client package (already a dependency): real
  `initialize` → `tools/list` → `tools/call` against the real production
  process with real argv and cwd. The CLI side runs the same binary via
  `exec.Command` argv. This is the seam in-process `BuildServer`→`CallTool`
  tests structurally bypass (serve.go's cwd/argv → path-derivation → handler
  wiring — the exact CR-01 hole).
- **D-20:** Anchor case (mandatory, from the requirement text): real git repo
  fixture in `t.TempDir` (Phase-2 D-15 real-git pattern — fake `.git` dirs
  cannot reproduce the 4-gate cascade), `git worktree add` into
  `.claude/worktrees/<name>`, `init`+`index` via the subprocess binary, then a
  `serve --mcp` session with **cwd inside the worktree**: the
  `codegraph_explore` payload must carry the U+26A0 notice (use the Phase-2
  bare-glyph helper — bare `⚠` is a byte-prefix of `⚠️`, naive Contains matches
  both); a control session from the main checkout must show none.
  Mutation-proof it: reintroducing CR-01 at its root cause (passing repoPath
  for both BuildServer args in serve.go) must turn this test red.
- **D-21:** WATCH coverage rides the same harness: one case asserts default-on
  `serve --mcp` (no watch flag in argv) completes the handshake promptly; one
  asserts `CODEGRAPH_NO_WATCH=1` in the child env disables the watcher
  (observed via the verbatim stderr disabled message). Remember
  `CODEGRAPH_MCP_TOOLS` allowlist env when calling companion tools from the
  harness (Phase-2 gotcha: `codegraph_callers` etc. need it; `codegraph_explore`
  is always visible).

### Claude's Discretion
- Goroutine/channel structure inside serve.go's watcher block; retry interval
  constant and jitter shape; exact file layout and test naming inside
  `test/integration/`; whether the policy check logs before or after lock
  acquisition in the goroutine (both are off-path).

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase scope & requirements
- `.planning/ROADMAP.md` § "Phase 3: Watcher-on-MCP Default" — goal, the 5
  success criteria, the "default flip is ~2 lines but MUST be bundled with the
  watch-policy port" note, and the TEST-04 mapping rationale.
- `.planning/REQUIREMENTS.md` § WATCH-01..04 (lines ~33–36) and § TEST-04
  (line ~84, full text — it IS the harness spec) + the TEST-04 note (~line 189).
  **Also read the "Out of Scope" table row "Auto-spawning daemons on
  `serve --mcp`"** (~line 122) — the guardrail: in-process watcher only, no
  daemon auto-spawn in this phase.
- `.planning/PROJECT.md` § Current Milestone "Watcher-on-MCP by default" bullet.

### TS 1.3.1 reference implementation (white-box ground truth)
**ABSOLUTE external paths — top-level `…/codegraph/dist/` is `.d.ts` stubs
only; the real `.js` lives under the platform sub-package:**
- `/opt/homebrew/lib/node_modules/@colbymchenry/codegraph/node_modules/@colbymchenry/codegraph-darwin-arm64/lib/dist/sync/watch-policy.js`
  — the COMPLETE WATCH-03 ground truth: `detectWsl` (env vars → /proc/version,
  cached), `isWindowsDriveMount` (`/^\/mnt\/[a-z](\/|$)/i`, deliberately
  excludes `/mnt/wsl/…`), `watchDisabledReason` precedence (NO_WATCH → 
  FORCE_WATCH → WSL2+mnt), exact reason strings, `'1'`-strict env checks, and
  the #199 issue narrative in the header comment.
- `…/codegraph-darwin-arm64/lib/dist/mcp/engine.js` ~lines 185–270 —
  `startWatching()` (policy check + verbatim disabled stderr message + the
  "watcher is per-engine … daemon path collapses N inotify sets to one"
  comment), called from the lazy project-open path (never on the handshake),
  followed by `catchUpSync()`; also the `CODEGRAPH_WATCH_DEBOUNCE_MS` handling
  (~266, deferred — see Deferred Ideas).
- `…/codegraph-darwin-arm64/lib/dist/sync/watcher.js` ~lines 295–315 — policy
  enforced INSIDE `watcher.start()` (why our enforcement point is daemon.Run).
- `…/codegraph-darwin-arm64/lib/dist/bin/codegraph.js` ~lines 1570–1576 — the
  `--no-watch` option help text and the Commander env-routing artifact
  (`process.env.CODEGRAPH_NO_WATCH = '1'`) we deliberately do NOT port (D-05).

### Current implementation (the extension points)
- `internal/cli/serve.go` — the whole file (153 lines): `serveServerPaths` +
  its WR-01 doc comment (the extract-a-seam-so-tests-pin-real-wiring precedent
  D-08 reuses), the CR-01 comment on `BuildServer(hasIndex, allowlist,
  repoPath, start)` (the two DELIBERATELY DISTINCT path args — the harness
  anchor guards this), the reconcile Sync block (D-07 keeps it), and the
  current `--watch` block (D-01..06 rework it).
- `internal/daemon/daemon.go` — `New` (cheap: Abs + one Stat), `Run` (lock
  acquire → `watch.Open` → debounce loop; returns `ErrLockLive` immediately —
  the defer-once behavior D-14 replaces), teardown invariants (CR-01 comment:
  no goroutine outlives Run, no Sync writing when lock released).
- `internal/daemon/lock.go` + `internal/daemon/lock_test.go` — `acquire`/
  `release` and stale-lock/liveness semantics (D-16 research target).
- `internal/watch/watcher.go` — `Open` (the recursive add walk that must stay
  off the handshake path), `Run(ctx, deb)`, `Close`.
- `internal/daemon/soak_test.go`, `internal/watch/soak_test.go` — the goleak
  soak patterns D-15 extends.
- `internal/cli/serve_test.go` — the WR-01 test pinning `serveServerPaths`;
  the new watch-wiring seam test sits beside it.
- `internal/agents/*.go` (claude/cursor/codex/gemini/kiro/opencode/…) — the 8
  agents' `serve --mcp` config writers. **MUST NOT change** — the whole point
  of WATCH-01 is that these bare invocations light up with zero config change.
- `.github/workflows/ci.yml` — the explicit `go test ./testdata/golden/...`
  step (GOLDEN-01 fix) the new integration step lands beside.

### Prior-phase context that carries forward
- `.planning/phases/02-status-content-git-worktree-awareness/02-CONTEXT.md`
  § D-14 (Engine startPath plumbing), § D-15 (real-git fixtures in t.TempDir —
  the TEST-04 anchor reuses this), § "Claude's Discretion".
- `.planning/phases/01-behavioral-parity-explore-node/01-CONTEXT.md` § D-02 —
  the normalized parity oracle + documented-allowed-divergence convention that
  D-13's `fs.watch`→`file watching` wording change invokes.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/daemon` lockfile (`acquire`/`release`, `ErrLockLive`): the entire
  WATCH-04 single-writer mechanism already exists — this phase adds only the
  retry loop and the default flip; do not invent a second coordination scheme.
- `internal/watch.Open` + `Debouncer` + `daemon.Run`'s watch/debounce/Sync
  loop: the complete watcher engine; WATCH-01 changes only when it starts, not
  how it runs.
- `serveServerPaths` + `serve_test.go` (WR-01): the proven pattern for making
  serve.go's own wiring testable — extract the watch-start decision the same way.
- Phase-2 real-git fixture pattern (D-15) + `.claude/worktrees/` as the
  motivating true positive: the TEST-04 anchor case is a direct reuse.
- Phase-2 bare-glyph U+26A0 assertion helper: required for the anchor case
  (`⚠` is a byte-prefix of `⚠️`; naive `strings.Contains` matches both).
- mcp-go (existing dep) ships a client package: the harness's stdio JSON-RPC
  side needs no new dependency. Zero new deps expected this phase.

### Established Patterns
- **Best-effort, never-block** (Phase-2 WORK-03; v0.1 WR-04): watcher failures
  log to stderr and degrade — they never fail `serve --mcp` or delay tools.
- **Verbatim TS strings on parity surfaces**, with documented allowed
  divergences at the code (StatusResult-style decision-table comments).
- **Mutation-test the gates** (Phase-2 lesson, 6th/7th recurrence): every
  "serve.go performs X" test must go red when X is reverted at its root cause —
  a test that replicates the derivation without asserting serve.go performs it
  proves nothing.
- **Explicit CI steps for anything `go test ./...` can silently skip**
  (GOLDEN-01): run BOTH `go test ./...` and the named golden + integration steps.
- **Drive the real entry point; never infer reachability across surfaces**
  (the dominant Phase-1/2 meta-lesson — TEST-04 exists to mechanize it).

### Integration Points
- `internal/cli/serve.go` RunE → watcher block reworked: policy + daemon.New +
  lock + watch all inside the background goroutine; flags `--no-watch` (new) /
  `--watch` (repurposed force-on) → policy inputs.
- `internal/watch/policy.go` (new) ← consumed by `internal/daemon.Run`
  (enforcement) and `internal/cli/serve.go` (human-visible message).
- `internal/daemon.Run` → gains policy gate + (via serve's goroutine) the
  ErrLockLive retry cadence.
- `test/integration/` (new) → spawns the TestMain-built binary; asserts CR-01
  anchor, default-on watcher, NO_WATCH env off-switch.
- `.github/workflows/ci.yml` → new explicit integration-test step.

</code_context>

<specifics>
## Specific Ideas

- The WATCH-02 guarantee should be **provable by construction**: after this
  phase, no watcher code executes between process start and
  `server.ServeStdio` — reviewers can verify it by reading serve.go top to
  bottom, and the seam test enforces it mechanically.
- The CR-01 anchor case is the harness's reason to exist — it retroactively
  guards Phase 2's wiring and must be mutation-proven (revert serve.go's
  two-arg distinction → test goes red) before the phase can claim TEST-04.

</specifics>

<deferred>
## Deferred Ideas

- **`CODEGRAPH_WATCH_DEBOUNCE_MS` env parity** (TS #403, mcp/engine.js ~266:
  clamped [100ms, 60s], logged when active) — real TS surface but not in
  WATCH-01..04's text; Phase 8 surface-reconciliation candidate.
- **TS liveness watchdog** (#850, `CODEGRAPH_NO_WATCHDOG`) and **PPID
  watchdog** — Phase 7 (DMON-03) territory; noticed in mcp/server.js while
  reading the watcher wiring.
- **TS pool/proxy multi-session sharing** (one CodeGraph + watcher shared via
  spawned proxy) — DMON-FUT-01, explicitly out of v1.0; our lockfile+retry
  model is the v1.0 answer.

### Reviewed Todos (not folded)
- `2026-07-14-document-release-cut-procedures-runbook.md` (match score 0.6) —
  release/maintainer runbook docs belong with Phase 8 (release hardening), the
  same call made in Phases 1 and 2; folding docs work into a watcher phase
  would be scope creep. Third consecutive review — consider retitling the todo
  so the matcher stops flagging it.

</deferred>

---

*Phase: 03-watcher-on-mcp-default*
*Context gathered: 2026-07-16*
