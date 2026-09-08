---
phase: 01
slug: engine-seam-wire-protocol-secure-transport
status: verified
# threats_open = count of OPEN threats at or above workflow.security_block_on (high)
threats_open: 0
asvs_level: 1
created: 2026-09-07
---

# Phase 1 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

**Register origin:** authored at plan time. All eleven PLAN.md files carry a parseable
`<threat_model>` block. This audit runs **retroactively**, during the milestone audit for
v0.12.0 — Phase 1 is the only phase in this milestone (of six) that shipped without a
`/gsd-secure-phase` run, despite `security_enforcement: true` being active throughout. The
verification below is against the **current working tree at HEAD (`5679dea8`)**, five phases
after Phase 1 shipped — not against what Phase 1's own commits looked like at the time — per
the explicit instruction that a mitigation present in Phase 1 and since removed counts as OPEN,
not closed.

**Why this phase carries the milestone's highest-severity threats:** Phase 1 establishes the
loopback bind and the exact-match Origin/Host guard (SRV-02) that every later phase's HTTP
surface (the SPA shell, live-push streaming, permalink generation) inherits by construction —
it is the single security boundary the rest of the milestone is built on top of.

---

## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| any web page the developer visits → `127.0.0.1:<port>` | A browser on the developer's machine can be induced by an attacker-controlled page to issue requests at the loopback listener via a rebound `Host` header (CVE-2024-28224 / CVE-2025-66414/66416 class). Loopback binding is NOT itself a security boundary. |
| CLI/UI process → `.codegraph/store` | The RPC handler opens the Pebble store per call; `codegraph daemon`, `serve --mcp` and `codegraph ui` contend for the same directory lock. |
| browser → `codegraph.ui.v1.UIService` RPC surface | The method set itself is the attack surface — every RPC argument crosses from an untrusted caller into the query engine. |
| RPC argument → `internal/query` | Symbol names, file paths, depths and limits from an untrusted caller reach engine methods whose own clamps must be the only bound. |
| caller-supplied file path → `readSourceFile`/`resolveSourcePath` | Every verbatim-source read (Node file mode, multi-def, Explore, later GetPermalink) shares one repo-root confinement gate. |
| repository source file → RPC response body | Arbitrary file content — including pathological minified files and non-UTF-8 bytes — is read from disk and written into a browser-bound response. |
| Engine error → Connect/RPC error code | What a browser is told about a failure is decided by the error's class; a message-text match would change meaning whenever a message changes. |
| MCP client → `codegraph serve --mcp` stdin/stdout | Newline-delimited JSON-RPC crosses a process boundary; server-owed responses and server-initiated notifications share one writer. |
| repository working tree → `git rev-parse` subprocess | Indexing shells out to `git` against a caller-supplied repository path. |
| stored `Meta` → wire layer | `(*Engine).IndexMeta` carries on-disk metadata (including `commit_sha`) out of `internal/query` onto the wire. |
| `.proto` source → committed generated Go | Generated code compiled into the shipped binary; a stale or hand-edited generated file is executable code no longer matching its declared contract. |
| drift guard → developer's working tree | A guard that regenerates in place could destroy uncommitted work while reporting on it. |
| fetched corpus tree / frozen golden file → test oracle | Locked third-party corpora and committed goldens are what the byte-identity regression oracle trusts; a silently-edited golden would make a real regression read green. |
| concurrent `codegraph` processes → one Pebble directory lock | Lock contention must degrade a response, never block or crash the process. |

---

## Threat Register

40 distinct threats across 11 plans (52 table rows total — 12 IDs are restated across sibling
plans because one mitigation covers a second call site a later plan in the phase adds; no
restated ID carries conflicting severities, independently re-verified). All 40 closed: 34
mitigated with controls verified live against HEAD `5679dea8`, 6 accepted and logged below.

### Critical

| Threat ID | Category | Component | Disposition | Evidence (verified at HEAD) | Status |
|-----------|----------|-----------|-------------|------------------------------|--------|
| T-01-01 | Spoofing | `internal/uiserver/originguard.go` — `Host` header | mitigate | Exact map-key equality, three loopback spellings only, no substring/prefix/case-insensitive matching (`originguard.go:47-50,74-80`). `TestOriginHostGuard` (20 subtests: admits, rejects `evil.com`/`127.0.0.1.evil.com`/wrong-port, race-safe concurrency) + `TestOriginHostGuardAdmitsOriginlessGET` — both run fresh, PASS. | closed |

### High

