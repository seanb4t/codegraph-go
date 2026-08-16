---
phase: 06-benchmark-de-coupling-memory-sweep
requirement: MEM-02
scope: agent-facing framing files (D-12) — the engram spine sections are appended by 06-06
---

# Phase 6 Memory Sweep — Agent-Facing Framing Files

D-12's test, restated: MEM-02's stated outcome ("a session started after the sweep recalls no
memory describing codegraph-go as a port or parity project in the present tense") is defeated by
`.claude/CLAUDE.md`, `.planning/PROJECT.md` and `.planning/STATE.md` regardless of the engram
spine's state, because these three files are read at every session start. D-13's test — applied to
prose in this section — is the supersession rule: **sweep** an occurrence that states the framing
as a current standing fact or a future obligation; **keep, untouched** an occurrence that records
what was decided, measured or shipped at a point in time (rewriting those falsifies project
history, the same failure mode as editing `.planning/` archives).

## Census instrument

Pattern set (word-bounded, `rg -U -o -i -w`, counted with `wc -l`, never `rg -c`): `parity`,
`port`, `ported`, `drop-in`, `upstream`, `rewrite of`, `TS\s+CodeGraph` (multiline-safe form of
`TS CodeGraph` — a literal space does not cross a line break even under `-U`; a whitespace class
is required to bridge a wrapped occurrence), `colbymchenry`, `reimplementation`.

**Positive control** (proves the instrument before any zero is trusted). Fixture — a hard-wrapped
prose paragraph, representative of the markdown prose these three files are, not of a
comment-prefixed code fixture:

```
This project reaches parity with the original, having been
ported with attribution from colbymchenry's TS
CodeGraph as a rewrite of the upstream drop-in
reimplementation.
```

`POSCTRL_MULTILINE=9 POSCTRL_LINEBASED=7` — multiline strictly beats line-based (9 > 7) and both
are strictly greater than 0. The extra multiline matches over the same match count are a known
`rg -o`/`wc -l` interaction: a single match that itself spans a newline (`TS\nCodeGraph`) is
printed with its embedded newline intact, so `wc -l` counts it as two lines for one match — the
same shape 06-CENSUS.md's own positive control recorded (`POSCTRL_MULTILINE=6 POSCTRL_LINEBASED=2`
for a comment-prefixed fixture). The instrument is proven live.

**Raw hit counts** over the declared three-file surface, `rg -U -o -i -w`:

| File | Raw hits |
|---|---|
| `.claude/CLAUDE.md` | 7 |
| `.planning/PROJECT.md` | 75 |
| `.planning/STATE.md` | 4 |
| **Total** | **86** |

`.claude/CLAUDE.md` hits land on lines **7, 15, 17, 83, 93** — measured exactly as predicted
during planning. `.planning/STATE.md` hits land on lines **26, 115, 170, 188**.

