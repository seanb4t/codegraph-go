---
phase: 06-claude-code-pretooluse-nudge
plan: 01
subsystem: agents
tags: [claude-code, hooks, pretooluse, nudge, cobra, posix-sh, go-embed]
status: complete

requires:
  - phase: v0.10.0 Phase 7 / v0.14.0 Phase 5
    provides: embedded Claude skill package, writeHookEntry exact-identity hook writer, writeEmbeddedFile
provides:
  - internal/nudge harness-neutral core (Tool, Qualifies for Grep/Glob, pinned Text)
  - hidden `codegraph hook pretooluse` Claude envelope adapter (never errors, recovers panics)
  - embedded POSIX guard template .claude/hooks/pretooluse-nudge.sh (also the dogfood guard)
  - hooks.json PreToolUse fragment and claudeFragmentEventBlocks (event-generic derivation)
  - InstallOptions.PreToolNudge (Keep/On/Off) and `install --pretool-nudge`
  - guard-level D-16 exec suite, render-safety suite, Family (a) mutation evidence
affects: [06-02, 06-03, 06-04, 06-05, 06-06, 06-07, CODEX-05]

actuals:
  tokens: 15700
  tasks: 2
  commits: 4
plan_head_before: 603efc95deef27d4381f3fbc6faf3148cf56a639

tech-stack:
  added: []
  patterns:
    - "Single-token template + explicit POSIX single-quote escaping for install-time script rendering"
    - "Event-generic fragment derivation: claudeFragmentEventBlocks(event, fragmentCommand, ownCommand)"
    - "Hidden two-level cobra command covered by two bare allowlist lines"

key-files:
  created:
    - internal/nudge/classify.go
    - internal/nudge/text.go
    - internal/cli/hook_pretooluse.go
    - internal/cli/hook_pretooluse_test.go
    - internal/agents/claude_pretooluse.go
    - internal/agents/claude_pretooluse_test.go
    - .claude/hooks/pretooluse-nudge.sh
    - .planning/phases/06-claude-code-pretooluse-nudge/06-MUTATION-LOG.md
  modified:
    - claudeassets.go
    - .claude/hooks/hooks.json
    - internal/agents/types.go
    - internal/agents/claude.go
    - internal/agents/claude_skillpackage_test.go
    - internal/cli/root.go
    - internal/cli/install.go
    - internal/cli/testdata/cli-reference-allowlist.txt
    - docs/CLI-REFERENCE.md

key-decisions:
  - "Phase 6 06-01: guard binary path delivered as one single-quoted token replaced with a POSIX-quoted absolute ExecPath (not text/template), so the unrendered checked-in guard stays valid, executable sh"
  - "Phase 6 06-01: the unrendered dogfood guard alone falls back to `command -v codegraph`; rendered guards never consult PATH (D-01b)"
  - "Phase 6 06-01: two bare allowlist lines (`codegraph hook`, `codegraph hook pretooluse`) instead of D-01a's one, because both hidden commands carry cobra's --help"
  - "Phase 6 06-01: PreToolUse handler timeout 5 s; guard file name pretooluse-nudge.sh; flag --pretool-nudge"

patterns-established:
  - "Guard exec tests render the REAL embedded template with stub binaries that record a start marker, so 'no process started' is a positive assertion"

requirements-completed: []

