---
created: 2026-08-09T00:00:00.000Z
title: verify:self-upgrade downloads and executes a prior release binary with no signature verification
area: release
severity: medium
files:
  - Taskfile.yml:1969-1982
  - .github/workflows/post-release-verify.yml (self-upgrade job)
---

## Problem

`verify:self-upgrade` (`Taskfile.yml:1969-1982`) downloads a PRIOR release's
raw binary asset, `chmod +x`s it (`Taskfile.yml:1973`), and then EXECUTES it
(`"${PRIOR_BIN}" upgrade "${TAG}"`, `Taskfile.yml:1982`) — with no cosign
signature verification anywhere between the download and the execution.

This is the SAME ordering hazard that T-02-19 (this phase's threat
register) fixes for the new `verify:notarized-suite` target (plan 02-06
Task 3): parallel independent CI jobs give no ordering guarantee, so an
unverified, network-fetched binary is made executable and run on the
runner regardless of whether a sibling supply-chain-verification job's own
check later fails.

`self-upgrade` (the post-release-verify.yml job that calls this target)
deliberately does NOT declare `needs: verify-supply-chain` — that
independence is itself a recorded, deliberate choice (a broken sidecar and
a broken self-upgrade should report as two separate, distinguishable
signals rather than one masking the other). That rationale is sound for
REPORTING independence. It says nothing about EXECUTION safety, and the
job's `PRIOR_BIN` is executed with no signature check regardless of what
`verify-supply-chain` finds.

## Why this is flagged now, not fixed now

T-02-19's corrected mitigation (plan 02-06 Task 3) establishes that this
project understands the download-then-execute ordering hazard and knows
how to close it (cosign verify-blob against the same issuer/identity flags,
BEFORE chmod). Leaving an identical, unremarked hazard three jobs away in
the same file would read as ignorance rather than a scoped decision.

This is PRE-EXISTING debt (the `self-upgrade` job predates this phase), not
a Phase 2 regression, and closing it here was out of scope: changing that
job's shape would touch a verification path (REL-08's self-upgrade proof)
this milestone depends on staying stable, and doing so was not part of
plan 02-06's task list.

## Solution

Add a cosign verify-blob step to `verify:self-upgrade`, mirroring
`verify:notarized-suite`'s pattern: download the prior release's raw binary
AND its `.sigstore.json` bundle (into separate temp dirs, per this
repository's own two-temp-dirs discipline), verify with the same
`--certificate-oidc-issuer`/`--certificate-identity-regexp` flags
`verify:release-assets` and `verify:notarized-suite` already use, and only
`chmod +x` after a successful verification. A failed verification must be a
hard failure before the binary is ever executed.

Prove the fix RED-then-GREEN per this repository's standing rule: a
tampered/unsigned prior-release binary must be observed failing the new
verification step before the fix, and the real (correctly signed) prior
release must still pass after it.
