---
phase: 06-claude-code-pretooluse-nudge
plan: 02
subsystem: agents
tags: [claude-code, hooks, pretooluse, nudge, classifier, corpora, drift-guard]
status: complete

requires:
  - phase: 06-claude-code-pretooluse-nudge
    provides: 06-01 internal/nudge core (Tool, Qualifies stub, pinned Text) and the hook adapter
provides:
  - Qualifies with D-02's first-word shell rule (assignment skip, parse doubt silent) and D-03's Read exclusions
  - D-15 harness-neutral corpora internal/nudge/testdata/{true,false}-positives.json
  - TestQualifiesCorpora, TestCorporaShape, TestQualifiesShellFirstWord, TestQualifiesReadNonCodeExtensions
  - D-14 drift guards over nudge.Text (NamesOnlyRealTools, CarriesNoUnpinnedFacts, IsFactualOneLiner)
  - 06-MUTATION-LOG.md Family (b1)-(b3)
affects: [06-03, 06-06, 06-07, CODEX-05]

actuals:
  tokens: 6900
  tasks: 2
  commits: 4
plan_head_before: a97234d12e69df86443378281fe90bee3e41ba25

tech-stack:
  added: []
  patterns:
    - "JSON-fixture table test: one t.Run per corpus row, disagreeing rows must carry a note"
    - "Drift guards applied to a Go constant directly, no file read"

key-files:
  created:
    - internal/nudge/classify_test.go
    - internal/nudge/testdata/true-positives.json
    - internal/nudge/testdata/false-positives.json
  modified:
    - internal/nudge/classify.go
    - internal/mcp/skill_claims_drift_test.go
    - .planning/phases/06-claude-code-pretooluse-nudge/06-MUTATION-LOG.md

key-decisions:
  - "Phase 6 06-02: TestCorporaShape's logged fire rates are measured through Qualifies, not counted from the rows' want fields"

patterns-established:
  - "Corpus rows are {tool, input, want, note}; want records what Qualifies does, and an accepted miss or false positive says why in note"

requirements-completed: []

coverage:
  - id: D1
    description: "shell qualifies iff the first word after NAME=value assignments is grep/egrep/fgrep/rg/find; parse doubt is silent"
    requirement: NUDGE-05
    verification:
      - kind: unit
        ref: "internal/nudge/classify_test.go#TestQualifiesShellFirstWord (20 cases), TestQualifiesCorpora"
        status: pass
  - id: D2
    description: "read qualifies unless empty or .md/.json/.yaml/.yml/.toml/.txt/.lock (case-insensitive)"
    requirement: NUDGE-05
    verification:
      - kind: unit
        ref: "internal/nudge/classify_test.go#TestQualifiesReadNonCodeExtensions (15 cases)"
        status: pass
  - id: D3
    description: "D-15 corpora (18 true-positive rows, 22 false-positive rows) with shape floor and measured rates"
    requirement: NUDGE-05
    verification:
      - kind: unit
        ref: "internal/nudge/classify_test.go#TestQualifiesCorpora (40 subtests), TestCorporaShape"
        status: pass
  - id: D4
    description: "nudge.Text names only real tools, states no unpinned fact, and is a factual one-liner"
    requirement: NUDGE-03
    verification:
      - kind: unit
        ref: "internal/mcp/skill_claims_drift_test.go#TestPreToolUseNudgeTextNamesOnlyRealTools, TestPreToolUseNudgeTextCarriesNoUnpinnedFacts, TestPreToolUseNudgeTextIsFactualOneLiner"
        status: pass

duration: 5min
completed: 2026-09-19
---

# Phase 6 Plan 02: Nudge classifier and corpora Summary

**`nudge.Qualifies` now fires on a shell command only when its first word after `NAME=value` assignments is grep/egrep/fgrep/rg/find, and on Read unless the file is an obvious non-code type. Two harness-neutral corpora pin that behaviour row by row, and three drift guards hold `nudge.Text` to the tool roster and to factual wording.**

## Performance

