---
phase: 07-codegraph-install-skill-hooks-distribution-claude-code
plan: 01
subsystem: agents
tags: [go-embed, claude-code, hooks, skill-install, json-merge, atomic-write]

# Dependency graph
requires:
  - phase: 06-agent-skill-package-skill-md-sessionstart-nudge
    provides: "the canonical .claude/skills/codegraph/SKILL.md, .claude/hooks/hooks.json, .claude/hooks/session-nudge.sh package this plan embeds and installs verbatim"
provides:
  - "claudeassets (repo-root go:embed package) carrying the binary's own copy of Phase 6's .claude/ package"
  - "claudeTarget.Install writes SKILL.md, an executable session-nudge.sh, and a SessionStart hooks.json merge into Claude Code's real read locations at both --location scopes"
  - "readJSONFileStrict / writeHookEntry / atomicWriteExecutableFile / writeEmbeddedFile in internal/agents/shared.go — reusable array-scoped JSON merge and fail-closed read primitives for Plan 02's uninstall/read-error-matrix work"
affects: [07-02-uninstall, 07-03-manifest-version-observability, 07-04-upgrade-auto-refresh]

# Actuals (#2632)
actuals:
  tokens: 9200
  tasks: 2
  commits: 3

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Repository-root go:embed source package (claudeassets) — the only directory that is an ancestor of both internal/ and .claude/, required because Go embed patterns reject '..' path elements"
    - "Array-scoped JSON hook merge (writeHookEntry) identifying ownership by command-string match inside hooks[], never by matcher value alone — the array analog of writeMcpEntry's single-key merge"
    - "Fail-closed strict JSON reader (readJSONFileStrict) as a second, stricter posture alongside the pre-existing permissive readJSONFile — modeled on internal/githooks' skip-and-accumulate read switch"

key-files:
  created:
    - claudeassets.go
    - internal/agents/claude_skillpackage_test.go
  modified:
    - internal/agents/claude.go
    - internal/agents/shared.go
    - internal/agents/claude_test.go

key-decisions:
  - "claudeassets.go placed at the repository root, not internal/agents — Go's //go:embed cannot cross the .claude/<->internal/ sibling boundary via '..' (verified this session by the RESEARCH doc's live experiment and confirmed here by a clean go build)."
  - "Global-scope hook command uses the fully-resolved absolute script path (filepath.Join(home, ...)) rather than a literal '~' — sidesteps RESEARCH Assumption A3's unverified shell-expansion question entirely, at zero cost."
  - "Hooks ownership in writeHookEntry is identified by exact command-string match inside a block's hooks[] sub-array, never by matcher value — a user may legitimately own an unrelated block under the same matcher (e.g. 'startup'), proven by TestClaude_Install_PreservesUnrelatedHooksContent."

patterns-established:
  - "writeEmbeddedFile: raw-byte idempotent write for non-JSON embedded artifacts (compares current disk bytes to target content before writing, so a rewrite with identical bytes never happens)"
  - "claudeSessionStartBlocks derives its output by rewriting the embedded hooks fragment's own command field per-location, rather than hand-authoring matcher/command literals in Go — keeps Phase 6's .claude/ the single canonical source (Phase 6 D-04)"

requirements-completed: [AGENT-01]

