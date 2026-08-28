---
phase: 3
reviewers: [codex, claude]
reviewed_at: 2026-08-28T17:12:00Z
review_cycle: 3
plans_reviewed: [03-01-PLAN.md, 03-02-PLAN.md, 03-03-PLAN.md, 03-04-PLAN.md, 03-05-PLAN.md, 03-06-PLAN.md, 03-07-PLAN.md, 03-08-PLAN.md, 03-09-PLAN.md, 03-10-PLAN.md]
plans_commit: be318edc
models:
  codex: "gpt-5.6-sol (reasoning=high)"
  claude: "unknown"
model_sources:
  codex: "banner"
  claude: "unknown"
cycle_summary:
  current_high: 4
  current_actionable: 5
cycle_history:
  - cycle: 1
    current_high: 7
    current_actionable: 34
  - cycle: 2
    current_high: 2
    current_actionable: 7
  - cycle: 3
    current_high: 4
    current_actionable: 5
---

# Cross-AI Plan Review — Phase 3 (Cycle 3, FINAL)

Third and final convergence cycle over the REVISED plan set (commit `be318edc`,
10 plans / 29 tasks), after cycle 2's 2 HIGH + 7 actionable findings. Both lanes
received the same source-grounding prompt plus the cycle-3 directive to mentally
EXECUTE every `<verify>` block, the six pre-approved deviations, the deliberate
deferrals, and the list of vacuous shapes already recorded in this repository.
Both lanes ran to completion with `file:line` citations; neither carries a
`[reviewed-without-repo-access]` or `[reviewed-without-source-citations]` marker,
so both count at full weight.

Cycles 1 and 2 are preserved BELOW as a labeled audit trail. Findings recorded
there that are now fixed are NOT re-counted here.

## Consensus Summary — Cycle 3

**All nine cycle-2 findings are closed.** Both lanes verified this independently
and the orchestrator re-derived each one against the current plan text and its
cited source. Codex: "the nine cycle-2 findings appear closed." Claude: "I
re-derived each one against the current plan text and the source it cites, and
none survives."

**The residual defect class has shifted.** Cycles 1-2 hunted guards that could
not fail. Cycle 3's findings are almost entirely **claims about the source that
reading the source falsifies** — a named test that does not exist, a pinned
Connect code the code never produces, a server refusal the engine explicitly
declines to perform, and a named unformatted file that is already clean. A
guard-shape audit cannot catch these; only resolving every identifier and every
asserted behaviour against the actual tree can. That is why they survived two
cycles.

Orchestrator-executed corroboration of every counted finding:

| Claim under test | Command | Observed | Verdict |
|---|---|---|---|
| `TestGateStancesAgree` exists | `rg -o 'GateStancesAgree' -g '!.planning' .` | `0` | plan names a nonexistent test |
| that guard's exit status | `go test ./internal/upgrade/ -run '^TestGateStancesAgree$' -count=1 -v` | `no tests to run` / `PASS` / exit **0** | **VACUOUS** |
| the real guard | `go test … -run '^TestGateStancesStated$' -v` PASS count | `1` | correct name is `…Stated` |
| depth above ceiling is refused | `internal/query/traverse.go:438-442` → `validateDepth` (rejects `n<0` only) then `clampDepth` | clamps `999` → `50`, returns success | plan's refusal is false |
| nonexistent path → `CodeInvalidArgument` | `node.go:66-72` returns raw `EvalSymlinks` err; `errors.go:43-45` `Is` matches only its own sentinel; `handlers.go:112-131` default arm | `CodeInternal` + `an internal error occurred` | plan's code is wrong |
| `internal/query/files_status_test.go` is unformatted | `gofmt -l` / `gofumpt -l` on that file | `0` / `0` | already clean |
| the real formatting backlog | `gofmt -l . \| grep -v worktrees \| wc -l` | `14` | 13 undeclared files |
| stale coverage figure in `vuln` desc | `rg -n '357' Taskfile.yml` | `Taskfile.yml:1128` | stale after a 5th binary |
| 03-04 threat row vs its own gate | `03-04:464` names `hljs.highlightAuto`; `:309`/`:477` gate it at `0` | contradiction | confirmed |
| 03-03 subtest citation | `oracle_test.go:608` | `TestToolsListOrderIsDeterministic` | miscited; real site is `:119` |

### Agreed Strengths

- **No `<automated>` block in the ten plans is vacuous in the "task never done"
  world.** Both lanes executed them; the orchestrator re-ran the discriminating
  baselines. Every counted finding below is about a guard testing the *wrong
  thing correctly*, not a guard that cannot fail.
- **Both cycle-2 HIGHs are closed by construction, not prose.** `createStatusGate`
  now takes `(client, initialNavigationIdentity)` (`03-09:118,197`) and — the part
  both lanes single out — the fix is anchored by `rg -o 'navigationIdentity\(page\.url\)'
  … -eq 2`, where **1 would mean the client-only constructor cycle 2 rejected**.
  A count assertion that encodes the semantics of the fix, not merely its presence.
- **The `inScopeJobs` / `TestToolModfilesRemainIsolated` blindness is correctly
  diagnosed and registered at both sites**, with positive `rg -o … | wc -l` counts
  hoisted INTO `<verify>` rather than inferred from the guards going green. Both
  lanes confirmed the hardcoded literals at `taskfile_shape_test.go:126-137` and
  `:928`, and that all four registration counts are `0` today.
- **The `.build-manifest` pair is genuinely discriminating.** `source-sha256` → `1`
  (control green), `codegraph init` in `web/build/` → `0` (claim red). `rg` does
  search an explicitly-named dotfile, so the control is not defeated by hidden-file
  skipping — verified.
- **The `--numstat` additive-only awk is correct** and its rejection of the earlier
  reviewer's `exit`-in-a-rule form is right; executed across four worlds → `1/0/1/1`.
- **Requirement coverage is complete** — the union of the ten plans' `requirements:`
  is exactly BRW-01..09, NAV-01..04, SRV-05.

### Agreed Concerns

- **03-10's `vuln`-stance guard is the one outright vacuous block left.** Claude
  found it by executing the named command; the orchestrator reproduced
  `no tests to run` / exit `0`. Codex independently rated 03-10 Task 2 as having
  no task-specific completion signal. Both point at the same plan.
- **03-10 Task 2's factual premise is wrong.** Codex: `gofmt -d
  internal/query/files_status_test.go` "currently produces zero lines"; the
  orchestrator confirms `0` under both `gofmt` and `gofumpt`, against a real
  backlog of `14`.

### Divergent Views

- **The generalized hardcoded-subject-set finding.** Codex rates three separate
  HIGHs on the *residual* generality of `TestToolModfilesRemainIsolated`,
  `inScopeJobs` and the `vuln` binary list — that the NEXT modfile, job or tool
  binary will still be undiscovered — and proposes replacing all three with
  discovery-plus-exceptions. Claude rates the same structures as correctly closed
  for this phase's additions. **Orchestrator adjudicates for Claude, and disposes
  Codex's three:** the brief's directive was to check that Phase 3's own additions
  are registered, which 03-10 does at all three sites; the residual generality is
  the same pre-existing repo gap already recorded for the maintainer and ruled OUT
  OF SCOPE for this phase (the `go.tool-proto.mod` disposition). Converting three
  hand-enumerated gates to discovery is repo-architecture work, not a Phase 3 plan
  defect, and an unnamed edit of that size in the final cycle is exactly the risk
  that disposition was made to avoid. **Recorded for the maintainer, not counted.**
- **03-10 Task 2's severity mechanism.** Codex rates the `<verify>` block VACUOUS
  "in a valid zero-backlog world." The orchestrator could NOT reproduce that: the
  backlog is `14` files, so `task lint:go` is genuinely red if Task 2 is skipped
  and the block does discriminate. The finding is upheld at HIGH on the *other*
  two grounds the orchestrator verified — a false named exemplar and a `<files>`
  list that declares one clean file while 13 dirty ones go undeclared.
- **Overall risk.** Codex: HIGH. Claude: MEDIUM-LOW, becoming LOW after three
  edits. The gap is entirely Codex's three disposed hardcoded-set HIGHs.
  **Orchestrator verdict: MEDIUM** — four execution-affecting defects, all
  small edits, none architectural, none requiring re-planning.

### Findings Considered and Disposed (not counted)

- **Codex's three generalized hardcoded-subject-set HIGHs** (isolation guard,
  `vuln` list, `inScopeJobs`) — the pre-existing gap explicitly ruled out of scope;
  Phase 3's own additions ARE registered at all three sites. See Divergent Views.
- **Claude M1 — 03-01 Task 3's bare `task web:test`.** The CI-wiring assertions
  (`rg -o 'task web:test' .github/workflows/ci.yml | wc -l` = 1 and the
  `name: web unit tests (vitest)` count) are present in `<acceptance_criteria>`
  (`03-01:367-368`). Incorporated, therefore disposed — the same basis on which
  cycle 2 disposed the analogous name-only vitest greps. Worth hoisting into
  `<verify>`, not counted.
- **Claude L2 — 03-10 Task 2's byte-identity conjunct is untracked-file-satisfiable.**
  Covered by the adjacent criterion `git log --oneline -- .golangci.yml | wc -l`
  returns exactly 1, which the plan itself describes as "the commit-order-independent
  form of the same property." Written rationale present.
- **Claude S9 — 03-06 requires its deferral todo by acceptance criterion with no
  matching `<action>` step.** Confirmed (03-08 has such a paragraph; 03-06 does
  not). The artifact and its content ARE specified at `03-06:232` and its path at
  `:26`. Incorporated, therefore disposed; the asymmetry is worth closing.
- **Orchestrator observation, uncorroborated by either lane — the shallow-routing
  negative grep is evadable by a multi-line import.** `rg` is line-oriented by
  default, so a planted `import {\n  pushState\n} from '$app/navigation';` returns
  `0` while the single-line form returns `1` (both executed). Claude tested the
  single-line plant and rated the blocks SOUND; Codex did not raise it. The
  practical exposure is narrow — the real call site is short enough that no
  formatter would wrap it — and the primary NAV-01/NAV-02 contract is carried by
  D-11's `goto()` tests, not by this prohibition. Recorded, not counted.
- **Environmental, not a plan defect (Claude).** `go.mod` declares `go 1.26.5` with
  `GOTOOLCHAIN=auto`; a host defaulting to go1.27.0 resolves *up* and
  `cockroachdb/swiss` (indirect via `pebble/v2`) fails to compile, making every Go
  `<verify>` locally unrunnable. CI is unaffected (`ci.yml` uses
  `go-version-file: go.mod`). Workaround: `GOTOOLCHAIN=go1.26.5`. Hand to the
  executor rather than letting wave 1 debug pebble.

### Current HIGH Concerns (4)

1. **03-07 — the plan requires a server refusal the engine explicitly declines to
   perform.** `03-07:41`, `:196`, `:216` and the acceptance criterion at `:247`
   state that an out-of-range `depth` "produces the server's own refusal rendered
   as a named state," and that `?depth=999` therefore "renders an explicit message
   instead of an empty pane." The source disagrees: `validateDepth`
   (`internal/query/validate.go:137-142`) rejects only `n < 0`, and
   `Engine.Impact` (`internal/query/traverse.go:438-442`) then calls `clampDepth`,
   which returns `MaxDepth` for anything above `50`. `internal/uiserver/handlers.go:486-491`
   documents the pass-through and echoes back the Engine's own clamped value.
   `?depth=999` returns **success with depth 50**. A mocked client can satisfy the
   planned UI test while the real RPC succeeds. Note the asymmetry the plan
   conflates: `limit` IS refused (`validateLimit` rejects `n > MaxLimit`,
   `validate.go:107-115`) — `depth` is not. Codex raised; orchestrator confirmed.
   The plan's own threat row `:364` half-exposes it by saying "refuse or bound."
   Fix: state the clamp contract, treat the response's echoed `depth` as
   authoritative, and drop the invalid-input expectation for `depth` (keeping it
   for `limit`).

2. **03-10 — the ADVISORY-stance guard names a test that does not exist and
   therefore exits 0 having run nothing.** `03-10:130`, `:280`, `:304` and `:470`
   all name `TestGateStancesAgree`. `rg -o 'GateStancesAgree' -g '!.planning' .`
   returns `0`; the real guard is `TestGateStancesStated`
   (`internal/upgrade/taskfile_shape_test.go:825`, with the `hasStanceWord(vulnDesc,
   "advisory")` assertion at `:836`). Executed:
   `go test ./internal/upgrade/ -run '^TestGateStancesAgree$' -count=1 -v` →
   `no tests to run` / `PASS` / **exit 0**. This is the repository's own recorded
   `go test -run PATTERN matches nothing` vacuity shape, and it is the SOLE check
   that step (f) edit 3's rewrite of the `vuln` `desc:` preserved the word
   ADVISORY — a constraint the plan flags twice as red-turning. Claude raised;
   orchestrator executed. Fix: rename to `TestGateStancesStated` and count the
   PASS line (`grep -Eo '^--- PASS: TestGateStancesStated' | wc -l` = 1 — which
   returns 1 with the correct name and 0 with the wrong one), and label it a
   REGRESSION guard that is green by design, whose completion partner is the
   already-red five-name `for name in …` count.

3. **03-05 — the since-deleted-file case pins `CodeInvalidArgument`; the source
   produces `CodeInternal` with a scrubbed message.** `03-05:373` (`<behavior>`),
   `:434-436`, `:444` and the pinned test at `:465` all state that a repo-relative,
   non-escaping path absent from the working tree "is refused as
   `CodeInvalidArgument`, carrying the confinement gate's own message." Two steps
   falsify it: `resolveSourcePath` returns the **raw** `filepath.EvalSymlinks`
   error, not a classified one (`internal/query/node.go:66-72` — every other
   refusal in that function uses `invalidArgumentf`); and `classifiedError.Is` is
   `return target == e.class` (`internal/query/errors.go:43-45`), so an
   `*os.PathError` never matches `ErrInvalidArgument` and `mapEngineError` falls to
   its default arm → `connect.NewError(connect.CodeInternal, errInternal)` with the
   fixed `"an internal error occurred"` (`internal/uiserver/handlers.go:112-140`).
   That arm's own comment names `resolveSourcePath` PathErrors as its common
   occupants. Two consequences: the pinned test fails as specified, and both
   tempting repairs are wrong (accepting reality silently downgrades the case to an
   opaque internal error; classifying `EvalSymlinks` changes the shared gate that
   `GetNodeDetail`, `Explore` and the MCP path all use, which `:449-451` defers).
   Worse, the product argument justifying the disposition — *"Being refused with a
   clear message is an acceptable answer"* — is void, and it is being put to the
   maintainer at a `gate="blocking-human"`, `reversibility="one-way"` checkpoint.
   Claude raised; orchestrator confirmed at every cited line. Fix: pin the real
   behaviour (`CodeInternal` + the scrubbed message + the `writeDiagLine` cause),
   and re-open the checkpoint question with the trade-off stated accurately.

4. **03-10 Task 2 — the named unformatted file is already clean, and `<files>`
   declares one clean file against a 14-file backlog.** `03-10:41`, `:317` and
   `:319` assert that `internal/query/files_status_test.go` "has carried an
   unformatted block since it was introduced" and make it Task 2's ONLY declared
   `<files>` entry. Measured: `gofmt -l internal/query/files_status_test.go` → `0`
   and `gofumpt -l` → `0`; the real backlog is `14` files under `gofmt` (26 under
   `gofumpt`), none of them declared. `03-10:377`/`:401` then have Task 3 plant its
   RED proof in that file *"because Task 2 has just fixed it"* — a premise that
   will not hold. Codex raised (as a vacuity claim); orchestrator confirmed the
   facts but NOT the vacuity — `task lint:go` is genuinely red against 14 dirty
   files, so the block discriminates. Upheld at HIGH for the false premise, the
   under-scoped `<files>`/`files_modified` accounting, and Task 3's broken
   dependency. Fix: drop the specific-file claim (or re-derive it), declare the
   real backlog set, and pick Task 3's plant target from a file Task 2 actually
   touches.

### Current Actionable Non-HIGH Concerns (5)

1. **LOW (Claude) — 03-10 step (f) edit 3 leaves a stale coverage figure in the
   `vuln` desc.** The instruction (`:275-279`) names the two "four" occurrences and
   the build echo, but the `desc:` also states *"the 357 measured third-party
   modules that execute as credentialed CI tooling"* (`Taskfile.yml:1128`).
   golangci-lint is, by the plan's own words (`:258-259`), "plausibly the largest
   third-party executable tree this repository would carry." PLAN change: extend
   edit 3 to recompute the figure or restate it as a floor, using the module count
   Task 1 already records.

2. **LOW (Claude) — 03-04's threat row T-03-02 names a function the same plan
   forbids and gates at zero.** `03-04:464` says markup is produced by
   `hljs.highlight(rawText, {...}).value` **"or `hljs.highlightAuto(rawText).value`"**,
   while `:309` and `:477` assert `rg -o 'highlightAuto' web/src/lib/ | wc -l`
   returns **0** and the action prohibits auto-detection outright. PLAN change:
   replace the `highlightAuto` clause in the threat row with the plaintext-escape
   branch the action actually specifies.

3. **LOW (Claude) — 03-05's `--numstat` additive check has no stated ordering and
   reads red on a clean tree.** The awk is correct, but `NR == 0 → exit 1` means it
   fails against an unmodified working tree, and `03-05:459-462` says only "Run
   exactly … and require exit 0." 03-10 states its equivalent ordering explicitly
   (*"Run the block BEFORE committing this task"*, `:345`). Fails closed, so the
   risk is a confusing red, but Task 3 touches nine files. PLAN change: add the
   pre-commit ordering sentence.

4. **LOW (Codex) — 03-03 and 03-10 move a todo to `completed/` without declaring
   the destination.** `03-03:339` and `03-10:421` both move a todo from
   `.planning/todos/pending/` to `.planning/todos/completed/`, and both assert the
   destination in an acceptance criterion (`03-03:356`, `03-10:435`), but neither
   declares the `completed/` path in `files_modified` or in the responsible task's
   `<files>`. This is the same artifact-ownership gap cycle 2 closed for the
   `pending/` paths in 03-06 and 03-08. PLAN change: add both completed paths to
   `files_modified` and to Task 3's `<files>`.

5. **LOW (Claude) — 03-03 Task 1 miscites the pre-existing subtest.** `03-03:150-151`
   says `TestFrozenTranscriptsMatch/toolslist-repeat` "already runs today
   (`test/wireoracle/oracle_test.go:608-643`)". `:608` is
   `TestToolsListOrderIsDeterministic`; the subtest is generated by
   `TestFrozenTranscriptsMatch`'s `t.Run(sc.Name, …)` at `oracle_test.go:119` from
   the scenario at `scenarios.go:669`. The *claim* is correct and verified — only
   the citation is wrong, so a reader checking the cycle-1 defect lands on a
   different test. PLAN change: correct the two line references.

---

## Codex Review (Cycle 3)

# Phase 3 Plan Review — Cycle 3

## Summary

The revised plan set is substantially stronger: the nine cycle-2 findings appear closed, and most task-level verification now fails against the unimplemented tree through materialized test output, positive controls, or direct assertions on newly introduced artifacts. However, five HIGH issues remain. Plan 03-07 contradicts the real traversal contract; Plan 03-10 contains a task whose guard can pass without that task running; and all three warned-about hardcoded subject sets remain structurally incomplete despite being extended for the immediate additions. All findings below are newly raised; none are carried over from cycle 2.

## Strengths

- Source confinement is correctly centralized. [`resolveSourcePath`](</Volumes/Code/github.com/seanb4t/codegraph-go/internal/query/node.go:17>) rejects empty, absolute, traversal, and post-symlink-escape paths, while [`readSourceFile`](</Volumes/Code/github.com/seanb4t/codegraph-go/internal/query/node.go:81>) delegates through that single gate. This supports the plans’ intended `ValidateRepoRelativePath` delegation rather than introducing a second confinement implementation.

- The wire model gives the UI reliable discriminants. Status fields independently represent initialization, store existence, indexing, and indexed commit in [`ui.proto`](</Volumes/Code/github.com/seanb4t/codegraph-go/internal/uiproto/uiv1/ui.proto:116>). Node detail similarly uses an explicit mode and `detail_gathered`, preventing empty calls/source lists from being misread as another response shape in [`ui.proto`](</Volumes/Code/github.com/seanb4t/codegraph-go/internal/uiproto/uiv1/ui.proto:382>). Plans 03-07 through 03-09 build against these distinctions rather than inferring state from emptiness.

- The final web-bundle guard is now bound to Phase 3 output. Plan 03-09 requires named new suites, `StatusBanner` in source, `codegraph init` in built bytes, a manifest positive control, and replacement of the pre-phase digest in [`03-09-PLAN.md`](</Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/03-browse-inspect-navigation/03-09-PLAN.md:401>). The current manifest contains exactly one source digest at [`web/build/.build-manifest`](</Volumes/Code/github.com/seanb4t/codegraph-go/web/build/.build-manifest:1>), so absence cannot masquerade as “old digest removed.” The shipped tree is also genuinely embedded through `//go:embed all:build` in [`web/embed.go`](</Volumes/Code/github.com/seanb4t/codegraph-go/web/embed.go:20>).

