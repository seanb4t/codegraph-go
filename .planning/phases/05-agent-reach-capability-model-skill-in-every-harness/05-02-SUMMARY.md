---
phase: 05-agent-reach-capability-model-skill-in-every-harness
plan: 02
subsystem: agents
tags: [skill-package, manifest, symlink-safety, ownership, agent-config]

# Dependency graph
requires:
  - phase: 05-01
    provides: "AgentTarget.Capabilities() table (Scopes/MCPConfig/Instructions/SkillDirs/Hooks/ConfigFormat), PathFunc/PathsFunc, skillFileName/skillManifestFileName"
provides:
  - "skillManifest.Targets []TargetID (json:\"targets,omitempty\"), manifestSchemaVersion bumped 1->2, manifestRequesters (D-07 planner amendment: unreadable/nil-Targets manifest reads as [claude]), targetSetEqual"
  - "internal/agents/skillshared.go: ActionKeptForeign, unmanifestedPolicy (refuseUnmanifested/adoptUnmanifested), sharedSkillDirPath/sharedSkillDirs, skillManifestPath, skillDirIsForeign, writeSkillFile, recordSkillManifest, installSkillPackage, uninstallSkillPackage — the single manifest-owned skill package writer every later plan wires targets onto"
  - "resolveSkillDir/sameSkillDir: bounded recursive symlink resolution (live, dangling, and non-existent paths); removeSkillDirIfEmpty gains an Lstat symlink guard"
affects: [05-03, 05-04, 05-05]

# Actuals (#2632)
actuals:
  tokens: 15549
  tasks: 2
  commits: 4

tech-stack:
  added: []
  patterns:
    - "Manifest-owned writer with a requester set as the ownership identity: a single-requester directory is the one-element case of the same shape a multi-requester shared directory uses — one code path (installSkillPackage/uninstallSkillPackage), never two"
    - "Read the manifest fresh on every call, never memoize 'already wrote this run' — required for a multi-target install run to correctly accumulate every requester into targets"
    - "Symlink resolution as a bounded, three-branch recursion (live link via EvalSymlinks, dangling link via Readlink+recurse, non-existent path via parent-then-join) rather than a single library call, since no stdlib function resolves a dangling symlink's target path"

key-files:
  created:
    - internal/agents/skillshared.go
    - internal/agents/skillshared_test.go
  modified:
    - internal/agents/manifest.go
    - internal/agents/manifest_test.go
    - internal/agents/shared.go

key-decisions:
  - "manifestRequesters reads BOTH an unreadable (corrupt) manifest and a present-but-Targets-nil manifest as owned solely by [claude] — never as 'unknown requester set' — per the plan's D-07 planner amendment: Claude's installer was the only writer of any manifest before this phase, so any other reading (e.g. treating it as ownerless) would let a later uninstall delete a package Claude still legitimately owns"
  - "The dangling-symlink MkdirAll pre-creation in writeSkillFile was deliberately deferred from Task 1 to Task 2's GREEN commit (it depends on resolveSkillDir, a Task 2 deliverable) — Task 1's writeSkillFile has no symlink awareness at all, matching the plan's own task boundary"
  - "manifestSchemaVersion bumped 1->2 in this plan's GREEN commit (flagged costly in 05-02-PLAN.md): a released binary older than this change will read a schema-2 manifest and, if it writes one back, drop Targets at schema 1 — self-healed on the next new-binary install via manifestRequesters' nil-Targets-as-[claude] rule, with the loss mode confined to D-17's symlinked layouts. Recorded here for the maintainer per the plan's <output> instruction."

requirements-completed: [AGENT-09, AGENT-13]

