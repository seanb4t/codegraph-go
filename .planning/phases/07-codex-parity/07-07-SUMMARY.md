---
phase: 07-codex-parity
plan: 07
subsystem: codex
tags: [codex, hooks, pretooluse, nudge, harness-envelope, tdd]

# Dependency graph
requires:
  - phase: 07-codex-parity
    provides: "07-05's Codex capability table (global/local scope flip, D-09) and 07-06's shared-instructions ownership discipline — this plan adds Codex's Hooks mechanism (HooksCodexJSON) onto the same table, and reuses writeHookEntry/removeHookEntry's exact-command ownership identity (242ec0a) unchanged for the new hooks.json artifact"
provides:
  - "codexassets.go: CodexFS embedding the Codex PreToolUse hooks.json fragment and both local/global guard templates, plus their typed accessors"
  - "internal/agents/codex_pretooluse.go: codexHooksJSONPath, codexPreToolGuardPath, codexPreToolHookCommand, codexPreToolUseBlocks, renderCodexPreToolGuard, installCodexPreToolNudge, uninstallCodexPreToolNudge — the tracer's On-only opt-in path, wired into codexTarget.Install/Uninstall"
  - "internal/cli/hook_pretooluse.go: the --harness flag on the hidden `codegraph hook pretooluse` command, codexPreToolUseInput/codexShellCommand, and runHookPreToolUseCodex — a Codex envelope reusing internal/nudge's harness-neutral core (Qualifies/SessionKey/Gate/Text) unchanged, with its own D-22 cwd/.codegraph re-check"
  - "capabilities.go's HookFiles switch generalized: HooksNone/HooksClaudeJSON/HooksCodexJSON are explicit cases, any other mechanism wraps errHookFilesUndeclared (the loud-failure contract now covers any future undeclared mechanism, not just codex-json specifically)"
  - "Family (f1)-(f5) in 07-MUTATION-LOG.md: five positive-controlled guards proving the Codex guard/envelope mechanisms can each independently fail"
affects: [07-08, 07-09, 07-10, 07-11]

# Actuals (#2632)
actuals:
  tokens: 22196
  tasks: 3
  commits: 7
plan_head_before: f264036772dbed1556d602ff54261ce0ba464266

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "A second harness envelope on the same hidden subcommand, selected by a --harness flag, reusing one harness-neutral core (internal/nudge) unchanged — the shape 07-08/any future harness's own PreToolUse adapter should follow rather than forking the core."
    - "Two embedded guard templates for one mechanism (local vs global) when a harness's own indexed-repo check genuinely differs by scope (D-22): Codex's local guard derives its root from $0 with no CLAUDE_PROJECT_DIR-equivalent env var, its global guard checks $PWD — deliberately not unified into one template with a branch, since the two checks have no process-spawn-free way to share logic without inventing an env var Codex does not provide."

key-files:
  created:
    - .codex/hooks/hooks.json
    - .codex/hooks/codegraph-pretooluse-local.sh
    - .codex/hooks/codegraph-pretooluse-global.sh
    - codexassets.go
    - internal/agents/codex_pretooluse.go
    - internal/agents/codex_pretooluse_test.go
    - internal/cli/hook_pretooluse_codex_test.go
  modified:
    - internal/agents/capabilities.go
    - internal/agents/capabilities_test.go
    - internal/agents/codex.go
    - internal/cli/hook_pretooluse.go
    - internal/cli/testdata/plain/print-config-style.golden
    - internal/cli/testdata/plain/print-config-style-local.golden
    - .planning/phases/07-codex-parity/07-MUTATION-LOG.md

