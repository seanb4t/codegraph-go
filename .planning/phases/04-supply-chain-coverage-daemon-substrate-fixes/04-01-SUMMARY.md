---
phase: 04-supply-chain-coverage-daemon-substrate-fixes
plan: 01
subsystem: infra
tags: [govulncheck, ci, taskfile, supply-chain, goreleaser, github-actions]

# Dependency graph
requires: []
provides:
  - "task vuln: govulncheck -mode=binary over all four tool-modfile binaries (task, goreleaser, govulncheck, actionlint), advisory stance"
  - "task vuln:selftest: permanent non-vacuity proof that the tool-modfile scan can still detect a known-vulnerable binary"
  - "testdata/vulnredpoc/main.go: permanent, isolated red-proof program targeting GO-2026-5932"
  - "ci.yml tool-vuln job: advisory CI coverage for the 357 measured third-party modules in go.tool.mod/go.tool-lint.mod"
affects: [04-03, release-tooling-supply-chain-audits]

# Actuals (#2632)
actuals:
  tokens: 3251
  tasks: 3
  commits: 3

# Tech tracking
tech-stack:
  added: []
  patterns: ["advisory-stance CI gate stated in three places (tool desc, CI job name, CI step names) — same pattern as Phase 3's tools/transcriptfreeze, opposite stance"]

key-files:
  created:
    - testdata/vulnredpoc/main.go
  modified:
    - Taskfile.yml
    - .github/workflows/ci.yml

key-decisions:
  - "The tool-modfile scan ships advisory rather than blocking. Implementation surfaced a fact the original decision was made without: goreleaser's own binary carries a real, permanently-unfixed, symbol-reachable match for GO-2026-5932 (110 vulnerable symbols in golang.org/x/crypto/openpgp, via goreleaser/cmd -> pipe/ko -> google/ko -> sigstore/cosign/oci -> sigstore/rekor/pkg/pki/pgp), confirmed with go list -deps and checked against -mode=binary's false-positive class. goreleaser never imports openpgp directly; the ko pipe compiles into every goreleaser binary regardless of config, and the upstream package is unmaintained (Fixed in: N/A). Shipped as originally specified the gate would have been permanently red from its first run on real in-scope tooling, and would have collapsed VULN-02's own signal. Advisory was chosen over a narrow allowlist and over dropping goreleaser from scope (which would have excluded ~322 of the 357 measured modules). The tradeoff — an advisory scan over credentialed tooling reports rather than prevents — was stated and accepted by the maintainer at 04-01's Task 1 checkpoint."
  - "D-04 (BLOCKING) is superseded by this checkpoint decision; the coordinator recorded the reversal in 04-CONTEXT.md."
  - "vuln:selftest itself keeps blocking (fail-capable) semantics even though the scan it drives is advisory: a failure there means the detection mechanism (govulncheck's own exit-3 + advisory-ID classification) broke, not that a new vulnerability was found. An advisory job that silently stops detecting is worse than a blocking one, so this specific meta-check is deliberately allowed to fail the CI job."
  - "GO-2026-5932 in goreleaser is recorded here as a real, accepted, unmitigated exposure in this project's release tooling — the advisory report is the only thing surfacing it. It is not resolved by this plan."

patterns-established:
  - "Three-place advisory stance: tool desc:, CI job name:, CI step name: — VULN-03's realization of the same pattern tools/transcriptfreeze (Phase 3) established for a different guard."
  - "Exit-code classification (0=clean, 3=symbol match, other=scan error) drives REPORTING regardless of blocking/advisory stance — the classification and the build-failure decision are separable concerns."

requirements-completed: [VULN-01, VULN-02, VULN-03]

