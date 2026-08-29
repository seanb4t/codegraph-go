---
phase: 03-browse-inspect-navigation
plan: 04
subsystem: ui
tags: [svelte, sveltekit, connectrpc, highlight.js, url-state, go-testing]

requires:
  - phase: 03-browse-inspect-navigation
    provides: "03-01's vitest/jsdom/@testing-library/svelte JS test harness and task web:test; highlight.js@11.12.0 already installed"
provides:
  - "web/src/lib/browse-url.ts — the ONE URL grammar for the Browse view (D-13), extended by later plans in this phase rather than duplicated"
  - "web/src/lib/rpc-errors.ts — the ONE Connect-error classification (D-04), reused by Phases 4/5/6"
  - "web/src/lib/highlight.ts — the ONE syntax-highlighting rendering path (D-19/BRW-06), with HIGHLIGHT_COVERAGE bound to the indexer's registry by a Go set-equality guard"
  - "web/src/lib/browse-state.ts — the load seam mapping BrowseParams to GetNodeDetail and back to a named BrowseTargetState, testable without a SvelteKit runtime"
  - "web/src/lib/components/browse/SourcePane.svelte — the phase's one @html render site"
  - "web/src/routes/browse/+page.svelte — filled in, reads page.url.searchParams reactively, imports no shallow-routing history exports"
  - "web/highlight_coverage_test.go — Go-side proof the highlighter's registered language set equals the indexer's own registry"
affects: [03-05, 03-06, 03-07, 03-08, 03-09]

actuals:
  tokens: 13414
  tasks: 3
  commits: 3

tech-stack:
  added: []
  patterns:
    - "vite.config.ts's test.reporters set to 'verbose' so `pnpm test`'s own stdout names every test file even on a full pass — vitest's default reporter collapses to bare counts once nothing fails, which would make a plain `pnpm test` invocation unable to prove any SPECIFIC test executed (rule 84d1gfpywd). task web:test is unaffected: its --reporter=json CLI flag overrides this config value entirely."
    - "Ordered-pairs (not object) representation for unknown/passthrough URL parameters — an object silently collapses a repeated key, which is precisely the 'preserved' vs 'ignored' distinction D-12's own text calls out."
    - "highlight.ts declares a single-purpose, string-literal-only HIGHLIGHT_COVERAGE array specifically so a Go test can parse it as constrained text rather than guessing at free-form TypeScript call syntax — the array is proven not to drift from the real registerLanguage(...) calls by a TS-side unit test (forward direction) and a Go-side reverse-binding check (every array entry has a real registration or alias-key backing it)."
    - "Go coverage guards for a cross-language artifact belong in an EXTERNAL test package (web_test) sited beside the artifact they guard, never inside the producer package they'd otherwise couple, and never inside a package whose own test-only init() registrations would corrupt the authoritative set being compared against (internal/indexer/extract_test.go's synthetic 'go-dup' language)."

key-files:
  created:
    - web/src/lib/browse-url.ts
    - web/src/lib/rpc-errors.ts
    - web/src/lib/highlight.ts
    - web/src/lib/browse-state.ts
    - web/src/lib/components/browse/SourcePane.svelte
    - web/src/ambient-node.d.ts
    - web/tests/browse-tracer.test.ts
    - web/tests/browse-url.test.ts
    - web/tests/rpc-errors.test.ts
    - web/highlight_coverage_test.go
  modified:
    - web/src/routes/browse/+page.svelte
    - web/vite.config.ts

