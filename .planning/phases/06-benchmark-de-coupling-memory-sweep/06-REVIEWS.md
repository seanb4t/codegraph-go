---
phase: 6
reviewers: [codex]
reviewed_at: 2026-08-16T11:40:21Z
plans_reviewed: [06-01-PLAN.md, 06-02-PLAN.md, 06-03-PLAN.md, 06-04-PLAN.md, 06-05-PLAN.md, 06-06-PLAN.md]
---

# Cross-AI Plan Review — Phase 6

## Codex Review

# Cross-AI Plan Review

## Summary

The plans are unusually thorough and generally trace the real implementation accurately. Their wave ordering is sound, the comparison-runner subtraction is well scoped, and the mutation/census protocols reflect known repository failure modes. The main weaknesses are verification strength rather than implementation intent: 06-02 does not mechanically enforce all four locked D-06 properties; 06-03 proves the pure `CheckRegression` function but does not prove the committed `baseline.json` participates; and 06-04 permits BENCH-03 to close after a local fallback without a successful workflow run. Several negative-only `git diff` and `rg` criteria also remain vacuous-pass-shaped under rule `84d1gfpywd`.

Overall risk: **MEDIUM**. The plans are likely to produce the intended code, but their acceptance gates can certify incomplete BENCH-02/BENCH-03 outcomes.

---

## Plan 06-01 — Benchmark runner de-coupling

### Strengths

- The subtraction points match the current code precisely: the old mode dispatch, flag, resolver, and two-subject loop are all real surfaces at [tools/bench/runner/main.go:156](/Volumes/Code/github.com/seanb4t/codegraph-go/tools/bench/runner/main.go:156), [tools/bench/runner/main.go:161](/Volumes/Code/github.com/seanb4t/codegraph-go/tools/bench/runner/main.go:161), [tools/bench/runner/main.go:178](/Volumes/Code/github.com/seanb4t/codegraph-go/tools/bench/runner/main.go:178), and [tools/bench/runner/main.go:306](/Volumes/Code/github.com/seanb4t/codegraph-go/tools/bench/runner/main.go:306).

- The paired test edits are correctly identified. Existing tests directly exercise `-ts-binary` and `resolveTSBinary` at [tools/bench/runner/main_test.go:482](/Volumes/Code/github.com/seanb4t/codegraph-go/tools/bench/runner/main_test.go:482) and [tools/bench/runner/main_test.go:494](/Volumes/Code/github.com/seanb4t/codegraph-go/tools/bench/runner/main_test.go:494).

- The end-to-end assertion is strongly positive: exactly two records, exact subject value, positive throughput, and `median_of_trials == 1`. This avoids an empty-output pass.

- The surviving-pin check has independent support from the proposed exact-entry test, so the negative diff check is not the only oracle.

### Concerns

- **MEDIUM — The verification introduces an undeclared Node dependency.** The plan removes Node from the benchmark architecture but parses smoke-test JSON with `node -e` at [06-01-PLAN.md:246](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/06-benchmark-de-coupling-memory-sweep/06-01-PLAN.md:246). A machine with Go and git but no Node fails despite the implemented Go path being correct.

- **LOW — “No `t.Skip`/`//nolint` added” is negative-only and underspecified.** At [06-01-PLAN.md:257](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/06-benchmark-de-coupling-memory-sweep/06-01-PLAN.md:257), there is no companion test-count or file-set assertion specifically proving the test files still exist. The named positive test assertions mitigate this for the three critical tests, but not for the entire package.

- **LOW — `git diff --quiet -- NOTICE` is vacuous-pass-shaped.** At [06-01-PLAN.md:311](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/06-benchmark-de-coupling-memory-sweep/06-01-PLAN.md:311), it passes if `NOTICE` is absent from both compared states or if the check is run after an unintended commit. It does not prove the file exists or retains its expected legal content.

### Suggestions

- Replace Node JSON checks with a small Go verifier or an existing Go test.
- Add `test -f NOTICE` plus a stable positive assertion for its MIT copyright line.
- Record before/after `go test -list` output or exact expected test names rather than relying on the `t.Skip` absence check alone.

### Risk Assessment

**LOW–MEDIUM.** The implementation path is solid; most risk lies in portable verification.

---

