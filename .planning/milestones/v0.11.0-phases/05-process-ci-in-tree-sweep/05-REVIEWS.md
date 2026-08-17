---
phase: 5
reviewers: [codex]
reviewed_at: 2026-08-15T21:14:31Z
plans_reviewed: [05-01-PLAN.md, 05-02-PLAN.md, 05-03-PLAN.md, 05-04-PLAN.md, 05-05-PLAN.md, 05-06-PLAN.md]
---

# Cross-AI Plan Review — Phase 5 (Process, CI & In-Tree Sweep)

Reviewer: Codex (`gpt-5.6-sol`), source-grounded against the repo at
`/Volumes/Code/github.com/seanb4t/codegraph-go` (workdir pinned, read-only
sandbox). Convergence cycle 1 of the phase's review loop.

## Consensus Summary

Codex and the orchestrator's own source checks agree: the phase's six plans
are structurally strong — deletion boundaries match the live dependency graph,
the blast radius is owned per plan, the wave split is sensible, and the
explicit `go test -count=1 ./testdata/golden/...` discipline avoids the
testdata/ coverage trap. But the sweep as planned does not yet **guarantee**
the phase goal (framing-free surfaces with `codegraph migrate` removed), and
several acceptance gates cannot run green as written. All code-level claims
below were re-verified against the working tree during this review.

### Agreed Strengths

- **Blast radius ownership (05-01).** The migrate cut ties the CLI
  registration (`internal/cli/root.go:59`), the whole `internal/migrate`
  package, the graphstore cursor API (`GetMigration`/`PutMigration`/
  `migrationRecordName`), the test fakes, the `ts-*` fixtures, and the
  `modernc.org/sqlite` sole-use dependency together in one diff — verified as
  the real shape of the tree (store.go:51-56/:197-203, batch.go:113-118,
  keys.go:195-204, resolve_test.go:1033, six query fake readers, go.mod
  :40/:160-162, 28 go.sum lines).
- **The golden-suite invocation is called out correctly.** `TestGoldenFixturesExist`
  (`testdata/golden/golden_test.go:66`) lives under `testdata/`, outside
  `go test ./...` discovery; 05-01's explicit `go test -count=1
  ./testdata/golden/...` and `TestGoldenScenarioCountIsExact` (golden_test.go:332)
  are the right guards.
- **Required-check protection is correctly identified (05-03).** The fixture
  at `internal/upgrade/taskfile_shape_test.go:36-51` and the ruleset rationale
  for the seven required contexts are real and load-bearing.
- **Census discipline is right in principle.** Term-by-term with recorded
  reasons, no bare-word regex, product surface (`tsextract`, language
  registry, capability-matrix cells) preserved — all consistently stated.
- **Small attributed re-freeze (05-06).** The corpus-src↔golden coupling is
  verified (go-explore-multi.json embeds the comment bytes) and the
  same-phase re-freeze via `task golden:regen` is the correct D-07 unit.

### Agreed Concerns

1. **Census gaps leave PROC-01/CODE-01 formally unfulfilled and several
   acceptance gates cannot pass AS WRITTEN.** Verified rows that the plans
   either misclassify or omit:
   - `bug_report.yml:77` "Migrated from TypeScript CodeGraph" — 05-02 labels
     the file census-clean and forbids edits, yet its own verify
     (`rg -i 'parity|ports|TypeScript CodeGraph' .github/ISSUE_TEMPLATE`) then
     can never reach 0 with 05-02's task-1/acceptance gate (both include
     `TypeScript CodeGraph`).
   - `internal/query/{engine.go:180, explore.go:211, files.go:25,
     worktree_notice.go:43, render_status_test.go:132}` carry verified
     comparison rows ("mirrors TS's", "TS-parity explore pipeline", "matching TS
     1.3.1", "matching TS's", "not TS's string") yet none of these files appears
     in 05-04's `files_modified`.
   - `internal/cli/githooks.go:15` ("Go-only surface extension over
     internal/githooks — TS 1.3.1 has no standalone githooks command") is in
     neither 05-05's `files_modified` nor any task action; `install_test.go`
     appears in the task `<files>` but not in the frontmatter `files_modified`.
   - `.github/workflows/bench.yml:96` (`name: head-to-head publish`) and
     `:127` (`name: Run head-to-head benchmark`) are comparison framing that
     PROC-03's success criterion ("job names, step names and comments carry no
     retired framing") covers; 05-03 both forbids all job-name edits and
     greps for `head-to-head` expecting 0 — the two cannot both hold.
2. **Verification is narrower than prose across several plans.** Grep families
   in 05-04/05-05 omit variants (`TS's`, `TS-parity`, "matching TS 1.3.1"),
   the pr-template policy check is non-enforcing
   (`python3 scripts/pr_template_policy.py --check 2>/dev/null || true` — the
   script has no `--check` flag and reads `PR_BODY`/`AUTHOR_ASSOCIATION`
   from the environment, confirmed by reading scripts/pr_template_policy.py:91),
   and 05-06's regen gate prints a file count without asserting it.
3. **A final retained-use inventory is missing.** Given the verified census
   omissions above, no plan produces a positive, classified inventory of every
   remaining `TS|parity|port|original|upstream` occurrence across
   `internal/`,`tools/`,`test/` — the phase cannot prove "nothing left
   referencing" (CODE-03) or "no comparison framing" (CODE-01) without one.

### The Converged Verdict

