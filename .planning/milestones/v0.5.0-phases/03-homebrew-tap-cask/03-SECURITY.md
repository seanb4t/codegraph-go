---
phase: 3
slug: homebrew-tap-cask
status: verified
# threats_open = count of OPEN threats at or above workflow.security_block_on severity (the blocking gate)
threats_open: 0
asvs_level: 1
created: 2026-08-10
---

# Phase 3 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

**Register origin:** `register_authored_at_plan_time: true` — all five plans (03-01 … 03-05) carry a parseable `<threat_model>` block, and all five summaries carry a `## Threat Flags` heading. Mode was **verify-mitigations**, not retroactive STRIDE.

**Auditor was spawned despite the L1 short-circuit.** Step 3's rule (`threats_open: 0 AND register_authored_at_plan_time: true AND asvs_level == 1`) permits skipping straight to the file write. That short-circuit was declined, on the precedent recorded in `02-SECURITY.md`: spawn the auditor on phases carrying critical rows or an irreversible publish. Phase 3 carries **eight high-severity rows** and shipped an irreversible publish (`v0.8.0` plus the permanently-public `seanb4t/homebrew-tap`). The decision was load-bearing — all six findings below are invisible to a first-clause grep, and two of them were live defects.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| `.goreleaser.yaml` → the rendered cask → the user's machine | Config text in this repository becomes Ruby that executes on every installing user's Mac. Nothing between here and there re-reads it. | Executable Ruby; download URLs; sha256 checksums |
| The downloaded archive → the binary `hooks.post.install` executes | Everything before this point is the release pipeline's guarantee; everything after is this hook's. | Signed, notarized darwin binary |
| The hook's assertions → the user's belief that the install succeeded | A green `brew install` is the only signal most users read. | Exit status; raised error text |
| The sentinel → `internal/upgrade` in Phase 4 | Written by Ruby on the user's machine, read later by Go on the same machine, with no schema negotiation between them. | `schema=1` key/value file |
| `seanb4t/codegraph-go`'s release run → `seanb4t/homebrew-tap` | A cross-repository write. The default release token cannot make it — the entire reason a second credential exists. | GitHub App installation token (~1h TTL) |
| The OIDC-bearing release job → every Action executing inside it | That job's token produces the cosign certificate subject the upgrade verifier anchors on. | OIDC id-token |
| `release:rehearse-cask` → the maintainer's real Homebrew prefix | Installs a real cask outside any sandbox. | Local filesystem writes under `HOMEBREW_PREFIX` |
| The recorded evidence → this phase's completion claim | An argument recorded as an observation makes the claim false while everything looks green. | Prose in `03-EVIDENCE.md`, `ROADMAP.md` |

---

## Threat Register

All 16 registered threats verified **CLOSED**, leg by leg. A mitigation row is a conjunction of legs; each leg below was checked separately, per the method lesson recorded in `02-SECURITY.md` (T-02-18 read fully closed on a first-clause grep while its second leg had never been written).