## Plan 06-02 — Workflow publisher and baseline prose

### Strengths

- The plan correctly identifies the current comparison setup at [bench.yml:96](/Volumes/Code/github.com/seanb4t/codegraph-go/.github/workflows/bench.yml:96), Node setup at [bench.yml:117](/Volumes/Code/github.com/seanb4t/codegraph-go/.github/workflows/bench.yml:117), npm installation at [bench.yml:122](/Volumes/Code/github.com/seanb4t/codegraph-go/.github/workflows/bench.yml:122), and dual-binary invocation at [bench.yml:138](/Volumes/Code/github.com/seanb4t/codegraph-go/.github/workflows/bench.yml:138).

- The job-id, dispatch option, and conditional are treated as one identifier family. The current three linked sites are visible at [bench.yml:69](/Volumes/Code/github.com/seanb4t/codegraph-go/.github/workflows/bench.yml:69), [bench.yml:71](/Volumes/Code/github.com/seanb4t/codegraph-go/.github/workflows/bench.yml:71), and [bench.yml:99](/Volumes/Code/github.com/seanb4t/codegraph-go/.github/workflows/bench.yml:99).

- Preserving action SHA pins is appropriate. The existing actions are fully pinned, for example [bench.yml:104](/Volumes/Code/github.com/seanb4t/codegraph-go/.github/workflows/bench.yml:104) and [bench.yml:154](/Volumes/Code/github.com/seanb4t/codegraph-go/.github/workflows/bench.yml:154).

- The plan correctly recognizes that `TestWorkflowRunBodiesInvokeTask` does not protect `bench.yml`; the exclusion is explicit at [internal/upgrade/taskfile_shape_test.go:99](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/upgrade/taskfile_shape_test.go:99).

### Concerns

- **HIGH — D-06.1 is stated but not positively asserted.** The plan requires exact `runs-on` and `CODEGRAPH_BENCH_RUNNER` values, but its automated verification at [06-02-PLAN.md:154](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/06-benchmark-de-coupling-memory-sweep/06-02-PLAN.md:154) checks neither. A rewrite could move the publisher to `ubuntu-latest`, omit the environment variable, and still pass every automated criterion. The current load-bearing values are at [bench.yml:98](/Volumes/Code/github.com/seanb4t/codegraph-go/.github/workflows/bench.yml:98) and [bench.yml:101](/Volumes/Code/github.com/seanb4t/codegraph-go/.github/workflows/bench.yml:101).

- **HIGH — D-06.4 is checked workflow-wide rather than publish-job-wide.** `rg -c 'if-no-files-found: error' >= 1` at [06-02-PLAN.md:162](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/06-benchmark-de-coupling-memory-sweep/06-02-PLAN.md:162) passes if the publisher drops its upload assertion because another job already contains the same setting at [bench.yml:274](/Volumes/Code/github.com/seanb4t/codegraph-go/.github/workflows/bench.yml:274). There is also no automated positive assertion for `$GITHUB_STEP_SUMMARY`.

- **HIGH — D-06.3 is asserted mainly through prose.** The new job could call `CheckRegression`, invoke `-mode regression`, or add a shell threshold failure while retaining a “non-blocking” comment. The acceptance criteria only require that a line contain `-mode publish`; they do not structurally inspect all run bodies in the publish job.

- **MEDIUM — D-06.2 is incompletely enforced.** `git diff --quiet -- Taskfile.yml` proves this plan did not edit the Taskfile in the current diff, but not that a `bench:publish` target does not already exist or arrive through another concurrent change. The workflow test deliberately excludes all of `bench.yml` at [internal/upgrade/taskfile_shape_test.go:100](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/upgrade/taskfile_shape_test.go:100).

- **MEDIUM — The SHA equality check is vacuous when all `uses:` lines disappear.** At [06-02-PLAN.md:163](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/06-benchmark-de-coupling-memory-sweep/06-02-PLAN.md:163), `0 == 0` passes. Actionlint does not require a checkout, Go setup, cache, or artifact-upload action.

- **LOW — The `rebless` diff criterion is ambiguous.** “No matches beyond comment-prose lines explicitly listed in the SUMMARY” at [06-02-PLAN.md:165](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/06-benchmark-de-coupling-memory-sweep/06-02-PLAN.md:165) is not mechanically checkable and does not prove the job block is byte-identical.