This is the convergence CYCLE 1 review. The plans are directionally right and
individually strong, but they do not yet converge on the phase goal: verified
framing rows remain on automatically-scanned surfaces and the phase's own
success criteria (PROC-01/03, CODE-01) are not met by the file sets as planned.
Executive verdict: **NOT CONVERGED — request changes** and re-review once the
four HIGH census gaps and the non-enforcing-verification class are closed;
with a final retained-use inventory gate added, the phase would be
independently verifiable.

### Agreed Notes (LOW)

- 05-01's counts: "2 ts-fixtures" (there are 3 files) and "eight test-fake
  stubs" (7 method stubs: resolve_test PutMigration + six query GetMigration),
  plus "only deletions" overstates the non-deletion edits.
- 05-06 lists both `go-explore-multi.json` and `go-node-multi.json` as
  re-freeze targets, but only go-explore-multi.json embeds the "synthetic-parity
  corpus" bytes; the node transcript may not change.
- `.claude/CLAUDE.md`'s "One-way migration from its SQLite format still binds"
  (line 15) and the `TS CodeGraph v1.3.x … baseline` sentence are outside the
  D-10 row set 05-05 touches — either sweep or record as KEEP.

---

## Codex Review

# Cross-AI Plan Review

## Overall assessment

The plans are unusually thorough about blast radius, coupling, and verification, but they do not yet guarantee the Phase 5 goal. Three source-verified blockers stand out:

1. Plan 02 wrongly classifies `bug_report.yml` as clean, despite its explicit TypeScript migration option.
2. Plan 03 intentionally preserves `head-to-head` benchmark job/step names, contradicting PROC-03’s requirement for framing-free job and step names.
3. Plans 04–05 omit multiple comparison-framing comments that remain under `internal/`, despite claiming a complete CODE-01 sweep.

Overall risk: **HIGH until those coverage gaps are corrected.**

---

## Plan 01 — Remove `codegraph migrate`

### Summary

This is the strongest plan. Its deletion boundary closely matches the live dependency graph: CLI registration, migration package, graphstore cursor API, test doubles, fixtures, documentation, and the sole-use SQLite dependency are correctly connected. The principal weaknesses are verification wording and a few inaccurate counts, not the architectural cut itself.

### Strengths

- The CLI removal is correctly anchored at [`internal/cli/root.go:59`](internal/cli/root.go:59), which registers `newMigrateCmd()`.
- The graphstore ripple is real and properly included:

  - [`internal/graphstore/store.go:56`](internal/graphstore/store.go:56) exposes `GetMigration`.
  - [`internal/graphstore/store.go:203`](internal/graphstore/store.go:203) exposes `PutMigration`.
  - [`internal/graphstore/pebble_store.go:281`](internal/graphstore/pebble_store.go:281) and [`internal/graphstore/batch.go:117`](internal/graphstore/batch.go:117) implement them.
  - [`internal/graphstore/keys.go:204`](internal/graphstore/keys.go:204) defines the dedicated key.
  - Query and indexer fakes genuinely implement the otherwise-orphaned interface methods, for example [`internal/query/seeding_test.go:36`](internal/query/seeding_test.go:36) and [`internal/indexer/resolve_test.go:1033`](internal/indexer/resolve_test.go:1033).

- The explicit golden-suite invocation is necessary and well chosen. The fixture checks live at [`testdata/golden/golden_test.go:66`](testdata/golden/golden_test.go:66), outside ordinary `go test ./...` package discovery.
- The dependency claim is supported: [`go.mod:40`](go.mod:40) directly requires `modernc.org/sqlite`, and the production reader imports it at [`internal/migrate/reader.go:20`](internal/migrate/reader.go:20).
- Deleting the fixtures together with their assertions is the correct atomic boundary. The README and node-ID comment really do reference them at [`testdata/golden/README.md:10`](testdata/golden/README.md:10) and [`internal/indexer/nodeid/nodeid.go:37`](internal/indexer/nodeid/nodeid.go:37).

### Concerns

- **MEDIUM — The plan’s “nothing references migrate” verification is narrower than its stated truth.** It checks `internal/migrate` imports and selected symbols, but the success language says the command has no man page or documentation anywhere. A full bounded census should include Cobra help tests, shell completion/man snapshots, scripts, `.claude/`, and `.github/`.
- **LOW — Fixture counts are inconsistent.** The plan deletes three files, but its final success criteria say “2 ts-fixtures are gone.” The source confirms three distinct artifacts: `ts-schema.sql`, `ts-schema.dump.sql`, and `ts-version.txt`.
- **LOW — “Only deletions” is inaccurate.** `root.go`, graphstore interfaces, README files, node-ID comments, and module metadata are edited, not merely deleted.
- **LOW — The first task runs build/vet before the cursor API is removed.** This likely remains green because retaining unused interface methods is legal, but calling it an end-to-end proof at that intermediate point overstates what has been established.

### Suggestions

- Add a final command-surface assertion, such as checking `go run ./cmd/codegraph help` and ensuring `migrate` is absent from command listings.
- Add a bounded retained-use report for `migrate` rather than only absence checks. This prevents legitimate terms such as installer-state migration from being mistaken for CODE-03 residue.
- Correct all “two fixture” references to “three fixture files.”
- Replace the “only deletions” expectation with an explicit allowed-path list.

### Risk assessment

**LOW–MEDIUM.** The implementation boundary is correct and source-supported. Most remaining risk is in verification precision.

