---
phase: 04-cli-glow-up
plan: 08
subsystem: cli
tags: [cobra, groups, help, lipgloss, present, docs-cli, tdd, phase-close]

# Dependency graph
requires:
  - phase: 04-02
    provides: "fang declined (04-FANG-VERDICT.md) — help must be hand-rolled per D-14, not fang-wrapped"
  - phase: 04-03
    provides: "resolveColor(cmd), colorMode.Writer, present.Palette/NewPalette — the resolver and palette installHelpFunc/RenderHelp consume"
  - phase: 04-07
    provides: "every verb styled end to end — this plan's <verb> --help styling check needs the final, fully-styled tree"
provides:
  - "internal/cli/root.go: groupQuery/groupBuild/groupAgents/groupMaintenance consts, one commandGroups name->id table, root.AddGroup in D-13 order, SetHelpCommandGroupID/SetCompletionCommandGroupID, installHelpFunc(root) — the D-14 hand-rolled help wiring"
  - "internal/cli/present/help.go: RenderHelp(c, pal, w) — walks cobra's own group/flag metadata, styled chrome only, stock cobra template on a non-styled invocation"
  - "internal/cli/cli_reference_test.go: TestEveryCommandHasGroupID, commandGroupIDs — the CLI-06 positive-counted guard"
  - "internal/cli/present/help_test.go: TestRenderHelpGroupsInOrder — the D-14 guard over a synthetic cobra tree (present cannot import cli)"
  - "docs/CLI-REFERENCE.md regenerated through the drift gate (--color-only diff); .planning/PROJECT.md's Key Decisions row for the fang verdict"
affects: []

# Actuals (#2632)
actuals:
  tokens: 11208
  tasks: 3
  commits: 7
  plan_head_before: 4ea19a3911572438dbed10a23519683dcd5b79ea

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "One commandGroups name->id table applied to root's direct children in a single loop after AddCommand, rather than setting GroupID inline at each new<Verb>Cmd() constructor — keeps the D-13 assignment auditable in one place"
    - "installHelpFunc captures cobra's own HelpFunc() (the stock template) BEFORE calling SetHelpFunc, so the styled branch's fallback is always the real stock behavior, never a hand-maintained duplicate of it — and cobra's own HelpFunc() parent-walk propagates the same func to every subcommand with no per-verb wiring"
    - "present.RenderHelp cannot reuse internal/cli's real NewRootCmd() in its own test (present cannot import cli — cli already imports present) — help_test.go instead builds a small synthetic cobra tree that mirrors root.go's exact D-13 shape (same group IDs/titles/order), the same pattern documentation-generation guards in this codebase already use when the real tree isn't reachable"

key-files:
  created:
    - internal/cli/present/help.go
    - internal/cli/present/help_test.go
  modified:
    - internal/cli/root.go
    - internal/cli/cli_reference_test.go
    - docs/CLI-REFERENCE.md
    - .planning/PROJECT.md
    - .planning/phases/04-cli-glow-up/04-MUTATION-LOG.md

key-decisions:
  - "Fang was declined (04-02 verdict) so Task 2 took the present.RenderHelp/installHelpFunc branch exactly as the plan specified for that outcome — no fang code, no applyColorFlagToProcessEnv, present/help.go exists."
  - "TestEveryCommandHasGroupID's visibility predicate special-cases cobra's own default help command: cobra.Command.IsAvailableCommand() deliberately excludes it (command.go:1612's `c.Parent().helpCommand == c` check) even though cobra's own usage template still lists it as visible (`sub.IsAvailableCommand() || sub == c.helpCommand`, command.go:1285/1378/1996). Citing this split (not testing cobra's rendering) is D-00-compliant; the guard's own assertion — that OUR help command carries GroupID \"maintenance\" — is unaffected."
  - "docs/CLI-REFERENCE.md's regeneration produced 2 removed lines beyond pure --color additions: root's pre-existing -h/--help and -v/--version option lines re-emit with wider column padding, because pflag's FlagUsages() column-width computation widens to fit the newly-added, longer --color flag name in the same LocalFlags() block. Wording is byte-identical; only whitespace changed. Reviewed and accepted as an unavoidable, deterministic side effect of the sanctioned generator (task docs:cli) — not a hand edit, and task docs:cli:drift confirms it is exactly what a fresh regeneration produces."
  - "The 04-FANG-VERDICT.md 'Key Decisions row (for phase close)' table has a 2-column header (Decision | Rationale) but its one data row is a single un-split cell — PROJECT.md's Key Decisions table has 3 columns (Decision | Rationale | Outcome). The verbatim text was partitioned across the three columns at existing sentence boundaries (no wording added, removed, or paraphrased) to fit the table's established shape."
  - "A standalone fix(04-08): commit was created for the TestEveryCommandHasGroupID visibility-predicate correction, found during Task 1's own GREEN verification but not folded into the feat(04-08) commit at the time. This is a valid conventional-commit type, correctly scoped to (04-08), but is not one of the five types (test|feat|docs|refactor|chore) the plan's own commit-history audit regex enumerates — flagged explicitly in the audit table below rather than silently passed."