key-decisions:
  - "Added a highlight.js theme stylesheet import (highlight.js/styles/github.css) that no plan document named. highlight.js emits `hljs-*` class names only — no inline colour — so without a theme the markup is syntactically correct but visually indistinguishable from plain text. Caught by the plan's own mandated manual UAT step (a real browser), not by any automated grep. Chosen theme: github.css, matching the app's single (light-only, no dark-mode toggle) visual style and echoing BRW-09's GitHub-permalink framing elsewhere in this phase."
  - "Added a minimal ambient `declare const process` (web/src/ambient-node.d.ts) rather than installing @types/node. `pnpm check` (svelte-check) was not wired into any prior Taskfile target or CI job, so pre-existing, unrelated `process.env` reads in vite.config.ts (02-01/02-06) had never been type-checked before — this plan is the first to assert `pnpm check` exits 0. Installing a new npm package would require its own legitimacy checkpoint for a one-line fix; the ambient declaration avoids that without touching the pre-existing config code."
  - "The Go coverage guard adds a fourth structural check beyond the plan's literal three: a REVERSE binding from HIGHLIGHT_COVERAGE back to actual registration calls/alias keys. Without it, deleting a `hljs.registerLanguage(...)` call while leaving the (separately hand-maintained) coverage array untouched would not name the affected language in the failure output — only report a bare count below the floor. Verified live: with only the forward-binding check, the rust-deletion RED proof failed on 'found only 12 calls' without naming rust; with the reverse check added, the same deletion fails naming exactly `[rust]`."
  - "GetNodeDetailResponse's FILE mode carries no Node.language field (only SINGLE_DEF/MULTI_DEF do, via Node.language) — SourcePane.svelte derives the highlight language from the requested file's extension via a small local (unexported) extension map mirroring internal/indexer/languages_*.go's own registered Extensions lists, falling through to highlightSource's own honest plaintext-escape degrade for anything unmapped."

patterns-established:
  - "A Connect-error-classifying module (rpc-errors.ts) is constructed and tested against the REAL @connectrpc/connect ConnectError type and real generated detail-message schemas — never hand-shaped plain objects — so the test proves the mapper, not the test's own assumptions about the shape."
  - "TDD proof for already-correct behavior: when a task's tests all pass immediately against a prior task's implementation (no natural RED), substitute an invert-then-restore proof — temporarily reintroduce the exact bug class the code guards against, observe the specific test fail, restore byte-identically (git diff clean), re-run GREEN. Used twice in this plan (browse-url.ts's ordered-pairs serialization; highlight.ts's registration list) and once in 03-02."

requirements-completed: [BRW-02, BRW-06, NAV-01, NAV-04]

