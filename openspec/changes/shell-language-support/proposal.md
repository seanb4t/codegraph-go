# Proposal
Issue: #240

## Why

CodeGraph has no shell language registration, despite shell scripts and startup files carrying important project behavior. Shared syntax cannot justify treating Bash, POSIX shell, Zsh, and Fish as one language.

## What Changes

- Add separate indexing contracts for POSIX shell, Bash, Zsh, and Fish.
- Discover scripts through extensions, recognized shebangs, and a bounded list of startup filenames.
- Index named functions, statically named variables, reads, and direct calls within each dialect contract.
- Resolve literal sourced files within the repository without executing scripts or searching the runtime environment.
- Support repository-local `.config/fish` discovery through the same rules that file watching uses.
- Qualify maintained parser candidates before registration and publish dialect-specific limitations.

Out of scope: arbitrary shell execution, `eval`, runtime expansion, environment-dependent command lookup, plugin autoloading, and shell embedded in YAML or Makefiles. The change does not index the user home directory as a project.

## Capabilities

### New Capabilities

- `shell-indexing`: Dialect-aware shell discovery, static graph extraction, and explicit resolution boundaries.

### Modified Capabilities

None.

## Impact

The change affects language registration, discovery, directory exclusions, file watching, parser adapters, extractors, and public language filters.

Parser dependencies require license and platform review. The released application remains one binary without a shell runtime dependency. Existing CLI and MCP output envelopes remain compatible.

The implementation of `index-invariants`, proposed in [PR #237](https://github.com/seanb4t/codegraph-go/pull/237), supplies structural invariants. Issue #238 supplies evaluation conventions. Issue #239 supplies shared field and binding conventions where useful, but shell support does not depend on all core-language fixes.
