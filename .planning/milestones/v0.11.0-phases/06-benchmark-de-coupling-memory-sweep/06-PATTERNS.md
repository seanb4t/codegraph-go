# Phase 6: Benchmark De-coupling & Memory Sweep - Pattern Map

**Mapped:** 2026-08-16
**Files analyzed:** 15 (creates + modifies + deletes)
**Analogs found:** 13 / 15 (2 explicitly "no analog" — see below)

**Calibration note:** This phase is subtraction-heavy. For deleted/swept files the useful
analog is a *prior deletion or sweep done well in this repo* (Phase 4's FLAG-PARITY removal,
Phase 5's `codegraph migrate` removal, Phase 5's `behavioral_test.go` multiline census), not a
structurally-similar file. RESEARCH.md already carries file:line grounding for nearly every
touch point in this phase's own surface — this document's job is to attach *removal-shaped*
precedent from Phase 4/5, plus the two genuinely-new artifacts' direct analogs.

## File Classification

| New/Modified/Deleted File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| `docs/BENCHMARKS.md` | config/doc (rewrite-from-scratch) | batch (measurement report) | Phase 4's `docs/FLAG-PARITY.md` deletion (comparison-doc removal) + the file's own surviving §2/§3 sections (RESEARCH.md Pattern 1) | role-match (rewrite, not delete, so partial) |
| `tools/bench/BASELINE.md` | doc (framing sweep, content kept) | batch | Phase 5's `testdata/golden/behavioral_test.go` sweep (05-07/05-08) — reword framing, preserve substantive content | exact (same shape: sweep prose, keep the engineering finding) |
| `tools/bench/headtohead-*.json` (6 files) | fixture/data (deleted) | batch | Phase 5's `internal/migrate/` + fixture deletion (`05-01-SUMMARY.md`) | exact (outright deletion, git history preserves) |
| `tools/bench/runner/main.go` | CLI/service (subtraction: `-mode headtohead` → `-mode publish`) | request-response (CLI flag dispatch) → batch | Phase 5's `internal/cli/migrate.go` command-removal precedent (delete a mode/branch, its flags, its helper functions) | role-match |
| `tools/bench/runner/main_test.go` | test | unit | Phase 5's `internal/cli/migrate_test.go` deletion pattern (delete tests asserting the removed surface) | exact |
| `tools/bench/realcorpus/manifest.go` | config/model (data) | CRUD (static manifest) | Phase 4's identifier-family sweep (FLAG-PARITY) applied to comparison prose; D-09's entry-drop mirrors D-01's file-deletion shape | role-match |
| `tools/bench/realcorpus/manifest_test.go` | test | unit | Phase 5's `05-01-SUMMARY.md` test-fallout handling (assertions on since-removed manifest entries deleted alongside the entry) | exact |
| `internal/corpora/manifest.go` | config/model (doc-comment + `Manifest.Note` review) | CRUD | Phase 4 FLAG-PARITY reference sweep (cross-file pointer updated to match the renamed/reworded target) | role-match |
| `internal/bench/rss.go` | utility (doc-comment reword only) | transform | Phase 5's `behavioral_test.go` comment-reword-only sweep (no literal/logic touched, only doc comment framing) | exact |
| `internal/bench/regression.go` | service (gate logic, UNCHANGED except transient mutation) | request-response (pure comparison function) | `03-MUTATION-LOG.md`'s FIXT-07 protocol applied to a comparison function | exact |
| `.github/workflows/bench.yml` | CI config (job deleted + fresh job authored) | event-driven (workflow_dispatch/schedule) | Phase 5's `internal/migrate` command-removal + Phase 4's PR-template/workflow reference sweep, combined | role-match |
| `06-MUTATION-LOG.md` | doc (new artifact) | batch (record) | `.planning/phases/03-non-vacuity-proof-unconditional-ci-execution/03-MUTATION-LOG.md` | exact |
| `06-MEMORY-SWEEP.md` | doc (new artifact) | batch (record) | **No analog in this repo** — no prior phase has swept an external durable-memory store | none |
| `.claude/CLAUDE.md` | doc (framing reword) | batch | Phase 4/5's identifier-family sweep pattern, applied to prose rather than code identifiers | role-match |
| `.planning/PROJECT.md`, `.planning/STATE.md` | doc (framing reword) | batch | Same as above | role-match |

## Pattern Assignments

### `docs/BENCHMARKS.md` (doc, rewrite-from-scratch, D-02)