coverage:
  - id: D1
    description: "skillManifest.Targets (schema 2) added additively; writeManifest's unchanged check additionally requires targetSetEqual(existing.Targets, m.Targets) so any install order re-run is a byte-level no-op; a manifest with nil Targets serializes with no targets key"
    requirement: "AGENT-09"
    verification:
      - kind: unit
        ref: "internal/agents/manifest_test.go#TestManifest_TargetsRoundTripAndSetEquality"
        status: pass
    human_judgment: false
  - id: D2
    description: "installSkillPackage/uninstallSkillPackage: one manifest-owned writer covering D-05 (write once, accumulate targets), D-07 (legacy/corrupt manifest read as [claude]), D-08 (last-requester deletes, non-requester is a no-op, exclusive keys dropped on departure), D-14 (foreign dir kept untouched, ownership = manifest presence only), D-16 (hand-edited own file rewritten, manifest unaffected)"
    requirement: "AGENT-09"
    verification:
      - kind: unit
        ref: "internal/agents/skillshared_test.go#TestSharedSkillPackage_WritesOnceAndAccumulatesTargets"
        status: pass
      - kind: unit
        ref: "internal/agents/skillshared_test.go#TestSharedSkillPackage_UninstallRemovesOnlyRequester"
        status: pass
      - kind: unit
        ref: "internal/agents/skillshared_test.go#TestSharedSkillPackage_LastRequesterDeletesPackage"
        status: pass
      - kind: unit
        ref: "internal/agents/skillshared_test.go#TestSharedSkillPackage_UninstallNonRequesterIsNoop"
        status: pass
      - kind: unit
        ref: "internal/agents/skillshared_test.go#TestSharedSkillPackage_ForeignDirKeptForeign"
        status: pass
      - kind: unit
        ref: "internal/agents/skillshared_test.go#TestSharedSkillPackage_EmptyDirIsNotForeign"
        status: pass
      - kind: unit
        ref: "internal/agents/skillshared_test.go#TestSharedSkillPackage_AdoptPolicyAdoptsUnmanifested"
        status: pass
      - kind: unit
        ref: "internal/agents/skillshared_test.go#TestSharedSkillPackage_HandEditedOwnFileRewritten"
        status: pass
      - kind: unit
        ref: "internal/agents/skillshared_test.go#TestSharedSkillPackage_LegacyAndCorruptManifestReadAsClaude"
        status: pass
      - kind: unit
        ref: "internal/agents/skillshared_test.go#TestSharedSkillPackage_UserFileKeepsDir"
        status: pass
      - kind: unit
        ref: "internal/agents/skillshared_test.go#TestSharedSkillPackage_ExclusiveKeysDroppedWhenRequesterLeaves"
        status: pass
    human_judgment: false
  - id: D3
    description: "AGENT-09 adjacency/empty/ordering invariant: every install/uninstall sequence of length 1-4 over 3 requesters (1554 sequences) leaves the package present iff the modelled requester set is non-empty, with the manifest's targets set exactly matching that model at every step"
    requirement: "AGENT-09"
    verification:
      - kind: unit
        ref: "internal/agents/skillshared_test.go#TestSharedSkillPackage_TargetsInvariantOverAllSequences"
        status: pass
    human_judgment: false
  - id: D4
    description: "D-17 symlink-safety groundwork: removeSkillDirIfEmpty never unlinks a symlink; resolveSkillDir/sameSkillDir resolve live, dangling, and non-existent paths (bounded against cycles); a symlinked skill directory is recognized as one physical package regardless of which path (link or real dir) install/uninstall is called through"
    requirement: "AGENT-13"
    verification:
      - kind: unit
        ref: "internal/agents/skillshared_test.go#TestRemoveSkillDirIfEmpty_NeverUnlinksSymlink"
        status: pass
      - kind: unit
        ref: "internal/agents/skillshared_test.go#TestSameSkillDir_ResolvesSymlinksAndDanglingLinks"
        status: pass
      - kind: unit
        ref: "internal/agents/skillshared_test.go#TestSkillPackage_DanglingSymlinkDirIsRecreated"
        status: pass
      - kind: unit
        ref: "internal/agents/skillshared_test.go#TestSkillPackage_WritesThroughSymlinkedDirAsOnePackage"
        status: pass
    human_judgment: false
  - id: D5
    description: "No target calls the new writer yet — nothing on disk changes for existing users; every pre-existing internal/agents and internal/cli test passes unmodified"
    requirement: "AGENT-09"
    verification:
      - kind: unit
        ref: "internal/agents/ + internal/cli/... (full suites, -count=1)"
        status: pass
      - kind: unit
        ref: "internal/agents/claude_skillpackage_test.go (unmodified, still pass)"
        status: pass
    human_judgment: false