### Suggestions

Add a structural YAML test that parses the `publish` job and asserts:

- exact `runs-on`;
- exact matching `CODEGRAPH_BENCH_RUNNER`;
- one inline `go run ./tools/bench/runner -mode publish`;
- no `task bench:*`, `CheckRegression`, `-mode regression`, `-rebless`, Node, npm, or alternate binary;
- a summary step containing `$GITHUB_STEP_SUMMARY`;
- an upload step for `publish-results.json` with `if-no-files-found: error`;
- expected action set/count, not merely pin-format equality.

Compare the `rebless` job’s parsed subtree before and after rather than grepping diff text.

### Risk Assessment

**HIGH.** This is the largest verification gap: all four locked D-06 properties are described, but two are weakly checked and two are not mechanically checked at all.

---

## Plan 06-03 — Regression-gate proof

### Strengths

- The selected mutations target real, isolated tolerance constants at [internal/bench/regression.go:9](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/bench/regression.go:9) and [internal/bench/regression.go:15](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/bench/regression.go:15).

- Existing tests provide exact non-vacuous oracles: the 11% throughput case at [internal/bench/regression_test.go:43](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/bench/regression_test.go:43) and the 16% RSS case at [internal/bench/regression_test.go:62](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/bench/regression_test.go:62).

- The plan correctly rejects category-error failures. `CheckRegression` checks platform/runner identity before numeric tolerances, so requiring the RED to name throughput or RSS is important.

- The clean-tree precondition, `EXIT` trap, confirmed mutation diff, RED, revert, and green rerun are a strong application of the repository’s mutation protocol.

### Concerns

- **HIGH — This does not prove “against the committed `baseline.json`.”** The tests construct in-memory `Metrics` values and call `CheckRegression` directly at [internal/bench/regression_test.go:502](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/bench/regression_test.go:502). The committed baseline is merely asserted untouched at [06-03-PLAN.md:229](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/06-benchmark-de-coupling-memory-sweep/06-03-PLAN.md:229); it is never loaded or passed to the function. This proves the tolerance branches remain live, but not the complete BENCH-02 mechanism named in the requirement.

- **MEDIUM — The oracle-integrity check is diff-relative and can pass after an unintended prior commit.** `git diff --quiet -- internal/bench/regression_test.go` only proves the file is currently clean relative to HEAD. It does not prove the test table is byte-identical to the pre-plan version. The before/after subtest count catches deletion/addition counts but not changed values or expected outcomes.

- **LOW — “Comment-only” does not guarantee semantic comments were preserved.** The diff-shape check is useful, but a complete deletion/replacement of all comments would pass if every changed line still starts with `//`. The positive retained-`comparison` count partly mitigates this.

### Suggestions

- Add a small test or command that reads `tools/bench/baseline.json`, unmarshals it into `bench.Metrics`, derives same-category current records with 11% lower throughput and 16% higher RSS, then calls `CheckRegression` and positively asserts the expected error categories.
- Hash `regression_test.go` before Task 1 and compare the hash after Task 2, or explicitly assert the values/names of the two critical cases.
- Keep the mutation rehearsal; it remains valuable even after adding the committed-baseline integration check.

### Risk Assessment

**MEDIUM–HIGH.** The pure function is convincingly proven, but the plan overstates that proof as covering the committed baseline.

---

## Plan 06-04 — Measurement, documentation, and census

### Strengths

- The task order is explicit and genuinely prevents document-first execution. Task 2 has a hard precondition that the capture exists and contains two positive records at [06-04-PLAN.md:151](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/06-benchmark-de-coupling-memory-sweep/06-04-PLAN.md:151). Within normal GSD sequential task execution, numbers cannot be published before Task 1 produces the artifact.

- The capture validation is strongly positive: exact record count, exact repo set, exact subject, and positive values at [06-04-PLAN.md:135](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/06-benchmark-de-coupling-memory-sweep/06-04-PLAN.md:135).

