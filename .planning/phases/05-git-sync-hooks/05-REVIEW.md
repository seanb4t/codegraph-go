---
phase: 05-git-sync-hooks
reviewed: 2026-07-16T00:00:00Z
depth: deep
files_reviewed: 14
files_reviewed_list:
  - internal/agents/shared.go
  - internal/cli/githooks.go
  - internal/cli/githooks_test.go
  - internal/cli/init.go
  - internal/cli/init_advisory_test.go
  - internal/cli/root.go
  - internal/cli/uninit.go
  - internal/cli/uninit_test.go
  - internal/fsatomic/fsatomic.go
  - internal/fsatomic/fsatomic_test.go
  - internal/githooks/githooks.go
  - internal/githooks/githooks_test.go
  - internal/gitmeta/githooks.go
  - internal/gitmeta/githooks_test.go
findings:
  critical: 1
  warning: 3
  info: 3
  total: 7
status: issues_found
---

# Phase 05: Code Review Report

**Reviewed:** 2026-07-16
**Depth:** deep
**Files Reviewed:** 14
**Status:** issues_found

## Summary

Phase 5 ports TS CodeGraph's `sync/git-hooks.js` into `internal/githooks`,
adds `internal/gitmeta.IsGitRepo`/`HooksDir`, extracts `internal/fsatomic`
from `internal/agents/shared.go`'s prior `atomicWriteFile`, and wires the
result into a new `codegraph githooks install|remove|status` command tree
plus best-effort hooks into `init`'s success path and `uninit`'s cleanup
path. The cross-file wiring checks the phase context asked for all came
back clean:

1. **CLI reaches the real package** — `root.go` registers `newGithooksCmd()`
   and the CLI tests (`githooks_test.go`, `uninit_test.go`,
   `init_advisory_test.go`) drive `execCmd(...)` against the actual
   `newRootCmd()` tree, not a stub — reverting the wiring turns them red.
2. **`init`'s advisory wiring is load-bearing** — `TestInitAdvisory_WatcherDisabled*`
   force the disabled state via the injectable `watch.Probe` env seam and
   assert on `init`'s real stdout; they are not asserting a mirrored
   reimplementation of the logic.
3. **The `fsatomic` extraction is byte-identical** to the prior
   `internal/agents/shared.go` `atomicWriteFile` (confirmed via `git show
   cbc394d`) — no behavior drift from the refactor.
4. **TS-block interop works** — `TestRemove_TSInstalledBlock_DetectedAndRemovable`
   pastes verbatim TS marker bytes and asserts Go's `Status`/`Remove`
   recognize and operate on them.
5. **Concurrency/crash safety** — writes are individually atomic via
   `fsatomic.WriteFile` (temp file + rename), but see WR-02/CR-01 below for
   gaps in the surrounding read-modify-write and strip logic.
6. **Silent-failure paths** — several exist; see CR-01 and WR-01. Some are
   faithful (and explicitly TS-parity-locked) ports of upstream's own
   swallowing behavior; one (CR-01) is a genuine data-loss-capable defect
   present in both TS and this port, and a "must be fixed before ship"
   item independent of parity intent.

The one Critical finding below is inherited byte-for-byte from TS's own
`stripMarkerBlock` (confirmed against the RESEARCH.md transcription) rather
than a regression introduced by this Go port — flagged anyway per this
review's scope (data-loss risk is a Critical-tier criterion regardless of
origin), with the parity context called out so a fix can be scoped
correctly (Go-only divergence vs. an upstream report).

## Critical Issues

### CR-01: Unterminated/malformed marker block silently destroys trailing file content

**File:** `internal/githooks/githooks.go:54-73` (`stripMarkerBlock`), reachable from `Install` (`internal/githooks/githooks.go:145-176`) and `Remove` (`internal/githooks/githooks.go:189-220`)

**Issue:** `stripMarkerBlock` sets `inBlock = true` on the trimmed begin
marker and only clears it on the trimmed end marker:

```go
for _, line := range lines {
    trimmed := strings.TrimSpace(line)
    if trimmed == markerBegin {
        inBlock = true
        continue
    }
    if trimmed == markerEnd {
        inBlock = false
        continue
    }
    if !inBlock {
        kept = append(kept, line)
    }
}
```