---

## Plan 02 — Issue and PR templates

### Summary

The plan correctly identifies several framing-bearing templates and correctly verifies that the PR policy does not depend on a `## Parity` heading. However, it contains a source-verified omission that directly prevents PROC-01 from being satisfied: `bug_report.yml` is not clean.

### Strengths

- The policy mechanism is correctly understood. [`scripts/pr_template_policy.py:74`](scripts/pr_template_policy.py:74) defines recognized template headings, and none is `## Parity`.
- The actionable feature-request framing is real:

  - The migration option appears at [`feature_request.yml:49`](.github/ISSUE_TEMPLATE/feature_request.yml:49).
  - The `id: parity` block starts at [`feature_request.yml:55`](.github/ISSUE_TEMPLATE/feature_request.yml:55).
  - Its label explicitly names TypeScript CodeGraph at [`feature_request.yml:57`](.github/ISSUE_TEMPLATE/feature_request.yml:57).

- The default PR template contains the claimed framing at [`pull_request_template.md:35`](.github/pull_request_template.md:35).
- The feature and enhancement PR templates explicitly compare against TypeScript CodeGraph at [`feature.md:46`](.github/PULL_REQUEST_TEMPLATE/feature.md:46) and [`enhancement.md:67`](.github/PULL_REQUEST_TEMPLATE/enhancement.md:67).

### Concerns

- **HIGH — `bug_report.yml` is incorrectly classified as clean and prohibited from editing.** The live file includes the option `Migrated from TypeScript CodeGraph` at [`bug_report.yml:77`](.github/ISSUE_TEMPLATE/bug_report.yml:77). Leaving it byte-identical violates PROC-01.
- **MEDIUM — The policy verification is non-enforcing.** The command ends with:
  ```sh
  python3 scripts/pr_template_policy.py --check 2>/dev/null || true
  ```
  The script does not implement the claimed `--check` interface; it reads `PR_BODY`, `AUTHOR_ASSOCIATION`, and `CHANGED_FILES` from the environment beginning at [`scripts/pr_template_policy.py:91`](scripts/pr_template_policy.py:91). `|| true` converts every failure into success.
- **MEDIUM — The positive controls are only described, not consistently enforced.** A zero-result grep can pass because the scan target changed or a pattern became wrong. The plan acknowledges this class of defect elsewhere but does not encode the control here.
- **LOW — The plan ambiguously permits replacing a `## Parity` heading with a `### Compatibility` subsection.** That may introduce inconsistent hierarchy depending on surrounding headings. The final text should be predetermined after inspecting each template’s structure.

### Suggestions

- Add `.github/ISSUE_TEMPLATE/bug_report.yml` to the edited files and remove or reword the migration-origin option.
- Replace the policy invocation with explicit fixture runs, for example setting `PR_BODY` to each template body and checking the actual exit status.
- Remove `|| true`.
- Add YAML parsing or the repository’s normal template/workflow test target so malformed issue-form YAML is caught.
- Make the expected section hierarchy explicit for `feature.md` and `enhancement.md`.

### Risk assessment

**HIGH.** As written, the plan knowingly preserves a source line that violates PROC-01, and its main policy verification cannot fail.

---

## Plan 03 — Workflow sweep

### Summary

The plan correctly protects required-check identities and distinguishes workflow logic from prose. But its handling of `bench.yml` conflicts with PROC-03: it promises framing-free job and step names while deliberately retaining `head-to-head` names.

### Strengths

- The required-check fixture is real and clearly documented at [`internal/upgrade/taskfile_shape_test.go:36`](internal/upgrade/taskfile_shape_test.go:36), with the seven names enumerated at [`taskfile_shape_test.go:43`](internal/upgrade/taskfile_shape_test.go:43).
- The run-body guard really checks in-scope step bodies and exception matching at [`taskfile_shape_test.go:1337`](internal/upgrade/taskfile_shape_test.go:1337).
- The two contributor-visible JavaScript messages are correctly located:

  - [`require-issue-link.yml:125`](.github/workflows/require-issue-link.yml:125)
  - [`auto-close-unsolicited-prs.yml:91`](.github/workflows/auto-close-unsolicited-prs.yml:91)

- Renaming only the corpora step label while preserving `run: task test:golden` is a sensible low-risk edit.
- Keeping identifiers such as `TestGoreleaserPinParity` is defensible because they describe agreement among in-repository restatements, not another product.

### Concerns

- **HIGH — Framing remains in workflow names by design.** `bench.yml` currently has:

  - Job display name `head-to-head publish` at [`bench.yml:96`](.github/workflows/bench.yml:96)
  - Step name `Run head-to-head benchmark` at [`bench.yml:127`](.github/workflows/bench.yml:127)

  The plan explicitly prohibits all job-name edits and only searches for certain phrases afterward. PROC-03 requires job and step names to carry no retired framing. These two names remain direct comparison framing.
- **HIGH — The plan claims “no `run:` body changes,” but the two JavaScript edits are inside action `with.script` bodies.** They are not YAML `run:` fields, so the existing Go guard will not validate their semantic preservation. The plan should say “no shell `run:` changes,” not imply that all executable bodies are untouched.
- **MEDIUM — The benchmark rewording is semantically awkward and temporary.** Calling the TypeScript binary merely “the benchmark binary” can obscure what the workflow actually executes. Phase 6 will delete it, but Phase 5 should not make the workflow misleading.
- **MEDIUM — The final grep does not cover `head-to-head`.** Thus the verification can pass while the known offending names remain.
- **LOW — Live ruleset verification is described as optional/read-only but placed inside an autonomous plan.** Network or authentication failure should produce an explicit “unverified live state” result, not an ambiguous success.

