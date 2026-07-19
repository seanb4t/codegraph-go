---
status: testing
phase: 08-surface-reconciliation-signed-v1-0-0-release
source: [08-VERIFICATION.md]
started: 2026-07-19T21:30:00Z
updated: 2026-07-19T21:30:00Z
---

## Current Test

number: 1
name: Cut the signed v1.0.0 release (maintainer-manual — REL-02)
expected: |
  Following docs/RELEASE-PROCEDURES.md: the pre-tag 6-target `go list -mod=readonly ./...`
  gate passes, the integration branch is squash-merged to main, `v1.0.0` is tagged and
  pushed, release.yml publishes a signed release (per-binary cosign keyless + SLSA3
  provenance + syft SBOM), and post-release `cosign verify-blob` + `slsa-verifier
  verify-artifact` succeed. Closes v0.1's pending DIST-02.
awaiting: user response

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
result: [pending]

### 2. Affected() BFS output-ordering determinism (SURF-04 backstop)
expected: |
  `codegraph affected` (and the engine Engine.Affected) returns results in a stable,
  deterministic order across repeated runs on the same index (no map-iteration-order
  nondeterminism). Assert by running `affected` twice on the same input and diffing the
  output, or add a determinism test. The BFS logic (depth bound, test-leaf pruning,
  dangling-edge skip) is already tested green — only cross-run ordering stability is
  unasserted.
result: [pending]

## Summary

total: 2
passed: 0
issues: 0
pending: 2
skipped: 0
blocked: 0

## Gaps
