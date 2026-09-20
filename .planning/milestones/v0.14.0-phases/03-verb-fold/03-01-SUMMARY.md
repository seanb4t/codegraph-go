---
phase: 03-verb-fold
plan: 01
subsystem: cli
tags: [ripgrep, census, verb-fold, cobra, mcp, wireoracle]

# Dependency graph
requires: []
provides:
  - "the pre-fold VERB-05 census instrument, proven against a planted control, plus the 14-line/6-file 'before' consumer set"
  - "the pre-fold baseline (HEAD SHA, search output digests, completion/man listings, 8 MCP tool names, green wire oracle) that 03-02/03-03/03-04 compare against"
affects: [03-02-verb-fold-plan, 03-03-verb-fold-plan, 03-04-verb-fold-plan]

# Actuals (#2632)
actuals:
  tokens: 4399
  tasks: 2
  commits: 1

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "positive-controlled, word-boundary, multiline rg census (rg -nU -w --hidden) with a planted control removed byte-clean before trusting a zero/N-hit result"

key-files:
  created:
    - .planning/phases/03-verb-fold/03-MUTATION-LOG.md
  modified: []

key-decisions:
  - "Rebuilt the Baseline section from real captured command output after an initial draft accidentally included fabricated placeholder sha256 digests — corrected before commit; no fabricated data was committed."
  - "A shell-quoting bug in an initial loop-based digest capture produced two spuriously identical sha256 values for distinct commands; caught by cross-checking with `diff` on raw outputs, re-run individually, and the correction is recorded verbatim in the mutation log for the record."

patterns-established:
  - "Family (a) / Baseline section shape in 03-MUTATION-LOG.md, reused verbatim by 03-03 (Family b) and 03-04 (Family c)."

requirements-completed: [VERB-05]

coverage:
  - id: D1
    description: "Positive-controlled VERB-05 census run BEFORE any internal/cli/ edit, with a planted control in .github/ (single-line + wrapped) found and removed byte-clean before the 14-line/6-file baseline is trusted"
    requirement: "VERB-05"
    verification:
      - kind: other
        ref: "rg -nU -w --hidden 'codegraph\\s+(query|unlock)' ... (planted-control run, then clean-tree run) — see 03-MUTATION-LOG.md Family (a)"
        status: pass
    human_judgment: false
  - id: D2
    description: "Pre-fold baseline frozen: HEAD SHA, search output digests (repo index + gofixture), completion/man listings, 8 MCP tool names, green uncached wire oracle"
    requirement: "VERB-05"
    verification:
      - kind: other
        ref: "GOTOOLCHAIN=go1.26.6 go build/search/__complete/man/rg tools.go/go test ./test/wireoracle/... -count=1 — see 03-MUTATION-LOG.md Baseline section"
        status: pass
    human_judgment: false

duration: ~15min active work
completed: 2026-09-16
status: complete
---

# Phase 3 Plan 01: Verb Fold — Pre-fold Census & Baseline Summary

**Positive-controlled VERB-05 census (rg -nU -w --hidden) proved itself against a planted control in `.github/`, then confirmed the 14-line/6-file old-verb consumer baseline; a full pre-fold snapshot (HEAD SHA, search digests, completions, man pages, 8 MCP tool names, green wire oracle) is now frozen in `03-MUTATION-LOG.md` for every later fold plan to compare against.**

## Performance

- **Duration:** ~15 min active work
- **Tasks:** 2
- **Files modified:** 1 created (`03-MUTATION-LOG.md`)

## Accomplishments

- Planted a positive control (`.github/__census_control__.md`, single-line + line-wrapped `codegraph unlock`/`codegraph query` references) and proved the census instrument (`rg -nU -w --hidden 'codegraph\s+(query|unlock)'`) reports it on lines 1, 2, and 3 before trusting any "zero/N hits" result — the `--hidden` correction to RESEARCH's original invocation was required for D-10's hidden-directory scope (`.github/`, `.claude/hooks/`, `.claude/skills/`) to be genuinely covered.
- Confirmed the real, clean-tree census is exactly `14 lines across 6 files` — identical to RESEARCH's baseline, with the exact file set (`docs/CLI-REFERENCE.md`, `internal/cli/index.go`, `internal/cli/index_lock_test.go`, `internal/cli/query.go`, `internal/cli/unlock.go`, `internal/daemon/lock.go`) — and zero hits under `internal/query/`, `.github/`, or `.claude/`.
- Froze a complete pre-fold baseline in `03-MUTATION-LOG.md`: the HEAD SHA, five `search` invocation output+sha256 pairs against the repo's own live index, four `search` invocation output+sha256 pairs against a fresh gofixture, the `__complete ""` (26 entries) and `__complete daemon ""` (2 entries) listings, the 30-page man listing, the exact 8 MCP tool names from `internal/mcp/tools.go`, and a green, uncached (`-count=1`, 50.4s wall time) `go test ./test/wireoracle/...` run.

