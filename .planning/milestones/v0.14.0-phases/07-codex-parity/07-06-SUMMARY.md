---
phase: 07-codex-parity
plan: 06
subsystem: codex
tags: [codex, opencode, agents-md, ownership, tdd]

# Dependency graph
requires:
  - phase: 07-codex-parity
    provides: "07-05's Codex local scope through the capability table — codex and opencode already share the identical repo-root AGENTS.md path and marker contract this plan builds the shared-uninstall rule on top of"
provides:
  - "instructionsRequestedElsewhere(path, loc, self) — a shared.go helper, derived from AllTargets() on every call, reporting which OTHER registered targets still declare an instructions path at loc and report Detect(loc).AlreadyConfigured (D-11)"
  - "codexTarget.Uninstall and opencodeTarget.Uninstall both gate removeMarkedSection on that helper: the shared repo-root AGENTS.md's codegraph block is kept (ActionKept + a Note naming the sibling) while another registered target still needs it, and only stripped once no sharer remains"
  - "codexTarget.Install adds a D-12 Note when an AGENTS.override.md shadows the instructions file Codex reads; the block is still written to AGENTS.md and the override file is never touched"
  - "TestOwnershipSharedInstructions extends the D-13/242ec0a exact-identity ownership guard to the shared AGENTS.md across all three uninstall orders"
  - "Family (e1)-(e3) in 07-MUTATION-LOG.md: three positive-controlled guards proving the D-11/D-12 mechanisms can each independently fail"
affects: [07-09, 07-10, 07-11]

# Actuals (#2632)
actuals:
  tokens: 9091
  tasks: 3
  commits: 4
plan_head_before: 316e5ad5879083283bb7a4d5426c09f66403b403

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "A shared, marker-fenced instructions file (repo-root AGENTS.md) now has a DERIVED ownership model instead of a single-owner one: instructionsRequestedElsewhere queries the live registry on every Uninstall call rather than storing a requester index anywhere on disk — the same 'derive, never hand-roll' discipline capabilities.go's declaredSkillFallback already established for the shared skill package, generalized from one fixed comparison target (Claude) to the full registry."

key-files:
  created: []
  modified:
    - internal/agents/shared.go
    - internal/agents/codex.go
    - internal/agents/opencode.go
    - internal/agents/codex_test.go
    - internal/agents/shared_test.go
    - internal/agents/ownership_test.go
    - .planning/phases/07-codex-parity/07-MUTATION-LOG.md

key-decisions:
  - "instructionsRequestedElsewhere compares filepath.Abs(filepath.Clean(...)) of both the target file and each candidate target's own InstructionsPath(loc) — this is what makes a relative local-scope \"AGENTS.md\" compare correctly against another relative \"AGENTS.md\" resolved from the same cwd, and what makes global scope (different absolute paths for codex vs opencode) a correct no-op with no special-casing."
  - "Family (e2)'s mutation (opencode.go's own gate call site disabled) does NOT turn TestOwnershipSharedInstructions/opencode_then_codex RED, contrary to the plan's own prediction — recorded as a scope observation in 07-MUTATION-LOG.md rather than forcing an artificial red. TestOwnershipSharedInstructions asserts END-of-sequence state only (per its own Task 2 behavior spec); the defect's effect (opencode stripping the block prematurely on its own first-in-order uninstall) converges away by the time codex's still-correct gate runs second and finds nothing left to touch. TestSharedAgentsMD_UninstallOrders's per-step assertion is the guard that demonstrably catches this exact regression (RED on both preexisting and absent subtests), which is sufficient positive control for the D-11 mechanism as specified."
  - "Fixed a bug in my own RED-commit test assertions (not a plan or production bug): the first draft of assertAgentsMDKept compared a WriteResult's FileResult.Path (which is the relative literal \"AGENTS.md\" caps.InstructionsPath(LocationLocal) returns) against an absolute joined path — corrected before the GREEN commit."

requirements-completed: [CODEX-04]

