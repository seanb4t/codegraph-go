---
phase: 06-agent-skill-package-skill-md-sessionstart-nudge
fixed_at: 2026-08-13T00:03:40Z
review_path: .planning/phases/06-agent-skill-package-skill-md-sessionstart-nudge/06-REVIEW.md
iteration: 1
findings_in_scope: 7
fixed: 2
skipped: 5
status: partial
---

# Phase 6: Code Review Fix Report

**Fixed at:** 2026-08-13T00:03:40Z
**Source review:** .planning/phases/06-agent-skill-package-skill-md-sessionstart-nudge/06-REVIEW.md
**Iteration:** 1

**Summary:**
- Findings in scope: 7 (fix_scope: all — WR-01, WR-02, WR-03, WR-04, IN-01, IN-02, IN-03)
- Fixed: 2 (WR-01, WR-02, applied together — same commit — plus IN-02 in a separate commit)
- Skipped: 5 (WR-03, WR-04 — runtime behavior gaps outside this repo's code; IN-01, IN-03 — reviewer explicitly marked "no action needed"/"deliberate tradeoff, not to fix now")

## Fixed Issues

### WR-01: `runSessionNudge`'s env-injection branch does not strip a pre-existing `CLAUDE_PROJECT_DIR`

**Files modified:** `internal/agents/hookpackage_test.go`
**Commit:** 156cda4
**Applied fix:** Both branches of `runSessionNudge` now build the environment by filtering any
inherited `CLAUDE_PROJECT_DIR=` entry out of `os.Environ()` first, then (for `useEnv == true`)
appending the test's own override. This removes the asymmetry the reviewer flagged — the override is
now deterministic regardless of what the invoking process happens to carry, matching the
`useEnv == false` branch's existing behavior. Verified with `go test -race ./internal/agents/...`
(all `TestSessionNudge*` subtests pass).

### WR-02: `t.Fatalf`/`t.Helper` called from spawned goroutines in the concurrency subtest

**Files modified:** `internal/agents/hookpackage_test.go`
**Commit:** 156cda4 (same commit as WR-01 — both fixes live inside `runSessionNudge` and its four
call sites and could not be separated without an intermediate broken state)
**Applied fix:** `runSessionNudge` no longer calls `t.Fatalf` on the non-`*exec.ExitError` path;
it now returns `(stdout, stderr, exitCode string/int, err error)` instead. All four call sites were
updated: the three call sites running on the test's own goroutine now check `err != nil` and
`t.Fatalf` immediately (same failure behavior as before, just moved to the correct goroutine); the
concurrency subtest's 8 spawned goroutines now capture `err` into the existing per-goroutine
`results` slice and the main test goroutine reports any captured error via `t.Errorf` after
`wg.Wait()` returns — satisfying the `testing` package's documented requirement that
`Fatal`/`FailNow` only be called from the goroutine running the test function. Verified with
`go vet ./internal/agents/...` and `go test -race ./internal/agents/...`.

### IN-02: `countSkillWorkedExamples`'s section-boundary heuristic is a loose case-insensitive substring match on "example"

**Files modified:** `internal/mcp/skill_claims_drift_test.go`
**Commit:** 15e27da
**Applied fix:** Tightened the anchor from `strings.HasPrefix(line, "## ") &&
strings.Contains(strings.ToLower(line), "example")` to
`strings.HasPrefix(strings.ToLower(line), "## worked example")`, per the reviewer's suggested fix.
This still matches the current `"## Worked examples"` heading (case-insensitive prefix) but would no
longer be fooled by an unrelated heading such as `"## Bad examples of misrouted queries"` appearing
earlier in the document. Verified with `go vet ./internal/mcp/...` and
`go test ./internal/mcp/... -run Skill` (all skill-drift tests, including
`TestSkillWorkedExampleCounterIsNotVacuous`, pass).

## Skipped Issues

### WR-03: The `resume` SessionStart matcher is registered but demonstrably does not fire

**File:** `.claude/hooks/hooks.json:13-21`, `.claude/settings.json:13-21`
**Reason:** This is a Claude-Code-runtime-behavior gap, not a bug in this repository's own code — the
matcher configuration is byte-correct per the platform's documented schema and both files are already
verified identical by `TestHookRegistrationMatchesFragmentAndScript`. Whether `SessionStart:resume`
actually fires is controlled entirely by the Claude Code client, outside this repo's code paths.
Already tracked as an open follow-up in `.planning/STATE.md`; no in-repo code change can fix it, and
fabricating a workaround here (e.g. papering over the config) would misrepresent the actual state of
the platform behavior.
**Original issue:** `NUDGE-live-session.md`'s own rehearsal recorded zero `SessionStart:resume` hook
events when resuming a session, despite the matcher being registered correctly in both files.

### WR-04: `SKILL.md` was not observed to be surfaced by a fresh session's skill catalog

**File:** `.claude/skills/codegraph/SKILL.md`
**Reason:** Same disposition as WR-03 — a Claude-Code-runtime discovery/loading behavior gap, not a
defect in `SKILL.md`'s content or structure (which the extensive `skill_claims_drift_test.go` suite
already verifies exhaustively). Whether Claude Code's skill catalog loads this file is controlled by
the client, outside this repo's code. Already tracked as an open follow-up in `.planning/STATE.md`;
no in-repo code change can fix it.
**Original issue:** `SKILL-03-rehearsal.md`'s own "Verdict" states the newly-authored skill "was not
surfaced to the captured session at all" and that observed correct routing is fully explained by two
other, pre-existing mechanisms unrelated to this phase's `SKILL.md`.

### IN-01: `.claude/hooks/hooks.json` and `.claude/settings.json` are byte-for-byte duplicate files with no automatic sync

**File:** `.claude/hooks/hooks.json`, `.claude/settings.json`
**Reason:** The reviewer explicitly classified this as "a deliberate, documented tradeoff (per the
doc comments in `hookpackage_test.go`), not a hidden defect — noted here as a maintainability risk to
keep in mind at Phase 7, not something to fix now." No fix was requested; applying one now (e.g.
introducing a sync mechanism or additional guard) would be scope creep beyond a review-fix pass and
would preempt Phase 7's own `go:embed` design, which is explicitly where this is meant to be
addressed.
**Original issue:** `hooks.json` (a future `go:embed` source) and `settings.json` (what Claude Code
actually reads) are currently identical, but nothing besides
`TestHookRegistrationMatchesFragmentAndScript`'s narrow `hooks.SessionStart`-key comparison would
catch future drift if `settings.json` grows additional top-level keys.

### IN-03: `skillFrontmatterField` requires the key to start at column zero, with no tolerance for indented YAML

**File:** `internal/mcp/skill_claims_drift_test.go:156-164`
**Reason:** The reviewer's own Fix section reads "Optional — no action needed unless frontmatter
formatting conventions change; flagging for awareness only." No code change was requested; the
current behavior fails loudly (`t.Fatalf("... has no name field")`) rather than silently misreading a
value if frontmatter formatting ever changes, which the reviewer noted limits the risk to an
unnecessary build break rather than a wrong answer. Left unchanged per the reviewer's own guidance.
**Original issue:** `skillFrontmatterField` matches only `strings.HasPrefix(line, key+":")` with no
leading-whitespace tolerance — correct for the current flat, unindented frontmatter `SKILL.md` uses.

---

_Fixed: 2026-08-13T00:03:40Z_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 1_
