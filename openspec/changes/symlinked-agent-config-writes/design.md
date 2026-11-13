# Design

## Context

See proposal.md for the motivation. These facts are read from `main` at `a12602e6`.

- `fsatomic.WriteFile(path, content)` makes a temporary file in `filepath.Dir(path)` and renames it over `path`. A rename over a symlink replaces the link. It copies the permission bits from `os.Stat(path)`.
- The helpers in `internal/agents` take one path. They read it with `os.ReadFile` and write it through `atomicWriteFile`. Three places delete a user's file with `os.Remove`: `shared.go:490`, `shared.go:750` and `codex.go:314`.
- `fileExists` is `os.Stat`, so a symlink loop reads as "no file".
- Most `Install` and `Uninstall` steps record an outcome through `recordFile` or `recordAction` in `shared.go`. Both take the path for the error text. Some steps append to `result.Files` and `result.Errors` directly. They are the configuration edits in `codex.go` and `hermes.go`, and the Antigravity migration with its unified write (`antigravity.go:210-250`). In `claude.go`, they are the legacy `.claude.json` migration (lines 460-467) and the removal of the PreToolUse entry (lines 636-647).
- The uninstalls of codex (`codex.go:302`) and hermes (`hermes.go:407`) test `fileExists` before they read the configuration file.
- Some call sites use the error of a helper after they record it. Claude's `Install` sets flags for its manifest from it. The Antigravity migration sets `migrationOK` from it.
- The error of a helper does not always name a path. The TOML conflict error and a JSON parse error carry none.
- Each target's `Install` and `Uninstall` receive the `Location`. The helpers do not. Local paths are relative to the working directory. `cursor.go:115` derives the local root from `os.Getwd` and `filepath.EvalSymlinks`, and falls back to the unresolved directory.
- `uninstall` runs every registered target by default.
- The "inside this directory" test exists twice: `isWithin` with `resolveExistingPrefix` in `internal/mcp/tools.go`, and inline `filepath.Rel` code in `resolveSourcePath` in `internal/query/node.go`.
- `resolveSkillDir` in `internal/agents/skillshared.go` makes a path absolute and resolves it, also through a link whose target does not exist yet. Its bound is `maxSymlinkResolveDepth`, which is 8. The bound counts each link hop and each missing parent directory. It joins a relative link target to the textual directory of the link and cleans the result as text. `sameSkillDir` compares two paths by that resolver.
- `instructionsRequestedElsewhere` in `shared.go` decides whether another agent uses the same instruction file. It compares absolute paths as text. The uninstalls of codex and opencode call it before `removeMarkedSection`. If it finds an agent, they record `ActionKept` with a note. The uninstalls of claude (`claude.go:711`) and gemini (`gemini.go:158`) call `removeMarkedSection` with no such test.
- `ActionKept` is the action for a file left as it is on purpose. `codex.go:332` and `opencode.go:346` each append it and a note with the same two lines.
- `errTOMLTableConflict` in `toml.go` is a sentinel error that callers wrap with context.
- The Antigravity `Install` migration at `antigravity.go:210-222` tests, reads and writes the unified configuration file outside the helpers. It also reads the legacy file and removes the entry from it.
- #303, the fix for #292, makes `writeMcpEntry` and `writeOpencodeEntry` refuse a file that does not parse. It is not on `main` yet.

## Goals / Non-Goals

**Goals:**

- An edit to a user's file reaches the file that the operating system resolves, for the read, the write and the delete alike.
- A local install cannot write a user's file outside the project through a symlink, and cannot copy content from one.
- The helpers and `fsatomic` keep their contracts.

**Non-Goals:**

- No protection against a person who changes a link between the check and the write. That needs write access to the directory while the command runs.
- No change to files that codegraph owns outright. Their writers stay as they are, also under a project directory that is a link.
- No change to `Detect`. It only reads.
- No change to `readFileOrEmpty`. #304 tracks its fault.
- No change to the behavior of the MCP `path` confinement.

## Decisions

### D1. Resolve the path once, then use the existing helpers

