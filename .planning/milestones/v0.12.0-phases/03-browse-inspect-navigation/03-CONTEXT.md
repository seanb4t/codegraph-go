# Phase 3: Browse, Inspect & Navigation - Context

**Gathered:** 2026-08-28
**Status:** Ready for planning

<domain>
## Phase Boundary

The first view that shows code. A developer types a partial symbol or file name,
sees results as they type, opens one, reads syntax-highlighted verbatim source
alongside its callers, callees and blast radius, clicks a neighbour and keeps
going — and every state they reach is a URL they can hand to someone else.

Fills the `/browse` slot Phase 2's D-18 created. Does **not** restructure
navigation, does **not** ship the query workbench (Phase 4), the graph view
(Phase 5), or live push (Phase 6).

**Requirements:** BRW-01..BRW-09, NAV-01..NAV-04, SRV-05 (14 total).

</domain>

<decisions>
## Implementation Decisions

### Source Serving & Path Confinement (SRV-05, NAV-04)

- **D-01:** **Whether Phase 3 adds a source endpoint is decided by enumerating
  what the views actually need**, not assumed from criterion 5's wording.

  **The tension, recorded so the planner cannot silently pick either reading.**
  Criterion 5 says the confinement refusal must be "proven by a regression test
  aimed at **the new endpoint**" — but the endpoint is not new, and the refusal
  already works. Verified in tree:
  - `GetNodeDetail(file: "../outside.txt")` is already refused —
    `internal/query/errors_test.go:139` asserts `ErrInvalidArgument` with
    `query: path "../outside.txt" escapes the repo root`. Absolute paths
    (`/etc/passwd`) and symlink escapes are refused too
    (`internal/query/node.go:46,60,75`; WR-03 re-verifies confinement *after*
    resolving symlinks).
  - The `file` field of `GetNodeDetailRequest` is the **only** client-steerable
    path input on the whole service.

  So the planner must first enumerate every source fetch the Browse view
  performs — open a file, open a symbol, click a caller in another file, jump to
  a definition — and add an endpoint **only if a concrete call site is poorly
  served** by `GetNodeDetail`. Adding one "because criterion 5 says new" would
  create the second confinement implementation SRV-05 exists to prevent
  ("reuses the existing MCP path-confinement fix rather than reimplementing it").

- **D-02:** **SRV-05's real deliverable is the missing proof, not missing
  behaviour.** `internal/uiserver/*_test.go` has **zero** occurrences of
  `confinement` or `escapes the repo root` — the gate is only tested one layer
  down, in `internal/query`. Phase 3 adds a regression test at the RPC boundary
  covering `../` escape, absolute path, and symlink escape, asserting the refusal
  is surfaced across the wire.

  Per rule `84d1gfpywd` this guard needs a **positive assertion that it
  discriminates** — a test that only asserts "the bad path was refused" passes
  vacuously if the RPC starts refusing everything. Pair it with a control: a
  legitimate in-repo path that returns real source in the same test.

- **D-03:** **Stale is split out of the source-pane collapse; the other two
  causes stay collapsed.** `singleDefSourceBlob` (`internal/uiserver/handlers.go:641`)
  deliberately returns `nil` for all three read failures under CR-02 — empty file
  path (package pseudo-node), file removed since indexing, and confinement
  rejection — arguing that "distinguishing them on the wire would ask a browser
  to render a taxonomy of read errors it cannot act on."

  That reasoning holds for two of the three. It does **not** hold for
  *file removed since indexing*, which is precisely the stale-index case NAV-04
  names by hand and which the user **can** act on by re-indexing.

  **No proto change is needed.** `GetStatusResponse.stale` (field 6) is already
  on the wire. When `stale` is true and source is absent, render
  "source unavailable — index is stale, re-run `codegraph index`"; otherwise
  render a plain "no source for this node". CR-02's collapse is preserved as-is
  on the wire.

