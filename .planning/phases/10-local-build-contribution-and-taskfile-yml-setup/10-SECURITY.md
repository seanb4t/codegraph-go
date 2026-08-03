---
phase: 10
slug: local-build-contribution-and-taskfile-yml-setup
status: verified
# threats_open = count of OPEN threats at or above workflow.security_block_on severity (the blocking gate)
threats_open: 0
asvs_level: 1
created: 2026-08-03
---

# Phase 10 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

Register origin: `register_authored_at_plan_time: true` — all seven PLAN.md files carry a
filled, parseable `<threat_model>` block. This audit **verified that the declared mitigations
exist**; it did not scan for new threats. Verification depth exceeded ASVS L1: ten independent
mutation proofs were run against the real files, each confirmed to have landed before its
result was trusted, and every one reverted (`git status --porcelain` empty at `40d9530`).

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| Go module proxy -> CI runner | Tool binaries (`task`, `goreleaser`, `actionlint`, `govulncheck`) are fetched and compiled from an external registry into the environment that gates this repo | Executable build-tool source |
| Tool bootstrap -> release-gating workflow | `release-please.yml`'s `pretag-gate` newly depends on building `task` from the module proxy | Gate verdict |
| Third-party GitHub Action -> workflow job | `namespacelabs/nscloud-cache-action` executes with the job's token and can read/write the cache volume | Job token, cache contents |
| Namespace runner infrastructure -> repo secrets/token | Job execution moves off GitHub-hosted infrastructure to a third-party runner provider | `contents: read` token |
| Namespace runner -> both gates | Both required checks now execute on third-party runner infrastructure | Gate verdicts |
| Namespace runner -> release artifact | Released, signed, SLSA-attested binaries are produced on third-party runner infrastructure | Published binaries, signatures |
| Cache volume -> build | Restored Go module/build cache contents are consumed by later build steps | Compiled objects, module source |
| Cache volume -> release build | Restored Go module/build cache contributes to the bytes users download | Compiled objects |
| Reusable SLSA workflow -> provenance | The provenance attestation is produced by an external reusable workflow this repo cannot configure | Provenance attestation |
| Reproducibility gate -> release trust | This gate is the only independent evidence that the release build is deterministic | Build hash verdict |
| Cross-toolchain (zig) -> darwin binaries | If darwin were cross-linked, its DNS resolution path would change relative to a native build | Linked binary behaviour |
| CI runner -> apt package source | The mingw-w64 install step fetches a system package into the gating environment | System package |
| Contributor shell -> local toolchain | `task` targets execute arbitrary build/test commands on a contributor machine from a repo-controlled file | Local command execution |
| Documentation -> contributor behaviour | Prose that overstates a guarantee causes contributors to skip a real prerequisite | Contributor assumptions |
| `.goreleaser.yaml` target list -> pre-tag sweep | Two independently-edited files must enumerate the same set, or a release target ships unswept | Target set |
| `pretag-gate` -> tag creation | This job is a `needs:` dependency of the job that creates release tags | Release authorization |
| Contributor machine -> perf baseline | Anything that can write `baseline.json` can silently redefine what "no regression" means | Perf authority |
| Benchmark artifact -> committed authority | An artifact downloaded from a workflow run becomes the repository's perf authority | Measurement record |
| CI runner class -> measured throughput | The measurement environment is part of the measurement; a change in it invalidates comparison | Frame descriptor |
| Measurement environment -> perf verdict | Comparing across machines produces meaningless numbers that read as meaningful | Frame descriptor |
| Missing toolchain -> gate outcome | Whether an absent cross-toolchain produces a failure or a pass determines whether the local signal is trustworthy | Gate outcome |
| Guard allowlist -> enforced invariant | An exception list that can grow silently converts an enforced property into a decorative one | Invariant strength |
| Scratch checkout -> working tree | A verification run that writes back into the repository contaminates the thing it is verifying | Working-tree integrity |

---

## Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation | Status |
|-----------|----------|-----------|----------|-------------|------------|--------|
| T-10-01-01 | Tampering | `.github/actions/install-task/action.yml` tool bootstrap | high | mitigate | `GOWORK=off go build -modfile=go.tool.mod` only (`action.yml:22-24`); no release download / install script / `curl \| sh` under `.github/`; `go.tool.sum:397-398` carries task v3.52.0 hashes; no `GOFLAGS`/`GOPRIVATE`/`GONOSUMDB` override. Mutation M10: root-`go.mod` tool directive -> `TestToolModfilesRemainIsolated` RED | closed |
| T-10-01-02 | Tampering | `namespacelabs/nscloud-cache-action` (new third-party action) | high | mitigate | All 8 usages SHA-pinned to `c5f8dab7560444c4bf8dbc64f1b203431873c547 # v1.6.1`; zero unpinned. Live re-resolve `gh api .../git/ref/tags/v1.6.1` -> same SHA, type `commit` | closed |
| T-10-01-03 | Spoofing | Restored Go module/build cache on shared Namespace Cache Volume | medium | mitigate | All 8 cache steps are `with: cache: go` only; no artifact or signed output restored from cache; `ci.yml:173-230` reproducibility double-build is a required check. See Weak Evidence 1 | closed |
| T-10-01-04 | Elevation of Privilege | Namespace runner executing `pull_request`-triggered jobs | medium | accept | `ci.yml:36-43` — `on: pull_request` (never `pull_request_target`) + `permissions: contents: read`. Stated reason holds verbatim. AR-10-01 | closed |
| T-10-01-05 | Information Disclosure | Job-name preservation vs branch-protection ruleset | low | mitigate | Mutation M5: shortened `ci.yml:174` job name -> `TestRequiredCheckNamesPreserved` RED naming the missing context; restored -> GREEN. Fixture is a strict superset of the live ruleset (see Weak Evidence 2) | closed |
| T-10-01-SC | Tampering | npm/pip/cargo installs | low | accept | Only npm invocation in the repo is the pre-existing `bench.yml:122 npm install -g @colbymchenry/codegraph@1.3.1`, untouched by the phase diff. No pip/cargo anywhere. AR-10-02 | closed |
| T-10-02-01 | Tampering | `vet:daemon-windows` / any cross-toolchain target | high | mitigate | `Taskfile.yml:120-122` `preconditions:` + actionable `msg:`; zero `status:`/`platforms:` in the file. Mutations M1 (`platforms: [linux]`) and M9 (blanked `msg:`) both drove `TestTaskfileGatesFailLoud` RED | closed |
| T-10-02-02 | Tampering | `Taskfile.yml` as an executable file in the repo | medium | accept | Accepted on **amended** justification — see AR-10-03. Load-bearing clauses verified: the only URL in the file sits inside a precondition `msg:` string (`:419`), so no target fetches or executes remote content; `Taskfile.yml` was code-reviewed. Clause 3 of the plan's wording is falsified | closed |
| T-10-02-03 | Tampering | `Install mingw-w64` apt step | low | accept | `ci.yml:150-151` body unchanged in the phase diff (context line, no `±`); `set -e` semantics, fails loud. AR-10-04 | closed |
| T-10-02-04 | Tampering | New `go vet ./...` gate surfacing pre-existing findings | low | mitigate | `ci.yml:74-75` `Vet (go vet ./...)` inside the required `test` job -> `Taskfile.yml:94-101`; phase diff adds zero `//nolint` / `//lint:ignore`; `task vet` in a clean checkout -> exit 0 | closed |
| T-10-02-SC | Tampering | npm/pip/cargo installs | low | accept | govulncheck resolved via `go.tool.mod`; `go.tool.sum:905-906` carries `golang.org/x/vuln v1.6.0` hashes. No npm/pip/cargo. AR-10-05 | closed |
| T-10-03-01 | Tampering | Release build executed on third-party (Namespace) runner infrastructure | high | mitigate | `.goreleaser.yaml` `-trimpath`, `-buildid=`, `mod_timestamp: {{ .CommitTimestamp }}`; `ci.yml` double-build gate; `release.yml:268` `cosign sign-blob --bundle` per binary; `:282` `syft -o spdx-json` per binary; provenance job carries no `runs-on` (machine-enforced, M4) | closed |
| T-10-03-02 | Tampering | Silent substitution of a cross-linked darwin build for a native one | high | mitigate | Mutation M1': darwin/arm64 `needs_zig: false->true` -> `TestDarwinLegsBuildNatively` RED. M2': darwin runner -> linux label -> RED. Live control: canary run 30763026703 success on `namespace-profile-macos-6x14-tahoe`, runner `nsc-runner-i1oh37q1f82ik` | closed |
| T-10-03-03 | Spoofing | Restored cache on a shared Cache Volume feeding a release build | medium | mitigate | `release.yml:128-130` `cache: go` only; no release artifact restored from cache. See Weak Evidence 1 | closed |
| T-10-03-04 | Tampering | Provenance job moved or its generator reference weakened | high | mitigate | Mutation M3: `@v2.1.0` -> 40-char SHA -> `TestProvenanceJobUsesTaggedSLSAGenerator` RED. M4: added `runs-on: ubuntu-latest` to the provenance job -> RED naming the D-07 violation. Both restored -> GREEN | closed |
| T-10-03-05 | Tampering | Incidental version-pin drift during the runner edit | medium | mitigate | `git diff 82ffd60^ HEAD -- .github/workflows/release.yml` (comments stripped): only `runner:`/`runs-on:`/`cache:` values and the new cache step changed — **zero `run:` lines**. `release.yml:56 GORELEASER_VERSION: "v2.17.0"` unchanged | closed |
| T-10-03-SC | Tampering | npm/pip/cargo installs | low | accept | No package install in the 10-03 diff. AR-10-06 | closed |
| T-10-04-01 | Tampering | `tools/bench/baseline.json` | high | mitigate | `rg rebless Taskfile.yml` -> only `desc:`/`echo` prose; no target invokes `-rebless`. `git show --stat 335a88f` -> `BASELINE.md` + `baseline.json` only; no code or workflow rode along. See Weak Evidence 3 | closed |
| T-10-04-02 | Tampering | Runner-class change invisible to the platform guard | high | mitigate | `internal/bench/metrics.go:28,48` frame fields; `bench.yml:213-217` records `-runner "$CODEGRAPH_BENCH_RUNNER"`; `baseline.json` carries `"runner":"ubuntu-latest"`,`"scratch_fs":"disk"`; comparison landed in 10-06 and is proven firing (T-10-06-01). See Weak Evidence 4 | closed |
| T-10-04-03 | Tampering | Mixed-runner trial aggregation | medium | mitigate | `tools/bench/runner/main.go:772-779` refuses a mixed-runner trial set naming both values, mirroring the GOOS/GOARCH convention | closed |
| T-10-04-04 | Denial of Service | Namespace runner unavailable when a baseline must be recorded | medium | accept | `bench.yml:180` `if: github.event_name == 'workflow_dispatch' && …` — manual only, not a required check, `contents: read` only. Premise now strictly safer: rebless runs on `ubuntu-latest`. AR-10-07 | closed |
| T-10-04-05 | Tampering | Cache Volume contents influencing measured throughput | medium | accept | The `rebless` job (`bench.yml:171-311`) carries **no** nscloud-cache-action at all; corpus is deterministic and network-free (`ci.yml:259-261`); deferred item recorded at `tools/bench/BASELINE.md:328-340`. AR-10-08 | closed |
| T-10-04-SC | Tampering | npm/pip/cargo installs | low | accept | Pinned `@colbymchenry/codegraph@1.3.1`, untouched by the phase diff. AR-10-09 | closed |
| T-10-05-01 | Tampering | `pretag-gate` newly depending on a module-proxy tool build | high | mitigate | `release-please.yml:51-52 uses: ./.github/actions/install-task`; bootstrap of `task 3.52.0` from `go.tool.mod` on a restricted PATH -> exit 0; `go.tool.sum` hashes present; `release-please` job `needs: pretag-gate` (`:59`) so a loud bootstrap failure blocks tag creation | closed |
| T-10-05-02 | Tampering | The moved sweep becoming a gate that cannot fail | high | mitigate | Independent RED: injected a `//go:build windows` file importing a nonexistent module -> `task check:cross` exit **201** with `::error::go list -mod=readonly ./... failed for GOOS=windows GOARCH=amd64`; removed -> exit 0. Wiring: `release-please.yml:54-55 run: task check:cross`. See Weak Evidence 5 | closed |
| T-10-05-03 | Tampering | Divergence between the sweep's target list and the release build's target list | high | mitigate | Mutation M2: dropped `darwin/arm64` from `check:cross` -> `TestCheckCrossMatchesGoreleaserTargets` RED naming it. Inverse: added a 7th `freebsd/amd64` goreleaser build -> RED naming it. Parser demonstrably reads real inline flow sequences; `TestParseGoreleaserCrossPairs_InlineFlowSequence` covers the syntax explicitly | closed |
| T-10-05-04 | Denial of Service | Cold tool build on every push to `main` | low | mitigate | `rg actions/cache .github/workflows/` -> none; measured 2.8s / 9.7s / 20.2s across three cache states, all far under the 60s threshold. Measured locally, not on `ubuntu-latest` | closed |
| T-10-05-05 | Elevation of Privilege | `release-please.yml` runner scope | low | accept | `git diff 82ffd60^ HEAD -- .github/workflows/release-please.yml` (comments stripped): only the sweep step replaced by `Install Task` + `Run check:cross sweep`. `runs-on: ubuntu-latest` and the `permissions:` block untouched. AR-10-10 | closed |
| T-10-05-SC | Tampering | npm/pip/cargo installs | low | accept | No npm/pip/cargo in `release-please.yml` or `Taskfile.yml`. AR-10-11 | closed |
| T-10-06-01 | Tampering | `internal/bench.CheckRegression` runner blind spot | high | mitigate | `internal/bench/regression.go:72-82` refuses a runner/scratch_fs mismatch **before** any tolerance arithmetic (`:112`), naming both values; empty is refused, not treated as a wildcard. Probed against the real committed `baseline.json`: identical frame -> `nil`; wrong runner -> refused; empty runner -> refused; wrong `scratch_fs` -> refused; wrong runner + 90% drop -> the runner error fires (ordering proven); matching frame + real 50% drop -> `throughput regressed 50.0% (budget: 10.0%)`. See Weak Evidence 6 | closed |
| T-10-06-02 | Tampering | Reproducibility double-build scripts moved into `Taskfile.yml` | high | mitigate | `Taskfile.yml:385-396` / `:420-443` vs `git show 82ffd60:.github/workflows/ci.yml:209-226,239-260`: build fn, ldflags, sha256sum, compare and messages byte-identical; only the two documented `${SOURCE_DATE_EPOCH:-…}` / `${COMMIT:-…}` local-default lines added. `ci.yml:208-230` keeps two separate steps; `continue-on-error: true` only on arm64 | closed |
| T-10-06-03 | Tampering | `check:reproducibility:arm64` silently not running locally | medium | mitigate | `Taskfile.yml:417-419` `preconditions: - sh: command -v zig` with an actionable install `msg:`; `TestTaskfileGatesFailLoud` enforces a non-empty `msg:` for every zig/mingw-referencing target (proven RED in M9) | closed |
| T-10-06-04 | Tampering | Namespace runner producing the reproducibility verdict | medium | accept | `check:reproducibility` is a two-build same-job sha256 self-comparison (`Taskfile.yml:391-402`); provenance is produced by the reusable workflow with no caller `runs-on` — machine-enforced (M4). Stated reason holds. AR-10-12 | closed |
| T-10-06-05 | Tampering | Perf gate red between plan 10-04's baseline commit and this plan | medium | mitigate | `10-04-SUMMARY.md:207` records the disposition and reason ("never moved off `ubuntu-latest`, so there is no red/unreliable window"); 10-06 closed the frame gap via `ci.yml:243-244 CODEGRAPH_BENCH_RUNNER: ubuntu-latest` matching the committed baseline; job green in CI run 30763026686 | closed |
| T-10-06-SC | Tampering | npm/pip/cargo installs | low | accept | No npm/pip/cargo; no new module dependency in the 10-06 diff. AR-10-13 | closed |
| T-10-07-01 | Tampering | `TestWorkflowRunBodiesInvokeTask` exception list | high | mitigate | Mutation M6: re-inlined `perf-regression`'s run body -> RED naming workflow/job/step. M7: fixture entry for a step that does not exist -> RED (*"was never matched against a real step … a stale exception silently widens the allowlist (T-10-07-01)"*). M8: blanked a real exception's `Reason` -> RED at `validateRunBodyExceptions`. Fixture is 2 literal entries (`taskfile_shape_test.go:138-151`) | closed |
| T-10-07-02 | Spoofing | `CONTRIBUTING.md` prose overstating tool-update coverage | medium | mitigate | `CONTRIBUTING.md:92-96` states the two tool modfiles are updated manually and that "neither Dependabot nor Renovate is configured for this repository at all". Verified true: no `.github/dependabot.y*ml`, no `renovate.json`, no `.renovaterc*` | closed |
| T-10-07-03 | Tampering | Scratch checkout containing repository content outside the working tree | low | mitigate | `git status --porcelain` empty at HEAD `40d9530`; scratch clone documented as created outside the tree and deleted (`10-07-SUMMARY.md:165-180`) | closed |
| T-10-07-04 | Spoofing | Criterion 3 claimed on text evidence alone | high | mitigate | Independently reproduced: fresh `git archive HEAD` checkout outside the repo, PATH reduced to a clean bin plus system dirs with `task`/`zig`/`goreleaser`/`actionlint`/`x86_64-w64-mingw32-gcc` confirmed absent. Bootstrapped `task` from `go.tool.mod` -> exit 0; bare `task` listed targets and left `./codegraph` absent; `task build` -> exit 0; `task lint` -> exit 0 (actionlint cold-built from `go.tool-lint.mod`) | closed |
| T-10-07-SC | Tampering | npm/pip/cargo installs | low | accept | 10-07 touched only `CONTRIBUTING.md` and a test file; no package install. AR-10-14 | closed |

