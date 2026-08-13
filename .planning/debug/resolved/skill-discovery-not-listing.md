---
status: resolved
trigger: "Newly-added project skill (.claude/skills/codegraph/SKILL.md) is correctly placed and committed but is not surfaced in a freshly started session's skill catalog"
created: 2026-08-13
updated: 2026-08-13
---

# Debug Session: skill-discovery-not-listing

## Symptoms

**Expected behavior:** A genuinely fresh Claude Code session started in this repository should
list `codegraph` in its skill-listing system reminder — the mechanism that surfaces
`.claude/skills/*/SKILL.md` files so the model knows the skill exists and can invoke it. The skill
file is correctly placed at `.claude/skills/codegraph/SKILL.md`, has valid-looking frontmatter
(`name: codegraph`, a `description` field), and is committed to the repository.

**Actual behavior:** A freshly started session's own skill catalog — inspected directly in its
transcript, not inferred — did not name `codegraph` at all. Confirmed by grepping that session's
raw transcript, per `.claude/skills/codegraph/verification/SKILL-03-rehearsal.md`.

**Error messages:** None. This is a silent absence — no error, no warning, the skill file simply
does not appear in the session's skill listing.

**Timeline:** First observed 2026-08-12 during the same Phase 6 live-session rehearsal that also
produced the (now-retracted) resume-matcher finding — see
`.planning/debug/resolved/resume-matcher-not-firing.md` for that unrelated, already-closed
investigation. Given that investigation's outcome — a verification-methodology false negative, not
a real defect — this skill-discovery gap should be investigated with the same skepticism toward
the observation method before trusting it as a real product defect.

**Reproduction steps:**
1. Confirm `.claude/skills/codegraph/SKILL.md` exists, is committed, and has frontmatter matching
   the shape other working skills use (`name`, `description`).
2. Start a genuinely fresh Claude Code session in this repository — no prior conversation turns,
   no authoring context.
3. Send a where-is-X / how-does-Y-work prompt that names no tool (the rehearsal used: "How does
   the daemon decide when to trigger a re-sync of the graph store?").
4. Inspect the session's own transcript directly for its skill-listing system reminder (the
   mechanism that lists available skills, distinct from the `<system-reminder>` blocks visible in
   this transcript for other purposes).
5. Observe: `codegraph` is absent from that listing, despite the file being present and committed.

## Evidence

- timestamp: 2026-08-12 (Phase 6 rehearsal)
  source: .claude/skills/codegraph/verification/SKILL-03-rehearsal.md ("After" section + Verdict)
  finding: >
    A freshly captured session's skill catalog (inspected directly, not inferred) did not list
    `codegraph`. Despite that, the session's actual tool-choice behavior was still correct — its
    first code-search action for the where-is-X-class prompt was `codegraph_explore`, not
    grep/find/Read — but the rehearsal explicitly could NOT attribute that correct routing to the
    skill itself, because two other independent mechanisms already in context (the operator's
    global `~/.claude/CLAUDE.md` CodeGraph section, and the codegraph MCP server's own
    `instructions` string) are each independently sufficient to explain the same outcome. So the
    skill's absence from the catalog was not masked by a passing behavioral test — it went
    unnoticed by the *literal* SKILL-03 criterion while remaining a real, separately-flagged gap.

- timestamp: 2026-08-12
  source: .planning/STATE.md (Blockers/Concerns)
  finding: >
    Recorded as an open follow-up, not closed by Phase 6: "Newly-added project skill not surfaced
    to a fresh session ... Needs investigation into why project-skill discovery didn't pick up a
    newly-added, correctly-placed `.claude/skills/*/SKILL.md` in a fresh session."

- timestamp: 2026-08-13 (pre-investigation check)
  checked: .claude/skills/codegraph/SKILL.md frontmatter and directory placement, and whether any
    sibling directories/files exist under .claude/skills/ that could shadow or conflict with it
  found: >
    Frontmatter is `name: codegraph` / `description: <one paragraph, no obvious syntax error>`.
    Directory contains only SKILL.md plus a `verification/` subdirectory with two evidence
    markdown files (not skill content). No naming collision, no second SKILL.md, no obviously
    malformed YAML observed on a plain read. This does not rule out subtler causes (encoding,
    line-ending, a required field this repo's Claude Code version expects that isn't in the
    frontmatter, a discovery-timing issue, a cache, etc.) — those are for the investigation to
    determine, not assumed here.

