---
phase: 09-release-please-and-goreleaser
reviewed: 2026-08-01T00:00:00Z
depth: deep
files_reviewed: 46
files_reviewed_list:
  - .github/ISSUE_TEMPLATE/bug_report.yml
  - .github/ISSUE_TEMPLATE/chore.yml
  - .github/ISSUE_TEMPLATE/config.yml
  - .github/ISSUE_TEMPLATE/enhancement.yml
  - .github/ISSUE_TEMPLATE/feature_request.yml
  - .github/PULL_REQUEST_TEMPLATE/enhancement.md
  - .github/PULL_REQUEST_TEMPLATE/feature.md
  - .github/PULL_REQUEST_TEMPLATE/fix.md
  - .github/pull_request_template.md
  - .github/workflows/auto-close-unsolicited-prs.yml
  - .github/workflows/auto-label-issues.yml
  - .github/workflows/bench.yml
  - .github/workflows/ci.yml
  - .github/workflows/close-draft-prs.yml
  - .github/workflows/pr-template-format.yml
  - .github/workflows/pr-title.yml
  - .github/workflows/release-please.yml
  - .github/workflows/release.yml
  - .github/workflows/require-issue-link.yml
  - .gitignore
  - .release-please-manifest.json
  - CHANGELOG.md
  - CODE_OF_CONDUCT.md
  - CONTRIBUTING.md
  - LICENSE
  - NOTICE
  - README.md
  - SECURITY.md
  - docs/RELEASE-PROCEDURES.md
  - docs/RELEASE.md
  - go.mod
  - go.sum
  - internal/bench/metrics.go
  - internal/bench/regression.go
  - internal/bench/regression_test.go
  - internal/daemon/daemon.go
  - internal/daemon/daemon_test.go
  - internal/daemon/lock.go
  - internal/daemon/lock_test.go
  - internal/daemon/registry_test.go
  - internal/daemon/stop_test.go
  - internal/upgrade/pr_title_lint_test.go
  - internal/upgrade/release_publish_step_test.go
  - internal/upgrade/release_workflow_shape_test.go
  - release-please-config.json
  - scripts/pr_template_policy.py
  - tools/bench/BASELINE.md
  - tools/bench/baseline.json
  - tools/bench/runner/main.go
  - tools/bench/runner/main_test.go
findings:
  critical: 0
  warning: 3
  info: 1
  total: 4
status: issues_found
---

# Phase 09: Code Review Report

**Reviewed:** 2026-08-01
**Depth:** deep
**Files Reviewed:** 46
**Status:** issues_found

## Summary

This phase wires release-please + a hand-written GoReleaser-adjacent
`release.yml` pipeline, a PR-title Conventional-Commits gate, a workflow-shape
test suite (`internal/upgrade/*_test.go`) that executes each locked workflow's
own shipped shell rather than parsing YAML as prose, a `daemon` fix for a real
process-identity defect (`StartedAt` recorded via `time.Now()` instead of the
OS-reported process start time), and a `CheckRegression` platform-mismatch
guard for the perf gate. The daemon fix and its test coverage are careful and
well-corroborated (multiple independent regression tests, including a
deliberately-forced "aged process" condition that reproduces the original
intermittent-CI failure deterministically). The workflow-shape tests are a
genuinely strong pattern: they extract and execute the real on-disk `run:`
blocks via `bash`, including an adversarial shell-metacharacter injection
case, rather than trusting that the YAML text matches what will actually run.

Two structural defects were pre-flagged as already-known and out of this
phase's diff scope: `internal/daemon`'s `getppid` watchdog test-seam data
race (issue #13, in `watchdog_posix.go`/`watchdog_test.go`, neither of which
changed in this diff) and `internal/bench.CheckRegression` not comparing
`Metrics.Repo`. The second one is confirmed present and analyzed below with
its actual (currently low, but real) blast radius.

The most concrete new defect found in this review is a documentation
consistency bug: the SLSA-provenance wording fix that PR-history describes as
"corrected 2026-08-01" (moving from "attested over the checksums file" to
"attested over each binary") was applied to some passages but not others, in
all three of the places that describe this fact — `release.yml` itself,
`docs/RELEASE.md`, and `docs/RELEASE-PROCEDURES.md` each still carry one
uncorrected copy of the old, wrong claim sitting a few lines away from the
corrected one. No runtime behavior is affected (the SLSA generic generator's
actual inputs are correct), but a reader following the uncorrected passage
alone gets exactly the wrong mental model the "Corrected" notes exist to fix.

A second, narrower finding: two `pull_request_target` workflows write a
multi-line `GITHUB_OUTPUT` value using a fixed, guessable heredoc delimiter
over attacker-fully-controlled content (file paths from a fork PR). This is a
known, if narrow-impact-here, GitHub Actions anti-pattern.

