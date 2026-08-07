---
phase: 03
slug: 2026-07-28-spec-compliance
status: verified
# threats_open = count of OPEN threats at or above workflow.security_block_on severity (the blocking gate)
threats_open: 0
asvs_level: 1
created: 2026-08-06
---

# Phase 03 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

Register origin: **authored at plan time**. All 5 plans (`03-01` … `03-05`) carry a
parseable `<threat_model>` block, so this audit verifies that each declared mitigation
is present in the implementation — it does not retroactively scan for new threats.
No summary emitted a `## Threat Flags` section; the plan registers are the complete
source. Row count reconciles: 23 numbered rows (`T-03-01` … `T-03-23`) plus `T-03-SC`
repeated once per plan = **24 unique**. No compound categories are present in this
phase's register.

**Threat-ID scope note.** `T-03-*` here means *phase* 03. This differs from
`01-SECURITY.md`, where `T-01-*`…`T-07-*` are *plan* numbers within phase 01. Do not
merge threat IDs across phase files.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| MCP client → stdio JSON-RPC → `internal/mcp` middleware | Untrusted, possibly agent-relayed, possibly attacker-influenced request bytes. go-sdk owns framing and `_meta` validation upstream of any codegraph-go code (03-RESEARCH Q1, PROVEN-ABSENT) | Raw JSON-RPC frames |
| MCP client → per-request `_meta` → go-sdk's `validateRequestMeta` | Every field is attacker-shapeable. Wholly owned by go-sdk and runs before any codegraph-go code, which is why this plan's only correct move is assertion | Protocol version tokens, `_meta` fields |
| `internal/mcp` middleware → `mcp.DiscoverResult` fields | Server-authored response data crossing back out to the client; anything written here is client-visible on every discover | `ttlMs`, `cacheScope` |
| `ServerOptions.Instructions` → every `initialize` and `server/discover` result | Server-authored text crossing to every client on every session, and into committed repository artifacts. Anything interpolated here is published twice over | Instruction text |
| MCP client request → the per-request index re-check | New this phase. The re-check runs as a side effect of handling an untrusted request; whatever it reads from the request becomes attacker-influenceable | Index presence (boolean) |
| Construction-time `repoPath` → `confineToRepoRoot` | The pre-existing CR-02 trust boundary. This phase must leave it exactly where it is | Filesystem anchor path |
| Filesystem state (`.codegraph/` presence) → the advertised tool catalog | New this phase. A process outside the server's control now changes what the server advertises mid-session | Tool catalog contents |
| Wire-oracle capture → frozen `.golden` file in the repository | Captured subprocess output becomes a committed artifact; host-specific bytes reaching a golden would leak the capturing machine's paths into the repo | Raw wire frames, host paths |
| Wire-oracle capture → committed `.golden` (error payloads) | Error `data` payloads echo the requested version back, so whatever the scenario sends lands in the repo | Error `data` payloads |
| Wire-oracle `Capture` → a spawned `codegraph init` subprocess | The harness now runs a second subprocess mid-capture against a temp directory it owns | Subprocess argv |
| Pull-request diff → `tools/transcriptfreeze` classifier | The changed-file list and go.mod diff are computed by CI from a contributor-controlled branch; a contributor controls the file paths and diff content the classifier reads | Changed-file list, go.mod diff |
| Classifier verdict → CI job exit status | The boundary this phase deliberately moves. What crosses it determines whether a change can merge | Exit status |
| Guard's report → the human reviewer | The only remaining control under option-advisory. If the report stops being produced or stops naming both sides, the control silently disappears | Verdict reason text |
| The reviewed diff read → the decision to re-freeze | Under 03-02's outcome this is the phase's **primary** anti-regeneration control, not its secondary one | Transcript diff |

---

## Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation | Status |
|-----------|----------|-----------|----------|-------------|------------|--------|
| T-03-01 | Information Disclosure | `modern-discover-explore.golden` | low | mitigate | `oracle_test.go:122` `NormalizeWithLedger(tr.Stdout, Substitutions{RepoDir: tr.RepoDir})`; `TestEveryDeclaredFiringRuleActuallyFires` (`oracle_test.go:467`); field content pinned by `assertDiscoverCacheControl` (`anchors.go:238`), not by accepting the capture. Re-run: `rg '/Users/\|/home/\|/tmp/\|/var/folders' testdata/wireoracle/transcripts/` → 0 hits | closed |
| T-03-02 | Tampering | `internal/mcp/server.go` middleware branch | low | mitigate | `server.go:672-674` — one field (`CacheScope`), one method, reads only `res`, never `req`. `TTLMs` appears only in comments (`server.go:650`, `:669`), never assigned anywhere in the package | closed |
| T-03-03 | Information Disclosure | `server/discover` without `initialize` | low | accept | See AR-03-01 | closed |
| T-03-04 | Spoofing | Hand-authored anchor literals | low | mitigate | `test/wireoracle/anchors.go:19-38`, `:227-230` — all expected values hand-authored consts. `go list -deps ./test/wireoracle \| rg 'modelcontextprotocol\|mark3labs'` → no match (run, not quoted) | closed |
| T-03-05 | Tampering | The anti-regeneration control itself | medium | mitigate | Advisory stance stated in both operator-facing places: `Taskfile.yml:404` ("ADVISORY since 03-02") and `.github/workflows/ci.yml:415` step name. Full report retained — `tools/transcriptfreeze/main.go:87-95` prints the complete `verdict.Reason`. D-06 reviewed-diff pass is a blocking checkpoint (`03-05-PLAN.md:195`). See audit note on residual risk | closed |
| T-03-06 | Repudiation | `sdkSwapExemption` firing for the wrong reason | medium | mitigate | `03-02-SUMMARY.md:200-215` quotes the actual observed run — full exemption stderr plus `EXIT_CODE=0` — and states explicitly what the same command does post-merge. Misattribution recorded, not inherited | closed |
| T-03-07 | Elevation of Privilege | Runtime bypass in the guard's inputs | low | mitigate | `tools/transcriptfreeze/main.go:50-81` — `Classify` receives only the two file-derived values; the sole flags are `-changed-list` / `-gomod-diff`. No `os.Getenv`, marker file, or commit-message keyword reaches it. `classify.go:234` `v.Violation = true` intact | closed |
| T-03-08 | Tampering | `.github/workflows/ci.yml` edit | low | mitigate | `git show 855b8e2 -- .github/workflows/ci.yml` — diff is exactly one line, the step `name:`. `TRANSCRIPT_FREEZE_BASE` env indirection and `run: task check:transcript-freeze` byte-unchanged; no `${{` in the run body | closed |
| T-03-09 | Denial of Service | Malformed `_meta` reaching codegraph-go | low | accept | See AR-03-02 | closed |
| T-03-10 | Spoofing | A test asserting the wrong error path | medium | mitigate | `test/wireoracle/scenarios.go:281-293` documents the `"2099-01-01"` choice and its lexical-ordering reason in-file; golden has **0** occurrences of `-32601` and **1** of `2099-01-01`; red proof quoted at `03-03-SUMMARY.md:153` | closed |
| T-03-11 | Information Disclosure | The `-32022` error `data` payload | low | accept | See AR-03-03 | closed |
| T-03-12 | Tampering | VRFY-02 bypass via SDK constant import | medium | mitigate | Tree-wide grep: `CodeUnsupportedProtocolVersion` / `MetaKeyProtocolVersion` occur only in `internal/mcp/archtest/` prose and the guard's own in-memory-overlay self-test — zero real references. Guard is AST-based (`packages.Load` + `go/ast`, `protocol_version_test.go:166-264`), so wording cannot evade it. `go test ./internal/mcp/archtest/...` ok | closed |
| T-03-13 | Elevation of Privilege | Per-request index re-check | **high** | mitigate | `server.go:536-559` — `recheckCatalog` takes **no parameters** and resolves `query.ResolveCodegraphDir(startPath)` from the `BuildServer` closure. Sole call site `server.go:583` passes nothing; `req` never crosses into it. No reassignment or shadowing of `startPath`/`repoPath` in `server.go`. Both `registerTools` call sites (`:489`, `:544`) use construction-time values | closed |
| T-03-14 | Tampering | `confineToRepoRoot` anchor on mid-session index | medium | mitigate | `server.go:540` discards `ResolveCodegraphDir`'s return (`_, err :=`) — the anchor is never widened; `:544` re-passes the closure `repoPath`. Deliberate omission recorded at `:528-535`. Enforced by `TestConfinementAnchoredOnRepoRootNotStartPath` (`server_test.go:243`) | closed |
| T-03-15 | Denial of Service | `ResolveCodegraphDir` per request | low | mitigate | Gated to four methods (`server.go:581-584`). Reuses the pre-existing bounded upward walk `internal/query/resolve.go:31-48` (terminates at `parent == dir`), the same function `internal/cli/serve.go:44` already calls at startup. No new mechanism | closed |
| T-03-16 | Denial of Service | Tool-registry mutation under concurrency | medium | accept | See AR-03-04 | closed |
| T-03-17 | Repudiation | `tools=N` becoming point-in-time | low | mitigate | Semantics pinned at `server.go:619-633` (prefix and key names unchanged); `TestSessionLineReflectsPostAppearanceToolCount` (`server_test.go:389`) | closed |
| T-03-18 | Tampering | `Capture` running a second subprocess | low | accept | See AR-03-05 | closed |
| T-03-19 | Information Disclosure | The `instructions` string | medium | mitigate | `server.go:56` — `const instructions = "..."`, a Go compile-time literal, so interpolation is structurally impossible. Independently re-run: `rg -o '"instructions":"[^"]*"' testdata/wireoracle/transcripts/*.golden \| sort -u \| wc -l` → **1** unique string across the **24** goldens carrying it. Zero host paths | closed |
| T-03-20 | Information Disclosure | Host paths in re-frozen transcripts | medium | mitigate | Re-run: `rg '/Users/\|/home/\|/tmp/\|/var/folders' testdata/wireoracle/transcripts/` → **0 hits** across all 28 goldens. `NormalizeWithLedger` substitution plus `TestEveryDeclaredFiringRuleActuallyFires` | closed |
| T-03-21 | Tampering | Wholesale regeneration laundering a regression | **high** | mitigate | Blocking checkpoint precedes any frozen-file write (`03-05-PLAN.md:195`; capture into scratch, `03-05-SUMMARY.md:179`). Re-freeze commit `7c5d074` message carries 3 named causes with per-cause transcript counts, 3 byte-identical confirmations, and 5 must-not-have-changed properties with observed values. Independently verified: `git diff --name-status 7c5d074~1 7c5d074` = exactly **24** files, all under `testdata/wireoracle/transcripts/`, matching the approved cause list's union. Mutation-1 red re-proof on the post-change binary (`03-05-SUMMARY.md:181`). `go test ./test/wireoracle/... -count=1` ok | closed |
| T-03-22 | Spoofing | Instructions asserting absent behavior | low | mitigate | Content point 4 ("Tools appear automatically once an index exists, with no client restart required") is true on the wire: `index-appears-mid-session.golden` shows id=2 `"tools":[]` then id=3 populated, backed by `server.go:536-559` | closed |
| T-03-23 | Repudiation | An unpredicted cause silently attributed | **high** | mitigate | The control **fired in practice**: `03-05-SUMMARY.md:209` records the 22-vs-23 count discrepancy escalated at the Task 2 checkpoint under the "anything else" instruction, with observed values rather than absorbed; the maintainer directed recording it as a note, and commit `7c5d074` carries it verbatim as "Count note (not a cause)" | closed |
| T-03-SC | Tampering | npm/pip/cargo/go dependency installs | low | accept | See AR-03-06 | closed |

