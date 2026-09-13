# Phase 8: tmux Real-PTY Harness - Research

**Researched:** 2026-09-10
**Domain:** tmux-driven real-PTY end-to-end testing of a bubbletea TUI, from Go, via `os/exec`
**Confidence:** HIGH for the tmux driving mechanics and the two highest-risk assertion classes
(all empirically reproduced on this machine, tmux 3.7c); MEDIUM for CI-specific values (exact
`ubuntu-latest` tmux version) that could not be observed from this machine; HIGH for the
build-tag/`go test -json` mechanics (also empirically reproduced).

## Summary

Every mechanical question in this phase was answered by actually running tmux and the real
(mutated and unmutated) `codegraph` binary on this machine — tmux 3.7c, go1.26.6 (see the
toolchain landmine below), darwin/arm64 — rather than by reasoning from tmux's man page alone.
Two results are load-bearing and revise what CONTEXT.md assumed:

1. **D-06 family (a)'s stated mutation target (`internal/cli/daemon.go:82` alone) does NOT
   reproduce the G-07-1 leak.** `internal/cli/tui/daemonpicker.go`'s `RunDaemonPicker` carries
   its own independent, defense-in-depth empty-registry guard (line 318) that is *not* mentioned
   in D-06's mutation table. Mutating only `daemon.go:82` was verified, empirically, to still
   print exactly `no running daemons` with no leak — because `RunDaemonPicker` catches the empty
   case before ever constructing a `tea.Program`, regardless of what its caller does. Reaching
   the historical bug requires mutating **both** guards. This is a correction the planner needs
   before writing task steps, not a planning-time detail.
2. **D-13's content assumption about `capture-pane -a` is backwards.** Empirically, while the
   alt-screen picker is active, plain `capture-pane -p` (no `-a`) shows the alt-screen's
   rendered content (`Running daemons`, the seeded record), and `capture-pane -a -p` shows the
   *frozen main buffer underneath* (blank/shell prompt) — exit 0 either way while active. Only
   after quitting does `-a` flip to exit 1 ("no alternate screen"). D-13's exit-code mechanism
   for "is the alt-screen active" is correct and directly confirmed; its claim that `-a`'s own
   captured *content* contains `Running daemons` is not. The tmux format variable
   `#{alternate_on}` (`1`/`0`) is a cleaner, more direct binary signal than parsing `-a`'s exit
   status and was verified to track the same transition.