- **D-04:** **A shared status gate and a shared Connect error mapper live in
  `web/src/lib/`, not inside the Browse view.**

  Motivating asymmetry, verified in `internal/uiserver/degrade.go`: **`GetStatus`
  is the only RPC that answers when the store is degraded** (D-16). Every other
  handler returns a Connect *error* — `CodeUnavailable` carrying a typed
  `IndexingInProgress` detail (`degrade.go:105-107`). So NAV-04's three states
  arrive by two different mechanisms:

  | NAV-04 state | Arrives as |
  |---|---|
  | No index | `GetStatus` **answer** (`initialized=false`, `store_exists=false`) |
  | Stale index | `GetStatus` **answer** (`stale=true`) |
  | Symbol not found | `CodeNotFound` **error** from the specific call |

  A client that only handles empty responses renders a blank pane in exactly the
  two cases NAV-04 exists to prevent. Two pieces of shared plumbing: (1) a status
  store holding `GetStatus`, driving a view-level banner every view inherits;
  (2) one error mapper turning `CodeUnavailable`+`IndexingInProgress`,
  `CodeNotFound` and `CodeInvalidArgument` into named UI states. Phases 4, 5 and
  6 hit the same degraded server — building this per-view means writing it four
  times and drifting three.

- **D-05:** **The status gate fetches on load and on navigation. No timer, no
  polling loop.** Phase 6 (Live Push) replaces this mechanism, so it is
  deliberately a small seam rather than a polling subsystem to tear out. A user
  who re-indexes mid-session sees the change on their next click.

### GitHub Permalinks (BRW-09)

- **D-06:** **BRW-09 is served by a new `GetPermalink` RPC, not by data fields.**
  *Maintainer proposal, adopted over the three options offered.*

  Signature shape: `GetPermalink(path, line, end_line?) -> {url, availability, reason}`.
  The server owns **all** of it — forge URL shape, remote-URL normalization,
  `url.<base>.insteadOf` rewrites, and the pushed-hint — in Go, where it is
  unit-testable. The client substitutes nothing.

  **Why a field could not do the job.** The browser cannot read `.git/config`,
  and `commit_sha` (`GetStatusResponse` field 7) is only half a permalink — the
  owner/repo half is on the wire nowhere (`rg 'remote|origin_url|repo_url|github'
  internal/uiproto/uiv1/ui.proto` returns only `go_package`). A static template
  string is a pure formatter: it can say where a file *would* live, but cannot
  express "…and that commit is not on the remote, so this link would 404" — which
  is BRW-09's actual failure mode. An RPC can.

  Adding a **method** also fits the service's additive discipline better than
  spending permanent `GetStatusResponse` field numbers.
  — **Reversibility:** one-way — a published RPC method on `uiv1.UIService`
  inherits D-02a's additive-only discipline; it can be deprecated but never
  removed or reshaped once a client depends on it.

  **Plumbing precedent already exists:** `internal/gitmeta/` shells to git for
  exactly this class of metadata (`worktree.go:38,64`), and
  `internal/indexer/commit.go:60-68` establishes the degradation rule —
  `gitExecLookPath("git")` fails → return empty → treated as "unknown, never an
  error", the same contract `commit_sha` already follows.

