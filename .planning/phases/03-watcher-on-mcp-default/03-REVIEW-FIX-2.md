---
phase: 03-watcher-on-mcp-default
fixed_at: 2026-07-16T18:55:00Z
review_path: .planning/phases/03-watcher-on-mcp-default/03-REVIEW-2.md
iteration: 2
findings_in_scope: 3
fixed: 3
skipped: 0
status: all_fixed
---

# Phase 3: Code Review Fix Report (Iteration 2)

**Fixed at:** 2026-07-16T18:55:00Z
**Source review:** .planning/phases/03-watcher-on-mcp-default/03-REVIEW-2.md
**Iteration:** 2

**Summary:**
- Findings in scope: 3 (fix_scope: critical_warning — IN-01..IN-05 excluded)
- Fixed: 3
- Skipped: 0

## Fixed Issues

### CR-01 + WR-01: Lock-held classification moved to the Open seam behind an exported `ErrStoreLocked` sentinel, with a build-tagged Windows arm

**Files modified:** `internal/graphstore/pebble_store.go`, `internal/graphstore/locked_unix.go` (new), `internal/graphstore/locked_windows.go` (new), `internal/cli/serve.go`, `internal/daemon/daemon.go`
**Commit:** 7699e9c

**One commit for two findings, deliberately:** the reviewer's own recommended fix for CR-01 ("Better still, combine with WR-01's fix below") and for WR-01 ("Classify once … inside Open's retry loop … and export a sentinel") is the same refactor at the same seam — splitting it would leave a non-compiling or artificially shimmed intermediate commit. Per the orchestrator guardrail, the unified architecture was applied:

1. **Exported sentinel:** `graphstore.ErrStoreLocked` (`errors.New("graphstore: store lock held")`). `Open`'s retry loop now runs every `pebble.Open` failure through `classifyOpenError`, which wraps lock-class errors as `fmt.Errorf("%w: %v", ErrStoreLocked, err)` (original text preserved for diagnostics) and returns everything else unchanged. Retry semantics are byte-identical to round 1 (5 attempts × 100ms backoff, sleep before attempts 2–5 only, immediate break on non-lock errors); the only behavioral delta is that an exhausted lock error now surfaces `errors.Is`-able instead of raw.
2. **Platform-correct raw matching, confined to one internal helper:** `isLockHeldOS` in build-tagged files. `locked_unix.go` (`//go:build !windows`) keeps the in-process `"lock held by current process"` string form and the fcntl `EAGAIN`/`EWOULDBLOCK` errnos — **`EACCES` dropped** per WR-01 (POSIX allows it for F_SETLK only on non-shipped systems; on every release target it means a genuine permission failure, e.g. `os.Create(LOCK)` on an unwritable store dir, which must stay fatal). `locked_windows.go` (`//go:build windows`) matches `syscall.Errno(32)` (`ERROR_SHARING_VIOLATION` — the single form every collision takes via pebble's `CreateFile(share=0)`, same-process and cross-process alike; verified against pinned `pebble/v2@v2.1.6` `vfs/file_lock_windows.go`). No windows-only syscall constants appear in cross-platform code; `x/sys/windows` returns plain `syscall.Errno`, so no new import is needed.
3. **Consumers drop all chain sniffing:** `daemon.Run`'s requeue branch and `serve.go`'s startup-reconcile downgrade now use `errors.Is(err, graphstore.ErrStoreLocked)`. `IsLockHeld` is deleted entirely — no API remains that could be misapplied to an arbitrary error chain. Since the sentinel is only ever attached inside `graphstore.Open` (and `indexer.Sync` propagates that error unwrapped — verified at `internal/indexer/sync.go:52,470` and `pipeline.go:116`), an `EACCES` from `Discover`'s WalkDir, `MatchFile`'s reads, or `contentHash` structurally cannot match: serve.go's "every non-lock error stays fatal" contract is restored, and the daemon burns zero requeues on permanent permission failures.

Round-1 semantics preserved everywhere else: bounded retries (5×100ms), bounded flush requeue (max 5, ctx-gated, sidecar left in place on every failure path — BL-01 honored), never-block serve, zero new dependencies, TS strings verbatim, `internal/agents/*.go` untouched, D-04a pebble confinement intact (archtest green — the raw error-shape matching still lives only in graphstore).

### WR-02: Direct unit tests for the classifier and Open's retry loop

**Files modified:** `internal/graphstore/open_lock_test.go` (new), `internal/graphstore/locked_unix_test.go` (new), `internal/graphstore/locked_windows_test.go` (new)
**Commit:** 1bdcd9c

**Applied fix:** All three coverage gaps closed with deterministic unit tests (new test files are required by the finding; documented per fixer contract):

- **In-process string form, pinned by a real double-open** (`TestOpenSecondOpenInProcessReturnsErrStoreLocked`): opens the same store twice on the host platform; asserts the second `Open` fails, satisfies `errors.Is(err, ErrStoreLocked)`, and consumed the full bounded retry budget (elapsed ≥ 4×100ms — a deterministic lower bound, no upper bound, so it cannot flake on slow machines). A pebble version bump that rewords the unexported message now turns this test red at unit speed instead of silently disabling the requeue/downgrade paths.
- **Success-after-contention** (`TestOpenConvergesWhenHolderCloses`): holder closes ~150ms into the second `Open`'s retry window; asserts convergence to success.
- **Errno forms, synthesized:** `locked_unix_test.go` (`//go:build !windows`) pins `isLockHeldOS` true for fcntl `EAGAIN` **wrapped in the `fs.PathError` shape pebble actually produces** (executable proof of the `errors.Is` traversal the review noted was only manually verified) and bare `EWOULDBLOCK`; false for `EACCES` (the WR-01 regression pin), `ErrNotFound`, and arbitrary errors; plus the sentinel-wrap round-trip preserving original text. `locked_windows_test.go` (`//go:build windows`) pins `Errno(32)` bare and PathError-wrapped as true; `ERROR_ACCESS_DENIED` (5), the unix in-process string, and `ErrNotFound` as false — compiled only on windows, held cross-GOOS by the vet check below.
- **Platform-neutral shared path** (`TestClassifyOpenErrorSharedPath`): synthesized `fs.PathError{Err: EACCES}` passes through `classifyOpenError` unchanged and without the sentinel on any platform; `nil` → `nil`; unrelated sentinels unchanged.

No probabilistic-only coverage remains for the classifier: the integration test (`TestLiveEditAutoSyncReachesExplore`) is now defense-in-depth, not the sole guard.

## Skipped Issues

None.

## Verification

All gates run in the fix worktree after the final commit:

- `go build ./...` — green
- `go vet ./...` — green
- `go test ./...` — green (full module)
- `go test ./testdata/golden/...` — green
- `go test ./test/integration/... -count=1` — green (includes `TestLiveEditAutoSyncReachesExplore` riding the new sentinel path end-to-end)
- `go test -race -count=1 -p 1 ./internal/daemon/... ./internal/watch/... ./internal/cli/...` — green
- `go test ./internal/graphstore/...` (incl. archtest) — green; new lock tests also run under `-race -count=1` — green

**Cross-GOOS verification (CR-01 guardrail):**

- `GOOS=windows GOARCH=amd64 go vet ./internal/graphstore/` — **green** (type-checks `locked_windows.go` and `locked_windows_test.go`)
- `GOOS=windows GOARCH=amd64 go build ./internal/graphstore/` — **green**
- `GOOS=windows GOARCH=arm64 go build ./internal/graphstore/` — **green** (both shipped windows targets)
- `GOOS=linux GOARCH=amd64 go build ./internal/graphstore/` — **green**
- `GOOS=windows|linux GOARCH=amd64 go build ./...` (full module) — **not achievable on this host, pre-existing and unrelated to this fix**: the CGo tree-sitter grammar bindings (`internal/indexer/routes` et al.) are excluded by build constraints when cross-compiling without a per-target C toolchain; the failure is byte-identical for linux and windows and exists before this change. The graphstore package (the entirety of this fix's cross-platform surface) cross-compiles and vets clean for the full release matrix, per the guardrail's documented fallback.

Zero new dependencies; verbatim TS strings untouched; `internal/agents/` untouched; D-04a pebble confinement (archtest) green.

---

_Fixed: 2026-07-16T18:55:00Z_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 2_