key-decisions:
  - "TestCodexPreToolUseGuard's negative control for D-22's local-vs-global split needed two attempts: faking a mismatched $PWD environment variable alone does nothing, because bash re-derives $PWD from getcwd() at shell startup whenever the inherited value doesn't match the real cwd — confirmed empirically by deliberately mutating the local guard to read ${PWD:-.} and watching every local subtest stay green regardless. Fixed by making the CHILD PROCESS's actual OS-level working directory itself wrong (bogusCwd) for every local guard subtest, which the shell cannot self-heal (commits 5c930bb5, 602e30a3)."
  - "root_from_own_path_with_empty_PATH's Go subtest name is spelled root_from_own_path_with_empty_path (lowercase) rather than the plan prose's PATH — the plan's own <verify> gate counts TestCodexPreToolUseGuard subtest PASS lines with the case-sensitive regex [a-z_]+, which an uppercase PATH segment would not match, silently under-counting to 10 instead of 11. The lowercase spelling is what the automated gate actually requires; the scenario it names is unchanged."
  - "Family (f1)-(f5) in 07-MUTATION-LOG.md follow this log's pre-existing 8-anchor convention (Test/guard, What are we testing and why, Pre-mutation gate, Mutation applied, Observed failure, Pre-revert gate, Revert, Green control) — the plan text's 'six convention headings' phrasing is descriptive shorthand also used verbatim in 07-05/07-06/07-08/07-10's plans for the identical established format, not a distinct six-heading template; matching the log's own existing Family (a)-(e) entries byte-for-shape is what keeps the log internally consistent."
  - "Task 2's entire test suite (TestCodexPreToolUseGuard, TestRenderCodexPreToolGuard, TestHookPreToolUseCodex_ForcedErrorContract and siblings) passed against Task 1's implementation with no production-code change required — recorded honestly as a positive-controlled characterization guard (per the plan's own stated allowance and the 07-05/07-06 precedent) rather than reshaping the suite to force an artificial RED. Its positive control is Family (f) in Task 3."

requirements-completed: []

