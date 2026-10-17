---
phase: 01-changie-baseline
verified: 2026-09-25T00:00:00Z
status: passed
score: 4/4 must-haves verified
covered_files: [".changes/header.tpl.md", ".changes/unreleased/.gitkeep", ".changie.yaml", ".github/workflows/ci.yml", ".planning/REQUIREMENTS.md", ".planning/ROADMAP.md", ".planning/phases/01-changie-baseline/01-01-PLAN.md", ".planning/phases/01-changie-baseline/01-01-SUMMARY.md", ".planning/phases/01-changie-baseline/01-02-PLAN.md", ".planning/phases/01-changie-baseline/01-02-SUMMARY.md", ".planning/phases/01-changie-baseline/01-CONTEXT.md", ".planning/phases/01-changie-baseline/01-MUTATION-LOG.md", ".planning/phases/01-changie-baseline/01-RESEARCH.md", "CONTRIBUTING.md", "Taskfile.yml", "go.tool-changie.mod", "go.tool-changie.sum", "internal/upgrade/changie_shape_test.go", "internal/upgrade/taskfile_shape_test.go"]
covered_digest: "v1:sha256:c9783588170ac4f2295881c2f58728295f0d4f611cd022be1091884144f23c5b"
behavior_unverified: 0
overrides_applied: 0
---

# Phase 1: Changie Baseline Verification Report

**Phase Goal:** The repository has a changie configuration and baseline that answers `v0.14.0` as the latest version and reproduces today's `CHANGELOG.md` exactly, so every later phase writes fragments against a fixed vocabulary and no history is rewritten.
**Verified:** 2026-09-25
**Status:** passed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths (ROADMAP Success Criteria)

| # | Truth (Success Criterion) | Status | Evidence |
|---|---|---|---|
| 1 (CHG-03) | `changie latest`/`next auto`/`merge --dry-run` run as a committed Taskfile target wired into CI, byte-reproduction demonstrated RED against a one-byte mutation and reverted | ✓ VERIFIED | Ran `task check:changie` live: all 11 legs pass, ends `11 of 11 checks passed against a scratch copy`. `.github/workflows/ci.yml` line 212 runs `task check:changie` immediately after `task docs:cli:drift` (line 203), confirmed by `TestChangieCheckWiredIntoCI` (PASS). `01-MUTATION-LOG.md` Family (a) shows a real RED transcript (`::error::check:changie: [3/11] ... does not reproduce CHANGELOG.md byte-for-byte`) from a live one-byte mutation in a disposable git worktree, followed by a byte-clean revert and GREEN re-run — verified present and internally consistent, worktree teardown confirmed (`git worktree list` shows only the main checkout). |
| 2 (CHG-01) | `.changie.yaml` declares exactly the five kinds with correct `auto` bumps and flip note, required int `PR` with `minInt: 1`, design-note formats; changie pinned ≥v1.26.0 with one recorded path for CI and contributors | ✓ VERIFIED | Read `.changie.yaml` directly: kinds Breaking(minor, `# pre-1.0: breaking bumps minor. Flip to major at 1.0.`)/Features(minor)/Fixes(patch)/Performance(patch)/Dependencies(patch); `custom: [{key: PR, type: int, minInt: 1}]` with no `optional` key; `versionFormat`/`kindFormat`/`changeFormat` match the design note's raw strings (`.planning/notes/changie-release-management.md` lines 79-81) plus permitted additions (`fragmentFileFormat`, `newlines.afterChangelogHeader: 1`). `TestChangieConfigShape` PASSES (exact key-set assertion). changie pinned at v1.26.0 in `go.tool-changie.mod` (confirmed via `go list -m -modfile=go.tool-changie.mod` → `v1.26.0`, `go mod verify` → `all modules verified`). One recorded path: `task changie` (Taskfile.yml wrapper), documented in `go.tool-changie.mod`'s header and in `CONTRIBUTING.md`'s tool bullet (confirmed by direct read — "changie runs as `task changie`"). |
| 3 (CHG-02) | `.changes/header.tpl.md`, `.changes/unreleased/.gitkeep`, `.changes/v0.14.0.md` exist; every existing `CHANGELOG.md` entry below the changie header is byte-identical to `main` | ✓ VERIFIED | All three files exist on disk (`ls .changes/`). Reassembly check run directly: `{header.tpl.md + \n + 14 seeds newest-first} | cmp - CHANGELOG.md` → byte-identical (no diff, exit 0). `git diff --quiet main -- CHANGELOG.md` → exit 0 (no drift from `main`). `git ls-files -- .changes/unreleased` prints exactly `.changes/unreleased/.gitkeep` (no fragment committed). `TestChangieBaselineLayout` and `TestChangieVersionSeedsMatchChangelog` PASS (14 seeds, 14 headings, set-equal both directions). |
| 4 (CHG-04) | `CI=true changie new -k <Kind> -b "<sentence>" -m PR=<n>` writes a fragment without prompting; missing-PR and undeclared-kind fragments are each refused, refusals confirmed to have executed (not inferred from a green exit) | ✓ VERIFIED | Live `task check:changie` run: leg [4/11] writes exactly one fragment non-interactively; legs [5/11]–[8/11] assert non-zero exit AND changie's own stderr substrings for missing PR (`custom missing and prompt is disabled: custom key 'PR'`), undeclared kind (`invalid kind: Undeclared`), PR below minInt (`input below minimum: 0 < 1`), and non-integer PR (`invalid number`) — text-based assertions, not bare exit codes (satisfies rule `84d1gfpywd`). `01-MUTATION-LOG.md` Family (b) shows the missing-PR refusal actually has teeth: with `optional: true` injected into `.changie.yaml`, the refusal leg goes RED (changie silently accepts the fragment), proving the guard is not vacuous. |

