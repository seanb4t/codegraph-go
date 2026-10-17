---
phase: 05-release-chain-release-please-retirement
plan: 04
subsystem: infra
tags: [pr-hygiene, fragment-policy, regex, redos, changie, github-actions, guard-tests]

requires:
  - phase: 05-release-chain-release-please-retirement
    provides: "05-01: release-pr.yml and the changie release PR whose paths (.changes/**, CHANGELOG.md) this plan exempts; 05-MUTATION-LOG.md convention"
provides:
  - "pr_template_policy.EXEMPT_PREFIXES gains .changes/; EXEMPT_EXACT loses the two retired manifest names (D-05, GRD-15)"
  - "pr_template_policy.is_housekeeping(paths) and pr_template_policy.find_exemption(body), both pure and used by main()"
  - "require-issue-link.yml: .changes/* case arm replaces the retired manifest arm; comments and notice updated"
  - "test_hygiene_allowlists_mirror_require_issue_link: the 'keep in sync' comment is now a set-equality test over the workflow's case arms"
  - "EXEMPT_MARKER and CHANGELOG_EXEMPT_MARKER are linear: reason captured whole, validated after the match; a reason starting with a dash exempts nothing in either policy (WR-01)"
  - "HygieneAllowlistTests (6) and MarkerGrammarTests (4); check:fragment-policy floor 46"
  - "05-MUTATION-LOG.md: 05-04 RED and GREEN transcripts for both swaps, Families 05-04/a-f"
affects: [05-05, 05-06, 05-07, phase-06-live-run]

actuals:
  tokens: 11503
  tasks: 3
  commits: 8

tech-stack:
  added: []
  patterns:
    - "an allowlist mirrored between Python and a shell case block is held equal by a test that parses the arms from the workflow text and compares sets in both directions, with population floors against a vacuous parse"
    - "a marker regex captures its free-text field whole and validates it after the match, instead of encoding the validation in the pattern"
    - "a timing test with an explicit adversarial body and a 1.0 s bound, sized so a RED run against the old pattern finishes in seconds"

key-files:
  created: []
  modified:
    - scripts/pr_template_policy.py
    - scripts/fragment_policy.py
    - scripts/test_fragment_policy.py
    - .github/workflows/require-issue-link.yml
    - Taskfile.yml
    - .planning/phases/05-release-chain-release-please-retirement/05-MUTATION-LOG.md

key-decisions:
  - "Tests live in scripts/test_fragment_policy.py, which already imports pr_template_policy and is already gated by check:fragment-policy's executed-count guard; no new file, target or CI step"
  - "Grammar (RESEARCH Pitfall 7): a reason that starts with a dash after strip does not exempt, in either policy; the old accidental acceptance of '- ' and '--x' through a leading space is closed"
  - "require-issue-link.yml changes only for D-05 (its allowlist arm); its marker check is a single grep -E with no nested quantifier and is not a Python backtracking regex, so WR-01 is not claimed for it (rule eyva2svrcz)"
  - "A .changes/-only (fragment-only) PR is housekeeping for the template and issue-link checks; fragment-required is unaffected because it looks only at cmd/** and internal/**"

patterns-established:
  - "_workflow_allowlist in the policy tests: parse a shell case block's empty-body arms from file text"

requirements-completed: []

