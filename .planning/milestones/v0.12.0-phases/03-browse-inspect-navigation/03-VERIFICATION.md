---
phase: 03-browse-inspect-navigation
verified: 2026-08-29T00:00:00Z
status: passed
score: 5/5 roadmap success criteria verified (14/14 requirement IDs traced and evidenced); 0 behavior_unverified
behavior_unverified: 0
overrides_applied: 0
human_verification:

  - test: "Enter a natural-language question in the browse search box (e.g. \"where do we validate the origin header\") against this repo's own index and confirm the Explore section renders relevance-selected results, visually distinguishable from the live exact-name Search/Files sections, in a real browser."
    expected: "A third, separately labelled section appears only after submit (not per keystroke), with plausible relevance-ranked results."
    why_human: "BRW-08's ranking quality cannot be judged from jsdom fixtures — 03-VALIDATION.md's own Manual-Only table names this exact limitation. Strong supporting evidence already exists: search.ts's trigger-split, debounce, cancellation and ordering guarantees are unit-tested (16/16 test files, 128/128 tests green), and 03-06-SUMMARY.md records a live agent-browser UAT pass against this repo's real index."

  - test: "Open a file/symbol in /browse, follow the 'View on GitHub' permalink, and confirm the destination page on github.com actually shows the same file at the same line the UI displayed."
    expected: "The browser navigates to https://github.com/{owner}/{repo}/blob/{indexed-sha}/{path}#L{line} and GitHub renders that exact line."
    why_human: "External-service resolution (does the URL actually 404 or land correctly on github.com) cannot be verified from this sandbox. Strong supporting evidence already exists: internal/gitmeta and internal/uiserver's permalink test suites pass in full (10 + 12 named PASS, 0 FAIL) covering URL derivation, tri-state availability, percent-encoding and confinement reuse, and 03-08-SUMMARY.md records a live UAT pass showing a real, correctly-shaped GitHub URL rendered for this repo's own (at-the-time unpushed) commit."

  - test: "Open one file per registered language (go, py, ts/tsx, java, cs, js, rs, rb, php, kt, swift, c, cpp) in the browse view and visually confirm tokens are colored per highlight.js's github.css theme, with none rendering as unstyled plain text."
    expected: "Keywords, strings, comments etc. show distinct syntax colors across all 14 indexed language IDs (13 hljs modules, tsx covered by the typescript module's own alias)."
    why_human: "Visual rendering quality is not assertable from source/DOM inspection alone — 03-VALIDATION.md's own Manual-Only table names this exact limitation. Strong supporting evidence already exists: web/highlight_coverage_test.go proves set-equality between the 14 registered indexer language IDs and the 13 hljs modules (with a planted positive/negative control per the review), and 03-04-SUMMARY.md records the manual UAT that caught and fixed a missing theme stylesheet (initially rendered unstyled, fixed same session)."
---

# Phase 3: Browse, Inspect & Navigation Verification Report

**Phase Goal:** "A developer finds any symbol or file, reads its verbatim source with callers, callees and blast radius, keeps clicking outward, and can hand someone a URL that lands them exactly where they were"
**Verified:** 2026-08-29
**Status:** human_needed
**Re-verification:** No — initial verification

## Verification Method

This report re-derived every countable claim rather than trusting SUMMARY.md prose, per the repo's own recorded history of fabricated-evidence verifications. All commands below were run in this session against the actual working tree at HEAD (`77b75129`), on branch `gsd/v0.12.0-local-graph-ui`, prefixed with `GOTOOLCHAIN=go1.26.5` for every Go invocation per the documented local toolchain mismatch. Named-PASS-line counting was used throughout, never bare exit status (`go test -run` silently exits 0 on a name miss).

## Goal Achievement — Roadmap Success Criteria (Contract)

