---
created: 2026-10-17T16:51:49.035Z
title: Close a stale changie/release PR in the zero-fragment path (WR-01)
area: release
severity: minor
files:
  - .github/workflows/release-pr.yml
  - Taskfile.yml
---

## Problem

WR-01 from the 2026-10-14 Phase 5 review (`git show f05db688:.planning/phases/05-release-chain-release-please-retirement/05-REVIEW.md`). When `release:batch` finds no unreleased fragments, `release-pr.yml` skips `create-pull-request` and leaves any open `changie/release` PR in place. If fragments are removed after a release PR opens, that PR goes stale and could still be merged.

This was deferred out of v0.15.0 by the maintainer's short-path decision (2026-10-17, Phase 6 CONTEXT D-03/D-07 amendments). It does not affect the v0.15.0 run, because no `changie/release` PR is open when the release merges.

## Solution

In the zero-fragment path, close any open `changie/release` PR. Do the close through a Taskfile target, so `TestWorkflowRunBodiesInvokeTask` holds: every CI `run:` body is a `task` invocation. Record the finding as `fixed` with its commit through the review-disposition step.