Each target resolves the path of a user's file before it calls a helper. The helper then reads, writes or deletes the resolved path. Because the resolved path is not a symlink, each existing operation does the right thing with no change:

- `os.ReadFile` and `fsatomic.WriteFile` reach the same file.
- `fsatomic.WriteFile` renames inside the directory of the real file, so the link above it stays.
- `os.Remove` deletes the real file, so an uninstall that empties the file keeps the link.
- `fileExists` never sees a loop, because the resolve step already failed on it.

Alternative: a second `fsatomic` entry point that writes through a link, with a link policy passed down through every helper. Rejected after the first plan review. It covered the write and left the read, the delete and the existence check on the old path. It also put a project rule into a package whose contract is the atomic write alone.

### D2. One shared package holds the resolver and the "inside" test

New leaf package `internal/realpath`, with no import outside the standard library:

- `Within(root, path string) bool` is `isWithin` moved from `internal/mcp`, which then calls the shared one. The function does not change, so the MCP behavior does not change. It takes `filepath.Rel(root, path)`. A result of `..`, or one that starts with `..` plus the separator, means outside. A file named `..notes.md` inside the root is inside.
- `Resolve(path string) (string, error)` returns the location that the operating system reaches for `path`. It accepts a relative path and makes it absolute. It replaces `resolveSkillDir` and `resolveSkillDirDepth` in `internal/agents`, which go away.

`Resolve` tries `filepath.EvalSymlinks` first. If a part of the path does not exist, that call fails. `Resolve` then walks the path one element at a time from the longest part that exists:

- For `..`, it takes the parent of the real current directory.
- For an element that is a link, it reads the link and walks the target the same way. A relative target starts at the real directory of the link.
- For an element that does not exist, it joins the name on. Every later element is joined the same way.
- For `..` after an element that does not exist, it returns an error. The operating system refuses such a path.

Each link that the walk reads counts against the bound of 8. A missing directory does not count. The walk never cleans a path as text, because `..` after a link names the parent of the link's target. Two examples that a textual join gets wrong:

- With `~/.claude -> dotfiles/claude` and `~/.claude/CLAUDE.md -> ../shared/CLAUDE.md`, the operating system reaches `dotfiles/shared/CLAUDE.md`. The textual join names `~/shared/CLAUDE.md`.
- With `sub` in a project linked to `/outside/a` and `AGENTS.md -> sub/../x.md`, the operating system reaches `/outside/x.md`. The textual join names `x.md` in the project.

`sameSkillDir` becomes `sameRealPath` and calls `realpath.Resolve`. In `skillshared_test.go`, the six calls of `sameSkillDir` take the new name, and `TestSkillPackage_DanglingSymlinkDirIsRecreated` calls `realpath.Resolve`. No assertion of an existing test changes.

`resolveExistingPrefix` stays private in `internal/mcp`, its one caller. `resolveSourcePath` in `internal/query` stays as it is. Its inline test runs on two paths that exist. No part of this change needs it to move.

The spec names no number for the chain limit. `filepath.EvalSymlinks` follows a chain of up to 255 links. A chain that is longer than the limit of the operating system and shorter than 255 resolves. The helpers then receive a path with no link in it, so the edit works.

### D3. `resolveUserFile` applies the scope rule

`internal/agents` adds:

```go
// errOutsideProject reports a local file whose real location is outside
// the project. resolveUserFile wraps it with the real location.
var errOutsideProject = errors.New("leads outside the project")

// resolveUserFile returns the real location of a file that an install or an
// uninstall edits.
func resolveUserFile(path string, loc Location) (string, error)

// localProjectRoot returns the real location of the working directory.
func localProjectRoot() (string, error)
```

1. Call `realpath.Resolve(path)`. If it fails, return the error. A loop fails here.
2. At global scope, return the result.
3. At local scope, call `localProjectRoot`. If `realpath.Within` reports the result inside the root, return the result. Otherwise return `errOutsideProject`, wrapped with the real location.