### Suggestions

- Permit renaming the non-required benchmark job and step names. Required-check protection should apply to the seven actual required contexts, not every job in the repository.
- Include `head-to-head`, `comparison`, and related census terms in the final workflow review.
- Clarify that JavaScript action scripts are executable bodies and require syntax/behavior validation even though they are not `run:` fields.
- Use transparent neutral wording while the comparator remains, such as “legacy benchmark run,” if that wording is consistent with the milestone decision.
- Add a focused check that the edited JavaScript still parses.

### Risk assessment

**HIGH.** The plan cannot meet its own framing-free workflow truth while intentionally preserving two framing-bearing names.

---

## Plan 04 — Query package and identifier sweep

### Summary

The identifier renames and capability-matrix edits are well scoped, but this is not a complete `internal/query` sweep. Several framing-bearing files and comments are absent from both `files_modified` and task instructions, contradicting the plan’s “every comment across internal/query” claim.

### Strengths

- The `syntheticParity*` identifier family is real and correctly grouped:

  - [`internal/query/explore_test.go:75`](internal/query/explore_test.go:75)
  - [`testdata/golden/behavioral_test.go:237`](testdata/golden/behavioral_test.go:237)

- The capability-matrix comments are accurately identified at [`matrix.go:18`](internal/indexer/capability/matrix.go:18) and [`matrix_test.go:217`](internal/indexer/capability/matrix_test.go:217).
- The stale `golden_parity_test.go` reference is genuine at [`internal/query/render_results.go:12`](internal/query/render_results.go:12).
- Separating corpus-source comment edits from query-comment edits avoids mixing the re-freeze cause into the larger sweep.

### Concerns

- **HIGH — Multiple `internal/query` framing rows are omitted.** Examples include:

  - [`internal/query/engine.go:179`](internal/query/engine.go:179): “mirrors TS’s own…”
  - [`internal/query/explore.go:211`](internal/query/explore.go:211): “full TS-parity explore pipeline”
  - [`internal/query/files.go:25`](internal/query/files.go:25): “matching TS 1.3.1’s…”
  - [`internal/query/worktree_notice.go:43`](internal/query/worktree_notice.go:43): “matching TS’s…”
  - [`internal/query/render_status_test.go:132`](internal/query/render_status_test.go:132): direct TypeScript comparison

  None of these files appears in Plan 04’s `files_modified`.
- **HIGH — The plan’s verification patterns are too narrow to catch the omissions.** For example, searches for `TS CodeGraph`, `ported verbatim`, or `shape parity` will not catch every `TS's`, `TS-parity`, or “matching TS 1.3.1” variant.
- **MEDIUM — The plan says `status.go:44` must remain unchanged, but that line contains extensive present comparison vocabulary.** It is historical context, but preserving the entire paragraph conflicts with the broader milestone statement that framing should be removed from code comments. A short past-tense historical note could be retained without preserving the live comparison exposition.
- **LOW — Some instructions suggest replacing external-source provenance with invented internal provenance**, such as “captured as a constant list from the project’s generated-file taxonomy.” Unless that taxonomy exists, the rewrite should describe behavior without asserting a new source.

### Suggestions

- Add `engine.go`, `explore.go`, `files.go`, `worktree_notice.go`, and `render_status_test.go` to the census and plan.
- Run a post-edit retained-use inventory over all of `internal/query`, then classify every remaining `TS`, `parity`, `port`, `original`, and `upstream` occurrence.
- Rewrite the historical `status.go:44` note more compactly in past tense rather than preserving the full comparison.
- Avoid replacing removed provenance with unsupported provenance; use direct behavioral descriptions.

### Risk assessment

**HIGH.** The current file list leaves several directly verified CODE-01 violations behind.

---

## Plan 05 — Remaining in-tree sweep

### Summary

The plan recognizes several subtle couplings, especially marker bytes, MCP scan prose, and proto/generated-comment synchronization. Nevertheless, it has file-list omissions and overstates its ability to complete CODE-01.

### Strengths

- The marker-contract concern is valid. Comments can change while installed marker bytes remain stable; tests in `internal/agents` are the right guard.
- The stale MCP golden reference exists at [`internal/mcp/archtest/protocol_version_test.go:56`](internal/mcp/archtest/protocol_version_test.go:56), and limiting that edit to prose is prudent.
- The proto/generated pair genuinely contains matching comments:

  - [`internal/schema/graph.proto:36`](internal/schema/graph.proto:36)
  - [`internal/schema/graph.pb.go:54`](internal/schema/graph.pb.go:54)

  Editing both together avoids documentation drift.
- The stale TS/JS golden filename is correctly identified at [`internal/corpora/record.go:104`](internal/corpora/record.go:104).
- Including `.claude/CLAUDE.md` follows the explicit D-10 decision and addresses an important agent-facing surface.

### Concerns

