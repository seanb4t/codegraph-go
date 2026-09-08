---
phase: 05
slug: file-package-graph-view
status: verified
# threats_open = count of OPEN threats at or above workflow.security_block_on (high)
threats_open: 0
asvs_level: 1
created: 2026-08-31
---

# Phase 5 - Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

Register origin: register_authored_at_plan_time: true - all 8 plans carry a
<threat_model> block. Independently re-parsed this session (not taken from the
orchestrator summary): 51 distinct threat IDs, zero duplicates (T-05-01 through T-05-50 plus
T-05-SC, verified with a duplicate check returning empty). Severity breakdown, counted directly
from the eight threat_model tables: 27 high, 17 medium, 7 low. Disposition: 44 mitigate, 7
accept - matching the orchestrator inventory on disposition even though the independently
recomputed severity split (27/17/7) differs by one from the orchestrator quoted figure
(28/16/7), recomputed twice by table-column extraction with identical results, so the 27/17/7
figure is what this audit threats_open gate is computed against.

Verification depth is ASVS L1 (grep-level mitigation presence). Every zero-count check below is
paired with a positive control per rule 84d1gfpywd, and every threat evidence was checked
against the actual working tree at HEAD (c117a898), not accepted from plan or SUMMARY text.

Blocking threshold: security_block_on: high. All 27 high-severity threats are mitigate; zero
high-severity threats were accepted. HIGH intersect ACCEPT is empty - independently confirmed
by cross-referencing the severity and disposition columns extracted from all eight threat_model
tables (all 7 accepts are medium or low).

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| indexed record set to Engine.FileGraph() in-memory aggregation | untrusted repository content (paths, symbol names) becomes graph identity | file paths, symbol names |
| corpora/graph-render-threshold.json to GRF-01 recorded verdict | a human decision crosses into an automated pass/fail comparison | measurement thresholds |
| browser to FileGraph (12th rpc) / FileSymbols (13th rpc) over loopback ConnectRPC | two new read rpcs on the UI surface; FileSymbols.path is caller-supplied | integers absent, one repo-relative path (FileSymbols only) |
| FileGraphResult / FileSymbolsResult to browser | repository-derived file paths, language strings, symbol names cross the wire | identifiers, paths |
| npm registry to web/package.json / pnpm-lock.yaml to committed web/build/ to signed binary | three new runtime deps (cytoscape, cytoscape-elk, elkjs) plus one devDependency (@playwright/test) enter the artifact | executable JS |
| served page to browser worker/network surface, under CSP | elkjs wants a Web Worker; default-src self carries no worker-src | none (blocked by policy) |
| locked threshold artifact to live measurement session to recorded verdict | a pre-committed human decision is judged against numbers produced by an interactive browser session | latency/frame-time/byte-count metrics |
| a clicked file node to FileSymbols request | a path that originated in a server wire response is round-tripped back to the server | repo-relative path |
| decoded wire response to client element builder (rollupToElements) | untrusted-in-shape data (paths, counts, cycle ids) decides what is rendered and how much | node/edge counts, cycle ids |
| route to renderer, across the GraphCanvas prop/event seam | the swappable-renderer boundary GRF-05 requires; a direct reach-around would defeat it | plain-data props only |

---

## Threat Register

All 51 threats CLOSED. Every mitigate disposition below was verified against the actual
working tree this session - code read, tests run, or artifacts inspected directly - not
accepted from plan or SUMMARY prose.

### High severity - the blocking set (27, all mitigate, all verified)