- The proposed document content maps well to the existing methodology. The source currently contains median-of-5 at [docs/BENCHMARKS.md:20](/Volumes/Code/github.com/seanb4t/codegraph-go/docs/BENCHMARKS.md:20), a comparison target at [docs/BENCHMARKS.md:50](/Volumes/Code/github.com/seanb4t/codegraph-go/docs/BENCHMARKS.md:50), and the regression tolerances at [docs/BENCHMARKS.md:261](/Volumes/Code/github.com/seanb4t/codegraph-go/docs/BENCHMARKS.md:261).

- The census includes a positive control, a multiline-vs-line-based comparison, a scanned-file floor, and positive required-content checks. This is substantially better than a negative vocabulary scan alone.

### Concerns

- **HIGH — BENCH-03 can remain unverified while the plan still completes.** The plan explicitly allows a local fallback at [06-04-PLAN.md:118](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/06-benchmark-de-coupling-memory-sweep/06-04-PLAN.md:118), while the live workflow verification applies only “if the preferred CI path was used” at [06-04-PLAN.md:136](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/06-benchmark-de-coupling-memory-sweep/06-04-PLAN.md:136). BENCH-03 says `bench.yml` runs and publishes; a local runner invocation does not establish that.

- **MEDIUM — The byte-identical-to-stdout assertion does not establish byte identity.** At [06-04-PLAN.md:142](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/06-benchmark-de-coupling-memory-sweep/06-04-PLAN.md:142), checking that the indexed file’s first line starts with `[` plus a SUMMARY statement does not compare it to an independently retained stdout capture. It also depends on the file having been staged.

- **MEDIUM — Numeric traceability is only a spot check of `files_per_sec`.** The plan publishes five metrics per corpus but only requires spot-checking `files_per_sec`. Query latency, peak RSS, bytes/sec, and cold start could be transcribed incorrectly without failing.

- **MEDIUM — Local measurements may be unsuitable as published project figures.** The fallback records an empty runner label but does not impose stable runner conditions. This does not contradict D-02, but it weakens reproducibility and makes the published figures potentially machine-specific.

- **LOW — The file-floor assertion can still admit a shifted surface.** `>= 12` proves something was scanned, but not the exact expected file set. A deleted critical file and unrelated added files could preserve the count.

### Suggestions

- Make successful `bench.yml` dispatch a blocking BENCH-03 acceptance criterion. If unavailable, allow BENCH-01 documentation work to proceed but leave BENCH-03 explicitly open.
- Tee stdout simultaneously to a temporary file and the committed artifact, then compare SHA-256 values.
- Generate the Markdown data rows mechanically from JSON or verify all ten metric cells against parsed JSON.
- Record an exact sorted census file list and compare it to the expected set, rather than using only a floor.

### Risk Assessment

**MEDIUM–HIGH.** BENCH-01 ordering is sound, but BENCH-03 can be marked complete without its required workflow execution.

---

## Plan 06-05 — Agent-facing files

### Strengths

- Enumerate-before-edit is excellent for preventing post-hoc classification.
- Source-before-mirror addresses a real current inconsistency: `.claude/CLAUDE.md` still carries the old compatibility framing at [CLAUDE.md:15](/Volumes/Code/github.com/seanb4t/codegraph-go/.claude/CLAUDE.md:15), while PROJECT declares it retired at [PROJECT.md:244](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/PROJECT.md:244).
- The stale capability defect is real: STATE still promises migration at [STATE.md:26](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/STATE.md:26), and PROJECT’s constraint still names `codegraph migrate` at [PROJECT.md:244](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/PROJECT.md:244).
- Region-scoping the PROJECT check is correct because historical mentions legitimately survive elsewhere.

### Concerns

- **MEDIUM — The STATE assertion checks only two exact phrases, not the resulting core-value equality.** At [06-05-PLAN.md:214](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/06-benchmark-de-coupling-memory-sweep/06-05-PLAN.md:214), STATE can retain different stale framing and pass. The task says it must match PROJECT’s Core Value, but no automated equality check enforces that.

- **MEDIUM — Marker count is not byte-integrity.** `GSD_MARKERS=14` at [06-05-PLAN.md:221](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/06-benchmark-de-coupling-memory-sweep/06-05-PLAN.md:221) allows marker names, order, or source attributes to change while retaining the count.

- **LOW — Fresh-session verification is optional/manual in practice.** The human check at [06-05-PLAN.md:215](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/06-benchmark-de-coupling-memory-sweep/06-05-PLAN.md:215) is the closest direct test of MEM-02’s file half, but the completion criteria do not clearly block on it.

