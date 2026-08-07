# Phase 4: Supply-Chain Coverage & Daemon Substrate Fixes - Context

**Gathered:** 2026-08-06
**Status:** Ready for planning

<domain>
## Phase Boundary

Two unrelated pieces of maintenance that share one property: both are gates or
substrate that currently *look* fine and are not.

**In scope:** VULN-01, VULN-02, VULN-03 (the ~400 credentialed CI tooling modules
actually scanned by a job proven able to fail), MAINT-01 (issue #13, the daemon
`-race` failure on the `getppid` seam), MAINT-02 (issue #17, daemon tests failing
under full-suite load), MAINT-03 (the GoReleaser pin mismatch).

**Explicitly NOT in scope:** SPEC-09 / `subscriptions/listen` (Phase 5). Any
further MCP protocol work — Phase 3 closed SPEC-01…SPEC-08 and the server is
correctness-complete for the milestone's core value.

**Calibration (still binding, carried from Phases 2 and 3).** The maintainer
judged this milestone over-engineered and directed the cheap mechanism over the
ceremonious one. That has now paid off twice: Phase 2's relaxed diff bar caught a
semantic regression byte-identity would have buried, and Phase 3 turned out to be
mostly already-delivered once someone probed instead of reading the roadmap.
Same rule here.

</domain>

<decisions>
## Implementation Decisions

### The daemon flakes (MAINT-01, MAINT-02)

- **D-01: Diagnose the shared cause before fixing anything.** ROADMAP criterion 4
  names a single test (`TestRunWatchdogCancelsRunOnSimulatedReparent`), but this
  is what four full-suite runs during Phases 2–3 actually produced — a **rotating
  subset**, never the same one twice, with every member passing in isolation:

  | Run | Failing |
  |---|---|
  | A | `TestDaemonRunWaitsForInFlightFlushBeforeReleasingLock`, `TestSoak`, `TestConvergenceTwoSessions` |
  | B (isolated) | `TestConvergenceTwoSessions` only |
  | C | `TestRunWatchdogCancelsRunOnSimulatedReparent`, `TestDaemonRunWaitsForInFlightFlushBeforeReleasingLock`, `TestDaemonFlushLockRequeueGivesUpPerEpisode` |
  | D | `TestRunWatchdogCancelsRunOnSimulatedReparent`, `TestDaemonSharedWriter`, `TestDaemonRunWaitsForInFlightFlushBeforeReleasingLock`, `TestSoak` |

  Six distinct tests. Memory records this shape from Phase 1 as the "rotating set
  signature," and notes the pre-Phase-2 baseline failed too, with a different
  subset again.

  **Why this changes the work:** criterion 4 says "fixed at the cause rather than
  by isolating the test." If the six share one cause, fixing it satisfies both the
  criterion and the phase goal. If they are independent, fixing only the named one
  leaves five sources of masking noise and the goal — *"stop producing flaky noise
  that masks real regressions"* — goes unmet while the criterion reads satisfied.
  That gap is the thing to avoid.

  **Use the `systematic-debugging` skill.** Do not propose fixes before the cause
  is established. If diagnosis proves the six are genuinely independent, that
  finding is itself a legitimate result — record it and scope the remainder
  honestly rather than forcing a single-cause narrative.

- **D-02: A hypothesis to test first, not a conclusion to implement.**
  `internal/daemon/daemon_test.go:303-304` swaps a **package-level** `getppid`
  variable and restores it via `defer`:

  ```
  origGetppid := getppid
  defer func() { getppid = origGetppid }()
  ```

  A package-level seam mutated by one test while another test's goroutine reads it
  is a data race whose victim depends on timing — precisely the shape of a
  rotating failure set that disappears under isolation. Issue #13 (MAINT-01) names
  this same seam, so **MAINT-01 and MAINT-02 may share one root cause**.
  Attractive, and therefore worth disproving properly rather than assuming. The
  diagnosis owns this question.

- **D-03: The race must be demonstrated before the fix, not inferred from a green
  run afterward.** This is criterion 3's explicit wording and the repo's standing
  rule — a gate is not trusted until demonstrated RED against a confirmed-applied
  mutation. A `-race` run that goes green after a change proves nothing unless the
  same harness was seen to go red before it. For a *load-dependent* flake this is
  harder than usual: a single red run is weak evidence. Establish a reproduction
  with a failure rate you can actually measure (repeat count, `-race`, parallel
  package load) so "fixed" means the rate went to zero, not that one run passed.

