---
phase: 02-spa-toolchain-embedded-app-shell-js-supply-chain
verified: 2026-08-25T01:30:00Z
status: passed
score: 5/5 roadmap success criteria verified (35/35 plan-level must_haves truths checked, 0 failed)
behavior_unverified: 0
overrides_applied: 0
re_verification: null
---

# Phase 2: SPA Toolchain, Embedded App Shell & JS Supply Chain Verification Report

**Phase Goal:** The browser gets a real, pnpm-built Svelte app served from inside the binary —
committed, embedded, drift-guarded, and covered by a JS vulnerability gate the Go tooling cannot
see — with no Node or pnpm invocation reachable anywhere in the signed release path.

**Verified:** 2026-08-25T01:30:00Z
**Status:** passed
**Re-verification:** No — initial verification

## Method

This verification did not trust SUMMARY.md claims. For every claim that could be executed, it
was executed independently in this session: the binary was rebuilt from source
(`GOTOOLCHAIN=go1.26.5 go build -o /tmp/codegraph-verify ./cmd/codegraph`), `codegraph ui` was
launched twice — once against this repo (real index) and once against an empty scratch
directory (no index) — with `env -i PATH=/usr/bin:/bin` (no `node`, `pnpm`, `npx`, `npm`, or
Corepack reachable), and a real Chromium browser (`agent-browser`) was driven against both
servers: screenshots taken, the JS console read, and the network log inspected to prove
client-side routing (one `Document` request, all subsequent navigation via `_app/immutable/`
chunk fetches). All Taskfile gates (`web:deps`, `web:deps:strict`, `web:lockfile`, `web:audit`,
`web:build:verify`, `web:drift`, `proto:drift`) were re-run at HEAD (`6150f94b`), not merely read
from a SUMMARY transcript. The two `<human-check>` blocks the planner deferred to end-of-phase
UAT (02-01 Task 1, 02-05 Task 3) were both executed directly with photographic/console evidence
rather than left open — see "Human Verification" below.

## Goal Achievement

