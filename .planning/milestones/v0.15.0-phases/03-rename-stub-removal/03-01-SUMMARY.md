---
phase: 03-rename-stub-removal
plan: 01
subsystem: cli
tags: [cobra, cli, breaking-change, changie]

# Dependency graph
requires:
  - phase: 02-phase-close-fragment-capability
    provides: the changie-fragments capability, dispatched at verify:post, that consumes this plan's SUMMARY to write the milestone's Breaking fragment
provides:
  - the hidden `query` and `unlock` commands removed from the CLI, with cobra's own unknown-command error and exit 1 in their place
  - a spawned-binary guard (TestRemovedVerbsAreUnknownCommands) pinning that behavior with an executed-case count assertion
  - a positive-controlled before/after census proving zero surviving references to the removed verbs outside the removal commit itself
  - this phase's close as the first real firing of the Phase 2 changie-fragments capability (evidence gathered by the verifier, not this plan)
affects: [phase-04-fragment-gate, phase-05-remove-release-please]

# Actuals (#2632)
actuals:
  tokens: 11997
  tasks: 3
  commits: 5

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "RED-first spawned-binary guard with an executed-case-count assertion (rule 84d1gfpywd), reused from the v0.14.0 verb-fold shape"
    - "Positive-controlled census (rg -nU -w --hidden 'codegraph\\s+(query|unlock)' ...) reused byte-for-byte from v0.14.0 to make a before/after count trustworthy"

key-files:
  created:
    - test/integration/removed_verbs_test.go
    - .planning/phases/03-rename-stub-removal/03-MUTATION-LOG.md
  modified:
    - internal/cli/root.go
    - internal/cli/testdata/cli-reference-allowlist.txt

key-decisions:
  - "Rewrote test/integration/renamed_stubs_test.go in place (git mv to removed_verbs_test.go) as the RED-first guard, rather than deleting it with no replacement — keeps a real spawned-binary assertion that the removed verbs stay unknown commands (D-02)."
  - "The removal is one feat(cli)! commit with a BREAKING CHANGE footer naming both replacements, preceded by a single test(03-01) RED commit — no intermediate commits touch internal/cli or test/integration (D-04)."
  - "No task wrote a changelog fragment. CAP-05's evidence is left entirely to the verify:post capability dispatch and the phase verifier (D-08)."

requirements-completed: [VERB-09, CAP-05]

coverage:
  - id: D1
    description: "codegraph query and codegraph unlock are now unknown commands, exiting 1 with cobra's own unknown-command stderr line, with no alias, hint or SuggestFor reviving either name"
    requirement: VERB-09
    verification:
      - kind: integration
        ref: "test/integration/removed_verbs_test.go#TestRemovedVerbsAreUnknownCommands"
        status: pass
      - kind: integration
        ref: "internal/cli/cli_reference_test.go#TestEveryRegisteredFlagIsAccountedFor"
        status: pass
    human_judgment: false
  - id: D2
    description: "No other generated artifact changed: docs/CLI-REFERENCE.md is byte-unchanged, the drift gate and flag-accounting/grouping tests are green, and a positive-controlled census reports zero surviving references outside the removal commit"
    requirement: VERB-09
    verification:
      - kind: integration
        ref: "task docs:cli:drift"
        status: pass
      - kind: other
        ref: "rg -nU -w --hidden 'codegraph\\s+(query|unlock)' census, Families (c)/(d), 03-MUTATION-LOG.md"
        status: pass
    human_judgment: false
  - id: D3
    description: "This phase's close is the first observed firing of the changie-fragments capability (CAP-05), writing the milestone's Breaking fragment"
    requirement: CAP-05
    verification: []
    human_judgment: true
    rationale: "The fragment is written by the verify:post capability dispatch at phase close, after this plan returns — this plan deliberately writes no fragment (D-08). Only the phase verifier, running after aggregate_results dispatches verify:post, can observe whether the capability fired, needed a D-10 correction, or fell back to D-11's hand-written path."

# Metrics
duration: ~25min
completed: 2026-09-26
status: complete
---

# Phase 3 Plan 1: Rename-Stub Removal Summary

**The hidden `query` and `unlock` commands are removed; running either is now an unknown-command error that exits 1, with `search --full <term>` and `daemon unlock [path]` as the replacements.**

## Performance

- **Duration:** ~25 min
- **Started:** 2026-09-26T20:21:00Z (approximate)
- **Completed:** 2026-09-26T20:45:36Z
- **Tasks:** 3
- **Files modified:** 7

## Accomplishments