- timestamp: 2026-08-13T13:00Z
  checked: Knowledge-base match (Phase 0). Semantic + keyword match against
    `.planning/debug/knowledge-base.md`.
  found: >
    Strong match on `resume-matcher-not-firing` (2026-08-13): same reporting session, same
    rehearsal artifact family, same symptom grammar ("registered/placed correctly, no error, just
    silently absent from the transcript"), and the same oracle — a grep of a session transcript.
    Its transferable lesson is directly applicable: "Before concluding a component does not run,
    prove your oracle can observe it running. Absence of evidence in a log is a claim about the
    log, not about the world." Treated as hypothesis candidate H1 and tested FIRST.

- timestamp: 2026-08-13T13:02Z
  checked: This debugging agent's OWN skill catalog, visible verbatim in its context.
  found: >
    The catalog contains an entry `- codegraph` — present, but BARE: no description text after the
    name, unlike essentially every other entry which renders as `- name: description`. First
    direct evidence that (a) discovery does find the skill and (b) something distinct is wrong
    with the description.

- timestamp: 2026-08-13T13:05Z
  checked: git history of `.claude/skills/codegraph/SKILL.md` vs the rehearsal session's start time
  found: >
    Frontmatter is byte-identical across both commits that touch the file — `311a075` (2026-08-12
    18:30:49 -0400, skeleton) and `74e8e22` (19:08:14 -0400, worked examples). The rehearsal
    session's first user turn is 2026-08-12T23:37:28Z = 19:37 EDT, i.e. AFTER both. ELIMINATES
    "the frontmatter was different / the file did not yet exist at rehearsal time".

- timestamp: 2026-08-13T13:08Z
  checked: Located the actual rehearsal transcript by its verbatim prompt —
    `~/.claude/projects/-Volumes-Code-github-com-seanb4t-codegraph-go/c4b8f662-07d5-4549-ba5d-be3f7ba795d2.jsonl`
    (first user message is exactly "How does the daemon decide when to trigger a re-sync of the
    graph store?", 2026-08-12T23:37:28.913Z, Claude Code 2.1.229). Extracted its skill catalog.
  found: >
    THE ORIGINAL FINDING IS FALSE. The catalog IS recorded in that transcript, as an `attachment`
    record with `"type":"skill_listing"`, and it DOES contain `codegraph`. Verbatim excerpt:
      ...\n- terraform: Terraform Cloud operations and registry documentation lookup.\nWatch runs,
      view plan/apply logs, check workspace status, look up\nprovider docs. Invokes Terraform MCP
      server on-demand without loading\ntool definitions into context.\n- codegraph\n-
      c4-architecture:c4-architecture\n- claude-md-management:revise-claude-md: ...
    `codegraph` is appended at the END of the personal-skill block (which is otherwise
    alphabetical, ending at `terraform`) — exactly where a project-scoped `.claude/skills/` entry
    belongs. So project-skill DISCOVERY WORKED in the very session the finding was written from.
  implication: >
    The reported symptom ("not surfaced in the skill catalog") is a false negative, same class as
    resume-matcher. The entry renders as a bare `- codegraph` with no colon and no description, so
    any grep keyed on `codegraph:` or on the description text misses it. H1 CONFIRMED as to the
    literal claim. But a real, narrower defect survives: the DESCRIPTION is missing.

- timestamp: 2026-08-13T13:12Z
  checked: Classified all 142 personal/project entries in that same catalog block into
    with-description vs bare, then joined against each skill's on-disk frontmatter.
  found: >
    138 entries carry a description; exactly 4 are bare: `obsidian-cli`, `pitch-packager`,
    `simple-english`, `codegraph`. Three of those four (obsidian-cli len=467, pitch-packager
    len=243, codegraph len=297) have a well-formed, non-empty `description:` on disk. So "bare in
    catalog" does NOT mean "missing description in frontmatter" — the loader is dropping a
    description that is present.
  implication: >
    A real defect exists and is reproducible across four unrelated skills, i.e. it is not specific
    to this repo's authoring. Without a description in the catalog, the model has no trigger
    signal — the skill is listed but semantically undiscoverable, which fully explains the
    downstream symptom the rehearsal actually cared about (skill did not demonstrably fire).

- timestamp: 2026-08-13T13:16Z
  checked: Candidate discriminators between the 4 bare and the 138 working entries — frontmatter
    key set, BOM, CRLF, tabs, trailing whitespace, description length, non-ASCII characters,
    SKILL.md file size, directory contents (subdirs present/absent), SKILL.md mtime.
  found: >
    NONE discriminate. Bare and working entries are structurally identical on every axis measured:
    obsidian-cli (467 chars, ASCII, no subdir, mtime 2026-05-24) is bare while obsidian-bases (256
    chars, ASCII, has subdir, mtime 2026-06-25) works; agent-browser works at 925 chars; codegraph
    is the only bare one with non-ASCII. No skill metadata cache exists under `~/.claude`.
  implication: Correlation-hunting is exhausted; a controlled reproduction is required.

- timestamp: 2026-08-13T13:30Z
  checked: Controlled reproduction. Built a scratch project outside this repo carrying seven
    synthetic project skills — codegraph's description verbatim (`t-exact`), the same description
    with every em dash and apostrophe stripped (`t-ascii`), a 59-char ASCII control (`t-short`),
    obsidian-cli's and pitch-packager's descriptions verbatim (`t-obscli`, `t-pitch`),
    obsidian-bases' description verbatim (`t-ctrl`), and a byte-for-byte copy of the real
    `.claude/skills/codegraph/SKILL.md`. Ran headless `claude -p` there and read the
    `skill_listing` attachment.
  found: >
    Reproduced outside this repo. `t-short` rendered WITH its description; `codegraph`, `t-exact`,
    `t-ascii`, `t-obscli`, `t-pitch` AND `t-ctrl` all rendered BARE — even though `t-ctrl`'s
    description is byte-identical to obsidian-bases', which rendered WITH its description in the
    very same listing.
  implication: >
    Decisive. The same description text renders both ways in one listing, so the cause cannot be
    any property of the SKILL.md file. Also refutes the non-ASCII theory outright: `t-ascii` has
    every em dash and apostrophe removed and is still degraded.

- timestamp: 2026-08-13T13:40Z
  checked: Determinism. Ran the identical probe project four times end to end and diffed the
    resulting `skill_listing` attachments.
  found: 4/4 runs identical — 238 entries, 173 with descriptions, 65 bare, same membership.
  implication: Bohrbug, not a race or timeout. Deterministic function of the installed skill set.

- timestamp: 2026-08-13T13:45Z
  checked: Measured the rendered `skill_listing` attachment across four independent sessions (the
    2026-08-12 rehearsal, two other real sessions in this repo, and the probe project).
  found: >
    A hard cap. Rendered listing length clamps just under 45,000 characters in every session —
    44,976 / 45,002 / 45,002 / 45,014 — and the number of entries carrying a description is
    EXACTLY 173 in all four, while total entry counts differ (234 / 234 / 235 / 238) and bare
    counts move with them (61 / 61 / 62 / 65). Adding 3 net skills to the set added exactly 3 bare
    entries and left `withDesc` pinned at 173.
  implication: >
    ROOT CAUSE. Claude Code renders the available-skills catalog into a capped attachment; entries
    that do not fit are emitted as a BARE NAME with no description. This installation carries ~238
    skills totalling ~39,000 characters of descriptions, saturating the cap, so ~65 entries are
    degraded. Project-scoped `.claude/skills/` entries are appended AFTER every personal and
    plugin skill, putting this repository's skill at the back of the queue by construction.

- timestamp: 2026-08-13T13:52Z
  checked: >
    Fix-direction test. In the otherwise byte-identical 238-skill probe environment, replaced ONLY
    codegraph's description (297 chars) with a 65-char one and re-ran.
  found: >
    `- codegraph: Route where-is-X and how-does-Y questions to the codegraph index.` — admitted
    WITH its description. `withDesc` stayed pinned at 173 and another entry was displaced into the
    bare set, confirming a fixed-size budget being competed for rather than a per-skill verdict.
  implication: A shorter description is a real, verified lever this repository actually controls.

- timestamp: 2026-08-13T13:58Z
  checked: >
    Threshold. Set the five sibling probe skills to descriptions of 250 / 200 / 160 / 130 / 100
    characters at adjacent catalog positions and re-ran once.
  found: >
    100 admitted; 130, 160, 200 and 250 all degraded to bare. The 59-char `t-short` was itself
    displaced this run once codegraph (65) and t-pitch (100) consumed the leftover room.
  implication: >
    Admission is greedy competition for leftover budget, so NO length guarantees admission — the
    headroom depends on the operator's installed-skill count, which this repository does not
    control. 120 characters is the tightest bound the data supports (admitted at 100, degraded at
    130) and is what the recurrence guard pins.

## Eliminated

- hypothesis: The skill file did not exist, or had different/malformed frontmatter, at the time of
    the rehearsal session.
  evidence: Frontmatter byte-identical across both commits touching the file (18:30 and 19:08
    EDT); rehearsal session's first turn is 19:37 EDT, after both.
  timestamp: 2026-08-13T13:05Z

- hypothesis: Project-skill discovery failed to pick up `.claude/skills/codegraph/SKILL.md` — the
    skill was genuinely absent from the fresh session's catalog (the reported symptom, verbatim).
  evidence: The rehearsal session's own transcript
    (c4b8f662-07d5-4549-ba5d-be3f7ba795d2.jsonl) contains an `attachment` record of
    `"type":"skill_listing"` whose content includes `\n- codegraph\n`, positioned at the end of the
    personal-skill block where a project-scoped skill belongs. Discovery worked.
  timestamp: 2026-08-13T13:08Z

- hypothesis: The bare rendering is simply "the frontmatter has no description" (i.e. an authoring
    error in SKILL.md).
  evidence: 3 of the 4 bare entries — obsidian-cli, pitch-packager, codegraph — have well-formed
    non-empty `description:` values on disk (467/243/297 chars).
  timestamp: 2026-08-13T13:12Z

- hypothesis: The description is dropped because of an encoding/formatting property of the file —
    BOM, CRLF, tabs, trailing whitespace, non-ASCII characters, or description length.
  evidence: Measured all six across the 4 bare and 5 control entries. No BOM, no CRLF, no tabs, no
    trailing whitespace anywhere. Length does not discriminate (bare: 243/297/467; working: 256,
    262, 925). Non-ASCII does not discriminate (3 of 4 bare are pure ASCII).
  timestamp: 2026-08-13T13:16Z

- hypothesis: A stale skill-metadata cache holds an empty description for recently-added skills.
  evidence: No skill cache exists — `~/.claude/cache/` holds only `changelog.md` and
    `my-closed-issues.json`. Also refuted by mtime: obsidian-cli's SKILL.md is from 2026-05-24,
    older than most working entries.
  timestamp: 2026-08-13T13:16Z

- hypothesis: The description is dropped because of something in this repository's SKILL.md — its
    wording, its em dashes/apostrophes, or any other authored property of the file.
  evidence: In one probe listing, `t-ctrl` (description byte-identical to obsidian-bases') rendered
    BARE while obsidian-bases rendered WITH its description; and `t-ascii` (codegraph's description
    with every non-ASCII character stripped) still rendered bare. Identical text, both outcomes,
    same listing.
  timestamp: 2026-08-13T13:30Z

- hypothesis: A race or timeout in the skill loader — descriptions belong to whichever SKILL.md
    reads finish before a deadline.
  evidence: Four end-to-end runs of the identical probe project produced byte-identical listings —
    238 entries, 173 with descriptions, 65 bare, same membership every time.
  timestamp: 2026-08-13T13:40Z

- hypothesis: The cap is a running budget consumed in display order, so everything after the
    cutoff point is bare.
  evidence: Bare entries interleave with described ones throughout the listing, and cumulative
    description characters at codegraph's display position (index 141) were only 29,605 of the
    ~39,000 the same listing ultimately spent. Entries at indices 147, 149, 153-158 and 162-166
    carry descriptions after several bare ones. Selection order is not display order.
  timestamp: 2026-08-13T13:45Z

## Current Focus

known_pattern_candidate: "resume-matcher-not-firing — transcript-grep oracle produced a false
  negative about a mechanism that was working. CONFIRMED to apply here: the literal reported
  symptom is false."

bug_class: Bohrbug — deterministic (4/4 identical probe runs, 4/4 real sessions).

tdd_checkpoint:
  test_file: "internal/mcp/skill_claims_drift_test.go"
  test_name: "TestSkillDescriptionSurvivesSkillListingCap"
  status: "green"
  red_output: >
    skill_claims_drift_test.go:311: ../../.claude/skills/codegraph/SKILL.md frontmatter
    description is 299 characters, over the 120-character skill-listing budget
  green_output: >
    --- PASS: TestSkillFrontmatterIsSpecCompliant (0.00s)
    --- PASS: TestSkillDescriptionSurvivesSkillListingCap (0.00s)
    ok github.com/seanb4t/codegraph-go/internal/mcp 0.335s. Full package: ok, 3.564s. Neither test
    was weakened to reach green — the only source change is the description string itself.

reasoning_checkpoint:
  hypothesis: >
    The reported symptom is false — project-skill discovery works and `codegraph` IS in the
    catalog. The real defect is that Claude Code caps the rendered skill listing at ~45,000
    characters and degrades every entry past the cap to a bare name with no description. On this
    operator's ~238-skill installation the cap is saturated, and project-scoped skills are appended
    last, so codegraph's description is dropped — leaving an entry with zero trigger signal, which
    is why SKILL-03 could not attribute the routing to the skill.
  confirming_evidence:
    - "The rehearsal session's own transcript (c4b8f662) contains a skill_listing attachment whose content includes a bare `- codegraph` entry — the finding it produced says the opposite."
    - "Rendered listing length clamps at 44,976-45,014 chars and withDesc is EXACTLY 173 across four independent sessions with 234/234/235/238 total entries."
    - "Adding 3 net skills added exactly 3 bare entries while withDesc stayed pinned at 173."
    - "`t-ctrl`, whose description is byte-identical to obsidian-bases', rendered bare in the same listing where obsidian-bases rendered with its description."
    - "Shortening ONLY the description from 297 to 65 chars, in a byte-identical 238-skill environment, admitted the entry."
  falsification_test: >
    Shorten the description and observe the entry still rendering bare in the same environment; or
    find a session whose listing exceeds ~45,000 characters; or observe withDesc differing from 173
    while the installed skill set is unchanged. None occurred across 8 measured sessions.
  fix_rationale: >
    Addresses the mechanism, not the symptom. Two defects, each fixed at its own root. (1) The
    false finding is retracted at its source — it is hardened evidence that would misdirect the
    next reader exactly as the knowledge base warns, and deleting it silently would lose the
    lesson. (2) The description is cut under the measured admission band so the entry competes for
    the leftover budget instead of being dropped, which is the only lever this repository holds
    over an upstream cap.
  blind_spots: >
    The exact upstream selection algorithm is uncharacterized — 173 is stable but why that number
    rather than a pure character budget is unknown, and selection order is provably not display
    order. No length guarantees admission: headroom is a function of the operator's installed skill
    count. All measurements are one machine, Claude Code 2.1.229/2.1.231, darwin. On a normal
    ~30-skill installation the cap is never reached and the original 297-char description would
    have rendered fine, so this fix trades trigger detail for survival on loaded machines.
  candidate_causes:
    - "code/authoring — malformed or hostile SKILL.md frontmatter (REFUTED: t-ctrl, t-ascii)"
    - "environment — Claude Code's ~45,000-char skill-listing cap saturated by ~238 installed skills (CONFIRMED)"
    - "data/observation — the verification oracle was a grep of a transcript for text that renders without a colon (CONFIRMED)"
  and_gate: >
    Yes — two contributing causes, both required, neither sufficient alone. The environmental cap
    alone produces a degraded-but-listed entry that a correct oracle would have described
    accurately as "listed without a description". The oracle flaw alone would have had nothing to
    misreport. Together they produced a confident, wrong, and durable claim that project-skill
    discovery is broken. `root_cause` therefore holds both.

next_action: >
  None — session complete and archived. Green phase applied: description cut 299 -> 110 chars, both
  tests pass, false finding retracted at all three sources, postmortem and knowledge-base entry
  recorded.

## Resolution

root_cause: >
  AND-gate, two contributing causes, both required. (1) ENVIRONMENT — Claude Code renders the
  available-skills catalog into an attachment capped at ~45,000 characters (measured 44,976-45,014
  across four sessions, admitting exactly 173 descriptions each time) and degrades every entry past
  the cap to a bare name carrying no description; this operator's installation holds ~238 skills
  totalling ~39,000 characters of descriptions, saturating the cap, and project-scoped
  `.claude/skills/` entries are appended after all personal and plugin skills, so this repository's
  skill is degraded by construction. (2) DATA/OBSERVATION — the verification oracle was a grep of
  the rehearsal session's transcript, and a degraded entry renders as `- codegraph` with no colon
  and no description, so the grep returned a false negative that SKILL-03-rehearsal.md then
  hardened into "the session's own skill catalog ... does not list the `codegraph` skill at all".
  Discovery was never broken: that same transcript contains the entry.
fix: >
  Two defects, each fixed at its own root, matching the two-cause AND-gate.
  (1) THE DESCRIPTION (environment cause, mitigated at the only lever this repo holds). Cut
  `.claude/skills/codegraph/SKILL.md`'s frontmatter description from 299 to 110 bytes — "Use when
  asked where X is defined, how Y works, what calls X, or what changing X breaks in a .codegraph/
  repo." Kept trigger-shaped (retains "when", covers all four question shapes plus the gating
  `.codegraph/` condition) and kept pure-ASCII, since Go's `len()` and the renderer both count
  bytes and an em dash costs three. 110 sits inside the measured admission band (65 and 100
  admitted; 130, 160, 200, 250 degraded). The upstream cap itself is not ours to fix.
  (2) THE FALSE FINDING (observation cause, fixed by retraction-in-place, not deletion). Retracted
  at all three sources that carried it, each stating what was actually observed (the entry WAS
  listed, degraded to a bare name) and why the grep oracle missed it (a bare entry has no colon and
  no description text to match), so a future reader cannot re-derive the wrong conclusion:
  `SKILL-03-rehearsal.md` (the "Whether the skill triggered" claim and the Verdict caveat that
  restated it), `NUDGE-live-session.md` (follow-up item 2 plus the shared-lesson note tying it to
  resume-matcher), and `.planning/STATE.md` (the blocker entry, replaced with its resolution in the
  same style as the already-retracted NUDGE-01 entry directly above it).
verification: >
  TDD red-then-green, no test weakened. RED (independently re-verified by the session manager before
  the green phase): `TestSkillDescriptionSurvivesSkillListingCap` failed at
  skill_claims_drift_test.go:311 on the 299-char description against the 120-char budget, while
  `TestSkillFrontmatterIsSpecCompliant` passed — establishing the harness itself was sound and only
  the new bound was red. GREEN: both pass; full `go test ./internal/mcp/` passes (3.564s). The only
  source change reaching green is the description string.
  Guardrail signals: (1) the regression test fails on the pre-fix file and passes on the post-fix
  file; (2) the fix is not deletion-only — it replaces a string and its length is asserted from both
  sides (non-empty is a `t.Fatalf`, so the trivial "" that would satisfy a maximum-length bound
  while reproducing the exact defect is explicitly killed); (3) the two bounds cannot collapse into
  one another — the test `t.Fatalf`s unless skillDescriptionListingMaxChars is STRICTLY tighter than
  skillDescriptionMaxChars; (4) oracle type is `derived` (a measured environmental budget), not
  `implicit`; (5) reverting the description restores the failure.
  Boundary coverage of the fixed class: admission measured at 65 and 100 chars, degradation at 130,
  160, 200 and 250, so the 120 bound is pinned between the nearest observed admit (100) and the
  nearest observed degrade (130) rather than at an arbitrary round number.
  NOT verified, and deliberately so: that 110 chars *guarantees* admission. Admission is greedy
  competition for leftover budget and depends on the operator's installed-skill count. This fix
  makes the entry competitive; it cannot make it certain.

prevention: >
  BLAMELESS POSTMORTEM.

  Why no existing gate caught it. Three gates looked relevant and all three were structurally
  incapable. (a) `TestSkillFrontmatterIsSpecCompliant` checked the description against
  agentskills.io's 1024-character maximum — the published spec limit, which is ~8x looser than the
  real operative ceiling. The 299-char description was comfortably legal by the documented contract
  and still lost its description in practice, because the binding constraint was never the spec, it
  was a renderer budget nobody had measured. A gate calibrated to the documented limit cannot catch
  a violation of an undocumented one. (b) Typecheck, lint, vet and build could not apply — nothing
  was malformed; the YAML was valid and the file was exactly what its author intended. (c) Code
  review could not apply — a reviewer holding the same hardened evidence artifact would have read
  "project-skill discovery is broken" and gone looking upstream, which is precisely what happened.
  Deeper cause: the SKILL-03 acceptance criterion was written as a behavioral test ("the session
  picks codegraph_explore first") and passed, while the property that actually mattered (the skill
  reached the session with a usable trigger signal) was never a criterion at all. The rehearsal
  noticed the gap only incidentally, and then mis-described it.

  Blameless framing. The wrong finding was a reasonable inference from a reasonable check. Grepping
  a transcript for a skill's name is the obvious oracle, and it fails silently here for a
  non-obvious reason: the runtime renders a degraded entry in a *different shape* (`- codegraph`,
  no colon) than a healthy one (`- name: description`). Nothing warned that the shape could change.
  The cost was not the wrong conclusion itself but its durability — it was written into three
  artifacts, one of which (STATE.md) is read as project truth, and it named the wrong subsystem, so
  any follow-up would have started in the wrong place.

  What guard now exists. `TestSkillDescriptionSurvivesSkillListingCap`
  (internal/mcp/skill_claims_drift_test.go) pins the description at <=120 bytes, reads the same
  field the spec-compliance test reads so the two cannot drift, refuses to run unless its bound is
  strictly tighter than the spec bound (so it can never silently decay into a restatement of the
  looser gate), and `t.Fatalf`s on an empty description (so the degenerate way to satisfy a
  maximum-length bound — delete the description — is the one thing it will not accept). The
  constant carries the measurements inline, so the next person to loosen it has to argue with data.
  The retractions are the guard against the *observation* failure: each names the invalid oracle
  explicitly, and `NUDGE-live-session.md` now states the shared lesson once for both instances.

  Known gap, not fixed. The upstream selection algorithm remains uncharacterized — 173 admitted
  descriptions is stable across every measured session, but why that count rather than a pure
  character budget is unknown, and selection order is provably not display order. All measurements
  are one machine, Claude Code 2.1.229/2.1.231, darwin. On a normal ~30-skill installation the cap
  is never reached and the original 299-char description would have rendered fine, so this fix
  trades trigger detail for survival on loaded machines — a deliberate trade, not a free win. No
  test can enforce admission itself; only competitiveness.

files_changed:
  - .claude/skills/codegraph/SKILL.md (description 299 -> 110 bytes, trigger-shaped, ASCII)
  - .claude/skills/codegraph/verification/SKILL-03-rehearsal.md (retraction + corrected Verdict caveat + standing method caution)
  - .claude/skills/codegraph/verification/NUDGE-live-session.md (follow-up item 2 retracted; shared transcript-grep lesson recorded)
  - .planning/STATE.md (blocker entry replaced with its resolution)
  - internal/mcp/skill_claims_drift_test.go (TestSkillDescriptionSurvivesSkillListingCap + skillDescriptionListingMaxChars — added in the red phase)