- The immediate 03-10 additions are directly asserted. Its Task 1 verify requires the new target, files, modfile registration, forbidden package, and five-binary loop; Task 3 requires exactly one workflow invocation and exactly one `lint-go` fixture entry in [`03-10-PLAN.md`](</Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/03-browse-inspect-navigation/03-10-PLAN.md:290>) and [`03-10-PLAN.md`](</Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/03-browse-inspect-navigation/03-10-PLAN.md:425>). Those direct checks correctly close the immediate cycle-2 omissions.

- Plan 03-07’s navigation generation is well specified: one route-minted identity spans both detail and blast-radius requests, and the test must assert both committed halves by value in [`03-07-PLAN.md`](</Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/03-browse-inspect-navigation/03-07-PLAN.md:221>). That targets the actual cross-request race instead of merely checking individual abort controllers.

## Concerns

- **HIGH — NEWLY RAISED: Plan 03-07 requires a server refusal that the engine explicitly forbids.** The plan says an out-of-range depth produces the server’s refusal and specifically expects `?depth=999` to become an invalid-input state at [`03-07-PLAN.md`](</Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/03-browse-inspect-navigation/03-07-PLAN.md:41>) and [`03-07-PLAN.md`](</Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/03-browse-inspect-navigation/03-07-PLAN.md:212>). In the source, `validateDepth` deliberately accepts values above `MaxDepth` and documents that they are silently capped in [`validate.go`](</Volumes/Code/github.com/seanb4t/codegraph-go/internal/query/validate.go:129>). `Impact` then calls `clampDepth` in [`traverse.go`](</Volumes/Code/github.com/seanb4t/codegraph-go/internal/query/traverse.go:438>), and the regression test requires `999999` to succeed with `Depth == MaxDepth` in [`traverse_test.go`](</Volumes/Code/github.com/seanb4t/codegraph-go/internal/query/traverse_test.go:551>). A mocked client can therefore satisfy the planned UI test while the real RPC returns success with depth 50. The plan’s own threat table partly exposes the contradiction by saying the server may “refuse or bound” values at [`03-07-PLAN.md`](</Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/03-browse-inspect-navigation/03-07-PLAN.md:364>).

- **HIGH — NEWLY RAISED: the modfile-isolation guard remains a hardcoded-subject gate.** `TestToolModfilesRemainIsolated` reads paths from literals at [`taskfile_shape_test.go`](</Volumes/Code/github.com/seanb4t/codegraph-go/internal/upgrade/taskfile_shape_test.go:23>) and iterates an explicitly written slice at [`taskfile_shape_test.go`](</Volumes/Code/github.com/seanb4t/codegraph-go/internal/upgrade/taskfile_shape_test.go:928>). Plan 03-10 adds another literal to that set in [`03-10-PLAN.md`](</Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/03-browse-inspect-navigation/03-10-PLAN.md:216>), which protects the immediate new file but preserves the defect: the next `go.tool*.mod` can exist without being discovered, and the isolation test passes because it never visits it.

- **HIGH — NEWLY RAISED: the vulnerability subject set remains hardcoded.** The current `vuln` target independently hardcodes both five future build lines and the scan-loop word list; the present four-subject forms are visible in [`Taskfile.yml`](</Volumes/Code/github.com/seanb4t/codegraph-go/Taskfile.yml:1168>) and [`Taskfile.yml`](</Volumes/Code/github.com/seanb4t/codegraph-go/Taskfile.yml:1179>). Plan 03-10 appends the new binary to both lists at [`03-10-PLAN.md`](</Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/03-browse-inspect-navigation/03-10-PLAN.md:253>). That fixes current coverage but establishes no discoverable source of truth. A later tool binary omitted from both lists remains completely unbuilt and unscanned while the target reports normally.

- **HIGH — NEWLY RAISED: the workflow single-definition guard remains a hardcoded job allowlist.** `inScopeJobs` enumerates jobs by hand at [`taskfile_shape_test.go`](</Volumes/Code/github.com/seanb4t/codegraph-go/internal/upgrade/taskfile_shape_test.go:119>), and `TestWorkflowRunBodiesInvokeTask` visits only those entries at [`taskfile_shape_test.go`](</Volumes/Code/github.com/seanb4t/codegraph-go/internal/upgrade/taskfile_shape_test.go:1365>). Plan 03-10 explicitly adds only `lint-go` to that list at [`03-10-PLAN.md`](</Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/03-browse-inspect-navigation/03-10-PLAN.md:385>). The stale-exception check is useful, but it cannot detect an entirely unlisted job. A new workflow job containing an inline `go test ./...` is absent from the iteration and passes the guard.

- **HIGH — NEWLY RAISED: Plan 03-10 Task 2’s verify block does not prove Task 2 occurred.** Task 2 claims a known malformed indentation block in `internal/query/files_status_test.go` at [`03-10-PLAN.md`](</Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/03-browse-inspect-navigation/03-10-PLAN.md:315>), but `gofmt -d internal/query/files_status_test.go` currently produces zero lines; the cited test block is already conventionally formatted at [`files_status_test.go`](</Volumes/Code/github.com/seanb4t/codegraph-go/internal/query/files_status_test.go:59>). Its verify command checks only artifacts committed by Task 1, that the config is unchanged, and that the repository is currently lint-clean at [`03-10-PLAN.md`](</Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/03-browse-inspect-navigation/03-10-PLAN.md:352>). In a valid zero-backlog world, every conjunct succeeds when Task 2 is skipped. No inventory artifact, source change, or Task-2-specific test is required.

- **LOW — NEWLY RAISED: conditional todo moves omit their completed destinations from declared ownership.** Plan 03-03 declares only the pending todo path at [`03-03-PLAN.md`](</Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/03-browse-inspect-navigation/03-03-PLAN.md:7>) but may move it to `completed/` at [`03-03-PLAN.md`](</Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/03-browse-inspect-navigation/03-03-PLAN.md:338>). Plan 03-10 likewise declares only the pending path at [`03-10-PLAN.md`](</Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/03-browse-inspect-navigation/03-10-PLAN.md:9>) while Task 3 moves it at [`03-10-PLAN.md`](</Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/03-browse-inspect-navigation/03-10-PLAN.md:421>). The completed paths should be in `files_modified` and the responsible task’s `<files>`.

## Suggestions

1. Resolve the depth contract before executing 03-07. Either:

   - Change the real engine contract to reject `depth > MaxDepth`, updating `validateDepth`, the engine regression tests, and all CLI/MCP expectations; or
   - Preserve clamping, treat the response’s echoed depth as authoritative, and replace the URL with that value.

   A sound real-path test should call the actual engine or UI handler with `MaxDepth+1`, not a stub programmed to reject it. Under the plan’s current rejection expectation, that test fails before the server change because the real engine returns success with `Depth == MaxDepth`.

2. Replace the three manual subject sets with discovery plus explicit, reasoned exceptions:

   - Discover `go.tool*.mod` files and require set equality with the isolation test’s registered subjects.
   - Derive the vulnerability build/scan set from one checked manifest, then require discoverable tool directives to map to that manifest.
   - Parse every workflow job and require its `run:` steps to invoke Task unless the job or step has an explicit reasoned exception; also reject stale exceptions.

   These remedies fail correctly: a new modfile, tool directive, or inline workflow job appears in the discovered set but not the registered/manifest/exception set, producing a non-empty set difference and a nonzero Go test result. Synthetic unit cases should pin each mismatch independently.

3. Make 03-10 Task 2 produce a machine-checked inventory artifact and allow an honest zero-backlog outcome. For example:

   ```sh
   inventory=.planning/phases/03-browse-inspect-navigation/03-10-LINT-INVENTORY.txt
   test -f "$inventory" &&
   test "$(rg -o '^TOTAL=[0-9]+$' "$inventory" | wc -l | tr -d ' ')" -eq 1 &&
   task lint:go
   ```

   In the task-never-done world, `test -f` exits 1 and the `&&` chain stops. `rg -o` counts matches, not merely matching lines. If `TOTAL=0`, the task should record that result without claiming a source fix; if it is positive, the inventory must name the findings and the resulting edits.

4. Add both completed todo destinations to the plans’ frontmatter and Task 3 `<files>` declarations.

## Verify-block audit

| Plan/task | Command or guarded component | Verdict | Mental execution |
|---|---|---:|---|
| 03-07 Task 2 | `pnpm --dir web test …; test "$(grep -Eo 'browse-state' /tmp/bs.txt \| wc -l \| tr -d ' ')" -ge 1` | **SOUND** | With the task absent, the named suite is absent and the final numeric test fails. It remains semantically insufficient because a stub can invent the contradicted rejection response. |
| 03-09 Task 3 | Full conjunction at [`03-09:402`](</Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/03-browse-inspect-navigation/03-09-PLAN.md:402>) | **SOUND** | In the current tree the new suite names, `StatusBanner`, and built `codegraph init` marker are absent. The manifest-key check succeeds before the old-digest absence check, so a missing manifest cannot pass. |
| 03-10 Task 1, full block | Target/file checks plus exact registration and scan-loop counts at [`03-10:291`](</Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/03-browse-inspect-navigation/03-10-PLAN.md:290>) | **SOUND** | With Task 1 absent, `lint:go`, its files, registration symbols, and five-name loop all have zero matches. |
| 03-10 Task 1, embedded isolation test | `go test ./internal/upgrade -run '^TestToolModfilesRemainIsolated$'` | **VACUOUS** | Create a new unregistered `go.tool*.mod`: the literal iteration never reads it, so the test’s behavior is unchanged. The outer direct checks only bind the current golangci subject. |
| 03-10 Task 1, resulting vulnerability gate | `task vuln` over the manually written build and `for name in …` lists | **VACUOUS** | Add a tool outside both lists: no build or scan is attempted, and the target completes exactly as before. |
| 03-10 Task 2 | `test -f .golangci.yml && … && git diff --exit-code HEAD -- .golangci.yml && task lint:go` | **VACUOUS** | After Task 1, all config conjuncts are already true. If the first run has zero findings—as the already-gofmt-clean named target permits—`task lint:go` exits 0 without Task 2 running. |
| 03-10 Task 3, full block | Named Go test plus exact `task lint:go` and `JobID: "lint-go"` counts at [`03-10:426`](</Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/03-browse-inspect-navigation/03-10-PLAN.md:425>) | **SOUND** | With Task 3 absent, both exact counts are zero, so the block fails even if the pre-existing Go test passes. |
| 03-10 Task 3, embedded workflow test | `go test ./internal/upgrade -run '^TestWorkflowRunBodiesInvokeTask$'` | **VACUOUS** | Add a workflow job but omit it from `inScopeJobs`: the test never parses that job and its result is unchanged. |

The environment allowed complete source and plan inspection, plus read-only shell probes. `go test` execution itself could not create its temporary work directory under the read-only filesystem, so no runtime test-pass claim above depends on that failed invocation.

## Risk Assessment

**HIGH.** Most immediate task guards are now discriminating, but 03-07 currently asks implementers to test a behavior contradicted by the real engine, and 03-10 preserves all three explicitly warned-about subject-enumeration defects. Task 03-10-2 also has no task-specific completion signal and can be a genuine no-op. These are execution-affecting failures, not documentation polish.

UNRESOLVED: high=5 actionable_non_high=1

---

## Claude Review (Cycle 3)


# Cross-AI Plan Review — Phase 3, Cycle 3 (FINAL)

## 1. Summary

I read all ten plans in full and verified every load-bearing claim against the tree at `be318edc`, executing the risky `<verify>` shapes rather than reading them. **All nine cycle-2 findings are genuinely closed** — I re-derived each one against the current plan text and the source it cites, and none survives. The convergence is real: the guard discipline in this set is now unusually good (positive controls paired in the same `&&` chain, `rg -o | wc -l` never `rg -c`, `grep -Eo … | wc -l -eq N` instead of bare `go test -run` exit status, observed-zero baselines recorded inline). I found **two HIGH defects that no prior cycle raised**, and both are *factual claims about the source that reading the source falsifies* rather than guard-shape errors — which is a different failure class than cycles 1 and 2 hunted, and is why they survived. One is a vacuous guard I proved vacuous by executing it (`go test -run '^TestGateStancesAgree$'` exits 0 having run nothing, because that test does not exist). The other is a wrong Connect error code pinned into a `<behavior>` case and used to justify a product decision, where the source produces a different, materially worse answer. Both are small edits; neither is architectural.

`★ Insight ─────────────────────────────────────`
- The residual defect class in this plan set has shifted. Cycles 1–2 found **guards that could not fail**; cycle 3's findings are **guards that test the wrong thing correctly** — a named test that doesn't exist, and a pinned error code the code never produces. A guard-shape audit cannot catch either; only resolving every identifier the plan names against the actual tree can.
- The origin of HIGH-2 is instructive: the wrong claim entered at `03-REVIEWS.md:1235-1238` as an *orchestrator observation*, was adopted verbatim by the planner, and was then cited by cycle 2 as a **strength** (`03-REVIEWS.md:337`). A review artifact is an input to the next planner, so an unverified assertion inside it propagates with the authority of a finding.
- `classifiedError.Is` is `return target == e.class` (`internal/query/errors.go:43-45`) — a deliberately non-wrapping sentinel. That design makes classification *explicit*, which means every unclassified `return "", err` is silently an internal error. That is a repo-wide property worth knowing before writing any test that pins a Connect code.
`─────────────────────────────────────────────────`

## 2. Strengths

- **The cycle-2 HIGH on `createStatusGate` is closed by construction, not by prose.** `03-09-PLAN.md:118,197` now declares `createStatusGate(client, initialNavigationIdentity)`, and — critically — the fix is *anchored by a count*: `rg -o 'navigationIdentity\(page\.url\)' web/src/routes/+layout.svelte | wc -l` must equal exactly **2** (`03-09-PLAN.md:319,325`), with the plan stating that **1 would mean the client-only constructor cycle-2 rejected**. Observed 0 on the tree today. That is the rare case where a count assertion encodes the *semantics* of the fix, not just its presence.
- **The `inScopeJobs` / `TestToolModfilesRemainIsolated` blindness is correctly diagnosed and closed at both sites.** I confirmed both hardcoded iteration sets: `inScopeJobs` is a 10-element literal (`internal/upgrade/taskfile_shape_test.go:126-137`) with no `lint-go` entry, and `TestToolModfilesRemainIsolated` iterates `[]string{toolModfilePath, lintModfilePath}` at exactly the cited `:928`, with the `isolat` substring check at `:934` and `parseToolModfileHeaderComment` at `:340`. All three line citations are exact. 03-10 now asserts registration with positive `rg -o … | wc -l` counts **inside** both `<verify>` blocks, and I confirmed both are 0 today.
- **The `.build-manifest` absence-claim pair is now genuinely discriminating.** I ran both halves: `rg -o 'source-sha256' web/build/.build-manifest | wc -l` → **1** (control green today), `rg -o 'fc4ae27b…' … | wc -l` → **1** (absence claim red today). The literal in `03-09-PLAN.md:361,409` matches the committed manifest byte-for-byte. `rg` does search an explicitly-named dotfile, so the control is not itself defeated by hidden-file skipping — I verified that too.
- **The shallow-routing prohibition regex was fixed correctly and I reproduced the discrimination.** Against a planted `import { pushState, replaceState } from '$app/navigation';` the corrected single-quoted `\$` form returns **1**; against the real tree it returns **0**; the positive control `rg -o 'from .\$app/state.' web/src/ | wc -l` returns **1** (`web/src/routes/+layout.svelte`). 03-07 correctly uses a *stronger* control (`$app/navigation` itself, which it imports for `goto`) and says why 03-04 cannot.
- **The `--numstat` additive-only awk is correct, and the plan's rejection of the reviewer's suggested form is right.** I executed all four worlds: no-diff → 1, additions-only → 0, has-deletions → 1, deletion-on-a-later-line → 1. `03-05-PLAN.md:462`'s warning that `awk '$2 != 0 { exit 1 } END { exit NR == 0 }'` returns 0 for the deletion case (because a rule's `exit` still runs `END`, whose `exit` overrides) is accurate.
- **Go subtest indentation was checked, not assumed.** `03-02-PLAN.md:155`'s `^ *--- PASS: …/[a-zA-Z0-9_-]+` — I generated a real subtest named `in-repo control`, confirmed Go emits **four spaces** (not a tab) and rewrites the space to `_` while leaving the hyphen, and confirmed the regex counts 2/2. The cycle-1 `[a-z_]+` finding is genuinely closed.
- **Every load-bearing source citation I sampled is exact:** `resolveSourcePath` at `node.go:33-79`, `readSourceFile` at `:85`, `commit_sha` at `ui.proto:151/157`, `SourceFor` at `detail.go:259`, `uiMultiDefCap` at `handlers.go:153` with `gathered := i < uiMultiDefCap` at `:704`, `vuln` at `Taskfile.yml:1123-1197` with the four build lines at `:1169-1172` and `for name in task goreleaser govulncheck actionlint` at `:1179`, `proto:drift`'s floor of 4 with `compared ${nfiles} generated files` at `:345`, the `lint` wrapper at `:4977-4982`, `.svelte-kit/tsconfig.json`'s `../tests/**/*.svelte` include. The PATTERNS.md:93 staleness correction was actually written into the file.
- **Requirement coverage is complete.** The union of the ten plans' `requirements:` fields is exactly BRW-01..09, NAV-01..04, SRV-05 — all fourteen — plus the two folded TODO ids, with 03-03 and 03-10 both explicitly declaring they do not gate the phase's five browse criteria.

