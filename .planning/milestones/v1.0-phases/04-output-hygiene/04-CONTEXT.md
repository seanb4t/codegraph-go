# Phase 4: Output Hygiene - Context

**Gathered:** 2026-07-16
**Status:** Ready for planning
**Mode:** --auto (all gray areas auto-selected with recommended options; decisions logged in 04-DISCUSSION-LOG.md)

<domain>
## Phase Boundary

No library log noise ever pollutes command output or the MCP transport:
Pebble's internal WAL/INFO chatter is routed away via an explicit
`pebble.Options.Logger` (INFO→discard) while real error diagnostics are
preserved and still surface (HYG-01), and no library log output ever reaches
MCP stdout — JSON-RPC framing stays clean, all diagnostics go to stderr only
(HYG-02). Small and mechanical by design, but NOT wholesale silence: the
ROADMAP note is explicit that store-corruption errors must never be hidden.

Requirements: HYG-01, HYG-02.

**Not in this phase:** the TTY-gated lipgloss rendering seam and its
import-graph archtest (TUI-01/02/05, Phase 6 — which depends on this phase),
git sync hooks (HOOK, Phase 5), any change to WHAT commands print as their
own product output (only *library* noise is in scope), the >400ms
shared-store-handle lock-contention window left as residual design work by
Phase 3, and any new user-facing verbosity/debug surface (see Deferred).

</domain>

<decisions>
## Implementation Decisions

### Pebble logger routing (HYG-01)
- **D-01:** A small unexported logger type in `internal/graphstore` — the
  sole pebble-aware package (v0.1 D-04a; no other package may see pebble
  types) — implements pebble v2's three-method `base.Logger` interface
  (`Infof` / `Errorf` / `Fatalf`, verified against the pinned
  `pebble/v2@v2.1.6` sources). It is injected via `pebble.Options.Logger` at
  the module's SINGLE `pebble.Open` seam
  (`internal/graphstore/pebble_store.go:147`, today `&pebble.Options{}`).
  The requirement text names `pebble.Options.Logger` — that is the locked
  mechanism; no global stdlib-log hijacking.
- **D-02:** Level routing: `Infof` → discard unconditionally (this is the
  WAL/compaction/memtable chatter); `Errorf` → preserved, written to stderr;
  `Fatalf` → preserve Pebble's default semantics (message to stderr, then
  exit — `DefaultLogger.Fatalf` is stdlib `log.Fatalf`). Do NOT soften
  Fatalf: pebble only Fatalfs on invariant violations where continuing is
  unsafe, and the phase's own guardrail is "real errors are never hidden".
- **D-03:** Set ONLY `Options.Logger` — no `LoggerAndTracer`, no custom
  `EventListener`. Pebble's `EnsureDefaults` derives the default
  `EventListener` from `o.Logger` (options.go ~1469), so the quiet logger
  silences the derived event noise too. One field, one seam, whole surface.
- **D-04:** The preserved error path writes through a package-level
  injectable `io.Writer` seam defaulting to `os.Stderr` (the established
  test-only-seam convention from Phase 3) so tests can capture output;
  production is always stderr per the repo-wide diagnostics rule
  (T-03-07-Leak, `internal/mcp/server.go:63-66`). Pebble-originated error
  lines carry a provenance prefix (exact wording Claude's discretion,
  suggestion: `codegraph: pebble: `).
- **D-05:** NO new env escape hatch (`CODEGRAPH_PEBBLE_LOG`-style) to
  re-enable INFO logs in v1.0 — the discard is unconditional. TS has no
  analogue (better-sqlite3 doesn't chatter, which is exactly why this is a
  parity gap); a new env var is new documented/audited surface for a
  debugging convenience nobody has asked for. Real errors still surface via
  D-02. A general verbose/debug knob is a Deferred idea.

### MCP stdout cleanliness (HYG-02)
- **D-06:** Two-layer enforcement, belt and braces (Phase-3 convention):
  - **(a) Subprocess-harness assertion:** ride the existing
    `test/integration/` harness (Phase-3 D-17..D-21 — real binary, real
    stdio JSON-RPC; do NOT invent a second harness). A real `serve --mcp`
    session that exercises real store activity (the startup reconcile
    `indexer.Sync` + `initialize` → `tools/call`) must produce stdout where
    EVERY line parses as a JSON-RPC frame (`json.Unmarshal` succeeds AND the
    `jsonrpc` field is present); any non-frame byte on stdout fails the test.
  - **(b) Structural stdout guard:** a new archtest mirroring the two
    existing precedents (`internal/graphstore/archtest/import_graph_test.go`,
    `internal/migrate/archtest/modernc_confinement_test.go`) fails if any
    serve-reachable non-CLI package (`internal/mcp`, `internal/graphstore`,
    `internal/daemon`, `internal/watch`, `internal/indexer`,
    `internal/query`) references `os.Stdout`, calls bare stdout-writing
    `fmt.Print*` (no explicit writer), or calls `log.SetOutput`.
    `internal/cli` is EXCLUDED — it legitimately renders command output to
    `cmd.OutOrStdout()` (03-PATTERNS.md output-discipline pattern).
    Mechanism (go/ast walk vs token scan) is Claude's discretion; it must be
    a normal Go test so `go test ./...` runs it.
  - Note: `internal/daemon`'s existing `log.Printf` calls are COMPLIANT —
    stdlib log defaults to stderr; the guard targets stdout references, not
    stdlib-log usage.
