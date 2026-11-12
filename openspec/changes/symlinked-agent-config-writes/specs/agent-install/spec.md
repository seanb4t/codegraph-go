# Spec Delta

## ADDED Requirements

### Requirement: A global agent file is written through a symlink

At global scope, an install, a refresh or an uninstall that edits an agent configuration file or an instruction file SHALL write through a symlink at that path. The symlink SHALL stay a symlink, and the file it leads to SHALL receive the new content and keep its permission bits. A symlink whose target does not exist yet SHALL be treated the same way: the target is created.

#### Scenario: A configuration file kept in a dotfile manager

- **GIVEN** `~/.claude.json` is a symlink to `<dotfiles>/claude.json`, which lists another MCP server
- **WHEN** `codegraph install --location global` runs
- **THEN** `~/.claude.json` is still a symlink to `<dotfiles>/claude.json`
- **AND** `<dotfiles>/claude.json` holds the `codegraph` entry and the other MCP server

#### Scenario: A chain of two links

- **GIVEN** `~/.gemini/settings.json` is a symlink to `link2`, and `link2` is a symlink to `<dotfiles>/gemini.json`
- **WHEN** `codegraph install --location global` runs
- **THEN** both symlinks are unchanged and `<dotfiles>/gemini.json` holds the `codegraph` entry

#### Scenario: A link to a file that does not exist yet

- **GIVEN** `~/.claude.json` is a symlink to `<dotfiles>/claude.json`, and that file does not exist
- **WHEN** `codegraph install --location global` runs
- **THEN** `<dotfiles>/claude.json` exists and holds the `codegraph` entry, and `~/.claude.json` is still a symlink

#### Scenario: Uninstall keeps the link

- **GIVEN** a global install wrote through the symlink `~/.claude.json`
- **WHEN** `codegraph uninstall --location global` runs
- **THEN** `~/.claude.json` is still a symlink, and the file it leads to no longer holds the `codegraph` entry

#### Scenario: The target keeps its permission bits

- **GIVEN** `~/.claude.json` is a symlink to a file with mode `0600`
- **WHEN** `codegraph install --location global` runs
- **THEN** the file it leads to still has mode `0600`

### Requirement: A local agent file is written through a symlink only inside the project

At local scope, the project is the directory the command runs in. An install, a refresh or an uninstall that edits an agent configuration file or an instruction file SHALL write through a symlink at that path when the file it leads to is inside the project. When the symlink leads outside the project, the command SHALL replace the symlink with a regular file and SHALL leave the outside file byte for byte as it was.

#### Scenario: One instruction file linked to another in the project

- **GIVEN** `AGENTS.md` in the project is a symlink to `CLAUDE.md` in the project
- **WHEN** `codegraph install --location local` writes its section to `AGENTS.md`
- **THEN** `AGENTS.md` is still a symlink, and `CLAUDE.md` holds the section

#### Scenario: An instruction file linked outside the project

- **GIVEN** `CLAUDE.md` in the project is a symlink to `<home>/.zshrc`
- **WHEN** `codegraph install --location local` runs
- **THEN** `<home>/.zshrc` is byte for byte as it was
- **AND** `CLAUDE.md` in the project is a regular file that holds the codegraph section

#### Scenario: A configuration file linked outside the project

- **GIVEN** `.mcp.json` in the project is a symlink to `<home>/.claude.json`, which lists another MCP server
- **WHEN** `codegraph install --location local` runs
- **THEN** `<home>/.claude.json` is byte for byte as it was
- **AND** `.mcp.json` in the project is a regular file

#### Scenario: A link that leaves the project through a second link

- **GIVEN** `CLAUDE.md` in the project is a symlink to `docs/notes.md` in the project, and `docs` is a symlink to a directory outside the project
- **WHEN** `codegraph install --location local` runs
- **THEN** no file outside the project changes, and `CLAUDE.md` in the project is a regular file

### Requirement: A symlink loop writes nothing

When the symlink at a file to edit is part of a loop, or of a chain of more than 40 links, the command SHALL report an error that names the path, SHALL write nothing for that file and SHALL leave every link as it was. It SHALL continue with the other files and agents, and SHALL exit non-zero.

#### Scenario: Two links that point at each other

- **GIVEN** `~/.claude.json` is a symlink to `a`, and `a` is a symlink to `~/.claude.json`
- **WHEN** `codegraph install --location global` runs
- **THEN** the output has an error that names `~/.claude.json`, both symlinks are unchanged, and the command exits non-zero

### Requirement: Files that codegraph owns replace a symlink

A file that codegraph owns outright SHALL be written by a rename over its path at both scopes, so a symlink at that path becomes a regular file and the file it led to does not change. These files are the guard scripts, the skill package, the install manifest, the git hook scripts and the daemon records.

#### Scenario: A guard script path that is a symlink

- **GIVEN** the path of a guard script that a global install writes is a symlink to a file elsewhere
- **WHEN** `codegraph install --location global` runs
- **THEN** the guard script path is a regular file with the guard content, and the file elsewhere is byte for byte as it was
