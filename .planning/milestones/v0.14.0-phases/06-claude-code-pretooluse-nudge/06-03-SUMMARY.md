---
phase: 06-claude-code-pretooluse-nudge
plan: 03
subsystem: agents
tags: [claude-code, hooks, pretooluse, nudge, cooldown, sentinel, symlink-safety, d-16]
status: complete

requires:
  - phase: 06-claude-code-pretooluse-nudge
    provides: 06-01 hook adapter (always-fire tracer) and 06-02 Qualifies classifier
provides:
  - internal/nudge/cooldown.go — CooldownWindow, SessionKey, DefaultDir, Gate{Dir, Now}.Due, symlink-safe sentinel (checkSentinel / recordFire)
  - hook pretooluse gated on the per-(session, agent) cooldown, with a hook_event_name check and seams hookNow / hookQualifies
  - subcommand-level D-16 suite (TestHookPreToolUse_ForcedErrorContract + 5 sibling tests)
  - 06-MUTATION-LOG.md Family (c1)-(c6)
affects: [06-04, 06-06, 06-07, CODEX-05]

actuals:
  tokens: 11400
  tasks: 3
  commits: 5
plan_head_before: 30e0127954e5c6d26b107fc7bd1df444ef813f7d

tech-stack:
  added: []
  patterns:
    - "Two-layer symlink refusal: Lstat at read, O_NOFOLLOW open + fstat + fd-based Futimes at record, each proven RED separately"
    - "Unexported uid seam (currentUID) so a non-root test can simulate a foreign owner"
    - "Injected clock + planted mtimes for cooldown tests; no sleeping"

key-files:
  created:
    - internal/nudge/cooldown.go
    - internal/nudge/cooldown_test.go
  modified:
    - internal/cli/hook_pretooluse.go
    - internal/cli/hook_pretooluse_test.go
    - .planning/phases/06-claude-code-pretooluse-nudge/06-MUTATION-LOG.md

key-decisions:
  - "Phase 6 06-03: Gate.Due Lstat-checks the sentinel dir after every Mkdir, not only on EEXIST, so a dir it just created is held to the same symlink/is-dir/uid test"
  - "Phase 6 06-03: a sentinel mtime in the future counts as due and is re-recorded, so a stepped-back clock cannot silence the nudge"

patterns-established:
  - "assertHookContract(t, stdout, err) is the D-16 subcommand-level contract, applied to every hook run in the suite"

requirements-completed: []

coverage:
  - id: D1
    description: "fire on the first matched call, then at most once per CooldownWindow (60s) per (session, agent) key; subagents keyed separately"
    requirement: NUDGE-04
    verification:
      - kind: unit
        ref: "internal/nudge/cooldown_test.go#TestSessionKey, TestGate_FirstCallFiresThenCooldown, TestGate_CooldownBoundary, TestGate_SeparateKeysForMainAndSubagent; internal/cli/hook_pretooluse_test.go#TestHookPreToolUse_CooldownPerAgent"
        status: pass
  - id: D2
    description: "sentinel is symlink-safe, per-uid 0700, content-free, and every failure is silent (D-08)"
    requirement: NUDGE-04
    verification:
      - kind: unit
        ref: "internal/nudge/cooldown_test.go#TestSentinel_ReadRefusesSymlink, TestSentinel_RecordRefusesSymlink, TestGate_SymlinkedDirIsSilent, TestGate_ForeignOwnedIsSilent, TestGate_UnwritableBaseIsSilent, TestGate_DirCreated0700, TestGate_SentinelHoldsNoSessionContent, TestGate_ParallelRuns"
        status: pass
  - id: D3
    description: "hook pretooluse returns nil from Execute(), prints nothing or exactly the pinned object, and never a decision key, on every forced error path"
    requirement: NUDGE-03
    verification:
      - kind: unit
        ref: "internal/cli/hook_pretooluse_test.go#TestHookPreToolUse_ForcedErrorContract (11 subtests), _EnvSessionTakesPrecedence, _ParallelRuns, _PanicIsRecovered, _NoAncestorPreRun"
        status: pass
  - id: D4
    description: "stdin boundary: exactly maxHookStdinBytes processed, +1 silent; cooldown boundary 59s silent / 60s due / future due"
    requirement: NUDGE-05
    verification:
      - kind: unit
        ref: "internal/cli/hook_pretooluse_test.go#TestHookPreToolUse_ForcedErrorContract/{oversized_input,max_size_input_fires}; internal/nudge/cooldown_test.go#TestGate_CooldownBoundary"
        status: pass

duration: 10min
completed: 2026-09-19
---

# Phase 6 Plan 03: Per-(session, agent) cooldown and the D-16 subcommand contract Summary

**`codegraph hook pretooluse` now fires once per (session, agent) key and then at most once every 60 s. Each fire is recorded as the mtime of a zero-byte, SHA-256-named sentinel in a per-uid 0700 temp directory. The sentinel code refuses symlinks at both the read layer (Lstat) and the record layer (O_NOFOLLOW open, fstat, Futimes on the descriptor). Every forced error path is shown to leave `Execute()` nil with empty or pinned stdout.**

