---
phase: 12-cli-reference-docs-tail
plan: 01
subsystem: cli
tags: [cobra, cobra-doc, docs-generation, taskfile, ci, accounting-guard]

# Dependency graph
requires: []
provides:
  - "internal/cli.NewRootCmd() — exported wrapper around newRootCmd(), consumed by tools/clidoc"
  - "tools/clidoc/main.go — the DOCS-05 CLI reference generator"
  - "docs/CLI-REFERENCE.md — generated, committed reference (35 command sections)"
  - "task docs:cli / task docs:cli:drift — regen-in-place / regen-to-temp byte-compare gate"
  - "ci.yml CLI reference drift guard (DOCS-05) step, after proto:drift"
  - "internal/cli/cli_reference_test.go TestEveryRegisteredFlagIsAccountedFor — DOCS-06 accounting guard"
  - "internal/cli/testdata/cli-reference-allowlist.txt — one-entry allowlist (codegraph man)"
affects: [12-02-brew-trust-and-readme-link, 12-03-mutation-log-and-security]

# Actuals (#2632)
actuals:
  tokens: 34000
  tasks: 2
  commits: 2

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "One generator binary, two Taskfile call sites (regen-in-place vs regen-to-temp-and-compare), mirroring proto:gen/proto:drift"
    - "Positive-count-floor guards that log before asserting, never a vacuous pass (rule 84d1gfpywd)"
    - "Committed, reason-carrying allowlist for a generator's structural blind spot, with rot enforcement (unmatched entries fail the guard)"

key-files:
  created:
    - tools/clidoc/main.go
    - docs/CLI-REFERENCE.md
    - internal/cli/cli_reference_test.go
    - internal/cli/testdata/cli-reference-allowlist.txt
  modified:
    - internal/cli/root.go
    - Taskfile.yml
    - .github/workflows/ci.yml

key-decisions:
  - "NewRootCmd is an additive exported wrapper around newRootCmd(), not a rename (D-02/Claude's Discretion) — zero call-site changes, Execute() untouched"
  - "tools/clidoc completes the tree with both InitDefaultCompletionCmd() (per D-02) and InitDefaultVersionFlag() (planner addition beyond CONTEXT — see below) before generating or walking, in lockstep between the generator and the guard"
  - "docs:cli:drift's own floor-count variable is named ndocs, not nfiles, so its static Taskfile.yml text does not collide with proto:drift's pre-existing 'compared ${nfiles} generated files' line under a substring grep — a purely cosmetic naming choice with no behavioral effect"
  - "D-04 audit: no command gained a Long this phase — every Short-only command's Use line plus its flag usage strings already states its argument semantics, confirmed by reading the generated reference; only internal/cli/root.go changed among internal/cli/*.go non-test files"

patterns-established:
  - "A DOCS-05-shaped drift gate (regen-to-temp, byte-compare, report-count-before-comparing) is now established for any future single-file generated doc, not just protobuf"

requirements-completed: []  # DOCS-05/DOCS-06 are shared with 12-02/12-03 (shared-ID gate); not marked complete here — see Requirements note below

coverage:
  - id: D1
    description: "docs/CLI-REFERENCE.md generated from the live Cobra tree via cobra/doc, committed, and kept current by task docs:cli:drift wired into ci.yml (DOCS-05)"
    requirement: DOCS-05
    verification:
      - kind: integration
        ref: "task -s docs:cli:drift (RED on untracked file, GREEN x2 byte-identical)"
        status: pass
    human_judgment: false
  - id: D2
    description: "TestEveryRegisteredFlagIsAccountedFor walks the live tree (hidden commands included) and accounts for every flag via the reference or the allowlist (DOCS-06)"
    requirement: DOCS-06
    verification:
      - kind: unit
        ref: "internal/cli/cli_reference_test.go#TestEveryRegisteredFlagIsAccountedFor (RED missing allowlist, RED comment-only allowlist, GREEN one-entry allowlist)"
        status: pass
    human_judgment: false

duration: 55min
completed: 2026-09-13
status: complete
---

# Phase 12 Plan 01: CLI Reference Generator + Drift Gate + Accounting Guard Summary

**`docs/CLI-REFERENCE.md` is now `cobra/doc` output regenerated from the live 36-command tree by `tools/clidoc`, byte-stability-proven by `task docs:cli:drift` in CI, and every one of its 115 registered flags — including the hidden `man` command's — is accounted for by `TestEveryRegisteredFlagIsAccountedFor` against the reference or a one-entry, reason-carrying allowlist.**

## Performance

- **Duration:** 55 min
- **Started:** 2026-09-13T16:20:00Z (approx)
- **Completed:** 2026-09-13T17:15:00Z (approx)
- **Tasks:** 2
- **Files modified:** 7 (4 created, 3 modified)

## Accomplishments