| Threat ID | Category | Component | Disposition | Evidence (verified at HEAD) | Status |
|-----------|----------|-----------|-------------|------------------------------|--------|
| T-01-02 | Spoofing | `originguard.go` — `Origin` header | mitigate | Membership AND `origin == "http://"+r.Host` pairing (`originguard.go:52-63`); 4+ cross-spelling rejection rows in the same test suite above, all PASS. | closed |
| T-01-03 | Elevation of Privilege | `uiserver.Options.Addr` | mitigate | `DefaultAddr = "127.0.0.1:0"` (`server.go:25`); `Options{RepoPath, Addr}` only; `ui` command flags exactly `{path, no-open}`. `TestUICommandFlagSetIsExactlyPathAndNoOpen` + `TestUIServiceHoldsNoStoreTypedField` — PASS. | closed |
| T-01-04 | Tampering | `codegraph.ui.v1.UIService` method set | mitigate | Service has grown from 9 methods (Phase 1) to **14** (Phases 1–6) — set-equality-both-directions fixture updated at every wave, plus a negative mutating-verb scan. `TestUIServiceMethodSetIsExactlyTheReadSet`, `TestUIServiceDeclaresNoMutatingMethod`, `TestUIProtoFieldNumbersAreStableAndUnique` — all PASS fresh. The guard is a living invariant, not a one-time check. | closed |
| T-01-05 | Denial of Service | Verbatim source response body | mitigate | Two-layer cap (`sourceLineCap`=4096, `sourceByteCap`=262144, rune-safe cut) plus `connect.WithSendMaxBytes`(16MiB)/`WithReadMaxBytes`(1MiB) transport backstop strictly above it (`truncate.go`). `TestTransportBackstopSitsAboveApplicationCap`/`EveryAggregate`, `TestTruncateSourceExceedsLineCap`/`ExceedsByteCap`/`NeverSplitsARune` — PASS. | closed |
| T-01-07 | Information Disclosure | `internal/query.readSourceFile`/`resolveSourcePath` | mitigate | Two-stage confinement: string-level `Clean`/`Rel` rejection **plus `filepath.EvalSymlinks` re-verification** (`node.go:17-79`) — the WR-03 symlink gap the plan flagged as a known limitation is actually **closed in code**, not merely documented. Confirmed the single `os.ReadFile` call site across `internal/query`, `internal/uiserver`, `internal/mcp`. `TestResolveSourcePathRejectsSymlinkEscape` — PASS. Covers all restatements (01-04 Node/multi-def, 01-05 Explore, 01-09 GetNodeDetail/Explore RPC, 01-10 wire-layer single-def read). | closed |
| T-01-08 | Tampering | `internal/schema/graph.pb.go`, `internal/uiproto/uiv1/*.pb.go`, `uiv1connect/*.connect.go` | mitigate | Ran `task proto:drift` live: regenerates both surfaces into a temp tree with the pinned toolchain, compares 4 files, fails on zero-count or any byte diff including header-only. Fresh run: "all 4 generated files byte-identical" — PASS. | closed |
| T-01-09 | Repudiation | `internal/mcp/server.go` `pendingWriter` | mitigate | Decrement only on complete lines within `b[:n]`; underflow refused and counted (`server.go:433-451`). `go test -race`: `TestPendingWriter*` (7 tests incl. `NeverGoesNegative`, `DecrementsOnlyResponses`) — PASS. | closed |
| T-01-12 | Repudiation | The golden byte-identity oracle itself | mitigate | Ran live (not cached, indexes 4 real corpora): `TestGoldensMatchLiveEngineOutput` — 26/26 matched — and `TestGoldensMatchLiveEngineOutputIsNonVacuous` — both planted mutations (`appended-byte`, `flipped-interior-byte`) caught. PASS. | closed |
| T-01-16 | Tampering | The extracted `NodeDetail`/`ExploreResult` builders | mitigate | Same oracle as T-01-12 re-run post-extraction (01-04, 01-05) and still 26/26 at HEAD after 5 more phases of change. PASS. | closed |
| T-01-24 | Denial of Service | Store handle retained across calls | mitigate | Single `openEngine`/`withEngine` seam (`handlers.go:26,51-66`); traced all 14 RPC handlers — 12 via `withEngine`, `GetStatus` via a documented direct-open-with-deferred-close for its degrade path, `WatchGraph` holds no store handle at all (subscribes to the publisher instead). `go test -race`: `TestUIServiceOpensThroughTheSingleSeam`, `TestDegradedRPCOpensTheStoreExactlyOnce`, `TestAHolderAcquiresTheStoreAfterConcurrentRPCsComplete` — PASS. | closed |
| T-01-27 | Repudiation | Short write miscounted as delivered | mitigate | Classifier's sole input is `b[:n]`, never `b`. Covered by the same `TestPendingWriter*` -race run above. | closed |
| T-01-39 | Repudiation | Degrade tests that cannot fail | mitigate | Read `degrade_test.go` for the forbidden "succeeds-or-degrades" shape — absent; each test names one condition-discriminated outcome. `TestDegradedRPCOpensTheStoreExactlyOnce`, `TestDegradeIsIdempotent` — PASS. | closed |
| T-01-SC | Tampering | Go module installs (`connectrpc.com/connect`, `buf`, `github.com/pkg/browser`) | mitigate | Pinned exact versions with real `go.sum` hashes present at HEAD (`connectrpc.com/connect v1.20.0`, `github.com/pkg/browser v0.0.0-20240102092130-...`) — see Audit Note 3 re: the benign `// indirect` go.mod marker. | closed |

