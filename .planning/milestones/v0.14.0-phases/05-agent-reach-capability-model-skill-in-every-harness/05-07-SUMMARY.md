---
phase: 05-agent-reach-capability-model-skill-in-every-harness
plan: 07
subsystem: agents
tags: [antigravity, skill-package, cursor, d-11, help-text, cli-reference, phase-gate, tdd]
status: complete

# Dependency graph
requires:
  - phase: 05-06
    provides: "05-LIVE-SESSIONS.md: D-11 verdict `not probed`, Antigravity verdict `not read`, maintainer decisions 1A/2A/3A"
  - phase: 05-05
    provides: "antigravitySkillDirs (written antigravity-cli path + [ASSUMED] config path), the D-13 ownership oracle for antigravity"
provides:
  - "antigravitySkillDirs(global) = [~/.gemini/config/skills/codegraph] — the one directory agy 1.2.6 is live-proven to read (1A); antigravity-cli path no longer declared"
  - "TestAntigravity_Install_WritesConfigSkillDir with a byte-compared foreign gh-stack sibling, positive-controlled as 05-MUTATION-LOG Family (c)"
  - "install/uninstall Long help naming the skill package, joint ownership of shared dirs, the last-requester rule and --print-config-style; docs/CLI-REFERENCE.md regenerated via task docs:cli"
  - "D-11 applied as recorded: `not probed`, so Cursor unchanged (writes no instructions file)"
affects: [phase-07-agent-14]

actuals:
  tokens: 5838
  tasks: 3
  commits: 3
plan_head_before: 0f2549f5508e994232c74862e7543e0fa7413e15

tech-stack:
  added: []
  patterns:
    - "Positive control planted against an uncommitted GREEN change: byte-clean proof is a sha256 of the GREEN snapshot, not an empty git status"
    - "Foreign-sibling guard asserts on two axes (parent survives, sibling bytes equal) and each axis gets its own planted mutation"

key-files:
  created: []
  modified:
    - internal/agents/antigravity.go
    - internal/agents/antigravity_test.go
    - internal/agents/capabilities_test.go
    - internal/agents/ownership_test.go
    - internal/cli/testdata/plain/print-config-style.golden
    - internal/cli/install.go
    - internal/cli/uninstall.go
    - docs/CLI-REFERENCE.md
    - .planning/phases/05-agent-reach-capability-model-skill-in-every-harness/05-MUTATION-LOG.md

key-decisions:
  - "D-11 recorded as `not probed` (no Cursor account), so 05-07 took the no-change branch: Cursor keeps writing no instructions file and its legacy .cursor/rules/codegraph.mdc self-heal is unchanged."
  - "Antigravity's skill moves to ~/.gemini/config/skills/codegraph (1A). The antigravity-cli path shipped only inside this unreleased milestone (05-05), so no migration or legacy cleanup was added."
  - "Family (c) got a second plant (c-ii, a one-byte change to the sibling) because the parent-removal plant (c-i) trips the parent-exists assertion before the byte-compare runs. c-ii shows the byte-compare catches a change on its own."

requirements-completed: [AGENT-04, AGENT-07, AGENT-08]

duration: 13min
completed: 2026-09-18
---

# Phase 5 Plan 07: Close the phase on the live evidence Summary

**Antigravity's skill now lands in `~/.gemini/config/skills/codegraph`, the one directory `agy` 1.2.6 was shown to read live (1A), with a byte-compared foreign-sibling guard tested by planted mutations. Cursor is unchanged because D-11 was `not probed`, install/uninstall help now describes the skill package, and the CLI reference was regenerated through the drift gate. The phase gate is green.**

## Performance

- **Duration:** ~13 min
- **Started:** 2026-09-19T00:06:43Z
- **Completed:** 2026-09-19T00:19:32Z
- **Tasks:** 3
- **Files modified:** 9

## Accomplishments

