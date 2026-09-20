---
phase: 04-cli-glow-up
plan: 02
subsystem: cli
tags: [fang, cobra, colorprofile, archtest, spike, verdict, go-modules]

# Dependency graph
requires:
  - phase: 04-01
    provides: "TestPlainGolden/TestNoColorNonTTYRegression/TestShortFlagsConsistent — the live verify gate this plan's evidence runs re-confirmed stayed green"
provides:
  - "04-FANG-VERDICT.md: fang/v2 v2.0.1 spiked against all four conjunctive D-01...D-04 criteria, with evidence transcripts for each; verdict is declined"
  - "internal/cli/present/archtest/import_graph_test.go widened to prefix-match both charm vanity roots (charm.land/, github.com/charmbracelet/) instead of an exact-match 3-literal list"
  - "go.mod: github.com/charmbracelet/colorprofile promoted from // indirect to the direct require block"
  - "04-MUTATION-LOG.md Family (c): the widened archtest proven RED against a planted colorprofile import"
affects: [04-03, 04-04, 04-05, 04-06, 04-07, 04-08]

# Actuals (#2632)
actuals:
  tokens: 5844
  tasks: 2
  commits: 2

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Spike-in-working-tree, evidence-then-revert: the fang wrap was applied directly to root.go/main.go and go.mod/go.sum, exercised against real test suites and a real built binary, then git checkout -- reverted byte-clean before any commit — no throwaway harness, no simulated Execute() call"
    - "Prefix-match denylist widening landed in the SAME commit as the go.mod promotion it exists to guard, proven RED per newly-required module via an untracked planted-import file (Family (c)), matching the 02-/03-MUTATION-LOG.md shape"

key-files:
  created:
    - .planning/phases/04-cli-glow-up/04-FANG-VERDICT.md
  modified:
    - go.mod
    - internal/cli/present/archtest/import_graph_test.go
    - .planning/phases/04-cli-glow-up/04-MUTATION-LOG.md

key-decisions:
  - "fang/v2 v2.0.1 is DECLINED for cli.Execute(). Root cause (confirmed by running the real binary and the real integration test, not by reading docs): fang.Execute always calls DefaultErrorHandler with w = colorprofile.NewWriter(root.ErrOrStderr(), os.Environ()); *colorprofile.Writer has no Fd() method, so DefaultErrorHandler's own w.(term.File) TTY-detection type assertion always fails, and the non-TTY plain-print branch (fmt.Fprintln(w, err.Error())) is structurally unreachable through fang's own wrapping. Every error — on a TTY or a pipe alike — renders the styled box. This breaks D-03's exact-once plain-stderr contract for the query/unlock stubs (TestRenamedStubsPrintExactlyOnce FAILED under the wrap, real binary) and, by the same mechanism, every other command's error output. D-01 (wireoracle), D-02 (govulncheck/TestCharmCgoClosure), and D-04 composition (completion bash/zsh/fish, man, --help, --version) all PASSED in isolation, but the four criteria are conjunctive — one FAIL declines the whole candidate."
  - "Help is hand-rolled per D-14 in a later plan (04-05/04-06 per the phase sequence) — root.go/main.go/go.mod carry NO fang trace; this was verified directly (rg 'charm.land/fang' go.mod matches nothing at HEAD)."
  - "Family (d) (the fang-import RED proof D-15 requires 'if adopted') does not apply this plan — recorded explicitly in 04-MUTATION-LOG.md rather than silently omitted, so a future reader does not mistake its absence for an oversight."

patterns-established:
  - "A spike that edits go.mod/go.sum plus two source files can be fully evidenced (build, real binary, real test suites) and then reverted byte-clean via a single git checkout -- across all four paths, with the verdict document as the only artifact that survives the spike."

requirements-completed: [CLI-08, GRD-13]