### Observable Truths (ROADMAP Success Criteria)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | A developer with only Go on PATH builds the binary, runs `codegraph ui`, and gets the real app in the browser from embedded assets — including underscore-prefixed chunks — with no JS toolchain installed (BLD-02) | ✓ VERIFIED | Built with `go build`; ran with `env -i PATH=/usr/bin:/bin` (no node/pnpm/npx/npm/corepack on PATH); `GET /` → 200, real `index.html` (1230 bytes), referencing 8 `_app/immutable/` paths; fetched `_app/immutable/entry/start.CXTtfNDH.js` → 200, 77 bytes, served from the embedded FS. `web/embed.go` carries `//go:embed all:build` (the `all:` prefix present — its absence is exactly what would silently drop `_app/`). `git ls-files web/build` = 26 tracked files, 23 under `_app/`; `git check-ignore -v web/build` and `web/build/index.html` both report NOT IGNORED. Live in a real Chromium browser: status panel renders real data (nodeCount 6064, edgeCount 14543, fileCount 500), zero console errors. |
| 2 | Opening a client-side route directly loads the app; RPC paths and hashed asset URLs resolve to their real handlers rather than falling back to `index.html` (RPC-03) | ✓ VERIFIED | `GET /browse` → 200, body byte-identical to `index.html` (fallback). `GET /_app/immutable/entry/doesnotexist.js` (missing hashed asset) → 404, never HTML. `POST /codegraph.ui.v1.UIService/GetStatus` → 200 with `Content-Type: application/json` and real protobuf-JSON data — resolved by `ServeMux` precedence (D-09), not a hand-maintained allowlist (confirmed reading `internal/uiserver/server.go:121-141`: `mux.Handle(uiv1connect.NewUIServiceHandler(...))` then `mux.Handle("/", newSPAHandler(...))`, both wrapped by `originHostGuard`). Live browser network log for a nav click showed exactly one `(Document)` request for the whole session — all subsequent route changes fetched only `_app/immutable/*` script chunks, proving true client-side routing, not a full reload. Cache headers observed: `_app/immutable/...` → `public, max-age=31536000, immutable`; `/` and `/browse` → `no-store` (D-11). Every response carried `X-Content-Type-Options: nosniff` and a same-origin CSP (`default-src 'self'; connect-src 'self'; ...; script-src 'self' 'sha256-...'`, no `unsafe-inline` in `script-src`, no `unsafe-eval` anywhere). Code review finding WR-01 (unanchored `src=` substring match in the CSP hash extractor) was fixed in commit `0d110682` — confirmed `spaExternalSrcAttrRE = regexp.MustCompile(`(?i)(^|\s)src\s*=`)` is in place at `internal/uiserver/spa.go:127` and is what the extractor now calls. |
| 3 | A clean checkout runs `pnpm install --frozen-lockfile` at the Corepack-pinned pnpm version and rebuilds a `web/build/` the drift guard finds matching; the guard reports both counts; it has been watched fail against a stale build and a hand-edited byte (BLD-01, BLD-03) | ✓ VERIFIED | Ran live: `task web:deps` → `pnpm install --frozen-lockfile succeeded, web/pnpm-lock.yaml unchanged` (host pnpm 11.23.0, no Corepack — took the self-managed fallback path `web:deps` reports by name, closing the gap 02-01-SUMMARY.md recorded and 02-06 durably fixed). `task web:build:verify` → built 25 files into a scratch dir, `index.html present and non-empty`, 14 non-empty immutable `.js`, 1 non-empty immutable `.css`, committed tree confirmed untouched (`git status --short` empty after). `task web:drift` → `hashed 22 source files`, `manifested 25 output files`, both counts printed before comparison, PASS. **Non-vacuity of the RED claim**: 02-06-SUMMARY.md's Verification Evidence section captures six distinct RED transcripts with real computed hash values and revert steps (stale source, tampered output byte, added output file, removed output file, narrowed source enumeration via a scratch Taskfile copy, narrowed output enumeration, missing marker) — each followed by a confirmed clean revert. This is concrete transcript evidence (specific hex digests, specific file paths, specific error text), not a restated claim. |
| 4 | CI fails loudly when an unapproved dependency lifecycle script appears, rather than skipping it and quietly producing a different `web/build/` (BLD-05) | ✓ VERIFIED | `task web:deps:strict` run live: `strictDepBuilds resolved to 'true'`, `allowBuilds entries: 0 (denials: 0)`, PASS. `web/pnpm-workspace.yaml` carries `strictDepBuilds: true` and a hand-written `allowBuilds: {}` (with an inline comment explaining why: `pnpm approve-builds` writes nothing when there is nothing pending, and the gate treats total key-absence as a distinct named failure from a committed empty map). The assertion is a positive value check on parsed YAML, not a grep of pnpm's "Ignored build scripts" warning text (D-14's explicit prohibition) — confirmed reading the `web:deps:strict` Taskfile target, which decodes YAML via `go.yaml.in/yaml/v3`. |
| 5 | `pnpm audit` runs as a named CI gate distinguishing "scanned and clean" from "the scan itself failed" — proven by a sibling assertion that does not read `pnpm audit`'s exit code — and no `pnpm`/`node`/`npx`/`npm` invocation is reachable anywhere in `.goreleaser.yaml` or `release.yml`, checked structurally (BLD-06, BLD-07) | ✓ VERIFIED | `task web:audit` run live → CLEAN, PASS. Read the Taskfile target directly: `web:audit` declares `deps: [web:lockfile]` — a Taskfile dependency that runs to completion, prints its own package/integrity counts, and returns *before* `pnpm audit` is invoked in `web:audit`'s own `cmds:`; `web:lockfile`'s code path never references `pnpm audit`'s exit code or output — genuine sibling independence, not a second read of the same signal. The classifier parses `pnpm audit --json`'s JSON shape (an `error` key vs. an `advisories` map vs. unparseable/empty output) rather than trusting exit code, because 02-07-SUMMARY.md documents empirically that a clean scan and a registry-unreachable scan both exit 1 — exit code alone cannot distinguish them. `rg -c "pnpm\|node\|npx\|npm"` over `.goreleaser.yaml` and `.github/workflows/release.yml` → 0 matches in both, positive-controlled by `rg -c "\bgo\b"` → 29 and 14 matches respectively (the search itself works). Structural (not text-search) proof: `go test ./internal/upgrade/... -run "TestReleasePathHasNoJSToolchain\|TestReleasePathScanIsNonVacuous\|TestReleasePathScanIgnoresNearMisses\|TestReleasePathClosureIsTransitive\|TestJSInstallPathHasNoMutableCache\|TestMutableCacheScanIsNonVacuous\|TestUnsupportedReachabilityEdgeIsLoud"` — all 7 PASS, including a positive-control planted-token finding (`forbidden command "node"`) proving the scanner can actually fire, not just report a trusted zero. `web/pnpm-workspace.yaml`'s `overrides: { cookie@<0.7.0: ^0.7.2 }` (fixing a real, live GHSA-pxg6-pf52-xh8x found during 02-07's own empirical audit run) is present and `task web:audit` confirms zero advisories on the current tree. `SECURITY.md` states both scanners' disjoint scope (`govulncheck` = reachable Go graph, `pnpm audit` = `pnpm-lock.yaml` tree, neither a superset) and names the one known gap (registry-vendored shadcn-svelte component source is invisible to both, not yet live — Phase 2 ships zero components). |

**Score:** 5/5 ROADMAP success criteria verified. 0 present-but-behavior-unverified. 0 overrides.

### Plan-Level Must-Haves (all 7 plans)

Every plan's `must_haves.truths`, `.artifacts`, `.key_links`, and `.prohibitions` were checked
against the codebase (not the SUMMARYs) as part of the criteria above. None failed. Notable
individual confirmations beyond what's cited in the table:

- **D-01–D-04** (SvelteKit + adapter-static, `web/` location, `web/build/` output, `index.html`
  fallback, never `200.html`): all confirmed in code (`spaFallbackFile = "index.html"` constant,
  `web/` tree layout, `web/embed.go`'s doc comment naming D-03's deviation from the ROADMAP's
  original `dist/` text).
- **D-05–D-08** (scoped `protoc-gen-es`, committed+drift-guarded TS client, `proto:drift` folded
  not duplicated, Connect JSON not binary): `task proto:drift` run live → `compared 4 generated
  files`, `all 4 generated files byte-identical`. The floor-4 literal (`Taskfile.yml:346`) and its
  paired regression test (`TestProtoDriftGuardReportsAComparedCount`, which the orchestrator's
  pre-verification pass recorded as fixed at commit `1d2a1d2f` after 02-04 raised the floor without
  updating this test) both pass at HEAD. `Content-Type: application/json` confirmed on the live
  `GetStatus` response.
- **D-09–D-12** (ServeMux precedence, three-way asset rule, two-class cache, handler file
  location): confirmed live (see criterion 2) and in `internal/uiserver/server.go:121-141`.
- **D-13–D-16** (fold into `test` job, `strictDepBuilds`+`allowBuilds`, lockfile-count sibling,
  Corepack pin with a reported self-managed fallback): confirmed live (see criteria 4-5) and in
  `.github/workflows/ci.yml` (all 5 JS gate steps inside the single `test` job — job-key count
  unchanged at 10 per 02-06-SUMMARY.md, independently spot-checked by grepping `ci.yml` for
  `^  [a-z].*:$` job headers).
- **D-17–D-19** (Tailwind v4 via `@tailwindcss/vite`, four named nav slots, live `GetStatus`
  rendering all 9 response fields honestly): confirmed live in a real browser — screenshot shows
  a styled page (real CSS output, not near-empty), four nav links (Browse/Workbench/Graph/Health),
  and all 9 `GetStatusResponse` fields rendered (Initialized, Schema version, Nodes, Edges, Files,
  Stale, Indexed commit, Index directory found, Re-index in progress). Re-ran against a no-index
  scratch directory: renders "No index found for this repository" — an honest not-indexed state,
  not an error or a hung spinner — confirming `web/src/routes/+page.svelte`'s `classify()` reads
  `status.indexingInProgress` (field 9) directly rather than inferring it from
  `storeExists && !initialized` (the prohibited shortcut).
- **`shadcn-svelte` dev-dependency prohibition**: `rg -c "shadcn-svelte" web/package.json
  web/pnpm-lock.yaml` → 0 matches (init's own added devDependency was removed per 02-05's
  Deviation 3, confirmed still absent at HEAD).

### Code Review Resolution (02-REVIEW.md)

| Finding | Claimed disposition | Verified in code |
|---|---|---|
| WR-01 (unanchored `src=` substring CSP match) | Fixed, commit `0d110682` | Confirmed: `spaExternalSrcAttrRE = regexp.MustCompile(`(?i)(^\|\s)src\s*=`)` present at `internal/uiserver/spa.go:127`, used at lines 157 and 164. |
| IN-02 (CSP fail-closed path untested) | Fixed, commit `0d110682` | Confirmed: `TestNewSPAHandlerFailsClosedWithoutIndexHTML` exists in `internal/uiserver/spa_test.go:765`. |
| IN-01 (unreachable `fs.Sub` error branch) | Accepted, not changed | Confirmed unchanged at `internal/uiserver/server.go:134-138` — a deliberate, disclosed non-fix, not a gap. |

### Required Artifacts (spot-checked directly, not via SUMMARY claim)

| Artifact | Expected | Status | Details |
|---|---|---|---|
| `web/embed.go` | `package web`, `//go:embed all:build`, `var BuildFS embed.FS` | ✓ VERIFIED | Exact match; doc comment correctly explains the `..`-embed-pattern constraint and cites `claudeassets.go` precedent. |
| `web/build/**` | Committed, 26 tracked files incl. `_app/immutable/` | ✓ VERIFIED | `git ls-files web/build` = 26; not gitignored. |
| `internal/uiserver/spa.go` | Three-way rule, cache policy, CSP, `nosniff` | ✓ VERIFIED | Live-tested: 404 on missing asset, fallback on client route, immutable cache on hashed asset, `no-store` on `index.html`, CSP + `nosniff` on every response. |
| `Taskfile.yml` (`web:deps`, `web:deps:strict`, `web:lockfile`, `web:audit`, `web:build:verify`, `web:drift`) | All six targets present and green | ✓ VERIFIED | All six run live in this session at HEAD, all PASS. |
| `.github/workflows/ci.yml` | Node 24, no mutable cache, 5 JS steps in `test` job | ✓ VERIFIED | `node-version: "24"`, no `cache:` input on the Node step, all 5 steps' `run:` bodies are `task <target>`. |
| `web/pnpm-workspace.yaml` | `strictDepBuilds: true`, `allowBuilds`, `overrides` | ✓ VERIFIED | All three present and correctly shaped. |
| `SECURITY.md` | Two-scanner disjoint-scope statement | ✓ VERIFIED | Confirmed text present, names the vendored-component gap. |
| `web/src/routes/+page.svelte` | All 9 `GetStatusResponse` fields rendered | ✓ VERIFIED | Confirmed in source and live screenshot. |
| `.goreleaser.yaml`, `.github/workflows/release.yml` | Zero JS-toolchain invocations | ✓ VERIFIED | `rg -c` zero matches, positive-controlled. |

### Requirements Coverage

| Requirement | Source Plan(s) | Status | Evidence |
|---|---|---|---|
| RPC-03 | 02-02, 02-05 | ✓ SATISFIED | Criterion 2 above. REQUIREMENTS.md: `Complete`. |
| BLD-01 | 02-01, 02-03, 02-04, 02-06, 02-07 | ✓ SATISFIED | Criteria 1 & 3 above. REQUIREMENTS.md: `Complete`. |
| BLD-02 | 02-01, 02-02, 02-03, 02-05 | ✓ SATISFIED | Criterion 1 above. REQUIREMENTS.md: `Complete`. |
| BLD-03 | 02-06, 02-07 | ✓ SATISFIED | Criterion 3 above. REQUIREMENTS.md: `Complete`. |
| BLD-05 | 02-07 | ✓ SATISFIED | Criterion 4 above. REQUIREMENTS.md: `Complete`. |
| BLD-06 | 02-07 | ✓ SATISFIED | Criterion 5 above. REQUIREMENTS.md: `Complete`. |
| BLD-07 | 02-04 | ✓ SATISFIED | Criterion 5 above. REQUIREMENTS.md: `Complete`. |

No orphaned requirements — `.planning/REQUIREMENTS.md`'s Phase 2 rows exactly match the 7
declared requirement IDs across the 7 plans; `BLD-04` also lists `Phase 2` region proximity in
the file but is attributed to Phase 1, not orphaned here.

### Decision Coverage (informal, non-blocking per verifier-phase-gates.md)

All 19 locked decisions (D-01–D-19) from `02-CONTEXT.md` were traced into shipped artifacts
during this verification (see "Plan-Level Must-Haves" above and inline citations throughout).
None were found abandoned or silently reversed.

### Anti-Patterns Found

`TBD`/`FIXME`/`XXX` scan and `TODO`/`HACK`/`PLACEHOLDER`/"not yet implemented" scan across all
phase-modified core files (`web/embed.go`, `internal/uiserver/spa.go`,
`internal/uiserver/spa_test.go`, `internal/uiserver/server.go`,
`internal/upgrade/taskfile_shape_test.go`, `internal/upgrade/proto_task_test.go`, `Taskfile.yml`,
`.github/workflows/ci.yml`, `buf.gen.ts.yaml`, `web/src/lib/gen/ui_pb.ts`, `web/src/lib/client.ts`,
`web/src/routes/+page.svelte`, `SECURITY.md`, `web/src/routes/+layout.svelte`): **zero matches.**
No stub patterns (`return null`/`return {}`/empty handlers) found in the SPA handler or the app
shell. No skipped Go tests (`t.Skip`) in any requirement-linked test file.

### Behavioral Spot-Checks / Live Verification

| Behavior | Command | Result | Status |
|---|---|---|---|
| Go-only build | `GOTOOLCHAIN=go1.26.5 go build ./cmd/codegraph` | exit 0, 78MB binary | ✓ PASS |
| Server boots with no JS toolchain on PATH | `env -i PATH=/usr/bin:/bin CI=1 ./codegraph ui --no-open` | printed loopback URL, served real HTML | ✓ PASS |
| `GET /` from embedded FS | `curl` | 200, real `index.html`, 8 `_app/immutable/` refs | ✓ PASS |
| Underscore-prefixed chunk reachable | `curl` on `_app/immutable/entry/start.*.js` | 200, 77 bytes, immutable cache header | ✓ PASS |
| Client route fallback | `curl /browse` | 200, byte-identical to `index.html` | ✓ PASS |
| Missing hashed asset | `curl` on a nonexistent `_app/immutable/` path | 404 | ✓ PASS |
| RPC precedence | `curl -X POST /codegraph.ui.v1.UIService/GetStatus` | 200, `application/json`, real data | ✓ PASS |
| `task web:deps` | live run | frozen-lockfile install succeeded, lockfile unchanged, reported self-managed fallback path | ✓ PASS |
| `task web:build:verify` | live run | 25 files built to scratch, committed tree untouched | ✓ PASS |
| `task web:drift` | live run | hashed 22 source / manifested 25 output, both MATCH, PASS | ✓ PASS |
| `task proto:drift` | live run | compared 4 files, byte-identical | ✓ PASS |
| `task web:deps:strict` | live run | `strictDepBuilds` = true, `allowBuilds` committed | ✓ PASS |
| `task web:lockfile` | live run | 129 packages, 129/129 integrity-bearing, 0 rejected sources | ✓ PASS |
| `task web:audit` | live run | CLEAN, 0 advisories | ✓ PASS |
| BLD-07 structural scanner | `go test -run "TestReleasePath...\|TestJSInstallPath..."` | 7/7 PASS incl. positive control | ✓ PASS |
| `go test`/`go vet` on phase packages | `go test -count=1 ./internal/uiserver/... ./internal/upgrade/...` | both `ok`, uncached | ✓ PASS |
| Real Chromium browser render (indexed repo) | `agent-browser open/screenshot/console/network` | app renders, 4 nav slots, 9 fields with real data (6064 nodes), 0 console errors, 1 `Document` request total (client-side nav confirmed) | ✓ PASS |
| Real Chromium browser render (no-index repo) | same | honest "No index found for this repository" state, 0 console errors | ✓ PASS |

## Human Verification

Both `<human-check>` blocks the planner deferred to end-of-phase UAT (per
`workflow.human_verify_mode=end-of-phase`) were **executed directly in this verification session**
rather than left open, with hard evidence (not a restated claim):

1. **02-01 Task 1's human-check** ("On a shell whose PATH contains no `node` and no `pnpm`,
   build and run `codegraph ui`, confirm the page renders rather than 404s"). **Executed:**
   `env -i PATH=/usr/bin:/bin` (no node, pnpm, npx, npm, or corepack reachable) — `GET /` returned
   200 with the real SPA. **Result: satisfied.**
2. **02-05 Task 3's human-check** ("Confirm the status panel shows real counts, the four nav
   slots load without a full reload, a direct route load works, the console shows no errors and
   no CSP violations; then confirm the not-indexed state renders honestly"). **Executed:** live
   Chromium session against this repo (real index) showed all 9 fields with real non-zero counts,
   4 working nav links, zero console output; the network log for a nav-link click showed exactly
   one `(Document)` request for the whole session (all further navigation was `_app/immutable/`
   script fetches — genuine client-side routing, not a reload); a second live session against an
   empty scratch directory rendered "No index found for this repository" with zero console errors,
   never a hung spinner or an error page. **Result: satisfied.**

No human verification items remain open. Screenshots and network/console transcripts were
captured during this session but are ephemeral session artifacts (not committed to the repo);
this report records their content and pass/fail result rather than a file path, consistent with
how the phase's own SUMMARYs record human-check evidence.

## Gaps Summary

None. All 5 ROADMAP success criteria are independently verified against live, executed evidence —
not SUMMARY claims. All 7 requirement IDs are satisfied and correctly marked `Complete` in
REQUIREMENTS.md with no orphans. The one cross-plan integration break the orchestrator flagged
before this verification (02-03's `proto:drift` floor raised to 4 without updating
`proto_task_test.go`'s paired assertion) was already fixed at commit `1d2a1d2f` and is confirmed
green at HEAD. The two code-review findings marked "Fixed" (WR-01, IN-02) are confirmed present
in the code, not merely claimed in 02-REVIEW.md's resolution table. The one deliberately accepted
finding (IN-01) is confirmed unchanged, consistent with its "accepted, not changed" disposition.
The one deliberately deferred pnpm-pin/Corepack gap recorded in 02-01-SUMMARY.md's Deviation 4
is confirmed durably closed by 02-06's `web:deps` two-path (Corepack-or-self-managed) resolution,
demonstrated live in this session on a Corepack-less host. The one deliberately scoped-out
limitation (BLD-06 cannot see registry-vendored shadcn-svelte component source, and the
`pnpm audit` ADVISORIES-PRESENT branch is implemented-but-unexercised) is recorded honestly in
both `02-CONTEXT.md`/`SECURITY.md` and this plan's own `<known_limitations>` block — not a hidden
gap, a stated boundary Phase 3 inherits by name.

---

_Verified: 2026-08-25T01:30:00Z_
_Verifier: Claude (gsd-verifier)_
