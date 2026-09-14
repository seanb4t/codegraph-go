# Phase 8: tmux Real-PTY Harness - Pattern Map

**Mapped:** 2026-09-10
**Files analyzed:** 9 (7 created code artifacts + 3 modified files; `08-MUTATION-LOG.md` is a planning artifact, no code analog needed)
**Analogs found:** 9 / 9

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| `test/tmux/main_test.go` | test (harness bootstrap) | request-response (subprocess build+exec) | `test/integration/main_test.go` | exact |
| `test/tmux/` tmux argv wrappers (session/send-keys/capture-pane) | utility | event-driven (drives a real pty via subprocess) | `internal/gitmeta/*.go` (`exec.CommandContext` argv-construction idiom) + `test/integration/main_test.go`'s `runBinary` | role-match |
| `test/tmux/` bounded stability-poll helper (D-14) | utility | streaming/poll | `test/integration/piped_never_hang_test.go`'s `runPipedNeverHang` (goroutine + `time.After` select) | role-match |
| `test/tmux/` daemon seeding helper | utility (subprocess lifecycle) | event-driven (spawn + SIGTERM teardown) | `test/integration/main_test.go`'s `runBinary` + `internal/cli/daemon.go`'s `newDaemonStartCmd` (target surface) | partial (no existing background-subprocess-with-teardown analog; composed from two) |
| `test/tmux/*_test.go` (TTY-01..TTY-06 test files) | test | request-response / streaming | `test/integration/piped_never_hang_test.go` | role-match |
| `Taskfile.yml` `test:tmux` target | config | CRUD (none)/batch | `Taskfile.yml` `check:linux-cross-exec` (exact-count/non-zero gate shape) + `test:integration`/`test:wireoracle` (target body shape) | exact (gate) / exact (target shape) |
| `.github/workflows/ci.yml` `tmux-e2e` job | config | request-response (thin task caller) | `.github/workflows/ci.yml` `transcript-freeze` job (job-level `env:` + single `run: task <target>`) | exact |
| `internal/upgrade/taskfile_shape_test.go` `inScopeJobs` entry | test fixture | CRUD (literal slice append) | `internal/upgrade/taskfile_shape_test.go`'s own `inScopeJobs` slice | exact |
| `08-MUTATION-LOG.md` | planning artifact | n/a | `.planning/phases/07-guards-that-cannot-fire/07-MUTATION-LOG.md` | exact (not code, skipped per instructions) |

## Pattern Assignments

### `test/tmux/main_test.go` (test, request-response)

**Analog:** `test/integration/main_test.go` (full file read, 307 lines)

**Package doc pattern** (lines 1-13) — restate the GOLDEN-01 "why not testdata/" contract and the explicit-CI-step rationale, adapted for `test/tmux` and the `tmux` build tag:
```go
// Package integration is TEST-04's subprocess integration harness (D-17):
// a normal Go package (never testdata/ — GOLDEN-01 cost Phase 2 a Critical
// when a suite silently didn't run) so `go test ./...` reaches it, PLUS an
// explicit named CI step (.github/workflows/ci.yml) so a future refactor of
// the filtered `go list ./...` line can never silently drop it either.
package integration
```
For `test/tmux`, the doc comment must additionally state: (a) the `//go:build tmux` tag means `go list ./...` reaches the package but `go vet`/`go build` skip it without `-tags tmux` — so the CI step is not just defense-in-depth but the *only* way it runs at all; (b) D-08's "first feature build tag in this repo" framing.

**`testBinEnvVar` / override contract** (lines 28-40):
```go
var binPath string

const testBinEnvVar = "CODEGRAPH_TEST_BIN"
```
D-09 requires `test/tmux` to **duplicate**, not import, the resolver — use a distinct env var name only if warranted (RESEARCH.md doesn't mandate renaming; reusing `CODEGRAPH_TEST_BIN` is fine since it's a value, not a type, and duplicating the same var name across two independent packages is safe).