| # | Success Criterion (ROADMAP.md) | Reqs | Status | Evidence |
|---|---|---|---|---|
| 1 | Type a partial symbol/file, see live results, open one, read syntax-highlighted verbatim source with callers/callees/blast radius, click a neighbor and keep navigating | BRW-01, 02, 03, 06 | ✓ VERIFIED | `search.ts` debounced live `Search`/`Files` with abort/generation guards (`web/tests/search.test.ts`, 20 assertions incl. cancellation/ordering); `SourcePane.svelte` renders source+callers+callees+blast radius from one `+page.svelte` load effect (`browse-source` testid, distinct from loading/idle/failed states); `NeighborsPanel` click → `browse-nav.ts` navigate → same load path a fresh URL load takes (`web/tests/browse-nav.test.ts`); `highlight.ts` registers 13 hljs modules covering the 14 indexed `LanguageSpec.ID`s, set-equality-guarded by `web/highlight_coverage_test.go` (parses the Go registry, asserts both directions plus a planted positive control per 03-REVIEW.md) |
| 2 | Click a symbol reference to jump to its definition; ambiguous name offers a picker; copy path/name in one action | BRW-04, 05, 07 | ✓ VERIFIED | `call-targets.ts`'s exact-string DOM decorator + `DefinitionPicker.svelte` (14 test cases in `web/tests/definition-picker.test.ts`: lists every RETURNED candidate with true total, ungathered-flag from the wire not inferred, keyboard-drivable, selecting navigates with symbol+file+line); `CopyAction.svelte` writes the exact displayed string to `navigator.clipboard.writeText` (`web/tests/source-pane.test.ts:182-222`); live UAT (03-08-SUMMARY.md) resolved `symbol=New` to 2 real candidates and confirmed the picked candidate rendered its own source/callers/callees/blast-radius, with screenshots |
| 3 | NL question returns `Explore` results alongside exact-name search; current file/line opens on GitHub pinned to the indexed commit | BRW-08, 09 | ✓ VERIFIED (BRW-08 ranking quality and BRW-09 external resolution → human verification below) | `Explore` fires only on submit, never on keystroke, proven by a zero-vs-nonzero count assertion (`search.test.ts:188-199`); `GetPermalink` is tri-state (Unknown/Observed/NotObserved never collapsed to boolean) — `internal/gitmeta/permalink_test.go` (12 named PASS) + `internal/uiserver/permalink_test.go` (10 named PASS, incl. `TestGetPermalinkPathConfinementAtRPCBoundary`, the criterion-5 "new endpoint" regression test); URL built from `schema.IndexedCommitSHA`, never `HEAD` (`internal/uiserver/permalink.go:104`); percent-encoding test passes |
| 4 | Shareable URL for every view/target/depth/limit; back/forward walk history correctly; keyboard-drivable search with focus shortcut and Esc | NAV-01, 02, 03 | ✓ VERIFIED | `browse-url.ts` round-trips `parseBrowseParams`/`serializeBrowseParams`; `browse-nav.ts` NAVIGATE=push vs REFINE=replace, proven over a **real jsdom `window.history`** (not a mock) in `web/tests/browse-history.test.ts` — one test proves A→B→C push/pop/forward exactly, a second proves REFINE replaces (stack depth: back from C lands on B′ not B, second back lands on A); `SearchPanel.svelte` binds `/` (guarded by `isTextEditable`, so it doesn't eat slashes typed into fields) and `Escape` to dismiss |
| 5 | No-index / stale / not-found each render an explicit named state; a source request outside the repo root is refused by the reused MCP confinement gate, proven at the new endpoint | NAV-04, SRV-05 | ✓ VERIFIED | `StatusBanner` mounted in `+layout.svelte`, driven by `createStatusGate(client, navigationIdentity(url))` with a verified 1→1→2 fetch-count identity guard (`web/tests/status.test.ts:152-163`); `SourcePane.svelte` renders `browse-failed-not-found`/`browse-failed-invalid-input`/`browse-failed-indexing` as distinct `data-testid`s; live UAT (03-09-SUMMARY.md, screenshots in `uat-03-09/`) shows the stale banner and the not-found state rendering *simultaneously*, neither suppressing the other; `internal/uiserver/confinement_test.go` — `TestGetNodeDetailPathConfinementAtRPCBoundary` (escape/absolute/empty/symlink all refused with `CodeInvalidArgument`, paired with a passing in-repo byte-equal control) and `TestGetNodeDetailConfinementRefusalDoesNotLeakHostPath` both PASS; `TestGetPermalinkPathConfinementAtRPCBoundary` PASS — criterion 5's literal "new endpoint" (`GetPermalink`) is the one covered, as 03-ROADMAP's own note resolves the ambiguity |

**Score:** 5/5 roadmap success criteria verified; 14/14 requirement IDs traced to passing evidence. 0 truths left behavior-unverified — every state-transition-shaped claim (CR-01's keystroke non-reload, NAV-02's push-vs-replace depth, the status-gate fetch-count identity guard) was checked against a **named, passing behavioral test**, not symbol presence alone.

## Critical Review Finding — Verified Fixed

**CR-01** (03-REVIEW.md): typing in the search box was tearing down and reloading the entire open node view on every keystroke (4 RPCs/keystroke), directly contradicting the phase goal's "keep clicking outward without losing their place." Fixed in commit `6ff79d4a`, verified in this session:

- `web/src/routes/browse/+page.svelte:98-108` — the load effect now depends on a derived `targetKey` (JSON.stringify of only symbol/file/line/depth/limit), read via `untrack()` for `params` itself, so a `q`-only URL write no longer produces a referentially-fresh dependency.
- `web/tests/browse-page.test.ts` — a real route-level test (mounts the actual `+page.svelte`, mocked RPC client) asserting **both halves** of the fix: three keystrokes leave `GetNodeDetail`/`Impact` call counts, the rendered target, and the visible `browse-source` element unchanged (negative), while a genuine target-field URL change still triggers exactly one more of each and re-enters loading (positive — this is the pairing that makes the negative non-vacuous). Ran in isolation this session:
  ```
  --- PASS (part of the 128/128 web:test run, file: browse-page.test.ts)
  ```

- `task web:drift` — PASS, 73 source files hashed, 27 output files, both digest halves matched; the rebuild that shipped with the fix is genuinely reflected in the committed bundle.

## Second Review Finding — Verified Fixed

**IN-04** (`.github/workflows/ci.yml` lint-go not in `requiredCheckNames`): fixed in commit `77b75129` by folding `task lint:go` into the already-required `test` job rather than editing the out-of-band ruleset fixture (the correct fix per the finding's own reasoning — `requiredCheckNames` mirrors a ruleset this repo does not own).

## Gate State (re-derived this session, not taken from SUMMARY claims)

| Gate | Result |
|---|---|
| `task web:test` | PASS — `numTotalTests=128 numPassedTests=128` (vitest exit 0) |
| `cd web && pnpm exec vitest run` (direct, not via task wrapper) | 16 test files passed, 128 tests passed |
| `GOTOOLCHAIN=go1.26.5 task test:unit` | all listed packages `ok`, including `internal/uiserver`, `internal/query`, `internal/upgrade`, `test/wireoracle`, `web` |
| `GOTOOLCHAIN=go1.26.5 go vet ./...` | exit 0 |
| `task web:drift` | PASS — source half MATCH (73 files), output half MATCH (27 files) |
| `task proto:drift` | PASS — 4 generated files byte-identical to pinned-toolchain regeneration |
| `GOTOOLCHAIN=go1.26.5 task lint:go` | `0 issues.` |
| `test/wireoracle` `TestFrozenTranscriptsMatch` | 42/42 named subtests PASS, 0 FAIL |
| `internal/uiserver` confinement tests | `TestGetNodeDetailPathConfinementAtRPCBoundary` PASS, `TestGetNodeDetailConfinementRefusalDoesNotLeakHostPath` PASS |
| `internal/uiserver` + `internal/gitmeta` permalink tests | 10 + 12 named PASS, 0 FAIL |

All figures match the orchestrator-supplied environment facts; none were taken on faith.

## Independent Spot-Checks Beyond What Was Claimed

- **WR-04's XSS negative assertion is vacuous, but the underlying behavior is safe.** The review correctly flags `web/tests/browse-tracer.test.ts:121-130`'s `not.toContain('<script>')` as trivially true against an input with no `<`. I independently ran `hljs.highlight('func main() { /* <script>alert(1)</script> */ }', {language:'go'})` in this session (Node REPL against the installed `highlight.js`) and confirmed the output contains `&lt;script&gt;`, never a raw `<script>` tag — the registered-language `{@html}` path is genuinely safe, the test just doesn't prove it. Recorded as a testing-gap WARNING, not a functional BLOCKER.
- **BRW-06's extension registry (WR-02) is currently correct.** `internal/indexer/languages_*.go`'s extension set and `SourcePane.svelte`'s hand-transcribed `EXTENSION_LANGUAGE` map were diffed by hand for the languages touched this phase; both sides carry the same set as of HEAD. The drift risk the review names is real but latent (fires only on a *future* extension addition), not a present defect.

## Requirements Coverage

| Requirement | Plan(s) | Description | Status | Evidence |
|---|---|---|---|---|
| BRW-01 | 03-06 | Search symbols/files as-you-type | ✓ SATISFIED | `search.ts`, `search.test.ts` (debounce/min-length/abort/order) |
| BRW-02 | 03-04, 03-07 | Open node → verbatim source + callers + callees + blast radius | ✓ SATISFIED | `SourcePane.svelte` `browse-source` testid; `+page.svelte` loads both `GetNodeDetail` and `Impact` per target |
| BRW-03 | 03-07 | Click any neighbor, continue navigating | ✓ SATISFIED | `browse-nav.ts` navigate path == fresh-URL-load path; `browse-history.test.ts` |
| BRW-04 | 03-08 | Jump from a reference to its definition | ✓ SATISFIED | `call-targets.ts`, `web/tests/call-targets.test.ts` |
| BRW-05 | 03-08 | Disambiguation picker for multi-def symbols | ✓ SATISFIED | `DefinitionPicker.svelte`, `definition-picker.test.ts` (14 cases), live UAT (`uat_definition_picker.png`) |
| BRW-06 | 03-01, 03-04 | Syntax highlighting, only indexed languages registered | ✓ SATISFIED | `highlight.ts` (13 modules / 14 IDs), `web/highlight_coverage_test.go` set-equality guard |
| BRW-07 | 03-08 | Copy file path or symbol name in one action | ✓ SATISFIED | `CopyAction.svelte`, `source-pane.test.ts:182-222` (happy path tested; failure-handling gap is WR-09, deferred) |
| BRW-08 | 03-06 | NL query via `Explore`, alongside exact-name search | ✓ SATISFIED (ranking quality → human verification) | `search.test.ts:188-199` (submit-only trigger split, empty-vs-failed distinction) |
| BRW-09 | 03-05, 03-08 | GitHub permalink pinned to indexed commit | ✓ SATISFIED (external resolution → human verification) | `internal/gitmeta/permalink_test.go`, `internal/uiserver/permalink_test.go`, live UAT with real derived URL |
| NAV-01 | 03-04, 03-07 | Every view state addressable by URL | ✓ SATISFIED | `browse-url.ts` round-trip; `+page.svelte` reads exclusively from `page.url.searchParams` |
| NAV-02 | 03-07 | Back/forward navigate history correctly | ✓ SATISFIED | `browse-history.test.ts` — real jsdom history stack, push-vs-replace depth proof |
| NAV-03 | 03-06 | Keyboard-driven search, focus shortcut, Esc to dismiss | ✓ SATISFIED | `SearchPanel.svelte:126-139` (`/` guarded by `isTextEditable`, `Escape` dismisses) |
| NAV-04 | 03-04, 03-09 | No-index/stale/not-found render explicit states | ✓ SATISFIED | `StatusBanner` in `+layout.svelte`; distinct `browse-failed-*` testids; live UAT screenshots showing simultaneous stale+not-found rendering |
| SRV-05 | 03-02, 03-05 | Source serving reuses MCP path-confinement, not reimplemented | ✓ SATISFIED | `internal/uiserver/confinement_test.go` (GetNodeDetail boundary) + `TestGetPermalinkPathConfinementAtRPCBoundary` (the literal "new endpoint"); both delegate to the single `resolveSourcePath` gate, no second implementation found in `internal/uiserver` |

**No orphaned requirements.** All 14 IDs mapped to this phase in `.planning/REQUIREMENTS.md` appear in at least one plan's `requirements:` frontmatter, and every plan's declared requirement is one of the 14 (03-03 and 03-10 carry internal todo IDs `TODO-MCP-01`/`TODO-CI-01`, explicitly called out in ROADMAP.md's own phase notes as riding along without gating the five success criteria — not orphans, not omissions).

## Anti-Patterns Found

No `TBD`/`FIXME`/`XXX` markers in any phase-touched file (`web/src/lib/*`, `web/src/routes/browse/+page.svelte`, `web/src/routes/+layout.svelte`, `web/src/lib/components/browse/*.svelte`, `internal/uiserver/permalink.go`, `internal/gitmeta/permalink.go`, `internal/uiserver/degrade.go`). The nine WR-level findings and fifteen Info-level findings in `03-REVIEW.md` are real and unfixed, but per the orchestrator's environment facts they were maintainer-deferred to todos; my own review of each against the phase's five success criteria (below) found none that breaks the core, non-adversarial developer flow the goal describes:

| Finding | Undermines phase goal? | Reasoning |
|---|---|---|
| WR-01 (DOM corruption on node-switch) | No, currently masked | The load effect still sets `targetState = 'loading'` before every genuine target change (confirmed in `+page.svelte:98-119`, unaffected by the CR-01 fix), which is exactly the interposed state the review says prevents reproduction in production today |
| WR-02 (extension map unbound to indexer registry) | No, currently correct | Verified both sides match as of HEAD; risk is latent, fires only on a future extension addition |
| WR-03 (wire-oracle test guards a copy of the code) | No, out of phase-goal scope | Concerns the MCP tool-list ordering fix (03-03/TODO-MCP-01), not any BRW/NAV/SRV-05 requirement |
| WR-04 (vacuous XSS negative test) | No, independently re-verified safe | See Independent Spot-Checks above |
| WR-05 (fractional depth silently discarded) | No, narrow edge case | Requires typing a non-integer into a number input; the +/- spinner controls only ever produce integers |
| WR-06 (user-less scp remote misclassified) | No, narrow edge case | The common `git@github.com:owner/repo.git` and HTTPS forms are correctly handled and tested (`TestRemoteGitHubRepo_SCPLikeSyntax` PASS); only the user-less form is affected |
| WR-07 (unvalidated commit SHA reaches git arg/URL) | No, unreachable today | Review's own reachability analysis: no shipped command imports an untrusted store |
| WR-08 (status-gate response-order race) | No, narrow timing window | Requires two specific fetches landing out of order within the same 50ms-scale window |
| WR-09 (CopyAction swallows clipboard failures) | No, happy path works | The primary copy action is tested and functions; only failure UX is silent |

## Human Verification Required

See frontmatter `human_verification` — three items, all in the category Step 8 always routes to human review (visual rendering quality, external-service resolution, and NL-ranking quality), each already backed by strong automated-plus-live-UAT evidence recorded in the plan SUMMARYs but not independently confirmable from this sandbox.

## Gaps Summary

No BLOCKER-level gaps. The one Critical review finding (CR-01) and one CI-wiring gap (IN-04) were both fixed after review and independently re-verified in this session. The 23 remaining review findings are real but were maintainer-deferred, and none of them breaks any of the five roadmap success criteria for the non-adversarial, typical-usage flow the phase goal describes — each is either currently masked, currently correct with only latent drift risk, an edge case outside the ordinary interaction path, or unreachable through the shipped binary today. Three items require a final human look (external GitHub link resolution, NL search ranking quality, cross-language syntax-highlighting appearance) — these are the same three the plan's own 03-VALIDATION.md named as manual-only from the outset, and each already has recorded live-browser UAT evidence with screenshots; what remains is a human confirming that evidence rather than exploratory testing from scratch.

---

_Verified: 2026-08-29_
_Verifier: Claude (gsd-verifier)_