| Threat ID | Category | Component | Severity | Disposition | Mitigation | Status |
|-----------|----------|-----------|----------|-------------|------------|--------|
| T-03-01 | Tampering | `hooks.post.install` — the trust signal itself | high | mitigate | 2/2 legs. Rehearsal asserts the hook's **effect** (`Taskfile.yml:1962`, file on disk) not its exit status; two positive in-hook assertions (`.goreleaser.yaml:551-568`) held standing by `goreleaser_shape_test.go:1216` (raise-count floor) | closed |
| T-03-02 | Tampering | `release:rehearse-cask` on the real Homebrew prefix | medium | mitigate | 3/3 legs. Existing-install refusal `Taskfile.yml:1670`; `CASK_REHEARSE=1` opt-in `:1692`; cleanup trap on the **failure** path `:1725-1741` | closed |
| T-03-03 | Spoofing | Rehearsal's rewritten cask standing in for real output | medium | mitigate | 2/2 legs. Strict diff-line count `Taskfile.yml:1866-1871`, `:1928-1933` (fails closed on a missed `sed`); difference stated in the record `03-02-SUMMARY.md:167` | closed |
| T-03-04 | Tampering | The two post-install assertions — the only permanent guard | high | mitigate | 3/3 legs. **Equality** on `\Av`-normalized values, never containment (`.goreleaser.yaml:564-566`); page check reads the **directory** not the exit code (`:552-555`); version perturbation recorded firing (`Taskfile.yml:2110-2136`). See UF-5 for a narrow residual | closed |
| T-03-05 | Spoofing | The sentinel as Phase 4's install-channel signal | medium | mitigate | 3/3 legs. Location from `Pathname#realpath` symlink resolution, never a path-prefix match (`.goreleaser.yaml:577`); fixed `schema=1` first line (`:580`); cross-phase contract recorded in-comment (`:570-576`) | closed |
| T-03-06 | Repudiation | Completions presented as evidence the binary runs | medium | mitigate | 2/2 legs. Measured perturbation recording a zero-exit install **with** a warning (`Taskfile.yml:2163-2189`); mechanism stated beside **both** stanzas (`.goreleaser.yaml:515-520`, `:534-541`) | closed |
| T-03-07 | Elevation of Privilege | The tap-writing credential's reach | high | mitigate | 4/4 legs, **leg (c) weak** — see AR-01. App installed on one repo (`03-EVIDENCE.md:364`); mint scoped at request time (`release.yml:196-197`); distinctness test (`release_workflow_shape_test.go:1544`, tautological); one-time recorded 403 refusal (`03-EVIDENCE.md:429-438`) | closed — residual named |
| T-03-08 | Information Disclosure | A live installation token crossing a job boundary | medium | mitigate | 4/4 legs. Masking **measured** before being depended on — came back `REDACTED`, so the mint moved **inside** the release job (`release.yml:165-171`, `:196-197`); repo-scoped at mint; ~1h TTL; never echoed (`Taskfile.yml:1753-1758`) | closed |
| T-03-09 | Tampering | A release completing green with the tap credential absent/wrong | high | mitigate | 4/4 legs. **Not presence-only**: `[ -n "${HOMEBREW_TAP_TOKEN:-}" ]` (`Taskfile.yml:765`) fails closed on unset **and** empty, which is precisely the `client.NewIfToken` silent-fallback case; distinctness `:767` (both-empty also fails); each demonstrated in the failing state (`03-03-SUMMARY.md:145-161`); prohibition inline `:752-764` citing rule `84d1gfpywd` | closed |
| T-03-10 | Elevation of Privilege | Additional Actions inside the OIDC-bearing job | medium | mitigate | 4/4 legs. Separate mint job preferred but measurement forbade it; Action pinned to a 40-char SHA at parity with `release-please.yml:81` (`release.yml:192`); inputs scoped to **step** level (`:194-197`); widened surface recorded in-comment (`:164-184`) | closed |
| T-03-11 | Repudiation | An amendment that quietly weakens the claim it renames | high | mitigate | 4/4 legs. Prohibition; amendment written **after** the proof and citing it (`ROADMAP.md:183`); before/after side by side (`03-04-SUMMARY.md:114`); diff assertion that no heading changed and `getMilestonePhaseFilter()` was identical (`:85`, `:88`). **All four legs held and UF-1 still occurred** — see below | closed |
| T-03-12 | Spoofing | A refusal recorded without a positive control | high | mitigate | 3/3 legs. Successful **reverted** write first — `201` create → `204` delete (`03-EVIDENCE.md:388-403`); refusal distinguishable `403` vs `401` (`:432-449`); installation access list recorded beside the result (`:373-377`) | closed |
| T-03-13 | Repudiation | BREW-06's failure-and-recovery half recorded as demonstrated | high | mitigate | 4/4 legs. Prohibition (`03-05-PLAN.md:54`); labels in the **headings** (`03-EVIDENCE.md:465`, `:695`); four-claim-word search executed (`03-05-SUMMARY.md:427`); `ROADMAP.md:184` itself carries the split | closed |
| T-03-14 | Tampering | A release that silently shipped without pushing the cask | high | mitigate | 3/4 legs + 1 **substituted and disclosed**. Preconditions before GoReleaser (`Taskfile.yml:765`, `:767`); tap read through the API after release (`03-EVIDENCE.md:1059-1067`); commit author confirmed `goreleaserbot` (`:1064`). Leg (b) — the cask pipe's own log lines — was **not** pasted; substituted by the App-authored tap commit plus byte-matched checksums, a state observation strictly stronger than a log line, and named as a substitution in `03-05-SUMMARY.md:370-372` rather than smoothed over | closed |
| T-03-15 | Spoofing | A "cold" install that was not cold | medium | mitigate | 3/3 legs. Teardown performed and confirmed (`03-EVIDENCE.md:792-833`); starting state stated as **torn-down, not never-installed**; three shell answers recorded separately (`:923`) | closed |
| T-03-SC | Tampering | `go-md2man/v2`, `blackfriday/v2` entering the shipped binary | low | **accept** | Accepted on RESEARCH.md's manual audit (both OK, no `[SUS]`/`[SLOP]`). Compensating control **half-recorded** — see AR-03 | closed — accepted |