The error is a wrapped sentinel, as `errTOMLTableConflict` is. A caller needs only `errors.Is`, and the configured path comes from the funnel in D4.

If `os.Getwd` or `filepath.EvalSymlinks` fails, `localProjectRoot` returns the error, and `resolveUserFile` refuses the file. `cursorTarget.Install` calls the same function and keeps its present fallback to the unresolved directory on an error. Its value goes into a configuration entry and decides no write.

The check is on the real location of the whole path, so a linked directory above a user's file counts.

Alternatives:

- `os.Root`, the standard library's file API that is confined to one directory tree. Rejected. It refuses every absolute link, also one with a target inside the tree. It gives nothing at global scope, where a link leaves the directory by design. The helpers and `fsatomic` take paths, so each of them needs a second form.
- Derive the scope from the path, and treat a path under the working directory as local. Rejected. A global install that runs with the home directory as the working directory then gets the local rule.

### D4. One funnel records a user's file under the configured path

```go
// recordUserFile resolves path, calls edit with the real location, and
// records the outcome under the configured path. It returns the error that
// it recorded, or nil.
func recordUserFile(result *WriteResult, path string, loc Location, op userFileOp, edit func(real string) (FileResult, error)) error

// recordUserAction adapts a helper that returns a FileAction.
func recordUserAction(result *WriteResult, path string, loc Location, op userFileOp, edit func(real string) (FileAction, error)) error

// recordKept records path as kept on purpose, with a note.
func recordKept(result *WriteResult, path, note string)
```

`recordUserFile` holds the resolve, the scope rule of D5 and the path that is reported. It replaces the `FileResult.Path` of the helper with the configured path, and ends in `recordFile`. `recordUserAction` only wraps the `FileAction` into a `FileResult` and calls `recordUserFile`. `op` is `opInstall` or `opUninstall`.

The funnel returns the error. Claude's `Install` keeps its manifest flags from that value, and the Antigravity migration keeps `migrationOK` from it.

If `os.Lstat` reports that the configured path is a symlink, the funnel adds the real location to a helper error. `recordFile` puts the configured path in front. An error line for a linked file therefore names both, whatever the helper error says. For a path that is not a symlink, the error text is the same as today.

The steps that record directly today become `edit` functions that return a `FileResult`. The Context lists them. The existence tests of the codex and hermes uninstalls move into the `edit` function. They then run after the resolve, so a loop is an error there too. The Antigravity migration also resolves its legacy path with `resolveUserFile`, and uses the result for its existence test, its read and its `removeMcpEntry` call.

`recordKept` is the two lines that `codex.go:332` and `opencode.go:346` repeat. Both sites and D5 call it.

The implementation settled four points that the plan review left open:

- An `edit` function that returns a `FileResult` with no `Action` records nothing. The Claude legacy migration and the Claude PreToolUse removal use this for a step that changed nothing.
- `removedOnly` wraps a remover for a step that reports only a removal. The Claude legacy migration, the Claude PreToolUse removal and the Antigravity legacy removal use it.
- The Antigravity migration does not become one `edit` function. It resolves the unified path and the legacy path with `resolveUserFile` for its existence tests and its reads. Its copy of the legacy entry into the unified file records directly, under the configured path. The legacy removal and the unified `writeMcpEntry` go through the funnel.
- The OpenCode cleanup of a stale configuration file resolves its path with `resolveUserFile` at global scope. It skips a path that does not resolve. It also skips a path whose real location is inside the live configuration directory, because the entry there is the one the install just wrote.
- The Codex `hooks.json` file holds hook entries, so its write and its removal go through the funnel too. The removal takes the operation from its caller. An install that turns the nudge off passes `opInstall`, so D5 applies to it as an install.
- `resolveSourcePath` in `internal/query` keeps its two inline copies of the "inside" test. The user cut that move from the scope after the fourth plan review.
- `resolveExistingPrefix` in `internal/mcp` and `realpath.Resolve` both start at the longest part of a path that exists. They differ on purpose in the rest. The MCP confinement joins the missing rest as text, because a path that does not exist has no links of its own. `Resolve` follows a link in the rest, because a dangling link is an entry that exists.