- Duration: about 5 min
- Started: 2026-09-19T09:18:20Z
- Tasks: 2/2
- Files: 6 (3 created, 3 modified)

## Accomplishments

- D-02: `shellQualifies` splits on whitespace, skips leading assignment words (package-level regexp), treats a quote, backtick, `$` or backslash in an assignment as parse doubt, and requires exact first-word membership. D-03: Read excludes empty paths and seven extensions, lower-cased.
- D-15: `true-positives.json` has 18 rows covering all four tools, 16 with want=true. `false-positives.json` has 22 rows (shell, read, grep, and the `webfetch` unknown-tool row), 19 with want=false. All 5 disagreeing rows carry notes.
- D-14: three guards over the constant, with no file read. `internal/nudge/text.go` is byte-identical to its 06-01 commit.

## Measured corpus rates (TestCorporaShape)

```
classify_test.go:110: true-positive corpus: 16/18 fire; false-positive corpus: 3/22 fire (accepted, noted)
```

The 2 non-firing true positives are the noted accepted misses (`git grep`, `cd … && grep`). The 3 firing false positives are the noted accepted ones (`grep -c ERROR app.log`, `find /tmp -mtime +7 -delete`, Grep tool `TODO`). This is NUDGE-05's unit-level half; plan 06-06 measures the live fire rate.

## Task Commits

1. **Task 1 RED:** `a1bb45c1` test(06-02): add D-15 corpora and failing classifier tests
2. **Task 1 GREEN:** `f7ee5e27` feat(06-02): D-02 first-word shell rule and D-03 Read exclusions for the nudge classifier
3. **Task 2 guards:** `2111fa58` test(06-02): extend nudge-text drift guards to the PreToolUse constant
4. **Task 2 Family (b):** `d46dbb44` docs(06-02): record Family (b) classifier and nudge-text RED demonstrations

## TDD RED evidence (Task 1, commit a1bb45c1, run against the 06-01 stub)

`GOTOOLCHAIN=go1.26.6 go test ./internal/nudge/ -count=1 -v`, exit 1. The `--- FAIL:` lines:

```
--- FAIL: TestQualifiesCorpora (0.00s)
    --- FAIL: TestQualifiesCorpora/true-positives/0 (0.00s)
    --- FAIL: TestQualifiesCorpora/true-positives/1 (0.00s)
    --- FAIL: TestQualifiesCorpora/true-positives/2 (0.00s)
    --- FAIL: TestQualifiesCorpora/true-positives/3 (0.00s)
    --- FAIL: TestQualifiesCorpora/true-positives/4 (0.00s)
    --- FAIL: TestQualifiesCorpora/true-positives/5 (0.00s)
    --- FAIL: TestQualifiesCorpora/true-positives/6 (0.00s)
    --- FAIL: TestQualifiesCorpora/true-positives/7 (0.00s)
    --- FAIL: TestQualifiesCorpora/true-positives/8 (0.00s)
    --- FAIL: TestQualifiesCorpora/true-positives/15 (0.00s)
    --- FAIL: TestQualifiesCorpora/true-positives/16 (0.00s)
    --- FAIL: TestQualifiesCorpora/true-positives/17 (0.00s)
    --- FAIL: TestQualifiesCorpora/false-positives/9 (0.00s)
    --- FAIL: TestQualifiesCorpora/false-positives/10 (0.00s)
--- FAIL: TestQualifiesShellFirstWord (0.00s)
    --- FAIL: TestQualifiesShellFirstWord/"grep_x" (0.00s)
    --- FAIL: TestQualifiesShellFirstWord/"__rg_x" (0.00s)
    --- FAIL: TestQualifiesShellFirstWord/"find_._-name_x" (0.00s)
    --- FAIL: TestQualifiesShellFirstWord/"egrep_x" (0.00s)
    --- FAIL: TestQualifiesShellFirstWord/"fgrep_x" (0.00s)
    --- FAIL: TestQualifiesShellFirstWord/"LC_ALL=C_grep_x" (0.00s)
    --- FAIL: TestQualifiesShellFirstWord/"A=1_B=2_rg_x" (0.00s)
    --- FAIL: TestQualifiesShellFirstWord/"grep_x_|_head" (0.00s)
--- FAIL: TestQualifiesReadNonCodeExtensions (0.00s)
    --- FAIL: TestQualifiesReadNonCodeExtensions/"/repo/a.go" (0.00s)
    --- FAIL: TestQualifiesReadNonCodeExtensions/"/repo/a.tsx" (0.00s)
    --- FAIL: TestQualifiesReadNonCodeExtensions/"/repo/a.py" (0.00s)
    --- FAIL: TestQualifiesReadNonCodeExtensions/"/repo/Makefile" (0.00s)
    --- FAIL: TestQualifiesReadNonCodeExtensions/"/repo/a.mdx" (0.00s)
    --- FAIL: TestQualifiesReadNonCodeExtensions/"/repo/a.json5" (0.00s)
FAIL	github.com/seanb4t/codegraph-go/internal/nudge	0.107s
```

