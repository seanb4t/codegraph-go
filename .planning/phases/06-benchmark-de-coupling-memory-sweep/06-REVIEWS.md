---
phase: 6
reviewers: [codex]
reviewed_at: 2026-08-16T12:55:14Z
review_cycle: 3
plans_reviewed: [06-01-PLAN.md, 06-02-PLAN.md, 06-03-PLAN.md, 06-04-PLAN.md, 06-05-PLAN.md, 06-06-PLAN.md]
---

# Cross-AI Plan Review — Phase 6

> This file holds **three** convergence cycles. Cycle 3 (immediately below) assesses the CURRENT
> plan text as revised by commit `fd02272`, and is the **final automated cycle** before the
> escalation gate. Cycles 2 and 1 are retained further down as audit history — their findings are
> **not** current; consult them only to trace how a cycle-3 statement came to be.

---

# Cycle 3 — FINAL automated cycle, assessment of the revised plans (`fd02272`)

**Reviewer:** Codex (source-grounded, full repo access), cross-checked and extended by the
orchestrating reviewer against the working tree at `fd02272`.

## Consensus Summary — Cycle 3

The cycle-2 replan did what it claimed. **All five claims the replan makes were independently
verified against source and plan text, and all five hold.** The `TOLNAMED` gate that cycle 2 flagged
as unsatisfiable is now asserted on `want error` plus the Go-mangled band-naming subtest names, and
both are text the mutated path genuinely emits; `CATERR=0` survives a full verbatim `-v` paste
because Go renders the category subtests' names with underscores while the negative grep searches
space-separated phrases. The self-contradictory pre-plan-SHA guard is replaced by two properties
that are mechanically true given the task ordering. The `.claude/CLAUDE.md` sweep is bounded by
three criteria whose numbers all check out against the file as it stands today. The planner's own
class fix C is real and completely applied — all nine sites now count with `wc -l`, and no new
instance of the broken form was introduced. The three rejected findings are each legitimate and
each recorded in a PLAN.md rather than only in a commit message.

**Two new HIGHs survive, and both are the same defect class in new places — an assertion the correct
path cannot satisfy.**

- **HIGH 1 — 06-01.** Task 1 asserts an exact before/after `go test -list` delta over
  `./tools/bench/...` whose allowed *added* set contains exactly three names, while the same task
  creates the `tools/bench/publishcheck` package with its own test file, necessarily contributing at
  least one more top-level `Test*` name to that package pattern.
- **HIGH 2 — 06-03 and 06-04.** Both censuses assert a total of `0` over a scope that includes
  `corpora/manifest.json`, whose line 10 carries `TS/JS` inside a `LOCKED (01-06)` corpus-language
  note. That file is in no plan's `files_modified`, the text is the plans' own declared KEEP class,
  and the two escapes an executor has — reword it, or add an exclusion — are both explicitly
  prohibited. The sanctioned escape, "bound the pattern", is unavailable because the pattern is
  hard-coded inside two `<automated>` blocks.

Neither is a style defect. 06-01 and 06-03 are both `wave: 1`, and `06-02 → 06-04 → 06-05 → 06-06`
chain off them, so between the two findings every plan in the phase is affected.

The lesson this phase earned applies to both verbatim: **a positive assertion the correct path
cannot satisfy is worse than no assertion at all**, because it fails on correct work and trains the
executor to treat red as expected. Cycle 1's class fix B produced class fix C by exactly this
mechanism; these two are the same shape one layer up — an exact *set*, and an over-broad *pattern*,
rather than an exact *count*.

Both fixes are small and local, and neither requires a locked decision to be re-opened.

### Agreed Strengths

- **The `TOLNAMED` rewrite is correct, not merely reworded.** `internal/bench/regression.go:113-126`
  is unreachable once either tolerance is widened to `1.0`, so the plan's new prohibition
  (`06-03-PLAN.md:49`) forbidding a criterion that demands `throughput regressed` / `peak RSS grew`
  is exactly right, and the replacement assertion at `06-03-PLAN.md:448` names only
  `want error` (emitted at `internal/bench/regression_test.go:511`) and the two band subtest names.
- **The band names are reachable twice per family, by construction.** Task 2 pins the new test's
  subtests as `committed baseline throughput 11% slower: exceeds band fails` and
  `committed baseline peak RSS 16% larger: exceeds band fails` (`06-03-PLAN.md:286-290`), whose
  Go-mangled forms *contain* the `throughput_11%_slower` / `peak_RSS_16%_larger` substrings the gate
  greps for. The planner's dry-run figures (`WANT_ERROR_ASSERTIONS=4 THROUGHPUT_BAND_REDS=2
  RSS_BAND_REDS=2`) fall out of this arithmetic exactly.
- **The CATERR prose trap is closed explicitly.** `06-03-PLAN.md:429-432` instructs the executor not
  to quote the three category phrases in the log's own prose, because the gate counts them over the
  whole file — the one way a correct rehearsal could still have tripped it.
- **Class fix C was applied beyond the nine named sites, as a reading discipline.** 06-04, 06-05 and
  06-06 each carry an explicit "checked, not assumed" row (`06-04-PLAN.md:143`, `06-05-PLAN.md:113`,
  `06-06-PLAN.md:123`) recording that their only surviving `rg -c` uses are `= "1"` comparisons
  backed by a real match. That is the right way to record a negative.
- **The three rejections are recorded where the executor will read them**, not only in git history:
  `06-03-PLAN.md:144`, `06-05-PLAN.md:112`, `06-02-PLAN.md:283`.

### Agreed Concerns

Two concerns rise to HIGH — 06-01's exact test-surface delta, and the 06-03/06-04 census zero — both
stated in full below. Four non-HIGH findings follow them (two MEDIUM, two LOW), each with the
concrete PLAN.md change it needs.

### Divergent Views

Codex and the orchestrating reviewer agreed on HIGH 1 and on all five claim verdicts. HIGH 2 and the
four non-HIGH findings came from the orchestrating reviewer's own measurements and were not in
Codex's output; none contradicts anything Codex reported.

**One cycle-2 statement is refuted, not merely superseded.** The cycle-2 coverage table below records
*"Phase-6 census is reachable | Ran the full cycle-2 pattern set → 168 hits today, every one inside a
declared sweep target."* Re-measured file-by-file this cycle, that is wrong for exactly one file:
`corpora/manifest.json` holds one hit and is a declared sweep target of no plan. HIGH 2 is the
consequence. The cycle-2 aggregate count was right; the "every one" claim was not checked
per-file, and the aggregate hid it.

Codex's line citations for `internal/bench/regression_test.go` and `regression.go` were each off by
roughly two lines (`:509`/`:112`/`:120` against measured `:511`/`:113`/`:121`); the substance was
correct and the measured numbers are used throughout this cycle.

---

## HIGH 1 (NEW) — 06-01's exact test-surface delta is unsatisfiable by its own Task 1

**Where.** `06-01-PLAN.md:272` (before-snapshot), `06-01-PLAN.md:380-381` (the delta assertion),
`06-01-PLAN.md:470` (the same statement in `<verification>`), `06-01-PLAN.md:86-93` (artifact
register).

**The contradiction, measured.**

1. Task 1's action opens with *"Before touching a single file"* and captures
   `go test -list '.*' ./tools/bench/... | rg '^Test' | sort` (`06-01-PLAN.md:263,272`). Measured
   against the tree at `fd02272`, that before-set is **55 names** across four packages
   (`go list ./tools/bench/...` → `cpudiag`, `gencorpus`, `realcorpus`, `runner`). It contains all
   three names the plan expects to remove — `TestParseFlags_OverridesApply`,
   `TestResolveTSBinary_EmptyWhenNotFound`, `TestResolveTSBinary_FindsOnPath` — so the *removed*
   half of the assertion is sound.
2. The **same task** creates `tools/bench/publishcheck/main.go` and
   `tools/bench/publishcheck/main_test.go` (`06-01-PLAN.md:12-13`, `files_modified`;
   `06-01-PLAN.md:241-254` specifies seven behaviours for that test file).
   `tools/bench/publishcheck` is inside the `./tools/bench/...` package pattern, so its top-level
   test names appear in the **after**-list at `06-01-PLAN.md:380`.
3. `06-01-PLAN.md:371` independently requires `go test -run TestPublishCheck -v
   ./tools/bench/publishcheck/...` to name the rejecting cases as RUN/PASS — which is only possible
   if at least one top-level test whose name matches `TestPublishCheck` exists.
4. But `06-01-PLAN.md:381` states: *"the added set MUST equal exactly `TestCorporaHasExactlyTwoEntries`,
   `TestParseFlags_PublishOverridesApply`, `TestParseFlags_RejectsRetiredComparisonBinaryFlag`"* and
   requires the executor to *"verify with `diff` against those two literal sorted lists."*

A correct execution therefore produces an added set of **at least four** names and the `diff`
against the three-name literal list is non-empty. The criterion fails on correct work.

**Why HIGH rather than MEDIUM.** 06-01 is `wave: 1` with `depends_on: []`; 06-02 (`wave: 2`), 06-04
(`wave: 3`), 06-05 (`wave: 4`) and 06-06 (`wave: 5`) chain off it. A gate that reddens on correct
work at the head of the chain halts five of six plans, and the failure looks like a real
regression — the executor's most likely response is to hunt for a test that "vanished for another
reason" that does not exist, or to weaken the assertion ad hoc. That is precisely the outcome the
phase's own lesson forbids.

**The fix (small, entirely inside 06-01).**

1. `06-01-PLAN.md:241` — name the publishcheck test symbol(s) explicitly in the `<behavior>` block.
   Prefer **one** top-level `TestPublishCheck` carrying the seven cases as subtests: that adds
   exactly one stable name to the delta while `-run TestPublishCheck -v` still names each case.
2. `06-01-PLAN.md:381` — extend the allowed added set to the four-name literal list
   (`TestCorporaHasExactlyTwoEntries`, `TestParseFlags_PublishOverridesApply`,
   `TestParseFlags_RejectsRetiredComparisonBinaryFlag`, `TestPublishCheck`).
3. `06-01-PLAN.md:470` — restate "three removals and three additions" as "three removals and four
   additions" so `<verification>` agrees with the criterion.
4. `06-01-PLAN.md:86-93` — add the concrete test symbol to the artifact register, which today lists
   the publishcheck *files* but not the test symbol the delta assertion turns on.

An acceptable alternative is to scope both snapshots to the packages that existed before the task
(`./tools/bench/realcorpus/... ./tools/bench/runner/... ./tools/bench/cpudiag/...
./tools/bench/gencorpus/...`), leaving the new package's surface to `06-01-PLAN.md:371`. The
four-name list is preferable: it keeps the assertion over the whole `tools/bench` surface, which is
what the criterion exists to police.

---

## HIGH 2 (NEW) — the retired-term census cannot reach zero in either 06-03 or 06-04: `corpora/manifest.json:10` is matched by `\bTS\b` and is in no plan's `files_modified`

**Where.** `06-03-PLAN.md:224` (`BENCH_PKG_RETIRED_TERMS_TOTAL = 0`), `06-04-PLAN.md:494` and the
matching criterion at `06-04-PLAN.md:501` (`PHASE6_CENSUS_TOTAL = 0`).

**Measured.** Both censuses scope `corpora/manifest.json` and both include a bare `\bTS\b`
alternation (06-04 additionally with `-i`). Running each pattern set verbatim against the tree at
`fd02272`:

```
06-03 census over its full scope  (internal/bench/ internal/corpora/manifest.go corpora/manifest.json) → 7
06-03 census over corpora/manifest.json alone                                                          → 1   (line 10)
06-04 census over corpora/manifest.json alone, case-insensitive                                        → 1   (line 10)
```

The single hit is the substring `TS` inside `TS/JS` at `corpora/manifest.json:10`:

> `"note": "LOCKED (01-06): closes the Go priority-4 coverage gap and supplies TS/JS via its 25
> javascript tracked files, which is how this minimum-cardinality set covers all five priority
> languages. Roadmap shortlist."`