*Status: open · closed · open — below high threshold (non-blocking)*
*Severity: critical > high > medium > low — only open threats at or above workflow.security_block_on count toward threats_open*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| AR-03-01 | T-03-03 | `server/discover` answers before `initialize`, which is spec-correct for a discovery method. Rationale verified against actual bytes rather than accepted as prose: `modern-discover-explore.golden` id=1 carries only `resultType`, `serverInfo{codegraph,0.1.0}`, `ttlMs`, `cacheScope`, `supportedVersions`, `capabilities` and `instructions` — no host paths, no repository contents. `confineToRepoRoot` is untouched and still on every tool path via `openEngine` (`internal/mcp/tools.go:65-70`), so pre-initialize discovery reaches no filesystem surface. | seanb4t | 2026-08-06 |
| AR-03-02 | T-03-09 | Malformed `_meta` is validated by go-sdk's `validateRequestMeta` before any codegraph-go code runs, so defensive re-validation in `internal/mcp` would be duplicated logic that could drift from the SDK's and mask a real upstream change. Deliberately not added, and verified genuinely absent: `rg '_meta\|GetMeta\|Meta\[' internal/mcp/*.go` excluding tests → **zero hits**. | seanb4t | 2026-08-06 |
| AR-03-03 | T-03-11 | The `-32022` error `data` payload echoes the client's own rejected input back to it. Verified against actual golden bytes to contain exactly `{"supported":[5 versions],"requested":"2099-01-01"}` and nothing else — the supported-version list is public protocol information and the requested version is the caller's own string. | seanb4t | 2026-08-06 |
| AR-03-04 | T-03-16 | Tool-registry mutation under concurrency relies on go-sdk's own `s.mu` rather than a second codegraph-owned mutex, which would risk lock-order inversion against the SDK's. Exercised rather than assumed: `go test ./internal/mcp/... -count=1 -race` → ok (5.1 s) with `TestIndexAppearingMidSessionRegistersTools`, `TestIndexDisappearingMidSessionUnregistersTools` and `TestRepeatedListsDoNotDuplicateTools` present. | seanb4t | 2026-08-06 |
| AR-03-05 | T-03-18 | `Capture` spawns a second subprocess (`codegraph init`) mid-capture. All argv is harness-authored (`capture.go:375`), `workDir` is the `t.TempDir()` fixture copy (`oracle_test.go:62`, `:504`; `capture.go:236`), and the spawn is bounded by the pre-existing 30 s `runCtx` (`capture.go:240`). No captured byte reaches any `exec.Command`. | seanb4t | 2026-08-06 |
| AR-03-06 | T-03-SC | No package-manager install occurs in this phase. Asserted, not assumed by absence: `git diff --stat d91ebb6 HEAD -- go.mod go.sum` → empty. No dependency moved during (or since) the phase. The dependency closure was finalized in Phase 02 and re-audited there. | seanb4t | 2026-08-06 |

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-08-06 | 24 | 24 | 0 | Claude Opus 5 (`gsd-security-auditor` subagent, `/gsd-secure-phase 03`, ASVS L1) |