**Score:** 4/4 truths verified (0 present-but-behavior-unverified)

### Required Artifacts

| Artifact | Expected | Status | Details |
|---|---|---|---|
| `.changie.yaml` | Locked kinds/auto/PR/format vocabulary | ✓ VERIFIED | Exists, exact key set, `TestChangieConfigShape` PASS |
| `.changes/header.tpl.md` | Exactly `# Changelog\n` | ✓ VERIFIED | `printf '# Changelog\n' | cmp -` confirmed via reassembly check |
| `.changes/unreleased/.gitkeep` | Tracks empty unreleased dir | ✓ VERIFIED | Present, only tracked file in that dir |
| 14 `.changes/v*.md` seeds (v0.2.0–v0.14.0) | Verbatim byte slices of CHANGELOG.md | ✓ VERIFIED | All 14 present; reassembly `cmp` byte-identical to CHANGELOG.md |
| `go.tool-changie.mod` / `.sum` | Isolated pin of changie v1.26.0 | ✓ VERIFIED | `go list -m` → v1.26.0; `go mod verify` → all modules verified; registered in `isolatedModfilePaths` and `forbiddenToolPackages` |
| `internal/upgrade/changie_shape_test.go` | 8 TestChangie* guards | ✓ VERIFIED | All 8 tests present and PASS (`TestChangieConfigShape`, `TestChangieBaselineLayout`, `TestChangieVersionSeedsMatchChangelog`, `TestChangieToolPinnedInIsolatedModfile`, `TestChangieWrapperTaskRecordsInstallPath`, `TestChangieShapeParsersFailLoudly`, `TestChangieCheckWiredIntoCI`, `TestChangieBinaryInToolVulnScan`) |
| `Taskfile.yml` (`changie`, `check:changie`, `vuln`) | Wrapper, 11-leg live guard, vuln scan extension | ✓ VERIFIED | All three present; `task check:changie` runs live and passes 11/11; `task vuln` builds and scans changie (`changie: CLEAN`) |
| `.github/workflows/ci.yml` | `check:changie` step after `docs:cli:drift` | ✓ VERIFIED | Line 212 (`task check:changie`) immediately follows line 203 (`task docs:cli:drift`); `task lint:actions` clean |
| `CONTRIBUTING.md` | Tool bullet names changie/go.tool-changie.mod/`task changie` | ✓ VERIFIED | Confirmed by direct read; scoped to the tool bullet, `## Pull requests` untouched |
| `.planning/phases/01-changie-baseline/01-MUTATION-LOG.md` | 4 RED-demonstration families | ✓ VERIFIED | All 4 families present with pre-mutation gate, RED transcript, byte-clean revert, GREEN re-run; Family (d) documents an honest deviation (floor-vs-set-mismatch ordering) rather than hiding it |