coverage:
  - id: D1
    description: "codegraph install writes the binary's own embedded SKILL.md, executable session-nudge.sh, and SessionStart hooks registration into Claude Code's real global read locations (~/.claude/...), through claudeTarget.Install"
    requirement: AGENT-01
    verification:
      - kind: unit
        ref: "internal/agents/claude_skillpackage_test.go#TestClaude_Install_WritesSkillPackage_EndToEnd"
        status: pass
    human_judgment: false
  - id: D2
    description: "A second install at the same location is a true no-op: SKILL.md/session-nudge.sh raw bytes and settings.json (jsonDeepEqual) are unchanged, and every FileResult reports ActionUnchanged"
    requirement: AGENT-01
    verification:
      - kind: unit
        ref: "internal/agents/claude_skillpackage_test.go#TestClaude_Install_SkillPackageReRunIsUnchanged"
        status: pass
    human_judgment: false
  - id: D3
    description: "Installing preserves unrelated pre-existing SessionStart blocks (including one sharing codegraph's own 'startup' matcher), unrelated hook events, and unrelated top-level settings.json keys"
    requirement: AGENT-01
    verification:
      - kind: unit
        ref: "internal/agents/claude_skillpackage_test.go#TestClaude_Install_PreservesUnrelatedHooksContent"
        status: pass
      - kind: unit
        ref: "internal/agents/claude_skillpackage_test.go#TestClaude_Install_HooksBoundaryStates"
        status: pass
    human_judgment: false
  - id: D4
    description: "The hook command string is never a bare filename: local scope matches Phase 6's dogfooded ${CLAUDE_PROJECT_DIR}-relative fragment byte-for-byte; global scope is the fully-resolved absolute path to the script this same install wrote"
    requirement: AGENT-01
    verification:
      - kind: unit
        ref: "internal/agents/claude_skillpackage_test.go#TestClaude_Install_HookCommandIsLocationAware"
        status: pass
      - kind: unit
        ref: "internal/agents/hookpackage_test.go#TestHookRegistrationMatchesFragmentAndScript"
        status: pass
    human_judgment: false
  - id: D5
    description: "The binary embeds exactly the three package files and nothing under .claude/skills/codegraph/verification/"
    requirement: AGENT-01
    verification:
      - kind: unit
        ref: "internal/agents/claude_skillpackage_test.go#TestClaudeAssets_EmbedsNoVerificationTranscripts"
        status: pass
    human_judgment: false

duration: ~20min
completed: 2026-08-13
status: complete
---

# Phase 7 Plan 1: Claude Code Skill Package Install (Tracer) Summary

**`codegraph install` now writes the binary's own embedded Claude Code skill package — SKILL.md, an executable session-nudge.sh, and a command-string-scoped SessionStart hooks merge into settings.json — at both `--location` scopes, proven idempotent and boundary-safe by 6 new tests.**

## Performance

- **Duration:** ~20 min
- **Started:** 2026-08-13T16:05:44Z (first commit)
- **Completed:** 2026-08-13T16:14:36Z
- **Tasks:** 2
- **Files modified:** 5 (2 created, 3 modified)

## Accomplishments
- New root-level `claudeassets` package embeds Phase 6's `.claude/skills/codegraph/SKILL.md`, `.claude/hooks/hooks.json`, and `.claude/hooks/session-nudge.sh` via three exact-file `//go:embed` directives (no directory pattern, so Phase 6's `verification/` rehearsal transcripts are never shipped).
- `claudeTarget.Install` gained three new steps — skill file write, executable script write, SessionStart hooks merge — that run unconditionally per `--location`, independent of `--auto-allow` (which now controls only the `permissions.allow` entry).
- New array-scoped `writeHookEntry` merges `hooks.SessionStart` by command-string identity (never matcher value), preserving any unrelated block under the same matcher.
- New fail-closed `readJSONFileStrict` gives the hooks-merge path (and future manifest work) a stricter read posture than `readJSONFile`'s existing permissive empty-map fallback, which is left untouched for its existing callers.
- `writeEmbeddedFile` makes the two non-JSON artifacts raw-byte idempotent; `atomicWriteExecutableFile` closes the "fsatomic defaults new files to 0644" gap that would have shipped a non-executable nudge script.
- 6 new tests (1 tracer end-to-end + 5 property tests) plus the pre-existing `TestHookRegistrationMatchesFragmentAndScript` all pass; full `go test ./...` is green across every package in the repo.

## Task Commits

Each task was committed atomically (TDD RED/GREEN split for Task 1, single commit for Task 2 since its tests characterized already-correct behavior with no new implementation needed):

1. **Task 1 (tracer): install writes SKILL.md/script/hooks — RED** - `c4d7006` (test)
2. **Task 1 (tracer): install writes SKILL.md/script/hooks — GREEN** - `2dc7184` (feat)
3. **Task 2: prove idempotency, boundaries, location divergence** - `f829987` (test)

