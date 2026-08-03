---
phase: 4
slug: output-hygiene
status: verified
# threats_open = count of OPEN threats at or above workflow.security_block_on severity (the blocking gate)
threats_open: 0
asvs_level: 1
created: 2026-07-16
---

# Phase 4 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| Pebble internal diagnostics → process diagnostic channel | Pebble emits Infof/Errorf/Fatalf; this phase routes them so only real errors reach stderr and none reach stdout | Pebble-generated diagnostic strings (may include store paths — accepted, T-02-14 precedent) |
| serve-reachable package code → MCP stdout (JSON-RPC transport) | Any stray stdout write from a package reachable during `serve --mcp` corrupts the JSON-RPC frame stream a downstream agent parses | JSON-RPC frames only (transport integrity) |
| serve `--mcp` process → agent stdin/stdout | The agent parses stdout as a JSON-RPC frame stream; any non-frame byte is a transport-integrity break | JSON-RPC frames |
| CLI command → operator stderr | Diagnostics belong on stderr; noise clutter degrades the human/agent signal | Diagnostic text |

---

## Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation | Status |
|-----------|----------|-----------|----------|-------------|------------|--------|
| T-04-01 | Information Disclosure | quietLogger.Errorf/Fatalf provenance-prefixed messages | low | accept | Messages format Pebble's own internally-generated diagnostic strings, not user/attacker input; store-path disclosure in diagnostics is an already-accepted exception (T-02-14 precedent). Deep review (iter 2, clean) confirmed no new sensitive field is echoed. | closed |
| T-04-02 | Denial of Service | quietLogger.Fatalf → os.Exit(1) | low | accept | Pebble Fatalf sites are internal invariant/corruption checks, not attacker-reachable; D-02 deliberately preserves fatal semantics — hiding a real corruption signal is worse than exiting (`internal/graphstore/logger.go` doc comment + Fatalf impl). | closed |
| T-04-03 | Tampering / Information Disclosure | Diagnostic text leaking onto stdout (JSON-RPC transport) from a serve-reachable package | medium | mitigate | Build-time archtest `TestNoStdoutNoiseInServeReachablePackages` (`internal/graphstore/archtest/stdout_confinement_test.go:205`) fails on any `os.Stdout` / bare `fmt.Print*` / `log.SetOutput` reference across the full serve-reachable import closure (`packages.NeedDeps` walk, CR-01 fix `21e47b9`), with a transitive-dependency regression self-test proving the closure scan can fail. | closed |
| T-04-04 | Tampering / Information Disclosure | stdout of a real `serve --mcp` session | medium | mitigate | Raw-stdio frame-purity harness `TestServeMCPStdoutIsPureJSONRPC` (`test/integration/mcp_stdout_purity_test.go:71`) reads every stdout byte via its own pipe (not mcp-go's silently-skipping client), fails on the first non-JSON-RPC line, asserts the tools/call response is not an error (WR-02 `cbd134d`), and checks `scanner.Err()` after the scan loop (`8a732cc`). Fail-capability mutation-proven. | closed |
| T-04-05 | Information Disclosure | `sync` stderr noise | low | mitigate | `TestSyncStderrNoPebbleNoise` (`test/integration/sync_noise_test.go:25`) drives the real binary's `sync` and asserts absence of Pebble noise shapes on stderr (noise-shape absence, not emptiness — D-09); mutation-proven (reverting the logger injection reintroduces real noise and turns it red). | closed |
| T-04-SC | Tampering | npm/pip/cargo/go installs (supply chain) | low | accept | No new packages this phase — RESEARCH Package Legitimacy Audit: "Not applicable." Only already-pinned deps touched (`pebble/v2@v2.1.6`, `golang.org/x/tools/go/packages`, stdlib). | closed |

*Status: open · closed · open — below high threshold (non-blocking)*
*Severity: critical > high > medium > low — only open threats at or above workflow.security_block_on count toward threats_open*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| AR-04-01 | T-04-01 | Pebble diagnostic strings are internally generated, not attacker input; path disclosure follows the accepted T-02-14 precedent | plan-time register (04-01-PLAN) | 2026-07-16 |
| AR-04-02 | T-04-02 | Fatalf exit on store corruption is the deliberately preserved safe behavior (D-02); suppressing it would hide corruption | plan-time register (04-01-PLAN) | 2026-07-16 |
| AR-04-03 | T-04-SC | Zero new dependencies introduced; supply-chain surface unchanged | plan-time register (all plans) | 2026-07-16 |

*Accepted risks do not resurface in future audit runs.*

---

## Residual Risks (documented, non-blocking)

- The stdout-confinement predicates cannot see indirect fd-1 writes (`os.NewFile(1, ...)`, `syscall.Write(1, ...)`) — documented at the guard (IN-01, `68e7a91`); the runtime frame-purity harness (T-04-04) is the compensating control.
- `diagWriterMu` guards the `diagWriter` variable, not concurrent `Write` calls on the writer it returns — currently inert (IN-02, documented in 04-REVIEW.md iter 2).

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-07-16 | 6 | 6 | 0 | /gsd-secure-phase (L1 short-circuit: plan-time register, threats_open 0, ASVS 1; evidence from deep review iter-2 clean + verifier 9/9) |

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-07-16