- **LOW — Negative `git diff --quiet` checks remain context-dependent.** The protection for NOTICE/LICENSE/CHANGELOG at [06-05-PLAN.md:224](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/06-benchmark-de-coupling-memory-sweep/06-05-PLAN.md:224) proves only no current uncommitted delta, not immutable content across task commits.

### Suggestions

- Parse both Core Value sections and assert exact normalized equality.
- Capture the exact ordered GSD marker lines before editing and compare them afterward.
- Make the fresh-session review an explicit blocking checkpoint or clearly retain MEM-02 as pending until performed.
- Compare protected-file hashes against the pre-plan commit rather than the current working-tree diff.

### Risk Assessment

**MEDIUM.** The editorial reasoning is strong; generated-file integrity and direct MEM-02 proof need firmer gates.

---

## Plan 06-06 — Engram memory sweep

### Strengths

- The fail-loud tool-availability precondition directly respects D-16.
- The plan correctly separates read-only enumeration, one-way classification approval, and durable writes.
- Pagination exhaustion, per-scope page counts, deterministic sorting, zero-record scope reporting, and verdict arithmetic are strong positive assertions.
- The blocking decision checkpoint is appropriate because superseding is one-way.
- The plan explicitly preserves historical clauses in mixed records and forbids substituting overwrite/delete for supersede.

### Concerns

- **MEDIUM — A returned identifier proves write acceptance, not supersede semantics.** The plan acknowledges this at [06-06-PLAN.md:86](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/06-benchmark-de-coupling-memory-sweep/06-06-PLAN.md:86) and [06-06-PLAN.md:228](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/06-benchmark-de-coupling-memory-sweep/06-06-PLAN.md:228). This is the locked D-15 trade-off, so it is not a defect to reopen, but residual risk remains material.

- **MEDIUM — The known-minimum-three-rules assertion may conflate memory records with rule-list entries.** The plan requires a rule-scope record count of at least three at [06-06-PLAN.md:160](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/06-benchmark-de-coupling-memory-sweep/06-06-PLAN.md:160), but research explicitly says the live tool surface is unverified. If rules are exposed through a distinct endpoint or pagination shape, this condition could falsely report a truncated enumeration.

- **MEDIUM — The automated closure check does not reconcile executed entries with approved IDs.** It checks section presence and absence of “delete/overwrite” verdicts, but not set equality between approved `supersede` short IDs and executed rows. The prose acceptance criterion requires equality; the automated command at [06-06-PLAN.md:248](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/06-benchmark-de-coupling-memory-sweep/06-06-PLAN.md:248) does not enforce it.

- **LOW — MEM-02 remains assertion-based by locked decision.** The plan properly calls this out at [06-06-PLAN.md:82](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/06-benchmark-de-coupling-memory-sweep/06-06-PLAN.md:82). This is an accepted evidence limitation, not a planning contradiction.

### Suggestions

- After live tool discovery, distinguish rule objects from ordinary memory records and apply the “at least three” assertion to the correct population.
- Add an exact set comparison: approved supersede `short_id`s must equal executed-original `short_id`s, with no duplicates.
- Record the supersede operation name and returned response shape for each write, not only an identifier.
- Clearly mark MEM-02 as “accepted by D-15 evidence standard” rather than “demonstrated,” unless the optional fresh-session check is actually performed.

### Risk Assessment

**MEDIUM.** The one-way operation is handled carefully, but external-tool uncertainty and the deliberately accepted lack of re-query evidence prevent a low-risk rating.

---

## Cross-plan dependency assessment

The wave graph is coherent:

- 06-01 and 06-03 can run independently with no declared file overlap.
- 06-02 correctly depends on the new `publish` runner mode.
- 06-04 correctly depends on all benchmark-side plans.
- 06-05 follows the completed in-tree state.
- 06-06 follows the file sweep and places durable external writes behind a checkpoint.

The principal dependency issue is semantic rather than scheduling: **06-04’s local fallback allows BENCH-03 to remain unproven even though later plans can proceed and the phase may appear complete.**

## Final recommendation

Approve after revisions to three gates:

