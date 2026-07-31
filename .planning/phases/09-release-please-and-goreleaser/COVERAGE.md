# API Coverage — Phase 9 (release-please + GoReleaser)

No external API integration: this phase changes CI workflow configuration, two
release-please JSON config files, tests, and documentation — it adds no external
API/SDK surface to product code.

## Why the detector fired

The `api-coverage` detector returned `detected: false` at plan time and
`detected: true` at `verify:pre`, on the signal `{verb: "(surface)", noun: "api"}`.
The difference is scope: at `verify:pre` the detector reads the phase SUMMARYs,
which describe *how release-please and GitHub Actions work* — phrases like
`octokit.repos.createRelease`, "tag and Release in one API call", and "GitHub's
search API mishandles the parentheses". Those are explanations of third-party
behaviour, not capabilities this phase integrates.

## Evidence

Every non-planning, non-docs file changed by this phase (`git diff --name-only
7e3a5fb..HEAD`):

| File | Kind |
|---|---|
| `.github/workflows/ci.yml` | CI config |
| `.github/workflows/pr-title.yml` | CI config |
| `.github/workflows/release-please.yml` | CI config |
| `.github/workflows/release.yml` | CI config (one publish step) |
| `release-please-config.json` | release tooling config |
| `.release-please-manifest.json` | release tooling config |
| `internal/upgrade/pr_title_lint_test.go` | test |
| `internal/upgrade/release_publish_step_test.go` | test |
| `internal/upgrade/release_workflow_shape_test.go` | test |

No product source file is modified. In particular
`internal/upgrade/upgrade.go` and `internal/upgrade/verify.go` — the **only**
product code in this repo that calls the GitHub API (release resolution and
artifact download for `codegraph upgrade`) — are byte-identical to their state at
the planning commit, verified by `git diff --stat 7e3a5fb..HEAD` returning empty.
That integration predates this phase (v0.1, DIST-01/DIST-02) and is unchanged by
it, so its capability surface is not this phase's to enumerate.

The `gh` CLI is invoked inside `release.yml` and by maintainers following
`docs/RELEASE-PROCEDURES.md`. That is release automation calling a tool, not the
product integrating an API.

## Status

Recorded per the `api-coverage` capability's documented path for a
false-positive detection: a reasoned declaration, not a fabricated matrix. Adding
INTEGRATE/OPT-OUT rows for GitHub API capabilities this phase does not touch would
assert coverage decisions that were never made — the precise failure the gate
exists to prevent.

Reversible: if a later phase adds real API integration to product code, delete
this declaration and produce a real matrix.

*Declared 2026-07-30 during `/gsd-verify-work 9`.*
