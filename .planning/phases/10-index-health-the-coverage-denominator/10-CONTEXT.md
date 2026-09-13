# Phase 10: Index Health — The Coverage Denominator - Context

**Gathered:** 2026-09-12
**Status:** Ready for planning
**Mode:** Smart discuss (autonomous) — four grey areas proposed in batch tables, every recommended answer accepted by the user with the question set visible

<domain>
## Phase Boundary

The index answers "why is my file missing" itself: a real discovered-versus-indexed denominator with a **recorded** reason for every gap, distinguishing extraction failures (already persisted in `File.errors`) from pre-extraction exclusions (recorded at the discovery decision point and persisted additively within `SchemaVersion 1`).

1. **HLT-05** — the write-path change: exclusion reasons (vendor/dot-dir, unsupported extension, build tag, size limit) captured where `Discover` decides them, persisted in a new graphstore namespace in the same commit batch as `File` records, never reconstructed by a query-time walk.
2. **HLT-06** — the wire surface: `GetHealthResponse` extended additively with a coverage summary, plus a new paged `GetCoverage` rpc for the per-file reason list, with the read-only method-set guard updated by set-equality in both directions.
3. **HLT-04** — the health page shows the denominator and lists each unindexed file with its reason, extraction failures visibly distinct from exclusions, demonstrated against a fixture repo containing at least one file of each kind.

This is the milestone's only write-path change. Out of scope by construction (ROADMAP notes): "re-index this file" auto-remediation on the coverage view (same `SRV-03` violation class as an editor shell-out — remedy is text only); `codegraph status` / MCP exposure of coverage (deferred — `status` mirrors a frozen golden shape); any change to which directories `ShouldSkipDir` prunes (e.g. `node_modules` is walked today; that is pre-existing behaviour and a separate decision).

</domain>

<decisions>
## Implementation Decisions

### Discovered-count definition & what gets a record

- **D-01:** "Discovered" is pinned once: **every regular file the walker actually visits.** Pruned subtrees (`vendor/`, dot-prefixed directories via `ShouldSkipDir`) are **not** discovered — their contents are never enumerated. The same definition feeds both the denominator and the reason list, so the two can never disagree in the user's face (ROADMAP note). — **Reversibility:** costly — changing the definition later changes every reported number.
- **D-02:** A pruned directory is represented by **one directory-level exclusion record** (`vendor/` → `DIR_VENDOR`; `.github/`, `.codegraph/` … → `DIR_DOTPREFIX`), never by phantom per-file rows. The UI renders these as "directory excluded" rows.
- **D-03:** An unsupported-extension file (`.md`, `.json`, `.yaml`, `.sum` …) gets **one record per file** (path + reason code + the extension in `detail`). Criterion 1 says "lists each unindexed file"; the records are tiny and `.git` is pruned so counts stay in the hundreds–low thousands. The UI groups by reason (and extension) with counts and expands to rows. Aggregate-only counts were rejected because the per-file list would then have to be reconstructed at query time — exactly what HLT-05 forbids.
- **D-04:** Size limit becomes a **stat-based pre-extraction check at discovery**: `SizeBytes > parser.MaxSourceBytes` → reason `SIZE_LIMIT`, bytes never read. The parser's own `ErrSourceTooLarge` stays as the backstop; any file that still trips it remains an extraction failure persisted in `File.errors`, exactly as today (`resolve.go:298`).

### Persistence shape