- Task 1 (D-11): verdict `not probed` → no-change branch; Task 1 verify green (ownership 32/32, local golden cursor line `instructions=none`).
- Task 2 (1A): RED `test(05-07):` commit landed before GREEN `fix(05-07):`. The plan now writes one Antigravity skill dir, the config path. The plain golden's antigravity line reads `skill=<HOME>/.gemini/config/skills/codegraph`, and Family (c) is recorded with a byte-clean revert.
- Task 3: help text updated, `task docs:cli` regenerated only the install/uninstall sections, and the phase gate is green (see below).

## Task 1: D-11 verdict applied, no-change branch

`V=not probed`, exactly one matching line (`05-LIVE-SESSIONS.md:149`). Supporting lines from 05-LIVE-SESSIONS.md:

> Maintainer decision: no Cursor account (2026-09-18) — Cursor is `[ASSUMED]`; its installed and control sessions are not run.
>
> Probe repo scaffolded at `/tmp/05-live/cursor-agentsmd-probe` (sentinel `PLUMTREE-4471` in `AGENTS.md`, no `.cursor/rules/` file present). The probe could not run for the same reason as above. D-11 only permits switching Cursor's instructions target after an **observed** pickup, so an unrun probe leaves Cursor writing no instructions file; 05-07 takes its no-change branch.
>
> Maintainer decision: the D-11 probe cannot run without a Cursor account (2026-09-18, 05-CONTEXT.md D-11, `f83724ee`) — recorded as not probed.
>
> D-11 verdict: not probed

Result: Cursor writes no instructions file and its legacy `.cursor/rules/codegraph.mdc` self-heal is unchanged. The CONTEXT line describing `.cursor/rules/codegraph.mdc` as Cursor's current target was inaccurate (cursor.go:9-16); the code's actual behaviour is what ships. `git log --format=%s -- internal/agents/cursor.go` shows no 05-07 commit. Task 1 verify: agents and cli both `ok`, no FAIL lines, 32 `TestOwnershipExactIdentity` PASS leaves, local golden `cursor: ... instructions=none`.

## Task 2: RED evidence (commit 21383622, before GREEN 798c9d4b)

`GOTOOLCHAIN=go1.26.6 go test ./internal/agents/ -count=1 -run 'TestAntigravity|TestCapabilitiesDeclared$|TestOwnershipExactIdentity(_CrossCheckWrittenSkillDir)?$' -v` (temp paths shortened):

```
--- FAIL: TestAntigravity_Install_WritesConfigSkillDir (0.00s)
    antigravity_test.go:226: expected the codegraph SKILL.md at .../.gemini/config/skills/codegraph/SKILL.md after install
--- FAIL: TestAntigravity_NoReadOnlySkillDirs (0.00s)
    antigravity_test.go:286: ReadOnlySkillDirs(global) = [.../.gemini/config/skills/codegraph], want none
--- FAIL: TestCapabilitiesDeclared (0.00s)
    --- FAIL: TestCapabilitiesDeclared/antigravity (0.00s)
    capabilities_test.go:218: WrittenSkillDir(global) = ".../.gemini/antigravity-cli/skills/codegraph", want ".../.gemini/config/skills/codegraph"
--- FAIL: TestOwnershipExactIdentity (0.12s)
    --- FAIL: TestOwnershipExactIdentity/antigravity/global/clean (0.00s)
    --- FAIL: TestOwnershipExactIdentity/antigravity/global/foreign-codegraph-dir (0.00s)
--- FAIL: TestOwnershipExactIdentity_CrossCheckWrittenSkillDir (0.01s)
    --- FAIL: TestOwnershipExactIdentity_CrossCheckWrittenSkillDir/antigravity/global (0.00s)
    ownership_test.go:578: WrittenSkillDir(global) = ".../.gemini/antigravity-cli/skills/codegraph", want oracle ".../.gemini/config/skills/codegraph"
FAIL	github.com/seanb4t/codegraph-go/internal/agents	0.239s
```

