---
phase: 1
slug: behavioral-parity-explore-node
status: verified
# threats_open = count of OPEN threats at or above workflow.security_block_on severity (the blocking gate)
threats_open: 0
asvs_level: 1
created: 2026-07-17
---

# Phase 1 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.
> First-time (State-B) review, verified by gsd-security-auditor against the plan-time register.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| CLI arg → query engine | `explore`/`node` accept a user-supplied query string and optional `[path]`; tokenized and used to seed graph traversal | query text, path string |
| MCP tool input → shared Engine | `codegraph_explore` / `codegraph_node` args reach the *same* `Engine.Explore`/`Engine.Node` as the CLI (no separate, less-guarded path) | tool arguments (query, maxFiles, budget) |
| process → filesystem | source-file reads for node bodies resolve through `resolveSourcePath` (abs-reject + `Clean`/`Rel` + `EvalSymlinks` re-confinement) | file paths, file contents |
| process → git subprocess | golden-capture stamps a version via `codegraph version`; gated by an explicit `--version` check before spawn | version string |
| process → graphstore (Pebble) | queries read a read-only snapshot via the Reader; no filesystem or network egress on the query path | graph records (symbols, edges) |
| golden parity harness → frozen TS goldens | determinism/provenance assertions compare canonicalized output against frozen v1.3.1 goldens | captured explore/node/status payloads |

---

## Threat Register

30 threats (T-01-01…T-01-29 + the recurring T-01-SC supply-chain accept). Register authored at plan time across all 17 PLAN `<threat_model>` blocks; verified at ASVS L1 (grep/read-depth) by gsd-security-auditor — every `mitigate` located at a concrete call site, every `accept` documented in its PLAN.

| Threat ID | Category | Severity | Disposition | Status | Evidence |
|-----------|----------|----------|-------------|--------|----------|
| T-01-01 | Tampering | medium | mitigate | closed | `testdata/golden/capture.sh:44-47` version stamp from `codegraph version`; `mcp-capture.mjs:22-30` `REQUIRED_VERSION='1.3.1'` gated before spawn |
| T-01-02 | Information Disclosure | low | mitigate | closed | `capture.sh:65-70` `strip_json` (paths → `<CORPUS_PATH>`); `:75` `strip_sql_timestamps`; applied `:93-97,147` |
| T-01-03 | Tampering | low | accept | closed | `01-02-PLAN.md` — additive proto3 strings, fresh Pebble keys, no SchemaVersion bump |
| T-01-04 | EoP / Input Validation | high | mitigate | closed | `internal/query/tokenize.go:104-106,167-169` return `[]string{}` on empty/whitespace |
| T-01-05 | Denial of Service | low | accept | closed | `01-03-PLAN.md` — linear-time char-class regex over bounded CLI arg |
| T-01-06 | Tampering / Path Traversal | high | mitigate | closed | `internal/query/node.go:204-257` pure in-memory filter; file read gated by `resolveSourcePath :33-79` (abs-reject + Clean/Rel + `EvalSymlinks` re-confine `:65-76`) |
| T-01-07 | Denial of Service | medium | mitigate | closed | `internal/query/render_markdown.go:146-148` hardCap=16/bodyBudget=12000/listCap=20; enforced `:211,220,242-249` |
| T-01-08 | Denial of Service | medium | mitigate | closed | `internal/query/resolve.go:667-705` collapseEdges de-dups by (source,kind,target) |
| T-01-09 | Tampering | low | mitigate | closed | `internal/indexer/dispatch/implements.go:11-12,31` synthesized edges carry `Provenance="heuristic"`, distinct from `"ast"` |
| T-01-10 | Denial of Service | high | mitigate | closed | `internal/query/rwr.go:69,141` fixed 25 iters, O(iters·edges); input bounded upstream (`explore.go:305-386`) |
| T-01-11 | Tampering | high | mitigate | closed | `rwr.go:114-118` sorted seeds, `:69,141` fixed iters, `:74,163,170-172` 1e-9 rounding, `:189-194` deterministic tie-break |
| T-01-12 | Denial of Service | high | mitigate | closed | `internal/query/gather.go:187-190` Channel-1 trim to `searchLimit*2`; WR-05 empty-query guard |
| T-01-13 | Information Disclosure | low | accept | closed | `01-07-PLAN.md` — gather reads only graphstore snapshot via Reader; no FS access |
| T-01-14 | Denial of Service | medium | mitigate | closed | collapseEdges de-dup `resolve.go:667-705` (java/csharp extractors) |
| T-01-15 | Denial of Service | medium | mitigate | closed | collapseEdges de-dup `resolve.go:667-705` (python/ts extractors) |
| T-01-16 | Tampering | low | mitigate | closed | `pyextract.go:742-746` & `tsextract.go:579,880` nil/absent annotation emits no ref (no guessed edge) |
| T-01-17 | Denial of Service | low | accept | closed | `01-10-PLAN.md` — pure arithmetic over searchLimit-capped candidate set |
| T-01-18 | Denial of Service | high | mitigate | closed | `internal/query/expand.go:37-43` ExpandMaxNodes=200/TraversalDepth=3/MinScore=0.2/GlueNodeCap=60; enforced `:311,342,350-359,521-523` |
| T-01-19 | Denial of Service | medium | mitigate | closed | `internal/query/seeding.go:67` seedTokenMaxCount=16; `:70,299-301` largeOverloadCorroboratedCap=4; `:304` top-1 fallback |
| T-01-20 | Denial of Service | low | accept | closed | `01-13-PLAN.md` — buried-rescue bounded to signature-type seeds, score≥3 cutoff |
| T-01-21 | Denial of Service | low | accept | closed | `01-14-PLAN.md` — pure comparison/filter over bounded post-scoring set |
| T-01-22 | Tampering | high | mitigate | closed | `internal/query/explore_gate.go:157,188,211` epsilon mass tier; `:196-236` 5-tier deterministic sort tail |
| T-01-23 | Tampering | medium | mitigate | closed | `internal/cli/index.go` `--force` re-extracts every file; missing-kind verification recorded (`01-15-SUMMARY.md`) |
| T-01-24 | Tampering | low | accept | closed | `01-15-PLAN.md` — additive strings at fresh keys, no SchemaVersion bump |
| T-01-25 | EoP / Input Validation | high | mitigate | closed | `internal/query/explore.go:241-243` `TrimSpace(query)==""` → error at top of Explore (CLI + MCP `mcp/tools.go:117`); tokenizer second layer |
| T-01-26 | Denial of Service | high | mitigate | closed | `explore.go:244-248` validate/clampMaxFiles + `:480-486` `clampExploreBudget[1,20]`; subgraph caps; RWR fixed iters |
| T-01-27 | Tampering | medium | mitigate | closed | No ANSI in render path (`render_status_test.go:305` asserts no `\x1b`); MCP `mcp/tools.go:117,243` calls the same `eng.Explore`/`eng.Node` as CLI |
| T-01-28 | Tampering | high | mitigate | closed | `testdata/golden/golden_parity_test.go` asserts ordering/membership/counts via `assertSubset` after canonicalization; float sizes get plausibility not raw equality (`:788-792`); vs FROZEN TS goldens |
| T-01-29 | Repudiation | medium | mitigate | closed | `golden_parity_test.go:7-26` four divergences documented; harness skips documented-divergence surfaces |
| T-01-SC | Tampering (supply chain) | low | accept | closed | All 17 PLAN threat_models — no new dependencies added across the phase (stdlib + pre-existing grammars) |