coverage:
  - id: D1
    description: "A PR touching only .changes/** and CHANGELOG.md, or only a fragment, is housekeeping in pr_template_policy; shipped code beside .changes/, the empty list, .changesX/ and .Changes/ are not"
    requirement: GRD-15
    verification:
      - kind: unit
        ref: "scripts/test_fragment_policy.py#HygieneAllowlistTests (test_release_pr_paths_are_housekeeping, test_fragment_only_pr_is_housekeeping, test_shipped_code_beside_changes_is_not_housekeeping, test_empty_path_list_is_not_housekeeping)"
        status: pass
      - kind: other
        ref: "Family 05-04/e in 05-MUTATION-LOG.md (all four fail on their own assertTrue/assertFalse)"
        status: pass
    human_judgment: false
  - id: D2
    description: "EXEMPT_PREFIXES and EXEMPT_EXACT are exactly the documented sets and the retired manifest names are gone"
    requirement: GRD-15
    verification:
      - kind: unit
        ref: "scripts/test_fragment_policy.py#test_exempt_sets_are_the_documented_sets"
        status: pass
      - kind: other
        ref: "Family 05-04/a in 05-MUTATION-LOG.md"
        status: pass
    human_judgment: false
  - id: D3
    description: "pr_template_policy's EXEMPT_* sets and require-issue-link.yml's case arms are equal in both directions, over at least 4 prefixes and 14 exact names"
    requirement: GRD-15
    verification:
      - kind: unit
        ref: "scripts/test_fragment_policy.py#test_hygiene_allowlists_mirror_require_issue_link"
        status: pass
      - kind: other
        ref: "Family 05-04/b and its variant in 05-MUTATION-LOG.md"
        status: pass
    human_judgment: false
  - id: D4
    description: "Both exemption markers decide a 2,500-space and a 60,000-space adversarial body in microseconds (bound 1.0 s); the old patterns took 2.8 to 3.9 s per call"
    requirement: GRD-15
    verification:
      - kind: unit
        ref: "scripts/test_fragment_policy.py#test_exempt_markers_are_linear_on_adversarial_bodies"
        status: pass
      - kind: other
        ref: "Family 05-04/c in 05-MUTATION-LOG.md (cubic markers restored, four timing subTests fail with measured times)"
        status: pass
    human_judgment: false
  - id: D5
    description: "Unterminated, empty, whitespace and dash-led reasons exempt nothing; the two find_exemption functions agree over a 10-row table and the first marker decides"
    requirement: GRD-15
    verification:
      - kind: unit
        ref: "scripts/test_fragment_policy.py#test_unterminated_marker_does_not_exempt, test_marker_reason_starting_with_dash_does_not_exempt, test_marker_semantics_mirror_pr_template_exempt"
        status: pass
      - kind: other
        ref: "Family 05-04/f in 05-MUTATION-LOG.md (13 own-assertion failures)"
        status: pass
    human_judgment: false
  - id: D6
    description: "check:fragment-policy binds all 46 tests (floor 46) and a skipped test fails it"
    requirement: GRD-15
    verification:
      - kind: other
        ref: "task check:fragment-policy prints 'executed 46 of 46 defined test methods (floor 46)'; Family 05-04/d prints 'result line is not exactly OK'"
        status: pass
    human_judgment: false

duration: 10min
completed: 2026-10-14
status: complete
plan_head_before: 725b4f815ea55e39ae73bff000b3c9acd6e87414
plan_head_after: 0945d1081c3ba8ac6f8622dc35e96285e6a12cfa
commits: 8
---

# Phase 5 Plan 04: Hygiene gates re-pointed at the changie release PR, WR-01 closed Summary

**The template and issue-link hygiene gates now exempt the changie release PR by path (`.changes/` plus CHANGELOG.md), a parity test holds the Python and shell allowlists equal, and both exemption markers run in linear time (60,000-character adversarial body: 3.8 s before, 0.3 ms after).**

This plan changes CI hygiene scripts, one workflow's shell arm and one Taskfile floor. No codegraph user-visible behavior changes. Nothing was pushed, no tag was created, no PR was touched.

## Performance

- **Duration:** about 10 min (start time was not captured at launch; the plan-head ledger was written at 21:55:04Z, and the last task commit landed at 22:01Z)
- **Tasks:** 3 of 3
- **Files modified:** 6 (5 code and config files, 1 planning artifact)

## Accomplishments

- `pr_template_policy.py`: `.changes/` joins `EXEMPT_PREFIXES`; the two retired manifest names leave `EXEMPT_EXACT`; `is_housekeeping(paths)` is a pure function that `main()` uses. An empty list, shipped code beside `.changes/`, `.changesX/a.md` and `.Changes/a.md` are not housekeeping.
- `require-issue-link.yml`: `.changes/*) ;;` replaces the retired manifest arm; the comment now names the parity test; the path-exemption notice names `.changes/`. Nothing else in the file changed.
- `test_hygiene_allowlists_mirror_require_issue_link` parses the `case` arms from the workflow text and compares prefix and exact sets with `pr_template_policy` in both directions, with floors of 4 prefixes and 14 exact names. Neither file carries a comment naming the retired tool.
- WR-01: both markers are now `<!--\s*<name>:(?P<reason>[^>]*?)-->` with `re.I`; the reason is stripped and validated after the match (non-empty, not starting with `-`) in `find_exemption`, which now exists in both policies and is mirrored by a behavior table.
- `check:fragment-policy` floor 36 to 46; its desc names GRD-15 beside GATE-04.
- Families 05-04/a-f recorded in `05-MUTATION-LOG.md`, each reverted byte-clean.