**Plan metadata:** commit pending (this SUMMARY.md + STATE/ROADMAP, applied by the orchestrator per wave-merge protocol — worktree mode excludes shared-file writes from this plan's own commits)

## Files Created/Modified
- `claudeassets.go` - repo-root `go:embed` source for Phase 6's `.claude/` package; exports `FS`, path constants, and typed accessors
- `internal/agents/claude_skillpackage_test.go` - 6 tests: the tracer end-to-end proof plus 5 property tests (idempotency, unrelated-content preservation, boundary states, location-aware command, embed scope)
- `internal/agents/claude.go` - new `claudeSkillDirPath`/`claudeSkillFilePath`/`claudeHooksScriptPath`/`claudeHookCommand`/`claudeSessionStartBlocks` helpers; `Install` gains the three new steps
- `internal/agents/shared.go` - new `readJSONFileStrict`, `writeHookEntry`, `atomicWriteExecutableFile`, `writeEmbeddedFile` (additive only — `readJSONFile` unmodified)
- `internal/agents/claude_test.go` - renamed/narrowed `TestClaude_Install_NoAutoAllow_NoSettingsWrite` to `TestClaude_Install_NoAutoAllow_NoPermissionsWrite` (see Deviations)

## Decisions Made
- `claudeassets.go` at the repository root, not `internal/agents` — the only directory that is an ancestor of both `.claude/` and `internal/`; a `go build ./...` failure would be the guardrail against a later relocation attempt.
- Global-scope hook command resolves to an absolute `filepath.Join(home, ...)` path rather than relying on `~` shell expansion, sidestepping RESEARCH's unverified Assumption A3 at zero implementation cost.
- Hooks ownership identity is the `command` string inside a block's `hooks[]` sub-array, never the block's `matcher` value — required to keep a user's own unrelated `"startup"`-matcher block safe from corruption or replacement.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Updated a pre-existing test whose invariant this phase's own locked decision supersedes**
- **Found during:** Task 1 (running `go test ./internal/agents/... ./internal/cli/...` after the GREEN implementation)
- **Issue:** `TestClaude_Install_NoAutoAllow_NoSettingsWrite` asserted `settings.json` is never written without `--auto-allow`. CONTEXT.md's locked D-01 ("the skill+hooks package follows `--location` with no special-casing") means Phase 7 now writes `settings.json` unconditionally to carry the SessionStart hooks registration — the old assertion is false by design, not by bug.
- **Fix:** Renamed to `TestClaude_Install_NoAutoAllow_NoPermissionsWrite` and narrowed it to what `--auto-allow` actually still gates: the `permissions.allow` entry. `settings.json` existing is now asserted as expected; the absence of a `permissions` key without `--auto-allow` is still asserted.
- **Files modified:** `internal/agents/claude_test.go`
- **Verification:** `go test ./internal/agents/... ./internal/cli/...` green; full `go test ./...` green.
- **Committed in:** `2dc7184` (part of Task 1's GREEN commit)

---

**Total deviations:** 1 auto-fixed (1 Rule 1 — stale test invariant)
**Impact on plan:** Necessary to reconcile a pre-existing test with this phase's own locked scope decision (D-01). No scope creep — the fix only re-scopes what the existing test asserts, adding no new behavior.

## Issues Encountered
None beyond the deviation above.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Plan 02 (uninstall + read-error/malformed-file matrix) can now build directly on `readJSONFileStrict`, `writeHookEntry`, and the path helpers this plan introduced — no new file-safety primitive is needed.
- Plan 03 (manifest/version observability) and Plan 04 (upgrade auto-refresh) both depend on `claudeassets` and the location-aware path helpers established here; both are now unblocked.
- `writeHookEntry` currently only supports the "add/replace codegraph's own blocks" direction Install needs — Plan 02 will need the mirror removal (locate-by-command, strip-if-empties, delete-key-if-empties) analogous to `removeMcpEntry`.

---
*Phase: 07-codegraph-install-skill-hooks-distribution-claude-code*
*Completed: 2026-08-13*