*Status: open · closed · open — below high threshold (non-blocking)*
*Severity: critical > high > medium > low — only open threats at or above `workflow.security_block_on` count toward `threats_open`*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

---

## Unregistered Flags

Security-relevant facts found during the audit that no threat row covers. Two were fixed in this session; four are recorded.

| # | Finding | Severity | Disposition |
|---|---------|----------|-------------|
| **UF-1** | `ROADMAP.md:96`'s Phase 3 index bullet, marked `[x] completed`, still asserted "a real `test:` block, and a proven-recoverable tap-push failure" — **both falsified by criteria 3 and 4 in the same file**. Criterion 3 records that Homebrew Casks have no `test:` stanza at all; criterion 4 (D-19/D-18R) records the failure-and-recovery half as a structural argument with no executed evidence. The amendments landed on the criteria and never on the index line. | **high** | **FIXED** this session. Line 96 amended to match criteria 3 and 4, and to state the `brew trust` requirement. Verified with T-03-11 leg (d)'s own shape: `getMilestonePhaseFilter()` returns an identical phase list, heading count 7 → 7, exactly one line changed, zero heading lines in the diff. |
| **UF-2** | `brew trust` is presented as a routine step with **no security framing**. `docs/RELEASE.md:518-533` and `README.md:78` instruct the user to run the command the error names, without stating what the control is: Homebrew refuses because a third-party cask executes arbitrary Ruby at install time — which this cask's own `postflight` does. The docs recommend `brew trust --tap seanb4t/tap`, granting trust to **every current and future cask and command in the tap**, when the error itself offers the narrower `brew trust --cask seanb4t/tap/codegraph`. | medium | **Recorded + todo.** Below `block_on: high`, non-blocking. The register was authored before Homebrew 6.0.16's refusal was discovered, so no row could have covered it. Impact bounded by T-03-07/T-03-12 (tap credential scoped and proven refused cross-repo); likelihood of the *step* is 100%, of *exploitation* requires tap compromise. |
| **UF-3** | `03-EVIDENCE.md:829-831` and `:1039` claim a failed install "can leave the sentinel behind with no cask installed". **This is false.** The hook writes the sentinel at `.goreleaser.yaml:577-587`, after **both** raises (`:553-555`, `:566-568`) — a gate failure raises before the sentinel exists. The evidence file corroborates its own contradiction: after plan 03-04's Mutation 2 it found 30 orphaned man pages but the sentinel glob returned nothing (`:807-808`, `:815`). Only man pages leak. | low (doc defect) | **Recorded + todo.** Phase 4 reads this sentinel and must not design around a stranded one. |
| **UF-4** | `TestHomebrewTapTokenScopedToReleaseJob` carried **no positive-presence assertion on the match set**. Every scoping assertion in it is negative (no reference outside `release.yml`, none outside the release job, none above step level) and all three pass vacuously on an empty `refs` slice. The `len(files) == 0` Fatal guards non-vacuity of the *input scan* only — not of the *match set*, which is the quantity carrying the invariant. **Sixth instance of this repo's vacuous-pass family**, and a direct violation of normative rule `84d1gfpywd`. | medium | **FIXED** this session. Added a set-membership floor asserting `release.yml` references all three of `HOMEBREW_TAP_APP_ID`, `HOMEBREW_TAP_APP_PRIVATE_KEY`, `HOMEBREW_TAP_TOKEN`, probed **through the same walker** the negative assertions use. Proven RED by renaming one credential: the rename left `refs: 2`, both correctly scoped — so the old test **would have passed green** with that credential entirely unguarded. Vulnerability was live, not theoretical. |
| **UF-5** | Post-install assertion one can be satisfied by **stale** man pages. The glob `#{man_dir}/codegraph*.1` targets the shared `HOMEBREW_PREFIX/share/man/man1`, not the Caskroom, with no pre-install baseline. `03-EVIDENCE.md:815-833` documents 30 orphaned pages surviving a failed install's rollback (Homebrew purges the Caskroom but never runs the cask's uninstall hook). A later install whose binary runs `man` successfully but writes nothing would find that residue and pass. | medium | **Recorded + todo.** Exposure is narrow — `system_command`'s `must_succeed: true` still catches a binary that cannot run at all. But the assertion does not distinguish "this install wrote pages" from "pages are present", and D-12 accepts that the RED proof does not re-fire, making this one of only two permanent standing gates. |
| **UF-6** | T-03-SC's compensating control is half-recorded. `03-01-PLAN.md:234,396` requires recording that the two promoted deps enter **the SBOM and** `govulncheck`'s reachable set. `03-01-SUMMARY.md:105` records govulncheck (0 reachable). No SBOM result is recorded anywhere. | low | **Recorded** as AR-03 below. |

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| AR-01 | T-03-07 leg (c) | The distinctness test `TestHomebrewTapAppSecretsDistinctFromReleasePleaseAppSecrets` (`release_workflow_shape_test.go:1544`) is **tautological**: it compares the in-test constant `homebrewTapCredentialNames[:2]` against the hardcoded literal `{"APP_ID","APP_PRIVATE_KEY"}` and reads neither workflow file. If either workflow changed its secret names the test stays green; it can only fail if someone edits the test file. The leg is present as declared but does not bind reality. Not fixed here — the fix (decode both workflows and compare actual referenced names) is larger than a secure-phase edit. | maintainer | 2026-08-10 |
| AR-02 | T-03-07 | Cross-repo refusal is proven **once** by design (D-17 declined a standing CI assertion). A later widening of the App's installation is not detected. The named remedy — a scheduled re-mint asserting the `403` persists — does not exist. | maintainer | 2026-08-10 |
| AR-03 | T-03-SC / UF-6 | The two newly-compiled deps (`go-md2man/v2`, `blackfriday/v2`) are accepted on RESEARCH.md's manual audit. The compensating control promised two records; only the `govulncheck` half (0 reachable) was recorded. The SBOM half is **unrecorded, not clean-by-verification**. Low + `accept`, so non-blocking. | maintainer | 2026-08-10 |
| AR-04 | T-03-04 / D-12 | The RED proof for the post-install assertions does not re-fire. The two in-hook assertions are the whole standing guard, and UF-5 names a narrow way assertion one can be satisfied by residue. | maintainer | 2026-08-10 |
| AR-05 | T-03-13 / D-18R | BREW-06's failure-and-recovery half has **zero executed evidence** by maintainer decision. It is a structural argument from the pinned GoReleaser module's source (the cask publisher runs strictly after the release publisher). Never observed, none planned. Must never be reported as demonstrated. | maintainer | 2026-08-10 |
| AR-06 | T-03-14 | The tap-push **UPDATE** path and `brew upgrade` remain unexercised — scope was reduced to one release. Both are code this project does not own, cannot patch, and which surface on the next natural release regardless. | maintainer | 2026-08-10 |
| AR-07 | T-03-15 | The cold install's starting state was **torn down**, not never-installed — a weaker form of evidence, named rather than smoothed over. | maintainer | 2026-08-10 |
| AR-08 | T-03-09 leg (c) | The two credential preconditions were demonstrated by executing their guard text verbatim under `sh -c` rather than through `task`, because the "no tag at HEAD" precondition halts the chain first. Guard text proven; chain integration not exercised end-to-end. | maintainer | 2026-08-10 |
| AR-09 | UF-2 | The published install contract requires the user to run `brew trust`, i.e. to **opt out of a Homebrew security control**. Accepted as inherent to third-party cask distribution; the docs' recommendation of the broader `--tap` form over the narrower `--cask` form is filed as a todo, not accepted. | maintainer | 2026-08-10 |

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-08-10 | 16 | 16 | 0 | `gsd-security-auditor` (opus) + orchestrator cross-check |