If a hook file contains the begin marker but the matching end marker is
missing or precedes it (a plausible real-world state: a user hand-edits the
hook and accidentally deletes/mangles the `# <<< codegraph sync hook <<<`
line, or a prior write is interrupted outside `fsatomic`'s crash-safe path,
e.g. via a text editor), every line from the begin marker to EOF is dropped
— including any of the user's own shell content that happens to sit after
the block. `Remove` (and `Install`'s re-strip-then-append path) then writes
the truncated result back to disk via `fsatomic.WriteFile`, permanently
destroying that content with no warning, no error, and no way to recover it
(the atomic write means there is no crash window to exploit for recovery —
the data is just gone).

**Reproduction:** write a hook file with the begin marker present but no
end marker, with distinguishing content both before and after where the end
marker should be:

```
#!/bin/sh
echo before

# >>> codegraph sync hook >>>
... (block body, end marker line deleted) ...
echo after
echo more-user-content
```

Running `codegraph githooks remove` (or `install`, which also calls
`stripMarkerBlock` on the existing file) against this fixture silently
rewrites the file to `#!/bin/sh\necho before\n`, dropping `echo after` and
`echo more-user-content` with no error returned from `Remove`/`Install` and
no message printed by the CLI layer.

**Context:** this is a byte-for-byte faithful port of TS's own
`stripMarkerBlock` (`sync/git-hooks.js:116-134`, transcribed in
`05-RESEARCH.md`), which has the identical unbounded-`inBlock` behavior — it
is not a regression introduced by the Go port, and D-02/D-03 lock verbatim
TS semantics as a phase requirement. Flagging it as Critical anyway because
data-loss risk is a Critical-tier finding independent of whether the defect
originates upstream; the fix should be scoped deliberately (either as a
documented, intentional Go-only divergence, or filed upstream too) rather
than silently inherited.

**Fix:** guard against an unterminated block before trusting the strip —
e.g. bail out (return content unchanged, or return an explicit error) when
a begin marker is found with no matching end marker after it, rather than
treating "no end marker" as "block extends to EOF":

```go
func stripMarkerBlock(content string) string {
    lines := strings.Split(content, "\n")
    var kept []string
    inBlock := false
    sawUnterminatedBegin := false
    for _, line := range lines {
        trimmed := strings.TrimSpace(line)
        if trimmed == markerBegin {
            inBlock = true
            sawUnterminatedBegin = true
            continue
        }
        if trimmed == markerEnd {
            inBlock = false
            sawUnterminatedBegin = false
            continue
        }
        if !inBlock {
            kept = append(kept, line)
        }
    }
    if sawUnterminatedBegin {
        // No matching end marker — do not trust the strip; leave content
        // untouched rather than risk destroying everything after a
        // malformed/partial block.
        return content
    }
    return strings.Join(kept, "\n")
}
```
(Callers of `Install`/`Remove` should then treat this case explicitly —
e.g. `Remove` skipping the hook rather than reporting it as removed.)

## Warnings

### WR-01: Install/Remove silently swallow per-hook write/delete errors with no diagnostic

**File:** `internal/githooks/githooks.go:169-171` (`Install`), `internal/githooks/githooks.go:198-215` (`Remove`)

**Issue:** Both loops discard the underlying error entirely on a per-hook
write/delete failure:

```go
if err := fsatomic.WriteFile(file, content); err != nil {
    continue
}
```

```go
if err := os.Remove(file); err != nil {
    continue
}
...
if err := fsatomic.WriteFile(file, content); err != nil {
    continue
}
```

If, say, `post-checkout` is unwritable (wrong ownership, read-only mount,
disk full) while `post-commit`/`post-merge` succeed, `InstallResult.Installed`
silently comes back with only 2 of 3 hooks and neither `Install` nor the CLI
(`internal/cli/githooks.go`'s `newGithooksInstallCmd`) has any way to tell
the user *why* — there's no error, no log line, nothing. TS's own
`installGitSyncHook` doesn't have this gap: `fs.writeFileSync`'s exception
is uncaught within the function and propagates out (confirmed in
`05-RESEARCH.md`'s verbatim transcription), so a write failure in TS is loud
(throws), not silent. The RESEARCH.md illustrative Go snippet explicitly
flagged this as "Claude's discretion on result shape" (accumulate vs.
discard) — the discretion was exercised in the direction that loses the
most information.

**Fix:** at minimum, accumulate the errors so the caller/CLI can report them
even if the per-hook loop still continues past a failure:

```go
type InstallResult struct {
    Installed []string
    HooksDir  string
    Skipped   string
    Errors    []error // new: one entry per hook that failed to write
}
...
if err := fsatomic.WriteFile(file, content); err != nil {
    result.Errors = append(result.Errors, fmt.Errorf("%s: %w", hook, err))
    continue
}
```
and have `newGithooksInstallCmd`/`newGithooksRemoveCmd` print a line per
error (or at least a "N of 3 hooks could not be written" summary) instead of
only ever describing the hooks that succeeded.

### WR-02: No synchronization across the read-modify-write sequence in Install/Remove

**File:** `internal/githooks/githooks.go:145-176` (`Install`), `internal/githooks/githooks.go:189-220` (`Remove`)

**Issue:** `fsatomic.WriteFile` guarantees an individual write is atomic and
crash-safe (temp file + rename), but the surrounding operation —
`os.ReadFile` the current hook, compute a new body from it, then
`fsatomic.WriteFile` it back — is not atomic as a whole. Two concurrent
invocations against the same hooks directory (two `codegraph githooks
install` runs, or an `install` racing a `remove`, or `init`'s advisory path
racing an explicit `githooks install`) can both read the same "before"
state, and whichever writes last silently discards the other's update (lost
update), with neither process aware anything raced. There is no file lock
or CAS-style guard anywhere in this path.

**Fix:** low-priority given this is a rarely-concurrent CLI operation, but
worth at least documenting the constraint (e.g. a doc comment on `Install`/
`Remove` noting they are not safe to call concurrently against the same
`projectRoot`), or take an `flock`/lockfile around the hooks-dir mutation if
concurrent invocation is a realistic scenario for this project (e.g. CI
running `codegraph init` and a user's own tooling running `codegraph
githooks install` at the same time).

### WR-03: Missing test coverage for install/remove failure and partial-failure paths

**File:** `internal/githooks/githooks_test.go`, `internal/cli/githooks_test.go`

**Issue:** No test in either package exercises:
- `Install`'s `"could not access the git hooks directory"` skip branch
  (`internal/githooks/githooks.go:150-152`).
- The CLI's `"Could not install git hooks. Run \`codegraph sync\` after
  changes instead."` fallback message (`internal/cli/githooks.go:44`) —
  only ever reached when `Install` returns zero installed hooks without
  being `Skipped`.
- A partial-success `Install`/`Remove` (some hooks succeed, one or more
  fail) — the exact scenario WR-01 is about.
- The CLI's `"No git sync hooks were installed — nothing to remove."`
  message (`internal/cli/githooks.go:72`).

These are all reachable, user-facing code paths (not dead code), and the
silent-failure behavior in WR-01 means they're also the paths most likely
to hide a real bug if one is introduced later.

**Fix:** add a fixture where the hooks directory (or one hook file within
it) is made unwritable (e.g. `os.Chmod(hooksDir, 0o500)` on a POSIX CI
runner, skipped under `t.Skip` when running as root or on Windows) and
assert on the partial/failure result shape and CLI message.

## Info

### IN-01: Duplicate pluralization logic between uninit.go and githooks.go

**File:** `internal/cli/githooks.go:75-78`, `internal/cli/uninit.go:91-98`

**Issue:** `cli/uninit.go` defines a shared `plural(n int) string` helper
used for the "Removed git ... sync hook(s)" message. `cli/githooks.go`'s
`newGithooksRemoveCmd` reimplements the identical "s"/"" logic inline
instead of calling it:

```go
suffix := "s"
if len(result.Removed) == 1 {
    suffix = ""
}
```

**Fix:** replace with `plural(len(result.Removed))` (both are in package
`cli`, so no import is needed).

### IN-02: Watcher-enabled advisory test depends on ambient environment rather than asserting it

**File:** `internal/cli/init_advisory_test.go:15-25`

**Issue:** `TestInitAdvisory_WatcherEnabled` asserts no advisory prints when
the watcher runs normally, but never explicitly unsets `CODEGRAPH_NO_WATCH`
or otherwise controls for `watch.DetectWSL()`/`/mnt` state — it relies on
the ambient test/CI environment happening not to have the env var set and
not running on a WSL2 `/mnt` drive. A stray exported `CODEGRAPH_NO_WATCH=1`
in a developer's shell (or a future WSL2 CI runner) would silently flip
what this test is actually asserting without the test itself failing loudly
in an obviously-attributable way.

**Fix:** `t.Setenv("CODEGRAPH_NO_WATCH", "")` (or equivalent) at the top of
the test to make the "watcher enabled" precondition explicit rather than
ambient.

### IN-03: `githooks status`/TS's `isSyncHookInstalled` report "installed" based on marker text only, not executability

**File:** `internal/githooks/githooks.go:228-244` (`Status`)

**Issue:** `Status` (and, per the RESEARCH.md transcription, TS's
`isSyncHookInstalled`) only checks whether the hook file exists and
contains `markerBegin` — it never checks the file's exec bit. Because
`fsatomic.WriteFile`'s atomic rename and the subsequent best-effort
`os.Chmod(file, 0o755)` (`internal/githooks/githooks.go:172,215`) are two
separate, non-atomic steps, a process crash between them (or any external
`chmod -x` on the hook afterward) leaves a hook file that `githooks status`
reports as `installed: true` even though git will never actually execute it
(no `+x`). This is inherited TS-parity behavior, not a Go-introduced
regression, but is worth documenting as a known limitation since `status`
is the primary way a user would sanity-check hook health.

**Fix:** optional — `Status` could additionally check
`info.Mode().Perm()&0o111 != 0` and report a distinct state (e.g.
"installed but not executable") rather than folding it into a flat
boolean. Not required for TS parity; purely a robustness improvement if the
team wants `status` to be a more trustworthy health check than TS's.

---

_Reviewed: 2026-07-16_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep_