### Medium

| Threat ID | Category | Component | Disposition | Evidence (verified at HEAD) | Status |
|-----------|----------|-----------|-------------|------------------------------|--------|
| T-01-06 | Information Disclosure | `IndexingInProgress.message` | mitigate | Fixed generic constant via `connect.NewErrorDetail(&uiv1.IndexingInProgress{Message: indexingInProgressMessage})` (`degrade.go:131`) — no path/lock-file/pid embedded. | closed |
| T-01-10 | Denial of Service | Store lock contention | **accept** | See Accepted Risks Log AR-04. | closed |
| T-01-11 | Tampering | `testdata/golden/corpus/**/*.json` | mitigate | Byte-exact comparison, no normalisation; covered by the same live 26/26 oracle run (T-01-12). | closed |
| T-01-14 | Denial of Service | `waitForDrain` under an over-counting sniff | **accept** | See Accepted Risks Log AR-02. | closed |
| T-01-17 | Denial of Service | Reverse adjacency built once per call | mitigate | `TestMultiDefReverseAdjacencyBuiltOnce` — PASS. | closed |
| T-01-18 | Tampering | `internal/query`'s dependency direction | mitigate | **No persisted regression test found** — the plan's mitigation was a one-shot `go list -deps` check, never committed as a test (see Audit Note 1). Re-ran it live: `go list -deps ./internal/query` contains zero `connectrpc.com/connect` and zero `internal/uiproto` entries; positive control confirms `google.golang.org/protobuf` (legitimate, via `internal/schema`) does appear. Invariant holds today; unguarded against regression. | closed (empirically true, not regression-proof) |
| T-01-19 | Elevation of Privilege | `git rev-parse` subprocess | mitigate | Fixed arg vector via `exec.CommandContext`, `-C` flag not interpolation, bounded timeout, output accepted only at exactly 40/64 lowercase-hex chars (`internal/indexer/commit.go:47-66`). | closed |
| T-01-21 | Denial of Service | Indexing blocked by unavailable git | mitigate | `resolveHeadCommitSHA` returns `""` on every failure path, never an error (`commit.go:48-50,57-58`). | closed |
| T-01-22 | Denial of Service | Unbounded depth/limit from caller | mitigate | `validateDepth`/`clampDepth`/`clampAffectedDepth`/`validateLimit`/`MaxLimit=1000` all present (`internal/query/validate.go:26,71,86,107,137`). | closed |
| T-01-23 | Information Disclosure | Error text reaching the browser | mitigate | `mapEngineError` classifies by `errors.Is`/`errors.As` against exported sentinels only, never message text (`handlers.go:81-100`). | closed |
| T-01-25 | Tampering | Error classification by sentinel, not message | mitigate | `internal/cli/serve.go` uses `errors.Is(err, graphstore.ErrStoreLocked)`/`query.ErrNotInitialized` exclusively. | closed |
| T-01-26 | Tampering | `internal/goldenspec.LockedCorpusArgs` | mitigate | Shared table present, single copy (`internal/goldenspec/spec.go:45`); `TestLockedCorpusArgsAreTheFrozenValues` exists. | closed |
| T-01-28 | Information Disclosure | Eager gather surfacing unreachable read error | mitigate | Lazy per-candidate gather confirmed; `detail_test.go:328,333` assert an out-of-cap unreadable candidate never surfaces an error. | closed |
| T-01-29 | Repudiation | Classification table coverage | mitigate | `TestEveryReachableErrorIsClassified` logs exercised-vs-recorded count and fails below the floor (`internal/query/errors_test.go:118-182`). | closed |
| T-01-31 | Tampering | `codegraph status` output drifting silently | mitigate | `TestStatusResultFieldSetIsUnchanged` (reflected field-set equality) present (`internal/query/meta_test.go:127`). | closed |
| T-01-32 | Denial of Service | Drift guard overwriting uncommitted edits | mitigate | `proto:drift` targets a temp root only, removed on exit; live run confirmed source tree untouched. | closed |
| T-01-33 | Tampering | Toolchain downgrade presenting as no drift | mitigate | Generated header compared, not normalised away — confirmed by the same live `proto:drift` run (byte-identical including header). | closed |
| T-01-35 | Denial of Service | Unbounded per-candidate source reads | mitigate | `uiMultiDefCap = 20` bounds gathered candidates (`handlers.go:143-154,728`). | closed |
| T-01-36 | Tampering | Field-number collision/renumber across plans | mitigate | `TestUIProtoFieldNumbersAreStableAndUnique` — extended through Phase 6 (97→166 fields across 6 chained-extension constants), both-direction resolution, PASS at HEAD. | closed |
| T-01-37 | Tampering | Rune-boundary cut duplicated across packages | mitigate | Single `textutil.TruncateOnRuneBoundary`, consumed by both `internal/mcp/session_line.go:69` and `internal/uiserver/truncate.go:162` — no second copy. | closed |
| T-01-38 | Denial of Service | Non-UTF-8 source file marshal failure | mitigate | `SourceBlob.content` is `bytes`, not `string` (`ui.proto:428`) — scoped UTF-8 guarantee documented. | closed |