GREEN: `antigravitySkillDirs(global)` returns exactly `[~/.gemini/config/skills/codegraph]`, with its doc comment rewritten to cite the live session and 1A. The plain golden was regenerated with `-update-plain-goldens`; only the antigravity line changed, and `print-config-style-local.golden` is unchanged. `TestAntigravity_Install_UnifiedWriteFailure_PreservesLegacyEntryNoMarker` still passes (the chmod of `~/.gemini/config` now also blocks the skill write; its assertions still hold). Task 2 verify is green: both packages `ok`, 32/32 ownership leaves, and WritesConfigSkillDir, NoReadOnlySkillDirs and CrossCheckWrittenSkillDir all PASS.

The antigravity-cli path shipped only inside this unreleased milestone (05-05), so no migration or legacy cleanup was added.

### Family (c) positive control (05-MUTATION-LOG.md)

- (c-i) `os.RemoveAll(filepath.Dir(dir))` after the skill sweep →
  `antigravity_test.go:264: uninstall removed .../.gemini/config/skills, which still holds the foreign gh-stack skill` / `--- FAIL: TestAntigravity_Install_WritesConfigSkillDir (0.00s)`, exit=1
- (c-ii) append one byte to `gh-stack/SKILL.md` during uninstall →
  `antigravity_test.go:270: foreign sibling skill .../.gemini/config/skills/gh-stack/SKILL.md changed by uninstall:` (got/want shown) / `--- FAIL: TestAntigravity_Install_WritesConfigSkillDir (0.00s)`, exit=1
- Revert proof: after each revert, `shasum -a 256 -c` against the GREEN snapshot hash `aee9b451…c20d6` printed `internal/agents/antigravity.go: OK`, `cmp` showed no difference, and `git diff --stat -- internal/agents/` showed only the GREEN change. The green re-run printed `ok … exit=0`.

## Task 3: help, reference, phase gate

Help text and reference: install Long names the skill package (SKILL.md plus sidecar manifest), jointly owned shared dirs, untouched foreign `codegraph/` dirs and `--print-config-style`. uninstall Long names the last-requester rule and says foreign dirs are never touched. The `docs/CLI-REFERENCE.md` diff covers only the install and uninstall sections.

| Gate | Result |
|---|---|
| `GOTOOLCHAIN=go1.26.6 go build ./...` | exit 0 |
| module suite minus `internal/daemon` (`/tmp/05-07-suite.txt`) | exit=0, 52 `ok`, no `--- FAIL`/`FAIL` lines (two runs, both clean) |
| `internal/daemon` alone, run 1 (`/tmp/05-07-daemon-run1.txt`) | **exit=1**: all tests PASS, then goleak reported an unexpected `pebble/v2/vfs.(*diskHealthCheckingFS).startTickerLocked.func1` goroutine → `FAIL internal/daemon 64.714s` |
| `internal/daemon` alone, run 2 (`/tmp/05-07-daemon-run2.txt`) | exit=0, `ok 63.977s` (load average ~23) |
| `internal/daemon` alone, run 3 (in Task 3 verify, `/tmp/05-07-daemon.txt`) | exit=0, `ok 64.006s` |
| `task docs:cli:drift` | exit=0, "docs/CLI-REFERENCE.md byte-identical to a fresh regeneration" |
| `TestEveryRegisteredFlagIsAccountedFor` | ok |
| 05-MUTATION-LOG.md families (a1), (a2), (b1), (b2) (+ new (c)) | present |
| 05-LIVE-SESSIONS.md PENDING-free | yes |
| AGENT-14 surfaces (`instructions.go`, `internal/mcp/`) untouched this phase | yes against this milestone's Phase 5 base (see Deviation 1) |
| no `[ci skip]`/`[skip ci]` in any Phase 5 commit | yes |

The daemon run-1 failure is a goleak finding on a pebble disk-health ticker goroutine in a package this plan does not touch (no diff under `internal/daemon`, `internal/graphstore`, `go.mod`, `go.sum`). STATE already records the same class of stray pebble vfs ticker goroutine under goleak (Phase 06 uiserver decision) and WINDOWS #37 daemon load-flakiness. Runs 2 and 3 passed alone. Nothing was suppressed or skipped. No `internal/cli` failure occurred in either module-suite run.

## Advisories for the maintainer (not changed here)

