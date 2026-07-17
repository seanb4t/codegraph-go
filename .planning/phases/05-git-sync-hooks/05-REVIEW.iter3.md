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
  critical: 1
  warning: 1
  info: 0
  total: 2
status: issues_found
---

# Phase 05: Code Review Report (Iteration 2 — Post-Fix Re-Review)

**Reviewed:** 2026-07-16
**Depth:** deep
**Files Reviewed:** 14
**Status:** issues_found

## Summary

This is the iteration-2 re-review after all 7 iteration-1 findings (1 Critical + 3 Warnings + 3 Info) were fixed in commits `3be729f..5bc6348`. Each of the 7 fixes was verified individually and is real:

- **CR-01 (iter-1)** — `stripMarkerBlock` now returns `(string, bool)` and reports `ok=false` on an unterminated begin marker; `Install` falls back to raw content as the append base, `Remove` skips the file untouched. Verified against `TestStripMarkerBlock_UnterminatedBegin_ReturnsUnchanged` and `TestRemove_UnterminatedMarkerBlock_LeavesFileUntouched` — both pass, and the `Remove`-only, single-call path genuinely does leave the file byte-for-byte untouched.
- **WR-01 (iter-1)** — `InstallResult.Errors`/`RemoveResult.Errors` now accumulate per-hook failures, and `internal/cli/githooks.go`'s `printHookErrors` surfaces them to stderr. Verified against `TestInstall_OneHookPathIsDirectory_PartialSuccessWithErrors` and `TestRemove_UnwritableHooksDir_AccumulatesErrors` — both pass.
- **WR-02 (iter-1)** — concurrency caveat is now documented on `Install`/`Remove`. Doc-only, no behavioral claim to falsify.
- **WR-03 (iter-1)** — four new error-path tests added, all pass, and materially increase coverage of the partial-failure branches.
- **IN-01/IN-02/IN-03 (iter-1)** — `plural()` reuse, `t.Setenv` precondition, and the `Executable` field on `HookStatus` are all present and correctly wired, with passing regression tests.

`go build ./...`, `go vet` on all reviewed packages, and `gofmt -l` all come back clean.

However, per the adversarial-stance mandate ("check whether any fix introduced a NEW defect"), tracing `stripMarkerBlock`'s new `(string, bool)` contract through `Install`'s fallback branch surfaced a genuine new data-loss bug that iteration-1's tests do not exercise (they only ever call `stripMarkerBlock`/`Remove` **once** against a malformed file; nothing exercises `Install` on a malformed file **followed by** a second `Install` or `Remove` call). This is the same shape as the Phase-2 BL-01 lesson referenced in the task: the CR-01 fix protects the *first* call correctly, but `Install`'s "fall back to raw content and append a fresh block anyway" recovery strategy writes a new file shape that reintroduces the exact hazard CR-01 was built to close, one round-trip later. I reproduced this empirically with a standalone test driven directly against the real `Install` function (not just a manual trace) — see CR-01 below.

A second, smaller gap: the WR-01 fix (error surfacing) was applied to the standalone `githooks remove` CLI command but not to `uninit`'s own call to `githooks.Remove`, the only other call site in the tree — `uninit`'s D-06 best-effort cleanup still silently discards `RemoveResult.Errors`.

## Critical Issues

### CR-01: `stripMarkerBlock`'s unterminated-marker guard is defeated by `Install`'s own append-raw-content recovery — silent data loss on the *next* strip