**Audit method.** Verify-mitigations mode, ASVS L1, `block_on: high`. Every mitigation row was decomposed into its constituent legs and each leg checked separately; a leg that could not be located was treated as MISSING, not assumed. Auditor findings were independently re-verified by the orchestrator against source before being recorded here — UF-1, UF-4, UF-5, UF-3 and T-03-09's non-vacuity were each confirmed directly, per this repo's recorded practice of confirming with two independent bases.

**Vacuous-guard sweep.** 15 guards examined. 13 carry a positive assertion. Two flagged: UF-4 (fixed) and AR-01 (accepted). Notable non-findings — `Taskfile.yml:1866-1871`'s diff-line count fails closed when its `sed` anchor misses (empty diff → `grep -c` yields `0`, `0 != 4` → hard error), and every line-number lookup at `:1855`, `:2105`, `:2163` is explicitly `-z`-guarded against a missed anchor. The authors were clearly applying rule `84d1gfpywd`; UF-4 slipped because non-vacuity was guarded on the input scan rather than on the match set.

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer) — 15 mitigate, 1 accept
- [x] Accepted risks documented in Accepted Risks Log — AR-01 … AR-09
- [x] `threats_open: 0` confirmed — 16/16 registered threats closed, leg by leg
- [x] Unregistered flags recorded — UF-1 and UF-4 fixed this session; UF-2, UF-3, UF-5, UF-6 recorded with todos
- [x] Auditor spawned despite the ASVS L1 short-circuit, on the `02-SECURITY.md` precedent — decision was load-bearing