| Threat ID | Component | Mitigation | Verified | Status |
|-----------|-----------|------------|----------|--------|
| T-05-01 | threshold artifact tampering | committed alone, before any measurement exists | corpora/graph-render-threshold.json has exactly 1 commit (2fb27746) across the entire phase; confirmed ancestor of the observation commit (ancestor check confirms true) | closed |
| T-05-02 | package pseudo-node phantom-file rollup | explicit kind+empty-path exclusion, regression test against this repo own index | filegraphKindPackage equals the string package, excluded at traverse.go:188; ExcludedPackageNodes field populated and asserted greater than zero; TestFileGraphAgainstThisRepositoryIndex PASS (measured ExcludedPackageNodes=43) | closed |
| T-05-06 | PASS verdict with no auditable judgement record | two separate committed artifacts, ordering machine-checkable via git log | threshold commit 2fb27746 confirmed ancestor of observation commit 1cb8c71e | closed |
| T-05-07 | UIService gaining a dispatchable mutating method | positive set-equality fixture plus one entry, count 11 to 12; negative verb fixture untouched | wantUIServiceMethods gained FileGraph; mutatingVerbs block absent from the diff entirely (confirmed: diff of readonly_test.go across the phase touches only additive doc comments, wantUIServiceMethods, the fixture-length constants and uiProtoFieldNumbers - zero lines inside mutatingVerbs) | closed |
| T-05-08 | unbounded FileGraph response exceeding transport ceiling | no caller-controlled fan-out; measured, not estimated, against guava | FileGraph() takes zero parameters (traverse.go:172); TestFileGraphResponseSizeGuava exists and is corpus-gated (SKIP without CODEGRAPH_GUAVA_STORE); the actual measured value 3,713,528 bytes is recorded in corpora/graph-render-observations.json, well under the 16,777,216-byte transportSendMaxBytes ceiling | closed |
| T-05-11 | wire shape frozen without a recorded, durable human decision | blocking-human checkpoint before codegen; frozen field numbers made durable via TestUIProtoFieldNumbersAreStableAndUnique | 05-02-SUMMARY.md records verbatim approve-as-proposed; uiProtoFieldNumbers fixture extended (16 new entries); the field-number stability test PASS | closed |
| T-05-SC | cytoscape/cytoscape-elk/elkjs entering the signed binary | blocking-human package-legitimacy checkpoint before install; exact pins on direct deps | 05-03-SUMMARY.md records maintainer approval with live registry figures (14 years old, 15.5M weekly downloads, 0 deps); web/package.json pins cytoscape at 3.34.2 / cytoscape-elk at 2.3.0 with no caret | closed |
| T-05-13 | malicious layout dependency executing code via a Web Worker | no worker configured; CSP default-src self with no worker-src blocks any attempt | spa.go:99 CSP confirmed no worker-src; new-Worker occurrence count over web/src = 0, positive-controlled by a cytoscape-import search = 1 (proves the search tooling functions) | closed |
| T-05-16 | monorepo-scale element array locking the main thread during layout | GRF-01 measured, pre-locked interaction-latency threshold; halt-and-escalate on miss | The binding measurement FAILED first (graph-render-observations.json: seamReady exceeded its 60000ms deadline); the phase correctly halted per the committed remedy path and 05-08 re-measured the collapse-by-default remedy, which PASSED all four bars (graph-render-observations-collapsed.json: verdict PASS) | closed |
| T-05-18 | threshold artifact edited during/after measurement to force PASS | 05-04 touches no threshold file; ancestor-commit assertion | corpora/graph-render-threshold.json still has exactly 1 commit total, and it predates every 05-04/05-08 commit; graph-measure.mjs (05-04 own creation) has exactly 1 commit in its history and was never touched by 05-08 either | closed |
| T-05-19 | PASS recorded for a measurement that silently did not run | fail-closed comparator plus independent verify command plus machine-checked frameSampleCount floor | graph-verdict.mjs: an absent, non-numeric or non-finite metric maps to status measurement-failed, verdict FAIL (Number.isFinite guard present) | closed |
| T-05-20 | a number presented as measured that was estimated or off-corpus | every value carries a method field; comparator asserts bindingObservation.corpus equals the locked threshold corpus | graph-verdict.mjs: a corpus repo/sha mismatch against the threshold corpus throws before judgement | closed |
| T-05-25 | client-derived cycle highlight disagreeing with the server answer | test: adjacency closes a loop, wire cycle fields unset yields zero cycle classes | graph-cycles.test.ts exact test present and passing (ran this session) | closed |
| T-05-27 / T-05-38 | committed bundle diverging from source (mid-phase and end-of-phase) | build task re-run, two-part drift guard re-verified each time | task web:drift run live this session, PASS - hashed 108 source files, manifested 32 output files, both digests matched | closed |
| T-05-28 | caller-supplied FileSymbols path escaping the repository root | Engine existing confinement called FIRST, before any store read | filesymbols.go:56: ValidateRepoRelativePath is the first statement in FileSymbols, before IterateNodes(); path-escape and absolute-path rejection tests PASS (ran this session) | closed |
| T-05-29 | read-only surface gaining a dispatchable mutating method (13th rpc) | same positive/negative fixture discipline as T-05-07 | wantUIServiceMethods count 12 to 13 confirmed in diff; mutatingVerbs block absent from diff; the method-set-exactly-the-read-set test PASS (ran this session, inspected 40 messages and 162 fields) | closed |
| T-05-30 | generated file with enormous symbol count threatening the transport ceiling | single-owner query.MaxFileSymbols cap; total plus truncated ride with the capped list; wire-size test | filesymbols.go:20 const MaxFileSymbols = 2000, applied at line 94; the capped-response-fits-transport-ceiling test PASS (ran this session) | closed |
| T-05-33 | a file path round-tripped from the graph into FileSymbols, escaping the root | server re-validates regardless of origin; no second client-side implementation | same ValidateRepoRelativePath call path as T-05-28, one confinement, applied to every caller | closed |
| T-05-35 | partial symbol list presented as complete | truncation flag plus true total ride on the response and render as a visible statement | +page.svelte renders symbol count of total, gated on the truncated flag | closed |
| T-05-40 | corpora/graph-render-threshold.json edited during 05-08 re-measurement | named prohibition, diff-exit-code gates, commit-count-equals-one asserted | confirmed: 1 commit total for the file across the entire phase (including all of 05-08) | closed |
| T-05-41 | corpora/graph-render-observations.json overwritten to hide the recorded FAIL | comparator gains an out-path flag so re-measure writes a NEW artifact | graph-verdict.mjs has an out flag; graph-render-observations.json (the FAIL) and graph-render-observations-collapsed.json (the PASS) both exist as separate files | closed |
| T-05-42 | a re-measured verdict not actually produced by the comparator | verdict fields: generatedBy, recorded browser identity, judged-metric provenance, failure reasons | graph-verdict.mjs sets generatedBy to its own script path; the collapsed observations file tail confirms generatedBy, browserIdentity, and a note field with provenance | closed |
| T-05-43 | progressive expansion in +page.svelte exceeding a safe render scale | named ceiling constant, planned-count predicate consulted BEFORE expansion, readable refusal | EXPANSION_NODE_CEILING equals 1500 (file-graph-transform.ts:125); +page.svelte checks plannedNodeCount against the ceiling and sets a refusal message before mutating state | closed |
| T-05-44 | the measurement seam in GraphCanvas.svelte republishing across an expansion | publish-once flag; construction separated from application | metricsPublished boolean flag (GraphCanvas.svelte:224,263-264); a deep-equality test across a replace() call confirms metricsCallCount stays 1 and capturedMetrics deep-equals the first, ran this session, PASS | closed |
| T-05-45 | new package/lockfile entries in 05-08 (no install authorized) | named prohibition; diff asserted empty in all three implementation tasks | git log for web/package.json and web/pnpm-lock.yaml shows the most recent touch is b71d3876 (05-04), zero commits from 05-08 touch either file | closed |
| T-05-46 | the measurement seam (graph-measure.mjs) modified so the re-measure gesture disagrees with the FAIL | file named unmodifiable, clean diff asserted every task | confirmed: exactly 1 commit (b71d3876, 05-04) in the file entire history, 05-08 never touched it | closed |