coverage:
  - id: D1
    description: "task vuln replaces the no-op main-module-only scan with govulncheck -mode=binary over all four tool-modfile binaries (task, goreleaser, govulncheck, actionlint), reporting every finding via ::warning:: annotations and always exiting 0 (advisory)"
    requirement: "VULN-01"
    verification:
      - kind: manual_procedural
        ref: "task vuln (run live; goreleaser reported DETECTED for GO-2026-5932, task/govulncheck/actionlint reported CLEAN, task exited 0)"
        status: pass
    human_judgment: false
  - id: D2
    description: "task vuln:selftest proves the tool-modfile scan's detection mechanism can still fire against a permanent, deliberately-vulnerable program (testdata/vulnredpoc), asserting exit status 3 and the GO-2026-5932 advisory ID in the scan output"
    requirement: "VULN-02"
    verification:
      - kind: manual_procedural
        ref: "task vuln:selftest (run live, exit 0, PASS); disproof — removing the vulnerable call made it fail with the expected message, reverted after observing it"
        status: pass
    human_judgment: false
  - id: D3
    description: "Advisory stance stated in three places: Taskfile.yml vuln desc:, ci.yml tool-vuln job name:, and both ci.yml step name: fields; ci.yml's govulncheck (DIST-03, blocking) job stays untouched and blocking"
    requirement: "VULN-03"
    verification:
      - kind: unit
        ref: "go test ./internal/upgrade/... -run TestRequiredCheckNamesPreserved (pass); git diff .github/workflows/ci.yml shows a pure insertion, 0 deletions"
        status: pass
      - kind: other
        ref: "task lint:actions"
        status: pass
    human_judgment: false

duration: 45min
completed: 2026-08-06
status: complete
---

# Phase 4 Plan 1: Supply-Chain Tool-Modfile Vulnerability Scanning Summary

**Advisory `govulncheck -mode=binary` scan over all four `go.tool.mod`/`go.tool-lint.mod` binaries, with a permanent self-test proving detection still fires — demoted from blocking after the first real run found a genuine, permanently-unfixed exposure in `goreleaser` itself.**

## Performance

- **Duration:** ~45 min (including a mid-plan checkpoint and coordinator decision)
- **Started:** 2026-08-06 (approx.)
- **Completed:** 2026-08-06T19:45:58Z
- **Tasks:** 3
- **Files modified:** 2 (`Taskfile.yml`, `.github/workflows/ci.yml`); 1 created (`testdata/vulnredpoc/main.go`)

## Accomplishments

- Demonstrated the pre-existing D-05 gap: `task vuln` previously ran `govulncheck ./...` over the main module only, naming neither tool modfile.
- Replaced `task vuln`'s body with a real scan: builds `task`, `goreleaser`, `govulncheck` (from `go.tool.mod`) and `actionlint` (from `go.tool-lint.mod`), scans each with `govulncheck -mode=binary`, classifies exit status 0/3/other, and reports every result.
- **Discovered a real, permanently-unfixed vulnerability during Task 1's own verification run:** `goreleaser`'s binary carries a genuine, symbol-reachable match for `GO-2026-5932` (`golang.org/x/crypto/openpgp`, 110 vulnerable symbols), confirmed via `go list -deps` tracing (`goreleaser/cmd → pipe/ko → google/ko → sigstore/cosign/oci → sigstore/rekor/pkg/pki/pgp`) and explicitly checked against `-mode=binary`'s documented false-positive class before escalating. This finding, not a hypothetical, is why VULN-01/03 shipped advisory instead of blocking.
- Added `testdata/vulnredpoc/main.go`, a permanent program that imports and *calls* `openpgp.ReadArmoredKeyRing` (reachable, not just imported — a bare import would only match at module level and exit 0, proving nothing) — a stable target since `GO-2026-5932` is permanently unfixed upstream (`Fixed in: N/A`).
- Added `task vuln:selftest`: builds the red-proof program from `go.tool.mod`, scans it, and asserts exit status is exactly 3 and the output names `GO-2026-5932`. Unlike `vuln`, this target keeps fail-capable semantics — a failure here signals the detection mechanism itself broke, not a new supply-chain finding.
- Added a new `tool-vuln` CI job in `ci.yml`, placed immediately after the existing `govulncheck (DIST-03, blocking)` job, running on `namespace-profile-linux-amd64-4x8`. Job name and both step names state the advisory stance in words. The job is a pure insertion — the `govulncheck` job above it is byte-identical to its pre-plan state.

## Task Commits

1. **Task 1: End-to-end tool-modfile scan (advisory, redesigned mid-task)** - `b7eff89` (feat)
2. **Task 2: VULN-02 — permanent red-proof program and vuln:selftest** - `3308315` (feat)
3. **Task 3: Blocking→advisory CI job, stance stated in three places** - `94aff95` (feat)

_No plan-metadata commit issued separately — this SUMMARY plus STATE.md/ROADMAP.md updates cover it._

## Files Created/Modified

