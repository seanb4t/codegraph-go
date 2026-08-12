---
phase: 04
slug: codegraph-upgrade-homebrew
status: secured
# threats_open = count of OPEN threats at or above workflow.security_block_on severity (the blocking gate)
threats_open: 0
asvs_level: 1
register_authored_at_plan_time: true
created: 2026-08-11
---

# Phase 04 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

Register authored at plan time across all six PLAN.md files (22 threats, 5 high / 10 medium /
7 low; 18 `mitigate`, 4 `accept`). Verified leg-by-leg against the implementation at
`0e87f00`, diff base `98f7935`.

**Audit depth note.** ASVS L1 with `register_authored_at_plan_time: true` and a preliminary
`threats_open: 0` permits the workflow's short-circuit (skip the auditor, write this file).
That short-circuit was **deliberately not taken**. Repo memory `xkbc8m36hm` records that the
same short-circuit would have skipped Phase 3's audit, which then found 6 unregistered flags of
which two were live defects — and notes that was already the second consecutive phase where it
would have rubber-stamped. Phase 4 carries 5 high rows, a binary-mutation path, a supply-chain
signature-ordering change, and a real `brew install` executed against the maintainer's
workstation: both of the recorded "spawn anyway" conditions. Every mitigation below was checked
against the shipped code or the recorded evidence, not inferred from SUMMARY claims.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| Homebrew-managed install tree ↔ `codegraph upgrade` | The Caskroom/Cellar tree Homebrew owns and records in `INSTALL_RECEIPT.json`, versus this binary's self-replacement path | Resolved binary path, install directory, cask token, Homebrew's own receipt |
| Published release assets ↔ `verify:self-upgrade` | GitHub release downloads (prior-release binary + cosign bundle) entering a target that makes a file executable and runs it | Release binary bytes, cosign signature bundle, certificate identity |
| Release-identity policy ↔ its restatements | One compiled `releaseWorkflowRefPattern` versus five hand-written restatements across executed and published files | `--certificate-identity-regexp` literals |
| Maintainer workstation ↔ the acceptance run | A real `brew tap` / `brew trust` / `brew install` plus a payload substitution, performed on a live machine | Homebrew tap list, per-user trust store, Caskroom payload bytes, shared man directory |
| Planning artifacts ↔ scope-sensitive readers | `.planning/ROADMAP.md` structure consumed by `extractCurrentMilestone` / `getMilestonePhaseFilter` | Milestone scope, phase membership |

---

## Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation | Status |
|-----------|----------|-----------|----------|-------------|------------|--------|
| T-04-01 | Tampering | `detectBrewManaged` (`internal/upgrade/brew.go`) | medium | mitigate | Detection requires a Homebrew-authored `INSTALL_RECEIPT.json` at the tree-specific location (`brew.go:14,39,49`); shape alone never suffices. 16-row table incl. executing false-positive rows for `Cellar`/`Caskroom`-shaped paths with no receipt | closed |
| T-04-02 | Spoofing | the install tree itself | low | accept | An attacker who can author a fake Caskroom tree *and* a receipt above the binary already controls the install location. See AR-01 | closed (accepted) |
| T-04-03 | Denial of Service | `filepath.EvalSymlinks` error handling | medium | mitigate | `brew.go:59-66` — an unresolvable path returns not-detected **immediately**: no fallback to the unresolved path, no continued shape scan. Single-error contract, cited in-code as the contract T-04-03 relies on | closed |
| T-04-04 | Elevation of Privilege | `Options.Force` | low | mitigate | `upgrade.go:94` — the refusal branch **never reads** `opts.Force`; its absence is the enforcement mechanism (D-06), not a runtime check. No code path exists to bypass. `TestUpgradeRun_ForceDoesNotOverrideBrewRefusal` passes | closed |
| T-04-05 | Tampering | `hooks.post.install` man-page assertion | medium | mitigate | Per-path freshness baseline capturing **mtime AND size** (`.goreleaser.yaml:547-580`); size catches the same-tick rewrite mtime alone misses. Replaces the glob that residue from a prior failed install satisfied vacuously (UF-5) | closed |
| T-04-06 | Denial of Service | removal of the marker write | low | accept | Detection is structural (D-02/D-03); the marker had exactly one consumer. See AR-02 | closed (accepted) |
| T-04-07 | Tampering | `release:rehearse-cask` assertion removal | medium | mitigate | Marker assertions removed with an **independent** freshness re-check added in the rehearsal target; `task check:goreleaser` and `task --list-all` both exit 0 post-merge | closed |
| T-04-08 | Repudiation | `03-EVIDENCE.md` | low | mitigate | The false sentinel-stranding claim corrected in place with a dated amendment note; two todos folded and moved to `todos/completed/` | closed |
| T-04-09 | Tampering | `.planning/ROADMAP.md` structure | high | mitigate | `###` heading count is **7 on `origin/main` and 7 at HEAD** — 04-03's edits were value-level only, no version-bearing or check-marked heading introduced. `getMilestonePhaseFilter` independently re-proved to return 4 phases incl. `04-codegraph-upgrade-homebrew` after every ROADMAP write this phase | closed |
| T-04-10 | Repudiation | amended criteria | medium | mitigate | Every amendment carries a dated `**Amended 2026-08-11 (D-xx, plan 04-03)**` marker naming its governing decision, and quotes the superseded wording so prior references still resolve | closed |
| T-04-11 | Information Disclosure | falsification citations | low | mitigate | D-04's falsified premise recorded with its evidence (Homebrew PR #19121, Homebrew 6.0.0 Linux-cask items) rather than silently deleted | closed |
| T-04-12 | Repudiation | `README.md`, `docs/RELEASE.md` | medium | mitigate | Stale present-tense claims (`does not detect`, `undefined interaction`, `not yet shipped`) removed — 0 occurrences remain; `brew upgrade codegraph` now present in 3 places. Dated amendment note added | closed |
| T-04-13 | Information Disclosure | `--help` | low | mitigate | Cobra `Long`/`Example` state the refusal and both exit codes; `TestUpgradeCommand_HelpDocumentsBrewRefusalAndExitCodes` pins the text with **present-form** `Contains` assertions, observed RED before GREEN | closed |
| T-04-14 | Tampering | documented override | medium | mitigate | No override is documented, because none exists — T-04-04's structural absence of a `Force` read means there is nothing to document | closed |
| T-04-15 | Elevation of Privilege | `verify:self-upgrade` download-then-execute | high | mitigate | `cosign verify-blob` (`Taskfile.yml:2760`) strictly precedes `chmod +x` (`:2769`), under `set -euo pipefail` (`:2587`) so a failed verify aborts before the file is ever made executable. Post-chmod guard at `:2776` refuses to execute a file that is missing or non-executable | closed |
| T-04-16 | Spoofing | identity-regexp drift | high | mitigate | `TestCosignIdentityPolicyBoundaryParityWithCompiledPattern` — **three non-subsuming checks**: (1) total floor `len(all) >= 7`, derived from a measured pre-task baseline of 6 so it cannot be pre-satisfied; (2) **per-file** membership requiring every one of the five restatement files to yield ≥1 literal; (3) region-scoped requirement that ≥1 literal come from the `verify:self-upgrade` region, so relocation goes RED. Scan set verified **complete**: the only other file in the repo containing `--certificate-identity-regexp` is the test itself. Paired with `..._ZeroLiteralsIsError` | closed |
| T-04-17 | Tampering | bundle download | medium | mitigate | `Taskfile.yml:2737-2745` — named hard-fail asserting the cosign bundle actually downloaded, because `gh release download` **can exit 0 having matched nothing**. Asserts on the artifact, not the exit status | closed |
| T-04-18 | Repudiation | unexecuted RED proof | medium | accept | Recorded as not-observed rather than claimed. See AR-03 | closed (accepted) |
| T-04-19 | Tampering | the maintainer's Homebrew install | high | mitigate | One trapped script, `trap restore_and_cleanup EXIT INT TERM` armed before the first mutating byte, proved by a `TRAP_ARMED=1` marker appended between trap registration and `brew tap`. Gate (`04-06-PLAN.md:442`) asserts **per-key** `-eq 1` cardinality for `PAYLOAD_SHA256_BEFORE` and `_AFTER` separately (the rejected alternation form survives only in the value-distinctness leg, correctly), `RESTORE_VERDICT=ok`, `RESTORE_INVOCATIONS=1`, `RUN_ID` binding across baseline/receipt/harness-log so a stale artifact cannot stand in, tap/trust actions agreeing with the recorded baseline, and a **live re-probe** of the machine. Receipt: equal hashes `99763960…b32e62`, `TAP_ACTION=untapped`, `TRUST_ACTION=left-trusted`. Independently re-verified by the orchestrator on 8 dimensions after the run | closed |
| T-04-20 | Repudiation | `04-EVIDENCE.md` | high | mitigate | Leg 3 marked `NOT executed, NOT claimed` (`:32`, `:335`) with a stated closing condition; machine-readable summary line carries `leg3_executed=no` alongside its positive results. The document explicitly names blurring the substituted-payload observation with a natural install as "the most consequential available" error | closed |
| T-04-21 | Spoofing | `brew trust --tap` | medium | accept | Pre-existing grant on this workstation; the run did not create it and correctly did not revoke it. See AR-04 | closed (accepted) |
| T-04-22 | Information Disclosure | captured transcripts | low | mitigate | Transcripts record paths, versions, exit codes and hashes; no credentials or tokens. Tap App secrets never enter this path | closed |