**Analog A — the deletion precedent:** Phase 4's FLAG-PARITY removal (`04-02-SUMMARY.md:4-30`).

```
title: Delete FLAG-PARITY and its drift guard, sweep all references
Deleted `docs/FLAG-PARITY.md` and `internal/cli/flag_parity_test.go` (the comparison-framed
doc and its drift guard) and swept every reference to either.
## Reference sweep (word-boundary identifier family: FLAG-PARITY, flag_parity, flag-parity,
flagParity, TestFlagParityDocCoversRegisteredFlags)
```
`docs/BENCHMARKS.md` is a rewrite, not a delete, but the identifier-family-sweep discipline
(word-boundary pattern set, per-file reference table) is the exact instrument to reuse for
confirming no `TS`/`comparison`/`colbymchenry` residue survives in the new doc.

**Analog B — the section-by-section survival map:** already produced in RESEARCH.md Pattern 1
(`06-RESEARCH.md:198-215`) — reuse verbatim, do not re-derive:
- Title/intro: rewritten (currently "Benchmarks: codegraph-go vs TS CodeGraph")
- `## 1. Methodology` (`docs/BENCHMARKS.md:7-48`): keep the measurement discipline (median-of-5,
  OS-level `getrusage` RSS), reword only the closing TS-contrast paragraph (`:40-48`)
- `### Comparison target` (`:50-55`): deleted entirely
- `### Pinned real repos (PERF-01)` (`:57-73`): table narrows to 2 rows (weft-go,
  cockroachdb-pebble), reword "why this repo" per D-09
- `### Regenerating the numbers` (`:75-87`): reword to cite `-mode publish`
- `## 2. Raw numbers` (`:89-241`): **requires a fresh absolute-only measurement run** — this is
  a plan-level task dependency, not an editing task (D-02)
- `### The one real, committed number...` (`:213-240`): keep near-verbatim
- `## 3. Regression gate (PERF-02, INDX-06)` (`:242-300`): keep near-verbatim

### `tools/bench/BASELINE.md` (doc, framing sweep, content preserved, D-03)

**Analog:** Phase 5's `testdata/golden/behavioral_test.go` framing sweep
(`05-08-PLAN.md:1-60`, gap-closure plan for CODE-01).

Key excerpt — the shape of "sweep framing, keep substance" as a must_have:
```
"`testdata/golden/behavioral_test.go`'s `assertSubset` and `assertNameFileLineSubset` doc
comments still explain WHY a subset (not equality) is the right assertion ... without naming
another implementation's output as the standard the engine is measured against."
...
"NO change to any string literal, regex literal, assertion, expected value, test name,
subtest name, fixture or helper signature ... this plan changes Go comments only."
```
RESEARCH.md already confirms `tools/bench/BASELINE.md` scores 0 hits on the `\bTS\b` census
(`06-RESEARCH.md:283`) — treat this file's D-03 work as a **confirming pass**, same as Pitfall
3's warning about `corpora/manifest.json`'s `Note` field (do not over-schedule work here).

### `tools/bench/headtohead-*.json` (6 files, deleted, D-01)

**Analog:** Phase 5's `internal/migrate/` package + fixture deletion
(`05-01-SUMMARY.md:106,118-119,168`):
```
Deleted the entire `codegraph migrate` capability — command, `internal/migrate` package,
`modernc.org/sqlite` dependency, graphstore migration-cursor API, and the ts-* golden
fixtures — in three atomic, individually-verified commits ... per maintainer ruling D-04.
...
`internal/migrate/` — MISSING (deleted, as expected): confirmed via `test -d internal/migrate`
-> false
```
Apply the same verification shape to the 6 JSON files: `git rm` each, then assert absence with
`test -f tools/bench/headtohead-<name>.json` → false (or `git ls-files tools/bench/headtohead-*.json`
→ empty) rather than trusting the `git rm` exit code alone.

### `tools/bench/runner/main.go` (CLI, `-mode headtohead` → `-mode publish`, D-07)

**Analog:** Phase 5's `codegraph migrate` command removal — same "delete a command/mode, its
flags, its dedicated helper functions" shape as `internal/cli/migrate.go` + `internal/migrate/`
(`05-01-SUMMARY.md:118`).