**`resolveTestBinPath` — the no-silent-fallback contract to restate verbatim in the new package's own doc comment** (lines 42-95, especially the doc comment at 42-70):
```go
// Contract: for a non-empty raw value there are exactly two outcomes —
// (path, true, nil): the override is valid, use it; or ("", false, err):
// the override is invalid, abort by name. There is no third outcome: no
// input returns useEnv=true together with a non-nil error, and no
// non-empty input returns useEnv=false with a nil error. That absence is
// the property that forbids a silent fallback to a local `go build` on a
// bad override...
func resolveTestBinPath(raw string) (path string, useEnv bool, err error) {
	if raw == "" {
		return "", false, nil
	}
	info, statErr := os.Stat(raw)
	if statErr != nil {
		return "", false, fmt.Errorf("%s=%q: %w", testBinEnvVar, raw, statErr)
	}
	if info.IsDir() {
		return "", false, fmt.Errorf("%s=%q: not a regular file (is a directory)", testBinEnvVar, raw)
	}
	if !info.Mode().IsRegular() {
		return "", false, fmt.Errorf("%s=%q: not a regular file", testBinEnvVar, raw)
	}
	if info.Mode().Perm()&0o111 == 0 {
		return "", false, fmt.Errorf("%s=%q: not executable (no execute permission bit set)", testBinEnvVar, raw)
	}
	abs, absErr := filepath.Abs(raw)
	if absErr != nil {
		return "", false, fmt.Errorf("%s=%q: resolve absolute path: %w", testBinEnvVar, raw, absErr)
	}
	return abs, true, nil
}
```
D-09's own required addition: state explicitly in the doc comment (not just inherit by proximity) that this is the mandatory no-silent-fallback rule, since `test/tmux` is a structurally separate copy that a future reader will not automatically associate with `test/integration`'s reasoning.

**`TestMain` build-or-use-override pattern** (lines 96-144):
```go
func TestMain(m *testing.M) {
	resolved, useEnv, err := resolveTestBinPath(os.Getenv(testBinEnvVar))
	if err != nil {
		fmt.Fprintln(os.Stderr, "integration: TestMain:", err)
		os.Exit(1)
	}
	if useEnv {
		binPath = resolved
		os.Exit(m.Run())
	}
	tmpDir, err := os.MkdirTemp("", "codegraph-integration-*")
	...
	binPath = filepath.Join(tmpDir, "codegraph")
	buildCmd := exec.Command("go", "build", "-o", binPath, "github.com/seanb4t/codegraph-go/cmd/codegraph")
	if out, err := buildCmd.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "integration: TestMain: go build ... failed: %v\n%s\n", err, out)
		_ = os.RemoveAll(tmpDir)
		os.Exit(1)
	}
	code := m.Run()
	_ = os.RemoveAll(tmpDir)
	os.Exit(code)
}
```
For `test/tmux`, rename the tmp-dir prefix (e.g. `codegraph-tmux-*`) and the log prefix (`"tmux: TestMain:"`) to keep failure output package-identifiable — this is the only substantive change D-09 implies beyond the build tag itself. The RED demonstrations (D-05/D-06) need this same build step to pick up mutated source with no extra plumbing, since it always rebuilds from the working tree unless overridden.

---

### `test/tmux/` tmux argv wrappers (utility, event-driven)

**No direct analog exists in this repo for tmux argv specifically** — this is genuinely new surface. The closest transferable idiom is the `exec.CommandContext` + typed-error-wrap pattern from `internal/gitmeta/worktree.go`:
```go
cmd := exec.CommandContext(ctx, "git", "rev-parse", "--show-toplevel")
```
(`internal/gitmeta/worktree.go:38`, `:64`; `internal/gitmeta/permalink.go:72`, `:259`; `internal/gitmeta/githooks.go:19`, `:41` — all six call sites share the shape: one `exec.CommandContext(ctx, "<tool>", args...)` per external-tool operation, output captured and parsed inline, error wrapped with context naming the operation).

