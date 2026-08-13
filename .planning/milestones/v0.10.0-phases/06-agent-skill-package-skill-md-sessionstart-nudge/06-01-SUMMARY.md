---
phase: 06-agent-skill-package-skill-md-sessionstart-nudge
plan: 01
subsystem: agents
tags: [claude-code, hooks, sessionstart, tdd, dogfooding]

requires:
  - phase: 05-mcp-resources-capability-claims-drift-guard
    provides: "codegraph:// resource URIs that SKILL.md (06-02/06-03) will point to instead of restating"
provides:
  - "Working, dogfooded SessionStart nudge in this repo's own .claude/ — a session started in a .codegraph/-indexed tree receives exactly one pointer line"
  - ".claude/settings.json, the one file Claude Code actually reads for a project-scoped SessionStart hook (corrects CONTEXT.md's stated .claude/hooks/hooks.json mechanism, per 06-RESEARCH.md's Critical Correction)"
  - ".claude/hooks/hooks.json, a versioned fragment gated equal to settings.json by a test, for Phase 7's go:embed"
  - "internal/agents/hookpackage_test.go proving addClaudeAllowPermission/removeClaudeAllowPermission never disturb the hooks block (T-06-03)"
affects: ["07-*: go:embed of .claude/skills/codegraph/ and .claude/hooks/hooks.json depends on these exact paths (D-04, costly reversibility)"]

actuals:
  tokens: 4604
  tasks: 2
  commits: 3

tech-stack:
  added: []
  patterns:
    - "SessionStart hook registered in .claude/settings.json (committed, project-scoped), never in a standalone hooks.json — Claude Code only reads hooks.json inside an installed plugin package"
    - "Two-file hook fragment: settings.json is the live registration, hooks.json is a gated-equal versioned embed source for a later phase — kept in sync by a Go test (TestHookRegistrationMatchesFragmentAndScript), not by discipline"
    - "printf with a single-quoted literal argument, never echo, for any hook script emitting a message containing backticks or other shell-meaningful characters"

key-files:
  created:
    - .claude/hooks/session-nudge.sh
    - .claude/settings.json
    - .claude/hooks/hooks.json
    - internal/agents/hookpackage_test.go
  modified: []

key-decisions:
  - "Followed 06-RESEARCH.md's Critical Correction over 06-CONTEXT.md's D-03/D-04 literal text: registered the SessionStart hook in .claude/settings.json (which Claude Code actually reads for a non-plugin project), not in .claude/hooks/hooks.json alone. hooks.json was still created, carrying the identical block, as Phase 7's go:embed source — this preserves D-04's location commitment without shipping an inert hook registration."
  - "Task 1's acceptance-criteria grep for command-substitution syntax ('eval|\\$(|`') is documented as returning 1 rather than the specified 0 — see Deviations."

requirements-completed: [NUDGE-01, NUDGE-02]

coverage:
  - id: D1
    description: "session-nudge.sh emits exactly one pinned line to stdout, nothing to stderr, exit 0, when CLAUDE_PROJECT_DIR points at a tree with a .codegraph directory (including an empty one, and independent of whether the env var is set or CLAUDE_PROJECT_DIR is unset and cmd.Dir is used instead)"
    requirement: "NUDGE-01"
    verification:
      - kind: unit
        ref: "internal/agents/hookpackage_test.go#TestSessionNudgeBehavesPerIndexPresence"
        status: pass
      - kind: unit
        ref: "internal/agents/hookpackage_test.go#TestSessionNudgeOutputIsPinnedAndStateless"
        status: pass
    human_judgment: false
  - id: D2
    description: "session-nudge.sh emits zero bytes to stdout and stderr, exit 0, when no .codegraph entry exists, or when .codegraph exists as a regular file (not a directory) — no extra filesystem work beyond the one directory check"
    requirement: "NUDGE-02"
    verification:
      - kind: unit
        ref: "internal/agents/hookpackage_test.go#TestSessionNudgeBehavesPerIndexPresence"
        status: pass
    human_judgment: false
  - id: D3
    description: ".claude/settings.json registers session-nudge.sh under hooks.SessionStart for both the startup and resume matchers, each ${CLAUDE_PROJECT_DIR}-anchored, with exactly one top-level key"
    verification:
      - kind: unit
        ref: "node -e settings.json shape check (SETTINGS_OK)"
        status: pass
    human_judgment: false
  - id: D4
    description: ".claude/hooks/hooks.json is structurally equal to .claude/settings.json's SessionStart block, proven by a Go test, and every command path it names resolves to a real executable file"
    verification:
      - kind: unit
        ref: "internal/agents/hookpackage_test.go#TestHookRegistrationMatchesFragmentAndScript"
        status: pass
      - kind: unit
        ref: "internal/agents/hookpackage_test.go#TestHookRegistrationMatchesFragmentAndScript_ComparisonDiscriminates"
        status: pass
    human_judgment: false
  - id: D5
    description: "addClaudeAllowPermission and removeClaudeAllowPermission (the real codegraph install/uninstall merge functions) leave the hooks block byte-for-byte unchanged"
    requirement: ""
    verification:
      - kind: unit
        ref: "internal/agents/hookpackage_test.go#TestClaudeInstallPreservesHooksBlock"
        status: pass
    human_judgment: false

