---
phase: quick-261011-sor
plan: 01
subsystem: dependencies
tags: [govulncheck, pnpm-audit, x-net, go-toolchain, devalue, undici, source-map-js]
requires: []
provides:
  - "govulncheck (DIST-03) clean on go1.26.9 with golang.org/x/net v0.60.0"
  - "task web:audit (BLD-06) clean: 18 advisories -> 0"
affects: [go.mod, go.sum, Taskfile.yml, web/pnpm-lock.yaml, web/build]
key-files:
  modified:
    - go.mod
    - go.sum
    - Taskfile.yml
    - test/tmux/main_test.go
    - internal/cli/plain_golden_test.go
    - web/pnpm-lock.yaml
    - web/build/
decisions:
  - "Landing ref is origin/main (PR 95 merged); commits are cherry-picked by the orchestrator onto a new fix branch"
status: complete
commits: 2
plan_head_before: 1a7fc999957fcb10da1cb7d7a4ac6f7c4e2e0f34
plan_head_after: 88b07d761535792966d0d0d9918bc950d1c1a645
actuals:
  tasks: 3
  commits: 2
---

# Quick 261011-sor: Clear the govulncheck and pnpm-audit advisories Summary

Two code commits clear the two red required checks (`govulncheck (DIST-03, blocking)` and the `pnpm audit gate (BLD-06)` step of `test`), which were failing on main itself: golang.org/x/net v0.58.0 -> v0.60.0 with Go 1.26.6 -> 1.26.9, and a web lockfile refresh of devalue, undici and source-map-js within already-declared ranges.

## Commits

| Task | SHA (full) | Subject |
| ---- | ---------- | ------- |
| 1 (Go) | `0555b5ab8434820d8cc35cb72d1e25cbb19a01fa` | fix(deps): bump golang.org/x/net to v0.60.0 and Go to 1.26.9 |
| 2 (web) | `88b07d761535792966d0d0d9918bc950d1c1a645` | fix(deps): bump devalue, undici and source-map-js in the web lockfile |

Landing tip both were proven against: `origin/main` = `65a2b9a9cac4596332a11579470e1eca99140f81` (unchanged at the end).

`git merge-tree --write-tree --merge-base <C>^ origin/main <C>` exits 0 for both:
- Go commit: tree `3f5247ec98da86325afbbf7c5491eeb1e4a111e2`
- Web commit: tree `fb886cd806879e46d79de2d3b9c656d7c86ebd91`

## RED -> GREEN

### govulncheck (CI's scan form: `go run golang.org/x/vuln/cmd/govulncheck@latest ./...`)

- Before (go1.26.6, `govulncheck-before.txt`, exit non-zero, tool status 3): 12 reachable findings. GO-2026-6617, 6613, 6612, 6611, 6610, 6609, 6608, 6607, 6605, 6603, 6600, 6599.
  - x/net: `Found in: golang.org/x/net@v0.58.0`, `Fixed in: golang.org/x/net@v0.60.0` (6617, 6612, 6611, 6610, 6603)
  - stdlib: `Found in: net/http@go1.26.6` (also net/textproto, crypto/tls, html/template), `Fixed in: ...@go1.26.9`
- After (go1.26.9, `govulncheck-after.txt`, exit 0): "No vulnerabilities found." / "Your code is affected by 0 vulnerabilities."
  - Informational only, not called by the code and not gating: GO-2026-5841 (klauspost/compress@v1.18.6, fixed v1.18.7) and GO-2026-5932 (x/crypto@v0.57.0, no fix available). Neither affects the gate and neither is in scope here.

### web:audit

- Before (`web-audit-before.txt`, non-zero): `ADVISORIES PRESENT ... reported 18 advisory(ies): 1240054 1240872 1239932 1240035 1240040 1240051 1240870 1240871 1240873 1240874 1240037 1240049 1240053 1240875 1240043 1240045 1240047 1241209`
- After (`web-audit-after.txt`, exit 0): `web:lockfile: PASS - 242 packages declared ... zero non-registry/non-link sources` and `web:audit: PASS - ... zero advisories reported`

## Details

- go.mod diff: the `go` directive (1.26.6 -> 1.26.9) plus exactly eight golang.org/x modules: crypto v0.55.0->v0.57.0, mod v0.38.0->v0.41.0, net v0.58.0->v0.60.0, sync v0.22.0->v0.23.0, sys v0.47.0->v0.48.0, term v0.45.0->v0.46.0, text v0.41.0->v0.42.0, tools v0.48.0->v0.49.0. Matches the planning probe exactly. No `toolchain` line, no `go mod tidy`. go.sum numstat `16 0`.
- Pins: six GOTOOLCHAIN occurrences (4 Taskfile.yml, 2 test comments) all go1.26.9; numstat Taskfile 4/4, plain_golden_test 1/1, tmux main_test 2/2. Merged tree on origin/main shows four go1.26.9 pins in Taskfile.yml.
- Web versions resolved: devalue 5.9.1 -> 5.9.4, undici 8.10.0 -> 8.11.2, source-map-js 1.2.1 -> 1.2.2 (identical to planning). Lockfile diff names only those three packages. `web/package.json` and `web/pnpm-workspace.yaml` byte-unchanged. Override fallback did NOT run.
- web:drift branch taken: SOURCE-half mismatch (marker 311c9308... vs recomputed 28bd3956...), output half matched. `task web:build` regenerated `web/build/` (wrote manifest: 119 source files, 36 output files); re-run drift PASS; `go test ./web/... ./internal/uiserver/...` ok. The 17-file web commit therefore contains lockfile + regenerated build tree (hash-named assets replaced, three deleted/re-added JS/CSS names are expected).

