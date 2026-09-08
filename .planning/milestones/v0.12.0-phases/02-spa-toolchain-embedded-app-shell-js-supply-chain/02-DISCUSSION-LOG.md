# Phase 2: SPA Toolchain, Embedded App Shell & JS Supply Chain - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-08-23
**Phase:** 2-SPA Toolchain, Embedded App Shell & JS Supply Chain
**Areas discussed:** SPA tree & dist location, TypeScript client generation, SPA fallback discrimination, JS gate placement & approval state, Design-system scope, App-shell content
**Mode:** default interactive; `workflow.research_before_questions=true` (each area researched before questions)

---

## Todo Cross-Reference

| Option | Description | Selected |
|--------|-------------|----------|
| Fold none — review only | Record all six as reviewed-but-not-folded | ✓ |
| dry-run-signed vacuous guard | Adjacent via BLD-07's structural check over `.goreleaser.yaml` | |
| post-release-verify assertion | Folded into Phase 1, not implemented, no backing requirement | |
| golangci-lint | Folded into Phase 1, not implemented; `.golangci.yml` absent from tree | ✓ |

**User's choice:** golangci-lint **and** "Fold none — review only" (both selected).
**Notes:** Contradictory as literally stated. Read as *fold golangci-lint, review the
other five without folding* — the only reading under which both selections carry
meaning. Interpretation recorded explicitly in CONTEXT.md §Folded Todos rather than
resolved silently.

---

## SPA tree & dist location

**Research surfaced first:** `adapter-static` defaults to `pages: 'build'` /
`assets: 'build'` (not `dist/`); SvelteKit recommends against `index.html` as the SPA
fallback; the roadmap's `_app`-prefix note identifies SvelteKit rather than plain Vite.

### Framework shape

| Option | Description | Selected |
|--------|-------------|----------|
| SvelteKit + adapter-static | `_app/` is SvelteKit's asset dir — makes the roadmap's `all:` note true; file-based routing for Phases 3–6 | ✓ |
| Plain Svelte + Vite | Lighter, no Kit runtime; emits `dist/assets/` with no underscore, making the `all:` note moot | |
| You decide | | |

### Source tree location

| Option | Description | Selected |
|--------|-------------|----------|
| `web/` at repo root | Conventional; keeps JS visibly separate; makes BLD-07 path-scoped | ✓ |
| `internal/uiserver/webui/` | Mirrors `internal/mcp/resources.go`; buries a pnpm workspace under `internal/` | |
| `ui/` at repo root | Matches the `codegraph ui` command name | |

### Committed output directory

| Option | Description | Selected |
|--------|-------------|----------|
| Nested `dist/`, narrow the ignore | `.gitignore:4` is `/dist/` — root-anchored, so `web/dist/` needs no negation | |
| Rename to `build/` (adapter default) | Adapter's own default; zero output-path config; deviates from the roadmap's `all:dist` note | ✓ |
| Distinct name, e.g. `web/assets-dist/` | Names the collision out of existence; nonstandard for both toolchains | |

**Notes:** Verified after selection that `web/build/` is likewise not ignored —
`git check-ignore -v web/build` and `.../index.html` both report not-ignored,
positive-controlled against `dist/artifacts.json` → `.gitignore:4`. So **no
`.gitignore` change is needed** under either of the top two options.

### SPA fallback filename

| Option | Description | Selected |
|--------|-------------|----------|
| `index.html` — keep the roadmap's word | SvelteKit's warning targets static hosts and prerendering; neither applies to our own Go mux | ✓ |
| `200.html` — follow the framework's advice | Sidesteps a future prerender collision; makes criterion 2's wording inaccurate | |
| You decide | | |

---

## TypeScript client generation

**Research surfaced first:** Connect-ES v2 **removed** `protoc-gen-connect-es` —
`protoc-gen-es` alone emits service definitions, used with `createClient(Service, transport)`.
Plugins may be `local:` (needs Node) or `remote:` (needs network).

### Codegen mechanism

