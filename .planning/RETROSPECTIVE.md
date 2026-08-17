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

## Milestone: v0.10.0 — Agent Onboarding Skill & MCP Resources

**Shipped:** 2026-08-13
**Phases:** 4 | **Plans:** 15 | **Tasks:** 34 | **Calendar days:** 2 (2026-08-12 → 08-13)

### What Was Built
Every prior milestone made the tools better; none made an agent *use* them — a stale `instructions.go` string that deferred tool guidance to "the MCP initialize response (Phase 3)," a hand-off never built, actively misdirected a real debug session on 2026-08-08. The fix landed as three surfaces built and sequenced to reinforce each other: the MCP server now serves 10 embedded resources (`resources/list`/`resources/read`) over the wire, gated by a structural claims-drift guard proven red-then-reverted against real mutations; a decision-procedure-first SKILL.md plus a SessionStart nudge — both dogfooded straight into this repo's own `.claude/` — teach an agent to prefer `codegraph_explore` over grep, verified by three genuinely fresh live-session rehearsals rather than assertion; `codegraph install`/`uninstall`/`upgrade` distribute and refresh that package idempotently with a sidecar manifest making the installed version observable from disk; and the wire `instructions` string plus the marker block were rewritten *last*, so every capability they name is resolvable at test time (WIRE-03) — closing the milestone's own origin incident by construction, not convention.

### What Worked
- **Ordering was load-bearing, not stylistic, and it held.** Resources shipped before anything pointed at them, the claims-drift guard shipped in the same phase as the resources it gates, and the `instructions` rewrite ran genuinely last. The roadmap named this explicitly going in — a skill naming a URI the server doesn't serve is "the same class of defect as the 'Phase 3' promise this milestone exists to retire, just relocated" — and no phase had to backtrack to satisfy a dependency it should have had already.
- **Tracer-first, again.** Phase 5's 05-01 proved the whole Resources architecture end-to-end on one URI (with a full wire-oracle re-freeze) before 05-02 fanned out to the other 9 — the same pattern v1.0 and v0.5.0 both credited for containing rework.
- **Dogfooding the artifact into its own canonical location.** SKILL.md and the SessionStart hook were authored directly into this repo's `.claude/`, not a staging copy — Phase 7's `go:embed` then pointed at that same source, so there was never a second copy to drift.
- **An independent review outside the execution loop caught a real vulnerability before merge.** A background security review found a matcher+shape hook-ownership heuristic that could silently overwrite an unrelated user's hook sharing a matcher name, confirmed it RED against the vulnerable code, and it was reverted same-day (commit `242ec0a`) — the fix-composition-Critical pattern v1.0's retrospective already named recurring here in a new shape.
- **Verification that goes looking for gaps reports them.** The Phase 6/7 live-session rehearsals found and filed two real gaps rather than a clean-pass fiction, and Phase 7's UAT was pushed back into live CLI execution (real binaries, real scratch-`$HOME` runs) after the user objected to being asked to do mechanical verification a shell could do directly.

### What Was Inefficient
- **A three-session live-debug cluster chased two false regressions in one day, from one root instrument.** The Phase 6 rehearsal reported a SessionStart `resume` matcher failure and a skill-catalog-listing absence; a follow-up session then reported the skill missing *entirely*. All three were false negatives from variations of the same invalid oracle — grepping a session transcript for evidence a mechanism it was blind to. The genuinely broken thing, when one was finally found, was small (a 299→110-byte description trim); most of the cost was diagnosing that nothing was broken, three times, with an evidence artifact each time that had already hardened its own false conclusion into "not a rehearsal methodology error."
- **A hardened false finding actively resists correction.** The retraction that finally worked used an oracle *outside* the suspect chain entirely — a separate, fresh `claude` process asked with no tools what its own context showed — rather than a second look through the same transcript-grep method that produced the original claim.

