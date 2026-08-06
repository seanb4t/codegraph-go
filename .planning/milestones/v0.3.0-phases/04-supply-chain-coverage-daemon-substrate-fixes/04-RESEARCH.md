# Phase 4: Supply-Chain Coverage & Daemon Substrate Fixes - Research

**Researched:** 2026-08-06
**Domain:** CI/CD supply-chain scanning (Go tool-modfile vulnerability coverage) + Go concurrency/test-infrastructure diagnosis (daemon package flake set)
**Confidence:** HIGH (both halves — every load-bearing claim below was reproduced or read from source this session, not assumed)

## Summary

This phase is two unrelated pieces of maintenance. The supply-chain half (VULN-01…03) is mechanically straightforward once the `govulncheck -mode=binary` contract is understood precisely: it is **symbol-level, not call-graph-level** — a materially different (weaker) guarantee than the main-module's existing blocking gate, and that difference must be stated in the implementation, not silently inherited from CLAUDE.md's "call-graph-aware" framing. The mechanism was built and run live this session against real binaries compiled from `go.tool.mod`/`go.tool-lint.mod` — it works exactly as documented, and a real, currently-unfixable vulnerability (`GO-2026-5932`, `golang.org/x/crypto/openpgp`) already sits latent in the closure, giving VULN-02's RED demonstration a real target instead of an invented one.

The daemon half (MAINT-01/02) is the highest-value finding in this research. **D-02's getppid-race hypothesis is real but not the root cause it was framed as.** I reproduced the full six-plus-one-test rotating failure set at a measured **100% failure rate** (8/8 runs) using `go test -count=1 ./...` unfiltered, and separately reproduced two genuine `-race`-detected data races (on `getppid` and, previously unnoticed, on `registryDir` too) under `-race` + full-suite load. Reading the actual race-detector stack traces end to end shows the true mechanism: it is **not** a plain concurrent-access race exposed by scheduling luck. It is a **secondary effect of `t.Fatalf`'s `runtime.Goexit()` bypassing the daemon's own goroutine-join discipline** whenever a test's own fixed-duration timeout fires first. Five of the seven rotating tests fail with a plain timeout and **no race at all**; two (`getppid`- and `registryDir`-touching tests) additionally race, *only when* their own timeout fires before `Daemon.Run`'s internal `stop()`/`Deregister()` join completes. One shared proximate trigger (CPU/scheduler contention under full-suite parallel load blowing fixed wall-clock budgets), one derivative secondary mechanism (orphaned background goroutines outliving a `t.Fatalf`'d test and racing package-level test seams in later tests). Neither is a data race intrinsic to production code reachable outside tests.