- Exported `internal/cli.NewRootCmd()` so an out-of-package generator can build the exact tree `Execute()` runs, with zero call-site churn (`newRootCmd()` untouched, `Execute()` untouched)
- Built `tools/clidoc/main.go`: completes the tree with the completion family and the `-v/--version` flag, disables the date-stamped auto-gen tag, and walks it into ONE `docs/CLI-REFERENCE.md` via `cobra/doc.GenMarkdownCustom`, never cobra's own one-file-per-command layout
- Committed the generated `docs/CLI-REFERENCE.md` (35 `## codegraph…` sections, no hidden `man` section, 4 completion shells, `--editor-url`, root `-v, --version`, zero date footers, alphabetically pre-ordered) in the same commit as its generator and gates
- Added `task docs:cli` (regen-in-place) and `task docs:cli:drift` (regen-to-`mktemp -d`, byte-compare, count-before-compare, D-14-pinned toolchain), wired `ci.yml`'s `CLI reference drift guard (DOCS-05)` step immediately after `proto:drift`
- Built `TestEveryRegisteredFlagIsAccountedFor`: walks the completed tree recursively (hidden commands included), classifies each of 115 flags as reference-eligible or allowlist-eligible, and fails on any unaccounted flag or any allowlist entry that matches nothing (rot enforcement)
- Committed `internal/cli/testdata/cli-reference-allowlist.txt` with its one sanctioned entry, `codegraph man`, covering that hidden command's only flag (`--help`)

## Task Commits

Each task was committed atomically:

1. **Task 1: End-to-end "one generated CLI reference, kept current"** — `302e5fb1` (feat)
2. **Task 2: DOCS-06 accounting guard** — `2a899875` (test)

**Plan metadata:** committed separately (this SUMMARY)

## Files Created/Modified

- `internal/cli/root.go` — added `NewRootCmd()` exported wrapper (additive; `newRootCmd()` and `Execute()` unchanged)
- `tools/clidoc/main.go` — the DOCS-05 generator (new package, no `go.mod` change — `cobra/doc` already a dependency via `internal/cli/man.go`)
- `docs/CLI-REFERENCE.md` — generated, committed, never hand-edited
- `Taskfile.yml` — `docs:cli` / `docs:cli:drift` targets, inserted after `proto:drift`, before `web:deps`
- `.github/workflows/ci.yml` — `CLI reference drift guard (DOCS-05)` step, directly after `Proto codegen drift guard (BLD-04)`
- `internal/cli/cli_reference_test.go` — `TestEveryRegisteredFlagIsAccountedFor` and its helpers (`cliReferenceTree`, `documentedByReference`, `inheritedFromAncestor`, `docMentionsFlag`, `parseCLIReferenceAllowlist`, `ineligibleReason`)
- `internal/cli/testdata/cli-reference-allowlist.txt` — one entry: `codegraph man` (tab-separated reason)

## RED/GREEN Transcripts

### Task 1 — drift gate

**RED (untracked `docs/CLI-REFERENCE.md`, before `git add`):**
```
docs:cli:drift: compared 0 generated file
::error::docs:cli:drift: enumerated only 0 committed generated file at docs/CLI-REFERENCE.md (expected exactly 1) — a broken enumeration must fail loud, never read as a clean pass
task: Failed to run task "docs:cli:drift": exit status 1
```

**GREEN (run 1, after commit):**
```
docs:cli:drift: compared 1 generated file
docs:cli:drift: docs/CLI-REFERENCE.md byte-identical to a fresh regeneration (temporary file only — source tree untouched)
```

**GREEN (run 2, byte-identical to run 1):**
```
docs:cli:drift: compared 1 generated file
docs:cli:drift: docs/CLI-REFERENCE.md byte-identical to a fresh regeneration (temporary file only — source tree untouched)
```

### Task 2 — accounting guard

**RED 1 (allowlist file missing):**
```
--- FAIL: TestEveryRegisteredFlagIsAccountedFor (0.00s)
    cli_reference_test.go:154: fail-closed: testdata/cli-reference-allowlist.txt must exist and be readable: open testdata/cli-reference-allowlist.txt: no such file or directory
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/cli	0.482s
FAIL
```

**RED 2 (comment-only allowlist — zero entries):**
```
    cli_reference_test.go:236: walked 36 commands (hidden included), inspected 115 flags: 114 accepted via ../../docs/CLI-REFERENCE.md, 0 accepted via testdata/cli-reference-allowlist.txt
    cli_reference_test.go:256: 1 problem(s):
        unaccounted flag: codegraph man --help (flag on a hidden command — add an allowlist entry with a reason)
--- FAIL: TestEveryRegisteredFlagIsAccountedFor (0.02s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/cli	0.437s
FAIL
```

**GREEN (one-entry allowlist):**
```
    cli_reference_test.go:236: walked 36 commands (hidden included), inspected 115 flags: 114 accepted via ../../docs/CLI-REFERENCE.md, 1 accepted via testdata/cli-reference-allowlist.txt
--- PASS: TestEveryRegisteredFlagIsAccountedFor (0.02s)
ok  	github.com/seanb4t/codegraph-go/internal/cli	0.438s
```

## D-04 Audit Outcome