patterns-established: []

requirements-completed: [CLI-06, CLI-01, CLI-02, CLI-04]

coverage:
  - id: D1
    description: "codegraph --help groups commands into the four D-13 cobra.Groups (query/build/agents/maintenance) in registration order, membership exactly matching D-13's table, help/completion filed under Maintenance, hidden man/query/unlock stay groupless — TestEveryCommandHasGroupID RED-first (0 groups before the feature) and RED again against a planted ungrouped command (Family (e), names telemetry)"
    requirement: "CLI-06"
    verification:
      - kind: unit
        ref: "internal/cli/cli_reference_test.go#TestEveryCommandHasGroupID"
        status: pass
      - kind: other
        ref: ".planning/phases/04-cli-glow-up/04-MUTATION-LOG.md#Family-(e)"
        status: pass
      - kind: other
        ref: "real binary: codegraph --help contains all four titles in order, help/completion listed under Maintenance:"
        status: pass
    human_judgment: false
  - id: D2
    description: "Help is styled per the recorded fang verdict (declined -> hand-rolled present.RenderHelp behind resolveColor(cmd)): <verb> --help inherits root's styling via cobra's HelpFunc() propagation; non-styled invocations (piped, NO_COLOR, --color=never) fall back to byte-identical stock cobra help"
    requirement: "CLI-01"
    verification:
      - kind: unit
        ref: "internal/cli/present/help_test.go#TestRenderHelpGroupsInOrder"
        status: pass
      - kind: other
        ref: "real binary: --help/status --help on a pipe are ESC-free with the four group titles present; --color=always --help (root and status) carries ESC; --color=never --help is ESC-free"
        status: pass
    human_judgment: false
  - id: D3
    description: "docs/CLI-REFERENCE.md regenerated ONCE through task docs:cli as a reviewed --color-only diff (plus an unavoidable column-realignment side effect on 2 pre-existing lines, documented); task docs:cli:drift and TestEveryRegisteredFlagIsAccountedFor green with no new allowlist entry; the scheduled RED window plan 03 opened is closed"
    requirement: "CLI-02"
    verification:
      - kind: other
        ref: "task docs:cli:drift (compared 1 generated file, byte-identical to a fresh regeneration)"
        status: pass
      - kind: unit
        ref: "internal/cli/cli_reference_test.go#TestEveryRegisteredFlagIsAccountedFor (37 commands, 111 via reference, 3 via allowlist)"
        status: pass
    human_judgment: false
  - id: D4
    description: "PROJECT.md's Key Decisions table gains exactly one new row recording the fang verdict verbatim, no new heading; full phase gates green at HEAD (build, gofmt over internal/cmd/tools, vet, test:unit, test:golden, integration, wireoracle, present archtest, -race over cli+present, govulncheck zero-new); internal/query, internal/mcp, testdata/wireoracle, testdata/golden untouched all phase; tree clean after the final commit"
    requirement: "CLI-04"
    verification:
      - kind: other
        ref: "git diff .planning/PROJECT.md: exactly one +| line, zero +# lines"
        status: pass
      - kind: other
        ref: "full phase-gate run (see Phase Gate Results below)"
        status: pass
    human_judgment: false
  - id: D5
    description: "Human UAT (CLI-04 palette readability, CLI-02 terminal-capability rendering, --help/<verb> --help consistency, --color=always|less -R vs --color=never) harvested at end of phase from every <human-check> across plans 03 and 08"
    verification: []
    human_judgment: true
    rationale: "D-06/CLI-04's backstop truth explicitly names this a human UAT item — colour legibility, downsampling fidelity and OSC-11 terminal behavior are not unit-testable per D-00 (never test lipgloss/colorprofile/cobra's own rendering). Harvested at end-of-phase per workflow.human_verify_mode=end-of-phase from this task's <verify><human-check> block."