**Concrete subtraction points (already file:line-verified in RESEARCH.md Pattern 2,
`06-RESEARCH.md:217-233` — reuse directly, do not re-grep):**
```go
// Delete:
case "headtohead":                              // main.go:161-162
func runHeadToHead(...) { ... }                  // main.go:306-340
fs.StringVar(&cfg.tsBinary, "ts-binary", ...)     // main.go:178
cfg.tsBinary field                                // main.go:129
func resolveTSBinary(...) { ... }                 // main.go:210-218
const macOSHomebrewTSBinary = ...                 // main.go:109-116
// (cfg.mode == "headtohead" && cfg.tsBinary == "") branch  // main.go:156-158

// Keep, adapt:
func measureSubject(name, binary string, entry, srcDir, scratchRoot, runner) (Metrics, error)
  // already single-subject-shaped (main.go:342-433) — call once with "go" only
```
The `publish` mode skeleton is given verbatim in RESEARCH.md's Code Examples section
(`06-RESEARCH.md:354-397`) — copy that shape directly rather than re-deriving.

### `tools/bench/runner/main_test.go` (test, subtractive, D-07 fallout)

**Analog:** Phase 5's test-fallout handling in `05-01-SUMMARY.md` — tests asserting removed
surface are deleted alongside the surface, not weakened or skipped.

**Exact targets (RESEARCH.md Pattern 2, `06-RESEARCH.md:233`):**
```go
// TestParseFlags_OverridesApply — delete the -ts-binary assertion at :482-487
// TestResolveTSBinary_FindsOnPath (:496-511) — delete whole function
// TestResolveTSBinary_EmptyWhenNotFound (:512-533) — delete whole function
```

### `tools/bench/realcorpus/manifest.go` + `manifest_test.go` (D-09/D-10)

**Analog:** Phase 4's identifier-family sweep for FLAG-PARITY (`04-02-SUMMARY.md:20-30`) —
same word-boundary reference-table discipline, applied here to `colbymchenry-codegraph`,
`SiblingDir: codegraph-ts`, and comparison prose in package/field doc comments (lines 2-15, 39,
54, 82, 157 per D-10).

**Test analog:** the `manifest_test.go:72-77` assertion (`06-RESEARCH.md:233`) that the entry
exists must be deleted, mirroring Phase 5's pattern of deleting tests for removed manifest
entries rather than leaving a stale RED:
```go
tscg, ok := byName["colbymchenry-codegraph"]
if !ok {
    t.Fatal("manifest missing colbymchenry-codegraph entry")
}
```

### `internal/corpora/manifest.go` (D-10, doc-comment + `Manifest.Note` review)

**Analog:** Phase 4's cross-file pointer update (`04-02-SUMMARY.md:24`, `internal/cli/man.go`
row: `"the FLAG-PARITY divergence footprint stays one documented hidden command" → "its
divergence footprint stays documented in this comment"`) — same shape: a comment that
cross-references a renamed/reworded target gets its wording updated to match, not rewritten
wholesale.