### Supply-chain scanning (VULN-01, VULN-02, VULN-03)

- **D-04 — SUPERSEDED 2026-08-06 at 04-01's checkpoint. The scan ships ADVISORY,
  not blocking.** Read the replacement below; the original text is kept only so
  the reversal is legible.

  **Original (no longer in force):** the scan is BLOCKING and says so, because
  these modules execute as credentialed CI tooling — repository write access and
  secrets — which is exactly the surface worth failing a build over.

  **What changed:** implementing it surfaced a fact the decision was made without.
  `goreleaser`'s own binary carries a **real, permanently-unfixed,
  symbol-reachable** match for `GO-2026-5932` — 110 vulnerable symbols in
  `golang.org/x/crypto/openpgp`, arriving through `goreleaser/cmd → pipe/ko →
  google/ko → sigstore/cosign/oci → sigstore/rekor/pkg/pki/pgp`, confirmed with
  `go list -deps` and specifically checked against `-mode=binary`'s known
  false-positive class. `goreleaser` never imports `openpgp` directly; the `ko`
  pipe is compiled into every `goreleaser` binary regardless of config, and the
  upstream package is unmaintained (`Fixed in: N/A`).

  Shipped as originally specified, the gate would have been **permanently red
  from its first run** on real in-scope tooling this repo already builds — and it
  would have collapsed VULN-02's own signal, since "the gate can fire" stops
  being informative once `task vuln` always exits 3.

  **Maintainer decision at the checkpoint: ship advisory.** Detection and the full
  report stay intact; the job never fails the build. Chosen over a narrow
  `GO-2026-5932` allowlist (which would have kept the gate blocking for anything
  new) and over dropping `goreleaser` from scope (which would have excluded ~322
  of the 357 measured modules). The tradeoff was stated at the checkpoint and
  accepted: an advisory scan over credentialed tooling reports rather than
  prevents.

  **Consequences that follow, and must be honored:**
  - The stance stated in all three places (tool output, `Taskfile.yml` `desc:`,
    CI step `name:`) is **ADVISORY**, per VULN-03. Nothing may read as blocking.
  - **VULN-02's "demonstrated RED" now means the detection fires and reports**,
    not that the build fails — the same distinction Phase 3's advisory transcript
    guard drew. The self-test asserts the report, not an exit code that fails CI.
  - This now **matches** the D-03 transcript guard's stance rather than
    deliberately differing from it. The earlier note about two coexisting stances
    no longer applies.
  - `ci.yml`'s existing blocking `govulncheck (DIST-03, blocking)` gate over the
    main module is **unaffected and stays blocking**. Advisory applies only to the
    new tool-modfile scan.
  - `GO-2026-5932` is a real, accepted, unmitigated exposure in release tooling.
    It should be recorded as such — the advisory report is now the only thing
    surfacing it.

