---
phase: 3
slug: browse-inspect-navigation
# status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6)
# audit-milestone §5.5 distinguishes NOT-VALIDATED (draft) from PARTIAL (validated + nyquist_compliant: false) (#2117)
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-08-28
---

# Phase 3 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.
> Seeded from `03-RESEARCH.md` § Validation Architecture (lines 547-583).

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework (Go)** | stdlib `testing`, run via `task test:unit` (`go test` over every package except `internal/daemon`) — `Taskfile.yml:116-131` |
| **Framework (JS)** | **vitest + jsdom + @testing-library/svelte** — DECIDED 2026-08-28 (maintainer). None installed today: zero `*.test.*`/`*.spec.*` files under `web/`, no test deps in `web/package.json`. Wave 0 installs and configures. |
| **Config file (Go)** | none (stdlib) |
| **Config file (JS)** | none yet — Wave 0 adds a `test:` block to the EXISTING `web/vite.config.ts`. A separate `web/vitest.config.ts` is deliberately NOT used: `Taskfile.yml`'s `web_source_files()` enumerates `web/vite.config.ts` and would not hash a separate file, so the BLD-03 source digest would not see a change to the test configuration. |
| **Quick run command** | `go test ./internal/uiserver/... ./internal/query/... ./internal/gitmeta/... ./web/...` and `cd web && pnpm test` |
| **Full suite command** | `task test:unit` and `task web:test` |
| **Estimated runtime** | quick ~15s (scoped Go packages); JS and full suite ~TBD at first green run |

---

## Sampling Rate

- **After every task commit:** Run the touched package's `go test ./internal/<pkg>/...`
- **After every plan wave:** Run `task test:unit` (full Go suite) and `task web:test` (JS suite, which prints its observed executed-test count before judging it), plus manual browser UAT for client-only surfaces
- **Before `/gsd-verify-work`:** `task test:unit`, `task web:test`, `task proto:drift`, `task web:build:verify` and `task web:drift` all green
- **Max feedback latency:** 60 seconds for the quick run

---

## Per-Task Verification Map

Task IDs use `T-{phase}-{plan}-{task}`, matching the task order inside each PLAN.md.
Reconciled with the real planner-assigned tasks on 2026-08-28, and **re-reconciled after
cycle-1 cross-AI review** (same date) for waves, commands and the "File Exists" column.

**What changed in the re-reconciliation, and why it matters here specifically:**

- **Waves shifted for plans 03-05 through 03-10.** `03-05` regenerates
  `web/src/lib/gen/ui_pb.ts` and previously shared wave 2 with `03-04`, which
  type-checks against that same directory — a same-wave data-contract race in a
  shared worktree. `03-05` now depends on `03-04` (a serialization edge, not a code
  dependency) and `03-06` on `03-05`, so the generated client has exactly one writer
  at a time. Everything downstream moves by one: 03-05 → 3, 03-06 → 4, 03-07 → 5,
  03-08 → 6, 03-09 → 7, 03-10 → 8.
- **Three "File Exists ✅" marks were the vacuity signal and are now ❌.**
  T-03-03-01, T-03-09-03 and T-03-10-03 were each marked ✅ because the command they
  named ran green against the CURRENT tree — which is exactly the property that made
  those three `<verify>` blocks pass in a world where their task had not been done.
  Their commands now bind to identities that do not exist yet, so the honest mark is ❌.
  A ✅ in this column for a task whose whole job is to ADD something is a red flag,
  not a convenience.
- **Commands updated** wherever the plan's `<verify>` changed: the vitest JSON reporter
  (T-03-01-02), the new named evidence test (T-03-03-01), `go test -v` run directly
  rather than through the no-`-v` `task test:wireoracle` (T-03-03-03), the
  deliverable-bound gate sweep (T-03-09-03), and the folded CI-registration counts
  (T-03-10-03).
- **`${PIPESTATUS[0]}` removed from every command in this table.** It is a bashism that
  expands to empty under `sh`/`dash`, where `test "" -eq 0` errors rather than judging.
  The replacement shape — redirect to a file, check the command's own exit status with
  `||`, then count against the file — was executed in all three worlds during the
  revision (command fails / passes with pattern / passes without pattern → 1 / 0 / 1).

**What changed after cycle-2 review (2026-08-28), and why:**

- **Two hardcoded-iteration-set vacuities closed in 03-10 Task 1.**
  `TestToolModfilesRemainIsolated` iterates `[]string{toolModfilePath, lintModfilePath}`
  (`internal/upgrade/taskfile_shape_test.go:928`) and `Taskfile.yml`'s `vuln` target
  enumerates four binaries by hand. A new tool modfile absent from either is not merely
  unguarded — the test PASSES BECAUSE it is absent, and the `vuln` `desc:`'s "all four
  stay in scope" claim silently becomes false. T-03-10-01's command now asserts the
  fixture const, the forbidden-package entry and the five-binary scan loop with positive
  `rg -o … | wc -l` counts (all **0** today against working controls of 10 and 1), and
  the full chain was EXECUTED against the current tree and exited 1.
