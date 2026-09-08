---
phase: 02-spa-toolchain-embedded-app-shell-js-supply-chain
plan: 02
subsystem: ui
tags: [http-servemux, csp, cache-control, spa-routing, connect-rpc, uiserver]

requires:
  - phase: 02-spa-toolchain-embedded-app-shell-js-supply-chain
    provides: "02-01's newSPAHandler stub, spaFallbackFile/spaAssetPrefix/spaSubdirName consts, the mux mounting SPA at \"/\" inside originHostGuard"
provides:
  - "internal/uiserver/spa.go — the complete D-10 three-way serve rule (immutable-prefix 404, real-file passthrough, client-route fallback)"
  - "D-11's two-class Cache-Control policy (spaCacheControlImmutable / spaCacheControlNoStore / spaCacheControlDefault)"
  - "A same-origin Content-Security-Policy, composed at handler-construction time from spaCSPBaseDirectives plus script-src/style-src hashes spaInlineBlockHashes derives from the embedded index.html"
  - "Proof, against a real origin-guarded server, that ServeMux precedence alone (no allowlist) separates the Connect RPC prefix from the SPA's \"/\""
affects: [02-03, 02-05, 02-06, 02-07]

actuals:
  tokens: 9443
  tasks: 2
  commits: 2

tech-stack:
  added: []
  patterns:
    - "CSP script-src/style-src hashes are derived from the embedded index.html at handler-construction time (buildSPACSPPolicy), never hand-maintained as a literal — the policy re-derives itself for free whenever the SPA rebuild changes the bootstrap script's bytes"
    - "X-Content-Type-Options and Content-Security-Policy are set unconditionally as the FIRST two lines of ServeHTTP, before the method check, so even the 405 response carries both headers"
    - "A purpose-built regexp scan (spaScriptBlockRE/spaStyleBlockRE) extracts inline <script>/<style> block content instead of pulling in an HTML parser, justified because the input is this repo's own committed, drift-guarded build output, not adversarial HTML"

key-files:
  created: []
  modified:
    - internal/uiserver/spa.go
    - internal/uiserver/spa_test.go

key-decisions:
  - "style-src falls back to 'self' 'unsafe-inline' rather than a hash source, because the embedded index.html's only inline styling is a style=\"display: contents\" ATTRIBUTE on the SPA root <div>, not a <style> element — CSP hash sources apply only to <style> element content, never to attribute values (that requires the separate 'unsafe-hashes' keyword). This is the plan's explicitly permitted fallback (Claude's Discretion under D-11); script-src carries no such fallback and none was needed — the one inline bootstrap <script> hashes cleanly."
  - "Local dev toolchain drift (not a code change, recorded as a deviation): this machine's `go` resolved to 1.27.0, one minor ahead of go.mod's declared `go 1.26.5`, and go1.27.0's runtime broke a go:linkname hack in the pinned github.com/cockroachdb/swiss@v0.0.0-20251224182025-b0f6560f979b dependency (undefined: hashFn/fastrand64/getRuntimeHasher), which is pulled in transitively via internal/graphstore. All verification in this plan ran under `GOTOOLCHAIN=go1.26.5`, which Go auto-downloaded and used successfully. CI is unaffected: every workflow uses `go-version-file: go.mod`, which resolves the pinned 1.26.5 exactly. No go.mod/go.sum change was made — the pin already names the correct version; the gap is this host's `go` binary being ahead of it, mirroring 02-01's pnpm/Corepack finding in shape."
  - "TDD tests were written for BOTH tasks (10 named Task-1 tests plus 2 named Task-2 tests) before any implementation existed, and RED was confirmed as a single build failure (undefined spaCacheControlNoStore, spaCacheControlDefault, spaInlineBlockHashes) rather than as 12 separate compile attempts — spa_test.go is one file and Go fails the whole package on any undefined symbol. The commit history still carries a genuine RED gate (`test(02-02)`, 7247ecc) followed by a genuine GREEN gate (`feat(02-02)`, 7d351d8): the RED commit alone does not build; the GREEN commit makes every test in the file pass, including Task 2's, since D-09 precedence was already correctly wired by 02-01's server.go and needed no new implementation of its own."