coverage:
  - id: D1
    description: "fang/v2 v2.0.1 spiked against all four conjunctive D-01...D-04 criteria with evidence-carrying transcripts for each; verdict recorded as declined and committed alone before any renderer/resolver/palette/help template"
    requirement: "CLI-08"
    verification:
      - kind: other
        ref: ".planning/phases/04-cli-glow-up/04-FANG-VERDICT.md"
        status: pass
      - kind: integration
        ref: "test/integration#TestRenamedStubsPrintExactlyOnce (real binary, run against the spiked wrap — FAILED, which is the evidence the verdict is built on)"
        status: pass
    human_judgment: false
  - id: D2
    description: "present archtest's forbiddenImportPaths replaced by forbiddenImportPathPrefixes, prefix-matching both charm.land/ and github.com/charmbracelet/, reporting every hit; self-defeat probe and charm_cgo_test.go's separate, narrower guard both unchanged"
    requirement: "GRD-13"
    verification:
      - kind: unit
        ref: "internal/cli/present/archtest/import_graph_test.go#TestNoCharmInServeReachablePackages"
        status: pass
    human_judgment: false
  - id: D3
    description: "go.mod: github.com/charmbracelet/colorprofile promoted from // indirect to direct, in the same commit as the archtest prefix widening"
    verification:
      - kind: other
        ref: "shell: rg '^\\s+github.com/charmbracelet/colorprofile v0.4.3$' go.mod (no // indirect suffix)"
        status: pass
    human_judgment: false
  - id: D4
    description: "Family (c): a planted colorprofile import in an untracked internal/query file demonstrated the widened guard RED, naming both the package and the import, then reverted byte-clean"
    verification:
      - kind: other
        ref: ".planning/phases/04-cli-glow-up/04-MUTATION-LOG.md#Family-(c)"
        status: pass
    human_judgment: false

# Metrics
duration: ~20min
completed: 2026-09-17
status: complete
---

# Phase 4 Plan 2: Fang Spike, Verdict, and the D-15 Denylist Commit Summary

**Fang/v2 v2.0.1 spiked against all four conjunctive criteria and declined — `fang.Execute`'s error handler can never satisfy its own TTY-detection check when wrapped in a `colorprofile.Writer`, so it always double-renders the styled box and breaks the stub's exact-once stderr contract — then the `present` archtest widened to prefix-match both charm vanity roots in the same commit as promoting `colorprofile` to a direct dependency, proven RED against a planted import.**

## Performance

- **Duration:** ~20 min
- **Started:** 2026-09-17T20:41:00Z
- **Completed:** 2026-09-17T21:01:00Z
- **Tasks:** 2
- **Files modified:** 4 (1 created, 3 modified)

## Accomplishments

- Ran the fang spike directly in the working tree — real `go get`, real edits to `root.go`/`main.go`, a real built binary — against all four D-01…D-04 criteria, with every evidence run's transcript pasted verbatim into `04-FANG-VERDICT.md`.
- Found and documented the exact root cause of a real fang defect via execution, not documentation: `*colorprofile.Writer` (what `fang.Execute` always wraps stderr in) has no `Fd()` method, so `DefaultErrorHandler`'s `w.(term.File)` check always fails, making the "plain non-TTY" branch unreachable — every error renders the styled box regardless of TTY state.
- Confirmed this breaks D-03 empirically: `TestRenamedStubsPrintExactlyOnce` (the real-binary stub-contract test) FAILED under the fang wrap, with the styled-box stderr shown verbatim in both the verdict and the mutation log.
- Confirmed D-01 (wireoracle, 38 frozen transcripts), D-02 (zero new govulncheck findings, `TestCharmCgoClosure` green), and D-04 composition (bash/zsh/fish completions, `man`, `--help`, `--version` identity) all PASS in isolation — the verdict names D-03 as the sole disqualifying criterion.
- Reverted the entire spike (`go.mod`, `go.sum`, `root.go`, `main.go`) byte-clean via `git checkout --` before committing anything; committed `04-FANG-VERDICT.md` alone as `docs(04-02)`.
- Widened `internal/cli/present/archtest/import_graph_test.go`'s `forbiddenImportPaths` to `forbiddenImportPathPrefixes`, prefix-matching both `charm.land/` and `github.com/charmbracelet/` (colorprofile/x-ansi live under the latter, which the old exact-match list could never catch) and reporting every hit rather than the first.
- Promoted `github.com/charmbracelet/colorprofile` to a direct `go.mod` requirement in the same commit as the denylist widening, per D-15's commit-discipline requirement.
- Proved the widened guard RED (Family (c)) against a planted, untracked `colorprofile` import in `internal/query`, naming both the offending package and the exact import; removed the plant and confirmed `internal/query` byte-clean and the guard green again.

