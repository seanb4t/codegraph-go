---
phase: 2
slug: spa-toolchain-embedded-app-shell-js-supply-chain
status: verified
# threats_open = count of OPEN threats at or above workflow.security_block_on severity (the blocking gate)
threats_open: 0
asvs_level: 1
created: 2026-08-25
---

# Phase 2 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

**Register origin:** authored at plan time. All seven PLAN.md files carry a parseable
`<threat_model>` block, so this audit **verifies mitigations exist** — it does not
retroactively scan for new threats.

**Why this phase has a security contract at all:** Phase 2 is the first phase to put a
JavaScript toolchain anywhere in this repository, and the first to serve HTML from the
binary. Both are new trust boundaries for a project whose value proposition is a single
static Go binary with a verifiable supply chain.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| npm registry → `web/node_modules` → `web/pnpm-lock.yaml` | Third-party code enters the repository and executes at install time | Arbitrary executable code (lifecycle scripts) |
| npm advisory database → the audit verdict | A scan whose transport fails must not read as a clean verdict | Vulnerability findings, scan status |
| browser → `internal/uiserver` mux | Untrusted request paths reach the new static-asset handler; this server previously emitted only JSON | Request paths, `Host` header |
| `web/build/` on disk → embedded bytes in the shipped binary | Whatever is committed here is what every user executes in their browser | HTML, JS, CSS |
| browser cache → shipped binary | A stale cached shell against a new RPC surface is a correctness failure presenting as a broken UI | Cached assets |
| repository source → signed, attested release artifacts | Anything executable inside `.goreleaser.yaml` or `release.yml` runs with release credentials and lands in SLSA provenance | Build commands, action references |
| Connect JSON response → rendered DOM | Server-controlled values are interpolated into the page | Status fields, commit SHA |
| shadcn-svelte registry → committed `.svelte` source | Registry-fetched component source enters the repo **outside** the lockfile | *Not crossed in Phase 2* — see AR-04 |

---

## Threat Register

50 threats. All closed: 43 mitigated with verified controls, 7 accepted and logged below.

### 02-01 — SPA toolchain tracer

| Threat ID | Category | Component | Severity | Disposition | Mitigation | Status |
|-----------|----------|-----------|----------|-------------|------------|--------|
| T-02-01-SC | Tampering | `pnpm install` lifecycle scripts | high | mitigate | `strictDepBuilds: true` in `web/pnpm-workspace.yaml:22` before first install; `allowBuilds` committed. Blocking-human package-legitimacy checkpoint preceded the install. | closed |
| T-02-01-02 | Tampering | SPA handler path handling | high | mitigate | `fs.Sub` over `embed.FS` (`server.go:134`); `path.Clean` + `fs.ValidPath` (`spa.go:252-254`). An `embed.FS` has no parent-directory to escape to. | closed |
| T-02-01-03 | Spoofing | New `"/"` route on the loopback listener | high | mitigate | Registered inside `originHostGuard` (`server.go:141`), asserted by `TestSPAInheritsOriginHostGuard`. | closed |
| T-02-01-04 | Information disclosure | Embedded tree contents | low | accept | AR-01 | closed |
| T-02-01-05 | Tampering | MIME-sniffing on newly served static assets | medium | mitigate | Deferred to 02-02 and delivered there: `X-Content-Type-Options: nosniff` (`spa.go:242`). Deferral was tracked, not silent. | closed |
| T-02-01-06 | Tampering | Injected embedded asset reaching a third-party origin | medium | mitigate | Deferred to 02-02 and delivered there — see T-02-02-06. | closed |
| T-02-01-07 | Tampering | Project-generation tooling from a floating registry tag | high | mitigate | `pnpm dlx sv@0.17.0` with registry `dist.integrity` confirmed byte-for-byte before execution (02-01-SUMMARY.md:126). A `dlx` CLI never enters the lockfile, so pin + integrity is its only provenance. | closed |