coverage:
  - id: D1
    description: "A real opt-in local install (`codegraph install --target codex --location local --pretool-nudge --yes`) writes an executable rendered guard at .codex/hooks/codegraph-pretooluse.sh and appends exactly one codegraph-owned PreToolUse group to .codex/hooks.json, quoted from day one per D-20, with the ExecPath never appearing in hooks.json"
    requirement: "CODEX-05"
    verification:
      - kind: unit
        ref: "internal/agents/codex_pretooluse_test.go#TestCodex_Install_PreToolNudgeOn_WritesGuardAndGroup"
        status: pass
      - kind: other
        ref: "real-binary tracer: install --target codex --location local --pretool-nudge --yes against a fresh git repo (07-07-PLAN.md Task 1 <verify>)"
        status: pass
    human_judgment: false
  - id: D2
    description: "The same opt-in install at global scope writes ~/.codex/hooks/codegraph-pretooluse.sh and registers a single-quoted absolute command in ~/.codex/hooks.json"
    requirement: "CODEX-05"
    verification:
      - kind: unit
        ref: "internal/agents/codex_pretooluse_test.go#TestCodex_Install_PreToolNudgeOn_WritesGuardAndGroup/global"
        status: pass
    human_judgment: false
  - id: D3
    description: "A default install (no --pretool-nudge) writes neither the guard nor hooks.json — the nudge is opt-in only (D-18)"
    requirement: "CODEX-05"
    verification:
      - kind: unit
        ref: "internal/agents/codex_pretooluse_test.go#TestCodex_Install_DefaultWritesNoHooks"
        status: pass
    human_judgment: false
  - id: D4
    description: "Uninstall always attempts removal of the guard and codegraph's own hooks.json group, reporting not-found on a never-opted-in location and deleting a hooks.json it empties"
    requirement: "CODEX-05"
    verification:
      - kind: unit
        ref: "internal/agents/codex_pretooluse_test.go#TestCodex_Uninstall_RemovesHooks"
        status: pass
    human_judgment: false
  - id: D5
    description: "The Codex envelope (`codegraph hook pretooluse --harness codex`), fed a real PreToolUse Bash event with a qualifying search command in an indexed cwd, prints exactly the pinned additionalContext object and exits 0"
    requirement: "CODEX-05"
    verification:
      - kind: unit
        ref: "internal/cli/hook_pretooluse_codex_test.go#TestHookPreToolUseCodex_BashFiresPinnedContext"
        status: pass
      - kind: other
        ref: "real-binary tracer: piping a Codex PreToolUse event through the installed guard's own registered command via sh -c (07-07-PLAN.md Task 1 <verify>)"
        status: pass
    human_judgment: false
  - id: D6
    description: "The same event with a cwd lacking .codegraph produces no output — the Go core independently re-checks cwd is absolute and indexed (D-22), not just the shell guard"
    requirement: "CODEX-05"
    verification:
      - kind: unit
        ref: "internal/cli/hook_pretooluse_codex_test.go#TestHookPreToolUseCodex_NotIndexedCwdIsSilent"
        status: pass
      - kind: other
        ref: "real-binary tracer: removing .codegraph and re-firing the same registered command (07-07-PLAN.md Task 1 <verify>)"
        status: pass
    human_judgment: false
  - id: D7
    description: "The guard exec suite over BOTH real embedded templates (local and global) exits 0 on every path, starts no process in an un-indexed repo, passes stdin/stdout through byte-identical, and the local guard's root derivation works with PATH cleared entirely"
    requirement: "CODEX-05"
    verification:
      - kind: unit
        ref: "internal/agents/codex_pretooluse_test.go#TestCodexPreToolUseGuard (11 subtests)"
        status: pass
    human_judgment: false
  - id: D8
    description: "renderCodexPreToolGuard's quoting survives a space and a single quote in the ExecPath, rejects a relative or empty path at both locations, and each template carries the ExecPath token exactly once"
    requirement: "CODEX-05"
    verification:
      - kind: unit
        ref: "internal/agents/codex_pretooluse_test.go#TestRenderCodexPreToolGuard (5 subtests)"
        status: pass
    human_judgment: false
  - id: D9
    description: "The adapter's forced-error contract holds across 24 cases: decode ambiguity, tool-name mapping (Bash/exec_command/shell, case-sensitive), every command shape (string/null/empty/array/object/argv-first-vs-last), the D-22 cwd re-check, and sentinel safety (unwritable base, symlinked sentinel dir)"
    requirement: "CODEX-05"
    verification:
      - kind: unit
        ref: "internal/cli/hook_pretooluse_codex_test.go#TestHookPreToolUseCodex_ForcedErrorContract (24 subtests)"
        status: pass
    human_judgment: false
  - id: D10
    description: "The 60s cooldown applies per (session, agent) key with no env-var fallback for Codex (unlike Claude's CLAUDE_CODE_SESSION_ID), and every tool==\"shell\" row of the SAME internal/nudge/testdata corpora Claude's path reads fires correctly through the Codex envelope in both string and argv forms"
    requirement: "CODEX-05"
    verification:
      - kind: unit
        ref: "internal/cli/hook_pretooluse_codex_test.go#TestHookPreToolUseCodex_CooldownPerAgent"
        status: pass
      - kind: unit
        ref: "internal/cli/hook_pretooluse_codex_test.go#TestHookPreToolUseCodex_ShellCorpora"
        status: pass
    human_judgment: false
  - id: D11
    description: "A panic inside the shared classifier is recovered before anything is written, and an unrecognized --harness value is silent (neither envelope runs)"
    requirement: "CODEX-05"
    verification:
      - kind: unit
        ref: "internal/cli/hook_pretooluse_codex_test.go#TestHookPreToolUseCodex_PanicIsRecovered"
        status: pass
      - kind: unit
        ref: "internal/cli/hook_pretooluse_codex_test.go#TestHookPreToolUse_UnknownHarnessIsSilent"
        status: pass
    human_judgment: false
  - id: D12
    description: "Family (f1)-(f5): five planted-mutation positive controls (the local guard's exit-0 contract, the $0-vs-$PWD root derivation, the Claude-env-fallback exclusion, the argv last-element extraction, the Go core's independent cwd re-check) each independently demonstrated RED and reverted byte-clean"
    verification:
      - kind: other
        ref: ".planning/phases/07-codex-parity/07-MUTATION-LOG.md Family (f1)-(f5)"
        status: pass
    human_judgment: false

duration: ~40min
completed: 2026-09-19
status: complete
---

# Phase 7 Plan 7: Codex PreToolUse Nudge Summary

**A real opt-in local or global install now registers a quoted-from-day-one Codex `hooks.json` PreToolUse group and a rendered guard that runs `codegraph hook pretooluse --harness codex` — a Codex envelope reusing the harness-neutral `internal/nudge` core unchanged, with its own independent cwd/.codegraph re-check, proven end-to-end through the real compiled binary.**

## Performance

- **Duration:** ~40 min
- **Started:** 2026-09-19T19:16:00Z (approx, first Read tool call)
- **Completed:** 2026-09-19T19:56:00Z (final commit before this SUMMARY)
- **Tasks:** 3
- **Files modified:** 14