coverage:
  - id: D1
    description: "hook pretooluse prints exactly the pinned additionalContext object on Grep/Glob with a session id, silent without one"
    requirement: NUDGE-03
    verification:
      - kind: unit
        ref: "internal/cli/hook_pretooluse_test.go#TestHookPreToolUse_GrepFiresPinnedContext, TestHookPreToolUse_NoSessionIsSilent"
        status: pass
  - id: D2
    description: "hook and hook pretooluse are hidden and groupless"
    verification:
      - kind: unit
        ref: "internal/cli/hook_pretooluse_test.go#TestHookCmd_HiddenTwoLevel"
        status: pass
  - id: D3
    description: "opt-in install writes the rendered guard and four PreToolUse blocks; default install writes neither"
    requirement: NUDGE-06
    verification:
      - kind: unit
        ref: "internal/agents/claude_pretooluse_test.go#TestClaude_Install_PreToolNudgeOn_WritesGuardAndBlocks, TestClaude_Install_DefaultWritesNoPreToolUse"
        status: pass
  - id: D4
    description: "guard exits 0 on every path, starts nothing un-indexed, passes stdin/stdout through"
    requirement: NUDGE-04
    verification:
      - kind: unit
        ref: "internal/agents/claude_pretooluse_test.go#TestPreToolUseGuard (9 subtests)"
        status: pass
  - id: D5
    description: "ExecPath rendering is POSIX-quoted and validated; dogfood copy alone falls back to PATH"
    verification:
      - kind: unit
        ref: "internal/agents/claude_pretooluse_test.go#TestRenderPreToolGuard (5), TestPreToolUseGuardSourceFallsBackToPATH (2)"
        status: pass
  - id: D6
    description: "real binary: install --pretool-nudge, piped Grep event fires once and exits 0; un-indexed prints nothing"
    requirement: NUDGE-03
    verification:
      - kind: integration
        ref: "Task 1 <verify> block (/tmp/06-01-bin install ... | sh -c guard)"
        status: pass

duration: 10min
completed: 2026-09-19
---

# Phase 6 Plan 01: PreToolUse nudge tracer Summary

**Opt-in `install --pretool-nudge` renders an embedded POSIX guard with the installing binary's quoted absolute path and registers four PreToolUse blocks; the guard's hidden `codegraph hook pretooluse` prints only the pinned additionalContext object and every path exits 0.**

## Performance

- **Duration:** about 10 min
- **Started:** 2026-09-19T09:05:15Z
- **Completed:** 2026-09-19T09:15:07Z
- **Tasks:** 2
- **Files modified:** 17 (1233 insertions, 36 deletions)

## Accomplishments

- `internal/nudge` holds the harness-neutral core and imports nothing from this module. `go list -deps` shows only itself.
- The hidden `codegraph hook pretooluse` defers a recover as its first statement, reads at most 1 MiB, maps Bash/Grep/Glob/Read onto `nudge.Tool`, keys on `CLAUDE_CODE_SESSION_ID` with stdin `session_id` as the fallback, and makes a single Write of the pinned JSON line. It never returns an error.
- `.claude/hooks/pretooluse-nudge.sh` (mode 100755, shellcheck clean) runs the D-04 directory check first. The token appears exactly once, the binary runs as a child, and the last line is `exit 0`.
- `claudeFragmentEventBlocks` is extracted from `claudeSessionStartBlocks`, which now delegates to it. Existing SessionStart tests pass unchanged.
- `install --pretool-nudge` → `PreToolNudgeOn`. `docs/CLI-REFERENCE.md` was regenerated via `task docs:cli` in the same commit, and `docs:cli:drift` exits 0.

## Task Commits

1. **Task 1 RED:** `5905039e` test(06-01): add failing PreToolUse tracer tests
2. **Task 1 GREEN:** `4a0a0b75` feat(06-01): opt-in PreToolUse nudge — hidden hook subcommand, embedded guard, install --pretool-nudge
3. **Task 2 suite:** `e8a91d61` test(06-01): guard-level D-16 exec suite and render safety for the PreToolUse guard
4. **Task 2 Family (a):** `35cbfe28` docs(06-01): record Family (a) guard and render RED demonstrations

## TDD RED evidence (Task 1, commit 5905039e)

`GOTOOLCHAIN=go1.26.6 go test ./internal/cli/ ./internal/agents/ -count=1 -run '<five tests>' -v`. Every test failed on an assertion, and none failed on a build error:

