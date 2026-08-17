---
phase: 01
slug: corpus-selection-by-measurement
status: verified
# threats_open = count of OPEN threats at or above workflow.security_block_on (high)
threats_open: 0
asvs_level: 1
register_authored_at_plan_time: true
threats_total: 35
threats_closed: 35
threats_mitigated: 23
threats_accepted: 12
created: 2026-08-16
verified: 2026-08-16
audit_mode: retroactive-L1-shortcircuit
---

# Phase 01 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

**Audit mode:** retroactive. This phase executed before `/gsd-secure-phase` was run against it, but
every one of its 7 PLAN files carried a `<threat_model>` block, so
`register_authored_at_plan_time: true` and the register below is the phase's own — not one
reconstructed after the fact. With `asvs_level: 1` and `threats_open: 0`, the workflow's L1
short-circuit applies and no `gsd-security-auditor` subagent was required; mitigations were verified
at grep depth against the working tree at commit `69159a3`.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| Pebble store → `Engine.Status` | Index content produced by this project's indexer over third-party source; `Edge.Kind` is an arbitrary store string that becomes a JSON object key | Edge-kind identifiers, counts |
| `StatusResult` → `--json` / MCP markdown | Read-only projection crossing to an agent-consumable surface | Index statistics |
| CLI flag surface → MCP surface | The dense `--all-kinds` form must not become reachable from the argument-less MCP tool (D-05) | Density flag state |
| Live capture output → committed `.golden` oracle | Bytes from a locally built binary become the frozen baseline every later CI run compares against | Transcript bytes |
| `corpora/manifest.json` → Taskfile `git` invocation | `repo` and `sha` fields are interpolated into shell commands | Repository identity, commit SHA |
| `github.com` → local corpus cache | Third-party source enters developer and CI environments | Arbitrary third-party source |
| Local corpus cache → the indexer | A cached tree is read as trusted input; mutable between fetch and use | Parsed source |
| Environment variables → fetch target | `CORPUS_REPO` / `CORPUS_SHA` are caller-supplied, not necessarily manifest-derived | Fetch parameters |
| Local machine state → committed measurement artifacts | The tool writes files that are committed and published | Paths, per-machine values |
| Generated document → curated document | Two committed files with different owners; a generator→curated write path would be silent data loss | Measurement record vs. policy |
| Namespace persistent cache volume → CI workspace | A **mutable** volume populatable by any PR on any branch supplies corpus trees | Cached corpus trees |
| `id-token: write` signing boundary | **NOT crossed** by this phase's workflow — stated explicitly because this repository has twice acted on cache trust at exactly this line | (none) |

---

## Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation | Status |
|-----------|----------|-----------|----------|-------------|------------|--------|
| T-01-01-01 | Information Disclosure | `StatusResult` JSON projection | low | mitigate | `ProjectPath`/`IndexPath` stay blank; new keys carry counts only | closed |
| T-01-01-02 | Denial of Service | `Engine.Status` full edge scan | low | accept | Local embedded store, no untrusted caller — see Accepted Risks | closed |
| T-01-01-SC | Tampering | dependency surface | low | accept | Zero `go.mod` additions; Package Legitimacy Gate N/A | closed |
| T-01-02-01 | Tampering | `present.RenderStatus` terminal output | low | mitigate | Breakdown keys are extractor-produced identifiers from a closed vocabulary, not free-form file content | closed |
| T-01-02-02 | Elevation of Privilege | MCP density leak | medium | mitigate | Density applied at exactly one point in `internal/cli/status.go`, downstream of `Engine.Status`, upstream of every render surface | closed |
| T-01-02-SC | Tampering | dependency surface | low | accept | Zero `go.mod` additions | closed |
| T-01-03-01 | Tampering | `call-status.golden` re-freeze | **high** | mitigate | Re-freeze reviewed as a single-cause diff; VERIFICATION 5/5 truths | closed |
| T-01-03-02 | Tampering | capture-to-golden redirect | medium | mitigate | Temp-then-move; capture never truncates the oracle in place | closed |
| T-01-03-03 | Information Disclosure | frozen transcript content | low | accept | `normalize.go` substitutes repo dir and server version — see Accepted Risks | closed |
| T-01-03-SC | Tampering | dependency surface | low | accept | Zero `go.mod` additions; no new capture dependency | closed |
| T-01-04-01 | Tampering | manifest `repo`/`sha` interpolated into `git remote add` / `git fetch` | **high** | mitigate | Two independent allowlists. `internal/corpora.Validate` enforces exact 40-hex SHA; `Taskfile.yml` rejects a repo without exactly one slash and refuses any pair with no manifest entry (`no manifest entry found for repo=… sha=…`) | closed |
| T-01-04-02 | Tampering | corpus tree at the correct commit with altered contents | **high** | mitigate | Four-part integrity check on the staged tree; fails `exit 1` on both the already-present and post-fetch paths — *"refusing to silently re-fetch over a corrupt or tampered destination"* | closed |
| T-01-04-03 | Tampering | third-party source entering dev/CI | medium | mitigate | Pinned SHAs only, never a floating branch or tag | closed |
| T-01-04-04 | Denial of Service | interrupted or concurrent fetch | medium | mitigate | Atomic claim `mkdir "${dest}.lock"` taken **before** the network fetch (`Taskfile.yml:3472`), contention reported, `rmdir` cleanup at 3486 | closed |
| T-01-04-05 | Information Disclosure | corpus cache path | low | accept | Resolved root may contain `$HOME` — see Accepted Risks | closed |
| T-01-04-SC | Tampering | dependency surface | low | accept | Zero `go.mod` additions | closed |
| T-01-05-01 | Information Disclosure | committed measurement artifacts | medium | mitigate | `StripVolatile` removes both path fields, the worktree-mismatch object, the staleness flag and every per-machine key | closed |
| T-01-05-02 | Tampering | curated policy destroyed by regeneration | **high** | mitigate | The generator has **no write path** to `corpora/selection.json` — asserted in `tools/corpora/main.go:17` and **tested** at `tools/corpora/measure_test.go:105`; only `LoadSelection` (read) touches it at runtime | closed |
| T-01-05-03 | Denial of Service | indexing arbitrary third-party source | medium | accept | CGo tree-sitter external C scanners — see Accepted Risks | closed |
| T-01-05-04 | Tampering | observation determinism | **high** | mitigate | Deterministic ordering enforced by explicit sorts in `internal/corpora`; a record that reordered between runs would make the drift diff meaningless | closed |
| T-01-05-05 | Repudiation | silently omitted corpus | **high** | mitigate | `internal/corpora/coverage.go` *"fails with the path named when any is missing or malformed"*; 27 fail/refuse tokens, no silent skip | closed |
| T-01-05-SC | Tampering | dependency surface | low | accept | Zero `go.mod` additions | closed |
| T-01-06-01 | Repudiation | the coverage claim | **high** | mitigate | Every coverage entry derives from measured output; `CheckCoverage(m, obs, sel)` reconstructs the claim rather than validating a stated one | closed |
| T-01-06-02 | Tampering | synthetic coverage silently satisfying a criterion | **high** | mitigate | `coverage.go` step 8 *"require syntheticKinds to be empty"*; Plan 01-07's guard fails on a non-empty value | closed |
| T-01-06-03 | Tampering | curated selection diverging from the manifest | **high** | mitigate | Reconciliation asserted in both `coverage_test.go` and `manifest_test.go` | closed |
| T-01-06-04 | Tampering | licence drift between seeding and locking | medium | mitigate | Licences re-verified live against the GitHub licence API at lock time, not carried forward | closed |
| T-01-06-05 | Information Disclosure | committed measurement artifacts | medium | mitigate | Inherits T-01-05-01's `StripVolatile` | closed |
| T-01-06-06 | Denial of Service | indexing large third-party monorepos | low | accept | Bounded by the six-candidate search box — see Accepted Risks | closed |
| T-01-06-SC | Tampering | third-party source at pinned SHAs | medium | mitigate | Exact-commit pinning, integrity-verified on all four parts | closed |
| T-01-07-01 | Tampering | mutable cache volume poisoning | **high** | mitigate | Cache treated as untrusted; corpora re-verified after restore rather than trusted from cache | closed |
| T-01-07-02 | Tampering | manifest values reaching the fetch shell | **high** | mitigate | Inherits and re-asserts T-01-04-01: entries emitted only after `corpora.Validate` | closed |
| T-01-07-03 | Repudiation | vacuous CI pass | **high** | mitigate | Three derived positive assertions — `CheckedKinds: len(query.RankEdges)`, `CheckedCorpora: checkedCorpora`, and a `checkedCorpora == 0` guard (`coverage.go:216-224`). Satisfies rule `84d1gfpywd` | closed |
| T-01-07-04 | Tampering | coverage claim drifting from its evidence | **high** | mitigate | `CheckCoverage` **derives** the claim from manifest + observations + selection rather than validating a stated one | closed |
| T-01-07-05 | Denial of Service | cache storage exhaustion | low | accept | D-13's GitHub-cache premise explicitly retired — see Accepted Risks | closed |
| T-01-07-SC | Tampering | caching-action supply chain | low | accept | `namespacelabs/nscloud-cache-action` pinned by commit, already trusted in-repo — see Accepted Risks | closed |

