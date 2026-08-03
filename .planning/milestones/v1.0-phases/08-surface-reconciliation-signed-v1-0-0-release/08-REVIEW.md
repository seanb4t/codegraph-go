---
phase: 08-surface-reconciliation-signed-v1-0-0-release
reviewed: 2026-07-26T00:00:00Z
depth: deep
iteration: 3
files_reviewed: 5
files_reviewed_list:
  - internal/cli/affected.go
  - internal/cli/affected_test.go
  - internal/query/traverse.go
  - internal/query/traverse_test.go
  - internal/query/validate.go
findings:
  critical: 0
  warning: 0
  info: 3
  total: 3
status: issues_found
release_gating: none
---

# Phase 8: Code Review Report (iteration 3 — re-review)

**Reviewed:** 2026-07-26
**Depth:** deep
**Files Reviewed:** 5
**Status:** issues_found (0 critical, 0 warning, 3 info — **none gate v1.0.0**)

## Summary

**Bottom line: all three iteration-2 fixes are verified correct, complete, and
non-vacuously tested. Zero blockers, zero warnings. This loop is safe to close.**

The three `info` findings below are comment/message polish in the same code —
two of them are stale prose that *actively misdescribes* the very `bufio`
semantics that produced the two-iteration no-op. They are worth fixing, but none
of them changes behavior and none should block the signed release.

Unlike iterations 1 and 2, every claim below was proved empirically by
**reverting the fix in an isolated `git worktree` and observing its test fail**,
not by reading the diff. The main working tree was never modified (verified
below).

### Fix 1 — CR-01 (`333ab82`, `internal/cli/affected.go`): VERIFIED CORRECT

`scanner.Buffer(make([]byte, 0, affectedStdinMaxLineBytes), affectedStdinMaxLineBytes)`
now makes `max` the binding constraint. `bufio.Scanner.Buffer` assigns
`s.buf = buf[0:cap(buf)]` and errors `ErrTooLong` when the buffer is full and
`len(s.buf) >= s.maxTokenSize`; with `cap(buf) == max == 4096` the two agree, so
the ceiling is genuinely 4096 rather than the old effective 65535.

- **Cap genuinely enforced:** proved by a table test in the scratch worktree —
  a 4095-byte line is accepted, a 4096-byte line and a 4097-byte line are both
  rejected. (`bufio`'s documented contract is "token size must be less than
  max", so 4095 is the true maximum accepted line; the existing test's comment
  already says this correctly.)
- **`bufio.ErrTooLong` path reachable and correct:** observed live —
  `affected: --stdin line exceeds maximum 4096 bytes: bufio.Scanner: token too long`.
  It is surfaced as a hard error, so an over-cap line does *not* silently drop
  every subsequent valid line (confirmed with a 3-line input where the over-cap
  line sat between two valid paths).
- **No legitimately-long path wrongly rejected:** 4095 bytes is exactly the
  maximum usable path length under a `PATH_MAX` of 4096 (which counts the NUL).
  The boundary is right, not merely close.
- **`TestAffectedStdinLineTooLong` is NOT vacuous:** reverting only the initial
  capacity back to `64*1024` makes the `line exceeding the cap by one byte
  errors` subtest fail with `expected error for 4097-byte line (max 4096), got nil`.

### Fix 2 — WR-01 (`331a5a6`, `internal/cli/affected.go`): VERIFIED CORRECT

`query.ValidateAffectedFiles(len(files) + 1)` is semantically **identical** to
the inline `len(files)+1 > query.MaxAffectedFiles` it replaced — the validator's
body is `if n > MaxAffectedFiles`, the same strict `>` on the same argument.

- **Off-by-one identical (the highest-risk item):** proved two ways. (a) Direct
  unit assertions: `ValidateAffectedFiles(10000) == nil`,
  `ValidateAffectedFiles(10001) != nil`. (b) End-to-end: piping exactly
  `MaxAffectedFiles` (10000) distinct paths through `affected --stdin`
  **succeeds**, and 10001 fails. Nothing regressed to `>=`.
- **Cross-check against the old code:** restoring the *old inline check* in the
  scratch worktree leaves `TestAffectedStdinTooManyFiles` **passing**. That is
  the desired result here — it confirms the two forms are behaviorally
  indistinguishable (this fix is a pure de-duplication, not a behavior change).
  The test is still non-vacuous with respect to the bound itself, which is what
  it exists to pin.
- **`%w` chain preserved:** `fmt.Errorf("affected: %w", err)` wraps, not
  formats. (No sentinel exists to unwrap to today, so this is forward-looking
  hygiene only.)
