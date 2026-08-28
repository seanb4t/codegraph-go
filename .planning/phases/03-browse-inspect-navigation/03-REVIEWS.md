---
phase: 3
reviewers: [codex, claude]
reviewed_at: 2026-08-28T15:36:35Z
plans_reviewed: [03-01-PLAN.md, 03-02-PLAN.md, 03-03-PLAN.md, 03-04-PLAN.md, 03-05-PLAN.md, 03-06-PLAN.md, 03-07-PLAN.md, 03-08-PLAN.md, 03-09-PLAN.md, 03-10-PLAN.md]
models:
  codex: "gpt-5.6-sol (reasoning=low)"
  claude: "unknown"
model_sources:
  codex: "banner"
  claude: "unknown"
---

# Cross-AI Plan Review — Phase 3

Both lanes received the same source-grounding prompt, the three pre-approved
deviations (D-19 14/13 language count, D-11 `goto()` over shallow routing,
SRV-05 confinement reuse), and the primary directive to mentally EXECUTE every
`<verify>` block against World A (implemented) and World B (not implemented).
Both lanes returned `file:line`-cited, repo-grounded reviews; neither carries a
`[reviewed-without-repo-access]` or `[reviewed-without-source-citations]` marker.

## Codex Review

# Cross-AI Plan Review

## Overall assessment

The phase is architecturally coherent and unusually evidence-conscious. The plans correctly reuse the existing query and transport seams, distinguish successful degraded status responses from Connect errors, preserve URL state as the source of truth, and treat generated/browser artifacts as supply-chain inputs.

However, I would not approve the plan set unchanged. Four `<verify>` blocks are vacuous against the task they claim to verify, one test-harness step is not implementable as written, and the permalink plan contains contradictory error semantics. There are also several medium-risk integration gaps around status refresh, history, and cross-wave generated artifacts.

Overall risk: **HIGH until the verification and permalink-contract issues are corrected; MEDIUM afterward.**

---

## Plan 03-01 — JavaScript test harness

### Summary

Good foundation and correct ordering, but the proposed “inline Svelte component” test is not implementable in the listed `.ts` file without an actual compiled component fixture.

### Strengths

- Keeping Vitest configuration in `vite.config.ts` is consistent with the build’s source-digest enumeration. The current build configuration already lives there, including adapter configuration ([web/vite.config.ts](/Volumes/Code/github.com/seanb4t/codegraph-go/web/vite.config.ts:14)).
- The plan correctly recognizes that no JS test command exists today: current scripts stop at `check:watch` ([web/package.json](/Volumes/Code/github.com/seanb4t/codegraph-go/web/package.json:7)).
- Both automated verifications are non-vacuous:
  - Task 2 requires Vitest success plus at least one reported passing test.
  - Task 3 calls a target that does not exist before implementation, so an unimplemented task fails.
- Adding the test target to the existing CI job is compatible with the repository’s task-driven workflow convention.

### Concerns

- **HIGH — The harness component test is infeasible as specified.** Task 2 says `web/tests/harness.test.ts` should “render a trivial inline Svelte 5 component.” Testing Library’s `render` expects a compiled component; ordinary TypeScript cannot declare Svelte markup inline. No `.svelte` fixture is listed in `files_modified` or artifacts.
- **MEDIUM — The verification parses human-formatted Vitest output.**  
  Exact command:
  ```sh
  grep -Eo 'Tests +[0-9]+ passed'
  ```
  This is non-vacuous, but brittle across reporter/version changes. The later `web:test` design already proposes the more robust JSON reporter.
- **LOW — The plan changes `pnpm-workspace.yaml` even though it may not need to.** The existing strict-build policy should only be edited if approval entries genuinely change.

### Suggestions

- Add `web/tests/fixtures/Harness.svelte` and render that component, or use an existing simple component.
- Make Task 2 verification use Vitest JSON output and assert `numTotalTests >= 1` and `numPassedTests == numTotalTests`.
- Keep the reported-test floor at one, as planned; do not turn it into a suite-size snapshot.

### Risk Assessment

**MEDIUM**, primarily due to the unusable inline-component instruction.

---

## Plan 03-02 — Path-confinement regression

### Summary

This is a strong, correctly scoped plan. It tests the missing RPC-boundary proof without creating a second confinement implementation.

### Strengths

- The traced production path is real:
  - `GetNodeDetail` calls `Engine.NodeDetail` ([internal/uiserver/handlers.go](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/uiserver/handlers.go:728)).
  - File mode reads through `buildFileNodeDetail` and `readSourceFile` ([internal/query/detail.go](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/query/detail.go:107)).
  - `readSourceFile` delegates to `resolveSourcePath` ([internal/query/node.go](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/query/node.go:81)).
- The existing confinement mechanism genuinely covers:
  - Empty paths at line 37.
  - Absolute paths at line 40.
  - `..` traversal at lines 44–46.
  - Post-symlink confinement at lines 63–75.
- Error classification reaches `connect.CodeInvalidArgument` through `mapEngineError` ([internal/uiserver/handlers.go](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/uiserver/handlers.go:101)).
- Both `<verify>` commands are non-vacuous:
  - Task 1 demands at least five named subtest passes.
  - Task 2 demands the exact named top-level pass.
- The in-repo positive control is a good application of the non-vacuity rule.

### Concerns

- **LOW — Symlink skipping is too permissive.** The plan allows skipping whenever `os.Symlink` fails. On supported Linux/macOS CI, this should normally be a hard failure; otherwise the most important WR-03 case could quietly disappear.
- **LOW — The second test duplicates fixture/server setup.** This increases runtime without materially improving isolation.

### Suggestions

- Skip the symlink case only on an explicitly unsupported platform or known permission error; fail unexpected symlink errors.
- Consider combining the host-path disclosure assertions into the first table so every refusal is tested once.

### Risk Assessment

**LOW**.

---

## Plan 03-03 — Wire-oracle ordering flake

### Summary

The evidence-first intent is excellent, but both automated verification blocks are vacuous for their respective tasks. The plan also risks changing production ordering to enforce a property the current protocol stack may not promise.

### Strengths

- The plan correctly identifies a stale claim in the source. The scenario still attributes synchronous dispatch behavior to `mark3labs v0.56.0` ([test/wireoracle/scenarios.go](/Volumes/Code/github.com/seanb4t/codegraph-go/test/wireoracle/scenarios.go:456)), although the project has migrated SDKs.
- The capture path preserves stdout arrival order: the scanner sends lines in scan order and `drainUntil` appends them in that order ([test/wireoracle/capture.go](/Volumes/Code/github.com/seanb4t/codegraph-go/test/wireoracle/capture.go:318)).
- The plan properly forbids transcript re-baselining and requires a planted content mutation.
- The production writer is already serialized by a mutex covering the underlying write and classification buffer ([internal/mcp/server.go](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/mcp/server.go:470)). That makes “harness reordered bytes” less likely and gives the investigation a concrete starting point.

### Concerns

- **HIGH — Task 1 `<verify>` passes without any investigation.**  
  Exact command:
  ```sh
  go test ./test/wireoracle/ -run TestFrozenTranscriptsMatch/toolslist-repeat ...;
  grep -Eqo -- '--- (PASS|FAIL): TestFrozenTranscriptsMatch/toolslist-repeat'
  ```
  The existing subtest already runs, and either PASS or FAIL satisfies the grep. No instrumentation, timestamps, contention reproduction, or root-cause verdict is required.
