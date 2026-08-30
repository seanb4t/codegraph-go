---
created: 2026-08-28T00:00:00.000Z
title: shadcn-svelte registry-version pinning with a source-match assertion
area: supply-chain
severity: minor
files:

  - web/components.json
  - web/src/lib/components/ui/
---

## Problem

**Vendored shadcn-svelte component source never enters `pnpm-lock.yaml`, so
`BLD-06`'s `task web:audit` gate structurally cannot see it.** Every other
generated or fetched artifact in this repo that can drift from its source
(`web/build/`, `web/src/lib/gen/ui_pb.ts`) is protected by an explicit drift
guard comparing a committed digest against a freshly regenerated one. The
`components/ui/` directory has no equivalent: the CLI is re-run by hand, at a
point in time, against whatever the registry serves that day, and nothing
proves a later `pnpm dlx shadcn-svelte@latest add <component>` produced the
same bytes as what is committed — or even ran against the same registry
schema version.

This is a **declined, not unrecognized**, gap:

- **Phase 2** (`02-CONTEXT.md`, Known Limitation) named it first, but shipped
  zero components, so there was nothing to pin yet.
- **Phase 3** (`03-CONTEXT.md` D-22) is the first phase to make the gap live —
  the `command` component plus its five registry-composed siblings (`dialog`,
  `button`, `input-group`, `input`, `textarea`) were vendored under
  `web/src/lib/components/ui/` and reviewed at add-time via a
  `gate="blocking-human"` checkpoint (03-06 Task 1), per the mitigation D-22
  already committed to: scope discipline plus human review at add time, not
  new machinery.
- Registry-version pinning with a source-match assertion — the same drift
  discipline `task web:drift` already applies to `web/build/` and
  `task proto:gen`/`task proto:drift` already applies to the generated
  TypeScript client — was **considered and declined** at both points as scope
  growth beyond what `BLD-06`'s text asks for. That ruling stands; this todo
  exists so the deferral stays visible rather than dissolving, per
  `03-CONTEXT.md`'s own instruction ("File a todo for it rather than growing
  this phase").

## Solution

Sketch, for whoever picks this up:

1. Record the exact `shadcn-svelte` registry version and `components.json`
   `registry` URL used at each `add` invocation (already done informally in
   each phase's SUMMARY.md — e.g. `03-06-SUMMARY.md` records `shadcn-svelte
   v1.5.1` for the `command` add).
2. Add a `task web:components:drift`-shaped target (naming to match the
   existing `web:build`/`web:drift`, `proto:gen`/`proto:drift` pairing
   convention) that re-runs the pinned-version CLI against each already-added
   component and diffs the result against what's committed under
   `web/src/lib/components/ui/`, failing on any byte difference.
3. Decide whether re-running `add` against a live network registry belongs in
   CI at all (the existing `web:drift`/`proto:drift` guards are self-contained
   — they diff a locally regenerated artifact, not one fetched over the
   network at CI time) — a network-dependent CI gate is a different risk
   profile than the existing drift guards and may need its own design pass
   rather than a literal copy of the pattern.
4. Extend `BLD-06`'s audit framing (or `SECURITY.md`) to state plainly that
   `pnpm audit`/Syft/SBOM coverage does NOT extend to vendored component
   source, and that this guard (once built) is the only mechanism that does.

Open question for whoever picks this up: whether a source-match assertion is
even meaningful for a registry that can legitimately change its output between
runs at the same pinned CLI version (e.g. upstream Bits UI patch releases) —
this may turn out to need a different mechanism than a literal digest compare.

## Resolution (2026-08-30, Phase 4, 04-07-PLAN.md)

Closed by adding `task web:components:drift` (`Taskfile.yml`), following
`proto:drift`'s regenerate-into-scratch-and-byte-compare shape rather than
`web:drift`'s manifest-hash shape (no marker file exists for vendored
component source). The determinism premise was proven live before the guard
was wired (04-07-SUMMARY.md Task 1): all 8 disk-derived component families
(`button`, `command`, `dialog`, `input`, `input-group`, `table`, `tabs`,
`textarea` — 50 files) regenerate byte-identically at pinned
`shadcn-svelte@1.5.1`, across two independent scratch runs, including the
six families 03-06 originally vendored via `@latest`.

The subject set is derived from `git ls-files -- 'web/src/lib/components/ui/'`
(no trailing slash — a trailing-slash pathspec returns zero files in this
repository and was the vacuous-pass shape rejected here) and its component
family list via `awk -F/ '{print $6}' | sort -u`. Both the file count and
component count print before any comparison (`compared 50 vendored component
files across 8 components`), guarded by small structural floors (8 files, 2
components) that never ratchet to today's observed count. The guard was
watched fail two independent ways: (1) a planted one-byte mutation in
`button/button.svelte` failed and named the exact file, then passed clean
after revert; (2) temporarily swapping to the trailing-slash pathspec made
the population floor reject with `compared 0 vendored component files
across 0 components` rather than a silent pass.

Wired into `.github/workflows/components-drift.yml` — `schedule` (weekly,
Monday 08:00 UTC) + `workflow_dispatch` only, deliberately never a
per-PR/per-push trigger and never added to `requiredCheckNames`
(`internal/upgrade/taskfile_shape_test.go`) or `release.yml` — so a live
registry fetch never becomes a merge blocker (D-16, T-04-28). The workflow's
bootstrap chain (`actions/checkout` → `./.github/actions/install-task` →
`actions/setup-node` at Node 24 → the run step) is registered in
`inScopeWorkflowFiles`/`inScopeJobs` and verified green:
`GOTOOLCHAIN=go1.26.5 go test ./internal/upgrade/... -run
'TestWorkflowRunBodiesInvokeTask|TestWorkflowFilePopulationMatchesDisk|TestInScopeJobsPopulationMatchesDisk'`.

See `.planning/phases/04-query-workbench-index-health/04-07-SUMMARY.md` for
the full probe evidence and both RED-proof transcripts.