### Key Link Verification

| From | To | Via | Status | Details |
|---|---|---|---|---|
| `Taskfile.yml` (`changie` task) | `go.tool-changie.mod` | `GO_TOOL_CHANGIE` var | ✓ WIRED | `GO_TOOL_CHANGIE: GOWORK=off go tool -modfile=go.tool-changie.mod` present and used in `changie` task's `cmds` |
| `.changie.yaml` | `.changes/header.tpl.md` | `headerPath` key | ✓ WIRED | `headerPath: header.tpl.md` present; live `changie merge --dry-run` renders it correctly |
| `.changes/v*.md` | `CHANGELOG.md` | `changie merge` byte identity | ✓ WIRED | Live `task changie -- merge --dry-run | cmp - CHANGELOG.md` exits 0 |
| `ci.yml` (`test` job) | `Taskfile.yml check:changie` | `run: task check:changie`, immediately after `docs:cli:drift` | ✓ WIRED | Confirmed by direct file read and `TestChangieCheckWiredIntoCI` PASS |
| `Taskfile.yml vuln` | `go.tool-changie.mod` | build + govulncheck scan | ✓ WIRED | `task vuln` output includes `changie: CLEAN`; `TestChangieBinaryInToolVulnScan` PASS |
| `internal/upgrade/taskfile_shape_test.go` (`isolatedModfilePaths`, `forbiddenToolPackages`) | `go.tool-changie.mod` | `changieModfilePath` const | ✓ WIRED | Confirmed via `rg` on both fixture arrays |

### Behavioral Spot-Checks (live commands run by verifier, not summary claims)