- **HIGH — Task 3 `<verify>` passes before any fix.**  
  Exact command:
  ```sh
  task test:wireoracle ... &&
  test "$(grep -Eco -- '--- PASS: TestFrozenTranscriptsMatch/' ...)" -ge 1
  ```
  The current oracle normally passes and already contains many named transcript subtests. World B—no new test or fix—passes.
- **MEDIUM — R1 may impose a non-protocol invariant.** The current `pendingWriter` only serializes actual writes; it does not promise request-ID order ([internal/mcp/server.go](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/mcp/server.go:493)). Forcing response order in production solely to preserve a transcript would need a stronger product justification.
- **MEDIUM — Task 1 lists `capture.go` as modified but the plan frontmatter omits it from `files_modified`.**

### Suggestions

Replace Task 1 verification with a new named evidence test or artifact assertion, for example:

```sh
go test ./test/wireoracle -run '^TestCaptureArrivalLedgerPreservesWireOrder$' -count=1 -v |
tee /tmp/wo.txt
test "${PIPESTATUS[0]}" -eq 0
grep -Eq -- '^--- PASS: TestCaptureArrivalLedgerPreservesWireOrder' /tmp/wo.txt
```

Replace Task 3 verification with the exact new test plus the affected scenario:

```sh
go test ./test/wireoracle \
  -run '^(TestCanonicalResponseOrder|TestPipelinedResponseOrder|TestFrozenTranscriptsMatch/toolslist-repeat)$' \
  -count=1 -v | tee /tmp/wo2.txt
test "${PIPESTATUS[0]}" -eq 0
grep -Eq -- '^--- PASS: Test(CanonicalResponseOrder|PipelinedResponseOrder)' /tmp/wo2.txt
grep -Eq -- '^    --- PASS: TestFrozenTranscriptsMatch/toolslist-repeat' /tmp/wo2.txt
```

Prefer R2 if official SDK/protocol evidence confirms response order is unspecified; do not make that selection solely from observed timing.

### Risk Assessment

**HIGH**.

---

## Plan 03-04 — Browse tracer, URL grammar, errors, highlighting

### Summary

The tracer approach is excellent, and all three verifications are non-vacuous. The largest issue is that the proposed highlighter guard parses TypeScript source text with an underspecified extraction format.

### Strengths

- The production RPC response is genuinely discriminated by `mode`; reading fields by emptiness would be wrong ([internal/uiproto/uiv1/ui.proto](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/uiproto/uiv1/ui.proto:405)).
- `SourceBlob.content` is correctly treated as bytes, not guaranteed UTF-8 ([internal/uiproto/uiv1/ui.proto](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/uiproto/uiv1/ui.proto:345)).
- The shared error mapper aligns with server behavior:
  - Not found → `CodeNotFound`.
  - Invalid input → `CodeInvalidArgument`.
  - Lock contention → typed unavailable error ([internal/uiserver/handlers.go](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/uiserver/handlers.go:105)).
- Every verification requires the newly added test identity or exact new Go tests, so none passes in an unimplemented tree.
- The single-`{@html}` restriction complements the existing CSP, whose `default-src` and `connect-src` are already restricted to self ([internal/uiserver/spa.go](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/uiserver/spa.go:99)).

### Concerns

- **MEDIUM — The Go guard depends on parsing TypeScript syntax with regex/text extraction.** Small formatting changes, computed keys, or refactoring the alias map could break or misread it. The positive count helps, but does not prove complete parsing.
- **MEDIUM — `highlightAuto` is semantically questionable for unknown indexed languages.** It may misidentify a language rather than degrade to plain escaped source. Since the expected set is guarded exactly, unknown language IDs should probably render escaped plaintext and surface a diagnostic.
- **LOW — “Unknown params are preserved” is stronger than “ignored.”** Carrying all future parameters through Phase 3 serialization is sensible, but it should be represented explicitly as an ordered multimap; converting to an object may lose repeated keys.

### Suggestions

- Export a literal `HIGHLIGHT_COVERAGE` array in TypeScript and have the Go test parse only that tightly constrained declaration.
- For unregistered languages, use an escaping-only plaintext path rather than `highlightAuto`.
- Add repeated and empty unknown-parameter cases to URL tests.

### Risk Assessment

**MEDIUM**.

---

## Plan 03-05 — `GetPermalink`

### Summary

The endpoint design is appropriate and the verifications are non-vacuous, but the plan contradicts itself about whether git failures are observable. That must be resolved before implementation.

### Strengths

- Reading the indexed commit is correct: `GetStatusResponse.commit_sha` explicitly represents the graph’s indexed commit and allows absence for pre-upgrade graphs ([internal/uiproto/uiv1/ui.proto](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/uiproto/uiv1/ui.proto:151)).
- The confinement wrapper is consistent with the established `SourceFor` delegation pattern ([internal/query/detail.go](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/query/detail.go:254)).
- Every new RPC automatically inherits the existing Connect handler, message-size limits, and Origin/Host guard ([internal/uiserver/server.go](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/uiserver/server.go:121)).
- Both automated verifications demand newly named permalink tests, and Task 3 additionally runs proto drift.
- Credential stripping, exact `github.com` matching, no fetch, and per-segment escaping are appropriate controls.

### Concerns

- **HIGH — The git error contract is internally contradictory.**
  - Artifact signature: `CommitOnRemoteTrackingBranch(...) (bool, error)`.
  - Task 2 says “every function degrades to its zero value on any failure—never to an error.”
  - Task 1 asks the maintainer to decide what happens when the check cannot execute.
  
  If errors are swallowed, the handler cannot distinguish “checked and not observed” from “could not check,” yet D-07 requires honest uncertainty.
- **MEDIUM — “Missing git” is not currently testable through `internal/gitmeta`’s existing design.** `WorktreeRoot` directly calls `exec.CommandContext` with no lookup seam ([internal/gitmeta/worktree.go](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/gitmeta/worktree.go:34)). The plan must introduce a package-local seam and ensure all new git functions use it.
- **MEDIUM — Remote reason ownership is unclear.** `RemoteGitHubRepo` returns only `(owner, repo)`, but no-link must distinguish missing remote, unsupported host, malformed remote, and missing git.
- **LOW — The plan’s “proto only additions” check uses `git diff --stat`; that does not prove deletions are absent.**

### Suggestions

Adopt an explicit result type:

```go
type RemotePresence int
const (
    RemotePresenceUnknown RemotePresence = iota
    RemotePresenceObserved
    RemotePresenceNotObserved
)
```

Return structured remote resolution:

```go
type GitHubRemote struct {
    Owner, Repo string
    Host        string
    Reason      string
}
```

Replace the additions-only assertion with:

```sh
git diff --numstat -- internal/uiproto/uiv1/ui.proto |
awk '$2 != 0 { exit 1 } END { exit NR == 0 }'
```

### Risk Assessment

**HIGH** until the error semantics are made consistent.

---

## Plan 03-06 — Search and keyboard navigation

### Summary