Combine with `runBinary` from `test/integration/main_test.go:252-270` for the general "spawn a subprocess, capture stdout/stderr into buffers, return them plus err" shape:
```go
func runBinary(t *testing.T, dir string, env []string, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	cmd := exec.Command(binPath, args...)
	cmd.Dir = dir
	if env != nil {
		cmd.Env = append(os.Environ(), env...)
	}
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err = cmd.Run()
	return outBuf.String(), errBuf.String(), err
}
```
For the tmux wrappers, each argv-builder (`tmux new-session`, `tmux send-keys`, `tmux capture-pane -p -e -C -S -`, `tmux kill-session`) should follow this exact shape: one small function per tmux subcommand, `exec.Command("tmux", args...)`, `CombinedOutput()` or split stdout/stderr per D-12/D-13's needs, and an error that names the tmux subcommand and its args on failure (mirroring gitmeta's `fmt.Errorf("...: %w", err)` idiom — grep gitmeta's non-test files for the exact wrap strings if the planner wants a literal template).

D-12's exact capture invocation to hard-code:
```
tmux capture-pane -p -e -C -S - -t <target>
```
D-13's alt-screen probe:
```
tmux capture-pane -a -p -t <target>   # exit 0 while alt-screen active, non-zero after quit
```
(also acceptable per the CORRECTED note: query `#{alternate_on}` via `tmux display-message -p -t <target> '#{alternate_on}'`.)

---

### `test/tmux/` bounded stability-poll helper (utility, streaming/poll)

**Analog:** `test/integration/piped_never_hang_test.go` (full file read, 90 lines), specifically `runPipedNeverHang` (lines 57-77):
```go
func runPipedNeverHang(t *testing.T, dir string, env []string, args ...string) (stdout, stderr string) {
	t.Helper()
	done := make(chan struct{})
	var runErr error
	go func() {
		stdout, stderr, runErr = runBinary(t, dir, env, args...)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatalf("codegraph %v (piped stdio) did not exit within 10s — likely blocked on tea.NewProgram()", args)
	}
	if runErr != nil {
		t.Fatalf("codegraph %v (piped stdio) exited non-zero: %v\nstdout: %s\nstderr: %s", args, runErr, stdout, stderr)
	}
	return stdout, stderr
}
```
D-14's stability poll adapts this shape but as a **loop with a deadline**, not a single race: capture on a short interval, compare successive captures byte-for-byte (D-15: no normalization), succeed once two consecutive captures match, and `t.Fatalf` naming the last two differing captures if the deadline (`time.After` or a `context.WithTimeout`) fires first. The never-hang file's package doc (lines 1-7) is also directly relevant framing: "riding the same subprocess harness substrate ... the bounded context/time.After convention."

---

### `test/tmux/` daemon seeding helper (utility, event-driven subprocess lifecycle)

**No direct existing analog for "spawn as background subprocess, wait for readiness, SIGTERM on teardown."** Compose from:
- `test/integration/main_test.go`'s `runBinary` (lines 252-270) for the `exec.Command(binPath, args...)` + `cmd.Env = append(os.Environ(), env...)` throwaway-`$HOME` convention — but note the seeding helper must NOT use `cmd.Run()` (blocking); it needs `cmd.Start()` then hold the `*exec.Cmd` for later `cmd.Process.Signal(syscall.SIGTERM)` + `cmd.Wait()`.
- `test/integration/piped_never_hang_test.go`'s throwaway-`HOME` convention (lines 26-27):
```go
home := t.TempDir()
env := []string{"HOME=" + home, "USERPROFILE=" + home}
```
- `internal/cli/daemon.go:119` `newDaemonStartCmd` — the target surface being exercised; read for the cobra `Use: "start"` argv shape the seeding helper must invoke via the real binary (`<bin> daemon start`), not an in-process call — this mirrors D-19's whole-harness rationale of driving through the subprocess boundary.
- CONTEXT.md's own settled note: do not hand-write registry JSON — `internal/daemon/lock.go`'s unexported `isStale` liveness/clock logic would need reimplementing. The helper's only job is process lifecycle (start, wait-for-ready via polling `daemon` list or the registry file appearing, SIGTERM, wait for exit), never registry content.

---

### `test/tmux/*_test.go` (TTY-01..TTY-06 test files) (test, request-response/streaming)