- The hidden `query` and `unlock` commands are removed from `codegraph`. Invoking either now prints cobra's own `unknown command "..." for "codegraph"` line to stderr and exits 1, with empty stdout — no rename message, no alias, no hint. Run `search --full <term>` instead of `query <term>`, and `daemon unlock [path]` instead of `unlock [path]`.
- `test/integration/removed_verbs_test.go` (rewritten from `renamed_stubs_test.go`) is a spawned-binary guard, `TestRemovedVerbsAreUnknownCommands`, asserting all four old invocations (`query`, `query --json`, `unlock -p <dir>`, `unlock -p /nonexistent`) reach the unknown-command path, with a positive executed-case-count assertion (rule 84d1gfpywd).
- `internal/cli/renamed.go`, `internal/cli/renamed_test.go`, the two `AddCommand` registrations, and the two allowlist entries are gone; `root.go`'s comments now record the v0.15.0 removal instead of describing the stubs.
- A positive-controlled census (the v0.14.0 instrument, reused byte-for-byte) reports 4 surviving references on the pre-removal tree and 0 on HEAD, proving nothing else in the tree still names the removed verbs.
- No task wrote a changelog fragment — this phase's close is intended to be the first real firing of the Phase 2 changie-fragments capability (CAP-05), which the phase verifier observes and records after `execute-phase`'s `aggregate_results` dispatches `verify:post`.

## Task Commits

Each task was committed atomically:

1. **Task 1: Tracer, RED first — the spawned binary answers `query`/`unlock` with cobra's unknown-command error, through one feat(cli)! removal**
   - `c9df8e62` — `test(03-01): expect the removed query and unlock verbs to be unknown commands (VERB-09)` (test, RED)
   - `90dc419a` — `feat(cli)!: remove the query and unlock rename stubs` (feat, GREEN, BREAKING CHANGE)
   - `4e8220d4` — `docs(03-01): record the RED-first and GREEN transcripts for the removed-verb guard` (docs)
2. **Task 2: Expand — the removal changes nothing else (D-03), and both guards go RED against a revived verb or re-added allowlist entries**
   - `3bd9e132` — `docs(03-01): record regression gates and the guard mutation families (VERB-09)` (docs)
3. **Task 3: Expand — the v0.14.0 census reports zero surviving references, and the commit history has D-04's shape**
   - `0ab1be61` — `docs(03-01): record the before and after census and the commit-history audit (VERB-09)` (docs)

**Plan metadata:** committed alongside this SUMMARY, STATE.md and ROADMAP.md.

## TDD Gate Compliance

