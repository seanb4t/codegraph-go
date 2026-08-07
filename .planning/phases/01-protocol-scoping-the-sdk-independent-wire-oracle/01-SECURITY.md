---
phase: 01
slug: protocol-scoping-the-sdk-independent-wire-oracle
status: verified
# threats_open = count of OPEN threats at or above workflow.security_block_on severity (the blocking gate)
threats_open: 0
asvs_level: 1
created: 2026-08-05
---

# Phase 01 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

Register origin: **authored at plan time**. All 7 plans (`01-01` … `01-07`) carry a
parseable `<threat_model>` block, so this audit verifies that each declared mitigation
is present in the implementation — it does not retroactively scan for new threats.
No `## Threat Flags` sections were emitted by the summaries; the plan registers are
the complete source.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| MCP client → `serve --mcp` stdin | Untrusted JSON-RPC frames; `clientInfo` and `protocolVersion` are self-reported, attacker-influenceable strings | Client identity strings, protocol version tokens |
| `serve --mcp` → operator stderr | Always-on diagnostic stream carrying client-supplied values into a human- and machine-parsed surface | Sanitized client name/version, requested + negotiated versions, tool count |
| binary-under-test → wire oracle | Captured stdout/stderr bytes are untrusted input to the test harness | Raw wire frames |
| agent client → `tools/mcpaudit` stdin | Untrusted JSON-RPC frames carrying self-reported client identity | Client identity, capabilities |
| `tools/mcpaudit` → observation log file | Client identity and capabilities written to disk | Declared handshake fields, frame digests |
| audit procedure → developer's real agent configuration files | A measurement instrument temporarily inserted into the developer's working setup | Config file contents (mutated then restored) |
| agent config contents → `docs/MCP-8-AGENT-AUDIT.md` | SHA-256 digests of private agent config files published into a checked-in document | Digest pairs only |
| dependency upgrade → wire behavior | A `go.mod` bump can change what the server declares on the wire with no first-party source edit | Protocol version constant |
| scenario arguments → `confineToRepoRoot` | The `error-confinement-reject` scenario deliberately drives the client-supplied-path trust boundary | Client-supplied filesystem path |
| pull-request metadata → CI shell | `github.event.pull_request.base.ref` is contributor-influenceable and reaches a shell running `git` | Branch ref name |
| contributor diff → the freeze rule | The guard is the only structural barrier between "regenerate deliberately" and "regenerate to make a failure go away" | Changed-file list |
| the oracle → every later phase in this milestone | Phases 2 and 3 rely on this suite as their sole wire-level regression gate | Frozen transcripts |

---

## Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation | Status |
|-----------|----------|-----------|----------|-------------|------------|--------|
| T-01-01 | Tampering | `internal/mcp/session_line.go` — client-supplied fields formatted into the always-on stderr line | high | mitigate | `sanitizeClientField` (verified present) — control chars/spaces → `_`, invalid UTF-8 → U+FFFD, 256-byte rune-boundary truncation, `<unknown>` for empty | closed |
| T-01-02 | Denial of Service | `test/wireoracle/capture.go` — hung binary-under-test or surviving subprocess | medium | mitigate | Single `context.WithTimeout(ctx, 30*time.Second)` and `Process.Kill()` both verified present; `cmd.Wait()` via named-return `defer` | closed |
| T-01-03 | Tampering | `test/wireoracle` — captured stdout treated as data | medium | mitigate | Captured bytes only compared and `%q`-quoted; never executed, shell-interpolated, or written outside `testdata/wireoracle/transcripts/` | closed |
| T-01-04 | Information Disclosure | `testdata/wireoracle/transcripts/*.golden` could embed host paths | medium | mitigate | `repoDir` normalization → `<REPO>` placeholder, verified present in `test/wireoracle` | closed |
| T-01-SC | Tampering | package-manager installs | high | accept | See AR-01 | closed |
| T-02-01 | Information Disclosure | `tools/mcpaudit` observation log | high | mitigate | Log opened `0o600` (verified present), append-only; observation stops at end of initialize; malformed frames stored as SHA-256 + 64-byte `%q` prefix | closed |
| T-02-02 | Tampering | developer's agent MCP configuration files | high | mitigate | One client at a time; backup taken and hash-verified before any edit; restore registered on every exit path; `sha256-before` == `sha256-after` pair published in the audit doc | closed |
| T-02-03 | Spoofing | `tools/mcpaudit` sits inside the agent's trust path | medium | mitigate | `TestProxyIsByteExactInBothDirections` and `TestProxyPreservesCRLFAndUnterminatedFinalFrame` both verified present | closed |
| T-02-04 | Denial of Service | A frame-parse error aborting the proxy would break the live agent session | medium | mitigate | `ParseError`/`FrameDigest` fields verified present; proxy loop continues unconditionally | closed |
| T-02-05 | Information Disclosure | SHA-256 digests of private agent config files published | low | accept | See AR-02 | closed |
| T-02-SC | Tampering | package-manager installs | high | accept | See AR-01 | closed |
| T-03-01 | Tampering | Hostile `clientInfo.name` embedding a newline + session-line prefix to forge a second diagnostic line | high | mitigate | `TestSessionLineSanitizesHostileClientInfo` verified present — nine hostile shapes including the prefix-injection case | closed |
| T-03-02 | Tampering | `go.mod` bump silently moving the declared protocol version | high | mitigate | `TestNoExternalProtocolVersionConstantReferences` verified present in `internal/mcp/archtest` | closed |
| T-03-03 | Repudiation | A guard that passes vacuously certifies a property nobody verified | high | mitigate | Planted-defect companion (`protocol_version_selftest_test.go`) and self-defeat guard both verified present alongside `protocol_version_test.go` | closed |
| T-03-04 | Information Disclosure | Mid-rune truncation could emit invalid UTF-8 into a log stream | low | mitigate | `TestSanitizeClientFieldTruncatesOnRuneBoundary` verified present | closed |
| T-03-SC | Tampering | package-manager installs | high | accept | See AR-01 | closed |
| T-04-01 | Denial of Service | 23 subprocess spawns per suite run, any of which can wedge | medium | mitigate | Bounded by `Capture`'s 30s timeout, kill-on-error and unconditional `Wait` (same control as T-01-02) | closed |
| T-04-02 | Information Disclosure | Frozen transcripts checked into the repository | medium | mitigate | `error-confinement-reject` sends the portable literal `/codegraph-wire-oracle-outside-root` so the `%q`-quoted rejection echo cannot embed a host path; dedicated fixture is purpose-written source only | closed |
| T-04-03 | Elevation of Privilege | Client-supplied `path` escaping the index root | high | mitigate | `confineToRepoRoot` verified present; its rejection is frozen on the wire, so loosened confinement produces a transcript diff | closed |
| T-04-04 | Repudiation | An anchor that cannot fail certifies nothing | high | mitigate | Anchor set demonstrated red against a confirmed-applied altered expected value, failure message quoted in `01-04-SUMMARY.md` | closed |
| T-04-SC | Tampering | package-manager installs | high | accept | See AR-01 | closed |
| T-05-01 | Spoofing | Server silently coerces an unrecognized `protocolVersion` to its own latest | medium | accept | See AR-03 | closed |
| T-05-02 | Repudiation | Freezing an assumed error response would encode a behavior that never fires | high | mitigate | Scenario captured-and-frozen as the observed success, carries no error-code anchor; Legacy silent coercion recorded in both the transcript comment and the scoping doc | closed |
| T-05-03 | Information Disclosure | Frozen transcripts checked into the repository | medium | mitigate | Same `<REPO>` placeholder normalization as T-01-04 | closed |
| T-05-SC | Tampering | package-manager installs | high | accept | See AR-01 | closed |
| T-06-01 | Tampering | Command construction from contributor-influenceable pull-request metadata in CI | high | mitigate | **Verified at `.github/workflows/ci.yml:361-364`** — `github.event.pull_request.base.ref` reaches the step only via a step-level `env:` entry; the `run:` body is `task check:transcript-freeze` and contains no `${{` sequence; `git rev-parse --verify` rejects an unresolvable ref before use | closed |
| T-06-02 | Repudiation | A guard that cannot fire certifies a property nobody enforced | high | mitigate | `TestGuardPatternsMatchRealTree` verified present; guard demonstrated red on a confirmed-applied cross-change and green on a single-sided change | closed |
| T-06-03 | Denial of Service | Unusable input silently reported as a pass | medium | mitigate | `ParseChangedList` verified present and errors on empty input; the task body fails loudly when `TRANSCRIPT_FREEZE_BASE` is unset (verified in `Taskfile.yml`) | closed |
| T-06-04 | Tampering | A `status:` or `platforms:` key would silently skip the guard instead of failing it | medium | mitigate | Neither key present on `check:transcript-freeze`; `TestTaskfileGatesFailLoud` (`internal/upgrade/taskfile_shape_test.go:717`) rejects both by name | closed |
| T-06-SC | Tampering | package-manager installs | high | accept | See AR-01 | closed |
| T-07-01 | Repudiation | A vacuous oracle certifies a wire contract nobody verified | high | mitigate | Four one-time mutations demonstrated red against the real binary; permanent guards `TestScenarioCountIsExact` and `TestEmptyTranscriptNeverMatches` verified present | closed |
| T-07-02 | Tampering | An over-matching normalization rule silently erases field-presence evidence | high | mitigate | `TestNormalizationRulesMatchOnlyTheirOwnField`, `TestRuleTestCoverageScalesWithRules` and `TestNormalizeIsIdentityWhenNothingMatches` all verified present | closed |
| T-07-03 | Tampering | A mutation left un-reverted would ship a deliberate defect | high | mitigate | Each section records an explicit revert confirmation; acceptance required `git status --porcelain` empty and the suite green | closed |
| T-07-04 | Denial of Service | Mutation 1 transiently violated stdout purity in a serve-reachable package | low | mitigate | Edit reverted within the task; `internal/graphstore/archtest`'s stdout confinement guard independently failed while applied, corroborating the mutation landed | closed |
| T-07-SC | Tampering | package-manager installs | high | accept | See AR-01 | closed |