**Primary recommendation:** For VULN-01, replace `task vuln`'s body (Taskfile.yml:148-155) with a `-mode=binary` scan over binaries built from `go.tool.mod` (3 tool directives) and `go.tool-lint.mod` (1), keeping the same task name so the existing single-definition-property test (`TestWorkflowRunBodiesInvokeTask`) needs only an `inScopeJobs` addition, not new machinery. For MAINT-01/02, do not chase the getppid race as an isolated bug — the fix surface is the test package's goroutine-join discipline on the failure path (or, alternatively, further isolating/generously-budgeting daemon's CI execution), and any fix must be demonstrated against the **measured, repeatable** reproduction command in this document, not a single green run.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Tool-modfile vulnerability scanning (VULN-01/02/03) | CI/CD Pipeline (`ci.yml`, `Taskfile.yml`) | Build Tooling (`go.tool.mod`, `go.tool-lint.mod`) | The scan is a pipeline gate; what it scans is defined by the isolated tool modfiles, not application code |
| Daemon watchdog/lock/registry lifecycle (MAINT-01/02 production surface) | Concurrency/Runtime (`internal/daemon`) | — | Package-level test seams (`getppid`, `registryDir`) only exist for test injection; no production caller mutates them |
| Daemon test reliability (MAINT-01/02 defect surface) | Test Infrastructure (`internal/daemon/*_test.go`) | CI/CD Pipeline (`ci.yml`'s `task test:daemon`/`task test:race` steps, runner class) | The observed failures are test-harness timing/join defects, not defects in `daemon.go`'s public contract |
| GoReleaser version pin (MAINT-03) | CI/CD Pipeline (`release.yml`, `ci.yml` via `go.tool.mod`) | Build Tooling (`go.tool.mod`'s `goreleaser` require line) | Two independent pin sites for the same tool; reconciliation is a pipeline-configuration decision |

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| VULN-01 | `govulncheck` covers `go.tool.mod`/`go.tool-lint.mod` via `-mode=binary` over built binaries, replacing the no-op `task vuln` | Mechanism built and run live this session against real `task`/`goreleaser`/`govulncheck`/`actionlint` binaries; exact module counts measured; exit-code contract confirmed |
| VULN-02 | Job demonstrated RED against a deliberately known-vulnerable pin before being trusted | A real, already-present, permanently-unfixed vulnerability (`GO-2026-5932`) identified and used to produce a genuine exit-code-3 RED run this session — no fabricated pin needed |
| VULN-03 | Job's blocking-vs-advisory stance stated explicitly in the workflow | `tools/transcriptfreeze`'s advisory-stance precedent read at the source level; the same two-place (Taskfile `desc:` + CI step `name:`) pattern transfers directly |
| MAINT-01 | Issue #13 `-race` failure on the `getppid` seam fixed, race demonstrated first | Race reproduced and its exact mechanism (Goexit bypassing goroutine join) identified via source reading + race-detector stack traces, not inferred |
| MAINT-02 | Issue #17 `TestRunWatchdogCancelsRunOnSimulatedReparent` fixed at the cause under full-suite load | Full rotating-set reproduced at 100% (8/8) measured failure rate; shared vs. independent causes for all 6 named tests (+1 previously-unnamed) explicitly separated |
| MAINT-03 | GoReleaser pin agreement between `ci.yml` (transitively, via `go.tool.mod`) and `release.yml` | Both pin sites located precisely; v2.17.1 changelog read — confirmed a pure bugfix release irrelevant to this repo's build matrix |
</phase_requirements>

## Standard Stack

No new libraries are required by this phase. Both halves reuse tooling already present in the repository.

### Core (already present, reused)
| Library | Version (measured) | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `golang.org/x/vuln/cmd/govulncheck` | v1.6.0 [VERIFIED: go.tool.mod:389, confirmed via `govulncheck -version` this session] | Vulnerability scanning, `-mode=binary` | Already the repo's chosen scanner (CLAUDE.md); this phase adds a second invocation mode, not a new tool |
| `go.uber.org/goleak` | v1.3.0 [VERIFIED: /Users/sean/go/pkg/mod/go.uber.org/goleak@v1.3.0, path observed in this session's race-detector stack traces] | Goroutine-leak detection, `internal/daemon`'s `TestMain` | Already gates the package; relevant context for any MAINT-01/02 fix touching goroutine lifecycle |
| `github.com/go-task/task/v3`, `github.com/goreleaser/goreleaser/v2`, `github.com/rhysd/actionlint` | v3.52.0, v2.17.1, v1.7.12 [VERIFIED: go.tool.mod:212,234 and go.tool-lint.mod:34, all three built and run successfully this session] | The three tool-modfile binaries VULN-01 must scan | Already the repo's pinned build tooling; VULN-01 builds and scans them, does not add or replace them |

### Installation
No `go get`/`npm install` needed. If VULN-02's self-test lands as a permanent Go program importing `golang.org/x/crypto/openpgp` (see VULN-02 findings below), that import must be promoted from `// indirect` to a direct require in `go.tool.mod` — a `go mod tidy -modfile=go.tool.mod` after adding the import, not a new dependency addition.

## Package Legitimacy Audit

**Not applicable in the traditional sense.** This phase does not install new external packages — VULN-01 builds and scans binaries from modules **already declared** in `go.tool.mod`/`go.tool-lint.mod` (isolated tool-only modfiles per D-03, unmanaged by Dependabot/Renovate, manually bumped). No `npm install`/`go get` of a new package is part of this phase's scope. The one candidate addition — promoting the already-indirect `golang.org/x/crypto` to a direct require for VULN-02's self-test — is not a new package (it is already vendored transitively at v0.54.0 via `goreleaser`'s own dependency tree) and is trivially legitimate (official `golang.org/x/*` module, `go.dev`-hosted).

**Packages removed due to [SLOP] verdict:** none.
**Packages flagged as suspicious [SUS]:** none.

## Architecture Patterns

### System Architecture Diagram — VULN-01/02/03 data flow

```
                    ┌─────────────────────────────┐
                    │   go.tool.mod (3 tools)      │
                    │   go.tool-lint.mod (1 tool)  │
                    └──────────────┬───────────────┘
                                   │  go build -modfile=<X>
                                   ▼
        ┌───────────────┬───────────────┬───────────────┬────────────────┐
        │   task-bin     │ goreleaser-bin│ govulncheck-bin│  actionlint-bin│
        │  (122 deps)    │  (322 deps)   │   (4 deps)     │   (11 deps)    │
        └───────┬────────┴───────┬───────┴───────┬────────┴────────┬──────┘
                │                │               │                 │
                └────────────────┴───────┬───────┴─────────────────┘
                                          │  govulncheck -mode=binary <bin>
                                          ▼
                              ┌───────────────────────┐
                              │ symbol-table scan      │
                              │ against vuln.go.dev DB │
                              └───────────┬────────────┘
                                          │
                         exit 0 (clean or module-only)  |  exit 3 (symbol match: RED)
                                          │
                                          ▼
                         `task vuln` (replaces the no-op body) — new CI job,
                         BLOCKING, stance stated in Taskfile desc: + CI step name:
                         (mirrors "govulncheck (DIST-03, blocking)")
```

### System Architecture Diagram — daemon flake causal chain (empirically confirmed this session)

```
go test ./...  (unfiltered, ~51 pkgs, default cross-package parallelism)
        │
        ▼
CPU/scheduler contention on the runner
        │
        ▼
internal/daemon test's fixed wall-clock timeout fires FIRST
(200ms / 5s / 10s windows chosen assuming uncontended scheduling)
        │
        ├──► test has no getppid/registryDir mutation ──► plain t.Fatalf, test FAILS
        │     (TestSoak, TestDaemonSharedWriter,               (no race — 5 of 7 tests)
        │      TestConvergenceTwoSessions, etc.)
        │
        └──► test mutated getppid/registryDir via `defer`/`t.Cleanup`
                    │
                    ▼
             t.Fatalf → runtime.Goexit() unwinds THIS test's stack,
             running its deferred/cleanup restores IMMEDIATELY —
             WITHOUT waiting for the orphaned `go func(){ d.Run(ctx) }()`
             goroutine (still blocked, still ticking its watchdog) to
             reach Run's OWN internal stop()/Deregister() join
                    │
                    ▼
             package-level var WRITE (test's restore) races the
             orphaned goroutine's ongoing READ  ──►  -race fires
             (TestRunWatchdogCancelsRunOnSimulatedReparent; and,
              cross-test, an EARLIER failed test's orphan can even
              race a LATER unrelated test — confirmed against
              TestWatchdogCancelsOnReparent this session)
```

### Recommended Project Structure (no new directories needed)
```
Taskfile.yml           # `vuln:` task body replaced (D-05) — same name, new mechanism
.github/workflows/
  ci.yml                # new job added (name/desc must say BLOCKING); existing
                         # "govulncheck (DIST-03, blocking)" job untouched
internal/upgrade/
  taskfile_shape_test.go # `inScopeJobs` needs the new job added if its run: body
                          # is `task vuln` and single-definition coverage is wanted
internal/daemon/         # MAINT-01/02 fix surface — test files' goroutine-join
                          # discipline, not daemon.go's public contract
tools/                   # candidate location for VULN-02's permanent self-test,
                          # mirroring tools/transcriptfreeze's own layout
```

### Pattern 1: govulncheck -mode=binary over a built tool binary
**What:** Build the tool from its isolated modfile, then scan the resulting binary — never the modfile's source, since these tools' source lives outside this repo.
**When to use:** Any modfile-isolated build tool whose vulnerability exposure needs coverage without merging its dependency graph into the main module (exactly D-05's stated reason: `govulncheck` cannot call-graph-analyze a module it does not build).
**Example (run live this session):**
```bash
# Source: this session's direct verification, not documentation alone
GOWORK=off go build -modfile=go.tool.mod -o /tmp/task-bin github.com/go-task/task/v3/cmd/task
GOWORK=off go tool -modfile=go.tool.mod govulncheck -mode=binary /tmp/task-bin
# => "No vulnerabilities found." exit 0
#    (one module-level-only GO-2026-5932 hit noted separately — not blocking,
#     because govulncheck's own code doesn't call the vulnerable symbols)
```

### Pattern 2: VULN-02's RED demonstration via a real, permanently-unfixed advisory
**What:** `golang.org/x/crypto/openpgp` (all sub-packages) carries advisory `GO-2026-5932` — "unmaintained, unsafe by design... Fixed in: N/A" [CITED: https://pkg.go.dev/vuln/GO-2026-5932]. Because there is no fix, a program that imports and calls it stays vulnerable indefinitely — ideal for a stable, permanent self-test (never needs re-pinning to stay red).
**When to use:** VULN-02's "demonstrated RED against a deliberately known-vulnerable pin" requirement.
**Example (built and run live this session, exit code captured):**
```go
// Source: verified live this session against golang.org/x/crypto@v0.54.0
// (the exact version already pinned indirectly in go.tool.mod)
package main

import (
	"fmt"
	"strings"

	"golang.org/x/crypto/openpgp"
)

func main() {
	_, err := openpgp.ReadArmoredKeyRing(strings.NewReader(""))
	fmt.Println("openpgp.ReadArmoredKeyRing returned:", err)
}
```
```
$ govulncheck -mode=binary ./redpoc-bin
Vulnerability #1: GO-2026-5932
  Vulnerable symbols found:
    #1: armor.Decode
    #2: armor.lineReader.Read
    ... (28 more, "-show traces" for the rest)
Your code is affected by 1 vulnerability from 1 module.
$ echo $?
3
```
This matches the repo's own archtest/self-defeat idiom (`internal/mcp/archtest`, `internal/cli/archtest`) — a permanent, repo-owned proof the gate can fire, landing as a Go test asserting `ExitCode()==3` (or equivalent), not a throwaway manual step.

### Pattern 3: stating a gate's blocking/advisory stance (VULN-03, precedent read from source)
**What:** `tools/transcriptfreeze/main.go:1-26` [VERIFIED: read this session] documents its ADVISORY stance directly in the `main` package doc comment (exit-code contract: 0 always, even on a detected collision — the report replaces the exit code as the deliverable). The stance is stated in **three** places, all confirmed by direct read:
1. Tool source doc comment (`main.go:7-26`)
2. `Taskfile.yml:293-311`'s `check:transcript-freeze` `desc:` — "ADVISORY since 03-02... reports, but never fails the build"
3. `ci.yml`'s `transcript-freeze` job step `name:` — `"Anti-regeneration guard (D-03, advisory since 03-02)"`

**Transfers directly to VULN-03:** the new job should mirror pattern 2 exactly but with "BLOCKING" instead of "advisory since 03-02" — Taskfile `desc:` states it, and the CI job/step `name:` states it, exactly as the existing `"govulncheck (DIST-03, blocking)"` job already names itself. No new mechanism needed.

### Anti-Patterns to Avoid
- **Assuming `-mode=binary` inherits source mode's call-graph precision:** it does not. [CITED: https://pkg.go.dev/golang.org/x/vuln/cmd/govulncheck] "Because Go binaries do not contain detailed call information, govulncheck cannot show the call graphs... It may also report false positives for code that is in the binary but unreachable." CLAUDE.md's rationale for choosing `govulncheck` ("call-graph-aware, low-noise") is true of the *existing* main-module gate (source mode) but only partially true of VULN-01's new binary-mode gate — this must be stated in the new job's documentation, not silently assumed to carry over.
- **Treating a module-level-only govulncheck finding as blocking-worthy:** confirmed empirically this session — `govulncheck -mode=binary` on `task-bin` reports `golang.org/x/crypto@v0.54.0`/GO-2026-5932 as present in the module graph but **exits 0** because `task`'s own code doesn't call the vulnerable symbols. Only a symbol/package-level match (exit 3) is a real RED. VULN-02's demonstration must produce a symbol-level hit, not merely reference an old module version.
- **Chasing the getppid race as an isolated MAINT-01 bug in isolation from MAINT-02:** the race only manifests as a *consequence* of the same timeout pressure that fails the other five tests. A fix that patches only the `getppid` synchronization (e.g., an atomic pointer) without addressing the underlying orphaned-goroutine-on-Goexit pattern will still leave `registryDir` (and any future package-level test seam) exposed to the identical failure shape.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Vulnerability scanning of built tool binaries | A custom dependency-version-diff script against OSV | `govulncheck -mode=binary` (already chosen, already in `go.tool.mod`) | Symbol-level reachability analysis, even if coarser than source mode, is still far more precise than a naive version-list scan; reuses the existing DB client/cache logic |
| Proving a gate can fire | An ad-hoc manual test run, documented only in a plan/commit message | A permanent Go test in the archtest/self-defeat idiom (`tools/vulnselftest` or similar) | This repo has shipped multiple gates that could not fire (transcript-freeze's own history, the `rg -qv` inversion, the 51.5%-stale bench baseline per STATE.md) — a manual one-time proof rots; a permanent self-test does not |
| Daemon test timing coordination | Ad-hoc `time.Sleep` calibration or larger timeout constants alone | Explicit goroutine-join on every return path (success AND failure/timeout), mirroring `Daemon.Run`'s own `stop()`-blocks-until-`done`-closes discipline, applied at the TEST level too | Enlarging timeouts treats the symptom (a plain timeout fails less often) but does not close the orphaned-goroutine race window this session traced precisely; a join closes both |

**Key insight:** Both halves of this phase are variations on one theme — a *documented gate that cannot actually fire the way its documentation implies*. VULN-01 replaces a gate (`task vuln`) proven this repo already knows is a no-op duplicate; MAINT-01/02 diagnoses a "race" that's real but framed at the wrong level, which would have led to a fix that patches one symptom (`getppid`) while leaving its sibling (`registryDir`) and the actual mechanism untouched.

## Common Pitfalls

### Pitfall 1: Believing a single green `-race` run proves the getppid fix
**What goes wrong:** 30 isolated `-race` runs of `TestRunWatchdogCancelsRunOnSimulatedReparent` alone (this session) and 3 isolated full-package `-race` runs produced **zero** race reports — the race only appeared once combined with full-suite (`./...`, ~51 packages) parallel load. A fix validated only in isolation will look green and still be broken under CI's actual contention profile.
**Why it happens:** The race requires an orphaned goroutine to still be alive when a write happens — both conditions need genuine scheduler pressure to manifest with any regularity.
**How to avoid:** Validate any fix against the measured reproduction command below (Validation Architecture), not an isolated single-test run.
**Warning signs:** "It passed 5 times in a row locally" without the `./...` unfiltered full-suite context.

### Pitfall 2: Assuming CI's existing isolation (`task test:daemon`, `task test:race -p 1`) already fully protects against this
**What goes wrong:** Isolating `internal/daemon` from *other packages* (which `Taskfile.yml`'s `test:daemon`/`test:race` targets already do, and which `ci.yml` inherits by running them as separate sequential steps) prevents *cross-package* contention. It does **not** prevent the daemon package's *own* tests from self-contending — `test:race` alone still runs `-race` (5-20x CPU/memory overhead) across four packages (`internal/daemon`, `internal/watch`, `internal/cli`, `internal/mcp`) on `namespace-profile-linux-amd64-4x8` (a **4-vCPU** runner per `ci.yml:48`). D-01's evidence ("four full-suite runs during Phases 2-3") predates none of this isolation — it is unclear from this research alone whether the isolation already fully protects the *current* CI runner class or only reduces the failure rate. This is a genuine open question for the planner, not something safe to assume resolved.
**Why it happens:** My local reproduction needed 16 real CPU cores' worth of *cross-package* contention to hit reliably; CI's 4-vCPU runner may need far less concurrent activity to hit the same contention *within* a single package's own goroutine load.
**How to avoid:** Do not close MAINT-02 on "the isolation already exists, therefore it's fine" — validate on the actual CI runner class or a resource-constrained local approximation (e.g., `GOMAXPROCS=4`).
**Warning signs:** A fix that only re-verifies on a large local dev machine.

### Pitfall 3: Fixing only the named test (ROADMAP criterion 4's literal wording)
**What goes wrong:** Criterion 4 names only `TestRunWatchdogCancelsRunOnSimulatedReparent`. Fixing that one test's specific timeout margin (e.g., raising it past 10s) would make *that* test pass more often without touching the shared timeout-budget-vs-contention mechanism affecting the other six.
**Why it happens:** The rotating set makes any single run's failing subset look like an isolated defect.
**How to avoid:** Per D-01/CONTEXT.md, treat all 7 (not 6 — `TestWatchdogCancelsOnReparent` in `watchdog_test.go` was also observed to fail+race this session, previously unlisted) as one investigation.
**Warning signs:** A fix that touches only `daemon_test.go:340-349`'s 10-second window.

## Code Examples

### Verified failure reproduction (run this session, exact commands)
```bash
# Source: this session's direct execution against HEAD (38a8486)
# Measured: 8/8 runs (100%) produced at least one internal/daemon FAILURE
for i in $(seq 1 8); do go test -count=1 ./... ; done
```
Rotating subsets observed across the 8 runs (union = 7 distinct tests, confirming and extending D-01's 6):
`TestDaemonSharedWriter`, `TestDaemonRunWaitsForInFlightFlushBeforeReleasingLock`, `TestDaemonFlushLockRequeueGivesUpPerEpisode`, `TestRunWatchdogCancelsRunOnSimulatedReparent`, `TestConvergenceTwoSessions`, `TestSoak`, and (new, under `-race`+full-suite specifically) `TestWatchdogCancelsOnReparent`.

### Verified race reproduction (run this session, exact command)
```bash
go test -race -count=1 ./...
# WARNING: DATA RACE
#   Write at .../daemon_test.go:304 (deferred `getppid = origGetppid`, invoked
#     via runtime.Goexit() from t.Fatalf at daemon_test.go:348 — the test's
#     OWN 10s timeout firing)
#   Previous read at .../watchdog_posix.go:15 (parentChanged -> getppid(),
#     inside startWatchdog.func1 -- the STILL-RUNNING watchdog ticker
#     goroutine, never joined because Run() itself never returned)
#
# A SECOND, independent race on registryDir (registry_test.go:21 vs
# registry.go:64's Deregister) fires in the SAME run via the identical
# mechanism -- confirming this is a general pattern (any package-level
# test seam mutated by defer/t.Cleanup), not specific to getppid.
```

### Disproof attempt that failed (D-02's hypothesis tested in isolation)
```bash
# Source: this session — 30 iterations, isolated, zero races found
go test -race -run TestRunWatchdogCancelsRunOnSimulatedReparent -count=30 ./internal/daemon/
# ok  	github.com/seanb4t/codegraph-go/internal/daemon	32.893s
go test -race -count=3 ./internal/daemon/...
# ok  	github.com/seanb4t/codegraph-go/internal/daemon	43.698s (x3)
```
This is the evidence that the race is **not** a plain always-present concurrency defect in `getppid`'s own synchronization — it requires the specific precondition (a timeout-triggered `t.Fatalf` racing an orphaned goroutine) that only manifests under contention.

### GoReleaser version check (run this session)
```bash
$ grep -rn "2\.17" .github/workflows/*.yml go.tool.mod
go.tool.mod:234:	github.com/goreleaser/goreleaser/v2 v2.17.1 // indirect
.github/workflows/release.yml:56:  GORELEASER_VERSION: "v2.17.0"
$ GOWORK=off go tool -modfile=go.tool.mod goreleaser -v
GitVersion: v2.17.1
```
`ci.yml` itself never spells out a version literal — its `goreleaser-check` job runs `task check:goreleaser`, which transitively resolves to whatever `go.tool.mod` pins (v2.17.1). `release.yml`'s `GORELEASER_VERSION: "v2.17.0"` is a **separate, standalone binary install** via `goreleaser/goreleaser-action`, used only for the actual release build — it is not the same code path as `task check:goreleaser`/`task check:darwin-release-build`.

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|---------------|--------|
| `task vuln` scanning only the main module (`./...`) | `-mode=binary` over built tool-modfile binaries | This phase (VULN-01) | Extends coverage to the ~357 modules (measured, see below) actually linked into `task`/`goreleaser`/`govulncheck`/`actionlint`, previously entirely unscanned |
| GoReleaser v2.17.0 (release.yml, "validated locally in Plan 08-01") | v2.17.1 (go.tool.mod, drifted independently) | Unclear when go.tool.mod moved — no commit citation available in this research pass; flagged for the planner to `git log -S` if provenance matters | v2.17.1's changelog [CITED: https://github.com/goreleaser/goreleaser/releases/tag/v2.17.1] is a pure bugfix (GORISCV64/GOPPC64 target-name parsing, build-target sorting) — no feature or behavior this repo's linux/darwin/windows amd64/arm64 matrix depends on |

**Not deprecated, but under-characterized until this session:** `govulncheck -mode=binary`'s exit-code contract. Confirmed via reading `golang.org/x/vuln@v1.6.0/internal/scan/errors.go` [VERIFIED: /Users/sean/go/pkg/mod/golang.org/x/vuln@v1.6.0/internal/scan/errors.go:17, quoted: `errVulnerabilitiesFound = &exitCodeError{message: "vulnerabilities found", code: 3}`]: exit 0 = clean or module-only match (not blocking), exit 2 = usage error, **exit 3 = a real, symbol-matched vulnerability (blocking-worthy)**.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|----------------|
| A1 | The ~400-module figure in ROADMAP/VULN-01 is best matched by the **357** unique modules actually linked across all 4 built tool binaries (measured via `go version -m` on real builds this session), not the 377 raw `require`-block entries or the 1002-entry full transitive MVS graph (`go list -m all -mod=mod`, likely inflated by test-only/unused-in-build modules) | Q3, Standard Stack | Low — all three numbers are the same order of magnitude; this affects only which figure the plan/docs quote, not the mechanism |
| A2 | No commit history was traced for *when* `go.tool.mod`'s goreleaser pin drifted from v2.17.0 to v2.17.1, or whether it was deliberate | State of the Art (MAINT-03) | Low-medium — if it was a deliberate `go get -u` bump for an unrelated reason, "align down" may fight a later intentional change; a `git log -p go.tool.mod` check before deciding direction is cheap insurance |
| A3 | Whether `ci.yml`'s current 4-vCPU runner class (`namespace-profile-linux-amd64-4x8`) is *itself* sufficient to reproduce the timeout-driven failures even with `internal/daemon` fully isolated from other packages (Pitfall 2) was not directly tested — this session's reproduction used a 16-core local machine and needed full cross-package (`./...`) load | Common Pitfalls (Pitfall 2) | Medium — if the CI runner alone is *not* contended enough, MAINT-02 may already be a non-issue in practice on `main`'s actual CI, and the fix's real value is defensive/local-dev-only. If it *is* contended enough, the current isolation is insufficient and the planner needs a different mitigation (e.g., higher-vCPU runner class for this step, or the goroutine-join fix). Recommend the planner check recent CI run history for these exact failures before committing to a fix's scope. |
| A4 | `golang.org/x/crypto/openpgp`'s `GO-2026-5932` advisory will remain permanently unfixed ("Fixed in: N/A") for the life of any self-test built against it | Pattern 2 (VULN-02) | Low — the advisory itself states no fix is planned (package is deprecated in favor of `ProtonMail/go-crypto`); a future upstream fix would only require re-picking a target, not redesigning the self-test |

**If this table is empty:** N/A — see entries above; all are low-to-medium risk and none block planning.

## Open Questions

1. **Does the current CI runner class alone reproduce the daemon timeout/race failures, independent of this local reproduction's cross-package load?**
   - What we know: full reproduction (100% failure rate) required unfiltered `go test ./...` across ~51 packages on a 16-core machine; CI's `test` job isolates `internal/daemon` into its own step on a 4-vCPU runner.
   - What's unclear: whether 4 vCPUs alone (running only `internal/daemon`'s ~15 tests, or `test:race`'s 4-package `-race` combination) is contended enough to reproduce this without cross-package interference.
   - Recommendation: the planner/executor should check recent `ci.yml` run history (`gh run list` / `gh api`) for actual occurrences of these specific test names failing on `main`, to calibrate how much of this is a real production-CI problem versus a worst-case local-machine artifact worth fixing defensively.

2. **Is the getppid/registryDir race present WITHOUT `-race`, i.e., does it cause silent corruption rather than just detector noise?**
   - What we know: the race is a read of a `func()` value concurrent with a write of the same `func()` value — on amd64/arm64 in practice this is very unlikely to produce a torn read (function pointers are word-sized), so the *practical* risk is low even though it is formally undefined behavior.
   - What's unclear: whether any platform/architecture this project targets (Windows, in particular, given `watchdog_windows.go`'s different code path) has different word-tearing behavior for this exact access pattern.
   - Recommendation: treat as a correctness bug to fix regardless (Go's memory model gives no guarantee here even if amd64/arm64 are forgiving in practice), but do not treat "no crash observed without -race" as evidence it's safe.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | Both halves | ✓ | go1.26.5 [VERIFIED: `go version` this session] | — |
| `go.tool.mod` build (task/goreleaser/govulncheck) | VULN-01/02 | ✓ | Built successfully this session (task-bin, goreleaser-bin, govulncheck-bin) | — |
| `go.tool-lint.mod` build (actionlint) | VULN-01 | ✓ | Built successfully this session | — |
| Network access to `vuln.go.dev` | VULN-01/02 (govulncheck's default DB) | Not directly tested this session (existing DIST-03 gate already depends on equivalent network access via `golang/govulncheck-action`) | — | Already an established CI capability; not a new risk this phase introduces |
| `gh` CLI / GH API access to `ci.yml` run history | Open Question 1 (recommended, not blocking) | Available in this environment per tool inventory, not exercised this session | — | Skip if unavailable; treat A3 as unresolved and scope MAINT-02's fix defensively |

**Missing dependencies with no fallback:** none.
**Missing dependencies with fallback:** none blocking.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go built-in `testing` + `go.uber.org/goleak` (goroutine-leak gate, `internal/daemon/soak_test.go:21-23`) |
| Config file | None — `Taskfile.yml` defines invocation shape; no `.golangci.yml`-style test config exists for this package |
| Quick run command | `go test -race -run TestRunWatchdogCancelsRunOnSimulatedReparent -count=30 ./internal/daemon/` (isolated disproof-style check, ~33s, matches this session's exact reproduction) |
| Full suite / measured-rate command | `go test -count=1 ./...` repeated 8+ times (measured this session at 100% failure rate unfiltered); `go test -race -count=1 ./...` for the race-specific signature |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| VULN-01 | Tool-modfile binaries actually get scanned | integration (task target itself) | `task vuln` (post-replacement) | ✅ target exists, ❌ new body — Wave 0 |
| VULN-02 | Job demonstrably goes RED on a real vulnerable symbol | unit/self-test | `go test ./tools/<vulnselftest>/...` (proposed location, mirrors `tools/transcriptfreeze`) | ❌ Wave 0 — does not exist |
| VULN-03 | Blocking stance stated in workflow text | text-assertion test (mirrors `taskfile_shape_test.go`'s existing pattern) | new `Test<X>StatesBlockingStance` in `internal/upgrade/taskfile_shape_test.go` or similar | ❌ Wave 0 — does not exist |
| MAINT-01 | getppid `-race` failure fixed, demonstrated red-then-green | race detector | `go test -race -count=1 ./...` (full-suite, unfiltered — isolated runs are insufficient per Pitfall 1) | ✅ tests exist, fix does not |
| MAINT-02 | `TestRunWatchdogCancelsRunOnSimulatedReparent` (+6 siblings) pass under load | measured-rate reproduction | `for i in $(seq 1 8); do go test -count=1 ./... ; done` (100% baseline measured this session) | ✅ tests exist, fix does not |
| MAINT-03 | GoReleaser pins agree | text-assertion test | new parser in `internal/upgrade/taskfile_shape_test.go` comparing `go.tool.mod`'s goreleaser require version to `release.yml`'s `GORELEASER_VERSION` | ❌ Wave 0 — does not exist |

### Sampling Rate
- **Per task commit:** the specific isolated command for whatever was touched (e.g., `go test -race -run <name> -count=30 ./internal/daemon/` for a daemon fix)
- **Per wave merge:** `go test -count=1 ./...` unfiltered, run at least 5x (this session's evidence: single runs are not reliable evidence either way — see Constraints)
- **Phase gate:** `task test` (full serial wrapper) AND at least one `go test -count=1 ./...` unfiltered run showing zero daemon failures, since `task test`'s own isolation (`test:daemon`, `test:race -p 1`) is exactly the isolation Pitfall 2 flags as unconfirmed-sufficient

### Wave 0 Gaps
- [ ] `tools/<name>/main.go` + `main_test.go` — VULN-02's permanent self-test (mirrors `tools/transcriptfreeze`'s shape: a small `main` + a `runCLI`-style testable core)
- [ ] A new parser + test in `internal/upgrade/taskfile_shape_test.go` for VULN-03 (blocking-stance text assertion) and MAINT-03 (GoReleaser pin parity) — this file already has the exact parsing idioms needed (`parseGoModRequireVersion`, `parseWorkflowJobNames`) to build both without new infrastructure
- [ ] `inScopeJobs` entry (`internal/upgrade/taskfile_shape_test.go:107-115`) for the new VULN-01 CI job, if its `run:` body is `task vuln` and the planner wants `TestWorkflowRunBodiesInvokeTask` to actually cover it (not automatic — a new job absent from this list is simply unchecked, not a violation)
- [ ] `requiredCheckNames` (`internal/upgrade/taskfile_shape_test.go:43-51`) does NOT need the new job added for VULN-01/02/03 to function — that fixture only guards existing GitHub branch-protection required-context strings from silent rename. Making the new job an *actual* required/blocking GH check (branch protection ruleset 20157557) is a separate, out-of-repo action this research cannot verify or perform.

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | No | No user-facing auth surface in this phase |
| V3 Session Management | No | N/A |
| V4 Access Control | No | N/A |
| V5 Input Validation | Marginal | `govulncheck` CLI invocation args are static (Taskfile-defined), no untrusted input path |
| V6 Cryptography | No | N/A — the `golang.org/x/crypto/openpgp` reference (VULN-02) is a *vulnerability target for a scanner self-test*, never used for actual cryptographic operations in this repo |
| V14 Configuration (informal — supply chain) | Yes | `govulncheck -mode=binary` blocking gate; this is precisely the control category this phase implements |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Compromised/vulnerable transitive dependency in credentialed CI tooling (task/goreleaser run with repo write access + secrets per D-04) | Tampering, Elevation of Privilege | VULN-01's blocking scan is the direct mitigation this phase implements; note that `go.tool.mod`/`go.tool-lint.mod` are explicitly unmanaged by Dependabot/Renovate [VERIFIED: go.tool.mod:28-30, quoted: "Neither Dependabot nor Renovate manages this file (both gomod managers target go.mod only) — version bumps here are manual"] — this is an existing, documented gap this phase does not close (bumping stays manual; scanning becomes automatic) |
| Symbol-level false negative (a vulnerable function present but the binary's linker-retained symbol table doesn't surface it, or a vulnerability with no assigned advisory yet) | Tampering | Inherent limitation of `-mode=binary`; no mitigation beyond what govulncheck itself provides — must be stated as a known limitation, not silently absorbed into "the gate covers this now" |
| Data race on daemon test seams theoretically visible to a concurrent malicious actor | N/A | Not a security-relevant threat — `getppid`/`registryDir` are unexported package-level vars mutated only by tests, never reachable from production code paths or external input |

## Sources

### Primary (HIGH confidence — read/executed this session)
- `/Volumes/Code/github.com/seanb4t/codegraph-go/internal/daemon/*.go` and `*_test.go` — read in full
- `/Volumes/Code/github.com/seanb4t/codegraph-go/Taskfile.yml`, `go.tool.mod`, `go.tool-lint.mod`, `.github/workflows/ci.yml`, `.github/workflows/release.yml` — read in full
- `/Volumes/Code/github.com/seanb4t/codegraph-go/internal/upgrade/taskfile_shape_test.go` — read in full
- `/Volumes/Code/github.com/seanb4t/codegraph-go/tools/transcriptfreeze/main.go` — read (advisory-stance precedent)
- Live execution: `go test -race`, `go build -modfile=...`, `govulncheck -mode=binary` — all run against real artifacts this session, outputs captured verbatim above
- `golang.org/x/vuln@v1.6.0/internal/scan/errors.go` — read directly from the local module cache for the exit-code contract

### Secondary (MEDIUM confidence)
- https://pkg.go.dev/golang.org/x/vuln/cmd/govulncheck — official docs, fetched this session, cross-checked against live `-mode=binary` behavior (matched exactly)
- https://pkg.go.dev/vuln/GO-2026-5932 — official advisory, fetched this session, cross-checked against live `govulncheck -show verbose` output (matched exactly)
- https://github.com/goreleaser/goreleaser/releases/tag/v2.17.1 — release notes, web search this session

### Tertiary (LOW confidence)
- None — every claim above either cites a file+line read this session, an official doc/advisory fetched this session, or a command executed and its output captured this session.

## Metadata

**Confidence breakdown:**
- Supply-chain mechanism (VULN-01/02/03): HIGH — built and ran the actual mechanism against real repo artifacts, not simulated
- Daemon causal diagnosis (MAINT-01/02): HIGH — measured failure rate (8/8), captured actual race-detector stack traces, performed and reported a genuine disproof attempt (30x isolated runs, zero races) before the full-load reproduction confirmed the real mechanism
- GoReleaser pin reconciliation (MAINT-03): HIGH on the mechanism (both pin sites located precisely, changelog read), MEDIUM on directional recommendation (no commit-history provenance check performed — see A2)

**Research date:** 2026-08-06
**Valid until:** 30 days for the mechanism/architecture findings (stable); the specific vulnerability (`GO-2026-5932`) and version pins (goreleaser v2.17.0/v2.17.1) should be re-verified at plan time if more than ~2 weeks elapse before execution, since both are live upstream data points.