Every failure is an assertion failure: the stub returned false for shell and read. The stub-era rate line read `true-positive corpus: 4/18 fire; false-positive corpus: 1/22 fire`.

## Family (b) outcome (06-MUTATION-LOG.md)

- **(b1)** A `strings.Contains` loop replaced the first-field test in `classify.go`. Result: `--- FAIL: TestQualifiesCorpora/false-positives/0` (the `git log --oneline | grep fix` row) plus false-positives/1,2,3,5,6,7 and true-positives/9,10 (the accepted misses). Exit 1. Gates: pre 0, pre-revert 1, post-revert 0. The green control returned `ok`.
- **(b2)** Deleting the assignment-skip loop gave exactly one failure, `--- FAIL: TestQualifiesCorpora/true-positives/5` (`LC_ALL=C grep -rn handleRequest internal`). Exit 1. Gates 0/1/0. The green control returned `ok`.
- **(b3)** Changing `codegraph_explore (` to `codegraph_explorer (` in `text.go` gave `--- FAIL: TestPreToolUseNudgeTextNamesOnlyRealTools` (`nudge.Text names codegraph_explorer, which is not a member of allToolNames()`). Exit 1. Gates 0/1/0. The green control returned `ok`. The plan's Task 2 `<verify>` block planted and reverted (b3) a second time, and that run also passed.

## Verification

- Task 1 `<verify>`: build, gofmt, and vet clean. 4 named top-level PASS lines, 18/22 corpus row subtests PASS, rate line present. `TestHookPreToolUse_*` returned `ok`.
- Task 2 `<verify>`: printed `VERIFY2-PASS` (5 named PASS lines, b3 RED and byte-clean revert, log sections present). `internal/nudge` and `internal/mcp` both returned `ok`.
- Plan verification: `GOTOOLCHAIN=go1.26.6 go test ./internal/nudge/ ./internal/mcp/ ./internal/cli/ -count=1` returned `ok` for all three (cli 19.5 s). No `internal/cli` flake occurred.
- Acceptance checks: the jq key/note checks passed; `Qualifies` compiles no regexp; the new drift tests contain no `os.ReadFile`; `nudge.Text` appears 16 times (at least 3 required); `text.go` is unchanged since `feat(06-01)`; no 06-02 commit contains `[ci skip]` or `[skip ci]`. `golangci-lint` on `internal/nudge` and `internal/mcp` reported 0 issues.

## Decisions Made

- The fire rates logged by `TestCorporaShape` are measured by calling `Qualifies`, not counted from the rows' `want` fields. The logged number is therefore an observation, not a restatement of the fixture.

## Deviations from Plan

None. The plan executed as written.

Housekeeping: `.git/gsd-plan-head-before-06-02` held a stale SHA (`48a45e23`, written 2026-09-07 by an earlier milestone's 06-02). It was rewritten to the current plan base `a97234d1` before the first commit, so `commits: 4` counts from the correct base.

## Known Stubs

None. The 06-01 stub branches for shell and read are now implemented.

## Threat Flags

None. The plan's register covers every new surface (T-06-09, T-06-10, T-06-11).

## Self-Check: PASSED
