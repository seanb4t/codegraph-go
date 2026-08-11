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

## Resolution (04-05, honest about what is and is not proved)

`verify:self-upgrade` now downloads the prior release's `.sigstore.json`
bundle into its own `SIG_DIR` (mirroring `verify:notarized-suite`'s
two-temp-dirs discipline) and runs `cosign verify-blob` — same issuer and
`--certificate-identity-regexp` flags `verify:release-assets` and
`verify:notarized-suite` already use — strictly before `chmod +x`. A
region-scoped ordering assertion (byte offset of `cosign verify-blob`
within `verify:self-upgrade` is strictly less than the offset of
`chmod +x`) proves that structurally.

A new drift guard,
`TestCosignIdentityPolicyBoundaryParityWithCompiledPattern`, proves all
seven identity-regexp restatements across five files (`Taskfile.yml` ×3,
`README.md`, `docs/RELEASE.md`, `SECURITY.md`,
`docs/RELEASE-PROCEDURES.md`) exhibit selected boundary-case behavioural
parity with the compiled `releaseWorkflowRefPattern`, including a
region-scoped requirement pinning that at least one literal lives inside
`verify:self-upgrade` itself. This guard was demonstrated RED three times
against confirmed-applied mutations — a semantic loosening (branch refs
accepted), an emptied file list (the total floor), and a total-preserving
relocation of the new literal out of `verify:self-upgrade` into
`verify:gatekeeper` (the region-scoped check, which no count-based floor
can detect) — and reverted byte-clean after each.

**What is NOT proved, stated plainly rather than papered over:** the
end-to-end RED-then-GREEN this todo originally asked for — a tampered or
unsigned prior-release binary observed actually failing
`cosign verify-blob` inside a real `verify:self-upgrade` run — requires a
real published release and network access, and was not executed in this
session. The cosign step is wired and its ordering is asserted
structurally; the drift guard is demonstrated RED against synthetic
mutations, not against a real signature failure. The next natural
`post-release-verify.yml` run on a real tag is what first exercises this
path end to end; a broken bundle download or a mismatched identity would
surface there.
