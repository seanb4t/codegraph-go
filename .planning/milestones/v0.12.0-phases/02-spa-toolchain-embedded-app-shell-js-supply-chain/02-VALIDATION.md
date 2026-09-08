---
phase: 2
slug: spa-toolchain-embedded-app-shell-js-supply-chain
# status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6)
# audit-milestone §5.5 distinguishes NOT-VALIDATED (draft) from PARTIAL (validated + nyquist_compliant: false) (#2117)
status: validated
nyquist_compliant: true
wave_0_complete: true
created: 2026-08-23
validated: 2026-09-07
---

# Phase 2 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | `go test` (Go side — this repo's only test framework). No JS test framework: no v1 requirement asks for one, and adding `vitest` is outside this phase's locked scope. |
| **Config file** | `Taskfile.yml` (existing) gains new targets; no new Go test config needed |
| **Quick run command** | `go test ./internal/uiserver/... ./internal/upgrade/...` |
| **Full suite command** | `task test:unit` |
| **Measured runtime** | **`task test:unit` 10s** (measured 2026-09-07); the `web:*` Taskfile gates add ~5–40s each, `web:build:verify` being the slowest since it runs a real scratch build |

---

## Sampling Rate

- **After every task commit:** `go test ./internal/uiserver/... ./internal/upgrade/...` for handler/embed/structural changes; `task web:drift` for anything touching `web/`
- **After every plan wave:** `task test:unit` plus the new `web:drift` and the extended `proto:drift`
- **Before `/gsd-verify-work`:** full suite green — asserted under `set -o pipefail` with a captured pipeline status, zero `^FAIL` lines and at least one `^ok ` line, never by a `tail`-terminated pipeline's exit code — **including demonstrated-RED runs of BLD-03's guard against BOTH a deliberately staled `web/build/` AND a hand-edited byte inside the committed `web/build/`** (rule `84d1gfpywd` — a guard is not trusted green until it has been watched fail, and the output half is a distinct guard from the source half)
- **Max feedback latency:** **10s** for `task test:unit`, measured 2026-09-07 (well inside the ~60s budget)

---

## Per-Task Verification Map

