---
status: resolved
trigger: "codegraph skill still absent from the rendered skill listing after both the description-trim fix (bc1c853) and a skillListingBudgetFraction increase, confirmed after a full binary restart"
created: 2026-08-13
updated: 2026-08-13
---

# Debug Session: codegraph-skill-not-listed

## Symptoms

**Expected behavior:** After debug session `skill-discovery-not-listing` (see
`.planning/debug/resolved/skill-discovery-not-listing.md`, commits bc1c853/d42b9f8), the
`codegraph` skill's frontmatter description was cut from 299 to 110 bytes specifically to survive
Claude Code's undocumented skill-listing render cap (measured ~45,000 chars / 173 descriptions
admitted at the default `skillListingBudgetFraction: 0.01`). That session's own known-gap note
said 110 bytes doesn't *guarantee* admission, just competitiveness. Immediately afterward, the
user raised `skillListingBudgetFraction` globally from 0.015 to 0.03 (doubling the character
budget) specifically to make room for more of their ~238 installed skills, including this one.
With both changes in place — description under the per-entry admission threshold AND double the
aggregate budget — `codegraph` should now render in the skill listing with its description
(`- codegraph: Use when asked where X is defined...`), or at minimum appear as a bare name if
still marginal.

**Actual behavior:** `codegraph` does not appear in the rendered skill listing AT ALL — not with
a description, not as a bare name. The system-reminder block "The following skills are available
for use with the Skill tool:" lists exactly one entry (`engram:curating-spine`) in the current
session. This is a stronger absence than the original bug (which produced a degraded bare-name
entry, not a total absence), so this may not be the same failure mode recurring — it could be a
different mechanism entirely.

> **RETRACTED 2026-08-13 — this reported symptom is false.** Preserved verbatim above rather than
> deleted, because the shape of the error is the finding. Two things were misread. (a) The
> one-entry block is an `isInitial: false` DELTA skill_listing carrying `skillCount: 1`, emitted
> 3.4 seconds after `SessionStart:resume`; Claude Code renders deltas under the byte-identical
> header the full catalog uses, so it is not the catalog. That same session's catalog holds 234
> entries, and it also contains a codegraph delta at 14:00:00.601Z whose entire content is the
> correctly-described entry. (b) `codegraph` was never ABSENT anywhere — in the session's (stale,
> replayed) catalog it is BARE, one of 61 degraded entries; in every freshly computed listing it
> renders WITH its description. The instinct recorded above — that "stronger absence" implied a
> different mechanism — was the right instinct pointed at an artifact of the observation rather
> than at the system. See Resolution.

**Error messages:** None observed. Silent absence, same class as both prior Phase 6 findings.

**Timeline:**
1. `skill-discovery-not-listing` resolved 2026-08-13 (commits bc1c853, d42b9f8) — root cause was
   the render-cap degrading the entry to a bare name; fix was trimming the description to 110
   bytes; known gap explicitly stated live confirmation was NOT done.