- **HIGH — `internal/cli/install_test.go` is required by the task but absent from `files_modified`.** Its framing comment exists at [`internal/cli/install_test.go:328`](internal/cli/install_test.go:328). Depending on executor enforcement, the task may either fail to edit it or mutate an undeclared file.
- **HIGH — `internal/cli/githooks.go` is omitted entirely.** It contains direct comparison framing at [`internal/cli/githooks.go:15`](internal/cli/githooks.go:15), but it appears in neither the files list nor task actions.
- **MEDIUM — The plan’s claim that only comments change is inaccurate.** `.claude/CLAUDE.md` content changes, and some described “string literals in output messages” could affect user-visible behavior. Those should be distinguished from comments and tested accordingly.
- **MEDIUM — `graph.pb.go` is generated code marked as such.** A manual comment edit may be pragmatic, but the plan should add a byte-level assertion that only comment lines changed and document how future regeneration preserves the new comments.
- **MEDIUM — The final grep families are incomplete.** They would not catch every remaining comparison phrase, especially “Go-only surface extension,” “no TS counterpart,” or simple `TS's`.
- **LOW — The plan says past-tense history stays, but the milestone goal names comments broadly.** The retained-history rule needs a single explicit boundary so executors do not make inconsistent decisions file by file.

### Suggestions

- Add `internal/cli/install_test.go` and `internal/cli/githooks.go` to the formal file ownership list and actions.
- Perform a complete retained-use report after Plans 04 and 05 over `internal/`, `tools/`, and `test/`.
- Add a schema-pair check that compares the relevant field comments between `.proto` and `.pb.go`.
- Separate comment-only verification from user-visible string verification.
- Enumerate the accepted historical-comment keep set explicitly.

### Risk assessment

**HIGH.** Directly verified omitted files prevent the plan from completing CODE-01.

---

## Plan 06 — Corpus comments and re-freeze

### Summary

The plan correctly isolates the corpus-source rewording and its required golden regeneration. The main weakness is that its automated verification does not actually enforce the narrow-diff guarantee it treats as the core safety property.

### Strengths

- The four source comments are correctly identified:

  - [`accounts/manager.go:3`](corpus/behavioral/src/accounts/manager.go:3)
  - [`accounts/validate.go:6`](corpus/behavioral/src/accounts/validate.go:6)
  - [`orders/validate.go:6`](corpus/behavioral/src/orders/validate.go:6)
  - [`recovery/recovery_test.go:5`](corpus/behavioral/src/recovery/recovery_test.go:5)

- The coupling is real: `go-explore-multi.json` embeds the source text, visible at [`corpus/behavioral/go-explore-multi.json:3`](corpus/behavioral/go-explore-multi.json:3).
- `golden:regen` genuinely invokes the Go capture program after fetching and asserting corpora at [`Taskfile.yml:70`](Taskfile.yml:70).
- The scenario-count guard is substantive, not a mere passing exit. [`testdata/golden/golden_test.go:332`](testdata/golden/golden_test.go:332) derives and asserts the exact count.

### Concerns

- **MEDIUM — The automated verification prints the number of changed files but never asserts it.** This line:
  ```sh
  n=$(git diff --name-only | wc -l); echo "files in diff: $n"
  ```
  does not fail on extra files. The key “STOP on any unexpected diff” property remains manual.
- **MEDIUM — `task golden:regen` regenerates all Go-side captures.** The capture program has multiple output paths, including files written around [`gocapture/main.go:247`](testdata/golden/gocapture/main.go:247), [`main.go:270`](testdata/golden/gocapture/main.go:270), and MCP outputs around [`main.go:312`](testdata/golden/gocapture/main.go:312). A strict allowed-diff check is therefore essential.
- **MEDIUM — Redirecting all regeneration output to `/dev/null 2>&1` hides useful failure evidence.** A capture failure or corpus-fetch problem should remain visible.
- **LOW — `CASES.json` is listed as potentially modified even though the plan prohibits changing it.** This weakens file ownership and the narrow-diff contract.
- **LOW — The plan runs regeneration in the verification step after already regenerating in the action.** The second run is useful as an idempotence check, but it should be described and assert that it produces no additional diff.

### Suggestions

- Replace the file-count print with an exact allowed-path comparison.
- Explicitly fail if `CASES.json` changes.
- Preserve command output on failure.
- Treat the second regeneration as an idempotence proof: capture the diff hash before and after and require equality.
- Verify that JSON differences are exclusively `synthetic-parity corpus` → `behavioral corpus`, not merely that the old phrase disappeared.

### Risk assessment

**MEDIUM.** The change is narrow and conceptually sound, but the automated guard does not enforce its most important safety condition.

---

# Phase-wide concerns

- **HIGH — The source requirements contain an unresolved contradiction.** CODE-03 is amended to remove migration, while the “Out of Scope” table still says “Removing `codegraph migrate`” is out of scope. The plans follow the newer maintainer ruling, but the stale row should be corrected before execution to prevent tooling or reviewers from selecting the wrong authority.
- **HIGH — There is no final comprehensive census plan.** Plans 04–05 divide the tree, but verified omissions show that the research table was not exhaustive. Add a final plan or phase gate that produces a classified retained-use inventory across all in-scope paths.
- **MEDIUM — Several verification commands are weaker than their prose.** Examples include `|| true`, printing counts without asserting them, and phrase lists that omit known variants.
- **MEDIUM — “Zero job-name edits” is treated as a universal safety rule when only required-check names are protected.** This blocks necessary cleanup of non-required framing-bearing workflow names.

# Recommended disposition

**Request changes before approval.** At minimum:

1. Fix `bug_report.yml`.
2. Resolve the two `head-to-head` workflow names.
3. Add the omitted query and CLI files to CODE-01.
4. Replace non-enforcing verification commands.
5. Add a final, classified retained-use census for all in-scope directories.
---

### Convergence verdict

CYCLE_SUMMARY: current_high=4 current_actionable=6

**Verdict: NOT CONVERGED — request changes.** The plans are structurally strong and source-accurate in their deletion boundaries and wave ordering, but verified census gaps on automatically-scanned surfaces mean the phase success criteria are not guaranteed: PROC-01 (`bug_report.yml:77` "Migrated from TypeScript CodeGraph" misclassified as census-clean, and 05-02's own verification cannot reach zero while edits to that file are forbidden), PROC-03 (`bench.yml` job name "head-to-head publish" :96 and step name "Run head-to-head benchmark" :127 retain comparison framing that the phase criterion demands removed), and CODE-01 (internal/query rows in engine.go:180/explore.go:211/files.go:25/worktree_notice.go:43/render_status_test.go:132 and internal/cli/githooks.go:15 are outside every plan's files_modified set). Several acceptance gates cannot pass as written (bench.yml `rg head-to-head` == 0, `.github/ISSUE_TEMPLATE` grep with bug_report untouched, `pr_template_policy.py --check || true`). A final classified retained-use inventory across internal/tools/test is needed so "nothing left referencing" is provable rather than asserted. Re-plan to close the four HIGH census gaps and the non-enforcing-verification class, then re-review.

---

## Convergence Cycle 2 Review (2026-08-15)

Source-verified against the working tree at `dc8606c`. All four cycle-1 HIGH
census gaps are resolved in PLAN text; the revision introduced **two NEW
gate-blocking defects** (one empirically reproduced) and **two NEW actionable
MEDIUM/LOW items**.

### Cycle-1 finding dispositions

| Cycle-1 finding | Disposition | Verification |
|---|---|---|
| H1 — `bug_report.yml:77` "Migrated from TypeScript CodeGraph" (PROC-01) | **RESOLVED** | bug_report.yml now in 05-02 `files_modified`, swept by task 1 (review H1); must_haves truth states bug_report is swept, not clean |
| H2 — `bench.yml` job `:96` + step `:127` display names (PROC-03) | **RESOLVED** (gate is then broken — NEW HIGH 1) | 05-03 task-2 renames both to own-terms; acceptance greps `head-to-head` = 0. Verified bench.yml:96/:127 carry the old names today, so the rename target is real |
| H3 — five `internal/query` rows (engine.go:180, explore.go:211, files.go:25, worktree_notice.go:43, render_status_test.go:132) | **RESOLVED** | all 5 files now in 05-04 `files_modified`, tasks check "VERIFIED present" |
| H4 — `internal/cli/githooks.go:15` + `install_test.go` omitted (05-05) | **RESOLVED** | both added to files_modified and task scope (review H4) |
| Retained-use inventory gate | **RESOLVED** | A3 gates in 05-04 (internal/query) and 05-05 (internal/tools/test) |
| Pr-policy non-enforcing (`--check \|\| true`) | **RESOLVED** | 05-02 task-2 drives the real env interface (no `--check`, no `|| true`) |
| 05-06 regen gate printed count without asserting | **RESOLVED** | 05-06 verify now asserts the exact allowed-path set, CASES forbidden |
| JS-edit phrasing ("no `run:` changes" vs `with.script`) | **RESOLVED** | 05-03 truth now correctly distinguishes `with.script` action scripts, validated with `node --check` (A4) |
| `.claude/CLAUDE.md` core-value + "One-way migration" line (D-10/D-04) | **RESOLVED** | 05-05 task-3 sweeps the migrate contract, the parity-command-surface, and the modernc rows |

### NEW HIGH concern 1 — 05-03 task-2 automated bench verify can never pass (outer-shell quoting bug)

The rewritten verify reads:

```
bash -c "set -e; c=$(rg -n -e 'head-to-head' -e 'vs TS' ... .github/workflows/bench.yml | wc -l | tr -d ' '); echo 'bench framing terms left: $c'; test \"'$c'\" = '0'; actionlint .github/workflows/bench.yml; ..."
```

When GSD executes the `<automated>` content, the outer double-quoted layer is parsed by the executor's own shell **before** the inner `bash -c` runs: `$(…)` and `$c` are expanded at the OUTER level — so the inner script receives an empty `$c` and `test"''" = '0'`. Running the exact content (both raw and wrapped) against the current tree (4 hits) and against a zero-hit simulation both exit 1. This is a **gate that can never go green**, even after the head-to-head renames are done. It is a fresh defect introduced in this revision (the cycle-1 verify form — a plain pipe to `grep -qx` — was correct). Fix: single-quote the outer and `test "$c" = "0"` (or keep the `… | grep -qx "0"` pipe).

### NEW HIGH concern 2 — 05-05 (wave 1) task-3 verify scans `internal/indexer` including files owned by 05-04 (wave 2); cannot pass in wave order

05-05's task-3 `rg -c -e "Go-parity" -e "golden-parity" ... internal/indexer` includes `internal/indexer/capability/matrix.go:18/:32` and `matrix_test.go:217/:221`, which still carry `golden-parity` comments today. The plan itself says those are "handled by 05-04" (wave-2 plan). Because wave 1 (05-05) runs before wave 2 (05-04), the verify as written **always sees n ≥ 4 and fails**. Options: scope 05-05's scan to its own file set (e.g. `--glob '!**/capability/**'` or explicit file list) or move the phase-wide allowed-ownership match into the wave-2 plan. As written it cannot pass.