- **D-05: `task vuln` is confirmed a no-op duplicate and must be replaced, not
  supplemented.** Measured, not assumed: `Taskfile.yml:148-155` runs
  `govulncheck ./...` over the **main module only**, and its own description
  concedes it "does NOT replace ci.yml's blocking govulncheck gate … this target
  exists so the same check is reachable locally." It never touches `go.tool.mod`
  or `go.tool-lint.mod`. Both files exist. The replacement uses `-mode=binary`
  over binaries built from those manifests (VULN-01's stated mechanism), because
  `govulncheck` cannot call-graph-analyze a module it does not build.

### MAINT-03

- **D-06: Align the GoReleaser pins.** `release.yml:56` sets
  `GORELEASER_VERSION: "v2.17.0"` with a comment tying it to "the version
  validated locally in Plan 08-01" for build-config reproducibility. `ci.yml`
  names v2.17.1. Trivial mechanically — the only judgment is **which way to
  align**, and the comment suggests release.yml's pin is the deliberate one. Do
  not silently bump the validated release pin to match CI; if CI moves down
  instead, say why.

### Claude's Discretion

- Whether the tool-modfile scan is one CI job or two (`go.tool.mod` and
  `go.tool-lint.mod` are different risk surfaces even though both are blocking).
- The known-vulnerable pin used for VULN-02's RED demonstration, and whether it
  lands as a throwaway local proof or a permanent self-test. Leaning: a permanent
  self-test in the repo's established archtest/self-defeat idiom, since this repo
  has shipped gates that could not fire and the self-test is what catches that.
- Whether the daemon fix warrants a permanent load-reproduction target
  (`task test:daemon-load` or similar) once a reproduction exists.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Scope
- `.planning/ROADMAP.md` § "Phase 4" — the five success criteria
- `.planning/REQUIREMENTS.md` § VULN-01…03, MAINT-01…03
- `.planning/STATE.md` § Blockers/Concerns — open issues #13–#17 and the residual darwin release-path check

### The daemon surface
- `internal/daemon/daemon_test.go:296-310` — the injectable `getppid` seam and its package-level swap (D-02's hypothesis)
- `internal/daemon/watchdog_windows.go:8-10` — the cross-platform `os.Getppid()` capture and reparent semantics
- The six tests named in D-01, all in `internal/daemon`

### Supply chain
- `Taskfile.yml:148-155` — the `vuln` target, whose own description concedes it duplicates the CI gate
- `go.tool.mod`, `go.tool-lint.mod` — the ~400 modules currently unscanned
- `.github/workflows/ci.yml` — the existing blocking "govulncheck (DIST-03, blocking)" gate this must not disturb; also the v2.17.1 GoReleaser pin
- `.github/workflows/release.yml:54-56` — `GORELEASER_VERSION: "v2.17.0"` and the Plan-08-01 provenance comment

### Repo standards
- `.claude/CLAUDE.md` — `govulncheck` is the chosen scanner precisely because it is call-graph-aware and low-noise; do not substitute a naive SCA tool
- `Taskfile.yml` is the single definition of every CI job body — `TestWorkflowRunBodiesInvokeTask` enforces it. Job bodies change there, never inline in a workflow.
- Phase 3's `tools/transcriptfreeze` advisory-stance change is the in-repo precedent for stating a gate's stance explicitly (D-04's sibling)

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable assets
- **The archtest self-defeat idiom** (`internal/mcp/archtest`, `internal/cli/archtest`, and Phase 2's re-pointed non-vacuity self-tests) is the established way this repo proves a guard can still fire. VULN-02's RED demonstration should follow it rather than inventing a mechanism.
- **`task vuln` already exists** as a target name — replacing its body keeps the developer-facing entry point stable.

### Established patterns
- A gate is not trusted until demonstrated RED against a confirmed-applied mutation. Standing rule; VULN-02 makes it an explicit criterion for the first time.
- `Taskfile.yml` is the single source of CI job bodies.
- Stances are stated out loud (Phase 3's advisory guard set the precedent VULN-03 now generalizes).

### Integration points
- `ci.yml`'s existing blocking `govulncheck (DIST-03, blocking)` gate over the main module **must keep working unchanged** — VULN-01 adds coverage, it does not replace that gate.
- `internal/daemon` is the only package this phase's MAINT work touches. It does **not** import `internal/mcp`, which is why its flakes were correctly excluded from Phases 2 and 3 verdicts — that exclusion ends here.

</code_context>

<specifics>
## Specific Ideas

- The rotating-set evidence in D-01 is real observation from this session, not a
  reconstruction. Four runs, six distinct tests, all passing isolated. Use it as
  the diagnosis's starting dataset rather than re-gathering it from scratch.
- Three consecutive phases have hand-excluded daemon failures from their verdicts.
  That exclusion is the cost this phase is paying down.

</specifics>

<deferred>
## Deferred Ideas

- **SPEC-09** — Phase 5, untouched here.
- **go-sdk issue #976** (`code: 0` on the pre-initialize rejection) — upstream, tracked, not ours.
- **The `legacy-unsupported-2026-07-28` rename and the `annotations` hint correction** — declined twice already (Phases 2 and 3). Not resurrected here; this phase touches no transcripts.
- **The residual darwin release-path check** — `release.yml`'s goreleaser/cosign/SLSA steps still have not run on the macOS runner class outside the permanent canary. MAINT-03 touches the GoReleaser pin but does not close that gap.

</deferred>

---

*Phase: 04-supply-chain-coverage-daemon-substrate-fixes*
*Context gathered: 2026-08-06*
