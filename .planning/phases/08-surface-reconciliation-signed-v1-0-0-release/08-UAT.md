---
status: complete
phase: 08-surface-reconciliation-signed-v1-0-0-release
source: [08-VERIFICATION.md]
started: 2026-07-19T21:30:00Z
updated: 2026-07-28T00:00:00Z
---

## Current Test

[testing complete]

Closed 2026-07-28. One in-scope human-verification item (Test 2) — passed.
Test 1 is recorded `out_of_scope`: REL-02 moved to Phase 9, so it is no longer a
Phase 8 obligation. It was NEVER EXECUTED and is not claimed as passed — no
`v1.0.0` tag exists. Its full two-block history and scope_note are preserved
verbatim above as the record of why it blocked twice.

## Tests

### 1. Cut the signed v1.0.0 release (REL-02, maintainer-manual)
expected: |
  Per docs/RELEASE-PROCEDURES.md, run the pre-tag gate for all 6 targets
  (linux/amd64, linux/arm64, windows/amd64, windows/arm64, darwin/amd64, darwin/arm64):
    for pair in linux/amd64 linux/arm64 windows/amd64 windows/arm64 darwin/amd64 darwin/arm64; do
      GOOS="${pair%/*}" GOARCH="${pair#*/}" go list -mod=readonly ./... >/dev/null && echo "OK $pair" || echo "FAIL $pair"; done
  Then squash-merge gsd/v1.0-drop-in-parity-human-ux → main, `git tag v1.0.0`,
  `git push origin main && git push origin v1.0.0`. Confirm release.yml publishes a
  signed release and that cosign verify-blob (identity regexp in verify.go) +
  slsa-verifier verify-artifact both succeed. Do NOT alter verify.go's LOCKED
  cosign identity or release.yml's v[0-9]* trigger.
result: out_of_scope
former_result: blocked
reason: "Requirement moved to Phase 9 on 2026-07-28 — not a Phase 8 obligation. Never executed; no release was cut."
note: |
  BLOCKED TWICE. Second attempt 2026-07-27 (this session): re-presented after 4 of the
  5 readiness gaps below closed (see Readiness Recheck). Maintainer answered "blocked"
  again, on a different and stronger basis than the first time:

    - The release MECHANISM is being replaced, not just postponed. Maintainer: "We need
      a PR, we don't do tags - it's release please." Captured as backlog Phase 999.3
      (release-please + GoReleaser, modeled on seanb4t/engram; commit 4de3aea).
    - Investigation during this session established that release-please is NOT currently
      configured in this repo — `rg -i release-please` across the tree returns zero hits.
      The only mechanism that exists today is the tag-triggered release.yml this test
      describes. So the test is not merely unexecuted; the procedure it prescribes is one
      the maintainer has decided not to use.
    - Maintainer also judged this test mis-scoped: "I don't need to test that slsa, etc,
      work - they're not my projects." Partly correct — see Test 1 Scope Note below.

  Still no v1.0.0 tag created or pushed. Still recorded as blocked, not as an issue: no
  defect was found, so this produces no gap entry and no fix plan. REL-02 remains
  unverified and Phase 8 cannot be marked complete on the strength of this UAT.

  Corroborating gaps found during the FIRST UAT run (2026-07-26, commit 387cb4b):
    - Phase 8 has NO 08-SECURITY.md — phases 01-07 all have one; /gsd-secure-phase 8
      was never run. This workflow's own verify:post gate blocks phase advancement
      on exactly that artifact.
    - ROADMAP.md still lists BOTH Phase 7 and Phase 8 as unchecked "[ ]".
    - 07-VERIFICATION.md and 08-VERIFICATION.md are both status: human_needed.
    - internal/daemon carries four known-flaky, load-sensitive timing tests that
      have never been de-flaked.
    - Phase 7 UAT test 3 (Windows runtime) was SKIPPED for lack of a Windows host —
      Windows is covered only by cross-vet, never executed.

scope_note: |
  Test 1 Scope Note (recorded 2026-07-27) — the maintainer's critique is half right, and
  this test should be rewritten when REL-02 is next attempted under Phase 999.3.

  WRONG as written: "Confirm cosign verify-blob + slsa-verifier both succeed" reads as a
  requirement to test third-party tooling. Sigstore and SLSA working is not this project's
  responsibility and should not be a UAT item.

  RIGHT, and must be preserved in any rewrite: the risk actually being guarded is
  first-party self-consistency — does *this repo's* release.yml sign with an identity
  *this repo's own* internal/upgrade/verify.go accepts. If those drift, `codegraph upgrade`
  breaks for every user, silently. That coupling is real and load-bearing.

  Already automated, so NOT a UAT item: the verify.go side is pinned by locked-constant
  tests over releaseWorkflowRefPattern / releaseOIDCIssuer / releaseRepoSlug.

  The only genuinely first-party, genuinely manual claim left: "a published release exists
  and `codegraph upgrade` accepts it end-to-end" — unverifiable until a real release exists.
  A rewrite should reduce Test 1 to exactly that.