- **No double validation:** `query.ValidateAffectedFiles` has exactly one call
  site repo-wide (`internal/cli/affected.go:200`), and `Engine.Affected` has
  exactly one caller (`internal/cli/affected.go:73`) and does not re-check the
  bound. There is no MCP `affected` tool, so no second ingestion path exists.

### Fix 3 — WR-02 (`4feb6ff`, `internal/query/traverse.go`): VERIFIED CORRECT

`sortLocations(locs)` at `traverse.go:331` precedes **every** truncation path.

- **Placement genuinely precedes all truncation:** `Callees` (lines 290–339) has
  no early return that yields a result — every pre-sort `return` is an error
  return (`validateLimit`, `resolveSymbolNode`, `IterateEdges`, non-`ErrNotFound`
  `GetNode`, `it.Err()`). Both caps (`limit` at 332, `MaxLimit` at 335) sit
  strictly after the sort, so the capped subset is a deterministic "first N in
  sorted order".
- **Ordering matches its siblings exactly:** all four call the *same*
  `sortLocations` helper — identical `sort.SliceStable` comparator, identical
  `(FilePath, Name, StartLine)` field precedence. `Callers` (403) and `Callees`
  (331) now have byte-identical sort/cap sequencing; `Impact` (494) and
  `Affected` (639) sort with no cap to follow.
- **`TestCalleesSortedDeterministically` is NOT vacuous:** removing
  `sortLocations(locs)` from `Callees` makes it fail —
  `Callees[0].Name: got "Zeta", want "Alpha"`. `traverseFakeReader.IterateEdges`
  filters by source without reordering, so the fake genuinely preserves the
  deliberately-unsorted insertion order.
  - Note: the *other* new test — the `Callees` subtest in
    `TestImpactCallersAffectedDeterministicAcrossRepeatedCalls` — still passes
    without the fix. That is expected and not a defect: repeat-call stability
    and sortedness are different properties, and the dedicated test above pins
    the one this fix delivers.
- **No performance cliff:** sorting now runs over the full pre-cap result rather
  than the capped subset, but the set is one symbol's outbound `calls` edges —
  small and bounded — and this is the identical shape already shipped in
  `Callers`/`Impact`/`Affected` in iteration 1. Perf is also explicitly out of
  v1 review scope, and no correctness consequence exists.
- **No downstream ordering assumption broken:** the two consumers
  (`internal/cli/callees.go:36`, `internal/mcp/tools.go:306`) render the slice
  in place — neither re-sorts nor applies a second cap. Golden parity is
  unaffected because `golden_parity_test.go` compares `Callees` as a *set*
  (`toLocSet` + `assertSubset`), and the corpus result (2 entries) is under the
  test's `limit` of 5 anyway.

### Commands run and results

| Command | Result |
|---|---|
| `go build ./...` | clean |
| `go vet ./internal/cli ./internal/query` | clean |
| `go test ./internal/query/...` | `ok ... 5.013s` |
| `go test ./internal/cli/... -run 'TestAffected' -v` | PASS (all 8 tests, incl. `TestAffectedStdinLineTooLong` both subtests, `TestAffectedStdinTooManyFiles`) |
| `go test ./testdata/golden/...` | `ok ... 30.702s` (golden parity green, no ordering regression) |
| `go test $(go list ./... \| grep -v '/internal/daemon')` | exit 0, **entire suite green** |
| Revert CR-01 (`64*1024` cap) in scratch worktree → `go test -run TestAffectedStdinLineTooLong` | **FAIL** — `expected error for 4097-byte line (max 4096), got nil` (fix non-vacuous) |
| Revert WR-01 (old inline bound) in scratch worktree → `go test -run TestAffectedStdinTooManyFiles` | **PASS** — confirms identical `>` semantics, no off-by-one drift |
| Scratch boundary probe: `ValidateAffectedFiles(10000)` / `(10001)` and end-to-end 10000-path stdin | 10000 accepted, 10001 rejected — boundary exact |
| Scratch boundary probe: stdin lines of 4095 / 4096 / 4097 bytes | 4095 accepted; 4096 and 4097 rejected — cap exact |
| Revert WR-02 (`sortLocations` removed from `Callees`) → `go test -run TestCalleesSortedDeterministically` | **FAIL** — `Callees[0].Name: got "Zeta", want "Alpha"` (fix non-vacuous) |
| `git worktree remove --force <scratch>` then `git status --short` + `git diff HEAD --stat` | scratch removed; **working tree unmodified** (only the pre-existing untracked `08-REVIEW*.iter3.md` and `bin/`) |