The search controller is well designed. The main risk is over-dependence on a registry-vendored Command primitive for cross-group keyboard behavior without a prior proof that it supports that exact interaction.

### Strengths

- The trigger split matches the wire: `ExploreGroup` carries a source blob per group ([internal/uiproto/uiv1/ui.proto](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/uiproto/uiv1/ui.proto:468)), while `Search` returns lightweight locations ([internal/uiproto/uiv1/ui.proto](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/uiproto/uiv1/ui.proto:87)).
- Abort plus monotonically increasing request identity is the correct race defense.
- All automated verifications require newly named test files, so they fail in World B.
- The `/` suppression and command-chord behavior are specified precisely.
- The blocking review of vendored source acknowledges that registry-authored files are outside lockfile auditing.

### Concerns

- **MEDIUM — Cross-group arrow navigation is assumed before the vendored implementation is known.** The plan should treat this as a spike/acceptance check immediately after vendoring, not defer discovery to Task 3.
- **MEDIUM — The checkpoint mutates the repository before approval.** That is intentional, but rollback instructions should be explicit if the human rejects a file or package.
- **LOW — Search result selection temporarily bypasses URL state.** Task 3 intentionally assigns target state directly until 03-07 replaces it. That creates an intermediate commit violating the URL-as-source-of-truth contract.

### Suggestions

- After vendoring, add a minimal keyboard test before building the controller.
- If Command does not traverse groups as required, use one flat item collection with visually grouped presentation rather than custom ARIA key handling.
- Consider landing 03-06 and 03-07 together or have 03-06 selection update the URL immediately.

### Risk Assessment

**MEDIUM**.

---

## Plan 03-07 — Node navigation and history

### Summary

The URL-as-state approach is correct and the verifications are non-vacuous. The biggest gap is that the automated suite verifies navigation options, but real back/forward correctness remains entirely manual.

### Strengths

- The plan correctly uses the response’s `mode` discriminator; `GetNodeDetailResponse` explicitly warns that wrong-mode fields produce zero values ([internal/uiproto/uiv1/ui.proto](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/uiproto/uiv1/ui.proto:405)).
- `goto` injection makes push-versus-replace semantics testable.
- All three `<verify>` blocks require new test identities.
- Passing depth/limit unchanged is consistent with the wire’s existing server-owned validation discipline.
- The route already derives active navigation from reactive `page.url` ([web/src/routes/+layout.svelte](/Volumes/Code/github.com/seanb4t/codegraph-go/web/src/routes/+layout.svelte:24)), supporting the chosen URL-driven design.

### Concerns

- **MEDIUM — Back/forward correctness is manual-only.** This is a central NAV-02 requirement, not a peripheral interaction.
- **MEDIUM — `loadBrowseTarget` and `loadBlastRadius` can race independently.** The plan discusses aborting individual loads but does not define a single navigation generation that prevents results from two different URL states being combined.
- **LOW — The plan cites `Location` for neighbor entries, but `GetNodeDetail.calls` and `called_by` are full `Node` messages** ([internal/uiproto/uiv1/ui.proto](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/uiproto/uiv1/ui.proto:437)). This does not block implementation but can cause incorrect typing assumptions.

### Suggestions

- Add a browser-level navigation test using the app’s actual router, even if limited to push → push → back.
- Give the page one navigation generation/abort controller shared by node detail and blast-radius loads.
- Correct plan typing references from `Location` to `Node` for callers/callees.

### Risk Assessment

**MEDIUM**.

---

## Plan 03-08 — References, picker, copy, permalink UI

### Summary

The plan faithfully implements the constrained click-to-definition design, but DOM decoration after syntax highlighting is fragile and needs lifecycle/idempotence rules.

### Strengths

- The storage limitation is real: the phase cannot obtain call-site spans, so text-driven matching is a defensible bounded implementation.
- DOM node construction avoids creating a second HTML-string producer.
- Every `<verify>` block requires its newly added test identity and is non-vacuous.
- The picker correctly relies on `detail_gathered`, whose wire comment explicitly distinguishes genuine empty details from ungathered candidates ([internal/uiproto/uiv1/ui.proto](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/uiproto/uiv1/ui.proto:382)).
- The truncation UI uses server-provided counts and preserves the existing bounded source design ([internal/uiproto/uiv1/ui.proto](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/uiproto/uiv1/ui.proto:353)).

### Concerns

- **MEDIUM — Decorator lifecycle is unspecified.** A Svelte rerender can destroy decorated nodes; rerunning without cleanup can double-wrap tokens or retain stale event listeners.
- **MEDIUM — Highlight.js may split a qualified identifier across multiple text nodes.** A TreeWalker tokenizing each text node independently cannot match identifiers spanning markup boundaries.
- **LOW — “All candidates the server counted” is overstated.** The server currently sends `definitions` for all matches, with detail gathering capped ([internal/uiserver/handlers.go](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/uiserver/handlers.go:699)), but the UI must still handle a future response where `total_candidates > definitions.length`.

### Suggestions

- Implement decoration as a Svelte action with teardown and idempotence tests.
- Add a test where highlight markup splits a qualified or Unicode identifier across spans.
- Phrase the picker as “all returned candidates, plus the true total,” and explicitly render omitted-count differences.

### Risk Assessment

**MEDIUM**.

---

## Plan 03-09 — Status gate and committed bundle

### Summary

The shared degrade design is correct, but Task 3’s verification is vacuous if the entire plan is absent, and navigation-driven status refresh is insufficiently wired.

### Strengths

- The status asymmetry is accurately grounded:
  - `initialized` is false in both no-index and locked-store cases ([internal/uiproto/uiv1/ui.proto](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/uiproto/uiv1/ui.proto:117)).
  - `store_exists` and `indexing_in_progress` distinguish them ([internal/uiproto/uiv1/ui.proto](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/uiproto/uiv1/ui.proto:159)).
  - `degradedStatus` populates exactly those fields ([internal/uiserver/degrade.go](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/uiserver/degrade.go:136)).
- Task 1 and Task 2 verifications require their new tests and are non-vacuous.
- The status banner belongs at layout level; the existing shell already wraps every route there ([web/src/routes/+layout.svelte](/Volumes/Code/github.com/seanb4t/codegraph-go/web/src/routes/+layout.svelte:33)).
- Existing web drift protection genuinely checks source and output counts and digests ([Taskfile.yml](/Volumes/Code/github.com/seanb4t/codegraph-go/Taskfile.yml:1014)).

### Concerns

- **HIGH — Task 3 `<verify>` passes in an entirely unimplemented Phase-3 world.**  
  Exact command:
  ```sh
  task web:build:verify &&
  task web:drift &&
  task web:test &&
  task proto:drift &&
  task test:unit
  ```
  Against the existing Phase-2 source/build pair, these can all be green. The command never asserts `status.test`, `degrade-states`, `StatusBanner`, or that the build manifest changed.
- **MEDIUM — “Fetch on navigation” has no defined router integration mechanism.** The plan exposes a notification method but does not state how `$layout.svelte` observes `page.url` changes and calls it.
- **MEDIUM — Creating the gate and observing navigation reactively can double-fetch on initial load.**
- **LOW — `git status --porcelain web/build` being empty is not proof that the correct build was committed; it only proves the working tree matches HEAD.**