*Status: open · closed · open — below `high` threshold (non-blocking)*
*Severity: critical > high > medium > low — only open threats at or above `workflow.security_block_on` count toward `threats_open`*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| AR-10-01 | T-10-01-04 | The Namespace runner provider gains the same read-only token the GitHub-hosted runner already had. `ci.yml` is `pull_request`-triggered (never `pull_request_target`) with `permissions: contents: read`; the runner-class change grants no additional token scope | maintainer | 2026-08-03 |
| AR-10-02 | T-10-01-SC | Supply-chain legitimacy checkpoint. Plan 10-01 installs no npm, pip or cargo package; all tools resolve through the Go module proxy and were audited in `10-RESEARCH.md`'s Package Legitimacy Audit (all entries `OK`, zero `[ASSUMED]`/`[SUS]`/`[SLOP]`) | maintainer | 2026-08-03 |
| AR-10-03 | T-10-02-02 | **Amended justification.** The plan's stated acceptance rested on three clauses; clause 3 — "every target body is ported verbatim from an existing, reviewed CI step" — is **falsified as shipped**: `diag:cpu` (`Taskfile.yml:219`), `diag:storage-fit` (`:342`), `check:darwin-toolchain` (`:151`) and `default` were authored fresh in plans 10-03/10-04, not ported from any pre-existing CI step. Acceptance is re-grounded on the two clauses that were verified true: (a) no target fetches or executes remote content — the only URL in the file sits inside a precondition `msg:` string at `:419`; (b) `Taskfile.yml` was code-reviewed like any other source change (`10-REVIEW.md` files-reviewed list). The residual risk is that a contributor executes repo-controlled commands by running `task`, which is inherent to adopting a task runner and is bounded by (a) | maintainer | 2026-08-03 |
| AR-10-04 | T-10-02-03 | The `Install mingw-w64` apt step is unchanged by this phase (context line in the diff, no `±`) and fails loud under `set -e`. Trusting the distribution's package source for a cross-compiler is pre-existing, not introduced here | maintainer | 2026-08-03 |
| AR-10-05 | T-10-02-SC | Supply-chain legitimacy checkpoint. No npm/pip/cargo install in plan 10-02; govulncheck resolves via `go.tool.mod` with `go.tool.sum:905-906` hashes | maintainer | 2026-08-03 |
| AR-10-06 | T-10-03-SC | Supply-chain legitimacy checkpoint. No package install in the 10-03 diff | maintainer | 2026-08-03 |
| AR-10-07 | T-10-04-04 | Baseline re-blessing is `workflow_dispatch`-only (`bench.yml:180`), is not a required check, and holds `contents: read` only. Runner unavailability delays a manual maintainer action; it cannot silently produce a wrong baseline. The premise is now strictly safer than when written — rebless never moved off `ubuntu-latest` | maintainer | 2026-08-03 |
| AR-10-08 | T-10-04-05 | The `rebless` job carries no cache action at all, so cache contents cannot influence the recorded baseline; the corpus is deterministic and network-free. A broader cache-volume-vs-measurement question is deferred and recorded at `tools/bench/BASELINE.md:328-340` | maintainer | 2026-08-03 |
| AR-10-09 | T-10-04-SC | Supply-chain legitimacy checkpoint. The single pinned npm package `@colbymchenry/codegraph@1.3.1` is pre-existing and untouched by the phase diff | maintainer | 2026-08-03 |
| AR-10-10 | T-10-05-05 | `release-please.yml` keeps `runs-on: ubuntu-latest` and its original `permissions:` block; the phase changed only the sweep step. No scope was widened | maintainer | 2026-08-03 |
| AR-10-11 | T-10-05-SC | Supply-chain legitimacy checkpoint. No npm/pip/cargo in `release-please.yml` or `Taskfile.yml` | maintainer | 2026-08-03 |
| AR-10-12 | T-10-06-04 | The reproducibility verdict is a same-job two-build sha256 self-comparison, so a hostile runner could only make it agree with itself — it cannot forge agreement with an independent build. Provenance, which is the externally-trusted artifact, is produced by the reusable SLSA workflow with no caller-supplied `runs-on` (machine-enforced). Accepted with the caveat in Weak Evidence 1 | maintainer | 2026-08-03 |
| AR-10-13 | T-10-06-SC | Supply-chain legitimacy checkpoint. No npm/pip/cargo and no new module dependency in the 10-06 diff | maintainer | 2026-08-03 |
| AR-10-14 | T-10-07-SC | Supply-chain legitimacy checkpoint. Plan 10-07 touched only `CONTRIBUTING.md` and a test file | maintainer | 2026-08-03 |

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-08-03 | 40 | 40 | 0 | gsd-security-auditor (opus) + orchestrator verification |