`internal/daemon`'s four known load-sensitive flaky tests were excluded per
instruction and are untouched by Phase 8. gopls/IDE diagnostics were ignored per
instruction; the compiler is the authority and it is clean.

## Critical Issues

None.

## Warnings

None.

## Info

### IN-01: Two doc comments in `affected.go` still describe the pre-fix (and factually inverted) `bufio` semantics

**File:** `internal/cli/affected.go:159-165`, `internal/cli/affected.go:186-191`

**Issue:** The prose that surrounds the CR-01 fix now contradicts the code, in
the exact direction that caused the original two-iteration no-op.

1. Line 163-164 calls `bufio.MaxScanTokenSize` a *"much smaller default 64KB ...
   ceiling"* than `affectedStdinMaxLineBytes`. `bufio.MaxScanTokenSize` is
   65536 — **16x larger** than 4096, not smaller. The stated rationale ("so a
   pathological line is an explicit error rather than tripping the smaller
   default") is backwards: the constant exists to *tighten* the default, not to
   escape it.
2. Line 186-187 says *"scanner.Buffer **raises** the per-line token ceiling to
   `affectedStdinMaxLineBytes`"*. Post-fix it **lowers** it, from 65536 to 4096.
   A maintainer trusting this sentence could "restore" the initial capacity to
   `64*1024` and silently reintroduce CR-01 verbatim.

Also minor: "silently truncating every line after it" mis-states `bufio`
behavior — on `ErrTooLong` the scanner *stops* and drops the remainder; it does
not truncate.

**Fix:**
```go
// affectedStdinMaxLineBytes bounds a single --stdin line (WR-06): a file
// path legitimately never needs to exceed this — 4096 is the common
// PATH_MAX — so a single pathological/malicious "line" far longer than any
// real path is treated as malformed input (an explicit error). This
// deliberately TIGHTENS bufio.Scanner's 64 KiB bufio.MaxScanTokenSize
// default, which is 16x larger than any real path needs.
const affectedStdinMaxLineBytes = 4096

// ...
// WR-06/CR-01: scanner.Buffer LOWERS the per-line token ceiling from
// bufio's 64 KiB default to affectedStdinMaxLineBytes. Both arguments must
// stay equal: Buffer's effective ceiling is max(maxArg, cap(buf)), so a
// larger initial capacity silently defeats the cap (this was the CR-01
// no-op). A genuine bufio.ErrTooLong is surfaced as an explicit error
// rather than silently stopping the scan and dropping every line after it.
```

### IN-02: WR-01 changed the user-visible over-limit message to a double-prefixed one that leaks the internal package name

**File:** `internal/cli/affected.go:200-202` (message produced by `internal/query/validate.go:155`)

**Issue:** The message changed from
`affected: input exceeds maximum 10000 files` to
`affected: query: 10001 files exceeds maximum 10000`. The `affected: query:`
double prefix is awkward on a v1.0 parity surface, and `query:` names an
`internal/` package that means nothing to a CLI user. (No doc, golden fixture,
or UAT item pins either string — grep confirms the old text had no other
references — so this is cosmetic only, and de-duplicating the bound was still
the right call.)

**Fix:** Either drop the redundant command prefix at the call site
(`return err`, matching how `internal/cli/affected.go:73` already surfaces raw
`query:`-prefixed engine errors), or restate the message without the second
prefix:
```go
if err := query.ValidateAffectedFiles(len(files) + 1); err != nil {
    return fmt.Errorf("affected: input exceeds maximum %d files: %w", query.MaxAffectedFiles, err)
}
```

### IN-03: The `--stdin` over-length error message is off by one against the behavior it reports

**File:** `internal/cli/affected.go:236`

**Issue:** The message reads `--stdin line exceeds maximum 4096 bytes`, but a
line of *exactly* 4096 bytes is rejected — verified empirically (4095 accepted,
4096 rejected, 4097 rejected). 4096 does not "exceed" 4096. This follows from
`bufio.Scanner`'s "token size must be less than max" contract, which the test's
own comment states correctly; only the user-facing string is imprecise. Harmless
in practice (4095 is exactly the usable `PATH_MAX` budget), but a user debugging
a boundary case will be told a number that is not the real limit.

**Fix:**
```go
return nil, fmt.Errorf("affected: --stdin line exceeds maximum %d bytes: %w",
    affectedStdinMaxLineBytes-1, err)
```
or reword to `"--stdin line must be under %d bytes"` and keep the constant.

---

_Reviewed: 2026-07-26_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep (iteration 3 — re-review of iteration-2 fixes `333ab82`, `331a5a6`, `4feb6ff`)_
