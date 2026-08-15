# Phase 5: Process, CI & In-Tree Sweep - Context

**Gathered:** 2026-08-15
**Status:** Ready for planning

<domain>
## Phase Boundary

Sweep the contributor-facing and in-tree surfaces so a contributor filing an issue or opening a PR, and an agent reading the source, meet a project described on its own terms — with TypeScript-the-indexed-language intact.

**In scope:** PROC-01 (5 issue templates carry no comparison framing), PROC-02 (PR template + 3 variants), PROC-03 (workflows pass actionlint + own checks, no retired framing in job/step names/comments), CODE-01 (comments across `internal/`, `tools/`, `test/` carry no comparison framing, term-by-term with recorded reasons, never regex), CODE-03 AMENDED (**drop `codegraph migrate` entirely** — maintainer ruling 2026-08-15).

**Depends on:** Phase 3 (done), so no in-tree comment change shares a diff with a golden change.

**Does NOT touch:** the golden harness, the corpus lock, the benchmark path (`bench.yml`'s own de-coupling is Phase 6; only its framing is this phase's), `docs/BENCHMARKS.md` (Phase 6), `tsextract`, the language registry, the capability matrix (product surface).

**AMENDED SCOPE (maintainer ruling 2026-08-15):** `codegraph migrate` is **dropped entirely**, not reframed. The migration path was parity framing (a drop-in-competitor posture), inconsistent with the milestone's standalone-identity goal. This reverses the milestone's standing "sweep removes framing, never capability" rule for this one capability — a deliberate, recorded amendment (REQUIREMENTS.md CODE-03, ROADMAP.md).
</domain>

<decisions>
## Implementation Decisions

### CODE-01 — the in-tree sweep line

- **D-01:** Comparison framing = **references to the prior implementation as a comparison baseline** ("matches TS", "parity with the original", "drop-in", "vs the TS version", "behavioral parity"). These are removed. — **Reversibility:** reversible — comment edits.
- **D-02:** Product truth **stays**: `tsextract` and other package names, the language registry, the capability matrix, `TypeScript` as an indexed language, and **past-tense history** ("began as a rewrite of", "was originally based on"). Every retained use is resolved **term-by-term with a recorded reason, never by regex** — the naive-regex failure mode is named in the requirements (a find-and-replace over "TypeScript" breaks `tsextract` and de-lists a supported language). — **Reversibility:** reversible.
- **D-03:** The `syntheticParitySrc` residual (`testdata/golden/behavioral_test.go:239`) is a **Go code identifier** carrying the retired "parity" name — Phase 4 flagged it to Phase 5. It is CODE-01's scope (rename to a behavioral name). The matrix `golden-parity` code comments (`matrix.go:18,32`, `matrix_test.go:217,221`) are also CODE-01's scope. — **Reversibility:** reversible — identifier rename.

### CODE-03 AMENDED — drop migrate entirely

- **D-04:** `codegraph migrate` is **removed entirely** (maintainer ruling 2026-08-15): the command registration in `internal/cli/root.go`, the whole `internal/migrate/` package (migrate.go, reader.go, translate.go, swap.go, validate.go, progress.go, batchwriter.go + tests + `migratetest/` fixture + `archtest/` confinement test), and the `modernc.org/sqlite` sole-use dependency (with its `modernc.org/libc` / `modernc.org/mathutil` indirect deps). Nothing left referencing it. — **Reversibility:** one-way — recreates no capability; a TS user re-indexes from source. Restorable only from git history (the archived build phase `milestones/v0.1-phases/07-migration-tool` stays).
- **D-05:** The project's framing is on its own terms — a code knowledge graph with a pre-indexed query surface, not a drop-in alternative to any prior implementation. The "uninstall TS CodeGraph... migrate indexes... works the same or better" core-value text is itself competitive framing (the parity story), not a binding constraint. Where it appears in the swept surfaces it is removed as framing, not product truth.

### PROC-01/02/03 — contributor-facing + workflows