- **D-05:** Exclusion records live in a **new graphstore key namespace** — one free single-byte prefix (the planner picks the letter; `m n e f a x` are taken) keyed `<prefix>/<path>` — holding a new `graph.proto` message `ExcludedFile { string path; ExclusionReason reason; string detail; int64 size_bytes; }`. `SchemaVersion` stays `1` (additive only), mirroring how `prefixFileIndex` was added in v0.12.0 Phase 4. `File` records are **not** overloaded with a not-indexed flag (`file_count` and every `f/` iterator keep their meaning). — **Reversibility:** one-way once a release ships the namespace (a stored shape).
- **D-06:** An old graph reports coverage via an additive **`Meta.has_coverage = 9`** (next free field after `commit_sha = 8`), following the `HasFileIndex` precedent: absent/false → coverage **unknown**, never `0/0`, never an error. A graph written before this phase opens and says "re-index to record coverage". Presence of any exclusion key is **not** used to infer "known" (a repo with genuinely zero exclusions would be indistinguishable from a pre-phase graph).
- **D-07:** Records are written **in the same commit batch as the `File` records** at the pipeline's commit point (indexer → graphstore), honouring Phase 7's archtest D-01: `query` reads them back through `graphstore` only and never imports the discovery helper. A full index range-deletes the namespace then rewrites; `Sync` upserts per path and prunes paths no longer present — the lifecycle `f/` records already have. Reasons are **not** written eagerly during the walk (a mid-run extraction failure would leave a half-written set).
- **D-08:** The reason vocabulary is a **closed proto enum** `ExclusionReason { UNSPECIFIED = 0; DIR_VENDOR; DIR_DOTPREFIX; UNSUPPORTED_EXTENSION; BUILD_TAG; SIZE_LIMIT; }` plus free-text `detail` (the extension, the byte size, the pruned directory name). Closed like `EditorLinkAvailability` so the UI can never meet an unknown reason and set-equality tests can pin the set.

### Wire & UI surface

- **D-09:** `GetHealthResponse` is extended **additively** with `Coverage coverage = 17` (corrected 2026-09-12 by research: field 16 is already `commit_sha`, see `ui.proto:756-822`) — `bool known`, `int64 discovered`, `int64 indexed`, `int64 excluded`, `int64 extraction_failed`, `map<string,int64> excluded_by_reason` — so the health page shows the denominator on its existing poll with no second call. — **Reversibility:** one-way once shipped on the wire (field number and message name).
- **D-10:** The per-file reason list is a **new rpc `GetCoverage(GetCoverageRequest{ int32 page_size; string page_token; ExclusionReason reason; }) → GetCoverageResponse{ repeated CoverageRow rows; string next_page_token; }`**, where a row carries path, kind (excluded vs extraction-failed), reason, detail. The name clears all 19 `mutatingVerbs` substrings (no "Index"; "List" is not in the set either). It is the **16th** rpc: `wantUIServiceMethods` 15→16 by set-equality in both directions with the count asserted from both sides, and `uiProtoFieldFixture` extended additively (the 09-01 pattern, same commit as the proto change). The full list is **not** inlined in `GetHealthResponse` (unbounded payload on a polled call — the v0.12.0 Phase 1 transport-cap lesson). — **Reversibility:** one-way once shipped (rpc name).
- **D-11:** The health page (`web/src/routes/health/+page.svelte`, `web/src/lib/health-view.ts`) gains a **"Coverage" section**: `N discovered · M indexed · K excluded · J extraction failures`; grouped by reason with counts, each group expanding to file rows (path + detail); `File.errors` rows rendered as "extraction failed: <error>" and visibly distinct from exclusions; an old graph renders "Coverage unknown — re-index to record it". Remedy is **text only** — no "re-index this file" action (SRV-03). No separate `/coverage` route.
- **D-12:** `codegraph status` and the MCP surface are **out of scope** this phase — recorded under Deferred Ideas.

### Proof & guards