requirements-completed: [RPC-03, BLD-02]  # Both are declared by more than one sibling plan in this phase — RPC-03 by 02-02 and 02-05; BLD-02 by 02-01, 02-02, 02-03 and 02-05. Neither marks complete in REQUIREMENTS.md until every declaring plan finishes, via requirements.ready-ids (shared-ID gate, #2388). Confirmed via `requirements.ready-ids`: 0/2 ready at this plan's completion.

coverage:
  - id: D1
    description: "The complete D-10 three-way serve rule (immutable-prefix miss -> 404, real file -> itself, everything else -> index.html) plus D-11's two-class Cache-Control policy and a same-origin CSP with hashes derived from the embedded index.html, set unconditionally on every response including the 405 path"
    requirement: BLD-02
    verification:
      - kind: unit
        ref: "internal/uiserver/spa_test.go#TestSPAClientRouteFallsBackToIndex"
        status: pass
      - kind: unit
        ref: "internal/uiserver/spa_test.go#TestSPAImmutableAssetMissReturns404"
        status: pass
      - kind: unit
        ref: "internal/uiserver/spa_test.go#TestSPAImmutableAssetHitIsCachedImmutable"
        status: pass
      - kind: unit
        ref: "internal/uiserver/spa_test.go#TestSPAIndexIsNotCached"
        status: pass
      - kind: unit
        ref: "internal/uiserver/spa_test.go#TestSPASetsNoSniffOnEveryResponse"
        status: pass
      - kind: unit
        ref: "internal/uiserver/spa_test.go#TestSPASetsCSPOnEveryResponse"
        status: pass
      - kind: unit
        ref: "internal/uiserver/spa_test.go#TestSPACSPForbidsUnsafeDirectives"
        status: pass
      - kind: unit
        ref: "internal/uiserver/spa_test.go#TestSPACSPHashesCoverEmbeddedInlineScripts"
        status: pass
      - kind: unit
        ref: "internal/uiserver/spa_test.go#TestSPARejectsNonReadMethods"
        status: pass
      - kind: unit
        ref: "internal/uiserver/spa_test.go#TestSPAServesRootStaticFileWithoutFallback"
        status: pass
    human_judgment: false
  - id: D2
    description: "RPC-versus-SPA precedence proven in both directions against a real, origin-guarded server: a Connect JSON POST to GetStatus reaches the Connect handler (JSON Content-Type, body not index.html) while a GET on a client-side route returns the SPA index.html on the same server instance; a foreign Host is rejected before the SPA handler ever runs"
    requirement: RPC-03
    verification:
      - kind: integration
        ref: "internal/uiserver/spa_test.go#TestSPARPCPathReachesConnectHandler"
        status: pass
      - kind: integration
        ref: "internal/uiserver/spa_test.go#TestSPAInheritsOriginHostGuard"
        status: pass
    human_judgment: false

duration: ~40min
completed: 2026-08-24
status: complete
---

# Phase 2 Plan 2: SPA Toolchain, Embedded App Shell & JS Supply Chain — Routing & CSP Summary

**`internal/uiserver/spa.go` now implements D-10's complete three-way asset/route rule, D-11's two-class Cache-Control policy, and a same-origin Content-Security-Policy whose script-src hashes are derived from the embedded `index.html` at handler-construction time — closing ROADMAP criterion 2 in full.**

## Performance

- **Duration:** ~40 min
- **Started:** 2026-08-24 (session start; exact wall-clock not separately logged)
- **Completed:** 2026-08-24
- **Tasks:** 2
- **Files modified:** 2 (`internal/uiserver/spa.go`, `internal/uiserver/spa_test.go`)

## Accomplishments

