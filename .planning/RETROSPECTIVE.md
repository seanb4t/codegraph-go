# CodeGraph Go — Retrospective

Living retrospective across milestones. Newest milestone first.

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

| Metric | v0.1 |
|--------|------|
| Phases | 8 |
| Plans | 66 |
| Tasks | 142 |
| Deep-review bugs caught (green suite missed) | Phases 4/6/7/8 — recurring, high-value |
| Release candidates to first clean release | 3 (rc.1 go.sum, rc.2 SLSA, rc.3 ✓) |

*Trends to watch: (1) deep review on I/O/crypto/CI phases catches real defects every time — keep it a standing gate. (2) the first live release run finds what darwin-only + green CI cannot. (3) version/label claims need a human product check.*
