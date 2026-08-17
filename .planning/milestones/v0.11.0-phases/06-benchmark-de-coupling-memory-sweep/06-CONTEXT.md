# Phase 6: Benchmark De-coupling & Memory Sweep - Context

**Gathered:** 2026-08-16
**Status:** Ready for planning

<domain>
## Phase Boundary

The project publishes its own absolute performance numbers with no second implementation
in the picture, and an agent that starts a session afterward recalls codegraph-go as a
project rather than as a port.

**Requirements:** BENCH-01, BENCH-02, BENCH-03, MEM-01, MEM-02

**In scope:**
- `docs/BENCHMARKS.md` — rewritten to publish absolute throughput / query-latency / peak-RSS
  with methodology and the regression-gate description, no head-to-head content
- `tools/bench/` — the comparison runner removed, the committed head-to-head captures removed,
  `BASELINE.md` framing swept, `realcorpus`'s corpus and prose de-coupled
- `.github/workflows/bench.yml` — the comparison-invoking publish job replaced
- `internal/bench.CheckRegression` — unchanged in behaviour, but proven still-firing
- The engram memory store: repo spine, `rule:repo:` scope, and workspace overlay scopes
- Agent-facing framing files: `.claude/CLAUDE.md`, `.planning/PROJECT.md`, `.planning/STATE.md`

**Out of scope:**
- Any change to `CheckRegression`'s gate semantics or the committed `baseline.json` values
- The `internal/corpora` ↔ `tools/bench/realcorpus` two-manifest separation itself
  (a deliberate, recorded Phase 1 decision — only the cross-reference *text* is touched)
- New benchmark capabilities, new corpora beyond what de-coupling requires

</domain>

<decisions>
## Implementation Decisions

### Historical Artifacts & Published Documentation

- **D-01:** The six committed `tools/bench/headtohead-*.json` captures are **deleted**, not
  archived or retained in place. Git history preserves them. This mirrors the Phase 5
  `codegraph migrate` precedent — remove entirely rather than reframe.
  — **Reversibility:** reversible — the files are recoverable from git history at any time.

- **D-02:** `docs/BENCHMARKS.md` is **rewritten from scratch** on the project's own terms,
  not surgically edited. Its current title is literally "Benchmarks: codegraph-go vs TS
  CodeGraph" and roughly 80 of its 300 lines are comparison content (§Comparison target,
  §Go vs TS 1.3.1 summary, §Run-to-run repeatability, §Superseded v0.1 provisional table).
  — **Reversibility:** costly — the surviving methodology rationale (median-of-5, OS-level
  peak RSS, pinned corpora, regeneration procedure) must be re-authored rather than copied,
  and losing it would leave the published numbers unreproducible.

  **Consequence the plan MUST resolve:** the document's current absolute figures live *inside*
  the comparison tables. A from-scratch rewrite therefore depends on a **fresh absolute-only
  measurement run** producing numbers to publish. This is a plan-level task dependency, not
  an editing task.

- **D-03:** `tools/bench/BASELINE.md` is **in scope** — sweep its comparison framing while
  keeping the investigation findings intact. Its guard rationale (why `CheckRegression` has
  platform / runner-class / `scratch_fs` refusal guards) is load-bearing for BENCH-02 and is
  live engineering documentation, not an archive.
  — **Reversibility:** reversible.

- **D-04:** The in-tree comparison-framing sweep is proven complete by a **positive-controlled
  multiline census**: `rg -U` with a bounded pattern set, positive-controlled by planting a
  known phrase and observing it caught *before* trusting any zero result, counted with
  `rg -o | wc -l` (never `rg -c`, which counts lines rather than matches).
  Rationale: Phase 5's line-based census missed 34 comment-wrapped occurrences, and an
  unbounded `ports` pattern matched the substring inside "reports".

### CI & Runner Shape

- **D-05:** The `headtohead` job in `.github/workflows/bench.yml` (line 96) is **deleted
  entirely** and a fresh absolute-numbers publishing job is authored in its place — not
  renamed-and-stripped.
  — **Reversibility:** costly — several load-bearing properties exist only in that job's body
  and comments (see D-06); rebuilding without them silently degrades measurement validity.

