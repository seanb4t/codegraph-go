---
created: 2026-10-17T16:51:49.035Z
title: Fix the release runbook framing, tap remedy and trigger claim (WR-02, WR-03, WR-04)
area: docs
severity: minor
files:
  - docs/RELEASE-PROCEDURES.md
audit_acknowledged:
  milestone: v0.15.0
  at: 2026-10-17
---

## Problem

Runbook WR-02, WR-03 and WR-04 from the 2026-10-16 Phase 5 review (`.planning/phases/05-release-chain-release-please-retirement/05-REVIEW.md`), all in `docs/RELEASE-PROCEDURES.md`:

- **§7/§9** frame the first release as a "disposable live proof". A published `vX.Y.Z` is real and patch-forward only (Phase 6 D-09), and §7 forbids reusing a stable tag name.
- **§12**'s tap-failure remedy re-runs the full `goreleaser release`.
- **§4 step 5**'s trigger claim is wrong.

This was deferred out of v0.15.0 by the maintainer's short-path decision (2026-10-17, Phase 6 CONTEXT D-03/D-07 amendments).

## Solution

- **§7/§9:** drop the "disposable live proof" framing. Use D-09's wording: real, stable, patch-forward only.
- **§12:** replace the remedy with one that does not re-run the full release.
- **§4 step 5:** word the claim as a condition: "if a merge to `main` changes `CHANGELOG.md`, `release-tag.yml` runs and skips with success when the tag already exists". The milestone merge (#88) did not change `CHANGELOG.md`, because `origin/main` already carried the v0.14.1 section, so the original premise was stale (Phase 6 RESEARCH §5).
- Record the findings as `fixed` through the review-disposition step.