**Why a correct execution cannot clear it.**

1. `corpora/manifest.json` appears in **no** plan's `files_modified` — checked across all six.
   06-03's list carries `internal/corpora/manifest.go` (the Go file) but not the JSON.
2. 06-03's declared reading and editing scope for that file is line **2**, the *top-level* `note`
   (`06-03-PLAN.md:166`, `:220-221`). Line 10 is a *per-entry* note and is outside it. Line 2 is
   already clean for every 06-03 pattern — measured.
3. The flagged text is the plan's own **KEEP class**. `06-04-PLAN.md:448-449` already excludes
   `internal/indexer/**` and `testdata/golden/**` as "TypeScript-the-indexed-language product
   surface … Phase 5's recorded KEEP class". `TS/JS` in a corpus-language-coverage note is that
   class exactly — it is not retired comparison framing.
4. Both escapes are closed to the executor. `06-04-PLAN.md:45` prohibits rewording innocent
   third-party prose *or* widening an exclusion to make the total read zero; `06-04-PLAN.md:451-452`
   forbids adding an exclusion not on the list. The sanctioned remedy — `06-04-PLAN.md:454-455`,
   *"If a pattern flags text that is genuinely innocent, the finding is about the pattern. Bound it
   tighter"* — is unavailable at execution time, because the pattern is **hard-coded inside the
   `<automated>` blocks** of two different plans, one of which (06-03) is `wave: 1` and
   `autonomous: true`.

So the executor of 06-03 hits a terminal gate that reads `1`, cannot legitimately reach `0`, and has
no in-plan instruction covering the case. 06-04 then hits the same wall with a wider pattern set.

**This also refutes a cycle-2 verification.** The cycle-2 coverage table recorded *"Phase-6 census is
reachable | Ran the full cycle-2 pattern set → 168 hits today, every one inside a declared sweep
target."* That is wrong for exactly one file: `corpora/manifest.json` is not a declared sweep target
of any plan, and it holds one of those hits.

**The fix (in PLAN.md, before execution — it cannot be done by the executor).**

1. `06-03-PLAN.md:224` and `06-04-PLAN.md:494` — bound `\bTS\b` to the retired-framing senses the
   sweep is actually about, e.g. `\bTS CodeGraph\b`, `\bGo-vs-TS\b`, `\bts-binary\b`, `\bTS binary\b`,
   dropping the bare alternation. 06-03 already carries `\bGo-vs-TS\b` as a separate pattern, so its
   bare `\bTS\b` is largely redundant; 06-04's `-i` additionally makes it match any lowercase `ts`
   identifier that ever appears on the census surface, which is a standing false-positive generator.
2. `06-04-PLAN.md:445-452` — record the bounding in `06-CENSUS.md` with the flagged text quoted, as
   `06-04-PLAN.md:454-455` requires. That is the plan's own prescribed handling; it just has to be
   pre-authorised rather than improvised.
