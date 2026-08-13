---
phase: 06-agent-skill-package-skill-md-sessionstart-nudge
reviewed: 2026-08-13T00:07:48Z
depth: deep
files_reviewed: 9
files_reviewed_list:
  - .claude/hooks/hooks.json
  - .claude/hooks/session-nudge.sh
  - .claude/settings.json
  - .claude/skills/codegraph/SKILL.md
  - .claude/skills/codegraph/verification/NUDGE-live-session.md
  - .claude/skills/codegraph/verification/SKILL-03-rehearsal.md
  - internal/agents/hookpackage_test.go
  - internal/mcp/skill_claims_drift_test.go
  - test/wireoracle/MUTATION-PROOF.md
findings:
  critical: 0
  warning: 2
  info: 2
  total: 4
status: issues_found
---

# Phase 6: Code Review Report

**Reviewed:** 2026-08-13T00:07:48Z
**Depth:** deep
**Files Reviewed:** 9
**Status:** issues_found

## Summary

Re-review after commits `156cda4` and `15e27da`, which addressed WR-01, WR-02, and IN-02 from the
prior deep review (`06-REVIEW.iter2.md`). All three fixes were verified directly, not taken on
faith:

- **WR-01** (`runSessionNudge`'s `useEnv==true` branch not stripping an inherited
  `CLAUDE_PROJECT_DIR` before appending the override) — fixed. Both branches now build `env` by
  filtering any `CLAUDE_PROJECT_DIR=` entry out of `os.Environ()` first; the `useEnv` branch then
  appends the test's own override to that filtered slice. Read the diff line-by-line and confirmed
  the asymmetry is gone.
- **WR-02** (`t.Fatalf`/`t.Helper` reached from spawned goroutines in the concurrency subtest) —
  fixed. `runSessionNudge` now returns `(stdout, stderr, exitCode, err)` instead of calling
  `t.Fatalf` internally; the three call sites on the test's own goroutine check `err` and
  `t.Fatalf` immediately, and the 8-goroutine concurrency subtest captures `err` per-goroutine into
  the existing `results` slice, reporting via `t.Errorf` only after `wg.Wait()` on the main test
  goroutine. Confirmed clean under `go test -race ./internal/agents/... -run
  TestSessionNudgeOutputIsPinnedAndStateless -v` (no data races, all subtests pass).
- **IN-02** (`countSkillWorkedExamples`'s section-boundary heuristic was a loose
  `strings.Contains(..., "example")` match) — fixed. The anchor is now
  `strings.HasPrefix(strings.ToLower(line), "## worked example")`, matching the reviewer's
  suggested fix exactly. `TestSkillWorkedExampleCounterIsNotVacuous` still exercises the section
  boundaries (heading outside the section, a `####` heading) and passes.

`go build ./...`, `go vet ./internal/agents/... ./internal/mcp/...`, and the full `TestSkill*` /
`TestSessionNudge*` / `TestHookRegistration*` / `TestClaudeInstallPreservesHooksBlock` /
`TestNudgeText*` suites all pass. No collateral regressions were introduced by either fix — both
diffs are scoped exactly to the flagged functions.

Two warnings and two info items from the prior review were explicitly **not** addressed and remain
open, per the fix report's own stated (and reasonable) disposition: WR-03/WR-04 are genuine
functional gaps but are Claude-Code-runtime behavior outside this repository's code, and IN-01/IN-03
were marked "no action needed" by the reviewer that raised them. They are carried forward below
unchanged so this review does not imply they were silently resolved.

## Warnings

### WR-03: The `resume` SessionStart matcher is registered but demonstrably does not fire — half of the shipped hook configuration is currently dead

**File:** `.claude/hooks/hooks.json:13-21`, `.claude/settings.json:13-21`, evidenced in
`.claude/skills/codegraph/verification/NUDGE-live-session.md:22-40`
**Issue:** Both `hooks.json` and `settings.json` register the nudge script under both the `startup`
and `resume` `SessionStart` matchers, and this phase's own test suite thoroughly verifies the
*script's* and the *registration's* byte-level correctness. But the project's own live-session
rehearsal recorded that resuming a session via `claude --resume` produced **zero**
`SessionStart:resume` hook events in the transcript — the matcher syntax is correct per
documentation, yet the nudge never actually reaches a resumed session. The `resume` entry in both
files is currently inert configuration: it passes every static/unit check this phase added, but
does nothing at runtime. Unchanged since the prior review — no code change was made or expected
here.
**Fix:** Not a repository code fix. Already tracked in `.planning/STATE.md` as an open follow-up:
either confirm this is a Claude Code platform limitation and file it upstream, or find and fix a
project-side config/timing cause, then extend the live-session rehearsal to cover the resume path
before considering NUDGE-01/02 closed for that matcher.

### WR-04: `SKILL.md` was not observed to be surfaced by a fresh session's skill catalog — the skill-routing deliverable is unproven in the one live test that exists for it

**File:** `.claude/skills/codegraph/SKILL.md`, evidenced in
`.claude/skills/codegraph/verification/SKILL-03-rehearsal.md:64-95`
**Issue:** The rehearsal's own "Verdict" section states plainly that the newly-authored skill "was
not surfaced to the captured session at all" and that the correct tool routing observed in that
session is fully explained by two *other*, pre-existing mechanisms (the operator's global
`CLAUDE.md` and the MCP server's own `instructions` string) — not by this phase's `SKILL.md`. The
extensive `skill_claims_drift_test.go` suite verifies the file's internal correctness
(frontmatter, structure, no stale claims, no dead resource pointers) exhaustively, but none of
that establishes that Claude Code ever loads it. Unchanged since the prior review.
**Fix:** Not a repository code fix. Already tracked in `.planning/STATE.md` as an open follow-up —
investigate why a newly-added project skill isn't picked up by a genuinely fresh session's skill
discovery before treating SKILL-03 as fully demonstrated.

## Info

### IN-01: `.claude/hooks/hooks.json` and `.claude/settings.json` are byte-for-byte duplicate files with no automatic sync

**File:** `.claude/hooks/hooks.json`, `.claude/settings.json`
**Issue:** `hooks.json` exists solely as a future `go:embed` source for a not-yet-built Phase 7 and
is never read by Claude Code itself; `settings.json` is what actually runs. The two are currently
identical. The only thing preventing silent drift between them is
`TestHookRegistrationMatchesFragmentAndScript`, which compares exactly the `hooks.SessionStart`
key — if `settings.json` later grows a `permissions` block or other top-level keys (as
`TestClaudeInstallPreservesHooksBlock` implies it will), nothing requires `hooks.json` to track
that. Deliberate, documented tradeoff per the doc comments in `hookpackage_test.go` — left
unaddressed intentionally, as agreed in the prior review-fix pass.
**Fix:** No action needed now; revisit at Phase 7 when `go:embed` is actually wired up.

### IN-03: `skillFrontmatterField` requires the key to start at column zero, with no tolerance for indented YAML

**File:** `internal/mcp/skill_claims_drift_test.go:156-164`
**Issue:** `skillFrontmatterField` matches only `strings.HasPrefix(line, key+":")` with no leading
whitespace trim. Correct for the current flat, unindented frontmatter `SKILL.md` uses; brittle if
frontmatter is ever reformatted with indentation. Fails loudly (`t.Fatalf`) rather than silently
misreading a value, so risk is limited to an unnecessary build break. Left unaddressed
intentionally per the prior review-fix pass ("no action needed unless frontmatter formatting
conventions change").
**Fix:** No action needed unless frontmatter conventions change.

---

_Reviewed: 2026-08-13T00:07:48Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep_