- Implemented `ServeHTTP`'s complete three-way rule: a miss under `_app/immutable/` returns 404 (never falls back to `index.html`, by design — D-10), a real file outside that prefix is served as itself with its own resolved `Content-Type` and `spaCacheControlDefault`, and everything else (including a client-route path with dotted segments, the case the rejected dot-heuristic would have broken) falls back to `index.html` with 200.
- Implemented D-11's two-class `Cache-Control` policy as three named consts: `spaCacheControlImmutable` (`public, max-age=31536000, immutable`) on hashed assets, `spaCacheControlNoStore` (`no-store`) on `index.html`, and `spaCacheControlDefault` (`no-cache`, Claude's Discretion) on other static passthrough files such as `robots.txt`.
- `X-Content-Type-Options: nosniff` and `Content-Security-Policy` are set as the FIRST two header writes in `ServeHTTP`, before the method check — so even the 405 response on a non-GET/HEAD request carries both, closing T-02-02-01 and T-02-02-06 unconditionally rather than per-branch.
- Built the Content-Security-Policy from three symbols with distinct roles: `spaCSPBaseDirectives` (the fixed half — `default-src 'self'; connect-src 'self'; object-src 'none'; base-uri 'self'; form-action 'self'; frame-ancestors 'none'`), `spaInlineBlockHashes` (a purpose-built regexp scan extracting byte-exact inline `<script>`/`<style>` block content from the embedded `index.html` and SHA-256/base64-encoding each into a `'sha256-…'` source), and `buildSPACSPPolicy` (composes the two, called once in `newSPAHandler` and stored on the handler).
- Empirically verified the embedded `index.html`'s inline content before writing any CSP code, per the plan's own instruction: it carries exactly one inline `<script>` bootstrap block (no `src=`) and zero `<style>` elements, but one inline `style="display: contents"` ATTRIBUTE on its root `<div>`. `script-src` hashes the one script block cleanly (`'self' 'sha256-0rSP+rkAV1Y/HKiwC5kF+LZ4d6XIRZzEPt8M1L9Du7k='`). `style-src` falls into the plan's explicitly permitted fallback branch: no `<style>` element exists to hash, but the inline attribute does, so `style-src` is `'self' 'unsafe-inline'` — `script-src` needed no such relaxation.
- The full served policy, captured verbatim: `default-src 'self'; connect-src 'self'; object-src 'none'; base-uri 'self'; form-action 'self'; frame-ancestors 'none'; script-src 'self' 'sha256-0rSP+rkAV1Y/HKiwC5kF+LZ4d6XIRZzEPt8M1L9Du7k='; style-src 'self' 'unsafe-inline'`.
- Proved D-09's RPC-vs-SPA precedence in both directions against a REAL, origin-guarded, `mustListen`-started server (never a bare mux): a plain Connect-protocol JSON `POST` to `uiv1connect.UIServiceGetStatusProcedure` (path derived from the generated package's own exported constant, never hardcoded) returns `application/json...` and a body that is not `index.html`'s bytes, while a `GET` on a client-side route on the SAME server instance returns `index.html` — and a foreign `Host` header is rejected with 403 before the SPA handler ever runs (`TestSPAInheritsOriginHostGuard`), proving the SPA handler was never mounted outside `originHostGuard`.
- TDD RED->GREEN: all 12 named tests (10 for Task 1, 2 for Task 2) were written first; the whole package failed to build (undefined `spaCacheControlNoStore`, `spaCacheControlDefault`, `spaInlineBlockHashes`) before implementation; every test — including the pre-existing tracer tests from 02-01 — passes after `spa.go`'s implementation, with zero `--- FAIL` lines under `set -o pipefail`.
- Used `http.ServeContent` only for byte-serving an already-resolved file with headers already set (immutable-hit and real-file branches) — never `http.FileServer`/`http.FileServerFS`/`http.ServeFileFS` for the SPA tree, avoiding their implicit `index.html` redirect and directory-listing behavior (RESEARCH Pitfall 6). Verified: zero non-comment matches for the forbidden helpers in `spa.go`, alongside a positive control confirming the file is non-empty and does reference `ServeHTTP`.

## Task Commits

TDD tasks: both tasks' tests were authored together before implementation (see Deviations for why RED was confirmed as one build failure across both tasks' tests rather than two separate compile attempts).