3. `06-03-PLAN.md:7-15` — while here, add `corpora/manifest.json` to `files_modified`, or state
   explicitly that it is read-only to this plan. `06-03-PLAN.md:220-221` already contemplates editing
   it (*"If `corpora/manifest.json`'s `note` needs a change, say so in the SUMMARY: it is committed
   data, so the edit ships in the repo"*) while the frontmatter does not declare it — an executor
   following the action would write a file the plan says it does not modify.

---

## Non-HIGH findings

### MEDIUM 1 — 06-06's approved-set extraction is still file-wide while the executed-set extraction is section-bounded

Cycle 2's MEDIUM 4 replaced the executed-set extractor with a section boundary
(`awk '/^## Supersedes executed/{f=1;next} /^## /{f=0} f'`, `06-06-PLAN.md:321`), which is correct.
The **approved**-set extractor on the same line was left file-wide:
`rg -N '^\|' "$F" | rg -F '| supersede |'`.

`06-06-PLAN.md:196` fixes the enumeration table's columns as
`short_id | scope | durable content | verdict | reason`, so `| supersede |` is a verdict literal —
but the extractor scans the whole document, including `## Supersedes executed (MEM-01)` (columns at
`06-06-PLAN.md:292-295`) and `## Completeness and accepted gaps`. If any executed row or completeness
row renders a bare `| supersede |` cell, that row's first-column `short_id` joins the approved set as
a duplicate, `APPROVED_SUPERSEDE_IDS != APPROVED_UNIQUE`, and the task fails on correct work. The
risk is modest — the executed table's third column is specified as "the operation name called", which
in practice is `supersede_memory`, not `supersede` — but this is the identical failure mode the
section-boundary fix was introduced to remove, left applied to only one of the two sets.

**PLAN.md change needed:** at `06-06-PLAN.md:321`, bound the approved-set extraction to its own
section with the same technique — `awk '/^## Engram enumeration/{f=1;next} /^## /{f=0} f' "$F"` piped
into the existing `rg -F '| supersede |'`. (`^## ` does not match the `### ` per-scope subheadings, so
the scope sections stay inside the window.) Record the extractor's raw line count in the SUMMARY
alongside the executed set's, as `06-06-PLAN.md`'s own criterion already requires for the executed
half.

### MEDIUM 2 — two of the four counters in 06-03's replacement gate pass vacuously on a GREEN run (measured)

`06-03-PLAN.md:448` asserts four numbers over `06-MUTATION-LOG.md`: `CATERR = 0`,
`WANTERR >= 2`, `THROUGHPUT_BAND_REDS >= 2`, `RSS_BAND_REDS >= 2`. The rehearsal command at
`06-03-PLAN.md:376` uses `-v`, and `-v` prints `=== RUN` and `--- PASS` lines for **every** subtest,
not only failing ones.

Ran the gate's own four counters against a verbatim `-v` paste of the **unmutated** tree at `fd02272`
(`go test -count=1 -v -run 'TestCheckRegression' ./internal/bench/...`):

```
CATERR=0   THRU=2   RSSN=2   WANTERR=0
```

So `THROUGHPUT_BAND_REDS` and `RSS_BAND_REDS` are **already satisfied by a green run** — they measure
that the subtests exist, not that either one went red. Only `WANTERR` discriminates: it is `0` on
green and becomes `>= 2` only on the mutated path. The reachability note at `06-03-PLAN.md:449`
("Family A's pasted `-v` output carries `throughput_11%_slower` on the table subtest's `=== RUN` and
`--- FAIL` lines … so `>= 2` has headroom") is therefore true but describes the counters as evidence
of a RED when they are not.

The gate as a whole is **not** vacuous — `WANTERR` carries it, and `RED_BLOCKS_RECORDED >= 2`
(`06-03-PLAN.md:439`) plus `COMMITTED_BASELINE_REDS_RECORDED` (`06-03-PLAN.md:446`) enforce
two-family coverage — which is why this is MEDIUM and not HIGH. But rule `84d1gfpywd` is precisely
about a positive assertion that can pass without the thing it names having happened, and two of these
four can.

**PLAN.md change needed:** at `06-03-PLAN.md:448`, tie the band counters to the RED rather than to the
paste — count the band name only on `--- FAIL` lines, e.g.
`THRU=$(rg -N -- '--- FAIL' "$LOG" | rg -o -F 'throughput_11%_slower' | wc -l | tr -d ' ')` and the
same for `peak_RSS_16%_larger`, keeping `>= 2` (table + committed-baseline subtest per family). Then
amend the reachability note at `06-03-PLAN.md:449` to record the measured green-run values above as
the reason the counters were narrowed.

### LOW 1 — 06-05's `CLAUDE_MD_CHANGED_LINES <= 24` ceiling is tighter than the task's own scope statement

`06-05-PLAN.md:290-292` expects "roughly four changed lines" (7, 15, 17, 93) — eight diff lines,
comfortably inside the 6..24 window, and the floor of 6 is exactly reachable since at least three
distinct lines must change (measured: framing matches live at `.claude/CLAUDE.md:7`, `:15`, `:93`).
But the same action also says "Re-sync the generated PROJECT block from the corrected PROJECT.md"
(`06-05-PLAN.md:287`), and 06-05 edits `.planning/PROJECT.md` in the same task. The generated block
spans `.claude/CLAUDE.md:1-19`; if the PROJECT.md corrections touch the Core Value paragraph (line 9)
and more than one further constraint bullet, the re-sync alone can push past twelve changed lines and
trip the ceiling on correct work.

**PLAN.md change needed:** at `06-05-PLAN.md:290`, state the ceiling's basis — that 24 diff lines
allows up to twelve changed lines, i.e. the four expected plus eight of slack for the PROJECT-block
re-sync — and add a one-line instruction that if the re-sync legitimately exceeds it, the executor
records the count and the reason rather than trimming the re-sync to fit the number.

### LOW 2 — two of 06-06 Task 1's gates require literal phrases its `<action>` never asks for

`06-06-PLAN.md:203` asserts `PAGES=$(rg -o -i 'pages enumerated' …); test "$PAGES" -ge 3` and
`SCOPES=$(rg -o -i '^### .*(spine|rule:repo|overlay)' …)`. Both are counted over
`06-MEMORY-SWEEP.md`, a file this same task authors — so a correct execution *can* satisfy them, but
only by guessing the wording. The `<action>` asks the executor to "record the page count" and to
write "one subsection per scope"; it never states that the literal phrase **"pages enumerated"** must
appear three times, nor that the `###` headings must contain `spine` / `rule:repo` / `overlay`.

This is the mildest form of the family this cycle is about: not unsatisfiable, but satisfiable only
by accident. It is LOW because the whole file is executor-authored and a re-read of the criterion
resolves it.

**PLAN.md change needed:** in `06-06-PLAN.md`'s Task 1 `<action>`, state the two required literals
directly — the per-scope subsection heading form (`### <scope> — spine` / `rule:repo` / `overlay`)
and the phrase "pages enumerated: <n>" once per scope — so the executor writes them deliberately
rather than reverse-engineering them from the gate.

---

## Risk Assessment — Cycle 3

**MEDIUM until HIGH 1 and HIGH 2 are corrected; LOW afterward.**

The verification architecture is now sound in design: every load-bearing gate in the six plans was
traced to the file and line it asserts over, and with the two exceptions above each one both fails on
a mutated tree and passes on a correct one. What remains is not architectural — it is two gates whose
constants were not re-measured against the tree they will run on.

Both HIGHs are narrow and fixable entirely inside PLAN.md: four edits in `06-01-PLAN.md` for HIGH 1,
and one pattern-bounding edit in each of `06-03-PLAN.md:224` and `06-04-PLAN.md:494` (plus a
`files_modified` correction) for HIGH 2. Neither requires new work, new tasks, or re-planning. But
both sit on `wave: 1` plans, so both must be fixed *before* execution rather than caught during it —
and neither is something the executor is permitted to fix for itself.

The 16 locked decisions D-01..D-16 remain respected. **No finding in this cycle requires re-opening a
locked decision**, and no finding is an escalation about one. In particular HIGH 2 is a pattern
defect, not a challenge to D-08 or to the Phase-5 KEEP-class adjudication the plans correctly cite —
it is the plans' own doctrine (`06-04-PLAN.md:454-455`, "the finding is about the pattern") applied
one file further than the plans applied it themselves.

---

## Verification coverage — Cycle 3

Codex ran with full repo access and produced `file:line`-grounded findings throughout; no
`[reviewed-without-repo-access]` marker was present. The orchestrating reviewer independently
re-executed or re-read every load-bearing claim below against the working tree at `fd02272` before
accepting it. Where Codex's line numbers differed from measurement (it cited
`internal/bench/regression_test.go:509` and `regression.go:112/120`), the measured values are used.

| Claim | Source / command actually run | Verdict |
|---|---|---|
| Cycle-2 claim 1 — mutated path emits `want error`, not the tolerance strings | `internal/bench/regression.go:9,15,113-126`; `rg -n 'want error' internal/bench/regression_test.go` → **511** | **VERIFIED** — widening either constant to `1.0` makes 113-126 unreachable; the only emitted assertion is at `:511` |
| Cycle-2 claim 1 — band subtest names are Go-mangled substrings the gate can grep | `internal/bench/regression_test.go:43,62` → `throughput 11% slower: exceeds band fails`, `peak RSS 16% larger: exceeds band fails`; `06-03-PLAN.md:286-290` pins the committed-baseline analogues | **VERIFIED** — `throughput_11%_slower` / `peak_RSS_16%_larger` are substrings of both pairs |
| Cycle-2 claim 1 — the planner's dry-run figures are arithmetically consistent | 2 failing subtests per family × 2 families = `WANT_ERROR_ASSERTIONS=4`; table + committed-baseline per band = `THROUGHPUT_BAND_REDS=2`, `RSS_BAND_REDS=2` | **VERIFIED** — figures fall out of the specified test surface |
| Cycle-2 claim 1 — `CATERR=0` holds against a full `-v` paste | Ran the gate's four counters over a verbatim `-v` paste of the unmutated tree: `CATERR=0 THRU=2 RSSN=2 WANTERR=0`; underscore rendering confirmed (`=== RUN TestCheckRegression/runner_mismatch_between_baseline_and_current_fails…`); error strings at `regression.go:74,94` print only on failure | **VERIFIED** — and the same measurement surfaced MEDIUM 2: `THRU`/`RSSN` are already 2 on green |
| Tree is green at `fd02272` before any plan work | `go build ./...`, `go vet ./internal/bench/... ./tools/bench/...`, `go test -count=1 ./internal/bench/... ./tools/bench/...` | **CONFIRMED** — all green; the plans start from a clean baseline |
| Cycle-2 claim 2 — HEAD-relative property is true by construction | `06-03-PLAN.md:334-336`; Task 2's `files_modified` adds only `internal/bench/baseline_gate_test.go` | **VERIFIED** |
| Cycle-2 claim 2 — pre-plan-SHA property is comment-only-scoped | `06-03-PLAN.md:337-339` — `rg '^[+-][^+-]' \| rg -v '^[+-]\s*//' \| wc -l` over the four files | **VERIFIED** — no `git diff --quiet` "apart from" semantics remain |
| Cycle-2 claim 3 — 25 headings | `rg '^#{1,6} ' .claude/CLAUDE.md \| wc -l` → **25** | **VERIFIED** |
| Cycle-2 claim 3 — `>= 12` stack-block anchors | `rg -o -F -e 'The Parser Decision' -e 'cockroachdb/pebble' -e 'tree-sitter/go-tree-sitter' .claude/CLAUDE.md \| wc -l` → **16** | **VERIFIED** — floor holds with headroom |
| Cycle-2 claim 3 — GSD marker count | `rg -o '<!-- GSD:[^>]*-->' .claude/CLAUDE.md \| wc -l` → **14** | **VERIFIED** |
| Cycle-2 claim 3 — the 6..24 window is satisfiable | Framing matches live at `.claude/CLAUDE.md:7` (three on one line), `:15`, `:93`; plan also sweeps `:17`. 3 lines minimum → 6 diff lines; 4 expected → 8 | **VERIFIED at the floor** — ceiling caveat recorded as LOW 2 |
| Cycle-2 claim 4 — all nine class-C sites converted to `wc -l` | `PINDIFF` `06-01:376`; `PERMDIFF` `06-02:341`; `FIXDIFF` `06-02:399,407`; `FIGDIFF` `06-02:410`; `NC` `06-03:224`; `CONTENT`/`NONCOMMENT` `06-03:231,338`; `TOLDIFF` `06-03:234`; `CATERR` `06-03:448` | **VERIFIED** — 9/9 |
| Cycle-2 claim 4 — no NEW broken instance | `rg -n '\|\| true' 06-0*-PLAN.md` → 7 hits; every one is either prose describing the defect or a `= "1"` comparison against a real match | **VERIFIED** |
| Cycle-2 claim 4 — 06-04/05/06 genuinely needed none | `06-04-PLAN.md:143`, `06-05-PLAN.md:113`, `06-06-PLAN.md:123`; their only `rg -c` uses are the `NOTICE` copyright `= "1"` checks | **VERIFIED** |
| Class-C shape surviving at `06-03-PLAN.md:176` (baseline capture, not a gate) | Ran it verbatim → `metrics.go=51 regression.go=70 regression_test.go=70 rss.go=18` | **CONFIRMED SAFE** — all four non-zero, so the `±25%` comparison at `06-03:232` is well-defined; not raised as a finding |
| Cycle-2 claim 5a — mutation-alternative rejection recorded in a PLAN.md | `06-03-PLAN.md:144`, and the prohibition it produced at `06-03-PLAN.md:47` (forbids editing the test table) | **VERIFIED** — legitimate and recorded where the executor reads it |
| Cycle-2 claim 5b — `STACK.md` rejection is factually right | `rg -c -i 'parity' .planning/research/STACK.md` → **no match**; rejection recorded at `06-05-PLAN.md:112` | **VERIFIED** — Codex's original finding was false |
| Cycle-2 claim 5c — Node step line numbers | `.github/workflows/bench.yml` → `- name:` at **117**, `uses: actions/setup-node@…` at **118**, `node-version:` at **120**; recorded at `06-02-PLAN.md:283` as `117-120` | **VERIFIED** — the LOW claim of `118-121` was wrong |
| 06-02's `U_PINNED = 19` arithmetic still correct | `rg -o '^\s*uses: '` → 22, `'^\s*uses: \./'` → 2, `'uses: [^@[:space:]]+@[0-9a-f]{40}'` → **20**; `bench.yml:118` is the one pinned action removed, `bench.yml:123` is a `run:` step | **CONFIRMED** — 20 − 1 = 19, + 2 local = 21 total |
| 06-02's Taskfile exact-set + floor | `BENCHTASKS=bench:regression:,`, `ALLTASKS=47` (floor 40) | **CONFIRMED** |
| `\bcomparison\b` floors survive the sweep | `rg -c` → `tools/bench/BASELINE.md` **5** (floor 5, zero headroom — carried forward from cycle 2), `internal/bench/regression.go` **7** (floor 5) | **CONFIRMED** |
| 06-04's three `docs/BENCHMARKS.md` headings are reachable | `rg -n '^## ' docs/BENCHMARKS.md` → `## 1. Methodology` (7), `## 2. Raw numbers` (89), `## 3. Regression gate (PERF-02, INDX-06)` (242); Task 2 renames §2 to `## 2. Absolute numbers` | **CONFIRMED** — `H=3` attainable |
| **NEW HIGH — 06-01 added-set is exactly three** | `go test -list '.*' ./tools/bench/... \| rg '^Test' \| sort \| wc -l` → **55** across `cpudiag`/`gencorpus`/`realcorpus`/`runner`; Task 1 adds package `tools/bench/publishcheck` with `main_test.go` (`06-01-PLAN.md:12-13,241-254`) and requires `-run TestPublishCheck` to name cases (`06-01-PLAN.md:371`) | **REFUTED** — the added set must contain ≥ 4 names; `06-01-PLAN.md:381` and `:470` demand exactly 3 |
| 06-01's *removed* set is sound | All three of `TestParseFlags_OverridesApply`, `TestResolveTSBinary_EmptyWhenNotFound`, `TestResolveTSBinary_FindsOnPath` present in the measured before-set | **CONFIRMED** |
| No other exact-set assertion in 06-01/02/03 is unsatisfiable | `rg -n 'MUST equal exactly\|equal exactly' 06-01/02/03-PLAN.md` — only `06-01:381` and the Taskfile set, which measures true today | **CONFIRMED** |
| **NEW HIGH — census zero over `corpora/manifest.json`** | Ran both pattern sets verbatim: 06-03's over its full scope → **7**, over `corpora/manifest.json` alone → **1** at line 10; 06-04's case-insensitive set over the same file → **1** at line 10. `rg -n 'corpora/manifest.json'` across all six plans' `files_modified` → **absent from every one** | **REFUTED** — `06-03:224` and `06-04:494` assert `= "0"`; the residual hit is `TS/JS` in a `LOCKED (01-06)` note the plans' own KEEP class protects |
| Cycle-2's "every census hit is inside a declared sweep target" | Re-measured per-file rather than in aggregate | **NOT UPHELD** — true for every file except `corpora/manifest.json` |
| 06-03 declares the file it may edit | `06-03-PLAN.md:166,220-221` contemplate editing `corpora/manifest.json`'s top-level note; `06-03-PLAN.md:7-15` `files_modified` lists `internal/corpora/manifest.go` but not the JSON | **REFUTED** — declared-artifact mismatch, folded into HIGH 2's fix |
| 06-04's other exact-equality gates | `CENSUS_FILES_SCANNED` 26 (floor 12); `CENSUS_CRITICAL_PRESENT` **13/13**; positive control `POSCTRL_MULTILINE=5 POSCTRL_LINEBASED=3`; `## 3. Regression gate (PERF-02, INDX-06)` byte-exact at `docs/BENCHMARKS.md:242`; `EMITTED=2` matches the 3→2 corpus drop (`weft-go`, `cockroachdb-pebble` survive `tools/bench/realcorpus/manifest.go`) | **CONFIRMED** — all reachable |
| 06-05's exact-equality gates | `MK=14`, `HD=25`, `STACKANCH=16`, `PV_LEN=208`, `STATE_MD_RETIRED_CORE_VALUE` phrases confined to `STATE.md:26`, `CM=0` with the bullet retained at `PROJECT.md:244`, `state.load` parses | **CONFIRMED** — plan is clean |
| 06-05's re-sync does not break its own gate | The Licensing line re-synced from `PROJECT.md:246` imports "reimplementation", which is **not** in the `CLAUDE_MD_FRAMING_TOTAL` pattern set | **CONFIRMED** |
| 06-06 status-token namespaces do not collide | `MEM02_STORE_STATUS=` is not a substring of `MEM02_FILES_STATUS=` (06-05), and neither matches `BENCH03_STATUS=` (06-04); all three count over the shared `06-LIVE-VERIFICATIONS.md` | **CONFIRMED** — the three "exactly one token" counts stay independent |
| `NOTICE` copyright literal is byte-correct | `rg -c -F 'Copyright (c) 2026 Colby Mchenry' NOTICE` → **1**; the unusual "Mchenry" capitalisation is what the file contains and carries its own do-not-correct warning | **CONFIRMED** |

---

# Cycle 2 — audit history (superseded by Cycle 3 above)

**Reviewer:** Codex (source-grounded, full repo access), cross-checked and extended by the
orchestrating reviewer against the working tree at `6a2816b`.

## Consensus Summary — Cycle 2

The replan is a substantial, mostly successful convergence. All five cycle-1 HIGHs received real
structural answers rather than prose patches: the D-06 checklist became one publish-job-scoped Go
test, the committed `baseline.json` became a live test input whose own non-vacuity is demonstrated
by the same mutation that reddens the pre-existing table, and BENCH-03 was separated from BENCH-01
behind a blocking decision gate with a two-token ledger. The SHA-pin arithmetic is correct. The
three HEAD-relative guard exemptions are each justified. No sound `rg -c` floor was over-corrected,
and every floor the plans rely on is satisfiable against the current tree.

What remains is concentrated in **verification correctness rather than intent**: three criteria are
mechanically impossible or self-defeating as written, and several instruments are weaker than the
claims resting on them.

### Agreed Strengths

- **06-02's publish-job scoping is real.** Every D-06 assertion in `<behavior>` (06-02-PLAN.md:140-184)
  is anchored to the parsed `jobs.publish` subtree, including the upload step's OWN `with:` map. The
  untouched `rebless` upload at `.github/workflows/bench.yml:274` — verified present, carrying
  `if-no-files-found: error` — can no longer stand in for the publish job's guard. `key_links`
  (06-02-PLAN.md:41) states the property explicitly, so an executor cannot read it as optional.
- **06-03's HIGH 4 fix is genuine, not nominal.** `internal/bench/baseline_gate_test.go` reads
  `../../tools/bench/baseline.json` from disk (06-03-PLAN.md:236-240) — not a temp copy — and Task 3
  requires each mutation family to redden BOTH oracles (06-03-PLAN.md:329-343, 370). The committed
  file at `tools/bench/baseline.json` carries populated `goos`/`goarch`/`runner`/`scratch_fs`, so the
  four category guards in `internal/bench/regression.go:36-105` stay quiet for a frame-derived
  current record and the numeric bands really are the thing under test.
- **SHA-pin arithmetic is correct.** Measured on the working tree: `USES_TOTAL=22`, `USES_LOCAL=2`
  (`./.github/actions/install-task` at `bench.yml:330` and `:380`), `USES_PINNED=20`. The edit deletes
  exactly one pinned action (`actions/setup-node`, `bench.yml:118`), leaving 19 pinned + 2 local = 21.
  The latent false failure cycle 1 flagged is genuinely fixed.
- **The three HEAD-relative exemptions are each correct.** 06-05 Task 1 is the plan's first task, so
  HEAD and the pre-plan SHA are the same commit; 06-03's FIXT-07 gates assert working-tree cleanliness
  at an instant, which is what makes "the mutation applied" and "the revert was byte-clean" provable;
  06-02's fixture check constrains Task 2's own uncommitted diff. None is the blind-window bug wearing
  a rationale (one residual caveat at LOW-3 below).
- **Class fix B was applied without over-correction.** Floors and line-oriented counts were correctly
  left alone, and each is satisfiable today: `rg -c '\bcomparison\b' tools/bench/BASELINE.md` = 5 at
  lines 39/87/97/120/297 — **disjoint from the two lines 06-02 Task 2 sweeps (64 and 241)**, so the
  zero-headroom `>= 5` floor survives the sweep; `internal/bench/regression.go` = 7;
  `TASKFILE_BENCH_TARGETS` = `bench:regression:,` exactly; `ALLTASKS` = 47 (floor 40); `GSD_MARKERS`
  = 14; `SHAPETEST_FIXTURE_ENTRIES` = 10 (floor 10); both surviving corpus pins present.
- **06-05's core-value equality gate executes correctly.** Run against the current tree the extraction
  yields `PROJECT_CORE_VALUE_LEN=208` (floor 60) and a non-empty STATE side, and the Compatibility
  bullet at `.planning/PROJECT.md:244` matches `^\- \*\*Compatibility\*\*` and contains
  `codegraph migrate` exactly once — so the region-scoped criterion is both satisfiable and non-vacuous.
- **The D-15 rejection is legitimate and adequately recorded.** Requiring post-sweep re-query evidence
  would reopen a locked decision (`06-CONTEXT.md:166-172`). It is recorded three times over — in
  `<edge_probe_coverage>`, in `<review_incorporation>`'s "Explicitly NOT applied" block, and now
  mechanically as the `MEM02_STORE_STATUS=accepted-by-d15-evidence-standard` token. This is the right
  shape: the risk is named, not denied.

### Agreed Concerns

Three HIGHs are **new in cycle 2** — two of them introduced by the class-fix restatements themselves.

---

## HIGH 1 (CYCLE 2 — RESOLVED in `fd02272`, re-verified in cycle 3) — 06-03 Task 3's `TOLNAMED >= 2` gate cannot pass with the specified mutation

`06-03-PLAN.md:372` requires:

```
TOLNAMED=$(rg -o -e 'throughput regressed' -e 'peak RSS grew' 06-MUTATION-LOG.md | wc -l); test "$TOLNAMED" -ge 2
```

But the mutation widens `DefaultThroughputTolerance` / `DefaultRSSTolerance` to `1.0`
(`06-03-PLAN.md:325`, `:341`), which makes `CheckRegression` return **`nil`**. Those two strings live
only inside the error-construction branches at `internal/bench/regression.go:114` and `:124`, which
the mutation suppresses by design. The observed RED is
`CheckRegression() = nil, want error` — the assertion at `internal/bench/regression_test.go:512`.
No tolerance-naming line can appear in verbatim output.

The same defect is in the plan's prohibition at `06-03-PLAN.md:47` ("MUST NOT record a mutation family
as demonstrated RED unless the observed failure names the tolerance being violated"). Read literally,
the chosen mutation direction obliges the executor to **report a failed rehearsal**, or to hand-write
the phrases as prose — which is exactly the certifying-your-own-result failure the FIXT-07 protocol
exists to prevent.

This was introduced *by* class fix B: the pre-revision form was the `CATERR` half alone; the positive
companion is new and is mis-specified.

**Fix:** assert what the specified mutation actually emits — the two failing subtest names plus the
literal `CheckRegression() = nil, want error` — and keep `CATERR=0` as the wrong-RED disqualifier.
If the intent is genuinely to see the tolerance named, the mutation must move the *current* values
past the band instead of widening the constant, which is a different (and more invasive) rehearsal.

## HIGH 2 (CYCLE 2 — RESOLVED in `fd02272`, re-verified in cycle 3) — 06-03 Task 2 carries a self-contradictory pre-plan-SHA guard

`06-03-PLAN.md:296`:

> `git diff --quiet "$BASE" -- internal/bench/regression.go internal/bench/regression_test.go internal/bench/metrics.go internal/bench/rss.go` exits 0 **apart from Task 1's committed comment sweep**

Task 1 commits comment edits to exactly those four files (`06-03-PLAN.md:140`, `:166-192`). A plain
`git diff --quiet "$BASE"` has no "apart from" semantics; it will exit 1 every time. The criterion is
mechanically impossible as written, and an executor that runs it verbatim halts on a correct tree.

**Fix:** either scope it HEAD-relative (Task 2 adds a file and edits none, which HEAD compares exactly),
or keep the pre-plan anchor and filter the diff to comment-only hunks — the same shape Task 1 already
uses at `06-03-PLAN.md:212`.

## HIGH 3 (CYCLE 2 — RESOLVED in `fd02272`, re-verified in cycle 3) — 06-05's stack-block "re-sync" would replace ~119 lines of agent instructions and break the plan's own gate

`06-05-PLAN.md:161` gives the CLAUDE.md:93 occurrence this verdict rule:

> check whether this block still matches its named source (`.planning/research/STACK.md`). **If it is a
> stale mirror, re-sync**; if the phrase is live in the source, sweep the source and re-sync

Measured against the tree: **the phrase is NOT live in the source.** `rg -i parity
.planning/research/STACK.md` returns nothing. But the block is not merely stale — the source has been
**wholly replaced**. `.claude/CLAUDE.md` lines 21-141 mirror the original v1 tech-stack research
(tree-sitter, Pebble, mcp-go, the Parser Decision), while the current
`.planning/research/STACK.md` (198 lines) is the v0.11.0 *agent-onboarding skill/plugin + MCP Resources*
research — an entirely different document. A `diff` of the two shares essentially nothing beyond the
`### Core Technologies` heading.

Executing the stated rule therefore:

1. replaces ~119 lines of live agent instructions with unrelated content — far beyond a framing sweep,
   and in tension with this plan's own prohibition against removing headings in `.claude/CLAUDE.md`
   (`06-05-PLAN.md:44`);
2. **breaks the plan's own acceptance gate**: `.planning/research/STACK.md:151` contains `drop-in`,
   which is in the `CLAUDE_MD_FRAMING_TOTAL=0` pattern set (`06-05-PLAN.md:264`). The re-sync imports
   the very framing the census forbids.

Codex reported an adjacent MEDIUM here — that `.planning/research/STACK.md` belongs in
`files_modified` — which is a **false finding**: nothing in that file needs editing. The real defect is
that the re-sync branch is unexecutable.

**Fix:** give CLAUDE.md:93 an explicit verdict in the plan rather than a conditional. The minimal
correct action is to sweep the `parity` aside in `.claude/CLAUDE.md` in place, and record in
`06-MEMORY-SWEEP.md` that the stack block has diverged wholesale from its named source and that a full
regeneration is out of Phase 6's scope (a separate concern to raise, since the next regeneration will
overwrite the block regardless). Add a `keep-divergent` verdict class or a stated exception; do not
leave "re-sync" as the standing instruction.

