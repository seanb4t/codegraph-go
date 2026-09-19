---
phase: 06-claude-code-pretooluse-nudge
reviewed: 2026-09-19T11:11:55Z
depth: deep
files_reviewed: 34
files_reviewed_list:
  - .claude/hooks/hooks.json
  - .claude/hooks/pretooluse-nudge.sh
  - .claude/settings.json
  - claudeassets.go
  - docs/CLI-REFERENCE.md
  - internal/agents/capabilities.go
  - internal/agents/capabilities_test.go
  - internal/agents/claude.go
  - internal/agents/claude_pretooluse.go
  - internal/agents/claude_pretooluse_lifecycle_test.go
  - internal/agents/claude_pretooluse_test.go
  - internal/agents/claude_skillpackage_test.go
  - internal/agents/hookpackage_test.go
  - internal/agents/manifest.go
  - internal/agents/ownership_test.go
  - internal/agents/skillshared.go
  - internal/agents/types.go
  - internal/cli/hook_pretooluse.go
  - internal/cli/hook_pretooluse_test.go
  - internal/cli/install.go
  - internal/cli/install_test.go
  - internal/cli/root.go
  - internal/cli/testdata/cli-reference-allowlist.txt
  - internal/cli/testdata/plain/uninstall-local.golden
  - internal/cli/uninstall.go
  - internal/cli/upgrade.go
  - internal/cli/upgrade_test.go
  - internal/mcp/skill_claims_drift_test.go
  - internal/nudge/classify.go
  - internal/nudge/classify_test.go
  - internal/nudge/cooldown.go
  - internal/nudge/cooldown_test.go
  - internal/nudge/testdata/false-positives.json
  - internal/nudge/testdata/true-positives.json
  - internal/nudge/text.go
findings:
  critical: 1
  warning: 1
  info: 1
  total: 3
status: issues_found
---

# Phase 06: Code Review Report

**Reviewed:** 2026-09-19T11:11:55Z
**Depth:** deep
**Files Reviewed:** 34
**Status:** issues_found

## Summary