### 02-02 — SPA routing, cache policy, CSP

| Threat ID | Category | Component | Severity | Disposition | Mitigation | Status |
|-----------|----------|-----------|----------|-------------|------------|--------|
| T-02-02-01 | Tampering | MIME-sniffing on served assets | medium | mitigate | `nosniff` set unconditionally before any branch in `ServeHTTP`, asserted across all four response shapes including the 404. | closed |
| T-02-02-02 | Information disclosure | Path traversal into the host filesystem | low | mitigate | `fs.Sub` over `embed.FS`; `path.Clean` + `fs.ValidPath` as defense in depth. | closed |
| T-02-02-03 | Spoofing | New static routes reachable via DNS rebinding | high | mitigate | Handler registered on the mux `originHostGuard` wraps; `TestSPAInheritsOriginHostGuard` is the regression test for a future refactor moving it outside. | closed |
| T-02-02-04 | Tampering | Stale cached shell against a new RPC surface | medium | mitigate | `index.html` `no-store`; non-hashed statics `no-cache`; only `_app/immutable/` gets a long-lived directive (D-11). | closed |
| T-02-02-05 | Denial of service | Unbounded response from a large embedded asset | low | accept | AR-02 | closed |
| T-02-02-06 | Tampering | Compromised embedded asset exfiltrating to a third-party origin | medium | mitigate | CSP on every response: `default-src`/`connect-src` `'self'`, `object-src 'none'`, `base-uri 'self'`, `form-action 'self'`, `frame-ancestors 'none'`, `script-src 'self'` + per-block `'sha256-…'` derived from the embedded `index.html`. `TestSPASetsCSPOnEveryResponse`, `TestSPACSPForbidsUnsafeDirectives`, `TestSPACSPHashesCoverEmbeddedInlineScripts`. | closed |
| T-02-02-07 | Elevation of privilege | A "make the CSP stop complaining" edit loosening `script-src` | medium | mitigate | `'unsafe-eval'` anywhere and `'unsafe-inline'` in `script-src` prohibited in `must_haves.prohibitions` and asserted against the parsed policy by `TestSPACSPForbidsUnsafeDirectives`, which carries its own positive control. | closed |

### 02-03 — TypeScript Connect client

| Threat ID | Category | Component | Severity | Disposition | Mitigation | Status |
|-----------|----------|-----------|----------|-------------|------------|--------|
| T-02-03-SC | Tampering | `@bufbuild/protoc-gen-es` + Connect/protobuf runtimes | high | mitigate | Covered by 02-01's blocking-human legitimacy checkpoint plus `strictDepBuilds`. Pinned in `web/pnpm-lock.yaml`; scoped name verified against the registry, unscoped homoglyph-adjacent name confirmed to 404 rather than resolve to a squatter. | closed |
| T-02-03-02 | Tampering | Committed generated TS drifting from its `.proto` source | high | mitigate | `task proto:drift` enumerates four files, prints the count before comparing, hard-fails below the floor of 4. Floor independently pinned by `internal/upgrade/proto_task_test.go` (see Audit Note 1). | closed |
| T-02-03-03 | Spoofing | A generation step that silently skips when its plugin is absent | high | mitigate | Hard precondition on `web/node_modules/.bin/protoc-gen-es` with a named remedy (`Taskfile.yml:257-258`); skip-when-missing prohibited by D-05. | closed |
| T-02-03-04 | Information disclosure | Storage schema shape leaking into browser-shipped code | medium | mitigate | `protoc-gen-es` scoped to `internal/uiproto/uiv1`, so `internal/schema/graph.proto` never reaches it; asserted by the single-file `git ls-files` count. | closed |
| T-02-03-05 | Tampering | Runtime/plugin major-version skew | low | mitigate | `@bufbuild/protobuf` and `@bufbuild/protoc-gen-es` pinned to the same minor line and locked. | closed |

### 02-04 — Release-path JS-purity scanner

