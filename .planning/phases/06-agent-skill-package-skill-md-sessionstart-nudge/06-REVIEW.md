---
phase: 06-agent-skill-package-skill-md-sessionstart-nudge
reviewed: 2026-08-12T23:52:47Z
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
  warning: 4
  info: 3
  total: 7
status: issues_found
---

# Phase 6: Code Review Report

**Reviewed:** 2026-08-12T23:52:47Z
**Depth:** deep
**Files Reviewed:** 9
**Status:** issues_found

## Summary

This phase ships the SessionStart nudge (`session-nudge.sh` + `hooks.json`/`settings.json`
registration) and the `.claude/skills/codegraph/SKILL.md` agent skill, backed by an unusually
thorough drift-guard test suite (`internal/agents/hookpackage_test.go`,
`internal/mcp/skill_claims_drift_test.go`) and a large, honestly-written mutation-proof and
live-session evidence trail. The shell script and skill markdown themselves are small, careful, and
free of the classic anti-patterns (no unquoted expansion that matters, no eval, no secrets, no dead
code). `go vet` and `shellcheck` are clean (one SC2016 info-level false positive on an intentionally
single-quoted literal).

The defects found here are concentrated in the test harness and in two functional gaps the
project's own evidence artifacts already surface but that a reviewer should still flag rather than
wave through: (1) the SessionStart `resume` matcher is registered correctly but demonstrably does
not fire in a live session, and (2) `SKILL.md` was not observed to be picked up by a fresh session's
skill catalog at all — meaning two of this phase's three deliverables are unproven or non-functional
in production as shipped, despite the code artifacts themselves being correct. Both are already
tracked as open follow-ups (not silently hidden), but "already disclosed" is not the same as
"resolved," and a code review should not downgrade a real functional gap to invisible just because
the authoring session wrote it down honestly.

Additionally, `internal/agents/hookpackage_test.go`'s `runSessionNudge` helper has a real,
demonstrable env-var precedence bug (asymmetric handling between its two branches) and calls
`t.Fatalf` from spawned goroutines in a concurrency subtest, both of which undermine the very tests
whose job is to make NUDGE-01/02's guarantees trustworthy.

## Warnings

### WR-01: `runSessionNudge`'s env-injection branch does not strip a pre-existing `CLAUDE_PROJECT_DIR`, unlike its sibling branch

**File:** `internal/agents/hookpackage_test.go:62-75`
**Issue:** When `useEnv` is `true`, the helper does:
```go
cmd.Env = append(os.Environ(), "CLAUDE_PROJECT_DIR="+dir)
```
This appends the override to the *end* of the inherited environment without first removing any
existing `CLAUDE_PROJECT_DIR` entry. The `useEnv == false` branch, immediately below, does the
opposite correctly — it explicitly filters out any inherited `CLAUDE_PROJECT_DIR=` entry before
setting `cmd.Dir`. The two branches are inconsistent for no stated reason.