## 3. Concerns

### HIGH — NEWLY RAISED

**H1. `03-10` — the ADVISORY-stance guard runs zero tests and exits 0. The test it names does not exist.**

`03-10-PLAN.md:304` and `:470` both require:

```
go test ./internal/upgrade/ -run '^TestGateStancesAgree$' -count=1 -v
```

There is no `TestGateStancesAgree` in this repository — `rg -o 'GateStancesAgree' -g '!.planning' .` returns **0**. The real guard is `TestGateStancesStated` (`internal/upgrade/taskfile_shape_test.go:825`), whose `hasStanceWord(vulnDesc, "advisory")` assertion is at `:836`. The plan's line citation `:831-837` lands inside the right function body; only the *name* is wrong.

Executed, not read:

```
$ GOTOOLCHAIN=go1.26.5 go test ./internal/upgrade/ -run '^TestGateStancesAgree$' -count=1 -v
testing: warning: no tests to run
PASS
ok  github.com/seanb4t/codegraph-go/internal/upgrade  0.227s [no tests to run]
exit 0
```

This is the repository's own recorded vacuity shape — `go test -run PATTERN` exiting 0 when the pattern matches nothing — and it is the **sole** check that 03-10 step (f) edit 3's rewrite of the `vuln` `desc:` preserved the word ADVISORY. The plan itself flags that constraint twice as red-turning ("a rewrite that drops it turns this edit red", `:282`; "which is what proves the rewrite did not drop the stance the guard binds", `:304`). It proves nothing. Failure scenario: the executor rewrites the multi-paragraph `desc:` to say five binaries, drops or rephrases ADVISORY, runs the named command, sees green, and ships a `vuln` target whose stated stance no longer matches `ci.yml`'s advisory-guard step — the exact "unstated stance mismatch" `TestGateStancesStated`'s own doc comment (`:820-825`) exists to prevent.

**H2. `03-05` — the since-deleted-file case pins `CodeInvalidArgument`; the source produces `CodeInternal` with a scrubbed message.**

`03-05-PLAN.md:373` (`<behavior>`), `:434-436` and `:444` (step (f)) all state that a repo-relative, non-escaping path that does not exist in the working tree "is refused as `CodeInvalidArgument`", "carrying the confinement gate's own message", and `:465` pins a test on it. That is false against the source, in two steps:

1. `internal/query/node.go:64-71` — the two `filepath.EvalSymlinks` calls return the **raw** error, not a classified one:
   ```go
   resolvedAbs, err := filepath.EvalSymlinks(abs)
   if err != nil {
       return "", err        // *os.PathError — never invalidArgumentf
   }
   ```
   Every *other* refusal layer in that function uses `invalidArgumentf(...)`. `EvalSymlinks` is precisely the layer that fires for a nonexistent path.
2. `internal/query/errors.go:43-45` — `func (e *classifiedError) Is(target error) bool { return target == e.class }`. An `*os.PathError` is not a `*classifiedError`, so `errors.Is(err, query.ErrInvalidArgument)` is **false**, and `mapEngineError` (`internal/uiserver/handlers.go:101-131`) falls to its default arm: `connect.NewError(connect.CodeInternal, errInternal)` where `errInternal` is the fixed `"an internal error occurred"` (`:140`). That arm's own comment names this case explicitly: *"in practice the common occupants are `*os.PathError` values from `readSourceFile` and `resolveSourcePath`."*

Two distinct consequences, and the second is worse than the first:

- **The pinned test fails as specified.** The executor gets `CodeInternal` where the plan demands `CodeInvalidArgument`, and both tempting repairs are wrong. Changing the test to match reality silently converts the recorded product decision into "the most useful BRW-09 case returns an opaque internal error." Classifying `EvalSymlinks` failures inside `resolveSourcePath` is a behaviour change to the shared gate that `GetNodeDetail`, `Explore` and the MCP path all use — which `03-05-PLAN.md:449-451` explicitly defers to a later phase.
- **The product argument that justifies the disposition is void.** Step (f) adopts the refusal on the grounds that *"Being refused with a clear message is an acceptable answer; a divergent second gate is not."* There is no clear message. The developer sees `an internal error occurred`; the cause goes only to the server's diagnostic stream via `writeDiagLine`. The maintainer is being asked, at a `gate="blocking-human"` checkpoint, to accept a trade-off described inaccurately — and this is the case BRW-09 is *most* useful in, by the plan's own words (`:435-436`).

Origin note: the wrong code entered at `03-REVIEWS.md:1235-1238` as an orchestrator observation, was adopted by the planner, and was cited by cycle 2 as a strength at `03-REVIEWS.md:337`. It was never raised as a finding, so this is NEWLY RAISED, not carried over.

### MEDIUM — NEWLY RAISED

**M1. `03-01` Task 3's `<verify>` is a bare `task web:test`; the CI wiring it exists to install is unguarded.**

`03-01-PLAN.md:362` is `<automated>task web:test</automated>`. That is red in the never-done world (the target does not exist), so the block is not vacuous — but it passes in the world where step (b) (adding the `web unit tests (vitest)` step to `ci.yml`'s `test` job) and step (d) (the RED demo against an empty include glob) were never performed. The two assertions that *would* catch it live only in `<acceptance_criteria>` (`:367-368`). I confirmed both are 0 today: `rg -o 'task web:test' .github/workflows/ci.yml | wc -l` → 0, against a working control of 8 for `rg -o 'task web:[a-z:]*'`.

This is the identical shape cycle 1 raised against 03-10 Task 3 and cycle 1 raised against 03-03 Task 1 — both fixed by hoisting the counts into `<verify>`. 03-01 was not swept in either pass. It matters more here than usual because 03-01 is the wave-1 plan every JS `<verify>` in the phase depends on, and because the phase deliberately opens a five-wave intended-RED `web:drift` window in this same plan, so "CI is red for a known reason" is already the expected state — a *second* missing CI leg would be very easy to lose in that noise.

### LOW — NEWLY RAISED

**L1. `03-10` step (f) edit 3 leaves a stale coverage figure in the `vuln` desc.** The instruction (`:275-279`) names the two "four" occurrences and the `==> Building tool binaries` echo. The `desc:` also states *"the **357 measured third-party modules** that execute as credentialed CI tooling"* (verified verbatim in `task --list-all`). golangci-lint's tree is, by the plan's own words (`:258-259`), "plausibly the largest third-party executable tree this repository would carry." Leaving 357 makes the desc a coverage claim that stopped being true — which is exactly the reasoning the plan uses to justify the other two edits (`:279`: *"an unchanged `desc:` is a claim about coverage that stopped being true"*). Task 1 already requires recording "the module count the new modfile gained," so the number is in hand.

**L2. `03-10` Task 2's byte-identity conjunct is satisfiable by an untracked file.** `git diff --exit-code HEAD -- .golangci.yml` (`:353`) outputs nothing and exits 0 for an untracked path, so the "byte-identical to what Task 1 committed" property holds vacuously if Task 1 wrote the file but did not commit it. The two preceding conjuncts (`test -f`, the `# enabled-linters: N` anchor) keep the block red in the task-never-done world, and the `git log --oneline -- .golangci.yml | wc -l` = 1 criterion (`:357`) covers it after the commit — so this is a narrow gap, not a vacuous block.

**L3. `03-04`'s threat row T-03-02 still names a function the same plan forbids and gates at zero.** `03-04-PLAN.md:464` says markup is produced by `hljs.highlight(rawText, {...}).value` **"or `hljs.highlightAuto(rawText).value`"**. The action (`:257-263`) prohibits auto-detection with a paragraph of reasoning, and the acceptance criterion (`:309`) asserts `rg -o 'highlightAuto' web/src/lib/ | wc -l` returns **0**. The threat register now describes a mitigation the plan's own gate makes impossible — cycle-1 residue that survived both revisions.

**L4. `03-05`'s `--numstat` additive check must run pre-commit, and does not say so.** The awk is correct, but `NR == 0 → exit 1` means it fails against a clean working tree. 03-10 Task 2 states its ordering explicitly (*"Run the block BEFORE committing this task"*, `:346`); `03-05-PLAN.md:459-462` says only "Run exactly … and require exit 0." Fails closed, so the risk is a confusing red rather than a false green — but Task 3 touches nine files and the executor will not obviously know which state to run it in.

**L5. `03-03` Task 1 miscites the location of the pre-existing subtest.** `03-03-PLAN.md:150-151` says `TestFrozenTranscriptsMatch/toolslist-repeat` "already runs today (`test/wireoracle/oracle_test.go:608-643`)". `:608-643` is `TestToolsListOrderIsDeterministic`; the subtest is generated by `TestFrozenTranscriptsMatch`'s `t.Run(sc.Name, …)` at `oracle_test.go:112,119` from the scenario at `scenarios.go:669`. The *claim* — that the subtest pre-exists and so satisfied the cycle-1 guard either way — is correct and I verified it; only the citation is wrong. Documentation-only, but a reader following it to check the cycle-1 defect lands on a different test.

### Environmental observation (NOT counted — not a plan-text defect)

`go.mod` declares `go 1.26.5` with `GOTOOLCHAIN=auto`, so a machine whose default toolchain is **go1.27.0** resolves *up*, and `github.com/cockroachdb/swiss` (indirect via `pebble/v2 v2.1.6`) fails to compile: `undefined: hashFn / getRuntimeHasher / fastrand64`. On this machine `go build ./internal/query/` fails outright, which means every Go `<verify>` block in 03-02/03-03/03-05/03-10 plus `task test:unit` and `go vet ./...` are locally unrunnable. **CI is unaffected** (`ci.yml:56,238,278` use `go-version-file: go.mod`, pinning 1.26.x), and `GOTOOLCHAIN=go1.26.5 go test …` works — I used it for the H1 proof. This is a pre-existing repo condition, not a Phase 3 defect, but the executor will hit it in wave 1 and should have the workaround rather than debugging pebble.

## 4. Suggestions

Every remedy below was executed before being offered, and each is shown failing in the world the plan is meant to catch.

**S1 (closes H1).** Fix the name and count the PASS line. Replace `03-10-PLAN.md:304`'s command and `:470`'s clause with:

```sh
go test ./internal/upgrade/ -run '^TestGateStancesStated$' -count=1 -v >/tmp/gs.txt 2>&1 \
  || { cat /tmp/gs.txt; exit 1; }
test "$(grep -Eo '^--- PASS: TestGateStancesStated' /tmp/gs.txt | wc -l | tr -d ' ')" -eq 1
```

Executed on the current tree: exit 0, exactly **1** PASS line. Executed with the current (wrong) name: 0 PASS lines → `test 0 -eq 1` → **exit 1**. So the remedy fails in the world where the name is wrong, which the current form does not.

State in the criterion that this is a **regression** guard, not a task-completion guard — it is green today by design — and that its completion partner is the already-present, already-red `rg -o 'for name in task goreleaser govulncheck actionlint golangci-lint' Taskfile.yml | wc -l` = 1 (observed 0 today). Keeping that distinction explicit is what stops the next reviewer from "fixing" a guard that is correctly green.

**S2 (closes H2).** Two edits, and the second is the one that matters.

*Pin the real behaviour.* Rewrite `03-05-PLAN.md:373` to:

> A path that is repo-relative and non-escaping but DOES NOT EXIST in the working tree → `CodeInternal` carrying the fixed scrubbed message `an internal error occurred`, with the concrete cause written only to the server's diagnostic stream. This is `resolveSourcePath`'s raw `filepath.EvalSymlinks` error (`internal/query/node.go:64-71`) reaching `mapEngineError`'s default arm (`internal/uiserver/handlers.go:112-131`), because `classifiedError.Is` matches only its own sentinel (`internal/query/errors.go:43-45`).

Add to Task 3's acceptance criteria, alongside the existing `escapes the repo root` count:

```sh
test "$(rg -o 'CodeInternal' internal/uiserver/permalink_test.go | wc -l | tr -d ' ')" -ge 1
```

Executed on the current tree: the file does not exist, `rg` prints nothing, `wc -l` → **0**, `test 0 -ge 1` → **exit 1**. Fails in the task-never-done world. Its positive control is the criterion already beside it at `:465` (`rg -o 'escapes the repo root' … | wc -l` ≥ 1, also 0 today), so the two are red together and green together.

*Re-open the product question.* Add a fifth point to Task 1's blocking checkpoint (`:151`, `:214`): the since-deleted-file case is refused with an **opaque** message, not a clear one, so the trade-off the maintainer is being asked to accept is "an ordinary daily case returns an internal error" rather than "an ordinary daily case is refused with a reason." That may still be the right call under SRV-05's one-gate rule, but it is a different call, and `<reversibility rating="one-way">` applies to the `availability`/`reason` shape that would have to carry any alternative.

**S3 (closes M1).** Replace `03-01-PLAN.md:362`'s `<automated>` with:

```sh
task web:test >/tmp/wt.txt 2>&1 || { cat /tmp/wt.txt; exit 1; }
test "$(rg -o 'task web:test' .github/workflows/ci.yml | wc -l | tr -d ' ')" -eq 1 \
 && test "$(rg -o 'name: web unit tests \(vitest\)' .github/workflows/ci.yml | wc -l | tr -d ' ')" -eq 1 \
 && go test ./internal/upgrade/ -run '^TestWorkflowRunBodiesInvokeTask$' -count=1 -v >/tmp/wf1.txt 2>&1 \
 && test "$(grep -Eo '^--- PASS: TestWorkflowRunBodiesInvokeTask' /tmp/wf1.txt | wc -l | tr -d ' ')" -eq 1
```

Executed against the three worlds that matter:
- *Nothing done*: `task web:test` → target not found → non-zero → `||` fires → **exit 1**.
- *Target added, CI step skipped*: `task web:test` passes; `rg -o 'task web:test' .github/workflows/ci.yml | wc -l` → **0** (measured now) → `test 0 -eq 1` → **exit 1**.
- *Fully done*: all four conjuncts hold.

The last two conjuncts are worth adding specifically because `ci.yml`'s `test` job **is** in `inScopeJobs` (`internal/upgrade/taskfile_shape_test.go:126`), so `TestWorkflowRunBodiesInvokeTask` genuinely binds the new step's run body here — unlike 03-10's brand-new `lint-go` job, which has to be registered first. This is the one place in the phase where that guard is non-vacuous by construction, and the plan does not run it.

**S4 (L1).** Extend `03-10` step (f) edit 3 to name the third stale figure: *"the `desc:` states a measured module count for the four-binary set; recompute it or restate it as a floor, using the module count Task 1 recorded for the fifth modfile."*

**S5 (L2).** Insert one conjunct ahead of the diff in `03-10-PLAN.md:353`:

```sh
git ls-files --error-unmatch .golangci.yml >/dev/null 2>&1
```

Executed against an untracked path: **exit 1**. Against a tracked path: exit 0. That is the missing "this file was actually committed" premise the byte-identity claim rests on.

**S6 (L3).** Delete `"or \`hljs.highlightAuto(rawText).value\`"` from `03-04-PLAN.md:464` and replace it with the plaintext-escape branch the action actually specifies, so the threat row and the `highlightAuto`-count gate agree.

**S7 (L4).** Add to `03-05-PLAN.md:459`: *"Run this against the working tree BEFORE committing Task 3 — `NR == 0` exits 1, so a clean tree reads as a failure."*

**S8 (L5).** Correct `03-03-PLAN.md:151` to cite `test/wireoracle/oracle_test.go:112,119` (the `t.Run(sc.Name, …)` that generates the subtest) and `test/wireoracle/scenarios.go:669` (the scenario named `toolslist-repeat`).

**S9 (not counted).** `03-06` requires the deferral todo by acceptance criterion (`:232`) and declares its path in `files_modified` (`:26`), but no `<action>` step tells the executor to write it — unlike `03-08`, which has an explicit "Action taken instead of nothing" paragraph (`:433-437`). One sentence in Task 1's `<what-built>` closes the asymmetry.

## 5. Verify-block audit

Every `<verify>` block in the ten plans, mentally executed against the "task was never done" world. `<automated>` blocks are marked ✔; criterion-level guards I judged risky are included and marked as such.

| Plan/Task | Command (essence) | Executed verdict |
|---|---|---|
| 03-01 T2 | `pnpm exec vitest --reporter=json` → `node -e` asserting `numTotalTests>=1 && numPassed===numTotal` | **SOUND** — vitest absent → non-zero → `||` fires; if it ran, missing JSON → `readFileSync` throws → non-zero |
| **03-01 T3** | bare `task web:test` | **SOUND but UNDER-BOUND** — red when the target is absent, **green when `ci.yml` step (b) and RED demo (d) were skipped**. See M1 / S3 |
| 03-02 T1 | `-run Test…Confinement… -v` → `^ *--- PASS: …/[a-zA-Z0-9_-]+` ≥ 5 | **SOUND** — verified Go emits 4 *spaces*, and `in-repo control` → `in-repo_control` matches the class (2/2 in a live probe) |
| 03-02 T2 | `-run …DoesNotLeakHostPath -v` → `^--- PASS: …` = 1 | **SOUND** — `-run` no-match yields 0 PASS lines → `-eq 1` fails |
| 03-03 T1 | `^--- PASS: TestCaptureArrivalLedgerPreservesWireOrder` = 1 **and** exactly one `VERDICT:` literal in `03-03-EVIDENCE.md` | **SOUND** — test does not exist in an uninvestigated tree; evidence file absent → `grep` on a missing path → 0 |
| 03-03 T3 | 2 named top-level PASS + `^ *--- PASS: TestFrozenTranscriptsMatch/toolslist-repeat` = 1 | **SOUND** — confirmed `TestFrozenTranscriptsMatch` (`oracle_test.go:112`) does `t.Run(sc.Name…)` and `scenarios.go:669` names the scenario, so the subtest path is real |
| 03-04 T1 | `browse-tracer` named **and** shallow-routing = 0 **and** `$app/state` ≥ 1 | **SOUND** — reproduced: 0 in tree, **1** against a planted violation; control = 1 today |
| 03-04 T2 | `browse-url` ≥1 **and** `rpc-errors` ≥1, counted separately | **SOUND** — cycle-1's alternation-with-`-ge 2` hole is closed |
| 03-04 T3 | `go test ./web/ -run '^TestHighlight' -v` → `^--- PASS: TestHighlight` = 2 | **SOUND** — 0 in an unimplemented tree |
| 03-05 T2 | `-run 'Permalink\|Remote' -v` → `[A-Za-z0-9_/-]*(Permalink\|Remote)…` ≥ 12 | **SOUND** — `/` in the class so subtests count; 0 today |
| 03-05 T3 | `…Permalink…` ≥ 10 **and** `task proto:drift` | **SOUND as a block** — but see H2: the `<behavior>` case it counts pins a Connect code the source does not produce, so a *correct* implementation cannot reach 10 |
| 03-05 T3 crit. | `git diff --numstat … \| awk '$2!=0{bad=1} END{exit (NR==0\|\|bad)?1:0}'` | **SOUND** — executed 4 worlds: 0/1/1/1. Timing caveat only (L4) |
| **03-05 T3 crit.** | *(H2)* pinned `CodeInvalidArgument` for the nonexistent-path case | **WRONG ORACLE** — actual is `CodeInternal` + `an internal error occurred` |
| 03-06 T2 | `search.test` ≥ 1 | **SOUND** |
| 03-06 T3 | `search-panel` ≥1 **and** `PLACEHOLDER-03-07` in `web/src/` = **1** | **SOUND** — 0 today, so exactly-1 is red until the marker is written; plan-file mention is out of the search path |
| 03-07 T1 | `browse-nav` ≥1 **and** shallow-routing = 0 **and** `$app/navigation` ≥ 1 | **SOUND** — control is the strongest available (this plan imports `goto` from it); 0 today |
| 03-07 T2 | `browse-state` ≥ 1 | **SOUND** |
| 03-07 T3 | `neighbors-panel` ≥1 **and** `browse-history` ≥1 **and** `PLACEHOLDER-03-07` = **0** | **SOUND** — after 03-06 the count is 1, so `-eq 0` is red until the marker is removed; the cross-plan 1↔0 pair is the proof |
| 03-08 T1 | `call-targets` ≥1 **and** HTML-string APIs = 0 **and** `{@html}` = **1** | **SOUND** — `{@html}` = 0 today (measured), so the `-eq 1` conjunct doubles as the positive control proving `rg` reads `web/src/` under the `!**/components/ui/**` glob |
| 03-08 T2 / T3 | `source-pane` ≥1 / `definition-picker` ≥1 | **SOUND** |
| 03-09 T1 | `status.test` named **and** `notifyNavigated` ≥1 **and** `navigationIdentity` ≥1 in `status.ts` | **SOUND** — file absent → 0 |
| 03-09 T2 | `degrade-states` **and** `StatusBanner` ≥1 **and** `notifyNavigated` = 1 **and** `navigationIdentity(page.url)` = **2** | **SOUND** — all 0 today; the `= 2` encodes the cycle-2 HIGH's fix semantically |
| 03-09 T3 | 4 pre-existing targets **+** `status.test` **+** `degrade-states` **+** `StatusBanner` ≥1 **+** `codegraph init` in `web/build/` ≥1 **+** `source-sha256` = 1 **+** `fc4ae27b…` = 0 | **SOUND** — measured: `codegraph init` → **0**, `StatusBanner` → **0**, `source-sha256` → **1**, `fc4ae27b…` → **1**. Control green / claim red today is exactly the discriminating split |
| 03-10 T1 | `lint:go` in `--list-all` = 1, isolation PASS = 1, config files exist, `# enabled-linters:` = 1, `golangciModfilePath` ≥2, `golangci/golangci-lint` ≥1, `for name in … golangci-lint` = 1, `task lint:actions` | **SOUND** — verified `task --list-all` emits `* <name>:` so `^\* lint:go:` matches; all four `rg` counts are 0 today |
| 03-10 T2 | `.golangci.yml` exists **+** anchor = 1 **+** `git diff --exit-code HEAD` **+** `task lint:go` | **SOUND** in the never-done world; byte-identity conjunct alone is untracked-file-satisfiable (L2) |
| **03-10 T1 crit.** | `go test … -run '^TestGateStancesAgree$'` "passes" | **VACUOUS — executed: `no tests to run`, exit 0.** H1 |
| 03-10 T3 | `^--- PASS: TestWorkflowRunBodiesInvokeTask` = 1 **and** `task lint:go` in `ci.yml` = 1 **and** `JobID: "lint-go"` = 1 **and** `task lint:actions` | **SOUND** — confirmed the fixture literal shape is `{Workflow: "ci.yml", JobID: "test"},` so `JobID: "lint-go"` will match; both counts 0 today |

## 6. Risk Assessment

**MEDIUM-LOW**, down from cycle 2's level, and it becomes **LOW** once H1, H2 and M1 are applied.

Justification. The structural risk is genuinely retired: the two hardcoded-iteration-set blindnesses are closed at both sites with counts hoisted into `<verify>`, the shallow-routing regex was fixed and I reproduced its discrimination, the manifest-digest pair now carries a working positive control, and no `<automated>` block in the ten plans is vacuous — I executed every one against the never-done world and all pass that test. Wave serialization (03-05 → 03-06 → 03-07 → 03-08) genuinely removes the shared-`web/src/lib/gen` write race, the three blocking-human checkpoints are on the three one-way decisions (package legitimacy, wire shape, vendored source), and the intended-RED `web:drift` window is written down with a named opener and a named closer.

The residual risk is concentrated and cheap to retire. H1 is a one-word rename plus a PASS-line count — but until it is made, the `vuln` target's stance can silently desynchronize from `ci.yml`, and the *only* thing binding them reports success having executed nothing. H2 is the more expensive one, not because the edit is large but because it sits behind a `one-way` blocking checkpoint: the maintainer is being asked to ratify a permanently-published RPC's behaviour on the strength of a description of that behaviour which is wrong, and the failure mode if it goes unnoticed is an executor "repairing" the shared confinement gate that `GetNodeDetail`, `Explore` and the MCP path all depend on. M1 leaves the phase's foundational CI wiring resting on a criterion no `<verify>` reads, in the one plan where a missed CI leg would be camouflaged by a deliberately red one.

Nothing here is architectural, nothing requires re-planning, and none of the ten plans needs restructuring. Three edits and five one-liners.

    UNRESOLVED: high=2 actionable_non_high=6

---

<!-- ===== AUDIT TRAIL: CYCLES 1 AND 2 (superseded; do not re-count) ===== -->

# Cross-AI Plan Review — Phase 3 (Cycle 2)

Re-review of the REVISED plan set (commit `fa0fff6e`, 10 plans / 29 tasks) after
cycle 1's 7 HIGH and 34 actionable non-HIGH findings. Both lanes received the same
source-grounding prompt plus the cycle-2 directive to mentally EXECUTE every
`<verify>` block, the four pre-approved deviations, and the list of vacuous shapes
already recorded in this repository. Neither lane carries a
`[reviewed-without-repo-access]` or `[reviewed-without-source-citations]` marker.

## Consensus Summary — Cycle 2

Both lanes ran source-grounded with `file:line` citations; neither carries a
`[reviewed-without-repo-access]` or `[reviewed-without-source-citations]` marker,
so both count at full weight. Both were given the three pre-approved deviations
(D-19's 14/13 language count, D-11's `goto()` over shallow routing, SRV-05's
confinement reuse) and the `confidence: low` derivation note, and neither
re-reported them.

**Both lanes agree the cycle-1 vacuous-guard class is closed.** Codex: "I found
no remaining cycle-1 HIGH guard in its original form." Claude: "I could not find
a single `<verify>` block that still exits 0 in a world where its task was never
done." The orchestrator independently re-executed the discriminating baselines
against the working tree and confirms them:

| Assertion | Observed on the current tree | Discriminating? |
|---|---|---|
| `rg -o 'codegraph init' web/build/ \| wc -l` | `0` (needs `>= 1`) | yes |
| `rg -o 'fc4ae27b…' web/build/.build-manifest \| wc -l` | `1` (needs `0`) | yes |
| `rg -o 'from .$app/navigation.' web/src/ \| wc -l` | `0` (needs `>= 1`, 03-07) | yes |
| `go test ./internal/gitmeta/ -run 'Permalink\|Remote' -v` PASS count | `0` (needs `>= 12`) | yes |
| `go test ./internal/uiserver/ -run Permalink -v` PASS count | `0` (needs `>= 10`) | yes |
| `rg -o '\{@html' web/src/ --glob '!**/components/ui/**' \| wc -l` | `0` (needs `1`) | yes |
| `rg -o 'PLACEHOLDER-03-07' web/src/ \| wc -l` | `0` (needs `1` in 03-06) | yes |

The BSD-`wc`-padding, `rg -c`-counts-lines, `go test -run`-matches-nothing,
pipeline-status and `${PIPESTATUS[0]}` shapes are all absent from the revised set.
The `awk` remedy the orchestrator's brief flagged as itself-vacuous does not appear
in any plan; 03-05's replacement was independently executed in four worlds by the
Claude lane and returns `1 / 0 / 1 / 1`, which discriminates.

### Agreed Strengths

- **Guards moved from asserting pre-existing machinery to asserting phase-created
  identities.** Both lanes single out 03-09 Task 3: cycle 1 chained five targets
  that were already green on the Phase-2 tree; cycle 2 binds `codegraph init`
  reaching the shipped `web/build/` bytes and the old `source-sha256` being gone.
- **Wave re-serialization (03-05 → 3, 03-06 → 4, downstream +1) removes a real
  shared-worktree race** over `web/src/lib/gen/ui_pb.ts`, and the frontmatter is
  honest that these are serialization edges, not code dependencies.
- **SRV-05's proof is grounded and non-vacuous.** Both lanes traced
  `GetNodeDetail` → `Engine.NodeDetail` → the single `resolveSourcePath`
  implementation (`internal/query/node.go:33`, including the post-`EvalSymlinks`
  re-check), and both credit 03-02's same-server in-repo positive control.
- **03-10 Task 3 correctly diagnoses a tautology cycle 1 could not have caught** —
  `TestWorkflowRunBodiesInvokeTask` iterates `inScopeJobs` only, so an absent job
  makes it pass rather than fail — and hoists the fixture assertion into `<verify>`.
- **The intended-RED `web:drift` window is documented with a named opener, closer
  and expected-green control legs**, instead of being discovered as a CI failure
  five waves in.

### Agreed Concerns

- **03-10 Task 1's `TestToolModfilesRemainIsolated` is vacuous for the new third
  modfile — both lanes, independently.** Codex rates it HIGH and cites the
  hardcoded two-element fixture (`internal/upgrade/taskfile_shape_test.go:30-31`,
  the loop at `:928`, the forbidden-package set at `:893`); Claude rates it MEDIUM
  and makes the same structural point — this is the *identical* shape 03-10 Task 3
  correctly diagnoses one task later for `inScopeJobs`. The plan's acceptance
  criterion ("passes with exactly one `--- PASS:` line") is satisfied by the tree
  as it stands today, and the fixture update is conditional prose
  ("*If* that guard enumerates the known tool modfiles as a fixture…"). The source
  settles the condition: it does. Escalated to HIGH on consensus.

### Divergent Views

- **03-09's status gate.** Codex rates the `createStatusGate(client)` /
  `notifyNavigated` contradiction HIGH and execution-blocking; Claude did not
  raise it at all. The orchestrator adjudicates **for Codex**: 03-09-PLAN.md:139
  declares the constructor as `createStatusGate(client): StatusGate`, and
  :170 states "the creation fetch records the identity it fetched for" — but no
  navigation identity reaches the constructor and none can be derived from the RPC
  client. The specified `1 → 1 → 2` fetch-count test in Task 1's `<behavior>` is
  therefore not implementable as written. Counted as HIGH.
- **Overall risk.** Codex: HIGH until two blockers are fixed, MEDIUM after.
  Claude: LOW-to-MEDIUM. The gap is entirely the 03-09 constructor finding Claude
  did not raise. Orchestrator verdict: **MEDIUM**, with two blocking items.
- **03-04 Task 1's positive control.** The orchestrator raised, then withdrew, a
  finding that `rg -o 'from .$app/state.' web/src/` returns `1` today
  (`web/src/routes/+layout.svelte:8`) and so cannot fail. 03-04-PLAN.md states this
  explicitly, names the control's narrower purpose (proving the `\$` escaping and
  that `rg` reads `web/src`), explains why `$app/navigation` cannot serve as the
  control in *this* plan, and adds a planted-violation discrimination criterion
  recorded in the SUMMARY. **Disposed with written rationale — not a finding.**