*Status: open · closed · open — below high threshold (non-blocking)*
*Severity: critical > high > medium > low — only open threats at or above workflow.security_block_on count toward threats_open*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| AR-01 | T-01-SC, T-02-SC, T-03-SC, T-04-SC, T-05-SC, T-06-SC, T-07-SC | No package-manager install occurs in this phase. `go.mod` is deliberately unchanged throughout — VRFY-04 pins `mark3labs/mcp-go v0.56.0` through end of phase, and the two new tools (`tools/mcpaudit`, `tools/transcriptfreeze`) use only the Go standard library. `01-RESEARCH.md` records the Package Legitimacy Audit as not applicable. **Re-opens in Phase 2** when the SDK swap lands. | seanb4t | 2026-08-05 |
| AR-02 | T-02-05 | A SHA-256 of a file whose contents are not published discloses nothing recoverable, and the digest pair is the only evidence that makes T-02-02's restoration mitigation auditable after the fact rather than a bare claim. The alternative — trusting the executor's assertion of restoration — is the exact failure mode both cross-AI reviewers flagged. No config contents, no paths beyond the client's own name, and no credentials are recorded. | seanb4t | 2026-08-05 |
| AR-03 | T-05-01 | `serve --mcp` silently coercing an unrecognized `protocolVersion` to its own latest is spec-permitted for a Legacy implementation and explicitly out of scope for this phase. The compensating control is VRFY-03's always-on session line, which makes the downgrade visible by reporting requested and negotiated separately. The rejection path lands in Phase 3 (SPEC-02). Recorded here rather than silently inherited. | seanb4t | 2026-08-05 |

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-08-05 | 30 | 30 | 0 | Claude Opus 5 (`/gsd-secure-phase 01`, ASVS L1) |

### Audit 2026-08-05 — notes

Short-circuit conditions met per the secure-phase workflow: `threats_open: 0`,
`register_authored_at_plan_time: true`, `asvs_level == 1`. L1 grep-depth
verification was therefore sufficient and no `gsd-security-auditor` subagent was
spawned. Each `mitigate` threat's named artifact (test function, constant,
or code path) was confirmed present in the tree; each `accept` threat's rationale
was transcribed into the Accepted Risks Log above rather than left implicit in a
plan file.

**Post-plan implementation delta.** `internal/mcp/session_line_concurrency_test.go`
was added during phase-01 UAT (`01-UAT.md` test 2, gap G-01-2), after the plan
registers were authored. It is test-only, introduces no new trust boundary, and
touches no production code. It strengthens the T-01-01 / T-03-01 neighborhood by
covering the one session-line seam the register named but no test measured: the
mutex guarding concurrent and repeated `AddAfterInitialize` writes to the shared
operator stderr stream. `Taskfile.yml`'s `test:race` gained `./internal/mcp/...`
in the same change. Neither warrants a new threat entry.

**Scope note (L1).** ASVS L1 verifies that each declared mitigation is *present*.
It does not perform L2 boundary-placement analysis or L3 end-to-end trace
verification. If `workflow.security_asvs_level` is raised to 2 or 3 before Phase 2,
this phase should be re-audited with the auditor subagent — the workflow's own
short-circuit rule declines to skip at those levels for exactly this reason.

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-08-05
