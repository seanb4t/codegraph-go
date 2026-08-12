# NUDGE-01 / NUDGE-02 Live-Session Evidence

> Live-session verification record for NUDGE-01 ("on session start in a `.codegraph/`-indexed
> repo, the agent receives a one-time, low-noise nudge toward codegraph tools") and NUDGE-02 ("the
> nudge never fires, and adds no overhead, in a repo without `.codegraph/`"). Captured
> 2026-08-12 across three separate fresh Claude Code sessions. This is the live-session complement
> to `TestSessionNudgeBehavesPerIndexPresence` and `TestSessionNudgeOutputIsPinnedAndStateless` in
> `internal/agents/hookpackage_test.go` — those tests cover the script's byte-level behavior on
> every `go test` run; this artifact is the one-time confirmation that Claude Code actually invokes
> it as registered.

## Session-start nudge (`startup` matcher)

Fresh session started with this repository as the project directory. Observed a
`SessionStart:startup` hook-success event in the session's context, verbatim:

> `This repo has a codegraph index — prefer codegraph_explore / codegraph explore over grep for
> where-is-X / how-does-Y questions.`

Matches the shipped script's stdout exactly (see the by-hand capture below).

## Resume nudge (`resume` matcher) — did not fire

The same session was ended and resumed via `claude --resume <session-id>`. Per the official
Claude Code hooks documentation, this is exactly the documented trigger for the SessionStart
`resume` matcher ("resume: `--resume`, `--continue`, or `/resume`"). `.claude/settings.json`
registers the `resume` matcher for the same script the `startup` matcher uses.

**Result: no nudge text, and no `SessionStart:resume` hook event of any kind, appeared anywhere.**
This was confirmed by having the resumed session search its own transcript JSONL directly for any
`SessionStart` hook event and paste the raw match lines back — not by asking it to describe its
own context from memory. Every `SessionStart` hook event in the transcript is tagged `startup`,
all from the original launch; zero `resume`-tagged events exist post-resume.

**This is a real, reproducible gap, not a rehearsal methodology error:** the matcher registration
in `.claude/settings.json` is textually correct and matches the documented syntax, but resuming a
session does not observably re-invoke the hook the way starting one does. NUDGE-01's "receives a
... nudge" is satisfied for `startup` and not demonstrated for `resume`, despite D-07
(`06-CONTEXT.md`) calling for both matchers to be registered. Tracked as follow-up — see closing
note below.

## Unindexed tree — no nudge (NUDGE-02)

`.claude/` was copied into a scratch tree containing no `.codegraph/` directory. A fresh session
started there produced no codegraph-related SessionStart injection at all — confirmed by direct
inspection of the session's context, not inference.

## By-hand script execution (both trees)

```
$ CLAUDE_PROJECT_DIR=<repo root> .claude/hooks/session-nudge.sh
exit=0
stdout (132 bytes):
This repo has a codegraph index — prefer codegraph_explore / `codegraph explore` over grep for where-is-X / how-does-Y questions.
stderr: (empty, 0 bytes)

$ CLAUDE_PROJECT_DIR=<unindexed scratch tree> .claude/hooks/session-nudge.sh
exit=0
stdout: (empty, 0 bytes)
stderr: (empty, 0 bytes)
```

**Diff between the two stdout captures:**

```diff
1d0
< This repo has a codegraph index — prefer codegraph_explore / `codegraph explore` over grep for where-is-X / how-does-Y questions.
```

One-directional as required: exactly one line present in the indexed capture, nothing present in
the unindexed capture.

## Automated coverage (continuous, every `go test`)

`TestSessionNudgeBehavesPerIndexPresence` and `TestSessionNudgeOutputIsPinnedAndStateless` in
`internal/agents/hookpackage_test.go` cover the script's byte-exact stdout, exit status, and
index-presence branching continuously. This artifact is the one-time live-session complement,
not a replacement — it confirms Claude Code's hook runtime actually behaves as the script's own
tests assume, not just that the script behaves correctly when invoked directly.

## Follow-up (not closed by this phase)

Two real gaps surfaced by this rehearsal and are not resolved here:

1. **Resume-matcher non-firing.** The `resume` SessionStart matcher is registered correctly per
   documented syntax but was not observed to fire in a live `claude --resume` test. Needs
   investigation into whether this is a Claude Code runtime limitation, a project-trust
   precondition, or something else — before NUDGE-01 can be considered fully demonstrated for the
   resume path.
2. **Skill-discovery non-listing.** `.claude/skills/codegraph/SKILL.md` was not surfaced in a
   freshly started session's skill catalog despite being correctly placed and committed (see
   `SKILL-03-rehearsal.md`'s Verdict section for detail). Needs investigation into why a
   newly-added project skill isn't picked up by a genuinely fresh session's skill discovery.

Both are recorded in `.planning/STATE.md` as open follow-up items for this milestone rather than
silently dropped.