1. Add a structural, publish-job-scoped verifier for all four D-06 properties in 06-02.
2. Add a committed-`baseline.json` integration assertion to 06-03.
3. Make a successful `bench.yml` publish run mandatory for BENCH-03, while allowing local measurement only as a BENCH-01 fallback.

Those changes would reduce overall execution risk from **MEDIUM** to **LOW–MEDIUM** without reopening any locked decision.

---

## Consensus Summary

Only one external reviewer lane was requested (`--codex`), so there is no multi-reviewer
consensus to synthesize. What follows is the orchestrator's own verification of Codex's
findings against the repository — every HIGH below was independently re-checked against the
cited file before being carried forward.

Codex's verdict: **approve after revisions**, overall risk MEDIUM. The plans are judged
unusually thorough and their wave ordering sound; the weaknesses are concentrated in
*verification strength*, not implementation intent. Three gates would have to change to close
the review: a publish-job-scoped structural verifier for all four D-06 properties (06-02), a
committed-`baseline.json` integration assertion (06-03), and a mandatory successful `bench.yml`
run for BENCH-03 (06-04).

### Strengths (single reviewer, source-verified)

- Subtraction points in 06-01 match the live code exactly (`tools/bench/runner/main.go:156,161,178,306`), and the paired test edits at `main_test.go:482,494` are correctly identified.
- 06-03's mutation targets are real isolated constants (`internal/bench/regression.go:9,15`) with exact pre-existing oracles (`internal/bench/regression_test.go:43,62`), and the plan correctly rejects category-error REDs.
- 06-04 Task 2 carries a hard `<precondition>` that `06-PUBLISH-RESULTS.json` exists with two positive records (`06-04-PLAN.md:151`) — **the ordering question raised for this review is answered affirmatively**: under normal GSD sequential task execution the measurement provably lands before the document that publishes it.
- 06-06 correctly separates read-only enumeration, a blocking one-way classification checkpoint, and durable writes, and respects D-16's fail-loud tool-availability precondition.

### Concerns — HIGH (all five independently re-verified)

1. **06-02 / D-06.1 is never asserted.** The `<automated>` block at `06-02-PLAN.md:154` and every acceptance criterion at `:157-166` check the job key, the conditional, the retired-term census, the SHA pins, and three `git diff` negatives — but **nothing** checks `runs-on: namespace-profile-linux-amd64-4x8` or the `CODEGRAPH_BENCH_RUNNER` env var. Confirmed: those are the load-bearing values at `.github/workflows/bench.yml:98,101`, and D-06.1 calls runner identity "part of the measurement's validity, not cosmetics" (`06-CONTEXT.md:78-82`). A from-scratch rewrite onto `ubuntu-latest` with no env var passes every automated criterion.
2. **06-02 / D-06.4 is checked workflow-wide, and is already satisfied elsewhere.** `test "$(rg -c 'if-no-files-found: error' .github/workflows/bench.yml)" -ge 1` (`06-02-PLAN.md:154,162`) is satisfied by the pre-existing `rebless` job upload at `.github/workflows/bench.yml:274`. Confirmed by reading that line. The publish job can therefore omit its own upload assertion and still pass. Separately, D-06.4 also requires a `GITHUB_STEP_SUMMARY` block (`06-02-PLAN.md:26,132`) and **no criterion asserts `$GITHUB_STEP_SUMMARY` at all**.
3. **06-02 / D-06.3 (non-blocking publish-not-gate) rests on prose.** The contract is restated only in a run-step comment (`06-02-PLAN.md:128`). No criterion structurally inspects the publish job's run bodies, so a rewrite that calls `CheckRegression`, invokes `-mode regression`, or adds a shell threshold failure — while keeping the "non-blocking" comment — passes.
4. **06-03 does not prove the gate fires "against the committed `baseline.json`."** Roadmap success criterion 2 names the committed baseline explicitly. The rehearsal drives `CheckRegression` through in-memory `Metrics` values; `06-03-PLAN.md:229` only asserts `git diff --quiet -- tools/bench/baseline.json`, i.e. that the file was *not touched*. Verified by search: **no Go test loads `tools/bench/baseline.json`** — the only test references (`tools/bench/runner/main_test.go:318,731`) write their own baseline into a temp dir. The tolerance branches are proven live; the committed-data path is not.
5. **06-04 lets BENCH-03 close without ever running the workflow.** The plan sanctions a local fallback at `06-04-PLAN.md:118-124`, and BENCH-03's only live verification is a `<human-check>` explicitly scoped "If the preferred CI path was used" (`06-04-PLAN.md:136`). Verified verbatim. Taking the fallback leaves BENCH-03 — "a `bench.yml` run publishes the absolute numbers without invoking another implementation anywhere in the job" — with **zero** verification, while the plan still completes.

