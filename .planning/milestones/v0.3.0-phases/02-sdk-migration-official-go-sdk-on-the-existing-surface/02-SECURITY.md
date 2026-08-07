---
phase: 02
slug: sdk-migration-official-go-sdk-on-the-existing-surface
status: verified
# threats_open = count of OPEN threats at or above workflow.security_block_on severity (the blocking gate)
threats_open: 0
asvs_level: 1
created: 2026-08-06
---

# Phase 02 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

Register origin: **authored at plan time**. All 5 plans (`02-01` … `02-05`) carry a
parseable `<threat_model>` block, so this audit verifies that each declared mitigation
is present in the implementation — it does not retroactively scan for new threats.
No summary emitted a `## Threat Flags` section; the plan registers are the complete
declared source. Row count reconciles to **25 unique** `T-02-*` rows (02-01: 7, 02-02: 4,
02-03: 3, 02-04: 6, 02-05: 5) = `T-02-01` … `T-02-24` plus `T-02-SC`. Dispositions:
22 `mitigate`, 2 `accept`, 1 `transfer`. No compound-category row exists in this phase.

> **⚠ Threat-ID collision — read before cross-referencing.** `01-SECURITY.md` already
> contains rows labelled `T-02-01` … `T-02-05` and `T-02-SC`. Those are phase 01's
> **plan-02** threats and are entirely unrelated to the identically-spelled IDs in this
> file, because phase 01 namespaced its register by *plan* number while phases 02–05
> namespace by *phase* number. Never merge, dedup, or grep threat IDs across these two
> documents.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| MCP client → `tools/call` arguments | Every tool argument, including `path`, is attacker-influenceable: the client is an AI agent that may be processing hostile content. Crosses into `openEngine` | Filesystem paths, tool arguments |
| MCP client → `initialize` params | `protocolVersion` and `clientInfo.name`/`version` are self-reported and reach the stderr session line | Client identity strings, protocol version tokens |
| Client-supplied `path` → `confineToRepoRoot` | `error-confinement-reject.golden` is the frozen proof that the trust boundary still rejects | Filesystem paths |
| Process stdout | Reserved exclusively for the JSON-RPC transport; any other write corrupts the session | JSON-RPC frames only |
| Error text → agent context | Whatever a failed tool call returns becomes model-visible text under go-sdk, where under mark3labs a protocol error would not have reached the model at all | Error message text |
| Go module graph / Go module proxy → dependency closure | go-sdk adds `segmentio/encoding`, `segmentio/asm`, `golang.org/x/oauth2`, `golang.org/x/time` to the closure; `go mod tidy` resolves and pins them, and `go.sum` is the integrity control | Module content hashes |
| Test subprocess → real binary | `test/integration` spawns the real `serve --mcp`; substituting an in-process transport would silently remove the boundary these tests exist to cross | Process argv, wire bytes |
| Pull-request author → the anti-regeneration guard | The author controls the entire diff the guard classifies, including `go.mod`. Any exemption is, by construction, author-influenceable input | Changed-file list, go.mod diff |
| CI job exit code → merge decision | `transcript-freeze` is a required PR leg; a guard that passes wrongly is indistinguishable from no guard | Exit status |
| Archtest guard → merge decision | Both guards are required test legs. A guard that passes vacuously is indistinguishable from no guard, and this is the exact change that could make both vacuous at once | Test verdict |
| Regenerated golden file → future regression detection | A golden regenerated without review asserts whatever the new code happens to do, forever. `PITFALLS.md` Testing Trap C names this the highest-value trap in this phase | Frozen transcript bytes |
| Frozen transcript bytes → Phase 3's baseline | Phase 3 reads this corpus as its comparison target; anything wrong that lands here propagates | Frozen transcript bytes |

---

## Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation | Status |
|-----------|----------|-----------|----------|-------------|------------|--------|
| T-02-01 | Tampering | `openEngine`'s `argPath` parameter (`internal/mcp/tools.go`) | high | mitigate | `tools.go:67` `confineToRepoRoot` precedes `:71` `query.OpenAt` — and `:71` is the **only** `query.OpenAt` call site in `internal/mcp`, so all 9 `openEngine` call sites (`:147`, `:352`, `:368`, `:385`, `:402`, `:419`, `:443`, `:460`) are covered. `TestOpenEnginePathConfinedToRepoRoot` (`server_test.go:211`) and `TestConfinementAnchoredOnRepoRootNotStartPath` (`:242`) both re-run PASS | closed |
| T-02-02 | Information Disclosure | `confineToRepoRoot`'s `%q`-embedded rejected path | low | accept | See AR-02-01 | closed |
| T-02-03 | Spoofing | stderr session line fields (`requested`, `negotiated`, `client`) | medium | mitigate | `session_line.go:32-35` applies `sanitizeClientField` to all four fields; reached from `AddReceivingMiddleware` (`server.go:571`) at `server.go:635`. `TestSessionLineSanitizesHostileClientInfo` (`session_line_test.go:202`), `TestSanitizeClientFieldTruncatesOnRuneBoundary` (`:72`) | closed |
| T-02-04 | Denial of Service | malformed or oversized `tools/call` arguments | low | mitigate | `"additionalProperties":false` present on **all 8** tool schemas in the frozen corpus (`toolslist-repeat.golden` 8/8, and every `tools/list`-bearing transcript). `TestUnknownArgumentIsRejected` (`error_mapping_test.go:126`) and `TestMissingRequiredArgumentIsToolVisibleError` (`:95`) PASS. See advisory UF-2 for the pre-framing boundary this does not reach | closed |
| T-02-05 | Tampering | stdout purity under the new SDK's transport | high | mitigate | `TestNoStdoutNoiseInServeReachablePackages` (`internal/graphstore/archtest/stdout_confinement_test.go:269`) PASS; `scanForStdoutViolations:208-222` applies the allowlist **only** to the `os.Stdout` predicate, never to `fmt.Print*` / `log.SetOutput`. `assertFramingInvariant` (`test/wireoracle/oracle_test.go:530-556`) fatals on any stdout line failing `json.Unmarshal` or `jsonrpc != "2.0"`; full suite PASS (28 scenarios, 20.8 s). See advisory UF-1 — the guard is **not** literally unmodified | closed |
| T-02-06 | Tampering | new transitive modules in the dependency closure | medium | transfer | Receiving plan (02-04) demonstrably performed it: `02-04-SUMMARY.md:118-119`, `:162`, `:177-181` (`go mod tidy -e`, govulncheck, syft SBOM). Independently re-verified by the auditor: `go.sum:319` go-sdk, `:369`/`:371` segmentio, `:511`/`:530` x/oauth2 + x/time; `govulncheck ./...` re-run → 0 reachable vulnerabilities | closed |
| T-02-07 | Tampering | `sdkSwapExemption` in the freeze guard | high | mitigate | `tools/transcriptfreeze/classify.go:180-194` requires `removedMark3labs && addedGoSDK`, reusing `mcpSDKModulePrefixes[0]`/`[1]` by index. Three `Violation: true` neighbours (`classify_test.go:131`, `:137`, `:143`) plus `TestSDKSwapExemptionRequiresActualDiffLines` (`:181`). Self-expired: `rg -c mark3labs go.sum` → 0 | closed |
| T-02-08 | Repudiation | exemption firing without an audit trail | medium | mitigate | `main.go:82-86` writes `verdict.Reason` to stderr (`main.go:41` passes `os.Stderr`) before `return 0`; the notice names transcripts plus `internal/mcp` files (`classify.go:281-295`). Guard leg still wired at `.github/workflows/ci.yml:418` | closed |
| T-02-09 | Elevation of Privilege | runtime bypass of the exemption logic | high | mitigate | `rg -n 'Getenv\|os\.Environ\|flag\.' tools/transcriptfreeze/classify.go` → **zero matches**; `rg -n 'os\.\|ReadFile\|Stat\('` on the same file → zero matches. `sdkSwapExemption`'s sole input is the diff string | closed |
| T-02-10 | Tampering | the guard's trigger set as a floor | low | accept | See AR-02-02 | closed |
| T-02-11 | Information Disclosure | SDK-authored error text reaching agent context | medium | mitigate | Four tests pin per-class text (`error_mapping_test.go:82`, `:115`, `:149`, `:178`). SDK-authored texts are frozen and echo only caller input: `error-unknown-tool.golden` → `unknown tool "codegraph_bogus_tool"`. Prohibition held — `git show --name-only 946f12c` = `internal/mcp/error_mapping_test.go` only | closed |
| T-02-12 | Tampering | handler error becoming a protocol error | high | mitigate | `TestHandlerErrorIsToolResultNotProtocolError` (`error_mapping_test.go:60-85`) drives the real `confineToRepoRoot` path through a live session and asserts `outside` — PASS | closed |
| T-02-13 | Spoofing | a test asserting a weaker property than it claims | high | mitigate | All four tests assert three properties: nil session error, `IsError`, and `len(Content)==1` + `*mcp.TextContent` + substring (`:69-84`, `:102-121`, `:136-151`, `:165-180`) | closed |
| T-02-14 | Tampering | dependency closure integrity | medium | mitigate | Same `go.sum` pins as T-02-06; auditor-run `govulncheck` green over the final closure | closed |
| T-02-15 | Spoofing | VRFY-02 guard passing vacuously | high | mitigate | `internal/mcp/archtest/protocol_version_selftest_test.go:98` plants `mcp.MetaKeyProtocolVersion`; zero-`TypeErrors` assertion at `:109-112`; test PASS | closed |
| T-02-16 | Spoofing | SDK-confinement guard passing vacuously | high | mitigate | `internal/cli/archtest/mcp_sdk_selftest_test.go:85-87` plants the go-sdk import plus a used reference; `TypeErrors` guard `:103-106`; `assertMCPSDKImporterExists` (`mcp_sdk_confinement_test.go:63`, invoked `:120`). Both tests PASS | closed |
| T-02-17 | Elevation of Privilege | silent reintroduction of the old SDK | medium | mitigate | `github.com/mark3labs/mcp-go` retained in **both** lists: `internal/cli/archtest/mcp_sdk_confinement_test.go:40` and `tools/transcriptfreeze/classify.go:52` | closed |
| T-02-18 | Tampering | golden-parity suite weakened during migration | high | mitigate | `go test ./testdata/golden/...` PASS (13.7 s). `git show 3bae505 -- golden_parity_test.go` removes only `NewInProcessClient` / client `Initialize` setup checks; equality assertions retained (`:742`, `:1001`, `:1208`). Count 114→112 documented as a Rule-1 deviation (`02-04-SUMMARY.md:45`, `:144`) | closed |
| T-02-19 | Information Disclosure | stale attribution in `NOTICE` | low | mitigate | `NOTICE:66` — `modelcontextprotocol/go-sdk` (Apache-2.0); zero `mark3labs` matches in the file | closed |
| T-02-20 | Tampering | wholesale regeneration laundering a regression | high | mitigate | `rg UPDATE_TRANSCRIPTS` over the tree → **zero matches**; no regeneration switch exists. Commit `f4c9052`'s message attributes every changed line to nine named causes; the human checkpoint fired and escalated cause #9 (`02-05-SUMMARY.md:174-183`) | closed |
| T-02-21 | Spoofing | the frozen suite passing vacuously | high | mitigate | `02-05-SUMMARY.md:87-91`, `:134` — Mutation 1 applied → 23/23 `TestFrozenTranscriptsMatch` plus 23/23 framing failures → reverted → green, `server.go` byte-identical. Corroborated at HEAD: the stdout-purity guard Mutation 1 trips PASSES, and permanent guards `TestScenarioCountIsExact` (`oracle_test.go:161`) and `TestEmptyTranscriptNeverMatches` (`:71`) remain | closed |
| T-02-22 | Tampering | the confinement-rejection golden silently moving | high | mitigate | `git show f4c9052 -- error-confinement-reject.golden`: the rejection line is an **unchanged context line**; only the initialize key-order line moved (cause #1) | closed |
| T-02-23 | Repudiation | re-freeze causes unattributed | medium | mitigate | `git log f4c9052 --format=%B` carries all nine causes, both non-events with observed values, both anchor codes (`-32601` / `-32602`), and the SDK-05 finding | closed |
| T-02-24 | Elevation of Privilege | harness scope creep during the swap | high | mitigate | `git diff --name-only abea7ea d91ebb6 -- test/wireoracle/` → exactly `scenarios.go`. `normalize.go` and `anchors.go` untouched across the entire phase | closed |
| T-02-SC | Tampering | Go module install (`go get modelcontextprotocol/go-sdk@v1.7.0`) | high | mitigate | `02-RESEARCH.md:45-63` Package Legitimacy Audit — every verdict `[VERIFIED]`, explicit "SUS: none"; the two `[ASSUMED]` grep hits are the explanatory note, not verdicts. `go.mod:14` pins v1.7.0; `go.sum:319-320` carries both `h1:` and `/go.mod` hashes. **This is phase 01's AR-01 coming due** — that acceptance was explicitly scoped "re-opens in Phase 2 when the SDK swap lands", and it is discharged here by audit rather than re-accepted | closed |

*Status: open · closed · open — below high threshold (non-blocking)*
*Severity: critical > high > medium > low — only open threats at or above workflow.security_block_on count toward threats_open*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| AR-02-01 | T-02-02 | `confineToRepoRoot`'s `%q` echo returns the caller's own supplied path and nothing else (`internal/mcp/tools.go:44`) — the rejected path is echoed deliberately so the caller can see what was refused. Verified unchanged by both the SDK migration and the re-freeze. The frozen transcript uses the portable literal `/codegraph-wire-oracle-outside-root`, so no host path enters the repository. Inherits the phase-01 reasoning for its own T-04-02. | seanb4t | 2026-08-06 |
| AR-02-02 | T-02-10 | The freeze guard's trigger set (`transcriptDirPrefix`, `serverDirPrefix`, `mcpSDKModulePrefixes`) is deliberately narrow and was **not** widened to accommodate the SDK-swap exemption — `git show 00b0603 -- classify.go` contains no `const`/`var` trigger-set line. The in-file disclosure at `classify.go:244-249` ("a floor, not a proof of innocence", including "Do NOT widen") and its emission at `:272` are themselves the mitigation: the narrowness is stated to reviewers rather than left implicit. | seanb4t | 2026-08-06 |

---

## Advisory — Unregistered Surface

Neither item below counts toward `threats_open`; both are implementation-introduced
surface with **no threat-register mapping**, recorded here rather than left latent. This
follows the repo's existing precedent, where the v1.0 phase-10 audit's "Advisory —
Unregistered Surface" section became backlog phase 999.4.

**UF-1 — the HYG-02 stdout guard gained an allowlist during implementation.**
`stdoutTransportWriterAllowlist` (`internal/graphstore/archtest/stdout_confinement_test.go:180-182`)
was added in phase-02 commit `aac7536`, so T-02-05's claim that the guard runs
"unmodified" is literally false. The mitigation still holds on the merits: the entry is
keyed on (package path, enclosing function `ServeStdio`), applies only to the
`os.Stdout` predicate, and `TestStdoutTransportAllowlistDoesNotOverSuppress` (same
commit) proves it cannot smuggle another reference — re-run by the auditor, PASS. It is
documented as a deviation at `02-01-SUMMARY.md:184-189` but was never mapped to a
threat ID.

**UF-2 — `stdinLingerReader` is new hand-written code on the untrusted-stdin boundary.**
`internal/mcp/server.go:186-244` (deviation 1, commit `db46e64`) replaced go-sdk's
`StdioTransport` with `IOTransport` plus a custom line-buffering reader, because go-sdk
has a real stdin-close race that deterministically loses in-flight responses. `Read`
uses `bufio.Reader.ReadBytes('\n')` (`:212`), which **accumulates without bound**, so a
client sending a very long line with no newline is a memory-growth path that T-02-04's
schema-layer mitigation cannot reach — `applySchema` runs only after a full line is
framed. `waitForDrain` is bounded (`stdinLingerGrace = 5 s`, `:180`), so there is no
unbounded-hang path, and the local trust model (one stdio subprocess serving one local
agent client) mitigates severity. The component is nonetheless unregistered and warrants
a register row in a future phase.

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-08-06 | 25 | 25 | 0 | Claude Opus 5 (`gsd-security-auditor` subagent, `/gsd-secure-phase 02`, ASVS L1) |

### Audit 2026-08-06 — notes

Audited retroactively, after the phase was verified and the milestone archived, to close
the coverage gap left by phases 02–05 shipping without a `SECURITY.md`. The
`gsd-security-auditor` subagent was spawned rather than taking the workflow's L1
short-circuit. Test suites (`internal/mcp/archtest`, `internal/cli/archtest`,
`testdata/golden`, `test/wireoracle`, `govulncheck ./...`) were **re-run by the auditor**
rather than quoted from the summaries.

**Phase 01's AR-01 is discharged here, not inherited.** Phase 01 accepted the
package-manager-install risk only because `go.mod` was deliberately frozen for its whole
duration, and its rationale says in as many words: "Re-opens in Phase 2 when the SDK swap
lands." Phase 02 is where `go get modelcontextprotocol/go-sdk@v1.7.0` actually ran, so
`T-02-SC` is that reopened risk coming due. It is closed on the Package Legitimacy Audit
plus `go.sum` hash pinning, not on phase 01's "not applicable".

**One threat's mitigation text is literally inaccurate while remaining materially sound.**
T-02-05 asserts the stdout archtest runs "unmodified"; it does not (UF-1). The claim was
checked rather than accepted, and the guard's substance survives — but the register text
overstates. Recorded rather than quietly reconciled.

**The transfer disposition was verified at the receiving end.** T-02-06 defers the
module-closure audit to plan 02-04. A transfer is only closed if the receiving party
demonstrably performed the work, so 02-04's `go mod tidy -e` / govulncheck / SBOM run was
confirmed in its summary *and* independently re-verified against `go.sum` and a fresh
`govulncheck` run.

**Scope note (L1).** ASVS L1 verifies that each declared mitigation is *present*. It does
not perform L2 boundary-placement analysis or L3 end-to-end trace verification.

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-08-06