## Performance

- **Duration:** about 10 min
- **Started:** 2026-09-19T09:24:53Z
- **Completed:** 2026-09-19T09:35Z
- **Tasks:** 3
- **Files modified:** 5

## Accomplishments

- `internal/nudge/cooldown.go` holds the harness-neutral gate: `CooldownWindow = 60 * time.Second` (the one constant), `SessionKey` (agent id or `main`, and `ok=false` without a session), `DefaultDir` (honours TMPDIR), and `Gate.Due`. The package still imports nothing from this module (`go list -deps` shows 1 internal package, itself).
- `hook pretooluse` silences a `hook_event_name` other than PreToolUse. It classifies through the `hookQualifies` seam before any sentinel I/O, resolves the session (env first, then stdin), and emits only when `Gate.Due` records a fire. The deferred recover is still RunE's first statement and its only return is `nil`.
- There are 13 cooldown and sentinel unit tests, plus the D-16 subcommand suite: 11 ForcedErrorContract subtests and 5 sibling tests, including an ancestor PreRun walk.
- Family (c1)-(c6) is in 06-MUTATION-LOG.md. Every family went RED and reverted byte-clean, with a green control.

## Task Commits

1. **Task 1 RED:** `ea2baab9` test(06-03): add failing cooldown and sentinel-safety tests
2. **Task 1 GREEN:** `fb5dea87` feat(06-03): per-(session, agent) symlink-safe cooldown gate for the nudge
3. **Task 2 RED:** `917854fb` test(06-03): add failing subcommand-level D-16 contract tests
4. **Task 2 GREEN:** `0e5ac58d` feat(06-03): gate the PreToolUse nudge on the per-(session, agent) cooldown
5. **Task 3:** `ec9583a7` docs(06-03): record Family (c) cooldown, sentinel and contract RED demonstrations

## TDD RED evidence

**Task 1** (commit `ea2baab9`, run against the zero-value placeholders; `go test ./internal/nudge/ -count=1 -run 'TestSessionKey$|TestGate_|TestSentinel_|TestDefaultDirHonoursTMPDIR$' -v`, exit 1):

```
--- FAIL: TestSessionKey (0.00s)
--- FAIL: TestGate_FirstCallFiresThenCooldown (0.00s)
--- FAIL: TestGate_CooldownBoundary (0.00s)
    --- FAIL: TestGate_CooldownBoundary/age_59s_inside (0.00s)
    --- FAIL: TestGate_CooldownBoundary/age_60s_exactly_due (0.00s)
    --- FAIL: TestGate_CooldownBoundary/age_61s_due (0.00s)
    --- FAIL: TestGate_CooldownBoundary/future_mtime_due (0.00s)
--- FAIL: TestGate_SeparateKeysForMainAndSubagent (0.00s)
--- FAIL: TestGate_DirCreated0700 (0.00s)
--- FAIL: TestSentinel_ReadRefusesSymlink (0.00s)
--- FAIL: TestSentinel_RecordRefusesSymlink (0.00s)
--- FAIL: TestGate_SymlinkedDirIsSilent (0.00s)
--- FAIL: TestGate_ForeignOwnedIsSilent (0.00s)
--- FAIL: TestGate_UnwritableBaseIsSilent (0.00s)
--- FAIL: TestGate_ParallelRuns (0.00s)
--- FAIL: TestGate_SentinelHoldsNoSessionContent (0.00s)
--- FAIL: TestDefaultDirHonoursTMPDIR (0.00s)
FAIL	github.com/seanb4t/codegraph-go/internal/nudge	0.099s
```

