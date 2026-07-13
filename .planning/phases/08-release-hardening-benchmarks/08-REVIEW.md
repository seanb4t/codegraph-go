---
phase: 08-release-hardening-benchmarks
reviewed: 2026-07-13T00:00:00Z
depth: deep
files_reviewed: 18
files_reviewed_list:
  - .github/workflows/release.yml
  - .github/workflows/ci.yml
  - .github/workflows/bench.yml
  - .goreleaser.yaml
  - internal/bench/rss.go
  - internal/bench/rss_test.go
  - internal/bench/metrics.go
  - internal/bench/regression.go
  - internal/bench/regression_test.go
  - internal/upgrade/upgrade.go
  - internal/upgrade/verify_release_e2e_test.go
  - tools/bench/gencorpus/gen.go
  - tools/bench/gencorpus/gen_test.go
  - tools/bench/gencorpus/main.go
  - tools/bench/runner/main.go
  - tools/bench/realcorpus/manifest.go
  - tools/bench/realcorpus/manifest_test.go
  - tools/bench/baseline.json
findings:
  critical: 0
  warning: 3
  info: 3
  total: 6
status: issues_found
---

# Phase 8: Code Review Report

**Reviewed:** 2026-07-13
**Depth:** deep
**Files Reviewed:** 18
**Status:** issues_found

## Summary

This is a re-review of the release-hardening/benchmark phase after the prior background review's GitHub-Actions command-injection fix (commit `2a5aa2d`, env-indirection for `github.ref_name`). I re-verified that fix and hunted for the same class of defect across all three workflows, cross-checked every third-party Action pin against the real upstream repo (via `gh api`, dereferencing annotated tags), traced the full upgrade crypto call chain (`upgrade.go` → `release.go` → `verify.go` → `swap.go`), re-derived the release asset naming contract across `.goreleaser.yaml` / `release.yml` / `internal/upgrade`, and read every shell-out surface in `tools/bench/*` for injection, path-traversal, and silent-gate-defeat risk.

**No Critical/security-blocking defects found.** Specifically:
- The command-injection class flagged previously is fully fixed and does not recur anywhere else: no `${{ }}` expression is spliced directly into any `run:` block body in any of the three workflows (verified programmatically, not just by inspection) — every workflow-context value crosses into shell via `env:` indirection.
- Every third-party Action pin resolves to a real commit (cross-checked live via `gh api`, including peeling annotated tags), and the SLSA generator is the one documented exception pinned by full semver tag as its own tooling requires.
- `permissions:` blocks are minimal and job-scoped (`contents: read` by default; `id-token: write`/`contents: write` only on the `assemble`/`provenance` jobs that actually need OIDC signing / release-asset upload).
- The upgrade verifier's identity policy (`releaseOIDCIssuer`, anchored `releaseWorkflowRefPattern`, per-binary sha256 digest match) is correct, fully anchored (`^...$`), and a dedicated regression test (`TestReleaseWorkflowRefPattern_RejectsNonReleaseWorkflowInSameRepo`) already proves the previously-fixed prefix-match vulnerability (WR-08) stays fixed. `verifyRelease`'s error is always fatal and `Run` never falls through to swap on a verify error (proven at both the unit and `Run`-orchestration level).
- `atomicSwap`/`swapWindows` leave the original binary intact on every failure path, including the tested Windows-specific rename-aside recovery.
- `CheckRegression`'s tolerance-band arithmetic and the independent absolute-RSS-ceiling check are correct at the tested boundaries, and re-blessing the baseline is gated behind an explicit `-rebless` flag that's the only code path that writes `baseline.json` — no accidental auto-rewrite.
- `rss.go`'s Linux-KB/Darwin-bytes normalization is correct and any other `GOOS` fails loudly rather than silently misreporting.
- `TestVerifyReleaseE2E` genuinely skips only because no real signed artifact/fixture exists yet (confirmed by running it) — it cannot silently mask a real failure.

I did find three **Warning**-level issues worth fixing before this phase is considered fully hardened, and three minor **Info**-level notes. None are blocking, but the first Warning in particular contradicts this exact file's own stated contract and should be corrected so a future maintainer doesn't chase a naming bug in the wrong file.

## Warnings

### WR-01: `.goreleaser.yaml`'s `archives:`/`checksum:` blocks are dead configuration that contradicts its own header comment