coverage:
  - id: D1
    description: "Opening /browse?file=<repo-relative-path> fetches that file through GetNodeDetail and renders its verbatim source, syntax-highlighted, in the browser — end to end, RED observed before any of the five source modules existed"
    requirement: BRW-02
    verification:
      - kind: unit
        ref: "web/tests/browse-tracer.test.ts#parseBrowseParams -> loadBrowseTarget -> SourcePane renders the file text with highlight markup"
        status: pass
      - kind: manual_procedural
        ref: "codegraph ui against this repo's own index; /browse?file=internal/query/node.go opened via agent-browser in a real Chrome; rendered text byte-identical (diff, 404/404 lines) to the file on disk; screenshot confirms visible colour (keywords red, strings blue, comments green) after the github.css theme fix"
        status: pass
    human_judgment: false
  - id: D2
    description: "/browse with no parameters renders an explicit idle state; a CodeNotFound/CodeInvalidArgument/CodeUnavailable+IndexingInProgress each render a distinguishable named state, never a blank pane"
    requirement: NAV-04
    verification:
      - kind: unit
        ref: "web/tests/browse-tracer.test.ts (idle state, failed state with not-found kind); web/tests/rpc-errors.test.ts (all four kinds mutually distinct, Set size 4)"
        status: pass
      - kind: manual_procedural
        ref: "live browser: /browse renders 'Search for a symbol or open a file to see its source.'; /browse?file=internal/does-not-exist.go renders 'Something went wrong: an internal error occurred' — correctly classified as the documented CodeInternal case (a nonexistent repo-relative path), never misclassified as not-found"
        status: pass
    human_judgment: false
  - id: D3
    description: "parseBrowseParams/serializeBrowseParams round-trip byte-identically in canonical BROWSE_PARAM_KEYS order; unknown params (including repeated and empty-valued) are preserved as ordered pairs, never collapsed or rejected; line/depth/limit are shape-checked only, never clamped"
    requirement: NAV-01
    verification:
      - kind: unit
        ref: "web/tests/browse-url.test.ts (11 cases: jumbled-order round-trip, two-orderings-identical, limit boundaries 0/1/1001, non-integer/empty/out-of-safe-range line all absent paired with a present integer, empty query, symbol+file adjacency, single/repeated/empty-valued unknown params, dotted-path round-trip)"
        status: pass
      - kind: manual_procedural
        ref: "invert-then-restore RED proof: temporarily collapsed repeated unknown keys via Object.fromEntries — the repeated-unknown-parameter test failed (1 failed, 10 passed; 'futureThing=1&futureThing=2' collapsed to 'futureThing=2'); restored byte-identically (git diff clean), re-ran GREEN (24/24)"
        status: pass
    human_judgment: false
  - id: D4
    description: "The set of language identifiers web/src/lib/highlight.ts covers equals indexer.RegisteredLanguageIDs() exactly (14 identifiers: c, cpp, csharp, go, java, javascript, kotlin, php, python, ruby, rust, swift, tsx, typescript — verified empirically this session), and the guard is proven to discriminate in both directions plus catch a stale array left behind by a deleted registration"
    requirement: BRW-06
    verification:
      - kind: unit
        ref: "web/highlight_coverage_test.go#TestHighlightRegistrationCoversRegisteredLanguages, #TestHighlightCoverageComparisonDiscriminates (2/2 PASS, go test ./web/ -run '^TestHighlight' -count=1 -v)"
        status: pass
      - kind: manual_procedural
        ref: "RED proof: deleted `hljs.registerLanguage('rust', rust);` from highlight.ts — FAIL, 'HIGHLIGHT_COVERAGE entries with no backing registerLanguage(...) call or alias-map key: [rust]' (exit 1). Restored byte-identically (git diff clean); re-ran 2/2 PASS (exit 0)"
        status: pass
    human_judgment: false
  - id: D5
    description: "Repository source bytes reach the DOM through exactly one path (highlightSource -> SourcePane's single @html site), never auto-detected, always escaped for unregistered languages"
    requirement: BRW-06
    verification:
      - kind: other
        ref: "rg -o '\\{@html' web/src/ --glob '!**/components/ui/**' | wc -l = 1; rg -o 'highlightAuto' web/src/lib/ | wc -l = 0 paired with rg -o 'hljs\\.highlight\\(' web/src/lib/highlight.ts | wc -l = 2 (>=1)"
        status: pass
    human_judgment: false
  - id: D6
    description: "The shallow-routing prohibition guard actually matches its target (corrected quoting AND -U line-orientation), proven to discriminate against two planted violations (single-line and wrapped imports), never present in this plan's own committed code"
    requirement: NAV-01
    verification:
      - kind: other
        ref: "planted single-line import: conjunct A=1, conjunct B=2 (non-zero, caught); planted wrapped multi-line import: conjunct A WITHOUT -U=0 (evaded, demonstrating why -U is load-bearing), conjunct A WITH -U=3, conjunct B=2 (caught either way); both plants removed: conjunct A=0, conjunct B=0, positive control ($app/state)=2 (>=1)"
        status: pass
    human_judgment: false

duration: ~35min
completed: 2026-08-29
status: complete
---

# Phase 3 Plan 4: Browse Tracer — URL to Highlighted Source Summary

**A URL carrying a file target renders its highlighted verbatim source end to end (browse-url.ts -> GetNodeDetail -> rpc-errors.ts -> highlight.ts -> SourcePane.svelte), with a Go set-equality guard binding the highlighter's 14-language coverage to the indexer's own registry, live-verified in a real browser with visible syntax colouring.**