# Metrics
duration: 20min
completed: 2026-09-18
status: complete
---

# Phase 4 Plan 8: Command Groups, Hand-Rolled Help, Docs Regen — Phase Close Summary

**`codegraph --help` now groups all 24 visible commands into D-13's four titled sections via `cobra.Group`, `<verb> --help` renders through a hand-rolled `present.RenderHelp` behind the same colour resolver every verb uses (fang was declined), `docs/CLI-REFERENCE.md` is regenerated through the drift gate as a reviewed `--color`-only diff, and every full-phase gate is green at HEAD.**

## Performance

- **Duration:** 20 min
- **Started:** 2026-09-17T19:47:09-04:00
- **Completed:** 2026-09-17T20:07:10-04:00
- **Tasks:** 3
- **Files modified:** 7 (2 created, 5 modified)

## Accomplishments

- `internal/cli/root.go` gained four `group*` consts, a single 22-entry `commandGroups` name→ID table, `root.AddGroup(...)` registering the four D-13 titles in order, the `GroupID`-application loop over `root.Commands()`, and `SetHelpCommandGroupID`/`SetCompletionCommandGroupID` filing `help`/`completion` under Maintenance.
- `TestEveryCommandHasGroupID` (`internal/cli/cli_reference_test.go`) is the CLI-06 guard: RED on the ungrouped tree (`root.Groups() has 0 groups, want 4`), GREEN after implementation (24 visible commands inspected across 4 groups), and RED again against a planted ungrouped command (Family (e): removing `telemetry`'s map entry names `telemetry` explicitly in the failure).
- Real `codegraph --help` renders all four titles (`Query the graph:`, `Build the index:`, `Agents & serving:`, `Maintenance:`) in order, with `completion` and `help` both listed under `Maintenance:`.
- Since the fang verdict is **declined** (04-FANG-VERDICT.md), Task 2 took the hand-rolled branch: `present.RenderHelp(c, pal, w)` walks cobra's own command/group/flag metadata (Long/Short, Usage:, each group's titled section with its visible members, ungrouped visible commands under `Additional Commands:`, `Flags:`/`Global Flags:`, and the trailer) and `installHelpFunc(root)` captures cobra's stock `HelpFunc()` before installing the styled one, falling back to byte-identical stock help on any non-styled invocation.
- `TestRenderHelpGroupsInOrder` (`internal/cli/present/help_test.go`) proves this over a synthetic cobra tree mirroring root.go's exact D-13 shape (present cannot import `internal/cli` — that would be circular): RED against a no-op placeholder (no ESC byte, missing every title), GREEN after the real implementation.
- Real binary confirms: `--help`/`status --help` on a pipe are ESC-free stock text carrying the four group titles; `--color=always --help` and `status --color=always --help` carry ESC bytes and strip back to containing "Query the graph:"; `--color=never --help` is ESC-free.
- `docs/CLI-REFERENCE.md` regenerated once via `task docs:cli`: 201 insertions / 2 deletions, every insertion a `--color string` option line (root's Options block, every subcommand's "Options inherited from parent commands" block) or its blank-line/header neighbor; the 2 deletions are root's pre-existing `-h, --help`/`-v, --version` lines re-emitted with wider column padding (pflag's own alignment computation, triggered by the longer `--color` flag name in the same block) — same wording, whitespace only. `task docs:cli:drift` is green; `TestEveryRegisteredFlagIsAccountedFor` reports 37 commands walked, 111 accepted via the reference, 3 via the (unchanged) allowlist.
- `.planning/PROJECT.md`'s Key Decisions table gained exactly one new row recording the 04-02 fang verdict, verbatim (partitioned across the table's Decision/Rationale/Outcome columns at existing sentence boundaries, no new heading).
- Every full-phase gate is green at HEAD: build, gofmt (scoped to `internal/ cmd/ tools/` — see Deviations for one pre-existing, out-of-scope `test/tmux` finding), vet, `task test:unit`, `task test:golden`, `test/integration`, `test/wireoracle`, `internal/cli/present/archtest`, `-race` over `internal/cli`+`internal/cli/present`, `govulncheck` (identical to the 04-02 baseline: 0 code-affecting, 3 package-level, 3 module-level, zero new). `internal/query`, `internal/mcp`, `testdata/wireoracle`, `testdata/golden` are untouched since the phase's first commit; the tree is clean after the final commit.

