---
phase: 05-language-coverage-resolution-breadth
plan: 08
subsystem: parsing
tags: [tree-sitter, cgo, rust, ruby, php, c, cpp, swift, kotlin, supply-chain, mainstream-tier]

# Dependency graph
requires:
  - phase: 05-language-coverage-resolution-breadth
    plan: 03
    provides: LanguageSpec registry seam and newCGoParser/Parse pattern that this plan's seven constructors route through
provides:
  - "cgo.NewRustParser, NewRubyParser, NewPHPParser, NewCParser, NewCppParser, NewSwiftParser, NewKotlinParser — all seven mainstream-tier grammar constructors, each a one-liner routing through newCGoParser + MaxSourceBytes"
  - "go.mod pins: five tree-sitter-org grammars at exact semver (tree-sitter-rust@v0.24.2, tree-sitter-ruby@v0.23.1, tree-sitter-php@v0.24.2, tree-sitter-c@v0.24.2, tree-sitter-cpp@v0.23.4) plus two re-approved [SUS] community grammars pinned by exact commit/semver (alex-pinkus/tree-sitter-swift@v0.0.0-20260601004120-31d17fe7e818, tree-sitter-grammars/tree-sitter-kotlin@v1.1.0)"
  - "TestCGoParsesMainstreamSources — table-driven test proving all seven mainstream constructors parse valid source without error"
affects: [05-10, 05-11 (mainstream extractor plans consume these seven constructors), docs/LANGUAGE-CAPABILITY-MATRIX.md D-11 row for Rust/Ruby/PHP/C/C++/Swift/Kotlin]

# Tech tracking
tech-stack:
  added:
    - "github.com/tree-sitter/tree-sitter-rust@v0.24.2"
    - "github.com/tree-sitter/tree-sitter-ruby@v0.23.1"
    - "github.com/tree-sitter/tree-sitter-php@v0.24.2"
    - "github.com/tree-sitter/tree-sitter-c@v0.24.2"
    - "github.com/tree-sitter/tree-sitter-cpp@v0.23.4"
    - "github.com/alex-pinkus/tree-sitter-swift@v0.0.0-20260601004120-31d17fe7e818"
    - "github.com/tree-sitter-grammars/tree-sitter-kotlin@v1.1.0"
  patterns:
    - "Mainstream-tier constructors follow the identical one-liner newCGoParser(tree_sitter_<lang>.Language()) shape as every prior grammar — no new abstraction introduced for the two [SUS] community grammars."
    - "Blocking human-verify checkpoint (T-05-SC) as the supply-chain gate before any go get of a non-tree-sitter-org grammar; when an approved commit turns out not to build, re-run the checkpoint for a revised source rather than silently substituting — this plan's Swift pin used the same maintainer's alternate 'with-generated-files' lineage, and Kotlin's approved source was swapped from an individual-maintainer fork (fwcd) to a proper org (tree-sitter-grammars) root-module semver tag."

key-files:
  created: []
  modified:
    - internal/parser/cgo/parser_cgo.go
    - internal/parser/cgo/parser_cgo_test.go
    - go.mod
    - go.sum

key-decisions:
  - "Swift pinned at github.com/alex-pinkus/tree-sitter-swift@v0.0.0-20260601004120-31d17fe7e818 (commit 31d17fe7e818, 2026-06-01) — the 'with-generated-files' lineage that ships a generated src/parser.c, required because CGo needs the pre-generated C parser, not just the grammar.js source. The originally-approved commit lacked this generated file and failed to build."
  - "Kotlin pinned at github.com/tree-sitter-grammars/tree-sitter-kotlin@v1.1.0 (proper semver tag, community tree-sitter-grammars org, root module) — replaces the originally-approved fwcd/tree-sitter-kotlin/bindings/go source, which also failed to build. This is a supply-chain quality IMPROVEMENT over the original approval: a real org (not an individual maintainer) and a proper semver release (not a bare commit pseudo-version), and its own go.mod requires go-tree-sitter v0.24.0 (MVS-compatible with this project's pinned v0.25.0)."
  - "Both revisions were re-submitted for and received explicit human approval before pinning — no silent source substitution occurred (Rule 4 boundary respected: this was an architectural/supply-chain decision requiring a human, not an auto-fixable blocker)."

patterns-established:
  - "When an approved supply-chain pin fails to build, treat it as a NEW checkpoint (not an auto-fix): find a genuinely buildable alternative, get explicit re-approval on the exact revised commit/version, then proceed. Never fall back to @latest or a same-repo alternate branch without re-approval."

requirements-completed: [LANG-06]