If the parent process (`go test`, or whatever invoked it — e.g. a CI wrapper, or a nested Claude
Code session) already exports `CLAUDE_PROJECT_DIR`, `sh`'s `${CLAUDE_PROJECT_DIR:-.}` expansion
resolves via `getenv()`, and glibc/most libc implementations return the **first** matching entry in
the environment array, not the last. `append(os.Environ(), ...)` places the new value last, so the
*inherited* value wins, not the test's intended override. Concretely, if the inherited
`CLAUDE_PROJECT_DIR` happens to point at this very repository (which does have `.codegraph/`), the
"no codegraph entry at all, env set" sub-test and the "codegraph dir present, env set" sub-test would
both silently exercise the wrong directory — the tempdir the test seeded is never actually looked at,
yet the assertions could still pass by coincidence (this repo's own `.codegraph/` presence), masking
the fact that the override never took effect. This is exactly the kind of test that "looks green but
proves nothing" the surrounding non-vacuity tests in this same phase are otherwise careful to guard
against.
**Fix:**
```go
if useEnv {
    var env []string
    for _, kv := range os.Environ() {
        if strings.HasPrefix(kv, "CLAUDE_PROJECT_DIR=") {
            continue
        }
        env = append(env, kv)
    }
    cmd.Env = append(env, "CLAUDE_PROJECT_DIR="+dir)
} else {
    ...
}
```
Filter first in both branches so the helper's env is deterministic regardless of what the invoking
process happens to carry.

### WR-02: `t.Fatalf`/`t.Helper` called from spawned goroutines in the concurrency subtest

**File:** `internal/agents/hookpackage_test.go:226-253` (calling into `runSessionNudge` at
`internal/agents/hookpackage_test.go:54-93`)
**Issue:** `TestSessionNudgeOutputIsPinnedAndStateless`'s `"concurrency and statelessness"` subtest
spawns 8 goroutines, each calling `runSessionNudge(t, dir, true)` directly:
```go
go func(i int) {
    defer wg.Done()
    stdout, _, exit := runSessionNudge(t, dir, true)
    ...
}(i)
```
`runSessionNudge` itself calls `t.Fatalf` on the non-`*exec.ExitError` branch (line ~90) and
`t.Helper()` at its top. Per the `testing` package's documented contract, `FailNow`/`Fatal`/`Fatalf`
"must be called from the goroutine running the test function, not from other goroutines created
during the test" — calling it from another goroutine "will not run subsequent goroutines" or the
harness's subsequent assertions correctly. In the common path this never fires (the exec succeeds or
exits non-zero, both handled without `Fatalf`), so in practice this test currently passes, but the
guard is one flaky `exec.Command` failure (e.g. a permissions hiccup, `ETXTBSY` under heavy parallel
load, or a transient resource-exhaustion error from `cmd.Run()`) away from producing a confusing,
possibly-hung, or silently-incomplete test run instead of a clean, attributable failure.
**Fix:** Have the goroutine capture the error and report it back through the `results` slice (already
used for `stdout`/`exit`) instead of routing it through `t.Fatalf` inside the goroutine, e.g. change
`runSessionNudge` to return an `error` for the non-ExitError case and let the *calling* goroutine's
owner (the main test body, after `wg.Wait()`) call `t.Fatalf` there.

### WR-03: The `resume` SessionStart matcher is registered but demonstrably does not fire — half of the shipped hook configuration is currently dead

**File:** `.claude/hooks/hooks.json:13-21`, `.claude/settings.json:13-21`, evidenced in
`.claude/skills/codegraph/verification/NUDGE-live-session.md:22-40`
**Issue:** Both `hooks.json` and `settings.json` register the nudge script under both the `startup`
and `resume` `SessionStart` matchers, and `TestHookRegistrationMatchesFragmentAndScript` /
`TestSessionNudgeBehavesPerIndexPresence` thoroughly verify the *script's* and the *registration's*
byte-level correctness. But the project's own live-session rehearsal (`NUDGE-live-session.md`)
recorded that resuming a session via `claude --resume` produced **zero** `SessionStart:resume` hook
events in the transcript — the matcher syntax is correct per documentation, yet the nudge never
actually reaches a resumed session. This means the `resume` entry in both files is currently
inert configuration: it passes every static/unit check this phase added, but does nothing at
runtime. The phase's own D-07 requirement ("both matchers registered") is met textually, not
behaviorally.
**Fix:** This is a genuine open item already tracked in `STATE.md` as follow-up, which is the right
place for the actual fix/investigation. Flagging it here so it is not lost as "handled" — either (a)
confirm this is a Claude Code platform limitation and file it upstream, or (b) if a config or timing
issue on this project's side is found, fix it and extend the live-session rehearsal to cover the
resume path before considering NUDGE-01/02 closed.

### WR-04: `SKILL.md` was not observed to be surfaced by a fresh session's skill catalog — the skill-routing deliverable is unproven in the one live test that exists for it