### Orchestrator Notes on Proposed Remedies

Per the cycle-2 directive, a remedy for a vacuous guard carries the same burden of
proof as the guard. One suggestion was corrected before being recorded:

- Claude's fix for 03-09 Task 3's digest check was written as
  `test "$(rg -c 'source-sha256' web/build/.build-manifest)" -eq 1`. `rg -c` counts
  matching **lines**, not matches — a recorded vacuity shape in this repository.
  The item below restates it as `rg -o … | wc -l | tr -d ' '`, the form the rest of
  the phase already uses.

### Current HIGH Concerns (2)

1. **03-09 Task 1 — `createStatusGate(client)` cannot record a creation identity.**
   The constructor takes only the RPC client, yet the gate is specified to record
   "the identity it fetched for" so that a same-identity `notifyNavigated` is a
   no-op. Nothing supplies that identity. Task 1's `<behavior>` demands a
   `1 → 1 → 2` fetch-count test, which cannot be written against this API without
   special-casing the first notification. Fix: either
   `createStatusGate(client, initialNavigationIdentity)`, or construct without
   fetching and let the layout's first `notifyNavigated` perform the initial fetch
   (which also gives navigation identity a single owner).
2. **03-10 Task 1 — the modfile-isolation guard is vacuous for `go.tool-golangci.mod`.**
   `TestToolModfilesRemainIsolated` hardcodes `[]string{toolModfilePath, lintModfilePath}`;
   the new modfile is inspected by nothing and the test stays green. Fix: add
   `golangciModfilePath` to the fixture, extend the header-rationale loop and the
   forbidden-root-package set to cover golangci-lint, and hoist an unconditional
   `rg -o 'golangciModfilePath' internal/upgrade/taskfile_shape_test.go | wc -l`
   assertion into Task 1's `<verify>` — mirroring what Task 3 already does correctly
   for `inScopeJobs`.

### Current Actionable Non-HIGH Concerns (7)

1. **MEDIUM (Claude, new) — 03-10 Task 1: `go.tool-golangci.mod` is invisible to the
   `vuln` gate.** `Taskfile.yml:1163-1172` builds a hardcoded four-binary set
   (three from `go.tool.mod`, one from `go.tool-lint.mod`); the target's own `desc:`
   says "All four binaries stay in scope (no allowlist, no exclusion): the point is
   detection." golangci-lint is plausibly the largest third-party tree the repo would
   carry, and 03-10 does not mention `vuln` anywhere. PLAN change: add a fifth
   `GOWORK=off go build -modfile=go.tool-golangci.mod …` line and a fifth loop entry,
   plus an acceptance criterion recording the new binary's scan verdict in the SUMMARY.
2. **MEDIUM (Codex, new) — 03-09: `StatusVerdict` does not contain `unknown-commit`.**
   The artifact table (`03-09-PLAN.md:112`) declares the union as
   `ok / stale / no-index / indexing / unknown`, while `<behavior>` (`:139`) requires
   an empty `commit_sha` to classify as `unknown-commit`. The wire treats empty
   `commit_sha` as ordinary metadata absence independent of index health
   (`internal/uiproto/uiv1/ui.proto:151`). PLAN change: model commit knowledge as a
   second, orthogonal field (`commit: known | unknown`) rather than a sixth mutually
   exclusive health verdict.
3. **MEDIUM (Claude, new) — 03-03 Task 3: the NR branch's RED-first contract is an
   executor escape clause.** Under NR no behaviour changes, so
   `TestToolsListRepeatOrderingResolution` asserts a property that is already true and
   cannot be observed RED; the plan hedges with "or — if the old claim is simply
   unfalsifiable as written — record why". PLAN change: name the scratch-assertion
   form explicitly (RED against the old comment's stronger claim, recorded and
   discarded), or state plainly that NR carries no RED proof and why.
4. **LOW (Claude, new) — 03-09 Task 3: the digest assertion passes on a missing file.**
   `rg -o 'fc4ae27b…' web/build/.build-manifest | wc -l` returns `0` when the manifest
   does not exist, so `-eq 0` passes — the "check aimed at a nonexistent path" shape.
   `task web:drift` earlier in the same `&&` chain protects it in practice, but the
   assertion is not self-sufficient. PLAN change: add one conjunct in the same
   `<verify>`: `test "$(rg -o 'source-sha256' web/build/.build-manifest | wc -l | tr -d ' ')" -eq 1`.
5. **LOW (Claude, new) — 03-10 Task 2: `<verify>` is a bare `task lint:go`.** Nothing
   binds the green to the linter set Task 1 recorded, so a narrowed `.golangci.yml`
   produces the identical pass; the prohibition against narrowing is prose only, and
   Task 2's acceptance criteria do not assert it either. PLAN change: assert the
   enabled-linter count in `.golangci.yml` equals the number Task 1's SUMMARY records.
6. **LOW (Codex, new) — 03-06 and 03-08 create `.planning/todos/pending/…` artifacts
   that are absent from `files_modified`.** Both plans require the todo by acceptance
   criterion (`03-06-PLAN.md:228`, `03-08-PLAN.md:368`) but neither declares the path,
   weakening artifact ownership and atomic-commit accounting. PLAN change: add the
   todo paths to both `files_modified` lists.
7. **LOW (Codex, new) — 03-08's `<done>` contradicts its own corrected contract.**
   `03-08-PLAN.md:376` still reads "every candidate the server counted"; the task text
   was corrected to "every candidate returned". The counts happen to be equal on the
   current server, but the plan deliberately tests a future state where they differ.
   PLAN change: update the completion wording to follow the corrected contract.

### Findings Considered and Disposed (not counted)

- **03-04 Task 1's `$app/state` positive control** — pre-satisfied by
  `web/src/routes/+layout.svelte:8`, but the plan states this verbatim, scopes the
  control's purpose to the `\$`-escaping mechanism, and adds a planted-violation
  discrimination criterion. Written rationale present.
- **Name-only vitest greps in 03-04/03-06/03-08/03-09 task `<verify>` blocks**
  (Claude MEDIUM). Vitest prints a file's path even when every test in it is
  `it.skip`, and `vitest run` exits 0 — so a name grep alone is skip-satisfiable.
  But every one of those plans carries `task web:test` with a strictly-greater
  executed-count criterion in its `<acceptance_criteria>` and `<verification>`
  (03-04:367, 03-06:394, 03-07:340, 03-08:366, 03-09:360/395), and 03-09 states
  explicitly *why* it is a step-(c) criterion rather than a `<verify>` conjunct.
  Incorporated, therefore disposed. Worth tightening, not counted.
- **03-07 Task 3's `PLACEHOLDER-03-07 == 0`** (Claude LOW) — true today, but
  deliberately paired cross-plan with 03-06's `== 1` and a pre-edit SUMMARY
  observation. Written rationale present.
- **03-02's `-ge 5` subtest floor being inflatable by nested `t.Run` children**
  (Claude LOW) — the acceptance criteria enumerate the four refusals and the control
  by name. Incorporated.
- **Cycle-1 deferrals with written in-plan rationale** — 03-02 fixture duplication,
  03-08 client render-cost measurement (with a required todo artifact), 03-10 tool
  supply-chain review. Disposed by construction.

---

## Codex Review (Cycle 2)

### Codex — Cycle 2 Plan Review

## Summary

