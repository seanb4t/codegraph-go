# Phase 7: Interactive TUI — Daemon Picker & Install Multi-Select - Pattern Map

**Mapped:** 2026-07-18
**Files analyzed:** 12 (9 new, 3 modified)
**Analogs found:** 12 / 12

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| `internal/daemon/registry.go` | model/service | CRUD (file-per-record) | `internal/daemon/lock.go` (`acquire`/`release`/`isStale`) + `internal/fsatomic/fsatomic.go` (`WriteFile`) | exact (self-heal discipline) + exact (write primitive) |
| `internal/daemon/registry_test.go` | test | CRUD | `internal/githooks/githooks_test.go` (tempdir fixture + assert-bytes style) | role-match |
| `internal/daemon/watchdog.go` | service | event-driven (poll→cancel) | `internal/daemon/daemon.go`'s `Run` goroutine/`wg.Wait()` join discipline | exact (join discipline) |
| `internal/daemon/watchdog_posix.go` | service | event-driven | `internal/daemon/procstart_linux.go` (build-tag POSIX half) | exact (build-tag split precedent) |
| `internal/daemon/watchdog_windows.go` | service | event-driven | `internal/daemon/procstart_other.go` (build-tag non-Linux half) | role-match (POSIX/Windows split, not Linux/other) |
| `internal/daemon/watchdog_test.go` | test | event-driven | `internal/daemon` package tests (goleak `TestMain` gate) | role-match |
| `internal/daemon/stop_posix.go` | service | request-response (signal) | `lock.go`'s `isProcessLive` (`os.FindProcess`+`Signal`) | exact (same call shape, real signal instead of Signal(0)) |
| `internal/daemon/stop_windows.go` | service | request-response | `internal/graphstore/locked_windows.go` (documented Windows syscall divergence) | role-match |
| `internal/cli/daemon.go` (MODIFY) | controller/route | request-response (cobra tree) | `internal/cli/githooks.go` (`newGithooksCmd` + 3 subcommands via `AddCommand`) | exact |
| `internal/cli/tui/daemonpicker.go` | component (bubbletea Model) | streaming (event loop) | `internal/cli/present/progress.go` (charm-in-cli, TTY-gated, goroutine-joined) — style precedent only; no existing bubbletea Model | role-match (no exact analog — first bubbletea usage) |
| `internal/cli/tui/agentpicker.go` | component (bubbletea Model) | streaming (event loop) | `internal/cli/install.go`'s `promptAgentMultiSelect`/`selectByIndices` (selection semantics to preserve) | role-match (input-mechanism swap only) |
| `internal/cli/install.go` (MODIFY) | controller | request-response | itself (pre-existing `RunE`/`installStdinIsInteractive` switch) | exact (in-place edit) |
| `internal/cli/uninstall.go` (MODIFY) | controller | request-response | `internal/cli/install.go` (shares `printAgentResults`/`selectByIndices`) | exact |
| `internal/cli/present/archtest/import_graph_test.go` (VERIFY, no edit expected) | test | — | itself | exact (already covers `internal/daemon` + `charm.land/bubbles/v2`) |
| `internal/githooks/githooks_test.go` (ADD test) | test | file-I/O | `TestRemove_WithUserContent_PreservesRemainderBytes` (same file) | exact |
| `test/integration/piped_never_hang_test.go` | test | request-response (subprocess) | `test/integration/watch_default_test.go` + `main_test.go`'s `runBinary`/`TestMain` harness | exact |

## Pattern Assignments

### `internal/daemon/registry.go` (model, CRUD/file-per-record)

**Analogs:** `internal/daemon/lock.go`, `internal/fsatomic/fsatomic.go`

**Self-heal predicate — reuse verbatim, same package, unexported call** (`internal/daemon/lock.go` lines 82-92):
```go
func isStale(info lockInfo) bool {
	if !isProcessLive(info.PID) {
		return true
	}
	if actualStart, ok := processStartTime(info.PID); ok {
		if !startTimesCorroborate(info.StartedAt, actualStart) {
			return true
		}
	}
	return false
}
```
`registry.go`'s `List()` must call `isStale(lockInfo{PID: rec.PID, StartedAt: rec.StartedAt})` directly — do not reimplement liveness. `lockInfo` (lines 41-44) is the shape to construct inline; no export needed since same package.