ownership_note: |
  Ownership moved to Phase 9 (recorded 2026-07-28). REL-02 was rewritten as a
  release-automation property — release-please owns the version bump, `CHANGELOG.md`,
  and tag creation, with the resulting signed artifacts still satisfying
  `internal/upgrade/verify.go`'s cosign identity — and reassigned from Phase 8 to Phase 9.
  This test exercises a requirement Phase 8 no longer owns: it is not a Phase 8 obligation
  and must not be re-run as written. On UAT close (2026-07-28) `result` moved
  `blocked` → `out_of_scope` with `former_result: blocked` retained — "blocked" would have
  implied Phase 8 was still waiting on it, which is no longer true. This is a SCOPE change,
  not a pass: the test was never executed and no `v1.0.0` tag exists. The note and
  scope_note above are preserved verbatim as the record of the two blocks. See
  `.planning/REQUIREMENTS.md`'s rewritten REL-02 and `.planning/ROADMAP.md` Phase 9 for
  current ownership.

### 2. Affected() BFS output-ordering determinism (SURF-04 backstop)
expected: |
  `codegraph affected` (and the engine Engine.Affected) returns results in a stable,
  deterministic order across repeated runs on the same index (no map-iteration-order
  nondeterminism). Assert by running `affected` twice on the same input and diffing the
  output, or add a determinism test. The BFS logic (depth bound, test-leaf pruning,
  dangling-edge skip) is already tested green — only cross-run ordering stability is
  unasserted.
result: pass
source: automated
evidence: |
  Closed by the code-review fix loop, which took the "add a determinism test" branch
  of this item's own expected-result text. internal/query/traverse.go now applies
  sortLocations to all four traversals — Impact/Callers/Affected (WR-05, d3f077c) and
  Callees (WR-02, 4feb6ff), the latter sorting *before* the limit/MaxLimit truncation
  so the cap selects a stable prefix rather than an arbitrary one.

  Asserted by TestImpactCallersAffectedDeterministicAcrossRepeatedCalls
  (internal/query/traverse_test.go:596), whose Affected subtest calls
  engine.Affected([]string{"pkga/target.go"}, 2) six times and requires byte-identical
  MarshalAffectedJSON output across every call. Verified green on 2026-07-26 at commit
  387cb4b: `go test ./internal/query/ -run TestImpactCallersAffectedDeterministic... -v`
  → all four subtests PASS.

  Scope caveat: this is a backstop assertion, not proof the sort is load-bearing — it
  would also pass if the underlying Pebble scan order were already stable. What the
  item asked for (cross-run ordering stability is asserted rather than assumed) is now
  satisfied. The companion TestCalleesSortedDeterministically *is* proven non-vacuous:
  removing sortLocations fails it with got "Zeta", want "Alpha".

## Readiness Recheck

Re-verified 2026-07-27 against the 5 corroborating gaps recorded under Test 1 on
2026-07-26. Four have closed; one is unremediated but not currently firing.

| # | Gap recorded 2026-07-26 | Status 2026-07-27 | Evidence |
|---|--------------------------|-------------------|----------|
| 1 | No 08-SECURITY.md | **Closed** | `08-SECURITY.md` status verified, 22/22 threats closed, `threats_open: 0`, 3 accepted risks (commit 80c452c) |
| 2 | ROADMAP lists Phase 7 AND 8 unchecked | **Partly closed** | Phase 7 now `[x]` (completed 2026-07-26). Phase 8 still `[ ]` — that is *this* UAT's own gate, correct to remain open |
| 3 | 07- and 08-VERIFICATION both `human_needed` | **Partly closed** | 07 canonicalized to `passed` with AG-07-01 recorded under Acknowledged Gaps. 08 still `human_needed` — again, this UAT's own gate |
| 4 | Four flaky daemon timing tests never de-flaked | **Open (not firing)** | `go test ./internal/daemon/...` → `ok 7.694s` on 2026-07-27. No de-flaking work was done; a single green run under light load is not proof the load-sensitivity is gone |
| 5 | Phase 7 Windows runtime test never executed | **Accepted, not fixed** | Formally recorded as AG-07-01 in 07-VERIFICATION.md — accepted as a platform gap, explicitly *not* claimed as verified. Windows remains cross-vet-only |

Not previously recorded, still true at recheck time: no `v1.0.0` tag exists (newest
tags: `v0.1.0`, `v0.0.0-rc.3`), and the branch is 465 commits ahead of `main`.

Also relevant to a 1.0 judgment: `08-VALIDATION.md` is `status: validated` but
`nyquist_compliant: false` — 7/9 requirements automated, REL-02 and REL-03 manual-only,
and one automatable gap (MO-08-03, pinning BENCHMARKS.md to its raw run JSONs)
deliberately deferred.

## Summary

total: 2
in_scope_total: 1
passed: 1
issues: 0
pending: 0
skipped: 0
blocked: 0
out_of_scope: 1

## Gaps