| Option | Description | Selected |
|--------|-------------|----------|
| buf `local: protoc-gen-es` | One buf.gen.yaml, both languages regenerate together; pnpm lockfile pins the generator; adds a Node dep to `task proto:gen` | ✓ |
| buf `remote: buf.build/bufbuild/es` | Keeps Node out of codegen; needs network and trusts a hosted build service | |
| Hand-written fetch client | No generator; proto and client can silently diverge | |

### Commit the generated TS?

| Option | Description | Selected |
|--------|-------------|----------|
| Committed + drift-guarded | Matches the Go side and `ROADMAP.md:98`'s stated pattern | ✓ |
| Generated at build time, gitignored | Nothing to drift; pulls buf into the JS build for BLD-03 | |
| You decide | | |

### Drift-check placement

| Option | Description | Selected |
|--------|-------------|----------|
| Extend `proto:drift`, raise the floor | One guard, one definition of currency; gains a Node/pnpm precondition that must fail loudly | ✓ |
| Separate task, e.g. `web:proto-drift` | Keeps `proto:drift` Go-pure; two floors to keep correct | |
| You decide | | |

**Notes:** Enumeration afterwards found the trap recorded in CONTEXT.md D-07 — the buf
module holds **two** protos and `buf.gen.yaml` applies plugins module-wide, so an
unscoped `protoc-gen-es` would also emit TS for `internal/schema/graph.proto`. Floor
becomes 4 (scoped) or 5 (unscoped); per rule `84d1gfpywd` it must be the correct
number, not merely a higher one.

### Wire encoding

| Option | Description | Selected |
|--------|-------------|----------|
| JSON | Keeps devtools/`curl` reproducibility Phase 1 verified with; size is free on loopback | ✓ |
| Binary protobuf | Smaller and faster to parse; opaque in devtools | |
| You decide | | |

---

## SPA fallback discrimination

**Research surfaced first:** Phase 1 mounts Connect via `mux.Handle(NewUIServiceHandler(...))`,
whose generated constructor returns a `(path, handler)` pair; the repo is on Go 1.26,
where `ServeMux` resolves most-specific-pattern-wins.

### Discrimination mechanism

| Option | Description | Selected |
|--------|-------------|----------|
| ServeMux precedence | SPA at `"/"`; stdlib resolves precedence; nothing to keep in sync when Phase 6 adds a method | ✓ |
| Explicit prefix allowlist | More obvious read; hand-maintained list needing its own non-vacuity guard | |
| You decide | | |

### Behaviour on an asset miss

| Option | Description | Selected |
|--------|-------------|----------|
| 404 for asset prefixes, fallback elsewhere | Criterion 2's teeth; avoids a MIME/module error pointing nowhere near the cause | ✓ |
| 404 for anything with a file extension | No prefix list; heuristic — Phase 3's dotted deep links would wrongly 404 | |
| Always fall back to `index.html` | Explicitly rejected by criterion 2 | |

### Cache headers

| Option | Description | Selected |
|--------|-------------|----------|
| Immutable assets long-lived, `index.html` no-store | Prevents the silent post-upgrade stale-shell failure | ✓ |
| No caching anywhere | Trivially correct, essentially free on loopback | |
| You decide | | |

### Handler placement

| Option | Description | Selected |
|--------|-------------|----------|
| New file in `internal/uiserver` | Matches `degrade.go`/`truncate.go`/`originguard.go` layout | ✓ |
| Own package, e.g. `internal/webui` | Reusable by a future `codegraph serve`; a package boundary for one handler | |
| You decide | | |

---

## JS gate placement & approval state

**Research surfaced first:** pnpm's `strictDepBuilds` makes install exit non-zero on
unreviewed build scripts; `allowBuilds` in `pnpm-workspace.yaml` replaced
`onlyBuiltDependencies`, and `pnpm approve-builds` records denials as `false`; pnpm
auto-enables frozen-lockfile in CI and fails on a newer-major lockfile. Separately,
`internal/upgrade/taskfile_shape_test.go:43` carries the literal required-check fixture,
and ruleset membership lives outside the repo.

### CI job placement

