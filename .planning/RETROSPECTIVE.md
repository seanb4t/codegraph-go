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

## Milestone: v0.5.0 — macOS Distribution & Homebrew

**Shipped:** 2026-08-11 (released as `v0.9.0` 2026-08-12)
**Phases:** 4 | **Plans:** 24 | **Commits:** ~80 over 3 days | **Releases cut:** 6 (`v0.5.0` … `v0.9.0`)

### What Was Built
codegraph became *installable by convention* on macOS. The release pipeline moved from a per-platform `goreleaser build --single-target` matrix onto a single `goreleaser release` invocation — the only OSS path, since `release` refuses a `dist/` built elsewhere and both escape hatches are GoReleaser Pro — with both linux legs `zig cc`-cross-compiled from one macOS runner. GoReleaser now owns archives, checksums, signing and SBOMs declaratively, replacing hand-rolled shell loops. Apple Developer ID signing plus Quill-backed notarization moved `spctl` from `rejected` to `accepted` on both darwin arches. A `seanb4t/homebrew-tap` repository and `homebrew_casks:` block make `brew tap && brew install codegraph` work cold, with man pages and three-shell completions. `codegraph upgrade` detects a Homebrew-managed install structurally and steps aside rather than mutating a Caskroom the package manager owns.

### What Worked
- **Spiking the one unproven claim first, blocking, with a FAIL-bar written in advance.** The zig-cross question could have sunk the milestone at any point; instead it was answered on day one and passed on variation V1 at first dispatch. The V1–V5 variation list was written into the canary header *before* the first run specifically so that exhausting it would declare failure falsifiably rather than by argument. The costed GoReleaser Pro fallback was named up front and never needed.
- **Cross-AI plan convergence on the highest-risk phase.** Phase 1 ran six review cycles (19 → 0 HIGH findings). Cycle 3 caught a *silent regression* — a test that pinned the broken SBOM template demonstrated-RED, which would have actively resisted correction — and it was found only because cycle 2's oracle was itself blind (it asserted filesystem basenames, identical under broken and correct configs).
- **Asserting relationships instead of counts.** `verify:release-assets` classifies published assets by set equality both directions rather than against a fixed total; when Phase 2 later added two jobs to `post-release-verify`, the 1:1 guard invariant survived where `== 3` would have gone red on a correct change.
- **Removing code paths instead of guarding them.** Phase 4's strongest mitigations are deletions: `Options.Force` is never read, `EvalSymlinks` errors fail closed with no fallback, and the self-authored install sentinel was replaced by Homebrew's own receipt.

### What Was Inefficient
- **Three phases shipped before their retroactive artifacts existed.** Phase 1 predated the security, Nyquist and API-coverage capabilities, so `01-SECURITY.md`, `01-UAT.md`, `COVERAGE.md` and a reconciled `01-VALIDATION.md` were all produced *after the fact* at milestone-close time. They came out accurate, but writing a coverage declaration for code that shipped four phases ago is strictly more expensive than writing it at plan time.
- **The same defect class recurred three times**: a shared target gains a dependency and not every caller is swept. `binary_signs:` broke the pre-existing darwin canary; wave 1 pre-fixed the *new* canary and missed the *old* one; plan 04-05 fixed the `verify:self-upgrade` target and missed the workflow job that invokes it — surfacing only when v0.9.0's verification failed. Each fix was local; only the third produced a derived-property test.
- **A milestone close was attempted while a quarter of the milestone sat unmerged.** `ALL_PHASES_VERIFIED` read `true` because it reads planning artifacts and never inspects git; `internal/upgrade/brew.go` did not exist on `main`. Caught at pre-flight, but only by an explicit check that is not part of the gate.
- **`[ci skip]` on a ship-note commit deadlocked a PR.** The ship workflow both requires pushing the note onto the PR branch and appends `[ci skip]`; on a repo with required status checks those instructions cannot both hold, and the PR blocked on checks that would never report.

### Patterns Established
- **Assert the relationship, not the cardinal.** Set equality over fixed totals; 1:1 invariants over counts; derived job lists over hardcoded ones.
- **A query that does not match reality reports absence, not truth.** Four self-inflicted false signals in one session — `go test -run` printing `ok` on zero matches, `rg -c '\b17\b'` matching version strings, `task` echoing unexpanded `${VAR}` so an evidence-line count doubles, and a receipt glob searching the wrong tree level. Every one produced plausible output meaning something other than it appeared to.
- **Distinguish debt that needs *work* from debt that needs *time*.** Three items closed by v0.9.0 simply existing. Recording them as "closing conditions" rather than tasks kept them out of a closure phase.
- **Refuse the short-circuit and write the refusal down.** ASVS-L1 permits skipping the security auditor on a clean preliminary pass; it was refused for the fourth consecutive phase, and the refusal is recorded *inside* each SECURITY.md so a reader cannot mistake inherited depth for verified depth. Same discipline applied to carrying integration verdicts forward in the milestone audit.

