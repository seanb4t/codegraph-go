# Phase 3: Browse, Inspect & Navigation - Research

**Researched:** 2026-08-28
**Domain:** SvelteKit SPA client (search/browse/navigation UX, URL-as-state, syntax highlighting) over an existing Connect RPC surface; one new Go RPC (`GetPermalink`) plus one new regression test proving reuse of existing path confinement.
**Confidence:** HIGH — the phase's hard decisions were already locked by `/gsd-discuss-phase` (`03-CONTEXT.md`, D-01..D-22) with file:line citations against the real tree. This document verifies those citations, corrects one factual error found in the verification pass (the indexed-language count), and answers the concrete "how do I implement this in the installed toolchain" questions the planner needs that discussion did not cover: highlight.js's exact API for selective registration, SvelteKit's exact API for URL-as-state (which turned out to be `goto()`, not the confusingly-named `pushState`/`replaceState` shallow-routing exports 03-CONTEXT.md's prose evokes), and shadcn-svelte's `Command` primitive for the search UI.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

D-01 through D-22, copied verbatim from `03-CONTEXT.md`'s `<decisions>` section — reproduced here as headings only, with the planner directed to the full text in `03-CONTEXT.md` (reproducing all ~600 lines verbatim here would duplicate rather than usefully summarize; every citation in the Locked Decisions below was independently re-verified against the tree during this research session — see Sources):