- **D-13:** Criterion 1's fixture is a **committed fixture module under `internal/indexer/testdata/coverage/`** containing at least one file of each kind: `vendor/x.go` (DIR_VENDOR), `.hidden/y.go` (DIR_DOTPREFIX), `notes.md` (UNSUPPORTED_EXTENSION), `tagged.go` with `//go:build ignore` (BUILD_TAG), an oversize `.go` **generated at test time** above `parser.MaxSourceBytes` (SIZE_LIMIT — never committed), and one file that **fails extraction** so it lands in `File.errors`. The test indexes it and asserts the exact counts **and** the per-file reasons — counts reported, not implied.
- **D-14:** Criterion 2 ("fails if the reason is reconstructed at query time") gets **two guards**: (a) behavioural — index the fixture, then mutate the disk **without re-indexing** (strip the build tag from `tagged.go`, delete `notes.md`) and assert the read-back still reports `BUILD_TAG` and `UNSUPPORTED_EXTENSION` for those paths (a query-time re-walk would say "no longer excluded" / "not there"); (b) structural — `internal/query/archtest` still forbids the indexer root, plus a positive-controlled source scan **scoped to the new coverage-reading file(s) in `internal/query`** asserting zero `filepath.WalkDir` / `os.ReadDir` calls (corrected 2026-09-12 by research: `status.go` already has two production `WalkDir` calls for mtime/db-size, so a package-wide zero guard would be false on day one; follow `TestEditorDiscoverySourceNeverSpawnsAProcess`'s file-scoped shape with a positive control).
- **D-15:** Criterion 3 gets a dedicated **old-graph test**: open a store written without coverage records (`Meta.has_coverage` unset) and assert `GetHealth` reports `coverage.known == false` with no counts — never `0/0`, never an error — following `internal/query/meta_test.go`'s `HasFileIndex` precedent. Mutation family: flip the reader to treat unset as zero and watch it RED.
- **D-16:** Criterion 4: `readonly_test.go` asserts `wantUIServiceMethods` set-equality in **both** directions with the count from both sides (proto descriptor vs fixture), and a table test proves `GetCoverage` clears every `mutatingVerbs` substring while a decoy `GetIndexCoverage` is **rejected** — the positive control that the guard discriminates. Mutation family: rename to the decoy and watch RED.
- **D-17:** `10-MUTATION-LOG.md` follows the 07/08/09 house format (pre-mutation cleanliness gate, captured RED output, byte-clean revert); `10-SECURITY.md` is required (security enforcement on, ASVS L1) and covers the write path's new trust surface (path strings from the walker persisted and rendered — same escaping discipline as `File.path` today) with every threat row test-or-verdict.

### Claude's Discretion

- The exact key-prefix byte for the new namespace, and whether directory-level records share the file namespace or get a sub-prefix.
- `GetCoverage` default/maximum page size and the page-token encoding (follow whatever the existing paged rpcs use, if any).
- Whether `excluded_by_reason` keys are the enum names or numbers on the wire (pick what the generated TS client makes easiest to display).
- Health-page grouping order and expansion UX details.
- Whether the discovery helper is a new file in `internal/indexer` or lives inside `discover.go` — either satisfies the archtest.

</decisions>

<canonical_refs>
## Canonical References

### Phase scope and requirements
- `.planning/ROADMAP.md` → `### Phase 10: Index Health — The Coverage Denominator` — goal, four success criteria, the notes on reasons-as-lies, the pinned discovered definition, and the `mutatingVerbs` "Index" trap.
- `.planning/REQUIREMENTS.md` — HLT-04, HLT-05, HLT-06.
- `.planning/phases/07-guards-that-cannot-fire/07-CONTEXT.md` → "Archtest boundary (GRD-02)" D-01..D-04 and the recorded "Consequence for Phase 10".

### Discovery and persistence
- `internal/indexer/discover.go` — `Discover`, `ShouldSkipDir`, `DiscoveredFile`, the Go-only `build.Default.MatchFile` build-tag check, `lookupLanguageByExt`.
- `internal/indexer/pipeline.go` — `Stats.Skipped`, the commit point.
- `internal/indexer/resolve.go:285-305` — how a failed extraction becomes a `File` record with `Errors`.
- `internal/indexer/extract.go:52-58` — `ErrSourceTooLarge` recorded on `FileResult.Err`.
- `internal/parser` — `MaxSourceBytes`, `ErrSourceTooLarge`.
- `internal/graphstore/keys.go` — the key-namespace table (`m n e f a x`), prefix range-delete discipline.
- `internal/schema/graph.proto` — `File` (fields 1–8), `Meta` (fields 1–8, `has_file_index = 7`), `SchemaVersion = 1` in `internal/schema/meta.go`.
- `internal/query/meta_test.go` — `HasFileIndex` round-trip precedent.
- `internal/query/archtest/import_direction_test.go` — GRD-02 archtest.

### Wire surface and UI
- `internal/uiproto/uiv1/ui.proto` — `GetHealthResponse` (fields 1–15), `IndexHealth`.
- `internal/uiserver/handlers.go:968` — `GetHealth` handler.
- `internal/uiserver/readonly_test.go:71,134` — `wantUIServiceMethods` (15 today), `mutatingVerbs` (19).
- `internal/uiserver/editorlink.go`, `.planning/phases/09-source-view-follow-through-breadcrumb-editor-handoff/09-01-SUMMARY.md` — the additive-rpc + same-commit-fixture-update pattern from Phase 9.
- `web/src/routes/health/+page.svelte`, `web/src/lib/health-view.ts`, `web/src/lib/status.ts` — the health page and its view helpers.

### Guard and proof precedents
- `.planning/phases/09-source-view-follow-through-breadcrumb-editor-handoff/09-MUTATION-LOG.md`, `09-SECURITY.md` — house formats.
- Repo rule `84d1gfpywd` — every guard carries a positive assertion that it did its work.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `ShouldSkipDir` (`discover.go:53`) is the single directory-exclusion predicate shared with the fsnotify watcher — the directory-level reason (D-02) is decided exactly there.
- `DiscoveredFile` already carries `SizeBytes` from the discovery stat, so the D-04 size pre-check costs no extra syscall.
- `File.errors` + `Stats.Skipped` already distinguish extraction failures; the coverage summary's `extraction_failed` count can be derived from `File` records with non-empty `errors`.
- `Meta.has_file_index` is the additive-compat precedent for `has_coverage` (D-06); `prefixFileIndex` is the additive-namespace precedent (D-05).
- Phase 9's `GetEditorLink` landing (`09-01-SUMMARY.md`) is the exact recipe for adding an rpc: proto edit + `task proto:gen` + `wantUIServiceMethods` + field fixture in one commit.

### Established Patterns
- Read-only-by-construction (`SRV-03`): the UI never mutates; remedies are text.
- Closed enums on the wire for config/availability state (`EditorLinkAvailability`, `EditorTemplateSource`).
- Additive-only schema evolution within `SchemaVersion 1`; unknown states reported as unknown, never as zero.
- Guards carry positive controls; mutation logs demonstrate RED before a gate is called fixed.

### Integration Points
- Write path: `Discover` (decision points) → pipeline commit batch → graphstore new namespace (D-05/D-07).
- Read path: `graphstore` → `internal/query` (Engine surface for coverage summary + paged rows) → `internal/uiserver` (`GetHealth` extension + `GetCoverage`) → generated TS client → `health-view.ts` → `/health` page.
- Guards: `internal/query/archtest` (no indexer root import), `readonly_test.go` (method set + verb guard), the new fixture test under `internal/indexer/testdata/coverage/`.

</code_context>

<specifics>
## Specific Ideas

- The "discovered" definition and the reason list must come from the **same walk result** — pin it in one helper so the denominator and the rows are computed from one source (ROADMAP: "pinned once").
- Name the rpc `GetCoverage`, not anything containing "Index" — the verb guard is a substring check and `GetIndexCoverage` fails it outright.
- The old-graph "unknown" state must be a first-class UI state with its own copy, not an empty table.

</specifics>

<deferred>
## Deferred Ideas

- `codegraph status --json` additive `coverage` field and any MCP exposure of coverage (D-12) — a later phase; re-freezing the status golden does not belong in a UI phase.
- Whether `node_modules` (not dot-prefixed, not `vendor`) should be pruned by `ShouldSkipDir` — pre-existing behaviour surfaced by this phase's denominator; a separate decision with watcher implications.
- "Re-index this file" remediation action on the coverage view — out of scope by construction (SRV-03).

</deferred>
