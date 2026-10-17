---
created: 2026-10-17T16:51:49.035Z
title: Paginate and fail closed in the issue-link and template hygiene checks (WR-02, WR-03)
area: ci
severity: minor
files:
  - .github/workflows/require-issue-link.yml
  - scripts/pr_template_policy.py
  - scripts/fragment_policy.py
---

## Problem

WR-02 and WR-03 from the 2026-10-14 Phase 5 review (`git show f05db688:.planning/phases/05-release-chain-release-please-retirement/05-REVIEW.md`).

- **WR-02:** `require-issue-link.yml` and `pr_template_policy.is_housekeeping` list a PR's changed files without paginating. On a large PR they can judge a partial list and wrongly exempt it.
- **WR-03:** the exemption reason reaches stderr unescaped.

This was deferred out of v0.15.0 by the maintainer's short-path decision (2026-10-17, Phase 6 CONTEXT D-03/D-07 amendments). `Issue link required` is not a required check, so neither finding blocks a merge.

## Solution

- Copy `scripts/fragment_policy.py`: `fetch_changed_files` (paginated REST, count cross-checked against `changed_files`, fail closed on a mismatch) and `escape_data` (or refuse a multi-line reason).
- The two hygiene allowlists are a mirrored pair with a parity test. Edit both together.
- Record both findings as `fixed` through the review-disposition step.
