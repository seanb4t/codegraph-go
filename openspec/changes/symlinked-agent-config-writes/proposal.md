# Write agent configuration through a symlink when that is safe
Issue: #293

## Why

`codegraph install` writes each agent file with a temporary file and a rename over the path. When the path is a symlink, the rename replaces the link with a regular file. The file that the link pointed to keeps its old content. A person who keeps `~/.claude.json` in a dotfile manager loses the link on every install. A repository that links `AGENTS.md` to `CLAUDE.md` loses that link too.

Following every link is not the answer. A file at local scope comes from the repository, and a repository can commit `CLAUDE.md` as a link to `~/.zshrc`. An install that follows such a link writes to a file outside the project.

## What Changes

- An install, a refresh or an uninstall at global scope writes through a symlink at an agent configuration or instruction file. The link stays, and the file it leads to receives the new content.
- At local scope, if the symlink leads to a file inside the project, the same operations write through it. If it leads outside the project, they replace the link with a regular file, as today, and do not touch the outside file.
- A symlink loop, or a chain of more than 40 links, is an error that names the path. Nothing is written for that file.
- Files that codegraph owns outright keep the present behavior: guard scripts, the skill package, the install manifest, git hook scripts and daemon records.

Out of scope:

- A symlink in a parent directory of the path. The rename already works through it.
- The MCP `path` confinement, which #299 fixed.
- Git hook scripts. A hook path can be a link to a script that a hook manager shares between repositories, so those writes keep replacing the link.
- Native Windows.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `agent-install`: adds requirements for how an install, a refresh and an uninstall treat a symlink at a file they edit.

## Impact

- `internal/fsatomic`: one new entry point that writes through a link. `WriteFile` keeps its contract, so `internal/githooks` and `internal/daemon` need no change.
- `internal/agents`: the helpers that edit a user's file take a link policy that the target derives from the install location. About 15 write sites and their callers change signature.
- `SECURITY.md`: the threat model states the local-scope rule.
- No CLI flag, output shape or MCP result changes. A file that is written through a link reports the same action as today.
- No new dependency.
