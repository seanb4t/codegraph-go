# Spec Delta

## ADDED Requirements

### Requirement: A global agent file is edited through a symlink

An agent file is an agent configuration file or an instruction file. At global scope, an install or a refresh SHALL edit the file that a symlink at an agent file leads to. The symlink SHALL stay as it was. The file it leads to SHALL keep its permission bits. If that file does not exist, the install SHALL create it. The result SHALL name the configured path.

#### Scenario: A configuration file kept in a dotfile manager

- **GIVEN** `~/.claude.json` is a symlink to `<dotfiles>/claude.json`, which lists another MCP server
- **WHEN** `codegraph install --location global` runs
- **THEN** `~/.claude.json` is still a symlink to `<dotfiles>/claude.json`
- **AND** `<dotfiles>/claude.json` holds the `codegraph` entry and the other MCP server

#### Scenario: A chain of two links

- **GIVEN** `~/.gemini/settings.json` is a symlink to `~/.gemini/link2`, and `~/.gemini/link2` is a symlink to `<dotfiles>/gemini.json`
- **WHEN** `codegraph install --location global` runs
- **THEN** both symlinks are unchanged and `<dotfiles>/gemini.json` holds the `codegraph` entry

#### Scenario: A link to a file that does not exist yet

- **GIVEN** `~/.claude.json` is a symlink to `<dotfiles>/claude.json`, and that file does not exist
- **WHEN** `codegraph install --location global` runs
- **THEN** `<dotfiles>/claude.json` exists and holds the `codegraph` entry, and `~/.claude.json` is still a symlink

#### Scenario: A relative link under a linked directory

- **GIVEN** `~/.claude` is a symlink to `<dotfiles>/claude`, and `~/.claude/CLAUDE.md` is a symlink to `../shared/CLAUDE.md`
- **WHEN** `codegraph install --location global` runs
- **THEN** `<dotfiles>/shared/CLAUDE.md` holds the codegraph section, and no file exists at `~/shared/CLAUDE.md`

#### Scenario: The file keeps its permission bits

- **GIVEN** `~/.claude.json` is a symlink to a file with mode `0640`
- **WHEN** `codegraph install --location global` runs
- **THEN** the file it leads to holds the `codegraph` entry and still has mode `0640`

### Requirement: An uninstall keeps a symlink

An uninstall SHALL edit the file that a symlink at an agent file leads to. The symlink SHALL stay as it was. If the uninstall leaves that file with no content, it SHALL delete the file that the symlink leads to, and SHALL keep the symlink. If another installed agent uses the same real instruction file, the uninstall SHALL keep the codegraph section and SHALL report the file as `kept`.

#### Scenario: Uninstall keeps the link

- **GIVEN** `~/.claude.json` is a symlink to `<dotfiles>/claude.json`, which holds the `codegraph` entry and another MCP server
- **WHEN** `codegraph uninstall --location global` runs
- **THEN** `~/.claude.json` is still a symlink, and `<dotfiles>/claude.json` holds only the other MCP server

#### Scenario: Uninstall that empties the file keeps the link

- **GIVEN** `~/.claude/CLAUDE.md` is a symlink to `<dotfiles>/CLAUDE.md`, which holds only the codegraph section
- **WHEN** `codegraph uninstall --location global` runs
- **THEN** `<dotfiles>/CLAUDE.md` does not exist, and `~/.claude/CLAUDE.md` is still a symlink to that path

#### Scenario: Uninstall of one of two agents that share a linked file

- **GIVEN** `GEMINI.md` in the project is a symlink to `AGENTS.md` in the project, and a local install of codex and gemini ran
- **WHEN** `codegraph uninstall --location local --target gemini` runs
- **THEN** `AGENTS.md` still holds the codegraph section
- **AND** the output reports `GEMINI.md` as `kept`

### Requirement: A local agent file inside the project is edited through a symlink

At local scope, the project is the directory the command runs in. The real location of a file is its path after every symlink in it is resolved. If the real location of an agent file is inside the project, the command SHALL edit the file there. This applies to an install, a refresh and an uninstall. Every symlink SHALL stay as it was.

#### Scenario: One instruction file linked to another in the project

- **GIVEN** `GEMINI.md` in the project is a symlink to `AGENTS.md` in the project
- **WHEN** `codegraph install --location local --target gemini` runs
- **THEN** `GEMINI.md` is still a symlink, and `AGENTS.md` holds the codegraph section

#### Scenario: A file inside the project whose name starts with two dots

- **GIVEN** `AGENTS.md` in the project is a symlink to `..notes.md` in the project
- **WHEN** `codegraph install --location local --target codex` runs
- **THEN** `..notes.md` holds the codegraph section, and `AGENTS.md` is still a symlink

### Requirement: A local install refuses a file outside the project

This requirement applies at local scope. If the real location of an agent file is outside the project, an install or a refresh SHALL report an error. The error SHALL name the configured path and the real location. The command SHALL write nothing for that file. The symlink and the outside file SHALL stay byte for byte as they were. The command SHALL continue with the other files and agents, and SHALL exit non-zero.

#### Scenario: An instruction file linked outside the project

- **GIVEN** `AGENTS.md` in the project is a symlink to `<home>/.zshrc`
- **WHEN** `codegraph install --location local --target codex` runs
- **THEN** the output has an error that names `AGENTS.md` and `<home>/.zshrc`, and the command exits non-zero
- **AND** `<home>/.zshrc` is byte for byte as it was, and `AGENTS.md` is still a symlink to it

#### Scenario: A configuration file linked outside the project