## Warnings

### WR-01: `CheckRegression` never validates corpus/subject identity, only platform — the perf gate can pass against a mismatched baseline

**File:** `internal/bench/regression.go:36-90`
**Issue:** `CheckRegression(baseline, current Metrics, ceilingBytes int64)` validates that `baseline.GOOS`/`GOARCH` match `current`'s (the recently-added, well-tested platform guard), but never compares `baseline.Repo` (or `Subject`) against `current.Repo`/`Subject`. `Metrics.Repo` encodes the corpus identity — for regression mode it is `fmt.Sprintf("synthetic-seed%d-count%d", cfg.seed, cfg.count)` (`tools/bench/runner/main.go:594`). Today the CI gate (`ci.yml`'s `perf-regression` job) and the `-rebless` job (`bench.yml`) both invoke the runner without overriding `-seed`/`-count`, so in practice `current.Repo` always equals the committed `baseline.json`'s `"synthetic-seed42-count120000"` and the gap is dormant. But nothing in `CheckRegression` itself, nor in the runner, enforces that agreement — a baseline reblessed with a different `-count` (e.g. a smaller corpus for a faster local iteration, then accidentally committed), or a future change to `regressionFileCount`/the default seed, would silently compare throughput/RSS numbers computed over two different-sized corpora as if they were the same measurement. Given this harness's own documented finding that throughput is dominated by environment (`tools/bench/BASELINE.md`'s "6.7x-148x apart" note) — corpus size is exactly the same class of confound: a 12k-file baseline vs a 120k-file current run does not produce a directly-comparable files/sec number, and the gate would gate on it anyway, either failing loudly for the wrong reason or (more dangerously) passing when the code genuinely regressed but the smaller current corpus happens to look faster.
**Fix:** Add a `Repo` (and ideally `Subject`) equality check to `CheckRegression`, symmetric with the existing GOOS/GOARCH guard — including the same "unattributed on both sides is allowed" carve-out documented for platform, so pre-Repo-field baselines and unit tests that construct bare `Metrics{}` aren't broken:
```go
if baseline.Repo != "" && current.Repo != "" && baseline.Repo != current.Repo {
	return fmt.Errorf(
		"bench: corpus mismatch: baseline was measured against %q but this run used %q; "+
			"throughput/RSS numbers from a different-sized corpus are not comparable. "+
			"Record a baseline against the same corpus (runner -mode regression -rebless) instead of comparing across them",
		baseline.Repo, current.Repo,
	)
}
```

### WR-02: SLSA-provenance "attested over the checksums file" wording was corrected in some passages but not others — three internally-contradictory copies remain

**File:** `.github/workflows/release.yml:325-331`, `docs/RELEASE.md:19-28`, `docs/RELEASE-PROCEDURES.md:113-122`
**Issue:** This same phase's own history (per `release.yml:190-202`'s "PRECISION (corrected 2026-08-01)" comment and `docs/RELEASE.md:93-102`'s "Corrected 2026-08-01" callout) fixed the claim that SLSA provenance is "generated over the checksums file" to the accurate "attested over the six binaries directly (the checksums file is only the base64-encoded transport for the subject list)". That correction was applied to the `assemble` job's comment block in `release.yml` and to one paragraph each in `docs/RELEASE.md` / `docs/RELEASE-PROCEDURES.md` — but three other passages describing the exact same fact were missed and still say the old, wrong thing, sitting only a few lines away from the corrected text in the same files:
  - `release.yml:325-331` — the `provenance:` job's own header comment: *"SLSA3 provenance via the GENERIC generator ... over the assemble job's checksums file."*
  - `docs/RELEASE.md:19-28` — §1's asset-list intro: *"SLSA3 build provenance: an `.intoto.jsonl` attestation, generated over the checksums file..."* (§1b, a few lines below, correctly says "attested over each platform binary directly").
  - `docs/RELEASE-PROCEDURES.md:113-122` — §4 step 6's job-by-job description: *"`provenance` — runs SLSA3 build provenance ... over the checksums file..."* (§6, further down the same doc, correctly says "attested over each BINARY, in one shared bundle").

  No runtime behavior is affected — the SLSA generic generator's actual `base64-subjects` input (from the `hash` step) is correct, and the corrected passages accurately describe what it does. This is a documentation-consistency defect: a reader who reads only the uncorrected passage (which is exactly as likely as reading the corrected one, since they're presented with equal prominence) walks away with the wrong verification mental model — the same wrong model that produced the `FAILED: artifact hash does not match provenance subject` failure the correction was written to prevent from recurring.