Applied the D-04 criterion to every Short-only command (`affected`, `callees`, `callers`, `explore`, `files`, `impact`, `index`, `init`, `node`, `query`, `search`, `status`, `sync`, `uninit`, `unlock`, `daemon start|stop`, `githooks install|remove|status`) by reading the generated `docs/CLI-REFERENCE.md`: each command's `Use` line plus its flag usage strings already states its argument semantics (e.g. `node [symbol]` with `--file`/`--line` explaining both lookup modes, `affected [files...]` with `--stdin` explaining the union, `explore <query...>`, `impact <symbol>` with `--depth (default 2, max 50)`), and `ui`'s existing `Long` already documents the `--editor-url` / `CODEGRAPH_EDITOR_URL` / `--no-editor-url` precedence chain. **No `Long` was added to any command.** Confirmed by `git diff ca015c4b -- internal/cli` showing only `root.go`, `cli_reference_test.go`, and `testdata/cli-reference-allowlist.txt` changed — no other `internal/cli/*.go` file was touched.

## `--version` Planner Addition

D-02/CONTEXT specified only `root.InitDefaultCompletionCmd()` as the tree-completion step. This plan additionally calls `root.InitDefaultVersionFlag()` in both `tools/clidoc/main.go` and the guard's `cliReferenceTree()` helper, in lockstep. **Reason:** `codegraph --version` (`-v, --version`) is a real, registered root flag (the root command's `Version` field is non-empty — see `internal/cli/root.go`'s `versionLine()`), but like the completion family it is only wired into the tree inside `Command.ExecuteC()`, never by a bare `newRootCmd()`/`NewRootCmd()` call. Without this addition, both the generated reference AND the accounting guard would silently agree on an incomplete tree missing this one real, user-facing flag — exactly the vacuous-guard shape this milestone exists to eliminate. This raised the guard's flag floor from the research-session-measured 114 to 115 (114 documented commands' flags + root's own `--version`), all 115 accounted for (114 via the reference, 1 via the allowlist for `codegraph man --help`).

## Decisions Made

- **Wrapper over rename for `NewRootCmd`** (Claude's Discretion, resolved per D-02's own steer): additive, zero call-site changes.
- **`ndocs` variable name in `docs:cli:drift`** instead of reusing `proto:drift`'s `nfiles`: purely to keep the Taskfile.yml static-shape verification (`compared ${nfiles} generated file` count == 1) sound — `rg -o` substring-matches `proto:drift`'s pre-existing "generated file**s**" line, so reusing the identical variable name would have doubled the static match count. No runtime behavior difference; the runtime transcript still literally reads `compared 1 generated file`.
- **`root.InitDefaultVersionFlag()` planner addition** — see above.
- **Kept `### SEE ALSO` blocks** in the generated reference (Claude's Discretion, per D-02: harmless in a single file) — every section links to its parent and children via in-file anchors.

## Deviations from Plan

None — plan executed exactly as written, including the one explicitly-flagged planner addition (`--version`) which the plan itself anticipated and required recording here rather than treating as an unplanned deviation.

## Issues Encountered

- Two `go build` invocations during development (`go build ./tools/clidoc/...` and `go build ./internal/cli/... ./tools/clidoc/...`) produced a stray `clidoc` binary at the repo root (default `go build` output-naming behavior for a `main` package with no `-o`). Caught by the post-build `git status --short` check before committing and removed (`rm -f clidoc`) — never staged, never committed. Documented here for traceability, not as a Rule 1/2/3 deviation (no code was wrong; the fix was a workspace hygiene step, not an implementation change).

## User Setup Required

None — no external service configuration required.

## Requirements Note

`requirements-completed` is intentionally empty in this SUMMARY's frontmatter. DOCS-05 is declared by all three plans in this phase (12-01, 12-02, 12-03) and DOCS-06 is additionally declared by 12-03 — per the shared-ID gate (#2388), neither requirement should read `Complete` in REQUIREMENTS.md until every declaring plan has finished. This plan's own contribution to both requirements is fully verified above (RED/GREEN transcripts); the orchestrator's `requirements.mark-complete` step (run once per finishing plan) will flip them to `Complete` when the last declaring plan (12-03) lands its SUMMARY.

## Next Phase Readiness

- `docs/CLI-REFERENCE.md`, `tools/clidoc`, the Taskfile targets, the ci.yml step, and the accounting guard are all in place and green.
- Ready for 12-02 (README link + brew-trust wording, DOCS-06/DOCS-07) and 12-03 (MUTATION-LOG RED families, SECURITY.md, VALIDATION.md) — both plans build on this plan's generator/guard pair without needing any further change to it.
- No blockers.

---
*Phase: 12-cli-reference-docs-tail*
*Completed: 2026-09-13*

## Self-Check: PASSED

All 7 key files confirmed present on disk (`internal/cli/root.go`, `tools/clidoc/main.go`, `docs/CLI-REFERENCE.md`, `Taskfile.yml`, `.github/workflows/ci.yml`, `internal/cli/cli_reference_test.go`, `internal/cli/testdata/cli-reference-allowlist.txt`). Both task commits (`302e5fb1`, `2a899875`) confirmed present in `git log`. `commits: 2` measured via `git rev-list --count d762d575..HEAD` against the plan-head ledger, matching the two task commits with no code changes left uncommitted.