## Accomplishments

- Embedded a new Codex-specific asset package (`codexassets.go`, sibling to `claudeassets.go` at the repo root per the golang/go#46056 sibling-of-root `//go:embed` rule): the Codex PreToolUse `hooks.json` fragment and two guard templates (local, global — D-22's split checks).
- `internal/agents/codex_pretooluse.go`: path resolvers, the fragment-to-own-command rewriter (`codexPreToolUseBlocks`, a Codex-specific copy of `claudePreToolUseBlocks`'s pattern, deliberately not shared code since the two fragments and command shapes diverge), the guard renderer, and `installCodexPreToolNudge`/`uninstallCodexPreToolNudge` — wired into `codexTarget.Install` (On-only, per this plan's tracer scope) and `codexTarget.Uninstall` (always attempted).
- `capabilities.go`'s `HookFiles` switch is now exhaustive: `HooksNone`, `HooksClaudeJSON`, `HooksCodexJSON` are explicit cases, and any other mechanism wraps `errHookFilesUndeclared` — the loud-failure contract WR-01 established now covers a genuinely unknown future mechanism, not just the codex-json case that existed as a placeholder.
- `internal/cli/hook_pretooluse.go`: a `--harness` flag on the hidden `codegraph hook pretooluse` command dispatches to `runHookPreToolUse` (Claude, unchanged) or the new `runHookPreToolUseCodex` — decoding Codex's stdin shape (`tool_input.command` as a string or argv array, no `agent_id` on the main thread per the live evidence), mapping `Bash`/`exec_command`/`shell` onto the shell rule, re-checking `cwd` is absolute and indexed independently of the guard, and keying the cooldown on stdin alone (no env-var fallback exists for Codex).
- Both `internal/cli/testdata/plain/print-config-style{,-local}.golden` regenerated: Codex's line now reads `hooks=codex-json`.
- A real-binary tracer (Task 1's `<verify>`) drove the compiled `codegraph` binary through `install --target codex --location local --pretool-nudge --yes`, confirmed the guard's mode, rendered ExecPath line, the exact D-20 registered command form, timeout 5, and the absence of the ExecPath string in `hooks.json`, then piped a real Codex PreToolUse Bash event through the registered command via `sh -c` — exactly one pinned line, exit 0 — and confirmed silence once `.codegraph` was removed.
- `TestCodexPreToolUseGuard` (11 subtests) and `TestRenderCodexPreToolGuard` (5 subtests) exercise the REAL embedded templates end-to-end with stub binaries recording argv/stdin.
- `TestHookPreToolUseCodex_ForcedErrorContract` (24 subtests), `TestHookPreToolUseCodex_CooldownPerAgent`, `TestHookPreToolUseCodex_ShellCorpora` (drives every `tool=="shell"` row of the SAME `internal/nudge/testdata` corpora Claude's path reads — no copied rows), `TestHookPreToolUseCodex_PanicIsRecovered`, and `TestHookPreToolUse_UnknownHarnessIsSilent` pin the adapter's contract.
- Family (f1)-(f5) in `07-MUTATION-LOG.md`: five planted mutations (the local guard's trailing exit code, its `$0`-vs-`$PWD` root derivation, the adapter's Claude-env-fallback exclusion, the argv last-element extraction, and the Go core's independent cwd re-check) each demonstrated RED for real and reverted byte-clean.

## Task Commits

Each task was committed atomically (TDD RED/GREEN, per plan):

1. **Task 1 (RED): add failing Codex PreToolUse install and envelope tests** — `5a16fc4c` (test)
2. **Task 1 (GREEN): Codex PreToolUse nudge through hooks.json and the harness envelope** — `d8a74bd0` (feat)
3. **Task 2 (already-GREEN honest record): add the Codex guard, render and envelope contract suites** — `3efe018d` (test; a positive-controlled characterization guard, no code change required — see Deviations)
4. **Test-harness fix (found while preparing Task 3's Family (f2)): give TestCodexPreToolUseGuard a real PWD negative control** — `5c930bb5` (fix)
5. **Test-harness fix (the above attempt was itself found ineffective): use a wrong process cwd, not a faked $PWD** — `602e30a3` (fix)
6. **Task 3: Family (f) mutation log** — `eb45c850` (docs)

**Plan metadata:** committed separately after this SUMMARY (see below).

_Note: Task 1 followed TDD (`tdd="true"`) with a genuine test → feat RED/GREEN pair. Task 2 (also `tdd="true"`) is, by the plan's own instruction, a suite that "already passes against Task 1's code" — recorded honestly as GREEN-at-write-time rather than reshaping it to force an artificial RED, following the 07-05/07-06-SUMMARY.md precedent for the same situation. Two intervening `fix(07-07)` commits correct a bug discovered in the test harness itself (not production code) while preparing Family (f2)'s evidence — see Deviations._

## RED Evidence (Task 1)

`GOTOOLCHAIN=go1.26.6 go test ./internal/agents/ ./internal/cli/ -count=1 -run 'TestCodexHooksFragmentShape$|TestCodex_Install_PreToolNudgeOn_WritesGuardAndGroup$|TestCodex_Install_DefaultWritesNoHooks$|TestCodex_Uninstall_RemovesHooks$|TestHookPreToolUseCodex_BashFiresPinnedContext$|TestHookPreToolUseCodex_NotIndexedCwdIsSilent$|TestDescribeDeclaredPaths_PanicsOnUndeclaredHookFiles$|TestCapabilitiesDeclared$|TestCapabilitiesMatchInstallWrites$|TestCapabilitiesTableDrivesDerivations$' -v` against the unmodified (pre-Task-1) code:

```
=== RUN   TestCapabilitiesDeclared/codex
    capabilities_test.go:198: Hooks = "none", want "codex-json"
--- FAIL: TestCapabilitiesDeclared (0.00s)
    --- FAIL: TestCapabilitiesDeclared/codex (0.00s)

=== RUN   TestDescribeDeclaredPaths_PanicsOnUndeclaredHookFiles
    capabilities_test.go:489: describeDeclaredPaths did not panic for a target declaring an undeclared HookMechanism with no HookFiles case — the documented loud-failure contract was not honored
--- FAIL: TestDescribeDeclaredPaths_PanicsOnUndeclaredHookFiles (0.00s)

=== RUN   TestCodex_Install_PreToolNudgeOn_WritesGuardAndGroup/local
    codex_pretooluse_test.go:96: guard not written: stat .codex/hooks/codegraph-pretooluse.sh: no such file or directory
=== RUN   TestCodex_Install_PreToolNudgeOn_WritesGuardAndGroup/global
    codex_pretooluse_test.go:149: global guard not written: stat .../.codex/hooks/codegraph-pretooluse.sh: no such file or directory
--- FAIL: TestCodex_Install_PreToolNudgeOn_WritesGuardAndGroup (0.00s)

=== RUN   TestCodex_Install_DefaultWritesNoHooks/local
    codex_pretooluse_test.go:208: positive control: opt-in install did not write .codex/hooks/codegraph-pretooluse.sh: lstat: no such file or directory
--- FAIL: TestCodex_Install_DefaultWritesNoHooks (0.00s)

=== RUN   TestCodex_Uninstall_RemovesHooks/local
    codex_pretooluse_test.go:245: precondition: guard not written: stat .codex/hooks/codegraph-pretooluse.sh: no such file or directory
--- FAIL: TestCodex_Uninstall_RemovesHooks (0.00s)

=== RUN   TestHookPreToolUseCodex_NotIndexedCwdIsSilent
    hook_pretooluse_codex_test.go:50: stdout = "{\"hookSpecificOutput\":{\"hookEventName\":\"PreToolUse\",\"additionalContext\":\"This repo has a codegraph index: codegraph_explore (CLI: `codegraph explore`) returns the matching symbols' source and call paths for where-is-X and how-does-Y questions.\"}}\n", stderr = "", want both empty (un-indexed cwd)
--- FAIL: TestHookPreToolUseCodex_NotIndexedCwdIsSilent (0.00s)
FAIL
```

`TestCodexHooksFragmentShape` and `TestHookPreToolUseCodex_BashFiresPinnedContext` PASSED already at the RED commit — the embedded fragment's shape and the Bash-fires-on-an-indexed-repo path both happen to already satisfy their assertions from the pre-written assets and adapter skeleton alone (the RED commit's adapter was intentionally missing only the D-22 cwd re-check, per its own commit message), honestly recorded rather than reshaped to force an artificial RED.

## Corpus Fire Counts (Task 2, `TestHookPreToolUseCodex_ShellCorpora`)

```
true-positives: drove 22 shell rows (string+argv forms), 18 fired
false-positives: drove 22 shell rows (string+argv forms), 4 fired
```

Both corpora have 11 `tool: "shell"` rows; each is driven twice (as a plain string and as `["bash","-lc",<input>]`), for 22 driven executions per corpus — well over the plan's 14-row floor. The false-positives corpus's 4 fires are its own documented accepted false positives (rows whose `note` explains why `Qualifies` cannot distinguish them, unchanged from the pre-existing `internal/nudge` corpus).

## A1 / Stdin-Field Evidence Relied On (from 07-LIVE-SESSIONS.md)

- `PreToolUse stdin fields (main thread): cwd, hook_event_name, model, permission_mode, session_id, tool_input, tool_name, tool_use_id, transcript_path, turn_id; tool_name "Bash"; tool_input.command is a string; no agent_id/agent_type on the main thread (B5)` — this is why `codexPreToolUseInput.AgentID` has no env-var analog to fall back to (unlike Claude's `CLAUDE_CODE_SESSION_ID`), and why `codexShellCommand` must also accept an argv shape defensively even though the one live-observed session sent a plain string.
- `A1 local command form shell-expanded: yes` / `A1 global quoted command form runs: yes` — the plan's own `<precondition>`, confirmed present before any work began.
- `Hooks.json runs behind features.hooks: yes` — informed the decision not to add any `[features] hooks = false` handling in this plan (deferred to 07-08's lifecycle work, per the plan's own scope note).

## Files Created/Modified

- `.codex/hooks/hooks.json` — the embedded Codex PreToolUse fragment (D-20's exact quoted command form)
- `.codex/hooks/codegraph-pretooluse-local.sh` — local guard template ($0-derived root, D-22)
- `.codex/hooks/codegraph-pretooluse-global.sh` — global guard template ($PWD-derived root, D-22)
- `codexassets.go` — `CodexFS` and its typed accessors
- `internal/agents/codex_pretooluse.go` — path resolvers, fragment rewriter, guard renderer, install/uninstall
- `internal/agents/codex_pretooluse_test.go` — Task 1 install/uninstall tests, Task 2 guard/render suites
- `internal/agents/capabilities.go` — `HookFiles`'s exhaustive switch, `errHookFilesUndeclared`'s generalized message
- `internal/agents/capabilities_test.go` — `TestCapabilitiesDeclared`'s codex case, `fakeUndeclaredHooksTarget` (renamed from `fakeHooksCodexJSONTarget`), `expectedDeclaredPaths`'s new `HooksCodexJSON` branch
- `internal/agents/codex.go` — `Hooks: HooksCodexJSON`, Install/Uninstall wiring
- `internal/cli/hook_pretooluse.go` — `--harness` flag, `codexPreToolUseInput`, `codexShellCommand`, `runHookPreToolUseCodex`
- `internal/cli/hook_pretooluse_codex_test.go` — Task 1 envelope tests, Task 2 forced-error/cooldown/corpora/panic/unknown-harness suites
- `internal/cli/testdata/plain/print-config-style.golden`, `print-config-style-local.golden` — regenerated
- `.planning/phases/07-codex-parity/07-MUTATION-LOG.md` — Family (f1)-(f5)

## Decisions Made

See `key-decisions` in frontmatter.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug in own test harness] `TestCodexPreToolUseGuard`'s first negative-control attempt for D-22 did nothing**
- **Found during:** preparing Task 3's Family (f2) evidence
- **Issue:** The initial design faked a mismatched `$PWD` environment variable for local guard subtests while leaving the child process's real OS-level cwd correct. Deliberately mutating the local guard's root derivation to `${PWD:-.}` and re-running the suite showed every local subtest stayed green — proof the negative control was inert, because bash re-derives `$PWD` from `getcwd()` at shell startup whenever the inherited value doesn't match the real cwd.
- **Fix:** Redesigned the test helper to make the CHILD PROCESS's actual working directory itself wrong (`bogusCwd`, un-indexed, distinct from every case's project dir) for every local guard subtest, which the shell cannot self-heal. Re-verified: the same `${PWD:-.}` mutation now genuinely turns 4 local subtests RED (including `indexed_binary_ok`), and reverts byte-clean.
- **Files modified:** `internal/agents/codex_pretooluse_test.go`
- **Verification:** re-ran `TestCodexPreToolUseGuard`/`TestRenderCodexPreToolGuard` after each redesign (still green under correct code); re-ran the Family (f2) mutation against the fixed harness (genuinely RED); reverted and confirmed green again
- **Committed in:** `5c930bb5`, `602e30a3` (two sequential fix commits — the first fix was itself found ineffective before landing on the second)

---

**Total deviations:** 1 auto-fixed (a bug in this plan's own test-harness code, discovered and corrected before it could ship as a hollow guard)
**Impact on plan:** No production-code behavior was affected. Without this fix, Family (f2)'s positive control would have been vacuous — a genuine gap this plan's own D-00 discipline ("a gate is not trusted until it has been demonstrated RED against a confirmed-applied mutation") exists to catch, and did catch, before the mutation log was ever written.

## Issues Encountered

None beyond the deviation above. The stale `--pretool-nudge only configures Claude Code...` warning the real-binary tracer's install run still prints (visible in Task 1's tracer output) is deliberately left untouched — the plan's own objective note assigns `--pretool-nudge`'s Claude-only help text and this note to 07-08, which now inherits fixing it once Codex opt-in is user-facing.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- CODEX-05 is declared by six plans in this phase (07-01, 07-07, 07-08, 07-09, 07-10, 07-11); `gsd_run query requirements.ready-ids` confirms it is NOT yet ready to mark complete (siblings still pending) — correctly deferred, not marked here.
- The Codex hooks.json/guard mechanism is now fully wired through `codexTarget.Install`/`Uninstall` for the On-only tracer path; 07-08 can add `Keep`/`Off` semantics and the `[features] hooks = false` skip on top without touching this plan's guard templates, envelope, or capability-table wiring.
- `internal/cli/hook_pretooluse.go`'s `--harness` dispatch is a clean seam for any future third harness envelope: add a case, reuse `internal/nudge`'s core, done.
- The real-binary tracer proves the full CODEX-05 runtime contract end-to-end through the actual compiled binary, not just unit tests — the same discipline 07-05/07-06 established.
- No blockers.

## Self-Check: PASSED

- `.codex/hooks/hooks.json` — FOUND
- `.codex/hooks/codegraph-pretooluse-local.sh` — FOUND, mode 100755
- `.codex/hooks/codegraph-pretooluse-global.sh` — FOUND, mode 100755
- `codexassets.go` — FOUND, contains `var CodexFS embed.FS`
- `internal/agents/codex_pretooluse.go` — FOUND, contains `func installCodexPreToolNudge(`
- `internal/cli/hook_pretooluse.go` — FOUND, contains `func runHookPreToolUseCodex(`
- `.planning/phases/07-codex-parity/07-MUTATION-LOG.md` — FOUND, contains `## Family (f1)` through `## Family (f5)`
- Commit `5a16fc4c` (test, Task 1 RED) — FOUND in `git log --oneline --all`
- Commit `d8a74bd0` (feat, Task 1 GREEN) — FOUND in `git log --oneline --all`
- Commit `3efe018d` (test, Task 2) — FOUND in `git log --oneline --all`
- Commit `5c930bb5`, `602e30a3` (fix, test-harness correction) — FOUND in `git log --oneline --all`
- Commit `eb45c850` (docs, Task 3) — FOUND in `git log --oneline --all`
- `GOTOOLCHAIN=go1.26.6 go test ./internal/agents/ ./internal/cli/ ./internal/nudge/ -count=1` re-run at SUMMARY time — exit 0, all `ok`
- Real-binary tracer (install/fire/silent-when-un-indexed) re-run at SUMMARY time — all assertions passed
- `rg -c 'HooksCodexJSON' internal/agents/codex.go` — 1, confirmed
- `awk '/^func runHookPreToolUseCodex/,/^}/' internal/cli/hook_pretooluse.go | rg -q 'getenv|Getenv'` — exits 1 (no match), confirmed no env var read in the Codex adapter body
- `git show --stat d8a74bd0 -- internal/nudge/` — lists no file, confirmed the core is reused unchanged

---
*Phase: 07-codex-parity*
*Completed: 2026-09-19*