---

### Actionable non-HIGH concerns

- **MEDIUM 1 — BENCH-03's `closed-by-ci-run` token is accepted with no URL or run validation.**
  06-04 Task 4's check (`06-04-PLAN.md:454`) counts the token and requires exactly one. The
  acceptance text names `<url>` (`:459`) but nothing asserts a non-empty URL, a matching workflow, a
  matching ref, or `conclusion == success`. A mis-selected closed option passes the machine check.
  *Plan change:* on the closed branch, require a non-empty `https://github.com/.../actions/runs/<id>`
  match and record `gh run view <id> --json conclusion,event,headBranch,url` output beneath the token.
- **MEDIUM 2 — `TestBenchReblessJobShape` does not cover rebless's run bodies.** The fixture
  (`06-02-PLAN.md:173-184`) covers `runs-on`, `env`, `if`, ordered step names, the upload `with:` map
  and the absence of `permissions:` — but not the three load-bearing `run:` bodies in that job
  (`bench.yml` ~194, ~220, ~276), which include the `-rebless` invocation the D-13/D-01 exception exists
  to contain. A body could change while the fixture stays green, so the `must_haves` claim "The `rebless`
  job's block ... unchanged by this edit" (`06-02-PLAN.md:29`) exceeds the instrument. The literal-fixture
  choice over a before/after hash is defensible; its *coverage* is not yet sufficient.
  *Plan change:* extend the fixture to normalised run bodies, or hash the canonical serialisation of the
  whole parsed `jobs.rebless` subtree as the fixture value.