| Option | Description | Selected |
|--------|-------------|----------|
| New named job + ruleset edit | Separates Go and JS concerns; needs a `gh api` ruleset edit or it cannot block merge | |
| Fold into existing `test` job | Inherits required status immediately; every Go run pays a pnpm install | ✓ |
| You decide | | |

### BLD-05 mechanism

| Option | Description | Selected |
|--------|-------------|----------|
| `strictDepBuilds` + a positive assertion | pnpm itself exits non-zero; assertion covers silent loss of the setting | ✓ |
| Grep the "Ignored build scripts" warning | The scoping-time approach; couples the gate to an upstream string | |
| Both | Belt and braces; two things to maintain | |

### BLD-06 sibling assertion

| Option | Description | Selected |
|--------|-------------|----------|
| Scanned-package count from the lockfile | Proves the scan had real input, never reading `pnpm audit`'s exit code | ✓ |
| Deliberately-vulnerable fixture, like `vuln:selftest` | Proves the detector fires; the fixture rots when the advisory ages out | |
| Both | The Go side ships both shapes | |

**Notes:** The fixture option was flagged in CONTEXT.md as worth revisiting if a
low-maintenance advisory pin exists — not simply discarded.

### pnpm version pinning

| Option | Description | Selected |
|--------|-------------|----------|
| `packageManager` via Corepack | One source of truth; CI skew surfaces loudly on its own | ✓ |
| `packageManager` + `pnpm/action-setup` | Explicit in CI logs plus caching; two versions that can disagree | |
| You decide | | |

---

## Design-system scope *(surfaced during discussion, not in the original area list)*

**Research surfaced first:** `shadcn-svelte init` targets **existing** projects;
current version requires Svelte 5 + Tailwind v4; Tailwind v4 wires in via
`@tailwindcss/vite`, not PostCSS. Components are **vendored source** fetched from a
registry — invisible to `pnpm audit`.

| Option | Description | Selected |
|--------|-------------|----------|
| Phase 2 — shell includes the design system | Phase 3 opens with `add <component>` only; drift guard proven against real CSS | ✓ |
| Phase 3 — defer to when views need it | Keeps Phase 2 strictly toolchain; `init` supports existing projects so it is not rework | |
| You decide | | |

### Vendored-component audit gap

| Option | Description | Selected |
|--------|-------------|----------|
| Record as a named limitation | Names the boundary so Phase 3 inherits it rather than discovering it | ✓ |
| Pin the registry + assert provenance | Strongest coverage; real scope growth BLD-06 does not ask for | |
| Out of scope — Phase 3's problem | Leaves an unrecorded gap in the supply-chain phase | |

---

## App-shell content *(surfaced during discussion)*

### What `/` renders

| Option | Description | Selected |
|--------|-------------|----------|
| Layout + nav skeleton for the four views | Phases 3–6 fill slots; gives criterion 2 a second route to test against | ✓ |
| Single page proving the pipeline | Smallest surface; leaves criterion 2 with no real route to exercise | |
| You decide | | |

### Live RPC in the shell

| Option | Description | Selected |
|--------|-------------|----------|
| Yes — call `GetStatus` and render it | Proves embedded asset → TS client → Connect JSON → Phase 1 handler end-to-end | ✓ |
| No — static shell only | Smaller blast radius; ships a generated client never once executed | |
| You decide | | |

---

## Claude's Discretion

Recorded in CONTEXT.md §Claude's Discretion. No area was answered "You decide" —
discretion items are the residual mechanics under decided choices:
`adapter-static` options beyond `fallback`, exact `Cache-Control` values, where the
`go:embed` directive physically sits, the buf mechanism used to scope
`protoc-gen-es` to `ui.proto`, and navigation placeholder styling.

## Deferred Ideas

- Registry pinning + provenance assertion for vendored shadcn-svelte components
- A `vuln:selftest`-shaped red proof for `pnpm audit`
- `200.html` as the SPA fallback (revisit only if prerendering is introduced)
- A dedicated `web` CI job with matching ruleset + fixture edits