1. **Tests (RED) — Task 1 + Task 2's 12 named tests** — `7247ecc` `test(02-02): add failing tests for SPA three-way rule, cache policy, CSP, and RPC precedence`
2. **Implementation (GREEN) — Task 1's `spa.go` deliverable, which also makes Task 2's tests pass with no further implementation needed** — `7d351d8` `feat(02-02): implement SPA three-way rule, two-class cache policy, and derived CSP`

**Plan metadata:** committed separately after this SUMMARY (see below).

## Files Created/Modified

- `internal/uiserver/spa.go` — `ServeHTTP`'s complete three-way rule; `spaCacheControlImmutable`/`spaCacheControlNoStore`/`spaCacheControlDefault`/`spaCSPBaseDirectives` consts; `spaScriptBlockRE`/`spaStyleBlockRE`/`spaHasInlineStyleAttrRE` regexps; `cspHashSource`, `spaInlineBlockHashes`, `buildSPACSPPolicy` helpers; `spaHandler.csp` field computed once in `newSPAHandler`; `serveFile`/`serveFallback` helper methods
- `internal/uiserver/spa_test.go` — 12 new named tests (`TestSPAClientRouteFallsBackToIndex`, `TestSPAImmutableAssetMissReturns404`, `TestSPAImmutableAssetHitIsCachedImmutable`, `TestSPAIndexIsNotCached`, `TestSPASetsNoSniffOnEveryResponse`, `TestSPASetsCSPOnEveryResponse`, `TestSPACSPForbidsUnsafeDirectives`, `TestSPACSPHashesCoverEmbeddedInlineScripts`, `TestSPARejectsNonReadMethods`, `TestSPAServesRootStaticFileWithoutFallback`, `TestSPARPCPathReachesConnectHandler`, `TestSPAInheritsOriginHostGuard`) plus helpers `newTestSPAHandler`, `findImmutableAssetPath`, `parseCSP`, `spaRequestShape`/`spaBaseRequestShapes`, `cspForbidsUnsafe`

## Decisions Made

See `key-decisions` in frontmatter. Summary: the `style-src` fallback is the plan's own explicitly permitted relaxation, taken because the empirical check the plan required (inspecting the real `web/build/index.html`) found an inline style ATTRIBUTE rather than a `<style>` element — no ambiguity, no alternative reading. The local Go-toolchain drift is an environment finding, not a code decision, recorded so the next session doesn't have to re-diagnose it.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Test table design conflicted with the plan's own `--- PASS: $t` grep-count verify command**
- **Found during:** Task 1, first verification run
- **Issue:** The plan's `<verify>` command counts exactly one `--- PASS: <TestName>` line per named test via `grep -c -- "--- PASS: $t"`. My first draft used `t.Run(...)` subtests for `TestSPAClientRouteFallsBackToIndex` (2 cases), `TestSPASetsNoSniffOnEveryResponse` (4 cases) and `TestSPASetsCSPOnEveryResponse` (5 cases). Each subtest line (`--- PASS: TestName/case_name`) contains the parent test's name as a substring, so the grep count came back 3, 5 and 6 respectively instead of 1 — the verify command would have failed even though the tests themselves were all green.
- **Fix:** Rewrote all three tests to iterate their case tables with a plain `for` loop and `t.Fatalf` (no `t.Run`), so each emits exactly one top-level `--- PASS` line. Documented the reason inline as a comment on each affected test, so a future edit doesn't reintroduce subtests without re-reading why.
- **Files modified:** `internal/uiserver/spa_test.go`
- **Verification:** Re-ran the plan's exact `<verify>` command; all ten Task-1 test names show `count=1`, zero `--- FAIL` lines.
- **Committed in:** `7247ecc` (part of the RED commit — the fix was made before the first GREEN run, so it never appeared as a separate commit)

