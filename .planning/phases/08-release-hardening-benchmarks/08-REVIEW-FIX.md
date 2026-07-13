---
phase: 08-release-hardening-benchmarks
fixed_at: 2026-07-13T22:30:00Z
review_path: .planning/phases/08-release-hardening-benchmarks/08-REVIEW.md
iteration: 1
findings_in_scope: 6
fixed: 6
skipped: 0
status: all_fixed
---

# Phase 8: Code Review Fix Report

**Fixed at:** 2026-07-13T22:30:00Z
**Source review:** .planning/phases/08-release-hardening-benchmarks/08-REVIEW.md
**Iteration:** 1

**Summary:**
- Findings in scope: 6 (3 Warning, 3 Info; fix_scope=all)
- Fixed: 6
- Skipped: 0

## Fixed Issues

### WR-01: `.goreleaser.yaml`'s `archives:`/`checksum:` blocks are dead configuration that contradicts its own header comment

**Files modified:** `.goreleaser.yaml`
**Commit:** 9b52ffa
**Applied fix:** Rewrote the file's header comment (option (a) from the review) to state plainly that `release.yml` only ever invokes `goreleaser build --single-target` — never `goreleaser release` — so the `archives:`/`checksum:` blocks never execute in this project's pipeline. The real asset-naming/checksum contract is enforced entirely by `release.yml`'s own shell steps, which must independently agree with `internal/upgrade.releaseAssetName()`. Added a matching inline note directly above the `archives:` block itself so a reader scanning the block in isolation (not just the file header) also sees the dead-config warning. Preserved the `name_template` cross-reference to the asset-naming contract shape. Verified with `goreleaser check` (passes).

### WR-02: `realcorpus.Entry.Resolve()` never verifies an existing local checkout is pinned at `CommitSHA`, and no caller checks either

**Files modified:** `tools/bench/realcorpus/manifest.go`, `tools/bench/runner/main.go`
**Commit:** 9351512
**Applied fix:** Added a `pinnedAt(dir string) string` helper in `runner/main.go` that runs `git -C dir rev-parse HEAD` and returns the trimmed SHA (or `""` on error). `resolveOrClone` now calls `pinnedAt` on any checkout path returned by `entry.Resolve()`, and only trusts it when `pinnedAt(p) == entry.CommitSHA`; otherwise it logs a warning to stderr and falls through to the existing fresh shallow-clone-at-pinned-SHA path unchanged. Updated `Entry.Resolve`'s doc comment in `manifest.go` to point at this new caller-side check instead of describing it only hypothetically. Verified with `go build ./...` and `go test ./tools/bench/realcorpus/... ./tools/bench/runner/...`.

### WR-03: `tools/bench/runner` has zero test coverage for the logic that drives both the blocking gate and the published numbers

**Files modified:** `tools/bench/runner/main_test.go` (new)
**Commit:** 0c6bc5d
**Applied fix:** Added `main_test.go` covering: `medianFloat64`/`medianInt64` for odd/even/empty/single-element inputs; `medianOfN`'s rejection of non-positive `n` and its independent-per-metric median computation; `copyTree`'s skipping of `.git`/`.codegraph` directories and non-following of symlinks; `countTree`'s file/byte counting and its exclusion of skipped directories; and `parseFlags`'s required-`-mode` error path, its defaults, and flag-override behavior. Did not attempt to test the full shell-out benchmark orchestration (`runHeadToHead`/`runRegression`), which genuinely requires real binaries per the review's own guidance. Verified with `go build ./...` and `go test ./tools/bench/runner/... -v` (all new tests pass).

### IN-01: `ci.yml`'s perf-regression job comment overstates what's actually gated

**Files modified:** `.github/workflows/ci.yml`
**Commit:** 58155f9
**Applied fix:** Tightened the header comment's `perf-regression` bullet to read "gating PERF-02 (throughput and peak-RSS tolerance bands; query latency and cold start are measured and reported, not yet gated)" instead of the previous overstated "PERF-02 (throughput/query tolerance bands)". Comment-only change; verified with `actionlint .github/workflows/ci.yml` (clean, no findings).

### IN-02: `tools/bench/runner`'s `-ts-binary` default is a macOS-Homebrew-only path

**Files modified:** `tools/bench/runner/main.go`, `tools/bench/runner/main_test.go`
**Commit:** 2f9a9fa
**Applied fix:** Changed the `-ts-binary` flag's default to the empty string. Added `resolveTSBinary()`, called from `run()` only in `headtohead` mode when `-ts-binary` wasn't set explicitly: it first tries `exec.LookPath("codegraph")` (portable across Linux/macOS/Windows), then falls back to the conventional Apple-Silicon Homebrew path (`/opt/homebrew/bin/codegraph`) if that file exists, and otherwise returns `""` so `measureSubject`'s existing "no binary configured for subject %q" error reports the failure clearly. `bench.yml` is unaffected since it always passes `-ts-binary` explicitly. Added `TestResolveTSBinary_FindsOnPath` and `TestResolveTSBinary_EmptyWhenNotFound`. Verified with `go build ./...` and `go test ./tools/bench/runner/... -v`.

### IN-03: `gencorpus`'s fixed query term assumes `goCount >= 1`, which fails for very small `-count` values

**Files modified:** `tools/bench/gencorpus/gen.go`, `tools/bench/gencorpus/gen_test.go`
**Commit:** 541df29
**Applied fix:** Chose the review's option (a): `Generate` now clamps `goCount` to at least 1 whenever `opts.FileCount >= 1` and the weighted truncation would otherwise produce 0 (this only engages for `FileCount == 1`; production's `ProductionFileCount = 120000` is unaffected). This guarantees `generateGo` always writes `pkg0000/file0000.go` defining `Fn0000_0000` — the exact symbol `tools/bench/runner`'s `regressionQueryTerm` queries against — making that constant's existing doc comment ("for any FileCount >= 1") actually true rather than a claim with an undocumented lower bound. Added `TestTinyFileCountStillWritesRootSymbol` (FileCount 1/2/3) to `gen_test.go`. Verified with `go build ./...` and `go test ./tools/bench/gencorpus/...` (full suite, including the pre-existing `TestDeterministic`/`TestFileCountExceeds100k`/`TestHasCrossFileRefs`, all pass).

## Skipped Issues

None — all findings were fixed.

---

_Fixed: 2026-07-13T22:30:00Z_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 1_
