---
phase: 05-git-sync-hooks
reviewed: 2026-07-16T00:00:00Z
depth: deep
files_reviewed: 14
files_reviewed_list:
  - internal/agents/shared.go
  - internal/cli/githooks.go
  - internal/cli/githooks_test.go
  - internal/cli/init.go
  - internal/cli/init_advisory_test.go
  - internal/cli/root.go
  - internal/cli/uninit.go
  - internal/cli/uninit_test.go
  - internal/fsatomic/fsatomic.go
  - internal/fsatomic/fsatomic_test.go
  - internal/githooks/githooks.go
  - internal/githooks/githooks_test.go
  - internal/gitmeta/githooks.go
  - internal/gitmeta/githooks_test.go
findings:
  critical: 0
  warning: 1
  info: 1
  total: 2
status: issues_found
---

# Phase 05: Code Review Report (Iteration 5 — Round-5 Fix Verification / Round-6 Final Clean-Check)

**Reviewed:** 2026-07-16
**Depth:** deep
**Files Reviewed:** 14
**Status:** issues_found

## Summary

Task: verify the two round-5 commits (`73aa510` — `Status` switched to
`hasMarkerLine`'s exact-trimmed-line detection; `b7c38ad` — `Remove`
accumulates non-`fs.ErrNotExist` read errors, mirroring `Install`'s CR-02
handling) are genuine and internally consistent, then spot-check the whole
marker-handling surface (`Install`/`Remove`/`Status`/init advisory/uninit
cleanup) for one remaining class of defect: same detection semantics, same
error-surfacing, same skip-untouched-on-unparseable convention.

