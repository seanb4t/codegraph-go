---
created: 2026-08-14T00:00:00.000Z
title: tools/bench/runner pinnedAt() validates a checkout by git rev-parse HEAD alone — the HEAD-only anti-pattern Phase 1's four-part integrity check replaces
area: bench
severity: minor
files:

  - tools/bench/runner/main.go:435-441

audit_acknowledged:
  milestone: v0.14.0
  at: 2026-09-20
---

## Problem

`pinnedAt(dir)` (`tools/bench/runner/main.go:435-441`) checks whether a
locally-resolved corpus checkout is at the expected commit by running `git -C
dir rev-parse HEAD` and comparing the result to the pinned SHA — nothing else.
`resolveOrClone` (WR-02, Phase 8 re-review) trusts that single equality before
benchmarking the checkout.

HEAD alone is not sufficient: a tree at the right commit with one modified
tracked file, one changed file mode, one retargeted symlink, or one injected
untracked file has the correct HEAD and the wrong contents. A checkout that
was manually edited, partially `git reset`, or left dirty by an interrupted
prior run would still pass `pinnedAt`'s check and silently get benchmarked as
though it were the pristine pinned tree — corrupting throughput numbers with
no visible signal.

This is exactly the anti-pattern v0.11.0 Phase 1 (Corpus Selection by
Measurement) identified and replaced elsewhere: `Taskfile.yml`'s
`corpora:assert-one` target's four-part integrity check requires ALL of (1) a
real `.git` directory, (2) HEAD equal to the pinned SHA exactly, (3) `git
status --porcelain --ignored` empty, and (4) `git rev-parse HEAD^{tree}`
resolvable — and its own `desc:` block cites `tools/bench/runner/main.go:482`'s
`pinnedAt` by name as the weaker pattern it deliberately improves on
(`01-04-SUMMARY.md`). `pinnedAt` itself was never updated to match.

## Solution

Mirror `corpora:assert-one`'s four-part check inside `pinnedAt` (or a
sibling helper `resolveOrClone` calls alongside it): real `.git` directory +
HEAD-equals-pinned-SHA + empty `git status --porcelain --ignored` + a
resolvable `git rev-parse HEAD^{tree}`. Any failing part means "not usable,"
the same disposition `resolveOrClone` already gives a HEAD mismatch today (it
re-clones). TBD on timing — this todo files the decision to fix it; it does
not schedule which milestone/phase does the work.
