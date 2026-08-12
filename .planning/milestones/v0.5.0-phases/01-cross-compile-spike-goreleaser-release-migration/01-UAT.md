---
status: complete
phase: 01-cross-compile-spike-goreleaser-release-migration
source: [01-01-SUMMARY.md, 01-02-SUMMARY.md, 01-03-SUMMARY.md, 01-04-SUMMARY.md, 01-05-SUMMARY.md, 01-06-SUMMARY.md]
started: 2026-08-11T00:00:00Z
updated: 2026-08-11T00:00:00Z
---

## Current Test

[testing complete]

## Tests

### 1. REL-05 real-hardware execution evidence
expected: The zig-crossed linux binaries EXECUTED on real, non-emulated Linux hardware of their own architecture and indexed a real tree to a non-zero graph — canary run 31273571889, two resolved REL05-EVIDENCE lines (x86_64 and aarch64, both files=430 symbols=4281), runner labels confirmed via the jobs API.
coverage_id: 01-D5
requirement: REL-05
result: pass

### 2. Sign-pipe proof escalated from config-only to real-pipe
expected: Plan 01-02 could only prove the `signs:` sidecar template resolves correctly via a standalone text/template execution — it never ran the real pipe against a real cosign invocation, and explicitly routed full proof to plan 01-06. That proof landed: `task release:dry-run-signed` reports SIGN-EVIDENCE count=4 distinct=4 with per-platform names. Re-confirmed on 2026-08-11.
coverage_id: 02-D3
requirement: REL-06
result: pass
source: automated
evidence: superseded by coverage entry 06-D1 (test 21) — the wave-3 leg 01-02 was waiting on has executed. Re-run 2026-08-11: `task release:dry-run-signed` exit 0, SIGN-EVIDENCE count=4 distinct=4, SBOM-EVIDENCE count=4 distinct=4.

### 3. `release:` block behavior observed against a real release
expected: The `release:` pipe is structurally unexercisable by any dry run — `--skip=publish,sign` cannot reach it, and no local mode publishes to a real GitHub Release without publishing to one. Its config pinning was statically proven; its real behavior was first observed during plan 01-05's actual v0.5.1 release.
coverage_id: 02-D5
requirement: REL-06
result: pass
source: automated
evidence: `gh release view v0.5.1` — published 2026-08-09, isDraft=false, isPrerelease=false: the `prerelease: auto` pin resolved a stable tag correctly against the real Releases API.

### 4. JOINT wave-2 end-state (REL-06/REL-07) re-evaluated after merge
expected: Plan 01-03 could not evaluate this from its own worktree — the `.goreleaser.yaml` half was plan 01-02's concurrent commit. Both sides re-run at HEAD with comments stripped: release.yml `sha256sum` = 0 and `--clobber` = 0 (hand-rolled writer and clobber publisher gone); .goreleaser.yaml `checksum.ids: [raw, zip]` = 1 and `replace_existing_artifacts: true` = 1 (declarative writer live). All four sides confirmed.
coverage_id: 03-D3
requirement: REL-07
result: pass
source: automated
evidence: both sides re-run at HEAD with comments stripped — release.yml sha256sum=0, --clobber=0; .goreleaser.yaml checksum.ids [raw, zip]=1, replace_existing_artifacts: true=1. All four sides confirmed.

### 5. ROADMAP Phase 1 Success Criterion 3 reconciliation
expected: Plan 01-04 deliberately did NOT hand-edit ROADMAP.md's Phase 1 criteria — they are a tool-owned generated artifact — and recorded the reconciliation as owed. ROADMAP.md:110 now reads `gh attestation verify`; `slsa-verifier` appears nowhere in the file. Reconciled.
coverage_id: 04-D3
requirement: REL-08
result: pass
source: automated
evidence: `rg 'gh attestation verify|slsa-verifier' .planning/ROADMAP.md` — one hit, ROADMAP.md:110, reading `gh attestation verify`. No slsa-verifier occurrence in the file.

### 6. Post-release verification fires automatically and goes green
expected: `.github/workflows/post-release-verify.yml` triggers on `workflow_run` (never `release: [published]`, which fires before assets upload), with all three jobs carrying the identical event-aware conclusion guard. Against the real v0.5.1 release it ran green — run 31285981504, all four jobs. Its first real run also exposed three false negatives in its own verifiers, fixed in PR #41.
requirement: REL-06, REL-08
result: pass
source: automated
evidence: `gh run view 31285981504` — workflow=post-release-verify, conclusion=success, 4/4 jobs success (resolve-tag, verify-supply-chain, self-upgrade darwin/arm64, self-upgrade linux/amd64).

### 7. Published asset set classified, not counted
expected: `task verify:release-assets` classifies published assets REQUIRED / ALLOWED / UNCLASSIFIED with both-directions set equality — never a fixed total. Observed on v0.5.1: 8 payloads covered by checksums (4 raw + 4 zip), 0 sidecars in checksums, no self-reference, no duplicates, no uncovered payloads, 17 total published (8 payloads + 8 sidecars + 1 checksums).
requirement: REL-06, REL-07
result: pass
source: automated
evidence: `gh release view v0.5.1 --json assets` — 17 assets = 4 raw + 4 zip + 4 .sigstore.json + 4 .spdx.json + 1 checksums. Checksums file re-downloaded: 8 lines, 4 raw + 4 zip, 0 sidecars, 0 duplicates, 0 self-reference.