**Task 2** (commit `917854fb`, run against the tracer's always-fire adapter with the seams unwired; `go test ./internal/cli/ -count=1 -run 'TestHookPreToolUse_|TestHookCmd_HiddenTwoLevel$' -v`, exit 1):

```
    hook_pretooluse_test.go:230: stdout = "{\"hookSpecificOutput\":...}\n", want ""   (x3: the three subtests below)
--- FAIL: TestHookPreToolUse_ForcedErrorContract (0.01s)
    --- FAIL: TestHookPreToolUse_ForcedErrorContract/non_pretooluse_event (0.00s)
    --- FAIL: TestHookPreToolUse_ForcedErrorContract/unwritable_sentinel_base (0.00s)
    --- FAIL: TestHookPreToolUse_ForcedErrorContract/symlinked_sentinel_dir (0.00s)
    hook_pretooluse_test.go:252: env A + stdin B again (key A is cooling down): stdout = "{...}\n", want ""
--- FAIL: TestHookPreToolUse_EnvSessionTakesPrecedence (0.00s)
    hook_pretooluse_test.go:282: main again (cooldown): stdout = "{...}\n", want ""
    hook_pretooluse_test.go:282: sub-1 again (cooldown): stdout = "{...}\n", want ""
    hook_pretooluse_test.go:282: main at T+59s (inside the window): stdout = "{...}\n", want ""
--- FAIL: TestHookPreToolUse_CooldownPerAgent (0.00s)
    hook_pretooluse_test.go:328: stdout = "{...}\n", want empty (the panic is recovered before anything is written)
--- FAIL: TestHookPreToolUse_PanicIsRecovered (0.00s)
FAIL	github.com/seanb4t/codegraph-go/internal/cli	0.518s
```

(JSON bodies elided with `{...}`; each is the full pinned fire object.) Eight subtests and ParallelRuns / NoAncestorPreRun passed at RED because the tracer already behaves correctly on those paths. ParallelRuns and NoAncestorPreRun are regression guards, and NoAncestorPreRun's walk asserts it visited at least 3 commands.

## PASS counts (GREEN)

- Task 1: 13 of 13 named PASS lines, with `-race` (`/tmp/06-03-t1.txt`). `time.Sleep` does not appear under internal/nudge.
- Task 2: 11 of 11 ForcedErrorContract subtest PASS lines, and 7 of 7 named top-level PASS lines (5 new plus the two 06-01 tracer tests), with `-race` (`/tmp/06-03-t2.txt`).

## Family (c) outcome (06-MUTATION-LOG.md)

| Family | Mutation | RED line | Revert |
|---|---|---|---|
| c1 | SessionKey keys every agent as `main` | `--- FAIL: TestSessionKey`, `--- FAIL: TestGate_SeparateKeysForMainAndSubagent` (exit 1) | byte-clean, control ok |
| c2 | `os.Stat` for `os.Lstat` in checkSentinel | `--- FAIL: TestSentinel_ReadRefusesSymlink` (exit 1) | byte-clean, control ok |
| c3 | `\|syscall.O_NOFOLLOW` dropped from recordFire | `--- FAIL: TestSentinel_RecordRefusesSymlink`: the victim's mtime was re-timed (exit 1) | byte-clean, control ok |
| c4 | RunE returns `errors.New("planted")` | all 11 `--- FAIL: TestHookPreToolUse_ForcedErrorContract/...` (exit 1) | byte-clean, control ok |
| c5 | output gains `permissionDecision: "allow"` | the 3 firing subtests FAIL (forbidden key and not pinned); the 8 silent subtests stay green (exit 1) | byte-clean, control ok |
| c6 | deferred recover removed | `panic: forced panic in the classifier [recovered, repanicked]`, test binary killed (exit 1) | byte-clean, control ok |

The Task 3 verify block re-planted (c3) on its own and saw the same FAIL line, with a non-zero exit and a clean revert.

## Verification

- `GOTOOLCHAIN=go1.26.6 go test -race ./internal/nudge/ ./internal/cli/ -count=1`: both packages ok. No data race.
- `GOTOOLCHAIN=go1.26.6 go test ./internal/cli/ -count=1` (the full package, every pre-existing test unmodified): ok, 18.3 s. The known load-sensitive flake did not appear.
- `go build ./...`, `go vet ./...`, `gofmt -l`: clean.
- **Linux path:** `GOOS=linux GOTOOLCHAIN=go1.26.6 go vet ./internal/nudge/` is clean, and a `GOOS=linux` test binary cross-compiles. The cooldown tests also passed on real Linux: the cross-compiled `nudge.test` ran in `golang:1.26.5` under Docker, non-root (uid 1000) with `--network none`, and all 13 named tests passed. No build tags are needed: `syscall.O_NOFOLLOW`, `Futimes`, `NsecToTimeval` and `Stat_t.Uid` (uint32) exist on both darwin and linux, the only release GOOS.
- All acceptance greps hold. There is exactly one `CooldownWindow = 60 * time.Second`. `O_NOFOLLOW` and `Futimes` are present, and `O_TRUNC`, `os.Chtimes` and `.Truncate(` are absent from cooldown.go. `os.Lstat` is inside checkSentinel. RunE has one `return nil` after its `defer func()`. `hookQualifies(` comes before `.Due(`. `Stderr` does not appear. No `[ci skip]`.
- `git diff --quiet HEAD -- internal/nudge/cooldown.go internal/cli/hook_pretooluse.go`: clean.

## Decisions Made

- Due Lstat-checks the directory after every Mkdir, including a successful one. The plan named the EEXIST path only. The stricter check gives one code path and costs nothing.
- Beyond the plan's behaviour line, TestGate_ForeignOwnedIsSilent/existing_sentinel calls `checkSentinel` and `recordFire` directly on the foreign-owned sentinel. Through `Due`, the directory-level uid check refuses first and would hide the file-level check.
- recordFire opens `O_WRONLY|O_CREATE|O_NOFOLLOW` (no read access is needed). It rejects a non-regular or foreign file through the open descriptor's fstat before calling Futimes.

## Deviations from Plan

None. The plan was executed as written. The Linux Docker run was extra verification, not a change.

One cosmetic item was left in place on purpose. The `isolateHookEnv` doc comment (from 06-01) still says "(once 06-03 adds it)". The plan limits edits to the 06-01 tracer tests, so the comment was not reworded.

## Known Stubs

None.

## Threat Flags

None. The only new surface is the sentinel directory under os.TempDir(), which the plan's threat model already covers as T-06-12/13/14.

## Self-Check: PASSED