| Behavior | Command | Result | Status |
|---|---|---|---|
| `changie latest` answers baseline | `task changie -- latest` | `v0.14.0` | ✓ PASS |
| Merge dry-run byte-identical | `task changie -- merge --dry-run \| cmp - CHANGELOG.md` | exit 0, no output | ✓ PASS |
| CHANGELOG.md unchanged vs main | `git diff --quiet main -- CHANGELOG.md` | exit 0 | ✓ PASS |
| Empty-unreleased refusal | `task changie -- next auto` | stderr `no unreleased changes found for automatic bumping`, exit 201 (task wraps changie's exit 1) | ✓ PASS |
| Full 11-leg live guard | `task check:changie` | `11 of 11 checks passed against a scratch copy (source tree byte-unchanged)` | ✓ PASS |
| Source tree unmutated after guard run | `git status --porcelain` (post-run) | empty | ✓ PASS |
| All TestChangie* + related fixture guards | `GOWORK=off go test ./internal/upgrade/ -run '^(TestChangie\|TestToolModfiles\|TestContributingReferencesRealTaskTargets\|TestWorkflowRunBodiesInvokeTask\|TestGateStancesStated\|TestTaskfileGatesFailLoud)' -count=1 -v` | all PASS (20 top-level tests) | ✓ PASS |
| Full internal/upgrade package regression | `GOWORK=off go test ./internal/upgrade/ -count=1` | `ok` | ✓ PASS |
| Reassembly byte-check (independent of changie) | header + 14 seeds newest-first `\| cmp - CHANGELOG.md` | exit 0 | ✓ PASS |
| `go.tool-changie.mod` integrity | `go mod verify -modfile=go.tool-changie.mod` | `all modules verified` | ✓ PASS |
| Module-count header claim | `go list -m -modfile=go.tool-changie.mod all \| wc -l` | `64` (matches header) | ✓ PASS |
| `task vuln` scans changie | `task vuln` | `changie: CLEAN (exit 0 ...)` present in output | ✓ PASS |
| `task lint:actions` clean on ci.yml edit | `task lint:actions` | no findings | ✓ PASS |
| `gofmt`/`go vet` on new test file | `gofmt -l internal/upgrade/`, `go vet ./internal/upgrade/` | both clean | ✓ PASS |
| RED commits isolated to test file | `git show --stat 01057ac1`, `git show --stat 1d023e70` | each touches only `internal/upgrade/changie_shape_test.go` | ✓ PASS |
| No leaked mutation-testing worktree | `git worktree list` | only the main checkout | ✓ PASS |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|---|---|---|---|---|
| CHG-01 | 01-01, 01-02 | `.changie.yaml` config, changie pinned with one recorded path | ✓ SATISFIED | `.changie.yaml` shape, `go.tool-changie.mod` pin, `task changie` wrapper, CI wiring in `check:changie`/`vuln` |
| CHG-02 | 01-01 | Baseline layout + byte-identical CHANGELOG.md | ✓ SATISFIED | Files present, reassembly and `git diff` byte checks pass |
| CHG-03 | 01-02 | Live proofs as committed Taskfile target, wired into CI, RED-demonstrated | ✓ SATISFIED | `check:changie` runs live, wired into ci.yml, Family (a) mutation log |
| CHG-04 | 01-02 | Non-interactive write + refusal proofs, confirmed executed | ✓ SATISFIED | Legs 4-8 of `check:changie`, Family (b) mutation log |

No orphaned requirements: REQUIREMENTS.md's Phase 1 mapping (CHG-01..CHG-04) exactly matches the requirement IDs declared across both plans' frontmatter. CHG-05 (historical regeneration from GitHub Releases) is explicitly out of scope for this phase per ROADMAP notes and CONTEXT.md's Phase Boundary — deferred to v2, not orphaned.

### Anti-Patterns Found

None. Scanned all phase-modified files (`internal/upgrade/changie_shape_test.go`, `internal/upgrade/taskfile_shape_test.go`, `.changie.yaml`, `Taskfile.yml`, `CONTRIBUTING.md`, `.github/workflows/ci.yml`, `go.tool-changie.mod`, `.changes/header.tpl.md`) for TBD/FIXME/XXX/TODO/HACK/PLACEHOLDER markers; a diff against `main` confirms zero such markers were introduced by this phase (pre-existing unrelated matches in `Taskfile.yml`/`taskfile_shape_test.go` — a `stage=... XXXXXX` mktemp template and a homebrew-token placeholder comment — predate this phase and are outside its diff).

### Probe Execution

No `scripts/*/tests/probe-*.sh` files exist in this repository and none are declared in the plans. The phase's own equivalent — the `check:changie` Taskfile target and `01-MUTATION-LOG.md`'s RED/GREEN demonstrations — were run directly (see Behavioral Spot-Checks above) rather than via the probe-script convention. `task check:changie` was run twice in this verification session (once standalone, once as part of the full behavioral pass) and was idempotent both times.

### Human Verification Required

None. Every must-have truth was verified by a live, deterministic command run in this session (not inferred from SUMMARY.md claims), and no behavior-dependent state-transition/cancellation invariant in this phase's scope lacked a directly-executable proof.

### Gaps Summary

None found. Both plans' RED-before-GREEN commit ordering was independently confirmed via `git log --reverse`, both RED commits touch only the test file (no premature implementation), all declared artifacts exist on disk with the exact shape the shape-tests assert, all key links are wired and exercised live, all four ROADMAP success criteria were independently re-executed (not merely re-read from SUMMARY.md) and passed, and the one documented deviation (01-MUTATION-LOG.md Family (d)'s floor-vs-set-mismatch ordering) is an honest methodology adjustment to a demonstration script, not a weakening of the underlying guard (`TestChangieVersionSeedsMatchChangelog` was independently re-run in this session and correctly failed in both directions when re-triggered via the log's own transcripts).

---

*Verified: 2026-09-25*
*Verifier: Claude (gsd-verifier)*