## Task Commits

1. **Task 1: Run the fang spike, record 04-FANG-VERDICT.md, commit the verdict alone** — `d722804a`
2. **Task 2: The D-15 commit — prefix denylist + colorprofile promotion** — `53205c24`

**Plan metadata:** committed alongside SUMMARY.md/STATE.md/ROADMAP.md/REQUIREMENTS.md in the metadata commit that follows this SUMMARY.

## Files Created/Modified

- `.planning/phases/04-cli-glow-up/04-FANG-VERDICT.md` — the four-criterion evidence-carrying verdict (declined), shipped call set, information section, Key Decisions row
- `internal/cli/present/archtest/import_graph_test.go` — `forbiddenImportPathPrefixes`, `strings.HasPrefix` walk, updated doc comments; `charmImporterProbePath`/`assertCharmImporterExists` unchanged
- `go.mod` — `github.com/charmbracelet/colorprofile` moved to the direct require block
- `.planning/phases/04-cli-glow-up/04-MUTATION-LOG.md` — Family (c) added, Summary table extended, Family (d) explicitly marked not-applicable

## Decisions Made

See `key-decisions` in frontmatter for the full rationale. Summary: fang declined on D-03 (the `colorprofile.Writer`/`term.File` TTY-detection mismatch makes fang's plain-stderr branch unreachable); help stays hand-rolled per D-14 in a later plan; Family (d) is recorded as explicitly not-applicable rather than silently skipped.

## Deviations from Plan

None — plan executed exactly as written. The spike's outcome (declined) is one of the two designed branches the plan itself anticipates (D-04: "if adopted... otherwise help is hand-rolled"), not a deviation from it.

## Issues Encountered

None beyond the fang defect itself, which is the spike's own subject matter, not an execution problem — it was investigated to root cause (source-level, both `fang.go` and `colorprofile/writer.go` read directly) and documented, not worked around.

## Authentication Gates

None.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- The fang verdict is settled: `internal/cli/root.go`/`cmd/codegraph/main.go`/`go.mod` carry no fang trace (`rg 'charm.land/fang' go.mod internal/cli/root.go cmd/codegraph/main.go` matches nothing at HEAD). Later plans building `--color`/palette/help must plan for D-14's hand-rolled `SetHelpFunc` path, not a fang wrap.
- The `present` archtest now catches every charm-family import path added this milestone by prefix, not just the original three literals — `colorprofile`'s promotion to direct is safe and guarded.
- `charm_cgo_test.go` is untouched (byte-identical diff confirmed against the pre-plan commit) — its narrower `charm.land`-only CGo-closure guard is a separate concern D-02 only required to stay green.
- No blockers for 04-03 (resolver/`--color` per the hard sequence in `04-CONTEXT.md`).

---
*Phase: 04-cli-glow-up*
*Completed: 2026-09-17*

## Self-Check: PASSED

- FOUND: .planning/phases/04-cli-glow-up/04-FANG-VERDICT.md
- FOUND: internal/cli/present/archtest/import_graph_test.go (forbiddenImportPathPrefixes present)
- FOUND: commit d722804a (git log --oneline --all)
- FOUND: commit 53205c24 (git log --oneline --all)
- Re-ran plan `<verification>`: `GOTOOLCHAIN=go1.26.6 go test ./internal/cli/present/archtest/... -count=1` — ok; `GOTOOLCHAIN=go1.26.6 go build ./... && go test ./internal/cli/... -count=1` — ok (5 packages, including TestPlainGolden/TestNoColorNonTTYRegression/TestShortFlagsConsistent from 04-01, all green).
- `git status --porcelain` — empty.
- Adopted-branch-only checks (`TestRenamedStubsPrintExactlyOnce`, `test/wireoracle` at HEAD) intentionally skipped — verdict is declined, so no fang code exists at HEAD for them to cover.
