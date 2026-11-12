# Design

## Context

See proposal.md for the motivation.

- `fsatomic.WriteFile(path, content)` makes a temporary file in `filepath.Dir(path)` and renames it over `path`. A rename over a symlink replaces the link.
- `internal/agents` sends every write through `atomicWriteFile` and `atomicWriteExecutableFile` in `shared.go`. Both take a path and content only.
- Eight helpers sit above them: `writeJSONFile`, `writeMcpEntry`, `removeMcpEntry`, `writeHookEntry`, `removeHookEntry`, `replaceOrAppendMarkedSection`, `removeMarkedSection` and `upsertInstructionsEntry`. `codex.go`, `hermes.go` and `opencode.go` also write directly. Together these have 59 call sites outside tests.
- Each target's `Install` and `Uninstall` receive the `Location`. The helpers do not.
- Every read in these helpers is `os.ReadFile`, which follows a symlink. So the read half of each read-modify-write already sees the content of the link target.
- Local paths are relative to the working directory. No helper holds a project root today.
- `internal/githooks` and `internal/daemon` also call `fsatomic.WriteFile`.

## Goals / Non-Goals

**Goals:**

- If the spec says so, a write to a user's own file keeps a symlink in place.
- A write at local scope cannot change a file outside the project through a symlink.
- The safe behavior is the default. A caller that does nothing new keeps today's rename over the path.

**Non-Goals:**

- No protection against a person who changes a link between the check and the rename. That needs write access to the directory while the command runs.
- No change to how a file is read.
- No new output line or action name.

## Decisions

### D1. `fsatomic` gains one new entry point, and `WriteFile` does not change

`fsatomic.WriteFileThroughLink(path, content, within string) error` resolves a symlink at the last element of `path`. It then writes with the same temporary file and rename as `WriteFile`, in the directory of the resolved file.

- `within` is empty at global scope: any target is accepted.
- `within` is a directory at local scope: the target is accepted only inside it. A target outside it makes the function write exactly as `WriteFile` does, over `path` itself.

`WriteFile` keeps its contract, so `internal/githooks`, `internal/daemon` and every owned file in `internal/agents` need no change. The spec requirement "Files that codegraph owns replace a symlink" is then true by default.

Alternative: one `WriteFile` with an options value. Rejected. The zero value must be the safe behavior at every existing call. A second named function also makes each write-through call visible in a search.

### D2. The link is resolved by hand, one step at a time

`filepath.EvalSymlinks` fails on a link whose target does not exist, and the spec requires that case to work. The function therefore loops over these steps:

1. `os.Lstat` the path.
2. If the path does not exist or is not a symlink, stop.
3. `os.Readlink` the path, and join a relative target to the directory of the link.

More than 40 steps is an error that names the first path, and nothing is written. 40 is the limit Linux uses for a path lookup.

Only the last element is resolved this way. Directories above it are left to the operating system, as today.

### D3. "Inside the project" is checked on real paths

For a non-empty `within`, the function compares real locations and not path text. It resolves `within` with `filepath.EvalSymlinks`. For the target, it resolves the longest part of the target's directory that exists and joins the rest back on, then adds the file name. If `filepath.Rel` from the real `within` to that path does not start with `..`, the target is inside.

This is what makes the spec scenario "A link that leaves the project through a second link" hold. The same shape is in `confineToRepoRoot` in `internal/mcp/tools.go`. The two stay separate: one decides a read, the other a write, and neither package imports the other.

### D4. The helpers take a link policy, passed down from the target

`internal/agents` adds a small value type:

```go
// linkPolicy says what a write does with a symlink at its path.
type linkPolicy struct {
	through bool   // false: rename over the path, as fsatomic.WriteFile does
	within  string // with through: accept only a target inside this directory; "" accepts any
}
```

`linkPolicyFor(loc Location) (linkPolicy, error)` returns `{through: true}` for global and `{through: true, within: <working directory>}` for local. The zero value is the present behavior.

The helpers that edit a user's file take the policy as a parameter and pass it to one new leaf, `writeUserFile(path, content string, p linkPolicy) error`. `atomicWriteFile` and `atomicWriteExecutableFile` keep their signatures and stay the writers for owned files.

Alternatives:

- Derive the policy from the path at the leaf: treat a path under the working directory as local. Rejected. A global install that runs with the home directory as the working directory then gets the local rule. The rule is also hidden in one function instead of stated at each call.
- Package state that `Install` sets before it writes. Rejected: two targets write in one run, and hidden state is what the compiler cannot check.

The cost is a signature change at 59 call sites. The compiler finds every one, and no site can be missed without a build failure.

### D5. Which files are a user's file

A user's file is one that codegraph edits in place and shares with the user: MCP configuration files, agent configuration files that hold hook entries or permissions, TOML and YAML agent configuration, and instruction files with a marked section. These go through `writeUserFile`.

An owned file is one whose whole content codegraph writes: guard scripts, the skill package, the install manifest. These stay on `atomicWriteFile` and `atomicWriteExecutableFile`.

### D6. No notice when a local link is replaced

When a local link leads outside the project, the write replaces the link and reports the same action as today. A new output line is a change to an output shape that agents and scripts read. The replaced link is also visible in `git status` in the project. `SECURITY.md` states the rule.

### D7. The refusal from #292 applies to the file that is read

`writeMcpEntry` refuses a file that is not valid JSON. It reads through the link, so the refusal is about the content of the link target, and the error names the path the user configured. No extra work is needed.

### Decisions the proposal already records

"Always follow", "never follow", and "refuse every symlink" are rejected in proposal.md.

## Risks / Trade-offs

- [A local link to a file outside the project is still replaced. That surprises a person who made the link on purpose.] → The outside file is safe, the project file is correct, and `git status` shows the change. `SECURITY.md` explains why.
- [The working directory is the project root only by convention] → Local paths are already relative to it, so the check uses the same root as the write. A person who runs a local install from a subdirectory gets files in that subdirectory today, and the check follows.
- [A link is changed between the resolve and the rename] → Out of scope, see Non-Goals.
- [59 call sites change] → The change is mechanical and compiler-checked. Each target's existing tests run through the new parameter.
- [A write through a link puts the temporary file in the target's directory. That directory can be read-only.] → The error from `os.CreateTemp` names that directory and reaches the user through `WriteResult.Errors`. Nothing is written, and the link stays.

## Migration Plan

No data migration. A release that carries this change starts to keep links at the next install or refresh. A link that an earlier release already replaced stays a regular file until the user restores it.