- `Taskfile.yml` - `vuln` target replaced (D-05 no-op → real four-binary scan, advisory); `vuln:selftest` target added
- `.github/workflows/ci.yml` - new `tool-vuln` job added (advisory, two `task` steps); `govulncheck` job byte-identical, unchanged
- `testdata/vulnredpoc/main.go` - new permanent red-proof program (excluded from `go list ./...` per GOLDEN-01, built only via `-modfile=go.tool.mod` by explicit path)

## Decisions Made

**Primary decision — D-04 superseded, tool-modfile scan ships advisory, not blocking (verbatim rationale from the coordinator, recorded here as directed):**

> The tool-modfile scan ships advisory rather than blocking. Implementation surfaced a fact the original decision was made without: `goreleaser`'s own binary carries a real, permanently-unfixed, symbol-reachable match for `GO-2026-5932` (110 vulnerable symbols in `golang.org/x/crypto/openpgp`, via `goreleaser/cmd → pipe/ko → google/ko → sigstore/cosign/oci → sigstore/rekor/pkg/pki/pgp`), confirmed with `go list -deps` and checked against `-mode=binary`'s false-positive class. `goreleaser` never imports `openpgp` directly; the `ko` pipe compiles into every `goreleaser` binary regardless of config, and the upstream package is unmaintained (`Fixed in: N/A`). Shipped as originally specified the gate would have been permanently red from its first run on real in-scope tooling, and would have collapsed VULN-02's own signal. Advisory was chosen over a narrow allowlist and over dropping `goreleaser` from scope (which would have excluded ~322 of the 357 measured modules). The tradeoff — an advisory scan over credentialed tooling reports rather than prevents — was stated and accepted by the maintainer at 04-01's Task 1 checkpoint.

**Secondary decisions:**
- `vuln:selftest` keeps fail-capable (effectively blocking) semantics, deliberately asymmetric with `vuln`'s own advisory stance — proving detection survives is meaningful even when the underlying finding is advisory; an advisory job that silently stops detecting is worse than a blocking one that at least used to work.
- All four binaries stay in scope for `vuln` — no allowlist, no exclusion of `goreleaser`. The advisory demotion applies uniformly; a *new* finding in any binary is exactly as visible (a `::warning::` annotation) as the known `GO-2026-5932` one, which is the point of not carving out an exception.
- `GO-2026-5932` in `goreleaser` is recorded here as a **real, accepted, unmitigated exposure** in this project's release tooling. The advisory `task vuln` report is now the only thing surfacing it in CI — it is not resolved, patched, or worked around by this plan, and no upstream fix exists (`Fixed in: N/A`). Anyone auditing this project's supply-chain posture should read this as an open, known risk, not a closed item.

## Deviations from Plan

### Checkpoint-driven redesign (not a Rule 1-3 auto-fix — required and received an explicit coordinator decision)