**File:** `.goreleaser.yaml:6-22, 118-129`
**Issue:** The file's own header comment asserts: *"archives.name_template below MUST produce the exact same string as internal/upgrade.releaseAssetName() ... or `codegraph upgrade` 404s"*. But `release.yml` never invokes `goreleaser release` (or GoReleaser Pro's `continue --merge`) — it only ever runs `goreleaser build --single-target --clean` (see `release.yml:132`). Per GoReleaser's own documentation, the `build` subcommand "compile[s] only the project's binaries, without generating release artifacts" — the archive, checksum, sign, and SBOM pipes only run under `goreleaser release`. That means the `archives:` and `checksum:` blocks in this file **never execute** in this project's actual pipeline. The real naming/checksum contract is entirely enforced by hand-rolled bash: `release.yml`'s "Rename to release-asset contract name" step (build job) independently reconstructs `codegraph_<tag>_<goos>_<goarch>[.exe]`, and the `assemble` job's "Checksums" step independently runs `sha256sum` to produce `codegraph_<tag>_checksums.txt`. The two happen to agree today (proven only by `TestReleaseAssetNameMatchesGoReleaser`, which is itself a hand-transcribed reproduction of the YAML template in Go, not an execution of the template), but nothing in the codebase would catch a future divergence between the YAML template and the bash script, because the YAML template is inert.
**Fix:** Either (a) delete the `archives:`/`checksum:` blocks and rewrite the header comment to state plainly that GoReleaser is used purely as a native cross-compiling `build` tool and the asset-naming/checksum contract lives entirely in `release.yml`'s shell steps, or (b) actually invoke `goreleaser release --single-target`-equivalent per platform so the config is load-bearing again. Given the project's own documented reason for the manual-bash approach (native darwin builds via `goreleaser build --single-target` across two runners, no GoReleaser Pro split/merge), option (a) — fixing the comment to match reality — is the lower-risk change:
```yaml
# NOTE: goreleaser build --single-target (the only command release.yml invokes)
# does NOT run the archive/checksum/sign/sbom pipes -- those only run under
# `goreleaser release`. The archives/checksum blocks below are therefore
# inert under this project's actual pipeline and exist only as
# documentation of intent; the real naming/checksum contract is enforced by
# release.yml's own shell steps ("Rename to release-asset contract name",
# "Checksums"). If you change the asset name shape, change BOTH
# release.yml's DEST= line and internal/upgrade.releaseAssetName() --
# editing this file's archives.name_template alone has no effect.
```

### WR-02: `realcorpus.Entry.Resolve()` never verifies an existing local checkout is pinned at `CommitSHA`, and no caller checks either

**File:** `tools/bench/realcorpus/manifest.go:164-181`, `tools/bench/runner/main.go:362-366`
**Issue:** `Resolve()`'s doc comment explicitly says: *"Resolve does not verify the checkout is actually pinned at e.CommitSHA ... callers that need that guarantee ... should check it themselves."* But `resolveOrClone` in `tools/bench/runner/main.go` calls `entry.Resolve()` and, on success, uses the returned path directly with no verification step at all — the "callers that need that guarantee" caveat is never actually exercised anywhere in this codebase. If an operator (or a stale CI cache/self-hosted runner) has an existing `../weft`, `../codegraph-ts`, or `../pebble` sibling checkout sitting at a different commit than `entry.CommitSHA` (e.g. `git pull`ed since the pin was last bumped), `resolveOrClone` silently benchmarks the wrong commit and `bench.yml` publishes a PERF-01 number for a repo state that doesn't match the manifest's documented provenance — defeating the entire point of `realcorpus`'s package doc ("every entry MUST carry a commit SHA ... so a published head-to-head number stays reproducible run-to-run and machine-to-machine").
**Fix:** Verify the resolved checkout's `HEAD` SHA before trusting it, falling through to a fresh pinned clone otherwise:
```go
func resolveOrClone(entry realcorpus.Entry, scratchRoot string) (string, error) {
	if p, err := entry.Resolve(); err == nil {
		if pinnedAt(p) == entry.CommitSHA {
			return p, nil
		}
		fmt.Fprintf(os.Stderr, "runner: %s checkout at %s is not pinned at %s; cloning fresh\n", entry.Name, p, entry.CommitSHA)
	}
	// ... existing clone path
}

func pinnedAt(dir string) string {
	out, err := exec.Command("git", "-C", dir, "rev-parse", "HEAD").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
```