- **D-07:** HYG-02 is expected to be a guarantee-and-regression-lock, not a
  behavior change: scouting found zero stdout writers in the guarded
  packages today (the only `os.Stdout` match is a comment). If the archtest
  or harness DOES surface a real violation, fixing it is in scope for this
  phase — that's the point of running the guard before claiming the
  requirement.

### Verification (both requirements)
- **D-08:** Mutation-proof the HYG-01 wiring (the 8×-recurred green-suite
  lesson: assert the wiring, not a replica). A test must assert
  `graphstore.Open` actually passes the quiet logger — reverting
  pebble_store.go:147 to `&pebble.Options{}` must turn it red. Plus a
  behavioral test: capture the D-04 seam writer during store activity that
  provokes pebble INFO output (open/write/flush/close cycles emit WAL/job
  lines under the default logger) and assert zero pebble noise, while a
  directly-invoked `Errorf` still reaches the writer.
- **D-09:** One cheap CLI-side behavioral check: a normal command driven
  end-to-end (e.g. `sync` or `status` via the subprocess harness) asserts
  its stderr carries no pebble-shaped noise (no `[JOB `-style / WAL / 
  compaction lines) — absence-of-substring on noise shapes only, NOT
  emptiness (legit codegraph warnings may appear). Placement and command
  choice are Claude's discretion.
