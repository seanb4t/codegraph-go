---
created: 2026-10-17T16:51:49.035Z
title: Harden the release-tag job credential (IN-05)
area: release
severity: minor
files:
  - .github/workflows/release-tag.yml
  - Taskfile.yml
  - internal/upgrade/changie_release_shape_test.go
audit_acknowledged:
  milestone: v0.15.0
  at: 2026-10-17
---

## Problem

IN-05 from the 2026-10-14 Phase 5 review (`git show f05db688:.planning/phases/05-release-chain-release-please-retirement/05-REVIEW.md`). The `tag` job in `release-tag.yml` checks out with persisted credentials, so the App token lives in the checkout's git config for the whole job instead of only at push time.

This was deferred out of v0.15.0 by the maintainer's short-path decision (2026-10-17, Phase 6 CONTEXT D-03/D-07 amendments). The tag job ships v0.15.0 exactly as Phase 5 built and rehearsed it.

## Solution

- Check out the `tag` job with `persist-credentials: false`.
- Give `release:tag` the App token only at push time, through `env:`.
- This intentionally changes `validateTagJobCredential` and the `release-tag.yml` comment. Re-point them RED-first (rules `84d1gfpywd`, `x1cjy9vyhq`).
- Rehearse `release:tag` against a scratch bare repo before merging. A failed `release-tag` run re-runs at its original SHA and nothing else re-fires it, so a broken tag job cannot be repaired by fixing forward (engram `8ehs9r12s9`).
- Record the finding as `fixed` with its commit through the review-disposition step, never by hand.
