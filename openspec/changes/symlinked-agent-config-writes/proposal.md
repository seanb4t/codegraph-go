# Write agent configuration through a symlink when that is safe
Issue: #293

## Why

`codegraph install` writes each agent file with a temporary file and a rename over the path. When the path is a symlink, the rename replaces the link with a regular file. The file that the link pointed to keeps its old content. A person who keeps `~/.claude.json` in a dotfile manager loses the link on every install. A repository that links `GEMINI.md` to `AGENTS.md` loses that link too.

Following every link is not the answer. A file at local scope comes from the repository, and a repository can commit `AGENTS.md` as a link to `~/.zshrc`. An install that follows such a link writes to a file outside the project.

Today's behavior has a second fault at local scope. The read follows the link and the write does not. When `.mcp.json` links to `~/.claude.json`, the install copies the whole of `~/.claude.json` into the project.

## What Changes

- At global scope, an install, a refresh or an uninstall edits the file that a symlink leads to. The link stays.
- If a local file is a symlink to a file inside the project, the same operations edit that file.
- At local scope, an install or a refresh refuses a file whose real location is outside the project. It reports an error that names the path and the location, writes nothing for that file and exits non-zero. An uninstall leaves such a file alone and adds a note.
- An uninstall that empties a linked file deletes the file that the link leads to. The link stays.
- A symlink loop is an error that names the path, for an install and for an uninstall. Nothing is written for that file.
- An MCP configuration file that does not parse is not written. #292 added this rule in code for two writers. This change records it in the spec and extends it to a file behind a link.

The refusal at local scope is a behavior change. A local install in a directory whose `AGENTS.md` or `.mcp.json` links to a parent directory succeeded before and now fails for that file. The error states three ways forward: run the install in the directory that holds the real file, use global scope, or replace the link with a regular file.

Out of scope:

- Files that codegraph owns outright: guard scripts, the skill package, the install manifest, git hook scripts and daemon records. Their behavior does not change, and this change adds no requirement for them. A local install still writes them through a project directory that is a link to an outside directory.
- A Codex or Hermes configuration file that cannot be read. #304 tracks that fault.
- The symlink normalizers in `internal/gitmeta`, `internal/daemon`, the terminal UI and `cursor.go`. Each one makes a path canonical for a comparison and falls back to the input on failure. None of them decides a write.
- The MCP `path` confinement, which #299 fixed.
- A person who changes a link between the check and the write.
- Native Windows.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `agent-install`: adds requirements for a symlink at a file that an install, a refresh or an uninstall edits. It also adds one for a configuration file that does not parse.

## Impact

- New leaf package `internal/realpath`: the "is this path inside that directory" test and one resolver for a path whose target does not exist yet. The test moves from `internal/mcp` with no change. The resolver replaces the one in `internal/agents`. `internal/query` does not change.
- `internal/agents`: each target resolves the path of a user's file once, before the existing helpers read, write or delete it. The helpers keep their signatures. The check for two agents that share one instruction file compares real locations.
- `internal/fsatomic`: no change.
- `SECURITY.md`: the threat model states the local-scope rule.
- Output: a file behind a link reports the configured path and the same action as today. The new error is an additional line. An uninstall reports a local file outside the project with the existing action `kept` and a note. No CLI flag or MCP result changes.
- Depends on #303, the fix for #292. The implementation starts after #303 is on `main`.
- No new dependency. The property test uses `pgregory.net/rapid`, which `go.mod` already lists.
