# Phase 6: Agent Skill Package — SKILL.md & SessionStart Nudge - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-08-12
**Phase:** 6-Agent Skill Package — SKILL.md & SessionStart Nudge
**Areas discussed:** Transcript-diff verification (SKILL-03), Dogfooding — does this repo install its own copy?, Worked examples beyond the misdirection incident, Nudge message content & cadence, Todo fold decision

---

## Transcript-diff verification (SKILL-03)

| Option | Description | Selected |
|--------|-------------|----------|
| Committed rehearsal transcript | Capture a real before/after session pair, save as markdown/JSON artifact in the phase dir, reviewed once at ship time — mirrors UAT/live-binary verification elsewhere in this repo. Not re-run by CI. | ✓ |
| Scripted/repeatable harness | Build tooling to re-run the transcript diff on demand (headless CLI invocation). More upfront engineering; no precedent in this codebase. | |
| Claude's discretion | Let researcher/planner pick the mechanism. | |

**User's choice:** Committed rehearsal transcript.
**Notes:** Follow-up question — reuse the 2026-08-08 misdirection log as the "before" half, or capture a fresh matched pair?

| Option | Description | Selected |
|--------|-------------|----------|
| Reuse the 2026-08-08 log as "before" | Pair with one fresh "after" capture (skill installed, same class of prompt). Doubles as evidence the skill fixes the exact incident that motivated this milestone. | ✓ |
| Capture a fresh matched pair | Run the same prompt twice in close succession for an apples-to-apples comparison uncontaminated by other repo changes since 2026-08-08. | |

**User's choice:** Reuse the 2026-08-08 log.

---

## Dogfooding — does this repo install its own copy?

| Option | Description | Selected |
|--------|-------------|----------|
| Dogfood into this repo's `.claude/` | `.claude/skills/codegraph/` + `.claude/hooks/hooks.json`, committed. Makes the transcript-diff proof natural; this repo benefits from its own skill immediately. | ✓ |
| Source content lives elsewhere; throwaway install only | Author under `internal/agents/skillfiles/` as canonical; use a temp install only for the proof capture. | |

**User's choice:** Dogfood into `.claude/`.
**Notes:** Follow-up — is `.claude/` also the canonical source Phase 7 embeds, or a separate copy?

| Option | Description | Selected |
|--------|-------------|----------|
| `.claude/` IS the canonical source | One copy, no duplication risk. Phase 7's `go:embed` points directly at `.claude/skills/codegraph/` and `.claude/hooks/hooks.json`. | ✓ |
| Separate embed-source copy | Author under `internal/agents/skillfiles/`, install a copy into `.claude/` mirroring what Phase 7's install command will eventually do. Two copies to keep in sync. | |

**User's choice:** `.claude/` is the canonical source.

---

## Worked examples beyond the misdirection incident

| Option | Description | Selected |
|--------|-------------|----------|
| Impact analysis before a refactor | "What breaks if I change this function's signature?" via `codegraph_impact` over manual call-site grepping. | ✓ |
| Cross-file symbol lookup with dynamic dispatch | The case grep structurally cannot follow but explore's call-graph resolution can. | |
| Let me describe specific scenarios | User has particular examples from daily use. | |

**User's choice:** Impact analysis before a refactor.
**Notes:** Follow-up — add a third example (cross-file dynamic dispatch) or stop at 2?

| Option | Description | Selected |
|--------|-------------|----------|
| Stop at 2 | Misdirection incident + impact analysis; keeps skill tighter. | |
| Add a third: cross-file dynamic-dispatch lookup | Rounds out the 2-3 example range at the high end. | ✓ |

**User's choice:** Add the third example.

---

## Nudge message content & cadence

| Option | Description | Selected |
|--------|-------------|----------|
| Minimal pointer | One line: prefer `codegraph_explore` over grep for where-is-X/how-does-Y. No tool list, no examples. | ✓ |
| Slightly richer nudge | Same pointer plus a fallback shell-command reminder. | |
| Let me write the exact wording | User has specific phrasing in mind. | |

**User's choice:** Minimal pointer.
**Notes:** Follow-up — fire on both SessionStart matchers (startup/resume), or startup only?

| Option | Description | Selected |
|--------|-------------|----------|
| Fire on startup only | New session gets the nudge; resume does not re-show it. | |
| Fire on both startup and resume | Every SessionStart event gets exactly one nudge for that event; simpler hook logic. | ✓ |

**User's choice:** Fire on both.

---

## Todo fold decision

| Option | Description | Selected |
|--------|-------------|----------|
| Fold it in | Mark `2026-08-08-author-a-codegraph-usage-skill-for-agents.md` as addressed by Phase 6; close once shipped. | ✓ |
| Reviewed but not folded | Leave it open/separate from this phase's formal scope. | |

**User's choice:** Fold it in.

---

## Claude's Discretion

- Exact markdown structure/headers within SKILL.md beyond decision-table-first, worked-examples-second, catalog-deferred-to-resources.
- Mechanism for the nudge script's `.codegraph/`-existence check (shell one-liner vs. small Go helper).
- Exact `hooks.json` event/matcher JSON shape (follows documented Claude Code hooks schema).
- Where within `.claude/skills/codegraph/` the rehearsal transcript artifact is filed.

## Deferred Ideas

None — discussion stayed within phase scope.