**Analog:** `test/integration/piped_never_hang_test.go`'s `TestPipedNeverHang` (lines 24-50) for overall test-function shape — subtests via `t.Run`, throwaway `HOME`/`t.TempDir()` per subtest, assert on captured stdout/stderr:
```go
func TestPipedNeverHang(t *testing.T) {
	t.Run("daemon_bare", func(t *testing.T) {
		home := t.TempDir()
		env := []string{"HOME=" + home, "USERPROFILE=" + home}
		dir := t.TempDir()
		stdout, stderr := runPipedNeverHang(t, dir, env, "daemon")
		if !strings.Contains(stdout, "no running daemons") {
			t.Fatalf(...)
		}
		assertNoInteractiveEscape(t, "daemon", stdout, stderr)
	})
	...
}
```
For `test/tmux`, replace `runPipedNeverHang`'s subprocess call with: create tmux session → seed daemon (or not, for TTY-03's empty-registry case) → `send-keys` the command → capture-pane per D-12/D-13 → assert. The `assertNoInteractiveEscape`-style helper (lines 79-90) is the direct ancestor of TTY-03's "no leaked mode-query bytes" assertion, just inverted: TTY-03 asserts the leak pattern is **present** pre-mutation-revert (RED) and **absent** post-fix (GREEN), using the same `strings.Contains` idiom against the `-e -C` capture text.

Skip-on-tmux-absent convention (D-04) — follow the `t.Skipf`-with-reason shape used for missing `git`:
```go
// internal/mcp/markdown_test.go:116
t.Skipf("git %v failed (git missing or unsupported here): %v: %s", args, err, string(out))
// internal/githooks/githooks_test.go:31
t.Skipf("git %v failed (git missing or fixture unsupported here): %v: %s", args, err, string(out))
```
Apply the identical shape for tmux: `t.Skipf("tmux %v failed (tmux missing or unsupported here): %v: %s", args, err, string(out))`, called from a shared `requireTmux(t)`-style helper at the top of each test (or in `TestMain`, per D-03/D-04's split between per-test skip and package-level skip reporting — Claude's Discretion on exact placement).

---

### `Taskfile.yml` `test:tmux` target (config, batch)

**Analog 1 — exact-count gate shape:** `Taskfile.yml` `check:linux-cross-exec` (verified at the lines shown by direct read; the block runs from the `check:linux-cross-exec:` key through its `cmds:` script). Key transferable elements:
```yaml
  check:linux-cross-exec:
    desc: >-
      ...
    preconditions:
      - sh: '[ "$(go env GOHOSTOS)" = "linux" ]'
        msg: "..."
      - sh: command -v jq
        msg: "jq not found — required to parse codegraph status --json's numeric fields."
    cmds:
      - |
        set -euo pipefail
        ...
        FILES=$(printf '%s' "${STATUS_JSON}" | jq -r '.fileCount')
        if ! [[ "${FILES}" =~ ^[0-9]+$ ]] || [ "${FILES}" -eq 0 ]; then
          echo "::error::codegraph status --json reports fileCount=${FILES} — REL-05 requires a non-zero count, not a green exit code"
          exit 1
        fi
```
D-01/D-02's target inverts the comparison to an **exact-equality** check (`-ne "${EXPECTED}"` rather than `-eq 0`), gated on `CI` per D-03: parse `go test -json ./test/tmux/... -tags tmux` output through `jq`, count `"Action":"pass"` events at test granularity, and fail with an `::error::` line when `CI` is set and the count is not exactly the committed constant; when `CI` is unset, print both counts and exit 0 regardless (the lenient local path).

**Analog 2 — target/desc shape for a package-scoped `test:*` target:** `Taskfile.yml:159-178`, `test:integration` and `test:wireoracle`:
```yaml
  test:integration:
    desc: >-
      Subprocess integration harness (test/integration) — already reached by
      go list ./... (it's a normal package, not testdata/), but this explicit
      target guards against a future refactor of test:unit's filtered go list
      line silently dropping it too.
    cmds:
      - go test ./test/integration/...

  test:wireoracle:
    desc: >-
      Wire-level regression oracle (test/wireoracle) — spawns the real
      binary over real stdio and byte-compares normalized transcripts
      against testdata/wireoracle/transcripts/*.golden. ...
    cmds:
      - go test ./test/wireoracle/...
```
`test:tmux` follows this `desc:` + `cmds:` shape but must additionally pass `-tags tmux` (the package is invisible to a bare `go test ./test/tmux/...` without it) and route through `-json` for the count-parsing script, per D-01. The `jq` precondition message precedent at `Taskfile.yml:3512` (`msg: "jq not found — required to parse codegraph status --json's numeric fields."`) is the literal template for `test:tmux`'s own `jq`-missing precondition message.