*Status: open · closed · open — below high threshold (non-blocking)*
*Severity: critical > high > medium > low — only open threats at or above `workflow.security_block_on` (high) count toward threats_open*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| AR-01 | T-04-02 | Spoofing the install tree requires authoring both a Caskroom/Cellar-shaped path **and** a Homebrew-formatted `INSTALL_RECEIPT.json` above the binary. An attacker with that much write access to the install location already controls the binary; the detection was never a security boundary against them. Consistent with the maintainer ownership ruling — this is not wiring this project would fix | maintainer (plan-time) | 2026-08-11 |
| AR-02 | T-04-06 | Removing the Phase-3 marker write cannot deny service because detection no longer consults it (D-02/D-03). The marker had exactly one consumer — Phase 4's detection — which chose structure instead | maintainer (plan-time) | 2026-08-11 |
| AR-03 | T-04-18 | The RED proof for one leg was not executed. Recorded as not-observed rather than asserted; the phase does not rest a claim on it | maintainer (plan-time) | 2026-08-11 |
| AR-04 | T-04-21 | `brew trust --tap` grants a broader trust than the `brew trust --cask` form Homebrew's own error suggests. Carried forward from Phase 3 (UF-2, already filed as a todo); Phase 4 did not widen it and correctly preserved the pre-existing grant rather than re-granting or revoking it | maintainer (plan-time) | 2026-08-11 |

---

## Unregistered Flags (incidental findings)

| Flag | Source | Finding | Severity | Disposition |
|------|--------|---------|----------|-------------|
| UF-1 | 04-REVIEW.md WR-01 | `verify:self-upgrade` has a named "matched nothing" hard-fail for the **signature bundle** download (`:2745`) but not for the **release binary** download immediately above it (`:2733`), despite identical risk. Not exploitable — a missing binary still aborts via `cosign verify-blob` failing under `set -e`, and `:2776` refuses to execute a missing/non-executable file — but it loses the actionable named diagnostic the rest of the target uses consistently | low | Recommend fixing for diagnostic consistency; **not** a threat gap. T-04-17 is closed because the bundle is precisely the leg that has the named check |
| UF-2 | 04-REVIEW.md IN-01 | TOCTOU between `detectBrewManaged`'s check and `atomicSwap`'s later action on `targetPath` | info | Accepted. Requires an attacker with write access to the binary's directory, at which point the check was never a boundary against them. Same reasoning as AR-01. No fix proposed |
| UF-3 | this audit | Three of six SUMMARYs (**04-01**, 04-03, 04-04) have no `## Threat Flags` section — and 04-01 is the plan shipping the detection predicate. Hygiene regression against Phase 3, where all five SUMMARYs carried one | low | No threat left unverified as a result: T-04-01/03/04 and T-04-09/10/11 and T-04-12/13/14 were each verified directly against the implementation during this audit. Recommend the executor prompt require the section |

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-08-11 | 22 | 22 | 0 | orchestrator (inline; auditor subagent stopped by user, audit completed by direct verification against the implementation) |

**Method.** Every high row verified by direct observation of the shipped artifact — file and line
cited above — not from SUMMARY claims. Machine restoration for T-04-19 was independently
re-probed against the live workstation after the run (cask absent, tap untapped, pre-existing
trust grant preserved, 0 orphaned `codegraph*.1` man pages, no Caskroom dir, no brew-prefix
binary, `~/.local/bin/codegraph` mtime unchanged, 2 stale Phase-3 mutation trust entries left
alone). Supporting gates at the same commit: `task test` 6 steps / 59 packages / 0 FAIL;
`go build`, `go vet`, `task check:goreleaser`, `go.mod`/`go.sum` clean; regression gate over four
prior-phase packages green; deep code review 0 Critical.

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed
