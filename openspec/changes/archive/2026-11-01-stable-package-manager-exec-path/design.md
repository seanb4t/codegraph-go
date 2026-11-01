# Design

## Context

See proposal.md for why. The relevant code today:

- `newInstallCmd` (`internal/cli/install.go`) calls `os.Executable()` once, puts the result in `agents.InstallOptions.ExecPath`, and calls `opts.Validate()` before the picker and before any target runs. Every target writes `ExecPath` into its MCP entry, and the Claude and Codex guards bake it into their scripts.
- `newUpgradeCmd` (`internal/cli/upgrade.go`) calls `os.Executable()` for the self-replace target. It passes the same path to `refreshInstalledSkills`. That function re-runs the Claude install for every location that has a manifest.
- `ephemeralExecPath` (`internal/agents/types.go`) classifies a path by its shape only. The `agents` package never reads the environment. The CLI reads `CODEGRAPH_ALLOW_EPHEMERAL_EXEC_PATH`.
- On Linux, `os.Executable()` reads `/proc/self/exe`, which the kernel resolves through every link. On macOS it returns the path given to `exec`, which can be a link. The mise shim on macOS runs the binary through its versioned path, so both platforms give a versioned path for a mise install.
- mise keeps `installs/<tool>/latest` as a relative link to the newest installed version, for example `latest -> ./0.16.3`. asdf uses the same `installs/<tool>/<version>` layout but keeps no `latest` link.
- The release ships a Homebrew cask (`homebrew_casks` in `.goreleaser.yaml`), not a formula. A cask puts the binary at `<prefix>/Caskroom/codegraph/<version>/codegraph` and links it from `<prefix>/bin/codegraph`.

## Goals / Non-Goals

**Goals:**

- A mise or Homebrew cask install records a path that keeps working after the next upgrade and after the old version is pruned.
- The recorded path always starts the same binary that ran `install`, at the time of the install.
- A user with a versioned path that the install cannot fix learns about it once, at install time.

**Non-Goals:**

- No change to the self-replace target of `codegraph upgrade`.
- No warning from the upgrade refresh. Its output stays as it is.
- No Homebrew formula (`Cellar`) layout, because the project ships no formula.

## Decisions

### One path function in `internal/agents`

A new `StableExecPath(path string) (recorded string, versioned bool)` sits next to `ephemeralExecPath` in `types.go`. If a stable link qualifies, it returns the link. Otherwise it returns `path` unchanged. `versioned` is true if `path` matches a versioned layout. The function reads the filesystem but not the environment, so it keeps the package's rule. Alternative considered: put the rewrite inside `InstallOptions.Validate` or inside each target. Rejected, because `Validate` only accepts or refuses. A rewrite per target also prints the warning once per agent.

### The layout comes from path elements, not from tool names

For mise and asdf, the function looks for the last `installs` element. Two elements must follow it: `<tool>` and `<version>`. The candidate replaces `<version>` with `latest`. For a cask, the function looks for the last `Caskroom` element, followed by `<token>` and `<version>`. The candidate is `<parent of Caskroom>/bin/<base name>`. A `<version>` of `latest` is not versioned, so a path that is already stable returns unchanged.

Alternative considered: read `MISE_DATA_DIR`, `ASDF_DATA_DIR` or `brew --prefix`. Rejected for three reasons. The path already holds the prefix. The environment of the install can differ from the user's shell. A `brew` call from an install is slow and can fail.

### The candidate must be the same file

The function calls `os.Stat` on the path and on the candidate, and compares the results with `os.SameFile`. A missing candidate, a broken link or a different file means no stable link. This check is the positive assertion. Without it, a `latest` that names another version switches the agents to a binary the user did not run. Alternative considered: compare `filepath.EvalSymlinks` results as strings. Rejected, because `os.SameFile` also handles a hard link and a case-insensitive volume.

### The CLI applies the function, and only `install` warns

`install.go` calls `StableExecPath` on the `os.Executable()` result before it builds `InstallOptions`. So `Validate` checks the recorded path. If `versioned` is true, the command writes one line to `cmd.ErrOrStderr()` before the target loop:

```
warning: <path> is a versioned package-manager path. A package-manager upgrade or prune can break it. Run `codegraph install` again after each upgrade.
```

`refreshInstalledSkills` calls the function on its `execPath` argument and ignores `versioned`. The self-replace target in `newUpgradeCmd` keeps the real path. Alternative considered: a refusal, like the ephemeral check. Rejected, because the binary works today. A refusal also blocks every asdf user.

### A seam over `os.Executable` in `install.go`

A package var `executablePath = os.Executable` lets a CLI test give the command a path inside a temporary mise or asdf layout. `upgrade.go` needs no seam. A test can call `refreshInstalledSkills` with a path directly.

## Risks / Trade-offs

- [A user pins an older version with mise while `latest` names it, then installs a newer version] → The agents follow `latest` to the newer version. This is the same as the Homebrew `bin` link, and it is the behavior #231 asks for. A user who wants the pin runs `install` from a path that is not under `installs/`.
- [mise changes its `latest` link or the layout] → The same-file check fails, and the install records the real path with a warning. The result is never worse than today.
- [A non-package-manager path has an `installs` element, for example `/opt/installs/team/1.0/codegraph`] → It counts as versioned. With no `latest` sibling, it gets a warning but is recorded unchanged.
- [The warning is a new line on stderr] → stdout and the exit status do not change, so agents that parse stdout see no difference.

## Migration Plan

Nothing migrates on its own. After the release, a user with a stale versioned path runs `codegraph install` once from the new version, and the configurations then name the stable link. Rollback is a revert. The configurations that name a stable link keep working after a revert.