## TDD Gate Compliance

Commit order (`git log --first-parent --no-merges --reverse` since `5a8603f5`, restricted to this plan):

| Gate | Task 1 (exemption swap) | Task 2 (WR-01) |
|---|---|---|
| RED | `1649aec9` test(05-04): expect the release PR's own paths to be housekeeping in both hygiene gates | `161a4a7a` test(05-04): expect linear-time exemption markers with mirrored grammar |
| GREEN | `7dc38cc2` ci(05-04): exempt the changie release PR by path in both hygiene gates | `164e7af2` ci(05-04): make both exemption markers linear and keep their grammar in lockstep |
| Floor | | `44b004ef` ci(05-04): raise the fragment-policy floor to the 46 defined tests |
| Transcripts | `561ace7f` docs(05-04) | `cf430a6d` docs(05-04) |
| Mutation families | `0945d108` docs(05-04): record mutation families 05-04/a-f | |

No REFACTOR commit: nothing needed cleanup. The `test(05-04):` commits touch exactly `scripts/test_fragment_policy.py`; the `ci(05-04):` commits touch no test file and no `.planning/` path.

`gsd-tools check tdd-red-evidence` was **skipped** under rule x1cjy9vyhq: it is TAP-only and reports INVALID_RED for a unittest run. RED evidence is the `test(05-04):` commit landing before its implementation plus the pasted transcripts below (full text in `05-MUTATION-LOG.md`, sections `## 05-04 RED transcript (exemption swap)` and `## 05-04 RED transcript (WR-01)`).

**RED 1** (`python3 -B -m unittest discover -s scripts -p 'test_fragment_policy.py' -v`; `Ran 42 tests`, `FAILED (failures=6)`, zero `ERROR:` lines):

```
FAIL: test_empty_path_list_is_not_housekeeping (test_fragment_policy.HygieneAllowlistTests.test_empty_path_list_is_not_housekeeping)
AssertionError: False is not true
FAIL: test_exempt_sets_are_the_documented_sets (test_fragment_policy.HygieneAllowlistTests.test_exempt_sets_are_the_documented_sets)
AssertionError: Items in the second set but not the first:
'.changes/'
FAIL: test_fragment_only_pr_is_housekeeping (test_fragment_policy.HygieneAllowlistTests.test_fragment_only_pr_is_housekeeping)
AssertionError: False is not true
FAIL: test_hygiene_allowlists_mirror_require_issue_link (test_fragment_policy.HygieneAllowlistTests.test_hygiene_allowlists_mirror_require_issue_link)
AssertionError: 3 not greater than or equal to 4 : parsed prefixes: ['.github/', '.planning/', 'docs/']
FAIL: test_release_pr_paths_are_housekeeping (test_fragment_policy.HygieneAllowlistTests.test_release_pr_paths_are_housekeeping)
AssertionError: False is not true
FAIL: test_shipped_code_beside_changes_is_not_housekeeping (test_fragment_policy.HygieneAllowlistTests.test_shipped_code_beside_changes_is_not_housekeeping)
AssertionError: False is not true
```

**RED 2** (`Ran 46 tests`, `FAILED (failures=7)`, zero `ERROR:` lines; the run took 13.5 s):

```
FAIL: test_exempt_markers_are_linear_on_adversarial_bodies (...) (marker='changelog-exempt', shape='cubic shape')
AssertionError: 2.893123999936506 not less than 1.0 : changelog-exempt cubic shape: search took 2.89s on a 2522-char body
FAIL: test_exempt_markers_are_linear_on_adversarial_bodies (...) (marker='changelog-exempt', shape='quadratic shape')
AssertionError: 3.828044166089967 not less than 1.0 : changelog-exempt quadratic shape: search took 3.83s on a 60024-char body
FAIL: test_exempt_markers_are_linear_on_adversarial_bodies (...) (marker='pr-template-exempt', shape='cubic shape')
AssertionError: 2.9537973750848323 not less than 1.0 : pr-template-exempt cubic shape: search took 2.95s on a 2524-char body
FAIL: test_exempt_markers_are_linear_on_adversarial_bodies (...) (marker='pr-template-exempt', shape='quadratic shape')
AssertionError: 3.8166788339149207 not less than 1.0 : pr-template-exempt quadratic shape: search took 3.82s on a 60026-char body
FAIL: test_marker_reason_starting_with_dash_does_not_exempt (...)
AssertionError: False is not true
FAIL: test_marker_semantics_mirror_pr_template_exempt (...)
AssertionError: False is not true
FAIL: test_unterminated_marker_does_not_exempt (...)
AssertionError: False is not true
```

