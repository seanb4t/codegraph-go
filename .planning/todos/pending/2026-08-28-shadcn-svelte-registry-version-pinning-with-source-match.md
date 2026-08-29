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