---

### `.github/workflows/ci.yml` `tmux-e2e` job (config, request-response)

**Analog:** `transcript-freeze` job (read at lines 481-517), specifically its job-level `env:` + single-line `run: task <target>` shape — the exact pattern D-03/`taskCallLineRe` requires:
```yaml
  transcript-freeze:
    name: transcript-freeze
    if: github.event_name == 'pull_request'
    runs-on: namespace-profile-linux-amd64-2x4
    steps:
      - name: Checkout
        uses: actions/checkout@df4cb1c069e1874edd31b4311f1884172cec0e10 # v6.0.3
        with:
          fetch-depth: 0
      - name: Set up Go
        uses: actions/setup-go@924ae3a1cded613372ab5595356fb5720e22ba16 # v6.5.0
        with:
          go-version-file: go.mod
          cache: false
      - name: Install Task
        uses: ./.github/actions/install-task
      - name: Anti-regeneration guard (D-03, advisory since 03-02)
        env:
          TRANSCRIPT_FREEZE_BASE: origin/${{ github.event.pull_request.base.ref }}
        run: task check:transcript-freeze
```
The step-level `env:` (never interpolated into `run:` body) is the exact mechanism `tmux-e2e` needs for `CI=1` (D-03's strict-mode trigger), since `taskCallLineRe = ^task\s+[A-Za-z0-9:_-]+$` (`internal/upgrade/taskfile_shape_test.go:204`) forbids anything else in the `run:` line.

**Analog for `runs-on: ubuntu-latest` + Go setup, D-10:** `perf-regression` job (lines 390-410):
```yaml
  perf-regression:
    name: perf regression gate (PERF-02, INDX-06)
    runs-on: ubuntu-latest
    env:
      CODEGRAPH_BENCH_RUNNER: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@df4cb1c069e1874edd31b4311f1884172cec0e10 # v6.0.3
      - name: Set up Go
        uses: actions/setup-go@924ae3a1cded613372ab5595356fb5720e22ba16 # v6.5.0
        with:
          go-version-file: go.mod
          cache: true
      - name: Install Task
        uses: ./.github/actions/install-task
```
`tmux-e2e` follows this same checkout/setup-go/install-task preamble, then adds an "Install tmux" step (`sudo apt-get install -y tmux` on `ubuntu-latest`) and a "Verify tmux version" step asserting `tmux -V` against the committed expected string (D-11), before the final `env: {CI: "1"} / run: task test:tmux` step.

---

### `internal/upgrade/taskfile_shape_test.go` `inScopeJobs` entry (test fixture, CRUD)

**Analog:** the slice itself, `internal/upgrade/taskfile_shape_test.go:152-165`:
```go
var inScopeJobs = []inScopeJob{
	{Workflow: "ci.yml", JobID: "test"},
	{Workflow: "ci.yml", JobID: "actionlint"},
	{Workflow: "ci.yml", JobID: "goreleaser-check"},
	{Workflow: "ci.yml", JobID: "reproducibility"},
	{Workflow: "ci.yml", JobID: "perf-regression"},
	{Workflow: "ci.yml", JobID: "transcript-freeze"},
	{Workflow: "ci.yml", JobID: "tool-vuln"},
	{Workflow: "release-please.yml", JobID: "pretag-gate"},
	{Workflow: "corpora.yml", JobID: "corpora"},
	{Workflow: "corpora.yml", JobID: "golden"},
	{Workflow: "components-drift.yml", JobID: "components-drift"},
}
```
Add exactly one literal entry: `{Workflow: "ci.yml", JobID: "tmux-e2e"}`. `TestInScopeJobsPopulationMatchesDisk` (named in the VALIDATION.md checklist item) will fail once the job exists on disk without a matching entry here — this is a required, not optional, edit per the Wave 0 checklist.

---

## Shared Patterns

### No-silent-fallback binary resolution
**Source:** `test/integration/main_test.go:42-95` (`resolveTestBinPath`)
**Apply to:** `test/tmux/main_test.go`'s own resolver (D-09 — duplicated, not imported; state the contract in the new file's own doc comment, do not rely on the reader finding the sibling package).