- **GIVEN** `.mcp.json` in the project is a symlink to `<home>/.claude.json`, which lists another MCP server
- **WHEN** `codegraph install --location local --target claude` runs
- **THEN** the output has an error that names `.mcp.json`, and the command exits non-zero
- **AND** `<home>/.claude.json` is byte for byte as it was, and `.mcp.json` is still a symlink to it

#### Scenario: A configuration file linked outside that already holds the entry

- **GIVEN** `.mcp.json` in the project is a symlink to `<home>/.claude.json`, which already holds the same `codegraph` entry that the install writes
- **WHEN** `codegraph install --location local --target claude` runs
- **THEN** the output has an error that names `.mcp.json`, and the command exits non-zero

#### Scenario: A link that leaves the project through a linked directory

- **GIVEN** `AGENTS.md` in the project is a symlink to `docs/notes.md` in the project
- **AND** `docs` is a symlink to a directory outside the project that holds `notes.md`
- **WHEN** `codegraph install --location local --target codex` runs
- **THEN** the output has an error that names `AGENTS.md`, and `AGENTS.md` is still a symlink
- **AND** `notes.md` in the outside directory is byte for byte as it was

#### Scenario: A link whose target climbs out of a linked directory

- **GIVEN** `sub` in the project is a symlink to `<outside>/a`, and `AGENTS.md` is a symlink to `sub/../x.md`, which does not exist
- **WHEN** `codegraph install --location local --target codex` runs
- **THEN** the output has an error that names `AGENTS.md`, and no file exists at `<outside>/x.md` or at `x.md` in the project

#### Scenario: A file under a directory that is linked outside the project

- **GIVEN** `.claude` in the project is a symlink to a directory outside the project
- **WHEN** `codegraph install --location local --target claude` runs
- **THEN** the output has an error that names `.claude/settings.json` and an error that names `.claude/CLAUDE.md`
- **AND** the outside directory holds no `settings.json` and no `CLAUDE.md`

### Requirement: A local uninstall leaves a file outside the project

This requirement applies at local scope. If the real location of an agent file is outside the project, an uninstall SHALL change nothing for that file. The symlink and the outside file SHALL stay byte for byte as they were. The output SHALL report the file with the action `kept` and SHALL have a note that names the configured path. That file SHALL NOT make the command exit non-zero.

#### Scenario: Uninstall leaves a link that leads outside the project

- **GIVEN** `AGENTS.md` in the project is a symlink to `<home>/notes/agents.md`, which holds the codegraph section
- **WHEN** `codegraph uninstall --location local --target codex` runs
- **THEN** `<home>/notes/agents.md` is byte for byte as it was, and `AGENTS.md` is still a symlink to it
- **AND** the output reports `AGENTS.md` as `kept`, has a note that names `AGENTS.md`, and the command exits zero

### Requirement: A symlink loop writes nothing

If the symlink at an agent file is part of a loop, the command SHALL report an error that names the configured path. This applies to an install, a refresh and an uninstall. The same rule SHALL apply to a chain of symlinks that cannot be resolved. The command SHALL write nothing for that file and SHALL leave every link as it was. It SHALL continue with the other files and agents, and SHALL exit non-zero.

#### Scenario: Two links that point at each other on install

- **GIVEN** `~/.claude.json` is a symlink to `~/a`, and `~/a` is a symlink to `~/.claude.json`
- **WHEN** `codegraph install --location global` runs
- **THEN** the output has an error that names `~/.claude.json`, both symlinks are unchanged, and the command exits non-zero

#### Scenario: Two links that point at each other on uninstall

- **GIVEN** `~/.codex/config.toml` is a symlink to `~/.codex/a`, and `~/.codex/a` is a symlink to `~/.codex/config.toml`
- **WHEN** `codegraph uninstall --location global --target codex` runs
- **THEN** the output has an error that names `~/.codex/config.toml`, both symlinks are unchanged, and the command exits non-zero

### Requirement: An MCP configuration file that does not parse is not written

This requirement applies to a JSON or JSONC file that holds an agent's MCP server entries. If its content does not parse, an install or a refresh SHALL report an error that names the file. The file SHALL stay byte for byte as it was. The command SHALL continue with the other files and agents, and SHALL exit non-zero. A file that is missing or empty SHALL be written as a new file. The rule SHALL apply to the file that a symlink leads to.

#### Scenario: A file with a syntax error

- **GIVEN** `~/.gemini/settings.json` holds text that is not valid JSON
- **WHEN** `codegraph install --location global --target gemini` runs
- **THEN** the output has an error that names `~/.gemini/settings.json`, the file is byte for byte as it was, and the command exits non-zero

#### Scenario: An OpenCode file with a syntax error

- **GIVEN** the global `opencode.jsonc` holds text that is not valid JSONC
- **WHEN** `codegraph install --location global --target opencode` runs
- **THEN** the output has an error that names that file, and the file is byte for byte as it was

#### Scenario: An empty file

- **GIVEN** `~/.gemini/settings.json` exists and has no content
- **WHEN** `codegraph install --location global --target gemini` runs
- **THEN** `~/.gemini/settings.json` holds the `codegraph` entry

#### Scenario: A link to a file with a syntax error

- **GIVEN** `~/.claude.json` is a symlink to `<dotfiles>/claude.json`, which holds text that is not valid JSON
- **WHEN** `codegraph install --location global --target claude` runs
- **THEN** the output has an error that names `<dotfiles>/claude.json`, and that file is byte for byte as it was
- **AND** `~/.claude.json` is still a symlink