**File:** `.claude/skills/codegraph/SKILL.md`, evidenced in
`.claude/skills/codegraph/verification/SKILL-03-rehearsal.md:64-95`
**Issue:** The rehearsal's own "Verdict" section states plainly that the newly-authored skill "was
not surfaced to the captured session at all" and that the correct tool routing observed in that
session is fully explained by two *other*, pre-existing mechanisms (the operator's global
`CLAUDE.md` and the MCP server's own `instructions` string) — not by this phase's `SKILL.md`. The
rehearsal is careful to say it "cannot cleanly attribute the correct routing to the phase's own
artifact." That is a meaningful gap for a deliverable whose entire purpose is agent-facing routing:
as far as this phase's own evidence shows, `SKILL.md` may not be doing anything at all yet. The
extensive `skill_claims_drift_test.go` suite verifies the file's internal correctness (frontmatter,
structure, no stale claims) exhaustively, but none of that establishes that Claude Code ever loads
it.
**Fix:** Same disposition as WR-03 — already tracked as follow-up in `STATE.md`, correctly not
silently dropped. Recording it here so the review does not imply the skill-discovery mechanism was
verified when the project's own evidence says it was not.

## Info

### IN-01: `.claude/hooks/hooks.json` and `.claude/settings.json` are byte-for-byte duplicate files with no automatic sync

**File:** `.claude/hooks/hooks.json`, `.claude/settings.json`
**Issue:** `hooks.json` exists solely as a future `go:embed` source for a not-yet-built Phase 7 and is
never read by Claude Code itself; `settings.json` is what actually runs. The two are currently
identical (`diff` confirms zero bytes differ). The only thing preventing silent drift between them is
`TestHookRegistrationMatchesFragmentAndScript`, which compares exactly the `hooks.SessionStart` key —
if `settings.json` later grows a `permissions` block or other top-level keys (as
`TestClaudeInstallPreservesHooksBlock` implies it will, via `addClaudeAllowPermission`), nothing
requires `hooks.json` to track that, and nothing currently would notice if a future edit touched one
file's `SessionStart` block by hand without the other (short of running the test). This is a
deliberate, documented tradeoff (per the doc comments in `hookpackage_test.go`), not a hidden defect
— noted here as a maintainability risk to keep in mind at Phase 7, not something to fix now.

### IN-02: `countSkillWorkedExamples`'s section-boundary heuristic is a loose case-insensitive substring match on "example"

**File:** `internal/mcp/skill_claims_drift_test.go:87-111`
**Issue:** The worked-examples section is located by scanning for the first `"## "` heading whose
text `strings.Contains(strings.ToLower(line), "example")`. This works correctly against the current
`SKILL.md` (`"## Worked examples"`), but the heuristic would also match an unrelated heading such as
`"## Bad examples of misrouted queries"` if one were ever added before the real worked-examples
section, silently redefining what gets counted. Low risk today (the non-vacuity test only exercises
the intended shape), but worth a stricter anchor (e.g. requiring the heading to *start with* "worked
example") if the document grows more sections.
**Fix:** Optional — tighten the match to `strings.HasPrefix(strings.ToLower(line), "## worked example")` or similar if more `"## "` sections are added later.

### IN-03: `skillFrontmatterField` requires the key to start at column zero, with no tolerance for indented YAML

**File:** `internal/mcp/skill_claims_drift_test.go:156-164`
**Issue:** `skillFrontmatterField` matches only `strings.HasPrefix(line, key+":")` with no leading
whitespace trim. This is correct for the current flat, unindented frontmatter `SKILL.md` uses, but is
brittle if frontmatter is ever reformatted with indentation (e.g. under a nested key, or via a
formatter that adds leading spaces) — the field would silently be reported as absent
(`t.Fatalf("... has no name field")`), which at least fails loudly rather than silently misreading a
value, so the risk is limited to an unnecessary build break rather than a wrong answer.
**Fix:** Optional — no action needed unless frontmatter formatting conventions change; flagging for
awareness only.

---

_Reviewed: 2026-08-12T23:52:47Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep_