### Key Lessons
1. **Following upstream documentation faithfully was the bug.** GoReleaser's own documented `signature: "${artifact}.sigstore.json"` idiom collapses all four platforms onto one filename for `formats: [binary]`, because `${artifact}` expands from Path but publishes under Name. Correct-looking, upstream-blessed, and silently destructive under `replace_existing_artifacts: true`.
2. **A verification gate's trigger is a design decision with a long tail.** Choosing `workflow_run` + a validated `tag` input over `release: [published]` — because the latter fires before assets upload — is what let a one-step CI fix re-prove an already-published release months later without cutting a new version.
3. **Repo inspection cannot settle account-side infrastructure state.** "The repo never references an arm64 runner" was recorded as "no arm64 runner exists"; three were already provisioned. Bound confidence by what the evidence can actually reach.
4. **Zero findings is not the same as nothing to find.** Phase 1's security audit found no unregistered threats — and none of its six SUMMARYs carries a `## Threat Flags` section, so there was nothing to reconcile. Absence of reporting reads identically to absence of problems.

### Cost Observations
- Model mix: opus for planning/orchestration, sonnet for execution, haiku for checkers — with the standing caveat that haiku checker output required independent re-verification twice (a prior run claimed files absent that were present).
- Sessions: milestone executed over ~4 days; the close itself (Phase 1 retro-artifacts + audit + release + a quick-task fix) ran in a single long session.
- Notable: the most expensive single defect was not a code bug but a *coverage* one — Phase 1's missing artifacts cost a full retroactive pass at close time, which plan-time authoring would have avoided entirely.

**Process gap noted:** v0.3.0 has no entry in this file. Its close skipped the retrospective step; the milestone record exists only in `MILESTONES.md` and `milestones/v0.3.0-*`.

## Cross-Milestone Trends

| Metric | v0.1 | v1.0 | v0.5.0 |
|--------|------|------|--------|
| Phases | 8 | 10 | 4 |
| Plans | 66 | 72 | 24 |
| Tasks | 142 | — | — |
| Calendar days | — | 20 (2026-07-14 → 08-03) | 3 (2026-08-08 → 08-11) |
| Commits | — | 594 | ~80 |
| Releases cut during the milestone | 1 | 1 | 6 (`v0.5.0` … `v0.9.0`) |
| Deep-review bugs caught (green suite missed) | Phases 4/6/7/8 — recurring, high-value | Every phase touching I/O, concurrency, or CI — 10/10 recurrence | 29 findings across 6 convergence cycles on Phase 1 alone; cycle 3 caught a test pinning a *broken* invariant |
| Release candidates to first clean release | 3 (rc.1 go.sum, rc.2 SLSA, rc.3 ✓) | n/a — release-please cut `v0.2.0` clean on the first live run | n/a — but the first `post-release-verify` on v0.9.0 failed closed on a missing installer |
| Gates found unable to fire | — | 3 (cross-platform baseline, inverted `rg -qv`, 51.5%-stale baseline) | 2 (a test pinning the broken SBOM template; `dry-run-signed` additions-only diff guard, still open) |
| Self-inflicted false signals (query ≠ reality) | — | — | 4 in one session |

*Trends to watch: (1) deep review on I/O/crypto/CI phases catches real defects every time — keep it a standing gate; v1.0 made it 10-for-10. (2) the first live release run finds what darwin-only + green CI cannot. (3) version/label claims need a human product check. (4) **New in v1.0 — the dominant defect class is no longer "the gate failed" but "the gate could not fail."** Three instances, each looking healthy while measuring nothing. Every new gate now has to be demonstrated RED against a confirmed-applied mutation before it is trusted. (5) **New — subagent judgment needs a deterministic gate downstream of it.** Two haiku agents reported success while skipping or misreading their own checks; both were caught only by the mechanical check that ran next. (6) **New in v0.5.0 — the "gate could not fail" class has a sibling: the *query* that could not find.** Four instances in one session, each reporting absence where the search, not the world, was wrong. (7) **New — planning-artifact debt compounds silently.** Phase 1 shipped before three capabilities existed and cost a full retroactive pass at close; the artifacts came out accurate but strictly more expensive than plan-time authoring. (8) **New — verified ≠ landed.** A milestone close was attempted with a quarter of the milestone unmerged, because the readiness gate reads planning artifacts and never inspects git.*