| Threat ID | Category | Component | Severity | Disposition | Mitigation | Status |
|-----------|----------|-----------|----------|-------------|------------|--------|
| T-02-04-01 | Tampering | A JS-toolchain invocation entering the signed release path | high | mitigate | Fixture-backed structural scan over the release path's transitive execution closure, run by the required `test` CI context; `TestReleasePathScanIsNonVacuous` proves the scanner fires. | closed |
| T-02-04-02 | Elevation of privilege | A marketplace action installing a JS runtime on the release runner | high | mitigate | `uses:` values scanned by owner/repo prefix — an action that names no command but installs a runtime is caught. | closed |
| T-02-04-03 | Spoofing | A guard that passes because it scanned nothing | high | mitigate | Zero-guard on scanned-file count and examined-scalar count per file; a missing or unreadable file errors rather than returning empty. | closed |
| T-02-04-04 | Spoofing | A guard weakened until it matches nothing | medium | mitigate | Near-miss table pins the exact real strings that must NOT be flagged (`pnpm-lock.yaml`, `protoc-gen-es`), so weakening the matcher breaks the positive control instead of quietly passing. | closed |
| T-02-04-05 | Tampering | A JS toolchain entering via an edge the closure does not model | medium | accept | AR-03 | closed |
| T-02-04-06 | Tampering | A mutable CI cache producing a `node_modules` differing from the lockfile | medium | mitigate | `TestJSInstallPathHasNoMutableCache` forbids `actions/cache`, a JS-scoped cache action, and a `cache:` input on the Node setup action in `ci.yml`'s `test` job, while asserting the allowed Go cache was examined. Authored one wave **before** 02-06 added the Node step, so the step could not arrive with a cache attached. | closed |
| T-02-04-07 | Spoofing | The scanner's coverage model shrinking without its verdict changing | medium | mitigate | `errUnsupportedReachabilityEdge` refuses, naming unit and edge kind, on an unmodelled edge rather than scanning the remainder and reporting a clean zero. `TestUnsupportedReachabilityEdgeIsLoud` proves it fires on all five planted shapes and does not fire on the current closure. | closed |

### 02-05 — App shell and live status

| Threat ID | Category | Component | Severity | Disposition | Mitigation | Status |
|-----------|----------|-----------|----------|-------------|------------|--------|
| T-02-05-SC | Tampering | Tailwind/shadcn dependency install | high | mitigate | `strictDepBuilds: true` already in effect from 02-01 — an unreviewed lifecycle script in the new tree fails the install rather than running silently. `allowBuilds` remains `{}`. | closed |
| T-02-05-02 | Tampering | Registry-vendored component source bypassing `pnpm audit` | medium | accept | AR-04 | closed |
| T-02-05-03 | Tampering | XSS via a status field rendered into the DOM | low | mitigate | Svelte escapes interpolated text by default; **verified zero `{@html}` directives** anywhere under `web/src/`. | closed |
| T-02-05-04 | Information disclosure | Status panel exposing repository paths | low | accept | AR-05 | closed |
| T-02-05-05 | Denial of service | A never-resolving status call leaving a permanent loading state | low | mitigate | The call is wrapped with a `.catch` rendering a named `{ kind: 'error' }` state (`+page.svelte:12,23-25`) rather than an indefinite spinner. | closed |
| T-02-05-06 | Tampering | Project-generation tooling from a floating registry tag | high | mitigate | `pnpm dlx shadcn-svelte@1.5.0` with registry `dist.integrity` confirmed twice before execution (02-05-SUMMARY.md:116). CLI deliberately absent from `web/package.json`, so pin + integrity is its only provenance. | closed |
| T-02-05-07 | Spoofing | A status panel misreporting a locked store as "never indexed" | medium | mitigate | Re-index-in-progress selected from `indexing_in_progress` (field 9) directly rather than inferred from `store_exists && !initialized`, so the client cannot diverge from the server's degrade classification. | closed |