- **D-07:** **The unpushed-commit hint is in scope, and `availability` is
  three-valued — never a boolean.** States: *linkable* / *linkable-but-possibly-
  unpushed* / *no-link*, each with a `reason`.

  **The check is sound in one direction only, and this must not be collapsed.**
  `git branch -r --contains <sha>` returning a hit **proves** the commit is on
  the remote. Returning nothing proves only that the last fetch did not see it —
  someone else may have pushed it, or the user simply has not fetched. A
  read-only local tool **must not fetch** to find out. So the third state is
  named as *uncertainty* ("this commit is not in any remote-tracking branch; the
  link may 404"), never as fact. A boolean here would cry wolf on every repo
  that has not fetched recently, and users learn to ignore warnings that are
  wrong half the time.

  Why it matters at all: BRW-09 pins to the **indexed** commit, and for a local
  dev tool the indexed commit is very frequently an unpushed local commit. The
  expected failure is ordinary daily use, not an edge case.

- **D-08:** **GitHub only. Every other forge resolves to an explicit `no-link`
  with a reason — never a wrong URL.**

  Forge URL shapes are genuinely not interchangeable (GitHub `/blob/{sha}/{path}#L{n}`,
  GitLab `/-/blob/...`, Bitbucket `/src/{sha}/{path}#lines-{n}`, Codeberg
  `/src/commit/...`), and correct detection for self-hosted GitLab or GitHub
  Enterprise needs a **configurable host mapping** — which this project has no
  mechanism for (Phase 1 D-08 explicitly declined a config file; the repo's
  convention is env-vars-paired-with-flags). Honors BRW-09's literal text.
  Extension point is a switch arm plus a test — **no wire change**, because
  `availability`/`reason` already carry the vocabulary.

- **D-09:** **Range link when `end_line` is known, single line otherwise.**
  An opened symbol links to `#L{start}-L{end}` — `Node` carries `start_line`
  (field 7) and `end_line` (field 8) — matching BRW-09's stated purpose that
  "the remote view matches what the UI just showed". A list-sourced target links
  to `#L{start}`, because `Location` (the shared projection returned by `Search`,
  `Callers`, `Callees`, `Impact`, `Affected`) carries `start_line` only.

  **`Location` is deliberately not extended.** Its own doc comment defines it as
  "a lightweight name/kind/file/line-only reference"; spending a permanent field
  number on five RPCs' shared message to enrich a link the user has not opened
  yet is the wrong trade. `GetPermalink` takes `end_line` as optional and the
  caller passes what it has.

### URL State & History (NAV-01, NAV-02)

- **D-10:** **Query params named after `GetNodeDetailRequest`'s own fields.**
  `/browse?symbol=Foo&file=internal/x.go&line=42&depth=3&limit=50`.

  View is **already** a path segment — Phase 2's D-18 created `/browse`,
  `/graph`, `/workbench`, `/health` as real routes — so NAV-01's remaining job is
  target/depth/limit only.

  Two reasons for query params over path segments:
  1. **The URL→RPC mapping is identity**, so it cannot drift. `symbol`/`file`/
     `line` are literally the request's field names, and `symbol` vs `file`
     disambiguates a target exactly the way the RPC already does — giving BRW-05's
     multi-def picker a natural address.
  2. **A file path in a path segment would re-open what Phase 2's D-10 just
     closed.** D-10 rejected the "any path with a dot is an asset" heuristic
     *specifically because* "Phase 3's deep links will carry file paths containing
     dots." In a query value, dots and slashes are inert.
  — **Reversibility:** costly — Phase 4's workbench deep-links against this shape
  by the roadmap's own statement, and URLs shared out of the tool are the one
  artifact this project cannot recall or migrate.

- **D-11:** **Navigation pushes history; refinement replaces it.**
  `pushState` for opening a node, clicking a neighbour, or picking a
  disambiguation candidate — places the user wants to come back to.
  `replaceState` for typing in search and for adjusting depth/limit — the URL
  stays shareable and correct at every instant, but back skips the intermediate
  states.

  This is what NAV-02's "walk that history correctly" actually asks for: pushing
  every keystroke makes back useless (200 entries to escape one search), while
  pushing only view changes makes "click through five callers, return to the
  third" impossible — and that motion is the phase goal's own wording ("keep
  clicking outward without losing their place").

- **D-12:** **The client parses shape only; the server validates values.**
  The client checks that `depth` is an integer, not that it is in range, and
  passes it through. `validateLimit`/`MaxLimit`/`validateFilesDepth` already
  refuse out-of-range values with `CodeInvalidArgument`, routed through D-04's
  error mapper into a named UI state — so `?depth=999` renders an explicit
  message, not an empty pane, satisfying NAV-04 with machinery already decided.

  This follows a discipline stated on nearly every request message in
  `ui.proto`: *"limit is passed straight through … so this rpc adds no second
  copy of that rule that could drift from the Engine's own."* Client-side
  clamping would both duplicate the bound in TypeScript and silently turn a
  shared link into a different query than its sender saw.

  Unknown params are **ignored, not rejected** — a strict reading would make an
  older cached shell reject a newer phase's link, the same stale-shell failure
  class Phase 2's D-11 `no-store` rule exists to prevent.

- **D-13:** **One shared parse/serialize module in `web/src/lib/`, defining only
  the params Phase 3 uses.** Lives alongside D-04's status gate and error mapper.
  Later phases add params to the same module rather than inventing a second
  grammar. This settles the **shape** the roadmap asked to be settled — param
  naming convention, pass-through validation, push-vs-replace rule — without
  speculating about Phase 4/5 params that have not been specified. A wrong guess
  that ships inside a shareable URL is worse than no guess.

### Search Surface (BRW-01, BRW-08, NAV-03)

- **D-14:** **`Search` and `Files` fire live as you type; `Explore` fires on
  Enter only, rendering alongside in its own section.**

  **`Explore` cannot ride a keystroke, and the proto proves it.** `ExploreGroup`
  carries a **`SourceBlob` per matched file** (field 4) for up to
  `defaultMaxFiles = 5` (`internal/query/validate.go:56`; H21 adaptive budget
  clamped to [1,20], `explore.go:131`). `Explore` does not merely rank — it reads
  files off disk and returns their contents. `Search`, by contrast, returns bare
  `Location`s. The two differ by roughly an order of magnitude in cost.

  BRW-08's word "alongside" is honoured — both are present — but they cannot
  share a trigger. The interaction also teaches itself: typing narrows, Enter
  asks a question. And it degrades honestly: if `Explore` is slow, the live
  results are already on screen.

  Note for NAV-04: `Explore` returning nothing is a **successful** response with
  `empty=true`, deliberately never an error, because "turning it into an error
  would make an ordinary 'no results' indistinguishable from a genuine failure."
  No-results and failure are therefore already distinguishable on the wire.

- **D-15:** **Two labelled sections — Symbols, then Files — each preserving its
  own RPC's ordering. Explore appears as a third section on submit.**

  **No client-side cross-ranking.** `Location` and `FileEntry` carry no
  comparable score, so a merged list would require an invented heuristic that
  silently disagrees with the server's own ordering and has nothing to be tested
  against. `SearchResponse`'s doc comment is explicit that results come back "in
  the same order `Engine.Search` returns them." Keyboard selection walks across
  sections in visual order, so the merged *feel* costs nothing.

- **D-16:** **~150ms debounce, 2-character minimum, `AbortController` mandatory.**
  The 250-300ms figure in current guidance is explicitly for queries crossing a
  **network**; this server is on loopback, so the low end of the 150-300ms range
  applies. 2 characters matches the "names" case, which is what symbol search is.

  **Cancellation is non-negotiable and independent of the delay.** Debounce alone
  does not fix out-of-order responses: without aborting the superseded request, a
  slow response to `"car"` can land after the response to `"card"` and push the
  UI backward.

- **D-17:** **Both `/` and `Cmd`/`Ctrl`+`K` focus the search box; `Esc`
  dismisses.** `/` is muscle memory for exactly this audience (GitHub,
  Sourcegraph); `Cmd+K` is the modern command-palette expectation. Supporting
  both means neither audience has to learn the other's habit. **`/` must be
  suppressed while an input already has focus** — file paths are full of slashes.

### Source Rendering & Navigation (BRW-04, BRW-05, BRW-06)

- **D-18:** **Click-to-definition resolves from clicked text, and only
  identifiers matching a name in the node's `calls` list are clickable.**

  **Server-side resolved reference spans are structurally impossible this phase**
  — this was checked, not assumed. `SourceBlob`'s `reserved 50 to 59` band
  explicitly anticipates "syntax-highlight spans", but `graph.proto`'s `Edge`
  comment (lines 75-84) states that although `Edge` *can* carry `line`/`col`
  (fields 4-5), the Pebble edge key `e/<src>/<kind>/<dst>` **"intentionally omits
  line/col today, so two call sites between the same (source, kind, target)
  collapse to one stored edge."** The store cannot enumerate call sites at all.
  Server-side spans would need a storage key-shape change — far outside Phase 3.

  So BRW-04 has exactly one viable shape, and BRW-05's picker exists in the same
  phase precisely because text-driven resolution is inherently ambiguous.

  `GetNodeDetail` already returns this node's `calls` and `called_by`. Making an
  identifier clickable only when its text matches a name in `calls` lights up
  genuine outbound references while leaving `err`, `ctx`, `i` and every local
  inert — using data already on the wire, with no new RPC. A click still issues
  `GetNodeDetail`, so a name with several definitions lands in BRW-05's picker.

- **D-19:** **highlight.js with selective registration — `lib/core` plus exactly
  the 12 indexed languages.** The indexed set is
  `c, cpp, csharp, go, java, kotlin, php, python, ruby, rust, swift, typescript`
  (`internal/indexer/languages_*.go`).

  Bundle size matters more than usual here because the built bundle is
  **committed to git and embedded in the signed binary** — it is a supply-chain
  artifact, not just a load-time cost. Prism measures smaller (~2KB core +
  0.3-0.5KB/language ≈ ~9KB for 12), but Prism v1 is in maintenance and v2 has
  been in alpha a long time; highlight.js's maintenance story is the stronger
  consideration. `starry-night` was rejected — 185KB **plus a WASM binary**
  before any grammars, which is a supply-chain conversation this phase does not
  need. Shiki was ruled out at roadmap time.
  — **Reversibility:** costly — the choice is baked into the committed
  `web/build/` output and its drift guard, so swapping it later means
  regenerating and re-committing the bundle, not just changing an import.

- **D-20:** **A truncated file says so plainly and offers the BRW-09 permalink as
  the route to the rest.** Render "showing first N of M lines" from the fields
  `SourceBlob` already carries (`total_lines`/`returned_lines`, whose doc comment
  prescribes exactly this string).

  **There is no pagination RPC and Phase 3 does not add one.** A ranged source
  fetch would be a second bounded-read path — the very thing SRV-05 warns
  against — and would partly undo RPC-05's deliberate caps. Two requirements
  specified separately turn out to complete each other: the local view is bounded
  by design, the remote view is the unbounded one. When the permalink is
  unavailable (no remote, non-GitHub forge, per D-08), the message states the
  truncation without an escape hatch.

- **D-21:** **BRW-05's picker lists all `total_candidates`, visually marking
  entries with `detail_gathered = false`.** Phase 1 built exactly this
  distinction: `uiMultiDefCap` bounds how many candidates get `calls`/`called_by`/
  `source` fetched, but `total_candidates` reports the **true** total, and
  `detail_gathered` tells a client which entries are complete "without inferring
  it from emptiness alone."

  Picking an ungathered candidate issues `GetNodeDetail` with its `file` (and
  `line`), returning it as a single-def with everything populated — and D-10's
  URL params address that candidate directly. **No auto-picking a "best"
  candidate:** BRW-05's text is that a bare name "offers a picker instead of
  guessing", and nothing on the wire ranks candidates, so "best" would be
  invented.

### Supply Chain

- **D-22:** **Phase 3 inherits Phase 2's vendored-component gap, records it, and
  constrains blast radius instead of closing it.**

  Phase 2 named this boundary deliberately: `shadcn-svelte add <component>`
  fetches from a registry over the network and writes `.svelte` **source files**
  into the repo, which are then committed. Those files never enter
  `pnpm-lock.yaml`, so **BLD-06's audit gate structurally cannot see them.**
  Phase 2 could record it safely because it shipped zero components
  (`web/src/lib/` today holds only `client.ts`, `gen/`, `utils.ts`, `assets/`).
  **Phase 3 is the first phase that makes the gap live.**

  Mitigation is scope discipline, not new machinery: add only the components the
  Browse view genuinely needs, and review the vendored source at add time like
  any other committed code — it *is* reviewable code in the repo, not an opaque
  dependency. Registry-version pinning with a source-match assertion (the drift
  discipline already applied to `web/build/` and the generated TypeScript) was
  considered and declined by Phase 2 as scope growth BLD-06's text does not ask
  for; that ruling stands. File a todo for it rather than growing this phase.

### Claude's Discretion

- Exact panel layout of the Browse view — how source, callers, callees and blast
  radius share the screen.
- BRW-07's copy affordance (button vs. icon vs. click-to-copy) and which of
  file path / symbol name / qualified name are copyable.
- How `Impact`/`Affected` depth is surfaced for blast radius, within D-10's
  `depth` param and D-12's pass-through validation.
- Precise `AbortController` wiring and whether a request-id guard backs it up.
- Whether the status banner from D-04 is dismissible.
- Exact `GetPermalink` field names and whether `availability` is an enum or a
  string, provided it is three-valued per D-07.
- Which shadcn-svelte components are added, subject to D-22's minimality.

### Folded Todos

Two todos were folded.

1. **Add golangci-lint with gofmt and idiomatic Go linters**
   (`.planning/todos/pending/2026-08-10-add-golangci-lint-with-gofmt-and-idiomatic-go-linters.md`,
   area `ci`, score 0.7).

   **Carried-forward history the planner must act on — this is the THIRD fold.**
   It was folded into Phase 1 (`01-CONTEXT.md` §Folded Todos, item 2) and **not
   implemented**; folded again into Phase 2 (`02-CONTEXT.md` §Folded Todos, item
   1) and **not implemented**. Verified again at this phase: no `.golangci.yml`
   or `.golangci.yaml` exists in the tree, and no pending todo carries a
   `resolves_phase` marker. It still has **no backing requirement** — flagged
   once here, and the maintainer reaffirmed the fold.

   Two prior folds produced nothing, so folding is demonstrably not what has been
   missing. This one must land as an **actual task with a verifiable outcome**
   (config file exists, task target exists, CI invokes it, and it fails on a
   planted violation per rule `84d1gfpywd`), not as a mention in a plan.
   Phase 3 does add Go code — the D-06 permalink derivation and the D-02
   regression test — so there is real surface for it to lint.

2. **Wire oracle `toolslist-repeat` response ordering flake**
   (`.planning/todos/pending/2026-08-07-wire-oracle-toolslist-repeat-response-ordering-flake.md`,
   area `mcp`, score 0.9).

   Phase 1 proved its root cause **separable** from FIX-01 (`pendingWriter.Write`
   decrements on every write at `internal/mcp/server.go:339`, while only client
   request lines increment at `:276`) and deliberately left it open on its own
   merits. Phase 3 touches neither `internal/mcp` nor the wire oracle, so this
   rides along as unrelated work — the planner should scope it as its own plan
   with no dependency on any Browse-view task, and it must not gate phase
   completion on the browse criteria.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase scope and requirements
- `.planning/ROADMAP.md` §"Phase 3: Browse, Inspect & Navigation" — goal, the 5
  success criteria, and the Notes clause that SRV-05 reuses rather than
  reimplements confinement, that BRW-06 registers only indexed languages, and
  that NAV-01's depth/limit shape is settled here for Phase 4.
- `.planning/REQUIREMENTS.md` lines 19, 38-46, 50-53 — SRV-05, BRW-01..BRW-09,
  NAV-01..NAV-04 requirement text. Line 123 records why server-persisted
  bookmarks are out of scope (NAV-01's URLs already are bookmarks).

### Prior-phase decisions this phase depends on
- `.planning/phases/01-engine-seam-wire-protocol-secure-transport/01-CONTEXT.md`
  — D-02a additive-only proto discipline; D-05/D-06 commit SHA as Meta field 8,
  UI/RPC only; D-08 bind address as an unwired field and the explicit **decline
  of a config file** (bears on D-08 here); D-11/D-12 the three truncation limits
  and `truncated` as a field never an error; D-14/D-15/D-16 the degrade contract.
- `.planning/phases/01-.../01-VERIFICATION.md` — records the live-verified RPC
  surface and the deferred UAT gap G-01-1 that Phase 2 closed.
- `.planning/phases/02-spa-toolchain-embedded-app-shell-js-supply-chain/02-CONTEXT.md`
  — D-08 Connect JSON encoding; D-09/D-10 ServeMux precedence and the rejected
  dotted-path heuristic (directly constrains D-10 here); D-11 `no-store` on
  `index.html`; D-17 Tailwind v4 + shadcn-svelte already installed; D-18 the four
  routes; D-19 the shell's real `GetStatus` call; and the **Known Limitation**
  section on vendored components that D-22 inherits.

### Wire contract
- `internal/uiproto/uiv1/ui.proto` — the whole service. Specifically:
  `GetStatusResponse` fields 6-9 (`stale`, `commit_sha`, `store_exists`,
  `indexing_in_progress`) for D-03/D-05; `SourceBlob` and its `reserved 50 to 59`
  band for D-20; `GetNodeDetailResponse`'s three-mode table for D-10/D-21;
  `NodeDefinition.detail_gathered` for D-21; `ExploreGroup.source` for D-14;
  `Location`'s deliberate lightness for D-09; `IndexingInProgress` for D-04.
- `internal/schema/graph.proto` §`message Edge` lines 70-95 — **the load-bearing
  citation for D-18**: the Pebble edge key omits line/col, collapsing multiple
  call sites into one edge.

### Server implementation touched or relied on
- `internal/uiserver/handlers.go` — `GetNodeDetail` (:728), `nodeDetailToProto`
  (:671), `singleDefSourceBlob` (:641) and its CR-02 rationale (D-03).
- `internal/uiserver/degrade.go` — `classifyDegrade` (:72), `errIndexingInProgress`
  (:105), `degradedStatus` (:136); the source of D-04's asymmetry.
- `internal/uiserver/truncate.go` — the line-then-byte cap and `countLines`, the
  single implementation both totals go through (D-20).
- `internal/query/node.go` lines 29-75 — `resolveSourcePath`, the confinement gate
  with WR-03 post-symlink re-verification (D-01, D-02).
- `internal/query/detail.go` :258 — `SourceFor`, documented as "a wrapper over the
  existing confinement gate, not a second read path" (D-01).
- `internal/query/errors_test.go` :45, :139 — the existing refusal assertions
  D-02's RPC-level test mirrors.
- `internal/query/validate.go` :51-56 — `defaultMaxFiles = 5` (D-14).
- `internal/query/explore.go` :110-144 — H21 adaptive budget, range [1,20] (D-14).
- `internal/gitmeta/worktree.go` :38,:64 and `internal/indexer/commit.go` :60-68 —
  the git-exec-with-graceful-degradation pattern D-06 extends.
- `internal/indexer/languages_*.go` — the 12-language indexed set for D-19.

### Client
- `web/src/lib/client.ts` — the existing Connect client factory and D-08's JSON
  encoding rationale; new shared modules (D-04, D-13) sit beside it.
- `web/src/routes/browse/+page.svelte` — the placeholder D-18 created for this
  phase to fill.
- `web/embed.go` — the `embed.FS` the committed bundle ships through; why D-19's
  bundle size is a supply-chain concern.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- **The entire RPC surface already exists.** All nine methods (`GetStatus`,
  `Search`, `Files`, `Callers`, `Callees`, `Impact`, `Affected`,
  `GetNodeDetail`, `Explore`) ship from Phase 1 and were live-verified against a
  real index. Phase 3 adds at most one method (D-06's `GetPermalink`, plus D-01's
  conditional source endpoint).
- **Verbatim source serving already works**, bounded, with the confinement gate
  wired in — `GetNodeDetailResponse.source` (field 9) covers both file mode and
  single-def mode; multi-def carries it per candidate.
- **The confinement gate is already shared** between MCP and uiserver via
  `internal/query`'s `readSourceFile`/`resolveSourcePath` — SRV-05's "reuse, do
  not reimplement" is already satisfied structurally.
- **The app shell, routes, design system and typed client all exist** from
  Phase 2: `/browse` is a real route, Tailwind v4 and shadcn-svelte are
  initialised, and `uiClient` is a working generated client proven end-to-end by
  D-19's live `GetStatus` call.

### Established Patterns
- **"No second copy of a rule."** Nearly every request message's doc comment
  states that bounds are enforced server-side once and passed through — this
  directly produced D-12, and argues against client-side clamping anywhere.
- **Degrade to "unknown", never to an error.** `commit_sha` empty means unknown;
  git absent means empty; `Explore` empty is a successful response. D-03, D-06
  and D-07 all follow this shape.
- **One concern per file with a doc comment naming its decision IDs** —
  `degrade.go`, `truncate.go`, `originguard.go`, `spa.go`. New server code
  (permalink derivation) should follow it.
- **Guards carry positive assertions** (rule `84d1gfpywd`) — D-02's regression
  test needs a passing control alongside the refusal case.
- **`reserved` bands sit far above organic field numbering** so pre-agreed future
  fields land at agreed numbers (`Node`, `Edge`, `Meta`, `SourceBlob`).

### Integration Points
- `internal/uiserver/` gains the permalink handler (and possibly D-01's source
  endpoint), beside the existing per-concern files.
- `internal/gitmeta/` gains remote-URL derivation, following `commit.go`'s
  git-may-be-absent contract.
- `internal/uiproto/uiv1/ui.proto` gains `GetPermalink` + its messages —
  **additive only**, and `task proto:drift` must stay green (note Phase 2's D-07
  trap: the drift floor is a hardcoded file count that must move to the *correct*
  number, never merely upward).
- `web/src/lib/` gains three shared modules: status gate, Connect error mapper
  (D-04), and URL parse/serialize (D-13).
- `web/src/routes/browse/+page.svelte` is filled in; `web/build/` is regenerated
  and re-committed, so the D-19 highlighter choice lands in the signed binary.

</code_context>

<specifics>
## Specific Ideas

- The maintainer explicitly redirected D-06 away from all three offered wire
  shapes toward a server-side RPC: *"why not have the client send the details to
  the server, so that it can return the right url?"* — which turned out to be a
  strict superset, and the only option able to express the unpushed-commit case.
- Both `/` and `Cmd+K` for search focus, so neither the GitHub-conditioned nor
  the command-palette-conditioned user has to learn the other's habit.
- "Showing first N of M lines" plus the GitHub permalink is the intended
  truncation experience — the bounded local view and the unbounded remote view
  are deliberately complementary.

</specifics>

<deferred>
## Deferred Ideas

- **Server-provided syntax-highlight spans / resolved reference ranges.**
  Anticipated by `SourceBlob`'s `reserved 50 to 59` band, but blocked by the
  Pebble edge key's omission of line/col (D-18). Revisit only if the storage key
  shape changes.
- **A ranged/paginated source fetch** so a truncated file can be read fully
  in-UI (D-20). Declined as a second bounded-read path.
- **`end_line` on `Location`**, which would make every permalink a range (D-09).
  Declined as the wrong trade for the shared five-RPC message.
- **Registry-version pinning with a committed-source match assertion for
  shadcn-svelte components** (D-22) — the drift discipline already applied to
  `web/build/` and the generated TS. Declined by Phase 2, re-declined here; file
  as a todo.
- **Additional forges for BRW-09** (GitLab, Bitbucket, Codeberg, Gitea) and the
  configurable host mapping self-hosted instances would need (D-08). Blocked on
  this project having any config mechanism at all.
- **A vulnerable-JS-fixture red proof for BLD-06**, mirroring `task vuln:selftest`
  — carried forward unresolved from Phase 2's D-15.

### Reviewed Todos (not folded)

Four of the six matched todos were reviewed and not folded. All matched on `area`
keywords (`ci`, `release`, `testing`, `docs`) rather than on scope — Phase 3
touches `web/` and `internal/uiserver`.

- **`post-release-verify.yml`'s event-aware conclusion guard has no test**
  (area `ci`, score 0.9) — release-workflow surface, untouched by this phase.
- **`release:dry-run-signed`'s additions-only diff guard passes vacuously when
  the awk anchor stops matching** (area `release`, score 0.6) — a genuine
  `84d1gfpywd` instance, but in the release path.
- **tap App secret-distinctness test is tautological** (area `testing`,
  score 0.6) — Homebrew tap surface.
- **brew trust instructions recommend the broader `--tap` grant** (area `docs`,
  score 0.2) — docs, unrelated.

</deferred>

---

*Phase: 3-Browse, Inspect & Navigation*
*Context gathered: 2026-08-28*
