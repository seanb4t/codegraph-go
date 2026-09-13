---
phase: "11"
slug: "graph-view-community-clustering"
status: verified
# threats_open = count of OPEN threats at or above workflow.security_block_on severity (the blocking gate)
threats_open: 0
asvs_level: 1
created: "2026-09-13"
---

# Phase 11 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

**Register origin:** `register_authored_at_plan_time: true` — plans 11-01 through 11-04 each
carry a `<threat_model>` block. The register below is the union of all four plans' rows,
deduplicated by id; where two plans reused the same id, the mitigation cell merges both plans'
text (noted inline by `(11-0N)` tags). Verification depth is ASVS L1 (grep/execution-level
mitigation presence), which the workflow's short-circuit rule declares sufficient for
`threats_open: 0` at L1.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|----------------|
| store snapshot → `FileGraph()` → `AssignCommunities` | inputs are the store's OWN already-validated file paths and edge counts, computed fresh on every call, never cached (D-16) | file paths, aggregated file-pair edge counts — no external or user-controlled input reaches the algorithm |
| `gonum.org/v1/gonum` import closure → `internal/query` | the milestone's only new direct dependency, executing inside every `FileGraph()` call — this phase's new trust surface | a Go module's compiled code, proven cgo-free by a positive-controlled closure scan |
| `corpora/graph-cluster-threshold.json` → `tools/graphcluster` | the tamper target — a committed bar that must outlive the measurement it judges | threshold values read only; a digest of the file is recorded in the observation, never the reverse |
| `internal/uiserver` → browser | two additive read-only int32 fields (`community_id`, `community_count`) cross to the loopback SPA on the existing `FileGraph` rpc; nothing crosses back | server-computed integers only |
| `web/src` → `check-no-force-layout.mjs` | source text scanned for a rejected layout family — the scan's own blindness is the risk this boundary manages | plain text of tracked `.ts`/`.js`/`.svelte`/`.mjs` files and `package.json` dependency names |

**The phase's new trust surface is a numerical dependency executing inside a read-only query over the store's own data, and a committed bar that must outlive the measurement it judges.**

---

## Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation | Status |
|-----------|----------|-----------|----------|-------------|------------|--------|
| T-11-01 | Tampering (supply chain) | `gonum.org/v1/gonum v0.17.0` direct require | high | mitigate | Pinned to the version already in `go.sum` (no new module, no bump — one added `go.mod` line, `go.sum` unchanged, asserted) (11-01). `check:gonum` (11-03) proved govulncheck SOURCE mode clean over the main module with gonum in the scanned set (26 packages), gonum present by name and pinned version in a release-shaped SBOM (148 packages total, `gonum.org/v1/gonum` present once at v0.17.0 beside a `github.com/cockroachdb/pebble/v2` positive control present once), and a 91-package cgo-free import closure over `gonum.org/v1/gonum/graph/community` beside a `github.com/tree-sitter/go-tree-sitter` control the same scan sees as cgo-bearing (3 packages). 11-05 family (d) watched the cgo control-neutered target fail live (`positive control found no cgo`), reverted byte-clean. Legitimacy assumption recorded verbatim below (`[ASSUMED]`) — see Notes. | closed |
| T-11-02 | Tampering | `corpora/graph-cluster-threshold.json` | high | mitigate | Committed ALONE as the phase's first commit (11-01 Task 1 asserted one file, one commit; still exactly one commit as of phase close). 11-02's harness never opens it for writing (`TestHarnessSourceNeverIndexesOrWritesTheStore`); the observation records the threshold's sha256 digest and `TestClusterThresholdDigestMatchesCommittedObservation` re-checks it; `TestClusterThresholdCommitIsAncestorOfEveryMeasurement` proves the threshold commit is an ancestor of every measurement commit via `git merge-base --is-ancestor`. 11-05 family (b) watched the digest test go RED against a working-tree-only widening (500 to 5000) and reverted byte-clean — the threshold has exactly one commit at phase close. | closed |
| T-11-03 | Repudiation (a colouring that silently changes between renders misleads a reviewer about ownership boundaries) | `AssignCommunities` | medium | mitigate | Three determinism mechanisms in code (sorted node ids, fixed PCG seed, canonical relabel) (11-01). `TestAssignCommunitiesDeterministic` (5 runs each, count logged) plus the seed-perturbation RED control `TestAssignCommunitiesSeedPerturbationFlipsTheTieFixture` (11-01 Task 3) plus 11-05 family (a) watched a continuously-varying per-call seed counter flip the `tie` sub-test RED while `structured` correctly stayed invariant, reverted byte-clean. | closed |
| T-11-04 | Denial of Service | pathological file-pair graph runtime inside `FileGraph()` | low | accept | Local, read-only developer tool over the user's own index; GRF-09 (11-02) measured the largest known corpus (guava, 3,233 file nodes / 21,554 file-pair edges) against a committed bar — median 106 ms vs 500 ms max, PASS, 4.7x margin — the bar is the control, no additional hardening warranted (11-01, 11-02). | closed (accepted) |
| T-11-05 | Information Disclosure | `communitySeed1`/`communitySeed2` misread as a secret | low | accept | These are determinism constants, not secrets — committed in the clear by design (11-RESEARCH V6); the doc comment on `community.go` says so explicitly; nothing cryptographic depends on them; changing them only changes which of several equally-valid modularity-tying partitions wins a tie, never the correctness of the result (11-01). | closed (accepted) |
| T-11-06 | Spoofing | DNS rebinding against the FileGraph rpc, now carrying two more fields and read by the UI colouring/toolbar | low | accept | Inherited — no new rpc or mux entry; `originHostGuard` unchanged (03-SECURITY.md T-03-13 disposition, 09-SECURITY.md T-09-08 precedent) (11-01, 11-04). | closed (accepted) |
| T-11-07 | Repudiation | a hand-typed or averaged GRF-09 verdict | medium | mitigate | Verdict word written only by `judge` from `medianInt64` (sorted middle, int64, never a float) (11-02). `TestRunRecordsFailVerbatimAndRunsInOrder` proves a FAIL is preserved verbatim with runs in order; the real measurement's verdict (PASS, median 106 ms, runs [108, 95, 106]) is committed verbatim in `corpora/graph-cluster-observations.json`. | closed |
| T-11-08 | Tampering | harness re-indexing or writing the shared corpus store | medium | mitigate | `graphstore.Open` + `Snapshot` only, never `indexer.Run`/`NewWriter` (11-02). `TestHarnessSourceNeverIndexesOrWritesTheStore` is a positive-controlled source scan proving the harness never re-indexes and never writes the store; the real measurement run opened the pinned guava store read-only and produced exactly one observation write. | closed |
| T-11-10 | Elevation of Privilege | cgo entering the query engine's hot path through a transitive gonum package (e.g. `blas/cgo`) | high | mitigate | Closure scan asserts `0 with cgo` and `blas/cgo references 0` over `>= 50` packages (91 inspected) (11-03). The `github.com/tree-sitter/go-tree-sitter` control independently proves the same scan mechanism correctly detects cgo (3 packages) when it is genuinely present. 11-05 family (d) watched the control-neutered target fail on exactly this claim (`positive control found no cgo`), reverted byte-clean. | closed |
| T-11-11 | Repudiation | a vacuous green from an empty closure, an empty SBOM, or a scan that never included gonum | medium | mitigate | Every half of `check:gonum` prints its inspected count BEFORE asserting anything, with floors: `ngonum >= 1` (26), SBOM `npkg > 0` (148), closure `n >= 50` (91), control `cnon >= 1` (3) (11-03). The same discipline applies to `check-no-force-layout.mjs` (`elkLayoutRefs >= 1`, `elkImportRefs >= 1` before any forbidden-match assertion) and to GRF-09's harness (node/edge/community counts always logged, never a silent zero) (11-01, 11-02, 11-04). | closed |
| T-11-12 | Tampering (layout decision regression) | a force-directed layout re-entering via a layout name or a `cytoscape-*` extension | medium | mitigate | `check-no-force-layout.mjs` proves no cytoscape layout is invoked with a forbidden name written as a string literal at the `name:` option position, or as a `cytoscape-<x>` import/dependency specifier — a string-built or variable layout name is NOT detected by this scan (WR-03, narrowed from the original "reachable from any code path" claim); the walker also follows symlinked directories rather than only `Dirent.isDirectory()` (WR-02), and a non-fatal `unresolvedLayoutNames` finding reports call sites (e.g. `GraphCanvas.svelte`'s own `{ ...LAYOUT_OPTIONS, fit }` spread) this scan could not statically resolve to a literal. Scoped, layout-name-position regexes over `web/src` + `package.json`, with an `elk` positive control and a `--self-test` injection (11-04 Task 3). `GraphCanvas.svelte` confirmed byte-identical to the GRF-09 threshold commit throughout the phase (11-04 Task 1 verify). 11-05 family (c) planted a real `name: 'cose'` literal in a tracked `web/src` file and watched the scan fail, naming the exact file and line, then reverted byte-clean. | closed |
| T-11-13 | Tampering (selector injection) | a wire integer reaching a cytoscape selector string | low | mitigate | Only `(id - 1) % 12` — a bounded integer 0..11 — is interpolated into a class name; the 12 selectors are static literals generated from the palette at module load, never from response data (11-04). | closed |
| T-11-14 | Repudiation (misleading grouping) | a directory compound coloured by one member's community | low | mitigate | D-11: `expandedDirElement`/`collapsedDirElement` gain no `communityId` field or class (`web/tests/file-graph-transform.test.ts`); no `[?isDirectory]` colour rule exists in `graph-style.ts` (11-04). | closed |
| T-11-SC | Tampering | npm/pip/cargo installs | low | accept | None across all four plans — the only dependency change is a Go module promotion of a package already resolved in `go.sum`; no npm/pypi/crates seam applies to a Go module (11-01 through 11-04, 11-RESEARCH Package Legitimacy Audit). See T-11-01's `[ASSUMED]` legitimacy note below. | closed (accepted) |

*T-11-09 (11-02's threat model: `Node.community_id` written by a second writer or left stale after Sync, medium, mitigate — the D-07 index-time-persistence fallback's own risk) is NOT included above: 11-02's Task 3 conditional fallback did not execute (the GRF-09 verdict was PASS, median 106 ms vs 500 ms max), so the fallback code path this threat concerns was never built. Not applicable — verdict PASS.*

*Status: open · closed · open — below high threshold (non-blocking)*
*Severity: critical > high > medium > low — only open threats at or above `high` count toward threats_open*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|--------------|------|
| R-11-01 | T-11-04 | Local, read-only developer tool over the user's own index; GRF-09 measured the largest known corpus against a committed bar (median 106 ms vs 500 ms max) rather than adding runtime hardening | plans 11-01, 11-02 | 2026-09-13 |
| R-11-02 | T-11-05 | `communitySeed1`/`communitySeed2` are determinism constants, not secrets, committed in the clear by design; nothing cryptographic depends on them | plan 11-01 | 2026-09-13 |
| R-11-03 | T-11-06 | `originHostGuard` already wraps the whole mux; the FileGraph rpc's two new fields add no new mux entry, so no new DNS-rebinding surface (inherited from 03-SECURITY.md T-03-13 / 09-SECURITY.md T-09-08 / 10-SECURITY.md R-10-01) | plans 11-01, 11-04 | 2026-09-13 |
| R-11-04 | T-11-SC | No package-manager install runs in this phase; the only dependency change is a Go module already resolved in `go.sum` promoted from transitive to direct | plans 11-01 through 11-04 | 2026-09-13 |

---

## Notes

**1. GRF-10 `[ASSUMED]` package legitimacy — recorded verbatim, an explicit unresolved edge, not a resolved one.**
`gonum.org/v1/gonum`'s legitimacy verdict is `[ASSUMED]` (11-RESEARCH.md § Package Legitimacy
Audit), NOT machine-verified: "Not re-run via `gsd-tools query package-legitimacy check` this
session — `[ASSUMED]` pending that seam call at plan time." The evidence backing the assumption
is real but qualitative — a long-established, org-maintained numerical/graph library (the gonum
project, active since ~2013), already resolved transitively in `go.sum` via sigstore before this
phase, source-read directly from `$GOMODCACHE` during research. No npm/pypi/crates
package-legitimacy seam covers a Go module, so `gsd_run query package-legitimacy check` has no
applicable ecosystem flag for this dependency, and no `checkpoint:human-verify` was inserted at
any point in this autonomous run — there was no seam to trigger one. This assumption is carried
forward unresolved: a future session with Go-module legitimacy tooling should re-verify it
rather than treat this phase's `[ASSUMED]` tag as equivalent to a machine-checked verdict.

