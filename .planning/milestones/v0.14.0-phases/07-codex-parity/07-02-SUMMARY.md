---
phase: 07-codex-parity
plan: 02
subsystem: agent-installer
tags: [cli, install, uninstall, flag-resolution, tdd, mutation-testing]

# Dependency graph
requires:
  - phase: 07-codex-parity
    provides: "findTOMLTableRange/tomlTableConflict/tomlLineEnding (07-01) — untouched by this plan"
provides:
  - "install.go/uninstall.go RunE switches: an explicit --target is checked before --yes (D-13)"
  - "TestInstall_YesWithExplicitTarget_HonoursTarget, TestUninstall_YesWithExplicitTarget_HonoursTarget"
  - "07-MUTATION-LOG.md Family (b): b1 (install.go), b2 (uninstall.go)"
  - "The 2026-09-18-install-yes-discards-explicit-target.md todo, completed"
affects: [07-04, 07-09]

# Actuals (#2632)
actuals:
  tokens: 3730
  tasks: 2
  commits: 3

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Switch-case reordering as the fix shape: the explicit-selector case moves above the flag-short-circuit case with no other logic touched, mirrored identically across install.go and uninstall.go"

key-files:
  created: []
  modified:
    - internal/cli/install.go
    - internal/cli/uninstall.go
    - internal/cli/install_test.go
    - .planning/phases/07-codex-parity/07-MUTATION-LOG.md
    - .planning/todos/pending/2026-09-18-install-yes-discards-explicit-target.md
    - .planning/todos/completed/2026-09-18-install-yes-discards-explicit-target.md

key-decisions:
  - "New tests placed at the end of install_test.go rather than interleaved next to the existing Pitfall-6 tests they extend — avoids disturbing the existing test ordering/line-number references other plans' interfaces cite (07-CONTEXT.md's install_test.go:496/551 line anchors stay valid)"
  - "Both RunE switches keep the exact same comment-explains-intent convention the plan's interfaces section required: the moved case explains D-13's precedence, and the --yes case's comment is reworded to state it applies only when --target was not given"

patterns-established: []

requirements-completed: [CODEX-02]

coverage:
  - id: D1
    description: "install --target X --yes and uninstall --target X --yes both configure/remove exactly X, never widening to auto (install) or all (uninstall) — D-13 closed in both commands"
    requirement: "CODEX-02"
    verification:
      - kind: unit
        ref: "internal/cli/install_test.go#TestInstall_YesWithExplicitTarget_HonoursTarget"
        status: pass
      - kind: unit
        ref: "internal/cli/install_test.go#TestUninstall_YesWithExplicitTarget_HonoursTarget"
        status: pass
      - kind: unit
        ref: "internal/cli/install_test.go#TestInstall_Yes_ShortCircuitsBeforeInteractiveBranch (Pitfall-6 regression, still green)"
        status: pass
      - kind: unit
        ref: "internal/cli/install_test.go#TestUninstall_Yes_ShortCircuitsBeforeInteractiveBranch (Pitfall-6 regression, still green)"
        status: pass
      - kind: unit
        ref: "internal/cli/install_test.go#TestInstall_InteractiveAllowed_CallsRunAgentPicker (switch wiring order, still green)"
        status: pass
      - kind: e2e
        ref: "real codegraph binary: install --target codex --yes --location global against a mktemp -d HOME — Codex CLI: configured, Claude Code: absent, ~/.codex/config.toml written, ~/.claude.json absent"
        status: pass
    human_judgment: false
  - id: D2
    description: "07-MUTATION-LOG.md Family (b) proves the resolution-order guard can fail against a confirmed-applied, byte-cleanly-reverted mutation, in both install.go and uninstall.go"
    verification:
      - kind: other
        ref: "perl-planted re-shadowing mutation and revert session, transcripts pasted into 07-MUTATION-LOG.md Family (b1)/(b2); git diff --quiet gates before/after each plant and revert"
        status: pass
    human_judgment: false
  - id: D3
    description: "The D-13 todo is completed through the todo verb, not by hand-moving files"
    verification:
      - kind: other
        ref: "gsd_run todo complete 2026-09-18-install-yes-discards-explicit-target.md → {completed: true}"
        status: pass
    human_judgment: false

duration: ~15min
completed: 2026-09-19
status: complete
---

# Phase 7 Plan 2: install/uninstall --yes vs --target Resolution Order Summary

**Swapped the `case yes:` / `case cmd.Flags().Changed("target"):` order in `install.go` and `uninstall.go` so an explicit `--target` always wins over `--yes`, closing the D-13 data-loss-adjacent bug where `install --target codex --yes` silently configured the auto-detected agent instead of Codex.**

## Performance

- **Duration:** ~15 min
- **Started:** 2026-09-19 (commits span 12:16–12:20 local; context reading preceded the first commit)
- **Completed:** 2026-09-19T16:20:38Z
- **Tasks:** 2
- **Files modified:** 6 (2 source, 1 test, 1 mutation log, 1 todo moved pending→completed)

## Accomplishments