### Audit 2026-08-06 — notes

Audited retroactively, after the phase was verified and the milestone archived, to
close the coverage gap left by phases 02–05 shipping without a `SECURITY.md`. The
`gsd-security-auditor` subagent was spawned rather than taking the workflow's L1
short-circuit. Several threats were verified at L2/L3 depth where the boundary was
load-bearing (T-03-13, T-03-19, T-03-21, T-03-23) — greps and test runs were
re-executed by the auditor rather than quoted from the summaries.

**T-03-05's residual risk is real and correctly priced.** The D-03 anti-regeneration
guard now exits 0 on a detected collision (`tools/transcriptfreeze/main.go:87-95`) —
it was deliberately made advisory in 03-02 because its two-PR remedy is structurally
impossible and leaving it blocking would have passed silently for the wrong reason
under Phase 02's waiver. The compensating control is therefore entirely the human
reviewed-diff checkpoint. That checkpoint **did** run and **did** catch something
(T-03-23's count discrepancy). This is a deliberate, maintainer-approved posture
change, not a gap — but it is worth carrying forward as a standing note for any later
phase that regenerates the corpus.

**A deviation checked rather than accepted.** 03-03's executor reworded `anchors.go`'s
doc comment to avoid spelling `CodeUnsupportedProtocolVersion`, so an acceptance grep
would pass against its own file. This could have been guard-evasion theater; it is not.
The VRFY-02 guard resolves through `packages.Load` and walks `go/ast`
(`internal/mcp/archtest/protocol_version_test.go:166-264`), so prose wording is
invisible to it and only a real reference trips it. The claim at `03-03-SUMMARY.md:148`
holds.

**Unregistered surface.** No summary emits a `## Threat Flags` section, matching the
Phase 01 precedent. Every `## Deviations from Plan` section was read as a cross-check:
03-01 and 03-04 report none; 03-02's is a package doc-comment rewrite; 03-03's is the
doc-comment rewording above; 03-05's is the count correction. None introduces new
attack surface.

**Scope note (L1).** ASVS L1 verifies that each declared mitigation is *present*. It does
not perform L2 boundary-placement analysis or L3 end-to-end trace verification as a
matter of course, though the four high-severity threats above were verified at that depth.

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-08-06