duration: 55min
completed: 2026-09-18
status: complete
---

# Phase 5 Plan 2: Manifest-Owned Skill Package Writer & Symlink Safety Summary

**One manifest-owned writer (`installSkillPackage`/`uninstallSkillPackage`) now owns every codegraph skill directory via a `targets` requester set proven present-iff-non-empty over 1554 install/uninstall sequences, with bounded symlink resolution so a user's own skill-sharing symlink is never unlinked.**

## Performance

- **Duration:** 55 min
- **Started:** 2026-09-18T19:57:00Z
- **Completed:** 2026-09-18T20:52:00Z
- **Tasks:** 2
- **Files modified:** 5 (2 created, 3 modified)

## Accomplishments

- `skillManifest` gained `Targets []TargetID` (schema 2, additive); `manifestRequesters` folds readManifest's three outcomes into one non-destructive reading — an unreadable OR nil-Targets manifest reads as owned by `[claude]`, never as "unknown," per the plan's accepted D-07 planner amendment.
- `internal/agents/skillshared.go` is the single manifest-owned writer every later plan (05-03/05-04/05-05) wires targets onto: `installSkillPackage`/`uninstallSkillPackage` implement D-05 (write once, accumulate requesters), D-07 (legacy/corrupt manifests read as Claude), D-08 (last-requester deletes, non-requester is a no-op, departing requester's exclusive manifest keys are dropped), D-14 (a manifest-less directory with content is foreign — `kept (foreign)`, never touched), and D-16 (a hand-edited own file is silently rewritten, the manifest's own drift-not-tamper posture unaffected).
- `TestSharedSkillPackage_TargetsInvariantOverAllSequences` exhaustively walks every install/uninstall sequence of length 1-4 over 3 requesters — 1554 sequences from a fresh temp dir each — asserting `(SKILL.md exists AND manifest exists) == (modelled requester set non-empty)` and `manifest.Targets == model` after every single step.
- `resolveSkillDir`/`sameSkillDir` resolve a live symlink, a genuinely dangling symlink (target never created), or a non-existent path with no link at all, bounded to depth 8 so a self-referential link errors instead of hanging; `removeSkillDirIfEmpty` gained an `os.Lstat` guard so it never unlinks a symlink passed as `dir` (a user's own `~/.claude/skills/codegraph -> ../../.agents/skills/codegraph` link, confirmed present on the maintainer's own machine in 05-RESEARCH.md Pitfall 2, survives codegraph's own uninstall).
- Installing through a dangling relative symlink now recreates the resolved target directory (rather than failing with `mkdir: file exists` against the symlink's own dirent) and writes the package into it, leaving the link itself intact; installing/uninstalling through either a symlink or its real target operates on the SAME physical manifest with no special-case code required — the OS's own symlink transparency already gives this for free once the dangling-target case is handled.
- No target calls the new writer yet (05-03 moves Claude onto it): zero diff outside `internal/agents`, and every pre-existing test — including `internal/agents/claude_skillpackage_test.go`'s tracer test — passes unmodified.

## Task Commits

Each task followed RED-GREEN TDD discipline:

1. **Task 1 (RED): failing manifest-owned skill package writer tests** — `95bf13b2` (test)
2. **Task 1 (GREEN): manifest-owned skill package writer with a requester set** — `26b5d96f` (feat)
3. **Task 2 (RED): failing symlink-safety tests for skill directories** — `10380bab` (test)
4. **Task 2 (GREEN): symlink-safe skill directory resolution and removal** — `d1011f55` (feat)

**Plan metadata:** committed alongside this SUMMARY.

### RED evidence (Task 1)

`GOTOOLCHAIN=go1.26.6 go test ./internal/agents/ -count=1 -run 'TestSharedSkillPackage_|TestManifest_' -v` against the first `test(05-02):` commit — every named test fails on an assertion, never a build error:

```
    manifest_test.go:441: added-target writeManifest action = "unchanged", want "updated"
--- FAIL: TestManifest_TargetsRoundTripAndSetEquality (0.00s)
    skillshared_test.go:51: read SKILL.md after first install: open SKILL.md: no such file or directory
--- FAIL: TestSharedSkillPackage_WritesOnceAndAccumulatesTargets (0.00s)
    --- FAIL: TestSharedSkillPackage_WritesOnceAndAccumulatesTargets/global (0.00s)
    --- FAIL: TestSharedSkillPackage_WritesOnceAndAccumulatesTargets/local (0.00s)
    skillshared_test.go:148: manifest not present after partial uninstall: present=false err=<nil>
--- FAIL: TestSharedSkillPackage_UninstallRemovesOnlyRequester (0.00s)
    skillshared_test.go:193: result.Files missing entry for  (want action "removed"): []
--- FAIL: TestSharedSkillPackage_LastRequesterDeletesPackage (0.00s)
    skillshared_test.go:214: read SKILL.md before uninstall: open .../codegraph/SKILL.md: no such file or directory
--- FAIL: TestSharedSkillPackage_UninstallNonRequesterIsNoop (0.00s)
    skillshared_test.go:265: expected exactly one {dir, kept (foreign)} entry, got []
--- FAIL: TestSharedSkillPackage_ForeignDirKeptForeign (0.00s)
    skillshared_test.go:309: result.Files missing entry for .../codegraph/SKILL.md (want action "created"): []
--- FAIL: TestSharedSkillPackage_EmptyDirIsNotForeign (0.00s)
    skillshared_test.go:342: SKILL.md not rewritten to embedded content under adopt policy
--- FAIL: TestSharedSkillPackage_AdoptPolicyAdoptsUnmanifested (0.00s)
    skillshared_test.go:369: read manifest before hand-edit: open : no such file or directory
--- FAIL: TestSharedSkillPackage_HandEditedOwnFileRewritten (0.00s)
    skillshared_test.go:425: seed legacy manifest: open : no such file or directory
    skillshared_test.go:471: seed corrupt manifest: open : no such file or directory
--- FAIL: TestSharedSkillPackage_LegacyAndCorruptManifestReadAsClaude (0.00s)
    --- FAIL: TestSharedSkillPackage_LegacyAndCorruptManifestReadAsClaude/legacy_schema_version_1,_no_targets_key (0.00s)
    --- FAIL: TestSharedSkillPackage_LegacyAndCorruptManifestReadAsClaude/corrupt_manifest_self-heals (0.00s)
    skillshared_test.go:503: seed user file: open .../codegraph/notes.md: no such file or directory
--- FAIL: TestSharedSkillPackage_UserFileKeepsDir (0.00s)
    skillshared_test.go:555: manifest not present: present=false err=<nil>
--- FAIL: TestSharedSkillPackage_ExclusiveKeysDroppedWhenRequesterLeaves (0.00s)
    skillshared_test.go:611: sequence [{install:true target:cursor}]: after step {install:true target:cursor}, package present=false, want model non-empty=true
--- FAIL: TestSharedSkillPackage_TargetsInvariantOverAllSequences (0.00s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/agents	0.120s
```

All 12 named `TestSharedSkillPackage_*` tests plus `TestManifest_TargetsRoundTripAndSetEquality` failed on assertions against the RED-phase zero-body placeholders (`installSkillPackage`/`uninstallSkillPackage`/etc. were no-ops; `manifestRequesters`/`targetSetEqual` returned `nil`/`false`).

### RED evidence (Task 2)

`GOTOOLCHAIN=go1.26.6 go test ./internal/agents/ -count=1 -run 'TestRemoveSkillDirIfEmpty_NeverUnlinksSymlink$|TestSameSkillDir_ResolvesSymlinksAndDanglingLinks$|TestSkillPackage_DanglingSymlinkDirIsRecreated$|TestSkillPackage_WritesThroughSymlinkedDirAsOnePackage$' -v` against the second `test(05-02):` commit:

```
    skillshared_test.go:653: Lstat(link) after removeSkillDirIfEmpty: lstat .../link-empty: no such file or directory
    skillshared_test.go:681: Lstat(link) after removeSkillDirIfEmpty: lstat .../link-nonempty: no such file or directory
--- FAIL: TestRemoveSkillDirIfEmpty_NeverUnlinksSymlink (0.00s)
    --- FAIL: TestRemoveSkillDirIfEmpty_NeverUnlinksSymlink/symlink_to_empty_dir (0.00s)
    --- FAIL: TestRemoveSkillDirIfEmpty_NeverUnlinksSymlink/symlink_to_non-empty_dir (0.00s)
    --- PASS: TestRemoveSkillDirIfEmpty_NeverUnlinksSymlink/plain_empty_dir_still_removed (0.00s)
    --- PASS: TestRemoveSkillDirIfEmpty_NeverUnlinksSymlink/plain_non-empty_dir_still_kept (0.00s)
    skillshared_test.go:763: sameSkillDir(relative, absolute) = false, want true
    skillshared_test.go:786: sameSkillDir(symlink, real target) = false, want true
    skillshared_test.go:810: sameSkillDir(dangling symlink, its not-yet-created target) = false, want true
    skillshared_test.go:840: sameSkillDir(self-referential symlink) returned no error
--- FAIL: TestSameSkillDir_ResolvesSymlinksAndDanglingLinks (0.00s)
    --- PASS: TestSameSkillDir_ResolvesSymlinksAndDanglingLinks/identical_path (0.00s)
    --- FAIL: TestSameSkillDir_ResolvesSymlinksAndDanglingLinks/relative_vs_absolute (0.00s)
    --- FAIL: TestSameSkillDir_ResolvesSymlinksAndDanglingLinks/live_symlink_resolves_to_real_target (0.00s)
    --- FAIL: TestSameSkillDir_ResolvesSymlinksAndDanglingLinks/dangling_symlink_resolves_to_same_not-yet-created_target (0.00s)
    --- PASS: TestSameSkillDir_ResolvesSymlinksAndDanglingLinks/unrelated_dirs (0.00s)
    --- FAIL: TestSameSkillDir_ResolvesSymlinksAndDanglingLinks/self-referential_symlink_is_an_error,_not_a_hang (0.00s)
    skillshared_test.go:872: installSkillPackage through dangling symlink: [claude/skills/codegraph/SKILL.md: mkdir claude/skills/codegraph: file exists]
--- FAIL: TestSkillPackage_DanglingSymlinkDirIsRecreated (0.00s)
--- PASS: TestSkillPackage_WritesThroughSymlinkedDirAsOnePackage (0.00s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/agents	0.093s
```

`TestRemoveSkillDirIfEmpty_NeverUnlinksSymlink` failed exactly as predicted: `removeSkillDirIfEmpty` had no symlink guard yet, so `os.Remove` unlinked the symlink itself regardless of its target's emptiness. `TestSkillPackage_DanglingSymlinkDirIsRecreated` failed with the predicted `mkdir: file exists` error (the OS sees an existing dirent — the dangling symlink — where `writeEmbeddedFile`'s own `MkdirAll` expects nothing). `TestSameSkillDir_ResolvesSymlinksAndDanglingLinks` failed against its RED-phase literal-string-equality placeholder. `TestSkillPackage_WritesThroughSymlinkedDirAsOnePackage` passed even at RED — it exercises no symlink-specific code path beyond what Task 1's writer already provides (both the link and its real target resolve to the same physical file through ordinary OS symlink transparency), which is expected and not a defect in the RED demonstration.

## Files Created/Modified

- `internal/agents/manifest.go` — `skillManifest.Targets` (schema 2), `manifestRequesters`, `targetSetEqual`, `writeManifest`'s extended unchanged check
- `internal/agents/manifest_test.go` — `TestManifest_TargetsRoundTripAndSetEquality`
- `internal/agents/skillshared.go` (new) — the shared writer: `ActionKeptForeign`, `unmanifestedPolicy`, `sharedSkillDirPath`/`sharedSkillDirs`, `skillManifestPath`, `skillDirIsForeign`, `writeSkillFile`, `recordSkillManifest`, `installSkillPackage`, `uninstallSkillPackage`, `resolveSkillDir`, `sameSkillDir`
- `internal/agents/skillshared_test.go` (new) — 12 `TestSharedSkillPackage_*` tests, `TestRemoveSkillDirIfEmpty_NeverUnlinksSymlink`, `TestSameSkillDir_ResolvesSymlinksAndDanglingLinks`, `TestSkillPackage_DanglingSymlinkDirIsRecreated`, `TestSkillPackage_WritesThroughSymlinkedDirAsOnePackage`
- `internal/agents/shared.go` — `removeSkillDirIfEmpty` gained an `os.Lstat` symlink guard

## Decisions Made

See `key-decisions` in frontmatter — the D-07 legacy/corrupt-manifest reading, the Task 1/Task 2 boundary on the dangling-symlink `writeSkillFile` change, and the schema-version-bump `costly` flag recorded for the maintainer.

## Deviations from Plan

None — plan executed exactly as written, including the Task 1/Task 2 split of `writeSkillFile`'s symlink handling (Task 1's `<action>` text does not mention it; Task 2's does, and depends on Task 2's `resolveSkillDir`).