## Gate results

| Gate | Result |
| ---- | ------ |
| `task build`, `task vet`, `go vet -tags tmux ./test/tmux/...` | pass (go list printed `codegraph-go/test/tmux`) |
| `task lint:go` | `0 issues.` |
| `task test:unit` (Go commit, retry) | exit 0, 53 `ok` lines |
| `task test:unit` (final tree) | exit 0, 53 `ok` lines incl. internal/uiserver |
| `task test:daemon` | exit 0, `ok internal/daemon 64.047s` |
| `task docs:cli:drift` | "compared 1 generated file" + "byte-identical to a fresh regeneration" |
| `task web:deps` / `web:deps:strict` | frozen-lockfile succeeded, lockfile unchanged / PASS |
| `task web:test` | PASS, 585 of 585 |
| direct `pnpm exec vitest run` | exit 0, 49 files / 585 tests passed, zero unhandled-error report lines |
| `task web:build:verify`, `task web:drift` (post-commit) | PASS, PASS |
| Task 1 automated verify (L=origin/main) | PASS |
| Task 2 automated verify (L=origin/main, one adjustment below) | PASS |
| Task 3 audit | PASS (branch/parent/landing/merge-tree/clean tree) |

## Deviations from Plan

### Orchestrator-directed substitutions (premise change: PR 95 merged)

1. Landing ref `origin/ci/fragment-required-gate` -> `origin/main` everywhere (`L=`, merge-tree proofs, basis re-verify, `landing.sha`). Fetch was `git fetch origin main`; basis check `git diff --quiet HEAD origin/main -- go.mod go.sum test/tmux/main_test.go internal/cli/plain_golden_test.go web/` exited 0 and origin/main's Taskfile.yml carries 4 GOTOOLCHAIN pins.
2. "PR #95" as cherry-pick destination read as "a new fix PR to main, opened by the orchestrator". No PR or issue numbers appear in either commit message.

### Auto-handled

3. **[Rule 1 - flake] `TestLiveEditAutoSyncReachesExplore` failed once under `task test:unit`.** First run (Go commit) failed in `test/integration` with `codegraph_explore returned a tool error during a live watch session (CR-01 store-lock collision?)`; output kept as `test-unit-attempt1.txt`. The test passed 3 of 3 in isolation (`go test -run TestLiveEditAutoSyncReachesExplore ./test/integration/`) and the full `task test:unit` re-run passed (53 ok, exit 0), as did the final-tree run. A Pebble store-lock collision under parallel load, unrelated to a dependency bump of this shape; not currently in STATE.md's documented flakes. Recommend adding it to the flake list.
4. **Plan verify defect, vitest unhandled-error check (Task 2).** The plan's `rg -i 'unhandled (error|rejection)'` count of 0 false-positives on two PASSING test titles ("degrades to idle on a rejected getEditorLink promise, no unhandled rejection", "a rejected navigation never becomes an unhandled rejection (IN-09)"). Verified by hand that vitest emitted no `Unhandled Errors` report, then ran the verify with passing-test lines (`^\s*✓`) excluded; it PASSED. No file changed.
5. Evidence directory stayed `/tmp/261011-sor` because the plan's verify commands reference it.
6. Plan ledger file for `plan_head_before` was not created; the base is the recorded `start.sha` (`1a7fc999957fcb10da1cb7d7a4ac6f7c4e2e0f34`), and `git log start..HEAD` over non-.planning paths yields exactly the two commits.

## Follow-ups for the orchestrator (NOT done here)

- STATE.md carry-over still tells local gates to pin the old 1.26 patch toolchain; it should now say `GOTOOLCHAIN=go1.26.9`.
- `scripts/live-push-concurrency-check.sh` pins go1.26.5, which go.mod has refused since its directive moved past 1.26.5 (pre-existing, untouched).
- No changie fragment was authored: `.changes/` is outside this task's scope and the cherry-pick basis; whether the new fix PR needs one under the now-landed fragment-required gate is the orchestrator's call. (On main the gate now exists, so this deserves a deliberate look.)
- Dependabot web PRs are superseded but untouched.
- Informational non-gating govulncheck findings: klauspost/compress GO-2026-5841 (fix v1.18.7) and x/crypto GO-2026-5932 (no fix). Consider a follow-up bump for compress.
- Nothing was pushed. `.boost/` and `.planning/` were not staged.

## Known Stubs

None.

## Self-Check: PASSED

- Commits 0555b5ab and 88b07d76 are ancestors of HEAD (`git log` shows both on gsd/v0.15.0-milestone after start.sha).
- Tracked tree clean; untracked only `.boost/` and the quick-task directory.