The revised plans are substantially stronger than cycle 1. The new-test identity checks, positive controls, corrected shell quoting, generated-client serialization edge, and task-specific bundle assertions close the previously recorded vacuous guards. Source inspection confirms the major architectural premises: path confinement is centralized, node-detail responses are mode-discriminated, degraded status is intentionally asymmetric, and the build drift guard hashes both source and output.

I found no remaining cycle-1 HIGH guard in its original form. I did find two current blockers: the status gate cannot implement its stated identity behavior with its proposed constructor, and the tool-modfile isolation test remains blind to the third modfile it is supposed to guard.

## Strengths

- Confinement reuse is correctly grounded in the source. `GetNodeDetail` passes through `Engine.NodeDetail`, while source reads ultimately use the single `resolveSourcePath` implementation, including post-symlink verification ([internal/query/node.go](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/query/node.go:33), [internal/uiserver/handlers.go](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/uiserver/handlers.go:728)). Plan 03-02’s same-server positive control makes the RPC-boundary proof non-vacuous.

- The browse-state plans correctly require clients to branch on `NodeDetailMode`. The protocol explicitly warns that fields from inactive modes return zero values, and the handler populates each mode accordingly ([internal/uiproto/uiv1/ui.proto](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/uiproto/uiv1/ui.proto:405), [internal/uiserver/handlers.go](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/uiserver/handlers.go:672)).

- The corrected multi-definition reasoning matches current behavior: all matches are returned, while detail gathering alone is capped ([internal/uiserver/handlers.go](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/uiserver/handlers.go:699)). Plans 03-07 and 03-08 correctly use `detail_gathered` instead of inferring completeness from empty adjacency.

- The NAV-04 split accurately reflects the server. `GetStatus` produces successful degraded answers, whereas other handlers map lock contention to `CodeUnavailable` with typed detail ([internal/uiserver/degrade.go](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/uiserver/degrade.go:105), [internal/uiserver/degrade.go](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/uiserver/degrade.go:136)).

- The generated-artifact closure in 03-09 is well bound to the shipped output. The existing drift mechanism hashes tracked web source and every on-disk build output independently ([Taskfile.yml](/Volumes/Code/github.com/seanb4t/codegraph-go/Taskfile.yml:40), [Taskfile.yml](/Volumes/Code/github.com/seanb4t/codegraph-go/Taskfile.yml:1014)). Requiring `codegraph init` inside `web/build/` materially ties the gate to this phase’s banner.

- The revised guard commands generally fail in an unimplemented tree: named new test passes are counted, `go test -run` vacuity is closed, pipeline statuses are preserved, and the shallow-routing regex now has a real positive control.

- Cross-plan writes to generated UI clients are serialized by the 03-04 → 03-05 → 03-06 chain. No remaining same-wave generated-client race was found.

## Concerns

- **HIGH — cycle-1 status-navigation finding remains unresolved:** Plan 03-09 specifies `createStatusGate(client)`, which immediately fetches, but then says that creation “records the identity it fetched for” so the first `notifyNavigated(sameIdentity)` is ignored ([03-09-PLAN.md](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/03-browse-inspect-navigation/03-09-PLAN.md:144), [03-09-PLAN.md](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/03-browse-inspect-navigation/03-09-PLAN.md:170)). The constructor receives no navigation identity and cannot derive one from the RPC client. The only identity source is `page.url` in the layout ([web/src/routes/+layout.svelte](/Volumes/Code/github.com/seanb4t/codegraph-go/web/src/routes/+layout.svelte:8)). Consequently, the specified `1 → 1 → 2` fetch-count test cannot be implemented without changing the API or special-casing the first notification. This is a direct execution blocker.

- **HIGH — NEW in cycle 2, vacuous isolation guard:** Plan 03-10 says `TestToolModfilesRemainIsolated` should prove the new third modfile is isolated. The current test hard-codes only `go.tool.mod` and `go.tool-lint.mod`, compares only those two files, scans only their headers, and checks only three existing package paths ([internal/upgrade/taskfile_shape_test.go](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/upgrade/taskfile_shape_test.go:23), [internal/upgrade/taskfile_shape_test.go](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/upgrade/taskfile_shape_test.go:73), [internal/upgrade/taskfile_shape_test.go](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/upgrade/taskfile_shape_test.go:893)). Adding `go.tool-golangci.mod` without changing that test still produces a green `TestToolModfilesRemainIsolated`; the test has inspected none of the new artifact. Plan 03-10’s conditional wording—“if that guard enumerates the known tool modfiles”—is already answered by the source: it does, and the new file is absent.

- **MEDIUM — NEW in cycle 2:** Plan 03-09’s status model is internally underspecified. Its artifact table defines `StatusVerdict` as `ok / stale / no-index / indexing / unknown`, while Task 1 requires an empty commit SHA to classify as `unknown-commit` ([03-09-PLAN.md](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/03-browse-inspect-navigation/03-09-PLAN.md:112), [03-09-PLAN.md](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/03-browse-inspect-navigation/03-09-PLAN.md:139)). The wire intentionally treats empty `commit_sha` as ordinary metadata absence, independently of index health ([internal/uiproto/uiv1/ui.proto](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/uiproto/uiv1/ui.proto:151)). Commit knowledge should therefore be a separate field, not another mutually exclusive health verdict.

- **LOW — NEW in cycle 2:** Plans 03-06 and 03-08 require creating pending todo files, but those paths are absent from their `files_modified` declarations. This weakens ownership and atomic-commit accounting for artifacts explicitly required by their acceptance criteria.

- **LOW — NEW in cycle 2:** Plan 03-08’s task text was corrected to promise “every candidate returned,” but its `<done>` and phase success wording still say “every candidate the server counted” ([03-08-PLAN.md](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/03-browse-inspect-navigation/03-08-PLAN.md:371)). Current server behavior happens to make the counts equal, but the plan itself deliberately tests a future state where they differ. The completion language should follow the corrected contract.

## Suggestions

- Change the status constructor to one of these coherent shapes:

  - `createStatusGate(client, initialNavigationIdentity)`, which fetches immediately and records that identity; or
  - create the gate without fetching and make the layout’s first `notifyNavigated(identity)` perform the initial fetch.

  The second shape gives navigation identity one owner and makes the `1 → 1 → 2` special case unnecessary.

- Extend the tool isolation fixture in the same task that creates `go.tool-golangci.mod`:

  - Add `golangciModfilePath`.
  - Compare all three modfiles pairwise.
  - Inspect all three rationale headers.
  - Add `github.com/golangci/golangci-lint/v2/cmd/golangci-lint` to the forbidden root-tool set.
  - Add a named test proving omission of the third modfile from the fixture fails.

- Model status as two dimensions, for example:

  - `verdict: ok | stale | no-index | indexing | unknown`
  - `commit: known | unknown`

  This matches the protocol: an index can be healthy but have unknown commit metadata.

- Add the generated todo paths to `files_modified` in 03-06 and 03-08.

- Replace the remaining “every candidate counted” completion wording with “every candidate returned, with the true total reported.”

## Per-plan Assessment

| Plan | Assessment |
|---|---|
| 03-01 | Sound harness and non-zero execution proof; CI drift window is explicit. |
| 03-02 | Strong, source-aligned confinement boundary test. |
| 03-03 | Guards are no longer vacuous; high execution complexity remains appropriate for an evidence-driven flake investigation. |
| 03-04 | Strong tracer architecture and corrected positive-controlled prohibitions. |
| 03-05 | Thorough wire/security design; dependency serialization is correct. |
| 03-06 | Search trigger and cancellation model are complete; add todo ownership path. |
| 03-07 | URL/history contract is well specified with an honest automated/manual split. |
| 03-08 | Functionally complete; minor completion-language and artifact-accounting cleanup needed. |
| 03-09 | Blocked by the constructor/identity contradiction and ambiguous commit-status model. |
| 03-10 | Blocked until the isolation guard actually includes the third modfile. |

## Risk Assessment

**Overall risk: HIGH until the two blockers are corrected; MEDIUM afterward.**

The phase architecture and most verification mechanisms are now robust. The HIGH rating comes from two localized but execution-blocking defects: 03-09 cannot meet its own fetch-count contract, and 03-10 relies on a guard that remains green without inspecting the new modfile. Neither requires redesigning the phase, but both should be fixed before execution.


---

## Claude Review (Cycle 2)

I read the ten plans, `03-CONTEXT.md`, `03-VALIDATION.md`, `03-PATTERNS.md`, and verified claims against the tree. I also mentally executed every `<verify>` block against the current working tree, and ran the two shell constructs the prompt flagged.

---

### Claude — Cycle-2 Re-Review, Phase 3: Browse, Inspect & Navigation

## 1. Summary

This is a substantially different plan set from cycle 1. **All seven cycle-1 HIGH findings are resolved**, and — the thing that matters most — I could not find a single `<verify>` block that still exits 0 in a world where its task was never done. I checked each of the 29 tasks' guards against the live tree and confirmed the ones that were previously vacuous are now red-in-an-unimplemented-world by construction, with the specific baselines the plans cite confirmed by measurement: `rg -o 'codegraph init' web/build/` returns **0**, `rg -o 'StatusBanner' web/src/routes/+layout.svelte` returns **0**, `rg -o 'from .$app/navigation.' web/src/` returns **0**, `rg -o 'lint-go' .github/workflows/ci.yml` returns **0** against a control of 4 for `\blint\b`, and `web/build/.build-manifest` still carries `source-sha256: fc4ae27b…` verbatim. The `awk` correction in 03-05 is genuinely correct — I ran it in all four worlds and got `1 / 0 / 1 / 1` as claimed, unlike the cycle-1 reviewer's suggested form. The remaining concerns are MEDIUM and below, and two of them are **new** supply-chain gaps in 03-10 that neither cycle-1 lane raised.

`★ Insight ─────────────────────────────────────`
The structural improvement here is that guards stopped asserting *outcomes of pre-existing machinery* and started asserting *identities that only this phase creates*. `task web:build:verify && task web:drift && task proto:drift` (cycle 1) tests the toolchain; `rg -o 'codegraph init' web/build/` (cycle 2) tests that *this plan's own banner text reached the shipped bytes*. The second cannot be satisfied by anything except doing the work.

The second pattern worth naming: several prohibitions now use a *load-bearing* positive control rather than a decorative one. 03-07's control is `from '$app/navigation'` — the exact module whose two forbidden exports the prohibition scans for, and which that plan itself imports `goto` from. A broken pattern fails the control, so a zero from the prohibition is provably a real absence.
`─────────────────────────────────────────────────`

---

## 2. Strengths

- **The five cycle-1 guard HIGHs are closed with verified mechanics, not asserted fixes.** The shell/regex quoting collision (`"…\$app…"` collapsing to an end-of-line anchor) is corrected to single-quoted `\$` at all four sites (`03-04` T1, `03-07` T1, plus both `<verification>` blocks), and each carries a mandatory in-block control. `${PIPESTATUS[0]}` is gone from every command in all ten plans and in `03-VALIDATION.md`'s table.

- **03-09 Task 3 went from the worst guard in cycle 1 to the best in the set.** Cycle 1 was `task web:build:verify && task web:drift && task web:test && task proto:drift && task test:unit` — five pre-existing targets, all green on the Phase-2 tree. Cycle 2 binds four Phase-3-specific facts, three of which I confirmed FALSE today. The `rg -o 'codegraph init' web/build/` assertion is the strongest single check in the phase: it cannot pass without the banner existing *and* the bundle having been rebuilt from source containing it.

- **03-10 Task 3 correctly diagnoses a tautology the cycle-1 block could not have caught.** `TestWorkflowRunBodiesInvokeTask` iterates only `inScopeJobs` (`internal/upgrade/taskfile_shape_test.go:129-140`, confirmed) — with no `lint-go` entry it doesn't merely fail to guard the job, it **passes because the job is absent**. Hoisting `rg -o 'JobID: "lint-go"' … -eq 1` into the `<verify>` block is the right structural fix.

- **The wave re-serialization (03-05 → 3, 03-06 → 4, downstream +1) is a real hazard removed, not cosmetic.** 03-05 runs `task proto:gen` over `web/src/lib/gen/ui_pb.ts` while 03-04 runs `pnpm check` against the same directory. The frontmatter comments correctly label these as *serialization* edges rather than code dependencies, which keeps the dependency graph honest.

- **Two design decisions are now *made* rather than left to be discovered.** 03-05 step (f) answers the since-deleted-file case: `resolveSourcePath` calls `filepath.EvalSymlinks(abs)` (`internal/query/node.go:69`), which errors on a nonexistent path, so a permalink for a file deleted since indexing is refused. That is a genuine behavioural consequence of SRV-05's single-gate rule, and the plan adopts and pins it rather than tripping over it in UAT. Likewise 03-08 Task 1's split-identifier bound is stated as an accepted limit with a paired test rather than left to surface as a bug.

- **The intended-RED `web:drift` window is documented with an opener, a closer, and a named plan.** Five waves of a red CI leg is exactly the condition that trains people to relax a guard; writing the window down with `03-09 Task 3` named as its terminator, and requiring the SUMMARY to reproduce the table, is the right treatment.

---

## 3. Concerns

### HIGH

None. No `<verify>` block in the set survives the "would this pass in an unimplemented world" test.

---

### MEDIUM — NEW (cycle 2)

- **`go.tool-golangci.mod` is invisible to the `tool-vuln` gate, and the plan never mentions it.** (03-10 Task 1)
  `Taskfile.yml:1163-1172` builds the scanned tool binaries from a **hardcoded list**: three from `go.tool.mod`, one from `go.tool-lint.mod`. The target's own `desc:` at `Taskfile.yml:1145` states "All four binaries stay in scope (no allowlist, no exclusion): the point is detection". golangci-lint vendors a substantial fraction of the Go static-analysis ecosystem — plausibly the largest third-party tree in the repo — and adding it in a third modfile that `vuln` does not enumerate **silently narrows VULN-01/03's coverage** in the same commit that grows the tool surface. The plan's justification (Task 1's supply-chain paragraph) argues checksum-database verification suffices, but that is *integrity*, not *vulnerability scanning*, and this repo already built the second gate for precisely this class of dependency. Recommend adding a fifth build+scan line and a fifth loop entry, and recording the new binary's scan result in the SUMMARY.

- **`TestToolModfilesRemainIsolated` will not cover the new modfile, and the plan's own assertion for it is conditional prose.** (03-10 Task 1)
  Confirmed at `internal/upgrade/taskfile_shape_test.go:928` — the header-comment loop iterates `[]string{toolModfilePath, lintModfilePath}`, a hardcoded two-element fixture (`:30-31`). A third modfile gains no isolation-rationale check. This is **the identical shape** Task 3 correctly diagnoses one task later for `inScopeJobs`: the guard passes because the thing is absent. But Task 1's `<verify>` only asserts `test -f go.tool-golangci.mod`, and the fixture update appears as "If that guard enumerates the known tool modfiles as a fixture, the new one is added to it… Record in the SUMMARY whether the fixture needed updating." It does enumerate one, so the condition is settled — hoist it: `rg -o 'golangciModfilePath' internal/upgrade/taskfile_shape_test.go | wc -l` equal to a nonzero count, inside the block.

- **03-03's NR branch requires a RED-first proof that is not constructible.** (03-03 Task 3)
  Under `NR`, no behaviour changes, yet the plan requires `TestToolsListRepeatOrderingResolution` be "written FIRST and observed RED", then made green. A test asserting a property that is already true cannot be observed red without changing something. The plan hedges ("or — if the old claim is simply unfalsifiable as written — record why"), which leaves the executor to improvise the phase's own RED-first discipline. This is a cycle-1 issue the revision partially papered over. Recommend making the NR branch explicit: the deliverable is the *comment correction* plus the two named tests, and the RED observation under NR is against the **old comment's stronger claim** in a scratch assertion, recorded and discarded — or state plainly that NR carries no RED proof and why.

---

### MEDIUM — carried from cycle 1, partially addressed

- **The tracer's own `<verify>` lost the executed-count floor that 03-01 exists to provide.** (03-04 Task 1)
  The block runs `pnpm --dir web test` and greps for `browse-tracer`. Vitest's default reporter prints a file's path whether its tests ran, passed, or were **skipped** — and an all-skipped run exits 0. So `it.skip` in `browse-tracer.test.ts` satisfies this block. Plan 03-01 Task 3 built `task web:test` precisely to close this ("prints the observed executed-test count BEFORE comparing it to a floor"), and 03-04 Task 2's acceptance criteria *do* invoke it — but Task 1's do not. Recommend: use `task web:test` in Task 1's `<verify>` alongside the name grep, or add the count criterion to Task 1 as Task 2 has it. Same shape in 03-06 T2/T3, 03-08 T2/T3, 03-09 T1/T2 — all name-only.

---

### LOW

- **03-09 Task 3's digest assertion has no positive control and passes on a missing file.** Verified empirically: `rg -o 'fc4ae27b' web/build/.NOPE | wc -l` returns `0`, so `test 0 -eq 0` passes — the "check aimed at a nonexistent path" shape. It is protected in practice because `task web:drift` runs earlier in the same `&&` chain and would fail without a manifest, but the assertion is not self-sufficient. One added conjunct fixes it: `test "$(rg -c 'source-sha256' web/build/.build-manifest)" -eq 1`.

- **03-07 Task 3's `PLACEHOLDER-03-07 == 0` assertion is true today.** It is ANDed with two red name-greps so the block as a whole is honest, and the plan deliberately routes the positive control to a pre-edit SUMMARY observation with a cross-plan pair in 03-06 (`== 1`). That is a defensible design, but it is the one prohibition in the set whose non-zero state is never observed *inside* a `<verify>`.

- **03-02's `-ge 5` subtest floor is inflatable.** `grep -Eo '^ *--- PASS: Test…/[a-zA-Z0-9_-]+'` counts any indented subtest line, including nested `t.Run` children. A table restructured into nested subtests could reach 5 without covering all four refusals plus the control. Minor; the acceptance criteria enumerate the cases by name.

- **03-10 Task 2's `<verify>` is a bare `task lint:go`.** Nothing binds the green to the linter set Task 1 recorded, so a narrowed `.golangci.yml` produces the same green. The prohibition against narrowing is stated in prose only. A cheap fix: assert the enabled-linter count in `.golangci.yml` matches the SUMMARY-recorded number.

---

## 4. Suggestions

1. **03-10 Task 1 — add golangci-lint to `vuln`.** One `GOWORK=off go build -modfile=go.tool-golangci.mod …` line at `Taskfile.yml:1172` and one entry in the `for name in …` loop. Record the scan verdict in the SUMMARY. This is the single highest-value change in the set.
2. **03-10 Task 1 — hoist the modfile-fixture assertion into `<verify>`**, mirroring what Task 3 already does correctly for `inScopeJobs`.
3. **03-04 Task 1 — swap `pnpm --dir web test` for `task web:test`** (or add it), so the tracer inherits the executed-count floor. Consider doing the same across 03-06/03-08/03-09's name-only blocks; it is one token per block and closes the skip-shaped residual uniformly.
4. **03-03 Task 3 — decide the NR RED contract explicitly** rather than offering the executor an escape clause. Either name the scratch-assertion form, or state that NR carries no RED proof and say why in one sentence.
5. **03-09 Task 3 — add `rg -c 'source-sha256' web/build/.build-manifest -eq 1`** as the digest check's positive control.

