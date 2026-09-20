---
phase: 04-cli-glow-up
plan: 01
subsystem: testing
tags: [cli, golden-test, cobra, pflag, tdd-mutation-log, no-color]

# Dependency graph
requires: []
provides:
  - "29 frozen plain-output goldens at internal/cli/testdata/plain/*.golden, captured before any renderer/resolver/palette/--color flag exists"
  - "TestPlainGolden + TestNoColorNonTTYRegression (internal/cli/plain_golden_test.go) — the byte-equality + zero-ESC verify gate every later 04-NN plan's <verify> runs"
  - "TestShortFlagsConsistent (internal/cli/short_flags_test.go) — the pinned CLI-07 short-flag invariant"
  - "04-MUTATION-LOG.md Families (a)/(b) — both guards demonstrated RED against confirmed mutations"
affects: [04-02, 04-03, 04-04, 04-05, 04-06, 04-07, 04-08]

# Actuals (#2632)
actuals:
  tokens: 10020
  tasks: 3
  commits: 1

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Golden-freeze-before-measurement: capture the CURRENT plain output as testdata/*.golden BEFORE any implementation lands, so later plans' <verify> proves byte-identity rather than a reviewer's memory"
    - "Positive-counted table-walk tests (cliReferenceTree + shorthand lookup) instead of hand-enumerated assertions, with a floor guarding against a walk that silently inspects nothing"
    - "sync.Once-cached real-process invocation for a command whose underlying SDK closes a process-global resource (os.Stdin) on session end, so it can be exercised for real exactly once per test binary instead of never or unsafely-repeatedly"

key-files:
  created:
    - internal/cli/plain_golden_test.go
    - internal/cli/short_flags_test.go
    - internal/cli/testdata/plain/*.golden (29 files)
    - .planning/phases/04-cli-glow-up/04-MUTATION-LOG.md
  modified: []

key-decisions:
  - "Captured 29 goldens, not exactly 28 — D-16's own verb enumeration (status, search, search --full, callers, callees, impact, affected, files×2, explore, node×2, version, telemetry, init, index --force, sync, uninit --force, githooks×3, install, uninstall, daemon×3, upgrade, ui, serve --mcp) counts to 29. Every plan/must-have floor is phrased as '>= 28', so 29 satisfies every check; the extra case is the complete, correct set, not a discrepancy."
  - "install-local/uninstall-local use --target claude, not the plan's literal --target claude-code — 'claude-code' is not a registered agents.TargetID (the real id is 'claude', confirmed against internal/agents/types.go and every existing install_test.go invocation); a literal claude-code invocation would error instead of producing a capturable golden. Rule 3 (blocking issue) auto-fix."
  - "Two case-setup orderings were swapped (daemon-stop-nomatch, serve-mcp-stderr): copyFixture/setupIndexedFixture resolves the shared fixture via a package-relative path, which breaks once fakeHome's t.Chdir has already moved the process cwd — fixed by always calling setupIndexedFixture before fakeHome in those two cases (Rule 1 — bug found during Task 2)."
  - "Golden read/write paths are resolved to ABSOLUTE paths (filepath.Abs) before any case's setup() runs, not after — several cases' setup (fakeHome) chdirs into a fresh temp project directory, and a relative 'testdata/plain/…' path computed afterward silently reads/writes the wrong location (confirmed empirically: the first golden-generation run appeared to write 5 files that were actually never written to internal/cli/testdata/plain/ at all). Rule 1 (bug) auto-fix, caught before generating the final goldens."
  - "serve-mcp-stderr is executed via a package-level sync.Once, not per NO_COLOR variant — internal/mcp's goSDKServer.ServeStdio reads directly from the process-global os.Stdin (never cmd.InOrStdin(), so execCmd's configured input has no effect on it), and stdinLingerReader.Close() closes that same *os.File at session end. A second in-process `serve --mcp` invocation anywhere in the test binary therefore always fails with 'read /dev/stdin: file already closed' — confirmed empirically. Since this stderr banner is unstyled, NO_COLOR-invariant text (no present renderer exists on this path yet), caching one real invocation's result and reusing it for TestPlainGolden and both NO_COLOR variants asserts the same byte-identity property without re-triggering the SDK's fd close. internal/mcp is out of scope for this plan (D-16's own prohibitions), so no production fix was made. Rule 1 (bug in the test harness design, not production) auto-fix."
  - "TestPlainGolden/TestNoColorNonTTYRegression's byte-equality and zero-ESC assertions run independently (t.Errorf, never a short-circuiting t.Fatalf) — the plan's own acceptance criteria required a planted ESC-byte mutation to surface BOTH the byte-mismatch and the ESC-present messages in the same failing subtest; the initial Fatalf-based version only ever reported whichever check ran first."

patterns-established:
  - "Every plan-01 golden capture goes through the existing execCmd/copyFixture/setupIndexedFixture/fakeHome/initGitRepo corpus — no second harness introduced."

requirements-completed: [CLI-05, CLI-07]

coverage:
  - id: D1
    description: "29 plain-output goldens frozen from the pre-glow-up tree, one per verb that gains a styled branch this phase"
    requirement: "CLI-05"
    verification:
      - kind: unit
        ref: "internal/cli/plain_golden_test.go#TestPlainGolden"
        status: pass
    human_judgment: false
  - id: D2
    description: "Byte-equality + zero-ESC held under NO_COLOR unset, =1 and =banana (the gh #13335 lesson plus the D-09 'banana' research correction, pinned before the resolver exists)"
    requirement: "CLI-05"
    verification:
      - kind: unit
        ref: "internal/cli/plain_golden_test.go#TestNoColorNonTTYRegression"
        status: pass
    human_judgment: false
  - id: D3
    description: "CLI-07's already-true short-flag invariant pinned by a positive-counted tree walk (20 pairs across 9 query verbs, floor 18)"
    requirement: "CLI-07"
    verification:
      - kind: unit
        ref: "internal/cli/short_flags_test.go#TestShortFlagsConsistent"
        status: pass
    human_judgment: false
  - id: D4
    description: "Both guards demonstrated RED against a confirmed-applied mutation and reverted byte-clean (Families a and b)"
    verification:
      - kind: other
        ref: ".planning/phases/04-cli-glow-up/04-MUTATION-LOG.md"
        status: pass
    human_judgment: false

# Metrics
duration: 22min
completed: 2026-09-17
status: complete
---

# Phase 4 Plan 1: Freeze the Plain-Output Bar Summary

**29 plain-output goldens frozen from the pre-glow-up tree, pinned byte-equal + zero-ESC under NO_COLOR unset/1/banana, plus a positive-counted CLI-07 short-flag walk — both guards demonstrated RED against a confirmed mutation and reverted byte-clean, with zero production files touched.**

## Performance

- **Duration:** 22 min
- **Started:** 2026-09-17T20:15:44Z
- **Completed:** 2026-09-17T20:37:53Z
- **Tasks:** 3
- **Files modified:** 32 (2 test files, 29 golden fixtures, 1 mutation log)

## Accomplishments

- Froze `internal/cli/testdata/plain/*.golden` for every human-output verb that gains a styled branch this phase (query verbs, lifecycle/agent verbs, one-line verbs, `ui`, `serve --mcp`) — 29 files, above the plan's own `>= 28` floor.
- `TestPlainGolden` asserts byte-equality + zero ESC bytes against every golden, with a positive floor and a set-equality check between cases and golden files (no orphan golden, no case with no golden).
- `TestNoColorNonTTYRegression` re-runs the same table under `NO_COLOR=1` and `NO_COLOR=banana`, pinning both the gh #13335 lesson and D-09's research correction that any non-empty `NO_COLOR` value disables color today.
- `TestShortFlagsConsistent` pins CLI-07's already-true short-flag invariant with a positive-counted walk of the nine D-13 Query-the-graph verbs (20 pairs inspected, floor 18).
- `04-MUTATION-LOG.md` records Family (a) — a planted ESC byte in `status.go` turning both golden tests RED across all three `NO_COLOR` states — and Family (b) — a long-only `--limit` on `callers.go` turning the short-flag walk RED, naming `callers --limit`. Both reverted byte-clean.

## Task Commits

All three tasks land in ONE commit, per this plan's own contract (no production source changes across any task, so nothing to commit until the full test suite + mutation log exist):

1. **Task 1: Golden harness + the twelve query-verb plain goldens** — folded into the single commit below
2. **Task 2: The sixteen lifecycle/agent/one-line goldens + Family (a) RED** — folded into the single commit below
3. **Task 3: CLI-07 short-flag pinning test + Family (b) RED + the plan's single commit** — `71494af3`

**Plan metadata:** committed alongside SUMMARY.md/STATE.md/ROADMAP.md/REQUIREMENTS.md in the metadata commit that follows this SUMMARY.

## Files Created/Modified

- `internal/cli/plain_golden_test.go` — `plainCase`/`plainRun` table, `TestPlainGolden`, `TestNoColorNonTTYRegression`, the `-update-plain-goldens` flag
- `internal/cli/short_flags_test.go` — `TestShortFlagsConsistent`, `queryVerbNames`, `shortFlagPairs`
- `internal/cli/testdata/plain/*.golden` — 29 frozen plain outputs
- `.planning/phases/04-cli-glow-up/04-MUTATION-LOG.md` — Families (a) and (b)

## Decisions Made

See `key-decisions` in frontmatter for the full rationale on each. Summary: captured 29 goldens (not exactly 28 — every floor in the plan is `>= 28`); used the real `agents.TargetID` `"claude"` instead of the plan's literal `"claude-code"` (not a registered target); fixed two fixture-setup ordering bugs (`setupIndexedFixture` before `fakeHome`, not after); resolved golden paths to absolute before any `t.Chdir`-performing setup runs; cached the one safe real `serve --mcp` invocation via `sync.Once` to work around a real, out-of-scope `internal/mcp` process-global `os.Stdin`-close side effect; and made the golden/ESC assertions independent (`t.Errorf`, not `t.Fatalf`) so a mutation tripping both surfaces both messages.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] `install --target claude-code` is not a valid target id**
- **Found during:** Task 2 (install-local/uninstall-local goldens)
- **Issue:** The plan's literal text specified `install --target claude-code --location local`, but `agents.TargetID` for Claude Code is `"claude"` (confirmed in `internal/agents/types.go` and every existing `install_test.go` invocation) — `claude-code` is not registered and would error instead of producing a capturable golden.
- **Fix:** Used `--target claude` for both `install-local` and `uninstall-local`.
- **Files modified:** `internal/cli/plain_golden_test.go` (never committed with the wrong value — caught before the first golden generation)
- **Verification:** `TestPlainGolden/install-local` and `/uninstall-local` pass; goldens show `Claude Code: configured` / `Claude Code: removed`.
- **Committed in:** `71494af3` (part of the plan's single commit)

**2. [Rule 1 - Bug] `copyFixture`'s package-relative path resolution breaks after `fakeHome`'s `t.Chdir`**
- **Found during:** Task 2 (daemon-stop-nomatch, serve-mcp-stderr goldens) — first golden-generation attempt failed with `copy fixture: lstat .../indexer/testdata/gofixture: no such file or directory`.
- **Issue:** `copyFixture` resolves the shared gofixture via `filepath.Abs(filepath.Join("..", "indexer", "testdata", "gofixture"))`, relative to the process's current working directory. `fakeHome(t)` calls `t.Chdir` into a fresh temp project directory. The two cases called `fakeHome(t)` BEFORE `setupIndexedFixture(t)`, so the relative fixture path resolved against the wrong directory.
- **Fix:** Swapped the call order in both cases: `setupIndexedFixture(t)` (or the fixture copy) now always runs before `fakeHome(t)`.
- **Files modified:** `internal/cli/plain_golden_test.go`
- **Verification:** Both cases pass; `TestPlainGolden` green end to end.
- **Committed in:** `71494af3`

**3. [Rule 1 - Bug] Golden read/write path computed AFTER a chdir-performing setup silently targets the wrong directory**
- **Found during:** Task 2 — after fixing deviation #2, the update-golden run reported "wrote golden testdata/plain/install-local.golden" (and 4 other `fakeHome`-using cases) but a subsequent `ls` showed only 24 of 29 files actually present in `internal/cli/testdata/plain/`.
- **Issue:** `goldenPath := filepath.Join("testdata", "plain", c.name+".golden")` was computed AFTER `run := c.setup(t)`, and `fakeHome`'s `t.Chdir` had already moved the process cwd by then — the relative golden path silently resolved (and wrote) inside the ephemeral fake-home/fake-project temp directory instead of the package's real `testdata/plain/` directory, which is then deleted by `t.TempDir()`'s own cleanup, and appears to succeed (no error) while writing nowhere useful.
- **Fix:** Resolve `goldenPath` to an ABSOLUTE path via `filepath.Abs` BEFORE calling `c.setup(t)`, in both `TestPlainGolden` and `TestNoColorNonTTYRegression`.
- **Files modified:** `internal/cli/plain_golden_test.go`
- **Verification:** Re-ran `-update-plain-goldens`; `ls internal/cli/testdata/plain/*.golden | wc -l` now reports 29; full suite green.
- **Committed in:** `71494af3`

**4. [Rule 1 - Bug] A second in-process `serve --mcp` invocation always fails ("file already closed")**
- **Found during:** Task 2 (TestNoColorNonTTYRegression's second and third `serve-mcp-stderr` runs)
- **Issue:** `internal/mcp`'s `goSDKServer.ServeStdio` reads directly from the process-global `os.Stdin` (bypassing `cmd.InOrStdin()` entirely, so `execCmd`'s configured empty-string reader has no effect on it), and its `stdinLingerReader.Close()` closes that same `*os.File` when the session ends. Any subsequent `serve --mcp` invocation in the same test binary process then fails with `read /dev/stdin: file already closed` — reproduced deterministically. `internal/mcp` is explicitly out of scope for this plan (D-16's own prohibitions forbid editing it), so this could not be fixed at the root.
- **Fix:** The `serve-mcp-stderr` case now executes the real command exactly once per test binary, behind a package-level `sync.Once`, and every subsequent call (across `TestPlainGolden` and both `NO_COLOR` variants of `TestNoColorNonTTYRegression`) reuses the cached, already-normalized result. This is sound because the captured banner is unstyled, `NO_COLOR`-invariant stderr text — no `present` renderer exists on this path yet — so the byte-identity property under test genuinely does not vary between invocations; caching does not paper over a real difference.
- **Files modified:** `internal/cli/plain_golden_test.go`
- **Verification:** Full suite (`TestPlainGolden` + `TestNoColorNonTTYRegression`, 3 environments) green and stable across 4 repeated runs.
- **Committed in:** `71494af3`

**5. [Rule 1 - Bug] Fatalf-based assertions could not surface both failure messages the plan's own acceptance criteria required**
- **Found during:** Family (a)'s first RED demonstration attempt — the planted ESC byte only produced the byte-mismatch message, never the "contains an ESC byte" message, because `t.Fatalf` on the first check aborted the subtest before the second check ran.
- **Issue:** The plan's own acceptance criteria for Task 2 require the observed transcript to show "BOTH the byte-mismatch and the ESC-present messages" for the `status` mutation.
- **Fix:** Changed both assertions (in `TestPlainGolden` and `TestNoColorNonTTYRegression`) from `t.Fatalf` to `t.Errorf`, so a case that fails both checks reports both, without otherwise changing pass/fail semantics for the subtest as a whole.
- **Files modified:** `internal/cli/plain_golden_test.go`
- **Verification:** Re-ran Family (a)'s RED demonstration; transcript now shows both `plain_golden_test.go:483: output does not match golden …` and `plain_golden_test.go:486: output contains an ESC byte …` for `TestPlainGolden/status`, and the analogous pair for both `NO_COLOR` variants.
- **Committed in:** `71494af3`

---

**Total deviations:** 5 auto-fixed (1 blocking — wrong target id; 4 bugs — 2 fixture-ordering bugs, 1 test-harness resource-lifecycle workaround, 1 assertion short-circuiting fix).
**Impact on plan:** All five were necessary for the test suite to be correct and for the plan's own acceptance criteria to be satisfiable. No scope creep — no production file was touched by any fix; every fix lives entirely inside the two new test files.

## Issues Encountered

None beyond the deviations above, all resolved during execution.

## Authentication Gates

None.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- `TestPlainGolden` and `TestNoColorNonTTYRegression` are now the live verify gate every subsequent 04-NN plan's `<verify>` runs — a renderer, resolver or palette that leaks an escape byte, reorders a line, or paraphrases wording on the plain path will be caught by these frozen files.
- `TestShortFlagsConsistent` is live and green; no short flag needs to be added in a later plan for CLI-07.
- No blockers for 04-02 (fang spike & verdict, next per the hard sequence in `04-CONTEXT.md`).

---
*Phase: 04-cli-glow-up*
*Completed: 2026-09-17*

## Self-Check: PASSED

- FOUND: internal/cli/plain_golden_test.go
- FOUND: internal/cli/short_flags_test.go
- FOUND: .planning/phases/04-cli-glow-up/04-MUTATION-LOG.md
- FOUND: internal/cli/testdata/plain/status.golden
- FOUND: internal/cli/testdata/plain/version.golden
- FOUND: internal/cli/testdata/plain/ui-url.golden
- FOUND: commit 71494af3 (git log --oneline --all)
- Re-ran plan `<verification>`: `GOTOOLCHAIN=go1.26.6 go test ./internal/cli/ -count=1 -run 'TestPlainGolden$|TestNoColorNonTTYRegression$|TestShortFlagsConsistent$' -v` — PASS, 29 golden subtests green, 20 (verb, flag) pairs inspected.
- `git status --porcelain internal/ cmd/ docs/ go.mod` — empty.