Audit scope: 24 implementation files, diff base `82ffd60^` (confirmed ancestor of HEAD; the same
file set was independently corroborated by two bases during code review). Ten mutation proofs
executed and reverted; working tree byte-clean at `40d9530` after the audit.

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-08-03

---

## Weak Evidence

Threats closed where the evidence did not fully earn the verdict. Recorded so a future audit
does not inherit an unexamined ✓. None is blocking; all are noted for the next phase that
touches the relevant surface.

1. **T-10-01-03 / T-10-03-03 — cache spoofing, wording overstates coverage.** The mitigation
   says restored cache contents are "checksum-verified against `go.sum`/`go.tool.sum` at use
   time". That holds for the **module** cache; the Go **build** cache is content-addressed by
   action ID, not verified against a trusted checksum database. Separately, the "independent
   double-build" is not fully independent — `ci.yml`'s `reproducibility` job runs on
   `namespace-profile-linux-amd64-4x8` with the same restored cache, so identically-poisoned
   output twice would pass. T-10-06-04/AR-10-12 accepts exactly this.
2. **T-10-01-05 — fixture drift, over-strict.** `requiredCheckNames`
   (`internal/upgrade/taskfile_shape_test.go:43-51`) lists **7** contexts; the live ruleset
   returns **6** and does **not** include `goreleaser check (config validation, DIST-01)`
   (verified: `gh api repos/seanb4t/codegraph-go/rulesets/20157557`). The fixture is a strict
   superset — over-strict, never under-strict — so the mitigation still covers every live
   required context and T-10-01-05 stands. But the fixture's own comment ("six … plus
   pr-title") and `10-01-SUMMARY.md`'s "set-equal to the live ruleset" claim are now false.
   **Corollary: this refutes code-review finding WR-04.** `CONTRIBUTING.md:42-44` lists exactly
   the six live required contexts; WR-04's evidence was the stale fixture and
   `10-01-PLAN.md:181`, not the live API. Neither file was modified during this audit.