**1. [Plan-breaking discovery, escalated per the plan's own deviation_rule] Task 1's blocking design does not survive contact with real data**

- **Found during:** Task 1, first live run of the newly-written `task vuln` target
- **Issue:** As literally designed (any `govulncheck -mode=binary` exit 3 → build fails), the new gate would have been **permanently red from its first CI run** — not from a test artifact, but from `goreleaser`, a real, in-scope binary this project already builds in `check:goreleaser` and the release pipeline. Traced with `go list -deps` to rule out `-mode=binary`'s documented false-positive class before escalating: the import chain from `goreleaser/cmd` through `pipe/ko`, `google/ko`, `sigstore/cosign/oci`, to `sigstore/rekor/pkg/pki/pgp` is real and structurally reachable, not linker-retained dead code.
- **Fix:** Stopped and returned a `checkpoint:decision` (blocking-human gate) with four options (narrow allowlist, drop goreleaser from scope, ship advisory, ship as-designed and accept permanent red). Coordinator selected **Option C — advisory**, superseding D-04. Task 1's `vuln` target, Task 2's `vuln:selftest` framing, and Task 3's CI job stance were all redesigned around this decision before implementation continued.
- **Files modified:** `Taskfile.yml` (both commits), `.github/workflows/ci.yml`
- **Verification:** `task vuln` now exits 0 on the current tree while still reporting the real `goreleaser` finding via `::warning::`; `task vuln:selftest` independently proves detection still fires.
- **Committed in:** `b7eff89`, `3308315`, `94aff95`

---

**Total deviations:** 1 checkpoint-escalated architectural change (not auto-fixed — explicit human decision required and obtained mid-task, per the plan's own `<deviation_rule>`: "If the plan's approach doesn't survive contact, STOP and report rather than improvising — especially anything touching the DIST-03 gate.")
**Impact on plan:** Every acceptance criterion below was re-derived from blocking to advisory semantics where the original wording assumed build-failure behavior; nothing was silently reinterpreted (see next section). `ci.yml`'s DIST-03 gate itself was never touched.

## Acceptance Criteria Re-derived from Blocking to Advisory (explicit, not silent)

Per the coordinator's instruction to say which criteria changed and why, rather than silently reinterpreting them:

| Original (blocking) wording | Re-derived (advisory) behavior | Why |
|---|---|---|
| must_haves truth: "A symbol-matched vulnerability in any of the four tool binaries fails the build: the task exits non-zero when any scan exits 3" | A symbol-matched vulnerability is reported (`::warning::` + full `govulncheck` output) but `task vuln` always exits 0 | D-04 superseded — advisory decision |
| must_haves truth: "A scan exit code that is neither 0 nor 3 ... fails the build rather than being reported as clean" | Reported distinctly (`::warning:: SCAN ERROR`) but never fails the build; still never folded into CLEAN | Same |
| Task 1 acceptance criterion: "Temporarily replacing the scanned-binary path with a binary known to carry a symbol-level match makes `task vuln` exit non-zero; revert after observing it" | Not performed as literally written — under the advisory design `task vuln` is architecturally incapable of exiting non-zero from a symbol match. Verified the advisory-equivalent instead: the real `goreleaser` finding (a genuine symbol-level match) produces a `::warning:: DETECTED` report while the task still exits 0 | The literal criterion assumed blocking semantics that no longer apply; re-derived to the property that actually matters under advisory — detection is visible, build is not blocked |
| must_haves prohibition: "Do not make the new job advisory or add continue-on-error to it (D-04: this one is blocking and says so)" | Superseded — the job is now advisory by design (no `continue-on-error` needed, since `task vuln` never fails on its own; `task vuln:selftest` is deliberately left able to fail, not suppressed) | D-04 superseded |
| VULN-02 "demonstrated RED" (originally: the *build* goes red against a deliberately vulnerable pin) | Re-framed as: *detection* fires and reports — `task vuln:selftest`'s own exit-3 + advisory-ID assertion is the sole proof, since `task vuln` itself never fails by design | Coordinator instruction #3, mirroring Phase 3's advisory-guard precedent |
| Task 3 job/step `name:` wording (originally BLOCKING language) | Re-worded to state ADVISORY in the job name and both step names | Coordinator instruction #1 |

Unchanged: the exit-code classification (0/clean, 3/symbol match, other/scan error) itself; `testdata/vulnredpoc`'s isolation from `go list ./...` and the root `go.mod` (T-04-04 mitigation, unaffected by advisory-vs-blocking); `ci.yml`'s `govulncheck (DIST-03, blocking)` job (byte-identical); the requirement that binary-mode never be described as call-graph-aware; all four binaries staying in scope with no allowlist.

## Issues Encountered

None beyond the plan-breaking discovery documented above (which was the correct outcome of following the plan as written, not a bug in execution).

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- VULN-01/02/03 complete under the advisory stance recorded above.
- `04-03` (not yet executed) is expected to add `tool-vuln` to `internal/upgrade/taskfile_shape_test.go`'s `inScopeJobs` and assert the advisory-stance wording — this plan's job already satisfies `TestWorkflowRunBodiesInvokeTask`'s shape (exactly two `run:` steps, each a bare `task <target>` call), verified even though the job isn't yet in that fixture.
- **Open, unmitigated risk carried forward:** `goreleaser`'s `GO-2026-5932` exposure (via cosign/rekor's PGP verifier) is real, permanent, and now only visible via the advisory `task vuln` report / CI job annotation. No further action was taken on it in this plan. Any future phase auditing release-tooling supply-chain posture should treat this as a known, open item, not something this plan resolved.
- `internal/daemon/` was not touched, per explicit instruction — 04-02 owns that surface and completed in parallel.

## Self-Check: PASSED

- `testdata/vulnredpoc/main.go` — FOUND
- `Taskfile.yml` — FOUND (modified)
- `.github/workflows/ci.yml` — FOUND (modified)
- Commit `b7eff89` — FOUND in `git log --oneline`
- Commit `3308315` — FOUND in `git log --oneline`
- Commit `94aff95` — FOUND in `git log --oneline`

---
*Phase: 04-supply-chain-coverage-daemon-substrate-fixes*
*Completed: 2026-08-06*