coverage:
  - id: D1
    description: "All seven mainstream-tier grammar constructors (Rust, Ruby, PHP, C, C++, Swift, Kotlin) construct through the existing newCGoParser/Parse seam with MaxSourceBytes enforced, each returning a non-nil parser for valid source"
    requirement: "LANG-06"
    verification:
      - kind: unit
        ref: "internal/parser/cgo/parser_cgo_test.go#TestCGoParsesMainstreamSources"
        status: pass
    human_judgment: false
  - id: D2
    description: "The five tree-sitter-org grammars are pinned at VERIFIED exact semver; the two [SUS] community grammars (Swift, Kotlin) are pinned only after explicit human-verify approval, by exact commit/semver, never @latest"
    requirement: "LANG-06"
    verification:
      - kind: other
        ref: "grep -n '@latest|@main' go.mod go.sum internal/parser/cgo/parser_cgo.go (no floating pins found); grep confirms exact pins alex-pinkus/tree-sitter-swift@v0.0.0-20260601004120-31d17fe7e818 and tree-sitter-grammars/tree-sitter-kotlin@v1.1.0 in go.mod"
        status: pass
    human_judgment: false

duration: 25min
completed: 2026-07-12
status: complete
---

# Phase 5 Plan 08: Mainstream Grammar Layer (LANG-06) Summary

**Seven mainstream-tier CGo parser constructors (Rust, Ruby, PHP, C, C++, Swift, Kotlin) wired through the existing MaxSourceBytes-guarded parser seam, with the two [SUS] community grammars (Swift, Kotlin) pinned only after a blocking human-verify supply-chain checkpoint — including a mid-execution re-approval when the originally-approved commits turned out not to build.**

## Performance

- **Duration:** ~25 min (Task 3 continuation session)
- **Completed:** 2026-07-12
- **Tasks:** 3 (1 auto, 1 checkpoint, 1 auto)
- **Files modified:** 4 (go.mod, go.sum, internal/parser/cgo/parser_cgo.go, internal/parser/cgo/parser_cgo_test.go)

## Accomplishments
- `NewRustParser`, `NewRubyParser`, `NewPHPParser`, `NewCParser`, `NewCppParser` (Task 1, prior session) — five tree-sitter-org grammars pinned at exact semver, PHP using the `php/src` (`LanguagePHP`) accessor
- Blocking human-verify checkpoint for the two `[SUS]` community grammars (Task 2, prior session) — approved
- `NewSwiftParser`, `NewKotlinParser` (Task 3, this session) — the two community grammars, pinned at **revised, re-approved** sources after the originally-approved commits failed to build upstream:
  - Swift: `github.com/alex-pinkus/tree-sitter-swift@v0.0.0-20260601004120-31d17fe7e818` — the "with-generated-files" lineage that ships the generated `src/parser.c` CGo needs (20,630,054 bytes, 615,415 lines, confirmed present on disk)
  - Kotlin: `github.com/tree-sitter-grammars/tree-sitter-kotlin@v1.1.0` — a proper semver-tagged root module from a genuine community org (`tree-sitter-grammars`), replacing the originally-approved individual-maintainer `fwcd` fork; its own `go.mod` requires `go-tree-sitter v0.24.0`, MVS-compatible with this project's pinned `v0.25.0`
- All seven constructors verified to compile and each returns a non-nil parser for valid source via a new `TestCGoParsesMainstreamSources` table-driven test (extending the existing `TestCGoParsesPriority4Sources` pattern) — this also backfilled test coverage for Task 1's five constructors, which had none
- `go build ./...`, `go vet ./...`, and `go test ./internal/parser/... -count=1` all pass

## Task Commits

1. **Task 1: tree-sitter-org mainstream constructors + pins (Rust, Ruby, PHP, C, C++)** - `0638bf4` (feat, prior session)
2. **Task 2: Human-verify gate for [SUS] Swift/Kotlin grammar pins** - checkpoint, approved (prior session)
3. **Task 3: Swift + Kotlin constructors + pins (post-approval, revised sources)** - `6e38b93` (feat, this session)

**Plan metadata:** this SUMMARY's own commit closes out the plan.

## Files Created/Modified
- `internal/parser/cgo/parser_cgo.go` - added `NewSwiftParser`/`NewKotlinParser` (one-liner `newCGoParser(tree_sitter_<lang>.Language())` shape) and their imports
- `internal/parser/cgo/parser_cgo_test.go` - added `TestCGoParsesMainstreamSources`, a table-driven test covering all seven mainstream-tier constructors (Rust, Ruby, PHP, C, C++, Swift, Kotlin)
- `go.mod` - pinned `alex-pinkus/tree-sitter-swift@v0.0.0-20260601004120-31d17fe7e818` and `tree-sitter-grammars/tree-sitter-kotlin@v1.1.0` as direct requires (promoted from indirect, no `go mod tidy` per project convention)
- `go.sum` - checksums for both new modules