## Issues Encountered

None.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- `05-03` can now move Claude's own `Install`/`Uninstall` onto `installSkillPackage`/`uninstallSkillPackage` (or, per D-17, detect when Claude's own directory coincides with the shared directory via `sameSkillDir` and treat it as one package) with a stable, tested writer contract underneath it.
- `05-04`/`05-05` can wire Cursor/opencode/Antigravity/Gemini/Kiro onto `installSkillPackage(sharedSkillDirPath(loc), loc, <target>, refuseUnmanifested)` directly.
- No blockers or concerns carried forward. `manifestSchemaVersion` is now 2 — flagged `costly` per the plan, self-healing behavior is proven by `TestSharedSkillPackage_LegacyAndCorruptManifestReadAsClaude`.

## Self-Check: PASSED

- `internal/agents/skillshared.go` exists: FOUND
- `internal/agents/skillshared_test.go` exists: FOUND
- Commit `95bf13b2` (test): FOUND in `git log --oneline --all`
- Commit `26b5d96f` (feat): FOUND in `git log --oneline --all`
- Commit `10380bab` (test): FOUND in `git log --oneline --all`
- Commit `d1011f55` (feat): FOUND in `git log --oneline --all`
- `GOTOOLCHAIN=go1.26.6 go test ./internal/agents/ ./internal/cli/... -count=1`: PASS (all packages ok)
- `TestSharedSkillPackage_TargetsInvariantOverAllSequences` logs "executed exactly 1554 install/uninstall sequences": PASS
- Full module test (excluding `internal/daemon`, run alone per WINDOWS #37): all 60 packages `ok`
- `internal/daemon` alone: `ok`
- Zero diff outside `internal/agents` since `44fafa27` (05-01 complete): confirmed via `git diff --stat`

---
*Phase: 05-agent-reach-capability-model-skill-in-every-harness*
*Completed: 2026-09-18*
