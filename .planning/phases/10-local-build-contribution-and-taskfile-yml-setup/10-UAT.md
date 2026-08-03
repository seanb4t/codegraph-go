---
status: testing
phase: 10-local-build-contribution-and-taskfile-yml-setup
source: [10-VERIFICATION.md]
started: 2026-08-02T22:00:00Z
updated: 2026-08-02T22:00:00Z
---

## Current Test

number: 1
name: release.yml darwin legs — goreleaser build, cosign signing, and SLSA attestation on namespace-profile-macos-6x14-tahoe
expected: |
  The `build` job's goreleaser step produces both signed darwin binaries with release
  ldflags, the `assemble`/`provenance` steps attest them correctly, and (ideally) a
  `codegraph upgrade` smoke test succeeds against the resulting macOS binary on a real Mac.
awaiting: user response

## Tests

### 1. release.yml darwin legs — goreleaser build, cosign signing, and SLSA attestation on namespace-profile-macos-6x14-tahoe

expected: Push a real `v[0-9]*` tag and watch `release.yml`'s `build` job end-to-end for the two darwin matrix legs (goos=darwin, goarch=arm64/amd64), specifically the goreleaser invocation, cosign signing, and SLSA attestation steps that run on top of the plain `go build` the canary already proves. The goreleaser step should produce both signed darwin binaries with release ldflags, the `assemble`/`provenance` steps should attest them correctly, and a `codegraph upgrade` smoke test should succeed against the resulting macOS binary on a real Mac.
result: [pending]

## Summary

total: 1
passed: 0
issues: 0
pending: 1
skipped: 0
blocked: 0

## Gaps