```
    hook_pretooluse_test.go:42: stdout = "", want the pinned fire line "{\"hookSpecificOutput\":...}\n"
--- FAIL: TestHookPreToolUse_GrepFiresPinnedContext (0.00s)
    --- FAIL: TestHookPreToolUse_GrepFiresPinnedContext/grep (0.00s)
    --- FAIL: TestHookPreToolUse_GrepFiresPinnedContext/glob (0.00s)
    hook_pretooluse_test.go:71: env session: stdout = "", want the pinned fire line
--- FAIL: TestHookPreToolUse_NoSessionIsSilent (0.00s)
    hook_pretooluse_test.go:87: hook.Hidden = false, want true (D-01a)
    hook_pretooluse_test.go:90: hook pretooluse.Hidden = false, want true (D-01a)
    hook_pretooluse_test.go:107: codegraph --help lists hook: "  hook        "
--- FAIL: TestHookCmd_HiddenTwoLevel (0.00s)
    claude_pretooluse_test.go:97: guard not written: stat .claude/hooks/pretooluse-nudge.sh: no such file or directory
--- FAIL: TestClaude_Install_PreToolNudgeOn_WritesGuardAndBlocks (0.01s)
    --- FAIL: TestClaude_Install_PreToolNudgeOn_WritesGuardAndBlocks/local (0.00s)
    --- FAIL: TestClaude_Install_PreToolNudgeOn_WritesGuardAndBlocks/global (0.00s)
    claude_pretooluse_test.go:148: claudePreToolGuardPath(local): not implemented
    claude_pretooluse_test.go:148: claudePreToolGuardPath(global): not implemented
--- FAIL: TestClaude_Install_DefaultWritesNoPreToolUse (0.00s)
    --- FAIL: TestClaude_Install_DefaultWritesNoPreToolUse/local (0.00s)
    --- FAIL: TestClaude_Install_DefaultWritesNoPreToolUse/global (0.00s)
FAIL	github.com/seanb4t/codegraph-go/internal/cli	0.494s
FAIL	github.com/seanb4t/codegraph-go/internal/agents	0.144s
```

`gsd-tools check tdd-red-evidence` was not used. It accepts only TAP output.

## Real-binary tracer transcripts (Task 1 verify)

The install ran under a fake `HOME` into a temp project that had `.codegraph/`:

```
Claude Code: configured
  created: .mcp.json
  created: .claude/CLAUDE.md
  created: .claude/skills/codegraph/SKILL.md
  created: .claude/hooks/session-nudge.sh
  created: .claude/settings.json
  created: .claude/hooks/pretooluse-nudge.sh
  updated: .claude/settings.json
  created: .claude/skills/codegraph/.codegraph-manifest.json
```

Fire (Grep event piped through `sh -c '${CLAUDE_PROJECT_DIR}/.claude/hooks/pretooluse-nudge.sh'`):

```
{"hookSpecificOutput":{"hookEventName":"PreToolUse","additionalContext":"This repo has a codegraph index: codegraph_explore (CLI: `codegraph explore`) returns the matching symbols' source and call paths for where-is-X and how-does-Y questions."}}
exit=0
```

Silent (`.codegraph/` removed): `exit=0` and no other output. `task docs:cli:drift` returned exit 0, meaning the file is byte-identical to a fresh regeneration. The full Task 1 verify block printed `tracer green: real binary installs, fires once, silent un-indexed`.

## Guard suite counts (Task 2 verify)

- `TestPreToolUseGuard`: 9/9 PASS (indexed_binary_ok, not_indexed, codegraph_is_file, binary_missing, binary_not_executable, binary_exits_nonzero, binary_crashes, project_dir_unset_indexed, project_dir_unset_not_indexed)
- `TestRenderPreToolGuard`: 5/5 PASS
- `TestPreToolUseGuardSourceFallsBackToPATH`: 2/2 PASS (no_codegraph_on_path first uses `exec.LookPath` to confirm no codegraph is on PATH)

## Family (a) outcome (06-MUTATION-LOG.md)

- **(a1)** Planting `exit 2` as the guard's last line exits 1 with `--- FAIL: TestPreToolUseGuard/binary_exits_nonzero (0.06s)`. The subtests indexed_binary_ok, binary_crashes and project_dir_unset_indexed also go RED, while the five early-exit subtests stay green. Before the revert the gate was dirty (1); after `git checkout --`, `git diff --quiet` returns 0. The green control passed. The Task 2 verify block ran this demonstration a second time, with the same result.
- **(a2)** With `shellSingleQuote` reduced to `"'" + s + "'"`, the run exits 1 with `--- FAIL: TestRenderPreToolGuard/quote_and_space_in_path (0.00s)`, because `sh -n` reports an unterminated quote. plain_path stays green. The revert is byte-clean and the green control passed.

## Verification

