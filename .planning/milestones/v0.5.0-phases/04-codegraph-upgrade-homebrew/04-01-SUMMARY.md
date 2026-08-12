---
phase: 04-codegraph-upgrade-homebrew
plan: 01
subsystem: internal/upgrade
tags: [homebrew, self-upgrade, filesystem-detection, security]
dependency-graph:
  requires: []
  provides:
    - detectBrewManaged
    - brewPointerMessage
    - brewInstall
  affects:
    - internal/upgrade/upgrade.go (Run())
tech-stack:
  added: []
  patterns:
    - "Early error-returning refusal inside Run() (checkWritable precedent)"
    - "Seam-based proof via existing func-var fakes (resolveLatest/download/verify/swap recorders)"
    - "filepath.EvalSymlinks before any path-shape comparison, fail-open to not-detected on error"
key-files:
  created:
    - internal/upgrade/brew.go
    - internal/upgrade/brew_test.go
  modified:
    - internal/upgrade/upgrade.go
    - internal/upgrade/upgrade_test.go
decisions:
  - "detectBrewManaged walks the resolved-path ancestry via repeated filepath.Dir (never string split-and-rejoin), so InstallDir stays absolute by construction (review cycle 1, HIGH fix)"
  - "Single brewPointerMessage() builder consumed by both the refusal (non-nil error) and the --check step-aside (nil, printed) so the two surfaces cannot drift (D-07, D-09, key_links)"
  - "opts.Force is never read inside the brew branch — its absence from that code path is the whole D-06 enforcement mechanism, not a runtime check"
metrics:
  duration: "~55 minutes"
  completed: 2026-08-11
status: complete
actuals:
  tokens: 7372
  tasks: 3
  commits: 3
---

# Phase 4 Plan 01: `codegraph upgrade` Homebrew detection Summary

Structural, prefix-agnostic detection of a Homebrew-managed Caskroom or Cellar install
(path shape + a real `INSTALL_RECEIPT.json`), wired into `upgrade.Run()` strictly before
`resolveLatest` so both the hard refusal and the `--check` step-aside are zero-network.

## What Was Built

- **`internal/upgrade/brew.go`** (new): `detectBrewManaged(targetPath string) (brewInstall, bool)`
  resolves `targetPath` through `filepath.EvalSymlinks`, walks the ancestry root-preservingly
  looking for a `Caskroom` or `Cellar` directory with at least two descendant components
  (token, then version) below it, and requires a Homebrew-authored `INSTALL_RECEIPT.json` at
  the tree-specific ancestor (`.metadata/` for Caskroom, the version directory itself for
  Cellar) before reporting detection. An `EvalSymlinks` error (dangling symlink, symlink loop)
  returns not-detected immediately — no fallback scan of the unresolved path. `brewPointerMessage`
  is the single producer of `codegraph is managed by Homebrew (<dir>). Upgrade with: brew
  upgrade codegraph`.
- **`internal/upgrade/upgrade.go`**: one new branch at the head of `Run()`, above
  `resolveLatest`. When `detectBrewManaged` reports true: under `--check` it prints the pointer
  message and returns `nil` (exit 0); otherwise it returns a non-nil error wrapping the same
  message (exit non-zero). `opts.Force` is never consulted in this branch.
- **`internal/upgrade/brew_test.go`** (new): `TestDetectBrewManaged`, a 16-row table (9
  detected: 4 prefixes × 2 trees + one symlink-reached row; 7 not-detected: no receipt ×2
  trees, receipt at the wrong ancestor, no tree segment, too-few segments, dangling symlink,
  symlink loop), guarded by row-count floors (`>=16` total, `>=7` not-detected) that fail loudly
  if a row is silently dropped. `TestDetectBrewManaged_RealInstall` is the env-gated
  (`CODEGRAPH_BREW_ACCEPTANCE_PATH`) real-install harness for plan 04-06's acceptance run.
- **`internal/upgrade/upgrade_test.go`**: three `Run()`-level tests —
  `TestUpgradeRun_RefusesBrewManagedCask` (bare upgrade refuses, all four seams unfired, exact
  anchored-regexp message match), `TestUpgradeRun_CheckBrewManagedStepsAside` (`--check` returns
  nil, prints the pointer, all four seams unfired), `TestUpgradeRun_ForceDoesNotOverrideBrewRefusal`
  (`--force` still refuses, download/swap stay uninvoked).

## Verification

- `go test ./internal/upgrade/...` — all green, including the 16/16 detection subtests and all
  five brew-related `Run()`-level tests.
- `go build ./...` and `go vet ./...` — clean.
- `task test` (unit, golden, integration, wireoracle, daemon, race) — exit 0, no regressions.
- `git diff --exit-code go.mod go.sum` — clean, no dependency added (T-04-SC).
- Row-count guard proved live: temporarily deleted the symlink-loop row, re-ran
  `TestDetectBrewManaged`, observed `row-count guard: table has 15 rows, want at least 16 (a
  row was silently dropped)`, reverted the file byte-clean via a backup copy, re-ran the full
  suite green.
- Scripted `node -e` offset assertions (not eyeball line-number comparisons, per review cycle 1
  MEDIUM) confirmed: `detectBrewManaged` sits above `resolveLatest(releaseRepoSlug)` in
  `Run()`, and the brew-branch `opts.Check` fork sits above the network-touching `opts.Check`
  block.

## Deviations from Plan

None — plan executed exactly as written, including the review-cycle-1 and review-cycle-2
dispositions already folded into the plan text (root-preserving parent walk, single
`EvalSymlinks`-error contract, scripted offset assertions, single `brewPointerMessage` call
site in `upgrade.go`).

## Known Stubs

None.

## Self-Check: PASSED

- `internal/upgrade/brew.go` — FOUND
- `internal/upgrade/brew_test.go` — FOUND
- `internal/upgrade/upgrade.go` — FOUND (modified)
- `internal/upgrade/upgrade_test.go` — FOUND (modified)
- Commit `5b3ccb3` — FOUND in `git log --oneline --all`
- Commit `b88887d` — FOUND in `git log --oneline --all`
- Commit `c9bac58` — FOUND in `git log --oneline --all`