3. **T-10-04-01 — process controls are documentary.** "Replaced byte-for-byte from the
   machine-written artifact (SHA-256 verified)" and "a blocking maintainer checkpoint precedes
   the commit" are not re-verifiable post hoc. What is machine-verifiable was verified: no
   rebless flag is reachable from any task target, and commit `335a88f` carries no code or
   workflow change. The literal "the commit contains nothing else" holds only if `BASELINE.md`
   (+297 lines of provenance record) is read as part of "the baseline".
4. **T-10-04-02 — residual seam.** `CODEGRAPH_BENCH_RUNNER` is a hand-mirrored literal of
   `runs-on:` (`bench.yml:173` + `:182`; `ci.yml:234` + `:244`), not derived from the label, and
   no test ties the two. Editing `runs-on:` without editing the env var would record or compare
   a **mislabelled** frame, and `CheckRegression` would silently agree because both sides would
   carry the stale label — a live extension of the exact blind spot the threat names.
5. **T-10-05-02 — CI leg unperformed.** The mitigation requires a demonstrated RED "once
   locally and once in CI". Local RED is proven twice (executor, then independently). The CI
   leg was never run: the rewired `pretag-gate` has not executed in any workflow run — the last
   `release-please.yml` run on `main` (`30727791117`, sha `82ffd60`) still shows the pre-phase
   `6-target go list -mod=readonly sweep` step. Closed because the security property (the gate
   can go RED; a bootstrap failure fails loud rather than silently passing) is directly
   demonstrated; the unproven part is environmental and resolves on the next push to `main`.
