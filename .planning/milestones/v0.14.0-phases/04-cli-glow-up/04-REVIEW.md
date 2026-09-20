---
phase: 04-cli-glow-up
reviewed: 2026-09-17T21:10:00Z
depth: standard
files_reviewed: 49
files_reviewed_list:
  - docs/CLI-REFERENCE.md
  - go.mod
  - internal/cli/affected.go
  - internal/cli/callees.go
  - internal/cli/callers.go
  - internal/cli/cli_reference_test.go
  - internal/cli/colorflag_test.go
  - internal/cli/colorflag.go
  - internal/cli/daemon.go
  - internal/cli/explore.go
  - internal/cli/files.go
  - internal/cli/githooks.go
  - internal/cli/impact.go
  - internal/cli/init.go
  - internal/cli/init_advisory_test.go
  - internal/cli/install_test.go
  - internal/cli/install.go
  - internal/cli/node.go
  - internal/cli/plain_golden_test.go
  - internal/cli/present/ansistrip_test.go
  - internal/cli/present/archtest/import_graph_test.go
  - internal/cli/present/explore_test.go
  - internal/cli/present/explore.go
  - internal/cli/present/files_test.go
  - internal/cli/present/files.go
  - internal/cli/present/help_test.go
  - internal/cli/present/help.go
  - internal/cli/present/line_test.go
  - internal/cli/present/line.go
  - internal/cli/present/node_test.go
  - internal/cli/present/node.go
  - internal/cli/present/palette_test.go
  - internal/cli/present/palette.go
  - internal/cli/present/results_test.go
  - internal/cli/present/results.go
  - internal/cli/present/sanitize.go
  - internal/cli/present/sanitize_test.go
  - internal/cli/present/status_test.go
  - internal/cli/present/status.go
  - internal/cli/present/styles.go
  - internal/cli/present/tty.go
  - internal/cli/root.go
  - internal/cli/search.go
  - internal/cli/serve.go
  - internal/cli/short_flags_test.go
  - internal/cli/status.go
  - internal/cli/sync.go
  - internal/cli/telemetry.go
  - internal/cli/ui.go
  - internal/cli/uninit.go
  - internal/cli/upgrade.go
  - internal/cli/version.go
  - test/integration/status_files_plain_test.go
findings:
  critical: 0
  warning: 0
  info: 1
  total: 1
status: clean
---

# Phase 04: Code Review Report (iteration 2)

**Reviewed:** 2026-09-17T21:10:00Z
**Depth:** standard
**Files Reviewed:** 49 (full phase scope; 4 fix commits examined in detail)
**Status:** clean

## Summary

This is iteration 2 of the `--auto` fix loop. Iteration 1 (`04-REVIEW.md`, superseded by this file) found CR-01, CR-02, WR-01, WR-02 and left IN-01 as an out-of-scope Info item. All four in-scope findings were fixed across commits `35852c5e` (CR-01), `fad9879b` (CR-02), `df8f718a` (WR-01), `8d65f2f3` (WR-02). This review re-examined each fix's diff against the actual current source (not just the fix report's claims), re-derived the fix's correctness by tracing call sites and edge cases, and ran the full verification suite.