- **Source Serving & Path Confinement (SRV-05, NAV-04):** D-01 (endpoint-need enumerated, not assumed from wording — `GetNodeDetail`'s `file` field is the only client-steerable path input); D-02 (SRV-05's deliverable is the missing RPC-boundary regression test, with a positive-control pass per rule `84d1gfpywd`); D-03 (stale is split from the source-pane collapse — `GetStatusResponse.stale` already on the wire, no proto change); D-04 (shared status gate + shared Connect error mapper live in `web/src/lib/`, not per-view — `GetStatus` is the only RPC that answers when degraded); D-05 (status gate fetches on load/navigation only, no polling loop — Phase 6 replaces this).
- **GitHub Permalinks (BRW-09):** D-06 (served by a new `GetPermalink` RPC — `(path, line, end_line?) -> {url, availability, reason}` — not data fields; maintainer-proposed, additive method); D-07 (unpushed-commit hint in scope, `availability` is three-valued: linkable / linkable-but-possibly-unpushed / no-link — `git branch -r --contains` proves presence, never proves absence); D-08 (GitHub only — every other forge resolves to explicit `no-link` with a reason, no config mechanism exists for self-hosted mapping); D-09 (range link when `end_line` known, single-line otherwise — `Location` deliberately not extended with `end_line`).
- **URL State & History (NAV-01, NAV-02):** D-10 (query params named after `GetNodeDetailRequest`'s own fields — `symbol`/`file`/`line`/`depth`/`limit` on the existing `/browse` route; identity mapping cannot drift, and query values are inert to Phase 2's dotted-path SPA-fallback exclusion); D-11 (navigation pushes history, refinement replaces it — pushState for opening/clicking/picking, replaceState for typing/adjusting); D-12 (client parses shape only, server validates values — no client-side range clamping, unknown params ignored not rejected); D-13 (one shared parse/serialize module in `web/src/lib/`, defining only the params this phase uses).
- **Search Surface (BRW-01, BRW-08, NAV-03):** D-14 (`Search`/`Files` fire live as-you-type; `Explore` fires on Enter only, rendering alongside — `Explore` attaches a `SourceBlob` per matched file, ~10x `Search`'s cost; `Explore` empty is a successful response, never an error); D-15 (two labelled sections — Symbols then Files — each preserving its own RPC's order; no client-side cross-ranking; Explore appears as a third section on submit); D-16 (~150ms debounce, 2-character minimum, `AbortController` mandatory — loopback server, low end of the range applies); D-17 (both `/` and `Cmd`/`Ctrl`+`K` focus search, `Esc` dismisses — `/` suppressed while an input has focus).
- **Source Rendering & Navigation (BRW-04, BRW-05, BRW-06):** D-18 (click-to-definition resolves from clicked text, only identifiers matching a name in the node's `calls` list are clickable — server-side resolved reference spans are structurally impossible this phase, the Pebble edge key omits line/col); D-19 (highlight.js with selective registration — `lib/core` plus the indexed languages; bundle is committed to git AND embedded in the signed binary, so size is a real supply-chain constraint, not a preference — **this research corrects the exact language count from 12 to 14 registered IDs / 13 hljs modules, see Summary**); D-20 (a truncated file says so plainly and offers the BRW-09 permalink as the route to the rest — no pagination RPC, `total_lines`/`returned_lines` already on the wire); D-21 (BRW-05's picker lists all `total_candidates`, visually marking `detail_gathered = false` entries — no auto-picking a "best" candidate, nothing on the wire ranks candidates).
- **Supply Chain:** D-22 (Phase 3 inherits Phase 2's vendored-shadcn-svelte-component gap and makes it live for the first time — mitigation is scope discipline: add only what Browse genuinely needs, review vendored source at add time; registry-version pinning with source-match assertion was considered and declined, file a todo instead).

### Claude's Discretion

- Exact panel layout of the Browse view — how source, callers, callees and blast radius share the screen.
- BRW-07's copy affordance (button vs. icon vs. click-to-copy) and which of file path / symbol name / qualified name are copyable.
- How `Impact`/`Affected` depth is surfaced for blast radius, within D-10's `depth` param and D-12's pass-through validation.
- Precise `AbortController` wiring and whether a request-id guard backs it up.
- Whether the status banner from D-04 is dismissible.
- Exact `GetPermalink` field names and whether `availability` is an enum or a string, provided it is three-valued per D-07.
- Which shadcn-svelte components are added, subject to D-22's minimality.
- **This research's addition to the discretion set:** whether to adopt `vitest` for the phase's new pure-TS modules, given `web/` currently has zero JS test infrastructure (see Validation Architecture, Wave 0 Gaps) — not a CONTEXT.md discretion item originally, but a real open decision this research surfaced that the plan must resolve one way or the other, not silently skip.

### Deferred Ideas (OUT OF SCOPE)

- Server-provided syntax-highlight spans / resolved reference ranges — blocked by the Pebble edge key's omission of line/col (D-18); revisit only if the storage key shape changes.
- A ranged/paginated source fetch so a truncated file can be read fully in-UI (D-20) — declined as a second bounded-read path.
- `end_line` on `Location`, which would make every permalink a range (D-09) — declined as the wrong trade for the shared five-RPC message.
- Registry-version pinning with a committed-source match assertion for shadcn-svelte components (D-22) — declined by Phase 2, re-declined here; filed as a todo.
- Additional forges for BRW-09 (GitLab, Bitbucket, Codeberg, Gitea) and the configurable host mapping self-hosted instances would need (D-08) — blocked on this project having any config mechanism at all.
- A vulnerable-JS-fixture red proof for BLD-06, mirroring `task vuln:selftest` — carried forward unresolved from Phase 2's D-15, not this phase's job to close.
- Two folded todos ride along per `03-CONTEXT.md` §Folded Todos: golangci-lint (third fold, must land as an actual verifiable task this time, not a mention) and the wire-oracle `toolslist-repeat` ordering flake (unrelated, must not gate phase completion).

</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-------------------|
| BRW-01 | User can search symbols and files as they type | Pattern 4 (`Command` w/ `shouldFilter={false}`), D-14/D-15/D-16 confirmed against `validate.go`/`explore.go` |
| BRW-02 | User can open a node and see verbatim source, callers, callees, and blast radius | Existing `GetNodeDetail` three-mode response (verified `ui.proto`); no new RPC needed |
| BRW-03 | User can click any neighbor and continue navigating from there | Pattern 1 (`goto()`-driven URL state) + Pattern 2 (reactive param derivation) |
| BRW-04 | User can jump from a symbol reference in rendered source to its definition | D-18 confirmed — `Edge` key line/col omission verified at `graph.proto:70-95`; client-side name-match against `calls` |
| BRW-05 | User is offered a disambiguation picker when a bare symbol name resolves to multiple definitions | D-21 confirmed — `total_candidates`/`detail_gathered` fields verified in `ui.proto` and `handlers.go`'s `nodeDetailToProto` |
| BRW-06 | Source is syntax-highlighted using a lightweight highlighter with only the indexed languages registered | Standard Stack + Pattern 3 + Pitfall 2 (the 12→14 language correction) — the phase's most substantive research finding |
| BRW-07 | User can copy a file path or symbol name in one action | Claude's Discretion — no research blocker, standard clipboard API |
| BRW-08 | User can enter a natural-language query and get `Explore`'s relevance-selected results, alongside exact-name search | D-14/D-15 confirmed — `Explore`'s per-file `SourceBlob` cost and `empty=true` non-error contract verified in `ui.proto` |
| BRW-09 | User can open the current file/line on GitHub, permalinked to the indexed commit | Code Examples (`GetPermalink` proto sketch, pushed-commit check) — D-06/D-07/D-08/D-09 confirmed against `commit.go`/`worktree.go`/`Location`'s doc comment |
| NAV-01 | Every view, symbol, and query state is addressable by a shareable URL encoding view, target, depth and limit | Pattern 1/2, D-10/D-13 confirmed — query-string-vs-Path-only routing verified at `spa.go:222-224` (Pitfall 3) |
| NAV-02 | Browser back and forward navigate view history correctly | Pattern 1 + Pitfall 1 (the `pushState`/`replaceState`-vs-`goto()` correction — the phase's other substantive finding) |
| NAV-03 | User can drive search and result selection from the keyboard, including a focus shortcut and `Esc` to dismiss | Pattern 4 (`Command`/`Command.Dialog` built-in keyboard nav + Esc), D-17 confirmed |
| NAV-04 | No-index, stale-index, and symbol-not-found each render an explicit state rather than an empty pane | D-04 confirmed — `degrade.go`'s `classifyDegrade`/`degradedStatus` and `handlers.go`'s `mapEngineError` fully re-verified this session |
| SRV-05 | Verbatim source serving reuses the existing MCP path-confinement fix rather than reimplementing it | D-01/D-02 confirmed — `node.go:29-90`'s `resolveSourcePath` re-verified; Validation Architecture names the missing RPC-boundary test |

</phase_requirements>

## Summary

Phase 3 is almost entirely a client build against an RPC surface Phase 1 already shipped and live-verified, mounted on an app shell Phase 2 already shipped. `03-CONTEXT.md` (Ready for planning, dated 2026-08-28) already resolved every architecturally significant question — path confinement reuse, the source/stale/not-found three-state degrade contract, the `GetPermalink` RPC shape, URL query-param naming and push/replace semantics, the search-surface trigger split (`Search`/`Files` live, `Explore` on Enter), and the syntax-highlighter choice. This research verified every file:line citation in that document against the actual tree (all held) and did the toolchain-level investigation discussion does not do: it pins down the *exact installed-version API* for the two riskiest "sounds obvious, isn't" spots — SvelteKit URL-state updates and highlight.js selective registration — and it corrects one factual claim.

**The correction:** D-19 states "the indexed set is c, cpp, csharp, go, java, kotlin, php, python, ruby, rust, swift, typescript" (12 languages). Reading `internal/indexer/languages_*.go` directly shows **14** registered `LanguageSpec.ID` values — `languages_typescript.go` registers three separate IDs (`typescript`, `tsx`, `javascript`) sharing one extractor, not one. The highlight.js registration set must therefore cover `javascript` too (as its own hljs module) — although `tsx` needs no separate hljs module, because hljs's own `typescript` grammar already carries `tsx` as a declared alias. Net effect: register 13 hljs modules, not 12, to properly cover all 14 emitted `Node.language` values. Everything else in D-19 (highlight.js over Prism/starry-night, `lib/core` + selective `registerLanguage`, bundle-as-supply-chain-artifact framing) is independently confirmed and strengthened by fresh registry data below.

**The other load-bearing finding:** SvelteKit's `$app/navigation` exports two *different* mechanisms that both touch history, and D-11's "pushState... replaceState..." prose is ambiguous between them. The installed `@sveltejs/kit@2.70.3`'s own `pushState`/`replaceState` functions are for **shallow routing** — they update `page.state` and the address bar, but do **not** update the reactive `page.url` (`$app/state`) the rest of the app reads. For NAV-01/NAV-02 to work — a URL-driven view that also updates on back/forward — the correct primitive is `goto(url, { replaceState: <bool>, noScroll: true, keepFocus: true })`, whose own `replaceState` boolean option is the real lever D-11 needs. This is verified by reading the installed package's source directly (below), not by trusting either the SvelteKit docs' naming or 03-CONTEXT.md's prose.

**Primary recommendation:** Build the Browse view as one SvelteKit route reading state exclusively from `page.url.searchParams` (via `$app/state`), writing state exclusively through `goto()` with its `replaceState` option (never the shallow-routing `pushState`/`replaceState` functions), rendering search with shadcn-svelte's `Command` primitive (`shouldFilter={false}`, server-driven), and highlighting source with `highlight.js/lib/core` plus the 13-module registration list in this document.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Symbol/file search-as-you-type, Explore-on-Enter | API / Backend (`Search`/`Files`/`Explore` RPCs) | Browser / Client (debounce, abort, render) | Ranking and result production already live in `internal/query.Engine`; the client's only job is triggering the right RPC at the right time and rendering server order (D-14, D-15, D-16) |
| Verbatim source + syntax highlighting | Browser / Client (highlight.js) | API / Backend (bounded read + truncation) | Server owns bytes and bounds (RPC-05, `truncate.go`); client owns rendering — highlighting is presentation, not data |
| Click-to-definition resolution | Browser / Client (name-match against `calls`) | API / Backend (`GetNodeDetail` re-fetch) | No server-side resolved spans exist (Edge key omits line/col — D-18); client does text-to-name matching against data already on the wire, then re-queries |
| Disambiguation picker (BRW-05) | API / Backend (`total_candidates`, `detail_gathered`) | Browser / Client (render) | Server already tracks and exposes the true/gathered split; client is a renderer, not a decision-maker |
| URL state (view/target/depth/limit) | Browser / Client (SvelteKit router) | — | No server involvement; URL parse/serialize is pure client logic (D-13) |
| GitHub permalink derivation | API / Backend (new `GetPermalink` RPC) | — | Browser cannot read `.git/config`; owner/repo/remote-normalization/pushed-state logic must be server-side and unit-testable (D-06) |
| Path confinement for source reads | API / Backend (`internal/query.resolveSourcePath`, reused) | — | Already shared between MCP and uiserver; SRV-05 adds a test at the RPC boundary, not new logic (D-01, D-02) |
| Degrade/error classification (no-index/stale/not-found) | API / Backend (`GetStatus` answer + Connect error codes) | Browser / Client (shared status gate + error mapper) | Server already classifies (`degrade.go`); client needs one shared translation layer so four future phases don't reinvent it (D-04) |

## Standard Stack

### Core (already installed — Phase 2)

| Library | Version (installed) | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `@sveltejs/kit` | 2.70.3 (`^2.63.0` in `package.json`) `[VERIFIED: web/node_modules/@sveltejs/kit/package.json]` | SPA router, `goto`/`pushState`/`replaceState` | Already the project's framework; confirmed installed version exports both mechanisms distinctly (see Pitfalls) |
| `svelte` | `^5.56.1` | Component runtime (runes) | Already installed; `$app/state`'s `page` is rune-based reactive state, confirmed in use at `web/src/routes/+layout.svelte:5,20-22` `[VERIFIED: web/src/routes/+layout.svelte:5,20-22]` |
| `@sveltejs/adapter-static` | `^3.0.10` | Static SPA build with `fallback: 'index.html'` | Already configured (`web/vite.config.ts`); `ssr=false`/`prerender=false` set at `web/src/routes/+layout.ts:1-3` `[VERIFIED: web/src/routes/+layout.ts, web/vite.config.ts]` |
| `@connectrpc/connect-web` | `2.1.2` | Connect RPC transport | Already installed and proven live (D-19 `GetStatus` call) |

### New for Phase 3

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `highlight.js` | `11.12.0` `[VERIFIED: npm registry — npm view highlight.js version, dist-tags.latest]` | Syntax highlighting, selective per-language registration | D-19's locked choice. Registry-confirmed: latest release 2026-08-12 (16 days before this research), 34.8M weekly downloads, no `postinstall` script `[VERIFIED: npm view highlight.js scripts.postinstall — empty]`. Compare Prism (`prismjs@1.30.0`, last published 2025-03-10 — ~17 months stale as of this research) `[VERIFIED: npm registry — both dist-tags.latest and time.latest for both packages]` |
| shadcn-svelte `command` + `popover` components | via `shadcn-svelte@1.5.1` CLI (installed dev tool) `[VERIFIED: npm view shadcn-svelte version]` | Search input + results list with built-in keyboard nav | Built on Bits UI's `Command` primitive (cmdk-sv successor); ships arrow-key/Enter navigation, `Command.Dialog` gives Esc-to-close for free `[CITED: bits-ui.com/docs/components/command via WebSearch; huntabyte/shadcn-svelte docs via Context7]` |

### Recommended, not yet decided (Claude's Discretion per CONTEXT.md)

| Library | Version | Purpose | When to add |
|---------|---------|---------|-------------|
| `vitest` | `4.1.11` `[VERIFIED: npm view vitest version, peerDependencies]` | Unit tests for the phase's new pure-TS modules (URL parse/serialize, error mapper, click-to-definition name matcher) | `web/` currently has **zero** test files and no test framework (`find web -iname '*.test.*' -o -iname '*.spec.*'` returns nothing excluding `node_modules`; no `vitest`/`playwright`/`@testing-library` in `package.json`) `[VERIFIED: web/package.json — full devDependencies list read; find web -iname "*.test.*"]`. Vitest's peer range `^6.0.0 \|\| ^7.0.0 \|\| ^8.0.0` covers this repo's installed `vite@^8.0.16` `[VERIFIED: npm view vitest peerDependencies]`. See Validation Architecture |

### Alternatives Considered (already decided against — recorded for completeness)

| Instead of | Could use | Tradeoff | Verdict |
|------------|-----------|----------|---------|
| highlight.js | Prism.js | Smaller per-language footprint (~9KB total claim, unverified this session) but v1 in maintenance, v2 alpha-stalled | Rejected by D-19; maintenance story matters more given the bundle is a signed-binary artifact |
| highlight.js | starry-night | Uses the same TextMate grammars VS Code uses (higher fidelity) | Rejected by D-19: 185KB **plus a WASM binary** before any grammars — a supply-chain conversation this phase does not need |
| highlight.js | Shiki | VS Code-quality theming | Rejected at roadmap time (phase description explicitly: "LIGHTWEIGHT, NOT SHIKI") |
| `$app/navigation` `pushState`/`replaceState` | `goto(url, {replaceState, noScroll, keepFocus})` | The former updates `page.state`, not `page.url`; the latter is what NAV-01/02 actually need | See Common Pitfalls — this is not a style choice, the former will not work for URL-driven view state |

**Installation:**
```bash
cd web
pnpm add highlight.js@11.12.0
pnpm dlx shadcn-svelte@latest add command popover
# Discretion: pnpm add -D vitest@4.1.11 (if the plan adopts JS unit tests — see Validation Architecture)
```

**Version verification performed this session:** `npm view highlight.js version` → `11.12.0`; `npm view highlight.js dist-tags` → `latest: 11.12.0`; `npm view prismjs version` → `1.30.0`; registry `time.latest` compared for both (highlight.js 2026-08-12 vs prismjs 2025-03-10); `npm view vitest version/peerDependencies` → `4.1.11`, `vite: ^6.0.0 || ^7.0.0 || ^8.0.0`; `npm view shadcn-svelte version` → `1.5.1` (already the version D-22's Phase 2 precedent used).

## Package Legitimacy Audit

Both new packages triggered `too-new` from the legitimacy heuristic (it flags recency of the *latest publish*, not package age) despite both being large, established, actively-maintained projects. Documented per protocol rather than silently overridden.

| Package | Registry | Latest publish | Weekly downloads | Source Repo | Verdict | Disposition |
|---------|----------|-----------------|-------------------|-------------|---------|-------------|
| `highlight.js` | npm | 2026-08-12 (16 days before this research) | 34,793,164 | `github.com/highlightjs/highlight.js` | `[SUS]` (reason: `too-new`) | Flagged — planner must add `checkpoint:human-verify` before install |
| `vitest` | npm | 2026-08-18 (10 days before this research) | 98,448,132 | `github.com/vitest-dev/vitest` | `[SUS]` (reason: `too-new`) | Flagged — planner must add `checkpoint:human-verify` before install (only if the plan adopts it) |

`[VERIFIED: gsd-tools query package-legitimacy check --ecosystem npm highlight.js vitest]` — both signals object show `postinstall: null` (no lifecycle script risk) and `deprecated: false`.

**Mitigating context (not a verdict override, informational for the human checkpoint):** both packages have download counts in the tens of millions per week and multi-year GitHub histories under their stated `repoUrl`. The `too-new` flag is an artifact of a recent minor/patch release, not a signal of a new or hijacked package. The human checkpoint should confirm this reasoning rather than re-deriving it from scratch.

**Packages removed due to `[SLOP]` verdict:** none.
**Packages flagged as suspicious `[SUS]`:** `highlight.js`, `vitest` — both require `checkpoint:human-verify` per protocol despite the mitigating context above.

`shadcn-svelte` itself is a **CLI dev-dependency** (`pnpm dlx`, not committed to `pnpm-lock.yaml` per D-22/Phase 2's Known Limitation), so the package-legitimacy gate does not apply to it the same way — its output (`.svelte` component source) is reviewed as committed code at add-time, per D-22.

## Architecture Patterns

### System Architecture Diagram

```
Browser (SvelteKit SPA, client-side routed, no SSR)
│
│  1. GET /browse?symbol=Foo&file=...&line=42&depth=3&limit=50
│     (or any other query-param combination D-13's module parses)
▼
Go binary: internal/uiserver's http.ServeMux
│
├─ originHostGuard (SRV-02) — Host/Origin exact-match, outermost handler
│
├─ path under "/codegraph.ui.v1.UIService/..." → Connect RPC handlers
│  (most-specific-pattern-wins, Go 1.26 ServeMux — D-09, Phase 2)
│
└─ everything else, including "/browse?..." (query string is NOT part
   of r.URL.Path, confirmed by direct read of spa.go's routing —
   see Pitfalls) → spaHandler.ServeFallback → index.html, no-store
       │
       ▼
   Browser boots SvelteKit router client-side, resolves "/browse" route,
   +page.svelte reads page.url.searchParams (via $app/state) reactively
       │
       ▼
   uiClient (Connect-ES, JSON encoding, baseUrl "/") issues RPCs:
     - Search / Files  → live, as-you-type, debounced+aborted (D-16)
     - Explore         → on Enter only (D-14) — reads files off disk,
                          ~10x cost of Search
     - GetNodeDetail   → on open / on click-through / on picker select
     - GetPermalink    → NEW rpc, on "open on GitHub" click (D-06)
       │
       ▼
   Go handler → withEngine (opens/snapshots/closes Engine per call,
   SRV-04) → internal/query.Engine → graphstore (Pebble) → response
   truncated (truncate.go) → mapped to uiv1 proto → Connect JSON
       │
       ▼
   Client renders: two-section live results (Symbols, Files) + Explore
   section on submit; source pane (highlight.js) + callers/callees/
   blast-radius panels; every state change → goto(url, {replaceState})
   updates the address bar AND page.url reactively, so a re-render is
   driven by the SAME reactive read the initial page load used
       │
       ▼
   Browser back/forward → SvelteKit's normal router history stack
   (NOT the shallow-routing pushState/replaceState mechanism) →
   page.url updates → view re-derives from query params, same as a
   fresh load of that URL
```

### Recommended Project Structure

```
web/src/
├── lib/
│   ├── client.ts              # existing — Connect transport (Phase 2)
│   ├── gen/ui_pb.ts           # existing — generated, regenerate after
│   │                            adding GetPermalink to ui.proto
│   ├── status.ts              # NEW (D-04) — shared status-gate store:
│   │                            fetches GetStatus on load/navigation,
│   │                            drives a view-level banner
│   ├── rpc-errors.ts          # NEW (D-04) — Connect error → named UI
│   │                            state mapper (CodeUnavailable+
│   │                            IndexingInProgress, CodeNotFound,
│   │                            CodeInvalidArgument)
│   ├── browse-url.ts          # NEW (D-13) — parse/serialize for
│   │                            symbol/file/line/depth/limit query
│   │                            params; the ONE place Phase 4 extends
│   ├── highlight.ts           # NEW (D-19/BRW-06) — highlight.js
│   │                            lib/core + 13-module registration
│   │                            (see Code Examples)
│   └── components/ui/
│       ├── command/           # NEW — shadcn-svelte-added
│       └── popover/           # NEW — shadcn-svelte-added
└── routes/
    └── browse/
        └── +page.svelte       # fills the D-18 placeholder — this
                                  phase's entire UI surface
```

### Pattern 1: URL-as-state via `goto()`, never shallow-routing `pushState`/`replaceState`

**What:** Every view/target/depth/limit change is expressed as a `goto()` call against the current route with an updated `URLSearchParams`, using `goto`'s own `replaceState` boolean option to choose push-vs-replace history semantics.
**When to use:** Any state change NAV-01/NAV-02 must make shareable and back/forward-walkable — opening a node, clicking a neighbor, picking a disambiguation candidate (push); typing in search, adjusting depth/limit (replace), per D-11.
**Example:**
```typescript
// Source: installed @sveltejs/kit@2.70.3 source
// (web/node_modules/@sveltejs/kit/src/runtime/client/client.js:2311-2342),
// cross-checked against Context7 /sveltejs/kit docs.
import { goto } from '$app/navigation';
import { page } from '$app/state';

function updateBrowseParams(next: Record<string, string | undefined>, opts: { push: boolean }) {
	const url = new URL(page.url.href); // mutable copy — page.url itself is readonly
	for (const [key, value] of Object.entries(next)) {
		if (value === undefined) url.searchParams.delete(key);
		else url.searchParams.set(key, value);
	}
	goto(url, {
		replaceState: !opts.push, // D-11: push for navigation, replace for refinement
		noScroll: true,
		keepFocus: true, // keeps the search input focused while typing (D-16)
		invalidateAll: false // no `load` function depends on these params —
		//                     the +page.svelte itself derives state from
		//                     page.url.searchParams reactively; no reload needed
	});
}

// Navigation (push): opening a node, clicking a neighbor, picking a picker entry
updateBrowseParams({ symbol: 'Foo', file: undefined, line: undefined }, { push: true });

// Refinement (replace): typing in search, adjusting depth/limit
updateBrowseParams({ q: inputValue }, { push: false });
```

### Pattern 2: Reactive param derivation, never a `load` function

**What:** `+page.svelte` derives every piece of view state from `page.url.searchParams` directly (via a `$derived`), rather than a SvelteKit `load` function.
**When to use:** This app is `ssr=false`/`prerender=false` (pure SPA, Phase 2's `+layout.ts`); a `load` function here would add a re-run cycle for no benefit, since the RPC calls are already client-only and `goto()`'s reactive `page.url` update is sufficient to re-derive.
```typescript
import { page } from '$app/state';
import { parseBrowseParams } from '$lib/browse-url';

let params = $derived(parseBrowseParams(page.url.searchParams));
// D-12: parses SHAPE only (is depth an integer?), never validates RANGE —
// the server's validateLimit/MaxLimit/validateFilesDepth already do that,
// and an out-of-range value surfaces as a CodeInvalidArgument the shared
// error mapper (D-04) turns into a named UI state, not a client-side
// silent clamp.
```

### Pattern 3: Selective highlight.js registration

**What:** Import `highlight.js/lib/core` plus exactly the language modules the indexer can produce, register each explicitly, never import the full `highlight.js` package or `highlight.js/lib/common`.
**When to use:** Every source-rendering call site (single-def, file-mode, multi-def candidate, Explore group).
```typescript
// Source: highlight.js README pattern, confirmed against the ACTUAL
// installed npm package's registerLanguage implementation
// (unpkg.com/highlight.js@11.12.0/lib/core.js:2405-2422, fetched and
// read this session) and against languages_*.go's real registered IDs
// (internal/indexer/languages_c.go:44, languages_cpp.go:17,
// languages_csharp.go:85, languages_go.go:24, languages_java.go:87,
// languages_kotlin.go:47, languages_php.go:110, languages_python.go:70,
// languages_ruby.go:51, languages_rust.go:91, languages_swift.go:60,
// languages_typescript.go:130,140,150 — all read this session).
import hljs from 'highlight.js/lib/core';

import c from 'highlight.js/lib/languages/c';
import cpp from 'highlight.js/lib/languages/cpp';
import csharp from 'highlight.js/lib/languages/csharp';
import go from 'highlight.js/lib/languages/go';
import java from 'highlight.js/lib/languages/java';
import javascript from 'highlight.js/lib/languages/javascript'; // CORRECTION: not in D-19's 12
import kotlin from 'highlight.js/lib/languages/kotlin';
import php from 'highlight.js/lib/languages/php';
import python from 'highlight.js/lib/languages/python';
import ruby from 'highlight.js/lib/languages/ruby';
import rust from 'highlight.js/lib/languages/rust';
import swift from 'highlight.js/lib/languages/swift';
import typescript from 'highlight.js/lib/languages/typescript'; // covers 'tsx' too (own alias)

// Registration name MUST equal the Node.language value the wire sends
// (schema.Node.Language, populated from LanguageSpec.ID —
// internal/indexer/discover.go:201 — read this session) so a lookup by
// that exact string succeeds.
hljs.registerLanguage('c', c);
hljs.registerLanguage('cpp', cpp);
hljs.registerLanguage('csharp', csharp);
hljs.registerLanguage('go', go);
hljs.registerLanguage('java', java);
hljs.registerLanguage('javascript', javascript);
hljs.registerLanguage('kotlin', kotlin);
hljs.registerLanguage('php', php);
hljs.registerLanguage('python', python);
hljs.registerLanguage('ruby', ruby);
hljs.registerLanguage('rust', rust);
hljs.registerLanguage('swift', swift);
hljs.registerLanguage('typescript', typescript);
// NOTHING registered for 'tsx': highlight.js's typescript module declares
// aliases: ['ts','tsx','mts','cts'] (unpkg.com/highlight.js@11.12.0/
// lib/languages/typescript.js:902-910, fetched and read this session),
// and registerLanguage() auto-registers a module's own aliases
// (core.js:2421-2423: `if (lang.aliases) registerAliases(...)`) — so
// 'tsx' already resolves once 'typescript' is registered. Do NOT add a
// 14th import for it.

export function highlightSource(code: string, language: string): string {
	if (hljs.getLanguage(language)) {
		return hljs.highlight(code, { language }).value; // pre-escaped HTML —
		//     safe for {@html} ONLY because v11's default has NO HTML
		//     pass-through plugin loaded (see Security Domain)
	}
	return hljs.highlightAuto(code).value; // graceful fallback, never a crash,
	//     for any Node.language value not in this list (defense in depth)
}
```

### Pattern 4: shadcn-svelte `Command` for server-driven search

**What:** `Command.Root shouldFilter={false}` disables the primitive's built-in client-side re-filtering, so typed input drives server RPCs instead of re-filtering an already-server-ranked result set.
**When to use:** BRW-01's search-as-you-type surface; D-15's two-section (Symbols, Files) layout maps directly onto two `Command.Group`s, each preserving its own RPC's order (no client cross-ranking).
```svelte
<!-- Source: bits-ui.com/docs/components/command (CITED via WebSearch,
     official docs page), cross-checked against huntabyte/shadcn-svelte's
     Context7-served command.md examples for the Root/Input/List/Group/
     Item/Empty composition shape. -->
<Command.Root shouldFilter={false}>
	<Command.Input bind:value={query} placeholder="Search symbols and files..." />
	<Command.List>
		<Command.Empty>No results.</Command.Empty>
		<Command.Group heading="Symbols">
			{#each symbolResults as loc (loc.filePath + loc.startLine)}
				<Command.Item value={loc.name} onSelect={() => openSymbol(loc)}>
					{loc.name}
				</Command.Item>
			{/each}
		</Command.Group>
		<Command.Group heading="Files">
			{#each fileResults as entry (entry.path)}
				<Command.Item value={entry.path} onSelect={() => openFile(entry)}>
					{entry.path}
				</Command.Item>
			{/each}
		</Command.Group>
	</Command.List>
</Command.Root>
```

### Anti-Patterns to Avoid

- **Using `pushState`/`replaceState` from `$app/navigation` for view state:** these are the shallow-routing exports (installed source: `web/node_modules/@sveltejs/kit/src/runtime/client/client.js:2504,2551`). They set `page.state`, not `page.url` (confirmed by reading `client.js` for every `page.url =` assignment — the only two occurrences, at lines 1944 and 2984, are inside the main navigation/`goto` path, never inside `pushState`/`replaceState`). A component reading `page.url.searchParams` will not react to them. Use `goto(url, {replaceState: bool})` instead.
- **Client-side range clamping of `depth`/`limit`:** D-12 is explicit — the client parses shape, the server validates value. A client-side clamp both duplicates a bound that can drift and silently turns a shared link into a different query than its sender saw.
- **Importing `highlight.js` (unscoped) or `highlight.js/lib/common`:** both defeat the entire point of D-19's bundle-size argument — `lib/common` alone bundles ~35+ languages, most unused by this indexer's 14-language set.
- **Feeding pre-rendered HTML into `hljs.highlight()`:** the source parameter must be the raw verbatim bytes from `SourceBlob.content`, treated as plain text — see Security Domain.
- **A merged/cross-ranked search result list:** `Location` and `FileEntry` carry no comparable score (confirmed: neither message in `ui.proto` has a numeric relevance field); D-15 mandates two independently-ordered sections, not a client-invented merge.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Path confinement for source reads | A second `resolveSourcePath`-equivalent for any new Browse-view read | `internal/query.Engine.SourceFor` / `resolveSourcePath` (`internal/query/node.go:29-75`, verified read this session) | Already shared between MCP and uiserver; SRV-05's entire point is not creating a second implementation |
| Syntax highlighting engine | A hand-rolled tokenizer/regex highlighter | highlight.js `lib/core` + selective registration | Language grammars are large, subtle, and already maintained; hand-rolling 13 grammars is out of scope for any phase |
| Keyboard-navigable listbox/combobox semantics | Custom `keydown` handlers wiring ARIA roles by hand | shadcn-svelte's `Command` (Bits UI primitive) | Bits UI's `Command` already implements the WAI-ARIA combobox/listbox pattern including arrow-key nav, `aria-activedescendant`, and `Command.Dialog`'s built-in Esc-to-close |
| Connect error → UI state translation | Per-view try/catch with ad-hoc error messages | D-04's shared `rpc-errors.ts` mapper | Phases 4/5/6 hit the identical degraded-server contract; building it once now prevents three future re-implementations that will drift |
| GitHub forge URL construction | String templates guessing at `/blob/{sha}/{path}#L{n}` client-side | D-06's server-side `GetPermalink` RPC | The client cannot know whether the indexed commit was ever pushed (D-07); only the server can shell to `git branch -r --contains` |

**Key insight:** every "don't hand-roll" in this phase already has a server-side or library-side answer that was either built in Phase 1 or is a well-maintained npm package; the phase's actual engineering work is composition and correct API usage, not new algorithms.

## Common Pitfalls

### Pitfall 1: SvelteKit `pushState`/`replaceState` do not update `page.url`

**What goes wrong:** A component that calls the shallow-routing `pushState('/browse?symbol=Foo', {})` and then reads `page.url.searchParams.get('symbol')` gets the OLD value — the address bar changed, but the reactive `page.url` did not.
**Why it happens:** These two functions exist for a different purpose (ephemeral modal-style state that survives back/forward without changing the underlying route's *data*) and only ever assign to `page.state`, never `page.url`. Verified this session by reading the entire `pushState`/`replaceState` implementations in the installed package (`web/node_modules/@sveltejs/kit/src/runtime/client/client.js:2496-2560`) and confirming the only two `page.url =` assignments in the whole client runtime are inside the `goto`-driven navigation path (`client.js:1944, 2984`), never inside these two functions.
**How to avoid:** Use `goto(url, {replaceState, noScroll, keepFocus})` for every URL-as-state update in this phase (Pattern 1 above).
**Warning signs:** A search box that "loses" typed characters on refinement, or a symbol view that doesn't update after clicking a neighbor, while the browser's address bar visibly does change.

### Pitfall 2: The indexed-language count is 14, not 12

**What goes wrong:** Registering exactly the 12 language IDs `03-CONTEXT.md`'s D-19 lists leaves `.js`/`.jsx`/`.mjs`/`.cjs` files rendering unhighlighted (silent degrade to plaintext via `highlightAuto`'s fallback — not a crash, but a real, verifiable gap against BRW-06's "only the indexed languages registered" intent, since `javascript` genuinely IS an indexed language).
**Why it happens:** `internal/indexer/languages_typescript.go`'s single `init()` function registers THREE separate `LanguageSpec.ID`s (`typescript`, `tsx`, `javascript` — read directly at `languages_typescript.go:126-158` this session) sharing one `tsextract.Extract` function, because the underlying extractor "self-derives which of the three grammars produced a given file purely from its own relPath extension" (the file's own doc comment, `languages_typescript.go:122-125`). A search for `ID:\s*"` across `internal/indexer/languages_*.go` returns 14 matches, not 12 (verified this session, exact command and output recorded in this research session's tool log).
**How to avoid:** Register 13 hljs modules (the 12 from D-19 plus `javascript`); `tsx` needs no separate import because hljs's own `typescript` module aliases it (verified this session by fetching and reading `unpkg.com/highlight.js@11.12.0/lib/languages/typescript.js` directly — `aliases: ['ts','tsx','mts','cts']` at lines 902-910).
**Warning signs:** A `.js` file opened in Browse renders with no color at all while `.py`/`.go` files in the same repo highlight correctly.

### Pitfall 3: A query string never reaches the SPA fallback's routing decision — but this is a *feature*, verify it, don't assume it

**What goes wrong (that this pitfall PREVENTS):** Someone might worry that Phase 2's D-10 dotted-path exclusion (`/browse?file=internal/x.go` — the deep link contains a dot) could accidentally 404 under the asset-prefix rule, or that a query string containing `_app/immutable/`-like text could be misrouted.
**Why it's actually fine:** Read directly (`internal/uiserver/spa.go:222-224`, this session): the routing decision is computed from `path.Clean(strings.TrimPrefix(r.URL.Path, "/"))` — Go's `net/url` parser splits `RawQuery` from `Path` before this code ever runs, so `/browse?symbol=Foo&file=internal/x.go&line=42` resolves `r.URL.Path` to exactly `"/browse"`. That path is not under `spaAssetPrefix = "_app/immutable/"` (`spa.go:59-65`) and has no on-disk file (SvelteKit SPA mode emits no per-route `.html`), so it falls straight to `serveFallback` regardless of what the query string contains. **Verified, not assumed** — no code change needed here, but the planner should not add defensive query-string handling to `spa.go` that isn't necessary.
**How to avoid:** Nothing to build; just don't over-engineer this boundary.
**Warning signs:** N/A — this pitfall is "don't invent a problem that verification shows doesn't exist."

### Pitfall 4: highlight.js's default rendering is safe — but only under specific, checkable conditions

**What goes wrong:** Rendering `hljs.highlight(sourceCode, {language}).value` via Svelte's `{@html ...}` is safe by default, but highlight.js has historically shipped an HTML-passthrough capability (moved to an opt-in plugin as of v11) that, if ever enabled, would make raw `<script>` tags in an indexed repository's source files executable in the browser.
**Why it happens:** highlight.js escapes `<`, `>`, `&` in its input by default and does not treat the input as pre-rendered HTML — this is the current, v11+ default with no plugin loaded `[CITED: github.com/highlightjs/highlight.js/wiki/security, via WebSearch]`.
**How to avoid:** Never add or import any highlight.js "HTML passthrough" plugin; always call `.highlight(code, {language})` with `code` as the raw string from `SourceBlob.content` (decoded, never pre-templated); never construct the highlighted HTML string via string concatenation before handing it to highlight.js.
**Warning signs:** A code review flags any `dangerouslySetInnerHTML`/`{@html}` site that does NOT route through the one `highlightSource()` helper in `$lib/highlight.ts`.

## Code Examples

### `GetPermalink` RPC shape (additive to `ui.proto`, D-06/D-07/D-08/D-09)

```protobuf
// Source: this session's synthesis of 03-CONTEXT.md D-06/D-07/D-08/D-09,
// following the additive-only discipline already documented at
// internal/uiproto/uiv1/ui.proto's package header (verified read this
// session) and the SourceBlob/NodeDefinition precedent for how a new
// message is added at the next free field number.
message GetPermalinkRequest {
  string path = 1;
  int32 line = 2;
  optional int32 end_line = 3; // D-09: range link when known, single-line
  //                              otherwise. Location (Search/Callers/
  //                              Callees/Impact/Affected's shared
  //                              projection) carries only start_line —
  //                              confirmed by reading Location's own doc
  //                              comment (ui.proto: "a lightweight
  //                              name/kind/file/line-only reference")
  //                              this session — so a list-sourced call
  //                              site simply omits end_line.
}

enum PermalinkAvailability {
  PERMALINK_AVAILABILITY_UNSPECIFIED = 0;
  PERMALINK_AVAILABILITY_LINKABLE = 1;              // known-pushed commit
  PERMALINK_AVAILABILITY_LINKABLE_UNVERIFIED = 2;   // D-07: commit not
  //                                                    seen in any remote-
  //                                                    tracking branch —
  //                                                    "may 404", never
  //                                                    asserted as fact
  PERMALINK_AVAILABILITY_NO_LINK = 3;               // D-08: non-GitHub
  //                                                    forge, no remote,
  //                                                    or no commit_sha
}

message GetPermalinkResponse {
  string url = 1;       // empty when availability is NO_LINK
  PermalinkAvailability availability = 2;
  string reason = 3;    // human-readable, e.g. "remote is GitLab, not
  //                        supported yet" or "commit not found on any
  //                        remote-tracking branch"
}
```

### Server-side unpushed-commit check (D-07)

```go
// Source: this session's synthesis, following the established
// gitmeta/commit.go pattern (gitExecLookPath seam, 5s timeout, degrade-
// to-empty-never-error) verified read this session at
// internal/indexer/commit.go:19-73 and internal/gitmeta/worktree.go:29-52.
// `git branch -r --contains <sha>` returning ANY line proves the commit
// is on a remote-tracking branch (sound in exactly one direction — see
// D-07's own reasoning, verified against 03-CONTEXT.md).
cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "branch", "-r", "--contains", sha)
out, err := cmd.Output()
if err != nil {
    return availabilityNoLink, "could not verify remote branches"
}
if len(bytes.TrimSpace(out)) == 0 {
    return availabilityLinkableUnverified, "commit not found on any remote-tracking branch — link may 404"
}
return availabilityLinkable, ""
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|---------------|--------|
| `cmdk-sv` (standalone Svelte port of `cmdk`) | Bits UI's built-in `Command` component | huntabyte/shadcn-svelte's "October 2023 - Command and Combobox" changelog entry `[CITED: huntabyte/shadcn-svelte docs, via Context7]` folded `cmdk-sv` into Bits UI directly; shadcn-svelte's `command` registry component now wraps Bits UI's `Command`, not a separate `cmdk-sv` dependency | No action needed — `pnpm dlx shadcn-svelte add command` already installs the current shape |
| SvelteKit `$page` store (`$app/stores`) | `$app/state`'s `page` (rune-based) | Already the pattern this repo uses (`web/src/routes/+layout.svelte:5`, verified) | None — already correctly adopted in Phase 2 |
| Prism.js as the "lightweight" default choice | highlight.js, given Prism's stalled v2 and v1's maintenance-only status | Confirmed this session: `prismjs@1.30.0` last published 2025-03-10, no newer release as of 2026-08-28 (~17 months) | Reinforces, does not change, D-19's already-locked choice |

**Deprecated/outdated:**
- `cmdk-sv` as a standalone package: superseded by Bits UI's native `Command` — do not `pnpm add cmdk-sv` directly; use the shadcn-svelte CLI's `command` component, which pulls the current Bits UI dependency.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|----------------|
| A1 | `Command.Root shouldFilter={false}` is the correct prop to disable Bits UI's built-in client-side filtering for server-driven results | Pattern 4 | If wrong, typed search results get double-filtered client-side against already-server-filtered results, likely hiding valid matches whose display text doesn't literally contain the typed substring (e.g. a symbol matched by fuzzy/semantic ranking). Confirmed via WebSearch summary of bits-ui.com's own docs page (CITED, not independently loaded/read this session) — treat as needing a quick confirmation read of the live bits-ui.com page during planning/execution, not a re-derivation |
| A2 | `noScroll: true, keepFocus: true` on every debounced-typing `goto()` call is sufficient to keep the search input focused through rapid re-navigation | Pattern 1 | If insufficient, the search box could lose focus on every keystroke during live search — a real UX break for NAV-03. Low risk (these are exactly the documented options for this use case per the installed source's own JSDoc, verified this session), but the interaction with `AbortController`-cancelled in-flight RPCs was not tested live in this research pass |
| A3 | Bits UI's `Command` implements the WAI-ARIA combobox/listbox keyboard pattern sufficiently for NAV-03's accessibility bar, without additional hand-wiring | Don't Hand-Roll | If the built-in pattern is incomplete (e.g. missing `aria-activedescendant` wiring for screen readers), NAV-03's "drivable from the keyboard" criterion could still pass a sighted manual test while failing an accessibility audit. Not verified against a live accessibility tree in this session — Claude's Discretion area per 03-CONTEXT.md, but worth a UAT pass with a screen reader or `axe` if `ui_review`/accessibility gates are enabled |

**A1 and A3 both trace to bits-ui.com content read via WebSearch summary, not a direct page fetch in this session — the underlying claim is almost certainly correct (it matches the well-known `cmdk`/`cmdk-sv` API this component descends from), but neither was independently confirmed against the live docs page.**

## Open Questions

1. **Does `Command.Root`'s `shouldFilter={false}` need a companion `filter` no-op, or does setting it alone fully disable internal filtering?**
   - What we know: WebSearch-summarized bits-ui.com docs state `shouldFilter={false}` alone is sufficient for async/custom-filtering use cases.
   - What's unclear: whether any Bits UI version-specific behavior differs from the summarized text (the summary was not independently cross-checked against a second source).
   - Recommendation: a 2-minute confirmation read of `bits-ui.com/docs/components/command` at plan/execution time, or a quick manual smoke test once the component is wired (type a query whose server results don't literally substring-match the typed text — e.g. a symbol matched by camelCase-abbreviation search if the Engine supports it — and confirm it still renders).

2. **What does the Engine's `Search`/`Files` matching actually rank on (exact substring vs. something fuzzier)?**
   - What we know: `SearchResponse`'s doc comment says results come back "in the same order `Engine.Search` returns them" — order is server-owned, per D-15.
   - What's unclear: whether `shouldFilter={false}` is even load-bearing if `Search`'s own matching is a literal substring match (in which case Bits UI's default filter would have produced the same order anyway, and disabling it is a belt-and-suspenders correctness matter, not a functional necessity).
   - Recommendation: not blocking — `shouldFilter={false}` is correct regardless of the answer, so this doesn't gate planning, but understanding the Engine's actual match semantics would help write better UAT search-quality checks.

3. **Should `GetPermalinkResponse` also carry the `SourceBlob.total_lines`-derived range, or does the caller always know its own `end_line`?**
   - What we know: D-09 says the caller passes what it has — an opened node has `end_line`, a list-sourced target does not.
   - What's unclear: whether the truncated-file "showing first N of M lines" message (D-20) should link to `#L1-L{total_lines}` (the WHOLE untruncated file, honoring the "match what the UI just showed" framing loosely) or `#L1-L{returned_lines}` (only the truncated portion actually rendered, honoring it strictly).
   - Recommendation: strict reading — link to the returned/rendered range, since D-20's whole point is "the remote view is the escape hatch for the REST of the file," which argues for linking to the file generally (no end_line, GitHub's own view shows the whole file) rather than either partial range. Flag for `/gsd-plan-phase` to make an explicit call, since this is a two-line UX decision with no wrong technical answer.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|--------------|-----------|---------|----------|
| Go toolchain | Server-side `GetPermalink` RPC, SRV-05 regression test | ✓ | 1.26.5 (`go.mod`) `[VERIFIED: go.mod:1-5]` | — |
| `git` binary | `GetPermalink`'s pushed-commit check (D-07), reusing `gitmeta`'s exec pattern | Assumed present (existing `gitmeta`/`commit.go` code already degrades gracefully to empty when absent — same contract this phase's new code follows) | — | Already-established graceful degrade: `availability = NO_LINK` when git absent, per D-06's extension of `commit.go`'s pattern |
| `pnpm` | `highlight.js`/shadcn-svelte component installs, `web/` build | ✓ (repo already builds via `pnpm`) `[VERIFIED: web/package.json packageManager field, existing Phase 2 build]` | `pnpm@11.23.0` pinned via `packageManager` | — |
| `connectrpc.com/connect` (Go) | New `GetPermalink` RPC wiring | ✓ | `v1.20.0` `[VERIFIED: go.mod:41]` | — |

**Missing dependencies with no fallback:** none.
**Missing dependencies with fallback:** `git` binary absence — already has a documented, tested fallback pattern this phase reuses, not invents.

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Go framework | stdlib `testing`, run via `task test:unit` (`go test` over every package except `internal/daemon`) `[VERIFIED: Taskfile.yml:116-131]` |
| Go config file | none (stdlib) |
| JS framework | **none installed** — `find web -iname '*.test.*' -o -iname '*.spec.*'` (excluding `node_modules`) returns zero files; no `vitest`/`playwright`/`@testing-library` in `web/package.json`'s `devDependencies` `[VERIFIED: this session's direct find + package.json read]` |
| JS config file | none — Wave 0 gap if the plan adopts JS unit tests |
| Quick run command (Go) | `go test ./internal/uiserver/... ./internal/query/... ./internal/gitmeta/...` (scoped to this phase's touched packages) |
| Full suite command (Go) | `task test:unit` |
| Quick run command (JS) | none yet — see Wave 0 Gaps |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|---------------------|--------------|
| SRV-05 | `../` escape, absolute path, symlink escape refused at the RPC boundary; a legitimate in-repo path returns real source in the SAME test (rule `84d1gfpywd` positive control) | Go integration (Connect handler, real client) | `go test ./internal/uiserver/ -run TestGetNodeDetailConfinement` (name TBD by planner) | ❌ Wave 0 — new test file, e.g. `internal/uiserver/confinement_test.go` |
| BRW-09/D-06/D-07/D-08/D-09 | `GetPermalink` returns correct URL/availability/reason for: pushed commit, unpushed commit, no remote, non-GitHub remote, missing `commit_sha` | Go unit (new `internal/uiserver`/`internal/gitmeta` package) | `go test ./internal/uiserver/... ./internal/gitmeta/... -run Permalink` | ❌ Wave 0 |
| NAV-01/NAV-02 | URL round-trips: parse then serialize is idempotent; unknown params ignored, not rejected (D-12); push vs replace produces the correct history-entry count | JS unit (if vitest adopted) OR Go-adjacent manual UAT | `pnpm --dir web vitest run browse-url` (if adopted) | ❌ Wave 0 (both the test file and the framework) |
| BRW-06 | Registered language set matches `indexer.RegisteredLanguageIDs()` exactly — no missing, no extra (mirroring the existing `TestMatrix_CoversRegisteredLanguages` pattern) | Go unit, literal-fixture comparison | `go test ./internal/indexer/... -run TestHighlightRegistrationCoversRegisteredLanguages` (name TBD; new test comparing a hardcoded Go-side literal — kept in sync with `web/src/lib/highlight.ts`'s import list by code comment cross-reference, since Go cannot import TS) | ❌ Wave 0 — new test, mirrors `internal/indexer/capability/matrix_test.go:33-41` (verified read this session) |
| BRW-01/BRW-08/NAV-03 | Search debounce/abort/keyboard nav | Manual UAT (browser-driven; no JS test framework installed) | — | N/A — manual only unless vitest+jsdom or Playwright is added |

### Sampling Rate

- **Per task commit:** relevant Go package's `go test ./internal/<pkg>/...`
- **Per wave merge:** `task test:unit` (full Go suite) plus manual browser UAT for the client-only surfaces
- **Phase gate:** `task test:unit` green, `task proto:drift` green (after regenerating for `GetPermalink`), `task web:build:verify` green, before `/gsd-verify-work`

### Wave 0 Gaps

- [ ] `internal/uiserver/confinement_test.go` (or similarly named) — covers SRV-05's positive-and-negative pair
- [ ] `internal/uiserver/permalink_test.go` and/or `internal/gitmeta/remote_test.go` — covers BRW-09's D-06/D-07/D-08/D-09 branches
- [ ] `internal/indexer/highlight_coverage_test.go` (name TBD) — mirrors `capability/matrix_test.go`'s "no missing, no extra" pattern against `indexer.RegisteredLanguageIDs()`, guarding the 13-module hljs registration list against silent drift when a 15th language is added later
- [ ] **Decision needed from the plan:** adopt `vitest` for `web/`'s new pure-TS modules (`browse-url.ts`, `rpc-errors.ts`, the click-to-definition name matcher), or accept Go-RPC-tests-plus-manual-UAT as this phase's JS coverage. No JS test framework currently exists; this is a real Wave-0-or-explicitly-declined decision, not an oversight to silently skip.

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|----------------|---------|-------------------|
| V2 Authentication | No | Explicitly out of scope this milestone (loopback-only, no auth — `.planning/REQUIREMENTS.md` "Multi-user auth / accounts" Out of Scope table) |
| V3 Session Management | No | No session concept in this read-only tool |
| V4 Access Control | No | Single-user, single-repo, loopback-bound |
| V5 Input Validation | Yes | Server-side `validateLimit`/`MaxLimit`/`validateFilesDepth`/`validateDepth` (existing, `internal/query/validate.go`, verified read this session) — client passes through, never re-validates (D-12) |
| V6 Cryptography | No new surface | No new cryptographic operation this phase; `GetPermalink` reads `.git/config`-derived data, no secrets |
| V7 Output Encoding / XSS | Yes — NEW this phase | highlight.js's default HTML escaping (v11+, no passthrough plugin) — see Pitfall 4 and Common Pitfalls |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|-----------------------|
| XSS via rendered verbatim source (a malicious/compromised indexed file's literal content, e.g. `<script>...`) | Tampering / Elevation of Privilege | highlight.js v11's default escaping (`<`, `>`, `&`) with no HTML-passthrough plugin loaded — verified this session against `github.com/highlightjs/highlight.js/wiki/security` `[CITED]`. Never construct the highlighted string via concatenation; always route through `hljs.highlight(rawText, {...}).value` |
| Path traversal via `GetNodeDetail`'s `file` field | Tampering / Information Disclosure | Already closed — `internal/query.resolveSourcePath` (`node.go:29-75`), SRV-05 adds only a regression test at the RPC boundary, no new logic |
| Information disclosure via unclassified error messages (absolute host paths in `*os.PathError`) | Information Disclosure | Already closed — `mapEngineError`'s default arm scrubs to a fixed `errInternal` message, real error goes to server diagnostic stream only (`internal/uiserver/handlers.go:101-127`, verified read this session) |
| DNS rebinding against the loopback `GetPermalink` RPC | Spoofing | Already closed — `originHostGuard` wraps the WHOLE mux including any new RPC (`internal/uiserver/server.go:111-130`, verified — new handlers registered on the same mux automatically inherit this) |
| Open redirect / SSRF via a crafted `GetPermalinkRequest.path` | Tampering | Not a new attack surface — `GetPermalink` only ever renders a `github.com/<owner>/<repo>/blob/...` URL derived from the server's OWN resolved remote + the server's OWN resolved `commit_sha`; `path`/`line` are interpolated into the URL PATH, not executed or fetched server-side. Standard mitigation: URL-encode `path` when building the permalink string (a file path containing `#`/`?`/space could otherwise corrupt the link) |

## Sources

### Primary (HIGH confidence — read or executed directly this session)

- `internal/query/node.go:29-90` — `resolveSourcePath`/`readSourceFile`, the confinement gate
- `internal/query/errors_test.go:1-60,120-150` — existing refusal assertions (`escapes the repo root`, absolute path)
- `internal/uiserver/handlers.go:1-135,195-330,600-760` — `withEngine`, `mapEngineError`, `GetStatus`, `GetNodeDetail`, `singleDefSourceBlob`, `statusToProto`
- `internal/uiserver/degrade.go` (full file) — `classifyDegrade`, `errIndexingInProgress`, `degradedStatus`
- `internal/uiserver/truncate.go` (full file) — `truncateSource`, `countLines`, the two-cap bound
- `internal/uiserver/spa.go` (full file) — SPA fallback routing, confirming query strings never reach path-based routing decisions
- `internal/uiserver/server.go:1-140` — `Listen`, mux construction, transport backstops
- `internal/uiserver/originguard.go:1-60` — `originHostGuard`
- `internal/uiproto/uiv1/ui.proto` (full file) — every message cited in this document
- `internal/schema/graph.proto:1-100` — `Edge`'s line/col reservation and its key-collapse comment
- `internal/schema/meta.go` (full file) — `IndexedCommitSHA`
- `internal/gitmeta/worktree.go:1-90`, `internal/indexer/commit.go` (full file) — git-exec-with-graceful-degradation pattern
- `internal/indexer/languages_*.go` (all 12 files, `ID:` lines) — the 14-value registered language ID set (the correction)
- `internal/indexer/languages_typescript.go:100-158` — three separate registrations sharing one extractor
- `internal/indexer/languages.go` (full file) — `LanguageSpec`, `RegisteredLanguageIDs`, `registerLanguage`
- `internal/indexer/capability/matrix_test.go:1-40` — the "no missing, no extra" drift-guard precedent
- `internal/query/validate.go:1-70` — `defaultMaxFiles`, `MaxDepth`/`MaxLimit`/`MaxFiles`
- `internal/query/explore.go:100-144` — H21 adaptive budget, `[1,20]` clamp
- `web/package.json`, `web/vite.config.ts`, `web/src/routes/+layout.ts`, `web/src/routes/+layout.svelte`, `web/src/routes/+page.svelte`, `web/src/routes/browse/+page.svelte`, `web/src/lib/client.ts`, `web/embed.go` — full reads, current installed state
- `web/node_modules/@sveltejs/kit/src/runtime/client/client.js:2280-2570` — `goto`, `pushState`, `replaceState` implementations, read directly from the installed package (not docs)
- `go.mod:1-45` — Go 1.26.5, `connectrpc.com/connect v1.20.0`
- `Taskfile.yml` (`proto:gen`, `proto:drift`, `test:unit` sections) — codegen and test tooling
- npm registry direct queries: `npm view highlight.js version/dist-tags/scripts.postinstall/repository.url`, `npm view prismjs version`, `npm view vitest version/peerDependencies`, `npm view shadcn-svelte version`
- `unpkg.com/highlight.js@11.12.0/lib/core.js` and `.../lib/languages/{c,cpp,csharp,go,java,javascript,kotlin,php,python,ruby,rust,swift,typescript}.js` — fetched and read directly this session (registration mechanism, aliases, byte sizes)
- `registry.npmjs.org/highlight.js` and `.../prismjs` — direct registry JSON, `time.latest` compared
- `gsd-tools query package-legitimacy check --ecosystem npm highlight.js vitest` — executed this session

### Secondary (MEDIUM confidence — official docs via Context7/WebFetch/WebSearch, not independently re-verified against a second source)

- `/sveltejs/kit` (Context7) — `goto`/`pushState`/`replaceState` docs, `URLSearchParams` access pattern
- `/highlightjs/highlight.js` (Context7) — import patterns, `registerLanguage`, `lib/common`
- `/huntabyte/shadcn-svelte` (Context7) — `Command`/`Combobox`/`Command.Dialog` examples, October 2023 changelog entry
- `raw.githubusercontent.com/highlightjs/highlight.js/main/SUPPORTED_LANGUAGES.md` (WebFetch) — confirmed all 13 needed hljs module names/aliases exist verbatim
- `github.com/highlightjs/highlight.js/wiki/security` (WebSearch summary) — default escaping behavior, HTML-passthrough-as-opt-in-plugin

### Tertiary (LOW confidence — WebSearch summary only, not independently loaded)

- `bits-ui.com/docs/components/command` — `shouldFilter={false}` prop (A1 in Assumptions Log — recommend a direct confirmation read before relying on it in a task)

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — every version number and maintenance-recency claim confirmed via direct registry query this session
- Architecture: HIGH — every architectural decision was already locked in `03-CONTEXT.md` with citations; all citations verified to hold against the real tree this session, plus the corrected language-count finding and the SvelteKit URL-state API finding, both independently verified against installed source
- Pitfalls: HIGH for Pitfalls 1-4 (all independently verified against either installed package source or live registry/GitHub data this session); MEDIUM for the `Command` `shouldFilter` claim specifically (WebSearch-summarized, not directly loaded)

**Research date:** 2026-08-28
**Valid until:** 30 days for the architectural/decision content (stable — locked by discuss-phase); 7 days for the exact npm version numbers cited (highlight.js/vitest both had very recent releases at research time — re-check `npm view <pkg> version` immediately before the install task runs if more than a few days have passed)