- `install.go`'s `RunE` switch now checks `cmd.Flags().Changed("target")` before `yes` — an explicit `--target` is resolved first; `--yes` only supplies the non-interactive `auto` default when no target was named, and still short-circuits before the interactive picker branch (Pitfall 6) in that case.
- `uninstall.go`'s structurally identical switch gets the same fix: an explicit `--target` wins over `--yes`, which otherwise falls back to `all` and would remove every installed agent's configuration.
- Two new tests (`TestInstall_YesWithExplicitTarget_HonoursTarget`, `TestUninstall_YesWithExplicitTarget_HonoursTarget`) were RED on the pre-fix ordering — pasted below — and are GREEN after the fix, alongside all five named regression/wiring tests the plan's verify gate names.
- Verified against the real binary, not just unit tests: a fresh `mktemp -d` HOME with `install --target codex --yes --location global` writes only `~/.codex/config.toml` (plus `~/.codex/AGENTS.md`) and never touches `~/.claude.json`.
- `07-MUTATION-LOG.md` gained Family (b): two entries (b1 install.go, b2 uninstall.go) proving the resolution-order guard can fail against a confirmed-applied, byte-cleanly-reverted mutation (`case !yes && cmd.Flags().Changed("target"):` re-shadowing).
- The `2026-09-18-install-yes-discards-explicit-target.md` todo is completed through `gsd_run todo complete`, not by hand-moving the file.

## Task Commits

Each task was committed atomically (TDD RED/GREEN pair, then docs):

1. **Task 1: RED — failing --target-with---yes resolution tests** — `0a403bc8` (test)
2. **Task 1: GREEN — honour an explicit --target under --yes; complete the todo** — `416d553d` (fix)
3. **Task 2: Family (b) mutation log** — `7a353a11` (docs)

**Plan metadata:** committed separately after this SUMMARY (see below).

## Files Created/Modified

- `internal/cli/install.go` — `RunE`'s target-resolution switch: `case cmd.Flags().Changed("target"):` moved above `case yes:`, comments reworded to state the new precedence
- `internal/cli/uninstall.go` — identical switch-order fix, same comment convention
- `internal/cli/install_test.go` — `TestInstall_YesWithExplicitTarget_HonoursTarget`, `TestUninstall_YesWithExplicitTarget_HonoursTarget` (uninstall's tests live in this file, per the existing convention — there is no separate `uninstall_test.go`)
- `.planning/phases/07-codex-parity/07-MUTATION-LOG.md` — Family (b1)/(b2) mutation entries appended
- `.planning/todos/pending/2026-09-18-install-yes-discards-explicit-target.md` → `.planning/todos/completed/2026-09-18-install-yes-discards-explicit-target.md` (moved via the `todo complete` verb)

## RED Transcript

### Task 1 RED (`test(07-02): add failing --target with --yes resolution tests`, commit `0a403bc8`)

```
=== RUN   TestInstall_YesWithExplicitTarget_HonoursTarget
    install_test.go:882: expected explicit --target codex to configure Codex, got:
        Claude Code: configured
          created: /var/folders/.../T/TestInstall_YesWithExplicitTarget_HonoursTarget1299165345/001/.claude.json
          created: /var/folders/.../T/TestInstall_YesWithExplicitTarget_HonoursTarget1299165345/001/.claude/CLAUDE.md
          created: /var/folders/.../T/TestInstall_YesWithExplicitTarget_HonoursTarget1299165345/001/.claude/skills/codegraph/SKILL.md
          created: /var/folders/.../T/TestInstall_YesWithExplicitTarget_HonoursTarget1299165345/001/.claude/hooks/session-nudge.sh
          created: /var/folders/.../T/TestInstall_YesWithExplicitTarget_HonoursTarget1299165345/001/.claude/settings.json
          created: /var/folders/.../T/TestInstall_YesWithExplicitTarget_HonoursTarget1299165345/001/.claude/skills/codegraph/.codegraph-manifest.json
--- FAIL: TestInstall_YesWithExplicitTarget_HonoursTarget (0.00s)
=== RUN   TestUninstall_YesWithExplicitTarget_HonoursTarget
    install_test.go:920: expected --yes NOT to widen an explicit --target codex to Claude, got:
        Antigravity: not-configured
        ...
        Claude Code: removed
          removed: /var/folders/.../T/TestUninstall_YesWithExplicitTarget_HonoursTarget1321580334/001/.claude.json
          removed: /var/folders/.../T/TestUninstall_YesWithExplicitTarget_HonoursTarget1321580334/001/.claude/CLAUDE.md
          ...
        Codex CLI: removed
          removed: /var/folders/.../T/TestUninstall_YesWithExplicitTarget_HonoursTarget1321580334/001/.codex/config.toml
          removed: /var/folders/.../T/TestUninstall_YesWithExplicitTarget_HonoursTarget1321580334/001/.codex/AGENTS.md
        Cursor: not-configured
        Gemini CLI: not-configured
        Hermes Agent: not-configured
        Kiro: not-configured
        opencode: not-configured
--- FAIL: TestUninstall_YesWithExplicitTarget_HonoursTarget (0.00s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/cli	0.458s
FAIL
exit=1
```

Both new tests failed exactly as predicted: install's `-y`/`--target codex` combination resolved to `auto` (falling back to Claude in a fresh fake home), and uninstall's `--target codex --yes` resolved to `all`, removing Claude's configuration alongside Codex's instead of leaving it untouched. No pre-existing test in the file was affected.

### Post-fix (`fix(07-02): honour an explicit --target under --yes`, commit `416d553d`)

```
=== RUN   TestInstall_Yes_ShortCircuitsBeforeInteractiveBranch
--- PASS: TestInstall_Yes_ShortCircuitsBeforeInteractiveBranch (0.00s)
=== RUN   TestInstall_InteractiveAllowed_CallsRunAgentPicker
--- PASS: TestInstall_InteractiveAllowed_CallsRunAgentPicker (0.00s)
=== RUN   TestUninstall_Yes_ShortCircuitsBeforeInteractiveBranch
--- PASS: TestUninstall_Yes_ShortCircuitsBeforeInteractiveBranch (0.00s)
=== RUN   TestInstall_YesWithExplicitTarget_HonoursTarget
--- PASS: TestInstall_YesWithExplicitTarget_HonoursTarget (0.00s)
=== RUN   TestUninstall_YesWithExplicitTarget_HonoursTarget
--- PASS: TestUninstall_YesWithExplicitTarget_HonoursTarget (0.00s)
PASS
ok  	github.com/seanb4t/codegraph-go/internal/cli	0.516s
```

Full package (`go test ./internal/cli/ -count=1`) → `ok  github.com/seanb4t/codegraph-go/internal/cli  17.291s`.

Real binary (`install --target codex --yes --location global` against a `mktemp -d` HOME):

```
Codex CLI: configured
  created: /var/folders/.../T/tmp.AlNN4h5U5o/home/.codex/config.toml
  created: /var/folders/.../T/tmp.AlNN4h5U5o/home/.codex/AGENTS.md
```

`~/.claude.json` does not exist afterward.

## Decisions Made

- The two new tests were appended at the end of `install_test.go` rather than interleaved beside `TestInstall_Yes_ShortCircuitsBeforeInteractiveBranch`/`TestUninstall_Yes_ShortCircuitsBeforeInteractiveBranch` — this avoids shifting the line numbers 07-CONTEXT.md's interfaces section cites for the existing tests (`:496`, `:551`), which other in-flight plan context may still reference.
- Both switches' reworded comments make the new precedence explicit inline (the moved case explains it wins over `--yes`; the `--yes` case's comment now states it applies "whenever `--target` was not given") rather than relying solely on the commit message to convey the change — matching the file's existing convention of explaining non-obvious ordering decisions in-line.

