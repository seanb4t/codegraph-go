---
phase: 06-agent-skill-package-skill-md-sessionstart-nudge
fixed_at: 2026-08-12T00:00:00Z
review_path: .planning/phases/06-agent-skill-package-skill-md-sessionstart-nudge/06-REVIEW.md
iteration: 2
findings_in_scope: 4
fixed: 0
skipped: 4
status: none_fixed
---

# Phase 6: Code Review Fix Report

**Fixed at:** 2026-08-12T00:00:00Z
**Source review:** .planning/phases/06-agent-skill-package-skill-md-sessionstart-nudge/06-REVIEW.md
**Iteration:** 2

**Summary:**
- Findings in scope: 4 (fix_scope: all — WR-03, WR-04, IN-01, IN-03; WR-01, WR-02, IN-02 were
  already fixed in iteration 1's commits `156cda4`/`15e27da` and no longer appear in this REVIEW.md)
- Fixed: 0
- Skipped: 4 — independently re-verified against source before skipping (not carried forward on
  faith). See reasoning per finding below.

## Fixed Issues

None — all findings in scope were re-confirmed as not fixable in this repository (or explicitly
marked no-action-needed by the reviewer) and skipped.

## Skipped Issues

### WR-03: The `resume` SessionStart matcher is registered but demonstrably does not fire

**File:** `.claude/hooks/hooks.json:13-21`, `.claude/settings.json:13-21`
**Reason:** Independently re-verified: `diff .claude/hooks/hooks.json .claude/settings.json`
confirms the two files are still byte-identical, and both register the `resume` matcher with
schema-correct syntax. This is a Claude-Code-runtime-behavior gap — whether `SessionStart:resume`
actually fires is controlled entirely by the Claude Code client, outside any code path this
repository owns. Confirmed still tracked as an open follow-up in `.planning/STATE.md` (line 284:
"Not fixed in Phase 6 — needs its own debug session to determine whether this is a Claude Code
runtime limitation or a project-side gap"). No in-repo edit exists that would change platform
hook-dispatch behavior; fabricating one (e.g. altering matcher syntax not shown to be wrong, or
adding speculative config) would misrepresent the actual state of the platform behavior without
addressing the underlying gap. Same disposition as iteration 1.
**Original issue:** `NUDGE-live-session.md`'s live-session rehearsal recorded zero
`SessionStart:resume` hook events when resuming a session via `claude --resume`, despite the
matcher being registered correctly in both files.

### WR-04: `SKILL.md` was not observed to be surfaced by a fresh session's skill catalog

**File:** `.claude/skills/codegraph/SKILL.md`
**Reason:** Independently re-verified: read `skill_claims_drift_test.go`'s frontmatter/structure
checks (lines 145-168 and surrounding), which continue to validate `SKILL.md`'s internal
correctness exhaustively — that is not in question. The gap is whether Claude Code's skill catalog
*loads* the file at all, which the rehearsal's own "Verdict" section attributes to the client's
discovery mechanism, not to any defect in the file's content. Confirmed still tracked in
`.planning/STATE.md` (line 285: "Needs investigation into why project-skill discovery didn't pick
up a newly-added, correctly-placed `.claude/skills/*/SKILL.md` in a fresh session"). No
repository-side code change can force the client to load the file; this is a platform-discovery
question requiring its own debug session, not a fix applicable via this review-fix pass. Same
disposition as iteration 1.
**Original issue:** `SKILL-03-rehearsal.md`'s "Verdict" states the newly-authored skill "was not
surfaced to the captured session at all," and the correct routing observed in that session is fully
explained by two other, pre-existing mechanisms (operator's global `CLAUDE.md`, MCP server's own
`instructions` string) unrelated to this phase's `SKILL.md`.

### IN-01: `.claude/hooks/hooks.json` and `.claude/settings.json` are byte-for-byte duplicate files with no automatic sync

**File:** `.claude/hooks/hooks.json`, `.claude/settings.json`
**Reason:** Independently re-verified the files are still identical (`diff` exit 0) and that the
REVIEW.md's own Fix section for this finding reads "No action needed now; revisit at Phase 7 when
`go:embed` is actually wired up." This is the reviewer's explicit disposition, not a defect awaiting
a fix. Introducing a sync mechanism now would be scope creep beyond a review-fix pass and would
preempt Phase 7's own `go:embed` design, which is where the reviewer says this belongs. Same
disposition as iteration 1.
**Original issue:** `hooks.json` (a future `go:embed` source, not yet read by Claude Code) and
`settings.json` (what actually runs) are currently identical; only
`TestHookRegistrationMatchesFragmentAndScript`'s narrow `hooks.SessionStart`-key comparison guards
against drift, and would not catch drift if `settings.json` grows additional top-level keys.

### IN-03: `skillFrontmatterField` requires the key to start at column zero, with no tolerance for indented YAML

**File:** `internal/mcp/skill_claims_drift_test.go:156-164`
**Reason:** Independently re-read the function (lines 153-168): it matches only
`strings.HasPrefix(line, key+":")` with no leading-whitespace trim, exactly as described. The
REVIEW.md's own Fix section reads "No action needed unless frontmatter conventions change." No code
change was requested by the reviewer; the current behavior fails loudly (`t.Fatalf`) rather than
silently misreading a value if frontmatter formatting ever changes, which limits the risk to an
unnecessary build break rather than a wrong answer. Left unchanged per the reviewer's own guidance.
Same disposition as iteration 1.
**Original issue:** `skillFrontmatterField` matches only `strings.HasPrefix(line, key+":")` with no
leading-whitespace tolerance — correct for the current flat, unindented frontmatter `SKILL.md` uses;
would be brittle if frontmatter is ever reformatted with indentation.

---

_Fixed: 2026-08-12T00:00:00Z_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 2_