### Concerns — MEDIUM / LOW

Per-plan detail is in the Codex section above. Two cross-cutting shapes the orchestrator adds
after censusing all six plans (28 occurrences of `git diff --quiet` / "returns no matches"):

- **`git diff --quiet -- <path>` is HEAD-relative, not baseline-relative.** Every use of it as a
  tamper guard (NOTICE, LICENSE, CHANGELOG.md, `tools/bench/baseline.json`, `internal/bench/`,
  the 06-05 target files) proves only that the *working tree* currently matches HEAD. Because
  GSD commits per task, any change made and committed by an earlier task in the same plan is
  invisible to a later task's guard. To hold across tasks these must compare against the
  pre-plan commit (`git diff --quiet <pre-plan-sha> -- <path>`) or a recorded hash.
- **`rg -c '<pat>'` used as a "returns no matches" assertion is an inverted gate.** `rg -c`
  exits 1 and prints nothing on the clean path, so any executor chaining it with `&&` sees the
  *good* outcome as a failure — and any executor reading exit status leniently sees a broken
  command as a pass. This repo already has the inverted-`rg -qv` gate in its documented
  failure history. Occurrences: `06-01-PLAN.md:255`, `06-02-PLAN.md:164,165,215`,
  `06-03-PLAN.md:158,228`. Restate each as `test "$(… | rg -c … || true)" = "0"` with a printed
  positive token, paired with a count that must be > 0.

### Divergent Views

None — single reviewer.

### Locked-decision contacts (reported, not raised as defects)

- 06-06's "a returned identifier proves write acceptance, not supersede semantics" is the
  **locked D-15 trade-off**, acknowledged in-plan at `06-06-PLAN.md:86,228`. Reported as
  residual risk, not a defect.
- MEM-02 remaining assertion-based rather than demonstrated is likewise accepted by D-15 and
  called out at `06-06-PLAN.md:82`.
- The local-measurement fallback in 06-04 does not contradict D-02; the HIGH above is about
  BENCH-03's *verification*, not about re-opening the fallback decision.

---

## Verification coverage

The orchestrator independently re-read the following before accepting Codex's findings. Every
HIGH was confirmed at its cited line; none were taken on the reviewer's word.

| Claim | Source checked | Verdict |
|---|---|---|
| D-06.1 unasserted | `06-02-PLAN.md:140-166`, `.github/workflows/bench.yml:96-101` | CONFIRMED — no `runs-on` / `CODEGRAPH_BENCH_RUNNER` assertion exists |
| D-06.4 satisfied by another job | `.github/workflows/bench.yml:274` | CONFIRMED — `if-no-files-found: error` present in the `rebless` upload |
| D-06.4 step-summary unasserted | `06-02-PLAN.md:26,132,154-166` | CONFIRMED — requirement stated, never asserted |
| D-06.3 prose-only | `06-02-PLAN.md:128,157-166` | CONFIRMED |
| D-06 is a four-property locked list | `06-CONTEXT.md:77-90` | CONFIRMED |
| Committed baseline never loaded | `rg 'baseline\.json' --type go internal/ tools/`; `06-03-PLAN.md:221-231` | CONFIRMED — only temp-dir baselines in tests |
| BENCH-03 human-check is conditional | `06-04-PLAN.md:118-124,136` | CONFIRMED verbatim |
| 06-04 ordering is guaranteed | `06-04-PLAN.md:151` `<precondition>` | CONFIRMED — Codex's *strength* finding is correct; the measurement provably precedes the document |
| Negative-guard census | 28 hits across all six `*-PLAN.md` | CONFIRMED — enumerated above |

Codex ran with repo access and produced `file:line`-grounded findings throughout; no
`[reviewed-without-repo-access]` marker was present.