- **D-10:** No new CI steps needed: the archtest lives under a normal
  package (covered by `go test ./...`) and the harness additions live in
  `test/integration/` (covered by Phase 3's explicit named CI step). Verify
  both remain green rather than adding steps.

### Claude's Discretion
- Logger type name and file placement inside `internal/graphstore`; exact
  provenance-prefix wording; archtest implementation mechanism (go/ast vs
  scanner); which command the D-09 CLI-noise check drives and where it
  lives; whether the quiet logger buffers/rate-limits repeated Errorf lines
  (only if trivially cheap — otherwise pass-through).

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase scope & requirements
- `.planning/ROADMAP.md` § "Phase 4: Output Hygiene" (lines ~174-185) —
  goal, the 2 success criteria, and the Notes guardrail: "do NOT
  wholesale-silence — route INFO→discard explicitly so store-corruption
  errors are never hidden".
- `.planning/REQUIREMENTS.md` § HYG-01/HYG-02 (lines ~59-60) — the exact
  requirement text (HYG-01 names `pebble.Options.Logger` as the mechanism).
- `.planning/PROJECT.md` § Current Milestone "Output hygiene" bullet —
  "silence Pebble WAL log noise on stderr; TTY-gate all styling" (the
  TTY-gating half is Phase 6, not this phase).

### Pebble v2 ground truth (pinned dependency sources)
**Read from the module cache (`go env GOMODCACHE`), version-locked to
`github.com/cockroachdb/pebble/v2@v2.1.6`:**
- `…/pebble/v2@v2.1.6/internal/base/logger.go` — the `Logger` interface
  (line ~19: exactly `Infof`/`Errorf`/`Fatalf`), `DefaultLogger` (logs to Go
  stdlib log → stderr; `Fatalf` = `log.Fatalf`), and `InMemLogger` (useful
  test double shape).
- `…/pebble/v2@v2.1.6/options.go` — `Options.Logger` + `LoggerAndTracer`
  fields (~869-877; LoggerAndTracer wins if non-nil — we set neither trace
  path), and `EnsureDefaults` (~1464-1469): `o.Logger = DefaultLogger` when
  nil and `o.EventListener.EnsureDefaults(o.Logger)` — proof that setting
  `Options.Logger` alone covers derived event-listener noise (D-03).

### Current implementation (the extension points)
- `internal/graphstore/pebble_store.go` — `Open` (~141-157): the module's
  ONLY `pebble.Open` call site (line 147, `&pebble.Options{}` — the D-01
  injection point), the CR-01 bounded lock-retry loop AROUND it (the quiet
  logger must not disturb retry/classification semantics), and
  `classifyOpenError`/`ErrStoreLocked` (~110-127).
- `internal/graphstore/locked_unix.go` / `locked_windows.go` — build-tagged
  lock classifiers; context for why Open's error path is delicate.
- `internal/mcp/server.go` lines ~63-66 — the T-03-07-Leak comment: stdout
  is reserved for the MCP JSON-RPC transport; diagnostics must use stderr.
  This is the HYG-02 rule already stated in code; the phase mechanizes it.
- `internal/cli/serve.go` — all serve-path diagnostics already go through an
  injected `stderr io.Writer` / `cmd.ErrOrStderr()` (lines ~93, 116-151,
  220-242); the reconcile `indexer.Sync` block (runs pre-handshake — its
  store open is a pebble-noise source on the serve path).
- `internal/daemon/daemon.go` — stdlib `log.Printf` diagnostics (~233, 277,
  410-424): stderr-compliant today; do not convert, just don't regress.
- `internal/cli/query.go` (~27) — the output-discipline comment for the CLI
  exclusion boundary in D-06(b).

### Test patterns to mirror
- `internal/graphstore/archtest/import_graph_test.go` and
  `internal/migrate/archtest/modernc_confinement_test.go` — the archtest
  precedents D-06(b) mirrors (normal Go test that fails the build on a
  structural violation).
- `test/integration/` — the Phase-3 subprocess harness (TestMain-built real
  binary, mcp-go stdio client). D-06(a) and D-09 add cases HERE.
- `.github/workflows/ci.yml` — the explicit `go test ./test/integration/...`
  and `go test ./testdata/golden/...` steps (D-10: no new steps, just stay
  inside them).

### Prior-phase context that carries forward
- `.planning/phases/03-watcher-on-mcp-default/03-CONTEXT.md` § D-17..D-21
  (the harness this phase rides), § "Established Patterns" (best-effort
  never-block; mutation-test the gates; explicit CI steps).
- `.planning/phases/02-status-content-git-worktree-awareness/02-CONTEXT.md`
  § D-14/D-15 context only if the harness cases need fixtures — no direct
  dependency expected.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/graphstore/pebble_store.go:147` — the single `pebble.Open` seam:
  HYG-01 is a one-field change at one call site plus a small logger type;
  no other package touches pebble (D-04a confinement, archtest-enforced).
- The two existing archtests: copy-paste-adjacent pattern for the D-06(b)
  stdout guard.
- `test/integration/` harness (Phase 3): already spawns the real binary and
  drives real stdio JSON-RPC — the D-06(a) frame-purity assertion is a new
  case in an existing file set, not new machinery.
- Phase-3 test-only seam convention (package-level fn/writer fields,
  nil/default in production): the D-04 stderr seam follows it.
- `pebble/v2`'s own `base.InMemLogger`: a ready-made captured-logger shape
  for behavioral tests (or trivially hand-rolled — discretion).

### Established Patterns
- **Diagnostics → stderr, never stdout** (T-03-07-Leak; serve.go's injected
  stderr writer): HYG-02 mechanizes an already-stated rule.
- **Mutation-proof the gates** (Phases 1-3, 8 recurrences): every "X is
  wired" test must go red when X is reverted at its root cause (D-08).
- **Single-seam confinement** (graphstore is the only pebble-aware package):
  the logger lives there and nowhere else.
- **Best-effort, never-block**: logging changes must never make Open/Sync
  fail or slow down; discard is cheaper than the current default logger.
- **Explicit CI steps for anything `go test ./...` skips** (GOLDEN-01):
  satisfied structurally here — archtest is a normal package, harness cases
  land in the already-explicit integration step.

### Integration Points
- `internal/graphstore/pebble_store.go` `Open` → `&pebble.Options{Logger:
  quietLogger{…}}` (D-01/D-02/D-03) + package-level stderr seam (D-04).
- `internal/graphstore/archtest/` (or sibling archtest dir) → new stdout
  guard over the six serve-reachable packages (D-06b).
- `test/integration/` → new JSON-RPC frame-purity case (D-06a) + CLI
  noise-absence case (D-09).

</code_context>

<specifics>
## Specific Ideas

- The HYG-02 guarantee should be **provable two ways** like Phase 3's
  WATCH-02: structurally (the archtest makes stdout references in
  serve-reachable packages a build failure) and end-to-end (the harness
  parses every real stdout line as a JSON-RPC frame). Neither alone
  survived past phases' review rounds; together they have.
- Keep the diff minimal at the Open seam — the CR-01 retry loop and
  ErrStoreLocked classification around line 147 were hard-won in three
  Phase-3 review rounds; the logger injection must not restructure them.

</specifics>

<deferred>
## Deferred Ideas

- **Verbose/debug knob to re-enable pebble INFO logs** (env or flag) —
  rejected for v1.0 (D-05); revisit only if real-world store debugging
  demands it (Phase 8 surface-reconciliation candidate at the earliest).
- **TUI-01 lipgloss/bubbletea import-graph archtest** (Phase 6) — the
  D-06(b) stdout guard is a sibling precedent, not a substitute; Phase 6
  should mirror it for the Charm packages.
- **Shared in-process store handle** to close the residual >400ms
  lock-contention window (Phase-3 residual) — future design work, untouched
  by this phase.

### Reviewed Todos (not folded)
- `2026-07-14-document-release-cut-procedures-runbook.md` (match score 0.4,
  generic keyword match only) — release/maintainer runbook docs belong with
  Phase 8 (release hardening), the identical call made in Phases 1, 2, and
  3. Fourth consecutive review; retitle the todo so the matcher stops
  flagging it.

</deferred>

---

*Phase: 04-output-hygiene*
*Context gathered: 2026-07-16*
