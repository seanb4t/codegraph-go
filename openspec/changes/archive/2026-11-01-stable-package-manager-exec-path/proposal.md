# Proposal: Record a stable link in place of a versioned package-manager path

Issue: #231

## Why

`codegraph install` records the path that `os.Executable()` returns. For a mise install, that path names one version, for example `~/.local/share/mise/installs/github-seanb4t-codegraph-go/0.15.0/codegraph`. A user who upgrades with mise does not run `codegraph upgrade`, so nothing rewrites the agent configurations. The agents keep the old version. If mise prunes that version, the MCP server cannot start. A Homebrew cask install on Linux has the same problem with `Caskroom/codegraph/<version>/codegraph`.

## What Changes

- Before it records a path, an install or refresh looks for a stable link that the package manager keeps. For mise it is the `latest` entry next to the version directory. For a Homebrew cask it is the `bin/<name>` link under the Homebrew prefix.
- If the link resolves to the same file as the running binary, it records the link. Otherwise it records the real path, as it does today.
- If the real path is a versioned package-manager path and no link qualifies, the install prints one warning. The warning names the path and says that the next upgrade can break it. An asdf install is the usual case, because asdf keeps no such link.
- The ephemeral-path refusal and its `CODEGRAPH_ALLOW_EPHEMERAL_EXEC_PATH` override do not change.

Out of scope:

- Rewriting configurations that already hold a stale versioned path. The user runs `codegraph install` once more to repair them.
- Recording the mise shim (`~/.local/share/mise/shims/codegraph`). The shim is the mise binary, not codegraph, so the same-file check cannot accept it, and it picks a version per directory.
- Package managers other than mise, asdf and Homebrew casks, for example aqua or Nix.
- The self-replace target of `codegraph upgrade` inside a package-manager directory.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `agent-install`: the durable-path requirement (D-04) names the path that is recorded, which can now be a stable link. A new requirement defines the stable links, the same-file check and the warning.

## Impact

- `internal/agents`: a new path function next to `ephemeralExecPath` in `types.go`, with tests in `types_test.go`.
- `internal/cli/install.go` and `internal/cli/upgrade.go`: both apply the function to the `os.Executable()` result before `Validate`, and print the warning. `install.go` gets a seam over `os.Executable` so that a test can give it a versioned path.
- CLI output: a new warning line on stderr for a versioned path with no stable link. stdout does not change.
- No new dependencies.
