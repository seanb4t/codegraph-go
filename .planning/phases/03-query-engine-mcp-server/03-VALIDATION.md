---
phase: 3
slug: query-engine-mcp-server
status: validated
nyquist_compliant: true
wave_0_complete: true
created: 2026-07-11
validated: 2026-07-11
---

# Phase 3 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution. Derived from 03-RESEARCH.md §Validation Architecture.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go stdlib `testing` (stdlib-only convention — no testify in project code) |
| **Config file** | none — plain `go test` |
| **Quick run command** | `go test ./internal/query/... ./internal/mcp/... ./internal/cli/...` |
| **Full suite command** | `go test ./...` (includes `testdata/golden/golden_test.go` and `internal/graphstore/archtest`) |
| **Estimated runtime** | ~30 seconds (quick); full suite dominated by golden parity + pebble store tests |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/query/... ./internal/mcp/... ./internal/cli/...`
- **After every plan wave:** Run `go test ./...`
- **Before `/gsd-verify-work`:** Full suite must be green (including `archtest` and `golden_test.go`)
- **Max feedback latency:** ~30 seconds

---

## Per-Task Verification Map

| Req ID | Behavior | Test Type | Automated Command | File Exists | Status |
|--------|----------|-----------|-------------------|-------------|--------|
| QRY-01 | `query <search>` with `--kind`/`--limit`/`--json` | unit + golden-diff | `go test ./internal/query/... -run TestQuery` | ✅ | ✅ green |
| QRY-02 | `node <symbol\|file>` detail / line-numbered file read | unit + golden-diff | `go test ./internal/query/... -run TestNode` | ✅ | ✅ green |
| QRY-03 | `search` locations-only | unit | `go test ./internal/query/... -run TestSearch` | ✅ | ✅ green |
| QRY-04 | `callers`/`callees` traversal | unit + golden-diff | `go test ./internal/query/... -run TestCallersCallees` | ✅ | ✅ green |
| QRY-05 | `impact --depth` | unit + golden-diff | `go test ./internal/query/... -run TestImpact` | ✅ | ✅ green |
| QRY-06 | `affected [files...]` (D-07 query-time derivation) | unit (no golden — D-07a) | `go test ./internal/query/... -run TestAffected` | ✅ | ✅ green |
| QRY-07 | `files` browse (format/filter/pattern/depth) | unit (no golden — D-07a) | `go test ./internal/query/... -run TestFiles` | ✅ | ✅ green |
| QRY-08 | `explore <query>` verbatim source + blast radius | unit + golden-diff (byte-exact markdown) | `go test ./internal/query/... -run TestExplore` | ✅ | ✅ green |
| QRY-09 | `status --json` health/counts | unit + golden-diff (shape-normalized) | `go test ./internal/query/... -run TestStatus` | ✅ | ✅ green |
| MCP-01 | `codegraph_explore` only default-visible tool | unit (server construction) | `go test ./internal/mcp/... -run TestDefaultToolVisibility` | ✅ | ✅ green |
| MCP-02 | `CODEGRAPH_MCP_TOOLS` allowlist exposes companions | unit | `go test ./internal/mcp/... -run TestAllowlist` | ✅ | ✅ green |
| MCP-03 | Zero tools when no `.codegraph/` | unit | `go test ./internal/mcp/... -run TestNoIndexZeroTools` | ✅ | ✅ green |
| MCP-04 | Tool output shapes match golden corpus | golden-diff (extends `testdata/golden/`) | `go test ./testdata/golden/... -run TestGoldenParity` | ✅ | ✅ green |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [x] `internal/query/engine_test.go` — **shared test infrastructure, created once by plan 03-02** (Wave 2). It holds the package-`query` `copyFixture`/`indexFixture` harness (reusing `internal/indexer/testdata/gofixture` per `cli_test.go`'s `copyFixture` pattern) plus the engine-foundation tests. **Wave-3 plans 03-03/03-04/03-05 EXTEND it by reuse only** — they call `copyFixture`/`indexFixture` from `engine_test.go` at runtime and add their own cases to their **own isolated files** (`search_test.go`, `traverse_test.go`, `files_status_test.go` respectively). They MUST NOT re-create or edit `engine_test.go` (parallel Wave-3 execution would clobber it); `engine_test.go` is intentionally absent from 03-03/04/05's `files_modified`.
- [x] `internal/mcp/server_test.go` — MCP-01/02/03 tool-registration logic (construct the server, introspect registered tool names — no live stdio transport needed)
- [x] `testdata/golden/golden_parity_test.go` — MCP-04: run CLI commands against the `weft-go` corpus and diff vs `corpus/weft-go/*.json` with the D-05 normalizations (id fields ignored, edge-multiplicity tolerance, `status` field remapping, no `score` key)
- [x] Concrete plan for making the `seanb4t/weft` golden-corpus source tree reachable by the parity test (README says "weft is cloned/available separately" — needs a CI-reproducible fetch/submodule/skip-if-absent decision)

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Live MCP client sees `codegraph_explore` only by default; companions appear with `CODEGRAPH_MCP_TOOLS`; zero tools when no `.codegraph/` | MCP-01/02/03 | Real stdio handshake with an agent client is not scriptable as a Go unit test | Connect Claude Code to `codegraph serve --mcp` in a repo with/without `.codegraph/` and with/without the env allowlist; observe advertised tools (`checkpoint:human-verify`) |

---

## Security Domain (ASVS L1)

- **V5 Input Validation (applies):** clamp `--depth`/`--limit`/`--max-files` to documented maxima before traversal/allocation (DoS); confine `-p`/`-f` path args via `filepath.Clean` + repo-root confinement; MCP handlers apply identical validation to CLI (D-08b shares the engine).
- **Path-traversal defense-in-depth:** the `explore`/`node` "read fresh from disk" step resolves paths relative to the resolved `.codegraph/` parent repo root and rejects escapes.
- Not applicable this phase: V2 Auth, V3 Session, V4 Access Control, V6 Crypto (local CLI + local stdio only; no network auth surface — HTTP/SSE is v2/SERVER-01).

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 30s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** approved 2026-07-11

---

## Validation Audit 2026-07-11

| Metric | Count |
|--------|-------|
| Requirements audited | 13 (QRY-01…09, MCP-01…04) |
| COVERED (test exists + green) | 13 |
| PARTIAL | 0 |
| MISSING | 0 |
| Gaps found | 0 |
| Resolved | 0 (none needed) |
| Escalated to manual-only | 0 |

**Result: NYQUIST-COMPLIANT.** Every phase requirement has an automated test that exists and runs green. All 13 expected test functions were located in the codebase and executed:

- `TestQuery`/`TestSearch` (`internal/query/search_test.go`) — QRY-01/03
- `TestNode`/`TestExplore` (`internal/query/render_markdown_test.go`) — QRY-02/08
- `TestCallersCallees`/`TestImpact`/`TestAffected` (`internal/query/traverse_test.go`) — QRY-04/05/06
- `TestFiles`/`TestStatus` (`internal/query/files_status_test.go`) — QRY-07/09
- `TestDefaultToolVisibility`/`TestAllowlist`/`TestNoIndexZeroTools` (`internal/mcp/server_test.go`) — MCP-01/02/03
- `TestGoldenParity` + 7 subtests (`testdata/golden/golden_parity_test.go`) — MCP-04

Beyond the per-requirement map, the review-fix regression tests are also green (`TestQueryDefaultCapAtMaxLimit`, `TestSearchDefaultCapAtMaxLimit`, `TestQueryRejectsEmptyTerm`, `TestSearchRejectsEmptyTerm`, `TestExploreRejectsEmptyQuery`, `TestImpactSkipsDanglingEdgeInsteadOfFailing`, `TestQueryUnknownKindRejectedBeforeScan`), covering the CR-01/CR-02/WR-01…05 fixes. No new tests were generated — Wave 0 test infrastructure was fully built during execution.

**Manual-only note:** the live end-to-end MCP stdio-client handshake (MCP-01/02/03) is exercised at the unit level (server construction + tool introspection, plus the verifier's raw-JSON-RPC drive during phase verification); a real agent client connecting remains a recommended smoke test but is not a blocking gap — the behavior is automated-covered.
