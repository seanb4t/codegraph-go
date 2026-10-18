# Proposal: Keep tests out of the real home directory

Issue: #73

## Why

Running the `internal/cli` tests rewrites the developer's real `~/.claude.json`. Afterwards the codegraph MCP entry points at a `go-build…/cli.test` binary, which is deleted when `go test` exits, so Claude Code can no longer start the server. The cause is `TestUpgradeCommand_NoArgsDefaultsToLatest`. It fakes a successful binary swap, but it does not stub the skill refresh that runs after a swap, and it does not fake `HOME`. The refresh then installs into the real home with `os.Executable()`, which inside a test is the test binary. Nothing in `internal/cli` or `internal/agents` stops a test that forgets `fakeHome` from doing the same.

## What Changes

- `TestUpgradeCommand_NoArgsDefaultsToLatest` stubs `refreshInstalledSkillsFunc` and calls `fakeHome(t)`, like the other `upgrade` tests that reach a successful swap.
- `internal/cli` and `internal/agents` each get a `TestMain`. Before any test runs, it points `HOME`, `XDG_CONFIG_HOME` and `HERMES_HOME` at a temporary directory, and it removes that directory after the run. A test that forgets `fakeHome` then writes to a throwaway home, not the real one.
- In each package, `TestMain` and `fakeHome` take the list of environment variables from one place, so the two cannot drift apart.
- No product code changes. No CLI output, MCP result or installed file changes.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

None. This change alters only test code, so it sets `skip_specs: true`.

## Out of scope

- A product guard that refuses to record a Go test binary as the install `ExecPath`. Issue #73 lists it as optional, and it would be a behavior change with its own spec.
- Home isolation for other test packages. No other test package uses `internal/agents`. `internal/corpora` already sets `HOME` and `XDG_CACHE_HOME` in the tests that read them.
- Repairing a machine that has already been hit. The fix is `codegraph install --target claude --location global`, run from the real binary, as issue #73 says.

## Impact

- Test code only: `internal/cli/upgrade_test.go`, `internal/cli/install_test.go` (`fakeHome`), `internal/agents/testhelpers_test.go` (`fakeHome`), and a new `main_test.go` in each of the two packages.
- No changelog fragment. The PR body carries a `changelog-exempt` marker.
