---
phase: "02"
status: resolved
critical: 0
warning: 0
info: 1
reviewed_files: 30
depth: deep
---

# Phase 02 Code Review — SPA Toolchain, Embedded App Shell & JS Supply Chain

**Reviewed:** 2026-08-25T01:07:48Z
**Depth:** deep (cross-file: Go handler ↔ Go tests ↔ Taskfile guards ↔ CI wiring ↔ frontend source ↔ real embedded build output)

## Summary

This phase was reviewed against its own locked decisions (D-01…D-19), its plans' `must_haves.prohibitions`, and the seven SUMMARY.md deviation logs, with the real committed `web/build/` tree read alongside the code to confirm claims empirically (e.g. counting the actual `<script>`/`<style>` blocks in the shipped `index.html`, tracing `fs.Sub`'s real behavior in the Go standard library, and independently checking pnpm's `engine-strict` semantics rather than assuming they were vacuous).

The result is a narrow finding set. The SPA handler's three-way asset/route rule, the CSP fail-closed derivation, the two-class cache policy, and `ServeMux`-based RPC/SPA precedence are all implemented as decided and are exercised by tests that run against the real embedded tree and a real guarded server (`TestSPARPCPathReachesConnectHandler`, `TestSPAInheritsOriginHostGuard`) rather than synthetic fixtures. The supply-chain gates in `Taskfile.yml` (`web:deps`, `web:deps:strict`, `web:lockfile`, `web:audit`, `web:build:verify`, `web:drift`, `proto:drift`) all print a positive count before comparing, have their own non-vacuity tests planting the exact case they must detect (`TestReleasePathScanIsNonVacuous`, `TestMutableCacheScanIsNonVacuous`, `TestProtoDriftGuardReportsAComparedCount`), and none of the "always exits 0" or "grep pattern that can never match" failure modes the review brief asked me to hunt for were found — the two candidates I traced (`web/.npmrc`'s `engine-strict` with no root `engines:` field, and a possible off-by-one in `proto:drift`'s floor) both turned out to be real, functioning checks on closer inspection (the former enforces every *dependency's* declared `engines.node`, confirmed against pnpm's own `packageIsInstallable`/`checkPackage` source; the latter is independently pinned and cross-checked by `internal/upgrade/proto_task_test.go`).

The one Warning is a real, demonstrable logic gap in the CSP hash derivation's external-script detection, plus a test-independence gap that means nothing would catch it. The two Info items are a dead error-handling branch and an untested (but correctly implemented) fail-closed path — worth recording, not urgent.

## Narrative Findings (AI reviewer)

## Warnings

### WR-01: CSP inline-script/style detection uses an unanchored substring match, not an attribute-boundary check, and its own regression test is not independent of the bug

**File:** `internal/uiserver/spa.go:141-156` (the classification), `internal/uiserver/spa.go:121` (the style-attribute variant), `internal/uiserver/spa_test.go:498-536` (`TestSPACSPHashesCoverEmbeddedInlineScripts`)

**Issue:** `spaInlineBlockHashes` decides whether a `<script>`/`<style>` block is "external" (and therefore excluded from CSP hashing) by checking `bytes.Contains(bytes.ToLower(attrs), []byte("src="))` — a raw substring search over the tag's entire attribute text, not a check for an actual `src` attribute. The doc comment states the intent correctly ("checked for a `src=` attribute, which marks an EXTERNAL reference"), but the implementation doesn't enforce an attribute-name boundary. Any attribute whose *name* ends in `-src` (e.g. a hypothetical `data-hydrate-src="…"`) or whose *value* contains the literal text `src=` would also match, even though the tag carries no real `src` attribute and has real inline content that needs a hash source.

The same class of issue exists one level up: `spaHasInlineStyleAttrRE` (`(?i)\sstyle\s*=\s*['"]`) scans the **whole HTML document** for anything shaped like ` style="` — including inside the served `<script>` block's own JS text (e.g. `el.style="color:red"` or a template literal containing that substring) — not just genuine HTML attribute positions. A match there incorrectly triggers the `style-src 'self' 'unsafe-inline'` fallback even when no real style attribute independent of the script exists.