coverage:
  - id: D1
    description: "instructionsRequestedElsewhere derives, from AllTargets() on every call, which other registered targets still declare an instructions path at a location and report themselves AlreadyConfigured there — never a stored index"
    requirement: "CODEX-04"
    verification:
      - kind: unit
        ref: "internal/agents/shared_test.go#TestSharedAgentsMD_GlobalPathsNotShared"
        status: pass
      - kind: unit
        ref: "internal/agents/shared_test.go#TestSharedAgentsMD_KeptWhileOtherConfigured"
        status: pass
    human_judgment: false
  - id: D2
    description: "Every uninstall order (codex-then-opencode, opencode-then-codex, --target all) leaves repo-root AGENTS.md byte-identical to its pre-install bytes once the last sharer is gone, whether AGENTS.md pre-existed with foreign content or did not exist"
    requirement: "CODEX-04"
    verification:
      - kind: unit
        ref: "internal/agents/shared_test.go#TestSharedAgentsMD_UninstallOrders"
        status: pass
      - kind: unit
        ref: "internal/agents/ownership_test.go#TestOwnershipSharedInstructions"
        status: pass
      - kind: other
        ref: "real-binary tracer: install --target codex,opencode / uninstall codex / uninstall opencode / reinstall / uninstall --target all, each cmp against the original AGENTS.md bytes (07-06-PLAN.md Task 1 <verify>)"
        status: pass
    human_judgment: false
  - id: D3
    description: "An AGENTS.override.md beside the instructions file Codex reads gets exactly one Note that it shadows AGENTS.md for Codex; the block is still written to AGENTS.md and the override file is never created, written, or modified"
    requirement: "CODEX-04"
    verification:
      - kind: unit
        ref: "internal/agents/codex_test.go#TestCodex_Install_OverrideNote"
        status: pass
      - kind: other
        ref: "real-binary tracer: AGENTS.override.md planted, install --target codex, note grep + cmp against original override bytes (07-06-PLAN.md Task 1 <verify>)"
        status: pass
    human_judgment: false
  - id: D4
    description: "The D-13 ownership table covers the shared instructions file: a foreign MCP entry, a foreign AGENTS.md section, a foreign sibling skill, and (for the foreign-codegraph-dir variant) a manifest-less skill root all survive byte-identical across every uninstall order, with every codegraph entry/table/block/manifest gone"
    requirement: "CODEX-04"
    verification:
      - kind: unit
        ref: "internal/agents/ownership_test.go#TestOwnershipSharedInstructions"
        status: pass
      - kind: unit
        ref: "internal/agents/ownership_test.go#TestOwnershipExactIdentity (32-leaf count unchanged)"
        status: pass
    human_judgment: false
  - id: D5
    description: "Family (e1)-(e3): three planted-mutation positive controls (the shared helper stubbed to nil, opencode's own gate call site disabled, the D-12 Note append dropped) each independently demonstrated RED and reverted byte-clean"
    verification:
      - kind: other
        ref: ".planning/phases/07-codex-parity/07-MUTATION-LOG.md Family (e1)-(e3)"
        status: pass
    human_judgment: false

duration: ~25min
completed: 2026-09-19
status: complete
---

# Phase 7 Plan 6: Shared Repo-Root AGENTS.md Summary

**The repo-root `AGENTS.md` codex and opencode both write at local scope now has a derived-ownership uninstall rule: `instructionsRequestedElsewhere` queries the live target registry on every call so the marker block survives while a sibling agent still needs it and is restored byte-for-byte, in every uninstall order, once the last one leaves — plus a D-12 Note when a user's `AGENTS.override.md` shadows it for Codex.**

## Performance

- **Duration:** ~25 min
- **Started:** 2026-09-19T18:30:00Z (approx, first Read tool call)
- **Completed:** 2026-09-19T18:55:00Z (approx, final commit before this SUMMARY)
- **Tasks:** 3
- **Files modified:** 7

## Accomplishments

