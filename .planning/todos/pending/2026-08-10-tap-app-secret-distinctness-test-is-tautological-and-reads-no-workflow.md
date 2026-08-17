---
created: 2026-08-10T00:00:00.000Z
title: tap App secret-distinctness test is tautological — it compares two in-test constants and reads no workflow
area: testing
severity: medium
files:

  - internal/upgrade/release_workflow_shape_test.go:1544-1553

threat_ref: AR-01 (03-SECURITY.md), T-03-07 leg (c)
audit_acknowledged:
  milestone: v0.11.0
  at: 2026-08-17
---

## Problem

`TestHomebrewTapAppSecretsDistinctFromReleasePleaseAppSecrets` is T-03-07's
leg (c) — "a test asserting the new secret names are not the release-please
App's". It is present, and it does not bind reality:

    releasePleaseAppSecretNames := []string{"APP_ID", "APP_PRIVATE_KEY"}
    for _, tapName := range homebrewTapCredentialNames[:2] {
        for _, rpName := range releasePleaseAppSecretNames {
            if tapName == rpName { t.Errorf(...) }

Both sides are **constants declared inside the test file**. It reads neither
`release.yml` nor `release-please.yml`. If either workflow changed its secret
names — the exact "consolidate the two Apps" edit the test's own doc comment says
it guards against — the test stays green. It can only fail if someone edits the
test file itself.

## Why it matters

ROADMAP criterion 5 asserts the tap-writing token can write `homebrew-tap` and
nothing else. D-16 makes "two distinct GitHub Apps" the structural basis for
that. This test is the only standing (non-one-time) assertion of that
distinctness — T-03-07 leg (d)'s `403` refusal is proven once by design (AR-02).

So the one guard meant to survive over time is the one that cannot fail.

This is a sibling of UF-4 (fixed 2026-08-10): same test file, same class of
defect — an assertion whose subject is not the artifact it claims to guard.

## Fix shape

Decode both workflows and compare the **actually referenced** secret names:

1. Read `release.yml`, collect the secret names its tap-mint step references.
2. Read `release-please.yml`, collect the secret names its token step references.
3. Assert the two sets are disjoint.
4. Add a positive floor to each: both sets must be non-empty, or the disjointness
   assertion passes vacuously (rule `84d1gfpywd`).

Prove RED by pointing `release.yml`'s mint at `APP_ID` / `APP_PRIVATE_KEY`.