The Cursor and Kiro legacy instruction files stay outside the funnel. `codegraph` wrote their whole content, and `os.Remove` on a link deletes the link and never its target.

### D5. A local file outside the project: refuse on install, keep on uninstall

With `opInstall`, the funnel records the error. The message names the configured path and the real location, and ends with the three ways forward: run the install in the directory that holds the real file, use `--location global`, or replace the link with a regular file. The check runs before any read, so the unchanged short-circuit in the helpers cannot skip it, and no content of the outside file is copied.

With `opUninstall`, the funnel tests for `errOutsideProject` with `errors.Is`. It calls `recordKept` with a note that names the configured path, and returns nil. `uninstall` runs every target by default. An error here fails the command for an agent that was never installed in this project.

Alternative considered with the user: replace the link with a new regular file that holds only the codegraph content, and add a note. Rejected. It makes the install succeed by removing a link that the user made on purpose. In a monorepo, the agent then starts without the shared servers, and the note is easy to miss. The refusal matches #292: when a user's file is in a state the tool cannot handle safely, the tool names the file and writes nothing.

### D6. Which files are a user's file

A user's file is one that codegraph edits in place and shares with the user: MCP configuration files, agent configuration files that hold hook entries or permissions, TOML and YAML agent configuration, and instruction files with a marked section. These go through the funnel of D4.

An owned file is one whose whole content codegraph writes: guard scripts, the skill package, the install manifest. These keep their present calls. Take a project whose `.claude` is a link to an outside directory. A local install refuses the user's files under it. It still writes the owned files there, as today.

### D7. Two agents that share one file are found by real location

With `GEMINI.md` linked to `AGENTS.md`, the edit of either path now reaches one file. An uninstall of one agent must keep the section that the other agent still uses.

- `instructionsRequestedElsewhere` compares paths with `sameRealPath`.
- One new helper, `removeSharedInstructions(result, path, loc, self)`, holds the whole step. It calls `instructionsRequestedElsewhere`. If that finds an agent, it calls `recordKept` with `instructionsKeptNote`. Otherwise it routes `removeMarkedSection` through `recordUserAction`.
- Every `Uninstall` that removes a marked section calls the helper: claude, gemini, codex and opencode. Codex and opencode hold a copy of the step today, and the copies go away. Claude and gemini gain the test.

Without a link, no two of these agents share an instruction path with claude or gemini, so their present behavior does not change.

### D8. The parse rule comes from #303

The requirement "An MCP configuration file that does not parse is not written" records what #303 implements for `writeMcpEntry` and `writeOpencodeEntry`. This change adds only the scenario through a link, which D1 and D4 produce. The hook and permission writers read with `readJSONFileStrict`, which has its own rule for an empty file, and the requirement does not cover them. The implementation starts after #303 is on `main`.

## Risks / Trade-offs

- [A local install that succeeded before now fails for a file that links outside the project.]
  → The error names the link, the real location and three ways forward. The changelog fragment states the change.
- [A project whose `.claude` directory is a link to an outside directory has its user files refused.]
  → The same rule and the same error apply. The owned files are still written there.
- [The working directory is the project root only by convention.]
  → Local paths are already relative to it. The check uses the same root as the write.
- [A link changes between the resolve and the write.] → Out of scope, see Non-Goals.
- [`Resolve` is the one place that decides where a write lands, and a fault in it is a write to the wrong file.]
  → Two property tests compare it with the operating system on random link graphs. The existing skill directory tests pass unchanged.
- [The temporary file is created in the directory of the real file. That directory can be read-only.]
  → The error from `os.CreateTemp` names that directory and reaches the user through `WriteResult.Errors`. Nothing is written, and the link stays.
- [An uninstall that empties a linked file leaves a link with no target.] → The next install creates the target through the link.

## Migration Plan

No data migration. A release that carries this change starts to keep links at the next install or refresh. A link that an earlier release already replaced stays a regular file until the user restores it.
