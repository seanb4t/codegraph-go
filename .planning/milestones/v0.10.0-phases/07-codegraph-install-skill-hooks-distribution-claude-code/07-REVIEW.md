---
phase: 07-codegraph-install-skill-hooks-distribution-claude-code
reviewed: 2026-08-13T00:00:00Z
depth: deep
files_reviewed: 10
files_reviewed_list:
  - claudeassets.go
  - internal/agents/claude.go
  - internal/agents/claude_readerror_test.go
  - internal/agents/claude_skillpackage_test.go
  - internal/agents/claude_test.go
  - internal/agents/manifest.go
  - internal/agents/manifest_test.go
  - internal/agents/shared.go
  - internal/cli/upgrade.go
  - internal/cli/upgrade_test.go
findings:
  critical: 2
  warning: 4
  info: 0
  total: 6
status: issues_found
---

# Phase 7: Code Review Report

**Reviewed:** 2026-08-13
**Depth:** deep
**Files Reviewed:** 10
**Status:** issues_found

## Summary

This phase adds a go:embed'd Claude Code skill package (SKILL.md, session-nudge.sh,
SessionStart hooks registration) with a sidecar manifest, distributed via
`codegraph install`/`upgrade`. The hook-ownership vulnerability flagged in the
task brief (matcher-and-shape recovery heuristic) was cleanly reverted in
242ec0a — no residual shape/matcher-based logic remains in either
`writeHookEntry` or `removeHookEntry`, both of which determine ownership
solely by exact command-string match, and `TestClaude_Install_NeverClaimsOwnershipOfUnrelatedHookUnderSameMatcher`
correctly pins this. The `go:embed` boundary in `claudeassets.go` is scoped
to exactly the three intended files via named `//go:embed` directives (not a
directory pattern), and `readJSONFileStrict` is used consistently at every
settings.json call site (`addClaudeAllowPermission`, `removeClaudeAllowPermission`,
`writeHookEntry`, `removeHookEntry`) — the permissive `readJSONFile` fallback
is correctly confined to files this package does not co-own with the hooks
step (MCP config paths).

