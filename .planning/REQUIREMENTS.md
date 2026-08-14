# Requirements: CodeGraph Go — v0.11.0 Standalone Project Identity

**Defined:** 2026-08-13
**Core Value:** An agent user can uninstall TypeScript CodeGraph, install the Go binary, migrate their indexes, and everything works the same or better — faster, from a single verifiably-built binary. **As of v1.0 this is delivered, not aspirational.**

**Milestone goal:** Make codegraph-go read, test, and benchmark as a project in its own right — one legally-sufficient acknowledgment in `NOTICE` plus a single clause in README's License section, with *parity*, *upstream*, *drop-in* and origin-derived framing removed from every doc, template, workflow, script, code comment, identifier, and test fixture.

## v1 Requirements

### Attribution (ATTR)

- [ ] **ATTR-01**: A reader finds the origin acknowledgment in exactly one place — `NOTICE`, trimmed to the MIT copyright transcription plus one sentence of origin, with the drop-in / ported-heuristics / flag-parity argument removed
- [ ] **ATTR-02**: README's `## Relationship to the original` section is gone; the only origin mention in README is one past-tense clause inside `## License` linking to `NOTICE`
- [ ] **ATTR-03**: `LICENSE` stays verbatim MIT text and GitHub's license detection still reports `MIT` — verified live via `gh api repos/seanb4t/codegraph-go/license`, not assumed

### Documentation (DOCS)

- [ ] **DOCS-01**: A reader of `README.md`, `CONTRIBUTING.md`, `SECURITY.md`, `CODE_OF_CONDUCT.md` or `PARSER-DECISION.md` encounters no comparison framing — no *parity*, *upstream*, *drop-in*, or "the original" positioning
- [ ] **DOCS-02**: `docs/FLAG-PARITY.md` is deleted and `internal/cli/flag_parity_test.go` is removed with it, with nothing left in the tree referencing either
- [ ] **DOCS-03**: `docs/LANGUAGE-CAPABILITY-MATRIX.md` describes codegraph-go's own per-language capability on its own terms, without reference to another implementation's coverage
- [ ] **DOCS-04**: The remaining `docs/*` files (`RELEASE.md`, `RELEASE-PROCEDURES.md`, `MCP-2026-07-28-SCOPING.md`, `MCP-8-AGENT-AUDIT.md`) carry no retired framing

### Process & CI (PROC)

- [ ] **PROC-01**: A contributor filing an issue sees no comparison framing in any of the 5 issue templates (`bug_report`, `chore`, `config`, `enhancement`, `feature_request`)
- [ ] **PROC-02**: A contributor opening a PR sees none in `pull_request_template.md` or the 3 `PULL_REQUEST_TEMPLATE/*` variants
- [ ] **PROC-03**: Workflow job names, step names and comments carry no retired framing, and every workflow still passes `actionlint` and its own required status checks

### Code (CODE)

- [ ] **CODE-01**: Code and doc comments across `internal/`, `tools/` and `test/` carry no comparison framing, with TypeScript-as-indexed-language uses (`internal/indexer/tsextract`, language registries, capability matrix) explicitly preserved
- [ ] **CODE-02**: Test and fixture identifiers no longer encode comparison framing — `parity_*_test.go` and `TestGoldenParity*` renamed or removed — and `go test ./...` plus `go test ./testdata/golden/...` both pass
- [ ] **CODE-03**: `codegraph migrate` still imports a legacy `.codegraph/` SQLite index end-to-end; its help text, docs and comments describe it as a legacy-index import rather than a port or parity feature

### Fixtures & goldens (FIXT)

