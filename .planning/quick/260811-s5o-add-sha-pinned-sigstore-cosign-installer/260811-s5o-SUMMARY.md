---
quick_id: 260811-s5o
slug: add-sha-pinned-sigstore-cosign-installer
status: complete
completed: 2026-08-11
files_modified:
  - internal/upgrade/taskfile_shape_test.go
  - .github/workflows/post-release-verify.yml
commits:
  - 3cd6fd6
  - 6135785
---

# Add the SHA-pinned cosign-installer step to post-release-verify.yml's `self-upgrade` job — Summary

Guarded and fixed the missing `sigstore/cosign-installer` step in the `self-upgrade` job that caused release v0.9.0 (run 31549287269) to fail closed on both `self-upgrade proof` legs with `task: cosign not found ... precondition not met`.

## What shipped

**Task 1 (RED):** `internal/upgrade/taskfile_shape_test.go` gained `TestCosignRequiringJobsInstallCosign` and its non-vacuity companion `TestCosignInstallerGuardParsersFailLoudly`. The guard derives — at runtime, never hardcoded — every Taskfile target with a `command -v cosign` precondition and every job in `post-release-verify.yml`, then asserts each cosign-requiring job installs cosign via `sigstore/cosign-installer` exactly once, with one uniform SHA+version-comment pin across the file. Three new pieces of reusable machinery: `parseCosignRequiringTaskTargets`, `parseWorkflowJobsAllSteps`, `taskTargetsInRunBody`, plus a raw-source pin regex `cosignInstallerPinRe` and one additive field (`Uses`) on the existing `workflowRunStep` struct.

**Task 2 (GREEN):** `.github/workflows/post-release-verify.yml`'s `self-upgrade` job now installs cosign between `Set up Go` and `Install Task`, byte-identical to the two sibling jobs (`verify-supply-chain`, `notarized-suite`) — same SHA (`6f9f17788090df1f26f669e9d70d6ae9567deba6`), same `# v4.1.2` comment. `Taskfile.yml`'s `command -v cosign` precondition on `verify:self-upgrade` was not touched — it was correct and load-bearing; it failed closed exactly as designed.

## RED confirmation (verbatim)

```
=== RUN   TestCosignRequiringJobsInstallCosign
    taskfile_shape_test.go:2434: job "notarized-suite" invokes cosign-requiring target(s) [verify:notarized-suite], installer step count=1
    taskfile_shape_test.go:2434: job "self-upgrade" invokes cosign-requiring target(s) [verify:self-upgrade], installer step count=0
    taskfile_shape_test.go:2437: job "self-upgrade" invokes cosign-requiring target(s) [verify:self-upgrade] but declares 0 sigstore/cosign-installer step(s) (want exactly 1) — a job whose run: body invokes a task target with a 'command -v cosign' precondition must install cosign exactly once
    taskfile_shape_test.go:2434: job "verify-supply-chain" invokes cosign-requiring target(s) [verify:release-assets], installer step count=1
--- FAIL: TestCosignRequiringJobsInstallCosign (0.00s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/upgrade	0.235s
FAIL
```

Full package run at that point showed only this one test red — no other test in `./internal/upgrade/` changed status. `TestCosignInstallerGuardParsersFailLoudly` passed at the same commit, proving the parsers fail loudly on broken/empty input rather than passing vacuously.

## GREEN confirmation

```
=== RUN   TestCosignRequiringJobsInstallCosign
    taskfile_shape_test.go:2434: job "notarized-suite" invokes cosign-requiring target(s) [verify:notarized-suite], installer step count=1
    taskfile_shape_test.go:2434: job "self-upgrade" invokes cosign-requiring target(s) [verify:self-upgrade], installer step count=1
    taskfile_shape_test.go:2434: job "verify-supply-chain" invokes cosign-requiring target(s) [verify:release-assets], installer step count=1
--- PASS: TestCosignRequiringJobsInstallCosign (0.00s)
PASS
ok  	github.com/seanb4t/codegraph-go/internal/upgrade	0.151s
```

Positive-run-count check (per repo rule 84d1gfpywd — `go test -run` can print `ok` with zero tests matched): `=== RUN` lines counted via `rg -o '^=== RUN' | wc -l` = **1**. The test actually ran, not skipped.

`go test ./internal/upgrade/... -count=1` — full package green.
`task lint:actions` — clean, no output beyond the task invocation line.

Positive file-content assertions on `post-release-verify.yml`:
- `sigstore/cosign-installer@[0-9a-f]{40}` occurrence count = 3 (`verify-supply-chain`, `self-upgrade`, `notarized-suite`)
- distinct `SHA # vX.Y.Z` pin strings = 1
- exact pin `sigstore/cosign-installer@6f9f17788090df1f26f669e9d70d6ae9567deba6 # v4.1.2` present

`git diff .github/` confirmed exactly one added `Install cosign` step plus its blank separator line — nothing else changed.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Added `(?m)` flag to `cosignInstallerPinRe`**
- **Found during:** Task 1, writing `cosignInstallerPinRe`
- **Issue:** The plan's literal regex (`` uses:\s*(sigstore/cosign-installer@[0-9a-f]{40})\s+#\s*(v\d+\.\d+\.\d+)\s*$ ``) anchors `$` at end-of-text without the multiline flag. Go's `regexp` package only matches `$` at end-of-line when `(?m)` is set; without it, `$` matches only the absolute end of the string (or immediately before a final trailing newline). Since the workflow file has two (later three) installer pins each followed by more content, `FindAllStringSubmatch` without `(?m)` would find at most one match near the end of the file, making the pin-uniformity assertion (`match count == installer step count`) fail even on a correctly fixed file.
- **Fix:** Added `(?m)` as a regex flag prefix: `` (?m)uses:\s*(sigstore/cosign-installer@[0-9a-f]{40})\s+#\s*(v\d+\.\d+\.\d+)\s*$ ``. Verified: GREEN run correctly found `len(shas)==1` and `len(versions)==1` across the three occurrences.
- **Files modified:** `internal/upgrade/taskfile_shape_test.go`
- **Commit:** 3cd6fd6

## Known Stubs

None.

## Threat Flags

None — this change only adds a step invoking an already-vetted, already-pinned third-party action (`sigstore/cosign-installer`) a second/third time in an existing workflow; no new surface.

## Self-Check: PASSED

- FOUND: `.github/workflows/post-release-verify.yml` (modified)
- FOUND: `internal/upgrade/taskfile_shape_test.go` (modified)
- FOUND commit 3cd6fd6 (`git log --oneline --all | grep 3cd6fd6`)
- FOUND commit 6135785 (`git log --oneline --all | grep 6135785`)