**Liveness primitive** (`lock.go` lines 115-134): `isProcessLive(pid int) bool` via `os.FindProcess` + `proc.Signal(syscall.Signal(0))`. Registry's `List()`/`stop --all` target resolution calls this the same way.

**Atomic write — exact signature to call** (`internal/fsatomic/fsatomic.go` line 32):
```go
func WriteFile(path, content string) error
```
Registry's `Register(rec Record) error` marshals JSON to a string and calls `fsatomic.WriteFile(path, string(data))` — do not hand-roll a temp-file+rename.

**Detect-and-clear-stale-on-every-call discipline** — mirror `acquire`'s comment/structure (`lock.go` lines 136-186): every independent read (`List()`) prunes dead records in place; no background reaper (D-05).

**Register/deregister as a defer, exactly like `release()`** (`lock.go` lines 229-238):
```go
func release(codegraphDir string) error {
	err := os.Remove(lockPath(codegraphDir))
	if err != nil && os.IsNotExist(err) {
		return nil
	}
	return err
}
```
`registry.Deregister(pid)` should follow this exact "remove, treat IsNotExist as success" shape, wired into `daemon.Run` via `defer` alongside the existing lock `release()` call.

---

### `internal/daemon/watchdog.go` / `watchdog_posix.go` / `watchdog_windows.go` (service, event-driven)

**Analog:** `internal/daemon/procstart_linux.go` / `procstart_other.go` (build-tag split shape) + `daemon.Run`'s goroutine-join discipline.