- **MEDIUM 3 — 06-02's TDD RED does not discriminate publish-job scoping.** Task 1 is `tdd="true"` and
  requires the verbatim RED to be pasted (`06-02-PLAN.md:201-207`), but the RED is produced by
  `jobs.publish` being absent — a single missing-anchor failure that reddens every assertion at once.
  It is therefore no evidence at all about the property cycle-1 HIGH 1/2/3 turned on: whether each
  individual assertion is scoped to the publish job rather than reading the file globally. 06-03 holds
  itself to a stricter standard (a mutation rehearsal) for exactly this reason.
  *Plan change:* after GREEN, perturb one publish-job property in a scratch copy (e.g. delete the
  publish job's `if-no-files-found`, leaving rebless's intact) and record that the named assertion
  reddens — one discriminating observation, not a second protocol.
- **MEDIUM 4 — 06-06 Task 3's executed-set extraction uses a fixed `rg -A400` window**
  (`06-06-PLAN.md:312`). It fails in both directions: a long execution table is truncated (false set
  mismatch), and a short one spills into `## Completeness and accepted gaps`, whose rows can also match
  `^\| \`?[a-z0-9]{6,}\`? \|` (false extra ids). The set-equality check is the mechanism that closes
  cycle-1's `06-06:248`, so its extractor should not be the weak link.
  *Plan change:* bound the section at the next `^## ` heading (e.g. `awk '/^## Supersedes executed/{f=1;next}/^## /{f=0}f'`).
- **LOW 1 — the publish job's action check is a SET, so duplicates survive** (`06-02-PLAN.md:164`). An
  extra checkout or setup-go step passes. Not central to D-06; assert an ordered list or per-action counts.
- **LOW 2 — the `both` dispatch option is not in the identifier family.** `bench.yml:73` keeps `both`
  as a choice and the current job's `if:` at `:99` tests `inputs.job == 'both'`. 06-02's identifier list
  (`:244-247`) names only the map key, the `default:`/`options:` entry and the `if:` conditional, and the
  shape test asserts only "the scheduled-event term and the dispatch term naming this job id"
  (`:146-147`). Drop `|| inputs.job == 'both'` and the `both` option silently becomes rebless-only —
  the exact dead-dispatch-option failure `key_links` (`:38`) warns about.
  *Plan change:* add `both` to the identifier family and to the `if:` assertion.
- **LOW 3 — 06-02's HEAD-relative fixture guard has a residual blind window.** `FIXDIFF` +
  `FIXPRESENT >= 10` (`06-02-PLAN.md:321`) catches a *gutted* fixture but not a *modified* one if the
  executor commits Task 2's edit before running the gate. The exemption's rationale is sound; the
  companion is one-sided.
  *Plan change:* make the companion an exact entry-count equality against the pre-plan SHA extraction,
  or state that the gate must run pre-commit.
- **LOW 4 — `rg -c '^## ' docs/BENCHMARKS.md` cannot show which sections exist** (`06-04-PLAN.md:309`).
  A heading count is consistent with any three headings. Assert the three heading texts instead.
- **LOW 5 — line-reference drift in 06-02.** `:103` cites `internal/bench/regression.go:72` for the
  cross-runner refusal (correct), while `:224` cites `:53` for the same claim — `:53` is the *platform*
  mismatch message. `:216` cites the Node setup step at 117-120; it is at 118-121. Cosmetic, but these
  are the `read_first` coordinates an executor navigates by.

### Divergent Views

Single-reviewer cycle; divergence is between Codex and the orchestrating reviewer's own verification:

- Codex raised MEDIUM "`.planning/research/STACK.md` must be added to 06-05's `files_modified` because
  the parity phrase is live at STACK.md:93". **Not upheld** — `rg -i parity .planning/research/STACK.md`
  returns no match; STACK.md:93 is a different document's line. The underlying block *is* divergent, but
  in a far more serious way (HIGH 3 above).
- Codex rated the class-fix-A exemptions "conceptually valid" but flagged 06-03 Task 2 as mixing the
  anchor with an inexpressible exception. Confirmed independently and promoted to HIGH 2.

### Cycle-1 status roll-up

| Cycle-1 HIGH | Cycle-2 status |
|---|---|
| HIGH 1 — D-06.1 never asserted | **RESOLVED** — exact `runs-on` + `CODEGRAPH_BENCH_RUNNER` on the publish subtree |
| HIGH 2 — D-06.4 pre-satisfied workflow-wide | **RESOLVED** — upload `with:` map and step-summary assertion both publish-scoped |
| HIGH 3 — D-06.3 rested on a comment | **RESOLVED** — all publish-job run bodies inspected structurally |
| HIGH 4 — committed baseline never loaded | **RESOLVED** — loads the committed path; non-vacuity demonstrated by the shared mutation |
| HIGH 5 — BENCH-03 closable in silence | **RESOLVED in design**; residual mechanical gap tracked as MEDIUM 1 |

## Risk Assessment — Cycle 2

**MEDIUM.** Architecture, wave ordering and instrument design are sound and materially better than
cycle 1. Two acceptance criteria will halt a correct execution (HIGH 1, HIGH 2) and one instruction
will damage `.claude/CLAUDE.md` if followed literally (HIGH 3). All three are localised text fixes in
two plans; none requires re-opening a locked decision. With them corrected the set drops to
**LOW-MEDIUM**.

---

# Cycle 1 — audit history (superseded by Cycles 2 and 3 above)

## Codex Review

# Cross-AI Plan Review

## Summary

The plans are unusually thorough and generally trace the real implementation accurately. Their wave ordering is sound, the comparison-runner subtraction is well scoped, and the mutation/census protocols reflect known repository failure modes. The main weaknesses are verification strength rather than implementation intent: 06-02 does not mechanically enforce all four locked D-06 properties; 06-03 proves the pure `CheckRegression` function but does not prove the committed `baseline.json` participates; and 06-04 permits BENCH-03 to close after a local fallback without a successful workflow run. Several negative-only `git diff` and `rg` criteria also remain vacuous-pass-shaped under rule `84d1gfpywd`.

Overall risk: **MEDIUM**. The plans are likely to produce the intended code, but their acceptance gates can certify incomplete BENCH-02/BENCH-03 outcomes.

---

## Plan 06-01 — Benchmark runner de-coupling

### Strengths

- The subtraction points match the current code precisely: the old mode dispatch, flag, resolver, and two-subject loop are all real surfaces at [tools/bench/runner/main.go:156](/Volumes/Code/github.com/seanb4t/codegraph-go/tools/bench/runner/main.go:156), [tools/bench/runner/main.go:161](/Volumes/Code/github.com/seanb4t/codegraph-go/tools/bench/runner/main.go:161), [tools/bench/runner/main.go:178](/Volumes/Code/github.com/seanb4t/codegraph-go/tools/bench/runner/main.go:178), and [tools/bench/runner/main.go:306](/Volumes/Code/github.com/seanb4t/codegraph-go/tools/bench/runner/main.go:306).