However, deep tracing of `claudeTarget.Install`'s manifest-gating logic and
`writeEmbeddedFile`'s idempotency short-circuit surfaced two BLOCKER-level
correctness bugs that directly contradict the code's own stated design
invariants (D-05 self-healing, and the "only proceeds if all three artifacts'
content resolved" manifest guard), plus four lower-severity issues around
manifest self-healing asymmetry, duplicate status-line reporting, an
unenforced single-command assumption, and a silent-skip path in
`ConfiguredSkillLocations`.

## Critical Issues

### CR-01: Manifest is written on partial failure — "have*" flags gate on content *resolution*, not write *success*

**File:** `internal/agents/claude.go:417-493`
**Issue:**
The comment directly above this block states the manifest write is gated so
that "a manifest recording a hash for content that was never actually
available would be worse than no manifest at all." The implementation does
not actually enforce that: `haveSkillMDContent`, `haveScriptContent`, and
`haveSessionStart` are each set to `true` immediately after the embedded
content is successfully *read* from `claudeassets`/decoded from the fragment
— **before** the corresponding disk write (`writeEmbeddedFile` /
`writeHookEntry`) is even attempted:

```go
skillMDContent = content
haveSkillMDContent = true
fr, werr := writeEmbeddedFile(skillFilePath, string(content), false)
recordFile(&result, skillFilePath, fr, werr)   // werr never gates haveSkillMDContent
```

The same pattern repeats for `scriptContent`/`haveScriptContent` and
`sessionStartBlocks`/`haveSessionStart`. If exactly one of the three writes
fails (disk full, EACCES creating `~/.claude/hooks/`, a malformed
`settings.json` that makes `writeHookEntry` refuse to touch the file, etc.)
while content resolution for all three succeeded, the manifest gate
`haveSkillMDContent && haveScriptContent && haveSessionStart` is still
`true`. The manifest step then runs and writes a `.codegraph-manifest.json`
recording `hashContent(skillMDContent)`, `hashContent(scriptContent)`, and
`hashOwnedHookBlocks(sessionStartBlocks)` for the *intended* content — even
for the one artifact whose write just failed and is therefore stale or
missing on disk.

`WriteResult.Errors` will still contain the underlying write error (so the
CLI's immediate exit code/output is correct), but the persisted manifest
file will falsely assert full success. Any future/consuming code that
trusts the manifest as "what's actually installed" (the exact use case the
package doc for `manifest.go` describes: "which binary version wrote the
installed skill package," a future `install --status` surface per
`manifest.go:45`) will be misled after a partial-failure install.

**Fix:** Gate the manifest write on write success, not content resolution —
e.g. track `skillMDWritten`, `scriptWritten`, `hooksWritten` set only when
`werr == nil`:

```go
fr, werr := writeEmbeddedFile(skillFilePath, string(content), false)
recordFile(&result, skillFilePath, fr, werr)
if werr == nil {
    skillMDContent = content
    haveSkillMDContent = true
}
```
and analogously for the script and hooks steps.

### CR-02: Script's executable bit is never self-healed once content matches — `writeEmbeddedFile`'s idempotency check ignores file mode

**File:** `internal/agents/shared.go:292-319` (called from `internal/agents/claude.go:449`)
**Issue:**
`writeEmbeddedFile`'s no-op short-circuit compares only file *bytes*:

```go
if existed {
    current, err := os.ReadFile(path)
    ...
    if string(current) == content {
        return FileResult{Path: path, Action: ActionUnchanged}, nil
    }
}
```

For `session-nudge.sh` (`executable=true`), this is the only path that would
trigger `atomicWriteExecutableFile`'s `os.Chmod(path, 0o755)` restoring the
executable bit. If the script's *content* is unchanged but its *mode* has
drifted (a user runs `chmod -x` on it, a backup/restore tool resets
permissions, an antivirus quarantine-and-restore cycle strips the bit,
etc.), the byte comparison still matches, so the function returns
`ActionUnchanged` and **never calls chmod**. Because the embedded content
never changes except across a binary upgrade, this is not a transient
edge case — once the executable bit is lost while content stays pinned to
the embedded version, no subsequent `codegraph install` or `codegraph
upgrade` (which reuses this same code path via `refreshInstalledSkills`)
can ever restore it. The SessionStart hook Claude Code invokes then
silently fails to execute, with no error surfaced anywhere in this
project's own tooling — directly contradicting D-05's "hand-edited content
is silently restored" self-healing invariant, which the test suite only
verifies for *content* drift (`TestClaude_Install_SilentlyOverwritesHandEditedSkill`,
`TestClaude_Install_HandEditedHookBlockDuplicatesRatherThanOverwritesUnrelated`)
and never for *mode* drift.

**Fix:** When `executed && executable`, also check the current file's mode
and force a (re-)write/chmod if the executable bits are missing, even when
content bytes match:

```go
if existed {
    current, err := os.ReadFile(path)
    ...
    info, statErr := os.Stat(path)
    modeOK := statErr == nil && (!executable || info.Mode()&0o111 != 0)
    if string(current) == content && modeOK {
        return FileResult{Path: path, Action: ActionUnchanged}, nil
    }
}
```

## Warnings

### WR-01: A corrupted/unreadable manifest permanently blocks `writeManifest` — no self-heal, unlike SKILL.md/script

**File:** `internal/agents/manifest.go:127-153` (`writeManifest`), `internal/agents/manifest.go:90-103` (`readManifest`)
**Issue:** `writeManifest` calls `readManifest` first and returns its error
unwritten if the existing manifest can't be parsed:

```go
existing, existedBefore, err := readManifest(path)
if err != nil {
    return FileResult{}, err
}
```

Unlike `settings.json` (a file this package only partially owns, where
refusing to touch malformed content is the *correct*, deliberate posture to
protect user data), `.codegraph-manifest.json` is a wholly codegraph-owned,
dot-prefixed internal sidecar file with no user content to protect — the
same category as `SKILL.md`/`session-nudge.sh`, both of which are silently
overwritten on drift per D-05. If the manifest file is ever corrupted
(partial write despite atomicity elsewhere, disk corruption, an errant
manual edit), every future `codegraph install`/`upgrade` run will
permanently fail the manifest step (recorded as an `Errors` entry, making
the CLI report a partial failure) even though SKILL.md, the script, and the
hooks registration all successfully self-heal in the same run. There is no
code path that ever deletes/overwrites a malformed manifest — only manual
user intervention (deleting the file) recovers.

**Fix:** Treat an unreadable/undecodable manifest the same way `writeEmbeddedFile`
treats drifted SKILL.md content — proceed to overwrite it rather than
erroring, since no third-party content is at risk in this file.

### WR-02: `settings.json` is reported twice in `WriteResult.Files` for a single `--auto-allow` install, producing duplicate CLI status lines

**File:** `internal/agents/claude.go:396-403` (AutoAllow step) and `internal/agents/claude.go:454-466` (hooks step)
**Issue:** `Install()` now touches `claudeSettingsPath(loc)` via two
independent read-modify-write cycles in the same call:
`addClaudeAllowPermission` (when `opts.AutoAllow`) and `writeHookEntry`
(unconditionally). Each funnels through `recordFile`, so
`result.Files` ends up with two separate `FileResult` entries for the same
path with potentially different `Action` values. `printAgentResults` (the
CLI's per-file reporting loop) prints one line per `FileResult`, so a
`codegraph install --auto-allow` run prints `~/.claude/settings.json`
twice under one target's output — confusing, and `installStatus`'s
"unchanged only if every file action is Unchanged" summary works correctly
by coincidence rather than by design (it iterates all entries regardless of
path, so duplicates don't corrupt the *summary word*, but do corrupt the
per-file listing a user reads underneath it).
**Fix:** Either merge the two settings.json read-modify-write steps into a
single pass before the one write, or dedupe/collapse `WriteResult.Files`
entries by path before display (keeping the "most severe" action) in
`printAgentResults`.

### WR-03: Ownership identity assumes every embedded SessionStart block shares exactly one command string — unenforced invariant

**File:** `internal/agents/claude.go:207-260` (`claudeSessionStartBlocks`)
**Issue:** `claudeSessionStartBlocks` derives a single `ownCommand` and
returns `[]string{ownCommand}` as the *only* identity `writeHookEntry`/
`removeHookEntry` use to recognize codegraph's own blocks. The rewrite loop
only replaces a hook's `command` field when it exactly equals
`claudeFragmentCommand`. This is currently safe because
`.claude/hooks/hooks.json` happens to use the identical literal command
string in both its `startup` and `resume` blocks (verified by inspection).
But nothing in this file, `shared.go`, or the test suite enforces that
invariant going forward — if a future edit to `.claude/hooks/hooks.json`
adds a block with a *different* command (e.g. a second script, or a
different invocation form for one event), `claudeSessionStartBlocks` would
silently fail to rewrite that block's command for non-local scope (it
stays as the literal `${CLAUDE_PROJECT_DIR}/...` string, wrong for global
installs) and `ownCommands` would not include it, so `writeHookEntry`/
`removeHookEntry` would treat that block as *unowned* on every future
install/uninstall — producing silent duplication (write side) or an
un-removable orphan block (uninstall side) with no test or compile-time
signal that anything broke.
**Fix:** Either assert-fail (return an error) in `claudeSessionStartBlocks`
if it encounters a `command` value other than `claudeFragmentCommand`
inside `hooks.SessionStart`, or collect the full set of *rewritten*
commands into `ownCommands` rather than assuming a single value — whichever
is chosen, add a test that would fail if `.claude/hooks/hooks.json` ever
introduces a second distinct command.

### WR-04: `ConfiguredSkillLocations` silently drops a location whose manifest is unreadable, so `codegraph upgrade`'s refresh silently skips it

**File:** `internal/agents/manifest.go:162-179`
**Issue:**
```go
_, present, err := readManifest(path)
if err != nil || !present {
    continue
}
```
A manifest that exists but fails to parse (`err != nil`) is treated
identically to "never configured" (`!present`) — both cause the location to
be silently excluded from the returned list. Since `refreshInstalledSkills`
(`internal/cli/upgrade.go:47-72`) uses this function as its *sole* source of
truth for which locations to re-`Install()` after a binary swap, a
corrupted manifest — itself proof the user previously ran `install` at that
location — means `codegraph upgrade` will silently fail to refresh that
location's skill package, with no warning printed (the CLI only warns when
`refreshInstalledSkillsFunc` itself returns an error; a silently-shrunk
`locs` slice never produces one). Combined with WR-01 (no self-heal for a
corrupted manifest), a user in this state gets no signal at all that their
skill package has stopped being kept in sync with upgrades.
**Fix:** Distinguish "genuinely absent" from "present but corrupted" in the
return value (or log/return the corrupted-location set separately) so
`refreshInstalledSkillsFunc` can at least warn the user their previously
configured location needs a manual `codegraph install`.

---

_Reviewed: 2026-08-13_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep_