**Scope discipline — Pitfall 3 (`06-RESEARCH.md:346-350`):** `corpora/manifest.json`'s `Note`
field already reads on its own terms (names `tools/bench/realcorpus/manifest.go` as "this
repository's OTHER pinned-corpus manifest," no TS/comparison language). Treat this as a
**verification pass**, not a rewrite — do not budget multi-paragraph edits here.

### `internal/bench/rss.go` (doc-comment reword only)

**Analog:** Phase 5's `behavioral_test.go` "NO change to any string literal ... comments only"
discipline (`05-08-PLAN.md` prohibitions block, quoted above).

**Exact target (`06-RESEARCH.md:285`):**
```go
// internal/bench/rss.go:1-9 doc comment currently reads:
// "...which cannot be compared fairly against the TS Node process (D-05)"
```
Reword to describe the OS-level RSS measurement on its own terms; no logic change.

### `internal/bench/regression.go` (D-08, transient mutation only — behavior UNCHANGED)

**Analog:** `.planning/phases/03-non-vacuity-proof-unconditional-ci-execution/03-MUTATION-LOG.md`
— the FIXT-07 five-step protocol, reused verbatim per D-08.

**Header pattern to copy (`03-MUTATION-LOG.md:1-19`):**
```markdown
# 06-MUTATION-LOG — FIXT-07 mutation rehearsal (BENCH-02)
**Rehearsal suite:** internal/bench/regression_test.go (21 subtests, unchanged by this
rehearsal)
## Pre-mutation cleanliness gate (review finding)
Before every tracked-file mutation AND revert, `git diff --quiet -- <file>` was asserted.
```

**Concrete command sequence (RESEARCH.md Code Examples, `06-RESEARCH.md:399-427` — copy
directly):**
```bash
git diff --quiet -- internal/bench/regression.go   # step 1: must exit 0
# step 2: mutate DefaultThroughputTolerance 0.10 -> 1.0 (regression.go:9)
git diff --stat -- internal/bench/regression.go     # step 3: must be non-empty
go test -count=1 ./internal/bench/... -run TestCheckRegression -v   # step 4: observe RED
git checkout -- internal/bench/regression.go         # step 5: revert
git diff --stat -- internal/bench/regression.go     # step 6: must be empty; re-run green
```
**Pitfall to avoid (RESEARCH.md Pitfall 2, `06-RESEARCH.md:340-344`):** do not mutate
`baseline.json` or run on a mismatched runner class — `CheckRegression`'s category-error guards
(`regression.go:49-103`) fire before the numeric comparison and would produce a
platform-mismatch RED that proves nothing about BENCH-02's actual claim.

### `.github/workflows/bench.yml` (D-05/D-06, job deleted + fresh job authored)

**Analog A:** Phase 5's command-removal precedent (delete cleanly, no rename-and-strip).
**Analog B:** Phase 4's workflow/template reference sweep (`04-02-SUMMARY.md:29`):
```
.github/workflows/auto-close-unsolicited-prs.yml | docs/FLAG-PARITY.md or .planning/ ->
.planning/
```
— same shape as updating `workflow_dispatch.inputs.job`'s choice list (`headtohead` → `publish`)
and any `if: inputs.job == 'headtohead'` conditionals (`bench.yml:99`).

**D-06's four carry-forward properties, already located (RESEARCH.md Pattern 3,
`06-RESEARCH.md:235-254` — copy the table directly into the plan):**
1. Runner pinning: `runs-on: namespace-profile-linux-amd64-4x8` (`:98`) +
   `CODEGRAPH_BENCH_RUNNER: namespace-profile-linux-amd64-4x8` (`:100-101`)
2. No-Taskfile exception inherited context (comment block `:160-171`)
3. Non-blocking contract, job header comment (`:129-135`)
4. Artifact upload `if-no-files-found: error` + job summary (`:143-158`)

**Deletions:** Node/npm setup (`:117-120`), `npm install -g @colbymchenry/codegraph@1.3.1`
(`:122-123`), `-ts-binary` flag on the `go run` invocation (`:140`).

### `06-MUTATION-LOG.md` (new artifact, D-08)

**Analog:** `.planning/phases/03-non-vacuity-proof-unconditional-ci-execution/03-MUTATION-LOG.md`
— full structure to copy (header, D-01-style "re-mutate vs cite" call if applicable, per-family
mutation/RED/revert/green blocks). See excerpt above.

### `06-MEMORY-SWEEP.md` (new artifact, D-15)

**No analog found in this repository.** No prior phase has swept an external durable-memory
store (engram). RESEARCH.md Pattern 6 (`06-RESEARCH.md:291-304`) provides an *assumed*, LOW
confidence workflow shape (enumerate via `list_memory`/`list_rules`, classify present-tense vs.
past-tense per D-13, `supersede_memory`, record every `short_id` + verdict + reason) — this must
be treated as a starting hypothesis to verify live against actual `mcp__engram__*` tool
signatures at plan-execution time, not a proven pattern. D-16's fail-loud precondition applies:
if the engram store cannot be enumerated, the plan must fail loud rather than record a sweep as
complete.

**Named precedent to start from:** memory `myywc0y9vm` (`06-CONTEXT.md:147-148`) — `get_memory`
this specific ID first as a concrete anchor before an unscoped `list_memory` enumeration.

### `.claude/CLAUDE.md`, `.planning/PROJECT.md`, `.planning/STATE.md` (D-12, framing reword)

**Analog:** Phase 4/5's identifier-family sweep discipline, applied to prose rather than code
identifiers — same "locate every occurrence, reword on the file's own terms, verify zero
residue" shape.

**Exact targets (RESEARCH.md Runtime State Inventory, `06-RESEARCH.md:328`):**
```
.claude/CLAUDE.md:1-19 — "Originated as a ground-up Go rewrite of [CodeGraph]
(MIT-licensed; ported with attribution)." + Core Value line
.planning/PROJECT.md:1-9 — identical opener + Core Value at :5,7,9
.planning/STATE.md:26 — "An agent user can uninstall TS CodeGraph, install the Go binary,
migrate their indexes, and everything works the same or better..."
```
**Compound defect note:** the STATE.md line also names `migrate`, a capability removed in
Phase 5 (CODE-03) — this is a two-part defect (framing + stale capability reference), not just
framing.