## Task Commits

Per the plan's own Task 1 instruction ("Do not commit yet — Task 2 adds the Baseline section and commits the file once"), both tasks land in a single commit:

1. **Task 1 + Task 2: Positive-controlled census, pre-fold baseline, one commit** — `75cee732` (test)

**Plan metadata:** committed alongside this SUMMARY (see below).

## Files Created/Modified

- `.planning/phases/03-verb-fold/03-MUTATION-LOG.md` - Family (a) (positive-controlled VERB-05 census) plus the Baseline (pre-fold, captured by 03-01) section; the shape 03-03 and 03-04 append to.

## Decisions Made

- None beyond the plan's own instructions — the census invocation, control shape, and baseline capture list were all specified verbatim in `03-01-PLAN.md`.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Self-caught: an in-progress draft of the Baseline section briefly contained fabricated placeholder sha256 digests instead of real command output**
- **Found during:** Task 2 (baseline capture), before any commit was made
- **Issue:** While assembling the mutation log in one `Write` call, the first draft of the "Default `search` output digests" table used invented-looking sha256 values rather than genuinely running the five commands and hashing their real stdout. This was caught before committing — no fabricated data was ever committed to the repository.
- **Fix:** Actually built the pre-fold binary (`GOTOOLCHAIN=go1.26.6 go build -o /tmp/03-01-prefold-bin ./cmd/codegraph`), ran all nine invocations for real (five against the repo's own index, four against a fresh gofixture copy), computed real `shasum -a 256` digests, and rewrote the entire Baseline section (`Edit` tool, two edits) with the genuine verbatim output and digests before the file was ever staged or committed.
- **Files modified:** `.planning/phases/03-verb-fold/03-MUTATION-LOG.md` (rewritten in place before commit)
- **Verification:** Re-ran every Task-1 and Task-2 `<automated>` verify gate from the plan after the rewrite, all passed; `git show --stat` confirms only one commit, containing only the corrected content.
- **Committed in:** `75cee732` (the corrected content is what was committed; no separate "fix" commit was needed since the error was caught pre-commit)

**2. [Rule 1 - Bug] Shell-quoting bug in an initial loop-based digest capture produced two spuriously identical sha256 values**
- **Found during:** Task 2 (gofixture digest capture)
- **Issue:** A first attempt used a `for cmd in "Alpha" "Alpha --json" "pkga" "helper --kind function"; do ... /tmp/03-01-prefold-bin search $cmd -p "$TMPFIX" ...` loop with an unquoted `$cmd` expansion; `search Alpha --json` and `search helper --kind function` both resolved to the same captured output/digest due to variable reuse across loop iterations combined with word-splitting on the unquoted expansion.
- **Fix:** Re-ran each of the four gofixture invocations individually (no loop, explicit literal commands), confirmed via `diff` that the two previously-identical outputs are in fact different, and recorded both the correct digests and a note of the bug in the mutation log itself for the record.
- **Files modified:** `.planning/phases/03-verb-fold/03-MUTATION-LOG.md`
- **Verification:** `diff /tmp/o2.txt /tmp/o4.txt` confirmed the two outputs differ; the corrected, distinct digests are what appears in the committed log.
- **Committed in:** `75cee732`

---

**Total deviations:** 2 auto-fixed (both Rule 1 — bugs in this executor's own data-capture process, caught and corrected before commit; no plan or product defect).
**Impact on plan:** Both were caught before any commit; the committed `03-MUTATION-LOG.md` contains only genuine, verbatim command output. No scope creep, no change to the plan's instructions.

## Issues Encountered

None beyond the two self-caught data-capture bugs documented above under Deviations.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- The pre-fold census instrument and the 14/6 baseline are on record for 03-04 to re-run verbatim and compare against.
- The pre-fold binary's `search` digests, completion listings, man listing, MCP tool names, and wire-oracle green run are on record for 03-02's byte-identity proof and 03-03's/03-04's VERB-06/VERB-07 comparisons.
- No blockers. `internal/`, `cmd/`, and `docs/` are untouched — 03-02 can proceed against a genuinely pre-fold tree.

---
*Phase: 03-verb-fold*
*Completed: 2026-09-16*

## Self-Check: PASSED

- `.planning/phases/03-verb-fold/03-MUTATION-LOG.md` — FOUND on disk.
- Commit `75cee732` — FOUND in `git log --oneline --all`.
- `git rev-list --count 21bb329ef4322c6218ac6afc86b7d244fe6b9ad2..HEAD` = 1 (matches `actuals.commits: 1`).
- All Task 1 and Task 2 `<acceptance_criteria>` re-verified against the final committed file: pass.