**File:** `internal/githooks/githooks.go:67-92` (algorithm) and `internal/githooks/githooks.go:205-223` (`Install`'s fallback)

**Issue:** `stripMarkerBlock` tracks exactly one `inBlock`/`sawUnterminatedBegin` pair across the whole scan. It never rejects a **second** `markerBegin` encountered while already `inBlock == true` — it just re-sets `inBlock = true` (a no-op) and keeps scanning for the next `markerEnd`, whichever one comes first, regardless of which `begin` it "belongs" to.

`Install`'s CR-01 recovery path, when `stripMarkerBlock` reports `ok == false` (unterminated begin marker), intentionally does **not** use the stripped/untrustworthy result — but it still *writes* the file: it falls back to the **raw, unstripped** existing content as the append base, then appends a brand-new, well-formed marker block after it (`githooks.go:212-220`). This is documented as "a stray dangling marker left in place beats silently destroying the user's file" — true for *that* write, but it leaves the dangling (unterminated) begin marker physically in the file, now followed later in the same file by a **second**, complete, well-formed begin/end pair.

The next time *any* strip runs against that file — a second `Install`, or a `Remove` — `stripMarkerBlock` sees: old dangling `begin` → (never closed) → new `begin` (silently absorbed as a no-op re-entry into `inBlock`) → ... → new `end` (the first `end` it has seen at all). Because the scan doesn't distinguish "this `end` closes the *second* `begin`" from "this `end` closes the *first*, still-open `begin`", it treats the entire span from the **first** dangling begin through the **new** end marker as one block and drops everything inside it — including all of the user's real content that sat between the old dangling marker and the newly-appended block. `sawUnterminatedBegin` ends up `false` (the trailing `end` resets it), so this call now reports `ok == true`: the caller has no signal anything is wrong, and (for `Remove`) the truncated result gets written back to disk.

**Reproduction (verified against the actual `Install` function via a standalone test, not just traced):**
```go
malformed := "#!/bin/sh\necho before\n\n" + markerBegin +
    "\n... (block body, end marker line deleted) ...\necho after\necho more-user-content\n"
// write malformed to .git/hooks/post-commit

Install(ctx, root) // 1st call: CR-01 protects this — raw content preserved, block appended after
// file now: [malformed content incl. dangling begin] + blank lines + [full new begin..end block]

Install(ctx, root) // 2nd call
// "echo after" and "echo more-user-content" are GONE from the file — silently deleted,
// no error, no Errors entry, InstallResult reports success.
```
Running this exact scenario confirms: the second `Install` call silently drops both `echo after` and `echo more-user-content` from the file, with no error returned anywhere in the call chain. The same landmine fires if `Remove` (instead of a second `Install`) is the second call against the doctored file — `Remove` will report the hook as successfully `Removed` and write the truncated content, since `stripMarkerBlock` now (incorrectly) reports `ok == true`.

This directly undermines the CR-01 fix's own stated contract ("leave the file untouched... don't destroy the user's file") — it holds for exactly one call, then fails silently on the next one touching the same file. Any hand-edited or merge-conflict-damaged hook file that a user runs `codegraph githooks install` against twice (or installs then later removes) will lose data with zero indication anything went wrong.

**Fix:** The root cause is that `stripMarkerBlock` allows a `markerBegin` encountered while `inBlock` is already `true` to silently re-enter the "open" state instead of treating it as a second, distinct malformed condition. Reject nesting explicitly and stop trusting *any* `end` that follows it:

```go
func stripMarkerBlock(content string) (string, bool) {
	lines := strings.Split(content, "\n")
	var kept []string
	inBlock := false
	malformed := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == markerBegin {
			if inBlock {
				// A second begin marker while one is already open can never
				// be validly closed by inspection alone — don't guess which
				// end belongs to which begin. Bail out entirely rather than
				// let a later end marker "rescue" this into a false ok=true.
				malformed = true
			}
			inBlock = true
			continue
		}
		if trimmed == markerEnd {
			if !inBlock {
				// A dangling end with no open begin is also unreliable —
				// same treatment as CR-01's original unterminated-begin case.
				malformed = true
			}
			inBlock = false
			continue
		}
		if !inBlock {
			kept = append(kept, line)
		}
	}
	if inBlock || malformed {
		return content, false
	}
	return strings.Join(kept, "\n"), true
}
```

This alone stops the false `ok == true` on the second pass — but `Install`'s fallback still needs to stop *writing new bytes into the file* when the base is untrustworthy, otherwise the file just accumulates additional dangling begin markers on every subsequent `install` attempt (never resolving, though no longer losing data once the check above is in place). Recommended: when `stripMarkerBlock` reports `ok == false`, `Install` should **not** attempt to append at all — skip that hook, accumulate an explicit error/warning (e.g. `"%s: hook file has a malformed codegraph marker block — please fix or remove it manually"`), and leave the file byte-for-byte untouched, matching `Remove`'s existing (correct) behavior for the `ok == false` case.

Add a regression test exercising `Install` → `Install` (and `Install` → `Remove`) against a malformed/unterminated-begin fixture, asserting the user content is preserved across **both** calls, not just the first — the existing `TestStripMarkerBlock_UnterminatedBegin_ReturnsUnchanged` and `TestRemove_UnterminatedMarkerBlock_LeavesFileUntouched` tests only ever call the function once and would not have caught this.

## Warnings

### WR-01: `uninit`'s D-06 hook cleanup still silently discards `RemoveResult.Errors`, unlike the standalone `githooks remove` command

**File:** `internal/cli/uninit.go:61-64`

**Issue:** The iteration-1 WR-01 fix added `printHookErrors` to `internal/cli/githooks.go`'s `install`/`remove` subcommands so per-hook write/delete failures are no longer silently discarded. `uninit.go` is the only other call site of `githooks.Remove` in the codebase, and it was not updated to match:

```go
if result := githooks.Remove(cmd.Context(), root); len(result.Removed) > 0 {
    fmt.Fprintf(cmd.OutOrStdout(), "Removed git %s sync hook%s\n",
        strings.Join(result.Removed, ", "), plural(len(result.Removed)))
}
```

`result.Errors` is never inspected. If the hooks directory is unwritable during `uninit` (the exact scenario `TestRemove_UnwritableHooksDir_AccumulatesErrors` exercises for the standalone command), `uninit --force` will silently proceed as if hook cleanup succeeded with nothing to report — no warning, no indication that a hook file was left in a broken or partially-stripped state. This is inconsistent with the fix just applied one file over, and reintroduces the exact "silently discarded" failure mode WR-01 was written to eliminate, just on a different call path.

**Fix:**
```go
result := githooks.Remove(cmd.Context(), root)
printHookErrors(cmd, result.Errors)
if len(result.Removed) > 0 {
    fmt.Fprintf(cmd.OutOrStdout(), "Removed git %s sync hook%s\n",
        strings.Join(result.Removed, ", "), plural(len(result.Removed)))
}
```
(`printHookErrors` is already defined in `internal/cli/githooks.go` in the same `cli` package — no new helper needed.) The D-06 contract ("cleanup can never fail uninit") is preserved — this only adds a warning line, it does not change `RunE`'s return value.

---

_Reviewed: 2026-07-16_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep_
