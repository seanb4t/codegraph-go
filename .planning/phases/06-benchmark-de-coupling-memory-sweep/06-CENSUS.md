# Phase 6 Acceptance Census

Positive-controlled multiline census over the declared Phase 6 file surface, per D-04 and
06-04-PLAN.md Task 4. The instrument is proven live against a planted wrapped-phrase fixture
before any zero it later reports is trusted.

## Step 1-2: Positive control

Fixture (`.planning/phases/06-benchmark-de-coupling-memory-sweep/.scratch-poscontrol.txt`,
deleted before this plan's final commit — content reproduced verbatim below):

```
// the retired harness invoked a second (TS
// codegraph@1.3.1) binary here.
// this mirrored the head-to-
// head runner from colbymchenry/
// codegraph-ts.
```

Probe patterns run twice over the fixture — once with `rg -U -o` (multiline), once with plain
`rg -o` (line-based):

```
-e "\bTS[[:space:]/*#`>@-]*codegraph"  -e '\bhead-to-head\b'
-e '\bcolbymchenry\b'                  -e '\bcodegraph-ts\b'
-e "\bhead-to-[[:space:]/*#]+head\b"   -e "\bcolbymchenry/[[:space:]/*#]+codegraph-ts\b"
```

**`POSCTRL_MULTILINE=6 POSCTRL_LINEBASED=2`** — multiline strictly beats line-based (6 > 2) and is
strictly greater than 0. The instrument catches wrapped phrases the line-based form misses; its
later zero is trusted.

## Step 3: The bounded pattern set

Run with `rg -U -o -i`, `W='[[:space:]/*#`>@-]*'` and `S='[[:space:]/*#]+'`:

```
-e "\bTS${W}codegraph"                 -e "\bTS${W}Node\b"
-e "\bTS[[:space:]-]*(binary|binaries|CLI|project|subject|runner|harness)\b"
-e '\bts-binary\b'                     -e '\btsBinary\b'
-e "\bGo[[:space:]/-]*vs\.?[[:space:]/-]*TS\b"   -e '\bGo/TS\b'
-e "\bTS[[:space:]]*1\.3\.1\b"         -e '"ts"'
-e '\|[[:space:]]*ts[[:space:]]*\|'
-e '\bhead-to-head\b'                  -e '\bheadtohead\b'
-e '\bcolbymchenry\b'                  -e '\bcodegraph-ts\b'
-e '\bcomparator\b'                    -e '\b1\.3\.1\b'
-e "\bcomparison${S}(binary|harness|runner|target|subject|mode)\b"
```

Bare `comparison` is deliberately EXCLUDED: `internal/bench` and `tools/bench/BASELINE.md` use it
correctly for the gate comparing a baseline record against a current record, and Phase 5 already
recorded an incident where an unbounded pattern matched innocent prose and an executor reworded
the prose instead of bounding the pattern.

Bare `\bTS\b` is EXCLUDED for the same reason — class fix D. `TS` meaning
TypeScript-the-indexed-language is this repository's recorded KEEP class, and leaving the
alternation bare made `PHASE6_CENSUS_TOTAL = 0` unreachable by any correct execution (see the
`corpora/manifest.json:10` adjudication below). It is still run as a separate completeness control
(`BARE_TS_RESIDUAL_LINES`, below) so the bounding cannot hide anything from the instrument that
exists to check it.

## File surface

`docs/BENCHMARKS.md`, `tools/bench/`, `internal/bench/`, `internal/corpora/manifest.go`,
`corpora/manifest.json`, `.github/workflows/bench.yml`, `internal/upgrade/taskfile_shape_test.go`.

**`CENSUS_FILES_SCANNED=23`** (`rg -U --files` over the surface, floor is `>= 12`).
**`CENSUS_CRITICAL_PRESENT=13/13`** — every one of the thirteen critical files appears in the
walked list by name: `docs/BENCHMARKS.md`, `tools/bench/BASELINE.md`, `tools/bench/baseline.json`,
`tools/bench/realcorpus/manifest.go`, `tools/bench/realcorpus/manifest_test.go`,
`tools/bench/runner/main.go`, `tools/bench/runner/main_test.go`, `internal/bench/regression.go`,
`internal/bench/metrics.go`, `internal/bench/rss.go`, `internal/corpora/manifest.go`,
`.github/workflows/bench.yml`, `internal/upgrade/taskfile_shape_test.go`. None missing.

Full walked list (23 files):

```
.github/workflows/bench.yml
corpora/manifest.json
docs/BENCHMARKS.md
internal/bench/baseline_gate_test.go
internal/bench/metrics.go
internal/bench/regression.go
internal/bench/regression_test.go
internal/bench/rss.go
internal/bench/rss_test.go
internal/corpora/manifest.go
internal/upgrade/taskfile_shape_test.go
tools/bench/BASELINE.md
tools/bench/baseline.json
tools/bench/cpudiag/main.go
tools/bench/gencorpus/gen.go
tools/bench/gencorpus/gen_test.go
tools/bench/gencorpus/main.go
tools/bench/publishcheck/main.go
tools/bench/publishcheck/main_test.go
tools/bench/realcorpus/manifest.go
tools/bench/realcorpus/manifest_test.go
tools/bench/runner/main.go
tools/bench/runner/main_test.go
```

## Per-file bounded-set hit counts (measured post-execution, this replan)

| File | bounded-set hits | disposition |
|---|---|---|
| `docs/BENCHMARKS.md` | 0 | rewritten by this plan's Task 2 on absolute-measurement terms |
| `tools/bench/runner/main.go` | 0 | swept by 06-01 |
| `.github/workflows/bench.yml` | 0 | swept by 06-02 |
| `tools/bench/realcorpus/manifest.go` | 0 | swept by 06-01 |
| `tools/bench/runner/main_test.go` | excluded (see below) | pre-authorised file-level exclusion — the removal-proof test 06-01 mandates |
| `tools/bench/realcorpus/manifest_test.go` | 0 | swept by 06-01 |
| `internal/bench/metrics.go` | 0 | swept by 06-03 |
| `tools/bench/BASELINE.md` | 0 | swept by 06-02 |
| `internal/bench/regression.go` | 0 | swept by 06-03 |
| `internal/upgrade/taskfile_shape_test.go` | 0 | swept by 06-02 |
| `internal/bench/rss.go` | 0 | swept by 06-03 |
| `internal/bench/regression_test.go` | 0 | swept by 06-03 |
| `corpora/manifest.json` | 0 | never needed sweeping (KEEP-class residual only under the bare-`TS` completeness control, adjudicated below) |
| `tools/bench/publishcheck/main_test.go` | 0 under the two narrowed patterns (see Pattern narrowing below); would be 2 unnarrowed | new file, authored by 06-01, carries a meta-commentary comment about the retired `"ts"` value it deliberately does NOT use |
| every other walked file | 0 | not a sweep target of any plan; no retired framing present |

**`PHASE6_CENSUS_TOTAL=0`** — the sum of (a) the bounded set minus the two `"ts"`-quoting patterns,
run over the whole surface with `tools/bench/runner/main_test.go` excluded (**0**), plus (b) the
two `"ts"`-quoting patterns (`'"ts"'` and `'\|[[:space:]]*ts[[:space:]]*\|'`) run over the whole
surface with `tools/bench/publishcheck/main_test.go` excluded (**0**). See "Pattern narrowing"
below for why (b) is split out and why the exclusion is sound.

## The one FILE-level exclusion (cycle-5 HIGH 1, pre-authorised)

`tools/bench/runner/main_test.go` is excluded from the bounded census via the path-anchored glob
`-g '!tools/bench/runner/main_test.go'`. `06-01-PLAN.md:34` (`must_haves`) and its Task 1
`<behavior>` block mandate `TestParseFlags_RejectsRetiredComparisonBinaryFlag`, required by D-07
and rule `84d1gfpywd` as the positive proof that the retired `-ts-binary` flag is gone rather than
merely undocumented. Go's `flag` package identifies flags by name, so every faithful
implementation of that test contains the literal `ts-binary`, which the bounded set's
`\bts-binary\b` pattern counts. Neither this plan nor its executor can fix the source (the file is
not in this plan's `files_modified`), and the pattern lives inside this plan's own `<action>`, not
an editable target.

**Bounded three ways, per plan design:**
1. Quoted and adjudicated here, citing `06-01-PLAN.md:34` / D-07 / rule `84d1gfpywd`.
2. Companion assertions over the excluded file, run alongside the census:
   `EXCLUDED_FILE_TS_BINARY_OCCURRENCES=1` (exactly one `ts-binary` literal, at
   `tools/bench/runner/main_test.go:503`, inside `TestParseFlags_RejectsRetiredComparisonBinaryFlag`),
   `REJECTION_TEST_PRESENT=1` (the named test exists, exactly once), `EXCLUDED_FILE_OTHER_RETIRED_TERMS=0`
   (no other retired term appears in the file — measured with the full pattern set minus
   `ts-binary` itself).
3. The bare-`TS` completeness control below still walks this file unexcluded — the excluded line
   (`_, err := parseFlags([]string{"-ts-binary", "/custom/codegraph"})`) reappears there and is
   quoted and adjudicated in that section.

## Pattern narrowing found during this replan (new — not anticipated by 06-01/06-02/06-03)

**`tools/bench/publishcheck/main_test.go:48` and `:50`** flagged the bounded set's `'"ts"'`
pattern. Quoted verbatim:

```
48:		// Deliberately "other", not "ts": tools/bench sits inside 06-04's
50:		// quoted "ts" as retired subject framing (cycle-3 new trap).
```

This is genuinely innocent: it is a comment 06-01 authored explaining *why* the wrong-subject test
fixture uses the literal `"other"` rather than `"ts"` — the very trap 06-01-SUMMARY.md's Decisions
section already records (`"the obvious literal for publishcheck's wrong-subject fixture is
'ts'... 06-01's <behavior> block now pins that fixture value to 'other' and states why"`). The
comment discusses the retired term to explain its deliberate avoidance; it does not commit it. No
Go code in this file ever assigns `"ts"` as a value — verified positively:
`rg -o -F '"ts"' tools/bench/publishcheck/main_test.go` returns exactly 2 (both inside this one
comment), and `rg -n 'Subject.*=.*"ts"' tools/bench/publishcheck/main_test.go` returns 0.