### 02-06 — CI build and drift guard

| Threat ID | Category | Component | Severity | Disposition | Mitigation | Status |
|-----------|----------|-----------|----------|-------------|------------|--------|
| T-02-06-01 | Tampering | A stale committed `web/build/` shipping code not matching its source | high | mitigate | `task web:drift` compares a source-tree digest against a committed marker, reports the hashed count first, watched RED against real staleness. | closed |
| T-02-06-02 | Spoofing | A drift guard passing because its enumeration matched nothing | high | mitigate | Count printed unconditionally before any comparison, plus a derived floor with an enumerated `::error::`, demonstrated by narrowing the enumeration until the floor fires. | closed |
| T-02-06-03 | Spoofing | A marker that survives a rebuild and passes vacuously | high | mitigate | The marker lives inside the directory `pnpm build` empties, so a rebuild that skips it leaves it absent; the absent-marker branch is a named failure and is demonstrated. | closed |
| T-02-06-04 | Tampering | Version skew between developer pnpm and CI's | medium | mitigate | `packageManager: "pnpm@11.23.0"` is the single source of truth; `web:deps` reports which resolution path it took and fails if the install rewrites the lockfile. | closed |
| T-02-06-05 | Elevation of privilege | An unpinned marketplace action in a per-PR job | medium | mitigate | `actions/setup-node@820762786026740c76f36085b0efc47a31fe5020 # v7.0.0` — full commit SHA with version comment, matching every other action in the workflow. | closed |
| T-02-06-06 | Tampering | A `node_modules` cache producing a tree differing from the lockfile | medium | mitigate | Upgraded from `accept` during plan review. No caching action added, AND 02-04's `TestJSInstallPathHasNoMutableCache` makes the omission a structural invariant rather than a fact about today. | closed |
| T-02-06-07 | Tampering | A hand-edited, added or removed file inside committed `web/build/` — the exact bytes `//go:embed all:build` ships | high | mitigate | The marker carries a second, independent output manifest (file list via `find`, so untracked strays count, plus per-file hashes, excluding the marker). Watched RED three ways — edited byte, added file, removed file — each with the SOURCE half reporting MATCH in the same run, which is the direct evidence a source-only marker would have passed. | closed |
| T-02-06-08 | Spoofing | A gate proving the marker matches source while never proving the toolchain can still build | medium | mitigate | `task web:build:verify` runs `pnpm build` into a scratch dir on every PR and asserts structural invariants on the fresh output, printing each observed number before comparing, then asserts the committed tree is untouched. Deliberately not byte-identity — see Audit Note 2. | closed |

### 02-07 — JS supply-chain gates