duration: 37min
completed: 2026-08-12
status: complete
---

# Phase 6 Plan 1: SessionStart Nudge — Script, Live Registration & Embed Fragment Summary

**A working, dogfooded Claude Code SessionStart hook in this repo's own `.claude/`: a 12-line POSIX script gated by one directory-existence check, registered where Claude Code actually reads project hooks (`.claude/settings.json`, not a bare `hooks.json`), with a byte-equal versioned fragment for Phase 7's future `go:embed`.**

## Performance

- **Duration:** 37 min
- **Started:** 2026-08-12T22:19:14Z (prior commit `docs(06): create phase plan`)
- **Completed:** 2026-08-12T22:56:02Z
- **Tasks:** 2
- **Files modified:** 4 (all created, none modified)

## Accomplishments

- `.claude/hooks/session-nudge.sh`: single `[ -d ... ]` test, single `printf` with a single-quoted D-06 message, `exit 0` unconditionally — no `eval`, no command substitution, no stdin parsing, mode `100755`.
- `.claude/settings.json` (net-new, committed): the one file Claude Code actually reads for this repo's project-scoped `SessionStart` hook, registering the script on both the `startup` and `resume` matchers via `${CLAUDE_PROJECT_DIR}`-anchored paths.
- `.claude/hooks/hooks.json`: a byte-structurally-equal fragment held equal to `settings.json`'s `hooks.SessionStart` by `TestHookRegistrationMatchesFragmentAndScript`, existing solely as Phase 7's future `go:embed` source (never read by Claude Code in this repository).
- `internal/agents/hookpackage_test.go`: six test functions proving the whole path end to end — script behavior across six presence/absence/env boundaries, byte-exact and concurrency-safe output, fragment/registration parity, and survival of `codegraph install`/`uninstall`'s real JSON-merge functions.
- Demonstrated RED twice: the whole test suite before the script/settings existed (six failures, all "no such file"/missing-file, never a logic mismatch), and a targeted one-field mutation to `hooks.json` after both files existed (caught by name, reverted byte-clean, re-ran green).

## Task Commits

Each task was committed atomically, following the plan's `tdd="true"` RED→GREEN cycle:

1. **Task 1 RED — failing tests** - `184fb6a` (test)
2. **Task 1 GREEN — script + live registration** - `3da7af4` (feat)
3. **Task 2 GREEN — embed fragment + parity/install tests** - `9c66d8f` (feat)

_Note: Task 2's tests were authored together with Task 1's in the single RED commit (`184fb6a`), since both were written before either script or settings.json existed and both went RED together for the same missing-file reason. Only the GREEN half is split across two commits, matching the plan's two-task structure._

**Plan metadata:** (this commit, made after this SUMMARY)

## Files Created/Modified

- `.claude/hooks/session-nudge.sh` - the nudge script (NUDGE-01/02)
- `.claude/settings.json` - live `SessionStart` hook registration (net-new, committed)
- `.claude/hooks/hooks.json` - Phase-7 `go:embed` source fragment, gated equal to `settings.json` by test
- `internal/agents/hookpackage_test.go` - six test functions covering script behavior, registration/fragment parity, and install/uninstall survival

## Decisions Made

- **Registered the hook in `.claude/settings.json`, not `.claude/hooks/hooks.json`**, per 06-RESEARCH.md's Critical Correction: Claude Code only reads `hooks/hooks.json` inside an installed plugin package (`.claude-plugin/plugin.json`-referenced), never in a plain project directory. `.claude/hooks/hooks.json` was still created — carrying the identical block — because D-04 already commits Phase 7 to embedding from that exact path; this preserves that commitment while making the nudge actually fire in this repo today. `TestHookRegistrationMatchesFragmentAndScript` is what keeps the two files from silently drifting, rather than manual discipline.
- **`printf` with a single-quoted argument, never `echo`**, because D-06's mandated message text contains a literal backtick pair around `codegraph explore`. Single quotes make the backticks structurally inert (no command substitution is possible inside single quotes), which double-quoted `echo` cannot guarantee across all `sh` implementations.