6. **T-10-06-01 — CI leg unperformed.** Same shape: "demonstrated against the real committed
   baseline [done] and in a real CI run with a deliberately wrong label [never done]". The perf
   gate ran green in CI at `a61ccf1` with the *correct* label, proving it is wired and evaluated
   without spurious failure, but not that it goes red there.

---

## Advisory — Unregistered Surface

New or newly-exposed attack surface with no entry in this phase's register. Not blocking,
not counted in `threats_open`. Recommended for a registered threat in the next phase that
touches these files. No SUMMARY in this phase carried a `## Threat Flags` section, so these
were surfaced by the auditor rather than declared at execution time.

- **Tool-modfile supply chain is unscanned** (also code-review WR-07). This phase introduced
  `go.tool.mod` / `go.tool-lint.mod` — roughly 400 modules including goreleaser plus AWS/GCP/Azure
  SDKs, k8s client libraries, cosign and sigstore, plus actionlint's tree. These are built from
  source and **executed as credentialed CI tooling**: `goreleaser` signs and publishes releases;
  `task` drives every CI job body. `ci.yml:156-171`'s blocking `govulncheck` job scans the **root
  `go.mod` only**. `Taskfile.yml`'s `vuln` target is the only thing that ever points govulncheck
  at `go.tool.mod`, is documented as local-only (`go.tool.mod:10-15`), and is invoked by no CI
  job. `go.tool-lint.mod` has no vulnerability-scanning path at all. T-10-01-01 covers *how*
  these modules are fetched (checksummed proxy); nothing covers a known-vulnerable dependency
  being *executed* in the credentialed release pipeline.