| Threat ID | Category | Component | Severity | Disposition | Mitigation | Status |
|-----------|----------|-----------|----------|-------------|------------|--------|
| T-02-07-SC | Tampering / EoP | Malicious npm lifecycle script during `pnpm install` | high | mitigate | `strictDepBuilds: true` makes pnpm exit non-zero on any unreviewed build script; `task web:deps:strict` asserts the setting's **parsed value** via a real YAML decoder, watched fail with the setting removed and with a real approval entry removed. | closed |
| T-02-07-02 | Tampering | Known-vulnerable JS dependency shipped inside committed `web/build/` | high | mitigate | `task web:audit` gates every PR inside the required `test` context; a sibling lockfile count proves the scan had real input. A live advisory (GHSA-pxg6-pf52-xh8x) was found and fixed by this gate during execution — see Audit Note 3. | closed |
| T-02-07-03 | Spoofing | A failed audit scan reading as a clean verdict | high | mitigate | Three-way classification on output shape with a distinct `SCAN ERROR` branch that fails the build (`Taskfile.yml:999`), plus an unrecognized-class branch that also fails closed (`Taskfile.yml:1009`). Demonstrated firing against an unreachable registry. | closed |
| T-02-07-04 | Spoofing | A gate coupled to an upstream warning string a reword silently disables | high | mitigate | BLD-05 asserts the parsed setting value with `go.yaml.in/yaml/v3`, **never** pnpm's "Ignored build scripts" warning text — prohibited explicitly and documented inline at `Taskfile.yml:445-462`. | closed |
| T-02-07-05 | Information disclosure | Documented security posture overstating coverage | medium | mitigate | Root `SECURITY.md` updated to state both scanners' scope, the boundary between them, and the registry-vendored-component gap the audit structurally cannot see. | closed |
| T-02-07-06 | Tampering | Registry-vendored shadcn-svelte source bypassing every scanner | medium | accept | AR-06 | closed |
| T-02-07-07 | Spoofing | A gate that cannot fire at all (no vulnerable input ever reaches it) | medium | accept | AR-07 | closed |
| T-02-07-08 | Tampering | A non-registry or integrity-free dependency source entering the lockfile | high | mitigate | `task web:lockfile` asserts the declared `lockfileVersion`, asserts integrity-bearing resolutions EQUAL total resolutions, and rejects `file:`, mutable git-branch and HTTP-tarball sources — each printing its observed count before comparing. Demonstrated RED three ways. `--frozen-lockfile` alone proved only that the file was not rewritten. | closed |
| T-02-07-09 | Spoofing | The phase's own BLD-03 gate left RED on the final commit | medium | mitigate | 02-07 edits a file inside `web:drift`'s hashed source set, so Task 3 ran `task web:drift` as the phase's last gate, refreshing the marker on a source-half mismatch and halting on an output-half mismatch rather than rebuilding it away. Verified green at HEAD. | closed |

*Status: open · closed · open — below high threshold (non-blocking)*
*Severity: critical > high > medium > low — only open threats at or above `workflow.security_block_on` (high) count toward `threats_open`*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| AR-01 | T-02-01-04 (low) | The embedded tree is the committed, public build output of an MIT-licensed repository. Nothing secret is embeddable here by construction, and the file-list diff test makes any unexpected addition visible. | Plan author, 02-01 | 2026-08-25 |
| AR-02 | T-02-02-05 (low) | Embedded assets are this repository's own committed build output, bounded at build time and reviewed in the diff. Phase 1's transport caps do not apply to the static path and do not need to. | Plan author, 02-02 | 2026-08-25 |
| AR-03 | T-02-04-05 (medium) | The closure resolves local composite actions and `task <target>` invocations from the roots automatically. Three of the four residual boundaries (shelled-out script, container image shipping Node, marketplace action outside the denylist) are TRIPWIRED by `errUnsupportedReachabilityEdge` (T-02-04-07). The residual accepted risk narrows to the **general-purpose-installer** boundary (`brew`, a piped `curl` install script), which is recognisable only by enumerating command spellings — refusing on it would be a second curated denylist rather than a structural rule. Enumerated in the plan's `<recorded_limitations>` and in `resolveReleasePathClosure`'s doc comment, with instructions for adding a new root. | Plan author, 02-04 | 2026-08-25 |
| AR-04 | T-02-05-02 (medium) | Registry-vendored component source enters outside the lockfile and so bypasses `pnpm audit`. **Not live in Phase 2** — zero components ship, asserted by a zero-entry check on the components directory. Becomes live at Phase 3's first `shadcn-svelte add`. Named in the SUMMARY and in 02-CONTEXT.md so Phase 3 inherits it rather than discovering it. | Plan author, 02-05 | 2026-08-25 |
| AR-05 | T-02-05-04 (low) | The listener is loopback-only and `originHostGuard` (Phase 1 SRV-02) rejects cross-origin and rebound-`Host` requests before any handler runs. The panel shows counts and a commit SHA, not paths. | Plan author, 02-05 | 2026-08-25 |
| AR-06 | T-02-07-06 (medium) | Same boundary as AR-04, viewed from the scanner side. Not live in Phase 2. Provenance pinning deliberately deferred; recorded in root `SECURITY.md` and 02-CONTEXT.md so Phase 3 inherits it named. | Plan author, 02-07 | 2026-08-25 |
| AR-07 | T-02-07-07 (medium) | D-15 declined a `vuln:selftest`-shaped pinned-advisory proof because a pinned JS advisory rots. The lockfile count proves the audit had real input but deliberately does **not** prove the detector fires. The phase's success language was narrowed to match — it claims clean-versus-scan-failed, not a proven three-way classification. Recorded in `<known_limitations>` with a per-branch table; revisitable. | Plan author, 02-07 | 2026-08-25 |

