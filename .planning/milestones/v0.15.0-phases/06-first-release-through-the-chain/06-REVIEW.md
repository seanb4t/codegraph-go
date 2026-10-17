---
phase: 06-first-release-through-the-chain
reviewed: 2026-10-17T00:00:00Z
depth: deep
files_reviewed: 2
files_reviewed_list:
  - .changes/v0.15.0.md
  - CHANGELOG.md
findings:
  critical: 0
  warning: 2
  info: 2
  total: 4
status: issues_found
---

# Phase 6: Code Review Report

**Reviewed:** 2026-10-17
**Depth:** deep
**Files Reviewed:** 2
**Status:** issues_found

## Summary

Both files are changie output, and the review treated them as published, patch-forward-only release notes. Nothing in either file is wrong about what shipped. No finding proposes hand-editing either file. Any correction would go in a later changie fragment.

What I checked against the `v0.15.0` tag (`0246de0e`):

- **The claim is accurate.** At `v0.14.1`, `internal/cli/renamed.go` registered hidden `query` and `unlock` stubs that printed a rename notice and returned an error. At `v0.15.0` the file is gone and `root.go` no longer registers either command. `test/integration/removed_verbs_test.go` asserts that the real binary answers both with cobra's `unknown command` error and exit 1.
- **Both replacements exist at `v0.15.0`.** `search --full` is defined in `internal/cli/search.go:170`. `daemon unlock [path]` is defined in `internal/cli/daemon.go:331`.
- **The `Breaking` kind matches `.changie.yaml`.** It is a `minor` bump pre-1.0, so `0.14.1` to `0.15.0` is consistent.
- **Ordering and dates are consistent.** The 16 `## ` headings in `CHANGELOG.md` run newest first. Dates fall monotonically from 2026-10-17 to 2026-08-01. The `v0.15.0` tag commit is 2026-10-17 10:35 -0700 and the GitHub Release was published 2026-10-17T17:40Z.
- **The release body matches the fragment.** `gh release view v0.15.0` returns the `.changes/v0.15.0.md` text exactly.
- **The link target exists.** `pull/88` resolves to the merged PR (see WR-02).
- **`CHANGELOG.md` has the right shape.** It is `# Changelog`, a blank line, then the newest version, and it ends in a newline.

I found no wrong claims and no malformed markdown. The findings below are discontinuities and traceability gaps that could mislead or inconvenience a reader.

## Warnings

### WR-01: The v0.15.0 entry breaks the heading and link convention that every other entry follows

**File:** `CHANGELOG.md:3` (and `.changes/v0.15.0.md:1`)
**Issue:** Every entry from 0.2.0 to 0.14.1 uses the form `## [X.Y.Z](…/compare/vA...vB) (date)`, with no `v` prefix, a compare link and a parenthesised date. The v0.15.0 entry uses `## [v0.15.0](…/releases/tag/v0.15.0) — date`, with a `v` prefix, a release-tag link and an em-dash date. It is the only entry with no diff link between the previous and current tag. A reader moving from the 0.14.1 section to the 0.15.0 section gets a different look. Anyone who finds the section by searching for `[0.15.0]` or `0.15.0)` will miss it. A tool that reads the heading in the old shape (`[X.Y.Z](…compare/…)`) will not parse it. Under the `v0.15.0` heading the kind heading is `### Breaking`. Earlier breaking entries use `### ⚠ BREAKING CHANGES` or no breaking heading at all. This comes from `versionFormat`/`kindFormat` in `.changie.yaml`, which differ from the seeded history. It is a deliberate design choice, but nothing in the file says the format changed at this version.
**Fix:** No edit to the published files. Record the format change in a later release. The `RELEASE-PROCEDURES.md` or `CONTRIBUTING.md` text that describes the changie chain should say that entries from `v0.15.0` onwards use `v`-prefixed release-tag headings and have no compare link. If compare links matter, `versionFormat` could add one in a later release. The `v0.15.0` heading cannot be changed in place.

### WR-02: The only link in the entry goes to a milestone PR that does not describe the change

**File:** `.changes/v0.15.0.md:5` (and `CHANGELOG.md:7`)
**Issue:** The entry cites `[#88]`. PR #88 is titled `feat: v0.15.0 — changie release management`. Its body is a milestone goal about replacing release-please with changie, and it was opened as a draft so fragments could carry a PR number. The PR touches 53 files. The removal of `query` and `unlock` is a small part of it, in `internal/cli/renamed.go`, `internal/cli/root.go` and `test/integration/removed_verbs_test.go`. A user who follows the link to find out why the commands were removed, or what the earlier deprecation was, reaches a description about release tooling and finds nothing about the verbs. This follows from the optional `PR` field in `.changie.yaml` (02-CONTEXT D-13): every fragment of a milestone carries the milestone PR number.
**Fix:** No edit to the published files. If more detail is wanted, a later release can add context in a fragment, for example by linking the v0.14.0 change that introduced the stubs. For future milestones, omit `-m PR=` when the PR is a bundle PR, or link the specific commit or issue, so the link explains the change.

## Info

### IN-01: The GitHub Release body repeats its own title

**File:** `.changes/v0.15.0.md:1`
**Issue:** The release body begins with an `## [v0.15.0](…) — 2026-10-17` heading that links to the page it is shown on. The GitHub Release already shows the tag, title and date. The body also ends with two trailing blank lines. These come from `endOfVersion: 2` and `--release-notes .changes/${TAG}.md` in the `.goreleaser.yaml` comment. The Release is accurate. The heading is redundant and its link points to the page the reader is already on.
**Fix:** Cosmetic. Accept it, or have a later release feed goreleaser a body that omits the version heading.

### IN-02: "Breaking" overstates the runtime change, and the deprecation was never in the changelog

**File:** `CHANGELOG.md:5-7`
**Issue:** At `v0.14.1` the hidden stubs already failed with exit 1. The runtime difference in `v0.15.0` is that stderr changes from a rename message to cobra's `unknown command "query" for "codegraph"`. The new message no longer names a replacement, by design (D-01). The `### Breaking` label is still right in principle, because the commands are removed. The entry does give the replacements. But the changelog never announced the rename. The 0.14.0 entry reads only "polish, agent reach and Codex parity", so a reader sees the removal in 0.15.0 with no earlier notice in the changelog. Readers who skipped the stub message will not know the old commands were already non-functional.
**Fix:** No edit. A later fragment can state that the two commands had been non-functional rename stubs since v0.14.0. Future deprecations should get their own changelog line when they are introduced.

---

_Reviewed: 2026-10-17_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep_
