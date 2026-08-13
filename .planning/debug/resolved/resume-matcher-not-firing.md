---
status: resolved
trigger: "SessionStart resume matcher (.claude/hooks/hooks.json / .claude/settings.json) is registered but does not fire on `claude --resume`"
created: 2026-08-12
updated: 2026-08-13
---

# Debug Session: resume-matcher-not-firing

## Current Focus

status: resolved

bug_class: >
  Bohrbug — deterministic and fully reproducible (100% non-firing across every observed resume,
  machine-wide). Route: deterministic reproduction first. SBFL skipped (no Go test suite covers
  the Claude Code hook runtime; the suspect component is a third-party closed-source binary).

known_pattern_candidate: (none — knowledge base had no overlapping entry)

root_cause_summary: >
  AND-gate, both conditions required. (1) Claude Code 2.1.229 suppresses a SessionStart hook's
  context injection AND its transcript `hook_success` record when the hook's stdout is
  byte-identical to context already injected in that session; `session-nudge.sh` emits one
  constant line for every source, so its `resume` dispatch is always the suppressed case.
  (2) The Phase 6 rehearsal's oracle was a transcript grep, which is structurally blind to a
  suppressed dispatch and returned a false negative. The registration, the script, and NUDGE-01
  on the resume path were never broken. Full statement in Resolution.

human_verify: >
  2026-08-13 — user waived the interactive `/resume` spot-check and directed that the automated
  proof be trusted: the TDD red->green retraction test, the 6/6 mutation kill with zero survivors,
  the 3-source SessionStart census, and the probe 5 vs probe 7 single-variable comparison. Treated
  as human-verify PASSED.

next_action: >
  None — session complete. Retraction committed (.claude/skills/codegraph/verification/
  NUDGE-live-session.md, .planning/STATE.md, internal/agents/hookpackage_test.go), session
  archived to .planning/debug/resolved/, knowledge-base entry appended.

## Symptoms