## RED Evidence (pasted verbatim)

**Task 1** (`GOTOOLCHAIN=go1.26.6 go test ./internal/cli/ -count=1 -run 'TestEveryCommandHasGroupID$' -v`, against the ungrouped tree, before `root.AddGroup`/`commandGroups` existed):

```
cli_reference_test.go:324: root.Groups() has 0 groups, want 4 [query build agents maintenance]
--- FAIL: TestEveryCommandHasGroupID (0.00s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/cli	0.481s
FAIL
```

**Task 1, Family (e)** (`GOTOOLCHAIN=go1.26.6 go test ./internal/cli/ -count=1 -run 'TestEveryCommandHasGroupID$' -v`, against the GREEN tree with `telemetry`'s `commandGroups` entry planted-removed):

```
cli_reference_test.go:360: visible command "telemetry" has GroupID "", which is not one of the four registered groups [query build agents maintenance]
    cli_reference_test.go:378: inspected 24 visible commands across 4 groups
--- FAIL: TestEveryCommandHasGroupID (0.00s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/cli	0.459s
FAIL
exit=1
```

**Task 2** (`GOTOOLCHAIN=go1.26.6 go test ./internal/cli/present/ -count=1 -run 'TestRenderHelpGroupsInOrder$' -v`, against the no-op placeholder `RenderHelp`):

```
help_test.go:80: styled RenderHelp(root) output does not contain an ESC byte
    help_test.go:91: stripped RenderHelp(root) output missing group title "Query the graph:":
--- FAIL: TestRenderHelpGroupsInOrder (0.00s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/cli/present	0.214s
FAIL
```

## GREEN Results

```
$ GOTOOLCHAIN=go1.26.6 go test ./internal/cli/ -count=1 -run 'TestEveryCommandHasGroupID$' -v
    cli_reference_test.go:378: inspected 24 visible commands across 4 groups
--- PASS: TestEveryCommandHasGroupID (0.00s)
ok  	github.com/seanb4t/codegraph-go/internal/cli	0.417s
```

```
$ GOTOOLCHAIN=go1.26.6 go test ./internal/cli/present/ -count=1 -run 'TestRenderHelpGroupsInOrder$' -v
--- PASS: TestRenderHelpGroupsInOrder (0.00s)
ok  	github.com/seanb4t/codegraph-go/internal/cli/present	0.181s
```

## Reference Diff Stats (Task 3)

```
$ git diff --shortstat <before>..<after> -- docs/CLI-REFERENCE.md
 docs/CLI-REFERENCE.md | 203 ++++++++++++++++++++++++++++++++++++++++++++++--
 1 file changed, 201 insertions(+), 2 deletions(-)
```

Every one of the 201 added lines mentions `--color string`, is a blank-line neighbor, or is an added `### Options inherited from parent commands` header. The 2 removed lines are root's own pre-existing `-h, --help`/`-v, --version` option lines, re-emitted with wider column padding by pflag's alignment computation (same wording — see Deviations).

## Phase Gate Results (Task 3, tails pasted)

```
$ GOTOOLCHAIN=go1.26.6 task docs:cli:drift
docs:cli:drift: compared 1 generated file
docs:cli:drift: docs/CLI-REFERENCE.md byte-identical to a fresh regeneration (temporary file only — source tree untouched)

$ GOTOOLCHAIN=go1.26.6 go test ./internal/cli/ -count=1 -run 'TestEveryRegisteredFlagIsAccountedFor$|TestPlainGolden$|TestEveryCommandHasGroupID$|TestShortFlagsConsistent$' -v
    cli_reference_test.go:254: walked 37 commands (hidden included), inspected 114 flags: 111 accepted via ../../docs/CLI-REFERENCE.md, 3 accepted via testdata/cli-reference-allowlist.txt
    short_flags_test.go:73: inspected 20 (verb, flag) pairs across 9 query verbs
ok  	github.com/seanb4t/codegraph-go/internal/cli	2.443s

$ GOTOOLCHAIN=go1.26.6 go build ./...
(clean)

$ test -z "$(gofmt -l internal/ cmd/ tools/)"
(clean — see Deviations for the one pre-existing test/tmux finding outside this scope)

$ GOTOOLCHAIN=go1.26.6 go vet ./...
(clean)

$ GOTOOLCHAIN=go1.26.6 task test:unit
ok  	github.com/seanb4t/codegraph-go/internal/cli	(cached)
ok  	github.com/seanb4t/codegraph-go/internal/cli/archtest	(cached)
ok  	github.com/seanb4t/codegraph-go/internal/cli/present	(cached)
ok  	github.com/seanb4t/codegraph-go/internal/cli/present/archtest	(cached)
ok  	github.com/seanb4t/codegraph-go/internal/cli/tui	(cached)
... (all other packages ok)

$ GOTOOLCHAIN=go1.26.6 task test:golden
ok  	github.com/seanb4t/codegraph-go/testdata/golden	25.744s

$ GOTOOLCHAIN=go1.26.6 go test ./test/integration/... -count=1
ok  	github.com/seanb4t/codegraph-go/test/integration	9.860s

$ GOTOOLCHAIN=go1.26.6 go test ./test/wireoracle/... -count=1
ok  	github.com/seanb4t/codegraph-go/test/wireoracle	58.431s

$ GOTOOLCHAIN=go1.26.6 go test ./internal/cli/present/archtest/... -count=1
ok  	github.com/seanb4t/codegraph-go/internal/cli/present/archtest	0.963s

$ GOTOOLCHAIN=go1.26.6 go test -race -count=1 ./internal/cli/ ./internal/cli/present/
ok  	github.com/seanb4t/codegraph-go/internal/cli	29.137s
ok  	github.com/seanb4t/codegraph-go/internal/cli/present	3.163s

$ GOTOOLCHAIN=go1.26.6 govulncheck ./...
No vulnerabilities found.
Your code is affected by 0 vulnerabilities.
This scan also found 3 vulnerabilities in packages you import and 3
vulnerabilities in modules you require, but your code doesn't appear to call
these vulnerabilities.
(identical to the 04-02 baseline — zero new findings)

$ S=$(git log --format=%H -E --grep='^test\(04-01\): ' -1) && git diff --name-only "$S^..HEAD" -- internal/query internal/mcp testdata/wireoracle testdata/golden
(empty)

$ test -z "$(git status --porcelain)"
(clean, after the final commit)
```

## Phase Commit-History Audit (Task 3)

`git log --format=%s <phase-start>..HEAD` where `<phase-start>` is the `test(04-01):` commit (the first plan-scoped commit; earlier `docs(04)`/`docs(phase-4)` commits are discuss/research/plan-authoring workflow commits that predate any plan and legitimately use the bare `(04)`/`(phase-4)` scope):

| Check | Result |
|---|---|
| Every subject matches `^(test\|feat\|docs\|refactor\|chore)\(04-0[1-8]\)` | 2 exceptions, both explained below — no others |
| No `[ci skip]`/`skip ci` token anywhere in any commit message body | Confirmed — none found |
| Every `feat(04-NN)` for a TDD plan is preceded by at least one `test(04-NN)` | Confirmed for every TDD plan (03, 04, 05, 06, 08); 02 and 07 are `type: execute` (not TDD) and carry no test-first requirement |

**The 2 regex exceptions:**
1. `fix(04-08): TestEveryCommandHasGroupID's visibility predicate must special-case cobra's own help command` — a legitimate conventional-commit type (`fix`), correctly scoped to `(04-08)`, just not one of the five types (`test|feat|docs|refactor|chore`) this audit's own regex enumerates. See Deviations for why this commit exists as a standalone commit rather than folded into `feat(04-08)`.
2. `docs(phase-4): update tracking after wave 1` — a pre-existing orchestrator tracking commit from a prior session/wave, using `phase-4` instead of `04-NN` scope. Not authored by this plan's execution; reported here for completeness of the audit.

## Task Commits

Each task committed atomically (TDD: RED test commit precedes the GREEN feat commit):

1. **Task 1 (RED): failing GroupID coverage guard** — `f74dcafc` (test)
2. **Task 1 (GREEN): group the command tree into Query/Build/Agents/Maintenance** — `44ba22bb` (feat)
3. **Task 1 (docs): record Family (e) RED proof in the mutation log** — `ca77b10f` (docs)
4. **Task 2 (RED): failing RenderHelp group-order test** — `d60bc6a9` (test)
5. **Fix: TestEveryCommandHasGroupID's help-command visibility predicate** — `a73c1966` (fix — see Deviations)
6. **Task 2 (GREEN): styled help via present.RenderHelp/installHelpFunc** — `53c6c49a` (feat)
7. **Task 3: regenerate docs/CLI-REFERENCE.md; PROJECT.md Key Decisions row** — `d0fa12dd` (docs)

**Plan metadata:** committed alongside SUMMARY.md/STATE.md/ROADMAP.md/REQUIREMENTS.md in the metadata commit that follows this SUMMARY.

## Files Created/Modified

- `internal/cli/root.go` — `group*` consts, `commandGroups` table, `AddGroup`/`GroupID` loop, `SetHelpCommandGroupID`/`SetCompletionCommandGroupID`, `installHelpFunc`
- `internal/cli/cli_reference_test.go` — `TestEveryCommandHasGroupID`, `commandGroupIDs`
- `internal/cli/present/help.go` — `RenderHelp`, `renderCommandSections`, `writeCommandRows`, `writeFlagUsages`, `rpad`
- `internal/cli/present/help_test.go` — `TestRenderHelpGroupsInOrder`, `buildHelpTestTree`
- `docs/CLI-REFERENCE.md` — regenerated (the `--color` hunks plus the 2-line column realignment)
- `.planning/PROJECT.md` — one Key Decisions row (fang verdict)
- `.planning/phases/04-cli-glow-up/04-MUTATION-LOG.md` — Family (e) added, Summary table extended

## Decisions Made

See `key-decisions` in frontmatter for full rationale. Summary: took the DECLINED/hand-rolled branch of Task 2 (fang verdict was already settled at 04-02); special-cased cobra's own default help command in the GroupID guard's visibility predicate (citing, not testing, cobra's documented split); accepted the docs regeneration's 2-line column-realignment side effect as an unavoidable, deterministic consequence of `task docs:cli` rather than a hand edit; partitioned the fang verdict's 2-column prepared row text across PROJECT.md's 3-column table at existing sentence boundaries with no wording changed; and left a standalone `fix(04-08):` commit in the history rather than rewriting it into an earlier commit (no amend/rebase per project rules), flagging it explicitly in the commit-history audit instead.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] `TestEveryCommandHasGroupID`'s visibility predicate misclassified cobra's own default help command**
- **Found during:** Task 1's own GREEN verification, immediately after implementing `root.go`'s grouping
- **Issue:** `cobra.Command.IsAvailableCommand()` deliberately excludes the command's own parent's default help command (`command.go:1612`'s `c.Parent().helpCommand == c` check) — so the test's plain `IsAvailableCommand() && !IsAdditionalHelpTopicCommand()` visibility check classified `help` as hidden, even though `help` legitimately carries `GroupID: "maintenance"` (set via `SetHelpCommandGroupID`) and cobra's own usage template still lists it as a real, visible command (`sub.IsAvailableCommand() || sub == c.helpCommand`, `command.go:1285/1378/1996`).
- **Fix:** Added a `c.Name() == "help"` special case to the test's visibility predicate, citing cobra's own documented split in a comment (D-00: citing behavior, not testing it) rather than changing the guard's actual assertion (that OUR help command carries `GroupID: "maintenance"`).
- **Files modified:** `internal/cli/cli_reference_test.go`
- **Verification:** `TestEveryCommandHasGroupID` passes, inspecting 24 visible commands (was 23 before the fix, with `help` misclassified as hidden and its non-empty GroupID flagged as an error).
- **Committed in:** `a73c1966` (a standalone `fix(04-08):` commit — see below for why it is not folded into the preceding `feat(04-08)` commit).

**2. [Rule 1 - Bug, process note] The visibility-predicate fix was left uncommitted at Task 1's GREEN commit time**
- **Found during:** Preparing Task 2's changes, running `git status --short` before staging
- **Issue:** The fix above (deviation #1) was made and verified during Task 1's own GREEN verification loop, but was not staged/committed alongside `44ba22bb` (Task 1's GREEN `feat(04-08)` commit) — an oversight in this session's own commit discipline, not a defect in the shipped code (the fix was applied and working in the working tree throughout; only its commit was delayed).
- **Fix:** Committed the fix as a standalone `fix(04-08):` commit (`a73c1966`) before proceeding to Task 2, so Task 1's own acceptance criteria (a green `TestEveryCommandHasGroupID` at HEAD) hold at every commit from that point forward. Confirmed via `git stash`/re-test that HEAD *before* this fix commit genuinely fails the test (23 visible commands, "help" misclassified) — the fix commit is load-bearing, not cosmetic.
- **Files modified:** `internal/cli/cli_reference_test.go`
- **Verification:** `git stash && go test ... && git stash pop` round-trip confirmed the pre-fix state fails and the post-fix state (current tree) passes.
- **Committed in:** `a73c1966`

---

**Total deviations:** 2 auto-fixed (both Rule 1 — a test-predicate bug and its own commit-discipline correction; no production rendering or resolution logic changed beyond what the plan specified).
**Impact on plan:** No scope creep. Both fixes are confined to the test file the plan itself lists under Task 1's `<files>`. The commit-history audit above reports the resulting `fix(04-08):` commit honestly rather than silently reshaping history to fit the audit's literal regex.

## Issues Encountered

**One pre-existing, out-of-scope gofmt finding surfaced by the plan's own broader `gofmt -l internal/ cmd/ test/ tools/` check:** `test/tmux/poll_contract_test.go` has a gofmt-normalizable smart-quote character in a doc comment (`gofmt -d` wants to convert `''` to `”`), confirmed via `git log` to have last been touched in commit `3da59354` (v0.13.0's tmux e2e harness landing, PR #69) — well before this phase began and entirely unrelated to CLI/present work. Per the deviation-rule scope boundary ("only auto-fix issues DIRECTLY caused by the current task's changes"), this was NOT fixed. `gofmt -l internal/ cmd/ tools/` (the subset this plan actually touches) is clean; the phase-gate tail above records the scoped result.

## Authentication Gates

None.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- CLI-06 is complete: the command tree is grouped per D-13, guarded RED-first and RED against a planted regression.
- CLI-01's full verb list (styled through every plan 03–07) plus `<verb> --help` styling (this plan) are all live; `--color=always | less -R` shows colour, `--color=never` on a TTY is plain, per real-binary checks.
- `docs/CLI-REFERENCE.md` is current with `--color` and the phase's scheduled RED window (opened at plan 03) is closed.
- The fang verdict is recorded in `PROJECT.md`'s Key Decisions (D-04).
- Human UAT items (CLI-04 palette readability, CLI-02 terminal-capability rendering across `TERM=dumb`/16/256/truecolor and SSH/tmux, `--help`/`<verb> --help` consistency) are queued for end-of-phase harvesting from every `<human-check>` block across plans 03 and 08, per `workflow.human_verify_mode=end-of-phase` — not silently passed.
- This is the phase's final plan (wave 6, no plans depend on it). No blockers for phase verification/code review.

---
*Phase: 04-cli-glow-up*
*Completed: 2026-09-18*

## Self-Check: PASSED

- FOUND: internal/cli/present/help.go
- FOUND: internal/cli/present/help_test.go
- FOUND: commit f74dcafc (git log --oneline --all)
- FOUND: commit 44ba22bb (git log --oneline --all)
- FOUND: commit ca77b10f (git log --oneline --all)
- FOUND: commit d60bc6a9 (git log --oneline --all)
- FOUND: commit a73c1966 (git log --oneline --all)
- FOUND: commit 53c6c49a (git log --oneline --all)
- FOUND: commit d0fa12dd (git log --oneline --all)
- Re-ran plan `<verification>`:
  - `GOTOOLCHAIN=go1.26.6 task docs:cli:drift` — green
  - `GOTOOLCHAIN=go1.26.6 go test ./internal/cli/ -count=1 -run 'TestEveryRegisteredFlagIsAccountedFor$|TestEveryCommandHasGroupID$'` — green (37 commands, 111 via reference, 3 via allowlist)
  - `GOTOOLCHAIN=go1.26.6 go test ./internal/cli/... -count=1` — all 5 packages ok (no scheduled RED remains)
  - `GOTOOLCHAIN=go1.26.6 go test -race -count=1 ./internal/cli/ ./internal/cli/present/` — no DATA RACE
  - `GOTOOLCHAIN=go1.26.6 go test ./test/wireoracle/... -count=1` — ok, `git status --porcelain testdata/wireoracle` empty
  - `S=$(git log --format=%H -E --grep='^test\(04-01\): ' -1) && git diff --name-only "$S^..HEAD" -- internal/query internal/mcp testdata/wireoracle testdata/golden` — empty
- `git status --porcelain` — clean.