- The paired test edits are correctly identified. Existing tests directly exercise `-ts-binary` and `resolveTSBinary` at [tools/bench/runner/main_test.go:482](/Volumes/Code/github.com/seanb4t/codegraph-go/tools/bench/runner/main_test.go:482) and [tools/bench/runner/main_test.go:494](/Volumes/Code/github.com/seanb4t/codegraph-go/tools/bench/runner/main_test.go:494).

- The end-to-end assertion is strongly positive: exactly two records, exact subject value, positive throughput, and `median_of_trials == 1`. This avoids an empty-output pass.

- The surviving-pin check has independent support from the proposed exact-entry test, so the negative diff check is not the only oracle.

### Concerns

- **MEDIUM — The verification introduces an undeclared Node dependency.** The plan removes Node from the benchmark architecture but parses smoke-test JSON with `node -e` at [06-01-PLAN.md:246](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/06-benchmark-de-coupling-memory-sweep/06-01-PLAN.md:246). A machine with Go and git but no Node fails despite the implemented Go path being correct.

- **LOW — “No `t.Skip`/`//nolint` added” is negative-only and underspecified.** At [06-01-PLAN.md:257](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/06-benchmark-de-coupling-memory-sweep/06-01-PLAN.md:257), there is no companion test-count or file-set assertion specifically proving the test files still exist. The named positive test assertions mitigate this for the three critical tests, but not for the entire package.

- **LOW — `git diff --quiet -- NOTICE` is vacuous-pass-shaped.** At [06-01-PLAN.md:311](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/06-benchmark-de-coupling-memory-sweep/06-01-PLAN.md:311), it passes if `NOTICE` is absent from both compared states or if the check is run after an unintended commit. It does not prove the file exists or retains its expected legal content.

### Suggestions

- Replace Node JSON checks with a small Go verifier or an existing Go test.
- Add `test -f NOTICE` plus a stable positive assertion for its MIT copyright line.
- Record before/after `go test -list` output or exact expected test names rather than relying on the `t.Skip` absence check alone.

### Risk Assessment

**LOW–MEDIUM.** The implementation path is solid; most risk lies in portable verification.

---

## Plan 06-02 — Workflow publisher and baseline prose

### Strengths

- The plan correctly identifies the current comparison setup at [bench.yml:96](/Volumes/Code/github.com/seanb4t/codegraph-go/.github/workflows/bench.yml:96), Node setup at [bench.yml:117](/Volumes/Code/github.com/seanb4t/codegraph-go/.github/workflows/bench.yml:117), npm installation at [bench.yml:122](/Volumes/Code/github.com/seanb4t/codegraph-go/.github/workflows/bench.yml:122), and dual-binary invocation at [bench.yml:138](/Volumes/Code/github.com/seanb4t/codegraph-go/.github/workflows/bench.yml:138).

- The job-id, dispatch option, and conditional are treated as one identifier family. The current three linked sites are visible at [bench.yml:69](/Volumes/Code/github.com/seanb4t/codegraph-go/.github/workflows/bench.yml:69), [bench.yml:71](/Volumes/Code/github.com/seanb4t/codegraph-go/.github/workflows/bench.yml:71), and [bench.yml:99](/Volumes/Code/github.com/seanb4t/codegraph-go/.github/workflows/bench.yml:99).

- Preserving action SHA pins is appropriate. The existing actions are fully pinned, for example [bench.yml:104](/Volumes/Code/github.com/seanb4t/codegraph-go/.github/workflows/bench.yml:104) and [bench.yml:154](/Volumes/Code/github.com/seanb4t/codegraph-go/.github/workflows/bench.yml:154).

- The plan correctly recognizes that `TestWorkflowRunBodiesInvokeTask` does not protect `bench.yml`; the exclusion is explicit at [internal/upgrade/taskfile_shape_test.go:99](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/upgrade/taskfile_shape_test.go:99).

### Concerns

- **HIGH — D-06.1 is stated but not positively asserted.** The plan requires exact `runs-on` and `CODEGRAPH_BENCH_RUNNER` values, but its automated verification at [06-02-PLAN.md:154](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/06-benchmark-de-coupling-memory-sweep/06-02-PLAN.md:154) checks neither. A rewrite could move the publisher to `ubuntu-latest`, omit the environment variable, and still pass every automated criterion. The current load-bearing values are at [bench.yml:98](/Volumes/Code/github.com/seanb4t/codegraph-go/.github/workflows/bench.yml:98) and [bench.yml:101](/Volumes/Code/github.com/seanb4t/codegraph-go/.github/workflows/bench.yml:101).

- **HIGH — D-06.4 is checked workflow-wide rather than publish-job-wide.** `rg -c 'if-no-files-found: error' >= 1` at [06-02-PLAN.md:162](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/06-benchmark-de-coupling-memory-sweep/06-02-PLAN.md:162) passes if the publisher drops its upload assertion because another job already contains the same setting at [bench.yml:274](/Volumes/Code/github.com/seanb4t/codegraph-go/.github/workflows/bench.yml:274). There is also no automated positive assertion for `$GITHUB_STEP_SUMMARY`.

- **HIGH — D-06.3 is asserted mainly through prose.** The new job could call `CheckRegression`, invoke `-mode regression`, or add a shell threshold failure while retaining a “non-blocking” comment. The acceptance criteria only require that a line contain `-mode publish`; they do not structurally inspect all run bodies in the publish job.

- **MEDIUM — D-06.2 is incompletely enforced.** `git diff --quiet -- Taskfile.yml` proves this plan did not edit the Taskfile in the current diff, but not that a `bench:publish` target does not already exist or arrive through another concurrent change. The workflow test deliberately excludes all of `bench.yml` at [internal/upgrade/taskfile_shape_test.go:100](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/upgrade/taskfile_shape_test.go:100).

- **MEDIUM — The SHA equality check is vacuous when all `uses:` lines disappear.** At [06-02-PLAN.md:163](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/06-benchmark-de-coupling-memory-sweep/06-02-PLAN.md:163), `0 == 0` passes. Actionlint does not require a checkout, Go setup, cache, or artifact-upload action.

- **LOW — The `rebless` diff criterion is ambiguous.** “No matches beyond comment-prose lines explicitly listed in the SUMMARY” at [06-02-PLAN.md:165](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/06-benchmark-de-coupling-memory-sweep/06-02-PLAN.md:165) is not mechanically checkable and does not prove the job block is byte-identical.

### Suggestions

Add a structural YAML test that parses the `publish` job and asserts:

- exact `runs-on`;
- exact matching `CODEGRAPH_BENCH_RUNNER`;
- one inline `go run ./tools/bench/runner -mode publish`;
- no `task bench:*`, `CheckRegression`, `-mode regression`, `-rebless`, Node, npm, or alternate binary;
- a summary step containing `$GITHUB_STEP_SUMMARY`;
- an upload step for `publish-results.json` with `if-no-files-found: error`;
- expected action set/count, not merely pin-format equality.

Compare the `rebless` job’s parsed subtree before and after rather than grepping diff text.

### Risk Assessment

**HIGH.** This is the largest verification gap: all four locked D-06 properties are described, but two are weakly checked and two are not mechanically checked at all.

---

## Plan 06-03 — Regression-gate proof

### Strengths

- The selected mutations target real, isolated tolerance constants at [internal/bench/regression.go:9](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/bench/regression.go:9) and [internal/bench/regression.go:15](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/bench/regression.go:15).

- Existing tests provide exact non-vacuous oracles: the 11% throughput case at [internal/bench/regression_test.go:43](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/bench/regression_test.go:43) and the 16% RSS case at [internal/bench/regression_test.go:62](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/bench/regression_test.go:62).

- The plan correctly rejects category-error failures. `CheckRegression` checks platform/runner identity before numeric tolerances, so requiring the RED to name throughput or RSS is important.

- The clean-tree precondition, `EXIT` trap, confirmed mutation diff, RED, revert, and green rerun are a strong application of the repository’s mutation protocol.

### Concerns

- **HIGH — This does not prove “against the committed `baseline.json`.”** The tests construct in-memory `Metrics` values and call `CheckRegression` directly at [internal/bench/regression_test.go:502](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/bench/regression_test.go:502). The committed baseline is merely asserted untouched at [06-03-PLAN.md:229](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/06-benchmark-de-coupling-memory-sweep/06-03-PLAN.md:229); it is never loaded or passed to the function. This proves the tolerance branches remain live, but not the complete BENCH-02 mechanism named in the requirement.

- **MEDIUM — The oracle-integrity check is diff-relative and can pass after an unintended prior commit.** `git diff --quiet -- internal/bench/regression_test.go` only proves the file is currently clean relative to HEAD. It does not prove the test table is byte-identical to the pre-plan version. The before/after subtest count catches deletion/addition counts but not changed values or expected outcomes.

- **LOW — “Comment-only” does not guarantee semantic comments were preserved.** The diff-shape check is useful, but a complete deletion/replacement of all comments would pass if every changed line still starts with `//`. The positive retained-`comparison` count partly mitigates this.

### Suggestions

- Add a small test or command that reads `tools/bench/baseline.json`, unmarshals it into `bench.Metrics`, derives same-category current records with 11% lower throughput and 16% higher RSS, then calls `CheckRegression` and positively asserts the expected error categories.
- Hash `regression_test.go` before Task 1 and compare the hash after Task 2, or explicitly assert the values/names of the two critical cases.
- Keep the mutation rehearsal; it remains valuable even after adding the committed-baseline integration check.

### Risk Assessment

**MEDIUM–HIGH.** The pure function is convincingly proven, but the plan overstates that proof as covering the committed baseline.

---

## Plan 06-04 — Measurement, documentation, and census

### Strengths

- The task order is explicit and genuinely prevents document-first execution. Task 2 has a hard precondition that the capture exists and contains two positive records at [06-04-PLAN.md:151](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/06-benchmark-de-coupling-memory-sweep/06-04-PLAN.md:151). Within normal GSD sequential task execution, numbers cannot be published before Task 1 produces the artifact.

- The capture validation is strongly positive: exact record count, exact repo set, exact subject, and positive values at [06-04-PLAN.md:135](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/06-benchmark-de-coupling-memory-sweep/06-04-PLAN.md:135).

- The proposed document content maps well to the existing methodology. The source currently contains median-of-5 at [docs/BENCHMARKS.md:20](/Volumes/Code/github.com/seanb4t/codegraph-go/docs/BENCHMARKS.md:20), a comparison target at [docs/BENCHMARKS.md:50](/Volumes/Code/github.com/seanb4t/codegraph-go/docs/BENCHMARKS.md:50), and the regression tolerances at [docs/BENCHMARKS.md:261](/Volumes/Code/github.com/seanb4t/codegraph-go/docs/BENCHMARKS.md:261).