**Verification performed (not just re-stated from the fix report):**
- `GOTOOLCHAIN=go1.26.6 go build ./...` — clean. (Note: the ambient default `go1.27.1` toolchain fails to build an unrelated transitive dependency, `github.com/cockroachdb/swiss`, with `undefined: hashFn` etc. — a pre-existing environment/pinning issue, not caused by or related to this phase's diff. `GOTOOLCHAIN=go1.26.6` — the pinned toolchain per `go.mod` — builds and tests cleanly.)
- `go test -count=1 ./internal/cli/... ` — all packages pass, including the new `TestInitAdvisory_ColorResolvedOnce` and `TestLineWriterPartialFailureReturnsBytesWritten`.
- `go test -count=1 ./internal/cli/ -run 'TestPlainGolden$' -v` — all 29 golden subtests pass byte-for-byte.
- `go vet ./internal/cli/...` — clean. `gofmt -l internal/cli/` — no output.
- `task docs:cli:drift` — `docs/CLI-REFERENCE.md` byte-identical to a fresh regeneration.
- `git diff --stat 51025160..HEAD -- internal/query internal/mcp testdata/golden testdata/wireoracle go.mod` — empty, confirming the frozen-module constraint held across all four fix commits.
- `git diff --stat 51025160..HEAD` — touches exactly `internal/cli/files.go`, `internal/cli/init.go`, `internal/cli/init_advisory_test.go`, `internal/cli/present/line.go`, `internal/cli/present/line_test.go`, `internal/cli/present/node_test.go`, `internal/cli/present/sanitize.go`, `internal/cli/present/sanitize_test.go`, `internal/cli/present/status.go` — no scope creep beyond the four findings' fix + their test/comment updates.

**Per-fix re-verification:**

1. **CR-01** (`internal/cli/init.go`): `resolveColor(cmd)` is now called exactly once in `RunE`, threaded as `mode colorMode` into both `printSummaryMode` and `printWatchFallbackAdvisory` (whose signature changed to accept `mode` instead of resolving it internally). Confirmed `index.go` — the only other caller of `printSummary`/`printSummaryMode` — does not also call `printWatchFallbackAdvisory`, so no analogous double-resolve exists elsewhere in the codebase. `TestInitAdvisory_ColorResolvedOnce` genuinely exercises the real `init` `RunE` end-to-end (via `os.File`-backed stdout/stdin, correctly noting `execCmd`'s `bytes.Buffer` harness can't reach this gate) and asserts `queryDarkBackground` fires exactly once. Holds.

2. **CR-02** (`internal/cli/present/sanitize.go`): the strip predicate is narrowed to `unicode.IsControl(r) && r != '\t'` — every other C0/C1 control (ESC, CR, BS, DEL, etc.) is still stripped; only the bare tab survives, which is correctly reasoned as unable to initiate an OSC/CSI/DCS sequence. Cross-checked the one other test that intentionally probes ESC-stripping (`node_test.go`'s `ControlBytesStrippedFromStyled`, using `\x1b[31m`) — unaffected by the narrowing, still asserts the real injection vector is closed. `status.go`'s doc comment claiming `sanitizeControl` strips `\t` was correctly updated. `sanitize_test.go` and `node_test.go`'s `sanitizeLines`/`styledTabWidth` fixtures were updated in both directions (tab now survives; CR/ESC/DEL still stripped) and correctly model lipgloss's independent tab-to-4-spaces expansion so the test oracle matches the real renderer's output rather than the raw tab. Holds — no new injection surface, no widening beyond tab.

3. **WR-01** (`internal/cli/files.go`): the worktree notice print now happens after `resolveColor`, and the styled branch routes it through `present.RenderNotice` before `present.RenderFiles`, exactly matching the six sibling verbs (`explore`, `node`, `search`, `callers`, `callees`, `impact`, `affected` all verified via `rg` to share this identical `mode := resolveColor(cmd)` → `RenderNotice` → `Render<Verb>` shape). The plain branch is untouched in content/position — `files-flat.golden`/`files-tree.golden` still pass byte-for-byte. Holds.

4. **WR-02** (`internal/cli/present/line.go`): `Write` now accumulates `n` from the original (pre-sanitization) byte length of each successfully-written segment (`len(seg)+1` for a complete line including its separator, `len(seg)` for the final partial segment), returning that count alongside the error on a partial failure instead of `(0, err)`. Traced the arithmetic against `strings.Split`'s segment/separator accounting for both the full-success case (`n == len(p)`, matching prior behavior) and the partial-failure case (`n` == bytes of `p` actually consumed before the failing `io.WriteString`) — correct per `io.Writer`'s contract. The two real callers (`serve.go`'s watcher-stderr `NewLineWriter`, `upgrade.go`'s refresh-warning `NewLineWriter`) don't retry on short writes, so this is a pure correctness improvement with no behavioral risk to existing call sites. `TestLineWriterPartialFailureReturnsBytesWritten`'s `failAfterWriter` stub correctly fails on the second `Write` call and the test's expected byte count (`len("one\n")`) matches the traced arithmetic. Holds.

No new defects were introduced by any of the four fixes. The change set for this iteration is minimal and tightly scoped to the four findings plus their required test/comment updates.

IN-01 (`RenderStatus`'s "Project:" value sanitized but unstyled, `internal/cli/present/status.go:163`) remains open, unchanged, and out of scope per the original review's classification (Info-level, `fix_scope: critical_warning`) — not re-litigated here.

## Structural Findings (fallow)

None provided for this iteration.

## Narrative Findings (AI reviewer)

None. All four in-scope findings from iteration 1 (CR-01, CR-02, WR-01, WR-02) are confirmed fixed and hold under trace-level re-verification and the full test/build/vet/golden/drift suite. No new Critical or Warning issues were found in the fixed files or their call sites.

### IN-01: `RenderStatus`'s "Project:" value is sanitized but never styled (carried over, unchanged, out of scope)

**File:** `internal/cli/present/status.go:163`
**Issue:** Every other data field in `RenderStatus` is wrapped in a palette role (`pal.Count.Render(...)`, etc.), but the `Project:` line renders the sanitized path as bare text with no style applied. Not a security issue — the value is still sanitized — and not asserted against by any test.
**Fix:** Wrap the value in `pal.Path.Render(...)` (matching how paths are styled elsewhere, e.g. `RenderFiles`) if there's no deliberate reason to leave it undecorated. Deliberately left unfixed this iteration (Info-level, out of the `critical_warning` fix scope).

---

_Reviewed: 2026-09-17T21:10:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