**One instrument gap found and recorded, per this task's own instruction ("if the census failed
to find one, the SUMMARY records the pattern that missed it and the correction"):**
`.planning/PROJECT.md:244` (the Compatibility constraint's stale-capability clause) is a
pre-identified occurrence from planning that the word-pattern census does **not** find — its text
contains no pattern-set word (`compatibility`, `codegraph migrate`, `functionality baseline` are
not in the set). This is not a framing-word defect; it is a **stale-capability-reference** defect
(a present-tense constraint naming a capability Phase 5 removed), caught instead by Task 2's
dedicated `COMPATIBILITY_BULLET_STALE_CAPABILITY_REFS` check. It still gets its own row below,
found manually from the plan's pre-identified table rather than by the word census, because the
row-completeness requirement ("every occurrence named in the planning table appears as a row")
does not depend on which instrument surfaced it.

## Agent-facing framing files (D-12)

| File | Line | Quoted text | Verdict | Reason |
|---|---|---|---|---|
| `.planning/PROJECT.md` | 5 | "Originated as a ground-up Go rewrite of [CodeGraph](https://github.com/colbymchenry/codegraph) (MIT-licensed; ported with attribution)." | sweep | Sits in the paragraph a session reads as the project's current identity (`## What This Is`). The licence attribution stays in `NOTICE`, the single sanctioned site the Licensing constraint (line 246) already names. |
| `.claude/CLAUDE.md` | 7 | "Originated as a ground-up Go rewrite of [CodeGraph](https://github.com/colbymchenry/codegraph) (MIT-licensed; ported with attribution)." | sweep | Mirrors `.planning/PROJECT.md:5` inside the generated `GSD:project-*` block. Source-before-mirror: fix PROJECT.md, then re-sync this block. |
| `.claude/CLAUDE.md` | 15 | "**Compatibility**: TS CodeGraph v1.3.x is a **functionality baseline, not a binding constraint** (maintainer ruling 2026-08-09) — match its agent-facing capability surface (MCP tools, CLI semantics), but deliberate, documented divergence is acceptable where its behavior is worse. A TS behavior is evidence about what users expect, not a veto; do not cite this line as a blocker without weighing that ruling. See Key Decisions" | sweep | Stale mirror — `.planning/PROJECT.md:244` already retired this constraint at v0.11.0 scoping (2026-08-13). Re-sync rather than re-author. |
| `.claude/CLAUDE.md` | 17 | "**Licensing**: Original is MIT — port with attribution" | sweep | Stale mirror of `.planning/PROJECT.md:246`'s current, already-corrected Licensing wording. Re-sync rather than re-author. |
| `.planning/PROJECT.md` | 244 | "**Compatibility**: **Retired as a constraint (v0.11.0, 2026-08-13).** ... What still binds is narrower and concrete: `codegraph migrate` must keep reading a legacy `.codegraph/` SQLite index one-way. That is a feature contract with existing users, not a behavioral obligation. See Key Decisions" | sweep | Compound defect: a present-tense standing constraint naming a capability Phase 5 removed entirely (`codegraph migrate` no longer exists — CODE-03). Not found by the word census (see "instrument gap" above); found manually from the plan's pre-identified table. |
| `.planning/PROJECT.md` | 219 | "Local Svelte web UI for browsing/querying the graph (SEED-001 — triggers once parity lands; the v1.0 bubbletea TUI is a distinct terminal surface)" | sweep | Forward-looking trigger condition — a future obligation phrased against another implementation's state, D-13's second supersede class. |
| `.planning/PROJECT.md` | 220 | "Worktree support **beyond** TS parity — auto-init or `git-common-dir` index sharing (v1.0 ships TS-parity detect+warn+notice only; going further is a deliberate later call)" | sweep | Parity used as the scoping yardstick for future worktree work. Restated on the project's own terms — naming the shipped behaviour a future item would go beyond. |
| `.planning/PROJECT.md` | 234 | "**User environment:** Sean runs TS CodeGraph daily as an MCP server across his projects, so parity gaps will be felt immediately — a strength for validation. Work team uses Java/C# heavily (hence its priority position); open-source release is intended from day one, so docs, release engineering, and compatibility promises matter early." | sweep | Present-tense operating description ("Sean runs TS CodeGraph daily") plus a forward-looking consequence clause ("parity gaps will be felt", "compatibility promises matter early") — both present-tense and forward-looking, and factually stale now that Compatibility is retired as a constraint. |
| `.planning/STATE.md` | 26 | "**Core value:** An agent user can uninstall TS CodeGraph, install the Go binary, migrate their indexes, and everything works the same or better — faster, from a single verifiably-built binary. **As of v1.0 this is delivered, not aspirational.**" | sweep | Compound defect: retired competitive framing (maintainer ruling — the migration/uninstall byproduct framing is not codegraph-go's core value) plus a removed capability (`codegraph migrate` no longer imports a legacy index the way this sentence describes). Replaced with the statement `.planning/PROJECT.md`'s `## Core Value` section already carries, so mirror and source agree. |
| `.claude/CLAUDE.md` | 93 | "Don't block v1 parity on this decision either way — the two APIs are structurally similar enough that a later swap is a bounded refactor, not a rewrite." | sweep | **Sweep IN PLACE**, inside the otherwise keep-divergent stack block (fixed verdict, not left conditional — cycle-2 HIGH 3). Restated on the project's own terms: a bounded later refactor, not a blocker for shipping. Does not re-sync the surrounding block from `.planning/research/STACK.md`; see the range row below. |
| `.claude/CLAUDE.md` | 21-141 (whole `GSD:stack-*` block, including line 83) | Block header: `## Technology Stack` … through `## Sources`. Line 83 specifically: "### Option C — Native pure-Go tree-sitter reimplementation — NOT RECOMMENDED for v1, monitor only" | keep-divergent | The block mirrors the ORIGINAL v1 tech-stack research; its named source, `.planning/research/STACK.md`, now holds a wholly different v0.11.0 document (agent-onboarding / MCP Resources / skills-hooks-plugins research — confirmed this task: `rg -n '^#' .planning/research/STACK.md` returns 15 headings sharing nothing with the stack block beyond `### Core Technologies`, and `rg -i parity .planning/research/STACK.md` returns no match). Re-syncing would replace ~119 lines of live agent instructions and pull `drop-in` in from `STACK.md:151` — verified this task (`rg -n -i 'drop-in' .planning/research/STACK.md` → line 151, "schema is NOT drop-in compatible with Claude Code's"), which is itself in this plan's `CLAUDE_MD_FRAMING_TOTAL=0` pattern set, so a re-sync would break the gate meant to prove the sweep. **This row also absorbs `.claude/CLAUDE.md:83`, which does NOT take a row of its own** (cycle-5 MEDIUM 1): line 83's `reimplementation` hit is the parser-runtime sense of the word ("Native pure-Go tree-sitter reimplementation"), unrelated to project framing, and lies inside the range this task may only touch at line 93 — its only semantically correct verdict is `keep-divergent` (not `sweep`, outside the sanctioned edit scope and the prohibitions forbid re-authoring the block; not `keep-historical`, it is not a past-tense record of a decision). A second `keep-divergent` row would drive `DIVERGENT_VERDICTS` to 2 against the plan's exact `=1` gate. Concern raised for the record: this divergence is not resolved by this phase — the next legitimate regeneration of `.claude/CLAUDE.md` will overwrite the block regardless, which is out of Phase 6's scope. |
| `.planning/PROJECT.md` | 31, 40, 45, 50-51, 58, 62-66, 143-180, 236, 252-273, 299 | Milestone records (e.g. "v1.0 — Drop-in Parity & Human UX: SHIPPED 2026-08-03... codegraph-go is now a true drop-in replacement for TS CodeGraph v1.3.x"), the v1.0 goal `<details>` block, Requirements → Validated bullets, the Current-Milestone goal/Target-features text (which uses "parity"/"upstream"/"drop-in" as the *objects being removed*, not as standing framing), Key Decisions outcomes, and the footer change log | keep-historical | Past-tense records of what was decided, measured or shipped, or (for the Current-Milestone goal and Target-features bullets at 58, 62-66) meta-referential use of the terms as the subject of this very sweep, never an assertion that codegraph-go currently IS a port. Rewriting any of these falsifies project history — the same failure mode as editing `.planning/` archives. This is the same range the plan's own pre-identified table names. |
| `.planning/PROJECT.md` | 232 | "**Source project:** colbymchenry/codegraph, TypeScript (~4.5 MB source), v1.3.1, ~59k stars, MIT license. ... Distributes via `curl \| sh` installer or npm, bundling its own Node.js runtime — the supply-chain surface this port eliminates." | keep-historical | Factually describes the *source* project (what TS CodeGraph is), not codegraph-go's own present-tense identity. Part of "the surrounding historical paragraphs" the plan's Task 2 action explicitly leaves alone — only line 234's operating description and consequence clause are in scope for this Context section. Flagged by the census (word "port" in the closing clause) but adjudicated keep on the strength of the plan's own scoping language, not swept by inference. |
| `.planning/PROJECT.md` | 246 | "**Licensing**: MIT. Attribution for the project this one began as a reimplementation of lives in `NOTICE` — the single sanctioned place — and is discharged there, not restated across the repo. `LICENSE` stays verbatim MIT text..." | keep-historical | D-13 KEEP class: "began as a reimplementation of" is a past-tense origin statement, exactly what D-13 requires to survive untouched. This is also the LOCKED, authoritative source Task 2's Licensing re-sync copies into `.claude/CLAUDE.md:17` — re-authoring it to be origin-neutral is rejected on separate grounds (it would re-open D-13). |
| `.planning/STATE.md` | 115, 188 | "`docs/CLI-REFERENCE.md` (DOCS-05) is deferred... This milestone *deletes* `docs/FLAG-PARITY.md` and its drift guard..." / "DOCS-05 — self-authored `docs/CLI-REFERENCE.md` with its own drift guard \| v2; this milestone deletes `docs/FLAG-PARITY.md`..." | keep-historical | Both hits are the literal filename `docs/FLAG-PARITY.md` (an actual document being deleted this milestone per CONTEXT.md D-02 of Phase 4/06's scoping), not standing parity framing. Deferral records citing a real artifact, not an assertion of current parity. |
| `.planning/STATE.md` | 170 | "GO-2026-5932 is a real, ACCEPTED, unmitigated exposure in release tooling.** goreleaser's binary reaches `golang.org/x/crypto/openpgp` (110 vulnerable symbols) via pipe/ko → google/ko → sigstore/cosign/oci → sigstore/rekor/pkg/pki/pgp. Upstream is unmaintained (Fixed in: N/A)..." | keep-historical | Instrument false-positive: "Upstream" here is the ordinary technical sense — `golang.org/x/crypto/openpgp` (an unrelated third-party dependency) being unmaintained — not project-origin framing about codegraph-go. Not applicable to D-12/D-13's test at all; recorded as keep rather than invented a fourth verdict class. |
| `.claude/CLAUDE.md` | 17 (post-edit) | "**Licensing**: MIT. Attribution for the project this one began as a reimplementation of lives in `NOTICE` — the single sanctioned place — and is discharged there, not restated across the repo. `LICENSE` stays verbatim MIT text (an appended attribution paragraph downgrades GitHub's license detection to `NOASSERTION`; observed, not assumed)" | keep-historical | **Added by Task 2, cycle-4 MEDIUM 1.** Introduced by the authorised re-sync from `PROJECT.md:246`; past-tense origin statement — D-13 KEEP class, not present-tense framing. This is the one term this task's Licensing re-sync *introduces* into `.claude/CLAUDE.md` (`reimplementation`, replacing the file's prior bare `port` at the same line); it is adjudicated here on the record rather than admitted as a side effect. The `sweep` row already recorded for line 17 above stays exactly as written — that row accounts for what the edit REMOVED ("Original is MIT — port with attribution"), this row records what it PUT THERE. `\breimplementation\b` and bare `\bport\b` stay deliberately outside `CLAUDE_MD_FRAMING_TOTAL`'s pattern set (line 83's parser-runtime sense of `reimplementation` already makes a whole-file zero gate over it unreachable, and re-authoring `PROJECT.md:246` to avoid the term would re-open D-13, which is LOCKED). |

**`SWEEP_ROWS_TOTAL`:** 17 rows total. **`SWEEP_VERDICTS`:** 10 — `PROJECT.md:5`, `CLAUDE.md:7`,
`CLAUDE.md:15`, `CLAUDE.md:17`, `PROJECT.md:244`, `PROJECT.md:219`, `PROJECT.md:220`,
`PROJECT.md:234`, `STATE.md:26`, `CLAUDE.md:93`. **`KEEP_VERDICTS`:** 6 (the
`PROJECT.md:31,40,...` range row, `PROJECT.md:232`, `PROJECT.md:246`, `STATE.md:115,188`,
`STATE.md:170`, and the new `CLAUDE.md:17 (post-edit)` row above). **`DIVERGENT_VERDICTS`:** 1 (the
`CLAUDE.md:21-141` range row, which also accounts for line 83).

Every occurrence named in the plan's pre-identified table appears above as a row. The one
occurrence the plan's table names that the word-census instrument does not surface
(`PROJECT.md:244`) is recorded in the "instrument gap" note above with the pattern that misses it
and why a separate check (Task 2's `COMPATIBILITY_BULLET_STALE_CAPABILITY_REFS`) is the correct
verification mechanism for that specific defect class.

Do NOT edit any of the three files in this task — this document is the decision record Task 2
executes.

## Engram enumeration (D-14) — HALTED: precondition not met (06-06 Task 1)

**Precondition named by 06-06-PLAN.md Task 1, verbatim:** "The executing session has engram MCP
tools registered and callable — a read-only listing call against the repo spine scope returns a
result rather than a tool-not-found error. This is the real precondition, NOT network reachability
of the gateway host: engram is configured at user level and not in this repository's `.mcp.json`,
so a session on a reachable network with no registered tools still cannot enumerate anything. If
the tools are unavailable, HALT and surface a checkpoint. Do not substitute recall context, do not
infer the population, and do not proceed to the checkpoint."

**Check performed live, in this executing session, before any enumeration attempt:**

1. This executor's own registered/callable tool surface was inspected directly (the complete
   function list made available to this session for invocation). It contains exactly: `Read`,
   `Write`, `Edit`, `Bash`, `Skill`, `mcp__context7__resolve-library-id`,
   `mcp__context7__query-docs`, `mcp__plugin_context7_context7__resolve-library-id`,
   `mcp__plugin_context7_context7__query-docs`. **No `mcp__engram__*` tool of any kind is present**
   — not `list_memory`, not `search_memory`, not `get_memory`, not `list_rules`, not
   `supersede_memory`.
2. `${PROJECT_ROOT}/.mcp.json` was read in full this session (both the shared-checkout path and
   this worktree's own copy, identical contents). It registers only two servers, `gsd-workflow`
   and `gsd-browser`. No `engram` entry exists at the project scope — confirming RESEARCH.md
   Pattern 6's LOW-confidence note that engram is not wired into this repo's own MCP config.
3. `~/.claude/plugins/installed_plugins.json` was read this session and confirms `engram@engram`
   IS installed, but at **user scope** (v0.13.0) — not project scope, and not passed through to
   this worktree-spawned executor's tool surface. This is exactly the scenario the precondition
   text names: a session can have a reachable engram gateway host and still have zero registered
   tools to call against it, because tool registration and network reachability are different
   preconditions, and this task's precondition is the former.
4. **No `mcp__engram__*` call was attempted.** There is nothing named `mcp__engram__list_memory`
   (or any sibling) in this session's callable surface to invoke — there is no tool-not-found
   *error* to observe, because the tool does not exist in the surface to be called in the first
   place. This is the unavailable case the precondition anticipates, not a distinct failure mode.

**Result: PRECONDITION UNMET.** Per Task 1's own instruction, this task HALTS here. No
enumeration, no classification, and no inference from recall context was performed for any of the
three D-14 scopes (repo spine, `rule:repo:` scope, workspace overlays). Zero engram tool calls —
read or write — were made in this session, before or after this check.

**Disposition:** This plan (06-06) is NOT marked complete. Task 2's blocking
`checkpoint:decision` (classification approval) is not reached, because Task 1 produced no
classification for it to approve — proceeding to it would mean approving an empty or fabricated
population, which the precondition text explicitly forbids ("do not proceed to the checkpoint").

**What resolves this:** a subsequent execution of this task — either a fresh interactive session
that carries the `mcp__engram__*` tool surface (the orchestrator's own session, per the
maintainer-supplied context in this run's spawn prompt, which references specific engram
`short_id`s it evidently can recall/read), or a re-spawned executor whose tool grant explicitly
includes the `engram` MCP server. Until then, D-16 requires this sweep be recorded as blocked,
not complete.

---

## Engram enumeration (D-14) — RESUMED AND COMPLETED by the orchestrator session

The halt above stands as an accurate record of the executor's session. The orchestrator session
DOES carry the `mcp__engram__*` tool surface, so it performed Task 1's enumeration directly, per
the first of the two resolutions the halt names.

### Population (enumerated to exhaustion, not inferred)

Scope `repo:github.com/seanb4t/codegraph-go`, paged with `list_memory` until `next_cursor` came
back empty — 5 pages, 4×40 + 1×9.

| Measure | Value |
|---|---|
| Total records | **169** |
| Pages to exhaustion | 5 (`next_cursor` empty on page 5) |
| Date range | 2026-07-11 .. 2026-08-16 |
| By category | decision 69, gotcha 82, convention 7, preference 2 (of the 160 captured to file) |
| Records whose summary carries any parity/comparison/migrate framing | 32 |

Scope `rule:repo:github.com/seanb4t/codegraph-go`: 3 rules (`xmz3xknbj0`, `84d1gfpywd`,
`f18zrdsgx5`) — none carries retired framing; all three are process constraints (commit signing,
positive-assertion guards, `[ci skip]` prohibition).

### Classification criterion

Per the plan: a record that HISTORICALLY described a then-true state is not wrong and is left as
history. A record that asserts a CURRENTLY-FALSE fact **that a future session would act on** is
what needs superseding. Per-phase completion records (`SHIPPED`/`COMPLETE`/`PLANNED`/`EXECUTED`)
are history by construction and are left intact — rewriting them would falsify the project's own
past, the same objection that puts `.planning/` archives out of scope.

Correction is by **supersede only**. Nothing is deleted or overwritten; a superseded record stops
surfacing in recall but stays fetchable by id, preserving "what we used to believe and when it
changed."

### Verdicts — SUPERSEDE (4 candidates, pending maintainer approval)

| short_id | Date | The currently-false standing claim | Why it must not stand |
|---|---|---|---|
| `3ekc84hbqt` | 2026-08-09 | Maintainer ruling "parity is a functionality baseline, not a binding constraint", closing with **"The functionality floor and the migration path still bind."** | Both halves are now false. v0.11.0 retired the Compatibility constraint outright (`gw79qy2a9z`), and Phase 5 deleted the migration path entirely (`4j8kjck5gd`). The record is written as live guidance for future work ("Do NOT cite CLAUDE.md's parity line as a blocker without weighing this ruling") so a future session would act on it. |
| `gw79qy2a9z` | 2026-08-13 | v0.11.0 scoping record asserting `codegraph-migrate-capability-PRESERVED-reframed` and `migrate-one-way-sqlite-read-still-binds` | Reversed two days later by maintainer ruling D-04 (`4j8kjck5gd`), which dropped migrate entirely as the milestone's single deliberate exception. The rest of this record's scoping content is still accurate. |
| `agggksad53` | 2026-07-15 | v1.0 Phase-2 ruling: **"FilesByLanguage MUST be `json:\"-\"` (TS discards counts; emitting the key = NEW divergence)"** | Reversed by v0.11.0 Phase 1 D-03, which un-suppresses it — justified explicitly by the Compatibility constraint being retired. The claim's whole warrant was parity with the other implementation, and that warrant is gone. **Composite record**: the other ~95% (four inference-vs-source lessons, the CR-02 silent-skip hazard, the `rg -r`/`rg -h` gotchas) remains true and valuable. |
| `7f0pq2wepv` | 2026-07-15 | Same reversed suppression ruling, reached via the same parity reasoning (D-05/STAT-02) | Same as above. Also composite — its 4-gate worktree cascade correction and D-14 plumbing notes remain load-bearing. |

`agggksad53` and `7f0pq2wepv` were **named in advance** as targets of this sweep: `gwzjpq9xvq`
(v0.11.0 Phase 1 CONTEXT) carries the tags `CONTRADICTS-agggksad53`, `CONTRADICTS-7f0pq2wepv`,
`NOT-superseded-here-on-purpose`, `composite-records-other-content-still-true`, and
`FLAGGED-AS-NAMED-TARGET-FOR-MEM-01-PHASE-6-SWEEP`. Phase 1 deliberately deferred them here rather
than superseding them wholesale, precisely because a blanket supersede would drop still-true
content out of recall.

### Verdicts — LEAVE-HISTORICAL (representative; the criterion applies to all 165 others)

| short_id | Why it stays |
|---|---|
| `e2ff222j6b`, `4khd1zp4bt`, `nqwgt6r53a`, `ggjepdd4ds`, `7grs09xyxz` | v0.11.0 per-phase completion records. History by construction. |
| `swzxf5tr2x`, `jqwc8eqjt3` | Phase-7 migration-tool SHIPPED / CONTEXT records. True as history: migrate *was* built and *was* shipped. Their subject was later deleted, which `4j8kjck5gd` records; that is the correction, not a reason to rewrite the past. |
| `76t84ynav5` | Milestone v1.0 "Drop-in Parity & Human UX" started. A real milestone name; renaming history would falsify it. |
| `myywc0y9vm`, `4j8kjck5gd` | Currently correct — these *are* the corrections this milestone rests on. |
| `veqtqg9r6r` | Already self-retracted in its own text. |
| `whad9x6gxq` | Flaky-test gotcha; updated in place this session to widen the set to `test/wireoracle` and record the confirmed `-p 1` remedy. Carries no framing. |
| `hakm8j156e`, `0x27xn3wvr`, `x2fgwgxf6k` | Method/instrument lessons (census blind spots, false absence, executor recovery). No framing claim. |
| `xmz3xknbj0`, `84d1gfpywd`, `f18zrdsgx5` | The three repo rules. Process constraints, no framing. |

### Status

`MEM01_ENUMERATION_STATUS=complete-169-records-5-pages`

Task 2's blocking `checkpoint:decision` is now reachable: the classification above exists and is
put to the maintainer for approval before any durable write.

### Task 3 (apply the approved corrections) — COMPLETE

The maintainer approved all 4 supersedes at Task 2's blocking checkpoint. All 4 are APPLIED.

| Superseded | Correcting record | The claim that was corrected |
|---|---|---|
| `3ekc84hbqt` | `xj1stbrsw6` | "The functionality floor and the migration path still bind" — both halves false |
| `gw79qy2a9z` | `gxwkk3necn` | migrate "PRESERVED-reframed" / "one-way sqlite read still binds" — reversed by D-04 |
| `agggksad53` | `b9wjge7375` | `FilesByLanguage` MUST be `json:"-"` — reversed by v0.11.0 Phase 1 D-03 |
| `7f0pq2wepv` | `mw5z9s9bft` | same suppression ruling, reached via the same parity reasoning |

**Method.** Correction by supersede only — no `delete_memory`, no `update_memory`, no overwrite of
any target, satisfying T-06-18's mitigation. Each correcting record restates the superseded
record's still-true content verbatim before correcting the one reversed claim, so the composite
records (`agggksad53`, `7f0pq2wepv`) lose nothing from recall: the four-errors-from-inference
lesson, the CR-02 silent-skip trap, the 4-gate worktree cascade with its inverted gate 4, the
`OpenAt` startPath plumbing, and the `rg -r`/`rg -h` traps all carry forward.

**Verified two ways, not asserted:**
1. `get_memory("3ekc84hbqt")` returns the original content intact with
   `superseded_by: fc4e8512-70a3-4af9-9bcc-5368eb09cd18` stamped — history preserved, nothing
   deleted or overwritten.
2. A `search_memory` probe phrased in the superseded records' own terms ("parity is a functionality
   baseline"; "FilesByLanguage must be json dash suppressed"; "migrate capability preserved and
   reframed") returns all four CORRECTING records and none of the four superseded originals — the
   property that actually matters, since recall is what a future session acts on.

**A recorded false alarm, kept deliberately.** The first four supersede attempts failed with
`not found: ["<id>"]` and were misdiagnosed in this artifact as an ownership write-gate denial,
with the session wrongly believed to be authenticated as a non-owner. That was wrong. The real
cause: `supersede_memory`'s `supersedes` parameter is a **string**, and an array was passed. The
error echoed the array back verbatim — brackets and quotes included — which was the server
correctly reporting that no record has the id `["3ekc84hbqt"]`. The maintainer corrected the
identity claim, and reading the tool's actual schema found the real cause in one step. The lesson
generalizes and is worth more than the incident: **a "not found" that echoes your argument back is
reporting your argument, not the store's contents** — and a tool called without reading its schema
is a guess, however plausible the resulting error story.

**Status token:**

`MEM01_STORE_STATUS=complete-4-superseded-0-deleted-0-overwritten`