- The census includes a positive control, a multiline-vs-line-based comparison, a scanned-file floor, and positive required-content checks. This is substantially better than a negative vocabulary scan alone.

### Concerns

- **HIGH — BENCH-03 can remain unverified while the plan still completes.** The plan explicitly allows a local fallback at [06-04-PLAN.md:118](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/06-benchmark-de-coupling-memory-sweep/06-04-PLAN.md:118), while the live workflow verification applies only “if the preferred CI path was used” at [06-04-PLAN.md:136](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/06-benchmark-de-coupling-memory-sweep/06-04-PLAN.md:136). BENCH-03 says `bench.yml` runs and publishes; a local runner invocation does not establish that.

- **MEDIUM — The byte-identical-to-stdout assertion does not establish byte identity.** At [06-04-PLAN.md:142](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/06-benchmark-de-coupling-memory-sweep/06-04-PLAN.md:142), checking that the indexed file’s first line starts with `[` plus a SUMMARY statement does not compare it to an independently retained stdout capture. It also depends on the file having been staged.

- **MEDIUM — Numeric traceability is only a spot check of `files_per_sec`.** The plan publishes five metrics per corpus but only requires spot-checking `files_per_sec`. Query latency, peak RSS, bytes/sec, and cold start could be transcribed incorrectly without failing.

- **MEDIUM — Local measurements may be unsuitable as published project figures.** The fallback records an empty runner label but does not impose stable runner conditions. This does not contradict D-02, but it weakens reproducibility and makes the published figures potentially machine-specific.

- **LOW — The file-floor assertion can still admit a shifted surface.** `>= 12` proves something was scanned, but not the exact expected file set. A deleted critical file and unrelated added files could preserve the count.

### Suggestions

- Make successful `bench.yml` dispatch a blocking BENCH-03 acceptance criterion. If unavailable, allow BENCH-01 documentation work to proceed but leave BENCH-03 explicitly open.
- Tee stdout simultaneously to a temporary file and the committed artifact, then compare SHA-256 values.
- Generate the Markdown data rows mechanically from JSON or verify all ten metric cells against parsed JSON.
- Record an exact sorted census file list and compare it to the expected set, rather than using only a floor.

### Risk Assessment

**MEDIUM–HIGH.** BENCH-01 ordering is sound, but BENCH-03 can be marked complete without its required workflow execution.

---

## Plan 06-05 — Agent-facing files

### Strengths

- Enumerate-before-edit is excellent for preventing post-hoc classification.
- Source-before-mirror addresses a real current inconsistency: `.claude/CLAUDE.md` still carries the old compatibility framing at [CLAUDE.md:15](/Volumes/Code/github.com/seanb4t/codegraph-go/.claude/CLAUDE.md:15), while PROJECT declares it retired at [PROJECT.md:244](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/PROJECT.md:244).
- The stale capability defect is real: STATE still promises migration at [STATE.md:26](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/STATE.md:26), and PROJECT’s constraint still names `codegraph migrate` at [PROJECT.md:244](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/PROJECT.md:244).
- Region-scoping the PROJECT check is correct because historical mentions legitimately survive elsewhere.

### Concerns

- **MEDIUM — The STATE assertion checks only two exact phrases, not the resulting core-value equality.** At [06-05-PLAN.md:214](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/06-benchmark-de-coupling-memory-sweep/06-05-PLAN.md:214), STATE can retain different stale framing and pass. The task says it must match PROJECT’s Core Value, but no automated equality check enforces that.

- **MEDIUM — Marker count is not byte-integrity.** `GSD_MARKERS=14` at [06-05-PLAN.md:221](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/06-benchmark-de-coupling-memory-sweep/06-05-PLAN.md:221) allows marker names, order, or source attributes to change while retaining the count.

- **LOW — Fresh-session verification is optional/manual in practice.** The human check at [06-05-PLAN.md:215](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/06-benchmark-de-coupling-memory-sweep/06-05-PLAN.md:215) is the closest direct test of MEM-02’s file half, but the completion criteria do not clearly block on it.

- **LOW — Negative `git diff --quiet` checks remain context-dependent.** The protection for NOTICE/LICENSE/CHANGELOG at [06-05-PLAN.md:224](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/06-benchmark-de-coupling-memory-sweep/06-05-PLAN.md:224) proves only no current uncommitted delta, not immutable content across task commits.

### Suggestions

- Parse both Core Value sections and assert exact normalized equality.
- Capture the exact ordered GSD marker lines before editing and compare them afterward.
- Make the fresh-session review an explicit blocking checkpoint or clearly retain MEM-02 as pending until performed.
- Compare protected-file hashes against the pre-plan commit rather than the current working-tree diff.

### Risk Assessment

**MEDIUM.** The editorial reasoning is strong; generated-file integrity and direct MEM-02 proof need firmer gates.

---

## Plan 06-06 — Engram memory sweep

### Strengths

- The fail-loud tool-availability precondition directly respects D-16.
- The plan correctly separates read-only enumeration, one-way classification approval, and durable writes.
- Pagination exhaustion, per-scope page counts, deterministic sorting, zero-record scope reporting, and verdict arithmetic are strong positive assertions.
- The blocking decision checkpoint is appropriate because superseding is one-way.
- The plan explicitly preserves historical clauses in mixed records and forbids substituting overwrite/delete for supersede.

### Concerns

- **MEDIUM — A returned identifier proves write acceptance, not supersede semantics.** The plan acknowledges this at [06-06-PLAN.md:86](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/06-benchmark-de-coupling-memory-sweep/06-06-PLAN.md:86) and [06-06-PLAN.md:228](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/06-benchmark-de-coupling-memory-sweep/06-06-PLAN.md:228). This is the locked D-15 trade-off, so it is not a defect to reopen, but residual risk remains material.

- **MEDIUM — The known-minimum-three-rules assertion may conflate memory records with rule-list entries.** The plan requires a rule-scope record count of at least three at [06-06-PLAN.md:160](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/06-benchmark-de-coupling-memory-sweep/06-06-PLAN.md:160), but research explicitly says the live tool surface is unverified. If rules are exposed through a distinct endpoint or pagination shape, this condition could falsely report a truncated enumeration.

- **MEDIUM — The automated closure check does not reconcile executed entries with approved IDs.** It checks section presence and absence of “delete/overwrite” verdicts, but not set equality between approved `supersede` short IDs and executed rows. The prose acceptance criterion requires equality; the automated command at [06-06-PLAN.md:248](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/06-benchmark-de-coupling-memory-sweep/06-06-PLAN.md:248) does not enforce it.

- **LOW — MEM-02 remains assertion-based by locked decision.** The plan properly calls this out at [06-06-PLAN.md:82](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/06-benchmark-de-coupling-memory-sweep/06-06-PLAN.md:82). This is an accepted evidence limitation, not a planning contradiction.

### Suggestions

- After live tool discovery, distinguish rule objects from ordinary memory records and apply the “at least three” assertion to the correct population.
- Add an exact set comparison: approved supersede `short_id`s must equal executed-original `short_id`s, with no duplicates.
- Record the supersede operation name and returned response shape for each write, not only an identifier.
- Clearly mark MEM-02 as “accepted by D-15 evidence standard” rather than “demonstrated,” unless the optional fresh-session check is actually performed.

### Risk Assessment

**MEDIUM.** The one-way operation is handled carefully, but external-tool uncertainty and the deliberately accepted lack of re-query evidence prevent a low-risk rating.

---

## Cross-plan dependency assessment

The wave graph is coherent:

- 06-01 and 06-03 can run independently with no declared file overlap.
- 06-02 correctly depends on the new `publish` runner mode.
- 06-04 correctly depends on all benchmark-side plans.
- 06-05 follows the completed in-tree state.
- 06-06 follows the file sweep and places durable external writes behind a checkpoint.

The principal dependency issue is semantic rather than scheduling: **06-04’s local fallback allows BENCH-03 to remain unproven even though later plans can proceed and the phase may appear complete.**

## Final recommendation

Approve after revisions to three gates:

1. Add a structural, publish-job-scoped verifier for all four D-06 properties in 06-02.
2. Add a committed-`baseline.json` integration assertion to 06-03.
3. Make a successful `bench.yml` publish run mandatory for BENCH-03, while allowing local measurement only as a BENCH-01 fallback.

Those changes would reduce overall execution risk from **MEDIUM** to **LOW–MEDIUM** without reopening any locked decision.

---

## Consensus Summary

Only one external reviewer lane was requested (`--codex`), so there is no multi-reviewer
consensus to synthesize. What follows is the orchestrator's own verification of Codex's
findings against the repository — every HIGH below was independently re-checked against the
cited file before being carried forward.

Codex's verdict: **approve after revisions**, overall risk MEDIUM. The plans are judged
unusually thorough and their wave ordering sound; the weaknesses are concentrated in
*verification strength*, not implementation intent. Three gates would have to change to close
the review: a publish-job-scoped structural verifier for all four D-06 properties (06-02), a
committed-`baseline.json` integration assertion (06-03), and a mandatory successful `bench.yml`
run for BENCH-03 (06-04).

### Strengths (single reviewer, source-verified)

- Subtraction points in 06-01 match the live code exactly (`tools/bench/runner/main.go:156,161,178,306`), and the paired test edits at `main_test.go:482,494` are correctly identified.
- 06-03's mutation targets are real isolated constants (`internal/bench/regression.go:9,15`) with exact pre-existing oracles (`internal/bench/regression_test.go:43,62`), and the plan correctly rejects category-error REDs.
- 06-04 Task 2 carries a hard `<precondition>` that `06-PUBLISH-RESULTS.json` exists with two positive records (`06-04-PLAN.md:151`) — **the ordering question raised for this review is answered affirmatively**: under normal GSD sequential task execution the measurement provably lands before the document that publishes it.
- 06-06 correctly separates read-only enumeration, a blocking one-way classification checkpoint, and durable writes, and respects D-16's fail-loud tool-availability precondition.

### Concerns — HIGH (all five independently re-verified)