Task IDs are assigned by the planner. Plans exist as of 2026-08-23; the Plan and Wave
columns name the plan that owns each requirement's primary verification. Several requirements
are touched by more than one plan (see each PLAN.md's `requirements` field) — the column names
the plan whose acceptance criteria carry the automated command in this row.

**Command hygiene applied across every row (2026-08-24, cross-AI review cycle).** Two defect
shapes were swept out of every automated command in all seven plans and this map was re-synced to
match:

1. **A verify whose exit status does not depend on the property it checks.** Every command that
   pipes now runs under `set -o pipefail`, because a pipeline reports its LAST stage's status —
   `go test ./... 2>&1 | tail -30` returns `tail`'s status, so a red suite verified green
   (`sh -c 'exit 1' | tail -3` exits 0). Every `go test -run` assertion counts `--- PASS: <exact
   test name>` lines per name and asserts zero `--- FAIL` lines, because `go test -run` exits 0
   when the pattern matches nothing. Every `svelte-check` assertion is anchored to the tool's own
   `found 0 errors` summary line, because a bare `0 errors` is a substring of `found 10 errors`.
2. **An absence check with no positive control.** Every negative `rg`/`git ls-files`/`test !`
   assertion is now preceded, in the SAME command, by an assertion that the subject existed and
   was non-empty — so no check can pass by examining nothing.
3. **A `wc` count compared without stripping BSD padding** (added 2026-08-24, cycle 3). BSD `wc`
   (macOS, this repository's development host) right-pads its output: `printf '' | wc -l` emits
   `       0`, so `… | wc -l | grep -qx '0'` exits 1 on a genuinely empty result — a guaranteed
   false RED. Every `wc` count in the plan set is now either piped through `tr -d ' '` before
   comparison or replaced by `test -z "$(…)"`. This class fails CLOSED, so it never shipped
   anything unsafe, but it stalls a wave and invites an executor to "fix" it by loosening the
   guard.

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 02-02-T1 | 02-02 | 2 | RPC-03 | T-02-02-01, T-02-02-06 | A client-side route serves `index.html`; `/codegraph.ui.v1.UIService/*` reaches the Connect handler; a miss under the immutable-asset prefix returns 404, never HTML; every response carries `nosniff` and a same-origin CSP whose `script-src` hashes are derived from the embedded `index.html` | unit (Go, `httptest`) | `set -o pipefail; go test ./internal/uiserver/... -run 'TestSPA' -v` piped to `tee`, then one `--- PASS: <name>` count assertion per named test and a zero `--- FAIL` assertion | ✅ | ✅ green (14 PASS) |
| 02-01-T2 | 02-01 | 1 | BLD-02 | — | The embedded FS file-list equals the on-disk `web/build/` file-list, `_app/` included (criterion 1 is a file-list diff, not "the build succeeded") | unit (Go, `fs.WalkDir` + `filepath.WalkDir` set-diff) | `set -o pipefail; go test ./internal/uiserver/... -run 'TestEmbeddedFSMatchesOnDiskBuildTree|TestEmbeddedBuildTreeIsNonTrivial' -v`, asserting one `--- PASS` per name and zero `--- FAIL` | ✅ | ✅ green (2 PASS) |
| 02-06-T2 | 02-06 | 4 | BLD-03 | T-02-06-01, T-02-06-07 | Guard reports how many SOURCE files it hashed AND how many committed OUTPUT files it manifested, and fails RED against a stale `web/build/`, a hand-edited byte inside it, an added file, a removed file, a narrowed enumeration and a missing marker | shell (Taskfile target) | `set -o pipefail; task web:drift` piped to `tee`, asserting BOTH printed count lines (`hashed N source files`, `manifested N output files`) and a clean `git diff --exit-code web/` | ✅ | ✅ green (exit 0 — hashed 110 source, manifested 32 output, both digests MATCH) |
| 02-06-T1a | 02-06 | 4 | BLD-01 | — | `pnpm install --frozen-lockfile` succeeds at the Corepack-pinned version on a clean checkout | integration (CI step) | `set -o pipefail; task web:deps` (resolves Corepack or a pnpm at major 10+, then installs frozen; reports which path it took) piped to `tee` against a non-empty log | ✅ | ✅ green (exit 0 — frozen install resolves) |
| 02-06-T1b | 02-06 | 4 | BLD-01 | T-02-06-08 | CI actually REBUILDS: `pnpm build` runs into a scratch directory on every PR and the fresh output is asserted structurally valid, without touching the committed tree | integration (CI step) | `set -o pipefail; task web:build:verify` piped to `tee`, asserting the printed built-file count, then `git diff --exit-code -- web/` and an empty `git status --porcelain web/build` | ✅ | ✅ green (exit 0 — built into scratch, committed tree untouched) |
| 02-07-T1 | 02-07 | 5 | BLD-05 | T-02-07-04 | `strictDepBuilds` is in effect — asserted positively, not inferred from a passing install | shell/CI | `set -o pipefail; task web:deps:strict` — reads `pnpm-workspace.yaml`'s `strictDepBuilds` key structurally and asserts boolean `true`, and reports the `allowBuilds` entry and denial counts | ✅ | ✅ green (strictDepBuilds true, allowBuilds 0 entries, 0 denials) |
| 02-07-T2a | 02-07 | 5 | BLD-06 | T-02-07-08 | The lockfile's SHAPE is checked, not only its size: expected `lockfileVersion`, integrity-bearing resolutions EQUAL to total resolutions, and zero `file:` / git-branch / HTTP-tarball sources | shell/CI | `set -o pipefail; task web:lockfile` piped to `tee`, asserting the declared-package count line, the `lockfileVersion` line and the `integrity-bearing resolutions: N of M` line | ✅ | ✅ green (242 packages, lockfileVersion 9.0, 242/242 integrity-bearing, 0 non-registry) |
| 02-07-T2b | 02-07 | 5 | BLD-06 | T-02-07-03 | `pnpm audit` runs; the sibling half counts packages from `pnpm-lock.yaml` **without reading `pnpm audit`'s exit code**; a failed scan never reads as clean | shell/CI | `set -o pipefail; task web:audit` — runs `web:lockfile` to completion BEFORE invoking the audit, then classifies clean / advisories / SCAN ERROR distinctly | ✅ | ✅ green (audit exited 0, classified CLEAN, zero advisories) |
| 02-04-T1 | 02-04 | 2 | BLD-07 | T-02-04-01, T-02-04-05, T-02-04-07 | No `node`/`npm`/`npx`/`pnpm` invocation is reachable from `.goreleaser.yaml` or `release.yml` **through the release path's own execution edges** — the local composite actions they `uses:` and the `Taskfile.yml` targets their `run:` bodies invoke — checked structurally, with the closure derived from the workflow rather than hardcoded; and an edge kind the closure does NOT model (a scanned scalar invoking a repository-local executable script, or a workflow acquiring `container:`/`services:`) refuses with `unsupported reachability edge` rather than shrinking the model behind a still-green verdict | unit (Go, structural parse + transitive closure) | `set -o pipefail; go test ./internal/upgrade/... -run 'TestReleasePathClosureIsTransitive|TestReleasePathHasNoJSToolchain|TestReleasePathScanIsNonVacuous|TestReleasePathScanIgnoresNearMisses|TestReleasePathMissingFileIsError|TestUnsupportedReachabilityEdgeIsLoud' -v`, asserting one `--- PASS` per name and zero `--- FAIL` | ✅ | ✅ green (6 PASS) |
| 02-04-T3 | 02-04 | 2 | BLD-01 | T-02-04-06, T-02-06-06 | No mutable cache is reachable on the JS install path in `ci.yml` — no `actions/cache` step, no JS-scoped cache action, no `cache:` input on the Node setup action — while the existing Go cache stays allowed and is proven to have been examined | unit (Go, structural parse) | `set -o pipefail; go test ./internal/upgrade/... -run 'TestJSInstallPathHasNoMutableCache|TestMutableCacheScanIsNonVacuous' -v`, asserting one `--- PASS` per name and zero `--- FAIL` | ✅ | ✅ green (2 PASS) |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [x] `internal/uiserver/spa_test.go` — RPC-03: fallback vs. 404 vs. RPC dispatch, plus the Content-Security-Policy tests (`TestSPASetsCSPOnEveryResponse`, `TestSPACSPForbidsUnsafeDirectives`, `TestSPACSPHashesCoverEmbeddedInlineScripts`) added this review cycle (02-01 creates the file, 02-02 completes the rule)
- [x] `internal/uiserver/spa_test.go` — BLD-02 criterion 1: embedded-vs-on-disk file-list diff plus the non-triviality guard-the-guard (02-01 Task 2; the plans keep this in `spa_test.go` rather than a separate `embed_test.go`)
- [x] `Taskfile.yml` `web:deps` / `web:build` / `web:build:verify` / `web:drift` targets — BLD-01 and BLD-03: a **two-part** marker (`web/build/.build-manifest`) binding a source-tree digest AND a manifest of the committed output tree's file list and per-file content hashes, both counts printed before any comparison, provable RED six ways; plus a scratch-directory rebuild check so CI actually builds (02-06)
- [x] `Taskfile.yml` extension of `proto:gen` / `proto:drift` — D-05/D-06/D-07: second buf template, floor moved from 3 to **4** (02-03)
- [x] `.github/workflows/ci.yml` `test` job — `actions/setup-node` pinned to **Node 24** (Corepack is unbundled from Node 25+) with **no `cache:` input**, `pnpm install --frozen-lockfile`, the scratch rebuild check, the drift guard, the `strictDepBuilds` assertion, `pnpm audit`, and the sibling lockfile count-and-shape assertion (D-13) — split across 02-06 (Node setup, install, rebuild check, drift guard) and 02-07 (both supply-chain gates). Every new step's `run:` body must be exactly `task <target>`: `ci.yml`'s `test` job is bound by the single-definition property that `internal/upgrade/taskfile_shape_test.go`'s `inScopeJobs` fixture enforces
- [x] `internal/upgrade/taskfile_shape_test.go` extension — BLD-07's **transitive** structural proof (closure resolver over local `uses: ./…` actions and `task <target>` invocations, derived not hardcoded) plus its positive control across all four reachable unit kinds, its near-miss table, and its `unsupported reachability edge` tripwire (`TestUnsupportedReachabilityEdgeIsLoud`, added this review cycle: a repo-local script invocation or a `container:`/`services:` key refuses loudly instead of silently narrowing the coverage model); and the JS-install-path mutable-cache invariant, following that file's existing fixture pattern (02-04)
- [x] `Taskfile.yml` `web:lockfile` target — BLD-06's lockfile-shape half: expected `lockfileVersion`, integrity-bearing resolutions equal to total resolutions, zero non-registry sources, each printing its observed number before comparing (02-07)

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| "A developer with only Go on `PATH` … with no JS toolchain installed anywhere on the machine" | BLD-02 (criterion 1) | The automated test proves embedded-FS ≡ on-disk tree, which is the substance. It cannot prove the *absence* of a JS toolchain on the verifying machine — that is a property of the environment, not of the code, and a test asserting it would be asserting something about its own host. | On a machine (or container) with no `node`/`pnpm` on `PATH`: `go build ./cmd/codegraph && ./codegraph ui`, open the printed URL, confirm the app renders and `GetStatus` data appears. Record which `PATH` was used. |
| The rendered app is visually coherent (D-17/D-18 shell) | — | No requirement asks for visual regression testing, and none is configured. | Open `codegraph ui`, confirm the layout renders with the four navigation slots and no console errors. |

---

**Second row reconciled 2026-09-07.** *"The rendered app is visually coherent"* remains
un-automated (no requirement asks for visual regression testing, and none is configured), but
it has since been **exercised repeatedly against a real browser** — the live UAT passes in
Phases 3, 4, 5 and 6 each drove the running binary in Chrome/Playwright and each found defects
headless tests structurally could not (Phase 4's shipped placeholder copy, Phase 5's renderer
error and 1.1px collapse target, Phase 6's two `$effect` double-invocation bugs and the root
route's missing live subscription). It is manual by design, not by neglect.

---

## Validation Audit 2026-09-07

| Metric | Count |
|--------|-------|
| Requirements in scope | 7 (RPC-03, BLD-01, BLD-02, BLD-03, BLD-05, BLD-06, BLD-07) |
| Map rows | 10 |
| Gaps found (MISSING) | **0** |
| Gaps found (PARTIAL) | 0 |
| Already COVERED | 10/10 rows |
| Resolved this audit | 0 (none needed) |
| Escalated to manual-only | 0 |

**Finding:** as in Phase 1, this file was a plan-time seed that execution never reconciled.
Every row already had a passing automated command. Re-run fresh 2026-09-07: Go side **24
`--- PASS`, 0 `--- FAIL`** across `TestSPA` (14), the embed set-diff (2), the release-path
closure (6) and the mutable-cache invariant (2). Taskfile gates all exit 0 with their asserted
count lines printed: `web:drift` hashed 110 source / manifested 32 output, both digests MATCH;
`web:deps:strict` resolved `strictDepBuilds` to `true` with 0 `allowBuilds` entries and 0
denials; `web:lockfile` 242 packages, `lockfileVersion '9.0'`, 242 of 242 integrity-bearing, 0
non-registry sources; `web:audit` classified CLEAN with zero advisories; `web:build:verify`
built into scratch with the committed tree untouched. Working tree clean afterwards.

**Positive control:** `go test -count=1 -v -run 'TestNoSuchTestPhase2' ./internal/uiserver/`
returned **PASS=0** while exiting 0 — the counting method discriminates.

**BLD-03's demonstrated-RED sign-off item verified, not ticked.** `02-06-SUMMARY.md:82,89`
records the guard *"watched RED six ways (stale source; tampered/added/removed output byte;
narrowed enumeration per half; missing marker), each reverted byte-clean"*, with six captured
transcripts. **Known residual:** WINDOWS #29 — all six RED cases are filesystem-level
tampering, which the output half's `find` enumeration does see; what it cannot see is a file
present on disk but *not staged*, because it hashes that as though committed. The gate
therefore still cannot detect an incompletely-staged bundle. Out of this audit's scope; Phase
6 closed it operationally with an assert-after-commit pattern.

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 60s
- [x] No automated command's exit status is independent of the property it checks (every pipe under `set -o pipefail`; every `go test -run` scored by per-name `--- PASS` counts; every `svelte-check` anchored to `found 0 errors`)
- [x] No absence-only check lacks a command-local positive assertion
- [x] BLD-03's guard has been **watched fail** against a staled `web/build/` AND against a hand-edited byte, an added file and a removed file inside the committed `web/build/`, then watched pass on a clean revert in each case
- [x] `task web:drift` is green on the phase's FINAL commit (02-07 edits a file inside its hashed source set)
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** validated 2026-09-07
