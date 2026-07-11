---
phase: 3
slug: query-engine-mcp-server
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-07-11
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
| QRY-01 | `query <search>` with `--kind`/`--limit`/`--json` | unit + golden-diff | `go test ./internal/query/... -run TestQuery` | ❌ W0 | ⬜ pending |
| QRY-02 | `node <symbol\|file>` detail / line-numbered file read | unit + golden-diff | `go test ./internal/query/... -run TestNode` | ❌ W0 | ⬜ pending |
| QRY-03 | `search` locations-only | unit | `go test ./internal/query/... -run TestSearch` | ❌ W0 | ⬜ pending |
| QRY-04 | `callers`/`callees` traversal | unit + golden-diff | `go test ./internal/query/... -run TestCallersCallees` | ❌ W0 | ⬜ pending |
| QRY-05 | `impact --depth` | unit + golden-diff | `go test ./internal/query/... -run TestImpact` | ❌ W0 | ⬜ pending |
| QRY-06 | `affected [files...]` (D-07 query-time derivation) | unit (no golden — D-07a) | `go test ./internal/query/... -run TestAffected` | ❌ W0 | ⬜ pending |
| QRY-07 | `files` browse (format/filter/pattern/depth) | unit (no golden — D-07a) | `go test ./internal/query/... -run TestFiles` | ❌ W0 | ⬜ pending |
| QRY-08 | `explore <query>` verbatim source + blast radius | unit + golden-diff (byte-exact markdown) | `go test ./internal/query/... -run TestExplore` | ❌ W0 | ⬜ pending |
| QRY-09 | `status --json` health/counts | unit + golden-diff (shape-normalized) | `go test ./internal/query/... -run TestStatus` | ❌ W0 | ⬜ pending |
| MCP-01 | `codegraph_explore` only default-visible tool | unit (server construction) | `go test ./internal/mcp/... -run TestDefaultToolVisibility` | ❌ W0 | ⬜ pending |
| MCP-02 | `CODEGRAPH_MCP_TOOLS` allowlist exposes companions | unit | `go test ./internal/mcp/... -run TestAllowlist` | ❌ W0 | ⬜ pending |
| MCP-03 | Zero tools when no `.codegraph/` | unit | `go test ./internal/mcp/... -run TestNoIndexZeroTools` | ❌ W0 | ⬜ pending |
| MCP-04 | Tool output shapes match golden corpus | golden-diff (extends `testdata/golden/`) | `go test ./testdata/golden/... -run TestGoldenParity` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/query/engine_test.go` — **shared test infrastructure, created once by plan 03-02** (Wave 2). It holds the package-`query` `copyFixture`/`indexFixture` harness (reusing `internal/indexer/testdata/gofixture` per `cli_test.go`'s `copyFixture` pattern) plus the engine-foundation tests. **Wave-3 plans 03-03/03-04/03-05 EXTEND it by reuse only** — they call `copyFixture`/`indexFixture` from `engine_test.go` at runtime and add their own cases to their **own isolated files** (`search_test.go`, `traverse_test.go`, `files_status_test.go` respectively). They MUST NOT re-create or edit `engine_test.go` (parallel Wave-3 execution would clobber it); `engine_test.go` is intentionally absent from 03-03/04/05's `files_modified`.
- [ ] `internal/mcp/server_test.go` — MCP-01/02/03 tool-registration logic (construct the server, introspect registered tool names — no live stdio transport needed)
- [ ] `testdata/golden/golden_parity_test.go` — MCP-04: run CLI commands against the `weft-go` corpus and diff vs `corpus/weft-go/*.json` with the D-05 normalizations (id fields ignored, edge-multiplicity tolerance, `status` field remapping, no `score` key)
- [ ] Concrete plan for making the `seanb4t/weft` golden-corpus source tree reachable by the parity test (README says "weft is cloned/available separately" — needs a CI-reproducible fetch/submodule/skip-if-absent decision)

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

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 30s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
