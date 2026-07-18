---
phase: 6
slug: rendering-seam-pretty-status-files
status: verified
# threats_open = count of OPEN threats at or above workflow.security_block_on severity (the blocking gate)
threats_open: 0
asvs_level: 1
created: 2026-07-17
---

# Phase 6 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

Register authored at plan time (all 3 PLANs carry `<threat_model>` blocks) → verified at L1 (grep-depth) via the short-circuit rule (`threats_open: 0`, `register_authored_at_plan_time: true`, `asvs_level: 1`). No auditor spawn required. All mitigations confirmed against implemented source + green gates.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| build/test tier → agent path | The TUI-01 archtest is the enforcement point: no styling (charm) code reachable from the serve-side (`internal/mcp`/`internal/query`/`internal/graphstore`/`internal/daemon`/`internal/watch`/`internal/indexer`) closure | import graph (compile-time) |
| dependency supply chain → go.mod | Adding `charm.land/lipgloss/v2` via the vanity domain / module proxy | Go module + checksum |
| indexed repo content → styled terminal output | File paths / dir names / language strings from an arbitrary (possibly adversarial) indexed repo flow into the new pretty renderer bound for a human's real terminal | untrusted bytes → TTY |
| CLI RunE → stdout / stderr streams | The isTTY decision references the real `os.Stdout`/`os.Stderr`; product output stays byte-clean; progress lands on stderr only | rendered text / ANSI |
| progress goroutine → stderr | Animated ticker frames must land on stderr only, never the stdout/MCP JSON-RPC stream (D-08, reuses Phase-4 HYG-02 discipline) | ANSI frames |

---

## Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation | Status |
|-----------|----------|-----------|----------|-------------|------------|--------|
| T-06-01 | Tampering / Info Disclosure | charm/ANSI reaching `internal/mcp` or `internal/query` (garbled/misleading bytes in an agent's context) | high | mitigate | TUI-01 archtest — build-time forbidden-set closure walk over the 6-package serve-reachable set; `go test ./internal/cli/present/archtest/...` green; `go list -deps` confirms query/mcp charm-free | closed |
| T-06-02 | Tampering | archtest becoming vacuously green after a refactor drops the only charm import | medium | mitigate | `assertCharmImporterExists` self-defeat guard (D-12) + guarded-package count guard (`import_graph_test.go:119,143`); a real importer (`present/styles.go`) satisfies it | closed |
| T-06-03 | Tampering | bare `charm.land/lipgloss` (no `/v2`) silently resolving to the wrong v1 module, defeating the archtest list + API assumptions | medium | mitigate | `/v2`-suffixed paths everywhere: forbidden-path list (`import_graph_test.go:52-54`), go.mod require `charm.land/lipgloss/v2 v2.0.5`; grep acceptance checks | closed |
| T-06-04 | Tampering | terminal escape/OSC-sequence injection via untrusted repo-derived strings (file paths / dir names containing raw control bytes) rendered by the new pretty path | low | mitigate | **Upgraded `accept`→`mitigate` by code-review CR-01.** `present.sanitizeControl` strips `unicode.IsControl` runes (incl. ESC) applied at all 3 pretty interpolation sites (`files.go:22,25,45`, commit `ebaab25`). Pretty-path only — the frozen plain path keeps byte-identity (TUI-02) and a non-TTY sink does not interpret escapes | closed |
| T-06-05 | Tampering | pretty path re-deriving content and diverging from the plain/golden bytes (byte-identity break) | high | mitigate | D-02 reuse-not-rederive (renderers consume `StatusResult`/`FilesResult` read-only); `internal/query/render_status.go` git-unmodified since Phase 2; byte-identity integration test + `go test ./testdata/golden/...` green | closed |
| T-06-06 | Information Disclosure | ANSI leaking into piped/agent output by relying on lipgloss auto-degrade | high | mitigate | D-04 explicit non-TTY plain-branch bypass at CLI boundary (`status.go:61`, `files.go:68` gate on `term.IsTerminal(os.Stdout.Fd())`); zero-ANSI assertion in the byte-identity integration test | closed |
| T-06-07 | Tampering | progress frames corrupting stdout / the MCP JSON-RPC stream | high | mitigate | D-08 stderr-only (`present.NewProgress(os.Stderr)`; writer takes an injected `io.Writer`); `progress_test` asserts zero stdout; init/index/sync are CLI-tier, never serve-reachable; stdout-purity + golden tests green | closed |
| T-06-08 | Denial of Service | ticker goroutine leak / non-deterministic teardown on a long or interrupted index | medium | mitigate | `Stop()` signals `stopCh` and waits on `doneCh` for the goroutine to return (`progress.go:51,73,79`); `defer Stop` covers error paths; no-goroutine-leak test + `go test -race` clean | closed |
| T-06-09 | Information Disclosure | ANSI progress frames appearing in piped/non-TTY output | medium | mitigate | Same `ChoosePresentation` gate evaluated against `os.Stderr`'s fd (`init.go:71`, `index.go:69`, `sync.go:47`) + NO_COLOR; non-TTY reachability test asserts zero ANSI on stderr | closed |
| T-06-SC | Tampering | supply-chain integrity of the new go.mod require | low | accept | Per RESEARCH Package Legitimacy Audit, `charm.land/lipgloss/v2` (official charmbracelet/lipgloss) and `golang.org/x/term` (Go team) are both OK/Approved; no `[ASSUMED]`/`[SUS]` package installed (`creack/pty` explicitly NOT added). Full CGo/govulncheck/SBOM closure audit deferred to Phase 8 (REL-01) | closed |

*Status: open · closed · open — below `high` threshold (non-blocking)*
*Severity: critical > high > medium > low — only open threats at or above `high` count toward threats_open*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| AR-06-01 | T-06-SC | New charm/x-term requires are official, approved origins; no assumed/suspect package installed. Full dependency-closure audit (CGo/govulncheck/SBOM/reproducible build) is Phase 8 REL-01 scope, not this phase. | secure-phase (L1) | 2026-07-17 |

*Accepted risks do not resurface in future audit runs.*

**Note:** T-06-04 (escape-injection) was dispositioned `accept` (pre-existing risk class) at plan time. Deep code review (CR-01) elevated it: the pretty path is a *new* injection surface, so it was **mitigated** (`sanitizeControl`, byte-safe, pretty-path only) rather than accepted. The frozen plain renderer still shares the underlying pre-existing gap (out of scope here — fixing it would break TUI-02 byte-identity); a holistic sanitize-at-the-data-layer pass is a candidate future security follow-up.

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-07-17 | 10 | 10 | 0 | secure-phase (L1 grep-depth, short-circuit; no auditor) |

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-07-17