### NEW actionable MEDIUM/LOW

1. **(MEDIUM) 05-05's `rg -n 'golden_parity_test.go|the original type project'` grooming typo**: acceptance criterion says `...|the original type project` — the origin phrase in manifest.go is "the original TS CodeGraph project…". The automated gate uses the correct phrase but the acceptance line has a typo (`type` vs. correct product name). Cosmetic wording, but it is a cross-doc inconsistency.
2. **(MEDIUM) 05-03 JS-validation gate depends on PyYAML not declared.** The heredoc does `python3 - … <<PYEOF` importing `yaml`. On hosts without PyYAML (absent on this host), python writes nothing, the `*.mjs` loop iterates a literal glob, and `node --check` fails spuriously. Either declare PyYAML in `user_setup` or avoid the runtime dep (grep-based extraction).
3. **(LOW) 05-01 task-2 `<name>` still reads "… and 8 fakes in one diff"** while the action/truth now correctly say 7 method stubs — the title was not brought into line with the count correction.
4. **(LOW) 05-02 `CHANGED_FILES=$(printf "%.docs/x")` yields `ocs/x`, not a valid path** — the policy gate runs with a non-doc filename; the "docs-only change" exemption annotates verification intent but the actual string does not match any EXEMPT prefix. Low impact (the verify still passes on the template-body headings), but the analyze is off.

### Cycle-2 Convergence verdict

CYCLE_SUMMARY: current_high=2 current_actionable=4

**Verdict: NOT CONVERGED on gate defects — request PLANNED corrections.** The
four cycle-1 HIGH census gaps are all genuinely closed in the PLAN text, and the
verification-class fixes (policy env-drive, regen assertion, JS `with.script`
description, retained-use inventory gates, REQUIREMENTS.md row) are properly
reflected. But the revision's own written verify blocks introduce: (1) a NEW
never-green gate in 05-03 task-2 shown by shell-quoting (the `bash -c "…"` +
`$(…)`/`$c` expands in the outer shell); and (2) a NEW cross-wave ownership
mismatch — 05-05 (wave 1) scans `internal/indexer` including capability/matrix
rows that are authored by 05-04 (wave 2), so its task-3 verify cannot go green
in wave order. Both would hard-fail `/gsd-execute-phase`. Actionables: declare
PyYAML for the JS node-check gate, and fix the "8 fakes" title, the
`CHANGED_FILES="%.docs/x"` printf bug, and the acceptance-criteria typo
(`the original type project`). Re-plan those two verify blocks, then re-review.

---

## Convergence Cycle 3 Review (2026-08-15)

Source-verified and empirically reproduced against the working tree at `4ab8670`,
the cycle-2 fix commit. Both cycle-2 HIGHs (bench-verify quoting, indexer
wave-scoping) are genuinely closed, and all four cycle-2 actionables
(PyYAML-free JS check, "7 fakes" title, literal `docs/RELEASE.md`, manifest
phrase typo) landed — but the revision **introduces one NEW never-green verify
gate and leaves one pre-existing sibling gate unflagged**, plus one resolves
wrongly-deemed verification.

### Cycle-2 finding dispositions

| Cycle-2 finding | Disposition | Verification |
|---|---|---|
| HIGH 1 — 05-03 task-2 bench verify quoting (`bash -c "…"` + `$(…)`/`$c` outer expansion) | **RESOLVED** | verify is now `bash -c 'set -e; rg … \| wc -l \| tr -d " " \| grep -qx "0"; actionlint …'` — a pure pipeline, single-quoted, no shell variable. Empirically: simulated post-rename tree → pipeline GREEN; current tree (4 hits) → correctly RED (renames land first). |
| HIGH 2 — 05-05 task-3 wave-1 scan includes capability/matrix rows owned by wave-2 05-04 | **RESOLVED** | verify now scans `… internal/indexer … --glob "!internal/indexer/capability/**"`, and the acceptance criterion names the H2-2 scoping. Empirically: the glob drops matrix.go:2 + matrix_test.go:2 hits; all 7 remaining non-capability hit-lines (record.go:104, graph.proto/pb.go Go-parity rows, markdown_test.go:28, pyextract/types.go:74, languages_python.go:39, phpextract/resolution_test.go:159) are each inside 05-05's own task-3 edit set, so the gate reaches 0 at its wave. |
| Actionable — PyYAML-free JS check (05-03 task-1) | **FIXED in intent, BUT the fix is itself a broken never-green gate — NEW HIGH 1** | `import yaml` is gone, but the replacement Python heredoc introduces `'` into the single-quoted `bash -c '…'` wrapper (see below). |
| Actionable — "8 fakes" → 05-01 task-2 title | **RESOLVED** | `<name>` reads "… and the 7 fakes in one diff". |
| Actionable — `CHANGED_FILES="%.docs/x"` printf | **RESOLVED** | task-2 uses the literal `CHANGED_FILES="docs/RELEASE.md"`; task-3 omits it (script defaults to empty, ok). |
| Actionable — "the original type project" typo | **RESOLVED** | action/acceptance now read "the original TS CodeGraph project". |

### NEW HIGH concern 1 — 05-03 task-1 JS-validation gate can never pass (single-quote-in-single-quote, introduced this revision)

