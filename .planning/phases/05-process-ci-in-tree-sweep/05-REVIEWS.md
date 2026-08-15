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