### Medium and low severity (24, closed)

| Threat ID | Severity | Disposition | Verification |
|---|---|---|---|
| T-05-04 | medium | mitigate | Iterative Tarjan SCC, explicit work stack (scccFrame), zero recursive self-calls (the only occurrence of the function name in the file is its own definition); the deep-chain-does-not-recurse test (10,000-node chain) PASS |
| T-05-09 | medium | mitigate | TestFileGraphRequestPathIsIgnored present and passing; FileGraphRequest.path field is declared but never read by the v1 handler |
| T-05-21 | medium | mitigate | Always-write finally block, per-operation deadlines, write-before-teardown ordering, structurally-proven coverage via OPS_KEYS (6-entry frozen enumeration); the never-settling sweep test drives each of the 6 operations and asserts collected labels set-equal OPS_KEYS, ran this session, PASS |
| T-05-14, T-05-36 | medium | mitigate | Canvas renders labels as text, never DOM; occurrences of innerHTML/outerHTML/at-html over the graph components directory equal 0, positive-controlled by data-testid equal 1 in the same scope, and by the known SourcePane at-html site elsewhere in the tree |
| T-05-17, T-05-24, T-05-37 | medium | mitigate | Single-importer discipline: exactly 1 file (GraphCanvas.svelte) imports cytoscape; the route file has 0 cytoscape references, positive-controlled by 11 GraphCanvas references in the same file; T-05-24 second criterion (class-list stripping) has a dedicated test |
| T-05-39 | medium | mitigate | at-playwright/test at 1.62.1 exact-pinned, no caret; strictDepBuilds true and allowBuilds empty in web/pnpm-workspace.yaml confirmed live |
| T-05-23 | medium | mitigate | Shared table renders via Svelte default text escaping; zero at-html sites in the graph/table rendering path for this phase |
| T-05-32 | medium | mitigate | FileSymbols reads IterateNodes() only, confirmed by reading filesymbols.go end to end; no file-read call present |
| T-05-34 | medium | mitigate | graph-expand.test.ts CLICK ONE/TWO/THREE tests (expand, collapse, re-expand) assert the fileSymbols call count stays 1 across all three, ran this session, PASS |
| T-05-47 | medium | mitigate | expandedDirs/expandedFiles are Set-string-typed (+page.svelte:42,59); zero classList occurrences in the route file |
| T-05-48 | medium | mitigate | buildEdgeElements is a single for-of loop with map lookups (file-graph-transform.ts:267-296), no nested scan |
| T-05-49 | medium | mitigate | Provenance note present and non-empty in graph-render-observations-collapsed.json, naming both the rendered view and the authorizing remedy |
| T-05-05 | low | accept | TestFileGraphRepoRelativePaths asserts a non-zero node-count floor first, then checks every path is non-absolute and non-drive-letter, ran this session, PASS |
| T-05-10 | low | accept | Repo-relative by construction (same guarantee as T-05-05); loopback-bound listener with exact-match Origin/Host inherited from SRV-02/03 (verified in Phase 1/4 audits, unchanged) |
| T-05-15 | low | accept | Measurement global documented (window.codegraphFileGraphMetrics), also dispatched as a CustomEvent so tests do not depend on the global; carries 4 numbers only, on a loopback-bound single-user page |
| T-05-22 | low | mitigate | Both observation artifacts key the corpus by repo and sha only; host-path pattern search over both files returns 0 matches |
| T-05-26 | low | mitigate | edge-kind-columns.ts tests assert exactly one row per present kind and no absent-kind row |
| T-05-31 | low | accept | Raised explicitly as sub-decision 1 at 05-06 Task 1 blocking-human checkpoint; maintainer confirmed shared Node message reuse rather than a narrower shape, recorded verbatim in 05-06-SUMMARY.md |
| T-05-50 | low | accept | Directory paths/node ids already rendered by existing Browse views; server read-only, loopback-bound; no new data class exposed |
| T-05-03 | medium | accept | FileGraph() takes zero parameters, confirmed by reading traverse.go:172, so response size is bounded by the repository own fixed size, not by any caller-controlled fan-out |
| T-05-12 | medium | accept | TestFileGraphOpensEngineExactlyOnce PASS (ran this session); ENG-03 forbids caching as the alternative mitigation, and CONTEXT.md records a too-slow scan as an escalation, not a caching fix |