- Added `instructionsRequestedElsewhere(path, loc, self) []TargetID` to `internal/agents/shared.go`: iterates `AllTargets()` on every call (never a stored index), skips `self` and targets that don't support `loc`, resolves each candidate's own `Capabilities().InstructionsPath(loc)`, compares `filepath.Abs(filepath.Clean(...))` of both sides, and returns the ids whose `Detect(loc).AlreadyConfigured` is true.
- Wired the gate into both `codexTarget.Uninstall` and `opencodeTarget.Uninstall`, immediately before the existing `removeMarkedSection` call: when the helper returns a non-empty requester set, the file is left alone (`ActionKept`) with a Note (`instructionsKeptNote`) naming the sibling(s) still using it; otherwise the block is stripped exactly as before.
- Added the D-12 advisory to `codexTarget.Install`: when `AGENTS.override.md` sits beside the instructions file Codex reads, a Note is appended that the override shadows `AGENTS.md` for Codex — the block is still written to `AGENTS.md`, and the override file is only ever `fileExists`-checked, never opened for writing.
- `TestSharedAgentsMD_UninstallOrders` (6 leaves: 3 orders × {preexisting, absent}) proves every uninstall order restores `AGENTS.md` byte-for-byte once the last sharer is gone, and that a two-step order's FIRST uninstall reports the file `kept` while the block is still present.
- `TestOwnershipSharedInstructions` (6 leaves: 3 orders × {clean, foreign-codegraph-dir}) extends the D-13/`242ec0a` exact-identity ownership discipline to the shared file — it passes against this plan's own implementation today (a guard, not a novel assertion), and the 32-leaf `TestOwnershipExactIdentity` count is unchanged.
- Family (e1)-(e3) in `07-MUTATION-LOG.md`: three planted mutations (the shared helper stubbed to `nil`, opencode's own gate call site short-circuited, the D-12 Note block deleted) each demonstrated RED for real and reverted byte-clean.
- The real-binary tracer (Task 1's `<verify>`) drove the actual compiled `codegraph` binary through `install --target codex,opencode` → `uninstall codex` (kept) → `uninstall opencode` (byte-identical restore) → reinstall → `uninstall --target all` (byte-identical restore again) → plant `AGENTS.override.md` → `install --target codex` (Note present, override untouched) — all steps passed.

## Task Commits

Each task was committed atomically (TDD RED/GREEN, per plan):

1. **Task 1 (RED): expect the shared AGENTS.md block to survive while another agent uses it** — `835db47f` (test)
2. **Task 1 (GREEN): keep a shared instructions block while another agent still uses it** — `c171fb6c` (feat)
3. **Task 2 (already-GREEN honest record): cover the shared AGENTS.md in the ownership table** — `ace22edd` (test; a positive-controlled guard, no code change required — see Deviations)
4. **Task 3: Family (e) mutation log** — `ed6a6043` (docs)

**Plan metadata:** committed separately after this SUMMARY (see below).

_Note: Task 1 followed TDD (`tdd="true"`) with a genuine test → feat RED/GREEN pair. Task 2 (also `tdd="true"`) is, by the plan's own instruction, a guard that "passes against Task 1's code" — recorded honestly as GREEN-at-write-time rather than reshaping it to force an artificial RED, following the 07-05-SUMMARY.md precedent for the same situation._

## RED Evidence (Task 1)

`GOTOOLCHAIN=go1.26.6 go test ./internal/agents/ -count=1 -run 'TestSharedAgentsMD_|TestCodex_Install_OverrideNote$' -v` against the unmodified (pre-D-11/D-12) code:

```
--- FAIL: TestCodex_Install_OverrideNote (0.01s)
    --- FAIL: TestCodex_Install_OverrideNote/local (0.00s)
    --- FAIL: TestCodex_Install_OverrideNote/global (0.00s)
    --- PASS: TestCodex_Install_OverrideNote/no_override_present (0.00s)
--- FAIL: TestSharedAgentsMD_UninstallOrders (0.02s)
    --- FAIL: TestSharedAgentsMD_UninstallOrders/codex_then_opencode/preexisting (0.00s)
    --- FAIL: TestSharedAgentsMD_UninstallOrders/codex_then_opencode/absent (0.00s)
    --- FAIL: TestSharedAgentsMD_UninstallOrders/opencode_then_codex/preexisting (0.00s)
    --- FAIL: TestSharedAgentsMD_UninstallOrders/opencode_then_codex/absent (0.00s)
    --- PASS: TestSharedAgentsMD_UninstallOrders/target_all/preexisting (0.00s)
    --- PASS: TestSharedAgentsMD_UninstallOrders/target_all/absent (0.00s)
--- FAIL: TestSharedAgentsMD_KeptWhileOtherConfigured (0.00s)
--- PASS: TestSharedAgentsMD_GlobalPathsNotShared (0.00s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/agents	0.151s
FAIL
```

`target_all` and `GlobalPathsNotShared` correctly stay green even at RED — the alphabetical `AllTargets()` uninstall order happens to strip codex before opencode either way, and global scope's differing absolute paths were never in question. All four affected tests turned GREEN after Task 1's `feat(07-06)` commit.

## Files Created/Modified

- `internal/agents/shared.go` — `instructionsRequestedElsewhere`, `instructionsKeptNote`
- `internal/agents/codex.go` — `Uninstall` gated on the helper; `Install` gains the D-12 override Note
- `internal/agents/opencode.go` — `Uninstall` gated on the same helper
- `internal/agents/codex_test.go` — `TestCodex_Install_OverrideNote` (local, global, no-override subtests)
- `internal/agents/shared_test.go` — `TestSharedAgentsMD_UninstallOrders`, `TestSharedAgentsMD_KeptWhileOtherConfigured`, `TestSharedAgentsMD_GlobalPathsNotShared`
- `internal/agents/ownership_test.go` — `TestOwnershipSharedInstructions`
- `.planning/phases/07-codex-parity/07-MUTATION-LOG.md` — Family (e1)-(e3)

## Decisions Made

See `key-decisions` in frontmatter.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Test-only path-comparison bug found while writing the RED commit**
- **Found during:** Task 1, first RED run
- **Issue:** `assertAgentsMDKept`'s call sites compared a `WriteResult.FileResult.Path` (which is the relative literal `"AGENTS.md"` — `caps.InstructionsPath(LocationLocal)`'s actual return value) against an absolute path built by joining the test's temp dir. This produced spurious failures reporting the file as NOT kept even where the underlying gate was already working correctly.
- **Fix:** Pass the literal relative string `"AGENTS.md"` to `assertAgentsMDKept`, keeping the separately-tracked absolute `agentsPath` variable only for `readFile`/`fileExists` byte checks.
- **Files modified:** `internal/agents/shared_test.go`
- **Verification:** re-ran the full targeted suite after the fix — all intended RED subtests failed for the correct reason (missing gate), all intended-green subtests (target_all, GlobalPathsNotShared) stayed green
- **Committed in:** `835db47f` (Task 1 RED commit — fixed before commit, never landed as broken)

**2. [Rule 1 - Bug, documented not silently patched] Task 3's own plan prediction for Family (e2) does not hold**
- **Found during:** Task 3, Family (e2) mutation
- **Issue:** The plan text predicted the opencode-gate-disabled mutation would turn BOTH `TestSharedAgentsMD_UninstallOrders/opencode_then_codex` AND `TestOwnershipSharedInstructions/opencode_then_codex` RED. Empirically, only the former does — `TestOwnershipSharedInstructions` (Task 2, built exactly to its own behavior spec: end-state-only assertions) cannot observe an intermediate-step regression whose effect converges away by the time both targets finish uninstalling.
- **Fix:** None needed — the plan's Task 3 `<verify>` gate itself already accepts either test's FAIL line via an OR (`rg -q -- '...UninstallOrders/opencode_then_codex|...SharedInstructions/opencode_then_codex'`), so this is not a blocking gap. Documented the structural reason in `07-MUTATION-LOG.md`'s Family (e2) entry rather than reshaping `TestOwnershipSharedInstructions` to add an intermediate-state assertion it was never specified to have.
- **Files modified:** none (documentation only, in `07-MUTATION-LOG.md`)
- **Verification:** confirmed via direct experimentation — ran the mutation, observed which test(s) actually failed, confirmed the plan's own automated gate still passes via its OR condition
- **Committed in:** `ed6a6043` (Family (e) mutation log)

---

**Total deviations:** 2 auto-fixed (1 test-only bug fixed before it ever landed; 1 documented finding that a plan prediction didn't hold, with no code or test change required)
**Impact on plan:** Neither affected product behavior or test coverage. No scope creep.

## Issues Encountered

None beyond the two deviations above.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- CODEX-04 is the last requirement 07-06 declares; `gsd_run query requirements.ready-ids` confirms it is ready to mark complete (no sibling plan also declares it).
- The shared-instructions mechanism (`instructionsRequestedElsewhere`) is now a general-purpose, registry-derived primitive — any future target sharing an instructions path with codex/opencode (or with each other) at any location automatically participates in the same "keep while another sharer needs it" discipline with no code change to this plan's work.
- `internal/agents/codex.go` and `internal/agents/opencode.go` are both fully consistent with the capability-table-driven shape 07-05 established; 07-07 (the Codex PreToolUse nudge) can build directly on top without touching anything this plan changed.
- The real-binary tracer proves the full CODEX-04 contract end-to-end through the actual compiled binary, not just unit tests.
- No blockers.

## Self-Check: PASSED

- `internal/agents/shared.go` — FOUND, contains `func instructionsRequestedElsewhere(`
- `internal/agents/codex.go` — FOUND, contains `instructionsRequestedElsewhere(instrPath, loc, t.ID())` and the D-12 `AGENTS.override.md` Note block
- `internal/agents/opencode.go` — FOUND, contains `instructionsRequestedElsewhere(instrPath, loc, t.ID())`
- `.planning/phases/07-codex-parity/07-MUTATION-LOG.md` — FOUND, contains `## Family (e1)` through `## Family (e3)`
- Commit `835db47f` (test, Task 1 RED) — FOUND in `git log --oneline --all`
- Commit `c171fb6c` (feat, Task 1 GREEN) — FOUND in `git log --oneline --all`
- Commit `ace22edd` (test, Task 2) — FOUND in `git log --oneline --all`
- Commit `ed6a6043` (docs, Task 3) — FOUND in `git log --oneline --all`
- `GOTOOLCHAIN=go1.26.6 go test ./internal/agents/ -count=1` re-run at SUMMARY time — exit 0, `ok`
- Real-binary tracer (install/uninstall sequences + override Note) re-run at SUMMARY time — all `cmp`/`rg` assertions passed (`TRACER_OK`)
- `rg -c 'instructionsRequestedElsewhere\(' internal/agents/codex.go internal/agents/opencode.go` — 1 in each file, confirmed
- `rg -n 'AGENTS\.override\.md' internal/agents/*.go` — hits only a path join and Note text, never a write call argument, confirmed

---
*Phase: 07-codex-parity*
*Completed: 2026-09-19*