*Status: open · closed · open — below high threshold (non-blocking)*
*Severity: critical > high > medium > low — only open threats at or above `workflow.security_block_on` (high) count toward threats_open*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

**High-severity threats (the blocking tier): T-01-04, T-01-06, T-01-10, T-01-11, T-01-12, T-01-18, T-01-22, T-01-25, T-01-26, T-01-28 — all mitigate, all verified present at a concrete call site (0 open).**

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| AR-01-01 | T-01-03 | Additive proto3 strings at fresh Pebble keys, no SchemaVersion bump — no read-path corruption. | secure-phase (plan-time disposition) | 2026-07-17 |
| AR-01-02 | T-01-05 | Linear-time char-class regex over a bounded CLI argument — no catastrophic backtracking surface. | secure-phase (plan-time disposition) | 2026-07-17 |
| AR-01-03 | T-01-13 | Gather reads only a graphstore snapshot via the Reader; no filesystem access, nothing sensitive to disclose. | secure-phase (plan-time disposition) | 2026-07-17 |
| AR-01-04 | T-01-17 | Pure arithmetic over a searchLimit-capped candidate set — bounded by construction. | secure-phase (plan-time disposition) | 2026-07-17 |
| AR-01-05 | T-01-20 | Buried-rescue expansion bounded to signature-type seeds with a score≥3 cutoff. | secure-phase (plan-time disposition) | 2026-07-17 |
| AR-01-06 | T-01-21 | Pure comparison/filter over a bounded post-scoring set. | secure-phase (plan-time disposition) | 2026-07-17 |
| AR-01-07 | T-01-24 | Additive strings at fresh keys, no SchemaVersion bump. | secure-phase (plan-time disposition) | 2026-07-17 |
| AR-01-08 | T-01-SC | No new dependencies added across the entire phase (stdlib + pre-existing grammars/patterns); supply-chain surface unchanged. | secure-phase (plan-time disposition) | 2026-07-17 |

*Accepted risks do not resurface in future audit runs.*

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-07-17 | 30 | 30 | 0 | gsd-security-auditor (opus, ASVS L1) |

State-B first-time review. Register authored at plan time (all 17 PLAN files carried `<threat_model>` blocks). 22 mitigate verified present in source at concrete call sites, 8 accept documented in their PLANs. 10 high-severity threats (the blocking tier) all mitigate + verified. Zero ESCALATE (no claimed-but-missing mitigations). No unregistered attack surface (no `## Threat Flags` in any of the 18 SUMMARY files). No implementation files modified.

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-07-17