The rewritten verify wraps the whole Python extractor in a single-quoted
`bash -c '…'`, and the Python body calls `.replace('.yml','')` (lines 119 and
123). Because the executor's shell parses the `<automated>` content before the
inner `bash -c` runs (the same mechanism the cycle-2 HIGH-1 established), the
`'` characters inside `replace('.yml','')` terminate the outer single-quoted
string. Reproduced by tracing the verbatim block: the outer shell hands the
inner `bash -c` `replace(.yml,)` (quotes eaten). Running that in a staged,
fully-post-reword tree (0 framing hits) gives
`SyntaxError: f-string: expecting '=', or '!', or ':', or '}'` then
`extractor failed` then `exit 1`. **The gate is never green even after the
reword lands.** Introduced by commit `4ab8670` ("PyYAML-free JS check"). Fix:
use double-quoted literals in the Python (`replace(".yml","")`) — double quotes
are safe inside the single-quoted span — or move the extractor out of the
nested `bash -c '…'` entirely.

### NEW HIGH concern 2 — 05-04 task-2 verify gate is a shell syntax error and can never pass (pre-existing, unflagged in cycles 1-2)

05-04 task-2's verify contains the pattern `-e "TS's"` inside a single-quoted
`bash -c '…'`. The apostrophe in `"TS's"` terminates the outer single-quoted
span, producing `unexpected EOF while looking for matching "` — the whole gate
fails to parse and never runs the intended check. Reproduced by `bash -x` on
the verbatim block. This gate is present unchanged since `5e03a92` (cycle-2
line 160) and both prior cycles missed it (they attributed the quoting defect
only to 05-03 task-2). It is a gate that can never go green for
`/gsd-execute-phase`. Fix: replace `-e "TS's"` with a regex that avoids an
apostrophe in a single-quoted span (e.g. `-e "TS.s"`), or re-quote the span.

### NEW actionable MEDIUM — 05-02 pr-template-policy gate asserts an invariant that the script documents as always true

05-02 tasks 2 and 3 run
`PR_BODY="…" AUTHOR_ASSOCIATION=OWNER CHANGED_FILES="docs/RELEASE.md" python3 scripts/pr_template_policy.py >/dev/null 2>&1 || { … exit 1; }`
and claim this "proves the format gate … enforced." But `scripts/pr_template_policy.py:14-16`
states "Exit status is always 0 — the workflow decides what to do with the
verdict." Confirmed empirically: a `close` verdict (an untrusted author with
no matching template headings) still returns exit 0, and the verdict is written
to `$GITHUB_OUTPUT` (unset here) or stderr (discarded by `>/dev/null 2>&1`).
So the gate is a tautology that never fails and verifies nothing about the
templates. The cycle-2 "RESOLVED" disposition (env-drive with no `--check`/no
`|| true`) was based on asserting an exit code that is invariant. Fix: capture
the verdict — `GITHUB_OUTPUT=$(mktemp) … python3 …` then `grep -q 'action="?pass' "$f"`
(or grab the stderr `pass:` line), or assert the surviving TEMPLATE_SIGNALS
headings directly. The framing-only gates (`rg parity … = 0`, heading-presence)
do enforce the phase goal; only this policy gate overclaims.

### NEW actionable LOW — 05-05 must_haves truth tail is garbled

05-05 line 68's schema truth ends "… D-03's comment text … is present but no
od-parity")." — a truncated/typo'd phrase an executor must guess at while
reading the frontmatter. Cosmetic; fix the sentence tail.

### Also verified clean (no action needed)

- All six plans' remaining `<automated>` verify blocks are single-quoted with
  no stray inner apostrophes (05-01 tasks 1-3, 05-02 tasks 1/2/3 grep legs,
  05-03 tasks 2-3, 05-04 tasks 1/3, 05-05 tasks 1-3, 05-06 tasks 1-2) — the
  05-03 task-1 and 05-04 task-2 blocks above are the only two breakers found by
  the full-plan scanner.
- 05-01's `m/migration` grep tail is a no-op literal (matches nowhere) but the
  real `GetMigration|PutMigration|migrationRecordName` alternatives cover the
  symbols, so the gate is intact — not raised.
- Cycle-2's four other actionables confirmed landed in text.

### Convergence verdict

CYCLE_SUMMARY: current_high=2 current_actionable=2

**Verdict: NOT CONVERGED on gate defects — request PLANNED corrections.**
Cycle-2's HIGHs and actionables are genuinely resolved and empirically
confirmed (the bench-quoting pipeline is green on a simulated post-rename tree;
the 05-05 wave-1 gate reaches zero at its wave). But two sibling never-green
verify gates of the same shell-spanning class remain in these plans at
`4ab8670` and would hard-fail `/gsd-execute-phase`: 05-03 task-1
(its PyYAML-free rewrite now terminates the single-quoted `bash -c` via
`replace('.yml','')`, reproduced as `replace(.yml,)` → Python SyntaxError →
`exit 1` even post-edit) and 05-04 task-2 (pattern `-e "TS's"` breaks the
single-quoted span → `unexpected EOF`, never parses; uncaught since cycle 2).
One verification overclaims: 05-02's policy gate asserts an exit code the script
documents as invariant (always 0), so it proves nothing about the verdict.
Re-quote the two verify blocks (double-quoted literals / apostrophe-free
regex) and make the policy gate capture `action=pass`, then re-review.