### Suggestions

Use a verification that binds the rebuilt artifact to the new source and tests:

```sh
task web:build:verify &&
task web:drift &&
task web:test 2>&1 | tee /tmp/web-test.txt &&
grep -Eq 'status\\.test' /tmp/web-test.txt &&
grep -Eq 'degrade-states' /tmp/web-test.txt &&
test -f web/build/.build-manifest &&
rg -q 'StatusBanner' web/src/routes/+layout.svelte
```

Also record the manifest source digest before and after the rebuild and require it to change when Phase-3 source changed.

Specify a single initial fetch, then an effect keyed on a normalized navigation identity that skips its first execution.

### Risk Assessment

**HIGH** until Task 3 verification is corrected.

---

## Plan 03-10 — golangci-lint

### Summary

The plan appropriately treats the twice-folded lint item as real work, but the CI-wiring verification is vacuous and the tool-module strategy is likely to encounter the exact MVS conflict the repository already documents.

### Strengths

- The tool isolation concern is real and documented in the existing modfile ([go.tool-lint.mod](/Volumes/Code/github.com/seanb4t/codegraph-go/go.tool-lint.mod:1)).
- Task 1’s verification is non-vacuous: without `lint:go`, the exact target count fails.
- Task 2’s `task lint:go` verification directly propagates the linter’s result.
- Adding the CI job to `inScopeJobs` is necessary because the guard’s scope is a literal allowlist ([internal/upgrade/taskfile_shape_test.go](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/upgrade/taskfile_shape_test.go:119)).
- The plan correctly leaves the six-leg test wrapper untouched; its expected set is pinned at lines 96–107.

### Concerns

- **HIGH — Task 3 `<verify>` does not verify Task 3.**  
  Exact command:
  ```sh
  task lint:actions &&
  go test ./internal/upgrade/ -run TestWorkflowRunBodiesInvokeTask ...
  ```
  Both pass before adding the `lint-go` job because that job is absent from both the workflow and `inScopeJobs`. This is the precise silent-gap shape the plan itself describes.
- **MEDIUM — Co-locating golangci-lint with actionlint is attempted despite the modfile being explicitly dedicated to avoiding MVS conflicts.** The fallback is present, but trying the known-fragile combination first adds churn.
- **MEDIUM — Tool supply-chain legitimacy is not reviewed.** Plan 03-01 requires human approval for test dependencies, while this much larger executable dependency tree gets no corresponding review.
- **LOW — “Scratch Go file” location is unspecified.** If written outside included package paths, golangci-lint may not inspect it, making the red proof misleading.

### Suggestions

Correct Task 3 verification:

```sh
task lint:actions &&
test "$(rg -o 'task lint:go' .github/workflows/ci.yml | wc -l | tr -d ' ')" -eq 1 &&
test "$(rg -o 'JobID: \"lint-go\"' internal/upgrade/taskfile_shape_test.go | wc -l | tr -d ' ')" -eq 1 &&
go test ./internal/upgrade -run '^TestWorkflowRunBodiesInvokeTask$' -count=1 -v |
tee /tmp/wf.txt &&
test "${PIPESTATUS[0]}" -eq 0 &&
grep -Eq '^--- PASS: TestWorkflowRunBodiesInvokeTask' /tmp/wf.txt
```

Create a separate `go.tool-golangci.mod` from the outset unless a dry-run proves co-location is clean.

Plant lint violations in an existing included package file, record its original hash, and require byte-identical restoration afterward.

### Risk Assessment

**HIGH** until CI-wiring verification is fixed.

---

# Cross-plan findings

## Dependency and wave structure

### Strengths

- The critical path is sensible: harness → tracer → search → navigation → source completion → degrade/bundle.
- `03-08` depends on both the permalink server and navigation work.
- `03-09` transitively depends on the permalink proto via `03-08`, so the final build includes regenerated client code.
- The two unrelated todo plans remain off the browse critical path.

### Concerns

- **MEDIUM — Wave 2 has concurrent plans touching the generated client context.** `03-05` regenerates `web/src/lib/gen/ui_pb.ts`, while `03-04` compiles/tests against that directory. Shared-worktree concurrent execution can race.
- **MEDIUM — Several plans rely on manual UAT as the only proof of central requirements**, especially history reconstruction, real browser keyboard navigation, and permalink following.
- **MEDIUM — Performance validation is thin.** There is no planned measurement for:
  - Rendering/highlighting a source blob at the server cap.
  - Hundreds of callers/callees.
  - Search result list size and DOM cost.
- **LOW — CSP and rebinding controls are inherited correctly.** The mux wraps RPC and SPA under the same exact Host/Origin guard ([internal/uiserver/server.go](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/uiserver/server.go:127)), and CSP already limits connections to self.

## Highest-priority corrections

1. Fix the four vacuous verification blocks:
   - 03-03 Task 1.
   - 03-03 Task 3.
   - 03-09 Task 3.
   - 03-10 Task 3.
2. Resolve 03-05’s contradictory git-failure contract with a structured result.
3. Add a real `.svelte` fixture to 03-01.
4. Prevent concurrent execution of 03-04 and proto regeneration in 03-05, or isolate their worktrees.
5. Add automated browser/router coverage for NAV-02 and lifecycle/idempotence coverage for source decoration.

## Final Risk Assessment

**HIGH.** The architecture should achieve the phase goals, and most task-level tests are thoughtfully non-vacuous. The remaining high risk comes from verification commands that can report green without their tasks being implemented, plus an unresolved permalink error contract that affects a permanent RPC. Once those are corrected, the plan set falls to **MEDIUM**, driven mainly by UI integration complexity and manual-only browser-history validation.

---

## Claude Review

`★ Insight ─────────────────────────────────────`
Two of the three defects I found are only visible by *executing* the check, not reading it: a `$` that reads as a literal in shell but as an end-of-line anchor in regex, and a `--- PASS:` grep against a `go test` invocation that never passes `-v`. Both read perfectly correct on the page.
`─────────────────────────────────────────────────`

# Cross-AI Plan Review — Phase 3 (03-01 … 03-10)

Repo read at `/Volumes/Code/github.com/seanb4t/codegraph-go`, branch `gsd/v0.12.0-local-graph-ui`, commit `f11bcb9c`. Every finding below is traced to a file I opened or a command I ran.

## 1. Summary

