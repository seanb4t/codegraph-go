---
phase: 4
reviewers: [codex]
reviewed_at: 2026-08-29T16:57:54-04:00
plans_reviewed: [04-01-PLAN.md, 04-02-PLAN.md, 04-03-PLAN.md, 04-04-PLAN.md, 04-05-PLAN.md, 04-06-PLAN.md, 04-07-PLAN.md]
models:
  codex: "gpt-5.6-sol (reasoning=low)"
model_sources:
  codex: "banner"
---

# Cross-AI Plan Review — Phase 4

## Codex Review
### Summary

The phase is thoughtfully decomposed and has unusually strong non-vacuity, supply-chain, and error-state coverage. However, several cross-plan interfaces do not compose as written. The largest blockers are: `DataTable` is designed specifically for `Location` but Plan 05 reuses it for `CountRow`; `AnalysisPanel.run` returns only rows but Plans 04/06 require response metadata; workbench URL updates will likely retrigger the global status gate; and Plan 07’s proposed component enumeration currently returns zero files. These are plan-level defects that should be corrected before execution. Overall risk: **HIGH until replanned**, then likely MEDIUM.

### Strengths

- The backend split is well grounded. `GetStatus` is explicitly the sole degrade-and-answer exception ([internal/uiserver/handlers.go:290](internal/uiserver/handlers.go#L290)), while ordinary handlers use `withEngine` ([internal/uiserver/handlers.go:50](internal/uiserver/handlers.go#L50)). Plan 03 correctly preserves that boundary.

- The health projection is genuinely mapping existing data rather than inventing engine work. `StatusResult` already includes all required maps, counts, staleness, mismatch, and index health ([internal/query/status.go:47](internal/query/status.go#L47)), populated by three bounded scans in `Status` ([internal/query/status.go:253](internal/query/status.go#L253), [internal/query/status.go:272](internal/query/status.go#L272), [internal/query/status.go:293](internal/query/status.go#L293)).

- The four analysis RPCs already preserve server-side bounds and structured rows. `Callers`, `Impact`, and `Affected` pass inputs to the engine and map `Location` results ([internal/uiserver/handlers.go:461](internal/uiserver/handlers.go#L461), [internal/uiserver/handlers.go:506](internal/uiserver/handlers.go#L506), [internal/uiserver/handlers.go:550](internal/uiserver/handlers.go#L550)). Avoiding duplicated frontend bounds is appropriate.

- Plan 02 fixes `Files` at the shared engine seam. The existing implementation has exactly the two `filepath.Match` sites described ([internal/query/files.go:148](internal/query/files.go#L148), [internal/query/files.go:171](internal/query/files.go#L171)), so CLI, MCP, and UI receive the fix together.

- The URL grammar decision follows an established stable-order contract. Browse preserves repeated unknown parameters and stable-sorts them ([web/src/lib/browse-url.ts:70](web/src/lib/browse-url.ts#L70), [web/src/lib/browse-url.ts:89](web/src/lib/browse-url.ts#L89)); Plan 01’s separate `getAll('file')` grammar is a reasonable extension.

- The read-only RPC guard is strong and non-vacuous: it checks exact set equality in both directions ([internal/uiserver/readonly_test.go:44](internal/uiserver/readonly_test.go#L44)) and independently scans all method names for mutation verbs ([internal/uiserver/readonly_test.go:95](internal/uiserver/readonly_test.go#L95)). Plan 03 preserves both mechanisms.

- Plans consistently require behavioral assertions rather than cosmetic proxies: rendered row order for sorting, DOM order for trust warnings, both present/absent mismatch cases, exact call arguments, and RED demonstrations for artifact guards.

### Concerns

#### Plan 01 — Workbench tracer

- **MEDIUM — The plan installs a runtime-imported table package as a dev dependency without documenting why that is safe.** The project separates build-only packages under `devDependencies` and shipped application libraries under `dependencies` ([web/package.json:17](web/package.json#L17), [web/package.json:39](web/package.json#L39)). Bundling may make this safe, but `pnpm add -D` should be justified and covered by a production-mode build check.

- **MEDIUM — The tracer mixes route orchestration, URL grammar, error taxonomy, dependency intake, a new table system, and vendored-source review in one 95k-token plan.** That makes the architectural tracer harder to diagnose: a failure could come from SvelteKit route context, TanStack, URL state, or test harness setup. The plan’s own confidence is low.

- **LOW — The malformed integer grammar accepts negative values by design.** `INTEGER_SHAPE` is `^-?\d+$` ([web/src/lib/browse-url.ts:48](web/src/lib/browse-url.ts#L48)). That is consistent with server ownership of validation, but tests should explicitly cover negative depth/limit passthrough, not only overly large positive values.

#### Plan 02 — Recursive file glob

- **MEDIUM — The malformed-pattern test does not prove rejection happens before store iteration.** The current sanity check is indeed before matching ([internal/query/files.go:148](internal/query/files.go#L148)), but merely asserting the error text cannot detect a future move below `IterateFiles`. If “before any scan” is a must-have, the test needs an instrumented reader whose `IterateFiles` fails the test when called.

- **LOW — The proposed fixture may need new files rather than reusing the current corpus unchanged.** Existing `filesStatusFixture` copies the shared fixture ([internal/query/files_status_test.go:16](internal/query/files_status_test.go#L16)). The plan should explicitly verify that corpus contains the required root-level, two-directory-deep, and mixed-extension names before committing the RED expectations.

#### Plan 03 — GetHealth RPC

- **MEDIUM — The test plan says to reuse a helper from another package’s `_test.go`, which is impossible directly.** `worktreeMismatchFixture` is package-private test code in `internal/query` ([internal/query/engine_worktree_test.go:72](internal/query/engine_worktree_test.go#L72)). `internal/uiserver/health_test.go` cannot import it. The plan needs either a uiserver-local fixture or a shared exported/internal test helper.

- **LOW — `GetHealthRequest.path` is called “RESERVED,” but it is an active wire field.** The existing precedent uses the same misleading wording while declaring `string path = 1` ([internal/uiproto/uiv1/ui.proto:107](internal/uiproto/uiv1/ui.proto#L107)). In protobuf terminology, this is ignored/reserved-for-future-use, not a `reserved` field. Tighten the comment before duplicating the ambiguity.

- **LOW — The checkpoint freezes five new messages although several names are broad enough to collide conceptually with existing domain types.** `PendingChanges` and `IndexHealth` already exist as Go domain types ([internal/query/status.go:67](internal/query/status.go#L67), [internal/query/status.go:76](internal/query/status.go#L76)). Names such as `HealthPendingChanges` would reduce generated-code ambiguity, though this is primarily a naming consideration.

#### Plan 04 — Tabs, Impact, Callers, Callees

- **HIGH — `AnalysisPanel.run` discards metadata that the plan requires the UI to render.** The proposed contract is `run: (...) => Promise<Location[]>`, but `ImpactResponse` separately carries `node_count` and `edge_count` ([internal/uiproto/uiv1/ui.proto:290](internal/uiproto/uiv1/ui.proto#L290)). A summary snippet cannot reactively obtain those values through the declared interface without closure side effects or duplicated route state. Plan 06 has the same problem with `AffectedResponse.files` ([internal/uiproto/uiv1/ui.proto:312](internal/uiproto/uiv1/ui.proto#L312)).

- **HIGH — `replaceState` does not by itself preserve the “one GetStatus per navigation” behavior claimed by the plan.** The layout reacts to every change in `page.url` and calls `notifyNavigated` ([web/src/routes/+layout.svelte:37](web/src/routes/+layout.svelte#L37)). `navigationIdentity` excludes only `q`; it retains `mode`, `symbol`, `depth`, `limit`, and `file` ([web/src/lib/status.ts:64](web/src/lib/status.ts#L64)). Therefore workbench control updates can mint new status identities and trigger additional `GetStatus` calls even if `goto` is never invoked. The mount-count and zero-`goto` tests do not detect this.

- **MEDIUM — The proposed test’s zero-`goto` assertion proves only that one API was not called.** It does not prove no navigation lifecycle occurred or that the global status gate did not refetch. The test should observe `statusGate.notifyNavigated` or the shared status client’s call count.

#### Plan 05 — Health view

- **HIGH — `CountTable` cannot reuse the `DataTable` contract defined by Plan 01.** Plan 01 fixes `DataTable` to `rows: Location[]`, and its row ID dereferences `filePath`, `startLine`, and `name`. Those fields are defined by the `Location` schema ([internal/uiproto/uiv1/ui.proto:95](internal/uiproto/uiv1/ui.proto#L95)). Plan 05’s `CountRow` has only `{key, count}`. This will fail type checking or produce invalid identities.

- **MEDIUM — The view’s “freshness” sources can disagree without a display rule.** The verdict comes from the shared `GetStatus` gate, while commit SHA, version, and reindex recommendation come from a later `GetHealth` call. The gate fetches on construction and navigation ([web/src/lib/status.ts:116](web/src/lib/status.ts#L116)); `GetHealth` is a separate store snapshot. During reindexing, those two snapshots may legitimately disagree. Reusing `classifyStatus` is correct, but the UI needs an explicit rule for mixed-snapshot presentation.

- **LOW — Reusing `describeWorkbenchFailure` for health creates a domain naming leak.** It may be mechanically correct, but the helper’s name and taxonomy were designed around query failures. Consider a neutral shared `describeRpcFailure` composition layer rather than making `/health` depend conceptually on Workbench.

#### Plan 06 — Multi-file Affected

- **HIGH — The plan simultaneously requires a new controller implementation and prohibits one.** Task 1 directs `file-search.ts` to create its own timer, `AbortController`, monotonic request ID, state store, and error handling while the plan says it “MUST NOT write a second debounce or a second AbortController/request-identity implementation.” The existing implementation already owns those mechanics at [web/src/lib/search.ts:129](web/src/lib/search.ts#L129) through [web/src/lib/search.ts:231](web/src/lib/search.ts#L231). Importing two constants is not reuse of the controller mechanism.

- **HIGH — File-search metacharacters are not escaped.** The plan interpolates user text directly into `**/*${term}*`. Because the API intentionally supports glob syntax, a filename search for `[`, `{`, `*`, `?`, or `\\` can become malformed or behave as a glob expression rather than a literal substring. The existing server rejects malformed patterns, but that turns ordinary filename characters into user-visible errors.

- **HIGH — Affected inherits the `AnalysisPanel` metadata mismatch.** `AffectedResponse` carries both echoed `files` and `affected_tests` ([internal/uiproto/uiv1/ui.proto:312](internal/uiproto/uiv1/ui.proto#L312)), while the proposed panel accepts only `Promise<Location[]>`. Rendering the server-echoed files above the table requires widening the result contract.

- **MEDIUM — The test expecting “two aborted signals” for three rapid keystrokes conflicts with a true debounce.** Before the debounce fires, earlier keystrokes should create no RPC and therefore no `AbortSignal`. The existing controller creates an abort controller only when dispatch actually begins ([web/src/lib/search.ts:143](web/src/lib/search.ts#L143)), while rapid pre-dispatch keystrokes merely clear timers ([web/src/lib/search.ts:207](web/src/lib/search.ts#L207)). That expected assertion should distinguish cleared timers from aborted in-flight requests.

#### Plan 07 — Drift, measurement, bundle refresh

- **HIGH — The proposed disk enumeration is known to be empty.** Running the exact suggested command now, `git ls-files -- 'web/src/lib/components/ui/*/'`, returns zero files. Git tracks files, not directory entries. The target must enumerate `web/src/lib/components/ui/**/*` or all files under that prefix and derive the first component path segment.

- **HIGH — The target is described as “release-path” but the plan explicitly wires it nowhere.** Existing frontend gates are invoked explicitly from CI ([.github/workflows/ci.yml:149](.github/workflows/ci.yml#L149), [.github/workflows/ci.yml:163](.github/workflows/ci.yml#L163), [.github/workflows/ci.yml:186](.github/workflows/ci.yml#L186)). Merely adding an unreferenced Taskfile target makes it manual/local, not release-path. The plan needs a non-PR release workflow hook or a dependency of an existing release task.

- **HIGH — A wall-clock performance threshold should not become a normal Vitest pass/fail test.** `task web:test` runs every Vitest test indiscriminately ([Taskfile.yml:567](Taskfile.yml#L567), [Taskfile.yml:592](Taskfile.yml#L592)). A fixed 400ms/200ms jsdom threshold will vary with CI load, architecture, coverage/instrumentation, and accumulated DOM. This risks creating the same nondeterministic gate behavior the plan explicitly rejects elsewhere.

- **MEDIUM — The drift floor is specified as the currently observed full count despite the repository’s established rule that floors are small structural minima.** `web:test` states its floor is only for zero-test detection and must not track growth ([Taskfile.yml:577](Taskfile.yml#L577)); `web:drift` similarly uses a stable structural lower bound ([Taskfile.yml:1123](Taskfile.yml#L1123)). Using all currently observed component files as the floor means intentional removal requires updating a brittle historical count.

- **MEDIUM — One CLI version may not reproduce every previously vendored component.** The current tree already contains six component families. Task 1’s determinism branch appropriately detects this, but Task 2 assumes a single `1.5.1` regeneration command can authoritatively cover all of them. If older components came from another registry revision, stopping the plan is more likely than the objective suggests.

### Suggestions

- Redesign `DataTable` as a generic component with an explicit `getRowId` prop, or create a separate generic table core plus thin `LocationTable` and `CountTable` wrappers.

- Change `AnalysisPanel` to accept a typed result object, for example `{ rows, summary }`, rather than only `Location[]`. Impact can return `{rows: affected, summary: {nodeCount, edgeCount}}`; Affected can return `{rows: affectedTests, summary: {files}}`.

- Extend `navigationIdentity` intentionally for Workbench-local parameters, or provide a route-aware identity function. Add a test proving depth/limit/file edits do not cause another `GetStatus` call.

- Extract a reusable debounced RPC controller from `search.ts` and configure it for symbol search and file search. This would satisfy D-15 structurally instead of merely sharing constants.

- Escape user input before embedding it in a glob. Preserve `**/*…*` as the structural pattern while treating the typed term literally.

- Replace Plan 07’s enumeration with tracked-file enumeration under `web/src/lib/components/ui/`, then derive unique component directories. Test the enumeration itself before invoking the network CLI.

- Wire `web:components:drift` into an explicitly named release-only task or the release workflow—not PR CI—if “release-path” is a requirement.

- Separate performance measurement from the regular test suite. Use a dedicated task that records medians and applies the decision threshold during this phase, while retaining only deterministic structural/regression assertions in Vitest.

- Split Plan 01 into dependency/vendor intake and the actual Callers tracer. This reduces the failure surface of the highest-risk frontend integration.

### Risk Assessment

**Overall risk: HIGH.**

The backend changes in Plans 02 and 03 are generally executable with modest corrections. The primary risk is frontend composition: Plans 04–06 rely on component contracts that cannot carry the data they promise to render, and workbench URL changes interact incorrectly with the existing global status identity mechanism. Plan 07 additionally contains a demonstrably empty enumeration command, lacks an actual release-path integration, and would introduce a timing-sensitive test into the required frontend suite. Correcting those contracts and gate mechanics before execution should reduce the phase to MEDIUM risk.

---

## Consensus Summary

One prompt-fed, source-grounded reviewer ran this cycle (Codex, `gpt-5.6-sol`,
reasoning=low). It cited concrete `file:line` evidence throughout, so its findings carry
full weight; there is no second lane to form a cross-model consensus against, so the
orchestrator independently re-verified every HIGH against the tree before recording it
(see **Verification coverage** below). Two Codex findings were downgraded on that
evidence, and one HIGH the reviewer did not raise was added.

**Verdict: HIGH risk until replanned.** The backend plans (04-02, 04-03) are executable
with modest corrections. The frontend plans do not compose: three shared component
contracts (`DataTable`, `AnalysisPanel`, the file-search controller) are declared with
types and prohibitions that the call sites in later plans cannot satisfy, and two guard
mechanisms (04-07's enumeration, the ten frontend count floors) are vacuous as written.

### Agreed Strengths

Single-reviewer cycle — these are Codex's, each re-checked against the tree by the
orchestrator and confirmed:

- The backend seam split is correct. `GetStatus` is the documented sole degrade-and-answer
  exception (`internal/uiserver/handlers.go:290`) and 04-03 explicitly refuses to copy that
  shape, routing `GetHealth` through `withEngine` (`internal/uiserver/handlers.go:50`).
- 04-03 maps existing data rather than inventing engine work: `StatusResult` already carries
  every map, count, staleness flag, mismatch and health block HLT-01 names
  (`internal/query/status.go:47`).
- 04-02 fixes `Files` at the one shared seam — the two `filepath.Match` call sites are
  exactly where the plan says (`internal/query/files.go:148`, `:171`), so CLI, MCP and UI
  are fixed together.
- The read-only rpc guards are genuinely non-vacuous in both directions
  (`internal/uiserver/readonly_test.go:44`, `:95`) and 04-03 preserves both mechanisms
  rather than relaxing either.
- Plans consistently demand behavioural assertions over cosmetic proxies: rendered row
  order for sorting, `compareDocumentPosition` for the trust-verdict ordering, both the
  present AND absent worktree cases, and RED demonstrations for every new guard.

### Agreed Concerns

Ordered by severity. Every HIGH below was re-verified in-tree this session.

**HIGH — `AnalysisPanel`'s result contract cannot carry what its `summary` snippet must
render.** 04-04 declares `run: (signal: AbortSignal) => Promise<Location[]>`
(`04-04-PLAN.md:224`) yet requires Impact's `nodeCount`/`edgeCount` to render in a header
summary (`04-04-PLAN.md:32`, `:188-192`, `:243`); 04-06 requires the same panel to render
`AffectedResponse`'s echoed `files` (`04-06-PLAN.md:283-286`). Neither value survives the
declared return type. As written the route must either issue a second request or reach the
value through a closure side effect. **Fix:** widen the contract to a typed result object
(`{rows, summary}`) in 04-04 before 04-06 depends on it.

**HIGH — `DataTable` is typed to `Location` but 04-05 reuses it for `CountRow`.** 04-01
fixes the props at `rows: Location[]` with `getRowId: (row) => ${row.filePath}:${row.startLine}:${row.name}`
(`04-01-PLAN.md:350-353`). 04-05's `CountTable` is specified as "a thin wrapper over
`DataTable`" over `CountRow{key,count}` (`04-05-PLAN.md:215-216`, `:122`), and 04-05's own
key_link asserts the count tables "render through the ONE DataTable shell"
(`04-05-PLAN.md:41`). Those cannot both hold: `pnpm check` — which 04-05's Task 2 verify
requires to exit 0 — will reject it. **Fix:** make `DataTable` generic with an explicit
`getRowId` prop in 04-01, or split a generic core from a `LocationTable` wrapper.

**HIGH — Workbench URL writes will refire `GetStatus` on every control change.** The layout
calls `statusGate.notifyNavigated(navigationIdentity(page.url))` on every `page.url` change
(`web/src/routes/+layout.svelte:47-48`), and `navigationIdentity` strips only
`VIEW_LOCAL_PARAMS = ['q']` (`web/src/lib/status.ts:74`, `:81-88`). Every Workbench
parameter — `mode`, `symbol`, `depth`, `limit`, `file` — therefore mints a new navigation
identity, so `replaceState` from a depth slider fires an extra `GetStatus` RPC per move.
This is the exact defect the `q` exclusion was added to fix, reintroduced by a different
route. 04-04's proposed proofs (route-not-remounted, `goto` never called) cannot detect it.
**Verified unaddressed:** `rg 'navigationIdentity|VIEW_LOCAL_PARAMS|notifyNavigated'` across
all of `.planning/phases/04-query-workbench-index-health/` returns zero matches — no plan,
context, or research note mentions it. **Fix:** extend `VIEW_LOCAL_PARAMS` (or add a
route-aware identity) and assert the `GetStatus` call count across a depth/limit/file edit.

**HIGH — 04-06 Task 1 both forbids and mandates a second search controller.** The plan
prohibits "a second debounce or a second `AbortController`/request-identity implementation
(D-15)" (`04-06-PLAN.md:44`) and its key_link claims `file-search.ts` "reuses
`web/src/lib/search.ts`'s debounce, `AbortController` and monotonic request-identity
contract rather than implementing a second one" (`04-06-PLAN.md:39`). Task 1's action then
directs exactly that second implementation, reusing only two exported constants
(`04-06-PLAN.md:141-150`). `createSearchController` is not parameterisable for `Files` —
it owns two symbol-specific RPCs (`web/src/lib/search.ts:129-231`). **Fix:** extract a
reusable debounced-RPC controller from `search.ts` and configure it twice, or restate the
prohibition to match what the plan actually asks for.

**HIGH — 04-06's "two aborted `AbortSignal`s" assertion is unsatisfiable against the
mechanism it reuses.** The behaviour block requires "Three keystrokes in rapid succession
produce ONE dispatched request, and the two aborted `AbortSignal`s report `aborted === true`"
(`04-06-PLAN.md:127-128`). In `search.ts` the `AbortController` is created only inside
`dispatchLive` (`web/src/lib/search.ts:143-145`); pre-dispatch keystrokes merely clear the
debounce timer (`web/src/lib/search.ts:207-211`). One dispatch means zero aborted signals,
not two. Written as specified this RED test can never go green. **Fix:** assert cleared
timers for the pre-dispatch case and reserve the abort assertion for genuinely overlapping
in-flight dispatches.

**HIGH — 04-06 interpolates the raw search term into a glob with no escaping.** Task 1
builds `{ pattern: '**/*' + term + '*' }` (`04-06-PLAN.md:151`). `Engine.Files` treats the
pattern as glob syntax, so a typed `[`, `{`, `*`, `?` or `\` either errors through
`query: invalid pattern %q` (`internal/query/files.go:148-149`) or silently matches
something other than the substring the user typed. Ordinary filename characters become
user-visible errors. The plan's own prohibition against "a client-side pattern rewrite to
work around glob behavior" (`04-06-PLAN.md:47`) is likely to be read as forbidding the fix.
**Fix:** escape the term's metacharacters while keeping `**/*…*` structural, and carve that
explicitly out of prohibition 47.

**HIGH — 04-07's enumeration command returns zero files in this repository, making its
floor vacuous.** The plan derives the subject set with
`git ls-files -- 'web/src/lib/components/ui/*/'` (`04-07-PLAN.md:179`) and its acceptance
criterion verifies the printed count against that same command (`04-07-PLAN.md:237`).
Executed in-tree this session that command returns **0**; without the trailing slash it
returns **36**. Git tracks files, not directory entries. A floor derived from and checked
against a zero-returning enumeration is precisely the vacuous pass T-04-29
(`04-07-PLAN.md:341`) exists to prevent. **Fix:** enumerate
`web/src/lib/components/ui/**/*` (or the prefix without the trailing slash), derive the
component directories from the first path segment, and test the enumeration itself before
the target invokes the network CLI.

**HIGH — 04-07 puts a wall-clock threshold inside a required PR check.** Task 3 fixes
"median initial render > 400ms OR median sort toggle > 200ms" and instructs "Assert the
thresholds so a future regression fails the suite" (`04-07-PLAN.md:271-276`). That
assertion lands in `web/tests/data-table-render-cost.test.ts`, which `task web:test` runs
indiscriminately — and `task web:test` is a PR-triggered CI job
(`.github/workflows/ci.yml:149`). The plan itself concedes "jsdom timing is not browser
timing" (`04-07-PLAN.md:273`). A load-sensitive threshold in a required check is the
nondeterministic gate this phase rejects elsewhere. **Fix:** measure and decide in a
dedicated task this phase; keep only deterministic structural assertions in Vitest.

**HIGH — ten frontend verify commands discard their count-floor exit status (orchestrator
finding; not raised by Codex).** Every vitest-based `<automated>` block has the shape
`… vitest run …; RC=$?; node -e '…process.exit(1)'; test "$RC" -eq 0` —
`04-01-PLAN.md:387`, `:448`; `04-04-PLAN.md:256`, `:319`; `04-05-PLAN.md:161`, `:233`,
`:300`; `04-06-PLAN.md:162`, `:230`, `:305`; `04-07-PLAN.md:309`. In POSIX shell the
compound's status is the LAST command's, so the `node` floor check's non-zero exit is
discarded and only vitest's own status gates. Verified empirically:
`bash -c 'true; RC=$?; (exit 1); test "$RC" -eq 0'` exits **0**. A suite that shrinks below
its floor while all remaining tests pass reads as GREEN. This defeats the scoring
discipline `04-VALIDATION.md` states as binding ("a verify gate must honor **both** the
command's exit status **and** a named `--- PASS` / executed-test count floor"). The Go-side
gates got it right (`04-02-PLAN.md:152`, `:215`; `04-03-PLAN.md:308`, `:405`;
`04-07-PLAN.md:233` all use `&&`). **Fix:** chain with `&&`, or capture the node exit
separately and require both.

**MEDIUM — 04-03 directs reuse of a test helper that cannot be imported.** Task 3's action
says to reuse "`engine_worktree_test.go`'s worktree-creation helper"
(`04-03-PLAN.md:364-366`). `worktreeMismatchFixture` is unexported test code in package
`query` (`internal/query/engine_worktree_test.go:72`); `internal/uiserver/health_test.go`
cannot import it. **Fix:** say "replicate the technique" and add a uiserver-local fixture,
or promote a shared exported helper.

**MEDIUM — 04-07's "release-path" claim is unbacked.** The plan calls the target "a local
and release-path target" (`04-07-PLAN.md:41`) while requiring
`rg -c 'web:components:drift' .github/workflows/*.yml` to report 0
(`04-07-PLAN.md:241`, `:352`). Verified: no workflow, `release.yml` included, references
it. It is therefore local-only, so T-04-27 — a `high` threat whose only structural
mitigation this target is — is mitigated by a gate nobody is obliged to run. **Fix:** wire
it into a release-only (non-PR) job, or drop "release-path" and state plainly that the
mitigation is manual.

**MEDIUM — `/health` renders two snapshots with no rule for disagreement.** The verdict
comes from the shared `GetStatus` gate, which fetches on construction and navigation
(`web/src/lib/status.ts:116`), while commit SHA, version and reindex recommendation come
from a separate `GetHealth` call. During a re-index the two can legitimately disagree, and
HLT-02's whole point is that the verdict is trustworthy. Reusing `classifyStatus` (D-04) is
right; the missing piece is a display rule. **Fix:** state which snapshot wins and how a
mismatch is surfaced.

**MEDIUM — 04-02's malformed-pattern test cannot prove refusal precedes the scan.** The
sanity check is where the plan says (`internal/query/files.go:148`), but asserting error
text alone would still pass if a future edit moved it below `IterateFiles` — which is the
property T-04-05 actually relies on. **Fix:** assert with an instrumented reader whose
`IterateFiles` fails the test if called.

**MEDIUM — one pinned CLI version may not reproduce six already-vendored component
families.** `web/src/lib/components/ui/` already holds `button`, `command`, `dialog`,
`input`, `input-group`, `textarea` (verified in-tree), with `table` and `tabs` to come.
Task 2 assumes a single `shadcn-svelte@1.5.1` regeneration is authoritative for all of
them. Task 1's determinism probe is the right instrument, but the plan's objective
under-weights how likely a HALT is. **Fix:** state the HALT branch as the expected outcome
if any pre-existing family came from another registry revision.

**MEDIUM — 04-01 carries intake, vendoring, URL grammar, error taxonomy and the tracer in
one 95k-token plan** with self-declared `confidence: low`. A tracer failure could originate
in SvelteKit route context, TanStack, the URL module, or the harness. **Fix (optional):**
split dependency/vendor intake from the Callers tracer.

**LOW — 04-07's floor is set at the full observed count.** The plan correctly requires the
"never a growth ratchet" comment (`04-07-PLAN.md:187-193`, matching `Taskfile.yml:577`),
but a floor equal to today's count still ratchets on intentional removal. A small
structural minimum matches the repo's own convention better.

**LOW — negative `depth`/`limit` passthrough is untested.** `INTEGER_SHAPE` is `^-?\d+$`
(`web/src/lib/browse-url.ts:48`), so negatives parse and reach the RPC by design (D-07).
Tests cover oversized positives; add the negative case.

**LOW — `GetHealthRequest.path` is labelled "RESERVED" while being an active wire field**,
copying the same ambiguous wording already at `internal/uiproto/uiv1/ui.proto:107`. Tighten
before duplicating it.

**LOW — new proto messages `PendingChanges` / `IndexHealth` shadow existing Go domain type
names** (`internal/query/status.go:67`, `:76`). Generated-code ambiguity only; a `Health…`
prefix would avoid it.

**LOW — reusing `describeWorkbenchFailure` on `/health`** makes the health view depend
conceptually on Workbench's failure taxonomy. A neutral `describeRpcFailure` would read
better.

**LOW — 04-02's RED expectations assume a corpus that is stated, not verified.** Task 1
requires a corpus "that contains at least one root-level file and at least one file nested
two directories deep" (`04-02-PLAN.md:119-120`) while also reusing the existing
`filesStatusFixture` (`internal/query/files_status_test.go:16`). Confirm the shared fixture
actually carries those names before writing the RED expectations.

### Divergent Views

No second reviewer, so no model-vs-model divergence. Two Codex findings were **downgraded**
by orchestrator verification and should not be actioned:

- **`@tanstack/svelte-table` as a `devDependency` (Codex: MEDIUM).** Downgraded to noise.
  The repo already places runtime-imported libraries in `devDependencies` — `bits-ui`,
  `svelte`, `@lucide/svelte`, `@internationalized/date` (`web/package.json:17-38`) — because
  Vite bundles them. 04-01 follows the established precedent; no justification is owed.
- **04-07's floor conflicting with the "structural minimum" convention (Codex: MEDIUM).**
  Partially incorporated already: the plan explicitly requires the "exists ONLY to catch
  'enumerated nothing' … must never be raised as a growth ratchet" comment
  (`04-07-PLAN.md:187-193`). Only the removal-ratchet residue survives, recorded as LOW
  above.

## Verification coverage

Every HIGH and every MEDIUM above was re-checked against the working tree by the
orchestrator before being recorded. Commands executed this session:

| Claim | Check | Result |
|---|---|---|
| `navigationIdentity` excludes only `q` | read `web/src/lib/status.ts:74-88`, `web/src/routes/+layout.svelte:47-48` | confirmed |
| No plan addresses the refetch | `rg 'navigationIdentity\|VIEW_LOCAL_PARAMS\|notifyNavigated' 04-*.md` | 0 matches |
| `DataTable` typed to `Location` | `04-01-PLAN.md:350-353` vs `04-05-PLAN.md:215-216` | confirmed conflict |
| `AnalysisPanel.run` returns `Location[]` | `04-04-PLAN.md:224` vs `:32`, `:188-192`; `04-06-PLAN.md:283-286` | confirmed conflict |
| 04-06 prohibition vs action | `04-06-PLAN.md:44`, `:39` vs `:141-150`; `web/src/lib/search.ts:129-231` | confirmed contradiction |
| abort-count assertion | `04-06-PLAN.md:127-128` vs `web/src/lib/search.ts:143-145`, `:207-211` | confirmed unsatisfiable |
| unescaped glob term | `04-06-PLAN.md:151`; `internal/query/files.go:148-149` | confirmed |
| enumeration returns zero | `git ls-files -- 'web/src/lib/components/ui/*/' \| wc -l` | **0** (vs 36 without the trailing slash) |
| perf threshold in required CI | `04-07-PLAN.md:271-276`; `.github/workflows/ci.yml:149` | confirmed |
| count floors discarded | `bash -c 'true; RC=$?; (exit 1); test "$RC" -eq 0'` | exit **0** — floor swallowed |
| cross-package fixture | `internal/query/engine_worktree_test.go:72` (unexported, package `query`) | confirmed impossible |
| release-path unwired | `ls .github/workflows/`; no reference to `web:components:drift` | confirmed |
| six vendored families exist | `ls web/src/lib/components/ui/` | button, command, dialog, input, input-group, textarea |
| devDependency precedent | `web/package.json:17-38` | Codex finding downgraded |