The G-07-1 DECRQM leak (the milestone's single highest-risk assertion, TTY-03) was reproduced
end-to-end: a real mutated binary, in a real tmux pane, on an empty registry, leaked
`^[[?2026;2$y^[[?2027;0$y` into `capture-pane -p -e -C -S -` output, appearing a short but
nonzero interval *after* `no running daemons` printed — directly validating both TTY-02's
stability-poll requirement (a single immediate capture would sometimes miss it) and D-12's
capture-instrument choice (`-e -C` renders the leaked ESC bytes as literal, matchable text; no
escalation to `pipe-pane` is needed).

**Primary recommendation:** Build the harness exactly as D-07/D-08/D-09 specify, but (1) update
family (a)'s mutation to touch both `internal/cli/daemon.go:82` and
`internal/cli/tui/daemonpicker.go:318`, (2) get TTY-04's "contains Running daemons" content from
a plain (non-`-a`) capture taken while `#{alternate_on}`/`-a`'s exit code confirms alt-mode is
still active — never from `-a`'s own captured bytes, and (3) seed TTY-04's daemon record by
actually running `codegraph daemon start` as a real background process against a throwaway
`$HOME`+repo (verified to work end-to-end, including clean SIGTERM teardown), not by
hand-writing the registry JSON (which would require reimplementing an OS-specific, dead process
liveness/clock-corroboration check that is unexported and platform-gated).

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Spawn/drive/capture a real terminal session | Test harness (`test/tmux/`, new) | OS (`tmux` binary via `os/exec`) | The harness never re-implements a terminal emulator; it shells out to the one real tmux binary already on `PATH` (D-07/D-08 out-of-scope: no Go tmux client library) |
| Binary-under-test provenance | Test harness `TestMain` (`test/tmux/`) | — | D-09: its own resolver, duplicating (not sharing) `test/integration/main_test.go`'s no-silent-fallback contract |
| TUI rendering / alt-screen / escape hygiene | Product code (`internal/cli`, `internal/cli/tui`) | — | Unchanged in this phase; the harness only *observes* it. The four mutation families temporarily touch it and are reverted byte-clean |
| Daemon registry seeding for TTY-04 | Test harness, via the **real** `daemon start` subprocess | `internal/daemon` (registry/liveness logic, unmodified) | Least-coupled: reuses the actual production registration path instead of reimplementing `isStale`'s liveness+clock corroboration in test/tmux |
| CI execution-count proof | `Taskfile.yml` `test:tmux` target (shell + jq over `go test -json`) | GitHub Actions `tmux-e2e` job (thin `task` caller only) | Matches this repo's existing `TestWorkflowRunBodiesInvokeTask` single-definition rule: the job's `run:` body must be the single literal line `task test:tmux` |

## User Constraints

<user_constraints>
### Locked Decisions (verbatim from 08-CONTEXT.md `## Implementation Decisions`)

- **D-01:** CI proves execution by parsing `go test -json` from the Task target and counting
  test-granularity `pass` events — never by an in-test mechanism. This is the only option that
  survives a misspelled build tag: a mistyped `-tags` value compiles zero test files and
  `go test` exits 0, and any env-var or counter mechanism living inside the test code cannot
  fire because no test code runs. — Reversibility: reversible.
- **D-02:** The assertion is executed count equals a committed constant, not a greater-than-zero
  floor. A floor is satisfied by one case running while the rest skip, which is the same vacuity
  shape one level up. Adding, deleting, or silently skipping a case fails the job until the
  constant is updated in the same commit. — Reversibility: reversible.
- **D-03:** One `task test:tmux` target, not two. It always prints both an executed and a
  skipped count. It enforces the expected-count equality only when `CI` is set, so a contributor
  without tmux gets a clean skip report and exit 0, while CI fails hard.
- **D-04:** The tmux-absent path reports a skip with a reason and a skipped-case count — never a
  silent pass. Follows the repo's existing `t.Skipf`-with-reason convention.
- **D-05:** Assertions are proven to fire by the Phase 7 mutation-log shape, not by
  fault-injection in product code and not by a checked-in bad-pane fixture. For each family: a
  pre-mutation cleanliness gate (`git diff --quiet -- <file>`), the mutation applied, the binary
  rebuilt, the pasted verbatim failing output, the revert, and a byte-clean proof.
- **D-06:** Four families, covering TTY-03 through TTY-06 — not only the two the ROADMAP success
  criteria name.

  | Family | Req | File | Mutation |
  |---|---|---|---|
  | (a) | TTY-03 | `internal/cli/daemon.go:82` | drop `&& len(records) > 0` from the picker guard |
  | (b) | TTY-04 | `internal/cli/tui/daemonpicker.go:241` | `v.AltScreen = false` |
  | (c) | TTY-05 | `internal/cli/tui/agentpicker.go` cancel path | make cancel write config |
  | (d) | TTY-06 | `internal/cli/tui/agentpicker.go:147` | `v.AltScreen = false` (inline render ⇒ flicker) |

  Log file: `.planning/phases/08-tmux-real-pty-harness/08-MUTATION-LOG.md`, following
  `07-MUTATION-LOG.md`'s shape.

  **RESEARCH CORRECTION to family (a):** empirically verified insufficient on its own — see
  Common Pitfalls, "Family (a)'s single-location mutation does not reach the leak."

- **D-07:** The harness lives at `test/tmux/`, a top-level sibling of `test/integration/` and
  `test/wireoracle/`. This overrides `.planning/research/ARCHITECTURE.md` §5. — Reversibility:
  reversible.
- **D-08:** The build tag is `tmux` (`//go:build tmux`). This is the repo's first feature build
  tag. — Reversibility: costly (a later rename touches every file in the package, the Taskfile
  target, and the CI job together).
- **D-09:** `test/tmux` writes its own `TestMain` binary resolver following
  `test/integration/main_test.go`'s contract — honor an env override, abort by name when the
  override is invalid, otherwise `go build` locally — rather than extracting a shared package.
  The no-silent-fallback rule is mandatory and must be stated in the new resolver's own doc
  comment.
- **D-10:** The CI job runs on `ubuntu-latest` — a documented, publicly-inspectable image.
- **D-11:** TTY-07's "pinned tmux version" is satisfied by install-then-assert: install tmux from
  the distro, then fail loudly when `tmux -V` does not equal a committed expected string. A hard
  apt version pin was rejected as brittle. REQUIREMENTS.md TTY-07 was amended in this session to
  match.
- **D-12:** Escape-hygiene assertions capture with `capture-pane -p -e -C -S -`. All three flags
  are load-bearing, verified against the tmux 3.7c man page (and now against a live captured
  leak, see Common Pitfalls / Code Examples). `-e` includes escape sequences (default strips
  them); `-C` renders non-printable bytes as escaped text, which is what makes a leaked ESC byte
  matchable *as text*; `-S -` starts at the beginning of history. Without `-e -C` the TTY-03
  assertion cannot fire at all. If the mutation produces no observable difference under this
  capture, escalate to `pipe-pane` and record why in the mutation log — **not needed**: the leak
  was captured cleanly under `-e -C -S -` in this research session (see Code Examples).
- **D-13:** Alt-screen entry and exit are asserted by `capture-pane -a` exit status, a two-sided
  positive probe. `-a` targets the alternate screen and errors when none exists. During the
  picker it must exit 0 and contain `Running daemons`; after quit it must exit non-zero. Under
  mutation (b) it is already non-zero during the picker.

  **RESEARCH CORRECTION:** the exit-code half is verified correct; the "and contain `Running
  daemons`" half is not — that content comes from a plain (non-`-a`) capture. See Common
  Pitfalls.

- **D-14:** The stability poll is a bounded poll whose non-convergence is a FAILURE, never a skip
  and never a whole-case retry. Capture on a short interval until two consecutive captures are
  byte-identical, up to a deadline; blowing the deadline fails the test naming the last two
  differing captures. TTY-02's "no fixed sleeps in the assertion path" forbids an *unconditional*
  sleep-then-assert; a bounded poll with a convergence condition satisfies it.
- **D-15:** Captures are compared raw, with no normalization — no masked regions, no `-J`.
  Default capture already trims trailing whitespace. Any normalization added later must justify
  what it is allowed to hide.

### Claude's Discretion (verbatim)

- Poll interval and deadline values; TTY-06's `N` and how it is reported.
- The expected-count constant's home (Taskfile variable vs a committed file the suite reads).
- tmux session/window naming, pane geometry, and teardown; helper and test function names.
- Whether TTY-05's config-tree hash walks the tree in Go or shells out.
- How the daemon registry is seeded for TTY-04's "seeded record" — `daemon.Register` vs writing
  the registry file directly. **Research recommendation: neither — use the real `daemon start`
  subprocess. See Code Examples and Common Pitfalls.**
- The `jq` precondition message on the Task target (precedent: `Taskfile.yml:3512`).
- Commit granularity and exact error wording.

### Deferred Ideas (OUT OF SCOPE, verbatim)

- Extracting a shared binary-resolution package for `test/integration` and `test/tmux` (D-09
  chose duplication).
- Running the tmux suite on macOS as well as Linux (D-10 chose Linux only).
- A `pipe-pane` raw-byte-stream capture path as a general instrument, if D-12's `capture-pane -e
  -C` proves insufficient — **empirically it is sufficient; this stays deferred.**
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| TTY-01 | Build-tagged harness builds the binary, spawns in tmux, sends keys, captures the pane; skip-with-reason when tmux absent | Verified `tmux`-tag mechanics (Q8, empirical); `t.Skipf` precedent already in repo (`internal/mcp/markdown_test.go:116`, `internal/githooks/githooks_test.go:31`) |
| TTY-02 | Poll until two consecutive captures are byte-identical; no fixed sleeps | Directly demonstrated by the DECRQM-leak timing race captured in this session — the leak appears after a delay, which is the literal case D-14's poll exists to catch |
| TTY-03 | Bare `daemon` on empty registry: only `no running daemons`, zero DECRQM bytes | Reproduced RED with a corrected two-file mutation (see Common Pitfalls); reproduced the exact leaked bytes under D-12's capture flags |
| TTY-04 | Daemon picker enters alt-screen, shows `Running daemons` + seeded record, restores main buffer on quit | Reproduced GREEN with a real seeded daemon (via real `daemon start` subprocess) and RED via family (b) mutation; corrected D-13's capture mechanics |
| TTY-05 | Checkbox picker `[x]`/`[ ]`, space toggles, q/esc cancel with zero writes (hashed) | Verified `$HOME`-tree hash technique end-to-end (empty→non-vacuous on a real write); enumerated per-agent write paths |
| TTY-06 | Flicker proxy: frame stability across N captures on an idle picker | Mechanically identical poll-and-compare primitive as TTY-02/D-14; family (d) mutation targets the same `AltScreen` field pattern verified for family (b) |
| TTY-07 | CI installs tmux, asserts version, runs suite, asserts executed-count constant | `go test -json` event shape confirmed (Q4, empirical); `ubuntu-latest`'s exact tmux version could not be observed from this machine — flagged as an open item to bootstrap on first real CI run |
</phase_requirements>

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|---------------|
| `tmux` | 3.7c installed locally `[VERIFIED: tmux -V on dev machine]`; exact `ubuntu-latest` version unknown, see Open Questions | Real PTY driver | Only tool that provides a real terminal emulator answering DECRQM/DA queries the way a real terminal does; this repo's own G-07-1/G-07-2 incidents are invisible to piped tests by construction |
| `os/exec` (stdlib) | Go 1.26.6 stdlib | Shell out to `tmux` | D-07/D-08's own out-of-scope line: "A Go tmux client library" is explicitly rejected in REQUIREMENTS.md's Out of Scope table — no viable one exists; matches this repo's existing git/brew interop style |
| `testing` (stdlib) + build tag `tmux` | Go 1.26.6 stdlib | Gate the harness | `//go:build tmux`, confirmed empirically to compile to zero matched packages under `go list ./...`/`go vet ./...`/`go test ./...` with no tag, and to compile and run cleanly with `-tags tmux` |

No new third-party dependency is introduced by this phase — everything above is either already
present (`tmux` on `PATH`, checked at harness start) or stdlib.

### Package Legitimacy Audit

**Not applicable.** This phase introduces no new `go.mod` dependency (confirmed: `os/exec` +
stdlib `testing` only, per REQUIREMENTS.md's own Out-of-Scope line rejecting a Go tmux client
library). No package-legitimacy check is required.

## Architecture Patterns

### System Architecture Diagram

```
 go test -tags tmux ./test/tmux/...           (invoked by `task test:tmux`)
        |
        v
 TestMain (test/tmux, D-09)
   - resolveTestBinPath(): CODEGRAPH_TEST_BIN override, or `go build` locally
   - tmux on PATH? --------------------------- no --> every Test* calls t.Skipf(reason);
   |                                                    skipped-case count printed; exit 0
   yes
   |
   v
 per-test: acquireSession(t) (unique name, -d, -x/-y fixed geometry)
   |
   +--> [TTY-01/03] send-keys -l "<env HOME=... BIN daemon>" ; send-keys Enter
   |        |
   |        v
   |    pollUntilStable(capture-pane -p -e -C -S -)   <-- D-14 bounded poll, D-12 flags
   |        |
   |        v
   |    assert: exact "no running daemons" text; assert: no DECRQM byte sequence anywhere
   |            in the stabilized capture (TTY-03) -- must poll past the leak's arrival delay
   |
   +--> [TTY-04] spawn REAL `codegraph daemon start --path <repo> --quiet` as background
   |        subprocess against the SAME throwaway $HOME (own os/exec.Cmd, Start() not Run())
   |        |
   |        v
   |    poll for ~/.codegraph/daemons/<pid>.json to appear (registration confirmed)
   |        |
   |        v
   |    send-keys the bare `daemon` picker command into the tmux pane
   |        |
   |        v
   |    assert #{alternate_on}==1 (or capture-pane -a exit==0) WHILE ALSO asserting
   |    plain capture-pane -p contains "Running daemons" + the seeded record line
   |        |
   |        v
   |    send-keys q ; assert #{alternate_on}==0 (or capture-pane -a exit!=0)
   |    assert stabilized plain capture shows no residual escapes in scrollback
   |        |
   |        v
   |    defer: SIGTERM the background daemon subprocess, Wait() (graceful Deregister)
   |
   +--> [TTY-05] hash throwaway $HOME tree (find -type f | sha256sum | sort | sha256sum)
   |        send-keys the `install` picker; send-keys Space (toggle); send-keys q/Escape (cancel)
   |        re-hash $HOME tree; assert equal
   |
   +--> [TTY-06] send-keys the picker command; loop N capture-pane -p -e -C -S - calls at a
   |        short interval with NO input in between; assert all N byte-identical; report N
   |
   v
 kill-session (tolerate "no server running", already-gone is not a test failure)
```

### Recommended Project Structure

```
test/tmux/
├── main_test.go       # D-09: own TestMain, own resolveTestBinPath (duplicated, not shared)
├── session.go          # tmux session lifecycle: new/kill, send-keys wrappers, geometry consts
├── capture.go          # capture-pane wrappers + D-14 pollUntilStable; D-12 flag constants
├── daemon_seed.go      # real `daemon start` subprocess seeding helper for TTY-04
├── tty01_skip_test.go  # TTY-01: tmux-absent skip path (can be forced by PATH manipulation)
├── tty02_03_daemon_empty_test.go   # TTY-02 (stability poll) + TTY-03 (escape hygiene), same fixture
├── tty04_daemon_picker_test.go     # TTY-04 (alt-screen + seeded record)
├── tty05_install_hash_test.go      # TTY-05 (checkbox + config-tree hash)
└── tty06_flicker_test.go           # TTY-06 (N-capture stability proxy)
```

Flat, no `t.Run` subtests recommended (see Common Pitfalls — "subtests double-count against
D-02's exact-equality constant unless the counting convention is chosen deliberately"). Keeping
`test/tmux` to one `Test*` function per assertion class keeps the `go test -json` count
trivially unambiguous.

### Pattern 1: tmux session lifecycle from Go (verified argv)

**What:** Every `tmux` argv this harness needs, run and confirmed on tmux 3.7c on this machine.
**When to use:** Every test in `test/tmux`.
**Example:**

```go
// Source: verified by direct execution on this machine (tmux 3.7c), not transcribed from memory.

// 1. Create a detached session with a known pane size. Unique name per test avoids
//    collisions if tests ever run with t.Parallel() (not recommended for this package,
//    but the name should still be collision-safe: e.g. fmt.Sprintf("cgtmux-%d-%d", os.Getpid(), n)).
exec.Command("tmux", "new-session", "-d", "-s", sessionName, "-x", "100", "-y", "30").Run()

// 2. Run the binary: two-step send-keys (literal command line, then a separate Enter) —
//    verified reliable; a single send-keys call with the command AND Enter is also possible
//    but two calls is what was tested here.
exec.Command("tmux", "send-keys", "-t", sessionName, "-l",
    fmt.Sprintf("env HOME=%s USERPROFILE=%s %s daemon", home, home, binPath)).Run()
exec.Command("tmux", "send-keys", "-t", sessionName, "Enter").Run()

// 3a. Send a literal key/string (e.g. arbitrary text) — use -l:
exec.Command("tmux", "send-keys", "-t", sessionName, "-l", "hello").Run()

// 3b. Send a named key — NO -l, use tmux's own key name. Verified: "Space", "Enter",
//     "Escape" all work as named keys; a single visible character like "q" also works
//     WITHOUT -l (tmux passes an unrecognized bareword through as literal input).
exec.Command("tmux", "send-keys", "-t", sessionName, "Space").Run()
exec.Command("tmux", "send-keys", "-t", sessionName, "q").Run()
exec.Command("tmux", "send-keys", "-t", sessionName, "Escape").Run()

// 4. Capture with D-12's flags:
out, _ := exec.Command("tmux", "capture-pane", "-t", sessionName, "-p", "-e", "-C", "-S", "-").Output()

// 4b. Alt-screen binary signal — EITHER form verified equivalent; #{alternate_on} avoids
//     parsing exit codes and stderr text:
altOn, _ := exec.Command("tmux", "display-message", "-t", sessionName, "-p", "#{alternate_on}").Output()
// altOn == "1\n" while active, "0\n" after quit.
// Equivalently: exec.Command("tmux", "capture-pane", "-t", sessionName, "-a", "-p").Run()
// returns exit 0 while active, exit 1 ("no alternate screen" on stderr) after quit —
// but its STDOUT while active is the frozen main screen, NOT the alt-screen content.
// Get "Running daemons" from the plain (non -a) capture in 4, taken in the same window.

// 5. Teardown — NOT idempotent at the server level: killing an already-gone session/server
//    returns exit 1 ("no server running..."). Tolerate this in cleanup:
exec.Command("tmux", "kill-session", "-t", sessionName).Run() // ignore error
```

### Pattern 2: Seeding a live daemon record for TTY-04 (verified end-to-end)

**What:** `internal/daemon.List()` self-heals: any record whose PID is not a genuinely live
process, OR whose recorded `StartedAt` does not corroborate against that PID's real OS-reported
start time within `procStartTimeSlack` (5s, `internal/daemon/lock.go:108`), is pruned on the
very call that reads it (`internal/daemon/registry.go` `List()`, calling `isStale` from
`internal/daemon/lock.go:82-92`). `isStale`/`processStartTime` are unexported and
platform-gated (`procstart_linux.go` / `procstart_other.go`) — reimplementing this
corroboration logic in a black-box `test/tmux` package to hand-write a registry JSON file is
high-risk and unnecessary duplication.

**When to use:** TTY-04's "seeded record" requirement.

**Example (verified — real background process, real registration, clean teardown):**

```go
// Source: verified by direct execution on this machine.
// 1. `codegraph init` the throwaway repo first — `daemon start` refuses an
//    uninitialized repo ("daemon: not initialized: ... run `codegraph init` first").
initCmd := exec.Command(binPath, "init", repoDir)
initCmd.Env = append(os.Environ(), "HOME="+home, "USERPROFILE="+home)
initCmd.Run()

// 2. Start the REAL daemon in the background. No fork/double-fork: the launched
//    process's own PID IS the registered PID (verified: cmd.Process.Pid matched the
//    PID written into ~/.codegraph/daemons/<pid>.json).
daemonCmd := exec.Command(binPath, "daemon", "start", "--path", repoDir, "--quiet")
daemonCmd.Env = append(os.Environ(), "HOME="+home, "USERPROFILE="+home)
daemonCmd.Start() // NOT Run() — must stay backgrounded for the test's duration
defer func() {
    daemonCmd.Process.Signal(syscall.SIGTERM) // graceful: daemon.go's SIGTERM handler
    daemonCmd.Wait()                          // calls Deregister via defer (daemon.go:245)
}()

// 3. Poll for the registry file rather than a fixed sleep (mirrors D-14's own discipline).
waitForFile(t, filepath.Join(home, ".codegraph", "daemons"), 2*time.Second)

// 4. NOW launch the interactive picker in the tmux pane against the SAME $HOME.
//    It will show "Running daemons" + the seeded record — verified end-to-end.
```

**Pitfall specific to sandboxed/ephemeral shells:** if this recipe is prototyped across
multiple *separate* shell invocations (e.g. one shell backgrounds the daemon with `&`, a later,
different shell invocation tries to use it), the backgrounded process dies when its parent shell
exits — this is an artifact of a multi-invocation harness, not of the technique. A real
`*testing.T` keeps the whole process tree alive for the test's duration, so this does not affect
the actual Go harness; it only affected early prototyping in this research session and is
recorded here so it isn't rediscovered and misdiagnosed as a technique failure.

### Pattern 3: Config-tree hash for TTY-05 (verified non-vacuous)

**What:** `codegraph install`/`uninstall` write under `$HOME` (each `internal/agents/*.go`
target's `*ConfigPath(loc)` resolves under `os.UserHomeDir()` for `LocationGlobal` — the
picker's actual invocation path, since `install`'s `--location` defaults to `global` and the
interactive-picker branch never overrides it). One exception to flag: `cursorConfigPath`'s
`LocationLocal` branch falls back to `os.Getwd()` `[VERIFIED: internal/agents/cursor.go:84]` —
irrelevant to TTY-05 specifically since the picker path uses the default `global` location, but
worth a one-line comment in the test so a future local-location test doesn't reuse this exact
hash recipe unmodified.

**When to use:** TTY-05's "hash the config tree before and after."

**Example (verified — empty tree hash changed after a real write, proving non-vacuity):**

```bash
# Source: verified by direct execution on this machine.
before=$(find "$HOME_THROWAWAY" -type f -exec sha256sum {} + | sort | sha256sum)
# ... run the picker, space-toggle, q/esc cancel ...
after=$(find "$HOME_THROWAWAY" -type f -exec sha256sum {} + | sort | sha256sum)
# before == after required for TTY-05's cancel-path assertion.
# Non-vacuity proof (run once, in research, not part of the shipped test): the SAME recipe
# against a REAL `install --yes` (which does write) produced a DIFFERENT hash — confirmed
# the hash is sensitive to real writes, not a hash of nothing.
```

`find ... | sort | sha256sum` naturally covers newly-created files and directories with no need
to enumerate paths in advance (confirmed: a single target's install created 6 files across 3
new nested directories, all captured by one `find` call). This satisfies Claude's Discretion's
open question ("walks the tree in Go or shells out") — shelling out is simplest and was the
form verified; a pure-Go `filepath.WalkDir` + `sha256` equivalent works identically if the
plan prefers no subprocess for the hash step itself (the harness already shells out to `tmux`
for everything else, so shelling out here is consistent with the rest of the package).

### Pattern 4: `go test -json` event shape and the exact counting filter (D-01/D-02)

**What:** Confirmed shape (`go1.26.6 test -json ./internal/gitmeta/...`, 188 JSON lines):

```json
{"Time":"...","Action":"start","Package":"...gitmeta"}
{"Time":"...","Action":"run","Package":"...gitmeta","Test":"TestCachingDetectorMemoizesPositive"}
{"Time":"...","Action":"output","Package":"...gitmeta","Test":"TestCachingDetectorMemoizesPositive","Output":"=== RUN ...\n"}
{"Time":"...","Action":"pass","Package":"...gitmeta","Test":"TestCachingDetectorMemoizesPositive","Elapsed":0.11}
...
{"Time":"...","Action":"output","Package":"...gitmeta","Test":"TestFixtureVerdicts/linked-worktree","Output":"..."}
{"Time":"...","Action":"pass","Package":"...gitmeta","Test":"TestFixtureVerdicts/linked-worktree","Elapsed":0.1}
... (7 more TestFixtureVerdicts/<name> subtests, each with their own run+pass) ...
{"Time":"...","Action":"pass","Package":"...gitmeta","Test":"TestFixtureVerdicts","Elapsed":0.83}
...
{"Time":"...","Action":"output","Package":"...gitmeta","Output":"PASS\n"}
{"Time":"...","Action":"output","Package":"...gitmeta","Output":"ok  \t...gitmeta\t13.181s\n"}
{"Time":"...","Action":"pass","Package":"...gitmeta","Elapsed":13.329}
```

Two facts are load-bearing for D-02's exact-count assertion:

1. **A parent test with subtests emits BOTH its own `pass` event AND one `pass` event per
   subtest.** `TestFixtureVerdicts` (8 subtests) produced 9 `Action=="pass"` events carrying a
   `Test` field — counting all of them double-counts the parent against its own children.
2. **The final package-level summary is ALSO an `Action=="pass"` event, but with NO `Test`
   field** (only `Package`+`Elapsed`). A naive `jq 'select(.Action=="pass")' | length` count
   includes this line too.

**Recommended jq filter** (counts top-level `Test` functions only, excluding subtests and the
package summary — the simplest convention to reason about and to keep the committed constant
stable):

```bash
go test -tags tmux -json ./test/tmux/... | \
  jq -s '[.[] | select(.Action=="pass" and .Test != null and (.Test | contains("/") | not))] | length'
```

**Recommendation on subtests:** keep `test/tmux` flat (Pattern in Recommended Project Structure
above — one `Test*` function per assertion class, no `t.Run`). This makes the counting
convention above unambiguous by construction and removes any need to special-case subtests in
the committed constant at all. If a future test genuinely needs table-driven subtests, switch
the filter to the leaf-counting form (count `.Test` values that are not a `/`-prefix of any
other `.Test` value in the same run) rather than changing the flat convention ad hoc.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Terminal emulation / DECRQM response behavior | A fake terminal emulator or `script`/`expect`-based simulation | The real `tmux` binary via `os/exec` | Only a real PTY answers capability queries the way a real terminal does — this is the entire premise of the phase (REQUIREMENTS.md Out of Scope explicitly rejects a Go tmux client library) |
| Daemon liveness + clock corroboration for a seeded fixture | Hand-written registry JSON + a reimplementation of `isStale`/`processStartTime`'s platform-gated corroboration | The real `codegraph daemon start` subprocess (Pattern 2) | `isStale` is unexported, platform-gated (`procstart_linux.go`/`procstart_other.go`), and corroborates against a 5s slack window against the OS's actual process-start clock — reimplementing this in test/tmux for a fixture is both harder and less authentic than just running the real code path |
| Config-tree diffing for TTY-05 | A per-agent enumeration of every `DescribePaths()` call across all 9 registered targets, kept in sync by hand | A single `find $HOME -type f \| sort \| sha256sum` (or Go `filepath.WalkDir` equivalent) over the whole throwaway `$HOME` | Every agent target writes under `$HOME` for the default (`global`) location the picker uses; a whole-tree hash is invariant to which specific files change and needs no per-target maintenance as new agent targets are added |

**Key insight:** every "don't hand-roll" in this phase is the same shape — reuse the real,
already-correct mechanism (real tmux, real daemon binary, real filesystem walk) instead of
building a parallel, necessarily-incomplete model of it in the test harness.

## Common Pitfalls

### Pitfall 1: Family (a)'s single-location mutation does not reach the leak

**What goes wrong:** Dropping `&& len(records) > 0` from `internal/cli/daemon.go:82` alone,
rebuilding, and running bare `daemon` on an empty registry still prints exactly
`no running daemons` with **no leak** — verified empirically in this research session.

**Why it happens:** `internal/cli/tui/daemonpicker.go`'s `RunDaemonPicker` (line ~318) carries
its own, independent empty-registry short-circuit with an explicit doc comment calling it
"Defense-in-depth ... so this function can never be the source of that leak regardless of
caller." That second guard was not named in D-06's mutation table and silently absorbs the
caller-side mutation.

**How to avoid:** Mutate **both** locations for family (a)'s RED demonstration:
`internal/cli/daemon.go:82` (`if interactiveAllowed(cmd) {`) AND
`internal/cli/tui/daemonpicker.go:318` (`if len(records) < 0 {` — using `< 0`, since `len()` is
never negative, rather than deleting the guard body, keeps the mutation single-token-ish and the
diff minimal). Verified: with both mutations applied, the exact historical leak
(`^[[?2026;2$y^[[?2027;0$y`) reappears in `capture-pane -p -e -C -S -` output.

**Warning signs:** A RED demonstration for family (a) that "passes" (i.e., still shows only
`no running daemons`, no leak) after mutating only `daemon.go:82` — that is not a fixed bug, it
is an unmutated second guard.

### Pitfall 2: `capture-pane -a`'s content is not what D-13 assumes

**What goes wrong:** Asserting `capture-pane -a -p`'s stdout contains `Running daemons` while
the picker is active will fail even when TTY-04's real behavior is completely correct, because
`-a`'s own captured content while active is the **frozen main screen** (blank/shell prompt),
not the alt-screen's rendered content.

**Why it happens:** Empirically, tmux's `-a` flag and plain (no-flag) `capture-pane -p` target
different buffers while alt-screen mode is active: plain capture-pane already shows "whatever is
currently displayed" (which, correctly, is the alt-screen content when alt-screen is active);
`-a` appears to always target the buffer that would be the underlying/main one. The man page's
wording ("if -a is given, the alternate screen is used, and the history is not accessible") is
easy to misread as "the alt-screen's rendered content", but only its *exit status* semantics
(0 while an alternate screen exists for the pane, 1/"no alternate screen" once it's gone) were
confirmed reliable.

**How to avoid:** Use TWO separate signals: (1) `#{alternate_on}` (via
`tmux display-message -p "#{alternate_on}"`, verified `1` while active / `0` after quit) or
equivalently `capture-pane -a`'s exit code, purely as the binary "is alt-screen active right
now" signal; (2) a plain `capture-pane -p -e -C -S -` (D-12's flags), taken in the same
poll-stabilized window, for the actual "contains `Running daemons`" content assertion.

**Warning signs:** A TTY-04 test that fails on a correct build because it tried to `strings.
Contains` on `-a`'s own stdout.

### Pitfall 3: The `go1.27.1` vs `go.mod`'s pinned `1.26.6` toolchain breaks ANY local `go build`

**What goes wrong:** `go build ./cmd/codegraph` (and therefore `test/integration`'s existing
`TestMain`, and the new `test/tmux` `TestMain` D-09 specifies) fails on this machine with
`undefined: hashFn` / `undefined: getRuntimeHasher` / `undefined: fastrand64` from
`github.com/cockroachdb/swiss` when built with the locally-installed go1.27.1, because
`go.mod` pins `go 1.26.6` with **no explicit `toolchain` directive**, so `GOTOOLCHAIN=auto`
does not auto-download 1.26.6 (it only downloads when the installed toolchain is *older* than
required, not newer) — the newer local 1.27.1 is used instead, and `cockroachdb/swiss`'s
`go:linkname`-based runtime hooks are incompatible with it.

**This is a pre-existing, already-known repo issue, not something Phase 8 introduces** — it is
explicitly documented in-repo: `.github/workflows/ci.yml:260-272`'s `govulncheck` job comment
records the identical failure from a real CI run (34232338047) when `go-version-input` was left
non-empty and resolved to `stable` instead of `go.mod`'s pin.

**How to avoid (for this phase specifically):** Document in the harness's own `TestMain` doc
comment (D-09 already requires a doc comment for the no-silent-fallback contract) that a local
`go build` requires a toolchain resolving to exactly `go.mod`'s pinned version — install it with
`go install golang.org/dl/go1.26.6@latest && go1.26.6 download` and invoke tests via that
binary, or set up the environment so `go` itself resolves to 1.26.6. **This affects contributors
running `task test:tmux` locally on any machine whose default `go` is newer than 1.26.6** — it
is not new to this phase (the identical exposure already exists for `test/integration` and
`test/wireoracle`'s existing `TestMain`s) but is worth a one-line note in `test/tmux/main_test.go`
since this phase is a new opportunity for a contributor to hit it fresh.

**Warning signs:** A `go build`/`go test` failure mentioning `cockroachdb/swiss`,
`hashFn`, `fastrand64`, or `getRuntimeHasher` — this is a toolchain-version problem, not a
tmux, bubbletea, or test-logic problem. Do not debase into `internal/cli`/`internal/cli/tui`
looking for a real bug.

### Pitfall 4: A build-tag-only package is invisible to `./...`, silently or loudly depending on the invocation shape

**What goes wrong / how it actually behaves (verified):**

- `go build ./...`, `go vet ./...`, `go list ./...` (repo-wide wildcard, matching hundreds of
  other packages): a directory containing ONLY `//go:build tmux`-gated files is **silently
  excluded** — a `go: warning: "./path/..." matched no packages` on stderr, but **exit 0** for
  the overall command, because other real packages still match. This is safe: existing
  `task test:unit`/`vet`/`build ./...` targets will never fail or need updating because of the
  new `test/tmux` package.
- A pattern that **exclusively** targets the tagged directory (e.g. a hypothetical
  `go vet ./test/tmux/...` with no tag) produces the SAME warning but **exit 1**
  ("no packages to vet") — a hard, loud failure, not vacuous.
- A **misspelled** `-tags` value (e.g. `tmuxx`) behaves identically to no tag at all: if the
  invocation is package-scoped (`go test -tags tmuxx ./test/tmux/...`), it also fails loudly
  (exit 1, "no packages to test") *if every file in the package carries the tag* (D-08 implies
  this, including `TestMain`) — so this specific narrow-invocation failure mode is already
  safe, not vacuous, contrary to a literal reading of D-01's own justification text.

**Why D-01's `go test -json` approach is still exactly right regardless:** the exact-count check
is uniform insurance against *every* zero-execution scenario, not just the one where the command
exits 0. It does not need to know or care whether zero tests ran because of a hard package-match
failure (exit 1, already loud) or because a package matched but had literally 0 `Test*` functions
(the classic `ok ... [no test files]`, exit 0, silent) — both are equally caught by asserting the
`pass`-count equals a committed constant. Do not weaken the plan to "just check the exit code" on
the theory that a misspelled tag already fails loudly in the narrow-invocation case — that theory
is only true as long as EVERY file in `test/tmux/` (including `TestMain`) carries the tag; if a
future edit leaves one shared helper untagged, the failure mode silently flips to the vacuous
"ok, no test files" shape, and only the JSON count check still catches it.

**Warning signs:** None specific — this is a "verify, don't assume" note for the plan, not a
runtime symptom.

### Pitfall 5: Subtests double-count against D-02's exact-equality constant

**What goes wrong:** If `test/tmux` uses `t.Run` subtests anywhere, a naive
`jq 'select(.Action=="pass")' | length` count includes the parent's own `pass` event *and* each
subtest's `pass` event, so adding one subtest silently changes the count by 2 (subtest + updated
parent elapsed-time event — no, the parent's pass event isn't duplicated per subtest, but the
count still includes both the parent and every child), making the committed constant fragile and
hard to reason about by inspection.

**How to avoid:** Keep `test/tmux` flat (see Recommended Project Structure) — no `t.Run`. If a
future need forces subtests, switch to leaf-counting (only `.Test` values that are not a
`/`-prefixed parent of any other `.Test` value) rather than top-level-only counting.

## Code Examples

### The DECRQM leak, captured live (verified end-to-end, this session)

RED demonstration recipe that reproduces TTY-03's target defect exactly, using D-12's capture
flags, against a binary built from a two-file mutation (see Pitfall 1):

```
$ tmux capture-pane -t <session> -p -e -C -S -
no running daemons
^[[?2026;2$y^[[?2027;0$y\033[1;7m%\033[0m
sean@denver codegraph-go %
```

The leaked bytes (`^[[?2026;2$y^[[?2027;0$y`) are tmux's terminal emulator answering the
bubbletea Program's synchronized-output (mode 2026) and grapheme-clustering (mode 2027) DECRQM
capability probes; because the Program already quit (per `daemonPickerModel.View()`'s immediate
`no running daemons` render + implicit quit when nothing to pick, once BOTH guards are defeated),
nothing consumes the terminal's replies, and they surface on the next reader — the shell's own
line editor, which echoes unrecognized control bytes visibly. **This appeared a short but
nonzero interval after `no running daemons` printed**, not in the same capture — directly
motivating D-14's bounded stability poll (a single immediate capture, taken right after seeing
"no running daemons", can and did miss this in early manual testing before the poll pattern was
applied).

### Alt-screen signal, both forms verified

```
$ tmux display-message -t <session> -p "#{alternate_on}"
1                                    # picker active
... (send-keys q; sleep) ...
$ tmux display-message -t <session> -p "#{alternate_on}"
0                                    # after quit

$ tmux capture-pane -t <session> -a -p; echo "exit=$?"
exit=0                               # picker active (content is the frozen main screen, NOT "Running daemons")
... (after quit) ...
$ tmux capture-pane -t <session> -a -p; echo "exit=$?"
no alternate screen
exit=1
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|---------------|--------|
| Piped/TTY-blind integration suite (`test/integration/`) as the deepest automated rung | Real-PTY tmux harness as an additional, deeper rung, gated behind a build tag | This phase | `test/integration` is unchanged and stays the fast, always-run default; `test/tmux` is the new, deliberately-optional-locally / mandatory-in-CI deeper check |

**Deprecated/outdated:** Nothing in this repo is deprecated by this phase; it is purely
additive test infrastructure.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | The exact `tmux -V` string `ubuntu-latest` will report at CI execution time | Open Questions / D-11 | D-11's committed expected-version constant would need updating on the very first CI run; D-11 already anticipates this ("determined empirically at execution time — not guessed from the dev machine's version") |
| A2 | `go test -json`'s event shape (Action/Test/Package/Elapsed field names, pass/run/output/start actions) is stable across the Go 1.26.x line used by both dev machines and CI | Code Examples / Pattern 4 | Low risk — this is a long-stable, documented `cmd/go` JSON contract; verified directly against go1.26.6 in this session |
| A3 | No GitHub-hosted `ubuntu-latest` runner ships tmux preinstalled (ROADMAP.md's own Notes paragraph: "could not be confirmed and is treated as absent-by-default") | D-10/D-11 install-then-assert design | If tmux IS actually preinstalled, `apt-get install tmux` is a harmless no-op/upgrade either way — D-11's install-then-assert design is robust to this either way |

**If this table is empty:** N/A — three items above need eventual confirmation from a real CI
run, but none blocks writing the plan; D-11's own design already tolerates A1/A3.

## Open Questions

1. **Exact `tmux -V` string on the `ubuntu-latest` GitHub-hosted runner at CI execution time**
   - What we know: tmux 3.7c is installed on this dev machine (darwin/arm64, Homebrew); the
     `ubuntu-latest` label was, per a same-day web search, in the middle of a transition from
     Ubuntu 24.04 toward a newer Ubuntu 26.04 image (`actions/runner-images` release
     `ubuntu26/20260810.99`) around the time of this research (2026-09) — meaning the apt-default
     tmux version on `ubuntu-latest` is itself a moving target right now, not a stable value to
     hardcode.
   - What's unclear: the literal string `tmux -V` will print on whatever `ubuntu-latest` resolves
     to at the moment the CI job first runs.
   - Recommendation: exactly what D-11 already specifies — install tmux, run `tmux -V` once for
     real in the first CI execution, and commit whatever it actually printed as the expected
     constant in the same PR (a deliberate bootstrap step), rather than guessing a version number
     now. Do not pre-populate this constant from the dev machine's 3.7c or from a web search guess.

2. **Whether GitHub's `ubuntu-latest` image ships tmux preinstalled at all**
   - What we know: ROADMAP.md's own Notes paragraph already flags this as unconfirmed and
     "treated as absent-by-default." This research's web search could not confirm a software
     manifest entry for tmux on the current image either.
   - What's unclear: whether `apt-get install tmux` in the CI job will be a fresh install or a
     no-op/upgrade.
   - Recommendation: D-10/D-11's install-then-assert design already tolerates either case — no
     plan change needed; the CI job installs unconditionally rather than branching on presence.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|--------------|-----------|---------|----------|
| `tmux` | Entire harness (TTY-01…07) | Yes, on this dev machine | 3.7c `[VERIFIED: tmux -V]` | D-04's `t.Skipf` path when absent — already the designed fallback, not a gap |
| `jq` | `task test:tmux`'s CI-mode exact-count assertion (mirrors `check:linux-cross-exec`'s existing precondition) | Present on this dev machine; standard on `ubuntu-latest` | — | Taskfile `preconditions: - sh: command -v jq` (precedent: `Taskfile.yml:3511-3512`) — hard-fail the task with a clear message if absent, matching the existing pattern exactly |
| Go toolchain resolving to `go.mod`'s pinned `1.26.6` | `test/tmux`'s `TestMain` local `go build` fallback path (D-09), and equally `test/integration`'s existing one | Not the default on this dev machine (system `go` is 1.27.1) — fixed for this session via `go install golang.org/dl/go1.26.6@latest && go1.26.6 download` | 1.26.6 pinned in `go.mod` `[VERIFIED: go.mod:3]` | `CODEGRAPH_TEST_BIN` env override (D-09's own designed fallback) bypasses the local-build path entirely — use it on any machine where the ambient `go` doesn't resolve to 1.26.6 |

**Missing dependencies with no fallback:** None — every dependency above already has a
documented, in-repo fallback (skip-with-reason for tmux, a Taskfile precondition for jq, an env
override for the toolchain/build issue).

**Missing dependencies with fallback:** See table above; none block writing or executing the plan.

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Go `testing` stdlib, build-tag-gated (`//go:build tmux`) |
| Config file | none — a new Taskfile target (`test:tmux`) is the config surface, following `test:integration`/`test:wireoracle`'s precedent of no separate config file |
| Quick run command | `task test:tmux` (local, lenient — prints skip count, exits 0 without tmux) |
| Full suite command | `CI=1 task test:tmux` (strict — asserts executed-count equals the committed constant) |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|---------------------|--------------|
| TTY-01 | tmux-absent skip-with-reason | integration (real PTY, build-tagged) | `task test:tmux` (run with `PATH` stripped of tmux to observe the skip path) | ❌ Wave 0 (new package) |
| TTY-02 | Stability poll, no fixed sleep | integration | Same fixture as TTY-03 (the DECRQM-arrival delay IS the race case) | ❌ Wave 0 |
| TTY-03 | Empty-registry escape hygiene | integration | `go test -tags tmux -run TestDaemonEmptyRegistry ./test/tmux/...` (name illustrative) | ❌ Wave 0 |
| TTY-04 | Daemon picker alt-screen + seeded record | integration | `go test -tags tmux -run TestDaemonPicker ./test/tmux/...` | ❌ Wave 0 |
| TTY-05 | Checkbox picker zero-write-on-cancel | integration | `go test -tags tmux -run TestInstallPickerCancel ./test/tmux/...` | ❌ Wave 0 |
| TTY-06 | Flicker proxy (N-capture stability) | integration | `go test -tags tmux -run TestPickerFrameStability ./test/tmux/...` | ❌ Wave 0 |
| TTY-07 | CI executed-count + tmux-version gate | CI job (`tmux-e2e` in `ci.yml`, thin `task test:tmux` caller) | `CI=1 task test:tmux` | ❌ Wave 0 (new Taskfile target + CI job + `inScopeJobs` fixture entry) |

### Sampling Rate

- **Per task commit:** `task test:tmux` (lenient locally)
- **Per wave merge:** `CI=1 task test:tmux` if tmux is available locally, else defer to CI
- **Phase gate:** the new `tmux-e2e` CI job green, with the executed-count assertion actually
  exercised (not skipped) — confirm by inspecting the job's own log output before calling the
  phase done

### Wave 0 Gaps

- [ ] `test/tmux/main_test.go` — TestMain + `resolveTestBinPath` (D-09)
- [ ] `test/tmux/session.go`, `capture.go` — tmux argv wrappers (Pattern 1), D-14's poll helper
- [ ] `test/tmux/daemon_seed.go` — real-subprocess seeding helper (Pattern 2)
- [ ] `Taskfile.yml` `test:tmux` target — D-01/D-02/D-03's jq-based exact-count gate, following
      the `check:linux-cross-exec` positive-count-gate shape (`Taskfile.yml:3548-3559`) but with
      `-ne "${EXPECTED}"` instead of `-eq 0`
- [ ] `.github/workflows/ci.yml` `tmux-e2e` job — `runs-on: ubuntu-latest`, install tmux, assert
      `tmux -V`, single `run:` step whose body is the literal line `task test:tmux` (verified:
      `taskCallLineRe = ^task\s+[A-Za-z0-9:_-]+$` in `internal/upgrade/taskfile_shape_test.go:204`
      forbids anything else on that line — env vars like `CI=1` must be set via the job/step
      `env:` YAML key, never inlined into the run body)
- [ ] `internal/upgrade/taskfile_shape_test.go` `inScopeJobs` — add
      `{Workflow: "ci.yml", JobID: "tmux-e2e"}`, or `TestInScopeJobsPopulationMatchesDisk` will
      fail once the new job exists on disk but isn't in the fixture (verified: this test diffs
      `inScopeJobs` against every job actually present in the in-scope workflow files)
- [ ] `.planning/phases/08-tmux-real-pty-harness/08-MUTATION-LOG.md` — four RED demonstrations,
      family (a) corrected per Pitfall 1

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|----------------|---------|-------------------|
| V2 Authentication | No | Test infrastructure only; no auth surface touched |
| V3 Session Management | No | N/A |
| V4 Access Control | No | N/A |
| V5 Input Validation | No | The harness constructs its own tmux argv from fixed, non-user-controlled strings (test fixture paths, throwaway `$HOME`); no untrusted input reaches `os/exec` |
| V6 Cryptography | No | The config-tree hash (Pattern 3) uses `sha256sum` for change-detection, not a security boundary — collision resistance is not a threat-relevant property here |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|------------------------|
| Command injection via constructed tmux argv | Tampering | All values passed to `os/exec.Command` in this harness are either fixed literals or paths derived from `t.TempDir()`/`os.Executable()` — never raw external/user input. Continue passing argv elements as separate `exec.Command` arguments (never via a shell string) as this repo's existing `os/exec` call sites already do |
| Leaking the developer's real `~/.codegraph`/agent configs during a test run | Info exposure | Every test MUST use a throwaway `$HOME`/`USERPROFILE` (the same convention `test/integration/piped_never_hang_test.go` already uses) — verified in this research session's own scratch testing that omitting this would touch the real registry |
| A stray backgrounded daemon subprocess (Pattern 2) surviving a failed/panicking test | Denial of service (resource leak, not a real vuln) | `defer` the SIGTERM+`Wait()` teardown immediately after `Start()` succeeds, before any assertion that could `t.Fatal` |

This phase is test infrastructure with no product-facing attack surface; the table above is
included for completeness per the security-enforcement default, not because a material threat
was found.

## Sources

### Primary (HIGH confidence — empirically verified this session)

- Direct execution of `tmux` 3.7c on this machine (session lifecycle, `send-keys`,
  `capture-pane` with and without `-a`/`-e`/`-C`/`-S`, `display-message -p "#{alternate_on}"`,
  `kill-session` idempotency) — Q1, Q2, Q3
- Direct execution of `go1.26.6 test -json` against `internal/gitmeta` — Q4, event shape and
  subtest/package-summary double-counting behavior
- Direct execution of a scratch build-tagged package under `go build`/`go vet`/`go list`/
  `go test`, with and without `-tags`, with a correct and a misspelled tag value — Q8
- Direct execution of the real `codegraph init` / `daemon start` / `daemon` (bare, picker) /
  `install` commands against throwaway `$HOME`s, including two temporary, git-tracked, and
  fully reverted source mutations (`internal/cli/daemon.go`, `internal/cli/tui/daemonpicker.go`)
  used only to reproduce the historical G-07-1 leak and the G-07-2 alt-screen regression for
  research purposes — Q2, Q3, Q5, Q6
- `Read` of `internal/cli/daemon.go`, `internal/cli/tui/daemonpicker.go`,
  `internal/cli/tui/agentpicker.go`, `internal/cli/tui/tty.go`, `internal/cli/install.go`,
  `internal/daemon/{registry,lock,daemon}.go`, `internal/agents/{registry,claude,cursor}.go`,
  `test/integration/{main_test.go,piped_never_hang_test.go}`, `Taskfile.yml` (lines 117-210,
  3475-3563), `internal/upgrade/taskfile_shape_test.go` (lines 152-220, 1400-1510, 1728+),
  `.github/workflows/ci.yml` (lines 1-60, 238-312) — Q5, Q6, Q7, Q8, Q9

### Secondary (MEDIUM confidence)

- WebSearch, "ubuntu-latest github actions runner image tmux preinstalled apt version 2026" —
  confirmed `ubuntu-latest` is mid-transition (24.04 → 26.04, `actions/runner-images`
  `ubuntu26/20260810.99`) as of this research date; did not find a tmux-specific manifest entry
  — Open Question 1/2

### Tertiary (LOW confidence)

- None used as a basis for any claim in this document; every mechanical claim above was either
  empirically verified this session or is a direct `Read` quote from the codebase.

## Metadata

**Confidence breakdown:**

- Standard stack: HIGH — no new dependency; tmux presence/version verified directly
- Architecture / tmux mechanics: HIGH — every argv and capture behavior was executed, not
  recalled, including the two corrections to CONTEXT.md's D-06/D-13 assumptions
- Pitfalls: HIGH — both the family-(a) and `-a`-content corrections are reproducible, pasted
  evidence, not inference
- CI-specific values (exact `tmux -V` on `ubuntu-latest`): MEDIUM — could not be observed from
  this machine; D-11's own bootstrap design already accounts for this

**Research date:** 2026-09-10
**Valid until:** Re-verify the `tmux -V` constant and the go.mod toolchain pin at execution time;
otherwise this research does not have a fast-moving expiry (tmux's `capture-pane`/`send-keys`
surface used here is long-stable stdlib-adjacent CLI behavior).
