## What this adds

A `fragment-required` check that fails a pull request when it changes shipped code and neither adds a changelog fragment nor explains why it needs none.

- `.github/workflows/fragment-required.yml` runs on `pull_request_target` and checks out the base branch, so the policy that judges a PR always comes from `main`. It holds read-only scopes, posts no comment, and its job body is one `task` call.
- `scripts/fragment_policy.py` holds the policy, with a 36-case unittest suite in `scripts/test_fragment_policy.py`. Untrusted input (the PR body and the changed paths) reaches Python only through environment variables, and the script lists the changed files itself through gh's paginated REST listing.
- `task check:fragment-required` runs the policy, and `task check:fragment-policy` runs the suite behind an executed-count guard that fails if fewer than 36 test methods ran. The `ci.yml` test job runs the second one.
- The ruleset-drift guard is retired: `scripts/check-ruleset-drift.sh`, `.github/required-status-checks.txt`, their `ci.yml` step, and the required-check tests in `internal/upgrade/taskfile_shape_test.go` are removed. Once `fragment-required` joins the ruleset, a text-file mirror of the required contexts would go red on every PR to `main`, and a required check that never reports already blocks loudly.

## Why it belongs here

The v0.15.0 milestone (#88) makes changelog fragments the changelog's input, and this gate makes a missing fragment visible on every PR. The workflow reads from the base branch, so it cannot report on any PR, #88 included, until it exists on `main`. This PR lands it ahead of the milestone, so the check exists before it becomes required.

## How it behaves

- Shipped code is any file under `cmd/` or `internal/`, other than `*_test.go` files and `testdata/` directories. A removed or renamed-away shipped file counts as a change.
- "Has a fragment" means a newly added file under `.changes/unreleased/` other than `.gitkeep`. Editing or deleting an existing fragment does not count.
- A PR can opt out with `<!-- changelog-exempt: <reason> -->` in its body. The reason must be non-empty, and it is shown as a notice in the check log.
- The gate fails closed in three states: the changed-file list cannot be read, the list is empty or shorter than GitHub reports, or the PR body cannot be read. Each state has its own message.
- On a miss, the check output names the offending paths, the `task changie -- new` command that adds a fragment, and the exemption marker.
- Planning-only, docs-only and dependency-only PRs touch no shipped code, so they pass without a marker.

This PR itself does not get a `fragment-required` run, because `pull_request_target` reads the workflow from `main`, where it does not exist yet.

## Required checklist

- [x] The 36-case unittest suite passes, and `task check:fragment-policy` reports `executed 36 of 36 defined test methods (floor 36)`
- [x] `go test ./internal/upgrade/` passes (the workflow-shape suite covers the new workflow and no longer covers the retired fixture)
- [x] `task lint:actions` (actionlint) is clean
- [x] Ten mutations of the gate's own rules, count guard and single-definition registration each turn a named test or guard red and revert byte-clean. The log is `.planning/phases/04-fragment-required-gate/04-MUTATION-LOG.md` on the milestone branch

## Compatibility

- [x] No change to the codegraph binary, the graph schema or any MCP output shape. This PR changes CI only.
- [x] Commit and PR types are `ci`, `test` and `chore`, so release-please opens no release from this merge.
- [x] I did not touch `internal/upgrade/verify.go`, `release.yml`, or `tools/bench/baseline.json`.
- After this merges, the maintainer adds `fragment-required` as a required status check on ruleset `protect-main`.

🤖 Generated with [Claude Code](https://claude.com/claude-code)

https://claude.ai/code/session_013NcLoeED2PgUHSnC46XKLZ