## Decisions Made
See `key-decisions` in frontmatter for the full list. Highlights:
- The originally human-approved Swift/Kotlin commits from the prior session did NOT build (upstream packaging defects — Swift's approved commit lacked the generated `parser.c`; Kotlin's `fwcd` source had an unbuildable module layout). Rather than silently substituting a different source, both revised pins were surfaced back to the human for explicit re-approval before this session pinned them.
- Kotlin's revision is a net supply-chain IMPROVEMENT over the original approval: a genuine community org (`tree-sitter-grammars`) with a proper semver tag, versus an individual maintainer's unversioned `bindings/go` subpath.

## Deviations from Plan

None beyond what the continuation prompt already specified — this session executed exactly the constraints handed off by the prior (failed) Task 3 attempt: discard the broken uncommitted state, pin the human-approved revised sources, add the two constructors, verify, commit.

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] Added test coverage for all seven mainstream constructors**
- **Found during:** Task 3 (Swift/Kotlin constructors)
- **Issue:** Task 1's five constructors (Rust, Ruby, PHP, C, C++) had no dedicated test coverage — only verified via `go build`. The plan's own Task 3 constraints called for extending the test pattern "if it exists" (it does, `TestCGoParsesPriority4Sources`).
- **Fix:** Added `TestCGoParsesMainstreamSources`, a table-driven test covering all seven mainstream-tier constructors (backfilling the five from Task 1, adding the two from Task 3), each parsing a trivial valid source snippet and asserting a non-nil tree.
- **Files modified:** internal/parser/cgo/parser_cgo_test.go
- **Verification:** `go test ./internal/parser/... -run TestCGoParsesMainstreamSources -v` — all 7 subtests pass
- **Committed in:** `6e38b93` (Task 3 commit)

---

**Total deviations:** 1 auto-fixed (1 missing test coverage). 0 scope-creep.
**Impact on plan:** The test-coverage addition strengthens the plan's own acceptance criteria ("each returns a non-nil parser + nil error") with an actual automated check rather than relying solely on `go build` succeeding. No production behavior changed.

## Issues Encountered

- **Originally-approved Swift/Kotlin commits failed to build upstream** (discovered in a prior Task 3 attempt, not this session): the originally-approved Swift commit did not ship a generated `src/parser.c`, and the originally-approved `fwcd/tree-sitter-kotlin/bindings/go` module had a packaging defect preventing CGo compilation. That attempt left broken uncommitted changes in `go.mod`/`go.sum`/`parser_cgo.go` (a `replace` directive, a floating Swift main-branch pin, a Kotlin `fwcd` require). This session's first action was `git checkout -- go.mod go.sum internal/parser/cgo/parser_cgo.go` to discard those uncommitted changes and confirm a clean rebuild from the Task 1 commit (`0638bf4`) before proceeding — verified via `go build ./...` green. The revised, re-approved sources (documented above) build cleanly with no `replace` directive needed.
- **Benign compiler warning on Swift's scanner.c:** `warning: 'TOKEN_COUNT' macro redefined [-Wmacro-redefined]` — Swift's `scanner.c` redefines a macro also defined in the generated `parser.c`. This is a non-fatal warning from the third-party grammar's own C source (not our code), does not affect build exit status (0) or test results, and is out of scope to fix (would require patching the pinned grammar's C source).

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- All seven mainstream-tier constructors (`cgo.NewRustParser`, `NewRubyParser`, `NewPHPParser`, `NewCParser`, `NewCppParser`, `NewSwiftParser`, `NewKotlinParser`) are available and proven through `newCGoParser`/`Parse`, ready for the mainstream extractor plans (05-10, 05-11) to consume without any further `parser_cgo.go`/`go.mod` changes — this plan was explicitly the file-disjoint enabler for those plans to run in parallel.
- `docs/LANGUAGE-CAPABILITY-MATRIX.md`'s D-11 row for Rust/Ruby/PHP/C/C++/Swift/Kotlin should record: (1) all seven grammars carry the shared `MaxSourceBytes`/in-process-crash-risk caveat (T-05-DoS, Pitfall 4, accepted Phase-1 contract); (2) Swift and Kotlin are `[SUS]`-tier supply-chain grammars pinned by exact commit/semver after human-verify (T-05-SC), with Kotlin's final source differing from the RESEARCH document's original `fwcd` recommendation.
- `go build ./...`, `go vet ./...`, and `go test ./internal/parser/... -count=1` all pass with all seven mainstream grammars compiled in.

---
*Phase: 05-language-coverage-resolution-breadth*
*Completed: 2026-07-12*

## Self-Check: PASSED

All modified files confirmed present on disk; commit `6e38b93` confirmed in git log; commit `0638bf4` (Task 1) confirmed intact and unmodified.