**2. [Rule 3 - Blocking] Local `go` binary (1.27.0) ahead of go.mod's pinned `go 1.26.5`, breaking a transitive dependency's runtime hook**
- **Found during:** first `go build`/`go vet` attempt, before any test ran
- **Issue:** `go build ./internal/uiserver/...` (and `./...`) failed with `undefined: hashFn` / `undefined: fastrand64` / `undefined: getRuntimeHasher` inside `github.com/cockroachdb/swiss@v0.0.0-20251224182025-b0f6560f979b` (pulled in transitively via `internal/graphstore` -> `internal/indexer`/`internal/query`, which `internal/uiserver`'s own tests import). This is a `go:linkname` hook into Go's runtime internals (`runtime_go1.20.go` in that dependency) that go1.27.0's runtime no longer exposes under the same names — unrelated to any code in this plan.
- **Fix:** Ran every verification command in this plan under `GOTOOLCHAIN=go1.26.5` (Go auto-downloaded that exact toolchain on first use and cached it). No `go.mod`/`go.sum` change — the declared `go 1.26.5` directive is already correct; the gap is this host's `go` binary resolving to a newer minor than the pin.
- **Files modified:** none (invocation-only workaround)
- **Verification:** `GOTOOLCHAIN=go1.26.5 go build ./...` and `GOTOOLCHAIN=go1.26.5 go test ./internal/uiserver/...` both clean.
- **Committed in:** n/a (no file change)

---

**Total deviations:** 2 auto-fixed (1 test-design fix to satisfy the plan's own verify command, 1 local-environment toolchain workaround).
**Impact on plan:** Neither changes architecture or scope. Deviation 1 is purely a test-authoring correction caught by re-running the plan's own literal verify command — a useful signal that "exactly one PASS line" grep-based verification and `t.Run` subtests are structurally incompatible, worth remembering for future plans in this repo. Deviation 2 mirrors 02-01's pnpm/Corepack finding in shape (a local dev-machine version ahead of the repo's pin) and is CI-safe: every workflow uses `go-version-file: go.mod`, which resolves the pinned `1.26.5` exactly, so this is a local-only finding, not a CI risk.

## Issues Encountered

None beyond the deviations above.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- ROADMAP criterion 2 is now demonstrable end to end: a directly-opened client-side route loads the app, RPC paths and hashed-asset URLs resolve to their real handlers, a missing hashed chunk 404s instead of arriving as HTML, and the cache policy makes the post-upgrade stale-shell failure impossible.
- Both `RPC-03` and `BLD-02` are declared by more than one sibling plan in this phase (`RPC-03`: 02-02, 02-05; `BLD-02`: 02-01, 02-02, 02-03, 02-05) — neither marks complete in REQUIREMENTS.md until every declaring plan finishes, via the shared-ID gate. `requirements.ready-ids` confirmed 0/2 ready at this plan's completion, which is correct, not a gap.
- The CSP's `script-src`/`style-src` hash sources are re-derived automatically whenever `web/build/index.html`'s inline blocks change — 02-05 (the JS toolchain drift guard) rebuilding the tree will not go stale against this policy without a code change.
- **02-05's human-check (browser open) is what confirms the app actually boots under this CSP in a real browser** — this plan's tests prove the policy's shape and its hash-coverage of the embedded bytes, not that Chrome/Firefox actually execute the app under it. If the app does not boot under 02-05's check, the flagged-assumption instruction is explicit: stop and report, do not loosen `script-src`.
- If a future SvelteKit rebuild adds a `<style>` ELEMENT to `index.html` (as opposed to today's inline attribute-only styling), `buildSPACSPPolicy`'s branch 1 will automatically hash it — no code change needed, since the branch selection is driven by what `spaInlineBlockHashes` actually finds in the embedded bytes each time the handler is constructed.

## Self-Check: PASSED

Confirmed on disk: `internal/uiserver/spa.go`, `internal/uiserver/spa_test.go`, this SUMMARY.md.
Confirmed in `git log --oneline --all`: `7247ecc` (test/RED), `7d351d8` (feat/GREEN).
Re-ran the plan's exact `<verify>` commands for both tasks post-commit: Task 1's 10 named tests each show exactly one `--- PASS` line and zero `--- FAIL` lines; Task 2's 2 named tests each show exactly one `--- PASS` line and zero `--- FAIL` lines; `go test ./internal/uiserver/...` and `go vet ./internal/uiserver/...` both clean; `gofmt -l` reports no formatting issues on either changed file.

---
*Phase: 02-spa-toolchain-embedded-app-shell-js-supply-chain*
*Plan: 02*
*Completed: 2026-08-24*