1. The name-based self-heal deletes `.cursor/rules/codegraph.mdc` and `.kiro/steering/codegraph.md` without an ownership check. This same-name ownership gap predates this phase and is outside its writes; candidate todo.
2. `codegraph upgrade` refreshes only Claude's skill package. The shared and harness-specific SKILL.md files refresh on the next `codegraph install`.
3. Kiro may read the codegraph block twice, via the AGENTS.md written by opencode/Codex (D-06(d) advisory).
4. AGENT-14 (Phase 7) must update instructions.go's "4 of 8" comment and the MCP instructions skill sentence in internal/mcp/server.go.
5. The CONTEXT code-insights lines describing Cursor's and Kiro's instruction files were inaccurate. The code's actual behaviour shipped unchanged.
6. Uninstall leaves behind an empty parent skills directory that install created (observed live for Antigravity's old path, 05-LIVE-SESSIONS.md § Cleanup). `removeSkillDirIfEmpty` sweeps only the codegraph dir, not its parent; candidate todo. Removing the parent unconditionally would be wrong: Family (c-i) shows that plant turning the gh-stack guard RED.

## Task Commits

1. **Task 1: D-11 verdict** — no commit (no-change branch, by design)
2. **Task 2 RED** — `21383622` test(05-07): expect Antigravity's skill at ~/.gemini/config/skills (1A)
3. **Task 2 GREEN** — `798c9d4b` fix(05-07): Antigravity writes its skill to ~/.gemini/config/skills, the directory agy reads (1A)
4. **Task 3** — `1c10c974` docs(05-07): describe the skill package and --print-config-style in install/uninstall help; regenerate CLI reference

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Phase-gate AGENT-14 base selector picked a prior milestone's commit**
- **Found during:** Task 3 verify
- **Issue:** `B=$(git log --format=%H -E --grep='^test\(05-01\): ' | tail -1)` selects the *oldest* matching commit, `e3de9205` (2026-07-16, an earlier milestone's Phase 5 "fsatomic.WriteFile" test). Against it, `instructions.go` and 30 `internal/mcp/` files differ because of v0.10.0–v0.13 work (#24, #44, #60, #63, #66). As written, the check could never pass.
- **Fix:** re-ran the check against this milestone's Phase 5 base: `16708421` (test(05-01): add failing capability-table …, 2026-09-18) and the phase's first commit `86c3cd87`. `git diff --quiet <base>^ HEAD -- internal/agents/instructions.go internal/mcp/` → exit 0 for both. All other Task 3 checks passed individually as written. Future plans should select with `head -1` or bound the log with `main..HEAD`.
- **Files modified:** none (evidence only)

**2. [Rule 2 - Missing critical] Second Family (c) plant for the byte-compare axis**
- **Found during:** Task 2 positive control
- **Issue:** the plan's parent-removal plant trips the parent-exists assertion before the byte-compare, so on its own it does not show the byte-compare discriminates.
- **Fix:** added plant (c-ii), a one-byte change to the sibling that leaves the parent in place. It turned the byte-compare RED, and it was reverted byte-clean. Both plants are in Family (c). The mutation-log Summary closing note now says Family (c)'s clean state is the snapshot hash, not an empty `git status`.
- **Commit:** 798c9d4b

---

**Total deviations:** 2 (1 gate-evidence correction, 1 extra positive control). No scope creep; no production behaviour beyond the plan.

## Issues Encountered

- `internal/daemon` goleak failure on the first isolated run (pebble disk-health ticker goroutine). It passed on the next two isolated runs. Details are in the gate table.

## Known Stubs

None.

## Next Phase Readiness

This was Phase 5's last plan. Phase 7 (AGENT-14) still owns instructions.go's coverage comment, the MCP instructions skill sentence and the published per-harness capability table (advisory 4).

## Self-Check: PASSED

- FOUND: internal/agents/antigravity.go, internal/agents/antigravity_test.go, internal/cli/install.go, internal/cli/uninstall.go, docs/CLI-REFERENCE.md, 05-MUTATION-LOG.md
- FOUND commits: 21383622, 798c9d4b, 1c10c974