- **D-06:** The fresh publisher **MUST carry forward all four** of these properties. This is a
  checkable list, not guidance:
  1. **Runner pinning + env contract** — `runs-on: namespace-profile-linux-amd64-4x8` and the
     matching `CODEGRAPH_BENCH_RUNNER` env var. `CheckRegression` refuses cross-runner
     comparison (`internal/bench/regression.go:53`), so runner identity is part of the
     measurement's validity, not cosmetics.
  2. **The D-13 / D-01 no-Taskfile exception** — documented at `bench.yml:160-171`. The
     `go run ./tools/bench/runner` invocations stay inline rather than behind `task bench:*`
     targets, specifically so `-rebless` (the only flag that overwrites `baseline.json`)
     stays unreachable by tab-completion. A wrong-platform baseline already produced a
     stable, entirely fictitious 10.6% "regression" that survived three triage rounds.
  3. **The non-blocking publish-not-gate contract** — a slower number must never fail CI. The
     gate is `CheckRegression` against the committed `baseline.json`; conflating publish and
     gate is how a noisy measurement becomes a broken build.
  4. **Artifact upload + job summary** — raw results uploaded with `if-no-files-found: error`
     (a positive assertion that the measurement actually produced output — rule
     `84d1gfpywd`), plus the human-readable `GITHUB_STEP_SUMMARY` block.

- **D-07:** The runner's `-mode headtohead` (`tools/bench/runner/main.go:161`) is **replaced by
  a new single-subject `publish` mode** emitting absolute per-repo `Metrics` JSON for the Go
  binary only. The `headtohead` case, the two-subject measurement loop, and the `-ts-binary`
  flag are deleted. Keeping a two-subject architecture under a new name would be the
  comparison runner surviving, which BENCH-02 forbids.
  — **Reversibility:** costly — the emitted JSON shape changes, so the summary/publish step
  and any consumer of that output change with it.

- **D-08:** BENCH-02's "the gate still fires" proof **reuses Phase 3's FIXT-07 protocol
  verbatim**: pre-mutation `git diff --quiet` gate, confirm the mutation actually applied,
  observe RED, `EXIT`-trap restore, byte-clean revert verified, all recorded in a committed
  `06-MUTATION-LOG.md`. Consistency with a protocol already proven in this repo beats a novel
  one; its failure modes are known.

### Corpus Manifest

- **D-09:** `tools/bench/realcorpus`'s `colbymchenry-codegraph` entry (and its
  `SiblingDir: codegraph-ts`) is **dropped**, leaving `weft-go` and `cockroachdb-pebble` as
  the benchmark corpus. It is the only entry whose presence is milestone-entangled.
  — **Reversibility:** reversible.

- **D-10:** The prose rewrite reaches **realcorpus plus both `internal/corpora`
  cross-references**. `realcorpus`'s package doc and field comments are re-authored on
  absolute-measurement terms (they are currently head-to-head throughout: lines 2-15, 39, 54,
  82, 157), and `internal/corpora`'s package-doc paragraph **and** its `Manifest.Note` text
  are updated to match. Treat this as an identifier-family sweep in the shape of Phase 4's
  FLAG-PARITY work.
  **Why the `Manifest.Note` half matters:** it is a committed *data* field, so a stale pointer
  ships inside `corpora/manifest.json` — not just in source comments.
  — **Reversibility:** reversible.

- **D-11:** `cockroachdb-pebble` (BSD-3-Clause) is **kept**, and the licence-policy difference
  between the two manifests is **re-justified on its own terms**: redistribution risk for a
  fetch-only, never-vendored measurement corpus is a genuinely different question from
  FIXT-01's golden-fixture bar. No surviving comment may cite a head-to-head benchmark as the
  reason. Dropping pebble would leave the benchmark corpus with a single entry — thin ground
  for published absolute numbers.

  **Standing constraint (do not re-litigate):** the two-manifest separation itself is a
  deliberate Phase 1 decision with four reasons recorded in `01-04-PLAN.md` and restated in
  `internal/corpora/manifest.go:6-15` — `realcorpus` performs no network I/O, and its licence
  bar is intentionally wider than `Validate`'s. Merging would force a policy-parameterised
  validator or silently widen the FIXT-01 bar. Phase 6 touches the *text*, never the split.

### Memory Sweep

- **D-12:** The sweep covers **the engram spine plus the agent-facing framing files**.
  MEM-02's stated outcome — "a session started after the sweep recalls no memory describing
  codegraph-go as a port or parity project in the present tense" — is defeated by
  `.claude/CLAUDE.md`'s "ground-up Go rewrite of CodeGraph" opener and Core Value line, and by
  `.planning/STATE.md`'s "uninstall TS CodeGraph… migrate their indexes" sentence, regardless
  of the spine's state. That sentence additionally describes a capability that no longer
  exists (`migrate` was removed in Phase 5). This executes memory `myywc0y9vm`'s recorded
  maintainer correction: the core-value text is competitive framing and should be swept.
  Files: `.claude/CLAUDE.md`, `.planning/PROJECT.md`, `.planning/STATE.md`.
  — **Reversibility:** reversible.

