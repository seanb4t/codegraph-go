---
created: 2026-09-25T00:00:00.000Z
title: Adopt changie for changelog + version and replace release-please with a changie release-PR workflow
area: release
severity: major
resolves_phase: 5
files:
  - .planning/notes/changie-release-management.md
  - .github/workflows/release-please.yml
  - release-please-config.json
  - .release-please-manifest.json
  - CONTRIBUTING.md
  - docs/RELEASE-PROCEDURES.md
---

## Problem

CHANGELOG.md is one bullet per squash title and the version bump depends on that title being
honest (spine `q1hg0dewy6`). Design and decisions: `.planning/notes/changie-release-management.md`.

## Checklist (ordered; each step is independently shippable)

1. **Config + baseline.** Add `.changie.yaml` (kinds `Breaking→minor` until 1.0, `Features→minor`,
   `Fixes/Performance/Dependencies→patch`; custom `PR`), `.changes/header.tpl.md`,
   `.changes/unreleased/.gitkeep`, and `.changes/v0.14.0.md` so `changie latest` == `v0.14.0`.
   Keep existing CHANGELOG entries verbatim below the changie header. Prove: `changie latest`,
   `changie next auto` (fails with no fragments — expected), `changie merge --dry-run`.
2. **Release workflow.** New job graph replacing `release-please.yml`: on push to `main` with
   unreleased fragments → `pretag-gate` (moved) → `changie batch auto && changie merge` →
   open/update `chore(main): release vX.Y.Z` PR. On merge (new `.changes/vX.Y.Z.md` on main) →
   `actions/create-github-app-token` (same App secrets) → `git tag` + push. **Do not touch
   `release.yml`'s trigger.** Pin all actions by SHA.
3. **GoReleaser body.** `release.yml`: pass `--release-notes .changes/${TAG}.md`; confirm
   `changelog.disable` is NOT set. Rehearse with `task release:dry-run`.
4. **Fragment-required gate.** New required check: PR touching `cmd/**` or non-test
   `internal/**` must add `.changes/unreleased/*.yaml` unless body has
   `<!-- changelog-exempt: <reason> -->`. Fail closed on empty file list. Add to ruleset
   `protect-main` required checks (becomes 7).
5. **Retire release-please.** Delete `release-please.yml`, `release-please-config.json`,
   `.release-please-manifest.json`. Update `CONTRIBUTING.md` (fragment instead of title-driven
   version; remove Release-As rules), `docs/RELEASE-PROCEDURES.md`, `pr-title.yml` comment
   header (hygiene only).
6. **GSD hook.** Phase close writes fragments non-interactively:
   `CI=true changie new -k <Kind> -b "<sentence>" -m PR=<n>`. Pending the research question on
   where this lives (gsd-core capability vs upstream request).
7. **First real release.** Merge the first release PR; verify the tag is App-authored, `release.yml`
   fired, Release body == `.changes/vX.Y.Z.md`, `post-release-verify` green, `codegraph upgrade`
   verifies. Then plant SEED-005 (other repos).

## Guards

- A guard MUST carry a positive assertion (rule `84d1gfpywd`): the gate test must show a PR
  *with* a fragment passing and one *without* failing.
- Never `[ci skip]` (rule `f18zrdsgx5`).
