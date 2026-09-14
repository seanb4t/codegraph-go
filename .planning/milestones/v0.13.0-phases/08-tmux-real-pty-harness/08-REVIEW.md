---
status: issues_found
phase: 08-tmux-real-pty-harness
depth: deep
files_reviewed: 7
files_reviewed_list:
  - Taskfile.yml
  - test/tmux/capture.go
  - test/tmux/daemon_empty_test.go
  - test/tmux/daemon_picker_test.go
  - test/tmux/frame_stability_test.go
  - test/tmux/install_cancel_test.go
  - test/tmux/poll_contract_test.go
findings:
  critical: 0
  warning: 1
  info: 1
  total: 2
reviewed_at: 2026-09-11T00:00:00Z
---

# Phase 08: tmux Real-PTY Harness — Code Review Report (Incremental — plan 08-05, gap G-08-1)

**Reviewed:** 2026-09-11
**Depth:** deep (cross-file; diff-scoped against `3f7a70f1..HEAD`)
**Files Reviewed:** 7
**Status:** issues_found (no Critical, 1 Warning, 1 Info)

## Scope note

A prior `08-REVIEW.md` (committed at `3f7a70f1`) covered plans 08-01..08-04 and is superseded
by this file per the workflow's overwrite instruction. That pass found 0 Critical / 5 Warning /
4 Info; none of those five findings (WR-01 through WR-05) touch the lines this diff changed, so
none are re-litigated here as new findings — WR-01 (`capture.go`'s `<-time.After` comment is
misleading) and WR-02 (`frame_stability_test.go` bypasses `pollUntilStable`) both remain
present, byte-unchanged, in the current tree and stay open from the prior pass.

This pass reviews only the diff `3f7a70f1..HEAD` against the seven files in scope: `capture.go`
gained a required readiness predicate on `pollUntilStable` (`ready(capture) && capture ==
predecessor`, nil-predicate and empty-needle both `t.Fatal`), every one of the 11 call sites
across the five `*_test.go` files was updated to pass an anchor, `poll_contract_test.go` is a
new deterministic self-test of the wait primitive itself, and `Taskfile.yml`'s
`TMUX_EXPECTED_TESTS` moved 5→6 to match the new top-level test count.

## Summary

The core fix is sound. I traced `pollUntilStable`'s convergence logic instruction-by-instruction
and confirmed: (1) all 11 call sites pass a non-nil predicate (`rg -n
'pollUntilStable\(' test/tmux` — grep count matches manual enumeration); (2) every
`paneContains` anchor used (`"no running daemons"`, `"Running daemons"`, `"[ ]"`, `"[x]"`,
`"cgtmux-ready"`) is verifiably absent from the typed command line that precedes it in the same
test, so none of the five test files can converge on the pre-output frame the way G-08-1
exploited; (3) the two `altScreenOff` call sites correctly gate on a live, uncached
`alternate_on` query rather than on byte-content, matching the documented rationale that the
post-quit main-buffer content is not a usable anchor; (4) `poll_contract_test.go`'s
self-consistency guard (`delay > 2*interval` and `delay+3*interval < deadline`) is correct
arithmetic for the chosen `stabilityPollInterval`/`stabilityPollDeadline` pair, and the test's
own `elapsed < delay` assertion is a genuine regression guard — reverting to the pre-08-05
two-sample-only equality check would converge at ~sample 2 (~2s), well under the commanded 4s
delay, and this test would then fail loudly rather than passing vacuously. `go vet -tags tmux
./test/tmux/...` is clean, and `TMUX_EXPECTED_TESTS: 6` matches the six top-level `Test*`
functions post-diff exactly (`TestMain` does not emit its own pass/skip event, consistent with
the existing jq filter).

One structural gap in the new mechanism itself (not present in the old code, because the old
code had no anchors to misuse) is worth flagging as a WARNING below, plus one INFO on a
documentation nit. No BLOCKER-class defect found in this diff.

## Warnings

### WR-06: The new anchor-required convergence mechanism guards against an empty needle but not against a needle that appears in the pre-output frame — the exact vacuity class this plan closes has a second instance the design doesn't mechanically prevent

**File:** `test/tmux/capture.go:190-201` (`paneContains`); doc comment at `capture.go:127-133`
**Issue:** `pollUntilStable`'s doc comment states the caller-facing contract precisely: "An
anchor must be absent from the typed command line that precedes it, or the pre-output frame
satisfies it too." This is correct and every current call site respects it (verified above).
But `paneContains` only mechanically enforces the *empty-needle* case (`if needle == ""`)
— it has no way to check the *non-empty-but-still-present-in-the-echoed-command* case, because
it doesn't have access to what was typed. That means the property this whole plan exists to
guarantee (no anchor can be trivially satisfied by a pre-output frame) is, for any *future* call
site, enforced by nothing but a doc comment and manual review — the same "documented invariant,
zero durable enforcement" shape the prior review's WR-01/WR-02 already identified in this same
package, now reproduced in the mechanism built specifically to close that class of bug.
**Failure scenario:** A future test adds `pollUntilStable(t, session, paneContains(t,
binPath))` or anchors on a flag name that also appears verbatim in the `sendLiteral` command
line (e.g. anchoring on `"install"` for an `install` command test). The pre-output echoed
command line satisfies the anchor immediately, `readyEverHeld` becomes true at sample 0, and
`pollUntilStable` converges on the very first byte-stable capture — which could be the
pre-execution frame if the shell hasn't started executing yet by the second sample. This is
exactly G-08-1's shape, silently reintroduced through a call site the readiness mechanism was
built to make safe.
**Fix:** Land a companion always-running (no `tmux` tag) static check, following this
package's own `poll_contract_test.go` / the repo's `taskfile_shape_test.go` convention: parse
each `sendLiteral(t, session, X)` immediately followed by a `pollUntilStable(t, session,
paneContains(t, Y))` in the same test function and fail if `Y` is a substring of `X`. Absent
that, at minimum add a one-line note to `paneContains`'s doc comment pointing at this exact
failure mode so a reviewer checking a new call site knows what to look for (the current comment
states the rule but not the concrete way a call site violates it).

## Info

### IN-05: `poll_contract_test.go`'s `start` timestamp is taken before `sendKey(t, session, "Enter")`, not after

**File:** `test/tmux/poll_contract_test.go:52-54`
**Issue:** `start := time.Now()` is captured between `sendLiteral` (typing, no execution) and
`sendKey(t, session, "Enter")` (the keystroke that actually starts the command). The elapsed
time the test later checks against `delay` therefore includes the `send-keys` subprocess's own
latency, which is harmless in the direction this test needs (it only makes `elapsed` larger,
never smaller, so it can't cause a false pass) but does mean the reported/asserted "elapsed"
figure over-counts the genuine wait by a small, unaccounted margin.
**Fix:** Move `start := time.Now()` to immediately after `sendKey(t, session, "Enter")` for a
tighter, more accurate elapsed measurement. Not required — current placement is conservative,
not incorrect — but worth a one-line comment noting the placement is deliberate if it's staying.

---

_Reviewed: 2026-09-11_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep_