### Low

| Threat ID | Category | Component | Disposition | Evidence (verified at HEAD) | Status |
|-----------|----------|-----------|-------------|------------------------------|--------|
| T-01-13 | Denial of Service | Corpus indexing in the test suite | **accept** | See Accepted Risks Log AR-01. | closed |
| T-01-15 | Tampering | Malformed inbound line driving the MCP sniff | **accept** | See Accepted Risks Log AR-03. | closed |
| T-01-20 | Information Disclosure | `Meta.commit_sha` reaching a client | **accept** | See Accepted Risks Log AR-05. | closed |
| T-01-30 | Information Disclosure | `(*Engine).IndexMeta` widening exported surface | mitigate | Returns only the stored `*schema.Meta`, no new filesystem access or writer (`internal/query/engine.go:158-167`). | closed |
| T-01-34 | Information Disclosure | `GetStatusResponse.commit_sha` | **accept** | See Accepted Risks Log AR-06. | closed |

*Severity: critical > high > medium > low — only OPEN threats at or above `workflow.security_block_on` (high) count toward `threats_open`.*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (none in this phase).*

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| AR-01 | T-01-13 (low) | Indexing four real third-party repositories in the golden test suite is slow by design and already the job's cost profile; adds one more index-and-query pass per corpus. Test-suite cost, not a production security exposure. | Plan author, 01-02 | 2026-08-23 |
| AR-02 | T-01-14 (medium) | The conservative default (an unparseable outbound MCP line does not decrement the pending counter) can only lengthen `waitForDrain`, strictly bounded by `stdinLingerGrace` = 5s (verified present at `internal/mcp/server.go:249`). A bounded wait is preferable to a premature drain that could truncate a legitimate response. | Plan author, 01-03 | 2026-08-23 |
| AR-03 | T-01-15 (low) | `looksLikeJSONRPCCall` (verified at `internal/mcp/server.go:366`) already reports true conservatively on a parse failure, and the sniff reads only `method`/`id`. No new parsing surface introduced. | Plan author, 01-03 | 2026-08-23 |
| AR-04 | T-01-10 (medium) | No second retry layer above `graphstore.Open`'s bounded budget (D-15) — verified: no additional retry wrapper found around any `graphstore.Open` call site at HEAD. Adding one would violate the exact condition SRV-04 (the degrade design) names. Past-budget contention degrades to `CodeUnavailable` rather than blocking. | Plan author, 01-11 | 2026-08-23 |
| AR-05 | T-01-20 (low) | A commit SHA of the repository the developer is already indexing on their own machine, served only over the loopback listener the SRV-02 guard (T-01-01/02, verified above) protects. No new exposure beyond what the developer already has on disk. | Plan author, 01-06 | 2026-08-23 |
| AR-06 | T-01-34 (low) | Same boundary as AR-05, viewed from `GetStatusResponse` rather than `Meta`. Loopback + Origin/Host guard covers it identically. | Plan author, 01-08 | 2026-08-23 |