## Deviations from Plan

### Auto-fixed Issues

None — the script and registration were authored per 06-RESEARCH.md's Critical Correction from the start, not discovered as a bug mid-task.

### Documented Acceptance-Criteria Conflict (not auto-fixed; recorded per instructions)

**1. [Acceptance-criteria defect, not a script defect] The Task 1 backtick-detection grep cannot return 0 while conforming to D-06**
- **Found during:** Task 1, running the acceptance-criteria verification commands after the script was authored.
- **Issue:** Task 1's acceptance criteria specify `grep -v '^#' .claude/hooks/session-nudge.sh | grep -cE 'eval|[$][(]|[\x60]' || true` must print `0` (no dynamic evaluation, no command substitution). It prints `1`. The reason is structural, not a bug: D-06's mandated nudge text is `...prefer codegraph_explore / \`codegraph explore\` over grep...` — it contains a literal backtick pair around `codegraph explore`, by explicit design (06-RESEARCH.md's own Pattern 2 code example includes the identical escaped-backtick pair). The grep pattern is a naive text scan that cannot distinguish "backtick used as the shell's command-substitution operator" from "backtick appearing as a literal byte inside a single-quoted string" — and the script's only backticks are the latter.
- **Why this is not command substitution in practice:** the backticks sit inside `printf`'s single-quoted argument (`printf '%s\n' '...codegraph_explore / `codegraph explore`...'`). POSIX shell treats every character inside single quotes literally, with no exception for backtick — this is exactly the property Task 1's own `<action>` text cites as the reason to use `printf` + single quotes over `echo` + double quotes for this specific message. `TestSessionNudgeOutputIsPinnedAndStateless`'s precision sub-test asserts the script's real stdout is byte-equal to `nudgeLine + "\n"` with the backticks intact; had any substitution occurred, that assertion would have failed (either erroring on an undefined `codegraph` command, or silently stripping/mangling the backtick-delimited segment). A direct manual execution (`CLAUDE_PROJECT_DIR=/tmp/nudge-check .claude/hooks/session-nudge.sh` against a freshly created `.codegraph/` dir) was also run this session and printed the literal backtick-bearing line unchanged.
- **Resolution:** left as documented, not "fixed" — fixing it would require either violating D-06's mandated message text (removing the backticks) or weakening the acceptance grep to distinguish quoting context, both of which are decisions outside this task's discretion (D-06 is a locked CONTEXT.md decision; rewriting the acceptance script mid-execution is a plan-authoring change, not an implementation one). The other two Task 1 grep checks (exactly 1 `printf`, exactly 1 `-d ` test) both pass as specified.
- **Files affected:** none — no code change; documentation only.
- **Verification:** manual execution + `TestSessionNudgeOutputIsPinnedAndStateless/precision`, both confirming byte-exact, substitution-free output.

---

**Total deviations:** 0 auto-fixed; 1 documented acceptance-criteria conflict (structural, non-blocking).
**Impact on plan:** None on functionality — NUDGE-01/02 are fully met and independently verified by execution, not by reading. The conflict is between two of the plan's own requirements (D-06's message content vs. a naive substitution-detection grep) and does not indicate a defect in the shipped script.

## Issues Encountered

None beyond the documented acceptance-criteria conflict above.

## User Setup Required

None — no external service configuration required. `.claude/settings.json` is committed and takes effect the next time a Claude Code session starts or resumes in this repository.

## Next Phase Readiness

- `.claude/hooks/session-nudge.sh`, `.claude/settings.json`, and `.claude/hooks/hooks.json` are all in place at the exact paths D-04 commits Phase 7 to `go:embed` from — no relocation needed.
- Plan 06-02/06-03 (SKILL.md) can proceed independently — no file overlap with this plan's `files_modified`.
- `internal/mcp/skill_claims_drift_test.go` (06-03) is the second of the two layers pinning `nudgeLine`'s honesty; this plan's `nudgeLine` constant in `internal/agents/hookpackage_test.go` is a hand-typed literal pin only (documented as such at its declaration), not itself a claims-drift guard.

## Self-Check: PASSED

- FOUND: `.claude/hooks/session-nudge.sh`
- FOUND: `.claude/settings.json`
- FOUND: `.claude/hooks/hooks.json`
- FOUND: `internal/agents/hookpackage_test.go`
- FOUND: commit `184fb6a` (test RED)
- FOUND: commit `3da7af4` (feat GREEN — Task 1)
- FOUND: commit `9c66d8f` (feat GREEN — Task 2)

---
*Phase: 06-agent-skill-package-skill-md-sessionstart-nudge*
*Completed: 2026-08-12*