1. **06-02 / D-06.1 is never asserted.** The `<automated>` block at `06-02-PLAN.md:154` and every acceptance criterion at `:157-166` check the job key, the conditional, the retired-term census, the SHA pins, and three `git diff` negatives — but **nothing** checks `runs-on: namespace-profile-linux-amd64-4x8` or the `CODEGRAPH_BENCH_RUNNER` env var. Confirmed: those are the load-bearing values at `.github/workflows/bench.yml:98,101`, and D-06.1 calls runner identity "part of the measurement's validity, not cosmetics" (`06-CONTEXT.md:78-82`). A from-scratch rewrite onto `ubuntu-latest` with no env var passes every automated criterion.
2. **06-02 / D-06.4 is checked workflow-wide, and is already satisfied elsewhere.** `test "$(rg -c 'if-no-files-found: error' .github/workflows/bench.yml)" -ge 1` (`06-02-PLAN.md:154,162`) is satisfied by the pre-existing `rebless` job upload at `.github/workflows/bench.yml:274`. Confirmed by reading that line. The publish job can therefore omit its own upload assertion and still pass. Separately, D-06.4 also requires a `GITHUB_STEP_SUMMARY` block (`06-02-PLAN.md:26,132`) and **no criterion asserts `$GITHUB_STEP_SUMMARY` at all**.
3. **06-02 / D-06.3 (non-blocking publish-not-gate) rests on prose.** The contract is restated only in a run-step comment (`06-02-PLAN.md:128`). No criterion structurally inspects the publish job's run bodies, so a rewrite that calls `CheckRegression`, invokes `-mode regression`, or adds a shell threshold failure — while keeping the "non-blocking" comment — passes.
4. **06-03 does not prove the gate fires "against the committed `baseline.json`."** Roadmap success criterion 2 names the committed baseline explicitly. The rehearsal drives `CheckRegression` through in-memory `Metrics` values; `06-03-PLAN.md:229` only asserts `git diff --quiet -- tools/bench/baseline.json`, i.e. that the file was *not touched*. Verified by search: **no Go test loads `tools/bench/baseline.json`** — the only test references (`tools/bench/runner/main_test.go:318,731`) write their own baseline into a temp dir. The tolerance branches are proven live; the committed-data path is not.
5. **06-04 lets BENCH-03 close without ever running the workflow.** The plan sanctions a local fallback at `06-04-PLAN.md:118-124`, and BENCH-03's only live verification is a `<human-check>` explicitly scoped "If the preferred CI path was used" (`06-04-PLAN.md:136`). Verified verbatim. Taking the fallback leaves BENCH-03 — "a `bench.yml` run publishes the absolute numbers without invoking another implementation anywhere in the job" — with **zero** verification, while the plan still completes.

### Concerns — MEDIUM / LOW

Per-plan detail is in the Codex section above. Two cross-cutting shapes the orchestrator adds
after censusing all six plans (28 occurrences of `git diff --quiet` / "returns no matches"):

- **`git diff --quiet -- <path>` is HEAD-relative, not baseline-relative.** Every use of it as a
  tamper guard (NOTICE, LICENSE, CHANGELOG.md, `tools/bench/baseline.json`, `internal/bench/`,
  the 06-05 target files) proves only that the *working tree* currently matches HEAD. Because
  GSD commits per task, any change made and committed by an earlier task in the same plan is
  invisible to a later task's guard. To hold across tasks these must compare against the
  pre-plan commit (`git diff --quiet <pre-plan-sha> -- <path>`) or a recorded hash.
- **`rg -c '<pat>'` used as a "returns no matches" assertion is an inverted gate.** `rg -c`
  exits 1 and prints nothing on the clean path, so any executor chaining it with `&&` sees the
  *good* outcome as a failure — and any executor reading exit status leniently sees a broken
  command as a pass. This repo already has the inverted-`rg -qv` gate in its documented
  failure history. Occurrences: `06-01-PLAN.md:255`, `06-02-PLAN.md:164,165,215`,
  `06-03-PLAN.md:158,228`. Restate each as `test "$(… | rg -c … || true)" = "0"` with a printed
  positive token, paired with a count that must be > 0.

### Divergent Views

None — single reviewer.

### Locked-decision contacts (reported, not raised as defects)

- 06-06's "a returned identifier proves write acceptance, not supersede semantics" is the
  **locked D-15 trade-off**, acknowledged in-plan at `06-06-PLAN.md:86,228`. Reported as
  residual risk, not a defect.
- MEM-02 remaining assertion-based rather than demonstrated is likewise accepted by D-15 and
  called out at `06-06-PLAN.md:82`.
- The local-measurement fallback in 06-04 does not contradict D-02; the HIGH above is about
  BENCH-03's *verification*, not about re-opening the fallback decision.

---

## Verification coverage

The orchestrator independently re-read the following before accepting Codex's findings. Every
HIGH was confirmed at its cited line; none were taken on the reviewer's word.

| Claim | Source checked | Verdict |
|---|---|---|
| D-06.1 unasserted | `06-02-PLAN.md:140-166`, `.github/workflows/bench.yml:96-101` | CONFIRMED — no `runs-on` / `CODEGRAPH_BENCH_RUNNER` assertion exists |
| D-06.4 satisfied by another job | `.github/workflows/bench.yml:274` | CONFIRMED — `if-no-files-found: error` present in the `rebless` upload |
| D-06.4 step-summary unasserted | `06-02-PLAN.md:26,132,154-166` | CONFIRMED — requirement stated, never asserted |
| D-06.3 prose-only | `06-02-PLAN.md:128,157-166` | CONFIRMED |
| D-06 is a four-property locked list | `06-CONTEXT.md:77-90` | CONFIRMED |
| Committed baseline never loaded | `rg 'baseline\.json' --type go internal/ tools/`; `06-03-PLAN.md:221-231` | CONFIRMED — only temp-dir baselines in tests |
| BENCH-03 human-check is conditional | `06-04-PLAN.md:118-124,136` | CONFIRMED verbatim |
| 06-04 ordering is guaranteed | `06-04-PLAN.md:151` `<precondition>` | CONFIRMED — Codex's *strength* finding is correct; the measurement provably precedes the document |
| Negative-guard census | 28 hits across all six `*-PLAN.md` | CONFIRMED — enumerated above |

Codex ran with repo access and produced `file:line`-grounded findings throughout; no
`[reviewed-without-repo-access]` marker was present.

---

## Verification coverage — Cycle 2

Codex ran with full repo access and produced `file:line`-grounded findings throughout; no
`[reviewed-without-repo-access]` marker was present. The orchestrating reviewer independently
re-executed or re-read every load-bearing claim below against the working tree at `6a2816b`
before accepting it. One Codex finding was **not upheld**.

| Claim | Source / command actually run | Verdict |
|---|---|---|
| SHA-pin arithmetic (expected 19) | `rg -o '^\s*uses: ' / '^\s*uses: \./' / 'uses: [^@[:space:]]+@[0-9a-f]{40}' .github/workflows/bench.yml` → 22 / 2 / 20; `bench.yml:118` is the deleted pinned action | CONFIRMED — 19 pinned + 2 local = 21 total after the edit |
| Publish-job scoping is real | `06-02-PLAN.md:41,140-184,266` | CONFIRMED — every D-06 assertion anchored to the `jobs.publish` subtree |
| `rebless` upload would pre-satisfy a file-wide check | `.github/workflows/bench.yml:270-274` | CONFIRMED present — and now provably out of scope |
| Committed baseline genuinely loaded | `06-03-PLAN.md:236-240` (`../../tools/bench/baseline.json`), `tools/bench/baseline.json` (populated `goos`/`goarch`/`runner`/`scratch_fs`) | CONFIRMED — real file, category guards stay quiet for a frame-derived record |
| Mutation reddens both oracles | `internal/bench/regression.go:9,15,114,124`; `internal/bench/regression_test.go:42-69,512` | CONFIRMED for the RED itself — **but** the emitted text is `CheckRegression() = nil, want error` (→ HIGH 1) |
| Task 2 pre-plan diff is impossible | `06-03-PLAN.md:140,166-192,296` | CONFIRMED — Task 1 commits all four named files |
| `\bcomparison\b >= 5` floor survives the sweep | `rg -n '\bcomparison\b' tools/bench/BASELINE.md` → lines 39/87/97/120/297; swept lines are 64 and 241 | CONFIRMED — disjoint; floor holds with zero headroom |
| `internal/bench/regression.go` comparison floor | `rg -c` → 7 (floor 5) | CONFIRMED |
| Taskfile exact-set + floor | `BENCHTASKS=bench:regression:,`, `ALLTASKS=47` (floor 40) | CONFIRMED |
| Fixture-entry floor | `rg -o '^\s*\{Workflow:' internal/upgrade/taskfile_shape_test.go` → 10 (floor 10) | CONFIRMED — exactly at the floor |
| GSD marker count | `rg -o '<!-- GSD:[^>]*-->' .claude/CLAUDE.md` → 14 | CONFIRMED |
| Core-value extraction executes | 06-05's `PV`/`SV` commands run verbatim → `PV_LEN=208` (floor 60), `SV` non-empty | CONFIRMED — gate is satisfiable and non-vacuous |
| Compatibility bullet region-scope | `rg '^\- \*\*Compatibility\*\*' .planning/PROJECT.md` matches `:244`; `codegraph migrate` in it = 1, in the file = 3 | CONFIRMED — file-wide ban would indeed be unsatisfiable |
| Census surface + critical files | `rg -U --files <surface>` → 26 files today, 20 after 06-01 deletes the six captures; all 13 critical files present | CONFIRMED — floor 12 and `13/13` both hold |
| Phase-6 census is reachable | Ran the full cycle-2 pattern set → 168 hits today, every one inside a declared sweep target | CONFIRMED — a zero is attainable |
| Surviving corpus pins | `rg -o 'f89ae3ea…|dbdc1acb…' tools/bench/realcorpus/manifest.go` → 2; manifest has 3 entries today | CONFIRMED |
| Codex: `.planning/research/STACK.md` needs editing | `rg -i parity .planning/research/STACK.md` → **no match** | **NOT UPHELD** — false finding; the real defect is the wholesale block divergence (HIGH 3) |
| Stack block divergence | `diff` of `.claude/CLAUDE.md` lines 22-140 vs `.planning/research/STACK.md` | CONFIRMED — different documents; `STACK.md:151` contains `drop-in`, which would break `CLAUDE_MD_FRAMING_TOTAL=0` |
| Three HEAD-relative exemptions | `06-05-PLAN.md:101`, `06-02-PLAN.md:321`, `06-03-PLAN.md:402-407` | CONFIRMED correct (one residual caveat → LOW 3) |
| No sound `rg -c` floor over-corrected | Enumerated all 24 `rg -c` occurrences across the six plans | CONFIRMED — every remaining use is a floor, a line-oriented count, or a `|| true`-guarded restatement |
