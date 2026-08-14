---
phase: 01
plan: 07
title: Coverage drift guard + CI cache — derived claim, unconditional idempotent fetch
agents: gsd-executor (worktree isolation)
status: complete
---

# Plan 01-07 — Coverage Drift Guard & CI Cache

## What was built

| Commit | What |
|---|---|
| `ceccdff` | The coverage guard — `CheckCoverage(manifest, observations, selection)` derives the claim from the three raw documents in seven steps and reads no stored `coverage[kind]` summary. New `internal/corpora/coverage.go` + `coverage_test.go`. |
| `2d495fc` | Split the drift guard into a cheap every-run leg (`corpora:verify` — guard package tests, corpora-free) and a path-filtered re-index leg (`corpora:drift` — re-measures at locked scope, diff-clean) so the expensive run fires only when the pins or the measuring code change. |
| `6bb7c63` | `.github/workflows/corpora.yml` — path-filtered sibling workflow on the established `namespacelabs/nscloud-cache-action` mechanism, with the **unconditional** idempotent fetch and the four-part integrity assertion. |

## Contracts honored

- **Cache is `namespacelabs/nscloud-cache-action`** (9 existing uses, locked v1.0 Phase 10 D-06), not `actions/cache`. Verified against its `action.yml`: no `key` input, so content-addressing comes from the SHA-bearing destination path `<root>/<slug>@<sha>`, not from a cache key. A pin bump makes a *new* path; there is no prefix-match surface to disable.
- **The fetch step is unconditional and idempotent**, never gated on `cache-hit`. A cache MISS falls through to a REAL fetch with a non-zero corpus count — no skip, no negative-only assertion. Content-addressed subdirectories make a prefix match on a SHA structurally impossible; there are no cache keys to leak across commits.
- **`CheckCoverage` derives the claim, never validates a stored summary.** It rejects: a kind below threshold; a supplier not in the locked set; a supplier with no observation; and a rejected (unlocked) candidate named as a supplier. Loading is the caller's job — the function takes decoded documents and performs no I/O.
- **Positive assertion everywhere (rule `84d1gfpywd`).** `corpora:assert` reports the exact count it verified versus `len(LockedEntries(m))`, and requires it to be > 0. Both `TestCorpusCoverageClaim` legs are specified to be demonstrated RED against a confirmed mutation (a count dropped below threshold; a non-locked repository named in the locked set) and reverted byte-clean.
- **`TestWorkflowRunBodiesInvokeTask` respected** — every in-scope `run:` step is exactly `task <target>`; the cache is a `uses:` step. The path-filtered sibling follows `linux-cross-canary.yml`.
- Every 40-hex string in the workflow sits on a `uses:` line (three pinned action refs). No corpus SHA is hardcoded.
- Every `go test` carried `-count=1`. No `[ci skip]` / `[skip ci]` in any commit.

## Verification

- `go test -count=1 ./internal/corpora/... ./tools/corpora/... ./internal/upgrade/...` — green.
- `go build ./...` — clean.
- `Taskfile.yml` carries `corpora:assert`, `corpora:assert-one`, `corpora:verify`, `corpora:drift`; the workflow is `.github/workflows/corpora.yml`.

## Deviations

Two upstream idle timeouts during execution. The agent's work landed in three clean commits; the final SUMMARY.md write did not complete before the second interruption, so the orchestrator wrote this summary from the commits and by independently verifying the packages, build, workflow, cache mechanism and SHA-pin discipline on the worktree. No code deviations from plan.

## Handoff — phase close

FIXT-01 and FIXT-02 are both satisfied by the locked set (hugo, guava, serilog, psf/requests — all Apache-2.0, pinned SHAs re-verified at lock time), the coverage record (`corpora/observations.json` + `corpora/selection.json` + `docs/CORPUS-MEASUREMENT.md`), and the two-legged drift guard. The phase is ready for `/gsd-verify-work 1`.