- **D-13:** The supersede test is **present-tense OR forward-looking**. Supersede any record
  whose durable content states the framing as a current standing fact ("parity with TS 1.3.x
  is the baseline") *or* as a future obligation ("Phase N must maintain parity", "the TS
  behaviour is the compatibility target") — because forward-looking records steer future
  sessions just as effectively. Records that describe what was decided or done at a point in
  time are true and **must survive untouched**.
  — **Reversibility:** one-way — superseding a record is an append to the store's durable
  history; the correction cannot be un-issued, only further superseded, and the classification
  judgement is baked into what future sessions recall.

- **D-14:** The sweep covers **spine + `rule:repo:` scope + workspace overlay scopes** — every
  scope a fresh session recalls. Rules are injected as MUST-follow ground truth, so framing
  there would be the most load-bearing of all.

- **D-15:** Completeness is evidenced by a **committed `06-MEMORY-SWEEP.md` enumerating every
  record by `short_id` with a per-record verdict (supersede / leave-historical) and the
  reason**. Post-sweep re-query output is **not** required.

  **Accepted trade-off, recorded deliberately:** without re-query evidence, "the supersede
  actually landed" is asserted rather than demonstrated — the vacuous-pass shape rule
  `84d1gfpywd` warns about. The maintainer accepted this; the full enumeration still gives a
  reviewer the complete population to check against.

- **D-16 (precondition):** The engram MCP endpoint was **unreachable at context-gathering time**
  (`getaddrinfo ENOTFOUND mcp-gw.fzymgc.house`). Any MEM plan must treat engram reachability as
  an explicit precondition and **fail loud** if the store cannot be enumerated — never record a
  sweep as complete against a store it could not read. This is the same fail-loud discipline
  Phase 3 applied to the corpus fetch (`nqwgt6r53a`).

### Claude's Discretion

None — every area presented was decided by the user. No "you decide" options were selected.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Requirements & roadmap
- `.planning/ROADMAP.md` §"Phase 6: Benchmark De-coupling & Memory Sweep" (lines 262-275) —
  goal, the five success criteria, and the Notes paragraph that leaves the head-to-head JSON
  disposition as a scoping call (resolved here as D-01)
- `.planning/REQUIREMENTS.md` lines 47-54 — BENCH-01/02/03, MEM-01/02 verbatim

### Benchmark surface (BENCH-01/02/03)
- `docs/BENCHMARKS.md` — the document being rewritten; read before replacing to identify which
  methodology sections must survive re-authoring
- `tools/bench/BASELINE.md` — the investigation narrative; its guard rationale explains why
  `CheckRegression`'s platform / runner-class / `scratch_fs` refusals exist
- `tools/bench/runner/main.go` — `case "headtohead"` at line 161, `measureSubject` at 349,
  `realcorpus.Corpora()` consumption at 314
- `internal/bench/regression.go` — the gate. Refusal guards at lines 53, 76, 96 are the
  reason runner pinning is load-bearing (D-06.1)
- `tools/bench/baseline.json` — the committed baseline the gate compares against; NOT modified
  by this phase
- `.github/workflows/bench.yml` — the `headtohead` job at line 96; the D-13/D-01 no-Taskfile
  exception documented at lines 160-171; the deliberate runner-profile asymmetry at lines 42-51

### Corpus manifests
- `tools/bench/realcorpus/manifest.go` — the benchmark corpus manifest being de-coupled
- `internal/corpora/manifest.go` lines 6-15, 58-65, 98-108, 134-141 — the recorded four-reason
  non-merge decision, the narrower MIT/Apache-2.0 `validLicenses` bar, and the `Manifest.Note`
  field carrying the cross-reference that D-10 updates
- `.planning/phases/01-corpus-selection-by-measurement/01-04-PLAN.md` — where the four
  non-merge reasons are recorded in full
- `corpora/manifest.json` — the committed data file `Manifest.Note` ships in

### Protocol precedents to reuse
- `.planning/phases/03-non-vacuity-proof-unconditional-ci-execution/03-MUTATION-LOG.md` —
  the FIXT-07 mutation protocol D-08 reuses verbatim
- `.planning/phases/05-process-ci-in-tree-sweep/` (05-07, 05-08 SUMMARYs) — the CODE-01 gap
  closure whose census technique D-04 adopts

### Memory sweep targets (MEM-01/02)
- `.claude/CLAUDE.md` — the "ground-up Go rewrite" opener and the Core Value paragraph
- `.planning/PROJECT.md` — the Core Value statement
- `.planning/STATE.md` — the "uninstall TS CodeGraph… migrate their indexes" line, which also
  names a removed capability

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- **`internal/bench.CheckRegression` + `baseline.json`** — the gate survives this phase
  untouched in behaviour. Only its *proof* is new work.
- **The `bench.yml` measurement scaffolding** — runner pinning, `CODEGRAPH_BENCH_RUNNER`
  wiring, `nscloud-cache-action` caching, and artifact upload are all implementation-neutral
  and transfer directly into the fresh publisher (D-06).
- **`realcorpus.Corpora()` / `Resolve()` / `ErrNeedsClone`** — the fetch-free, pinned-SHA
  resolution contract is sound and stays; only one entry and the surrounding prose change.
- **The FIXT-07 mutation harness discipline** from Phase 3 — a proven, in-repo protocol
  D-08 adopts wholesale rather than reinventing.

### Established Patterns
- **Two deliberately separate pinned-corpus manifests.** `internal/corpora` (network-fetching,
  strict MIT/Apache-2.0) and `tools/bench/realcorpus` (fetch-free, wider licence bar) are
  split on purpose with four recorded reasons. Phase 6 must not merge them.
- **Positive assertions in guards** (rule `84d1gfpywd`) — `if-no-files-found: error` is an
  existing instance; the new publisher keeps it.
- **Identifier-family sweeps** — Phase 4's FLAG-PARITY removal swept
  `FLAG-PARITY` / `flag_parity` / `flag-parity` / `flagParity` together. D-10's
  cross-reference update is the same shape.
- **Fail-loud on a missing precondition** — Phase 3's corpus fetch asserts the corpus exists
  before measuring, so a missing corpus is never misattributed. D-16 applies this to engram.

### Integration Points
- `tools/bench/runner/main.go:314` consumes `realcorpus.Corpora()` — dropping an entry (D-09)
  changes what the publisher measures and therefore what BENCHMARKS.md can publish.
- `internal/corpora/manifest.go` and `tools/bench/realcorpus/manifest.go` point at each other
  bidirectionally; D-10 keeps both halves accurate.
- `internal/bench/metrics.go` documents which fields do and do not participate in
  `CheckRegression`'s comparison — the new `publish` mode's JSON must stay compatible with
  what the gate path expects.

</code_context>

<specifics>
## Specific Ideas

- **The `migrate` precedent is the governing analogy for D-01.** Phase 5's maintainer ruling
  removed a capability outright rather than reframing it. Deleting the head-to-head captures
  applies the same instinct: the parity era's artifacts go, git history remembers.
- **"Delete and rebuild" must not mean "silently drop what was hard-won."** D-06 exists
  specifically because the four carried-forward properties live only in comments in the job
  being deleted — the enumeration converts an open-ended rewrite into a checkable list.
- **The census instrument is itself the thing under test.** Phase 5 produced three instances of
  the same meta-bug: each verification layer trusted the layer below's enumeration. D-04's
  planted-phrase positive control exists to break that chain.

</specifics>

<deferred>
## Deferred Ideas

- **Replacement third benchmark corpus entry.** Dropping `colbymchenry-codegraph` (D-09) narrows
  the corpus from three repos to two. Whether the published absolute numbers want broader
  language/size coverage is a measurement-quality question, not a de-coupling question — a
  future phase.
- **Post-sweep re-query verification for the memory sweep.** Explicitly declined for this phase
  (D-15); noted here so a future audit knows it was considered, not overlooked.
- **Renaming the `realcorpus` package itself.** D-10 rewrites its prose but keeps the package
  name; a rename would ripple through five importing files and is not required by BENCH-02.

### Reviewed Todos (not folded)
- **"post-release-verify.yml's event-aware conclusion guard has no test asserting it"**
  (score 0.9, area: ci) — a genuine instance of rule `84d1gfpywd`'s negative-only-guard
  problem, but it targets `post-release-verify.yml`, not the benchmark surface. Out of scope.
- **"Add golangci-lint with gofmt and idiomatic Go linters"** (score 0.9, area: ci) — general
  CI tooling, unrelated to benchmark de-coupling or the memory sweep. Out of scope.

</deferred>

---

*Phase: 6-Benchmark De-coupling & Memory Sweep*
*Context gathered: 2026-08-16*