- **03-09's constructor contradiction resolved at every site.** `createStatusGate(client)`
  could not record "the identity it fetched for", so the `1 → 1 → 2` fetch-count test was
  not implementable. The constructor now takes `(client, initialNavigationIdentity)`, and
  the artifact table, `<behavior>`, both task bodies, the tests and this table all state
  it. Commit knowledge moved to an orthogonal `commit: known | unknown` field so a healthy
  index with an unrecorded SHA no longer renders as a degraded verdict.
- **One "check aimed at a nonexistent path" closed.** T-03-09-03's
  `rg -o 'fc4ae27b…' … | wc -l` = 0 passes when the manifest is ABSENT. A positive control
  (`rg -o 'source-sha256' … | wc -l` = 1, observed 1 today) now precedes it in the same
  chain. Executed against a missing path, the pair exits 1 where the bare claim exited 0.
- **T-03-10-02 is no longer a bare `task lint:go`.** The green is bound to the linter set
  T-03-10-01 committed, by byte-identity of `.golangci.yml` — strictly stronger than the
  equal-count check the review asked for, and runnable.
- **Every command changed this cycle was executed against the working tree** and confirmed
  to exit non-zero in the unimplemented world, and every positive control confirmed to
  return a non-zero count, per this repository's standing rule that a remedy for a vacuous
  guard carries the same burden of proof as the guard.

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| T-03-01-01 | 03-01 | 1 | BRW-06 | T-03-SC | Blocking-human package-legitimacy gate for all five new npm packages before any install runs; never auto-approvable | Human checkpoint | — | N/A | ⬜ pending |
| T-03-01-02 | 03-01 | 1 | BRW-06 | T-03-10, T-03-11 | vitest + jsdom + @testing-library installed and configured; a deliberately-failing assertion observed RED before any green is trusted; the GREEN proof renders a real compiled `web/tests/fixtures/Harness.svelte` (Testing Library's `render` needs a compiled component — an inline one is not expressible in `.ts`); `allowBuilds` outcome recorded, not assumed | JS harness bring-up | `pnpm exec vitest run --reporter=json --outputFile=/tmp/vitest-harness.json` then assert `numTotalTests >= 1 && numPassedTests === numTotalTests` | ❌ W0 | ⬜ pending |
| T-03-01-03 | 03-01 | 1 | BRW-06 | T-03-12 | `task web:test` prints its observed executed-test count BEFORE comparing to a floor; demonstrated RED against an empty include glob | Taskfile guard | `task web:test` | ❌ W0 | ⬜ pending |
| T-03-02-01 | 03-02 | 1 | SRV-05 | T-03-01 | `..` escape, absolute path, empty path and post-symlink escape each refused at the RPC boundary, **and** a legitimate in-repo path returns real source in the same test (positive control, rule `84d1gfpywd`); the symlink case may be skipped only on a genuinely unsupported platform, never on linux/darwin | Go integration (Connect handler, real client) | `go test ./internal/uiserver/ -run TestGetNodeDetailPathConfinementAtRPCBoundary -count=1 -v`, then `grep -Eo '^ *--- PASS: TestGetNodeDetailPathConfinementAtRPCBoundary/[a-zA-Z0-9_-]+' \| wc -l` ≥ 5 (character class widened from `[a-z_]+`, which dropped hyphenated and capitalised subtest names) | ❌ W0 | ⬜ pending |
| T-03-02-02 | 03-02 | 1 | SRV-05 | T-03-06 | No confinement refusal message discloses the host checkout path; paired with a positive containment assertion so the absence check is proven to inspect a real message | Go integration | `go test ./internal/uiserver/ -run TestGetNodeDetailConfinementRefusalDoesNotLeakHostPath -count=1 -v` | ❌ W0 | ⬜ pending |
| T-03-03-01 | 03-03 | 1 | TODO-MCP-01 | T-03-16 | Wire-oracle ordering flake reproduced under contention with a timestamped arrival sequence; verdict recorded in `03-03-EVIDENCE.md` **before** Task 2's checkpoint reads it (the SUMMARY does not exist until after Task 3) | Go integration + evidence artifact | `go test ./test/wireoracle/ -run '^TestCaptureArrivalLedgerPreservesWireOrder$' -count=1 -v` with a `--- PASS:` count of 1, **plus** exactly one `VERDICT:` literal in `03-03-EVIDENCE.md`. Was `-run TestFrozenTranscriptsMatch/toolslist-repeat` matching `PASS\|FAIL` — that subtest already runs — it is GENERATED by `TestFrozenTranscriptsMatch`'s `t.Run(sc.Name, …)` at `oracle_test.go:119` from the scenario at `scenarios.go:669` — and either verdict satisfied it, so it demanded no investigation. (Cycle-3 Claude LOW: the previous citation `oracle_test.go:608-643` points at `TestToolsListOrderIsDeterministic`, a different test; claim correct, pointer wrong, both re-verified.) | ❌ W0 | ⬜ pending |
| T-03-03-02 | 03-03 | 1 | TODO-MCP-01 | T-03-14 | Maintainer selects R1 / R2 / NR from the evidence artifact; blocking-human so auto-advance cannot decide it; R1 additionally requires the SDK citation, since it would add a response-ordering guarantee to production dispatch that `pendingWriter` does not currently make | Human decision | — | N/A | ⬜ pending |
| T-03-03-03 | 03-03 | 1 | TODO-MCP-01 | T-03-15 | Chosen resolution proven RED first under a decision-independent test name; content-discrimination now an automated test as well as a live plant; **the NR branch's RED contract is settled by `03-03-EVIDENCE.md`, not by executor judgement** — NR-a (recording falsifies the old claim) REQUIRES a scratch assertion replaying that recording against the old claim, observed RED, recorded verbatim, then deleted; NR-b (recording is consistent with it) carries no RED and says so in one sentence quoting the deciding evidence lines. Exactly one record is present. Was an 'or record why' hedge the executor could invoke at will | Go integration | `go test ./test/wireoracle/ -run '^(TestToolsListRepeatOrderingResolution\|TestFrozenTranscriptComparisonDetectsContentMutation\|TestFrozenTranscriptsMatch)$' -count=1 -v` — **run directly, not via `task test:wireoracle`**, whose recipe (`Taskfile.yml:176`) omits `-v` and therefore emits zero `--- PASS:` lines regardless of outcome. `task test:wireoracle` remains a separate full-suite criterion | ❌ W0 | ⬜ pending |
| T-03-04-01 | 03-04 | 2 | BRW-02, BRW-06, NAV-01, NAV-04 | T-03-02 | TRACER: URL -> RPC -> error classification -> highlight -> DOM, one path end to end; exactly one unescaped-HTML site in `web/src` (excluding the yet-to-be-vendored `components/ui/`); unregistered languages render escaped plaintext, never an auto-detected guess | JS component (jsdom + @testing-library/svelte) | `pnpm test` names `browse-tracer`, **plus** the corrected shallow-routing prohibition `rg -o 'import \{[^}]*(pushState\|replaceState)[^}]*\} from .\$app/navigation.' web/src/ \| wc -l` = 0 **paired with** its positive control `rg -o 'from .\$app/state.' web/src/ \| wc -l` ≥ 1 | ❌ W0 | ⬜ pending |
| T-03-04-02 | 03-04 | 2 | NAV-01, NAV-04 | — | URL round-trip idempotent and deterministically ordered; unknown params preserved as an ordered multimap (repeated and empty-valued keys survive), not rejected (D-12); non-integer numerics treated as absent, not coerced; Connect error kinds mutually distinct | JS unit (pure TS, no DOM) | `pnpm test` naming `browse-url` and `rpc-errors`, counted separately | ❌ W0 | ⬜ pending |
| T-03-04-03 | 03-04 | 2 | BRW-06 | — | Highlighter coverage set-equals `indexer.RegisteredLanguageIDs()` — no missing, no extra; parsed from the constrained exported `HIGHLIGHT_COVERAGE` array rather than free-form registration syntax; a planted fixture proves the comparison discriminates in both directions | Go unit, set-equality over the real TS file | `go test ./web/ -run '^TestHighlight' -count=1 -v` with a `--- PASS:` count of exactly 2 | ❌ W0 | ⬜ pending |
| T-03-05-01 | 03-05 | 3 | BRW-09 | T-03-24 | One-way wire shape frozen by a human before any generated code exists, **including** an unambiguous answer to the git-failure contract: `could not check` must be a different value from `checked and not observed`, or the rejection of that must be recorded in those words. **Cycle-3 addition — point 5:** the since-deleted-file disposition, put to the maintainer on the CORRECTED premise (today's answer is `CodeInternal` + `an internal error occurred`, not `CodeInvalidArgument` + the confinement message), recorded as one of `deleted-accept-internal` / `deleted-classify-in-handler` / `deleted-classify-at-gate` together with the exact `(code, message)` pair it implies and whether it was chosen or defaulted | Human decision (blocking-human) | — | N/A | ⬜ pending |
| T-03-05-02 | 03-05 | 3 | BRW-09 | T-03-03, T-03-20, T-03-22 | Remote URL parsed across all real transport shapes; exact `github.com` host equality; credentials stripped (asserted against a populated result); no network call; tri-state `RemotePresence` and structured `GitHubRemote{Owner,Repo,Host,Reason}` so each no-link cause names itself; every git call routed through the exec seam so "git absent" is a testable branch | Go unit | `go test ./internal/gitmeta/ -run 'Permalink\|Remote' -count=1 -v` with a named-PASS count ≥ 12 | ❌ W0 | ⬜ pending |
| T-03-05-03 | 03-05 | 3 | BRW-09, SRV-05 | T-03-01b, T-03-21 | `GetPermalink` correct across pushed / unpushed / no-remote / non-GitHub / absent-SHA; path percent-encoded; escaping AND nonexistent paths refused by the reused confinement gate, with a passing control. **The since-deleted case is pinned by a FIXED test name, `TestGetPermalinkRefusesSinceDeletedFile`, whose ASSERTIONS follow T-03-05-01's point-5 answer** — `CodeInternal` + `an internal error occurred` under the default, or `CodeInvalidArgument` + a handler-built message naming the repo-relative path (and NOT the absolute host path) if classification was chosen. The previous revision pinned `CodeInvalidArgument` + the confinement message, which the source does not produce: `resolveSourcePath` returns the raw `EvalSymlinks` `*fs.PathError` (`node.go:66-72`), `classifiedError.Is` matches only its own sentinel (`errors.go:43-45`), and `mapEngineError` falls to its scrubbing default arm (`handlers.go:112-140`) | Go integration (Connect handler, real client) | `go test ./internal/uiserver/ -run Permalink -count=1 -v` with a named-PASS count ≥ 10 **and** exactly one `--- PASS: TestGetPermalinkRefusesSinceDeletedFile` line (both 0 today), `task proto:drift`, and additive-only via `git diff --numstat ... \| awk '$2 != 0 { bad=1 } END { exit (NR == 0 \|\| bad) ? 1 : 0 }'` | ❌ W0 | ⬜ pending |
| T-03-06-01 | 03-06 | 4 | BRW-01 | T-03-07, T-03-SC-b | Every vendored component file and every npm package the registry CLI added is enumerated and human-approved before code is written against it; **plus** an unescaped-HTML inventory of the vendored directory dispositioned at add time, **plus** a keyboard-capability spike answered before the controller is built; explicit rollback path if anything is rejected | Human checkpoint (blocking-human) | — | N/A | ⬜ pending |
| T-03-06-02 | 03-06 | 4 | BRW-01, BRW-08 | T-03-25 | Debounce, two-character minimum, mandatory abort AND a request-identity guard; a late response cannot overwrite a newer one (asserted by value); explore never rides a keystroke | JS unit (fake timers, manual promises) | `pnpm test` (names `search.test`) | ❌ W0 | ⬜ pending |
| T-03-06-03 | 03-06 | 4 | BRW-01, BRW-08, NAV-03 | T-03-26 | Two live sections in server order plus a third explore section; empty and failed render differently; `/`, the command chord and Escape behave, with `/` suppressed inside inputs; arrow traversal built to match Task 1's recorded keyboard finding | JS component (jsdom) | `pnpm test` names `search-panel`, plus `rg -o 'PLACEHOLDER-03-07' web/src/ \| wc -l` = 1 (the one-wave selection placeholder 03-07 removes) | ❌ W0 | ⬜ pending |
| T-03-07-01 | 03-07 | 5 | NAV-01, NAV-02 | T-03-28 | One URL writer; navigate pushes and refine replaces (asserted as differing booleans); scroll and focus preserved; no shallow-routing history export imported anywhere | JS unit (spy for the injected navigator) | `pnpm test` names `browse-nav`, **plus** the corrected prohibition `rg -o '...from .\$app/navigation.' \| wc -l` = 0 **paired with** its positive control `rg -o 'from .\$app/navigation.' web/src/ \| wc -l` ≥ 1 (this plan imports `goto` from that module) | ❌ W0 | ⬜ pending |
| T-03-07-02 | 03-07 | 5 | BRW-02, NAV-01 | T-03-27 | Response mode read from its own discriminator, not inferred from emptiness; depth and limit reach the server exactly as the URL states them, with the two parameters' DIFFERENT server contracts asserted separately — `?limit=100000` is REFUSED (`validateLimit` rejects `n > MaxLimit`, `validate.go:107-115`) and renders the named invalid-input state, while `?depth=999` SUCCEEDS and is CLAMPED to 50 (`validateDepth` rejects only `n < 0`, `validate.go:137-142`; `Engine.Impact` calls `clampDepth`, `traverse.go:438-442`) with the response's echoed depth authoritative. The previous revision asserted a depth REFUSAL, which the engine declines to perform — a mocked client could have satisfied it while the real RPC returned success; ONE navigation generation governs both the detail load and the blast-radius load, so results from two URL states can never be combined | JS unit | `pnpm test` (names `browse-state`) | ❌ W0 | ⬜ pending |
| T-03-07-03 | 03-07 | 5 | BRW-02, BRW-03, NAV-02 | T-03-17b | Callers (typed `Node`) and blast radius (typed `Location`) visible together with source; click-through carries the navigate intent, depth change the refine intent; **NAV-02's URL↔state half now automated** over a real jsdom history stack, router half still manual with recorded URLs | JS component + JS unit + manual UAT | `pnpm test` naming BOTH `neighbors-panel` and `browse-history`, plus `rg -o 'PLACEHOLDER-03-07' web/src/ \| wc -l` = 0 | ❌ W0 | ⬜ pending |
| T-03-08-01 | 03-08 | 6 | BRW-04 | T-03-02b | Identifier matching is exact — no case folding, no normalization, no prefix; each negative case paired with a positive exact match; decoration builds no markup strings, is idempotent, tears down cleanly, and does not match an identifier split across highlight spans (a documented bound) | JS unit + DOM | `pnpm test` names `call-targets`, plus the two HTML-construct counts scoped `--glob '!**/components/ui/**'` and each paired with a positive control | ❌ W0 | ⬜ pending |
| T-03-08-02 | 03-08 | 6 | BRW-07, BRW-09 | T-03-29, T-03-31 | Truncation notice present/absent proven both ways; three permalink availability states pairwise different; empty-value copy affordance absent, paired with a present case | JS component (jsdom, stubbed clipboard) | `pnpm test` (names `source-pane`) | ❌ W0 | ⬜ pending |
| T-03-08-03 | 03-08 | 6 | BRW-05 | — | Picker lists every RETURNED candidate and states the true total, naming the difference when they diverge (tested with a `totalCandidates > definitions.length` fixture today's server does not produce); marks ungathered ones from the wire's own flag (proven against an empty-calls gathered candidate); auto-selects nothing; preserves server order | JS component (jsdom) | `pnpm test` (names `definition-picker`) | ❌ W0 | ⬜ pending |
| T-03-09-01 | 03-09 | 7 | NAV-04 | T-03-34 | No-index and indexing verdicts distinct despite both having `initialized` false; no polling, proven by an unchanged call count across advanced time paired with an increase after navigation; **`createStatusGate(client, initialNavigationIdentity)` takes TWO arguments** so the gate can record the identity it fetched for and the `1 → 1 → 2` test is implementable (a client-only constructor made it unimplementable); **commit knowledge is an ORTHOGONAL `commit: known \| unknown` field, not a sixth health verdict** — `StatusVerdict` stays at five members and an empty `commit_sha` yields verdict `ok` with `commit: unknown` | JS unit (fake timers) | `pnpm test` names `status.test`, plus `rg -o 'notifyNavigated' web/src/lib/status.ts \| wc -l` ≥ 1 and `rg -o 'navigationIdentity' web/src/lib/status.ts \| wc -l` ≥ 1 (0 today) | ❌ W0 | ⬜ pending |
| T-03-09-02 | 03-09 | 7 | NAV-04 | T-03-33 | Each of no-index / stale / not-found renders its own named state, pairwise different; stale banner and in-view not-found coexist; the two source-absent messages differ by stale flag; the banner is mounted at layout level with exactly ONE navigation caller | JS component (jsdom) | `pnpm test` names `degrade-states`, plus `rg -o 'StatusBanner' web/src/routes/+layout.svelte \| wc -l` ≥ 1 (observed 0 today) and `rg -o 'notifyNavigated' web/src/routes/+layout.svelte \| wc -l` = 1 and `rg -o 'navigationIdentity\(page\.url\)' web/src/routes/+layout.svelte \| wc -l` = **2** (0 today) — the constructor's initial identity and the effect's identity, from one exported normalizer; a count of 1 would be the client-only constructor | ❌ W0 | ⬜ pending |
| T-03-09-03 | 03-09 | 7 | NAV-04 | T-03-32 | Committed bundle rebuilt; both drift digests recomputed and their observed counts recorded; no floor adjusted; **the sweep is bound to this plan's OWN deliverables**, and the intended-RED `web:drift` window opened by 03-01 is closed here | Taskfile guards + artifact binding | `pnpm --dir web test` materialized, then `task web:build:verify && task web:drift && task proto:drift && task test:unit` **plus** `status.test` and `degrade-states` present in the output, `StatusBanner` in `+layout.svelte`, `rg -o 'codegraph init' web/build/ \| wc -l` ≥ 1 (observed **0** today), and, as a PAIR, `rg -o 'source-sha256' web/build/.build-manifest \| wc -l` = 1 (the positive control, 1 today — proving the file exists and is readable) immediately preceding `rg -o 'fc4ae27b…' \| wc -l` = 0 (1 today). Without the control the absence claim passes against a manifest that does not exist; executed against a missing path the pair now exits 1. Was a chain of five pre-existing targets, all green on a Phase-2 tree | ❌ W0 | ⬜ pending |
| T-03-10-01 | 03-10 | 8 | TODO-CI-01 | T-03-SC-c | Linter pinned in its OWN `go.tool-golangci.mod` (a FOURTH tool modfile — `go.tool-proto.mod` also exists), created from the outset; **REGISTERED with `TestToolModfilesRemainIsolated`, whose hardcoded two-element slice (`taskfile_shape_test.go:928`) means an unregistered modfile makes the test PASS BECAUSE it is absent** — the same shape T-03-10-03 fixes for `inScopeJobs`; **IN SCOPE for the `vuln` gate**, whose hardcoded four-binary set and 'all four stay in scope' `desc:` would otherwise be silently falsified; `.golangci.yml` states its own enabled/rejected set and carries an `# enabled-linters: N` anchor | Taskfile + tool modfile + fixture registration | `task --list-all` and `go test -run '^TestToolModfilesRemainIsolated$' -v` materialized, then `rg -o '^\* lint:go:' \| wc -l` = 1, one `--- PASS:` line, `rg -o 'golangciModfilePath' internal/upgrade/taskfile_shape_test.go \| wc -l` ≥ 2 (**0 today**, control 10), `rg -o 'golangci/golangci-lint' <same> \| wc -l` ≥ 1 (**0 today**), `rg -o 'for name in task goreleaser govulncheck actionlint golangci-lint' Taskfile.yml \| wc -l` = 1 (**0 today**, 4-word control 1), the `# enabled-linters:` anchor = 1, both modfiles exist, `task lint:actions` exits 0, **the ADVISORY-stance regression guard asserted as a PAIR** — exactly one `--- PASS: TestGateStancesStated` line AND `rg -o 'func TestGateStancesStated' internal/upgrade/taskfile_shape_test.go \| wc -l` = 1 (the previous revision named `TestGateStancesAgree`, which does not exist: `go test -run '^TestGateStancesAgree$'` prints `no tests to run` and exits 0, so the sole guard on the ADVISORY word asserted nothing; both conjuncts return 1 today and 0 under the wrong name) — and `rg -o '357 measured' Taskfile.yml \| wc -l` = 0 (**1 today**, `Taskfile.yml:1128` — the third stale figure in the same `desc:`, alongside the two "four"s) | ❌ W0 | ⬜ pending |
| T-03-10-02 | 03-10 | 8 | TODO-CI-01 | T-03-38 | First-run backlog FIXED not suppressed; pre-fix inventory recorded; suppression count recorded before and after. **The backlog is a PINNED SET named by an exact command over an exact path** — `git ls-files '*.go' \| rg -v '^testdata/' \| xargs <formatter> -l`, returning **8** under `gofmt` and **23** under `gofumpt` today. The previous revision named `internal/query/files_status_test.go` as the exemplar and made it Task 2's only `<files>` entry; that file is CLEAN under both formatters (measured), while the real set went undeclared. Generated `.pb.go`/`.connect.go` files appear in the gofumpt measurement and are EXCLUDED, never edited — `task proto:drift` byte-compares them and their own headers say DO NOT EDIT | Lint gate bound to Task 1's set | `test -f .golangci.yml && <the '# enabled-linters:' anchor> = 1 && git diff --exit-code HEAD -- .golangci.yml && <the pinned backlog command with gofmt> = 0 && task lint:go` — the backlog conjunct uses `gofmt` regardless of Task 1's choice because gofumpt's output is gofmt-stable, making it the one branch-free form; it returns **8** on the pre-implementation tree. Never `gofmt -l .` (28 — walks an untracked worktree), never `gofmt -l internal/` (5 — misses `tools/`), never a Go PACKAGE pattern (a false 0). — the config must be byte-identical to what T-03-10-01 committed, so a green reached by NARROWING the linter set fails the third conjunct. Was a bare `task lint:go`, which a narrowed `.golangci.yml` satisfies identically. Post-commit form: `git log --oneline -- .golangci.yml \| wc -l` = 1 (0 today) | ❌ W0 | ⬜ pending |
| T-03-10-03 | 03-10 | 8 | TODO-CI-01 | T-03-36, T-03-37 | Gate demonstrated RED on both a formatting and an idiomatic planted violation **in `internal/corpora/coverage_test.go`, a file the linter actually inspects AND that Task 2 actually fixes**, restored byte-identically (hash recorded both sides), then green. The previous revision planted in `internal/query/files_status_test.go` "because Task 2 has just fixed it" — a premise that does not hold, since that file is already clean. The replacement was chosen against three executed checks: it is in the formatter-independent backlog floor (`gofmt -d` shows two hunks today), its package is reported by `go list ./...`, and no other plan in this phase touches `internal/corpora`; the new CI job is REGISTERED with the single-definition guard | Go unit + CI wiring | `go test ./internal/upgrade/ -run '^TestWorkflowRunBodiesInvokeTask$' -count=1 -v` with a `--- PASS:` count of 1, **plus** `rg -o 'task lint:go' .github/workflows/ci.yml \| wc -l` = 1 **and** `rg -o 'JobID: "lint-go"' internal/upgrade/taskfile_shape_test.go \| wc -l` = 1 — both 0 today. Registration is asserted directly because the guard iterates only `inScopeJobs` and therefore PASSES when the job is absent | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

**Sampling continuity:** no three consecutive task IDs above lack an `<automated>` verify.
The four that carry none are the two blocking-human legitimacy/review checkpoints and the
two blocking-human decision checkpoints; none of them is adjacent to another such task
within the same plan.

**Manual-only residue** (each recorded in its plan's SUMMARY, none substituting for an
automated check): the SvelteKit-router half of the history walk (T-03-07-03 — the URL↔state
half is now automated in `browse-history.test.ts`, and the SUMMARY must state which half is
which rather than calling all of NAV-02 manual), the live permalink follow-through before
and after a re-index (T-03-08-02), the three live degrade observations (T-03-09-02), and
the rendered-highlighting spot check (T-03-04-01).

**Guards that carry a demonstrated positive control** (rule `84d1gfpywd`). Five of cycle
1's seven HIGH findings were `<verify>` blocks, and one — the shallow-routing prohibition —
was a pattern that could never match anything. Every prohibition-shaped check in this phase
is now paired with an assertion that the search can find what it is looking for:

| Prohibition | Its positive control | Where |
|---|---|---|
| no `pushState`/`replaceState` import from the navigation module (`rg -U`, multiline — a wrapped import evades the line-oriented form) | `from '$app/state'` count ≥ 1 (**1 today**) | T-03-04-01 |
| no bare `pushState` identifier anywhere in `web/src` (wrap-proof backstop; `pushState` only — `replaceState` is legitimate as `goto`'s option) | shares the row above's control | T-03-04-01, T-03-07-01 |
| no `pushState`/`replaceState` import from the navigation module (`rg -U`, multiline) | `from '$app/navigation'` count ≥ 1 (this plan imports `goto`; **0 today**, so it is red until 03-07 lands) | T-03-07-01 |
| no `highlightAuto` | `hljs.highlight(` count ≥ 1 in the same file | T-03-04-01 |
| no HTML-string APIs in `web/src` | `createElement\|createTextNode` count ≥ 1 in `call-targets.ts` | T-03-08-01 |
| no case-folding or normalization in the matcher | `lookupCallTarget` count ≥ 1 in the same file | T-03-08-01 |
| no timers in the status gate | `getStatus` count ≥ 1 in the same file | T-03-09-01 |
| no clamping in the load seam | `depth` count ≥ 1 in the same file | T-03-07-02 |
| no second serializer in `browse-nav.ts` | `serializeBrowseParams` count ≥ 1 in the same file | T-03-07-01 |
| no confinement logic in `internal/uiserver` | the in-repo positive control returns real bytes from the same live service | T-03-02-01 |
| no host-path disclosure in refusals | the caller's own submitted path IS present in the same message | T-03-02-02 |

Additionally, both shallow-routing guards carry a **planted-violation demonstration**
recorded in the SUMMARY — and since cycle 3 it is TWO plants, not one: the forbidden import
is written into a scratch file under `web/src/` first on ONE line and then WRAPPED across
lines, the counts observed NON-ZERO for each, each plant removed, and the counts observed
zero. The wrapped form is the one the cycle-2 guard missed (executed: `0` line-oriented vs
`≥1` with `-U`), so a demonstration that plants only the single-line form has not tested the
correction. `rg` is line-oriented by default; a guard that reads correctly and searches the
right files can still be blind to a formatting variant of its own target.
A prohibition guard nobody has watched fire is indistinguishable from a broken one — which
is precisely what cycle 1 shipped, four times.

---

## Wave 0 Requirements

- [x] `web/tests/fixtures/Harness.svelte` — a REAL compiled Svelte component for the harness bring-up proof (planned: 03-01, T-03-01-02). Added in the cycle-1 revision: `@testing-library/svelte`'s `render` takes a compiled component, and TypeScript has no expression form for Svelte markup, so the original "render a trivial inline Svelte 5 component" instruction was not implementable and the RED→GREEN conversion could not have been performed as written.
- [x] `test/wireoracle/capture_test.go` + `.planning/phases/03-browse-inspect-navigation/03-03-EVIDENCE.md` — the arrival-ledger evidence test and the verdict artifact Task 2's checkpoint reads (planned: 03-03, T-03-03-01). The evidence file exists because the checkpoint previously pointed at `03-03-SUMMARY.md`, which this plan's own `<output>` does not create until after Task 3.
- [x] `web/tests/browse-history.test.ts` — the automated half of NAV-02, over a real jsdom history stack (planned: 03-07, T-03-07-03). Added in the cycle-1 revision because back/forward correctness was manual-only despite NAV-02 being a central requirement.
- [x] `internal/uiserver/confinement_test.go` — SRV-05's negative cases **paired with** a passing in-repo control (planned: 03-02, T-03-02-01/T-03-02-02)
- [x] `internal/gitmeta/permalink_test.go` and `internal/uiserver/permalink_test.go` — BRW-09's D-06/D-07/D-08/D-09 branches (planned: 03-05, T-03-05-02/T-03-05-03)
- [x] `web/highlight_coverage_test.go` (package `web_test`, so the production `web` package gains no dependency on `internal/indexer`; planned: 03-04, T-03-04-03) — mirrors `internal/indexer/capability/matrix_test.go:33-41`'s "no missing, no extra" set-equality pattern against `indexer.RegisteredLanguageIDs()`, guarding the 13-module hljs registration list (covering 14 registered `LanguageSpec.ID` values, since the typescript module's own alias list already claims `tsx`) against silent drift when a 15th language is added
- [x] **JS test harness — DECIDED (maintainer, 2026-08-28): adopt `vitest` + `jsdom` + `@testing-library/svelte`.** Planned: 03-01, T-03-01-01/T-03-01-02/T-03-01-03. Configuration lives in `web/vite.config.ts`'s `test:` block, NOT a separate `vitest.config.ts` — `web_source_files()` in `Taskfile.yml` enumerates `vite.config.ts` and would not see a separate file. Tests live in `web/tests/`, which SvelteKit's generated tsconfig already includes and which `web_source_files()` deliberately does NOT hash, so a test edit does not churn the BLD-03 source digest. Scope covers both pure-TS modules (`browse-url.ts`, `rpc-errors.ts`, the click-to-definition name matcher) and Svelte component behavior (search combobox, keyboard nav, error states).
  - Harness bring-up MUST prove the runner executes: land one deliberately-failing assertion, observe it RED, then fix it. A test suite that reports 0 failures because it ran 0 tests is the exact vacuous-pass shape rule `84d1gfpywd` names, and a fresh harness is where it is most likely.
  - New devDependencies enter `pnpm-lock.yaml`. Verified during planning: `task web:lockfile`'s assertion is a FLOOR of 50 against an observed 129, not an exact expected count, so adding packages cannot break it — but the target's `desc:` records the observed count as of 02-07 and 03-01 re-runs the gate so the new observed number is printed and recorded.
  - `pnpm` ≥10 blocks lifecycle scripts by default, and a bare pnpm install with newly-blocked scripts exits 0 with only a warning. Verified during planning: this repository already closes that hole — `web/pnpm-workspace.yaml` sets `strictDepBuilds: true`, which makes such an install exit NON-zero, and `task web:deps:strict` separately asserts the `allowBuilds` key is present (an absent key is a named failure distinct from a committed empty map). 03-01 still runs `pnpm approve-builds --all` and records WHICH of the two outcomes occurred, because a silent no-op and a real approval are not the same event.

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| `Explore` NL results rendered alongside exact-name search | BRW-08 | Requires a live index with real relevance ranking; jsdom cannot supply meaningful ranked data | Run `codegraph ui` against this repo's index; enter a natural-language question; confirm both sections populate and are visually distinguishable |
| GitHub permalink opens the correct remote line at the indexed commit | BRW-09 | Requires a real remote and a browser; the Go unit tests cover URL *derivation*, not that the resulting page is right | Open a file at a known line, follow the permalink, confirm the remote view matches what the UI showed |
| Syntax highlighting renders correctly across the registered languages | BRW-06 | Visual correctness is not assertable; the automated guard covers *registration coverage*, not rendering quality | Open one file per registered language; confirm tokens are colored and nothing renders as plain text |

*Reduced from the pre-decision list: debounce/abort, keyboard nav and error states moved to automated jsdom coverage.*

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Per-Task Verification Map rewritten with real planner-assigned task IDs
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] JS harness proven RED-then-GREEN before any JS test result is trusted
- [ ] Every guard carries a positive assertion that it did its work (rule `84d1gfpywd`)
- [ ] No watch-mode flags
- [ ] Feedback latency < 60s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
