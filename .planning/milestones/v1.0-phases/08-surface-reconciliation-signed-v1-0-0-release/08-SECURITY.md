---
phase: 08
slug: surface-reconciliation-signed-v1-0-0-release
status: verified
# threats_open = count of OPEN threats at or above workflow.security_block_on severity (the blocking gate)
threats_open: 0
asvs_level: 1
block_on: high
register_authored_at_plan_time: true
threats_total: 22
threats_closed: 22
accepted_risks: 3
created: 2026-07-26
---

# Phase 08 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

Register authored at plan time across all 9 PLAN files (every one carried a
`<threat_model>` block), then verified against the implementation by
`gsd-security-auditor` at ASVS L1 with `block_on: high`. Audited diff range
`529d818..b55d543`.

**Scope note — the `v1.0.0` release has NOT been cut.** REL-02 is
maintainer-manual and the go-ahead was explicitly withheld (`08-UAT.md`, test 1:
*"I don't think we're ready to declare 1.0"*). T-08-09-02 and T-08-09-03 are
therefore closed on **specification correctness** — that the runbook's documented
cosign identity and verification procedure are right and match `verify.go` — not
on any observed verification of a `v1.0.0` artifact. See their residual notes.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| MCP client → `codegraph_impact` → shared engine | Untrusted client supplies `--depth`; clamp bounds traversal work | integer depth |
| CLI/MCP caller → `files --dir` | Untrusted filter string used as a path prefix over the frozen index (no filesystem walk) | path prefix string |
| user → `upgrade` CLI → `internal/upgrade.Run` → `verify.go` | `--force` alters the same-version guard but must not reach the signature-verification decision | release binary + signature |
| cobra command construction | Duplicate shorthand within a command panics at flag registration (availability) | — |
| caller → `Affected(files, depth)` | Untrusted depth bounds BFS; untrusted file list seeds traversal over the frozen index | path list, integer depth |
| git hook / CI stdin → `affected --stdin` | Untrusted newline-delimited path list ingested for scripting | path list |
| `affected --filter <glob>` | Untrusted glob pattern matched against affected paths | glob pattern |
| `affected --quiet` stdout → downstream shell | Machine-readable path list may be piped into another command | path list |
| flag surface ↔ parity doc | Drift between registered cobra flags and the documented parity claim REL-04 reads | — |
| third-party Charm deps → single static binary | New transitive packages could introduce CGo, memory-unsafe scanners, or known vulns | dependency closure |
| published benchmark numbers → users / release notes | Integrity of the performance claim | benchmark data |
| release edits → `internal/upgrade/verify.go` LOCKED contract | Any change to release.yml identity/trigger must move verify.go's 3 constants in lockstep | cosign identity constants |
| maintainer tag push → release.yml signed build | The tag ref binds the cosign keyless identity (per-binary signing + SLSA provenance) | signed artifacts + provenance |

---

## Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation (verified evidence) | Status |
|-----------|----------|-----------|----------|-------------|--------------------------------|--------|
| T-08-01-01 | Denial of Service | `clampDepth` / Impact BFS | medium | mitigate | `validate.go:22` `MaxDepth=50`, `:49` `defaultDepth=2`, `:71-79` `clampDepth`, `:137-142` `validateDepth` rejects n<0. Applied at **both** entry points: `traverse.go:429-432` reached by CLI `impact.go` and MCP `mcp/tools.go:327`. Tests `engine_test.go:174-181,235-242`. | closed |
| T-08-01-02 | Tampering | golden fixtures | low | mitigate | `git diff 529d818..HEAD -- testdata/` is **empty** — no fixture and no harness file changed (stronger than the "only impact.json may change" allowance). `go test ./testdata/golden/...` ok. | closed |
| T-08-02-01 | Information Disclosure | `Files()` `--dir` prefix | low | accept | `files.go:152` reads `e.reader.IterateFiles()` only — no `os.ReadDir`, no filesystem walk; `Dir` is a post-filter over already-indexed paths at `:166`. No path join, no traversal escape. → Accepted Risks AR-08-01 | closed |
| T-08-02-02 | Denial of Service | `--dir` matching | low | mitigate | `files.go:112-123` `dirPrefixMatches`: `TrimSuffix` + `==` + `HasPrefix` in a fixed 2-iteration loop. No regex, no glob, no backtracking. *(Implementation changed post-plan — see Drift 1.)* | closed |
| T-08-03-01 | Tampering | `upgrade --force` → `Run` | **high** | mitigate | `upgrade.go:112` — `opts.Force` gates *only* the same-version no-op return. Ordering intact: download `:127` → verify `:136` → fatal return `:140` → swap `:147`. `upgrade_test.go:220-259` `TestUpgradeRun_ForceStillVerifiesBeforeSwap` asserts `swapCalled == false` and target bytes unchanged under `Force: true` + failing verify. | closed |
| T-08-03-02 | Denial of Service (build) | cobra shorthand registration | low | mitigate | Real catcher is `flag_parity_test.go:47`, which constructs the whole `newRootCmd()` tree; green. *(The PLAN's stated catcher is wrong — see Drift 4.)* | closed |
| T-08-03-03 | Spoofing | preserved short bindings | low | accept | Phase diff contains **zero removed `VarP(` lines** — every short-flag change is `Var(`→`VarP(`, an addition, so no pre-existing `-p/-q/-v/-y/-f/-l` was remapped. Divergences recorded in `docs/FLAG-PARITY.md:48`. → Accepted Risks AR-08-02 | closed |
| T-08-04-01 | Denial of Service | Affected BFS depth | medium | mitigate | `traverse.go:545-548` (`validateDepth` then `clampAffectedDepth`); `validate.go:86-94` caps at the same `MaxDepth=50`. Test `traverse_test.go:711-717`. | closed |
| T-08-04-02 | Denial of Service | reverse-adjacency expansion | low | mitigate | `traverse.go:624-630` test symbols recorded then `continue` (never queued); `:611`/`:623` visited-set; `:616-620` dangling edge `ErrNotFound` → `continue`, not abort. | closed |
| T-08-05-01 | Injection / Elevation | stdin path ingestion | **high** | mitigate | Full data-flow trace: `affected.go:43` → `:73 eng.Affected(files, depth)` → `traverse.go:575-578` where each path becomes a **map key only** (`fileSet[f]=true`, compared to `n.FilePath` at `:590`). No `exec`, no `filepath.Join`, no interpolation on any stdin-derived string. `affected_test.go:104-123` asserts paths-only output. *(Extra hardening beyond plan — see Drift 2.)* | closed |
| T-08-05-02 | Denial of Service (ReDoS) | `--filter` glob | medium | mitigate | `affected.go:81` `filepath.Match` with `ErrBadPattern` surfaced at `:83`. Phase `go.mod` diff adds **no** glob/regex dependency (only `spf13/pflag` promoted indirect→direct). Test `affected_test.go:240`. | closed |
| T-08-05-03 | Denial of Service (hang) | `--stdin` on non-TTY | medium | mitigate | `affected.go:219` `bufio.NewScanner(cmd.InOrStdin())`; `:231-232` `Buffer` with both args pinned to `affectedStdinMaxLineBytes+1`; `:247-249` `bufio.ErrTooLong` surfaced not swallowed; `:204` count cap via `ValidateAffectedFiles` (`validate.go:159-164`, `MaxAffectedFiles=10000` at `:40`). Tests `affected_test.go:65,177,221`. *(Stronger than plan — see Drift 3.)* | closed |
| T-08-05-04 | Information Disclosure | worktree notice under quiet/json | low | mitigate | `affected.go:136` `WorktreeNotice` sits strictly after the `--json` early return (`:92-98`) and the `--quiet` early return (`:102-128`). Tests `affected_test.go:104,125`. | closed |
| T-08-06-01 | Repudiation / Integrity | FLAG-PARITY.md accuracy | low | mitigate | `flag_parity_test.go:40-71` walks the real `newRootCmd()` tree recursively, fail-closed on a missing doc (`:41-44`), `t.Fatalf` on any undocumented flag. *(Weaker than its doc comment implies — see Soundness Caveat.)* | closed |
| T-08-06-02 | Tampering | `install --auto-allow` default | low | accept | `install.go:107` — `auto-allow` still `BoolVar(..., false, ...)`: default not flipped, no shorthand added. Divergence documented `docs/FLAG-PARITY.md:16,230`. → Accepted Risks AR-08-03 | closed |
| T-08-07-01 | Tampering (supply chain) | charm.land closure | **high** | mitigate | All four controls located: (1) `archtest/charm_cgo_test.go:87-105` — non-empty assertion `:90-92` + zero-`CgoFiles` `:95-102`; green: *"charm.land closure audited: 10 packages, 0 with CgoFiles"*. (2) govulncheck blocking job `ci.yml:148-160`. (3) SBOM `release.yml:243-254` (syft, per-binary SPDX). (4) reproducibility double-build `ci.yml:165-236`. | closed |
| T-08-07-02 | Denial of Service (build integrity) | CI CGo expectation | medium | mitigate | Repo-wide grep for `CGO_ENABLED` returns **no `=0`** in any non-`.md` file; `ci.yml:139-145` retains `gcc-mingw-w64-x86-64` + `CGO_ENABLED: "1"` + `CC: x86_64-w64-mingw32-gcc`. `git diff 529d818..HEAD -- .github/` is empty. | closed |
| T-08-08-01 | Repudiation / Integrity | BENCHMARKS.md numbers | low | mitigate | **Independently re-derived, not taken on trust.** All 6 rows of `docs/BENCHMARKS.md:109-114` recompute exactly as median-of-3 across `tools/bench/headtohead-linux-amd64-ci-20260719-run{1,2,3}.json` (12/12 spot-checked cells match). Methodology `:7-26`, TS pinned 1.3.1 at `:53`, raw artifacts committed. | closed |
| T-08-09-01 | Tampering | `verify.go` LOCKED constants | **high** | mitigate | `git diff 529d818..HEAD -- internal/upgrade/verify.go .github/workflows/release.yml` → **empty**. `docs/RELEASE-PROCEDURES.md:92-98` reproduces all three constants byte-identical to `verify.go:42-44`; `:107-115` states the same-commit lockstep rule. | closed |
| T-08-09-02 | Spoofing | post-release cosign identity | **high** | mitigate | Judged on runbook correctness, **not** on a release. `verify.go:44` is the anchored source of truth (`^...$`, scoped to `/\.github/workflows/release\.ya?ml@refs/tags/v[0-9]`). `RELEASE-PROCEDURES.md:100-105` states the full-match requirement and forbids weakening to a prefix; `:124-129` gives the `cosign verify-blob --certificate-identity-regexp` command with both anchors and the same scopes. A `pull_request`/`workflow_dispatch` run of the same file yields a SAN ending `@refs/heads/…` or `@refs/pull/…`, which cannot satisfy `@refs/tags/v[0-9]…$`. **Residual:** closed on the specification; the absent tag is correctly not held against it. | closed |
| T-08-09-03 | Tampering (supply chain) | release artifacts | **high** | mitigate | Every control that can exist pre-tag exists: `release.yml:40-43` (`v[0-9]*` trigger), `:196-201` (`id-token: write` on the signing job), `:226-241` (per-binary `cosign sign-blob --bundle`, not checksums-only, matching `defaultVerify`'s `sha256.Sum256(binary)` at `upgrade.go:175-176`), `:243-254` (per-binary syft SPDX), `:294-300` (SLSA3 generator pinned `@v2.1.0`). Pipeline already exercised end-to-end on the real `v0.0.0-rc.3` tag, which `v[0-9]*` and verify.go's regexp match identically to a stable tag. **Residual:** covers the control's *design*; `RELEASE-PROCEDURES.md:117-141` §6 must still be executed by the maintainer after the tag is pushed. | closed |
| T-08-09-04 | Repudiation | drop-in caveat retirement | medium | mitigate | Three declared mechanics all green: `go test ./testdata/golden/...` ok; `go test ./internal/cli/... -run FlagParity` ok; zero-hit grep for the retired caveat. **Was OPEN at audit:** commit `57da8c8` replaced the caveat with a stronger claim than the gate covers — `.planning/PROJECT.md` asserted "`v1.0.0` shipped" at 5 sites while no such tag exists (`git tag --list` → `milestone-v0.1`, `v0.0.0-rc.3`, `v0.1.0`). **Resolved 2026-07-26:** all 5 sites reworded to "parity gates green; signed `v1.0.0` not yet cut"; zero-hit sweep re-run clean; README and `docs/` were already clean. | closed |

*Status: open · closed · open — below `high` threshold (non-blocking)*
*Severity: critical > high > medium > low — only open threats at or above `block_on: high` count toward threats_open*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| AR-08-01 | T-08-02-01 | `files --dir` cannot disclose anything outside the index: `Files()` reads `IterateFiles()` only (no filesystem walk, no path join), so `--dir` is a post-filter over paths the unfiltered `files` command already exposes. Verified in code at `files.go:152,166`. | Phase 08 plan (D-03) | 2026-07-26 |
| AR-08-02 | T-08-03-03 | Short-flag additions never remap an existing binding — the phase diff removes zero `VarP(` lines, so every change is purely additive. Same-letter/different-semantic divergences from TS (e.g. `index -f`) are documented in `docs/FLAG-PARITY.md:48` rather than silently changed. | Phase 08 plan (SURF-03/SURF-05) | 2026-07-26 |
| AR-08-03 | T-08-06-02 | `install --auto-allow` keeps its security-conservative `false` default rather than matching TS. Flipping a security default inside a mechanical surface-reconciliation phase was explicitly rejected (RESEARCH Pitfall 4 / Open Question 1); divergence documented at `docs/FLAG-PARITY.md:16,230`. | Phase 08 plan (SURF-05) | 2026-07-26 |

---

## Mitigations That Diverge From Their PLAN Description

All four remain effective; recorded so the register does not drift from the code.

1. **T-08-02-02** — plan says "plain `strings.HasPrefix`". Current code (`files.go:112-123`, commit `ea2b889`) is `TrimSuffix` + equality + `HasPrefix(path, d+"/")` over a 2-element slice. Anti-ReDoS claim survives unchanged (no pattern engine, no backtracking); the added boundary check additionally fixes `--dir pkga` wrongly matching `pkgab/`.
2. **T-08-05-01** — plan's mitigation is data-flow only. Code adds `affected.go:122-124`, skipping any `FilePath` containing `\n`/`\r` in `--quiet` output (WR-03) — defense-in-depth against line injection into a downstream shell pipeline. **Gap: no regression test covers this branch**; deleting lines 122-124 would fail no test. Low-cost hardening for a follow-up.
3. **T-08-05-03** — plan cites only "Scanner returns on EOF". Code adds two caps the plan never described: a 4096-byte line ceiling (`affected.go:169`, `:231-232`, both `Buffer` args aligned so the cap actually fires — per CR-01) and `MaxAffectedFiles=10000` (`validate.go:40`, enforced `affected.go:204`). Strictly stronger.
4. **T-08-03-02** — plan claims "`go build ./...` catches it." **Factually wrong:** cobra panics on a duplicate shorthand at *flag-registration* time, which `go build` never reaches. The real catcher is `flag_parity_test.go:47`. Threat is closed because a real, green catcher exists — but the guard is one deleted test away from being unenforced.

---

## Soundness Caveat on a Closed Control

`TestFlagParityDocCoversRegisteredFlags` (`flag_parity_test.go:56`) asserts
`strings.Contains(docText, f.Name)` — an unanchored substring match on **long
flag names only**. It does not check shorthand letters, does not bind a flag to
its own command's section, and a short name like `dir` or `json` would pass on
any incidental occurrence anywhere in a 297-line document. The declared
mitigation is present and non-vacuous, so T-08-06-01 is closed at ASVS L1 — but
the guard is weaker than its own doc comment implies.

---

## Findings Outside the Register (informational)

1. **No SUMMARY.md in this phase contains a `## Threat Flags` section** (all 9 checked). The executor's new-attack-surface channel was never populated and contributed no evidence either way; the auditor ran an independent phase-diff sweep instead.
2. **MCP `codegraph_impact` depth description drift** — `mcp/tools.go:165` advertised `"BFS depth (default 5, max 50)"` after this phase changed `defaultDepth` 5→2. Not a control failure (`MaxDepth` and `validateDepth` both apply on the MCP path at `tools.go:327`) but a misleading contract on the exact boundary T-08-01-01 names. `TestFlagParityDocCoversRegisteredFlags` cannot catch it — it walks cobra flags, never MCP tool schemas. **Fixed 2026-07-26** to `"BFS depth (default 2, max 50)"`.
3. **`Engine.Affected` does not enforce `ValidateAffectedFiles` itself** — the `MaxAffectedFiles` cap lives solely in the CLI's `collectAffectedFiles` (`affected.go:204`). This covers all entry points today (`mcp/tools.go` registers 8 tools, none is `affected`), but a future `codegraph_affected` MCP tool would bypass the cap. Out of scope for this phase's register; worth a guard when that tool is added.
4. **`RELEASE-PROCEDURES.md:128`** writes the cosign identity regexp with `[^[:space:]]*` where `verify.go:44` uses `[^\s]*`. Semantically identical under Go's RE2 (cosign is Go), both anchored, both scoped identically — not a weakening, and the runbook states "the source wins" at `:88-90`. Noted only because §5 claims the constants are reproduced *verbatim*.

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-07-26 | 22 | 21 | 1 (T-08-09-04, medium — non-blocking) | gsd-security-auditor (opus), ASVS L1 |
| 2026-07-26 | 22 | 22 | 0 | orchestrator — T-08-09-04 resolved (PROJECT.md reworded, MCP drift fixed) |

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log (AR-08-01..03)
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-07-26

**Not covered by this sign-off:** the signed `v1.0.0` release itself. T-08-09-02
and T-08-09-03 are closed on the correctness of the runbook and pipeline design;
`docs/RELEASE-PROCEDURES.md` §6 (post-release `cosign verify-blob` +
`slsa-verifier verify-artifact`) must still be executed against the real
artifacts once the maintainer cuts the tag.