## Shared Patterns

### The positive-controlled multiline census (D-04) — applies to ALL prose-touching files above
**Source:** `.planning/phases/05-process-ci-in-tree-sweep/05-08-PLAN.md` (Pattern 5,
`06-RESEARCH.md:272-289,429-455` for the runnable script).
**Apply to:** `docs/BENCHMARKS.md`, `tools/bench/BASELINE.md`, `tools/bench/runner/main.go`,
`tools/bench/realcorpus/manifest.go`, `internal/bench/rss.go`, `internal/corpora/manifest.go`,
`.github/workflows/bench.yml`.
```bash
S="[[:space:]/*]+"
F="${TMPDIR:-/tmp}/bench-poscontrol.txt"
printf "%b\n" "// the installed comparison binary (TS codegraph@1.3.1) is invoked here." \
  "// this mirrors TS's own head-to-head harness." > "$F"
pc=$(rg -U -o -i -e "\bTS\b" -e "\b(mirrors|matches)${S}TS" "$F" | wc -l | tr -d ' ')
lb=$(rg    -o -i -e "\bTS\b" -e "\b(mirrors|matches)${S}TS" "$F" | wc -l | tr -d ' ')
test "$pc" -gt "$lb" && test "$pc" -gt 0   # prove instrument works before trusting a zero
n=$(rg -U -o -i -e "\bTS\b" -e "\bcolbymchenry\b" -e "\bcodegraph-ts\b" <declared files...> \
  | wc -l | tr -d ' ')
echo "TOTAL=$n"   # target: 0
```
**Never use `rg -c`** (counts lines, not matches) — this repo's own history (05-04, 05-05)
shows a line-based census missing 10 then 34 wrapped occurrences.

### The FIXT-07 mutation-rehearsal protocol
**Source:** `03-MUTATION-LOG.md`.
**Apply to:** `internal/bench/regression.go` (D-08) only, in this phase.

### Identifier/framing-family sweep with a reference table
**Source:** `04-02-SUMMARY.md`'s reference-sweep table shape.
**Apply to:** any file whose fix is "reword this cross-reference to match the renamed/reworded
target" — `internal/corpora/manifest.go`'s `Manifest.Note`, `.github/workflows/bench.yml`'s
`workflow_dispatch.inputs.job` choice list.

### Deletion-with-git-history-preserved, verified by absence
**Source:** `05-01-SUMMARY.md` (`internal/migrate/` deletion).
**Apply to:** `tools/bench/headtohead-*.json` (D-01). Verify with `git ls-files` /
`test -f` returning false, not merely a clean `git rm` exit code.

## No Analog Found

| File | Role | Data Flow | Reason |
|---|---|---|---|
| `06-MEMORY-SWEEP.md` | doc (record) | batch | No prior phase has swept an external durable-memory store (engram); RESEARCH.md Pattern 6 is explicitly LOW-confidence/unverified and must be re-confirmed live against actual tool signatures at execution time (D-16) |
| `mcp__engram__*` interaction shape itself (not a file, but the workflow the MEM plan will encode) | — | event-driven | No `.mcp.json` registration in this repo; no prior in-repo tool-call precedent exists to copy from — the planner must treat Pattern 6 in RESEARCH.md as a hypothesis, not a template |

## Metadata

**Analog search scope:** `.planning/phases/03-*/`, `.planning/phases/04-*/`,
`.planning/phases/05-*/` (SUMMARY/PLAN/MUTATION-LOG files), plus direct reads of
`06-CONTEXT.md` and `06-RESEARCH.md` (which already carry file:line grounding for nearly the
entire in-repo surface — this phase's research was unusually thorough, so pattern-mapping work
concentrated on attaching *removal-shaped* precedent rather than re-deriving analogs from
scratch).
**Files scanned:** 06-CONTEXT.md, 06-RESEARCH.md (full), 03-MUTATION-LOG.md (header + protocol
shape), 04-02-SUMMARY.md, 05-01-SUMMARY.md, 05-08-PLAN.md (must_haves/prohibitions blocks).
**Pattern extraction date:** 2026-08-16
