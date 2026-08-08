# Deferred Items

Out-of-scope discoveries logged during plan execution, per the executor's scope-boundary rule
("only auto-fix issues directly caused by the current task's changes; log out-of-scope
discoveries here rather than fixing them").

## 01-03: docs/RELEASE-PROCEDURES.md describes the pre-collapse build/assemble topology

**Found during:** 01-03 Task 2 (collapse release.yml to one job)

**Issue:** `docs/RELEASE-PROCEDURES.md` repeatedly references the now-deleted `build`/`assemble`
job split (e.g. "fires the full signed build/assemble/provenance pipeline", "`assemble`'s
publish step finds the Release release-please already", "`assemble`'s `Publish GitHub release`
step"). After plan 01-03's collapse, these references describe a topology that no longer exists.

**Why deferred:** `docs/RELEASE-PROCEDURES.md` is outside this worktree's assigned file scope for
this wave (`.github/workflows/release.yml`, `Taskfile.yml`, `CONTRIBUTING.md`,
`internal/upgrade/release_workflow_shape_test.go`). It also documents the full end-to-end release
runbook, which depends on the whole phase (plans 01-02 through 01-06) landing before it can be
rewritten accurately — rewriting it now, mid-phase, from only this plan's perspective would risk
describing an intermediate state rather than the phase's final one.

**Recommended follow-up:** A later plan in this phase (or a phase-close docs pass) should rewrite
`docs/RELEASE-PROCEDURES.md`'s job-topology references to describe the single `release` job and
`.goreleaser.yaml`'s declarative pipes, once the full migration (through plan 01-04's provenance
job removal) has landed.