### Throwaway-`$HOME` subprocess isolation
**Source:** `test/integration/piped_never_hang_test.go:26-27`, `:39-40`
**Apply to:** every `test/tmux` test that spawns the real binary or the seeded daemon — never touch the developer's real `~/.codegraph`.

### Bounded never-hang / bounded-poll goroutine + `time.After`
**Source:** `test/integration/piped_never_hang_test.go:57-77`
**Apply to:** the stability-poll helper (D-14) and any tmux capture that could otherwise block indefinitely (e.g. waiting on a session to become ready).

### Skip-with-reason for a missing external tool
**Source:** `internal/mcp/markdown_test.go:116`, `internal/githooks/githooks_test.go:31` (both for missing `git`)
**Apply to:** every `test/tmux` test's tmux-absence path (D-04), via `t.Skipf` naming tmux and the underlying error — never a silent pass.

### `exec.CommandContext` argv-construction + wrapped error
**Source:** `internal/gitmeta/worktree.go:38,64`, `internal/gitmeta/permalink.go:72,259`, `internal/gitmeta/githooks.go:19,41`
**Apply to:** every tmux argv wrapper function (session create/teardown, send-keys, capture-pane).

### Non-zero-count-is-the-real-gate (not green-exit)
**Source:** `Taskfile.yml` `check:linux-cross-exec` (`::error::` on zero count, non-zero exit)
**Apply to:** `test:tmux`'s exact-count assertion (D-01/D-02) — same anti-vacuity idiom, tightened from `-eq 0` to `-ne "${EXPECTED}"`.

### Job-level `env:` + single-line `task <target>` run body
**Source:** `.github/workflows/ci.yml` `transcript-freeze` job (`env: TRANSCRIPT_FREEZE_BASE: ...` / `run: task check:transcript-freeze`)
**Apply to:** the `tmux-e2e` job's final step (`env: CI: "1"` / `run: task test:tmux`) — required by `taskCallLineRe`'s exact-match constraint in `internal/upgrade/taskfile_shape_test.go:204`.

## No Analog Found

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| tmux argv wrappers (session/send-keys/capture-pane specifically) | utility | event-driven | No existing code in this repo shells out to tmux; composed from the gitmeta `exec.CommandContext` idiom + `test/integration`'s `runBinary` shape, not a single existing analog |
| daemon background-subprocess seeding helper (start + hold + SIGTERM teardown) | utility | event-driven | No existing helper starts a long-lived background subprocess and tears it down with a signal; `runBinary` is `cmd.Run()` (blocking) only. Composed from `runBinary`'s env/argv shape plus new `cmd.Start()`/`cmd.Process.Signal`/`cmd.Wait()` logic |
| `//go:build tmux` tag itself | n/a | n/a | D-08 confirms this is the repo's first feature build tag — only GOOS gates and one `//go:build ignore` exist; there is no analog for the tag syntax/placement convention to copy, stated plainly rather than stretched for |

## Metadata

**Analog search scope:** `test/integration/`, `test/wireoracle/`, `Taskfile.yml`, `.github/workflows/ci.yml`, `internal/upgrade/taskfile_shape_test.go`, `internal/cli/daemon.go`, `internal/cli/tui/daemonpicker.go`, `internal/cli/tui/agentpicker.go`, `internal/gitmeta/`, `internal/mcp/markdown_test.go`, `internal/githooks/githooks_test.go`
**Files scanned:** ~15 read/grepped directly, all citations verified against actual file contents (no invented line numbers)
**Pattern extraction date:** 2026-09-10