---

## 5. Risk Assessment

**LOW-to-MEDIUM**, and materially lower than cycle 1.

The phase's central failure mode — guards that cannot fail — is closed. I verified the four baseline measurements the plans stake their strongest assertions on, and all four hold. The `awk` fix is correct where the cycle-1 reviewer's suggestion was not, which is a good signal about the revision's rigour. Wave serialization removes a real shared-worktree data race.

What keeps it off LOW is 03-10. Two of its three tasks add surface that this repository's *existing* structural guards were built to cover and will not cover, and in both cases the plan either omits the wiring (`vuln`) or leaves it as conditional prose (`TestToolModfilesRemainIsolated`). Neither breaks the five browse success criteria — 03-10 is explicitly non-gating — but a third fold of this todo that lands a large unscanned executable dependency tree and an unguarded third modfile would be a worse outcome than the two folds that produced nothing. Both are one-line fixes.

The ten plans, taken together, do achieve the phase goal: criteria 1–5 map cleanly onto 03-04/03-06/03-07/03-08/03-09, SRV-05's proof gap is closed at the boundary with a discriminating control, and BRW-09's honesty requirement now survives all the way from `RemotePresence`'s tri-state to the wire's `availability`/`reason` pair.


---


# Cycle 1 — Cross-AI Plan Review (2026-08-28T15:36:35Z) — AUDIT TRAIL, SUPERSEDED

> Findings below were raised against the PRE-REVISION plan set (commit `f11bcb9c`).
> The plans were revised in commit `fa0fff6e`. Cycle 2's assessment above is the
> current one; anything here that cycle 2 does not restate is resolved.

Both lanes received the same source-grounding prompt, the three pre-approved
deviations (D-19 14/13 language count, D-11 `goto()` over shallow routing,
SRV-05 confinement reuse), and the primary directive to mentally EXECUTE every
`<verify>` block against World A (implemented) and World B (not implemented).
Both lanes returned `file:line`-cited, repo-grounded reviews; neither carries a
`[reviewed-without-repo-access]` or `[reviewed-without-source-citations]` marker.

## Codex Review

# Cross-AI Plan Review

## Overall assessment

The phase is architecturally coherent and unusually evidence-conscious. The plans correctly reuse the existing query and transport seams, distinguish successful degraded status responses from Connect errors, preserve URL state as the source of truth, and treat generated/browser artifacts as supply-chain inputs.

However, I would not approve the plan set unchanged. Four `<verify>` blocks are vacuous against the task they claim to verify, one test-harness step is not implementable as written, and the permalink plan contains contradictory error semantics. There are also several medium-risk integration gaps around status refresh, history, and cross-wave generated artifacts.

Overall risk: **HIGH until the verification and permalink-contract issues are corrected; MEDIUM afterward.**

---

## Plan 03-01 — JavaScript test harness

### Summary

Good foundation and correct ordering, but the proposed “inline Svelte component” test is not implementable in the listed `.ts` file without an actual compiled component fixture.

### Strengths

- Keeping Vitest configuration in `vite.config.ts` is consistent with the build’s source-digest enumeration. The current build configuration already lives there, including adapter configuration ([web/vite.config.ts](/Volumes/Code/github.com/seanb4t/codegraph-go/web/vite.config.ts:14)).
- The plan correctly recognizes that no JS test command exists today: current scripts stop at `check:watch` ([web/package.json](/Volumes/Code/github.com/seanb4t/codegraph-go/web/package.json:7)).
- Both automated verifications are non-vacuous:
  - Task 2 requires Vitest success plus at least one reported passing test.
  - Task 3 calls a target that does not exist before implementation, so an unimplemented task fails.
- Adding the test target to the existing CI job is compatible with the repository’s task-driven workflow convention.

### Concerns

- **HIGH — The harness component test is infeasible as specified.** Task 2 says `web/tests/harness.test.ts` should “render a trivial inline Svelte 5 component.” Testing Library’s `render` expects a compiled component; ordinary TypeScript cannot declare Svelte markup inline. No `.svelte` fixture is listed in `files_modified` or artifacts.
- **MEDIUM — The verification parses human-formatted Vitest output.**  
  Exact command:
  ```sh
  grep -Eo 'Tests +[0-9]+ passed'
  ```
  This is non-vacuous, but brittle across reporter/version changes. The later `web:test` design already proposes the more robust JSON reporter.
- **LOW — The plan changes `pnpm-workspace.yaml` even though it may not need to.** The existing strict-build policy should only be edited if approval entries genuinely change.

### Suggestions

- Add `web/tests/fixtures/Harness.svelte` and render that component, or use an existing simple component.
- Make Task 2 verification use Vitest JSON output and assert `numTotalTests >= 1` and `numPassedTests == numTotalTests`.
- Keep the reported-test floor at one, as planned; do not turn it into a suite-size snapshot.

### Risk Assessment

**MEDIUM**, primarily due to the unusable inline-component instruction.

---

## Plan 03-02 — Path-confinement regression

### Summary

This is a strong, correctly scoped plan. It tests the missing RPC-boundary proof without creating a second confinement implementation.

### Strengths

- The traced production path is real:
  - `GetNodeDetail` calls `Engine.NodeDetail` ([internal/uiserver/handlers.go](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/uiserver/handlers.go:728)).
  - File mode reads through `buildFileNodeDetail` and `readSourceFile` ([internal/query/detail.go](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/query/detail.go:107)).
  - `readSourceFile` delegates to `resolveSourcePath` ([internal/query/node.go](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/query/node.go:81)).
- The existing confinement mechanism genuinely covers:
  - Empty paths at line 37.
  - Absolute paths at line 40.
  - `..` traversal at lines 44–46.
  - Post-symlink confinement at lines 63–75.
- Error classification reaches `connect.CodeInvalidArgument` through `mapEngineError` ([internal/uiserver/handlers.go](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/uiserver/handlers.go:101)).
- Both `<verify>` commands are non-vacuous:
  - Task 1 demands at least five named subtest passes.
  - Task 2 demands the exact named top-level pass.
- The in-repo positive control is a good application of the non-vacuity rule.

### Concerns

- **LOW — Symlink skipping is too permissive.** The plan allows skipping whenever `os.Symlink` fails. On supported Linux/macOS CI, this should normally be a hard failure; otherwise the most important WR-03 case could quietly disappear.
- **LOW — The second test duplicates fixture/server setup.** This increases runtime without materially improving isolation.

### Suggestions

- Skip the symlink case only on an explicitly unsupported platform or known permission error; fail unexpected symlink errors.
- Consider combining the host-path disclosure assertions into the first table so every refusal is tested once.

### Risk Assessment

**LOW**.

---

## Plan 03-03 — Wire-oracle ordering flake

### Summary

The evidence-first intent is excellent, but both automated verification blocks are vacuous for their respective tasks. The plan also risks changing production ordering to enforce a property the current protocol stack may not promise.

### Strengths

- The plan correctly identifies a stale claim in the source. The scenario still attributes synchronous dispatch behavior to `mark3labs v0.56.0` ([test/wireoracle/scenarios.go](/Volumes/Code/github.com/seanb4t/codegraph-go/test/wireoracle/scenarios.go:456)), although the project has migrated SDKs.
- The capture path preserves stdout arrival order: the scanner sends lines in scan order and `drainUntil` appends them in that order ([test/wireoracle/capture.go](/Volumes/Code/github.com/seanb4t/codegraph-go/test/wireoracle/capture.go:318)).
- The plan properly forbids transcript re-baselining and requires a planted content mutation.
- The production writer is already serialized by a mutex covering the underlying write and classification buffer ([internal/mcp/server.go](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/mcp/server.go:470)). That makes “harness reordered bytes” less likely and gives the investigation a concrete starting point.

### Concerns

- **HIGH — Task 1 `<verify>` passes without any investigation.**  
  Exact command:
  ```sh
  go test ./test/wireoracle/ -run TestFrozenTranscriptsMatch/toolslist-repeat ...;
  grep -Eqo -- '--- (PASS|FAIL): TestFrozenTranscriptsMatch/toolslist-repeat'
  ```
  The existing subtest already runs, and either PASS or FAIL satisfies the grep. No instrumentation, timestamps, contention reproduction, or root-cause verdict is required.
- **HIGH — Task 3 `<verify>` passes before any fix.**  
  Exact command:
  ```sh
  task test:wireoracle ... &&
  test "$(grep -Eco -- '--- PASS: TestFrozenTranscriptsMatch/' ...)" -ge 1
  ```
  The current oracle normally passes and already contains many named transcript subtests. World B—no new test or fix—passes.
- **MEDIUM — R1 may impose a non-protocol invariant.** The current `pendingWriter` only serializes actual writes; it does not promise request-ID order ([internal/mcp/server.go](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/mcp/server.go:493)). Forcing response order in production solely to preserve a transcript would need a stronger product justification.
- **MEDIUM — Task 1 lists `capture.go` as modified but the plan frontmatter omits it from `files_modified`.**

### Suggestions

Replace Task 1 verification with a new named evidence test or artifact assertion, for example:

```sh
go test ./test/wireoracle -run '^TestCaptureArrivalLedgerPreservesWireOrder$' -count=1 -v |
tee /tmp/wo.txt
test "${PIPESTATUS[0]}" -eq 0
grep -Eq -- '^--- PASS: TestCaptureArrivalLedgerPreservesWireOrder' /tmp/wo.txt
```

Replace Task 3 verification with the exact new test plus the affected scenario:

```sh
go test ./test/wireoracle \
  -run '^(TestCanonicalResponseOrder|TestPipelinedResponseOrder|TestFrozenTranscriptsMatch/toolslist-repeat)$' \
  -count=1 -v | tee /tmp/wo2.txt
test "${PIPESTATUS[0]}" -eq 0
grep -Eq -- '^--- PASS: Test(CanonicalResponseOrder|PipelinedResponseOrder)' /tmp/wo2.txt
grep -Eq -- '^    --- PASS: TestFrozenTranscriptsMatch/toolslist-repeat' /tmp/wo2.txt
```

Prefer R2 if official SDK/protocol evidence confirms response order is unspecified; do not make that selection solely from observed timing.

### Risk Assessment

**HIGH**.

---

## Plan 03-04 — Browse tracer, URL grammar, errors, highlighting

### Summary

The tracer approach is excellent, and all three verifications are non-vacuous. The largest issue is that the proposed highlighter guard parses TypeScript source text with an underspecified extraction format.

### Strengths

- The production RPC response is genuinely discriminated by `mode`; reading fields by emptiness would be wrong ([internal/uiproto/uiv1/ui.proto](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/uiproto/uiv1/ui.proto:405)).
- `SourceBlob.content` is correctly treated as bytes, not guaranteed UTF-8 ([internal/uiproto/uiv1/ui.proto](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/uiproto/uiv1/ui.proto:345)).
- The shared error mapper aligns with server behavior:
  - Not found → `CodeNotFound`.
  - Invalid input → `CodeInvalidArgument`.
  - Lock contention → typed unavailable error ([internal/uiserver/handlers.go](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/uiserver/handlers.go:105)).
- Every verification requires the newly added test identity or exact new Go tests, so none passes in an unimplemented tree.
- The single-`{@html}` restriction complements the existing CSP, whose `default-src` and `connect-src` are already restricted to self ([internal/uiserver/spa.go](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/uiserver/spa.go:99)).

### Concerns

- **MEDIUM — The Go guard depends on parsing TypeScript syntax with regex/text extraction.** Small formatting changes, computed keys, or refactoring the alias map could break or misread it. The positive count helps, but does not prove complete parsing.
- **MEDIUM — `highlightAuto` is semantically questionable for unknown indexed languages.** It may misidentify a language rather than degrade to plain escaped source. Since the expected set is guarded exactly, unknown language IDs should probably render escaped plaintext and surface a diagnostic.
- **LOW — “Unknown params are preserved” is stronger than “ignored.”** Carrying all future parameters through Phase 3 serialization is sensible, but it should be represented explicitly as an ordered multimap; converting to an object may lose repeated keys.

### Suggestions

- Export a literal `HIGHLIGHT_COVERAGE` array in TypeScript and have the Go test parse only that tightly constrained declaration.
- For unregistered languages, use an escaping-only plaintext path rather than `highlightAuto`.
- Add repeated and empty unknown-parameter cases to URL tests.

### Risk Assessment

**MEDIUM**.

---

## Plan 03-05 — `GetPermalink`

### Summary

The endpoint design is appropriate and the verifications are non-vacuous, but the plan contradicts itself about whether git failures are observable. That must be resolved before implementation.

### Strengths

- Reading the indexed commit is correct: `GetStatusResponse.commit_sha` explicitly represents the graph’s indexed commit and allows absence for pre-upgrade graphs ([internal/uiproto/uiv1/ui.proto](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/uiproto/uiv1/ui.proto:151)).
- The confinement wrapper is consistent with the established `SourceFor` delegation pattern ([internal/query/detail.go](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/query/detail.go:254)).
- Every new RPC automatically inherits the existing Connect handler, message-size limits, and Origin/Host guard ([internal/uiserver/server.go](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/uiserver/server.go:121)).
- Both automated verifications demand newly named permalink tests, and Task 3 additionally runs proto drift.
- Credential stripping, exact `github.com` matching, no fetch, and per-segment escaping are appropriate controls.

### Concerns

- **HIGH — The git error contract is internally contradictory.**
  - Artifact signature: `CommitOnRemoteTrackingBranch(...) (bool, error)`.
  - Task 2 says “every function degrades to its zero value on any failure—never to an error.”
  - Task 1 asks the maintainer to decide what happens when the check cannot execute.
  
  If errors are swallowed, the handler cannot distinguish “checked and not observed” from “could not check,” yet D-07 requires honest uncertainty.
- **MEDIUM — “Missing git” is not currently testable through `internal/gitmeta`’s existing design.** `WorktreeRoot` directly calls `exec.CommandContext` with no lookup seam ([internal/gitmeta/worktree.go](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/gitmeta/worktree.go:34)). The plan must introduce a package-local seam and ensure all new git functions use it.
- **MEDIUM — Remote reason ownership is unclear.** `RemoteGitHubRepo` returns only `(owner, repo)`, but no-link must distinguish missing remote, unsupported host, malformed remote, and missing git.
- **LOW — The plan’s “proto only additions” check uses `git diff --stat`; that does not prove deletions are absent.**

### Suggestions

Adopt an explicit result type:

```go
type RemotePresence int
const (
    RemotePresenceUnknown RemotePresence = iota
    RemotePresenceObserved
    RemotePresenceNotObserved
)
```

Return structured remote resolution:

```go
type GitHubRemote struct {
    Owner, Repo string
    Host        string
    Reason      string
}
```

Replace the additions-only assertion with:

```sh
git diff --numstat -- internal/uiproto/uiv1/ui.proto |
awk '$2 != 0 { exit 1 } END { exit NR == 0 }'
```

### Risk Assessment

**HIGH** until the error semantics are made consistent.

---

## Plan 03-06 — Search and keyboard navigation

### Summary

The search controller is well designed. The main risk is over-dependence on a registry-vendored Command primitive for cross-group keyboard behavior without a prior proof that it supports that exact interaction.

### Strengths

- The trigger split matches the wire: `ExploreGroup` carries a source blob per group ([internal/uiproto/uiv1/ui.proto](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/uiproto/uiv1/ui.proto:468)), while `Search` returns lightweight locations ([internal/uiproto/uiv1/ui.proto](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/uiproto/uiv1/ui.proto:87)).
- Abort plus monotonically increasing request identity is the correct race defense.
- All automated verifications require newly named test files, so they fail in World B.
- The `/` suppression and command-chord behavior are specified precisely.
- The blocking review of vendored source acknowledges that registry-authored files are outside lockfile auditing.

### Concerns

- **MEDIUM — Cross-group arrow navigation is assumed before the vendored implementation is known.** The plan should treat this as a spike/acceptance check immediately after vendoring, not defer discovery to Task 3.
- **MEDIUM — The checkpoint mutates the repository before approval.** That is intentional, but rollback instructions should be explicit if the human rejects a file or package.
- **LOW — Search result selection temporarily bypasses URL state.** Task 3 intentionally assigns target state directly until 03-07 replaces it. That creates an intermediate commit violating the URL-as-source-of-truth contract.

### Suggestions

- After vendoring, add a minimal keyboard test before building the controller.
- If Command does not traverse groups as required, use one flat item collection with visually grouped presentation rather than custom ARIA key handling.
- Consider landing 03-06 and 03-07 together or have 03-06 selection update the URL immediately.

### Risk Assessment

**MEDIUM**.

---

## Plan 03-07 — Node navigation and history

### Summary

The URL-as-state approach is correct and the verifications are non-vacuous. The biggest gap is that the automated suite verifies navigation options, but real back/forward correctness remains entirely manual.

### Strengths

- The plan correctly uses the response’s `mode` discriminator; `GetNodeDetailResponse` explicitly warns that wrong-mode fields produce zero values ([internal/uiproto/uiv1/ui.proto](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/uiproto/uiv1/ui.proto:405)).
- `goto` injection makes push-versus-replace semantics testable.
- All three `<verify>` blocks require new test identities.
- Passing depth/limit unchanged is consistent with the wire’s existing server-owned validation discipline.
- The route already derives active navigation from reactive `page.url` ([web/src/routes/+layout.svelte](/Volumes/Code/github.com/seanb4t/codegraph-go/web/src/routes/+layout.svelte:24)), supporting the chosen URL-driven design.

### Concerns

- **MEDIUM — Back/forward correctness is manual-only.** This is a central NAV-02 requirement, not a peripheral interaction.
- **MEDIUM — `loadBrowseTarget` and `loadBlastRadius` can race independently.** The plan discusses aborting individual loads but does not define a single navigation generation that prevents results from two different URL states being combined.
- **LOW — The plan cites `Location` for neighbor entries, but `GetNodeDetail.calls` and `called_by` are full `Node` messages** ([internal/uiproto/uiv1/ui.proto](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/uiproto/uiv1/ui.proto:437)). This does not block implementation but can cause incorrect typing assumptions.

### Suggestions

- Add a browser-level navigation test using the app’s actual router, even if limited to push → push → back.
- Give the page one navigation generation/abort controller shared by node detail and blast-radius loads.
- Correct plan typing references from `Location` to `Node` for callers/callees.

### Risk Assessment

**MEDIUM**.

---

## Plan 03-08 — References, picker, copy, permalink UI

### Summary

The plan faithfully implements the constrained click-to-definition design, but DOM decoration after syntax highlighting is fragile and needs lifecycle/idempotence rules.

### Strengths

- The storage limitation is real: the phase cannot obtain call-site spans, so text-driven matching is a defensible bounded implementation.
- DOM node construction avoids creating a second HTML-string producer.
- Every `<verify>` block requires its newly added test identity and is non-vacuous.
- The picker correctly relies on `detail_gathered`, whose wire comment explicitly distinguishes genuine empty details from ungathered candidates ([internal/uiproto/uiv1/ui.proto](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/uiproto/uiv1/ui.proto:382)).
- The truncation UI uses server-provided counts and preserves the existing bounded source design ([internal/uiproto/uiv1/ui.proto](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/uiproto/uiv1/ui.proto:353)).