- [ ] **FIXT-01**: Corpus selection is decided by measurement — a blocking Phase 1 spike indexes candidate third-party MIT / Apache-2.0 repositories and records actual per-kind edge counts and per-language file counts, locking a final set that collectively covers all 9 `RANK_EDGES` kinds (`calls`, `implements`, `imports`, `extends`, `overrides`, `references`, `instantiates`, `returns`, `type_of`) and the 5 priority-4 languages (Go, Java, C#, Python, TS/JS)
- [x] **FIXT-02**: The locked corpora are fetched at exact pinned commit SHAs by a Taskfile target and restored from CI cache; no corpus source is vendored into the repository, and a re-fetch at the pinned SHA reproduces the same tree
- [ ] **FIXT-03**: No golden test self-skips in CI — the suite runs against the fetched corpora on every CI run, a fetch or cache failure fails the job loudly rather than skipping, and the job carries a positive assertion that the suite actually executed (scenario count, not merely a non-failing exit)
- [ ] **FIXT-04**: The `weft-go` and `colbymchenry-codegraph` corpora and their captured fixtures are removed, and `capture.sh` and `mcp-capture.mjs` are deleted in favor of the Go-side capture path (`testdata/golden/gocapture`)
- [ ] **FIXT-05**: The purpose-built behavioral corpus survives as `corpus/behavioral/` with its case map intact and its framing stripped — no targeted case (overloaded same-named symbols, multi-word queries, the `Test*`-heavy weakly-connected cluster, structural-beats-lexical ranking) is lost to the rename
- [ ] **FIXT-06**: Every golden is re-frozen from codegraph-go's own output against the locked corpora
- [ ] **FIXT-07**: The re-baselined golden suite is demonstrated RED against a confirmed-applied, byte-cleanly-reverted mutation per assertion family, proving it did not go vacuous

### Benchmarks (BENCH)

- [ ] **BENCH-01**: `docs/BENCHMARKS.md` publishes absolute throughput / query latency / peak-RSS figures with methodology, and no head-to-head comparison table
- [ ] **BENCH-02**: The comparison runner is removed from `tools/bench`, and the regression gate (`internal/bench.CheckRegression` + committed `baseline.json`) still fires on a real regression
- [ ] **BENCH-03**: `bench.yml` runs and publishes the absolute numbers without invoking another implementation

### Memory (MEM)

- [ ] **MEM-01**: Every engram spine record whose durable content asserts retired framing is superseded — not overwritten, not deleted — with a corrected record
- [ ] **MEM-02**: A session started after the sweep recalls no memory describing codegraph-go as a port or parity project in the present tense

## v2 Requirements

Deferred to a future milestone. Tracked but not in this roadmap.

### Documentation (DOCS)

- **DOCS-05**: A self-authored `docs/CLI-REFERENCE.md` replacing what `docs/FLAG-PARITY.md` used to carry, with its own drift guard asserting every registered flag is documented — deliberately deferred; this milestone deletes, a later one authors

### Vocabulary (VOCAB)

- **VOCAB-01**: A build-time guard failing on reintroduction of the retired vocabulary outside `NOTICE` — deliberately declined for this milestone (see Out of Scope), recorded here so the decision is visible rather than forgotten

## Out of Scope

| Feature | Reason |
|---------|--------|
| Rewriting `.planning/` archives (`milestones/*`, shipped ROADMAP/REQUIREMENTS, phase dirs) | An append-only record of what actually happened. Rewriting it would falsify the project's own history, and GSD's scope-sensitive parsers key on those structures |
| Rewriting `CHANGELOG.md` | release-please-owned generated file; hand-editing it breaks the tool that both writes and re-reads it |
| Removing `codegraph migrate` | The origin project is a live dependency there, not framing. Removing comparison language must not remove a capability existing users depend on |
| Renaming `internal/indexer/tsextract` or de-listing TypeScript as a supported language | TypeScript-the-indexed-language is product surface. Conflating it with TypeScript-the-origin-project would break real capability |
| A vocabulary drift guard (VOCAB-01) | A term blocklist either goes vacuous (rule `84d1gfpywd`) or fights legitimate uses like `tsextract`. One-time sweep plus review discipline is the chosen posture |
| Authoring `docs/CLI-REFERENCE.md` (DOCS-05) | This milestone deletes the comparison matrix; authoring a replacement reference is separate work with its own scope |
| Vendoring corpus source into the repository | Fetch-at-pinned-SHA with CI cache was chosen instead — avoids repo bloat and avoids adding redistribution obligations to the `NOTICE` file this milestone is trimming |
| Retiring the historical parity language in PROJECT.md's Key Decisions, Context, and Validated requirements | Those record what happened. Only the live *Compatibility constraint* was retired (2026-08-13), because it governs execution rather than recording it |

## Traceability

Which phases cover which requirements.

| Requirement | Phase | Status |
|-------------|-------|--------|
| ATTR-01 | Phase 4 | Pending |
| ATTR-02 | Phase 4 | Pending |
| ATTR-03 | Phase 4 | Pending |
| DOCS-01 | Phase 4 | Pending |
| DOCS-02 | Phase 4 | Pending |
| DOCS-03 | Phase 4 | Pending |
| DOCS-04 | Phase 4 | Pending |
| PROC-01 | Phase 5 | Pending |
| PROC-02 | Phase 5 | Pending |
| PROC-03 | Phase 5 | Pending |
| CODE-01 | Phase 5 | Pending |
| CODE-02 | Phase 2 | Pending |
| CODE-03 | Phase 5 | Pending |
| FIXT-01 | Phase 1 | Pending |
| FIXT-02 | Phase 1 | Complete |
| FIXT-03 | Phase 3 | Pending |
| FIXT-04 | Phase 2 | Pending |
| FIXT-05 | Phase 2 | Pending |
| FIXT-06 | Phase 2 | Pending |
| FIXT-07 | Phase 3 | Pending |
| BENCH-01 | Phase 6 | Pending |
| BENCH-02 | Phase 6 | Pending |
| BENCH-03 | Phase 6 | Pending |
| MEM-01 | Phase 6 | Pending |
| MEM-02 | Phase 6 | Pending |

**Phase index:**

| Phase | Name | Requirements |
|-------|------|--------------|
| 1 | Corpus Selection by Measurement | FIXT-01, FIXT-02 |
| 2 | Golden Harness Re-authoring & Re-freeze | CODE-02, FIXT-04, FIXT-05, FIXT-06 |
| 3 | Non-Vacuity Proof & Unconditional CI Execution | FIXT-03, FIXT-07 |
| 4 | Attribution & Documentation Sweep | ATTR-01, ATTR-02, ATTR-03, DOCS-01, DOCS-02, DOCS-03, DOCS-04 |
| 5 | Process, CI & In-Tree Sweep | PROC-01, PROC-02, PROC-03, CODE-01, CODE-03 |
| 6 | Benchmark De-coupling & Memory Sweep | BENCH-01, BENCH-02, BENCH-03, MEM-01, MEM-02 |

**Coverage:**

- v1 requirements: 25 total
- Mapped to phases: 25
- Unmapped: 0 ✓ (every v1 requirement maps to exactly one phase)

---
*Requirements defined: 2026-08-13*
*Last updated: 2026-08-13 — traceability populated by roadmap creation (6 phases)*