**2. `communitySeed1`/`communitySeed2` are determinism constants, not secrets.** Recorded
explicitly per V6 (ASVS Cryptography) in 11-RESEARCH.md: "must NOT be treated as, or documented
as, a secret; it is committed in the clear precisely because reproducibility (not
unpredictability) is the goal." The literal bytes spell "codegrap"/"louvain1" in ASCII for
readability. See T-11-05 above.

**3. GRF-09 verdict and branch: PASS, D-07 fallback NOT executed.** The measured verdict
(11-02) was **PASS** — median 106 ms over three cold runs [108, 95, 106] ms against a 500 ms
max, a 4.7x margin, at the pinned guava corpus scale (3,233 file nodes / 21,554 file-pair
edges, 355 communities). 11-02's Task 3 — the conditional D-07 index-time-persistence
fallback (`Node.community_id = 50`) — was correctly evaluated and NOT executed. `Node`'s
reserved `50-59` range stays fully reserved (unclaimed); `FileGraph()`'s fresh-per-call
community compute (D-15: no cache, computed on every call) is the sole, unmodified path to
production. T-11-09 (the fallback's own second-writer/staleness risk) is therefore not
applicable this phase — see the Threat Register footnote above.

**4. `check:gonum` and `check:no-force-layout` are NOT wired into CI this phase — a visible
follow-up, not a silent gap.** Both new Taskfile targets (11-03's `check:gonum`, 11-04's
`check:no-force-layout`) are local, developer-run gates, deliberately NOT added to
`test:`/`lint:` wrappers or `.github/workflows/ci.yml` in this phase (11-03-SUMMARY.md "Next
Phase Readiness": "CI wiring is 11-05's decision, recorded in `11-SECURITY.md`'s notes"). CI's
existing blocking supply-chain gate — the `govulncheck (DIST-03, blocking)` job in `ci.yml` —
remains the CI-enforced control for the dependency surface this phase adds; `check:gonum`'s
govulncheck half runs in the SAME source-mode, main-module scope as that CI job (not the
advisory binary-mode `vuln:` target), so the two are consistent in what they scan, but only
`ci.yml`'s job runs automatically on every push. Wiring `check:gonum` and
`check:no-force-layout` into CI is recorded here as an explicit, visible follow-up rather than
silently deferred: until it happens, a regression in gonum's cgo-freedom, SBOM presence, or the
force-layout scan is caught only by a developer manually running `task check:gonum` /
`task check:no-force-layout`, not automatically on every PR.

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-09-13 | 15 | 15 | 0 | plan 11-05 (ASVS L1 inline; auditor short-circuited per `threats_open:0` + `register_authored_at_plan_time:true` + `asvs_level:1`) |

**Audit note — what was checked by execution vs. by reading.** All three `high`-severity rows
(T-11-01, T-11-02, T-11-10) were checked by **execution** during this plan, re-run live at HEAD
at phase close: `GOTOOLCHAIN=go1.26.6 task -s check:gonum` (`check:gonum: PASS`, 26 gonum
packages scanned, SBOM presence 1/1 at v0.17.0, cgo closure 91 packages / 0 cgo, control 3 with
cgo), `GOTOOLCHAIN=go1.26.6 go test -count=1 -v -run 'TestClusterThreshold' ./tools/graphcluster/...`
(both ancestry and digest tests PASS), and `internal/query/community_test.go`'s determinism
suite (`TestAssignCommunitiesDeterministic`, both sub-tests PASS). `11-MUTATION-LOG.md` proves
all four of these guards RED against a confirmed-applied mutation and byte-clean reverted — the
strongest evidence class this project uses (a gate demonstrated failing on the exact claim it
exists to protect, not merely observed passing). Every other `Test…`-cited name in this register
was independently resolved to a real `func Test…(` definition via `rg` against `internal/` and
`tools/` at HEAD — not trusted from any plan's prose. Every `.test.ts`-cited vitest file was
confirmed present under `web/tests/`. The `low`-severity rows disposed `accept` (T-11-04,
T-11-05, T-11-06, T-11-SC) were checked by **reading** the cited source/design property and the
inherited prior-phase disposition, consistent with 08-SECURITY.md's, 09-SECURITY.md's, and
10-SECURITY.md's own audit-note precedent for accepted risks that inherit an existing,
already-verified guard.

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-09-13

**Outstanding, not security-blocking:**
- `check:gonum` and `check:no-force-layout` are not yet wired into `.github/workflows/ci.yml`
  (Notes item 4 above) — CI's existing blocking `govulncheck` job remains the automated
  supply-chain gate; both new phase-specific gates are currently developer-run only.
- The GRF-10 package-legitimacy verdict for `gonum.org/v1/gonum` remains `[ASSUMED]`, not
  machine-verified (Notes item 1) — no Go-module seam exists in the package-legitimacy tooling
  today.