## Deviations from Plan

None — plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- D-13 is fully closed in both `install.go` and `uninstall.go`: an explicit `--target` is checked before `--yes` in both RunE switches, proven RED before the fix and through the real binary after.
- The `2026-09-18-install-yes-discards-explicit-target.md` todo is completed (moved to `.planning/todos/completed/`).
- `internal/agents/codex.go` remains untouched, consistent with D-01's plan ordering — the CODEX-01 live verification (07-04) and the scope-flip work (07-05+) are next, both of which depend on `install --target codex --yes` behaving correctly, which this plan now guarantees.
- 07-VALIDATION row 07-YES is satisfiable: `go test ./internal/cli/ -run 'TestInstall.*Yes.*Target|TestUninstall.*Yes.*Target' -count=1` passes both named tests.

## Self-Check: PASSED

- `internal/cli/install.go` — case-order fix present (`rg -n` confirms `case cmd.Flags().Changed("target"):` at line 112, `case yes:` at line 117)
- `internal/cli/uninstall.go` — case-order fix present (line 51 before line 56)
- `internal/cli/install_test.go` — `TestInstall_YesWithExplicitTarget_HonoursTarget` and `TestUninstall_YesWithExplicitTarget_HonoursTarget` — FOUND
- `.planning/phases/07-codex-parity/07-MUTATION-LOG.md` — Family (b1)/(b2) — FOUND
- `.planning/todos/completed/2026-09-18-install-yes-discards-explicit-target.md` — FOUND; `.planning/todos/pending/2026-09-18-install-yes-discards-explicit-target.md` — absent
- Commit `0a403bc8` (test) — FOUND in `git log --oneline`
- Commit `416d553d` (fix) — FOUND in `git log --oneline`
- Commit `7a353a11` (docs) — FOUND in `git log --oneline`
- All plan-level `<verification>` commands re-run at HEAD: `go test ./internal/cli/ -count=1` → `ok`; real-binary `--target codex --yes` round trip → Codex configured, Claude absent; `07-MUTATION-LOG.md` Family (b1)/(b2) present with all six required subsections each; `git diff --quiet -- internal/cli/install.go internal/cli/uninstall.go` clean at rest — all PASS.

---
*Phase: 07-codex-parity*
*Completed: 2026-09-19*