- **D-06:** The `.github` issue templates (5), PR template + 3 variants, and the workflow files (`ci`, `release`, `bench`, `post-release-verify`, `linux-cross-canary`, `require-issue-link`, `auto-close-unsolicited-prs`, and the Phase-3 `corpora.yml`) carry no comparison framing in job names, step names, and comments. Phase 4 already re-pointed the FLAG-PARITY links to `.planning/` — this phase sweeps the remaining framing prose ("ports observable behavior from another implementation", "parity decisions") in those same templates. `bench.yml`'s own de-coupling is Phase 6's; only its framing is swept here. — **Reversibility:** reversible — text edits.

### Claude's Discretion

- The exact rewording of each contributor-facing template, provided it is on codegraph-go's own terms and carries no comparison framing.
- The exact new name for `syntheticParitySrc` (a behavioral name consistent with the Phase 2 rename).
- The order of the internal//tools//test/ comment sweep (which packages first).

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### The amended scope
- `.planning/REQUIREMENTS.md` — CODE-03 amended (migrate dropped)
- `.planning/ROADMAP.md` — Phase 5 amended (migrate removed); the standing decision at the milestone index also amended
- Engram records `4j8kjck5gd` (drop migrate) and `myywc0y9vm` (the core-value text is competitive framing)

### The surfaces to sweep
- `.github/ISSUE_TEMPLATE/*.yml` (5: bug_report, chore, config, enhancement, feature_request) — PROC-01
- `.github/pull_request_template.md` + `.github/PULL_REQUEST_TEMPLATE/*.md` (3 variants) — PROC-02
- `.github/workflows/*.yml` (ci, release, bench, post-release-verify, linux-cross-canary, require-issue-link, auto-close-unsolicited-prs, corpora) — PROC-03
- `internal/`, `tools/`, `test/` code comments — CODE-01
- `testdata/golden/behavioral_test.go:239` (`syntheticParitySrc`) — CODE-01 identifier residual
- `internal/indexer/capability/matrix.go:18,32`, `matrix_test.go:217,221` (`golden-parity` code comments) — CODE-01

### What to remove
- `internal/migrate/` (whole package) + `modernc.org/sqlite` go.mod dependency — CODE-03 amended
- `internal/cli/root.go` migrate command registration — CODE-03 amended

### What stays
- `internal/indexer/tsextract/`, the language registry, `docs/LANGUAGE-CAPABILITY-MATRIX.md` (product surface)
- `bench.yml`'s own de-coupling (Phase 6), `docs/BENCHMARKS.md` (Phase 6)
</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- Phase 4's reference sweep established the `.planning/` re-point pattern for deleted-file links.
- The Phase 2 rename (`parity` → `behavioral`) established the vocabulary the in-tree sweep follows.

### Established Patterns
- **Term-by-term with recorded reasons, never regex** — the naive-regex failure mode is named in the requirements.
- **Remove framing, never capability** — with the ONE recorded exception (migrate, D-04).
- **Past-tense history stays** — "began as a rewrite of" is product truth, not framing.

### Integration Points
- The migrate removal touches `internal/cli/root.go` (command registration) and `go.mod` (the modernc.org/sqlite dependency).
- The in-tree sweep touches every package in `internal/`, `tools/`, `test/` — the largest text surface in the milestone.
</code_context>

<specifics>
## Specific Ideas

- The comparison-baseline vocabulary to remove: "matches TS", "parity with the original", "drop-in", "vs the TS version", "behavioral parity", "ports observable behavior from another implementation", "parity decisions".
- Product truth to keep: `tsextract`, the language registry, the capability matrix, TypeScript-as-indexed-language, past-tense history.
- The migrate removal removes `modernc.org/sqlite` — the only reason that dependency exists in go.mod.

</specifics>

<deferred>
## Deferred Ideas

- `bench.yml`'s own de-coupling (removing the comparison runner) — Phase 6.
- `docs/BENCHMARKS.md` rewrite — Phase 6.
- The memory sweep (MEM-01/02) — Phase 6.
- Whether to restore migrate later — a future decision; nothing is deleted from git history.

</deferred>

---

*Phase: 05-process-ci-in-tree-sweep*
*Context gathered: 2026-08-15*