**Per this plan's own instruction — "if a pattern flags text that is genuinely innocent, the
finding is about the pattern; bound it tighter and record the change; never reword the innocent
text" — the two `"ts"`-quoting patterns (`'"ts"'`, `'\|[[:space:]]*ts[[:space:]]*\|'`) are run with
a second, path-anchored exclusion, `-g '!tools/bench/publishcheck/main_test.go'`, separately from
the rest of the bounded set.** This is the same bounding shape as the pre-authorised
`tools/bench/runner/main_test.go` exclusion above, applied here at pattern granularity because only
two of the sixteen patterns are affected, and companion-checked the same way:

- `rg -o -F '"ts"' tools/bench/publishcheck/main_test.go` = **2** (both are the flagged comment
  lines, quoted above — the exclusion's entire scope, pinned).
- `rg -n 'Subject.*=.*"ts"' tools/bench/publishcheck/main_test.go` = **0** (no real assignment
  hides behind the comment).
- The bare-`TS` completeness control below still walks this file unexcluded — both flagged lines
  reappear there and are quoted and adjudicated in that section, so the narrowing cannot hide them
  from the instrument that exists to check it.

The rewording rule does not apply here: nothing in this comment describes retired comparison
framing as this project's present or future state; it documents a naming trap for future census
authors, which is exactly the kind of institutional knowledge D-04 exists to preserve, not erase.

`rg -o -F 'bare `comparison`' 06-CENSUS.md` note: **bare `comparison`** is deliberately excluded from
the bounded pattern set (see "The bounded pattern set" above) — `internal/bench` and
`tools/bench/BASELINE.md` use the word correctly for `CheckRegression`'s baseline-vs-current
comparison, and Phase 5 already recorded an incident where an unbounded pattern matched innocent
prose and an executor reworded the prose instead of bounding the pattern. That decision is
restated here so this record is self-contained.

## Completeness control: bare `\bTS\b` (no exclusions, whole surface)

**`BARE_TS_RESIDUAL_LINES=4`** (floor `>= 1`, not `= 1`, deliberately — a rewritten
`docs/BENCHMARKS.md` may legitimately name `TS/JS` as an indexed-language pair, so an exact count
would redden correct work). Every residual line, quoted and adjudicated:

1. **`corpora/manifest.json:10`**
   ```
   "note": "LOCKED (01-06): closes the Go priority-4 coverage gap and supplies TS/JS via its 25 javascript tracked files, which is how this minimum-cardinality set covers all five priority languages. Roadmap shortlist."
   ```
   Adjudication: **(b) TypeScript-the-indexed-language KEEP sense.** `TS/JS` here names the
   indexed-language pair this corpus manifest entry covers — not retired comparison framing. Cited
   to the same Phase-5 KEEP-class adjudication the `internal/indexer/**` and `testdata/golden/**`
   exclusions cite (05-08-PLAN.md's product-surface assertions; see also 06-01-PLAN.md's cycle-3
   HIGH 2 finding, which first measured this exact line as the reason the bare pattern is
   unreachable). This file is in no plan's `files_modified` and its text is unchanged since
   `fd02272`.

2. **`tools/bench/publishcheck/main_test.go:48`**
   ```
   // Deliberately "other", not "ts": tools/bench sits inside 06-04's
   ```
   Adjudication: **(a) already covered by another pattern in the set** — the bounded `'"ts"'`
   pattern, which is why this exact line was flagged and adjudicated above under "Pattern
   narrowing", where its exclusion (and the reasoning for it) is recorded in full.

3. **`tools/bench/publishcheck/main_test.go:50`**
   ```
   // quoted "ts" as retired subject framing (cycle-3 new trap).
   ```
   Adjudication: **(a) already covered by another pattern in the set** — same disposition and same
   citation as line 48 above (both lines are the same meta-commentary comment).

4. **`tools/bench/runner/main_test.go:503`**
   ```
   _, err := parseFlags([]string{"-ts-binary", "/custom/codegraph"})
   ```
   Adjudication: **(a) already covered by another pattern in the set** — the bounded
   `\bts-binary\b` pattern, whose match against this exact line is the pre-authorised file-level
   exclusion documented above ("The one FILE-level exclusion"), companion-checked there
   (`EXCLUDED_FILE_TS_BINARY_OCCURRENCES=1`, `REJECTION_TEST_PRESENT=1`,
   `EXCLUDED_FILE_OTHER_RETIRED_TERMS=0`). The bare-`TS` control re-surfacing it here is exactly the
   design intent: the narrowing cannot hide the file from this instrument.

No residual is unadjudicated. No residual required rewording innocent text — every non-KEEP
residual was already covered by a bounded, companion-checked, cited exclusion.

## Exclusion list (every entry traces to a recorded prior adjudication)

| Exclusion | Reason | Source |
|---|---|---|
| `NOTICE` | Upstream MIT licence attribution — a legal obligation, not framing | Phase 4 adjudication |
| `CHANGELOG.md` | release-please-owned | `.planning/STATE.md` §Accumulated Context → Decisions |
| `.planning/**` | Append-only record parsed by scope-sensitive tooling | `.planning/STATE.md` §Accumulated Context → Decisions |
| `internal/indexer/**`, `testdata/golden/**` | TypeScript-the-indexed-language product surface and its captured fixtures — KEEP class | Phase 5 (05-08-PLAN.md's product-surface assertions) |
| `Taskfile.yml`, wireoracle / query / TUI / agents / MCP packages | Outside the Phase 6 surface; their `comparison` uses are the ordinary sense | Not walked by this census's declared file surface (not applicable — never in scope) |
| `tools/bench/runner/main_test.go` (bounded pattern census only; still walked by the bare-`TS` control) | Removal-proof test for the retired `-ts-binary` flag must contain the literal by Go `flag` naming | `06-01-PLAN.md:34` / D-07 / rule `84d1gfpywd` (cycle-5 HIGH 1, pre-authorised) |
| `tools/bench/publishcheck/main_test.go` (only the `'"ts"'` and pipe-`ts` patterns; every other pattern in the set still walks this file; still walked by the bare-`TS` control) | Meta-commentary comment quoting the retired `"ts"` value to explain why it is deliberately NOT used | New this replan — see "Pattern narrowing" above, following D-04's own remedy ("bound the pattern, record the change, never reword innocent text") |

Adding an exclusion not on this list, or not backed by a citation, is forbidden.

## Build/vet/test gate

`go build ./...`, `go vet ./...`, and
`go test -count=1 ./internal/bench/... ./internal/corpora/... ./internal/upgrade/... ./tools/bench/...`
all pass (verified this replan; zero behaviour changed by this plan's edits to
`docs/BENCHMARKS.md`, `COVERAGE.md`, `06-LIVE-VERIFICATIONS.md`, or this census).

`rg -c '\bcomparison\b' internal/bench/regression.go` = 7; `rg -c '\bcomparison\b'
tools/bench/BASELINE.md` = 5 — both non-zero, proving the legitimate ordinary-sense prose survives
and this phase's zero was not bought with a find-and-replace.