This phase adds the Claude Code `PreToolUse` nudge end to end: the harness-neutral
`internal/nudge` core (classification + cooldown gate), the hidden `codegraph hook
pretooluse` adapter, the embedded/rendered guard script, and the Claude-only
sticky opt-in lifecycle in `internal/agents/claude.go`/`claude_pretooluse.go`. The
implementation is unusually well defended at the layers the context flags as
highest-risk: the exit-0 guarantee is enforced twice (cobra-level `recover()` plus
the shell guard's own unconditional trailing `exit 0`), the `ExecPath` is rendered
into the guard through correct POSIX single-quoting (verified against paths
containing both a space and a quote), the cooldown sentinel is written through an
`O_NOFOLLOW`-guarded descriptor with an owner check on both the directory and the
file, and the ownership-by-exact-command-string discipline from commit `242ec0a`
is exercised by a 32-leaf table plus a dedicated hand-edit/duplicate test.

One genuine, unguarded correctness defect was found in the sticky opt-in's
interaction with the pre-existing foreign-skill-directory protection (D-14): when
Claude's skill directory is a symlinked shared directory that already holds
unmanifested content, an `install --pretool-nudge` writes the guard script and the
`hooks.PreToolUse` registration to disk successfully, reports no error, but never
records the opt-in — silently breaking the "sticky until explicitly turned off"
promise (D-10) and the `upgrade` guard-refresh path. No existing test (including
the ownership table and the symlink-specific test file) exercises this exact
combination, which is why it survived to this review. A smaller defense-in-depth
gap in the cooldown gate's directory-permission check is also noted, along with
one accepted-by-design limitation recorded for completeness.

## Critical Issues

### CR-01: PreToolUse opt-in silently unrecorded when Claude's skill directory is a foreign, unmanifested symlinked directory

**File:** `internal/agents/claude.go:592-688`
**Issue:**

`claudeTarget.Install` gates the *entire* manifest write — including the two new
`manifestKeyPreToolGuard`/`manifestKeyPreToolFrag` keys that make the opt-in
sticky (D-10) — on the base skill-package trio succeeding first:

```go
if claudeSkillDir != "" && haveSkillMDContent && haveScriptContent && haveSessionStart {
    ...
    if havePreTool {
        ownFiles[manifestKeyPreToolGuard] = hashContent(preToolGuardContent)
        ownFiles[manifestKeyPreToolFrag] = fragHash
    }
    ...
    recordSkillManifestWithFallback(&result, claudeSkillDir, loc, Claude, ownFiles, []TargetID{Claude}, dropKeys)
}
```

But the PreToolUse guard write and its `hooks.PreToolUse` registration
(`claude.go:598-650`) are **entirely independent** of `claudeSkillDir`/
`haveSkillMDContent` — they live at `.claude/hooks/pretooluse-nudge.sh` and inside
`settings.json`, not inside the skill directory. `havePreTool` is set purely from
those two writes succeeding.

`haveSkillMDContent` is false whenever `writeSkillFile` reports
`ActionKeptForeign` — which happens under `refuseUnmanifested` policy (returned by
`claudeSkillPolicy` exactly when Claude's skill directory is symlinked onto the
shared `.agents/skills/codegraph` directory, per D-17/the `npx skills` convention
this repository's own `CLAUDE.md` documents as the maintainer's own layout) when
that shared directory already holds content with no codegraph manifest.

`internal/agents/claude_symlink_test.go`'s
`TestSymlinkedSkillDir_ForeignContentKeptForeign` proves exactly this precondition
produces `ActionKeptForeign` and *asserts that no manifest is created anywhere*
(`len(manifestPaths) != 0` fails the test) — but that test never sets
`PreToolNudge: PreToolNudgeOn`, so it never exercises the interaction. Every test
in this phase that *does* set `PreToolNudge: PreToolNudgeOn` (`ownership_test.go`,
`claude_pretooluse_lifecycle_test.go`'s
`TestPreToolNudge_ClaudeUninstallDropsKeysFromSharedManifest`,
`capabilities_test.go`'s `TestCapabilitiesMatchInstallWrites`) runs against a fresh
`fakeHome()` where the shared directory is empty (not foreign) before install, so
none of them hit `ActionKeptForeign`. The combination — symlinked layout **and**
pre-existing unmanifested content **and** `--pretool-nudge` — has no test
coverage anywhere in the suite.

Consequences once triggered:
1. `Install` returns **zero errors** and reports the guard as `created` and
   settings.json as `updated` — indistinguishable from a fully successful,
   durably-recorded opt-in.
2. `preToolNudgeRecorded(loc)` subsequently returns `(false, true)` — genuinely
   "readable, not opted in" — because no manifest was ever written at that
   location.
3. The next plain `codegraph install` (or any `install` with `PreToolNudge:
   PreToolNudgeKeep`, the zero value) will neither refresh nor remove the
   now-orphaned guard/registration, since `enablePreTool` requires
   `preToolRecorded`.
4. `codegraph upgrade`'s `refreshInstalledSkills` (`internal/cli/upgrade.go:53-57`)
   drives entirely off `agents.ConfiguredSkillLocations(agents.Claude)`, which
   walks manifest presence — this location is invisible to it, so the guard's
   baked-in absolute `ExecPath` is never re-rendered after a binary move/replace.
   The guard's own `[ ! -f "$codegraph_bin" ] || [ ! -x "$codegraph_bin" ]` check
   then silently exits 0 forever with no diagnostic anywhere (by design, per
   D-08's "any failure is silent" — but here applied to a location the user
   legitimately opted into).
5. The only way to definitively clean up the orphan is `codegraph uninstall`
   (D-11's "always attempted" unconditional removal) or a manual
   `--pretool-nudge=false`, neither of which the user has any signal they need
   to run, since nothing reports the opt-in as unrecorded.

This is a real, silent divergence between what the user asked for (a sticky,
upgrade-surviving opt-in) and what actually persists, with no error or warning
surfaced anywhere in the CLI output.

**Fix:**

The root cause is that D-10's sticky record and D-14's foreign-directory
protection share one storage location (the skill directory's manifest) for two
independent concerns (skill-package ownership vs. a hooks/guard opt-in that lives
entirely outside that directory). Two directions resolve it without reintroducing
242ec0a's differential:

1. **Decouple the PreToolUse record from the skill-package manifest gate.** Record
   `manifestKeyPreToolGuard`/`manifestKeyPreToolFrag` in their own
   `recordSkillManifestWithFallback` call keyed only on `havePreTool`, independent
   of `haveSkillMDContent`/`haveScriptContent`/`haveSessionStart`. This still
   writes into `claudeSkillDir`'s manifest file, so it needs its own foreign-dir
   guard consistent with D-14 (e.g., skip recording, and note in `result.Notes`,
   when `skillPolicy == refuseUnmanifested && haveSkillMDContent == false`) rather
   than silently proceeding as if nothing happened.
2. **Or store the sticky bit somewhere that can never be foreign**, e.g. a small
   marker colocated with the guard script itself
   (`.claude/hooks/.pretooluse-nudge-manifest.json` or a Files-map entry the
   caller reads directly from `claudeHooksScriptPath`'s directory instead of
   `claudeSkillDirPath`'s), since that directory is unambiguously codegraph-owned
   regardless of the skill package's foreign/adopted status.

Either way, add a regression test combining `symlinkedClaudeLayout(t)` +
pre-existing foreign content in the shared directory (as
`TestSymlinkedSkillDir_ForeignContentKeptForeign` already sets up) with
`PreToolNudge: PreToolNudgeOn`, asserting that the guard/registration are either
(a) recorded and therefore refreshed by a subsequent `Keep` install, or (b) not
written at all and reported to the user — but never "written and silently
forgotten," which is the current behavior.

## Warnings

### WR-01: Cooldown gate does not re-verify an existing sentinel directory's permission bits

**File:** `internal/nudge/cooldown.go:66-90`
**Issue:**

`Gate.Due` creates the sentinel directory with `os.Mkdir(g.Dir, 0o700)`, but when
the directory already exists (`errors.Is(err, fs.ErrExist)`), the only checks
applied are `Lstat`-based: not-a-symlink, `IsDir()`, and owned by the current uid
(`ownedByCurrentUser`). The directory's actual mode bits are never inspected. If
`os.TempDir()/codegraph-nudge-<uid>` already exists with looser permissions than
0700 (e.g. it was created by an older binary before this Gate existed, or a
misconfigured `umask` widened it, or something else with the same uid relaxed it),
any other local process running as a *different* uid on a shared multi-user
machine can list, create, or delete entries inside it — a same-uid check does not
protect against that, since ordinary directory permission bits (not just
ownership) govern who can write into a directory.

The comment on `DefaultDir`/D-08 states the invariant as "created mode 0700," but
nothing enforces that invariant is still true for a pre-existing directory before
trusting it.

**Fix:** After the `Lstat` ownership/symlink/is-dir checks, also reject (return
`false`) when `info.Mode().Perm() != 0o700`, mirroring `TestGate_DirCreated0700`'s
own assertion so the runtime enforces exactly what that test pins:

```go
info, err := os.Lstat(g.Dir)
if err != nil || info.Mode()&fs.ModeSymlink != 0 || !info.IsDir() ||
    info.Mode().Perm() != 0o700 || !ownedByCurrentUser(info) {
    return false
}
```

The impact is availability-only (a worst case is the nudge going silent or firing
early on a compromised shared-tmp machine, never privilege escalation, since
sentinel files carry no session content — `TestGate_SentinelHoldsNoSessionContent`
already guards that), but it is a straightforward, low-cost hardening that matches
the stated invariant exactly.

## Info

### IN-01: Shell classifier's whitespace-only tokenizer misses glued shell metacharacters (accepted false-negative, noted for completeness)

**File:** `internal/nudge/classify.go:69-81`
**Issue:** `shellQualifies` uses `strings.Fields`, which splits only on
whitespace. A compound command with no space around a shell operator — e.g.
`"true||grep x"` or `"A=1;grep x"` — is tokenized as a single word
(`"true||grep"`, `"A=1;grep"`) rather than being split into separate commands the
way a real POSIX shell would. This under-classifies (the nudge stays silent) but
cannot over-classify, since the fused token never matches the exact
`grep`/`egrep`/`fgrep`/`rg`/`find` map lookup. D-02 explicitly accepts "any parse
doubt stays silent" as the design's bias, so this is not a defect against the
documented contract — flagged only so the limitation is recorded rather than
rediscovered later as a surprise gap in the corpus.
**Fix:** None required under the current design. If tighter recall is ever
wanted, a minimal shell-operator-aware pre-split (on `;`, `|`, `&&`, `||`, `&`
outside quotes) before `strings.Fields` would catch these without materially
increasing complexity.

---

_Reviewed: 2026-09-19T11:11:55Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep_