This is a strong plan set. The factual grounding is unusually good: I spot-checked ~20 load-bearing citations (`internal/query/node.go:33-79` confinement, `internal/uiserver/handlers.go:101-133` `mapEngineError`, `handlers.go:640-650` `singleDefSourceBlob`, `SourceBlob`/`Location`/`GetStatusResponse` field numbers, `Taskfile.yml:40-51` `web_source_files()`, `internal/upgrade/taskfile_shape_test.go:100-138` wrapper legs and `inScopeJobs`, the 14 `LanguageSpec.ID` values, `.svelte-kit/tsconfig.json`'s `../tests/**/*.ts` include) and **every one held**. The wave DAG is sound and the dependency edges are real. The three pre-approved deviations are correctly applied.

The problems are concentrated in exactly the place the primary directive points at: three `<verify>`/acceptance checks are structurally incapable of doing their job — one passes in World B, one can *never* pass even in World A, and one is a negative-only regex that never matches anything. Separately, there is a real cross-wave CI consequence nobody scheduled: this phase leaves `task web:drift` red in CI from wave 1 through wave 6.

## 2. Strengths

- **The confinement design is correct and the plan resisted the obvious wrong turn.** `internal/query/node.go:33-79` implements four layers (empty, absolute, `..`-prefix, and a post-`EvalSymlinks` re-verification comparing resolved-root to resolved-path at `:65-76`). `readSourceFile` (`:84-90`) is the sole read primitive, and `handlers.go:698` reaches it via `eng.SourceFor`. 03-02 correctly concludes SRV-05's deliverable is the missing *proof*, and 03-05's `ValidateRepoRelativePath` is a delegating wrapper — the acceptance criterion `rg -o 'EvalSymlinks|filepath.Clean|HasPrefix' internal/uiserver/confinement_test.go | wc -l` returns 0 enforces that structurally.
- **The `GetNodeDetail` mode-discrimination hazard is correctly identified.** `nodeDetailToProto` (`handlers.go:672-718`) populates exactly one field group per mode, and `nodeDefinitionToProto` (`:612-623`) leaves `calls`/`called_by`/`source` at zero when `gathered` is false. 03-07 Task 2's requirement that a *single-def response with an empty calls list* still classify as single-def, and 03-08 Task 3's requirement that an *empty-calls, gathered-true* candidate carry no marker, are precisely the two cases an emptiness-inference gets wrong. That is the strongest test design in the set.
- **`web/highlight_coverage_test.go` reads the real TypeScript file rather than a Go literal** (03-04 Task 3). A hardcoded Go list compared only against `indexer.RegisteredLanguageIDs()` would pass while `highlight.ts` registered anything at all. The ≥13-registration-count floor plus the alias-map-values-must-be-registered check plus `TestHighlightCoverageComparisonDiscriminates` is three independent non-vacuity assertions on one guard. `RegisteredLanguageIDs()` genuinely returns 14 (verified: `grep -rh 'ID:\s*"' internal/indexer/languages_*.go` → 14, including `tsx` and `javascript`).
- **`task proto:drift`'s floor is understood correctly.** `Taskfile.yml:345-347` prints `compared ${nfiles}` then fails below 4 — a *floor*, not equality. 03-05 correctly asserts the count stays 4 (adding messages + one rpc to an existing `.proto` adds no generated file) and explicitly forbids moving the floor.
- **The `test` wrapper set-equality trap is caught.** `taskWrapperExpectedLegs` (`taskfile_shape_test.go:100-107`) is 6 entries compared as a sorted set; 03-01 Task 3(c) correctly refuses to add a seventh leg. Meanwhile the `lint` wrapper has *no* such guard (`Taskfile.yml`: `cmds: [vet, lint:actions]`), which 03-10 also states correctly.
- **`web/tests/` placement is deliberate and verified.** `.svelte-kit/tsconfig.json` already includes `../tests/**/*.ts`, and `web_source_files()` (`Taskfile.yml:41-50`) does *not* enumerate `web/tests` — so test edits do not churn the BLD-03 source digest. Both halves of that reasoning check out.

## 3. Concerns

### HIGH — vacuous / inverted guards

**H1. The `pushState`/`replaceState` prohibition regex can never match. `03-04-PLAN` and `03-07-PLAN`.**

```
rg -o "import \{[^}]*(pushState|replaceState)[^}]*\} from '\$app/navigation'" web/src/ | wc -l   # returns 0
```

`\$` inside double quotes yields a *literal* `$` to `rg`, and in the regex engine `$` is an end-of-anchor assertion — so `$app` can never match the text `$app`. I proved this against a file containing exactly the forbidden import:

```
$ cat /tmp/rgt/a.ts
import { pushState, replaceState } from '$app/navigation';
$ rg -o "...from '\$app/navigation'" /tmp/rgt/ | wc -l
       0                      # plan's pattern — MISSES the violation
$ rg -o "...from '\\\$app/navigation'" /tmp/rgt/ | wc -l
       1                      # corrected — catches it
```

This appears **four times**: `03-04` acceptance criteria + `<verification>`, `03-07` Task 1 acceptance + `<verification>`. It is guarding the single correction this phase makes to a locked CONTEXT decision (D-11 shallow routing → `goto`), and it is the one check that would catch an executor reverting to D-11's literal wording.

Fix: escape for the regex, not just the shell —
```
rg -o "import \{[^}]*(pushState|replaceState)[^}]*\} from '\\\$app/navigation'" web/src/ | wc -l
```
and add the mandated positive companion (rule `84d1gfpywd`) — a non-zero count of `from '\$app/navigation'` imports overall, proving the search can find that module path when it is present.

**H2. `03-03-PLAN` Task 3's `<verify>` can never pass, even when everything is correct.**

```
task test:wireoracle 2>&1 | tee /tmp/wo2.txt; test "${PIPESTATUS[0]}" -eq 0 && test "$(grep -Eco -- '--- PASS: TestFrozenTranscriptsMatch/' /tmp/wo2.txt)" -ge 1
```

`test:wireoracle` is `go test ./test/wireoracle/...` with **no `-v`** (`Taskfile.yml:176`). Without `-v`, Go prints `ok  github.com/.../test/wireoracle  12.3s` and *zero* `--- PASS:` lines. The count is always 0, so `-ge 1` always fails. The danger is the obvious repair: an executor under time pressure deletes the count assertion, leaving a bare `task test:wireoracle` — which is exactly the vacuity the assertion was added to prevent.

Fix: run the pattern directly rather than through the target —
```
go test ./test/wireoracle/ -run 'TestFrozenTranscriptsMatch' -count=1 -v 2>&1 | tee /tmp/wo2.txt
```
and keep `task test:wireoracle` as a separate acceptance criterion for the full-suite green.

**H3. `03-10-PLAN` Task 3's `<verify>` is green in World B.**

```
task lint:actions && go test ./internal/upgrade/ -run 'TestWorkflowRunBodiesInvokeTask' ... && grep -Eqo -- '--- PASS: TestWorkflowRunBodiesInvokeTask'
```

`TestWorkflowRunBodiesInvokeTask` iterates only `inScopeJobs` (`taskfile_shape_test.go:129-138`, doc comment at `:1357-1358`). If Task 3 is not done at all — no `lint-go` job in `ci.yml`, no fixture entry — that test still passes, because there is nothing new for it to bind. `task lint:actions` also passes today. **The entire `<verify>` block exits 0 against an unimplemented task.** The real checks live only in the acceptance criteria (`rg -o 'task lint:go' .github/workflows/ci.yml | wc -l` returns 1, `rg -o 'JobID: "lint-go"' ... | wc -l` returns 1), which is precisely the split the primary directive warns about.

Fix: fold the two `rg` counts into the `<verify>` block itself, before the `go test`.

### MEDIUM

**M1. This phase leaves `task web:drift` red in CI for five consecutive waves.** `.github/workflows/ci.yml:164` runs `task web:drift` in the `test` job on every push. `web_source_files()` (`Taskfile.yml:41-50`) enumerates `web/package.json`, `web/vite.config.ts` and `web/src`. Plan 03-01 (wave 1) modifies `web/package.json` (5 new deps + `test` scripts) and `web/vite.config.ts` (the `test:` block). Every subsequent plan touches `web/src`. The bundle is not rebuilt until **03-09 Task 3, wave 6**. So every commit from wave 1 to wave 5 fails CI's `test` job on the source-vs-output digest mismatch. No plan acknowledges this. Either schedule a `task web:build` at the end of each web-touching plan, or state explicitly in 03-01 that `web:drift` is expected red until 03-09 and record the intended-red window — otherwise the failure will be read as noise and someone will relax the guard.

**M2. `PIPESTATUS` is a bashism, used in 8 `<verify>` blocks.** 03-01 T2, 03-02 T1/T2, 03-03 T1/T3, 03-04 T1/T3, 03-05 T2/T3, 03-10 T3. Under `sh`/`dash`, `${PIPESTATUS[0]}` expands to empty and `test "" -eq 0` errors out. It fails closed rather than vacuously, so this is a *reliability* not a *safety* defect — but it will fail in World A on a POSIX-`sh` executor. Either declare bash explicitly or restructure as `cmd > file 2>&1 || { cat file; exit 1; }`.

**M3. `03-03-PLAN` Task 2 reads an artifact that does not exist yet.** Task 2's `<read_first>` names `.planning/phases/.../03-03-SUMMARY.md` for Task 1's recorded verdict, but the plan's `<output>` says to create that SUMMARY "when done" — i.e. after Task 3. The blocking-human decision checkpoint therefore has nothing to read. Fix: have Task 1 write an interim evidence file (`03-03-EVIDENCE.md`) that Task 2 reads.

**M4. The `{@html}`-count and `innerHTML`-count invariants are asserted before the vendored components arrive.** 03-04 asserts `rg -o '\{@html' web/src/ | wc -l` returns 1; 03-08 (wave 5) re-asserts it *and* asserts `rg -o 'innerHTML|outerHTML|insertAdjacentHTML|createContextualFragment' web/src/ | wc -l` returns 0. But 03-06 (wave 3) vendors shadcn-svelte `command` source into `web/src/lib/components/ui/`. `web/src` contains none of these constructs today (verified), but Bits UI-derived component source plausibly does. If it does, 03-08 fails for a reason unrelated to its own work and the temptation is to scope the check to `web/src/lib/components/browse/`, which would silently exempt the vendored code — the exact surface D-22 says has *no* other review. Fix: scope the assertion now to `web/src` excluding `components/ui/`, and add a separate explicit review item in 03-06's checkpoint covering unescaped-HTML constructs in the vendored files.

**M5. 03-01 Task 2's vitest output parse is format-coupled.** `grep -Eo 'Tests +[0-9]+ passed'` assumes vitest's default reporter emits `Tests  N passed (N)` uncoloured. A reporter change or an unexpected TTY makes this fail in World A. Task 3 already solves this properly (JSON reporter → file → parse `numPassedTests`); Task 2 should use the same mechanism rather than a second, weaker one.

### LOW

- **L1.** 03-02's subtest-count regex `^ *--- PASS: TestGetNodeDetailPathConfinementAtRPCBoundary/[a-z_]+` will not match a subtest named with a digit or hyphen (Go converts spaces to `_`, so `positive control` → `positive_control` matches, but `in-repo control` → `in-repo_control` does not). Widen to `[a-zA-Z0-9_-]+`.
- **L2.** All ten plans carry `confidence: low` with token estimates of 45k–95k. That is 785k estimated tokens for one phase. Not wrong, but worth a maintainer sanity check before dispatch.
- **L3.** 03-05's threat register says `GetPermalink` "will NOT need `withEngine`" in `03-PATTERNS.md:93`, but the plan's Task 3(d) correctly requires `withEngine` (for the indexed commit SHA and the confinement wrapper). PATTERNS is stale on this point; the plan is right. Worth a one-line note so an executor reading PATTERNS first does not follow the wrong shape.

## 4. Suggestions

1. Fix H1/H2/H3 before dispatch — all three are one-line edits, and all three are in checks that exist specifically to be non-vacuous.
2. Add a standing "intended-red window" note to 03-01 covering M1, or insert a `task web:build` step at the end of each web-touching plan.
3. Normalize the `<verify>` idiom across all ten plans to one bash-explicit shape: materialize output to a file, check the command's own exit status directly, then assert a count. 03-01 Task 3's Taskfile guard already models this correctly; the rest should copy it rather than each inventing a pipeline.
4. Move 03-10 Task 3's two `rg` count assertions from acceptance criteria into the `<verify>` block.
5. Have 03-06's blocking checkpoint explicitly enumerate unescaped-HTML constructs in the vendored source, closing M4 at the point the files enter the repo rather than two waves later.

## 5. Risk Assessment

**MEDIUM.**

The architecture is right, the citations hold, the wave DAG is sound, and the requirement coverage is complete (14 requirements across 03-02/04/05/06/07/08/09, with 03-03 and 03-10 correctly fenced off as non-gating). Nothing here threatens the phase's five success criteria on design grounds.

The risk is entirely in verification integrity: three guards that a reader would sign off on and an executor would trust, two of which (H1, H3) return green against a completely unimplemented task. Given this milestone's own record — Phase 2 shipped six vacuous guards, three invisible to reading — shipping H1 and H3 would repeat the exact defect class the phase's own prohibitions are written to prevent. H2 is lower-consequence but higher-probability: it *will* fire, and the natural repair reintroduces vacuity.

With those three fixed and M1 acknowledged, I would rate this LOW.

---
## Consensus Summary

Two independent, repo-grounded reviewers agree the **architecture is sound and the
citations hold** — the wave DAG is real, the confinement reuse (SRV-05) is correct,
the `GetNodeDetail` mode-discrimination hazard is correctly identified, and the three
pre-approved deviations are correctly applied. Neither reviewer found a design-level
threat to the phase's success criteria.

**All the risk is concentrated in verification integrity.** Between them the two
reviewers found **seven HIGH-severity defects**, five of which are `<verify>`/guard
blocks that a reader signs off on and an executor trusts, but that do not do their
job when executed. This is the exact defect class Phase 2 shipped six instances of.

Three of the seven were independently reproduced by the orchestrator against the
working tree before this file was written (see *Orchestrator verification* below).

Codex rates the set **HIGH until the verification and permalink-contract issues are
corrected, MEDIUM afterward**. Claude rates it **MEDIUM**, dropping to **LOW** once
its three HIGHs are fixed. The spread is a severity-scale difference, not a
disagreement about the findings: neither reviewer would dispatch the set unchanged.

### Agreed Strengths

- **Confinement is right and the plan resisted the obvious wrong turn.** Both
  reviewers independently traced `internal/query/node.go:33-79` and confirmed it
  genuinely confines across all four layers (empty, absolute, `..`-prefix, and a
  post-`EvalSymlinks` resolved-root-to-resolved-path re-verification at `:65-76`).
  Both confirm 03-02's deliverable is correctly scoped as the *missing proof*, and
  03-05's `ValidateRepoRelativePath` is a genuine delegating wrapper — not a second
  implementation. The acceptance criterion forbidding `EvalSymlinks|filepath.Clean|
  HasPrefix` in the new test file enforces that structurally.
- **Mode discrimination over emptiness inference.** Both cite
  `internal/uiproto/uiv1/ui.proto:405` and `internal/uiserver/handlers.go:672-718`:
  `nodeDetailToProto` populates exactly one field group per mode, so reading fields
  by emptiness would be wrong. 03-07 Task 2 and 03-08 Task 3 test precisely the two
  cases an emptiness-inference gets wrong. Claude calls this "the strongest test
  design in the set."
- **Most task-level `<verify>` blocks ARE non-vacuous.** Both reviewers explicitly
  cleared 03-02 (both tasks), 03-04 (all three), 03-05 (both), 03-06 (all), 03-07
  (all three), 03-08 (all) and 03-01 Task 3 as requiring a newly-added test identity
  that cannot exist in World B. The defects are localized, not systemic.
- **The error mapper and transport inheritance are correctly understood.**
  `mapEngineError` (`handlers.go:101-133`) maps not-found → `CodeNotFound`, invalid →
  `CodeInvalidArgument`, lock contention → typed unavailable; every new RPC inherits
  the existing Connect handler, message-size limits and Origin/Host guard
  (`internal/uiserver/server.go:121-127`), and CSP already restricts `default-src`/
  `connect-src` to self (`internal/uiserver/spa.go:99`).
- **Build-digest reasoning checks out.** `web_source_files()` (`Taskfile.yml:41-50`)
  does not enumerate `web/tests`, which is why vitest config belongs in
  `vite.config.ts` and tests belong in `web/tests/` — both reviewers verified both
  halves. `task proto:drift`'s count-4 is a floor, not an equality, and 03-05
  correctly forbids moving it.
- **The `test` wrapper set-equality trap is caught.** `taskWrapperExpectedLegs`
  (`internal/upgrade/taskfile_shape_test.go:100-107`) is a 6-entry sorted-set
  comparison; 03-01 correctly refuses to add a seventh leg, and 03-10 correctly notes
  the `lint` wrapper has no equivalent guard.

### Agreed Concerns

**HIGH — 03-10 Task 3's `<verify>` is green against a completely unimplemented task.**
Raised by BOTH reviewers (Codex "HIGH — Task 3 does not verify Task 3"; Claude H3).
Command: `task lint:actions && go test ./internal/upgrade/ -run 'TestWorkflowRunBodiesInvokeTask' ... && grep -Eqo -- '--- PASS: TestWorkflowRunBodiesInvokeTask'`.
`TestWorkflowRunBodiesInvokeTask` iterates only `inScopeJobs`
(`taskfile_shape_test.go:129-138`), so with no `lint-go` job in `ci.yml` and no
fixture entry there is nothing new for it to bind — it passes today. `task
lint:actions` passes today. The real assertions live only in the acceptance criteria,
which is exactly the split the primary directive warns about.
**Orchestrator-confirmed:** `rg -c 'lint-go|task lint:go' .github/workflows/ci.yml
internal/upgrade/taskfile_shape_test.go` returns zero matches on the current tree.
*Fix (both reviewers converge):* fold the two `rg` count assertions
(`rg -o 'task lint:go' .github/workflows/ci.yml | wc -l` = 1 and
`rg -o 'JobID: "lint-go"' internal/upgrade/taskfile_shape_test.go | wc -l` = 1) into
the `<verify>` block itself, before the `go test`.

**HIGH — 03-03 Task 3's `<verify>` is broken, and its natural repair is vacuous.**
Both reviewers flag this block; they diagnose complementary halves of the same defect.
Command: `task test:wireoracle 2>&1 | tee /tmp/wo2.txt; test "${PIPESTATUS[0]}" -eq 0 && test "$(grep -Eco -- '--- PASS: TestFrozenTranscriptsMatch/' /tmp/wo2.txt)" -ge 1`.
- Claude (H2): `test:wireoracle` is `go test ./test/wireoracle/...` with **no `-v`**
  (`Taskfile.yml:176`). Without `-v`, Go prints one `ok ...` line and *zero* `--- PASS:`
  lines, so the count is always 0 and `-ge 1` can never pass — even in World A.
- Codex: the danger is the obvious repair — deleting the count assertion leaves a bare
  `task test:wireoracle`, which passes in World B because the oracle already passes and
  already contains many named transcript subtests.
**Orchestrator-confirmed:** `Taskfile.yml:176` is `go test ./test/wireoracle/...`,
no `-v`.
*Fix (both reviewers converge):* run the pattern directly with `-v` rather than through
the target — `go test ./test/wireoracle/ -run 'TestFrozenTranscriptsMatch' -count=1 -v`
— and keep `task test:wireoracle` as a separate acceptance criterion for full-suite green.
Codex additionally recommends binding the new named tests
(`TestCanonicalResponseOrder` / `TestPipelinedResponseOrder`) into the same block.

**MEDIUM — 03-01 Task 2's vitest output parse is format-coupled.** Both reviewers
(Codex MEDIUM; Claude M5). `grep -Eo 'Tests +[0-9]+ passed'` assumes the default
reporter's uncoloured `Tests  N passed (N)`. Non-vacuous but brittle; a reporter change
or unexpected TTY makes it fail in World A. Both note 03-01 Task 3 already solves this
correctly with the JSON reporter (`numPassedTests`) and Task 2 should use the same
mechanism rather than a second, weaker one.

**MEDIUM — central requirements rest on manual UAT only.** Both reviewers, differently
scoped. Codex: history reconstruction, real browser keyboard navigation, and permalink
following have no automated proof; back/forward correctness is manual-only despite
NAV-02 being a central requirement. Claude reaches the same conclusion via the
`web:drift` window (below). Codex suggests a browser-level push → push → back test using
the app's actual router.

**MEDIUM — performance validation is thin.** Codex only, but uncontradicted: no planned
measurement for rendering/highlighting a source blob at the server cap, hundreds of
callers/callees, or search-result list DOM cost.

### HIGH concerns raised by one reviewer (both orchestrator-verified where marked)

**HIGH (Claude H1) — the `pushState`/`replaceState` prohibition regex can NEVER match.**
`rg -o "import \{[^}]*(pushState|replaceState)[^}]*\} from '\$app/navigation'" web/src/ | wc -l`
— `\$` inside double quotes yields a literal `$` to `rg`, where `$` is an end-of-line
anchor, so `$app` can never match the text `$app`. The check returns 0 whether or not
the violation is present. It appears **four times**: `03-04-PLAN.md:268` and `:414`,
`03-07-PLAN.md:161` and `:314` — acceptance criteria *and* `<verify>` in both plans.
This is the single check guarding the phase's one correction to a locked CONTEXT
decision (D-11), i.e. the only thing that would catch an executor reverting to the
literal locked wording.
**Orchestrator-confirmed empirically:** against a file containing exactly
`import { pushState, replaceState } from '$app/navigation';`, the plan's pattern
returns `0`; the escaped form (`'\\\$app/navigation'`) returns `1`.
*Fix:* escape for the regex, not just the shell, and add the mandated positive
companion (rule `84d1gfpywd`) — a non-zero count of `from '\\\$app/navigation'` imports
overall, proving the search can find that module path when present.

**HIGH (Codex) — 03-03 Task 1's `<verify>` accepts a FAILING test and requires no
investigation.** `grep -Eqo -- '--- (PASS|FAIL): TestFrozenTranscriptsMatch/toolslist-repeat'`
— either verdict satisfies the grep, and the subtest already runs today, so the block
demands no instrumentation, no contention reproduction and no root-cause verdict.
**Orchestrator-confirmed:** the `toolslist-repeat` scenario exists today
(`test/wireoracle/oracle_test.go:608-643`), so the guard is green in World B.
*Fix:* assert a new named evidence test, e.g.
`go test ./test/wireoracle -run '^TestCaptureArrivalLedgerPreservesWireOrder$' -count=1 -v`
plus `grep -Eq -- '^--- PASS: TestCaptureArrivalLedgerPreservesWireOrder'`.

**HIGH (Codex) — 03-09 Task 3's `<verify>` never asserts 03-09's own deliverables.**
`task web:build:verify && task web:drift && task web:test && task proto:drift && task test:unit`
— every one of these is a pre-existing target. The block never asserts `status.test`,
`degrade-states`, `StatusBanner`, or that the build manifest changed. Once someone runs
`task web:build` the chain goes green regardless of whether the status gate exists.
*Fix:* bind the rebuilt artifact to the new source and tests — tee `task web:test`,
`grep -Eq 'status\.test'` and `grep -Eq 'degrade-states'`, `rg -q 'StatusBanner'
web/src/routes/+layout.svelte`, and record the manifest source digest before and after
the rebuild, requiring it to change when Phase-3 source changed.

**HIGH (Codex) — 03-05's git-failure error contract is internally contradictory.**
The artifact signature is `CommitOnRemoteTrackingBranch(...) (bool, error)`; Task 2 says
"every function degrades to its zero value on any failure — never to an error"; Task 1
asks the maintainer to decide what happens when the check cannot execute. If errors are
swallowed, the handler cannot distinguish "checked and not observed" from "could not
check" — yet D-07 requires honest uncertainty. This lands in a **permanent published RPC
shape**, so it must be resolved before Task 1's wire freeze.
*Fix:* an explicit tri-state (`RemotePresenceUnknown|Observed|NotObserved`) and a
structured `GitHubRemote{Owner, Repo, Host, Reason}` so `availability`/`reason` can be
answered honestly.

**HIGH (Codex) — 03-01 Task 2's harness component test is not implementable as written.**
Task 2 says `web/tests/harness.test.ts` should "render a trivial inline Svelte 5
component," but Testing Library's `render` expects a *compiled* component and ordinary
TypeScript cannot declare Svelte markup inline. No `.svelte` fixture appears in
`files_modified` or the artifacts table. The RED→GREEN conversion (step g) cannot be
performed as specified.
*Fix:* add `web/tests/fixtures/Harness.svelte` (and list it as an artifact), or render
an existing simple component.

### Divergent Views

- **Is `task web:drift` red between waves 1 and 6?** Claude (M1) says yes: `ci.yml:164`
  runs `task web:drift` on every push; `web_source_files()` enumerates
  `web/package.json`, `web/vite.config.ts` and `web/src`; 03-01 (wave 1) modifies the
  first two and every later plan touches `web/src`, but the bundle is not rebuilt until
  03-09 Task 3 (wave 6) — so CI's `test` job fails on the digest mismatch for five
  consecutive waves, and no plan acknowledges it. Codex reads 03-09 Task 3's chain as
  "can all be green against the existing Phase-2 source/build pair," which implicitly
  assumes drift is *not* red. **The orchestrator judges Claude correct on the mechanism**
  (`web:drift` compares committed source digest against the committed build manifest;
  03-01 changes an enumerated file without rebuilding). Codex's underlying point survives
  either way: the block never asserts 03-09's own deliverables. Resolve by either
  scheduling a `task web:build` at the end of each web-touching plan, or recording an
  explicit intended-red window in 03-01 — an unacknowledged red CI is read as noise and
  invites someone to relax the guard.
- **How severe is `PIPESTATUS`?** Claude (M2) flags it as a bashism in 8 `<verify>` blocks
  that fails *closed* under `sh`/`dash` — a reliability defect, not a safety one. Codex
  does not raise it and in fact *recommends* `PIPESTATUS` in three of its suggested fixes.
  **Orchestrator note:** the executor shell on this host resolves to `bash -lc` (bash
  5.3), where `PIPESTATUS` works; the outer wrapper is zsh, where it would not. The risk
  is real but host-dependent — worth normalizing the idiom rather than treating as urgent.
- **Overall risk rating.** Codex: HIGH until the verification and permalink-contract
  issues are fixed, then MEDIUM. Claude: MEDIUM, dropping to LOW once H1–H3 are fixed and
  M1 is acknowledged. The gap comes from Codex weighting the 03-05 contract and the
  03-01 infeasibility as blockers, and Claude weighting only the guard defects.

### Orchestrator verification

Three consensus/single-reviewer HIGHs were independently reproduced against the working
tree before this file was written, so they are not taken on a reviewer's word:

| Finding | Check run | Result |
|---|---|---|
| Claude H1 (`\$app` regex) | plan pattern vs corrected pattern against a file containing the forbidden import | plan pattern `0`, corrected `1` — **confirmed inverted** |
| Claude H2 (no `-v`) | `Taskfile.yml:176` | `go test ./test/wireoracle/...`, no `-v` — **confirmed** |
| Both, 03-10 T3 | `rg -c 'lint-go\|task lint:go' .github/workflows/ci.yml internal/upgrade/taskfile_shape_test.go` | zero matches — **confirmed green in World B** |
| Codex, 03-03 T1 | `rg -n 'toolslist-repeat' test/wireoracle/` | scenario exists at `oracle_test.go:608-643` — **confirmed green in World B** |
| SRV-05 reuse (pre-approved) | read `internal/query/node.go:33-79` | genuinely confines across all four layers — **reuse is sound** |

One further orchestrator observation not raised by either reviewer, recorded for the
planner: `resolveSourcePath` calls `filepath.EvalSymlinks(abs)`, which **errors when the
path does not exist**. `GetPermalink` is pinned to the *indexed* commit, so a permalink
request for a path that existed at that commit but has since been deleted from the
working tree would be refused as `CodeInvalidArgument`. 03-05 should state whether that
is intended.

### Ordered correction list

1. 03-04 ×2 and 03-07 ×2 — escape `\\\$app/navigation` for the regex and add the positive companion.
2. 03-10 Task 3 — fold the two `rg` count assertions into the `<verify>` block.
3. 03-03 Task 3 — run `go test ... -v` directly; keep `task test:wireoracle` as a separate criterion.
4. 03-03 Task 1 — assert a new named evidence test instead of accepting `PASS|FAIL`.
5. 03-09 Task 3 — bind the verify to `status.test`, `degrade-states`, `StatusBanner` and a changed manifest digest.
6. 03-05 Task 1 — resolve the git-failure contract to a tri-state before the wire freeze.
7. 03-01 Task 2 — add a real `.svelte` fixture.
8. 03-01 — record the intended-red `web:drift` window (or rebuild per web-touching plan).
9. Normalize the `<verify>` idiom across all ten plans to one bash-explicit shape.
