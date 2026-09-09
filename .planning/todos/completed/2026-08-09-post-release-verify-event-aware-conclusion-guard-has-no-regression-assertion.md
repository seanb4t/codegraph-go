---
created: 2026-08-09T00:00:00.000Z
title: post-release-verify.yml's event-aware conclusion guard has no test asserting it, so a regression would be silent
area: ci
resolves_phase: 7
severity: high
status: resolved
resolved_at: 2026-09-08T00:00:00.000Z
resolved_by_phase: 7
files:

  - .github/workflows/post-release-verify.yml:303
  - .github/workflows/post-release-verify.yml:408
  - internal/upgrade/release_workflow_shape_test.go:1369

threat_ref: T-02-18
audit_acknowledged:
  milestone: v0.11.0
  at: 2026-08-17
---

## Problem

Every job in `post-release-verify.yml` carries the event-aware conclusion guard:

```yaml
if: github.event_name != 'workflow_run' || github.event.workflow_run.conclusion == 'success'
```

The disjunct is load-bearing. Under a `workflow_dispatch` (the retained
historical re-verification trigger), the upstream run object is null, so the
bare form — `github.event.workflow_run.conclusion == 'success'` — evaluates
false and **every job skips while the workflow reports green**: a suite that
verifies nothing and says so nowhere.

`T-02-18` (`02-06-PLAN.md`) registered three mitigation legs. Two shipped:

- The verbatim disjunct is present on both new jobs
  (`post-release-verify.yml:303`, `:408`) and in fact on all five jobs (5/5).

- The dry evaluation under both trigger events is recorded in
  `02-EVIDENCE.md:806` (`workflow_run`, run 31338004416) and `:841-855`
  (`workflow_dispatch`, run 31338409898 — all seven jobs ran, none skipped).

The third leg — **a count assertion over the file** — was never written.
Confirmed 2026-08-09 during `/gsd-secure-phase 02`:
`rg 'workflow_run|conclusion|event_name' --glob '*_test.go'` returns **zero
hits repo-wide**. `TestPostReleaseJobsDeclareCheckoutPolicy`
(`internal/upgrade/release_workflow_shape_test.go:1369`) enumerates the same
file's jobs but asserts only checkout policy.

## Risk shape

**Regression, not present exposure.** The guard is correct today and has been
empirically proven under both trigger events on real runs. What is missing is
the control that keeps it correct: nothing fails if a future edit drops the
`github.event_name != 'workflow_run' ||` disjunct, and the resulting breakage
is silent by construction — a green workflow whose jobs all skipped.

Accepted as a documented risk during `/gsd-secure-phase 02` (2026-08-09,
maintainer decision) on that basis. This todo is the follow-up artifact for
that acceptance.

## Solution

Extend `TestPostReleaseJobsDeclareCheckoutPolicy`'s existing enumeration
scaffolding — it already parses `post-release-verify.yml`'s job map, so the
addition is small:

- Assert **every** job id in the file carries the disjunct verbatim, not a
  minimum count. A bare count (`>= 2`) would pass while a newly added job
  silently lacks the guard; set-equality over job ids is the correct shape,
  consistent with `TestNotarizeMacosIdsCoverDarwinBuildIDs`'s exact-set
  discipline elsewhere in this package.

- Include a non-vacuity companion, matching the pattern
  `TestAppleSecretsScopedToSingleReleaseJob_EmptyDocIsError`
  (`release_workflow_shape_test.go:1289`) already establishes: an empty or
  unparseable document must be an error, never a pass.

Prove it RED-then-GREEN: strip the disjunct from one job, observe the test
fail naming that job, restore, observe green.

## Resolution (2026-09-08, Phase 7, 07-04-PLAN.md, D-08)

Landed as a sibling test, `TestPostReleaseJobsDeclareConclusionGuard`
(`internal/upgrade/release_workflow_shape_test.go`), rather than an
extension of `TestPostReleaseJobsDeclareCheckoutPolicy`, so a `-run` filter
can exercise the conclusion guard alone. A new `If` field (`yaml:"if"`) on
`fullWorkflowJob` exposes the parsed value; the new package-level const
`postReleaseConclusionGuard` holds the exact disjunct byte-for-byte. Every
job's `If` is compared against the constant with plain string equality — no
fixed expected-job-id list and no normalising helper, per this todo's own
"set-equality over job ids" framing, generalized to "no id list at all" so a
newly added unguarded job fails on its own rather than because a list went
stale. The inspected job count is logged and fataled on zero.
`TestPostReleaseJobsDeclareConclusionGuard_EmptyDocIsError` mirrors
`TestAppleSecretsScopedToSingleReleaseJob_EmptyDocIsError` exactly.

Proven RED-then-GREEN both ways this todo's risk shape names: the disjunct
stripped from the `gatekeeper` job (failure naming `gatekeeper` with an
empty value), and — going further than "strip it" — the conclusion
comparison inverted on the `resolve-tag` job while keeping an `if:` present
and plausible-looking (failure naming `resolve-tag` and quoting both the
value found and the value wanted), proving the assertion discriminates on
content, not merely on presence. Both mutations reverted byte-clean
(`git diff --quiet -- .github/workflows/post-release-verify.yml`). Full
transcripts in `07-MUTATION-LOG.md` family (d).