**RED (`test(03-01)`, commit `c9df8e62`):** `GOTOOLCHAIN=go1.26.6 go test ./test/integration/ -run '^TestRemovedVerbsAreUnknownCommands$' -count=1 -v` failed with all four subtests failing on the stderr assertion (the still-registered stubs' rename text), an assertion failure, not a build failure:

```
    removed_verbs_test.go:61: [query main]: stderr = "\"query\" has been renamed to \"search --full\" — run: codegraph search --full <term>\nthe \"query\" stub is removed in the next minor release (v0.15.0)\n", want exactly "unknown command \"query\" for \"codegraph\"\n" (cobra's own unknown-command line, VERB-09/D-01)
    removed_verbs_test.go:61: [query --json main]: stderr = "\"query\" has been renamed to \"search --full\" — run: codegraph search --full <term>\nthe \"query\" stub is removed in the next minor release (v0.15.0)\n", want exactly "unknown command \"query\" for \"codegraph\"\n" (cobra's own unknown-command line, VERB-09/D-01)
    removed_verbs_test.go:61: [unlock -p /var/folders/_b/3hyf5qvs62q0wh2vyh856z580000gn/T/TestRemovedVerbsAreUnknownCommands3880884257/001]: stderr = "\"unlock\" has been renamed to \"daemon unlock\" — run: codegraph daemon unlock [path]\nthe \"unlock\" stub is removed in the next minor release (v0.15.0)\n", want exactly "unknown command \"unlock\" for \"codegraph\"\n" (cobra's own unknown-command line, VERB-09/D-01)
    removed_verbs_test.go:61: [unlock -p /nonexistent]: stderr = "\"unlock\" has been renamed to \"daemon unlock\" — run: codegraph daemon unlock [path]\nthe \"unlock\" stub is removed in the next minor release (v0.15.0)\n", want exactly "unknown command \"unlock\" for \"codegraph\"\n" (cobra's own unknown-command line, VERB-09/D-01)
    removed_verbs_test.go:74: removed-verb guard: executed 4 of 4 cases
--- FAIL: TestRemovedVerbsAreUnknownCommands (0.64s)
    --- FAIL: TestRemovedVerbsAreUnknownCommands/query_bare (0.01s)
    --- FAIL: TestRemovedVerbsAreUnknownCommands/query_with_--json_flag (0.01s)
    --- FAIL: TestRemovedVerbsAreUnknownCommands/unlock_bare (0.01s)
    --- FAIL: TestRemovedVerbsAreUnknownCommands/unlock_with_nonexistent_path_arg (0.01s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/test/integration	5.668s
FAIL
```

**GREEN (`feat(cli)!`, commit `90dc419a`):** the same guard passed 4 of 4 immediately after the removal commit.

**Commit order:** `test(03-01)` (`c9df8e62`) precedes `feat(cli)!` (`90dc419a`) — verified by `git log --reverse --format=%s` over the range, and no other commit touching `internal/cli` or `test/integration` sits between them.

**`gsd-tools check tdd-red-evidence` was not used** to certify this RED — it is TAP-only and reports `INVALID_RED` against every Go test (rule x1cjy9vyhq, 03-RESEARCH.md Pitfall 1). RED is certified by the pasted `--- FAIL:` transcript above, showing the named test failing on an assertion for the planned behavior.

## Files Created/Modified

- `test/integration/removed_verbs_test.go` (renamed from `renamed_stubs_test.go`) — the RED-first spawned-binary guard, `TestRemovedVerbsAreUnknownCommands`
- `internal/cli/root.go` — dropped the two `AddCommand` registrations; updated four comment blocks to record the v0.15.0 removal instead of describing the stubs
- `internal/cli/testdata/cli-reference-allowlist.txt` — removed the two `codegraph query`/`codegraph unlock` allowlist lines, leaving 6 header lines plus 3 entries (`man`, `hook`, `hook pretooluse`)
- `internal/cli/renamed.go`, `internal/cli/renamed_test.go` — deleted
- `.planning/phases/03-rename-stub-removal/03-MUTATION-LOG.md` — new: RED-first/GREEN transcripts, regression gates, Families (a)-(d), commit-history audit

## Decisions Made

See `key-decisions` in the frontmatter above.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None. `task test:unit` and `task test:integration` were both green on the first attempt — no load flake was hit, so no rerun-alone/rerun-whole cycle was needed.

## User Setup Required

None - no external service configuration required.

## Pointers into 03-MUTATION-LOG.md

- `## RED-first transcript (Task 1)` and `## GREEN after the removal (Task 1)` — the TDD gate evidence above, in full.
- `## Regression gates (Task 2)` — `task docs:cli`, `task docs:cli:drift`, `task test:unit`, `task test:integration`, all green with no unexpected diff.
- `## Family (a)` and `## Family (b)` (Task 2) — the removed-verb guard demonstrated RED against a revived alias and a revived `SuggestFor` hint; the allowlist guard demonstrated RED against the two re-added entries. All mutations ran in a disposable scratch worktree, confirmed applied, reverted byte-clean.
- `## Family (c)` and `## Family (d)` (Task 3) — the v0.14.0 census instrument, byte-for-byte, run before removal (4 hits across 2 files) and after removal (0 hits), each preceded by a positive control found on `:1:`/`:3:`.
- `## Commit-history audit (Task 3)` and `## Verdict (Task 3)` — the commit range from `84778a82` has exactly one `test(03-01):` commit followed by one `feat(cli)!:` commit, no CI-skip marker, and every subject matches the pr-title grammar.

## CAP-05 Note

CAP-05's evidence is produced by the `verify:post` capability dispatch at this phase's close, not by any task in this plan — no task wrote a changelog fragment (D-08). The phase verifier, running after `execute-phase`'s `aggregate_results` step dispatches `verify:post` and before `verify_phase_goal` runs `gsd-verifier`, owns the verdict: whether the capability fired as `Breaking` (D-08), fired but needed a D-10 correction, or did not fire and fell back to the D-11 hand-written path.

## Next Phase Readiness

VERB-09 is fully satisfied by this plan's own evidence: the removed verbs are unknown commands, no other generated artifact drifted, and the census confirms zero surviving references. CAP-05 is not yet closed by this plan — it depends on the verifier's observation of the phase close, which happens after this SUMMARY is committed. No blockers for Phase 4 (the fragment gate) or Phase 5 (removing release-please).

---
*Phase: 03-rename-stub-removal*
*Completed: 2026-09-26*

## Self-Check: PASSED

- FOUND: test/integration/removed_verbs_test.go
- FOUND: .planning/phases/03-rename-stub-removal/03-MUTATION-LOG.md
- CONFIRMED DELETED: internal/cli/renamed.go
- FOUND commit c9df8e62 (test(03-01))
- FOUND commit 90dc419a (feat(cli)!)
- FOUND commit 4e8220d4 (docs(03-01))
- FOUND commit 3bd9e132 (docs(03-01))
- FOUND commit 0ab1be61 (docs(03-01))