**Fix:** Update all three uncorrected passages to match the corrected wording already present elsewhere in the same files, e.g. for `release.yml:325-331`:
```diff
- # SLSA3 provenance via the GENERIC generator (not the Go builder,
- # which would rebuild the binary under its own fixed config and
- # cannot accommodate the zig-cc CGo cross-build) over the assemble
- # job's checksums file. Per slsa-github-generator's own documented
+ # SLSA3 provenance via the GENERIC generator (not the Go builder,
+ # which would rebuild the binary under its own fixed config and
+ # cannot accommodate the zig-cc CGo cross-build), attested over the
+ # six platform binaries directly — the checksums file computed above
+ # is only the base64-encoded transport for that subject list, never
+ # a subject itself (see the `assemble` job's own comment above). Per
+ # slsa-github-generator's own documented
```
and the analogous one-line fixes in `docs/RELEASE.md:26-28` and `docs/RELEASE-PROCEDURES.md:120-122`.

### WR-03: Two `pull_request_target` workflows write `GITHUB_OUTPUT` multi-line values with a fixed heredoc delimiter over attacker-controlled file paths

**File:** `.github/workflows/require-issue-link.yml:39-50`, `.github/workflows/pr-template-format.yml:35-46`
**Issue:** Both workflows collect a fork PR's changed-file paths via `gh pr view --json files --jq '.files[].path'` and write them to `$GITHUB_OUTPUT` using a fixed, predictable heredoc delimiter (`PRFILES_EOF`):
```sh
{
  echo "list<<PRFILES_EOF"
  gh pr view "$PR_NUMBER" --repo "$GITHUB_REPOSITORY" --json files --jq '.files[].path'
  echo "PRFILES_EOF"
} >> "$GITHUB_OUTPUT"
```
  In a `pull_request_target`-triggered workflow, the PR (and therefore every file name in it) is fully attacker-controlled — a fork contributor can add a file to their own PR whose repo-relative path is literally `PRFILES_EOF`. GitHub Actions' `$GITHUB_OUTPUT` file format terminates a multi-line block on the first line matching the delimiter exactly, so that filename prematurely closes the heredoc: everything after it is parsed by the Actions runtime as new, independent `key=value` lines appended to the same step's outputs, rather than as content of `list`/`files`. In these two specific workflows the practical blast radius is narrow — the corrupted/truncated output is only consumed as a whole string for prefix-matching (`case "$f" in .planning/*|docs/*|...`) or fed to `scripts/pr_template_policy.py`'s `CHANGED_FILES` env var, and neither downstream consumer currently reads an attacker-named output key from the same step — but it is a real instance of the documented "workflow command / `$GITHUB_OUTPUT` injection via untrusted heredoc delimiter" class, and it silently corrupts the changed-file list (truncating it, or attaching extra bogus output keys) rather than failing loudly, which is exactly the failure mode this review's brief calls out as "a gate passes vacuously" — a truncated `CHANGED_FILES` list changes which prefix-exemption/template-policy branch fires.
**Fix:** Use a random, non-guessable delimiter per invocation instead of a fixed string, e.g.:
```sh
delim="PRFILES_$(openssl rand -hex 16)"
{
  echo "list<<$delim"
  gh pr view "$PR_NUMBER" --repo "$GITHUB_REPOSITORY" --json files --jq '.files[].path'
  echo "$delim"
} >> "$GITHUB_OUTPUT"
```
Apply the same fix to `pr-template-format.yml`'s equivalent `files<<PRFILES_EOF` block.

## Info

### IN-01: `docs/RELEASE.md`/`docs/RELEASE-PROCEDURES.md` "Corrected" callouts read as later additions bolted onto un-updated prose rather than a single coherent rewrite

**File:** `docs/RELEASE.md:93-102`, `docs/RELEASE-PROCEDURES.md:230-237`
**Issue:** Both correction notes are well-written and clearly dated, but because the surrounding prose (see WR-02) wasn't updated in the same pass, each document now reads as "X (wrong) ... later: actually not-X (right)" rather than presenting the corrected fact once, consistently. This is the same root cause as WR-02, called out separately here because fixing WR-02 without also collapsing the "originally said / corrected to say" scaffolding back into single, consistent prose would leave the docs technically accurate but still awkward to read for a first-time verifier.
**Fix:** Once WR-02's uncorrected passages are fixed, consider folding each "Corrected 2026-08-01" callout into a single accurate paragraph (keeping a brief historical note only if there's ongoing value in warning readers who saw the old `v0.2.0`-era wording) rather than leaving two separate "this used to be wrong" annotations doing the same job in two files.

---

_Reviewed: 2026-08-01_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep_