### 8. A genuinely shipped prior binary self-upgrades byte-identically
expected: `task verify:self-upgrade` resolves the prior release by semver-predecessor order (not `gh`'s publication chronology), runs `codegraph upgrade "$TAG"` explicitly, and asserts byte identity against the re-downloaded published asset. A v0.4.0 binary from the OLD pipeline upgraded to v0.5.1 from the NEW one on both darwin/arm64 and linux/amd64, with `upgraded_sha256` == `published_sha256` in each case.
requirement: REL-08
result: pass
source: automated
evidence: both self-upgrade legs green in run 31285981504 (darwin/arm64, linux/amd64); 01-05-SUMMARY records upgraded_sha256 == published_sha256 for each.

### 9. zig cross-compile invariant on both linux build ids
expected: codegraph-linux-amd64 cross-compiles via zig cc/zig c++ (x86_64-linux-gnu) matching codegraph-linux-arm64; darwin build ids carry no CC/CXX override
result: pass
source: automated
coverage_id: 01-D1

### 10. One goreleaser invocation produces all four correctly-typed binaries
expected: `task release:dry-run` produces 2 ELF + 2 Mach-O binaries verified by a file(1) case statement, not a bare exit code
result: pass
source: automated
coverage_id: 01-D2

### 11. `release:dry-run` halts on a named precondition when syft is absent
expected: Halts with an actionable named message rather than failing mid-pipe, and resumes cleanly once syft is restored
result: pass
source: automated
coverage_id: 01-D3

### 12. Permanent canary with the D-04 FAIL-bar list, SHA-pinned actions
expected: linux-cross-canary.yml exists and is dispatchable, carries the V1–V5 FAIL-bar variation list written before any dispatch, contents=read-only permissions, all third-party Actions SHA-pinned
result: pass
source: automated
coverage_id: 01-D4

### 13. REL-05 decided on third-party-re-inspectable evidence
expected: Canary run 31273571889 — 0 non-success jobs, all three jobs present, exactly two resolved REL05-EVIDENCE lines
result: pass
source: automated
coverage_id: 01-D6

### 14. Dual archives — raw stays binary, zip added alongside
expected: `archives:` split into id raw (formats [binary], name_template byte-unchanged) and id zip (formats [zip], same stem), with the raw entry's contract with internal/upgrade.releaseAssetName() provably unchanged
result: pass
source: automated
coverage_id: 02-D1

### 15. Checksums scoped to exactly the 8 downloadable payloads
expected: `checksum.ids: [raw, zip]` covers the 8 payloads and nothing else (D-12)
result: pass
source: automated
coverage_id: 02-D2

### 16. SBOMs per binary with four distinct SPDX names
expected: `sboms:` reproduces the hand-rolled syft loop declaratively with `artifacts: binary` explicit and a NAME-derived documents template producing four distinct <asset>.spdx.json names
result: pass
source: automated
coverage_id: 02-D4

### 17. `release:goreleaser` is the single definition site
expected: One definition of `goreleaser release --clean` with prefer-then-build GoReleaser resolution, pin assertion before invoking, and six named preconditions
result: pass
source: automated
coverage_id: 03-D1

### 18. release.yml collapsed to one job
expected: One `release` job with exactly one run: body (task release:goreleaser); build matrix, assemble job, and hand-rolled checksum/sign/sbom/rename steps all deleted
result: pass
source: automated
coverage_id: 03-D2

### 19. Native attestor swapped in, SLSA generator deleted
expected: provenance: job deleted entirely; actions/attest-build-provenance SHA-pinned as the last step of the goreleaser job over the 8-payload subject set; attestations: write added; exactly one id-token: write holder
result: pass
source: automated
coverage_id: 04-D1

### 20. Published verification instructions rewritten
expected: docs/RELEASE.md, docs/RELEASE-PROCEDURES.md, SECURITY.md, README.md and REQUIREMENTS.md name `gh attestation verify` in place of `slsa-verifier verify-artifact` for post-migration releases
result: pass
source: automated
coverage_id: 04-D2

### 21. Real sign and SBOM pipes produce four distinct published names
expected: `task release:dry-run-signed` runs the real pipes against a throwaway local cosign key and asserts four distinct published signature names and four distinct SBOM names from dist/artifacts.json, treating a zero-match filter as a hard failure
result: pass
source: automated
coverage_id: 06-D1

### 22. `sign-snapshot` canary keeps the proof re-firing
expected: sign-snapshot job in linux-cross-canary.yml re-fires this proof on every .goreleaser.yaml change, with no needs: on the exec jobs and no permissions: escalation
result: pass
source: automated
coverage_id: 06-D2

## Summary

total: 22
passed: 22
issues: 0
pending: 0
skipped: 0
blocked: 0

## Gaps

[none yet]
