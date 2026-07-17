---
phase: 05-git-sync-hooks
fixed_at: 2026-07-17T02:16:10Z
review_path: .planning/phases/05-git-sync-hooks/05-REVIEW.md
iteration: 2
findings_in_scope: 2
fixed: 2
skipped: 0
status: all_fixed
---

# Phase 05: Code Review Fix Report

**Fixed at:** 2026-07-17T02:16:10Z
**Source review:** .planning/phases/05-git-sync-hooks/05-REVIEW.md
**Iteration:** 2

**Summary:**
- Findings in scope: 2
- Fixed: 2
- Skipped: 0

## Fixed Issues

### CR-01: `stripMarkerBlock`'s unterminated-marker guard is defeated by `Install`'s own append-raw-content recovery — silent data loss on the *next* strip

**Files modified:** `internal/githooks/githooks.go`, `internal/githooks/githooks_test.go`
**Commit:** f514e57
**Applied fix:**

Root cause: `stripMarkerBlock` only tracked a single `inBlock`/`sawUnterminatedBegin` pair, so a second `markerBegin` encountered while already `inBlock == true` silently re-entered the "open" state instead of being rejected. Combined with `Install`'s iteration-1 recovery strategy (fall back to raw content, append a fresh block after it when the strip couldn't be trusted), a file left with a dangling begin marker followed by a later well-formed block would have its *next* strip falsely report `ok == true` and silently delete everything between the dangling begin and the new end marker.

Applied a two-part fix, adapted from the REVIEW.md guidance to the actual current code (iteration-1's fix had already changed the signature/doc comments from what the finding quoted, so the patch was reconciled against the live file rather than pasted verbatim):

1. **`stripMarkerBlock`** now rejects both malformed shapes explicitly: a second `begin` encountered while one is already open sets `malformed = true` (never trusts a later `end` to "rescue" the pairing), and a dangling `end` with no open `begin` also sets `malformed = true`. Both cases, plus the original unterminated-begin case, now return `(content, false)` — the original content, untouched, with `ok == false`.
2. **`Install`**'s fallback branch no longer appends a fresh block after untrustworthy raw content. When `stripMarkerBlock` reports `ok == false`, `Install` now skips that hook entirely, accumulates an error in `InstallResult.Errors` ("hook file has a malformed codegraph marker block — please fix or remove it manually"), and leaves the file byte-for-byte untouched — matching `Remove`'s existing `ok == false` handling. This closes the invariant gap: no sequence of `Install`/`Remove` calls can delete user content, on well-formed or malformed input, because a file whose strip can't be trusted is never written to by either function.

Well-formed-path TS-verbatim semantics (D-02/D-03) are untouched — all pre-existing well-formed-input tests (`TestInstall_PriorBlockReplaced_StripThenAppendAtEnd`, `TestInstall_ReinstallOnUnmodifiedFile_ByteIdentical`, `TestMarkerBlock`, etc.) still pass unmodified.

**Regression tests added** (all pass):
- `TestStripMarkerBlock_NestedBegin_ReturnsUnchanged` / `TestStripMarkerBlock_DanglingEnd_ReturnsUnchanged` — unit-level coverage of the two new malformed shapes.
- `TestInstall_MalformedMarkerBlock_SkipsHookAndLeavesFileUntouched` — Install's first encounter with a malformed file: hook skipped, error accumulated, file untouched.
- `TestInstall_Install_TwiceOnMalformedFile_UserContentSurvivesBothCalls` — the exact two-call reproduction from the review (Install → Install), asserting `echo after`/`echo more-user-content` survive both calls byte-for-byte.
- `TestInstall_ThenRemoveOnMalformedFile_UserContentSurvives` — the Install → Remove variant, asserting the same invariant.

## WR-01: `uninit`'s D-06 hook cleanup still silently discards `RemoveResult.Errors`, unlike the standalone `githooks remove` command

**Files modified:** `internal/cli/uninit.go`, `internal/cli/uninit_test.go`
**Commit:** f2af40a
**Applied fix:**

`internal/cli/uninit.go`'s D-06 best-effort cleanup call to `githooks.Remove` now calls `printHookErrors(cmd, result.Errors)` (already defined in the same `cli` package by `internal/cli/githooks.go`, no new helper needed) immediately after the `Remove` call, matching the standalone `githooks remove` subcommand. The D-06 contract ("cleanup can never fail uninit") is preserved unchanged — this only adds a warning line to stderr; `RunE`'s return value is untouched.

**Regression test added** (passes): `TestUninit_UnwritableHooksDir_SurfacesWarning` — installs hooks, chmods the hooks directory to `0500` so the strip-and-rewrite path fails, runs `uninit --force`, and asserts stderr contains a `warning:` line. Skipped under root and on Windows, matching the existing convention in `internal/githooks`'s `TestRemove_UnwritableHooksDir_AccumulatesErrors`.

## Skipped Issues

None — all findings were fixed.

## Verification

- `go build ./...` — clean
- `go vet ./...` — clean
- `gofmt -l .` — clean
- `go test ./internal/githooks/...` — all pass (28 tests, including 7 new)
- `go test ./internal/cli/...` — all pass (including 1 new)
- `go test ./...` (full suite) — all packages pass

---

_Fixed: 2026-07-17T02:16:10Z_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 2_