**Failure scenario:** A future SvelteKit/Vite bump (or an added `<script>` attribute of any kind whose name or value happens to contain `src=`) causes `spaInlineBlockHashes` to classify a real inline script as external and omit its hash from `script-src`. The resulting policy still says `script-src 'self'`, but `'self'` does not cover inline script *content* — only script *files* loaded from the same origin — so the browser blocks that script outright. The failure mode is fail-closed (breaks the app, doesn't open a hole), but it is silent: `TestSPACSPHashesCoverEmbeddedInlineScripts` derives its "expected" hash list by calling `spaInlineBlockHashes` directly on the same HTML — the exact function under test — so a bug in the extraction itself can never surface as a test failure. The doc comment's claim that the expectation is derived "INDEPENDENTLY … never from the handler's own policy" is true only with respect to the handler's cached `csp` field, not with respect to the extraction logic itself.

Not exploitable today: the real, committed `web/build/index.html` was checked directly and contains exactly one `<script>` block with no attributes at all (`attrs == nil`) and one inline `style="display: contents"` attribute on the root `<div>` — neither trips this edge case. This is a latent fragility, not a live bug.

**Fix:** Anchor the match to an actual attribute boundary, e.g. a regex requiring a preceding whitespace or tag-start and a word boundary before `src`: `(?i)(^|\s)src\s*=` applied only within the captured attribute text, or better, parse attributes with `regexp.MustCompile(`(?i)([a-zA-Z-]+)\s*=\s*"[^"]*"`)` and check the attribute *name* equals `src` exactly. Separately, give `TestSPACSPHashesCoverEmbeddedInlineScripts` (or a new unit test) a synthetic HTML fixture with an attribute like `data-hydrate-src="1"` and assert the script is still classified as inline/hashed — a genuinely independent check that doesn't call `spaInlineBlockHashes` to build its own expectation.

## Info

### IN-01: `fs.Sub` error branch in `Listen` is unreachable — `fs.Sub` never validates that the directory exists

**File:** `internal/uiserver/server.go:134-138`

```go
buildFS, err := fs.Sub(web.BuildFS, spaSubdirName)
if err != nil {
    _ = ln.Close()
    return nil, fmt.Errorf("uiserver: Listen could not derive SPA build sub-filesystem: %w", err)
}
```

**Issue:** `io/fs.Sub`'s own doc comment states plainly: *"Sub does not check if the directory currently exists."* Reading the stdlib source confirms this — `Sub` only returns a non-nil error when `!fs.ValidPath(dir)`, and `spaSubdirName` is the compile-time constant `"build"`, which is always a valid path. So this branch can never execute for any embedded tree shape, including a pathological one where `web/embed.go`'s `//go:embed all:build` somehow captured nothing. It reads as defensive error handling but provides no actual defense — if the embedded `build/` subtree were ever empty or absent, the real failure would surface later and more confusingly, inside `newSPAHandler`'s `fs.ReadFile` calls at request time, not here at bind time.

**Fix:** Not a functional bug (nothing currently relies on this branch firing), but the comment/code would be more honest either by removing the dead branch or by replacing it with an explicit `fs.Stat(web.BuildFS, spaSubdirName)` check that actually validates the directory exists before proceeding — turning a cosmetic error path into a real one, and giving a clearer failure message at bind time instead of a deferred 404-everything failure mode discovered only via manual testing.

### IN-02: The CSP fail-closed fallback path in `newSPAHandler` has zero test coverage

**File:** `internal/uiserver/spa.go:209-215`

```go
func newSPAHandler(fsys fs.FS) http.Handler {
	csp := spaCSPBaseDirectives + "; script-src 'self'; style-src 'self'"
	if indexHTML, err := fs.ReadFile(fsys, spaFallbackFile); err == nil {
		csp = buildSPACSPPolicy(indexHTML)
	}
	return &spaHandler{fsys: fsys, csp: csp}
}
```

**Issue:** This is exactly the fail-closed behavior the review's focus area 2 asked to verify — when the embedded tree lacks `index.html` at handler-construction time, the served policy is the strictest possible variant (no hash sources, no `unsafe-inline` anywhere), which is correct and is the right default. I could not find any test in `internal/uiserver/spa_test.go` that constructs a handler over an `fs.FS` missing `index.html` and asserts this branch is taken — every test in the file builds the handler over the real embedded `web.BuildFS` (via `newTestSPAHandler`), so the `err == nil` branch is the only one ever exercised.

**Fix:** Not a live defect — the code is correct as written and I verified the logic directly. Recorded because this is a security-relevant default with no regression test: a future refactor that accidentally inverted the condition, or changed the fallback to something more permissive "for convenience," would ship with every other test still green. A small test using `fstest.MapFS{}` (empty, no `index.html`) passed to `newSPAHandler` and asserting the resulting policy contains no `sha256-` source and no `unsafe-inline` would close this gap cheaply.

---

## Areas checked with no findings worth reporting

For completeness, given the depth requested, the following were traced and found correct — noted so this isn't mistaken for unreviewed territory:

- **D-10 three-way rule** (`internal/uiserver/spa.go:227-271`): asset-prefix-miss → 404 (never `index.html`), real-file hit → served as itself, everything else → `index.html`. No file-extension or dot-in-path heuristic anywhere; confirmed by reading the whole `ServeHTTP` body and cross-checking `TestSPAClientRouteFallsBackToIndex`'s dotted-path case (`/browse/internal/uiserver/spa.go`).
- **Path traversal**: `path.Clean` + `fs.ValidPath` rejects any path containing `..` or a leading slash before it reaches `fs.Stat`/`fs.ReadFile`; embed.FS itself cannot be escaped by `..` regardless. Traced several encoded/double-slash edge cases by hand; all resolve to the safe `serveFallback` branch, never a traversal.
- **Cache policy** (D-11): `spaCacheControlImmutable` only ever applied under `_app/immutable/`; `index.html` is unconditionally `no-store` on every path that reaches `serveFallback`, including the 404 path variants. Verified against the real build tree's asset paths.
- **RPC vs SPA precedence** (D-09): confirmed the test suite exercises the *guarded* server (`TestSPARPCPathReachesConnectHandler`, `TestSPAInheritsOriginHostGuard` both call `mustListen`/`srv.Serve` — real `originHostGuard`-wrapped mux), not a bare `http.ServeMux` — this was an explicit ask in the review brief and it holds.
- **Supply-chain gate non-vacuity**: `web:deps:strict`, `web:lockfile`, `web:audit`, `web:drift`, `proto:drift`, and `TestReleasePathHasNoJSToolchain`/`TestJSInstallPathHasNoMutableCache` all print a count before any pass/fail branch and have a sibling test that plants the exact violation shape and asserts detection (rule `84d1gfpywd` applied consistently). No gate found that could pass on an empty enumeration.
- **`cookie` advisory override** (`web/pnpm-workspace.yaml:38-39`): `overrides: { cookie@<0.7.0: ^0.7.2 }` lives in the committed `pnpm-workspace.yaml`, which pnpm re-reads and re-applies on every `pnpm install`, not baked only into the lockfile — it survives a lockfile refresh as required.
- **`web/.npmrc`'s `engine-strict=true`**: initially suspected vacuous (no `engines:` field in `web/package.json`), but confirmed via pnpm's own `packageIsInstallable`/`checkPackage` source (fetched from the pnpm.io docs) that `engine-strict` enforces every individually-resolved dependency's own declared `engines` field, not only the root manifest's — and the committed `pnpm-lock.yaml` shows dozens of dependencies with real `engines.node` constraints. This is a functioning check, not a no-op.
- **Frontend RPC error handling** (`web/src/routes/+page.svelte:17-30`): the one live `GetStatus` call has an explicit `.catch()` rendering a named error state; no unhandled promise rejection, no `{@html}`/`innerHTML` anywhere in the reviewed frontend files (checked via search across all `web/src` sources).
- **BLD-07 release-path scanner** (`internal/upgrade/taskfile_shape_test.go`): closure resolution is derived from `release.yml`'s own content (not a hardcoded file list), the four documented reachability gaps are each backed by an explicit tripwire test (`TestUnsupportedReachabilityEdgeIsLoud`) except the one boundary (general-purpose installers) the plan explicitly and correctly declines to model.

---

_Reviewed: 2026-08-25T01:07:48Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep_

---

## Resolution (orchestrator, post-review)

| Finding | Disposition | Commit |
|---------|-------------|--------|
| WR-01 — unanchored `src=` substring match in `spaInlineBlockHashes` | **Fixed.** Replaced with `spaExternalSrcAttrRE`, anchored to an attribute boundary. New test computes expected hashes independently of `cspHashSource`/`spaInlineBlockHashes` and was verified RED against the original check, GREEN against the fix. | `0d110682` |
| IN-02 — CSP fail-closed fallback untested | **Fixed.** `TestNewSPAHandlerFailsClosedWithoutIndexHTML` asserts the no-`index.html` path still emits a policy, that all four directives are `'self'`, and that neither `'unsafe-inline'` nor `'unsafe-eval'` appears. | `0d110682` |
| IN-01 — unreachable `fs.Sub` error branch in `server.go` | **Accepted, not changed.** The branch is dead against `io/fs.Sub`'s current behavior, but it is defensive error handling on a call that returns an `error`; deleting it would have to be re-added if the subdirectory name ever became non-constant. Left in place deliberately. | — |

Verified after the fix: `go vet ./...` clean, `go test -count=1 ./internal/uiserver/...` green (10.97s, uncached).