**Both round-5 fixes verified genuine**, confirmed by diffing `73aa510` and
`b7c38ad` directly against the current file content (no drift between the
committed patch and what's on disk) and by re-running the full package test
suite (`go test ./internal/githooks/... ./internal/cli/... ./internal/gitmeta/... ./internal/fsatomic/...`
— all green, `go build ./...` and `go vet` clean):

- **`73aa510` (WR-01, `Status` exact-trimmed-line detection):** `Status` now
  calls the same `hasMarkerLine` helper `Remove`'s WR-05/IN-04 fix relies on
  (`strings.TrimSpace(line) == markerBegin` scanned line-by-line), replacing
  the old `strings.Contains(content, markerBegin)` raw substring check.
  Hand-traced `TestStatus_MarkerTextEmbeddedInLine_ReportsNotInstalled`'s
  fixture (`echo "not a real <begin> marker"`) against the new code: no line
  trims to an exact match, `hasMarkerLine` returns `false`, `Status` reports
  `not installed` — no longer contradicting what `Remove` finds for the
  identical file. `Status`/`Remove`/`stripMarkerBlock` now share one
  detection primitive; the `init` advisory's "hooks already installed" gate
  (which reads `Status.Hooks[].Installed`) inherits the fix for free since it
  only consumes `Status`'s output.
- **`b7c38ad` (WR-02, `Remove` read-error accumulation):** `Remove`'s
  `os.ReadFile` error branch is now `if !errors.Is(err, fs.ErrNotExist) { errs
  = append(...) }` before the `continue`, mirroring `Install`'s existing
  three-way switch (`err == nil` / `errors.Is(err, fs.ErrNotExist)` /
  `default`). Hand-traced the new
  `TestRemove_UnreadableExistingFile_LeavesFileUntouchedAndAccumulatesError`
  fixture (chmod 0000 an installed hook, then `Remove`): the file is left
  untouched (unchanged from before this fix — the safety property was never
  in question) and now also lands an entry in `RemoveResult.Errors` naming
  the hook, which both `githooks remove`'s and `uninit`'s D-06 cleanup's
  shared `printHookErrors` call surfaces as a `stderr` warning. Symmetric
  with `Install`'s CR-02 fix for the identical error class.

**One new, non-destructive finding surfaced during the round-6 pass** — a
signal/consistency gap in the same family as the WR-01/WR-02 findings this
round closed, on a code path those two fixes did not touch.

## Warnings

### WR-01: `Remove` still surfaces zero signal for a malformed marker block, unlike `Install`'s identical-condition handling

**File:** `internal/githooks/githooks.go:362-370`

**Issue:** `Install`'s CR-01 handling of `stripMarkerBlock` returning
`ok == false` (unterminated begin, nested begin, or dangling end — a
malformed marker pairing) both skips the write **and** accumulates an
actionable error:

```go
// Install, githooks.go:260-273
stripped, ok := stripMarkerBlock(base)
if !ok {
    errs = append(errs, fmt.Errorf("%s: hook file has a malformed codegraph marker block — please fix or remove it manually", hook))
    continue
}
```

`Remove`'s handling of the identical condition only does the first half —
it correctly leaves the file untouched and does not report it as `Removed`,
but it accumulates nothing in `RemoveResult.Errors`:

```go
// Remove, githooks.go:362-370
stripped, ok := stripMarkerBlock(string(original))
if !ok {
    // Unterminated/dangling begin or end marker (CR-01): don't
    // trust the strip. Leave the file untouched and don't report
    // it as removed ...
    continue
}
```

This is the exact "same detection semantics, same skip-untouched
convention, but NOT the same error-surfacing" gap the round-6 task asked to
verify is closed — WR-01 (this round's `73aa510`) and WR-02 (`b7c38ad`) each
closed one instance of it (detection parity, read-error parity), but the
malformed-marker case was never symmetrized. Confirmed neither
`TestRemove_UnterminatedMarkerBlock_LeavesFileUntouched`,
`TestStripMarkerBlock_NestedBegin_ReturnsUnchanged`/`DanglingEnd_...`, nor
`TestRemove_DanglingEndMarkerOnly_NotReportedRemoved` assert anything about
`RemoveResult.Errors` — all three only check `Removed` membership and
byte-for-byte file content, so this gap is untested as well as unfixed.

End-to-end impact, traced through `internal/cli`: a hand-damaged hook file
(e.g. someone deletes the end-marker line while editing) makes
`githooks remove <repo>` print only `"No git sync hooks were installed —
nothing to remove."` — actively misleading, since a codegraph-owned block
*is* present, just unparseable, and needs manual attention. `codegraph
uninit`'s D-06 best-effort cleanup hits the same silent path: no `warning:`
line, `.codegraph/` still gets removed and the command still reports
success, and the broken hook file is left behind with no signal it exists.
Contrast with `githooks install <repo>` against the same file, which prints
a `warning: post-commit: hook file has a malformed codegraph marker block —
please fix or remove it manually` line — so the two sibling subcommands give
contradictory levels of visibility into the identical on-disk condition,
the same "signal/consistency, not data loss" class as the WR-01/WR-02
findings this round fixed.

**Fix:** Mirror `Install`'s error message in `Remove`'s `ok == false`
branch:

```go
stripped, ok := stripMarkerBlock(string(original))
if !ok {
    errs = append(errs, fmt.Errorf("%s: hook file has a malformed codegraph marker block — please fix or remove it manually", hook))
    continue
}
```

Add a regression test analogous to
`TestInstall_MalformedMarkerBlock_SkipsHookAndLeavesFileUntouched` but
driving `Remove` against `malformedHookFixture()` (already defined in this
test file), asserting the hook is absent from `Removed`, present in
`Errors`, the file is byte-for-byte unchanged, and — via a
`internal/cli`-level test mirroring
`TestUninit_UnwritableHooksDir_SurfacesWarning` — that both the standalone
`githooks remove` command and `uninit`'s D-06 cleanup path surface the
warning on `stderr` for a malformed fixture, not just an unwritable
directory.

## Info

### IN-01: `WR-02` label reused for two unrelated findings within the same file

**File:** `internal/githooks/githooks.go:232, 337`

**Issue:** The package already carries a documented "Concurrency (WR-02)"
finding on `Install`'s and `Remove`'s doc comments (the non-atomic
read-modify-write race, first recorded several review iterations ago —
`fe3f4e1`). `b7c38ad`'s new read-error-accumulation fix reuses the same
`WR-02` label in its inline comment (`// WR-02: mirror Install's CR-02
distinction...`) for a completely different finding. Both are genuine and
both are correctly fixed/documented, but a reader searching this file for
"WR-02" now finds two unrelated defects sharing one ID, from two different
review cycles' numbering namespaces — a minor traceability nit, not a
functional issue.

**Fix:** No code change needed; purely a documentation-hygiene item for a
future pass — e.g. rename the newer inline comment's tag (it has no
independent ID of its own in this round's numbering) or cross-reference
which review iteration minted which `WR-02` when either is next touched.

---

_Reviewed: 2026-07-16_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep_