### Concerns

- **MEDIUM — Decorator lifecycle is unspecified.** A Svelte rerender can destroy decorated nodes; rerunning without cleanup can double-wrap tokens or retain stale event listeners.
- **MEDIUM — Highlight.js may split a qualified identifier across multiple text nodes.** A TreeWalker tokenizing each text node independently cannot match identifiers spanning markup boundaries.
- **LOW — “All candidates the server counted” is overstated.** The server currently sends `definitions` for all matches, with detail gathering capped ([internal/uiserver/handlers.go](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/uiserver/handlers.go:699)), but the UI must still handle a future response where `total_candidates > definitions.length`.

### Suggestions

- Implement decoration as a Svelte action with teardown and idempotence tests.
- Add a test where highlight markup splits a qualified or Unicode identifier across spans.
- Phrase the picker as “all returned candidates, plus the true total,” and explicitly render omitted-count differences.

### Risk Assessment

**MEDIUM**.

---

## Plan 03-09 — Status gate and committed bundle

### Summary

The shared degrade design is correct, but Task 3’s verification is vacuous if the entire plan is absent, and navigation-driven status refresh is insufficiently wired.

### Strengths

- The status asymmetry is accurately grounded:
  - `initialized` is false in both no-index and locked-store cases ([internal/uiproto/uiv1/ui.proto](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/uiproto/uiv1/ui.proto:117)).
  - `store_exists` and `indexing_in_progress` distinguish them ([internal/uiproto/uiv1/ui.proto](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/uiproto/uiv1/ui.proto:159)).
  - `degradedStatus` populates exactly those fields ([internal/uiserver/degrade.go](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/uiserver/degrade.go:136)).
- Task 1 and Task 2 verifications require their new tests and are non-vacuous.
- The status banner belongs at layout level; the existing shell already wraps every route there ([web/src/routes/+layout.svelte](/Volumes/Code/github.com/seanb4t/codegraph-go/web/src/routes/+layout.svelte:33)).
- Existing web drift protection genuinely checks source and output counts and digests ([Taskfile.yml](/Volumes/Code/github.com/seanb4t/codegraph-go/Taskfile.yml:1014)).

### Concerns

- **HIGH — Task 3 `<verify>` passes in an entirely unimplemented Phase-3 world.**  
  Exact command:
  ```sh
  task web:build:verify &&
  task web:drift &&
  task web:test &&
  task proto:drift &&
  task test:unit
  ```
  Against the existing Phase-2 source/build pair, these can all be green. The command never asserts `status.test`, `degrade-states`, `StatusBanner`, or that the build manifest changed.
- **MEDIUM — “Fetch on navigation” has no defined router integration mechanism.** The plan exposes a notification method but does not state how `$layout.svelte` observes `page.url` changes and calls it.
- **MEDIUM — Creating the gate and observing navigation reactively can double-fetch on initial load.**
- **LOW — `git status --porcelain web/build` being empty is not proof that the correct build was committed; it only proves the working tree matches HEAD.**

### Suggestions

Use a verification that binds the rebuilt artifact to the new source and tests:

```sh
task web:build:verify &&
task web:drift &&
task web:test 2>&1 | tee /tmp/web-test.txt &&
grep -Eq 'status\\.test' /tmp/web-test.txt &&
grep -Eq 'degrade-states' /tmp/web-test.txt &&
test -f web/build/.build-manifest &&
rg -q 'StatusBanner' web/src/routes/+layout.svelte
```

Also record the manifest source digest before and after the rebuild and require it to change when Phase-3 source changed.

Specify a single initial fetch, then an effect keyed on a normalized navigation identity that skips its first execution.

### Risk Assessment

**HIGH** until Task 3 verification is corrected.

---

## Plan 03-10 — golangci-lint

### Summary

The plan appropriately treats the twice-folded lint item as real work, but the CI-wiring verification is vacuous and the tool-module strategy is likely to encounter the exact MVS conflict the repository already documents.

### Strengths

- The tool isolation concern is real and documented in the existing modfile ([go.tool-lint.mod](/Volumes/Code/github.com/seanb4t/codegraph-go/go.tool-lint.mod:1)).
- Task 1’s verification is non-vacuous: without `lint:go`, the exact target count fails.
- Task 2’s `task lint:go` verification directly propagates the linter’s result.
- Adding the CI job to `inScopeJobs` is necessary because the guard’s scope is a literal allowlist ([internal/upgrade/taskfile_shape_test.go](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/upgrade/taskfile_shape_test.go:119)).
- The plan correctly leaves the six-leg test wrapper untouched; its expected set is pinned at lines 96–107.

### Concerns

- **HIGH — Task 3 `<verify>` does not verify Task 3.**  
  Exact command:
  ```sh
  task lint:actions &&
  go test ./internal/upgrade/ -run TestWorkflowRunBodiesInvokeTask ...
  ```
  Both pass before adding the `lint-go` job because that job is absent from both the workflow and `inScopeJobs`. This is the precise silent-gap shape the plan itself describes.
- **MEDIUM — Co-locating golangci-lint with actionlint is attempted despite the modfile being explicitly dedicated to avoiding MVS conflicts.** The fallback is present, but trying the known-fragile combination first adds churn.
- **MEDIUM — Tool supply-chain legitimacy is not reviewed.** Plan 03-01 requires human approval for test dependencies, while this much larger executable dependency tree gets no corresponding review.
- **LOW — “Scratch Go file” location is unspecified.** If written outside included package paths, golangci-lint may not inspect it, making the red proof misleading.

### Suggestions

Correct Task 3 verification:

```sh
task lint:actions &&
test "$(rg -o 'task lint:go' .github/workflows/ci.yml | wc -l | tr -d ' ')" -eq 1 &&
test "$(rg -o 'JobID: \"lint-go\"' internal/upgrade/taskfile_shape_test.go | wc -l | tr -d ' ')" -eq 1 &&
go test ./internal/upgrade -run '^TestWorkflowRunBodiesInvokeTask$' -count=1 -v |
tee /tmp/wf.txt &&
test "${PIPESTATUS[0]}" -eq 0 &&
grep -Eq '^--- PASS: TestWorkflowRunBodiesInvokeTask' /tmp/wf.txt
```

Create a separate `go.tool-golangci.mod` from the outset unless a dry-run proves co-location is clean.

Plant lint violations in an existing included package file, record its original hash, and require byte-identical restoration afterward.

### Risk Assessment

**HIGH** until CI-wiring verification is fixed.

---

# Cross-plan findings

## Dependency and wave structure

### Strengths

- The critical path is sensible: harness → tracer → search → navigation → source completion → degrade/bundle.
- `03-08` depends on both the permalink server and navigation work.
- `03-09` transitively depends on the permalink proto via `03-08`, so the final build includes regenerated client code.
- The two unrelated todo plans remain off the browse critical path.

### Concerns

- **MEDIUM — Wave 2 has concurrent plans touching the generated client context.** `03-05` regenerates `web/src/lib/gen/ui_pb.ts`, while `03-04` compiles/tests against that directory. Shared-worktree concurrent execution can race.
- **MEDIUM — Several plans rely on manual UAT as the only proof of central requirements**, especially history reconstruction, real browser keyboard navigation, and permalink following.
- **MEDIUM — Performance validation is thin.** There is no planned measurement for:
  - Rendering/highlighting a source blob at the server cap.
  - Hundreds of callers/callees.
  - Search result list size and DOM cost.
- **LOW — CSP and rebinding controls are inherited correctly.** The mux wraps RPC and SPA under the same exact Host/Origin guard ([internal/uiserver/server.go](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/uiserver/server.go:127)), and CSP already limits connections to self.

## Highest-priority corrections

1. Fix the four vacuous verification blocks:
   - 03-03 Task 1.
   - 03-03 Task 3.
   - 03-09 Task 3.
   - 03-10 Task 3.
2. Resolve 03-05’s contradictory git-failure contract with a structured result.
3. Add a real `.svelte` fixture to 03-01.
4. Prevent concurrent execution of 03-04 and proto regeneration in 03-05, or isolate their worktrees.
5. Add automated browser/router coverage for NAV-02 and lifecycle/idempotence coverage for source decoration.

## Final Risk Assessment

**HIGH.** The architecture should achieve the phase goals, and most task-level tests are thoughtfully non-vacuous. The remaining high risk comes from verification commands that can report green without their tasks being implemented, plus an unresolved permalink error contract that affects a permanent RPC. Once those are corrected, the plan set falls to **MEDIUM**, driven mainly by UI integration complexity and manual-only browser-history validation.

---

## Claude Review

`★ Insight ─────────────────────────────────────`
Two of the three defects I found are only visible by *executing* the check, not reading it: a `$` that reads as a literal in shell but as an end-of-line anchor in regex, and a `--- PASS:` grep against a `go test` invocation that never passes `-v`. Both read perfectly correct on the page.
`─────────────────────────────────────────────────`

# Cross-AI Plan Review — Phase 3 (03-01 … 03-10)

Repo read at `/Volumes/Code/github.com/seanb4t/codegraph-go`, branch `gsd/v0.12.0-local-graph-ui`, commit `f11bcb9c`. Every finding below is traced to a file I opened or a command I ran.

## 1. Summary

