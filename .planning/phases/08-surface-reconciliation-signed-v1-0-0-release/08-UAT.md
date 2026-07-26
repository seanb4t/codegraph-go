---
status: partial
phase: 08-surface-reconciliation-signed-v1-0-0-release
source: [08-VERIFICATION.md]
started: 2026-07-19T21:30:00Z
updated: 2026-07-26T00:00:00Z
---

## Current Test

[testing paused — 1 item outstanding: REL-02 release go-ahead withheld]

Test 1 (REL-02, the signed v1.0.0 release) is blocked on a maintainer readiness
decision, not on a defect. Re-run `/gsd-verify-work 8` to resume once the
readiness gaps recorded under Test 1 are closed.

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
result: blocked
blocked_by: other
reason: "I don't think we're ready to declare 1.0"
note: |
  Maintainer withheld the go-ahead — a readiness judgment, not a defect, so this is
  recorded as blocked rather than an issue and produces no gap entry or fix plan.
  No v1.0.0 tag was created or pushed.

  Corroborating gaps found during this UAT run (2026-07-26, commit 387cb4b):
    - Phase 8 has NO 08-SECURITY.md — phases 01-07 all have one; /gsd-secure-phase 8
      was never run. This workflow's own verify:post gate blocks phase advancement
      on exactly that artifact.
    - ROADMAP.md still lists BOTH Phase 7 and Phase 8 as unchecked "[ ]".
    - 07-VERIFICATION.md and 08-VERIFICATION.md are both status: human_needed.
    - internal/daemon carries four known-flaky, load-sensitive timing tests that
      have never been de-flaked.
    - Phase 7 UAT test 3 (Windows runtime) was SKIPPED for lack of a Windows host —
      Windows is covered only by cross-vet, never executed.

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

## Summary

total: 2
passed: 1
issues: 0
pending: 0
skipped: 0
blocked: 1

## Gaps