---

## Accepted Risks Log

No accepted risk sits at or above the high blocking threshold.

| Risk ID | Threat Ref | Severity | Rationale | Accepted By | Date |
|---------|------------|----------|-----------|-------------|------|
| R-05-01 | T-05-03 | medium | FileGraph() has no caller-controlled parameter, response size is bounded by the repository own fixed scale, and the transport ceiling is separately measured (T-05-08) rather than assumed. | plan 05-01 | 2026-08-30 |
| R-05-02 | T-05-12 | medium | Two full store scans per request is required by the correction to D-05 (node IDs are opaque hashes; resolving to a containing file needs a node scan before the edge scan). ENG-03 forbids caching as the fix; the threat model is a single local user with no rate limiting in scope. | plan 05-02 / D-09 | 2026-08-30 |
| R-05-03 | T-05-05, T-05-10 | low | Repo-relative paths crossing to the browser, the identical disposition every existing read rpc already carries (SRV-02/03 loopback plus exact-match Origin/Host, unchanged since Phase 1/4). | plans 05-01, 05-02 | 2026-08-30 |
| R-05-04 | T-05-15 | low | A 4-number measurement global on a loopback-bound, single-local-user, no-third-party-script page; also dispatched as an event so tests avoid depending on the global directly. | plan 05-03 | 2026-08-30 |
| R-05-05 | T-05-31 | low | Shared 15-field Node message reused for FileSymbols rather than a narrower shape, raised explicitly as a sub-decision and confirmed at a blocking-human checkpoint, not silently absorbed. | plan 05-06, maintainer at Task 1 checkpoint | 2026-08-31 |
| R-05-06 | T-05-50 | low | Directory labels and the geometry seam expose only data already rendered elsewhere in the app (Browse); the server remains read-only and loopback-bound. | plan 05-08 | 2026-08-31 |

---

## Notes

Note 1 - GRF-01 own gate did what it was designed to do. The binding measurement against
google/guava FAILED (seamReady exceeded its 60000ms deadline, recorded in
corpora/graph-render-observations.json), and the phase correctly halted per the committed
remedy path rather than widening the pre-locked threshold. 05-08 implemented the maintainer
chosen remedy (collapse-by-default with progressive expansion) and re-measured against the SAME
locked bars, producing a genuine PASS (corpora/graph-render-observations-collapsed.json). This
is the exact sequence T-05-01/T-05-18/T-05-40 tamper-resistance mitigations exist to make
possible: a real FAIL was recorded, not hidden, and the eventual PASS is against unmodified bars.

Note 2 - code review found two Critical and four Warning defects, none mapping to a security
threat. 05-REVIEW.md (CR-01: duplicate-request race with a rapid re-expand; CR-02: silent
symbol-state desync on directory toggle; WR-01: stale geometry after focus(); IN-01: unguarded
int64-to-int32 narrowing, judged unreachable at any realistic corpus scale) and 05-REVIEW-2.md
(WR-01a: review-finding-id leak into shipped doc comments, a recurrence of a defect class already
cleaned up three times in this codebase; WR-01b: same-directory collapse/re-expand symbol
state-desync gap in CR-02 original fix; WR-01c: a regression test that proved a republish call
fired but not that the coordinates changed) are all correctness/robustness findings, not attack
surface: none disclose data across a trust boundary, bypass a confinement check, or weaken an
access control. All six are fixed and independently verified (417/417 vitest, pnpm check
0/0, task web:drift PASS, task test:unit PASS, reconfirmed live this session).

Note 3 - WINDOWS.md entries 26, 28, 29 are known-open, non-security findings, recorded here
rather than re-raised. Entry 26 (a non-fatal cytoscape-elk adapter console error, layout
completes correctly) and entry 28 (a pre-existing guava-scale cytoscape layout console warning,
functional) are rendering-robustness observations with no security implication. Entry 29
(web:drift output half enumerates the filesystem via find while its source half uses git
ls-files, so it cannot by itself detect an incompletely-staged bundle) is a real gate blind spot
that touches the T-05-27/T-05-38 evidence chain, found and documented by this phase own fix
pass, with a staging gap it exposed already caught and fixed once and reconfirmed clean at every
subsequent bundle rebuild (git status check on web/build empty, checked live this session). The
suggested structural fix (git ls-files on web/build for both halves) is recorded as
out-of-phase-scope by the maintainer, not silently dropped.

Note 4 - severity-count discrepancy with the orchestrator inventory, resolved by independent
recount. The orchestrator registry inventory quoted 28 high / 16 medium / 7 low; this audit own
extraction of the severity column from all eight threat_model tables (script-based, run twice
with identical results) produced 27 high / 17 medium / 7 low. The totals (51) and the
disposition split (44/7) agree in both counts, and the HIGH-intersect-ACCEPT-is-empty property
holds under either count. The 27/17/7 figure, being independently re-derived rather than taken
on trust, is what this audit threats_open computation uses.

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-08-31 | 51 | 51 | 0 | gsd-secure-phase (retroactive verification, ASVS L1; every mitigate threat verified against working-tree code/tests at HEAD c117a898, diff base bc5dee97) |

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in the Accepted Risks Log
- [x] No accepted risk at or above the high blocking threshold
- [x] threats_open: 0 confirmed
- [x] Every zero-count verification paired with a positive control
- [x] status: verified set in frontmatter

Approval: verified 2026-08-31