Semantic assessment: the timing test fails on its own elapsed-time assertion for the planned reason (catastrophic backtracking). The `hasattr` failures (the four `is_housekeeping` tests, and the last three above) prove commit order only; their own-assertion RED is in Families 05-04/e and 05-04/f, where a function-body mutation makes every `hasattr` pass and each `FAIL:` is the test's own assertion.

## Marker timings, old versus new

`.search` on the bodies the timing test builds (bound 1.0 s):

| marker | body | old pattern | new pattern |
|---|---|---|---|
| changelog-exempt | 2,500 spaces after the opener (2522 chars) | 2.89 s | 0.018 ms |
| changelog-exempt | `a` then 60,000 spaces (60024 chars) | 3.83 s | 0.291 ms |
| pr-template-exempt | 2,500 spaces after the opener (2524 chars) | 2.95 s | 0.012 ms |
| pr-template-exempt | `a` then 60,000 spaces (60026 chars) | 3.82 s | 0.293 ms |

Family 05-04/c restores the old patterns and the same four calls fail again at 2.82 to 3.85 s.

## Decisions for the maintainer (two-way doors)

- **Dash-reason grammar.** A reason that starts with `-` after strip does not exempt, in either policy. That was the old pattern's evident intent (its first character class was `[^->]`); the old acceptance of `<!-- pr-template-exempt: - -->` and `<!-- pr-template-exempt: --x -->` through a leading space was accidental. This matches 04-REVIEW.md's proposed fix, and `test_marker_reason_starting_with_dash_does_not_exempt` states it.
- **require-issue-link.yml and WR-01.** It changes here only for D-05, its allowlist arm. Its marker check is one `grep -qiE` with no nested quantifier, and 04-REVIEW.md names WR-01's subjects as `fragment_policy.py:49` and `pr_template_policy.py:83`. Claiming anything about grep's matcher would be third-party behavior (rule eyva2svrcz). The orchestrator note listed the file under WR-01; this is a deliberate reading the maintainer can override.
- **Fragment-only PRs.** `.changes/` also exempts a fragment-only PR from the template and issue-link checks; fragment-required is unaffected because it looks only at `cmd/**` and `internal/**`.
- **Residual (T-05-18, accepted).** A body of many unterminated openers is still scanned once per opener, O(openers x length). At GitHub's roughly 65k-character body limit that is about a second, down from hours.

## Mutation families

All in `05-MUTATION-LOG.md`, each with a pre-mutation gate, applied-confirmation, RED guard output, revert and byte-clean proof:

- **05-04/a**: retired manifest name re-added to EXEMPT_EXACT. RED: `test_exempt_sets_are_the_documented_sets` and the parity test (exact side).
- **05-04/b**: `.changes/*) ;;` arm deleted. RED: the parity test, on the prefix floor naming the parsed prefix side. A variant (arm renamed `.changesx/*`) keeps the floor and reaches the set comparison: `prefix side differs: only in workflow ['.changesx/'], only in pr_template_policy ['.changes/']`.
- **05-04/c**: cubic markers restored. RED: the four timing subTests, with measured times; no other test turns red.
- **05-04/d**: one new test skipped. RED: `task check:fragment-policy` prints `result line is not exactly OK (skips are not executions)`, exit 201.
- **05-04/e**: `is_housekeeping` inverted. RED, each on its own `assertTrue`/`assertFalse`: the four housekeeping tests, and only those (`failures=4`).
- **05-04/f**: `pr_template_policy.find_exemption` always returns `"mutated"`. RED, each on its own `assertIsNone`/`assertEqual`: the unterminated test, both dash bodies, and 10 rows of the behavior table (`failures=13`).