*Accepted risks do not resurface in future audit runs.*

---

## Audit Notes

**1. T-01-18's mitigation was never committed as a regression test.** The plan's stated control
(`go list -deps ./internal/query` containing no `connectrpc.com/connect`/`internal/uiproto`
entry) was written into 01-04-PLAN.md and 01-05-PLAN.md's `<verification>` blocks as a one-shot
acceptance check, not into any `_test.go` file — confirmed absent by `rg` across the whole
tree (checked with a positive control: the same query-style dependency-boundary test *does*
exist for other packages, e.g. `internal/graphstore/archtest/import_graph_test.go`, proving the
search technique works and the absence here is real, not a search miss). This audit re-ran the
check live rather than accepting the gap: it is empirically true at HEAD (`internal/query`
pulls in neither `connectrpc.com/connect` nor `internal/uiproto`, with `google.golang.org/protobuf`
confirmed present as the positive control), but the invariant has no regression guard. Below
`block_on` (medium), so non-blocking — flagged for a future plan to add the missing archtest.

**2. Restated threat IDs, not double-counting.** 52 threat-register rows across 11 plans
collapse to 40 distinct IDs. Twelve IDs are restated verbatim in a sibling plan because the
same mitigation covers a second call site that plan adds (e.g. T-01-07's confinement gate is
cited by 01-04, 01-05, 01-09 and 01-10 as each adds a new read path through it; T-01-04's
read-only assertion is cited by 01-01, 01-08 and 01-09 as the method count grows). Verified by
script that no restated ID carries a conflicting severity across its occurrences — none found.

**3. `go.mod`'s `// indirect` marker on `connectrpc.com/connect` and `github.com/pkg/browser`
is a stale-tidy artifact, not a supply-chain gap.** Both packages are imported directly
(`server.go`, `internal/cli`) but `go mod tidy` is documented as broken on this branch for an
unrelated pre-existing reason, per this audit's own operating instructions, so the `// indirect`
comment has not been refreshed. `go.sum` carries real cryptographic hashes for both at their
pinned versions regardless of the comment's accuracy — the property T-01-SC actually protects
(a legitimate, pinned, hash-verified module) holds.

**4. The WR-03 symlink-escape gap named in 01-04/01-05's threat model is closed, not merely
flagged.** `resolveSourcePath` performs `filepath.EvalSymlinks` on both the repo root and the
candidate path and re-verifies confinement against the resolved forms — this is real hardening
beyond what the phase's own plans described as a residual limitation, confirmed by a passing
`TestResolveSourcePathRejectsSymlinkEscape`.

**5. New attack surface added in later phases is independently accounted for.** Phases 2–6 all
carry their own `status: verified`, `threats_open: 0` SECURITY.md files covering the SPA shell,
browse/permalink, health/workbench, file-graph, and live-push additions built on top of Phase
1's RPC surface. This audit did not need to re-litigate those; it confirmed Phase 1's own
boundaries (the Origin/Host guard, the single store-open seam, the source-read confinement gate)
still wrap every one of the 14 current RPC methods and the SPA route, which they do
(`server.go` registers exactly two handlers on the guarded mux).

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-09-07 | 40 | 40 | 0 | `/gsd-secure-phase` (retroactive — run during the v0.12.0 milestone audit, five phases after Phase 1 shipped; register authored at plan time, verified against HEAD `5679dea8`, not against Phase 1's original commits) |

**Method.** Register built from the `<threat_model>` blocks in all eleven PLAN.md files. Every
`mitigate` disposition at or above `high` (14 threats) was verified with a fresh, live test run
or direct source read against the current working tree — not against SUMMARY/PLAN claims — per
the explicit instruction that a mitigation removed after Phase 1 shipped counts as OPEN. The
remaining 20 medium/low `mitigate` threats were verified at ASVS L1 grep/read depth. All 6
`accept` threats were re-validated against current code (bounded timeouts, sentinel checks,
loopback-guard coverage) rather than accepted on the plan's word alone. `task test:unit` and
`go vet` both run clean at HEAD. `threats_open: 0` — nothing at or above `block_on: high` is
open, so this phase does not block ship.

---

## Sign-Off

- [x] All 40 distinct threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log (6 entries, all below `high`)
- [x] `threats_open: 0` confirmed by fresh test runs and source verification, not by trusting PLAN/SUMMARY claims
- [x] `status: verified` set in frontmatter
- [x] Retroactive-audit context recorded (Audit Notes, Security Audit Trail)

**Approval:** verified 2026-09-07 (retroactive)
