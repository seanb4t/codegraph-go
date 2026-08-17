---
created: 2026-08-09T00:00:00.000Z
title: release:dry-run-signed's additions-only diff guard passes vacuously when the awk anchor stops matching
area: release
severity: medium
files:

  - Taskfile.yml:586-605
  - Taskfile.yml:1085-1100

threat_ref: T-02-08
audit_acknowledged:
  milestone: v0.11.0
  at: 2026-08-17
---

## Problem

`release:dry-run-signed` generates a copy of the committed `.goreleaser.yaml`
with `--key=` injected into the `signs:` pipe's `sign-blob` args, using an awk
anchor on the line `      - "sign-blob"`. It then runs an "additions-only" diff
guard (`Taskfile.yml:586-605`) that asserts the generated config differs from
the committed one by additions only — no removals, no modifications.

That guard is **negative-only**. If the awk anchor stops matching — because the
`sign-blob` args block is re-indented, renamed, or restructured — the injection
becomes a no-op, the generated config equals the committed one, and the diff
produces zero added lines. Zero additions contains zero *bad* additions, so the
guard reports success. The guard is silent on precisely the condition it was
written to catch.

The downstream failure is the one `T-02-08` registered in `02-02-PLAN.md`: with
no `--key=` injected, the rehearsal falls through to a keyless OIDC call with no
token, and **the failure presents as a hang, not an error**.

This is a member of this repository's recurring failure family — a check whose
failure is indistinguishable from its success. Four instances were recorded
during phase-02 plan convergence (memory `vep9bdqkw9`); this is a fifth, and it
was sitting inside the mitigation for a threat about that very family.

## Current state is benign

Verified 2026-08-09 during `/gsd-secure-phase 02`: the anchor
`/^      - "sign-blob"$/` matches exactly once against both the worktree and the
`HEAD` copy of `.goreleaser.yaml`. Nothing is currently mis-injected. This is a
latent unsoundness in the guard, not a live defect in the artifact.

## Why this is flagged now, not fixed now

`T-02-08`'s registered mitigation was an acceptance criterion comparing the args
line's indentation against `HEAD`. That criterion exists only as plan text
(`02-02-PLAN.md:294`) — it was never executed and left no durable control
(`rg 'sign-blob|indent'` over all seven phase-02 SUMMARY files and
`02-EVIDENCE.md` → 0 hits; the only `*_test.go` hit,
`internal/upgrade/goreleaser_shape_test.go:660`, asserts the args *list
contents*, never its indentation).

Closing it is implementation work outside `/gsd-secure-phase`'s remit, and the
better fix is not the one the register named — see below.

## Solution

Prefer a **positive** assertion over the indentation comparison the register
originally specified. Indentation is a proxy for the property that actually
matters; assert the property directly:

- In `release:dry-run-signed` (and the same anchor's second user,
  `release:rehearse-notarize` at `Taskfile.yml:1085-1100`), assert that the
  injection added **at least one** `--key=` line, and hard-fail if it added
  zero. That makes a non-matching anchor a loud error at the point of
  injection rather than a silent no-op discovered later as a hang.

- Optionally retain the indentation-vs-`HEAD` comparison as a second, cheaper
  signal, but it is redundant once the positive assertion exists.

Prove the fix RED-then-GREEN per this repository's standing rule: deliberately
perturb the anchor (re-indent the `sign-blob` args block in a generated copy),
observe the new assertion fail loudly, then restore and observe it pass.