2. User asked how to raise the underlying cap; `skillListingBudgetFraction` (a real, schema-
   documented setting, confirmed via the full settings.json schema: "Fraction of the context
   window (in characters) reserved for the skill listing sent to Claude (default: 0.01 = 1%)...
   Raise to opt in to higher per-turn context cost") was bumped from 0.015 to 0.03 in
   `~/.claude/settings.json`.
3. First verification attempt used `/resume` or `/clear` (unclear which) — user said "session
   restarted." Skill listing still showed only `engram:curating-spine`, no `codegraph`, not even
   as a bare name.
4. Asked whether this was a real process restart (settings.json is typically read once at process
   startup) vs. an in-process `/resume`/`/clear` that might not reload it. User confirmed: **"I
   restarted the binary"** — a genuine new process launch, not an in-process resume/clear.
5. Pre-investigation sanity checks (this session, before writing this file) confirmed the simple
   explanations are ruled out:
   - `jq -r '.skillListingBudgetFraction' ~/.claude/settings.json` → `0.03` (correct value on
     disk, not reverted)
   - `codegraph` SKILL.md description → 111 bytes (matches the committed fix, not regressed)
   - `git status`/`git diff HEAD` on the SKILL.md file → clean, working tree matches HEAD, no
     uncommitted regression
   - Rough count of `SKILL.md` files under `~/.claude/skills` + `~/.claude/plugins/cache`
     (maxdepth 3) → 74. This is far short of the "~238 installed skills" figure cited in the prior
     session's fix commit message, so that figure (or this rough count) may be wrong, or the real
     catalog draws from more locations than this shallow find covers (project-scoped skills,
     deeper marketplace plugin nesting, skills contributed by other mechanisms). Not resolved —
     flagged for the investigation, not assumed either way.

**Reproduction steps:**
1. Confirm `~/.claude/settings.json` has `skillListingBudgetFraction: 0.03` and
   `.claude/skills/codegraph/SKILL.md`'s description is the committed 110-byte version (both
   already confirmed true as of this file's creation).
2. Fully quit and relaunch the `claude` binary (not `/resume`, not `/clear` — a genuine new
   process) in this repository.
3. Inspect the actual rendered "available skills" system-reminder block for the new session —
   the literal listing content, not a transcript grep for the string `codegraph` (methodology
   lesson carried forward from BOTH prior sessions in this repo: a text-search oracle can be
   blind to a suppressed OR degraded entry in ways that don't mean "absent").
4. Observe: `codegraph` is not present in that listing in any form.

## Evidence

- timestamp: 2026-08-13 (prior session, cited not re-litigated)
  source: .planning/debug/resolved/skill-discovery-not-listing.md
  finding: >
    Established the render-cap mechanism (~45,000 chars / 173 descriptions at 0.01 budget
    fraction, project-scoped skills appended last) and fixed the description length. Explicitly
    left "confirming the entry now actually renders" as an open, undone gap — this session is
    that follow-up, and it found something worse than expected (total absence, not degradation).

- timestamp: 2026-08-13 (this session, live)
  checked: The actual rendered skill-listing system-reminder in the current, freshly-restarted
    session (not a transcript grep — the live content itself)
  found: >
    Exactly one entry: `engram:curating-spine`. No `codegraph` entry in any form (no description,
    no bare name).
  implication: >
    Either (a) codegraph is being excluded from the catalog by some mechanism other than the
    character-budget cap the prior session diagnosed — a different cap, an ordering/precedence
    rule, a project-skill-eligibility check unrelated to size, or something about this specific
    session's state — or (b) the render cap's actual behavior is more complex than the prior
    session's model (e.g., an entry that doesn't fit is dropped entirely rather than degraded to
    bare-name, under some condition not yet identified — contradicting the prior session's own
    observed bare-name-degradation behavior, which makes this the more interesting anomaly to
    pursue) or (c) something about the environment differs from the prior investigation's probes
    (e.g., those probes used headless `-p` mode; this is genuinely interactive).

- timestamp: 2026-08-13 (this session, pre-investigation sanity check)
  checked: settings.json value, SKILL.md description bytes and git state, rough skill-file count
  found: see Symptoms/Timeline item 5 above — all simple regression explanations ruled out.
  implication: This is not a reverted fix or stale cache in any form checked so far. Needs real
    investigation, not another quick check.

- timestamp: 2026-08-13T14:1xZ (this session)
  checked: >
    THIS debugging subagent's OWN skill listing, visible verbatim in its context, and its recorded
    form on disk at
    `~/.claude/projects/-Volumes-.../67ab93a5-.../subagents/agent-ad1b315732876b040.jsonl`.
  found: >
    `- codegraph: Use when asked where X is defined, how Y works, what calls X, or what changing X
    breaks in a .codegraph/ repo.` — PRESENT, WITH ITS FULL DESCRIPTION, positioned after
    `terraform` (end of the personal block) and before `c4-architecture:c4-architecture`, exactly
    where a project-scoped entry belongs. Attachment: isInitial=true, skillCount=235, len=60,405,
    computed 2026-08-13T14:20:55Z.
  implication: >
    The reported symptom is FALSE as stated. A listing computed from current on-disk state admits
    the entry with its description. The fix works.

- timestamp: 2026-08-13T14:1xZ (this session)
  checked: >
    All `skill_listing` attachments in the user's own live session transcript
    `67ab93a5-d283-46af-b3d3-7e46723a78f4.jsonl` — the session the symptom was reported from.
  found: >
    THREE, not one, and they are not the same kind of thing. The attachment schema carries an
    `isInitial` boolean and a `skillCount`:
      1. isInitial=true,  skillCount=234, len=45,002, ts=2026-08-13T00:32:49.600Z — the FULL catalog
      2. isInitial=false, skillCount=1,   len=123,    ts=2026-08-13T14:00:00.601Z — names=[codegraph]
      3. isInitial=false, skillCount=1,   len=854,    ts=2026-08-13T14:06:56.573Z — names=[engram:curating-spine]
    Attachment 2's entire content is verbatim: `- codegraph: Use when asked where X is defined, how
    Y works, what calls X, or what changing X breaks in a .codegraph/ repo.`
  implication: >
    Both halves of the diagnosis fall out of this record. (a) The block the user read — "lists
    exactly one entry (engram:curating-spine)" — is attachment 3, an `isInitial: false` DELTA
    carrying one skill. Claude Code renders deltas under the SAME system-reminder header as the
    full catalog ("The following skills are available for use with the Skill tool:"), so the
    most-recent such block in a session is NOT the catalog. (b) Attachment 2 is positive proof, in
    the user's own session six minutes before the reported observation, that codegraph renders WITH
    its corrected description.

- timestamp: 2026-08-13T14:2xZ (this session)
  checked: >
    Whether the "binary restart" produced a freshly computed listing. Scanned 67ab93a5's hook
    attachments for SessionStart source, and its timestamp gaps.
  found: >
    `SessionStart:resume` fired at 2026-08-13T14:06:53.154Z — 3.4 seconds before the one-entry
    engram delta (14:06:56.573Z) the user read. The session's only `isInitial: true` listing remains
    the 00:32:49Z one (SessionStart:clear at 00:32:36Z), with a 743-minute idle gap between
    00:34:04Z and 12:57:20Z. No new full listing was emitted on resume.
  implication: >
    The restart was real but it RESUMED this conversation rather than starting a new one. A resumed
    session replays the transcript's original `isInitial: true` attachment and emits only deltas for
    newly-available skills. That original attachment was computed 2026-08-13T00:32:49Z — 13 hours
    before the description fix (bc1c853, 13:42:50Z) and 13.5 hours before the settings change
    (settings.json mtime 14:03:56Z) — so it could not possibly reflect either fix. Verification via a
    resumed session is structurally incapable of observing this class of fix.

- timestamp: 2026-08-13T14:2xZ (this session)
  checked: >
    Whether codegraph is bare or absent in that stale replayed listing, classified per-name against
    the attachment's own `names` array (not a regex, which mis-parses plugin names containing colons).
  found: >
    codegraph is BARE (`- codegraph`, no colon, no description) — one of 61 bare entries out of 234.
    NOT absent. Absence was never observed anywhere in this investigation.
  implication: >
    Even the stale listing does not exhibit the reported symptom. "Absent entirely" is not a state
    this system ever produced; the reporter conflated a bare entry in a stale listing with a
    one-entry delta block.

- timestamp: 2026-08-13T14:26Z (this session)
  checked: >
    DECISIVE TEST. Launched a genuinely fresh, non-resumed session in this repository on
    `claude-sonnet-5` — the user's exact model — via `claude -p`, then read the new transcript's
    `isInitial: true` attachment (ef5cd673-1573-420d-a8fe-5cfdff354d35.jsonl).
  found: >
    skillCount=232, len=59,681, ts=14:26:05.513Z, and codegraph renders as
    `- codegraph: Use when asked where X is defined, how Y works, what calls X, or what changing X
    breaks in a .codegraph/ repo.` — admitted WITH its description. Bare entries: ZERO of 232.
  implication: >
    CONFIRMED. In the user's own model, own repository, own machine, current on-disk state, a fresh
    session renders the entry correctly and NOTHING is degraded any more. Both fixes work. There is
    no surviving product defect.

- timestamp: 2026-08-13T14:2xZ (this session)
  checked: >
    The prior session's claim that the cap is a fixed ~45,000 characters admitting exactly 173
    descriptions. Measured rendered listing length across 8 listings straddling the settings.json
    mtime boundary (14:03:56Z) and across two model context-window sizes.
  found: >
    The cap is NOT a constant — it tracks `skillListingBudgetFraction` x the model's context window,
    exactly as the setting's own schema text says.
      BEFORE 14:03:56Z: 44,990 (233 skills, 14:00:33Z) / 44,993 (238, 12:33:20Z) / 45,002 (234,
        00:32:49Z) — 61-67 bare entries each.
      AFTER  14:03:56Z: 59,426 (233, 14:15:35Z) / 59,557 (231, 14:24:55Z) / 59,681 (232, 14:26:05Z)
        / 60,405 (235, 14:20:55Z) — ZERO bare entries.
      Different model, same day, post-change: `claude-haiku-4-5` (small context window) → 23,984
        chars, 85 of 232 entries bare, codegraph bare.
    The tier boundary coincides exactly with the settings.json mtime, and the post-change length is
    the listing's FULL size rather than a clamp: at 0.03 the budget exceeds the ~59,700 chars the
    complete catalog needs, so nothing is truncated at all.
  implication: >
    Corrects the prior session's model, which is hardened into a doc comment, a test failure message
    and an evidence artifact as if ~45,000/173 were invariants. They are one operator's measurements
    at one fraction on one model. A reader trusting them would mis-predict on any other model or
    fraction — as this investigation's own first probe did: forcing `--model haiku` produced a 23,984
    -char budget and a bare codegraph, briefly looking like the fix had failed.

- timestamp: 2026-08-13 (human verification, INDEPENDENT ORACLE)
  checked: >
    The user's own confirmation, obtained deliberately WITHOUT re-running this investigation's
    probe. They split a sibling pane via Herdr and started a genuinely fresh `claude` process (a
    NEW session id — explicitly not `--resume`, so not the stale-catalog path that produced the
    original symptom), then asked that process, with no tools available, purely from its own
    injected system context, whether `codegraph` appears in its skill listing.
  found: >
    It answered yes and quoted the description verbatim: `Use when asked where X is defined, how Y
    works, what calls X, or what changing X breaks in a .codegraph/ repo.` — byte-for-byte the
    committed fix.
  implication: >
    This is the methodologically load-bearing evidence of the session, and its INDEPENDENCE is the
    whole point. Every prior instance of this bug class in this repository (all three) failed by
    trusting a single instrument and mistaking a claim about that instrument for a claim about the
    world. My own decisive test at 14:26Z read a transcript file I selected, parsed with a script I
    wrote, from a session I launched — a chain in which one wrong assumption anywhere reproduces the
    prior failures exactly. This oracle shares none of those links: a different process, launched by
    a different actor through a different mechanism, reporting on its OWN context rather than on a
    file about that context, with no tools to consult the on-disk state. It could not have inherited
    my parsing, my file selection, or my framing. Two independent oracles agreeing on the byte-exact
    description string is what a confirmation is supposed to look like; either one alone is what the
    three prior false findings looked like.

## Eliminated

- hypothesis: Settings change didn't persist / was reverted
  test: jq -r '.skillListingBudgetFraction' ~/.claude/settings.json
  result: 0.03, matches the intended change
  eliminated: yes

- hypothesis: SKILL.md description regressed back to the original 299 bytes
  test: grep description field, byte-count; git status/diff against HEAD
  result: 111 bytes, clean working tree, matches committed fix
  eliminated: yes

- hypothesis: Verification was invalidated by testing via /resume or /clear instead of a real
    process restart (settings.json typically read once at startup)
  test: asked the user directly how they restarted
  result: "I restarted the binary" — genuine new process
  eliminated: PARTIALLY, AND NOW REINSTATED IN CORRECTED FORM. The restart was real, but the
    question asked was the wrong one. A real process restart that RESUMES a conversation replays
    the transcript's original `isInitial: true` skill_listing rather than recomputing it, so the
    distinction that mattered was never "restart vs /resume" — it was "new conversation vs resumed
    conversation." `SessionStart:resume` at 14:06:53.154Z proves this session was resumed. The
    settings file WAS re-read (its effect is visible in every listing computed after 14:03:56Z);
    the skill listing simply was not recomputed for the replayed conversation.

- hypothesis: The skill is silently skipped at parse time — malformed/unparseable frontmatter, a
    missing required field, wrong directory name or location, or filtering by an allow/deny list
    (i.e. a mechanism unrelated to the character budget, which would look identical in the listing
    but have a completely different fix).
  evidence: A freshly started session in this repo on the user's own model, reading the same
    on-disk SKILL.md at the same path, renders `- codegraph: Use when asked where X is defined,
    how Y works, what calls X, or what changing X breaks in a .codegraph/ repo.` — parsed, admitted
    and described. A skipped-at-parse-time skill cannot render its description.
  timestamp: 2026-08-13T14:26Z

- hypothesis: `.claude/settings.local.json`, per-project settings, or plugin-vs-project precedence
    overrides the global `skillListingBudgetFraction` for this repository.
  evidence: The decisive fresh probe ran IN this repository and therefore resolved exactly the same
    settings chain the user's session does. It rendered 232 entries in 59,681 characters with ZERO
    degraded — the raised fraction is plainly in effect here. A project-level override suppressing
    the setting would have produced the ~45,000-char/61-bare tier instead.
  timestamp: 2026-08-13T14:26Z

- hypothesis: The prior session's render-cap model is wrong because an entry that does not fit is
    dropped ENTIRELY rather than degraded to a bare name (the "total absence is a stronger symptom"
    reading in this file's own Symptoms section).
  evidence: Absence was never observed. Across 8 measured listings — classified per-name against
    each attachment's own `names` array rather than by regex — every entry is either described or
    bare, and codegraph specifically is bare (not absent) in the stale listing and described in
    every freshly computed one. The apparent "total absence" was a one-entry delta block being read
    as the catalog.
  timestamp: 2026-08-13T14:2xZ

## Current Focus

bug_class: Bohrbug — deterministic, fully explained by recorded transcript state; no timing or
  concurrency component.

known_pattern_candidate: >
  `skill-discovery-not-listing` AND `resume-matcher-not-firing` — both knowledge-base entries share
  the transferable lesson "a transcript/listing read is a claim about what you read, not about the
  world." CONFIRMED to apply a THIRD time here, in a new shape: the oracle this time is not a grep
  but reading the WRONG system-reminder block — a single-skill DELTA listing mistaken for the full
  catalog.

hypothesis: >
  AND-gate, two contributing causes. (1) ENVIRONMENT — the "restarted binary" session is a RESUMED
  session (67ab93a5) whose `isInitial: true` skill_listing attachment was computed 2026-08-13
  T00:32:49Z (Aug 12 20:32 EDT), i.e. BEFORE both the description fix (bc1c853, Aug 13 13:42:50Z)
  and the settings change (settings.json mtime Aug 13 10:03:56 EDT). Resuming replays that stale
  attachment from the transcript rather than recomputing it, so the pre-fix bare `- codegraph`
  entry is what the session carries. (2) DATA/OBSERVATION — the reported symptom ("the listing
  shows exactly one entry, engram:curating-spine") describes an `isInitial: false` DELTA listing
  (skillCount 1), not the catalog. Claude Code emits deltas under the SAME system-reminder header
  as the full listing, so the most-recent block a reader finds is a one-entry delta.

test: >
  Launch a genuinely fresh, non-resumed session in this repo with current on-disk state and read
  its `isInitial: true` skill_listing attachment directly from the new transcript file.

expecting: >
  CONFIRMS if the fresh initial listing contains `- codegraph: Use when asked where X is defined,
  ...` WITH its description. REFUTES if codegraph is bare or absent in a freshly-computed listing —
  which would mean a real admission failure survives and cause (1) is wrong.

reasoning_checkpoint:
  hypothesis: >
    CONFIRMED, AND-gate, two contributing causes, neither sufficient alone. There is NO surviving
    product defect — both prior fixes work. (1) ENVIRONMENT: `SessionStart:resume` does not
    recompute the skill catalog. The user's "binary restart" resumed conversation 67ab93a5, which
    replays the `isInitial: true` attachment recorded at 2026-08-13T00:32:49Z — 13 hours before the
    description fix and 13.5 hours before the settings change — so the session carries a stale,
    pre-fix listing in which codegraph is one of 61 bare entries. A resumed session is structurally
    incapable of observing either fix. (2) DATA/OBSERVATION: the block actually read and reported
    ("exactly one entry, engram:curating-spine") is an `isInitial: false` DELTA carrying
    skillCount=1, emitted 3.4s after the resume hook. Claude Code renders deltas under the same
    system-reminder header as the full catalog, so "the most recent skills block" is not "the
    catalog."
  confirming_evidence:
    - "A fresh, non-resumed `claude-sonnet-5` session in this repo at 14:26:05Z renders `- codegraph: Use when asked where X is defined...` WITH its description; 0 of 232 entries are bare."
    - "This debugger's own listing (14:20:55Z, 235 skills, 60,405 chars) carries codegraph with its full description."
    - "The user's OWN session contains a codegraph delta at 14:00:00.601Z whose entire content is the correctly-described entry — six minutes before the reported observation."
    - "`SessionStart:resume` at 14:06:53.154Z, engram one-entry delta at 14:06:56.573Z: the reported block is 3.4s downstream of a resume, and the session's only isInitial=true listing is still the 00:32:49Z one."
    - "Classified per-name against the attachment's own `names` array: codegraph is BARE (not absent) in the stale listing. Absence was never observed in any of 8 listings measured."
    - "Rendered length tiers split exactly at the settings.json mtime (14:03:56Z): 44,990/44,993/45,002 before (61-67 bare) vs 59,426/59,557/59,681/60,405 after (0 bare)."
  falsification_test: >
    A freshly started (non-resumed) session in this repo on the user's model rendering codegraph
    bare or absent would refute it. Run at 14:26:05Z on claude-sonnet-5: rendered WITH description,
    zero bare entries. Also refutable by finding an `isInitial: true` listing anywhere that omits
    codegraph entirely — none of the 8 measured listings did.
  fix_rationale: >
    No code fix is warranted; the mechanism is upstream and both prior fixes are verified working.
    What IS defective is hardened evidence, exactly the failure mode the two prior sessions were
    resolved for. The prior session recorded "~45,000 characters / exactly 173 descriptions" as an
    invariant in a doc comment, a test failure message and an evidence artifact. That is now proven
    false — the budget is `skillListingBudgetFraction` x the model's context window, measured at
    three different values (23,984 on haiku, ~45,000 at the old fraction, ~59,700 at 0.03) on one
    machine in one day. A reader trusting the constant mis-predicts, as this investigation's own
    first probe did. The fix corrects that model where it is written down and records the two new
    invalid oracles (resumed-session staleness, delta-vs-catalog) so instance four does not happen.
  blind_spots: >
    The exact budget formula is not derived, only bounded by three measurements; I do not know the
    prior fraction value with certainty (the file says 0.015, and 0.015 -> ~45,000 -> 0.03 -> ~90,000
    is consistent with the post-change listing being UNCAPPED at its full ~59,700 size, but I did not
    vary the fraction myself). Why the runtime emitted a codegraph delta at 14:00:00Z (file-change
    rescan? periodic?) is uncharacterized. All measurements are one machine, darwin, one day. I have
    not verified what an interactive `/clear` (as opposed to `--resume`) does to the listing.
  candidate_causes:
    - "code/authoring — SKILL.md malformed, mis-located, or filtered by an allow/deny list (REFUTED: a fresh session on the same file renders it correctly, with description)"
    - "config — .claude/settings.local.json or project settings overriding the global fraction (REFUTED: the fresh in-repo probe reads the same config and renders 0 bare entries)"
    - "environment — resumed session replays a stale pre-fix skill_listing attachment (CONFIRMED)"
    - "data/observation — a single-skill DELTA listing read as if it were the full catalog (CONFIRMED)"
  and_gate: >
    Yes. Cause 1 alone yields a stale listing in which codegraph is BARE — a correct oracle would
    have reported "listed without a description," i.e. the prior session's symptom recurring, not a
    new one. Cause 2 alone, in a fresh session, yields a reader seeing a one-entry delta and calling
    the skill absent even though the catalog is correct. Only together do they produce the reported
    symptom — "absent entirely, a stronger symptom than before" — which is a state neither the stale
    listing nor the fresh listing actually exhibits. `root_cause` therefore holds both.

tdd_checkpoint:
  test_file: "internal/mcp/skill_claims_drift_test.go"
  test_name: "TestSkillListingEvidenceRecordsBudgetIsNotFixed"
  status: "green"
  red_output: >
    6 assertion failures against the uncorrected artifact — two on the retracted invariant
    phrasings ("admitting **exactly 173** descriptions in every", "unlike the 173"), two on the
    missing replacement model ("skilllistingbudgetfraction", "context window"), two on the missing
    oracle traps ("isInitial", "delta"). "resume" already appeared in the artifact and passed,
    correctly, on the first run.
  green_output: >
    All of TestSkillListingEvidenceRecordsBudgetIsNotFixed, TestSkillFrontmatterIsSpecCompliant and
    TestSkillDescriptionSurvivesSkillListingCap PASS. Full packages: internal/mcp ok 3.381s,
    internal/agents ok. go vet clean. No existing test was weakened — the only changes to
    pre-existing tests are a doc comment and one failure-message string, neither of which is
    asserted on.

next_action: >
  DONE. Human verification returned CONFIRMED via an independent oracle (a separate fresh `claude`
  process, no tools, asked what its own context showed — not a re-run of this session's probe). The
  retraction stands, both prior fixes are verified working, and the session is archived to
  `.planning/debug/resolved/codegraph-skill-not-listed.md`.

## Resolution

root_cause: >
  AND-gate, two contributing causes, neither sufficient alone — and NO surviving product defect.
  Both prior fixes (description 299->110 bytes, skillListingBudgetFraction 0.015->0.03) work, and
  the second one completely eliminated the degradation class: a fresh claude-sonnet-5 session in
  this repository now renders all 232 entries with descriptions, zero bare.
  (1) ENVIRONMENT — `SessionStart:resume` does not recompute the skill catalog. The user's genuine
  binary restart RESUMED conversation 67ab93a5 (SessionStart:resume at 2026-08-13T14:06:53.154Z)
  rather than starting a new one, so the session replays the `isInitial: true` skill_listing
  attachment recorded at 2026-08-13T00:32:49Z — 13 hours before the description fix (bc1c853,
  13:42:50Z) and 13.5 hours before the settings change (14:03:56Z). In that stale listing codegraph
  is BARE, one of 61 degraded entries of 234. Verification from a resumed session is structurally
  incapable of observing either fix, no matter how genuine the restart.
  (2) DATA/OBSERVATION — the block actually read and reported ("lists exactly one entry,
  engram:curating-spine") is an `isInitial: false` DELTA attachment carrying `skillCount: 1`,
  emitted 3.4 seconds after the resume hook. Claude Code renders deltas under the byte-identical
  system-reminder header the full catalog uses, so the most recent such block in a session is
  routinely a one-entry delta, not the catalog. The same session contains a codegraph delta at
  14:00:00.601Z whose entire content is the correctly-described entry.
  Together these produced "absent entirely" — a state neither the stale listing (bare) nor the fresh
  listing (described) actually exhibits, which is why the symptom looked stronger than the prior
  session's and matched no known mechanism. This is the THIRD instance in this repository of the
  same class: a claim about what an observation instrument showed, mistaken for a claim about the
  world.
fix: >
  No product change — there is nothing broken to fix, and shipping one would have been the real
  defect. What IS fixed is hardened false evidence, the exact failure mode the two prior sessions
  were resolved for.
  (1) THE FALSE INVARIANT, corrected where it was written down. The prior session recorded "~45,000
  characters / exactly 173 descriptions" in three places as though it were a property of Claude
  Code. Direct measurement refutes it: the budget tracks `skillListingBudgetFraction` x the model's
  context window, per that setting's own schema text. Corrected in
  `SKILL-03-rehearsal.md` (the figures are now labelled a measurement, with the tier table),
  in `skillDescriptionListingMaxChars`'s doc comment, and in
  `TestSkillDescriptionSurvivesSkillListingCap`'s failure message. The 120-byte bound is KEPT and
  its rationale sharpened: it must hold at the DEFAULT fraction on an ordinary machine, which this
  operator's raised setting no longer represents.
  (2) THE TWO INVALID ORACLES, recorded so instance four does not happen. A new "Method note"
  section in `SKILL-03-rehearsal.md` names the `isInitial` flag, the single-skill deltas that share
  the catalog's header, the stale-catalog replay on resume, and the only valid check (new
  conversation, read `isInitial: true` from its own transcript, classify against the attachment's
  `names` array rather than by regex, which mis-parses plugin names containing colons). It also
  records that absence has never actually been observed here — every report of it found a bare or
  delta-rendered entry.
verification: >
  TDD red-then-green, no test weakened. RED: 6 assertion failures against the uncorrected artifact.
  GREEN: all skill tests pass; `internal/mcp` ok 3.381s, `internal/agents` ok, `go vet` clean.
  Guardrail signals: (1) the new test fails on the pre-fix artifact and passes on the post-fix one;
  (2) the fix is not deletion-only — removing the false claim without recording its replacement is
  explicitly rejected, since the test requires both "skillListingBudgetFraction" and "context
  window" to be present; (3) MUTATION: 7 mutants, 7 killed, ZERO survivors — re-inserting the
  retracted phrasing, stripping the setting name, stripping "context window", stripping
  "isInitial", stripping "delta", stripping "resume", and deleting the entire Method note section
  each turn the test red; (4) oracle type is `derived` — a measured environmental budget plus the
  attachment schema's own fields, not an implicit crash oracle; (5) reverting the artifact restores
  the failure (demonstrated by mutant M7).
  Boundary coverage of the corrected model: the budget was measured at three distinct values on one
  installation in one day — 23,984 chars (claude-haiku-4-5, 85 of 232 degraded), ~45,000
  (claude-sonnet-5, pre-change fraction, 61-67 degraded), 59,681 (claude-sonnet-5, fraction 0.03,
  0 of 232 degraded) — with the tier boundary coinciding exactly with the settings file mtime.
  (6) HUMAN VERIFICATION VIA AN INDEPENDENT ORACLE — the signal this session actually turned on.
  The user did not accept my probe; they built a second, disjoint one: a fresh `claude` process
  (new session id, explicitly not `--resume`) started in a sibling Herdr pane and asked, with no
  tools, what its OWN injected context showed. It answered that codegraph is listed and quoted the
  description byte-for-byte. That chain shares no link with mine — different process, different
  launcher, different mechanism, and it reports on its own context rather than on a transcript file
  someone selected and parsed. Every prior instance of this bug class here died of single-instrument
  trust; two independent oracles agreeing on the exact string is the standard this class requires.
  NOT verified, deliberately: the exact budget formula. Three measurements bound it and are
  consistent with fraction x context-window, but the fraction was NEVER VARIED under controlled
  conditions, and the prior 0.015 figure is taken from the debug record rather than measured by me.
  Archiving this session does not upgrade that inference to a fact — it remains a bounded hypothesis
  and is recorded as an open gap in the knowledge-base entry.
files_changed:
  - .claude/skills/codegraph/verification/SKILL-03-rehearsal.md (false invariant relabelled as a measurement; new Method note recording the corrected budget model, the two invalid oracles, the only valid check, and the two-disjoint-instruments rule the human verification demonstrated)
  - internal/mcp/skill_claims_drift_test.go (TestSkillListingEvidenceRecordsBudgetIsNotFixed + skillListingEvidencePath/skillListingFixedCapClaims/skillListingOracleTraps added in the red phase; skillDescriptionListingMaxChars doc comment and TestSkillDescriptionSurvivesSkillListingCap failure message corrected)
  - .planning/debug/resolved/codegraph-skill-not-listed.md (this session record)

## Prevention

### Blameless 5-whys, branched (from reasoning_checkpoint.candidate_causes)

**Branch A — environment (CONFIRMED cause 1).** Why did the verification miss the fix? The session
carried a skill listing computed 13 hours earlier. Why? `SessionStart:resume` replays the
transcript's `isInitial: true` attachment instead of recomputing it. Why was the session resumed at
all? The user restarted the binary and the binary resumed the prior conversation — which is its
ordinary, useful behavior. Why did that read as a valid test? Because the *previous* round of this
investigation had asked "was it a real restart or a `/resume`?", got "I restarted the binary," and
recorded that as settling the question. The distinction that mattered was never restart-vs-resume;
it was **new conversation vs resumed conversation**, and no one had characterized that axis yet.
Actionable condition: the check-list for verifying a listing fix did not name the property that
makes an observation valid, so a sincere, careful restart satisfied it while observing nothing.

**Branch B — data/observation (CONFIRMED cause 2).** Why was the skill reported absent? The block
read listed one unrelated skill. Why was that read as the catalog? Because Claude Code delivers
single-skill deltas under a header byte-identical to the full catalog's. Why wasn't the difference
noticed? The distinguishing fields (`isInitial`, `skillCount`) exist only in the attachment record,
not in the rendered text a reader sees. Actionable condition: the rendered surface is
**lossy by construction** — no amount of care reading it recovers what was lost, so the method has
to leave the rendered surface entirely.

**Branch C — code/authoring (REFUTED).** A fresh session renders the same on-disk SKILL.md with its
description. Nothing about the file, its path, or its frontmatter is wrong.

**Branch D — config (REFUTED).** The fresh probe ran in this repository, resolving the same settings
chain, and rendered zero degraded entries. The raised fraction is plainly in effect here.

**AND-gate:** A alone yields a stale listing where codegraph is *bare* — a correct reader would have
reported "listed without a description," i.e. the prior session's symptom recurring. B alone, in a
fresh session, yields "absent" against a correct catalog. Only together do they produce the reported
"absent entirely, a stronger symptom than before" — a state neither listing actually exhibits, which
is precisely why it matched no known mechanism and sent the investigation looking for a new one.

**Blame-free note:** three false reports of this class in one repository in one day is not three
lapses of care — every one of them was produced by a careful person using the instrument the system
offers. The system renders suppressed, degraded, stale, and delta states as text indistinguishable
from the healthy state. That is the actionable condition; "read more carefully" prevents nothing.

### Why wasn't this caught?

**No gate existed for this class, and no gate in this repository could have.** The failure is in a
*verification method*, not in a program state: nothing was malformed (typecheck/vet/lint/build all
inapplicable), every test passed both before and after because the product was never broken, and
code review could not have caught it because there was no defective code to review — the reviewer
would have held the same false evidence artifact and drawn the same conclusion, which is exactly
what happened at the start of this session. The one gate that *looked* relevant,
`TestSkillDescriptionSurvivesSkillListingCap`, was passing throughout and correctly so. The
deeper gap: the prior session shipped its measurements ("~45,000 chars / exactly 173 descriptions")
into a doc comment, a failure message, and an evidence artifact **as invariants**, with no gate
requiring a measured environmental figure to be labelled as a measurement. Hardened false evidence
is the recurring defect in this repository; nothing was watching for it.

### Recurrence guard

`TestSkillListingEvidenceRecordsBudgetIsNotFixed` in `internal/mcp/skill_claims_drift_test.go`
(verified passing 2026-08-13; RED-then-GREEN confirmed with 6 assertion failures pre-fix; 7 mutants,
7 killed, 0 survivors). It pins three properties of `SKILL-03-rehearsal.md` simultaneously, so no
single edit can quietly restore the false model:

1. the retracted invariant phrasings (`skillListingFixedCapClaims`) must be **absent** — re-asserting
   a fixed cap turns it red;
2. the replacement model must be **present** — both `skillListingBudgetFraction` and
   `context window`, so deleting the false claim without recording what the budget actually varies
   with is rejected (this is what makes the fix non-deletion-only);
3. the oracle traps (`skillListingOracleTraps`: `isInitial`, `delta`, `resume`) must be **named**,
   so the two invalid observation methods cannot be dropped from the artifact.

Secondary guard, unchanged and still enforcing: `skillDescriptionListingMaxChars = 120` continues to
bound the description, with its rationale sharpened — it must hold at the **default** fraction on an
ordinary machine, which this operator's raised `0.03` setting no longer represents. Tertiary guard:
this knowledge-base entry, so a future Phase-0 recall surfaces the three invalid oracles (transcript
grep, resumed-session catalog, single-skill delta) before the fourth instance is investigated.