### Patterns Established
- **Two disjoint instruments, not one.** Before recording "X did not happen" from a single observation method, either prove that method can detect X when X *is* present, or confirm with an oracle that shares no machinery with the first. Written into this repo's own `codegraph` skill and its `SKILL-03` rehearsal as a rule, not just a lesson, after the third recurrence in one day.
- **Ownership of a shared-array entry (JSON hooks, config blocks) must be exact-identity, never shape/position.** A "recovery" heuristic that reclaims ownership by matcher name + shape cannot distinguish a hand-edited own-entry from an unrelated entry occupying the same slot; guessing wrong safely (a duplicate) is strictly better than guessing wrong unsafely (an overwrite).
- **Point, don't restate.** SKILL.md names Phase 5's resource URIs instead of embedding their content — the same discipline that, if it had existed sooner, is what this whole milestone was created to retrofit onto `instructions.go`.
- **A measured budget beats a documented one.** The skill-catalog investigation replaced a false "~45,000 chars / exactly 173 descriptions" invariant, recorded three times as fact, with the actual measured relationship (`skillListingBudgetFraction` × context window) — and relabeled it explicitly as a measurement, not a constant, so the next session cannot re-harden it.

### Key Lessons
1. **Absence of evidence in a transcript is a claim about the transcript, not the world.** A runtime that suppresses redundant output (SessionStart dedup) or renders past a cap (the 45KB skill-catalog budget) suppresses the evidence of its own behavior along with the behavior itself — the transcript looks identical to "never happened."
2. **An evidence artifact that asserts its own methodology is sound is the highest-value claim to attack first, not accept.** The `resume`-matcher finding's own write-up said "not a rehearsal methodology error" in bold; that sentence, not the symptom, was the thing that turned out to be wrong.
3. **A symptom stronger than a known mechanism predicts is usually the observation, not a new mechanism.** The third false alarm ("skill absent entirely," stronger than either prior finding's "degraded") reasoned the opposite in its own write-up and was wrong — it was a stale resumed-conversation replay layered under a one-entry delta block, not a regression of either prior fix.
4. **Sequencing content before the pointer to it is the structural fix for a stale-promise bug class**, not a review checklist item — WIRE-03's "every named capability is resolvable at test time" was only buildable because Phases 5-7 had already shipped everything Phase 8 needed to name.

### Cost Observations
- Model mix: consistent with prior milestones — opus for orchestration/planning/security, sonnet for execution.
- Notable: the fastest milestone by calendar time (2 days, 15 plans) despite three separate live-debug investigations in the middle of it — the debug cost was concentrated in re-diagnosing the *same* oracle failure, not in spreading across genuinely different bugs.

## Milestone: v0.11.0 — Standalone Project Identity

**Shipped:** 2026-08-16
**Phases:** 6 | **Plans:** 30 | **Tasks:** 60 | **Calendar days:** 4 (2026-08-13 → 08-16) | **Commits:** 261

### What Was Built
Everything the project *did* already stood on its own; the way it *described and tested itself* did not. The origin is now acknowledged exactly once — legally, in the past tense — in `NOTICE`, plus one clause in README's `## License`, with `LICENSE` left verbatim MIT because an appended attribution paragraph downgrades GitHub's detection to `NOASSERTION`. Comparison vocabulary is gone from every doc, issue/PR template, workflow, script, comment, identifier and fixture, resolved term-by-term with a recorded verdict per occurrence and closed by a positive-controlled census reporting `TOTAL=0` across 285 Go files. The load-bearing work was never the prose: the golden suite derived its oracle from corpora chosen because the origin project used them, so re-basing it meant selecting corpora **by recorded measurement** — per-kind edge counts and per-language file counts from real indexing runs — fetching them at pinned SHAs rather than vendoring, re-freezing every golden from codegraph-go's own output, and then re-proving the result could still fail. `docs/BENCHMARKS.md` publishes absolute throughput / query-latency / peak-RSS figures mechanically generated from a committed raw measurement, no figure typed by hand. `codegraph migrate`, `internal/migrate` and the sole-use `modernc.org/sqlite` dependency were removed outright (D-04) — a supply-chain reduction. Net effect outside `.planning/`: **−806 lines**.

### What Worked
- **Making corpus selection a blocking measurement spike, not a judgment call.** The trap was already documented in this repo's own history: v1.0 Phase 1 found codegraph-go's own idiomatic Go source legitimately emits **zero** `overrides` and `type_of` edges, so a corpus set can under-cover the 9-kind `RANK_EDGES` vocabulary while every test stays green — a silently narrowed oracle, exactly what a re-baseline invites. Phase 1 locked the set by recorded counts, and recorded a *rejected* candidate (apache/arrow) alongside the accepted ones.
- **Separating the rename pass from the re-freeze pass into distinct reviewed diffs.** Doing both at once makes any resulting regression un-attributable — was it the rename, or the new corpus? The identifier pass changed no golden byte; the re-freeze pass changed no identifier.
- **Non-vacuity proof as its own phase, applied to the suite in its final form.** A re-baseline that also authors its own proof is the same author certifying their own oracle in the same breath. All five assertion families were red-demonstrated, byte-cleanly reverted, and recorded with pasted failing output. Nothing landed.
- **Positive controls on every zero.** After v0.5.0's four false-absence findings, this milestone treated an unverified zero as a broken instrument by default — the CODE-01 census proved it could return 2 against a planted fixture before its `TOTAL=0` was believed.
- **Census gates that assert what must SURVIVE, not only what must be absent.** T-06-04 prints three counts and asserts `PINNED + LOCAL == TOTAL` (19 + 2 = 21, re-derived at audit) — it cannot pass by finding nothing. T-06-08 asserts legitimate uses survive against a floor of 5 and measured 61, so a green census cannot be bought with find-and-replace. The repo now carries 49 non-vacuity guards, 19 in `taskfile_shape_test.go` alone; rule `84d1gfpywd` is institutionalized rather than aspirational.
- **Recording two requirement halves honestly rather than closing them with a stand-in.** BENCH-03 stayed `pending-no-ci-run` rather than being closed by a local measurement (later discharged by live run `31973967130`), and MEM-02's store half is labelled `accepted-by-d15-evidence-standard` — acceptance is a weaker claim than observation and says so.

### What Was Inefficient
- **CODE-01 needed three census passes because the earlier ones were line-based and the phrase wrapped across a comment line break.** Nine Go files were missed twice by instruments that were structurally incapable of seeing them; the fix was a multiline `rg -U` census, plus auditing the full 30-file `internal/agents` population instead of the prior 10-name sample. The same class as v0.5.0's "query that could not find," one layer down: not a wrong pattern, a wrong *unit of matching*.
- **Two ROADMAP defects went undetected for a day and surfaced only at milestone-close pre-flight.** A GSD plan-write (`4d3e4d7`) had replaced the `### Phase 3:` detail heading with a Phase 2 plan bullet, so `init.manager` reported a five-phase milestone with Phase 3 absorbed into Phase 2 — while `roadmap validate` returned `{"warnings":[]}`. Separately, Phase 3 has no UAT file, so `phase uat-passed 3` returned `passed:false` with **empty blockers**, the transition never ran, and two plan checkboxes plus FIXT-03/FIXT-07 stayed unflipped. Both were silent by construction.
- **The security and Nyquist-validation sweep ran retroactively across all six phases at the end.** ~4,000 tests, 131 threats and six VALIDATION reconciliations in one pass, one of which (Phase 5) had to be reconstructed from scratch because no VALIDATION artifact existed at all. Per-phase, at phase close, is strictly cheaper — the same planning-artifact-debt pattern v0.5.0's retrospective already named.
- **`supersede_memory` was called without ever reading its schema.** Its `supersedes` parameter is a string; passing an array returned `not found: ["id"]` — an error echoing the argument back — which was misread as a missing target, and a whole ownership-write-gate theory was built on it before the maintainer corrected it in one line.

### Patterns Established
- **Score by counting `--- PASS` lines, never by exit status.** `go test -run PATTERN` exits 0 when the pattern matches nothing. Three plan-declared verification commands matched zero tests while reading green, one of them the named guard for a CRITICAL threat.
- **Use `roadmap get-phase N` per phase — never `validate` — to prove a roadmap is readable.** `validate` does not check heading↔summary correspondence; `get-phase 3` returned `malformed_roadmap` and named the exact required fix in its own message.
- **Match the union, not one form.** The `<threat_model>` XML detector missed the `## Threat Model` markdown form entirely, under-reporting Phase 5 as 2/8 modelled when it is 8/8. Plans use both shapes.
- **A mitigation can be legitimately undone by a later ruling; what must not happen is the undoing going unrecorded.** Phase 4's threat T-04-02 protected a README `migrate` row that D-04 then deleted in Phase 5.
- **Restoring a tool's own required shape is filling in a value, not inventing structure.** The deleted Phase 3 heading was repaired verbatim; the repair ran *before* auditing or archiving, because a missing heading truncates every scope-sensitive reader downstream of it.

### Key Lessons
1. **Validator-green is not evidence.** `roadmap validate` returned no warnings on a roadmap that had silently lost a phase. Verify the property you actually care about — here, `getMilestonePhaseFilter` returning all six — not the validator's opinion of the file.
2. **A guard's *reference* can be vacuous even when the guard is not.** Rule `84d1gfpywd` says a guard must positively assert it did its work; this milestone found the same failure one level up, in the map pointing at the guard. The behavior was covered; the command naming it matched nothing.
3. **An error that quotes your own argument back is reporting your argument, not the store.** A coherent story explaining every observation can still be wholly wrong when it was built on an unread schema.
4. **Resuming a halted plan means re-reading its acceptance gate, not just its intent.** When an executor halted on an unmet precondition and the orchestrator finished the work itself, the plan's artifact contract was silently skipped — prose was written where machine-readable tables were required, and the set-equality extractor would have found an empty approved set. Verification caught it.
5. **A live external fact has a shelf life; a frozen transcription does not record that.** A VERIFICATION.md asserted "branch unpushed, no CI run exists" in past-tense prose; the fact reversed, the closure landed in a different file, nothing cross-checked the two, and the report stayed confidently wrong for two commits.

### Cost Observations
- Model mix: consistent with prior milestones — opus for orchestration, planning, security and audit; sonnet for execution.
- Notable: the milestone audit's integration-checker resolved to haiku and **reached the right conclusions from partly wrong evidence** — citing PLAN documents as proof of deletion (a plan states intent, not outcome) and asserting a count was "derived from authoritative sources" when `expectedGoCaptures` is a hand-written Go slice. The verdict held; the evidence did not. Verdict and evidence are separately trustworthy, and anything load-bearing has to be re-derived.
- Notable: 261 commits in 4 days, and a **net −806 lines outside `.planning/`** — the only milestone so far that shrank the product.

## Cross-Milestone Trends

| Metric | v0.1 | v1.0 | v0.5.0 | v0.10.0 | v0.11.0 |
|--------|------|------|--------|---------|---------|
| Phases | 8 | 10 | 4 | 4 | 6 |
| Plans | 66 | 72 | 24 | 15 | 30 |
| Tasks | 142 | — | — | 34 | 60 |
| Calendar days | — | 20 (2026-07-14 → 08-03) | 3 (2026-08-08 → 08-11) | 2 (2026-08-12 → 08-13) | 4 (2026-08-13 → 08-16) |
| Commits | — | 594 | ~80 | — | 261 |
| Net product LOC (outside `.planning/`) | — | — | — | — | **−806** (first shrinking milestone) |
| Releases cut during the milestone | 1 | 1 | 6 (`v0.5.0` … `v0.9.0`) | 0 (no milestone tag by design — D-06R) | 0 (same) |
| Deep-review bugs caught (green suite missed) | Phases 4/6/7/8 — recurring, high-value | Every phase touching I/O, concurrency, or CI — 10/10 recurrence | 29 findings across 6 convergence cycles on Phase 1 alone; cycle 3 caught a test pinning a *broken* invariant | Phase 7: 2 CRITICAL + 4 WARNING from a deep review after a green TDD suite; a separate independent security review caught 1 real vulnerability pre-merge | Retroactive secure+validate sweep across all 6 phases: 131 threats modelled, 2 CRITICAL; 3 phantom `-run` patterns found reading green |
| Release candidates to first clean release | 3 (rc.1 go.sum, rc.2 SLSA, rc.3 ✓) | n/a — release-please cut `v0.2.0` clean on the first live run | n/a — but the first `post-release-verify` on v0.9.0 failed closed on a missing installer | n/a — no release gate in scope | n/a — no release gate in scope |
| Gates found unable to fire | — | 3 (cross-platform baseline, inverted `rg -qv`, 51.5%-stale baseline) | 2 (a test pinning the broken SBOM template; `dry-run-signed` additions-only diff guard, still open) | — | 3 (`go test -run` patterns matching zero tests, one guarding a CRITICAL threat) |
| Self-inflicted false signals (query ≠ reality) | — | — | 4 in one session | 3 in one day, all one oracle class (transcript grep blind to suppressed/degraded/stale rendering) | 3 new mechanisms: line-based census vs. a line-wrapped phrase; `rg` exiting non-zero for *both* no-match and file-not-found; a quoted file list collapsing into one argument |
| Non-vacuity guards in tree | — | — | — | — | 49 (19 in `taskfile_shape_test.go` alone) |

*Trends to watch: (1) deep review on I/O/crypto/CI phases catches real defects every time — keep it a standing gate; v1.0 made it 10-for-10. (2) the first live release run finds what darwin-only + green CI cannot. (3) version/label claims need a human product check. (4) **New in v1.0 — the dominant defect class is no longer "the gate failed" but "the gate could not fail."** Three instances, each looking healthy while measuring nothing. Every new gate now has to be demonstrated RED against a confirmed-applied mutation before it is trusted. (5) **New — subagent judgment needs a deterministic gate downstream of it.** Two haiku agents reported success while skipping or misreading their own checks; both were caught only by the mechanical check that ran next. (6) **New in v0.5.0 — the "gate could not fail" class has a sibling: the *query* that could not find.** Four instances in one session, each reporting absence where the search, not the world, was wrong. (7) **New — planning-artifact debt compounds silently.** Phase 1 shipped before three capabilities existed and cost a full retroactive pass at close; the artifacts came out accurate but strictly more expensive than plan-time authoring. (8) **New — verified ≠ landed.** A milestone close was attempted with a quarter of the milestone unmerged, because the readiness gate reads planning artifacts and never inspects git. (9) **New in v0.10.0 — the "query that could not find" class generalizes beyond `rg`/`go test` to LLM-transcript reading itself.** Three false negatives in one day shared a single root cause — grepping a session transcript for evidence of a mechanism the transcript is structurally blind to (suppressed dedup output, a degraded catalog entry, a stale resumed-conversation replay) — each hardened into a confident write-up before a second, disjoint instrument retracted it. (10) **New in v0.11.0 — the "gate could not fail" class now has a third layer: the gate is sound, but the *reference to it* names nothing.** `go test -run PATTERN` exits 0 on zero matches, so three plan-declared verification commands read green while executing no tests — one of them the named guard for a CRITICAL threat. The standing correction is mechanical: score by counting `--- PASS` lines, never by exit status. (11) **New — a tool's own validator is not a check of the property you care about.** `roadmap validate` returned `{"warnings":[]}` on a roadmap that had silently lost a phase heading, while the tool's `get-phase N` named the exact fix; before blaming a tool, confirm whether our own edits (or a tool write) introduced the triggering shape. (12) **New — retroactive artifact sweeps are the recurring tax.** v0.5.0 paid it on planning artifacts; v0.11.0 paid it again on security and Nyquist validation, reconciling six phases and ~4,000 tests at close, with one phase's VALIDATION reconstructed from nothing. Both times the output was accurate and strictly more expensive than doing it at phase close.*