*Accepted risks do not resurface in future audit runs.*

**Phase 3 inherits AR-04 / AR-06 as live.** The first `shadcn-svelte add` crosses the
registry→committed-source boundary that `pnpm audit` structurally cannot see. This is the
single most important thing to carry out of this phase.

---

## Audit Notes

**1. A guard/guarded pair drifted mid-phase and was caught by the post-merge gate, not by
either plan.** 02-03 correctly raised `proto:drift`'s enumeration floor from 3 to 4, but the
paired assertion in `internal/upgrade/proto_task_test.go` still pinned 3, so
`TestProtoDriftGuardReportsAComparedCount` failed once the wave merged. Both plans verified
honestly; neither could see it from inside its own scope. Fixed in `1d2a1d2f`, with the guard
re-proven to discriminate (RED pinned to 5, GREEN pinned to 4). **Planning implication:**
`Taskfile.yml` and `proto_task_test.go` are a guard/guarded pair and belong in the same plan's
`files_modified`.

**2. `web:build:verify` is deliberately not a byte-identity check.** `web:build`'s *output*
digest is not stable across two consecutive runs with no source change (Vite content-hash
filename churn, confirmed directly during 02-06), while the *source* digest is stable. A guard
that rebuilt to compare would report drift on every run and be disabled within a week. This
asymmetry is why `web:drift` must never rebuild.

**3. The audit gate proved itself on a real advisory during execution.** 02-07's mandatory
empirical `pnpm audit --json` run surfaced GHSA-pxg6-pf52-xh8x (`cookie` <0.7.0, pulled in
transitively by `@sveltejs/kit@2.70.3`), resolved with a `pnpm-workspace.yaml` override. The
gate found a live vulnerability on its first real run rather than on a synthetic fixture.

**4. One code-review finding in this phase was security-adjacent and is fixed.** WR-01: CSP
hash extraction classified `<script>` blocks as external using an unanchored
`bytes.Contains(attrs, "src=")`, which also matches an attribute *name* ending in "src"
(`data-hydrate-src="1"`), silently dropping that script's hash from `script-src`. The failure
mode is unusual — it *strengthens* the policy, so no scanner fires; the browser simply refuses
to run the app's own bootstrap. Not reachable with the `index.html` committed today. Fixed in
`0d110682` with a boundary-anchored regex and a test that computes its expected hashes
independently of the function under test (the pre-existing guard called that function to build
its own expectation, so it could never have caught this class).

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-08-25 | 50 | 50 | 0 | /gsd-secure-phase (L1, register authored at plan time) |

**Method.** Register built from the `<threat_model>` blocks in all seven PLAN.md files
(`register_authored_at_plan_time: true`), so this run verified mitigations rather than scanning
for new threats. Verification was at ASVS L1 (grep depth) per `workflow.security_asvs_level`,
with `workflow.security_block_on: high`. Every `mitigate` disposition was confirmed against a
named control in the tree at commit `9d932507`; every `accept` disposition is logged above.
Because `threats_open: 0` with a plan-time register at L1, the workflow's short-circuit applied
and no separate auditor pass was spawned — **note that this short-circuit would not apply at
ASVS L2 or above**, where grep-depth classification is explicitly insufficient for
boundary-placement and end-to-end trace checks.

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-08-25