### WR-03: `tools/bench/runner` has zero test coverage for the logic that drives both the blocking gate and the published numbers

**File:** `tools/bench/runner/main.go` (whole file, ~700 lines)
**Issue:** `go test ./tools/bench/runner/...` reports "no test files". This package contains `medianOfN`/`medianFloat64`/`medianInt64` (the median-of-5 computation the phase's own D-05 decision depends on), `copyTree`/`countTree` (symlink-skip and `skipDirs` logic that determines what actually gets measured), and `runRegression`/`runHeadToHead` (the orchestration that decides pass/fail for the blocking PERF-02/INDX-06 CI gate and the numbers published to `BENCHMARKS.md`). Every other package touched in this phase (`internal/bench`, `tools/bench/gencorpus`, `tools/bench/realcorpus`) has direct unit tests for its pure-logic core; this is the one package whose pure-logic helpers (`medianFloat64`/`medianInt64`/`copyTree`/`countTree`) are entirely untested, despite being exactly the kind of arithmetic/file-walk logic where an off-by-one or wrong `skipDirs` entry would silently corrupt every downstream metric without a single test catching it.
**Fix:** Add a `main_test.go` (or split the pure helpers into a small internal package) covering at minimum: `medianFloat64`/`medianInt64` even/odd-length inputs, `copyTree` skipping `.git`/`.codegraph` and not following symlinks, and `countTree`'s byte/file counting against a small fixture tree.

## Info

### IN-01: `ci.yml`'s perf-regression job comment overstates what's actually gated

**File:** `.github/workflows/ci.yml:20-24`
**Issue:** The header comment says the job gates "PERF-02 (throughput/query tolerance bands)", but `internal/bench.CheckRegression` (`internal/bench/regression.go:33-65`) only compares `FilesPerSec` and `PeakRSSBytes` against tolerance bands — `QueryLatencyMedianMS` and `ColdStartMS` are measured and printed (`tools/bench/runner/main.go:450-478`) but never compared against any baseline or tolerance. This matches the phase's own scoped design in `08-06-PLAN.md`/`08-06-SUMMARY.md` (which describe PERF-02 as throughput+RSS only), so this is not a functional gap — just a comment that promises more than the gate delivers.
**Fix:** Tighten the comment to "throughput and peak-RSS tolerance bands (query latency and cold start are measured and reported, not yet gated)".

### IN-02: `tools/bench/runner`'s `-ts-binary` default is a macOS-Homebrew-only path

**File:** `tools/bench/runner/main.go:130`
**Issue:** `fs.StringVar(&cfg.tsBinary, "ts-binary", "/opt/homebrew/bin/codegraph", ...)` hardcodes an Apple-Silicon Homebrew prefix as the default. `bench.yml` always overrides this explicitly (`-ts-binary "$(command -v codegraph)"`), so CI is unaffected, but anyone running `go run ./tools/bench/runner -mode headtohead` directly on Linux or Intel macOS gets a silently-wrong default that will fail with a confusing "not runnable" error rather than a clear "you must pass -ts-binary" message.
**Fix:** Default to empty string and rely on the existing `measureSubject` error path ("no binary configured for subject %q") to give a clearer message, or resolve via `exec.LookPath("codegraph")` at runtime.

### IN-03: `gencorpus`'s fixed query term assumes `goCount >= 1`, which fails for very small `-count` values

**File:** `tools/bench/gencorpus/gen.go:94`, `tools/bench/runner/main.go:75`
**Issue:** `regressionQueryTerm = "Fn0000_0000"` is documented as always present "for any FileCount >= 1", but `goCount := int(float64(opts.FileCount) * goWeight)` truncates to 0 for `FileCount` values below `~2` (e.g. `-count 1` gives `goCount = int(0.85) = 0`, so `generateGo` returns immediately and never writes `pkg0000/file0000.go`). Production always uses `regressionFileCount = 120000` so this never fires in CI, but an operator manually re-running `go run ./tools/bench/runner -mode regression -count 1` for a quick local smoke test gets a query measured against a symbol that doesn't exist, with no error — just a near-zero/meaningless `QueryLatencyMedianMS`.
**Fix:** Either have `generateGo` guarantee at least one Go file regardless of weighting for tiny counts, or note the `FileCount >= 1` claim's actual lower bound (`>= ~12`, the smallest count where `int(0.85*n) >= 1`) in the doc comment.

---

_Reviewed: 2026-07-13_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep_