**Expected behavior:** The SessionStart `resume` matcher, registered in `.claude/settings.json`
for the same `session-nudge.sh` script the `startup` matcher uses, should fire when a session is
resumed via `claude --resume <session-id>`, `claude --continue`, or `/resume` — per Claude Code's
documented hooks reference ("resume: `--resume`, `--continue`, or `/resume`"). This would produce
a `SessionStart:resume` hook-success event and re-emit the codegraph nudge text in the resumed
session's context, matching what the `startup` matcher does on initial launch. D-07
(`06-CONTEXT.md`) explicitly calls for both matchers to be registered so NUDGE-01 ("agent receives
a one-time, low-noise nudge toward codegraph tools") holds across both entry paths.

**Actual behavior:** No `SessionStart:resume` hook event of any kind appears anywhere in the
resumed session — confirmed by having the resumed session search its own transcript JSONL
directly for any `SessionStart` hook event and paste the raw matching lines back (not by asking it
to describe its own context from memory). Every `SessionStart` hook event in the transcript is
tagged `startup`, all from the original launch; zero `resume`-tagged events exist post-resume. No
nudge text appears either.

**Error messages:** None. This is silent non-firing, not a crash or visible error. The script
itself runs cleanly when invoked by hand (`exit=0`, correct stdout, empty stderr) for both the
indexed and unindexed cases — see by-hand capture below. The gap is specifically that Claude Code's
hook runtime does not appear to invoke the script under the `resume` matcher during a real
`claude --resume` session, even though the matcher registration in `.claude/settings.json` is
textually correct and matches the documented syntax.

**Timeline:** First observed 2026-08-12 during a Phase 6 live-session rehearsal
(`.claude/skills/codegraph/verification/NUDGE-live-session.md`). This is the first live-session
verification attempt for the `resume` matcher — it was never previously confirmed working, so
there is no known "last good" state to bisect against. The `startup` matcher was independently
confirmed working in the same rehearsal, in the same repo, using the same underlying script.

**Reproduction steps:**
1. Start a fresh Claude Code session with this repository (`.codegraph/`-indexed) as the project
   directory.
2. Confirm the `SessionStart:startup` hook fires — the nudge text ("This repo has a codegraph
   index — prefer codegraph_explore / `codegraph explore` over grep for where-is-X /
   how-does-Y questions.") appears in the session's injected context.
3. End that session.
4. Resume it via `claude --resume <session-id>`.
5. In the resumed session, search its own transcript JSONL directly for `SessionStart` hook
   events (not self-report from memory).
6. Observe: only `startup`-tagged events exist, all timestamped to the original launch. No
   `resume`-tagged event exists; no nudge text appears in the resumed session's context.

## Evidence

- timestamp: 2026-08-12 (Phase 6 rehearsal)
  source: .claude/skills/codegraph/verification/NUDGE-live-session.md
  finding: >
    Live 3-session rehearsal. `startup` matcher fires correctly (verified via raw
    SessionStart:startup hook-success event, text matches script stdout exactly). `resume`
    matcher registered identically in `.claude/settings.json` but produces zero SessionStart
    events of any kind on a real `claude --resume <session-id>` — confirmed via direct
    transcript grep, not self-report. By-hand script execution (`CLAUDE_PROJECT_DIR=<repo root>
    .claude/hooks/session-nudge.sh`) is clean in both the indexed and unindexed trees: exit=0,
    correct stdout in the indexed case (132 bytes, byte-exact against automated test
    expectations), empty stdout in the unindexed case, empty stderr both times. So the script
    itself is not implicated — the gap is in whether Claude Code's hook runtime invokes it under
    the `resume` matcher at all.

- timestamp: 2026-08-12
  source: .planning/STATE.md (Blockers/Concerns)
  finding: >
    Recorded as an open follow-up, not closed by Phase 6: "NUDGE-01 resume-matcher does not
    observably fire ... Needs investigation into whether this is a Claude Code runtime
    limitation, a project-side gap." Explicitly deferred to its own debug session rather than
    fixed inline, since Phase 6's scope was building the nudge, not chasing this gap.

- timestamp: 2026-08-12 (investigation)
  checked: Knowledge base `.planning/debug/knowledge-base.md`
  found: >
    Single entry (`perf-gate-throughput-regress`); no keyword overlap with hooks/SessionStart/
    matcher/resume. No known-pattern candidate.
  implication: No prior art. Investigate from first principles.

- timestamp: 2026-08-12 (investigation)
  checked: >
    Global census of SessionStart hook dispatches across EVERY Claude Code transcript on this
    machine — ~180 project directories under `~/.claude/projects/`, 293 JSONL files in this
    project alone:
    `rg -oI '"hookName":"SessionStart:[a-z]*"' . | sort | uniq -c`
  found: >
    SessionStart:clear   = 1040
    SessionStart:startup =  501
    SessionStart:compact =   11
    SessionStart:resume  =    0
  implication: >
    ORIGINALLY read as: the absence is NOT project-specific and NOT config-specific to this repo;
    three of the four documented SessionStart sources dispatch and are recorded while `resume`
    never has, machine-wide.
  status: DOWNGRADED (see the 2026-08-13 correction entry below)
  revised_implication: >
    The census is consistent with the confirmed root cause but is NOT probative of it. It is
    equally explained by "no project on this machine had a resume-matched SessionStart hook whose
    stdout differed from what was already injected." Retained as context, not as support.

- timestamp: 2026-08-12 (investigation)
  checked: >
    Whether a hook producing EMPTY stdout is still recorded in the transcript (i.e. whether the
    zero count above could be an artifact of invisible-but-successful dispatch).
  found: >
    Empty-output hooks ARE recorded. Confirmed instances:
    `"type":"hook_success","hookName":"PreToolUse:Bash",...,"content":"","stdout":""`
    (many, plus PostToolUse:Bash). The transcript records the dispatch, not merely the output.
  implication: >
    ORIGINALLY read as: zero `SessionStart:resume` entries means zero dispatches — a valid
    negative, not a visibility artifact. This was treated as the load-bearing control for the
    census above.
  status: REVOKED 2026-08-13 — the control does not transfer across hook events (see correction
    entry below). PreToolUse/PostToolUse recording behavior says nothing about SessionStart.

- timestamp: 2026-08-12 (investigation)
  checked: >
    The 441 bare `"hookName":"SessionStart"` entries (no matcher suffix) — candidate for a
    resume event recorded under a different label.
  found: >
    They are `"type":"hook_additional_context"`, not `"type":"hook_success"`. Distinct key set
    (`content`/`hookEvent`/`hookName`/`toolUseID` only — no `command`, `exitCode`, `stdout`).
    This is the aggregated context-injection record that follows a batch of SessionStart
    hook_success entries, not a separate matcher dispatch. Verified in
    `~/.claude/projects/-private-tmp-codegraph-nudge-scratch/44dec116-*.jsonl` lines 5,6 =
    hook_success:SessionStart:startup, line 7 = hook_additional_context:SessionStart.
  implication: Rules out "resume fires but is labelled without a suffix."

- timestamp: 2026-08-12 (investigation)
  checked: Claude Code version and registration syntax
  found: >
    Claude Code 2.1.229. `.claude/settings.json` and `.claude/hooks/hooks.json` both register
    `{"matcher":"resume","hooks":[{"type":"command","command":"${CLAUDE_PROJECT_DIR}/.claude/hooks/session-nudge.sh"}]}`
    — byte-identical in shape to the `startup` entry that demonstrably works.
  implication: >
    Registration is not differentially wrong; the two entries differ only in the matcher string.

- timestamp: 2026-08-13 (probe round 1 — instrumented isolated temp project)
  checked: >
    Three SessionStart entries (matcher "startup", matcher "resume", and a catch-all with no
    matcher) pointing at a logger that appends the raw stdin JSON payload — which carries the
    `source` field — to an append-only probe.log independent of the transcript. Started a session,
    then resumed that exact session id.
  found: >
    The resume matcher DOES fire. probe.log:
      08:15:11 | label=M-catchall | ... "hook_event_name":"SessionStart","source":"resume"
      08:15:11 | label=M-resume   | ... "hook_event_name":"SessionStart","source":"resume"
    Yet the same session's transcript JSONL contains ZERO SessionStart:resume records
    (2x SessionStart:startup, 1x bare SessionStart, 4x Stop — nothing else).
  implication: >
    H1 (runtime never emits source=resume) and H4 (no SessionStart dispatch at all on resume) are
    both REFUTED. The dispatch happens; the transcript does not always record it. The Phase 6
    oracle — transcript grep — is therefore suspect as an instrument.

- timestamp: 2026-08-13 (full probe ladder — final)
  checked: >
    Seven probes. All: start a session with `claude -p`, capture the session_id, then
    `claude --resume <id> -p`. Ground truth for "did the hook EXECUTE" is the hook's own
    append-only probe.log, deliberately independent of the transcript (the instrument under
    suspicion). Third oracle for "was it INJECTED" is asking the resumed model what it can see.
  found: >
    | probe | settings source    | path form   | hook stdout            | resume EXECUTED | in transcript | injected |
    |-------|--------------------|-------------|------------------------|-----------------|---------------|----------|
    | 1     | `--settings` flag  | absolute    | EMPTY                  | YES             | no            | n/a      |
    | 2     | `--settings` flag  | absolute    | per-source nonce       | YES             | YES           | YES      |
    | 3     | project `.claude/` | `${CPD}`    | CONSTANT 132B nudge    | (unlogged)      | **NO**        | **NO**   |
    | 4     | project `.claude/` | absolute    | per-source nonce       | YES             | YES           | YES      |
    | 5     | project `.claude/` | `${CPD}`    | per-source nonce       | YES             | YES           | YES      |
    | 6     | probe5 + hooks.json| `${CPD}`    | per-source nonce       | YES             | YES           | YES      |
    | 7     | project `.claude/` | `${CPD}`    | **CONSTANT** string    | **YES (log)**   | **NO**        | **NO**   |
  implication: >
    Probe 2 refutes H1/H4 outright: `"source":"resume"` reaches the hook, the hook runs, its stdout
    is injected, and it IS recorded as `SessionStart:resume`.
    Probe 3 REPRODUCES the reported bug exactly, using the repo's `.claude/` copied verbatim into a
    scratch tree with `.codegraph/` present: `SessionStart:startup` x3 recorded and nudge injected;
    ZERO `SessionStart:resume`, ZERO hook_error, `session-nudge.sh` named exactly once in the whole
    transcript (the startup dispatch) — silent non-dispatch by appearance.
    Probe 4 refutes "project-scoped settings are ignored on resume".
    Probe 5 REFUTES C2 (`${CLAUDE_PROJECT_DIR}` expands fine on the resume path).
    Probe 6 REFUTES the hooks.json-shadowing theory AND independently confirms the code comment at
    `hookpackage_test.go:28-32` — `.claude/hooks/hooks.json` is genuinely not read by Claude Code
    (its `H-startup`/`H-resume` entries never executed). Not a defect; it is Phase 7's embed source.
    Probe 7 is the decisive single-variable test: identical to probe 5 except the hook emits a
    CONSTANT string instead of a per-source one. The single discriminating variable across the
    whole ladder is therefore stdout constancy, not path form, not settings source, not payload.

- timestamp: 2026-08-13 (probe 7 transcript forensics — per-record, not aggregate)
  checked: The probe 7 session transcript JSONL, record by record, against probe.log timestamps.
  found: >
    probe.log proves `source=resume` executed at 12:49:56Z. The transcript holds ZERO
    `SessionStart:resume` records, and the resumed model counts the constant token exactly 2x
    (the startup injection + my own prompt echoing it), not 3x.
      line  5: attach=hook_success hookName=SessionStart:startup  ts=12:49:43Z  <- only hook record
      line 17: queue-operation                                    ts=12:49:56Z
      line 19: user message (my prompt, contains the token)       ts=12:49:57Z
    The resume dispatch executed between lines 5 and 19 and left no record of any kind.
  implication: >
    Byte-identical stdout is suppressed in BOTH channels — transcript record and context
    injection. This is the confirmed mechanism.

- timestamp: 2026-08-13 (correction — revokes a previously banked control)
  checked: >
    Whether the earlier conclusion "empty-stdout hooks ARE recorded, so the zero count is a valid
    negative" survives the probe ladder.
  found: >
    It does not. That control was drawn from PreToolUse:Bash / PostToolUse:Bash records and DOES
    NOT transfer to SessionStart. Probe 1 proves it: its three SessionStart hooks all executed
    (probe.log) and all emitted empty stdout, and NONE produced any transcript record — not even
    under the `startup` matcher. The two `SessionStart:startup` records in probe 1's transcript
    belong to the user-level global hooks (`"command":"bash \"/Users/sean/.claude/hooks/...`),
    not to the probe.
  implication: >
    The machine-wide census (resume=0 across ~180 projects) is NOT the strong evidence it was
    taken for — it is equally explained by "no project on this machine had a resume-matched
    SessionStart hook that emitted distinct output." Downgraded to consistent-but-not-probative.
    Both affected entries above are annotated accordingly.

- timestamp: 2026-08-13
  checked: `internal/agents/hookpackage_test.go:371-416` (TestHookRegistrationMatchesFragmentAndScript)
  found: >
    The test iterates `hooks.SessionStart` entries and asserts each entry's command resolves to an
    existing executable file, and that settings.json equals the hooks.json fragment. It never
    asserts WHICH matchers are present. Deleting the entire `resume` entry from BOTH files keeps
    every assertion satisfied — the arrays stay equal, and the surviving `startup` command still
    resolves.
  implication: >
    The false finding actively invites a green-tests regression: a reader who believes "resume
    never fires" would delete the now-'dead' matcher and the suite would applaud. This is the
    concrete recurrence risk and the thing worth guarding.

## Eliminated

- hypothesis: The `resume` matcher entry in `.claude/settings.json` is malformed or mis-registered.
  evidence: >
    It is structurally identical to the `startup` entry in the same file, which fires correctly
    and is recorded as `SessionStart:startup`. Only the matcher string differs. Later confirmed
    positively by probes 4/5/6, which fire and record with the same registration shape.
  timestamp: 2026-08-12

- hypothesis: The hook fires on resume but produces no transcript record because its stdout is empty.
  evidence: >
    Eliminated on 2026-08-12 via PreToolUse/PostToolUse empty-stdout records — and that
    elimination was itself WRONG. Re-opened and resolved in the opposite direction on 2026-08-13:
    probe 1 shows empty-stdout SessionStart hooks execute and produce no record at all. Empty
    stdout is a sufficient (but not necessary) condition for suppression. The final root cause
    generalizes it: stdout that adds nothing new to context is suppressed, empty being the
    degenerate case. Not applicable to this repo's script, which emits 132 non-empty bytes.
  timestamp: 2026-08-12, corrected 2026-08-13

- hypothesis: Resume events are recorded under the bare `SessionStart` hookName (441 instances).
  evidence: >
    Those entries are `hook_additional_context` aggregation records with a disjoint key set,
    always immediately following `hook_success` entries from the same startup batch. Not
    matcher dispatches.
  timestamp: 2026-08-12

- hypothesis: The `session-nudge.sh` script itself is broken.
  evidence: >
    By-hand execution is clean in both trees (exit=0, byte-exact 132-byte stdout in the indexed
    case, empty in the unindexed case, empty stderr). The identical script fires successfully
    under the `startup` matcher, and under `resume` in probe 3 (proven by the resumed model
    quoting the nudge line verbatim).
  timestamp: 2026-08-12

- hypothesis: >
    H1 (environment/tooling) — Claude Code 2.1.229's SessionStart dispatcher does not emit a
    `resume`-sourced event on `claude --resume` / `--continue` / `/resume`, despite the hooks
    reference documenting that source.
  evidence: >
    Probe round 1 and probe 2: the hook's own append-only log captures the raw stdin payload
    containing `"hook_event_name":"SessionStart","source":"resume"`. The runtime demonstrably
    emits the source. Probe 2 additionally records it as `SessionStart:resume` in the transcript.
  timestamp: 2026-08-13

- hypothesis: >
    H2 (config) — resume DOES dispatch SessionStart but with a source string our matcher does not
    equal (empty, different casing, or a different token), so `"resume"` misses while a catch-all
    would catch it.
  evidence: >
    Probe round 1: the catch-all AND the `"resume"`-matched entry both fired, in the same
    millisecond, on the same dispatch. The token is literally `resume` and the matcher matches it.
  timestamp: 2026-08-13

- hypothesis: >
    H4 (environment) — resume dispatches no SessionStart event at all, under any matcher.
  evidence: Refuted by the same probe round 1 log lines that refuted H1.
  timestamp: 2026-08-13

- hypothesis: >
    H5 — Claude Code executes the SessionStart `resume` hook but NEVER writes a `hook_success`
    record for resume-sourced dispatches into the transcript.
  evidence: >
    Too strong. Probes 2, 4, 5 and 6 all produce `SessionStart:resume` transcript records. The
    suppression is conditional on output content, not on the source. H5's useful half survives —
    the transcript is not a valid oracle for dispatch — and became condition (2) of the root cause.
  timestamp: 2026-08-13

- hypothesis: >
    C2 (config) — the `${CLAUDE_PROJECT_DIR}` placeholder in the hook `command` string is
    substituted on the startup dispatch path but not on the resume path, so the resume entry's
    command never resolves and is dropped silently.
  evidence: >
    Probe 5 is byte-for-byte probe 4 with EXACTLY ONE variable changed — the command path form
    becomes `${CLAUDE_PROJECT_DIR}/.claude/hooks/nonce-hook.sh`. Resume fired, was recorded, and
    was injected. The placeholder expands fine on the resume path.
    (Confound that motivated this probe, acknowledged at the time: probe 3 differed from probe 4
    in TWO ways — the path form, and probe 3 carrying the repo's full `.claude/` payload
    (`settings.local.json`, `hooks/hooks.json`, a 23KB `CLAUDE.md`) versus probe 4's bare
    `settings.json`. Probe 5 isolated (i); probe 6 isolated (ii).)
  timestamp: 2026-08-13

- hypothesis: >
    Probe 3's extra `.claude/` payload — `hooks/hooks.json` or `settings.local.json` — shadows or
    overrides `settings.json` on the resume path.
  evidence: >
    Probe 6 = probe 5 plus `hooks.json`. Resume fired, was recorded, and was injected. The
    `hooks.json` `H-startup`/`H-resume` entries never executed at all, independently confirming
    the code comment at `hookpackage_test.go:28-32` that Claude Code does not read
    `.claude/hooks/hooks.json`.
  timestamp: 2026-08-13

## Reasoning Checkpoint

reasoning_checkpoint:
  hypothesis: >
    Claude Code 2.1.229 deduplicates SessionStart hook output: when a hook's stdout is
    byte-identical to context already injected in that session, the runtime suppresses BOTH the
    context injection AND the transcript `hook_success` record. `session-nudge.sh` emits one
    constant string for every source, so on `--resume` the hook runs but its output — identical
    to the startup injection already carried in the resumed conversation — is dropped. The
    resulting transcript is byte-indistinguishable from "the matcher never fired."
  confirming_evidence:
    - "probe 7 probe.log: `08:49:56 | entry=E-resume | source=resume` — direct proof of execution."
    - "probe 7 transcript: zero SessionStart:resume records; only hook record is startup at 12:49:43Z."
    - "probe 5 vs probe 7 differ in exactly ONE variable (per-source vs constant stdout); probe 5
       records and injects the resume dispatch, probe 7 records and injects nothing."
    - "probe 3 reproduces the original report verbatim using the repo's own .claude/ copied into a
       scratch tree with .codegraph/ present."
    - "probe 3's RESUMED model quoted the nudge line verbatim — the nudge IS in resumed context,
       carried from startup. The user-facing requirement was never actually broken."
  falsification_test: >
    Probe 7 was itself the falsification test: had the constant-output hook produced a
    `SessionStart:resume` record or a 3rd copy of the token, output-dedup would have been refuted
    and the cause would have had to lie in probe 3's residual `.claude/` payload.
  fix_rationale: >
    The runtime behavior is upstream, benign, and arguably correct (it suppresses a redundant
    re-injection of text already in context). Nothing in `session-nudge.sh` or the matcher
    registration is broken, so changing either would be fixing a symptom that does not exist.
    The actual repo-side defect is a FALSE RECORD: the evidence artifact asserts "did not fire ...
    a real, reproducible gap, not a rehearsal methodology error" — the opposite of the truth — and
    STATE.md carries a matching false blocker. Correcting those addresses the root cause of the
    wrong belief. The recurrence guard is the missing matcher-set assertion identified above,
    which converts "someone deletes the resume matcher" from silent-and-green into a test failure.
  blind_spots:
    - "All probes used `-p` (headless) mode; the original rehearsal was interactive. The dedup
       result is consistent across both (probe 3 reproduces the interactive finding in -p), but
       interactive `/resume` and `--continue` were not separately exercised. The user waived the
       interactive spot-check at the human-verify checkpoint, so this remains untested."
    - "The exact dedup scope is uncharacterized: whether it compares against the whole context or
       only prior SessionStart injections, and whether a compacted-away startup nudge would let an
       identical resume nudge through. Not needed for this fix, but it bounds what I can claim."
    - "n=1 per probe configuration. The effect is deterministic across 7 probes with a clean
       single-variable ladder, but no configuration was repeated to test flakiness."
    - "Only Claude Code 2.1.229 on darwin was tested; the dedup may be version-specific."
  candidate_causes:
    - "code — session-nudge.sh or the matcher registration is wrong (REFUTED: probes 4/5/6 fire
       with the same registration shape; the script's own tests pass byte-exact)"
    - "config — `${CLAUDE_PROJECT_DIR}` unexpanded, or hooks.json shadowing settings.json
       (REFUTED by probes 5 and 6 respectively)"
    - "environment/tooling — Claude Code's SessionStart runtime dedupes identical injected output
       (CONFIRMED by probe 7)"
    - "data/observation — the verification oracle (transcript grep) cannot see a suppressed
       dispatch, producing a false negative (CONFIRMED; this is what made the runtime behavior
       look like a project defect)"
  and_gate: >
    YES — this failure required TWO conditions simultaneously, and neither alone is visible.
    (1) the runtime's identical-output suppression, and (2) an oracle that reads only the
    transcript. With per-source output (probes 2/4/5/6) the suppression never engages and the
    same oracle reports correctly. With a context-based oracle (asking the resumed model, as
    probe 3 did) the suppression is present but harmless and the nudge is demonstrably there.
    Only the conjunction produces "correctly registered hook appears never to fire."

## Resolution

root_cause: >
  AND-gate, both conditions required (neither alone produces the symptom):
  (1) ENVIRONMENT/TOOLING — Claude Code 2.1.229 suppresses a SessionStart hook's context
      injection AND its transcript `hook_success` record when the hook's stdout is byte-identical
      to context already injected in that session. `session-nudge.sh` emits one constant line for
      every source, so its `resume` dispatch is always the suppressed case.
  (2) DATA/OBSERVATION — the Phase 6 rehearsal's oracle was a grep of the resumed session's
      transcript for `SessionStart` hook events. That oracle is structurally blind to a suppressed
      dispatch, so it returned a false negative, which the evidence artifact then hardened into
      "a real, reproducible gap, not a rehearsal methodology error."
  With per-source output (probes 2/4/5/6) condition (1) never engages and the same oracle reports
  correctly. With a context-based oracle (probe 3: ask the resumed model) condition (1) is present
  but harmless and the nudge is demonstrably in context. Only the conjunction yields "a correctly
  registered hook appears never to fire."
  The registration, the script, and NUDGE-01 on the resume path were never broken.

tdd_checkpoint:
  test_file: "internal/agents/hookpackage_test.go"
  test_name: "TestNudgeLiveSessionEvidenceRetractsResumeFailureClaim"
  status: "green"
  red_evidence: "7 assertion failures before the fix (4 retracted claims present, 3 mechanism terms absent)"

fix: >
  No code change to session-nudge.sh or the matcher registration — neither is defective, and the
  runtime behavior is upstream and benign (it suppresses a redundant re-injection of text already
  in context). The defect fixed is the FALSE RECORD and the absent recurrence guard:
  1. Retracted and rewrote the resume section of NUDGE-live-session.md with the probe-ladder
     evidence, both oracles, the ruled-out alternatives, and an explicit methodology note that
     transcript grep is not a valid oracle for SessionStart dispatch (it is blind to empty or
     duplicated output under ANY matcher, including `startup`).
  2. Marked the follow-up item resolved-and-not-a-gap rather than deleting it, so the retraction
     is legible to a future reader.
  3. Replaced STATE.md's false blocker with the resolution and the transferable correction.
  4. Added TestSessionStartRegistersBothDocumentedEntryPoints, pinning the matcher SET — the gap
     that made the invited regression silent.

verification:
  guardrail_verdict: accepted
  signal_1_regression_test_goes_red: >
    PASS. TestNudgeLiveSessionEvidenceRetractsResumeFailureClaim failed with 7 assertions before
    the fix and passes after. Confirmed RED-then-GREEN, not written green.
  signal_2_mutation_at_fix_site: >
    PASS — 6/6 mutants killed, zero survivors.
      M1 delete the `resume` entry from both files (the edit the false finding invites) -> killed.
         CONTROL: the pre-existing TestHookRegistrationMatchesFragmentAndScript PASSED on this
         mutant, proving the gap was real and the new guard is not redundant.
      M2 duplicate the `resume` entry -> killed (double-dispatch).
      M3 widen the set with a `clear` matcher -> killed.
      M4 drop the `matcher` key (catch-all entry) -> killed.
      M5 re-insert a retracted claim verbatim -> killed.
      M6 strip the mechanism word "suppress" (retraction without explanation) -> killed.
  signal_3_revert_restores_bug: >
    PASS. M5/M6 are the revert test: restoring the retracted prose, or removing the mechanism
    explanation, returns the suite to failing.
  signal_4_not_deletion_only: >
    PASS. The diff adds a corrected evidence record and two guards. The deletions are exactly the
    four disproved claims, each justified by a named probe in the RCA.
  signal_5_full_regression: >
    PASS. `go build ./...` clean; `go test ./...` 47 packages ok, zero failures; `go vet ./...`
    clean; `gofmt -l internal/agents/` empty; `task lint` (vet + actionlint) clean.
  oracle_type: >
    specified — the artifact is a specified milestone deliverable (D-07 / NUDGE-01) and the
    required content is derived from the confirmed root cause, not from crash-absence. Boundary
    neighbors covered by M2/M3/M4 (duplicate, widened, and matcher-less entries around the exact
    {startup, resume} equivalence class), not just the single reported value.
  human_verification: >
    2026-08-13 — WAIVED BY USER at the human-verify checkpoint. The interactive `/resume`
    spot-check was explicitly skipped in favour of the automated proof (TDD red->green retraction
    test, 6/6 mutation kill, 3-source SessionStart census, probe 5 vs probe 7 single-variable
    comparison). Interactive `/resume` and `--continue` therefore remain unexercised — recorded
    as a standing blind spot, not as a passed check.

files_changed:
  - .claude/skills/codegraph/verification/NUDGE-live-session.md: retracted the false resume finding; documented mechanism, both oracles, ruled-out alternatives, and the oracle-validity note
  - .planning/STATE.md: replaced the false NUDGE-01 blocker with the resolution and transferable correction
  - internal/agents/hookpackage_test.go: added TestNudgeLiveSessionEvidenceRetractsResumeFailureClaim and TestSessionStartRegistersBothDocumentedEntryPoints