## Performance

- **Duration:** ~35 min
- **Started:** 2026-08-28T20:25:54-04:00 (Task 1 commit)
- **Completed:** 2026-08-29T00:35:33Z
- **Tasks:** 3
- **Files modified:** 12 (10 created, 2 modified)

## Accomplishments

- `web/src/lib/browse-url.ts` — the one URL grammar (D-13): parses `symbol`/`file`/`line`/`depth`/`limit`/`q` by SHAPE only (never range-validates or bounds a value), preserves unknown params as ordered pairs (never collapsing a repeated parameter), serializes deterministically in `BROWSE_PARAM_KEYS` order with unknowns sorted after.
- `web/src/lib/rpc-errors.ts` — the one Connect-error classifier (D-04): `classifyRpcError` never throws, distinguishes `CodeUnavailable`+`IndexingInProgress` from a bare `CodeUnavailable`, tested against real `ConnectError`/`IndexingInProgress` fixtures.
- `web/src/lib/highlight.ts` — the one rendering path (D-19/BRW-06): `highlight.js/lib/core` plus exactly 13 per-language modules (verified: 13 `registerLanguage(...)` calls, 0 bare/`lib/common` imports, 0 auto-detection calls), an `HLJS_ALIAS_COVERAGE` declaring `tsx` -> `typescript`, and a machine-parseable `HIGHLIGHT_COVERAGE` literal array for the Go guard. Added a `github.css` theme import (a real gap caught only by manual browser UAT — see Deviations).
- `web/src/lib/browse-state.ts` — the load seam: maps `BrowseParams` to `GetNodeDetailRequest` field-for-field, returns a `BrowseTargetState` (`idle`/`loading`/`file`/`failed`), catches every rejection through `classifyRpcError`.
- `web/src/lib/components/browse/SourcePane.svelte` — the phase's single `@html` render site; decodes `SourceBlob.content` via `TextDecoder`, derives the highlight language from the file's extension (FILE mode carries no `Node.language` on the wire), renders a named state for every `BrowseTargetState` variant.
- `web/src/routes/browse/+page.svelte` — fills the Phase 2 placeholder; reads `page.url.searchParams` via `$app/state` reactively; imports no shallow-routing history exports.
- `web/highlight_coverage_test.go` — Go set-equality guard (package `web_test`, sited to avoid both a dependency-direction violation and `internal/indexer/extract_test.go`'s synthetic `go-dup` language corrupting the comparison), plus a planted-fixture discrimination test and a reverse-binding check beyond the plan's literal three structural assertions.

## Task Commits

Each task was committed atomically:

1. **Task 1: End-to-end tracer — open a file and read its highlighted source** — `aae14dc8` (feat)
2. **Task 2: Edge coverage for the URL grammar and the error mapper** — `bd70783b` (test)
3. **Task 3: Bind the highlighter's language coverage to the indexer's registry** — `9253d6da` (test)

**Plan metadata:** committed alongside this SUMMARY.

## Files Created/Modified

- `web/src/lib/browse-url.ts` — the URL grammar (D-13)
- `web/src/lib/rpc-errors.ts` — the Connect error classifier (D-04)
- `web/src/lib/highlight.ts` — the syntax-highlighting rendering path (D-19/BRW-06)
- `web/src/lib/browse-state.ts` — the params-to-RPC-to-state load seam
- `web/src/lib/components/browse/SourcePane.svelte` — the source-rendering component
- `web/src/ambient-node.d.ts` — minimal ambient declaration for `process.env` (deviation, see below)
- `web/src/routes/browse/+page.svelte` — the filled-in route
- `web/vite.config.ts` — `test.reporters: ['verbose']` so `pnpm test` names every file even on a full pass
- `web/tests/browse-tracer.test.ts` — the end-to-end tracer test
- `web/tests/browse-url.test.ts` — URL grammar edge coverage
- `web/tests/rpc-errors.test.ts` — error mapper edge coverage
- `web/highlight_coverage_test.go` — the Go set-equality guard

## Decisions Made

See `key-decisions` in frontmatter for full rationale. Summary:
- Added a `highlight.js/styles/github.css` theme import — caught by manual UAT, not any grep.
- Added a minimal ambient `process` declaration rather than a new npm dependency, to make `pnpm check` (never gated before this plan) pass against pre-existing, unrelated config code.
- Added a fourth (reverse-binding) structural check to the Go coverage guard, beyond the plan's literal three, closing a real gap the plan's own header comment warned about.
- Derived the FILE-mode highlight language client-side from the file extension, since `GetNodeDetailResponse`'s FILE mode carries no `Node.language` field on the wire.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] highlight.js markup had no visual colour — added a theme stylesheet**
- **Found during:** Task 1, manual UAT (live browser screenshot showed plain black text despite correct `hljs-*` classes in the DOM)
- **Issue:** highlight.js emits class names only; without a loaded theme CSS file, the classes carry no colour, making BRW-06's "syntax-highlighted" requirement functionally unmet despite the markup being structurally correct
- **Fix:** Added `import 'highlight.js/styles/github.css';` to `highlight.ts` (a side-effecting CSS import, code-split with the Browse route)
- **Files modified:** `web/src/lib/highlight.ts`
- **Verification:** Rebuilt to scratch dir, swapped into `web/build/` temporarily, ran real `codegraph ui` + a real browser via `agent-browser`; screenshot confirms visible colour (keywords, strings, comments distinctly coloured); restored `web/build/` byte-identically afterward
- **Committed in:** `aae14dc8` (Task 1 commit)

**2. [Rule 3 - Blocking] `pnpm check` failed on pre-existing, unrelated `process.env` reads**
- **Found during:** Task 1, running the plan's own `pnpm check` acceptance criterion
- **Issue:** `vite.config.ts`'s pre-existing (02-01/02-06) `process.env.VITEST`/`process.env.CODEGRAPH_WEB_BUILD_DIR` reads have no `@types/node` behind them; `pnpm check` (svelte-check) was never wired into any prior Taskfile target or CI job, so this had never been caught
- **Fix:** Added `web/src/ambient-node.d.ts` — a minimal ambient `declare const process: { env: Record<string, string | undefined> }` — rather than installing `@types/node`, which would trigger its own package-legitimacy checkpoint for a one-line fix
- **Files modified:** `web/src/ambient-node.d.ts` (new)
- **Verification:** `pnpm check` — 0 errors, 0 warnings, 345→347 files across the three tasks
- **Committed in:** `aae14dc8` (Task 1 commit)

**3. [Rule 2 - Missing Critical] The Go coverage guard's forward-only binding could not name a deleted registration**
- **Found during:** Task 3, performing the mandated RED proof (deleting `hljs.registerLanguage('rust', rust);`)
- **Issue:** With only the plan's literal three structural checks, deleting a registration call while leaving the separately-declared `HIGHLIGHT_COVERAGE` array untouched failed the test (via the `>= 13 registrations` floor) but did NOT name `rust` specifically — only reported a bare count below floor, undermining the RED proof's own acceptance criterion ("fail naming that exact identifier as missing")
- **Fix:** Added a reverse-binding check — every `HIGHLIGHT_COVERAGE` entry must be backed by an actual `registerLanguage(...)` call or a declared alias-map key — and changed the two floor checks from `t.Fatalf` to `t.Errorf` so all diagnostics run to completion in one pass
- **Files modified:** `web/highlight_coverage_test.go`
- **Verification:** Re-ran the rust-deletion RED proof — now fails naming exactly `[rust]`; restored byte-identically, re-ran 2/2 PASS
- **Committed in:** `9253d6da` (Task 3 commit)

---

**Total deviations:** 3 auto-fixed (2 missing-critical, 1 blocking).
**Impact on plan:** All three necessary for the plan's own acceptance criteria to be honestly satisfied (visible highlighting, a passing new gate, a RED proof that actually names its target). No scope creep — each fix is scoped to the exact gap found.

## TDD Gate Compliance

Task 1 (`type="tracer" tdd="true"`) followed the full RED-GREEN cycle: `browse-tracer.test.ts` was written and observed RED (`Failed to resolve import "$lib/browse-state"`, exit 1) before any of the five source modules existed, then built to GREEN.

Task 2 (`tdd="true"`) is a documented substitution, mirroring 03-02's precedent: all 24 new tests passed immediately against Task 1's already-correct implementation (no natural RED was possible — the behavior already existed). Per `tdd.md`'s own guidance ("Test doesn't fail in RED phase: Feature may already exist - investigate"), substituted an invert-then-restore proof: temporarily reintroduced the exact bug class D-13 warns against (an object-shaped collapse of repeated unknown params), observed the specific test fail, restored byte-identically, re-ran GREEN.

No `feat(03-04-...)` commit exists for Task 2 or Task 3 — both are pure test authorship over already-correct (Task 1's) or newly-authored-and-immediately-correct (Task 3's Go guard) production code, matching the shape 03-02 already established for this repository.

## Issues Encountered

None beyond the three recorded deviations above.

## User Setup Required

None — no external service configuration required. `highlight.js` was already installed and lockfile-visible from 03-01; no new npm packages were added this plan.

## Next Phase Readiness

- The four shared modules (`browse-url.ts`, `rpc-errors.ts`, `highlight.ts`, `browse-state.ts`) exist in their final architectural shape — later plans in this phase (03-05 disambiguation picker, 03-06 search surface, 03-07 URL writes/history, 03-08 truncation/permalink) extend them rather than adding parallel implementations.
- `web/highlight_coverage_test.go` is live in `task test:unit`'s default sweep (package `web` already had a Go file via `embed.go`) — no Taskfile change needed.
- The intended-RED `task web:drift` leg (opened 03-01) is still open and unaffected: SOURCE-half mismatch (28 source files now vs 22 at 03-01, since this plan adds source files), OUTPUT half MATCH (same digest as before this plan — `web/build/` was verified restored byte-identically after the manual-UAT temporary swap). Closes at 03-09 Task 3.
- No blockers for 03-05 or any other wave-2 plan.

---
*Phase: 03-browse-inspect-navigation*
*Completed: 2026-08-29*

## Self-Check: PASSED

- `test -f web/src/lib/browse-url.ts` → FOUND
- `test -f web/src/lib/rpc-errors.ts` → FOUND
- `test -f web/src/lib/highlight.ts` → FOUND
- `test -f web/src/lib/browse-state.ts` → FOUND
- `test -f web/src/lib/components/browse/SourcePane.svelte` → FOUND
- `test -f web/src/ambient-node.d.ts` → FOUND
- `test -f web/tests/browse-tracer.test.ts` → FOUND
- `test -f web/tests/browse-url.test.ts` → FOUND
- `test -f web/tests/rpc-errors.test.ts` → FOUND
- `test -f web/highlight_coverage_test.go` → FOUND
- `git log --oneline --all | grep -q aae14dc8` → FOUND
- `git log --oneline --all | grep -q bd70783b` → FOUND
- `git log --oneline --all | grep -q 9253d6da` → FOUND
- `pnpm --dir web test` → exit 0, 24/24 tests, names browse-tracer/browse-url/rpc-errors
- `cd web && pnpm check` → exit 0, 0 errors, 0 warnings
- `go test ./web/ -run '^TestHighlight' -count=1 -v` → exit 0, 2/2 PASS
- `task test:unit` → exit 0, 0 FAIL lines
- `go vet ./...` → exit 0
- `git diff --stat web/build` → empty (byte-identical to HEAD after manual UAT)
