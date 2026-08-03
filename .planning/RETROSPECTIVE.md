# CodeGraph Go — Retrospective

Living retrospective across milestones. Newest milestone first.

## Milestone: v1.0 — Drop-in Parity & Human UX

**Shipped:** 2026-08-03
**Phases:** 10 | **Plans:** 72 | **Commits:** 594 over 20 days

### What Was Built
Drop-in behavioral parity with TS CodeGraph v1.3.x, plus the human-facing surface v0.1 lacked. TS-faithful `explore` relevance (deterministic Random-Walk-with-Restart, the full H1–H21 heuristic stack) and `node` multi-definition disambiguation in the shared engine, so CLI and MCP are byte-identical. Sectioned `status` with borrowed-index/worktree detection. Watcher-on-MCP by default with a WSL2 auto-off policy. Pebble WAL noise silenced and MCP stdout held to clean JSON-RPC, both build-enforced. Marker-fenced git sync hooks. A Charm TUI — lipgloss `status`/`files`, a bubbletea daemon picker, install multi-select — behind a fail-closed ANSI-isolation archtest so the agent path can never see styling. Systematic flag parity with a self-verifying drift guard. Release cutting fully automated via release-please + GoReleaser (`v0.2.0` shipped signed, SBOM'd, SLSA-attested, no human `git tag`). A `Taskfile.yml` that is the single definition of every CI job body, enforced as an invariant.

### What Worked
- **Deep review as a standing gate, again.** Every phase that touched I/O, concurrency, or CI produced Criticals the green TDD suite had missed — Phase 1 (RWR corruption via overload-collapse; NODE-03 unwired), Phase 3 (pebble lock contention made default-path by the flip), Phase 5 (a *fix-composition* Critical where round-1's guard plus Install's fallback deleted user content). The v0.1 trend held for all ten phases.
- **Mutation proofs over assertions.** The repo converged on "demonstrate the gate RED against a confirmed-applied mutation, then revert" as the acceptance bar. It repeatedly caught guards that would have passed vacuously.
- **Front-loading the load-bearing algorithms.** Phase 1 was 17 plans of shared-engine behavior; the TUI landed last behind a seam. Nothing downstream had to be reworked when styling arrived.
- **Refusing to close on a plausible story.** Phase 9 held REL-02 open rather than accept 4-of-5 green when the upgrade path 404'd — correctly, since it was unprovable-not-unmet, and passed first try once the repo went public.

### What Was Inefficient
- **Gates that could not fire.** Three separate instances: a retracted 10.6% "regression" from a cross-platform baseline comparison, an inverted `rg -qv` gate, and a 51.5%-stale perf baseline that left the gate needing a ~38% drop to trip. All three looked healthy. The cost was three rounds of triage and two superseded conclusions before anyone questioned the yardstick.
- **Phase 10-04 refuted its own plan's premise** after five investigation rounds — the Namespace migration it was written to perform was abandoned, and tmpfs (tried as the fix) made both runners worse. Valuable, but a control measurement up front would have cost one round instead of five.
- **LLM checkers reporting opinions as measurements.** A haiku plan-checker returned "VERIFICATION PASSED" with an explicit per-decision ✓ table; the deterministic gate then failed 12/16 on citation grounds. A haiku nyquist-auditor reported "GAPS FILLED" while silently skipping the mutation proof it was told to run. Both were caught only because a deterministic check ran afterward.

### Patterns Established
- **Frame descriptors are part of a measurement.** `CheckRegression` now refuses GOOS/GOARCH, runner, *and* scratch_fs mismatch as category errors before any numeric comparison — unrecorded identity is not a wildcard.
- **Single-definition CI.** Job bodies live in `Taskfile.yml`; `TestWorkflowRunBodiesInvokeTask` enforces it and was proven RED at the base commit.
- **Canaries for tag-only workflows.** `release.yml` only runs during a real release, and its bad outcomes are quiet, so a permanent path-scoped canary machine-proves the macOS runner class continuously.
- **release-please is the sole tag authority.** No hand-created tags, including milestone markers — the v1.0 close deliberately created none, unlike v0.1 (which predates release-please). Corollary if ever revisited: a planning tag must not match `release.yml`'s `v[0-9]*` trigger, or it fires the whole release pipeline.
- **An accept whose stated reason is false is not closed.** Phase 10's security audit re-grounded T-10-02-02 on the clauses that actually verified rather than letting a falsified justification stand.

### Key Lessons
- A tight, reproducible measurement proves a signal is *systematic*, not that the accused caused it. Stability does not discriminate between competing explanations — "the code got slower" and "the yardstick came from a different machine" predict the identical tight cluster. Confirm the measurement can detect the cause before bisecting for it.
- A review finding can inherit a stale fixture as its evidence. Phase 10's WR-04 claimed CONTRIBUTING's required-checks list had drifted; the live API showed CONTRIBUTING was set-equal and the *test fixture* was the stale artifact.
- A subagent's tick table is an opinion; a deterministic gate is a measurement. Never treat the former as evidence the latter will pass.
- Verifiers can *understate* coverage too — one claimed a runner label had never been scheduled when a canary had three green runs, including one at HEAD. Check the observation, not just the artifact.
- Enforcing a rule on subagents does not immunize the orchestrator: `echo $?` after a pipe captures the tail's status, a trap the orchestrator fell into immediately after instructing three executors to avoid it.

### Cost Observations
- Model mix: opus for orchestration, planning, security audit and verification; haiku for mechanical checkers — the two haiku failures above argue for keeping deterministic gates downstream of any haiku judgment.
- Notable: Phase 1 (17 plans) and Phase 10 (7 plans, 5 waves, a 5-round investigation) dominated. Phase 10 was nominally "add a Taskfile" and became the milestone's richest source of gate-integrity findings.

## Milestone: v0.1 — Initial Release

**Shipped:** 2026-07-14
**Phases:** 8 | **Plans:** 66 | **Tasks:** 142

### What Was Built
A working Go rewrite of CodeGraph's core capabilities in a single static binary: schema-versioned Pebble `GraphStore` behind a concurrency-tested interface (CGo tree-sitter parser); deterministic two-pass Go indexer; read-only query engine + stdio MCP server (output shapes verified against the TS v1.3.1 golden corpus); incremental sync + fsnotify watcher + single-writer daemon; 14-language coverage with framework-aware routing; agent install/uninstall for 8 agents + keyless verify-before-swap `upgrade`; a resumable, fail-loud TS→Pebble migration tool; and a signed/attested/SBOM'd/reproducible release with published head-to-head benchmarks (Go beats TS 1.3.1 on every measured metric, 6.1×–59.7× throughput). First real release `v0.0.0-rc.3` verified end-to-end.

### What Worked
- **Deep code review as a standing gate on I/O/crypto/CI phases.** In Phases 4, 6, 7, and 8, a dedicated review found real bugs a fully-green TDD/`-race`/goleak suite missed (concurrency/prune races, swallowed install I/O, data-loss, a GitHub-Actions command-injection). Green tests were necessary, never sufficient.
- **Capturing TS ground truth up front** (Phase 1 golden corpus + schema DDL, while the live TS tool was installed) let later phases measure output-shape parity against fact and enabled the real head-to-head benchmark at the end.
- **Narrow seams paid off:** the `parser.Parser` interface (CGo now, wazero later), the additive-only protobuf schema (annotation ranges reserved), and the `x/` file-owned index (planted in Phase 1, load-bearing in Phase 4).
- **Honest "human_needed" status.** Holding DIST-02 as not-Complete until a real tagged release ran was correct: the first live runs caught two bugs nothing local could.

### What Was Inefficient
- **Darwin-only development hid cross-platform gaps.** The first release (`rc.1`) failed on a missing **linux-only** `go.sum` hash (`prometheus/procfs`, build-tag-gated) that no local build or green CI on darwin ever exercised. Cost a full release round-trip.
- **The private-repo/public-log interaction wasn't anticipated.** `rc.2` published signed binaries but SLSA halted (private repo needs `private-repository: true`); separately, keyless cosign already exposes the repo name to the public transparency log — worth deciding on *before* the first signed release.
- **Over-claimed the milestone label.** It was initially closed as "v1.0 — Parity Release" before the human caught that the **CLI surface diverges from TS CodeGraph** — so it is not a drop-in parity replacement and shouldn't carry a 1.0 or "parity" label. Re-versioned to **0.1**. Lesson: "parity" is a claim to *earn against the actual TS command surface*, not to infer from "all planned phases done."

### Patterns Established
- **Version honestly.** Reserve 1.0 / "parity" for a validated drop-in swap against the real TS CLI surface; ship functional-but-incomplete work as 0.x.
- **Before tagging any release: `GOOS=<each> GOARCH=<each> go list -mod=readonly ./...`** across all 6 targets to catch platform-specific `go.sum`/module-graph gaps that darwin-only dev misses.
- **Benchmarks: report median-of-N-runs, store every raw run, label it a median.** Absolute magnitudes are hardware-specific; cite the Go-vs-TS *ratios* as the durable signal.
- **OS-level peak RSS** (`getrusage`/`ru_maxrss` on the child), never in-process, for fair cross-runtime comparison — with explicit KB(Linux)/bytes(macOS) normalization.
- **Fail the build gate before any signing/publish** — `rc.1`/`rc.2` failures produced no release and no Sigstore entries, exactly as designed.

### Key Lessons
1. **The first *real* release run is a distinct test surface** that green CI cannot substitute for — cross-platform `go.sum`, live OIDC signing, private-repo policy, provenance. Budget for at least one failed release candidate.
2. **CGo cross-compile fears were mostly misplaced:** darwin built natively fine (no DNS issue) and windows cross-compiled via zig; the actual failures were mundane (go.sum, a config flag).
3. **Verify artifacts, don't trust "CI green":** downloading the binary, running it, and `cosign verify-blob` against the shipped verifier's own identity (with a negative control) is the real acceptance test.
4. **Let the human sanity-check the milestone framing.** The version number and the "parity" claim are product judgments, not mechanical outputs of finishing the roadmap.

### Cost Observations
- Model mix: Opus (orchestrator + planner), Sonnet (researcher/executor/reviewer/verifier), Haiku (plan-checker).
- Notable: the `--auto` discuss→plan→execute chain drove all 9 Phase-8 plans sequentially (worktrees auto-degraded on unresolved `origin/HEAD`, #683 — recurring).

---

## Cross-Milestone Trends

| Metric | v0.1 | v1.0 |
|--------|------|------|
| Phases | 8 | 10 |
| Plans | 66 | 72 |
| Tasks | 142 | — |
| Calendar days | — | 20 (2026-07-14 → 08-03) |
| Commits | — | 594 |
| Deep-review bugs caught (green suite missed) | Phases 4/6/7/8 — recurring, high-value | Every phase touching I/O, concurrency, or CI — 10/10 recurrence |
| Release candidates to first clean release | 3 (rc.1 go.sum, rc.2 SLSA, rc.3 ✓) | n/a — release-please cut `v0.2.0` clean on the first live run |
| Gates found unable to fire | — | 3 (cross-platform baseline, inverted `rg -qv`, 51.5%-stale baseline) |

*Trends to watch: (1) deep review on I/O/crypto/CI phases catches real defects every time — keep it a standing gate; v1.0 made it 10-for-10. (2) the first live release run finds what darwin-only + green CI cannot. (3) version/label claims need a human product check. (4) **New in v1.0 — the dominant defect class is no longer "the gate failed" but "the gate could not fail."** Three instances, each looking healthy while measuring nothing. Every new gate now has to be demonstrated RED against a confirmed-applied mutation before it is trusted. (5) **New — subagent judgment needs a deterministic gate downstream of it.** Two haiku agents reported success while skipping or misreading their own checks; both were caught only by the mechanical check that ran next.*