## Task Commits

1. **Task 1: exemption swap** - `1649aec9` test (RED), `7dc38cc2` ci (GREEN), `561ace7f` docs (transcripts)
2. **Task 2: WR-01** - `161a4a7a` test (RED), `164e7af2` ci (GREEN), `44b004ef` ci (floor), `cf430a6d` docs (transcripts)
3. **Task 3: mutation families** - `0945d108` docs

## Files Created/Modified

- `scripts/pr_template_policy.py` - `.changes/` prefix, retired names removed, `is_housekeeping`, `find_exemption`, linear `EXEMPT_MARKER`
- `scripts/fragment_policy.py` - linear `CHANGELOG_EXEMPT_MARKER`, dash-led reasons rejected in `find_exemption`
- `scripts/test_fragment_policy.py` - `HygieneAllowlistTests` (6), `MarkerGrammarTests` (4), `_workflow_allowlist`; imports `time` and `pathlib.Path` only
- `.github/workflows/require-issue-link.yml` - `.changes/*` arm, comment, notice line
- `Taskfile.yml` - `check:fragment-policy` floor 46 and desc
- `.planning/phases/05-release-chain-release-please-retirement/05-MUTATION-LOG.md` - 05-04 sections and Families 05-04/a-f

## Decisions Made

See `key-decisions` above. Each follows the plan as written, apart from the deviations below.

## Deviations from Plan

### Auto-fixed Issues

None needing a code fix. Two departures from the plan's text, both on the side of a stronger guard:

**1. [Plan inconsistency] The parity test fails at RED, not passes**
- **Found during:** Task 1 RED
- **Issue:** The plan predicted `test_hygiene_allowlists_mirror_require_issue_link` would pass at RED "because today's lists already agree". Its own population floor (at least 4 prefixes, must_haves truth 3) fails at RED, because the shell list had only 3 prefix arms before the `.changes/*` arm was added.
- **Fix:** Kept the floor (it is a must-have and guards a vacuous parse). The RED run is still 6 `FAIL`, 0 `ERROR`. The set comparison's own RED is Family 05-04/b's variant.
- **Files modified:** none beyond the test; recorded in the RED 1 section of the log.

**2. [Coverage addition] Family 05-04/b ran a second mutation**
- **Found during:** Task 3
- **Issue:** Deleting the `.changes/*` arm trips the prefix floor (3 < 4) before the prefix set comparison runs, so the comparison's own prefix assertion was never seen failing.
- **Fix:** Added a recorded variant inside Family b (arm renamed `.changesx/*`), which keeps the floor satisfied and fails on the comparison, naming both sides. Reverted byte-clean.

**Total deviations:** 0 auto-fixed, 2 recorded departures. **Impact:** none on the delivered behavior; both strengthen the evidence.

## Issues Encountered

None.

## Known Stubs

None.

## Threat Flags

None. T-05-16 (backtracking) and T-05-19 (allowlist drift) are mitigated and held by Families 05-04/c and 05-04/b; T-05-17 and T-05-18 are accepted as planned.

## User Setup Required

None.

## Next Phase Readiness

GRD-15 is **partial** after this plan: its hygiene gates and WR-01 are done and held by tests and mutations here, but plan 05-06 (docs) and others also declare it, so `requirements-completed` is `[]` and GRD-15 stays open for the orchestrator to reconcile at phase close. Nothing in this plan blocks 05-05, 05-06 or 05-07. Plan 05-06 touches none of this plan's files.

## Self-Check: PASSED

- Files: `scripts/pr_template_policy.py`, `scripts/fragment_policy.py`, `scripts/test_fragment_policy.py`, `.github/workflows/require-issue-link.yml`, `Taskfile.yml`, `05-MUTATION-LOG.md` all exist; `git status --porcelain -- scripts .github Taskfile.yml` is empty.
- Commits: all 8 hashes above are ancestors of HEAD; `git rev-list --count 725b4f81..HEAD` was 8 before this SUMMARY.
- Verification: `task check:fragment-policy` prints `executed 46 of 46 defined test methods (floor 46)`; `task lint:actions` exits 0; no census hit in the four hygiene files; the Task 3 verify block exits 0; `git worktree list` shows no `p05-scratch-wt`.