This is a strong plan set. The factual grounding is unusually good: I spot-checked ~20 load-bearing citations (`internal/query/node.go:33-79` confinement, `internal/uiserver/handlers.go:101-133` `mapEngineError`, `handlers.go:640-650` `singleDefSourceBlob`, `SourceBlob`/`Location`/`GetStatusResponse` field numbers, `Taskfile.yml:40-51` `web_source_files()`, `internal/upgrade/taskfile_shape_test.go:100-138` wrapper legs and `inScopeJobs`, the 14 `LanguageSpec.ID` values, `.svelte-kit/tsconfig.json`'s `../tests/**/*.ts` include) and **every one held**. The wave DAG is sound and the dependency edges are real. The three pre-approved deviations are correctly applied.

The problems are concentrated in exactly the place the primary directive points at: three `<verify>`/acceptance checks are structurally incapable of doing their job — one passes in World B, one can *never* pass even in World A, and one is a negative-only regex that never matches anything. Separately, there is a real cross-wave CI consequence nobody scheduled: this phase leaves `task web:drift` red in CI from wave 1 through wave 6.

## 2. Strengths

- **The confinement design is correct and the plan resisted the obvious wrong turn.** `internal/query/node.go:33-79` implements four layers (empty, absolute, `..`-prefix, and a post-`EvalSymlinks` re-verification comparing resolved-root to resolved-path at `:65-76`). `readSourceFile` (`:84-90`) is the sole read primitive, and `handlers.go:698` reaches it via `eng.SourceFor`. 03-02 correctly concludes SRV-05's deliverable is the missing *proof*, and 03-05's `ValidateRepoRelativePath` is a delegating wrapper — the acceptance criterion `rg -o 'EvalSymlinks|filepath.Clean|HasPrefix' internal/uiserver/confinement_test.go | wc -l` returns 0 enforces that structurally.
- **The `GetNodeDetail` mode-discrimination hazard is correctly identified.** `nodeDetailToProto` (`handlers.go:672-718`) populates exactly one field group per mode, and `nodeDefinitionToProto` (`:612-623`) leaves `calls`/`called_by`/`source` at zero when `gathered` is false. 03-07 Task 2's requirement that a *single-def response with an empty calls list* still classify as single-def, and 03-08 Task 3's requirement that an *empty-calls, gathered-true* candidate carry no marker, are precisely the two cases an emptiness-inference gets wrong. That is the strongest test design in the set.
- **`web/highlight_coverage_test.go` reads the real TypeScript file rather than a Go literal** (03-04 Task 3). A hardcoded Go list compared only against `indexer.RegisteredLanguageIDs()` would pass while `highlight.ts` registered anything at all. The ≥13-registration-count floor plus the alias-map-values-must-be-registered check plus `TestHighlightCoverageComparisonDiscriminates` is three independent non-vacuity assertions on one guard. `RegisteredLanguageIDs()` genuinely returns 14 (verified: `grep -rh 'ID:\s*"' internal/indexer/languages_*.go` → 14, including `tsx` and `javascript`).
- **`task proto:drift`'s floor is understood correctly.** `Taskfile.yml:345-347` prints `compared ${nfiles}` then fails below 4 — a *floor*, not equality. 03-05 correctly asserts the count stays 4 (adding messages + one rpc to an existing `.proto` adds no generated file) and explicitly forbids moving the floor.
- **The `test` wrapper set-equality trap is caught.** `taskWrapperExpectedLegs` (`taskfile_shape_test.go:100-107`) is 6 entries compared as a sorted set; 03-01 Task 3(c) correctly refuses to add a seventh leg. Meanwhile the `lint` wrapper has *no* such guard (`Taskfile.yml`: `cmds: [vet, lint:actions]`), which 03-10 also states correctly.
- **`web/tests/` placement is deliberate and verified.** `.svelte-kit/tsconfig.json` already includes `../tests/**/*.ts`, and `web_source_files()` (`Taskfile.yml:41-50`) does *not* enumerate `web/tests` — so test edits do not churn the BLD-03 source digest. Both halves of that reasoning check out.

## 3. Concerns

### HIGH — vacuous / inverted guards

**H1. The `pushState`/`replaceState` prohibition regex can never match. `03-04-PLAN` and `03-07-PLAN`.**

```
rg -o "import \{[^}]*(pushState|replaceState)[^}]*\} from '\$app/navigation'" web/src/ | wc -l   # returns 0
```

`\$` inside double quotes yields a *literal* `$` to `rg`, and in the regex engine `$` is an end-of-anchor assertion — so `$app` can never match the text `$app`. I proved this against a file containing exactly the forbidden import:

```
$ cat /tmp/rgt/a.ts
import { pushState, replaceState } from '$app/navigation';
$ rg -o "...from '\$app/navigation'" /tmp/rgt/ | wc -l
       0                      # plan's pattern — MISSES the violation
$ rg -o "...from '\\\$app/navigation'" /tmp/rgt/ | wc -l
       1                      # corrected — catches it
```

This appears **four times**: `03-04` acceptance criteria + `<verification>`, `03-07` Task 1 acceptance + `<verification>`. It is guarding the single correction this phase makes to a locked CONTEXT decision (D-11 shallow routing → `goto`), and it is the one check that would catch an executor reverting to D-11's literal wording.

Fix: escape for the regex, not just the shell —
```
rg -o "import \{[^}]*(pushState|replaceState)[^}]*\} from '\\\$app/navigation'" web/src/ | wc -l
```
and add the mandated positive companion (rule `84d1gfpywd`) — a non-zero count of `from '\$app/navigation'` imports overall, proving the search can find that module path when it is present.

**H2. `03-03-PLAN` Task 3's `<verify>` can never pass, even when everything is correct.**

```
task test:wireoracle 2>&1 | tee /tmp/wo2.txt; test "${PIPESTATUS[0]}" -eq 0 && test "$(grep -Eco -- '--- PASS: TestFrozenTranscriptsMatch/' /tmp/wo2.txt)" -ge 1
```

`test:wireoracle` is `go test ./test/wireoracle/...` with **no `-v`** (`Taskfile.yml:176`). Without `-v`, Go prints `ok  github.com/.../test/wireoracle  12.3s` and *zero* `--- PASS:` lines. The count is always 0, so `-ge 1` always fails. The danger is the obvious repair: an executor under time pressure deletes the count assertion, leaving a bare `task test:wireoracle` — which is exactly the vacuity the assertion was added to prevent.

Fix: run the pattern directly rather than through the target —
```
go test ./test/wireoracle/ -run 'TestFrozenTranscriptsMatch' -count=1 -v 2>&1 | tee /tmp/wo2.txt
```
and keep `task test:wireoracle` as a separate acceptance criterion for the full-suite green.

**H3. `03-10-PLAN` Task 3's `<verify>` is green in World B.**

```
task lint:actions && go test ./internal/upgrade/ -run 'TestWorkflowRunBodiesInvokeTask' ... && grep -Eqo -- '--- PASS: TestWorkflowRunBodiesInvokeTask'
```

`TestWorkflowRunBodiesInvokeTask` iterates only `inScopeJobs` (`taskfile_shape_test.go:129-138`, doc comment at `:1357-1358`). If Task 3 is not done at all — no `lint-go` job in `ci.yml`, no fixture entry — that test still passes, because there is nothing new for it to bind. `task lint:actions` also passes today. **The entire `<verify>` block exits 0 against an unimplemented task.** The real checks live only in the acceptance criteria (`rg -o 'task lint:go' .github/workflows/ci.yml | wc -l` returns 1, `rg -o 'JobID: "lint-go"' ... | wc -l` returns 1), which is precisely the split the primary directive warns about.

Fix: fold the two `rg` counts into the `<verify>` block itself, before the `go test`.

### MEDIUM

**M1. This phase leaves `task web:drift` red in CI for five consecutive waves.** `.github/workflows/ci.yml:164` runs `task web:drift` in the `test` job on every push. `web_source_files()` (`Taskfile.yml:41-50`) enumerates `web/package.json`, `web/vite.config.ts` and `web/src`. Plan 03-01 (wave 1) modifies `web/package.json` (5 new deps + `test` scripts) and `web/vite.config.ts` (the `test:` block). Every subsequent plan touches `web/src`. The bundle is not rebuilt until **03-09 Task 3, wave 6**. So every commit from wave 1 to wave 5 fails CI's `test` job on the source-vs-output digest mismatch. No plan acknowledges this. Either schedule a `task web:build` at the end of each web-touching plan, or state explicitly in 03-01 that `web:drift` is expected red until 03-09 and record the intended-red window — otherwise the failure will be read as noise and someone will relax the guard.

**M2. `PIPESTATUS` is a bashism, used in 8 `<verify>` blocks.** 03-01 T2, 03-02 T1/T2, 03-03 T1/T3, 03-04 T1/T3, 03-05 T2/T3, 03-10 T3. Under `sh`/`dash`, `${PIPESTATUS[0]}` expands to empty and `test "" -eq 0` errors out. It fails closed rather than vacuously, so this is a *reliability* not a *safety* defect — but it will fail in World A on a POSIX-`sh` executor. Either declare bash explicitly or restructure as `cmd > file 2>&1 || { cat file; exit 1; }`.

**M3. `03-03-PLAN` Task 2 reads an artifact that does not exist yet.** Task 2's `<read_first>` names `.planning/phases/.../03-03-SUMMARY.md` for Task 1's recorded verdict, but the plan's `<output>` says to create that SUMMARY "when done" — i.e. after Task 3. The blocking-human decision checkpoint therefore has nothing to read. Fix: have Task 1 write an interim evidence file (`03-03-EVIDENCE.md`) that Task 2 reads.

**M4. The `{@html}`-count and `innerHTML`-count invariants are asserted before the vendored components arrive.** 03-04 asserts `rg -o '\{@html' web/src/ | wc -l` returns 1; 03-08 (wave 5) re-asserts it *and* asserts `rg -o 'innerHTML|outerHTML|insertAdjacentHTML|createContextualFragment' web/src/ | wc -l` returns 0. But 03-06 (wave 3) vendors shadcn-svelte `command` source into `web/src/lib/components/ui/`. `web/src` contains none of these constructs today (verified), but Bits UI-derived component source plausibly does. If it does, 03-08 fails for a reason unrelated to its own work and the temptation is to scope the check to `web/src/lib/components/browse/`, which would silently exempt the vendored code — the exact surface D-22 says has *no* other review. Fix: scope the assertion now to `web/src` excluding `components/ui/`, and add a separate explicit review item in 03-06's checkpoint covering unescaped-HTML constructs in the vendored files.

**M5. 03-01 Task 2's vitest output parse is format-coupled.** `grep -Eo 'Tests +[0-9]+ passed'` assumes vitest's default reporter emits `Tests  N passed (N)` uncoloured. A reporter change or an unexpected TTY makes this fail in World A. Task 3 already solves this properly (JSON reporter → file → parse `numPassedTests`); Task 2 should use the same mechanism rather than a second, weaker one.

### LOW

- **L1.** 03-02's subtest-count regex `^ *--- PASS: TestGetNodeDetailPathConfinementAtRPCBoundary/[a-z_]+` will not match a subtest named with a digit or hyphen (Go converts spaces to `_`, so `positive control` → `positive_control` matches, but `in-repo control` → `in-repo_control` does not). Widen to `[a-zA-Z0-9_-]+`.
- **L2.** All ten plans carry `confidence: low` with token estimates of 45k–95k. That is 785k estimated tokens for one phase. Not wrong, but worth a maintainer sanity check before dispatch.
- **L3.** 03-05's threat register says `GetPermalink` "will NOT need `withEngine`" in `03-PATTERNS.md:93`, but the plan's Task 3(d) correctly requires `withEngine` (for the indexed commit SHA and the confinement wrapper). PATTERNS is stale on this point; the plan is right. Worth a one-line note so an executor reading PATTERNS first does not follow the wrong shape.

## 4. Suggestions

1. Fix H1/H2/H3 before dispatch — all three are one-line edits, and all three are in checks that exist specifically to be non-vacuous.
2. Add a standing "intended-red window" note to 03-01 covering M1, or insert a `task web:build` step at the end of each web-touching plan.
3. Normalize the `<verify>` idiom across all ten plans to one bash-explicit shape: materialize output to a file, check the command's own exit status directly, then assert a count. 03-01 Task 3's Taskfile guard already models this correctly; the rest should copy it rather than each inventing a pipeline.
4. Move 03-10 Task 3's two `rg` count assertions from acceptance criteria into the `<verify>` block.
5. Have 03-06's blocking checkpoint explicitly enumerate unescaped-HTML constructs in the vendored source, closing M4 at the point the files enter the repo rather than two waves later.

## 5. Risk Assessment

**MEDIUM.**

The architecture is right, the citations hold, the wave DAG is sound, and the requirement coverage is complete (14 requirements across 03-02/04/05/06/07/08/09, with 03-03 and 03-10 correctly fenced off as non-gating). Nothing here threatens the phase's five success criteria on design grounds.

The risk is entirely in verification integrity: three guards that a reader would sign off on and an executor would trust, two of which (H1, H3) return green against a completely unimplemented task. Given this milestone's own record — Phase 2 shipped six vacuous guards, three invisible to reading — shipping H1 and H3 would repeat the exact defect class the phase's own prohibitions are written to prevent. H2 is lower-consequence but higher-probability: it *will* fire, and the natural repair reintroduces vacuity.

With those three fixed and M1 acknowledged, I would rate this LOW.

---
## Consensus Summary

Two independent, repo-grounded reviewers agree the **architecture is sound and the
citations hold** — the wave DAG is real, the confinement reuse (SRV-05) is correct,
the `GetNodeDetail` mode-discrimination hazard is correctly identified, and the three
pre-approved deviations are correctly applied. Neither reviewer found a design-level
threat to the phase's success criteria.

**All the risk is concentrated in verification integrity.** Between them the two
reviewers found **seven HIGH-severity defects**, five of which are `<verify>`/guard
blocks that a reader signs off on and an executor trusts, but that do not do their
job when executed. This is the exact defect class Phase 2 shipped six instances of.

Three of the seven were independently reproduced by the orchestrator against the
working tree before this file was written (see *Orchestrator verification* below).

Codex rates the set **HIGH until the verification and permalink-contract issues are
corrected, MEDIUM afterward**. Claude rates it **MEDIUM**, dropping to **LOW** once
its three HIGHs are fixed. The spread is a severity-scale difference, not a
disagreement about the findings: neither reviewer would dispatch the set unchanged.

### Agreed Strengths

- **Confinement is right and the plan resisted the obvious wrong turn.** Both
  reviewers independently traced `internal/query/node.go:33-79` and confirmed it
  genuinely confines across all four layers (empty, absolute, `..`-prefix, and a
  post-`EvalSymlinks` resolved-root-to-resolved-path re-verification at `:65-76`).
  Both confirm 03-02's deliverable is correctly scoped as the *missing proof*, and
  03-05's `ValidateRepoRelativePath` is a genuine delegating wrapper — not a second
  implementation. The acceptance criterion forbidding `EvalSymlinks|filepath.Clean|
  HasPrefix` in the new test file enforces that structurally.
- **Mode discrimination over emptiness inference.** Both cite
  `internal/uiproto/uiv1/ui.proto:405` and `internal/uiserver/handlers.go:672-718`:
  `nodeDetailToProto` populates exactly one field group per mode, so reading fields
  by emptiness would be wrong. 03-07 Task 2 and 03-08 Task 3 test precisely the two
  cases an emptiness-inference gets wrong. Claude calls this "the strongest test
  design in the set."
- **Most task-level `<verify>` blocks ARE non-vacuous.** Both reviewers explicitly
  cleared 03-02 (both tasks), 03-04 (all three), 03-05 (both), 03-06 (all), 03-07
  (all three), 03-08 (all) and 03-01 Task 3 as requiring a newly-added test identity
  that cannot exist in World B. The defects are localized, not systemic.
- **The error mapper and transport inheritance are correctly understood.**
  `mapEngineError` (`handlers.go:101-133`) maps not-found → `CodeNotFound`, invalid →
  `CodeInvalidArgument`, lock contention → typed unavailable; every new RPC inherits
  the existing Connect handler, message-size limits and Origin/Host guard
  (`internal/uiserver/server.go:121-127`), and CSP already restricts `default-src`/
  `connect-src` to self (`internal/uiserver/spa.go:99`).
- **Build-digest reasoning checks out.** `web_source_files()` (`Taskfile.yml:41-50`)
  does not enumerate `web/tests`, which is why vitest config belongs in
  `vite.config.ts` and tests belong in `web/tests/` — both reviewers verified both
  halves. `task proto:drift`'s count-4 is a floor, not an equality, and 03-05
  correctly forbids moving it.
- **The `test` wrapper set-equality trap is caught.** `taskWrapperExpectedLegs`
  (`internal/upgrade/taskfile_shape_test.go:100-107`) is a 6-entry sorted-set
  comparison; 03-01 correctly refuses to add a seventh leg, and 03-10 correctly notes
  the `lint` wrapper has no equivalent guard.

### Agreed Concerns

**HIGH — 03-10 Task 3's `<verify>` is green against a completely unimplemented task.**
Raised by BOTH reviewers (Codex "HIGH — Task 3 does not verify Task 3"; Claude H3).
Command: `task lint:actions && go test ./internal/upgrade/ -run 'TestWorkflowRunBodiesInvokeTask' ... && grep -Eqo -- '--- PASS: TestWorkflowRunBodiesInvokeTask'`.
`TestWorkflowRunBodiesInvokeTask` iterates only `inScopeJobs`
(`taskfile_shape_test.go:129-138`), so with no `lint-go` job in `ci.yml` and no
fixture entry there is nothing new for it to bind — it passes today. `task
lint:actions` passes today. The real assertions live only in the acceptance criteria,
which is exactly the split the primary directive warns about.
**Orchestrator-confirmed:** `rg -c 'lint-go|task lint:go' .github/workflows/ci.yml
internal/upgrade/taskfile_shape_test.go` returns zero matches on the current tree.
*Fix (both reviewers converge):* fold the two `rg` count assertions
(`rg -o 'task lint:go' .github/workflows/ci.yml | wc -l` = 1 and
`rg -o 'JobID: "lint-go"' internal/upgrade/taskfile_shape_test.go | wc -l` = 1) into
the `<verify>` block itself, before the `go test`.

**HIGH — 03-03 Task 3's `<verify>` is broken, and its natural repair is vacuous.**
Both reviewers flag this block; they diagnose complementary halves of the same defect.
Command: `task test:wireoracle 2>&1 | tee /tmp/wo2.txt; test "${PIPESTATUS[0]}" -eq 0 && test "$(grep -Eco -- '--- PASS: TestFrozenTranscriptsMatch/' /tmp/wo2.txt)" -ge 1`.
- Claude (H2): `test:wireoracle` is `go test ./test/wireoracle/...` with **no `-v`**
  (`Taskfile.yml:176`). Without `-v`, Go prints one `ok ...` line and *zero* `--- PASS:`
  lines, so the count is always 0 and `-ge 1` can never pass — even in World A.
- Codex: the danger is the obvious repair — deleting the count assertion leaves a bare
  `task test:wireoracle`, which passes in World B because the oracle already passes and
  already contains many named transcript subtests.
**Orchestrator-confirmed:** `Taskfile.yml:176` is `go test ./test/wireoracle/...`,
no `-v`.
*Fix (both reviewers converge):* run the pattern directly with `-v` rather than through
the target — `go test ./test/wireoracle/ -run 'TestFrozenTranscriptsMatch' -count=1 -v`
— and keep `task test:wireoracle` as a separate acceptance criterion for full-suite green.
Codex additionally recommends binding the new named tests
(`TestCanonicalResponseOrder` / `TestPipelinedResponseOrder`) into the same block.

**MEDIUM — 03-01 Task 2's vitest output parse is format-coupled.** Both reviewers
(Codex MEDIUM; Claude M5). `grep -Eo 'Tests +[0-9]+ passed'` assumes the default
reporter's uncoloured `Tests  N passed (N)`. Non-vacuous but brittle; a reporter change
or unexpected TTY makes it fail in World A. Both note 03-01 Task 3 already solves this
correctly with the JSON reporter (`numPassedTests`) and Task 2 should use the same
mechanism rather than a second, weaker one.

**MEDIUM — central requirements rest on manual UAT only.** Both reviewers, differently
scoped. Codex: history reconstruction, real browser keyboard navigation, and permalink
following have no automated proof; back/forward correctness is manual-only despite
NAV-02 being a central requirement. Claude reaches the same conclusion via the
`web:drift` window (below). Codex suggests a browser-level push → push → back test using
the app's actual router.

**MEDIUM — performance validation is thin.** Codex only, but uncontradicted: no planned
measurement for rendering/highlighting a source blob at the server cap, hundreds of
callers/callees, or search-result list DOM cost.

### HIGH concerns raised by one reviewer (both orchestrator-verified where marked)

**HIGH (Claude H1) — the `pushState`/`replaceState` prohibition regex can NEVER match.**
`rg -o "import \{[^}]*(pushState|replaceState)[^}]*\} from '\$app/navigation'" web/src/ | wc -l`
— `\$` inside double quotes yields a literal `$` to `rg`, where `$` is an end-of-line
anchor, so `$app` can never match the text `$app`. The check returns 0 whether or not
the violation is present. It appears **four times**: `03-04-PLAN.md:268` and `:414`,
`03-07-PLAN.md:161` and `:314` — acceptance criteria *and* `<verify>` in both plans.
This is the single check guarding the phase's one correction to a locked CONTEXT
decision (D-11), i.e. the only thing that would catch an executor reverting to the
literal locked wording.
**Orchestrator-confirmed empirically:** against a file containing exactly
`import { pushState, replaceState } from '$app/navigation';`, the plan's pattern
returns `0`; the escaped form (`'\\\$app/navigation'`) returns `1`.
*Fix:* escape for the regex, not just the shell, and add the mandated positive
companion (rule `84d1gfpywd`) — a non-zero count of `from '\\\$app/navigation'` imports
overall, proving the search can find that module path when present.

**HIGH (Codex) — 03-03 Task 1's `<verify>` accepts a FAILING test and requires no
investigation.** `grep -Eqo -- '--- (PASS|FAIL): TestFrozenTranscriptsMatch/toolslist-repeat'`
— either verdict satisfies the grep, and the subtest already runs today, so the block
demands no instrumentation, no contention reproduction and no root-cause verdict.
**Orchestrator-confirmed:** the `toolslist-repeat` scenario exists today
(`test/wireoracle/oracle_test.go:608-643`), so the guard is green in World B.
*Fix:* assert a new named evidence test, e.g.
`go test ./test/wireoracle -run '^TestCaptureArrivalLedgerPreservesWireOrder$' -count=1 -v`
plus `grep -Eq -- '^--- PASS: TestCaptureArrivalLedgerPreservesWireOrder'`.

**HIGH (Codex) — 03-09 Task 3's `<verify>` never asserts 03-09's own deliverables.**
`task web:build:verify && task web:drift && task web:test && task proto:drift && task test:unit`
— every one of these is a pre-existing target. The block never asserts `status.test`,
`degrade-states`, `StatusBanner`, or that the build manifest changed. Once someone runs
`task web:build` the chain goes green regardless of whether the status gate exists.
*Fix:* bind the rebuilt artifact to the new source and tests — tee `task web:test`,
`grep -Eq 'status\.test'` and `grep -Eq 'degrade-states'`, `rg -q 'StatusBanner'
web/src/routes/+layout.svelte`, and record the manifest source digest before and after
the rebuild, requiring it to change when Phase-3 source changed.

**HIGH (Codex) — 03-05's git-failure error contract is internally contradictory.**
The artifact signature is `CommitOnRemoteTrackingBranch(...) (bool, error)`; Task 2 says
"every function degrades to its zero value on any failure — never to an error"; Task 1
asks the maintainer to decide what happens when the check cannot execute. If errors are
swallowed, the handler cannot distinguish "checked and not observed" from "could not
check" — yet D-07 requires honest uncertainty. This lands in a **permanent published RPC
shape**, so it must be resolved before Task 1's wire freeze.
*Fix:* an explicit tri-state (`RemotePresenceUnknown|Observed|NotObserved`) and a
structured `GitHubRemote{Owner, Repo, Host, Reason}` so `availability`/`reason` can be
answered honestly.

**HIGH (Codex) — 03-01 Task 2's harness component test is not implementable as written.**
Task 2 says `web/tests/harness.test.ts` should "render a trivial inline Svelte 5
component," but Testing Library's `render` expects a *compiled* component and ordinary
TypeScript cannot declare Svelte markup inline. No `.svelte` fixture appears in
`files_modified` or the artifacts table. The RED→GREEN conversion (step g) cannot be
performed as specified.
*Fix:* add `web/tests/fixtures/Harness.svelte` (and list it as an artifact), or render
an existing simple component.

### Divergent Views

- **Is `task web:drift` red between waves 1 and 6?** Claude (M1) says yes: `ci.yml:164`
  runs `task web:drift` on every push; `web_source_files()` enumerates
  `web/package.json`, `web/vite.config.ts` and `web/src`; 03-01 (wave 1) modifies the
  first two and every later plan touches `web/src`, but the bundle is not rebuilt until
  03-09 Task 3 (wave 6) — so CI's `test` job fails on the digest mismatch for five
  consecutive waves, and no plan acknowledges it. Codex reads 03-09 Task 3's chain as
  "can all be green against the existing Phase-2 source/build pair," which implicitly
  assumes drift is *not* red. **The orchestrator judges Claude correct on the mechanism**
  (`web:drift` compares committed source digest against the committed build manifest;
  03-01 changes an enumerated file without rebuilding). Codex's underlying point survives
  either way: the block never asserts 03-09's own deliverables. Resolve by either
  scheduling a `task web:build` at the end of each web-touching plan, or recording an
  explicit intended-red window in 03-01 — an unacknowledged red CI is read as noise and
  invites someone to relax the guard.
- **How severe is `PIPESTATUS`?** Claude (M2) flags it as a bashism in 8 `<verify>` blocks
  that fails *closed* under `sh`/`dash` — a reliability defect, not a safety one. Codex
  does not raise it and in fact *recommends* `PIPESTATUS` in three of its suggested fixes.
  **Orchestrator note:** the executor shell on this host resolves to `bash -lc` (bash
  5.3), where `PIPESTATUS` works; the outer wrapper is zsh, where it would not. The risk
  is real but host-dependent — worth normalizing the idiom rather than treating as urgent.
- **Overall risk rating.** Codex: HIGH until the verification and permalink-contract
  issues are fixed, then MEDIUM. Claude: MEDIUM, dropping to LOW once H1–H3 are fixed and
  M1 is acknowledged. The gap comes from Codex weighting the 03-05 contract and the
  03-01 infeasibility as blockers, and Claude weighting only the guard defects.

### Orchestrator verification

Three consensus/single-reviewer HIGHs were independently reproduced against the working
tree before this file was written, so they are not taken on a reviewer's word:

| Finding | Check run | Result |
|---|---|---|
| Claude H1 (`\$app` regex) | plan pattern vs corrected pattern against a file containing the forbidden import | plan pattern `0`, corrected `1` — **confirmed inverted** |
| Claude H2 (no `-v`) | `Taskfile.yml:176` | `go test ./test/wireoracle/...`, no `-v` — **confirmed** |
| Both, 03-10 T3 | `rg -c 'lint-go\|task lint:go' .github/workflows/ci.yml internal/upgrade/taskfile_shape_test.go` | zero matches — **confirmed green in World B** |
| Codex, 03-03 T1 | `rg -n 'toolslist-repeat' test/wireoracle/` | scenario exists at `oracle_test.go:608-643` — **confirmed green in World B** |
| SRV-05 reuse (pre-approved) | read `internal/query/node.go:33-79` | genuinely confines across all four layers — **reuse is sound** |

One further orchestrator observation not raised by either reviewer, recorded for the
planner: `resolveSourcePath` calls `filepath.EvalSymlinks(abs)`, which **errors when the
path does not exist**. `GetPermalink` is pinned to the *indexed* commit, so a permalink
request for a path that existed at that commit but has since been deleted from the
working tree would be refused as `CodeInvalidArgument`. 03-05 should state whether that
is intended.

### Ordered correction list

1. 03-04 ×2 and 03-07 ×2 — escape `\\\$app/navigation` for the regex and add the positive companion.
2. 03-10 Task 3 — fold the two `rg` count assertions into the `<verify>` block.
3. 03-03 Task 3 — run `go test ... -v` directly; keep `task test:wireoracle` as a separate criterion.
4. 03-03 Task 1 — assert a new named evidence test instead of accepting `PASS|FAIL`.
5. 03-09 Task 3 — bind the verify to `status.test`, `degrade-states`, `StatusBanner` and a changed manifest digest.
6. 03-05 Task 1 — resolve the git-failure contract to a tri-state before the wire freeze.
7. 03-01 Task 2 — add a real `.svelte` fixture.
8. 03-01 — record the intended-red `web:drift` window (or rebuild per web-touching plan).
9. Normalize the `<verify>` idiom across all ten plans to one bash-explicit shape.