*Status: open · closed · open — below high threshold (non-blocking)*
*Severity: critical > high > medium > low — only open threats at or above `workflow.security_block_on` (high) count toward `threats_open`*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

**Totals:** 35 threats — 23 mitigated, 12 accepted, **0 open**. High-severity: 13, all mitigated and verified.

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| R-01-01 | T-01-01-02 | `Engine.Status`'s full edge scan is O(edges) on a local embedded store with no untrusted caller — `status` is a local CLI/MCP read. Cost is bounded by the operator's own index. | plan author (01-01) | 2026-08-14 |
| R-01-02 | T-01-01-SC, T-01-02-SC, T-01-03-SC, T-01-04-SC, T-01-05-SC | Zero `go.mod` entries added and no package-manager install runs in any of these plans, so the npm/pip/cargo Package Legitimacy Gate does not apply. Recorded rather than omitted. | plan author | 2026-08-14 |
| R-01-03 | T-01-03-03 | `normalize.go`'s existing substitution rules already replace the repo directory and server version with placeholders before any transcript is frozen. | plan author (01-03) | 2026-08-14 |
| R-01-04 | T-01-04-05 | The resolved corpus root may contain the developer's home directory and is printed by `tools/corpora -mode root` for local diagnosis. Local-only output; not committed (`StripVolatile` removes path fields from artifacts). | plan author (01-04) | 2026-08-14 |
| R-01-05 | T-01-05-03 | The indexer parses untrusted third-party source through CGo tree-sitter grammars whose external C scanners can, worst case, fault the host process. This is the project's **documented, accepted** CGo tail-risk (see `.claude/CLAUDE.md` "The Parser Decision", Option A). Corpora are pinned, licence-vetted, and integrity-checked, which bounds — but does not eliminate — exposure. | project constraint (Option A) | 2026-08-14 |
| R-01-06 | T-01-06-06 | Whole-repository pinning (D-10) means some candidates are large; cost is bounded by the six-candidate search box. | plan author (01-06) | 2026-08-14 |
| R-01-07 | T-01-07-05 | D-13's "10 GB per-repo ceiling with LRU eviction" premise describes GitHub's cache service and is **explicitly retired**, not inherited, for the Namespace volume. | plan author (01-07) | 2026-08-14 |
| R-01-08 | T-01-07-SC | `namespacelabs/nscloud-cache-action` is pinned at a commit already trusted elsewhere in this repository; this phase adds **no new** supply-chain surface. | plan author (01-07) | 2026-08-14 |

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-08-16 | 35 | 35 | 0 | `/gsd-secure-phase 1` (orchestrator, L1 short-circuit — no auditor subagent required) |

**Verification method:** L1 grep-depth against the working tree at commit `69159a3`. Every
high-severity `mitigate` threat was confirmed by locating its named control in source, not by reading
the plan that proposed it. Two searches initially returned zero and were re-run after correcting the
instrument (the corpora **fetch** lives in `Taskfile.yml`, not `internal/corpora/`); both controls
were present. No "zero hits" result in this audit was accepted without confirming the pattern matched
where matches should exist.

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-08-16
