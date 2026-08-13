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

## Resume nudge (`resume` matcher) — fires; identical output is suppressed

> **Retraction (2026-08-13).** This section previously reported that the `resume` matcher "did
> not fire" and called it "a real, reproducible gap, not a rehearsal methodology error." That
> finding was wrong, and the emphasis was misplaced: it *was* a methodology error. The
> rehearsal's oracle — grepping the resumed session's transcript for `SessionStart` hook events —
> cannot observe this dispatch, so it returned a false negative. Corrected below from a
> seven-probe investigation (`.planning/debug/resolved/resume-matcher-not-firing.md`).

The same session was ended and resumed via `claude --resume <session-id>`. Per the official
Claude Code hooks documentation, this is the documented trigger for the SessionStart `resume`
matcher ("resume: `--resume`, `--continue`, or `/resume`"). `.claude/settings.json` registers the
`resume` matcher for the same script the `startup` matcher uses.

**Result: the `resume` matcher fires. Claude Code suppresses a SessionStart hook's output — both
the context injection and the transcript record — when that output is byte-identical to context
already injected in the same session.** `session-nudge.sh` emits one constant line for every
source, so on resume its output is always an exact repeat of what `startup` already injected, and
is always the suppressed case. The dispatch happens; it simply leaves no trace in the transcript.

Two independent oracles establish this, neither of which the original rehearsal used:

1. **Execution oracle (`probe.log`).** A probe hook that appends its raw stdin payload to a file
   before writing stdout records the dispatch independently of Claude Code. On resume it logs
   `entry=E-resume | source=resume` — the runtime does deliver a `"source":"resume"` SessionStart
   event to the registered command.
2. **Context oracle.** Asking the *resumed* model what it can see. With per-source output the
   resumed model quotes the resume-specific text verbatim; with the shipped constant-output
   script the resumed model quotes the nudge line verbatim, carried forward from `startup`.

The single-variable proof: two probes identical in every respect — project-scoped
`.claude/settings.json`, `${CLAUDE_PROJECT_DIR}`-relative command path, `startup` + `resume`
matchers — except that one emits a *different* string per source and the other emits a *constant*
string. The per-source probe records `SessionStart:resume` in the transcript and injects it. The
constant-output probe executes (proven by `probe.log`) and records nothing. Output identity is
the only variable that changes the outcome.

Also ruled out along the way, each by its own single-variable probe: `${CLAUDE_PROJECT_DIR}`
failing to expand on the resume path (it expands correctly); `.claude/hooks/hooks.json` shadowing
`settings.json` (that file is not read by Claude Code at all, exactly as
`internal/agents/hookpackage_test.go` already documents — its probe entries never executed); and
project-scoped settings being ignored on resume (they are honoured).

**NUDGE-01 holds on the resume path.** The requirement is that the agent *receives* the nudge, not
that a hook re-fires visibly: a resumed session carries the `startup` injection in its context and
can quote it verbatim. Suppressing a byte-identical re-injection is the runtime avoiding redundant
context, and it is why D-07's second matcher produces no observable event here.

**Methodology note for future live-session checks.** Transcript grep is not a valid oracle for
SessionStart dispatch. It is blind to any hook whose output is empty or duplicated — an
empty-stdout SessionStart hook leaves no transcript record either, under *any* matcher including
`startup`. Verify dispatch with a side-channel the hook writes itself, and verify delivery by
asking the session what it can see.

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

Two items surfaced by this rehearsal. The first is now **resolved and was not a gap**; the second
remains open:

1. ~~**Resume-matcher non-firing.**~~ **RESOLVED 2026-08-13 — not a defect.** The `resume` matcher
   fires; Claude Code suppresses the injection and the transcript record because the shipped
   script's output is byte-identical to what `startup` already injected. NUDGE-01 holds on the
   resume path. See the retraction in the resume section above and
   `.planning/debug/resolved/resume-matcher-not-firing.md` for the probe ladder. The lasting
   correction is to the *method*, not the code: transcript grep is not a valid oracle for
   SessionStart dispatch.
2. **Skill-discovery non-listing.** `.claude/skills/codegraph/SKILL.md` was not surfaced in a
   freshly started session's skill catalog despite being correctly placed and committed (see
   `SKILL-03-rehearsal.md`'s Verdict section for detail). Needs investigation into why a
   newly-added project skill isn't picked up by a genuinely fresh session's skill discovery.

Both are recorded in `.planning/STATE.md` as open follow-up items for this milestone rather than
silently dropped.