- `go build ./...`, gofmt and vet are all clean.
- `go test ./internal/nudge/ ./internal/agents/ ./internal/cli/... -count=1` passes in every package.
- A wide run of every package except `internal/daemon` and `internal/cli` also passed (exit 0). `internal/daemon` was not run: this plan does not touch it, and it may only run alone.
- The `internal/cli` load-sensitive flake did not appear in either of the two `internal/cli/...` runs.
- All Task 1 and Task 2 acceptance-criteria greps pass. That includes: go list -deps; the single module import in hook_pretooluse.go; 100755 mode; token count 1; tail line `exit 0`; no exec or errexit; shellcheck; the embed directive and accessor; both allowlist lines; `"hook":` absent from root.go; SessionStart delegation; LookPath; and no `[ci skip]`.

## Decisions Made

The key-decisions frontmatter lists the discretion choices made here: the guard name, the flag, the 5 s timeout, the `internal/nudge` placement, and the single-token render. `Install`'s opt-in step lives inline in `claude.go`, directly after the SessionStart `writeHookEntry`. If rendering fails, both writes are skipped, so the registration can never point at a guard this call did not write.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Embed-scope test expected exactly three embedded files**
- **Found during:** Task 1 GREEN
- **Issue:** `TestClaudeAssets_EmbedsNoVerificationTranscripts` (`internal/agents/claude_skillpackage_test.go`) asserts `len(walked) == 3`. The plan requires a fourth `//go:embed`, so this existing test went RED.
- **Fix:** The count became 4 and the doc comment names the new file. Its guard-the-guard logic is unchanged. This test is not a SessionStart test, so the plan's intent in "no pre-existing test edited" still holds: every SessionStart test is untouched. It is still an edit to a pre-existing test, so it is recorded here.
- **Files modified:** internal/agents/claude_skillpackage_test.go
- **Commit:** 4a0a0b75

**2. [Design choice within TDD] RED placeholders made TestClaude_Install_DefaultWritesNoPreToolUse fail honestly**
- Once GREEN is written, the default-install test is naturally green, because nothing writes anything. It carries two positive preconditions. First, it resolves the guard path through `claudePreToolGuardPath`, and the placeholder returned an error. Second, it includes a positive control: an opt-in install into the same scope must make both the guard and the key appear. Both are real assertions. The RED failure is the helper erroring, not a build error.
- With the RED placeholders, the `hook` command was visible, so `TestEveryRegisteredFlagIsAccountedFor` was also RED at commit 5905039e. GREEN fixed it with the Hidden flag and the allowlist lines.

## Advisories

- **Crash stderr:** When the binary dies on a signal, macOS `/bin/sh` (bash in posix mode) writes `line 26: <pid> Segmentation fault: 11 "$codegraph_bin" hook pretooluse` to the guard's stderr. The guard still exits 0, and this is visible in the (a1) transcript. Claude Code shows stderr for a non-zero exit, not for exit 0. The D-18 live session (06-06) should confirm that no notice appears. The planned test does not assert stderr for binary_crashes.
- **Linux dash:** The guard is checked with `shellcheck -s sh` and `sh -n`, and exec-tested under macOS `/bin/sh`. Linux dash was not available locally, so CI on Linux is where it gets exec-tested.
- **Requirements:** NUDGE-03, NUDGE-04 and NUDGE-06 are only partly delivered by this tracer. 06-07 lists all three and closes them, so none are marked complete here, and `requirements-completed` is empty.

## Known Stubs

These are deliberate tracer stubs that the plan names, each with the later plan that fills it:
- `internal/nudge/classify.go` `Qualifies`: `ToolShell` and `ToolRead` always return false until 06-02 adds D-02 and D-03.
- `internal/cli/hook_pretooluse.go` has no cooldown yet. It fires on every qualifying call that has a session id; 06-03 adds the D-05…D-08 gate.
- `PreToolNudgeKeep` and `PreToolNudgeOff` do nothing in `Install` until 06-04 gives them their sticky D-10 meaning. The CLI never maps to Off until 06-05.

## Threat Flags

None. The surfaces are the ones in the plan's threat model: T-06-01/02/03/04/06 are mitigated as planned, and T-06-07/08 are accepted.

## Self-Check: PASSED