**Build-tag header pattern to copy** (`procstart_linux.go` lines 1-3 and `procstart_other.go` lines 1-3):
```go
//go:build linux

package daemon
```
```go
//go:build !linux

package daemon
```
For the watchdog, the split is POSIX vs Windows, not Linux vs other — use `//go:build !windows` / `//go:build windows` instead (RESEARCH's own Code Examples confirm this exact substitution):
```go
// watchdog_posix.go
//go:build !windows

package daemon

func parentChanged(original int) bool {
	return os.Getppid() != original
}
```

**Goroutine join discipline to mirror exactly** (`internal/daemon/daemon.go` lines 287-304, the watch-loop's own join):
```go
loopExited := make(chan struct{})
var wg sync.WaitGroup
wg.Add(1)
go func() {
	defer wg.Done()
	defer close(loopExited)
	w.Run(ctx, deb)
}()

var runErr error
select {
case <-ctx.Done():
case <-loopExited:
	if ctx.Err() == nil {
		runErr = ErrWatcherClosed
	}
}
wg.Wait()
```
`watchdog.Start(ctx, cancel, interval) (stop func())` must return a `stop func()` that blocks on a `done` channel closing (see RESEARCH Code Examples' `startWatchdog` skeleton) — this is the same pattern, adapted: `close(done)` on exit, caller's `stop()` does `<-done`. `daemon.Run` must call this `stop()` on every teardown path (ctx cancel AND the `ErrWatcherClosed` abnormal path) — Common Pitfall 4 in RESEARCH.md is explicit that forgetting this breaks the `goleak`-gated `TestMain` in this package.

---

### `internal/daemon/stop_posix.go` / `stop_windows.go` (service, request-response signal delivery)

**Analog:** `lock.go`'s `isProcessLive` call shape (lines 120-134) — reuse `os.FindProcess` then call `.Signal(...)`, just with `syscall.SIGTERM` instead of `syscall.Signal(0)`:
```go
func sendStop(pid int) error {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return proc.Signal(syscall.SIGTERM)
}
```
Windows precedent for the documented-divergence comment style: `internal/graphstore/locked_windows.go` — a minimal, direct syscall wrapper, no heavyweight library. Windows has no POSIX `SIGTERM` (`os.Process.Signal` only supports `os.Kill` there) — `stop_windows.go` must document this divergence explicitly per RESEARCH Pitfall 3, not silently no-op.

---

### `internal/cli/daemon.go` (MODIFY — controller, cobra command tree)

**Analog:** `internal/cli/githooks.go` — the exact `AddCommand` sub-tree shape to replicate for `daemon`.

**Parent command + AddCommand pattern** (`internal/cli/githooks.go` lines 18-25):
```go
func newGithooksCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "githooks",
		Short: "Manage git sync hooks (post-commit/post-merge/post-checkout)",
	}
	cmd.AddCommand(newGithooksInstallCmd(), newGithooksRemoveCmd(), newGithooksStatusCmd())
	return cmd
}
```
Apply the same shape: `newDaemonCmd()` becomes the bare-RunE (TTY-gated picker/plain-list) parent, with `cmd.AddCommand(newDaemonStartCmd(), newDaemonStopCmd())`. Each subcommand follows the `newGithooksInstallCmd`-style single-purpose `RunE` (lines 28-54) — resolve path/target, call into the domain package, print status lines, never embed business logic in the RunE itself.

**Current foreground RunE to preserve verbatim inside `daemon start`** (`internal/cli/daemon.go` lines 36-81, especially the `watch.DisabledError` friendly-exit branch at lines 73-79):
```go
ctx, stop := signal.NotifyContext(cmd.Context(), syscall.SIGINT, syscall.SIGTERM)
defer stop()

err = d.Run(ctx)
var disabled *watch.DisabledError
if errors.As(err, &disabled) {
	fmt.Fprintf(cmd.ErrOrStderr(), "File watcher disabled — %s. ...\n", disabled.Reason)
	return nil
}
return err
```
This whole block moves into `daemon start`'s `RunE` unchanged (D-01/D-02) — do not rewrite the `errors.As`/`DisabledError` handling.

---

### `internal/cli/tui/daemonpicker.go` / `agentpicker.go` (component, bubbletea Model — first usage, no exact analog)

**Style/TTY-gate precedent:** `internal/cli/present/progress.go` — charm-in-cli confinement, goroutine started/stopped deterministically via channel-close-then-join (lines 62-118). Not a bubbletea Model itself (uses `lipgloss` + `time.Ticker`, not `tea.Program`), but establishes the project's "own goroutine + deterministic Stop()" idiom the picker's TTY-gate wrapper should match.

**TTY-gate call sites to copy the pattern from** (`internal/cli/install.go` lines 24-33, the exact shape `interactiveAllowed` in RESEARCH.md's Pattern 1 generalizes):
```go
var installStdinIsInteractive = func(cmd *cobra.Command) bool {
	if cmd.InOrStdin() != os.Stdin {
		return false
	}
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}
```
Every `tea.NewProgram()` call site (daemon picker's bare RunE, install/uninstall's picker path) MUST gate on both this stdin check AND `term.IsTerminal(stdout fd)` BEFORE constructing the Program — RESEARCH.md's Pattern 1 `interactiveAllowed` combines both; implement it in `internal/cli/tui` or as a small addition beside `present.ChoosePresentation`.

**Selection pipeline to preserve, only the input mechanism swaps** (`internal/cli/install.go` lines 122-185, `promptAgentMultiSelect` + `selectByIndices`): pre-check state comes from `agents.DetectAll(loc)` exactly as today; `agentpicker.go`'s bubbletea Model replaces only the numbered-line prompt's input loop, not `selectByIndices`'s dedup/sort logic or `agents.ResolveTargetFlag`.

**Reporting loop, unchanged** (`internal/cli/install.go` lines 199-239, `printAgentResults`): both install and uninstall keep funneling through this exact function; the picker's output is just a `[]agents.AgentTarget` fed into the same `do`/`statusOf` callback shape already in place.

---

### `internal/cli/install.go` / `uninstall.go` (MODIFY — controller)

**Analog:** themselves — this is a targeted edit, not a rewrite.

**Target-resolution switch to extend** (`internal/cli/install.go` lines 86-96):
```go
var targets []agents.AgentTarget
switch {
case cmd.Flags().Changed("target"):
	targets, err = agents.ResolveTargetFlag(target, loc)
case installStdinIsInteractive(cmd):
	targets, err = promptAgentMultiSelect(cmd, loc)
default:
	targets, err = agents.ResolveTargetFlag("auto", loc)
}
```
Add `-y`/`--yes` as a case checked FIRST (RESEARCH Pitfall 6: `--yes` must short-circuit before the TTY branch, not just skip rendering the picker):
```go
switch {
case yes:
	targets, err = agents.ResolveTargetFlag("auto", loc)
case cmd.Flags().Changed("target"):
	targets, err = agents.ResolveTargetFlag(target, loc)
case tui.InteractiveAllowed(cmd): // replaces installStdinIsInteractive + stdout TTY check
	targets, err = tui.RunAgentPicker(cmd, loc) // replaces promptAgentMultiSelect
default:
	targets, err = agents.ResolveTargetFlag("auto", loc)
}
```
Flag registration pattern to copy (`install.go` lines 108-110): `cmd.Flags().BoolVar(&autoAllow, "auto-allow", false, "...")` — add `-y`/`--yes` the same way with `cmd.Flags().BoolVarP(&yes, "yes", "y", false, "skip the interactive picker; use the non-interactive default set")`.

`uninstall.go` gets the identical treatment — confirm it shares `printAgentResults`/`selectByIndices` from `install.go` (same file, same package) before duplicating any logic.

---

### `internal/cli/present/archtest/import_graph_test.go` (VERIFY — no edit expected)

**Already covers this phase by construction.** `guardedPackages` (lines 39-46) already lists `internal/daemon`; `forbiddenImportPaths` (lines 51-55) already includes `charm.land/bubbles/v2`:
```go
var guardedPackages = []string{
	"github.com/seanb4t/codegraph-go/internal/mcp",
	"github.com/seanb4t/codegraph-go/internal/graphstore",
	"github.com/seanb4t/codegraph-go/internal/daemon",
	"github.com/seanb4t/codegraph-go/internal/watch",
	"github.com/seanb4t/codegraph-go/internal/indexer",
	"github.com/seanb4t/codegraph-go/internal/query",
}

var forbiddenImportPaths = []string{
	"charm.land/lipgloss/v2",
	"charm.land/bubbletea/v2",
	"charm.land/bubbles/v2",
}
```
`excludedInternalPackagePrefixes` (lines 73-75) already excludes `internal/cli` — the new `internal/cli/tui` package is automatically outside the guarded closure since it's a subpackage of `internal/cli`. No changes needed to this test file; the planner should schedule a task that RUNS it after adding the bubbletea/bubbles deps to confirm it stays green, not one that edits it.

---

### `internal/githooks/githooks_test.go` (ADD `TestInstall_EditThenRemove_ByteInvariant`)

**Analog:** `TestRemove_WithUserContent_PreservesRemainderBytes` (same file, lines 381-411) — the exact fixture/assertion shape to extend with an explicit `Install()` call first:
```go
func TestRemove_WithUserContent_PreservesRemainderBytes(t *testing.T) {
	root := initRepo(t, filepath.Join(t.TempDir(), "repo"))
	hooksDir := filepath.Join(root, ".git", "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	withUser := "#!/bin/sh\necho hi\n\n" + markerBlock() + "\n"
	if err := os.WriteFile(filepath.Join(hooksDir, "post-commit"), []byte(withUser), 0o755); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	result := Remove(context.Background(), root)
	// ... assert result.Removed contains "post-commit"
	got, err := os.ReadFile(filepath.Join(hooksDir, "post-commit"))
	want := "#!/bin/sh\necho hi\n"
	if string(got) != want {
		t.Fatalf("post-commit after remove = %q, want %q", string(got), want)
	}
}
```
D-16's new test should: write a pristine original hook file → call `Install(ctx, root)` → simulate a user edit *outside* the marker block (append/prepend real content, mirroring `withUser` above but as a post-install edit) → call `Remove(ctx, root)` → assert `os.ReadFile` returns **byte-identical to the pre-install original**, not just "user content present." Use `initRepo` (existing test helper in this file) for the fixture root.

---

### `test/integration/piped_never_hang_test.go` (NEW — subprocess piped never-hang)

**Analog:** `test/integration/watch_default_test.go` (bounded-context subprocess convention) + `test/integration/main_test.go`'s harness primitives.

**Bounded-context convention to copy** (`watch_default_test.go` lines 33-37):
```go
func TestDefaultWatchHandshakePrompt(t *testing.T) {
	_, main := buildWorktreeFixture(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	...
}
```

**Real-binary subprocess spawn with piped stdio — exact signature to call** (`test/integration/main_test.go` lines 165-183):
```go
func runBinary(t *testing.T, dir string, env []string, args ...string) (stdout, stderr string, err error)
```
This already sets `cmd.Stdout`/`cmd.Stderr` to `bytes.Buffer`s (piped, not a TTY) and calls `cmd.Run()` synchronously — exactly the "closed/piped stdin+stdout" shape D-17 needs. For the never-hang assertion, wrap the call in a goroutine + `select` against `ctx.Done()` (or `time.After`) so a hang fails the test instead of blocking `go test` itself, e.g.:
```go
done := make(chan struct{})
var stdout, stderr string
var runErr error
go func() {
	stdout, stderr, runErr = runBinary(t, dir, nil, "daemon")
	close(done)
}()
select {
case <-done:
	// assert exit 0, plain-list stdout, etc.
case <-time.After(10 * time.Second):
	t.Fatal("codegraph daemon (piped stdio) did not exit — likely blocked on tea.NewProgram()")
}
```
`binPath` is package-level, built once by `TestMain` (`main_test.go` line 39-47) — no new build step needed; this test lives in the same `package integration` and reuses that binary.

## Shared Patterns

### Self-heal-on-read (no background reaper)
**Source:** `internal/daemon/lock.go`'s `acquire()` (lines 151-186) and `isStale()` (lines 82-92)
**Apply to:** `internal/daemon/registry.go`'s `List()` — same "detect+clear stale on every independent call" discipline, generalized from one lockfile to a directory of record files.

### Goroutine join discipline (`stop func()` blocks until done)
**Source:** `internal/daemon/daemon.go`'s `Run` (lines 287-304, `wg.Wait()`) and `internal/cli/present/progress.go`'s `Stop()` (lines 105-118, `close(stopCh); <-doneCh`)
**Apply to:** `internal/daemon/watchdog.go`'s `Start`/`stop` — every new goroutine in `internal/daemon` must be joinable before `Run` returns, or the package's `goleak`-gated `TestMain` fails intermittently.

### Atomic file writes
**Source:** `internal/fsatomic/fsatomic.go`'s `WriteFile(path, content string) error`
**Apply to:** `internal/daemon/registry.go`'s `Register` — no new temp-file+rename implementation.

### TTY-gate before any blocking interactive call
**Source:** `internal/cli/install.go`'s `installStdinIsInteractive` (lines 24-33) generalized in RESEARCH.md's Pattern 1 to also check `term.IsTerminal(stdout fd)`
**Apply to:** every `tea.NewProgram()` call site (`internal/cli/tui/daemonpicker.go`, `agentpicker.go`, and their RunE callers in `daemon.go`/`install.go`/`uninstall.go`) — gate BEFORE construction, never rely on `Program.Run()`'s own error return.

### cobra command-tree shape (`AddCommand` over a bare parent RunE)
**Source:** `internal/cli/githooks.go`'s `newGithooksCmd` (lines 18-25)
**Apply to:** `internal/cli/daemon.go`'s restructuring into `daemon` (bare) + `daemon start` + `daemon stop [--all]`.

### Charm-free domain layer, ANSI-only-in-cli
**Source:** `internal/cli/present/archtest/import_graph_test.go`'s `guardedPackages`/`forbiddenImportPaths` (lines 39-55), already listing `internal/daemon` and `charm.land/bubbles/v2`
**Apply to:** `internal/daemon/registry.go`, `watchdog*.go`, `stop_*.go` — must never import `charm.land/...`; all rendering happens exclusively in `internal/cli/tui`.

## No Analog Found

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| `internal/cli/tui/daemonpicker.go` | component | streaming | No existing bubbletea `Model`/`Update`/`View` usage anywhere in the codebase — this is the first. Use RESEARCH.md's "Daemon picker Model skeleton" and Pattern 2's checkbox `ItemDelegate` example as the primary reference instead of a codebase analog; `present/progress.go` only supplies the surrounding TTY-gate/goroutine-join conventions, not the Model shape itself. |
| `internal/cli/tui/agentpicker.go` | component | streaming | Same reason — no existing `list.Model`/custom `ItemDelegate` in the codebase. RESEARCH.md's Pattern 2 code example (`checkboxDelegate`) is the closest available reference. |

## Metadata

**Analog search scope:** `internal/daemon/`, `internal/cli/`, `internal/cli/present/`, `internal/cli/present/archtest/`, `internal/fsatomic/`, `internal/githooks/`, `test/integration/`
**Files scanned:** `lock.go`, `daemon.go`, `procstart_linux.go`, `procstart_other.go`, `daemon.go` (cli), `githooks.go` (cli), `install.go`, `progress.go`, `tty.go`, `import_graph_test.go`, `fsatomic.go`, `githooks.go`/`githooks_test.go` (internal/githooks), `watch_default_test.go`, `main_test.go`
**Pattern extraction date:** 2026-07-18