- **Degenerate `current` bypasses both RSS gates** (also code-review WR-06), reproduced during
  this audit: `CheckRegression(baseline, current, ceiling=1)` with `current.PeakRSSBytes = 0` and
  a matching frame returns `nil` — both the relative RSS check and the absolute INDX-06 ceiling
  silently pass. Unreachable today (`internal/bench.PeakRSSBytes` errors rather than returning
  0), but `CheckRegression` is exported and its doc comment claims it "never misleads".
- **`.github/workflows/darwin-toolchain-canary.yml`** is new attack surface created mid-phase
  (a Namespace macOS runner executing PR-triggered jobs) with no register entry. Its posture
  matches the accepted one in AR-10-01: `on: pull_request` (not `pull_request_target`) plus
  `permissions: contents: read`. Code-review WR-05 stands — its `paths:` filter (lines 29-35)
  omits `.github/actions/install-task/action.yml`, `go.tool.mod` and `go.tool.sum`, i.e. exactly
  the bootstrap files most likely to break it.
- **Stale doc comments in `internal/bench/metrics.go:25-27` and `:46-47`** still assert that
  `Runner` and `ScratchFS` do "NOT yet participate in CheckRegression's comparison". Both now
  gate the comparison (10-06, proven under T-10-06-01). `tools/bench/BASELINE.md` was updated;
  `metrics.go` was not. A future reader could conclude the T-10-06-01 mitigation does not exist.
